# M3/P6 停用账户与成员关系 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 停用账户（`deactivateMe` 和 `nerve users deactivate` 两条路）在同一个事务里结束他的全部成员关系、删除发给他邮箱的全部邀请（待接受的和已忽略的）；他是某个工作区或项目唯一的有效管理员、那里还有别的有效成员时拒绝（409 `workspace.sole_admin`、`project.sole_admin`），每张表不变。停用在 `identity` 已持有的账户行之下：按 id 锁住他有效成员关系所在的全部工作区（上锁时已删除的跳过），查规则 2，在最后一把工作区锁之后读一次时钟，删除邀请，结束工作区成员关系，再调用一次 `EndMemberships`（跨这些工作区按项目 id 锁住、查规则 2 的项目一侧、结束）。邮箱取自锁下的账户行；`project.NewCascade` 随 `nerve users deactivate` 加入；约定六的每一条增长路径（交错 7、8、13–16、19）两种顺序都与停用串行；故事 W9 的接口版本通过。

**Architecture:** `identity`：端口 `MembershipDeactivator`（`app/ports.go`），`LockedAccount.Email`，`deactivate` 最后调用它；`deactivateMe` 声明两个码。`workspace`：存储的四条语句（`queries/deactivation.sql`）和端口 `AllMembershipsEnder`，用例 `app.Deactivator`，导出的 `workspace.Deactivator`、`NewDeactivator`、`Module.Deactivator()`；两个模块的 `sole_admin` 说明改为对每个调用者都成立。`project`：`NewCascade`、`CascadeDeps`（`New` 经它建）。`bootstrap`：服务把 `ws.Deactivator()` 交给 `identity.New`，`nerve users` 把 `workspace.NewDeactivator` 和 `project.NewCascade` 交给 `identity.NewAdmin`；组合测试、竞争、交错。不加迁移、表、Go 模块、npm 包；跨模块的端口只有 3.9 的这一个（签名见 spec 第 3 节第 1 条）。

**Tech Stack:** Go 1.27.1、pgx v5.11.0、sqlc v1.31.1（`CGO_ENABLED=0`）、oapi-codegen v2.8.0、goose v3.28.0、River v0.47.0、golangci-lint 2.13.2、PostgreSQL 18.6（testcontainers）；Node 24、pnpm 11.10.0、Playwright 1.63.0。

**Spec:** `docs/v0/M3-workspace-project/specs/P6-deactivation.md`（上级：`docs/v0/M3-workspace-project/M3-design.md`）

## Global Constraints

- **Go 版本和依赖**：本 plan 不执行 `go get`，不加任何 Go 模块或 npm 包。`server/go.mod` 和 `server/tools/go.mod` 保持 `go 1.27` 和 `toolchain go1.27.1`（golangci-lint 2.13.2 由 go1.27.0 构建，`go` 行更高就拒绝运行）。每个 Task 提交前执行 `grep -n "^go \|^toolchain" server/go.mod server/tools/go.mod`，四行必须是 `go 1.27` 和 `toolchain go1.27.1`；`git diff --stat 6dcc0794 -- server/go.mod server/go.sum server/tools pnpm-lock.yaml` 在本 plan 的任何时刻都没有输出。
- **每个 Task 提交前**：`make lint-go` 两段都输出 `0 issues.`，`make test` 全部通过（没有 `FAIL`）。
- **生成的文件不手写、不从本 plan 复制**：有生成物的 Task（1、4）执行 Task 中的生成命令（只动了 sqlc 的 Task 1 执行 `make gen-go`，改了接口描述和 sqlc 的 Task 4 执行 `make gen`），用 `shasum -a 256` 和 `wc -l` 核对表中的 SHA-256 和行数，对不上时停下来：说明某个输入与本 plan 不一致。这些 Task **提交之后**执行 `make gen-check`（它用 `git status` 判断生成物是否已提交，提交之前必然报差异）。
- **前端检查**：改了接口描述的 Task 4 和改了 `e2e/` 的 Task 9 另执行 `make lint-web`、`make knip`、`make test-web`；P6 不声明新的错误码（`workspace.sole_admin`、`project.sole_admin` 已在 `PROBLEM_MESSAGES` 和两份 `auth.json` 里，它们的页面文案不改，spec 第 3 节第 17 条），所以 M3 设计 12 节约束 4 没有要加的文案；Task 9 执行 `make e2e`。
- **容器**：`make test` 和 `make e2e` 用自己的 testcontainers；机器忙时偶尔起不来，等 Docker 空闲之后重跑一次再当作失败。容器测试一次只跑一套。开发库 `nerve-dev-db-1` 可以用，但不要停止或重建它，不要执行 `make dev-db-down`、`make dev-db-reset`。不要碰其他项目的容器（`agentforge-*`、`plane-app-*`、`opennerve-*`、`nervewiki-*`）。
- **git**：每次 Bash 调用只执行一个 git 命令，不用 `;`、`&&`、`|` 串联 git；不用 `git -C`、`stash`、`clean`、`reset --hard`。`cd` 不与别的命令组合，只读的命令也不行。不碰 `plane/`、`refer/`。
- **安装**：除了 Docker、Go、Node 不做任何全局安装；不执行 `corepack enable`（pnpm 已在 PATH 上）。
- **规则**：不写 `init()`，不用全局可变状态，构造函数显式传入依赖；不建 `utils`、`common`、`helpers` 包；一个文件只做一件事，不超过约 400 行；依赖只能向内；模块之间不互相导入，不跨模块的表 JOIN；模块的 SQL 只经 sqlc；角色只按集合判断（`role = 20` 只出现在"管理员"这一个集合的查询里），不按大小比较；不留没有使用者的代码。**例外**：接口描述按模块一个文件（M0-P3 交接 5），`api/modules/identity.yaml`（657 行）不受约 400 行的限制。本 plan 的其余文件都在约 400 行以内（最终原型上量的）：最长的是 `bootstrap/interleaving_growth_test.go`（398 行）、`identity/app/ports.go`（379 行）、`bootstrap/reactivation_races_test.go`（363 行）、`bootstrap/interleaving_answers_test.go`（361 行）、`workspace/adapter/postgres/deactivation_test.go`（341 行）、`workspace/app/ports.go`（317 行）、`project/adapter/postgres/end_test.go`（316 行）、`bootstrap/app.go`（303 行）；`bootstrap/interleaving_growth_test.go` 在 P5b 结束时是 404 行，本 plan 把它的一段改角色提成 `changeRole`、放到交错 7 的文件里。
- **注释**：Go、TS 代码、SQL 查询和接口描述用英文；中文文档照本 plan 原样。
- **代码块**：每个改动都写成四个反引号围起来的块，块的第一行写明种类和路径，照原样使用（原型中逐字节运行过）：
  - ````` ````file <路径> ````` 新文件，块的内容加一个结尾换行就是整个文件；
  - ````` ````whole <路径> ````` 已有文件的完整新内容（同样加结尾换行）；
  - ````` ````old <路径> ````` 与紧跟着的 ````` ````new <路径> `````：`old` 的文字在文件当前版本中恰好出现一次，把它换成 `new` 的文字；
  - ````` ````delete <路径> ````` 删除这个文件（块是空的）。

  一个文件的几个块按出现的顺序依次应用。拼 plan 的脚本已从 `6dcc0794` 起按顺序核对过全部块：每个 `old` 恰好出现一次（在它之前的块应用之后的文件中），每个新文件原来不存在，逐 Task 应用之后的文件与原型逐字节相同（spec 附录 A）。可以用 `node <planapply.mjs> <本 plan> apply <仓库根> <n>` 写入第 n 个 Task 的块，也可以手工照抄。
- **过渡版本**：一些文件先在较早的 Task 写成过渡版本，较晚的 Task 再修改（`workspace/app/ports.go`、`bootstrap/users.go`、`bootstrap/interleaving_answers_test.go`、`bootstrap/interleaving_growth_test.go`、`bootstrap/reactivation_races_test.go`）。每个过渡版本都在逐 Task 复现中运行过。
- **变异**：每个 Task 末尾的"变异"表列出：把代码改坏的方式、必须因此失败的测试和它所在的层（单元：假实现；存储：真实数据库；组合：`bootstrap` 组合出的 app、命令或模块，`cmd/nerve`；端到端：单独运行的故事 W9、A12）。它们在最终的原型上逐个跑过（`$M3TMP/p6tools/mutants_p6.py`，由 `mutlevels.py` 在它写的每一层各跑一次；清扫之后加强的测试由 `mutants_p6_rerun.py` 在受影响的层重跑；`archtest` 从磁盘读源文件，它的两个由 `archmut.py` 写进副本再跑；spec 附录 A：119 个，119 个被发现）；实现者可以照表抽查，改坏之后必须恢复。表中"（Task n 起）"标出的测试在较晚的 Task 才有，或才改成最终的样子（P6 改写的已有测试，如交错 8、19）：这一行的变异最迟从那个 Task 起被它发现。**安全或加锁的性质只由单元一层发现的，算缺口**（brief 的缺陷类别）；表中每一条这类性质都另有存储、组合或端到端一层的测试，例外写在 spec 第 3 节。
- **评审敏感**（M3 设计 12 节约束 3）：停用跨越他所在的每一个工作区、有接口和命令两条路、改动 `identity`、与约定六的每一条增长路径串行。停用是安全操作：停用之后他没有有效的成员关系、没有发给他邮箱的邀请，回来只经 `nerve users activate` 加 `reactivate-member`。改动这些测试、锁、规则之前，先照"变异"表确认它在所说的性质去掉之后失败。
- **提交**：提交信息用英文，末尾加一行：`Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`
- 所有命令在仓库根目录下执行，除非步骤中另有说明。

## 文件结构

路径相对于仓库根目录。"生成"表示由命令生成并提交；一个文件由多个 Task 修改时，"Task"一列都列出。

| 文件 | 职责 | Task |
|---|---|---|
| `server/internal/modules/workspace/adapter/postgres/queries/deactivation.sql` | `LockMemberWorkspaces`、`SoleAdmin`、`DeleteInvitationsTo`、`EndWorkspaceMemberships` | 1 |
| `server/internal/modules/workspace/adapter/postgres/gen/deactivation.sql.go`（生成） | | 1 |
| `server/internal/modules/workspace/adapter/postgres/deactivation.go`、`server/internal/modules/workspace/adapter/postgres/deactivation_test.go`；`server/internal/modules/workspace/adapter/postgres/failures_test.go`（修改） | 四个存储方法和它们的测试（锁和它的强度、id 顺序、等锁时删除的工作区；规则 2 的十六个情形；别的地址、别人的行每一列不动）；每个方法的失败原样返回 | 1 |
| `server/internal/modules/workspace/app/ports.go`（修改） | 端口 `AllMembershipsEnder`；`ProjectCascade.EndMemberships` 的说明（Task 3） | 1、3 |
| `server/internal/modules/project/module.go`（修改） | `CascadeDeps`、`NewCascade`；`New` 经它建出 `Cascade` | 2 |
| `server/internal/modules/project/adapter/postgres/end_test.go`（修改） | `TestLockActiveMemberProjectsLocksInIDOrder` 跨两个工作区 | 2 |
| `server/internal/bootstrap/interleaving_deletion_test.go`、`server/internal/bootstrap/interleaving_endings_test.go`、`server/internal/bootstrap/interleaving_leaving_test.go`、`server/internal/bootstrap/interleaving_removal_test.go`、`server/internal/bootstrap/interleaving_roles_test.go`（修改） | 只给 `Pool` 的 `project.New(…).Cascade()` 换成 `project.NewCascade` | 2 |
| `server/internal/bootstrap/interleaving_growth_test.go`（修改） | 同上；`demote` 经 `changeRole`（Task 7） | 2、7 |
| `server/internal/bootstrap/interleaving_answers_test.go`（修改） | 同上；交错 19 换成真实的停用（Task 4）；`setBobsEmail`（Task 6）；`newAnswerRace` 的角色、`deactivateBob` 的会话（Task 7） | 2、4、6、7 |
| `server/internal/modules/workspace/app/deactivate_memberships.go`、`server/internal/modules/workspace/app/deactivate_memberships_test.go`、`server/internal/modules/workspace/app/fakes_deactivation_test.go`；`server/internal/modules/workspace/app/fakes_workspaces_test.go`、`server/internal/modules/workspace/app/clock_test.go`（修改） | `app.Deactivator`；假实现；时钟在锁之后读的一行 | 3 |
| `server/internal/modules/workspace/domain/errors.go`、`server/internal/modules/project/domain/errors.go`（修改） | 两个 `sole_admin` 的说明对每个调用者都成立 | 3 |
| `server/internal/modules/workspace/adapter/http/members_test.go`、`server/internal/modules/project/adapter/http/member_writes_test.go`、`server/internal/modules/workspace/app/remove_member_test.go`（修改） | 照新的说明 | 3 |
| `server/internal/modules/identity/app/ports.go`、`server/internal/modules/identity/app/deactivate.go`、`server/internal/modules/identity/app/deactivate_test.go`（修改） | `LockedAccount.Email`、`MembershipDeactivator`；`deactivate` 最后调用它 | 4 |
| `server/internal/modules/identity/adapter/postgres/queries/users.sql`、`server/internal/modules/identity/adapter/postgres/users.go`、`server/internal/modules/identity/adapter/postgres/credentials_test.go`（修改） | 锁下读邮箱 | 4 |
| `server/internal/modules/identity/adapter/postgres/gen/users.sql.go`（生成） | | 4 |
| `server/internal/modules/identity/adapter/http/me_test.go`（修改） | `deactivateMe` 透传两个 409（9.4） | 4 |
| `server/internal/modules/identity/module.go`、`server/internal/modules/identity/admin.go`（修改） | `Deps.Memberships`、`AdminDeps.Memberships` | 4 |
| `api/modules/identity.yaml`（修改） | `deactivateMe` 的描述和两个码 | 4 |
| `api/dist/openapi.yaml`、`web/packages/api-client/src/schema.gen.ts`（生成） | | 4 |
| `server/internal/modules/workspace/deactivator.go`；`server/internal/modules/workspace/module.go`（修改） | `workspace.Deactivator`、`DeactivatorDeps`、`NewDeactivator`；`Module.Deactivator()` | 4 |
| `server/internal/bootstrap/app.go`（修改） | 服务：`identity.New` 的 `Memberships` | 4 |
| `server/internal/bootstrap/users.go`（修改） | `nerve users`：`identity.NewAdmin` 的 `Memberships`；停用的一行（Task 5） | 4、5 |
| `server/internal/archtest/composition_test.go`（修改） | 每个命令组合建什么、不建什么 | 4 |
| `server/internal/bootstrap/deactivating_test.go` | `deactivating`（`nerve users deactivate` 照 `users.go` 的接法）、`endedHoldingAll` | 4 |
| `server/internal/bootstrap/interleaving_test.go`（修改） | 交错 8 换成真实的停用 | 4 |
| `server/internal/bootstrap/deactivation_world_test.go`、`server/internal/bootstrap/deactivation_test.go` | `deactivationWorld`、两条路；拒绝每张表不变、提交被拒每张表不变、成功的每一行 | 5 |
| `server/internal/bootstrap/users_test.go`、`server/internal/bootstrap/reactivation_races_test.go`（修改） | 停用的一行；`commandInBackground`；交错 16（Task 7） | 5、7（`reactivation_races_test.go`） |
| `server/cmd/nerve/users.go`、`server/cmd/nerve/users_test.go`（修改） | 两个命令的说明；拒绝时打印说明、退出码 1 | 5 |
| `server/internal/bootstrap/deactivation_locks_test.go`、`server/internal/bootstrap/deactivation_races_test.go` | 每把锁的顺序和强度、事务的连接、命令的中断；等锁期间的删除和移出、锁下的邮箱 | 6 |
| `server/internal/bootstrap/interleaving_accepting_test.go`；`server/internal/bootstrap/interleaving_invite_test.go`（修改） | 交错 7；`changeRole`（改角色，交错 7 (b) 和 P4b 的降为访客共用） | 7 |
| `server/internal/bootstrap/interleaving_shrinking_test.go` | 交错 13、14、15；两位管理员同时停用 | 8 |
| `e2e/stories/workspace/w9-deactivation.spec.ts`；`e2e/stories/identity/a12-deactivate.spec.ts`（修改） | 故事 W9 的接口版本；A12 的命令一行 | 9 |
| `README.md`、`docs/v0/plane-diff.md`、`docs/v0/v0-design.md`、`docs/v0/M3-workspace-project/handoffs/M2-closeout.md`（修改） | 8.7、3.20 的 P6 行；总体设计 4.2 的约定六停用一段；M2 收尾交接第 6 节的处理结果 | 9 |

---

### Task 1: 停用的四条语句：锁他的工作区、工作区的唯一管理员、按邮箱删除邀请、结束工作区成员关系

**Files:**
- Create: `server/internal/modules/workspace/adapter/postgres/deactivation.go`、`server/internal/modules/workspace/adapter/postgres/deactivation_test.go`、`server/internal/modules/workspace/adapter/postgres/queries/deactivation.sql`
- Modify: `server/internal/modules/workspace/adapter/postgres/failures_test.go`、`server/internal/modules/workspace/app/ports.go`
- Generate: `server/internal/modules/workspace/adapter/postgres/gen/deactivation.sql.go`

**Interfaces:**
- Produces（spec 2.3，M3 设计 3.6 约定六、3.7 规则 2、3.8、3.9）：端口 `workspace/app.AllMembershipsEnder`（`ports.go`），`Deactivator`（Task 3）的存储；`workspace/adapter/postgres.Store` 实现它的四个方法，都在 `ctx` 带着的事务里执行（停用的事务：`identity` 开的，已持有账户行的 `FOR NO KEY UPDATE`）：
  - `LockMemberWorkspaces(ctx, userID) ([]uuid.UUID, error)`：他是有效、未删除成员的未删除工作区，按 id 升序 `FOR NO KEY UPDATE` 锁住，按这个顺序回答；等锁时被删除的少返回一行，不是错误（复核 spike 15）。
  - `SoleAdmin(ctx, workspaceIDs, userID) (bool, error)`：他是不是其中某个"还有别的有效成员"的工作区唯一的有效管理员（`role = 20`，"管理员"这个集合）。只有他一人的、另有有效管理员的不算。
  - `DeleteInvitationsTo(ctx, email, by, now) error`：发给这个地址的每一份未删除的邀请，每个工作区、待接受的和已忽略的，`deleted_at = updated_at = now`、`updated_by_id = by`。
  - `EndWorkspaceMemberships(ctx, workspaceIDs, userID, by, now) error`：他在这些工作区有效、未删除的成员关系 `is_active = false`，`updated_at = now`、`updated_by_id = by`，行和角色留着；已结束的不再写。
- 四条查询照原样写进 `queries/deactivation.sql`（spec 2.3 有全文）。

**Tests:**（`adapter/postgres/deactivation_test.go`；`tableRows`、`stamp` 比较被写的行之外的每一行每一列）
- `TestLockMemberWorkspaces`：bob 在 acme、gamma 有效：回答这两个（按 id），`FOR SHARE` 等它们、外键的 `FOR KEY SHARE` 不等；beta（他的成员关系已结束）、delta（已删除）、gone（工作区已删除，他的成员关系未删除）、epsilon（只有 carol）、zeta（他不是成员）都不锁。
- `TestLockMemberWorkspacesLocksInIDOrder`：web 的 id 较小、在表和 slug 的索引里排在 alpha 之后；alpha 被持有时，`LockMemberWorkspaces` 等它，已持有 web（`FOR SHARE` web 等）：按别的顺序会先碰到 alpha、什么都不持有。
- `TestLockMemberWorkspacesLeavesOutAWorkspaceDeletedWhileItWaited`：另一个事务持 acme 的 `FOR NO KEY UPDATE`、软删除它和它的成员关系；`LockMemberWorkspaces` 等它，提交之后回答只有 beta。
- `TestSoleAdminOfAWorkspace`：十六个情形（只有他一人、有有效成员或访客、另一位已结束、已删除、两位有效管理员、他是成员、他已结束、已删除、一次问两个工作区的三种、不问任何工作区）；每个工作区只有情形说的成员（创建者的成员关系删除），各情形的工作区并排存在：问到不该问的工作区、把整个集合当一个工作区的实现答错（清扫 20）。
- `TestDeleteInvitationsTo`：bob 的三份（acme 待接受、beta 已忽略、gamma 他不是成员的）由 bob、在给的时刻删除；以前已接受、已删除的两份保留时刻；carol 的、`rebob@corp.com` 的每一列不动。
- `TestEndWorkspaceMemberships`：acme（成员）、gamma（管理员）的结束，由 bob（这两行之前由 alice 写）、在给的时刻，角色不变；beta 已由 alice 结束的保留结束者和时刻；他已删除的那一行、carol 的、delta（不问的）每一列不动。
- `failures_test.go`：`TestAFailedReadIsAnErrorNotAnAnswer` 加 `LockMemberWorkspaces`（不是"没有工作区"）、`SoleAdmin`（不是"不是唯一的管理员"）；`TestAFailedWriteIsAnError` 加 `DeleteInvitationsTo`、`EndWorkspaceMemberships`（清扫 19）。

- [ ] **Step 1: 查询**

`server/internal/modules/workspace/adapter/postgres/queries/deactivation.sql`（新文件，45 行）：

````file server/internal/modules/workspace/adapter/postgres/queries/deactivation.sql
-- name: LockMemberWorkspaces :many
-- The first step of ending a deactivated account's memberships (M3 design 3.6 convention 6, 3.9), under the account
-- row's FOR NO KEY UPDATE, which the deactivation took first: the undeleted workspaces of which he is an active member,
-- found now, locked FOR NO KEY UPDATE in id order. The lock is taken as the sorted rows come, so the order is the ids'.
-- After a wait, Postgres evaluates deleted_at IS NULL again on the row's newest version: a workspace deleted meanwhile is
-- left out, not an error, its memberships ended with it (review spike 15).
SELECT w.id
FROM workspaces w
WHERE w.deleted_at IS NULL
  AND EXISTS (SELECT 1 FROM workspace_members m
              WHERE m.workspace_id = w.id AND m.member_id = sqlc.arg(member_id) AND m.is_active AND m.deleted_at IS NULL)
ORDER BY w.id
FOR NO KEY UPDATE;

-- name: SoleAdmin :one
-- The second step, under the workspaces' locks: whether the account is the only active admin of one of the workspaces
-- that has another active member, whom ending his membership would leave without an admin (M3 design 3.7 rule 2). A
-- workspace where he is alone, or that has another active admin, does not count.
SELECT EXISTS (
    SELECT 1 FROM workspace_members m
    WHERE m.workspace_id = ANY (sqlc.arg(workspace_ids)::uuid[]) AND m.member_id = sqlc.arg(member_id) AND m.role = 20
      AND m.is_active AND m.deleted_at IS NULL
      AND NOT EXISTS (SELECT 1 FROM workspace_members a
                      WHERE a.workspace_id = m.workspace_id AND a.member_id <> m.member_id AND a.role = 20 AND a.is_active
                        AND a.deleted_at IS NULL)
      AND EXISTS (SELECT 1 FROM workspace_members o
                  WHERE o.workspace_id = m.workspace_id AND o.member_id <> m.member_id AND o.is_active
                    AND o.deleted_at IS NULL));

-- name: DeleteInvitationsTo :exec
-- The third step (M3 design 3.8, 3.9; Plane views/user/base.py:313): every undeleted invitation to the address, of
-- every workspace, pending or declined, soft-deleted at the moment and by the account given, before the memberships'
-- rows (the global order). One statement in scan order; a deleted one keeps its moment.
UPDATE workspace_member_invites
SET deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE email = sqlc.arg(email) AND deleted_at IS NULL;

-- name: EndWorkspaceMemberships :exec
-- The fourth step, one statement under the workspaces' locks (convention 5): the account's active memberships of the
-- workspaces end, at the moment and by the account given; the rows stay, each with its role, and an ended or deleted one
-- keeps its columns.
UPDATE workspace_members
SET is_active = false, updated_at = sqlc.arg(now)::timestamptz, updated_by_id = sqlc.arg(ended_by)::uuid
WHERE workspace_id = ANY (sqlc.arg(workspace_ids)::uuid[]) AND member_id = sqlc.arg(member_id) AND is_active
  AND deleted_at IS NULL;
````

Run: `make gen-go`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `233f91644120c604360fd4a2ce7c8422e42f772a17694a55da7d05dc805143ee` | 123 | `server/internal/modules/workspace/adapter/postgres/gen/deactivation.sql.go` |

Run: `shasum -a 256 server/internal/modules/workspace/adapter/postgres/gen/deactivation.sql.go`
Expected: 与上表相同。

- [ ] **Step 2: 端口、存储和测试**

`server/internal/modules/workspace/app/ports.go`（修改，1 处）：

````old server/internal/modules/workspace/app/ports.go
}

// ProjectCascade is what the workspace's writes ask of the projects (M3
````
````new server/internal/modules/workspace/app/ports.go
}

// AllMembershipsEnder ends every membership of a deactivated account and
// deletes every invitation to its address (M3 design 3.6 convention 6,
// 3.9): the Deactivator's repository. It runs in the transaction ctx
// carries, which identity's deactivation began and holds the account row's
// FOR NO KEY UPDATE in.
type AllMembershipsEnder interface {
	// LockMemberWorkspaces locks the undeleted workspaces of which userID is
	// an active member FOR NO KEY UPDATE in id order, until the transaction
	// ends, and returns their ids in that order. One deleted while the lock
	// waited is left out.
	LockMemberWorkspaces(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	// SoleAdmin reports whether userID is the only active admin of one of
	// the workspaces that has another active member (M3 design 3.7 rule 2).
	SoleAdmin(ctx context.Context, workspaceIDs []uuid.UUID, userID uuid.UUID) (bool, error)
	// DeleteInvitationsTo soft-deletes every undeleted invitation to email,
	// pending or declined, of every workspace, by the account by at now.
	DeleteInvitationsTo(ctx context.Context, email string, by uuid.UUID, now time.Time) error
	// EndWorkspaceMemberships ends userID's active, undeleted memberships
	// of the workspaces, by the account by at now; the rows stay.
	EndWorkspaceMemberships(ctx context.Context, workspaceIDs []uuid.UUID, userID, by uuid.UUID, now time.Time) error
}

// ProjectCascade is what the workspace's writes ask of the projects (M3
````

`server/internal/modules/workspace/adapter/postgres/deactivation.go`（新文件，60 行）：

````file server/internal/modules/workspace/adapter/postgres/deactivation.go
package postgresadapter

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres/gen"
)

// The statements of a deactivated account's ending (M3 design 3.6
// convention 6, 3.9): app.AllMembershipsEnder, in the transaction ctx
// carries, which identity's deactivation began and holds the account row's
// FOR NO KEY UPDATE in.

// LockMemberWorkspaces locks the undeleted workspaces of which userID is an
// active member FOR NO KEY UPDATE in id order, until the transaction ends,
// and returns their ids in that order. One deleted while the lock waited is
// left out.
func (s *Store) LockMemberWorkspaces(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).LockMemberWorkspaces(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("lock the member's workspaces: %w", err)
	}
	return ids, nil
}

// SoleAdmin reports whether userID is the only active admin of one of the
// workspaces that has another active member.
func (s *Store) SoleAdmin(ctx context.Context, workspaceIDs []uuid.UUID, userID uuid.UUID) (bool, error) {
	sole, err := s.queries(ctx).SoleAdmin(ctx, gen.SoleAdminParams{WorkspaceIds: workspaceIDs, MemberID: userID})
	if err != nil {
		return false, fmt.Errorf("look for a workspace he is the only admin of: %w", err)
	}
	return sole, nil
}

// DeleteInvitationsTo soft-deletes every undeleted invitation to email,
// pending or declined, of every workspace, by the account by at now.
func (s *Store) DeleteInvitationsTo(ctx context.Context, email string, by uuid.UUID, now time.Time) error {
	if err := s.queries(ctx).DeleteInvitationsTo(ctx, gen.DeleteInvitationsToParams{Email: email, DeletedBy: by, Now: now}); err != nil {
		return fmt.Errorf("delete the invitations to the address: %w", err)
	}
	return nil
}

// EndWorkspaceMemberships ends userID's active, undeleted memberships of
// the workspaces, by the account by at now; the rows stay. A workspace of
// workspaceIDs where he has none, his membership of it ended while its
// lock waited, is no error.
func (s *Store) EndWorkspaceMemberships(ctx context.Context, workspaceIDs []uuid.UUID, userID, by uuid.UUID, now time.Time) error {
	err := s.queries(ctx).EndWorkspaceMemberships(ctx, gen.EndWorkspaceMembershipsParams{
		WorkspaceIds: workspaceIDs, MemberID: userID, EndedBy: by, Now: now,
	})
	if err != nil {
		return fmt.Errorf("end the workspace memberships: %w", err)
	}
	return nil
}
````

`server/internal/modules/workspace/adapter/postgres/deactivation_test.go`（新文件，341 行）：

````file server/internal/modules/workspace/adapter/postgres/deactivation_test.go
package postgresadapter_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// waitsFor reports whether the lock mode of the workspace id, taken in a
// transaction of its own, waits for another transaction's lock: it ends
// with lock_not_available under withLockTimeout's 300ms. A lock that does
// not wait must find the row.
func waitsFor(t *testing.T, pool *pgxpool.Pool, id uuid.UUID, mode string) bool {
	t.Helper()
	var locked int64
	err := withLockTimeout(postgres.NewTxManager(pool, 2*time.Second), pool, func(ctx context.Context) error {
		tag, err := postgres.DB(ctx, pool).Exec(ctx, "SELECT id FROM workspaces WHERE id = $1 "+mode, id)
		locked = tag.RowsAffected()
		return err
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "55P03" {
		return true
	}
	if err != nil || locked != 1 {
		t.Fatalf("%s of %s: %d rows, %v; want its row", mode, id, locked, err)
	}
	return false
}

// LockMemberWorkspaces returns, in id order, the undeleted workspaces of
// which bob is an active member, acme and gamma, and holds each FOR NO KEY
// UPDATE until its transaction ends: a FOR SHARE of it waits, a foreign
// key's FOR KEY SHARE does not. It holds no other: not beta, where his
// membership ended; not delta, where it is deleted; not gone, deleted,
// though his membership of it is not; not epsilon, carol's alone; not
// zeta, where he is no member.
func TestLockMemberWorkspaces(t *testing.T) {
	s, pool := newStore(t)
	alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	ws := map[string]uuid.UUID{}
	for _, name := range []string{"acme", "beta", "gamma", "delta", "gone", "epsilon", "zeta"} {
		ws[name] = newWorkspace(t, s, name, name, alice).ID
	}
	for _, name := range []string{"acme", "beta", "gamma", "delta", "gone"} {
		join(t, s, ws[name], bob, shared.RoleMember)
	}
	join(t, s, ws["epsilon"], carol, shared.RoleAdmin)
	exec(t, pool, "UPDATE workspace_members SET is_active = false WHERE workspace_id = $1 AND member_id = $2", ws["beta"], bob)
	exec(t, pool, "UPDATE workspace_members SET deleted_at = $3 WHERE workspace_id = $1 AND member_id = $2", ws["delta"], bob, now)
	exec(t, pool, "UPDATE workspaces SET deleted_at = $2 WHERE id = $1", ws["gone"], now)
	var locked []uuid.UUID
	end := hold(t, postgres.NewTxManager(pool, 2*time.Second), func(ctx context.Context) error {
		var err error
		locked, err = s.LockMemberWorkspaces(ctx, bob)
		return err
	})

	if want := slices.SortedFunc(slices.Values([]uuid.UUID{ws["acme"], ws["gamma"]}), uuid.UUID.Compare); !slices.Equal(locked, want) {
		t.Errorf("LockMemberWorkspaces() = %v, want acme and gamma, %v", locked, want)
	}
	for name, id := range ws {
		held := name == "acme" || name == "gamma"
		if share, keyShare := waitsFor(t, pool, id, "FOR SHARE"), waitsFor(t, pool, id, "FOR KEY SHARE"); share != held || keyShare {
			t.Errorf("%s while locked: a FOR SHARE waits %v, a FOR KEY SHARE waits %v; want %v, false", name, share, keyShare, held)
		}
	}
	if err := end(); err != nil {
		t.Fatal(err)
	}
}

// LockMemberWorkspaces takes its locks in the workspaces' id order,
// whatever order the rows lie in: web has the smaller id, but lies after
// alpha in the table and in the slug's index. alpha's row is held.
// LockMemberWorkspaces waits for it holding web's, which a FOR SHARE then
// waits for; in any other order it would reach alpha first and wait
// holding nothing.
func TestLockMemberWorkspacesLocksInIDOrder(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	web, alpha := uuid.NewV7(), uuid.NewV7() // web drawn first: the smaller id
	for _, w := range []struct {
		id   uuid.UUID
		slug string
	}{{alpha, "alpha"}, {web, "web"}} {
		newWorkspaceWithID(t, s, w.id, w.slug, w.slug, alice)
		join(t, s, w.id, bob, shared.RoleMember)
	}
	tx := postgres.NewTxManager(pool, 10*time.Second)
	release := hold(t, tx, func(ctx context.Context) error {
		_, err := postgres.DB(ctx, pool).Exec(ctx, "SELECT 1 FROM workspaces WHERE id = $1 FOR NO KEY UPDATE", alpha)
		return err
	})
	done := make(chan error, 1)
	go func() {
		done <- tx.WithinTx(context.Background(), func(ctx context.Context) error {
			_, err := s.LockMemberWorkspaces(ctx, bob)
			return err
		})
	}()
	pgtest.WaitForLockWaitOn(t, pool, "workspaces", 10*time.Second)

	if !waitsFor(t, pool, web, "FOR SHARE") {
		t.Error("a FOR SHARE of web while LockMemberWorkspaces waits for alpha does not wait; want web locked first")
	}
	if err := release(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("LockMemberWorkspaces() = %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("LockMemberWorkspaces() did not end within 10s")
	}
}

// A workspace deleted while LockMemberWorkspaces waits for its row is left
// out, not an error (M3 design 3.9, review spike 15): another transaction
// holds acme FOR NO KEY UPDATE, as deleteWorkspace does, and soft-deletes
// it, its memberships too; LockMemberWorkspaces, which found acme
// undeleted, waits for it; once the deletion commits, it evaluates
// deleted_at again on the row's newest version and returns beta alone.
func TestLockMemberWorkspacesLeavesOutAWorkspaceDeletedWhileItWaited(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme, beta := newWorkspace(t, s, "Acme", "acme", alice).ID, newWorkspace(t, s, "Beta", "beta", alice).ID
	for _, w := range []uuid.UUID{acme, beta} {
		join(t, s, w, bob, shared.RoleMember)
	}
	tx := postgres.NewTxManager(pool, 10*time.Second)
	commit := hold(t, tx, func(ctx context.Context) error {
		for _, sql := range []string{"SELECT 1 FROM workspaces WHERE id = $1 FOR NO KEY UPDATE", "UPDATE workspaces SET deleted_at = now() WHERE id = $1",
			"UPDATE workspace_members SET deleted_at = now() WHERE workspace_id = $1"} {
			if _, err := postgres.DB(ctx, pool).Exec(ctx, sql, acme); err != nil {
				return err
			}
		}
		return nil
	})
	type locked struct {
		ids []uuid.UUID
		err error
	}
	done := make(chan locked, 1)
	go func() {
		var l locked
		l.err = tx.WithinTx(context.Background(), func(ctx context.Context) error {
			var err error
			l.ids, err = s.LockMemberWorkspaces(ctx, bob)
			return err
		})
		done <- l
	}()
	pgtest.WaitForLockWaitOn(t, pool, "workspaces", 10*time.Second)
	if err := commit(); err != nil {
		t.Fatal(err)
	}
	select {
	case l := <-done:
		if l.err != nil || !slices.Equal(l.ids, []uuid.UUID{beta}) {
			t.Errorf("LockMemberWorkspaces() = %v, %v; want beta (%s) alone, acme (%s) deleted while it waited", l.ids, l.err, beta, acme)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("LockMemberWorkspaces() did not end within 10s")
	}
}

// SoleAdmin reports whether bob is the only active admin of one of the
// workspaces asked about that has another active member (M3 design 3.7
// rule 2). Most cases ask about one workspace, in the state the case
// names; three ask about two at once, one of them with another admin or
// other members, so that a check of the set asked about, not of each of
// its workspaces, answers otherwise; one asks about none. Each workspace
// has the members its case names alone, its creator's membership deleted;
// the cases' workspaces all stand beside each other, so that a check of a
// workspace not asked about answers otherwise.
func TestSoleAdminOfAWorkspace(t *testing.T) {
	s, pool := newStore(t)
	alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	type member struct {
		user   uuid.UUID
		role   shared.Role
		active bool
	}
	// workspace stores a workspace with members, its creator's membership
	// deleted, the last of them deleted when deleteLast is set, and returns
	// its id.
	n := 0
	workspace := func(deleteLast bool, members ...member) uuid.UUID {
		t.Helper()
		n++
		w := newWorkspace(t, s, fmt.Sprintf("W%d", n), fmt.Sprintf("w%d", n), carol).ID
		exec(t, pool, "UPDATE workspace_members SET deleted_at = $2 WHERE workspace_id = $1", w, now)
		for i, m := range members {
			join(t, s, w, m.user, m.role)
			if !m.active {
				exec(t, pool, "UPDATE workspace_members SET is_active = false WHERE workspace_id = $1 AND member_id = $2 AND deleted_at IS NULL",
					w, m.user)
			}
			if deleteLast && i == len(members)-1 {
				exec(t, pool, "UPDATE workspace_members SET deleted_at = $3 WHERE workspace_id = $1 AND member_id = $2", w, m.user, now)
			}
		}
		return w
	}
	alone := workspace(false, member{bob, shared.RoleAdmin, true})
	withAMember := workspace(false, member{bob, shared.RoleAdmin, true}, member{carol, shared.RoleMember, true})
	twoAdmins := workspace(false, member{bob, shared.RoleAdmin, true}, member{alice, shared.RoleAdmin, true}, member{carol, shared.RoleMember, true})
	workspace(false, member{alice, shared.RoleAdmin, true}, member{carol, shared.RoleMember, true}) // not asked about
	tests := []struct {
		name       string
		workspaces []uuid.UUID
		want       bool
	}{
		{"the only admin, with an active member", []uuid.UUID{withAMember}, true},
		{"the only admin, with an active guest", []uuid.UUID{workspace(false, member{bob, shared.RoleAdmin, true},
			member{carol, shared.RoleGuest, true})}, true},
		{"the only admin, alone", []uuid.UUID{alone}, false},
		{"the only admin, the other's membership ended", []uuid.UUID{workspace(false, member{bob, shared.RoleAdmin, true},
			member{carol, shared.RoleMember, false})}, false},
		{"the only admin, the other's membership deleted", []uuid.UUID{workspace(true, member{bob, shared.RoleAdmin, true},
			member{carol, shared.RoleMember, true})}, false},
		{"one of two active admins", []uuid.UUID{twoAdmins}, false},
		{"the other admin's membership ended", []uuid.UUID{workspace(false, member{bob, shared.RoleAdmin, true},
			member{alice, shared.RoleAdmin, false}, member{carol, shared.RoleMember, true})}, true},
		{"the other admin's membership deleted", []uuid.UUID{workspace(true, member{bob, shared.RoleAdmin, true},
			member{carol, shared.RoleMember, true}, member{alice, shared.RoleAdmin, true})}, true},
		{"a member, beside the only admin and another member", []uuid.UUID{workspace(false, member{bob, shared.RoleMember, true},
			member{alice, shared.RoleAdmin, true}, member{carol, shared.RoleMember, true})}, false},
		{"a member, beside another member and no admin", []uuid.UUID{workspace(false, member{bob, shared.RoleMember, true},
			member{carol, shared.RoleMember, true})}, false},
		{"an admin whose membership ended", []uuid.UUID{workspace(false, member{bob, shared.RoleAdmin, false},
			member{carol, shared.RoleMember, true})}, false},
		{"an admin whose membership is deleted", []uuid.UUID{workspace(true, member{carol, shared.RoleMember, true},
			member{bob, shared.RoleAdmin, true})}, false},
		{"the only admin of one of two asked about", []uuid.UUID{alone, withAMember}, true},
		{"the only admin of one, another asked about having another admin", []uuid.UUID{withAMember, twoAdmins}, true},
		{"alone in one, another asked about having other members", []uuid.UUID{alone, twoAdmins}, false},
		{"none asked about", nil, false},
	}
	for _, tt := range tests {
		if got, err := s.SoleAdmin(context.Background(), tt.workspaces, bob); err != nil || got != tt.want {
			t.Errorf("%s: SoleAdmin() = %v, %v; want %v", tt.name, got, err, tt.want)
		}
	}
}

// DeleteInvitationsTo soft-deletes every undeleted invitation to the
// address, of every workspace, pending or declined, by the account and at
// the time given, and changes nothing else (M3 design 3.8, 3.9): bob's
// pending one to acme, his declined one to beta, his pending one to gamma,
// whose member he is not. Two earlier ones to his address in acme, one
// accepted, one deleted while pending, keep their moments; carol's pending
// one to acme, and an address that only ends with his, every column.
func TestDeleteInvitationsTo(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme, beta, gamma := newWorkspace(t, s, "Acme", "acme", alice).ID, newWorkspace(t, s, "Beta", "beta", alice).ID,
		newWorkspace(t, s, "Gamma", "gamma", alice).ID
	exec(t, pool, `INSERT INTO workspace_member_invites (id, workspace_id, email, role, accepted, responded_at, created_by_id,
		updated_by_id, created_at, updated_at, deleted_at) VALUES ($1, $2, 'bob@corp.com', 15, true, $4, $3, $3, $4, $4, $4)`,
		uuid.NewV7(), acme, alice, now.Add(-time.Hour))
	exec(t, pool, `INSERT INTO workspace_member_invites (id, workspace_id, email, role, created_by_id, updated_by_id, created_at,
		updated_at, deleted_at) VALUES ($1, $2, 'bob@corp.com', 5, $3, $3, $4, $4, $4)`, uuid.NewV7(), acme, alice, now.Add(-time.Hour))
	deleted := []uuid.UUID{
		invite(t, s, acme, "bob@corp.com", shared.RoleGuest, alice).ID,
		invite(t, s, beta, "bob@corp.com", shared.RoleMember, alice).ID,
		invite(t, s, gamma, "bob@corp.com", shared.RoleAdmin, alice).ID,
	}
	exec(t, pool, "UPDATE workspace_member_invites SET responded_at = $2 WHERE id = $1", deleted[1], now)
	invite(t, s, acme, "carol@corp.com", shared.RoleMember, alice)
	invite(t, s, beta, "rebob@corp.com", shared.RoleMember, alice)
	before := tableRows(t, pool, "workspace_member_invites", deleted, "deleted_at", "updated_at", "updated_by_id")
	later := now.Add(time.Hour)

	if err := s.DeleteInvitationsTo(context.Background(), "bob@corp.com", bob, later); err != nil {
		t.Fatal(err)
	}

	for i, id := range deleted {
		if got, want := stamp(t, pool, "workspace_member_invites", id), fmt.Sprintf("deleted at %[1]s at %[1]s by %[2]s", at(t, pool, later),
			bob); got != want {
			t.Errorf("bob's invitation %d: %s, want %s", i, got, want)
		}
	}
	if after := tableRows(t, pool, "workspace_member_invites", deleted, "deleted_at", "updated_at", "updated_by_id"); after != before {
		t.Errorf("the invitations, bob's three without the columns written:\n%s\nwant them as they were:\n%s", after, before)
	}
}

// EndWorkspaceMemberships ends bob's active, undeleted memberships of the
// workspaces asked about, acme's as a member and gamma's as an admin, each
// keeping its role, by the account and at the time given, though alice
// wrote them last, and changes
// nothing else (M3 design 3.6 convention 6): his membership of beta,
// asked about but ended by alice before, keeps its ender and moment; his
// deleted membership of acme, carol's of acme and his of delta, not asked
// about, every column.
func TestEndWorkspaceMemberships(t *testing.T) {
	s, pool := newStore(t)
	alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	acme, beta := newWorkspace(t, s, "Acme", "acme", alice).ID, newWorkspace(t, s, "Beta", "beta", alice).ID
	gamma, delta := newWorkspace(t, s, "Gamma", "gamma", alice).ID, newWorkspace(t, s, "Delta", "delta", alice).ID
	exec(t, pool, `INSERT INTO workspace_members (id, workspace_id, member_id, role, created_by_id, updated_by_id, created_at, updated_at,
		deleted_at) VALUES ($1, $2, $3, 20, $3, $3, $4, $4, $4)`, uuid.NewV7(), acme, bob, now.Add(-time.Hour))
	ended := []uuid.UUID{joinAt(t, s, acme, bob, shared.RoleMember, now).ID, joinAt(t, s, gamma, bob, shared.RoleAdmin, now).ID}
	exec(t, pool, "UPDATE workspace_members SET updated_by_id = $2 WHERE id = ANY($1)", ended, alice)
	joinAt(t, s, beta, bob, shared.RoleGuest, now)
	exec(t, pool, "UPDATE workspace_members SET is_active = false, updated_by_id = $3 WHERE workspace_id = $1 AND member_id = $2", beta, bob, alice)
	joinAt(t, s, acme, carol, shared.RoleMember, now)
	joinAt(t, s, delta, bob, shared.RoleMember, now)
	before := tableRows(t, pool, "workspace_members", ended, "is_active", "updated_at", "updated_by_id")
	later := now.Add(time.Hour)

	if err := s.EndWorkspaceMemberships(context.Background(), []uuid.UUID{acme, beta, gamma}, bob, bob, later); err != nil {
		t.Fatal(err)
	}

	for i, id := range ended {
		if got, want := stamp(t, pool, "workspace_members", id), fmt.Sprintf("ended at %s by %s", at(t, pool, later), bob); got != want {
			t.Errorf("bob's membership %d: %s, want %s", i, got, want)
		}
	}
	if after := tableRows(t, pool, "workspace_members", ended, "is_active", "updated_at", "updated_by_id"); after != before {
		t.Errorf("the memberships, bob's two ended without the columns written:\n%s\nwant them as they were:\n%s", after, before)
	}
}
````

`server/internal/modules/workspace/adapter/postgres/failures_test.go`（修改，2 处）：

````old server/internal/modules/workspace/adapter/postgres/failures_test.go
		t.Errorf("HasOtherAdmin() = %v, %v; want context.Canceled, not another admin", other, err)
	}
````
````new server/internal/modules/workspace/adapter/postgres/failures_test.go
		t.Errorf("HasOtherAdmin() = %v, %v; want context.Canceled, not another admin", other, err)
	}
	if ids, err := s.LockMemberWorkspaces(cancelled, alice); !failed(err) || ids != nil {
		t.Errorf("LockMemberWorkspaces() = %v, %v; want context.Canceled, no workspace", ids, err)
	}
	if sole, err := s.SoleAdmin(cancelled, []uuid.UUID{w.ID}, alice); !failed(err) || sole {
		t.Errorf("SoleAdmin() = %v, %v; want context.Canceled, not the only admin", sole, err)
	}
````

````old server/internal/modules/workspace/adapter/postgres/failures_test.go
		t.Errorf("DeletePendingInvitations() = %v; want context.Canceled", err)
	}
````
````new server/internal/modules/workspace/adapter/postgres/failures_test.go
		t.Errorf("DeletePendingInvitations() = %v; want context.Canceled", err)
	}
	if err := s.DeleteInvitationsTo(cancelled, "carol@corp.com", alice, now); !failed(err) {
		t.Errorf("DeleteInvitationsTo() = %v; want context.Canceled", err)
	}
	if err := s.EndWorkspaceMemberships(cancelled, []uuid.UUID{w.ID}, bob.MemberID, bob.MemberID, now); !failed(err) {
		t.Errorf("EndWorkspaceMemberships() = %v; want context.Canceled", err)
	}
````

- [ ] **Step 3: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/workspace/...`
Expected: 全部 `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`（`AllMembershipsEnder` 这时只由存储实现，`app` 包里还没有使用者；它是导出的接口，lint 不报未使用）

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 4: 提交**

```bash
git add server/internal/modules/workspace/adapter/postgres/deactivation.go server/internal/modules/workspace/adapter/postgres/deactivation_test.go server/internal/modules/workspace/adapter/postgres/failures_test.go server/internal/modules/workspace/adapter/postgres/queries/deactivation.sql server/internal/modules/workspace/app/ports.go server/internal/modules/workspace/adapter/postgres/gen/deactivation.sql.go
```
```bash
git commit -m "feat(M3/P6): the workspace store locks a member's workspaces, asks for their only admin, deletes the invitations to an address and ends his memberships

LockMemberWorkspaces locks the undeleted workspaces of which an
account is an active member FOR NO KEY UPDATE in id order, leaving out
one deleted while its lock waited; SoleAdmin asks whether he is the
only active admin of one of them that has another active member;
DeleteInvitationsTo soft-deletes every invitation to an address,
pending or declined; EndWorkspaceMemberships ends his active
memberships of the workspaces, the rows and their roles kept. Each
store test checks every column of every other row, and each method's
failure comes back as an error, never a plausible answer.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p6.py` 的编号，"层"是它被发现的每一层；端到端是单独运行的故事）：

| 变异 | 改坏 | 必须失败的测试 | 层 |
|---|---|---|---|
| `o-ws-desc` | `LockMemberWorkspaces` 按 id 降序锁 | `TestLockMemberWorkspaces`、`TestLockMemberWorkspacesLocksInIDOrder`、`TestEachLockOfADeactivationIsItsStrength`（Task 6 起） | 存储；组合 |
| `o-ws-unordered` | `LockMemberWorkspaces` 去掉 `ORDER BY` | `TestLockMemberWorkspacesLocksInIDOrder`、`TestEachLockOfADeactivationIsItsStrength`（Task 6 起） | 存储；组合 |
| `o-ws-share` | `LockMemberWorkspaces` 锁成 `FOR SHARE` | `TestLockMemberWorkspaces`、`TestLockMemberWorkspacesLocksInIDOrder`、`TestADeactivationAndACreationHeLeadsSerialize`（Task 8 起）、`TestADeactivationAndTheProjectSidesGrowthSerialize`（Task 8 起）、`TestADeactivationFindsWhatChangedMeanwhile`（Task 6 起）、`TestAnAdmittedAdminsWriteAndHisDeactivation`（Task 7 起） 等 6 个 | 存储；组合 |
| `o-ws-update` | `LockMemberWorkspaces` 锁成 `FOR UPDATE` | `TestLockMemberWorkspaces`、`TestEachLockOfADeactivationIsItsStrength`（Task 6 起） | 存储；组合 |
| `o-ws-unlocked` | `LockMemberWorkspaces` 不加锁 | `TestLockMemberWorkspaces`、`TestLockMemberWorkspacesLeavesOutAWorkspaceDeletedWhileItWaited`、`TestLockMemberWorkspacesLocksInIDOrder`、`TestADeactivationAndACreationHeLeadsSerialize`（Task 8 起）、`TestADeactivationAndTheProjectSidesGrowthSerialize`（Task 8 起）、`TestADeactivationFindsWhatChangedMeanwhile`（Task 6 起）、`TestAnAdmittedAdminsWriteAndHisDeactivation`（Task 7 起） 等 6 个 | 存储；组合 |
| `o-spike15-404` | 先列举再逐个上锁，上锁时已删除的答 404 | `TestLockMemberWorkspacesLeavesOutAWorkspaceDeletedWhileItWaited`、`TestADeactivationFindsWhatChangedMeanwhile`（Task 6 起） | 存储；组合 |
| `o-spike15-500` | 同上，答一个失败 | `TestLockMemberWorkspacesLeavesOutAWorkspaceDeletedWhileItWaited`、`TestADeactivationFindsWhatChangedMeanwhile`（Task 6 起） | 存储；组合 |
| `o-spike15-locked-deleted` | `LockMemberWorkspaces` 去掉 `w.deleted_at IS NULL`（已删除的工作区也锁、也回答） | `TestLockMemberWorkspaces`、`TestLockMemberWorkspacesLeavesOutAWorkspaceDeletedWhileItWaited` | 存储 |
| `s1-lmw-correlated` | `LockMemberWorkspaces` 去掉 `m.workspace_id = w.id` | `TestLockMemberWorkspaces`、`TestEachLockOfADeactivationIsItsStrength`（Task 6 起） | 存储；组合 |
| `s1-lmw-member` | `LockMemberWorkspaces` 去掉成员 | `TestLockMemberWorkspaces`、`TestEachLockOfADeactivationIsItsStrength`（Task 6 起） | 存储；组合 |
| `s1-lmw-active` | `LockMemberWorkspaces` 去掉 `m.is_active` | `TestLockMemberWorkspaces`、`TestEachLockOfADeactivationIsItsStrength`（Task 6 起） | 存储；组合 |
| `s1-lmw-mdeleted` | `LockMemberWorkspaces` 去掉 `m.deleted_at IS NULL` | `TestLockMemberWorkspaces` | 存储 |
| `s1-sole-workspaces` | `SoleAdmin` 去掉问到的工作区 | `TestSoleAdminOfAWorkspace` | 存储 |
| `s1-sole-member` | `SoleAdmin` 去掉成员 | `TestSoleAdminOfAWorkspace`、`TestADeactivationAndACreationHeLeadsSerialize`（Task 8 起）、`TestADeactivationAndTheProjectSidesGrowthSerialize`（Task 8 起）、`TestADeactivationEndsEveryMembership`（Task 5 起）、`TestADeactivationFindsWhatChangedMeanwhile`（Task 6 起） 等 11 个、W9（Task 9 起） | 存储；组合；端到端 |
| `s1-sole-role` | `SoleAdmin` 去掉他的 `role = 20` | `TestSoleAdminOfAWorkspace` | 存储 |
| `s1-sole-active` | `SoleAdmin` 去掉他的 `is_active` | `TestSoleAdminOfAWorkspace` | 存储 |
| `s1-sole-mdeleted` | `SoleAdmin` 去掉他的 `deleted_at IS NULL` | `TestSoleAdminOfAWorkspace` | 存储 |
| `s1-sole-a-correlated` | `SoleAdmin` 的另一位管理员去掉同一工作区 | `TestSoleAdminOfAWorkspace`、`TestARefusedDeactivationChangesNothing`（Task 5 起）、`TestTwoAdminsDeactivatedAtOnceLeaveAnAdmin`（Task 8 起）、W9（Task 9 起） | 存储；组合；端到端 |
| `s1-sole-a-other` | `SoleAdmin` 的另一位管理员把他自己算进去 | `TestSoleAdminOfAWorkspace`、`TestARefusedDeactivationChangesNothing`（Task 5 起）、`TestAnAdmittedAdminsWriteAndHisDeactivation`（Task 7 起）、`TestTwoAdminsDeactivatedAtOnceLeaveAnAdmin`（Task 8 起）、`TestUsersCommandsFail`（Task 5 起）、W9（Task 9 起） | 存储；组合；端到端 |
| `s1-sole-a-role` | `SoleAdmin` 的另一位管理员去掉 `role = 20` | `TestSoleAdminOfAWorkspace`、`TestARefusedDeactivationChangesNothing`（Task 5 起）、`TestAnAdmittedAdminsWriteAndHisDeactivation`（Task 7 起）、`TestTwoAdminsDeactivatedAtOnceLeaveAnAdmin`（Task 8 起）、`TestUsersCommandsFail`（Task 5 起）、W9（Task 9 起） | 存储；组合；端到端 |
| `s1-sole-a-active` | `SoleAdmin` 的另一位管理员去掉 `is_active` | `TestSoleAdminOfAWorkspace`、`TestARefusedDeactivationChangesNothing`（Task 5 起）、`TestTwoAdminsDeactivatedAtOnceLeaveAnAdmin`（Task 8 起） | 存储；组合 |
| `s1-sole-a-deleted` | `SoleAdmin` 的另一位管理员去掉 `deleted_at IS NULL` | `TestSoleAdminOfAWorkspace` | 存储 |
| `s1-sole-o-correlated` | `SoleAdmin` 的别的成员去掉同一工作区 | `TestSoleAdminOfAWorkspace`、`TestADeactivationEndsEveryMembership`（Task 5 起） | 存储；组合 |
| `s1-sole-o-other` | `SoleAdmin` 的别的成员把他自己算进去 | `TestSoleAdminOfAWorkspace`、`TestADeactivationEndsEveryMembership`（Task 5 起）、`TestCreationFirstHoldsOffTheDeactivation`（Task 4 起） | 存储；组合 |
| `s1-sole-o-active` | `SoleAdmin` 的别的成员去掉 `is_active` | `TestSoleAdminOfAWorkspace`、`TestADeactivationEndsEveryMembership`（Task 5 起） | 存储；组合 |
| `s1-sole-o-deleted` | `SoleAdmin` 的别的成员去掉 `deleted_at IS NULL` | `TestSoleAdminOfAWorkspace` | 存储 |
| `s1-dit-email` | `DeleteInvitationsTo` 去掉邮箱 | `TestDeleteInvitationsTo`、`TestADeactivationEndsEveryMembership`（Task 5 起）、`TestADeactivationReadsTheAddressUnderItsLock`（Task 6 起） | 存储；组合 |
| `s1-dit-deleted` | `DeleteInvitationsTo` 去掉 `deleted_at IS NULL` | `TestDeleteInvitationsTo`、`TestADeactivationEndsEveryMembership`（Task 5 起）、`TestADeactivationFindsWhatChangedMeanwhile`（Task 6 起） | 存储；组合 |
| `s1-ewm-workspaces` | `EndWorkspaceMemberships` 去掉工作区 | `TestEndWorkspaceMemberships` | 存储 |
| `s1-ewm-member` | `EndWorkspaceMemberships` 去掉成员 | `TestEndWorkspaceMemberships`、`TestADeactivationAndACreationHeLeadsSerialize`（Task 8 起）、`TestADeactivationAndTheProjectSidesGrowthSerialize`（Task 8 起）、`TestADeactivationEndsEveryMembership`（Task 5 起）、`TestReactivatingAndDeactivating`（Task 7 起） 等 5 个、W9（Task 9 起） | 存储；组合；端到端 |
| `s1-ewm-active` | `EndWorkspaceMemberships` 去掉 `is_active`（等锁时已结束的再写一次） | `TestEndWorkspaceMemberships`、`TestADeactivationFindsWhatChangedMeanwhile`（Task 6 起） | 存储；组合 |
| `s1-ewm-deleted` | `EndWorkspaceMemberships` 去掉 `deleted_at IS NULL` | `TestEndWorkspaceMemberships` | 存储 |
| `s3-dit-writes-created` | `DeleteInvitationsTo` 也写 `created_at` | `TestDeleteInvitationsTo`、`TestADeactivationEndsEveryMembership`（Task 5 起） | 存储；组合 |
| `s3-dit-answers` | `DeleteInvitationsTo` 把待接受的标成已回答 | `TestDeleteInvitationsTo`、`TestADeactivationEndsEveryMembership`（Task 5 起）、`TestDecliningAndDeactivating`（Task 7 起） | 存储；组合 |
| `s3-dit-keeps-time` | `DeleteInvitationsTo` 不写 `updated_at` | `TestDeleteInvitationsTo`、`TestADeactivationEndsEveryMembership`（Task 5 起） | 存储；组合 |
| `s14-dit-keeps-writer` | `DeleteInvitationsTo` 保留原来的写者 | `TestDeleteInvitationsTo`、`TestADeactivationEndsEveryMembership`（Task 5 起）、`TestADeactivationFindsWhatChangedMeanwhile`（Task 6 起）、`TestADeactivationReadsTheAddressUnderItsLock`（Task 6 起）、`TestAcceptingAndDeactivating`（Task 7 起） 等 5 个、W9（Task 9 起） | 存储；组合；端到端 |
| `s14-dit-by-inviter` | `DeleteInvitationsTo` 写成邀请人 | `TestDeleteInvitationsTo`、`TestADeactivationEndsEveryMembership`（Task 5 起）、`TestADeactivationFindsWhatChangedMeanwhile`（Task 6 起）、`TestADeactivationReadsTheAddressUnderItsLock`（Task 6 起）、`TestAcceptingAndDeactivating`（Task 7 起） 等 5 个、W9（Task 9 起） | 存储；组合；端到端 |
| `s3-ewm-keeps-time` | `EndWorkspaceMemberships` 保留原来的时刻 | `TestEndWorkspaceMemberships`、`TestADeactivationEndsEveryMembership`（Task 5 起）、`TestDecliningAndDeactivating`（Task 7 起）、W9（Task 9 起） | 存储；组合；端到端 |
| `s14-ewm-keeps-writer` | `EndWorkspaceMemberships` 保留原来的写者 | `TestEndWorkspaceMemberships`、`TestADeactivationEndsEveryMembership`（Task 5 起）、`TestADeactivationFindsWhatChangedMeanwhile`（Task 6 起）、`TestADeactivationReadsTheAddressUnderItsLock`（Task 6 起）、`TestDecliningAndDeactivating`（Task 7 起） 等 6 个 | 存储；组合 |
| `s3-ewm-writes-created` | `EndWorkspaceMemberships` 也写 `created_at` | `TestEndWorkspaceMemberships`、`TestADeactivationEndsEveryMembership`（Task 5 起） | 存储；组合 |
| `s3-ewm-deletes` | `EndWorkspaceMemberships` 也删除那一行 | `TestEndWorkspaceMemberships`、`TestADeactivationEndsEveryMembership`（Task 5 起）、`TestADeactivationFindsWhatChangedMeanwhile`（Task 6 起）、`TestADeactivationReadsTheAddressUnderItsLock`（Task 6 起）、W9（Task 9 起） | 存储；组合；端到端 |
| `s26-ewm-role-20` | `EndWorkspaceMemberships` 把角色写成 20 | `TestEndWorkspaceMemberships`、`TestADeactivationAndTheProjectSidesGrowthSerialize`（Task 8 起）、`TestADeactivationEndsEveryMembership`（Task 5 起）、W9（Task 9 起） | 存储；组合；端到端 |
| `s26-ewm-role-15` | `EndWorkspaceMemberships` 把角色写成 15 | `TestEndWorkspaceMemberships`、`TestADeactivationEndsEveryMembership`（Task 5 起）、`TestAcceptingAndDeactivating`（Task 7 起）、`TestAnAdmittedAdminsWriteAndHisDeactivation`（Task 7 起）、`TestReactivatingAndDeactivating`（Task 7 起）、W9（Task 9 起） | 存储；组合；端到端 |
| `s26-ewm-role-default` | `EndWorkspaceMemberships` 把角色写成列的默认（访客的） | `TestEndWorkspaceMemberships`、`TestADeactivationAndTheProjectSidesGrowthSerialize`（Task 8 起）、`TestADeactivationEndsEveryMembership`（Task 5 起）、`TestAcceptingAndDeactivating`（Task 7 起）、`TestAnAdmittedAdminsWriteAndHisDeactivation`（Task 7 起） 等 5 个、W9（Task 9 起） | 存储；组合；端到端 |
| `s8-lmw-pool` | `LockMemberWorkspaces` 经连接池执行 | `TestLockMemberWorkspaces`、`TestADeactivationAndACreationHeLeadsSerialize`（Task 8 起）、`TestADeactivationAndTheProjectSidesGrowthSerialize`（Task 8 起）、`TestEachLockOfADeactivationIsItsStrength`（Task 6 起）、`TestTheDeactivationRunsOnItsTransactionsConnection`（Task 6 起） 等 5 个 | 存储；组合 |
| `s8-sole-pool` | `SoleAdmin` 经连接池执行 | `TestTheDeactivationRunsOnItsTransactionsConnection`（Task 6 起） | 组合 |
| `s8-dit-pool` | `DeleteInvitationsTo` 经连接池执行（提交在拒绝之前） | `TestADeactivationRefusedAtItsCommitChangesNothing`（Task 5 起）、`TestARefusedDeactivationChangesNothing`（Task 5 起）、`TestAnInterruptedDeactivationChangesNothing`（Task 6 起）、`TestEachLockOfADeactivationIsItsStrength`（Task 6 起） 等 5 个、W9（Task 9 起） | 组合；端到端 |
| `s8-ewm-pool` | `EndWorkspaceMemberships` 经连接池执行 | `TestADeactivationRefusedAtItsCommitChangesNothing`（Task 5 起）、`TestARefusedDeactivationChangesNothing`（Task 5 起）、`TestAnInterruptedDeactivationChangesNothing`（Task 6 起）、`TestEachLockOfADeactivationIsItsStrength`（Task 6 起） 等 6 个 | 组合 |
| `s15-dit-pending-only` | `DeleteInvitationsTo` 只删待接受的（`membershipEnd` 的一步） | `TestDeleteInvitationsTo`、`TestADeactivationEndsEveryMembership`（Task 5 起）、`TestADeactivationFindsWhatChangedMeanwhile`（Task 6 起）、`TestDecliningAndDeactivating`（Task 7 起）、`TestEachLockOfADeactivationIsItsStrength`（Task 6 起）、W9（Task 9 起） | 存储；组合；端到端 |
| `s19-lmw-swallowed` | `LockMemberWorkspaces` 的失败读成"没有工作区" | `TestAFailedReadIsAnErrorNotAnAnswer` | 存储 |
| `s19-sole-swallowed` | `SoleAdmin` 的失败读成"不是唯一的管理员" | `TestAFailedReadIsAnErrorNotAnAnswer` | 存储 |
| `s19-dit-swallowed` | `DeleteInvitationsTo` 的失败被吞掉 | `TestAFailedWriteIsAnError` | 存储 |
| `s19-ewm-swallowed` | `EndWorkspaceMemberships` 的失败被吞掉 | `TestAFailedWriteIsAnError` | 存储 |
| `s20-sole-admin-any` | `SoleAdmin` 的另一位管理员改成"问到的任何工作区的" | `TestSoleAdminOfAWorkspace`、`TestARefusedDeactivationChangesNothing`（Task 5 起）、W9（Task 9 起） | 存储；组合；端到端 |
| `s20-sole-other-any` | `SoleAdmin` 的别的成员改成"问到的任何工作区的" | `TestSoleAdminOfAWorkspace`、`TestADeactivationEndsEveryMembership`（Task 5 起） | 存储；组合 |
| `s24-ws-only-ignored` | `SoleAdmin` 不看另一位管理员（"唯一"一半） | `TestSoleAdminOfAWorkspace`、`TestADeactivationEndsEveryMembership`（Task 5 起）、`TestADeactivationFindsWhatChangedMeanwhile`（Task 6 起）、`TestADeactivationReadsTheAddressUnderItsLock`（Task 6 起）、`TestADeactivationRefusedAtItsCommitChangesNothing`（Task 5 起） 等 12 个、W9（Task 9 起） | 存储；组合；端到端 |
| `s24-ws-alone-refused` | `SoleAdmin` 不看别的成员（只有他一人也拒绝） | `TestSoleAdminOfAWorkspace`、`TestADeactivationEndsEveryMembership`（Task 5 起）、`TestCreationFirstHoldsOffTheDeactivation`（Task 4 起） | 存储；组合 |
| `s31-check-before-lock` | 规则 2 在不加锁的列举上检查，写之前才锁工作区（检查和执行分开） | `TestLockMemberWorkspaces`、`TestLockMemberWorkspacesLeavesOutAWorkspaceDeletedWhileItWaited`、`TestLockMemberWorkspacesLocksInIDOrder`、`TestADeactivationAndTheProjectSidesGrowthSerialize`（Task 8 起）、`TestADeactivationFindsWhatChangedMeanwhile`（Task 6 起）、`TestAnAdmittedAdminsWriteAndHisDeactivation`（Task 7 起）、`TestEachLockOfADeactivationIsItsStrength`（Task 6 起） 等 5 个 | 存储；组合 |

**Done when:** 四个方法在 `AllMembershipsEnder` 里、由 `Store` 实现；六个存储测试和两个失败测试通过；`grep -n "role = 20" server/internal/modules/workspace/adapter/postgres/queries/deactivation.sql` 只有 `SoleAdmin` 的两处（"管理员"这个集合）；`make lint-go`、`make test` 通过；生成物与表相同。

### Task 2: `project.NewCascade`；跨工作区的项目锁顺序

**Files:**
- Modify: `server/internal/bootstrap/interleaving_answers_test.go`、`server/internal/bootstrap/interleaving_deletion_test.go`、`server/internal/bootstrap/interleaving_endings_test.go`、`server/internal/bootstrap/interleaving_growth_test.go`、`server/internal/bootstrap/interleaving_leaving_test.go`、`server/internal/bootstrap/interleaving_removal_test.go`、`server/internal/bootstrap/interleaving_roles_test.go`、`server/internal/modules/project/adapter/postgres/end_test.go`、`server/internal/modules/project/module.go`

**Interfaces:**
- Produces（spec 2.4，M3 设计 6.3、6.6，设计第 1043 行）：`project.CascadeDeps{Pool *pgxpool.Pool}`；`project.NewCascade(CascadeDeps) Cascade`：在连接池上建出模块的连带（`app.NewCascade(store, store, store)`），给只要连带、不建 HTTP 一侧的组合（Task 4 的 `nerve users`）；`project.New` 经它建出同一个实现，`Module.cascade` 的类型是 `Cascade`。`Cascade.EndMemberships` 的说明补上"跨这些工作区按项目 id 锁住"。
- Consumes：P5a 的 `LockActiveMemberProjects`（按项目 id 排序，不按工作区），不改。

**Tests:**
- `project/adapter/postgres/end_test.go` 的 `TestLockActiveMemberProjectsLocksInIDOrder` 改为跨两个工作区（P5a spec 第 5 节 P6 一行）：Web（acme）、Alpha（beta）、Zed（acme）的 id 依次增大，在表和名称、标识的索引里排成 Alpha、Zed、Web；Alpha 被持有时，跨 acme、beta 的 `LockActiveMemberProjects` 等它，已持有 Web、还没有 Zed：按行序的会先碰到 Alpha、什么都不持有；一个工作区接一个工作区的会连 Zed 也持有。
- 七个交错测试文件里只给 `Pool` 的 `project.New(project.Deps{Pool: …}).Cascade()` 换成 `project.NewCascade(project.CascadeDeps{Pool: …})`（P4a review 第 6 节 S5）；它们照旧通过。

- [ ] **Step 1: 构造**

`server/internal/modules/project/module.go`（修改，3 处）：

````old server/internal/modules/project/module.go
	// projects, found when it is called; project.sole_admin, and nothing
	// ended, when he is the only active admin of one that has other active
	// members.
	EndMemberships(ctx context.Context, workspaceIDs []uuid.UUID, userID, by uuid.UUID, now time.Time) error
````
````new server/internal/modules/project/module.go
	// projects, found when it is called and locked in the projects' id
	// order across the workspaces; project.sole_admin, and nothing ended,
	// when he is the only active admin of one that has other active members.
	EndMemberships(ctx context.Context, workspaceIDs []uuid.UUID, userID, by uuid.UUID, now time.Time) error
}

// CascadeDeps are what a composition that wants the cascade alone gives
// it (M3 design 6.3, 6.6).
type CascadeDeps struct {
	Pool *pgxpool.Pool
}

// NewCascade builds the module's Cascade alone, on the pool: for the
// command line, whose `nerve users deactivate` ends memberships through it
// and builds no HTTP side (M3 design 6.3, 6.6). New builds its own through
// it.
func NewCascade(d CascadeDeps) Cascade {
	store := postgresadapter.New(d.Pool)
	return app.NewCascade(store, store, store)
````

````old server/internal/modules/project/module.go
	cascade *app.Cascade
````
````new server/internal/modules/project/module.go
	cascade Cascade
````

````old server/internal/modules/project/module.go
	return &Module{cascade: app.NewCascade(store, store, store), uc: httpadapter.UseCases{
````
````new server/internal/modules/project/module.go
	return &Module{cascade: NewCascade(CascadeDeps{Pool: d.Pool}), uc: httpadapter.UseCases{
````

- [ ] **Step 2: 跨工作区的锁顺序**

`server/internal/modules/project/adapter/postgres/end_test.go`（修改，5 处）：

````old server/internal/modules/project/adapter/postgres/end_test.go
// LockActiveMemberProjects takes its locks in the projects' id order,
// whatever order the rows lie in: Web has the smaller id, but lies after
// Alpha in the table and in the indexes on the name and on the identifier.
// Alpha's row is held. LockActiveMemberProjects waits for it holding
// Web's, which a FOR SHARE then waits for; in any other order it would
// reach Alpha first and wait holding nothing.
````
````new server/internal/modules/project/adapter/postgres/end_test.go
// LockActiveMemberProjects takes its locks in the projects' id order, across
// the workspaces asked about, whatever order the rows lie in, and whatever
// workspace each is of (M3 design 3.6 convention 6: the deactivation locks
// every project at once, not a workspace's after another's). Web, acme's,
// has the smallest id, then Alpha, beta's, then Zed, acme's; they lie in
// the table and in the indexes on the name and on the identifier as Alpha,
// Zed, Web. Alpha's row is held. LockActiveMemberProjects waits for it
// holding Web's, which a FOR SHARE then waits for, and not Zed's, which it
// does not; in the rows' order it would reach Alpha first and wait holding
// nothing; a workspace's projects after another's, acme's first, it would
// hold Zed too.
````

````old server/internal/modules/project/adapter/postgres/end_test.go
	acme := newWorkspace(t, pool, "acme")
	web, alpha := uuid.NewV7(), uuid.NewV7() // web drawn first: the smaller id
````
````new server/internal/modules/project/adapter/postgres/end_test.go
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	web, alpha, zed := uuid.NewV7(), uuid.NewV7(), uuid.NewV7() // drawn in this order: ascending ids
````

````old server/internal/modules/project/adapter/postgres/end_test.go
		id   uuid.UUID
		name string
	}{{alpha, "Alpha"}, {web, "Web"}} {
		exec(t, pool, "INSERT INTO projects (id, workspace_id, name, identifier) VALUES ($1, $2, $3::text, upper($3::text))", p.id, acme, p.name)
		seedMember(t, pool, acme, p.id, bob, 15, true)
````
````new server/internal/modules/project/adapter/postgres/end_test.go
		id, workspace uuid.UUID
		name          string
	}{{alpha, beta, "Alpha"}, {zed, acme, "Zed"}, {web, acme, "Web"}} {
		exec(t, pool, "INSERT INTO projects (id, workspace_id, name, identifier) VALUES ($1, $2, $3::text, upper($3::text))", p.id, p.workspace,
			p.name)
		seedMember(t, pool, p.workspace, p.id, bob, 15, true)
````

````old server/internal/modules/project/adapter/postgres/end_test.go
			_, err := s.LockActiveMemberProjects(ctx, []uuid.UUID{acme}, bob)
````
````new server/internal/modules/project/adapter/postgres/end_test.go
			_, err := s.LockActiveMemberProjects(ctx, []uuid.UUID{acme, beta}, bob)
````

````old server/internal/modules/project/adapter/postgres/end_test.go
	if !waits(t, pool, web, "FOR SHARE") {
		t.Error("a FOR SHARE of Web while LockActiveMemberProjects waits for Alpha does not wait; want Web locked first")
````
````new server/internal/modules/project/adapter/postgres/end_test.go
	if web, zed := waits(t, pool, web, "FOR SHARE"), waits(t, pool, zed, "FOR SHARE"); !web || zed {
		t.Errorf("while LockActiveMemberProjects waits for Alpha, a FOR SHARE of Web waits %v, of Zed %v; want Web locked first, Zed not yet",
			web, zed)
````

- [ ] **Step 3: 交错测试里的部分构造**

`server/internal/bootstrap/interleaving_answers_test.go`（修改，1 处）：

````old server/internal/bootstrap/interleaving_answers_test.go
	return project.New(project.Deps{Pool: r.pool}).Cascade()
````
````new server/internal/bootstrap/interleaving_answers_test.go
	return project.NewCascade(project.CascadeDeps{Pool: r.pool})
````

`server/internal/bootstrap/interleaving_deletion_test.go`（修改，1 处）：

````old server/internal/bootstrap/interleaving_deletion_test.go
	return workspaceapp.NewDeleteWorkspace(workspaces, project.New(project.Deps{Pool: r.pool}).Cascade(), authorizerOn(r.pool),
````
````new server/internal/bootstrap/interleaving_deletion_test.go
	return workspaceapp.NewDeleteWorkspace(workspaces, project.NewCascade(project.CascadeDeps{Pool: r.pool}), authorizerOn(r.pool),
````

`server/internal/bootstrap/interleaving_endings_test.go`（修改，2 处）：

````old server/internal/bootstrap/interleaving_endings_test.go
		project.New(project.Deps{Pool: r.pool}).Cascade(), authorizerOn(r.pool), postgres.NewTxManager(r.pool, 2*time.Second), clock.System{}).
````
````new server/internal/bootstrap/interleaving_endings_test.go
		project.NewCascade(project.CascadeDeps{Pool: r.pool}), authorizerOn(r.pool), postgres.NewTxManager(r.pool, 2*time.Second), clock.System{}).
````

````old server/internal/bootstrap/interleaving_endings_test.go
		project.New(project.Deps{Pool: r.pool}).Cascade(), r.authorizer(), postgres.NewTxManager(r.pool, 2*time.Second), clock.System{}).
````
````new server/internal/bootstrap/interleaving_endings_test.go
		project.NewCascade(project.CascadeDeps{Pool: r.pool}), r.authorizer(), postgres.NewTxManager(r.pool, 2*time.Second), clock.System{}).
````

`server/internal/bootstrap/interleaving_growth_test.go`（修改，1 处）：

````old server/internal/bootstrap/interleaving_growth_test.go
					g, cascade := newGate(), project.New(project.Deps{Pool: r.pool}).Cascade()
````
````new server/internal/bootstrap/interleaving_growth_test.go
					g, cascade := newGate(), project.NewCascade(project.CascadeDeps{Pool: r.pool})
````

`server/internal/bootstrap/interleaving_leaving_test.go`（修改，1 处）：

````old server/internal/bootstrap/interleaving_leaving_test.go
		project.New(project.Deps{Pool: w.pool}).Cascade(), authorizerOn(w.pool), postgres.NewTxManager(w.pool, 2*time.Second), clock.System{}).
````
````new server/internal/bootstrap/interleaving_leaving_test.go
		project.NewCascade(project.CascadeDeps{Pool: w.pool}), authorizerOn(w.pool), postgres.NewTxManager(w.pool, 2*time.Second), clock.System{}).
````

`server/internal/bootstrap/interleaving_removal_test.go`（修改，1 处）：

````old server/internal/bootstrap/interleaving_removal_test.go
		project.New(project.Deps{Pool: r.pool}).Cascade(), authorizerOn(r.pool), postgres.NewTxManager(r.pool, 2*time.Second), clock.System{}).
````
````new server/internal/bootstrap/interleaving_removal_test.go
		project.NewCascade(project.CascadeDeps{Pool: r.pool}), authorizerOn(r.pool), postgres.NewTxManager(r.pool, 2*time.Second), clock.System{}).
````

`server/internal/bootstrap/interleaving_roles_test.go`（修改，1 处）：

````old server/internal/bootstrap/interleaving_roles_test.go
	return workspaceapp.NewUpdateWorkspaceMember(members, project.New(project.Deps{Pool: r.pool}).Cascade(),
````
````new server/internal/bootstrap/interleaving_roles_test.go
	return workspaceapp.NewUpdateWorkspaceMember(members, project.NewCascade(project.CascadeDeps{Pool: r.pool}),
````

- [ ] **Step 4: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/project/... ./internal/bootstrap/`
Expected: 全部 `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 5: 提交**

```bash
git add server/internal/bootstrap/interleaving_answers_test.go server/internal/bootstrap/interleaving_deletion_test.go server/internal/bootstrap/interleaving_endings_test.go server/internal/bootstrap/interleaving_growth_test.go server/internal/bootstrap/interleaving_leaving_test.go server/internal/bootstrap/interleaving_removal_test.go server/internal/bootstrap/interleaving_roles_test.go server/internal/modules/project/adapter/postgres/end_test.go server/internal/modules/project/module.go
```
```bash
git commit -m "feat(M3/P6): project.NewCascade builds the project module's cascade alone, and project.New builds it through it

NewCascade builds the cascade on the pool, for a composition that ends
memberships without the module's HTTP side. LockActiveMemberProjects'
id order is checked across two workspaces, and the interleaving tests
build the cascade through NewCascade, not through a project.New given
its pool alone.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A；`mutants_p6.py` 的编号，"层"是它被发现的每一层；端到端是单独运行的故事）：

| 变异 | 改坏 | 必须失败的测试 | 层 |
|---|---|---|---|
| `o-prj-per-workspace` | `LockActiveMemberProjects` 一个工作区接一个工作区地锁 | `TestLockActiveMemberProjectsLocksInIDOrder`、`TestEachLockOfADeactivationIsItsStrength`（Task 6 起） | 存储；组合 |
| `o-prj-unordered` | `LockActiveMemberProjects` 去掉 `ORDER BY` | `TestEndingAMembersProjectMemberships`、`TestLockActiveMemberProjectsLocksInIDOrder`、`TestEachLockOfADeactivationIsItsStrength`（Task 6 起） | 存储；组合 |
| `o-prj-share` | `LockActiveMemberProjects` 锁成 `FOR SHARE` | `TestEndingAMembersProjectMemberships`、`TestLockActiveMemberProjectsLocksInIDOrder`、`TestEachLockOfADeactivationIsItsStrength`（Task 6 起）、`TestEachLockOfAnEndingIsItsStrength` | 存储；组合 |
| `o-prj-update` | `LockActiveMemberProjects` 锁成 `FOR UPDATE` | `TestEndingAMembersProjectMemberships`、`TestEachLockOfADeactivationIsItsStrength`（Task 6 起）、`TestEachLockOfAnEndingIsItsStrength` | 存储；组合 |
| `o-prj-locked-deleted` | `LockActiveMemberProjects` 去掉 `p.deleted_at IS NULL` | `TestEndingAMembersProjectMemberships`、`TestLockActiveMemberProjectsLeavesOutAProjectDeletedWhileItWaited` | 存储 |
| `s4-newcascade-ends-nothing` | `NewCascade` 建出的连带不结束任何项目成员关系 | `TestADeactivationAndACreationHeLeadsSerialize`（Task 8 起）、`TestADeactivationAndTheProjectSidesGrowthSerialize`（Task 8 起）、`TestADeactivationEndsEveryMembership`（Task 5 起）、`TestADeactivationFindsWhatChangedMeanwhile`（Task 6 起） 等 22 个 | 组合 |
| `s20-prj-admin-any` | 项目的 `SoleAdmin`（跨工作区调用）的另一位管理员改成"问到的任何项目的" | `TestSoleAdmin`、`TestALeavingAndAnEndingOfAWorkspaceMembershipSerialize`、`TestARefusedDeactivationChangesNothing`（Task 5 起）、`TestAnEndingEndsTheMembershipsAndLeavesNoInvitation` | 存储；组合 |
| `s20-prj-other-any` | 项目的 `SoleAdmin` 的别的成员改成"问到的任何项目的" | `TestSoleAdmin`、`TestADeactivationEndsEveryMembership`（Task 5 起）、`TestADeactivationFindsWhatChangedMeanwhile`（Task 6 起）、`TestADeactivationReadsTheAddressUnderItsLock`（Task 6 起）、`TestADeactivationRefusedAtItsCommitChangesNothing`（Task 5 起） 等 9 个 | 存储；组合 |
| `s24-prj-ended-admin-counts` | 项目的 `SoleAdmin` 把已结束的管理员算作另一位 | `TestSoleAdmin`、`TestALeavingAndAnEndingOfAWorkspaceMembershipSerialize`、`TestARefusedDeactivationChangesNothing`（Task 5 起）、`TestAnEndingEndsTheMembershipsAndLeavesNoInvitation` | 存储；组合 |
| `s37-archived-left` | `LockActiveMemberProjects` 去掉已归档的项目 | `TestEndingAMembersProjectMemberships`、`TestADeactivationEndsEveryMembership`（Task 5 起）、`TestADeactivationFindsWhatChangedMeanwhile`（Task 6 起）、`TestADeactivationReadsTheAddressUnderItsLock`（Task 6 起） | 存储；组合 |

**Done when:** `grep -rn "project.New(project.Deps{Pool: r.pool})\|project.New(project.Deps{Pool: w.pool})" server/internal/bootstrap/` 没有输出；`project.New` 的 `cascade` 来自 `NewCascade`；`TestLockActiveMemberProjectsLocksInIDOrder` 跨两个工作区通过；`make lint-go`、`make test` 通过。

### Task 3: `workspace` 的 `Deactivator`；两个 `sole_admin` 的说明

**Files:**
- Create: `server/internal/modules/workspace/app/deactivate_memberships.go`、`server/internal/modules/workspace/app/deactivate_memberships_test.go`、`server/internal/modules/workspace/app/fakes_deactivation_test.go`
- Modify: `server/internal/modules/project/adapter/http/member_writes_test.go`、`server/internal/modules/project/domain/errors.go`、`server/internal/modules/workspace/adapter/http/members_test.go`、`server/internal/modules/workspace/app/clock_test.go`、`server/internal/modules/workspace/app/fakes_workspaces_test.go`、`server/internal/modules/workspace/app/ports.go`、`server/internal/modules/workspace/app/remove_member_test.go`、`server/internal/modules/workspace/domain/errors.go`

**Interfaces:**
- Produces（spec 2.5，M3 设计 3.3、3.6 约定六、3.7 规则 2、3.8、3.9）：`workspace/app.Deactivator`，`NewDeactivator(memberships AllMembershipsEnder, projects ProjectCascade, clock Clock) *Deactivator`，`DeactivateMemberships(ctx, userID, email) error`：在调用者的事务里依次 `LockMemberWorkspaces` → `SoleAdmin`（为真时 `domain.ErrSoleAdmin`）→ `clock.Now()` 一次 → `DeleteInvitationsTo(email, userID, now)` → `EndWorkspaceMemberships(workspaces, userID, userID, now)` → `projects.EndMemberships(workspaces, userID, userID, now)`；每一步的失败、项目一侧的 `project.sole_admin` 原样返回，之后什么都不运行。它是 `identity` 的 `MembershipDeactivator`（Task 4 接上）。
- `workspace.sole_admin`、`project.sole_admin` 的说明改为不说"谁去做"："The workspace (project) would be left without an admin: its only active admin cannot leave it, nor can his membership end while it has other active members. It must first be given another admin, or be deleted."（spec 2.5 的表；对执行 `nerve users deactivate` 的服务器管理员同样成立，清扫 30）。照它改的测试：`workspace/adapter/http/members_test.go`、`project/adapter/http/member_writes_test.go`、`workspace/app/remove_member_test.go`（`project.sole_admin` 的替身照抄新的文字）。
- `ProjectCascade.EndMemberships` 的说明补上停用（`ports.go`）。

**Tests:**（`app/deactivate_memberships_test.go`；`deactivating(f, tx, user)` 在 `fakeTx` 里运行它，`deactivationCalls(user, workspaces...)` 是整个调用记录）
- `TestDeactivateMembershipsEndsEveryMembership`：bob 在 acme 是成员、在 beta 是访客：调用记录是锁 → 问唯一管理员 → 读时钟 → 删除他的地址的邀请 → 结束两个工作区（按 id）→ 项目一步，每个写由 bob、在同一个时刻；他在两处的成员关系结束。carol 已没有有效成员关系：调用照样到达，工作区是空的。
- `TestDeactivateMembershipsRefusesTheOnlyAdmin`：alice 是 acme 唯一的管理员、bob 是成员：`workspace.sole_admin` 本身（`answeredAs`），事务的函数返回的正是它，调用记录停在问唯一管理员，时钟没有读，她的成员关系仍有效。
- `TestDeactivateMembershipsFailsAtEachStep`：锁、问、删除邀请、结束成员关系、项目一步各失败一次，项目一步另答 `project.sole_admin`（`project` 模块的码的替身）：回答的第一个 `*shared.Error` 就是注入的那个（或没有，500），调用记录停在那一步，没有重试。
- `clock_test.go`：`TestEachWriteReadsTheClockUnderItsLock` 加"the deactivation's memberships"一行：时钟在锁和唯一管理员的问之后读一次。
- `fakes_deactivation_test.go`：`fakeWorkspaces` 的四个方法，各记调用、照存储的包装返回 `endErrs` 给的失败；`LockMemberWorkspaces` 按 id 回答他有效的工作区。

- [ ] **Step 1: 两个说明**

`server/internal/modules/workspace/domain/errors.go`（修改，2 处）：

````old server/internal/modules/workspace/domain/errors.go
	// also when he is its only member (M3 design 3.7 rule 1).
````
````new server/internal/modules/workspace/domain/errors.go
	// also when he is its only member (M3 design 3.7 rule 1), and the
	// deactivation of an account that is the only active admin of a
	// workspace with other active members (rule 2, 3.9). Its detail says
	// what must happen, not who does it: it is true for every caller, the
	// server's administrator who runs `nerve users deactivate` too.
````

````old server/internal/modules/workspace/domain/errors.go
		"The workspace would be left without an admin; make another member an admin first.")
````
````new server/internal/modules/workspace/domain/errors.go
		"The workspace would be left without an admin: its only active admin cannot leave it, nor can his membership end while it "+
			"has other active members. It must first be given another admin, or be deleted.")
````

`server/internal/modules/project/domain/errors.go`（修改，2 处）：

````old server/internal/modules/project/domain/errors.go
	// (rule 2). The workspace's removal and leaving declare it too (M3
	// design 5.1). Each who gets it can have the project given another
	// admin, through a workspace admin if no one else, or delete it.
````
````new server/internal/modules/project/domain/errors.go
	// (rule 2). The workspace's removal and leaving and identity's
	// deactivateMe declare it too (M3 design 5.1, 3.9). Its detail says
	// what must happen, not who does it: it is true for every caller, the
	// server's administrator who runs `nerve users deactivate` too, who can
	// do neither himself.
````

````old server/internal/modules/project/domain/errors.go
			"other active members. Give the project another admin first, or delete it.")
````
````new server/internal/modules/project/domain/errors.go
			"other active members. It must first be given another admin, or be deleted.")
````

`server/internal/modules/workspace/adapter/http/members_test.go`（修改，4 处）：

````old server/internal/modules/workspace/adapter/http/members_test.go
func TestRemoveWorkspaceMemberRefusals(t *testing.T) {
	// The project module's ErrSoleAdmin, which this module does not import:
	// its kind, code and detail.
	soleAdmin := shared.NewError(shared.KindConflict, "project.sole_admin",
		"The project would be left without an admin: its only active admin cannot leave it, nor can his membership end while it has "+
			"other active members. Give the project another admin first, or delete it.")
	tests := []struct {
		err    error
		status int
		want   string
````
````new server/internal/modules/workspace/adapter/http/members_test.go
func TestRemoveWorkspaceMemberRefusals(t *testing.T) {
	// The project module's ErrSoleAdmin, which this module does not import:
	// its kind, code and detail.
	soleAdmin := shared.NewError(shared.KindConflict, "project.sole_admin",
		"The project would be left without an admin: its only active admin cannot leave it, nor can his membership end while it has "+
			"other active members. It must first be given another admin, or be deleted.")
	tests := []struct {
		err    error
		status int
		want   string
````

````old server/internal/modules/workspace/adapter/http/members_test.go
			`{"status":409,"code":"workspace.own_membership","title":"Conflict","detail":"You cannot change your own membership."}`},
		{fmt.Errorf("end the member's project memberships: %w", soleAdmin), http.StatusConflict,
			`{"status":409,"code":"project.sole_admin","title":"Conflict","detail":"The project would be left without an admin: its only ` +
				`active admin cannot leave it, nor can his membership end while it has other active members. Give the project another admin ` +
				`first, or delete it."}`},
		{errors.New("the database is gone"), http.StatusInternalServerError,
			`{"status":500,"code":"internal_error","title":"Internal Server Error"}`},
````
````new server/internal/modules/workspace/adapter/http/members_test.go
			`{"status":409,"code":"workspace.own_membership","title":"Conflict","detail":"You cannot change your own membership."}`},
		{fmt.Errorf("end the member's project memberships: %w", soleAdmin), http.StatusConflict,
			`{"status":409,"code":"project.sole_admin","title":"Conflict","detail":"The project would be left without an admin: its only ` +
				`active admin cannot leave it, nor can his membership end while it has other active members. It must first be given ` +
				`another admin, or be deleted."}`},
		{errors.New("the database is gone"), http.StatusInternalServerError,
			`{"status":500,"code":"internal_error","title":"Internal Server Error"}`},
````

````old server/internal/modules/workspace/adapter/http/members_test.go
func TestLeaveWorkspaceRefusals(t *testing.T) {
	// The project module's ErrSoleAdmin, which this module does not import:
	// its kind, code and detail.
	soleAdmin := shared.NewError(shared.KindConflict, "project.sole_admin",
		"The project would be left without an admin: its only active admin cannot leave it, nor can his membership end while it has "+
			"other active members. Give the project another admin first, or delete it.")
	tests := []struct {
		err    error
		status int
		want   string
````
````new server/internal/modules/workspace/adapter/http/members_test.go
func TestLeaveWorkspaceRefusals(t *testing.T) {
	// The project module's ErrSoleAdmin, which this module does not import:
	// its kind, code and detail.
	soleAdmin := shared.NewError(shared.KindConflict, "project.sole_admin",
		"The project would be left without an admin: its only active admin cannot leave it, nor can his membership end while it has "+
			"other active members. It must first be given another admin, or be deleted.")
	tests := []struct {
		err    error
		status int
		want   string
````

````old server/internal/modules/workspace/adapter/http/members_test.go
			`"detail":"The workspace would be left without an admin; make another member an admin first."}`},
		{fmt.Errorf("end the member's project memberships: %w", soleAdmin), http.StatusConflict,
			`{"status":409,"code":"project.sole_admin","title":"Conflict","detail":"The project would be left without an admin: its only ` +
				`active admin cannot leave it, nor can his membership end while it has other active members. Give the project another admin ` +
				`first, or delete it."}`},
		{errors.New("the database is gone"), http.StatusInternalServerError,
			`{"status":500,"code":"internal_error","title":"Internal Server Error"}`},
````
````new server/internal/modules/workspace/adapter/http/members_test.go
			`"detail":"The workspace would be left without an admin: its only active admin cannot leave it, nor can his membership end ` +
			`while it has other active members. It must first be given another admin, or be deleted."}`},
		{fmt.Errorf("end the member's project memberships: %w", soleAdmin), http.StatusConflict,
			`{"status":409,"code":"project.sole_admin","title":"Conflict","detail":"The project would be left without an admin: its only ` +
				`active admin cannot leave it, nor can his membership end while it has other active members. It must first be given ` +
				`another admin, or be deleted."}`},
		{errors.New("the database is gone"), http.StatusInternalServerError,
			`{"status":500,"code":"internal_error","title":"Internal Server Error"}`},
````

`server/internal/modules/project/adapter/http/member_writes_test.go`（修改，1 处）：

````old server/internal/modules/project/adapter/http/member_writes_test.go
			`it has other active members. Give the project another admin first, or delete it."}`},
````
````new server/internal/modules/project/adapter/http/member_writes_test.go
			`it has other active members. It must first be given another admin, or be deleted."}`},
````

`server/internal/modules/workspace/app/remove_member_test.go`（修改，1 处）：

````old server/internal/modules/workspace/app/remove_member_test.go
			"other active members. Give the project another admin first, or delete it.")
````
````new server/internal/modules/workspace/app/remove_member_test.go
			"other active members. It must first be given another admin, or be deleted.")
````

- [ ] **Step 2: 用例和它的测试**

`server/internal/modules/workspace/app/ports.go`（修改，1 处）：

````old server/internal/modules/workspace/app/ports.go
	// admin's removal of him (removeWorkspaceMember), or his own leaving
	// (leaveWorkspace). It refuses with project.sole_admin, and ends none,
	// when he is the only active admin of one of them that has other active
	// members (M3 design 3.7 rule 2).
````
````new server/internal/modules/workspace/app/ports.go
	// admin's removal of him (removeWorkspaceMember), his own leaving
	// (leaveWorkspace), or his account's deactivation (Deactivator), which
	// calls it once across every workspace of his. It refuses with
	// project.sole_admin, and ends none, when he is the only active admin
	// of one of them that has other active members (M3 design 3.7 rule 2).
````

`server/internal/modules/workspace/app/deactivate_memberships.go`（新文件，60 行）：

````file server/internal/modules/workspace/app/deactivate_memberships.go
package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
)

// Deactivator ends a deactivated account's memberships: identity's
// MembershipDeactivator (M3 design 3.9), which bootstrap wires.
type Deactivator struct {
	memberships AllMembershipsEnder
	projects    ProjectCascade
	clock       Clock
}

// NewDeactivator returns the Deactivator over memberships and projects,
// the project module's cascade.
func NewDeactivator(memberships AllMembershipsEnder, projects ProjectCascade, clock Clock) *Deactivator {
	return &Deactivator{memberships: memberships, projects: projects, clock: clock}
}

// DeactivateMemberships ends every membership of userID, whose address
// is email as read under his account row's lock, in the transaction ctx
// carries: identity's deactivation began it, holds that row FOR NO KEY
// UPDATE, and calls this last (M3 design 3.6 convention 6, 3.9). It locks
// the undeleted workspaces of which he is an active member FOR NO KEY
// UPDATE in id order, found now: no growth of his set of workspaces can
// commit once his row is held, and one deleted before its lock is left
// out. Were he the only active admin of one that has another active
// member, domain.ErrSoleAdmin, nothing written (3.7 rule 2). Else it reads
// the clock, once, after the last of those locks (3.3), and, at that
// moment and by him: every invitation to email, of any workspace, pending
// or declined, deleted (3.8); his memberships of those workspaces ended;
// then the project cascade's EndMemberships, called once across them,
// which locks his projects in id order and may refuse with
// project.sole_admin. A refusal or failure comes back as itself, and
// identity rolls the whole deactivation back.
func (d *Deactivator) DeactivateMemberships(ctx context.Context, userID uuid.UUID, email string) error {
	workspaces, err := d.memberships.LockMemberWorkspaces(ctx, userID)
	if err != nil {
		return err
	}
	sole, err := d.memberships.SoleAdmin(ctx, workspaces, userID)
	switch {
	case err != nil:
		return err
	case sole:
		return domain.ErrSoleAdmin
	}
	now := d.clock.Now()
	if err := d.memberships.DeleteInvitationsTo(ctx, email, userID, now); err != nil {
		return err
	}
	if err := d.memberships.EndWorkspaceMemberships(ctx, workspaces, userID, userID, now); err != nil {
		return err
	}
	return d.projects.EndMemberships(ctx, workspaces, userID, userID, now)
}
````

`server/internal/modules/workspace/app/fakes_workspaces_test.go`（修改，1 处）：

````old server/internal/modules/workspace/app/fakes_workspaces_test.go
	endErrs     map[string]error                  // by method, for HasOtherAdmin, DeletePendingInvitations and EndMember
````
````new server/internal/modules/workspace/app/fakes_workspaces_test.go
	endErrs     map[string]error                  // by method, for the endings' and the deactivation's statements
````

`server/internal/modules/workspace/app/fakes_deactivation_test.go`（新文件，74 行）：

````file server/internal/modules/workspace/app/fakes_deactivation_test.go
package app_test

import (
	"context"
	"fmt"
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The deactivation's statements (app.AllMembershipsEnder) over the
// memberships fakeWorkspaces holds: each logs its call and fails with the
// error endErrs sets for it, wrapped as the store wraps it.

// LockMemberWorkspaces answers the workspaces of which userID has an active
// membership, in id order.
func (f *fakeWorkspaces) LockMemberWorkspaces(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	f.log.add(ctx, "LockMemberWorkspaces %s", userID)
	if err := f.endErrs["LockMemberWorkspaces"]; err != nil {
		return nil, fmt.Errorf("lock the member's workspaces: %w", err)
	}
	var ids []uuid.UUID
	for workspace, list := range f.memberships {
		if slices.ContainsFunc(list, func(m domain.Membership) bool { return m.MemberID == userID && m.IsActive }) {
			ids = append(ids, workspace)
		}
	}
	slices.SortFunc(ids, uuid.UUID.Compare)
	return ids, nil
}

// SoleAdmin answers whether userID is the only active admin of one of the
// workspaces that has another active member.
func (f *fakeWorkspaces) SoleAdmin(ctx context.Context, workspaceIDs []uuid.UUID, userID uuid.UUID) (bool, error) {
	f.log.add(ctx, "SoleAdmin %v %s", workspaceIDs, userID)
	if err := f.endErrs["SoleAdmin"]; err != nil {
		return false, fmt.Errorf("look for a workspace he is the only admin of: %w", err)
	}
	return slices.ContainsFunc(workspaceIDs, func(w uuid.UUID) bool {
		list := f.memberships[w]
		admin := func(m domain.Membership) bool { return m.Role == shared.RoleAdmin && m.IsActive }
		other := func(m domain.Membership) bool { return m.MemberID != userID && m.IsActive }
		return slices.ContainsFunc(list, func(m domain.Membership) bool { return m.MemberID == userID && admin(m) }) &&
			!slices.ContainsFunc(list, func(m domain.Membership) bool { return other(m) && admin(m) }) && slices.ContainsFunc(list, other)
	}), nil
}

func (f *fakeWorkspaces) DeleteInvitationsTo(ctx context.Context, email string, by uuid.UUID, now time.Time) error {
	f.log.add(ctx, "DeleteInvitationsTo %s by %s at %s", email, by, now.Format(time.RFC3339Nano))
	if err := f.endErrs["DeleteInvitationsTo"]; err != nil {
		return fmt.Errorf("delete the invitations to the address: %w", err)
	}
	return nil
}

// EndWorkspaceMemberships also ends the memberships the fake holds, as the
// store ends the rows.
func (f *fakeWorkspaces) EndWorkspaceMemberships(ctx context.Context, workspaceIDs []uuid.UUID, userID, by uuid.UUID, now time.Time) error {
	f.log.add(ctx, "EndWorkspaceMemberships %v %s by %s at %s", workspaceIDs, userID, by, now.Format(time.RFC3339Nano))
	if err := f.endErrs["EndWorkspaceMemberships"]; err != nil {
		return fmt.Errorf("end the workspace memberships: %w", err)
	}
	for _, w := range workspaceIDs {
		for i, m := range f.memberships[w] {
			if m.MemberID == userID {
				f.memberships[w][i].IsActive = false
			}
		}
	}
	return nil
}
````

`server/internal/modules/workspace/app/deactivate_memberships_test.go`（新文件，134 行）：

````file server/internal/modules/workspace/app/deactivate_memberships_test.go
package app_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// deactivating runs the Deactivator over f's fakes, with the clock logging
// its reads among the calls, in tx, as identity's deactivation does: it is
// the last step of that transaction (M3 design 3.9). user is the account
// deactivated, at his address.
func deactivating(f *membersFixture, tx *fakeTx, user app.AccountState) error {
	d := app.NewDeactivator(f.workspaces, f.projects, clockAt{clockNow, f.log})
	return tx.WithinTx(context.Background(), func(ctx context.Context) error { return d.DeactivateMemberships(ctx, user.ID, user.Email) })
}

// deactivationCalls are the calls of the ending of user's memberships of
// workspaces, in id order: those workspaces locked, the only admin asked,
// the clock read, every invitation to his address deleted, his memberships
// of the workspaces ended, then the project cascade's step, called once
// across them; each write by him at the clock's one time.
func deactivationCalls(user app.AccountState, workspaces ...uuid.UUID) []string {
	at := clockNow.Format(time.RFC3339Nano)
	return []string{
		"LockMemberWorkspaces " + user.ID.String(),
		fmt.Sprintf("SoleAdmin %v %s", workspaces, user.ID),
		"Now",
		fmt.Sprintf("DeleteInvitationsTo %s by %s at %s", user.Email, user.ID, at),
		fmt.Sprintf("EndWorkspaceMemberships %v %s by %s at %s", workspaces, user.ID, user.ID, at),
		fmt.Sprintf("EndMemberships %v %s by %s at %s", workspaces, user.ID, user.ID, at),
	}
}

// The Deactivator ends bob's memberships of acme, a member's, and of beta,
// a guest's, the two workspaces whose active member he is, in their id
// order; the invitations to his address and his project memberships with
// them, by him at the clock's one time, read after the workspaces' locks
// and the check of the only admin (M3 design 3.3, 3.6 convention 6, 3.9),
// all in the transaction it runs in. carol, whose membership of acme
// ended, has none: her workspaces are none, and the calls come all the
// same, over none, so that the invitations to her address go too.
func TestDeactivateMembershipsEndsEveryMembership(t *testing.T) {
	for _, tt := range []struct {
		user       app.AccountState
		workspaces []uuid.UUID
	}{{bob, []uuid.UUID{acme.ID, beta.ID}}, {carol, nil}} {
		f, tx := newMembers(), &fakeTx{}

		err := deactivating(f, tx, tt.user)

		if want := deactivationCalls(tt.user, tt.workspaces...); err != nil || !slices.Equal(f.log.calls, want) {
			t.Errorf("%s: DeactivateMemberships() = %v, calls\n%q\nwant nil,\n%q", tt.user.Email, err, f.log.calls, want)
		}
		for _, w := range tt.workspaces {
			if m, found, err := f.workspaces.MemberOf(context.Background(), w, tt.user.ID); err != nil || !found || m.IsActive {
				t.Errorf("%s's membership of %s after the deactivation: %+v, %v, %v; want it ended", tt.user.Email, w, m, found, err)
			}
		}
	}
}

// The only active admin of a workspace that has another active member
// is refused: alice, acme's, with bob its member, gets
// workspace.sole_admin itself, the 409 of the contract (M3 design 3.7 rule
// 2), before the clock is read and before any write; her membership stays.
func TestDeactivateMembershipsRefusesTheOnlyAdmin(t *testing.T) {
	f, tx := newMembers(), &fakeTx{}

	err := deactivating(f, tx, alice)

	if !errors.Is(err, domain.ErrSoleAdmin) || !tx.answered(err) {
		t.Errorf("DeactivateMemberships() = %v, the transaction's function returned %v; want %v from it", err, tx.returned, domain.ErrSoleAdmin)
	}
	answeredAs(t, "the only admin", err, domain.ErrSoleAdmin)
	if want := deactivationCalls(alice, acme.ID)[:2]; !slices.Equal(f.log.calls, want) {
		t.Errorf("calls\n%q\nwant\n%q", f.log.calls, want)
	}
	if m, _, _ := f.workspaces.MemberOf(context.Background(), acme.ID, alice.ID); !m.IsActive {
		t.Error("alice's membership of acme after the refusal: ended; want it active")
	}
}

// Each failure of a step, and the project cascade's refusal, is the
// answer as it came, the first *shared.Error in its chain the one
// injected or none (a 500), out of the transaction as itself; no step
// runs after it, none is tried again.
func TestDeactivateMembershipsFailsAtEachStep(t *testing.T) {
	failure := errors.New("connection reset")
	// The project module's ErrSoleAdmin, which this module does not import:
	// its kind, code and detail.
	soleAdmin := shared.NewError(shared.KindConflict, "project.sole_admin",
		"The project would be left without an admin: its only active admin cannot leave it, nor can his membership end while it has "+
			"other active members. It must first be given another admin, or be deleted.")
	calls := deactivationCalls(bob, acme.ID, beta.ID)
	for _, tt := range []struct {
		name  string
		set   func(f *membersFixture)
		want  error
		calls []string
	}{
		{"the lock", func(f *membersFixture) { f.workspaces.endErrs = map[string]error{"LockMemberWorkspaces": failure} }, failure, calls[:1]},
		{"the check", func(f *membersFixture) { f.workspaces.endErrs = map[string]error{"SoleAdmin": failure} }, failure, calls[:2]},
		{"the invitations", func(f *membersFixture) { f.workspaces.endErrs = map[string]error{"DeleteInvitationsTo": failure} }, failure,
			calls[:4]},
		{"the memberships", func(f *membersFixture) { f.workspaces.endErrs = map[string]error{"EndWorkspaceMemberships": failure} }, failure,
			calls[:5]},
		{"the projects' step", func(f *membersFixture) { f.projects.errs = map[string]error{"EndMemberships": failure} }, failure, calls},
		{"the only admin of a project", func(f *membersFixture) { f.projects.errs = map[string]error{"EndMemberships": soleAdmin} }, soleAdmin,
			calls},
	} {
		f, tx := newMembers(), &fakeTx{}
		tt.set(f)

		err := deactivating(f, tx, bob)

		if !errors.Is(err, tt.want) || !tx.answered(err) {
			t.Errorf("%s failing: DeactivateMemberships() = %v, the transaction's function returned %v; want %v from it", tt.name, err,
				tx.returned, tt.want)
		}
		answeredAs(t, tt.name, err, tt.want)
		if !slices.Equal(f.log.calls, tt.calls) {
			t.Errorf("%s failing: calls\n%q\nwant\n%q", tt.name, f.log.calls, tt.calls)
		}
	}
}
````

`server/internal/modules/workspace/app/clock_test.go`（修改，2 处）：

````old server/internal/modules/workspace/app/clock_test.go
// workspace's lock and the membership's read.
````
````new server/internal/modules/workspace/app/clock_test.go
// workspace's lock and the membership's read; the deactivation's
// memberships, after every workspace's lock and the check of the only
// admin.
````

````old server/internal/modules/workspace/app/clock_test.go
			"CountInactive "+acme.ID.String()+" "+carol.ID.String())},
````
````new server/internal/modules/workspace/app/clock_test.go
			"CountInactive "+acme.ID.String()+" "+carol.ID.String())},
		{"the deactivation's memberships", func() ([]string, error) {
			f := newMembers()
			err := deactivating(f, &fakeTx{}, bob)
			return f.log.calls, err
		}, deactivationCalls(bob, acme.ID, beta.ID)},
````

- [ ] **Step 3: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/workspace/... ./internal/modules/project/...`
Expected: 全部 `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`（`Deactivator` 这时只有测试使用，Task 4 接上）

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 4: 提交**

```bash
git add server/internal/modules/project/adapter/http/member_writes_test.go server/internal/modules/project/domain/errors.go server/internal/modules/workspace/adapter/http/members_test.go server/internal/modules/workspace/app/clock_test.go server/internal/modules/workspace/app/deactivate_memberships.go server/internal/modules/workspace/app/deactivate_memberships_test.go server/internal/modules/workspace/app/fakes_deactivation_test.go server/internal/modules/workspace/app/fakes_workspaces_test.go server/internal/modules/workspace/app/ports.go server/internal/modules/workspace/app/remove_member_test.go server/internal/modules/workspace/domain/errors.go
```
```bash
git commit -m "feat(M3/P6): the workspace module's Deactivator ends every membership of a deactivated account

Under the account's row lock, it locks his workspaces in id order,
refuses with workspace.sole_admin when he is the only active admin of
one with other active members, reads the clock once after those
locks, deletes every invitation to his address, ends his workspace
memberships and calls the project cascade's EndMemberships once across
the workspaces, each write by him at that moment. The two sole_admin
details now say what must happen, not who does it, so that they hold
for the server's administrator who runs nerve users deactivate too.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A；`mutants_p6.py` 的编号，"层"是它被发现的每一层；端到端是单独运行的故事）：

| 变异 | 改坏 | 必须失败的测试 | 层 |
|---|---|---|---|
| `o-clock-early` | 时钟在锁工作区之前读 | `TestDeactivateMembershipsEndsEveryMembership`、`TestDeactivateMembershipsFailsAtEachStep`、`TestDeactivateMembershipsRefusesTheOnlyAdmin`、`TestEachWriteReadsTheClockUnderItsLock`、`TestADeactivationAndTheProjectSidesGrowthSerialize`（Task 8 起）、`TestEachLockOfADeactivationIsItsStrength`（Task 6 起） | 单元；组合 |
| `o-projects-first` | 项目一步挪到删除邀请和结束工作区成员关系之前 | `TestDeactivateMembershipsEndsEveryMembership`、`TestDeactivateMembershipsFailsAtEachStep`、`TestEachWriteReadsTheClockUnderItsLock`、`TestEachLockOfADeactivationIsItsStrength`（Task 6 起） | 单元；组合 |
| `s2-lock-swallowed` | 锁的失败被吞掉，在空集合上继续 | `TestDeactivateMembershipsFailsAtEachStep` | 单元 |
| `s2-sole-swallowed` | 检查的失败答成完成 | `TestDeactivateMembershipsFailsAtEachStep` | 单元 |
| `s2-sole-failure-404` | 检查的失败答成 `workspace.not_found` | `TestDeactivateMembershipsFailsAtEachStep` | 单元 |
| `s2-invitations-swallowed` | 删除邀请的失败被吞掉 | `TestDeactivateMembershipsFailsAtEachStep` | 单元 |
| `s2-end-swallowed` | 结束工作区成员关系的失败被吞掉 | `TestDeactivateMembershipsFailsAtEachStep` | 单元 |
| `s2-projects-swallowed` | 项目一步的失败、拒绝答成完成 | `TestDeactivateMembershipsFailsAtEachStep`、`TestARefusedDeactivationChangesNothing`（Task 5 起）、W9（Task 9 起） | 单元；组合；端到端 |
| `s2-projects-retried` | 项目一步失败之后再试一次 | `TestDeactivateMembershipsFailsAtEachStep` | 单元 |
| `s6-ws-detail` | `workspace.sole_admin` 的说明叫调用者去做（"Make another member an admin first."） | `TestLeaveWorkspaceRefusals`、`TestARefusedDeactivationChangesNothing`（Task 5 起）、`TestUsersCommandsFail`（Task 5 起）、W9（Task 9 起） | 单元；组合；端到端 |
| `s6-prj-detail` | `project.sole_admin` 的说明叫调用者去做（P5b 的原句） | `TestLeaveProject`、`TestARefusedDeactivationChangesNothing`（Task 5 起）、W9（Task 9 起） | 单元；组合；端到端 |
| `s6-two-moments` | 工作区成员关系的结束另读一次时钟 | `TestDeactivateMembershipsEndsEveryMembership`、`TestDeactivateMembershipsFailsAtEachStep`、`TestEachWriteReadsTheClockUnderItsLock`、`TestADeactivationEndsEveryMembership`（Task 5 起）、`TestDecliningAndDeactivating`（Task 7 起）、W9（Task 9 起） | 单元；组合；端到端 |
| `s9-end-before-invitations` | 先结束工作区成员关系、再删除邀请（全局顺序反了） | `TestDeactivateMembershipsEndsEveryMembership`、`TestDeactivateMembershipsFailsAtEachStep`、`TestEachWriteReadsTheClockUnderItsLock`、`TestEachLockOfADeactivationIsItsStrength`（Task 6 起） | 单元；组合 |
| `s9-check-after-end` | 规则 2 在结束工作区成员关系之后检查 | `TestDeactivateMembershipsEndsEveryMembership`、`TestDeactivateMembershipsFailsAtEachStep`、`TestDeactivateMembershipsRefusesTheOnlyAdmin`、`TestEachWriteReadsTheClockUnderItsLock`、`TestARefusedDeactivationChangesNothing`（Task 5 起）、`TestAnAdmittedAdminsWriteAndHisDeactivation`（Task 7 起）、`TestTwoAdminsDeactivatedAtOnceLeaveAnAdmin`（Task 8 起）、`TestUsersCommandsFail`（Task 5 起）、W9（Task 9 起） | 单元；组合；端到端 |
| `s11-refusal-after-invitations` | 规则 2 的拒绝在删除邀请之后 | `TestDeactivateMembershipsEndsEveryMembership`、`TestDeactivateMembershipsFailsAtEachStep`、`TestDeactivateMembershipsRefusesTheOnlyAdmin`、`TestEachWriteReadsTheClockUnderItsLock` | 单元 |
| `s16-workspaces-reversed` | `Deactivator` 把工作区的顺序倒过来 | `TestDeactivateMembershipsEndsEveryMembership`、`TestDeactivateMembershipsFailsAtEachStep`、`TestEachWriteReadsTheClockUnderItsLock` | 单元 |
| `s21-ws-joined-404` | 工作区的拒绝与 `workspace.not_found` 一起 `errors.Join` | `TestDeactivateMembershipsRefusesTheOnlyAdmin`、`TestARefusedDeactivationChangesNothing`（Task 5 起）、`TestAnAdmittedAdminsWriteAndHisDeactivation`（Task 7 起）、`TestTwoAdminsDeactivatedAtOnceLeaveAnAdmin`（Task 8 起）、`TestUsersCommandsFail`（Task 5 起） | 单元；组合 |
| `s24-projects-first-workspace` | 项目一步只传他的第一个工作区 | `TestDeactivateMembershipsEndsEveryMembership`、`TestDeactivateMembershipsFailsAtEachStep`、`TestEachWriteReadsTheClockUnderItsLock`、`TestADeactivationEndsEveryMembership`（Task 5 起）、`TestADeactivationFindsWhatChangedMeanwhile`（Task 6 起）、`TestADeactivationReadsTheAddressUnderItsLock`（Task 6 起）、`TestARefusedDeactivationChangesNothing`（Task 5 起） 等 6 个、W9（Task 9 起） | 单元；组合；端到端 |
| `s24-check-first-workspace` | 规则 2 只查他的第一个工作区 | `TestDeactivateMembershipsEndsEveryMembership`、`TestDeactivateMembershipsFailsAtEachStep`、`TestEachWriteReadsTheClockUnderItsLock`、`TestARefusedDeactivationChangesNothing`（Task 5 起） | 单元；组合 |

**Done when:** `Deactivator` 的调用记录、拒绝、每一步的失败由单元测试钉住；`TestEachWriteReadsTheClockUnderItsLock` 有停用一行；`grep -rn "make another member an admin first\|Give the project another admin first" server/` 没有输出；`make lint-go`、`make test` 通过。

### Task 4: `identity` 的端口和锁下的邮箱；两个码；`workspace.NewDeactivator`；两处组合；交错 8、19 换成真实的停用

**Files:**
- Create: `server/internal/bootstrap/deactivating_test.go`、`server/internal/modules/workspace/deactivator.go`
- Modify: `api/modules/identity.yaml`、`server/internal/archtest/composition_test.go`、`server/internal/bootstrap/app.go`、`server/internal/bootstrap/interleaving_answers_test.go`、`server/internal/bootstrap/interleaving_test.go`、`server/internal/bootstrap/users.go`、`server/internal/modules/identity/adapter/http/me_test.go`、`server/internal/modules/identity/adapter/postgres/credentials_test.go`、`server/internal/modules/identity/adapter/postgres/queries/users.sql`、`server/internal/modules/identity/adapter/postgres/users.go`、`server/internal/modules/identity/admin.go`、`server/internal/modules/identity/app/deactivate.go`、`server/internal/modules/identity/app/ports.go`、`server/internal/modules/identity/module.go`、`server/internal/modules/workspace/module.go`
- Modify（完整内容）: `server/internal/modules/identity/app/deactivate_test.go`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/identity/adapter/postgres/gen/users.sql.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Produces（spec 2.6，M3 设计 3.9、6.2、6.6、9.4）：
  - `identity/app.LockedAccount.Email`（锁下的地址；`LockUserForCredentials` 多读 `email`）；端口 `identity/app.MembershipDeactivator`（`DeactivateMemberships(ctx, userID, email) error`，不带 `now`：spec 第 3 节第 1 条）；`DeactivateDeps.Memberships`、`identity.Deps.Memberships`、`identity.AdminDeps.Memberships`。
  - `deactivate(ctx, id, email, now)`：账户、新手引导、会话之后调用 `Memberships.DeactivateMemberships(ctx, id, email)`；它的拒绝或失败原样返回，整个停用回滚，不记日志。`Execute` 传 `Lock` 回答的 `account.Email`；`ExecuteByEmail` 传它锁住那一行用的规范化地址。
  - `deactivateMe`：`x-problem-codes: [workspace.sole_admin, project.sole_admin]`，描述写明删除全部邀请、结束全部成员关系、两个拒绝什么都不改、回来的两步。
  - `workspace.Deactivator`（接口）、`workspace.DeactivatorDeps{Pool, Clock, Projects}`、`workspace.NewDeactivator(DeactivatorDeps) Deactivator`（`workspace/deactivator.go`）；`workspace.New` 经它建出自己的，`Module.Deactivator()` 交出（spec 第 3 节第 2 条）。
  - 组合：`bootstrap/app.go` 给 `identity.New` 的 `Memberships: ws.Deactivator()`；`bootstrap/users.go` 给 `identity.NewAdmin` 的 `Memberships: workspace.NewDeactivator(…{Pool: pool, Clock: clock.System{}, Projects: project.NewCascade(project.CascadeDeps{Pool: pool})})`。
  - `bootstrap/deactivating_test.go`：`deactivating(pool, sessions, memberships)`，`nerve users deactivate` 照 `users.go` 的接法，会话和成员关系两处可以换成被 gate 停住的；`endedHoldingAll`（停在结束工作区成员关系之后、项目一步之前）。
- Consumes：Task 1–3。

**Tests:**
- `identity/app/deactivate_test.go`（完整内容）：`fakeMemberships` 记下账户和地址；`TestDeactivate` 的调用记录在会话之后以锁下的地址 `alice@locked.example`（别处得不到的地址）调用成员关系一步；`TestDeactivateWhenAWriteFails`、`TestDeactivateByEmailWhenAWriteFails` 加 `membershipsErrors`（两个 409 的替身和一个失败）：原样返回、第一个 `*shared.Error` 不变、一个事务、不记日志；`TestDeactivateByEmail` 以规范化的地址调用。
- `identity/adapter/http/me_test.go`：`TestDeactivateMeProblems`：401、`workspace.sole_admin`、包装过的 `project.sole_admin`（两个替身带它们的类别、码、说明）、失败 500，回答逐字；`identity` 的 `apitest.Main` 两个方向通过（9.4）。
- `identity/adapter/postgres/credentials_test.go`：`TestLockForCredentialsReadsTheRow` 读出 alice 的地址，先存的 bob 的不是。
- `archtest/composition_test.go`：`TestCommandsComposeNoServerAndNoJobs`：`Users` 到达 `identity.NewAdmin`、`workspace.NewDeactivator`、`project.NewCascade`，到达不了 `workspace.NewAdmin`；`Workspaces` 到达 `workspace.NewAdmin`，到达不了 `NewDeactivator`、`NewCascade`。
- 交错 8（`bootstrap/interleaving_test.go` 的 `TestDeactivationFirstRefusesTheWorkspace`、`TestCreationFirstHoldsOffTheDeactivation`）：停用一方是 `deactivating`；创建在先时停用之后结束新工作区的成员关系（他是唯一的成员，规则 2 不拒绝，P1 review 第 6 节"成员关系一半"）。
- 交错 19（`interleaving_answers_test.go` 的 `TestDecliningAndDeactivating`）：P3 的 `gatedDeactivation` 删除，停用一方是 `deactivating`（停在 `endedHoldingAll`）；两种顺序都由停用最后写邀请和他的成员关系，由他、在一个时刻；没有 40P01。

- [ ] **Step 1: 生成的输入：锁下的邮箱、`deactivateMe` 的描述和两个码**

`server/internal/modules/identity/adapter/postgres/queries/users.sql`（修改，1 处）：

````old server/internal/modules/identity/adapter/postgres/queries/users.sql
-- the account does not wait.
SELECT password, is_active
````
````new server/internal/modules/identity/adapter/postgres/queries/users.sql
-- the account does not wait. The address is the one under the lock: the deactivation deletes the
-- invitations to it (M3 design 3.9), and a change of address commits before the lock or after it.
SELECT password, is_active, email
````

`api/modules/identity.yaml`（修改，2 处）：

````old api/modules/identity.yaml
        The password and the personal access tokens stay, but nothing
````
````new api/modules/identity.yaml
        Every invitation to the account's address, pending or declined, is
        deleted, and every membership of the account ends, of a workspace
        or of a project, each row and its role kept: all at one moment, in
        the same transaction. Were the account the only active admin of a
        workspace that has other active members, workspace.sole_admin; of
        such a project, project.sole_admin; and nothing changes. The
        password and the personal access tokens stay, but nothing
````

````old api/modules/identity.yaml
        personal access tokens authenticate again.
      security: [{bearer: []}]
      x-problem-codes: []
      responses:
````
````new api/modules/identity.yaml
        personal access tokens authenticate again. Its memberships stay
        ended: the administrator's `nerve workspaces reactivate-member`
        restores one workspace's at a time.
      security: [{bearer: []}]
      x-problem-codes: [workspace.sole_admin, project.sole_admin]
      responses:
````

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `7532d8c602d87ba6777d144ed3738a943404aaf88d51b5a163d32d88bd14f0eb` | 2500 | `api/dist/openapi.yaml` |
| `83ed69349dac0b56e835bd4d215ff97e62c46b2b4529375aa956aca05e4801ed` | 366 | `server/internal/modules/identity/adapter/postgres/gen/users.sql.go` |
| `209743a7cd55af0dd6b274360bad10a811a59d4d54479d21e98259d94d907676` | 2752 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 server/internal/modules/identity/adapter/postgres/gen/users.sql.go api/dist/openapi.yaml web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 2: 存储**

`server/internal/modules/identity/adapter/postgres/users.go`（修改，1 处）：

````old server/internal/modules/identity/adapter/postgres/users.go
	return app.LockedAccount{PasswordHash: row.Password, Active: row.IsActive}, nil
````
````new server/internal/modules/identity/adapter/postgres/users.go
	return app.LockedAccount{PasswordHash: row.Password, Active: row.IsActive, Email: row.Email}, nil
````

`server/internal/modules/identity/adapter/postgres/credentials_test.go`（修改，2 处）：

````old server/internal/modules/identity/adapter/postgres/credentials_test.go
}

func TestLockForCredentialsReadsTheRow(t *testing.T) {
	s, pool := newStore(t)
````
````new server/internal/modules/identity/adapter/postgres/credentials_test.go
}

// LockForCredentials reads the account's row under its lock: its hash,
// its state and its address, alice's, not bob's, stored before hers.
func TestLockForCredentialsReadsTheRow(t *testing.T) {
	s, pool := newStore(t)
	mustCreate(t, s, newUser("bob@corp.com"))
````

````old server/internal/modules/identity/adapter/postgres/credentials_test.go
	if want := (app.LockedAccount{PasswordHash: u.PasswordHash, Active: true}); err != nil || got != want {
````
````new server/internal/modules/identity/adapter/postgres/credentials_test.go
	if want := (app.LockedAccount{PasswordHash: u.PasswordHash, Active: true, Email: "alice@corp.com"}); err != nil || got != want {
````

- [ ] **Step 3: 端口、用例和 HTTP 测试**

`server/internal/modules/identity/app/ports.go`（修改，3 处）：

````old server/internal/modules/identity/app/ports.go
// LockedAccount is an account's row under the credential lock.
````
````new server/internal/modules/identity/app/ports.go
// LockedAccount is an account's row under the credential lock: its address
// too, as committed before the lock (M3 design 3.9).
````

````old server/internal/modules/identity/app/ports.go
	Active       bool
````
````new server/internal/modules/identity/app/ports.go
	Active       bool
	Email        string
````

````old server/internal/modules/identity/app/ports.go
	DeactivateUser(ctx context.Context, id uuid.UUID, now time.Time) error
````
````new server/internal/modules/identity/app/ports.go
	DeactivateUser(ctx context.Context, id uuid.UUID, now time.Time) error
}

// MembershipDeactivator ends a deactivated account's memberships (M3 design
// 3.9): the workspace module implements it, and bootstrap wires it.
type MembershipDeactivator interface {
	// DeactivateMemberships ends every workspace and project membership of
	// account userID and deletes every invitation to email, its address as
	// read under its row's lock, in the transaction ctx carries, which
	// holds that lock. Its refusal, workspace.sole_admin or
	// project.sole_admin, or its failure comes back as itself, and the
	// whole deactivation rolls back.
	DeactivateMemberships(ctx context.Context, userID uuid.UUID, email string) error
````

`server/internal/modules/identity/app/deactivate.go`（修改，9 处）：

````old server/internal/modules/identity/app/deactivate.go
// ExecuteByEmail with Accounts: a composition sets the one its entry uses.
````
````new server/internal/modules/identity/app/deactivate.go
// ExecuteByEmail with Accounts: a composition sets the one its entry uses.
// Memberships is the workspace module's MembershipDeactivator, which both
// call.
````

````old server/internal/modules/identity/app/deactivate.go
	Lock     CredentialLock
	Accounts AccountLocker
	Users    UserDeactivator
	Profiles OnboardingResetter
	Sessions SessionRevoker
	Tx       shared.TxManager
	Clock    Clock
	Logger   *slog.Logger
````
````new server/internal/modules/identity/app/deactivate.go
	Lock        CredentialLock
	Accounts    AccountLocker
	Users       UserDeactivator
	Profiles    OnboardingResetter
	Sessions    SessionRevoker
	Memberships MembershipDeactivator
	Tx          shared.TxManager
	Clock       Clock
	Logger      *slog.Logger
````

````old server/internal/modules/identity/app/deactivate.go
// then writes as deactivate does.
````
````new server/internal/modules/identity/app/deactivate.go
// then writes as deactivate does, with the address the lock read.
````

````old server/internal/modules/identity/app/deactivate.go
		if _, err := u.d.Lock.Lock(ctx, actor, now); err != nil {
````
````new server/internal/modules/identity/app/deactivate.go
		account, err := u.d.Lock.Lock(ctx, actor, now)
		if err != nil {
````

````old server/internal/modules/identity/app/deactivate.go
		var err error
		revoked, err = u.deactivate(ctx, actor.UserID, now)
````
````new server/internal/modules/identity/app/deactivate.go
		revoked, err = u.deactivate(ctx, actor.UserID, account.Email, now)
````

````old server/internal/modules/identity/app/deactivate.go
// (identity.account_not_found), then writes as deactivate does.
````
````new server/internal/modules/identity/app/deactivate.go
// (identity.account_not_found), then writes as deactivate does, with that
// address, the row's under the lock.
````

````old server/internal/modules/identity/app/deactivate.go
		result.Sessions, err = u.deactivate(ctx, id, now)
````
````new server/internal/modules/identity/app/deactivate.go
		result.Sessions, err = u.deactivate(ctx, id, email, now)
````

````old server/internal/modules/identity/app/deactivate.go
// deactivate writes in the global lock order (M2 design 3.5, 6.4): the
// account inactive, its onboarding started over, every session revoked
// with reason deactivated; it returns how many. The password and the
// personal access tokens stay; authentication refuses the tokens while the
// account is inactive. Call it under the account row lock.
func (u *Deactivate) deactivate(ctx context.Context, id uuid.UUID, now time.Time) (int, error) {
````
````new server/internal/modules/identity/app/deactivate.go
// deactivate writes in the global lock order (M2 design 3.5, 6.4; M3 design
// 3.6): the account inactive, its onboarding started over, every session
// revoked with reason deactivated, then, last, its memberships ended and
// the invitations to email, its address under the lock, deleted (M3 design
// 3.9); it returns how many sessions. The password and the personal access
// tokens stay; authentication refuses the tokens while the account is
// inactive. Call it under the account row lock.
func (u *Deactivate) deactivate(ctx context.Context, id uuid.UUID, email string, now time.Time) (int, error) {
````

````old server/internal/modules/identity/app/deactivate.go
	return u.d.Sessions.RevokeSessions(ctx, id, uuid.Nil(), domain.RevokeDeactivated, now)
````
````new server/internal/modules/identity/app/deactivate.go
	revoked, err := u.d.Sessions.RevokeSessions(ctx, id, uuid.Nil(), domain.RevokeDeactivated, now)
	if err != nil {
		return 0, err
	}
	return revoked, u.d.Memberships.DeactivateMemberships(ctx, id, email)
````

`server/internal/modules/identity/app/deactivate_test.go`（完整内容，232 行）：

````whole server/internal/modules/identity/app/deactivate_test.go
package app_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeMemberships is the workspace module's Deactivator
// (app.MembershipDeactivator): it logs its call, its account and address,
// to the log it shares, and answers err.
type fakeMemberships struct {
	log *callLog
	err error
}

func (f *fakeMemberships) DeactivateMemberships(ctx context.Context, userID uuid.UUID, email string) error {
	f.log.add(ctx, "deactivate the memberships of "+userID.String()+" at "+email)
	return f.err
}

// lockedEmail is the address the credential lock reads under the lock: not
// one the use case could have from anywhere else.
const lockedEmail = "alice@locked.example"

func (f *credentialFixture) deactivate(memberships *fakeMemberships) *app.Deactivate {
	f.creds.account.Email = lockedEmail
	return app.NewDeactivate(app.DeactivateDeps{
		Lock: f.lock(), Users: f.creds, Profiles: f.creds, Sessions: f.creds, Memberships: memberships, Tx: f.tx, Clock: clocktest.At(now),
		Logger: f.logger(),
	})
}

// The administrator's deactivation takes no credential lock: it has no
// caller's credential to check again.
func (f *adminFixture) deactivate(memberships *fakeMemberships) *app.Deactivate {
	return app.NewDeactivate(app.DeactivateDeps{
		Accounts: f.store, Users: f.store, Profiles: f.store, Sessions: f.store, Memberships: memberships, Tx: f.tx, Clock: clocktest.At(now),
		Logger: f.logger(),
	})
}

// One transaction takes the lock and checks the caller's credential again,
// then writes in the global lock order: the account, the profile, the
// sessions, all of them, whatever the credential (M2 design 3.5, 6.4); and
// last the memberships, of the account at the address read under the lock
// (M3 design 3.6, 3.9).
func TestDeactivate(t *testing.T) {
	tests := []struct {
		name       string
		actor      shared.Actor
		credential string
	}{
		{"with a session", sessionActor, "session " + sessionID.String()},
		{"with a token", tokenActor, "token " + tokenID.String()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newCredentialFixture()

			err := f.deactivate(&fakeMemberships{log: f.log}).Execute(shared.WithActor(context.Background(), tt.actor))

			want := []string{
				"lock " + userID.String(), tt.credential, "deactivate " + userID.String(), "reset onboarding of " + userID.String(),
				"revoke deactivated sessions of " + userID.String() + " but " + uuid.Nil().String(),
				"deactivate the memberships of " + userID.String() + " at " + lockedEmail,
			}
			if err != nil || !slices.Equal(f.log.calls, want) || f.tx.calls != 1 || !slices.Equal(f.creds.writtenAt, []time.Time{now, now, now}) {
				t.Errorf("Execute() = %v, calls %q in %d transactions at %v; want %q in one at %v", err, f.log.calls, f.tx.calls, f.creds.writtenAt, want, now)
			}
			logs := f.logs.String()
			if !strings.Contains(logs, `"msg":"account deactivated","user_id":"`+userID.String()+`","revoked_sessions":1,"by":"self"`) {
				t.Errorf("logs = %s, want the deactivation with user_id, the sessions revoked and by self", logs)
			}
		})
	}
}

// A credential that a concurrent reset revoked since authentication
// deactivates nothing (M2 design 3.5).
func TestDeactivateRechecksTheCredentialUnderTheLock(t *testing.T) {
	f := newCredentialFixture()
	f.tokens.credential.Revoked = true

	err := f.deactivate(&fakeMemberships{log: f.log}).Execute(shared.WithActor(context.Background(), tokenActor))

	if !errors.Is(err, shared.Unauthenticated()) || len(f.creds.writtenAt) != 0 || slices.ContainsFunc(f.log.calls, isMemberships) {
		t.Errorf("Execute() = %v, %d writes, calls %q; want 401 and none", err, len(f.creds.writtenAt), f.log.calls)
	}
}

// isMemberships reports whether call is the memberships' deactivation.
func isMemberships(call string) bool { return strings.HasPrefix(call, "deactivate the memberships") }

// The memberships' refusals and failures, as they come: a 409 of the
// workspace or project module, which identity does not import (stand-ins
// with their kind, code and detail), or a failure (M3 design 3.9, 9.4).
var membershipsErrors = []error{
	shared.NewError(shared.KindConflict, "workspace.sole_admin",
		"The workspace would be left without an admin: its only active admin cannot leave it, nor can his membership end while it "+
			"has other active members. It must first be given another admin, or be deleted."),
	shared.NewError(shared.KindConflict, "project.sole_admin",
		"The project would be left without an admin: its only active admin cannot leave it, nor can his membership end while it has "+
			"other active members. It must first be given another admin, or be deleted."),
	errors.New("connection reset"),
}

// A failed write fails the deactivation, which is then not logged; so does
// the memberships' refusal or failure, which comes back as itself, the
// first problem in its chain the one it was, out of the one transaction,
// after every write before it.
func TestDeactivateWhenAWriteFails(t *testing.T) {
	boom := errors.New("connection reset")
	tests := []struct {
		name string
		fail func(*fakeCredentials, *fakeMemberships)
		want error
	}{
		{"the account", func(c *fakeCredentials, _ *fakeMemberships) { c.deactivateErr = boom }, boom},
		{"the onboarding", func(c *fakeCredentials, _ *fakeMemberships) { c.resetErr = boom }, boom},
		{"the sessions", func(c *fakeCredentials, _ *fakeMemberships) { c.revokeErr = boom }, boom},
	}
	for _, err := range membershipsErrors {
		tests = append(tests, struct {
			name string
			fail func(*fakeCredentials, *fakeMemberships)
			want error
		}{"the memberships: " + err.Error(), func(_ *fakeCredentials, m *fakeMemberships) { m.err = err }, err})
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newCredentialFixture()
			m := &fakeMemberships{log: f.log}
			tt.fail(f.creds, m)

			err := f.deactivate(m).Execute(shared.WithActor(context.Background(), sessionActor))

			if !errors.Is(err, tt.want) || strings.Contains(f.logs.String(), "account deactivated") || f.tx.calls != 1 {
				t.Errorf("Execute() = %v in %d transactions, logs %s; want %v in one and no deactivation logged", err, f.tx.calls, f.logs.String(),
					tt.want)
			}
			if first := firstProblem(err); first != firstProblem(tt.want) {
				t.Errorf("Execute() = %v, answered as %v; want %v", err, first, firstProblem(tt.want))
			}
		})
	}
}

// firstProblem is the first *shared.Error in err's chain, the one the API
// answers, or nil.
func firstProblem(err error) error {
	var se *shared.Error
	if errors.As(err, &se) {
		return se
	}
	return nil
}

func TestDeactivateWithoutAnActor(t *testing.T) {
	f := newCredentialFixture()

	if err := f.deactivate(&fakeMemberships{log: f.log}).Execute(context.Background()); !errors.Is(err, shared.Unauthenticated()) ||
		len(f.log.calls) != 0 {
		t.Errorf("Execute() = %v after calls %q, want 401 before any", err, f.log.calls)
	}
}

// `nerve users deactivate` locks the account by its normalized address and
// writes what the self-service deactivation writes, in the same order, the
// memberships of the account at that address, the one it locked (M2
// decision 3, design 3.17; M3 design 3.9).
func TestDeactivateByEmail(t *testing.T) {
	f := newAdminFixture()

	got, err := f.deactivate(&fakeMemberships{log: f.log}).ExecuteByEmail(context.Background(), " Alice@Corp.com ")

	if want := (app.DeactivateResult{Email: "alice@corp.com", Sessions: 1}); err != nil || got != want {
		t.Fatalf("ExecuteByEmail() = %+v, %v; want %+v", got, err, want)
	}
	want := []string{
		"lock alice@corp.com", "deactivate " + userID.String(), "reset onboarding of " + userID.String(),
		"revoke deactivated sessions of " + userID.String() + " but " + uuid.Nil().String(),
		"deactivate the memberships of " + userID.String() + " at alice@corp.com",
	}
	if !slices.Equal(f.log.calls, want) || f.tx.calls != 1 || !slices.Equal(f.store.writtenAt, []time.Time{now, now, now}) {
		t.Errorf("calls %q in %d transactions at %v; want %q in one at %v", f.log.calls, f.tx.calls, f.store.writtenAt, want, now)
	}
	if logs := f.logs.String(); !strings.Contains(logs, `"msg":"account deactivated","user_id":"`+userID.String()+`","revoked_sessions":1,"by":"cli"`) {
		t.Errorf("logs = %s, want the deactivation with user_id and the sessions revoked, by cli", logs)
	}
}

func TestDeactivateByEmailOfAnUnknownAccount(t *testing.T) {
	f := newAdminFixture()

	_, err := f.deactivate(&fakeMemberships{log: f.log}).ExecuteByEmail(context.Background(), "carol@corp.com")

	if !errors.Is(err, domain.ErrAccountNotFound) || !slices.Equal(f.log.calls, []string{"lock carol@corp.com"}) || f.logs.Len() != 0 {
		t.Errorf("ExecuteByEmail() = %v after calls %q, logs %s; want identity.account_not_found after the lock alone", err, f.log.calls, f.logs.String())
	}
}

// A failed write fails the administrator's deactivation too, which is then
// not logged; so does the memberships' refusal or failure, as itself: the
// command prints its detail.
func TestDeactivateByEmailWhenAWriteFails(t *testing.T) {
	boom := errors.New("connection reset")
	f := newAdminFixture()
	f.store.revokeErr = boom

	if _, err := f.deactivate(&fakeMemberships{log: f.log}).ExecuteByEmail(context.Background(), "alice@corp.com"); !errors.Is(err, boom) ||
		f.logs.Len() != 0 {
		t.Errorf("ExecuteByEmail() = %v, logs %s; want %v and nothing logged", err, f.logs.String(), boom)
	}
	for _, want := range membershipsErrors {
		f := newAdminFixture()
		_, err := f.deactivate(&fakeMemberships{log: f.log, err: want}).ExecuteByEmail(context.Background(), "alice@corp.com")
		if !errors.Is(err, want) || firstProblem(err) != firstProblem(want) || f.logs.Len() != 0 {
			t.Errorf("ExecuteByEmail() with the memberships failing = %v, logs %s; want %v as itself and nothing logged", err, f.logs.String(),
				want)
		}
	}
}
````

`server/internal/modules/identity/module.go`（修改，2 处）：

````old server/internal/modules/identity/module.go
	SignupPolicy app.SignupPolicy
````
````new server/internal/modules/identity/module.go
	SignupPolicy app.SignupPolicy
	// Memberships is workspace's Deactivator: deactivateMe ends the
	// account's memberships through it (M3 design 3.9, 6.6 step 6).
	Memberships app.MembershipDeactivator
````

````old server/internal/modules/identity/module.go
				Lock: lock, Users: store, Profiles: store, Sessions: store, Tx: d.Tx, Clock: d.Clock, Logger: d.Logger,
````
````new server/internal/modules/identity/module.go
				Lock: lock, Users: store, Profiles: store, Sessions: store, Memberships: d.Memberships, Tx: d.Tx, Clock: d.Clock,
				Logger: d.Logger,
````

`server/internal/modules/identity/admin.go`（修改，3 处）：

````old server/internal/modules/identity/admin.go
// AdminDeps are what the server administrator's commands need: the pool and
// the password hashing, and no signing key, rate limit or sign-up policy
// (M2 design 3.17).
````
````new server/internal/modules/identity/admin.go
// AdminDeps are what the server administrator's commands need: the pool,
// the password hashing and the deactivation's MembershipDeactivator, and
// no signing key, rate limit or sign-up policy (M2 design 3.17, M3 design
// 6.6).
````

````old server/internal/modules/identity/admin.go
	Password PasswordHashing
````
````new server/internal/modules/identity/admin.go
	Password PasswordHashing
	// Memberships is workspace's Deactivator, which `nerve users
	// deactivate` ends the account's memberships through (M3 design 3.9).
	Memberships app.MembershipDeactivator
````

````old server/internal/modules/identity/admin.go
			Accounts: store, Users: store, Profiles: store, Sessions: store, Tx: d.Tx, Clock: d.Clock, Logger: d.Logger,
````
````new server/internal/modules/identity/admin.go
			Accounts: store, Users: store, Profiles: store, Sessions: store, Memberships: d.Memberships, Tx: d.Tx, Clock: d.Clock,
			Logger: d.Logger,
````

`server/internal/modules/identity/adapter/http/me_test.go`（修改，4 处）：

````old server/internal/modules/identity/adapter/http/me_test.go
	"encoding/json"
````
````new server/internal/modules/identity/adapter/http/me_test.go
	"encoding/json"
	"errors"
	"fmt"
````

````old server/internal/modules/identity/adapter/http/me_test.go
// A credential revoked since authentication: the use case's 401 is the
// answer, not 204.
func TestDeactivateMeProblem(t *testing.T) {
	req := withToken(httptest.NewRequest(http.MethodPost, "/api/v0/me/deactivate", nil))
````
````new server/internal/modules/identity/adapter/http/me_test.go
// The use case's refusals are the answer, not 204, as the contract declares
// them (M3 design 3.9, 9.4): a credential revoked since authentication,
// 401; the workspace module's workspace.sole_admin and the project module's
// project.sole_admin, which identity does not import (stand-ins with their
// kind, code and detail), each a 409 of deactivateMe as it comes, wrapped
// as the ending wraps it; a failure, 500, never another problem.
func TestDeactivateMeProblems(t *testing.T) {
	workspaceSoleAdmin := shared.NewError(shared.KindConflict, "workspace.sole_admin",
		"The workspace would be left without an admin: its only active admin cannot leave it, nor can his membership end while it "+
			"has other active members. It must first be given another admin, or be deleted.")
	projectSoleAdmin := shared.NewError(shared.KindConflict, "project.sole_admin",
		"The project would be left without an admin: its only active admin cannot leave it, nor can his membership end while it has "+
			"other active members. It must first be given another admin, or be deleted.")
	tests := []struct {
		err    error
		status int
		want   string
	}{
		{shared.Unauthenticated(), http.StatusUnauthorized, `{"status":401,"code":"unauthorized","title":"Unauthorized",` +
			`"detail":"Authentication is required."}`},
		{workspaceSoleAdmin, http.StatusConflict, `{"status":409,"code":"workspace.sole_admin","title":"Conflict","detail":"The workspace ` +
			`would be left without an admin: its only active admin cannot leave it, nor can his membership end while it has other active ` +
			`members. It must first be given another admin, or be deleted."}`},
		{fmt.Errorf("end the member's project memberships: %w", projectSoleAdmin), http.StatusConflict,
			`{"status":409,"code":"project.sole_admin","title":"Conflict","detail":"The project would be left without an admin: its only ` +
				`active admin cannot leave it, nor can his membership end while it has other active members. It must first be given ` +
				`another admin, or be deleted."}`},
		{errors.New("the database is gone"), http.StatusInternalServerError,
			`{"status":500,"code":"internal_error","title":"Internal Server Error"}`},
	}
	for _, tt := range tests {
		req := withToken(httptest.NewRequest(http.MethodPost, "/api/v0/me/deactivate", nil))
````

````old server/internal/modules/identity/adapter/http/me_test.go
	res, body := do(t, newServer(t, fakes{deactivate: &fakeDeactivate{err: shared.Unauthenticated()}}), req)
````
````new server/internal/modules/identity/adapter/http/me_test.go
		res, body := do(t, newServer(t, fakes{deactivate: &fakeDeactivate{err: tt.err}}), req)
````

````old server/internal/modules/identity/adapter/http/me_test.go
	if res.StatusCode != http.StatusUnauthorized || !strings.Contains(body, `"code":"unauthorized"`) {
		t.Errorf("POST /me/deactivate = %d %s, want 401 unauthorized", res.StatusCode, body)
````
````new server/internal/modules/identity/adapter/http/me_test.go
		if res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("POST /me/deactivate refused with %v = %d %s, want %d %s", tt.err, res.StatusCode, body, tt.status, tt.want)
		}
````

- [ ] **Step 4: `workspace.NewDeactivator` 和两处组合**

`server/internal/modules/workspace/deactivator.go`（新文件，41 行）：

````file server/internal/modules/workspace/deactivator.go
package workspace

import (
	"context"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
)

// Deactivator ends a deactivated account's memberships: identity's
// MembershipDeactivator (M3 design 3.9), which bootstrap wires into
// identity.New and identity.NewAdmin. It runs in the deactivation's
// transaction, which ctx carries, under the account row's lock.
type Deactivator interface {
	// DeactivateMemberships ends every workspace and project membership of
	// account userID and deletes every invitation to email; its refusal,
	// workspace.sole_admin or project.sole_admin, or its failure comes back
	// as itself.
	DeactivateMemberships(ctx context.Context, userID uuid.UUID, email string) error
}

// DeactivatorDeps are what the Deactivator needs: the pool, the clock and
// the project module's cascade, and no Authorizer, signing key or jobs
// client (M3 design 6.6).
type DeactivatorDeps struct {
	Pool  *pgxpool.Pool
	Clock app.Clock
	// Projects is the project module's Cascade: project.New's, or the
	// command line's project.NewCascade.
	Projects app.ProjectCascade
}

// NewDeactivator builds the Deactivator alone, on the pool: for the
// command line, whose `nerve users deactivate` ends memberships through it
// and builds no HTTP side (M3 design 6.6). New builds its own through it.
func NewDeactivator(d DeactivatorDeps) Deactivator {
	return app.NewDeactivator(postgresadapter.New(d.Pool), d.Projects, d.Clock)
}
````

`server/internal/modules/workspace/module.go`（修改，4 处）：

````old server/internal/modules/workspace/module.go
// member's display settings, and the invitations, and offers the other
// modules its reads through ports.
````
````new server/internal/modules/workspace/module.go
// member's display settings, and the invitations, ends a deactivated
// account's memberships for identity, and offers the other modules its
// reads through ports.
````

````old server/internal/modules/workspace/module.go
	uc     httpadapter.UseCases
	signup *app.SignupInvitations
````
````new server/internal/modules/workspace/module.go
	uc          httpadapter.UseCases
	signup      *app.SignupInvitations
	deactivator Deactivator
````

````old server/internal/modules/workspace/module.go
	return &Module{signup: app.NewSignupInvitations(store, d.InvitationMAC), uc: httpadapter.UseCases{
````
````new server/internal/modules/workspace/module.go
	deactivator := NewDeactivator(DeactivatorDeps{Pool: d.Pool, Clock: d.Clock, Projects: d.Projects})
	return &Module{signup: app.NewSignupInvitations(store, d.InvitationMAC), deactivator: deactivator, uc: httpadapter.UseCases{
````

````old server/internal/modules/workspace/module.go
}

// Actions lists the module's actions: bootstrap's test holds the union of
````
````new server/internal/modules/workspace/module.go
}

// Deactivator is the ending of a deactivated account's memberships, for
// identity's MembershipDeactivator (M3 design 3.9, 6.6 step 6).
func (m *Module) Deactivator() Deactivator {
	return m.deactivator
}

// Actions lists the module's actions: bootstrap's test holds the union of
````

`server/internal/bootstrap/app.go`（修改，1 处）：

````old server/internal/bootstrap/app.go
		SignupPolicy:    signupPolicy{enabled: cfg.Auth.SignupEnabled, invitations: ws.SignupInvitations()},
````
````new server/internal/bootstrap/app.go
		SignupPolicy:    signupPolicy{enabled: cfg.Auth.SignupEnabled, invitations: ws.SignupInvitations()},
		Memberships:     ws.Deactivator(),
````

`server/internal/bootstrap/users.go`（修改，3 处）：

````old server/internal/bootstrap/users.go
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
````
````new server/internal/bootstrap/users.go
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
````

````old server/internal/bootstrap/users.go
// identity's administrator use cases; no HTTP server, no jobs client. The
````
````new server/internal/bootstrap/users.go
// identity's administrator use cases, the deactivation's with workspace's
// Deactivator over project's cascade, each built on the pool alone (M3
// design 6.6); no HTTP server, no Authorizer, no jobs client. The
````

````old server/internal/bootstrap/users.go
		Password: passwordHashing(cfg.Auth.Password),
````
````new server/internal/bootstrap/users.go
		Password: passwordHashing(cfg.Auth.Password),
		Memberships: workspace.NewDeactivator(workspace.DeactivatorDeps{
			Pool: pool, Clock: clock.System{}, Projects: project.NewCascade(project.CascadeDeps{Pool: pool}),
		}),
````

`server/internal/archtest/composition_test.go`（修改，4 处）：

````old server/internal/archtest/composition_test.go
// client. The rule follows the static calls from each; the commands are
// func values it calls dynamically, so they are not followed: they only
// receive the composition. Reaching the module's NewAdmin shows the walk
// sees the composition at all.
````
````new server/internal/archtest/composition_test.go
// client. Each builds only what its commands use: Users, identity's
// NewAdmin and, for `nerve users deactivate`, workspace's NewDeactivator
// and project's NewCascade, never workspace's NewAdmin; Workspaces,
// workspace's NewAdmin, never the deactivation's two, whose cascade no
// command of it calls. The rule follows the static calls from each; the
// commands are func values it calls dynamically, so they are not followed:
// they only receive the composition. Reaching what each builds shows the
// walk sees the composition at all.
````

````old server/internal/archtest/composition_test.go
	for _, c := range []struct{ root, admin string }{
		{"Users", m("internal/modules/identity") + ".NewAdmin"},
		{"Workspaces", m("internal/modules/workspace") + ".NewAdmin"},
````
````new server/internal/archtest/composition_test.go
	identity, workspace, project := m("internal/modules/identity"), m("internal/modules/workspace"), m("internal/modules/project")
	for _, c := range []struct {
		root            string
		builds, unbuilt []string
	}{
		{"Users", []string{identity + ".NewAdmin", workspace + ".NewDeactivator", project + ".NewCascade"}, []string{workspace + ".NewAdmin"}},
		{"Workspaces", []string{workspace + ".NewAdmin"}, []string{workspace + ".NewDeactivator", project + ".NewCascade"}},
````

````old server/internal/archtest/composition_test.go
		if !slices.ContainsFunc(reached, func(chain []*ssa.Function) bool {
			return chain[len(chain)-1].String() == c.admin
		}) {
			var names []string
			for _, chain := range reached {
				names = append(names, funcName(chain[len(chain)-1]))
````
````new server/internal/archtest/composition_test.go
		var names []string
		for _, chain := range reached {
			names = append(names, chain[len(chain)-1].String())
		}
		for _, f := range c.builds {
			if !slices.Contains(names, f) {
				t.Errorf("bootstrap.%s does not reach %s, so the rule checks nothing; it reaches:\n%s",
					c.root, f, strings.ReplaceAll(strings.Join(names, "\n"), modulePath+"/", ""))
````

````old server/internal/archtest/composition_test.go
			t.Errorf("bootstrap.%s does not reach %s, so the rule checks nothing; it reaches:\n%s",
				c.root, c.admin, strings.Join(names, "\n"))
````
````new server/internal/archtest/composition_test.go
		}
		for _, f := range c.unbuilt {
			if slices.Contains(names, f) {
				t.Errorf("bootstrap.%s reaches %s, which none of its commands uses (M3 design 6.6)", c.root, f)
			}
````

- [ ] **Step 5: 交错 8、19 换成真实的停用**

`server/internal/bootstrap/deactivating_test.go`（新文件，49 行）：

````file server/internal/bootstrap/deactivating_test.go
package bootstrap

import (
	"context"
	"log/slog"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	identitypg "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// deactivating is `nerve users deactivate` as bootstrap's Users wires it
// (users.go): identity's deactivation over its store, its memberships'
// step workspace's Deactivator, with project's cascade (project.NewCascade).
// sessions is identity's store, or one a gate stops; memberships the
// workspace store, or one a gate stops. Its commit and its rollback have 2s
// each; the caller's context bounds its waits.
func deactivating(pool *pgxpool.Pool, sessions identityapp.SessionRevoker, memberships workspaceapp.AllMembershipsEnder) *identityapp.Deactivate {
	store := identitypg.New(pool)
	return identityapp.NewDeactivate(identityapp.DeactivateDeps{
		Accounts: store, Users: store, Profiles: store, Sessions: sessions,
		Memberships: workspaceapp.NewDeactivator(memberships, project.NewCascade(project.CascadeDeps{Pool: pool}), clock.System{}),
		Tx:          postgres.NewTxManager(pool, 2*time.Second), Clock: clock.System{}, Logger: slog.New(slog.DiscardHandler),
	})
}

// endedHoldingAll stops a deactivation once it has ended the account's
// workspace memberships, before the projects' step: it holds the account
// row, every workspace of his FOR NO KEY UPDATE, the invitations to his
// address and his memberships' rows.
type endedHoldingAll struct {
	*workspacepg.Store
	gate *gate
}

func (m endedHoldingAll) EndWorkspaceMemberships(ctx context.Context, workspaceIDs []uuid.UUID, userID, by uuid.UUID, now time.Time) error {
	if err := m.Store.EndWorkspaceMemberships(ctx, workspaceIDs, userID, by, now); err != nil {
		return err
	}
	return m.gate.wait(ctx)
}
````

`server/internal/bootstrap/interleaving_test.go`（修改，12 处）：

````old server/internal/bootstrap/interleaving_test.go
// Creating a workspace against M2's deactivation of its admin's account, in
// both orders, on a real database (M3 design 3.6 convention 6; the
// interleaving 8 of 9.3 without the membership's end, which the
// deactivation adds with its port). Each side runs its real use case; a
// gate inside its transaction, after its lock of the account row, holds the
// transaction open, and pgtest.WaitForLockWait proves that the other side
// waits on that row before the gate opens. Every wait has a deadline.
````
````new server/internal/bootstrap/interleaving_test.go
// Interleaving 8 of M3 design 9.3: creating a workspace against the
// deactivation of its admin's account, in both orders, on a real database
// (3.6 convention 6), through the API's creation and the command line's
// (`nerve workspaces create`), against `nerve users deactivate` as
// bootstrap wires it, its memberships' step included. Each side runs its
// real use case; a gate inside its transaction, after its lock of the
// account row, holds the transaction open, and pgtest.WaitForLockWaitOn
// proves that the other side waits on that row before the gate opens. Every
// wait has a deadline.
````

````old server/internal/bootstrap/interleaving_test.go
// the account row: a deactivation at its last write, a change of address
// after its write.
````
````new server/internal/bootstrap/interleaving_test.go
// the account row: a deactivation before its memberships' step, a change of
// address after its write.
````

````old server/internal/bootstrap/interleaving_test.go

// deactivation is M2's `nerve users deactivate` over sessions.
func (r race) deactivation(sessions identityapp.SessionRevoker) *identityapp.Deactivate {
	store := identitypg.New(r.pool)
	return identityapp.NewDeactivate(identityapp.DeactivateDeps{
		Accounts: store, Users: store, Profiles: store, Sessions: sessions,
		Tx: postgres.NewTxManager(r.pool, 2*time.Second), Clock: clocktest.At(time.Now()), Logger: slog.New(slog.DiscardHandler),
	})
}

````
````new server/internal/bootstrap/interleaving_test.go

````

````old server/internal/bootstrap/interleaving_test.go
// state is whether alice's account is active, and how many workspaces and
// memberships there are.
func (r race) state(t *testing.T) (active bool, workspaces, members int) {
````
````new server/internal/bootstrap/interleaving_test.go
// creating is one entry of the creation of acme by alice, and what it
// answers when it finds her account deactivated under its lock.
type creating struct {
	name        string
	create      func(ctx context.Context, uc *workspaceapp.CreateWorkspace, alice uuid.UUID) error
	deactivated error
}

var creatings = []creating{
	{"createWorkspace", func(ctx context.Context, uc *workspaceapp.CreateWorkspace, alice uuid.UUID) error {
		_, err := uc.Execute(shared.WithActor(ctx, shared.Actor{UserID: alice}), domain.NewWorkspace{Name: "Acme", Slug: "acme"})
		return err
	}, shared.Unauthenticated()},
	{"nerve workspaces create", func(ctx context.Context, uc *workspaceapp.CreateWorkspace, _ uuid.UUID) error {
		_, err := uc.ExecuteForAdmin(ctx, "alice@example.com", domain.NewWorkspace{Name: "Acme", Slug: "acme"})
		return err
	}, domain.ErrAccountDeactivated},
}

// state is whether alice's account is active, how many workspaces there
// are and how many memberships, and how many of those are active.
func (r race) state(t *testing.T) (active bool, workspaces, members, activeMembers int) {
````

````old server/internal/bootstrap/interleaving_test.go
	if err := r.pool.QueryRow(context.Background(),
		"SELECT is_active, (SELECT count(*) FROM workspaces), (SELECT count(*) FROM workspace_members) FROM users WHERE id = $1", r.alice).
		Scan(&active, &workspaces, &members); err != nil {
````
````new server/internal/bootstrap/interleaving_test.go
	if err := r.pool.QueryRow(soon(t), `SELECT is_active, (SELECT count(*) FROM workspaces), (SELECT count(*) FROM workspace_members),
		(SELECT count(*) FROM workspace_members WHERE is_active) FROM users WHERE id = $1`, r.alice).
		Scan(&active, &workspaces, &members, &activeMembers); err != nil {
````

````old server/internal/bootstrap/interleaving_test.go
	return active, workspaces, members
````
````new server/internal/bootstrap/interleaving_test.go
	return active, workspaces, members, activeMembers
````

````old server/internal/bootstrap/interleaving_test.go
// FOR SHARE, then reads the account deactivated under the lock and answers
// 401. No workspace.
````
````new server/internal/bootstrap/interleaving_test.go
// FOR SHARE, then reads the account deactivated under the lock and is
// refused: the API's 401, the command's workspace.account_deactivated. No
// workspace.
````

````old server/internal/bootstrap/interleaving_test.go
	r := newRace(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	g := newGate()
	deactivated := run(func() error {
		_, err := r.deactivation(gatedSessions{identitypg.New(r.pool), g}).ExecuteByEmail(ctx, "alice@example.com")
		return err
	})
	held(t, ctx, g, deactivated, "the deactivation")
	created := run(func() error {
		_, err := r.creation(workspacepg.New(r.pool)).Execute(shared.WithActor(ctx, shared.Actor{UserID: r.alice}),
			domain.NewWorkspace{Name: "Acme", Slug: "acme"})
		return err
	})
	pgtest.WaitForLockWait(t, r.pool, 5*time.Second)
	close(g.open)
````
````new server/internal/bootstrap/interleaving_test.go
	for _, c := range creatings {
		t.Run(c.name, func(t *testing.T) {
			r := newRace(t)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			g := newGate()
			deactivated := run(func() error {
				_, err := deactivating(r.pool, gatedSessions{identitypg.New(r.pool), g}, workspacepg.New(r.pool)).
					ExecuteByEmail(ctx, "alice@example.com")
				return err
			})
			held(t, ctx, g, deactivated, "the deactivation")
			created := run(func() error { return c.create(ctx, r.creation(workspacepg.New(r.pool)), r.alice) })
			pgtest.WaitForLockWaitOn(t, r.pool, "users", 5*time.Second)
			close(g.open)
````

````old server/internal/bootstrap/interleaving_test.go
	if err := result(t, ctx, deactivated, "the deactivation"); err != nil {
		t.Fatalf("the deactivation: %v", err)
	}
	if err := result(t, ctx, created, "the creation"); !errors.Is(err, shared.Unauthenticated()) {
		t.Errorf("the creation = %v, want 401 unauthorized", err)
	}
	if active, workspaces, members := r.state(t); active || workspaces != 0 || members != 0 {
		t.Errorf("alice active %v, %d workspaces, %d memberships; want deactivated and none", active, workspaces, members)
````
````new server/internal/bootstrap/interleaving_test.go
			if err := result(t, ctx, deactivated, "the deactivation"); err != nil {
				t.Fatalf("the deactivation: %v", err)
			}
			if err := result(t, ctx, created, "the creation"); !errors.Is(err, c.deactivated) {
				t.Errorf("the creation = %v, want %v", err, c.deactivated)
			}
			if active, workspaces, members, _ := r.state(t); active || workspaces != 0 || members != 0 {
				t.Errorf("alice active %v, %d workspaces, %d memberships; want deactivated and none", active, workspaces, members)
			}
		})
````

````old server/internal/bootstrap/interleaving_test.go
// deactivates the account. Both succeed.
````
````new server/internal/bootstrap/interleaving_test.go
// deactivates the account and ends its membership of the new workspace,
// whose only member, and so admin, it is (3.7 rule 2 allows it). Both
// succeed.
````

````old server/internal/bootstrap/interleaving_test.go
	r := newRace(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	g := newGate()
	created := run(func() error {
		_, err := r.creation(gatedWorkspaces{workspacepg.New(r.pool), g}).Execute(shared.WithActor(ctx, shared.Actor{UserID: r.alice}),
			domain.NewWorkspace{Name: "Acme", Slug: "acme"})
		return err
	})
	held(t, ctx, g, created, "the creation")
	deactivated := run(func() error {
		_, err := r.deactivation(identitypg.New(r.pool)).ExecuteByEmail(ctx, "alice@example.com")
		return err
	})
	pgtest.WaitForLockWait(t, r.pool, 5*time.Second)
	if active, workspaces, members := r.state(t); !active || workspaces != 0 || members != 0 {
		t.Errorf("while the creation holds the row: alice active %v, %d workspaces, %d memberships; want active and none committed",
			active, workspaces, members)
	}
	close(g.open)
````
````new server/internal/bootstrap/interleaving_test.go
	for _, c := range creatings {
		t.Run(c.name, func(t *testing.T) {
			r := newRace(t)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			g := newGate()
			created := run(func() error { return c.create(ctx, r.creation(gatedWorkspaces{workspacepg.New(r.pool), g}), r.alice) })
			held(t, ctx, g, created, "the creation")
			deactivated := run(func() error {
				_, err := deactivating(r.pool, identitypg.New(r.pool), workspacepg.New(r.pool)).ExecuteByEmail(ctx, "alice@example.com")
				return err
			})
			pgtest.WaitForLockWaitOn(t, r.pool, "users", 5*time.Second)
			if active, workspaces, members, _ := r.state(t); !active || workspaces != 0 || members != 0 {
				t.Errorf("while the creation holds the row: alice active %v, %d workspaces, %d memberships; want active and none committed",
					active, workspaces, members)
			}
			close(g.open)
````

````old server/internal/bootstrap/interleaving_test.go
	if err := result(t, ctx, created, "the creation"); err != nil {
		t.Errorf("the creation = %v, want it done", err)
	}
	if err := result(t, ctx, deactivated, "the deactivation"); err != nil {
		t.Errorf("the deactivation = %v, want it done", err)
	}
	if active, workspaces, members := r.state(t); active || workspaces != 1 || members != 1 {
		t.Errorf("alice active %v, %d workspaces, %d memberships; want deactivated after one workspace and its admin", active, workspaces, members)
````
````new server/internal/bootstrap/interleaving_test.go
			if err := result(t, ctx, created, "the creation"); err != nil {
				t.Errorf("the creation = %v, want it done", err)
			}
			if err := result(t, ctx, deactivated, "the deactivation"); err != nil {
				t.Errorf("the deactivation = %v, want it done", err)
			}
			if active, workspaces, members, activeMembers := r.state(t); active || workspaces != 1 || members != 1 || activeMembers != 0 {
				t.Errorf("alice active %v, %d workspaces, %d memberships, %d active; want deactivated after one workspace and its admin, "+
					"her membership ended", active, workspaces, members, activeMembers)
			}
		})
````

`server/internal/bootstrap/interleaving_answers_test.go`（修改，8 处）：

````old server/internal/bootstrap/interleaving_answers_test.go
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	identitydomain "github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
````
````new server/internal/bootstrap/interleaving_answers_test.go
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
````

````old server/internal/bootstrap/interleaving_answers_test.go
// gatedDeactivation takes, after it revokes the sessions, the lock of acme
// that P6's deactivation takes (M3 design 3.9: the account row, profiles,
// auth_sessions, then each workspace FOR NO KEY UPDATE), and stops there
// when it has a gate.
type gatedDeactivation struct {
	identityapp.SessionRevoker
	pool *pgxpool.Pool
	acme uuid.UUID
	gate *gate // nil: never stops
}

func (d gatedDeactivation) RevokeSessions(ctx context.Context, userID, keep uuid.UUID, reason identitydomain.RevokeReason, now time.Time) (int, error) {
	revoked, err := d.SessionRevoker.RevokeSessions(ctx, userID, keep, reason, now)
	if err == nil {
		_, err = postgres.DB(ctx, d.pool).Exec(ctx, "SELECT id FROM workspaces WHERE id = $1 FOR NO KEY UPDATE", d.acme)
	}
	if err == nil && d.gate != nil {
		err = d.gate.wait(ctx)
	}
	return revoked, err
}

// deactivateBob is `nerve users deactivate` of bob, over sessions.
func (r answerRace) deactivateBob(ctx context.Context, sessions identityapp.SessionRevoker) error {
	store := identitypg.New(r.pool)
	_, err := identityapp.NewDeactivate(identityapp.DeactivateDeps{Accounts: store, Users: store, Profiles: store, Sessions: sessions, Tx: r.tx(),
		Clock: clocktest.At(time.Now()), Logger: slog.New(slog.DiscardHandler)}).ExecuteByEmail(ctx, "bob@example.com")
````
````new server/internal/bootstrap/interleaving_answers_test.go
// deactivateBob is `nerve users deactivate` of bob as bootstrap wires it
// (deactivating), over memberships.
func (r answerRace) deactivateBob(ctx context.Context, memberships workspaceapp.AllMembershipsEnder) error {
	_, err := deactivating(r.pool, identitypg.New(r.pool), memberships).ExecuteByEmail(ctx, "bob@example.com")
````

````old server/internal/bootstrap/interleaving_answers_test.go
// Interleaving 19: bob is acme's member and has an invitation to it, as
// after reactivate-member or a change of address. The decline first holds
// his account row FOR SHARE, then acme's; the deactivation waits on the
// account row, then goes on through acme. The deactivation first holds the
// account row and acme's; the decline waits on the account row, then reads
// it deactivated under its lock: 401, and the invitation stays pending.
// Neither deadlocks: both lock the account row first (3.6 convention 1).
````
````new server/internal/bootstrap/interleaving_answers_test.go
// Interleaving 19 (M3 design 9.3, 3.6 convention 1): bob is acme's member
// and has an invitation to it, as after reactivate-member or a change of
// address; the deactivation is `nerve users deactivate` as bootstrap wires
// it. The decline first holds his account row FOR SHARE, then acme's; the
// deactivation waits on the account row, then goes on, and deletes the
// invitation, declined by then. The deactivation first holds the account
// row, acme's, the invitation, which it has deleted, and his membership,
// which it has ended; the decline waits on the account row, then reads it
// deactivated under its lock: 401, the invitation unanswered. Either way the
// deactivation is the last to write the invitation and the membership, as
// bob, at one moment, and nothing deadlocks: both lock the account row
// first.
````

````old server/internal/bootstrap/interleaving_answers_test.go
			g := newGate()
			store, users := workspacepg.New(r.pool), identitypg.New(r.pool)
			var declined, deactivated <-chan error
````
````new server/internal/bootstrap/interleaving_answers_test.go
			g := newGate()
			store := workspacepg.New(r.pool)
			var declined, deactivated <-chan error
````

````old server/internal/bootstrap/interleaving_answers_test.go
				deactivated = run(func() error { return r.deactivateBob(ctx, gatedDeactivation{users, r.pool, r.acme, nil}) })
````
````new server/internal/bootstrap/interleaving_answers_test.go
				deactivated = run(func() error { return r.deactivateBob(ctx, store) })
````

````old server/internal/bootstrap/interleaving_answers_test.go
				deactivated = run(func() error { return r.deactivateBob(ctx, gatedDeactivation{users, r.pool, r.acme, g}) })
````
````new server/internal/bootstrap/interleaving_answers_test.go
				deactivated = run(func() error { return r.deactivateBob(ctx, endedHoldingAll{store, g}) })
````

````old server/internal/bootstrap/interleaving_answers_test.go
			var active bool
			if err := r.pool.QueryRow(context.Background(), "SELECT is_active FROM users WHERE id = $1", r.bob).Scan(&active); err != nil {
````
````new server/internal/bootstrap/interleaving_answers_test.go
			var active, member, oneMoment, byBob bool
			if err := r.pool.QueryRow(soon(t), `SELECT u.is_active, m.is_active, m.updated_at = i.deleted_at,
				m.updated_by_id = u.id AND i.updated_by_id = u.id FROM users u, workspace_members m, workspace_member_invites i
				WHERE u.id = $1 AND m.workspace_id = $2 AND m.member_id = u.id AND i.id = $3`, r.bob, r.acme, r.invitation.id).
				Scan(&active, &member, &oneMoment, &byBob); err != nil {
````

````old server/internal/bootstrap/interleaving_answers_test.go
			if active || acc || responded != declineFirst || deleted {
				t.Errorf("bob active %v; the invitation accepted %v, answered %v, deleted %v; want deactivated, answered %v, undeleted",
					active, acc, responded, deleted, declineFirst)
````
````new server/internal/bootstrap/interleaving_answers_test.go
			if active || member || acc || responded != declineFirst || !deleted || !oneMoment || !byBob {
				t.Errorf("bob active %v, a member %v; the invitation accepted %v, answered %v, deleted %v; the two written at one moment %v, "+
					"by bob %v; want deactivated, no member, answered %v, deleted with his membership's end, by him", active, member, acc,
					responded, deleted, oneMoment, byBob, declineFirst)
````

- [ ] **Step 6: 测试、lint 和前端检查**

Run: `go -C server test -count=1 ./internal/modules/identity/... ./internal/modules/workspace/... ./internal/archtest/ ./internal/bootstrap/`
Expected: 全部 `ok`。

Run: `go -C server test -count=5 -race -run 'TestDeactivationFirstRefusesTheWorkspace$|TestCreationFirstHoldsOffTheDeactivation$|TestDecliningAndDeactivating$' ./internal/bootstrap/`
Expected: `ok`；输出里没有 `40P01`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

Run: `make lint-web`
Expected: 通过（关键词守卫的命中都有例外）。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

- [ ] **Step 7: 提交**

```bash
git add api/modules/identity.yaml server/internal/archtest/composition_test.go server/internal/bootstrap/app.go server/internal/bootstrap/deactivating_test.go server/internal/bootstrap/interleaving_answers_test.go server/internal/bootstrap/interleaving_test.go server/internal/bootstrap/users.go server/internal/modules/identity/adapter/http/me_test.go server/internal/modules/identity/adapter/postgres/credentials_test.go server/internal/modules/identity/adapter/postgres/queries/users.sql server/internal/modules/identity/adapter/postgres/users.go server/internal/modules/identity/admin.go server/internal/modules/identity/app/deactivate.go server/internal/modules/identity/app/deactivate_test.go server/internal/modules/identity/app/ports.go server/internal/modules/identity/module.go server/internal/modules/workspace/deactivator.go server/internal/modules/workspace/module.go api/dist/openapi.yaml server/internal/modules/identity/adapter/postgres/gen/users.sql.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M3/P6): a deactivation ends the account's memberships through identity's MembershipDeactivator

LockedAccount carries the address read under the account row's lock,
and deactivate, after the account, its onboarding and its sessions,
calls MembershipDeactivator with it; a refusal or failure comes back
as itself and rolls the whole deactivation back. deactivateMe declares
workspace.sole_admin and project.sole_admin, which identity's own HTTP
tests answer. The workspace module builds its Deactivator through
NewDeactivator, which the server wires into identity.New and nerve
users, over project.NewCascade, into identity.NewAdmin; the archtest
pins what each command composition builds. Interleavings 8 and 19 run
the real deactivation.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p6.py` 的编号，"层"是它被发现的每一层；端到端是单独运行的故事）：

| 变异 | 改坏 | 必须失败的测试 | 层 |
|---|---|---|---|
| `o-email-before-lock` | 邮箱在账户行锁之前读 | `TestADeactivationReadsTheAddressUnderItsLock`（Task 6 起） | 组合 |
| `o-account-share` | 账户行锁成 `FOR SHARE` | `TestAShareThatFailsReturnsTheError`、`TestLockAccountIsTheAccountRowLock`、`TestTheCredentialLockBlocksLocksNotInserts`、`TestTheShareLockBlocksDeactivationNotAnotherShare`、`TestAcceptingAndChangingTheAddress`（Task 7 起）、`TestAcceptingAndDeactivating`（Task 7 起）、`TestAnAdmittedAdminsWriteAndHisDeactivation`（Task 7 起）、`TestCreationFirstHoldsOffTheDeactivation` 等 8 个 | 存储；组合 |
| `o-account-update` | 账户行锁成 `FOR UPDATE` | `TestLockAccountIsTheAccountRowLock`、`TestTheCredentialLockBlocksLocksNotInserts`、`TestADeactivationAndACreationHeLeadsSerialize`（Task 8 起）、`TestADeactivationAndTheProjectSidesGrowthSerialize`（Task 8 起）、`TestAnAdmittedAdminsWriteAndHisDeactivation`（Task 7 起）、`TestEachLockOfADeactivationIsItsStrength`（Task 6 起） | 存储；组合 |
| `s2-id-memberships-swallowed` | `identity` 不管成员关系一步的回答照样提交 | `TestDeactivateByEmailWhenAWriteFails`、`TestDeactivateWhenAWriteFails`、`TestARefusedDeactivationChangesNothing`（Task 5 起）、`TestAnAdmittedAdminsWriteAndHisDeactivation`（Task 7 起）、`TestTwoAdminsDeactivatedAtOnceLeaveAnAdmin`（Task 8 起）、`TestUsersCommandsFail`（Task 5 起）、W9（Task 9 起） | 单元；组合；端到端 |
| `s4-app-skips-memberships` | 服务把一个什么都不结束的成员关系一步交给 `identity.New` | `TestADeactivationEndsEveryMembership`（Task 5 起）、`TestADeactivationFindsWhatChangedMeanwhile`（Task 6 起）、`TestADeactivationReadsTheAddressUnderItsLock`（Task 6 起）、`TestADeactivationRefusedAtItsCommitChangesNothing`（Task 5 起） 等 6 个、W9（Task 9 起） | 组合；端到端 |
| `s4-users-skips-memberships` | `nerve users` 把一个什么都不结束的成员关系一步交给 `identity.NewAdmin` | `TestADeactivationEndsEveryMembership`（Task 5 起）、`TestADeactivationFindsWhatChangedMeanwhile`（Task 6 起）、`TestADeactivationRefusedAtItsCommitChangesNothing`（Task 5 起）、`TestARefusedDeactivationChangesNothing`（Task 5 起） 等 8 个、W9（Task 9 起） | 组合；端到端 |
| `s4-users-no-project-end` | `nerve users` 的 `Deactivator` 接一个不结束项目成员关系的连带 | `TestADeactivationEndsEveryMembership`（Task 5 起）、`TestADeactivationFindsWhatChangedMeanwhile`（Task 6 起）、`TestADeactivationRefusedAtItsCommitChangesNothing`（Task 5 起）、`TestARefusedDeactivationChangesNothing`（Task 5 起） 等 6 个 | 组合 |
| `s4-users-frozen-clock` | `nerve users` 的 `Deactivator` 接 2001 年的时钟 | `TestADeactivationEndsEveryMembership`（Task 5 起）、`TestEachLockOfADeactivationIsItsStrength`（Task 6 起） | 组合 |
| `s4-server-no-project-end` | `workspace.New` 的 `Deactivator` 接一个不结束项目成员关系的连带 | `TestADeactivationEndsEveryMembership`（Task 5 起）、`TestADeactivationFindsWhatChangedMeanwhile`（Task 6 起）、`TestADeactivationReadsTheAddressUnderItsLock`（Task 6 起）、`TestADeactivationRefusedAtItsCommitChangesNothing`（Task 5 起） 等 6 个、W9（Task 9 起） | 组合；端到端 |
| `s4-server-frozen-clock` | `workspace.New` 的 `Deactivator` 接 2001 年的时钟 | `TestADeactivationEndsEveryMembership`（Task 5 起）、`TestEachLockOfADeactivationIsItsStrength`（Task 6 起） | 组合 |
| `s4-users-builds-workspace-admin` | `nerve users` 另建 `workspace.NewAdmin` | `TestCommandsComposeNoServerAndNoJobs` | 单元 |
| `s4-workspaces-builds-cascade` | `nerve workspaces` 另建 `project.NewCascade` | `TestCommandsComposeNoServerAndNoJobs` | 单元 |
| `s9-memberships-first` | `identity` 先结束成员关系、再写自己的行 | `TestDeactivate`、`TestDeactivateByEmail`、`TestAnAdmittedAdminsWriteAndHisDeactivation`（Task 7 起） | 单元；组合 |
| `s18-undeclared-workspace` | `deactivateMe` 不声明 `workspace.sole_admin` | `TestDeactivateMeProblems`、`TestARefusedDeactivationChangesNothing`（Task 5 起） | 单元；组合 |
| `s18-undeclared-project` | `deactivateMe` 不声明 `project.sole_admin` | `TestDeactivateMeProblems`、`TestARefusedDeactivationChangesNothing`（Task 5 起） | 单元；组合 |
| `s18-extra-code-module` | `deactivateMe` 在 `api/modules/identity.yaml` 里多声明 `workspace.not_found` | （见 spec 第 3 节） | 单元 |
| `s21-id-wrapped-404` | `identity` 把成员关系的拒绝与 `identity.account_not_found` 一起 `errors.Join` | `TestDeactivateByEmailWhenAWriteFails`、`TestDeactivateWhenAWriteFails`、`TestARefusedDeactivationChangesNothing`（Task 5 起）、`TestAnAdmittedAdminsWriteAndHisDeactivation`（Task 7 起）、`TestTwoAdminsDeactivatedAtOnceLeaveAnAdmin`（Task 8 起）、`TestUsersCommandsFail`（Task 5 起） | 单元；组合 |
| `s22-api-no-address` | `deactivateMe` 传空的地址 | `TestDeactivate`、`TestADeactivationEndsEveryMembership`（Task 5 起）、`TestADeactivationFindsWhatChangedMeanwhile`（Task 6 起）、`TestADeactivationReadsTheAddressUnderItsLock`（Task 6 起）、`TestADeactivationRefusedAtItsCommitChangesNothing`（Task 5 起） 等 5 个 | 单元；组合 |
| `s22-cli-raw-address` | 命令传原样输入的地址，不传规范化的 | `TestDeactivateByEmail`、`TestADeactivationEndsEveryMembership`（Task 5 起）、`TestADeactivationFindsWhatChangedMeanwhile`（Task 6 起）、`TestADeactivationRefusedAtItsCommitChangesNothing`（Task 5 起）、`TestEachLockOfADeactivationIsItsStrength`（Task 6 起） | 单元；组合 |

**Done when:** `grep -rn "gatedDeactivation" server/` 没有输出；`deactivateMe` 的两个码由 `identity` 的 HTTP 测试返回，`apitest.Main` 两个方向通过；`TestCommandsComposeNoServerAndNoJobs` 通过；交错 8、19 在 `-count=5 -race` 下通过、没有 40P01；`make lint-go`、`make test`、`make lint-web`、`make knip`、`make test-web` 通过；生成物与表相同。

### Task 5: 两条路的组合测试；命令的输出

**Files:**
- Create: `server/internal/bootstrap/deactivation_test.go`、`server/internal/bootstrap/deactivation_world_test.go`
- Modify: `server/cmd/nerve/users.go`、`server/cmd/nerve/users_test.go`、`server/internal/bootstrap/reactivation_races_test.go`、`server/internal/bootstrap/users.go`、`server/internal/bootstrap/users_test.go`

**Interfaces:**
- Produces（spec 2.7，M3 设计 3.9、8.7、9.3）：`DeactivateUser` 的一行 `deactivated <邮箱>: revoked <n> sessions and ended its memberships; to bring it back, run nerve users activate, then nerve workspaces reactivate-member in each workspace`；`nerve users deactivate`、`activate` 的说明（`cmd/nerve/users.go`）。拒绝时命令打印问题的说明、退出码 1（M2 的 `commandError`，不改）。
- 测试的共用：`deactivationWorld`、`preconditions`、`clears`；`deactivationPath` 和 `byAPI`、`byCommand`、`deactivationPaths`；`deactivatingUser`；`refusedWorkspaceSoleAdmin`、`refusedProjectSoleAdmin`；`queryIDs`；`commandInBackground`（`reactivation_races_test.go`，P5a 的 `reactivatingBobWith` 改用它）。

**Tests:**（`bootstrap/deactivation_test.go`；两条路各跑一遍）
- `TestARefusedDeactivationChangesNothing`：五个情形各一个世界：alice（acme 唯一的有效管理员，dave 已结束的管理员不算）`workspace.sole_admin`；bob（Ops、Lab 唯一的管理员）`project.sole_admin`；alice 加入 Lab 之后照样（Ops 里 erin 已结束的管理员不算）；她改为加入 Ops 之后照样（Lab 属 beta）；alice 离开 beta 之后，bob 是 beta 唯一的管理员：`workspace.sole_admin`（beta 不是他按 id 的第一个工作区）。每次每张表每一行不变（`tableRows(riversOwn)`）。
- `TestADeactivationRefusedAtItsCommitChangesNothing`：停用写的六张表各让提交失败一次：回答是失败（不是契约的问题），每张表不变。
- `TestADeactivationEndsEveryMembership`：bob 的 2 + 5 + 3 行（Solo 已归档也在），先由测试盖上 dave 的写者戳；停用之后每一行恰好变了三列，时刻相同、不早于请求，写者是 bob，角色不变；账户无效；别人的每一行不变；再停用一次：接口 401、命令完成，都不再写这些行；之后 carol（gamma 唯一的有效管理员，也是唯一的有效成员）的停用完成。
- `bootstrap/users_test.go`：`TestUsersCommands` 的停用一行；`cmd/nerve/users_test.go`：成功的一行，`TestUsersCommandsFail` 加 nia（acme 唯一的管理员，oto 是成员，由 SQL 在 5 秒的期限内写入）：打印 `workspace.sole_admin` 的说明，退出码 1。

- [ ] **Step 1: 命令**

`server/internal/bootstrap/users.go`（修改，2 处）：

````old server/internal/bootstrap/users.go
// DeactivateUser is `nerve users deactivate` (M2 decision 3).
````
````new server/internal/bootstrap/users.go
// DeactivateUser is `nerve users deactivate` (M2 decision 3): its line says
// what it ended and the way back, the two commands of M3 design 3.9, 8.7.
````

````old server/internal/bootstrap/users.go
		return fmt.Sprintf("deactivated %s: revoked %d sessions", r.Email, r.Sessions), err
````
````new server/internal/bootstrap/users.go
		return fmt.Sprintf("deactivated %s: revoked %d sessions and ended its memberships; to bring it back, run nerve users activate, "+
			"then nerve workspaces reactivate-member in each workspace", r.Email, r.Sessions), err
````

`server/cmd/nerve/users.go`（修改，2 处）：

````old server/cmd/nerve/users.go
		userCommand(load, stdin, "deactivate", "Deactivate an account and revoke its sessions; its API tokens stay",
````
````new server/cmd/nerve/users.go
		userCommand(load, stdin, "deactivate", "Deactivate an account, revoke its sessions and end its memberships; its API tokens stay",
````

````old server/cmd/nerve/users.go
		userCommand(load, stdin, "activate", "Activate an account; its unexpired API tokens work again, so run reset-password too if it may be compromised",
````
````new server/cmd/nerve/users.go
		userCommand(load, stdin, "activate", "Activate an account; its unexpired API tokens work again, so run reset-password too if it may be "+
			"compromised; its memberships stay ended until nerve workspaces reactivate-member",
````

`server/internal/bootstrap/users_test.go`（修改，1 处）：

````old server/internal/bootstrap/users_test.go
			"deactivated carol@new.example: revoked 1 sessions\n",
````
````new server/internal/bootstrap/users_test.go
			"deactivated carol@new.example: revoked 1 sessions and ended its memberships; to bring it back, run nerve users activate, " +
				"then nerve workspaces reactivate-member in each workspace\n",
````

`server/cmd/nerve/users_test.go`（修改，4 处）：

````old server/cmd/nerve/users_test.go
		{[]string{"users", "deactivate", "--email", "lee@corp.com"}, "deactivated lee@corp.com: revoked 0 sessions\n"},
````
````new server/cmd/nerve/users_test.go
		{[]string{"users", "deactivate", "--email", "lee@corp.com"}, "deactivated lee@corp.com: revoked 0 sessions and ended its memberships; " +
			"to bring it back, run nerve users activate, then nerve workspaces reactivate-member in each workspace\n"},
````

````old server/cmd/nerve/users_test.go
// A refused command exits 1 with one line on stderr and nothing on stdout.
````
````new server/cmd/nerve/users_test.go
// A refused command exits 1 with one line on stderr and nothing on stdout.
// nia is acme's only admin and oto its member, so that her deactivation is
// refused (M3 design 3.7 rule 2).
````

````old server/cmd/nerve/users_test.go
	environ, _ := usersDatabase(t)
	if code, _, stderr := executeWithInput(context.Background(), environ, "Tr0ub4dor&3\n", "users", "create", "--email", "nia@corp.com"); code != 0 {
		t.Fatalf("create = %d: %s", code, stderr)
````
````new server/cmd/nerve/users_test.go
	environ, pool := usersDatabase(t)
	for _, email := range []string{"nia@corp.com", "oto@corp.com"} {
		if code, _, stderr := executeWithInput(context.Background(), environ, "Tr0ub4dor&3\n", "users", "create", "--email", email); code != 0 {
			t.Fatalf("create %s = %d: %s", email, code, stderr)
		}
	}
	if code, _, stderr := execute(context.Background(), environ,
		"workspaces", "create", "--slug", "acme", "--name", "Acme", "--admin-email", "nia@corp.com"); code != 0 {
		t.Fatalf("create acme = %d: %s", code, stderr)
	}
	seeding, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := pool.Exec(seeding, `INSERT INTO workspace_members (id, workspace_id, member_id, role, is_active)
		SELECT gen_random_uuid(), w.id, u.id, 15, true FROM workspaces w, users u WHERE w.slug = 'acme' AND u.email = 'oto@corp.com'`); err != nil {
		t.Fatal(err)
````

````old server/cmd/nerve/users_test.go
		{"an unknown account", "", []string{"users", "activate", "--email", "may@corp.com"}, "nerve: No account has this e-mail address.\n"},
````
````new server/cmd/nerve/users_test.go
		{"an unknown account", "", []string{"users", "activate", "--email", "may@corp.com"}, "nerve: No account has this e-mail address.\n"},
		{"a workspace's only admin", "", []string{"users", "deactivate", "--email", "nia@corp.com"},
			"nerve: The workspace would be left without an admin: its only active admin cannot leave it, nor can his membership end while " +
				"it has other active members. It must first be given another admin, or be deleted.\n"},
````

- [ ] **Step 2: 两条路的世界和组合测试**

`server/internal/bootstrap/reactivation_races_test.go`（修改，3 处）：

````old server/internal/bootstrap/reactivation_races_test.go
	"context"
````
````new server/internal/bootstrap/reactivation_races_test.go
	"context"
	"io"
````

````old server/internal/bootstrap/reactivation_races_test.go
}

// bobReactivated is the line of bob's reactivation in acme.
````
````new server/internal/bootstrap/reactivation_races_test.go
}

// commandInBackground runs a command, run, which writes its logs and its
// line, in the background, and hands over its line and error. It takes no
// test, so nothing on its goroutine fails one.
func commandInBackground(run func(logs, out io.Writer) error) <-chan commandRun {
	done := make(chan commandRun, 1)
	go func() {
		var out, logs bytes.Buffer
		err := run(&logs, &out)
		done <- commandRun{out.String(), err}
	}()
	return done
}

// bobReactivated is the line of bob's reactivation in acme.
````

````old server/internal/bootstrap/reactivation_races_test.go
	done := make(chan commandRun, 1)
	go func() {
		var out, logs bytes.Buffer
		err := Workspaces(ctx, cfg, &logs, &out, ReactivateMember("acme", "bob@corp.com"))
		done <- commandRun{out.String(), err}
	}()
	return done
````
````new server/internal/bootstrap/reactivation_races_test.go
	return commandInBackground(func(logs, out io.Writer) error {
		return Workspaces(ctx, cfg, logs, out, ReactivateMember("acme", "bob@corp.com"))
	})
````

`server/internal/bootstrap/deactivation_world_test.go`（新文件，202 行）：

````file server/internal/bootstrap/deactivation_world_test.go
package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// deactivationWorld is endingWorld (ending_world_test.go) and, after it,
// through the API: gamma, carol's workspace, where dave's membership,
// which he took by accepting her invitation, ended by her removal of him;
// her invitation of bob's address to it, which he declined, no member of
// it; dave made acme's admin by alice, then removed by her, an admin whose
// membership ended; bob made beta's admin by alice, so that his roles
// differ across his memberships; Solo archived by bob; Docs, alice's
// project of acme, created after beta's Lab, which bob joined as its
// member, so that his projects' ids cross the workspaces: Web, Ops and Solo
// of acme, Lab of beta, then Docs of acme; acme's id is before beta's. url
// is the database's, for the command line.
type deactivationWorld struct {
	endingWorld
	url          string
	gamma, docs  uuid.UUID
	bobsDeclined uuid.UUID // his invitation to gamma
}

func newDeactivationWorld(t *testing.T) deactivationWorld {
	t.Helper()
	w := deactivationWorld{endingWorld: newEndingWorld(t)}
	w.url = w.pool.Config().ConnString()
	if status, body := call(t, w.contract, http.MethodPost, w.base+"/api/v0/workspaces", w.tokens["carol"], `{"name":"gamma","slug":"gamma"}`); status !=
		http.StatusCreated {
		t.Fatalf("creating gamma = %d %s", status, body)
	}
	w.gamma = w.workspace(t, "gamma")
	answerInvitation(t, w.contract, w.base, w.tokens["dave"], "accept", invite(t, w.contract, w.base, w.tokens["carol"], "gamma", "dave@example.com"),
		http.StatusOK)
	link := invite(t, w.contract, w.base, w.tokens["carol"], "gamma", "bob@example.com")
	answerInvitation(t, w.contract, w.base, w.tokens["bob"], "decline", link, http.StatusNoContent)
	w.bobsDeclined = link.id
	w.docs = createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Docs", "DOCS")
	inBeta := queryIDs(t, w.pool, `SELECT m.id FROM workspace_members m JOIN workspaces s ON s.id = m.workspace_id
		WHERE s.slug = 'beta' AND m.member_id = $1`, w.ids["bob"])[0]
	inGamma := queryIDs(t, w.pool, "SELECT id FROM workspace_members WHERE workspace_id = $1 AND member_id = $2", w.gamma, w.ids["dave"])[0]
	for _, step := range []struct{ token, method, path, body string }{
		{w.tokens["bob"], http.MethodPost, "/api/v0/projects/" + w.docs.String() + "/join", ""},
		{w.tokens["alice"], http.MethodPatch, "/api/v0/workspace-members/" + inBeta.String(), `{"role":20}`},
		{w.tokens["alice"], http.MethodPatch, "/api/v0/workspace-members/" + w.membership(t, "dave").String(), `{"role":20}`},
		{w.tokens["alice"], http.MethodDelete, "/api/v0/workspace-members/" + w.membership(t, "dave").String(), ""},
		{w.tokens["carol"], http.MethodDelete, "/api/v0/workspace-members/" + inGamma.String(), ""},
		{w.tokens["bob"], http.MethodPost, "/api/v0/projects/" + w.solo.String() + "/archive", ""},
	} {
		if status, body := call(t, w.contract, step.method, w.base+step.path, step.token, step.body); status >= http.StatusMultipleChoices {
			t.Fatalf("%s %s = %d %s", step.method, step.path, status, body)
		}
	}
	if acme, beta := w.workspace(t, "acme"), w.workspace(t, "beta"); acme.Compare(beta) >= 0 || w.lab.Compare(w.docs) >= 0 {
		t.Fatalf("acme %s, beta %s, Lab %s, Docs %s: want beta's id after acme's, Docs's after Lab's", acme, beta, w.lab, w.docs)
	}
	w.preconditions(t)
	return w
}

// preconditions checks the rows of deactivationWorld that decide what a
// deactivation does, each read by what makes it the one it is: dave, an
// admin of acme whose membership ended, and a member of gamma whose
// membership ended; bob, a member of acme and beta's admin; Solo archived;
// the invitation to gamma bob declined. Were one missing, a deactivation
// that counted an ended admin as another, or an ended member as another,
// wrote a role, left an archived project out, or deleted pending
// invitations alone, would pass.
func (w deactivationWorld) preconditions(t *testing.T) {
	t.Helper()
	var got string
	if err := w.pool.QueryRow(soon(t), `SELECT concat_ws('; ',
		(SELECT string_agg(s.slug || ' ' || split_part(u.email, '@', 1) || ' ' || m.role || CASE WHEN m.is_active THEN '' ELSE ' ended' END, ', '
			ORDER BY s.slug COLLATE "C", u.email COLLATE "C") FROM workspace_members m JOIN workspaces s ON s.id = m.workspace_id
			JOIN users u ON u.id = m.member_id WHERE u.email IN ('bob@example.com', 'dave@example.com')),
		(SELECT 'Solo archived' FROM projects WHERE id = $1 AND archived_at IS NOT NULL),
		(SELECT 'gamma declined' FROM workspace_member_invites WHERE id = $2 AND responded_at IS NOT NULL AND NOT accepted AND deleted_at IS NULL))`,
		w.solo, w.bobsDeclined).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if want := "acme bob 15, acme dave 20 ended, beta bob 20, gamma dave 15 ended; Solo archived; gamma declined"; got != want {
		t.Fatalf("the rows a deactivation decides on: %s; want %s", got, want)
	}
}

// clears has alice join projects, as their admin, so that bob is not their
// only admin: Ops and Lab both, and his deactivation goes through.
func (w deactivationWorld) clears(t *testing.T, projects ...uuid.UUID) {
	t.Helper()
	for _, p := range projects {
		if status, body := call(t, w.contract, http.MethodPost, w.base+"/api/v0/projects/"+p.String()+"/join", w.tokens["alice"],
			""); status != http.StatusOK {
			t.Fatalf("alice's joining %s = %d %s", p, status, body)
		}
	}
}

// deactivationPath is one path of a deactivation of name's account on w:
// deactivateMe with his bearer token, or `nerve users deactivate` of his
// address, typed in capitals between spaces, which the command normalizes
// to the one his invitations are to; as bootstrap wires each. sent starts
// it in the background, with no test on its goroutine, and returns its
// answer, which waits for its end on the test's goroutine: "" when it is
// done; else its problem's code and detail, "code: detail", the API's
// (whose status it checks against the code's) or the command's error's,
// which the command prints; "500" for a failure, which is no problem of
// the contract.
type deactivationPath struct {
	name string
	sent func(w deactivationWorld, t *testing.T, name string) (answer func() string)
}

// deactivate is p's deactivation of name's account, sent and answered.
func (p deactivationPath) deactivate(w deactivationWorld, t *testing.T, name string) string {
	t.Helper()
	return p.sent(w, t, name)()
}

// byAPI and byCommand are the deactivation's two paths (M3 design 9.3).
var (
	byAPI = deactivationPath{"deactivateMe", func(w deactivationWorld, t *testing.T, name string) func() string {
		t.Helper()
		req := newRequest(t, http.MethodPost, w.base+"/api/v0/me/deactivate", w.tokens[name], nil)
		w.contract.CheckRequest(t, req)
		answered := sendInBackground(req)
		return func() string {
			t.Helper()
			a := receiveWithin(t, answered, 10*time.Second, "the answer to deactivateMe")
			if a.err != nil {
				t.Fatal(a.err)
			}
			w.contract.CheckResponse(t, req, a.res)
			if a.res.StatusCode == http.StatusNoContent && len(a.body) == 0 {
				return ""
			}
			var p httpserver.Problem
			if err := json.Unmarshal(a.body, &p); err != nil {
				t.Fatalf("deactivateMe = %d %s, want a problem", a.res.StatusCode, a.body)
			}
			if p.Code == "internal_error" && a.res.StatusCode == http.StatusInternalServerError {
				return "500"
			}
			if want := map[string]int{"workspace.sole_admin": http.StatusConflict, "project.sole_admin": http.StatusConflict,
				"unauthorized": http.StatusUnauthorized}[p.Code]; a.res.StatusCode != want {
				t.Errorf("deactivateMe = %d %s; want %d for %s", a.res.StatusCode, a.body, want, p.Code)
			}
			return p.Code + ": " + p.Detail
		}
	}}
	byCommand = deactivationPath{"nerve users deactivate", func(w deactivationWorld, t *testing.T, name string) func() string {
		t.Helper()
		done := deactivatingUser(t.Context(), testConfig(t, w.url, false), " "+strings.ToUpper(name)+"@EXAMPLE.COM ")
		return func() string {
			t.Helper()
			run := receiveWithin(t, done, 10*time.Second, "the end of nerve users deactivate")
			var se *shared.Error
			switch {
			case run.err == nil:
				if line := regexp.MustCompile("^deactivated " + name + "@example.com: revoked [0-9]+ sessions and ended its memberships; to " +
					"bring it back, run nerve users activate, then nerve workspaces reactivate-member in each workspace\n$"); !line.MatchString(run.out) {
					t.Errorf("nerve users deactivate printed %q, want %s", run.out, line)
				}
				return ""
			case run.out != "":
				t.Errorf("nerve users deactivate printed %q, refused; want nothing", run.out)
			case errors.As(run.err, &se):
				return se.Code + ": " + run.err.Error()
			}
			return "500"
		}
	}}
	deactivationPaths = []deactivationPath{byAPI, byCommand}
)

// deactivatingUser runs `nerve users deactivate --email email` on cfg in
// the background until ctx ends, and hands over its line and error.
func deactivatingUser(ctx context.Context, cfg config.Config, email string) <-chan commandRun {
	return commandInBackground(func(logs, out io.Writer) error { return Users(ctx, cfg, logs, out, DeactivateUser(email)) })
}

// The two refusals of a deactivation as each path answers them: the
// problem's code and its detail, which the command prints as its line.
const (
	refusedWorkspaceSoleAdmin = "workspace.sole_admin: The workspace would be left without an admin: its only active admin cannot leave it, " +
		"nor can his membership end while it has other active members. It must first be given another admin, or be deleted."
	refusedProjectSoleAdmin = "project.sole_admin: The project would be left without an admin: its only active admin cannot leave it, " +
		"nor can his membership end while it has other active members. It must first be given another admin, or be deleted."
)
````

`server/internal/bootstrap/deactivation_test.go`（新文件，216 行）：

````file server/internal/bootstrap/deactivation_test.go
package bootstrap

import (
	"maps"
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"
)

// A deactivation refused leaves every row of every table as it was, on both
// paths (M3 design 3.7 rule 2, 3.9, 9.3): its refusal comes after its writes
// of the account, its profile and its sessions, and the workspace's refusal
// after its workspaces' locks, the project's after its deletion of the
// invitations and its end of the workspace memberships, so that each of
// those is shown rolled back. Each case has a world of its own.
//   - alice, acme's only active admin, dave's membership as its admin having
//     ended, which counts for nothing, with other active members:
//     workspace.sole_admin, its detail on both paths; beta, where bob is an
//     admin too, refuses nothing.
//   - bob, the only active admin of Ops, of acme, beside carol, its member,
//     and erin, an admin whose membership ended, and of Lab, of beta,
//     beside carol: project.sole_admin. So once alice has joined Lab, as its
//     admin: erin counts for nothing in Ops. So once she has joined Ops
//     instead: Lab is of beta, another of his workspaces, which the check
//     covers too.
//   - bob, once alice has left beta, its only active admin beside carol, its
//     member: workspace.sole_admin, though acme, his first workspace by id,
//     has alice; the check asks of each of his workspaces, and the
//     workspace's refusal comes before the project's.
func TestARefusedDeactivationChangesNothing(t *testing.T) {
	for _, p := range deactivationPaths {
		for _, tt := range []struct {
			name, who, want string
			before          func(w deactivationWorld, t *testing.T)
		}{
			{"acme's only admin", "alice", refusedWorkspaceSoleAdmin, nil},
			{"Ops's and Lab's only admin", "bob", refusedProjectSoleAdmin, nil},
			{"Ops's only admin", "bob", refusedProjectSoleAdmin, func(w deactivationWorld, t *testing.T) { w.clears(t, w.lab) }},
			{"Lab's only admin", "bob", refusedProjectSoleAdmin, func(w deactivationWorld, t *testing.T) { w.clears(t, w.ops) }},
			{"beta's only admin", "bob", refusedWorkspaceSoleAdmin, func(w deactivationWorld, t *testing.T) {
				if status, body := w.leave(t, "beta", "alice"); status != http.StatusNoContent {
					t.Fatalf("alice's leaving beta = %d %s", status, body)
				}
			}},
		} {
			t.Run(p.name+", "+tt.name, func(t *testing.T) {
				w := newDeactivationWorld(t)
				if tt.before != nil {
					tt.before(w, t)
				}
				before := tableRows(t, w.pool, riversOwn)
				if got := p.deactivate(w, t, tt.who); got != tt.want {
					t.Errorf("deactivating %s = %q, want %q", tt.who, got, tt.want)
				}
				if after := tableRows(t, w.pool, riversOwn); !maps.Equal(after, before) {
					t.Errorf("the tables after %s's refused deactivation changed:\n%v\nwant them as they were:\n%v", tt.who, after, before)
				}
			})
		}
	}
}

// A deactivation is one transaction, on both paths (M3 design 3.9): bob's,
// once alice has joined Ops and Lab, refused at its commit, after every
// statement ran, for each table it writes, fails, no problem of the
// contract, and no row of any table changes: no step wrote in a
// transaction of its own.
func TestADeactivationRefusedAtItsCommitChangesNothing(t *testing.T) {
	for _, p := range deactivationPaths {
		t.Run(p.name, func(t *testing.T) {
			w := newDeactivationWorld(t)
			w.clears(t, w.ops, w.lab)
			before := tableRows(t, w.pool, riversOwn)
			for _, table := range []string{"users", "profiles", "auth_sessions", "workspace_member_invites", "workspace_members", "project_members"} {
				restore := refusingCommits(t, w.pool, table)
				got := p.deactivate(w, t, "bob")
				restore()
				if after := tableRows(t, w.pool, riversOwn); got != "500" || !maps.Equal(after, before) {
					t.Errorf("the deactivation refused at its commit for %s = %q; want a failure and every table as it was", table, got)
				}
			}
		})
	}
}

// A deactivation ends every membership of the account and deletes every
// invitation to its address, on both paths, in one transaction at one
// moment, as the account (M3 design 3.6 convention 6, 3.8, 3.9, 9.3): once
// alice has joined Ops and Lab, bob's deactivation ends his memberships of
// acme and beta, and of Web, Ops, Solo, Lab and Docs, across the two
// workspaces, each row kept with its role; it deletes his invitations,
// pending ones to acme and beta and the one to gamma he declined, not
// being its member; each of those rows at one moment no earlier than the
// request, and as bob. Each was stamped by the test as last written by
// dave before, so that the claim of bob's writing can fail; his roles
// differ, a member's of acme and Docs, an admin's of beta and of his other
// projects, Solo among them archived (3.9 ends those too), so that a role
// written over shows. His account is inactive; every other row of every
// table, alice's, carol's, dave's and erin's memberships and the
// invitations to their addresses among them, is as it was. Deactivated, he
// is deactivated again: deactivateMe, his session revoked, is 401; the
// command goes through; neither writes a row of his the first one ended
// or deleted. Then carol's deactivation goes through: gamma's only active
// admin, she is its only active member too, dave's membership having ended
// (3.7 rule 2), and its membership ends with her others.
func TestADeactivationEndsEveryMembership(t *testing.T) {
	for _, p := range deactivationPaths {
		t.Run(p.name, func(t *testing.T) {
			w := newDeactivationWorld(t)
			w.clears(t, w.ops, w.lab)
			bob := w.ids["bob"]
			var written []uuid.UUID
			var tables []string
			for _, row := range []struct {
				table string
				sql   string
				args  []any
			}{
				{"workspace_members", "SELECT id FROM workspace_members WHERE member_id = $1 AND deleted_at IS NULL ORDER BY id", []any{bob}},
				{"project_members", "SELECT id FROM project_members WHERE member_id = $1 AND deleted_at IS NULL ORDER BY id", []any{bob}},
				{"workspace_member_invites", "SELECT id FROM workspace_member_invites WHERE email = 'bob@example.com' AND deleted_at IS NULL ORDER BY id",
					nil},
			} {
				ids := queryIDs(t, w.pool, row.sql, row.args...)
				written = append(written, ids...)
				tables = append(tables, slices.Repeat([]string{row.table}, len(ids))...)
			}
			if got := len(written); got != 2+5+3 {
				t.Fatalf("bob has %d memberships and invitations to his address; want 2 of workspaces, 5 of projects, 3 invitations", got)
			}
			rowsBefore := make([]map[string]any, len(written))
			for i, id := range written {
				if tag, err := w.pool.Exec(soon(t), "UPDATE "+tables[i]+" SET updated_by_id = $2 WHERE id = $1", id, w.ids["dave"]); err != nil ||
					tag.RowsAffected() != 1 {
					t.Fatalf("%s %s last written by dave: %v, %v", tables[i], id, tag, err)
				}
				rowsBefore[i] = rowJSON(t, w.pool, tables[i], id)
			}
			own := slices.Concat([]uuid.UUID{bob}, queryIDs(t, w.pool, "SELECT id FROM profiles WHERE user_id = $1", bob),
				queryIDs(t, w.pool, "SELECT id FROM auth_sessions WHERE user_id = $1", bob))
			others := rowsBut(t, w.pool, slices.Concat(written, own))
			started := time.Now()

			if got := p.deactivate(w, t, "bob"); got != "" {
				t.Fatalf("deactivating bob = %q, want it done", got)
			}

			moment, _ := rowJSON(t, w.pool, "workspace_member_invites", w.bobsDeclined)["deleted_at"].(string)
			if at, err := time.Parse(time.RFC3339Nano, moment); err != nil || at.Before(started.Truncate(time.Microsecond)) {
				t.Errorf("the declined invitation deleted at %q (%v); want a moment no earlier than the request, %v", moment, err, started)
			}
			for i, id := range written {
				want := maps.Clone(rowsBefore[i])
				want["updated_at"], want["updated_by_id"] = moment, bob.String()
				if tables[i] == "workspace_member_invites" {
					want["deleted_at"] = moment
				} else {
					want["is_active"] = false
				}
				if after := rowJSON(t, w.pool, tables[i], id); !maps.Equal(after, want) {
					t.Errorf("%s %s after the deactivation:\n%v\nwant\n%v", tables[i], id, after, want)
				}
			}
			if active, _ := rowJSON(t, w.pool, "users", bob)["is_active"].(bool); active {
				t.Error("bob's account after the deactivation: active; want it inactive")
			}
			if after := rowsBut(t, w.pool, slices.Concat(written, own)); !maps.Equal(after, others) {
				t.Errorf("every other row after the deactivation:\n%v\nwant them as they were:\n%v", after, others)
			}
			ended := make([]map[string]any, len(written))
			for i, id := range written {
				ended[i] = rowJSON(t, w.pool, tables[i], id)
			}
			if got, want := p.deactivate(w, t, "bob"), map[string]string{byAPI.name: "unauthorized: The bearer token is invalid or has expired."}[p.name]; got != want {
				t.Errorf("deactivating bob again = %q, want %q", got, want)
			}
			for i, id := range written {
				if after := rowJSON(t, w.pool, tables[i], id); !maps.Equal(after, ended[i]) {
					t.Errorf("%s %s after deactivating bob again:\n%v\nwant it as his deactivation left it:\n%v", tables[i], id, after, ended[i])
				}
			}
			if got := p.deactivate(w, t, "carol"); got != "" {
				t.Fatalf("deactivating carol, gamma's only active member = %q, want it done", got)
			}
			if left := queryIDs(t, w.pool, "SELECT id FROM workspace_members WHERE member_id = $1 AND is_active", w.ids["carol"]); len(left) != 0 {
				t.Errorf("carol's active memberships after her deactivation: %v; want none, gamma's ended too", left)
			}
		})
	}
}

// queryIDs is the ids sql reads on pool, in its order.
func queryIDs(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) []uuid.UUID {
	t.Helper()
	rows, err := pool.Query(soon(t), sql, args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return ids
}
````

- [ ] **Step 3: 测试和 lint**

Run: `go -C server test -count=1 ./internal/bootstrap/ ./cmd/nerve/`
Expected: 全部 `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 4: 提交**

```bash
git add server/cmd/nerve/users.go server/cmd/nerve/users_test.go server/internal/bootstrap/deactivation_test.go server/internal/bootstrap/deactivation_world_test.go server/internal/bootstrap/reactivation_races_test.go server/internal/bootstrap/users.go server/internal/bootstrap/users_test.go
```
```bash
git commit -m "test(M3/P6): both deactivation paths, refused with every table unchanged, refused at their commit, and ending every membership at one moment

deactivationWorld spans three workspaces and five projects of the
deactivated account, his roles varied, an archived project, a declined
invitation and admins whose memberships ended. Through deactivateMe
and through nerve users deactivate, each refusal leaves every row of
every table, a commit refused for any table it writes changes nothing,
and a deactivation ends every membership and deletes every invitation
to his address, by him at one moment, and nothing of anyone else. The
command's line says what it ended and the way back.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A；`mutants_p6.py` 的编号，"层"是它被发现的每一层；端到端是单独运行的故事）：

| 变异 | 改坏 | 必须失败的测试 | 层 |
|---|---|---|---|
| `s6-cli-line` | 命令的一行少了回来的第二步的"在每个工作区" | `TestADeactivationEndsEveryMembership`、`TestADeactivationFindsWhatChangedMeanwhile`（Task 6 起）、`TestAnInterruptedDeactivationChangesNothing`（Task 6 起）、`TestEachLockOfADeactivationIsItsStrength`（Task 6 起） 等 8 个、W9（Task 9 起） | 组合；端到端 |
| `s10-dave-member` | 世界里 dave 没有被改为 acme 的管理员 | `TestADeactivationEndsEveryMembership`、`TestADeactivationFindsWhatChangedMeanwhile`（Task 6 起）、`TestADeactivationReadsTheAddressUnderItsLock`（Task 6 起）、`TestADeactivationRefusedAtItsCommitChangesNothing` 等 8 个 | 组合 |
| `s10-dave-stays` | 世界里 dave 没有被移出 acme | `TestADeactivationEndsEveryMembership`、`TestADeactivationFindsWhatChangedMeanwhile`（Task 6 起）、`TestADeactivationReadsTheAddressUnderItsLock`（Task 6 起）、`TestADeactivationRefusedAtItsCommitChangesNothing` 等 8 个 | 组合 |
| `s10-bob-member-of-beta` | 世界里 bob 在 beta 仍是成员（改角色写 15） | `TestADeactivationEndsEveryMembership`、`TestADeactivationFindsWhatChangedMeanwhile`（Task 6 起）、`TestADeactivationReadsTheAddressUnderItsLock`（Task 6 起）、`TestADeactivationRefusedAtItsCommitChangesNothing` 等 8 个 | 组合 |
| `s10-gamma-dave-stays` | 世界里 dave 在 gamma 的成员关系没有结束（改为访客） | `TestADeactivationEndsEveryMembership`、`TestADeactivationFindsWhatChangedMeanwhile`（Task 6 起）、`TestADeactivationReadsTheAddressUnderItsLock`（Task 6 起）、`TestADeactivationRefusedAtItsCommitChangesNothing` 等 8 个 | 组合 |
| `s10-solo-standing` | 世界里 Solo 没有归档 | `TestADeactivationEndsEveryMembership`、`TestADeactivationFindsWhatChangedMeanwhile`（Task 6 起）、`TestADeactivationReadsTheAddressUnderItsLock`（Task 6 起）、`TestADeactivationRefusedAtItsCommitChangesNothing` 等 8 个 | 组合 |
| `s10-gamma-pending` | 世界里 bob 没有忽略 gamma 的邀请 | `TestADeactivationEndsEveryMembership`、`TestADeactivationFindsWhatChangedMeanwhile`（Task 6 起）、`TestADeactivationReadsTheAddressUnderItsLock`（Task 6 起）、`TestADeactivationRefusedAtItsCommitChangesNothing` 等 8 个 | 组合 |

**Done when:** 三个组合测试在两条路上通过；`deactivationWorld.preconditions` 核对世界；命令的一行在 `bootstrap`、`cmd/nerve` 的测试里逐字；`make lint-go`、`make test` 通过。

### Task 6: 每把锁的顺序和强度、事务的连接、命令的中断；等锁期间的变化；锁下的邮箱

**Files:**
- Create: `server/internal/bootstrap/deactivation_locks_test.go`、`server/internal/bootstrap/deactivation_races_test.go`
- Modify: `server/internal/bootstrap/interleaving_answers_test.go`

**Interfaces:**
- 没有产品代码。测试的共用：`setBobsEmail(ctx, pool, sessions)`（`interleaving_answers_test.go`：`nerve users set-email` 把 bob 的地址改为 `robert@example.com`，会话一步可以换成被 gate 停住的；P3 的交错 12 同用）；`deactivationWorld.endings`（bob 的每一个成员关系和发给他新旧地址的每一份邀请，按字节排序，每行写明最后写它的人，清扫 39、40）、`bobsStanding`。

**Tests:**（两条路各跑一遍，除非另说）
- `TestEachLockOfADeactivationIsItsStrength`（`deactivation_locks_test.go`）：alice 加入 Ops、Lab，carol 建 able（bob 加入，id 最后、slug 最前）和 delta（bob 加入后被她移出，`carolsWithBob`）之后，六个事务 `FOR SHARE` 持有他的账户、beta、gamma 的邀请、他在 beta 的成员关系、Lab、他在 Docs 的成员关系；acme 的行、Lab 的行和成员行先重写一遍，排到 beta、Docs 之后（不带 `ORDER BY` 的语句按表的顺序先碰到 beta、Docs，按 slug 的索引先碰到 able；原型上的计划是 slug 的索引）。逐个放开：停用依次等在 `users`、`workspaces`、`workspace_member_invites`、`workspace_members`、`projects`、`project_members` 上（`pgtest.WaitForLockWaitOn`），每一步 `lockOn` 读出十一行最强的锁：账户、acme、beta、able、邀请、他的成员关系、Web、Lab、Docs 各是 `FOR NO KEY UPDATE`（不更强、不更弱），gamma、delta 从不加锁（他已结束的成员关系不算）；等 beta 时已持有 acme、还没有 able；等 Lab 时已持有 Web、Ops、Solo，还没有 Docs。完成的时刻不早于 beta 放开的时刻（3.3）。
- `TestAnInterruptedDeactivationChangesNothing`（命令）：他在 Lab 的成员行被持有，配置给服务的请求期限 200 毫秒；过了两倍期限命令仍在等；中断之后失败、没有输出，每张表不变；放开之后再执行，完成。
- `TestTheDeactivationRunsOnItsTransactionsConnection`（命令）：连接池只有一个连接，5 秒的期限之内完成，输出逐字（走池的语句会等第二个连接到期限）。
- `TestADeactivationFindsWhatChangedMeanwhile`（`deactivation_races_test.go`，三个情形 × 两条路）：alice 的写（删除 acme、删除 Docs、移出 bob）持有一个工作区的锁、等一个被 `FOR SHARE` 持有的行，停用等那个工作区的行（两方都不写 `workspaces`，探针只由那一把锁的等待满足）；放开之后写 204、停用完成，`endings` 逐行核对：acme 跳过（不 404、不失败，复核 spike 15），它的行照删除留下；Docs 照删除留下；移出留下的结束和删除仍是 alice 的。
- `TestADeactivationReadsTheAddressUnderItsLock`：改地址停在撤销会话之前、持着账户行，停用等账户行，改地址提交：`deactivateMe`（PAT）删除发给 `robert@` 的邀请，发给 `bob@` 的不动，成员关系由 robert 结束；命令按旧地址答 `identity.account_not_found`，他什么都不变（复核 M5）。

- [ ] **Step 1: 锁的顺序和强度、连接、中断**

`server/internal/bootstrap/deactivation_locks_test.go`（新文件，227 行）：

````file server/internal/bootstrap/deactivation_locks_test.go
package bootstrap

import (
	"context"
	"maps"
	"net/http"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// Each lock of a deactivation is taken in the lock table's order, at its
// strength, on both paths (M3 design 3.6's lock table, convention 1,
// convention 6, the global order): his account first, then every workspace
// of his in id order, then the invitations to his address and his
// memberships, then every project of his in id order across the
// workspaces, then his memberships of them. alice has joined Ops and Lab;
// carol has made able, which bob joined, and delta, which he joined and
// she removed him from; acme and Lab have been written again. So acme's
// row lies after beta's, able, made last, comes first by slug, and Lab's
// row and memberships lie after Docs's: the order is the ids', not the
// rows' or the slugs' (a statement without its ORDER BY reaches beta or
// able, and Docs, first). Other transactions hold, FOR SHARE, his account,
// beta, the invitation to gamma he declined, his membership of beta, Lab
// and his membership of Docs, which the deactivation's statements wait for
// in turn; they let go one at a time, and lockOn reads each row's
// strongest lock then. The deactivation waits for his account, holding
// nothing; then for beta, holding his account and acme, the first
// workspace by id, and not able, the last; then for the invitation,
// holding beta and able too; then for his membership, holding the
// invitation; then for Lab, holding his membership, Web, Ops and Solo, of
// acme, before Lab, of beta, by id, and not Docs, of acme, after it; then
// for his membership of Docs, holding Lab and Docs; each FOR NO KEY
// UPDATE, no stronger, no weaker; never gamma, where he is no member, nor
// delta, where his membership ended. Then it is done, at a moment no
// earlier than beta's release: it read the clock under its last
// workspace's lock (3.3).
func TestEachLockOfADeactivationIsItsStrength(t *testing.T) {
	for _, p := range deactivationPaths {
		t.Run(p.name, func(t *testing.T) {
			w := newDeactivationWorld(t)
			w.clears(t, w.ops, w.lab)
			able, delta := w.carolsWithBob(t, "able", false), w.carolsWithBob(t, "delta", true)
			bob, acme, beta := w.ids["bob"], w.workspace(t, "acme"), w.workspace(t, "beta")
			inBeta := queryIDs(t, w.pool, "SELECT id FROM workspace_members WHERE workspace_id = $1 AND member_id = $2", beta, bob)[0]
			inDocs := projectMemberships(t, w.pool, bob, w.docs)[0]
			for _, again := range []struct {
				sql string
				id  uuid.UUID
			}{
				{"UPDATE workspaces SET updated_at = updated_at WHERE id = $1", acme},
				{"UPDATE projects SET updated_at = updated_at WHERE id = $1", w.lab},
				{"UPDATE project_members SET updated_at = updated_at WHERE project_id = $1", w.lab},
			} {
				if _, err := w.pool.Exec(soon(t), again.sql, again.id); err != nil {
					t.Fatal(err)
				}
			}
			rows := []struct {
				name, from string
				id         uuid.UUID
			}{
				{"his account", "users", bob}, {"acme", "workspaces", acme}, {"beta", "workspaces", beta}, {"able", "workspaces", able},
				{"gamma", "workspaces", w.gamma}, {"delta", "workspaces", delta}, {"the invitation", "workspace_member_invites", w.bobsDeclined},
				{"his membership", "workspace_members", inBeta}, {"Web", "projects", w.web}, {"Lab", "projects", w.lab},
				{"Docs", "projects", w.docs},
			}
			locks := func() string {
				t.Helper()
				held := make([]string, len(rows))
				for i, r := range rows {
					held[i] = r.name + " " + lockOn(t, w.pool, r.from+" WHERE id = $1", r.id)
				}
				return strings.Join(held, ", ")
			}
			holdsDocs := holding(t, w.pool, "SELECT 1 FROM project_members WHERE id = $1 FOR SHARE", inDocs)
			holdsLab := holding(t, w.pool, "SELECT 1 FROM projects WHERE id = $1 FOR SHARE", w.lab)
			holdsMembership := holding(t, w.pool, "SELECT 1 FROM workspace_members WHERE id = $1 FOR SHARE", inBeta)
			holdsInvitation := holding(t, w.pool, "SELECT 1 FROM workspace_member_invites WHERE id = $1 FOR SHARE", w.bobsDeclined)
			holdsBeta := holding(t, w.pool, "SELECT 1 FROM workspaces WHERE id = $1 FOR SHARE", beta)
			holdsAccount := holding(t, w.pool, "SELECT 1 FROM users WHERE id = $1 FOR SHARE", bob)
			answer := p.sent(w, t, "bob")
			var released time.Time
			// Each step lets go of one holder and names the table the
			// deactivation then waits on, and its locks; the FOR SHARE ones
			// are the holders'. A lock it took on a row held FOR SHARE before
			// its turn would hide behind the holder's, and show only as the
			// probe of the step that waits on that row's table timing out
			// (pgtest.WaitForLockWaitOn).
			const never = "gamma no lock, delta no lock"
			for _, step := range []struct {
				release pgx.Tx
				waitsOn string
				want    string
			}{
				{nil, "users", "his account FOR SHARE, acme no lock, beta FOR SHARE, able no lock, " + never + ", the invitation FOR SHARE, " +
					"his membership FOR SHARE, Web no lock, Lab FOR SHARE, Docs no lock"},
				{holdsAccount, "workspaces", "his account FOR NO KEY UPDATE, acme FOR NO KEY UPDATE, beta FOR SHARE, able no lock, " + never +
					", the invitation FOR SHARE, his membership FOR SHARE, Web no lock, Lab FOR SHARE, Docs no lock"},
				{holdsBeta, "workspace_member_invites", "his account FOR NO KEY UPDATE, acme FOR NO KEY UPDATE, beta FOR NO KEY UPDATE, " +
					"able FOR NO KEY UPDATE, " + never + ", the invitation FOR SHARE, his membership FOR SHARE, Web no lock, Lab FOR SHARE, " +
					"Docs no lock"},
				{holdsInvitation, "workspace_members", "his account FOR NO KEY UPDATE, acme FOR NO KEY UPDATE, beta FOR NO KEY UPDATE, " +
					"able FOR NO KEY UPDATE, " + never + ", the invitation FOR NO KEY UPDATE, his membership FOR SHARE, Web no lock, " +
					"Lab FOR SHARE, Docs no lock"},
				{holdsMembership, "projects", "his account FOR NO KEY UPDATE, acme FOR NO KEY UPDATE, beta FOR NO KEY UPDATE, " +
					"able FOR NO KEY UPDATE, " + never + ", the invitation FOR NO KEY UPDATE, his membership FOR NO KEY UPDATE, " +
					"Web FOR NO KEY UPDATE, Lab FOR SHARE, Docs no lock"},
				{holdsLab, "project_members", "his account FOR NO KEY UPDATE, acme FOR NO KEY UPDATE, beta FOR NO KEY UPDATE, " +
					"able FOR NO KEY UPDATE, " + never + ", the invitation FOR NO KEY UPDATE, his membership FOR NO KEY UPDATE, " +
					"Web FOR NO KEY UPDATE, Lab FOR NO KEY UPDATE, Docs FOR NO KEY UPDATE"},
			} {
				if step.release != nil {
					if step.release == holdsBeta {
						released = time.Now()
					}
					if err := step.release.Rollback(context.Background()); err != nil {
						t.Fatal(err)
					}
				}
				pgtest.WaitForLockWaitOn(t, w.pool, step.waitsOn, 5*time.Second)
				if got := locks(); got != step.want {
					t.Errorf("the deactivation waiting on %s:\n%s\nwant\n%s", step.waitsOn, got, step.want)
				}
			}
			if err := holdsDocs.Rollback(context.Background()); err != nil {
				t.Fatal(err)
			}

			if got := answer(); got != "" {
				t.Fatalf("the deactivation = %q, want it done", got)
			}
			moment, _ := rowJSON(t, w.pool, "workspace_member_invites", w.bobsDeclined)["deleted_at"].(string)
			if at, err := time.Parse(time.RFC3339Nano, moment); err != nil || at.Before(released.Truncate(time.Microsecond)) {
				t.Errorf("the deactivation's moment %q (%v); want one no earlier than beta's release, %v", moment, err, released)
			}
		})
	}
}

// carolsWithBob is carol's new workspace slug, which bob joined by
// accepting her invitation, and, when removed, she then removed him from:
// its id.
func (w deactivationWorld) carolsWithBob(t *testing.T, slug string, removed bool) uuid.UUID {
	t.Helper()
	if status, body := call(t, w.contract, http.MethodPost, w.base+"/api/v0/workspaces", w.tokens["carol"],
		`{"name":"`+slug+`","slug":"`+slug+`"}`); status != http.StatusCreated {
		t.Fatalf("creating %s = %d %s", slug, status, body)
	}
	id := w.workspace(t, slug)
	answerInvitation(t, w.contract, w.base, w.tokens["bob"], "accept", invite(t, w.contract, w.base, w.tokens["carol"], slug, "bob@example.com"),
		http.StatusOK)
	if removed {
		bobs := queryIDs(t, w.pool, "SELECT id FROM workspace_members WHERE workspace_id = $1 AND member_id = $2", id, w.ids["bob"])[0]
		if status, body := call(t, w.contract, http.MethodDelete, w.base+"/api/v0/workspace-members/"+bobs.String(), w.tokens["carol"],
			""); status != http.StatusNoContent {
			t.Fatalf("carol's removal of bob from %s = %d %s", slug, status, body)
		}
	}
	return id
}

// The command has no deadline of its own (M3 design 3.9, README): while
// bob's membership of Lab is held it waits, still after twice the request
// timeout its configuration gives the server, having written his account,
// his profile, his sessions, the invitations to his address and his
// memberships of acme and beta; and an interruption, its context's end as
// SIGINT or SIGTERM ends it (cmd/nerve), rolls it all back: it fails, and
// no row of any table changes. Run again once the row is free, it
// deactivates bob.
func TestAnInterruptedDeactivationChangesNothing(t *testing.T) {
	w := newDeactivationWorld(t)
	w.clears(t, w.ops, w.lab)
	before := tableRows(t, w.pool, riversOwn)
	holdsLab := holding(t, w.pool, "SELECT 1 FROM project_members WHERE id = $1 FOR SHARE", projectMemberships(t, w.pool, w.ids["bob"], w.lab)[0])
	ctx, interrupt := context.WithCancel(context.Background())
	defer interrupt()
	cfg := testConfig(t, w.url, false)
	cfg.Server.RequestTimeout = 200 * time.Millisecond
	done := deactivatingUser(ctx, cfg, "bob@example.com")
	pgtest.WaitForLockWaitOn(t, w.pool, "project_members", 5*time.Second)
	select {
	case run := <-done:
		t.Fatalf("the deactivation ended while bob's membership of Lab was held = %q, %v; want it waiting until interrupted", run.out, run.err)
	case <-time.After(2 * cfg.Server.RequestTimeout):
	}
	interrupt()

	if run := receiveWithin(t, done, 10*time.Second, "the end of nerve users deactivate"); run.err == nil || run.out != "" {
		t.Errorf("the deactivation interrupted = %q, %v; want no line and an error", run.out, run.err)
	}
	if err := holdsLab.Rollback(context.Background()); err != nil {
		t.Fatal(err)
	}
	if after := tableRows(t, w.pool, riversOwn); !maps.Equal(after, before) {
		t.Errorf("the tables after the interrupted deactivation changed:\n%v\nwant them as they were:\n%v", after, before)
	}
	if got := byCommand.deactivate(w, t, "bob"); got != "" {
		t.Errorf("the deactivation again = %q, want it done", got)
	}
}

// The deactivation runs every statement on its transaction's connection
// (M3 design 3.6 convention 2, 3.9): identity's, the Deactivator's and the
// cascade's. The command's composition runs on a pool of one connection: a
// statement sent through the pool rather than the transaction would wait
// for a second connection until the command's context ends, after 5
// seconds, and the command fail.
func TestTheDeactivationRunsOnItsTransactionsConnection(t *testing.T) {
	w := newDeactivationWorld(t)
	w.clears(t, w.ops, w.lab)
	cfg := testConfig(t, w.url, false)
	cfg.Database.MaxConns = 1
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	want := "deactivated bob@example.com: revoked 1 sessions and ended its memberships; to bring it back, run nerve users activate, then " +
		"nerve workspaces reactivate-member in each workspace\n"
	if run := receiveWithin(t, deactivatingUser(ctx, cfg, "bob@example.com"), 10*time.Second, "the end of nerve users deactivate"); run.out != want ||
		run.err != nil {
		t.Errorf("the deactivation on a pool of one connection = %q, %v; want %q", run.out, run.err, want)
	}
}
````

- [ ] **Step 2: 等锁期间的变化、锁下的邮箱**

`server/internal/bootstrap/interleaving_answers_test.go`（修改，3 处）：

````old server/internal/bootstrap/interleaving_answers_test.go
// setBobsEmail is `nerve users set-email` of bob's address, over sessions.
func (r answerRace) setBobsEmail(ctx context.Context, sessions identityapp.SessionRevoker) error {
	store := identitypg.New(r.pool)
	_, err := identityapp.NewSetEmail(identityapp.SetEmailDeps{Accounts: store, Users: store, Sessions: sessions, Tx: r.tx(),
		Clock: clocktest.At(time.Now()), Logger: slog.New(slog.DiscardHandler)}).Execute(ctx, "bob@example.com", "robert@example.com")
````
````new server/internal/bootstrap/interleaving_answers_test.go
// setBobsEmail is `nerve users set-email` of bob's address,
// bob@example.com to robert@example.com, on pool, over sessions. It takes
// no test, so it runs on any goroutine.
func setBobsEmail(ctx context.Context, pool *pgxpool.Pool, sessions identityapp.SessionRevoker) error {
	store := identitypg.New(pool)
	_, err := identityapp.NewSetEmail(identityapp.SetEmailDeps{Accounts: store, Users: store, Sessions: sessions,
		Tx: postgres.NewTxManager(pool, 2*time.Second), Clock: clocktest.At(time.Now()), Logger: slog.New(slog.DiscardHandler)}).
		Execute(ctx, "bob@example.com", "robert@example.com")
````

````old server/internal/bootstrap/interleaving_answers_test.go
				changed = run(func() error { return r.setBobsEmail(ctx, users) })
````
````new server/internal/bootstrap/interleaving_answers_test.go
				changed = run(func() error { return setBobsEmail(ctx, r.pool, users) })
````

````old server/internal/bootstrap/interleaving_answers_test.go
				changed = run(func() error { return r.setBobsEmail(ctx, gatedSessions{users, g}) })
````
````new server/internal/bootstrap/interleaving_answers_test.go
				changed = run(func() error { return setBobsEmail(ctx, r.pool, gatedSessions{users, g}) })
````

`server/internal/bootstrap/deactivation_races_test.go`（新文件，187 行）：

````file server/internal/bootstrap/deactivation_races_test.go
package bootstrap

import (
	"context"
	"net/http"
	"testing"
	"time"
	"uuid"

	identitypg "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// A deactivation against what another transaction changes meanwhile, on
// both paths (M3 design 3.6's lock table, convention 6, 3.9): what it
// finds once it has a lock, and whose each ending stays.

// endings is every membership of bob's and every invitation to his
// addresses, old and new, but the ones he accepted, which their acceptance
// deleted, one line each, in byte order: the project's name
// or the workspace's slug, or "invitation of <address> to <slug>"; then
// "active", "pending" or "declined" while it stands, else "ended by" or
// "deleted by" and who wrote it last.
func (w deactivationWorld) endings(t *testing.T) string {
	t.Helper()
	var got string
	if err := w.pool.QueryRow(soon(t), `SELECT string_agg(place || ' ' || CASE WHEN deleted THEN 'deleted by ' || by
			WHEN active THEN standing ELSE 'ended by ' || by END, '; ' ORDER BY place COLLATE "C") FROM (
		SELECT s.slug AS place, m.deleted_at IS NOT NULL AS deleted, m.is_active AS active, 'active' AS standing, m.updated_by_id AS writer
			FROM workspace_members m JOIN workspaces s ON s.id = m.workspace_id WHERE m.member_id = $1
		UNION ALL SELECT p.name, m.deleted_at IS NOT NULL, m.is_active, 'active', m.updated_by_id
			FROM project_members m JOIN projects p ON p.id = m.project_id WHERE m.member_id = $1
		UNION ALL SELECT 'invitation of ' || i.email || ' to ' || s.slug, i.deleted_at IS NOT NULL, true,
			CASE WHEN i.responded_at IS NULL THEN 'pending' ELSE 'declined' END, i.updated_by_id
			FROM workspace_member_invites i JOIN workspaces s ON s.id = i.workspace_id WHERE i.email IN ('bob@example.com', 'robert@example.com')
				AND NOT i.accepted) r
		LEFT JOIN LATERAL (SELECT split_part(u.email, '@', 1) AS by FROM users u WHERE u.id = r.writer) b ON true`, w.ids["bob"]).
		Scan(&got); err != nil {
		t.Fatal(err)
	}
	return got
}

// bobsStanding is endings once alice has joined Ops and Lab, before any
// ending.
const bobsStanding = "Docs active; Lab active; Ops active; Solo active; Web active; acme active; beta active; " +
	"invitation of bob@example.com to acme pending; invitation of bob@example.com to beta pending; " +
	"invitation of bob@example.com to gamma declined"

// A deactivation that waits for a workspace's row reads, once it has it,
// what alice's write committed meanwhile, and goes through; each row she
// wrote stays hers (3.9; review spike 15). Once she has joined Ops and Lab,
// her write, through the API, holds acme's row or beta's and waits for a
// row another transaction holds FOR SHARE; bob's deactivation waits for
// that workspace's row: neither writes a row of workspaces before then, so
// only that lock's wait satisfies the probe. Then the holder lets go, her
// write is done, and so is the deactivation.
//   - acme deleted: its deletion holds acme FOR NO KEY UPDATE and waits for
//     the invitation to bob's address in acme. The deactivation leaves acme
//     out, as deleted while its lock waited, no 404 and no failure, and
//     ends the rest; his memberships of acme and of its projects and the
//     invitation to acme are as the deletion left them.
//   - Docs deleted: its deletion holds acme FOR SHARE and waits for Docs's
//     row. The deactivation finds Docs deleted once it has acme: his
//     membership of Docs is as the deletion left it.
//   - bob removed from beta: the removal holds beta FOR NO KEY UPDATE, has
//     ended his membership of beta and deleted the invitation to beta, and
//     waits for his membership of Lab. The deactivation has acme, waits for
//     beta, then finds those ended: the removal's endings stay alice's.
func TestADeactivationFindsWhatChangedMeanwhile(t *testing.T) {
	for _, tt := range []struct {
		name            string
		write           func(w deactivationWorld, t *testing.T) (path string, held uuid.UUID)
		waitsOn, ending string
	}{
		{"acme deleted", func(w deactivationWorld, t *testing.T) (string, uuid.UUID) {
			return "/api/v0/workspaces/acme", w.bobsInvitation
		}, "workspace_member_invites", "Docs deleted by alice; Lab ended by bob; Ops deleted by alice; Solo deleted by alice; " +
			"Web deleted by alice; acme deleted by alice; beta ended by bob; invitation of bob@example.com to acme deleted by alice; " +
			"invitation of bob@example.com to beta deleted by bob; invitation of bob@example.com to gamma deleted by bob"},
		{"Docs deleted", func(w deactivationWorld, t *testing.T) (string, uuid.UUID) {
			return "/api/v0/projects/" + w.docs.String(), w.docs
		}, "projects", "Docs deleted by alice; Lab ended by bob; Ops ended by bob; Solo ended by bob; Web ended by bob; acme ended by bob; " +
			"beta ended by bob; invitation of bob@example.com to acme deleted by bob; invitation of bob@example.com to beta deleted by bob; " +
			"invitation of bob@example.com to gamma deleted by bob"},
		{"bob removed from beta", func(w deactivationWorld, t *testing.T) (string, uuid.UUID) {
			beta := queryIDs(t, w.pool, `SELECT m.id FROM workspace_members m JOIN workspaces s ON s.id = m.workspace_id
				WHERE s.slug = 'beta' AND m.member_id = $1`, w.ids["bob"])[0]
			return "/api/v0/workspace-members/" + beta.String(), projectMemberships(t, w.pool, w.ids["bob"], w.lab)[0]
		}, "project_members", "Docs ended by bob; Lab ended by alice; Ops ended by bob; Solo ended by bob; Web ended by bob; acme ended by bob; " +
			"beta ended by alice; invitation of bob@example.com to acme deleted by bob; invitation of bob@example.com to beta deleted by alice; " +
			"invitation of bob@example.com to gamma deleted by bob"},
	} {
		for _, p := range deactivationPaths {
			t.Run(tt.name+", "+p.name, func(t *testing.T) {
				w := newDeactivationWorld(t)
				w.clears(t, w.ops, w.lab)
				if got := w.endings(t); got != bobsStanding {
					t.Fatalf("bob's memberships and invitations before: %s; want %s", got, bobsStanding)
				}
				path, row := tt.write(w, t)
				holder := holding(t, w.pool, "SELECT 1 FROM "+tt.waitsOn+" WHERE id = $1 FOR SHARE", row)
				req := newRequest(t, http.MethodDelete, w.base+path, w.tokens["alice"], nil)
				w.contract.CheckRequest(t, req)
				written := sendInBackground(req)
				pgtest.WaitForLockWaitOn(t, w.pool, tt.waitsOn, 5*time.Second)
				answer := p.sent(w, t, "bob")
				pgtest.WaitForLockWaitOn(t, w.pool, "workspaces", 5*time.Second)
				if err := holder.Rollback(context.Background()); err != nil {
					t.Fatal(err)
				}

				a := receiveWithin(t, written, 10*time.Second, "the answer to alice's write")
				if a.err != nil {
					t.Fatal(a.err)
				}
				w.contract.CheckResponse(t, req, a.res)
				if a.res.StatusCode != http.StatusNoContent {
					t.Errorf("alice's write = %d %s, want 204", a.res.StatusCode, a.body)
				}
				if got := answer(); got != "" {
					t.Errorf("the deactivation = %q, want it done", got)
				}
				if got := w.endings(t); got != tt.ending {
					t.Errorf("bob's memberships and invitations after both:\n%s\nwant\n%s", got, tt.ending)
				}
			})
		}
	}
}

// The deactivation deletes the invitations to the address it reads under
// his account's lock (M3 design 3.6 convention 1, 3.9; review M5): the
// change of bob's address to robert@example.com, `nerve users set-email`,
// holds his account row, its address written, at its gate, before it
// revokes his sessions; the deactivation waits for that row, and the
// change commits. alice has invited robert@example.com to acme and beta,
// and joined Ops and Lab.
//   - deactivateMe, with his personal access token, which the change
//     leaves: it reads his new address under the lock and deletes the
//     invitations to it; those to bob@example.com, an address no account
//     has now, stay as they were; his memberships end.
//   - `nerve users deactivate --email bob@example.com`: it locks the account
//     by that address and finds none once the change has committed:
//     identity.account_not_found, and nothing of his changes.
func TestADeactivationReadsTheAddressUnderItsLock(t *testing.T) {
	for _, tt := range []struct {
		path            deactivationPath
		answer, endings string
	}{
		{byAPI, "", "Docs ended by robert; Lab ended by robert; Ops ended by robert; Solo ended by robert; Web ended by robert; " +
			"acme ended by robert; beta ended by robert; invitation of bob@example.com to acme pending; " +
			"invitation of bob@example.com to beta pending; invitation of bob@example.com to gamma declined; " +
			"invitation of robert@example.com to acme deleted by robert; invitation of robert@example.com to beta deleted by robert"},
		{byCommand, "identity.account_not_found: No account has this e-mail address.", "Docs active; Lab active; Ops active; " +
			"Solo active; Web active; acme active; beta active; invitation of bob@example.com to acme pending; " +
			"invitation of bob@example.com to beta pending; invitation of bob@example.com to gamma declined; " +
			"invitation of robert@example.com to acme pending; invitation of robert@example.com to beta pending"},
	} {
		t.Run(tt.path.name, func(t *testing.T) {
			w := newDeactivationWorld(t)
			w.clears(t, w.ops, w.lab)
			for _, slug := range []string{"acme", "beta"} {
				invite(t, w.contract, w.base, w.tokens["alice"], slug, "robert@example.com")
			}
			w.tokens["bob"] = createPAT(t, w.contract, w.base, w.tokens["bob"]).Token
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			g := newGate()
			changed := run(func() error { return setBobsEmail(ctx, w.pool, gatedSessions{identitypg.New(w.pool), g}) })
			held(t, ctx, g, changed, "the change of address")
			answer := tt.path.sent(w, t, "bob")
			pgtest.WaitForLockWaitOn(t, w.pool, "users", 5*time.Second)
			close(g.open)

			if err := result(t, ctx, changed, "the change of address"); err != nil {
				t.Fatalf("the change of address = %v, want it done", err)
			}
			if got := answer(); got != tt.answer {
				t.Errorf("the deactivation = %q, want %q", got, tt.answer)
			}
			if got := w.endings(t); got != tt.endings {
				t.Errorf("bob's memberships and invitations after both:\n%s\nwant\n%s", got, tt.endings)
			}
		})
	}
}
````

- [ ] **Step 3: 测试和 lint**

Run: `go -C server test -count=3 -race -run 'TestEachLockOfADeactivationIsItsStrength$|TestAnInterruptedDeactivationChangesNothing$|TestTheDeactivationRunsOnItsTransactionsConnection$|TestADeactivationFindsWhatChangedMeanwhile$|TestADeactivationReadsTheAddressUnderItsLock$|TestAcceptingAndChangingTheAddress$' ./internal/bootstrap/`
Expected: `ok`；输出里没有 `40P01`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 4: 提交**

```bash
git add server/internal/bootstrap/deactivation_locks_test.go server/internal/bootstrap/deactivation_races_test.go server/internal/bootstrap/interleaving_answers_test.go
```
```bash
git commit -m "test(M3/P6): each lock of a deactivation in its order and at its strength, its connection, its interruption, and what changes while it waits

On both paths, a deactivation waits on his account, his workspaces in
id order, the invitations to his address, his memberships, his
projects in id order across the workspaces and his project
memberships, each FOR NO KEY UPDATE, reading the clock after the last
workspace's lock. The command has no deadline of its own and an
interruption changes nothing; every statement runs on the
transaction's connection. A workspace deleted while it waits is left
out, a removal's endings stay the remover's, and a change of address
committed first has the invitations to the new address deleted.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A；`mutants_p6.py` 的编号，"层"是它被发现的每一层；端到端是单独运行的故事）：

| 变异 | 改坏 | 必须失败的测试 | 层 |
|---|---|---|---|
| `o-cli-deadline` | `nerve users` 在服务的请求期限之下运行 | `TestAnInterruptedDeactivationChangesNothing` | 组合 |
| `o-cli-uninterruptible` | `nerve users` 不随它的 context 结束 | `TestAnInterruptedDeactivationChangesNothing` | 组合 |
| `s39-endings-collation` | `endings` 按数据库的排序规则排序 | `TestADeactivationFindsWhatChangedMeanwhile`、`TestADeactivationReadsTheAddressUnderItsLock` | 组合 |

**Done when:** 五个测试在 `-race` 下通过，探针只认它们点名的表；`grep -n "t.Fatal\|FailNow" server/internal/bootstrap/deactivation_world_test.go` 只在测试的 goroutine 上（`sent` 返回的回答里、`newDeactivationWorld` 里，清扫 33）；`make lint-go`、`make test` 通过。

### Task 7: 交错 7（停用与接受邀请）、16（停用与恢复成员）

**Files:**
- Create: `server/internal/bootstrap/interleaving_accepting_test.go`
- Modify: `server/internal/bootstrap/interleaving_answers_test.go`、`server/internal/bootstrap/interleaving_growth_test.go`、`server/internal/bootstrap/interleaving_invite_test.go`、`server/internal/bootstrap/reactivation_races_test.go`

**Interfaces:**
- 没有产品代码。测试的共用：`newAnswerRace(t, role)`（邀请的角色）、`answerRace.deactivateBob(ctx, sessions, memberships)`（`interleaving_answers_test.go`）；`changeRole(ctx, pool, by, id, role, members, cascade)`（`interleaving_accepting_test.go`：`updateWorkspaceMember` 照 `bootstrap` 的接法，P4b 的降为访客和交错 7 (b) 共用）；`s1Race`、`newS1Race(t, former)`、`standing`（`interleaving_accepting_test.go`）；`endersOf(t, pool, id)`（`reactivation_races_test.go`：一个账户每一个已结束的成员关系，工作区按 slug、项目按名字，各写明最后写它的人，按字节排序，清扫 39、40；交错 13–16 和两位管理员共用）。

**Tests:**
- `TestAcceptingAndDeactivating`（交错 7 的 (a)、(d)、(e)，四个子测试）：接受先持他的账户 `FOR SHARE` 和 acme 的 N 停在 gate，停用等账户行；提交之后停用列举到 acme，结束他的成员关系（由他；alice 仍是管理员）。停用先持账户行停在 gate，接受等它，之后读到停用：401，没有成员关系，邀请由停用删除。以前的成员行：接受恢复那一行，(a) 结束它，(d) 照 alice 的移出留下。探针认 `users`。
- `TestAnAdmittedAdminsWriteAndHisDeactivation`（7 的 (b)、(c)）：接受先提交；停用停在账户行之后自己的 gate；bob 另一个事务：(b) 把 alice 改为成员（持 acme 的 N 停在 gate），(c) 加入 Web（持 acme 的 S 和他的成员关系停在 gate）；停用继续、等 acme 的行，两方都不写 `workspaces`。之后 (b) `workspace.sole_admin`，每张表照接受和改角色留下的；(c) 停用列举到 Web，和 acme 的一起结束（由他），acme 仍有 alice。
- `TestReactivatingAndDeactivating`（交错 16，`reactivation_races_test.go`）：恢复先：acme 被持有，恢复持他的账户 `FOR SHARE` 等它，停用等账户行；放开之后恢复完成，停用列举到 acme、再结束它，Web、Ops 仍结束。停用先：停用持账户行停在 gate，恢复等它；停用提交之后恢复照样进行（3.11 的例外），输出提示 `nerve users activate`。`endersOf`：恢复先是 `Ops alice, Web alice, acme bob`，停用先是 `Ops alice, Web alice`。
- `interleaving_invite_test.go`：P3 的交错测试照新的 `newAnswerRace(t, shared.RoleMember)`。

- [ ] **Step 1: 共用的竞争**

`server/internal/bootstrap/interleaving_answers_test.go`（修改，9 处）：

````old server/internal/bootstrap/interleaving_answers_test.go
// invitation is the link of bob's invitation, as a member.
````
````new server/internal/bootstrap/interleaving_answers_test.go
// invitation is the link of bob's invitation, with the role
// newAnswerRace is given.
````

````old server/internal/bootstrap/interleaving_answers_test.go
func newAnswerRace(t *testing.T) answerRace {
````
````new server/internal/bootstrap/interleaving_answers_test.go
func newAnswerRace(t *testing.T, role shared.Role) answerRace {
````

````old server/internal/bootstrap/interleaving_answers_test.go
		{ID: id, WorkspaceID: r.acme, Email: "bob@example.com", Role: shared.RoleMember, CreatedBy: r.alice, Now: now},
````
````new server/internal/bootstrap/interleaving_answers_test.go
		{ID: id, WorkspaceID: r.acme, Email: "bob@example.com", Role: role, CreatedBy: r.alice, Now: now},
````

````old server/internal/bootstrap/interleaving_answers_test.go
		name := map[bool]string{true: "the acceptance first", false: "the deletion first"}[acceptFirst]
		t.Run(name, func(t *testing.T) {
			r := newAnswerRace(t)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
````
````new server/internal/bootstrap/interleaving_answers_test.go
		name := map[bool]string{true: "the acceptance first", false: "the deletion first"}[acceptFirst]
		t.Run(name, func(t *testing.T) {
			r := newAnswerRace(t, shared.RoleMember)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
````

````old server/internal/bootstrap/interleaving_answers_test.go
		name := map[bool]string{true: "the acceptance first", false: "the change first"}[acceptFirst]
		t.Run(name, func(t *testing.T) {
			r := newAnswerRace(t)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
````
````new server/internal/bootstrap/interleaving_answers_test.go
		name := map[bool]string{true: "the acceptance first", false: "the change first"}[acceptFirst]
		t.Run(name, func(t *testing.T) {
			r := newAnswerRace(t, shared.RoleMember)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
````

````old server/internal/bootstrap/interleaving_answers_test.go
// (deactivating), over memberships.
func (r answerRace) deactivateBob(ctx context.Context, memberships workspaceapp.AllMembershipsEnder) error {
	_, err := deactivating(r.pool, identitypg.New(r.pool), memberships).ExecuteByEmail(ctx, "bob@example.com")
````
````new server/internal/bootstrap/interleaving_answers_test.go
// (deactivating), over sessions and memberships.
func (r answerRace) deactivateBob(ctx context.Context, sessions identityapp.SessionRevoker, memberships workspaceapp.AllMembershipsEnder) error {
	_, err := deactivating(r.pool, sessions, memberships).ExecuteByEmail(ctx, "bob@example.com")
````

````old server/internal/bootstrap/interleaving_answers_test.go
		t.Run(name, func(t *testing.T) {
			r := newAnswerRace(t)
			r.join(t, r.bob, shared.RoleMember)
````
````new server/internal/bootstrap/interleaving_answers_test.go
		t.Run(name, func(t *testing.T) {
			r := newAnswerRace(t, shared.RoleMember)
			r.join(t, r.bob, shared.RoleMember)
````

````old server/internal/bootstrap/interleaving_answers_test.go
				deactivated = run(func() error { return r.deactivateBob(ctx, store) })
````
````new server/internal/bootstrap/interleaving_answers_test.go
				deactivated = run(func() error { return r.deactivateBob(ctx, identitypg.New(r.pool), store) })
````

````old server/internal/bootstrap/interleaving_answers_test.go
				deactivated = run(func() error { return r.deactivateBob(ctx, endedHoldingAll{store, g}) })
````
````new server/internal/bootstrap/interleaving_answers_test.go
				deactivated = run(func() error { return r.deactivateBob(ctx, identitypg.New(r.pool), endedHoldingAll{store, g}) })
````

`server/internal/bootstrap/interleaving_invite_test.go`（修改，1 处）：

````old server/internal/bootstrap/interleaving_invite_test.go
	r := inviteRace{answerRace: newAnswerRace(t), carol: uuid.NewV7(), aliceSession: uuid.NewV7(), carolSession: uuid.NewV7()}
````
````new server/internal/bootstrap/interleaving_invite_test.go
	r := inviteRace{answerRace: newAnswerRace(t, shared.RoleMember), carol: uuid.NewV7(), aliceSession: uuid.NewV7(), carolSession: uuid.NewV7()}
````

`server/internal/bootstrap/interleaving_growth_test.go`（修改，4 处）：

````old server/internal/bootstrap/interleaving_growth_test.go
	"github.com/open-nerve/NerveProject/server/internal/modules/access"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
````
````new server/internal/bootstrap/interleaving_growth_test.go
	"github.com/open-nerve/NerveProject/server/internal/modules/access"
````

````old server/internal/bootstrap/interleaving_growth_test.go
	workspacedomain "github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
````
````new server/internal/bootstrap/interleaving_growth_test.go
	workspacedomain "github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
````

````old server/internal/bootstrap/interleaving_growth_test.go
// and cascade, on the system's clock: its time is read when it reads it.
````
````new server/internal/bootstrap/interleaving_growth_test.go
// and cascade (changeRole).
````

````old server/internal/bootstrap/interleaving_growth_test.go
	_, err := workspaceapp.NewUpdateWorkspaceMember(members, cascade, workspaceProfiles{profiles: identity.Provide(r.pool).PublicProfiles},
		r.authorizer(), postgres.NewTxManager(r.pool, 2*time.Second), clock.System{}).
		Execute(shared.WithActor(ctx, shared.Actor{UserID: r.alice}), r.bobIn, shared.RoleGuest)
	return err
````
````new server/internal/bootstrap/interleaving_growth_test.go
	return changeRole(ctx, r.pool, r.alice, r.bobIn, shared.RoleGuest, members, cascade)
````

- [ ] **Step 2: 交错 7**

`server/internal/bootstrap/interleaving_accepting_test.go`（新文件，248 行）：

````file server/internal/bootstrap/interleaving_accepting_test.go
package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	identitypg "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	projectpg "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres"
	projectapp "github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	projectdomain "github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	workspacedomain "github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Interleaving 7 of M3 design 9.3 (Codex S1): bob's acceptance of his
// invitation to acme, a workspace he is not in, against his deactivation,
// `nerve users deactivate` as bootstrap wires it (deactivating), each in
// its own transaction on a real database. Both lock bob's account row
// first (3.6 convention 6), the acceptance FOR SHARE and the deactivation
// FOR NO KEY UPDATE, so the deactivation's enumeration of his workspaces
// comes after any acceptance that committed first. Every wait has a
// deadline.

// s1Race is answerRace with bob invited to acme as its admin, and Web,
// acme's public project, of which alice is the admin; when former, bob has
// a membership of acme, as a member, which alice's removal ended before she
// invited him again (3.8).
type s1Race struct {
	answerRace
	web, aliceIn uuid.UUID
}

func newS1Race(t *testing.T, former bool) s1Race {
	t.Helper()
	r := s1Race{answerRace: newAnswerRace(t, shared.RoleAdmin), web: uuid.NewV7()}
	ctx, before := context.Background(), time.Now().Add(-time.Hour)
	r.aliceIn = queryIDs(t, r.pool, "SELECT id FROM workspace_members WHERE workspace_id = $1 AND member_id = $2", r.acme, r.alice)[0]
	projects := projectpg.New(r.pool)
	if err := projects.CreateProject(ctx, projectapp.ProjectRow{ID: r.web, WorkspaceID: r.acme, Name: "Web", Identifier: "WEB",
		Network: projectdomain.NetworkPublic, Timezone: "UTC", CreatedBy: r.alice, Now: before}); err != nil {
		t.Fatal(err)
	}
	if err := projects.CreateMember(ctx, projectapp.MemberRow{ID: uuid.NewV7(), WorkspaceID: r.acme, ProjectID: r.web, MemberID: r.alice,
		Role: shared.RoleAdmin, CreatedBy: r.alice, Now: before}); err != nil {
		t.Fatal(err)
	}
	if former {
		workspaces := workspacepg.New(r.pool)
		if err := workspaces.CreateMember(ctx, workspaceapp.MemberRow{ID: uuid.NewV7(), WorkspaceID: r.acme, MemberID: r.bob,
			Role: shared.RoleMember, CreatedBy: r.alice, Now: before}); err != nil {
			t.Fatal(err)
		}
		if err := workspaces.EndMember(ctx, r.acme, r.bob, r.alice, before); err != nil {
			t.Fatal(err)
		}
	}
	return r
}

// standing is bob's account, active or deactivated; each of his
// memberships of acme, in the order they were made, and of Web, with its
// role, and its ender when it has ended; the invitation, accepted or
// unanswered, and who deleted it when it is deleted; and alice's role in
// acme.
func (r s1Race) standing(t *testing.T) string {
	t.Helper()
	var s string
	if err := r.pool.QueryRow(soon(t), `SELECT concat_ws('; ', CASE WHEN u.is_active THEN 'bob active' ELSE 'bob deactivated' END,
		(SELECT string_agg('acme ' || m.role || CASE WHEN m.is_active THEN ' active' ELSE ' ended by ' || split_part(e.email, '@', 1) END,
			', ' ORDER BY m.created_at) FROM workspace_members m JOIN users e ON e.id = m.updated_by_id
			WHERE m.workspace_id = $2 AND m.member_id = u.id),
		(SELECT 'Web ' || m.role || CASE WHEN m.is_active THEN ' active' ELSE ' ended by ' || split_part(e.email, '@', 1) END
			FROM project_members m JOIN users e ON e.id = m.updated_by_id WHERE m.project_id = $3 AND m.member_id = u.id),
		(SELECT 'invitation ' || CASE WHEN i.accepted THEN 'accepted' ELSE 'unanswered' END ||
			CASE WHEN i.deleted_at IS NULL THEN '' ELSE ' deleted by ' || split_part(e.email, '@', 1) END
			FROM workspace_member_invites i LEFT JOIN users e ON e.id = i.updated_by_id WHERE i.id = $4),
		(SELECT 'alice ' || m.role FROM workspace_members m WHERE m.id = $5))
		FROM users u WHERE u.id = $1`, r.bob, r.acme, r.web, r.invitation.id, r.aliceIn).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

// changeRole is by's change of the role of the workspace membership id to
// role, updateWorkspaceMember over members and cascade, with identity's
// profiles and the Authorizer as bootstrap wires them, on pool and the
// system's clock: its time is read when it reads it. It takes no test, so
// it runs on any goroutine.
func changeRole(ctx context.Context, pool *pgxpool.Pool, by, id uuid.UUID, role shared.Role, members workspaceapp.MemberUpdater,
	cascade workspaceapp.ProjectCascade) error {
	_, err := workspaceapp.NewUpdateWorkspaceMember(members, cascade, workspaceProfiles{profiles: identity.Provide(pool).PublicProfiles},
		authorizerOn(pool), postgres.NewTxManager(pool, 2*time.Second), clock.System{}).
		Execute(shared.WithActor(ctx, shared.Actor{UserID: by}), id, role)
	return err
}

// Interleaving 7 (a), (d) and (e):
//   - (a) the acceptance first holds bob's account row FOR SHARE, and acme's
//     FOR NO KEY UPDATE, at its gate; the deactivation waits for his
//     account's row. Once the acceptance has committed, the deactivation
//     finds acme among his workspaces and ends his membership of it, as
//     his: he is no admin acme lacks, alice being one.
//   - (d) the deactivation first holds his account row at its gate, before
//     its memberships' step; the acceptance waits for it, then reads his
//     account deactivated under its lock: 401, no membership. The
//     deactivation deleted the invitation, as his.
//   - (e) bob has a former membership of acme: the acceptance restores that
//     row rather than inserting one, and serializes on his account row as
//     an insertion does: (a) ends the restored row, his only one; (d) leaves
//     it as alice's removal did.
//
// The probe sees the one wait on users that either order has.
func TestAcceptingAndDeactivating(t *testing.T) {
	for _, tt := range []struct {
		former, acceptFirst bool
		accept              error
		want                string
	}{
		{false, true, nil, "bob deactivated; acme 20 ended by bob; invitation accepted deleted by bob; alice 20"},
		{false, false, shared.Unauthenticated(), "bob deactivated; invitation unanswered deleted by bob; alice 20"},
		{true, true, nil, "bob deactivated; acme 20 ended by bob; invitation accepted deleted by bob; alice 20"},
		{true, false, shared.Unauthenticated(), "bob deactivated; acme 15 ended by alice; invitation unanswered deleted by bob; alice 20"},
	} {
		t.Run(fmt.Sprintf("former %v, the acceptance first %v", tt.former, tt.acceptFirst), func(t *testing.T) {
			r := newS1Race(t, tt.former)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			g := newGate()
			store, users := workspacepg.New(r.pool), identitypg.New(r.pool)
			var accepted, deactivated <-chan error
			if tt.acceptFirst {
				accepted = run(func() error { return r.accept(ctx, gatedAccepter{store, g}) })
				held(t, ctx, g, accepted, "the acceptance")
				deactivated = run(func() error { return r.deactivateBob(ctx, users, store) })
			} else {
				deactivated = run(func() error { return r.deactivateBob(ctx, gatedSessions{users, g}, store) })
				held(t, ctx, g, deactivated, "the deactivation")
				accepted = run(func() error { return r.accept(ctx, store) })
			}
			pgtest.WaitForLockWaitOn(t, r.pool, "users", 5*time.Second)
			close(g.open)

			if err := result(t, ctx, accepted, "the acceptance"); !errors.Is(err, tt.accept) {
				t.Errorf("the acceptance = %v, want %v", err, tt.accept)
			}
			if err := result(t, ctx, deactivated, "the deactivation"); err != nil {
				t.Errorf("the deactivation = %v, want it done", err)
			}
			if got := r.standing(t); got != tt.want {
				t.Errorf("after both: %s; want %s", got, tt.want)
			}
		})
	}
}

// Interleaving 7 (b) and (c), S1's steps 5 and 6: the acceptance first
// holds bob's account row at its gate; the deactivation waits for it, then,
// once the acceptance has committed, stops at its own gate holding his
// account row, before its memberships' step. bob, acme's admin now, begins
// another write of acme, which holds acme's row at its gate: (b) his change
// of alice's role to member, holding acme FOR NO KEY UPDATE before its
// write; (c) his joining Web, holding acme FOR SHARE and his membership of
// it (3.6 convention 2, option E). The deactivation goes on and waits for
// acme's row: nothing else waits then, and neither side writes a row of
// workspaces, so only that wait satisfies the probe. Once bob's write
// commits:
//   - (b) bob is acme's only active admin, alice its member:
//     workspace.sole_admin, and no row of any table changes from what the
//     acceptance and the change left;
//   - (c) the deactivation finds his membership of Web, made after its lock
//     of his account, and ends it with his membership of acme, as his;
//     acme keeps alice, its admin.
func TestAnAdmittedAdminsWriteAndHisDeactivation(t *testing.T) {
	for _, tt := range []struct {
		name  string
		write func(r s1Race, t *testing.T, ctx context.Context, g *gate) <-chan error
		err   error
		want  string
	}{
		{"(b) he makes alice a member", func(r s1Race, t *testing.T, ctx context.Context, g *gate) <-chan error {
			cascade := project.NewCascade(project.CascadeDeps{Pool: r.pool})
			return run(func() error {
				return changeRole(ctx, r.pool, r.bob, r.aliceIn, shared.RoleMember, gatedMembers{workspacepg.New(r.pool), g}, cascade)
			})
		}, workspacedomain.ErrSoleAdmin, "bob active; acme 20 active; invitation accepted deleted by bob; alice 15"},
		{"(c) he joins Web", func(r s1Race, t *testing.T, ctx context.Context, g *gate) <-chan error {
			route := newProjectRoute(t, r.pool, authorizerOn(r.pool), gatedShares{workspace.Provide(r.pool).WorkspaceMembers, g})
			return run(func() error {
				if _, rec := route.send(http.MethodPost, "/api/v0/projects/"+r.web.String()+"/join", r.bob, ""); rec.Code != http.StatusOK {
					return fmt.Errorf("bob's joining Web = %d %s", rec.Code, rec.Body)
				}
				return nil
			})
		}, nil, "bob deactivated; acme 20 ended by bob; Web 20 ended by bob; invitation accepted deleted by bob; alice 20"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := newS1Race(t, false)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			admitted, stopped, writing := newGate(), newGate(), newGate()
			store, users := workspacepg.New(r.pool), identitypg.New(r.pool)
			accepted := run(func() error { return r.accept(ctx, gatedAccepter{store, admitted}) })
			held(t, ctx, admitted, accepted, "the acceptance")
			deactivated := run(func() error { return r.deactivateBob(ctx, gatedSessions{users, stopped}, store) })
			pgtest.WaitForLockWaitOn(t, r.pool, "users", 5*time.Second)
			close(admitted.open)
			if err := result(t, ctx, accepted, "the acceptance"); err != nil {
				t.Fatalf("the acceptance = %v, want it done", err)
			}
			held(t, ctx, stopped, deactivated, "the deactivation")
			written := tt.write(r, t, ctx, writing)
			held(t, ctx, writing, written, "bob's write")
			close(stopped.open)
			pgtest.WaitForLockWaitOn(t, r.pool, "workspaces", 5*time.Second)
			close(writing.open)

			if err := result(t, ctx, written, "bob's write"); err != nil {
				t.Fatalf("bob's write = %v, want it done", err)
			}
			before := tableRows(t, r.pool, riversOwn)
			if err := result(t, ctx, deactivated, "the deactivation"); !sameOutcome(err, tt.err) {
				t.Errorf("the deactivation = %v, want %v", err, tt.err)
			}
			if after := tableRows(t, r.pool, riversOwn); tt.err != nil && !maps.Equal(after, before) {
				t.Errorf("the tables after the refused deactivation changed:\n%v\nwant them as bob's write left them:\n%v", after, before)
			}
			if got := r.standing(t); got != tt.want {
				t.Errorf("after all: %s; want %s", got, tt.want)
			}
		})
	}
}
````

- [ ] **Step 3: 交错 16**

`server/internal/bootstrap/reactivation_races_test.go`（修改，2 处）：

````old server/internal/bootstrap/reactivation_races_test.go
	"github.com/jackc/pgx/v5/pgxpool"

````
````new server/internal/bootstrap/reactivation_races_test.go
	"github.com/jackc/pgx/v5/pgxpool"

	identitypg "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
````

````old server/internal/bootstrap/reactivation_races_test.go
		t.Errorf("reactivate-member on a pool of one connection = %q, %v; want %q", run.out, run.err, bobReactivated)
	}
}

````
````new server/internal/bootstrap/reactivation_races_test.go
		t.Errorf("reactivate-member on a pool of one connection = %q, %v; want %q", run.out, run.err, bobReactivated)
	}
}

// endersOf is each ended membership of the account id, of a workspace by
// its slug, of a project by its name, with the name of who last wrote it,
// "nobody" when no account did, in byte order; "none" when none has ended.
func endersOf(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) string {
	t.Helper()
	var s string
	if err := pool.QueryRow(soon(t), `SELECT coalesce(string_agg(e.name || ' ' || coalesce(split_part(u.email, '@', 1), 'nobody'), ', '
			ORDER BY e.name COLLATE "C"), 'none')
		FROM (SELECT w.slug AS name, m.updated_by_id FROM workspace_members m JOIN workspaces w ON w.id = m.workspace_id
		      WHERE m.member_id = $1 AND NOT m.is_active
		      UNION ALL
		      SELECT p.name, m.updated_by_id FROM project_members m JOIN projects p ON p.id = m.project_id
		      WHERE m.member_id = $1 AND NOT m.is_active) e
		LEFT JOIN users u ON u.id = e.updated_by_id`, id).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

// Interleaving 16 (M3 design 9.3, 3.6 convention 6, 3.11): bob's
// reactivation in acme and his deactivation, each through its command's
// composition, serialize on his account's row, in both orders.
//   - The reactivation first: another transaction holds acme's row FOR NO
//     KEY UPDATE; the reactivation waits for it, holding his account FOR
//     SHARE; the deactivation waits for his account's row. The holder lets
//     go: the reactivation makes his membership of acme active again, and
//     the deactivation, which finds his workspaces once it holds his
//     account, finds acme among them and ends his membership again, as
//     his, his memberships of Web and Ops ended still, as alice's.
//   - The deactivation first holds his account row at its gate, before its
//     memberships' step; the reactivation waits for it. Once the
//     deactivation has committed, the reactivation reactivates him all the
//     same, 3.11's exception for a deactivated account, and its line says
//     what is next: nerve users activate; Web's and Ops's stay alice's.
//
// Each probe sees the one wait on its table that its order has.
func TestReactivatingAndDeactivating(t *testing.T) {
	t.Run("the reactivation first", func(t *testing.T) {
		url := pgtest.NewDatabase(t)
		pool := endedMembers(t, url)
		ids := idsOf(t, pool)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		holdsAcme := holding(t, pool, "SELECT 1 FROM workspaces WHERE id = $1 FOR NO KEY UPDATE", ids.acme)
		reactivated := reactivatingBob(ctx, t, url, 4)
		pgtest.WaitForLockWaitOn(t, pool, "workspaces", 5*time.Second)
		deactivated := deactivatingUser(ctx, testConfig(t, url, false), "bob@corp.com")
		pgtest.WaitForLockWaitOn(t, pool, "users", 5*time.Second)
		if err := holdsAcme.Rollback(context.Background()); err != nil {
			t.Fatal(err)
		}

		if run := receiveWithin(t, reactivated, 10*time.Second, "reactivate-member's end"); run.out != bobReactivated || run.err != nil {
			t.Errorf("reactivate-member = %q, %v; want %q", run.out, run.err, bobReactivated)
		}
		want := "deactivated bob@corp.com: revoked 0 sessions and ended its memberships; to bring it back, run nerve users activate, then " +
			"nerve workspaces reactivate-member in each workspace\n"
		if run := receiveWithin(t, deactivated, 10*time.Second, "the end of nerve users deactivate"); run.out != want || run.err != nil {
			t.Errorf("nerve users deactivate = %q, %v; want %q", run.out, run.err, want)
		}
		if got, want := memberStates(t, pool), strings.Join([]string{"alice@corp.com acme 20 true", "bob@corp.com Ops 20 false",
			"bob@corp.com Web 20 false", "bob@corp.com acme 20 false", "carol@corp.com Web 5 false", "carol@corp.com acme 5 false"}, "\n"); got != want {
			t.Errorf("the memberships after both:\n%s\nwant\n%s", got, want)
		}
		if got, want := endersOf(t, pool, ids.bob), "Ops alice, Web alice, acme bob"; got != want {
			t.Errorf("bob's ended memberships by their last writers: %s; want %s, his deactivation having ended acme's", got, want)
		}
	})
	t.Run("the deactivation first", func(t *testing.T) {
		url := pgtest.NewDatabase(t)
		pool := endedMembers(t, url)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		g := newGate()
		deactivated := run(func() error {
			_, err := deactivating(pool, gatedSessions{identitypg.New(pool), g}, workspacepg.New(pool)).ExecuteByEmail(ctx, "bob@corp.com")
			return err
		})
		held(t, ctx, g, deactivated, "the deactivation")
		reactivated := reactivatingBob(ctx, t, url, 4)
		pgtest.WaitForLockWaitOn(t, pool, "users", 5*time.Second)
		close(g.open)

		if err := result(t, ctx, deactivated, "the deactivation"); err != nil {
			t.Errorf("the deactivation = %v, want it done", err)
		}
		want := strings.TrimSuffix(bobReactivated, "\n") + "; the account is deactivated: run nerve users activate --email bob@corp.com next\n"
		if run := receiveWithin(t, reactivated, 10*time.Second, "reactivate-member's end"); run.out != want || run.err != nil {
			t.Errorf("reactivate-member = %q, %v; want %q", run.out, run.err, want)
		}
		if got, want := memberStates(t, pool), strings.Join([]string{"alice@corp.com acme 20 true", "bob@corp.com Ops 20 false",
			"bob@corp.com Web 20 false", "bob@corp.com acme 20 true", "carol@corp.com Web 5 false", "carol@corp.com acme 5 false"}, "\n"); got != want {
			t.Errorf("the memberships after both:\n%s\nwant\n%s", got, want)
		}
		if got, want := endersOf(t, pool, idsOf(t, pool).bob), "Ops alice, Web alice"; got != want {
			t.Errorf("bob's ended memberships by their last writers: %s; want %s", got, want)
		}
	})
}

````

- [ ] **Step 4: 测试和 lint**

Run: `go -C server test -count=5 -race -run 'TestAcceptingAndDeactivating$|TestAnAdmittedAdminsWriteAndHisDeactivation$|TestReactivatingAndDeactivating$' ./internal/bootstrap/`
Expected: `ok`；输出里没有 `40P01`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 5: 提交**

```bash
git add server/internal/bootstrap/interleaving_accepting_test.go server/internal/bootstrap/interleaving_answers_test.go server/internal/bootstrap/interleaving_growth_test.go server/internal/bootstrap/interleaving_invite_test.go server/internal/bootstrap/reactivation_races_test.go
```
```bash
git commit -m "test(M3/P6): interleavings 7 and 16, a deactivation against an acceptance and against reactivate-member

Codex S1's four ways and the former membership's: an acceptance that
holds the account first is ended by the deactivation, which finds the
workspace it joined; one that waits reads the account deactivated;
the admitted admin's change of role leaves him the only admin, and the
deactivation is refused; his joining a project is found and ended.
reactivate-member first is undone by the deactivation; after it, it
reactivates all the same and says nerve users activate is next.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**：本 Task 没有产品代码；它的测试在别的 Task 的变异表里（"（Task 7 起）"）。

**Done when:** 三个测试在 `-count=5 -race` 下通过，没有 40P01；每个结果写明每一行的结束者（清扫 40）；`make lint-go`、`make test` 通过。

### Task 8: 交错 13、14、15；两位管理员同时停用

**Files:**
- Create: `server/internal/bootstrap/interleaving_shrinking_test.go`

**Interfaces:**
- 没有产品代码。照 P5b 的 `TestALeavingAndAnEndingOfAWorkspaceMembershipSerialize` 写（P5b review 第 6 节）：第二方等工作区的行，两方都不写 `workspaces`，探针只由那一把锁的等待满足；`growthRace.deactivateBob`、`refusedAsEnded`；结果由 Task 7 的 `endersOf` 读；`soleAdminCheckedHolding`（停在规则 2 的检查之后、写之前）；`aliceDeactivated`、`ginaDeactivated`。

**Tests:**（`bootstrap/interleaving_shrinking_test.go`）
- `TestADeactivationAndTheProjectSidesGrowthSerialize`（交错 13、14：添加 / 加入 × 新的 / 以前的 Web 行 × 两种顺序）：增长先持 acme 和他的成员关系 `FOR SHARE`，停用等 acme 的行；之后停用列举到 Web 并结束（由他，时刻在 gate 打开之后；`endersOf` 是 `Web bob, acme bob`）。停用先（`endedHoldingAll`），增长等 acme 的行，之后读到他已结束：添加 422 `members[0].member_id` `not_allowed`，加入 404；以前的行照 alice 的移出留下（`Web alice, acme bob`，新的是 `acme bob`）。第二方等待时没有事务持有 Web。
- `TestADeactivationAndACreationHeLeadsSerialize`（交错 15）：建项目先：停用之后列举到 Ops 并结束他在那里的（alice 也是管理员，规则 2 不拒绝）；停用先：建项目 422 `project_lead_id`，没有 Ops（`endersOf` 分别是 `Ops bob, acme bob`、`acme bob`）。
- `TestTwoAdminsDeactivatedAtOnceLeaveAnAdmin`（`memberWorld`，四个子测试）：acme 的两位管理员同时停用，第一位停在她的检查之后（清扫 25、31 的 gate：检查和写之间、她的事务之外的实现在这里失败）或她的成员关系结束之后；第二位等 acme；之后第二位 `workspace.sole_admin`、账户仍有效；acme 仍有管理员；第一位每一个已结束的成员关系都由她本人结束（`endersOf`：`Ops alice, Web alice, acme alice` 或 `Web gina, acme gina`）。

- [ ] **Step 1: 交错 13、14、15 和两位管理员**

`server/internal/bootstrap/interleaving_shrinking_test.go`（新文件，299 行）：

````file server/internal/bootstrap/interleaving_shrinking_test.go
package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"uuid"

	identitypg "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	workspacedomain "github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// The interleavings 13, 14 and 15 of M3 design 9.3, and rule 2 under two
// deactivations at once: a deactivation, `nerve users deactivate` as
// bootstrap wires it (deactivating), against a growth of the account's set
// of memberships on the project side, through the project module as
// bootstrap wires it, or against another deactivation, on a real database,
// in both orders, as TestALeavingAndAnEndingOfAWorkspaceMembershipSerialize
// is: the second side waits for the workspace's row, and neither side
// writes a row of workspaces, so only that lock's wait satisfies the
// probe. Every wait has a deadline.

// deactivateBob is bob's deactivation on r's database (deactivating), over
// memberships.
func (r growthRace) deactivateBob(ctx context.Context, memberships workspaceapp.AllMembershipsEnder) error {
	_, err := deactivating(r.pool, identitypg.New(r.pool), memberships).ExecuteByEmail(ctx, "bob@example.com")
	return err
}

// refusedAsEnded reports whether rec is the project side's refusal of bob,
// his membership of acme ended: his joining Web, 404 project.not_found;
// alice's adding him to it, 422 at members[0].member_id; her creation of a
// project he leads, 422 at project_lead_id; each not_allowed, alone.
func refusedAsEnded(rec *httptest.ResponseRecorder, field string) bool {
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
	if field == "" {
		return rec.Code == http.StatusNotFound && problem.Code == "project.not_found"
	}
	return rec.Code == http.StatusUnprocessableEntity && problem.Code == "validation_failed" && len(problem.Errors) == 1 &&
		problem.Errors[0].Field == field && problem.Errors[0].Code == "not_allowed"
}

// Interleavings 13 and 14: bob's deactivation and the project side's
// growth, alice's adding him to Web or his joining it, serialize on acme's
// row, in both orders, for a new membership of Web and for his ended one,
// which the growth restores (3.6 conventions 2 and 6). The growth first: it
// holds acme and his membership of acme FOR SHARE and waits at its gate,
// before it locks Web; the deactivation waits for acme's row, holding his
// account. Once the growth has committed, the deactivation, which finds
// his projects when its step over them runs, after its end of his
// membership of acme, finds Web and ends his membership of it, as his, at
// a moment read once it held acme: after the gate opened (3.3); both
// endings are his. The
// deactivation first: it holds acme FOR NO KEY UPDATE and his membership's
// row after its end; the growth waits for acme's row, then reads his
// membership ended: alice's adding is 422 at members[0].member_id, his
// joining 404; his membership of acme is ended as his, and an ended
// membership of Web stays as alice's removal left it. In either order no
// transaction holds Web while the second side waits.
func TestADeactivationAndTheProjectSidesGrowthSerialize(t *testing.T) {
	contract := apitest.Load(t)
	for _, add := range []bool{true, false} {
		for _, ended := range []bool{false, true} {
			for _, growthFirst := range []bool{true, false} {
				t.Run(fmt.Sprintf("add %v, ended %v, growth first %v", add, ended, growthFirst), func(t *testing.T) {
					r := newGrowthRace(t, ended)
					before, enders := "15, Web none", "acme bob"
					if ended {
						before, enders = "15, Web 15 ended", "Web alice, acme bob"
					}
					if got := r.standing(t); got != before {
						t.Fatalf("before: %s, want %s", got, before)
					}
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancel()
					g := newGate()
					var req *http.Request
					var rec *httptest.ResponseRecorder
					var grew, deactivated <-chan error
					if growthFirst {
						grow := r.growth(t, add, g)
						grew = run(func() error { req, rec = grow(); return nil })
						held(t, ctx, g, grew, "the growth")
						deactivated = run(func() error { return r.deactivateBob(ctx, workspacepg.New(r.pool)) })
					} else {
						deactivated = run(func() error { return r.deactivateBob(ctx, endedHoldingAll{workspacepg.New(r.pool), g}) })
						held(t, ctx, g, deactivated, "the deactivation")
						grow := r.growth(t, add, nil)
						grew = run(func() error { req, rec = grow(); return nil })
					}
					pgtest.WaitForLockWaitOn(t, r.pool, "workspaces", 5*time.Second)
					if !r.webFree(t) {
						t.Error("Web is held while the second side waits for acme's row; want it locked after that row")
					}
					opened := time.Now()
					close(g.open)

					deactivation := result(t, ctx, deactivated, "the deactivation")
					if err := result(t, ctx, grew, "the growth"); err != nil {
						t.Fatal(err)
					}
					contract.CheckResponse(t, req, rec.Result())
					want, grown := "15 ended, Web 15 ended", rec.Code == map[bool]int{false: http.StatusOK, true: http.StatusCreated}[add]
					if growthFirst {
						enders = "Web bob, acme bob"
					} else {
						want = map[bool]string{false: "15 ended, Web none", true: "15 ended, Web 15 ended"}[ended]
						grown = refusedAsEnded(rec, map[bool]string{true: "members[0].member_id"}[add])
					}
					got, by := r.standing(t), endersOf(t, r.pool, r.bob)
					if deactivation != nil || !grown || got != want || by != enders {
						t.Errorf("the deactivation = %v, the growth = %d %s, bob %s, his ended memberships by their last writers %s; want the "+
							"deactivation done, the growth %s, bob %s, %s", deactivation, rec.Code, rec.Body, got, by, map[bool]string{true: "done",
							false: "refused"}[growthFirst], want, enders)
					}
					if made, written := r.membershipTimes(t); growthFirst && (written.Before(made) || written.Before(opened)) {
						t.Errorf("bob's membership of Web made at %v, ended at %v, the gate opened at %v; want it ended by the deactivation, at "+
							"a moment read once it held acme", made, written, opened)
					}
				})
			}
		}
	}
}

// Interleaving 15: bob's deactivation and alice's creation of Ops with him
// as its lead serialize on acme's row, in both orders. The creation first:
// it holds acme and his membership of acme FOR SHARE at its gate; the
// deactivation waits for acme's row. Once the creation has committed, the
// deactivation finds Ops among his projects and ends his membership of it,
// as his, with his membership of acme: alice, its creator, is its admin
// too, so he is not its only one (3.7 rule 2). The deactivation first: the
// creation waits for acme's row, then reads his membership ended, as his:
// 422 at project_lead_id, and no Ops.
func TestADeactivationAndACreationHeLeadsSerialize(t *testing.T) {
	contract := apitest.Load(t)
	for _, creationFirst := range []bool{true, false} {
		t.Run(fmt.Sprintf("creation first %v", creationFirst), func(t *testing.T) {
			r := newGrowthRace(t, false)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			g := newGate()
			var req *http.Request
			var rec *httptest.ResponseRecorder
			create := func(gate *gate) func() error {
				route := newProjectRoute(t, r.pool, r.authorizer(), gatedShares{workspace.Provide(r.pool).WorkspaceMembers, gate})
				return func() error {
					req, rec = route.send(http.MethodPost, "/api/v0/workspaces/acme/projects", r.alice,
						`{"name":"Ops","identifier":"OPS","project_lead_id":"`+r.bob.String()+`"}`)
					return nil
				}
			}
			var created, deactivated <-chan error
			if creationFirst {
				created = run(create(g))
				held(t, ctx, g, created, "the creation")
				deactivated = run(func() error { return r.deactivateBob(ctx, workspacepg.New(r.pool)) })
			} else {
				deactivated = run(func() error { return r.deactivateBob(ctx, endedHoldingAll{workspacepg.New(r.pool), g}) })
				held(t, ctx, g, deactivated, "the deactivation")
				created = run(create(nil))
			}
			pgtest.WaitForLockWaitOn(t, r.pool, "workspaces", 5*time.Second)
			close(g.open)

			deactivation := result(t, ctx, deactivated, "the deactivation")
			if err := result(t, ctx, created, "the creation"); err != nil {
				t.Fatal(err)
			}
			contract.CheckResponse(t, req, rec.Result())
			done, by := rec.Code == http.StatusCreated, endersOf(t, r.pool, r.bob)
			if want := map[bool]string{true: "Ops bob, acme bob", false: "acme bob"}[creationFirst]; creationFirst != done || deactivation != nil ||
				by != want {
				t.Errorf("the deactivation = %v, the creation = %d %s, bob's ended memberships by their last writers %s; want the deactivation "+
					"done, the creation %s, %s", deactivation, rec.Code, rec.Body, by, map[bool]string{true: "done",
					false: "refused at project_lead_id, and no Ops"}[creationFirst], want)
			}
			if !creationFirst && !refusedAsEnded(rec, "project_lead_id") {
				t.Errorf("the creation = %d %s, want 422 at project_lead_id", rec.Code, rec.Body)
			}
		})
	}
}

// soleAdminCheckedHolding stops a deactivation between its check of rule 2
// and its writes: once it has asked whether the account is the only active
// admin of one of its workspaces, holding the account and every workspace
// of it FOR NO KEY UPDATE.
type soleAdminCheckedHolding struct {
	*workspacepg.Store
	gate *gate
}

func (m soleAdminCheckedHolding) SoleAdmin(ctx context.Context, workspaceIDs []uuid.UUID, userID uuid.UUID) (bool, error) {
	sole, err := m.Store.SoleAdmin(ctx, workspaceIDs, userID)
	if err != nil {
		return false, err
	}
	return sole, m.gate.wait(ctx)
}

// The world's standing (worldStanding) with alice's or gina's memberships
// ended.
const (
	aliceDeactivated = "acme: bob 15, carol 15, dave 15, erin 5, gina 20; beta: bob 20, carol 15; Lab: bob 20, carol 15; " +
		"Ops: bob 15, carol 20; Web: bob 20, carol 15, dave 20, erin 5, gina 15"
	ginaDeactivated = "acme: alice 20, bob 15, carol 15, dave 15, erin 5; beta: bob 20, carol 15; Lab: bob 20, carol 15; " +
		"Ops: alice 20, bob 15, carol 20; Web: alice 20, bob 20, carol 15, dave 20, erin 5"
)

// Rule 2 under two deactivations at once (M3 design 3.7, 3.6 convention
// 6): acme's two admins, alice and gina, deactivated at once; bob, carol,
// dave and erin are its other active members. The first holds her account
// and acme FOR NO KEY UPDATE and waits at her gate: between her check of
// rule 2 and her writes; or once her membership has ended. The second
// waits for acme's row. Once the first has committed, the second finds
// herself acme's only active admin, with other active members:
// workspace.sole_admin, and her account is active still. acme keeps an
// admin; each membership of the first ended, of acme and of its projects,
// is hers. Held between her check and her writes, the first shows that the
// check holds the lock her writes do: a deactivation that let acme go
// between the two would leave the second nothing to wait for, and both
// would end.
func TestTwoAdminsDeactivatedAtOnceLeaveAnAdmin(t *testing.T) {
	for _, at := range []struct {
		name    string
		holding func(store *workspacepg.Store, g *gate) workspaceapp.AllMembershipsEnder
	}{
		{"at her check", func(store *workspacepg.Store, g *gate) workspaceapp.AllMembershipsEnder {
			return soleAdminCheckedHolding{store, g}
		}},
		{"past her membership's end", func(store *workspacepg.Store, g *gate) workspaceapp.AllMembershipsEnder {
			return endedHoldingAll{store, g}
		}},
	} {
		for _, aliceFirst := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s, alice first %v", at.name, aliceFirst), func(t *testing.T) {
				w := newMemberWorld(t)
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				first, second, want := "alice", "gina", aliceDeactivated
				if !aliceFirst {
					first, second, want = second, first, ginaDeactivated
				}
				deactivate := func(name string, memberships workspaceapp.AllMembershipsEnder) func() error {
					return func() error {
						_, err := deactivating(w.pool, identitypg.New(w.pool), memberships).ExecuteByEmail(ctx, name+"@example.com")
						return err
					}
				}
				g := newGate()
				ended := run(deactivate(first, at.holding(workspacepg.New(w.pool), g)))
				held(t, ctx, g, ended, "the first deactivation")
				refused := run(deactivate(second, workspacepg.New(w.pool)))
				pgtest.WaitForLockWaitOn(t, w.pool, "workspaces", 5*time.Second)
				if got := w.standing(t); got != worldStanding {
					t.Errorf("while the first holds acme: %s; want %s", got, worldStanding)
				}
				close(g.open)

				if err := result(t, ctx, ended, "the first deactivation"); err != nil {
					t.Errorf("the first deactivation = %v, want it done", err)
				}
				if err := result(t, ctx, refused, "the second deactivation"); !sameOutcome(err, workspacedomain.ErrSoleAdmin) {
					t.Errorf("the second deactivation = %v, want workspace.sole_admin", err)
				}
				var active bool
				if err := w.pool.QueryRow(soon(t), "SELECT is_active FROM users WHERE id = $1", w.ids[second]).Scan(&active); err != nil || !active {
					t.Errorf("%s's account active %v (%v); want it active still", second, active, err)
				}
				enders := map[string]string{"alice": "Ops alice, Web alice, acme alice", "gina": "Web gina, acme gina"}[first]
				if got := endersOf(t, w.pool, w.ids[first]); got != enders {
					t.Errorf("%s's ended memberships by their last writers: %s; want %s", first, got, enders)
				}
				if got := w.standing(t); got != want {
					t.Errorf("after both: %s; want %s", got, want)
				}
			})
		}
	}
}
````

- [ ] **Step 2: 全部的交错**

Run: `go -C server test -count=5 -race -run 'TestDeactivationFirstRefusesTheWorkspace$|TestCreationFirstHoldsOffTheDeactivation$|TestDecliningAndDeactivating$|TestAcceptingAndDeactivating$|TestAnAdmittedAdminsWriteAndHisDeactivation$|TestReactivatingAndDeactivating$|TestADeactivationAndTheProjectSidesGrowthSerialize$|TestADeactivationAndACreationHeLeadsSerialize$|TestTwoAdminsDeactivatedAtOnceLeaveAnAdmin$' ./internal/bootstrap/`
Expected: `ok`；输出里没有 `40P01`，没有数据竞争。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 3: 提交**

```bash
git add server/internal/bootstrap/interleaving_shrinking_test.go
```
```bash
git commit -m "test(M3/P6): interleavings 13, 14 and 15, and two admins deactivated at once

A deactivation and the project side's growth, an addition or a join,
for a new membership and a former one, and a creation he leads,
serialize on the workspace's row in both orders: growth first is found
and ended; deactivation first is refused as ended. Two admins
deactivated at once leave an admin: the second, held at the
workspace's row behind the first's check, is refused.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A；`mutants_p6.py` 的编号，"层"是它被发现的每一层；端到端是单独运行的故事）：

| 变异 | 改坏 | 必须失败的测试 | 层 |
|---|---|---|---|
| `s32-wait-moved` | 工作区只取 `FOR KEY SHARE`，改锁这些工作区的全部有效成员行 `FOR UPDATE`（结果不变，等待挪到 `workspace_members`） | `TestADeactivationAndACreationHeLeadsSerialize`、`TestADeactivationAndTheProjectSidesGrowthSerialize`、`TestADeactivationFindsWhatChangedMeanwhile`、`TestAnAdmittedAdminsWriteAndHisDeactivation` 等 6 个 | 组合 |
| `s32-wait-moved-blind` | 同上，本 Task 的三个测试的探针改成不认表的 `WaitForLockWait`：这三个测试通过（探针换成不认表的就看不出），只在探针仍认表的别的测试失败 | `TestADeactivationFindsWhatChangedMeanwhile`、`TestAnAdmittedAdminsWriteAndHisDeactivation`、`TestEachLockOfADeactivationIsItsStrength` | 组合 |

**Done when:** 交错 7、8、13–16、19 和两位管理员在 `-count=5 -race` 下通过，没有 40P01；`make lint-go`、`make test` 通过。

### Task 9: 故事 W9；文档

**Files:**
- Create: `e2e/stories/workspace/w9-deactivation.spec.ts`
- Modify: `README.md`、`docs/v0/M3-workspace-project/handoffs/M2-closeout.md`、`docs/v0/plane-diff.md`、`docs/v0/v0-design.md`、`e2e/stories/identity/a12-deactivate.spec.ts`

**Interfaces:**
- 没有产品代码。W9 用 e2e 已有的夹具（`createWorkspace`、`inviteAndAccept`、`invite`、`createProject`、`addProjectMembers`、`membershipOf`、`accountOf`、`tokensOf`、`expectDeactivated`、`expectMembership`、`nerveUsers`、`nerveUsersFails`、`nerveWorkspaces`）。
- 文档：README 部署一节的 `deactivate`、`activate`、"恢复被移出的成员"（8.7）；差异清单第四节"停用账户"（3.20）；总体设计 4.2 的"停用一段"（3.20 的 P2 一行）；M2 收尾交接第 6 节的处理结果（关闭；M4 交接第 1 节不提前）。

**Tests:**
- `e2e/stories/workspace/w9-deactivation.spec.ts`（spec 2.11）：两个拒绝各经接口、命令，每次之前读、之后核对六张表；A 加入 Lab 之后接口停用 B：A12 的断言，他的五行由他在一个时刻结束或删除（按 `COLLATE "C"` 排序），别人的每一行不变；`activate` 只恢复账户，`reactivate-member` 恢复 acme（B 读得到 acme、读不到 beta）；命令再停用一次，别人的不变。
- `a12-deactivate.spec.ts`：命令的新一行。

- [ ] **Step 1: 故事**

`e2e/stories/workspace/w9-deactivation.spec.ts`（新文件，166 行）：

````file e2e/stories/workspace/w9-deactivation.spec.ts
import {
  addProjectMembers,
  createProject,
  createWorkspace,
  invite,
  inviteAndAccept,
  membershipOf,
  slugFor,
} from "../../fixtures/api";
import { accountOf, expectDeactivated, tokensOf } from "../../fixtures/assert/identity";
import { expectMembership } from "../../fixtures/assert/workspace";
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";
import { nerveUsers, nerveUsersFails } from "../../fixtures/users";
import { nerveWorkspaces } from "../../fixtures/workspaces";

// W9, a deactivation ends every membership of the account (M3 design 2, 3.9; M2 handoff 6): the API version, with the
// command. The page's is P9's.

/** The two refusals' details: the API's problem and the command's line say the same. */
const workspaceRefusal =
  "The workspace would be left without an admin: its only active admin cannot leave it, nor can his membership end while it has other active members. It must first be given another admin, or be deleted.";
const projectRefusal =
  "The project would be left without an admin: its only active admin cannot leave it, nor can his membership end while it has other active members. It must first be given another admin, or be deleted.";

test("W9: a deactivation is refused while the account is the only admin of a workspace or a project with other members, by the API and by the command, and changes nothing; once each has another admin, it ends every membership and deletes every invitation to the address; nerve users activate, then reactivate-member, bring the account back a workspace at a time", async ({
  api,
  db,
}, testInfo) => {
  const aEmail = emailFor(testInfo, "a");
  const a = (await createPAT(api, (await register(api, aEmail)).access_token)).token;
  const bEmail = emailFor(testInfo, "b");
  const b = (await createPAT(api, (await register(api, bEmail)).access_token)).token;
  const cEmail = emailFor(testInfo, "c");
  const c = (await createPAT(api, (await register(api, cEmail)).access_token)).token;
  const acme = slugFor(testInfo, "acme");
  const beta = slugFor(testInfo, "beta");
  const gamma = slugFor(testInfo, "gamma");
  const delta = slugFor(testInfo, "delta");
  // B makes acme, A its member; A makes beta, B and C its members; B makes Lab there and adds C as its member. C
  // invites B to gamma, which he declines, and to delta.
  await createWorkspace(api, b, { name: "Acme", slug: acme });
  await inviteAndAccept(api, b, acme, { email: aEmail, token: a }, 15);
  await createWorkspace(api, a, { name: "Beta", slug: beta });
  await inviteAndAccept(api, a, beta, { email: bEmail, token: b }, 15);
  await inviteAndAccept(api, a, beta, { email: cEmail, token: c }, 15);
  const lab = await createProject(api, b, beta, { name: "Lab", identifier: "LAB" });
  await addProjectMembers(api, b, lab.id, [{ member_id: await accountId(api, c), role: 15 }]);
  await createWorkspace(api, c, { name: "Gamma", slug: gamma });
  await createWorkspace(api, c, { name: "Delta", slug: delta });
  const [toGamma] = await invite(api, c, gamma, [{ email: bEmail, role: 15 }]);
  await invite(api, c, delta, [{ email: bEmail, role: 15 }]);
  if (!toGamma) {
    throw new Error("the invitation to B in gamma was not created");
  }
  const declined = await api.POST("/api/v0/workspace-invitations/{invitation_id}/decline", {
    params: { path: { invitation_id: toGamma.id } },
    body: { token: toGamma.token },
    headers: bearer(b),
  });
  expect(declined.response.status, "B declines gamma's invitation").toBe(204);
  // The tables a deactivation writes, whole.
  const tables = async () => ({
    users: await db.query("SELECT * FROM users ORDER BY id"),
    profiles: await db.query("SELECT * FROM profiles ORDER BY id"),
    sessions: await db.query("SELECT * FROM auth_sessions ORDER BY id"),
    workspaces: await db.query("SELECT * FROM workspace_members ORDER BY id"),
    projects: await db.query("SELECT * FROM project_members ORDER BY id"),
    invitations: await db.query("SELECT * FROM workspace_member_invites ORDER BY id"),
  });
  const deactivate = async () => {
    const answer = await api.POST("/api/v0/me/deactivate", { headers: bearer(b) });
    return [answer.response.status, answer.error?.code, answer.error?.detail];
  };
  const refusedBoth = async (code: string, detail: string) => {
    const before = await tables();
    expect(await deactivate(), "deactivateMe").toEqual([409, code, detail]);
    expect(await tables(), `the tables after deactivateMe's ${code}`).toEqual(before);
    await nerveUsersFails(db, ["deactivate", "--email", bEmail], detail);
    expect(await tables(), `the tables after the command's ${code}`).toEqual(before);
  };

  // B is acme's only admin, A its member: neither the API nor the command deactivates him.
  await refusedBoth("workspace.sole_admin", workspaceRefusal);

  // B makes A acme's admin; he is Lab's only admin still, C its member, in beta, where he is a member.
  const aInAcme = await api.PATCH("/api/v0/workspace-members/{workspace_member_id}", {
    params: { path: { workspace_member_id: await membershipOf(api, b, acme, await accountId(api, a)) } },
    body: { role: 20 },
    headers: bearer(b),
  });
  expect(aInAcme.response.status, "B makes A acme's admin").toBe(200);
  await refusedBoth("project.sole_admin", projectRefusal);

  // A, beta's admin, joins Lab, as its admin (M3 design 3.5). The API deactivates B: as A12, and every membership of
  // his ends, every invitation to his address is deleted, the declined one too, at one moment, by him; every row of
  // anyone else is as it was.
  const joined = await api.POST("/api/v0/projects/{project_id}/join", {
    params: { path: { project_id: lab.id } },
    headers: bearer(a),
  });
  expect(joined.response.status, "A joins Lab").toBe(200);
  const account = await accountOf(db, bEmail);
  const tokensBefore = await tokensOf(db, account.id);
  const others = async () => ({
    users: await db.query("SELECT * FROM users WHERE id <> $1 ORDER BY id", [account.id]),
    profiles: await db.query("SELECT * FROM profiles WHERE user_id <> $1 ORDER BY id", [account.id]),
    sessions: await db.query("SELECT * FROM auth_sessions WHERE user_id <> $1 ORDER BY id", [account.id]),
    workspaces: await db.query("SELECT * FROM workspace_members WHERE member_id <> $1 ORDER BY id", [account.id]),
    projects: await db.query("SELECT * FROM project_members WHERE member_id <> $1 ORDER BY id", [account.id]),
    invitations: await db.query("SELECT * FROM workspace_member_invites WHERE email <> $1 ORDER BY id", [bEmail]),
  });
  const othersBefore = await others();
  expect(await deactivate(), "deactivateMe").toEqual([204, undefined, undefined]);
  await expectDeactivated(db, account, tokensBefore);
  // Each of his rows: where, whether it stands, who wrote it last; and how many moments they were written at.
  const ended = async () =>
    db.query(
      `WITH r AS (
         SELECT w.slug AS place, m.is_active AS active, b.email AS by, m.updated_at AS written
           FROM workspace_members m JOIN workspaces w ON w.id = m.workspace_id JOIN users b ON b.id = m.updated_by_id
          WHERE m.member_id = $1
         UNION ALL SELECT p.identifier, m.is_active, b.email, m.updated_at
           FROM project_members m JOIN projects p ON p.id = m.project_id JOIN users b ON b.id = m.updated_by_id WHERE m.member_id = $1
         UNION ALL SELECT 'invitation to ' || w.slug, i.deleted_at IS NULL, b.email, i.deleted_at
           FROM workspace_member_invites i JOIN workspaces w ON w.id = i.workspace_id JOIN users b ON b.id = i.updated_by_id
          WHERE i.email = $2 AND NOT i.accepted)
       SELECT place, active, by, (SELECT count(DISTINCT written) FROM r) AS moments FROM r ORDER BY place COLLATE "C"`,
      [account.id, bEmail]
    );
  const row = (place: string, active: boolean) => ({ place, active, by: bEmail, moments: "1" });
  expect(await ended(), "B's memberships and the invitations to him").toEqual(
    [
      row("LAB", false),
      row(acme, false),
      row(beta, false),
      row(`invitation to ${delta}`, false),
      row(`invitation to ${gamma}`, false),
    ].toSorted((x, y) => (x.place < y.place ? -1 : 1))
  );
  expect(await others(), "every row of anyone else").toEqual(othersBefore);

  // The way back: nerve users activate brings the account back alone; reactivate-member, his membership of one
  // workspace, as it was, his project memberships still ended.
  expect(await nerveUsers(db, ["activate", "--email", bEmail])).toBe(
    `activated ${bEmail}: 1 API tokens are usable again\n`
  );
  await expectMembership(db, acme, bEmail, { role: 20, is_active: false });
  expect(await nerveWorkspaces(db, ["reactivate-member", "--slug", acme, "--email", bEmail])).toBe(
    `reactivated ${bEmail} in ${acme} as admin; project memberships still ended: 0, each restored when the member joins or is added to its project\n`
  );
  await expectMembership(db, acme, bEmail, { role: 20, is_active: true });
  await expectMembership(db, beta, bEmail, { role: 15, is_active: false });
  const sees = async (slug: string) =>
    (await api.GET("/api/v0/workspaces/{slug}", { params: { path: { slug } }, headers: bearer(b) })).response.status;
  expect([await sees(acme), await sees(beta)], "B reads acme, not beta").toEqual([200, 404]);

  // The command deactivates him again: his membership of acme, the one active, ends, as the API's did.
  const again = await accountOf(db, bEmail);
  expect(await nerveUsers(db, ["deactivate", "--email", bEmail])).toBe(
    `deactivated ${bEmail}: revoked 0 sessions and ended its memberships; to bring it back, run nerve users activate, then nerve workspaces reactivate-member in each workspace\n`
  );
  await expectDeactivated(db, again, tokensBefore);
  await expectMembership(db, acme, bEmail, { role: 20, is_active: false });
  expect(await others(), "every row of anyone else").toEqual(othersBefore);
});
````

`e2e/stories/identity/a12-deactivate.spec.ts`（修改，1 处）：

````old e2e/stories/identity/a12-deactivate.spec.ts
  expect(await nerveUsers(db, ["deactivate", "--email", email])).toBe(`deactivated ${email}: revoked 1 sessions\n`);
````
````new e2e/stories/identity/a12-deactivate.spec.ts
  expect(await nerveUsers(db, ["deactivate", "--email", email])).toBe(
    `deactivated ${email}: revoked 1 sessions and ended its memberships; to bring it back, run nerve users activate, then nerve workspaces reactivate-member in each workspace\n`
  );
````

- [ ] **Step 2: 文档**

`README.md`（修改，2 处）：

````old README.md
  - `deactivate --email <邮箱>`：停用账户，与用户自己停用相同：结束全部会话，重置新手引导，不改密码；PAT 保留，但停用期间认证失败。
  - `activate --email <邮箱>`：恢复账户，没有过期的 PAT **重新可用**，输出它们的个数。
````
````new README.md
  - `deactivate --email <邮箱>`：停用账户，与用户自己停用相同：结束全部会话，重置新手引导，不改密码；PAT 保留，但停用期间认证失败。同一个事务里结束他在全部工作区和项目的成员关系（行保留，角色不变），删除发给他邮箱的全部邀请（待接受的和已忽略的）；输出提示回来的两步。他是某个工作区或项目唯一的有效管理员、那里还有别的有效成员时，退出码为 1，打印一行说明（与接口的 409 相同），数据库不变：先在那里指定另一位管理员，或删除它。命令没有请求期限：它取账户行和他的每个工作区、项目的锁，与这些工作区里的写连续重叠时一直等待；中断（SIGINT、SIGTERM）之后什么都不改，可以重试。
  - `activate --email <邮箱>`：恢复账户，没有过期的 PAT **重新可用**，输出它们的个数。**不恢复成员关系**：停用结束的成员关系按工作区用 `nerve workspaces reactivate-member` 恢复（下面"恢复被移出的成员"一条）。
````

````old README.md
- **恢复被移出的成员**：`nerve workspaces reactivate-member --slug <slug> --email <邮箱>` 把这个账户在这个工作区已结束的成员关系（被移出或自己离开）恢复为有效，角色不变。他在这个工作区的项目成员关系仍无效，命令输出这样的项目成员关系还有几个：每一个在他经接口加入那个项目时恢复，角色取原来的项目角色与他的工作区角色中较低的一个（工作区管理员能加入任何项目，成员只能加入公开的项目，访客不能加入；M3 设计 3.5），或在有权添加项目成员的人（项目管理员，或同时是工作区管理员的项目成员，M3 设计 3.4）重新添加他时恢复，角色取添加时给出的。账户已停用时照样恢复，输出另外提示下一步执行 `nerve users activate --email <邮箱>`：`nerve users activate` 只恢复账户，不恢复成员关系。它与 `nerve workspaces create` 一样直接连数据库，服务不用停；邮箱按注册时的规则规范化。已是有效成员时输出说明，退出码为 0，什么都不改；工作区不存在、邮箱没有账户、账户从来不是这个工作区的成员时，退出码为 1，打印一行说明，数据库不变。它取工作区的锁：同一工作区里项目级的写连续重叠时一直等待（命令没有请求期限）；中断（SIGINT、SIGTERM）之后什么都不改，可以重试。
````
````new README.md
- **恢复被移出的成员**：`nerve workspaces reactivate-member --slug <slug> --email <邮箱>` 把这个账户在这个工作区已结束的成员关系（被移出、自己离开或账户停用）恢复为有效，角色不变。他在这个工作区的项目成员关系仍无效，命令输出这样的项目成员关系还有几个：每一个在他经接口加入那个项目时恢复，角色取原来的项目角色与他的工作区角色中较低的一个（工作区管理员能加入任何项目，成员只能加入公开的项目，访客不能加入；M3 设计 3.5），或在有权添加项目成员的人（项目管理员，或同时是工作区管理员的项目成员，M3 设计 3.4）重新添加他时恢复，角色取添加时给出的。账户已停用时照样恢复，输出另外提示下一步执行 `nerve users activate --email <邮箱>`：`nerve users activate` 只恢复账户，不恢复成员关系。停用过的账户回来是两步：`nerve users activate`，再在要回去的每个工作区执行一次这个命令。它与 `nerve workspaces create` 一样直接连数据库，服务不用停；邮箱按注册时的规则规范化。已是有效成员时输出说明，退出码为 0，什么都不改；工作区不存在、邮箱没有账户、账户从来不是这个工作区的成员时，退出码为 1，打印一行说明，数据库不变。它取工作区的锁：同一工作区里项目级的写连续重叠时一直等待（命令没有请求期限）；中断（SIGINT、SIGTERM）之后什么都不改，可以重试。
````

`docs/v0/plane-diff.md`（修改，1 处）：

````old docs/v0/plane-diff.md
| 停用账户 | 自助停用：撤销会话、重置新手引导、把密码改成随机值、发邮件；"唯一管理员"的检查从不拒绝；只有命令 `activate_user` 能恢复 | 自助停用 `POST /api/v0/me/deactivate`：撤销全部会话，重置新手引导，不改密码，PAT 不删除但停用期间认证失败；服务器管理员的 `nerve users deactivate` 与自助停用相同，`nerve users activate` 恢复账户，恢复后没有过期的 PAT 重新可用；"唯一管理员"的检查由 M3 在同一个事务里实现（M2 设计决策点 3） |
````
````new docs/v0/plane-diff.md
| 停用账户 | 自助停用：撤销会话、重置新手引导、把密码改成随机值、发邮件；"唯一管理员"的检查从不拒绝；只有命令 `activate_user` 能恢复 | 自助停用 `POST /api/v0/me/deactivate`：撤销全部会话，重置新手引导，不改密码，PAT 不删除但停用期间认证失败；服务器管理员的 `nerve users deactivate` 与自助停用相同，`nerve users activate` 恢复账户，恢复后没有过期的 PAT 重新可用；停用在同一个事务里结束他在全部工作区和项目的成员关系（行保留），软删除发给他邮箱的全部邀请（待接受的和已忽略的，与 Plane 相同）；他是某个工作区或项目唯一的有效管理员、那里还有别的有效成员时拒绝（409 `workspace.sole_admin`、`project.sole_admin`，命令退出码 1），什么都不改；`nerve users activate` 只恢复账户，成员关系按工作区用 `nerve workspaces reactivate-member` 恢复（M2 设计决策点 3，M3 设计 3.9） |
````

`docs/v0/v0-design.md`（修改，1 处）：

````old docs/v0/v0-design.md
  "先锁父行，再判定"从 M3/P2 起落地（修改、删除工作区，改成员的角色，修改显示设置）：两位管理员互相降级时，先拿到工作区锁的一方成功，后到的一方判定时已是成员而被拒绝。约定六的接受邀请一段从 M3/P3 起落地：接受、忽略邀请最先以 `FOR SHARE` 锁住调用者的账户行，在锁下重读 `is_active` 和邮箱，再锁工作区（接受 `FOR NO KEY UPDATE`，忽略 `FOR SHARE`）和邀请行（`FOR UPDATE`）；接受比较的是在账户行锁下读到的邮箱，与改邮箱在账户行上串行：改邮箱先持有这一行时，接受等它提交，比较的是改后的邮箱；接受先持有时，比较的是原来的邮箱，改邮箱等接受结束（M3 设计 9.3 交错 12）。停用一段在 M3/P6 随实现核对。
````
````new docs/v0/v0-design.md
  "先锁父行，再判定"从 M3/P2 起落地（修改、删除工作区，改成员的角色，修改显示设置）：两位管理员互相降级时，先拿到工作区锁的一方成功，后到的一方判定时已是成员而被拒绝。约定六的接受邀请一段从 M3/P3 起落地：接受、忽略邀请最先以 `FOR SHARE` 锁住调用者的账户行，在锁下重读 `is_active` 和邮箱，再锁工作区（接受 `FOR NO KEY UPDATE`，忽略 `FOR SHARE`）和邀请行（`FOR UPDATE`）；接受比较的是在账户行锁下读到的邮箱，与改邮箱在账户行上串行：改邮箱先持有这一行时，接受等它提交，比较的是改后的邮箱；接受先持有时，比较的是原来的邮箱，改邮箱等接受结束（M3 设计 9.3 交错 12）。约定六的停用一段从 M3/P6 起落地：停用（自助的和服务器管理员的命令）最先以 `FOR NO KEY UPDATE` 锁住账户行，在锁下读邮箱，写完账户、新手引导和会话之后，按 `id` 锁住他有效成员关系所在的全部工作区（`FOR NO KEY UPDATE`，上锁时已删除的跳过），在最后一把工作区锁之后读时刻，删除发给这个邮箱的全部邀请，结束他的工作区成员行，再列举他的项目成员关系、按 `id` 一次锁住那些项目并结束它们；工作区一侧的增长在账户行上、项目一侧的增长在工作区行上与它串行（M3 设计 3.9，9.3 交错 7、8、13–16、19）。
````

`docs/v0/M3-workspace-project/handoffs/M2-closeout.md`（修改，1 处）：

````old docs/v0/M3-workspace-project/handoffs/M2-closeout.md
来源：[M3/P4a spec](../specs/P4a-projects.md) 第 7 节。

````
````new docs/v0/M3-workspace-project/handoffs/M2-closeout.md
来源：[M3/P4a spec](../specs/P4a-projects.md) 第 7 节。

## 处理结果（M3/P6）

- **第 6 节 停用的端口**（完成）：`identity/app` 声明 `MembershipDeactivator`（`DeactivateMemberships(ctx, userID, email)`），`deactivate`（`server/internal/modules/identity/app/deactivate.go`）在撤销会话之后调用它，自助停用（`Execute`）和 `nerve users deactivate`（`ExecuteByEmail`）都经过这里，邮箱取自锁下的账户行。它由 `workspace` 的 `Deactivator` 实现（`server/internal/modules/workspace/app/deactivate_memberships.go`），`bootstrap` 接上：服务用 `workspace.New` 的 `Deactivator()`，命令行用 `workspace.NewDeactivator` 和 `project.NewCascade`（`server/internal/bootstrap/users.go`）。三件事在停用的事务里：按 id 锁住他所在的全部工作区之后，他是某个工作区或项目唯一的有效管理员、那里还有别的有效成员时拒绝（409 `workspace.sole_admin`、`project.sole_admin`，数据库不变）；删除发给他邮箱的全部邀请（待接受的和已忽略的）；结束他在全部工作区和项目的成员关系，行和角色留着。`deactivateMe` 声明这两个码，`identity` 自己的 HTTP 测试返回过它们，`apitest` 两个方向的核对通过；接口和命令两条路各有组合测试（`server/internal/bootstrap/deactivation_test.go`、`deactivation_locks_test.go`、`deactivation_races_test.go`）和故事 W9；差异清单第四节"停用账户"一行已更新；这些写入在全局加锁顺序中的位置写在 M3 设计 3.6（账户行、工作区、邀请、工作区成员、项目、项目成员）。
- **第 6 节的"注意"**（M4 交接第 1 节的"只投递"River 客户端）：停用的端口不投递任务，命令行的组合仍没有 River 客户端，这一项不提前到 M3，照原计划留在 M4（M3 设计 3.9、13.1）。

仍未处理，状态保持 `open`：第 1 节的页面一侧（P9）；第 2、3 节，第 7 节的其余部分，第 10、11、13、14 节，随 M3 设计 13.1 中各自的 Phase；第 12 节等 M3 的收尾。

来源：[M3/P6 spec](../specs/P6-deactivation.md) 第 7 节。

````

- [ ] **Step 3: 检查**

Run: `make lint-web`
Expected: 通过。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 67 个故事全部通过（此前的 66 个，加 W9）。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 4: 提交**

```bash
git add README.md docs/v0/M3-workspace-project/handoffs/M2-closeout.md docs/v0/plane-diff.md docs/v0/v0-design.md e2e/stories/identity/a12-deactivate.spec.ts e2e/stories/workspace/w9-deactivation.spec.ts
```
```bash
git commit -m "test(M3/P6): story W9's API version, with the command, and the deactivation's documentation

W9 refuses a deactivation through the API and the command while the
account is a workspace's or a project's only admin, each refusal
checked against every table read before it; once each has another
admin, deactivateMe ends every membership and deletes every invitation
to the address, by the account at one moment, and nothing of anyone
else changes; nerve users activate, then reactivate-member, bring it
back a workspace at a time. The README, the Plane difference list, the
v0 design's convention 6 and the M2 handoff say what a deactivation
now does.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A；`mutants_p6.py` 的编号，"层"是它被发现的每一层；端到端是单独运行的故事）：

| 变异 | 改坏 | 必须失败的测试 | 层 |
|---|---|---|---|
| `s39-w9-collation` | W9 的行按数据库的排序规则排序 | W9 | 端到端 |

**Done when:** `make e2e` 通过（W9 在内）；README 的三条、差异清单的一行、总体设计 4.2 的一段、M2 收尾交接的处理结果写好；`grep -n "停用一段在 M3/P6 随实现核对" docs/v0/v0-design.md` 没有输出；`make lint-go`、`make test`、`make lint-web`、`make knip`、`make test-web` 通过。
