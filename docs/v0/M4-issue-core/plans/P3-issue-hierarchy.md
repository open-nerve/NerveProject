# M4/P3 父子与关联 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 工作项的父子：`deleteIssue` 连带子树（工作项和它下面的每个工作项，不论多深、已归档的和删除者看不到的也在，同一时刻、同一个删除者，每张表一条语句）；创建、修改带 `parent_id`（同一个项目、调用者看得到、没归档、不成环，锁下的祖先链 `WITH RECURSIVE`，动态第 5 行按标识写原来和新的父工作项）；`sub_issues_count`（按调用者在那个项目的规则数：只看自己的访客只数他建的），回答的 `parent_id` 指名调用者看不到的工作项时为 `null`；`listSubIssues`、`addSubIssues`。关联：迁移 `00020`（`issue_relations`，一对工作项一行）；删除工作项、项目、工作区连带两个方向的关联行（按 `id` 先锁，约定五的例外）；`createIssueRelations`（约定七：另一端先不加锁地预检，看不到的不锁、不进闸门；涉及的项目一次按 `id` 升序进写闸门、一条语句锁住；正反两行动态、两边的最后修改者）、`listIssueRelations`（对方看得到、没删除、没归档、不在别的已归档的项目里）、`deleteIssueRelation`（两端任一个的判定，看不到两端时 404 `issue.relation_not_found`）。归档的交互（已归档的父工作项、关联的任一端 409 `issue.archived`）；交错 9（两次移动；加子工作项与移动）、10、11、13、23、26 和加关联与删除工作区的两种顺序；矩阵；I8、I9、I17 的接口版本，I15 加子工作项、关联和它们的 409（M4 设计 12 节 P3）。

**Architecture:** `platform/gate`：`Keyed.Enter(ctx, keys...)`（按给的次序一个一个进，一个 `max_wait` 的期限管全部，任一个失败时放回已进的）。`project`：`LockIssueProject` 回答项目的标识，`LockIssueProjects`（一条语句，`id` 升序 `FOR NO KEY UPDATE`）、`ArchivedIssueProjects`。`issue` 模块：`domain`（`hierarchy.go`、`relation.go`，动态第 5、20–23 行，`ErrRelationNotFound`）、`app`（`Locks.run` 的两种新形式：跨项目的 `lock_across.go`、两端任一个的 `lock_either.go`；`seen.go`：回答按调用者的规则数子工作项，只看自己的访客看不到的父工作项回答为 `null`；`parents.go`、`sub_issues.go`、`create_relations.go`、`list_relations.go`、`delete_relation.go`；`delete_issue.go` 的子树和关联行）、`adapter/postgres`（`parents.sql`、`sub_issues.sql`、`relations.sql`，删除的语句加子树和关联）、`adapter/http`（五个新操作）。迁移 `00020_issue_issue_relations.sql`。`bootstrap`：接线（`LockMany`、`Archived`、多个项目的写闸门）、矩阵的父子和关联的行、组合、交错。`e2e`：I8、I9、I17（新），I15、I1、I2 的补充。不加 Go 模块、npm 包；模块之间只经端口。

**Tech Stack:** Go 1.27.1、pgx v5.11.0、sqlc v1.31.1（`CGO_ENABLED=0`）、oapi-codegen v2.8.0、goose v3.28.0、River v0.47.0、golangci-lint 2.13.2、PostgreSQL 18.6（testcontainers）；Node 24、pnpm 11.10.0、Playwright 1.63.0、TypeScript 5.8.3、oxlint 1.51.0、oxfmt 0.35.0。

**Spec:** `docs/v0/M4-issue-core/specs/P3-issue-hierarchy.md`（上级：`docs/v0/M4-issue-core/M4-design.md`）

## Global Constraints

- **基点**：本 plan 和 spec 在 `f8696490`（P2 的合并提交）之上的两个提交里：初稿（M4 设计第 15 节 P3 一行在其中改为"进行中"，12 节 P3 的标题和 15 节的任务数随重新切分改为 16），和照控制者的裁定（spec 第 3 节、预检）修订的一个；Task 1 从修订的提交开始。两个提交都只改文档，代码的基点是 `f8696490` 的树。
- **Go 版本和依赖**：本 plan 不执行 `go get`，不加任何 Go 模块或 npm 包。`server/go.mod` 和 `server/tools/go.mod` 保持 `go 1.27` 和 `toolchain go1.27.1`；`git diff --stat f86964900290c269f140af1dc241456c4f7866bb -- server/go.mod server/go.sum server/tools/go.mod server/tools/go.sum pnpm-lock.yaml` 在本 plan 的任何时刻都没有输出。
- **每个改了 `server/` 的 Task（Task 1–14）提交前**：`make lint-go` 两段都输出 `0 issues.`；`make test` 全部通过（没有 `FAIL`）。Task 15 只改 `e2e/`，跑 Web 的三个门禁和 `make e2e`；Task 16 只改文档，跑 `make lint-web`、`doclinks.py` 和本 Phase 最后一次全量的 `make test`。
- **生成的文件不手写、不从本 plan 复制**：有生成物的 Task 执行 Task 中的生成命令（改了契约的 Task 1、2、3、4、6、8、11、12、14 执行 `make gen`，只动了 sqlc 的 Task 5、7、9、13 执行 `make gen-go`；Task 10 没有生成物），用 `shasum -a 256` 和 `wc -l` 核对生成命令下面的表中的 SHA-256 和行数，对不上时停下来：说明某个输入与本 plan 不一致。这些 Task **提交之后**执行 `make gen-check`（它用 `git status` 判断生成物是否已提交，提交之前必然报差异）。
- **前端检查和新的错误码**（M4 设计 12 节约束 4）：本 Phase 只有一个新码 `issue.relation_not_found`，Task 14 把它的文案加进 `web/apps/web/core/lib/error-messages.ts`（`PROBLEM_MESSAGES`）和两份 `errors.json`（`web/packages/i18n/src/locales/en/errors.json`、`web/packages/i18n/src/locales/zh-CN/errors.json`）。改了契约、前端或 `e2e/` 的 Task 1（关键词守卫的 `phase`）、2、3、4、6、8、11、12、14、15 另执行 `make lint-web`、`make knip`、`make test-web`；改了故事经过的读写或断言的 Task 1（删除连带子树，I15 删除工作项）、Task 4（每个工作项的回答多了 `sub_issues_count`）、Task 8（迁移 `00020`；删除项目、工作区的断言暂时不要关联表的行）、Task 11（`furnishIssueRows` 加关联，暂时的豁免去掉）和 Task 15 执行 `make e2e`；Task 16 只改文档，执行 `make lint-web` 和 `doclinks.py`（下面"一次性的脚本"）。
- **e2e 的数目**：在 worktree 中 `make e2e` 必须全部通过：开始时 123 个，Task 15 起 126 个（I8、I9、I17）。只在不是 git 仓库的副本里，S3 因为构建读不到提交而失败（M3/P4b spec F4）；没有装 Firefox 的机器上 A4 的两个 firefox 测试因为浏览器不在而失败；两者都与本 plan 无关。
- **容器**：`make test` 和 `make e2e` 用自己的 testcontainers；机器忙时偶尔起不来，等 Docker 空闲之后重跑一次再当作失败（容器起不来不算任何检查的失败）。容器测试一次只跑一套。开发库 `nerve-dev-db-1` 不要启动、停止或重建，不要执行 `make dev-db-down`、`make dev-db-reset`。不要碰其他项目的容器（`agentforge-*`、`plane-app-*`、`opennerve-*`、`nervewiki-*`）和留下的 testcontainers。
- **git**：每次 Bash 调用只执行一个 git 命令，不用 `;`、`&&`、`|` 串联 git；不用 `git -C`、`stash`、`clean`、`reset --hard`。不用 `cd`。不碰 `plane/`、`refer/`。
- **安装**：除了 Docker、Go、Node 不做任何全局安装；不执行 `corepack enable`（pnpm 已在 PATH 上）。不把副本的 `node_modules` 链接到 worktree 的。
- **规则**：不写 `init()`，不用全局可变状态，构造函数显式传入依赖；不建 `utils`、`common`、`helpers` 包；一个文件只做一件事，不超过约 400 行；依赖只能向内；模块之间不互相导入，不跨模块的表 JOIN；模块的 SQL 只经 sqlc；端口由使用方声明；角色只按集合判断，不按大小比较；不留没有使用者的代码。工作项一侧的每个写经 `issue/app/lock.go` 的 `Locks.run`（一条共用的取锁路径：跨项目的写和两端任一个的写是它的两种形式，`lock_across.go`、`lock_either.go`，不另写第二条），每个指名工作项的判定经 `Locks.decide`（访客规则在那里），不写近似的副本；测试的帮手共用、扩展，不复制、不重写（`issue_world_test.go` 的 `issueWorld`、`newWorkItem`、`newWorkItemIn`、`workItemStamps`、`activityBy`，`issue_interleaving_test.go` 的 `itemWrite`、`wroteNothing`、`serialize`、`inBothOrders`，`interleaving_test.go` 的 `serializeOn`、`heldTx`、`holding`，矩阵的帮手和 `matrixRow.prepare`，`deleteOver`、`archiveOver`，`e2e/fixtures/assert/issue.ts` 的 `byId`、`changed`、`assigned`、`createdFor`、`expectIssueArchival`，`assert/catalog.ts` 的 `tablesUnder`、`keysBelow`；一种动态的行一个构造）。**例外**：契约按模块一个文件（M0-P3 交接 5），`api/modules/issue.yaml`（854 行）、`api/modules/project.yaml`（1363 行）和 `api/modules/workspace.yaml`（872 行）不受约 400 行的限制；关键词守卫的规则表 `tools/keywords.json`（2670 行，数据，本 plan 只改它的 `phase`）同样；三个基点已在约 400 行的测试文件，本 plan 各加几行（spec 第 3 节第 23 条）：`bootstrap/permission_matrix_test.go`（410 行，基点 406 行：矩阵的种子带上关联）、`bootstrap/issue_interleaving_test.go`（407 行，基点 393 行：`serializeAt`、`inBothOrdersAt` 让交错等在关联表的行上，与原来的两个帮手共用一份）、`server/migrations/schema_test.go`（401 行，基点 400 行：迁移的个数）。本 plan 改的其余代码文件都在约 400 行以内（最终原型上量的）：最长的是 `issue/app/fakes_test.go`（398 行）、`bootstrap/ports_test.go`（391 行）；新建的文件中最长的是 `issue/adapter/postgres/relations_test.go`（327 行）、`issue/app/sub_issues_test.go`（322 行）、`bootstrap/issue_interleaving_relations_test.go`（306 行）。
- **关键词守卫**：72 条规则、23 个例外，没有命中。Task 1 把 `tools/keywords.json` 的 `phase` 改为 `M4/P3`，因此也执行 `make lint-web`。
- **注释**：Go、TS 代码、SQL 查询和接口描述用英文；迁移和配置文件的注释、中文文档照本 plan 原样。每条注释和说明在它所在的 Task 结束时都与代码相符（spec 附录 A 的清扫 59）。
- **代码块**：每个改动都写成四个反引号围起来的块，块的第一行写明种类和路径，照原样使用（原型中逐字节运行过）：
  - ````` ````file <路径> ````` 新文件，块的内容加一个结尾换行就是整个文件；
  - ````` ````whole <路径> ````` 已有文件的完整新内容（同样加结尾换行）；
  - ````` ````old <路径> ````` 与紧跟着的 ````` ````new <路径> `````：`old` 的文字在文件当前版本中恰好出现一次，把它换成 `new` 的文字；
  - ````` ````delete <路径> ````` 删除这个文件（块是空的）。

  一个文件的几个块按出现的顺序依次应用。拼 plan 的脚本已从基点起按顺序核对过全部块：每个 `old` 恰好出现一次（在它之前的块应用之后的文件中），每个新文件原来不存在，逐 Task 应用之后的文件与原型逐字节相同；之后在另一个副本上从基点逐 Task 复现，每个 Task 之后跑它的门禁，最后的树与原型逐个文件相同（spec 附录 A）。块写入之后不必再格式化（原型中每个 Task 都经过 `gofmt`、`oxfmt`）。
- **plan 是"照执行的"**（P1b 的裁定）：执行中的每个修复轮在提交它的同时改本 plan 里它的 Task 的块，plan 始终从基点复现到 HEAD。一个 Task 的块由**一条命令**从两个固定的树重新生成：`python3 $M4TMP/p3tools/taskblocks.py <n> <之前的树> <之后的树>`（树是目录，或 `git archive -o <文件>.tar <提交>` 得到的 tar，每个 git 命令单独执行；它把两个树放进 `$M4TMP/p3exec/T<n-1>`、`T<n>`，写出 `plan-blocks/T<n>.md`、`T<n>.files` 和生成物的表 `gen-T<n>.txt`，并从基点核对块，`problems: 0`）；然后 `python3 $M4TMP/p3tools/compose_plan.py <plan>` 重新拼出本 plan（变异表照结果文件写出，执行中加的变异行在修复轮里加进 `mutants_p3*.py` 和结果文件）。
- **过渡版本**：一些文件先在较早的 Task 写成过渡版本，较晚的 Task 再修改（文件结构一节的"Task"一列列出每个文件的全部 Task）：`api/modules/issue.yaml`（Task 1、2、3、4、6、8、11、12、14）、`issue/app/lock.go`（Task 2、4、10、13）、`issue/app/lock_across.go`（Task 10、12、13）、`issue/app/ports.go`（Task 3、4、5、7、10、12、13）、`issue/app/rows.go`（Task 2、3、5、9、13）、`issue/app/fakes_test.go`（Task 1、2、3、4、5、7）、`issue/app/parents.go`（Task 2、3、5、10）、`issue/module.go` 和 `issue/adapter/http/handler.go`（Task 6、11、12、14）、`issue/adapter/postgres/queries/relations.sql`、`relations.go` 和它的测试（Task 9、12、13）、`bootstrap/gate_test.go`（Task 6、7、11、14）、`bootstrap/issue_write_locks_test.go`（Task 6、8、11、14）、`bootstrap/issue_writes_test.go`（Task 2、3、6、11、14）、`bootstrap/permission_matrix_test.go`（Task 6、11、12、14）、`bootstrap/permission_matrix_relations_test.go`（Task 11、12、14）。两个暂时的写法：Task 8 加了关联表，`e2e/fixtures/assert/project.ts`、`workspace.ts` 的删除断言暂时不要它的行（那时还没有经接口建关联的办法），Task 11 的 `furnishIssueRows` 建了关联之后去掉；`expectIssueDeleted` 在 Task 8 暂时不要求 `parent_id` 和关联的两个外键下有行，Task 15 的故事建了它们之后去掉（spec 第 3 节第 18 条）。每个过渡版本都在逐 Task 复现中运行过。
- **变异**：每个 Task 末尾的"变异"表列出：把代码改坏的方式、必须因此失败的检查和它所在的层（单元：假实现，或没有替身的平台、领域的测试；存储：真实数据库；组合：`bootstrap` 组合出的 app；端到端：单独运行的故事；守卫：生成器、文案的键的核对）。它们在各自 Task 的树上跑过，标"（Task n 起）"的检查在 Task n 的树上跑过，Go 的全部在最终的原型上再跑过，每个变异在一份新的副本里（`$M4TMP/p3tools/mutants_p3.py`，`mut.py` 在它写的每个检查上各跑一次；端到端的"发现"必须是故事的断言失败，`e2ekills.py` 核对，不算构建失败或超时；容器起不来不算；spec 附录 A）。"（Task n 起）"的检查从那个 Task 起才发现这个变异：Task n 的实现者在 Task n 的树上重跑它（那个 Task 的表下面列出）。**安全或加锁的性质只由单元一层发现的，算缺口**：表中每一条这类性质都另有存储、组合或端到端一层的检查，例外（"按性质只在单元一层"）写在 spec 附录 A.3。
- **评审敏感**（M4 设计 12 节约束 3：加锁约定和连带的这一段，P3 延伸约定七和关联行的例外）：多个项目的取锁和写闸门（spec 第 3 节第 2 条）和死锁的重推（第 3 条）；关联行按 `id` 先锁（第 4 条）；子树在同一时刻删除（第 5 条）；并发的移动不成环（第 6 条）；加关联的另一端的预检（第 7 条）；两端任一个的判定（第 8 条）；访客规则在列表、计数和回答里，回答指名的父工作项也在内（第 9 条）；归档的拒绝次序（第 11 条）。改动这些代码或测试之前，先照"变异"表确认它在所说的性质去掉之后失败。
- **提交**：提交信息用英文，末尾加一行：`Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`
- 所有命令在仓库根目录下执行，除非步骤中另有说明。

## 一次性的脚本

`$M4TMP/p3tools/doclinks.py`（Task 16 的文档核对：给出的 markdown 文件中每个相对链接都指向仓库中存在的文件或目录；代码块和行内代码里的不算链接）：

```python
# M4/P3 plan, Task 16 (from M4/P2's, from M4/P1b's, from M4/P1a's, from M3's closeout plan): every relative link of the markdown files
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
| `tools/keywords.json`（修改） | `phase` 改为 `M4/P3` | 1 |
| `api/modules/issue.yaml`（修改） | `deleteIssue` 连带子树（Task 1）；`IssueCreate.parent_id`（Task 2）；`IssueUpdate.parent_id`（Task 3）；`IssueDetail.sub_issues_count`（Task 4）；`listSubIssues`、`addSubIssues`，`Issue`、`IssueCollection`、`SubIssuesAdd`（Task 6）；删除的描述写上关联（Task 8）；`createIssueRelations` 和关联的模式（Task 11）；`listIssueRelations`（Task 12）；`deleteIssueRelation`、`RelationID`、`issue.relation_not_found`（Task 14） | 1、2、3、4、6、8、11、12、14 |
| `api/openapi.yaml`（修改） | 新的路径 | 6、11、14 |
| `server/internal/modules/issue/adapter/postgres/queries/delete_issue.sql`、`server/internal/modules/issue/adapter/postgres/delete_issue.go`、`server/internal/modules/issue/adapter/postgres/delete_issue_test.go`（修改） | `IssueSubtree`、`Subtree`，五条语句按工作项的集合写（`issue_ids`）（Task 1）；`DeleteIssueRelationRows`：子树的关联行，两个方向，按 `id` 先锁（Task 8） | 1、8 |
| `server/internal/modules/issue/app/delete_issue.go`、`server/internal/modules/issue/app/delete_issue_test.go`（修改）、`server/internal/modules/issue/app/fakes_deletion_test.go`（新） | `IssueDeletion.IssueIDs`、`IssueRowsDeleter.Subtree`：删除连带子树；删除的假实现从 `fakes_test.go` 移出（Task 1）；关联行的一步在标签之后（Task 8） | 1、8 |
| `server/internal/modules/issue/app/fakes_test.go`（修改） | 删除的假实现移出（Task 1）；`callLog.fail` 按整行、创建带父工作项（Task 2）；锁下的项目带标识、修改的父工作项（Task 3）；`SubIssueCounts`、`GetIssues` 的读（`read`）（Task 4）；`SubIssues`、`SetParents` 的读写记录（Task 5）；`LockMany`、写闸门取多个项目（Task 7） | 1、2、3、4、5、7 |
| `server/internal/bootstrap/issue_deletion_test.go`（修改） | 子树在同一时刻删除，管理员·访客的删除连带他看不到的子工作项（`deletesAtOneMoment`）；按目录的删除去掉 `subItems` 的豁免，`deletedWorkItem` 的工作项有子工作项（Task 1）；关联的两个外键下各有一行（Task 8） | 1、8 |
| `server/internal/bootstrap/project_deletion_test.go`（修改） | 按目录读 `keysTo(issues)`（Task 1）；删除项目连带两个方向的关联行（Task 8） | 1、8 |
| `server/internal/modules/issue/domain/hierarchy.go`（新） | `ParentNotAllowed`（Task 2）；`ParentChange`、`Ref`、`ParentActivity`（Task 3）；`Loops`、`MaxSubIssues`、`CheckSubIssueIDs`、`SubIssueNotAllowed`（Task 5） | 2、3、5 |
| `server/internal/modules/issue/domain/issue.go`、`server/internal/modules/issue/domain/issue_test.go`（修改） | `NewIssue.ParentID`、`Issue.ParentID` 进创建（Task 2）；`Issue.SubIssuesCount`（Task 4） | 2、4 |
| `server/internal/modules/issue/app/parents.go`、`server/internal/modules/issue/app/parents_test.go`（新） | `parentOf`：创建的父工作项在锁下读、判定（Task 2）；`parentChange`、`refOf`：修改的父工作项，不成环（Task 3）；回答带计数（Task 4）；`inReach`（Task 5）；`refOf` 带锁住的项目（Task 10） | 2、3、4、5、10 |
| `server/internal/modules/issue/app/lock.go`（修改） | `decide` 交回 `Grant`，`held.grant`（Task 2）；`see` 交回 `Grant`（Task 4）；跨项目的形式：`write.reach`、`held.reached`、`scope`（Task 10）；两端任一个的形式：`write.instead`、`held.instead`（Task 13） | 2、4、10、13 |
| `server/internal/modules/issue/app/create_issue.go`（修改） | 创建带父工作项（`check`、`parentOf`）（Task 2）；回答的计数是 0（Task 4） | 2、4 |
| `server/internal/modules/issue/app/rows.go`（修改） | `IssueRow.ParentID`（Task 2）；`IssueFields.ParentID`（Task 3）；`ParentRows`（Task 5）；`ChangedRows`（Task 9）；`RelationDeletion`（Task 13） | 2、3、5、9、13 |
| `server/internal/modules/issue/adapter/postgres/queries/issues.sql`、`server/internal/modules/issue/adapter/postgres/issues.go`（修改） | `InsertIssue` 写 `parent_id`（Task 2）；`GetIssues`、`issueOf`（Task 4） | 2、4 |
| `server/internal/modules/issue/adapter/postgres/issues_test.go`（修改） | 创建写下父工作项 | 2 |
| `server/internal/modules/issue/adapter/postgres/parents_test.go`（新） | `TestAWorkItemsParentIsWritten`（Task 2）；`TestAWorkItemsAncestry`、修改写父工作项（Task 3） | 2、3 |
| `server/internal/modules/issue/adapter/http/issues.go`（修改） | 创建的 `parent_id`（Task 2）；修改的 `parent_id`（Task 3）；回答的 `sub_issues_count`（Task 4） | 2、3、4 |
| `server/internal/modules/issue/adapter/http/issues_test.go`（修改） | 创建带父工作项（Task 2）；回答的计数（Task 4） | 2、4 |
| `server/internal/bootstrap/issue_parents_test.go`（新） | 创建在父工作项之下、访客规则（Task 2）；修改移动工作项、不成环（Task 3）；回答不指名只看自己的访客看不到的父工作项（Task 4） | 2、3、4 |
| `server/internal/bootstrap/issue_writes_test.go`（修改） | 一个连接的池上的写：在父工作项下建（Task 2）、移动（Task 3）、"管理员·访客"改他在别人的工作项下的工作项，父工作项在事务里读（Task 4）、加子工作项（Task 6）、跨项目的关联（Task 11）、删除关联（Task 14） | 2、3、4、6、11、14 |
| `server/internal/modules/issue/domain/patch.go`、`server/internal/modules/issue/domain/patch_test.go`、`server/internal/modules/issue/domain/dates.go`、`server/internal/modules/issue/domain/activity.go`、`server/internal/modules/issue/domain/activity_test.go`、`server/internal/modules/issue/domain/identifier.go`、`server/internal/modules/issue/domain/identifier_test.go`（修改） | `IssuePatch.SetParentID`、`Changes.Parent`、`same`（`dates.go` 的 `sameDay` 并入它）、动态第 5 行、`Identifier`（Task 3）；`VerbDeleted`（Task 13，`activity.go`） | 3、13 |
| `server/internal/modules/issue/adapter/postgres/queries/parents.sql`、`server/internal/modules/issue/adapter/postgres/parents.go`（新） | `Ancestry`（`WITH RECURSIVE`，`UNION`） | 3 |
| `server/internal/modules/issue/adapter/postgres/queries/update.sql`、`server/internal/modules/issue/adapter/postgres/update.go`（修改） | 修改写 `parent_id` | 3 |
| `server/internal/modules/issue/app/update_issue.go`（修改） | 修改的父工作项（`check`、`parentChange`）（Task 3）；回答带计数（Task 4） | 3、4 |
| `server/internal/modules/issue/app/create_issue_test.go`（修改） | 端口失败的核对 `wantStopped`，`wantStoppedAt`、`wantStoppedOn` 都经它（Task 2）；世界的项目的标识 `WEB`（Task 3）；动态的核对 `wantActivityRows`，`wantActivity` 经它（Task 5） | 2、3、5 |
| `server/internal/modules/issue/app/fakes_parents_test.go`（新） | `Ancestry`（Task 3）；`SubIssueCounts`、`GetIssues`（Task 4）；`SubIssues`、`SetParents`（Task 5） | 3、4、5 |
| `server/internal/modules/issue/app/ports.go`（修改） | `Ancestry`（Task 3）；`SubIssueCounts`、`SubIssueCount`、`IssueReader.GetIssues`（Task 4）；`SubIssueLister`、`SubIssueAdder`（Task 5）；`LockMany`、`WriteGate.Enter` 取多个项目（Task 7）；`RelationCreator`（Task 10）；`RelationLister`、`ArchivedProjects`（Task 12）；`RelationDeleter`（Task 13） | 3、4、5、7、10、12、13 |
| `server/internal/modules/project/adapter/postgres/queries/issue_ports.sql`、`server/internal/modules/project/adapter/postgres/issue_ports.go`（修改） | `LockIssueProject` 回答标识（Task 3）；`LockIssueProjects`（Task 7）；`ArchivedIssueProjects`（Task 12） | 3、7、12 |
| `server/internal/modules/project/adapter/postgres/issue_ports_test.go`（修改） | 标识（Task 3）；`pgtest.RowWaits`（Task 8）；`TestArchivedIssueProjects`（Task 12） | 3、8、12 |
| `server/internal/modules/project/app/issue_ports.go`（修改） | `IssueProject.Identifier` | 3 |
| `server/internal/bootstrap/ports_test.go`（修改） | 标识的转换（Task 3）；`LockMany` 的转换（Task 7） | 3、7 |
| `server/internal/modules/issue/adapter/http/update_test.go`（修改） | 修改的 `parent_id`、体的形状 | 3 |
| `server/internal/bootstrap/issue_interleaving_hierarchy_test.go`（新） | 交错 9 的两次移动（Task 3）；交错 13、交错 9 的加子工作项与移动（Task 6） | 3、6 |
| `server/internal/modules/issue/app/seen.go`、`server/internal/modules/issue/app/seen_test.go`（新） | `grants`、`seen`、`seenOne`、`hideParents`：回答按调用者的规则数子工作项，只看自己的访客看不到的父工作项回答为 `null`（Task 4）；`seeEach`（Task 12，`seen.go`） | 4、12 |
| `server/internal/modules/issue/adapter/postgres/queries/sub_issues.sql`、`server/internal/modules/issue/adapter/postgres/sub_issues.go`、`server/internal/modules/issue/adapter/postgres/sub_issues_test.go`（新） | `SubIssueCounts`、`GetIssues` 的存储测试（Task 4）；`SubIssues`、`SetParents`（Task 5） | 4、5 |
| `server/internal/modules/issue/app/get_issue.go`、`server/internal/modules/issue/app/get_issue_by_identifier.go`、`server/internal/modules/issue/app/archive_issue.go`（修改） | 回答按调用者的规则数子工作项 | 4 |
| `server/internal/modules/issue/app/get_issue_test.go`、`server/internal/modules/issue/app/get_issue_by_identifier_test.go`、`server/internal/modules/issue/app/archive_issue_test.go`、`server/internal/modules/issue/app/update_issue_test.go`、`server/internal/modules/issue/app/dates_test.go`、`server/internal/modules/issue/app/guest_rule_test.go`、`server/internal/modules/issue/app/lock_test.go`（修改） | 回答的计数、读计数失败时的回答（Task 4）；`TestAWriteAcrossProjectsTakesOnlyTheProjectsItReaches`（Task 10，`lock_test.go`）；`TestAWriteOnEitherWorkItemTakesBothProjects`（Task 13，`lock_test.go`） | 4、10、13 |
| `server/internal/bootstrap/issue_sub_issue_count_test.go`（新） | 每个回答按调用者的规则数子工作项 | 4 |
| `server/internal/modules/issue/app/sub_issues.go`、`server/internal/modules/issue/app/sub_issues_test.go`（新） | `ListSubIssues`、`AddSubIssues`（Task 5）；`refOf` 带锁住的项目（Task 10，`sub_issues.go`） | 5、10 |
| `server/internal/modules/issue/domain/hierarchy_test.go`（新） | `TestLoops`、`TestCheckSubIssueIDs` | 5 |
| `server/internal/modules/issue/adapter/http/handler.go`、`server/internal/modules/issue/module.go`（修改） | 子工作项的两个用例（Task 6）；`CreateIssueRelations`（Task 11）；`ListIssueRelations`（Task 12）；`DeleteIssueRelation`（Task 14） | 6、11、12、14 |
| `server/internal/modules/issue/adapter/http/sub_issues.go`、`server/internal/modules/issue/adapter/http/sub_issues_test.go`（新） | `ListSubIssues`、`AddSubIssues`、`issueCollection` 和它们的 HTTP 测试 | 6 |
| `server/internal/bootstrap/issue_sub_issues_test.go`（新） | 加子工作项、列子工作项的组合测试；访客规则 | 6 |
| `server/internal/bootstrap/permission_matrix_sub_issues_test.go`（新） | 子工作项的矩阵（Task 6）；`otherItemOf`、已归档的项目里已归档的工作项一行（Task 11） | 6、11 |
| `server/internal/bootstrap/permission_matrix_test.go`（修改） | 行的登记（Task 6、11）；种子的关联（Task 12）；关联的删除的行（Task 14） | 6、11、12、14 |
| `server/internal/bootstrap/gate_test.go`（修改） | 加子工作项进项目的写闸门，形状不对的先答 422（Task 6）；`across`：取多个项目的写闸门（Task 7）；加关联进它涉及的每个项目的写闸门，形状不对的先答 422（Task 11）；删除关联（Task 14） | 6、7、11、14 |
| `server/internal/bootstrap/issue_write_locks_test.go`（修改） | `body` 带上 `t`，`addSubIssues` 一行（Task 6）；表的列表加关联（Task 8）；`createIssueRelations` 一行（Task 11）；`deleteIssueRelation` 一行（Task 14） | 6、8、11、14 |
| `server/internal/platform/gate/gate.go`、`server/internal/platform/gate/gate_test.go`（修改）、`server/internal/platform/gate/keys_test.go`（新） | `Keyed.Enter(ctx, keys...)`：一个期限管全部、按给的次序进、失败时放回 | 7 |
| `server/internal/platform/config/config.go`（修改） | `max_wait` 的说明写上跨项目的写 | 7 |
| `server/internal/bootstrap/gate.go`、`server/internal/bootstrap/ports.go`（修改） | 写闸门取多个项目、`LockMany` 的接线（Task 7）；`Archived` 的接线（Task 12，`ports.go`） | 7、12 |
| `server/internal/modules/issue/app/fakes_across_test.go`（新） | `LockMany`（Task 7）；`Archived`（Task 12） | 7、12 |
| `server/internal/modules/project/adapter/postgres/lock_issue_projects_test.go`（新） | `LockIssueProjects` 的存储测试（Task 7）；`pgtest.RowWaits`（Task 8） | 7、8 |
| `server/internal/modules/project/adapter/postgres/failures_test.go`（修改） | `LockIssueProjects`（Task 7）、`ArchivedIssueProjects`（Task 12）失败的读 | 7、12 |
| `server/internal/modules/project/module.go`（修改） | `IssueProjects.LockIssueProjects`（Task 7）；`ArchivedIssueProjects`（Task 12） | 7、12 |
| `server/migrations/sql/00020_issue_issue_relations.sql`（新）、`server/sqlc.yaml`、`deploy/runtime-grants.sql`（修改） | `issue_relations`、它的检查和索引；sqlc 的迁移列表；运行时的权限 | 8 |
| `server/migrations/schema_test.go`、`server/migrations/issue_schema_test.go`（修改） | 迁移的个数；表、列、检查、一对工作项一行的键 | 8 |
| `server/internal/modules/issue/adapter/postgres/queries/delete_rows.sql`、`server/internal/modules/issue/adapter/postgres/delete_rows.go`、`server/internal/modules/issue/adapter/postgres/delete_rows_test.go`（修改） | `DeleteProjectRelationRows`：删除项目的关联行，两个方向，按 `id` 先锁 | 8 |
| `server/internal/modules/issue/adapter/postgres/delete_relations_test.go`（新）、`server/internal/modules/issue/adapter/postgres/store_test.go`（修改） | 关联行的删除语句的存储测试；`newRelation` | 8 |
| `server/internal/modules/issue/app/delete_project_rows.go`、`server/internal/modules/issue/app/cascade_test.go`（修改） | `rowSteps` 的关联一步，在标签之后 | 8 |
| `server/internal/platform/postgres/pgtest/rowwaits.go`、`server/internal/platform/postgres/pgtest/rowwaits_test.go`（新） | `RowWaits`：一行是否被冲突的锁挡住（`NOWAIT`），不手写 `55P03` | 8 |
| `server/internal/modules/project/adapter/postgres/demote_test.go`（修改） | 用 `pgtest.RowWaits` | 8 |
| `server/internal/bootstrap/issue_world_test.go`、`server/internal/bootstrap/issue_interleaving_deletion_test.go`（修改） | `adminGuest`（Task 1，`issue_world_test.go`）；`workItemStamps`、删除的交错数关联行（Task 8） | 1、8 |
| `server/internal/bootstrap/issue_interleaving_test.go`（修改） | `wroteNothing` 数关联行（Task 8）；`serializeAt`、`inBothOrdersAt`（Task 14） | 8、14 |
| `api/modules/project.yaml`、`api/modules/workspace.yaml`（修改） | 删除项目、工作区的描述写上关联 | 8 |
| `e2e/fixtures/assert/issue.ts`（修改） | `expectIssueDeleted` 暂时不要 `parent_id` 和关联的两个外键下的行（Task 8）；`parent`、`ItemRef`、`parentRow`、`relatedRow`、`unrelatedRow`，`activityOf` 读出指名的工作项，暂时的豁免去掉（Task 15） | 8、15 |
| `e2e/fixtures/assert/project.ts`、`e2e/fixtures/assert/workspace.ts`（修改） | 删除的断言暂时不要关联表的行（Task 8）；去掉（Task 11） | 8、11 |
| `server/internal/modules/issue/adapter/postgres/queries/relations.sql`、`server/internal/modules/issue/adapter/postgres/relations.go`、`server/internal/modules/issue/adapter/postgres/relations_test.go`（新） | `RelatedIssues`、`InsertRelations`、`MarkChanged`（Task 9）；`IssueRelations`、`relationOf`（Task 12）；`GetRelation`、`DeleteRelation`（Task 13） | 9、12、13 |
| `server/internal/modules/issue/domain/relation.go`、`server/internal/modules/issue/domain/relation_test.go`（新） | 关联的类型、`CheckNewRelations`、`Relation`、`NewRelation`、动态第 20、21 行（Task 9）；第 22、23 行（Task 13） | 9、13 |
| `server/internal/modules/issue/app/lock_across.go`（新） | `scope`、`end`、`lockProjects`、`reachAll`、`reachUnder`、`reach`（Task 10）；`ascending`（Task 12）；两端任一个的锁（Task 13） | 10、12、13 |
| `server/internal/modules/issue/app/create_relations.go`、`server/internal/modules/issue/app/create_relations_test.go`（新） | `CreateIssueRelations` 和单元测试 | 10 |
| `server/internal/modules/issue/app/fakes_relations_test.go`（新） | 关联的假存储（Task 10）；`Relations`（Task 12）；`GetRelation`、`DeleteRelation`（Task 13） | 10、12、13 |
| `server/internal/modules/issue/adapter/http/relations.go`、`server/internal/modules/issue/adapter/http/relations_test.go`（新） | `CreateIssueRelations`（Task 11）；`ListIssueRelations`（Task 12）；`DeleteIssueRelation`（Task 14） | 11、12、14 |
| `server/internal/bootstrap/issue_relations_test.go`（新） | 加关联的组合测试、访客规则（Task 11）；访客规则下列出关联、数子工作项（Task 12） | 11、12 |
| `server/internal/bootstrap/issue_relation_locks_test.go`（新） | 加关联不锁它到不了的项目 | 11 |
| `server/internal/bootstrap/gate_across_test.go`（新） | 跨项目的写在它的几个闸门上一共等 `max_wait`，通过或被拒绝都放回它占的每个位置 | 11 |
| `server/internal/bootstrap/permission_matrix_relations_test.go`（新） | 关联的矩阵：加（Task 11）、列（Task 12）、删（Task 14） | 11、12、14 |
| `e2e/fixtures/furnish.ts`（修改） | `furnishIssueRows` 加第二个工作项和一条关联 | 11 |
| `e2e/fixtures/issues.ts`（修改） | `createIssueRelations`（Task 11）；`issueRead`、`subIssuesAdded`、`issueRelationsCreated`、`issueRelationsListed`、`issueRelationDeleted`（Task 15） | 11、15 |
| `server/internal/modules/issue/app/list_relations.go`、`server/internal/modules/issue/app/list_relations_test.go`（新） | `ListIssueRelations` 和单元测试 | 12 |
| `server/internal/bootstrap/issue_relation_lists_test.go`、`server/internal/bootstrap/issue_archived_projects_test.go`（新） | 关联的列表的组合测试；`Archived` 的转换 | 12 |
| `server/internal/bootstrap/permission_matrix_seed_relations_test.go`（新） | 矩阵的种子的关联（Task 12）；删除的关联、"别人的"（Task 14） | 12、14 |
| `server/internal/bootstrap/permission_matrix_seeded_test.go`（修改） | `seeded.relations`、`relation` | 12 |
| `server/internal/modules/issue/app/delete_relation.go`、`server/internal/modules/issue/app/delete_relation_test.go`、`server/internal/modules/issue/app/lock_either.go`（新） | `DeleteIssueRelation` 和单元测试；`both`、`eitherOf`、`eitherUnder`、`decideEither` | 13 |
| `server/internal/modules/issue/domain/errors.go`（修改） | `ErrRelationNotFound` | 13 |
| `server/internal/bootstrap/issue_relation_deletes_test.go`（新） | 删除关联的组合测试、访客规则 | 14 |
| `server/internal/bootstrap/issue_interleaving_relations_test.go`（新）、`server/internal/bootstrap/interleaving_test.go`（修改） | 交错 10、11、23、26 和加关联与删除工作区；`serializeOnTable` | 14 |
| `server/internal/bootstrap/permission_matrix_seed_issues_test.go`、`server/internal/bootstrap/permission_matrix_targets_test.go`（修改） | 种子的工作项带上关联的工作项；`{relation_id}` 的目标 | 14 |
| `server/internal/modules/issue/adapter/http/handler_test.go`（修改） | 测试的用例集 | 14 |
| `web/apps/web/core/lib/error-messages.ts`、`web/packages/i18n/src/locales/en/errors.json`、`web/packages/i18n/src/locales/zh-CN/errors.json`（修改） | `issue.relation_not_found` 的文案 | 14 |
| `e2e/fixtures/assert/relation.ts`（新） | `StoredRelation`、`expectRelations` | 15 |
| `e2e/stories/issue/i8-sub-issues.spec.ts`、`e2e/stories/issue/i9-relations.spec.ts`、`e2e/stories/issue/i17-delete-issue.spec.ts`（新）、`e2e/stories/issue/i15-archive-issue.spec.ts`、`e2e/stories/issue/i1-create-issue.spec.ts`、`e2e/stories/issue/i2-update-issue.spec.ts`（修改） | I8、I9、I17 的接口版本；I15 的子工作项、关联和它们的 409；I1、I2 的 `parent` | 15 |
| `docs/v0/v0-design.md`、`docs/v0/plane-diff.md`、`docs/v0/M4-issue-core/handoffs/M2-closeout.md`、`docs/v0/M4-issue-core/handoffs/M3-closeout.md`（修改） | 3.23 中 P3 的行；两份交接的处理结果 | 16 |

---

### Task 1: `deleteIssue` 连带子树

**Files:**
- Create: `server/internal/modules/issue/app/fakes_deletion_test.go`
- Modify: `api/modules/issue.yaml`、`server/internal/bootstrap/issue_deletion_test.go`、`server/internal/bootstrap/issue_world_test.go`、`server/internal/bootstrap/project_deletion_test.go`、`server/internal/modules/issue/adapter/postgres/delete_issue.go`、`server/internal/modules/issue/adapter/postgres/delete_issue_test.go`、`server/internal/modules/issue/adapter/postgres/queries/delete_issue.sql`、`server/internal/modules/issue/app/delete_issue.go`、`server/internal/modules/issue/app/delete_issue_test.go`、`server/internal/modules/issue/app/fakes_test.go`、`tools/keywords.json`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/issue/adapter/postgres/gen/delete_issue.sql.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**（spec 2.3、第 3 节第 5 条；M4 设计 3.5、3.6、3.8、3.9）父子的第一个 Task 是子树的删除：之后的 Task 一加父工作项，删除一个工作项就不会留下指向它的子工作项。
- `tools/keywords.json`：`phase` 改为 `M4/P3`（12 节约束 6）。
- 契约：`deleteIssue` 的描述写上它删除的是工作项和它下面的每个工作项（不论多深，已归档的也在）和它们的行。
- `issue/adapter/postgres/queries/delete_issue.sql`（整个文件）：`IssueSubtree`（`WITH RECURSIVE … UNION`：未删除的工作项和它下面每个未删除的工作项，`id` 的次序；在项目 N 之下读，每个写父工作项的写都持它）；`DeleteIssueRows` 取代 P2 的 `DeleteIssueRow`，它和四张子表的语句都按 `issue_id = ANY (issue_ids)` 一条语句写子树的全部行（约定五：在项目 N 之下，不按 `id` 先锁）。`delete_issue.go`（整个文件）：`Subtree`、`deletion`、六个方法。
- `issue/app/delete_issue.go`：`IssueDeletion.IssueIDs`（取代 `IssueID`）；`IssueRowsDeleter.Subtree`、`DeleteIssueRows`；`Execute` 在锁下、项目已归档的 409 之后读子树，然后读时刻，逐张表一条语句（`issueRowSteps`），都是调用者在那一刻的。
- 测试的帮手：删除的假实现（`fakeIssues` 的 `Subtree`、`subtree`、删除的记录）从 `fakes_test.go` 移进新的 `fakes_deletion_test.go`（`fakes_test.go` 已近 400 行）。
- `bootstrap`：`issue_deletion_test.go`（整个文件）：按目录读 `keysTo(t, pool, "issues")`（P2 的 `workItemKeys` 和 `subItems` 的豁免去掉），`underWorkItem` 直接把工作项放到父工作项下（不靠 Task 2 之后的写），`deletedWorkItem` 删除的工作项下有一个子工作项，`deletesAtOneMoment` 是两个子树的组合测试共用的核对（删除，然后子树的每个工作项和它下面的每一行在根的删除时刻由删除者删除、读不到了，给的别的工作项不动）；`project_deletion_test.go` 同样读 `keysTo(t, pool, "issues")`；`issue_world_test.go` 加 `adminGuest`（walt 成为 acme 的管理员、Web 的访客：过项目的每条规则，只看得到自己的工作项），本 Task 的管理员·访客的删除先用它，Task 2 起的访客规则的组合测试也用它（spec 第 3 节第 5 条，预检 N3）。

**Tests:**
- 单元：`TestDeleteIssueDeletesTheWorkItemsSubtreeAndItsRows`（取代 P2 的 `TestDeleteIssueDeletesTheWorkItemAndItsRows`：`Locks` 的调用和两次判定，锁下读子树，时刻，逐张表写子树的每个工作项；子树里删除的那个不是第一个，只删所问的或只删子树的第一个都看得出；已归档的照样删除）、`TestDeleteIssueStopsAtAFailingPort`（子树的读也失败）。
- 存储：`TestTheSubtreeOfAWorkItem`（它和它下面每个未删除的，不论多深，已归档的也在，`id` 的次序；已删除的工作项下面的不经它走到；另一棵树、在数据库里造的环 x、y 各读一次；`id` 的次序不是建或连的次序；已删除的、没有的工作项没有子树）、`TestTheStatementsOfADeletedSubtreeDeleteItsRows`（取代 P2 的 `TestTheStatementsOfADeletedWorkItemDeleteItsRows`：一条语句写给的每个工作项的未删除的行；更早删除的行和别的工作项的行、每一列都不动；别的工作项的 `id` 在给的几个之间，按次序取的写会碰到它）。
- 组合：`TestDeletingAWorkItemDeletesItsSubtreeAtOneMoment`（bob 的 root 在 alice 的 p 下，root 下有 alice 的 a、bob 的 b，a 下有已归档的 c；alice 删除 root：root、a、b、c 和它们下面的每一行在 root 的删除时刻由她删除，p、s、d 不动）；`TestAnAdminGuestsDeletionTakesTheSubItemsHeDoesNotSee`（spec 第 3 节第 5 条，预检 N3：子树整个删除，删除者看不到的也在；walt 是 acme 的管理员、Web 的访客（`adminGuest`），他的 mine 下有他看不到的 alice 的 a（404），a 下有她的 b；walt 删除 mine：mine、a、b 和它们下面的每一行在 mine 的删除时刻由他删除，alice 的 k 不动）；`TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt`（按目录，没有豁免：`issues.parent_id` 下也有一行）；`TestDeletingAProjectLeavesNoUndeletedRowUnderIt`。

- [ ] **Step 1: 关键词守卫、契约和语句，生成**

`api/modules/issue.yaml`（修改，1 处）：

````old api/modules/issue.yaml
        work item is deleted as any other. The work item, its assignees,
        labels, subscribers and activity are deleted by the caller at one
        moment; its number is not used again. A project with too many
````
````new api/modules/issue.yaml
        work item is deleted as any other. The work item and each work item
        under it, however deep, archived ones too, with their assignees,
        labels, subscribers and activity, are deleted by the caller at one
        moment; their numbers are not used again. A project with too many
````

`server/internal/modules/issue/adapter/postgres/queries/delete_issue.sql`（整个文件，48 行）：

````whole server/internal/modules/issue/adapter/postgres/queries/delete_issue.sql
-- name: IssueSubtree :many
-- deleteIssue's work items (M4 design 3.6, 3.9): the undeleted work item and each undeleted work item under it, however
-- deep, archived ones too, in id order. Read under the project's FOR NO KEY UPDATE, which every write of a parent holds
-- (3.5), so it stays as read until the deletion commits; a work item's parent is of its project, so its subtree is too.
-- UNION reads each work item once. The index on parent_id serves it.
WITH RECURSIVE subtree (id) AS (
    SELECT i.id FROM issues i WHERE i.id = sqlc.arg(id) AND i.deleted_at IS NULL
    UNION
    SELECT c.id FROM issues c JOIN subtree s ON c.parent_id = s.id WHERE c.deleted_at IS NULL
)
SELECT id FROM subtree ORDER BY id;

-- name: DeleteIssueRows :exec
-- deleteIssue's work items, its subtree (M4 design 3.6), under their project's FOR NO KEY UPDATE, archived or not:
-- soft-deleted at the moment and by the account given, in one statement, which locks them in its scan's order. Every
-- write of a work item's rows holds its project's lock first (3.5), so the rows are as the use case read them; they are
-- not locked by id first (M3 design 3.6, convention 5). A row deleted before keeps its deletion.
UPDATE issues
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE id = ANY (sqlc.arg(issue_ids)::uuid[]) AND deleted_at IS NULL;

-- name: DeleteIssueAssigneeRows :exec
-- The deleted work items' assignees (M4 design 3.6): their undeleted rows, soft-deleted at the moment and by the account
-- given, in one statement, which locks them in its scan's order. Every write of these rows holds the project's FOR NO
-- KEY UPDATE first (3.5): no other write reaches them meanwhile, so they are not locked by id first (M3 design 3.6,
-- convention 5). A row deleted before keeps its deletion. The index on issue_id serves it.
UPDATE issue_assignees
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE issue_id = ANY (sqlc.arg(issue_ids)::uuid[]) AND deleted_at IS NULL;

-- name: DeleteIssueLabelRows :exec
-- DeleteIssueAssigneeRows' statement, of the deleted work items' labels.
UPDATE issue_labels
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE issue_id = ANY (sqlc.arg(issue_ids)::uuid[]) AND deleted_at IS NULL;

-- name: DeleteIssueSubscriberRows :exec
-- DeleteIssueAssigneeRows' statement, of the deleted work items' subscribers.
UPDATE issue_subscribers
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE issue_id = ANY (sqlc.arg(issue_ids)::uuid[]) AND deleted_at IS NULL;

-- name: DeleteIssueActivityRows :exec
-- DeleteIssueAssigneeRows' statement, of the deleted work items' activity (3.6: it goes with its work item); the index
-- on (issue_id, created_at, id) serves it.
UPDATE issue_activities
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE issue_id = ANY (sqlc.arg(issue_ids)::uuid[]) AND deleted_at IS NULL;
````

`tools/keywords.json`（修改，1 处）：

````old tools/keywords.json
  "phase": "M4/P2",
````
````new tools/keywords.json
  "phase": "M4/P3",
````

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `509fe080d6f969941a3d3ca0fbe47d334922222aa334cf369fb302b0ecf972df` | 3334 | `api/dist/openapi.yaml` |
| `4cc8adbf84be07869eff38afb0b1efd8dc81b42e5bb93d448548b97d739a924c` | 143 | `server/internal/modules/issue/adapter/postgres/gen/delete_issue.sql.go` |
| `d10b1d160e685325dae8c67e8e92d98e4755cc60d12edcee7e8e845de513dd28` | 3696 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/issue/adapter/postgres/gen/delete_issue.sql.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 2: 用例**

`server/internal/modules/issue/app/delete_issue.go`（修改，9 处）：

````old server/internal/modules/issue/app/delete_issue.go
// IssueDeletion is a work item's deletion as its rows take it: the work
// item, and the moment and the account that every row it deletes is
// stamped with (M4 design 3.6).
````
````new server/internal/modules/issue/app/delete_issue.go
// IssueDeletion is a work item's deletion as its rows take it (M4 design
// 3.6, 3.9): the work items it deletes, the one asked for and each work
// item under it, however deep, archived ones too, its subtree; and the
// moment and the account that every row it deletes is stamped with.
````
````old server/internal/modules/issue/app/delete_issue.go
	IssueID uuid.UUID
	By      uuid.UUID
	At      time.Time
````
````new server/internal/modules/issue/app/delete_issue.go
	IssueIDs []uuid.UUID
	By       uuid.UUID
	At       time.Time
````
````old server/internal/modules/issue/app/delete_issue.go
// IssueRowsDeleter soft-deletes a deleted work item's rows, one table a
// call, in the transaction ctx carries, under its project's lock (M4
// design 3.5, 3.6): each method soft-deletes the work item's undeleted
// rows of its table at d.At by d.By, and leaves a row deleted before as it
// is.
````
````new server/internal/modules/issue/app/delete_issue.go
// IssueRowsDeleter reads a deleted work item's subtree and soft-deletes
// its rows, one table a call, in the transaction ctx carries, under its
// project's lock (M4 design 3.5, 3.6): each Delete method soft-deletes the
// undeleted rows of its table of the work items d deletes, at d.At by d.By,
// and leaves a row deleted before as it is.
````
````old server/internal/modules/issue/app/delete_issue.go
	DeleteIssueRow(ctx context.Context, d IssueDeletion) error
````
````new server/internal/modules/issue/app/delete_issue.go
	// Subtree reads the undeleted work item and each undeleted work item
	// under it, however deep, archived ones too, in id order.
	Subtree(ctx context.Context, id uuid.UUID) ([]uuid.UUID, error)
	DeleteIssueRows(ctx context.Context, d IssueDeletion) error
````
````old server/internal/modules/issue/app/delete_issue.go
// order of M4 design 3.5, as a project's (rowSteps): the work item, then
// the rows under it.
````
````new server/internal/modules/issue/app/delete_issue.go
// order of M4 design 3.5, as a project's (rowSteps): the work items, then
// the rows under them.
````
````old server/internal/modules/issue/app/delete_issue.go
		d.DeleteIssueRow, d.DeleteIssueAssigneeRows, d.DeleteIssueLabelRows, d.DeleteIssueSubscriberRows, d.DeleteIssueActivityRows,
````
````new server/internal/modules/issue/app/delete_issue.go
		d.DeleteIssueRows, d.DeleteIssueAssigneeRows, d.DeleteIssueLabelRows, d.DeleteIssueSubscriberRows, d.DeleteIssueActivityRows,
````
````old server/internal/modules/issue/app/delete_issue.go
// 3.5, 3.6, 11.11): its precheck, the work item read and the decision on
// issue.delete on it, the guest rule first, before any lock: the
````
````new server/internal/modules/issue/app/delete_issue.go
// 3.5, 3.6, 3.9, 11.11): its precheck, the work item read and the decision
// on issue.delete on it, the guest rule first, before any lock: the
````
````old server/internal/modules/issue/app/delete_issue.go
// project.archived); the clock; the work item and each row under it
// soft-deleted, table by table in the global order (issueRowSteps), all of
// it the caller's at that moment. An archived work item is deleted as any
// other (3.8). It deletes no other work item: its sub-items come with P3
// (3.9). A failing statement ends it with its failure, and none after it
// runs.
````
````new server/internal/modules/issue/app/delete_issue.go
// project.archived); its subtree, the work item and each one under it,
// however deep, archived ones too, read under the lock, which every write
// of a parent holds; the clock; the subtree's work items and each row
// under them soft-deleted, table by table in the global order
// (issueRowSteps), all of it the caller's at that moment. An archived work
// item is deleted as any other (3.8). A failing statement ends it with its
// failure, and none after it runs.
````
````old server/internal/modules/issue/app/delete_issue.go
		d := IssueDeletion{IssueID: id, By: actor.UserID, At: u.d.Clock.Now()}
````
````new server/internal/modules/issue/app/delete_issue.go
		subtree, err := u.d.Rows.Subtree(ctx, id)
		if err != nil {
			return err
		}
		d := IssueDeletion{IssueIDs: subtree, By: actor.UserID, At: u.d.Clock.Now()}
````

`server/internal/modules/issue/app/delete_issue_test.go`（修改，7 处）：

````old server/internal/modules/issue/app/delete_issue_test.go
import (
````
````new server/internal/modules/issue/app/delete_issue_test.go
import (
	"reflect"
````
````old server/internal/modules/issue/app/delete_issue_test.go
// deletingRows are the calls of the deletion of the work item id's rows,
// in the global order of M4 design 3.5: the work item, its assignees, its
// labels, its subscribers, its activity.
func deletingRows(id uuid.UUID) []string {
	return []string{line("DeleteIssueRow", id), line("DeleteIssueAssigneeRows", id), line("DeleteIssueLabelRows", id),
		line("DeleteIssueSubscriberRows", id), line("DeleteIssueActivityRows", id)}
````
````new server/internal/modules/issue/app/delete_issue_test.go
// deletingRows are the calls of the deletion of the rows of the work items
// ids, a subtree, in the global order of M4 design 3.5: the work items,
// their assignees, their labels, their subscribers, their activity.
func deletingRows(ids []uuid.UUID) []string {
	return []string{line("DeleteIssueRows", ids), line("DeleteIssueAssigneeRows", ids), line("DeleteIssueLabelRows", ids),
		line("DeleteIssueSubscriberRows", ids), line("DeleteIssueActivityRows", ids)}
````
````old server/internal/modules/issue/app/delete_issue_test.go
// on issue.delete (lockedAs); the clock; the work item's rows, table by
// table in the global order (deletingRows), each alice's at the moment. An
// archived work item is deleted as any other (M4 design 3.8).
func TestDeleteIssueDeletesTheWorkItemAndItsRows(t *testing.T) {
````
````new server/internal/modules/issue/app/delete_issue_test.go
// on issue.delete (lockedAs); the work item's subtree, read under the
// lock; the clock; the rows of each work item of the subtree, table by
// table in the global order (deletingRows), each alice's at the moment.
// The subtree holds two work items under the one deleted, which is not its
// first, so that a deletion of the work item asked for alone, or of the
// subtree's first alone, shows. An archived work item is deleted as any
// other (M4 design 3.8).
func TestDeleteIssueDeletesTheWorkItemsSubtreeAndItsRows(t *testing.T) {
````
````old server/internal/modules/issue/app/delete_issue_test.go
			w.issues.archivedAt = archivedAt
````
````new server/internal/modules/issue/app/delete_issue_test.go
			w.issues.archivedAt = archivedAt
			subtree := []uuid.UUID{uuid.NewV7(), id, uuid.NewV7()}
			w.issues.subtree = subtree
````
````old server/internal/modules/issue/app/delete_issue_test.go
			if calls := slices.Concat(lockedAs(domain.ActionDelete, id), []string{"Now"}, deletingRows(id)); !slices.Equal(w.log.calls, calls) {
````
````new server/internal/modules/issue/app/delete_issue_test.go
			calls := slices.Concat(lockedAs(domain.ActionDelete, id), []string{line("Subtree", id), "Now"}, deletingRows(subtree))
			if !slices.Equal(w.log.calls, calls) {
````
````old server/internal/modules/issue/app/delete_issue_test.go
			deletion := app.IssueDeletion{IssueID: id, By: alice, At: clockNow}
			if want := slices.Repeat([]app.IssueDeletion{deletion}, 5); !slices.Equal(w.issues.deletions, want) {
````
````new server/internal/modules/issue/app/delete_issue_test.go
			deletion := app.IssueDeletion{IssueIDs: subtree, By: alice, At: clockNow}
			if want := slices.Repeat([]app.IssueDeletion{deletion}, 5); !reflect.DeepEqual(w.issues.deletions, want) {
````
````old server/internal/modules/issue/app/delete_issue_test.go
	for _, port := range []string{"GetIssue", "Authorize", "Lock", "DeleteIssueRow", "DeleteIssueAssigneeRows", "DeleteIssueLabelRows",
		"DeleteIssueSubscriberRows", "DeleteIssueActivityRows"} {
````
````new server/internal/modules/issue/app/delete_issue_test.go
	for _, port := range []string{"GetIssue", "Authorize", "Lock", "Subtree", "DeleteIssueRows", "DeleteIssueAssigneeRows",
		"DeleteIssueLabelRows", "DeleteIssueSubscriberRows", "DeleteIssueActivityRows"} {
````

`server/internal/modules/issue/app/fakes_deletion_test.go`（新文件，54 行）：

````file server/internal/modules/issue/app/fakes_deletion_test.go
package app_test

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// fakeIssues as deleteIssue reads and writes the work items
// (app.IssueRowsDeleter): the subtree it is told, and each deletion's
// arguments kept, each call logged.

// Subtree logs the read and answers subtree, or the work item id alone
// when subtree is nil.
func (f *fakeIssues) Subtree(ctx context.Context, id uuid.UUID) ([]uuid.UUID, error) {
	if err := f.log.add(ctx, "Subtree", id); err != nil {
		return nil, err
	}
	if f.subtree == nil {
		return []uuid.UUID{id}, nil
	}
	return f.subtree, nil
}

// deleteRows logs a deletion of name's rows, d's work items', and keeps d,
// or answers the failure.
func (f *fakeIssues) deleteRows(ctx context.Context, name string, d app.IssueDeletion) error {
	if err := f.log.add(ctx, name, d.IssueIDs); err != nil {
		return err
	}
	f.deletions = append(f.deletions, d)
	return nil
}

func (f *fakeIssues) DeleteIssueRows(ctx context.Context, d app.IssueDeletion) error {
	return f.deleteRows(ctx, "DeleteIssueRows", d)
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
````

`server/internal/modules/issue/app/fakes_test.go`（修改，3 处）：

````old server/internal/modules/issue/app/fakes_test.go
// work item's archived_at, nil for none. deletions are the deletion's
// calls' arguments, one a table, in order.
````
````new server/internal/modules/issue/app/fakes_test.go
// work item's archived_at, nil for none. subtree is the work item's
// subtree, nil for the work item alone, and deletions are the deletion's
// calls' arguments, one a table, in order (fakes_deletion_test.go).
````
````old server/internal/modules/issue/app/fakes_test.go
	activities []app.ActivityRow
````
````new server/internal/modules/issue/app/fakes_test.go
	activities []app.ActivityRow
	subtree    []uuid.UUID
````
````old server/internal/modules/issue/app/fakes_test.go

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

````
````new server/internal/modules/issue/app/fakes_test.go

````

- [ ] **Step 3: 存储**

`server/internal/modules/issue/adapter/postgres/delete_issue.go`（整个文件，68 行）：

````whole server/internal/modules/issue/adapter/postgres/delete_issue.go
package postgresadapter

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// The statements of a deleted work item (app.IssueRowsDeleter,
// delete_issue.sql): its subtree, and the statements that soft-delete the
// subtree's undeleted rows of one table each.

// Subtree reads the undeleted work item id and each undeleted work item
// under it, however deep, archived ones too, in id order (M4 design 3.6,
// 3.9).
func (s *Store) Subtree(ctx context.Context, id uuid.UUID) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).IssueSubtree(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("read the work items under work item %s: %w", id, err)
	}
	return ids, nil
}

// deletion is d as the statements take it; their parameters are of one
// shape.
func deletion(d app.IssueDeletion) gen.DeleteIssueRowsParams {
	return gen.DeleteIssueRowsParams{At: d.At, DeletedBy: d.By, IssueIds: d.IssueIDs}
}

// deletedRows is err, the deletion of the work items d's rows of what, as
// the deletion's error, which gives how many work items d deletes, not
// their ids: a subtree is bounded only by its project. nil is nil.
func deletedRows(err error, what string, d app.IssueDeletion) error {
	if err != nil {
		return fmt.Errorf("delete %s of %d work items: %w", what, len(d.IssueIDs), err)
	}
	return nil
}

// DeleteIssueRows soft-deletes the work items, archived or not.
func (s *Store) DeleteIssueRows(ctx context.Context, d app.IssueDeletion) error {
	return deletedRows(s.queries(ctx).DeleteIssueRows(ctx, deletion(d)), "the rows", d)
}

// DeleteIssueAssigneeRows soft-deletes the work items' assignees' rows.
func (s *Store) DeleteIssueAssigneeRows(ctx context.Context, d app.IssueDeletion) error {
	return deletedRows(s.queries(ctx).DeleteIssueAssigneeRows(ctx, gen.DeleteIssueAssigneeRowsParams(deletion(d))), "the assignees", d)
}

// DeleteIssueLabelRows soft-deletes the rows of the labels on the work
// items.
func (s *Store) DeleteIssueLabelRows(ctx context.Context, d app.IssueDeletion) error {
	return deletedRows(s.queries(ctx).DeleteIssueLabelRows(ctx, gen.DeleteIssueLabelRowsParams(deletion(d))), "the labels", d)
}

// DeleteIssueSubscriberRows soft-deletes the work items' subscribers'
// rows.
func (s *Store) DeleteIssueSubscriberRows(ctx context.Context, d app.IssueDeletion) error {
	return deletedRows(s.queries(ctx).DeleteIssueSubscriberRows(ctx, gen.DeleteIssueSubscriberRowsParams(deletion(d))), "the subscribers", d)
}

// DeleteIssueActivityRows soft-deletes the work items' activity.
func (s *Store) DeleteIssueActivityRows(ctx context.Context, d app.IssueDeletion) error {
	return deletedRows(s.queries(ctx).DeleteIssueActivityRows(ctx, gen.DeleteIssueActivityRowsParams(deletion(d))), "the activity", d)
}
````

`server/internal/modules/issue/adapter/postgres/delete_issue_test.go`（修改，9 处）：

````old server/internal/modules/issue/adapter/postgres/delete_issue_test.go
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
````
````new server/internal/modules/issue/adapter/postgres/delete_issue_test.go
// A work item's subtree (M4 design 3.6, 3.9) is it and each undeleted work
// item under it, however deep, archived ones too, in id order. In web, the
// work item root has under it a, archived, and b; a has c under it; b has
// d under it, deleted, which has e under it, which is not: a deleted work
// item's are not reached through it. other is another top work item, with
// f under it, and in ops, x is under y and y under x, a loop no write
// makes: the read ends, each once. The work items are drawn so that the
// subtree's id order is none of the order they were made or linked in. A
// deleted or missing work item has none.
func TestTheSubtreeOfAWorkItem(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, ops := newProject(t, pool, acme, "WEB"), newProject(t, pool, acme, "OPS")
	c, b, root, f, e, a, d, other, x, y := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(),
		uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	for n, id := range []uuid.UUID{root, a, b, c, d, e, other, f} {
		newIssue(t, pool, web, id, n+1, alice)
	}
	newIssue(t, pool, ops, x, 1, alice)
	newIssue(t, pool, ops, y, 2, alice)
	for _, link := range [][2]uuid.UUID{{a, root}, {b, root}, {c, a}, {d, b}, {e, d}, {f, other}, {x, y}, {y, x}} {
		exec(t, pool, "UPDATE issues SET parent_id = $2 WHERE id = $1", link[0], link[1])
	}
	exec(t, pool, "UPDATE issues SET archived_at = $2 WHERE id = $1", a, now)
	exec(t, pool, "UPDATE issues SET deleted_at = $2 WHERE id = $1", d, now)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, tt := range []struct {
		name string
		id   uuid.UUID
		want []uuid.UUID
	}{
		{"root", root, slices.SortedFunc(slices.Values([]uuid.UUID{root, a, b, c}), uuid.UUID.Compare)},
		{"a, archived", a, slices.SortedFunc(slices.Values([]uuid.UUID{a, c}), uuid.UUID.Compare)},
		{"c, with none under it", c, []uuid.UUID{c}},
		{"other", other, slices.SortedFunc(slices.Values([]uuid.UUID{other, f}), uuid.UUID.Compare)},
		{"x, in a loop", x, slices.SortedFunc(slices.Values([]uuid.UUID{x, y}), uuid.UUID.Compare)},
		{"d, deleted", d, nil},
		{"no work item", uuid.NewV7(), nil},
	} {
		got, err := s.Subtree(ctx, tt.id)
		if err != nil || !slices.Equal(got, tt.want) {
			t.Errorf("Subtree(%s) = %v, %v; want %v", tt.name, got, err, tt.want)
		}
	}
	cancelled, stop := context.WithCancel(context.Background())
	stop()
	if _, err := s.Subtree(cancelled, root); !errors.Is(err, context.Canceled) {
		t.Errorf("Subtree() on a cancelled context = %v; want context.Canceled", err)
	}
}

// Each statement of a deleted subtree (M4 design 3.6, delete_issue.sql)
// soft-deletes the undeleted rows of its table of each work item it names,
// at the moment and by the account given, and nothing else. In web, the
// work item item has the rows a and b of the table (of the work items'
// table, item itself) and c, which alice deleted an hour before; the work
// item sub has its row d (sub itself); the work item gone has its row g,
// deleted then too (gone itself); the work item other has its row o,
// other's id after gone's and before item's and sub's, so that a statement
// that took its work items by the ids' order, not by equality, would reach
// o. One statement names sub, item and gone, in descending id order, by
// bob at later: a, b and d are deleted at later by bob, those of the first
// work item it names and of the others; c and g keep alice's deletion, and
// o its row as it was; every other column of every row is kept. Each
// statement answers a failed write as its failure.
func TestTheStatementsOfADeletedSubtreeDeleteItsRows(t *testing.T) {
````
````old server/internal/modules/issue/adapter/postgres/delete_issue_test.go
		{"issues", (*postgresadapter.Store).DeleteIssueRow},
````
````new server/internal/modules/issue/adapter/postgres/delete_issue_test.go
		{"issues", (*postgresadapter.Store).DeleteIssueRows},
````
````old server/internal/modules/issue/adapter/postgres/delete_issue_test.go
			gone, other, item := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
			for n, id := range []uuid.UUID{item, gone, other} {
````
````new server/internal/modules/issue/adapter/postgres/delete_issue_test.go
			gone, other, item, sub := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
			for n, id := range []uuid.UUID{item, gone, other, sub} {
````
````old server/internal/modules/issue/adapter/postgres/delete_issue_test.go
			deleted, earlier := []uuid.UUID{item}, []uuid.UUID{gone}
````
````new server/internal/modules/issue/adapter/postgres/delete_issue_test.go
			deleted, earlier := []uuid.UUID{item, sub}, []uuid.UUID{gone}
````
````old server/internal/modules/issue/adapter/postgres/delete_issue_test.go
				a, b, c, g, o := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
````
````new server/internal/modules/issue/adapter/postgres/delete_issue_test.go
				a, b, c, d, g, o := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
````
````old server/internal/modules/issue/adapter/postgres/delete_issue_test.go
				for n, row := range []struct{ issue, id uuid.UUID }{{item, a}, {item, b}, {item, c}, {gone, g}, {other, o}} {
````
````new server/internal/modules/issue/adapter/postgres/delete_issue_test.go
				for n, row := range []struct{ issue, id uuid.UUID }{{item, a}, {item, b}, {item, c}, {sub, d}, {gone, g}, {other, o}} {
````
````old server/internal/modules/issue/adapter/postgres/delete_issue_test.go
				deleted, earlier = []uuid.UUID{a, b}, []uuid.UUID{c, g}
````
````new server/internal/modules/issue/adapter/postgres/delete_issue_test.go
				deleted, earlier = []uuid.UUID{a, b, d}, []uuid.UUID{c, g}
````
````old server/internal/modules/issue/adapter/postgres/delete_issue_test.go
			for _, id := range []uuid.UUID{item, gone} {
				if err := st.method(s, context.Background(), app.IssueDeletion{IssueID: id, By: bob, At: later}); err != nil {
					t.Fatal(err)
				}
````
````new server/internal/modules/issue/adapter/postgres/delete_issue_test.go
			if err := st.method(s, context.Background(), app.IssueDeletion{IssueIDs: []uuid.UUID{sub, item, gone}, By: bob, At: later}); err != nil {
				t.Fatal(err)
````
````old server/internal/modules/issue/adapter/postgres/delete_issue_test.go
			if err := st.method(s, cancelled, app.IssueDeletion{IssueID: other, By: bob, At: later}); !errors.Is(err, context.Canceled) {
````
````new server/internal/modules/issue/adapter/postgres/delete_issue_test.go
			if err := st.method(s, cancelled, app.IssueDeletion{IssueIDs: []uuid.UUID{other}, By: bob, At: later}); !errors.Is(err, context.Canceled) {
````

- [ ] **Step 4: 按目录的删除和子树的组合测试**

`server/internal/bootstrap/issue_deletion_test.go`（整个文件，218 行）：

````whole server/internal/bootstrap/issue_deletion_test.go
package bootstrap

import (
	"net/http"
	"slices"
	"strings"
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
// under the work item, or the test puts one there (underWorkItem), and the
// deletion deletes it.

// underWorkItem puts each of items under the work item parent, directly:
// a work item's sub-items (M4 design 3.9), seeded so that the test does
// not depend on which of the issue module's writes exist.
func underWorkItem(t testing.TB, pool *pgxpool.Pool, parent uuid.UUID, items ...uuid.UUID) {
	t.Helper()
	if tag, err := pool.Exec(pgtest.Soon(t), "UPDATE issues SET parent_id = $1 WHERE id = ANY ($2)", parent, items); err != nil ||
		tag.RowsAffected() != int64(len(items)) {
		t.Fatalf("putting %v under %s: %d rows, %v; want %d", items, parent, tag.RowsAffected(), err, len(items))
	}
}

// deletedWorkItem is a work item that the caller of token, account, makes
// in project through the API, assigned to him and labelled the project's
// Bug, with another he makes under it (underWorkItem), and deletes through
// the API: a row of each table under it, deleted.
func deletedWorkItem(t *testing.T, contract *apitest.Contract, base string, pool *pgxpool.Pool, token string, account,
	project uuid.UUID) uuid.UUID {
	t.Helper()
	var bug uuid.UUID
	if err := pool.QueryRow(pgtest.Soon(t), "SELECT id FROM labels WHERE project_id = $1 AND name = 'Bug'", project).Scan(&bug); err != nil {
		t.Fatal(err)
	}
	var made [2]uuid.UUID
	for i, body := range []string{`{"name":"Gone","assignee_ids":["` + account.String() + `"],"label_ids":["` + bug.String() + `"]}`,
		`{"name":"Part"}`} {
		status, answer := call(t, contract, http.MethodPost, base+"/api/v0/projects/"+project.String()+"/issues", token, body)
		if status != http.StatusCreated {
			t.Fatalf("creating the work item = %d %s, want 201", status, answer)
		}
		var created struct {
			ID uuid.UUID `json:"id"`
		}
		decodeAnswer(t, answer, &created)
		made[i] = created.ID
	}
	gone := made[0]
	underWorkItem(t, pool, gone, made[1])
	if status, body := call(t, contract, http.MethodDelete, base+"/api/v0/issues/"+gone.String(), token, ""); status != http.StatusNoContent {
		t.Fatalf("deleting the work item = %d %s, want 204", status, body)
	}
	return gone
}

// Deleting a work item through the API soft-deletes every row under it in
// every table the catalog ties to issues, and the work item's row, each at
// its deleted_at, a time within the request, and by the account that
// deleted it; its rows deleted before keep their deletion, and nothing
// under its project's other work item changes. The work item is bob's,
// assigned to him and labelled Bug (fullIssue), whom he unassigned and
// unlabelled, then assigned and labelled again, with a work item of his
// under it (underWorkItem): a row of each table under it, and of its
// assignees and its labels one deleted before, all of bob's writing. The
// other work item, made the same way, has a row of each table under it,
// none deleted before. alice, Web's admin, deletes it: no row under it is
// of her writing before, so a deletion that kept a row's writer would
// show. The deleted work item is not found any more.
func TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt(t *testing.T) {
	w := newIssueWorld(t)
	deleted, kept := w.newWorkItemBy(t, "bob", w.web, w.fullIssue()), w.newWorkItemBy(t, "bob", w.web, w.fullIssue())
	for _, parent := range []uuid.UUID{deleted, kept} {
		underWorkItem(t, w.pool, parent, w.newWorkItemBy(t, "bob", w.web, `{"name":"Part"}`))
	}
	for _, body := range []string{`{"assignee_ids":[],"label_ids":[]}`,
		`{"assignee_ids":["` + w.ids["bob"].String() + `"],"label_ids":["` + w.bug.String() + `"]}`} {
		w.send(t, "bob", http.MethodPatch, "/api/v0/issues/"+deleted.String(), body, http.StatusOK)
	}
	keys := keysTo(t, w.pool, "issues")
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

// deletesAtOneMoment has the account by delete the work item root through
// the API, and checks that its subtree is deleted at one moment, by one
// deleter (M4 design 3.6, 3.9): each work item of subtree, root among
// them, by its name, is deleted, and changed last by by, at root's
// deleted_at, and so is every row under each (workItemStamps: each changed
// by by and deleted at the work item's last change); each is not found any
// more, by alice, who sees every work item of Web; and each work item of
// kept keeps every row as it was.
func (w issueWorld) deletesAtOneMoment(t *testing.T, by string, root uuid.UUID, subtree map[string]uuid.UUID, kept ...uuid.UUID) {
	t.Helper()
	before := make([][]string, len(kept))
	for i, id := range kept {
		before[i] = w.workItemStamps(t, id)
	}

	w.send(t, by, http.MethodDelete, "/api/v0/issues/"+root.String(), "", http.StatusNoContent)

	var at time.Time
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at FROM issues WHERE id = $1", root).Scan(&at); err != nil {
		t.Fatal(err)
	}
	for name, id := range subtree {
		var deletedAt, changedAt *time.Time
		var writer *uuid.UUID
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at, updated_at, updated_by_id FROM issues WHERE id = $1", id).Scan(&deletedAt,
			&changedAt, &writer); err != nil || deletedAt == nil || !deletedAt.Equal(at) || !changedAt.Equal(at) || writer == nil ||
			*writer != w.ids[by] {
			t.Errorf("%s: deleted at %v, changed at %v by %v, %v; want both at the root's %v, by %s", name, deletedAt, changedAt, writer, err, at,
				by)
		}
		for _, line := range w.workItemStamps(t, id) {
			if !strings.Contains(line, "changed by "+by+" changed, deleted changed") {
				t.Errorf("%s: %s; want it changed by %s and deleted at the deletion", name, line, by)
			}
		}
		w.send(t, "alice", http.MethodGet, "/api/v0/issues/"+id.String(), "", http.StatusNotFound)
	}
	for i, id := range kept {
		if got := w.workItemStamps(t, id); !slices.Equal(got, before[i]) {
			t.Errorf("the rows of %s:\n%q\nwant them as they were:\n%q", id, got, before[i])
		}
	}
}

// Deleting a work item through the API deletes its subtree at one moment,
// by one deleter (deletesAtOneMoment): the work item and each one under
// it, however deep, an archived one too, and every row under each. In
// Web, bob's root, assigned and labelled (fullIssue), is under alice's p
// and has under it alice's a, assigned and labelled too, and bob's b; a
// has c under it, in Done, archived; alice's s has her d under it. alice
// deletes root: root, a, b and c are deleted, by her, at root's
// deleted_at, with every row under each; p, s and d keep every row as it
// was.
func TestDeletingAWorkItemDeletesItsSubtreeAtOneMoment(t *testing.T) {
	w := newIssueWorld(t)
	p := w.newWorkItem(t)
	root := w.newWorkItemBy(t, "bob", w.web, w.fullIssue())
	a, b, c := w.newWorkItem(t), w.newWorkItemBy(t, "bob", w.web, `{"name":"Fix"}`), w.doneWorkItem(t)
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+c.String()+"/archive", "", http.StatusOK)
	s, d := w.newWorkItem(t), w.newWorkItem(t)
	underWorkItem(t, w.pool, p, root)
	underWorkItem(t, w.pool, root, a, b)
	underWorkItem(t, w.pool, a, c)
	underWorkItem(t, w.pool, s, d)
	w.deletesAtOneMoment(t, "alice", root, map[string]uuid.UUID{"root": root, "a": a, "b": b, "c": c}, p, s, d)
}

// A work item's subtree is deleted whole, the work items its deleter does
// not see among them (M4 design 3.6, 3.9: the subtree has no exception for
// the guest rule). walt, acme's admin and Web's guest (adminGuest), may
// delete his own work item and sees only his own. Under his mine,
// assigned and labelled (fullIssue), is alice's a, which he does not see
// (404), and under a is her b, both assigned and labelled too; alice's k
// is under none. walt deletes mine (deletesAtOneMoment): mine, a and b are
// deleted, by him, at mine's deleted_at, with every row under each; k
// keeps every row as it was.
func TestAnAdminGuestsDeletionTakesTheSubItemsHeDoesNotSee(t *testing.T) {
	w := newIssueWorld(t)
	w.adminGuest(t)
	mine := w.newWorkItemBy(t, "walt", w.web, w.fullIssue())
	a, b, k := w.newWorkItem(t), w.newWorkItem(t), w.newWorkItem(t)
	underWorkItem(t, w.pool, mine, a)
	underWorkItem(t, w.pool, a, b)
	w.send(t, "walt", http.MethodGet, "/api/v0/issues/"+a.String(), "", http.StatusNotFound)
	w.deletesAtOneMoment(t, "walt", mine, map[string]uuid.UUID{"mine": mine, "a": a, "b": b}, k)
}
````

`server/internal/bootstrap/issue_world_test.go`（修改，1 处）：

````old server/internal/bootstrap/issue_world_test.go
}

// doneWorkItem is the id of a work item alice creates in Web's Done
````
````new server/internal/bootstrap/issue_world_test.go
}

// adminGuest makes walt, acme's member, Web's guest, then acme's admin,
// through the API: a workspace admin who is only a guest of Web, which
// does not show its guests every work item (M4 design 3.4, 9.2). As
// acme's admin he passes each rule of Web (Grant.ProjectAdmin), and as
// its guest he sees only his own work items (Grant.OwnOnly).
func (w issueWorld) adminGuest(t *testing.T) {
	t.Helper()
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/members",
		`{"members":[{"member_id":"`+w.ids["walt"].String()+`","role":5}]}`, http.StatusCreated)
	membership := queryIDs(t, w.pool, "SELECT id FROM workspace_members WHERE workspace_id = $1 AND member_id = $2", w.acme, w.ids["walt"])[0]
	w.send(t, "alice", http.MethodPatch, "/api/v0/workspace-members/"+membership.String(), `{"role":20}`, http.StatusOK)
}

// doneWorkItem is the id of a work item alice creates in Web's Done
````

`server/internal/bootstrap/project_deletion_test.go`（修改，1 处）：

````old server/internal/bootstrap/project_deletion_test.go
	goneKeys := workItemKeys(t, pool)
````
````new server/internal/bootstrap/project_deletion_test.go
	goneKeys := keysTo(t, pool, "issues")
````

- [ ] **Step 5: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/issue/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestDeletingAWorkItemDeletesItsSubtreeAtOneMoment$|TestAnAdminGuestsDeletionTakesTheSubItemsHeDoesNotSee$|TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt$|TestDeletingAProjectLeavesNoUndeletedRowUnderIt$|TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt$' ./internal/bootstrap/`
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
Expected: 123 个全部通过（I15 删除的工作项没有子工作项，它的子树就是它自己）。

- [ ] **Step 6: 提交**

```bash
git add api/modules/issue.yaml server/internal/bootstrap/issue_deletion_test.go server/internal/bootstrap/issue_world_test.go server/internal/bootstrap/project_deletion_test.go server/internal/modules/issue/adapter/postgres/delete_issue.go server/internal/modules/issue/adapter/postgres/delete_issue_test.go server/internal/modules/issue/adapter/postgres/queries/delete_issue.sql server/internal/modules/issue/app/delete_issue.go server/internal/modules/issue/app/delete_issue_test.go server/internal/modules/issue/app/fakes_deletion_test.go server/internal/modules/issue/app/fakes_test.go tools/keywords.json api/dist/openapi.yaml server/internal/modules/issue/adapter/postgres/gen/delete_issue.sql.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M4/P3): deleteIssue deletes the work item's subtree at one moment

The phase keyword is M4/P3. A work item's deletion reads, under its
project's lock, its subtree: the work item and each undeleted work item
under it, however deep, archived ones too. The work items and every row
under them are soft-deleted by the caller at one moment, one statement a
table in the global order. The deletion against the catalog has no
exemption for sub-items any more.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p3.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试，或没有替身的平台、领域、转换的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；守卫是生成器（`make gen-go`）和中英文翻译键（`make lint-web`）的核对。每个变异在一份新的副本里，在本 Task 的树上、标出的较晚的 Task 的树上和最终的树上都跑过，端到端的检查只在标出的 Task 的树上）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T1.1` | 子树只有工作项自己：递归找不到它下面的工作项 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingAWorkItemDeletesItsSubtreeAtOneMoment`（`make test`）、`internal/bootstrap` 的 `TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt`（`make test`）、`internal/bootstrap` 的 `TestDeletingAWorkItemAndPuttingAnotherUnderItSerialize`（`make test`）（Task 6 起）、故事 `i17-delete-issue`（Task 15 起）、故事 `i15-archive-issue`（Task 15 起） | 存储；组合；端到端 |
| `T1.2` | 递归穿过已删除的工作项继续往下走 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`） | 存储 |
| `T1.3` | `UNION ALL`：成环的一条线永远读不完 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`） | 存储 |
| `T1.4` | 读已删除的工作项的子树 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`） | 存储 |
| `T1.5` | `deleteIssue` 只删问到的工作项，不删它的子树 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingAWorkItemDeletesItsSubtreeAtOneMoment`（`make test`）、`internal/bootstrap` 的 `TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt`（`make test`）、故事 `i17-delete-issue`（Task 15 起） | 单元；组合；端到端 |
| `T1.6` | `deleteIssue` 先读时刻、后读子树 | `internal/modules/issue/app` 的全部测试（`make test`） | 单元 |
| `T1.7` | 读子树失败时删除不报错地结束，什么也没删 | `internal/modules/issue/app` 的全部测试（`make test`） | 单元 |
| `T1.8` | 动态的语句只删子树第一个工作项的行 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingAWorkItemDeletesItsSubtreeAtOneMoment`（`make test`） | 存储；组合 |
| `T1.9` | 工作项的语句只删子树的第一个工作项 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingAWorkItemDeletesItsSubtreeAtOneMoment`（`make test`）、故事 `i17-delete-issue`（Task 15 起） | 存储；组合；端到端 |
| `T1.10` | 删除工作项经连接池读子树，不在它的事务的连接上 | `internal/bootstrap` 的 `TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection`（`make test`） | 组合 |

---

### Task 2: 父工作项：创建带 `parent_id`

**Files:**
- Create: `server/internal/bootstrap/issue_parents_test.go`、`server/internal/modules/issue/adapter/postgres/parents_test.go`、`server/internal/modules/issue/app/parents.go`、`server/internal/modules/issue/app/parents_test.go`、`server/internal/modules/issue/domain/hierarchy.go`
- Modify: `api/modules/issue.yaml`、`server/internal/bootstrap/issue_writes_test.go`、`server/internal/modules/issue/adapter/http/issues.go`、`server/internal/modules/issue/adapter/http/issues_test.go`、`server/internal/modules/issue/adapter/postgres/issues.go`、`server/internal/modules/issue/adapter/postgres/issues_test.go`、`server/internal/modules/issue/adapter/postgres/queries/issues.sql`、`server/internal/modules/issue/app/create_issue.go`、`server/internal/modules/issue/app/create_issue_test.go`、`server/internal/modules/issue/app/fakes_test.go`、`server/internal/modules/issue/app/lock.go`、`server/internal/modules/issue/app/rows.go`、`server/internal/modules/issue/domain/issue.go`、`server/internal/modules/issue/domain/issue_test.go`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/issue/adapter/http/gen/bodyshape.gen.go`、`server/internal/modules/issue/adapter/http/gen/server.gen.go`、`server/internal/modules/issue/adapter/postgres/gen/issues.sql.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**（spec 2.4、第 3 节第 9、12 条；M4 设计 3.4、3.5、3.8、3.9、3.13、5.2）
- 契约：`IssueCreate.parent_id`（可省略，不可为 `null`，`format: uuid`）；`createIssue` 的描述写上父工作项的规则（同一个项目、调用者看得到、没归档，锁下读）和 422 `parent_id`。
- `issue/adapter/postgres/queries/issues.sql`：`InsertIssue` 写 `parent_id`；`issues.go` 照写。
- `issue/domain`：`hierarchy.go`（新）：父工作项的规则的说明、`ParentNotAllowed()`（`parent_id` 的 `not_allowed`，一句固定的文字）；`NewIssue.ParentID`，`CheckNewIssue` 不看它（要读，留给锁下）。
- `issue/app/lock.go`：`decide` 交回 `Authorize` 的 `Grant`；`held.grant` 是锁下判定的 `Grant`（它的 `OwnOnly` 说访客规则是否向他隐藏别人的工作项）。
- `issue/app/parents.go`（新）：`parentOf(ctx, issues, h, actor, projectID, id)`：在锁下读所问的父工作项，是这个项目的、未删除、没归档、调用者看得到（访客规则用 `h.grant`，`Issue.HiddenFrom`）时回答它。
- `issue/app/create_issue.go`：`check(ctx, h, actor, projectID, in)` 在标签之后、编号之前读父工作项，问题与别的一次报出（422 `parent_id`）；写工作项时带上父工作项；创建不写父工作项的动态（3.13 的第 5 行是修改的）。`rows.go`：`IssueRow.ParentID`。
- 测试的帮手：`fakes_test.go` 的 `callLog.fail` 按整行比较（`GetIssue <id>` 一类的行），读父工作项失败的一行用它；`create_issue_test.go` 的 `wantStopped` 是端口失败的一个核对（答复是 `errDisk` 本身，所说的调用是最后一个），`wantStoppedAt`（端口的第一个调用，照旧按位置）和 `wantStoppedOn`（按整行的调用）都经它，读父工作项失败的核对用 `wantStoppedOn`。
- HTTP：`CreateIssue` 映射 `parent_id`。

**Tests:**
- 领域：`TestCheckNewIssueAccepts`（带父工作项的创建照样通过）。
- 单元：`TestCreateIssueUnderAParent`（在标签之后、编号之前锁下读父工作项，存下、答出它，动态只有创建的行）、`TestCreateIssueRefusesAParentItMayNotHave`（没有的、别的项目的、已归档的、锁下的判定说只看自己的调用者的别人的：都在别的问题之后、同一个 422，不占编号；只按预检的判定他看得到，在他自己的工作项下照样可以）、`TestAParentsFailingReadEndsTheWrite`（读父工作项失败原样结束这个写，之后的调用都不发生）。
- 存储：`TestAWorkItemsParentIsWritten`（`InsertIssue` 写下给的父工作项，`GetIssue` 读出；别的工作项的父工作项不动）。
- HTTP：`TestCreateIssue`（`parent_id` 进用例、回答带出）。
- 组合：`TestAWorkItemIsPutUnderAnother`（创建的一半：alice 在 Up 下建 Part，回答和存下的都在 Up 下，创建不写父工作项的动态；在不存在的工作项下、带不存在的标签建的，两个问题一次报出（422 `label_ids[0]`、`parent_id`），什么都没建）、`TestTheGuestRuleHoldsForAParent`（walt 是 acme 的管理员、Web 的访客，只看自己的：他在 alice 的 Up 下建的 422 `parent_id`，与不存在的一样，什么都没建；在他自己的工作项下照样建；Web 的成员 bob 在 Up 下建，回答和存下的都在 Up 下）、`TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection`（一个连接的池：alice 在第一个工作项下建 Part，锁下读父工作项的语句在它的事务的连接上）。

- [ ] **Step 1: 契约和查询，生成**

`api/modules/issue.yaml`（修改，2 处）：

````old api/modules/issue.yaml
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
````new api/modules/issue.yaml
        project who is not its guest; each label is one of the project's;
        the parent, when it is given, is a work item of the project that is
        not archived and that the caller sees (validation_failed, every
        problem at once). Left out, the state is the project's default, the
        priority none, and the assignee the project's default assignee while
        he is still an active member of it who is not its guest, else no
        one. The work item takes the project's next number, never used
        again, also after a deletion; it comes after its state's work items,
        archived ones too: its sort order is their greatest plus 10000, or
        65535 for the state's first; in a state of the completed group it is
        completed at its creation. The caller and the assignees subscribe to
        it; its activity says it was created, and assigned to each assignee.
        Every row it writes is the caller's, at one moment. A project with
        too many writes on its work items at once answers server_busy, with
        Retry-After: try again then.
````
````old api/modules/issue.yaml
          type: string
          format: date
    IssueUpdate:
````
````new api/modules/issue.yaml
          type: string
          format: date
        parent_id:
          description: >-
            The work item it is under: one of the project's that is not
            archived; left out for none.
          type: string
          format: uuid
    IssueUpdate:
````

`server/internal/modules/issue/adapter/postgres/queries/issues.sql`（修改，2 处）：

````old server/internal/modules/issue/adapter/postgres/queries/issues.sql
INSERT INTO issues (id, workspace_id, project_id, state_id, name, priority, start_date, target_date, sequence_id, sort_order,
                    completed_at, created_by_id, updated_by_id, created_at, updated_at)
````
````new server/internal/modules/issue/adapter/postgres/queries/issues.sql
INSERT INTO issues (id, workspace_id, project_id, state_id, name, priority, start_date, target_date, parent_id, sequence_id,
                    sort_order, completed_at, created_by_id, updated_by_id, created_at, updated_at)
````
````old server/internal/modules/issue/adapter/postgres/queries/issues.sql
        sqlc.narg(start_date), sqlc.narg(target_date), sqlc.arg(sequence_id), sqlc.arg(sort_order), sqlc.narg(completed_at),
        sqlc.arg(by)::uuid, sqlc.arg(by), sqlc.arg(at)::timestamptz, sqlc.arg(at));
````
````new server/internal/modules/issue/adapter/postgres/queries/issues.sql
        sqlc.narg(start_date), sqlc.narg(target_date), sqlc.narg(parent_id), sqlc.arg(sequence_id), sqlc.arg(sort_order),
        sqlc.narg(completed_at), sqlc.arg(by)::uuid, sqlc.arg(by), sqlc.arg(at)::timestamptz, sqlc.arg(at));
````

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `772dc46587973f45487bd60e0ee52cb57d32c28ccb97697b73437efa3ed9d6e4` | 3338 | `api/dist/openapi.yaml` |
| `749eaa3330e862c0aaaacf6a669c097a0cabf4abf99ef16ed1685775d6aa2674` | 39 | `server/internal/modules/issue/adapter/http/gen/bodyshape.gen.go` |
| `f6e1cd41662e6b0486433577a6eb8a00837d1606d8536695d02b93418844e363` | 1115 | `server/internal/modules/issue/adapter/http/gen/server.gen.go` |
| `bf65930a9994d597eb20f33caca418fa8f99f1ebd92d068e8fbceefb1bcc993a` | 305 | `server/internal/modules/issue/adapter/postgres/gen/issues.sql.go` |
| `fa33a1aa564b418576c7743ffbe2759a40ef8dac709426b48cefe97627a58864` | 3701 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/issue/adapter/http/gen/bodyshape.gen.go server/internal/modules/issue/adapter/http/gen/server.gen.go server/internal/modules/issue/adapter/postgres/gen/issues.sql.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 2: 领域和用例**

`server/internal/modules/issue/app/create_issue.go`（修改，7 处）：

````old server/internal/modules/issue/app/create_issue.go
// to and its labels, each in id order.
````
````new server/internal/modules/issue/app/create_issue.go
// to and its labels, each in id order, and its parent, nil for none.
````
````old server/internal/modules/issue/app/create_issue.go
	labels    []uuid.UUID
````
````new server/internal/modules/issue/app/create_issue.go
	labels    []uuid.UUID
	parent    *uuid.UUID
````
````old server/internal/modules/issue/app/create_issue.go
// items; the work item; its assignees, the default one too; its labels;
// its subscribers, the caller and the assignees, in id order; its
// activity: the creation, then each assignee by his display name
// (domain.CreationActivity). All of it is the caller's at that moment. The
// answer is the work item as stored, read back in the transaction.
````
````new server/internal/modules/issue/app/create_issue.go
// items; the work item, under its parent when it has one; its assignees,
// the default one too; its labels; its subscribers, the caller and the
// assignees, in id order; its activity: the creation, then each assignee
// by his display name (domain.CreationActivity), and no row of its parent
// (3.13). All of it is the caller's at that moment. The answer is the work
// item as stored, read back in the transaction.
````
````old server/internal/modules/issue/app/create_issue.go
		c, err := u.check(ctx, projectID, project, in)
````
````new server/internal/modules/issue/app/create_issue.go
		c, err := u.check(ctx, h, actor.UserID, projectID, in)
````
````old server/internal/modules/issue/app/create_issue.go
			StateID: c.state.ID, Priority: *in.Priority, StartDate: in.StartDate, TargetDate: in.TargetDate,
````
````new server/internal/modules/issue/app/create_issue.go
			StateID: c.state.ID, Priority: *in.Priority, StartDate: in.StartDate, TargetDate: in.TargetDate, ParentID: c.parent,
````
````old server/internal/modules/issue/app/create_issue.go
// check checks the dependent rows of in under the project's lock (M4
// design 3.7): the state asked for is the project's and not its triage
// state, or the project's default state when none is; the assignees by
// domain.Assignees, from the roles of the accounts asked for, or of the
// project's default assignee when none is; each label asked for is the
// project's. Every problem is reported at once, in one 422.
func (u *CreateIssue) check(ctx context.Context, projectID uuid.UUID, p LockedProject, in domain.NewIssue) (checked, error) {
````
````new server/internal/modules/issue/app/create_issue.go
// check checks the dependent rows of in, by the caller actor, under h, his
// write's lock on the project (M4 design 3.7, 3.9): the state asked for is
// the project's and not its triage state, or the project's default state
// when none is; the assignees by domain.Assignees, from the roles of the
// accounts asked for, or of the project's default assignee when none is;
// each label asked for is the project's; the parent asked for may be one
// (parentOf). Every problem is reported at once, in one 422.
func (u *CreateIssue) check(ctx context.Context, h held, actor, projectID uuid.UUID, in domain.NewIssue) (checked, error) {
	p := h.project
````
````old server/internal/modules/issue/app/create_issue.go
		c.labels = slices.SortedFunc(slices.Values(in.LabelIDs), uuid.UUID.Compare)
	}
````
````new server/internal/modules/issue/app/create_issue.go
		c.labels = slices.SortedFunc(slices.Values(in.LabelIDs), uuid.UUID.Compare)
	}
	if in.ParentID != nil {
		_, ok, err := parentOf(ctx, u.d.Issues, h, actor, projectID, *in.ParentID)
		if err != nil {
			return checked{}, err
		}
		if !ok {
			problems = append(problems, domain.ParentNotAllowed())
		}
		c.parent = in.ParentID
	}
````

`server/internal/modules/issue/app/create_issue_test.go`（修改，1 处）：

````old server/internal/modules/issue/app/create_issue_test.go
	last := slices.IndexFunc(w.log.calls, func(c string) bool { return strings.HasPrefix(c, port+" ") })
	if !errors.Is(err, errDisk) || last != len(w.log.calls)-1 {
		t.Errorf("Execute() = %v after the calls\n%q\nwant %v, %s the last call", err, w.log.calls, errDisk, port)
````
````new server/internal/modules/issue/app/create_issue_test.go
	first := slices.IndexFunc(w.log.calls, func(c string) bool { return strings.HasPrefix(c, port+" ") })
	wantStopped(t, w, err, port, first == len(w.log.calls)-1)
}

// wantStoppedOn checks a use case over w that failed on call, a port's
// call with its arguments as line writes it (callLog.fail): its answer is
// the failure, errDisk, as itself, and call is its last.
func wantStoppedOn(t *testing.T, w *world, err error, call string) {
	t.Helper()
	n := len(w.log.calls)
	wantStopped(t, w, err, call, n > 0 && w.log.calls[n-1] == call)
}

// wantStopped is the check of wantStoppedAt and wantStoppedOn: err is the
// failure, errDisk, as itself, and the call named is the last of w's
// calls, as last says.
func wantStopped(t *testing.T, w *world, err error, named string, last bool) {
	t.Helper()
	if !errors.Is(err, errDisk) || !last {
		t.Errorf("Execute() = %v after the calls\n%q\nwant %v, %s the last call", err, w.log.calls, errDisk, named)
````

`server/internal/modules/issue/app/fakes_test.go`（修改，7 处）：

````old server/internal/modules/issue/app/fakes_test.go
// transaction. It fails the call named in fail with errDisk, after logging
// it. inTx is set while a fakeTx's transaction runs.
````
````new server/internal/modules/issue/app/fakes_test.go
// transaction. It fails the call named in fail, or the one fail writes
// with its arguments as line does, with errDisk, after logging it. inTx is
// set while a fakeTx's transaction runs.
````
````old server/internal/modules/issue/app/fakes_test.go
	call := line(name, args...)
````
````new server/internal/modules/issue/app/fakes_test.go
	call := line(name, args...)
	failed := name == l.fail || call == l.fail
````
````old server/internal/modules/issue/app/fakes_test.go
	if name == l.fail {
````
````new server/internal/modules/issue/app/fakes_test.go
	if failed {
````
````old server/internal/modules/issue/app/fakes_test.go
// calls' arguments, one a table, in order (fakes_deletion_test.go).
````
````new server/internal/modules/issue/app/fakes_test.go
// calls' arguments, one a table, in order (fakes_deletion_test.go). others
// are the store's other work items, answered as they are.
````
````old server/internal/modules/issue/app/fakes_test.go
	deletions  []app.IssueDeletion
````
````new server/internal/modules/issue/app/fakes_test.go
	deletions  []app.IssueDeletion
	others     map[uuid.UUID]domain.Issue
````
````old server/internal/modules/issue/app/fakes_test.go
	if err := f.log.add(ctx, "GetIssue", id); err != nil || f.issue == nil || f.issue.ID != id {
		return domain.Issue{}, false, err
````
````new server/internal/modules/issue/app/fakes_test.go
	if err := f.log.add(ctx, "GetIssue", id); err != nil {
		return domain.Issue{}, false, err
	}
	if o, ok := f.others[id]; ok {
		return o, true, nil
	}
	if f.issue == nil || f.issue.ID != id {
		return domain.Issue{}, false, nil
````
````old server/internal/modules/issue/app/fakes_test.go
		StateID: r.StateID, Priority: r.Priority, StartDate: r.StartDate, TargetDate: r.TargetDate, SortOrder: r.SortOrder,
````
````new server/internal/modules/issue/app/fakes_test.go
		StateID: r.StateID, Priority: r.Priority, StartDate: r.StartDate, TargetDate: r.TargetDate, ParentID: r.ParentID, SortOrder: r.SortOrder,
````

`server/internal/modules/issue/app/lock.go`（修改，10 处）：

````old server/internal/modules/issue/app/lock.go
// held is what a write holds under its lock: its project, locked, and the
// work item it names, read under the lock; zero for a write on a project.
````
````new server/internal/modules/issue/app/lock.go
// held is what a write holds under its lock: its project, locked; the
// work item it names, read under the lock, zero for a write on a project;
// and the caller's grant on the project, as decided under the lock, whose
// OwnOnly says whether the guest rule hides others' work items from him
// (M4 design 3.4).
````
````old server/internal/modules/issue/app/lock.go
	issue   domain.Issue
````
````new server/internal/modules/issue/app/lock.go
	issue   domain.Issue
	grant   shared.Grant
````
````old server/internal/modules/issue/app/lock.go
	return w, l.decide(ctx, actor, w.action, workspaceID, w.project, nil, w.notFound)
````
````new server/internal/modules/issue/app/lock.go
	_, err = l.decide(ctx, actor, w.action, workspaceID, w.project, nil, w.notFound)
	return w, err
````
````old server/internal/modules/issue/app/lock.go
// the work item as read again. It answers what they hold. A project
// deleted while the lock waited is w.notFound, and so is a work item
// deleted meanwhile, or no longer in the project, a project no longer
// visible to actor, and a work item the guest rule now hides from him; a
// role the rule no longer allows is the Authorizer's shared.Forbidden.
````
````new server/internal/modules/issue/app/lock.go
// the work item as read again. It answers what they hold, the decision's
// grant too. A project deleted while the lock waited is w.notFound, and so
// is a work item deleted meanwhile, or no longer in the project, a project
// no longer visible to actor, and a work item the guest rule now hides from
// him; a role the rule no longer allows is the Authorizer's
// shared.Forbidden.
````
````old server/internal/modules/issue/app/lock.go
	if err := l.decide(ctx, actor, w.action, p.WorkspaceID, w.project, on, w.notFound); err != nil {
````
````new server/internal/modules/issue/app/lock.go
	if h.grant, err = l.decide(ctx, actor, w.action, p.WorkspaceID, w.project, on, w.notFound); err != nil {
````
````old server/internal/modules/issue/app/lock.go
	if err := l.decide(ctx, actor, action, i.WorkspaceID, i.ProjectID, &i, domain.ErrIssueNotFound); err != nil {
````
````new server/internal/modules/issue/app/lock.go
	if _, err := l.decide(ctx, actor, action, i.WorkspaceID, i.ProjectID, &i, domain.ErrIssueNotFound); err != nil {
````
````old server/internal/modules/issue/app/lock.go
	return l.decide(ctx, actor, action, workspaceID, uuid.UUID{}, nil, domain.ErrWorkspaceNotFound)
````
````new server/internal/modules/issue/app/lock.go
	_, err := l.decide(ctx, actor, action, workspaceID, uuid.UUID{}, nil, domain.ErrWorkspaceNotFound)
	return err
````
````old server/internal/modules/issue/app/lock.go
// action on the project or in the workspace.
````
````new server/internal/modules/issue/app/lock.go
// action on the project or in the workspace. It answers the Authorizer's
// grant, which a write keeps (held).
````
````old server/internal/modules/issue/app/lock.go
	notFound error) error {
````
````new server/internal/modules/issue/app/lock.go
	notFound error) (shared.Grant, error) {
````
````old server/internal/modules/issue/app/lock.go
	case errors.Is(err, shared.ErrNotVisible):
		return notFound
	case i != nil && i.HiddenFrom(actor.UserID, grant.OwnOnly):
		return notFound
	}
	return err
````
````new server/internal/modules/issue/app/lock.go
	case errors.Is(err, shared.ErrNotVisible):
		return shared.Grant{}, notFound
	case i != nil && i.HiddenFrom(actor.UserID, grant.OwnOnly):
		return shared.Grant{}, notFound
	}
	return grant, err
````

`server/internal/modules/issue/app/parents.go`（新文件，28 行）：

````file server/internal/modules/issue/app/parents.go
package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// The parent a write puts a work item under (M4 design 3.4, 3.8, 3.9),
// read under the project's lock, which every write of a parent holds (3.5),
// so that the work items' lines stay as read until the write commits.

// parentOf reads the work item id that the caller actor asks for as the
// parent of a work item of the project projectID, under h, his write's
// lock on it, and answers it when it may be one: an undeleted work item of
// the project, not archived, that he sees (the guest rule with h's grant,
// Issue.HiddenFrom); ok is false for any other.
func parentOf(ctx context.Context, issues IssueReader, h held, actor, projectID, id uuid.UUID) (p domain.Issue, ok bool, err error) {
	p, found, err := issues.GetIssue(ctx, id)
	if err != nil || !found {
		return domain.Issue{}, false, err
	}
	if p.ProjectID != projectID || p.ArchivedAt != nil || p.HiddenFrom(actor, h.grant.OwnOnly) {
		return domain.Issue{}, false, nil
	}
	return p, true, nil
}
````

`server/internal/modules/issue/app/parents_test.go`（新文件，113 行）：

````file server/internal/modules/issue/app/parents_test.go
package app_test

import (
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// kin are the work items a write may put another under, or may not (M4
// design 3.9), in web's world: up, web's work item 3, bob's; top, its work
// item 12, carol's; mine, its work item 10, alice's; shelved, its work
// item 9, archived; far, a work item of another project.
type kin struct{ up, top, mine, shelved, far domain.Issue }

// withKin stores kin in w, besides the work item w keeps, and answers it.
func withKin(w *world) kin {
	made := func(number int, by uuid.UUID) domain.Issue {
		return domain.Issue{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web, SequenceID: number, Name: "Part", StateID: backlog.ID,
			CreatedByID: &by}
	}
	k := kin{up: made(3, bob), top: made(12, carol), mine: made(10, alice), shelved: made(9, carol), far: made(4, carol)}
	k.shelved.ArchivedAt, k.far.ProjectID = &earlier, uuid.NewV7()
	w.issues.others = map[uuid.UUID]domain.Issue{}
	for _, i := range []domain.Issue{k.up, k.top, k.mine, k.shelved, k.far} {
		w.issues.others[i.ID] = i
	}
	return k
}

// A creation under a parent, web's work item 3, reads it under the lock
// after the labels and before the number (M4 design 3.5, 3.9): the work
// item is stored and answered under it, and its activity is the
// creation's alone, with no row of its parent (3.13).
func TestCreateIssueUnderAParent(t *testing.T) {
	uc, w := newCreate()
	k := withKin(w)
	got, err := uc.Execute(as(alice), web, domain.NewIssue{Name: "Part", LabelIDs: []uuid.UUID{bug}, ParentID: &k.up.ID})
	if err != nil || w.issues.issue == nil || !reflect.DeepEqual(w.issues.issue.ParentID, &k.up.ID) || !reflect.DeepEqual(got.ParentID,
		&k.up.ID) {
		t.Fatalf("Execute() = %+v, %v, stored %+v; want the work item under %s", got, err, w.issues.issue, k.up.ID)
	}
	id := w.issues.issue.ID
	want := created(id, backlog.ID, nil, []uuid.UUID{bug}, nil, []uuid.UUID{bug}, []uuid.UUID{alice}, 1)
	want = slices.Insert(want, slices.Index(want, line("Next", web)), line("GetIssue", k.up.ID))
	if !slices.Equal(w.log.calls, want) {
		t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, want)
	}
	wantActivity(t, w, id, domain.CreationActivity(nil))
}

// A parent a creation may not have is refused at parent_id, after the
// other problems, in one 422, and no number is taken (M4 design 3.4, 3.9):
// a work item not there, another project's, an archived one, and another's
// for a caller who sees only his own work items by the decision under the
// lock (Grant.OwnOnly). By the precheck's alone he may, and under his own
// work item too.
func TestCreateIssueRefusesAParentItMayNotHave(t *testing.T) {
	decided := []string{line("Find", web, "outside tx"), line("Authorize", alice, domain.ActionCreate, acme, web, "outside tx"), "Begin",
		line("Lock", web), line("Authorize", alice, domain.ActionCreate, acme, web), line("States", web)}
	stateNotAllowed := shared.FieldError{Field: "state_id", Code: "not_allowed", Message: "must be a state of the project that is not its triage state"}
	for _, tt := range []struct {
		name   string
		grants []shared.Grant
		parent func(k kin) uuid.UUID
	}{
		{"not there", nil, func(kin) uuid.UUID { return uuid.NewV7() }},
		{"another project's", nil, func(k kin) uuid.UUID { return k.far.ID }},
		{"archived", nil, func(k kin) uuid.UUID { return k.shelved.ID }},
		{"another's, for one who sees his own alone", []shared.Grant{{}, {OwnOnly: true}}, func(k kin) uuid.UUID { return k.up.ID }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w := newCreate()
			k := withKin(w)
			w.auth.grants = tt.grants
			parent := tt.parent(k)
			_, err := uc.Execute(as(alice), web, domain.NewIssue{Name: "a", StateID: &triage.ID, ParentID: &parent})
			refused(t, w, err, shared.Invalid(stateNotAllowed, domain.ParentNotAllowed()), append(slices.Clone(decided), line("GetIssue", parent)))
		})
	}
	for _, tt := range []struct {
		name   string
		grants []shared.Grant
		parent func(k kin) uuid.UUID
	}{
		{"another's, for one who sees his own alone by the precheck's decision", []shared.Grant{{OwnOnly: true}, {}},
			func(k kin) uuid.UUID { return k.up.ID }},
		{"his own, for one who sees his own alone", []shared.Grant{{OwnOnly: true}, {OwnOnly: true}}, func(k kin) uuid.UUID { return k.mine.ID }},
	} {
		uc, w := newCreate()
		k := withKin(w)
		w.auth.grants = tt.grants
		parent := tt.parent(k)
		if got, err := uc.Execute(as(alice), web, domain.NewIssue{Name: "a", ParentID: &parent}); err != nil || !reflect.DeepEqual(got.ParentID,
			&parent) {
			t.Errorf("%s: Execute() = %+v, %v; want the work item under %s", tt.name, got, err, parent)
		}
	}
}

// A failing read of a parent ends the write with its failure, as itself,
// and no call after it runs (M4 design 3.5): the parent a creation asks
// for.
func TestAParentsFailingReadEndsTheWrite(t *testing.T) {
	create, w := newCreate()
	k := withKin(w)
	w.log.fail = line("GetIssue", k.up.ID)
	_, err := create.Execute(as(alice), web, domain.NewIssue{Name: "a", ParentID: &k.up.ID})
	wantStoppedOn(t, w, err, w.log.fail)
}
````

`server/internal/modules/issue/app/rows.go`（修改，1 处）：

````old server/internal/modules/issue/app/rows.go
	StartDate   *shared.Date
	TargetDate  *shared.Date
	SortOrder   float64
````
````new server/internal/modules/issue/app/rows.go
	StartDate   *shared.Date
	TargetDate  *shared.Date
	ParentID    *uuid.UUID
	SortOrder   float64
````

`server/internal/modules/issue/domain/hierarchy.go`（新文件，19 行）：

````file server/internal/modules/issue/domain/hierarchy.go
package domain

import (
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// A work item's parent (M4 design 3.9): an undeleted work item of its
// project that it is under, however deep, or none. A work item is never
// under itself, nor under a work item under it, so the work items of a
// project make trees; an archived work item takes no new work item under
// it (3.8), and a guest who sees only his own work items puts none under
// another's (3.4).

// ParentNotAllowed is the problem of a parent asked for that may not be
// the work item's (M4 design 3.9).
func ParentNotAllowed() shared.FieldError {
	return shared.FieldError{Field: "parent_id", Code: shared.FieldNotAllowed,
		Message: "must be a work item of the project that is not archived, and neither this one nor under it"}
}
````

`server/internal/modules/issue/domain/issue.go`（修改，3 处）：

````old server/internal/modules/issue/domain/issue.go
// none. A field given is checked as given, a priority given empty too.
````
````new server/internal/modules/issue/domain/issue.go
// none, ParentID nil for no parent. A field given is checked as given, a
// priority given empty too.
````
````old server/internal/modules/issue/domain/issue.go
	StartDate   *shared.Date
	TargetDate  *shared.Date
}
````
````new server/internal/modules/issue/domain/issue.go
	StartDate   *shared.Date
	TargetDate  *shared.Date
	ParentID    *uuid.UUID
}
````
````old server/internal/modules/issue/domain/issue.go
// assignee and each label may be is the use case's, under the project's
// lock.
````
````new server/internal/modules/issue/domain/issue.go
// assignee, each label and the parent may be is the use case's, under the
// project's lock.
````

`server/internal/modules/issue/domain/issue_test.go`（修改，3 处）：

````old server/internal/modules/issue/domain/issue_test.go
// assignees and labels, the dates in order, on the same day, or one alone:
// accepted, and returned as stored: the name without its surrounding white
// space, the priority none when none is given, the dates as given.
````
````new server/internal/modules/issue/domain/issue_test.go
// assignees and labels, the dates in order, on the same day, or one alone,
// a parent: accepted, and returned as stored: the name without its
// surrounding white space, the priority none when none is given, the dates
// and the parent as given.
````
````old server/internal/modules/issue/domain/issue_test.go
	state := uuid.NewV7()
````
````new server/internal/modules/issue/domain/issue_test.go
	state, parent := uuid.NewV7(), uuid.NewV7()
````
````old server/internal/modules/issue/domain/issue_test.go
			StartDate: october(12), TargetDate: october(19)},
			NewIssue{Name: "Fix", StateID: &state, Priority: ptr(PriorityUrgent), StartDate: october(12), TargetDate: october(19)}},
````
````new server/internal/modules/issue/domain/issue_test.go
			StartDate: october(12), TargetDate: october(19), ParentID: &parent},
			NewIssue{Name: "Fix", StateID: &state, Priority: ptr(PriorityUrgent), StartDate: october(12), TargetDate: october(19),
				ParentID: &parent}},
````

- [ ] **Step 3: 存储和 HTTP**

`server/internal/modules/issue/adapter/http/issues.go`（修改，2 处）：

````old server/internal/modules/issue/adapter/http/issues.go
// the priority and the dates nil and the lists empty when not given; a
// value given goes to the use case as it came, which checks it.
````
````new server/internal/modules/issue/adapter/http/issues.go
// the priority, the dates and the parent nil and the lists empty when not
// given; a value given goes to the use case as it came, which checks it.
````
````old server/internal/modules/issue/adapter/http/issues.go
		StartDate: req.Body.StartDate, TargetDate: req.Body.TargetDate}
````
````new server/internal/modules/issue/adapter/http/issues.go
		StartDate: req.Body.StartDate, TargetDate: req.Body.TargetDate, ParentID: req.Body.ParentID}
````

`server/internal/modules/issue/adapter/http/issues_test.go`（修改，3 处）：

````old server/internal/modules/issue/adapter/http/issues_test.go
// the body's fields: the state, the priority and the dates nil and the
// lists empty when not given; the answer is 201 with the work item the use
// case answers, every field of it, null for each it has none of, [] for a
// list without ids.
````
````new server/internal/modules/issue/adapter/http/issues_test.go
// the body's fields: the state, the priority, the dates and the parent nil
// and the lists empty when not given; the answer is 201 with the work item
// the use case answers, every field of it, null for each it has none of,
// [] for a list without ids.
````
````old server/internal/modules/issue/adapter/http/issues_test.go
			`"start_date":"2026-10-12","target_date":"2026-10-19"}`,
````
````new server/internal/modules/issue/adapter/http/issues_test.go
			`"start_date":"2026-10-12","target_date":"2026-10-19","parent_id":"0199a2b4-0000-7000-8000-0000000000e6"}`,
````
````old server/internal/modules/issue/adapter/http/issues_test.go
				LabelIDs: []uuid.UUID{bugID}, StartDate: &startDay, TargetDate: &targetDay}, webSeven, webSevenJSON},
````
````new server/internal/modules/issue/adapter/http/issues_test.go
				LabelIDs: []uuid.UUID{bugID}, StartDate: &startDay, TargetDate: &targetDay, ParentID: &webSixID}, webSeven, webSevenJSON},
````

`server/internal/modules/issue/adapter/postgres/issues.go`（修改，1 处）：

````old server/internal/modules/issue/adapter/postgres/issues.go
		StartDate: timeOf(r.StartDate), TargetDate: timeOf(r.TargetDate), SequenceID: int32(r.SequenceID), SortOrder: r.SortOrder,
		CompletedAt: r.CompletedAt, By: &r.By, At: r.At,
````
````new server/internal/modules/issue/adapter/postgres/issues.go
		StartDate: timeOf(r.StartDate), TargetDate: timeOf(r.TargetDate), ParentID: r.ParentID, SequenceID: int32(r.SequenceID),
		SortOrder: r.SortOrder, CompletedAt: r.CompletedAt, By: &r.By, At: r.At,
````

`server/internal/modules/issue/adapter/postgres/issues_test.go`（修改，1 处）：

````old server/internal/modules/issue/adapter/postgres/issues_test.go
// none at its default.
````
````new server/internal/modules/issue/adapter/postgres/issues_test.go
// none at its default, but its parent, which TestAWorkItemsParentIsWritten
// writes.
````

`server/internal/modules/issue/adapter/postgres/parents_test.go`（新文件，56 行）：

````file server/internal/modules/issue/adapter/postgres/parents_test.go
package postgresadapter_test

import (
	"context"
	"reflect"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// A work item's parent through the store (M4 design 3.9): InsertIssue
// writes the parent it is given, up, and GetIssue reads it. Every other
// work item keeps its parent: other is under up, and up and top under
// none.
func TestAWorkItemsParentIsWritten(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	alice := newAccount(t, pool, "alice@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	up, top, other, id := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	for n, i := range []uuid.UUID{up, top, other} {
		newIssue(t, pool, web, i, n+1, alice)
	}
	exec(t, pool, "UPDATE issues SET parent_id = $2 WHERE id = $1", other, up)
	parents := func() map[uuid.UUID]*uuid.UUID {
		t.Helper()
		rows, err := pool.Query(ctx, "SELECT id, parent_id FROM issues")
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		out := map[uuid.UUID]*uuid.UUID{}
		for rows.Next() {
			var i uuid.UUID
			var parent *uuid.UUID
			if err := rows.Scan(&i, &parent); err != nil {
				t.Fatal(err)
			}
			out[i] = parent
		}
		return out
	}

	if err := s.InsertIssue(ctx, app.IssueRow{ID: id, WorkspaceID: web.workspace, ProjectID: web.id, SequenceID: 4, Name: "Part",
		StateID: web.state, Priority: domain.PriorityNone, ParentID: &up, SortOrder: 65535, By: alice, At: now}); err != nil {
		t.Fatal(err)
	}
	if got, found, err := s.GetIssue(ctx, id); err != nil || !found || !reflect.DeepEqual(got.ParentID, &up) {
		t.Errorf("GetIssue() = %+v, %v, %v; want it under %s", got, found, err, up)
	}
	if got, want := parents(), map[uuid.UUID]*uuid.UUID{up: nil, top: nil, other: &up, id: &up}; !reflect.DeepEqual(got, want) {
		t.Errorf("the parents %v, want %v", got, want)
	}
}
````

- [ ] **Step 4: 组合**

`server/internal/bootstrap/issue_parents_test.go`（新文件，115 行）：

````file server/internal/bootstrap/issue_parents_test.go
package bootstrap

import (
	"net/http"
	"reflect"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// parentNotAllowed is the refusal of a parent a work item may not have, as
// refusalOf writes it with its messages (M4 design 3.9).
const parentNotAllowed = "validation_failed parent_id not_allowed: must be a work item of the project that is not archived, and neither " +
	"this one nor under it"

// parentOf is the parent of the work item id as the database has it, nil
// for none.
func (w issueWorld) parentOf(t *testing.T, id uuid.UUID) *uuid.UUID {
	t.Helper()
	var parent *uuid.UUID
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT parent_id FROM issues WHERE id = $1", id).Scan(&parent); err != nil {
		t.Fatal(err)
	}
	return parent
}

// parentRows are the rows of the work item item's activity on its parent
// (row 5 of M4 design 3.13), in their ids' order, each as "<actor>:
// <old value> → <new value>, <old identifier> → <new identifier>", an
// account by its name in the world, an identifier none for NULL.
func (w issueWorld) parentRows(t *testing.T, item uuid.UUID) []string {
	t.Helper()
	rows, err := w.pool.Query(pgtest.Soon(t), `SELECT actor_id, old_value, new_value, coalesce(old_identifier::text, 'none'),
		coalesce(new_identifier::text, 'none') FROM issue_activities WHERE issue_id = $1 AND field = 'parent' ORDER BY id`, item)
	if err != nil {
		t.Fatal(err)
	}
	names := make(map[uuid.UUID]string, len(w.ids))
	for name, account := range w.ids {
		names[account] = name
	}
	said, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (string, error) {
		var actor uuid.UUID
		var oldValue, newValue, oldID, newID string
		if err := row.Scan(&actor, &oldValue, &newValue, &oldID, &newID); err != nil {
			return "", err
		}
		return names[actor] + ": " + oldValue + " → " + newValue + ", " + oldID + " → " + newID, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return said
}

// underParent is the body of a work item named name under parent.
func underParent(name string, parent uuid.UUID) string {
	return `{"name":"` + name + `","parent_id":"` + parent.String() + `"}`
}

// Parents as bootstrap wires them (M4 design 3.5, 3.9, 3.13). alice makes
// Up, and Part under Up: Part is answered and stored under Up, and its
// creation writes no row of its parent. A work item under one not there,
// with a label not there, is refused with both problems at once (422),
// and not made.
func TestAWorkItemIsPutUnderAnother(t *testing.T) {
	w := newIssueWorld(t)
	up := w.newWorkItemIn(t, w.web, `{"name":"Up"}`)
	var part storedIssue
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/issues", underParent("Part", up),
		http.StatusCreated), &part)
	if !reflect.DeepEqual(part.ParentID, &up) || !reflect.DeepEqual(w.parentOf(t, part.ID), &up) || len(w.parentRows(t, part.ID)) != 0 {
		t.Errorf("Part answered under %v, stored under %v, its parent's rows %q; want under Up, %s, and none", part.ParentID,
			w.parentOf(t, part.ID), w.parentRows(t, part.ID), up)
	}
	count := w.workItems(t)
	lost := w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/issues", `{"name":"Lost","parent_id":"`+
		uuid.NewV7().String()+`","label_ids":["`+uuid.NewV7().String()+`"]}`, http.StatusUnprocessableEntity)
	if got := refusalOf(t, []byte(lost), false); got != "validation_failed label_ids[0] not_allowed; parent_id not_allowed" ||
		w.workItems(t) != count {
		t.Errorf("a work item under one not there, with a label not there, answers %s, and Web has %d work items; want label_ids[0] and "+
			"parent_id not_allowed at once, and %d", got, w.workItems(t), count)
	}
}

// The guest rule holds for a parent (M4 design 3.4, 3.9): walt, acme's
// admin and Web's guest (adminGuest), sees only his own work items in Web.
// His new work item under alice's Up is refused, 422 at parent_id, as
// under one not there: no work item made. bob, Web's member, makes one
// under Up, answered and stored under it. Under his own work item walt
// makes one.
func TestTheGuestRuleHoldsForAParent(t *testing.T) {
	w := newIssueWorld(t)
	w.adminGuest(t)
	up := w.newWorkItemIn(t, w.web, `{"name":"Up"}`)
	his := w.newWorkItemBy(t, "walt", w.web, `{"name":"His"}`)
	create := "/api/v0/projects/" + w.web.String() + "/issues"
	answer := w.send(t, "walt", http.MethodPost, create, underParent("Part", up), http.StatusUnprocessableEntity)
	if got := refusalOf(t, []byte(answer), true); got != parentNotAllowed || w.workItems(t) != 2 {
		t.Errorf("walt's work item under Up answers %s, and Web has %d work items; want %s, and 2", got, w.workItems(t), parentNotAllowed)
	}
	var sub storedIssue
	decodeAnswer(t, w.send(t, "bob", http.MethodPost, create, underParent("Sub", up), http.StatusCreated), &sub)
	if !reflect.DeepEqual(sub.ParentID, &up) || !reflect.DeepEqual(w.parentOf(t, sub.ID), &up) {
		t.Errorf("bob's work item under Up is answered under %v, stored under %v; want under Up, %s", sub.ParentID,
			w.parentOf(t, sub.ID), up)
	}
	below := w.newWorkItemBy(t, "walt", w.web, underParent("Below", his))
	if got := w.parentOf(t, below); !reflect.DeepEqual(got, &his) {
		t.Errorf("Below is under %v, want His, %s", got, his)
	}
}
````

`server/internal/bootstrap/issue_writes_test.go`（修改，2 处）：

````old server/internal/bootstrap/issue_writes_test.go
// archives it, in Done, and unarchives it; she deletes the second. With
// Web archived, her creation and her change are refused under the lock
// (409).
````
````new server/internal/bootstrap/issue_writes_test.go
// makes Part under it, its parent read under the lock. She archives it,
// in Done, and unarchives it; she deletes the second. With Web archived,
// her creation and her change are refused under the lock (409).
````
````old server/internal/bootstrap/issue_writes_test.go
	route.answerWithin(t, w.contract, http.MethodGet, "/api/v0/issues/"+first.ID.String(), alice, "", http.StatusOK)
````
````new server/internal/bootstrap/issue_writes_test.go
	route.answerWithin(t, w.contract, http.MethodGet, "/api/v0/issues/"+first.ID.String(), alice, "", http.StatusOK)
	var part struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, send(alice, `{"name":"Part","parent_id":"`+first.ID.String()+`"}`, http.StatusCreated), &part)
````

- [ ] **Step 5: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/issue/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestAWorkItemIsPutUnderAnother$|TestTheGuestRuleHoldsForAParent$|TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection$' ./internal/bootstrap/`
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
git add api/modules/issue.yaml server/internal/bootstrap/issue_parents_test.go server/internal/bootstrap/issue_writes_test.go server/internal/modules/issue/adapter/http/issues.go server/internal/modules/issue/adapter/http/issues_test.go server/internal/modules/issue/adapter/postgres/issues.go server/internal/modules/issue/adapter/postgres/issues_test.go server/internal/modules/issue/adapter/postgres/parents_test.go server/internal/modules/issue/adapter/postgres/queries/issues.sql server/internal/modules/issue/app/create_issue.go server/internal/modules/issue/app/create_issue_test.go server/internal/modules/issue/app/fakes_test.go server/internal/modules/issue/app/lock.go server/internal/modules/issue/app/parents.go server/internal/modules/issue/app/parents_test.go server/internal/modules/issue/app/rows.go server/internal/modules/issue/domain/hierarchy.go server/internal/modules/issue/domain/issue.go server/internal/modules/issue/domain/issue_test.go api/dist/openapi.yaml server/internal/modules/issue/adapter/http/gen/bodyshape.gen.go server/internal/modules/issue/adapter/http/gen/server.gen.go server/internal/modules/issue/adapter/postgres/gen/issues.sql.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M4/P3): a work item is made under a parent

createIssue takes parent_id: under the project's lock, after the labels
and before the number, the parent is read and must be an undeleted work
item of the project, not archived, that the caller sees by the decision
under the lock (Locks keeps its grant); any other is a 422 at parent_id
with the other problems, and no number is taken. The creation writes no
row of the parent.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p3.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试，或没有替身的平台、领域、转换的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；守卫是生成器（`make gen-go`）和中英文翻译键（`make lint-web`）的核对。每个变异在一份新的副本里，在本 Task 的树上、标出的较晚的 Task 的树上和最终的树上都跑过，端到端的检查只在标出的 Task 的树上）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T2.1` | 别的项目的工作项可以当父工作项 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemIsPutUnderAnother`（`make test`）（Task 3 起） | 单元；组合 |
| `T2.2` | 已归档的工作项可以当父工作项 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemIsPutUnderAnother`（`make test`）（Task 3 起）、故事 `i8-sub-issues`（Task 15 起） | 单元；组合；端到端 |
| `T2.3` | 父工作项不经访客规则：只看自己的访客把工作项放到别人的工作项下面 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestTheGuestRuleHoldsForAParent`（`make test`）、`internal/bootstrap` 的 `TestTheGuestRuleHoldsForSubItems`（`make test`）（Task 6 起） | 单元；组合 |
| `T2.4` | 写不保留锁下判定的 `Grant`：那里的访客规则看得到每个工作项 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestTheGuestRuleHoldsForAParent`（`make test`）、`internal/bootstrap` 的 `TestEachAnswerCountsTheSubItemsTheCallerSees`（`make test`）（Task 4 起）、`internal/bootstrap` 的 `TestTheGuestRuleHoldsForSubItems`（`make test`）（Task 6 起） | 单元；组合 |
| `T2.5` | `createIssue` 检查了父工作项却不存它 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemIsPutUnderAnother`（`make test`）、故事 `i8-sub-issues`（Task 15 起） | 单元；组合；端到端 |
| `T2.6` | `InsertIssue` 丢掉父工作项 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemIsPutUnderAnother`（`make test`） | 存储；组合 |
| `T2.7` | `POST` 丢掉请求体的 `parent_id` | `internal/modules/issue/adapter/http` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemIsPutUnderAnother`（`make test`）、故事 `i8-sub-issues`（Task 15 起） | 单元；组合；端到端 |
| `T2.8` | 创建不能有的父工作项单独答出，不带别的问题 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemIsPutUnderAnother`（`make test`） | 单元；组合 |
| `T2.9` | 读父工作项失败当作它不存在 | `internal/modules/issue/app` 的全部测试（`make test`） | 单元 |
| `T2.10` | 父工作项的拒绝写在 `parent` 而不是 `parent_id` | `internal/bootstrap` 的 `TestTheGuestRuleHoldsForAParent`（`make test`）、`internal/bootstrap` 的 `TestAWorkItemIsPutUnderAnother`（`make test`）、故事 `i8-sub-issues`（Task 15 起） | 组合；端到端 |
| `T2.11` | `CheckNewIssue` 丢掉要的父工作项 | `internal/modules/issue/domain` 的全部测试（`make test`）、`internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemIsPutUnderAnother`（`make test`） | 单元；组合 |
| `T2.12` | 删除者只看自己的工作项时，子树去掉他看不到的子工作项 | `internal/bootstrap` 的 `TestAnAdminGuestsDeletionTakesTheSubItemsHeDoesNotSee`（`make test`） | 组合 |
| `T2.13` | 父工作项只能是调用者自己建的 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestTheGuestRuleHoldsForAParent`（`make test`） | 单元；组合 |
| `T2.14` | 带父工作项的创建写第 5 行（改父工作项的一行） | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemIsPutUnderAnother`（`make test`） | 单元；组合 |

---

### Task 3: 父工作项：修改移动工作项，不成环

**Files:**
- Create: `server/internal/bootstrap/issue_interleaving_hierarchy_test.go`、`server/internal/modules/issue/adapter/postgres/parents.go`、`server/internal/modules/issue/adapter/postgres/queries/parents.sql`、`server/internal/modules/issue/app/fakes_parents_test.go`
- Modify: `api/modules/issue.yaml`、`server/internal/bootstrap/issue_parents_test.go`、`server/internal/bootstrap/issue_writes_test.go`、`server/internal/bootstrap/ports_test.go`、`server/internal/modules/issue/adapter/http/issues.go`、`server/internal/modules/issue/adapter/http/update_test.go`、`server/internal/modules/issue/adapter/postgres/parents_test.go`、`server/internal/modules/issue/adapter/postgres/queries/update.sql`、`server/internal/modules/issue/adapter/postgres/update.go`、`server/internal/modules/issue/app/create_issue_test.go`、`server/internal/modules/issue/app/fakes_test.go`、`server/internal/modules/issue/app/parents.go`、`server/internal/modules/issue/app/parents_test.go`、`server/internal/modules/issue/app/ports.go`、`server/internal/modules/issue/app/rows.go`、`server/internal/modules/issue/app/update_issue.go`、`server/internal/modules/issue/domain/activity.go`、`server/internal/modules/issue/domain/activity_test.go`、`server/internal/modules/issue/domain/dates.go`、`server/internal/modules/issue/domain/hierarchy.go`、`server/internal/modules/issue/domain/identifier.go`、`server/internal/modules/issue/domain/identifier_test.go`、`server/internal/modules/issue/domain/patch.go`、`server/internal/modules/issue/domain/patch_test.go`、`server/internal/modules/project/adapter/postgres/issue_ports.go`、`server/internal/modules/project/adapter/postgres/issue_ports_test.go`、`server/internal/modules/project/adapter/postgres/queries/issue_ports.sql`、`server/internal/modules/project/app/issue_ports.go`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/issue/adapter/http/gen/bodyshape.gen.go`、`server/internal/modules/issue/adapter/http/gen/server.gen.go`、`server/internal/modules/issue/adapter/postgres/gen/parents.sql.go`、`server/internal/modules/issue/adapter/postgres/gen/update.sql.go`、`server/internal/modules/project/adapter/postgres/gen/issue_ports.sql.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**（spec 2.5、第 3 节第 6、9、12 条；M4 设计 3.3、3.5、3.9、3.13、5.2；P1b spec 第 5 节的"`parent_id` 进 `IssuePatch`"）
- 契约：`IssueUpdate.parent_id`（可为 `null`：从父工作项下拿出）；`updateIssue` 的描述写上父工作项的规则（同一个项目、没归档、调用者看得到，不是它自己、不在它下面）和动态里的父工作项。
- `project`：`LockIssueProject` 多回答项目的标识（`issue_ports.sql`、`issue_ports.go`、`app/issue_ports.go` 的 `IssueProject.Identifier`）；`issue/app` 的 `LockedProject.Identifier`（动态按标识指名工作项，锁下读）。
- `issue/domain`：`IssuePatch.SetParentID`、`ParentID`；`Changes.Parent`（`ParentChange{To}`，与工作项的相同时不算改动：泛型的 `same`，日期和 `id` 共用，`dates.go` 的 `sameDay` 并入它）；`hierarchy.go`（整个文件）：`ParentChange`、`Ref{ID, Identifier}`、`ParentActivity`（第 5 行：`parent`、`updated`，原来和新的父工作项的标识，`""` 是没有，它们的 `id` 是 `old_identifier`、`new_identifier`）、`refText`；`UpdateActivity` 带上它；`identifier.go` 的 `Identifier(project, number)`（`ParseIdentifier` 的反面）。
- `issue/adapter/postgres`：`queries/parents.sql`（新）的 `Ancestry`（`WITH RECURSIVE … UNION`：它和它上面的每个工作项，已删除、已归档的也在，各一次；数据库里有环也会结束）、`parents.go`（新）；`update.sql`、`update.go` 的 `UpdateIssue` 写 `parent_id`。
- `issue/app`：`IssueUpdater.Ancestry`；`parents.go` 的 `parentChange(ctx, issues, h, actor, to)`（新的父工作项经 `parentOf`，再读它的祖先链，含工作项本身的不行（`domain.Loops` 在 Task 5 之前是这里的 `slices.Contains`）；原来的父工作项在锁下读，不在是错误：删除工作项连带它下面的）、`refOf(h, i)`（锁下的项目的标识和工作项的编号）；`update_issue.go` 的 `check` 在日期之后查父工作项，问题与别的一次报出，写 `IssueFields.ParentID`，动态写第 5 行。
- 测试的帮手：`fakes_parents_test.go`（新）：`fakeIssues.Ancestry`；`fakes_test.go` 的锁下的项目带标识 `WEB`（`create_issue_test.go` 的世界照改）。
- HTTP：`UpdateIssue` 映射 `parent_id`（`null` 是拿出）。

**Tests:**
- 领域：`TestChangesOf`、`TestChangesNone`（父工作项的改动，相同的不算）、`TestUpdateActivity`（第 5 行的各种情形）、`TestParseIdentifier`（`Identifier` 与它互为反面）。
- 单元：`TestUpdateIssueMovesTheWorkItem`（从 up 下移到 top 下、从没有到 top 下、从 up 下拿出，top 在 mine 下：新的父工作项有父工作项照样可以；日期之后，锁下读新的父工作项、它的祖先链、原来的父工作项；时刻；只改父工作项；一行第 5 行的动态，按标识和 `id`，`WEB-3`、`WEB-12`）、`TestUpdateIssueRefusesAParentItMayNotHave`（没有的、别的项目的、已归档的、它自己、它下面的、锁下的判定说只看自己的调用者的别人的：都在别的问题之后、同一个 422，什么都不写；给出与它现在相同的父工作项什么都不改、不读）、`TestAParentsFailingReadEndsTheWrite`（新的父工作项、它的祖先链、原来的父工作项的读失败；原来的父工作项不在是错误，在时刻之前）。
- 存储：`TestAWorkItemsAncestry`（c 在 b 下、b 在已归档的 a 下、a 在已删除的 root 下；数据库里造的环 x、y 各一次；`id` 的次序不是线的次序）、`TestAWorkItemsParentIsWritten`（`UpdateIssue` 写下 top，再写下没有）；`project` 的 `TestLockIssueProject`（回答标识）。
- HTTP：`TestUpdateIssue`、`TestUpdateIssueHoldsTheBodyToItsStructure`（`parent_id` 的值、`null`）。
- 组合：`TestIssueProjectsConvertsProjectsAnswers`（标识）；`TestAWorkItemIsPutUnderAnother`（在不存在的工作项下、带不存在的标签建的，两个问题一次报出；bob 把 Part 移到 Top 下、再拿出，各一行他的动态，`WEB-2` 到 `WEB-1`、`WEB-1` 到 `""`，再从没有父工作项拿出一次，什么都不改、不写；Part 不能要的父工作项各 422 `parent_id`、什么都不写：它自己，它下面的 Low（不论多深，中间的 Mid 已归档：祖先链穿过已归档的工作项），Ops 的，已归档的，已删除的，没有的；bob 把 Up 移到 Top 下，再把 Low 移到 Up 下：父工作项有父工作项照样可以，Up 存在 Top 下，Low 从已归档的 Mid 下存到 Up 下，一行他的第 5 行，`WEB-4` 到 `WEB-2`）、`TestTheGuestRuleHoldsForAParent`（walt 把他的工作项移到 alice 的 Up 下 422，移到他自己的下照常）、`TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection`（一个连接的池：alice 把 Part 从第一个工作项下移到第二个下，锁下读新的父工作项、它的祖先链、原来的父工作项的语句都在它的事务的连接上）；交错 9 `TestTwoMovesThatWouldMakeALoopSerialize`（新文件 `issue_interleaving_hierarchy_test.go`：alice 把 A 移到 B 下、bob 把 B 移到 A 下，两种顺序；第二个等 Web 的行，锁下读到祖先链里有它自己，422 `parent_id`，什么都不写；只有第一个写了第 5 行）。

- [ ] **Step 1: 契约和查询，生成**

`api/modules/issue.yaml`（修改，2 处）：

````old api/modules/issue.yaml
        start_date when the start date alone is given, else at target_date
        (validation_failed, every problem at once). A change that changes
        nothing writes nothing, and the work item is answered as it is.
        Otherwise the work item is changed last by the caller; moved into a
        state of the completed group, it is completed at the change, and
        moved into another, not completed; the accounts newly assigned
        subscribe to it; its activity says what changed: the name, the
        priority, the state, the start and the target dates, and each label
        and each assignee added and removed. Every row it writes is the
        caller's, at one moment. A project with too many writes on its work
        items at once answers server_busy, with Retry-After: try again then.
````
````new api/modules/issue.yaml
        start_date when the start date alone is given, else at target_date;
        the parent, when it changes to a work item, is one of the project's
        that is not archived and that the caller sees, and neither the work
        item nor one under it (validation_failed, every problem at once). A
        change that changes nothing writes nothing, and the work item is
        answered as it is. Otherwise the work item is changed last by the
        caller; moved into a state of the completed group, it is completed
        at the change, and moved into another, not completed; the accounts
        newly assigned subscribe to it; its activity says what changed: the
        name, the parent, the priority, the state, the start and the target
        dates, and each label and each assignee added and removed. Every row
        it writes is the caller's, at one moment. A project with too many
        writes on its work items at once answers server_busy, with
        Retry-After: try again then.
````
````old api/modules/issue.yaml
            start date the work item then has.
          type: [string, 'null']
          format: date

````
````new api/modules/issue.yaml
            start date the work item then has.
          type: [string, 'null']
          format: date
        parent_id:
          description: >-
            The work item it is under, null for none: one of the project's
            that is not archived, and neither this work item nor one under
            it.
          type: [string, 'null']
          format: uuid

````

`server/internal/modules/issue/adapter/postgres/queries/parents.sql`（新文件，11 行）：

````file server/internal/modules/issue/adapter/postgres/queries/parents.sql
-- name: Ancestry :many
-- A work item's ancestry (M4 design 3.9): the work item and those above it, its parent, its parent's parent and so on,
-- deleted and archived ones too, in no order; a work item in it may not be put under it. Read under the project's FOR
-- NO KEY UPDATE, which every write of a parent holds (3.5), so it stays as read until the write commits. UNION reads
-- each work item once, so a line that loops, which no write makes, ends. The primary key serves it.
WITH RECURSIVE line (id, parent_id) AS (
    SELECT i.id, i.parent_id FROM issues i WHERE i.id = sqlc.arg(id)
    UNION
    SELECT p.id, p.parent_id FROM issues p JOIN line l ON p.id = l.parent_id
)
SELECT id FROM line;
````

`server/internal/modules/issue/adapter/postgres/queries/update.sql`（修改，2 处）：

````old server/internal/modules/issue/adapter/postgres/queries/update.sql
-- updateIssue's work item (M4 design 3.7), under the project's FOR NO KEY UPDATE, read again under it: its name, state,
-- priority, dates and completion as the use case gives them, those it changes and the others as they are, last changed
-- at the moment and by the account given. Every write of a work item's row holds its project's lock first (3.5), so the
-- row is as the use case read it; it is not locked by id first (M3 design 3.6, convention 5).
````
````new server/internal/modules/issue/adapter/postgres/queries/update.sql
-- updateIssue's work item (M4 design 3.7, 3.9), under the project's FOR NO KEY UPDATE, read again under it: its name,
-- state, priority, dates, parent and completion as the use case gives them, those it changes and the others as they
-- are, last changed at the moment and by the account given. Every write of a work item's row holds its project's lock
-- first (3.5), so the row is as the use case read it; it is not locked by id first (M3 design 3.6, convention 5).
````
````old server/internal/modules/issue/adapter/postgres/queries/update.sql
    target_date = sqlc.narg(target_date), completed_at = sqlc.narg(completed_at), updated_by_id = sqlc.arg(by)::uuid,
    updated_at = sqlc.arg(at)::timestamptz
````
````new server/internal/modules/issue/adapter/postgres/queries/update.sql
    target_date = sqlc.narg(target_date), parent_id = sqlc.narg(parent_id), completed_at = sqlc.narg(completed_at),
    updated_by_id = sqlc.arg(by)::uuid, updated_at = sqlc.arg(at)::timestamptz
````

`server/internal/modules/project/adapter/postgres/queries/issue_ports.sql`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/queries/issue_ports.sql
-- reads no row.
SELECT workspace_id, (archived_at IS NOT NULL)::boolean AS archived, default_assignee_id
````
````new server/internal/modules/project/adapter/postgres/queries/issue_ports.sql
-- reads no row. The identifier names the project's work items in their activity (3.13), as it is under the lock.
SELECT workspace_id, (archived_at IS NOT NULL)::boolean AS archived, default_assignee_id, identifier
````

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `08133bd644463787d6118c58ae66f3f83a47f0125fa36c06a8ac991708ce160f` | 3344 | `api/dist/openapi.yaml` |
| `e9a7a36033a04f2413481e60f0b32437db9ee7338ff6bdb271552eb7812d363c` | 40 | `server/internal/modules/issue/adapter/http/gen/bodyshape.gen.go` |
| `8b5ad54721fdb8678c3e4e3793e56480d298f06f9c71e633415763c97d09ff2a` | 1118 | `server/internal/modules/issue/adapter/http/gen/server.gen.go` |
| `651cd6d01dd879b76de71c660d021e1898ce7640840aadea80280fbae5823e80` | 45 | `server/internal/modules/issue/adapter/postgres/gen/parents.sql.go` |
| `909672b194d07a2859918137bab91b46f363580e104a18562f311a8c13870bee` | 131 | `server/internal/modules/issue/adapter/postgres/gen/update.sql.go` |
| `b3271431c25ab3e9d43f0d4c7e0c13abef6801367d2b01075c7e3952732e63ce` | 213 | `server/internal/modules/project/adapter/postgres/gen/issue_ports.sql.go` |
| `552fd4cb34c7d24b8f1ed9d80c3ccec0b158bc69edbabb5ffc08f0c9e7e101c9` | 3706 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/issue/adapter/http/gen/bodyshape.gen.go server/internal/modules/issue/adapter/http/gen/server.gen.go server/internal/modules/issue/adapter/postgres/gen/parents.sql.go server/internal/modules/issue/adapter/postgres/gen/update.sql.go server/internal/modules/project/adapter/postgres/gen/issue_ports.sql.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 2: `project` 回答标识**

`server/internal/modules/project/adapter/postgres/issue_ports.go`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/issue_ports.go
	return app.IssueProject{WorkspaceID: r.WorkspaceID, Archived: r.Archived, DefaultAssigneeID: r.DefaultAssigneeID}, true, nil
````
````new server/internal/modules/project/adapter/postgres/issue_ports.go
	return app.IssueProject{WorkspaceID: r.WorkspaceID, Archived: r.Archived, DefaultAssigneeID: r.DefaultAssigneeID, Identifier: r.Identifier},
		true, nil
````

`server/internal/modules/project/adapter/postgres/issue_ports_test.go`（修改，2 处）：

````old server/internal/modules/project/adapter/postgres/issue_ports_test.go
// The issue module's lock of a project (M4 design 3.5, 6.5). It answers
// the undeleted project's workspace, whether it is archived, and its
// default assignee, an archived one too, none for a deleted one. Until the
// transaction ends the project is held FOR NO KEY UPDATE, which a FOR SHARE
// waits for and a foreign key's FOR KEY SHARE does not, and its
````
````new server/internal/modules/project/adapter/postgres/issue_ports_test.go
// The issue module's lock of a project (M4 design 3.5, 6.5). It answers the
// undeleted project's workspace, whether it is archived, its default
// assignee and its identifier, an archived one too, none for a deleted one.
// Until the transaction ends the project is held FOR NO KEY UPDATE, which a
// FOR SHARE waits for and a foreign key's FOR KEY SHARE does not, and its
````
````old server/internal/modules/project/adapter/postgres/issue_ports_test.go
		"Web":  {web, app.IssueProject{WorkspaceID: acme, DefaultAssigneeID: &alice}, true},
		"Arch": {arch, app.IssueProject{WorkspaceID: acme, Archived: true}, true},
````
````new server/internal/modules/project/adapter/postgres/issue_ports_test.go
		"Web":  {web, app.IssueProject{WorkspaceID: acme, DefaultAssigneeID: &alice, Identifier: "WEB"}, true},
		"Arch": {arch, app.IssueProject{WorkspaceID: acme, Archived: true, Identifier: "ARCH"}, true},
````

`server/internal/modules/project/app/issue_ports.go`（修改，2 处）：

````old server/internal/modules/project/app/issue_ports.go
// whether it is archived, and its default assignee, nil for none.
// bootstrap converts it into issue's value.
````
````new server/internal/modules/project/app/issue_ports.go
// whether it is archived, its default assignee, nil for none, and its
// identifier. bootstrap converts it into issue's value.
````
````old server/internal/modules/project/app/issue_ports.go
	DefaultAssigneeID *uuid.UUID
````
````new server/internal/modules/project/app/issue_ports.go
	DefaultAssigneeID *uuid.UUID
	Identifier        string
````

- [ ] **Step 3: 领域**

`server/internal/modules/issue/domain/activity.go`（修改，4 处）：

````old server/internal/modules/issue/domain/activity.go
// it (M4 design 3.13): its name, its priority, its state and its dates,
// each nil when it is left as it is; the labels and the accounts it adds
// and removes, each in id order.
````
````new server/internal/modules/issue/domain/activity.go
// it (M4 design 3.13): its name, its parent, its priority, its state and
// its dates, each nil when it is left as it is; the labels and the
// accounts it adds and removes, each in id order.
````
````old server/internal/modules/issue/domain/activity.go
	Name             *Changed[string]
````
````new server/internal/modules/issue/domain/activity.go
	Name             *Changed[string]
	Parent           *Changed[*Ref]
````
````old server/internal/modules/issue/domain/activity.go
// 3.13's table: row 3, the name; row 6, the priority; row 7, the state, by
// its names and its ids; rows 8 and 9, the start and the target dates, as
// YYYY-MM-DD or "" for none; then each label added (row 10) and removed
// (row 11), by its name and its id; then each account assigned (row 12)
// and unassigned (row 13), by his display name and his id; each list in
// the order given. A field left as it is writes no row (Plane the same).
````
````new server/internal/modules/issue/domain/activity.go
// 3.13's table: row 3, the name; row 5, the parent (ParentActivity); row 6,
// the priority; row 7, the state, by its names and its ids; rows 8 and 9,
// the start and the target dates, as YYYY-MM-DD or "" for none; then each
// label added (row 10) and removed (row 11), by its name and its id; then
// each account assigned (row 12) and unassigned (row 13), by his display
// name and his id; each list in the order given. A field left as it is
// writes no row (Plane the same).
````
````old server/internal/modules/issue/domain/activity.go
			Comment: "updated the name to"})
````
````new server/internal/modules/issue/domain/activity.go
			Comment: "updated the name to"})
	}
	if u.Parent != nil {
		rows = append(rows, ParentActivity(*u.Parent))
````

`server/internal/modules/issue/domain/activity_test.go`（修改，4 处）：

````old server/internal/modules/issue/domain/activity_test.go
// priority (row 6), the state by its names and ids (row 7), the start and
// the target dates as YYYY-MM-DD, "" for none (rows 8, 9), each label added
// and removed by its name and id (rows 10, 11), each account assigned and
// unassigned by his display name and id (rows 12, 13), each list in the
// order given; Plane's sentences go with them, the start date's with its
// space. A field left as it is writes none.
````
````new server/internal/modules/issue/domain/activity_test.go
// parent by the identifiers and ids of the old and the new one, "" and
// none for none (row 5), the priority (row 6), the state by its names and
// ids (row 7), the start and the target dates as YYYY-MM-DD, "" for none
// (rows 8, 9), each label added and removed by its name and id (rows 10,
// 11), each account assigned and unassigned by his display name and id
// (rows 12, 13), each list in the order given; Plane's sentences go with
// them, the start date's with its space, the parent's of a work item. A
// field left as it is writes none.
````
````old server/internal/modules/issue/domain/activity_test.go
	todo, done, bug, ui, ann, ben := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
````
````new server/internal/modules/issue/domain/activity_test.go
	todo, done, bug, ui, ann, ben := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	up, top := uuid.NewV7(), uuid.NewV7()
````
````old server/internal/modules/issue/domain/activity_test.go
			Comment: comment}
````
````new server/internal/modules/issue/domain/activity_test.go
			Comment: comment}
	}
	parented := func(oldValue, newValue string, oldID, newID *uuid.UUID) Activity {
		return row("parent", oldValue, newValue, oldID, newID, "updated the parent work item to")
````
````old server/internal/modules/issue/domain/activity_test.go
			Priority: &Changed[Priority]{Old: PriorityHigh, New: PriorityNone}, Name: &Changed[string]{Old: "Fix", New: "Mend"},
		}, []Activity{name, priority, state, start, target, labelled(bug, "Bug"), unlabelled(ui, "UI"), assigned(ann, "Ann L."),
			unassigned(ben, "ben")}},
````
````new server/internal/modules/issue/domain/activity_test.go
			Priority: &Changed[Priority]{Old: PriorityHigh, New: PriorityNone},
			Parent:   &Changed[*Ref]{Old: &Ref{ID: up, Identifier: "WEB-3"}, New: &Ref{ID: top, Identifier: "WEB-12"}},
			Name:     &Changed[string]{Old: "Fix", New: "Mend"},
		}, []Activity{name, parented("WEB-3", "WEB-12", &up, &top), priority, state, start, target, labelled(bug, "Bug"),
			unlabelled(ui, "UI"), assigned(ann, "Ann L."), unassigned(ben, "ben")}},
		{"a parent set", Update{Parent: &Changed[*Ref]{New: &Ref{ID: top, Identifier: "WEB-12"}}},
			[]Activity{parented("", "WEB-12", nil, &top)}},
		{"the parent taken away", Update{Parent: &Changed[*Ref]{Old: &Ref{ID: up, Identifier: "WEB-3"}}},
			[]Activity{parented("WEB-3", "", &up, nil)}},
````

`server/internal/modules/issue/domain/dates.go`（修改，1 处）：

````old server/internal/modules/issue/domain/dates.go

// sameDay reports whether a and b, either nil for none, are the same day.
func sameDay(a, b *shared.Date) bool {
	return a == nil && b == nil || a != nil && b != nil && *a == *b
}

````
````new server/internal/modules/issue/domain/dates.go

````

`server/internal/modules/issue/domain/hierarchy.go`（整个文件，58 行）：

````whole server/internal/modules/issue/domain/hierarchy.go
package domain

import (
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// A work item's parent (M4 design 3.9): an undeleted work item of its
// project that it is under, however deep, or none. A work item is never
// under itself, nor under a work item under it, so the work items of a
// project make trees; an archived work item takes no new work item under
// it (3.8), and a guest who sees only his own work items puts none under
// another's (3.4).

// ParentChange is a parent a change sets: To, the new parent's id, nil
// when it takes the work item out from under its parent.
type ParentChange struct {
	To *uuid.UUID
}

// ParentNotAllowed is the problem of a parent asked for that may not be
// the work item's (M4 design 3.9).
func ParentNotAllowed() shared.FieldError {
	return shared.FieldError{Field: "parent_id", Code: shared.FieldNotAllowed,
		Message: "must be a work item of the project that is not archived, and neither this one nor under it"}
}

// Ref is a work item as an activity names it (M4 design 3.13): its id, and
// its identifier when the row is written.
type Ref struct {
	ID         uuid.UUID
	Identifier string
}

// ParentActivity is the row of a work item put under another parent, or
// under none (row 5 of M4 design 3.13): parent, updated, from the old
// parent's identifier to the new one's, "" for none, their ids the old and
// the new identifier, nil for none. An update writes it (UpdateActivity).
func ParentActivity(c Changed[*Ref]) Activity {
	a := Activity{Field: ptr("parent"), Verb: VerbUpdated, OldValue: ptr(refText(c.Old)), NewValue: ptr(refText(c.New)),
		Comment: "updated the parent work item to"}
	if c.Old != nil {
		a.OldIdentifier = &c.Old.ID
	}
	if c.New != nil {
		a.NewIdentifier = &c.New.ID
	}
	return a
}

// refText is r as an activity's value: its identifier, "" for none.
func refText(r *Ref) string {
	if r == nil {
		return ""
	}
	return r.Identifier
}
````

`server/internal/modules/issue/domain/identifier.go`（修改，1 处）：

````old server/internal/modules/issue/domain/identifier.go
// letters too, finds its project.
````
````new server/internal/modules/issue/domain/identifier.go
// letters too, finds its project.

// Identifier is the identifier of the work item of the number in the
// project whose identifier is project: WEB-12, as ParseIdentifier reads it
// and as an activity names a work item (M4 design 3.13).
func Identifier(project string, number int) string {
	return project + "-" + strconv.Itoa(number)
}
````

`server/internal/modules/issue/domain/identifier_test.go`（修改，2 处）：

````old server/internal/modules/issue/domain/identifier_test.go
// have. Anything else is no identifier.
````
````new server/internal/modules/issue/domain/identifier_test.go
// have. Anything else is no identifier. Identifier writes each identifier
// it reads as it was.
````
````old server/internal/modules/issue/domain/identifier_test.go
			t.Errorf("ParseIdentifier(%q) = %q, %d, %v; want %q, %d, %v", tt.s, project, number, ok, tt.project, tt.number, tt.ok)
		}
````
````new server/internal/modules/issue/domain/identifier_test.go
			t.Errorf("ParseIdentifier(%q) = %q, %d, %v; want %q, %d, %v", tt.s, project, number, ok, tt.project, tt.number, tt.ok)
		}
		if ok && Identifier(project, number) != tt.s {
			t.Errorf("Identifier(%q, %d) = %q, want %q", project, number, Identifier(project, number), tt.s)
		}
````

`server/internal/modules/issue/domain/patch.go`（修改，8 处）：

````old server/internal/modules/issue/domain/patch.go
// none. A date may be cleared: SetStartDate and SetTargetDate tell whether
// the date is given, StartDate and TargetDate its day, nil for none.
````
````new server/internal/modules/issue/domain/patch.go
// none. A date and the parent may be cleared: SetStartDate, SetTargetDate
// and SetParentID tell whether the field is given, StartDate, TargetDate
// and ParentID its value, nil for none.
````
````old server/internal/modules/issue/domain/patch.go
	TargetDate    *shared.Date
````
````new server/internal/modules/issue/domain/patch.go
	TargetDate    *shared.Date
	SetParentID   bool
	ParentID      *uuid.UUID
````
````old server/internal/modules/issue/domain/patch.go
// change adds may be is the use case's, under the project's lock, and so
// is the order of the dates the change leaves (DatesOutOfOrder): they are
// the work item's, merged with those given.
````
````new server/internal/modules/issue/domain/patch.go
// change adds, and the parent it sets, may be is the use case's, under the
// project's lock, and so is the order of the dates the change leaves
// (DatesOutOfOrder): they are the work item's, merged with those given.
````
````old server/internal/modules/issue/domain/patch.go
// Changes is what a checked patch changes in a work item (M4 design 3.7):
// the new name, priority, state and dates where each differs from the
// work item's, nil where it is not given or is the same; and the accounts
// and the labels it adds and removes (Diff), where the patch gives the
// new set. An account or a label the work item has and the new set keeps
// is left as it is: its row is not written again.
````
````new server/internal/modules/issue/domain/patch.go
// Changes is what a checked patch changes in a work item (M4 design 3.7,
// 3.9): the new name, priority, state, dates and parent where each differs
// from the work item's, nil where it is not given or is the same; and the
// accounts and the labels it adds and removes (Diff), where the patch
// gives the new set. An account or a label the work item has and the new
// set keeps is left as it is: its row is not written again.
````
````old server/internal/modules/issue/domain/patch.go
	TargetDate *DateChange
````
````new server/internal/modules/issue/domain/patch.go
	TargetDate *DateChange
	Parent     *ParentChange
````
````old server/internal/modules/issue/domain/patch.go
	if p.SetStartDate && !sameDay(p.StartDate, i.StartDate) {
````
````new server/internal/modules/issue/domain/patch.go
	if p.SetStartDate && !same(p.StartDate, i.StartDate) {
````
````old server/internal/modules/issue/domain/patch.go
	if p.SetTargetDate && !sameDay(p.TargetDate, i.TargetDate) {
		c.TargetDate = &DateChange{To: p.TargetDate}
````
````new server/internal/modules/issue/domain/patch.go
	if p.SetTargetDate && !same(p.TargetDate, i.TargetDate) {
		c.TargetDate = &DateChange{To: p.TargetDate}
	}
	if p.SetParentID && !same(p.ParentID, i.ParentID) {
		c.Parent = &ParentChange{To: p.ParentID}
````
````old server/internal/modules/issue/domain/patch.go
	return c.Name == nil && c.Priority == nil && c.StateID == nil && c.StartDate == nil && c.TargetDate == nil && len(c.Assignees.Added) == 0 &&
		len(c.Assignees.Removed) == 0 && len(c.Labels.Added) == 0 && len(c.Labels.Removed) == 0
}

````
````new server/internal/modules/issue/domain/patch.go
	return c.Name == nil && c.Priority == nil && c.StateID == nil && c.StartDate == nil && c.TargetDate == nil && c.Parent == nil &&
		len(c.Assignees.Added) == 0 && len(c.Assignees.Removed) == 0 && len(c.Labels.Added) == 0 && len(c.Labels.Removed) == 0
}

// same reports whether a and b, either nil for none, are both none or the
// same value: the same day, the same id.
func same[T comparable](a, b *T) bool {
	return a == nil && b == nil || a != nil && b != nil && *a == *b
}

````

`server/internal/modules/issue/domain/patch_test.go`（修改，5 处）：

````old server/internal/modules/issue/domain/patch_test.go
// with no target date, assigned to ann and ben, labelled Bug and UI (M4
// design 3.7): a field not given, or given as it is, changes nothing, a
// date the same day or none again too; a date given another day, or
// cleared, is the new one; a set given is the new set, whose difference
// with the work item's is added and removed, each in id order, whatever the
// order given; a set not given leaves the work item's as it is, its
// accounts and labels too.
````
````new server/internal/modules/issue/domain/patch_test.go
// with no target date, under up, assigned to ann and ben, labelled Bug and
// UI (M4 design 3.7, 3.9): a field not given, or given as it is, changes
// nothing, a date the same day or none again too, the parent the same
// work item, or none again; a date given another day, or cleared, is the
// new one, and so is a parent given another work item, or none; a set
// given is the new set, whose difference with the work item's is added
// and removed, each in id order, whatever the order given; a set not given
// leaves the work item's as it is, its accounts and labels too.
````
````old server/internal/modules/issue/domain/patch_test.go
	todo, done := uuid.NewV7(), uuid.NewV7()
	i := Issue{Name: "Fix", StateID: todo, Priority: PriorityHigh, StartDate: october(12), AssigneeIDs: []uuid.UUID{ann, ben},
		LabelIDs: []uuid.UUID{bug, ui}}
````
````new server/internal/modules/issue/domain/patch_test.go
	todo, done, up, top := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	i := Issue{Name: "Fix", StateID: todo, Priority: PriorityHigh, StartDate: october(12), ParentID: ptr(up),
		AssigneeIDs: []uuid.UUID{ann, ben}, LabelIDs: []uuid.UUID{bug, ui}}
````
````old server/internal/modules/issue/domain/patch_test.go
			StartDate: october(12), SetTargetDate: true, AssigneeIDs: set(ben, ann), LabelIDs: set(ui, bug)}, Changes{}},
````
````new server/internal/modules/issue/domain/patch_test.go
			StartDate: october(12), SetTargetDate: true, SetParentID: true, ParentID: ptr(up), AssigneeIDs: set(ben, ann),
			LabelIDs: set(ui, bug)}, Changes{}},
		{"another parent", IssuePatch{SetParentID: true, ParentID: &top}, Changes{Parent: &ParentChange{To: &top}}},
		{"the parent taken away", IssuePatch{SetParentID: true}, Changes{Parent: &ParentChange{}}},
````
````old server/internal/modules/issue/domain/patch_test.go
			t.Errorf("%s: ChangesOf() = %+v, want %+v", tt.name, got, tt.want)
		}
````
````new server/internal/modules/issue/domain/patch_test.go
			t.Errorf("%s: ChangesOf() = %+v, want %+v", tt.name, got, tt.want)
		}
	}
	orphan := Issue{Name: "Fix", StateID: todo}
	if got := ChangesOf(orphan, IssuePatch{SetParentID: true}); !reflect.DeepEqual(got, Changes{}) {
		t.Errorf("no parent taken away from a work item under none: ChangesOf() = %+v, want nothing", got)
	}
	if got, want := ChangesOf(orphan, IssuePatch{SetParentID: true, ParentID: &up}), (Changes{Parent: &ParentChange{To: &up}}); !reflect.DeepEqual(got,
		want) {
		t.Errorf("a parent for a work item under none: ChangesOf() = %+v, want %+v", got, want)
````
````old server/internal/modules/issue/domain/patch_test.go
		"the start date cleared": {StartDate: &DateChange{}}, "a target date": {TargetDate: &DateChange{To: october(19)}},
````
````new server/internal/modules/issue/domain/patch_test.go
		"the start date cleared": {StartDate: &DateChange{}}, "a target date": {TargetDate: &DateChange{To: october(19)}},
		"the parent taken away": {Parent: &ParentChange{}}, "a parent": {Parent: &ParentChange{To: &id}},
````

- [ ] **Step 4: 用例**

`server/internal/modules/issue/app/create_issue_test.go`（修改，2 处）：

````old server/internal/modules/issue/app/create_issue_test.go
// world is web as the use cases' ports see it: its states, the greatest
// sort order of Done's work items 85535, its labels, its last number 6,
// its members' roles and display names; an Authorizer that allows each
// call it is not told to refuse; and the locks over them.
````
````new server/internal/modules/issue/app/create_issue_test.go
// world is web as the use cases' ports see it: its identifier WEB, its
// states, the greatest sort order of Done's work items 85535, its labels,
// its last number 6, its members' roles and display names; an Authorizer
// that allows each call it is not told to refuse; and the locks over them.
````
````old server/internal/modules/issue/app/create_issue_test.go
	w.project = &fakeProject{log: log, id: web, workspace: acme, found: true, locked: &app.LockedProject{WorkspaceID: acme}, last: 6,
		states: []app.StateRef{backlog, done, triage}, labels: []app.LabelRef{{ID: bug, Name: "Bug"}, {ID: ui, Name: "UI"}},
````
````new server/internal/modules/issue/app/create_issue_test.go
	w.project = &fakeProject{log: log, id: web, workspace: acme, found: true, locked: &app.LockedProject{WorkspaceID: acme, Identifier: "WEB"},
		last: 6, states: []app.StateRef{backlog, done, triage}, labels: []app.LabelRef{{ID: bug, Name: "Bug"}, {ID: ui, Name: "UI"}},
````

`server/internal/modules/issue/app/fakes_parents_test.go`（新文件，18 行）：

````file server/internal/modules/issue/app/fakes_parents_test.go
package app_test

import (
	"context"
	"uuid"
)

// Ancestry logs the read and answers the work item id's ancestry as the
// fake is told it, or the work item alone (fakeIssues' ancestry).
func (f *fakeIssues) Ancestry(ctx context.Context, id uuid.UUID) ([]uuid.UUID, error) {
	if err := f.log.add(ctx, "Ancestry", id); err != nil {
		return nil, err
	}
	if a, ok := f.ancestry[id]; ok {
		return a, nil
	}
	return []uuid.UUID{id}, nil
}
````

`server/internal/modules/issue/app/fakes_test.go`（修改，3 处）：

````old server/internal/modules/issue/app/fakes_test.go
// are the store's other work items, answered as they are.
````
````new server/internal/modules/issue/app/fakes_test.go
// are the store's other work items, answered as they are, and ancestry
// their ancestries (fakes_parents_test.go).
````
````old server/internal/modules/issue/app/fakes_test.go
	others     map[uuid.UUID]domain.Issue
````
````new server/internal/modules/issue/app/fakes_test.go
	others     map[uuid.UUID]domain.Issue
	ancestry   map[uuid.UUID][]uuid.UUID
````
````old server/internal/modules/issue/app/fakes_test.go
	f.issue.StartDate, f.issue.TargetDate = r.StartDate, r.TargetDate
````
````new server/internal/modules/issue/app/fakes_test.go
	f.issue.StartDate, f.issue.TargetDate, f.issue.ParentID = r.StartDate, r.TargetDate, r.ParentID
````

`server/internal/modules/issue/app/parents.go`（修改，3 处）：

````old server/internal/modules/issue/app/parents.go
	"context"
````
````new server/internal/modules/issue/app/parents.go
	"context"
	"fmt"
	"slices"
````
````old server/internal/modules/issue/app/parents.go
// Issue.HiddenFrom); ok is false for any other.
````
````new server/internal/modules/issue/app/parents.go
// Issue.HiddenFrom); ok is false for any other. Whether the work item
// would be under itself is its update's to check (Ancestry).
````
````old server/internal/modules/issue/app/parents.go
	return p, true, nil
}

````
````new server/internal/modules/issue/app/parents.go
	return p, true, nil
}

// parentChange checks the parent to, nil for none, that the caller actor's
// update sets on the work item h holds, under h, and names the change as
// the work item's activity writes it (row 5 of M4 design 3.13). A new
// parent may be one (parentOf) that is neither the work item nor a work
// item under it, the work item not in its ancestry; ok is false otherwise.
// The old parent, when there is one, is read under the lock: an undeleted
// work item of the project, since deleting a work item deletes those under
// it (3.6), so one not there is an error.
func parentChange(ctx context.Context, issues IssueUpdater, h held, actor uuid.UUID, to *uuid.UUID) (c domain.Changed[*domain.Ref], ok bool,
	err error) {
	i := h.issue
	if to != nil {
		parent, ok, err := parentOf(ctx, issues, h, actor, i.ProjectID, *to)
		if err != nil || !ok {
			return domain.Changed[*domain.Ref]{}, false, err
		}
		ancestry, err := issues.Ancestry(ctx, parent.ID)
		if err != nil || slices.Contains(ancestry, i.ID) {
			return domain.Changed[*domain.Ref]{}, false, err
		}
		c.New = refOf(h, parent)
	}
	if i.ParentID != nil {
		old, found, err := issues.GetIssue(ctx, *i.ParentID)
		switch {
		case err != nil:
			return domain.Changed[*domain.Ref]{}, false, err
		case !found:
			return domain.Changed[*domain.Ref]{}, false, fmt.Errorf("work item %s is under %s, which is not there", i.ID, *i.ParentID)
		}
		c.Old = refOf(h, old)
	}
	return c, true, nil
}

// refOf is the work item i, of the project h holds locked, as an activity
// names it: by its identifier under the lock (M4 design 3.13).
func refOf(h held, i domain.Issue) *domain.Ref {
	return &domain.Ref{ID: i.ID, Identifier: domain.Identifier(h.project.Identifier, i.SequenceID)}
}

````

`server/internal/modules/issue/app/parents_test.go`（修改，5 处）：

````old server/internal/modules/issue/app/parents_test.go
import (
````
````new server/internal/modules/issue/app/parents_test.go
import (
	"errors"
````
````old server/internal/modules/issue/app/parents_test.go
	"uuid"

````
````new server/internal/modules/issue/app/parents_test.go
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
````
````old server/internal/modules/issue/app/parents_test.go
}

// A failing read of a parent ends the write with its failure, as itself,
````
````new server/internal/modules/issue/app/parents_test.go
}

// An update that moves web's work item 7 from under up to under top, puts
// it under top from under none, or takes it from under up (M4 design 3.5,
// 3.9, 3.13), top being under mine, which is neither the work item nor
// under it: after the dates, the new parent read under the lock, then its
// ancestry, then the old parent; the clock; the work item's fields, its
// parent the new one, the others as they were; its activity a row of its
// parent alone, by the old and the new one's identifiers and ids, WEB-3
// and WEB-12, "" and none for none, alice's at the moment. The answer is
// the work item under its new parent.
func TestUpdateIssueMovesTheWorkItem(t *testing.T) {
	ref := func(i domain.Issue, identifier string) *domain.Ref {
		return &domain.Ref{ID: i.ID, Identifier: identifier}
	}
	for _, tt := range []struct {
		name     string
		from, to func(k kin) *uuid.UUID
		reads    func(k kin) []string
		change   func(k kin) domain.Changed[*domain.Ref]
	}{
		{"from up to top", func(k kin) *uuid.UUID { return &k.up.ID }, func(k kin) *uuid.UUID { return &k.top.ID },
			func(k kin) []string {
				return []string{line("GetIssue", k.top.ID), line("Ancestry", k.top.ID), line("GetIssue", k.up.ID)}
			},
			func(k kin) domain.Changed[*domain.Ref] {
				return domain.Changed[*domain.Ref]{Old: ref(k.up, "WEB-3"), New: ref(k.top, "WEB-12")}
			}},
		{"from none to top", func(kin) *uuid.UUID { return nil }, func(k kin) *uuid.UUID { return &k.top.ID },
			func(k kin) []string { return []string{line("GetIssue", k.top.ID), line("Ancestry", k.top.ID)} },
			func(k kin) domain.Changed[*domain.Ref] { return domain.Changed[*domain.Ref]{New: ref(k.top, "WEB-12")} }},
		{"from up to none", func(k kin) *uuid.UUID { return &k.up.ID }, func(kin) *uuid.UUID { return nil },
			func(k kin) []string { return []string{line("GetIssue", k.up.ID)} },
			func(k kin) domain.Changed[*domain.Ref] { return domain.Changed[*domain.Ref]{Old: ref(k.up, "WEB-3")} }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w, id := newUpdate()
			k := withKin(w)
			w.issues.issue.ParentID = tt.from(k)
			w.issues.ancestry = map[uuid.UUID][]uuid.UUID{k.top.ID: {k.top.ID, k.mine.ID}}
			got, err := uc.Execute(as(alice), id, domain.IssuePatch{SetParentID: true, ParentID: tt.to(k)})
			if err != nil || !reflect.DeepEqual(got.ParentID, tt.to(k)) {
				t.Fatalf("Execute() = %+v, %v; want the work item under %v", got, err, tt.to(k))
			}
			calls := slices.Concat(locked(id), tt.reads(k), []string{"Now", line("UpdateIssue", id), line("InsertActivities", 1),
				line("GetIssue", id)})
			if !slices.Equal(w.log.calls, calls) {
				t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
			}
			if fields := (app.IssueFields{ID: id, Name: "Fix", StateID: backlog.ID, Priority: domain.PriorityHigh, ParentID: tt.to(k), By: alice,
				At: clockNow}); w.issues.changed == nil || !reflect.DeepEqual(*w.issues.changed, fields) {
				t.Errorf("the fields written %+v, want %+v", w.issues.changed, fields)
			}
			change := tt.change(k)
			wantActivity(t, w, id, []domain.Activity{domain.ParentActivity(change)})
		})
	}
}

// A parent an update may not set is refused at parent_id, after the other
// problems, in one 422, and nothing is written (M4 design 3.4, 3.9): a
// work item not there, another project's, an archived one, read under the
// lock; the work item itself and one under it, whose ancestries hold it;
// and another's for a caller who sees only his own work items by the
// decision under the lock, as alice, web's admin, may in a project where
// she is its guest, who updates her own work item. Under her own work
// item she may put it. A parent given as the work item has it changes
// nothing and reads none.
func TestUpdateIssueRefusesAParentItMayNotHave(t *testing.T) {
	outOfOrder := shared.FieldError{Field: "target_date", Code: "not_allowed", Message: "must not be before the start date"}
	read := func(parent uuid.UUID) []string { return []string{line("GetIssue", parent)} }
	climbed := func(parent uuid.UUID) []string { return []string{line("GetIssue", parent), line("Ancestry", parent)} }
	ownOnly := func(w *world) {
		w.issues.issue.By, w.auth.grants = alice, []shared.Grant{{OwnOnly: true}, {OwnOnly: true}, {OwnOnly: true}, {OwnOnly: true}}
	}
	for _, tt := range []struct {
		name  string
		set   func(w *world, k kin, id uuid.UUID) uuid.UUID // the parent asked for
		reads func(parent uuid.UUID) []string
	}{
		{"not there", func(*world, kin, uuid.UUID) uuid.UUID { return uuid.NewV7() }, read},
		{"another project's", func(_ *world, k kin, _ uuid.UUID) uuid.UUID { return k.far.ID }, read},
		{"archived", func(_ *world, k kin, _ uuid.UUID) uuid.UUID { return k.shelved.ID }, read},
		{"itself", func(_ *world, _ kin, id uuid.UUID) uuid.UUID { return id }, climbed},
		{"one under it", func(w *world, k kin, id uuid.UUID) uuid.UUID {
			w.issues.ancestry = map[uuid.UUID][]uuid.UUID{k.top.ID: {k.top.ID, id, k.up.ID}}
			return k.top.ID
		}, climbed},
		{"another's, for one who sees her own alone", func(w *world, k kin, _ uuid.UUID) uuid.UUID { ownOnly(w); return k.up.ID }, read},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w, id := newUpdate()
			k := withKin(w)
			parent := tt.set(w, k, id)
			_, err := uc.Execute(as(alice), id, domain.IssuePatch{SetStartDate: true, StartDate: october(20), SetTargetDate: true,
				TargetDate: october(19), SetParentID: true, ParentID: &parent})
			unchanged(t, w, err, shared.Invalid(outOfOrder, domain.ParentNotAllowed()), append(locked(id), tt.reads(parent)...))
		})
	}
	uc, w, id := newUpdate()
	k := withKin(w)
	ownOnly(w)
	if got, err := uc.Execute(as(alice), id, domain.IssuePatch{SetParentID: true, ParentID: &k.mine.ID}); err != nil ||
		!reflect.DeepEqual(got.ParentID, &k.mine.ID) {
		t.Errorf("Execute() under her own work item = %+v, %v; want it under %s", got, err, k.mine.ID)
	}
	uc, w, id = newUpdate()
	k = withKin(w)
	w.issues.issue.ParentID = &k.up.ID
	again := k.up.ID
	if got, err := uc.Execute(as(alice), id, domain.IssuePatch{SetParentID: true, ParentID: &again}); err != nil ||
		!reflect.DeepEqual(got.ParentID, &k.up.ID) || !slices.Equal(w.log.calls, locked(id)) || w.issues.changed != nil {
		t.Errorf("Execute() of the parent it has = %+v, %v after\n%q; want it as it was after\n%q, nothing written", got, err, w.log.calls,
			locked(id))
	}
}

// A failing read of a parent ends the write with its failure, as itself,
````
````old server/internal/modules/issue/app/parents_test.go
// for.
````
````new server/internal/modules/issue/app/parents_test.go
// for; the new parent an update asks for, its ancestry, the old one. An
// old parent not there is an error too, before the clock: deleting a work
// item deletes those under it (3.6).
````
````old server/internal/modules/issue/app/parents_test.go
	wantStoppedOn(t, w, err, w.log.fail)
````
````new server/internal/modules/issue/app/parents_test.go
	wantStoppedOn(t, w, err, w.log.fail)
	for _, at := range []string{"new", "ancestry", "old"} {
		t.Run(at, func(t *testing.T) {
			uc, w, id := newUpdate()
			k := withKin(w)
			w.issues.issue.ParentID = &k.up.ID
			w.log.fail = map[string]string{"new": line("GetIssue", k.top.ID), "ancestry": line("Ancestry", k.top.ID),
				"old": line("GetIssue", k.up.ID)}[at]
			_, err := uc.Execute(as(alice), id, domain.IssuePatch{SetParentID: true, ParentID: &k.top.ID})
			wantStoppedOn(t, w, err, w.log.fail)
		})
	}
	uc, w, id := newUpdate()
	k = withKin(w)
	gone := uuid.NewV7()
	w.issues.issue.ParentID = &gone
	var se *shared.Error
	if _, err := uc.Execute(as(alice), id, domain.IssuePatch{SetParentID: true, ParentID: &k.top.ID}); err == nil || errors.As(err, &se) ||
		slices.Contains(w.log.calls, "Now") {
		t.Errorf("Execute() of a work item under one not there = %v after %q; want an error before the clock", err, w.log.calls)
	}
````

`server/internal/modules/issue/app/ports.go`（修改，3 处）：

````old server/internal/modules/issue/app/ports.go
// workspace, whether it is archived, and its default assignee, nil for
// none.
````
````new server/internal/modules/issue/app/ports.go
// workspace, whether it is archived, its default assignee, nil for none,
// and its identifier, which names its work items in their activity (M4
// design 3.13).
````
````old server/internal/modules/issue/app/ports.go
	DefaultAssigneeID *uuid.UUID
````
````new server/internal/modules/issue/app/ports.go
	DefaultAssigneeID *uuid.UUID
	Identifier        string
````
````old server/internal/modules/issue/app/ports.go
type IssueUpdater interface {
	IssueReader
````
````new server/internal/modules/issue/app/ports.go
type IssueUpdater interface {
	IssueReader
	// Ancestry reads the work item and those above it, its parent, its
	// parent's parent and so on, each once, in no order: a work item that
	// is in it may not be put under it (M4 design 3.9).
	Ancestry(ctx context.Context, id uuid.UUID) ([]uuid.UUID, error)
````

`server/internal/modules/issue/app/rows.go`（修改，2 处）：

````old server/internal/modules/issue/app/rows.go
// 3.7): those it changes and the others as they are, and the moment and
// the account that change it last.
````
````new server/internal/modules/issue/app/rows.go
// 3.7, 3.9): those it changes and the others as they are, and the moment
// and the account that change it last.
````
````old server/internal/modules/issue/app/rows.go
	StartDate   *shared.Date
	TargetDate  *shared.Date
	CompletedAt *time.Time
````
````new server/internal/modules/issue/app/rows.go
	StartDate   *shared.Date
	TargetDate  *shared.Date
	ParentID    *uuid.UUID
	CompletedAt *time.Time
````

`server/internal/modules/issue/app/update_issue.go`（修改，7 处）：

````old server/internal/modules/issue/app/update_issue.go
// changes it, and the new one's group; the labels it adds and removes,
// each in id order.
````
````new server/internal/modules/issue/app/update_issue.go
// changes it, and the new one's group; its parent before and after, when
// the update changes it; the labels it adds and removes, each in id order.
````
````old server/internal/modules/issue/app/update_issue.go
	group         domain.StateGroup
````
````new server/internal/modules/issue/app/update_issue.go
	group         domain.StateGroup
	parent        *domain.Changed[*domain.Ref]
````
````old server/internal/modules/issue/app/update_issue.go
// the answer is the work item as read; the dependent rows it adds and the
// order of the dates it leaves (check), every problem in one 422; the
// clock; the work item's fields, its completion by its new state's group
// when its state changes (domain.CompletedAt); the accounts and the labels
// it removes and adds, in id order; the accounts it adds subscribed; its
// activity (domain.UpdateActivity), each account by his display name. All
// of it is the caller's at that moment. The answer is the work item as
// stored, read back in the transaction.
````
````new server/internal/modules/issue/app/update_issue.go
// the answer is the work item as read; the dependent rows it adds, the
// order of the dates it leaves and the parent it sets (check), every
// problem in one 422; the clock; the work item's fields, its completion by
// its new state's group when its state changes (domain.CompletedAt); the
// accounts and the labels it removes and adds, in id order; the accounts
// it adds subscribed; its activity (domain.UpdateActivity), each account
// by his display name, each parent by its identifier. All of it is the
// caller's at that moment. The answer is the work item as stored, read
// back in the transaction.
````
````old server/internal/modules/issue/app/update_issue.go
		n, err := u.check(ctx, h.issue, p, c)
````
````new server/internal/modules/issue/app/update_issue.go
		n, err := u.check(ctx, h, actor.UserID, p, c)
````
````old server/internal/modules/issue/app/update_issue.go
// check checks the dependent rows that c adds to the work item i under the
// project's lock (M4 design 3.6, 3.7), and reads the names its activity
// writes: the new state, when c changes it, is the project's and not its
// triage state, by a creation's rule (pickState); each account added may
// be assigned (domain.Unassignable), while one the work item keeps stays
// whatever his role; each label added is the project's. The labels the
// work item keeps are on it, so they are the project's undeleted labels:
// deleting a label takes it off every work item (DetachLabel). Then the
// dates p leaves the work item with are in order, merged with those it
// keeps (domain.DatesOutOfOrder). Every problem is reported at once, in
// one 422, each at its place in p.
func (u *UpdateIssue) check(ctx context.Context, i domain.Issue, p domain.IssuePatch, c domain.Changes) (named, error) {
````
````new server/internal/modules/issue/app/update_issue.go
// check checks the dependent rows that c adds to the work item that h, the
// caller actor's write, holds under the project's lock (M4 design 3.6,
// 3.7, 3.9), and reads the names its activity writes: the new state, when
// c changes it, is the project's and not its triage state, by a creation's
// rule (pickState); each account added may be assigned
// (domain.Unassignable), while one the work item keeps stays whatever his
// role; each label added is the project's. The labels the work item keeps
// are on it, so they are the project's undeleted labels: deleting a label
// takes it off every work item (DetachLabel). Then the dates p leaves the
// work item with are in order, merged with those it keeps
// (domain.DatesOutOfOrder); then the parent c sets may be the work item's
// (parentChange). Every problem is reported at once, in one 422, each at
// its place in p.
func (u *UpdateIssue) check(ctx context.Context, h held, actor uuid.UUID, p domain.IssuePatch, c domain.Changes) (named, error) {
	i := h.issue
````
````old server/internal/modules/issue/app/update_issue.go
		problems = append(problems, *f)
	}
````
````new server/internal/modules/issue/app/update_issue.go
		problems = append(problems, *f)
	}
	if c.Parent != nil {
		parent, ok, err := parentChange(ctx, u.d.Issues, h, actor, c.Parent.To)
		if err != nil {
			return named{}, err
		}
		if ok {
			n.parent = &parent
		} else {
			problems = append(problems, domain.ParentNotAllowed())
		}
	}
````
````old server/internal/modules/issue/app/update_issue.go
		CompletedAt: i.CompletedAt, By: s.by, At: s.at}
	activity := domain.Update{State: n.state, LabelsAdded: n.labelsAdded, LabelsRemoved: n.labelsRemoved}
````
````new server/internal/modules/issue/app/update_issue.go
		ParentID: i.ParentID, CompletedAt: i.CompletedAt, By: s.by, At: s.at}
	activity := domain.Update{Parent: n.parent, State: n.state, LabelsAdded: n.labelsAdded, LabelsRemoved: n.labelsRemoved}
	if c.Parent != nil {
		f.ParentID = c.Parent.To
	}
````

- [ ] **Step 5: 存储和 HTTP**

`server/internal/modules/issue/adapter/http/issues.go`（修改，2 处）：

````old server/internal/modules/issue/adapter/http/issues.go
// not given, a date given null given as none; a value given goes to the
// use case as it came, which checks it.
````
````new server/internal/modules/issue/adapter/http/issues.go
// not given, a date or the parent given null given as none; a value given
// goes to the use case as it came, which checks it.
````
````old server/internal/modules/issue/adapter/http/issues.go
	p.SetTargetDate, p.TargetDate = given(req.Body.TargetDate)
````
````new server/internal/modules/issue/adapter/http/issues.go
	p.SetTargetDate, p.TargetDate = given(req.Body.TargetDate)
	p.SetParentID, p.ParentID = given(req.Body.ParentID)
````

`server/internal/modules/issue/adapter/http/update_test.go`（修改，5 处）：

````old server/internal/modules/issue/adapter/http/update_test.go
// date given null given as none; the answer is 200 with the work item the
// use case answers, as POST shows it.
````
````new server/internal/modules/issue/adapter/http/update_test.go
// date or the parent given null given as none; the answer is 200 with the
// work item the use case answers, as POST shows it.
````
````old server/internal/modules/issue/adapter/http/update_test.go
		{`{"start_date":null,"target_date":"2026-10-19"}`, domain.IssuePatch{SetStartDate: true, SetTargetDate: true, TargetDate: &targetDay}},
````
````new server/internal/modules/issue/adapter/http/update_test.go
		{`{"start_date":null,"target_date":"2026-10-19"}`, domain.IssuePatch{SetStartDate: true, SetTargetDate: true, TargetDate: &targetDay}},
		{`{"parent_id":"0199a2b4-0000-7000-8000-0000000000e6"}`, domain.IssuePatch{SetParentID: true, ParentID: &webSixID}},
		{`{"parent_id":null}`, domain.IssuePatch{SetParentID: true}},
````
````old server/internal/modules/issue/adapter/http/update_test.go
// a sort order, a parent, a description, which IssueUpdate has not in P2:
// M4 design 12, constraint 5), null for a field that is not nullable, a
// field of another type, an id that is no id, or a date that is no day or
// has a time: refused as bad_request before the use case.
````
````new server/internal/modules/issue/adapter/http/update_test.go
// a sort order, a description, which IssueUpdate has not in P3: M4 design
// 12, constraint 5), null for a field that is not nullable, a field of
// another type, an id that is no id, or a date that is no day or has a
// time: refused as bad_request before the use case.
````
````old server/internal/modules/issue/adapter/http/update_test.go
	for _, body := range []string{`{"sequence_id":3}`, `{"sort_order":1}`, `{"parent_id":null}`, `{"description_html":"<p>x</p>"}`,
````
````new server/internal/modules/issue/adapter/http/update_test.go
	for _, body := range []string{`{"sequence_id":3}`, `{"sort_order":1}`, `{"description_html":"<p>x</p>"}`,
````
````old server/internal/modules/issue/adapter/http/update_test.go
		`{"label_ids":["Bug"]}`, `{"start_date":"2026-02-30"}`, `{"target_date":"2026-10-19T00:00:00Z"}`} {
````
````new server/internal/modules/issue/adapter/http/update_test.go
		`{"label_ids":["Bug"]}`, `{"parent_id":"WEB-6"}`, `{"start_date":"2026-02-30"}`, `{"target_date":"2026-10-19T00:00:00Z"}`} {
````

`server/internal/modules/issue/adapter/postgres/parents.go`（新文件，17 行）：

````file server/internal/modules/issue/adapter/postgres/parents.go
package postgresadapter

import (
	"context"
	"fmt"
	"uuid"
)

// Ancestry reads the work item id and those above it, each once, in no
// order (M4 design 3.9): its parent, its parent's parent and so on.
func (s *Store) Ancestry(ctx context.Context, id uuid.UUID) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).Ancestry(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("read the work items above work item %s: %w", id, err)
	}
	return ids, nil
}
````

`server/internal/modules/issue/adapter/postgres/parents_test.go`（修改，3 处）：

````old server/internal/modules/issue/adapter/postgres/parents_test.go
	"context"
	"reflect"
	"testing"
````
````new server/internal/modules/issue/adapter/postgres/parents_test.go
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"
````
````old server/internal/modules/issue/adapter/postgres/parents_test.go
)

// A work item's parent through the store (M4 design 3.9): InsertIssue
// writes the parent it is given, up, and GetIssue reads it. Every other
````
````new server/internal/modules/issue/adapter/postgres/parents_test.go
)

// A work item's ancestry (M4 design 3.9) is it and each work item above it,
// however high, each once, archived and deleted ones too: in web, c is
// under b, under a, archived, under root, deleted; d has none above it. In
// ops, x is under y and y under x, a loop no write makes: the read ends,
// each once. The work items are drawn so that no line's id order is its
// order. A work item not there has none.
func TestAWorkItemsAncestry(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, ops := newProject(t, pool, acme, "WEB"), newProject(t, pool, acme, "OPS")
	b, root, d, c, a, y, x := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	for n, id := range []uuid.UUID{root, a, b, c, d} {
		newIssue(t, pool, web, id, n+1, alice)
	}
	newIssue(t, pool, ops, x, 1, alice)
	newIssue(t, pool, ops, y, 2, alice)
	for _, link := range [][2]uuid.UUID{{a, root}, {b, a}, {c, b}, {x, y}, {y, x}} {
		exec(t, pool, "UPDATE issues SET parent_id = $2 WHERE id = $1", link[0], link[1])
	}
	exec(t, pool, "UPDATE issues SET archived_at = $2 WHERE id = $1", a, now)
	exec(t, pool, "UPDATE issues SET deleted_at = $2 WHERE id = $1", root, now)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, tt := range []struct {
		name string
		id   uuid.UUID
		want []uuid.UUID
	}{
		{"c", c, []uuid.UUID{c, b, a, root}},
		{"a, archived", a, []uuid.UUID{a, root}},
		{"root, deleted", root, []uuid.UUID{root}},
		{"d, with none above it", d, []uuid.UUID{d}},
		{"x, in a loop", x, []uuid.UUID{x, y}},
		{"no work item", uuid.NewV7(), nil},
	} {
		got, err := s.Ancestry(ctx, tt.id)
		if want := slices.SortedFunc(slices.Values(tt.want), uuid.UUID.Compare); err != nil ||
			!slices.Equal(slices.SortedFunc(slices.Values(got), uuid.UUID.Compare), want) {
			t.Errorf("Ancestry(%s) = %v, %v; want %v in any order", tt.name, got, err, want)
		}
	}
	cancelled, stop := context.WithCancel(context.Background())
	stop()
	if _, err := s.Ancestry(cancelled, c); !errors.Is(err, context.Canceled) {
		t.Errorf("Ancestry() on a cancelled context = %v; want context.Canceled", err)
	}
}

// A work item's parent through the store (M4 design 3.9): InsertIssue
// writes the parent it is given, up, and GetIssue reads it; UpdateIssue
// writes the one it is given, top, then none, NULL. Each time every other
````
````old server/internal/modules/issue/adapter/postgres/parents_test.go
	if got, want := parents(), map[uuid.UUID]*uuid.UUID{up: nil, top: nil, other: &up, id: &up}; !reflect.DeepEqual(got, want) {
		t.Errorf("the parents %v, want %v", got, want)
````
````new server/internal/modules/issue/adapter/postgres/parents_test.go
	for _, to := range []*uuid.UUID{&up, &top, nil} {
		if to != &up {
			if err := s.UpdateIssue(ctx, app.IssueFields{ID: id, Name: "Part", StateID: web.state, Priority: domain.PriorityNone, ParentID: to,
				By: alice, At: now}); err != nil {
				t.Fatal(err)
			}
		}
		if got, want := parents(), map[uuid.UUID]*uuid.UUID{up: nil, top: nil, other: &up, id: to}; !reflect.DeepEqual(got, want) {
			t.Errorf("the parents %v, want %v", got, want)
		}
````

`server/internal/modules/issue/adapter/postgres/update.go`（修改，2 处）：

````old server/internal/modules/issue/adapter/postgres/update.go
// UpdateIssue writes the work item's fields f (M4 design 3.7). The use
// case read it under the project's lock and checked every value, so any
// violation is a bug: an internal error (500), not a domain error.
````
````new server/internal/modules/issue/adapter/postgres/update.go
// UpdateIssue writes the work item's fields f (M4 design 3.7, 3.9). The
// use case read it under the project's lock and checked every value, so
// any violation is a bug: an internal error (500), not a domain error.
````
````old server/internal/modules/issue/adapter/postgres/update.go
		TargetDate: timeOf(f.TargetDate), CompletedAt: f.CompletedAt, By: f.By, At: f.At,
````
````new server/internal/modules/issue/adapter/postgres/update.go
		TargetDate: timeOf(f.TargetDate), ParentID: f.ParentID, CompletedAt: f.CompletedAt, By: f.By, At: f.At,
````

- [ ] **Step 6: 组合和交错 9**

`server/internal/bootstrap/issue_interleaving_hierarchy_test.go`（新文件，62 行）：

````file server/internal/bootstrap/issue_interleaving_hierarchy_test.go
package bootstrap

import (
	"net/http"
	"reflect"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// workItemNamed is the id of the world's one work item named name, of any
// project: the test fails unless there is exactly one.
func (w issueWorld) workItemNamed(t *testing.T, name string) uuid.UUID {
	t.Helper()
	ids := queryIDs(t, w.pool, "SELECT id FROM issues WHERE name = $1", name)
	if len(ids) != 1 {
		t.Fatalf("the work items named %s: %v; want one", name, ids)
	}
	return ids[0]
}

// movesUnder is name's change of Web's work item named child, which puts
// it under Web's work item named parent (workItemNamed).
func movesUnder(name, child, parent string) racingWrite {
	return func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
		return w.sending(e, newIssueRoute(t, w.pool, tx), name, http.MethodPatch, "/api/v0/issues/"+w.workItemNamed(t, child).String(),
			`{"parent_id":"`+w.workItemNamed(t, parent).String()+`"}`)
	}
}

// Two moves, each allowed alone, that together would make a loop
// serialize on Web's row (M4 design 3.5, 3.9; interleaving 9), in both
// orders: alice puts A under B while bob puts B under A. The first holds
// Web FOR NO KEY UPDATE past its writes; the second, which passed its
// precheck, waits for Web's row, and once the first commits reads the
// ancestry of its parent under its lock, finds its own work item in it,
// and is refused: 422 at parent_id, nothing written. The first's work item
// is under the other, which is under none, and only the first writes a row
// of a parent.
func TestTwoMovesThatWouldMakeALoopSerialize(t *testing.T) {
	prepare := func(t *testing.T, w issueWorld) {
		w.newWorkItemIn(t, w.web, `{"name":"A"}`)
		w.newWorkItemIn(t, w.web, `{"name":"B"}`)
	}
	inBothOrders(t, prepare, "A under B", movesUnder("alice", "A", "B"), "B under A", movesUnder("bob", "B", "A"), func(t *testing.T,
		w issueWorld, aFirst bool, ea, eb exchange) {
		first, second, under, over := ea, eb, w.workItemNamed(t, "A"), w.workItemNamed(t, "B")
		if !aFirst {
			first, second, under, over = eb, ea, over, under
		}
		w.answered(t, first, http.StatusOK)
		if answer := w.answered(t, second, http.StatusUnprocessableEntity); refusalOf(t, []byte(answer), true) != parentNotAllowed {
			t.Errorf("the second move answers %s, want %s", answer, parentNotAllowed)
		}
		if !reflect.DeepEqual(w.parentOf(t, under), &over) || w.parentOf(t, over) != nil || len(w.parentRows(t, under)) != 1 ||
			len(w.parentRows(t, over)) != 0 {
			t.Errorf("the first's work item is under %v, the other under %v, their rows of a parent %q and %q; want under %s, under none, "+
				"one row and none", w.parentOf(t, under), w.parentOf(t, over), w.parentRows(t, under), w.parentRows(t, over), over)
		}
	})
}
````

`server/internal/bootstrap/issue_parents_test.go`（修改，6 处）：

````old server/internal/bootstrap/issue_parents_test.go
	"reflect"
````
````new server/internal/bootstrap/issue_parents_test.go
	"reflect"
	"slices"
````
````old server/internal/bootstrap/issue_parents_test.go
// Up, and Part under Up: Part is answered and stored under Up, and its
// creation writes no row of its parent. A work item under one not there,
// with a label not there, is refused with both problems at once (422),
// and not made.
func TestAWorkItemIsPutUnderAnother(t *testing.T) {
	w := newIssueWorld(t)
````
````new server/internal/bootstrap/issue_parents_test.go
// Top, Up, and Part under Up, WEB-1 to WEB-3: Part is answered and stored
// under Up, and its creation writes no row of its parent. A work item
// under one not there, with a label not there, is refused with both
// problems at once (422), and not made. bob moves Part under Top, then
// from under it, then from under none again: each answered with the
// parent as stored; the first two each a row of his activity, from WEB-2
// to WEB-1, then from WEB-1 to "", their ids the old and the new parent's,
// the third, which changes nothing, none. Then each parent Part may not
// have is refused, 422 at parent_id, nothing written: Part itself; Low,
// under Mid under Part, however deep, Mid archived; Ops's work item; an
// archived work item; a deleted one; one not there. Then bob puts Up under
// Top, and Low under Up, a parent that has a parent: Up is stored under
// Top, and Low, from under archived Mid, under Up, with one row of his,
// from WEB-4 to WEB-2, its ids Mid's and Up's.
func TestAWorkItemIsPutUnderAnother(t *testing.T) {
	w := newIssueWorld(t)
	top := w.newWorkItemIn(t, w.web, `{"name":"Top"}`)
````
````old server/internal/bootstrap/issue_parents_test.go
			"parent_id not_allowed at once, and %d", got, w.workItems(t), count)
	}
````
````new server/internal/bootstrap/issue_parents_test.go
			"parent_id not_allowed at once, and %d", got, w.workItems(t), count)
	}
	path := "/api/v0/issues/" + part.ID.String()
	for _, step := range []struct {
		body string
		to   *uuid.UUID
	}{{`{"parent_id":"` + top.String() + `"}`, &top}, {`{"parent_id":null}`, nil}, {`{"parent_id":null}`, nil}} {
		var moved storedIssue
		decodeAnswer(t, w.send(t, "bob", http.MethodPatch, path, step.body, http.StatusOK), &moved)
		if !reflect.DeepEqual(moved.ParentID, step.to) || !reflect.DeepEqual(w.parentOf(t, part.ID), step.to) {
			t.Errorf("PATCH %s answers it under %v, stored under %v; want %v", step.body, moved.ParentID, w.parentOf(t, part.ID), step.to)
		}
	}
	if said, want := w.parentRows(t, part.ID), []string{"bob: WEB-2 → WEB-1, " + up.String() + " → " + top.String(),
		"bob: WEB-1 → , " + top.String() + " → none"}; !slices.Equal(said, want) {
		t.Errorf("Part's rows of its parent say %q, want %q", said, want)
	}

	mid := w.newWorkItemIn(t, w.web, `{"name":"Mid","state_id":"`+w.done.String()+`","parent_id":"`+part.ID.String()+`"}`)
	low := w.newWorkItemIn(t, w.web, underParent("Low", mid))
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+mid.String()+"/archive", "", http.StatusOK)
	far := w.newWorkItemIn(t, createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS"), `{"name":"Far"}`)
	shelved := w.doneWorkItem(t)
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+shelved.String()+"/archive", "", http.StatusOK)
	gone := w.newWorkItemIn(t, w.web, `{"name":"Gone"}`)
	w.send(t, "alice", http.MethodDelete, "/api/v0/issues/"+gone.String(), "", http.StatusNoContent)
	for name, parent := range map[string]uuid.UUID{"itself": part.ID, "under it": low, "Ops's": far, "archived": shelved, "deleted": gone,
		"not there": uuid.NewV7()} {
		_, changed, by := archivalOf(t, w, part.ID)
		answer := w.send(t, "bob", http.MethodPatch, path, `{"parent_id":"`+parent.String()+`"}`, http.StatusUnprocessableEntity)
		if got := refusalOf(t, []byte(answer), true); got != parentNotAllowed {
			t.Errorf("Part under %s answers %s, want %s", name, got, parentNotAllowed)
		}
		if _, at, who := archivalOf(t, w, part.ID); w.parentOf(t, part.ID) != nil || !at.Equal(changed) || who != by {
			t.Errorf("after Part under %s it is under %v, changed last by %s at %v; want under none, as before, by %s at %v", name,
				w.parentOf(t, part.ID), who, at, by, changed)
		}
	}
	w.send(t, "bob", http.MethodPatch, "/api/v0/issues/"+up.String(), `{"parent_id":"`+top.String()+`"}`, http.StatusOK)
	w.send(t, "bob", http.MethodPatch, "/api/v0/issues/"+low.String(), `{"parent_id":"`+up.String()+`"}`, http.StatusOK)
	said, want := w.parentRows(t, low), []string{"bob: WEB-4 → WEB-2, " + mid.String() + " → " + up.String()}
	if !reflect.DeepEqual(w.parentOf(t, up), &top) || !reflect.DeepEqual(w.parentOf(t, low), &up) || !slices.Equal(said, want) {
		t.Errorf("Up is under %v, Low under %v, Low's rows of its parent %q; want Up under Top, %s, Low under Up, %s, and %q",
			w.parentOf(t, up), w.parentOf(t, low), said, top, up, want)
	}
````
````old server/internal/bootstrap/issue_parents_test.go
// His new work item under alice's Up is refused, 422 at parent_id, as
// under one not there: no work item made. bob, Web's member, makes one
// under Up, answered and stored under it. Under his own work item walt
// makes one.
````
````new server/internal/bootstrap/issue_parents_test.go
// His new work item under alice's Up is refused, and so is his work item
// put under it, 422 at parent_id, as under one not there: no work item
// made, nothing written. bob, Web's member, makes one under Up, answered
// and stored under it. Under his own work item walt makes one and moves
// one, and bob puts his own work item under Up.
````
````old server/internal/bootstrap/issue_parents_test.go
		t.Errorf("walt's work item under Up answers %s, and Web has %d work items; want %s, and 2", got, w.workItems(t), parentNotAllowed)
	}
````
````new server/internal/bootstrap/issue_parents_test.go
		t.Errorf("walt's work item under Up answers %s, and Web has %d work items; want %s, and 2", got, w.workItems(t), parentNotAllowed)
	}
	_, changed, by := archivalOf(t, w, his)
	answer = w.send(t, "walt", http.MethodPatch, "/api/v0/issues/"+his.String(), `{"parent_id":"`+up.String()+`"}`,
		http.StatusUnprocessableEntity)
	if _, at, who := archivalOf(t, w, his); refusalOf(t, []byte(answer), true) != parentNotAllowed || w.parentOf(t, his) != nil ||
		!at.Equal(changed) || who != by {
		t.Errorf("walt's work item put under Up answers %s, and is under %v, changed last by %s at %v; want %s, under none, as before",
			answer, w.parentOf(t, his), who, at, parentNotAllowed)
	}
````
````old server/internal/bootstrap/issue_parents_test.go
	if got := w.parentOf(t, below); !reflect.DeepEqual(got, &his) {
		t.Errorf("Below is under %v, want His, %s", got, his)
````
````new server/internal/bootstrap/issue_parents_test.go
	moved := w.newWorkItemBy(t, "walt", w.web, `{"name":"Moved"}`)
	w.send(t, "walt", http.MethodPatch, "/api/v0/issues/"+moved.String(), `{"parent_id":"`+his.String()+`"}`, http.StatusOK)
	bobs := w.newWorkItemBy(t, "bob", w.web, `{"name":"Bob's"}`)
	w.send(t, "bob", http.MethodPatch, "/api/v0/issues/"+bobs.String(), `{"parent_id":"`+up.String()+`"}`, http.StatusOK)
	for _, tt := range []struct {
		item uuid.UUID
		want *uuid.UUID
	}{{below, &his}, {moved, &his}, {bobs, &up}} {
		if got := w.parentOf(t, tt.item); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("the work item %s is under %v, want %v", tt.item, got, tt.want)
		}
````

`server/internal/bootstrap/issue_writes_test.go`（修改，2 处）：

````old server/internal/bootstrap/issue_writes_test.go
// makes Part under it, its parent read under the lock. She archives it,
// in Done, and unarchives it; she deletes the second. With Web archived,
// her creation and her change are refused under the lock (409).
````
````new server/internal/bootstrap/issue_writes_test.go
// makes Part under it, its parent read under the lock, then moves Part
// under the second, the second's ancestry read under it. She archives the
// first, in Done, and unarchives it; she deletes the second, and Part
// under it. With Web archived, her creation and her change are refused
// under the lock (409).
````
````old server/internal/bootstrap/issue_writes_test.go
	decodeAnswer(t, send(alice, `{"name":"Part","parent_id":"`+first.ID.String()+`"}`, http.StatusCreated), &part)
````
````new server/internal/bootstrap/issue_writes_test.go
	decodeAnswer(t, send(alice, `{"name":"Part","parent_id":"`+first.ID.String()+`"}`, http.StatusCreated), &part)
	route.answerWithin(t, w.contract, http.MethodPatch, "/api/v0/issues/"+part.ID.String(), alice, `{"parent_id":"`+defaulted.ID.String()+`"}`,
		http.StatusOK)
````

`server/internal/bootstrap/ports_test.go`（修改，2 处）：

````old server/internal/bootstrap/ports_test.go
	fake := &fakeIssueProjects{web: web, locked: project.IssueProject{WorkspaceID: acme, Archived: true, DefaultAssigneeID: &ann},
````
````new server/internal/bootstrap/ports_test.go
	fake := &fakeIssueProjects{web: web, locked: project.IssueProject{WorkspaceID: acme, Archived: true, DefaultAssigneeID: &ann, Identifier: "WEB"},
````
````old server/internal/bootstrap/ports_test.go
	if l, found, err := p.Lock(ctx, web); err != nil || !found || l != (issue.LockedProject{WorkspaceID: acme, Archived: true, DefaultAssigneeID: &ann}) {
````
````new server/internal/bootstrap/ports_test.go
	if l, found, err := p.Lock(ctx, web); err != nil || !found ||
		l != (issue.LockedProject{WorkspaceID: acme, Archived: true, DefaultAssigneeID: &ann, Identifier: "WEB"}) {
````

- [ ] **Step 7: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/issue/... ./internal/modules/project/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestAWorkItemIsPutUnderAnother$|TestTheGuestRuleHoldsForAParent$|TestTwoMovesThatWouldMakeALoopSerialize$|TestIssueProjectsConvertsProjectsAnswers$|TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection$' ./internal/bootstrap/`
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
git add api/modules/issue.yaml server/internal/bootstrap/issue_interleaving_hierarchy_test.go server/internal/bootstrap/issue_parents_test.go server/internal/bootstrap/issue_writes_test.go server/internal/bootstrap/ports_test.go server/internal/modules/issue/adapter/http/issues.go server/internal/modules/issue/adapter/http/update_test.go server/internal/modules/issue/adapter/postgres/parents.go server/internal/modules/issue/adapter/postgres/parents_test.go server/internal/modules/issue/adapter/postgres/queries/parents.sql server/internal/modules/issue/adapter/postgres/queries/update.sql server/internal/modules/issue/adapter/postgres/update.go server/internal/modules/issue/app/create_issue_test.go server/internal/modules/issue/app/fakes_parents_test.go server/internal/modules/issue/app/fakes_test.go server/internal/modules/issue/app/parents.go server/internal/modules/issue/app/parents_test.go server/internal/modules/issue/app/ports.go server/internal/modules/issue/app/rows.go server/internal/modules/issue/app/update_issue.go server/internal/modules/issue/domain/activity.go server/internal/modules/issue/domain/activity_test.go server/internal/modules/issue/domain/dates.go server/internal/modules/issue/domain/hierarchy.go server/internal/modules/issue/domain/identifier.go server/internal/modules/issue/domain/identifier_test.go server/internal/modules/issue/domain/patch.go server/internal/modules/issue/domain/patch_test.go server/internal/modules/project/adapter/postgres/issue_ports.go server/internal/modules/project/adapter/postgres/issue_ports_test.go server/internal/modules/project/adapter/postgres/queries/issue_ports.sql server/internal/modules/project/app/issue_ports.go api/dist/openapi.yaml server/internal/modules/issue/adapter/http/gen/bodyshape.gen.go server/internal/modules/issue/adapter/http/gen/server.gen.go server/internal/modules/issue/adapter/postgres/gen/parents.sql.go server/internal/modules/issue/adapter/postgres/gen/update.sql.go server/internal/modules/project/adapter/postgres/gen/issue_ports.sql.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M4/P3): an update moves a work item under another parent, never in a loop

updateIssue takes parent_id, null for none: under the project's lock the
new parent is read as a creation's is, then its ancestry, which must not
hold the work item, then the old parent. The activity names both by their
identifiers, read under the lock (the project's lock answers the
project's identifier), and their ids. Two moves that would together make
a loop serialize on the project's row: the second finds itself in the
ancestry and is refused (interleaving 9).

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p3.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试，或没有替身的平台、领域、转换的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；守卫是生成器（`make gen-go`）和中英文翻译键（`make lint-web`）的核对。每个变异在一份新的副本里，在本 Task 的树上、标出的较晚的 Task 的树上和最终的树上都跑过，端到端的检查只在标出的 Task 的树上）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T3.1` | 修改把工作项放到它下面的工作项下、或者它自己下面：不查祖先链 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemIsPutUnderAnother`（`make test`）、`internal/bootstrap` 的 `TestTwoMovesThatWouldMakeALoopSerialize`（`make test`） | 单元；组合 |
| `T3.2` | 工作项的祖先链只有它和它的父工作项 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemIsPutUnderAnother`（`make test`）、`internal/bootstrap` 的 `TestWorkItemsArePutUnderAWorkItem`（`make test`）（Task 6 起） | 存储；组合 |
| `T3.3` | 工作项的祖先链不含它自己 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemIsPutUnderAnother`（`make test`） | 存储；组合 |
| `T3.4` | 工作项的祖先链到已归档的工作项为止 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemIsPutUnderAnother`（`make test`） | 存储；组合 |
| `T3.5` | 移动不写父工作项的那行动态 | `internal/modules/issue/domain` 的全部测试（`make test`）、`internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemIsPutUnderAnother`（`make test`）、故事 `i8-sub-issues`（Task 15 起） | 单元；组合；端到端 |
| `T3.6` | 父工作项的那行动态把原来的和新的父工作项写反 | `internal/modules/issue/domain` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemIsPutUnderAnother`（`make test`）、故事 `i8-sub-issues`（Task 15 起） | 单元；组合；端到端 |
| `T3.7` | 动态只按编号、不带项目的标识称呼工作项 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemIsPutUnderAnother`（`make test`）、故事 `i8-sub-issues`（Task 15 起） | 单元；组合；端到端 |
| `T3.8` | `ProjectGate.Lock` 不回答标识 | `internal/modules/project/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemIsPutUnderAnother`（`make test`） | 存储；组合 |
| `T3.9` | 给出与工作项相同的父工作项也算改动 | `internal/modules/issue/domain` 的全部测试（`make test`）、`internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemIsPutUnderAnother`（`make test`） | 单元；组合 |
| `T3.10` | `PATCH` 丢掉请求体的 `parent_id` | `internal/modules/issue/adapter/http` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemIsPutUnderAnother`（`make test`）、故事 `i8-sub-issues`（Task 15 起） | 单元；组合；端到端 |
| `T3.11` | `UpdateIssue` 不写父工作项：每次修改都把工作项从父工作项下拿出来 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemIsPutUnderAnother`（`make test`） | 存储；组合 |
| `T3.12` | `updateIssue` 检查新的父工作项却写原来的 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemIsPutUnderAnother`（`make test`） | 单元；组合 |
| `T3.13` | 不存在的原父工作项当作没有 | `internal/modules/issue/app` 的全部测试（`make test`） | 单元 |
| `T3.14` | 父工作项的那行动态不写新父工作项的 id | `internal/modules/issue/domain` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemIsPutUnderAnother`（`make test`） | 单元；组合 |
| `T3.15` | 移动经连接池读祖先链，不在它的事务的连接上 | `internal/bootstrap` 的 `TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection`（`make test`） | 组合 |
| `T3.16` | 移动只允许到没有父工作项的工作项下 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemIsPutUnderAnother`（`make test`） | 单元；组合 |

本 Task 的检查也发现较早的 Task 的变异 `T2.1`、`T2.2`（列在各自的 Task 的表里，标"（Task 3 起）"）：本 Task 做完之后，在本 Task 的树上重跑它们（`mutants_p3.py`，`MUT_LATER=3`）。

---

### Task 4: `sub_issues_count` 和父工作项：按调用者的规则回答

**Files:**
- Create: `server/internal/bootstrap/issue_sub_issue_count_test.go`、`server/internal/modules/issue/adapter/postgres/queries/sub_issues.sql`、`server/internal/modules/issue/adapter/postgres/sub_issues.go`、`server/internal/modules/issue/adapter/postgres/sub_issues_test.go`、`server/internal/modules/issue/app/seen.go`、`server/internal/modules/issue/app/seen_test.go`
- Modify: `api/modules/issue.yaml`、`server/internal/bootstrap/issue_parents_test.go`、`server/internal/bootstrap/issue_writes_test.go`、`server/internal/modules/issue/adapter/http/issues.go`、`server/internal/modules/issue/adapter/http/issues_test.go`、`server/internal/modules/issue/adapter/postgres/issues.go`、`server/internal/modules/issue/adapter/postgres/queries/issues.sql`、`server/internal/modules/issue/app/archive_issue.go`、`server/internal/modules/issue/app/archive_issue_test.go`、`server/internal/modules/issue/app/create_issue.go`、`server/internal/modules/issue/app/dates_test.go`、`server/internal/modules/issue/app/fakes_parents_test.go`、`server/internal/modules/issue/app/fakes_test.go`、`server/internal/modules/issue/app/get_issue.go`、`server/internal/modules/issue/app/get_issue_by_identifier.go`、`server/internal/modules/issue/app/get_issue_by_identifier_test.go`、`server/internal/modules/issue/app/get_issue_test.go`、`server/internal/modules/issue/app/guest_rule_test.go`、`server/internal/modules/issue/app/lock.go`、`server/internal/modules/issue/app/lock_test.go`、`server/internal/modules/issue/app/parents_test.go`、`server/internal/modules/issue/app/ports.go`、`server/internal/modules/issue/app/update_issue.go`、`server/internal/modules/issue/app/update_issue_test.go`、`server/internal/modules/issue/domain/issue.go`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/issue/adapter/http/gen/server.gen.go`、`server/internal/modules/issue/adapter/postgres/gen/issues.sql.go`、`server/internal/modules/issue/adapter/postgres/gen/sub_issues.sql.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**（spec 2.6、第 3 节第 9 条；M4 设计 3.4、3.9、5.2，12 节约束 5）
- 契约：`IssueDetail.sub_issues_count`（必有，整数）：未删除、没归档的直接子工作项中调用者看得到的个数（只看自己的访客只数他建的）；`IssueDetail.parent_id` 的描述：没有父工作项时为 `null`，父工作项调用者看不到（只看自己的工作项的访客）时也是（Task 6 的 `Issue.parent_id` 照写）。
- `issue/domain`：`Issue.SubIssuesCount`（调用者按他在项目上的 `Grant` 看到的个数：存储的读留 0，建回答的时候数（`Locks.seen`）；新建的回答 0，新的工作项没有子工作项）。
- `issue/adapter/postgres`：`queries/sub_issues.sql`（新）的 `SubIssueCounts`（给的每个有子工作项的工作项一行：全部的个数和给的账户建的个数，`issues_parent_id_idx`）；`sub_issues.go`（新）：`SubIssueCounts(ctx, actor, ids) (map[uuid.UUID]app.SubIssueCount, error)`；`queries/issues.sql` 的 `GetIssues`（给的 `id` 中未删除的，已归档的也在，与 `GetIssue` 同样的列，`id` 的次序；不加锁，或在写的事务里、项目的锁下读）、`issues.go` 的 `issueOf`（两个读共用的转换）和 `GetIssues`。两者失败的错误只说几个工作项，不列出每个 `id`（数和读的工作项可以是整个列表）。
- `issue/app`：`IssueReader.SubIssueCounts`、`IssueReader.GetIssues`、`SubIssueCount{All, Own}`；`seen.go`（新）：`grants`（按项目的 `Grant`）、`Locks.seen(ctx, actor, g, items...)`（一次数完，`OwnOnly` 的项目用 `Own`；工作项的项目在 `g` 里没有 `Grant` 是错误：每个回答都先判定）、`Locks.hideParents`（`OwnOnly` 的项目里有父工作项的那些，在数的同一处、同一个连接上一次读出它们的父工作项（`GetIssues`，去重、按 `id`），访客规则的那一个谓词 `domain.Issue.HiddenFrom` 藏起的、读不到的答成 `null`；别的 `Grant` 什么都不多读；读失败是回答的错误）、`Locks.seenOne`；`Locks.see` 多交回判定的 `Grant`。`getIssue`、`getIssueByIdentifier` 在判定之后、事务之外数和读父工作项；`updateIssue`、`archiveIssue`、`unarchiveIssue` 在锁下判定的 `Grant` 上、事务里最后数（读回之后，或什么都不写时在判定之后）；`createIssue` 回答 0（新的工作项没有子工作项，不数）。数失败是回答的错误，照原样，哪条回答的路都不吞掉它。以后每个回答都经 `seen`（spec 第 3 节第 9 条：回答里 `parent_id` 指名调用者看不到的工作项时为 `null`）。
- 测试的帮手：`fakes_parents_test.go` 的 `fakeIssues.SubIssueCounts`（从假的工作项数）、`fakeIssues.GetIssues`（假的工作项里有的，按 `id`）；`fakes_test.go` 的 `fakeIssues.read`（`GetIssue` 和多个的读共用：它的别的工作项，或它留着的那个）；各用例的单元测试的期望的调用加上计数的一行（`counted`），读计数失败的回答。
- HTTP：`issueDetail` 映射 `sub_issues_count`。

**Tests:**
- 单元：`TestAReadCountsTheSubItemsTheCallerSees`（读：判定之后、事务之外按 `Grant` 数：全部，或"只看自己"时只数他自己的；项目"可见全部"时全部）、`TestAWritesAnswerCountsTheSubItemsTheCallerSees`（改、什么都不改、归档、再归档、恢复、再恢复：按锁下判定的 `Grant`、在事务里最后数；管理员数全部，只是项目访客的工作区管理员（`adminGuest`）只数自己的；每个写数失败时答它的失败，照原样，是它最后的调用）、`TestAnAnswerNamesTheParentTheCallerSees`（只看自己的访客：别人的父工作项答 `null`，她自己的照答，读不到的答 `null`，都在计数之后、事务之外一次读；看得到每个工作项的 `Grant` 照答、不多读；只是项目访客的工作区管理员改她的工作项时在事务里、计数之后读；读父工作项失败是回答的错误）；读、写的单元测试（`TestGetIssueReadsTheWorkItem`、`TestGetIssueRefusals`、`TestGetIssueByIdentifierReadsTheWorkItem`、归档、修改、日期、父工作项、访客规则、写闸门的测试）的期望的调用照改；数失败照原样是回答的错误：`TestGetIssueRefusals`、`TestGetIssueByIdentifierRefusals`、`TestArchiveIssueRefusals` 各有数失败的一行，`TestUpdateIssueStopsAtAFailingPort` 的端口表有 `SubIssueCounts`。
- 存储：`TestSubIssueCounts`（按建的人数，不论谁最后改的：p 的 a、b 是 alice 的，c、d 是 bob 的、c 最后由 alice 改、d 已归档，e 已删除；a 下的 g 不算 p 的；对 alice p 有 3 个、2 个是她的，对 bob p 有 3 个、1 个是他的；没有子工作项的、没有的工作项不回答）、`TestGetIssues`（未删除的，已归档的也在，`id` 的次序，不论给的、存的次序：`id` 最小的 a 在别的存好之后移到 q 下，它的行在表里、在 `(project_id, state_id)` 的索引里都到了最后，没有 `ORDER BY` 的读不论扫表还是读那个索引都把它答在最后；`newFamily` 造的一家，Task 5 的存储测试也用它）。
- HTTP：`issues_test.go` 的工作项和回答的 JSON 带 `sub_issues_count`（2；新建的 0），读、改、建的测试都比较它。
- 组合：`TestEachAnswerCountsTheSubItemsTheCallerSees`（walt 是 acme 的管理员、Web 的访客，建 His；alice 在它下面放她的 Part，walt 放他的 Mine（alice 改了它的名字：最后改它的是 alice，它仍是 walt 的），alice 放一个已归档的和一个已删除的，Part 下的 Low 不是 His 的；每一步 walt 和 alice 按 `id`、按编号读 His，改它、什么都不改、移到 Done、归档两次、恢复两次：每个回答 walt 是 1（Mine），alice 是 2；新建的是 0）、`TestAHiddenParentIsNotInTheAnswer`（alice 把 walt 的 His 放到她的 Up 下，walt 读 Up 是 404；walt 按 `id`、按编号读 His 和改它的名字，每个回答与 alice 此刻读到的整个相同，只是 `parent_id` 为 `null`；他自己的 Mine 下的 Below 照答 Mine；看得到每个工作项的 bob 读到 His 在 Up 下）、`TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection`（一个连接的池：walt 成为 acme 的管理员、Web 的访客（`adminGuest`），建 His，alice 把它放到第一个工作项下，walt 改它：他看不到的第一个工作项与回答的计数一起在锁下、在事务的连接上读，回答的 `parent_id` 为 `null`）。

- [ ] **Step 1: 契约和查询，生成**

`api/modules/issue.yaml`（修改，3 处）：

````old api/modules/issue.yaml
        completed_at, archived_at, assignee_ids, label_ids, created_by_id, updated_by_id, created_at, updated_at]
````
````new api/modules/issue.yaml
        completed_at, archived_at, assignee_ids, label_ids, sub_issues_count, created_by_id, updated_by_id, created_at, updated_at]
````
````old api/modules/issue.yaml
          description: The work item it is under, in the same project; null for none.
````
````new api/modules/issue.yaml
          description: >-
            The work item it is under, in the same project; null for none, or
            for one the caller does not see (a guest who sees only his own
            work items).
````
````old api/modules/issue.yaml
            type: string
            format: uuid
        created_by_id:
````
````new api/modules/issue.yaml
            type: string
            format: uuid
        sub_issues_count:
          description: >-
            The number of the work items directly under it that are not
            archived and that the caller sees: a guest of the project who
            sees only his own work items counts only his own.
          type: integer
        created_by_id:
````

`server/internal/modules/issue/adapter/postgres/queries/issues.sql`（修改，1 处）：

````old server/internal/modules/issue/adapter/postgres/queries/issues.sql
WHERE i.id = sqlc.arg(id) AND i.deleted_at IS NULL;

````
````new server/internal/modules/issue/adapter/postgres/queries/issues.sql
WHERE i.id = sqlc.arg(id) AND i.deleted_at IS NULL;

-- name: GetIssues :many
-- The undeleted work items among ids, archived or not, each as GetIssue reads it, in id order (M4 design 3.4, 3.9), in a
-- write's transaction, under its project's lock, or without a lock.
SELECT i.id, i.workspace_id, i.project_id, i.sequence_id, i.name, i.state_id, i.priority, i.start_date, i.target_date,
       i.parent_id, i.sort_order, i.completed_at, i.archived_at, i.created_by_id, i.updated_by_id, i.created_at, i.updated_at,
       (SELECT array_agg(a.assignee_id ORDER BY a.assignee_id) FROM issue_assignees a
        WHERE a.issue_id = i.id AND a.deleted_at IS NULL)::uuid[] AS assignee_ids,
       (SELECT array_agg(l.label_id ORDER BY l.label_id) FROM issue_labels l
        WHERE l.issue_id = i.id AND l.deleted_at IS NULL)::uuid[] AS label_ids
FROM issues i
WHERE i.id = ANY (sqlc.arg(ids)::uuid[]) AND i.deleted_at IS NULL
ORDER BY i.id;

````

`server/internal/modules/issue/adapter/postgres/queries/sub_issues.sql`（新文件，9 行）：

````file server/internal/modules/issue/adapter/postgres/queries/sub_issues.sql
-- name: SubIssueCounts :many
-- For each of the work items ids that has any, the number of its undeleted, unarchived direct sub-items, and of those
-- the account actor created (M4 design 3.9): a work item's sub_issues_count as its reader sees it. Read without a lock,
-- or under the project's lock of the write that answers. issues_parent_id_idx serves it.
SELECT parent_id::uuid AS parent_id, count(*)::int AS all_count,
       (count(*) FILTER (WHERE created_by_id = sqlc.arg(actor)::uuid))::int AS own_count
FROM issues
WHERE parent_id = ANY (sqlc.arg(ids)::uuid[]) AND deleted_at IS NULL AND archived_at IS NULL
GROUP BY parent_id;
````

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `fb704ca32d597cfec1e742ef582345fa49ab1ee7587163fc9ebb7d9eed4e95f6` | 3348 | `api/dist/openapi.yaml` |
| `1dc3d62bc22d44b03444e88615b235cdc27e64c27216fffc4fe1d83c29e4dbde` | 1121 | `server/internal/modules/issue/adapter/http/gen/server.gen.go` |
| `b918552b0063744b04cc59adecebdc58678ff1367b89c48161d58b615a573bd3` | 381 | `server/internal/modules/issue/adapter/postgres/gen/issues.sql.go` |
| `27f4d03d09592715645d9c0acfc08e49a1d58f40d4f721424f7ea7fe33391a65` | 54 | `server/internal/modules/issue/adapter/postgres/gen/sub_issues.sql.go` |
| `9458967469ec033daf357e6c008264a072cf167f09dfb6ba27e55c06432f0af1` | 3708 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/issue/adapter/http/gen/server.gen.go server/internal/modules/issue/adapter/postgres/gen/issues.sql.go server/internal/modules/issue/adapter/postgres/gen/sub_issues.sql.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 2: 领域和用例**

`server/internal/modules/issue/app/archive_issue.go`（修改，3 处）：

````old server/internal/modules/issue/app/archive_issue.go
// work item as stored, read back in the transaction.
````
````new server/internal/modules/issue/app/archive_issue.go
// work item as stored, read back in the transaction, as the caller sees
// it (Locks.seen), the work item as read when nothing is written.
````
````old server/internal/modules/issue/app/archive_issue.go
			answer = h.issue
			return nil
````
````new server/internal/modules/issue/app/archive_issue.go
			answer, err = d.Locks.seenOne(ctx, actor.UserID, h.grant, h.issue)
			return err
````
````old server/internal/modules/issue/app/archive_issue.go
		answer, err = readBack(ctx, d.Issues, id)
````
````new server/internal/modules/issue/app/archive_issue.go
		if answer, err = readBack(ctx, d.Issues, id); err != nil {
			return err
		}
		answer, err = d.Locks.seenOne(ctx, actor.UserID, h.grant, answer)
````

`server/internal/modules/issue/app/archive_issue_test.go`（修改，8 处）：

````old server/internal/modules/issue/app/archive_issue_test.go
// row 14, alice's at the moment; the answer read back. A work item in
// Cancelled is archived as one in Done.
````
````new server/internal/modules/issue/app/archive_issue_test.go
// row 14, alice's at the moment; the answer read back, its sub-items
// counted (Locks.seen). A work item in Cancelled is archived as one in
// Done.
````
````old server/internal/modules/issue/app/archive_issue_test.go
				line("GetIssue", id))
````
````new server/internal/modules/issue/app/archive_issue_test.go
				line("GetIssue", id), counted(id))
````
````old server/internal/modules/issue/app/archive_issue_test.go
	calls := append(lockedAs(domain.ActionArchive, id), "Now", line("SetArchivedAt", id), line("InsertActivities", 1), line("GetIssue", id))
````
````new server/internal/modules/issue/app/archive_issue_test.go
	calls := append(lockedAs(domain.ActionArchive, id), "Now", line("SetArchivedAt", id), line("InsertActivities", 1), line("GetIssue", id),
		counted(id))
````
````old server/internal/modules/issue/app/archive_issue_test.go
// last writer and moment as they were (M4 design 3.8).
````
````new server/internal/modules/issue/app/archive_issue_test.go
// last writer and moment as they were, its sub-items counted (M4 design
// 3.8, 3.9).
````
````old server/internal/modules/issue/app/archive_issue_test.go
			if calls := lockedAs(domain.ActionArchive, id); !slices.Equal(w.log.calls, calls) || w.issues.archival != nil ||
````
````new server/internal/modules/issue/app/archive_issue_test.go
			if calls := append(lockedAs(domain.ActionArchive, id), counted(id)); !slices.Equal(w.log.calls, calls) || w.issues.archival != nil ||
````
````old server/internal/modules/issue/app/archive_issue_test.go
// precheck; a read or a write that fails, its failure as itself; a request
// without a caller, unauthenticated before any read.
````
````new server/internal/modules/issue/app/archive_issue_test.go
// precheck; a read, a write or a count that fails, its failure as itself;
// a request without a caller, unauthenticated before any read.
````
````old server/internal/modules/issue/app/archive_issue_test.go
		{"the activity's write fails", true, func(w *world) { w.log.fail = "InsertActivities" }, as(alice), errDisk, 10},
````
````new server/internal/modules/issue/app/archive_issue_test.go
		{"the activity's write fails", true, func(w *world) { w.log.fail = "InsertActivities" }, as(alice), errDisk, 10},
		{"the count fails", true, func(w *world) { w.log.fail = "SubIssueCounts" }, as(alice), errDisk, 12},
````
````old server/internal/modules/issue/app/archive_issue_test.go
				line("InsertActivities", 1)})[:tt.calls]
````
````new server/internal/modules/issue/app/archive_issue_test.go
				line("InsertActivities", 1), line("GetIssue", id), counted(id)})[:tt.calls]
````

`server/internal/modules/issue/app/create_issue.go`（修改，1 处）：

````old server/internal/modules/issue/app/create_issue.go
// item as stored, read back in the transaction.
````
````new server/internal/modules/issue/app/create_issue.go
// item as stored, read back in the transaction, which has no sub-items to
// count yet (Issue.SubIssuesCount).
````

`server/internal/modules/issue/app/dates_test.go`（修改，2 处）：

````old server/internal/modules/issue/app/dates_test.go
// work item read back. Web's work item 7 runs from the 12th to the 19th:
// both dates moved past the target date it had, which a start date given
// with the target date is not held to; the start date cleared, and a
// target date before the start date it had; a start date alone, on the
// target date it keeps; the target date cleared.
````
````new server/internal/modules/issue/app/dates_test.go
// work item read back, its sub-items counted. Web's work item 7 runs from
// the 12th to the 19th: both dates moved past the target date it had, which
// a start date given with the target date is not held to; the start date
// cleared, and a target date before the start date it had; a start date
// alone, on the target date it keeps; the target date cleared.
````
````old server/internal/modules/issue/app/dates_test.go
			if calls := append(locked(id), "Now", line("UpdateIssue", id), line("InsertActivities", len(rows)), line("GetIssue", id)); !slices.Equal(
````
````new server/internal/modules/issue/app/dates_test.go
			if calls := append(locked(id), "Now", line("UpdateIssue", id), line("InsertActivities", len(rows)), line("GetIssue", id),
				counted(id)); !slices.Equal(
````

`server/internal/modules/issue/app/fakes_parents_test.go`（修改，2 处）：

````old server/internal/modules/issue/app/fakes_parents_test.go
	"context"
	"uuid"
````
````new server/internal/modules/issue/app/fakes_parents_test.go
	"context"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
````
````old server/internal/modules/issue/app/fakes_parents_test.go
	return []uuid.UUID{id}, nil
}

````
````new server/internal/modules/issue/app/fakes_parents_test.go
	return []uuid.UUID{id}, nil
}

// SubIssueCounts logs the count and answers the numbers of sub-items the
// fake is told of the work items ids, and none of the others (fakeIssues'
// counts).
func (f *fakeIssues) SubIssueCounts(ctx context.Context, actor uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]app.SubIssueCount, error) {
	if err := f.log.add(ctx, "SubIssueCounts", actor, ids); err != nil {
		return nil, err
	}
	counts := map[uuid.UUID]app.SubIssueCount{}
	for _, id := range ids {
		if c, ok := f.counts[id]; ok {
			counts[id] = c
		}
	}
	return counts, nil
}

// GetIssues logs the read and answers the work items among ids that the
// fake has (read), in id order.
func (f *fakeIssues) GetIssues(ctx context.Context, ids []uuid.UUID) ([]domain.Issue, error) {
	if err := f.log.add(ctx, "GetIssues", ids); err != nil {
		return nil, err
	}
	var out []domain.Issue
	for _, id := range slices.SortedFunc(slices.Values(ids), uuid.UUID.Compare) {
		if i, found := f.read(id); found {
			out = append(out, i)
		}
	}
	return out, nil
}

````

`server/internal/modules/issue/app/fakes_test.go`（修改，6 处）：

````old server/internal/modules/issue/app/fakes_test.go
// are the store's other work items, answered as they are, and ancestry
// their ancestries (fakes_parents_test.go).
````
````new server/internal/modules/issue/app/fakes_test.go
// are the store's other work items, answered as they are, ancestry their
// ancestries, and counts the work items' numbers of sub-items
// (fakes_parents_test.go).
````
````old server/internal/modules/issue/app/fakes_test.go
	ancestry   map[uuid.UUID][]uuid.UUID
````
````new server/internal/modules/issue/app/fakes_test.go
	ancestry   map[uuid.UUID][]uuid.UUID
	counts     map[uuid.UUID]app.SubIssueCount
````
````old server/internal/modules/issue/app/fakes_test.go
		return domain.Issue{}, false, err
	}
````
````new server/internal/modules/issue/app/fakes_test.go
		return domain.Issue{}, false, err
	}
	i, found := f.read(id)
	return i, found, nil
}

// read is the work item id as the fake has it: one of its others, or the
// work item it keeps; found is false when it has none.
func (f *fakeIssues) read(id uuid.UUID) (domain.Issue, bool) {
````
````old server/internal/modules/issue/app/fakes_test.go
		return o, true, nil
````
````new server/internal/modules/issue/app/fakes_test.go
		return o, true
````
````old server/internal/modules/issue/app/fakes_test.go
		return domain.Issue{}, false, nil
````
````new server/internal/modules/issue/app/fakes_test.go
		return domain.Issue{}, false
````
````old server/internal/modules/issue/app/fakes_test.go
			uuid.UUID.Compare), CreatedByID: creatorOf(r.By), UpdatedByID: &by, CreatedAt: r.At, UpdatedAt: at}, true, nil
````
````new server/internal/modules/issue/app/fakes_test.go
			uuid.UUID.Compare), CreatedByID: creatorOf(r.By), UpdatedByID: &by, CreatedAt: r.At, UpdatedAt: at}, true
````

`server/internal/modules/issue/app/get_issue.go`（修改，2 处）：

````old server/internal/modules/issue/app/get_issue.go
// forbidden.
````
````new server/internal/modules/issue/app/get_issue.go
// forbidden. The answer is the work item as he sees it (Locks.seen).
````
````old server/internal/modules/issue/app/get_issue.go
	return u.locks.see(ctx, actor, id, domain.ActionRead)
````
````new server/internal/modules/issue/app/get_issue.go
	i, grant, err := u.locks.see(ctx, actor, id, domain.ActionRead)
	if err != nil {
		return domain.Issue{}, err
	}
	return u.locks.seenOne(ctx, actor.UserID, grant, i)
````

`server/internal/modules/issue/app/get_issue_by_identifier.go`（修改，2 处）：

````old server/internal/modules/issue/app/get_issue_by_identifier.go
// not read it, forbidden. An archived work item is read as any other.
````
````new server/internal/modules/issue/app/get_issue_by_identifier.go
// not read it, forbidden. An archived work item is read as any other. The
// answer is the work item as he sees it (Locks.seen).
````
````old server/internal/modules/issue/app/get_issue_by_identifier.go
	return u.d.Locks.see(ctx, actor, id, domain.ActionRead)
````
````new server/internal/modules/issue/app/get_issue_by_identifier.go
	i, grant, err := u.d.Locks.see(ctx, actor, id, domain.ActionRead)
	if err != nil {
		return domain.Issue{}, err
	}
	return u.d.Locks.seenOne(ctx, actor.UserID, grant, i)
````

`server/internal/modules/issue/app/get_issue_by_identifier_test.go`（修改，5 处）：

````old server/internal/modules/issue/app/get_issue_by_identifier_test.go
		if calls := identified(7, id); !slices.Equal(b.log.calls, calls) {
````
````new server/internal/modules/issue/app/get_issue_by_identifier_test.go
		if calls := append(identified(7, id), counted(id)+" outside tx"); !slices.Equal(b.log.calls, calls) {
````
````old server/internal/modules/issue/app/get_issue_by_identifier_test.go
// role that may not read it, forbidden. A read or a decision that fails,
// its failure as itself; a request without a caller, unauthenticated
// before any read. Each answer is compared whole, not by its code alone.
````
````new server/internal/modules/issue/app/get_issue_by_identifier_test.go
// role that may not read it, forbidden. A read, a decision or the count of
// the answer's sub-items that fails, its failure as itself; a request
// without a caller, unauthenticated before any read. Each answer is
// compared whole, not by its code alone.
````
````old server/internal/modules/issue/app/get_issue_by_identifier_test.go
		calls      int // of identified's
````
````new server/internal/modules/issue/app/get_issue_by_identifier_test.go
		calls      int // of identified's, then the count's
````
````old server/internal/modules/issue/app/get_issue_by_identifier_test.go
		{"the work item's read fails", func(b byIdentifier) { b.log.fail = "GetIssue" }, as(alice), "acme", "WEB-7", errDisk, 5},
````
````new server/internal/modules/issue/app/get_issue_by_identifier_test.go
		{"the work item's read fails", func(b byIdentifier) { b.log.fail = "GetIssue" }, as(alice), "acme", "WEB-7", errDisk, 5},
		{"the count fails", func(b byIdentifier) { b.log.fail = "SubIssueCounts" }, as(alice), "acme", "WEB-7", errDisk, 7},
````
````old server/internal/modules/issue/app/get_issue_by_identifier_test.go
			calls := identified(number, id)[:tt.calls]
````
````new server/internal/modules/issue/app/get_issue_by_identifier_test.go
			calls := append(identified(number, id), counted(id)+" outside tx")[:tt.calls]
````

`server/internal/modules/issue/app/get_issue_test.go`（修改，7 处）：

````old server/internal/modules/issue/app/get_issue_test.go
// issue.read in its project, of its workspace; the answer is the work item
// as read.
func TestGetIssueReadsTheWorkItem(t *testing.T) {
	uc, w, id := newGet()
````
````new server/internal/modules/issue/app/get_issue_test.go
// issue.read in its project, of its workspace, then the count of its
// sub-items; the answer is the work item as read, with the number of its
// sub-items alice sees, every one (Locks.seen).
func TestGetIssueReadsTheWorkItem(t *testing.T) {
	uc, w, id := newGet()
	w.issues.counts = map[uuid.UUID]app.SubIssueCount{id: {All: 2, Own: 1}}
````
````old server/internal/modules/issue/app/get_issue_test.go
	if want := readIssue(id); err != nil || !reflect.DeepEqual(got, want) {
````
````new server/internal/modules/issue/app/get_issue_test.go
	want := readIssue(id)
	want.SubIssuesCount = 2
	if err != nil || !reflect.DeepEqual(got, want) {
````
````old server/internal/modules/issue/app/get_issue_test.go
	if calls := []string{line("GetIssue", id, "outside tx"), line("Authorize", alice, domain.ActionRead, acme, web, "outside tx")}; !slices.Equal(
		w.log.calls, calls) {
````
````new server/internal/modules/issue/app/get_issue_test.go
	if calls := []string{line("GetIssue", id, "outside tx"), line("Authorize", alice, domain.ActionRead, acme, web, "outside tx"),
		counted(id) + " outside tx"}; !slices.Equal(w.log.calls, calls) {
````
````old server/internal/modules/issue/app/get_issue_test.go
// see, is issue.not_found; a role that may not read it, forbidden; a read
// or a decision that fails, its failure as itself; a request without a
// caller, unauthenticated before any read. Each answer is compared whole,
// not by its code alone.
````
````new server/internal/modules/issue/app/get_issue_test.go
// see, is issue.not_found; a role that may not read it, forbidden; a read,
// a decision or a count that fails, its failure as itself; a request
// without a caller, unauthenticated before any read. Each answer is
// compared whole, not by its code alone.
````
````old server/internal/modules/issue/app/get_issue_test.go
		calls int // of read, decided
````
````new server/internal/modules/issue/app/get_issue_test.go
		calls int // of read, decided, counted
````
````old server/internal/modules/issue/app/get_issue_test.go
		{"the decision fails", func(w *world, _ *uuid.UUID) { w.log.fail = "Authorize" }, as(alice), errDisk, 2},
````
````new server/internal/modules/issue/app/get_issue_test.go
		{"the decision fails", func(w *world, _ *uuid.UUID) { w.log.fail = "Authorize" }, as(alice), errDisk, 2},
		{"the count fails", func(w *world, _ *uuid.UUID) { w.log.fail = "SubIssueCounts" }, as(alice), errDisk, 3},
````
````old server/internal/modules/issue/app/get_issue_test.go
			calls := []string{line("GetIssue", id, "outside tx"), decided}[:tt.calls]
````
````new server/internal/modules/issue/app/get_issue_test.go
			calls := []string{line("GetIssue", id, "outside tx"), decided, counted(id) + " outside tx"}[:tt.calls]
````

`server/internal/modules/issue/app/guest_rule_test.go`（修改，2 处）：

````old server/internal/modules/issue/app/guest_rule_test.go
// such a guest, the work item read and the one decision taken; carol's id
// is after hers and bob's before it, so the rule tells her own by its
// creator, not by the ids' order. When web shows its guests every work
// item, she reads carol's. A refusal of her role on another's work item
// is issue.not_found too, the guest rule first; on her own, forbidden. A
// work item whose creator is not known, as one whose creator's account is
// purged (P9), is no guest's own: issue.not_found.
````
````new server/internal/modules/issue/app/guest_rule_test.go
// such a guest, the work item read and the one decision taken, its
// sub-items not counted; carol's id is after hers and bob's before it, so
// the rule tells her own by its creator, not by the ids' order. When web
// shows its guests every work item, she reads carol's. A refusal of her
// role on another's work item is issue.not_found too, the guest rule
// first; on her own, forbidden. A work item whose creator is not known, as
// one whose creator's account is purged (P9), is no guest's own:
// issue.not_found.
````
````old server/internal/modules/issue/app/guest_rule_test.go
			if calls := []string{line("GetIssue", id, "outside tx"), line("Authorize", alice, domain.ActionRead, acme, web, "outside tx")}; !slices.Equal(
				w.log.calls, calls) {
````
````new server/internal/modules/issue/app/guest_rule_test.go
			calls := []string{line("GetIssue", id, "outside tx"), line("Authorize", alice, domain.ActionRead, acme, web, "outside tx")}
			if tt.want == nil {
				calls = append(calls, counted(id)+" outside tx")
			}
			if !slices.Equal(w.log.calls, calls) {
````

`server/internal/modules/issue/app/lock.go`（修改，6 处）：

````old server/internal/modules/issue/app/lock.go
		i, err := l.see(ctx, actor, *w.issue, w.action)
````
````new server/internal/modules/issue/app/lock.go
		i, _, err := l.see(ctx, actor, *w.issue, w.action)
````
````old server/internal/modules/issue/app/lock.go
// the Authorizer's shared.Forbidden. It answers the work item as read.
func (l Locks) see(ctx context.Context, actor shared.Actor, id uuid.UUID, action shared.Action) (domain.Issue, error) {
````
````new server/internal/modules/issue/app/lock.go
// the Authorizer's shared.Forbidden. It answers the work item as read, and
// the decision's grant.
func (l Locks) see(ctx context.Context, actor shared.Actor, id uuid.UUID, action shared.Action) (domain.Issue, shared.Grant, error) {
````
````old server/internal/modules/issue/app/lock.go
	case err != nil:
		return domain.Issue{}, err
````
````new server/internal/modules/issue/app/lock.go
	case err != nil:
		return domain.Issue{}, shared.Grant{}, err
````
````old server/internal/modules/issue/app/lock.go
		return domain.Issue{}, domain.ErrIssueNotFound
````
````new server/internal/modules/issue/app/lock.go
		return domain.Issue{}, shared.Grant{}, domain.ErrIssueNotFound
````
````old server/internal/modules/issue/app/lock.go
	if _, err := l.decide(ctx, actor, action, i.WorkspaceID, i.ProjectID, &i, domain.ErrIssueNotFound); err != nil {
		return domain.Issue{}, err
````
````new server/internal/modules/issue/app/lock.go
	grant, err := l.decide(ctx, actor, action, i.WorkspaceID, i.ProjectID, &i, domain.ErrIssueNotFound)
	if err != nil {
		return domain.Issue{}, shared.Grant{}, err
````
````old server/internal/modules/issue/app/lock.go
	return i, nil
````
````new server/internal/modules/issue/app/lock.go
	return i, grant, nil
````

`server/internal/modules/issue/app/lock_test.go`（修改，1 处）：

````old server/internal/modules/issue/app/lock_test.go
		line("InsertActivities", 1), line("GetIssue", id), line("Leave", web)})
````
````new server/internal/modules/issue/app/lock_test.go
		line("InsertActivities", 1), line("GetIssue", id), counted(id), line("Leave", web)})
````

`server/internal/modules/issue/app/parents_test.go`（修改，4 处）：

````old server/internal/modules/issue/app/parents_test.go
// the work item under its new parent.
````
````new server/internal/modules/issue/app/parents_test.go
// the work item under its new parent, its sub-items counted.
````
````old server/internal/modules/issue/app/parents_test.go
				line("GetIssue", id)})
````
````new server/internal/modules/issue/app/parents_test.go
				line("GetIssue", id), counted(id)})
````
````old server/internal/modules/issue/app/parents_test.go
		!reflect.DeepEqual(got.ParentID, &k.up.ID) || !slices.Equal(w.log.calls, locked(id)) || w.issues.changed != nil {
````
````new server/internal/modules/issue/app/parents_test.go
		!reflect.DeepEqual(got.ParentID, &k.up.ID) || !slices.Equal(w.log.calls, append(locked(id), counted(id))) || w.issues.changed != nil {
````
````old server/internal/modules/issue/app/parents_test.go
			locked(id))
````
````new server/internal/modules/issue/app/parents_test.go
			append(locked(id), counted(id)))
````

`server/internal/modules/issue/app/ports.go`（修改，2 处）：

````old server/internal/modules/issue/app/ports.go
// IssueReader reads a work item, in the transaction ctx carries, or
````
````new server/internal/modules/issue/app/ports.go
// IssueReader reads work items, in the transaction ctx carries, or
````
````old server/internal/modules/issue/app/ports.go
	GetIssue(ctx context.Context, id uuid.UUID) (i domain.Issue, found bool, err error)
````
````new server/internal/modules/issue/app/ports.go
	GetIssue(ctx context.Context, id uuid.UUID) (i domain.Issue, found bool, err error)
	// GetIssues reads the undeleted work items among ids, archived or
	// not, each as GetIssue reads it, in id order.
	GetIssues(ctx context.Context, ids []uuid.UUID) ([]domain.Issue, error)
	// SubIssueCounts counts the undeleted, unarchived direct sub-items of
	// each of the work items ids that has any (M4 design 3.9).
	SubIssueCounts(ctx context.Context, actor uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]SubIssueCount, error)
}

// SubIssueCount is a work item's number of undeleted, unarchived direct
// sub-items, as IssueReader counts them for an account: All of them, and
// Own, those the account created.
type SubIssueCount struct {
	All, Own int
````

`server/internal/modules/issue/app/seen.go`（新文件，97 行）：

````file server/internal/modules/issue/app/seen.go
package app

import (
	"context"
	"fmt"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// grants are the caller's grants on the projects of the work items an
// answer shows, by project, as their decisions answered them.
type grants map[uuid.UUID]shared.Grant

// seen is items as the caller actor sees them (M4 design 3.4, 3.9), by
// his grants on their projects: each with the number of its undeleted,
// unarchived direct sub-items he sees, every one, or those he created
// when the guest rule hides others' from him in its project
// (Grant.OwnOnly), read in one count; and with its parent only when he
// sees it (hideParents). A work item of a project without a grant in
// grants is an error: every answer is decided first.
func (l Locks) seen(ctx context.Context, actor uuid.UUID, g grants, items ...domain.Issue) ([]domain.Issue, error) {
	if len(items) == 0 {
		return nil, nil
	}
	ids := make([]uuid.UUID, len(items))
	for n, i := range items {
		ids[n] = i.ID
	}
	counts, err := l.issues.SubIssueCounts(ctx, actor, ids)
	if err != nil {
		return nil, err
	}
	out := slices.Clone(items)
	var guarded []int
	for n, i := range out {
		grant, ok := g[i.ProjectID]
		if !ok {
			return nil, fmt.Errorf("no grant on project %s, of work item %s", i.ProjectID, i.ID)
		}
		out[n].SubIssuesCount = counts[i.ID].All
		if grant.OwnOnly {
			out[n].SubIssuesCount = counts[i.ID].Own
		}
		if grant.OwnOnly && i.ParentID != nil {
			guarded = append(guarded, n)
		}
	}
	if err := l.hideParents(ctx, actor, out, guarded); err != nil {
		return nil, err
	}
	return out, nil
}

// hideParents answers as none (nil) the parent of each work item at the
// places guarded in items that the guest rule hides from the caller actor
// (M4 design 3.4): their grants hide others' work items from him
// (Grant.OwnOnly), so their parents are read, in one read, where the count
// is read, and one he did not create (domain.Issue.HiddenFrom), or one the
// read does not find, is hidden. Nothing is read when no place is guarded,
// as under a grant that shows him every work item. A failing read is its
// error.
func (l Locks) hideParents(ctx context.Context, actor uuid.UUID, items []domain.Issue, guarded []int) error {
	if len(guarded) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(guarded))
	for k, n := range guarded {
		ids[k] = *items[n].ParentID
	}
	parents, err := l.issues.GetIssues(ctx, slices.Compact(slices.SortedFunc(slices.Values(ids), uuid.UUID.Compare)))
	if err != nil {
		return err
	}
	shown := make(map[uuid.UUID]bool, len(parents))
	for _, p := range parents {
		shown[p.ID] = !p.HiddenFrom(actor, true)
	}
	for _, n := range guarded {
		if !shown[*items[n].ParentID] {
			items[n].ParentID = nil
		}
	}
	return nil
}

// seenOne is the work item i as the caller actor sees it (seen), by his
// grant on its project.
func (l Locks) seenOne(ctx context.Context, actor uuid.UUID, grant shared.Grant, i domain.Issue) (domain.Issue, error) {
	out, err := l.seen(ctx, actor, grants{i.ProjectID: grant}, i)
	if err != nil {
		return domain.Issue{}, err
	}
	return out[0], nil
}
````

`server/internal/modules/issue/app/seen_test.go`（新文件，194 行）：

````file server/internal/modules/issue/app/seen_test.go
package app_test

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// counted is the call that counts the sub-items of the work items ids for
// alice, whose answer shows them (Locks.seen), as her write's transaction
// logs it; a read outside any logs it " outside tx" after.
func counted(ids ...uuid.UUID) string {
	return line("SubIssueCounts", alice, ids)
}

// counts are the sub-items alice's work item id has, three, one of them
// hers, and those of another work item, which no answer of it shows.
func counts(id uuid.UUID) map[uuid.UUID]app.SubIssueCount {
	return map[uuid.UUID]app.SubIssueCount{id: {All: 3, Own: 1}, uuid.NewV7(): {All: 5, Own: 5}}
}

// adminGuest is the grant of a workspace admin who is only a guest of web,
// which does not show its guests every work item: as an admin he passes
// each of its rules (Grant.ProjectAdmin), and as its guest he sees only
// his own work items (Grant.OwnOnly).
var adminGuest = shared.Grant{ProjectRole: shared.RoleGuest, ProjectAdmin: true, OwnOnly: true}

// A read of a work item, getIssue's and getIssueByIdentifier's, answers the
// number of its sub-items the caller sees (M4 design 3.4, 3.9), counted
// after the decision, outside any transaction, by its grant: every one,
// or only his own when the guest rule hides others' from him
// (Grant.OwnOnly), as from alice, a guest of web who reads her own work
// item. Every one when web shows its guests every work item.
func TestAReadCountsTheSubItemsTheCallerSees(t *testing.T) {
	for _, tt := range []struct {
		name  string
		grant shared.Grant
		want  int
	}{{"an admin", shared.Grant{ProjectRole: shared.RoleAdmin}, 3}, {"a guest who sees his own", ownOnly, 1},
		{"a guest who sees every work item", everyItem, 3}} {
		get, w, id := newGet()
		w.issues.issue.By, w.issues.counts, w.auth.grants = alice, counts(id), []shared.Grant{tt.grant}
		got, err := get.Execute(as(alice), id)
		if err != nil || got.SubIssuesCount != tt.want || w.log.calls[len(w.log.calls)-1] != counted(id)+" outside tx" {
			t.Errorf("%s: getIssue = %+v, %v after %q; want %d sub-items, counted last", tt.name, got, err, w.log.calls, tt.want)
		}
		byIdentifier, b, bid := newGetByIdentifier()
		b.issues.issue.By, b.issues.counts, b.auth.grants = alice, counts(bid), []shared.Grant{{}, tt.grant}
		got, err = byIdentifier.Execute(as(alice), "acme", "WEB-7")
		if err != nil || got.SubIssuesCount != tt.want || b.log.calls[len(b.log.calls)-1] != counted(bid)+" outside tx" {
			t.Errorf("%s: getIssueByIdentifier = %+v, %v after %q; want %d sub-items, counted last", tt.name, got, err, b.log.calls, tt.want)
		}
	}
}

// An answer names the work item's parent only when the caller sees it (M4
// design 3.4, 3.9): alice reads her own work item 7, under parent. When the
// guest rule hides others' work items from her (Grant.OwnOnly), the parent
// is read after the count, as the count is, outside any transaction, in
// one read, and one carol made, or one the read does not find, is answered
// as none; her own is named. When she sees every work item it is named,
// and nothing more is read. Her change of the work item as a workspace
// admin who is only a guest of web reads the parent in the transaction,
// after the count. A failing read of the parent is the answer's error.
func TestAnAnswerNamesTheParentTheCallerSees(t *testing.T) {
	parent := uuid.NewV7()
	read := line("GetIssues", []uuid.UUID{parent})
	for _, tt := range []struct {
		name  string
		grant shared.Grant
		by    *uuid.UUID // the parent's creator; nil when the read does not find it
		want  *uuid.UUID
		reads bool
	}{
		{"carol's, to a guest who sees her own", ownOnly, &carol, nil, true},
		{"her own, to a guest who sees her own", ownOnly, &alice, &parent, true},
		{"not found, to a guest who sees her own", ownOnly, nil, nil, true},
		{"carol's, to a guest who sees every work item", everyItem, &carol, &parent, false},
	} {
		get, w, id := newGet()
		w.issues.issue.By, w.issues.issue.ParentID, w.auth.grants = alice, &parent, []shared.Grant{tt.grant}
		if tt.by != nil {
			w.issues.others = map[uuid.UUID]domain.Issue{parent: {ID: parent, WorkspaceID: acme, ProjectID: web, CreatedByID: tt.by}}
		}
		got, err := get.Execute(as(alice), id)
		want := []string{counted(id) + " outside tx"}
		if tt.reads {
			want = append(want, read+" outside tx")
		}
		if calls := w.log.calls[len(w.log.calls)-len(want):]; err != nil || !reflect.DeepEqual(got.ParentID, tt.want) ||
			!slices.Equal(calls, want) {
			t.Errorf("%s: Execute() = %v, %v, its last calls %q; want the parent %v, the last calls %q", tt.name, got.ParentID, err, calls,
				tt.want, want)
		}
	}

	_, w, id := newUpdate()
	w.issues.issue.By, w.issues.issue.ParentID = alice, &parent
	w.issues.others = map[uuid.UUID]domain.Issue{parent: {ID: parent, WorkspaceID: acme, ProjectID: web, CreatedByID: &carol}}
	w.auth.grants = []shared.Grant{adminGuest, adminGuest}
	got, err := updateOver(w).Execute(as(alice), id, domain.IssuePatch{Name: name("Mend")})
	if calls := w.log.calls[len(w.log.calls)-2:]; err != nil || got.ParentID != nil || !slices.Equal(calls, []string{counted(id), read}) {
		t.Errorf("a change: Execute() = %v, %v, its last calls %q; want no parent, the last calls %q", got.ParentID, err, calls,
			[]string{counted(id), read})
	}

	get, w, id := newGet()
	w.issues.issue.By, w.issues.issue.ParentID, w.auth.grants, w.log.fail = alice, &parent, []shared.Grant{ownOnly}, "GetIssues"
	if got, err := get.Execute(as(alice), id); !errors.Is(err, errDisk) || !reflect.DeepEqual(got, domain.Issue{}) {
		t.Errorf("Execute() with the read of the parent failing = %+v, %v; want %v", got, err, errDisk)
	}
}

// A write's answer of the work item it names, updateIssue's and
// archiveIssue's, shows the number of its sub-items the caller sees (M4
// design 3.4, 3.9), counted by the grant of the decision under the lock,
// in the transaction, last: after the work item read back, or after the
// decision when the write writes nothing: alice's own work item 7 changed,
// changed in nothing, archived, archived again, unarchived, unarchived
// again. Every one as an admin of web, or only her own as the guest rule
// has it for a workspace admin who is only a guest of web (adminGuest). A
// count that fails is each write's failure, as itself, and its last call.
func TestAWritesAnswerCountsTheSubItemsTheCallerSees(t *testing.T) {
	for _, tt := range []struct {
		name    string
		prepare func(w *world)
		write   func(w *world, id uuid.UUID) (domain.Issue, error)
		writes  bool
	}{
		{"changed", func(*world) {}, func(w *world, id uuid.UUID) (domain.Issue, error) {
			return updateOver(w).Execute(as(alice), id, domain.IssuePatch{Name: name("Mend")})
		}, true},
		{"changed in nothing", func(*world) {}, func(w *world, id uuid.UUID) (domain.Issue, error) {
			return updateOver(w).Execute(as(alice), id, domain.IssuePatch{Name: name("Fix")})
		}, false},
		{"archived", func(w *world) { w.issues.issue.StateID = done.ID }, func(w *world, id uuid.UUID) (domain.Issue, error) {
			archive, _ := archiveOver(w)
			return archive.Execute(as(alice), id)
		}, true},
		{"archived again", func(w *world) { w.issues.archivedAt = &earlier }, func(w *world, id uuid.UUID) (domain.Issue, error) {
			archive, _ := archiveOver(w)
			return archive.Execute(as(alice), id)
		}, false},
		{"unarchived", func(w *world) { w.issues.archivedAt = &earlier }, func(w *world, id uuid.UUID) (domain.Issue, error) {
			_, unarchive := archiveOver(w)
			return unarchive.Execute(as(alice), id)
		}, true},
		{"unarchived again", func(*world) {}, func(w *world, id uuid.UUID) (domain.Issue, error) {
			_, unarchive := archiveOver(w)
			return unarchive.Execute(as(alice), id)
		}, false},
	} {
		for _, view := range []struct {
			name  string
			grant shared.Grant
			want  int
		}{{"an admin", shared.Grant{ProjectRole: shared.RoleAdmin}, 3}, {"an admin who is a guest", adminGuest, 1}} {
			t.Run(tt.name+", "+view.name, func(t *testing.T) {
				_, w, id := newUpdate()
				w.issues.issue.By, w.issues.counts = alice, counts(id)
				w.auth.grants = []shared.Grant{view.grant, view.grant}
				tt.prepare(w)
				got, err := tt.write(w, id)
				before := line("GetIssue", id) // the read back
				if !tt.writes {
					before = line("Authorize", alice) // the decision under the lock
				}
				n := len(w.log.calls)
				if err != nil || got.SubIssuesCount != view.want || n < 2 || w.log.calls[n-1] != counted(id) ||
					!strings.HasPrefix(w.log.calls[n-2], before) {
					t.Errorf("Execute() = %+v, %v after %q; want %d sub-items, counted last, after %s", got, err, w.log.calls, view.want,
						before)
				}
				if !tt.writes && slices.Contains(w.log.calls, "Now") {
					t.Errorf("calls %q; want nothing written", w.log.calls)
				}
			})
		}
		t.Run(tt.name+", the count failing", func(t *testing.T) {
			_, w, id := newUpdate()
			w.log.fail = "SubIssueCounts"
			tt.prepare(w)
			_, err := tt.write(w, id)
			wantStoppedAt(t, w, err, "SubIssueCounts")
		})
	}
}
````

`server/internal/modules/issue/app/update_issue.go`（修改，3 处）：

````old server/internal/modules/issue/app/update_issue.go
// back in the transaction.
````
````new server/internal/modules/issue/app/update_issue.go
// back in the transaction, as the caller sees it (Locks.seen), the work
// item as read when nothing is written.
````
````old server/internal/modules/issue/app/update_issue.go
			updated = h.issue
			return nil
````
````new server/internal/modules/issue/app/update_issue.go
			updated, err = u.d.Locks.seenOne(ctx, actor.UserID, h.grant, h.issue)
			return err
````
````old server/internal/modules/issue/app/update_issue.go
		updated, err = readBack(ctx, u.d.Issues, id)
````
````new server/internal/modules/issue/app/update_issue.go
		if updated, err = readBack(ctx, u.d.Issues, id); err != nil {
			return err
		}
		updated, err = u.d.Locks.seenOne(ctx, actor.UserID, h.grant, updated)
````

`server/internal/modules/issue/app/update_issue_test.go`（修改，8 处）：

````old server/internal/modules/issue/app/update_issue_test.go
	uc := app.NewUpdateIssue(app.UpdateIssueDeps{Locks: w.locks, Issues: w.issues, States: w.project, Labels: w.project, Members: w.project,
		Profiles: w.profiles, Clock: clockAt{at: clockNow, log: w.log}})
	return uc, w, id
````
````new server/internal/modules/issue/app/update_issue_test.go
	return updateOver(w), w, id
}

// updateOver is updateIssue over w's ports.
func updateOver(w *world) *app.UpdateIssue {
	return app.NewUpdateIssue(app.UpdateIssueDeps{Locks: w.locks, Issues: w.issues, States: w.project, Labels: w.project, Members: w.project,
		Profiles: w.profiles, Clock: clockAt{at: clockNow, log: w.log}})
````
````old server/internal/modules/issue/app/update_issue_test.go
// An update, in the order of M4 design 3.5 and 6.7: Locks' calls
// (locked); then the new state, the roles of the accounts added and the
// labels added and removed, checked or named; the clock; the work item's
// fields; the assignees removed and added, the labels the same, each in id
// order; the accounts added subscribed; the display names of the accounts
// added and removed; the activity: the name, the priority, the state by
// its names, each label added and removed, each account added and removed
````
````new server/internal/modules/issue/app/update_issue_test.go
// An update, in the order of M4 design 3.5 and 6.7: Locks' calls (locked);
// then the new state, the roles of the accounts added and the labels added
// and removed, checked or named; the clock; the work item's fields; the
// assignees removed and added, the labels the same, each in id order; the
// accounts added subscribed; the display names of the accounts added and
// removed; the activity: the name, the priority, the state by its names,
// each label added and removed, each account added and removed
````
````old server/internal/modules/issue/app/update_issue_test.go
// the work item read back. The name is stored without its surrounding
// space; in Done the work item is completed at the moment. out, who is no
// member of web, stays assigned: only the accounts added, carol and dave,
// are checked, and each of them, not only the first, is assigned,
// subscribed and in the activity. Docs, which the work item has and
// keeps, is neither read nor written again (M4 design 3.7): Bug's row
// alone is removed and UI's alone added.
````
````new server/internal/modules/issue/app/update_issue_test.go
// the work item read back, its sub-items counted (Locks.seen). The name is
// stored without its surrounding space; in Done the work item is completed
// at the moment. out, who is no member of web, stays assigned: only the
// accounts added, carol and dave, are checked, and each of them, not only
// the first, is assigned, subscribed and in the activity. Docs, which the
// work item has and keeps, is neither read nor written again (M4 design
// 3.7): Bug's row alone is removed and UI's alone added.
````
````old server/internal/modules/issue/app/update_issue_test.go
		line("DisplayNames", []uuid.UUID{carol, dave, bob}), line("InsertActivities", 8), line("GetIssue", id))
````
````new server/internal/modules/issue/app/update_issue_test.go
		line("DisplayNames", []uuid.UUID{carol, dave, bob}), line("InsertActivities", 8), line("GetIssue", id), counted(id))
````
````old server/internal/modules/issue/app/update_issue_test.go
// they were.
````
````new server/internal/modules/issue/app/update_issue_test.go
// they were, its sub-items counted.
````
````old server/internal/modules/issue/app/update_issue_test.go
			if calls := locked(id); !slices.Equal(w.log.calls, calls) || w.issues.changed != nil {
````
````new server/internal/modules/issue/app/update_issue_test.go
			if calls := append(locked(id), counted(id)); !slices.Equal(w.log.calls, calls) || w.issues.changed != nil {
````
````old server/internal/modules/issue/app/update_issue_test.go
// the precheck's have passed: the transaction rolls back whatever was
// written. What cannot be is an error too, before any write: a work item
// whose state is not its project's; a label it is removed from that is not
// its project's; an account without a display name, before the activity;
// and a work item not there to read back after its write.
````
````new server/internal/modules/issue/app/update_issue_test.go
// the precheck's have passed, and the count of its answer's sub-items,
// the last: the transaction rolls back whatever was written. What cannot
// be is an error too, before any write: a work item whose state is not
// its project's; a label it is removed from that is not its project's; an
// account without a display name, before the activity; and a work item
// not there to read back after its write.
````
````old server/internal/modules/issue/app/update_issue_test.go
		"InsertAssignees", "RemoveLabels", "InsertLabels", "InsertSubscribers", "DisplayNames", "InsertActivities"} {
````
````new server/internal/modules/issue/app/update_issue_test.go
		"InsertAssignees", "RemoveLabels", "InsertLabels", "InsertSubscribers", "DisplayNames", "InsertActivities", "SubIssueCounts"} {
````

`server/internal/modules/issue/domain/issue.go`（修改，2 处）：

````old server/internal/modules/issue/domain/issue.go
// undeleted labels' rows, in the accounts' and the labels' id order. The
// fields of later phases come with them (M4 design 12, constraint 5).
````
````new server/internal/modules/issue/domain/issue.go
// undeleted labels' rows, in the accounts' and the labels' id order, and
// the caller's view of it. The fields of later phases come with them (M4
// design 12, constraint 5).
````
````old server/internal/modules/issue/domain/issue.go
	UpdatedAt   time.Time
````
````new server/internal/modules/issue/domain/issue.go
	UpdatedAt   time.Time
	// The caller's view (M4 design 3.9): the number of its undeleted,
	// unarchived direct sub-items that he sees, by his grant on its
	// project. A read of the store leaves it 0; it is counted when an
	// answer is built (the app's Locks.seen), and a creation answers 0, a
	// new work item having none.
	SubIssuesCount int
````

- [ ] **Step 3: 存储和 HTTP**

`server/internal/modules/issue/adapter/http/issues.go`（修改，1 处）：

````old server/internal/modules/issue/adapter/http/issues.go
		CreatedByID: orNull(i.CreatedByID), UpdatedByID: orNull(i.UpdatedByID), CreatedAt: i.CreatedAt, UpdatedAt: i.UpdatedAt}
````
````new server/internal/modules/issue/adapter/http/issues.go
		SubIssuesCount: i.SubIssuesCount, CreatedByID: orNull(i.CreatedByID), UpdatedByID: orNull(i.UpdatedByID), CreatedAt: i.CreatedAt,
		UpdatedAt: i.UpdatedAt}
````

`server/internal/modules/issue/adapter/http/issues_test.go`（修改，3 处）：

````old server/internal/modules/issue/adapter/http/issues_test.go
		CreatedByID: &aliceID, UpdatedByID: &bobID, CreatedAt: created, UpdatedAt: updated}
````
````new server/internal/modules/issue/adapter/http/issues_test.go
		SubIssuesCount: 2, CreatedByID: &aliceID, UpdatedByID: &bobID, CreatedAt: created, UpdatedAt: updated}
````
````old server/internal/modules/issue/adapter/http/issues_test.go
		`"state_id":"0199a2b4-0000-7000-8000-0000000000c1","target_date":"2026-10-19","updated_at":"2026-10-17T10:01:00.123456Z",` +
````
````new server/internal/modules/issue/adapter/http/issues_test.go
		`"state_id":"0199a2b4-0000-7000-8000-0000000000c1","sub_issues_count":2,"target_date":"2026-10-19",` +
		`"updated_at":"2026-10-17T10:01:00.123456Z",` +
````
````old server/internal/modules/issue/adapter/http/issues_test.go
		`"state_id":"0199a2b4-0000-7000-8000-0000000000c1","target_date":null,"updated_at":"2026-10-17T10:00:00.123456Z","updated_by_id":null,` +
````
````new server/internal/modules/issue/adapter/http/issues_test.go
		`"state_id":"0199a2b4-0000-7000-8000-0000000000c1","sub_issues_count":0,"target_date":null,"updated_at":"2026-10-17T10:00:00.123456Z",` +
		`"updated_by_id":null,` +
````

`server/internal/modules/issue/adapter/postgres/issues.go`（修改，2 处）：

````old server/internal/modules/issue/adapter/postgres/issues.go
		return domain.Issue{}, false, fmt.Errorf("read work item %s: %w", id, err)
	}
````
````new server/internal/modules/issue/adapter/postgres/issues.go
		return domain.Issue{}, false, fmt.Errorf("read work item %s: %w", id, err)
	}
	return issueOf(r), true, nil
}

// GetIssues reads the undeleted work items among ids, archived or not,
// each as GetIssue reads it, in id order (M4 design 3.9).
func (s *Store) GetIssues(ctx context.Context, ids []uuid.UUID) ([]domain.Issue, error) {
	rows, err := s.queries(ctx).GetIssues(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("read %d work items: %w", len(ids), err)
	}
	issues := make([]domain.Issue, len(rows))
	for n, r := range rows {
		issues[n] = issueOf(gen.GetIssueRow(r))
	}
	return issues, nil
}

// issueOf is the work item a read answers as r: each read of work items
// has GetIssue's columns.
func issueOf(r gen.GetIssueRow) domain.Issue {
````
````old server/internal/modules/issue/adapter/postgres/issues.go
	}, true, nil
````
````new server/internal/modules/issue/adapter/postgres/issues.go
	}
````

`server/internal/modules/issue/adapter/postgres/sub_issues.go`（新文件，25 行）：

````file server/internal/modules/issue/adapter/postgres/sub_issues.go
package postgresadapter

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// SubIssueCounts counts the undeleted, unarchived direct sub-items of each
// of the work items ids that has any, all of them and those the account
// actor created (M4 design 3.9).
func (s *Store) SubIssueCounts(ctx context.Context, actor uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]app.SubIssueCount, error) {
	rows, err := s.queries(ctx).SubIssueCounts(ctx, gen.SubIssueCountsParams{Actor: actor, Ids: ids})
	if err != nil {
		return nil, fmt.Errorf("count the sub-items of %d work items: %w", len(ids), err)
	}
	counts := make(map[uuid.UUID]app.SubIssueCount, len(rows))
	for _, r := range rows {
		counts[r.ParentID] = app.SubIssueCount{All: int(r.AllCount), Own: int(r.OwnCount)}
	}
	return counts, nil
}
````

`server/internal/modules/issue/adapter/postgres/sub_issues_test.go`（新文件，146 行）：

````file server/internal/modules/issue/adapter/postgres/sub_issues_test.go
package postgresadapter_test

import (
	"context"
	"errors"
	"maps"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// A work item's sub-items as SubIssueCounts counts them (M4 design 3.9):
// its undeleted, unarchived direct sub-items, all of them and those the
// account given created, whoever changed them last. In web, p has a and
// b, alice's, c, bob's, changed last by alice, d, bob's, archived, and e,
// alice's, deleted; a has g under it, which is not p's. q has r, bob's; s
// has none. For alice, p has 3, 2 of them hers, and q 1, none hers; for
// bob, p has 3, 1 his, and q 1, his. s, and a work item not there, have
// none, and no count.
func TestSubIssueCounts(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	p, a, b, c, d, e, g, q, r, sub := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(),
		uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	for n, made := range []struct {
		id uuid.UUID
		by uuid.UUID
	}{{p, alice}, {a, alice}, {b, alice}, {c, bob}, {d, bob}, {e, alice}, {g, alice}, {q, alice}, {r, bob}, {sub, alice}} {
		newIssue(t, pool, web, made.id, n+1, made.by)
	}
	for _, link := range [][2]uuid.UUID{{a, p}, {b, p}, {c, p}, {d, p}, {e, p}, {g, a}, {r, q}} {
		exec(t, pool, "UPDATE issues SET parent_id = $2 WHERE id = $1", link[0], link[1])
	}
	exec(t, pool, "UPDATE issues SET archived_at = $2 WHERE id = $1", d, now)
	exec(t, pool, "UPDATE issues SET deleted_at = $2 WHERE id = $1", e, now)
	exec(t, pool, "UPDATE issues SET updated_by_id = $2 WHERE id = $1", c, alice)

	ctx := context.Background()
	asked := []uuid.UUID{p, q, sub, uuid.NewV7()}
	for _, tt := range []struct {
		name  string
		actor uuid.UUID
		want  map[uuid.UUID]app.SubIssueCount
	}{
		{"alice", alice, map[uuid.UUID]app.SubIssueCount{p: {All: 3, Own: 2}, q: {All: 1}}},
		{"bob", bob, map[uuid.UUID]app.SubIssueCount{p: {All: 3, Own: 1}, q: {All: 1, Own: 1}}},
	} {
		if got, err := s.SubIssueCounts(ctx, tt.actor, asked); err != nil || !maps.Equal(got, tt.want) {
			t.Errorf("SubIssueCounts() for %s = %v, %v; want %v", tt.name, got, err, tt.want)
		}
	}
	cancelled, stop := context.WithCancel(ctx)
	stop()
	if _, err := s.SubIssueCounts(cancelled, alice, asked); !errors.Is(err, context.Canceled) {
		t.Errorf("SubIssueCounts() on a cancelled context = %v; want context.Canceled", err)
	}
}

// family is a work item p of web and the work items under it, drawn in
// this order: a, alice's, assigned to her and labelled; b, bob's, made an
// hour before the others, though drawn after a; x, bob's; c, alice's,
// archived; d, alice's, deleted; g, under a, not under p. q has none
// under it.
type family struct {
	p, a, b, x, c, d, g, q uuid.UUID
}

// newFamily stores a family in web, of alice and bob.
func newFamily(t *testing.T, s *postgresadapter.Store, pool *pgxpool.Pool, web project, alice, bob uuid.UUID) family {
	t.Helper()
	f := family{p: uuid.NewV7(), a: uuid.NewV7(), b: uuid.NewV7(), x: uuid.NewV7(), c: uuid.NewV7(), d: uuid.NewV7(), g: uuid.NewV7(),
		q: uuid.NewV7()}
	for n, made := range []struct{ id, by uuid.UUID }{{f.p, alice}, {f.a, alice}, {f.b, bob}, {f.x, bob}, {f.c, alice}, {f.d, alice},
		{f.g, alice}, {f.q, alice}} {
		newIssue(t, pool, web, made.id, n+1, made.by)
	}
	for _, link := range [][2]uuid.UUID{{f.a, f.p}, {f.b, f.p}, {f.x, f.p}, {f.c, f.p}, {f.d, f.p}, {f.g, f.a}} {
		exec(t, pool, "UPDATE issues SET parent_id = $2 WHERE id = $1", link[0], link[1])
	}
	exec(t, pool, "UPDATE issues SET created_at = $2 WHERE id = $1", f.b, now.Add(-time.Hour))
	exec(t, pool, "UPDATE issues SET archived_at = $2 WHERE id = $1", f.c, now)
	exec(t, pool, "UPDATE issues SET deleted_at = $2 WHERE id = $1", f.d, now)
	ctx := context.Background()
	if err := s.InsertAssignees(ctx, linkRows(web, f.a, alice, now, alice)); err != nil {
		t.Fatal(err)
	}
	if err := s.InsertLabels(ctx, linkRows(web, f.a, alice, now, newLabel(t, pool, web, "Bug"))); err != nil {
		t.Fatal(err)
	}
	return f
}

// read are the work items ids as GetIssue reads each, in their order.
func read(t *testing.T, s *postgresadapter.Store, ids ...uuid.UUID) []domain.Issue {
	t.Helper()
	out := []domain.Issue{}
	for _, id := range ids {
		i, found, err := s.GetIssue(context.Background(), id)
		if err != nil || !found {
			t.Fatalf("GetIssue(%s) = %v, %v", id, found, err)
		}
		out = append(out, i)
	}
	return out
}

// The work items GetIssues reads (M4 design 3.9): the undeleted ones among
// the ids given, archived ones too, each as GetIssue reads it, in id
// order, whatever the order given or stored; none for none. a, the first
// of them by id, moves under q after the others are stored, which gives
// its row a new place, the last of them both in the table and in the
// index on (project_id, state_id): a read without its ORDER BY answers it
// last whether the database scans the table or reads that index.
func TestGetIssues(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	f := newFamily(t, s, pool, web, alice, bob)
	exec(t, pool, "UPDATE issues SET parent_id = $2 WHERE id = $1", f.a, f.q)
	ctx := context.Background()
	for _, tt := range []struct {
		name string
		ids  []uuid.UUID
		want []domain.Issue
	}{
		{"some", []uuid.UUID{f.x, f.d, uuid.NewV7(), f.c, f.a}, read(t, s, f.a, f.x, f.c)},
		{"none", nil, read(t, s)},
	} {
		if got, err := s.GetIssues(ctx, tt.ids); err != nil || !reflect.DeepEqual(got, tt.want) {
			t.Errorf("GetIssues(%s) = %+v, %v; want %+v", tt.name, got, err, tt.want)
		}
	}
	cancelled, stop := context.WithCancel(ctx)
	stop()
	if got, err := s.GetIssues(cancelled, []uuid.UUID{f.a}); !errors.Is(err, context.Canceled) || got != nil {
		t.Errorf("GetIssues() on a cancelled context = %+v, %v; want context.Canceled", got, err)
	}
}
````

- [ ] **Step 4: 组合**

`server/internal/bootstrap/issue_parents_test.go`（修改，2 处）：

````old server/internal/bootstrap/issue_parents_test.go
	"slices"
````
````new server/internal/bootstrap/issue_parents_test.go
	"slices"
	"strconv"
````
````old server/internal/bootstrap/issue_parents_test.go
		}
	}
}

````
````new server/internal/bootstrap/issue_parents_test.go
		}
	}
}

// A parent the guest rule hides from the caller is not in the answers of a
// work item under it (M4 design 3.4, 3.9): walt, acme's admin and Web's
// guest (adminGuest), sees only his own work items in Web, and not alice's
// Up (his GET of it is 404), under which alice puts his His. Each answer
// of His to walt, his read of it by its id and by its identifier and his
// change of its name, is alice's answer of it then, compared whole, with
// parent_id null. Below, under his own Mine, he is answered under Mine;
// bob, Web's member, who sees every work item, His under Up.
func TestAHiddenParentIsNotInTheAnswer(t *testing.T) {
	w := newIssueWorld(t)
	w.adminGuest(t)
	up := w.newWorkItemIn(t, w.web, `{"name":"Up"}`)
	var his storedIssue
	decodeAnswer(t, w.send(t, "walt", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/issues", `{"name":"His"}`,
		http.StatusCreated), &his)
	path := "/api/v0/issues/" + his.ID.String()
	w.send(t, "alice", http.MethodPatch, path, `{"parent_id":"`+up.String()+`"}`, http.StatusOK)
	w.send(t, "walt", http.MethodGet, "/api/v0/issues/"+up.String(), "", http.StatusNotFound)
	for _, step := range []struct{ name, method, path, body string }{
		{"read", http.MethodGet, path, ""},
		{"read by its identifier", http.MethodGet, "/api/v0/workspaces/acme/issues/WEB-" + strconv.Itoa(his.SequenceID), ""},
		{"changed", http.MethodPatch, path, `{"name":"Mended"}`},
	} {
		var got, want map[string]any
		decodeAnswer(t, w.send(t, "walt", step.method, step.path, step.body, http.StatusOK), &got)
		decodeAnswer(t, w.send(t, "alice", http.MethodGet, path, "", http.StatusOK), &want)
		if want["parent_id"] != up.String() {
			t.Fatalf("%s: alice is answered His under %v, want Up, %s", step.name, want["parent_id"], up)
		}
		want["parent_id"] = nil
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s, by walt: %v; want alice's answer with parent_id null, %v", step.name, got, want)
		}
	}
	mine := w.newWorkItemBy(t, "walt", w.web, `{"name":"Mine"}`)
	below := w.newWorkItemBy(t, "walt", w.web, underParent("Below", mine))
	for _, tt := range []struct {
		by   string
		item uuid.UUID
		want *uuid.UUID
	}{{"walt", below, &mine}, {"bob", his.ID, &up}} {
		var got storedIssue
		decodeAnswer(t, w.send(t, tt.by, http.MethodGet, "/api/v0/issues/"+tt.item.String(), "", http.StatusOK), &got)
		if !reflect.DeepEqual(got.ParentID, tt.want) {
			t.Errorf("%s's read of %s names its parent %v, want %v", tt.by, tt.item, got.ParentID, tt.want)
		}
	}
}

````

`server/internal/bootstrap/issue_sub_issue_count_test.go`（新文件，81 行）：

````file server/internal/bootstrap/issue_sub_issue_count_test.go
package bootstrap

import (
	"net/http"
	"strconv"
	"testing"
	"uuid"
)

// subItems is the number of sub-items an answer shows of the work item it
// is.
func subItems(t *testing.T, answer string) int {
	t.Helper()
	var i struct {
		SubIssuesCount *int `json:"sub_issues_count"`
	}
	decodeAnswer(t, answer, &i)
	if i.SubIssuesCount == nil {
		t.Fatalf("the answer %s has no sub_issues_count", answer)
	}
	return *i.SubIssuesCount
}

// Each answer of a work item shows the number of its sub-items the caller
// sees (M4 design 3.4, 3.9), as bootstrap wires it: its undeleted,
// unarchived direct sub-items, those he created when the guest rule hides
// others' from him, whoever changed them last. walt, acme's admin and
// Web's guest (adminGuest), makes His; alice puts hers, Part, under it,
// walt his, Mine, which alice renames, and alice an archived one and a
// deleted one; Low, under Part, is not His. At each step walt, then alice,
// reads His, by its id and by its identifier, changes it, changes it in
// nothing, moves it into Done, archives it twice and unarchives it twice,
// the second of a step finding it done: each answer shows walt 1 sub-item,
// Mine, his though alice changed it last, and alice, who sees them all, 2.
// A new work item has none.
func TestEachAnswerCountsTheSubItemsTheCallerSees(t *testing.T) {
	w := newIssueWorld(t)
	w.adminGuest(t)
	var created struct {
		ID         uuid.UUID `json:"id"`
		SequenceID int       `json:"sequence_id"`
	}
	decodeAnswer(t, w.send(t, "walt", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/issues", `{"name":"His"}`, http.StatusCreated),
		&created)
	his := created.ID
	part := w.newWorkItemIn(t, w.web, underParent("Part", his))
	w.newWorkItemIn(t, w.web, underParent("Low", part))
	mine := w.newWorkItemBy(t, "walt", w.web, underParent("Mine", his))
	w.send(t, "alice", http.MethodPatch, "/api/v0/issues/"+mine.String(), `{"name":"Mine, renamed"}`, http.StatusOK)
	shelved := w.newWorkItemIn(t, w.web, `{"name":"Shelved","state_id":"`+w.done.String()+`","parent_id":"`+his.String()+`"}`)
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+shelved.String()+"/archive", "", http.StatusOK)
	gone := w.newWorkItemIn(t, w.web, underParent("Gone", his))
	w.send(t, "alice", http.MethodDelete, "/api/v0/issues/"+gone.String(), "", http.StatusNoContent)

	path := "/api/v0/issues/" + his.String()
	identifier := "/api/v0/workspaces/acme/issues/WEB-" + strconv.Itoa(created.SequenceID)
	for _, step := range []struct{ name, method, path, body string }{
		{"read", http.MethodGet, path, ""},
		{"read by its identifier", http.MethodGet, identifier, ""},
		{"changed", http.MethodPatch, path, `{"name":"Mend"}`},
		{"changed in nothing", http.MethodPatch, path, `{"name":"Mend"}`},
		{"moved into Done", http.MethodPatch, path, `{"state_id":"` + w.done.String() + `"}`},
		{"archived", http.MethodPost, path + "/archive", ""},
		{"archived again", http.MethodPost, path + "/archive", ""},
		{"unarchived", http.MethodPost, path + "/unarchive", ""},
		{"unarchived again", http.MethodPost, path + "/unarchive", ""},
	} {
		for _, by := range []struct {
			name string
			want int
		}{{"walt", 1}, {"alice", 2}} {
			if got := subItems(t, w.send(t, by.name, step.method, step.path, step.body, http.StatusOK)); got != by.want {
				t.Errorf("%s, by %s: %d sub-items, want %d", step.name, by.name, got, by.want)
			}
		}
	}
	answer := w.send(t, "walt", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/issues", `{"name":"New"}`, http.StatusCreated)
	if got := subItems(t, answer); got != 0 {
		t.Errorf("a new work item has %d sub-items, want 0", got)
	}
}
````

`server/internal/bootstrap/issue_writes_test.go`（修改，2 处）：

````old server/internal/bootstrap/issue_writes_test.go
// and walt is refused (403); and she reads it, without a transaction. She
// makes Part under it, its parent read under the lock, then moves Part
````
````new server/internal/bootstrap/issue_writes_test.go
// and walt is refused (403); and she reads it, without a transaction. walt,
// made acme's admin and Web's guest (adminGuest), changes His, his own,
// which she has put under the first, hidden from him: the first is read
// under the lock with the count of the answer's sub-items, both on the
// transaction's connection, and the answer names no parent. She makes Part
// under the first, its parent read under the lock, then moves Part
````
````old server/internal/bootstrap/issue_writes_test.go
	route.answerWithin(t, w.contract, http.MethodGet, "/api/v0/issues/"+first.ID.String(), alice, "", http.StatusOK)
````
````new server/internal/bootstrap/issue_writes_test.go
	route.answerWithin(t, w.contract, http.MethodGet, "/api/v0/issues/"+first.ID.String(), alice, "", http.StatusOK)
	w.adminGuest(t)
	var his storedIssue
	decodeAnswer(t, send(w.ids["walt"], `{"name":"His"}`, http.StatusCreated), &his)
	hisPath := "/api/v0/issues/" + his.ID.String()
	route.answerWithin(t, w.contract, http.MethodPatch, hisPath, alice, `{"parent_id":"`+first.ID.String()+`"}`, http.StatusOK)
	var mended storedIssue
	if decodeAnswer(t, route.answerWithin(t, w.contract, http.MethodPatch, hisPath, w.ids["walt"], `{"name":"Mended"}`, http.StatusOK),
		&mended); mended.ParentID != nil {
		t.Errorf("walt's His, under the first, is answered under %s; want no parent, the first being hidden from him", mended.ParentID)
	}
````

- [ ] **Step 5: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/issue/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestEachAnswerCountsTheSubItemsTheCallerSees$|TestAHiddenParentIsNotInTheAnswer$|TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection$|TestTheGuestRuleHoldsForAParent$|TestAWorkItemIsPutUnderAnother$|TestPermissionMatrix' ./internal/bootstrap/`
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
Expected: 123 个全部通过（每个工作项的回答多了 `sub_issues_count`）。

- [ ] **Step 6: 提交**

```bash
git add api/modules/issue.yaml server/internal/bootstrap/issue_parents_test.go server/internal/bootstrap/issue_sub_issue_count_test.go server/internal/bootstrap/issue_writes_test.go server/internal/modules/issue/adapter/http/issues.go server/internal/modules/issue/adapter/http/issues_test.go server/internal/modules/issue/adapter/postgres/issues.go server/internal/modules/issue/adapter/postgres/queries/issues.sql server/internal/modules/issue/adapter/postgres/queries/sub_issues.sql server/internal/modules/issue/adapter/postgres/sub_issues.go server/internal/modules/issue/adapter/postgres/sub_issues_test.go server/internal/modules/issue/app/archive_issue.go server/internal/modules/issue/app/archive_issue_test.go server/internal/modules/issue/app/create_issue.go server/internal/modules/issue/app/dates_test.go server/internal/modules/issue/app/fakes_parents_test.go server/internal/modules/issue/app/fakes_test.go server/internal/modules/issue/app/get_issue.go server/internal/modules/issue/app/get_issue_by_identifier.go server/internal/modules/issue/app/get_issue_by_identifier_test.go server/internal/modules/issue/app/get_issue_test.go server/internal/modules/issue/app/guest_rule_test.go server/internal/modules/issue/app/lock.go server/internal/modules/issue/app/lock_test.go server/internal/modules/issue/app/parents_test.go server/internal/modules/issue/app/ports.go server/internal/modules/issue/app/seen.go server/internal/modules/issue/app/seen_test.go server/internal/modules/issue/app/update_issue.go server/internal/modules/issue/app/update_issue_test.go server/internal/modules/issue/domain/issue.go api/dist/openapi.yaml server/internal/modules/issue/adapter/http/gen/server.gen.go server/internal/modules/issue/adapter/postgres/gen/issues.sql.go server/internal/modules/issue/adapter/postgres/gen/sub_issues.sql.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M4/P3): each answer of a work item shows its sub-items and parent as the caller sees them

IssueDetail.sub_issues_count is the number of the work item's undeleted,
unarchived direct sub-items that the caller sees: those he created when
the guest rule hides others' from him in the project, every one
otherwise, by the grant of the decision that let the answer through. A
read counts after its decision, a write in its transaction, last; a new
work item has none. Where the count is read, the parents of the work
items under such a grant are read too, in one read, and one the guest
rule hides from the caller is answered as no parent.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p3.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试，或没有替身的平台、领域、转换的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；守卫是生成器（`make gen-go`）和中英文翻译键（`make lint-web`）的核对。每个变异在一份新的副本里，在本 Task 的树上、标出的较晚的 Task 的树上和最终的树上都跑过，端到端的检查只在标出的 Task 的树上）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T4.1` | 计数算上已归档的子工作项 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestEachAnswerCountsTheSubItemsTheCallerSees`（`make test`） | 存储；组合 |
| `T4.2` | 计数算上已删除的子工作项 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestEachAnswerCountsTheSubItemsTheCallerSees`（`make test`） | 存储；组合 |
| `T4.3` | 调用者自己建的子工作项就是全部子工作项 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestEachAnswerCountsTheSubItemsTheCallerSees`（`make test`） | 存储；组合 |
| `T4.4` | 只看自己的访客看到全部子工作项的个数 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestEachAnswerCountsTheSubItemsTheCallerSees`（`make test`） | 单元；组合 |
| `T4.5` | `SubIssueCounts` 把两个数答反 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestEachAnswerCountsTheSubItemsTheCallerSees`（`make test`） | 存储；组合 |
| `T4.6` | 什么都没改的修改回答不带计数的工作项 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestEachAnswerCountsTheSubItemsTheCallerSees`（`make test`） | 单元；组合 |
| `T4.7` | 什么都没改的归档回答不带计数的工作项 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestEachAnswerCountsTheSubItemsTheCallerSees`（`make test`） | 单元；组合 |
| `T4.8` | `getIssue` 回答不带计数的工作项 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestEachAnswerCountsTheSubItemsTheCallerSees`（`make test`）、故事 `i8-sub-issues`（Task 15 起） | 单元；组合；端到端 |
| `T4.9` | `getIssueByIdentifier` 回答不带计数的工作项 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestEachAnswerCountsTheSubItemsTheCallerSees`（`make test`） | 单元；组合 |
| `T4.10` | 修改按看得到每个工作项的 `Grant` 数子工作项，而不是锁下判定的 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestEachAnswerCountsTheSubItemsTheCallerSees`（`make test`） | 单元；组合 |
| `T4.11` | 回答里没有子工作项 | `internal/modules/issue/adapter/http` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestEachAnswerCountsTheSubItemsTheCallerSees`（`make test`）、故事 `i8-sub-issues`（Task 15 起） | 单元；组合；端到端 |
| `T4.12` | 回答给只看自己的访客他看不到的父工作项的 id | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAHiddenParentIsNotInTheAnswer`（`make test`）、`internal/bootstrap` 的 `TestARelationListShowsWhatTheCallerSees`（`make test`）（Task 12 起） | 单元；组合 |
| `T4.13` | 看得到每个工作项的授权下也遮住别人建的父工作项 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAHiddenParentIsNotInTheAnswer`（`make test`） | 单元；组合 |
| `T4.14` | `GetIssues` 也读已删除的工作项 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWorkItemsArePutUnderAWorkItem`（`make test`）（Task 6 起） | 存储；组合 |
| `T4.15` | 修改的回答数子工作项失败时略过：答一个空的工作项 | `internal/modules/issue/app` 的全部测试（`make test`） | 单元 |
| `T4.16` | `getIssueByIdentifier` 的回答数子工作项失败时略过：答一个空的工作项 | `internal/modules/issue/app` 的全部测试（`make test`） | 单元 |
| `T4.17` | `GetIssues` 照数据库读到行的次序回答，不按 `id` 的次序 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`） | 存储 |
| `T4.18` | 最后修改子工作项的账户算作它的"自己的"，而不是建它的账户 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestEachAnswerCountsTheSubItemsTheCallerSees`（`make test`） | 存储；组合 |
| `T4.19` | 什么都没改的修改的回答数子工作项失败时略过：答一个空的工作项 | `internal/modules/issue/app` 的全部测试（`make test`） | 单元 |
| `T4.20` | 什么都没改的归档、恢复的回答数子工作项失败时略过：答一个空的工作项 | `internal/modules/issue/app` 的全部测试（`make test`） | 单元 |
| `T4.21` | `hideParents` 在写的事务之外、经连接池读父工作项 | `internal/bootstrap` 的 `TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection`（`make test`） | 组合 |
| `T4.22` | `GetIssues` 经连接池读，不在它的事务的连接上 | `internal/bootstrap` 的 `TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection`（`make test`） | 组合 |

本 Task 的检查也发现较早的 Task 的变异 `T2.4`（列在各自的 Task 的表里，标"（Task 4 起）"）：本 Task 做完之后，在本 Task 的树上重跑它们（`mutants_p3.py`，`MUT_LATER=4`）。

---

### Task 5: 子工作项：`listSubIssues`、`addSubIssues` 的用例、端口和查询

**Files:**
- Create: `server/internal/modules/issue/app/sub_issues.go`、`server/internal/modules/issue/app/sub_issues_test.go`、`server/internal/modules/issue/domain/hierarchy_test.go`
- Modify: `server/internal/modules/issue/adapter/postgres/queries/sub_issues.sql`、`server/internal/modules/issue/adapter/postgres/sub_issues.go`、`server/internal/modules/issue/adapter/postgres/sub_issues_test.go`、`server/internal/modules/issue/app/create_issue_test.go`、`server/internal/modules/issue/app/fakes_parents_test.go`、`server/internal/modules/issue/app/fakes_test.go`、`server/internal/modules/issue/app/parents.go`、`server/internal/modules/issue/app/ports.go`、`server/internal/modules/issue/app/rows.go`、`server/internal/modules/issue/domain/hierarchy.go`
- Generate: `server/internal/modules/issue/adapter/postgres/gen/sub_issues.sql.go`

**Interfaces:**（spec 2.7、第 3 节第 6、9、11、12、13 条；M4 设计 3.4、3.5、3.8、3.9、3.13、5.1）
- `issue/domain/hierarchy.go`：`Loops(id, ancestry)`（`parentChange` 和加子工作项共用）；`MaxSubIssues`（100）；`CheckSubIssueIDs(ids)`（至少一个、至多 100 个、各一次，在任何读之前，一个 422）；`SubIssueNotAllowed(i)`（`sub_issue_ids[i]` 的 `not_allowed`，一句固定的文字）。
- `issue/adapter/postgres`：`queries/sub_issues.sql` 的 `SubIssues`（直接的、未删除、没归档的子工作项，给了账户时只要他建的，按建的次序 `created_at, id`）和 `SetParents`（`:execrows`，未删除的放到父工作项下，各自的最后修改者和时刻）；`sub_issues.go` 照写（`SetParents` 写的行数不是给的个数时是错误；它的两个错误只说几个工作项，不列出每个 `id`）。所问的工作项和父工作项用 Task 4 的 `GetIssues` 读。
- `issue/app`：`ports.go` 的 `SubIssueLister`（`SubIssues`）、`SubIssueAdder`（`Ancestry`、`SetParents`、`InsertActivities`，嵌入 `IssueReader`，`GetIssues` 在它里面）；`rows.go` 的 `ParentRows{IDs, ParentID, By, At}`；`parents.go` 的 `inReach(h, actor, projectID, i)`（这个项目的、没归档、调用者看得到：父工作项和子工作项共用这一条），`parentOf` 用它，`parentChange` 用 `domain.Loops`。
- `issue/app/sub_issues.go`（新）：`ListSubIssues`、`NewListSubIssues(locks, issues)`、`Execute(ctx, id)`：与 `getIssue` 同一个读和判定（`Locks.see`），然后读子工作项（`Grant.OwnOnly` 时只要他建的），按他的 `Grant` 数它们的子工作项（`Locks.seen`）。`AddSubIssues`、`AddSubIssuesDeps{Locks, Issues, Clock}`、`NewAddSubIssues`、`Execute(ctx, id, ids)`：`CheckSubIssueIDs` 在任何读之前；`Locks.run` 的 `write{issue: &id, action: issue.update}`；锁下项目已归档 409 → 父工作项已归档 409 `issue.archived` → `check`（按 `id` 的次序读所问的工作项和父工作项的祖先链；每个都要在调用者能放的范围里（`inReach`）、不是父工作项、不在它的祖先链里，问题按它在请求里的位置一次报出）→ 已在它下面的不动；要动的按 `id` 的次序、锁下读它们原来的父工作项（不在是错误）→ 有要动的才读时刻 → `SetParents` → 每个一行第 5 行的动态（原来的父工作项按标识到这个）→ 读回（少了是错误，只说读回了几个、写了几个，不列出每个 `id`），按调用者的 `Grant` 数子工作项；都已在它下面时什么都不写、回答空的。
- 测试的帮手：`fakes_test.go` 的 `fakeIssues.subItems`（`SubIssues` 回答的）、`parented`（最后一次 `SetParents` 的行）；`fakes_parents_test.go` 的 `SubIssues`、`SetParents`；`create_issue_test.go` 的 `wantActivityRows`（逐行核对动态：各自的工作项和项目、动态，`acme` 的、alice 的、时钟的时刻，`id` 一个比一个大），`wantActivity` 改为经它。加子工作项的测试用它和 `wantStoppedOn`，不另写一份。

**Tests:**
- 领域：`TestLoops`（父工作项本身、它上面的会成环，旁边的、下面的不会）、`TestCheckSubIssueIDs`（没有、101 个、重复的各报在自己的位置；1 个、100 个通过）。
- 单元：`TestListSubIssuesReadsTheSubItemsTheCallerSees`（管理员读全部，只看自己的访客只读她的，只是 web 的访客的管理员（`adminGuest`）也一样；数按她看到的）、`TestListSubIssuesRefusals`（与 `getIssue` 同样的拒绝，在读子工作项之前；读、数失败）、`TestAddSubIssuesPutsTheWorkItemsUnderIt`（3.5 的次序：`Locks` 的调用、判定，按 `id` 读所问的和祖先链，读要动的原来的父工作项，时刻，按 `id` 的次序放下，各一行第 5 行；已在它下面的不动、不写；回答动了的，按 `id` 的次序，读回、数过）、`TestAddingSubItemsUnderItAlreadyWritesNothing`（不读时刻、不写、回答空的）、`TestAddSubIssuesCountsTheSubItemsTheCallerSees`（只看自己的访客加她自己的工作项：回答按锁下的判定数，只数她的）、`TestAddSubIssuesRefusals`（读之前的、预检的、锁下的：已归档的项目、已归档的父工作项；没有的、别的项目的、已归档的、父工作项本身、它上面的、锁下的判定说只看自己的调用者的别人的（只是访客的管理员也一样，与没有的一样拒绝），各在自己的位置、同一个 422；什么都不写）、`TestAddSubIssuesStopsAtAFailingPort`（停在失败的那次调用上，`wantStoppedOn`）、`TestAddSubIssuesErrsOnWhatCannotBe`（原来的父工作项不在、读回少了：是错误，不是拒绝）。
- 存储：`TestSubIssues`（直接的、未删除、没归档的，按建的次序再按 `id`：同一时刻建的 y、z，存的次序与 `id` 的次序相反；给账户时只要他建的，不论最后是谁改的：a 是 alice 建的、bob 最后改的）、`TestSetParents`（只放给的未删除的，别的工作项的父工作项和最后修改不动；给已删除的是错误），用 Task 4 的 `newFamily`。

- [ ] **Step 1: 查询，生成**

`server/internal/modules/issue/adapter/postgres/queries/sub_issues.sql`（修改，1 处）：

````old server/internal/modules/issue/adapter/postgres/queries/sub_issues.sql
GROUP BY parent_id;

````
````new server/internal/modules/issue/adapter/postgres/queries/sub_issues.sql
GROUP BY parent_id;

-- name: SubIssues :many
-- The undeleted, unarchived work items directly under the work item parent_id, each as GetIssue reads it, those the
-- account created_by created when it is given, in the order of their creation (M4 design 3.4, 3.9): listSubIssues',
-- read without a lock. issues_parent_id_idx serves it.
SELECT i.id, i.workspace_id, i.project_id, i.sequence_id, i.name, i.state_id, i.priority, i.start_date, i.target_date,
       i.parent_id, i.sort_order, i.completed_at, i.archived_at, i.created_by_id, i.updated_by_id, i.created_at, i.updated_at,
       (SELECT array_agg(a.assignee_id ORDER BY a.assignee_id) FROM issue_assignees a
        WHERE a.issue_id = i.id AND a.deleted_at IS NULL)::uuid[] AS assignee_ids,
       (SELECT array_agg(l.label_id ORDER BY l.label_id) FROM issue_labels l
        WHERE l.issue_id = i.id AND l.deleted_at IS NULL)::uuid[] AS label_ids
FROM issues i
WHERE i.parent_id = sqlc.arg(parent_id)::uuid AND i.deleted_at IS NULL AND i.archived_at IS NULL
  AND (sqlc.narg(created_by)::uuid IS NULL OR i.created_by_id = sqlc.narg(created_by)::uuid)
ORDER BY i.created_at, i.id;

-- name: SetParents :execrows
-- addSubIssues' work items put under their parent (M4 design 3.9), each last changed at the moment and by the account
-- given; the undeleted ones among ids. Under the project's FOR NO KEY UPDATE, which every write of the project's work
-- items holds (3.5): one statement in scan order (convention 5).
UPDATE issues
SET parent_id = sqlc.arg(parent_id)::uuid, updated_by_id = sqlc.arg(by)::uuid, updated_at = sqlc.arg(at)
WHERE id = ANY (sqlc.arg(ids)::uuid[]) AND deleted_at IS NULL;

````

Run: `make gen-go`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `fbbb6493ee93c6acb0b98b84c09d27f4d12336118b82a15817c03661852a298f` | 167 | `server/internal/modules/issue/adapter/postgres/gen/sub_issues.sql.go` |

Run: `shasum -a 256 server/internal/modules/issue/adapter/postgres/gen/sub_issues.sql.go`
Expected: 与上表相同。

- [ ] **Step 2: 领域**

`server/internal/modules/issue/domain/hierarchy.go`（修改，4 处）：

````old server/internal/modules/issue/domain/hierarchy.go
import (
````
````new server/internal/modules/issue/domain/hierarchy.go
import (
	"fmt"
	"slices"
````
````old server/internal/modules/issue/domain/hierarchy.go
}

// ParentNotAllowed is the problem of a parent asked for that may not be
````
````new server/internal/modules/issue/domain/hierarchy.go
}

// Loops reports whether putting the work item id under a work item whose
// ancestry, it and those above it, is ancestry would make a loop: id is
// in it (M4 design 3.9). A parent's update and an addition of sub-items
// hold their work items to it.
func Loops(id uuid.UUID, ancestry []uuid.UUID) bool {
	return slices.Contains(ancestry, id)
}

// ParentNotAllowed is the problem of a parent asked for that may not be
````
````old server/internal/modules/issue/domain/hierarchy.go
		Message: "must be a work item of the project that is not archived, and neither this one nor under it"}
````
````new server/internal/modules/issue/domain/hierarchy.go
		Message: "must be a work item of the project that is not archived, and neither this one nor under it"}
}

// MaxSubIssues is how many work items an addition of sub-items names at
// most (M4 design 3.9, 5.1).
const MaxSubIssues = 100

// CheckSubIssueIDs checks the work items an addition of sub-items names,
// before any read (M4 design 3.9): at least one, at most MaxSubIssues,
// each named once; every problem in one 422 validation_failed. Whether
// each may go under the work item is the use case's, under the project's
// lock (SubIssueNotAllowed).
func CheckSubIssueIDs(ids []uuid.UUID) error {
	if len(ids) == 0 {
		return shared.Invalid(shared.FieldError{Field: "sub_issue_ids", Code: shared.FieldTooShort, Message: "must name a work item"})
	}
	return invalid(checkIDs("sub_issue_ids", ids, MaxSubIssues)...)
}

// SubIssueNotAllowed is the problem of the work item at index i of those
// an addition of sub-items names that may not go under the work item (M4
// design 3.9).
func SubIssueNotAllowed(i int) shared.FieldError {
	return shared.FieldError{Field: fmt.Sprintf("sub_issue_ids[%d]", i), Code: shared.FieldNotAllowed,
		Message: "must be a work item of the project that is not archived, and neither this one nor above it"}
````
````old server/internal/modules/issue/domain/hierarchy.go
// the new identifier, nil for none. An update writes it (UpdateActivity).
````
````new server/internal/modules/issue/domain/hierarchy.go
// the new identifier, nil for none. An update writes it (UpdateActivity),
// and so does an addition of sub-items for each one it moves.
````

`server/internal/modules/issue/domain/hierarchy_test.go`（新文件，63 行）：

````file server/internal/modules/issue/domain/hierarchy_test.go
package domain

import (
	"errors"
	"reflect"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// A work item under a work item of an ancestry that holds it would be
// under itself (M4 design 3.9): the parent itself, or one above it, but
// not one beside or below it.
func TestLoops(t *testing.T) {
	item, parent, grandparent, other := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	ancestry := []uuid.UUID{parent, grandparent}
	for _, tt := range []struct {
		name string
		id   uuid.UUID
		want bool
	}{{"the parent", parent, true}, {"one above it", grandparent, true}, {"another", other, false}, {"the work item", item, false}} {
		if got := Loops(tt.id, ancestry); got != tt.want {
			t.Errorf("%s: Loops() = %v, want %v", tt.name, got, tt.want)
		}
	}
	if Loops(item, nil) {
		t.Error("Loops() under no ancestry = true, want false")
	}
}

// An addition of sub-items names one work item at least and a hundred at
// most, each once (M4 design 3.9): none is refused at sub_issue_ids, a
// hundred and one too, alone, and each work item named again at its index;
// one and a hundred are accepted.
func TestCheckSubIssueIDs(t *testing.T) {
	a, b := uuid.NewV7(), uuid.NewV7()
	for _, tt := range []struct {
		name string
		ids  []uuid.UUID
		want []shared.FieldError
	}{
		{"one", []uuid.UUID{a}, nil},
		{"a hundred", ids(MaxSubIssues), nil},
		{"none", nil, []shared.FieldError{{Field: "sub_issue_ids", Code: "too_short", Message: "must name a work item"}}},
		{"a hundred and one, one twice", append(ids(MaxSubIssues), a, a)[1:],
			[]shared.FieldError{{Field: "sub_issue_ids", Code: "too_long", Message: "must name at most 100"}}},
		{"two named again", []uuid.UUID{a, b, a, b}, []shared.FieldError{
			{Field: "sub_issue_ids[2]", Code: "duplicate", Message: "is listed before"},
			{Field: "sub_issue_ids[3]", Code: "duplicate", Message: "is listed before"}}},
	} {
		err := CheckSubIssueIDs(tt.ids)
		var se *shared.Error
		if tt.want == nil && err != nil || tt.want != nil && (!errors.As(err, &se) || se.Code != "validation_failed" ||
			!reflect.DeepEqual(se.Fields, tt.want)) {
			t.Errorf("%s: CheckSubIssueIDs() = %v, want %+v", tt.name, err, tt.want)
		}
	}
	if got, want := SubIssueNotAllowed(4), (shared.FieldError{Field: "sub_issue_ids[4]", Code: "not_allowed",
		Message: "must be a work item of the project that is not archived, and neither this one nor above it"}); got != want {
		t.Errorf("SubIssueNotAllowed(4) = %+v, want %+v", got, want)
	}
}
````

- [ ] **Step 3: 用例**

`server/internal/modules/issue/app/create_issue_test.go`（修改，3 处）：

````old server/internal/modules/issue/app/create_issue_test.go
// work item id for each of activities, in their order, alice's at the
// clock's moment, each id after the row's before.
````
````new server/internal/modules/issue/app/create_issue_test.go
// work item id for each of activities, in their order (wantActivityRows).
````
````old server/internal/modules/issue/app/create_issue_test.go
	if len(w.issues.activities) != len(activities) {
		t.Fatalf("the activity %+v, want %d rows", w.issues.activities, len(activities))
````
````new server/internal/modules/issue/app/create_issue_test.go
	rows := make([]app.ActivityRow, len(activities))
	for i, a := range activities {
		rows[i] = app.ActivityRow{ProjectID: web, IssueID: id, Activity: a}
	}
	wantActivityRows(t, w, rows)
}

// wantActivityRows checks the activity w's store was given: the rows, in
// their order, each of its work item and project with its activity, in
// acme, alice's at the clock's moment, each id after the row's before.
func wantActivityRows(t *testing.T, w *world, rows []app.ActivityRow) {
	t.Helper()
	if len(w.issues.activities) != len(rows) {
		t.Fatalf("the activity %+v, want %d rows", w.issues.activities, len(rows))
````
````old server/internal/modules/issue/app/create_issue_test.go
		wantRow := app.ActivityRow{ID: r.ID, WorkspaceID: acme, ProjectID: web, IssueID: id, Activity: activities[i], ActorID: alice, At: clockNow}
		if !reflect.DeepEqual(r, wantRow) || i > 0 && r.ID.Compare(w.issues.activities[i-1].ID) <= 0 {
			t.Errorf("activity row %d: %+v, want %+v, its id after the row before", i, r, wantRow)
````
````new server/internal/modules/issue/app/create_issue_test.go
		want := rows[i]
		want.ID, want.WorkspaceID, want.ActorID, want.At = r.ID, acme, alice, clockNow
		if !reflect.DeepEqual(r, want) || i > 0 && r.ID.Compare(w.issues.activities[i-1].ID) <= 0 {
			t.Errorf("activity row %d: %+v, want %+v, its id after the row before", i, r, want)
````

`server/internal/modules/issue/app/fakes_parents_test.go`（修改，2 处）：

````old server/internal/modules/issue/app/fakes_parents_test.go
	"context"
````
````new server/internal/modules/issue/app/fakes_parents_test.go
	"context"
	"fmt"
````
````old server/internal/modules/issue/app/fakes_parents_test.go
	return out, nil
}

````
````new server/internal/modules/issue/app/fakes_parents_test.go
	return out, nil
}

// SubIssues logs the read, the parent's and whose sub-items it asks for,
// anyone's for none, and answers the fake's subItems, whatever is asked.
func (f *fakeIssues) SubIssues(ctx context.Context, parent uuid.UUID, by *uuid.UUID) ([]domain.Issue, error) {
	whose := "anyone's"
	if by != nil {
		whose = by.String() + "'s"
	}
	if err := f.log.add(ctx, "SubIssues", parent, whose); err != nil {
		return nil, err
	}
	return f.subItems, nil
}

// SetParents logs the write, keeps its rows (parented) and puts each of
// the fake's others it names under its parent, last changed by its writer
// at its moment; a work item it names that the fake has not is an error.
func (f *fakeIssues) SetParents(ctx context.Context, r app.ParentRows) error {
	if err := f.log.add(ctx, "SetParents", r.IDs, r.ParentID); err != nil {
		return err
	}
	f.parented = &r
	for _, id := range r.IDs {
		o, ok := f.others[id]
		if !ok {
			return fmt.Errorf("SetParents: no work item %s", id)
		}
		o.ParentID, o.UpdatedByID, o.UpdatedAt = &r.ParentID, &r.By, r.At
		f.others[id] = o
	}
	return nil
}

````

`server/internal/modules/issue/app/fakes_test.go`（修改，2 处）：

````old server/internal/modules/issue/app/fakes_test.go
// ancestries, and counts the work items' numbers of sub-items
````
````new server/internal/modules/issue/app/fakes_test.go
// ancestries, counts the work items' numbers of sub-items, subItems the
// sub-items it answers, and parented the last addition of sub-items' rows
````
````old server/internal/modules/issue/app/fakes_test.go
	counts     map[uuid.UUID]app.SubIssueCount
````
````new server/internal/modules/issue/app/fakes_test.go
	counts     map[uuid.UUID]app.SubIssueCount
	subItems   []domain.Issue
	parented   *app.ParentRows
````

`server/internal/modules/issue/app/parents.go`（修改，6 处）：

````old server/internal/modules/issue/app/parents.go
	"fmt"
	"slices"
````
````new server/internal/modules/issue/app/parents.go
	"fmt"
````
````old server/internal/modules/issue/app/parents.go
// so that the work items' lines stay as read until the write commits.

````
````new server/internal/modules/issue/app/parents.go
// so that the work items' lines stay as read until the write commits.

// inReach reports whether the caller actor may put the undeleted work
// item i under another of the project projectID, or another under it,
// under h, his write's lock on the project (M4 design 3.4, 3.8, 3.9): it
// is the project's, not archived, and he sees it (the guest rule with h's
// grant, Issue.HiddenFrom). Whether that makes a loop is the write's to
// check (domain.Loops).
func inReach(h held, actor, projectID uuid.UUID, i domain.Issue) bool {
	return i.ProjectID == projectID && i.ArchivedAt == nil && !i.HiddenFrom(actor, h.grant.OwnOnly)
}

````
````old server/internal/modules/issue/app/parents.go
// lock on it, and answers it when it may be one: an undeleted work item of
// the project, not archived, that he sees (the guest rule with h's grant,
// Issue.HiddenFrom); ok is false for any other. Whether the work item
// would be under itself is its update's to check (Ancestry).
````
````new server/internal/modules/issue/app/parents.go
// lock on it, and answers it when it may be one: an undeleted work item in
// his reach (inReach); ok is false for any other.
````
````old server/internal/modules/issue/app/parents.go
	if err != nil || !found {
		return domain.Issue{}, false, err
	}
	if p.ProjectID != projectID || p.ArchivedAt != nil || p.HiddenFrom(actor, h.grant.OwnOnly) {
		return domain.Issue{}, false, nil
````
````new server/internal/modules/issue/app/parents.go
	if err != nil || !found || !inReach(h, actor, projectID, p) {
		return domain.Issue{}, false, err
````
````old server/internal/modules/issue/app/parents.go
// item under it, the work item not in its ancestry; ok is false otherwise.
````
````new server/internal/modules/issue/app/parents.go
// item under it, the work item not in its ancestry (domain.Loops); ok is
// false otherwise.
````
````old server/internal/modules/issue/app/parents.go
		if err != nil || slices.Contains(ancestry, i.ID) {
````
````new server/internal/modules/issue/app/parents.go
		if err != nil || domain.Loops(i.ID, ancestry) {
````

`server/internal/modules/issue/app/ports.go`（修改，1 处）：

````old server/internal/modules/issue/app/ports.go
}

// IssueArchiver is the work items' store as archiveIssue and
````
````new server/internal/modules/issue/app/ports.go
}

// SubIssueLister reads the sub-items of a work item (M4 design 3.9).
type SubIssueLister interface {
	// SubIssues reads the undeleted, unarchived work items directly under
	// the work item parent, with their assignees and labels, those the
	// account by created when by is not nil, in the order of their
	// creation, (created_at, id), without a lock.
	SubIssues(ctx context.Context, parent uuid.UUID, by *uuid.UUID) ([]domain.Issue, error)
}

// SubIssueAdder is the work items' store as addSubIssues writes and reads
// them, in the transaction ctx carries (M4 design 3.5, 3.9).
type SubIssueAdder interface {
	IssueReader
	// Ancestry reads the work item and those above it, as IssueUpdater's.
	Ancestry(ctx context.Context, id uuid.UUID) ([]uuid.UUID, error)
	// SetParents puts the undeleted work items r names under its parent.
	SetParents(ctx context.Context, r ParentRows) error
	// InsertActivities inserts the rows in their order.
	InsertActivities(ctx context.Context, rows []ActivityRow) error
}

// IssueArchiver is the work items' store as archiveIssue and
````

`server/internal/modules/issue/app/rows.go`（修改，1 处）：

````old server/internal/modules/issue/app/rows.go
}

// IssueArchival is a work item's archiving or unarchiving as it is written
````
````new server/internal/modules/issue/app/rows.go
}

// ParentRows are the work items an addition of sub-items puts under a
// work item (M4 design 3.9): IDs, in id order, under ParentID, each last
// changed at the moment and by the account given.
type ParentRows struct {
	IDs      []uuid.UUID
	ParentID uuid.UUID
	By       uuid.UUID
	At       time.Time
}

// IssueArchival is a work item's archiving or unarchiving as it is written
````

`server/internal/modules/issue/app/sub_issues.go`（新文件，215 行）：

````file server/internal/modules/issue/app/sub_issues.go
package app

import (
	"context"
	"fmt"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListSubIssues lists the work items directly under a work item (M4
// design 3.9, 5.1).
type ListSubIssues struct {
	locks  Locks
	issues SubIssueLister
}

// NewListSubIssues returns the use case.
func NewListSubIssues(locks Locks, issues SubIssueLister) *ListSubIssues {
	return &ListSubIssues{locks: locks, issues: issues}
}

// Execute reads the work item id and decides issue.read on it for the
// caller, as getIssue does, without a lock or a transaction (Locks.see),
// with its refusals; then its undeleted, unarchived direct sub-items that
// he sees, those he created when the guest rule hides others' from him
// (Grant.OwnOnly), in the order of their creation, each as he sees it
// (Locks.seen).
func (u *ListSubIssues) Execute(ctx context.Context, id uuid.UUID) ([]domain.Issue, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	i, grant, err := u.locks.see(ctx, actor, id, domain.ActionRead)
	if err != nil {
		return nil, err
	}
	var by *uuid.UUID
	if grant.OwnOnly {
		by = &actor.UserID
	}
	subItems, err := u.issues.SubIssues(ctx, i.ID, by)
	if err != nil {
		return nil, err
	}
	return u.locks.seen(ctx, actor.UserID, grants{i.ProjectID: grant}, subItems...)
}

// AddSubIssuesDeps are what addSubIssues reads and writes through.
type AddSubIssuesDeps struct {
	Locks  Locks
	Issues SubIssueAdder
	Clock  Clock
}

// AddSubIssues puts work items under a work item (M4 design 3.5, 3.9,
// 5.1).
type AddSubIssues struct {
	d AddSubIssuesDeps
}

// NewAddSubIssues returns the use case.
func NewAddSubIssues(d AddSubIssuesDeps) *AddSubIssues {
	return &AddSubIssues{d: d}
}

// Execute checks ids (domain.CheckSubIssueIDs), before any read; then
// Locks runs the write on the work item id, the parent (M4 design 3.5,
// 11.11): its precheck, the decision on issue.update on it, the guest rule
// first, before any lock; then the project's write gate, which refuses a
// write that may wait no longer (503 server_busy); then in one
// transaction, the project FOR NO KEY UPDATE, the parent read again and
// the decision again, and under them, in this order: an archived project
// refuses it (409 project.archived), then an archived parent (409
// issue.archived); the work items ids name, each in the caller's reach
// and not the parent or above it (check), every problem in one 422; then,
// when any is not under the parent yet, the clock; those work items under
// the parent, in id order, each changed last by the caller; for each of
// them a row of its activity, from its parent before, by its identifier,
// to this one (row 5 of 3.13). All of it is the caller's at that moment.
// The answer is those work items as stored, read back in the transaction,
// in id order, as the caller sees them (Locks.seen); none, and nothing
// written, when each is under the parent already.
func (u *AddSubIssues) Execute(ctx context.Context, id uuid.UUID, ids []uuid.UUID) ([]domain.Issue, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	if err := domain.CheckSubIssueIDs(ids); err != nil {
		return nil, err
	}
	w := write{issue: &id, action: domain.ActionUpdate, notFound: domain.ErrIssueNotFound}
	var added []domain.Issue
	err = u.d.Locks.run(ctx, actor, w, func(ctx context.Context, h held) error {
		switch {
		case h.project.Archived:
			return domain.ErrProjectArchived
		case h.issue.ArchivedAt != nil:
			return domain.ErrIssueArchived
		}
		moved, err := u.check(ctx, h, actor.UserID, ids)
		if err != nil || len(moved) == 0 {
			return err
		}
		s := stamp{workspace: h.issue.WorkspaceID, project: h.issue.ProjectID, issue: id, by: actor.UserID, at: u.d.Clock.Now()}
		movedIDs := make([]uuid.UUID, len(moved))
		for n, m := range moved {
			movedIDs[n] = m.item.ID
		}
		if err := u.d.Issues.SetParents(ctx, ParentRows{IDs: movedIDs, ParentID: id, By: s.by, At: s.at}); err != nil {
			return err
		}
		var rows []ActivityRow
		for _, m := range moved {
			under := s
			under.issue = m.item.ID
			rows = append(rows, under.activity([]domain.Activity{domain.ParentActivity(m.parent)})...)
		}
		if err := u.d.Issues.InsertActivities(ctx, rows); err != nil {
			return err
		}
		stored, err := u.d.Issues.GetIssues(ctx, movedIDs)
		switch {
		case err != nil:
			return err
		case len(stored) != len(movedIDs):
			return fmt.Errorf("only %d of %d work items are there after their write", len(stored), len(movedIDs))
		}
		added, err = u.d.Locks.seen(ctx, actor.UserID, grants{h.issue.ProjectID: h.grant}, stored...)
		return err
	})
	if err != nil {
		return nil, err
	}
	return added, nil
}

// move is a work item an addition of sub-items puts under the work item:
// it as read under the lock, and its parent's change as its activity names
// it.
type move struct {
	item   domain.Issue
	parent domain.Changed[*domain.Ref]
}

// check reads the work items ids, which the caller actor's addition puts
// under the work item h holds, under h (M4 design 3.4, 3.9): each an
// undeleted work item in his reach (inReach) that is neither the work item
// nor one above it, in its ancestry (domain.Loops). Every problem is
// reported at once, in one 422, each at its place in ids. It answers those
// not under the work item yet, in id order: the ones it moves, each with
// its parent before, read under the lock too, by its identifier. A parent
// before that is not there is an error: deleting a work item deletes those
// under it (3.6).
func (u *AddSubIssues) check(ctx context.Context, h held, actor uuid.UUID, ids []uuid.UUID) ([]move, error) {
	found, err := u.d.Issues.GetIssues(ctx, slices.SortedFunc(slices.Values(ids), uuid.UUID.Compare))
	if err != nil {
		return nil, err
	}
	ancestry, err := u.d.Issues.Ancestry(ctx, h.issue.ID)
	if err != nil {
		return nil, err
	}
	var problems []shared.FieldError
	var moving []domain.Issue
	for at, id := range ids {
		n := slices.IndexFunc(found, func(i domain.Issue) bool { return i.ID == id })
		if n < 0 || !inReach(h, actor, h.issue.ProjectID, found[n]) || domain.Loops(id, ancestry) {
			problems = append(problems, domain.SubIssueNotAllowed(at))
			continue
		}
		if found[n].ParentID == nil || *found[n].ParentID != h.issue.ID {
			moving = append(moving, found[n])
		}
	}
	if len(problems) > 0 {
		return nil, shared.Invalid(problems...)
	}
	slices.SortFunc(moving, func(a, b domain.Issue) int { return a.ID.Compare(b.ID) })
	return u.moves(ctx, h, moving)
}

// moves are the work items moving under the work item h holds, each with
// its parent's change: from its parent before, read under the lock, to the
// work item, each by its identifier (refOf).
func (u *AddSubIssues) moves(ctx context.Context, h held, moving []domain.Issue) ([]move, error) {
	var before []uuid.UUID
	for _, i := range moving {
		if i.ParentID != nil && !slices.Contains(before, *i.ParentID) {
			before = append(before, *i.ParentID)
		}
	}
	var parents []domain.Issue
	if len(before) > 0 {
		var err error
		if parents, err = u.d.Issues.GetIssues(ctx, slices.SortedFunc(slices.Values(before), uuid.UUID.Compare)); err != nil {
			return nil, err
		}
	}
	out := make([]move, len(moving))
	for n, i := range moving {
		out[n] = move{item: i, parent: domain.Changed[*domain.Ref]{New: refOf(h, h.issue)}}
		if i.ParentID == nil {
			continue
		}
		at := slices.IndexFunc(parents, func(p domain.Issue) bool { return p.ID == *i.ParentID })
		if at < 0 {
			return nil, fmt.Errorf("work item %s is under %s, which is not there", i.ID, *i.ParentID)
		}
		out[n].parent.Old = refOf(h, parents[at])
	}
	return out, nil
}
````

`server/internal/modules/issue/app/sub_issues_test.go`（新文件，322 行）：

````file server/internal/modules/issue/app/sub_issues_test.go
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

// listOver is listSubIssues over w's ports.
func listOver(w *world) *app.ListSubIssues {
	return app.NewListSubIssues(w.locks, w.issues)
}

// addOver is addSubIssues over w's ports.
func addOver(w *world) *app.AddSubIssues {
	return app.NewAddSubIssues(app.AddSubIssuesDeps{Locks: w.locks, Issues: w.issues, Clock: clockAt{at: clockNow, log: w.log}})
}

// putUnder stores the work item i of w's store under the work item
// parent.
func putUnder(w *world, i domain.Issue, parent uuid.UUID) {
	i.ParentID = &parent
	w.issues.others[i.ID] = i
}

// sorted are ids in id order.
func sorted(ids []uuid.UUID) []uuid.UUID {
	return slices.SortedFunc(slices.Values(ids), uuid.UUID.Compare)
}

// A list of sub-items, outside any transaction and without a lock (M4
// design 3.4, 3.9): web's work item 7, alice's, read and issue.read
// decided on it, as getIssue does; then its sub-items, anyone's for web's
// admin, only hers for a guest who sees only her own, and for an admin who
// is only web's guest (adminGuest); then their sub-items counted, as she
// sees them: every one, or only hers. The answer is the sub-items as the
// store answers them, each with its count.
func TestListSubIssuesReadsTheSubItemsTheCallerSees(t *testing.T) {
	for _, tt := range []struct {
		name  string
		grant shared.Grant
		whose string
		items func(k kin) []domain.Issue // as the store answers them
		want  []int                      // their counts
	}{
		{"an admin", shared.Grant{ProjectRole: shared.RoleAdmin}, "anyone's", func(k kin) []domain.Issue { return []domain.Issue{k.top, k.up} },
			[]int{3, 2}},
		{"a guest who sees her own", ownOnly, alice.String() + "'s", func(k kin) []domain.Issue { return []domain.Issue{k.mine} }, []int{1}},
		{"an admin who is a guest", adminGuest, alice.String() + "'s", func(k kin) []domain.Issue { return []domain.Issue{k.mine} }, []int{1}},
	} {
		_, w, id := newUpdate()
		k := withKin(w)
		items := tt.items(k)
		w.issues.issue.By, w.auth.grants, w.issues.subItems = alice, []shared.Grant{tt.grant}, items
		w.issues.counts = map[uuid.UUID]app.SubIssueCount{k.top.ID: {All: 3}, k.up.ID: {All: 2, Own: 2}, k.mine.ID: {All: 4, Own: 1}}
		got, err := listOver(w).Execute(as(alice), id)
		want, ids := slices.Clone(items), []uuid.UUID{}
		for n := range want {
			want[n].SubIssuesCount = tt.want[n]
			ids = append(ids, want[n].ID)
		}
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("%s: Execute() = %+v, %v; want %+v", tt.name, got, err, want)
		}
		calls := []string{line("GetIssue", id, "outside tx"), line("Authorize", alice, domain.ActionRead, acme, web, "outside tx"),
			line("SubIssues", id, tt.whose, "outside tx"), counted(ids...) + " outside tx"}
		if !slices.Equal(w.log.calls, calls) {
			t.Errorf("%s: calls:\n%q\nwant\n%q", tt.name, w.log.calls, calls)
		}
	}
}

// What refuses a list of sub-items is what refuses getIssue, before any
// sub-item is read: a work item not there, or whose project the caller
// does not see, or that the guest rule hides from him, issue.not_found; a
// role that may not read it, forbidden. A read or a count that fails ends
// it with its failure; a work item without sub-items counts none.
func TestListSubIssuesRefusals(t *testing.T) {
	for _, tt := range []struct {
		name  string
		set   func(w *world, id uuid.UUID) uuid.UUID // the work item asked for
		want  error
		calls int // of the read, the decision, the list and the count
	}{
		{"no work item", func(*world, uuid.UUID) uuid.UUID { return uuid.NewV7() }, domain.ErrIssueNotFound, 1},
		{"a project not seen", func(w *world, id uuid.UUID) uuid.UUID {
			w.auth.answers = []error{shared.ErrNotVisible}
			return id
		}, domain.ErrIssueNotFound, 2},
		{"another's, to a guest who sees only his own", func(w *world, id uuid.UUID) uuid.UUID {
			w.auth.grants = []shared.Grant{ownOnly}
			return id
		}, domain.ErrIssueNotFound, 2},
		{"a role not allowed", func(w *world, id uuid.UUID) uuid.UUID {
			w.auth.answers = []error{shared.Forbidden()}
			return id
		}, shared.Forbidden(), 2},
		{"the sub-items' read fails", func(w *world, id uuid.UUID) uuid.UUID { w.log.fail = "SubIssues"; return id }, errDisk, 3},
		{"the count fails", func(w *world, id uuid.UUID) uuid.UUID { w.log.fail = "SubIssueCounts"; return id }, errDisk, 4},
		{"no sub-items", func(w *world, id uuid.UUID) uuid.UUID { w.issues.subItems = nil; return id }, nil, 3},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, w, id := newUpdate()
			k := withKin(w)
			w.issues.subItems = []domain.Issue{k.up}
			asked := tt.set(w, id)
			got, err := listOver(w).Execute(as(alice), asked)
			if !reflect.DeepEqual(err, tt.want) || len(got) != 0 {
				t.Errorf("Execute() = %+v, %v; want none and %v", got, err, tt.want)
			}
			calls := []string{line("GetIssue", asked, "outside tx"), line("Authorize", alice, domain.ActionRead, acme, web, "outside tx"),
				line("SubIssues", id, "anyone's", "outside tx"), counted(k.up.ID) + " outside tx"}[:tt.calls]
			if !slices.Equal(w.log.calls, calls) {
				t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
			}
		})
	}
}

// An addition of sub-items, in the order of M4 design 3.5: Locks' calls on
// the parent, web's work item 7, issue.update decided on it (lockedAs);
// the work items asked for, read in id order, and the parent's ancestry;
// the parents before of those that move, read; the clock; those that move
// under it, in id order, though asked in another, each a row of its
// activity, alice's at the moment, from its parent before by its
// identifier to WEB-7: up, from none; top, from up, WEB-3. mine, under 7
// already, moves not and writes no row. The answer is those that moved,
// read back, in id order, each under 7, changed last by alice then, its
// sub-items counted.
func TestAddSubIssuesPutsTheWorkItemsUnderIt(t *testing.T) {
	_, w, id := newUpdate()
	k := withKin(w)
	putUnder(w, k.top, k.up.ID)
	putUnder(w, k.mine, id)
	w.issues.counts = map[uuid.UUID]app.SubIssueCount{k.top.ID: {All: 4, Own: 1}}
	asked := []uuid.UUID{k.mine.ID, k.top.ID, k.up.ID}
	got, err := addOver(w).Execute(as(alice), id, asked)
	moved := []uuid.UUID{k.up.ID, k.top.ID}
	var want []domain.Issue
	for _, m := range moved {
		i := w.issues.others[m]
		i.SubIssuesCount = w.issues.counts[m].All
		want = append(want, i)
	}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("Execute() = %+v, %v; want %+v", got, err, want)
	}
	for _, i := range got {
		if !reflect.DeepEqual(i.ParentID, &id) || !reflect.DeepEqual(i.UpdatedByID, &alice) || !i.UpdatedAt.Equal(clockNow) {
			t.Errorf("%s answered under %v, changed last by %v at %v; want under %s, by alice at %v", i.ID, i.ParentID, i.UpdatedByID,
				i.UpdatedAt, id, clockNow)
		}
	}
	calls := append(lockedAs(domain.ActionUpdate, id), line("GetIssues", sorted(asked)), line("Ancestry", id),
		line("GetIssues", []uuid.UUID{k.up.ID}), "Now", line("SetParents", moved, id), line("InsertActivities", 2), line("GetIssues", moved),
		counted(moved...))
	if !slices.Equal(w.log.calls, calls) {
		t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
	}
	if want := (app.ParentRows{IDs: moved, ParentID: id, By: alice, At: clockNow}); w.issues.parented == nil ||
		!reflect.DeepEqual(*w.issues.parented, want) {
		t.Errorf("the rows written %+v, want %+v", w.issues.parented, want)
	}
	seven := &domain.Ref{ID: id, Identifier: "WEB-7"}
	wantActivityRows(t, w, []app.ActivityRow{
		{ProjectID: web, IssueID: k.up.ID, Activity: domain.ParentActivity(domain.Changed[*domain.Ref]{New: seven})},
		{ProjectID: web, IssueID: k.top.ID, Activity: domain.ParentActivity(domain.Changed[*domain.Ref]{Old: &domain.Ref{ID: k.up.ID,
			Identifier: "WEB-3"}, New: seven})},
	})
}

// An addition of work items each under the work item already writes
// nothing, reads not even the clock, and answers none (M4 design 3.9).
func TestAddingSubItemsUnderItAlreadyWritesNothing(t *testing.T) {
	_, w, id := newUpdate()
	k := withKin(w)
	putUnder(w, k.up, id)
	got, err := addOver(w).Execute(as(alice), id, []uuid.UUID{k.up.ID})
	calls := append(lockedAs(domain.ActionUpdate, id), line("GetIssues", []uuid.UUID{k.up.ID}), line("Ancestry", id))
	if err != nil || len(got) != 0 || !slices.Equal(w.log.calls, calls) || w.issues.parented != nil || w.issues.activities != nil {
		t.Errorf("Execute() = %+v, %v after\n%q; want none, after\n%q, nothing written", got, err, w.log.calls, calls)
	}
}

// A guest who sees only her own work items, alice, puts her mine under her
// work item 7 (M4 design 3.4, 3.9): the answer counts mine's sub-items as
// she sees them, by the decision under the lock: hers alone.
func TestAddSubIssuesCountsTheSubItemsTheCallerSees(t *testing.T) {
	_, w, id := newUpdate()
	k := withKin(w)
	w.issues.issue.By, w.auth.grants = alice, []shared.Grant{ownOnly, ownOnly}
	w.issues.counts = map[uuid.UUID]app.SubIssueCount{k.mine.ID: {All: 4, Own: 1}}
	got, err := addOver(w).Execute(as(alice), id, []uuid.UUID{k.mine.ID})
	if err != nil || len(got) != 1 || got[0].ID != k.mine.ID || got[0].SubIssuesCount != 1 {
		t.Errorf("Execute() = %+v, %v; want mine, with the one sub-item of hers", got, err)
	}
}

// What refuses an addition of sub-items (M4 design 3.5, 3.8, 3.9): before
// any read, none named, or one named twice; before any lock, what refuses
// updateIssue's precheck; under the lock, an archived project, then an
// archived parent; then each work item asked for that is not there,
// another project's, archived, the parent itself, one above it, or
// another's for a caller who sees only his own, by the decision under the
// lock, an admin who is only web's guest too (adminGuest), refused as one
// not there is: each at its place, in one 422, a work item it may move
// among them. Nothing is written.
func TestAddSubIssuesRefusals(t *testing.T) {
	gone := uuid.NewV7()
	checked := func(id uuid.UUID, asked []uuid.UUID) []string {
		return append(locked(id), line("GetIssues", sorted(asked)), line("Ancestry", id))
	}
	for _, tt := range []struct {
		name  string
		set   func(w *world, k kin, id uuid.UUID) (parent uuid.UUID, asked []uuid.UUID)
		want  error
		calls func(parent uuid.UUID, asked []uuid.UUID) []string
	}{
		{"none named", func(_ *world, _ kin, id uuid.UUID) (uuid.UUID, []uuid.UUID) { return id, nil }, domain.CheckSubIssueIDs(nil),
			func(uuid.UUID, []uuid.UUID) []string { return nil }},
		{"one named twice", func(_ *world, k kin, id uuid.UUID) (uuid.UUID, []uuid.UUID) { return id, []uuid.UUID{k.up.ID, k.up.ID} },
			domain.CheckSubIssueIDs(make([]uuid.UUID, 2)), func(uuid.UUID, []uuid.UUID) []string { return nil }},
		{"a parent not there", func(_ *world, k kin, _ uuid.UUID) (uuid.UUID, []uuid.UUID) { return gone, []uuid.UUID{k.up.ID} },
			domain.ErrIssueNotFound, func(parent uuid.UUID, _ []uuid.UUID) []string { return locked(parent)[:1] }},
		{"a role not allowed", func(w *world, k kin, id uuid.UUID) (uuid.UUID, []uuid.UUID) {
			w.auth.answers = []error{shared.Forbidden()}
			return id, []uuid.UUID{k.up.ID}
		}, shared.Forbidden(), func(parent uuid.UUID, _ []uuid.UUID) []string { return locked(parent)[:2] }},
		{"an archived project", func(w *world, k kin, id uuid.UUID) (uuid.UUID, []uuid.UUID) {
			w.project.locked.Archived, w.issues.archivedAt = true, &earlier
			return id, []uuid.UUID{k.up.ID}
		}, domain.ErrProjectArchived, func(parent uuid.UUID, _ []uuid.UUID) []string { return locked(parent) }},
		{"an archived parent", func(w *world, _ kin, id uuid.UUID) (uuid.UUID, []uuid.UUID) {
			w.issues.archivedAt = &earlier
			return id, []uuid.UUID{gone}
		}, domain.ErrIssueArchived, func(parent uuid.UUID, _ []uuid.UUID) []string { return locked(parent) }},
		{"each one it may not move", func(w *world, k kin, id uuid.UUID) (uuid.UUID, []uuid.UUID) {
			w.issues.ancestry = map[uuid.UUID][]uuid.UUID{id: {id, k.mine.ID}}
			return id, []uuid.UUID{gone, k.far.ID, k.top.ID, k.shelved.ID, id, k.mine.ID}
		}, shared.Invalid(domain.SubIssueNotAllowed(0), domain.SubIssueNotAllowed(1), domain.SubIssueNotAllowed(3), domain.SubIssueNotAllowed(4),
			domain.SubIssueNotAllowed(5)), checked},
		{"another's, for one who sees her own alone", func(w *world, k kin, id uuid.UUID) (uuid.UUID, []uuid.UUID) {
			w.issues.issue.By, w.auth.grants = alice, []shared.Grant{{}, ownOnly}
			return id, []uuid.UUID{k.mine.ID, k.up.ID}
		}, shared.Invalid(domain.SubIssueNotAllowed(1)), checked},
		{"another's, for an admin who is a guest", func(w *world, k kin, id uuid.UUID) (uuid.UUID, []uuid.UUID) {
			w.issues.issue.By, w.auth.grants = alice, []shared.Grant{adminGuest, adminGuest}
			return id, []uuid.UUID{gone, k.up.ID, k.mine.ID}
		}, shared.Invalid(domain.SubIssueNotAllowed(0), domain.SubIssueNotAllowed(1)), checked},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, w, id := newUpdate()
			k := withKin(w)
			parent, asked := tt.set(w, k, id)
			got, err := addOver(w).Execute(as(alice), parent, asked)
			wantRefusal(t, w, err, tt.want, tt.calls(parent, asked))
			if len(got) != 0 || w.issues.parented != nil || w.issues.activities != nil {
				t.Errorf("Execute() = %+v, written %+v and %+v; want none, nothing written", got, w.issues.parented, w.issues.activities)
			}
		})
	}
}

// A port that fails ends the addition with its failure, as itself, and no
// call after it runs: the work items asked for read, the ancestry, the
// parents before read, the write, the activity, the read back, the count.
func TestAddSubIssuesStopsAtAFailingPort(t *testing.T) {
	for _, at := range []string{"asked", "Ancestry", "before", "SetParents", "InsertActivities", "read back", "SubIssueCounts"} {
		t.Run(at, func(t *testing.T) {
			_, w, id := newUpdate()
			k := withKin(w)
			putUnder(w, k.top, k.up.ID)
			putUnder(w, k.mine, id)
			asked := []uuid.UUID{k.top.ID, k.mine.ID}
			w.log.fail = map[string]string{"asked": line("GetIssues", sorted(asked)), "Ancestry": line("Ancestry", id),
				"before": line("GetIssues", []uuid.UUID{k.up.ID}), "SetParents": line("SetParents", []uuid.UUID{k.top.ID}, id),
				"InsertActivities": line("InsertActivities", 1), "read back": line("GetIssues", []uuid.UUID{k.top.ID}),
				"SubIssueCounts": counted(k.top.ID)}[at]
			_, err := addOver(w).Execute(as(alice), id, asked)
			wantStoppedOn(t, w, err, w.log.fail)
		})
	}
}

// losing is a store that loses the work items whose activity it writes,
// so that they are not there to read back.
type losing struct{ *fakeIssues }

func (l losing) InsertActivities(ctx context.Context, rows []app.ActivityRow) error {
	for _, r := range rows {
		delete(l.others, r.IssueID)
	}
	return l.fakeIssues.InsertActivities(ctx, rows)
}

// What cannot be is an error, not a refusal: a work item under one that is
// not there, before the clock, since deleting a work item deletes those
// under it (M4 design 3.6); a work item moved that is not there to read
// back after its write.
func TestAddSubIssuesErrsOnWhatCannotBe(t *testing.T) {
	var se *shared.Error
	_, w, id := newUpdate()
	k := withKin(w)
	putUnder(w, k.top, uuid.NewV7())
	if _, err := addOver(w).Execute(as(alice), id, []uuid.UUID{k.top.ID}); err == nil || errors.As(err, &se) ||
		slices.Contains(w.log.calls, "Now") {
		t.Errorf("Execute() of a work item under one not there = %v after %q; want an error before the clock", err, w.log.calls)
	}
	_, w, id = newUpdate()
	k = withKin(w)
	add := app.NewAddSubIssues(app.AddSubIssuesDeps{Locks: w.locks, Issues: losing{w.issues}, Clock: clockAt{at: clockNow, log: w.log}})
	if got, err := add.Execute(as(alice), id, []uuid.UUID{k.top.ID}); err == nil || errors.As(err, &se) || len(got) != 0 {
		t.Errorf("Execute() of a work item lost after its write = %+v, %v; want an error", got, err)
	}
}
````

- [ ] **Step 4: 存储**

`server/internal/modules/issue/adapter/postgres/sub_issues.go`（修改，2 处）：

````old server/internal/modules/issue/adapter/postgres/sub_issues.go
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
````
````new server/internal/modules/issue/adapter/postgres/sub_issues.go
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
````
````old server/internal/modules/issue/adapter/postgres/sub_issues.go
	return counts, nil
}

````
````new server/internal/modules/issue/adapter/postgres/sub_issues.go
	return counts, nil
}

// SubIssues reads the undeleted, unarchived work items directly under the
// work item parent, those the account by created when by is not nil, in
// the order of their creation (M4 design 3.9).
func (s *Store) SubIssues(ctx context.Context, parent uuid.UUID, by *uuid.UUID) ([]domain.Issue, error) {
	rows, err := s.queries(ctx).SubIssues(ctx, gen.SubIssuesParams{ParentID: parent, CreatedBy: by})
	if err != nil {
		return nil, fmt.Errorf("read the sub-items of work item %s: %w", parent, err)
	}
	issues := make([]domain.Issue, len(rows))
	for n, r := range rows {
		issues[n] = issueOf(gen.GetIssueRow(r))
	}
	return issues, nil
}

// SetParents puts the work items r names under its parent, each last
// changed at its moment and by its account (M4 design 3.9). The use case
// read each of them undeleted under the project's lock, so one not written
// is an error.
func (s *Store) SetParents(ctx context.Context, r app.ParentRows) error {
	n, err := s.queries(ctx).SetParents(ctx, gen.SetParentsParams{ParentID: r.ParentID, By: r.By, At: r.At, Ids: r.IDs})
	switch {
	case err != nil:
		return fmt.Errorf("put %d work items under %s: %w", len(r.IDs), r.ParentID, err)
	case n != int64(len(r.IDs)):
		return fmt.Errorf("put %d of %d work items under %s", n, len(r.IDs), r.ParentID)
	}
	return nil
}

````

`server/internal/modules/issue/adapter/postgres/sub_issues_test.go`（修改，2 处）：

````old server/internal/modules/issue/adapter/postgres/sub_issues_test.go
}

// The work items GetIssues reads (M4 design 3.9): the undeleted ones among
````
````new server/internal/modules/issue/adapter/postgres/sub_issues_test.go
}

// A work item's sub-items as SubIssues reads them (M4 design 3.9): its
// undeleted, unarchived direct sub-items, each as GetIssue reads it, its
// assignees and labels too, in the order of their creation, then of their
// ids; an account's, those it created, whoever changed them last: p's are
// b, a, x, y and z, anyone's; a, alice's, though bob changed it last; b,
// x, y and z, bob's. y and z, made at one moment an hour after the others,
// are stored z first, though y's id is the lesser: their rows are not in
// the order of their ids. q has none, nor has a work item not there.
func TestSubIssues(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	f := newFamily(t, s, pool, web, alice, bob)
	exec(t, pool, "UPDATE issues SET updated_by_id = $2 WHERE id = $1", f.a, bob)
	y, z := uuid.NewV7(), uuid.NewV7()
	for n, id := range []uuid.UUID{z, y} {
		newIssue(t, pool, web, id, 9+n, bob)
		exec(t, pool, "UPDATE issues SET parent_id = $2, created_at = $3 WHERE id = $1", id, f.p, now.Add(time.Hour))
	}
	ctx := context.Background()
	for _, tt := range []struct {
		name   string
		parent uuid.UUID
		by     *uuid.UUID
		want   []domain.Issue
	}{
		{"p's, anyone's", f.p, nil, read(t, s, f.b, f.a, f.x, y, z)},
		{"p's, alice's", f.p, &alice, read(t, s, f.a)},
		{"p's, bob's", f.p, &bob, read(t, s, f.b, f.x, y, z)},
		{"q's", f.q, nil, read(t, s)},
		{"no work item's", uuid.NewV7(), nil, read(t, s)},
	} {
		if got, err := s.SubIssues(ctx, tt.parent, tt.by); err != nil || !reflect.DeepEqual(got, tt.want) {
			t.Errorf("SubIssues(%s) = %+v, %v; want %+v", tt.name, got, err, tt.want)
		}
	}
	cancelled, stop := context.WithCancel(ctx)
	stop()
	if got, err := s.SubIssues(cancelled, f.p, nil); !errors.Is(err, context.Canceled) || got != nil {
		t.Errorf("SubIssues() on a cancelled context = %+v, %v; want context.Canceled", got, err)
	}
}

// The work items GetIssues reads (M4 design 3.9): the undeleted ones among
````
````old server/internal/modules/issue/adapter/postgres/sub_issues_test.go
		t.Errorf("GetIssues() on a cancelled context = %+v, %v; want context.Canceled", got, err)
	}
}

````
````new server/internal/modules/issue/adapter/postgres/sub_issues_test.go
		t.Errorf("GetIssues() on a cancelled context = %+v, %v; want context.Canceled", got, err)
	}
}

// SetParents puts the undeleted work items it names under the parent, each
// changed last by the account and at the moment given, and no other (M4
// design 3.9): a and b under q, by bob, an hour after; every other work
// item keeps its parent and its last change, d, deleted, too. Naming a
// deleted work item is an error, as is a failed statement.
func TestSetParents(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	f := newFamily(t, s, pool, web, alice, bob)
	ctx := context.Background()
	type change struct {
		parent *uuid.UUID
		by     uuid.UUID
		at     time.Time
	}
	changes := func() map[uuid.UUID]change {
		t.Helper()
		rows, err := pool.Query(ctx, "SELECT id, parent_id, updated_by_id, updated_at FROM issues")
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		out := map[uuid.UUID]change{}
		for rows.Next() {
			var id uuid.UUID
			var c change
			if err := rows.Scan(&id, &c.parent, &c.by, &c.at); err != nil {
				t.Fatal(err)
			}
			c.at = c.at.UTC()
			out[id] = c
		}
		return out
	}
	want := changes()
	later := now.Add(time.Hour)
	if err := s.SetParents(ctx, app.ParentRows{IDs: []uuid.UUID{f.a, f.b}, ParentID: f.q, By: bob, At: later}); err != nil {
		t.Fatal(err)
	}
	want[f.a], want[f.b] = change{&f.q, bob, later}, change{&f.q, bob, later}
	if got := changes(); !reflect.DeepEqual(got, want) {
		t.Errorf("the work items' parents and last changes %+v, want %+v", got, want)
	}
	if err := s.SetParents(ctx, app.ParentRows{IDs: []uuid.UUID{f.x, f.d}, ParentID: f.q, By: bob, At: later}); err == nil {
		t.Error("SetParents() of a deleted work item = nil; want an error")
	}
	cancelled, stop := context.WithCancel(ctx)
	stop()
	if err := s.SetParents(cancelled, app.ParentRows{IDs: []uuid.UUID{f.a}, ParentID: f.p, By: bob, At: later}); !errors.Is(err,
		context.Canceled) {
		t.Errorf("SetParents() on a cancelled context = %v; want context.Canceled", err)
	}
}

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
git add server/internal/modules/issue/adapter/postgres/queries/sub_issues.sql server/internal/modules/issue/adapter/postgres/sub_issues.go server/internal/modules/issue/adapter/postgres/sub_issues_test.go server/internal/modules/issue/app/create_issue_test.go server/internal/modules/issue/app/fakes_parents_test.go server/internal/modules/issue/app/fakes_test.go server/internal/modules/issue/app/parents.go server/internal/modules/issue/app/ports.go server/internal/modules/issue/app/rows.go server/internal/modules/issue/app/sub_issues.go server/internal/modules/issue/app/sub_issues_test.go server/internal/modules/issue/domain/hierarchy.go server/internal/modules/issue/domain/hierarchy_test.go server/internal/modules/issue/adapter/postgres/gen/sub_issues.sql.go
```
```bash
git commit -m "feat(M4/P3): the use cases that list and add a work item's sub-items

listSubIssues reads the work item as getIssue does, then its undeleted,
unarchived direct sub-items, only the caller's own when the guest rule
hides others' from him, in the order of their creation, each counted as
he sees it. addSubIssues checks the work items named, then on the lock
path refuses an archived project or parent, and puts under the parent
each work item in the caller's reach that is not the parent or above it,
in id order, each with a row naming its parent before; those under it
already move not, and when all are, nothing is written.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p3.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试，或没有替身的平台、领域、转换的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；守卫是生成器（`make gen-go`）和中英文翻译键（`make lint-web`）的核对。每个变异在一份新的副本里，在本 Task 的树上、标出的较晚的 Task 的树上和最终的树上都跑过，端到端的检查只在标出的 Task 的树上）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T5.1` | 没有工作项成环：工作项放进它自己或它下面的工作项下面 | `internal/modules/issue/domain` 的全部测试（`make test`）、`internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAWorkItemIsPutUnderAnother`（`make test`）、`internal/bootstrap` 的 `TestTwoMovesThatWouldMakeALoopSerialize`（`make test`）、`internal/bootstrap` 的 `TestWorkItemsArePutUnderAWorkItem`（`make test`）（Task 6 起） | 单元；组合 |
| `T5.2` | 加子工作项可以一个也不列 | `internal/modules/issue/domain` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWorkItemsArePutUnderAWorkItem`（`make test`）（Task 6 起） | 单元；组合 |
| `T5.3` | 加子工作项可以把一个工作项列两次、列一百个以上 | `internal/modules/issue/domain` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWorkItemsArePutUnderAWorkItem`（`make test`）（Task 6 起） | 单元；组合 |
| `T5.4` | 子工作项的问题不论它自己在哪个位置，都写成第一个位置 | `internal/modules/issue/domain` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWorkItemsArePutUnderAWorkItem`（`make test`）（Task 6 起） | 单元；组合 |
| `T5.5` | 调用者够不着的工作项被放到它下面：别的项目的、已归档的、访客规则对他隐藏的 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWorkItemsArePutUnderAWorkItem`（`make test`）（Task 6 起）、`internal/bootstrap` 的 `TestTheGuestRuleHoldsForSubItems`（`make test`）（Task 6 起）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `addSubIssues` 各格（`make test`）（Task 6 起） | 单元；组合 |
| `T5.6` | 加子工作项把这个工作项自己、或者它上面的工作项放到它下面 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWorkItemsArePutUnderAWorkItem`（`make test`）（Task 6 起） | 单元；组合 |
| `T5.7` | 读工作项的祖先链失败时略过：当作它上面什么也没有 | `internal/modules/issue/app` 的全部测试（`make test`） | 单元 |
| `T5.8` | 已经在它下面的工作项再移动一次 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWorkItemsArePutUnderAWorkItem`（`make test`）（Task 6 起） | 单元；组合 |
| `T5.9` | 移动的那行动态不写原来的父工作项 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWorkItemsArePutUnderAWorkItem`（`make test`）（Task 6 起）、故事 `i8-sub-issues`（Task 15 起） | 单元；组合；端到端 |
| `T5.10` | 列出的每个工作项都写，已经在它下面的也写 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWorkItemsArePutUnderAWorkItem`（`make test`）（Task 6 起） | 单元；组合 |
| `T5.11` | `SetParents` 也写已删除的工作项 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`） | 存储 |
| `T5.12` | `SetParents` 给工作项盖的时刻比写的时刻早一秒 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWorkItemsArePutUnderAWorkItem`（`make test`）（Task 6 起） | 存储；组合 |
| `T5.13` | 子工作项列表含已归档的 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWorkItemsArePutUnderAWorkItem`（`make test`）（Task 6 起） | 存储；组合 |
| `T5.14` | 子工作项列表给只看自己的访客看所有人的 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestTheGuestRuleHoldsForSubItems`（`make test`）（Task 6 起） | 存储；组合 |
| `T5.15` | `listSubIssues` 不看调用者的 `Grant`，要所有人的子工作项 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestTheGuestRuleHoldsForSubItems`（`make test`）（Task 6 起） | 单元；组合 |
| `T5.16` | 加子工作项在锁下、闸门之后才核对列出的工作项的个数和重复，而不是在任何读之前 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestEachWriteOnAWorkItemOfAProjectTooBusyAnswersServerBusy`（`make test`）（Task 6 起） | 单元；组合 |
| `T5.17` | 加子工作项回读到什么就答什么，少了一个工作项也答 | `internal/modules/issue/app` 的全部测试（`make test`） | 单元 |
| `T5.18` | 加子工作项回答的工作项不带计数 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestTheGuestRuleHoldsForSubItems`（`make test`）（Task 6 起） | 单元；组合 |
| `T5.19` | 已归档的工作项可以加子工作项 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWorkItemsArePutUnderAWorkItem`（`make test`）（Task 6 起）、故事 `i15-archive-issue`（Task 15 起） | 单元；组合；端到端 |
| `T5.20` | 已归档的项目里已归档的工作项先答"工作项已归档" | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `addSubIssues` 各格（`make test`）（Task 6 起） | 单元；组合 |
| `T5.21` | 读子工作项失败时略过：当作它没有子工作项 | `internal/modules/issue/app` 的全部测试（`make test`） | 单元 |
| `T5.22` | 不能放到这个工作项下面的每个工作项都在第一个位置上被拒绝 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWorkItemsArePutUnderAWorkItem`（`make test`）（Task 6 起） | 单元；组合 |
| `T5.23` | 子工作项的列表按不带访客规则的授权数每一项的子工作项 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestTheGuestRuleHoldsForSubItems`（`make test`）（Task 6 起） | 单元；组合 |
| `T5.24` | 加子工作项的回答按不带访客规则的授权数每一项的子工作项 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestTheGuestRuleHoldsForSubItems`（`make test`）（Task 6 起） | 单元；组合 |
| `T5.25` | `SetParents` 经连接池写，不在它的事务的连接上 | `internal/bootstrap` 的 `TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection`（`make test`）（Task 6 起） | 组合 |
| `T5.26` | 加子工作项在锁之前读父工作项的祖先链，锁下照它判定 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAddingSubItemsAndAMoveThatWouldMakeALoopSerialize`（`make test`）（Task 6 起） | 单元；组合 |
| `T5.27` | `SubIssues` 给只看自己的访客回答他最后修改的子工作项，而不是他建的 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`） | 存储 |
| `T5.28` | `SubIssues` 对同一时刻建的子工作项照数据库读到行的次序回答，不按 `id` 的次序 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`） | 存储 |
| `T5.29` | `listSubIssues` 给只是访客的管理员要所有人的子工作项，虽然他的 `Grant` 是 `OwnOnly` | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestTheGuestRuleHoldsForSubItems`（`make test`）（Task 6 起） | 单元；组合 |
| `T5.30` | 写的范围里访客规则对只是访客的管理员不成立：他能把别人的工作项放到他的下面，或把他的放到别人的下面 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestTheGuestRuleHoldsForAParent`（`make test`）、`internal/bootstrap` 的 `TestTheGuestRuleHoldsForSubItems`（`make test`）（Task 6 起） | 单元；组合 |

---

### Task 6: 子工作项：接口、组合、矩阵和交错 9、13

**Files:**
- Create: `server/internal/bootstrap/issue_sub_issues_test.go`、`server/internal/bootstrap/permission_matrix_sub_issues_test.go`、`server/internal/modules/issue/adapter/http/sub_issues.go`、`server/internal/modules/issue/adapter/http/sub_issues_test.go`
- Modify: `api/modules/issue.yaml`、`api/openapi.yaml`、`server/internal/bootstrap/gate_test.go`、`server/internal/bootstrap/issue_interleaving_hierarchy_test.go`、`server/internal/bootstrap/issue_write_locks_test.go`、`server/internal/bootstrap/issue_writes_test.go`、`server/internal/bootstrap/permission_matrix_test.go`、`server/internal/modules/issue/adapter/http/handler.go`、`server/internal/modules/issue/module.go`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/issue/adapter/http/gen/bodyshape.gen.go`、`server/internal/modules/issue/adapter/http/gen/server.gen.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**（spec 2.8、第 3 节第 9、11、13、15、17 条；M4 设计 3.5、3.9、5.1、5.2、9.2、9.3）
- 契约：`GET /api/v0/issues/{issue_id}/sub-issues`（`listSubIssues`，200 `IssueCollection`，`x-problem-codes: [issue.not_found, forbidden]`）、`POST` 同一路径（`addSubIssues`，体 `SubIssuesAdd{sub_issue_ids}`（1–100 个，`additionalProperties: false`），200 `IssueCollection`，`x-problem-codes: [validation_failed, issue.not_found, forbidden, project.archived, issue.archived, server_busy]`）；模式 `Issue`（列表的一行：`IssueDetail` 的字段，P3 两者相同，`IssueDetail` 的 `description_html` 在 P4）、`IssueCollection{data}`（没有分页）；`api/openapi.yaml` 的路径。描述写上谁能读、能加，访客规则，次序，已在它下面的不动、都在时什么都不写。
- `issue/adapter/http`：`handler.go` 的 `ListSubIssuesUseCase`、`AddSubIssuesUseCase` 和 `UseCases` 的两个字段；`sub_issues.go`（新）：`ListSubIssues`、`AddSubIssues`、`issueCollection`（经 `issueDetail` 转换：`gen.Issue` 与 `gen.IssueDetail` 的形状相同由编译钉住）。`issue/module.go` 接上两个用例，说明写上它们。
- `bootstrap`：`issue_sub_issues_test.go`（新：`subIssues`、`subIssueIDs`、`countedIssue`（列表回答的工作项：`storedIssue` 和调用者看到的子工作项的个数）、`listIn[T any]`（一个列表回答的元素，泛型：这里是 `countedIssue`，Task 11 的关联照用）、`itemIDs`、`counts`）；`issue_interleaving_hierarchy_test.go` 的交错 13（`deletesNamed`、`addsUnder`）和交错 9 的加子工作项一半（`TestAddingSubItemsAndAMoveThatWouldMakeALoopSerialize`）；`permission_matrix_sub_issues_test.go`（新：`listSubIssues`、`addSubIssues` 的行，`subItemOf`、`addsSubItem`、`listsNone`、`putsUnder`）并在 `permission_matrix_test.go` 登记；`issue_write_locks_test.go` 的 `body` 带上 `t`，`addSubIssues` 一行（每次发送新建父工作项，把 Web 的 Part 移到它下面；锁下不多读，等它先写的 `issues`，`SHARE`）；`gate_test.go` 的 `TestEachWriteOnAWorkItemOfAProjectTooBusyAnswersServerBusy`：写带上体，加子工作项一行，形状不对的请求一行（项目唯一的位置被占着时，把一个工作项列两次的加子工作项立刻 422，不是 503：这项核对在任何读之前，不占闸门）；`issue_writes_test.go` 的 `TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection`：alice 建 Low，把它放到 Part 下。

**Tests:**
- HTTP（`apitest.Main` 两个方向）：`TestListSubIssues`、`TestAddSubIssues`（体的工作项照原样、照次序进用例，重复的、空的也是；回答与 `GET` 的一样）、`TestAddSubIssuesHoldsTheBodyToItsStructure`（没有工作项、`null`、别的类型、不是 `id` 的、多出的字段：用例之前 400 `bad_request`）、`TestSubIssuesRefusals`（列的拒绝同 `getIssue`，加的同 `updateIssue`）。
- 组合：`TestWorkItemsArePutUnderAWorkItem`（bob 把 C、B、A 放到 Top 下：回答 A、B、C，按 `id` 的次序，各一行他的第 5 行，B 的从 `WEB-3`；Top 列出它们、数 3 个，已归档的 D 放到它下面之后谁都列不出；再放 A 回答空的、不写；放 A 和 E 只回答 E，不写 A；不给工作项、给两次 Up、给 101 次 Up 各 422，什么都不写；alice 把 Root 放到 Base 下之后，Top 不能要的各在自己的位置 422、什么都不写：Ops 的、Top 自己、已归档的、它上面的 Root、Root 上面的 Base（只读两层的祖先链放过它）、已删除的、没有的，可以要的 Up 在它们中间也不动；已归档的工作项不能加，但照样列出它下面的）、`TestTheGuestRuleHoldsForSubItems`（Mine 下有 alice 的 Low、walt 的 Below；walt 把 Mine 放到 His 下，回答的 Mine 数 1 个子工作项（他看得到的）；他看不到的 Hers 在它的位置被拒绝，回答与不存在的逐值相同；walt 只列出 Mine，数 1 个，alice、bob 列出两个，Mine 数 2 个；alice 的 Up 对 walt 列、加都是与不存在的工作项逐值相同的 404）；交错 13 `TestDeletingAWorkItemAndPuttingAnotherUnderItSerialize`（alice 删除 Up、bob 把 Part 放到它下面，两种顺序：先删，加在锁下找不到 Up，404，Part 照旧；先加，删除等 Web 的行，锁下读子树、Part 在其中，两个和它们的七行（含 bob 的第 5 行）由 alice 在同一时刻删除，不早于加的提交）；交错 9 的另一对 `TestAddingSubItemsAndAMoveThatWouldMakeALoopSerialize`（alice 把 A 加到 B 下、bob 把 B 移到 A 下，两种顺序：后来的等 Web 的行，在锁下读祖先链，看出成环，在它的位置被拒绝、什么都不写：移动 422 `parent_id`，加子工作项 422 `sub_issue_ids[0]`；先来的工作项在另一个下面，另一个不在谁下面，只有先来的写了一行第 5 行）；`TestEachIssueWriteLocksItsProjectFirst` 的 `addSubIssues` 一行；`TestEachWriteOnAWorkItemOfAProjectTooBusyAnswersServerBusy` 的加子工作项一行（项目的一个位置被占、不排队时立刻 503 `server_busy`、`Retry-After: 1`）和形状不对的一行（同时立刻 422 `sub_issue_ids[1]` `duplicate`）；`TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection`（一个连接的池：alice 把 Low 放到 Part 下，锁下的读、写和回答的读都在它的事务的连接上）；`TestPermissionMatrix` 的两个操作的行（列：照 `getIssue`；加：管理员、成员、建了它的成员能加，访客、"管理员·访客"对别人的 404……；已归档的项目 409，已归档的项目里已归档的工作项是项目的 409，已归档的工作项 409 `issue.archived`，别的项目的工作项 422）、`TestThePermissionMatrixCoversEveryOperation`。

- [ ] **Step 1: 契约，生成**

`api/modules/issue.yaml`（修改，2 处）：

````old api/modules/issue.yaml
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/workspaces/{slug}/issues/{identifier}:
````
````new api/modules/issue.yaml
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/issues/{issue_id}/sub-issues:
    parameters:
      - $ref: '#/components/parameters/IssueID'
    get:
      operationId: listSubIssues
      tags: [issue]
      summary: List the work items under a work item
      description: >-
        For the admins, members and guests of its project, as getIssue,
        which answers the same refusals: issue.not_found, forbidden. The
        work items directly under it, not those under them, that are not
        archived and that the caller sees: a guest of a project that does
        not show its guests every work item sees only those he created.
        They come in the order they were created in, the first first, all
        of them, without pages; an archived work item's too.
      security: [{bearer: []}]
      x-problem-codes: [issue.not_found, forbidden]
      responses:
        '200':
          description: The work items under it.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IssueCollection'
        default:
          $ref: '#/components/responses/Problem'
    post:
      operationId: addSubIssues
      tags: [issue]
      summary: Put work items under a work item
      description: >-
        For the admins and members of its project. Refused from the request
        alone, before the work item is looked at (validation_failed): no
        work item, or more than 100 (sub_issue_ids too_short, too_long;
        more than 100 is the one problem named), and a work item named
        twice (sub_issue_ids[i] duplicate). A work item that does not
        exist, is deleted, whose project the caller does not see, or that
        he did not create as a guest who sees only his own (getIssue)
        answers issue.not_found, whatever his role may do; one whose
        project he sees but whose work items he may not change, forbidden;
        both are answered before the project is locked, and with the
        project locked the role is decided again. A work item of an
        archived project takes none under it (project.archived), nor then
        an archived work item (issue.archived). Then each work item named
        is one of the project's that is not archived and that the caller
        sees, and neither this work item nor one above it
        (sub_issue_ids[i] not_allowed, every problem at once). One under
        another work item moves under this one; one under it already stays
        as it is. Each that moves is changed last by the caller, and its
        activity says its parent changed, from the one it was under. Every
        row it writes is the caller's, at one moment. A project with too
        many writes on its work items at once answers server_busy, with
        Retry-After: try again then.
      security: [{bearer: []}]
      x-problem-codes: [validation_failed, issue.not_found, forbidden, project.archived, issue.archived, server_busy]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/SubIssuesAdd'
      responses:
        '200':
          description: The work items that moved under it, in the order of their ids; none when each was under it already.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IssueCollection'
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/workspaces/{slug}/issues/{identifier}:
````
````old api/modules/issue.yaml
          type: string
          format: date-time
    IssueCreate:
````
````new api/modules/issue.yaml
          type: string
          format: date-time
    Issue:
      description: A work item of a project, as a list shows it.
      type: object
      additionalProperties: false
      required: [id, workspace_id, project_id, sequence_id, name, state_id, priority, start_date, target_date, parent_id, sort_order,
        completed_at, archived_at, assignee_ids, label_ids, sub_issues_count, created_by_id, updated_by_id, created_at, updated_at]
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
        start_date:
          description: The day it starts; null for none.
          type: [string, 'null']
          format: date
        target_date:
          description: The day it is due, not before its start date; null for none.
          type: [string, 'null']
          format: date
        parent_id:
          description: >-
            The work item it is under, in the same project; null for none, or
            for one the caller does not see (a guest who sees only his own
            work items).
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
        sub_issues_count:
          description: >-
            The number of the work items directly under it that are not
            archived and that the caller sees: a guest of the project who
            sees only his own work items counts only his own.
          type: integer
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
    IssueCollection:
      description: Work items, all of them, without pages.
      type: object
      additionalProperties: false
      required: [data]
      properties:
        data:
          type: array
          items:
            $ref: '#/components/schemas/Issue'
    SubIssuesAdd:
      type: object
      additionalProperties: false
      required: [sub_issue_ids]
      properties:
        sub_issue_ids:
          description: >-
            1–100 work items of the project, each named once, to put under
            the work item.
          type: array
          items:
            type: string
            format: uuid
    IssueCreate:
````

`api/openapi.yaml`（修改，1 处）：

````old api/openapi.yaml
    $ref: 'modules/issue.yaml#/paths/~1api~1v0~1issues~1{issue_id}~1unarchive'
````
````new api/openapi.yaml
    $ref: 'modules/issue.yaml#/paths/~1api~1v0~1issues~1{issue_id}~1unarchive'
  /api/v0/issues/{issue_id}/sub-issues:
    $ref: 'modules/issue.yaml#/paths/~1api~1v0~1issues~1{issue_id}~1sub-issues'
````

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `62e360146adf6d6012558c5c2c7273171eded55a490182aff9023b6824d85e72` | 3535 | `api/dist/openapi.yaml` |
| `acf6e13870602938104f5561f5bd52466c706890c85b7c4aff6686f8de125332` | 44 | `server/internal/modules/issue/adapter/http/gen/bodyshape.gen.go` |
| `956bc7188cf3808abdf964010039371fa3a8df7926fb1f601f82919e572131de` | 1402 | `server/internal/modules/issue/adapter/http/gen/server.gen.go` |
| `26386e7c2cb5aca5bfe8fc5ef37086f8697da46e13ff2043d821a6a019fc8719` | 3860 | `web/packages/api-client/src/schema.gen.ts` |

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

// ListSubIssuesUseCase is app.ListSubIssues.
type ListSubIssuesUseCase interface {
	Execute(ctx context.Context, id uuid.UUID) ([]domain.Issue, error)
}

// AddSubIssuesUseCase is app.AddSubIssues.
type AddSubIssuesUseCase interface {
	Execute(ctx context.Context, id uuid.UUID, ids []uuid.UUID) ([]domain.Issue, error)
}

// UseCases are the use cases behind the module's operations.
````
````old server/internal/modules/issue/adapter/http/handler.go
	UnarchiveIssue       ArchiveIssueUseCase
````
````new server/internal/modules/issue/adapter/http/handler.go
	UnarchiveIssue       ArchiveIssueUseCase
	ListSubIssues        ListSubIssuesUseCase
	AddSubIssues         AddSubIssuesUseCase
````

`server/internal/modules/issue/adapter/http/sub_issues.go`（新文件，39 行）：

````file server/internal/modules/issue/adapter/http/sub_issues.go
package httpadapter

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// ListSubIssues serves GET /api/v0/issues/{issue_id}/sub-issues.
func (h handler) ListSubIssues(ctx context.Context, req gen.ListSubIssuesRequestObject) (gen.ListSubIssuesResponseObject, error) {
	items, err := h.uc.ListSubIssues.Execute(ctx, req.IssueID)
	if err != nil {
		return nil, err
	}
	return gen.ListSubIssues200JSONResponse(issueCollection(items)), nil
}

// AddSubIssues serves POST /api/v0/issues/{issue_id}/sub-issues: the work
// items go to the use case as they came, which checks them.
func (h handler) AddSubIssues(ctx context.Context, req gen.AddSubIssuesRequestObject) (gen.AddSubIssuesResponseObject, error) {
	items, err := h.uc.AddSubIssues.Execute(ctx, req.IssueID, req.Body.SubIssueIds)
	if err != nil {
		return nil, err
	}
	return gen.AddSubIssues200JSONResponse(issueCollection(items)), nil
}

// issueCollection is items as the API lists them, in their order, [] for
// none: each as issueDetail shows it, a list's Issue having IssueDetail's
// fields (M4 design 5.2), so that the conversion stops compiling once the
// two part.
func issueCollection(items []domain.Issue) gen.IssueCollection {
	data := make([]gen.Issue, len(items))
	for n, i := range items {
		data[n] = gen.Issue(issueDetail(i))
	}
	return gen.IssueCollection{Data: data}
}
````

`server/internal/modules/issue/adapter/http/sub_issues_test.go`（新文件，168 行）：

````file server/internal/modules/issue/adapter/http/sub_issues_test.go
package httpadapter_test

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeList is listSubIssues: each call is recorded as "caller work item";
// it answers answer, or err.
type fakeList struct {
	calls  []string
	answer []domain.Issue
	err    error
}

func (f *fakeList) Execute(ctx context.Context, id uuid.UUID) ([]domain.Issue, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	f.calls = append(f.calls, actor.UserID.String()+" "+id.String())
	return f.answer, f.err
}

// fakeAdd is addSubIssues: each call is recorded as "caller work item",
// with the work items it got; it answers answer, or err.
type fakeAdd struct {
	calls  []string
	got    [][]uuid.UUID
	answer []domain.Issue
	err    error
}

func (f *fakeAdd) Execute(ctx context.Context, id uuid.UUID, ids []uuid.UUID) ([]domain.Issue, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	f.calls = append(f.calls, actor.UserID.String()+" "+id.String())
	f.got = append(f.got, ids)
	return f.answer, f.err
}

// subIssuesPath is the path of WEB-7's sub-items; listed, the answers of
// both operations: WEB-7 and WEB-8, each as getIssue shows it, in the use
// case's order, or none.
var (
	subIssuesPath = "/api/v0/issues/" + webSeven.ID.String() + "/sub-issues"
	listed        = []struct {
		items []domain.Issue
		json  string
	}{{[]domain.Issue{webSeven, webEight}, `{"data":[` + webSevenJSON + `,` + webEightJSON + `]}`}, {nil, `{"data":[]}`}}
)

// GET goes to the use case for the caller and the path's work item; the
// answer is 200 with the work items it answers, in its order, [] for none.
func TestListSubIssues(t *testing.T) {
	for _, answer := range listed {
		list := &fakeList{answer: answer.items}
		res, body := do(t, newServer(t, httpadapter.UseCases{ListSubIssues: list}), request(http.MethodGet, subIssuesPath, "alice", ""))
		if res.StatusCode != http.StatusOK || body != answer.json+"\n" {
			t.Errorf("GET = %d %s, want 200 %s", res.StatusCode, body, answer.json)
		}
		if want := []string{aliceID.String() + " " + webSeven.ID.String()}; !slices.Equal(list.calls, want) {
			t.Errorf("calls %q, want %q", list.calls, want)
		}
	}
}

// POST goes to the use case for the caller and the path's work item, with
// the body's work items as they came, in their order, one named twice too,
// or none; the answer is 200 with the work items it answers, as GET shows
// them.
func TestAddSubIssues(t *testing.T) {
	for _, tt := range []struct {
		body string
		ids  []uuid.UUID
	}{
		{`{"sub_issue_ids":["0199a2b4-0000-7000-8000-0000000000e8","0199a2b4-0000-7000-8000-0000000000e6",` +
			`"0199a2b4-0000-7000-8000-0000000000e8"]}`, []uuid.UUID{webEight.ID, webSixID, webEight.ID}},
		{`{"sub_issue_ids":[]}`, []uuid.UUID{}},
	} {
		for _, answer := range listed {
			add := &fakeAdd{answer: answer.items}
			res, body := do(t, newServer(t, httpadapter.UseCases{AddSubIssues: add}), request(http.MethodPost, subIssuesPath, "alice", tt.body))
			if res.StatusCode != http.StatusOK || body != answer.json+"\n" {
				t.Errorf("POST %s = %d %s, want 200 %s", tt.body, res.StatusCode, body, answer.json)
			}
			if want := []string{aliceID.String() + " " + webSeven.ID.String()}; !slices.Equal(add.calls, want) ||
				!reflect.DeepEqual(add.got, [][]uuid.UUID{tt.ids}) {
				t.Errorf("POST %s: calls %q with %v; want %q with %v", tt.body, add.calls, add.got, want, tt.ids)
			}
		}
	}
}

// A body without the work items, with null or another type for them, an id
// that is no id, or a field it may not have: refused as bad_request before
// the use case.
func TestAddSubIssuesHoldsTheBodyToItsStructure(t *testing.T) {
	add := &fakeAdd{}
	h := newServer(t, httpadapter.UseCases{AddSubIssues: add})
	for _, body := range []string{`{}`, `{"sub_issue_ids":null}`, `{"sub_issue_ids":"WEB-6"}`, `{"sub_issue_ids":["WEB-6"]}`,
		`{"sub_issue_ids":[],"parent_id":"0199a2b4-0000-7000-8000-0000000000e6"}`} {
		res, got := do(t, h, request(http.MethodPost, subIssuesPath, "alice", body))
		var problem struct{ Code string }
		if err := json.Unmarshal([]byte(got), &problem); err != nil || res.StatusCode != http.StatusBadRequest || problem.Code != "bad_request" {
			t.Errorf("POST %s = %d %s, want 400 bad_request", body, res.StatusCode, got)
		}
	}
	if len(add.calls) != 0 {
		t.Errorf("the use case got %q, want nothing", add.calls)
	}
}

// The use cases' refusals, as the contract declares them, and their
// failure: listSubIssues' are getIssue's; addSubIssues' updateIssue's.
func TestSubIssuesRefusals(t *testing.T) {
	post, both := []string{http.MethodPost}, []string{http.MethodGet, http.MethodPost}
	for _, tt := range []struct {
		name    string
		methods []string
		err     error
		status  int
		want    string
	}{
		{"none named", post, domain.CheckSubIssueIDs(nil), http.StatusUnprocessableEntity,
			`{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
				`"errors":[{"field":"sub_issue_ids","code":"too_short","message":"must name a work item"}]}`},
		{"a work item refused", post, shared.Invalid(domain.SubIssueNotAllowed(1)), http.StatusUnprocessableEntity,
			`{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
				`"errors":[{"field":"sub_issue_ids[1]","code":"not_allowed","message":"must be a work item of the project that is not archived, ` +
				`and neither this one nor above it"}]}`},
		{"no work item", both, domain.ErrIssueNotFound, http.StatusNotFound,
			`{"status":404,"code":"issue.not_found","title":"Not Found","detail":"The work item does not exist, or you cannot see it."}`},
		{"a project guest", both, shared.Forbidden(), http.StatusForbidden,
			`{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
		{"an archived project", post, domain.ErrProjectArchived, http.StatusConflict,
			`{"status":409,"code":"project.archived","title":"Conflict","detail":"The project is archived; unarchive it to change it."}`},
		{"an archived work item", post, domain.ErrIssueArchived, http.StatusConflict,
			`{"status":409,"code":"issue.archived","title":"Conflict","detail":"The work item is archived; unarchive it to change it."}`},
		{"a busy project", post, shared.ServerBusy(time.Second), http.StatusServiceUnavailable, busy},
		{"a failure", both, errGone, http.StatusInternalServerError, `{"status":500,"code":"internal_error","title":"Internal Server Error"}`},
	} {
		for _, method := range tt.methods {
			h := newServer(t, httpadapter.UseCases{ListSubIssues: &fakeList{err: tt.err}, AddSubIssues: &fakeAdd{err: tt.err}})
			body := ""
			if method == http.MethodPost {
				body = `{"sub_issue_ids":["0199a2b4-0000-7000-8000-0000000000e8"]}`
			}
			res, got := do(t, h, request(method, subIssuesPath, "alice", body))
			if res.StatusCode != tt.status || got != tt.want+"\n" || tt.want == busy && res.Header.Get("Retry-After") != "1" {
				t.Errorf("%s: %s = %d %s, Retry-After %q; want %d %s", tt.name, method, res.StatusCode, got, res.Header.Get("Retry-After"),
					tt.status, tt.want)
			}
		}
	}
}
````

`server/internal/modules/issue/module.go`（修改，2 处）：

````old server/internal/modules/issue/module.go
// changing, archiving, unarchiving and deleting a work item. For the
````
````new server/internal/modules/issue/module.go
// changing, archiving, unarchiving and deleting a work item, and listing
// the work items under one and putting work items under it. For the
````
````old server/internal/modules/issue/module.go
		UnarchiveIssue: app.NewUnarchiveIssue(archive),
````
````new server/internal/modules/issue/module.go
		UnarchiveIssue: app.NewUnarchiveIssue(archive),
		ListSubIssues:  app.NewListSubIssues(locks, store),
		AddSubIssues:   app.NewAddSubIssues(app.AddSubIssuesDeps{Locks: locks, Issues: store, Clock: d.Clock}),
````

- [ ] **Step 3: 组合、交错 9 和 13、取锁形状、写闸门和矩阵**

`server/internal/bootstrap/gate_test.go`（修改，7 处）：

````old server/internal/bootstrap/gate_test.go
// Each write P2 adds on a work item enters its project's write gate, as a
// change does (TestAProjectTooBusyForAWriteAnswersServerBusy; M4 design
// 3.5, 11.11): while a write holds Web's one place, each of them on a work
// item of Web is refused at once, the queue full, 503 server_busy with
````
````new server/internal/bootstrap/gate_test.go
// Each write on a work item enters its project's write gate, as a change
// does (TestAProjectTooBusyForAWriteAnswersServerBusy; M4 design 3.5,
// 11.11): while a write holds Web's one place, each of them on a work item
// of Web is refused at once, the queue full, 503 server_busy with
````
````old server/internal/bootstrap/gate_test.go
// item of Web enters the gate and waits for the row. Once the row is free,
// her change goes through, and so does each write refused.
````
````new server/internal/bootstrap/gate_test.go
// item of Web enters the gate and waits for the row. A write refused for
// the shape of its body takes no place at all: an addition of sub-items
// naming a work item twice is refused 422 at once, though Web's one place
// is taken. Once the row is free, her change goes through, and so does
// each write refused.
````
````old server/internal/bootstrap/gate_test.go
		name, method, path string
		want               int // its answer once Web's row is free
````
````new server/internal/bootstrap/gate_test.go
		name, method, path, body string
		want                     int // its answer once Web's row is free
````
````old server/internal/bootstrap/gate_test.go
		{"a deletion", http.MethodDelete, "/api/v0/issues/" + w.newWorkItem(t).String(), http.StatusNoContent},
		{"an archiving", http.MethodPost, "/api/v0/issues/" + w.doneWorkItem(t).String() + "/archive", http.StatusOK},
		{"an unarchiving", http.MethodPost, "/api/v0/issues/" + archived.String() + "/unarchive", http.StatusOK},
````
````new server/internal/bootstrap/gate_test.go
		{"a deletion", http.MethodDelete, "/api/v0/issues/" + w.newWorkItem(t).String(), "", http.StatusNoContent},
		{"an archiving", http.MethodPost, "/api/v0/issues/" + w.doneWorkItem(t).String() + "/archive", "", http.StatusOK},
		{"an unarchiving", http.MethodPost, "/api/v0/issues/" + archived.String() + "/unarchive", "", http.StatusOK},
		{"an addition of sub-items", http.MethodPost, subIssues(w.newWorkItem(t)), subIssueIDs(w.newWorkItem(t)), http.StatusOK},
````
````old server/internal/bootstrap/gate_test.go
		req := newRequest(t, write.method, w.base+write.path, w.tokens["alice"], nil)
````
````new server/internal/bootstrap/gate_test.go
		var body []byte
		if write.body != "" {
			body = []byte(write.body)
		}
		req := newRequest(t, write.method, w.base+write.path, w.tokens["alice"], body)
````
````old server/internal/bootstrap/gate_test.go
		w.wantBusy(t, write.name, req, a.res, a.body)
	}
````
````new server/internal/bootstrap/gate_test.go
		w.wantBusy(t, write.name, req, a.res, a.body)
	}
	for _, shape := range []struct{ name, path, body, want string }{
		{"an addition of sub-items naming one twice", subIssues(item), subIssueIDs(archived, archived), "sub_issue_ids[1] duplicate"},
	} {
		answer := w.send(t, "alice", http.MethodPost, shape.path, shape.body, http.StatusUnprocessableEntity)
		if got := refusalOf(t, []byte(answer), false); got != "validation_failed "+shape.want {
			t.Errorf("%s answers %s while Web's one place is taken; want %s at once, before the gate", shape.name, got, shape.want)
		}
	}
````
````old server/internal/bootstrap/gate_test.go
		w.send(t, "alice", write.method, write.path, "", write.want)
````
````new server/internal/bootstrap/gate_test.go
		w.send(t, "alice", write.method, write.path, write.body, write.want)
````

`server/internal/bootstrap/issue_interleaving_hierarchy_test.go`（修改，3 处）：

````old server/internal/bootstrap/issue_interleaving_hierarchy_test.go
	"uuid"

````
````new server/internal/bootstrap/issue_interleaving_hierarchy_test.go
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
````
````old server/internal/bootstrap/issue_interleaving_hierarchy_test.go
			`{"parent_id":"`+w.workItemNamed(t, parent).String()+`"}`)
	}
````
````new server/internal/bootstrap/issue_interleaving_hierarchy_test.go
			`{"parent_id":"`+w.workItemNamed(t, parent).String()+`"}`)
	}
}

// deletesNamed is name's deletion of the world's work item named item
// (workItemNamed).
func deletesNamed(name, item string) racingWrite {
	return func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
		return w.sending(e, newIssueRoute(t, w.pool, tx), name, http.MethodDelete, "/api/v0/issues/"+w.workItemNamed(t, item).String(), "")
	}
}

// addsUnder is name's addition of Web's work item named child under Web's
// work item named parent (workItemNamed).
func addsUnder(name, child, parent string) racingWrite {
	return func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
		return w.sending(e, newIssueRoute(t, w.pool, tx), name, http.MethodPost, subIssues(w.workItemNamed(t, parent)),
			subIssueIDs(w.workItemNamed(t, child)))
	}
}

// Deleting a work item and putting another under it serialize on Web's
// row (M4 design 3.5, 3.6, 3.9; interleaving 13), in both orders: alice
// deletes Up while bob puts Part under it. Deleted first, holding Web FOR
// NO KEY UPDATE: the addition, which passed its precheck on Up as it was,
// finds no Up under its lock: 404 issue.not_found, and Part is as it was,
// undeleted, under none, without a row of a parent. Added first: the
// deletion waits for Web's row, then reads Up's subtree under its lock,
// Part in it, and deletes both with their rows, bob's row of Part's parent
// among them, seven, by alice, at its one moment, no earlier than the
// addition's commit.
func TestDeletingAWorkItemAndPuttingAnotherUnderItSerialize(t *testing.T) {
	prepare := func(t *testing.T, w issueWorld) {
		w.newWorkItemIn(t, w.web, `{"name":"Up"}`)
		w.newWorkItemIn(t, w.web, `{"name":"Part"}`)
	}
	inBothOrders(t, prepare, "deleted", deletesNamed("alice", "Up"), "added", addsUnder("bob", "Part", "Up"), func(t *testing.T,
		w issueWorld, deletedFirst bool, deleted, added exchange) {
		w.answered(t, deleted, http.StatusNoContent)
		if !deletedFirst {
			w.answered(t, added, http.StatusOK)
			w.deletedWith(t, "issues", w.workItemNamed(t, "Up"), added.released, 7)
			return
		}
		w.refused(t, added, http.StatusNotFound, "issue.not_found")
		part := w.workItemNamed(t, "Part")
		var gone bool
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at IS NOT NULL FROM issues WHERE id = $1", part).Scan(&gone); err != nil {
			t.Fatal(err)
		}
		if gone || w.parentOf(t, part) != nil || len(w.parentRows(t, part)) != 0 {
			t.Errorf("Part is deleted %v, under %v, with the rows of a parent %q; want undeleted, under none, without", gone,
				w.parentOf(t, part), w.parentRows(t, part))
		}
	})
````
````old server/internal/bootstrap/issue_interleaving_hierarchy_test.go
				"one row and none", w.parentOf(t, under), w.parentOf(t, over), w.parentRows(t, under), w.parentRows(t, over), over)
		}
	})
}

````
````new server/internal/bootstrap/issue_interleaving_hierarchy_test.go
				"one row and none", w.parentOf(t, under), w.parentOf(t, over), w.parentRows(t, under), w.parentRows(t, over), over)
		}
	})
}

// Putting a work item under another and moving that other under the first
// serialize on Web's row (M4 design 3.5, 3.9; interleaving 9's addSubIssues
// half), in both orders: alice puts A under B while bob moves B under A.
// The first holds Web FOR NO KEY UPDATE past its writes; the second, which
// passed its precheck, waits for Web's row, and once the first commits
// reads the ancestry under its lock, finds the loop and is refused at its
// place, nothing written: the move 422 at parent_id, the addition 422 at
// sub_issue_ids[0]. The first's work item is under the other, which is
// under none, and only the first writes a row of a parent.
func TestAddingSubItemsAndAMoveThatWouldMakeALoopSerialize(t *testing.T) {
	prepare := func(t *testing.T, w issueWorld) {
		w.newWorkItemIn(t, w.web, `{"name":"A"}`)
		w.newWorkItemIn(t, w.web, `{"name":"B"}`)
	}
	inBothOrders(t, prepare, "A added under B", addsUnder("alice", "A", "B"), "B moved under A", movesUnder("bob", "B", "A"),
		func(t *testing.T, w issueWorld, addedFirst bool, added, moved exchange) {
			under, over := w.workItemNamed(t, "A"), w.workItemNamed(t, "B")
			if addedFirst {
				w.answered(t, added, http.StatusOK)
				if answer := w.answered(t, moved, http.StatusUnprocessableEntity); refusalOf(t, []byte(answer), true) != parentNotAllowed {
					t.Errorf("the move answers %s, want %s", answer, parentNotAllowed)
				}
			} else {
				under, over = over, under
				w.answered(t, moved, http.StatusOK)
				if answer := w.answered(t, added, http.StatusUnprocessableEntity); refusalOf(t, []byte(answer), false) !=
					"validation_failed sub_issue_ids[0] not_allowed" {
					t.Errorf("the addition answers %s, want sub_issue_ids[0] not_allowed", answer)
				}
			}
			if !reflect.DeepEqual(w.parentOf(t, under), &over) || w.parentOf(t, over) != nil || len(w.parentRows(t, under)) != 1 ||
				len(w.parentRows(t, over)) != 0 {
				t.Errorf("the first's work item is under %v, the other under %v, their rows of a parent %q and %q; want under %s, under "+
					"none, one row and none", w.parentOf(t, under), w.parentOf(t, over), w.parentRows(t, under), w.parentRows(t, over), over)
			}
		})
}

````

`server/internal/bootstrap/issue_sub_issues_test.go`（新文件，210 行）：

````file server/internal/bootstrap/issue_sub_issues_test.go
package bootstrap

import (
	"net/http"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"uuid"
)

// subIssues is the path of the sub-items of the work item id.
func subIssues(id uuid.UUID) string {
	return "/api/v0/issues/" + id.String() + "/sub-issues"
}

// subIssueIDs is the body that names the work items ids, in their order.
func subIssueIDs(ids ...uuid.UUID) string {
	quoted := make([]string, len(ids))
	for n, id := range ids {
		quoted[n] = `"` + id.String() + `"`
	}
	return `{"sub_issue_ids":[` + strings.Join(quoted, ",") + `]}`
}

// countedIssue is a work item as a list answers it: its fields
// (storedIssue), and the number of its sub-items the caller sees.
type countedIssue struct {
	storedIssue
	SubIssuesCount int `json:"sub_issues_count"`
}

// listIn are the elements of type T a list answers, in its order.
func listIn[T any](t *testing.T, answer string) []T {
	t.Helper()
	var list struct {
		Data []T `json:"data"`
	}
	decodeAnswer(t, answer, &list)
	if list.Data == nil {
		t.Fatalf("the answer %s has no list", answer)
	}
	return list.Data
}

// itemIDs are the ids of items, in their order.
func itemIDs(items []countedIssue) []uuid.UUID {
	ids := make([]uuid.UUID, len(items))
	for n, i := range items {
		ids[n] = i.ID
	}
	return ids
}

// counts are items, in their order, each as "<name> <number of its
// sub-items the answer shows>".
func counts(items []countedIssue) []string {
	said := make([]string, len(items))
	for n, i := range items {
		said[n] = i.Name + " " + strconv.Itoa(i.SubIssuesCount)
	}
	return said
}

// Sub-items as bootstrap wires them (M4 design 3.5, 3.9, 3.13). alice
// makes Root, Top under Root, Up, A, B under Up, and C, WEB-1 to WEB-6.
// bob puts C, B and A under Top: the answer is A, B and C, in id order,
// each under Top and changed last by bob, and each a row of his activity,
// from none, from WEB-3 for B, to WEB-2. Top lists them, and shows 3
// sub-items; archived, D, which bob puts under it too, is listed by none.
// Putting A under it again answers none and writes nothing; putting A and
// E answers E alone, and does not write A. Naming none, Up twice, or Up
// 101 times is refused, 422, nothing written. Then, once alice puts Root
// under Base, each work item Top may not take is refused at its place, in
// one 422, nothing written: Ops's work item, Top itself, an archived work
// item, Root, above it, Base, above Root, a deleted one, one not there;
// Up, which it may take, beside them, stays under none. An archived work
// item takes none (409), but lists those under it.
func TestWorkItemsArePutUnderAWorkItem(t *testing.T) {
	w := newIssueWorld(t)
	root := w.newWorkItemIn(t, w.web, `{"name":"Root"}`)
	top := w.newWorkItemIn(t, w.web, underParent("Top", root))
	up := w.newWorkItemIn(t, w.web, `{"name":"Up"}`)
	a := w.newWorkItemIn(t, w.web, `{"name":"A"}`)
	b := w.newWorkItemIn(t, w.web, underParent("B", up))
	c := w.newWorkItemIn(t, w.web, `{"name":"C"}`)
	moved := listIn[countedIssue](t, w.send(t, "bob", http.MethodPost, subIssues(top), subIssueIDs(c, b, a), http.StatusOK))
	if got := itemIDs(moved); !slices.Equal(got, []uuid.UUID{a, b, c}) {
		t.Errorf("the answer is %v, want A, B and C, %v", got, []uuid.UUID{a, b, c})
	}
	for _, i := range moved {
		if !reflect.DeepEqual(i.ParentID, &top) || !reflect.DeepEqual(w.parentOf(t, i.ID), &top) || i.UpdatedByID == nil ||
			*i.UpdatedByID != w.ids["bob"] {
			t.Errorf("%s answered under %v, changed last by %v, stored under %v; want under Top, by bob", i.Name, i.ParentID, i.UpdatedByID,
				w.parentOf(t, i.ID))
		}
	}
	for _, tt := range []struct {
		item uuid.UUID
		from string
	}{{a, " → WEB-2, none → "}, {b, "WEB-3 → WEB-2, " + up.String() + " → "}, {c, " → WEB-2, none → "}} {
		if said, want := w.parentRows(t, tt.item), []string{"bob: " + tt.from + top.String()}; !slices.Equal(said, want) {
			t.Errorf("the rows of %s's parent say %q, want %q", tt.item, said, want)
		}
	}
	d := w.doneWorkItem(t)
	w.send(t, "bob", http.MethodPost, subIssues(top), subIssueIDs(d), http.StatusOK)
	w.send(t, "bob", http.MethodPost, "/api/v0/issues/"+d.String()+"/archive", "", http.StatusOK)
	if got := itemIDs(listIn[countedIssue](t, w.send(t, "alice", http.MethodGet, subIssues(top), "", http.StatusOK))); !slices.Equal(got,
		[]uuid.UUID{a, b, c}) {
		t.Errorf("Top lists %v, want A, B and C", got)
	}
	if got := subItems(t, w.send(t, "alice", http.MethodGet, "/api/v0/issues/"+top.String(), "", http.StatusOK)); got != 3 {
		t.Errorf("Top shows %d sub-items, want 3", got)
	}
	_, changed, by := archivalOf(t, w, a)
	if again := listIn[countedIssue](t, w.send(t, "alice", http.MethodPost, subIssues(top), subIssueIDs(a), http.StatusOK)); len(again) != 0 {
		t.Errorf("A put under Top again answers %v, want none", itemIDs(again))
	}
	e := w.newWorkItemIn(t, w.web, `{"name":"E"}`)
	added := listIn[countedIssue](t, w.send(t, "alice", http.MethodPost, subIssues(top), subIssueIDs(a, e), http.StatusOK))
	if got := itemIDs(added); !slices.Equal(got, []uuid.UUID{e}) {
		t.Errorf("A and E put under Top answer %v, want E alone, %s", got, e)
	}
	if _, at, who := archivalOf(t, w, a); !at.Equal(changed) || who != by || len(w.parentRows(t, a)) != 1 {
		t.Errorf("A put under Top again is changed last by %s at %v, with %d rows of its parent; want as before, one row", who, at,
			len(w.parentRows(t, a)))
	}
	for body, refusal := range map[string]string{`{"sub_issue_ids":[]}`: "sub_issue_ids too_short", subIssueIDs(up, up): "sub_issue_ids[1] duplicate",
		subIssueIDs(slices.Repeat([]uuid.UUID{up}, 101)...): "sub_issue_ids too_long"} {
		answer := w.send(t, "bob", http.MethodPost, subIssues(top), body, http.StatusUnprocessableEntity)
		if got := refusalOf(t, []byte(answer), false); got != "validation_failed "+refusal || w.parentOf(t, up) != nil {
			t.Errorf("%.60s answers %s, and Up is under %v; want %s, and under none", body, got, w.parentOf(t, up), refusal)
		}
	}

	far := w.newWorkItemIn(t, createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS"), `{"name":"Far"}`)
	shelved := w.doneWorkItem(t)
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+shelved.String()+"/archive", "", http.StatusOK)
	gone := w.newWorkItemIn(t, w.web, `{"name":"Gone"}`)
	w.send(t, "alice", http.MethodDelete, "/api/v0/issues/"+gone.String(), "", http.StatusNoContent)
	base := w.newWorkItemIn(t, w.web, `{"name":"Base"}`)
	w.send(t, "alice", http.MethodPatch, "/api/v0/issues/"+root.String(), `{"parent_id":"`+base.String()+`"}`, http.StatusOK)
	answer := w.send(t, "bob", http.MethodPost, subIssues(top), subIssueIDs(far, top, shelved, root, base, gone, uuid.NewV7(), up),
		http.StatusUnprocessableEntity)
	refused := []string{}
	for _, field := range []string{"[0]", "[1]", "[2]", "[3]", "[4]", "[5]", "[6]"} {
		refused = append(refused, "sub_issue_ids"+field+" not_allowed")
	}
	if got, want := refusalOf(t, []byte(answer), false), "validation_failed "+strings.Join(refused, "; "); got != want ||
		w.parentOf(t, up) != nil {
		t.Errorf("the work items Top may not take answer %s, and Up is under %v; want %s, and under none", got, w.parentOf(t, up), want)
	}
	if answer := w.send(t, "bob", http.MethodPost, subIssues(shelved), subIssueIDs(up), http.StatusConflict); refusalOf(t, []byte(answer),
		false) != "issue.archived" || w.parentOf(t, up) != nil {
		t.Errorf("Up under an archived work item answers %s, and is under %v; want issue.archived, under none", answer, w.parentOf(t, up))
	}
	if got := listIn[countedIssue](t, w.send(t, "bob", http.MethodGet, subIssues(shelved), "", http.StatusOK)); len(got) != 0 {
		t.Errorf("an archived work item lists %v, want none", itemIDs(got))
	}
}

// The guest rule holds for sub-items (M4 design 3.4, 3.9): walt, acme's
// admin and Web's guest (adminGuest), sees only his own work items in Web.
// He makes His and Mine, alice Hers between them; under Mine, alice makes
// Low and walt Below. walt puts Mine under His, answered with the one
// sub-item he sees; Hers, which he does not see, is refused at its place
// with the very answer of a work item not there, and stays under none;
// alice puts it there. walt lists Mine alone under His, with its one
// sub-item, and alice and bob Hers and Mine, with its two; Up, alice's,
// walt finds neither to list nor to put a work item under: the very 404
// of a work item not there.
func TestTheGuestRuleHoldsForSubItems(t *testing.T) {
	w := newIssueWorld(t)
	w.adminGuest(t)
	up := w.newWorkItemIn(t, w.web, `{"name":"Up"}`)
	his := w.newWorkItemBy(t, "walt", w.web, `{"name":"His"}`)
	hers := w.newWorkItemIn(t, w.web, `{"name":"Hers"}`)
	mine := w.newWorkItemBy(t, "walt", w.web, `{"name":"Mine"}`)
	w.newWorkItemIn(t, w.web, underParent("Low", mine))
	w.newWorkItemBy(t, "walt", w.web, underParent("Below", mine))
	added := listIn[countedIssue](t, w.send(t, "walt", http.MethodPost, subIssues(his), subIssueIDs(mine), http.StatusOK))
	if got := counts(added); !slices.Equal(got, []string{"Mine 1"}) {
		t.Errorf("walt's putting Mine under His answers %q, want Mine with the one sub-item he sees", got)
	}
	answer := w.send(t, "walt", http.MethodPost, subIssues(his), subIssueIDs(hers), http.StatusUnprocessableEntity)
	notThere := w.send(t, "walt", http.MethodPost, subIssues(his), subIssueIDs(uuid.NewV7()), http.StatusUnprocessableEntity)
	if got := refusalOf(t, []byte(answer), false); got != "validation_failed sub_issue_ids[0] not_allowed" || answer != notThere ||
		w.parentOf(t, hers) != nil {
		t.Errorf("walt's putting Hers under His answers %s, and Hers is under %v; want sub_issue_ids[0] not_allowed, as for one not there: %s",
			answer, w.parentOf(t, hers), notThere)
	}
	w.send(t, "alice", http.MethodPost, subIssues(his), subIssueIDs(hers), http.StatusOK)
	for _, tt := range []struct {
		name string
		want []string
	}{{"walt", []string{"Mine 1"}}, {"alice", []string{"Hers 0", "Mine 2"}}, {"bob", []string{"Hers 0", "Mine 2"}}} {
		listed := listIn[countedIssue](t, w.send(t, tt.name, http.MethodGet, subIssues(his), "", http.StatusOK))
		if got := counts(listed); !slices.Equal(got, tt.want) {
			t.Errorf("%s lists %q under His, want %q", tt.name, got, tt.want)
		}
	}
	for _, tt := range []struct{ method, body string }{{http.MethodGet, ""}, {http.MethodPost, subIssueIDs(mine)}} {
		hidden := w.send(t, "walt", tt.method, subIssues(up), tt.body, http.StatusNotFound)
		if absent := w.send(t, "walt", tt.method, subIssues(uuid.NewV7()), tt.body, http.StatusNotFound); hidden != absent {
			t.Errorf("walt's %s of Up's sub-items answers %s, want as for a work item not there: %s", tt.method, hidden, absent)
		}
	}
}
````

`server/internal/bootstrap/issue_write_locks_test.go`（修改，8 处）：

````old server/internal/bootstrap/issue_write_locks_test.go
// its body, which names the world's rows; the answer it gets; waits, the
// table it waits for under its lock before any write of its own: the
// first it reads there that its precheck did not (held ACCESS EXCLUSIVE),
// or, when it reads none, the first it writes (held SHARE, which its
// precheck's plain read does not wait for); and mode, the lock another
// transaction holds of that table: ACCESS EXCLUSIVE, which a plain read
// waits for too, or SHARE, which a write waits for and a plain read does
// not, so that the write waits there and its precheck does not.
````
````new server/internal/bootstrap/issue_write_locks_test.go
// its body, which names the world's rows, read as the test sends it, with
// a plain read, which no lock the test holds stops; the answer it gets;
// waits, the table it waits for under its lock before any write of its
// own: the first it reads there that its precheck did not (held ACCESS
// EXCLUSIVE), or, when it reads none, the first it writes (held SHARE,
// which its precheck's plain read does not wait for); and mode, the lock
// another transaction holds of that table: ACCESS EXCLUSIVE, which a plain
// read waits for too, or SHARE, which a write waits for and a plain read
// does not, so that the write waits there and its precheck does not.
````
````old server/internal/bootstrap/issue_write_locks_test.go
	body             func(w issueWorld) string
````
````new server/internal/bootstrap/issue_write_locks_test.go
	body             func(t *testing.T, w issueWorld) string
````
````old server/internal/bootstrap/issue_write_locks_test.go
		body:   func(w issueWorld) string { return w.fullIssue() }, want: http.StatusCreated, waits: "states", mode: "ACCESS EXCLUSIVE"},
````
````new server/internal/bootstrap/issue_write_locks_test.go
		body:   func(_ *testing.T, w issueWorld) string { return w.fullIssue() }, want: http.StatusCreated, waits: "states", mode: "ACCESS EXCLUSIVE"},
````
````old server/internal/bootstrap/issue_write_locks_test.go
		body:   func(w issueWorld) string { return `{"state_id":"` + w.done.String() + `"}` }, want: http.StatusOK, waits: "states",
		mode: "ACCESS EXCLUSIVE"},
````
````new server/internal/bootstrap/issue_write_locks_test.go
		body:   func(_ *testing.T, w issueWorld) string { return `{"state_id":"` + w.done.String() + `"}` }, want: http.StatusOK,
		waits: "states", mode: "ACCESS EXCLUSIVE"},
````
````old server/internal/bootstrap/issue_write_locks_test.go
		body:   func(issueWorld) string { return "" }, want: http.StatusNoContent, waits: "issues", mode: "SHARE"},
````
````new server/internal/bootstrap/issue_write_locks_test.go
		body:   func(*testing.T, issueWorld) string { return "" }, want: http.StatusNoContent, waits: "issues", mode: "SHARE"},
````
````old server/internal/bootstrap/issue_write_locks_test.go
		body: func(issueWorld) string { return "" }, want: http.StatusOK, waits: "states", mode: "ACCESS EXCLUSIVE"},
````
````new server/internal/bootstrap/issue_write_locks_test.go
		body: func(*testing.T, issueWorld) string { return "" }, want: http.StatusOK, waits: "states", mode: "ACCESS EXCLUSIVE"},
````
````old server/internal/bootstrap/issue_write_locks_test.go
		body: func(issueWorld) string { return "" }, want: http.StatusOK, waits: "issues", mode: "SHARE"},
````
````new server/internal/bootstrap/issue_write_locks_test.go
		body: func(*testing.T, issueWorld) string { return "" }, want: http.StatusOK, waits: "issues", mode: "SHARE"},
	// A new work item, which the send puts Part under: Web's one work item
	// named so, made with the first target; each send moves it under the
	// next. It reads nothing under the lock that its precheck did not, so it
	// waits for the work items' table, which it writes first.
	{op: "addSubIssues", method: http.MethodPost, path: "/api/v0/issues/{issue_id}/sub-issues",
		target: func(t *testing.T, w issueWorld) string {
			if len(queryIDs(t, w.pool, "SELECT id FROM issues WHERE project_id = $1 AND name = 'Part'", w.web)) == 0 {
				w.newWorkItemIn(t, w.web, `{"name":"Part"}`)
			}
			return subIssues(w.newWorkItem(t))
		},
		body: func(t *testing.T, w issueWorld) string { return subIssueIDs(w.workItemNamed(t, "Part")) }, want: http.StatusOK,
		waits: "issues", mode: "SHARE"},
````
````old server/internal/bootstrap/issue_write_locks_test.go
				req := newRequest(t, iw.method, w.base+target, w.tokens[name], []byte(iw.body(w)))
````
````new server/internal/bootstrap/issue_write_locks_test.go
				req := newRequest(t, iw.method, w.base+target, w.tokens[name], []byte(iw.body(t, w)))
````

`server/internal/bootstrap/issue_writes_test.go`（修改，2 处）：

````old server/internal/bootstrap/issue_writes_test.go
// under the second, the second's ancestry read under it. She archives the
// first, in Done, and unarchives it; she deletes the second, and Part
// under it. With Web archived, her creation and her change are refused
// under the lock (409).
````
````new server/internal/bootstrap/issue_writes_test.go
// under the second, the second's ancestry read under it, and puts Low
// under Part. She archives the first, in Done, and unarchives it; she
// deletes the second, and Part and Low under it. With Web archived, her
// creation and her change are refused under the lock (409).
````
````old server/internal/bootstrap/issue_writes_test.go
	route.answerWithin(t, w.contract, http.MethodPatch, "/api/v0/issues/"+part.ID.String(), alice, `{"parent_id":"`+defaulted.ID.String()+`"}`,
		http.StatusOK)
````
````new server/internal/bootstrap/issue_writes_test.go
	route.answerWithin(t, w.contract, http.MethodPatch, "/api/v0/issues/"+part.ID.String(), alice, `{"parent_id":"`+defaulted.ID.String()+`"}`,
		http.StatusOK)
	var low struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, send(alice, `{"name":"Low"}`, http.StatusCreated), &low)
	route.answerWithin(t, w.contract, http.MethodPost, subIssues(part.ID), alice, subIssueIDs(low.ID), http.StatusOK)
````

`server/internal/bootstrap/permission_matrix_sub_issues_test.go`（新文件，103 行）：

````file server/internal/bootstrap/permission_matrix_sub_issues_test.go
package bootstrap

import (
	"net/http"
	"reflect"
	"testing"
	"uuid"
)

// The rows of the permission matrix of a work item's sub-items (M4 design
// 3.9, 9.2), naming the work item by its id: its column's work item
// (issueOf). A row of 9.2's table has the issue table's columns; a variant
// that asks what is checked after the decision, the project level's, as
// updateIssue's; one on the archived project, its table's.

// subItemOf is the work item a column's cells put under their column's
// work item: acme's public project's work item of PM, the member who made
// it, or, for his own column, whose work item that is, the project's
// matrixIssue, acme's admin's. Only the public project's cells weigh it.
func subItemOf(c caller, s seeded) uuid.UUID {
	name := issueOf(callerCreator)
	if c == callerCreator {
		name = matrixIssue
	}
	return s.issue("acme/public", name)
}

// addsSubItem is the request of a row whose callers each put a work item
// under their column's work item: subItemOf, or the one item gives.
func addsSubItem(item func(c caller, s seeded) uuid.UUID) func(caller, seeded) (string, string, string) {
	return func(c caller, s seeded) (string, string, string) {
		return toWorkItem(http.MethodPost, "/sub-issues", `{"sub_issue_ids":["`+item(c, s).String()+`"]}`)(c, s)
	}
}

func subIssueMatrixRows() []matrixRow {
	list := toWorkItem(http.MethodGet, "/sub-issues", "")
	// The workspace admin who is only a guest of the project may put work
	// items under his own, as its admin (Grant.ProjectAdmin), but not PM's,
	// which the guest rule hides from him: refused at its place (M4 design
	// 3.4).
	adds := onAWorkItem(cellOK, cellOK, cellOK, cellForbidden)
	adds[callerAdminGuestOwn] = cellValidationFailed
	refused := issueRows.of(cellValidationFailed, cellValidationFailed, cellIssueNotFound, cellValidationFailed, cellForbidden, cellForbidden)
	return []matrixRow{
		// Whom getIssue answers the work item (M4 design 3.4, 9.2): its
		// project's admins, members and the guests who see it.
		{op: "listSubIssues", columns: issueColumns, request: list, cells: onAWorkItem(cellOK, cellOK, cellOK, cellOK), check: listsNone},
		// An archived project's work items' are listed as any other's.
		{op: "listSubIssues", variant: "archived", columns: archivedColumns, request: list, cells: issueRows.ofArchived(cellOK, cellForbidden),
			check: listsNone},
		// The project's admins and members, the member who made it too, not
		// its guests, as updateIssue (M4 design 3.4, 9.2).
		{op: "addSubIssues", write: true, columns: issueColumns, request: addsSubItem(subItemOf), cells: adds,
			refusal: "sub_issue_ids[0] not_allowed", check: putsUnder},
		// An archived project's work items take none under them (M4 design
		// 3.8), after the decision.
		{op: "addSubIssues", variant: "archived", write: true, columns: archivedColumns, request: addsSubItem(subItemOf),
			cells: issueRows.ofArchived(cellProjectArchived, cellForbidden)},
		// Nor do its archived work items: the project's 409 comes first.
		{op: "addSubIssues", variant: "archived, its work item archived", write: true, columns: archivedColumns, prepare: inDone(true),
			request: addsSubItem(subItemOf), cells: issueRows.ofArchived(cellProjectArchived, cellForbidden)},
		// Nor does an archived work item, after the decision, before the
		// work items asked for are looked at.
		{op: "addSubIssues", variant: "archived work item", write: true, columns: issueColumns, prepare: inDone(true),
			request: addsSubItem(subItemOf), cells: onAWorkItem(cellIssueArchived, cellIssueArchived, cellIssueArchived, cellForbidden)},
		// The archived project's work item, another project's of the same
		// workspace: the work items asked for are checked after the
		// decision.
		{op: "addSubIssues", variant: "another project's work item", write: true, columns: projectColumns,
			request: addsSubItem(func(_ caller, s seeded) uuid.UUID { return s.issue("acme/archived", matrixIssue) }), cells: refused,
			refusal: "sub_issue_ids[0] not_allowed"},
	}
}

// listsNone: no seeded work item has a sub-item.
func listsNone(t *testing.T, c caller, _ seeded, answer string) {
	var got struct {
		Data []answeredIssue `json:"data"`
	}
	decodeAnswer(t, answer, &got)
	if got.Data == nil || len(got.Data) != 0 {
		t.Errorf("%s lists %s; want none", c, answer)
	}
}

// putsUnder: the column's sub-item (subItemOf) alone, under the column's
// work item, changed last by the caller.
func putsUnder(t *testing.T, c caller, s seeded, answer string) {
	var got struct {
		Data []struct {
			ID          uuid.UUID  `json:"id"`
			ParentID    *uuid.UUID `json:"parent_id"`
			UpdatedByID *uuid.UUID `json:"updated_by_id"`
		} `json:"data"`
	}
	decodeAnswer(t, answer, &got)
	parent, me := s.issue(projectOf(c), issueOf(c)), s.account(accountOf(c))
	if len(got.Data) != 1 || got.Data[0].ID != subItemOf(c, s) || !reflect.DeepEqual(got.Data[0].ParentID, &parent) ||
		!reflect.DeepEqual(got.Data[0].UpdatedByID, &me) {
		t.Errorf("%s puts %s; want %s alone, under %s, changed last by the caller", c, answer, subItemOf(c, s), parent)
	}
}
````

`server/internal/bootstrap/permission_matrix_test.go`（修改，1 处）：

````old server/internal/bootstrap/permission_matrix_test.go
		labelMatrixRows(), issueMatrixRows(), issueUpdateMatrixRows(), issueArchiveMatrixRows())
````
````new server/internal/bootstrap/permission_matrix_test.go
		labelMatrixRows(), issueMatrixRows(), issueUpdateMatrixRows(), issueArchiveMatrixRows(), subIssueMatrixRows())
````

- [ ] **Step 4: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/issue/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestWorkItemsArePutUnderAWorkItem$|TestTheGuestRuleHoldsForSubItems$|TestDeletingAWorkItemAndPuttingAnotherUnderItSerialize$|TestAddingSubItemsAndAMoveThatWouldMakeALoopSerialize$|TestEachIssueWriteLocksItsProjectFirst$|TestEachWriteOnAWorkItemOfAProjectTooBusyAnswersServerBusy$|TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection$|TestPermissionMatrix|TestThePermissionMatrixCoversEveryOperation$' ./internal/bootstrap/`
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
git add api/modules/issue.yaml api/openapi.yaml server/internal/bootstrap/gate_test.go server/internal/bootstrap/issue_interleaving_hierarchy_test.go server/internal/bootstrap/issue_sub_issues_test.go server/internal/bootstrap/issue_write_locks_test.go server/internal/bootstrap/issue_writes_test.go server/internal/bootstrap/permission_matrix_sub_issues_test.go server/internal/bootstrap/permission_matrix_test.go server/internal/modules/issue/adapter/http/handler.go server/internal/modules/issue/adapter/http/sub_issues.go server/internal/modules/issue/adapter/http/sub_issues_test.go server/internal/modules/issue/module.go api/dist/openapi.yaml server/internal/modules/issue/adapter/http/gen/bodyshape.gen.go server/internal/modules/issue/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M4/P3): listSubIssues and addSubIssues, with the matrix and interleavings 9 and 13

GET and POST /api/v0/issues/{issue_id}/sub-issues answer an
IssueCollection of Issue rows, the shape of IssueDetail in P3. The
addition enters its project's write gate and locks its project first;
the guest rule hides another's work item as one that is not there, and
a body of the wrong shape is refused before any read, taking no place.
Deleting a work item and putting another under it serialize on the
project's row: the addition finds the parent gone, or the deletion finds
the new sub-item in its subtree and deletes it at its one moment. An
addition and a move that together would make a loop serialize there
too: the second reads the ancestry under the lock and is refused.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p3.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试，或没有替身的平台、领域、转换的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；守卫是生成器（`make gen-go`）和中英文翻译键（`make lint-web`）的核对。每个变异在一份新的副本里，在本 Task 的树上、标出的较晚的 Task 的树上和最终的树上都跑过，端到端的检查只在标出的 Task 的树上）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T6.1` | 没有一项的集合答 `null` | `internal/modules/issue/adapter/http` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWorkItemsArePutUnderAWorkItem`（`make test`） | 单元；组合 |
| `T6.2` | `POST` 交给用例的工作项是空的 | `internal/modules/issue/adapter/http` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWorkItemsArePutUnderAWorkItem`（`make test`） | 单元；组合 |
| `T6.3` | 契约没有写 `addSubIssues` 回答 `issue.archived` | `internal/modules/issue/adapter/http` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWorkItemsArePutUnderAWorkItem`（`make test`） | 单元；组合 |
| `T6.4` | `addSubIssues` 的请求体收它没有列出的字段 | `internal/modules/issue/adapter/http` 的全部测试（`make test`） | 单元 |

本 Task 的检查也发现较早的 Task 的变异 `T1.1`、`T2.3`、`T2.4`、`T3.2`、`T4.14`、`T5.1`、`T5.2`、`T5.3`、`T5.4`、`T5.5`、`T5.6`、`T5.8`、`T5.9`、`T5.10`、`T5.12`、`T5.13`、`T5.14`、`T5.15`、`T5.16`、`T5.18`、`T5.19`、`T5.20`、`T5.22`、`T5.23`、`T5.24`、`T5.25`、`T5.26`、`T5.29`、`T5.30`（列在各自的 Task 的表里，标"（Task 6 起）"）：本 Task 做完之后，在本 Task 的树上重跑它们（`mutants_p3.py`，`MUT_LATER=6`）。

---

### Task 7: 多个项目的锁和写闸门：`LockMany`、`Keyed.Enter(ctx, keys...)`

**Files:**
- Create: `server/internal/modules/issue/app/fakes_across_test.go`、`server/internal/modules/project/adapter/postgres/lock_issue_projects_test.go`、`server/internal/platform/gate/keys_test.go`
- Modify: `server/internal/bootstrap/gate.go`、`server/internal/bootstrap/gate_test.go`、`server/internal/bootstrap/ports.go`、`server/internal/bootstrap/ports_test.go`、`server/internal/modules/issue/app/fakes_test.go`、`server/internal/modules/issue/app/ports.go`、`server/internal/modules/project/adapter/postgres/failures_test.go`、`server/internal/modules/project/adapter/postgres/issue_ports.go`、`server/internal/modules/project/adapter/postgres/queries/issue_ports.sql`、`server/internal/modules/project/module.go`、`server/internal/platform/config/config.go`、`server/internal/platform/gate/gate.go`、`server/internal/platform/gate/gate_test.go`
- Generate: `server/internal/modules/project/adapter/postgres/gen/issue_ports.sql.go`

**Interfaces:**（spec 2.9、第 3 节第 2、3 条；M4 设计 3.5 约定七、6.5、9.1、11.11；P1b spec 第 3 节第 10 条和第 5 节的 P3 一行）跨项目的写（Task 10 的加关联、Task 13 的删除关联）要的两样东西，先在这里各自钉住：
- `platform/gate/gate.go`：`Keyed.Enter(ctx, keys...)`：按给的次序一个一个进（`enter`），一个 `MaxWait` 的期限管全部的键（不是每个键一个）；等一个键时只持着已进的键，不持别的；任一个失败（队列满、到了期限、`ctx` 结束）时离开已进的键（`leave`），什么都不持；`leave` 只做一次。一个键的调用照旧。
- `platform/config/config.go`：`max_wait` 的说明写上跨项目的写在它的几个闸门上一共等这么久。
- `issue/app/ports.go`：`ProjectGate.LockMany(ctx, projectIDs) (map[uuid.UUID]LockedProject, error)`（一条语句、`id` 升序锁住未删除的；等锁期间被删除的不在其中）；`WriteGate.Enter(ctx, projectIDs...)`（各一次、`id` 升序，按这个次序进，等的时候不持连接、不持锁，只持着前面的闸门的位置）。`ProjectGate.Lock` 留着（一个项目的写照旧，设计 3.3、6.5 列出 `Find`、`Lock`、`LockMany` 三个）。
- `project`：`queries/issue_ports.sql` 的 `LockIssueProjects`（`WHERE id = ANY … AND deleted_at IS NULL ORDER BY id FOR NO KEY UPDATE`：按排好的行取锁，次序就是 `id` 的次序，S5 的 E4）；`issue_ports.go`；`module.go` 的 `IssueProjects.LockIssueProjects`。
- `bootstrap`：`gate.go` 的 `writeGate.Enter(ctx, projectIDs...)`；`ports.go` 的 `issueProjects.LockMany`（逐个转换）。
- 测试的帮手：`fakes_test.go` 的 `fakeGate` 记下给的全部项目；`fakes_across_test.go`（新）：`anys`、`fakeProject.LockMany`（从 `across` 回答，记一行）。

**Tests:**
- 单元（`platform/gate`）：`TestACallerHoldsEachKeyHeEnters`（持 api、web 的调用者让两个键的别的调用者都等；第二个键被占时他持着第一个等，第一个键的新调用者排在他后面；都离开之后什么都不留）、`TestACallerWhoDoesNotGetEachKeyHoldsNone`（第二个键的队列满、到了 `MaxWait`、`ctx` 取消：放回第一个，它的调用者立刻进，他什么都不持）、`TestACallerWaitsMaxWaitInAll`（等了 api 四分之三个 `MaxWait` 的调用者在从开始算起的 `MaxWait` 放弃 web，不是再等一个 `MaxWait`；`MaxWait` 一秒，两者相隔四分之三秒，回答要在 `MaxWait` 或之后、两者的中点之前，负载重的机器上也分得出）。组合一层要两个闸门都排着、各在 `max_wait` 之内放开才分得出一个期限和每个键一个期限：Task 11 的 `TestAWriteAcrossProjectsWaitsMaxWaitInAll`。
- 存储（`project`）：`TestLockIssueProjects`（所问的未删除的，已归档的也在，按 `id` 回答；已删除的、没有的不在；每个持 `FOR NO KEY UPDATE` 到事务结束：`FOR SHARE` 等它，外键的 `FOR KEY SHARE` 不等；没问的 Ops 和工作区的行不持）、`TestLockIssueProjectsLocksInIDOrder`（Web 的 `id` 小，但在表里和名字、标识的索引里在 Alpha 之后，又后问它；持着 Alpha 的行时，它持着 Web 等 Alpha：别的次序会先到 Alpha、什么都不持地等）、`TestLockIssueProjectsLeavesOutAProjectDeletedWhileItWaited`（等 Web 的行期间它被删除：提交之后只回答 Ops）、`TestAFailedReadIsAnErrorNotAnAnswer` 加 `LockIssueProjects` 一行。
- 组合：`TestTheWriteGateAnswersItsRefusalsAsServerBusy` 加 `across`（同时写别的项目和这个项目的写一样答 503，什么都不持：之后别的项目的写立刻进）；`TestIssueProjectsConvertsProjectsAnswers` 加 `LockMany` 的转换。

- [ ] **Step 1: 查询，生成**

`server/internal/modules/project/adapter/postgres/queries/issue_ports.sql`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/queries/issue_ports.sql
FROM projects
WHERE id = sqlc.arg(id) AND deleted_at IS NULL
````
````new server/internal/modules/project/adapter/postgres/queries/issue_ports.sql
FROM projects
WHERE id = sqlc.arg(id) AND deleted_at IS NULL
FOR NO KEY UPDATE;

-- name: LockIssueProjects :many
-- ProjectGate.LockMany (M4 design 6.5): the first locks of a write across projects (convention 7, 3.5), the undeleted
-- projects among ids FOR NO KEY UPDATE, as LockIssueProject, in one statement, in id order. The lock is taken as the
-- sorted rows come, so the order is the ids' (S5 E4). After a wait, Postgres evaluates deleted_at IS NULL again on the
-- row's newest version: a project deleted meanwhile is left out.
SELECT id, workspace_id, (archived_at IS NOT NULL)::boolean AS archived, default_assignee_id, identifier
FROM projects
WHERE id = ANY (sqlc.arg(ids)::uuid[]) AND deleted_at IS NULL
ORDER BY id
````

Run: `make gen-go`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `cf5a24b5fe04723d38b2385e00e45b01eb433020a311810b0a9896a07da8b68f` | 259 | `server/internal/modules/project/adapter/postgres/gen/issue_ports.sql.go` |

Run: `shasum -a 256 server/internal/modules/project/adapter/postgres/gen/issue_ports.sql.go`
Expected: 与上表相同。

- [ ] **Step 2: 平台的闸门**

`server/internal/platform/config/config.go`（修改，1 处）：

````old server/internal/platform/config/config.go
// most MaxWait; a write beyond answers 503 server_busy.
````
````new server/internal/platform/config/config.go
// most MaxWait, a write across projects for their gates in all; a write
// beyond answers 503 server_busy.
````

`server/internal/platform/gate/gate.go`（修改，10 处）：

````old server/internal/platform/gate/gate.go
// process (M4 design 3.5, 11.11): a caller enters a key's gate, waits in
// the key's queue, first come first served, while as many others hold it
// as its bound allows, and leaves it once. A key nobody holds or waits for
// takes no memory. It uses only the standard library.
````
````new server/internal/platform/gate/gate.go
// process (M4 design 3.5, 11.11): a caller enters a key's gate, or several
// keys' one after another, waits in each key's queue, first come first
// served, while as many others hold it as its bound allows, and leaves
// them once. A key nobody holds or waits for takes no memory. It uses only
// the standard library.
````
````old server/internal/platform/gate/gate.go
// Enter returns once the caller holds key, with the function that leaves
// it, which the caller calls once on every path, and which does nothing
// when called again. While PerKey callers hold key, or others wait for it
// already, the caller waits behind them, holding nothing; it is refused at
// once with ErrFull when MaxQueue others wait, and gives up with
// ErrTimeout after MaxWait, or with ctx's error once ctx is done, before
// or while it waits: then it holds nothing and leaves the queue.
func (g *Keyed[K]) Enter(ctx context.Context, key K) (leave func(), err error) {
````
````new server/internal/platform/gate/gate.go
// Enter returns once the caller holds each of keys, which it enters one
// after another, in their order, with the function that leaves them all,
// which the caller calls once on every path, and which does nothing when
// called again. While PerKey callers hold a key, or others wait for it
// already, the caller waits behind them, holding only the keys it entered
// before it; it is refused at once with ErrFull when MaxQueue others wait,
// and gives up with ErrTimeout once it has waited MaxWait in all, or with
// ctx's error once ctx is done, before or while it waits: then it leaves
// the keys it entered, and holds nothing. Callers that enter several keys
// enter them in one order, each once (the write gate: by project id, M4
// design 11.11), so that none waits for a key whose holder waits for one
// it holds.
func (g *Keyed[K]) Enter(ctx context.Context, keys ...K) (leave func(), err error) {
````
````old server/internal/platform/gate/gate.go
		return nil, err
	}
	g.mu.Lock()
````
````new server/internal/platform/gate/gate.go
		return nil, err
	}
	keys = slices.Clone(keys)
	deadline := time.Now().Add(g.limits.MaxWait)
	for n, key := range keys {
		if err := g.enter(ctx, key, deadline); err != nil {
			g.leave(keys[:n])
			return nil, err
		}
	}
	var once sync.Once
	return func() { once.Do(func() { g.leave(keys) }) }, nil
}

// enter takes key for the caller: at once when fewer than PerKey hold it
// and no one waits for it; else it waits behind those who do until it is
// handed the key, or gives up, as Enter says, at deadline or once ctx is
// done, holding nothing of key and out of its queue.
func (g *Keyed[K]) enter(ctx context.Context, key K, deadline time.Time) error {
	g.mu.Lock()
````
````old server/internal/platform/gate/gate.go
		g.mu.Unlock()
		return g.leaver(key), nil
````
````new server/internal/platform/gate/gate.go
		g.mu.Unlock()
		return nil
````
````old server/internal/platform/gate/gate.go
		return nil, ErrFull
````
````new server/internal/platform/gate/gate.go
		return ErrFull
````
````old server/internal/platform/gate/gate.go
	timer := time.NewTimer(g.limits.MaxWait)
	defer timer.Stop()
````
````new server/internal/platform/gate/gate.go
	timer := time.NewTimer(time.Until(deadline))
	defer timer.Stop()
	var err error
````
````old server/internal/platform/gate/gate.go
	case <-w.ready:
		return g.leaver(key), nil
````
````new server/internal/platform/gate/gate.go
	case <-w.ready:
		return nil
````
````old server/internal/platform/gate/gate.go
		g.release(key)
		return nil, err
````
````new server/internal/platform/gate/gate.go
		g.release(key)
		return err
````
````old server/internal/platform/gate/gate.go
	g.mu.Unlock()
	return nil, err
````
````new server/internal/platform/gate/gate.go
	g.mu.Unlock()
	return err
````
````old server/internal/platform/gate/gate.go
// leaver is the function that leaves key once.
func (g *Keyed[K]) leaver(key K) func() {
	var once sync.Once
	return func() { once.Do(func() { g.release(key) }) }
````
````new server/internal/platform/gate/gate.go
// leave gives up each of keys (release).
func (g *Keyed[K]) leave(keys []K) {
	for _, key := range keys {
		g.release(key)
	}
````

`server/internal/platform/gate/gate_test.go`（修改，2 处）：

````old server/internal/platform/gate/gate_test.go
// enter calls g.Enter(ctx, key) on a goroutine of its own.
func enter(g *Keyed[string], ctx context.Context, key string) *entry {
````
````new server/internal/platform/gate/gate_test.go
// enter calls g.Enter(ctx, keys...) on a goroutine of its own.
func enter(g *Keyed[string], ctx context.Context, keys ...string) *entry {
````
````old server/internal/platform/gate/gate_test.go
		e.leave, e.err = g.Enter(ctx, key)
````
````new server/internal/platform/gate/gate_test.go
		e.leave, e.err = g.Enter(ctx, keys...)
````

`server/internal/platform/gate/keys_test.go`（新文件，92 行）：

````file server/internal/platform/gate/keys_test.go
package gate

import (
	"context"
	"errors"
	"testing"
	"time"
)

// A caller who enters several keys holds each: while he holds api and web,
// a caller of either waits, and once he leaves, each holds the key he
// waits for. One who finds his second key held waits for it holding his
// first, before which a caller of it then waits too; once the second is
// free, he holds both. Once every caller has left, the gate keeps nothing.
func TestACallerHoldsEachKeyHeEnters(t *testing.T) {
	g := New[string](Limits{PerKey: 1, MaxQueue: 8, MaxWait: long})
	ctx := context.Background()
	both := enter(g, ctx, "api", "web").held(t)
	api, web := enter(g, ctx, "api"), enter(g, ctx, "web")
	api.waits(t)
	web.waits(t)
	both.leave()
	api.held(t).leave()
	web.held(t)
	second := enter(g, ctx, "api", "web")
	queued(t, g, "web", 1)
	behind := enter(g, ctx, "api")
	behind.waits(t)
	web.leave()
	second.held(t).leave()
	behind.held(t).leave()
	forgotten(t, g)
}

// A caller of api and web who does not get web, its queue full, his wait
// at MaxWait, or his context cancelled while he waits, leaves api, which a
// caller of it then holds at once, and holds nothing; once web's holder
// leaves, the gate keeps nothing.
func TestACallerWhoDoesNotGetEachKeyHoldsNone(t *testing.T) {
	for _, tt := range []struct {
		name     string
		maxQueue int
		maxWait  time.Duration
		cancels  bool // whether the test cancels his context once he waits
		want     error
	}{
		{"its queue full", 0, long, false, ErrFull},
		{"waited MaxWait", 8, 30 * time.Millisecond, false, ErrTimeout},
		{"cancelled while he waits", 8, long, true, context.Canceled},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g := New[string](Limits{PerKey: 1, MaxQueue: tt.maxQueue, MaxWait: tt.maxWait})
			web := enter(g, context.Background(), "web").held(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			gaveUp := enter(g, ctx, "api", "web")
			if tt.cancels {
				queued(t, g, "web", 1)
				cancel()
			}
			if gaveUp.returned(t); !errors.Is(gaveUp.err, tt.want) || gaveUp.leave != nil {
				t.Errorf("Enter() = %v, want %v and nothing to leave", gaveUp.err, tt.want)
			}
			enter(g, context.Background(), "api").held(t).leave()
			web.leave()
			forgotten(t, g)
		})
	}
}

// A caller waits MaxWait in all for his keys, not MaxWait for each: one
// who waited for api three quarters of MaxWait gives up on web at MaxWait
// from his start, where a wait of MaxWait for web alone would last until
// one and three quarters. MaxWait is a second, so the two are three
// quarters of a second apart: the answer must come at MaxWait at the
// earliest and before halfway between them.
func TestACallerWaitsMaxWaitInAll(t *testing.T) {
	const maxWait = time.Second
	g := New[string](Limits{PerKey: 1, MaxQueue: 8, MaxWait: maxWait})
	api, web := enter(g, context.Background(), "api").held(t), enter(g, context.Background(), "web").held(t)
	start := time.Now()
	both := enter(g, context.Background(), "api", "web")
	queued(t, g, "api", 1)
	time.Sleep(maxWait * 3 / 4)
	api.leave()
	both.returned(t)
	if took := time.Since(start); !errors.Is(both.err, ErrTimeout) || took < maxWait || took >= maxWait*11/8 {
		t.Errorf("Enter() = %v after %v; want ErrTimeout at %v, before %v", both.err, took, maxWait, maxWait*11/8)
	}
	web.leave()
	forgotten(t, g)
}
````

- [ ] **Step 3: `project` 的锁**

`server/internal/modules/project/adapter/postgres/failures_test.go`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/failures_test.go
		t.Errorf("LockIssueProject() = %+v, %v, %v; want context.Canceled, not no project", p, found, err)
````
````new server/internal/modules/project/adapter/postgres/failures_test.go
		t.Errorf("LockIssueProject() = %+v, %v, %v; want context.Canceled, not no project", p, found, err)
	}
	if locked, err := s.LockIssueProjects(cancelled, []uuid.UUID{web}); !failed(err) || locked != nil {
		t.Errorf("LockIssueProjects() = %+v, %v; want context.Canceled, not no project", locked, err)
````

`server/internal/modules/project/adapter/postgres/issue_ports.go`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/issue_ports.go
		true, nil
````
````new server/internal/modules/project/adapter/postgres/issue_ports.go
		true, nil
}

// LockIssueProjects locks the undeleted projects among ids as
// LockIssueProject does, in one statement, in id order, and answers them
// by id: the first locks of a write across projects (M4 design 3.5,
// convention 7). One that is not there or is deleted, also one deleted
// while the lock waited, is not among them. It locks in the transaction
// ctx carries.
func (s *Store) LockIssueProjects(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]app.IssueProject, error) {
	rows, err := s.queries(ctx).LockIssueProjects(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("lock projects %v for their work items: %w", ids, err)
	}
	locked := make(map[uuid.UUID]app.IssueProject, len(rows))
	for _, r := range rows {
		locked[r.ID] = app.IssueProject{WorkspaceID: r.WorkspaceID, Archived: r.Archived, DefaultAssigneeID: r.DefaultAssigneeID,
			Identifier: r.Identifier}
	}
	return locked, nil
````

`server/internal/modules/project/adapter/postgres/lock_issue_projects_test.go`（新文件，155 行）：

````file server/internal/modules/project/adapter/postgres/lock_issue_projects_test.go
package postgresadapter_test

import (
	"context"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// The issue module's lock of several projects (M4 design 3.5, 6.5,
// convention 7): those asked for that are undeleted, as LockIssueProject
// answers each, by id, an archived one too; not a deleted one, nor one
// that is not there. Until the transaction ends each is held FOR NO KEY
// UPDATE, which a FOR SHARE waits for and a foreign key's FOR KEY SHARE
// does not; Ops, not asked for, is not held, nor is the workspace's row.
func TestLockIssueProjects(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, arch, gone, ops := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Arch", "ARCH", alice), newProject(t, s, acme,
		"Gone", "GONE", alice), newProject(t, s, acme, "Ops", "OPS", alice)
	exec(t, pool, "UPDATE projects SET default_assignee_id = $2 WHERE id = $1", web, alice)
	exec(t, pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", arch, now)
	exec(t, pool, "UPDATE projects SET deleted_at = $2 WHERE id = $1", gone, now)

	err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
		locked, err := s.LockIssueProjects(ctx, []uuid.UUID{gone, uuid.NewV7(), arch, web})
		if err != nil {
			return err
		}
		want := map[uuid.UUID]app.IssueProject{web: {WorkspaceID: acme, DefaultAssigneeID: &alice, Identifier: "WEB"},
			arch: {WorkspaceID: acme, Archived: true, Identifier: "ARCH"}}
		if jsonOf(t, locked) != jsonOf(t, want) {
			t.Errorf("LockIssueProjects() = %+v, want %+v", locked, want)
		}
		for name, tt := range map[string]struct {
			id   uuid.UUID
			held bool
		}{"Web": {web, true}, "Arch": {arch, true}, "Ops": {ops, false}} {
			if share, keyShare := waits(t, pool, tt.id, "FOR SHARE"), waits(t, pool, tt.id, "FOR KEY SHARE"); share != tt.held || keyShare {
				t.Errorf("%s while locked: a FOR SHARE waits %v, a FOR KEY SHARE waits %v; want %v, false", name, share, keyShare, tt.held)
			}
		}
		if rowWaits(t, pool, "workspaces", acme, "FOR UPDATE") {
			t.Error("a FOR UPDATE of acme waits while the projects are locked; want the workspace's row not locked")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// LockIssueProjects takes its locks in the projects' id order, whatever
// order they are asked in or their rows lie in: Web has the smaller id,
// but lies after Alpha in the table and in the indexes on the name and on
// the identifier, and is asked for after it. Alpha's row is held.
// LockIssueProjects waits for it holding Web's, which a FOR SHARE then
// waits for; in any other order it would reach Alpha first and wait
// holding nothing.
func TestLockIssueProjectsLocksInIDOrder(t *testing.T) {
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
			_, err := s.LockIssueProjects(ctx, []uuid.UUID{alpha, web})
			return err
		})
	}()
	pgtest.WaitForLockWaitOn(t, pool, "projects", 10*time.Second)

	if !waits(t, pool, web, "FOR SHARE") {
		t.Error("a FOR SHARE of Web while LockIssueProjects waits for Alpha does not wait; want Web locked first")
	}
	if err := held.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("LockIssueProjects() = %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("LockIssueProjects() did not end within 10s")
	}
}

// A project deleted while LockIssueProjects waits for its row is left out
// (M4 design 3.5): another transaction holds Web FOR NO KEY UPDATE, as
// deleteProject does, and soft-deletes it; LockIssueProjects, which found
// Web undeleted, waits for it; once the deletion commits, it evaluates
// deleted_at again on the row's newest version and answers Ops alone.
func TestLockIssueProjectsLeavesOutAProjectDeletedWhileItWaited(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, ops := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Ops", "OPS", alice)
	deletion, err := pool.Begin(pgtest.Soon(t))
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
	type answer struct {
		locked map[uuid.UUID]app.IssueProject
		err    error
	}
	done := make(chan answer, 1)
	go func() {
		var a answer
		a.err = postgres.NewTxManager(pool, 10*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
			var err error
			a.locked, err = s.LockIssueProjects(ctx, []uuid.UUID{web, ops})
			return err
		})
		done <- a
	}()
	pgtest.WaitForLockWaitOn(t, pool, "projects", 10*time.Second)
	if err := deletion.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	select {
	case a := <-done:
		if _, hasOps := a.locked[ops]; a.err != nil || len(a.locked) != 1 || !hasOps {
			t.Errorf("LockIssueProjects() = %+v, %v; want Ops (%s) alone, Web (%s) deleted while it waited", a.locked, a.err, ops, web)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("LockIssueProjects() did not end within 10s")
	}
}
````

`server/internal/modules/project/module.go`（修改，2 处）：

````old server/internal/modules/project/module.go
// 6.5): its gate (ProjectGate's Find and Lock), its work items' numbers
// (Sequences), and its states', labels' and members' catalogs
````
````new server/internal/modules/project/module.go
// 6.5): its gate (ProjectGate's Find, Lock and LockMany), its work items'
// numbers (Sequences), and its states', labels' and members' catalogs
````
````old server/internal/modules/project/module.go
	LockIssueProject(ctx context.Context, id uuid.UUID) (p IssueProject, found bool, err error)
````
````new server/internal/modules/project/module.go
	LockIssueProject(ctx context.Context, id uuid.UUID) (p IssueProject, found bool, err error)
	// LockIssueProjects locks the undeleted projects among ids so, in one
	// statement, in id order, and answers them by id; one deleted while
	// the lock waited is not among them.
	LockIssueProjects(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]IssueProject, error)
````

- [ ] **Step 4: `issue` 的端口和接线**

`server/internal/bootstrap/gate.go`（修改，1 处）：

````old server/internal/bootstrap/gate.go
// Enter enters the project's gate: a write the gate refuses, its queue
// full or its wait at max_wait, or whose request's deadline has passed
// when it comes to the gate or passes while it waits, is server_busy; one
// whose request is cancelled keeps context.Canceled, which no one waits
// for the answer of.
func (g writeGate) Enter(ctx context.Context, projectID uuid.UUID) (func(), error) {
	leave, err := g.gate.Enter(ctx, projectID)
````
````new server/internal/bootstrap/gate.go
// Enter enters the projects' gates, one after another, in their order: a
// write the gate refuses, a queue full or its wait at max_wait, or whose
// request's deadline has passed when it comes to a gate or passes while
// it waits, is server_busy; one whose request is cancelled keeps
// context.Canceled, which no one waits for the answer of.
func (g writeGate) Enter(ctx context.Context, projectIDs ...uuid.UUID) (func(), error) {
	leave, err := g.gate.Enter(ctx, projectIDs...)
````

`server/internal/bootstrap/gate_test.go`（修改，2 处）：

````old server/internal/bootstrap/gate_test.go
// keeps context.Canceled. A write on another project meanwhile enters at
// once: each project has places of its own (11.11). Once the first leaves,
// a write enters.
````
````new server/internal/bootstrap/gate_test.go
// keeps context.Canceled. A write on another project and this one is
// answered so too, and holds neither: a write on the other enters at once
// after it. A write on another project meanwhile enters at once: each
// project has places of its own (11.11). Once the first leaves, a write
// enters.
````
````old server/internal/bootstrap/gate_test.go
		cancel()
````
````new server/internal/bootstrap/gate_test.go
		cancel()
		across := uuid.NewV7()
		ctx, cancel = tt.ctx()
		if _, err := g.Enter(ctx, across, project); !tt.wants(err) {
			t.Errorf("%s: a write on another project and this one: %v", tt.name, err)
		}
		cancel()
		if left, err := g.Enter(context.Background(), across); err != nil {
			t.Fatalf("%s: a write on the other project after it: %v, want it in at once", tt.name, err)
		} else {
			left()
		}
````

`server/internal/bootstrap/ports.go`（修改，1 处）：

````old server/internal/bootstrap/ports.go
}

func (p issueProjects) Next(ctx context.Context, id uuid.UUID) (int, error) {
````
````new server/internal/bootstrap/ports.go
}

func (p issueProjects) LockMany(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]issue.LockedProject, error) {
	locked, err := p.projects.LockIssueProjects(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]issue.LockedProject, len(locked))
	for id, l := range locked {
		out[id] = issue.LockedProject(l)
	}
	return out, nil
}

func (p issueProjects) Next(ctx context.Context, id uuid.UUID) (int, error) {
````

`server/internal/bootstrap/ports_test.go`（修改，4 处）：

````old server/internal/bootstrap/ports_test.go
}

func (f *fakeIssueProjects) NextIssueSequence(context.Context, uuid.UUID) (int, error) {
````
````new server/internal/bootstrap/ports_test.go
}

func (f *fakeIssueProjects) LockIssueProjects(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]project.IssueProject, error) {
	f.asked = append(f.asked, fmt.Sprint("lock ", ids))
	return map[uuid.UUID]project.IssueProject{f.web: f.locked}, f.err
}

func (f *fakeIssueProjects) NextIssueSequence(context.Context, uuid.UUID) (int, error) {
````
````old server/internal/bootstrap/ports_test.go
		t.Error("Lock() of another project found it")
	}
````
````new server/internal/bootstrap/ports_test.go
		t.Error("Lock() of another project found it")
	}
	if l, err := p.LockMany(ctx, []uuid.UUID{acme, web}); err != nil || !maps.Equal(l, map[uuid.UUID]issue.LockedProject{web: {WorkspaceID: acme,
		Archived: true, DefaultAssigneeID: &ann, Identifier: "WEB"}}) {
		t.Errorf("LockMany() = %+v, %v; want web as locked", l, err)
	}
````
````old server/internal/bootstrap/ports_test.go
	if want := []string{"workspace", "lock", "lock", "next", "states", fmt.Sprint("labels ", []uuid.UUID{bug, ui}),
		fmt.Sprint("roles ", []uuid.UUID{ann})}; !slices.Equal(fake.asked, want) {
````
````new server/internal/bootstrap/ports_test.go
	if want := []string{"workspace", "lock", "lock", fmt.Sprint("lock ", []uuid.UUID{acme, web}), "next", "states",
		fmt.Sprint("labels ", []uuid.UUID{bug, ui}), fmt.Sprint("roles ", []uuid.UUID{ann})}; !slices.Equal(fake.asked, want) {
````
````old server/internal/bootstrap/ports_test.go
		t.Errorf("Labels() = %+v, %v; want no labels and %v", got, err, failure)
````
````new server/internal/bootstrap/ports_test.go
		t.Errorf("Labels() = %+v, %v; want no labels and %v", got, err, failure)
	}
	if got, err := p.LockMany(ctx, []uuid.UUID{web}); !errors.Is(err, failure) || got != nil {
		t.Errorf("LockMany() = %+v, %v; want no projects and %v", got, err, failure)
````

`server/internal/modules/issue/app/fakes_across_test.go`（新文件，38 行）：

````file server/internal/modules/issue/app/fakes_across_test.go
package app_test

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// anys are ids as a logged call's arguments, each its own.
func anys(ids []uuid.UUID) []any {
	out := make([]any, len(ids))
	for n, id := range ids {
		out[n] = id
	}
	return out
}

// LockMany logs the lock, each project its own argument, and answers
// those of ids the fake has locked: its project, as Lock answers it, and
// those of across, as they are (fakeProject).
func (f *fakeProject) LockMany(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]app.LockedProject, error) {
	if f.waited != nil {
		f.waited()
	}
	if err := f.log.add(ctx, "LockMany", anys(ids)...); err != nil {
		return nil, err
	}
	locked := map[uuid.UUID]app.LockedProject{}
	for _, id := range ids {
		if p, ok := f.across[id]; ok {
			locked[id] = p
		} else if id == f.id && f.locked != nil {
			locked[id] = *f.locked
		}
	}
	return locked, nil
}
````

`server/internal/modules/issue/app/fakes_test.go`（修改，5 处）：

````old server/internal/modules/issue/app/fakes_test.go
func (f fakeGate) Enter(ctx context.Context, projectID uuid.UUID) (func(), error) {
````
````new server/internal/modules/issue/app/fakes_test.go
func (f fakeGate) Enter(ctx context.Context, projectIDs ...uuid.UUID) (func(), error) {
````
````old server/internal/modules/issue/app/fakes_test.go
	if err := f.log.add(ctx, "Enter", projectID); err != nil {
````
````new server/internal/modules/issue/app/fakes_test.go
	if err := f.log.add(ctx, "Enter", anys(projectIDs)...); err != nil {
````
````old server/internal/modules/issue/app/fakes_test.go
		call := line("Leave", projectID)
````
````new server/internal/modules/issue/app/fakes_test.go
		call := line("Leave", anys(projectIDs)...)
````
````old server/internal/modules/issue/app/fakes_test.go
// answers it, locked what Lock answers, nil for none; waited, when set,
// runs as Lock waits, before it answers: what commits meanwhile.
````
````new server/internal/modules/issue/app/fakes_test.go
// answers it, locked what Lock answers, nil for none, and across the
// other projects LockMany answers; waited, when set, runs as a lock waits,
// before it answers: what commits meanwhile.
````
````old server/internal/modules/issue/app/fakes_test.go
	locked    *app.LockedProject
````
````new server/internal/modules/issue/app/fakes_test.go
	locked    *app.LockedProject
	across    map[uuid.UUID]app.LockedProject
````

`server/internal/modules/issue/app/ports.go`（修改，2 处）：

````old server/internal/modules/issue/app/ports.go
	Lock(ctx context.Context, projectID uuid.UUID) (p LockedProject, found bool, err error)
````
````new server/internal/modules/issue/app/ports.go
	Lock(ctx context.Context, projectID uuid.UUID) (p LockedProject, found bool, err error)
	// LockMany locks the undeleted projects among projectIDs, each named
	// once, in ascending id order, as Lock does, in that order, in one
	// statement, and answers them by id: a write across projects takes
	// them so (convention 7, M4 design 3.5). One that is not there, also
	// one deleted while the lock waited, is not among them.
	LockMany(ctx context.Context, projectIDs []uuid.UUID) (map[uuid.UUID]LockedProject, error)
````
````old server/internal/modules/issue/app/ports.go
	// Enter returns once a write on the project's work items may begin
	// its transaction, with the function that leaves the gate, which the
	// write calls once its transaction has ended. It waits holding no
	// connection and no lock. A write that may wait no longer is refused
	// (503 server_busy); one whose request is cancelled while it waits
	// ends with context.Canceled. Either holds nothing.
	Enter(ctx context.Context, projectID uuid.UUID) (leave func(), err error)
````
````new server/internal/modules/issue/app/ports.go
	// Enter returns once a write on the projects' work items may begin its
	// transaction, with the function that leaves their gates, which the
	// write calls once its transaction has ended. A write across projects
	// names each once, in ascending id order (convention 7), and enters
	// their gates in that order, waiting for each holding no connection
	// and no lock, only its places in the gates before. A write that may
	// wait no longer is refused (503 server_busy); one whose request is
	// cancelled while it waits ends with context.Canceled. Either holds
	// nothing.
	Enter(ctx context.Context, projectIDs ...uuid.UUID) (leave func(), err error)
````

- [ ] **Step 5: 测试和 lint**

Run: `go -C server test -count=1 -race ./internal/platform/gate/`
Expected: `ok`。

Run: `go -C server test -count=1 ./internal/modules/issue/... ./internal/modules/project/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestTheWriteGateAnswersItsRefusalsAsServerBusy$|TestIssueProjectsConvertsProjectsAnswers$|TestEachWriteOnAWorkItemOfAProjectTooBusyAnswersServerBusy$' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 6: 提交**

```bash
git add server/internal/bootstrap/gate.go server/internal/bootstrap/gate_test.go server/internal/bootstrap/ports.go server/internal/bootstrap/ports_test.go server/internal/modules/issue/app/fakes_across_test.go server/internal/modules/issue/app/fakes_test.go server/internal/modules/issue/app/ports.go server/internal/modules/project/adapter/postgres/failures_test.go server/internal/modules/project/adapter/postgres/issue_ports.go server/internal/modules/project/adapter/postgres/lock_issue_projects_test.go server/internal/modules/project/adapter/postgres/queries/issue_ports.sql server/internal/modules/project/module.go server/internal/platform/config/config.go server/internal/platform/gate/gate.go server/internal/platform/gate/gate_test.go server/internal/platform/gate/keys_test.go server/internal/modules/project/adapter/postgres/gen/issue_ports.sql.go
```
```bash
git commit -m "feat(M4/P3): a write may enter several projects' gates and lock several projects

The gate enters several keys one after another, in their order, with one
max_wait for them all, holding only the keys before while it waits for
the next, and leaves them all when it cannot have one: a write across
projects names them once each, in ascending id order. The project module
locks several projects in one statement in id order, FOR NO KEY UPDATE,
leaving out one deleted while the lock waited, behind ProjectGate's
LockMany.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p3.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试，或没有替身的平台、领域、转换的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；守卫是生成器（`make gen-go`）和中英文翻译键（`make lint-web`）的核对。每个变异在一份新的副本里，在本 Task 的树上、标出的较晚的 Task 的树上和最终的树上都跑过，端到端的检查只在标出的 Task 的树上）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T7.1` | 进几个键的调用者在每个键上都可以等 `MaxWait` | `internal/platform/gate` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAWriteAcrossProjectsWaitsMaxWaitInAll`（`make test`）（Task 11 起） | 单元；组合 |
| `T7.2` | 没拿到每个键的调用者留着已经进了的那些 | `internal/platform/gate` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestTheWriteGateAnswersItsRefusalsAsServerBusy`（`make test`）、`internal/bootstrap` 的 `TestAWriteAcrossProjectsGivesBackEachPlaceItTook`（`make test`）（Task 11 起） | 单元；组合 |
| `T7.3` | 离开只让出第一个键 | `internal/platform/gate` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestAWriteAcrossProjectsGivesBackEachPlaceItTook`（`make test`）（Task 11 起） | 单元；组合 |
| `T7.4` | `LockMany` 按行到来的次序、而不是 id 的次序锁项目 | `internal/modules/project/adapter/postgres` 的全部测试（`make test`） | 存储 |
| `T7.5` | `LockMany` 也锁已删除的项目 | `internal/modules/project/adapter/postgres` 的全部测试（`make test`） | 存储 |
| `T7.6` | `LockMany` 不回答项目的标识 | `internal/modules/project/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWorkItemsAreRelatedAcrossProjects`（`make test`）（Task 11 起） | 存储；组合 |
| `T7.7` | 跨项目的写只进第一个项目的写闸门 | `internal/bootstrap` 的 `TestTheWriteGateAnswersItsRefusalsAsServerBusy`（`make test`）、`internal/bootstrap` 的 `TestAWriteAcrossProjectsGivesBackEachPlaceItTook`（`make test`）（Task 11 起） | 单元；组合 |
| `T7.8` | `bootstrap` 交给 `issue` 的 `LockMany` 的项目是空的 | `internal/bootstrap` 的 `TestIssueProjectsConvertsProjectsAnswers`（`make test`）、`internal/bootstrap` 的 `TestWorkItemsAreRelatedAcrossProjects`（`make test`）（Task 11 起） | 单元；组合 |
| `T7.9` | `LockMany` 经连接池锁项目，语句一结束锁就放开 | `internal/bootstrap` 的 `TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection`（`make test`）（Task 11 起）、`internal/bootstrap` 的 `TestOppositeRelationsOfTwoWorkItemsSerialize`（`make test`）（Task 14 起） | 组合 |

---

### Task 8: 迁移 `00020`；删除工作项、项目、工作区连带关联行

**Files:**
- Create: `server/internal/modules/issue/adapter/postgres/delete_relations_test.go`、`server/internal/platform/postgres/pgtest/rowwaits.go`、`server/internal/platform/postgres/pgtest/rowwaits_test.go`、`server/migrations/sql/00020_issue_issue_relations.sql`
- Modify: `api/modules/issue.yaml`、`api/modules/project.yaml`、`api/modules/workspace.yaml`、`deploy/runtime-grants.sql`、`e2e/fixtures/assert/issue.ts`、`e2e/fixtures/assert/project.ts`、`e2e/fixtures/assert/workspace.ts`、`server/internal/bootstrap/issue_deletion_test.go`、`server/internal/bootstrap/issue_interleaving_deletion_test.go`、`server/internal/bootstrap/issue_interleaving_test.go`、`server/internal/bootstrap/issue_world_test.go`、`server/internal/bootstrap/issue_write_locks_test.go`、`server/internal/bootstrap/project_deletion_test.go`、`server/internal/modules/issue/adapter/postgres/delete_issue.go`、`server/internal/modules/issue/adapter/postgres/delete_issue_test.go`、`server/internal/modules/issue/adapter/postgres/delete_rows.go`、`server/internal/modules/issue/adapter/postgres/delete_rows_test.go`、`server/internal/modules/issue/adapter/postgres/queries/delete_issue.sql`、`server/internal/modules/issue/adapter/postgres/queries/delete_rows.sql`、`server/internal/modules/issue/adapter/postgres/store_test.go`、`server/internal/modules/issue/app/cascade_test.go`、`server/internal/modules/issue/app/delete_issue.go`、`server/internal/modules/issue/app/delete_issue_test.go`、`server/internal/modules/issue/app/delete_project_rows.go`、`server/internal/modules/issue/app/fakes_deletion_test.go`、`server/internal/modules/project/adapter/postgres/demote_test.go`、`server/internal/modules/project/adapter/postgres/issue_ports_test.go`、`server/internal/modules/project/adapter/postgres/lock_issue_projects_test.go`、`server/migrations/issue_schema_test.go`、`server/migrations/schema_test.go`、`server/sqlc.yaml`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/issue/adapter/postgres/gen/delete_issue.sql.go`、`server/internal/modules/issue/adapter/postgres/gen/delete_rows.sql.go`、`server/internal/modules/issue/adapter/postgres/gen/models.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**（spec 2.10、第 3 节第 3、4、14、16、18 条；M4 设计 3.5 约定五的例外、3.6、3.10、4.6，12 节约束 8）
- 迁移 `server/migrations/sql/00020_issue_issue_relations.sql`（新）：`issue_relations`，Plane 的 11 列（4.6）；`relation_type` 只有三种存储值（`blocked_by`、`duplicate`、`relates_to`：阻塞反过来存为对方的 `blocked_by`），不设默认；`issue_relations_not_self_check`；`issue_relations_pair_key`（`LEAST`、`GREATEST` 两端，未删除的：一对工作项至多一条，不论类型和方向）；两端、工作区、项目的索引（不带条件：两个方向的读和物理级联按它们找行）。`server/sqlc.yaml` 加它；`deploy/runtime-grants.sql` 加表的权限。
- `issue/adapter/postgres/queries/delete_issue.sql` 的 `DeleteIssueRelationRows`：子树的关联行，两个方向（另一端在这个项目或别的项目），先按 `id` 升序 `FOR NO KEY UPDATE` 锁住再写（`UPDATE … WHERE id IN (SELECT … ORDER BY id FOR NO KEY UPDATE)`：别的项目的写不持这个项目的锁也写得到这些行，约定五的例外）；等锁期间被别人删除的行在最新的版本上重新判断、跳过，留着那次删除。`delete_rows.sql` 的 `DeleteProjectRelationRows`：删除项目的一批，`project_id` 是这个项目的行和 `related_issue_id` 是这个项目的工作项（已删除的也算：工作项的一批先删了它们）的别的项目的行，同样按 `id` 先锁、键集分批。`delete_issue.go`、`delete_rows.go` 照写。
- `issue/app`：`IssueRowsDeleter.DeleteIssueRelationRows`，`issueRowSteps` 的关联一步在标签之后、关注之前（全局的表序：工作项 → 负责人 → 标签 → 关联 → 关注 → 动态）；`delete_project_rows.go` 的 `ProjectRowsDeleter.DeleteProjectRelationRows`，`rowSteps` 同样的位置。
- `platform/postgres/pgtest/rowwaits.go`（新）：`RowWaits(t, pool, table, id, lock)`（`NOWAIT` 的探测：别的事务持着与 `lock` 冲突的锁时为真）；原来在 `project` 的 `demote_test.go` 里手写的 `rowWaits`（含一个手写的 `55P03`）换成它，`issue_ports_test.go`、`lock_issue_projects_test.go`、新的 `delete_relations_test.go` 共用（P1b 第 5 节：P2 起不手写 `55P03`）。
- 契约：`deleteIssue`、`deleteProject`、`deleteWorkspace` 的描述写上关联（生成的打包结果和 `schema.gen.ts` 随之，Go 不变）。
- 测试的帮手：`store_test.go` 的 `newRelation`；`issue_world_test.go` 的 `workItemStamps` 读关联行（两个方向）；`issue_interleaving_test.go` 的 `wroteNothing`、`issue_interleaving_deletion_test.go` 的删除的行数关联表；`issue_deletion_test.go` 的 `relate`（直接写一条关联：经接口建关联在 Task 11）；`project_deletion_test.go` 的 `seedIssues` 加一个工作项 Blocker 和种下的工作项到它的一条 `blocked_by`（删除项目、工作区的按目录的测试都用它，读得到关联表的行）。
- 端到端的两个暂时的写法（第 3 节第 18 条）：`assert/project.ts`、`assert/workspace.ts` 的删除断言暂时不要关联表的行（`withoutRows`：故事还建不了关联，Task 11 去掉）；`assert/issue.ts` 的 `expectIssueDeleted` 把 P2 的 `subItems` 换成 `unwritten`（`issues.parent_id` 和关联的两个外键：故事还不写它们，Task 15 去掉）。

**Tests:**
- 结构：`TestMigrationsGoUpDownAndUpAgain`（20 个迁移）；`TestIssueChecksRejectCounterexamples`（三种存储值通过；别的值、`blocking`、Plane 的 `start_before`、自己关联自己都被拒绝）、`TestIssueUniqueKeys`（一对工作项一条：反方向、别的类型都被拒绝；共用一端的两条通过；软删除之后这一对空出来）。
- 单元：`TestDeleteIssueDeletesTheWorkItemsSubtreeAndItsRows`、`TestDeleteIssueStopsAtAFailingPort` 加关联的一步；`TestTheCascadeStopsAtAFailingBatch` 加关联的一批。
- 存储：`TestTheRelationsOfADeletedSubtreeAreDeletedBothWays`（item、sub 被删除：四条名字里有它们的行，不论在哪一端、哪个项目的，由 bob 在 later 删除；名字里没有它们的两条和一小时前 alice 删的那条不动；每一列都不动）、`TestTheRelationsOfADeletedSubtreeAreLockedInIDOrder`（a、b、c 的 `id` 升序，反着插入，表或任一端的索引都先扫到 c；另一个事务删除并持着 b：语句持着 a 等 b（`FOR SHARE` 等 a），还没碰到 c；提交之后删 a、c，b 留着 alice 的删除）、`TestARelationsBatchTakesTheRowsOfTheProjectsWorkItemsBothWays`（web 的一批取名字里有 web 的工作项的三行，已删除的 w1 也算，只取它们）；`TestTheStatementsOfADeletedSubtreeDeleteItsRows` 加关联表；`pgtest` 的 `TestRowWaitsSeesALockThatConflicts`（`FOR SHARE` 等 `FOR NO KEY UPDATE`，不等 `FOR KEY SHARE`、没人持的、提交之后的）。
- 组合：`TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt`（关联的两个外键下各一行：它到 Ops 的工作项的（Web 的行）、Ops 的别的工作项到它的（Ops 的行），和一条早已删除的）；`TestDeletingAProjectDeletesItsWorkItemsRelationsEitherWay`（Web 的 Seeded 到 Ops 的 Seeded（Web 的行）、Ops 的 Blocker 到 Web 的 Seeded（Ops 的行，不在 Web 下面，按目录的删除测试读不到它）：删除 Web 时两条都在 Web 的删除时刻删除；Ops 自己的关联不动）；`TestDeletingAProjectLeavesNoUndeletedRowUnderIt`、`TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt`（`seedIssues` 的关联）；`TestEachIssueWriteLocksItsProjectFirst`、`TestDeletingAWorkItemAndChangingItSerialize`。

- [ ] **Step 1: 迁移、契约和语句，生成**

`api/modules/issue.yaml`（修改，1 处）：

````old api/modules/issue.yaml
        labels, subscribers and activity, are deleted by the caller at one
        moment; their numbers are not used again. A project with too many
        writes on its work items at once answers server_busy, with
        Retry-After: try again then.
````
````new api/modules/issue.yaml
        labels, relations, subscribers and activity, are deleted by the
        caller at one moment; their numbers are not used again. A project
        with too many writes on its work items at once answers server_busy,
        with Retry-After: try again then.
````

`api/modules/project.yaml`（修改，1 处）：

````old api/modules/project.yaml
        subscribers and activity), all at one moment: it is no longer read,
        listed or changed, and its name and identifier are free again in the
        workspace. A project that does not exist, is deleted, or that the
        caller does not see answers project.not_found; one he sees but may
        not delete, forbidden. The role is decided after the project row is
        locked.
````
````new api/modules/project.yaml
        relations, subscribers and activity), all at one moment: it is no
        longer read, listed or changed, and its name and identifier are free
        again in the workspace. A project that does not exist, is deleted, or
        that the caller does not see answers project.not_found; one he sees
        but may not delete, forbidden. The role is decided after the project
        row is locked.
````

`api/modules/workspace.yaml`（修改，1 处）：

````old api/modules/workspace.yaml
        (with the work items' assignees, labels, subscribers and activity)
        are soft-deleted in one transaction, at the same moment; the
````
````new api/modules/workspace.yaml
        (with the work items' assignees, labels, relations, subscribers and
        activity) are soft-deleted in one transaction, at the same moment; the
````

`deploy/runtime-grants.sql`（修改，1 处）：

````old deploy/runtime-grants.sql
GRANT SELECT, INSERT, UPDATE, DELETE ON issues, issue_assignees, issue_labels, issue_subscribers, issue_activities TO nerve_runtime;
````
````new deploy/runtime-grants.sql
GRANT SELECT, INSERT, UPDATE, DELETE ON issues, issue_assignees, issue_labels, issue_relations, issue_subscribers, issue_activities
    TO nerve_runtime;
````

`server/internal/modules/issue/adapter/postgres/queries/delete_issue.sql`（修改，1 处）：

````old server/internal/modules/issue/adapter/postgres/queries/delete_issue.sql
WHERE issue_id = ANY (sqlc.arg(issue_ids)::uuid[]) AND deleted_at IS NULL;

-- name: DeleteIssueSubscriberRows :exec
````
````new server/internal/modules/issue/adapter/postgres/queries/delete_issue.sql
WHERE issue_id = ANY (sqlc.arg(issue_ids)::uuid[]) AND deleted_at IS NULL;

-- name: DeleteIssueRelationRows :exec
-- The deleted work items' relations (M4 design 3.5, 3.6), both ways: the undeleted rows that name one of them at either
-- end, the other end in this project or in another, soft-deleted at the moment and by the account given, in one
-- statement. A relation with a work item of another project is reached by that project's writes without this
-- project's lock: its deletion, and deleting the work item at that end. Two writers that each took such rows in their
-- scans' order could each hold one the other waits for, so the statement locks its rows FOR NO KEY UPDATE in id order
-- before it writes them, as each of the others does (convention 5's exception, 3.5). A row another deleted while the
-- statement waited for it is evaluated again on its newest version and passed over, and keeps that deletion.
UPDATE issue_relations AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE t.id IN (
    SELECT r.id FROM issue_relations r
    WHERE (r.issue_id = ANY (sqlc.arg(issue_ids)::uuid[]) OR r.related_issue_id = ANY (sqlc.arg(issue_ids)::uuid[]))
        AND r.deleted_at IS NULL
    ORDER BY r.id FOR NO KEY UPDATE);

-- name: DeleteIssueSubscriberRows :exec
````

`server/internal/modules/issue/adapter/postgres/queries/delete_rows.sql`（修改，1 处）：

````old server/internal/modules/issue/adapter/postgres/queries/delete_rows.sql
RETURNING t.id;

-- name: DeleteProjectSubscriberRows :many
````
````new server/internal/modules/issue/adapter/postgres/queries/delete_rows.sql
RETURNING t.id;

-- name: DeleteProjectRelationRows :many
-- DeleteProjectIssueRows' batch, of the relations of the project's work items, both ways (M4 design 3.5, 3.6): the
-- project's rows (project_id is the issue_id end's) and another project's rows whose related_issue_id is a work item
-- of this project, deleted or not: the work items' batches have deleted them before this one. Another project's
-- writes reach such a row without this project's lock (its deletion, deleting its end's work item); every writer of
-- these rows takes them in id order (convention 5's exception, 3.5), as the batch's lock does.
UPDATE issue_relations AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE t.id IN (
    SELECT r.id FROM issue_relations r
    WHERE (r.project_id = sqlc.arg(project_id)
            OR r.related_issue_id IN (SELECT i.id FROM issues i WHERE i.project_id = sqlc.arg(project_id)))
        AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid
    ORDER BY r.id LIMIT sqlc.arg(batch) FOR NO KEY UPDATE)
RETURNING t.id;

-- name: DeleteProjectSubscriberRows :many
````

`server/migrations/sql/00020_issue_issue_relations.sql`（新文件，31 行）：

````file server/migrations/sql/00020_issue_issue_relations.sql
-- issue_relations：Plane 的 issue_relations 表（11 列）按 M4 设计 4.6 照搬 11 列。
-- 一对工作项一行（3.10）：阻塞反过来存为对方被本工作项阻塞；project_id 是 issue_id 那一端的项目。

-- +goose Up
CREATE TABLE issue_relations (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    -- issue_id 那一端的项目（Plane 写请求路径的项目，3.10）
    project_id uuid NOT NULL REFERENCES projects ON DELETE CASCADE,
    issue_id uuid NOT NULL REFERENCES issues ON DELETE CASCADE,
    related_issue_id uuid NOT NULL REFERENCES issues ON DELETE CASCADE,
    -- Plane 默认 blocked_by、不检查取值；Nerve 不设默认，只有页面的三种存储值（阻塞存为反向的 blocked_by）
    relation_type varchar(20) NOT NULL CHECK (relation_type IN ('blocked_by', 'duplicate', 'relates_to')),
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    CONSTRAINT issue_relations_not_self_check CHECK (issue_id <> related_issue_id)
);
-- 一对工作项之间最多一条未删除的关联，不论类型和方向（取代 Plane 有方向的键，3.10）
CREATE UNIQUE INDEX issue_relations_pair_key ON issue_relations (LEAST(issue_id, related_issue_id), GREATEST(issue_id, related_issue_id))
    WHERE deleted_at IS NULL;
-- 两个方向的读取和物理级联按它们找行，所以不带条件（4.6）
CREATE INDEX issue_relations_issue_id_idx ON issue_relations (issue_id);
CREATE INDEX issue_relations_related_issue_id_idx ON issue_relations (related_issue_id);
CREATE INDEX issue_relations_workspace_id_idx ON issue_relations (workspace_id);
CREATE INDEX issue_relations_project_id_idx ON issue_relations (project_id);

-- +goose Down
DROP TABLE issue_relations;
````

`server/sqlc.yaml`（修改，1 处）：

````old server/sqlc.yaml
      - migrations/sql/00019_issue_issue_activities.sql
````
````new server/sqlc.yaml
      - migrations/sql/00019_issue_issue_activities.sql
      - migrations/sql/00020_issue_issue_relations.sql
````

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `df39e8459a70f9275c4bdaf444bffb2aa3a11982faeae57844c0c50f28466d68` | 3535 | `api/dist/openapi.yaml` |
| `46638be2f0370a7bf248a8072117272025246f4e3c6cbc8ad0f09503065a2807` | 171 | `server/internal/modules/issue/adapter/postgres/gen/delete_issue.sql.go` |
| `db23530cfa12626898abafd28a1f93c4f7c010d969ee99f89551c5f06aa8f133` | 294 | `server/internal/modules/issue/adapter/postgres/gen/delete_rows.sql.go` |
| `d4082fbdf770a5f658ce47afd4a616018b9356bdc7764efebaf2f22e5cb47c4d` | 107 | `server/internal/modules/issue/adapter/postgres/gen/models.go` |
| `de76a3e9f5a4f54a72e4148e6ec7aec4e7aef86b8b9060ae6726ffdc2370e4a5` | 3860 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/issue/adapter/postgres/gen/delete_issue.sql.go server/internal/modules/issue/adapter/postgres/gen/delete_rows.sql.go server/internal/modules/issue/adapter/postgres/gen/models.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 2: 结构的测试**

`server/migrations/issue_schema_test.go`（修改，9 处）：

````old server/migrations/issue_schema_test.go
// (M4 design 4.2–4.5, 4.11), as TestConstraintAndIndexNames reads them: the
// part of its want that the migrations 00015–00019 add.
````
````new server/migrations/issue_schema_test.go
// (M4 design 4.2–4.6, 4.11), as TestConstraintAndIndexNames reads them: the
// part of its want that the migrations 00015–00020 add.
````
````old server/migrations/issue_schema_test.go
	"issue_labels_workspace_id_idx i",
````
````new server/migrations/issue_schema_test.go
	"issue_labels_workspace_id_idx i",
	"issue_relations_created_by_id_fkey f n",
	"issue_relations_issue_id_fkey f c",
	"issue_relations_issue_id_idx i",
	"issue_relations_not_self_check c",
	"issue_relations_pair_key iuw",
	"issue_relations_pkey iu",
	"issue_relations_pkey p",
	"issue_relations_project_id_fkey f c",
	"issue_relations_project_id_idx i",
	"issue_relations_related_issue_id_fkey f c",
	"issue_relations_related_issue_id_idx i",
	"issue_relations_relation_type_check c",
	"issue_relations_updated_by_id_fkey f n",
	"issue_relations_workspace_id_fkey f c",
	"issue_relations_workspace_id_idx i",
````
````old server/migrations/issue_schema_test.go
// item numbered 1, and an activity row of web's.
````
````new server/migrations/issue_schema_test.go
// item numbered 1, an activity row of web's, and web's work item related
// to ops'.
````
````old server/migrations/issue_schema_test.go
			", " + webIssue + ")",
````
````new server/migrations/issue_schema_test.go
			", " + webIssue + ")",
		relationOf(issueWeb, webIssue, opsIssue, "relates_to"),
````
````old server/migrations/issue_schema_test.go
}

// The issue tables' CHECKs accept what the domain writes and reject what
// bypasses it (M4 design 4.2, 4.11): every priority and verb, a start on
// its target date or with no target, a top-level work item, the number 1;
// and refuse an empty title, a priority outside the five, in another case
// too, the number 0, a work item its own parent, a start after its target,
// a verb outside the three.
````
````new server/migrations/issue_schema_test.go
}

// relationOf is the insertion of a relation of the work item issue, of the
// project project, to the work item related, of the type kind.
func relationOf(project, issue, related, kind string) string {
	return "INSERT INTO issue_relations (id, workspace_id, project_id, issue_id, related_issue_id, relation_type) VALUES (gen_random_uuid(), " +
		issueAcme + ", " + project + ", " + issue + ", " + related + ", '" + kind + "')"
}

// The issue tables' CHECKs accept what the domain writes and reject what
// bypasses it (M4 design 4.2, 4.6, 4.11): every priority, verb and stored
// relation type, a start on its target date or with no target, a top-level
// work item, the number 1; and refuse an empty title, a priority outside
// the five, in another case too, the number 0, a work item its own parent,
// a start after its target, a verb outside the three, a relation type
// outside the three stored ones (blocking, which is stored reversed, and
// Plane's start_before among them), a work item related to itself.
````
````old server/migrations/issue_schema_test.go
		"UPDATE issue_activities SET verb = 'created'",
````
````new server/migrations/issue_schema_test.go
		"UPDATE issue_activities SET verb = 'created'", "UPDATE issue_relations SET relation_type = 'blocked_by'",
		"UPDATE issue_relations SET relation_type = 'duplicate'", "UPDATE issue_relations SET relation_type = 'relates_to'",
````
````old server/migrations/issue_schema_test.go
		{"unknown verb", "UPDATE issue_activities SET verb = 'commented'", "issue_activities_verb_check"},
````
````new server/migrations/issue_schema_test.go
		{"unknown verb", "UPDATE issue_activities SET verb = 'commented'", "issue_activities_verb_check"},
		{"blocking stored", "UPDATE issue_relations SET relation_type = 'blocking'", "issue_relations_relation_type_check"},
		{"Plane's start_before", "UPDATE issue_relations SET relation_type = 'start_before'", "issue_relations_relation_type_check"},
		{"related to itself", "UPDATE issue_relations SET related_issue_id = issue_id", "issue_relations_not_self_check"},
````
````old server/migrations/issue_schema_test.go
// work item holds the same pair of its own.
````
````new server/migrations/issue_schema_test.go
// work item holds the same pair of its own. A pair of work items has at
// most one undeleted relation, whatever its type and direction (3.10,
// 4.6): beside web's work item related to ops', ops' is refused a relation
// to web's of another type, and web's one to ops' of another type; web's
// relation to another work item of web, and that one's to ops' work item,
// each sharing one end with the first, are accepted; soft-deleting the
// first frees the pair for ops' relation to web's.
````
````old server/migrations/issue_schema_test.go
				t.Errorf("after the first is deleted, %s = %v; want the pair free", insert(issueWeb, webIssue), err)
			}
		})
	}
}

````
````new server/migrations/issue_schema_test.go
				t.Errorf("after the first is deleted, %s = %v; want the pair free", insert(issueWeb, webIssue), err)
			}
		})
	}

	t.Run("issue_relations", func(t *testing.T) {
		other := "'0199a2b4-0000-7000-8000-00000000000b'"
		refused(t, relationOf(issueOps, opsIssue, webIssue, "blocked_by"), "issue_relations_pair_key")
		refused(t, relationOf(issueWeb, webIssue, opsIssue, "duplicate"), "issue_relations_pair_key")
		for _, stmt := range []string{
			"INSERT INTO issues (id, workspace_id, project_id, state_id, name, sequence_id) VALUES (" + other + ", " + issueAcme + ", " + issueWeb +
				", " + webState + ", 'Other', 2)",
			relationOf(issueWeb, webIssue, other, "relates_to"), relationOf(issueWeb, other, opsIssue, "blocked_by"),
			"UPDATE issue_relations SET deleted_at = now() WHERE issue_id = " + webIssue + " AND related_issue_id = " + opsIssue,
			relationOf(issueOps, opsIssue, webIssue, "blocked_by"),
		} {
			if _, err := pool.Exec(ctx, stmt); err != nil {
				t.Errorf("%s: %v, want it accepted", stmt, err)
			}
		}
	})
}

````

`server/migrations/schema_test.go`（修改，3 处）：

````old server/migrations/schema_test.go
	if err != nil || len(up) != 19 {
		t.Fatalf("Up() = %d migrations, %v; want 19", len(up), err)
````
````new server/migrations/schema_test.go
	if err != nil || len(up) != 20 {
		t.Fatalf("Up() = %d migrations, %v; want 20", len(up), err)
````
````old server/migrations/schema_test.go
		{tablesQuery, []string{"api_tokens", "auth_sessions", "issue_activities", "issue_assignees", "issue_labels", "issue_subscribers", "issues",
			"labels", "profiles", "project_members", "project_user_properties", "projects", "river_job", "river_leader", "river_notification",
			"river_queue", "states", "users", "workspace_member_invites", "workspace_members", "workspace_user_properties", "workspaces"}},
````
````new server/migrations/schema_test.go
		{tablesQuery, []string{"api_tokens", "auth_sessions", "issue_activities", "issue_assignees", "issue_labels", "issue_relations",
			"issue_subscribers", "issues", "labels", "profiles", "project_members", "project_user_properties", "projects", "river_job",
			"river_leader", "river_notification", "river_queue", "states", "users", "workspace_member_invites", "workspace_members",
			"workspace_user_properties", "workspaces"}},
````
````old server/migrations/schema_test.go
	if again, err := m.Up(ctx); err != nil || len(again) != 19 {
		t.Errorf("Up() again = %d migrations, %v; want 19", len(again), err)
````
````new server/migrations/schema_test.go
	if again, err := m.Up(ctx); err != nil || len(again) != 20 {
		t.Errorf("Up() again = %d migrations, %v; want 20", len(again), err)
````

- [ ] **Step 3: `pgtest.RowWaits`**

`server/internal/modules/project/adapter/postgres/demote_test.go`（修改，4 处）：

````old server/internal/modules/project/adapter/postgres/demote_test.go
	"context"
	"errors"
````
````new server/internal/modules/project/adapter/postgres/demote_test.go
	"context"
````
````old server/internal/modules/project/adapter/postgres/demote_test.go

	"github.com/jackc/pgx/v5/pgconn"
````
````new server/internal/modules/project/adapter/postgres/demote_test.go

````
````old server/internal/modules/project/adapter/postgres/demote_test.go
// row, lock, waits for a lock held on it (rowWaits on projects).
````
````new server/internal/modules/project/adapter/postgres/demote_test.go
// row, lock, waits for a lock held on it (pgtest.RowWaits on projects).
````
````old server/internal/modules/project/adapter/postgres/demote_test.go
	return rowWaits(t, pool, "projects", project, lock)
}

// rowWaits reports whether a statement of another transaction on the row
// id of table, lock, waits for a lock held on it: NOWAIT answers
// lock_not_available (55P03) at once instead of waiting. Its wait for a
// connection of the pool ends soon (pgtest.Soon).
func rowWaits(t *testing.T, pool *pgxpool.Pool, table string, id uuid.UUID, lock string) bool {
	t.Helper()
	_, err := pool.Exec(pgtest.Soon(t), "SELECT 1 FROM "+table+" WHERE id = $1 "+lock+" NOWAIT", id)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "55P03" {
		return true
	}
	if err != nil {
		t.Fatal(err)
	}
	return false
````
````new server/internal/modules/project/adapter/postgres/demote_test.go
	return pgtest.RowWaits(t, pool, "projects", project, lock)
````

`server/internal/modules/project/adapter/postgres/issue_ports_test.go`（修改，2 处）：

````old server/internal/modules/project/adapter/postgres/issue_ports_test.go
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
````
````new server/internal/modules/project/adapter/postgres/issue_ports_test.go
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
````
````old server/internal/modules/project/adapter/postgres/issue_ports_test.go
			if rowWaits(t, pool, "workspaces", acme, "FOR UPDATE") {
````
````new server/internal/modules/project/adapter/postgres/issue_ports_test.go
			if pgtest.RowWaits(t, pool, "workspaces", acme, "FOR UPDATE") {
````

`server/internal/modules/project/adapter/postgres/lock_issue_projects_test.go`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/lock_issue_projects_test.go
		if rowWaits(t, pool, "workspaces", acme, "FOR UPDATE") {
````
````new server/internal/modules/project/adapter/postgres/lock_issue_projects_test.go
		if pgtest.RowWaits(t, pool, "workspaces", acme, "FOR UPDATE") {
````

`server/internal/platform/postgres/pgtest/rowwaits.go`（新文件，28 行）：

````file server/internal/platform/postgres/pgtest/rowwaits.go
package pgtest

import (
	"errors"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RowWaits reports whether a statement that takes lock, such as "FOR
// SHARE", on the row id of table, in a transaction of its own, waits for a
// lock another transaction holds on the row: NOWAIT answers
// lock_not_available (55P03) at once instead of waiting. Its wait for a
// connection of pool ends soon (Soon); any other failure fails the test.
func RowWaits(t testing.TB, pool *pgxpool.Pool, table string, id uuid.UUID, lock string) bool {
	t.Helper()
	_, err := pool.Exec(Soon(t), "SELECT 1 FROM "+table+" WHERE id = $1 "+lock+" NOWAIT", id)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "55P03" {
		return true
	}
	if err != nil {
		t.Fatal(err)
	}
	return false
}
````

`server/internal/platform/postgres/pgtest/rowwaits_test.go`（新文件，55 行）：

````file server/internal/platform/postgres/pgtest/rowwaits_test.go
package pgtest_test

import (
	"context"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// RowWaits sees a lock another transaction holds on the row that the lock
// asked for conflicts with, and nothing else: FOR SHARE waits for the row
// held FOR NO KEY UPDATE; not for the one held FOR KEY SHARE, with which it
// does not conflict; not for a row nobody holds; and not for the first once
// its holder is done.
func TestRowWaitsSeesALockThatConflicts(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	pool := newPool(t, pgtest.NewDatabase(t))
	held, keyShared, free := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	if _, err := pool.Exec(ctx, "CREATE TABLE a (id uuid PRIMARY KEY)"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "INSERT INTO a VALUES ($1), ($2), ($3)", held, keyShared, free); err != nil {
		t.Fatal(err)
	}
	holder, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = holder.Rollback(ctx) }()
	for _, lock := range []struct {
		id     uuid.UUID
		clause string
	}{{held, "FOR NO KEY UPDATE"}, {keyShared, "FOR KEY SHARE"}} {
		if _, err := holder.Exec(ctx, "SELECT 1 FROM a WHERE id = $1 "+lock.clause, lock.id); err != nil {
			t.Fatal(err)
		}
	}
	for _, row := range []struct {
		name string
		id   uuid.UUID
		want bool
	}{{"held", held, true}, {"key-shared", keyShared, false}, {"free", free, false}} {
		if got := pgtest.RowWaits(t, pool, "a", row.id, "FOR SHARE"); got != row.want {
			t.Errorf("RowWaits() of the %s row = %v, want %v", row.name, got, row.want)
		}
	}
	if err := holder.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if pgtest.RowWaits(t, pool, "a", held, "FOR SHARE") {
		t.Error("RowWaits() of the held row once its holder committed = true, want false")
	}
}
````

- [ ] **Step 4: 用例和存储**

`server/internal/modules/issue/adapter/postgres/delete_issue.go`（修改，1 处）：

````old server/internal/modules/issue/adapter/postgres/delete_issue.go
}

// DeleteIssueSubscriberRows soft-deletes the work items' subscribers'
````
````new server/internal/modules/issue/adapter/postgres/delete_issue.go
}

// DeleteIssueRelationRows soft-deletes the work items' relations, both
// ways, locked in id order first.
func (s *Store) DeleteIssueRelationRows(ctx context.Context, d app.IssueDeletion) error {
	return deletedRows(s.queries(ctx).DeleteIssueRelationRows(ctx, gen.DeleteIssueRelationRowsParams(deletion(d))), "the relations", d)
}

// DeleteIssueSubscriberRows soft-deletes the work items' subscribers'
````

`server/internal/modules/issue/adapter/postgres/delete_issue_test.go`（修改，1 处）：

````old server/internal/modules/issue/adapter/postgres/delete_issue_test.go
		{"issue_labels", (*postgresadapter.Store).DeleteIssueLabelRows},
````
````new server/internal/modules/issue/adapter/postgres/delete_issue_test.go
		{"issue_labels", (*postgresadapter.Store).DeleteIssueLabelRows},
		{"issue_relations", (*postgresadapter.Store).DeleteIssueRelationRows},
````

`server/internal/modules/issue/adapter/postgres/delete_relations_test.go`（新文件，184 行）：

````file server/internal/modules/issue/adapter/postgres/delete_relations_test.go
package postgresadapter_test

import (
	"context"
	"maps"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// The relations' deletion statements (delete_issue.sql, delete_rows.sql;
// M4 design 3.5, 3.6): the rows of a deleted work item or project both
// ways, whichever project holds them, locked in id order.

// relationRow is a relation as the tests insert it: its id, the project
// that holds it, its issue_id end and its related end.
type relationRow struct {
	id             uuid.UUID
	p              project
	issue, related uuid.UUID
}

// The relations of a deleted subtree are deleted both ways, whichever
// project holds the row. Web's item and sub are deleted; x is another work
// item of web, y and z work items of ops. item relates to x (a row of web,
// item at its issue_id end); x to sub (web's, sub at its related end);
// ops' y to item (a row of ops, item at its related end); sub to ops' z
// (web's, the other end in ops). x relates to y, and z to x, naming
// neither; x's relation to item, which alice deleted an hour before,
// shares item and x's pair. One statement names sub and item, by bob at
// later: the first four rows are deleted at later by bob; the two naming
// neither are as they were, and the one deleted before keeps alice's
// deletion; every other column of every row is kept.
func TestTheRelationsOfADeletedSubtreeAreDeletedBothWays(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, ops := newProject(t, pool, acme, "WEB"), newProject(t, pool, acme, "OPS")
	item, sub, x, y, z := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	for n, id := range []uuid.UUID{item, sub, x} {
		newIssue(t, pool, web, id, n+1, alice)
	}
	newIssue(t, pool, ops, y, 1, alice)
	newIssue(t, pool, ops, z, 2, alice)
	before, later := now.Add(-time.Hour), now.Add(time.Hour)
	xItem := uuid.NewV7()
	newRelation(t, pool, web, x, item, xItem, alice)
	exec(t, pool, "UPDATE issue_relations SET deleted_at = $2, updated_at = $2, updated_by_id = $3 WHERE id = $1", xItem, before, alice)
	named := []relationRow{{uuid.NewV7(), web, item, x}, {uuid.NewV7(), web, x, sub}, {uuid.NewV7(), ops, y, item}, {uuid.NewV7(), web, sub, z}}
	for _, r := range append(slices.Clone(named), relationRow{uuid.NewV7(), web, x, y}, relationRow{uuid.NewV7(), ops, z, x}) {
		newRelation(t, pool, r.p, r.issue, r.related, r.id, alice)
	}
	rows, stamps := rowsAsText(t, pool, "issue_relations"), audits(t, pool, "issue_relations")

	if err := s.DeleteIssueRelationRows(context.Background(), app.IssueDeletion{IssueIDs: []uuid.UUID{sub, item}, By: bob, At: later}); err != nil {
		t.Fatal(err)
	}

	if after := rowsAsText(t, pool, "issue_relations"); !maps.Equal(after, rows) {
		t.Errorf("the rows but their audit columns:\n%v\nwant them as they were:\n%v", after, rows)
	}
	want := maps.Clone(stamps)
	for _, r := range named {
		want[r.id] = deletion(later, bob)
	}
	if after := audits(t, pool, "issue_relations"); !maps.Equal(after, want) {
		t.Errorf("the audit columns:\n%v\nwant\n%v", after, want)
	}
}

// The statement of a deleted subtree's relations locks them FOR NO KEY
// UPDATE in id order before it writes any (convention 5's exception), and
// passes over a row deleted while it waits for it. Web's item has the
// relations a, b and c, drawn in that order, so of ascending ids, item at
// the issue_id end of a and c and the related end of b; they are inserted
// in the opposite order, so that a scan of the table, or of the index on
// either end, meets c first. Another transaction soft-deletes b, as
// another project's deletion does, by alice an hour earlier, and holds it.
// The statement, naming item, waits for b holding a, which a FOR SHARE
// then waits for, and not c, which it has not reached. Once the other
// commits, the statement deletes a and c at the moment and by the account
// given, and b keeps alice's deletion.
func TestTheRelationsOfADeletedSubtreeAreLockedInIDOrder(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	item, p, q, r := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	for n, id := range []uuid.UUID{item, p, q, r} {
		newIssue(t, pool, web, id, n+1, alice)
	}
	a, b, c := uuid.NewV7(), uuid.NewV7(), uuid.NewV7() // drawn in this order: ascending ids
	newRelation(t, pool, web, item, r, c, alice)
	newRelation(t, pool, web, q, item, b, alice)
	newRelation(t, pool, web, item, p, a, alice)
	earlier, later := now.Add(-time.Hour), now.Add(time.Hour)

	soon := pgtest.Soon(t) // ends the test's statements and the statement's wait for b
	held, err := pool.Begin(soon)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = held.Rollback(context.Background()) }()
	if _, err := held.Exec(soon, "UPDATE issue_relations SET deleted_at = $2, updated_at = $2, updated_by_id = $3 WHERE id = $1", b, earlier,
		alice); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		done <- postgres.NewTxManager(pool, 5*time.Second).WithinTx(soon, func(ctx context.Context) error {
			return s.DeleteIssueRelationRows(ctx, app.IssueDeletion{IssueIDs: []uuid.UUID{item}, By: bob, At: later})
		})
	}()
	pgtest.WaitForLockWaitOn(t, pool, "issue_relations", 5*time.Second)
	if !pgtest.RowWaits(t, pool, "issue_relations", a, "FOR SHARE") || pgtest.RowWaits(t, pool, "issue_relations", c, "FOR SHARE") {
		t.Error("while the statement waits for b, a FOR SHARE of a does not wait or one of c does; want a locked first, c not yet")
	}
	if err := held.Commit(soon); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("DeleteIssueRelationRows() = %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("DeleteIssueRelationRows() did not end within 10s")
	}
	got := audits(t, pool, "issue_relations")
	for name, row := range map[string]struct {
		id   uuid.UUID
		want string
	}{"a": {a, deletion(later, bob)}, "b": {b, deletion(earlier, alice)}, "c": {c, deletion(later, bob)}} {
		if got[row.id] != row.want {
			t.Errorf("%s: %s, want %s", name, got[row.id], row.want)
		}
	}
}

// A deleted project's batch of relations takes the rows that name one of
// its work items at either end, whichever project holds them, the work
// items deleted before it among them, as the work items' batches delete
// them first. In web, w1 relates to w2 (a row of web); w2 to ops' o2
// (web's, the other end in ops); ops' o1 to w1 (a row of ops, web's w1 at
// its related end). w1 is deleted. ops' o1 relates to o2, naming no work
// item of web; o2's relation to w1, which alice deleted an hour before,
// keeps her deletion. One batch of web's, by bob at later, takes the three
// rows naming web's work items, and only them, and deletes them.
func TestARelationsBatchTakesTheRowsOfTheProjectsWorkItemsBothWays(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, ops := newProject(t, pool, acme, "WEB"), newProject(t, pool, acme, "OPS")
	w1, w2, o1, o2 := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	newIssue(t, pool, web, w1, 1, alice)
	newIssue(t, pool, web, w2, 2, alice)
	newIssue(t, pool, ops, o1, 1, alice)
	newIssue(t, pool, ops, o2, 2, alice)
	earlier, later := now.Add(-time.Hour), now.Add(time.Hour)
	exec(t, pool, "UPDATE issues SET deleted_at = $2 WHERE id = $1", w1, later)
	named := []relationRow{{uuid.NewV7(), web, w1, w2}, {uuid.NewV7(), web, w2, o2}, {uuid.NewV7(), ops, o1, w1}}
	gone := uuid.NewV7()
	for _, r := range append(slices.Clone(named), relationRow{uuid.NewV7(), ops, o1, o2}, relationRow{gone, ops, o2, w1}) {
		newRelation(t, pool, r.p, r.issue, r.related, r.id, alice)
	}
	exec(t, pool, "UPDATE issue_relations SET deleted_at = $2, updated_at = $2, updated_by_id = $3 WHERE id = $1", gone, earlier, alice)
	stamps := audits(t, pool, "issue_relations")

	ids, err := s.DeleteProjectRelationRows(context.Background(), app.RowBatch{ProjectID: web.id, Limit: 10, By: bob, At: later})
	want := []uuid.UUID{named[0].id, named[1].id, named[2].id}
	if got := slices.SortedFunc(slices.Values(ids), uuid.UUID.Compare); err != nil || !slices.Equal(got, want) {
		t.Errorf("web's batch = %v, %v; want %v", got, err, want)
	}
	for _, r := range named {
		stamps[r.id] = deletion(later, bob)
	}
	if after := audits(t, pool, "issue_relations"); !maps.Equal(after, stamps) {
		t.Errorf("the audit columns:\n%v\nwant\n%v", after, stamps)
	}
}
````

`server/internal/modules/issue/adapter/postgres/delete_rows.go`（修改，1 处）：

````old server/internal/modules/issue/adapter/postgres/delete_rows.go
}

// DeleteProjectSubscriberRows soft-deletes a batch of the project's work
````
````new server/internal/modules/issue/adapter/postgres/delete_rows.go
}

// DeleteProjectRelationRows soft-deletes a batch of the relations of the
// project's work items, both ways.
func (s *Store) DeleteProjectRelationRows(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).DeleteProjectRelationRows(ctx, gen.DeleteProjectRelationRowsParams(batch(b)))
	return deleted(ids, err, "work items' relations")
}

// DeleteProjectSubscriberRows soft-deletes a batch of the project's work
````

`server/internal/modules/issue/adapter/postgres/delete_rows_test.go`（修改，2 处）：

````old server/internal/modules/issue/adapter/postgres/delete_rows_test.go
}

var batchStatements = []batchStatement{
````
````new server/internal/modules/issue/adapter/postgres/delete_rows_test.go
}

// aRelation inserts the relation id of the work item issue of p to a work
// item of p of its own for the n-th row, numbered 200 + n.
func aRelation(t *testing.T, pool *pgxpool.Pool, p project, issue, id, by uuid.UUID, n int) {
	t.Helper()
	related := uuid.NewV7()
	newIssue(t, pool, p, related, 200+n, by)
	newRelation(t, pool, p, issue, related, id, by)
}

var batchStatements = []batchStatement{
````
````old server/internal/modules/issue/adapter/postgres/delete_rows_test.go
	{"issue_labels", (*postgresadapter.Store).DeleteProjectLabelRows, child("issue_labels", "label_id", aLabel)},
````
````new server/internal/modules/issue/adapter/postgres/delete_rows_test.go
	{"issue_labels", (*postgresadapter.Store).DeleteProjectLabelRows, child("issue_labels", "label_id", aLabel)},
	{"issue_relations", (*postgresadapter.Store).DeleteProjectRelationRows, aRelation},
````

`server/internal/modules/issue/adapter/postgres/store_test.go`（修改，1 处）：

````old server/internal/modules/issue/adapter/postgres/store_test.go
		updated_at) VALUES ($1, $2, $3, $4, 'Work', $5, $6, $6, $7, $7)`, id, p.workspace, p.id, p.state, sequence, by, now)
}

````
````new server/internal/modules/issue/adapter/postgres/store_test.go
		updated_at) VALUES ($1, $2, $3, $4, 'Work', $5, $6, $6, $7, $7)`, id, p.workspace, p.id, p.state, sequence, by, now)
}

// newRelation inserts the relation id, a row of p: p's work item issue
// relates to the work item related, of any project, by by at now.
func newRelation(t *testing.T, pool *pgxpool.Pool, p project, issue, related, id, by uuid.UUID) {
	t.Helper()
	exec(t, pool, `INSERT INTO issue_relations (id, workspace_id, project_id, issue_id, related_issue_id, relation_type, created_by_id,
		updated_by_id, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, 'relates_to', $6, $6, $7, $7)`, id, p.workspace, p.id, issue, related,
		by, now)
}

````

`server/internal/modules/issue/app/cascade_test.go`（修改，6 处）：

````old server/internal/modules/issue/app/cascade_test.go
}

func (f *fakeRows) DeleteProjectSubscriberRows(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
````
````new server/internal/modules/issue/app/cascade_test.go
}

func (f *fakeRows) DeleteProjectRelationRows(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	return f.batch(ctx, "relations", b)
}

func (f *fakeRows) DeleteProjectSubscriberRows(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
````
````old server/internal/modules/issue/app/cascade_test.go
// assignee and two activity rows, ops a work item, a label on it and a
// subscriber. Web's work items and its activity each fill a batch of two,
// which fakeRows answers greatest first and least first, in that order.
````
````new server/internal/modules/issue/app/cascade_test.go
// assignee, a relation and two activity rows, ops a work item, a label on
// it, a relation and a subscriber. Web's work items and its activity each
// fill a batch of two, which fakeRows answers greatest first and least
// first, in that order.
````
````old server/internal/modules/issue/app/cascade_test.go
	w, o := ids(6), ids(3)
````
````new server/internal/modules/issue/app/cascade_test.go
	w, o := ids(7), ids(4)
````
````old server/internal/modules/issue/app/cascade_test.go
		"labels":      {ops: o[1:2]},
````
````new server/internal/modules/issue/app/cascade_test.go
		"labels":      {ops: o[1:2]},
		"relations":   {web: w[6:7], ops: o[3:4]},
````
````old server/internal/modules/issue/app/cascade_test.go
	var calls []string
	for _, statement := range []string{"issues", "assignees", "labels", "subscribers", "activities"} {
````
````new server/internal/modules/issue/app/cascade_test.go
	var calls []string
	for _, statement := range []string{"issues", "assignees", "labels", "relations", "subscribers", "activities"} {
````
````old server/internal/modules/issue/app/cascade_test.go
	web, ops, by := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	for _, statement := range []string{"issues", "assignees", "labels", "subscribers", "activities"} {
````
````new server/internal/modules/issue/app/cascade_test.go
	web, ops, by := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	for _, statement := range []string{"issues", "assignees", "labels", "relations", "subscribers", "activities"} {
````

`server/internal/modules/issue/app/delete_issue.go`（修改，2 处）：

````old server/internal/modules/issue/app/delete_issue.go
	DeleteIssueLabelRows(ctx context.Context, d IssueDeletion) error
````
````new server/internal/modules/issue/app/delete_issue.go
	DeleteIssueLabelRows(ctx context.Context, d IssueDeletion) error
	// DeleteIssueRelationRows soft-deletes the relations that name one of
	// the work items at either end, its other end of any project, locked
	// in id order first (convention 5's exception).
	DeleteIssueRelationRows(ctx context.Context, d IssueDeletion) error
````
````old server/internal/modules/issue/app/delete_issue.go
		d.DeleteIssueRows, d.DeleteIssueAssigneeRows, d.DeleteIssueLabelRows, d.DeleteIssueSubscriberRows, d.DeleteIssueActivityRows,
````
````new server/internal/modules/issue/app/delete_issue.go
		d.DeleteIssueRows, d.DeleteIssueAssigneeRows, d.DeleteIssueLabelRows, d.DeleteIssueRelationRows, d.DeleteIssueSubscriberRows,
		d.DeleteIssueActivityRows,
````

`server/internal/modules/issue/app/delete_issue_test.go`（修改，4 处）：

````old server/internal/modules/issue/app/delete_issue_test.go
// their assignees, their labels, their subscribers, their activity.
````
````new server/internal/modules/issue/app/delete_issue_test.go
// their assignees, their labels, their relations, their subscribers, their
// activity.
````
````old server/internal/modules/issue/app/delete_issue_test.go
		line("DeleteIssueSubscriberRows", ids), line("DeleteIssueActivityRows", ids)}
````
````new server/internal/modules/issue/app/delete_issue_test.go
		line("DeleteIssueRelationRows", ids), line("DeleteIssueSubscriberRows", ids), line("DeleteIssueActivityRows", ids)}
````
````old server/internal/modules/issue/app/delete_issue_test.go
			if want := slices.Repeat([]app.IssueDeletion{deletion}, 5); !reflect.DeepEqual(w.issues.deletions, want) {
````
````new server/internal/modules/issue/app/delete_issue_test.go
			if want := slices.Repeat([]app.IssueDeletion{deletion}, 6); !reflect.DeepEqual(w.issues.deletions, want) {
````
````old server/internal/modules/issue/app/delete_issue_test.go
		"DeleteIssueLabelRows", "DeleteIssueSubscriberRows", "DeleteIssueActivityRows"} {
````
````new server/internal/modules/issue/app/delete_issue_test.go
		"DeleteIssueLabelRows", "DeleteIssueRelationRows", "DeleteIssueSubscriberRows", "DeleteIssueActivityRows"} {
````

`server/internal/modules/issue/app/delete_project_rows.go`（修改，2 处）：

````old server/internal/modules/issue/app/delete_project_rows.go
	DeleteProjectLabelRows(ctx context.Context, b RowBatch) ([]uuid.UUID, error)
````
````new server/internal/modules/issue/app/delete_project_rows.go
	DeleteProjectLabelRows(ctx context.Context, b RowBatch) ([]uuid.UUID, error)
	// DeleteProjectRelationRows' batch is of the relations whose either
	// end is a work item of the project.
	DeleteProjectRelationRows(ctx context.Context, b RowBatch) ([]uuid.UUID, error)
````
````old server/internal/modules/issue/app/delete_project_rows.go
		d.DeleteProjectIssueRows, d.DeleteProjectAssigneeRows, d.DeleteProjectLabelRows, d.DeleteProjectSubscriberRows,
		d.DeleteProjectActivityRows,
````
````new server/internal/modules/issue/app/delete_project_rows.go
		d.DeleteProjectIssueRows, d.DeleteProjectAssigneeRows, d.DeleteProjectLabelRows, d.DeleteProjectRelationRows,
		d.DeleteProjectSubscriberRows, d.DeleteProjectActivityRows,
````

`server/internal/modules/issue/app/fakes_deletion_test.go`（修改，1 处）：

````old server/internal/modules/issue/app/fakes_deletion_test.go
}

func (f *fakeIssues) DeleteIssueSubscriberRows(ctx context.Context, d app.IssueDeletion) error {
````
````new server/internal/modules/issue/app/fakes_deletion_test.go
}

func (f *fakeIssues) DeleteIssueRelationRows(ctx context.Context, d app.IssueDeletion) error {
	return f.deleteRows(ctx, "DeleteIssueRelationRows", d)
}

func (f *fakeIssues) DeleteIssueSubscriberRows(ctx context.Context, d app.IssueDeletion) error {
````

- [ ] **Step 5: 组合和端到端的暂时的写法**

`e2e/fixtures/assert/issue.ts`（修改，3 处）：

````old e2e/fixtures/assert/issue.ts
/** The key of a work item's sub-items, which come with P3, as their deletion with their parent's (M4 design 3.6, 3.9). */
const subItems = "issues.parent_id";
````
````new e2e/fixtures/assert/issue.ts
/**
 * The keys of a work item's rows that no story writes yet: its sub-items' (M4 design 3.6, 3.9) and its relations' both
 * ways (3.10), which come with P3 (M4 P3 spec, section 3).
 */
const unwritten = new Set(["issues.parent_id", "issue_relations.issue_id", "issue_relations.related_issue_id"]);
````
````old e2e/fixtures/assert/issue.ts
 * a comment, which names the comment) is under it too. Each key of the graph has such a row, but its sub-items'
 * (subItems), which no work item has before P3; a row deleted before the work item is neither held to its deletion nor
 * checked here (that it keeps its deletion is `TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt`'s).
````
````new e2e/fixtures/assert/issue.ts
 * a comment, which names the comment) is under it too. Each key of the graph has such a row, but those no story writes
 * yet (unwritten); a row deleted before the work item is neither held to its deletion nor checked here (that it keeps
 * its deletion is `TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt`'s).
````
````old e2e/fixtures/assert/issue.ts
  ).toEqual(names.map((key) => ({ key, deletedWithIt: key !== subItems, other: 0 })));
````
````new e2e/fixtures/assert/issue.ts
  ).toEqual(names.map((key) => ({ key, deletedWithIt: !unwritten.has(key), other: 0 })));
````

`e2e/fixtures/assert/project.ts`（修改，3 处）：

````old e2e/fixtures/assert/project.ts
 * project_id. Each table has such a row, and none is left undeleted; a row deleted before the project, as a label a
 * story deleted, is not counted.
````
````new e2e/fixtures/assert/project.ts
 * project_id. Each table but the relations' has such a row, and none is left undeleted; a row deleted before the
 * project, as a label a story deleted, is not counted.
````
````old e2e/fixtures/assert/project.ts
    })
  );
````
````new e2e/fixtures/assert/project.ts
    })
  );
  // No story relates work items yet (M4 P3 spec, section 3): the relations' table has no row to be deleted with the
  // project, and none left undeleted either.
  const withoutRows = new Set(["issue_relations"]);
````
````old e2e/fixtures/assert/project.ts
    tables.map((name) => ({ table: name, deletedWithIt: true, other: 0 }))
````
````new e2e/fixtures/assert/project.ts
    tables.map((name) => ({ table: name, deletedWithIt: !withoutRows.has(name), other: 0 }))
````

`e2e/fixtures/assert/workspace.ts`（修改，4 处）：

````old e2e/fixtures/assert/workspace.ts
 * and their work items' rows. Each table has such a row; none is left undeleted; each table of deletedAlone, whose
 * rows the story deleted alone, has rows deleted earlier, which kept their moment, and every row of the others carries
 * the workspace's.
````
````new e2e/fixtures/assert/workspace.ts
 * and their work items' rows. Each table but the relations' has such a row; none is left undeleted; each table of
 * deletedAlone but the relations', whose rows the story deleted alone, has rows deleted earlier, which kept their
 * moment, and every row of the others carries the workspace's.
````
````old e2e/fixtures/assert/workspace.ts
    })
  );
````
````new e2e/fixtures/assert/workspace.ts
    })
  );
  // No story relates work items yet (M4 P3 spec, section 3): the relations' table has no row to be deleted with the
  // workspace or before it, and none left undeleted either.
  const withoutRows = new Set(["issue_relations"]);
````
````old e2e/fixtures/assert/workspace.ts
      deletedWithIt: true,
````
````new e2e/fixtures/assert/workspace.ts
      deletedWithIt: !withoutRows.has(table),
````
````old e2e/fixtures/assert/workspace.ts
      deletedEarlier: deletedAlone.includes(table),
````
````new e2e/fixtures/assert/workspace.ts
      deletedEarlier: deletedAlone.includes(table) && !withoutRows.has(table),
````

`server/internal/bootstrap/issue_deletion_test.go`（修改，7 处）：

````old server/internal/bootstrap/issue_deletion_test.go
}

// deletedWorkItem is a work item that the caller of token, account, makes
````
````new server/internal/bootstrap/issue_deletion_test.go
}

// relate relates the work item from to the work item to, of any project,
// directly, by by: a relation row of from's project, relates_to (M4 design
// 3.10), seeded so that the test does not depend on which of the issue
// module's writes exist.
func relate(t testing.TB, pool *pgxpool.Pool, from, to, by uuid.UUID) {
	t.Helper()
	if tag, err := pool.Exec(pgtest.Soon(t), `INSERT INTO issue_relations (id, workspace_id, project_id, issue_id, related_issue_id,
		relation_type, created_by_id, updated_by_id) SELECT $1, workspace_id, project_id, id, $3, 'relates_to', $4, $4 FROM issues WHERE id = $2`,
		uuid.NewV7(), from, to, by); err != nil || tag.RowsAffected() != 1 {
		t.Fatalf("relating %s to %s: %d rows, %v; want 1", from, to, tag.RowsAffected(), err)
	}
}

// deletedWorkItem is a work item that the caller of token, account, makes
````
````old server/internal/bootstrap/issue_deletion_test.go
// Bug, with another he makes under it (underWorkItem), and deletes through
// the API: a row of each table under it, deleted.
````
````new server/internal/bootstrap/issue_deletion_test.go
// Bug, with another he makes under it (underWorkItem) and a third, Kin; he
// relates it to Kin and the one under it to it (relate), and deletes it
// through the API: a row of each table under it, deleted.
````
````old server/internal/bootstrap/issue_deletion_test.go
	var made [2]uuid.UUID
````
````new server/internal/bootstrap/issue_deletion_test.go
	var made [3]uuid.UUID
````
````old server/internal/bootstrap/issue_deletion_test.go
		`{"name":"Part"}`} {
````
````new server/internal/bootstrap/issue_deletion_test.go
		`{"name":"Part"}`, `{"name":"Kin"}`} {
````
````old server/internal/bootstrap/issue_deletion_test.go
	underWorkItem(t, pool, gone, made[1])
````
````new server/internal/bootstrap/issue_deletion_test.go
	underWorkItem(t, pool, gone, made[1])
	relate(t, pool, gone, made[2], account)
	relate(t, pool, made[1], gone, account)
````
````old server/internal/bootstrap/issue_deletion_test.go
// assignees and its labels one deleted before, all of bob's writing. The
// other work item, made the same way, has a row of each table under it,
// none deleted before. alice, Web's admin, deletes it: no row under it is
// of her writing before, so a deletion that kept a row's writer would
// show. The deleted work item is not found any more.
````
````new server/internal/bootstrap/issue_deletion_test.go
// assignees and its labels one deleted before, all of bob's writing; and,
// of his writing too, written directly (relate), its relation to a work
// item of Ops, a row of Web's; a relation of another work item of Ops to
// it, a row of Ops'; and its relation to another of Web's, deleted before.
// The other work item, made and related the same way, has a row of each
// table under it, of which only its relation to another of Web's is
// deleted before. alice, Web's admin, deletes it: no row under it is of
// her writing before, so a deletion that kept a row's writer would show.
// The deleted work item is not found any more.
````
````old server/internal/bootstrap/issue_deletion_test.go
		underWorkItem(t, w.pool, parent, w.newWorkItemBy(t, "bob", w.web, `{"name":"Part"}`))
````
````new server/internal/bootstrap/issue_deletion_test.go
		underWorkItem(t, w.pool, parent, w.newWorkItemBy(t, "bob", w.web, `{"name":"Part"}`))
	}
	ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
	for _, item := range []uuid.UUID{deleted, kept} {
		relate(t, w.pool, item, w.newWorkItemIn(t, ops, `{"name":"Far"}`), w.ids["bob"])
		relate(t, w.pool, w.newWorkItemIn(t, ops, `{"name":"Farther"}`), item, w.ids["bob"])
		near := w.newWorkItemBy(t, "bob", w.web, `{"name":"Near"}`)
		relate(t, w.pool, item, near, w.ids["bob"])
		if tag, err := w.pool.Exec(pgtest.Soon(t), `UPDATE issue_relations SET deleted_at = now(), updated_at = now() WHERE issue_id = $1
			AND related_issue_id = $2`, item, near); err != nil || tag.RowsAffected() != 1 {
			t.Fatalf("deleting the relation to Near: %d rows, %v; want 1", tag.RowsAffected(), err)
		}
````

`server/internal/bootstrap/issue_interleaving_deletion_test.go`（修改，1 处）：

````old server/internal/bootstrap/issue_interleaving_deletion_test.go
			UNION ALL SELECT deleted_at, updated_at, updated_by_id, created_at FROM issue_labels WHERE project_id = $2
````
````new server/internal/bootstrap/issue_interleaving_deletion_test.go
			UNION ALL SELECT deleted_at, updated_at, updated_by_id, created_at FROM issue_labels WHERE project_id = $2
			UNION ALL SELECT deleted_at, updated_at, updated_by_id, created_at FROM issue_relations WHERE project_id = $2
````

`server/internal/bootstrap/issue_interleaving_test.go`（修改，2 处）：

````old server/internal/bootstrap/issue_interleaving_test.go
// work items and the work items' assignees, labels, subscribers and
// activities; the deleted ones too, unless undeleted.
````
````new server/internal/bootstrap/issue_interleaving_test.go
// work items and the work items' assignees, labels, relations,
// subscribers and activities; the deleted ones too, unless undeleted.
````
````old server/internal/bootstrap/issue_interleaving_test.go
	for _, table := range []string{"issues", "issue_assignees", "issue_labels", "issue_subscribers", "issue_activities"} {
````
````new server/internal/bootstrap/issue_interleaving_test.go
	for _, table := range []string{"issues", "issue_assignees", "issue_labels", "issue_relations", "issue_subscribers", "issue_activities"} {
````

`server/internal/bootstrap/issue_world_test.go`（修改，2 处）：

````old server/internal/bootstrap/issue_world_test.go
// five tables, each a line, sorted: its table, who made it and changed it
// last, and when, when it was deleted if it was, and whose it is, for an
// assignee, a subscriber or an activity (the account assigned, subscribed
// or acting). An account goes by its name in the world, a moment by the
// work item's: made, its creation, or changed, its last change.
````
````new server/internal/bootstrap/issue_world_test.go
// six tables, its relations either way, each a line, sorted: its table,
// who made it and changed it last, and when, when it was deleted if it
// was, and whose it is, for an assignee, a subscriber or an activity (the
// account assigned, subscribed or acting). An account goes by its name in
// the world, a moment by the work item's: made, its creation, or changed,
// its last change.
````
````old server/internal/bootstrap/issue_world_test.go
		UNION ALL SELECT 'issue_labels', created_by_id, created_at, updated_by_id, updated_at, deleted_at, NULL FROM issue_labels
			WHERE issue_id = $1
````
````new server/internal/bootstrap/issue_world_test.go
		UNION ALL SELECT 'issue_labels', created_by_id, created_at, updated_by_id, updated_at, deleted_at, NULL FROM issue_labels
			WHERE issue_id = $1
		UNION ALL SELECT 'issue_relations', created_by_id, created_at, updated_by_id, updated_at, deleted_at, NULL FROM issue_relations
			WHERE issue_id = $1 OR related_issue_id = $1
````

`server/internal/bootstrap/issue_write_locks_test.go`（修改，1 处）：

````old server/internal/bootstrap/issue_write_locks_test.go
	"issue_subscribers, issue_activities"
````
````new server/internal/bootstrap/issue_write_locks_test.go
	"issue_relations, issue_subscribers, issue_activities"
````

`server/internal/bootstrap/project_deletion_test.go`（修改，5 处）：

````old server/internal/bootstrap/project_deletion_test.go
// and its subscriber, the project's label Bug (seedLabels) on it, and its
````
````new server/internal/bootstrap/project_deletion_test.go
// and its subscriber, the project's label Bug (seedLabels) on it, its
// relation to another work item of the project, its blocker, and its
````
````old server/internal/bootstrap/project_deletion_test.go
// issue module's writes exist. Its number is the project's counter, moved
````
````new server/internal/bootstrap/project_deletion_test.go
// issue module's writes exist. Each number is the project's counter, moved
````
````old server/internal/bootstrap/project_deletion_test.go
	issue := uuid.NewV7()
````
````new server/internal/bootstrap/project_deletion_test.go
	issue, blocker := uuid.NewV7(), uuid.NewV7()
````
````old server/internal/bootstrap/project_deletion_test.go
			WHERE i.id = $2`, []any{uuid.NewV7(), issue, by}},
````
````new server/internal/bootstrap/project_deletion_test.go
			WHERE i.id = $2`, []any{uuid.NewV7(), issue, by}},
		{`UPDATE projects SET last_issue_sequence = last_issue_sequence + 1 WHERE id = $1`, []any{project}},
		{`INSERT INTO issues (id, workspace_id, project_id, state_id, name, sequence_id, created_by_id, updated_by_id)
			SELECT $1, workspace_id, project_id, state_id, 'Blocker', (SELECT last_issue_sequence FROM projects WHERE id = $2), $3, $3
			FROM issues WHERE id = $4`, []any{blocker, project, by, issue}},
		{`INSERT INTO issue_relations (id, workspace_id, project_id, issue_id, related_issue_id, relation_type, created_by_id, updated_by_id)
			SELECT $1, workspace_id, project_id, id, $3, 'blocked_by', $4, $4 FROM issues WHERE id = $2`, []any{uuid.NewV7(), issue, blocker, by}},
````
````old server/internal/bootstrap/project_deletion_test.go
}

// Every statement of the deletion runs in its one transaction (M3 design
````
````new server/internal/bootstrap/project_deletion_test.go
}

// Deleting a project deletes the relations of its work items either way
// (M4 design 3.6, 3.10), each at the project's deleted_at, whichever
// project's row it is: beside the seeded relations, Web's Seeded related
// to Ops' Seeded, a row of Web's, and Ops' Blocker related to Web's
// Seeded, a row of Ops', written directly (relate). The second is no row
// under Web, which the catalog's test of the deletion reads
// (TestDeletingAProjectLeavesNoUndeletedRowUnderIt). Ops' own relation,
// of its Seeded to its Blocker, stays as it was.
func TestDeletingAProjectDeletesItsWorkItemsRelationsEitherWay(t *testing.T) {
	contract, base, pool, alice, aliceID, web, ops := twoProjects(t)
	const items = "SELECT id FROM issues WHERE project_id = $1 ORDER BY sequence_id"
	webItems, opsItems := queryIDs(t, pool, items, web), queryIDs(t, pool, items, ops)
	if len(webItems) != 2 || len(opsItems) != 2 {
		t.Fatalf("the work items %v of Web and %v of Ops; want Seeded and Blocker of each", webItems, opsItems)
	}
	seeded := webItems[0]
	relate(t, pool, seeded, opsItems[0], aliceID)
	relate(t, pool, opsItems[1], seeded, aliceID)
	const opsOwn = `SELECT coalesce(string_agg(id::text || ' ' || coalesce(deleted_at::text, 'undeleted'), ', ' ORDER BY id), '')
		FROM issue_relations WHERE project_id = $1 AND related_issue_id <> $2`
	var before, after string
	if err := pool.QueryRow(pgtest.Soon(t), opsOwn, ops, seeded).Scan(&before); err != nil || before == "" {
		t.Fatalf("Ops' own relations: %q, %v; want its seeded one", before, err)
	}

	if status, body := call(t, contract, http.MethodDelete, base+"/api/v0/projects/"+web.String(), alice, ""); status != http.StatusNoContent {
		t.Fatalf("deleting Web = %d %s, want 204", status, body)
	}

	var left, all int
	if err := pool.QueryRow(pgtest.Soon(t), `SELECT count(*) FILTER (WHERE r.deleted_at IS DISTINCT FROM p.deleted_at), count(*)
		FROM issue_relations r, projects p WHERE p.id = $1 AND $2 IN (r.issue_id, r.related_issue_id)`, web, seeded).Scan(&left, &all); err != nil ||
		left != 0 || all != 3 {
		t.Errorf("Web's Seeded's relations: %d of %d not deleted at Web's deleted_at, %v; want none of 3", left, all, err)
	}
	if err := pool.QueryRow(pgtest.Soon(t), opsOwn, ops, seeded).Scan(&after); err != nil || after != before {
		t.Errorf("Ops' own relations: %q, %v; want them as they were: %q", after, err, before)
	}
}

// Every statement of the deletion runs in its one transaction (M3 design
````

- [ ] **Step 6: 测试和 lint**

Run: `go -C server test -count=1 ./migrations/ ./internal/platform/postgres/pgtest/ ./internal/modules/issue/... ./internal/modules/project/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt$|TestDeletingAWorkItemDeletesItsSubtreeAtOneMoment$|TestDeletingAProjectDeletesItsWorkItemsRelationsEitherWay$|TestDeletingAProjectLeavesNoUndeletedRowUnderIt$|TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt$|TestEachIssueWriteLocksItsProjectFirst$|TestDeletingAWorkItemAndChangingItSerialize$' ./internal/bootstrap/`
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
Expected: 123 个全部通过（删除项目、工作区的故事 P4、W2、W3 暂时不要关联表的行）。

- [ ] **Step 7: 提交**

```bash
git add api/modules/issue.yaml api/modules/project.yaml api/modules/workspace.yaml deploy/runtime-grants.sql e2e/fixtures/assert/issue.ts e2e/fixtures/assert/project.ts e2e/fixtures/assert/workspace.ts server/internal/bootstrap/issue_deletion_test.go server/internal/bootstrap/issue_interleaving_deletion_test.go server/internal/bootstrap/issue_interleaving_test.go server/internal/bootstrap/issue_world_test.go server/internal/bootstrap/issue_write_locks_test.go server/internal/bootstrap/project_deletion_test.go server/internal/modules/issue/adapter/postgres/delete_issue.go server/internal/modules/issue/adapter/postgres/delete_issue_test.go server/internal/modules/issue/adapter/postgres/delete_relations_test.go server/internal/modules/issue/adapter/postgres/delete_rows.go server/internal/modules/issue/adapter/postgres/delete_rows_test.go server/internal/modules/issue/adapter/postgres/queries/delete_issue.sql server/internal/modules/issue/adapter/postgres/queries/delete_rows.sql server/internal/modules/issue/adapter/postgres/store_test.go server/internal/modules/issue/app/cascade_test.go server/internal/modules/issue/app/delete_issue.go server/internal/modules/issue/app/delete_issue_test.go server/internal/modules/issue/app/delete_project_rows.go server/internal/modules/issue/app/fakes_deletion_test.go server/internal/modules/project/adapter/postgres/demote_test.go server/internal/modules/project/adapter/postgres/issue_ports_test.go server/internal/modules/project/adapter/postgres/lock_issue_projects_test.go server/internal/platform/postgres/pgtest/rowwaits.go server/internal/platform/postgres/pgtest/rowwaits_test.go server/migrations/issue_schema_test.go server/migrations/schema_test.go server/migrations/sql/00020_issue_issue_relations.sql server/sqlc.yaml api/dist/openapi.yaml server/internal/modules/issue/adapter/postgres/gen/delete_issue.sql.go server/internal/modules/issue/adapter/postgres/gen/delete_rows.sql.go server/internal/modules/issue/adapter/postgres/gen/models.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M4/P3): the relations' table, and the deletions take its rows both ways

Migration 00020 adds issue_relations, one undeleted row a pair of work
items whatever its type and direction, blocking stored as the other's
blocked_by. Deleting a work item's subtree, and a project's batches,
soft-delete the relations naming their work items at either end, of any
project: another project's writes reach those rows without this
project's lock, so each statement locks them in id order before it
writes (convention 5's exception), in the relations' place in the global
order. pgtest.RowWaits replaces a hand-written 55P03. Until a story can
relate work items, the e2e deletions do not ask for a relation row.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p3.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试，或没有替身的平台、领域、转换的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；守卫是生成器（`make gen-go`）和中英文翻译键（`make lint-web`）的核对。每个变异在一份新的副本里，在本 Task 的树上、标出的较晚的 Task 的树上和最终的树上都跑过，端到端的检查只在标出的 Task 的树上）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T8.1` | 一对工作项可以有两条关联，每个方向一条 | `migrations` 的全部测试（`make test`） | 结构 |
| `T8.2` | 工作项可以与它自己关联 | `migrations` 的全部测试（`make test`） | 结构 |
| `T8.3` | 关联可以存成 `blocking` | `migrations` 的全部测试（`make test`） | 结构 |
| `T8.4` | 已删除的关联挡住这对工作项的新关联 | `migrations` 的全部测试（`make test`） | 结构 |
| `T8.5` | 删除的工作项的关联只删一个方向：它在 `related_issue_id` 一端的留着 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt`（`make test`）、故事 `i17-delete-issue`（Task 15 起） | 存储；组合；端到端 |
| `T8.6` | 删除的工作项的关联按语句扫到的次序加锁，而不是先按 id 的次序锁住 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`） | 存储 |
| `T8.7` | 删除的工作项早先已删除的关联被这次删除重写 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt`（`make test`） | 存储；组合 |
| `T8.8` | 删除的项目的关联只有它自己的行：别的项目里指向它的工作项的行留着 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingAProjectDeletesItsWorkItemsRelationsEitherWay`（`make test`）、`internal/bootstrap` 的 `TestDeletingARelatedWorkItemAndTheOtherEndsProjectSerialize`（`make test`）（Task 14 起） | 存储；组合 |
| `T8.9` | 删除的项目在 `related_issue_id` 一端的关联只算它未删除的工作项的 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingAProjectDeletesItsWorkItemsRelationsEitherWay`（`make test`） | 存储；组合 |
| `T8.10` | 删除工作项不删关联 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt`（`make test`）、`internal/bootstrap` 的 `TestDeletingARelatedWorkItemAndTheOtherEndsProjectSerialize`（`make test`）（Task 14 起）、故事 `i17-delete-issue`（Task 15 起） | 单元；组合；端到端 |
| `T8.11` | 删除项目、工作区不删关联 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingAProjectLeavesNoUndeletedRowUnderIt`（`make test`）、`internal/bootstrap` 的 `TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt`（`make test`）、`internal/bootstrap` 的 `TestRelatingAcrossProjectsAndDeletingTheWorkspaceSerialize`（`make test`）（Task 14 起）、故事 `p4-archive`（Task 15 起）、故事 `w3-workspace-settings.spec`（Task 15 起） | 单元；组合；端到端 |
| `T8.12` | 删除工作项先删关联、后删标签，不在全局顺序上 | `internal/modules/issue/app` 的全部测试（`make test`） | 单元 |
| `T8.13` | 运行时的角色不能写关联 | `internal/bootstrap` 的 `TestTheGrantsFileCoversEveryRelationAndFunction`（`make test`） | 组合 |
| `T8.14` | `pgtest.RowWaits` 把别的事务持着的行读成没人持着 | `internal/platform/postgres/pgtest` 的 `TestRowWaitsSeesALockThatConflicts`（`make test`）、`internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/modules/project/adapter/postgres` 的全部测试（`make test`） | 存储 |

---

### Task 9: 关联：领域和存储

**Files:**
- Create: `server/internal/modules/issue/adapter/postgres/queries/relations.sql`、`server/internal/modules/issue/adapter/postgres/relations.go`、`server/internal/modules/issue/adapter/postgres/relations_test.go`、`server/internal/modules/issue/domain/relation.go`、`server/internal/modules/issue/domain/relation_test.go`
- Modify: `server/internal/modules/issue/app/rows.go`
- Generate: `server/internal/modules/issue/adapter/postgres/gen/relations.sql.go`

**Interfaces:**（spec 2.11、第 3 节第 12、14 条；M4 设计 3.7、3.10、3.13、4.6）
- `issue/domain/relation.go`（新）：`RelationType` 和四个类型（`relates_to`、`duplicate`、`blocked_by`、`blocking`：页面的四种）；`Reversed()`（另一端读到的：`blocked_by`、`blocking` 互为反面，另两种是自己）；`MaxRelatedIssues`（20）；`CheckNewRelations(t, ids)`（类型是四种之一、至少一个、至多 20 个、各一次，在任何读之前，一个 422；Plane 的 `start_before` 页面不显示，不收）；`RelatedNotAllowed(i)`（`issue_ids[i]` 的 `not_allowed`）、`RelatedAlready(i)`（`issue_ids[i]` 的 `duplicate`），各一句固定的文字；`Relation`（如存的：工作区、`issue_id` 那一端的项目、两端、存储的类型（没有 `blocking`）、谁、何时）；`NewRelation(id, from, to, t, by, at)`（`blocking` 存为对方的 `blocked_by`，那一行是对方的项目的）；`TypeFrom(id)`、`Other(id)`；`RelationAdded(t, other)`（动态第 20、21 行：类型作为 `field`、`updated`，`""` 到对方的标识，对方的 `id` 是 `new_identifier`，`comment` 照 Plane 带上类型）。
- `issue/adapter/postgres/queries/relations.sql`（新）：`RelatedIssues`（给的工作项中哪些与它已有未删除的关联，任何类型、任一个方向，`id` 的次序；在两端的项目的锁下读）、`InsertRelations`（一条语句插入几条：`unnest` 各列，一个工作区、一个账户、一个时刻）、`MarkChanged`（`:execrows`：两端的工作项的最后修改者和时刻，只写未删除的）。`relations.go`（新）：`Related`、`InsertRelations`（不是一个工作区、一个账户、一个时刻的几条是错误，什么都不写）、`MarkChanged`（写的行数不是给的个数时是错误）。
- `issue/app/rows.go`：`ChangedRows{IDs, By, At}`（没改字段、只改了最后修改的工作项：关联的两端，`id` 的次序）。

**Tests:**
- 领域：`TestRelationTypeReversed`、`TestCheckNewRelations`（每种类型和一个、二十个通过；`start_before`、没有、二十一个、重复的各报在自己的位置，同一个 422）、`TestNewRelation`（从 Web 的工作项到 Ops 的：`relates_to`、`duplicate`、`blocked_by` 是问的那一端的、Web 的行；`blocking` 是对方的、Ops 的 `blocked_by`；每一端读到的类型和对方）、`TestRelationAdded`（两行同样的列，类型照它的工作项读的）。
- 存储：`TestRelated`（任何类型、任一个方向、任何项目的；已删除的不算；`id` 的次序；没有关联的回答空的；取消的读是它的失败）、`TestInsertRelations`（ops 的 o 被 web 的 item 阻塞（ops 的行）、item 到 web 的 b（web 的行），每一列照给的；两个工作区、两个时刻、两个账户的几条各被拒绝、什么都不写：每组都与 `alone`（c 到 item 的，之后单独写下）并列，所以拒绝的是这一条守卫，不是一对的键；已有关联的一对（任何类型、任一个方向，`issue_relations_pair_key`）、取消的插入也是）、`TestMarkChanged`（web 的 a、ops 的 o 由 bob 在 later 改，别的列不动；没给的 b 不动；给了已删除的是错误，取消的写也是）。

- [ ] **Step 1: 查询，生成**

`server/internal/modules/issue/adapter/postgres/queries/relations.sql`（新文件，29 行）：

````file server/internal/modules/issue/adapter/postgres/queries/relations.sql
-- name: RelatedIssues :many
-- Which of the work items ids relate to the work item id (M4 design 3.10): an undeleted relation between the two, of
-- any type, either way, in the other's id order. Read under the locks of both ends' projects, one of which every write
-- of a relation between them holds at least (3.5, convention 7), it stays as read until the write commits. The indexes
-- on issue_id and on related_issue_id serve it.
SELECT (CASE WHEN r.issue_id = sqlc.arg(id)::uuid THEN r.related_issue_id ELSE r.issue_id END)::uuid AS other
FROM issue_relations r
WHERE r.deleted_at IS NULL
  AND ((r.issue_id = sqlc.arg(id)::uuid AND r.related_issue_id = ANY (sqlc.arg(ids)::uuid[]))
    OR (r.related_issue_id = sqlc.arg(id)::uuid AND r.issue_id = ANY (sqlc.arg(ids)::uuid[])))
ORDER BY 1;

-- name: InsertRelations :exec
-- New relations of work items (M4 design 3.10), one statement: ids[i] the row of the i-th, in the order given (the
-- other work items' id order, convention 5's second half), each with its project, its ends and its type as stored, at
-- the moment and by the account given.
INSERT INTO issue_relations (id, workspace_id, project_id, issue_id, related_issue_id, relation_type, created_by_id, updated_by_id,
                             created_at, updated_at)
SELECT unnest(sqlc.arg(ids)::uuid[]), sqlc.arg(workspace_id), unnest(sqlc.arg(project_ids)::uuid[]),
       unnest(sqlc.arg(issue_ids)::uuid[]), unnest(sqlc.arg(related_issue_ids)::uuid[]), unnest(sqlc.arg(relation_types)::text[]),
       sqlc.arg(by)::uuid, sqlc.arg(by), sqlc.arg(at)::timestamptz, sqlc.arg(at);

-- name: MarkChanged :execrows
-- The work items a relation's addition changes last without a field of theirs (M4 design 3.7, 3.10): both its ends,
-- each last changed at the moment and by the account given; the undeleted ones among ids. Under the locks of their
-- projects, which every write of the projects' work items holds (3.5): one statement in scan order (convention 5).
UPDATE issues
SET updated_by_id = sqlc.arg(by)::uuid, updated_at = sqlc.arg(at)
WHERE id = ANY (sqlc.arg(ids)::uuid[]) AND deleted_at IS NULL;
````

Run: `make gen-go`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `580f747dac78ee62e94a604455ee029b27801243cdce7a293fac3cfd89c62657` | 110 | `server/internal/modules/issue/adapter/postgres/gen/relations.sql.go` |

Run: `shasum -a 256 server/internal/modules/issue/adapter/postgres/gen/relations.sql.go`
Expected: 与上表相同。

- [ ] **Step 2: 领域**

`server/internal/modules/issue/domain/relation.go`（新文件，132 行）：

````file server/internal/modules/issue/domain/relation.go
package domain

import (
	"fmt"
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// A relation between two work items (M4 design 3.10): one row a pair,
// whatever its type and direction. relates_to and duplicate read the same
// from both ends; blocked_by reads as blocking from the work item that
// blocks, so a relation asked for as blocking is stored the other way
// round, as the other work item's blocked_by (Plane the same). The work
// items may be of two projects of a workspace.

// RelationType is a relation's type as one of its work items reads it.
type RelationType string

// The types a caller names, from his work item (3.10): the page's four.
const (
	RelatesTo RelationType = "relates_to"
	Duplicate RelationType = "duplicate"
	BlockedBy RelationType = "blocked_by"
	Blocking  RelationType = "blocking"
)

// relationTypes are the types a caller may name.
var relationTypes = []RelationType{RelatesTo, Duplicate, BlockedBy, Blocking}

// Reversed is t as the relation's other end reads it: blocked_by and
// blocking each the other; relates_to and duplicate themselves.
func (t RelationType) Reversed() RelationType {
	switch t {
	case BlockedBy:
		return Blocking
	case Blocking:
		return BlockedBy
	}
	return t
}

// MaxRelatedIssues is how many work items an addition of relations names
// at most (M4 design 3.10, 5.1).
const MaxRelatedIssues = 20

// CheckNewRelations checks an addition of relations of the type t to the
// work items ids, before any read (M4 design 3.10): t one of the four, at
// least one work item, at most MaxRelatedIssues, each named once; every
// problem in one 422 validation_failed. Whether each may be related is the
// use case's, under the projects' locks (RelatedNotAllowed,
// RelatedAlready).
func CheckNewRelations(t RelationType, ids []uuid.UUID) error {
	var found []*shared.FieldError
	if !slices.Contains(relationTypes, t) {
		found = append(found, &shared.FieldError{Field: "relation_type", Code: shared.FieldInvalidFormat,
			Message: "must be one of relates_to, duplicate, blocked_by, blocking"})
	}
	if len(ids) == 0 {
		found = append(found, &shared.FieldError{Field: "issue_ids", Code: shared.FieldTooShort, Message: "must name a work item"})
	}
	return invalid(append(found, checkIDs("issue_ids", ids, MaxRelatedIssues)...)...)
}

// RelatedNotAllowed is the problem of the work item at index i of those an
// addition of relations names that the work item may not be related to
// (M4 design 3.10).
func RelatedNotAllowed(i int) shared.FieldError {
	return shared.FieldError{Field: fmt.Sprintf("issue_ids[%d]", i), Code: shared.FieldNotAllowed,
		Message: "must be another work item of the workspace that you see, not archived, in a project that is not archived"}
}

// RelatedAlready is the problem of the work item at index i of those an
// addition of relations names that relates to the work item already, in
// either direction, of any type (M4 design 3.10).
func RelatedAlready(i int) shared.FieldError {
	return shared.FieldError{Field: fmt.Sprintf("issue_ids[%d]", i), Code: shared.FieldDuplicate, Message: "relates to this work item already"}
}

// Relation is a relation as stored (M4 design 4.6): its id; its work item
// at the issue_id end, of the project ProjectID; its work item at the
// related end; its type as stored, never blocking; who made it, and when.
type Relation struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	ProjectID   uuid.UUID
	IssueID     uuid.UUID
	RelatedID   uuid.UUID
	Type        RelationType
	CreatedByID *uuid.UUID
	CreatedAt   time.Time
}

// NewRelation is the relation id that the account by makes at at, of the
// type t from the work item from to the work item to, as stored: from's
// row, blocking stored the other way round, as to's blocked_by, to's row.
func NewRelation(id uuid.UUID, from, to Issue, t RelationType, by uuid.UUID, at time.Time) Relation {
	r := Relation{ID: id, WorkspaceID: from.WorkspaceID, ProjectID: from.ProjectID, IssueID: from.ID, RelatedID: to.ID, Type: t,
		CreatedByID: &by, CreatedAt: at}
	if t == Blocking {
		r.ProjectID, r.IssueID, r.RelatedID, r.Type = to.ProjectID, to.ID, from.ID, BlockedBy
	}
	return r
}

// TypeFrom is r's type as its work item id reads it: as stored from its
// issue_id end, reversed from its related end.
func (r Relation) TypeFrom(id uuid.UUID) RelationType {
	if id == r.RelatedID {
		return r.Type.Reversed()
	}
	return r.Type
}

// Other is r's end that is not the work item id.
func (r Relation) Other(id uuid.UUID) uuid.UUID {
	if id == r.RelatedID {
		return r.IssueID
	}
	return r.RelatedID
}

// RelationAdded is the row of an added relation on one of its work items
// (rows 20 and 21 of M4 design 3.13, of the same columns): its type as the
// work item reads it, updated, from "" to the other work item's
// identifier, the other's id the new identifier.
func RelationAdded(t RelationType, other Ref) Activity {
	return Activity{Field: ptr(string(t)), Verb: VerbUpdated, OldValue: ptr(""), NewValue: ptr(other.Identifier), NewIdentifier: &other.ID,
		Comment: "added " + string(t) + " relation"}
}
````

`server/internal/modules/issue/domain/relation_test.go`（新文件，109 行）：

````file server/internal/modules/issue/domain/relation_test.go
package domain

import (
	"errors"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Each type reads from a relation's other end as the page shows it there
// (M4 design 3.10): blocked_by and blocking each as the other, relates_to
// and duplicate as themselves.
func TestRelationTypeReversed(t *testing.T) {
	for t1, t2 := range map[RelationType]RelationType{BlockedBy: Blocking, Blocking: BlockedBy, RelatesTo: RelatesTo, Duplicate: Duplicate} {
		if got := t1.Reversed(); got != t2 {
			t.Errorf("%s.Reversed() = %s, want %s", t1, got, t2)
		}
	}
}

// An addition of relations names one of the four types and one work item
// at least, twenty at most, each once (M4 design 3.10): each type and one
// work item, or twenty, are accepted; Plane's start_before, which the page
// does not show, is refused at relation_type, no work item at issue_ids,
// twenty-one too, alone, and each work item named again at its index;
// every problem in one 422.
func TestCheckNewRelations(t *testing.T) {
	a, b := uuid.NewV7(), uuid.NewV7()
	for _, kind := range relationTypes {
		if err := CheckNewRelations(kind, []uuid.UUID{a}); err != nil {
			t.Errorf("CheckNewRelations(%s, one) = %v, want nil", kind, err)
		}
	}
	unknown := shared.FieldError{Field: "relation_type", Code: "invalid_format", Message: "must be one of relates_to, duplicate, blocked_by, blocking"}
	for _, tt := range []struct {
		name string
		kind RelationType
		ids  []uuid.UUID
		want []shared.FieldError
	}{
		{"twenty", BlockedBy, ids(MaxRelatedIssues), nil},
		{"start_before, none", "start_before", nil, []shared.FieldError{unknown,
			{Field: "issue_ids", Code: "too_short", Message: "must name a work item"}}},
		{"twenty-one, one twice", RelatesTo, append(ids(MaxRelatedIssues), a, a)[1:],
			[]shared.FieldError{{Field: "issue_ids", Code: "too_long", Message: "must name at most 20"}}},
		{"blocking in another case, two named again", "Blocking", []uuid.UUID{a, b, a, b}, []shared.FieldError{unknown,
			{Field: "issue_ids[2]", Code: "duplicate", Message: "is listed before"},
			{Field: "issue_ids[3]", Code: "duplicate", Message: "is listed before"}}},
	} {
		err := CheckNewRelations(tt.kind, tt.ids)
		var se *shared.Error
		if tt.want == nil && err != nil || tt.want != nil && (!errors.As(err, &se) || se.Code != "validation_failed" ||
			!reflect.DeepEqual(se.Fields, tt.want)) {
			t.Errorf("%s: CheckNewRelations() = %v, want %+v", tt.name, err, tt.want)
		}
	}
	for got, want := range map[shared.FieldError]shared.FieldError{
		RelatedNotAllowed(3): {Field: "issue_ids[3]", Code: "not_allowed",
			Message: "must be another work item of the workspace that you see, not archived, in a project that is not archived"},
		RelatedAlready(0): {Field: "issue_ids[0]", Code: "duplicate", Message: "relates to this work item already"},
	} {
		if got != want {
			t.Errorf("the problem %+v, want %+v", got, want)
		}
	}
}

// A relation is stored one row a pair (M4 design 3.10): asked for from a
// work item of Web to one of Ops, relates_to, duplicate and blocked_by are
// the row of the work item asked from, of Web, at its issue_id end; blocking
// is the row of the other, of Ops, its blocked_by, the work item asked
// from at its related end. Each end reads the type as asked from it, the
// other end its reverse; each end's other is the other work item.
func TestNewRelation(t *testing.T) {
	acme, web, ops, by := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	from, to := Issue{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web}, Issue{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: ops}
	at := time.Date(2026, 10, 21, 10, 0, 0, 0, time.UTC)
	for _, kind := range relationTypes {
		id := uuid.NewV7()
		r := NewRelation(id, from, to, kind, by, at)
		want := Relation{ID: id, WorkspaceID: acme, ProjectID: web, IssueID: from.ID, RelatedID: to.ID, Type: kind, CreatedByID: &by, CreatedAt: at}
		if kind == Blocking {
			want.ProjectID, want.IssueID, want.RelatedID, want.Type = ops, to.ID, from.ID, BlockedBy
		}
		if !reflect.DeepEqual(r, want) {
			t.Errorf("NewRelation(%s) = %+v, want %+v", kind, r, want)
		}
		if r.TypeFrom(from.ID) != kind || r.TypeFrom(to.ID) != kind.Reversed() || r.Other(from.ID) != to.ID || r.Other(to.ID) != from.ID {
			t.Errorf("%s: from the work item asked from %s and %s, from the other %s and %s; want %s and %s, %s and %s", kind, r.TypeFrom(from.ID),
				r.Other(from.ID), r.TypeFrom(to.ID), r.Other(to.ID), kind, to.ID, kind.Reversed(), from.ID)
		}
	}
}

// An added relation writes a row on each of its work items, of the same
// columns (rows 20 and 21 of M4 design 3.13): the type as the work item
// reads it, updated, from "" to the other's identifier, the other's id
// the new identifier, Plane's sentence with the type.
func TestRelationAdded(t *testing.T) {
	other := Ref{ID: uuid.NewV7(), Identifier: "OPS-3"}
	want := Activity{Field: ptr("blocking"), Verb: VerbUpdated, OldValue: ptr(""), NewValue: ptr("OPS-3"), NewIdentifier: &other.ID,
		Comment: "added blocking relation"}
	if got := RelationAdded(Blocking, other); !reflect.DeepEqual(got, want) {
		t.Errorf("RelationAdded() = %+v, want %+v", got, want)
	}
}
````

- [ ] **Step 3: 存储和它取的 `ChangedRows`**

`server/internal/modules/issue/adapter/postgres/relations.go`（新文件，59 行）：

````file server/internal/modules/issue/adapter/postgres/relations.go
package postgresadapter

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// The relations between work items (relations.sql; M4 design 3.10, 4.6).

// Related reads which of the work items ids relate to the work item id: an
// undeleted relation between the two, of any type, either way, in id
// order.
func (s *Store) Related(ctx context.Context, id uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error) {
	related, err := s.queries(ctx).RelatedIssues(ctx, gen.RelatedIssuesParams{ID: id, Ids: ids})
	if err != nil {
		return nil, fmt.Errorf("read the relations of work item %s to %v: %w", id, ids, err)
	}
	return related, nil
}

// InsertRelations inserts the relations rs in their order, one statement;
// they are of one workspace, made by one account at one moment, as one
// write makes them (M4 design 3.10).
func (s *Store) InsertRelations(ctx context.Context, rs []domain.Relation) error {
	if len(rs) == 0 {
		return nil
	}
	p := gen.InsertRelationsParams{WorkspaceID: rs[0].WorkspaceID, By: rs[0].CreatedByID, At: rs[0].CreatedAt}
	for _, r := range rs {
		if r.WorkspaceID != p.WorkspaceID || r.CreatedByID == nil || p.By == nil || *r.CreatedByID != *p.By || !r.CreatedAt.Equal(p.At) {
			return fmt.Errorf("insert relations %+v: not of one workspace, account and moment", rs)
		}
		p.Ids, p.ProjectIds, p.IssueIds = append(p.Ids, r.ID), append(p.ProjectIds, r.ProjectID), append(p.IssueIds, r.IssueID)
		p.RelatedIssueIds, p.RelationTypes = append(p.RelatedIssueIds, r.RelatedID), append(p.RelationTypes, string(r.Type))
	}
	if err := s.queries(ctx).InsertRelations(ctx, p); err != nil {
		return fmt.Errorf("insert relations %v: %w", p.Ids, err)
	}
	return nil
}

// MarkChanged writes the work items r names last changed at its moment and
// by its account (M4 design 3.7). The use case read each of them undeleted
// under its project's lock, so one not written is an error.
func (s *Store) MarkChanged(ctx context.Context, r app.ChangedRows) error {
	n, err := s.queries(ctx).MarkChanged(ctx, gen.MarkChangedParams{By: r.By, At: r.At, Ids: r.IDs})
	switch {
	case err != nil:
		return fmt.Errorf("mark work items %v changed: %w", r.IDs, err)
	case n != int64(len(r.IDs)):
		return fmt.Errorf("marked %d of work items %v changed", n, r.IDs)
	}
	return nil
}
````

`server/internal/modules/issue/adapter/postgres/relations_test.go`（新文件，197 行）：

````file server/internal/modules/issue/adapter/postgres/relations_test.go
package postgresadapter_test

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// Which work items relate to a work item (M4 design 3.10): those with an
// undeleted relation to it, of any type, either way, of any project. Web's
// item relates to b, a relates to it, and so does ops' o; its relation to
// c was deleted; d relates to b alone. Of a, b, c, d and o, asked for in
// no order, a, b and o relate to it, in id order; none of a work item
// without relations; a cancelled read is its failure.
func TestRelated(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, ops := newProject(t, pool, acme, "WEB"), newProject(t, pool, acme, "OPS")
	item, a, b, c, d, o := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	for n, id := range []uuid.UUID{item, a, b, c, d} {
		newIssue(t, pool, web, id, n+1, alice)
	}
	newIssue(t, pool, ops, o, 1, alice)
	gone := uuid.NewV7()
	for _, r := range []relationRow{{uuid.NewV7(), web, item, b}, {uuid.NewV7(), web, a, item}, {uuid.NewV7(), ops, o, item},
		{gone, web, item, c}, {uuid.NewV7(), web, d, b}} {
		newRelation(t, pool, r.p, r.issue, r.related, r.id, alice)
	}
	exec(t, pool, "UPDATE issue_relations SET deleted_at = $2 WHERE id = $1", gone, now)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	got, err := s.Related(ctx, item, []uuid.UUID{o, d, c, b, a})
	if want := slices.SortedFunc(slices.Values([]uuid.UUID{a, b, o}), uuid.UUID.Compare); err != nil || !slices.Equal(got, want) {
		t.Errorf("Related(item) = %v, %v; want %v", got, err, want)
	}
	if got, err := s.Related(ctx, c, []uuid.UUID{item, a}); err != nil || len(got) != 0 {
		t.Errorf("Related(c) = %v, %v; want none", got, err)
	}
	cancelled, stop := context.WithCancel(context.Background())
	stop()
	if _, err := s.Related(cancelled, item, []uuid.UUID{a}); !errors.Is(err, context.Canceled) {
		t.Errorf("Related() on a cancelled context = %v; want context.Canceled", err)
	}
}

// storedRelations are the relations' rows, by id, each as text: its
// workspace, project, ends and type; who made it and when; who wrote it
// last and when; and whether it is deleted.
func storedRelations(t *testing.T, pool *pgxpool.Pool) map[uuid.UUID]string {
	t.Helper()
	rows, err := pool.Query(context.Background(), `SELECT id, workspace_id, project_id, issue_id, related_issue_id, relation_type, created_by_id,
		created_at, updated_by_id, updated_at, deleted_at IS NOT NULL FROM issue_relations`)
	if err != nil {
		t.Fatal(err)
	}
	out := map[uuid.UUID]string{}
	for rows.Next() {
		var id, workspace, project, issue, related uuid.UUID
		var kind string
		var made, written *uuid.UUID
		var madeAt, writtenAt time.Time
		var deleted bool
		if err := rows.Scan(&id, &workspace, &project, &issue, &related, &kind, &made, &madeAt, &written, &writtenAt, &deleted); err != nil {
			t.Fatal(err)
		}
		out[id] = relationText(workspace, project, issue, related, kind, made, madeAt, written, writtenAt, deleted)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// relationText is a relation's row as storedRelations writes it.
func relationText(workspace, project, issue, related uuid.UUID, kind string, made *uuid.UUID, madeAt time.Time, written *uuid.UUID,
	writtenAt time.Time, deleted bool) string {
	return fmt.Sprintf("%s %s %s %s %s, made by %v at %s, written by %v at %s, deleted %v", workspace, project, issue, related, kind, *made,
		madeAt.UTC().Format(time.RFC3339Nano), *written, writtenAt.UTC().Format(time.RFC3339Nano), deleted)
}

// Relations are inserted as given (M4 design 3.10, 4.6): ops' o blocked
// by web's item, a row of ops, and item relating to web's b, a row of
// web, by alice at the moment, in their order; every column of each row is
// as given. Relations of more than one workspace, moment or account are
// refused and none is written, each beside alone, c's relation to item,
// which is written by itself after them; so is one of a pair that relates
// already, of any type, either way (issue_relations_pair_key), and a
// cancelled insert.
func TestInsertRelations(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	web, ops := newProject(t, pool, acme, "WEB"), newProject(t, pool, acme, "OPS")
	item, b, c, o := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	newIssue(t, pool, web, item, 1, alice)
	newIssue(t, pool, web, b, 2, alice)
	newIssue(t, pool, web, c, 3, alice)
	newIssue(t, pool, ops, o, 1, alice)
	at := now.Add(time.Hour)
	rs := []domain.Relation{
		{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: ops.id, IssueID: o, RelatedID: item, Type: domain.BlockedBy, CreatedByID: &alice, CreatedAt: at},
		{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web.id, IssueID: item, RelatedID: b, Type: domain.RelatesTo, CreatedByID: &alice, CreatedAt: at},
	}
	if err := s.InsertRelations(context.Background(), rs); err != nil {
		t.Fatal(err)
	}
	want := map[uuid.UUID]string{}
	for _, r := range rs {
		want[r.ID] = relationText(r.WorkspaceID, r.ProjectID, r.IssueID, r.RelatedID, string(r.Type), &alice, at, &alice, at, false)
	}
	if got := storedRelations(t, pool); !maps.Equal(got, want) {
		t.Errorf("the relations:\n%v\nwant\n%v", got, want)
	}

	later := rs[1]
	later.ID, later.IssueID, later.RelatedID, later.CreatedAt = uuid.NewV7(), b, o, at.Add(time.Second)
	byBob := later
	byBob.CreatedAt, byBob.CreatedByID = at, &bob
	elsewhere := byBob
	elsewhere.CreatedByID, elsewhere.WorkspaceID = &alice, beta
	again := domain.Relation{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web.id, IssueID: item, RelatedID: o, Type: domain.Duplicate,
		CreatedByID: &alice, CreatedAt: at}
	alone := domain.Relation{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web.id, IssueID: c, RelatedID: item, Type: domain.RelatesTo,
		CreatedByID: &alice, CreatedAt: at}
	for name, rs := range map[string][]domain.Relation{"two workspaces": {alone, elsewhere}, "two moments": {alone, later},
		"two accounts": {alone, byBob}, "a pair related": {again}} {
		before := storedRelations(t, pool)
		if err := s.InsertRelations(context.Background(), rs); err == nil {
			t.Errorf("%s: InsertRelations() = nil, want an error", name)
		}
		if after := storedRelations(t, pool); !maps.Equal(after, before) {
			t.Errorf("%s: the relations\n%v\nwant them as they were\n%v", name, after, before)
		}
	}
	if err := s.InsertRelations(context.Background(), []domain.Relation{alone}); err != nil {
		t.Errorf("InsertRelations() of alone by itself = %v, want nil", err)
	}
	cancelled, stop := context.WithCancel(context.Background())
	stop()
	if err := s.InsertRelations(cancelled, []domain.Relation{again}); !errors.Is(err, context.Canceled) {
		t.Errorf("InsertRelations() on a cancelled context = %v; want context.Canceled", err)
	}
}

// Work items are marked changed at the moment and by the account given
// (M4 design 3.7), every other column of theirs as it was: web's a and
// ops' o, by bob at later; b, not named, is as it was. A deleted work item
// among those named is not written, and is an error; so is a cancelled
// write.
func TestMarkChanged(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, ops := newProject(t, pool, acme, "WEB"), newProject(t, pool, acme, "OPS")
	a, b, o, gone := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	for n, id := range []uuid.UUID{a, b, gone} {
		newIssue(t, pool, web, id, n+1, alice)
	}
	newIssue(t, pool, ops, o, 1, alice)
	exec(t, pool, "UPDATE issues SET deleted_at = $2 WHERE id = $1", gone, now)
	rows, stamps := rowsAsText(t, pool, "issues"), audits(t, pool, "issues")
	later := now.Add(time.Hour)

	if err := s.MarkChanged(context.Background(), app.ChangedRows{IDs: []uuid.UUID{a, o}, By: bob, At: later}); err != nil {
		t.Fatal(err)
	}

	if after := rowsAsText(t, pool, "issues"); !maps.Equal(after, rows) {
		t.Errorf("the work items but their audit columns:\n%v\nwant them as they were:\n%v", after, rows)
	}
	for _, id := range []uuid.UUID{a, o} {
		stamps[id] = fmt.Sprintf("undeleted, written at %s by %s", later.UTC().Format(time.RFC3339Nano), bob)
	}
	if after := audits(t, pool, "issues"); !maps.Equal(after, stamps) {
		t.Errorf("the audit columns:\n%v\nwant\n%v", after, stamps)
	}
	if err := s.MarkChanged(context.Background(), app.ChangedRows{IDs: []uuid.UUID{b, gone}, By: bob, At: later}); err == nil {
		t.Error("MarkChanged() of a deleted work item = nil, want an error")
	}
	cancelled, stop := context.WithCancel(context.Background())
	stop()
	if err := s.MarkChanged(cancelled, app.ChangedRows{IDs: []uuid.UUID{b}, By: bob, At: later}); !errors.Is(err, context.Canceled) {
		t.Errorf("MarkChanged() on a cancelled context = %v; want context.Canceled", err)
	}
}
````

`server/internal/modules/issue/app/rows.go`（修改，1 处）：

````old server/internal/modules/issue/app/rows.go
}

// IssueArchival is a work item's archiving or unarchiving as it is written
````
````new server/internal/modules/issue/app/rows.go
}

// ChangedRows are the work items a write changes last without a field of
// theirs (M4 design 3.7): the ends of a relation it adds or removes, IDs,
// in id order, each last changed at the moment and by the account given.
type ChangedRows struct {
	IDs []uuid.UUID
	By  uuid.UUID
	At  time.Time
}

// IssueArchival is a work item's archiving or unarchiving as it is written
````

- [ ] **Step 4: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/issue/...`
Expected: 全部 `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 5: 提交**

```bash
git add server/internal/modules/issue/adapter/postgres/queries/relations.sql server/internal/modules/issue/adapter/postgres/relations.go server/internal/modules/issue/adapter/postgres/relations_test.go server/internal/modules/issue/app/rows.go server/internal/modules/issue/domain/relation.go server/internal/modules/issue/domain/relation_test.go server/internal/modules/issue/adapter/postgres/gen/relations.sql.go
```
```bash
git commit -m "feat(M4/P3): relations between work items: the domain and the store

A relation is one row a pair of work items: relates_to, duplicate and
blocked_by from the work item asked from, blocking stored as the other's
blocked_by; each end reads the type as asked from it. An addition names
one of the four types and one to twenty work items, each once. The store
reads which work items relate to one already, either way, inserts one
write's relations in one statement, of one workspace, account and
moment, and marks both ends changed.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p3.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试，或没有替身的平台、领域、转换的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；守卫是生成器（`make gen-go`）和中英文翻译键（`make lint-web`）的核对。每个变异在一份新的副本里，在本 Task 的树上、标出的较晚的 Task 的树上和最终的树上都跑过，端到端的检查只在标出的 Task 的树上）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T9.1` | `blocked_by` 和 `blocking` 从两端读来一样 | `internal/modules/issue/domain` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWorkItemsAreRelatedAcrossProjects`（`make test`）（Task 11 起）、故事 `i9-relations`（Task 15 起） | 单元；组合；端到端 |
| `T9.2` | 要求 `blocking` 的关联照样存成 `blocking` | `internal/modules/issue/domain` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWorkItemsAreRelatedAcrossProjects`（`make test`）（Task 11 起）、故事 `i9-relations`（Task 15 起） | 单元；组合；端到端 |
| `T9.3` | 反过来存的 `blocking` 关联留着发起的工作项的项目 | `internal/modules/issue/domain` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWorkItemsAreRelatedAcrossProjects`（`make test`）（Task 11 起）、故事 `i9-relations`（Task 15 起） | 单元；组合；端到端 |
| `T9.4` | 加关联可以给任何类型 | `internal/modules/issue/domain` 的全部测试（`make test`） | 单元 |
| `T9.5` | 加关联可以列二十一个工作项 | `internal/modules/issue/domain` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWhatAWorkItemMayNotBeRelatedTo`（`make test`）（Task 11 起） | 单元；组合 |
| `T9.6` | 加关联可以一个也不列，或者把一个工作项列两次 | `internal/modules/issue/domain` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWhatAWorkItemMayNotBeRelatedTo`（`make test`）（Task 11 起） | 单元；组合 |
| `T9.7` | 加关联的那行动态把对方写成原值 | `internal/modules/issue/domain` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWorkItemsAreRelatedAcrossProjects`（`make test`）（Task 11 起）、故事 `i9-relations`（Task 15 起） | 单元；组合；端到端 |
| `T9.8` | 只按一个方向读这个工作项的关联 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWorkItemsAreRelatedAcrossProjects`（`make test`）（Task 11 起） | 存储；组合 |
| `T9.9` | `InsertRelations` 把每条关联反过来存 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWorkItemsAreRelatedAcrossProjects`（`make test`）（Task 11 起）、故事 `i9-relations`（Task 15 起） | 存储；组合；端到端 |
| `T9.10` | `MarkChanged` 也写已删除的工作项 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`） | 存储 |
| `T9.11` | `MarkChanged` 把没写到的工作项当作写了 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`） | 存储 |
| `T9.12` | `InsertRelations` 在一条语句里收两个工作区的关联 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`） | 存储 |
| `T9.13` | `InsertRelations` 在一条语句里收两个时刻的关联 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`） | 存储 |
| `T9.14` | `InsertRelations` 在一条语句里收两个账户的关联 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`） | 存储 |
| `T9.15` | `InsertRelations` 经连接池写，不在它的事务的连接上 | `internal/bootstrap` 的 `TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection`（`make test`）（Task 11 起） | 组合 |

---

### Task 10: 加关联：用例和跨项目的取锁

**Files:**
- Create: `server/internal/modules/issue/app/create_relations.go`、`server/internal/modules/issue/app/create_relations_test.go`、`server/internal/modules/issue/app/fakes_relations_test.go`、`server/internal/modules/issue/app/lock_across.go`
- Modify: `server/internal/modules/issue/app/lock.go`、`server/internal/modules/issue/app/lock_test.go`、`server/internal/modules/issue/app/parents.go`、`server/internal/modules/issue/app/ports.go`、`server/internal/modules/issue/app/sub_issues.go`

**Interfaces:**（spec 2.12、第 3 节第 2、3、7、9、11、12 条；M4 设计 3.4、3.5 约定七、3.7、3.8、3.10、3.13、11.11）`Locks.run` 的跨项目的形式和它的第一个使用者；一条取锁路径，不另写：
- `issue/app/lock.go`：`write.reach`（跨项目的写到达的别的工作项，每个以 `issue.read` 判定；判定拒绝的不是这个写的回答，而是"没到达"）；`held.reached`（锁下到达的，按 `write.reach` 的次序）；`precheck` 交回 `scope`（它放进锁的项目）；`run` 以 `scope` 的项目一次进写闸门（`WriteGate.Enter(ctx, s.projects...)`）；`lockAndDecide(ctx, actor, w, s)` 经 `lockProjects` 锁住，跨项目时锁下再到达（`reachUnder`）。
- `issue/app/lock_across.go`（新）：`scope{projects, reached}`（项目各一次、`id` 升序，写自己的在其中）；`end{issue, project, grant, ok}`；`lockProjects`（一个项目的写 `ProjectGate.Lock`；跨项目的写 `LockMany`，即使只有一个项目）；`reachAll`（预检：每个读、判定，没到达的不锁、不进闸门）；`reachUnder`（锁下：预检到达的再读、再判定；项目等锁期间被删除、不在锁住的里的，不算到达）；`reach`（工作项在、未删除、不是它自己、同一个工作区、调用者以 `issue.read` 看得到（访客规则在先）：否则 `ok` 为假，判定的拒绝在这里不是错误）。
- `issue/app/parents.go`：`refOf(p LockedProject, i)`（按锁住的那个项目的标识：别的项目的工作项也按它自己的项目写）；`parents.go`、`sub_issues.go` 的调用照改。
- `issue/app/ports.go`：`RelationCreator`（`Related`、`InsertRelations`、`MarkChanged`、`InsertActivities`，嵌入 `IssueReader`，`GetIssues` 在它里面）。
- `issue/app/create_relations.go`（新）：`IssueRelation{ID, Type, Issue, CreatedAt}`（关联作为它的一个工作项读到的）；`CreateIssueRelationsDeps{Locks, Relations, Clock}`、`NewCreateIssueRelations`、`Execute(ctx, id, t, ids)`：`CheckNewRelations` 在任何读之前；`Locks.run` 的 `write{issue: &id, action: issue.update, reach: ids}`；锁下项目已归档 409 → 工作项已归档 409 `issue.archived` → `check`（每个都到达、没归档、它的项目没归档、与它还没有关联（`Related`，在两端的项目的锁下读）；问题按它在 `issue_ids` 里的位置一次报出：没到达的、已归档的 `not_allowed`，已有关联的 `duplicate`）→ 时刻 → `write`（按对方的 `id` 的次序：`NewRelation`；`InsertRelations` 一条语句；`MarkChanged` 两端，`id` 的次序；每条关联两行动态，各在自己的项目：它的工作项读到的类型、对方的标识）→ `answer`（对方在事务里读回，少了是错误；按调用者在各项目的 `Grant` 数子工作项：`grants` 带上每个对方的项目）。
- 测试的帮手：`fakes_relations_test.go`（新）：`fakeRelations`（嵌入 `*fakeIssues`：`Related`、`InsertRelations`、`MarkChanged` 的记录）；`create_relations_test.go`（新）：`nearID`、`farID`、`newRelate`、`relateOver`、`reaching`、`relateCase`。

**Tests:**
- 单元：`TestCreateIssueRelationsRelatesTheWorkItemToOthers`（3.5、3.10 的次序：跨项目的 `Locks` 的调用（`reaching`）；已有的关联；时刻；关联；两端按 `id` 的次序改最后修改；动态；读回、数过。alice 以 `blocking` 问 ops 的 far、web 的 near：各存为对方的 `blocked_by`，对方的行，near 的在先；每条两行动态，各在自己的项目，按标识；回答按工作项读的 `blocking`，far 按 alice 在 ops 的 `Grant`（她是只看自己的访客）数，near 按 web 的）、`TestCreateIssueRelationsRefusals`（锁之前：`CheckNewRelations` 拒绝的、不在的、看不到项目的 404、不能改的 403，在读别的工作项之前；锁下：项目、工作项等锁期间被删除 404，不能再改 403，已归档的项目 409（工作项也已归档），然后已归档的工作项 409，都在检查别的工作项之前；都不写）、`TestCreateIssueRelationsRefusesWhatItMayNotRelateTo`（锁之前没到达的：不在的、它自己、别的工作区的、看不到项目的、不能读的、访客规则隐藏的；锁下：已归档的、在已归档的项目里的、等锁期间被删除的、项目等锁期间被删除的、不再看得到的；已有关联的 `duplicate`；两个都被拒绝时各在自己的位置）、`TestCreateIssueRelationsStopsAtAFailingPort`、`TestCreateIssueRelationsErrsOnWhatCannotBe`（读回少了是错误，不是拒绝）；`lock_test.go` 的 `TestAWriteAcrossProjectsTakesOnlyTheProjectsItReaches`（预检之后、事务之外一次进 web、ops 的闸门（`id` 升序），一次锁住同样的项目、同样的次序（`LockMany`），事务结束之后最后离开；预检没到达的（看不到的 far、没有的 far）不取锁、不进闸门，422 只在 web 的锁下答）。

- [ ] **Step 1: 跨项目的取锁**

`server/internal/modules/issue/app/lock.go`（修改，17 处）：

````old server/internal/modules/issue/app/lock.go
// then on; the action it decides; and the 404 of what the caller named.
````
````new server/internal/modules/issue/app/lock.go
// then on; the action it decides; and the 404 of what the caller named. A
// write across projects (convention 7, M4 design 3.5) reaches other work
// items besides issue, each of its project, issue's or another of the
// workspace: reach, each decided on issue.read, a refusal of which leaves
// the work item unreached rather than answering the write
// (createIssueRelations' other ends).
````
````old server/internal/modules/issue/app/lock.go
	action   shared.Action
	notFound error
````
````new server/internal/modules/issue/app/lock.go
	action   shared.Action
	notFound error
	reach    []uuid.UUID
````
````old server/internal/modules/issue/app/lock.go
// and the caller's grant on the project, as decided under the lock, whose
````
````new server/internal/modules/issue/app/lock.go
// the caller's grant on the project, as decided under the lock, whose
````
````old server/internal/modules/issue/app/lock.go
// (M4 design 3.4).
````
````new server/internal/modules/issue/app/lock.go
// (M4 design 3.4); and, for a write across projects, the work items of
// write.reach, in its order, as reached under the lock (reachUnder).
````
````old server/internal/modules/issue/app/lock.go
	grant   shared.Grant
````
````new server/internal/modules/issue/app/lock.go
	grant   shared.Grant
	reached []end
````
````old server/internal/modules/issue/app/lock.go
// then the write gate of w's project; then in one transaction w's lock and
// decision, and runs fn in it with what they hold (M4 design 3.5, 11.11):
// w's whole write. It leaves the gate once the transaction has ended, on
// every path, a panic's too. A refusal of the precheck, the gate or the
// decision is its answer, and fn does not run.
````
````new server/internal/modules/issue/app/lock.go
// then the write gates of its projects, w's and, across projects, those of
// the work items it reaches, all at once in ascending id order; then in
// one transaction w's lock and decision, and runs fn in it with what they
// hold (M4 design 3.5, 11.11): w's whole write. It leaves the gates once
// the transaction has ended, on every path, a panic's too. A refusal of the
// precheck, the gate or the decision is its answer, and fn does not run.
````
````old server/internal/modules/issue/app/lock.go
	w, err := l.precheck(ctx, actor, w)
````
````new server/internal/modules/issue/app/lock.go
	w, s, err := l.precheck(ctx, actor, w)
````
````old server/internal/modules/issue/app/lock.go
	leave, err := l.gate.Enter(ctx, w.project)
````
````new server/internal/modules/issue/app/lock.go
	leave, err := l.gate.Enter(ctx, s.projects...)
````
````old server/internal/modules/issue/app/lock.go
		h, err := l.lockAndDecide(ctx, actor, w)
````
````new server/internal/modules/issue/app/lock.go
		h, err := l.lockAndDecide(ctx, actor, w, s)
````
````old server/internal/modules/issue/app/lock.go
// transaction (M4 design 3.5), and answers w on its project. For a write
// that names a work item: the work item read without a lock and the
// decision on it (see); a work item that is not there, whose project actor
// does not see, or that the guest rule hides from him, is
// issue.not_found. For one on a project: the project read without a lock
// and the decision on it; a project that is not there, or not visible to
// actor, is w.notFound. A role the rule does not allow is the Authorizer's
// shared.Forbidden. What it lets through, lockAndDecide decides again.
func (l Locks) precheck(ctx context.Context, actor shared.Actor, w write) (write, error) {
````
````new server/internal/modules/issue/app/lock.go
// transaction (M4 design 3.5), and answers w on its project, with what it
// lets through to the lock (scope). For a write that names a work item:
// the work item read without a lock and the decision on it (see); a work
// item that is not there, whose project actor does not see, or that the
// guest rule hides from him, is issue.not_found; across projects, then,
// each work item it reaches (reachAll). For one on a project: the project
// read without a lock and the decision on it; a project that is not there,
// or not visible to actor, is w.notFound. A role the rule does not allow
// is the Authorizer's shared.Forbidden. What it lets through,
// lockAndDecide decides again.
func (l Locks) precheck(ctx context.Context, actor shared.Actor, w write) (write, scope, error) {
````
````old server/internal/modules/issue/app/lock.go
			return write{}, err
````
````new server/internal/modules/issue/app/lock.go
			return write{}, scope{}, err
````
````old server/internal/modules/issue/app/lock.go
		return w, nil
````
````new server/internal/modules/issue/app/lock.go
		if w.reach == nil {
			return w, scope{projects: []uuid.UUID{w.project}}, nil
		}
		s, err := l.reachAll(ctx, actor, i, w.reach)
		return w, s, err
````
````old server/internal/modules/issue/app/lock.go
	case err != nil:
		return write{}, err
````
````new server/internal/modules/issue/app/lock.go
	case err != nil:
		return write{}, scope{}, err
````
````old server/internal/modules/issue/app/lock.go
		return write{}, w.notFound
````
````new server/internal/modules/issue/app/lock.go
		return write{}, scope{}, w.notFound
````
````old server/internal/modules/issue/app/lock.go
	return w, err
````
````new server/internal/modules/issue/app/lock.go
	return w, scope{projects: []uuid.UUID{w.project}}, err
````
````old server/internal/modules/issue/app/lock.go
// transaction ctx carries: the project's row FOR NO KEY UPDATE; the work
// item w names read again; then the decision on the locked project, and on
// the work item as read again. It answers what they hold, the decision's
// grant too. A project deleted while the lock waited is w.notFound, and so
// is a work item deleted meanwhile, or no longer in the project, a project
// no longer visible to actor, and a work item the guest rule now hides from
// him; a role the rule no longer allows is the Authorizer's
// shared.Forbidden.
func (l Locks) lockAndDecide(ctx context.Context, actor shared.Actor, w write) (held, error) {
	p, found, err := l.projects.Lock(ctx, w.project)
	switch {
	case err != nil:
		return held{}, err
	case !found:
````
````new server/internal/modules/issue/app/lock.go
// transaction ctx carries: the project's row FOR NO KEY UPDATE, or across
// projects s's rows (lockProjects); the work item w names read again; then
// the decision on the locked project, and on the work item as read again;
// across projects, then, each work item it reaches (reachUnder). It
// answers what they hold, the decision's grant too. A project deleted
// while the lock waited is w.notFound, and so is a work item deleted
// meanwhile, or no longer in the project, a project no longer visible to
// actor, and a work item the guest rule now hides from him; a role the
// rule no longer allows is the Authorizer's shared.Forbidden.
func (l Locks) lockAndDecide(ctx context.Context, actor shared.Actor, w write, s scope) (held, error) {
	locked, err := l.lockProjects(ctx, w, s)
	if err != nil {
		return held{}, err
	}
	p, found := locked[w.project]
	if !found {
````
````old server/internal/modules/issue/app/lock.go
	if h.grant, err = l.decide(ctx, actor, w.action, p.WorkspaceID, w.project, on, w.notFound); err != nil {
		return held{}, err
````
````new server/internal/modules/issue/app/lock.go
	if h.grant, err = l.decide(ctx, actor, w.action, p.WorkspaceID, w.project, on, w.notFound); err != nil {
		return held{}, err
	}
	if w.reach != nil {
		if h.reached, err = l.reachUnder(ctx, actor, h.issue, w.reach, s, locked); err != nil {
			return held{}, err
		}
````

`server/internal/modules/issue/app/lock_across.go`（新文件，127 行）：

````file server/internal/modules/issue/app/lock_across.go
package app

import (
	"context"
	"errors"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The form of Locks.run for a write across projects (convention 7, M4
// design 3.5): the work items it reaches besides its own, each read and
// decided on without a lock first, so that one the caller may not reach
// takes no lock and no place in a gate, its project's above all; then the
// projects of the others, with the write's, locked all at once in
// ascending id order, and each work item read and decided on again under
// them.

// scope is what a write's precheck lets through to its lock: the projects
// it locks, in ascending id order, each once, the write's among them;
// and, across projects, which of the work items it reaches the precheck
// reached, in write.reach's order.
type scope struct {
	projects []uuid.UUID
	reached  []bool
}

// end is a work item a write across projects reaches, as its lock holds
// it: the work item and its project, read under the lock, and the caller's
// grant on the project, as decided under it; ok is false for a work item
// the write does not reach, and the rest is zero then.
type end struct {
	issue   domain.Issue
	project LockedProject
	grant   shared.Grant
	ok      bool
}

// lockProjects locks s's projects in the transaction ctx carries and
// answers those it locked, by id: a write on one project its row FOR NO
// KEY UPDATE (ProjectGate.Lock); a write across projects each of s's, all
// at once in ascending id order (ProjectGate.LockMany), even when they are
// one. A project deleted while the lock waited is not among them.
func (l Locks) lockProjects(ctx context.Context, w write, s scope) (map[uuid.UUID]LockedProject, error) {
	if w.reach != nil {
		return l.projects.LockMany(ctx, s.projects)
	}
	p, found, err := l.projects.Lock(ctx, w.project)
	if err != nil || !found {
		return nil, err
	}
	return map[uuid.UUID]LockedProject{w.project: p}, nil
}

// reachAll is the precheck of the work items ids that a write from the
// work item from reaches, before any lock (M4 design 3.5): each read and
// decided on without one (reach). It answers the scope of the write:
// from's project and those of the work items it reached, in ascending id
// order, each once, and which it reached; one it did not reach takes no
// lock. A failing read or decision is its error.
func (l Locks) reachAll(ctx context.Context, actor shared.Actor, from domain.Issue, ids []uuid.UUID) (scope, error) {
	s := scope{projects: []uuid.UUID{from.ProjectID}, reached: make([]bool, len(ids))}
	for n, id := range ids {
		e, err := l.reach(ctx, actor, from, id)
		if err != nil {
			return scope{}, err
		}
		if e.ok {
			s.reached[n] = true
			s.projects = append(s.projects, e.issue.ProjectID)
		}
	}
	slices.SortFunc(s.projects, uuid.UUID.Compare)
	s.projects = slices.Compact(s.projects)
	return s, nil
}

// reachUnder reads again, under the lock, each work item of ids that the
// precheck reached (s.reached), and decides on it again (reach): its ends,
// in ids' order. One the precheck did not reach is not reached, and
// neither is one whose project is not among locked, deleted while the lock
// waited. A failing read or decision is its error.
func (l Locks) reachUnder(ctx context.Context, actor shared.Actor, from domain.Issue, ids []uuid.UUID, s scope,
	locked map[uuid.UUID]LockedProject) ([]end, error) {
	ends := make([]end, len(ids))
	for n, id := range ids {
		if !s.reached[n] {
			continue
		}
		e, err := l.reach(ctx, actor, from, id)
		if err != nil {
			return nil, err
		}
		if p, found := locked[e.issue.ProjectID]; e.ok && found {
			e.project = p
			ends[n] = e
		}
	}
	return ends, nil
}

// reach reads the work item id and decides whether a write from the work
// item from reaches it (M4 design 3.4, 3.10): the work item is there,
// undeleted, it is not from, it is of from's workspace, and actor sees it,
// on issue.read on its project, the guest rule first. ok is false for one
// it does not reach: a refusal of the decision is no error here. A failing
// read or decision is its error.
func (l Locks) reach(ctx context.Context, actor shared.Actor, from domain.Issue, id uuid.UUID) (end, error) {
	i, found, err := l.issues.GetIssue(ctx, id)
	switch {
	case err != nil:
		return end{}, err
	case !found || i.ID == from.ID || i.WorkspaceID != from.WorkspaceID:
		return end{}, nil
	}
	grant, err := l.decide(ctx, actor, domain.ActionRead, i.WorkspaceID, i.ProjectID, &i, domain.ErrIssueNotFound)
	var refused *shared.Error
	switch {
	case errors.As(err, &refused):
		return end{}, nil
	case err != nil:
		return end{}, err
	}
	return end{issue: i, grant: grant, ok: true}, nil
}
````

`server/internal/modules/issue/app/parents.go`（修改，3 处）：

````old server/internal/modules/issue/app/parents.go
		c.New = refOf(h, parent)
````
````new server/internal/modules/issue/app/parents.go
		c.New = refOf(h.project, parent)
````
````old server/internal/modules/issue/app/parents.go
		c.Old = refOf(h, old)
````
````new server/internal/modules/issue/app/parents.go
		c.Old = refOf(h.project, old)
````
````old server/internal/modules/issue/app/parents.go
// refOf is the work item i, of the project h holds locked, as an activity
// names it: by its identifier under the lock (M4 design 3.13).
func refOf(h held, i domain.Issue) *domain.Ref {
	return &domain.Ref{ID: i.ID, Identifier: domain.Identifier(h.project.Identifier, i.SequenceID)}
````
````new server/internal/modules/issue/app/parents.go
// refOf is the work item i, of the project p, locked, as an activity names
// it: by its identifier under the lock (M4 design 3.13).
func refOf(p LockedProject, i domain.Issue) *domain.Ref {
	return &domain.Ref{ID: i.ID, Identifier: domain.Identifier(p.Identifier, i.SequenceID)}
````

`server/internal/modules/issue/app/ports.go`（修改，1 处）：

````old server/internal/modules/issue/app/ports.go
}

// IssueArchiver is the work items' store as archiveIssue and
````
````new server/internal/modules/issue/app/ports.go
}

// RelationCreator is the work items' store as createIssueRelations writes
// and reads them, in the transaction ctx carries (M4 design 3.5, 3.10).
type RelationCreator interface {
	IssueReader
	// Related reads which of the work items ids relate to the work item
	// id: an undeleted relation between them, of any type, either way, in
	// id order.
	Related(ctx context.Context, id uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error)
	// InsertRelations inserts the relations in their order.
	InsertRelations(ctx context.Context, rs []domain.Relation) error
	// MarkChanged writes the undeleted work items r names last changed.
	MarkChanged(ctx context.Context, r ChangedRows) error
	// InsertActivities inserts the rows in their order.
	InsertActivities(ctx context.Context, rows []ActivityRow) error
}

// IssueArchiver is the work items' store as archiveIssue and
````

`server/internal/modules/issue/app/sub_issues.go`（修改，2 处）：

````old server/internal/modules/issue/app/sub_issues.go
		out[n] = move{item: i, parent: domain.Changed[*domain.Ref]{New: refOf(h, h.issue)}}
````
````new server/internal/modules/issue/app/sub_issues.go
		out[n] = move{item: i, parent: domain.Changed[*domain.Ref]{New: refOf(h.project, h.issue)}}
````
````old server/internal/modules/issue/app/sub_issues.go
		out[n].parent.Old = refOf(h, parents[at])
````
````new server/internal/modules/issue/app/sub_issues.go
		out[n].parent.Old = refOf(h.project, parents[at])
````

- [ ] **Step 2: 用例和它的测试**

`server/internal/modules/issue/app/create_relations.go`（新文件，202 行）：

````file server/internal/modules/issue/app/create_relations.go
package app

import (
	"context"
	"fmt"
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// IssueRelation is a relation as one of its work items reads it (M4 design
// 3.10, 5.2): its id, its type from the work item, the other work item as
// the caller sees it, and when it was made.
type IssueRelation struct {
	ID        uuid.UUID
	Type      domain.RelationType
	Issue     domain.Issue
	CreatedAt time.Time
}

// CreateIssueRelationsDeps are what createIssueRelations reads and writes
// through.
type CreateIssueRelationsDeps struct {
	Locks     Locks
	Relations RelationCreator
	Clock     Clock
}

// CreateIssueRelations relates a work item to others: POST
// /api/v0/issues/{issue_id}/relations (M4 design 3.5, 3.10, 5.1).
type CreateIssueRelations struct {
	d CreateIssueRelationsDeps
}

// NewCreateIssueRelations returns the use case.
func NewCreateIssueRelations(d CreateIssueRelationsDeps) *CreateIssueRelations {
	return &CreateIssueRelations{d: d}
}

// Execute checks t and ids (domain.CheckNewRelations), before any read;
// then Locks runs the write on the work item id across projects (M4 design
// 3.5, 11.11, convention 7): its precheck, the decision on issue.update on
// it, the guest rule first, then each of ids read and decided on, on
// issue.read, the guest rule first, before any lock, so that one the
// caller may not reach takes no lock; then the write gates of the work
// item's project and of the others' it reaches, all at once in ascending
// id order, which refuse a write that may wait no longer (503
// server_busy); then in one transaction those projects FOR NO KEY UPDATE,
// in ascending id order, the work item and each other read and decided on
// again, and under them, in this order: an archived project refuses it
// (409 project.archived), then an archived work item (409 issue.archived);
// each of ids reached, not archived, of a project not archived, and not
// related to the work item yet (check), every problem in one 422; then the
// clock; the relations, in the others' id order, each as stored
// (domain.NewRelation); the work item and each other last changed by the
// caller; for each relation a row of the work item's activity and one of
// the other's, each of its own project, the type as its work item reads
// it, the other work item by its identifier (rows 20 and 21 of 3.13). All
// of it is the caller's at that moment. The answer is the relations as the
// work item reads them, in the others' id order, each other read back in
// the transaction, as the caller sees it (Locks.seen).
func (u *CreateIssueRelations) Execute(ctx context.Context, id uuid.UUID, t domain.RelationType, ids []uuid.UUID) ([]IssueRelation, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	if err := domain.CheckNewRelations(t, ids); err != nil {
		return nil, err
	}
	w := write{issue: &id, action: domain.ActionUpdate, notFound: domain.ErrIssueNotFound, reach: ids}
	var created []IssueRelation
	err = u.d.Locks.run(ctx, actor, w, func(ctx context.Context, h held) error {
		switch {
		case h.project.Archived:
			return domain.ErrProjectArchived
		case h.issue.ArchivedAt != nil:
			return domain.ErrIssueArchived
		}
		others, err := u.check(ctx, h)
		if err != nil {
			return err
		}
		relations, err := u.write(ctx, h, actor.UserID, t, others)
		if err != nil {
			return err
		}
		created, err = u.answer(ctx, actor.UserID, h, relations, others)
		return err
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

// check holds the work items the caller's addition relates the work item h
// holds to, as h reached them under the locks (M4 design 3.4, 3.8, 3.10):
// each reached, not archived, of a project that is not archived, and not
// related to the work item yet, of any type, either way. Every problem is
// reported at once, in one 422, each at its place in the addition's work
// items. It answers them, in id order.
func (u *CreateIssueRelations) check(ctx context.Context, h held) ([]end, error) {
	var reached []uuid.UUID
	for _, e := range h.reached {
		if e.ok {
			reached = append(reached, e.issue.ID)
		}
	}
	var related []uuid.UUID
	if len(reached) > 0 {
		var err error
		if related, err = u.d.Relations.Related(ctx, h.issue.ID, slices.SortedFunc(slices.Values(reached), uuid.UUID.Compare)); err != nil {
			return nil, err
		}
	}
	var problems []shared.FieldError
	for n, e := range h.reached {
		switch {
		case !e.ok || e.issue.ArchivedAt != nil || e.project.Archived:
			problems = append(problems, domain.RelatedNotAllowed(n))
		case slices.Contains(related, e.issue.ID):
			problems = append(problems, domain.RelatedAlready(n))
		}
	}
	if len(problems) > 0 {
		return nil, shared.Invalid(problems...)
	}
	others := slices.Clone(h.reached)
	slices.SortFunc(others, func(a, b end) int { return a.issue.ID.Compare(b.issue.ID) })
	return others, nil
}

// write relates the work item h holds to others, in their order, the
// caller by's at the clock's moment (M4 design 3.7, 3.10, 3.13): the
// relations, as stored (domain.NewRelation); the work item and each other
// last changed by him, in id order; for each relation a row of the work
// item's activity and one of the other's, each of its own project, the
// type as its work item reads it, the other by its identifier. It answers
// the relations.
func (u *CreateIssueRelations) write(ctx context.Context, h held, by uuid.UUID, t domain.RelationType, others []end) ([]domain.Relation,
	error) {
	at := u.d.Clock.Now()
	from := end{issue: h.issue, project: h.project}
	relations := make([]domain.Relation, len(others))
	changed := []uuid.UUID{h.issue.ID}
	var rows []ActivityRow
	for n, o := range others {
		relations[n] = domain.NewRelation(uuid.NewV7(), h.issue, o.issue, t, by, at)
		changed = append(changed, o.issue.ID)
		rows = append(rows, from.stamp(by, at).activity([]domain.Activity{domain.RelationAdded(t, *refOf(o.project, o.issue))})...)
		rows = append(rows, o.stamp(by, at).activity([]domain.Activity{domain.RelationAdded(t.Reversed(), *refOf(h.project, h.issue))})...)
	}
	if err := u.d.Relations.InsertRelations(ctx, relations); err != nil {
		return nil, err
	}
	slices.SortFunc(changed, uuid.UUID.Compare)
	if err := u.d.Relations.MarkChanged(ctx, ChangedRows{IDs: changed, By: by, At: at}); err != nil {
		return nil, err
	}
	if err := u.d.Relations.InsertActivities(ctx, rows); err != nil {
		return nil, err
	}
	return relations, nil
}

// answer is the relations as the work item h holds reads them, each with
// its other end, of others, read back in the transaction as the caller
// actor sees it (Locks.seen), by his grants under the locks. One not there
// to read back is an error: the write found each undeleted under the
// locks.
func (u *CreateIssueRelations) answer(ctx context.Context, actor uuid.UUID, h held, relations []domain.Relation, others []end) (
	[]IssueRelation, error) {
	ids, g := make([]uuid.UUID, len(others)), grants{h.issue.ProjectID: h.grant}
	for n, o := range others {
		ids[n], g[o.issue.ProjectID] = o.issue.ID, o.grant
	}
	stored, err := u.d.Relations.GetIssues(ctx, ids)
	switch {
	case err != nil:
		return nil, err
	case len(stored) != len(ids):
		return nil, fmt.Errorf("work items %v are not all there after their write", ids)
	}
	seen, err := u.d.Locks.seen(ctx, actor, g, stored...)
	if err != nil {
		return nil, err
	}
	out := make([]IssueRelation, len(relations))
	for n, r := range relations {
		out[n] = IssueRelation{ID: r.ID, Type: r.TypeFrom(h.issue.ID), Issue: seen[n], CreatedAt: r.CreatedAt}
	}
	return out, nil
}

// stamp is where the rows of e's work item go, the caller by's at the
// moment at.
func (e end) stamp(by uuid.UUID, at time.Time) stamp {
	return stamp{workspace: e.issue.WorkspaceID, project: e.issue.ProjectID, issue: e.issue.ID, by: by, at: at}
}
````

`server/internal/modules/issue/app/create_relations_test.go`（新文件，288 行）：

````file server/internal/modules/issue/app/create_relations_test.go
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

// nearID and farID are the ids of newRelate's other work items, near's
// drawn first, so the smaller.
var nearID, farID = uuid.NewV7(), uuid.NewV7()

// relating is the other work items of newRelate's world: near, bob's work
// item 8 of web, and far, carol's work item 3 of ops, OPS (convention 7).
type relating struct {
	near, far domain.Issue
}

// newRelate is createIssueRelations over newUpdate's world, web's work
// item 7 its work item, with near and far in the store, and ops locked by
// LockMany too, as web is.
func newRelate() (*app.CreateIssueRelations, *world, *fakeRelations, uuid.UUID, relating) {
	_, w, id := newUpdate()
	r := relating{
		near: domain.Issue{ID: nearID, WorkspaceID: acme, ProjectID: web, SequenceID: 8, Name: "Near", StateID: backlog.ID, CreatedByID: &bob},
		far:  domain.Issue{ID: farID, WorkspaceID: acme, ProjectID: ops, SequenceID: 3, Name: "Far", CreatedByID: &carol},
	}
	w.issues.others = map[uuid.UUID]domain.Issue{r.near.ID: r.near, r.far.ID: r.far}
	w.project.across = map[uuid.UUID]app.LockedProject{ops: {WorkspaceID: acme, Identifier: "OPS"}}
	rel := &fakeRelations{fakeIssues: w.issues}
	return relateOver(w, rel), w, rel, id, r
}

// relateOver is createIssueRelations over w's locks and rel.
func relateOver(w *world, rel *fakeRelations) *app.CreateIssueRelations {
	return app.NewCreateIssueRelations(app.CreateIssueRelationsDeps{Locks: w.locks, Relations: rel, Clock: clockAt{at: clockNow, log: w.log}})
}

// reaching are the calls of alice's addition of relations from the work
// item id to far, then near, as Locks takes them across projects (M4
// design 3.5): the work item read and decided on, then far and near, each
// in its project, before any lock and outside the transaction; in it, web
// and ops locked at once, in id order, then the same reads and decisions
// again.
func reaching(id uuid.UUID, r relating) []string {
	var before, under []string
	for _, read := range []struct {
		item, project uuid.UUID
		action        shared.Action
	}{{id, web, domain.ActionUpdate}, {r.far.ID, ops, domain.ActionRead}, {r.near.ID, web, domain.ActionRead}} {
		before = append(before, line("GetIssue", read.item, "outside tx"), line("Authorize", alice, read.action, acme, read.project, "outside tx"))
		under = append(under, line("GetIssue", read.item), line("Authorize", alice, read.action, acme, read.project))
	}
	return slices.Concat(before, []string{"Begin", line("LockMany", anys(sorted([]uuid.UUID{web, ops}))...)}, under)
}

// An addition of relations, in the order of M4 design 3.5 and 3.10:
// Locks' calls across projects (reaching); whether the others relate to
// the work item already; the clock; the relations; the work item and the
// others marked changed, in id order; the activity; the others read back
// and counted. alice asks for far, of ops, then near, of web, as blocking:
// each is stored the other way round, the other's row, its blocked_by,
// near's first. Each writes a row of the work item's activity, blocking
// the other by its identifier, and one of the other's, of its project,
// blocked_by WEB-7, near's first. The answer is the relations as the work
// item reads them, blocking, each other as stored then, alice's last
// change, counted as she sees it in its project: every one of near's, in
// web, and only hers of far's, in ops, where she is a guest who sees her
// own work items alone (Grant.OwnOnly of far's decisions), far among
// them, made by her here.
func TestCreateIssueRelationsRelatesTheWorkItemToOthers(t *testing.T) {
	uc, w, rel, id, r := newRelate()
	r.far.CreatedByID = &alice
	w.issues.others[farID] = r.far
	w.auth.grants = []shared.Grant{1: {OwnOnly: true}, 4: {OwnOnly: true}}
	w.issues.counts = map[uuid.UUID]app.SubIssueCount{r.near.ID: {All: 2, Own: 1}, r.far.ID: {All: 4, Own: 0}}
	got, err := uc.Execute(as(alice), id, domain.Blocking, []uuid.UUID{r.far.ID, r.near.ID})
	if err != nil {
		t.Fatal(err)
	}
	others := []uuid.UUID{r.near.ID, r.far.ID}
	calls := slices.Concat(reaching(id, r), []string{line("Related", id, others), "Now", line("InsertRelations", 2),
		line("MarkChanged", sorted([]uuid.UUID{id, r.near.ID, r.far.ID})), line("InsertActivities", 4), line("GetIssues", others),
		counted(others...)})
	if !slices.Equal(w.log.calls, calls) {
		t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
	}
	if len(rel.relations) != 2 {
		t.Fatalf("the relations %+v, want two", rel.relations)
	}
	for n, o := range []domain.Issue{r.near, r.far} {
		want := domain.Relation{ID: rel.relations[n].ID, WorkspaceID: acme, ProjectID: o.ProjectID, IssueID: o.ID, RelatedID: id,
			Type: domain.BlockedBy, CreatedByID: &alice, CreatedAt: clockNow}
		if !reflect.DeepEqual(rel.relations[n], want) || n > 0 && rel.relations[n].ID.Compare(rel.relations[n-1].ID) <= 0 {
			t.Errorf("relation %d: %+v, want %+v, its id after the one before", n, rel.relations[n], want)
		}
	}
	marked := app.ChangedRows{IDs: sorted([]uuid.UUID{id, r.near.ID, r.far.ID}), By: alice, At: clockNow}
	if rel.marked == nil || !reflect.DeepEqual(*rel.marked, marked) {
		t.Errorf("marked %+v, want %+v", rel.marked, marked)
	}
	work := domain.Ref{ID: id, Identifier: "WEB-7"}
	rows := []app.ActivityRow{
		{IssueID: id, ProjectID: web, Activity: domain.RelationAdded(domain.Blocking, domain.Ref{ID: r.near.ID, Identifier: "WEB-8"})},
		{IssueID: r.near.ID, ProjectID: web, Activity: domain.RelationAdded(domain.BlockedBy, work)},
		{IssueID: id, ProjectID: web, Activity: domain.RelationAdded(domain.Blocking, domain.Ref{ID: r.far.ID, Identifier: "OPS-3"})},
		{IssueID: r.far.ID, ProjectID: ops, Activity: domain.RelationAdded(domain.BlockedBy, work)},
	}
	if len(w.issues.activities) != len(rows) {
		t.Fatalf("the activity %+v, want %d rows", w.issues.activities, len(rows))
	}
	for n, row := range w.issues.activities {
		want := rows[n]
		want.ID, want.WorkspaceID, want.ActorID, want.At = row.ID, acme, alice, clockNow
		if !reflect.DeepEqual(row, want) || n > 0 && row.ID.Compare(w.issues.activities[n-1].ID) <= 0 {
			t.Errorf("activity row %d: %+v, want %+v, its id after the row before", n, row, want)
		}
	}
	var want []app.IssueRelation
	for n, o := range []domain.Issue{r.near, r.far} {
		o.UpdatedByID, o.UpdatedAt, o.SubIssuesCount = &alice, clockNow, []int{2, 0}[n]
		want = append(want, app.IssueRelation{ID: rel.relations[n].ID, Type: domain.Blocking, Issue: o, CreatedAt: clockNow})
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Execute() = %+v, want %+v", got, want)
	}
}

// Every refusal before any lock and outside the transaction (M4 design
// 3.5): an addition domain.CheckNewRelations refuses, with no call; then a
// work item not there, or whose project the caller does not see,
// issue.not_found, and a role that may not change it, forbidden, before
// any other work item is read. Under the locks, the decision is final: web
// deleted while they waited, or the work item meanwhile, issue.not_found;
// a role that may no longer change it, forbidden; an archived project 409
// project.archived, its work item archived too, then an archived work item
// 409 issue.archived, before any other is checked. None writes a row.
func TestCreateIssueRelationsRefusals(t *testing.T) {
	for _, tt := range []struct {
		name  string
		kind  domain.RelationType
		set   func(w *world, id *uuid.UUID)
		want  error
		calls int // of reaching's, before the use case's own
	}{
		{"an unknown type", "start_before", func(*world, *uuid.UUID) {}, domain.CheckNewRelations("start_before", []uuid.UUID{web}), 0},
		{"no work item", domain.RelatesTo, func(_ *world, id *uuid.UUID) { *id = uuid.NewV7() }, domain.ErrIssueNotFound, 1},
		{"a project not seen", domain.RelatesTo, func(w *world, _ *uuid.UUID) { w.auth.answers = []error{shared.ErrNotVisible} },
			domain.ErrIssueNotFound, 2},
		{"a role not allowed", domain.RelatesTo, func(w *world, _ *uuid.UUID) { w.auth.answers = []error{shared.Forbidden()} },
			shared.Forbidden(), 2},
		{"web deleted while the locks waited", domain.RelatesTo, func(w *world, _ *uuid.UUID) { w.project.locked = nil }, domain.ErrIssueNotFound,
			8},
		{"the work item deleted meanwhile", domain.RelatesTo, func(w *world, _ *uuid.UUID) {
			w.project.waited = func() { w.issues.issue.ID = uuid.NewV7() }
		}, domain.ErrIssueNotFound, 9},
		{"no longer allowed", domain.RelatesTo, func(w *world, _ *uuid.UUID) { w.auth.answers = []error{3: shared.Forbidden()} },
			shared.Forbidden(), 10},
		{"an archived project", domain.RelatesTo, func(w *world, _ *uuid.UUID) { w.project.locked.Archived, w.issues.archivedAt = true, &earlier },
			domain.ErrProjectArchived, 14},
		{"an archived work item", domain.RelatesTo, func(w *world, _ *uuid.UUID) { w.issues.archivedAt = &earlier }, domain.ErrIssueArchived, 14},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w, rel, id, r := newRelate()
			tt.set(w, &id)
			_, err := uc.Execute(as(alice), id, tt.kind, []uuid.UUID{r.far.ID, r.near.ID})
			wantRefusal(t, w, err, tt.want, reaching(id, r)[:tt.calls])
			if rel.relations != nil || rel.marked != nil || w.issues.activities != nil {
				t.Errorf("wrote %+v, %+v, %+v; want nothing", rel.relations, rel.marked, w.issues.activities)
			}
		})
	}
}

// Each work item an addition names that the work item may not relate to
// is refused at its place in issue_ids (M4 design 3.4, 3.8, 3.10), and
// nothing is written. alice asks for far, then near. Not reached before
// any lock: one not there, the work item itself, another workspace's, one
// whose project she does not see, one she may not read, or another's when
// the guest rule hides others' from her in its project; under the locks,
// one archived, of an archived project, deleted while the locks waited,
// whose project was deleted while they waited, or no longer seen. Under
// them too, one that relates to the work item already is a duplicate.
// Every problem is reported at once, each at its place: far's and near's
// both, when both are refused.
func TestCreateIssueRelationsRefusesWhatItMayNotRelateTo(t *testing.T) {
	notAllowed, already := shared.Invalid(domain.RelatedNotAllowed(0)), shared.Invalid(domain.RelatedAlready(0))
	for _, tt := range []struct {
		name string
		set  func(c *relateCase)
		want error
	}{
		{"one not there", func(c *relateCase) { c.ask[0] = uuid.NewV7() }, notAllowed},
		{"the work item itself", func(c *relateCase) { c.ask[0] = c.id }, notAllowed},
		{"another workspace's", func(c *relateCase) { c.far.WorkspaceID = uuid.NewV7() }, notAllowed},
		{"of a project not seen", func(c *relateCase) { c.auth.answers = []error{1: shared.ErrNotVisible} }, notAllowed},
		{"not to be read", func(c *relateCase) { c.auth.answers = []error{1: shared.Forbidden()} }, notAllowed},
		{"another's, to a guest of its project who sees his own alone", func(c *relateCase) {
			c.auth.grants = []shared.Grant{1: {OwnOnly: true}}
		}, notAllowed},
		{"archived", func(c *relateCase) { c.far.ArchivedAt = &earlier }, notAllowed},
		{"of an archived project", func(c *relateCase) {
			c.project.across[ops] = app.LockedProject{WorkspaceID: acme, Archived: true, Identifier: "OPS"}
		}, notAllowed},
		{"deleted while the locks waited", func(c *relateCase) { c.project.waited = func() { delete(c.issues.others, farID) } }, notAllowed},
		{"its project deleted while the locks waited", func(c *relateCase) { delete(c.project.across, ops) }, notAllowed},
		{"no longer seen", func(c *relateCase) { c.auth.answers = []error{4: shared.ErrNotVisible} }, notAllowed},
		{"related already", func(c *relateCase) { c.rel.related = []uuid.UUID{farID} }, already},
		{"both refused", func(c *relateCase) { c.far.ArchivedAt, c.rel.related = &earlier, []uuid.UUID{nearID} },
			shared.Invalid(domain.RelatedNotAllowed(0), domain.RelatedAlready(1))},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w, rel, id, r := newRelate()
			c := &relateCase{world: w, rel: rel, id: id, far: r.far, ask: []uuid.UUID{farID, nearID}}
			tt.set(c)
			w.issues.others[farID] = c.far
			_, err := uc.Execute(as(alice), id, domain.RelatesTo, c.ask)
			var se, wantSE *shared.Error
			if !errors.As(err, &se) || !errors.As(tt.want, &wantSE) || se.Code != wantSE.Code || !reflect.DeepEqual(se.Fields, wantSE.Fields) {
				t.Errorf("Execute() = %v, want %v", err, tt.want)
			}
			if rel.relations != nil || rel.marked != nil || w.issues.activities != nil {
				t.Errorf("wrote %+v, %+v, %+v; want nothing", rel.relations, rel.marked, w.issues.activities)
			}
		})
	}
}

// relateCase is newRelate's world as a case sets it: far is stored as the
// case leaves it, and the addition asks for ask, far and near unless the
// case changes it.
type relateCase struct {
	*world
	rel *fakeRelations
	id  uuid.UUID
	far domain.Issue
	ask []uuid.UUID
}

// A port that fails ends the addition with its failure, as itself, and no
// call after it runs: the transaction rolls back whatever was written.
func TestCreateIssueRelationsStopsAtAFailingPort(t *testing.T) {
	for _, port := range []string{"GetIssue", "Authorize", line("GetIssue", farID), line("Authorize", alice, domain.ActionRead, acme, ops),
		"LockMany", "Related", "InsertRelations", "MarkChanged", "InsertActivities", "GetIssues", "SubIssueCounts"} {
		t.Run(port, func(t *testing.T) {
			uc, w, _, id, _ := newRelate()
			w.log.fail = port
			_, err := uc.Execute(as(alice), id, domain.RelatesTo, []uuid.UUID{farID, nearID})
			wantStoppedAt(t, w, err, port)
		})
	}
}

// What cannot be is an error, never a refusal: the others not all there
// to read back after the write, which found each undeleted under the
// locks.
func TestCreateIssueRelationsErrsOnWhatCannotBe(t *testing.T) {
	_, w, rel, id, r := newRelate()
	lost := losingRelations{fakeRelations: rel, gone: r.far.ID}
	uc := app.NewCreateIssueRelations(app.CreateIssueRelationsDeps{Locks: w.locks, Relations: lost, Clock: clockAt{at: clockNow, log: w.log}})
	_, err := uc.Execute(as(alice), id, domain.RelatesTo, []uuid.UUID{r.far.ID, r.near.ID})
	var se *shared.Error
	if err == nil || errors.As(err, &se) {
		t.Errorf("Execute() = %v, want an error that is no refusal", err)
	}
}

// losingRelations loses the work item gone from the store once the
// activity is written, so that it is not there to read back.
type losingRelations struct {
	*fakeRelations
	gone uuid.UUID
}

func (l losingRelations) InsertActivities(ctx context.Context, rows []app.ActivityRow) error {
	if err := l.fakeRelations.InsertActivities(ctx, rows); err != nil {
		return err
	}
	delete(l.others, l.gone)
	return nil
}
````

`server/internal/modules/issue/app/fakes_relations_test.go`（新文件，56 行）：

````file server/internal/modules/issue/app/fakes_relations_test.go
package app_test

import (
	"context"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// fakeRelations is the relations' store over the work items' fake
// (app.RelationCreator), each call logged: related are the work items
// that relate to the work item asked about already, whichever it is;
// relations the last insert's relations, marked the last work items
// marked changed.
type fakeRelations struct {
	*fakeIssues
	related   []uuid.UUID
	relations []domain.Relation
	marked    *app.ChangedRows
}

// Related logs the read and answers those of ids that the fake's related
// names, in ids' order.
func (f *fakeRelations) Related(ctx context.Context, id uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error) {
	if err := f.log.add(ctx, "Related", id, ids); err != nil {
		return nil, err
	}
	return slices.DeleteFunc(slices.Clone(ids), func(i uuid.UUID) bool { return !slices.Contains(f.related, i) }), nil
}

// InsertRelations logs the insert, of how many, and keeps the relations.
func (f *fakeRelations) InsertRelations(ctx context.Context, rs []domain.Relation) error {
	if err := f.log.add(ctx, "InsertRelations", len(rs)); err != nil {
		return err
	}
	f.relations = rs
	return nil
}

// MarkChanged logs the write, keeps its rows and marks each of the fake's
// others it names last changed by its writer at its moment.
func (f *fakeRelations) MarkChanged(ctx context.Context, r app.ChangedRows) error {
	if err := f.log.add(ctx, "MarkChanged", r.IDs); err != nil {
		return err
	}
	f.marked = &r
	for _, id := range r.IDs {
		if o, ok := f.others[id]; ok {
			o.UpdatedByID, o.UpdatedAt = &r.By, r.At
			f.others[id] = o
		}
	}
	return nil
}
````

`server/internal/modules/issue/app/lock_test.go`（修改，1 处）：

````old server/internal/modules/issue/app/lock_test.go
		t.Errorf("Execute() = %v after\n%q\nwant server_busy once the gate refuses it", err, w.log.calls)
	}
}

````
````new server/internal/modules/issue/app/lock_test.go
		t.Errorf("Execute() = %v after\n%q\nwant server_busy once the gate refuses it", err, w.log.calls)
	}
}

// A write across projects (convention 7, M4 design 3.5, 11.11) enters the
// gates of its work item's project and of each project of the work items
// it reaches, all at once in ascending id order, after its precheck and
// outside any transaction; it locks the same projects all at once, in the
// same order (LockMany), and leaves the gates last, once its transaction
// has ended. alice relates web's work item 7 to far, of ops, then near, of
// web: web and ops. A work item the precheck does not reach takes neither
// a lock nor a place in a gate: far, which alice does not see, or which is
// not there, leaves ops out of both, and the refusal (422) is answered
// under web's lock alone.
func TestAWriteAcrossProjectsTakesOnlyTheProjectsItReaches(t *testing.T) {
	for _, tt := range []struct {
		name     string
		set      func(w *world)
		projects []uuid.UUID
	}{
		{"both reached", func(*world) {}, sorted([]uuid.UUID{web, ops})},
		{"far not seen", func(w *world) { w.auth.answers = []error{1: shared.ErrNotVisible} }, []uuid.UUID{web}},
		{"far not there", func(w *world) { delete(w.issues.others, farID) }, []uuid.UUID{web}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, w, rel, id, _ := newRelate()
			gated(w, nil)
			uc := relateOver(w, rel)
			tt.set(w)
			_, _ = uc.Execute(as(alice), id, domain.RelatesTo, []uuid.UUID{farID, nearID})
			enter, lock := line("Enter", anys(tt.projects)...)+" outside tx", line("LockMany", anys(tt.projects)...)
			at := slices.Index(w.log.calls, enter)
			if at < 0 || at+2 >= len(w.log.calls) || w.log.calls[at+1] != "Begin" || w.log.calls[at+2] != lock ||
				w.log.calls[len(w.log.calls)-1] != line("Leave", anys(tt.projects)...) || slices.Contains(w.log.calls[:at], "Begin") {
				t.Errorf("calls:\n%q\nwant %q after the precheck, then the transaction and %q, and the gates left last", w.log.calls, enter, lock)
			}
		})
	}
}

````

- [ ] **Step 3: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/issue/...`
Expected: 全部 `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 4: 提交**

```bash
git add server/internal/modules/issue/app/create_relations.go server/internal/modules/issue/app/create_relations_test.go server/internal/modules/issue/app/fakes_relations_test.go server/internal/modules/issue/app/lock.go server/internal/modules/issue/app/lock_across.go server/internal/modules/issue/app/lock_test.go server/internal/modules/issue/app/parents.go server/internal/modules/issue/app/ports.go server/internal/modules/issue/app/sub_issues.go
```
```bash
git commit -m "feat(M4/P3): createIssueRelations, on the lock path across projects

Locks.run gains its form across projects: before any lock, each other
work item is read and issue.read decided on it, the guest rule first;
one the caller may not reach takes no lock and no place in a gate. The
write enters the gates of its projects and locks them all at once in
ascending id order, then reaches each again under the locks. The use
case refuses an archived project or work item, then every other work
item it may not relate to, at its place; it stores each relation once,
marks both ends changed and writes a row on each, by its identifier.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A；`mutants_p3.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试，或没有替身的平台、领域、转换的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；守卫是生成器（`make gen-go`）和中英文翻译键（`make lint-web`）的核对。每个变异在一份新的副本里，在本 Task 的树上、标出的较晚的 Task 的树上和最终的树上都跑过，端到端的检查只在标出的 Task 的树上）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T10.1` | 判定拒绝调用者的工作项也算够得着，被关联上 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestTheGuestRuleHoldsForRelations`（`make test`）（Task 11 起）、`internal/bootstrap` 的 `TestWhatAWorkItemMayNotBeRelatedTo`（`make test`）（Task 11 起） | 单元；组合 |
| `T10.2` | 调用者看不到的工作项占它的项目的闸门位置、取它的锁 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestARelationLocksNoProjectItDoesNotReach`（`make test`）（Task 11 起） | 单元；组合 |
| `T10.3` | 对另一个工作项的判定不对它执行访客规则 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestTheGuestRuleHoldsForRelations`（`make test`）（Task 11 起） | 单元；组合 |
| `T10.4` | 跨项目的写按够到的次序进闸门、取锁 | `internal/modules/issue/app` 的全部测试（`make test`） | 单元 |
| `T10.5` | 别的工作区的工作项也够得着，被关联上 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWhatAWorkItemMayNotBeRelatedTo`（`make test`）（Task 11 起） | 单元；组合 |
| `T10.6` | 工作项从它自己够得着，与它自己关联 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWhatAWorkItemMayNotBeRelatedTo`（`make test`）（Task 11 起） | 单元；组合 |
| `T10.7` | 跨项目的写只锁它自己的项目 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWorkItemsAreRelatedAcrossProjects`（`make test`）（Task 11 起）、`internal/bootstrap` 的 `TestRelatingAWorkItemAndDeletingTheOtherEndsProjectSerialize`（`make test`）（Task 14 起） | 单元；组合 |
| `T10.8` | 跨项目的写只进它自己的项目的闸门 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestEachWriteOnAWorkItemOfAProjectTooBusyAnswersServerBusy`（`make test`）（Task 11 起） | 单元；组合 |
| `T10.9` | 已归档的项目里的工作项可以被关联 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWhatAWorkItemMayNotBeRelatedTo`（`make test`）（Task 11 起） | 单元；组合 |
| `T10.10` | 已归档的工作项可以被关联 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWhatAWorkItemMayNotBeRelatedTo`（`make test`）（Task 11 起） | 单元；组合 |
| `T10.11` | 已有关联的工作项再关联一次 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWorkItemsAreRelatedAcrossProjects`（`make test`）（Task 11 起）、`internal/bootstrap` 的 `TestOppositeRelationsOfTwoWorkItemsSerialize`（`make test`）（Task 14 起） | 单元；组合 |
| `T10.12` | 已归档的工作项可以关联别的工作项 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWhatAWorkItemMayNotBeRelatedTo`（`make test`）（Task 11 起）、故事 `i15-archive-issue`（Task 15 起） | 单元；组合；端到端 |
| `T10.13` | 已归档的项目里已归档的工作项先答"工作项已归档" | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `createIssueRelations` 各格（`make test`）（Task 11 起） | 单元；组合 |
| `T10.14` | 加关联只改这个工作项的最后修改，不改对方 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWorkItemsAreRelatedAcrossProjects`（`make test`）（Task 11 起）、故事 `i9-relations`（Task 15 起） | 单元；组合；端到端 |
| `T10.15` | 加关联不在对方的动态里写行 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWorkItemsAreRelatedAcrossProjects`（`make test`）（Task 11 起）、故事 `i9-relations`（Task 15 起） | 单元；组合；端到端 |
| `T10.16` | 对方的那行动态写这个工作项读到的类型 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWorkItemsAreRelatedAcrossProjects`（`make test`）（Task 11 起）、故事 `i9-relations`（Task 15 起） | 单元；组合；端到端 |
| `T10.17` | 这个工作项的那行动态用它自己项目的标识称呼对方 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWorkItemsAreRelatedAcrossProjects`（`make test`）（Task 11 起）、故事 `i9-relations`（Task 15 起） | 单元；组合；端到端 |
| `T10.18` | 加关联回答的每条关联是存储的类型 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWorkItemsAreRelatedAcrossProjects`（`make test`）（Task 11 起）、故事 `i9-relations`（Task 15 起） | 单元；组合；端到端 |
| `T10.19` | 加关联回读到什么就答什么，少了一个工作项也答 | `internal/modules/issue/app` 的全部测试（`make test`） | 单元 |
| `T10.20` | 读已有的关联失败时略过：当作还没有关联 | `internal/modules/issue/app` 的全部测试（`make test`） | 单元 |
| `T10.21` | 加关联的回答按不带访客规则的授权数另一端的子工作项 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestTheGuestRuleHoldsForRelations`（`make test`）（Task 11 起） | 单元；组合 |
| `T10.22` | 加关联在锁下、闸门之后才核对类型和 1–20 个工作项，而不是在任何读之前 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestEachWriteOnAWorkItemOfAProjectTooBusyAnswersServerBusy`（`make test`）（Task 11 起） | 单元；组合 |

---

### Task 11: 加关联：接口、组合、矩阵；`furnishIssueRows` 建关联

**Files:**
- Create: `server/internal/bootstrap/gate_across_test.go`、`server/internal/bootstrap/issue_relation_locks_test.go`、`server/internal/bootstrap/issue_relations_test.go`、`server/internal/bootstrap/permission_matrix_relations_test.go`、`server/internal/modules/issue/adapter/http/relations.go`、`server/internal/modules/issue/adapter/http/relations_test.go`
- Modify: `api/modules/issue.yaml`、`api/openapi.yaml`、`e2e/fixtures/assert/project.ts`、`e2e/fixtures/assert/workspace.ts`、`e2e/fixtures/furnish.ts`、`e2e/fixtures/issues.ts`、`server/internal/bootstrap/gate_test.go`、`server/internal/bootstrap/issue_write_locks_test.go`、`server/internal/bootstrap/issue_writes_test.go`、`server/internal/bootstrap/permission_matrix_sub_issues_test.go`、`server/internal/bootstrap/permission_matrix_test.go`、`server/internal/modules/issue/adapter/http/handler.go`、`server/internal/modules/issue/module.go`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/issue/adapter/http/gen/bodyshape.gen.go`、`server/internal/modules/issue/adapter/http/gen/server.gen.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**（spec 2.13、第 3 节第 7、9、11、15、17、18 条；M4 设计 3.5、3.10、5.1、5.2、9.2、9.3，12 节约束 8）
- 契约：`POST /api/v0/issues/{issue_id}/relations`（`createIssueRelations`，体 `IssueRelationsCreate{relation_type, issue_ids}`（1–20 个，`additionalProperties: false`），201 `IssueRelationCollection{data}`，`x-problem-codes: [validation_failed, issue.not_found, forbidden, project.archived, issue.archived, server_busy]`）；模式 `IssueRelationType`（四种）、`IssueRelation{id, relation_type, issue, created_at}`（`issue` 是对方，`Issue` 的行）；`api/openapi.yaml` 的路径。描述写上谁能加（对它 `issue.update`，对每个对方 `issue.read`），对方的规则（同一个工作区、看得到、没归档、项目没归档、还没有关联），看不到的不锁，每条存一次、两行动态、两端的最后修改。
- `issue/adapter/http`：`handler.go` 的 `CreateIssueRelationsUseCase`、`UseCases.CreateIssueRelations`；`relations.go`（新）：`CreateIssueRelations`（201）、`issueRelationCollection`。`issue/module.go` 接上，说明写上它。
- `bootstrap`：`issue_relations_test.go`（新：`relationsOf`、`relating`、`answeredRelation`（对方是 `countedIssue`）、`othersOf`、`storedRelations`、`relationRows`；关联的列表用 Task 6 的 `listIn[answeredRelation]`，不另写一份）；`gate_across_test.go`（新：跨项目的写在闸门上的两个组合测试 `TestAWriteAcrossProjectsWaitsMaxWaitInAll`、`TestAWriteAcrossProjectsGivesBackEachPlaceItTook`；放进 `gate_test.go` 它会超过 400 行）；`issue_relation_locks_test.go`（新）；`issue_write_locks_test.go` 的 `createIssueRelations` 一行（对方是 Far；锁下读的第一张预检没读的表是 `issue_relations`，`ACCESS EXCLUSIVE`）；`permission_matrix_relations_test.go`（新：`relatesTo`、`relates`、`beforesItem`、`archivedItem`、`relatedTo`）并在 `permission_matrix_test.go` 登记；`permission_matrix_sub_issues_test.go` 的 `subItemOf` 改名 `otherItemOf`（两个操作共用）、已归档的项目里已归档的工作项一行用它；`gate_test.go` 的 `TestEachWriteOnAWorkItemOfAProjectTooBusyAnswersServerBusy`：说明写上跨项目的写进它到达的每个项目的闸门，加两行（Web 的工作项的关联；Ops 的工作项到 Web 的工作项的关联：Ops 的位置空着也被拒绝：它先进 Web 的闸门，在那里被拒绝），形状不对的请求加一行（列了 21 个工作项的加关联：Web 唯一的位置被占着时立刻 422 `issue_ids` `too_long`，不是 503）；`issue_writes_test.go` 的 `TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection`：alice 建 Ops 和它的 Far，以 `blocking` 把第一个工作项关联到 Far。
- 端到端：`e2e/fixtures/issues.ts` 的 `createIssueRelations`；`furnish.ts` 的 `furnishIssueRows` 再建一个工作项，第一个阻塞它（两个工作项、一条关联、各一行动态）；`assert/project.ts`、`assert/workspace.ts` 去掉 Task 8 的 `withoutRows`（删除项目、工作区的故事读得到关联表的行了）。

**Tests:**
- HTTP（`apitest.Main` 两个方向）：`TestCreateIssueRelations`（体的类型和工作项照原样、照次序进用例：不在四种里的类型、重复的、空的也是；201 和用例回答的关联）、`TestCreateIssueRelationsHoldsTheBodyToItsStructure`（没有类型或工作项、`null`、别的类型、不是 `id` 的、多出的字段：400 `bad_request`）、`TestCreateIssueRelationsRefusals`。
- 组合：`TestWorkItemsAreRelatedAcrossProjects`（bob 以 `blocking` 把 Web 的 Near 关联到 Web 的 Mid 和 Ops 的 Far：回答 Mid、Far，按 `id` 的次序，各 `blocking`；各存为对方的 `blocked_by`、对方的项目的行；三个工作项同一时刻由 bob 最后修改；每条关联一行 Near 的动态、一行对方的，各在自己的项目；再从 Mid 关联 Near（任何类型）、再关联 Far 各 422 `duplicate`）、`TestWhatAWorkItemMayNotBeRelatedTo`（Near 自己、已删除的、没有的、已归档的、bob 是成员的已归档的项目的（他看得到它的工作项：拒绝它的只有"项目没归档"）、bob 不是成员的 acme 的项目的、别的工作区的：各在自己的位置、同一个 422、什么都不写；可以的 Up 在它们中间；别的工作区的对看得到它的 alice 也被拒绝；不给工作项、给两次 Up、给 21 次 Up 各 422，什么都不写（不在四种里的类型是契约外的请求，组合的测试只发契约内的）；已归档的工作项、已归档的项目的工作项不能加关联（409））、`TestTheGuestRuleHoldsForRelations`（Mine 下有 alice 的 Low、walt 的 Below；walt 关联 His 到 Mine 和他看不到的 Hers：Hers 在它的位置被拒绝，回答与不存在的逐值相同，什么都不写；alice 关联 Hers 到 His；walt 只关联 Mine 照常，回答的 Mine 数 1 个子工作项；alice 的 Up 对 walt 是与不存在的逐值相同的 404）、`TestARelationLocksNoProjectItDoesNotReach`（别的事务以 `FOR UPDATE` 持着私有的 Lab 和公开的 Pub（bob 看不到 Lab、读不了 Pub 的工作项）；bob 把 Near 关联到它们的工作项：两个位置一次被拒绝，不等任何一个；看得到 Lab 的 alice 等它，放开之后关联成）；`TestEachIssueWriteLocksItsProjectFirst` 的一行；`TestEachWriteOnAWorkItemOfAProjectTooBusyAnswersServerBusy` 的三行；`TestAWriteAcrossProjectsWaitsMaxWaitInAll`（每个项目一个位置、排队一个、`max_wait` 两秒、等锁五秒：别的事务持着 Web 和 Ops 的行，alice 改两个项目里各一个工作项，各进闸门、等行；她从 Web 的工作项到 Ops 的的关联先在 Web 的闸门等；关联发出一秒半之后放开 Web 的行，那里的修改通过，关联进 Web、在 Ops 等；它在发出两秒后（一共 `max_wait`）被拒绝 503，不是 Ops 再等一个 `max_wait` 的三秒半：两者相隔一秒半，回答要在 `max_wait` 或之后、中点之前；放开 Ops 的行之后那里的修改和她的关联都通过）；`TestAWriteAcrossProjectsGivesBackEachPlaceItTook`（每个项目一个位置、不排队、等锁三秒：alice 从 Ops 的工作项到 Web 的工作项的关联通过之后，Ops 里立刻建得了 Far，关联放回了 Ops 的位置；Far 的修改等 Ops 的行、占着 Ops 唯一的位置时，从 Web 的工作项到 Far 的关联先占 Web 的位置（Web 的 `id` 小），在 Ops 的闸门立刻被拒绝 503，之后 Web 的工作项的修改照常通过：被拒绝的关联放回了 Web 的位置；放开 Ops 的行之后 Far 的修改通过）；`TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection`（一个连接的池：alice 以 `blocking` 把第一个工作项关联到 Ops 的 Far，两个项目一次锁住：锁、锁下的读、写和回答的读都在它的事务的连接上）；`TestPermissionMatrix` 的 `createIssueRelations` 行（照常、已归档的项目、已归档的项目里已归档的工作项（项目的 409）、已归档的工作项、别的项目的工作项、已归档的项目的工作项）、`TestThePermissionMatrixCoversEveryOperation`。
- 端到端：删除项目、工作区的故事（P4、W2、W3）要关联表的行，`furnishIssueRows` 建了它。

- [ ] **Step 1: 契约，生成**

`api/modules/issue.yaml`（修改，2 处）：

````old api/modules/issue.yaml
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/workspaces/{slug}/issues/{identifier}:
````
````new api/modules/issue.yaml
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/issues/{issue_id}/relations:
    parameters:
      - $ref: '#/components/parameters/IssueID'
    post:
      operationId: createIssueRelations
      tags: [issue]
      summary: Relate a work item to others
      description: >-
        For the admins and members of its project. Refused from the request
        alone, before the work item is looked at (validation_failed): a type
        that is none of the four (relation_type invalid_format), no work
        item or more than 20 (issue_ids too_short, too_long; more than 20 is
        the one problem of issue_ids named), and a work item named twice
        (issue_ids[i] duplicate). A work item that does not exist, is
        deleted, whose project the caller does not see, or that he did not
        create as a guest who sees only his own (getIssue) answers
        issue.not_found, whatever his role may do; one whose project he
        sees but whose work items he may not change, forbidden. Both are
        answered before any project is locked, and so is whether he sees
        each work item named: one he does not takes no lock, and is refused
        below. Then the work item's project and those of the work items
        named are locked together, and his role is decided again. A work
        item of an archived project relates to none (project.archived), nor
        then an archived work item (issue.archived). Each work item named is
        another work item of the workspace, of any of its projects, that the
        caller sees, as getIssue would answer it, not archived, of a project
        not archived (issue_ids[i] not_allowed), and that does not relate to
        this one yet, of any type, either way (issue_ids[i] duplicate):
        every problem at once. Two work items have one relation at most. A
        work item blocking another is stored the other way round, as the
        other's blocked_by, and reads as blocking from this one alone. Both
        work items of each relation are changed last by the caller, and the
        activity of each says it relates to the other, with the type it
        reads. Every row it writes is the caller's, at one moment. A project
        with too many writes on its work items at once answers server_busy,
        with Retry-After: try again then.
      security: [{bearer: []}]
      x-problem-codes: [validation_failed, issue.not_found, forbidden, project.archived, issue.archived, server_busy]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/IssueRelationsCreate'
      responses:
        '201':
          description: The relations made, as this work item reads them, in the order of the other work items' ids.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IssueRelationCollection'
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/workspaces/{slug}/issues/{identifier}:
````
````old api/modules/issue.yaml
            type: string
            format: uuid
    IssueCreate:
````
````new api/modules/issue.yaml
            type: string
            format: uuid
    IssueRelationType:
      description: >-
        A relation's type as one of its work items reads it: relates_to and
        duplicate read the same from both; the work item blocked_by another
        reads as blocking from it.
      type: string
      enum: [relates_to, duplicate, blocked_by, blocking]
    IssueRelation:
      description: A relation of a work item to another, as the work item reads it; issue is the other, as a list shows it.
      type: object
      additionalProperties: false
      required: [id, relation_type, issue, created_at]
      properties:
        id:
          type: string
          format: uuid
        relation_type:
          $ref: '#/components/schemas/IssueRelationType'
        issue:
          $ref: '#/components/schemas/Issue'
        created_at:
          type: string
          format: date-time
    IssueRelationCollection:
      description: Relations of a work item, all of them, without pages.
      type: object
      additionalProperties: false
      required: [data]
      properties:
        data:
          type: array
          items:
            $ref: '#/components/schemas/IssueRelation'
    IssueRelationsCreate:
      type: object
      additionalProperties: false
      required: [relation_type, issue_ids]
      properties:
        relation_type:
          $ref: '#/components/schemas/IssueRelationType'
        issue_ids:
          description: >-
            1–20 work items of the workspace, each named once, to relate the
            work item to.
          type: array
          items:
            type: string
            format: uuid
    IssueCreate:
````

`api/openapi.yaml`（修改，1 处）：

````old api/openapi.yaml
    $ref: 'modules/issue.yaml#/paths/~1api~1v0~1issues~1{issue_id}~1sub-issues'
````
````new api/openapi.yaml
    $ref: 'modules/issue.yaml#/paths/~1api~1v0~1issues~1{issue_id}~1sub-issues'
  /api/v0/issues/{issue_id}/relations:
    $ref: 'modules/issue.yaml#/paths/~1api~1v0~1issues~1{issue_id}~1relations'
````

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `17f75cb201a7c5ce0385551879ad0d056f817d1ed90d52e07aaba9cabaf61fea` | 3622 | `api/dist/openapi.yaml` |
| `de9219db556c068656160b574b754102861a8eac0974cdd74db05bb172acb68a` | 49 | `server/internal/modules/issue/adapter/http/gen/bodyshape.gen.go` |
| `1b5516c983be2910c1188b2066399b39f6e8d25c27850ad3e31034313683a11f` | 1571 | `server/internal/modules/issue/adapter/http/gen/server.gen.go` |
| `64b244fc8a2f82ee4670bf1f258fcc5b2d6fb15a9ae258d7d66ad2a95e849511` | 3938 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/issue/adapter/http/gen/bodyshape.gen.go server/internal/modules/issue/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 2: HTTP 和接线**

`server/internal/modules/issue/adapter/http/handler.go`（修改，3 处）：

````old server/internal/modules/issue/adapter/http/handler.go
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http/gen"
````
````new server/internal/modules/issue/adapter/http/handler.go
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
````
````old server/internal/modules/issue/adapter/http/handler.go
}

// UseCases are the use cases behind the module's operations.
````
````new server/internal/modules/issue/adapter/http/handler.go
}

// CreateIssueRelationsUseCase is app.CreateIssueRelations.
type CreateIssueRelationsUseCase interface {
	Execute(ctx context.Context, id uuid.UUID, t domain.RelationType, ids []uuid.UUID) ([]app.IssueRelation, error)
}

// UseCases are the use cases behind the module's operations.
````
````old server/internal/modules/issue/adapter/http/handler.go
	AddSubIssues         AddSubIssuesUseCase
````
````new server/internal/modules/issue/adapter/http/handler.go
	AddSubIssues         AddSubIssuesUseCase
	CreateIssueRelations CreateIssueRelationsUseCase
````

`server/internal/modules/issue/adapter/http/relations.go`（新文件，32 行）：

````file server/internal/modules/issue/adapter/http/relations.go
package httpadapter

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// CreateIssueRelations serves POST /api/v0/issues/{issue_id}/relations:
// the type and the work items go to the use case as they came, which
// checks them.
func (h handler) CreateIssueRelations(ctx context.Context, req gen.CreateIssueRelationsRequestObject) (gen.CreateIssueRelationsResponseObject,
	error) {
	relations, err := h.uc.CreateIssueRelations.Execute(ctx, req.IssueID, domain.RelationType(req.Body.RelationType), req.Body.IssueIds)
	if err != nil {
		return nil, err
	}
	return gen.CreateIssueRelations201JSONResponse(issueRelationCollection(relations)), nil
}

// issueRelationCollection is relations as the API lists them, in their
// order, [] for none: each with its type as its work item reads it, and
// the other work item as a list shows it (issueCollection).
func issueRelationCollection(relations []app.IssueRelation) gen.IssueRelationCollection {
	data := make([]gen.IssueRelation, len(relations))
	for n, r := range relations {
		data[n] = gen.IssueRelation{ID: r.ID, RelationType: gen.IssueRelationType(r.Type), Issue: gen.Issue(issueDetail(r.Issue)), CreatedAt: r.CreatedAt}
	}
	return gen.IssueRelationCollection{Data: data}
}
````

`server/internal/modules/issue/adapter/http/relations_test.go`（新文件，143 行）：

````file server/internal/modules/issue/adapter/http/relations_test.go
package httpadapter_test

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeRelate is createIssueRelations: each call is recorded as "caller
// work item type", with the work items it got; it answers answer, or err.
type fakeRelate struct {
	calls  []string
	got    [][]uuid.UUID
	answer []app.IssueRelation
	err    error
}

func (f *fakeRelate) Execute(ctx context.Context, id uuid.UUID, t domain.RelationType, ids []uuid.UUID) ([]app.IssueRelation, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	f.calls = append(f.calls, actor.UserID.String()+" "+id.String()+" "+string(t))
	f.got = append(f.got, ids)
	return f.answer, f.err
}

// relationsPath is the path of WEB-6's relations; relatedAnswers, the
// answers of the use case: WEB-6 blocked by WEB-8 and a duplicate of
// WEB-7, each relation with an id, a type and a moment of its own, so that
// each shows as its own, the other work item as getIssue shows it, in the
// use case's order; or none.
var (
	relationsPath  = "/api/v0/issues/" + webSixID.String() + "/relations"
	relatedAnswers = []struct {
		relations []app.IssueRelation
		json      string
	}{
		{[]app.IssueRelation{
			{ID: uuid.MustParse("0199a2b4-0000-7000-8000-0000000000f1"), Type: domain.BlockedBy, Issue: webEight, CreatedAt: created.Add(time.Hour)},
			{ID: uuid.MustParse("0199a2b4-0000-7000-8000-0000000000f2"), Type: domain.Duplicate, Issue: webSeven, CreatedAt: created.Add(2 * time.Hour)},
		}, `{"data":[{"created_at":"2026-10-17T11:00:00.123456Z","id":"0199a2b4-0000-7000-8000-0000000000f1","issue":` + webEightJSON +
			`,"relation_type":"blocked_by"},{"created_at":"2026-10-17T12:00:00.123456Z","id":"0199a2b4-0000-7000-8000-0000000000f2","issue":` +
			webSevenJSON + `,"relation_type":"duplicate"}]}`},
		{nil, `{"data":[]}`},
	}
)

// POST goes to the use case for the caller and the path's work item, with
// the body's type and work items as they came: a type that is none of the
// four too, a work item named twice too, or none, in their order; the
// answer is 201 with the relations it answers, in its order, [] for none.
func TestCreateIssueRelations(t *testing.T) {
	for _, tt := range []struct {
		body string
		kind domain.RelationType
		ids  []uuid.UUID
	}{
		{`{"relation_type":"blocking","issue_ids":["0199a2b4-0000-7000-8000-0000000000e8","0199a2b4-0000-7000-8000-0000000000e7",` +
			`"0199a2b4-0000-7000-8000-0000000000e8"]}`, domain.Blocking, []uuid.UUID{webEight.ID, webSeven.ID, webEight.ID}},
		{`{"relation_type":"start_before","issue_ids":[]}`, "start_before", []uuid.UUID{}},
	} {
		for _, answer := range relatedAnswers {
			relate := &fakeRelate{answer: answer.relations}
			res, body := do(t, newServer(t, httpadapter.UseCases{CreateIssueRelations: relate}), request(http.MethodPost, relationsPath, "alice", tt.body))
			if res.StatusCode != http.StatusCreated || body != answer.json+"\n" {
				t.Errorf("POST %s = %d %s, want 201 %s", tt.body, res.StatusCode, body, answer.json)
			}
			if want := []string{aliceID.String() + " " + webSixID.String() + " " + string(tt.kind)}; !slices.Equal(relate.calls, want) ||
				!reflect.DeepEqual(relate.got, [][]uuid.UUID{tt.ids}) {
				t.Errorf("POST %s: calls %q with %v; want %q with %v", tt.body, relate.calls, relate.got, want, tt.ids)
			}
		}
	}
}

// A body without the type or the work items, with null or another type
// for either, an id that is no id, or a field it may not have: refused as
// bad_request before the use case.
func TestCreateIssueRelationsHoldsTheBodyToItsStructure(t *testing.T) {
	relate := &fakeRelate{}
	h := newServer(t, httpadapter.UseCases{CreateIssueRelations: relate})
	for _, body := range []string{`{}`, `{"issue_ids":["0199a2b4-0000-7000-8000-0000000000e8"]}`, `{"relation_type":"blocking"}`,
		`{"relation_type":null,"issue_ids":[]}`, `{"relation_type":1,"issue_ids":[]}`, `{"relation_type":"blocking","issue_ids":null}`,
		`{"relation_type":"blocking","issue_ids":"WEB-8"}`, `{"relation_type":"blocking","issue_ids":["WEB-8"]}`,
		`{"relation_type":"blocking","issue_ids":[],"sub_issue_ids":[]}`} {
		res, got := do(t, h, request(http.MethodPost, relationsPath, "alice", body))
		var problem struct{ Code string }
		if err := json.Unmarshal([]byte(got), &problem); err != nil || res.StatusCode != http.StatusBadRequest || problem.Code != "bad_request" {
			t.Errorf("POST %s = %d %s, want 400 bad_request", body, res.StatusCode, got)
		}
	}
	if len(relate.calls) != 0 {
		t.Errorf("the use case got %q, want nothing", relate.calls)
	}
}

// The use case's refusals, as the contract declares them, and its
// failure.
func TestCreateIssueRelationsRefusals(t *testing.T) {
	invalid := `{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.","errors":[`
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"a type of none of the four, and none named", domain.CheckNewRelations("start_before", nil), http.StatusUnprocessableEntity,
			invalid + `{"field":"relation_type","code":"invalid_format","message":"must be one of relates_to, duplicate, blocked_by, blocking"},` +
				`{"field":"issue_ids","code":"too_short","message":"must name a work item"}]}`},
		{"a work item refused, and one related already", shared.Invalid(domain.RelatedNotAllowed(1), domain.RelatedAlready(2)),
			http.StatusUnprocessableEntity, invalid + `{"field":"issue_ids[1]","code":"not_allowed","message":"must be another work item of ` +
				`the workspace that you see, not archived, in a project that is not archived"},` +
				`{"field":"issue_ids[2]","code":"duplicate","message":"relates to this work item already"}]}`},
		{"no work item", domain.ErrIssueNotFound, http.StatusNotFound,
			`{"status":404,"code":"issue.not_found","title":"Not Found","detail":"The work item does not exist, or you cannot see it."}`},
		{"a project guest", shared.Forbidden(), http.StatusForbidden,
			`{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
		{"an archived project", domain.ErrProjectArchived, http.StatusConflict,
			`{"status":409,"code":"project.archived","title":"Conflict","detail":"The project is archived; unarchive it to change it."}`},
		{"an archived work item", domain.ErrIssueArchived, http.StatusConflict,
			`{"status":409,"code":"issue.archived","title":"Conflict","detail":"The work item is archived; unarchive it to change it."}`},
		{"a busy project", shared.ServerBusy(time.Second), http.StatusServiceUnavailable, busy},
		{"a failure", errGone, http.StatusInternalServerError, `{"status":500,"code":"internal_error","title":"Internal Server Error"}`},
	} {
		h := newServer(t, httpadapter.UseCases{CreateIssueRelations: &fakeRelate{err: tt.err}})
		body := `{"relation_type":"blocking","issue_ids":["0199a2b4-0000-7000-8000-0000000000e8"]}`
		res, got := do(t, h, request(http.MethodPost, relationsPath, "alice", body))
		if res.StatusCode != tt.status || got != tt.want+"\n" || tt.want == busy && res.Header.Get("Retry-After") != "1" {
			t.Errorf("%s: POST = %d %s, Retry-After %q; want %d %s", tt.name, res.StatusCode, got, res.Header.Get("Retry-After"), tt.status, tt.want)
		}
	}
}
````

`server/internal/modules/issue/module.go`（修改，2 处）：

````old server/internal/modules/issue/module.go
// changing, archiving, unarchiving and deleting a work item, and listing
// the work items under one and putting work items under it. For the
````
````new server/internal/modules/issue/module.go
// changing, archiving, unarchiving and deleting a work item, listing the
// work items under one and putting work items under it, and relating a
// work item to others. For the
````
````old server/internal/modules/issue/module.go
		AddSubIssues:   app.NewAddSubIssues(app.AddSubIssuesDeps{Locks: locks, Issues: store, Clock: d.Clock}),
````
````new server/internal/modules/issue/module.go
		AddSubIssues:   app.NewAddSubIssues(app.AddSubIssuesDeps{Locks: locks, Issues: store, Clock: d.Clock}),
		CreateIssueRelations: app.NewCreateIssueRelations(app.CreateIssueRelationsDeps{
			Locks: locks, Relations: store, Clock: d.Clock,
		}),
````

- [ ] **Step 3: 组合、取锁形状、写闸门和矩阵**

`server/internal/bootstrap/gate_across_test.go`（新文件，119 行）：

````file server/internal/bootstrap/gate_across_test.go
package bootstrap

import (
	"net/http"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// A write across projects waits max_wait in all at their gates, not
// max_wait at each (M4 design 3.5, 11.11; config's rule that max_wait and
// write_lock_timeout together stay under request_timeout counts one). In
// an app that lets one write a project in at once, queues one more, ends a
// wait at the gate at two seconds and a write's wait for a lock at five:
// transactions hold Web's row and Ops', and alice's change of a work item
// of each enters its gate and waits for its row. Her relation of the work
// item of Web to the one of Ops waits at Web's gate first, Web's id the
// lower. Web's row is freed a second and a half after the relation was
// sent: her change there goes through, and the relation enters Web's gate
// and waits at Ops'. It is refused 503 server_busy at max_wait in all, two
// seconds after it was sent, where a wait of max_wait at Ops' gate alone
// would end at three and a half: the two are a second and a half apart,
// and the answer must come at max_wait at the earliest and before halfway
// between them. Once Ops' row is free, her change there goes through, and
// so does her relation.
func TestAWriteAcrossProjectsWaitsMaxWaitInAll(t *testing.T) {
	const maxWait, freed = 2 * time.Second, 1500 * time.Millisecond
	w := newIssueWorldWith(t, func(cfg *config.Config) {
		cfg.Issue = config.IssueConfig{WriteGate: config.WriteGateConfig{PerProject: 1, MaxWait: maxWait, MaxQueue: 1},
			WriteLockTimeout: 5 * time.Second}
	})
	ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
	near, far := w.newWorkItem(t), w.newWorkItemIn(t, ops, `{"name":"Far"}`)
	var rows []pgx.Tx
	var changed []<-chan answer
	for _, held := range []struct{ project, item uuid.UUID }{{w.web, near}, {ops, far}} {
		change := newRequest(t, http.MethodPatch, w.base+"/api/v0/issues/"+held.item.String(), w.tokens["alice"], []byte(`{"name":"Mend"}`))
		w.contract.CheckRequest(t, change)
		row := holding(t, w.pool, "SELECT 1 FROM projects WHERE id = $1 FOR UPDATE", held.project)
		changed = append(changed, sendInBackground(change))
		pgtest.WaitForLockWaitBehind(t, w.pool, row.Conn().PgConn(), 5*time.Second)
		rows = append(rows, row)
	}
	relate := newRequest(t, http.MethodPost, w.base+relationsOf(near), w.tokens["alice"], []byte(relating("relates_to", far)))
	w.contract.CheckRequest(t, relate)

	sent := time.Now()
	related := sendInBackground(relate)
	time.Sleep(time.Until(sent.Add(freed)))
	release(t, rows[0])
	if a := receiveWithin(t, changed[0], 4*time.Second, "answer to her change in Web"); a.err != nil || a.res.StatusCode != http.StatusOK {
		t.Fatalf("her change in Web = %v %s, want 200 once Web's row is free", a.err, a.body)
	}
	a := receiveWithin(t, related, 4*time.Second, "answer to her relation")
	if a.err != nil {
		t.Fatal(a.err)
	}
	if took := time.Since(sent); took < maxWait || took >= maxWait+freed/2 {
		t.Errorf("her relation answered after %v; want at max_wait in all, %v, before %v", took, maxWait, maxWait+freed/2)
	}
	w.wantBusy(t, "her relation", relate, a.res, a.body)

	release(t, rows[1])
	if a = receiveWithin(t, changed[1], 4*time.Second, "answer to her change in Ops"); a.err != nil || a.res.StatusCode != http.StatusOK {
		t.Fatalf("her change in Ops = %v %s, want 200 once Ops' row is free", a.err, a.body)
	}
	w.send(t, "alice", http.MethodPost, relationsOf(near), relating("relates_to", far), http.StatusCreated)
}

// A write across projects gives back each place it took at their gates,
// gone through or refused (M4 design 3.5, 11.11). In an app that lets one
// write a project in at once, queues none and ends a write's wait for a
// lock at three seconds: once alice's relation of a work item of Ops to
// one of Web has gone through, a work item, Far, is made in Ops at once.
// While her change of Far waits for Ops' row, which a transaction holds,
// and holds Ops' one place, her relation of the work item of Web to Far
// takes Web's place first, Web's id the lower, and is refused at once at
// Ops' gate: her change of the work item of Web then goes through. Once
// Ops' row is free, her change of Far goes through.
func TestAWriteAcrossProjectsGivesBackEachPlaceItTook(t *testing.T) {
	w := newIssueWorldWith(t, func(cfg *config.Config) {
		cfg.Issue = config.IssueConfig{WriteGate: config.WriteGateConfig{PerProject: 1, MaxWait: time.Minute},
			WriteLockTimeout: 3 * time.Second}
	})
	ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
	item := w.newWorkItem(t)
	w.send(t, "alice", http.MethodPost, relationsOf(w.newWorkItemIn(t, ops, `{"name":"First"}`)), relating("relates_to", item),
		http.StatusCreated)
	far := w.newWorkItemIn(t, ops, `{"name":"Far"}`)
	mend := newRequest(t, http.MethodPatch, w.base+"/api/v0/issues/"+far.String(), w.tokens["alice"], []byte(`{"name":"Mend"}`))
	w.contract.CheckRequest(t, mend)
	across := newRequest(t, http.MethodPost, w.base+relationsOf(item), w.tokens["alice"], []byte(relating("relates_to", far)))
	w.contract.CheckRequest(t, across)
	held := holding(t, w.pool, "SELECT 1 FROM projects WHERE id = $1 FOR UPDATE", ops)
	mended := sendInBackground(mend)
	pgtest.WaitForLockWaitBehind(t, w.pool, held.Conn().PgConn(), 5*time.Second)
	begun := time.Now()
	a := receiveWithin(t, sendInBackground(across), 5*time.Second, "answer to her relation of the work item of Web to Far")
	if a.err != nil {
		t.Fatal(a.err)
	}
	if took := time.Since(begun); took >= time.Second {
		t.Errorf("her relation of the work item of Web to Far answered after %v; want at once, refused at Ops' gate", took)
	}
	w.wantBusy(t, "her relation of the work item of Web to Far", across, a.res, a.body)
	w.send(t, "alice", http.MethodPatch, "/api/v0/issues/"+item.String(), `{"name":"Mended"}`, http.StatusOK)
	release(t, held)
	if a = receiveWithin(t, mended, 4*time.Second, "answer to her change of Far"); a.err != nil {
		t.Fatal(a.err)
	}
	if a.res.StatusCode != http.StatusOK {
		t.Errorf("her change of Far = %d %s, want 200 once Ops' row is free", a.res.StatusCode, a.body)
	}
}
````

`server/internal/bootstrap/gate_test.go`（修改，5 处）：

````old server/internal/bootstrap/gate_test.go
	"net/http"
````
````new server/internal/bootstrap/gate_test.go
	"net/http"
	"slices"
````
````old server/internal/bootstrap/gate_test.go
// item of Web enters the gate and waits for the row. A write refused for
// the shape of its body takes no place at all: an addition of sub-items
// naming a work item twice is refused 422 at once, though Web's one place
// is taken. Once the row is free, her change goes through, and so does
// each write refused.
````
````new server/internal/bootstrap/gate_test.go
// item of Web enters the gate and waits for the row. A write across
// projects enters the gate of each project it reaches: her relation of a
// work item of Ops to one of Web is refused so too, though Ops' place is
// free. A write refused for the shape of its body takes no place at all:
// an addition of sub-items naming a work item twice, and one of relations
// naming 21 work items, are each refused 422 at once, though Web's one
// place is taken. Once the row is free, her change goes through, and so
// does each write refused. That a write across projects gives back each
// place it took is TestAWriteAcrossProjectsGivesBackEachPlaceItTook.
````
````old server/internal/bootstrap/gate_test.go
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+archived.String()+"/archive", "", http.StatusOK)
````
````new server/internal/bootstrap/gate_test.go
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+archived.String()+"/archive", "", http.StatusOK)
	ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
````
````old server/internal/bootstrap/gate_test.go
		{"an addition of sub-items", http.MethodPost, subIssues(w.newWorkItem(t)), subIssueIDs(w.newWorkItem(t)), http.StatusOK},
````
````new server/internal/bootstrap/gate_test.go
		{"an addition of sub-items", http.MethodPost, subIssues(w.newWorkItem(t)), subIssueIDs(w.newWorkItem(t)), http.StatusOK},
		{"an addition of relations", http.MethodPost, relationsOf(w.newWorkItem(t)), relating("relates_to", w.newWorkItem(t)),
			http.StatusCreated},
		{"an addition of relations from Ops", http.MethodPost, relationsOf(w.newWorkItemIn(t, ops, `{"name":"Far"}`)),
			relating("relates_to", w.newWorkItem(t)), http.StatusCreated},
````
````old server/internal/bootstrap/gate_test.go
		{"an addition of sub-items naming one twice", subIssues(item), subIssueIDs(archived, archived), "sub_issue_ids[1] duplicate"},
````
````new server/internal/bootstrap/gate_test.go
		{"an addition of sub-items naming one twice", subIssues(item), subIssueIDs(archived, archived), "sub_issue_ids[1] duplicate"},
		{"an addition of relations naming 21", relationsOf(item), relating("relates_to", slices.Repeat([]uuid.UUID{archived}, 21)...),
			"issue_ids too_long"},
````

`server/internal/bootstrap/issue_relation_locks_test.go`（新文件，60 行）：

````file server/internal/bootstrap/issue_relation_locks_test.go
package bootstrap

import (
	"net/http"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// A relation to a work item the caller may not reach takes no lock of its
// project (M4 design 3.5 convention 7, 9.3 interleaving 21, adversarial
// review M1): Lab, private, which bob does not see, and Pub, public,
// whose work items he may not read, are held FOR UPDATE by other
// transactions, which a lock of either waits for. bob relates Near, of
// Web, to Lab's and Pub's work items: refused at both places at once,
// without waiting for either. alice, who reaches Lab's, waits for it: a
// write locks the projects it reaches; released, she relates them.
func TestARelationLocksNoProjectItDoesNotReach(t *testing.T) {
	w := newIssueWorld(t)
	var lab struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/workspaces/acme/projects", `{"name":"Lab","identifier":"LAB","network":0}`,
		http.StatusCreated), &lab)
	pub := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Pub", "PUB")
	labs, pubs := w.newWorkItemIn(t, lab.ID, `{"name":"Lab's"}`), w.newWorkItemIn(t, pub, `{"name":"Pub's"}`)
	near := w.newWorkItemIn(t, w.web, `{"name":"Near"}`)
	labHeld := holding(t, w.pool, "SELECT 1 FROM projects WHERE id = $1 FOR UPDATE", lab.ID)
	pubHeld := holding(t, w.pool, "SELECT 1 FROM projects WHERE id = $1 FOR UPDATE", pub)

	sent := func(name, body string) (*http.Request, <-chan answer) {
		req := newRequest(t, http.MethodPost, w.base+relationsOf(near), w.tokens[name], []byte(body))
		w.contract.CheckRequest(t, req)
		return req, sendInBackground(req)
	}
	answered := func(name string, req *http.Request, ch <-chan answer) answer {
		t.Helper()
		a := receiveWithin(t, ch, 10*time.Second, name+"'s answer")
		if a.err != nil {
			t.Fatal(a.err)
		}
		w.contract.CheckResponse(t, req, a.res)
		return a
	}
	req, bobs := sent("bob", relating("relates_to", labs, pubs))
	if a, want := answered("bob", req, bobs), "validation_failed issue_ids[0] not_allowed; issue_ids[1] not_allowed"; a.res.StatusCode !=
		http.StatusUnprocessableEntity || refusalOf(t, a.body, false) != want {
		t.Errorf("bob's relation = %d %s, want 422 %s", a.res.StatusCode, a.body, want)
	}

	req, alices := sent("alice", relating("relates_to", labs))
	pgtest.WaitForLockWaitBehind(t, w.pool, labHeld.Conn().PgConn(), 10*time.Second)
	release(t, labHeld)
	if a := answered("alice", req, alices); a.res.StatusCode != http.StatusCreated {
		t.Errorf("alice's relation = %d %s, want 201", a.res.StatusCode, a.body)
	}
	release(t, pubHeld)
}
````

`server/internal/bootstrap/issue_relations_test.go`（新文件，292 行）：

````file server/internal/bootstrap/issue_relations_test.go
package bootstrap

import (
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// relationsOf is the path of the relations of the work item id.
func relationsOf(id uuid.UUID) string {
	return "/api/v0/issues/" + id.String() + "/relations"
}

// relating is the body that relates a work item, as kind, to the work
// items ids, in their order.
func relating(kind string, ids ...uuid.UUID) string {
	quoted := make([]string, len(ids))
	for n, id := range ids {
		quoted[n] = `"` + id.String() + `"`
	}
	return `{"relation_type":"` + kind + `","issue_ids":[` + strings.Join(quoted, ",") + `]}`
}

// answeredRelation is a relation as an answer has it.
type answeredRelation struct {
	ID           uuid.UUID    `json:"id"`
	RelationType string       `json:"relation_type"`
	Issue        countedIssue `json:"issue"`
	CreatedAt    time.Time    `json:"created_at"`
}

// othersOf are the other work items of relations, in their order.
func othersOf(relations []answeredRelation) []countedIssue {
	others := make([]countedIssue, len(relations))
	for n, r := range relations {
		others[n] = r.Issue
	}
	return others
}

// storedRelations are the undeleted relations, sorted, each as "<work
// item> <type> <work item>, in <project's identifier>, by <maker>, <last
// writer> at <moment>", the work items by their names, an account by its
// name in the world, the moment "the moment" when it was made and changed
// last at at, else "another moment".
func (w issueWorld) storedRelations(t *testing.T, at time.Time) []string {
	t.Helper()
	rows, err := w.pool.Query(pgtest.Soon(t), `SELECT i.name, r.relation_type, o.name, p.identifier, r.created_by_id, r.updated_by_id,
		r.created_at, r.updated_at FROM issue_relations r JOIN issues i ON i.id = r.issue_id JOIN issues o ON o.id = r.related_issue_id
		JOIN projects p ON p.id = r.project_id WHERE r.deleted_at IS NULL`)
	if err != nil {
		t.Fatal(err)
	}
	names := w.names()
	lines, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (string, error) {
		var from, kind, to, project string
		var madeBy, changedBy uuid.UUID
		var madeAt, changedAt time.Time
		if err := row.Scan(&from, &kind, &to, &project, &madeBy, &changedBy, &madeAt, &changedAt); err != nil {
			return "", err
		}
		when := "at the moment"
		if !madeAt.Equal(at) || !changedAt.Equal(at) {
			when = "at another moment"
		}
		return from + " " + kind + " " + to + ", in " + project + ", by " + names[madeBy] + ", " + names[changedBy] + " " + when, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return slices.Sorted(slices.Values(lines))
}

// relationRows are the rows of the work item item's activity on its
// relations (rows 20–23 of M4 design 3.13), in their ids' order, each as
// "<actor>: <verb> <field> <old value> → <new value>, <old work item> →
// <new work item>, in <project's identifier>, <comment>", a work item by
// its name, none for NULL, and " at the moment" when it was made at at.
func (w issueWorld) relationRows(t *testing.T, item uuid.UUID, at time.Time) []string {
	t.Helper()
	rows, err := w.pool.Query(pgtest.Soon(t), `SELECT a.actor_id, a.verb, a.field, a.old_value, a.new_value, coalesce(o.name, 'none'),
		coalesce(n.name, 'none'), p.identifier, a.comment, a.created_at FROM issue_activities a JOIN projects p ON p.id = a.project_id
		LEFT JOIN issues o ON o.id = a.old_identifier LEFT JOIN issues n ON n.id = a.new_identifier
		WHERE a.issue_id = $1 AND a.field IN ('relates_to', 'duplicate', 'blocked_by', 'blocking') ORDER BY a.id`, item)
	if err != nil {
		t.Fatal(err)
	}
	names := w.names()
	said, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (string, error) {
		var actor uuid.UUID
		var verb, field, oldValue, newValue, oldItem, newItem, project, comment string
		var madeAt time.Time
		if err := row.Scan(&actor, &verb, &field, &oldValue, &newValue, &oldItem, &newItem, &project, &comment, &madeAt); err != nil {
			return "", err
		}
		line := names[actor] + ": " + verb + " " + field + " " + oldValue + " → " + newValue + ", " + oldItem + " → " + newItem + ", in " +
			project + ", " + comment
		if madeAt.Equal(at) {
			line += " at the moment"
		}
		return line, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return said
}

// names are the world's accounts' names, by their ids.
func (w issueWorld) names() map[uuid.UUID]string {
	names := make(map[uuid.UUID]string, len(w.ids))
	for name, account := range w.ids {
		names[account] = name
	}
	return names
}

// Relations as bootstrap wires them (M4 design 3.5, 3.7, 3.10, 3.13).
// alice makes Ops, of acme, with bob its member, then Near and Mid in Web,
// Far in Ops. bob relates Near to Far and Mid, blocking them: the answer
// is both, Mid then Far, in id order, each blocking, the other work item
// changed last by bob, at one moment; each is stored the other way round,
// as the other's blocked_by, a row of the other's project; Near, Mid and
// Far are changed last by bob then, and each relation is a row of Near's
// activity and one of the other's, in each one's project, the type as it
// reads it. Mid relating to Near, of any type, and Near to Far again, are
// refused as related already, at their places, nothing written.
func TestWorkItemsAreRelatedAcrossProjects(t *testing.T) {
	w := newIssueWorld(t)
	ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+ops.String()+"/members",
		`{"members":[{"member_id":"`+w.ids["bob"].String()+`","role":15}]}`, http.StatusCreated)
	near := w.newWorkItemIn(t, w.web, `{"name":"Near"}`)
	mid := w.newWorkItemIn(t, w.web, `{"name":"Mid"}`)
	far := w.newWorkItemIn(t, ops, `{"name":"Far"}`)

	made := listIn[answeredRelation](t, w.send(t, "bob", http.MethodPost, relationsOf(near), relating("blocking", far, mid), http.StatusCreated))
	if len(made) != 2 {
		t.Fatalf("the answer is %d relations, want 2", len(made))
	}
	at := made[0].CreatedAt
	for n, want := range []uuid.UUID{mid, far} {
		r := made[n]
		if r.RelationType != "blocking" || r.Issue.ID != want || r.Issue.UpdatedByID == nil || *r.Issue.UpdatedByID != w.ids["bob"] ||
			!r.Issue.UpdatedAt.Equal(at) || !r.CreatedAt.Equal(at) {
			t.Errorf("relation %d is %+v; want blocking %s, changed last by bob, at %v", n, r, want, at)
		}
	}
	if got, want := w.storedRelations(t, at), []string{"Far blocked_by Near, in OPS, by bob, bob at the moment",
		"Mid blocked_by Near, in WEB, by bob, bob at the moment"}; !slices.Equal(got, want) {
		t.Errorf("the relations stored:\n%q\nwant\n%q", got, want)
	}
	for _, item := range []uuid.UUID{near, mid, far} {
		if _, changed, by := archivalOf(t, w, item); !changed.Equal(at) || by != w.ids["bob"] {
			t.Errorf("the work item %s is changed last by %s at %v, want by bob at %v", item, by, changed, at)
		}
	}
	for _, tt := range []struct {
		item uuid.UUID
		want []string
	}{
		{near, []string{"bob: updated blocking  → WEB-2, none → Mid, in WEB, added blocking relation at the moment",
			"bob: updated blocking  → OPS-1, none → Far, in WEB, added blocking relation at the moment"}},
		{mid, []string{"bob: updated blocked_by  → WEB-1, none → Near, in WEB, added blocked_by relation at the moment"}},
		{far, []string{"bob: updated blocked_by  → WEB-1, none → Near, in OPS, added blocked_by relation at the moment"}},
	} {
		if got := w.relationRows(t, tt.item, at); !slices.Equal(got, tt.want) {
			t.Errorf("the rows of %s's relations:\n%q\nwant\n%q", tt.item, got, tt.want)
		}
	}

	for _, tt := range []struct {
		item uuid.UUID
		body string
	}{{mid, relating("relates_to", near)}, {near, relating("duplicate", far)}} {
		answer := w.send(t, "bob", http.MethodPost, relationsOf(tt.item), tt.body, http.StatusUnprocessableEntity)
		if got, want := refusalOf(t, []byte(answer), false), "validation_failed issue_ids[0] duplicate"; got != want ||
			len(w.storedRelations(t, at)) != 2 || len(w.relationRows(t, near, at)) != 2 {
			t.Errorf("%s answers %s, with %d relations; want %s, and the two", tt.body, got, len(w.storedRelations(t, at)), want)
		}
	}
}

// Each work item a work item may not be related to is refused at its
// place, in one 422, nothing written (M4 design 3.4, 3.8, 3.10): Near
// itself, a deleted work item, one not there, an archived one, one of an
// archived project bob is a member of, one of a project of acme bob is no
// member of, one of another workspace; Up, which it may be related to,
// beside them, is not. The one of another workspace is refused to alice
// too, who sees it. Naming no work item, Up twice, or Up 21 times is
// refused, 422, nothing written. An archived work item, and one of an
// archived project, relate to none (409).
func TestWhatAWorkItemMayNotBeRelatedTo(t *testing.T) {
	w := newIssueWorld(t)
	near := w.newWorkItemIn(t, w.web, `{"name":"Near"}`)
	up := w.newWorkItemIn(t, w.web, `{"name":"Up"}`)
	gone := w.newWorkItemIn(t, w.web, `{"name":"Gone"}`)
	w.send(t, "alice", http.MethodDelete, "/api/v0/issues/"+gone.String(), "", http.StatusNoContent)
	shelved := w.doneWorkItem(t)
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+shelved.String()+"/archive", "", http.StatusOK)
	old := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Old", "OLD")
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+old.String()+"/members",
		`{"members":[{"member_id":"`+w.ids["bob"].String()+`","role":15}]}`, http.StatusCreated)
	olds := w.newWorkItemIn(t, old, `{"name":"Old's"}`)
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+old.String()+"/archive", "", http.StatusOK)
	labs := w.newWorkItemIn(t, createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Lab", "LAB"), `{"name":"Lab's"}`)
	w.send(t, "alice", http.MethodPost, "/api/v0/workspaces", `{"name":"Beta","slug":"beta"}`, http.StatusCreated)
	betas := w.newWorkItemIn(t, createdProject(t, w.contract, w.base, w.tokens["alice"], "beta", "Beta", "BETA"), `{"name":"Beta's"}`)

	answer := w.send(t, "bob", http.MethodPost, relationsOf(near), relating("relates_to", near, gone, uuid.NewV7(), shelved, olds, labs, betas,
		up), http.StatusUnprocessableEntity)
	var refused []string
	for _, field := range []string{"[0]", "[1]", "[2]", "[3]", "[4]", "[5]", "[6]"} {
		refused = append(refused, "issue_ids"+field+" not_allowed")
	}
	if got, want := refusalOf(t, []byte(answer), false), "validation_failed "+strings.Join(refused, "; "); got != want ||
		len(w.storedRelations(t, time.Time{})) != 0 || len(w.relationRows(t, up, time.Time{})) != 0 {
		t.Errorf("the work items Near may not be related to answer %s, with %d relations; want %s, and none", got,
			len(w.storedRelations(t, time.Time{})), want)
	}
	answer = w.send(t, "alice", http.MethodPost, relationsOf(near), relating("relates_to", betas), http.StatusUnprocessableEntity)
	if got := refusalOf(t, []byte(answer), false); got != "validation_failed issue_ids[0] not_allowed" ||
		len(w.storedRelations(t, time.Time{})) != 0 {
		t.Errorf("alice's relating Near to Beta's, of another workspace, answers %s, with %d relations; want issue_ids[0] not_allowed, and none",
			got, len(w.storedRelations(t, time.Time{})))
	}
	for body, refusal := range map[string]string{relating("relates_to"): "issue_ids too_short", relating("relates_to", up, up): "issue_ids[1] duplicate",
		relating("relates_to", slices.Repeat([]uuid.UUID{up}, 21)...): "issue_ids too_long"} {
		answer = w.send(t, "bob", http.MethodPost, relationsOf(near), body, http.StatusUnprocessableEntity)
		if got := refusalOf(t, []byte(answer), false); got != "validation_failed "+refusal || len(w.storedRelations(t, time.Time{})) != 0 {
			t.Errorf("%.60s answers %s, with %d relations; want %s, and none", body, got, len(w.storedRelations(t, time.Time{})), refusal)
		}
	}
	for _, tt := range []struct {
		item uuid.UUID
		code string
	}{{shelved, "issue.archived"}, {olds, "project.archived"}} {
		answer := w.send(t, "alice", http.MethodPost, relationsOf(tt.item), relating("relates_to", up), http.StatusConflict)
		if got := refusalOf(t, []byte(answer), false); got != tt.code || len(w.storedRelations(t, time.Time{})) != 0 {
			t.Errorf("relating %s answers %s, with %d relations; want %s, and none", tt.item, got, len(w.storedRelations(t, time.Time{})),
				tt.code)
		}
	}
}

// The guest rule holds for relations (M4 design 3.4, 3.10): walt, acme's
// admin and Web's guest (adminGuest), sees only his own work items in Web.
// He makes His and Mine, alice Hers between them; under Mine, alice makes
// Low and walt Below. walt relates His to Mine and Hers, which he does
// not see, refused at its place with the very answer of a work item not
// there: nothing written. alice relates Hers to His; then walt His to Mine
// alone, answered with the one sub-item of Mine he sees. Up, alice's,
// walt finds not to relate: the very 404 of a work item not there.
func TestTheGuestRuleHoldsForRelations(t *testing.T) {
	w := newIssueWorld(t)
	w.adminGuest(t)
	up := w.newWorkItemIn(t, w.web, `{"name":"Up"}`)
	his := w.newWorkItemBy(t, "walt", w.web, `{"name":"His"}`)
	hers := w.newWorkItemIn(t, w.web, `{"name":"Hers"}`)
	mine := w.newWorkItemBy(t, "walt", w.web, `{"name":"Mine"}`)
	w.newWorkItemIn(t, w.web, underParent("Low", mine))
	w.newWorkItemBy(t, "walt", w.web, underParent("Below", mine))
	answer := w.send(t, "walt", http.MethodPost, relationsOf(his), relating("relates_to", mine, hers), http.StatusUnprocessableEntity)
	notThere := w.send(t, "walt", http.MethodPost, relationsOf(his), relating("relates_to", mine, uuid.NewV7()),
		http.StatusUnprocessableEntity)
	if got := refusalOf(t, []byte(answer), false); got != "validation_failed issue_ids[1] not_allowed" || answer != notThere ||
		len(w.storedRelations(t, time.Time{})) != 0 {
		t.Errorf("walt's relating His to Mine and Hers answers %s, with %d relations; want issue_ids[1] not_allowed, as for one not there: %s",
			answer, len(w.storedRelations(t, time.Time{})), notThere)
	}
	w.send(t, "alice", http.MethodPost, relationsOf(hers), relating("duplicate", his), http.StatusCreated)
	made := listIn[answeredRelation](t, w.send(t, "walt", http.MethodPost, relationsOf(his), relating("relates_to", mine), http.StatusCreated))
	if got := counts(othersOf(made)); !slices.Equal(got, []string{"Mine 1"}) {
		t.Fatalf("walt's relating His to Mine answers relations to %q; want one, to Mine with the one sub-item he sees", got)
	}
	if got, want := w.storedRelations(t, made[0].CreatedAt), []string{"Hers duplicate His, in WEB, by alice, alice at another moment",
		"His relates_to Mine, in WEB, by walt, walt at the moment"}; !slices.Equal(got, want) {
		t.Errorf("the relations stored:\n%q\nwant\n%q", got, want)
	}
	hidden := w.send(t, "walt", http.MethodPost, relationsOf(up), relating("relates_to", mine), http.StatusNotFound)
	absent := w.send(t, "walt", http.MethodPost, relationsOf(uuid.NewV7()), relating("relates_to", mine), http.StatusNotFound)
	if hidden != absent {
		t.Errorf("walt's relating Up answers %s, want as for a work item not there: %s", hidden, absent)
	}
}
````

`server/internal/bootstrap/issue_write_locks_test.go`（修改，1 处）：

````old server/internal/bootstrap/issue_write_locks_test.go
		waits: "issues", mode: "SHARE"},
````
````new server/internal/bootstrap/issue_write_locks_test.go
		waits: "issues", mode: "SHARE"},
	// A new work item, which the send relates to Far: Web's one work item
	// named so, made with the first target. Its precheck reads no relation,
	// so it waits for the relations' table, which it reads first under
	// the lock.
	{op: "createIssueRelations", method: http.MethodPost, path: "/api/v0/issues/{issue_id}/relations",
		target: func(t *testing.T, w issueWorld) string {
			if len(queryIDs(t, w.pool, "SELECT id FROM issues WHERE project_id = $1 AND name = 'Far'", w.web)) == 0 {
				w.newWorkItemIn(t, w.web, `{"name":"Far"}`)
			}
			return relationsOf(w.newWorkItem(t))
		},
		body: func(t *testing.T, w issueWorld) string { return relating("relates_to", w.workItemNamed(t, "Far")) }, want: http.StatusCreated,
		waits: "issue_relations", mode: "ACCESS EXCLUSIVE"},
````

`server/internal/bootstrap/issue_writes_test.go`（修改，2 处）：

````old server/internal/bootstrap/issue_writes_test.go
// under Part. She archives the first, in Done, and unarchives it; she
// deletes the second, and Part and Low under it. With Web archived, her
// creation and her change are refused under the lock (409).
````
````new server/internal/bootstrap/issue_writes_test.go
// under Part. She relates the first to Far, of Ops, as blocking it, both
// projects locked at once. She archives the first, in Done, and
// unarchives it; she deletes the second, and Part and Low under it. With
// Web archived, her creation and her change are refused under the lock
// (409).
````
````old server/internal/bootstrap/issue_writes_test.go
	route.answerWithin(t, w.contract, http.MethodPost, subIssues(part.ID), alice, subIssueIDs(low.ID), http.StatusOK)
````
````new server/internal/bootstrap/issue_writes_test.go
	route.answerWithin(t, w.contract, http.MethodPost, subIssues(part.ID), alice, subIssueIDs(low.ID), http.StatusOK)
	far := w.newWorkItemIn(t, createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS"), `{"name":"Far"}`)
	route.answerWithin(t, w.contract, http.MethodPost, relationsOf(first.ID), alice, relating("blocking", far), http.StatusCreated)
````

`server/internal/bootstrap/permission_matrix_relations_test.go`（新文件，96 行）：

````file server/internal/bootstrap/permission_matrix_relations_test.go
package bootstrap

import (
	"net/http"
	"reflect"
	"testing"
	"uuid"
)

// The rows of the permission matrix of a work item's relations (M4 design
// 3.10, 9.2), naming the work item by its id: its column's work item
// (issueOf). A row of 9.2's table has the issue table's columns; a variant
// that asks what is checked after the decision, the project level's, as
// updateIssue's; one on the archived project, its table's.

// relatesTo is the request of a row whose callers each relate their
// column's work item to the one other gives, as blocking it.
func relatesTo(other func(c caller, s seeded) uuid.UUID) func(caller, seeded) (string, string, string) {
	return func(c caller, s seeded) (string, string, string) {
		return toWorkItem(http.MethodPost, "/relations", `{"relation_type":"blocking","issue_ids":["`+other(c, s).String()+`"]}`)(c, s)
	}
}

// beforesItem is the work item of another project that a column's cells
// relate their column's work item to: acme's private project's of the
// member before, which its admins and members see (PA, PM and PM+WA, who
// answer alike in both projects).
func beforesItem(_ caller, s seeded) uuid.UUID {
	return s.issue("acme/private", issueOf(callerBefore))
}

// archivedItem is the work item of acme's archived project, which may be
// related to none.
func archivedItem(_ caller, s seeded) uuid.UUID {
	return s.issue("acme/archived", matrixIssue)
}

func relationMatrixRows() []matrixRow {
	// The workspace admin who is only a guest of the project may relate his
	// own work item, as its admin (Grant.ProjectAdmin), but not to PM's,
	// which the guest rule hides from him: refused at its place (M4 design
	// 3.4).
	relates := onAWorkItem(cellCreated, cellCreated, cellCreated, cellForbidden)
	relates[callerAdminGuestOwn] = cellValidationFailed
	return []matrixRow{
		// The project's admins and members, the member who made it too, not
		// its guests, as updateIssue (M4 design 3.4, 9.2).
		{op: "createIssueRelations", write: true, columns: issueColumns, request: relatesTo(otherItemOf), cells: relates,
			refusal: "issue_ids[0] not_allowed", check: relatedTo(otherItemOf)},
		// An archived project's work items relate to none (M4 design 3.8),
		// after the decision.
		{op: "createIssueRelations", variant: "archived", write: true, columns: archivedColumns, request: relatesTo(otherItemOf),
			cells: issueRows.ofArchived(cellProjectArchived, cellForbidden)},
		// Nor do its archived work items: the project's 409 comes first.
		{op: "createIssueRelations", variant: "archived, its work item archived", write: true, columns: archivedColumns,
			prepare: inDone(true), request: relatesTo(otherItemOf), cells: issueRows.ofArchived(cellProjectArchived, cellForbidden)},
		// Nor does an archived work item, after the decision, before the work
		// items asked for are looked at.
		{op: "createIssueRelations", variant: "archived work item", write: true, columns: issueColumns, prepare: inDone(true),
			request: relatesTo(otherItemOf), cells: onAWorkItem(cellIssueArchived, cellIssueArchived, cellIssueArchived, cellForbidden)},
		// A work item of another project of the workspace, which the caller
		// sees: related (M4 design 3.10).
		{op: "createIssueRelations", variant: "another project's work item", write: true, columns: projectColumns,
			request: relatesTo(beforesItem), cells: issueRows.of(cellCreated, cellCreated, cellIssueNotFound, cellCreated, cellForbidden, cellForbidden),
			check: relatedTo(beforesItem)},
		// The archived project's work item, after the decision: refused at
		// its place.
		{op: "createIssueRelations", variant: "an archived project's work item", write: true, columns: projectColumns,
			request: relatesTo(archivedItem), refusal: "issue_ids[0] not_allowed",
			cells: issueRows.of(cellValidationFailed, cellValidationFailed, cellIssueNotFound, cellValidationFailed, cellForbidden, cellForbidden)},
	}
}

// relatedTo is the check of a row whose callers relate their column's work
// item to the one other gives (relatesTo): the answer is that relation
// alone, blocking it as the column's work item reads it, the other work
// item changed last by the caller.
func relatedTo(other func(c caller, s seeded) uuid.UUID) func(t *testing.T, c caller, s seeded, answer string) {
	return func(t *testing.T, c caller, s seeded, answer string) {
		var got struct {
			Data []struct {
				RelationType string `json:"relation_type"`
				Issue        struct {
					ID          uuid.UUID  `json:"id"`
					UpdatedByID *uuid.UUID `json:"updated_by_id"`
				} `json:"issue"`
			} `json:"data"`
		}
		decodeAnswer(t, answer, &got)
		me := s.account(accountOf(c))
		if len(got.Data) != 1 || got.Data[0].RelationType != "blocking" || got.Data[0].Issue.ID != other(c, s) ||
			!reflect.DeepEqual(got.Data[0].Issue.UpdatedByID, &me) {
			t.Errorf("%s relates %s; want blocking %s alone, changed last by the caller", c, answer, other(c, s))
		}
	}
}
````

`server/internal/bootstrap/permission_matrix_sub_issues_test.go`（修改，9 处）：

````old server/internal/bootstrap/permission_matrix_sub_issues_test.go
// subItemOf is the work item a column's cells put under their column's
// work item: acme's public project's work item of PM, the member who made
// it, or, for his own column, whose work item that is, the project's
// matrixIssue, acme's admin's. Only the public project's cells weigh it.
func subItemOf(c caller, s seeded) uuid.UUID {
````
````new server/internal/bootstrap/permission_matrix_sub_issues_test.go
// otherItemOf is the work item a column's cells put under their column's
// work item, or relate it to (relationMatrixRows): acme's public project's
// work item of PM, the member who made it, or, for his own column, whose
// work item that is, the project's matrixIssue, acme's admin's. Only the
// public project's cells weigh it.
func otherItemOf(c caller, s seeded) uuid.UUID {
````
````old server/internal/bootstrap/permission_matrix_sub_issues_test.go
// under their column's work item: subItemOf, or the one item gives.
````
````new server/internal/bootstrap/permission_matrix_sub_issues_test.go
// under their column's work item: otherItemOf, or the one item gives.
````
````old server/internal/bootstrap/permission_matrix_sub_issues_test.go
		{op: "addSubIssues", write: true, columns: issueColumns, request: addsSubItem(subItemOf), cells: adds,
````
````new server/internal/bootstrap/permission_matrix_sub_issues_test.go
		{op: "addSubIssues", write: true, columns: issueColumns, request: addsSubItem(otherItemOf), cells: adds,
````
````old server/internal/bootstrap/permission_matrix_sub_issues_test.go
		{op: "addSubIssues", variant: "archived", write: true, columns: archivedColumns, request: addsSubItem(subItemOf),
````
````new server/internal/bootstrap/permission_matrix_sub_issues_test.go
		{op: "addSubIssues", variant: "archived", write: true, columns: archivedColumns, request: addsSubItem(otherItemOf),
````
````old server/internal/bootstrap/permission_matrix_sub_issues_test.go
			request: addsSubItem(subItemOf), cells: issueRows.ofArchived(cellProjectArchived, cellForbidden)},
````
````new server/internal/bootstrap/permission_matrix_sub_issues_test.go
			request: addsSubItem(otherItemOf), cells: issueRows.ofArchived(cellProjectArchived, cellForbidden)},
````
````old server/internal/bootstrap/permission_matrix_sub_issues_test.go
			request: addsSubItem(subItemOf), cells: onAWorkItem(cellIssueArchived, cellIssueArchived, cellIssueArchived, cellForbidden)},
````
````new server/internal/bootstrap/permission_matrix_sub_issues_test.go
			request: addsSubItem(otherItemOf), cells: onAWorkItem(cellIssueArchived, cellIssueArchived, cellIssueArchived, cellForbidden)},
````
````old server/internal/bootstrap/permission_matrix_sub_issues_test.go
// putsUnder: the column's sub-item (subItemOf) alone, under the column's
// work item, changed last by the caller.
````
````new server/internal/bootstrap/permission_matrix_sub_issues_test.go
// putsUnder: the column's other work item (otherItemOf) alone, under the
// column's work item, changed last by the caller.
````
````old server/internal/bootstrap/permission_matrix_sub_issues_test.go
	if len(got.Data) != 1 || got.Data[0].ID != subItemOf(c, s) || !reflect.DeepEqual(got.Data[0].ParentID, &parent) ||
````
````new server/internal/bootstrap/permission_matrix_sub_issues_test.go
	if len(got.Data) != 1 || got.Data[0].ID != otherItemOf(c, s) || !reflect.DeepEqual(got.Data[0].ParentID, &parent) ||
````
````old server/internal/bootstrap/permission_matrix_sub_issues_test.go
		t.Errorf("%s puts %s; want %s alone, under %s, changed last by the caller", c, answer, subItemOf(c, s), parent)
````
````new server/internal/bootstrap/permission_matrix_sub_issues_test.go
		t.Errorf("%s puts %s; want %s alone, under %s, changed last by the caller", c, answer, otherItemOf(c, s), parent)
````

`server/internal/bootstrap/permission_matrix_test.go`（修改，1 处）：

````old server/internal/bootstrap/permission_matrix_test.go
		labelMatrixRows(), issueMatrixRows(), issueUpdateMatrixRows(), issueArchiveMatrixRows(), subIssueMatrixRows())
````
````new server/internal/bootstrap/permission_matrix_test.go
		labelMatrixRows(), issueMatrixRows(), issueUpdateMatrixRows(), issueArchiveMatrixRows(), subIssueMatrixRows(), relationMatrixRows())
````

- [ ] **Step 4: `furnishIssueRows` 建关联，暂时的豁免去掉**

`e2e/fixtures/assert/project.ts`（修改，3 处）：

````old e2e/fixtures/assert/project.ts
 * project_id. Each table but the relations' has such a row, and none is left undeleted; a row deleted before the
 * project, as a label a story deleted, is not counted.
````
````new e2e/fixtures/assert/project.ts
 * project_id. Each table has such a row, and none is left undeleted; a row deleted before the project, as a label a
 * story deleted, is not counted.
````
````old e2e/fixtures/assert/project.ts
  );
  // No story relates work items yet (M4 P3 spec, section 3): the relations' table has no row to be deleted with the
  // project, and none left undeleted either.
  const withoutRows = new Set(["issue_relations"]);
````
````new e2e/fixtures/assert/project.ts
  );
````
````old e2e/fixtures/assert/project.ts
    tables.map((name) => ({ table: name, deletedWithIt: !withoutRows.has(name), other: 0 }))
````
````new e2e/fixtures/assert/project.ts
    tables.map((name) => ({ table: name, deletedWithIt: true, other: 0 }))
````

`e2e/fixtures/assert/workspace.ts`（修改，4 处）：

````old e2e/fixtures/assert/workspace.ts
 * and their work items' rows. Each table but the relations' has such a row; none is left undeleted; each table of
 * deletedAlone but the relations', whose rows the story deleted alone, has rows deleted earlier, which kept their
 * moment, and every row of the others carries the workspace's.
````
````new e2e/fixtures/assert/workspace.ts
 * and their work items' rows. Each table has such a row; none is left undeleted; each table of deletedAlone, whose
 * rows the story deleted alone, has rows deleted earlier, which kept their moment, and every row of the others carries
 * the workspace's.
````
````old e2e/fixtures/assert/workspace.ts
  );
  // No story relates work items yet (M4 P3 spec, section 3): the relations' table has no row to be deleted with the
  // workspace or before it, and none left undeleted either.
  const withoutRows = new Set(["issue_relations"]);
````
````new e2e/fixtures/assert/workspace.ts
  );
````
````old e2e/fixtures/assert/workspace.ts
      deletedWithIt: !withoutRows.has(table),
````
````new e2e/fixtures/assert/workspace.ts
      deletedWithIt: true,
````
````old e2e/fixtures/assert/workspace.ts
      deletedEarlier: deletedAlone.includes(table) && !withoutRows.has(table),
````
````new e2e/fixtures/assert/workspace.ts
      deletedEarlier: deletedAlone.includes(table),
````

`e2e/fixtures/furnish.ts`（修改，4 处）：

````old e2e/fixtures/furnish.ts
import { createIssue } from "./issues";
````
````new e2e/fixtures/furnish.ts
import { createIssue, createIssueRelations } from "./issues";
````
````old e2e/fixtures/furnish.ts
 * and the assignee.
````
````new e2e/fixtures/furnish.ts
 * and the assignee; and a second work item, which the first blocks: their relation, and a row of each one's activity
 * on it.
````
````old e2e/fixtures/furnish.ts
  await createIssue(api, token, projectId, {
````
````new e2e/fixtures/furnish.ts
  const fix = await createIssue(api, token, projectId, {
````
````old e2e/fixtures/furnish.ts
  });
````
````new e2e/fixtures/furnish.ts
  });
  const test = await createIssue(api, token, projectId, { name: "Test the login" });
  await createIssueRelations(api, token, fix.id, { relation_type: "blocking", issue_ids: [test.id] });
````

`e2e/fixtures/issues.ts`（修改，1 处）：

````old e2e/fixtures/issues.ts
  return answer(response, error);
}

````
````new e2e/fixtures/issues.ts
  return answer(response, error);
}

/**
 * Relates the work item of issueId with the bearer token given, a member's of its project, to the work items body
 * names, as its type, and returns the relations made, as the work item reads them.
 */
export async function createIssueRelations(
  api: Api,
  token: string,
  issueId: string,
  body: components["schemas"]["IssueRelationsCreate"]
): Promise<components["schemas"]["IssueRelation"][]> {
  const { data, error, response } = await api.POST("/api/v0/issues/{issue_id}/relations", {
    params: { path: { issue_id: issueId } },
    body,
    headers: bearer(token),
  });
  expect(response.status, `relate the work item ${issueId}: ${JSON.stringify(error)}`).toBe(201);
  if (!data) {
    throw new Error(`createIssueRelations ${issueId} answered 201 without the relations`);
  }
  return data.data;
}

````

- [ ] **Step 5: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/issue/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestWorkItemsAreRelatedAcrossProjects$|TestWhatAWorkItemMayNotBeRelatedTo$|TestTheGuestRuleHoldsForRelations$|TestARelationLocksNoProjectItDoesNotReach$|TestEachIssueWriteLocksItsProjectFirst$|TestEachWriteOnAWorkItemOfAProjectTooBusyAnswersServerBusy$|TestAWriteAcrossProjectsWaitsMaxWaitInAll$|TestAWriteAcrossProjectsGivesBackEachPlaceItTook$|TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection$|TestPermissionMatrix|TestThePermissionMatrixCoversEveryOperation$' ./internal/bootstrap/`
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
Expected: 123 个全部通过（删除项目、工作区的故事删除 `furnishIssueRows` 建的关联）。

- [ ] **Step 6: 提交**

```bash
git add api/modules/issue.yaml api/openapi.yaml e2e/fixtures/assert/project.ts e2e/fixtures/assert/workspace.ts e2e/fixtures/furnish.ts e2e/fixtures/issues.ts server/internal/bootstrap/gate_across_test.go server/internal/bootstrap/gate_test.go server/internal/bootstrap/issue_relation_locks_test.go server/internal/bootstrap/issue_relations_test.go server/internal/bootstrap/issue_write_locks_test.go server/internal/bootstrap/issue_writes_test.go server/internal/bootstrap/permission_matrix_relations_test.go server/internal/bootstrap/permission_matrix_sub_issues_test.go server/internal/bootstrap/permission_matrix_test.go server/internal/modules/issue/adapter/http/handler.go server/internal/modules/issue/adapter/http/relations.go server/internal/modules/issue/adapter/http/relations_test.go server/internal/modules/issue/module.go api/dist/openapi.yaml server/internal/modules/issue/adapter/http/gen/bodyshape.gen.go server/internal/modules/issue/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M4/P3): createIssueRelations through the API, the matrix, and the e2e furnishing

POST /api/v0/issues/{issue_id}/relations answers 201 with the relations
as the work item reads them, each other work item an Issue row. A
relation to a work item the caller may not reach takes no lock of its
project; the write enters the gate of each project it reaches, waits
max_wait at them in all, and gives back each place it took. The matrix
gains its rows. furnishIssueRows relates its two work items, so the e2e
deletions of projects and workspaces ask for relation rows again.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p3.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试，或没有替身的平台、领域、转换的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；守卫是生成器（`make gen-go`）和中英文翻译键（`make lint-web`）的核对。每个变异在一份新的副本里，在本 Task 的树上、标出的较晚的 Task 的树上和最终的树上都跑过，端到端的检查只在标出的 Task 的树上）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T11.1` | `POST` 交给用例的工作项是空的 | `internal/modules/issue/adapter/http` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWorkItemsAreRelatedAcrossProjects`（`make test`） | 单元；组合 |
| `T11.2` | 回答的每条关联的类型是对方读到的 | `internal/modules/issue/adapter/http` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWorkItemsAreRelatedAcrossProjects`（`make test`）、故事 `i9-relations`（Task 15 起） | 单元；组合；端到端 |
| `T11.3` | 回答的关联的时刻是对方工作项建立的时刻 | `internal/modules/issue/adapter/http` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWorkItemsAreRelatedAcrossProjects`（`make test`） | 单元；组合 |
| `T11.4` | 契约没有写 `createIssueRelations` 回答 `issue.archived` | `internal/modules/issue/adapter/http` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWhatAWorkItemMayNotBeRelatedTo`（`make test`） | 单元；组合 |
| `T11.5` | `createIssueRelations` 的请求体收它没有列出的字段 | `internal/modules/issue/adapter/http` 的全部测试（`make test`） | 单元 |

本 Task 的检查也发现较早的 Task 的变异 `T7.1`、`T7.2`、`T7.3`、`T7.6`、`T7.7`、`T7.8`、`T7.9`、`T9.1`、`T9.2`、`T9.3`、`T9.5`、`T9.6`、`T9.7`、`T9.8`、`T9.9`、`T9.15`、`T10.1`、`T10.2`、`T10.3`、`T10.5`、`T10.6`、`T10.7`、`T10.8`、`T10.9`、`T10.10`、`T10.11`、`T10.12`、`T10.13`、`T10.14`、`T10.15`、`T10.16`、`T10.17`、`T10.18`、`T10.21`、`T10.22`（列在各自的 Task 的表里，标"（Task 11 起）"）：本 Task 做完之后，在本 Task 的树上重跑它们（`mutants_p3.py`，`MUT_LATER=11`）。

---

### Task 12: `listIssueRelations`：对方看得到、没归档、不在别的已归档的项目里

**Files:**
- Create: `server/internal/bootstrap/issue_archived_projects_test.go`、`server/internal/bootstrap/issue_relation_lists_test.go`、`server/internal/bootstrap/permission_matrix_seed_relations_test.go`、`server/internal/modules/issue/app/list_relations.go`、`server/internal/modules/issue/app/list_relations_test.go`
- Modify: `api/modules/issue.yaml`、`server/internal/bootstrap/issue_relations_test.go`、`server/internal/bootstrap/permission_matrix_relations_test.go`、`server/internal/bootstrap/permission_matrix_seeded_test.go`、`server/internal/bootstrap/permission_matrix_test.go`、`server/internal/bootstrap/ports.go`、`server/internal/modules/issue/adapter/http/handler.go`、`server/internal/modules/issue/adapter/http/relations.go`、`server/internal/modules/issue/adapter/http/relations_test.go`、`server/internal/modules/issue/adapter/postgres/queries/relations.sql`、`server/internal/modules/issue/adapter/postgres/relations.go`、`server/internal/modules/issue/adapter/postgres/relations_test.go`、`server/internal/modules/issue/app/fakes_across_test.go`、`server/internal/modules/issue/app/fakes_relations_test.go`、`server/internal/modules/issue/app/lock_across.go`、`server/internal/modules/issue/app/ports.go`、`server/internal/modules/issue/app/seen.go`、`server/internal/modules/issue/module.go`、`server/internal/modules/project/adapter/postgres/failures_test.go`、`server/internal/modules/project/adapter/postgres/issue_ports.go`、`server/internal/modules/project/adapter/postgres/issue_ports_test.go`、`server/internal/modules/project/adapter/postgres/queries/issue_ports.sql`、`server/internal/modules/project/module.go`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/issue/adapter/http/gen/server.gen.go`、`server/internal/modules/issue/adapter/postgres/gen/relations.sql.go`、`server/internal/modules/project/adapter/postgres/gen/issue_ports.sql.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**（spec 2.14、第 3 节第 9、10、15、17 条；M4 设计 3.4、3.8、3.10、5.1、5.2、6.5、9.2）
- 契约：`GET /api/v0/issues/{issue_id}/relations`（`listIssueRelations`，200 `IssueRelationCollection`，`x-problem-codes: [issue.not_found, forbidden]`）；描述写上：与 `getIssue` 同样的读和拒绝；未删除的关联按建的次序，各按这个工作项读的类型；对方未删除、没归档、不在这个工作项的项目之外的已归档的项目里、调用者看得到（任何项目的，访客规则在内），按他在那个项目看到的样子。
- `project`：`queries/issue_ports.sql` 的 `ArchivedIssueProjects`（给的项目中未删除、已归档的）；`issue_ports.go`；`module.go` 的 `IssueProjects.ArchivedIssueProjects`。
- `issue/app`：`ports.go` 的 `RelationLister`（`Relations`、`GetIssues`（同 `IssueReader` 的））、`ArchivedProjects`（`Archived(ctx, projectIDs) (map[uuid.UUID]bool, error)`，嵌入 `ProjectPorts`）；`seen.go` 的 `Locks.seeEach(ctx, actor, items)`（不加锁：每个项目以 `issue.read` 判定一次（列表在它的项目上的判定，`issue.read` 不认创建者），再对每个工作项执行访客规则；看不到的项目、读不了的、访客规则隐藏的不在其中；交回各项目的 `Grant`）；`lock_across.go` 的 `ascending`（升序、各一次：`reachAll` 原来就地排序去重，`projectsBut` 是第二个使用者）；`list_relations.go`（新）：`ListIssueRelations`、`NewListIssueRelations(locks, relations, projects)`、`Execute(ctx, id)`：`Locks.see`（`issue.read`），读关联，读对方（`GetIssues`），问它们的项目（这个工作项的项目除外，`projectsBut`）是否已归档，去掉已归档的和在已归档的别的项目里的，`seeEach`，按 `Grant` 数（`seen`），照关联的次序回答。
- `issue/adapter/postgres`：`queries/relations.sql` 的 `IssueRelations`（两个方向、未删除、如存的、`created_at, id` 的次序）；`relations.go` 的 `Relations`、`relationOf`。
- `issue/adapter/http`：`handler.go` 的 `ListIssueRelationsUseCase`；`relations.go` 的 `ListIssueRelations`；`relations_test.go` 的拒绝合成 `TestIssueRelationsRefusals`（两个操作）。`issue/module.go` 接上（`d.Projects` 是它的 `ArchivedProjects`）。
- `bootstrap`：`ports.go` 的 `issueProjects.Archived`；`issue_archived_projects_test.go`（新）；`issue_relation_lists_test.go`（新）；`issue_relations_test.go` 的 `TestTheGuestRuleHoldsForRelations` 加列表；矩阵：`permission_matrix_seed_relations_test.go`（新：`seededRelation`、`seededRelations` 的五条：Blocked、Guest's、Member's、Admin guest's、Secret，各是关联的列表的一种情形；`relationsOf`、`relations`、`seededRelationRows`），`permission_matrix_seeded_test.go` 的 `seeded.relations`、`relation`，`permission_matrix_test.go` 的 `prepareMatrix` 种下、读回它们；`permission_matrix_relations_test.go` 的 `listIssueRelations` 两行（`relationsListed`、`listsRelations`）。
- 测试的帮手：`fakes_relations_test.go` 的 `Relations`；`fakes_across_test.go` 的 `Archived`（从 `across` 和锁住的项目回答）。

**Tests:**
- 单元：`TestListIssueRelationsListsWhatTheCallerSees`（`listed` 的调用；mine（它被它阻塞，按 alice 在 ops 的样子数：只数她自己的子工作项）、near（它阻塞的，在 web：web 已归档，但它是这个工作项自己的项目）；已归档的 shelved、别的已归档的项目的 boxed、ops 里访客规则隐藏的 far、她看不到的项目的 hidden 都不在其中）、`TestListIssueRelationsRefusals`（与 `getIssue` 同样的拒绝，在读关联之前；读、判定、数、项目的读失败；没有关联的回答空的）。
- 存储：`TestRelations`（两个方向、如存的、`(created_at, id)` 的次序；已删除的、别的工作项的不在；取消的读是它的失败）；`project` 的 `TestArchivedIssueProjects`（所问的未删除、已归档的；没问的、已删除的、没有的不在）、`TestAFailedReadIsAnErrorNotAnAnswer` 加一行。
- HTTP：`TestListIssueRelations`、`TestIssueRelationsRefusals`。
- 组合：`TestIssueProjectsConvertsArchivedProjects`；`TestARelationListShowsWhatTheCallerSees`（alice 列出 Near 的 Far、Mid（阻塞它）、Lab 的，按建的次序、各按 Near 读的类型；看不到 Lab 的 bob 只列出 Far、Mid；Far 列出 Near、Beside，Mid 列出 Near（它阻塞的）；已归档的 Shelved、已删除的 Gone 不在，Shelved 恢复之后又在；Ops 归档之后，Far（不是 Near 的项目的）不在 Near 的列表里，Far 自己的列表照样列出 Near、Beside（Far 自己的项目的）；只看自己的 walt（acme 的管理员、Web 的访客）把他的 His 关联到他的 Other，alice 把 Other 放到她的 Up 下：walt 列出的这条关联与 alice 列出的整个相同，只是内嵌的 Other 的 `parent_id` 为 `null`（Up 对他隐藏，Task 4 的 `seen`））；`TestTheGuestRuleHoldsForRelations`（walt 列出 His 的关联：只有 Mine，数 1 个子工作项，虽然排在前面的是同在 Web 的 Hers（每个项目判定一次，不以第一个工作项判定）；alice 列出 Hers、Mine，Mine 数 2 个）；`TestPermissionMatrix`（两行；种子的关联在每个格子的种子里，读回核对）、`TestThePermissionMatrixCoversEveryOperation`。

- [ ] **Step 1: 契约和查询，生成**

`api/modules/issue.yaml`（修改，1 处）：

````old api/modules/issue.yaml
  /api/v0/issues/{issue_id}/relations:
    parameters:
      - $ref: '#/components/parameters/IssueID'
    post:
````
````new api/modules/issue.yaml
  /api/v0/issues/{issue_id}/relations:
    parameters:
      - $ref: '#/components/parameters/IssueID'
    get:
      operationId: listIssueRelations
      tags: [issue]
      summary: List a work item's relations
      description: >-
        For the admins, members and guests of its project, as getIssue,
        which answers the same refusals: issue.not_found, forbidden. Its
        relations, all of them, without pages, in the order they were made,
        the first first, each as this work item reads it, whose other work
        item the caller sees, as getIssue would answer it, of any project of
        the workspace, and is not archived, nor of an archived project other
        than this work item's: one left out so shows again once it, or its
        project, is unarchived. An archived work item's are listed too, as
        are those of a work item of an archived project.
      security: [{bearer: []}]
      x-problem-codes: [issue.not_found, forbidden]
      responses:
        '200':
          description: The relations, as this work item reads them.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IssueRelationCollection'
        default:
          $ref: '#/components/responses/Problem'
    post:
````

`server/internal/modules/issue/adapter/postgres/queries/relations.sql`（修改，1 处）：

````old server/internal/modules/issue/adapter/postgres/queries/relations.sql
WHERE id = ANY (sqlc.arg(ids)::uuid[]) AND deleted_at IS NULL;

````
````new server/internal/modules/issue/adapter/postgres/queries/relations.sql
WHERE id = ANY (sqlc.arg(ids)::uuid[]) AND deleted_at IS NULL;

-- name: IssueRelations :many
-- The undeleted relations of the work item id, either way, each as stored, in the order they were made, (created_at,
-- id) (M4 design 3.10): listIssueRelations', read without a lock. The indexes on issue_id and on related_issue_id
-- serve it.
SELECT r.id, r.workspace_id, r.project_id, r.issue_id, r.related_issue_id, r.relation_type, r.created_by_id, r.created_at
FROM issue_relations r
WHERE (r.issue_id = sqlc.arg(id)::uuid OR r.related_issue_id = sqlc.arg(id)::uuid) AND r.deleted_at IS NULL
ORDER BY r.created_at, r.id;

````

`server/internal/modules/project/adapter/postgres/queries/issue_ports.sql`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/queries/issue_ports.sql
ORDER BY id;

-- name: ActiveRoles :many
````
````new server/internal/modules/project/adapter/postgres/queries/issue_ports.sql
ORDER BY id;

-- name: ArchivedIssueProjects :many
-- ArchivedProjects (M4 design 3.8): those among ids that are undeleted, archived projects, read without a lock. A list
-- across projects leaves out the work items of an archived project: listIssueRelations, the other work items of a
-- project other than the listed one's.
SELECT id
FROM projects
WHERE id = ANY (sqlc.arg(ids)::uuid[]) AND deleted_at IS NULL AND archived_at IS NOT NULL;

-- name: ActiveRoles :many
````

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `7d7e49381afa3793896cc968e81b9f12356080242a0d66ccbcd05f32941149ab` | 3642 | `api/dist/openapi.yaml` |
| `8ff9b9d60519ee15ee6a752cec5817836bae6102266ce3ceaa30d673938e64c5` | 1676 | `server/internal/modules/issue/adapter/http/gen/server.gen.go` |
| `764615799b0f8cacb9b0ca150ee257a148956f7552a77697c11150c1a7120ba4` | 160 | `server/internal/modules/issue/adapter/postgres/gen/relations.sql.go` |
| `7f5da04e66ed0249457420d4880656613ab47388c59032e4efe0f567a5f68427` | 288 | `server/internal/modules/project/adapter/postgres/gen/issue_ports.sql.go` |
| `425b1be2279860371e177a03113ceb34d59dab3e660efeb879e2c74edbe7acfa` | 3966 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/issue/adapter/http/gen/server.gen.go server/internal/modules/issue/adapter/postgres/gen/relations.sql.go server/internal/modules/project/adapter/postgres/gen/issue_ports.sql.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 2: `project` 回答已归档的项目**

`server/internal/modules/project/adapter/postgres/failures_test.go`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/failures_test.go
		t.Errorf("ProjectIdentifiers() = %+v, %v; want context.Canceled, not none", list, err)
	}
````
````new server/internal/modules/project/adapter/postgres/failures_test.go
		t.Errorf("ProjectIdentifiers() = %+v, %v; want context.Canceled, not none", list, err)
	}
	if archived, err := s.ArchivedIssueProjects(cancelled, []uuid.UUID{web}); !failed(err) || archived != nil {
		t.Errorf("ArchivedIssueProjects() = %v, %v; want context.Canceled, not none", archived, err)
	}
````

`server/internal/modules/project/adapter/postgres/issue_ports.go`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/issue_ports.go
}

// ActiveRoles reads the project roles of those of the accounts ids that
````
````new server/internal/modules/project/adapter/postgres/issue_ports.go
}

// ArchivedIssueProjects reads those of ids that are undeleted, archived
// projects, each true, without a lock (M4 design 3.8).
func (s *Store) ArchivedIssueProjects(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]bool, error) {
	rows, err := s.queries(ctx).ArchivedIssueProjects(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("read which of projects %v are archived: %w", ids, err)
	}
	archived := make(map[uuid.UUID]bool, len(rows))
	for _, id := range rows {
		archived[id] = true
	}
	return archived, nil
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

// Which projects are archived, for the issue module's lists across
// projects (M4 design 3.8): those asked for that are undeleted and
// archived, Arch; not Web, nor Gone, archived and deleted, nor one that is
// not there, nor Old, archived but not asked for.
func TestArchivedIssueProjects(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, arch, gone, old := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Arch", "ARCH", alice), newProject(t, s, acme,
		"Gone", "GONE", alice), newProject(t, s, acme, "Old", "OLD", alice)
	for _, p := range []uuid.UUID{arch, gone, old} {
		exec(t, pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", p, now)
	}
	exec(t, pool, "UPDATE projects SET deleted_at = $2 WHERE id = $1", gone, now)
	got, err := s.ArchivedIssueProjects(context.Background(), []uuid.UUID{web, arch, gone, uuid.NewV7()})
	if want := map[uuid.UUID]bool{arch: true}; err != nil || !maps.Equal(got, want) {
		t.Errorf("ArchivedIssueProjects() = %v, %v; want Arch (%s) alone", got, err, arch)
	}
}

// The catalogs of a project for the issue module (M4 design 6.5): its
````

`server/internal/modules/project/module.go`（修改，3 处）：

````old server/internal/modules/project/module.go
// (StateCatalog, LabelCatalog, ProjectMembers), for its writes; and a
````
````new server/internal/modules/project/module.go
// (StateCatalog, LabelCatalog, ProjectMembers), for its writes; a
````
````old server/internal/modules/project/module.go
// identifier. Each reads, or locks, in the transaction ctx carries; the
// catalogs are read under the project's lock, which every write of a
// state, a label or a membership holds too, so they stay as read until the
// write commits (M4 design 3.5).
````
````new server/internal/modules/project/module.go
// identifier; and which projects are archived (ArchivedProjects), for a
// list across projects. Each reads, or locks, in the transaction ctx
// carries; the catalogs are read under the project's lock, which every
// write of a state, a label or a membership holds too, so they stay as
// read until the write commits (M4 design 3.5).
````
````old server/internal/modules/project/module.go
	ProjectIdentifiers(ctx context.Context, workspaceID uuid.UUID) ([]IssueProjectIdentifier, error)
````
````new server/internal/modules/project/module.go
	ProjectIdentifiers(ctx context.Context, workspaceID uuid.UUID) ([]IssueProjectIdentifier, error)
	// ArchivedIssueProjects reads those of ids that are undeleted,
	// archived projects, each true, without a lock.
	ArchivedIssueProjects(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]bool, error)
````

- [ ] **Step 3: 用例**

`server/internal/modules/issue/app/fakes_across_test.go`（修改，1 处）：

````old server/internal/modules/issue/app/fakes_across_test.go
	return locked, nil
}

````
````new server/internal/modules/issue/app/fakes_across_test.go
	return locked, nil
}

// Archived logs the read, each project its own argument, and answers
// those of ids the fake holds archived, each true: its project, when Lock
// answers it archived, and those of across that are (fakeProject).
func (f *fakeProject) Archived(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]bool, error) {
	if err := f.log.add(ctx, "Archived", anys(ids)...); err != nil {
		return nil, err
	}
	archived := map[uuid.UUID]bool{}
	for _, id := range ids {
		if f.across[id].Archived || id == f.id && f.locked != nil && f.locked.Archived {
			archived[id] = true
		}
	}
	return archived, nil
}

````

`server/internal/modules/issue/app/fakes_relations_test.go`（修改，2 处）：

````old server/internal/modules/issue/app/fakes_relations_test.go
// (app.RelationCreator), each call logged: related are the work items
// that relate to the work item asked about already, whichever it is;
// relations the last insert's relations, marked the last work items
// marked changed.
````
````new server/internal/modules/issue/app/fakes_relations_test.go
// (app.RelationCreator, app.RelationLister), each call logged: related
// are the work items that relate to the work item asked about already,
// whichever it is; relations the relations a read answers, and the last
// insert's; marked the last work items marked changed.
````
````old server/internal/modules/issue/app/fakes_relations_test.go
	marked    *app.ChangedRows
````
````new server/internal/modules/issue/app/fakes_relations_test.go
	marked    *app.ChangedRows
}

// Relations logs the read and answers the fake's relations, as they are,
// whichever work item is asked about.
func (f *fakeRelations) Relations(ctx context.Context, id uuid.UUID) ([]domain.Relation, error) {
	if err := f.log.add(ctx, "Relations", id); err != nil {
		return nil, err
	}
	return f.relations, nil
````

`server/internal/modules/issue/app/list_relations.go`（新文件，90 行）：

````file server/internal/modules/issue/app/list_relations.go
package app

import (
	"context"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListIssueRelations lists a work item's relations: GET
// /api/v0/issues/{issue_id}/relations (M4 design 3.10, 5.1).
type ListIssueRelations struct {
	locks     Locks
	relations RelationLister
	projects  ArchivedProjects
}

// NewListIssueRelations returns the use case.
func NewListIssueRelations(locks Locks, relations RelationLister, projects ArchivedProjects) *ListIssueRelations {
	return &ListIssueRelations{locks: locks, relations: relations, projects: projects}
}

// Execute reads the work item id and decides issue.read on it for the
// caller, as getIssue does, without a lock or a transaction (Locks.see),
// with its refusals; then its undeleted relations, in the order they were
// made, each as the work item reads it, whose other work item is
// undeleted, unarchived, of no archived project but the work item's own
// (M4 design 3.8), and seen by the caller, of any project of the
// workspace (Locks.seeEach); each other as he sees it (Locks.seen).
func (u *ListIssueRelations) Execute(ctx context.Context, id uuid.UUID) ([]IssueRelation, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	i, _, err := u.locks.see(ctx, actor, id, domain.ActionRead)
	if err != nil {
		return nil, err
	}
	relations, err := u.relations.Relations(ctx, i.ID)
	if err != nil || len(relations) == 0 {
		return nil, err
	}
	ids := make([]uuid.UUID, len(relations))
	for n, r := range relations {
		ids[n] = r.Other(i.ID)
	}
	others, err := u.relations.GetIssues(ctx, ids)
	if err != nil {
		return nil, err
	}
	archived, err := u.projects.Archived(ctx, projectsBut(i.ProjectID, others))
	if err != nil {
		return nil, err
	}
	others = slices.DeleteFunc(others, func(o domain.Issue) bool { return o.ArchivedAt != nil || archived[o.ProjectID] })
	visible, g, err := u.locks.seeEach(ctx, actor, others)
	if err != nil {
		return nil, err
	}
	seen, err := u.locks.seen(ctx, actor.UserID, g, visible...)
	if err != nil {
		return nil, err
	}
	byID := make(map[uuid.UUID]domain.Issue, len(seen))
	for _, o := range seen {
		byID[o.ID] = o
	}
	var out []IssueRelation
	for _, r := range relations {
		if o, ok := byID[r.Other(i.ID)]; ok {
			out = append(out, IssueRelation{ID: r.ID, Type: r.TypeFrom(i.ID), Issue: o, CreatedAt: r.CreatedAt})
		}
	}
	return out, nil
}

// projectsBut are the projects of items but project, each once, in
// ascending id order: those whose work items a list of the relations of
// project's work item leaves out when they are archived (M4 design 3.8).
func projectsBut(project uuid.UUID, items []domain.Issue) []uuid.UUID {
	var ids []uuid.UUID
	for _, i := range items {
		if i.ProjectID != project {
			ids = append(ids, i.ProjectID)
		}
	}
	return ascending(ids)
}
````

`server/internal/modules/issue/app/list_relations_test.go`（新文件，159 行）：

````file server/internal/modules/issue/app/list_relations_test.go
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

// listing is newRelate's world with more work items: shelved, bob's,
// archived, of web; mine, alice's, of ops; hidden, carol's, of lab, a
// project alice does not see; boxed, alice's, of attic, an archived
// project. web, the work item's own project, is archived too. ops shows
// alice, its guest, only her own work items (Grant.OwnOnly): mine, not
// carol's far. relations are the work item's relations as the store
// answers them, in the order they were made, which is not the others' id
// order: far's duplicate of it; its blocked_by mine; its relates_to
// shelved; near's blocked_by it; hidden's relates_to it; boxed's
// duplicate of it.
type listing struct {
	relating
	shelved, mine, hidden, boxed domain.Issue
	lab, attic                   uuid.UUID
	relations                    []domain.Relation
}

// newListing is listIssueRelations over newListing's world, web's work
// item 7, carol's, its work item.
func newListing() (*app.ListIssueRelations, *world, *fakeRelations, uuid.UUID, listing) {
	_, w, rel, id, r := newRelate()
	l := listing{relating: r, lab: uuid.NewV7(), attic: uuid.NewV7()}
	l.shelved = domain.Issue{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web, SequenceID: 9, Name: "Shelved", CreatedByID: &bob,
		ArchivedAt: &earlier}
	l.mine = domain.Issue{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: ops, SequenceID: 4, Name: "Mine", CreatedByID: &alice}
	l.hidden = domain.Issue{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: l.lab, SequenceID: 1, Name: "Hidden", CreatedByID: &carol}
	l.boxed = domain.Issue{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: l.attic, SequenceID: 2, Name: "Boxed", CreatedByID: &alice}
	for _, o := range []domain.Issue{l.shelved, l.mine, l.hidden, l.boxed} {
		w.issues.others[o.ID] = o
	}
	w.project.locked.Archived = true
	w.project.across[l.attic] = app.LockedProject{WorkspaceID: acme, Archived: true, Identifier: "ATTIC"}
	made := func(n int, p, from, to uuid.UUID, t domain.RelationType) domain.Relation {
		return domain.Relation{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: p, IssueID: from, RelatedID: to, Type: t, CreatedByID: &bob,
			CreatedAt: earlier.Add(time.Duration(n) * time.Minute)}
	}
	l.relations = []domain.Relation{made(0, ops, r.far.ID, id, domain.Duplicate), made(1, web, id, l.mine.ID, domain.BlockedBy),
		made(2, web, id, l.shelved.ID, domain.RelatesTo), made(3, web, r.near.ID, id, domain.BlockedBy),
		made(4, l.lab, l.hidden.ID, id, domain.RelatesTo), made(5, l.attic, l.boxed.ID, id, domain.Duplicate)}
	rel.relations = l.relations
	// The decisions: the work item's, then near's project, far's and lab's,
	// as the others are read, in id order, each project once.
	w.auth.answers, w.auth.grants = []error{3: shared.ErrNotVisible}, []shared.Grant{2: ownOnly}
	w.issues.counts = map[uuid.UUID]app.SubIssueCount{r.near.ID: {All: 2, Own: 1}, l.mine.ID: {All: 4, Own: 3}}
	return app.NewListIssueRelations(w.locks, rel, w.project), w, rel, id, l
}

// listed are the calls of alice's list of the work item id's relations,
// outside any transaction and without a lock (M4 design 3.4, 3.8, 3.10):
// the work item read and issue.read decided on it, as getIssue does; its
// relations; their other work items, in the relations' order; which of
// the others' projects but web, the work item's own, are archived, in id
// order: ops, lab and attic, of which attic is; issue.read decided on the
// others' projects, each once, in the others' id order, shelved, archived,
// and boxed, of attic, left out: web, ops, lab; then the others she sees
// counted.
func listed(id uuid.UUID, l listing) []string {
	calls := []string{line("GetIssue", id), line("Authorize", alice, domain.ActionRead, acme, web), line("Relations", id),
		line("GetIssues", []uuid.UUID{l.far.ID, l.mine.ID, l.shelved.ID, l.near.ID, l.hidden.ID, l.boxed.ID}),
		line("Archived", anys(sorted([]uuid.UUID{ops, l.lab, l.attic}))...)}
	for _, p := range []uuid.UUID{web, ops, l.lab} {
		calls = append(calls, line("Authorize", alice, domain.ActionRead, acme, p))
	}
	calls = append(calls, counted(l.near.ID, l.mine.ID))
	for n := range calls {
		calls[n] += " outside tx"
	}
	return calls
}

// A list of relations (M4 design 3.4, 3.8, 3.10), with listed's calls,
// answers the relations whose other work item alice sees, in the order
// they were made, each as the work item reads it, each other as she sees
// it in its project: mine, which it is blocked by, counted as she sees it
// in ops, her own sub-items; then near, which it blocks, every one of its
// sub-items in web, which is archived but the work item's own. Shelved,
// archived, is not shown, nor boxed, of another archived project, nor
// far, which the guest rule hides from her in ops, nor hidden, of a
// project she does not see.
func TestListIssueRelationsListsWhatTheCallerSees(t *testing.T) {
	uc, w, _, id, l := newListing()
	got, err := uc.Execute(as(alice), id)
	mine, near := l.mine, l.near
	mine.SubIssuesCount, near.SubIssuesCount = 3, 2
	want := []app.IssueRelation{{ID: l.relations[1].ID, Type: domain.BlockedBy, Issue: mine, CreatedAt: l.relations[1].CreatedAt},
		{ID: l.relations[3].ID, Type: domain.Blocking, Issue: near, CreatedAt: l.relations[3].CreatedAt}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("Execute() = %+v, %v; want %+v", got, err, want)
	}
	if calls := listed(id, l); !slices.Equal(w.log.calls, calls) {
		t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
	}
}

// What refuses a list of relations is what refuses getIssue, before any
// relation is read: a work item not there, or whose project the caller
// does not see, or that the guest rule hides from him, issue.not_found; a
// role that may not read it, forbidden. A read, a decision or a count
// that fails ends it with its failure; a work item without relations has
// none.
func TestListIssueRelationsRefusals(t *testing.T) {
	for _, tt := range []struct {
		name  string
		set   func(w *world, rel *fakeRelations, id uuid.UUID) uuid.UUID // the work item asked for
		want  error
		calls int // of listed's
	}{
		{"no work item", func(*world, *fakeRelations, uuid.UUID) uuid.UUID { return uuid.NewV7() }, domain.ErrIssueNotFound, 1},
		{"a project not seen", func(w *world, _ *fakeRelations, id uuid.UUID) uuid.UUID {
			w.auth.answers = []error{shared.ErrNotVisible}
			return id
		}, domain.ErrIssueNotFound, 2},
		{"another's, to a guest who sees only his own", func(w *world, _ *fakeRelations, id uuid.UUID) uuid.UUID {
			w.auth.grants = []shared.Grant{ownOnly}
			return id
		}, domain.ErrIssueNotFound, 2},
		{"a role not allowed", func(w *world, _ *fakeRelations, id uuid.UUID) uuid.UUID {
			w.auth.answers = []error{shared.Forbidden()}
			return id
		}, shared.Forbidden(), 2},
		{"the relations' read fails", func(w *world, _ *fakeRelations, id uuid.UUID) uuid.UUID { w.log.fail = "Relations"; return id }, errDisk, 3},
		{"the others' read fails", func(w *world, _ *fakeRelations, id uuid.UUID) uuid.UUID { w.log.fail = "GetIssues"; return id }, errDisk, 4},
		{"the projects' read fails", func(w *world, _ *fakeRelations, id uuid.UUID) uuid.UUID { w.log.fail = "Archived"; return id }, errDisk, 5},
		{"a decision on another project fails", func(w *world, _ *fakeRelations, id uuid.UUID) uuid.UUID {
			w.log.fail = line("Authorize", alice, domain.ActionRead, acme, ops)
			return id
		}, errDisk, 7},
		{"the count fails", func(w *world, _ *fakeRelations, id uuid.UUID) uuid.UUID { w.log.fail = "SubIssueCounts"; return id }, errDisk, 9},
		{"no relations", func(_ *world, rel *fakeRelations, id uuid.UUID) uuid.UUID { rel.relations = nil; return id }, nil, 3},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w, rel, id, l := newListing()
			asked := tt.set(w, rel, id)
			got, err := uc.Execute(as(alice), asked)
			if !reflect.DeepEqual(err, tt.want) || len(got) != 0 {
				t.Errorf("Execute() = %+v, %v; want none and %v", got, err, tt.want)
			}
			calls := listed(id, l)[:tt.calls]
			calls[0] = line("GetIssue", asked, "outside tx")
			if !slices.Equal(w.log.calls, calls) {
				t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
			}
		})
	}
}
````

`server/internal/modules/issue/app/lock_across.go`（修改，2 处）：

````old server/internal/modules/issue/app/lock_across.go
}

// lockProjects locks s's projects in the transaction ctx carries and
````
````new server/internal/modules/issue/app/lock_across.go
}

// ascending is ids in ascending order, each once.
func ascending(ids []uuid.UUID) []uuid.UUID {
	return slices.Compact(slices.SortedFunc(slices.Values(ids), uuid.UUID.Compare))
}

// lockProjects locks s's projects in the transaction ctx carries and
````
````old server/internal/modules/issue/app/lock_across.go
	slices.SortFunc(s.projects, uuid.UUID.Compare)
	s.projects = slices.Compact(s.projects)
````
````new server/internal/modules/issue/app/lock_across.go
	s.projects = ascending(s.projects)
````

`server/internal/modules/issue/app/ports.go`（修改，2 处）：

````old server/internal/modules/issue/app/ports.go
}

// IssueArchiver is the work items' store as archiveIssue and
````
````new server/internal/modules/issue/app/ports.go
}

// RelationLister is the relations' store as listIssueRelations reads it,
// without a lock (M4 design 3.10).
type RelationLister interface {
	// Relations reads the undeleted relations of the work item id, either
	// way, each as stored, in the order they were made, (created_at, id).
	Relations(ctx context.Context, id uuid.UUID) ([]domain.Relation, error)
	// GetIssues reads the undeleted work items among ids, as
	// IssueReader's.
	GetIssues(ctx context.Context, ids []uuid.UUID) ([]domain.Issue, error)
}

// IssueArchiver is the work items' store as archiveIssue and
````
````old server/internal/modules/issue/app/ports.go
	Identifier string
````
````new server/internal/modules/issue/app/ports.go
	Identifier string
}

// ArchivedProjects reads which projects are archived, for a list across
// projects, which leaves out the work items of an archived project (M4
// design 3.8): project.Provide's, converted (bootstrap/ports.go).
type ArchivedProjects interface {
	// Archived reads those of projectIDs that are undeleted, archived
	// projects, each true, without a lock.
	Archived(ctx context.Context, projectIDs []uuid.UUID) (map[uuid.UUID]bool, error)
````

`server/internal/modules/issue/app/seen.go`（修改，2 处）：

````old server/internal/modules/issue/app/seen.go
	"context"
````
````new server/internal/modules/issue/app/seen.go
	"context"
	"errors"
````
````old server/internal/modules/issue/app/seen.go
}

// seenOne is the work item i as the caller actor sees it (seen), by his
````
````new server/internal/modules/issue/app/seen.go
}

// seeEach is those of items that the caller actor sees, in their order,
// without a lock (M4 design 3.4, 3.10), and his grants on their projects:
// issue.read decided on each one's project, once a project, as a list
// decides it on its project, which is each work item's decision there,
// issue.read allowing no creator (AllowCreator); then the guest rule on
// each. One of a project he does not see, or whose work items his role
// may not read, is left out, and so is one the guest rule hides from him.
// A failing decision is its error.
func (l Locks) seeEach(ctx context.Context, actor shared.Actor, items []domain.Issue) ([]domain.Issue, grants, error) {
	g, refused := grants{}, map[uuid.UUID]bool{}
	var out []domain.Issue
	for _, i := range items {
		if refused[i.ProjectID] {
			continue
		}
		grant, decided := g[i.ProjectID]
		if !decided {
			var err error
			grant, err = l.decide(ctx, actor, domain.ActionRead, i.WorkspaceID, i.ProjectID, nil, domain.ErrIssueNotFound)
			var refusal *shared.Error
			switch {
			case errors.As(err, &refusal):
				refused[i.ProjectID] = true
				continue
			case err != nil:
				return nil, nil, err
			}
			g[i.ProjectID] = grant
		}
		if !i.HiddenFrom(actor.UserID, grant.OwnOnly) {
			out = append(out, i)
		}
	}
	return out, g, nil
}

// seenOne is the work item i as the caller actor sees it (seen), by his
````

- [ ] **Step 4: 存储、HTTP 和接线**

`server/internal/modules/issue/adapter/http/handler.go`（修改，2 处）：

````old server/internal/modules/issue/adapter/http/handler.go
}

// CreateIssueRelationsUseCase is app.CreateIssueRelations.
````
````new server/internal/modules/issue/adapter/http/handler.go
}

// ListIssueRelationsUseCase is app.ListIssueRelations.
type ListIssueRelationsUseCase interface {
	Execute(ctx context.Context, id uuid.UUID) ([]app.IssueRelation, error)
}

// CreateIssueRelationsUseCase is app.CreateIssueRelations.
````
````old server/internal/modules/issue/adapter/http/handler.go
	AddSubIssues         AddSubIssuesUseCase
````
````new server/internal/modules/issue/adapter/http/handler.go
	AddSubIssues         AddSubIssuesUseCase
	ListIssueRelations   ListIssueRelationsUseCase
````

`server/internal/modules/issue/adapter/http/relations.go`（修改，1 处）：

````old server/internal/modules/issue/adapter/http/relations.go
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)
````
````new server/internal/modules/issue/adapter/http/relations.go
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// ListIssueRelations serves GET /api/v0/issues/{issue_id}/relations.
func (h handler) ListIssueRelations(ctx context.Context, req gen.ListIssueRelationsRequestObject) (gen.ListIssueRelationsResponseObject,
	error) {
	relations, err := h.uc.ListIssueRelations.Execute(ctx, req.IssueID)
	if err != nil {
		return nil, err
	}
	return gen.ListIssueRelations200JSONResponse(issueRelationCollection(relations)), nil
}
````

`server/internal/modules/issue/adapter/http/relations_test.go`（修改，12 处）：

````old server/internal/modules/issue/adapter/http/relations_test.go
	"github.com/open-nerve/NerveProject/server/internal/shared"
)
````
````new server/internal/modules/issue/adapter/http/relations_test.go
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeRelations is listIssueRelations: each call is recorded as "caller
// work item"; it answers answer, or err.
type fakeRelations struct {
	calls  []string
	answer []app.IssueRelation
	err    error
}

func (f *fakeRelations) Execute(ctx context.Context, id uuid.UUID) ([]app.IssueRelation, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	f.calls = append(f.calls, actor.UserID.String()+" "+id.String())
	return f.answer, f.err
}
````
````old server/internal/modules/issue/adapter/http/relations_test.go
	}
)
````
````new server/internal/modules/issue/adapter/http/relations_test.go
	}
)

// GET goes to the use case for the caller and the path's work item; the
// answer is 200 with the relations it answers, in its order, [] for none.
func TestListIssueRelations(t *testing.T) {
	for _, answer := range relatedAnswers {
		list := &fakeRelations{answer: answer.relations}
		res, body := do(t, newServer(t, httpadapter.UseCases{ListIssueRelations: list}), request(http.MethodGet, relationsPath, "alice", ""))
		if res.StatusCode != http.StatusOK || body != answer.json+"\n" {
			t.Errorf("GET = %d %s, want 200 %s", res.StatusCode, body, answer.json)
		}
		if want := []string{aliceID.String() + " " + webSixID.String()}; !slices.Equal(list.calls, want) {
			t.Errorf("calls %q, want %q", list.calls, want)
		}
	}
}
````
````old server/internal/modules/issue/adapter/http/relations_test.go
// The use case's refusals, as the contract declares them, and its
// failure.
func TestCreateIssueRelationsRefusals(t *testing.T) {
````
````new server/internal/modules/issue/adapter/http/relations_test.go
// The use cases' refusals, as the contract declares them, and their
// failure: listIssueRelations' are getIssue's; createIssueRelations'
// updateIssue's, and its own.
func TestIssueRelationsRefusals(t *testing.T) {
	post, both := []string{http.MethodPost}, []string{http.MethodGet, http.MethodPost}
````
````old server/internal/modules/issue/adapter/http/relations_test.go
		name   string
		err    error
		status int
		want   string
````
````new server/internal/modules/issue/adapter/http/relations_test.go
		name    string
		methods []string
		err     error
		status  int
		want    string
````
````old server/internal/modules/issue/adapter/http/relations_test.go
		{"a type of none of the four, and none named", domain.CheckNewRelations("start_before", nil), http.StatusUnprocessableEntity,
````
````new server/internal/modules/issue/adapter/http/relations_test.go
		{"a type of none of the four, and none named", post, domain.CheckNewRelations("start_before", nil), http.StatusUnprocessableEntity,
````
````old server/internal/modules/issue/adapter/http/relations_test.go
		{"a work item refused, and one related already", shared.Invalid(domain.RelatedNotAllowed(1), domain.RelatedAlready(2)),
````
````new server/internal/modules/issue/adapter/http/relations_test.go
		{"a work item refused, and one related already", post, shared.Invalid(domain.RelatedNotAllowed(1), domain.RelatedAlready(2)),
````
````old server/internal/modules/issue/adapter/http/relations_test.go
		{"no work item", domain.ErrIssueNotFound, http.StatusNotFound,
````
````new server/internal/modules/issue/adapter/http/relations_test.go
		{"no work item", both, domain.ErrIssueNotFound, http.StatusNotFound,
````
````old server/internal/modules/issue/adapter/http/relations_test.go
		{"a project guest", shared.Forbidden(), http.StatusForbidden,
````
````new server/internal/modules/issue/adapter/http/relations_test.go
		{"a project guest", both, shared.Forbidden(), http.StatusForbidden,
````
````old server/internal/modules/issue/adapter/http/relations_test.go
		{"an archived project", domain.ErrProjectArchived, http.StatusConflict,
````
````new server/internal/modules/issue/adapter/http/relations_test.go
		{"an archived project", post, domain.ErrProjectArchived, http.StatusConflict,
````
````old server/internal/modules/issue/adapter/http/relations_test.go
		{"an archived work item", domain.ErrIssueArchived, http.StatusConflict,
````
````new server/internal/modules/issue/adapter/http/relations_test.go
		{"an archived work item", post, domain.ErrIssueArchived, http.StatusConflict,
````
````old server/internal/modules/issue/adapter/http/relations_test.go
		{"a busy project", shared.ServerBusy(time.Second), http.StatusServiceUnavailable, busy},
		{"a failure", errGone, http.StatusInternalServerError, `{"status":500,"code":"internal_error","title":"Internal Server Error"}`},
````
````new server/internal/modules/issue/adapter/http/relations_test.go
		{"a busy project", post, shared.ServerBusy(time.Second), http.StatusServiceUnavailable, busy},
		{"a failure", both, errGone, http.StatusInternalServerError, `{"status":500,"code":"internal_error","title":"Internal Server Error"}`},
````
````old server/internal/modules/issue/adapter/http/relations_test.go
		h := newServer(t, httpadapter.UseCases{CreateIssueRelations: &fakeRelate{err: tt.err}})
		body := `{"relation_type":"blocking","issue_ids":["0199a2b4-0000-7000-8000-0000000000e8"]}`
		res, got := do(t, h, request(http.MethodPost, relationsPath, "alice", body))
		if res.StatusCode != tt.status || got != tt.want+"\n" || tt.want == busy && res.Header.Get("Retry-After") != "1" {
			t.Errorf("%s: POST = %d %s, Retry-After %q; want %d %s", tt.name, res.StatusCode, got, res.Header.Get("Retry-After"), tt.status, tt.want)
````
````new server/internal/modules/issue/adapter/http/relations_test.go
		for _, method := range tt.methods {
			h := newServer(t, httpadapter.UseCases{ListIssueRelations: &fakeRelations{err: tt.err}, CreateIssueRelations: &fakeRelate{err: tt.err}})
			body := ""
			if method == http.MethodPost {
				body = `{"relation_type":"blocking","issue_ids":["0199a2b4-0000-7000-8000-0000000000e8"]}`
			}
			res, got := do(t, h, request(method, relationsPath, "alice", body))
			if res.StatusCode != tt.status || got != tt.want+"\n" || tt.want == busy && res.Header.Get("Retry-After") != "1" {
				t.Errorf("%s: %s = %d %s, Retry-After %q; want %d %s", tt.name, method, res.StatusCode, got, res.Header.Get("Retry-After"),
					tt.status, tt.want)
			}
````

`server/internal/modules/issue/adapter/postgres/relations.go`（修改，1 处）：

````old server/internal/modules/issue/adapter/postgres/relations.go
}

// MarkChanged writes the work items r names last changed at its moment and
````
````new server/internal/modules/issue/adapter/postgres/relations.go
}

// Relations reads the undeleted relations of the work item id, either way,
// each as stored, in the order they were made, (created_at, id).
func (s *Store) Relations(ctx context.Context, id uuid.UUID) ([]domain.Relation, error) {
	rows, err := s.queries(ctx).IssueRelations(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("read the relations of work item %s: %w", id, err)
	}
	out := make([]domain.Relation, len(rows))
	for n, r := range rows {
		out[n] = relationOf(r)
	}
	return out, nil
}

// relationOf is a relation's row as the domain has it.
func relationOf(r gen.IssueRelationsRow) domain.Relation {
	return domain.Relation{ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, IssueID: r.IssueID, RelatedID: r.RelatedIssueID,
		Type: domain.RelationType(r.RelationType), CreatedByID: r.CreatedByID, CreatedAt: r.CreatedAt}
}

// MarkChanged writes the work items r names last changed at its moment and
````

`server/internal/modules/issue/adapter/postgres/relations_test.go`（修改，2 处）：

````old server/internal/modules/issue/adapter/postgres/relations_test.go
	"maps"
````
````new server/internal/modules/issue/adapter/postgres/relations_test.go
	"maps"
	"reflect"
````
````old server/internal/modules/issue/adapter/postgres/relations_test.go
		t.Errorf("MarkChanged() on a cancelled context = %v; want context.Canceled", err)
	}
}

````
````new server/internal/modules/issue/adapter/postgres/relations_test.go
		t.Errorf("MarkChanged() on a cancelled context = %v; want context.Canceled", err)
	}
}

// A work item's relations are read either way, each as stored, in the
// order they were made, (created_at, id) (M4 design 3.10): of web's item,
// a's blocked_by it, made first by bob; ops' o's duplicate of it and
// item's relates_to b, made a second later by alice, o's id drawn before,
// so first. Its relation to c was deleted; d's to b is not its. A work
// item without relations has none; a cancelled read is its failure.
func TestRelations(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, ops := newProject(t, pool, acme, "WEB"), newProject(t, pool, acme, "OPS")
	item, a, b, c, d, o := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	for n, id := range []uuid.UUID{item, a, b, c, d} {
		newIssue(t, pool, web, id, n+1, alice)
	}
	newIssue(t, pool, ops, o, 1, alice)
	ofO, ofItem, ofA, gone := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	later := now.Add(time.Second)
	want := []domain.Relation{
		{ID: ofA, WorkspaceID: acme, ProjectID: web.id, IssueID: a, RelatedID: item, Type: domain.BlockedBy, CreatedByID: &bob, CreatedAt: now},
		{ID: ofO, WorkspaceID: acme, ProjectID: ops.id, IssueID: o, RelatedID: item, Type: domain.Duplicate, CreatedByID: &alice, CreatedAt: later},
		{ID: ofItem, WorkspaceID: acme, ProjectID: web.id, IssueID: item, RelatedID: b, Type: domain.RelatesTo, CreatedByID: &alice,
			CreatedAt: later},
	}
	for _, r := range []domain.Relation{want[2], want[0], want[1]} {
		p := web
		if r.ProjectID == ops.id {
			p = ops
		}
		newRelation(t, pool, p, r.IssueID, r.RelatedID, r.ID, *r.CreatedByID)
		exec(t, pool, "UPDATE issue_relations SET relation_type = $2, created_at = $3 WHERE id = $1", r.ID, string(r.Type), r.CreatedAt)
	}
	newRelation(t, pool, web, item, c, gone, alice)
	exec(t, pool, "UPDATE issue_relations SET deleted_at = $2 WHERE id = $1", gone, now)
	newRelation(t, pool, web, d, b, uuid.NewV7(), alice)

	if got, err := s.Relations(context.Background(), item); err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("Relations(item) = %+v, %v; want %+v", got, err, want)
	}
	if got, err := s.Relations(context.Background(), c); err != nil || len(got) != 0 {
		t.Errorf("Relations(c) = %+v, %v; want none", got, err)
	}
	cancelled, stop := context.WithCancel(context.Background())
	stop()
	if _, err := s.Relations(cancelled, item); !errors.Is(err, context.Canceled) {
		t.Errorf("Relations() on a cancelled context = %v; want context.Canceled", err)
	}
}

````

`server/internal/modules/issue/module.go`（修改，4 处）：

````old server/internal/modules/issue/module.go
// work items under one and putting work items under it, and relating a
// work item to others. For the
````
````new server/internal/modules/issue/module.go
// work items under one and putting work items under it, and listing a
// work item's relations and relating it to others. For the
````
````old server/internal/modules/issue/module.go
// writes; a workspace's projects' identifiers, for a read by identifier.
````
````new server/internal/modules/issue/module.go
// writes; a workspace's projects' identifiers, for a read by identifier;
// which projects are archived, for a list across projects (3.8).
````
````old server/internal/modules/issue/module.go
	app.ProjectIdentifiers
````
````new server/internal/modules/issue/module.go
	app.ProjectIdentifiers
	app.ArchivedProjects
````
````old server/internal/modules/issue/module.go
		DeleteIssue:    app.NewDeleteIssue(app.DeleteIssueDeps{Locks: locks, Rows: store, Clock: d.Clock}),
		ArchiveIssue:   app.NewArchiveIssue(archive),
		UnarchiveIssue: app.NewUnarchiveIssue(archive),
		ListSubIssues:  app.NewListSubIssues(locks, store),
		AddSubIssues:   app.NewAddSubIssues(app.AddSubIssuesDeps{Locks: locks, Issues: store, Clock: d.Clock}),
````
````new server/internal/modules/issue/module.go
		DeleteIssue:        app.NewDeleteIssue(app.DeleteIssueDeps{Locks: locks, Rows: store, Clock: d.Clock}),
		ArchiveIssue:       app.NewArchiveIssue(archive),
		UnarchiveIssue:     app.NewUnarchiveIssue(archive),
		ListSubIssues:      app.NewListSubIssues(locks, store),
		AddSubIssues:       app.NewAddSubIssues(app.AddSubIssuesDeps{Locks: locks, Issues: store, Clock: d.Clock}),
		ListIssueRelations: app.NewListIssueRelations(locks, store, d.Projects),
````

- [ ] **Step 5: 组合和矩阵的种子**

`server/internal/bootstrap/issue_archived_projects_test.go`（新文件，44 行）：

````file server/internal/bootstrap/issue_archived_projects_test.go
package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"testing"
	"uuid"
)

// The conversion of a list's read across projects (M4 design 3.8): which
// projects are archived, as bootstrap/ports.go hands it to issue.

// ArchivedIssueProjects answers fakeIssueProjects' web archived, of
// whatever projects it is asked.
func (f *fakeIssueProjects) ArchivedIssueProjects(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]bool, error) {
	f.asked = append(f.asked, fmt.Sprint("archived ", ids))
	if f.err != nil {
		return nil, f.err
	}
	return map[uuid.UUID]bool{f.web: true}, nil
}

// issueProjects asks project for the projects issue asks about and hands
// over its answer as it came; an error as it came, without projects.
func TestIssueProjectsConvertsArchivedProjects(t *testing.T) {
	web, ops := uuid.NewV7(), uuid.NewV7()
	fake := &fakeIssueProjects{web: web}
	p := issueProjects{projects: fake}
	got, err := p.Archived(context.Background(), []uuid.UUID{web, ops})
	if want := map[uuid.UUID]bool{web: true}; err != nil || !maps.Equal(got, want) {
		t.Errorf("Archived() = %v, %v; want %v", got, err, want)
	}
	if want := []string{fmt.Sprint("archived ", []uuid.UUID{web, ops})}; !slices.Equal(fake.asked, want) {
		t.Errorf("project was asked %q, want %q", fake.asked, want)
	}
	failure := errors.New("connection reset")
	fake.err = failure
	if got, err := p.Archived(context.Background(), []uuid.UUID{web}); !errors.Is(err, failure) || got != nil {
		t.Errorf("Archived() = %v, %v; want no projects and %v", got, err, failure)
	}
}
````

`server/internal/bootstrap/issue_relation_lists_test.go`（新文件，109 行）：

````file server/internal/bootstrap/issue_relation_lists_test.go
package bootstrap

import (
	"net/http"
	"reflect"
	"slices"
	"testing"
	"uuid"
)

// listedRelations are the relations the account name lists of the work
// item id, in their order, each as "<type> <other work item's name>".
func (w issueWorld) listedRelations(t *testing.T, name string, id uuid.UUID) []string {
	t.Helper()
	var said []string
	for _, r := range listIn[answeredRelation](t, w.send(t, name, http.MethodGet, relationsOf(id), "", http.StatusOK)) {
		said = append(said, r.RelationType+" "+r.Issue.Name)
	}
	return said
}

// relationList is a list of relations a test expects: the account name
// lists the work item item's, which are want (listedRelations).
type relationList struct {
	name string
	item uuid.UUID
	want []string
}

// A work item's relations as bootstrap wires them (M4 design 3.4, 3.8,
// 3.10): alice makes Ops, bob its member, and Lab, private, of which he is
// none; Near, Mid, Shelved and Gone in Web, Far and Beside in Ops, Lab's
// in Lab. alice relates Near to Far, and Far to Beside; bob Mid to Near,
// as blocking it; alice Near to Lab's, Shelved and Gone, as their
// duplicate. Then Shelved is archived and Gone deleted. alice lists Near's
// relations in the order they were made, each as Near reads it: Far, Mid,
// which blocks it, and Lab's; bob, who does not see Lab, lists Far and
// Mid alone. Far lists Near and Beside, and Mid Near, which it blocks.
// Shelved unarchived shows again. Once Ops is archived, Far, of a project
// other than Near's, is no longer in Near's list, but Far's own list
// still shows Near and Beside, of Far's own project. walt, acme's admin
// and Web's guest (adminGuest), who sees only his own work items in Web,
// relates his His to his Other, which alice puts under her Up: he lists
// the relation as alice does, compared whole, but with Other under none,
// since Up is hidden from him.
func TestARelationListShowsWhatTheCallerSees(t *testing.T) {
	w := newIssueWorld(t)
	ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+ops.String()+"/members",
		`{"members":[{"member_id":"`+w.ids["bob"].String()+`","role":15}]}`, http.StatusCreated)
	var lab struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/workspaces/acme/projects", `{"name":"Lab","identifier":"LAB","network":0}`,
		http.StatusCreated), &lab)
	near := w.newWorkItemIn(t, w.web, `{"name":"Near"}`)
	mid := w.newWorkItemIn(t, w.web, `{"name":"Mid"}`)
	far := w.newWorkItemIn(t, ops, `{"name":"Far"}`)
	beside := w.newWorkItemIn(t, ops, `{"name":"Beside"}`)
	labs := w.newWorkItemIn(t, lab.ID, `{"name":"Lab's"}`)
	shelved := w.newWorkItemIn(t, w.web, `{"name":"Shelved","state_id":"`+w.done.String()+`"}`)
	gone := w.newWorkItemIn(t, w.web, `{"name":"Gone"}`)
	w.send(t, "alice", http.MethodPost, relationsOf(near), relating("relates_to", far), http.StatusCreated)
	w.send(t, "alice", http.MethodPost, relationsOf(far), relating("relates_to", beside), http.StatusCreated)
	w.send(t, "bob", http.MethodPost, relationsOf(mid), relating("blocking", near), http.StatusCreated)
	w.send(t, "alice", http.MethodPost, relationsOf(near), relating("duplicate", gone, shelved, labs), http.StatusCreated)
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+shelved.String()+"/archive", "", http.StatusOK)
	w.send(t, "alice", http.MethodDelete, "/api/v0/issues/"+gone.String(), "", http.StatusNoContent)

	lists := func(when string, expected ...relationList) {
		t.Helper()
		for _, l := range expected {
			if got := w.listedRelations(t, l.name, l.item); !slices.Equal(got, l.want) {
				t.Errorf("%s, %s lists %s's relations %q, want %q", when, l.name, l.item, got, l.want)
			}
		}
	}
	lists("at first", relationList{"alice", near, []string{"relates_to Far", "blocked_by Mid", "duplicate Lab's"}},
		relationList{"bob", near, []string{"relates_to Far", "blocked_by Mid"}},
		relationList{"bob", far, []string{"relates_to Near", "relates_to Beside"}},
		relationList{"bob", mid, []string{"blocking Near"}})
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+shelved.String()+"/unarchive", "", http.StatusOK)
	lists("Shelved unarchived", relationList{"bob", near, []string{"relates_to Far", "blocked_by Mid", "duplicate Shelved"}})
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+ops.String()+"/archive", "", http.StatusOK)
	lists("Ops archived", relationList{"bob", near, []string{"blocked_by Mid", "duplicate Shelved"}},
		relationList{"bob", far, []string{"relates_to Near", "relates_to Beside"}})

	w.adminGuest(t)
	up := w.newWorkItemIn(t, w.web, `{"name":"Up"}`)
	his, other := w.newWorkItemBy(t, "walt", w.web, `{"name":"His"}`), w.newWorkItemBy(t, "walt", w.web, `{"name":"Other"}`)
	w.send(t, "alice", http.MethodPatch, "/api/v0/issues/"+other.String(), `{"parent_id":"`+up.String()+`"}`, http.StatusOK)
	w.send(t, "walt", http.MethodPost, relationsOf(his), relating("relates_to", other), http.StatusCreated)
	var listed [2]answeredRelation
	for n, name := range []string{"walt", "alice"} {
		relations := listIn[answeredRelation](t, w.send(t, name, http.MethodGet, relationsOf(his), "", http.StatusOK))
		if len(relations) != 1 {
			t.Fatalf("%s lists %d relations of His, want 1", name, len(relations))
		}
		listed[n] = relations[0]
	}
	want := listed[1]
	if want.Issue.ParentID == nil || *want.Issue.ParentID != up {
		t.Errorf("alice lists Other under %v, want under Up, %s", want.Issue.ParentID, up)
	}
	want.Issue.ParentID = nil
	if !reflect.DeepEqual(listed[0], want) {
		t.Errorf("walt lists His's relation %+v; want alice's, with Other under none, %+v", listed[0], want)
	}
}
````

`server/internal/bootstrap/issue_relations_test.go`（修改，2 处）：

````old server/internal/bootstrap/issue_relations_test.go
// alone, answered with the one sub-item of Mine he sees. Up, alice's,
// walt finds not to relate: the very 404 of a work item not there.
````
````new server/internal/bootstrap/issue_relations_test.go
// alone, answered with the one sub-item of Mine he sees. walt lists His's
// relations: Mine alone, with its one sub-item, though Hers, of Web too,
// comes first; alice lists Hers and Mine, with its two. Up, alice's, walt
// finds not to relate: the very 404 of a work item not there.
````
````old server/internal/bootstrap/issue_relations_test.go
		t.Errorf("the relations stored:\n%q\nwant\n%q", got, want)
	}
	hidden := w.send(t, "walt", http.MethodPost, relationsOf(up), relating("relates_to", mine), http.StatusNotFound)
````
````new server/internal/bootstrap/issue_relations_test.go
		t.Errorf("the relations stored:\n%q\nwant\n%q", got, want)
	}
	for _, tt := range []struct {
		name string
		want []string
	}{{"walt", []string{"Mine 1"}}, {"alice", []string{"Hers 0", "Mine 2"}}} {
		listed := listIn[answeredRelation](t, w.send(t, tt.name, http.MethodGet, relationsOf(his), "", http.StatusOK))
		if got := counts(othersOf(listed)); !slices.Equal(got, tt.want) {
			t.Errorf("%s lists His's relations to %q, want %q", tt.name, got, tt.want)
		}
	}
	hidden := w.send(t, "walt", http.MethodPost, relationsOf(up), relating("relates_to", mine), http.StatusNotFound)
````

`server/internal/bootstrap/permission_matrix_relations_test.go`（修改，3 处）：

````old server/internal/bootstrap/permission_matrix_relations_test.go
	"reflect"
````
````new server/internal/bootstrap/permission_matrix_relations_test.go
	"reflect"
	"slices"
````
````old server/internal/bootstrap/permission_matrix_relations_test.go
	relates[callerAdminGuestOwn] = cellValidationFailed
	return []matrixRow{
````
````new server/internal/bootstrap/permission_matrix_relations_test.go
	relates[callerAdminGuestOwn] = cellValidationFailed
	list := toWorkItem(http.MethodGet, "/relations", "")
	return []matrixRow{
		// Whom getIssue answers the work item (M4 design 3.4, 9.2): its
		// project's admins, members and the guests who see it.
		{op: "listIssueRelations", columns: issueColumns, request: list, cells: onAWorkItem(cellOK, cellOK, cellOK, cellOK),
			check: listsRelations},
		// An archived project's work items' are listed as any other's.
		{op: "listIssueRelations", variant: "archived", columns: archivedColumns, request: list, cells: issueRows.ofArchived(cellOK, cellForbidden),
			check: listsRelations},
````
````old server/internal/bootstrap/permission_matrix_relations_test.go
}

// relatedTo is the check of a row whose callers relate their column's work
````
````new server/internal/bootstrap/permission_matrix_relations_test.go
}

// relationsListed are the seeded relations (seededRelations) a column
// lists of its work item (issueOf), each as "<its type as the work item
// reads it> <the other's project key>|<the other's name>", in the order
// they were made. PA, PM and PM+WA, members of acme's public and private
// projects, list both of the public project's matrixIssue; the member who
// made one, his own's; the guest who sees every work item, of the private
// project's matrixIssue, the member before's alone: the public project's
// guest rule hides its matrixIssue and the workspace admin's own from
// him. Every other column lists none: the guest's own's other end the
// guest rule hides from him, the workspace admin who is a guest's own's is
// in a project he is none of, and no seeded relation is of the archived
// project's work item.
var relationsListed = map[caller][]string{
	callerProjectAdmin:   {"blocked_by acme/private|" + matrixIssue, "relates_to acme/public|" + issueOf(callerGuestOwn)},
	callerProjectMember:  {"blocked_by acme/private|" + matrixIssue, "relates_to acme/public|" + issueOf(callerGuestOwn)},
	callerMemberAndAdmin: {"blocked_by acme/private|" + matrixIssue, "relates_to acme/public|" + issueOf(callerGuestOwn)},
	callerCreator:        {"duplicate acme/private|" + issueOf(callerBefore)},
	callerGuestAll:       {"blocking acme/private|" + issueOf(callerBefore)},
}

// listsRelations: the relations of the column's work item it sees, as
// relationsListed has them.
func listsRelations(t *testing.T, c caller, s seeded, answer string) {
	var got struct {
		Data []struct {
			RelationType string `json:"relation_type"`
			Issue        struct {
				ID uuid.UUID `json:"id"`
			} `json:"issue"`
		} `json:"data"`
	}
	decodeAnswer(t, answer, &got)
	listed := []string{}
	for _, r := range got.Data {
		other := r.Issue.ID.String()
		for key, id := range s.issues {
			if id == r.Issue.ID {
				other = key
			}
		}
		listed = append(listed, r.RelationType+" "+other)
	}
	if got.Data == nil || !slices.Equal(listed, relationsListed[c]) {
		t.Errorf("%s lists %s, that is %q; want %q", c, answer, listed, relationsListed[c])
	}
}

// relatedTo is the check of a row whose callers relate their column's work
````

`server/internal/bootstrap/permission_matrix_seed_relations_test.go`（新文件，79 行）：

````file server/internal/bootstrap/permission_matrix_seed_relations_test.go
package bootstrap

import (
	"context"
	"reflect"
	"slices"

	issuedomain "github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// seededRelation is a relation issueSeed stores: its name; its work items,
// each by its project's key and its name (seededItems), the first the end
// stored as issue_id, of its row's project; and its type as stored.
type seededRelation struct {
	name          string
	fromKey, from string
	toKey, to     string
	kind          issuedomain.RelationType
}

// seededRelations are the relations prepareMatrix seeds through the issue
// store (M4 design 9.2), each a case of a list of relations: acme's public
// project's matrixIssue blocked by the private one's (Blocked), which the
// guest rule of the public project hides from PG, who sees every work item
// of the private one; the guest's own work item's relates_to the public
// project's matrixIssue, which the guest rule hides from him; PM's
// duplicate of the member before's in the private project; the workspace
// admin who is a guest's own relates_to the private project's matrixIssue,
// of a project he is none of; and the member before's blocked by it
// (Secret), which PG sees there.
var seededRelations = []seededRelation{
	{"Blocked", "acme/public", matrixIssue, "acme/private", matrixIssue, issuedomain.BlockedBy},
	{"Guest's", "acme/public", issueOf(callerGuestOwn), "acme/public", matrixIssue, issuedomain.RelatesTo},
	{"Member's", "acme/public", issueOf(callerCreator), "acme/private", issueOf(callerBefore), issuedomain.Duplicate},
	{"Admin guest's", "acme/public", issueOf(callerAdminGuestOwn), "acme/private", matrixIssue, issuedomain.RelatesTo},
	{"Secret", "acme/private", issueOf(callerBefore), "acme/private", matrixIssue, issuedomain.BlockedBy},
}

// relationsOf are seededRelations as stored, each with the id sd names, of
// acme, by its admin, at the seed's moment.
func (s issueSeed) relationsOf(sd seeded) []issuedomain.Relation {
	by := s.ids[matrixAdmins["acme"]]
	rs := make([]issuedomain.Relation, len(seededRelations))
	for n, r := range seededRelations {
		rs[n] = issuedomain.Relation{ID: sd.relation(r.fromKey, r.name), WorkspaceID: s.workspaces["acme"], ProjectID: s.projects[r.fromKey],
			IssueID: sd.issue(r.fromKey, r.from), RelatedID: sd.issue(r.toKey, r.to), Type: r.kind, CreatedByID: &by, CreatedAt: s.now}
	}
	return rs
}

// relations stores seededRelations (relationsOf), in one insert, after
// the work items (issues).
func (s issueSeed) relations(sd seeded) {
	s.t.Helper()
	if err := s.store.InsertRelations(context.Background(), s.relationsOf(sd)); err != nil {
		s.t.Fatal(err)
	}
}

// seededRelationRows checks the relations the cells rest on, read back
// through the issue store among the relations of each one's stored
// issue_id end: each as relationsOf has it, but its moment, which the
// database keeps to the microsecond. One missing, or stored otherwise,
// would let a list of relations answer as a cell wants for another reason.
func (s issueSeed) seededRelationRows(sd seeded) {
	s.t.Helper()
	for _, want := range s.relationsOf(sd) {
		rs, err := s.store.Relations(context.Background(), want.IssueID)
		n := slices.IndexFunc(rs, func(r issuedomain.Relation) bool { return r.ID == want.ID })
		if err != nil || n < 0 {
			s.t.Fatalf("the relations of %s = %+v, %v; want %+v among them", want.IssueID, rs, err, want)
		}
		got := rs[n]
		got.CreatedAt = want.CreatedAt
		if !reflect.DeepEqual(got, want) {
			s.t.Fatalf("the relation %s = %+v; want %+v", want.ID, rs[n], want)
		}
	}
}
````

`server/internal/bootstrap/permission_matrix_seeded_test.go`（修改，6 处）：

````old server/internal/bootstrap/permission_matrix_seeded_test.go
// work item, by the project's key and its name; and each account, by its
// name in matrixAccounts, which prepareMatrix registers. t is the test
// that asks for them (in).
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
// work item, by the project's key and its name; each relation, by the key
// of its row's project and its name; and each account, by its name in
// matrixAccounts, which prepareMatrix registers. t is the test that asks
// for them (in).
````
````old server/internal/bootstrap/permission_matrix_seeded_test.go
	issues         map[string]uuid.UUID
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
	issues         map[string]uuid.UUID
	relations      map[string]uuid.UUID
````
````old server/internal/bootstrap/permission_matrix_seeded_test.go
// project, and each work item of seededItems, before prepareMatrix writes
// them, so that matrixViolations, without a database, sees the keys and
// the targets the cells will.
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
// project, each work item of seededItems and each relation of
// seededRelations, in its order, before prepareMatrix writes them, so that
// matrixViolations, without a database, sees the keys and the targets the
// cells will.
````
````old server/internal/bootstrap/permission_matrix_seeded_test.go
		issues: map[string]uuid.UUID{}, accounts: map[caller]uuid.UUID{}}
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
		issues: map[string]uuid.UUID{}, relations: map[string]uuid.UUID{}, accounts: map[caller]uuid.UUID{}}
````
````old server/internal/bootstrap/permission_matrix_seeded_test.go
		s.issues[i.key+"|"+i.name] = uuid.NewV7()
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
		s.issues[i.key+"|"+i.name] = uuid.NewV7()
	}
	for _, r := range seededRelations {
		s.relations[r.fromKey+"|"+r.name] = uuid.NewV7()
````
````old server/internal/bootstrap/permission_matrix_seeded_test.go
}

// projectOfRow is the key of the project of id, a row of rows, which are
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
}

// relation is the id of the relation name of the project key, its row's;
// one never seeded fails the test at once, as issue's does.
func (s seeded) relation(key, name string) uuid.UUID {
	id, ok := s.relations[key+"|"+name]
	if !ok {
		s.t.Helper()
		s.t.Fatalf("no relation %s of %s is seeded", name, key)
	}
	return id
}

// projectOfRow is the key of the project of id, a row of rows, which are
````

`server/internal/bootstrap/permission_matrix_test.go`（修改，3 处）：

````old server/internal/bootstrap/permission_matrix_test.go
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
````new server/internal/bootstrap/permission_matrix_test.go
// every one (issueSeed), and the relations of seededRelations; through the
// project store, acme's archived project archived. Through both stores,
// the removed member's removal; then, through the workspace store, the
// invitations of matrixInvitations. Through the project store, the
// memberships P5b's writes end (endings); through SQL, the membership
// states no store makes alone (partingStates). Through the API, gone
// deleted by its admin, which soft-deletes its memberships and its
// project, with its states, labels and work item, with it; then the checks
// that the rows the cells rest on are there (preconditions), the states,
// the labels, the work items and the relations among them (seededStates,
// seededLabels, seededIssues, seededRelationRows).
````
````old server/internal/bootstrap/permission_matrix_test.go
		issues.issues(s)
````
````new server/internal/bootstrap/permission_matrix_test.go
		issues.issues(s)
		issues.relations(s)
````
````old server/internal/bootstrap/permission_matrix_test.go
		issues.seededIssues(s)
````
````new server/internal/bootstrap/permission_matrix_test.go
		issues.seededIssues(s)
		issues.seededRelationRows(s)
````

`server/internal/bootstrap/ports.go`（修改，1 处）：

````old server/internal/bootstrap/ports.go
}

// issueWorkspaces is workspace's WorkspaceDirectory as issue's: the same
````
````new server/internal/bootstrap/ports.go
}

func (p issueProjects) Archived(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]bool, error) {
	return p.projects.ArchivedIssueProjects(ctx, ids)
}

// issueWorkspaces is workspace's WorkspaceDirectory as issue's: the same
````

- [ ] **Step 6: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/issue/... ./internal/modules/project/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestARelationListShowsWhatTheCallerSees$|TestTheGuestRuleHoldsForRelations$|TestIssueProjectsConvertsArchivedProjects$|TestPermissionMatrix|TestThePermissionMatrixCoversEveryOperation$' ./internal/bootstrap/`
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
git add api/modules/issue.yaml server/internal/bootstrap/issue_archived_projects_test.go server/internal/bootstrap/issue_relation_lists_test.go server/internal/bootstrap/issue_relations_test.go server/internal/bootstrap/permission_matrix_relations_test.go server/internal/bootstrap/permission_matrix_seed_relations_test.go server/internal/bootstrap/permission_matrix_seeded_test.go server/internal/bootstrap/permission_matrix_test.go server/internal/bootstrap/ports.go server/internal/modules/issue/adapter/http/handler.go server/internal/modules/issue/adapter/http/relations.go server/internal/modules/issue/adapter/http/relations_test.go server/internal/modules/issue/adapter/postgres/queries/relations.sql server/internal/modules/issue/adapter/postgres/relations.go server/internal/modules/issue/adapter/postgres/relations_test.go server/internal/modules/issue/app/fakes_across_test.go server/internal/modules/issue/app/fakes_relations_test.go server/internal/modules/issue/app/list_relations.go server/internal/modules/issue/app/list_relations_test.go server/internal/modules/issue/app/lock_across.go server/internal/modules/issue/app/ports.go server/internal/modules/issue/app/seen.go server/internal/modules/issue/module.go server/internal/modules/project/adapter/postgres/failures_test.go server/internal/modules/project/adapter/postgres/issue_ports.go server/internal/modules/project/adapter/postgres/issue_ports_test.go server/internal/modules/project/adapter/postgres/queries/issue_ports.sql server/internal/modules/project/module.go api/dist/openapi.yaml server/internal/modules/issue/adapter/http/gen/server.gen.go server/internal/modules/issue/adapter/postgres/gen/relations.sql.go server/internal/modules/project/adapter/postgres/gen/issue_ports.sql.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M4/P3): listIssueRelations shows the relations whose other work item the caller sees

GET /api/v0/issues/{issue_id}/relations reads the work item as getIssue
does, then its relations either way, in the order they were made, each
as the work item reads it. The other work item is shown when it is not
archived, nor of an archived project other than the work item's own, and
the caller sees it: issue.read decided once a project, then the guest
rule on each, counted by his grant there. The matrix seeds relations.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p3.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试，或没有替身的平台、领域、转换的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；守卫是生成器（`make gen-go`）和中英文翻译键（`make lint-web`）的核对。每个变异在一份新的副本里，在本 Task 的树上、标出的较晚的 Task 的树上和最终的树上都跑过，端到端的检查只在标出的 Task 的树上）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T12.1` | 别的已归档项目里的工作项出现在关联列表里 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestARelationListShowsWhatTheCallerSees`（`make test`） | 单元；组合 |
| `T12.2` | 工作项自己的已归档项目把它的其余工作项从关联列表里去掉 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestARelationListShowsWhatTheCallerSees`（`make test`） | 单元；组合 |
| `T12.3` | 读项目是否归档失败时略过：当作没有归档的 | `internal/modules/issue/app` 的全部测试（`make test`） | 单元 |
| `T12.4` | 已归档的工作项出现在关联列表里 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestARelationListShowsWhatTheCallerSees`（`make test`） | 单元；组合 |
| `T12.5` | 关联列表给只看自己的访客看别人的工作项 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `listIssueRelations` 各格（`make test`） | 单元；组合 |
| `T12.6` | 关联列表给调用者看他看不到的项目里的工作项 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestARelationListShowsWhatTheCallerSees`（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `listIssueRelations` 各格（`make test`） | 单元；组合 |
| `T12.7` | 对另一个工作项的项目的判定带着这个工作项：对调用者隐藏的一个藏起它项目里的每一个 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestTheGuestRuleHoldsForRelations`（`make test`） | 单元；组合 |
| `T12.8` | 只按一个方向读工作项的关联 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestARelationListShowsWhatTheCallerSees`（`make test`）、故事 `i9-relations`（Task 15 起） | 存储；组合；端到端 |
| `T12.9` | 工作项的关联列表含已删除的 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestARelationIsDeletedFromEitherWorkItem`（`make test`）（Task 14 起）、故事 `i9-relations`（Task 15 起） | 存储；组合；端到端 |
| `T12.10` | 工作项的关联按 id 的次序列出 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`） | 存储 |
| `T12.11` | 读出的关联两端反了 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestARelationListShowsWhatTheCallerSees`（`make test`） | 存储；组合 |
| `T12.12` | 问到的每个未删除的项目都读成已归档 | `internal/modules/project/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestARelationListShowsWhatTheCallerSees`（`make test`） | 存储；组合 |
| `T12.13` | 已删除的已归档项目读成已归档 | `internal/modules/project/adapter/postgres` 的全部测试（`make test`） | 存储 |
| `T12.14` | 每个已归档的项目都读成已归档，问没问到都算 | `internal/modules/project/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestARelationListShowsWhatTheCallerSees`（`make test`） | 存储；组合 |
| `T12.15` | 没有项目读成已归档 | `internal/modules/project/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestARelationListShowsWhatTheCallerSees`（`make test`） | 存储；组合 |
| `T12.16` | `bootstrap` 交给 `issue` 的已归档项目是空的 | `internal/bootstrap` 的 `TestIssueProjectsConvertsArchivedProjects`（`make test`）、`internal/bootstrap` 的 `TestARelationListShowsWhatTheCallerSees`（`make test`） | 单元；组合 |
| `T12.17` | `GET` 回答没有关联 | `internal/modules/issue/adapter/http` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestARelationListShowsWhatTheCallerSees`（`make test`） | 单元；组合 |
| `T12.18` | 关联的列表按不带访客规则的授权数另一端的子工作项 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestTheGuestRuleHoldsForRelations`（`make test`） | 单元；组合 |

本 Task 的检查也发现较早的 Task 的变异 `T4.12`（列在各自的 Task 的表里，标"（Task 12 起）"）：本 Task 做完之后，在本 Task 的树上重跑它们（`mutants_p3.py`，`MUT_LATER=12`）。

---

### Task 13: 删除关联：两端任一个的判定、用例和存储

**Files:**
- Create: `server/internal/modules/issue/app/delete_relation.go`、`server/internal/modules/issue/app/delete_relation_test.go`、`server/internal/modules/issue/app/lock_either.go`
- Modify: `server/internal/modules/issue/adapter/postgres/queries/relations.sql`、`server/internal/modules/issue/adapter/postgres/relations.go`、`server/internal/modules/issue/adapter/postgres/relations_test.go`、`server/internal/modules/issue/app/fakes_relations_test.go`、`server/internal/modules/issue/app/lock.go`、`server/internal/modules/issue/app/lock_across.go`、`server/internal/modules/issue/app/lock_test.go`、`server/internal/modules/issue/app/ports.go`、`server/internal/modules/issue/app/rows.go`、`server/internal/modules/issue/domain/activity.go`、`server/internal/modules/issue/domain/errors.go`、`server/internal/modules/issue/domain/relation.go`、`server/internal/modules/issue/domain/relation_test.go`
- Generate: `server/internal/modules/issue/adapter/postgres/gen/relations.sql.go`

**Interfaces:**（spec 2.15、第 3 节第 2、3、8、9、11、12 条；M4 设计 3.5 约定七、3.7、3.10、3.13）`Locks.run` 的第二种新形式：两端任一个的写，也是一条取锁路径：
- `issue/domain`：`ErrRelationNotFound`（404 `issue.relation_not_found`，"The relation does not exist, or you cannot see either of its work items."）；`VerbDeleted`；`RelationRemoved(t, other)`（动态第 22、23 行：类型、`deleted`，对方的标识到 `""`，对方的 `id` 是 `old_identifier`）。
- `issue/app/lock.go`：`write.instead`（两端任一个的写的另一端：先以动作判定 `issue`，拒绝时再判定 `instead`，各在自己的项目；两个都拒绝才拒绝；两端都读、两个项目都锁在事务里，不论判定的是哪一个：组合一层由 Task 14 的交错 26 钉住，变异 T13.25）；`held.instead`（锁下读的另一端和它的项目）；`precheck` 和 `lockAndDecide` 转给它。`lock_across.go` 的 `lockProjects`：两端任一个的写也用 `LockMany`。
- `issue/app/lock_either.go`（新）：`both`（读两端，`issue` 的在先；不在是 `w.notFound`）、`eitherOf`（预检：读两端、判定，交回两个项目的 `scope`）、`eitherUnder`（锁下：重读两端；任一端的项目不在锁住的里（等锁期间被删除）先答 `w.notFound`，在任何判定之前；再判定）、`decideEither`（先判定第一个，拒绝时判定第二个，都在访客规则之后；两个都拒绝时：两个都看不到是 `w.notFound`，否则是第一个看得到的那个的拒绝（角色的 403）；判定的失败是错误）。
- `issue/app/ports.go`：`RelationDeleter`（`GetRelation`、`DeleteRelation`、`MarkChanged`、`InsertActivities`，嵌入 `IssueReader`）；`rows.go` 的 `RelationDeletion{ID, By, At}`。
- `issue/app/delete_relation.go`（新）：`DeleteIssueRelationDeps{Locks, Relations, Clock}`、`NewDeleteIssueRelation`、`Execute(ctx, id)`：不加锁地读关联（没有、已删除是 `issue.relation_not_found`）→ `Locks.run` 的 `write{issue: &r.IssueID, instead: &r.RelatedID, action: issue.update, notFound: ErrRelationNotFound}` → 锁下：重读关联（等锁期间被删除 `issue.relation_not_found`）→ 任一端的项目已归档 409 → 任一端已归档 409 `issue.archived` → `delete`（时刻；软删除这一行；两端按 `id` 的次序改最后修改；各一行动态，各在自己的项目，存为 `issue_id` 的那一端在先，类型按它读的，对方按标识）。
- `issue/adapter/postgres`：`queries/relations.sql` 的 `GetRelation`（未删除的这一条，与 `IssueRelations` 同样的列）、`DeleteRelation`（`:execrows`，按键软删除一行）；`relations.go`（`GetRelation` 把行转成 `IssueRelationsRow` 共用 `relationOf`；`DeleteRelation` 没写到一行是错误）。
- 测试的帮手：`fakes_relations_test.go` 的 `GetRelation`、`DeleteRelation`、`deleted`；`delete_relation_test.go`（新）：`relationID`、`newUnrelate`（near 与 far 的关联，存为 far 的 `blocked_by`）、`unrelating`、`through`、`archived`。

**Tests:**
- 领域：`TestRelationRemoved`。
- 单元：`TestDeleteIssueRelationDeletesItOnEitherWorkItem`（3.5、3.10 的次序：两端任一个的 `Locks` 的调用（`unrelating`）；时刻；删除关联；两端按 `id` 的次序改最后修改；动态。调用者能改任一端：存为 `issue_id` 的 far 先判定，不能改它时（角色不允许，或 ops 里访客规则隐藏 far）判定 near；far 的一行在先（ops 的，它的 `blocked_by WEB-8` 删除），然后 near 的（web 的，`blocking OPS-3` 删除））、`TestDeleteIssueRelationRefusals`（锁之前：没有关联、一端不在 `issue.relation_not_found`；两端都看不到（项目或访客规则）也是，看得到一端而两端都不能改 403；锁下：任一端的项目等锁期间被删除（在任何判定之前）、任一端等锁期间被删除、关联被删除、两端都不再看得到 `issue.relation_not_found`，都不能再改 403，任一端的项目已归档 409，然后任一端已归档 409；每个回答整个比较：两端都看不到的关联与不存在的是同一个回答，细节都一样（8.4）；都不写）、`TestDeleteIssueRelationStopsAtAFailingPort`；`lock_test.go` 的 `TestAWriteOnEitherWorkItemTakesBothProjects`（像跨项目的写那样取两端的项目，不论判定放行的是哪一端：far 的 ops，或 near 的 web；同一个项目的两端只取一次它的闸门和锁）、`TestAWriteAcrossProjectsTakesOnlyTheProjectsItReaches` 共用 `wantAllAtOnce`。
- 存储：`TestGetRelation`（按 `id` 读出如存的每一列；已删除的、没有的读不到；取消的读是它的失败）、`TestDeleteRelation`（在给的时刻由给的账户软删除，别的列和别的关联不动；已删除的、没有的不写，是错误；取消的写也是）。

- [ ] **Step 1: 查询，生成**

`server/internal/modules/issue/adapter/postgres/queries/relations.sql`（修改，1 处）：

````old server/internal/modules/issue/adapter/postgres/queries/relations.sql
ORDER BY r.created_at, r.id;

````
````new server/internal/modules/issue/adapter/postgres/queries/relations.sql
ORDER BY r.created_at, r.id;

-- name: GetRelation :one
-- The undeleted relation id, as stored, of the same columns as IssueRelations (M4 design 3.10): deleteIssueRelation's,
-- read without a lock first, then again under the locks of both its work items' projects, one of which every write of
-- it holds at least (3.5, convention 7).
SELECT r.id, r.workspace_id, r.project_id, r.issue_id, r.related_issue_id, r.relation_type, r.created_by_id, r.created_at
FROM issue_relations r
WHERE r.id = sqlc.arg(id)::uuid AND r.deleted_at IS NULL;

-- name: DeleteRelation :execrows
-- deleteIssueRelation's soft deletion of the undeleted relation id, at the moment and by the account given (M4 design
-- 3.10), under the locks of both its work items' projects: one row, by its key.
UPDATE issue_relations
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(by)::uuid
WHERE id = sqlc.arg(id)::uuid AND deleted_at IS NULL;

````

Run: `make gen-go`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `ca88e55bf2869dbf94d7a9369b5279158c8d373b9f413d1f18d2fa8030e810fa` | 218 | `server/internal/modules/issue/adapter/postgres/gen/relations.sql.go` |

Run: `shasum -a 256 server/internal/modules/issue/adapter/postgres/gen/relations.sql.go`
Expected: 与上表相同。

- [ ] **Step 2: 领域**

`server/internal/modules/issue/domain/activity.go`（修改，1 处）：

````old server/internal/modules/issue/domain/activity.go
	VerbUpdated Verb = "updated"
````
````new server/internal/modules/issue/domain/activity.go
	VerbUpdated Verb = "updated"
	VerbDeleted Verb = "deleted"
````

`server/internal/modules/issue/domain/errors.go`（修改，1 处）：

````old server/internal/modules/issue/domain/errors.go
		"Only a completed or cancelled work item can be archived; move it to such a state first.")
````
````new server/internal/modules/issue/domain/errors.go
		"Only a completed or cancelled work item can be archived; move it to such a state first.")
	// ErrRelationNotFound answers a relation that does not exist or is
	// deleted, and one neither of whose work items the caller sees, all
	// with this one value (M4 design 3.10, 5.3).
	ErrRelationNotFound = shared.NewError(shared.KindNotFound, "issue.relation_not_found",
		"The relation does not exist, or you cannot see either of its work items.")
````

`server/internal/modules/issue/domain/relation.go`（修改，1 处）：

````old server/internal/modules/issue/domain/relation.go
		Comment: "added " + string(t) + " relation"}
}

````
````new server/internal/modules/issue/domain/relation.go
		Comment: "added " + string(t) + " relation"}
}

// RelationRemoved is the row of a deleted relation on one of its work
// items (rows 22 and 23 of M4 design 3.13, of the same columns): its type
// as the work item reads it, deleted, from the other work item's
// identifier to "", the other's id the old identifier.
func RelationRemoved(t RelationType, other Ref) Activity {
	return Activity{Field: ptr(string(t)), Verb: VerbDeleted, OldValue: ptr(other.Identifier), NewValue: ptr(""), OldIdentifier: &other.ID,
		Comment: "deleted " + string(t) + " relation"}
}

````

`server/internal/modules/issue/domain/relation_test.go`（修改，1 处）：

````old server/internal/modules/issue/domain/relation_test.go
		t.Errorf("RelationAdded() = %+v, want %+v", got, want)
	}
}

````
````new server/internal/modules/issue/domain/relation_test.go
		t.Errorf("RelationAdded() = %+v, want %+v", got, want)
	}
}

// A deleted relation writes a row on each of its work items, of the same
// columns (rows 22 and 23 of M4 design 3.13): the type as the work item
// reads it, deleted, from the other's identifier to "", the other's id the
// old identifier, Plane's sentence with the type.
func TestRelationRemoved(t *testing.T) {
	other := Ref{ID: uuid.NewV7(), Identifier: "OPS-3"}
	want := Activity{Field: ptr("blocked_by"), Verb: VerbDeleted, OldValue: ptr("OPS-3"), NewValue: ptr(""), OldIdentifier: &other.ID,
		Comment: "deleted blocked_by relation"}
	if got := RelationRemoved(BlockedBy, other); !reflect.DeepEqual(got, want) {
		t.Errorf("RelationRemoved() = %+v, want %+v", got, want)
	}
}

````

- [ ] **Step 3: 两端任一个的取锁**

`server/internal/modules/issue/app/lock.go`（修改，7 处）：

````old server/internal/modules/issue/app/lock.go
// (createIssueRelations' other ends).
````
````new server/internal/modules/issue/app/lock.go
// (createIssueRelations' other ends). A write on either of two work items
// (deleteIssueRelation's, 3.10) names the other too, instead: its action
// is decided on issue, then, when that refuses, on instead, each on its
// own project; only both refusing refuse it, and both work items are read
// and both projects locked, whichever was decided on (lock_either.go).
````
````old server/internal/modules/issue/app/lock.go
	issue    *uuid.UUID
````
````new server/internal/modules/issue/app/lock.go
	issue    *uuid.UUID
	instead  *uuid.UUID
````
````old server/internal/modules/issue/app/lock.go
// (M4 design 3.4); and, for a write across projects, the work items of
// write.reach, in its order, as reached under the lock (reachUnder).
````
````new server/internal/modules/issue/app/lock.go
// (M4 design 3.4); for a write across projects, the work items of
// write.reach, in its order, as reached under the lock (reachUnder); and,
// for a write on either of two work items, the other, write.instead's, as
// read under the lock (eitherUnder), and the grant the decision that let
// the write through answered, on either work item's project.
````
````old server/internal/modules/issue/app/lock.go
	reached []end
````
````new server/internal/modules/issue/app/lock.go
	reached []end
	instead end
````
````old server/internal/modules/issue/app/lock.go
// transaction (M4 design 3.5), and answers w on its project, with what it
// lets through to the lock (scope). For a write that names a work item:
// the work item read without a lock and the decision on it (see); a work
// item that is not there, whose project actor does not see, or that the
// guest rule hides from him, is issue.not_found; across projects, then,
// each work item it reaches (reachAll). For one on a project: the project
// read without a lock and the decision on it; a project that is not there,
// or not visible to actor, is w.notFound. A role the rule does not allow
// is the Authorizer's shared.Forbidden. What it lets through,
// lockAndDecide decides again.
func (l Locks) precheck(ctx context.Context, actor shared.Actor, w write) (write, scope, error) {
````
````new server/internal/modules/issue/app/lock.go
// transaction (M4 design 3.5), and answers w, on its project unless it is
// on either of two work items, with what it lets through to the lock
// (scope). For a write that names a work item: the work item read without
// a lock and the decision on it (see); a work item that is not there,
// whose project actor does not see, or that the guest rule hides from
// him, is issue.not_found; across projects, then, each work item it
// reaches (reachAll). For one on either of two work items: both read
// without a lock, and the decisions on them (eitherOf).
// For one on a project: the project read without a lock and the decision
// on it; a project that is not there, or not visible to actor, is
// w.notFound. A role the rule does not allow is the Authorizer's
// shared.Forbidden. What it lets through, lockAndDecide decides again.
func (l Locks) precheck(ctx context.Context, actor shared.Actor, w write) (write, scope, error) {
	if w.instead != nil {
		s, err := l.eitherOf(ctx, actor, w)
		return w, s, err
	}
````
````old server/internal/modules/issue/app/lock.go
// across projects, then, each work item it reaches (reachUnder). It
````
````new server/internal/modules/issue/app/lock.go
// across projects, then, each work item it reaches (reachUnder); on either
// of two work items, both read and decided on again (eitherUnder). It
````
````old server/internal/modules/issue/app/lock.go
	if err != nil {
		return held{}, err
````
````new server/internal/modules/issue/app/lock.go
	if err != nil {
		return held{}, err
	}
	if w.instead != nil {
		return l.eitherUnder(ctx, actor, w, locked)
````

`server/internal/modules/issue/app/lock_across.go`（修改，3 处）：

````old server/internal/modules/issue/app/lock_across.go
// end is a work item a write across projects reaches, as its lock holds
// it: the work item and its project, read under the lock, and the caller's
// grant on the project, as decided under it; ok is false for a work item
// the write does not reach, and the rest is zero then.
````
````new server/internal/modules/issue/app/lock_across.go
// end is a work item a write across projects reaches, or the other of a
// write on either of two work items, as its lock holds it: the work item
// and its project, read under the lock, and the caller's grant on the
// project, as decided under it, zero when none was; ok is false for a work
// item the write does not reach, and the rest is zero then.
````
````old server/internal/modules/issue/app/lock_across.go
// KEY UPDATE (ProjectGate.Lock); a write across projects each of s's, all
// at once in ascending id order (ProjectGate.LockMany), even when they are
// one. A project deleted while the lock waited is not among them.
````
````new server/internal/modules/issue/app/lock_across.go
// KEY UPDATE (ProjectGate.Lock); a write across projects, or on either of
// two work items, each of s's, all at once in ascending id order
// (ProjectGate.LockMany), even when they are one. A project deleted while
// the lock waited is not among them.
````
````old server/internal/modules/issue/app/lock_across.go
	if w.reach != nil {
````
````new server/internal/modules/issue/app/lock_across.go
	if w.reach != nil || w.instead != nil {
````

`server/internal/modules/issue/app/lock_either.go`（新文件，99 行）：

````file server/internal/modules/issue/app/lock_either.go
package app

import (
	"context"
	"errors"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The form of Locks.run for a write on either of two work items
// (deleteIssueRelation's, M4 design 3.10): it is the caller's to make
// when he may make it on either, decided on the first, then, when that
// refuses, on the second; both are read, and both their projects locked
// all at once in ascending id order (convention 7), whichever was decided
// on, since the write writes on both.

// both reads the two work items of a write on either of them, w.issue's,
// then w.instead's. One not there is w.notFound.
func (l Locks) both(ctx context.Context, w write) ([2]domain.Issue, error) {
	var items [2]domain.Issue
	for n, id := range []uuid.UUID{*w.issue, *w.instead} {
		i, found, err := l.issues.GetIssue(ctx, id)
		switch {
		case err != nil:
			return [2]domain.Issue{}, err
		case !found:
			return [2]domain.Issue{}, w.notFound
		}
		items[n] = i
	}
	return items, nil
}

// eitherOf is the precheck of a write on either of two work items, before
// any lock (M4 design 3.5, 3.10): both read (both), and the decision
// (decideEither). It answers the scope of the write: both their projects,
// in ascending id order, each once.
func (l Locks) eitherOf(ctx context.Context, actor shared.Actor, w write) (scope, error) {
	items, err := l.both(ctx, w)
	if err != nil {
		return scope{}, err
	}
	if _, err := l.decideEither(ctx, actor, w, items); err != nil {
		return scope{}, err
	}
	return scope{projects: ascending([]uuid.UUID{items[0].ProjectID, items[1].ProjectID})}, nil
}

// eitherUnder reads again, under the lock, the two work items of a write
// on either of them (both), and decides on them again (decideEither): it
// holds the first and its project, the second and its project
// (held.instead), and the grant that let the write through. A work item
// deleted while the lock waited is w.notFound, and so, before any
// decision, is one of a project deleted meanwhile, which locked does not
// hold.
func (l Locks) eitherUnder(ctx context.Context, actor shared.Actor, w write, locked map[uuid.UUID]LockedProject) (held, error) {
	items, err := l.both(ctx, w)
	if err != nil {
		return held{}, err
	}
	p, foundFirst := locked[items[0].ProjectID]
	q, foundSecond := locked[items[1].ProjectID]
	if !foundFirst || !foundSecond {
		return held{}, w.notFound
	}
	grant, err := l.decideEither(ctx, actor, w, items)
	if err != nil {
		return held{}, err
	}
	return held{project: p, issue: items[0], grant: grant, instead: end{issue: items[1], project: q, ok: true}}, nil
}

// decideEither decides w's action for actor on the first of items, then,
// when that refuses, on the second, each on its own project, the guest
// rule first (M4 design 3.4, 3.10), and answers the grant of the decision
// that let the write through. Both refusing is w.notFound when actor sees
// neither, else the refusal of the first he sees, his role's (forbidden).
// A failing decision is its error.
func (l Locks) decideEither(ctx context.Context, actor shared.Actor, w write, items [2]domain.Issue) (shared.Grant, error) {
	var refusal error
	for _, i := range items {
		grant, err := l.decide(ctx, actor, w.action, i.WorkspaceID, i.ProjectID, &i, domain.ErrIssueNotFound)
		var refused *shared.Error
		switch {
		case err == nil:
			return grant, nil
		case !errors.As(err, &refused):
			return shared.Grant{}, err
		case refusal == nil && !errors.Is(err, domain.ErrIssueNotFound):
			refusal = err
		}
	}
	if refusal == nil {
		refusal = w.notFound
	}
	return shared.Grant{}, refusal
}
````

`server/internal/modules/issue/app/lock_test.go`（修改，1 处）：

````old server/internal/modules/issue/app/lock_test.go
			enter, lock := line("Enter", anys(tt.projects)...)+" outside tx", line("LockMany", anys(tt.projects)...)
			at := slices.Index(w.log.calls, enter)
			if at < 0 || at+2 >= len(w.log.calls) || w.log.calls[at+1] != "Begin" || w.log.calls[at+2] != lock ||
				w.log.calls[len(w.log.calls)-1] != line("Leave", anys(tt.projects)...) || slices.Contains(w.log.calls[:at], "Begin") {
				t.Errorf("calls:\n%q\nwant %q after the precheck, then the transaction and %q, and the gates left last", w.log.calls, enter, lock)
			}
		})
	}
}

````
````new server/internal/modules/issue/app/lock_test.go
			wantAllAtOnce(t, w, tt.projects)
		})
	}
}

// A write on either of two work items (deleteIssueRelation's, M4 design
// 3.10, convention 7) takes the projects of both as a write across
// projects does (TestAWriteAcrossProjectsTakesOnlyTheProjectsItReaches),
// whichever work item the decision let it through on: far's, of ops, or
// near's, of web. Two work items of one project take its gate and its
// lock once.
func TestAWriteOnEitherWorkItemTakesBothProjects(t *testing.T) {
	for _, tt := range []struct {
		name     string
		set      func(w *world)
		projects []uuid.UUID
	}{
		{"decided on far", func(*world) {}, sorted([]uuid.UUID{web, ops})},
		{"decided on near", func(w *world) { w.auth.answers = []error{0: shared.Forbidden(), 2: shared.Forbidden()} }, sorted([]uuid.UUID{web, ops})},
		{"both of web", func(w *world) {
			far := w.issues.others[farID]
			far.ProjectID = web
			w.issues.others[farID] = far
		}, []uuid.UUID{web}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, w, rel := newUnrelate()
			gated(w, nil)
			tt.set(w)
			if err := unrelateOver(w, rel).Execute(as(alice), relationID); err != nil {
				t.Fatal(err)
			}
			wantAllAtOnce(t, w, tt.projects)
		})
	}
}

// wantAllAtOnce checks the calls of a write over w that takes projects all
// at once, in their order: it enters their gates after its precheck,
// outside any transaction, then begins it and locks them (LockMany), and
// leaves the gates last, once it has ended.
func wantAllAtOnce(t *testing.T, w *world, projects []uuid.UUID) {
	t.Helper()
	enter, lock := line("Enter", anys(projects)...)+" outside tx", line("LockMany", anys(projects)...)
	at := slices.Index(w.log.calls, enter)
	if at < 0 || at+2 >= len(w.log.calls) || w.log.calls[at+1] != "Begin" || w.log.calls[at+2] != lock ||
		w.log.calls[len(w.log.calls)-1] != line("Leave", anys(projects)...) || slices.Contains(w.log.calls[:at], "Begin") {
		t.Errorf("calls:\n%q\nwant %q after the precheck, then the transaction and %q, and the gates left last", w.log.calls, enter, lock)
	}
}

````

`server/internal/modules/issue/app/ports.go`（修改，1 处）：

````old server/internal/modules/issue/app/ports.go
}

// IssueArchiver is the work items' store as archiveIssue and
````
````new server/internal/modules/issue/app/ports.go
}

// RelationDeleter is the relations' store as deleteIssueRelation reads
// and writes it, without a lock first, then in the transaction ctx
// carries (M4 design 3.5, 3.10).
type RelationDeleter interface {
	// GetRelation reads the undeleted relation id, as stored; found is
	// false when there is none.
	GetRelation(ctx context.Context, id uuid.UUID) (r domain.Relation, found bool, err error)
	// DeleteRelation soft-deletes the undeleted relation d names.
	DeleteRelation(ctx context.Context, d RelationDeletion) error
	// MarkChanged writes the undeleted work items r names last changed.
	MarkChanged(ctx context.Context, r ChangedRows) error
	// InsertActivities inserts the rows in their order.
	InsertActivities(ctx context.Context, rows []ActivityRow) error
}

// IssueArchiver is the work items' store as archiveIssue and
````

`server/internal/modules/issue/app/rows.go`（修改，1 处）：

````old server/internal/modules/issue/app/rows.go
}

// IssueArchival is a work item's archiving or unarchiving as it is written
````
````new server/internal/modules/issue/app/rows.go
}

// RelationDeletion is a relation's deletion as it is written (M4 design
// 3.10): the relation ID, soft-deleted at the moment and by the account
// given.
type RelationDeletion struct {
	ID uuid.UUID
	By uuid.UUID
	At time.Time
}

// IssueArchival is a work item's archiving or unarchiving as it is written
````

- [ ] **Step 4: 用例和它的测试**

`server/internal/modules/issue/app/delete_relation.go`（新文件，96 行）：

````file server/internal/modules/issue/app/delete_relation.go
package app

import (
	"context"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// DeleteIssueRelationDeps are what deleteIssueRelation reads and writes
// through.
type DeleteIssueRelationDeps struct {
	Locks     Locks
	Relations RelationDeleter
	Clock     Clock
}

// DeleteIssueRelation deletes a relation: DELETE
// /api/v0/issue-relations/{relation_id} (M4 design 3.5, 3.10, 5.1).
type DeleteIssueRelation struct {
	d DeleteIssueRelationDeps
}

// NewDeleteIssueRelation returns the use case.
func NewDeleteIssueRelation(d DeleteIssueRelationDeps) *DeleteIssueRelation {
	return &DeleteIssueRelation{d: d}
}

// Execute reads the relation id without a lock: one not there, or
// deleted, is issue.relation_not_found. Then Locks runs the write on
// either of its work items (M4 design 3.5, 3.10, convention 7): its
// precheck, before any lock, both work items read, and issue.update
// decided on the one stored as issue_id, then, when that refuses, on the
// other, the guest rule first: neither seen is issue.relation_not_found,
// else a role that may change neither forbidden; then the write gates of
// both their projects, all at once in ascending id order, which refuse a
// write that may wait no longer (503 server_busy); then in one
// transaction those projects FOR NO KEY UPDATE, in ascending id order,
// both work items read and decided on again, and under them, in this
// order: the relation read again, deleted meanwhile
// issue.relation_not_found; either work item of an archived project
// refuses it (409 project.archived), then either archived work item (409
// issue.archived); then the clock and the write (delete). All of it is
// the caller's at that moment.
func (u *DeleteIssueRelation) Execute(ctx context.Context, id uuid.UUID) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	r, found, err := u.d.Relations.GetRelation(ctx, id)
	switch {
	case err != nil:
		return err
	case !found:
		return domain.ErrRelationNotFound
	}
	w := write{issue: &r.IssueID, instead: &r.RelatedID, action: domain.ActionUpdate, notFound: domain.ErrRelationNotFound}
	return u.d.Locks.run(ctx, actor, w, func(ctx context.Context, h held) error {
		_, found, err := u.d.Relations.GetRelation(ctx, id)
		switch {
		case err != nil:
			return err
		case !found:
			return domain.ErrRelationNotFound
		case h.project.Archived || h.instead.project.Archived:
			return domain.ErrProjectArchived
		case h.issue.ArchivedAt != nil || h.instead.issue.ArchivedAt != nil:
			return domain.ErrIssueArchived
		}
		return u.delete(ctx, h, r, actor.UserID)
	})
}

// delete deletes the relation r between the work items h holds, the
// caller by's at the clock's moment (M4 design 3.7, 3.10, 3.13): the
// relation soft-deleted; both work items last changed by him, in id
// order; a row of each one's activity, of its project, the one stored as
// issue_id first, the type as it reads it, the other by its identifier
// (rows 22 and 23 of 3.13).
func (u *DeleteIssueRelation) delete(ctx context.Context, h held, r domain.Relation, by uuid.UUID) error {
	at := u.d.Clock.Now()
	if err := u.d.Relations.DeleteRelation(ctx, RelationDeletion{ID: r.ID, By: by, At: at}); err != nil {
		return err
	}
	first, second := end{issue: h.issue, project: h.project}, h.instead
	changed := ascending([]uuid.UUID{first.issue.ID, second.issue.ID})
	if err := u.d.Relations.MarkChanged(ctx, ChangedRows{IDs: changed, By: by, At: at}); err != nil {
		return err
	}
	rows := slices.Concat(
		first.stamp(by, at).activity([]domain.Activity{domain.RelationRemoved(r.TypeFrom(first.issue.ID), *refOf(second.project, second.issue))}),
		second.stamp(by, at).activity([]domain.Activity{domain.RelationRemoved(r.TypeFrom(second.issue.ID), *refOf(first.project, first.issue))}))
	return u.d.Relations.InsertActivities(ctx, rows)
}
````

`server/internal/modules/issue/app/delete_relation_test.go`（新文件，214 行）：

````file server/internal/modules/issue/app/delete_relation_test.go
package app_test

import (
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// relationID is the id of newUnrelate's relation.
var relationID = uuid.NewV7()

// newUnrelate is deleteIssueRelation over newRelate's world, with a
// relation of near and far stored: far blocked by near, stored from far,
// made by carol at earlier.
func newUnrelate() (*app.DeleteIssueRelation, *world, *fakeRelations) {
	_, w, rel, _, _ := newRelate()
	rel.relations = []domain.Relation{{ID: relationID, WorkspaceID: acme, ProjectID: ops, IssueID: farID, RelatedID: nearID,
		Type: domain.BlockedBy, CreatedByID: &carol, CreatedAt: earlier}}
	return unrelateOver(w, rel), w, rel
}

// unrelateOver is deleteIssueRelation over w's locks and rel.
func unrelateOver(w *world, rel *fakeRelations) *app.DeleteIssueRelation {
	return app.NewDeleteIssueRelation(app.DeleteIssueRelationDeps{Locks: w.locks, Relations: rel, Clock: clockAt{at: clockNow, log: w.log}})
}

// unrelating are the calls of alice's deletion of newUnrelate's relation,
// as Locks takes them on either work item (M4 design 3.5, 3.10): before
// any lock and outside the transaction, the relation read, both work items
// read, far first, stored as issue_id, then near, and the decision on far,
// in ops, then, when that refuses, on near, in web, before of them; in the
// transaction, web and ops locked at once, in id order, both work items
// read again, the decisions again, under of them, and the relation read
// again.
func unrelating(before, under int) []string {
	reads := []string{line("GetIssue", farID), line("GetIssue", nearID)}
	decisions := []string{line("Authorize", alice, domain.ActionUpdate, acme, ops), line("Authorize", alice, domain.ActionUpdate, acme, web)}
	var outside []string
	for _, c := range slices.Concat([]string{line("GetRelation", relationID)}, reads, decisions[:before]) {
		outside = append(outside, c+" outside tx")
	}
	return slices.Concat(outside, []string{"Begin", line("LockMany", anys(sorted([]uuid.UUID{web, ops}))...)}, reads, decisions[:under],
		[]string{line("GetRelation", relationID)})
}

// through is calls up to call, the last; all of them for "".
func through(calls []string, call string) []string {
	if call == "" {
		return calls
	}
	return calls[:slices.Index(calls, call)+1]
}

// archived archives the work item id of w's others.
func archived(w *world, id uuid.UUID) {
	i := w.issues.others[id]
	i.ArchivedAt = &earlier
	w.issues.others[id] = i
}

// A deletion of a relation, in the order of M4 design 3.5 and 3.10:
// Locks' calls on either work item (unrelating); the clock; the relation
// deleted; both work items marked changed, in id order; the activity. The
// caller may change either work item: far, stored as issue_id, decided on
// first, or, when he may not change it, near, decided on next, whether his
// role may not, or the guest rule hides far from him in ops. Each writes
// a row of far's activity first, of ops, its blocked_by WEB-8 deleted,
// then one of near's, of web, its blocking OPS-3 deleted, both alice's at
// the clock's moment.
func TestDeleteIssueRelationDeletesItOnEitherWorkItem(t *testing.T) {
	for _, tt := range []struct {
		name          string
		set           func(w *world)
		before, under int
	}{
		{"decided on far", func(*world) {}, 1, 1},
		{"far not allowed, decided on near", func(w *world) { w.auth.answers = []error{0: shared.Forbidden(), 2: shared.Forbidden()} }, 2, 2},
		{"far hidden by the guest rule, decided on near", func(w *world) {
			w.auth.grants = []shared.Grant{0: {OwnOnly: true}, 2: {OwnOnly: true}}
		}, 2, 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w, rel := newUnrelate()
			tt.set(w)
			if err := uc.Execute(as(alice), relationID); err != nil {
				t.Fatal(err)
			}
			changed := sorted([]uuid.UUID{nearID, farID})
			calls := slices.Concat(unrelating(tt.before, tt.under), []string{"Now", line("DeleteRelation", relationID), line("MarkChanged", changed),
				line("InsertActivities", 2)})
			if !slices.Equal(w.log.calls, calls) {
				t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
			}
			if want := (app.RelationDeletion{ID: relationID, By: alice, At: clockNow}); rel.deleted == nil || !reflect.DeepEqual(*rel.deleted, want) {
				t.Errorf("deleted %+v, want %+v", rel.deleted, want)
			}
			if want := (app.ChangedRows{IDs: changed, By: alice, At: clockNow}); rel.marked == nil || !reflect.DeepEqual(*rel.marked, want) {
				t.Errorf("marked %+v, want %+v", rel.marked, want)
			}
			rows := []app.ActivityRow{
				{IssueID: farID, ProjectID: ops, Activity: domain.RelationRemoved(domain.BlockedBy, domain.Ref{ID: nearID, Identifier: "WEB-8"})},
				{IssueID: nearID, ProjectID: web, Activity: domain.RelationRemoved(domain.Blocking, domain.Ref{ID: farID, Identifier: "OPS-3"})},
			}
			if len(w.issues.activities) != len(rows) {
				t.Fatalf("the activity %+v, want %d rows", w.issues.activities, len(rows))
			}
			for n, row := range w.issues.activities {
				want := rows[n]
				want.ID, want.WorkspaceID, want.ActorID, want.At = row.ID, acme, alice, clockNow
				if !reflect.DeepEqual(row, want) || n > 0 && row.ID.Compare(w.issues.activities[n-1].ID) <= 0 {
					t.Errorf("activity row %d: %+v, want %+v, its id after the row before", n, row, want)
				}
			}
		})
	}
}

// Every refusal of a deletion (M4 design 3.5, 3.10), none of which writes
// a row. Before any lock and outside the transaction: no relation, or a
// work item of it not there, issue.relation_not_found; neither work item
// seen, by its project or by the guest rule, issue.relation_not_found too,
// and a role that may change neither, when he sees one, forbidden. Under
// the locks, the decision is final: either project deleted while they
// waited, before any decision, either work item deleted meanwhile, or the
// relation, or neither seen any more, issue.relation_not_found; neither to
// be changed any more, forbidden; either work item of an archived project
// 409 project.archived, then either archived work item 409 issue.archived.
// Each answer is compared whole: a relation neither of whose work items
// the caller sees is the very issue.relation_not_found of one not there,
// detail and all (8.4).
func TestDeleteIssueRelationRefusals(t *testing.T) {
	notFound, forbidden, hidden := domain.ErrRelationNotFound, shared.Forbidden(), shared.ErrNotVisible
	update := func(project uuid.UUID) string { return line("Authorize", alice, domain.ActionUpdate, acme, project) }
	for _, tt := range []struct {
		name          string
		set           func(w *world, rel *fakeRelations)
		want          error
		before, under int
		last          string // the last call of unrelating's, "" for all
	}{
		{"no relation", func(_ *world, rel *fakeRelations) { rel.relations = nil }, notFound, 0, 0,
			line("GetRelation", relationID, "outside tx")},
		{"far not there", func(w *world, _ *fakeRelations) { delete(w.issues.others, farID) }, notFound, 0, 0,
			line("GetIssue", farID, "outside tx")},
		{"near not there", func(w *world, _ *fakeRelations) { delete(w.issues.others, nearID) }, notFound, 0, 0,
			line("GetIssue", nearID, "outside tx")},
		{"neither seen", func(w *world, _ *fakeRelations) { w.auth.answers = []error{hidden, hidden} }, notFound, 2, 0, update(web) + " outside tx"},
		{"far not seen, near hidden by the guest rule", func(w *world, _ *fakeRelations) {
			w.auth.answers, w.auth.grants = []error{hidden}, []shared.Grant{1: {OwnOnly: true}}
		}, notFound, 2, 0, update(web) + " outside tx"},
		{"neither to be changed", func(w *world, _ *fakeRelations) { w.auth.answers = []error{forbidden, forbidden} }, forbidden, 2, 0,
			update(web) + " outside tx"},
		{"far not seen, near not to be changed", func(w *world, _ *fakeRelations) { w.auth.answers = []error{hidden, forbidden} }, forbidden, 2, 0,
			update(web) + " outside tx"},
		{"far not to be changed, near not seen", func(w *world, _ *fakeRelations) { w.auth.answers = []error{forbidden, hidden} }, forbidden, 2, 0,
			update(web) + " outside tx"},
		{"ops deleted while the locks waited, near no longer to be changed", func(w *world, _ *fakeRelations) {
			w.project.waited = func() { delete(w.project.across, ops) }
			w.auth.answers = []error{1: hidden, 2: forbidden}
		}, notFound, 1, 0, line("GetIssue", nearID)},
		{"web deleted while the locks waited, far no longer to be changed", func(w *world, _ *fakeRelations) {
			w.project.waited = func() { w.project.locked = nil }
			w.auth.answers = []error{1: forbidden, 2: hidden}
		}, notFound, 1, 0, line("GetIssue", nearID)},
		{"far deleted meanwhile", func(w *world, _ *fakeRelations) { w.project.waited = func() { delete(w.issues.others, farID) } }, notFound, 1, 0,
			line("GetIssue", farID)},
		{"the relation deleted meanwhile", func(w *world, rel *fakeRelations) { w.project.waited = func() { rel.relations = nil } }, notFound, 1, 1,
			""},
		{"neither seen any more", func(w *world, _ *fakeRelations) { w.auth.answers = []error{1: hidden, 2: hidden} }, notFound, 1, 2, update(web)},
		{"neither to be changed any more", func(w *world, _ *fakeRelations) { w.auth.answers = []error{1: forbidden, 2: forbidden} }, forbidden, 1,
			2, update(web)},
		{"ops archived", func(w *world, _ *fakeRelations) {
			w.project.across[ops] = app.LockedProject{WorkspaceID: acme, Archived: true, Identifier: "OPS"}
		}, domain.ErrProjectArchived, 1, 1, ""},
		{"web archived", func(w *world, _ *fakeRelations) { w.project.locked.Archived = true }, domain.ErrProjectArchived, 1, 1, ""},
		{"far archived", func(w *world, _ *fakeRelations) { archived(w, farID) }, domain.ErrIssueArchived, 1, 1, ""},
		{"near archived", func(w *world, _ *fakeRelations) { archived(w, nearID) }, domain.ErrIssueArchived, 1, 1, ""},
		{"web archived, far archived too", func(w *world, _ *fakeRelations) {
			w.project.locked.Archived = true
			archived(w, farID)
		}, domain.ErrProjectArchived, 1, 1, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w, rel := newUnrelate()
			tt.set(w, rel)
			err := uc.Execute(as(alice), relationID)
			wantRefusal(t, w, err, tt.want, through(unrelating(tt.before, tt.under), tt.last))
			if !reflect.DeepEqual(err, tt.want) {
				t.Errorf("Execute() = %#v, want %#v whole", err, tt.want)
			}
			if rel.deleted != nil || rel.marked != nil || w.issues.activities != nil {
				t.Errorf("wrote %+v, %+v, %+v; want nothing", rel.deleted, rel.marked, w.issues.activities)
			}
		})
	}
}

// A port that fails ends the deletion with its failure, as itself, and no
// call after it runs: the transaction rolls back whatever was written.
func TestDeleteIssueRelationStopsAtAFailingPort(t *testing.T) {
	for _, port := range []string{"GetRelation", "GetIssue", line("GetIssue", nearID), "Authorize", "LockMany", "DeleteRelation", "MarkChanged",
		"InsertActivities"} {
		t.Run(port, func(t *testing.T) {
			uc, w, _ := newUnrelate()
			w.log.fail = port
			wantStoppedAt(t, w, uc.Execute(as(alice), relationID), port)
		})
	}
}
````

`server/internal/modules/issue/app/fakes_relations_test.go`（修改，2 处）：

````old server/internal/modules/issue/app/fakes_relations_test.go
// (app.RelationCreator, app.RelationLister), each call logged: related
// are the work items that relate to the work item asked about already,
// whichever it is; relations the relations a read answers, and the last
// insert's; marked the last work items marked changed.
````
````new server/internal/modules/issue/app/fakes_relations_test.go
// (app.RelationCreator, app.RelationLister, app.RelationDeleter), each
// call logged: related are the work items that relate to the work item
// asked about already, whichever it is; relations the relations a read
// answers, and the last insert's; marked the last work items marked
// changed; deleted the last relation deleted.
````
````old server/internal/modules/issue/app/fakes_relations_test.go
	marked    *app.ChangedRows
````
````new server/internal/modules/issue/app/fakes_relations_test.go
	marked    *app.ChangedRows
	deleted   *app.RelationDeletion
}

// GetRelation logs the read and answers the fake's relation id, as it is;
// found is false when it has none.
func (f *fakeRelations) GetRelation(ctx context.Context, id uuid.UUID) (domain.Relation, bool, error) {
	if err := f.log.add(ctx, "GetRelation", id); err != nil {
		return domain.Relation{}, false, err
	}
	at := slices.IndexFunc(f.relations, func(r domain.Relation) bool { return r.ID == id })
	if at < 0 {
		return domain.Relation{}, false, nil
	}
	return f.relations[at], true, nil
}

// DeleteRelation logs the deletion, of which relation, and keeps it.
func (f *fakeRelations) DeleteRelation(ctx context.Context, d app.RelationDeletion) error {
	if err := f.log.add(ctx, "DeleteRelation", d.ID); err != nil {
		return err
	}
	f.deleted = &d
	return nil
````

- [ ] **Step 5: 存储**

`server/internal/modules/issue/adapter/postgres/relations.go`（修改，4 处）：

````old server/internal/modules/issue/adapter/postgres/relations.go
	"context"
````
````new server/internal/modules/issue/adapter/postgres/relations.go
	"context"
	"errors"
````
````old server/internal/modules/issue/adapter/postgres/relations.go
	"uuid"
````
````new server/internal/modules/issue/adapter/postgres/relations.go
	"uuid"

	"github.com/jackc/pgx/v5"
````
````old server/internal/modules/issue/adapter/postgres/relations.go
// relationOf is a relation's row as the domain has it.
````
````new server/internal/modules/issue/adapter/postgres/relations.go
// GetRelation reads the undeleted relation id, as stored; found is false
// when there is none.
func (s *Store) GetRelation(ctx context.Context, id uuid.UUID) (domain.Relation, bool, error) {
	r, err := s.queries(ctx).GetRelation(ctx, id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.Relation{}, false, nil
	case err != nil:
		return domain.Relation{}, false, fmt.Errorf("read relation %s: %w", id, err)
	}
	return relationOf(gen.IssueRelationsRow(r)), true, nil
}

// relationOf is a relation's row as the domain has it: IssueRelations'
// row, and GetRelation's, of the same columns, converted.
````
````old server/internal/modules/issue/adapter/postgres/relations.go
		Type: domain.RelationType(r.RelationType), CreatedByID: r.CreatedByID, CreatedAt: r.CreatedAt}
````
````new server/internal/modules/issue/adapter/postgres/relations.go
		Type: domain.RelationType(r.RelationType), CreatedByID: r.CreatedByID, CreatedAt: r.CreatedAt}
}

// DeleteRelation soft-deletes the undeleted relation d names, at its
// moment and by its account (M4 design 3.10). The use case read it
// undeleted under the locks of its work items' projects, so a relation
// not written is an error.
func (s *Store) DeleteRelation(ctx context.Context, d app.RelationDeletion) error {
	n, err := s.queries(ctx).DeleteRelation(ctx, gen.DeleteRelationParams{ID: d.ID, By: d.By, At: d.At})
	switch {
	case err != nil:
		return fmt.Errorf("delete relation %s: %w", d.ID, err)
	case n != 1:
		return fmt.Errorf("deleted %d rows of relation %s", n, d.ID)
	}
	return nil
````

`server/internal/modules/issue/adapter/postgres/relations_test.go`（修改，1 处）：

````old server/internal/modules/issue/adapter/postgres/relations_test.go
		t.Errorf("Relations() on a cancelled context = %v; want context.Canceled", err)
	}
}

````
````new server/internal/modules/issue/adapter/postgres/relations_test.go
		t.Errorf("Relations() on a cancelled context = %v; want context.Canceled", err)
	}
}

// A relation is read by its id as stored (M4 design 3.10): ops' o blocked
// by web's item, made by bob a second after the moment, every field as
// stored. One deleted, item's relation to b, or one not there, is not
// found; a cancelled read is its failure.
func TestGetRelation(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, ops := newProject(t, pool, acme, "WEB"), newProject(t, pool, acme, "OPS")
	item, b, o, gone := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	newIssue(t, pool, web, item, 1, alice)
	newIssue(t, pool, web, b, 2, alice)
	newIssue(t, pool, ops, o, 1, alice)
	want := domain.Relation{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: ops.id, IssueID: o, RelatedID: item, Type: domain.BlockedBy,
		CreatedByID: &bob, CreatedAt: now.Add(time.Second)}
	newRelation(t, pool, ops, o, item, want.ID, bob)
	exec(t, pool, "UPDATE issue_relations SET relation_type = $2, created_at = $3 WHERE id = $1", want.ID, string(want.Type), want.CreatedAt)
	newRelation(t, pool, web, item, b, gone, alice)
	exec(t, pool, "UPDATE issue_relations SET deleted_at = $2 WHERE id = $1", gone, now)

	if got, found, err := s.GetRelation(context.Background(), want.ID); err != nil || !found || !reflect.DeepEqual(got, want) {
		t.Errorf("GetRelation() = %+v, %v, %v; want %+v", got, found, err, want)
	}
	for _, id := range []uuid.UUID{gone, uuid.NewV7()} {
		if got, found, err := s.GetRelation(context.Background(), id); err != nil || found {
			t.Errorf("GetRelation(%s) = %+v, %v, %v; want none", id, got, found, err)
		}
	}
	cancelled, stop := context.WithCancel(context.Background())
	stop()
	if _, _, err := s.GetRelation(cancelled, want.ID); !errors.Is(err, context.Canceled) {
		t.Errorf("GetRelation() on a cancelled context = %v; want context.Canceled", err)
	}
}

// A relation is soft-deleted at the moment and by the account given (M4
// design 3.10), every other column as it was: item's relation to b, by bob
// at later; its relation to o is as it was. One deleted already, or one
// not there, is not written, and is an error; so is a cancelled write.
func TestDeleteRelation(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, ops := newProject(t, pool, acme, "WEB"), newProject(t, pool, acme, "OPS")
	item, b, o, toB, toO := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	newIssue(t, pool, web, item, 1, alice)
	newIssue(t, pool, web, b, 2, alice)
	newIssue(t, pool, ops, o, 1, alice)
	newRelation(t, pool, web, item, b, toB, alice)
	newRelation(t, pool, web, item, o, toO, alice)
	rows, stamps := rowsAsText(t, pool, "issue_relations"), audits(t, pool, "issue_relations")
	later := now.Add(time.Hour)

	if err := s.DeleteRelation(context.Background(), app.RelationDeletion{ID: toB, By: bob, At: later}); err != nil {
		t.Fatal(err)
	}

	if after := rowsAsText(t, pool, "issue_relations"); !maps.Equal(after, rows) {
		t.Errorf("the relations but their audit columns:\n%v\nwant them as they were:\n%v", after, rows)
	}
	at := later.UTC().Format(time.RFC3339Nano)
	stamps[toB] = fmt.Sprintf("deleted at %s, written at %s by %s", at, at, bob)
	if after := audits(t, pool, "issue_relations"); !maps.Equal(after, stamps) {
		t.Errorf("the audit columns:\n%v\nwant\n%v", after, stamps)
	}
	for _, id := range []uuid.UUID{toB, uuid.NewV7()} {
		if err := s.DeleteRelation(context.Background(), app.RelationDeletion{ID: id, By: alice, At: later.Add(time.Hour)}); err == nil {
			t.Errorf("DeleteRelation(%s) = nil, want an error", id)
		}
		if after := audits(t, pool, "issue_relations"); !maps.Equal(after, stamps) {
			t.Errorf("DeleteRelation(%s): the audit columns:\n%v\nwant them as they were:\n%v", id, after, stamps)
		}
	}
	cancelled, stop := context.WithCancel(context.Background())
	stop()
	if err := s.DeleteRelation(cancelled, app.RelationDeletion{ID: toO, By: bob, At: later}); !errors.Is(err, context.Canceled) {
		t.Errorf("DeleteRelation() on a cancelled context = %v; want context.Canceled", err)
	}
}

````

- [ ] **Step 6: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/issue/...`
Expected: 全部 `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 7: 提交**

```bash
git add server/internal/modules/issue/adapter/postgres/queries/relations.sql server/internal/modules/issue/adapter/postgres/relations.go server/internal/modules/issue/adapter/postgres/relations_test.go server/internal/modules/issue/app/delete_relation.go server/internal/modules/issue/app/delete_relation_test.go server/internal/modules/issue/app/fakes_relations_test.go server/internal/modules/issue/app/lock.go server/internal/modules/issue/app/lock_across.go server/internal/modules/issue/app/lock_either.go server/internal/modules/issue/app/lock_test.go server/internal/modules/issue/app/ports.go server/internal/modules/issue/app/rows.go server/internal/modules/issue/domain/activity.go server/internal/modules/issue/domain/errors.go server/internal/modules/issue/domain/relation.go server/internal/modules/issue/domain/relation_test.go server/internal/modules/issue/adapter/postgres/gen/relations.sql.go
```
```bash
git commit -m "feat(M4/P3): deleteIssueRelation's use case, on either of the relation's work items

Locks.run gains its form on either of two work items: issue.update is
decided on the end stored as issue_id, then, when that refuses, on the
other, the guest rule first; neither seen is issue.relation_not_found,
else a role that may change neither is forbidden. Both work items are
read and both projects locked all at once in ascending id order. The use
case refuses an archived project or work item at either end, deletes the
relation, marks both ends changed and writes a row on each.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p3.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试，或没有替身的平台、领域、转换的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；守卫是生成器（`make gen-go`）和中英文翻译键（`make lint-web`）的核对。每个变异在一份新的副本里，在本 Task 的树上、标出的较晚的 Task 的树上和最终的树上都跑过，端到端的检查只在标出的 Task 的树上）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T13.1` | 只有能改存为 `issue_id` 的那个工作项的人才能删关联 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestARelationIsDeletedFromEitherWorkItem`（`make test`）（Task 14 起） | 单元；组合 |
| `T13.2` | 第一个工作项的拒绝就是删除的回答，不看另一个的判定 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestARelationIsDeletedFromEitherWorkItem`（`make test`）（Task 14 起） | 单元；组合 |
| `T13.3` | 两个工作项都看不到的关联答第一个的 404，而不是关联的 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWhoMayNotDeleteARelation`（`make test`）（Task 14 起）、`internal/bootstrap` 的 `TestTheGuestRuleHoldsForDeletingARelation`（`make test`）（Task 14 起）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `deleteIssueRelation` 各格（`make test`）（Task 14 起） | 单元；组合 |
| `T13.4` | 锁等待期间一个工作项的项目被删除，删除照样进行 | `internal/modules/issue/app` 的全部测试（`make test`） | 单元 |
| `T13.5` | 删除只进第一个工作项的项目的闸门、只取它的锁 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestARelationIsDeletedFromEitherWorkItem`（`make test`）（Task 14 起） | 单元；组合 |
| `T13.6` | 两个工作项任一个上的写只经 `Lock` 锁第一个项目 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestARelationIsDeletedFromEitherWorkItem`（`make test`）（Task 14 起） | 单元；组合 |
| `T13.7` | 两个工作项任一个上的写只在第一个上预检 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestARelationIsDeletedFromEitherWorkItem`（`make test`）（Task 14 起） | 单元；组合 |
| `T13.8` | 两个工作项任一个上的写在锁下哪个都不重读 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestARelationIsDeletedFromEitherWorkItem`（`make test`）（Task 14 起） | 单元；组合 |
| `T13.9` | 锁等待期间已删除的关联再删一次，而不是答"不存在" | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestTwoDeletionsOfARelationSerialize`（`make test`）（Task 14 起） | 单元；组合 |
| `T13.10` | 对方工作项的项目已归档的关联照样删除 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWhoMayNotDeleteARelation`（`make test`）（Task 14 起） | 单元；组合 |
| `T13.11` | 对方工作项已归档的关联照样删除 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWhoMayNotDeleteARelation`（`make test`）（Task 14 起）、故事 `i15-archive-issue`（Task 15 起） | 单元；组合；端到端 |
| `T13.12` | 已归档的项目里已归档的工作项先答"工作项已归档" | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestPermissionMatrix` 的 `deleteIssueRelation` 各格（`make test`）（Task 14 起） | 单元；组合 |
| `T13.13` | 删除只改存为 `issue_id` 的工作项的最后修改，不改对方 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestARelationIsDeletedFromEitherWorkItem`（`make test`）（Task 14 起）、故事 `i9-relations`（Task 15 起） | 单元；组合；端到端 |
| `T13.14` | 删除不在对方的动态里写行 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestARelationIsDeletedFromEitherWorkItem`（`make test`）（Task 14 起）、故事 `i9-relations`（Task 15 起） | 单元；组合；端到端 |
| `T13.15` | 对方的那行动态写存为 `issue_id` 的工作项读到的类型 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestARelationIsDeletedFromEitherWorkItem`（`make test`）（Task 14 起）、故事 `i9-relations`（Task 15 起） | 单元；组合；端到端 |
| `T13.16` | 存为 `issue_id` 的工作项的那行动态用它自己项目的标识称呼对方 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestARelationIsDeletedFromEitherWorkItem`（`make test`）（Task 14 起） | 单元；组合 |
| `T13.17` | 删除关联的那行动态说它是修改 | `internal/modules/issue/domain` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestARelationIsDeletedFromEitherWorkItem`（`make test`）（Task 14 起）、故事 `i9-relations`（Task 15 起） | 单元；组合；端到端 |
| `T13.18` | 删除关联的那行动态把对方写成新值 | `internal/modules/issue/domain` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestARelationIsDeletedFromEitherWorkItem`（`make test`）（Task 14 起）、故事 `i9-relations`（Task 15 起） | 单元；组合；端到端 |
| `T13.19` | 已删除的关联读成还在 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestARelationIsDeletedFromEitherWorkItem`（`make test`）（Task 14 起） | 存储；组合 |
| `T13.20` | `DeleteRelation` 把没删到的关联当作删了 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`） | 存储 |
| `T13.21` | `DeleteRelation` 再删一次已删除的关联 | `internal/modules/issue/adapter/postgres` 的全部测试（`make test`） | 存储 |
| `T13.22` | 写两端的最后修改失败时略过 | `internal/modules/issue/app` 的全部测试（`make test`） | 单元 |
| `T13.23` | 两个工作项任一个上的写不进任何闸门 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestEachWriteOnAWorkItemOfAProjectTooBusyAnswersServerBusy`（`make test`）（Task 14 起） | 单元；组合 |
| `T13.24` | `DeleteRelation` 经连接池写，不在它的事务的连接上 | `internal/bootstrap` 的 `TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection`（`make test`）（Task 14 起） | 组合 |
| `T13.25` | 两个工作项任一个上的写只锁住一端的项目（id 较小的，交错 26 里即通过判定的那一端）：另一端的项目经连接池读、锁随即放开 | `internal/modules/issue/app` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestDeletingARelationAndItsOtherEndSerialize`（`make test`）（Task 14 起）、`internal/bootstrap` 的 `TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection`（`make test`）（Task 14 起） | 单元；组合 |

---

### Task 14: 删除关联：接口、组合、矩阵；交错 10、11、23、26 和加关联与删除工作区

**Files:**
- Create: `server/internal/bootstrap/issue_interleaving_relations_test.go`、`server/internal/bootstrap/issue_relation_deletes_test.go`
- Modify: `api/modules/issue.yaml`、`api/openapi.yaml`、`server/internal/bootstrap/gate_test.go`、`server/internal/bootstrap/interleaving_test.go`、`server/internal/bootstrap/issue_interleaving_test.go`、`server/internal/bootstrap/issue_write_locks_test.go`、`server/internal/bootstrap/issue_writes_test.go`、`server/internal/bootstrap/permission_matrix_relations_test.go`、`server/internal/bootstrap/permission_matrix_seed_issues_test.go`、`server/internal/bootstrap/permission_matrix_seed_relations_test.go`、`server/internal/bootstrap/permission_matrix_targets_test.go`、`server/internal/bootstrap/permission_matrix_test.go`、`server/internal/modules/issue/adapter/http/handler.go`、`server/internal/modules/issue/adapter/http/handler_test.go`、`server/internal/modules/issue/adapter/http/relations.go`、`server/internal/modules/issue/adapter/http/relations_test.go`、`server/internal/modules/issue/module.go`、`web/apps/web/core/lib/error-messages.ts`、`web/packages/i18n/src/locales/en/errors.json`、`web/packages/i18n/src/locales/zh-CN/errors.json`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/issue/adapter/http/gen/server.gen.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**（spec 2.16、第 3 节第 3、4、8、9、15、16、17 条；M4 设计 3.5、3.6、3.10、5.1、9.2、9.3，12 节约束 4）
- 契约：`DELETE /api/v0/issue-relations/{relation_id}`（`deleteIssueRelation`，204，`x-problem-codes: [issue.relation_not_found, forbidden, project.archived, issue.archived, server_busy]`），参数 `RelationID`；描述写上两端任一个的判定、看不到两端是 404、两端的归档、两行动态、两端的最后修改；`api/openapi.yaml` 的路径。
- 前端（约束 4）：`issue.relation_not_found` 的文案进 `error-messages.ts` 的 `PROBLEM_MESSAGES` 和两份 `errors.json`（en "The relation does not exist, or you cannot see either of its work items."，zh-CN "关联不存在，或者它的两个工作项你都看不到。"）。
- `issue/adapter/http`：`handler.go` 的 `DeleteIssueRelationUseCase`；`relations.go` 的 `DeleteIssueRelation`（204）；`handler_test.go` 的测试的用例集。`issue/module.go` 接上，说明写上它。
- `bootstrap`：`issue_relation_deletes_test.go`（新：`relationPath`、`relationDeletion`）；交错的帮手扩展、不复制（第 3 节第 16 条）：`interleaving_test.go` 的 `serializeOnTable`（第二个写等的是给的表的一行；`serializeOn` 是它在 `projects` 上），`issue_interleaving_test.go` 的 `serializeAt`、`inBothOrdersAt`（`serialize`、`inBothOrders` 是它们在 `projects` 上），`issue_interleaving_relations_test.go`（新：`nearAndFar`、`projectWith`、`deletedAt`、`relationRowOf`、`relates`、`unrelates`、`deletesOps`，名字找工作项用 Task 3 的 `workItemNamed`，删除工作项用 Task 6 的 `deletesNamed`）；`issue_write_locks_test.go` 的 `deleteIssueRelation` 一行（锁下第一个写的表 `issue_relations`，`SHARE`）；`gate_test.go` 的关联的删除一行；`issue_writes_test.go` 的 `TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection`：alice 删除她建的关联；矩阵：`permission_matrix_seed_relations_test.go`（整个文件：加"别人的"（Others'：公开项目的 `matrixIssue` 与"管理员·访客"的）、已归档的 Shelved、已删除的工作区的 Gone，`relatedItems` 是 Gone 的第二个工作项，`relationOf` 是每一列的格子删除的那一条），`permission_matrix_seed_issues_test.go` 种下 `relatedItems`，`permission_matrix_targets_test.go` 的 `{relation_id}` 目标（`relationOf`，它的列的项目里种下的），`permission_matrix_relations_test.go` 的 `deleteIssueRelation` 四行（`cellRelationNotFound`、`relationRows`、`toRelation`、`onARelation`）和列表的格子多了"别人的"、Shelved，`permission_matrix_test.go` 的种子说明。

**Tests:**
- HTTP（`apitest.Main` 两个方向）：`TestDeleteIssueRelation`（204，没有体）、`TestDeleteIssueRelationRefusals`。
- 组合：`TestARelationIsDeletedFromEitherWorkItem`（bob 不是 Ops 的成员：他的删除先判定 Far（看得到、不能改），再判定 Near（能改）：204；关联由 bob 在两端最后修改的时刻删除，各一行动态（在自己的项目、按自己读的类型、对方的标识），谁都不再列出它，再删 404）、`TestWhoMayNotDeleteARelation`（walt 是 acme 的成员、不是 Web 的：看得到 Near、Up 而都不能改 403；不是 acme 成员的 olga 看不到两端 404；walt 看不到 Lab 的两个工作项的关联 404，都与不存在的逐值相同；已归档的 Shelved 的关联从两端删都 409，已归档的项目 Old 的工作项的关联 409；都不写）、`TestTheGuestRuleHoldsForDeletingARelation`（walt 删 Hers 到 His 的关联：先判定访客规则隐藏的 Hers，再判定 His，204；Up 与 Hers 的关联对他是与不存在的逐值相同的 404）；交错（两种顺序，第二个都等在第一个持的行上，都有期限）：10 `TestOppositeRelationsOfTwoWorkItemsSerialize`（alice 把 Near 关联为阻塞 Far、bob 把 Far 关联为阻塞 Near：两个都一次锁住 Web、Ops，第二个在锁下读到第一个的关联，422 `duplicate`，什么都不写）、11 `TestDeletingARelatedWorkItemAndTheOtherEndsProjectSerialize`（bob 删除他的 Near、alice 删除 Ops：各锁自己的项目，再按 `id` 升序锁要删的关联行；第二个等第一个的关联行，提交之后读到已删除、跳过：每条只删一次，是先删的那一个的时刻和删除者）、23 `TestRelatingAWorkItemAndDeletingTheOtherEndsProjectSerialize`（bob 关联 Near 到 Far、alice 删除 Ops：先删，关联在锁下读到 Ops 已删除，Far 没到达，422，什么都不写；先关联，删除等 Ops 的行，再在它的时刻删除那条关联（Ops 的行）和 Far 的那行动态；Near 的那行留着，没有指向已删除的工作项的未删除的关联）、`TestRelatingAcrossProjectsAndDeletingTheWorkspaceSerialize`（P1a 的交接：bob 关联 Near、Far，alice 删除 acme：先删，关联在锁下读到自己的项目已删除，404，什么都不写；先关联，删除在 acme 的时刻删除它和别的每一行）、`TestTwoDeletionsOfARelationSerialize`（alice、bob 各删除 Near 到 Far 的关联：第二个等第一个的项目的行，锁下再读关联、已删除，404 `issue.relation_not_found`，什么都不写；关联只删一次，是第一个的删除者，两端各一行"删除"的动态，在同一时刻）、26 `TestDeletingARelationAndItsOtherEndSerialize`（bob 删除他做的 Near 到 Far 的关联（判定在 Near 上放行）、alice 删除 Far：关联的删除不论在哪一端判定都锁两端的项目，第二个等在 Ops 的行上；先删关联，Far 的删除读到它已删除、留着 bob 删的样子；先删 Far，关联的删除在锁下再读，它已随 Far 由 alice 在 Far 的时刻删除，404 `issue.relation_not_found`，什么都不写）；`TestEachIssueWriteLocksItsProjectFirst`、`TestEachWriteOnAWorkItemOfAProjectTooBusyAnswersServerBusy` 的一行；`TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection`（一个连接的池：alice 删除她建的关联，锁下的读和写都在它的事务的连接上）；`TestPermissionMatrix`（`deleteIssueRelation` 四行：照常（两端任一个的判定；看不到两端的列 `issue.relation_not_found`）、已归档的项目、已归档的项目里已归档的工作项、已归档的工作项）、`TestThePermissionMatrixCoversEveryOperation`。
- 前端：`error-messages.test.ts`（文案表有每个码），`check:sync`（中文的键）。
- `-race -count=5`：P3 的交错（9 的两对、10、11、13、23、26、加关联与删除工作区、两个删除同一条关联）、加关联不锁它到不了的项目、P2、P1b、P1a 的工作项一侧的交错、写闸门的组合测试（拒绝答 `server_busy`、事务等锁到期、项目太忙、每个写进闸门、请求在闸门上到期、跨项目的写一共等 `max_wait`、跨项目的写放回它占的每个位置）、连接池的测试（交错 20 的两个、写在事务的连接上）、取锁形状，没有 40P01；`platform/gate` 的单元测试同样五遍。

- [ ] **Step 1: 契约，生成**

`api/modules/issue.yaml`（修改，2 处）：

````old api/modules/issue.yaml
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/workspaces/{slug}/issues/{identifier}:
````
````new api/modules/issue.yaml
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/issue-relations/{relation_id}:
    parameters:
      - $ref: '#/components/parameters/RelationID'
    delete:
      operationId: deleteIssueRelation
      tags: [issue]
      summary: Delete a relation
      description: >-
        For whoever may change either of its work items, as updateIssue:
        the admins and members of the project of one of them, each work
        item decided on its own project, the guest rule first. A relation
        that does not exist, is deleted, or neither of whose work items the
        caller sees answers issue.relation_not_found, whatever his role may
        do; one of which he sees a work item but may change neither,
        forbidden. Both are answered before any project is locked. Then the
        projects of both work items are locked together, and his role is
        decided again. Either work item of an archived project answers
        project.archived; then either archived work item, issue.archived.
        Both work items are changed last by the caller, and the activity of
        each says it no longer relates to the other, with the type it
        read. Every row it writes is the caller's, at one moment. A project
        with too many writes on its work items at once answers server_busy,
        with Retry-After: try again then.
      security: [{bearer: []}]
      x-problem-codes: [issue.relation_not_found, forbidden, project.archived, issue.archived, server_busy]
      responses:
        '204':
          description: The relation is deleted.
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/workspaces/{slug}/issues/{identifier}:
````
````old api/modules/issue.yaml
      description: A work item's id (IssueDetail.id).
````
````new api/modules/issue.yaml
      description: A work item's id (IssueDetail.id).
      schema:
        type: string
        format: uuid
    RelationID:
      name: relation_id
      in: path
      required: true
      description: A relation's id (IssueRelation.id).
````

`api/openapi.yaml`（修改，1 处）：

````old api/openapi.yaml
    $ref: 'modules/issue.yaml#/paths/~1api~1v0~1issues~1{issue_id}~1relations'
````
````new api/openapi.yaml
    $ref: 'modules/issue.yaml#/paths/~1api~1v0~1issues~1{issue_id}~1relations'
  /api/v0/issue-relations/{relation_id}:
    $ref: 'modules/issue.yaml#/paths/~1api~1v0~1issue-relations~1{relation_id}'
````

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `e3bedae7dcc78320a389b284243b4f5f9503169dd580b3effd174b92257fff69` | 3672 | `api/dist/openapi.yaml` |
| `bf50cd56ab23f6f2a2ac8d1e05784c09f398e53c09e1d7881edc5ca99dc5c923` | 1778 | `server/internal/modules/issue/adapter/http/gen/server.gen.go` |
| `625294cd27ca3ac718aebb8859a63a734ffbd8dd4e7d80f74815ddc2eea8d580` | 4014 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/issue/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 2: HTTP、接线和文案**

`server/internal/modules/issue/adapter/http/handler.go`（修改，2 处）：

````old server/internal/modules/issue/adapter/http/handler.go
}

// UseCases are the use cases behind the module's operations.
````
````new server/internal/modules/issue/adapter/http/handler.go
}

// DeleteIssueRelationUseCase is app.DeleteIssueRelation.
type DeleteIssueRelationUseCase interface {
	Execute(ctx context.Context, id uuid.UUID) error
}

// UseCases are the use cases behind the module's operations.
````
````old server/internal/modules/issue/adapter/http/handler.go
	CreateIssueRelations CreateIssueRelationsUseCase
````
````new server/internal/modules/issue/adapter/http/handler.go
	CreateIssueRelations CreateIssueRelationsUseCase
	DeleteIssueRelation  DeleteIssueRelationUseCase
````

`server/internal/modules/issue/adapter/http/handler_test.go`（修改，1 处）：

````old server/internal/modules/issue/adapter/http/handler_test.go
// fakeDelete is deleteIssue: each call is recorded as "caller work item";
// it answers err.
````
````new server/internal/modules/issue/adapter/http/handler_test.go
// fakeDelete is deleteIssue, or deleteIssueRelation: each call is
// recorded as "caller work item", or "caller relation"; it answers err.
````

`server/internal/modules/issue/adapter/http/relations.go`（修改，1 处）：

````old server/internal/modules/issue/adapter/http/relations.go
}

// issueRelationCollection is relations as the API lists them, in their
````
````new server/internal/modules/issue/adapter/http/relations.go
}

// DeleteIssueRelation serves DELETE /api/v0/issue-relations/{relation_id}.
func (h handler) DeleteIssueRelation(ctx context.Context, req gen.DeleteIssueRelationRequestObject) (gen.DeleteIssueRelationResponseObject,
	error) {
	if err := h.uc.DeleteIssueRelation.Execute(ctx, req.RelationID); err != nil {
		return nil, err
	}
	return gen.DeleteIssueRelation204Response{}, nil
}

// issueRelationCollection is relations as the API lists them, in their
````

`server/internal/modules/issue/adapter/http/relations_test.go`（修改，1 处）：

````old server/internal/modules/issue/adapter/http/relations_test.go
					tt.status, tt.want)
			}
		}
	}
}

````
````new server/internal/modules/issue/adapter/http/relations_test.go
					tt.status, tt.want)
			}
		}
	}
}

// relationPath is the path of WEB-6's relation to WEB-8 (relatedAnswers).
var relationPath = "/api/v0/issue-relations/0199a2b4-0000-7000-8000-0000000000f1"

// DELETE goes to the use case for the caller and the path's relation; the
// answer is 204, without a body.
func TestDeleteIssueRelation(t *testing.T) {
	deletion := &fakeDelete{}
	res, body := do(t, newServer(t, httpadapter.UseCases{DeleteIssueRelation: deletion}), request(http.MethodDelete, relationPath, "alice", ""))
	if res.StatusCode != http.StatusNoContent || body != "" {
		t.Errorf("DELETE = %d %q, want 204 without a body", res.StatusCode, body)
	}
	if want := []string{aliceID.String() + " 0199a2b4-0000-7000-8000-0000000000f1"}; !slices.Equal(deletion.calls, want) {
		t.Errorf("calls %q, want %q", deletion.calls, want)
	}
}

// The use case's refusals, as the contract declares them, and its failure.
func TestDeleteIssueRelationRefusals(t *testing.T) {
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"no relation", domain.ErrRelationNotFound, http.StatusNotFound, `{"status":404,"code":"issue.relation_not_found","title":"Not Found",` +
			`"detail":"The relation does not exist, or you cannot see either of its work items."}`},
		{"a project guest of both", shared.Forbidden(), http.StatusForbidden,
			`{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
		{"an archived project", domain.ErrProjectArchived, http.StatusConflict,
			`{"status":409,"code":"project.archived","title":"Conflict","detail":"The project is archived; unarchive it to change it."}`},
		{"an archived work item", domain.ErrIssueArchived, http.StatusConflict,
			`{"status":409,"code":"issue.archived","title":"Conflict","detail":"The work item is archived; unarchive it to change it."}`},
		{"a busy project", shared.ServerBusy(time.Second), http.StatusServiceUnavailable, busy},
		{"a failure", errGone, http.StatusInternalServerError, `{"status":500,"code":"internal_error","title":"Internal Server Error"}`},
	} {
		h := newServer(t, httpadapter.UseCases{DeleteIssueRelation: &fakeDelete{err: tt.err}})
		res, body := do(t, h, request(http.MethodDelete, relationPath, "alice", ""))
		if res.StatusCode != tt.status || body != tt.want+"\n" || tt.want == busy && res.Header.Get("Retry-After") != "1" {
			t.Errorf("%s: DELETE = %d %s, Retry-After %q; want %d %s", tt.name, res.StatusCode, body, res.Header.Get("Retry-After"), tt.status,
				tt.want)
		}
	}
}

````

`server/internal/modules/issue/module.go`（修改，2 处）：

````old server/internal/modules/issue/module.go
// work item's relations and relating it to others. For the
// project module's deletions (its IssueCascade port) it soft-deletes a
// deleted project's rows of them, tells whether work items are in a state,
// and takes deleted labels off them.
````
````new server/internal/modules/issue/module.go
// work item's relations, relating it to others and deleting a relation.
// For the project module's deletions (its IssueCascade port) it
// soft-deletes a deleted project's rows of them, tells whether work items
// are in a state, and takes deleted labels off them.
````
````old server/internal/modules/issue/module.go
			Locks: locks, Relations: store, Clock: d.Clock,
		}),
````
````new server/internal/modules/issue/module.go
			Locks: locks, Relations: store, Clock: d.Clock,
		}),
		DeleteIssueRelation: app.NewDeleteIssueRelation(app.DeleteIssueRelationDeps{
			Locks: locks, Relations: store, Clock: d.Clock,
		}),
````

`web/apps/web/core/lib/error-messages.ts`（修改，1 处）：

````old web/apps/web/core/lib/error-messages.ts
  "issue.not_archivable": "errors.issue_not_archivable",
````
````new web/apps/web/core/lib/error-messages.ts
  "issue.not_archivable": "errors.issue_not_archivable",
  "issue.relation_not_found": "errors.issue_relation_not_found",
````

`web/packages/i18n/src/locales/en/errors.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/errors.json
    "issue_not_archivable": "Only a completed or cancelled work item can be archived. Move it to such a state first.",
````
````new web/packages/i18n/src/locales/en/errors.json
    "issue_not_archivable": "Only a completed or cancelled work item can be archived. Move it to such a state first.",
    "issue_relation_not_found": "The relation does not exist, or you cannot see either of its work items.",
````

`web/packages/i18n/src/locales/zh-CN/errors.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/errors.json
    "issue_not_archivable": "只有已完成或已取消的工作项才能归档。请先把它移到已完成或已取消的状态。",
````
````new web/packages/i18n/src/locales/zh-CN/errors.json
    "issue_not_archivable": "只有已完成或已取消的工作项才能归档。请先把它移到已完成或已取消的状态。",
    "issue_relation_not_found": "关联不存在，或者它的两个工作项你都看不到。",
````

- [ ] **Step 3: 组合、取锁形状、写闸门和矩阵**

`server/internal/bootstrap/gate_test.go`（修改，1 处）：

````old server/internal/bootstrap/gate_test.go
			relating("relates_to", w.newWorkItem(t)), http.StatusCreated},
````
````new server/internal/bootstrap/gate_test.go
			relating("relates_to", w.newWorkItem(t)), http.StatusCreated},
		{"a deletion of a relation", http.MethodDelete, relationPath(listIn[answeredRelation](t, w.send(t, "alice", http.MethodPost,
			relationsOf(w.newWorkItem(t)), relating("relates_to", w.newWorkItem(t)), http.StatusCreated))[0].ID), "", http.StatusNoContent},
````

`server/internal/bootstrap/issue_relation_deletes_test.go`（新文件，186 行）：

````file server/internal/bootstrap/issue_relation_deletes_test.go
package bootstrap

import (
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// relationPath is the path of the relation id.
func relationPath(id uuid.UUID) string {
	return "/api/v0/issue-relations/" + id.String()
}

// relationDeletion is how the relation id stands (rowDeletion).
func (w issueWorld) relationDeletion(t *testing.T, id uuid.UUID, at time.Time) string {
	t.Helper()
	return w.rowDeletion(t, "issue_relations", id, at)
}

// rowDeletion is how the row id of table stands: "undeleted", or "deleted
// by <account> at the moment" when it was deleted and written last at at,
// else "... at another moment", an account by its name in the world.
func (w issueWorld) rowDeletion(t *testing.T, table string, id uuid.UUID, at time.Time) string {
	t.Helper()
	var deleted *time.Time
	var by uuid.UUID
	var written time.Time
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at, updated_by_id, updated_at FROM "+table+" WHERE id = $1", id).Scan(
		&deleted, &by, &written); err != nil {
		t.Fatal(err)
	}
	if deleted == nil {
		return "undeleted"
	}
	when := "at the moment"
	if !deleted.Equal(at) || !written.Equal(at) {
		when = "at another moment"
	}
	return "deleted by " + w.names()[by] + " " + when
}

// A relation is deleted from either of its work items, as bootstrap wires
// it (M4 design 3.5, 3.7, 3.10, 3.13). alice makes Ops, of acme, of which
// bob is no member, Near in Web and Far in Ops, and relates Near to Far,
// blocking it: stored as Far's blocked_by. bob's deletion is decided on
// Far, which he sees and may not change, then on Near, which he may: 204.
// The relation is deleted by bob at the moment Near and Far are changed
// last, by him; each has a row of its activity on it, in its project, the
// type as it read it, the other by its identifier, after alice's of its
// making; neither lists it. Deleted, it is not found again.
func TestARelationIsDeletedFromEitherWorkItem(t *testing.T) {
	w := newIssueWorld(t)
	ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
	near := w.newWorkItemIn(t, w.web, `{"name":"Near"}`)
	far := w.newWorkItemIn(t, ops, `{"name":"Far"}`)
	made := listIn[answeredRelation](t, w.send(t, "alice", http.MethodPost, relationsOf(near), relating("blocking", far), http.StatusCreated))

	w.send(t, "bob", http.MethodDelete, relationPath(made[0].ID), "", http.StatusNoContent)
	_, at, _ := archivalOf(t, w, near)
	if got := w.relationDeletion(t, made[0].ID, at); got != "deleted by bob at the moment" {
		t.Errorf("the relation is %s, want deleted by bob at the moment", got)
	}
	for _, item := range []uuid.UUID{near, far} {
		if _, changed, by := archivalOf(t, w, item); !changed.Equal(at) || by != w.ids["bob"] {
			t.Errorf("the work item %s is changed last by %s at %v, want by bob at %v", item, by, changed, at)
		}
	}
	for _, tt := range []struct {
		item uuid.UUID
		want []string
	}{
		{near, []string{"alice: updated blocking  → OPS-1, none → Far, in WEB, added blocking relation",
			"bob: deleted blocking OPS-1 → , Far → none, in WEB, deleted blocking relation at the moment"}},
		{far, []string{"alice: updated blocked_by  → WEB-1, none → Near, in OPS, added blocked_by relation",
			"bob: deleted blocked_by WEB-1 → , Near → none, in OPS, deleted blocked_by relation at the moment"}},
	} {
		if got := w.relationRows(t, tt.item, at); !slices.Equal(got, tt.want) {
			t.Errorf("the rows of %s's relations:\n%q\nwant\n%q", tt.item, got, tt.want)
		}
		if listed := w.listedRelations(t, "alice", tt.item); len(listed) != 0 {
			t.Errorf("%s lists %q, want none", tt.item, listed)
		}
	}
	answer := w.send(t, "bob", http.MethodDelete, relationPath(made[0].ID), "", http.StatusNotFound)
	if got := refusalOf(t, []byte(answer), false); got != "issue.relation_not_found" {
		t.Errorf("deleting it again answers %s, want issue.relation_not_found", got)
	}
}

// Who may not delete a relation, and the relations that are not deleted
// (M4 design 3.4, 3.8, 3.10), each refused with nothing written: walt,
// acme's member and none of Web, sees Near and Up and may change neither
// (403); olga, no member of acme, sees neither (404
// issue.relation_not_found), nor walt the relation of two work items of
// Lab, a private project he is none of, each the very answer of a relation
// not there. Then a relation of an archived work item, Shelved, from
// either end, and one of a work item of an archived project, Old's, are
// not deleted (409), the archived one's end stored as issue_id or not.
func TestWhoMayNotDeleteARelation(t *testing.T) {
	w := newIssueWorld(t)
	var lab struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/workspaces/acme/projects", `{"name":"Lab","identifier":"LAB","network":0}`,
		http.StatusCreated), &lab)
	old := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Old", "OLD")
	near, up := w.newWorkItemIn(t, w.web, `{"name":"Near"}`), w.newWorkItemIn(t, w.web, `{"name":"Up"}`)
	shelved := w.newWorkItemIn(t, w.web, `{"name":"Shelved","state_id":"`+w.done.String()+`"}`)
	labs, lab2 := w.newWorkItemIn(t, lab.ID, `{"name":"Lab's"}`), w.newWorkItemIn(t, lab.ID, `{"name":"Lab's too"}`)
	olds := w.newWorkItemIn(t, old, `{"name":"Old's"}`)
	related := func(from, to uuid.UUID, kind string) uuid.UUID {
		return listIn[answeredRelation](t, w.send(t, "alice", http.MethodPost, relationsOf(from), relating(kind, to), http.StatusCreated))[0].ID
	}
	ofUp, ofLab := related(near, up, "relates_to"), related(labs, lab2, "relates_to")
	toShelved, fromShelved := related(near, shelved, "relates_to"), related(shelved, up, "relates_to")
	toOld, fromOld := related(up, olds, "relates_to"), related(olds, near, "relates_to")
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+shelved.String()+"/archive", "", http.StatusOK)
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+old.String()+"/archive", "", http.StatusOK)
	before := w.relationRows(t, near, time.Time{})

	for _, tt := range []struct {
		name     string
		relation uuid.UUID
		status   int
		code     string
	}{
		{"walt", ofUp, http.StatusForbidden, "forbidden"},
		{"olga", ofUp, http.StatusNotFound, "issue.relation_not_found"},
		{"walt", ofLab, http.StatusNotFound, "issue.relation_not_found"},
		{"bob", toShelved, http.StatusConflict, "issue.archived"},
		{"bob", fromShelved, http.StatusConflict, "issue.archived"},
		{"alice", toOld, http.StatusConflict, "project.archived"},
		{"alice", fromOld, http.StatusConflict, "project.archived"},
	} {
		answer := w.send(t, tt.name, http.MethodDelete, relationPath(tt.relation), "", tt.status)
		if got := refusalOf(t, []byte(answer), false); got != tt.code || w.relationDeletion(t, tt.relation, time.Time{}) != "undeleted" {
			t.Errorf("%s's deletion of %s answers %s, the relation %s; want %s, undeleted", tt.name, tt.relation, got,
				w.relationDeletion(t, tt.relation, time.Time{}), tt.code)
		}
		if tt.status != http.StatusNotFound {
			continue
		}
		if absent := w.send(t, tt.name, http.MethodDelete, relationPath(uuid.NewV7()), "", http.StatusNotFound); answer != absent {
			t.Errorf("%s's deletion of %s answers %s, want as for a relation not there: %s", tt.name, tt.relation, answer, absent)
		}
	}
	if after := w.relationRows(t, near, time.Time{}); !slices.Equal(after, before) {
		t.Errorf("Near's rows on its relations:\n%q\nwant them as they were:\n%q", after, before)
	}
}

// The guest rule holds for deleting a relation (M4 design 3.4, 3.10):
// walt, acme's admin and Web's guest (adminGuest), sees only his own work
// items in Web. alice relates Hers to his His, stored from Hers, which he
// does not see: his deletion is decided on Hers, hidden from him, then on
// His, which he may change as acme's admin: 204. The relation of Up and
// Hers, alice's both, he finds not: the very 404 of a relation not there.
func TestTheGuestRuleHoldsForDeletingARelation(t *testing.T) {
	w := newIssueWorld(t)
	w.adminGuest(t)
	up := w.newWorkItemIn(t, w.web, `{"name":"Up"}`)
	his := w.newWorkItemBy(t, "walt", w.web, `{"name":"His"}`)
	hers := w.newWorkItemIn(t, w.web, `{"name":"Hers"}`)
	ofHis := listIn[answeredRelation](t, w.send(t, "alice", http.MethodPost, relationsOf(hers), relating("relates_to", his, up), http.StatusCreated))
	if len(ofHis) != 2 {
		t.Fatalf("alice made %d relations, want 2", len(ofHis))
	}
	byID := map[uuid.UUID]uuid.UUID{ofHis[0].Issue.ID: ofHis[0].ID, ofHis[1].Issue.ID: ofHis[1].ID}

	w.send(t, "walt", http.MethodDelete, relationPath(byID[his]), "", http.StatusNoContent)
	_, at, _ := archivalOf(t, w, his)
	if got := w.relationDeletion(t, byID[his], at); got != "deleted by walt at the moment" {
		t.Errorf("the relation of His is %s, want deleted by walt at the moment", got)
	}
	answer := w.send(t, "walt", http.MethodDelete, relationPath(byID[up]), "", http.StatusNotFound)
	absent := w.send(t, "walt", http.MethodDelete, relationPath(uuid.NewV7()), "", http.StatusNotFound)
	if got := refusalOf(t, []byte(answer), false); got != "issue.relation_not_found" || answer != absent ||
		w.relationDeletion(t, byID[up], at) != "undeleted" {
		t.Errorf("walt's deletion of the relation of Up answers %s; want issue.relation_not_found, as for a relation not there (%s), and it "+
			"undeleted", answer, absent)
	}
}
````

`server/internal/bootstrap/issue_write_locks_test.go`（修改，1 处）：

````old server/internal/bootstrap/issue_write_locks_test.go
		waits: "issue_relations", mode: "ACCESS EXCLUSIVE"},
````
````new server/internal/bootstrap/issue_write_locks_test.go
		waits: "issue_relations", mode: "ACCESS EXCLUSIVE"},
	// A relation of two new work items, which the send deletes: it reads
	// nothing under the lock that its precheck did not, so it waits for the
	// relations' table, which it writes first.
	{op: "deleteIssueRelation", method: http.MethodDelete, path: "/api/v0/issue-relations/{relation_id}",
		target: func(t *testing.T, w issueWorld) string {
			related := listIn[answeredRelation](t, w.send(t, "alice", http.MethodPost, relationsOf(w.newWorkItem(t)),
				relating("relates_to", w.newWorkItem(t)), http.StatusCreated))
			return relationPath(related[0].ID)
		},
		body: func(*testing.T, issueWorld) string { return "" }, want: http.StatusNoContent, waits: "issue_relations", mode: "SHARE"},
````

`server/internal/bootstrap/issue_writes_test.go`（修改，2 处）：

````old server/internal/bootstrap/issue_writes_test.go
// projects locked at once. She archives the first, in Done, and
// unarchives it; she deletes the second, and Part and Low under it. With
// Web archived, her creation and her change are refused under the lock
// (409).
````
````new server/internal/bootstrap/issue_writes_test.go
// projects locked at once, and deletes the relation. She archives the
// first, in Done, and unarchives it; she deletes the second, and Part and
// Low under it. With Web archived, her creation and her change are
// refused under the lock (409).
````
````old server/internal/bootstrap/issue_writes_test.go
	route.answerWithin(t, w.contract, http.MethodPost, relationsOf(first.ID), alice, relating("blocking", far), http.StatusCreated)
````
````new server/internal/bootstrap/issue_writes_test.go
	related := listIn[answeredRelation](t, route.answerWithin(t, w.contract, http.MethodPost, relationsOf(first.ID), alice, relating("blocking", far),
		http.StatusCreated))
	route.answerWithin(t, w.contract, http.MethodDelete, relationPath(related[0].ID), alice, "", http.StatusNoContent)
````

`server/internal/bootstrap/permission_matrix_relations_test.go`（修改，4 处）：

````old server/internal/bootstrap/permission_matrix_relations_test.go
			cells: issueRows.of(cellValidationFailed, cellValidationFailed, cellIssueNotFound, cellValidationFailed, cellForbidden, cellForbidden)},
	}
````
````new server/internal/bootstrap/permission_matrix_relations_test.go
			cells: issueRows.of(cellValidationFailed, cellValidationFailed, cellIssueNotFound, cellValidationFailed, cellForbidden, cellForbidden)},
		// Whoever may change either of its work items, as updateIssue (M4
		// design 3.4, 3.10, 9.2); one who sees neither finds none.
		{op: "deleteIssueRelation", write: true, columns: issueColumns, request: toRelation,
			cells: onARelation(cellNoContent, cellNoContent, cellNoContent, cellForbidden)},
		// An archived project's work item's is not deleted (M4 design 3.8),
		// after the decision.
		{op: "deleteIssueRelation", variant: "archived", write: true, columns: archivedColumns, request: toRelation,
			cells: relationRows.ofArchived(cellProjectArchived, cellForbidden)},
		// Nor is its archived work item's: the project's 409 comes first.
		{op: "deleteIssueRelation", variant: "archived, its work item archived", write: true, columns: archivedColumns,
			prepare: inDone(true), request: toRelation, cells: relationRows.ofArchived(cellProjectArchived, cellForbidden)},
		// Nor is an archived work item's, after the decision.
		{op: "deleteIssueRelation", variant: "archived work item", write: true, columns: issueColumns, prepare: inDone(true),
			request: toRelation, cells: onARelation(cellIssueArchived, cellIssueArchived, cellIssueArchived, cellForbidden)},
	}
}

// cellRelationNotFound is the answer to a caller who sees neither work item
// of a relation (M4 design 3.10).
var cellRelationNotFound = cell{http.StatusNotFound, "issue.relation_not_found"}

// relationRows are the seeded relations, by the key of their row's project
// and their name (seededRelations), as the matrix's rows name them.
var relationRows = rowsByID{path: "issue-relations", notFound: cellRelationNotFound, find: seeded.relation}

// relationOf is the name of the relation of its project (projectOf) that a
// column's cells delete, one of whose work items is the column's own
// (issueOf): the one whose other work item the column sees as it sees its
// own, so that its cells answer as on its own, but the guest's and the
// workspace admin who is a guest's, on another's, who see neither. PA, PM,
// PM+WA, WA-, WM-公, WG- and X, Blocked, whose other work item is the
// private project's; the member who made one, his own (Member's); the
// guest who sees every work item, WM-私 and P-前, the private project's
// Secret; the guest on his own, and the workspace admin who is a guest on
// another's, the guest's own (Guest's); the guest on another's, Others';
// the workspace admin who is a guest on his own, his (Admin guest's); the
// archived project's columns, Shelved; and the deleted workspace's, Gone.
func relationOf(c caller) string {
	switch c {
	case callerCreator:
		return "Member's"
	case callerGuestAll, callerMemberPrivate, callerBefore:
		return "Secret"
	case callerGuestOwn, callerAdminGuest:
		return "Guest's"
	case callerGuestOthers:
		return "Others'"
	case callerAdminGuestOwn:
		return "Admin guest's"
	case callerArchivedAdmin, callerArchivedNever, callerArchivedMember:
		return "Shelved"
	case callerDeleted:
		return "Gone"
	}
	return "Blocked"
}

// toRelation is the request of a row whose callers each delete their
// column's relation (relationOf).
func toRelation(c caller, s seeded) (string, string, string) {
	return relationRows.to(http.MethodDelete, "", relationOf(c), "")(c, s)
}

// onARelation are the cells of a row of the issue table on a relation: a
// row's on a work item (onAWorkItem), issue.relation_not_found where it is
// issue.not_found, for a caller who sees neither of its work items.
func onARelation(admin, member, creator, guest cell) map[caller]cell {
	cells := onAWorkItem(admin, member, creator, guest)
	for c, answer := range cells {
		if answer == cellIssueNotFound {
			cells[c] = cellRelationNotFound
		}
	}
	return cells
````
````old server/internal/bootstrap/permission_matrix_relations_test.go
// projects, list both of the public project's matrixIssue; the member who
````
````new server/internal/bootstrap/permission_matrix_relations_test.go
// projects, list the public project's matrixIssue's three; the member who
````
````old server/internal/bootstrap/permission_matrix_relations_test.go
// him. Every other column lists none: the guest's own's other end the
// guest rule hides from him, the workspace admin who is a guest's own's is
// in a project he is none of, and no seeded relation is of the archived
// project's work item.
````
````new server/internal/bootstrap/permission_matrix_relations_test.go
// him, and he is no member of the archived project; the archived project's
// admin, its matrixIssue's, the private project's. Every other column
// lists none: the guest's own's other end the guest rule hides from him,
// and the workspace admin who is a guest's own's other ends from him, one
// in a project he is none of.
````
````old server/internal/bootstrap/permission_matrix_relations_test.go
	callerProjectAdmin:   {"blocked_by acme/private|" + matrixIssue, "relates_to acme/public|" + issueOf(callerGuestOwn)},
	callerProjectMember:  {"blocked_by acme/private|" + matrixIssue, "relates_to acme/public|" + issueOf(callerGuestOwn)},
	callerMemberAndAdmin: {"blocked_by acme/private|" + matrixIssue, "relates_to acme/public|" + issueOf(callerGuestOwn)},
	callerCreator:        {"duplicate acme/private|" + issueOf(callerBefore)},
	callerGuestAll:       {"blocking acme/private|" + issueOf(callerBefore)},
````
````new server/internal/bootstrap/permission_matrix_relations_test.go
	callerProjectAdmin: {"blocked_by acme/private|" + matrixIssue, "relates_to acme/public|" + issueOf(callerGuestOwn),
		"relates_to acme/public|" + issueOf(callerAdminGuestOwn)},
	callerProjectMember: {"blocked_by acme/private|" + matrixIssue, "relates_to acme/public|" + issueOf(callerGuestOwn),
		"relates_to acme/public|" + issueOf(callerAdminGuestOwn)},
	callerMemberAndAdmin: {"blocked_by acme/private|" + matrixIssue, "relates_to acme/public|" + issueOf(callerGuestOwn),
		"relates_to acme/public|" + issueOf(callerAdminGuestOwn)},
	callerCreator:       {"duplicate acme/private|" + issueOf(callerBefore)},
	callerGuestAll:      {"blocking acme/private|" + issueOf(callerBefore)},
	callerArchivedAdmin: {"duplicate acme/private|" + matrixIssue},
````

`server/internal/bootstrap/permission_matrix_seed_issues_test.go`（修改，2 处）：

````old server/internal/bootstrap/permission_matrix_seed_issues_test.go
// project's matrixIssue, by its workspace's admin, then ownIssues.
````
````new server/internal/bootstrap/permission_matrix_seed_issues_test.go
// project's matrixIssue, by its workspace's admin, then ownIssues, then
// relatedItems.
````
````old server/internal/bootstrap/permission_matrix_seed_issues_test.go
	return items
````
````new server/internal/bootstrap/permission_matrix_seed_issues_test.go
	return append(items, relatedItems...)
````

`server/internal/bootstrap/permission_matrix_seed_relations_test.go`（整个文件，94 行）：

````whole server/internal/bootstrap/permission_matrix_seed_relations_test.go
package bootstrap

import (
	"context"
	"reflect"
	"strings"

	issuedomain "github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// seededRelation is a relation issueSeed stores: its name; its work items,
// each by its project's key and its name (seededItems), the first the end
// stored as issue_id, of its row's project; and its type as stored.
type seededRelation struct {
	name          string
	fromKey, from string
	toKey, to     string
	kind          issuedomain.RelationType
}

// goneItem is the second work item of gone's project (relatedItems).
const goneItem = "Second"

// relatedItems are the work items issueSeed stores for a relation alone,
// after the others of seededItems: gone's second, by its admin, so that a
// relation of the deleted workspace's project is there for its column's
// cells to name (relationOf).
var relatedItems = []seededItem{{"gone/project", goneItem, matrixAdmins["gone"]}}

// seededRelations are the relations prepareMatrix seeds through the issue
// store (M4 design 9.2), each a case of a list of relations, or the one a
// column's cells delete (relationOf): acme's public project's matrixIssue
// blocked by the private one's (Blocked), which the guest rule of the
// public project hides from PG, who sees every work item of the private
// one; the guest's own work item's relates_to the public project's
// matrixIssue, which the guest rule hides from him; PM's duplicate of the
// member before's in the private project; the workspace admin who is a
// guest's own relates_to the private project's matrixIssue, of a project
// he is none of; the member before's blocked by it (Secret), which PG sees
// there; the public project's matrixIssue relates_to the workspace admin
// who is a guest's own (Others'), neither the guest's; the archived
// project's matrixIssue a duplicate of the private one's (Shelved); and
// gone's matrixIssue relates_to its second work item, deleted with gone.
var seededRelations = []seededRelation{
	{"Blocked", "acme/public", matrixIssue, "acme/private", matrixIssue, issuedomain.BlockedBy},
	{"Guest's", "acme/public", issueOf(callerGuestOwn), "acme/public", matrixIssue, issuedomain.RelatesTo},
	{"Member's", "acme/public", issueOf(callerCreator), "acme/private", issueOf(callerBefore), issuedomain.Duplicate},
	{"Admin guest's", "acme/public", issueOf(callerAdminGuestOwn), "acme/private", matrixIssue, issuedomain.RelatesTo},
	{"Secret", "acme/private", issueOf(callerBefore), "acme/private", matrixIssue, issuedomain.BlockedBy},
	{"Others'", "acme/public", matrixIssue, "acme/public", issueOf(callerAdminGuestOwn), issuedomain.RelatesTo},
	{"Shelved", "acme/archived", matrixIssue, "acme/private", matrixIssue, issuedomain.Duplicate},
	{"Gone", "gone/project", matrixIssue, "gone/project", goneItem, issuedomain.RelatesTo},
}

// relationsOf are seededRelations as stored, each with the id sd names, of
// its row's workspace, by that workspace's admin, at the seed's moment.
func (s issueSeed) relationsOf(sd seeded) []issuedomain.Relation {
	rs := make([]issuedomain.Relation, len(seededRelations))
	for n, r := range seededRelations {
		slug, _, _ := strings.Cut(r.fromKey, "/")
		by := s.ids[matrixAdmins[slug]]
		rs[n] = issuedomain.Relation{ID: sd.relation(r.fromKey, r.name), WorkspaceID: s.workspaces[slug], ProjectID: s.projects[r.fromKey],
			IssueID: sd.issue(r.fromKey, r.from), RelatedID: sd.issue(r.toKey, r.to), Type: r.kind, CreatedByID: &by, CreatedAt: s.now}
	}
	return rs
}

// relations stores seededRelations (relationsOf), each in an insert of its
// own, after the work items (issues).
func (s issueSeed) relations(sd seeded) {
	s.t.Helper()
	for _, r := range s.relationsOf(sd) {
		if err := s.store.InsertRelations(context.Background(), []issuedomain.Relation{r}); err != nil {
			s.t.Fatal(err)
		}
	}
}

// seededRelationRows checks the relations the cells rest on, read back
// through the issue store by the ids sd names: each as relationsOf has it,
// but its moment, which the database keeps to the microsecond; gone's
// deleted with it. One missing, or stored otherwise, would let a cell
// answer as it wants for another reason.
func (s issueSeed) seededRelationRows(sd seeded) {
	s.t.Helper()
	for _, want := range s.relationsOf(sd) {
		gone := want.WorkspaceID == s.workspaces["gone"]
		got, found, err := s.store.GetRelation(context.Background(), want.ID)
		got.CreatedAt = want.CreatedAt
		if err != nil || found == gone || !gone && !reflect.DeepEqual(got, want) {
			s.t.Fatalf("the relation %s = %+v, %v, %v; want %+v, deleted %v", want.ID, got, found, err, want, gone)
		}
	}
}
````

`server/internal/bootstrap/permission_matrix_targets_test.go`（修改，3 处）：

````old server/internal/bootstrap/permission_matrix_targets_test.go
// column's (issueOf), whose maker the column's case may be about.
````
````new server/internal/bootstrap/permission_matrix_targets_test.go
// column's (issueOf), whose maker the column's case may be about, and a
// relation its column's (relationOf), whose other work item it may be.
````
````old server/internal/bootstrap/permission_matrix_targets_test.go
	}},
````
````new server/internal/bootstrap/permission_matrix_targets_test.go
	}},
	"{relation_id}": {"relation", func(s seeded) map[string]uuid.UUID { return s.relations }, func(c caller, s seeded) (string, uuid.UUID) {
		return relationOf(c), s.relation(projectOf(c), relationOf(c))
	}},
````
````old server/internal/bootstrap/permission_matrix_targets_test.go
// work item) one seeded in projectOf(c), a work item the column's own
// (issueOf), as must a work item named by its identifier ({identifier}
// right after issues: identifierOf), each from a column of a project table
// (projectTables): a project's operation in a workspace-level row, or the
// only admin's, would leave the project level's own columns unasked.
````
````new server/internal/bootstrap/permission_matrix_targets_test.go
// work item, a relation) one seeded in projectOf(c), a work item and a
// relation the column's own (issueOf, relationOf), as must a work item
// named by its identifier ({identifier} right after issues:
// identifierOf), each from a column of a project table (projectTables): a
// project's operation in a workspace-level row, or the only admin's, would
// leave the project level's own columns unasked.
````

`server/internal/bootstrap/permission_matrix_test.go`（修改，1 处）：

````old server/internal/bootstrap/permission_matrix_test.go
// project, with its states, labels and work item, with it; then the checks
// that the rows the cells rest on are there (preconditions), the states,
// the labels, the work items and the relations among them (seededStates,
// seededLabels, seededIssues, seededRelationRows).
````
````new server/internal/bootstrap/permission_matrix_test.go
// project, with its states, labels, work items and relation, with it;
// then the checks that the rows the cells rest on are there
// (preconditions), the states, the labels, the work items and the
// relations among them (seededStates, seededLabels, seededIssues,
// seededRelationRows).
````

- [ ] **Step 4: 交错的帮手和交错 10、11、23、26、加关联与删除工作区**

`server/internal/bootstrap/interleaving_test.go`（修改，3 处）：

````old server/internal/bootstrap/interleaving_test.go
// serializeOn runs first, a write that waits at g holding its locks, then
// second, once the first is held there; once the second waits for a row
// lock of projects on pool's database (pgtest.WaitForLockWaitOn), it opens
// g. It answers the moment g opened, and each write's error. Every wait
// has a deadline: ctx's, g's or the probe's.
````
````new server/internal/bootstrap/interleaving_test.go
// serializeOn is serializeOnTable on projects.
````
````old server/internal/bootstrap/interleaving_test.go
	firstErr, secondErr error) {
````
````new server/internal/bootstrap/interleaving_test.go
	firstErr, secondErr error) {
	t.Helper()
	return serializeOnTable(t, ctx, pool, g, "projects", first, second)
}

// serializeOnTable runs first, a write that waits at g holding its locks,
// then second, once the first is held there; once the second waits for a
// row lock of table on pool's database (pgtest.WaitForLockWaitOn), it
// opens g. It answers the moment g opened, and each write's error. Every
// wait has a deadline: ctx's, g's or the probe's.
func serializeOnTable(t *testing.T, ctx context.Context, pool *pgxpool.Pool, g *gate, table string, first, second func() error) (
	opened time.Time, firstErr, secondErr error) {
````
````old server/internal/bootstrap/interleaving_test.go
	pgtest.WaitForLockWaitOn(t, pool, "projects", 5*time.Second)
````
````new server/internal/bootstrap/interleaving_test.go
	pgtest.WaitForLockWaitOn(t, pool, table, 5*time.Second)
````

`server/internal/bootstrap/issue_interleaving_relations_test.go`（新文件，306 行）：

````file server/internal/bootstrap/issue_interleaving_relations_test.go
package bootstrap

import (
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The interleavings of relations across projects (M4 design 3.5, 3.6,
// 3.10, convention 7; 9.3, interleavings 10, 11, 23 and 26), on
// issueWorld's real database, as issue_interleaving_test.go runs them.

// nearAndFar makes Ops, of acme, by alice, with bob its member, then bob's
// Near in Web and alice's Far in Ops: the two ends of the interleavings'
// relations.
func nearAndFar(t *testing.T, w issueWorld) {
	ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+ops.String()+"/members",
		`{"members":[{"member_id":"`+w.ids["bob"].String()+`","role":15}]}`, http.StatusCreated)
	w.newWorkItemBy(t, "bob", w.web, `{"name":"Near"}`)
	w.newWorkItemIn(t, ops, `{"name":"Far"}`)
}

// projectWith is the id of the world's one project of the identifier.
func (w issueWorld) projectWith(t *testing.T, identifier string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT id FROM projects WHERE identifier = $1", identifier).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// deletedAt is the moment the row id of table was deleted.
func (w issueWorld) deletedAt(t *testing.T, table string, id uuid.UUID) time.Time {
	t.Helper()
	var at time.Time
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at FROM "+table+" WHERE id = $1", id).Scan(&at); err != nil {
		t.Fatal(err)
	}
	return at
}

// relationRowOf is the id of the one row of the work item item's activity
// on its relations.
func (w issueWorld) relationRowOf(t *testing.T, item uuid.UUID) uuid.UUID {
	t.Helper()
	ids := queryIDs(t, w.pool, `SELECT id FROM issue_activities WHERE issue_id = $1
		AND field IN ('relates_to', 'duplicate', 'blocked_by', 'blocking')`, item)
	if len(ids) != 1 {
		t.Fatalf("the work item %s has %d rows of activity on its relations, want 1", item, len(ids))
	}
	return ids[0]
}

// relates is name's relation of the world's work item named from to the
// one named to, as kind (workItemNamed).
func relates(name, from, to, kind string) racingWrite {
	return func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
		return w.sending(e, newIssueRoute(t, w.pool, tx), name, http.MethodPost, relationsOf(w.workItemNamed(t, from)),
			relating(kind, w.workItemNamed(t, to)))
	}
}

// unrelates is name's deletion of the world's one relation.
func unrelates(name string) racingWrite {
	return func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
		return w.sending(e, newIssueRoute(t, w.pool, tx), name, http.MethodDelete,
			relationPath(queryIDs(t, w.pool, "SELECT id FROM issue_relations")[0]), "")
	}
}

// deletesOps is alice's deletion of Ops through the project module's
// route.
var deletesOps = aliceOnWeb(http.MethodDelete, func(t *testing.T, w issueWorld) string {
	return "/api/v0/projects/" + w.projectWith(t, "OPS").String()
}, "")

// Two relations of the same two work items, each the other way round,
// serialize on their projects' rows (M4 design 3.5, 3.10, convention 7;
// interleaving 10), in both orders: alice relates Near, of Web, to Far, of
// Ops, as blocking it, while bob relates Far to Near, as blocking it. Each
// locks Web and Ops at once, in ascending id order, so neither holds a row
// the other waits for while waiting for one of the other's: the second
// waits for the first's projects, then reads under its locks the relation
// the first made, either way, and is refused at its place, 422 duplicate,
// nothing written. The first's relation is the one stored, with its row
// of each work item's activity.
func TestOppositeRelationsOfTwoWorkItemsSerialize(t *testing.T) {
	inBothOrders(t, nearAndFar, "alice's", relates("alice", "Near", "Far", "blocking"), "bob's", relates("bob", "Far", "Near", "blocking"),
		func(t *testing.T, w issueWorld, aliceFirst bool, hers, his exchange) {
			first, second, stored := hers, his, "Far blocked_by Near, in OPS, by alice, alice at the moment"
			if !aliceFirst {
				first, second, stored = his, hers, "Near blocked_by Far, in WEB, by bob, bob at the moment"
			}
			made := listIn[answeredRelation](t, w.answered(t, first, http.StatusCreated))
			if answer := w.answered(t, second, http.StatusUnprocessableEntity); refusalOf(t, []byte(answer), false) !=
				"validation_failed issue_ids[0] duplicate" {
				t.Errorf("the second relation answers %s, want issue_ids[0] duplicate", answer)
			}
			if got := w.storedRelations(t, made[0].CreatedAt); !slices.Equal(got, []string{stored}) {
				t.Errorf("the relations stored:\n%q\nwant\n%q", got, []string{stored})
			}
			for _, item := range []string{"Near", "Far"} {
				w.relationRowOf(t, w.workItemNamed(t, item))
			}
		})
}

// Deleting a work item related across projects and deleting the project of
// the other ends serialize on their relations' rows (M4 design 3.5, 3.6;
// interleaving 11), in both orders: bob deletes his Near, of Web, while
// alice deletes Ops, whose Far relates to Near, stored from Near, and
// whose Far too is its duplicate, stored from Far too. Neither waits for
// the other's project: each locks its own, then the relation rows it
// deletes, by id, in ascending order, so neither holds a row the other
// waits for while waiting for one of the other's. The second waits for
// the first's relation rows and, once the first commits, finds them
// deleted and leaves them: each is deleted once, at the first's moment,
// by its deleter.
func TestDeletingARelatedWorkItemAndTheOtherEndsProjectSerialize(t *testing.T) {
	prepare := func(t *testing.T, w issueWorld) {
		nearAndFar(t, w)
		near := w.workItemNamed(t, "Near")
		farToo := w.newWorkItemIn(t, w.projectWith(t, "OPS"), `{"name":"Far too"}`)
		w.send(t, "alice", http.MethodPost, relationsOf(near), relating("relates_to", w.workItemNamed(t, "Far")), http.StatusCreated)
		w.send(t, "alice", http.MethodPost, relationsOf(farToo), relating("duplicate", near), http.StatusCreated)
	}
	inBothOrdersAt(t, "issue_relations", prepare, "Near deleted", deletesNamed("bob", "Near"), "Ops deleted", deletesOps,
		func(t *testing.T, w issueWorld, nearFirst bool, nearDeleted, opsDeleted exchange) {
			w.answered(t, nearDeleted, http.StatusNoContent)
			w.answered(t, opsDeleted, http.StatusNoContent)
			at, by := w.deletedAt(t, "issues", w.workItemNamed(t, "Near")), "bob"
			if !nearFirst {
				at, by = w.deletedAt(t, "projects", w.projectWith(t, "OPS")), "alice"
			}
			relations := queryIDs(t, w.pool, "SELECT id FROM issue_relations")
			if len(relations) != 2 {
				t.Fatalf("%d relations, want 2", len(relations))
			}
			for _, id := range relations {
				if got := w.relationDeletion(t, id, at); got != "deleted by "+by+" at the moment" {
					t.Errorf("the relation %s is %s, want deleted by %s at the moment", id, got, by)
				}
			}
		})
}

// Relating a work item to another and deleting the other's project
// serialize on the other's project's row (M4 design 3.5, 3.6, 3.10,
// convention 7; interleaving 23), in both orders: bob relates Near, of
// Web, to Far, of Ops, as blocking it, while alice deletes Ops. Deleted
// first: the relation, which passed its precheck with Far reached, waits
// for Ops' row, and under its locks finds Ops deleted: Far is not
// reached, 422 at its place, and nothing is written. Related first: the
// deletion waits for Ops' row, then deletes the relation, stored from
// Far, a row of Ops, and Far's row of its activity on it, at its one
// moment, by alice; Near's row stays, and no undeleted relation names a
// deleted work item.
func TestRelatingAWorkItemAndDeletingTheOtherEndsProjectSerialize(t *testing.T) {
	inBothOrders(t, nearAndFar, "Ops deleted", deletesOps, "related", relates("bob", "Near", "Far", "blocking"),
		func(t *testing.T, w issueWorld, deletedFirst bool, deleted, related exchange) {
			w.answered(t, deleted, http.StatusNoContent)
			if deletedFirst {
				if answer := w.answered(t, related, http.StatusUnprocessableEntity); refusalOf(t, []byte(answer), false) !=
					"validation_failed issue_ids[0] not_allowed" {
					t.Errorf("the relation answers %s, want issue_ids[0] not_allowed", answer)
				}
				if n := len(queryIDs(t, w.pool, "SELECT id FROM issue_relations")); n != 0 || len(w.relationRows(t, w.workItemNamed(t, "Near"),
					time.Time{})) != 0 {
					t.Errorf("%d relations and Near's rows on them %q after the refusal; want none", n, w.relationRows(t, w.workItemNamed(t, "Near"),
						time.Time{}))
				}
				return
			}
			made := listIn[answeredRelation](t, w.answered(t, related, http.StatusCreated))
			at := w.deletedAt(t, "projects", w.projectWith(t, "OPS"))
			for _, tt := range []struct {
				what, table string
				id          uuid.UUID
				want        string
			}{
				{"the relation", "issue_relations", made[0].ID, "deleted by alice at the moment"},
				{"Far's row on it", "issue_activities", w.relationRowOf(t, w.workItemNamed(t, "Far")), "deleted by alice at the moment"},
				{"Near's row on it", "issue_activities", w.relationRowOf(t, w.workItemNamed(t, "Near")), "undeleted"},
			} {
				if got := w.rowDeletion(t, tt.table, tt.id, at); got != tt.want {
					t.Errorf("%s is %s, want %s", tt.what, got, tt.want)
				}
			}
			if n := len(queryIDs(t, w.pool, `SELECT r.id FROM issue_relations r JOIN issues i ON i.id IN (r.issue_id, r.related_issue_id)
				WHERE r.deleted_at IS NULL AND i.deleted_at IS NOT NULL`)); n != 0 {
				t.Errorf("%d undeleted relations name a deleted work item, want none", n)
			}
		})
}

// Relating work items across projects and deleting their workspace
// serialize on the projects' rows (M4 design 3.5, 3.6, convention 7), in
// both orders: bob relates Near, of Web, to Far, of Ops, as blocking it,
// while alice deletes acme. The relation locks Web and Ops at once, and
// the deletion, under acme's row, its projects, each in ascending id
// order, so neither holds a row the other waits for while waiting for one
// of the other's. Deleted first: the relation, which passed its precheck,
// waits for the projects' rows and under its locks finds its own project
// deleted: 404 issue.not_found, and nothing is written. Related first: the
// deletion waits for them, then deletes the relation with every other
// row, at acme's moment, by alice.
func TestRelatingAcrossProjectsAndDeletingTheWorkspaceSerialize(t *testing.T) {
	inBothOrders(t, nearAndFar, "acme deleted", deletesAcme, "related", relates("bob", "Near", "Far", "blocking"),
		func(t *testing.T, w issueWorld, deletedFirst bool, deleted, related exchange) {
			w.answered(t, deleted, http.StatusNoContent)
			if deletedFirst {
				w.refused(t, related, http.StatusNotFound, "issue.not_found")
				if n := len(queryIDs(t, w.pool, "SELECT id FROM issue_relations")); n != 0 {
					t.Errorf("%d relations after the refusal, want none", n)
				}
				return
			}
			made := listIn[answeredRelation](t, w.answered(t, related, http.StatusCreated))
			if got := w.relationDeletion(t, made[0].ID, w.deletedAt(t, "workspaces", w.acme)); got != "deleted by alice at the moment" {
				t.Errorf("the relation is %s, want deleted by alice at the moment", got)
			}
		})
}

// Two deletions of one relation serialize on its projects' rows (M4
// design 3.5, 3.10), in both orders: alice and bob each delete the
// relation of Near, of Web, to Far, of Ops, which alice made. The second
// waits for the first's projects, then reads the relation again under its
// locks, finds it deleted, and is refused, 404 issue.relation_not_found,
// writing nothing: the relation is deleted once, by the first's deleter,
// at the moment of the row of each work item's activity on its deletion.
func TestTwoDeletionsOfARelationSerialize(t *testing.T) {
	prepare := func(t *testing.T, w issueWorld) {
		nearAndFar(t, w)
		w.send(t, "alice", http.MethodPost, relationsOf(w.workItemNamed(t, "Near")), relating("relates_to", w.workItemNamed(t, "Far")),
			http.StatusCreated)
	}
	inBothOrders(t, prepare, "alice's", unrelates("alice"), "bob's", unrelates("bob"),
		func(t *testing.T, w issueWorld, aliceFirst bool, hers, his exchange) {
			first, second, by := hers, his, "alice"
			if !aliceFirst {
				first, second, by = his, hers, "bob"
			}
			w.answered(t, first, http.StatusNoContent)
			w.refused(t, second, http.StatusNotFound, "issue.relation_not_found")
			relation := queryIDs(t, w.pool, "SELECT id FROM issue_relations")[0]
			at := w.deletedAt(t, "issue_relations", relation)
			if got := w.relationDeletion(t, relation, at); got != "deleted by "+by+" at the moment" {
				t.Errorf("the relation is %s, want deleted by %s at the moment", got, by)
			}
			for _, tt := range []struct {
				item string
				want []string
			}{
				{"Near", []string{"alice: updated relates_to  → OPS-1, none → Far, in WEB, added relates_to relation",
					by + ": deleted relates_to OPS-1 → , Far → none, in WEB, deleted relates_to relation at the moment"}},
				{"Far", []string{"alice: updated relates_to  → WEB-1, none → Near, in OPS, added relates_to relation",
					by + ": deleted relates_to WEB-1 → , Near → none, in OPS, deleted relates_to relation at the moment"}},
			} {
				if got := w.relationRows(t, w.workItemNamed(t, tt.item), at); !slices.Equal(got, tt.want) {
					t.Errorf("the rows of %s's relations:\n%q\nwant\n%q", tt.item, got, tt.want)
				}
			}
		})
}

// Deleting a relation and deleting the work item at its other end
// serialize on that end's project's row (M4 design 3.5, 3.10; interleaving
// 26), in both orders: bob deletes the relation of his Near, of Web, to
// alice's Far, of Ops, which he made, deciding on Near, while alice
// deletes Far. The relation's deletion locks both projects whichever work
// item it is decided on, since it writes on both. Relation deleted first:
// Far's deletion waits for Ops' row, then finds the relation deleted and
// leaves it as bob deleted it. Far deleted first: the relation's deletion,
// which passed its precheck, waits for Ops' row, then reads the relation
// again under its locks, finds it deleted with Far, by alice at Far's
// moment, and is refused, 404 issue.relation_not_found, writing nothing.
func TestDeletingARelationAndItsOtherEndSerialize(t *testing.T) {
	prepare := func(t *testing.T, w issueWorld) {
		nearAndFar(t, w)
		w.send(t, "bob", http.MethodPost, relationsOf(w.workItemNamed(t, "Near")), relating("relates_to", w.workItemNamed(t, "Far")),
			http.StatusCreated)
	}
	inBothOrders(t, prepare, "the relation deleted", unrelates("bob"), "Far deleted", deletesNamed("alice", "Far"),
		func(t *testing.T, w issueWorld, relationFirst bool, unrelated, farDeleted exchange) {
			w.answered(t, farDeleted, http.StatusNoContent)
			relation := queryIDs(t, w.pool, "SELECT id FROM issue_relations")[0]
			by, at := "alice", w.deletedAt(t, "issues", w.workItemNamed(t, "Far"))
			if relationFirst {
				w.answered(t, unrelated, http.StatusNoContent)
				by, at = "bob", w.deletedAt(t, "issue_relations", relation)
			} else {
				w.refused(t, unrelated, http.StatusNotFound, "issue.relation_not_found")
			}
			if got := w.relationDeletion(t, relation, at); got != "deleted by "+by+" at the moment" {
				t.Errorf("the relation is %s, want deleted by %s at the moment", got, by)
			}
		})
}
````

`server/internal/bootstrap/issue_interleaving_test.go`（修改，8 处）：

````old server/internal/bootstrap/issue_interleaving_test.go
// once the second waits for Web's row of projects, which each of the two
// takes FOR NO KEY UPDATE (M4 design 3.5, M3 design 3.6). Every wait has a
// deadline.
````
````new server/internal/bootstrap/issue_interleaving_test.go
// once the second waits for a row the first holds: a project's, Web's or
// another's, which each of the two takes FOR NO KEY UPDATE (M4 design 3.5,
// M3 design 3.6), or one of a table both write (serializeAt). Every wait
// has a deadline.
````
````old server/internal/bootstrap/issue_interleaving_test.go
// serialize sends first, through a route whose transaction a gate holds
````
````new server/internal/bootstrap/issue_interleaving_test.go
// serialize is serializeAt on projects: the second write waits for a
// project's row, Web's or another's.
func (w issueWorld) serialize(t *testing.T, first, second racingWrite, a, b *exchange) {
	t.Helper()
	w.serializeAt(t, "projects", first, second, a, b)
}

// serializeAt sends first, through a route whose transaction a gate holds
````
````old server/internal/bootstrap/issue_interleaving_test.go
// waits for a row of projects, Web's, it opens the gate, at a's released
// (serializeOn). Each answers into its exchange within 10 seconds.
func (w issueWorld) serialize(t *testing.T, first, second racingWrite, a, b *exchange) {
````
````new server/internal/bootstrap/issue_interleaving_test.go
// waits for a row of table, it opens the gate, at a's released
// (serializeOnTable). Each answers into its exchange within 10 seconds.
func (w issueWorld) serializeAt(t *testing.T, table string, first, second racingWrite, a, b *exchange) {
````
````old server/internal/bootstrap/issue_interleaving_test.go
	opened, firstErr, secondErr := serializeOn(t, ctx, w.pool, g, firstWrite, secondWrite)
````
````new server/internal/bootstrap/issue_interleaving_test.go
	opened, firstErr, secondErr := serializeOnTable(t, ctx, w.pool, g, table, firstWrite, secondWrite)
````
````old server/internal/bootstrap/issue_interleaving_test.go
// inBothOrders runs a, then b, and b, then a, each on a world of its own
// that prepare, when given, prepares first (serialize), and checks each
// order's answers with check. aName and bName name the writes in the
// subtests' names.
````
````new server/internal/bootstrap/issue_interleaving_test.go
// inBothOrders is inBothOrdersAt on projects.
````
````old server/internal/bootstrap/issue_interleaving_test.go
	check func(t *testing.T, w issueWorld, aFirst bool, ea, eb exchange)) {
````
````new server/internal/bootstrap/issue_interleaving_test.go
	check func(t *testing.T, w issueWorld, aFirst bool, ea, eb exchange)) {
	inBothOrdersAt(t, "projects", prepare, aName, a, bName, b, check)
}

// inBothOrdersAt runs a, then b, and b, then a, each on a world of its own
// that prepare, when given, prepares first, the second waiting for a row
// of table (serializeAt), and checks each order's answers with check.
// aName and bName name the writes in the subtests' names.
func inBothOrdersAt(t *testing.T, table string, prepare func(t *testing.T, w issueWorld), aName string, a racingWrite, bName string,
	b racingWrite, check func(t *testing.T, w issueWorld, aFirst bool, ea, eb exchange)) {
````
````old server/internal/bootstrap/issue_interleaving_test.go
				w.serialize(t, a, b, &ea, &eb)
````
````new server/internal/bootstrap/issue_interleaving_test.go
				w.serializeAt(t, table, a, b, &ea, &eb)
````
````old server/internal/bootstrap/issue_interleaving_test.go
				w.serialize(t, b, a, &eb, &ea)
````
````new server/internal/bootstrap/issue_interleaving_test.go
				w.serializeAt(t, table, b, a, &eb, &ea)
````

- [ ] **Step 5: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/issue/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestARelationIsDeletedFromEitherWorkItem$|TestWhoMayNotDeleteARelation$|TestTheGuestRuleHoldsForDeletingARelation$|TestOppositeRelationsOfTwoWorkItemsSerialize$|TestDeletingARelatedWorkItemAndTheOtherEndsProjectSerialize$|TestRelatingAWorkItemAndDeletingTheOtherEndsProjectSerialize$|TestRelatingAcrossProjectsAndDeletingTheWorkspaceSerialize$|TestTwoDeletionsOfARelationSerialize$|TestDeletingARelationAndItsOtherEndSerialize$|TestEachIssueWriteLocksItsProjectFirst$|TestEachWriteOnAWorkItemOfAProjectTooBusyAnswersServerBusy$|TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection$|TestPermissionMatrix|TestThePermissionMatrixCoversEveryOperation$' ./internal/bootstrap/`
Expected: `ok`。

Run: `go -C server test -count=5 -race -timeout 30m -run 'TestTwoMovesThatWouldMakeALoopSerialize$|TestAddingSubItemsAndAMoveThatWouldMakeALoopSerialize$|TestDeletingAWorkItemAndPuttingAnotherUnderItSerialize$|TestOppositeRelationsOfTwoWorkItemsSerialize$|TestDeletingARelatedWorkItemAndTheOtherEndsProjectSerialize$|TestRelatingAWorkItemAndDeletingTheOtherEndsProjectSerialize$|TestRelatingAcrossProjectsAndDeletingTheWorkspaceSerialize$|TestTwoDeletionsOfARelationSerialize$|TestDeletingARelationAndItsOtherEndSerialize$|TestARelationLocksNoProjectItDoesNotReach$|TestArchivingAWorkItemAndMovingItSerialize$|TestDeletingAWorkItemAndChangingItSerialize$|TestACallerRemovedOrDemotedWhileHisWriteWaitsIsRefused$|TestTwoCreationsOnAProjectSerialize$|TestTwoChangesOfAWorkItemSerialize$|TestAWorkItemGoneWhileItsChangeWaitsIsNotFound$|TestArchivingAProjectAndCreatingAWorkItemSerialize$|TestRemovingTheDefaultAssigneeAndCreatingAWorkItemSerialize$|TestAWorkItemsWriteAndDeletingItsProjectOrWorkspaceSerialize$|TestDeletingAStateAndPuttingAWorkItemInItSerialize$|TestDeletingALabelAndLabellingAWorkItemWithItSerialize$|TestTwoWritesOnAProjectSerialize$|TestChainedWorkItemWritesHoldOffNoWorkspaceWrite$|TestAHotProjectHoldsUpNoReadOfAnother$|TestEachIssueWriteLocksItsProjectFirst$|TestTheWriteGateAnswersItsRefusalsAsServerBusy$|TestABusyTransactionIsServerBusy$|TestAProjectTooBusyForAWriteAnswersServerBusy$|TestEachWriteOnAWorkItemOfAProjectTooBusyAnswersServerBusy$|TestAWriteWhoseRequestEndsWhileItWaitsAtTheGateIsServerBusy$|TestAWriteAcrossProjectsWaitsMaxWaitInAll$|TestAWriteAcrossProjectsGivesBackEachPlaceItTook$|TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection$' ./internal/bootstrap/`
Expected: `ok`；输出中没有 `40P01`、`deadlock detected`、`DATA RACE`（工作项一侧的每个交错、闸门和连接池的测试、取锁形状，各五遍）。

Run: `go -C server test -count=5 -race ./internal/platform/gate/`
Expected: `ok`（闸门的单元测试五遍，它们的时间窗在负载下也成立）。

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
git add api/modules/issue.yaml api/openapi.yaml server/internal/bootstrap/gate_test.go server/internal/bootstrap/interleaving_test.go server/internal/bootstrap/issue_interleaving_relations_test.go server/internal/bootstrap/issue_interleaving_test.go server/internal/bootstrap/issue_relation_deletes_test.go server/internal/bootstrap/issue_write_locks_test.go server/internal/bootstrap/issue_writes_test.go server/internal/bootstrap/permission_matrix_relations_test.go server/internal/bootstrap/permission_matrix_seed_issues_test.go server/internal/bootstrap/permission_matrix_seed_relations_test.go server/internal/bootstrap/permission_matrix_targets_test.go server/internal/bootstrap/permission_matrix_test.go server/internal/modules/issue/adapter/http/handler.go server/internal/modules/issue/adapter/http/handler_test.go server/internal/modules/issue/adapter/http/relations.go server/internal/modules/issue/adapter/http/relations_test.go server/internal/modules/issue/module.go web/apps/web/core/lib/error-messages.ts web/packages/i18n/src/locales/en/errors.json web/packages/i18n/src/locales/zh-CN/errors.json api/dist/openapi.yaml server/internal/modules/issue/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M4/P3): deleteIssueRelation through the API, the matrix, and the relations' interleavings

DELETE /api/v0/issue-relations/{relation_id} answers 204, and
issue.relation_not_found, which the web app shows in English and
Chinese, when the caller sees neither work item. Opposite relations of
two work items (interleaving 10), deleting a related work item and the
other end's project (11), relating a work item and deleting the other's
project (23), deleting a relation and the work item at its other end
(26), and relating across projects and deleting the workspace serialize
in both orders: on the projects' rows taken all at once, or on the
relation rows taken in id order. The interleaving helpers wait on a row
of any table.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p3.py` 的编号；"层"是发现它的检查所在的层：单元是假实现上的测试，或没有替身的平台、领域、转换的测试；存储是真实数据库上的存储测试；组合是 `bootstrap` 组合出的 app 上的测试；端到端是 `make e2e` 的故事；守卫是生成器（`make gen-go`）和中英文翻译键（`make lint-web`）的核对。每个变异在一份新的副本里，在本 Task 的树上、标出的较晚的 Task 的树上和最终的树上都跑过，端到端的检查只在标出的 Task 的树上）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T14.1` | `DELETE` 不问用例就答 204 | `internal/modules/issue/adapter/http` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestARelationIsDeletedFromEitherWorkItem`（`make test`）、故事 `i9-relations`（Task 15 起） | 单元；组合；端到端 |
| `T14.2` | 契约没有写 `deleteIssueRelation` 回答 `issue.archived` | `internal/modules/issue/adapter/http` 的全部测试（`make test`）、`internal/bootstrap` 的 `TestWhoMayNotDeleteARelation`（`make test`） | 单元；组合 |
| `T14.3` | 页面没有 `issue.relation_not_found` 的文案 | `error-messages.test.ts`（`make test-web`） | 单元 |
| `T14.4` | 中文的语言包没有 `issue_relation_not_found` | `web/packages/i18n` 的 `check:sync`（`make lint-web`） | 守卫 |

本 Task 的检查也发现较早的 Task 的变异 `T7.9`、`T8.8`、`T8.10`、`T8.11`、`T10.7`、`T10.11`、`T12.9`、`T13.1`、`T13.2`、`T13.3`、`T13.5`、`T13.6`、`T13.7`、`T13.8`、`T13.9`、`T13.10`、`T13.11`、`T13.12`、`T13.13`、`T13.14`、`T13.15`、`T13.16`、`T13.17`、`T13.18`、`T13.19`、`T13.23`、`T13.24`、`T13.25`（列在各自的 Task 的表里，标"（Task 14 起）"）：本 Task 做完之后，在本 Task 的树上重跑它们（`mutants_p3.py`，`MUT_LATER=14`）。

---

### Task 15: 端到端：I8、I9、I17 的接口版本，I15 的子工作项和关联，I1、I2 的 `parent`

**Files:**
- Create: `e2e/fixtures/assert/relation.ts`、`e2e/stories/issue/i17-delete-issue.spec.ts`、`e2e/stories/issue/i8-sub-issues.spec.ts`、`e2e/stories/issue/i9-relations.spec.ts`
- Modify: `e2e/fixtures/assert/issue.ts`、`e2e/fixtures/issues.ts`、`e2e/stories/issue/i1-create-issue.spec.ts`、`e2e/stories/issue/i15-archive-issue.spec.ts`、`e2e/stories/issue/i2-update-issue.spec.ts`

**Interfaces:**（spec 2.17、第 3 节第 18 条；M4 设计第 2 节 I8、I9、I15、I17，3.6、3.9、3.10、3.13）只改 `e2e/`，扩展共用的夹具，不复制：
- `e2e/fixtures/issues.ts`：`createIssueRelations` 移到读写的前面（`furnish.ts` 用它）；`issueRead`（状态和工作项，或问题的码）、`subIssuesAdded`、`issueRelationsCreated`、`issueRelationsListed`、`issueRelationDeleted`（各回答状态和回答的体，或问题的码，故事断言它们）；私有的类型 `IssueRelation`、`IssueRelationsCreate`。
- `e2e/fixtures/assert/issue.ts`：`ActivityRow` 的说明；`ItemRef`、`itemRef`；动态的行的构造 `parentRow`（第 5 行）、`relatedRow`（第 20、21 行）、`unrelatedRow`（第 22、23 行），一种行一个构造，经共用的写法；`activityOf` 读出 `old_identifier`、`new_identifier` 指名的工作项的名字（`LEFT JOIN issues`，与标签的同样）；`CreatedIssue`、`UpdatedIssue` 加 `parent`（存下的父工作项的名字，没有是 `null`；P2 的日期同样这样加）；`expectIssueDeleted` 去掉 Task 8 的 `unwritten`：每个外键下都要有一行（`issues.parent_id` 和关联的两个外键在内），说明照改。
- `e2e/fixtures/assert/relation.ts`（新）：`StoredRelation`、`expectRelations`（存下的关联的行：两端、项目、存储的类型、谁、何时，删除的也在）。
- `i8-sub-issues.spec.ts`（新）、`i9-relations.spec.ts`（新）、`i17-delete-issue.spec.ts`（新）：I8、I9、I17 的接口版本（本 Phase 的断言）；`i15-archive-issue.spec.ts`：恢复之后在它下面放一个工作项、两个方向各关联一个；再归档之后加子工作项、加关联、删除它的两条关联（存在它这一端的、存在对方那一端的）都 409，关联留着；管理员删除它时它的子工作项和关联随它删除（`expectIssueDeleted`）；`i1-create-issue.spec.ts`、`i2-update-issue.spec.ts` 的存下的工作项 `parent: null`。

**Tests:** 端到端，`make e2e` 126 个（新的是 I8、I9、I17）：
- I8：成员把两个工作项放到一个下面（请求里不按 `id` 的次序）：按 `id` 的次序回答，各一行她的第 5 行；再放一次什么都不写；在它下面建的没有父工作项的行；另一个成员修改把一个移到别的工作项下、再放一个到那个下面，每行的原值是它原来所在的；把一个拿出到哪里都不在；每个工作项数它直接下面的；放到它下面的下面的、反过来，都 422；已归档的不能加（409）、不能当父工作项（422），都不写。
- I9：成员把一个工作项关联为阻塞本项目的另一个和另一个项目的一个（各存为对方的 `blocked_by`、对方的项目的行）：各一行，每条关联在两端各一行动态、在各自的项目、按各自读的类型；对方列出它是 `blocked_by`；已有关联的一对（任一个方向）、看不到的项目的工作项一次 422；列表里没有调用者看不到的对方；两个项目之一的成员删除跨项目的关联（存在他不能改的那一端，按他能改的这一端删除），两端各多一行：不再列出、再删 404。
- I17：成员删不了别人的（403），访客删不了他自己的（403），离开私有项目的创建者删不了他自己的（404）；成员删除她自己的：它下面两层（下面一层已归档），与另一个项目的两个工作项各一条关联（各存在一端）：它、它下面的每一行（子工作项和两个方向的关联在其中）在那一刻由她删除；三个工作项都不再找得到（404），别的工作项不再列出关联。
- I15（补充）：见上；I1、I2：存下的 `parent` 是 `null`。

- [ ] **Step 1: 夹具**

`e2e/fixtures/assert/issue.ts`（修改，19 处）：

````old e2e/fixtures/assert/issue.ts
 * state or a label by its name.
````
````new e2e/fixtures/assert/issue.ts
 * state, a label or a work item by its name.
````
````old e2e/fixtures/assert/issue.ts
  /** What the row's new_identifier names: the account assigned, the state or the label set; else null. */
````
````new e2e/fixtures/assert/issue.ts
  /**
   * What the row's new_identifier names: the account assigned, the state or the label set, the work item it is put
   * under or related to; else null.
   */
````
````old e2e/fixtures/assert/issue.ts
  /** What its old_identifier names: the account unassigned, the state left or the label taken off; else null. */
````
````new e2e/fixtures/assert/issue.ts
  /**
   * What its old_identifier names: the account unassigned, the state left, the label taken off, the work item it was
   * under or is related to no more; else null.
   */
````
````old e2e/fixtures/assert/issue.ts
export const restoredRow = changed("archived_at", "archive", "restore", "restored the work item");

````
````new e2e/fixtures/assert/issue.ts
export const restoredRow = changed("archived_at", "archive", "restore", "restored the work item");

/** A work item as a row of activity names it: its identifier, as WEB-1, and its name. */
export interface ItemRef {
  identifier: string;
  name: string;
}

/** The work item issue, of the project whose identifier is identifier, as a row of activity names it. */
export function itemRef(identifier: string, issue: { sequence_id: number; name: string }): ItemRef {
  return { identifier: `${identifier}-${issue.sequence_id}`, name: issue.name };
}

/**
 * The row of a work item's activity that moves it from under the work item from to under the work item to, either
 * null for none (M4 design 3.13, row 5): their identifiers its values, "" for none.
 */
export function parentRow(from: ItemRef | null, to: ItemRef | null): ActivityRow {
  const [was, now] = [from?.identifier ?? "", to?.identifier ?? ""];
  return changed("parent", was, now, "updated the parent work item to", to?.name ?? null, from?.name ?? null);
}

/**
 * The row of a work item's activity that relates it to the work item other, of type as the work item reads it (M4
 * design 3.13, rows 20 and 21).
 */
export function relatedRow(type: string, other: ItemRef): ActivityRow {
  return changed(type, "", other.identifier, `added ${type} relation`, other.name);
}

/** The row of a work item's activity that relates it to the work item other no more, as relatedRow (rows 22 and 23). */
export function unrelatedRow(type: string, other: ItemRef): ActivityRow {
  return {
    field: type,
    verb: "deleted",
    old_value: other.identifier,
    new_value: "",
    named: null,
    was: other.name,
    comment: `deleted ${type} relation`,
  };
}

````
````old e2e/fixtures/assert/issue.ts
    `SELECT a.field, a.verb, a.old_value, a.new_value, COALESCE(nu.email, ns.name, nl.name) AS named,
            COALESCE(ou.email, os.name, ol.name) AS was, a.comment
````
````new e2e/fixtures/assert/issue.ts
    `SELECT a.field, a.verb, a.old_value, a.new_value, COALESCE(nu.email, ns.name, nl.name, ni.name) AS named,
            COALESCE(ou.email, os.name, ol.name, oi.name) AS was, a.comment
````
````old e2e/fixtures/assert/issue.ts
       LEFT JOIN labels nl ON nl.id = a.new_identifier
````
````new e2e/fixtures/assert/issue.ts
       LEFT JOIN labels nl ON nl.id = a.new_identifier LEFT JOIN issues ni ON ni.id = a.new_identifier
````
````old e2e/fixtures/assert/issue.ts
       LEFT JOIN labels ol ON ol.id = a.old_identifier
````
````new e2e/fixtures/assert/issue.ts
       LEFT JOIN labels ol ON ol.id = a.old_identifier LEFT JOIN issues oi ON oi.id = a.old_identifier
````
````old e2e/fixtures/assert/issue.ts
 * order; whether it is completed; the address of the account that created it; its assignees and subscribers by
 * address, its labels by name, each in order; and its activity, in its order.
````
````new e2e/fixtures/assert/issue.ts
 * order; the name of the work item it is under, null for none; whether it is completed; the address of the account
 * that created it; its assignees and subscribers by address, its labels by name, each in order; and its activity, in
 * its order.
````
````old e2e/fixtures/assert/issue.ts
  sort_order: number;
````
````new e2e/fixtures/assert/issue.ts
  sort_order: number;
  parent: string | null;
````
````old e2e/fixtures/assert/issue.ts
 * I1: the work item of issueId holds want, as created (M4 design 3.7, 3.13): undeleted, not archived, numbered
 * want.sequence_id, which its project's last_issue_sequence is now. It and every row of its assignees, labels,
 * subscribers and activity are of its project and workspace, made and last written by the account of want.by, the
 * activity's actor too, at one moment.
````
````new e2e/fixtures/assert/issue.ts
 * I1 (I8 from P3): the work item of issueId holds want, as created (M4 design 3.7, 3.9, 3.13): undeleted, not
 * archived, numbered want.sequence_id, which its project's last_issue_sequence is now, and under a work item of its
 * project when it is under one. It and every row of its assignees, labels, subscribers and activity are of its project
 * and workspace, made and last written by the account of want.by, the activity's actor too, at one moment.
````
````old e2e/fixtures/assert/issue.ts
            to_char(i.target_date, 'YYYY-MM-DD') AS target_date, i.sort_order, i.completed_at IS NOT NULL AS completed,
            c.email AS by, p.last_issue_sequence,
````
````new e2e/fixtures/assert/issue.ts
            to_char(i.target_date, 'YYYY-MM-DD') AS target_date, i.sort_order, up.name AS parent,
            i.completed_at IS NOT NULL AS completed, c.email AS by, p.last_issue_sequence,
````
````old e2e/fixtures/assert/issue.ts
              AND i.deleted_at IS NULL AND i.archived_at IS NULL AS as_created
````
````new e2e/fixtures/assert/issue.ts
              AND i.deleted_at IS NULL AND i.archived_at IS NULL AND (up.id IS NULL OR up.project_id = i.project_id)
              AS as_created
````
````old e2e/fixtures/assert/issue.ts
       JOIN users c ON c.id = i.created_by_id
````
````new e2e/fixtures/assert/issue.ts
       JOIN users c ON c.id = i.created_by_id LEFT JOIN issues up ON up.id = i.parent_id
````
````old e2e/fixtures/assert/issue.ts
 * A work item as changed: its name, state (by name), priority and dates (YYYY-MM-DD, null for none); whether it is
 * completed; the address of the account that changed it last; its undeleted assignees and subscribers by address, its
 * undeleted labels by name, each in order; its activity, in its order; and how many rows of its assignees, labels,
 * subscribers and activity its last change wrote, those it took off among them.
````
````new e2e/fixtures/assert/issue.ts
 * A work item as changed: its name, state (by name), priority and dates (YYYY-MM-DD, null for none); the name of the
 * work item it is under, null for none; whether it is completed; the address of the account that changed it last; its
 * undeleted assignees and subscribers by address, its undeleted labels by name, each in order; its activity, in its
 * order; and how many rows of its assignees, labels, subscribers and activity its last change wrote, those it took off
 * among them.
````
````old e2e/fixtures/assert/issue.ts
  start_date: string | null;
  target_date: string | null;
  completed: boolean;
````
````new e2e/fixtures/assert/issue.ts
  start_date: string | null;
  target_date: string | null;
  parent: string | null;
  completed: boolean;
````
````old e2e/fixtures/assert/issue.ts
 * I2: the work item of issueId holds want after a change (M4 design 3.7, 3.13): undeleted, last written by the
 * account of want.by at a moment after its creation, completed (completed_at set) exactly when want.completed. Its
 * undeleted assignees, labels and subscribers and its activity are want's. The rows its last change wrote, those
 * last written at the work item's updated_at, want.wrote of them, are of its project and workspace and want.by's at
 * that moment: those it made, made then; those it took off, soft-deleted then; its activity, acted then.
````
````new e2e/fixtures/assert/issue.ts
 * I2 (I8, I9 from P3): the work item of issueId holds want after a change (M4 design 3.7, 3.9, 3.10, 3.13): undeleted,
 * last written by the account of want.by at a moment after its creation, completed (completed_at set) exactly when
 * want.completed, and under a work item of its project when it is under one. Its undeleted assignees, labels and
 * subscribers and its activity are want's. The rows its last change wrote, those last written at the work item's
 * updated_at, want.wrote of them, are of its project and workspace and want.by's at that moment: those it made, made
 * then; those it took off, soft-deleted then; its activity, acted then.
````
````old e2e/fixtures/assert/issue.ts
            to_char(i.target_date, 'YYYY-MM-DD') AS target_date, i.completed_at IS NOT NULL AS completed, u.email AS by,
            i.updated_at > i.created_at AND i.deleted_at IS NULL AS as_changed
       FROM issues i JOIN states s ON s.id = i.state_id JOIN users u ON u.id = i.updated_by_id
````
````new e2e/fixtures/assert/issue.ts
            to_char(i.target_date, 'YYYY-MM-DD') AS target_date, up.name AS parent,
            i.completed_at IS NOT NULL AS completed, u.email AS by,
            i.updated_at > i.created_at AND i.deleted_at IS NULL AND (up.id IS NULL OR up.project_id = i.project_id)
              AS as_changed
       FROM issues i JOIN states s ON s.id = i.state_id JOIN users u ON u.id = i.updated_by_id
       LEFT JOIN issues up ON up.id = i.parent_id
````
````old e2e/fixtures/assert/issue.ts
 * The keys of a work item's rows that no story writes yet: its sub-items' (M4 design 3.6, 3.9) and its relations' both
 * ways (3.10), which come with P3 (M4 P3 spec, section 3).
 */
const unwritten = new Set(["issues.parent_id", "issue_relations.issue_id", "issue_relations.related_issue_id"]);

/**
 * I15 (I17 from P3): the work item of issueId is deleted by the account of email, and with it, at the same moment and
 * by the same account, every row under it that was not deleted before (M4 design 3.6). A row is under it when a
 * foreign key of its table names the work item or a row under it: the catalog's graph of foreign keys is walked down
 * from issues (keysBelow, which fails on a key of more than one column), so that a row two keys below (a reaction to
 * a comment, which names the comment) is under it too. Each key of the graph has such a row, but those no story writes
 * yet (unwritten); a row deleted before the work item is neither held to its deletion nor checked here (that it keeps
 * its deletion is `TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt`'s).
````
````new e2e/fixtures/assert/issue.ts
 * I15, I17: the work item of issueId is deleted by the account of email, and with it, at the same moment and by the
 * same account, every row under it that was not deleted before (M4 design 3.6). A row is under it when a foreign key
 * of its table names the work item or a row under it: the catalog's graph of foreign keys is walked down from issues
 * (keysBelow, which fails on a key of more than one column), so that a row two keys below (a reaction to a comment,
 * which names the comment, or the work item under a work item under it) is under it too. Each key of the graph has
 * such a row, its sub-items' and its relations' either way among them; a row deleted before the work item is neither
 * held to its deletion nor checked here (that it keeps its deletion is
 * `TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt`'s).
````
````old e2e/fixtures/assert/issue.ts
  ).toEqual(names.map((key) => ({ key, deletedWithIt: !unwritten.has(key), other: 0 })));
````
````new e2e/fixtures/assert/issue.ts
  ).toEqual(names.map((key) => ({ key, deletedWithIt: true, other: 0 })));
````

`e2e/fixtures/assert/relation.ts`（新文件，38 行）：

````file e2e/fixtures/assert/relation.ts
import { expect } from "@playwright/test";

import type { Database } from "../db";

// Database assertions of the relation stories. The page version and the
// API version of a story call the same function (M4 design 2).

/**
 * A relation as stored: "<work item> <type> <other work item>", the work items by name, a work item blocking another
 * stored as the other's blocked_by; the address of the account that wrote it last; and whether it is deleted.
 */
export interface StoredRelation {
  relation: string;
  by: string;
  deleted: boolean;
}

/**
 * I9: the relations stored of the work items of issueIds, either way, are want, in the order they were made (M4 design
 * 3.10): one row a pair of work items, a work item blocking another stored as the other's blocked_by. Each row is of
 * the workspace and the project of its work item (issue_id), the other work item of the same workspace; an undeleted
 * one is as it was made, by the account that made it, and a deleted one was written last at its deletion.
 */
export async function expectRelations(db: Database, issueIds: string[], want: StoredRelation[]): Promise<void> {
  const rows = await db.query(
    `SELECT i.name || ' ' || r.relation_type || ' ' || o.name AS relation, u.email AS by,
            r.deleted_at IS NOT NULL AS deleted,
            r.workspace_id = i.workspace_id AND r.project_id = i.project_id AND o.workspace_id = i.workspace_id
              AND (r.deleted_at IS NULL AND r.updated_by_id = r.created_by_id AND r.updated_at = r.created_at
                   OR r.deleted_at = r.updated_at) AS stamped
       FROM issue_relations r JOIN issues i ON i.id = r.issue_id JOIN issues o ON o.id = r.related_issue_id
       JOIN users u ON u.id = r.updated_by_id
      WHERE r.issue_id = ANY($1::uuid[]) OR r.related_issue_id = ANY($1::uuid[])
      ORDER BY r.created_at, r.id`,
    [issueIds]
  );
  expect(rows, `the relations of ${issueIds.join(", ")}`).toEqual(want.map((r) => ({ ...r, stamped: true })));
}
````

`e2e/fixtures/issues.ts`（修改，6 处）：

````old e2e/fixtures/issues.ts
export type IssueUpdate = components["schemas"]["IssueUpdate"];
````
````new e2e/fixtures/issues.ts
export type IssueUpdate = components["schemas"]["IssueUpdate"];
type IssueRelation = components["schemas"]["IssueRelation"];
type IssueRelationsCreate = components["schemas"]["IssueRelationsCreate"];
````
````old e2e/fixtures/issues.ts
  return created.issue;
````
````new e2e/fixtures/issues.ts
  return created.issue;
}

/** The answer of GET /api/v0/issues/{issue_id} by the caller of token: its status, the work item, or the problem's code. */
export async function issueRead(api: Api, token: string, issueId: string) {
  const { data, error, response } = await api.GET("/api/v0/issues/{issue_id}", {
    params: { path: { issue_id: issueId } },
    headers: bearer(token),
  });
  return data ? { status: response.status, issue: data } : answer(response, error);
````
````old e2e/fixtures/issues.ts

/**
 * Relates the work item of issueId with the bearer token given, a member's of its project, to the work items body
````
````new e2e/fixtures/issues.ts

/**
 * The answer of POST /api/v0/issues/{issue_id}/sub-issues by the caller of token, putting the work items of
 * subIssueIds under the work item of issueId: its status, the work items that moved under it, or the problem's code
 * and fields.
 */
export async function subIssuesAdded(api: Api, token: string, issueId: string, subIssueIds: string[]) {
  const { data, error, response } = await api.POST("/api/v0/issues/{issue_id}/sub-issues", {
    params: { path: { issue_id: issueId } },
    body: { sub_issue_ids: subIssueIds },
    headers: bearer(token),
  });
  return data ? { status: response.status, issues: data.data } : answer(response, error);
}

/**
 * The answer of POST /api/v0/issues/{issue_id}/relations by the caller of token: its status, the relations made, as
 * the work item reads them, or the problem's code and fields.
 */
export async function issueRelationsCreated(api: Api, token: string, issueId: string, body: IssueRelationsCreate) {
  const { data, error, response } = await api.POST("/api/v0/issues/{issue_id}/relations", {
    params: { path: { issue_id: issueId } },
    body,
    headers: bearer(token),
  });
  return data ? { status: response.status, relations: data.data } : answer(response, error);
}

/**
 * Relates the work item of issueId with the bearer token given, a member's of its project, to the work items body
````
````old e2e/fixtures/issues.ts
  body: components["schemas"]["IssueRelationsCreate"]
): Promise<components["schemas"]["IssueRelation"][]> {
  const { data, error, response } = await api.POST("/api/v0/issues/{issue_id}/relations", {
    params: { path: { issue_id: issueId } },
    body,
````
````new e2e/fixtures/issues.ts
  body: IssueRelationsCreate
): Promise<IssueRelation[]> {
  const created = await issueRelationsCreated(api, token, issueId, body);
  expect(created.status, `relate the work item ${issueId}: ${JSON.stringify(created)}`).toBe(201);
  if (!("relations" in created)) {
    throw new Error(`createIssueRelations ${issueId} answered 201 without the relations`);
  }
  return created.relations;
}

/** The relations of the work item of issueId, as it reads them, that the caller of token, who sees it, lists. */
export async function listIssueRelations(api: Api, token: string, issueId: string): Promise<IssueRelation[]> {
  const { data, error, response } = await api.GET("/api/v0/issues/{issue_id}/relations", {
    params: { path: { issue_id: issueId } },
````
````old e2e/fixtures/issues.ts
  expect(response.status, `relate the work item ${issueId}: ${JSON.stringify(error)}`).toBe(201);
````
````new e2e/fixtures/issues.ts
  expect(response.status, `the relations of ${issueId}: ${JSON.stringify(error)}`).toBe(200);
````
````old e2e/fixtures/issues.ts
    throw new Error(`createIssueRelations ${issueId} answered 201 without the relations`);
  }
  return data.data;
}

````
````new e2e/fixtures/issues.ts
    throw new Error(`listIssueRelations ${issueId} answered 200 without the relations`);
  }
  return data.data;
}

/** The answer of DELETE /api/v0/issue-relations/{relation_id} by the caller of token: its status, or the problem's code. */
export async function issueRelationDeleted(api: Api, token: string, relationId: string) {
  const { error, response } = await api.DELETE("/api/v0/issue-relations/{relation_id}", {
    params: { path: { relation_id: relationId } },
    headers: bearer(token),
  });
  return answer(response, error);
}

````

- [ ] **Step 2: I8、I9、I17**

`e2e/stories/issue/i17-delete-issue.spec.ts`（新文件，112 行）：

````file e2e/stories/issue/i17-delete-issue.spec.ts
import {
  addProjectMembers,
  amidAnotherWorkspace,
  createLabel,
  createProject,
  createWorkspace,
  inviteAndAccept,
  projectMemberWrites,
  projectMembershipOf,
  slugFor,
} from "../../fixtures/api";
import { expectIssueDeleted } from "../../fixtures/assert/issue";
import { newAccount } from "../../fixtures/auth";
import {
  createIssue,
  createIssueRelations,
  issueArchived,
  issueDeleted,
  issueRead,
  listIssueRelations,
} from "../../fixtures/issues";
import { stateIds } from "../../fixtures/states";
import { expect, test } from "../../fixtures/test";

// I17, deleting a work item and the work items under it (M4 design 2, 3.4, 3.6, 3.9, 3.10): the API version, with
// P3's assertions (the work item, two levels of work items under it, an archived one among them, and their assignees,
// labels, subscribers, activity and relations either way deleted at one moment by its deleter; who may delete it).
// The comments, reactions, mentions and versions are P4's, the links P7's, the page version P14's.

test("I17 (API): a member does not delete another's work item (403), a guest his own (403), nor a creator who has left the private project his own (404); a member deletes her own, which has two levels of work items under it, the lower archived, and relates to two work items of another project, one relation stored from each end: with it, at that moment and by her, every row under it, the work items under it and its relations either way among them; none of the three is found again (404), and the other work items list no relation", async ({
  api,
  db,
}, testInfo) => {
  const admin = await newAccount(api, testInfo, "admin");
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  // ann, bob, gus and lea, Web's members, ann Ops' too; each a member of Acme.
  const [ann, bob, gus, lea] = await Promise.all(
    ["ann", "bob", "gus", "lea"].map((label) => newAccount(api, testInfo, label))
  );
  if (!ann || !bob || !gus || !lea) {
    throw new Error("the accounts were not registered");
  }
  await Promise.all([ann, bob, gus, lea].map((who) => inviteAndAccept(api, admin.token, slug, who, 15)));
  // Web is private: who leaves it sees none of its work items.
  const [web, ops] = await amidAnotherWorkspace(api, admin.token, testInfo, async () => [
    await createProject(api, admin.token, slug, { name: "Web", identifier: "WEB", network: 0 }),
    await createProject(api, admin.token, slug, { name: "Ops", identifier: "OPS" }),
  ]);
  await addProjectMembers(
    api,
    admin.token,
    web.id,
    [ann, bob, gus, lea].map(({ id }) => ({ member_id: id, role: 15 }))
  );
  await addProjectMembers(api, admin.token, ops.id, [{ member_id: ann.id, role: 15 }]);
  const bug = await createLabel(api, admin.token, web.id, { name: "Bug" });
  const { Done: done } = await stateIds(api, ann.token, web.id);
  if (!done) {
    throw new Error("Web's states hold no Done");
  }

  // ann's Plan the release, assigned to bob and labelled Bug; Write the notes under it, and Check the mirrors, in Done,
  // under that, archived. Plan the release relates to Deploy the site, of Ops, stored from it, and blocks Ship it, of
  // Ops, stored from Ship it as its blocked_by: under Plan the release, a row of each key of the catalog.
  const plan = await createIssue(api, ann.token, web.id, {
    name: "Plan the release",
    assignee_ids: [bob.id],
    label_ids: [bug.id],
  });
  const notes = await createIssue(api, ann.token, web.id, { name: "Write the notes", parent_id: plan.id });
  const mirrors = await createIssue(api, ann.token, web.id, {
    name: "Check the mirrors",
    state_id: done,
    parent_id: notes.id,
  });
  expect((await issueArchived(api, ann.token, mirrors.id)).status).toBe(200);
  const deploy = await createIssue(api, ann.token, ops.id, { name: "Deploy the site" });
  const ship = await createIssue(api, ann.token, ops.id, { name: "Ship it" });
  await createIssueRelations(api, ann.token, plan.id, { relation_type: "relates_to", issue_ids: [deploy.id] });
  await createIssueRelations(api, ann.token, plan.id, { relation_type: "blocking", issue_ids: [ship.id] });

  // gus' Draft the notice, made while he was Web's member, before the admin made him its guest; lea's Tag the build,
  // made before she left Web.
  const notice = await createIssue(api, gus.token, web.id, { name: "Draft the notice" });
  const gusIn = await projectMembershipOf(api, admin.token, web.id, gus.id);
  expect(await projectMemberWrites(api, web.id).change(admin.token, gusIn, 5)).toEqual({ status: 200 });
  const build = await createIssue(api, lea.token, web.id, { name: "Tag the build" });
  expect(await projectMemberWrites(api, web.id).leave(lea.token)).toEqual({ status: 204 });

  // bob, a member, does not delete ann's work item (403), nor gus, a guest, his own (403); lea, who has left Web, finds
  // hers no more (404). Each is there still.
  expect(await issueDeleted(api, bob.token, plan.id)).toEqual({ status: 403, code: "forbidden" });
  expect(await issueDeleted(api, gus.token, notice.id)).toEqual({ status: 403, code: "forbidden" });
  expect(await issueDeleted(api, lea.token, build.id)).toEqual({ status: 404, code: "issue.not_found" });
  const kept = [plan, notice, build];
  expect(await Promise.all(kept.map(({ id }) => issueRead(api, admin.token, id)))).toMatchObject(
    kept.map(({ id }) => ({ status: 200, issue: { id } }))
  );

  // ann deletes Plan the release, her own: with it, at that moment and by her, the two work items under it and every
  // row under the three, its relations either way among them, the one stored from Ship it, of Ops, too.
  expect(await issueDeleted(api, ann.token, plan.id)).toEqual({ status: 204 });
  await expectIssueDeleted(db, plan.id, ann.email);
  // None of the three is found again, and neither Deploy the site nor Ship it lists a relation.
  const gone = [plan, notes, mirrors];
  expect(await Promise.all(gone.map(({ id }) => issueRead(api, ann.token, id)))).toEqual(
    gone.map(() => ({ status: 404, code: "issue.not_found" }))
  );
  expect(await listIssueRelations(api, ann.token, deploy.id)).toEqual([]);
  expect(await listIssueRelations(api, ann.token, ship.id)).toEqual([]);
});
````

`e2e/stories/issue/i8-sub-issues.spec.ts`（新文件，167 行）：

````file e2e/stories/issue/i8-sub-issues.spec.ts
import {
  addProjectMembers,
  amidAnotherWorkspace,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
} from "../../fixtures/api";
import {
  archivedRow,
  byId,
  createdFor,
  expectIssueArchival,
  expectIssueCreated,
  expectIssueUpdated,
  itemRef,
  parentRow,
} from "../../fixtures/assert/issue";
import { newAccount } from "../../fixtures/auth";
import {
  createIssue,
  issueArchived,
  issueRead,
  issueUpdated,
  subIssuesAdded,
  updateIssue,
} from "../../fixtures/issues";
import { stateIds } from "../../fixtures/states";
import { expect, test } from "../../fixtures/test";

// I8, the work items under a work item (M4 design 2, 3.9, 3.13): the API version, with P3's assertions (work items
// put under one, one created under it, moved under another and from under any, each move a row of the moved work
// item's activity, the count of the work items directly under one, a work item above refused under one below it, an
// archived work item taking none). The page version is P14's.

test("I8 (API): a member puts two work items under one, named out of their ids' order: they answer in id order, each moved with a row of its activity, hers; put under it again, nothing is written; one she creates under it has no row of its parent; another member moves one under another work item by changing it, and puts one under that other, each row's old value the work item it was under; she takes one from under any; each work item counts those directly under it; a work item is refused under one below it, either way (422), and an archived one takes none (409) nor becomes a parent (422), writing nothing", async ({
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
  const { Done: done } = await stateIds(api, ann.token, web.id);
  if (!done) {
    throw new Error("Web's states hold no Done");
  }
  // ann's Plan the release, Write the notes and Tag the build, in Backlog, and Ship it, in Done.
  const plan = await createIssue(api, ann.token, web.id, { name: "Plan the release" });
  const notes = await createIssue(api, ann.token, web.id, { name: "Write the notes" });
  const tag = await createIssue(api, ann.token, web.id, { name: "Tag the build" });
  const ship = await createIssue(api, ann.token, web.id, { name: "Ship it", state_id: done });
  const [planRef, shipRef] = [itemRef("WEB", plan), itemRef("WEB", ship)];
  // A work item of ann's in Backlog, as it is while only the work item it is under changes.
  const asMade = (name: string) => ({
    name,
    state: "Backlog",
    priority: "none",
    start_date: null,
    target_date: null,
    completed: false,
    assignees: [],
    labels: [],
    subscribers: [ann.email],
  });
  const counted = async (issueId: string, count: number) =>
    expect(await issueRead(api, ann.token, issueId)).toMatchObject({ status: 200, issue: { sub_issues_count: count } });

  // ann puts Tag the build and Write the notes under Plan the release, named out of their ids' order: they answer in
  // id order, under it; each has one row more, hers, from under none to under Plan the release.
  expect(await subIssuesAdded(api, ann.token, plan.id, [tag.id, notes.id].toSorted().toReversed())).toMatchObject({
    status: 200,
    issues: [notes, tag].toSorted(byId).map(({ id }) => ({ id, parent_id: plan.id })),
  });
  const notesUnder = {
    ...asMade("Write the notes"),
    parent: "Plan the release",
    by: ann.email,
    activity: [...createdFor(), parentRow(null, planRef)],
    wrote: 1,
  };
  await expectIssueUpdated(db, notes.id, notesUnder);
  await expectIssueUpdated(db, tag.id, { ...notesUnder, name: "Tag the build" });
  await counted(plan.id, 2);
  // Put under it again, Write the notes does not move: none answered, nothing written.
  expect(await subIssuesAdded(api, ann.token, plan.id, [notes.id])).toEqual({ status: 200, issues: [] });
  await expectIssueUpdated(db, notes.id, notesUnder);

  // ann creates Draft the notice under Plan the release: under it, its activity only its creation's row.
  const draft = await createIssue(api, ann.token, web.id, { name: "Draft the notice", parent_id: plan.id });
  await expectIssueCreated(db, draft.id, {
    ...asMade("Draft the notice"),
    sequence_id: 5,
    sort_order: draft.sort_order,
    parent: "Plan the release",
    by: ann.email,
    activity: createdFor(),
  });
  await counted(plan.id, 3);

  // bob moves Write the notes under Ship it by changing it, and puts Tag the build under Ship it: each a row of his,
  // from under Plan the release (Plane wrote the work item moved as the one it was under) to under Ship it.
  expect(await updateIssue(api, bob.token, notes.id, { parent_id: ship.id })).toMatchObject({ parent_id: ship.id });
  const notesMoved = {
    ...notesUnder,
    parent: "Ship it",
    by: bob.email,
    activity: [...notesUnder.activity, parentRow(planRef, shipRef)],
  };
  await expectIssueUpdated(db, notes.id, notesMoved);
  expect(await subIssuesAdded(api, bob.token, ship.id, [tag.id])).toMatchObject({
    status: 200,
    issues: [{ id: tag.id, parent_id: ship.id }],
  });
  await expectIssueUpdated(db, tag.id, { ...notesMoved, name: "Tag the build" });
  // ann takes Draft the notice from under any: its row's new value is "".
  await updateIssue(api, ann.token, draft.id, { parent_id: null });
  const draftAlone = {
    ...asMade("Draft the notice"),
    parent: null,
    by: ann.email,
    activity: [...createdFor(), parentRow(planRef, null)],
    wrote: 1,
  };
  await expectIssueUpdated(db, draft.id, draftAlone);
  await counted(plan.id, 0);

  // Check the mirrors, ann's, under Write the notes: Ship it counts the two directly under it, not the one under one of
  // them. It is above Check the mirrors: neither put under it nor made its parent by changing it (422), unchanged.
  const mirrors = await createIssue(api, ann.token, web.id, { name: "Check the mirrors", parent_id: notes.id });
  await counted(ship.id, 2);
  expect(await subIssuesAdded(api, ann.token, mirrors.id, [ship.id])).toEqual({
    status: 422,
    code: "validation_failed",
    errors: [{ field: "sub_issue_ids[0]", code: "not_allowed" }],
  });
  expect(await issueUpdated(api, ann.token, ship.id, { parent_id: mirrors.id })).toEqual({
    status: 422,
    code: "validation_failed",
    errors: [{ field: "parent_id", code: "not_allowed" }],
  });
  await expectIssueArchival(db, ship.id, { archived: false, by: ann.email, activity: createdFor() });

  // ann archives Ship it: it takes Draft the notice under it no more (409 issue.archived), nor does changing Draft the
  // notice make it its parent (422); neither writes anything.
  expect((await issueArchived(api, ann.token, ship.id)).status).toBe(200);
  expect(await subIssuesAdded(api, ann.token, ship.id, [draft.id])).toEqual({ status: 409, code: "issue.archived" });
  expect(await issueUpdated(api, ann.token, draft.id, { parent_id: ship.id })).toEqual({
    status: 422,
    code: "validation_failed",
    errors: [{ field: "parent_id", code: "not_allowed" }],
  });
  await expectIssueArchival(db, ship.id, { archived: true, by: ann.email, activity: [...createdFor(), archivedRow] });
  await expectIssueUpdated(db, draft.id, draftAlone);
});
````

`e2e/stories/issue/i9-relations.spec.ts`（新文件，173 行）：

````file e2e/stories/issue/i9-relations.spec.ts
import {
  addProjectMembers,
  amidAnotherWorkspace,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
} from "../../fixtures/api";
import { createdFor, expectIssueUpdated, itemRef, relatedRow, unrelatedRow } from "../../fixtures/assert/issue";
import { expectRelations } from "../../fixtures/assert/relation";
import { newAccount } from "../../fixtures/auth";
import {
  createIssue,
  createIssueRelations,
  issueRelationDeleted,
  issueRelationsCreated,
  listIssueRelations,
} from "../../fixtures/issues";
import { expect, test } from "../../fixtures/test";

// I9, relations (M4 design 2, 3.10, 3.13): the API version, with P3's assertions (a relation within a project and one
// across two, one row each, a work item blocking another stored as the other's blocked_by, each relation's two rows of
// activity, the reverse one of the other's project, the refusals, the lists, a deletion from either work item). The
// page version is P14's.

test("I9 (API): a member relates a work item blocking another of its project and one of another project, each stored as the other's blocked_by, of the other's project: one row each, and each relation a row of the activity of each work item, of the work item's project, as the work item reads it; the other work item lists it as blocked_by; a pair related already, either way, and a work item of a project she does not see are refused at once (422); a list holds no relation to a work item its caller does not see; a member of one of the two projects deletes the relation across them, a row more on each work item: it is listed no more, nor found again (404)", async ({
  api,
  db,
}, testInfo) => {
  const admin = await newAccount(api, testInfo, "admin");
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  // ann, Web's member and Ops' one; bob, Web's member alone; each a member of Acme.
  const [ann, bob] = await Promise.all(["ann", "bob"].map((label) => newAccount(api, testInfo, label)));
  if (!ann || !bob) {
    throw new Error("the accounts were not registered");
  }
  await Promise.all([ann, bob].map((who) => inviteAndAccept(api, admin.token, slug, who, 15)));
  const [web, ops, lab] = await amidAnotherWorkspace(api, admin.token, testInfo, async () => [
    await createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" }),
    await createProject(api, admin.token, slug, { name: "Ops", identifier: "OPS" }),
    // Lab is private, and the admin its one member.
    await createProject(api, admin.token, slug, { name: "Lab", identifier: "LAB", network: 0 }),
  ]);
  await addProjectMembers(api, admin.token, web.id, [
    { member_id: ann.id, role: 15 },
    { member_id: bob.id, role: 15 },
  ]);
  await addProjectMembers(api, admin.token, ops.id, [{ member_id: ann.id, role: 15 }]);
  // ann's Fix the login and Test the login in Web, Deploy the site in Ops; the admin's Probe the cache in Lab.
  const fix = await createIssue(api, ann.token, web.id, { name: "Fix the login" });
  const check = await createIssue(api, ann.token, web.id, { name: "Test the login" });
  const deploy = await createIssue(api, ann.token, ops.id, { name: "Deploy the site" });
  const probe = await createIssue(api, admin.token, lab.id, { name: "Probe the cache" });
  const [fixRef, checkRef, deployRef] = [itemRef("WEB", fix), itemRef("WEB", check), itemRef("OPS", deploy)];
  // A work item of ann's, as it is while only its relations change.
  const asMade = (name: string) => ({
    name,
    state: "Backlog",
    priority: "none",
    start_date: null,
    target_date: null,
    parent: null,
    completed: false,
    assignees: [],
    labels: [],
    subscribers: [ann.email],
  });

  // ann relates Fix the login blocking Test the login, then blocking Deploy the site: each answered as Fix the login
  // reads it, each stored as the other's blocked_by, of the other's project: Deploy the site's of Ops. Each work item
  // has one row more for each relation, hers, of its own project, as it reads the relation: Deploy the site's of Ops,
  // blocked_by.
  expect(
    await createIssueRelations(api, ann.token, fix.id, { relation_type: "blocking", issue_ids: [check.id] })
  ).toMatchObject([{ relation_type: "blocking", issue: { id: check.id, project_id: web.id } }]);
  const [across] = await createIssueRelations(api, ann.token, fix.id, {
    relation_type: "blocking",
    issue_ids: [deploy.id],
  });
  if (!across) {
    throw new Error("relating Fix the login to Deploy the site answered no relation");
  }
  expect(across).toMatchObject({ relation_type: "blocking", issue: { id: deploy.id, project_id: ops.id } });
  const stored = [
    { relation: "Test the login blocked_by Fix the login", by: ann.email, deleted: false },
    { relation: "Deploy the site blocked_by Fix the login", by: ann.email, deleted: false },
  ];
  await expectRelations(db, [fix.id], stored);
  const fixRelated = {
    ...asMade("Fix the login"),
    by: ann.email,
    activity: [...createdFor(), relatedRow("blocking", checkRef), relatedRow("blocking", deployRef)],
    wrote: 1,
  };
  await expectIssueUpdated(db, fix.id, fixRelated);
  await expectIssueUpdated(db, check.id, {
    ...asMade("Test the login"),
    by: ann.email,
    activity: [...createdFor(), relatedRow("blocked_by", fixRef)],
    wrote: 1,
  });
  const deployRelated = {
    ...asMade("Deploy the site"),
    by: ann.email,
    activity: [...createdFor(), relatedRow("blocked_by", fixRef)],
    wrote: 1,
  };
  await expectIssueUpdated(db, deploy.id, deployRelated);
  // bob lists Test the login's relations: Fix the login blocks it.
  expect(await listIssueRelations(api, bob.token, check.id)).toMatchObject([
    { relation_type: "blocked_by", issue: { id: fix.id } },
  ]);

  // Probe the cache, of Lab, which ann does not see, and Test the login, which Fix the login relates to already, the
  // other way round, are refused at once; nothing is written.
  expect(
    await issueRelationsCreated(api, ann.token, check.id, { relation_type: "duplicate", issue_ids: [probe.id, fix.id] })
  ).toEqual({
    status: 422,
    code: "validation_failed",
    errors: [
      { field: "issue_ids[0]", code: "not_allowed" },
      { field: "issue_ids[1]", code: "duplicate" },
    ],
  });
  await expectRelations(db, [fix.id, check.id], stored);

  // The admin relates Fix the login to Probe the cache: ann's list of Fix the login's relations holds it not, his does.
  await createIssueRelations(api, admin.token, fix.id, { relation_type: "relates_to", issue_ids: [probe.id] });
  expect(await listIssueRelations(api, ann.token, fix.id)).toMatchObject([
    { relation_type: "blocking", issue: { id: check.id } },
    { relation_type: "blocking", issue: { id: deploy.id } },
  ]);
  expect(await listIssueRelations(api, admin.token, fix.id)).toMatchObject([
    { relation_type: "blocking", issue: { id: check.id } },
    { relation_type: "blocking", issue: { id: deploy.id } },
    { relation_type: "relates_to", issue: { id: probe.id } },
  ]);

  // bob, who is no member of Ops, deletes the relation of Fix the login and Deploy the site: stored as Deploy the
  // site's, which he may not change, it is deleted as he may change Fix the login. The relation is deleted by him, and
  // each work item has a row more, his, as it reads the relation, Deploy the site's of Ops. Neither lists it, and it
  // is not found again.
  expect(await issueRelationDeleted(api, bob.token, across.id)).toEqual({ status: 204 });
  await expectRelations(
    db,
    [deploy.id],
    [{ relation: "Deploy the site blocked_by Fix the login", by: bob.email, deleted: true }]
  );
  await expectIssueUpdated(db, fix.id, {
    ...fixRelated,
    by: bob.email,
    activity: [
      ...fixRelated.activity,
      relatedRow("relates_to", itemRef("LAB", probe)),
      unrelatedRow("blocking", deployRef),
    ],
  });
  await expectIssueUpdated(db, deploy.id, {
    ...deployRelated,
    by: bob.email,
    activity: [...deployRelated.activity, unrelatedRow("blocked_by", fixRef)],
  });
  expect(await listIssueRelations(api, ann.token, deploy.id)).toEqual([]);
  expect(await listIssueRelations(api, ann.token, fix.id)).toMatchObject([
    { relation_type: "blocking", issue: { id: check.id } },
  ]);
  expect(await issueRelationDeleted(api, bob.token, across.id)).toEqual({
    status: 404,
    code: "issue.relation_not_found",
  });
});
````

- [ ] **Step 3: I15、I1、I2**

`e2e/stories/issue/i1-create-issue.spec.ts`（修改，4 处）：

````old e2e/stories/issue/i1-create-issue.spec.ts
    target_date: "2026-10-19",
    sort_order: 65535,
````
````new e2e/stories/issue/i1-create-issue.spec.ts
    target_date: "2026-10-19",
    sort_order: 65535,
    parent: null,
````
````old e2e/stories/issue/i1-create-issue.spec.ts
    priority: "none",
    start_date: null,
    target_date: null,
    sort_order: 65535,
    completed: false,
    by: bob.email,
    assignees: [ann.email],
````
````new e2e/stories/issue/i1-create-issue.spec.ts
    priority: "none",
    start_date: null,
    target_date: null,
    sort_order: 65535,
    parent: null,
    completed: false,
    by: bob.email,
    assignees: [ann.email],
````
````old e2e/stories/issue/i1-create-issue.spec.ts
    sort_order: 75535,
````
````new e2e/stories/issue/i1-create-issue.spec.ts
    sort_order: 75535,
    parent: null,
````
````old e2e/stories/issue/i1-create-issue.spec.ts
    priority: "none",
    start_date: null,
    target_date: null,
    sort_order: 65535,
    completed: false,
    by: bob.email,
    assignees: [],
````
````new e2e/stories/issue/i1-create-issue.spec.ts
    priority: "none",
    start_date: null,
    target_date: null,
    sort_order: 65535,
    parent: null,
    completed: false,
    by: bob.email,
    assignees: [],
````

`e2e/stories/issue/i15-archive-issue.spec.ts`（修改，8 处）：

````old e2e/stories/issue/i15-archive-issue.spec.ts
  expectIssueDeleted,
````
````new e2e/stories/issue/i15-archive-issue.spec.ts
  expectIssueDeleted,
  itemRef,
  relatedRow,
````
````old e2e/stories/issue/i15-archive-issue.spec.ts
  createIssue,
````
````new e2e/stories/issue/i15-archive-issue.spec.ts
  createIssue,
  createIssueRelations,
````
````old e2e/stories/issue/i15-archive-issue.spec.ts
  issueDeleted,
````
````new e2e/stories/issue/i15-archive-issue.spec.ts
  issueDeleted,
  issueRelationDeleted,
  issueRelationsCreated,
````
````old e2e/stories/issue/i15-archive-issue.spec.ts
  issueUpdated,
````
````new e2e/stories/issue/i15-archive-issue.spec.ts
  issueUpdated,
  subIssuesAdded,
````
````old e2e/stories/issue/i15-archive-issue.spec.ts
// refused). The archived list is P5's, the page version P13's. The archived work item's deletion at the end is P2's
// deleteIssue (3.6), whose rows expectIssueDeleted reads from the catalog.
````
````new e2e/stories/issue/i15-archive-issue.spec.ts
// refused) and P3's (an archived work item takes no work item under it, relates to none and keeps its relations). The
// archived list is P5's, the page version P13's. The archived work item's deletion at the end is P2's deleteIssue
// (3.6), whose rows expectIssueDeleted reads from the catalog: the work item under it and its relations either way
// among them, from P3.
````
````old e2e/stories/issue/i15-archive-issue.spec.ts
test("I15 (API): a member's work item in a started state is not archived (409); moved to a completed state, she archives it, and archiving it again writes nothing; it is not changed while it is archived (409); a guest who sees it neither restores nor archives it (403); she restores it, and restoring it again writes nothing; each archiving and restoring one row of its activity, hers at its moment; archived again, the admin deletes it, with every row under it, at one moment", async ({
````
````new e2e/stories/issue/i15-archive-issue.spec.ts
test("I15 (API): a member's work item in a started state is not archived (409); moved to a completed state, she archives it, and archiving it again writes nothing; it is not changed while it is archived (409); a guest who sees it neither restores nor archives it (403); she restores it, and restoring it again writes nothing; each archiving and restoring one row of its activity, hers at its moment; restored, a work item is put under it and it is related to two others, either way; archived again, it takes no work item under it, relates to none and keeps its relations (409), and the admin deletes it, with every row under it, at one moment", async ({
````
````old e2e/stories/issue/i15-archive-issue.spec.ts
  // Archived again, one row more, hers at its moment, it is deleted as any other: the admin deletes it, and with it
  // every row under it.
````
````new e2e/stories/issue/i15-archive-issue.spec.ts
  // Restored, it takes ann's Write the test under it, and she relates it to her Write the spec, stored from it, and
  // blocking her Ship the fix, stored from Ship the fix as its blocked_by: a row more of its activity each.
  const part = await createIssue(api, ann.token, web.id, { name: "Write the test" });
  const spec = await createIssue(api, ann.token, web.id, { name: "Write the spec" });
  const ship = await createIssue(api, ann.token, web.id, { name: "Ship the fix" });
  expect(await subIssuesAdded(api, ann.token, id, [part.id])).toMatchObject({ status: 200, issues: [{ id: part.id }] });
  const [toSpec] = await createIssueRelations(api, ann.token, id, {
    relation_type: "relates_to",
    issue_ids: [spec.id],
  });
  if (!toSpec) {
    throw new Error("relating Fix the login to Write the spec answered no relation");
  }
  const [toShip] = await createIssueRelations(api, ann.token, id, { relation_type: "blocking", issue_ids: [ship.id] });
  if (!toShip) {
    throw new Error("relating Fix the login to Ship the fix answered no relation");
  }
  const related = {
    ...restored,
    activity: [
      ...restored.activity,
      relatedRow("relates_to", itemRef("WEB", spec)),
      relatedRow("blocking", itemRef("WEB", ship)),
    ],
  };
  await expectIssueArchival(db, id, related);

  // Archived again, one row more, hers at its moment: it takes Fix the logout under it no more, relates to it no more,
  // and neither of its relations is deleted (409 issue.archived), the one stored from it nor the one stored from Ship
  // the fix; none of the four writes anything.
````
````old e2e/stories/issue/i15-archive-issue.spec.ts
  await expectIssueArchival(db, id, { archived: true, by: ann.email, activity: [...restored.activity, archivedRow] });
````
````new e2e/stories/issue/i15-archive-issue.spec.ts
  const archivedAgain = { archived: true, by: ann.email, activity: [...related.activity, archivedRow] };
  await expectIssueArchival(db, id, archivedAgain);
  const logout = await createIssue(api, ann.token, web.id, { name: "Fix the logout" });
  const refused = { status: 409, code: "issue.archived" };
  expect(await subIssuesAdded(api, ann.token, id, [logout.id])).toEqual(refused);
  expect(
    await issueRelationsCreated(api, ann.token, id, { relation_type: "relates_to", issue_ids: [logout.id] })
  ).toEqual(refused);
  expect(await issueRelationDeleted(api, ann.token, toSpec.id)).toEqual(refused);
  expect(await issueRelationDeleted(api, ann.token, toShip.id)).toEqual(refused);
  await expectIssueArchival(db, id, archivedAgain);

  // It is deleted as any other: the admin deletes it, and with it every row under it, Write the test and its relations
  // either way among them.
````

`e2e/stories/issue/i2-update-issue.spec.ts`（修改，4 处）：

````old e2e/stories/issue/i2-update-issue.spec.ts
    start_date: null,
    target_date: null,
    completed: false,
````
````new e2e/stories/issue/i2-update-issue.spec.ts
    start_date: null,
    target_date: null,
    parent: null,
    completed: false,
````
````old e2e/stories/issue/i2-update-issue.spec.ts
    priority: "high",
    start_date: null,
    target_date: null,
    completed: true,
    by: ann.email,
````
````new e2e/stories/issue/i2-update-issue.spec.ts
    priority: "high",
    start_date: null,
    target_date: null,
    parent: null,
    completed: true,
    by: ann.email,
````
````old e2e/stories/issue/i2-update-issue.spec.ts
    priority: "high",
    start_date: null,
    target_date: null,
    completed: true,
    by: bob.email,
````
````new e2e/stories/issue/i2-update-issue.spec.ts
    priority: "high",
    start_date: null,
    target_date: null,
    parent: null,
    completed: true,
    by: bob.email,
````
````old e2e/stories/issue/i2-update-issue.spec.ts
    sort_order: logout.sort_order,
````
````new e2e/stories/issue/i2-update-issue.spec.ts
    sort_order: logout.sort_order,
    parent: null,
````

- [ ] **Step 4: 检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 `keywords: 72 rules, 23 exceptions, no hits.`

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 126 个全部通过（新的是 I8、I9、I17）。

- [ ] **Step 5: 提交**

```bash
git add e2e/fixtures/assert/issue.ts e2e/fixtures/assert/relation.ts e2e/fixtures/issues.ts e2e/stories/issue/i1-create-issue.spec.ts e2e/stories/issue/i15-archive-issue.spec.ts e2e/stories/issue/i17-delete-issue.spec.ts e2e/stories/issue/i2-update-issue.spec.ts e2e/stories/issue/i8-sub-issues.spec.ts e2e/stories/issue/i9-relations.spec.ts
```
```bash
git commit -m "test(M4/P3): the stories of sub-items, relations and deleting a work item

I8 puts work items under one and moves them, each with a row of its
parent, and is refused a loop and an archived parent. I9 relates work
items in one project and across two, each relation one row and a row on
each end, lists what the caller sees and deletes a relation from either
end. I17 deletes a work item with two levels under it and relations
either way, every row at one moment, after the refusals of its rule.
I15 relates and puts work items under the restored work item, is refused
both once it is archived, and deletes it with them; the deletion's
assertion has no exemption left.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**：本 Task 只加故事和夹具，没有自己的变异：它的故事是较早的 Task 的变异的端到端一层。

本 Task 的检查也发现较早的 Task 的变异 `T1.1`、`T1.5`、`T1.9`、`T2.2`、`T2.5`、`T2.7`、`T2.10`、`T3.5`、`T3.6`、`T3.7`、`T3.10`、`T4.8`、`T4.11`、`T5.9`、`T5.19`、`T8.5`、`T8.10`、`T8.11`、`T9.1`、`T9.2`、`T9.3`、`T9.7`、`T9.9`、`T10.12`、`T10.14`、`T10.15`、`T10.16`、`T10.17`、`T10.18`、`T11.2`、`T12.8`、`T12.9`、`T13.11`、`T13.13`、`T13.14`、`T13.15`、`T13.17`、`T13.18`、`T14.1`（列在各自的 Task 的表里，标"（Task 15 起）"）：本 Task 做完之后，在本 Task 的树上重跑它们（`mutants_p3.py`，`MUT_LATER=15`）。

---

### Task 16: 文档：3.23 中 P3 的行，两份交接的处理结果

**Files:**
- Modify: `docs/v0/M4-issue-core/handoffs/M2-closeout.md`、`docs/v0/M4-issue-core/handoffs/M3-closeout.md`、`docs/v0/plane-diff.md`、`docs/v0/v0-design.md`

**Interfaces:**（spec 2.18、第 7 节；M4 设计 3.23、4.17，12 节 P3 的第 11 个任务）
- 总体设计 3.4：集合型的列表加上一个工作项的子工作项、关联、链接、表情（第一个是 `listSubIssues`）；3.6：第二个例外，关联内嵌对方工作项的列表行（`IssueRelation.issue`）；4.2：约定五的例外加上跨项目的关联行，约定七（跨项目的写按 `id` 升序同时锁住涉及的每个项目，加关联时看不到的对方不锁、不进闸门）。
- 差异清单：第二节 `issue_relations` 的四行（照搬 11 列；外键的级联和默认；类型的检查、不能关联自己；不分方向的一对一条的键）；第三节关联字段的第二个例外；第四节"访客只看自己的工作项""已归档的工作项""删除工作区"三行的 P3 部分（访客一行写上：回答里的 `parent_id` 指名他看不到的工作项时为 `null`），新的"删除工作项"（子树整个删除，已归档的、删除者看不到的也算）"父工作项""加子工作项的动态""子工作项数""关联""关联的动态"六行。
- 两份交接（`M2-closeout.md` 第 3、4 节，`M3-closeout.md` 第 2、3、4 节）写下"处理结果（M4/P3）"，`status` 照旧 `open`（spec 第 7 节）；`M3-closeout.md` 的结果写上 `parent_id` 按调用者的访客规则回答和它的组合测试 `TestAHiddenParentIsNotInTheAnswer`，五遍 `-race` 的交错写上 9 的第二对（加子工作项与移动）和 26。

**Tests:** 只改文档：`make lint-web`（关键词守卫读文档）和 `doclinks.py`（相对链接，本 spec 和 plan 也在内）。

- [ ] **Step 1: 设计和差异清单**

`docs/v0/plane-diff.md`（修改，6 处）：

````old docs/v0/plane-diff.md
| `issue_labels` | **新增**部分唯一索引 `issue_labels_issue_id_label_id_key ON (issue_id, label_id) WHERE deleted_at IS NULL`；外键列的索引只留不带条件的 `issue_labels_issue_id_idx`、`issue_labels_label_id_idx`、`issue_labels_workspace_id_idx`、`issue_labels_project_id_idx` | Plane 在数据库层没有这个约束，同一个标签可以挂两次（M4 设计 4.4）；物理级联要不带条件的索引，标签的也是 |
````
````new docs/v0/plane-diff.md
| `issue_labels` | **新增**部分唯一索引 `issue_labels_issue_id_label_id_key ON (issue_id, label_id) WHERE deleted_at IS NULL`；外键列的索引只留不带条件的 `issue_labels_issue_id_idx`、`issue_labels_label_id_idx`、`issue_labels_workspace_id_idx`、`issue_labels_project_id_idx` | Plane 在数据库层没有这个约束，同一个标签可以挂两次（M4 设计 4.4）；物理级联要不带条件的索引，标签的也是 |
| `issue_relations` | 11 列照搬（M4/P3，`00020_issue_issue_relations.sql`） | M4 设计 4.6 |
| `issue_relations` | `workspace_id`、`project_id`、`issue_id`、`related_issue_id`：加上 `ON DELETE CASCADE`；`created_by_id`、`updated_by_id`：加上 `ON DELETE SET NULL`；`created_at`、`updated_at`：新加 `DEFAULT now()`（兜底，见二·全局）；`project_id` 是存储的 `issue_id` 那一端的项目 | 二·全局（模型的 `on_delete`）；Plane 写请求路径里的项目（M4 设计 3.10） |
| `issue_relations` | `relation_type`：不设默认值（Plane 的模型默认 `blocked_by`），新加 `CHECK (relation_type IN ('blocked_by', 'duplicate', 'relates_to'))`；新加 `issue_relations_not_self_check CHECK (issue_id <> related_issue_id)` | 只有页面的四种类型，阻塞反过来存为对方的 `blocked_by`；工作项不能关联自己（M4 设计 3.10、4.6） |
| `issue_relations` | 部分唯一索引 `issue_relation_unique_issue_related_issue_when_deleted_at_null ON (issue_id, related_issue_id)` 改为不分方向的 `issue_relations_pair_key ON (LEAST(issue_id, related_issue_id), GREATEST(issue_id, related_issue_id)) WHERE deleted_at IS NULL`，去掉 `(issue_id, related_issue_id, deleted_at)` 的唯一约束（二·全局）；外键列的索引只留不带条件的 `issue_relations_issue_id_idx`、`issue_relations_related_issue_id_idx`、`issue_relations_workspace_id_idx`、`issue_relations_project_id_idx` | 一对工作项之间至多一条关联，不论类型和方向（Plane 的键有方向，A 被 B 阻塞和 B 被 A 阻塞可以并存）；两个方向的读取和物理级联按两列找行（M4 设计 3.10、4.6） |
````
````old docs/v0/plane-diff.md
| 关联字段 | 名字不带 `_id`（`workspace`、`member`、`project_lead`），有的内嵌对象 | 只给 id，名字带 `_id`（`WorkspaceMember.workspace_id`）；唯一的例外是工作区成员内嵌成员的公开资料 `member`（`MemberUser`，v0-design 3.6；M3 设计 5.2） |
````
````new docs/v0/plane-diff.md
| 关联字段 | 名字不带 `_id`（`workspace`、`member`、`project_lead`），有的内嵌对象 | 只给 id，名字带 `_id`（`WorkspaceMember.workspace_id`）；例外有两个：工作区成员内嵌成员的公开资料 `member`（`MemberUser`，v0-design 3.6；M3 设计 5.2），关联内嵌对方工作项的列表行 `issue`（`IssueRelation.issue`，M4/P3；M4 设计 3.10） |
````
````old docs/v0/plane-diff.md
| 访客只看自己的工作项 | 只在约 6 处执行 | 项目没开 `guest_view_all_features` 时，它的访客（按项目角色算，工作区管理员在项目里是访客时同样）只看得到自己创建的工作项：取、按编号取、修改、删除、归档、恢复别人的都答 404 `issue.not_found`；读在判定之后执行这条规则，写在不加锁的预检和锁下的判定中各执行一次（M4 设计 3.4；列表、子资源在各自的 Phase） |
````
````new docs/v0/plane-diff.md
| 访客只看自己的工作项 | 只在约 6 处执行 | 项目没开 `guest_view_all_features` 时，它的访客（按项目角色算，工作区管理员在项目里是访客时同样）只看得到自己创建的工作项：取、按编号取、修改、删除、归档、恢复别人的都答 404 `issue.not_found`；子工作项的列表和 `sub_issues_count`、关联的列表只含他看得到的工作项，回答里的 `parent_id` 指名他看不到的工作项时为 `null`（M4/P3）；读在判定之后执行这条规则，写在不加锁的预检和锁下的判定中各执行一次（M4 设计 3.4；其余的列表、子资源在各自的 Phase） |
````
````old docs/v0/plane-diff.md
| 已归档的工作项 | 只挡修改；评论、链接、表情、关联不挡；能成为父工作项 | 修改 409 `issue.archived`，什么都没改的修改也是；恢复、删除照常；在已归档的项目里先答 409 `project.archived`（M4 设计 3.8；子资源的写、成为父工作项和关联的对方在 M4 的各自的 Phase） |
````
````new docs/v0/plane-diff.md
| 已归档的工作项 | 只挡修改；评论、链接、表情、关联不挡；能成为父工作项 | 修改 409 `issue.archived`，什么都没改的修改也是；恢复、删除照常；在已归档的项目里先答 409 `project.archived`；往它下面加子工作项、从它加关联、删除它的关联（任一端已归档）409 `issue.archived`；不能被改成别的工作项的父工作项、被加到别的工作项下面、成为关联的对方（422 `not_allowed`）；已归档的子工作项、对方不在子工作项和关联的列表、`sub_issues_count` 里，恢复之后重新出现（M4/P3）（M4 设计 3.8、3.9、3.10；评论、链接、表情的写在 M4 的各自的 Phase） |
````
````old docs/v0/plane-diff.md
| 按编号取工作项 | 格式不对 400，不是成员 403 | `GET /api/v0/workspaces/{slug}/issues/{identifier}`：不是工作区的有效成员 404 `workspace.not_found`；之后格式不对、项目或编号不存在、看不到、访客规则都是同一个 404 `issue.not_found`；看得到项目而不是它的有效成员的（工作区管理员；项目公开时，工作区成员）403 `forbidden`，与取工作项相同；标识经同一个折叠比较（`identifier-fold.json`，`i`、`ı`、`I`、`İ` 一样）（M4 设计 3.7） |
````
````new docs/v0/plane-diff.md
| 按编号取工作项 | 格式不对 400，不是成员 403 | `GET /api/v0/workspaces/{slug}/issues/{identifier}`：不是工作区的有效成员 404 `workspace.not_found`；之后格式不对、项目或编号不存在、看不到、访客规则都是同一个 404 `issue.not_found`；看得到项目而不是它的有效成员的（工作区管理员；项目公开时，工作区成员）403 `forbidden`，与取工作项相同；标识经同一个折叠比较（`identifier-fold.json`，`i`、`ı`、`I`、`İ` 一样）（M4 设计 3.7） |
| 删除工作项 | 动态保留，另写一条"已删除"的动态；确认框不提子工作项 | 工作项和它下面每一层的工作项（已归档的、删除者看不到的也算）连同它们的负责人、标签、关联（两个方向，另一端在别的项目里的也算）、关注、动态，同一时刻、同一个删除者软删除；不写"已删除"的动态（M4 设计 3.6、3.9、3.13；确认框写明子树在 M4 的 P14） |
| 父工作项 | 不检查自己、不检查成环 | 父工作项要是同一个项目里调用者看得到、未归档的工作项，不是它自己、不是它下面的（创建、修改 422 `parent_id`，加子工作项 422 `sub_issue_ids[i]`，`not_allowed`）；同一个项目里改父工作项的写在项目的 N 之下一个接一个，两个并发的移动不会合起来成环（M4 设计 3.9） |
| 加子工作项的动态 | `old_value` 是子工作项自己的编号；`comment` 是 "updated the parent issue to" | 是它原来的父工作项的编号（没有时为空），`old_identifier` 是原来的父工作项（M4 设计 3.13 第 5 行）；`comment` 是 "updated the parent work item to"，修改父工作项的一行同样（Nerve 说 work item，同 M4/P2 的归档、恢复两行） |
| 子工作项数 | 数全部的直接子工作项 | `sub_issues_count` 数未删除、未归档的直接子工作项中调用者看得到的：项目里"只看自己"的访客只数自己建的（M4 设计 3.9；按它排序在 M4 的 P5） |
| 关联 | 六种类型，不检查类型和自己；一对工作项可有两条；显示看不到的项目里的工作项；删除不存在的 500 | 页面的四种（`relates_to`、`duplicate`、`blocked_by`、`blocking`，阻塞反过来存为对方的 `blocked_by`），别的类型 422 `relation_type` `invalid_format`；一对工作项至多一条，不论类型和方向（422 `issue_ids[i]` `duplicate`）；对方要是同一个工作区里调用者看得到、未归档、所在项目未归档的别的工作项（422 `not_allowed`），列表只含这样的对方；按关联的 `id` 删除，能改两端任一个的即可，不存在、已删除、两端都看不到的 404 `issue.relation_not_found`（M4 设计 3.10） |
| 关联的动态 | 对方 id 在 `old_identifier`；反向行的 `project_id` 是本项目；删除的反向行用正向的类型 | 增、删各两条：正向的写在本工作项上、反向的写在对方上，各用从自己看的类型，各在自己的项目里；加的一条对方 id 在 `new_identifier`，删的一条在 `old_identifier`（M4 设计 3.13 第 20–23 行） |
````
````old docs/v0/plane-diff.md
| 删除工作区 | 清掉别人的 `last_workspace_id`；成员、显示设置等的连带软删除由 Celery 异步完成 | 不清 `last_workspace_id`（登录后的落点规则让它无害，M3 设计 3.14）；工作区、成员、邀请、显示设置，项目和它们的成员关系、成员在项目里的显示设置、状态和标签，项目的工作项和它们的负责人、标签、关注、动态（M4/P1a 起），在同一个事务里、同一时刻软删除，删除之后 slug 立即可以重用 |
````
````new docs/v0/plane-diff.md
| 删除工作区 | 清掉别人的 `last_workspace_id`；成员、显示设置等的连带软删除由 Celery 异步完成 | 不清 `last_workspace_id`（登录后的落点规则让它无害，M3 设计 3.14）；工作区、成员、邀请、显示设置，项目和它们的成员关系、成员在项目里的显示设置、状态和标签，项目的工作项和它们的负责人、标签、关注、动态（M4/P1a 起）、关联（M4/P3 起），在同一个事务里、同一时刻软删除，删除之后 slug 立即可以重用 |
````

`docs/v0/v0-design.md`（修改，5 处）：

````old docs/v0/v0-design.md
- **集合型的列表不分页**：页面要整个集合、大小由管理员的操作决定的列表（工作区、成员、邀请、项目、项目成员、状态、标签）一次返回全部，响应仍是 `{ "data": [...] }` 封套，没有 `limit`、`cursor`、`next_cursor`；每个列表写明顺序，同值时按 `id`（M3 设计 3.12）。以后要给其中一个分页，是接口的改动，调用方同时改。第一个是 M3/P1 的 `listWorkspaces`。
````
````new docs/v0/v0-design.md
- **集合型的列表不分页**：页面要整个集合、大小由管理员的操作决定的列表（工作区、成员、邀请、项目、项目成员、状态、标签）一次返回全部，响应仍是 `{ "data": [...] }` 封套，没有 `limit`、`cursor`、`next_cursor`；每个列表写明顺序，同值时按 `id`（M3 设计 3.12）。一个工作项的子工作项、关联、链接、表情同样是集合型：它们的大小由人对这一个工作项的操作决定，页面要整个集合（M4 设计 3.16）。以后要给其中一个分页，是接口的改动，调用方同时改。第一个是 M3/P1 的 `listWorkspaces`，工作项一侧的第一个是 M4/P3 的 `listSubIssues`。
````
````old docs/v0/v0-design.md
  - **例外：工作区成员内嵌成员的公开资料**。`WorkspaceMember.member` 是 `MemberUser`（`id`、显示名、名字、头像、邮箱）：v0 没有读别的用户的接口，成员列表是看到同事名字和头像的唯一来源（Plane 同样内嵌）。邮箱按调用者的角色给出，访客看到 `null`（M3 设计 5.2、11.3）。第一个是 M3/P2 的 `listWorkspaceMembers`。
````
````new docs/v0/v0-design.md
  - **例外：工作区成员内嵌成员的公开资料**。`WorkspaceMember.member` 是 `MemberUser`（`id`、显示名、名字、头像、邮箱）：v0 没有读别的用户的接口，成员列表是看到同事名字和头像的唯一来源（Plane 同样内嵌）。邮箱按调用者的角色给出，访客看到 `null`（M3 设计 5.2、11.3）。第一个是 M3/P2 的 `listWorkspaceMembers`。
  - **例外：关联内嵌对方的工作项**。`IssueRelation.issue` 是对方工作项的列表行（`Issue`）：对方可以在另一个项目里，页面没有它的数据，只给 id 就要为每个对方再取一次，理由与成员内嵌资料相同；只给调用者看得到的对方（M4 设计 3.10）。第一个是 M4/P3 的 `createIssueRelations`、`listIssueRelations`。
````
````old docs/v0/v0-design.md
- **加锁顺序**（M3 设计 3.6；M4 设计 3.5）：全局顺序是 `users` → `profiles` → `auth_sessions` → `api_tokens` → `workspaces` → `workspace_member_invites` → `workspace_members` → `workspace_user_properties` → `projects` → `project_members` → `project_user_properties` → `states` → `labels` → `draft_issues` → `issues` → `issue_assignees` → `issue_labels` → `issue_relations` → `issue_links` → `issue_subscribers` → `issue_mentions` → `issue_comments` → `issue_reactions` → `comment_reactions` → `issue_activities` → `issue_description_versions`（M4 的表随各自的 Phase 建出），同一张表的多行按 `id` 升序。另有六条约定，后续里程碑的表照做：
````
````new docs/v0/v0-design.md
- **加锁顺序**（M3 设计 3.6；M4 设计 3.5）：全局顺序是 `users` → `profiles` → `auth_sessions` → `api_tokens` → `workspaces` → `workspace_member_invites` → `workspace_members` → `workspace_user_properties` → `projects` → `project_members` → `project_user_properties` → `states` → `labels` → `draft_issues` → `issues` → `issue_assignees` → `issue_labels` → `issue_relations` → `issue_links` → `issue_subscribers` → `issue_mentions` → `issue_comments` → `issue_reactions` → `comment_reactions` → `issue_activities` → `issue_description_versions`（M4 的表随各自的 Phase 建出），同一张表的多行按 `id` 升序。另有七条约定，后续里程碑的表照做：
````
````old docs/v0/v0-design.md
  5. 一条语句批量改子行时按扫描顺序加锁（都在父行的 `FOR NO KEY UPDATE` 之下）；别的写不持父行的锁也能写到的子行（今天是工作区的邀请：停用按地址删除发给他邮箱的那些）除外，先由一条语句按 `id` 升序锁住，再只写锁住的那些（M3/P6 的裁定 G-1）；只持父行的共享锁而批量插入时，按唯一键的顺序插入。
  6. 工作区成员关系的增长最先以 `FOR SHARE` 锁住得到它的账户行（项目成员关系的增长照约定三，锁他的工作区成员行、不锁账户行），收缩先改工作区成员行、再列举他的项目成员关系；恢复以前的成员关系不比新授予给得更多，也不比原来那一行更多。
````
````new docs/v0/v0-design.md
  5. 一条语句批量改子行时按扫描顺序加锁（都在父行的 `FOR NO KEY UPDATE` 之下）；别的写不持父行的锁也能写到的子行除外，先由一条语句按 `id` 升序锁住，再只写锁住的那些：工作区的邀请（停用按地址删除发给他邮箱的那些，M3/P6 的裁定 G-1），和 M4/P3 起跨项目的关联行（删除工作项软删除它和它下面的工作项两个方向的关联，删除项目、工作区软删除另一端在它里面的关联也算，另一个项目上的写不持这个项目的锁也能写到同一行；锁关联行的一步在全局顺序里关联行的位置，M4 设计 3.5）；只持父行的共享锁而批量插入时，按唯一键的顺序插入。
  6. 工作区成员关系的增长最先以 `FOR SHARE` 锁住得到它的账户行（项目成员关系的增长照约定三，锁他的工作区成员行、不锁账户行），收缩先改工作区成员行、再列举他的项目成员关系；恢复以前的成员关系不比新授予给得更多，也不比原来那一行更多。
  7. **跨项目的写按 `id` 升序同时锁住涉及的每个项目**（M4/P3 起，M4 设计 3.5）：不加锁地读出涉及的工作项、得到它们的项目，预检之后按项目 `id` 升序一次进入它们的写闸门（见下），再由一条语句按 `id` 升序以 `FOR NO KEY UPDATE` 锁住它们，然后重读、判定、写。今天只有关联：加关联、删除关联的两个工作项可以在两个项目里，关联的写会写到对方一侧的行（对方的 `updated_at`、对方工作项上的反向动态），所以两边的项目都要 N。加关联时调用者看不到的对方在预检里就被拒绝（锁下与别的问题一并答 422），它的项目不锁、不进闸门。与删除工作区按 `id` 升序锁它的全部项目同一个顺序，M3 的项目级的写持工作区 S 和一个项目的 N、不等别的项目，所以不会成环。
````
````old docs/v0/v0-design.md
  从 M4/P1b 起，工作项一侧的写在预检之后、开始事务（取连接）之前进入它的项目的**写闸门**（M4 设计 3.5，架构问题 11）：每个进程里一个项目至多 K 个写同时在数据库里（`issue.write_gate.per_project`，默认 2），多出的在进程里排队，不持连接、不持锁，排队至多 `issue.write_gate.max_queue` 个（默认 64），每个至多等 `issue.write_gate.max_wait`（默认 2 秒）；进入数据库的写，每一次等锁至多 `issue.write_lock_timeout`（默认 5 秒，事务开始时 `SET LOCAL lock_timeout`）。排满、等不到、在闸门上请求到期和等锁到期都答 503 `server_busy`，带 `Retry-After: 1`，只在组合根（`bootstrap/gate.go`）一处换成这个码。连接池的代价：等项目行的写持着池里的一个连接，没有闸门时一个热的项目能占满整个池（`database.max_conns`，默认 10），别的项目上无关的读写排在池的等待者后面（M4 设计的 SP1：200 个写排在一个项目上时，别的项目的读等约 875 ms）；有了闸门，一个项目上的工作项的写至多占 K 个连接（M4/P1b 的交错 20：同样的突发之下别的项目的读不到 50 毫秒，没有闸门时一秒以上）；闸门限的是每个项目的份额，同时热的项目达到池的大小除以 K（默认 5 个）时，等锁的写仍能占满池。代价是热的项目上超出的写得到 503、要由客户端按 `Retry-After` 重试。多个进程时每个进程各有 K 个，池的大小与 K、同时热的项目数由 M8 的部署说明定。
````
````new docs/v0/v0-design.md
  从 M4/P1b 起，工作项一侧的写在预检之后、开始事务（取连接）之前进入它的项目的**写闸门**（M4 设计 3.5，架构问题 11）：每个进程里一个项目至多 K 个写同时在数据库里（`issue.write_gate.per_project`，默认 2），多出的在进程里排队，不持连接、不持锁，排队至多 `issue.write_gate.max_queue` 个（默认 64），每个至多等 `issue.write_gate.max_wait`（默认 2 秒）；进入数据库的写，每一次等锁至多 `issue.write_lock_timeout`（默认 5 秒，事务开始时 `SET LOCAL lock_timeout`）。排满、等不到、在闸门上请求到期和等锁到期都答 503 `server_busy`，带 `Retry-After: 1`，只在组合根（`bootstrap/gate.go`）一处换成这个码。连接池的代价：等项目行的写持着池里的一个连接，没有闸门时一个热的项目能占满整个池（`database.max_conns`，默认 10），别的项目上无关的读写排在池的等待者后面（M4 设计的 SP1：200 个写排在一个项目上时，别的项目的读等约 875 ms）；有了闸门，一个项目上的工作项的写至多占 K 个连接（M4/P1b 的交错 20：同样的突发之下别的项目的读不到 50 毫秒，没有闸门时一秒以上）；闸门限的是每个项目的份额，同时热的项目达到池的大小除以 K（默认 5 个）时，等锁的写仍能占满池。代价是热的项目上超出的写得到 503、要由客户端按 `Retry-After` 重试。多个进程时每个进程各有 K 个，池的大小与 K、同时热的项目数由 M8 的部署说明定。M4/P3 起，跨项目的写（约定七）在预检之后按项目 `id` 升序进入涉及的每个项目的闸门，一次取齐、共用一个 `max_wait`，其中一个等不到就退回已进入的全部；等后一个闸门时只占着前面闸门的位置，不持连接、不持锁。闸门不是数据库的锁，不走闸门的写不等它，所以闸门与数据库的锁之间不会成环（M4 设计 3.5、11.11）。
````

- [ ] **Step 2: 交接**

`docs/v0/M4-issue-core/handoffs/M2-closeout.md`（修改，1 处）：

````old docs/v0/M4-issue-core/handoffs/M2-closeout.md
来源：[M4/P2 spec](../specs/P2-issue-access.md) 第 7 节。

````
````new docs/v0/M4-issue-core/handoffs/M2-closeout.md
来源：[M4/P2 spec](../specs/P2-issue-access.md) 第 7 节。

## 处理结果（M4/P3）

- **第 3 节，`updated_at`**（P3 的写完成）：改父工作项的 `updateIssue`，`addSubIssues`、`createIssueRelations`、`deleteIssueRelation` 和删除子树的 `deleteIssue` 在取得全部锁之后读用例的时钟（跨项目的写在涉及的每个项目的 N 之下），把这一个时刻写进它写的每一行：移动的子工作项的 `updated_at`、`updated_by_id` 和它的一行动态；关联行的 `created_at`、`updated_at`（删除时 `deleted_at`、`updated_at`），两端工作项的 `updated_at`、`updated_by_id` 和各自的一行动态；子树里每个工作项和它们的负责人、标签、关联、关注、动态的 `deleted_at`、`updated_at`。用例的测试用固定的时钟（`clockAt`）断言交给仓储的时刻等于它，仓储的测试断言这些列等于交给它的时刻；组合的测试断言一次写的每一行是同一个时刻、是写它的人（`TestWorkItemsArePutUnderAWorkItem`、`TestWorkItemsAreRelatedAcrossProjects`、`TestARelationIsDeletedFromEitherWorkItem`、`TestDeletingAWorkItemDeletesItsSubtreeAtOneMoment`）；端到端的 I8、I9 断言一次写在工作项下的行（`expectIssueUpdated` 的 `wrote`：P3 的每次写是它的一行动态）是工作项的 `updated_at`、是写它的人的，关联行由 `expectRelations` 断言是做它的人做的、删除它的人最后写的，I15、I17 断言删除的每一行同一时刻、同一个删除者（`expectIssueDeleted`）。
- **第 4 节，map 型对象和递归的请求体 schema**：M4/P3 没有用到。P3 的请求体（`SubIssuesAdd`、`IssueRelationsCreate`，`IssueCreate`、`IssueUpdate` 加的 `parent_id`）都是封闭的对象，数组的元素是 `uuid` 的字符串，代价由 P2 加进 `TestCheckCostsAboutTheBody` 的 uuid 数组守住；`closedObject` 不改。

仍未处理，状态保持 `open`：第 3 节其余写工作项的用例（M4 的 P4、P7）和自动归档推进固定时钟的测试（M4 的 P6）；第 4 节的草稿发布（M4 的 P8）、map 型对象和递归的请求体 schema（第一个用到它们的 Phase，或 M4 的 review 写明没有用到）；第 1、2、5–13 节，按 M4 设计 13.1 的落点。

来源：[M4/P3 spec](../specs/P3-issue-hierarchy.md) 第 7 节。

````

`docs/v0/M4-issue-core/handoffs/M3-closeout.md`（修改，1 处）：

````old docs/v0/M4-issue-core/handoffs/M3-closeout.md
来源：[M4/P2 spec](../specs/P2-issue-access.md) 第 7 节。

````
````new docs/v0/M4-issue-core/handoffs/M3-closeout.md
来源：[M4/P2 spec](../specs/P2-issue-access.md) 第 7 节。

## 处理结果（M4/P3）

- **第 2 节，加关联的另一端的预检**（完成）：`createIssueRelations` 在第一把锁之前对每个另一端做不加锁的预检：读它，按它的项目判定 `issue.read`，再照访客规则；看不到、不能读的另一端在那里记为它的位置上的 `issue_ids[i]` `not_allowed`（与不存在的工作项同一个答复），与锁下查到的其余问题在一个 422 里答，它的项目不进写闸门、不上锁；源工作项的项目和其余另一端的项目按 `id` 升序进入各自的写闸门，在一条语句里锁上（`LockMany`，总体设计 4.2 约定七），锁下的判定是最终的（M4 设计 3.5、3.10）。`deleteIssueRelation` 在预检里判定两端，调用者能改任一端即可；它写两端的行，两端的项目都进闸门、都锁上。单元的 `TestAWriteAcrossProjectsTakesOnlyTheProjectsItReaches`、`TestAWriteOnEitherWorkItemTakesBothProjects` 核对闸门和锁只取这些项目；组合的 `TestARelationLocksNoProjectItDoesNotReach`（交错 21 加的一条）：另一个事务持着 Lab（私有，bob 看不到）和 Pub（公开，bob 不能读它的工作项）的项目行时，bob 指向它们的关联立即得到 422，看得到 Lab 的 alice 等它；`TestEachIssueWriteLocksItsProjectFirst` 的 `createIssueRelations`、`deleteIssueRelation` 两行核对看不到、不能写的账户不取行锁、不写。
- **第 2 节，约定五**（P3 的批量写完成）：持项目 N、不落在例外里的：`SetParents`（`addSubIssues` 把子工作项放到父工作项下，一条语句）、删除子树的语句（工作项和它们的负责人、标签、关注、动态，按 `id`、`issue_id` 属于子树，一个项目之内）、`MarkChanged`（关联两端的工作项，持两端项目的 N）；`InsertRelations` 按另一端的 `id` 的顺序插入（约定五的后半句）；`DeleteRelation` 按主键写一行。例外：关联行的 `project_id` 是 `issue_id` 一端的项目，另一个项目的写（删除另一端的工作项、删除它的项目）不持这个项目的锁也写得到这些行，所以 `DeleteIssueRelationRows`（删除子树时两个方向的关联）和 `DeleteProjectRelationRows`（删除项目、工作区时一批的关联）先按 `id` 锁住（`ORDER BY id FOR NO KEY UPDATE`），再只写锁住的那些（总体设计 4.2 约定五的延伸，M4 设计 3.5）。钉住它的测试：存储的 `TestTheRelationsOfADeletedSubtreeAreLockedInIDOrder`（别的事务持着中间的一行时，语句持着前面的、还没有取后面的），`TestTheRelationsOfADeletedSubtreeAreDeletedBothWays`、`TestARelationsBatchTakesTheRowsOfTheProjectsWorkItemsBothWays`、`TestSetParents`、`TestInsertRelations`、`TestMarkChanged`；交错 11（`TestDeletingARelatedWorkItemAndTheOtherEndsProjectSerialize`）、23（`TestRelatingAWorkItemAndDeletingTheOtherEndsProjectSerialize`）、26（`TestDeletingARelationAndItsOtherEndSerialize`：删除关联与删除它另一端的工作项）和删除工作区与加关联的两种顺序（`TestRelatingAcrossProjectsAndDeletingTheWorkspaceSerialize`），连同交错 9（两次移动；加子工作项与移动）、10、13 以 `-race -count=5` 跑过，没有 40P01。
- **第 3 节，规则表、操作名、矩阵和码**（P3 的操作完成）：P3 的五个操作不加规则、不加操作名：`listSubIssues`、`listIssueRelations` 是对工作项的 `issue.read`，`addSubIssues`、`createIssueRelations` 是对工作项的 `issue.update`，`deleteIssueRelation` 是对任一端的 `issue.update`，完整性测试不变。矩阵加 19 行（`permission_matrix_sub_issues_test.go` 7 行，`permission_matrix_relations_test.go` 12 行）：每个操作在工作项表上一行、在已归档的项目上一行；写的另加已归档的项目里已归档的工作项一行（项目的 409 在先）和已归档的工作项一行，加子工作项、加关联另加另一个项目的工作项一行，加关联再加已归档的项目的工作项一行。新码 `issue.relation_not_found` 由 `issue` 的 HTTP 测试返回过（`apitest.Main` 两个方向），带前端文案。
- **第 3 节，工作项列表的访客规则**（P3 的列表和数目完成）：`listSubIssues` 只列调用者照访客规则看得到的子工作项；每个答复的 `sub_issues_count` 照调用者的规则数（只看自己的访客只数他自己建的）；`listIssueRelations` 只列另一端调用者看得到的关联（另一端的项目逐个判定 `issue.read`，再照访客规则）。看不到的工作项不在列表、数目和关联内嵌的 `Issue` 里，回答里的 `parent_id` 指名看不到的工作项时为 `null`（每个回答都经同一处 `Locks.seen`）；作为加子工作项、加关联的目标，它的答复与不存在的工作项的逐字节相同，看不到的关联的 404 与不存在的关联的逐字节相同。组合的测试：`TestTheGuestRuleHoldsForSubItems`、`TestEachAnswerCountsTheSubItemsTheCallerSees`、`TestAHiddenParentIsNotInTheAnswer`、`TestARelationListShowsWhatTheCallerSees`、`TestTheGuestRuleHoldsForRelations`、`TestTheGuestRuleHoldsForDeletingARelation`、`TestWhoMayNotDeleteARelation`。
- **第 4 节，表和删除**（P3 的 `issue_relations` 完成）：新表带 `workspace_id`、`project_id`（`issue_id` 一端的项目），进入删除工作项、删除项目、删除工作区的连带，同一时刻、同一个删除者；按目录的删除测试给它准备一行（`seedIssues` 的关联），端到端的 `furnishIssueRows` 带着一个关联。
- **第 4 节，`expectIssueDeleted` 的豁免**（完成）：`e2e/fixtures/assert/issue.ts` 不再豁免子工作项的外键：沿外键图走到的每个外键下都要有与工作项同一时刻、同一个删除者删除的一行，`issues.parent_id` 和关联的两个外键在内。I15 删除已归档的工作项时它带着一个子工作项和两个方向的关联，I17 删除一个带子树和跨项目关联的工作项；组合的 `TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt` 同样去掉了子工作项的豁免。

仍未处理，状态保持 `open`：第 2 节其余各项：P3 之后各 Phase 的批量写按约定五核对（到 M4 的 P9）、物理删除账户（M4 的 P9）；第 3 节其余各项：P3 之后各 Phase 的操作在规则表、操作名、矩阵中的行（每个后端的 Phase，到 M4 的 P9）、决策点 3 的个人主页一行（M4 的 P5）和其余工作项列表的访客规则（各列表的 Phase）；第 4 节的 `archive_in`（M4 的 P6）的说明和删除断言的限时轮询（M4 的 P9）；第 1、5–9 节，按 M4 设计 13.1 的落点。

来源：[M4/P3 spec](../specs/P3-issue-hierarchy.md) 第 7 节。

````

- [ ] **Step 3: 检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 `keywords: 72 rules, 23 exceptions, no hits.`

Run: `python3 $M4TMP/p3tools/doclinks.py . docs/v0/v0-design.md docs/v0/plane-diff.md docs/v0/M4-issue-core/handoffs/M2-closeout.md docs/v0/M4-issue-core/handoffs/M3-closeout.md docs/v0/M4-issue-core/specs/P3-issue-hierarchy.md docs/v0/M4-issue-core/plans/P3-issue-hierarchy.md`
Expected: 每个文件 `0 broken`，最后一行 `broken: 0`。

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`（文档不影响它；本 Phase 最后一次全量运行）。

- [ ] **Step 4: 提交**

```bash
git add docs/v0/M4-issue-core/handoffs/M2-closeout.md docs/v0/M4-issue-core/handoffs/M3-closeout.md docs/v0/plane-diff.md docs/v0/v0-design.md
```
```bash
git commit -m "docs(M4/P3): the design documents and the handoffs record P3

The overall design lists a work item's sub-items and relations among the
collections that come whole, embeds the other work item in a relation as
the second exception, and states convention 7 and the relation rows'
exception to convention 5. The Plane diff holds P3's rows, and the two
handoffs' outcomes say what P3 did.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**：本 Task 只改文档，没有变异（文档的相对链接由 `doclinks.py` 核对）。
