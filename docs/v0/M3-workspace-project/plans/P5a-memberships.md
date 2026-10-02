# M3/P5a 结束与恢复工作区的成员关系 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 移出工作区成员、离开工作区，两者共用的结束一步（软删除发给他的待接受邀请、结束成员关系、`EndMemberships` 结束他在这个工作区各项目的成员关系），M3 设计 3.7 的规则 1（离开）和规则 2（连带结束）在工作区一侧的部分；`nerve workspaces reactivate-member` 恢复被结束的成员关系。两个操作各带操作名、规则行和矩阵行（25 格）；三个写都取工作区行的 `FOR NO KEY UPDATE`、在它之后读时刻（3.3），连带按 id 升序以 `FOR NO KEY UPDATE` 锁住他的项目、在锁下查唯一管理员（约定六）；交错 1 的工作区一侧、4、5、6 在真实数据库上两种顺序都串行；S2 和 9d 顺序的集成测试；故事 W2、W7、W12 的接口版本通过；README 的部署一节、差异清单中 P5a 的三行写好。

**Architecture:** 只动 `workspace`、`project` 两个模块，`access` 的规则表，`bootstrap`，`cmd/nerve` 和 e2e；不加迁移。跨模块端口多两个方法：`ProjectCascade.EndMemberships`（`project.Cascade` 的第三个方法，设计 6.5，`by` 照 G1）和 `ProjectMembershipCounts.CountInactive`（`project.Provide` 提供，设计 6.5），两者的参数和回答都只是 `uuid`、`time.Time`、`int`，`bootstrap` 直接接上，不需要转换，模块之间不导入、不 JOIN。`workspace`：存储（`EndMember`、`DeletePendingInvitations`、`HasOtherAdmin`、`ReactivateMember`）；app（结束一步 `membershipEnd`，移出、离开共用；`lockedMember` 从 `updateWorkspaceMember` 提出，移出也用；三个用例 `RemoveWorkspaceMember`、`LeaveWorkspace`、`ReactivateMember`；按用例分的端口）；HTTP（`api/modules/workspace.yaml` 的两个操作）；`workspace.NewAdmin` 多一个恢复的用例。`project`：`Cascade.EndMemberships` 和它的三条语句（`LockActiveMemberProjects`、`SoleAdmin`、`EndMemberships`），`CountInactive`。`access`：两条规则。`bootstrap`：两个操作的接线；`nerve workspaces` 的组合多 `ProjectMembershipCounts`，不建项目的连带（设计 6.6）；组合出的结束、恢复、交错、集成测试；矩阵的"被移出的成员"改由存储写出、唯一管理员的一张表。

**Tech Stack:** Go 1.27.1、pgx v5.11.0、sqlc v1.31.1（`CGO_ENABLED=0`）、oapi-codegen v2.8.0、goose v3.28.0、River v0.47.0、golangci-lint 2.13.2、PostgreSQL 18.6（testcontainers）；Node 24、pnpm 11.10.0、Playwright 1.63.0。

**Spec:** `docs/v0/M3-workspace-project/specs/P5a-memberships.md`（上级：`docs/v0/M3-workspace-project/M3-design.md`）

## Global Constraints

- **Go 版本和依赖**：本 plan 不执行 `go get`，不加任何 Go 模块或 npm 包。`server/go.mod` 和 `server/tools/go.mod` 保持 `go 1.27` 和 `toolchain go1.27.1`（golangci-lint 2.13.2 由 go1.27.0 构建，`go` 行更高就拒绝运行）。每个 Task 提交前执行 `grep -n "^go \|^toolchain" server/go.mod server/tools/go.mod`，四行必须是 `go 1.27` 和 `toolchain go1.27.1`；`git diff --stat b5e826b3 -- server/go.mod server/go.sum server/tools pnpm-lock.yaml` 在本 plan 的任何时刻都没有输出。
- **每个 Task 提交前**：`make lint-go` 两段都输出 `0 issues.`，`make test` 全部通过（没有 `FAIL`）。
- **生成的文件不手写、不从本 plan 复制**：有生成物的 Task（1、2、4、5、7）执行 Task 中的生成命令（改了接口描述的 Task 4、5 执行 `make gen`，只动了 sqlc 的 Task 1、2、7 执行 `make gen-go`），用 `shasum -a 256` 和 `wc -l` 核对表中的 SHA-256 和行数，对不上时停下来：说明某个输入与本 plan 不一致。这些 Task **提交之后**执行 `make gen-check`（它用 `git status` 判断生成物是否已提交，提交之前必然报差异）。
- **前端检查**：改了接口描述、前端或 `e2e/` 的 Task（4、5、13、14）另执行 `make lint-web`、`make knip`、`make test-web`；声明新错误码的 Task 4（`project.sole_admin`：`project` 模块的码，第一次由工作区的操作声明）和 Task 5（`workspace.sole_admin`）在同一个 Task 里把码加进 `PROBLEM_MESSAGES` 和两份 `auth.json`（M3 设计 12 节约束 4）；`reactivate-member` 的两个码只在命令行（spec 第 3 节第 3 条），不进契约和前端；Task 13、14 执行 `make e2e`。
- **容器**：`make test` 和 `make e2e` 用自己的 testcontainers；机器忙时偶尔起不来，等 Docker 空闲之后重跑一次再当作失败。容器测试一次只跑一套。开发库 `nerve-dev-db-1` 可以用，但不要停止或重建它，不要执行 `make dev-db-down`、`make dev-db-reset`。不要碰其他项目的容器（`agentforge-*`、`plane-app-*`、`opennerve-*`、`nervewiki-*`）。
- **git**：每次 Bash 调用只执行一个 git 命令，不用 `;`、`&&`、`|` 串联 git；不用 `git -C`、`stash`、`clean`、`reset --hard`。`cd` 不与别的命令组合，只读的命令也不行。不碰 `plane/`、`refer/`。
- **安装**：除了 Docker、Go、Node 不做任何全局安装；不执行 `corepack enable`（pnpm 已在 PATH 上）。
- **规则**：不写 `init()`，不用全局可变状态，构造函数显式传入依赖；不建 `utils`、`common`、`helpers` 包；一个文件只做一件事，不超过约 400 行；依赖只能向内；模块之间不互相导入，值在 `bootstrap` 中转换或直接接上（M3 设计 6.5、6.6），不跨模块的表 JOIN；模块的 SQL 只经 sqlc；角色只按集合判断（规则表、`role = 20` 只出现在"管理员"这一个集合的查询里），不按大小比较；不留没有使用者的代码（`project.NewCascade` 不建，裁定 G2；`nerve workspaces` 的组合不建项目的连带）。**例外**：接口描述按模块一个文件（M0-P3 交接 5），`api/modules/workspace.yaml`（869 行）不受约 400 行的限制。本 plan 的其余文件都在约 400 行以内：最长的是 `bootstrap/interleaving_growth_test.go`（404 行，P4b 留下 402 行，本 plan 只改两处构造和一句注释）、`bootstrap/permission_matrix_test.go`（388 行）、`e2e/fixtures/assert/workspace.ts`（378 行）、`bootstrap/permission_matrix_coverage_test.go`（371 行）、`bootstrap/ending_test.go`（367 行）和 `project/app/ports.go`（351 行）。
- **注释**：Go、TS 代码、SQL 查询和接口描述用英文；中文文档照本 plan 原样。
- **代码块**：每个改动都写成四个反引号围起来的块，块的第一行写明种类和路径，照原样使用（原型中逐字节运行过）：
  - ````` ````file <路径> ````` 新文件，块的内容加一个结尾换行就是整个文件；
  - ````` ````whole <路径> ````` 已有文件的完整新内容（同样加结尾换行）；
  - ````` ````old <路径> ````` 与紧跟着的 ````` ````new <路径> `````：`old` 的文字在文件当前版本中恰好出现一次，把它换成 `new` 的文字；
  - ````` ````delete <路径> ````` 删除这个文件（块是空的）。

  一个文件的几个块按出现的顺序依次应用。拼 plan 的脚本已从 `b5e826b3` 起按顺序核对过全部块：每个 `old` 恰好出现一次（在它之前的块应用之后的文件中），每个新文件原来不存在，逐 Task 应用之后的文件与原型逐字节相同（spec 附录 A）。可以用 `node <planapply.mjs> <本 plan> apply <仓库根> <n>` 写入第 n 个 Task 的块，也可以手工照抄。
- **过渡版本**：一些文件先在较早的 Task 写成过渡版本，较晚的 Task 再修改（`api/modules/workspace.yaml`、`workspace/module.go`、`workspace/app/ports.go`、`fakes_members_test.go`、`fakes_projects_test.go`、`fakes_workspaces_test.go`、`clock_test.go`、`workspace/domain/actions.go`、`errors.go`、`adapter/http/handler.go`、`handler_test.go`、`members.go`、`members_test.go`、`queries/members.sql`、`project/module.go`、`access/domain/rules.go`、`rules_test.go`、矩阵的文件、前端文案、`bootstrap/interleaving_growth_test.go`、`interleaving_writes_test.go`、生成物）；`bootstrap/removal_test.go` 是 Task 4 的过渡文件，Task 5 把它换成同时测移出和离开的 `ending_test.go`（spec 第 3 节第 7 条）。每个过渡版本都在逐 Task 复现中运行过。
- **变异**：每个 Task 末尾的"变异"表列出：把代码改坏的方式、必须因此失败的测试和它所在的层（单元：假实现；存储：真实数据库；组合：`bootstrap` 组合出的 app、模块或命令；端到端：单独运行的故事）。它们在最终的原型上逐个跑过（`$M3TMP/p5tools/mutants_s1.py`、`mutants_p5a.py`、`mutants_probes.py`、`mutants_seed.py`、`mutants_fixture.py`、`mutants_extra.py`、`mutants_review.py`，`e2e_sweep1.py`、`e2e_code.py`、`e2e_actor.py`，spec 附录 A）；实现者可以照表抽查，改坏之后必须恢复。表中"（Task n 起）"标出的测试在较晚的 Task 才有：这一行的变异从那个 Task 起才被发现。**安全或加锁的性质只由单元一层发现的，算缺口**（brief 的缺陷类别）；表中每一条这类性质都另有存储、组合或端到端一层的测试，例外写在 spec 第 3 节。
- **评审敏感**（M3 设计 12 节约束 3）：结束的连带跨两个模块（`EndMemberships`、规则 2 的项目集合、邀请的一步、交错 1、4、5、6），恢复在账户行的 `FOR SHARE` 之下进行（约定六的唯一例外）；谁能移出、离开、恢复是安全性质，唯一管理员的两条规则也是。改动这些测试、锁、规则之前，先照"变异"表确认它在所说的性质去掉之后失败。
- **提交**：提交信息用英文，末尾加一行：`Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`
- 所有命令在仓库根目录下执行，除非步骤中另有说明。

## 文件结构

路径相对于仓库根目录。"生成"表示由命令生成并提交；一个文件由多个 Task 修改时，"Task"一列都列出。

| 文件 | 职责 | Task |
|---|---|---|
| `server/internal/modules/workspace/adapter/postgres/queries/members.sql`、`server/internal/modules/workspace/adapter/postgres/queries/invitations.sql`（修改） | `EndMember`、`HasOtherAdmin`、`DeletePendingInvitations`；`ReactivateMember`（Task 7） | 1、7 |
| `server/internal/modules/workspace/adapter/postgres/endings.go`、`server/internal/modules/workspace/adapter/postgres/endings_test.go` | 结束的三个存储方法和它们的测试（另一行、另一个成员、另一个工作区、已删除的行各在两个行序下一列不动） | 1 |
| `server/internal/modules/workspace/adapter/postgres/gen/members.sql.go`、`server/internal/modules/workspace/adapter/postgres/gen/invitations.sql.go`（生成） | | 1、7 |
| `server/internal/modules/project/adapter/postgres/queries/cascade.sql`、`server/internal/modules/project/adapter/postgres/cascade.go`（修改）；`server/internal/modules/project/adapter/postgres/end_test.go` | `LockActiveMemberProjects`、`SoleAdmin`、`EndMemberships`；锁的 id 顺序、等锁期间删除的项目 | 2 |
| `server/internal/modules/project/adapter/postgres/gen/cascade.sql.go`（生成） | | 2 |
| `server/internal/modules/project/app/cascade.go`、`server/internal/modules/project/app/cascade_test.go`、`server/internal/modules/project/app/ports.go`、`server/internal/modules/project/domain/errors.go`（修改） | `Cascade.EndMemberships`（在调用时列举、锁、查唯一管理员、结束）；端口 `MembershipEnder`；`ErrSoleAdmin`（409 `project.sole_admin`） | 2 |
| `server/internal/modules/project/module.go`（修改） | 连带的第三个方法；`ProjectMembershipCounts` 进 `Provide`（Task 7） | 2、7 |
| `server/internal/bootstrap/interleaving_growth_test.go`、`server/internal/bootstrap/interleaving_writes_test.go`（修改） | `NewCascade` 的第三个参数；交错 4 用到的 `growthRace` 的移出（Task 9）；替身的注释改指 P5b（Task 11） | 2、9、11 |
| `server/internal/modules/workspace/app/end_membership.go` | 移出、离开共用的结束一步 `membershipEnd` | 3 |
| `server/internal/modules/workspace/app/lock.go`、`server/internal/modules/workspace/app/update_member.go`（修改） | `lockedMember` 从改角色的用例提出，移出也用 | 3 |
| `server/internal/modules/workspace/app/remove_member.go`、`server/internal/modules/workspace/app/remove_member_test.go` | `removeWorkspaceMember` 的用例 | 3 |
| `server/internal/modules/workspace/app/ports.go`（修改） | `MemberLocker`、`MembershipEnder`、`MemberRemover`；`ProjectCascade.EndMemberships`；`WorkspaceLeaver`（Task 5）；`ProjectMembershipCounts`、`MemberReactivator`（Task 7） | 3、5、7 |
| `server/internal/modules/workspace/app/fakes_members_test.go`、`server/internal/modules/workspace/app/fakes_projects_test.go`、`server/internal/modules/workspace/app/fakes_workspaces_test.go`、`server/internal/modules/workspace/app/clock_test.go`（修改） | 假存储、假连带、假计数；每个写在锁之后读一次时钟 | 3、5、7（`fakes_projects_test.go`：3、7） |
| `server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go`（修改） | 假连带多一个方法 | 3 |
| `server/internal/modules/workspace/domain/actions.go`（修改）；`server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`（修改） | 操作名 `workspace_member.remove`、`workspace.leave`（Task 5）和它们的规则 | 3、5 |
| `api/modules/workspace.yaml`；`api/openapi.yaml`（修改） | 两个操作 | 4、5（`openapi.yaml`：5） |
| `api/dist/openapi.yaml`、`server/internal/modules/workspace/adapter/http/gen/server.gen.go`、`web/packages/api-client/src/schema.gen.ts`（生成） | | 4、5 |
| `server/internal/modules/workspace/adapter/http/handler.go`、`server/internal/modules/workspace/adapter/http/handler_test.go`、`server/internal/modules/workspace/adapter/http/members.go`、`server/internal/modules/workspace/adapter/http/members_test.go`、`server/internal/modules/workspace/module.go`（修改） | 两个 handler，它们答 `project.sole_admin`（9.4）；接线 | 4、5 |
| `web/apps/web/helpers/authentication.helper.ts`、`web/packages/i18n/src/locales/en/auth.json`、`web/packages/i18n/src/locales/zh-CN/auth.json`（修改） | `project.sole_admin`、`workspace.sole_admin`（Task 5）的文案 | 4、5 |
| `server/internal/bootstrap/permission_matrix_workspace_test.go`、`server/internal/bootstrap/permission_matrix_coverage_test.go`（修改） | `toMembership`；移出的三个变体、离开的两行 | 4、5 |
| `server/internal/bootstrap/removal_test.go`（Task 4 新建，Task 5 删除） | 组合出的移出（过渡版本） | 4、5 |
| `server/internal/modules/workspace/app/leave_workspace.go`、`server/internal/modules/workspace/app/leave_workspace_test.go`；`server/internal/modules/workspace/domain/errors.go`（修改） | `leaveWorkspace`；`ErrSoleAdmin`（409 `workspace.sole_admin`）；命令行的两个码（Task 7） | 5、7（`errors.go`） |
| `server/internal/bootstrap/ending_test.go` | 组合出的移出和离开：结束、不留邀请、一个时刻、写者；唯一管理员不能离开；`bystanders` 的前提 | 5、6 |
| `server/internal/bootstrap/permission_matrix_test.go`、`server/internal/bootstrap/permission_matrix_columns_test.go`、`server/internal/bootstrap/permission_matrix_targets_test.go`、`server/internal/bootstrap/permission_matrix_seed_test.go`（修改） | 唯一管理员的一张表（`soleAdminColumns`）和它的前提；被移出的成员改由存储写出（Task 11） | 5、11（`targets`：5） |
| `server/internal/bootstrap/project_write_locks_test.go`（修改） | 写在项目一级的完整性核对只数项目表的列，不把离开工作区的唯一管理员一行当作项目级的写 | 5 |
| `server/internal/bootstrap/ending_races_test.go`、`server/internal/bootstrap/ending_connection_test.go` | 等锁时目标、调用者、工作区被结束或删除（404，一行不改）；每把锁的强度和顺序；每条语句在事务的连接上 | 6 |
| `server/internal/bootstrap/project_connection_test.go`、`server/internal/bootstrap/interleaving_not_found_test.go`（修改） | `moduleRoute`：项目模块和工作区模块共用的一个只有一个连接的路由 | 6 |
| `server/internal/modules/project/adapter/postgres/queries/members.sql`、`server/internal/modules/project/adapter/postgres/members.go`、`server/internal/modules/project/adapter/postgres/members_test.go`（修改） | `CountInactive` | 7 |
| `server/internal/modules/project/adapter/postgres/gen/members.sql.go`（生成） | | 7 |
| `server/internal/modules/workspace/adapter/postgres/reactivation.go`、`server/internal/modules/workspace/adapter/postgres/reactivation_test.go` | `ReactivateMember` 的存储 | 7 |
| `server/internal/modules/workspace/app/reactivate_member.go`、`server/internal/modules/workspace/app/reactivate_member_test.go`；`server/internal/modules/workspace/app/create_workspace_test.go`（修改） | 恢复成员的用例；`logged` 共用 | 7 |
| `server/internal/modules/workspace/admin.go`、`server/internal/bootstrap/workspaces.go`、`server/cmd/nerve/workspaces.go`、`server/cmd/nerve/workspaces_test.go`（修改） | `nerve workspaces reactivate-member`：`NewAdmin`、组合、命令、输出的一行 | 8 |
| `server/internal/bootstrap/reactivation_test.go`、`server/internal/bootstrap/reactivation_races_test.go` | 命令在真实数据库上：恢复、已是有效、每个退出码 1 的情形数据库不变；等锁时的变化、锁的强度和顺序、中断、在事务的连接上 | 8 |
| `server/internal/bootstrap/interleaving_endings_test.go` | 交错 1 的工作区一侧、交错 4 | 9 |
| `server/internal/bootstrap/interleaving_removal_test.go` | 交错 5、6 | 10 |
| `server/internal/bootstrap/permission_matrix_members_test.go`、`server/internal/bootstrap/permission_matrix_project_test.go`、`server/internal/bootstrap/permission_matrix_seeded_test.go`、`server/internal/bootstrap/demotion_test.go`、`server/internal/bootstrap/project_members_test.go`、`server/internal/bootstrap/project_writes_test.go`（修改） | 被移出的成员不再在项目成员列表里；降级测试的离开经接口；替身的注释改指 P5b | 11 |
| `server/internal/bootstrap/invitation_rules_test.go` | S2 顺序、9d 顺序的集成测试 | 12 |
| `e2e/fixtures/api.ts`、`e2e/fixtures/assert/workspace.ts`、`e2e/stories/workspace/w3-workspace-settings.spec.ts`（修改） | `listMembers`、`membershipOf`；`expectMembershipEnded`、`expectWrittenLastBy`；`deletedAlone` 成为参数 | 13 |
| `e2e/stories/workspace/w2-landing.spec.ts`、`e2e/stories/workspace/w7-member-management.spec.ts` | W2、W7 的接口版本 | 13 |
| `e2e/stories/workspace/w12-reactivate-member.spec.ts` | W12 | 14 |
| `README.md`、`docs/v0/plane-diff.md`（修改） | 部署一节的"恢复被移出的成员"（8.7）；差异清单中 P5a 的三行（3.20） | 14 |

---

### Task 1: 工作区的存储：结束成员关系、删除待接受的邀请、其余的管理员

**Files:**
- Create: `server/internal/modules/workspace/adapter/postgres/endings.go`、`server/internal/modules/workspace/adapter/postgres/endings_test.go`
- Modify: `server/internal/modules/workspace/adapter/postgres/queries/invitations.sql`、`server/internal/modules/workspace/adapter/postgres/queries/members.sql`
- Generate: `server/internal/modules/workspace/adapter/postgres/gen/invitations.sql.go`、`server/internal/modules/workspace/adapter/postgres/gen/members.sql.go`

**Interfaces:**
- Produces（spec 2.3，M3 设计 3.6、3.7 规则 1、3.8、4.3）：`workspace/adapter/postgres.Store` 的三个方法，都在 `ctx` 带着的事务里执行，调用者持工作区行的 `FOR NO KEY UPDATE`：
  - `EndMember(ctx, workspaceID, userID, by uuid.UUID, now time.Time) error`：这个账户在这个工作区未删除的成员关系 `is_active = false`，`updated_at = now`、`updated_by_id = by`，行留着，角色不变。不是恰好一行时返回一个错误，不是 `app.ErrNotFound`（调用者在锁下读到它有效）。
  - `DeletePendingInvitations(ctx, workspaceID uuid.UUID, email string, by uuid.UUID, now time.Time) error`：这个工作区发给这个地址的待接受邀请（`responded_at IS NULL AND deleted_at IS NULL`）软删除，`deleted_at = updated_at = now`、`updated_by_id = by`；已拒绝的不动，已删除的留着自己的时刻。
  - `HasOtherAdmin(ctx, workspaceID, userID uuid.UUID) (bool, error)`：这个工作区除了他还有没有有效、未删除的管理员（`role = 20`，"管理员"这个集合）。
- 三条查询照原样写进 `members.sql`、`invitations.sql`（spec 2.3 有全文）。

**Tests:**（`adapter/postgres/endings_test.go`；存储测试的共同写法：被写的行除了写的列一列不动，其余每一行每一列不动，`tableRows` 比较前后）
- `TestEndMember`：bob 在 acme 的成员关系结束，由给的账户、在给的时刻；他已删除的成员关系（存在有效的之前或之后，两个行序）、他在 beta 的、carol 在 acme 的每一列不动，他被结束的那一行其余的列不动；bob 那一行之前由他自己写，所以"由 alice 写"可以不成立。没有未删除成员关系的一对（carol 在 beta、bob 在已删除的 gamma）返回不是 `ErrNotFound` 的错误，什么都不改。
- `TestDeletePendingInvitations`：acme 里发给 bob@corp.com 的待接受邀请（由 alice 写）被删除；他更早的两个（一个已接受、一个待接受时被删除，各在存入之前或之后）留着各自的时刻；carol 的待接受邀请、beta 里发给 bob 的待接受邀请、gamma 里他已拒绝的邀请一列不动；在 gamma 里结束不动那个已拒绝的。
- `TestHasOtherAdmin`：alice 是每个工作区的管理员，bob 按情形站在那里：只有那个工作区里有效、未删除的另一个管理员才算（别的工作区的管理员、他自己、成员、已结束的、已删除的都不算）。

- [ ] **Step 1: 查询**

`server/internal/modules/workspace/adapter/postgres/queries/members.sql`（修改，1 处）：

````old server/internal/modules/workspace/adapter/postgres/queries/members.sql
  AND m.is_active AND m.deleted_at IS NULL AND w.deleted_at IS NULL;

````
````new server/internal/modules/workspace/adapter/postgres/queries/members.sql
  AND m.is_active AND m.deleted_at IS NULL AND w.deleted_at IS NULL;

-- name: EndMember :execrows
-- removeWorkspaceMember and leaveWorkspace, under the workspace's FOR NO KEY UPDATE (M3 design 3.6): the user's
-- membership of the workspace ends, the row stays (4.3). The partial unique index holds at most one undeleted row per
-- pair, so a deleted one, which keeps its columns, is the only other row the pair can name.
UPDATE workspace_members
SET is_active = false, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(ended_by)::uuid
WHERE workspace_id = sqlc.arg(workspace_id) AND member_id = sqlc.arg(member_id) AND deleted_at IS NULL;

-- name: HasOtherAdmin :one
-- leaveWorkspace, under the workspace's FOR NO KEY UPDATE, which every change of an admin's membership takes too: whether
-- an active admin of the workspace other than the user is left (M3 design 3.7 rule 1).
SELECT EXISTS (SELECT 1 FROM workspace_members
               WHERE workspace_id = sqlc.arg(workspace_id) AND member_id <> sqlc.arg(member_id) AND role = 20 AND is_active
                 AND deleted_at IS NULL);

````

`server/internal/modules/workspace/adapter/postgres/queries/invitations.sql`（修改，1 处）：

````old server/internal/modules/workspace/adapter/postgres/queries/invitations.sql
WHERE workspace_id = sqlc.arg(workspace_id) AND deleted_at IS NULL;

````
````new server/internal/modules/workspace/adapter/postgres/queries/invitations.sql
WHERE workspace_id = sqlc.arg(workspace_id) AND deleted_at IS NULL;

-- name: DeletePendingInvitations :exec
-- An ended membership leaves no invitation (M3 design 3.8): removeWorkspaceMember and leaveWorkspace, under the
-- workspace's FOR NO KEY UPDATE and before the membership's row, soft-delete the workspace's pending invitation to the
-- address, if any; a declined one stays, and so does a deleted one's moment.
UPDATE workspace_member_invites
SET deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE workspace_id = sqlc.arg(workspace_id) AND email = sqlc.arg(email) AND responded_at IS NULL AND deleted_at IS NULL;

````

Run: `make gen-go`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `3c3361eb2cd4e6285d9e8ec659af19471b67adf78543a72c36195a54713c8b98` | 325 | `server/internal/modules/workspace/adapter/postgres/gen/invitations.sql.go` |
| `a79d38a335620b0c04631a06c18efeb230386525a081b76eb87ed7f1c40c0ad8` | 308 | `server/internal/modules/workspace/adapter/postgres/gen/members.sql.go` |

Run: `shasum -a 256 server/internal/modules/workspace/adapter/postgres/gen/invitations.sql.go server/internal/modules/workspace/adapter/postgres/gen/members.sql.go`
Expected: 与上表相同。

- [ ] **Step 2: 存储和测试**

`server/internal/modules/workspace/adapter/postgres/endings.go`（新文件，51 行）：

````file server/internal/modules/workspace/adapter/postgres/endings.go
package postgresadapter

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres/gen"
)

// The statements of an ended membership (M3 design 3.6, 3.7, 3.8):
// removeWorkspaceMember's and leaveWorkspace's, each under the workspace's
// FOR NO KEY UPDATE, in the transaction ctx carries.

// HasOtherAdmin reports whether the workspace has an active admin other
// than userID.
func (s *Store) HasOtherAdmin(ctx context.Context, workspaceID, userID uuid.UUID) (bool, error) {
	other, err := s.queries(ctx).HasOtherAdmin(ctx, gen.HasOtherAdminParams{WorkspaceID: workspaceID, MemberID: userID})
	if err != nil {
		return false, fmt.Errorf("look for another admin: %w", err)
	}
	return other, nil
}

// DeletePendingInvitations soft-deletes the workspace's pending invitation
// to email, if there is one, by the account by at now; a declined one
// stays.
func (s *Store) DeletePendingInvitations(ctx context.Context, workspaceID uuid.UUID, email string, by uuid.UUID, now time.Time) error {
	err := s.queries(ctx).DeletePendingInvitations(ctx, gen.DeletePendingInvitationsParams{WorkspaceID: workspaceID, Email: email, DeletedBy: by,
		Now: now})
	if err != nil {
		return fmt.Errorf("delete the pending invitations: %w", err)
	}
	return nil
}

// EndMember ends userID's membership of the workspace, by the account by at
// now; the row stays. The caller read the membership active under the
// workspace's lock: a pair without exactly one undeleted membership is an
// error, not app.ErrNotFound.
func (s *Store) EndMember(ctx context.Context, workspaceID, userID, by uuid.UUID, now time.Time) error {
	ended, err := s.queries(ctx).EndMember(ctx, gen.EndMemberParams{WorkspaceID: workspaceID, MemberID: userID, EndedBy: by, Now: now})
	switch {
	case err != nil:
		return fmt.Errorf("end workspace member: %w", err)
	case ended != 1:
		return fmt.Errorf("end workspace member %s of %s: %d undeleted memberships", userID, workspaceID, ended)
	}
	return nil
}
````

`server/internal/modules/workspace/adapter/postgres/endings_test.go`（新文件，215 行）：

````file server/internal/modules/workspace/adapter/postgres/endings_test.go
package postgresadapter_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// tableRows is every row of table as JSON, one a line by id: a row whose id
// is in written without the columns cols, which the write under test sets.
// Compared before and after a write, it shows that the write changed those
// columns of those rows alone; the caller checks the columns themselves.
func tableRows(t *testing.T, pool *pgxpool.Pool, table string, written []uuid.UUID, cols ...string) string {
	t.Helper()
	var rows string
	if err := pool.QueryRow(context.Background(), `SELECT coalesce(string_agg(CASE WHEN r.id = ANY ($1) THEN (to_jsonb(r) - $2::text[])::text
		ELSE to_jsonb(r)::text END, E'\n' ORDER BY r.id), '') FROM `+table+` r`, written, cols).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	return rows
}

// stamp is what an ending writes of a row, as text: whether a membership
// (workspace_members) is active, or when an invitation
// (workspace_member_invites) was deleted, then its updated_at and
// updated_by_id.
func stamp(t *testing.T, pool *pgxpool.Pool, table string, id uuid.UUID) string {
	t.Helper()
	state := "CASE WHEN is_active THEN 'active' ELSE 'ended' END"
	if table == "workspace_member_invites" {
		state = "coalesce('deleted at ' || deleted_at::text, 'undeleted')"
	}
	var got string
	if err := pool.QueryRow(context.Background(), "SELECT "+state+" || ' at ' || updated_at::text || ' by ' || updated_by_id::text FROM "+
		table+" WHERE id = $1", id).Scan(&got); err != nil {
		t.Fatal(err)
	}
	return got
}

// at is tm as PostgreSQL shows a timestamptz, as stamp does.
func at(t *testing.T, pool *pgxpool.Pool, tm time.Time) string {
	t.Helper()
	var s string
	if err := pool.QueryRow(context.Background(), "SELECT $1::timestamptz::text", tm).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

// EndMember ends the user's undeleted membership of the workspace, by the
// account and at the time given, and changes nothing else (M3 design 3.6,
// 4.3). bob's membership of acme was last written by himself. His deleted
// membership of acme, stored before his live one or after it, his
// membership of beta and carol's of acme keep every column, and his ended
// one its other columns. A pair with no undeleted membership is an error
// that is not app.ErrNotFound, and changes nothing: carol's in beta, of
// which there is none, and bob's in gamma, deleted.
func TestEndMember(t *testing.T) {
	for _, deletedFirst := range []bool{true, false} {
		t.Run(fmt.Sprintf("the deleted membership stored first %v", deletedFirst), func(t *testing.T) {
			s, pool := newStore(t)
			alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
			acme, beta, gamma := newWorkspace(t, s, "Acme", "acme", alice), newWorkspace(t, s, "Beta", "beta", alice),
				newWorkspace(t, s, "Gamma", "gamma", alice)
			deleted := func(workspace uuid.UUID) {
				exec(t, pool, `INSERT INTO workspace_members (id, workspace_id, member_id, role, created_by_id, updated_by_id, created_at,
					updated_at, deleted_at) VALUES ($1, $2, $3, 20, $3, $3, $4, $4, $4)`, uuid.NewV7(), workspace, bob, now.Add(-time.Hour))
			}
			if deletedFirst {
				deleted(acme.ID)
			}
			bobIn := joinAt(t, s, acme.ID, bob, shared.RoleMember, now)
			if !deletedFirst {
				deleted(acme.ID)
			}
			joinAt(t, s, acme.ID, carol, shared.RoleGuest, now)
			joinAt(t, s, beta.ID, bob, shared.RoleAdmin, now)
			deleted(gamma.ID)
			if got, want := stamp(t, pool, "workspace_members", bobIn.ID), fmt.Sprintf("active at %s by %s", at(t, pool, now), bob); got != want {
				t.Fatalf("bob's membership of acme before: %s, want %s", got, want)
			}
			before := tableRows(t, pool, "workspace_members", []uuid.UUID{bobIn.ID}, "is_active", "updated_at", "updated_by_id")
			later := now.Add(time.Hour)

			if err := s.EndMember(context.Background(), acme.ID, bob, alice, later); err != nil {
				t.Fatal(err)
			}

			if got, want := stamp(t, pool, "workspace_members", bobIn.ID), fmt.Sprintf("ended at %s by %s", at(t, pool, later), alice); got != want {
				t.Errorf("bob's membership of acme: %s, want %s", got, want)
			}
			if after := tableRows(t, pool, "workspace_members", []uuid.UUID{bobIn.ID}, "is_active", "updated_at", "updated_by_id"); after != before {
				t.Errorf("the memberships, bob's in acme without the columns written:\n%s\nwant them as they were:\n%s", after, before)
			}
			before = tableRows(t, pool, "workspace_members", nil)
			for _, pair := range []struct {
				name            string
				workspace, user uuid.UUID
			}{{"carol in beta", beta.ID, carol}, {"bob in gamma", gamma.ID, bob}} {
				if err := s.EndMember(context.Background(), pair.workspace, pair.user, alice, later.Add(time.Hour)); err == nil ||
					errors.Is(err, app.ErrNotFound) {
					t.Errorf("EndMember() of %s = %v, want an error that is not app.ErrNotFound", pair.name, err)
				}
			}
			if after := tableRows(t, pool, "workspace_members", nil); after != before {
				t.Errorf("the memberships after ending none:\n%s\nwant them as they were:\n%s", after, before)
			}
		})
	}
}

// DeletePendingInvitations soft-deletes the workspace's pending invitation
// to the address, by the account and at the time given, and changes nothing
// else (M3 design 3.8). In acme bob@corp.com has a pending invitation,
// written by alice, and two earlier ones, one accepted, one deleted while
// pending, each deleted then, stored before it or after; carol@corp.com has
// a pending one; beta has bob's pending one, gamma his declined one. Ending
// in acme deletes his pending one there; ending in gamma leaves the
// declined one; every other row keeps every column, and the earlier ones
// their moments.
func TestDeletePendingInvitations(t *testing.T) {
	for _, earlierFirst := range []bool{true, false} {
		t.Run(fmt.Sprintf("the earlier invitation stored first %v", earlierFirst), func(t *testing.T) {
			s, pool := newStore(t)
			alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
			acme, beta, gamma := newWorkspace(t, s, "Acme", "acme", alice), newWorkspace(t, s, "Beta", "beta", alice),
				newWorkspace(t, s, "Gamma", "gamma", alice)
			earlier := func() {
				exec(t, pool, `INSERT INTO workspace_member_invites (id, workspace_id, email, role, accepted, responded_at, created_by_id,
					updated_by_id, created_at, updated_at, deleted_at) VALUES ($1, $2, 'bob@corp.com', 15, true, $4, $3, $3, $4, $4, $4)`,
					uuid.NewV7(), acme.ID, alice, now.Add(-time.Hour))
				exec(t, pool, `INSERT INTO workspace_member_invites (id, workspace_id, email, role, created_by_id, updated_by_id, created_at,
					updated_at, deleted_at) VALUES ($1, $2, 'bob@corp.com', 5, $3, $3, $4, $4, $4)`, uuid.NewV7(), acme.ID, alice, now.Add(-time.Hour))
			}
			if earlierFirst {
				earlier()
			}
			pending := invite(t, s, acme.ID, "bob@corp.com", shared.RoleGuest, alice)
			if !earlierFirst {
				earlier()
			}
			invite(t, s, acme.ID, "carol@corp.com", shared.RoleMember, alice)
			invite(t, s, beta.ID, "bob@corp.com", shared.RoleMember, alice)
			declined := invite(t, s, gamma.ID, "bob@corp.com", shared.RoleMember, alice)
			exec(t, pool, "UPDATE workspace_member_invites SET responded_at = $2 WHERE id = $1", declined.ID, now)
			if got, want := stamp(t, pool, "workspace_member_invites", pending.ID), fmt.Sprintf("undeleted at %s by %s", at(t, pool, now), alice); got != want {
				t.Fatalf("bob's pending invitation to acme before: %s, want %s", got, want)
			}
			before := tableRows(t, pool, "workspace_member_invites", []uuid.UUID{pending.ID}, "deleted_at", "updated_at", "updated_by_id")
			later := now.Add(time.Hour)

			for _, w := range []uuid.UUID{acme.ID, gamma.ID} {
				if err := s.DeletePendingInvitations(context.Background(), w, "bob@corp.com", bob, later); err != nil {
					t.Fatal(err)
				}
			}

			if got, want := stamp(t, pool, "workspace_member_invites", pending.ID), fmt.Sprintf("deleted at %[1]s at %[1]s by %[2]s",
				at(t, pool, later), bob); got != want {
				t.Errorf("bob's pending invitation to acme: %s, want %s", got, want)
			}
			after := tableRows(t, pool, "workspace_member_invites", []uuid.UUID{pending.ID}, "deleted_at", "updated_at", "updated_by_id")
			if after != before {
				t.Errorf("the invitations, bob's pending one to acme without the columns written:\n%s\nwant them as they were:\n%s", after, before)
			}
		})
	}
}

// HasOtherAdmin reports whether the workspace has an active admin other
// than the user (M3 design 3.7 rule 1): alice is the admin of each
// workspace; bob, in each case, stands to it as the case says. Only an
// active, undeleted admin's membership of that workspace is another admin.
func TestHasOtherAdmin(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	set := func(sql string) func(uuid.UUID) {
		return func(w uuid.UUID) { exec(t, pool, sql, w, bob) }
	}
	tests := []struct {
		name string
		bob  func(workspace uuid.UUID)
		want bool
	}{
		{"an active admin", func(w uuid.UUID) { join(t, s, w, bob, shared.RoleAdmin) }, true},
		{"an active member", func(w uuid.UUID) { join(t, s, w, bob, shared.RoleMember) }, false},
		{"an admin whose membership ended", func(w uuid.UUID) {
			join(t, s, w, bob, shared.RoleAdmin)
			set("UPDATE workspace_members SET is_active = false WHERE workspace_id = $1 AND member_id = $2")(w)
		}, false},
		{"an admin whose membership is deleted", func(w uuid.UUID) {
			join(t, s, w, bob, shared.RoleAdmin)
			set("UPDATE workspace_members SET deleted_at = now() WHERE workspace_id = $1 AND member_id = $2")(w)
		}, false},
		{"an admin of another workspace", func(uuid.UUID) { join(t, s, newWorkspace(t, s, "Other", "other", alice).ID, bob, shared.RoleAdmin) },
			false},
		{"none: alice is alone", func(uuid.UUID) {}, false},
	}
	for i, tt := range tests {
		w := newWorkspace(t, s, "Acme", fmt.Sprintf("acme-%d", i), alice)
		tt.bob(w.ID)
		if got, err := s.HasOtherAdmin(context.Background(), w.ID, alice); err != nil || got != tt.want {
			t.Errorf("bob %s: HasOtherAdmin() = %v, %v; want %v", tt.name, got, err, tt.want)
		}
	}
}
````

- [ ] **Step 3: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/workspace/...`
Expected: 全部 `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 4: 提交**

```bash
git add server/internal/modules/workspace/adapter/postgres/endings.go server/internal/modules/workspace/adapter/postgres/endings_test.go server/internal/modules/workspace/adapter/postgres/queries/invitations.sql server/internal/modules/workspace/adapter/postgres/queries/members.sql server/internal/modules/workspace/adapter/postgres/gen/invitations.sql.go server/internal/modules/workspace/adapter/postgres/gen/members.sql.go
```
```bash
git commit -m "feat(M3/P5a): the workspace store ends a membership, deletes the pending invitations to an address, and asks for another admin

EndMember ends a user's undeleted membership of a workspace, the row
kept; DeletePendingInvitations soft-deletes the workspace's pending
invitation to an address, a declined one left; HasOtherAdmin asks
whether an active admin other than the user is left. Each runs under
the workspace's FOR NO KEY UPDATE, and its store test checks every
column of every other row.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A，清扫 1、3；故事一列是单独运行的故事）：

| 改坏 | 必须失败的测试 | 层 |
|---|---|---|
| `EndMember` 去掉工作区；去掉成员 | `TestEndMember`；W12（去掉工作区），W2、W7、W12（去掉成员）（Task 13、14 起） | 存储；端到端 |
| `EndMember` 去掉 `deleted_at IS NULL` | `TestEndMember`（故事看不到：工作区成员关系不单独删除，只随工作区删除，4.3） | 存储 |
| `EndMember` 也改角色；不写结束者 | `TestEndMember`；W2、W7、W12 的写者（不写结束者，Task 13、14 起） | 存储；端到端 |
| `DeletePendingInvitations` 去掉工作区、地址 | `TestDeletePendingInvitations`；W7、W12 | 存储；端到端 |
| `DeletePendingInvitations` 去掉 `responded_at IS NULL`；去掉 `deleted_at IS NULL`；也删已接受的 | `TestDeletePendingInvitations`；W12（已拒绝的），W7（已删除的） | 存储；端到端 |
| `HasOtherAdmin` 去掉工作区、成员、角色、有效 | `TestHasOtherAdmin`；W7（工作区），W2、W7（角色），W2、W7、W12（成员），W12（有效） | 存储；端到端 |
| `HasOtherAdmin` 去掉 `deleted_at IS NULL` | `TestHasOtherAdmin`（故事看不到，同 `EndMember`） | 存储 |
| 三个方法经连接池、在事务之外执行 | `TestTheEndingsRunOnTheirTransactionsConnection`（Task 6 起） | 组合 |

**Done when:** 三条查询生成；三个存储测试通过，每个谓词都有一个只由它决定的行。

---

### Task 2: `ProjectCascade.EndMemberships`

**Files:**
- Create: `server/internal/modules/project/adapter/postgres/end_test.go`
- Modify: `server/internal/bootstrap/interleaving_growth_test.go`、`server/internal/bootstrap/interleaving_writes_test.go`、`server/internal/modules/project/adapter/postgres/cascade.go`、`server/internal/modules/project/adapter/postgres/queries/cascade.sql`、`server/internal/modules/project/app/cascade.go`、`server/internal/modules/project/app/cascade_test.go`、`server/internal/modules/project/app/ports.go`、`server/internal/modules/project/domain/errors.go`、`server/internal/modules/project/module.go`
- Generate: `server/internal/modules/project/adapter/postgres/gen/cascade.sql.go`

**Interfaces:**
- Produces（spec 2.4，M3 设计 3.6 约定五、六，3.7 规则 2，6.5）：
  - `project/app.MembershipEnder`（`LockActiveMemberProjects(ctx, workspaceIDs, userID) ([]uuid.UUID, error)`、`SoleAdmin(ctx, projectIDs, userID) (bool, error)`、`EndMemberships(ctx, projectIDs, userID, by, now) error`）；`NewCascade(projects ProjectsDeleter, members MemberDemoter, enders MembershipEnder)`。
  - `(*Cascade).EndMemberships(ctx, workspaceIDs []uuid.UUID, userID, by uuid.UUID, now time.Time) error`：在调用时找出这些工作区里他有有效成员关系的未删除项目（已归档的也算），按 id 升序以 `FOR NO KEY UPDATE` 锁住；在锁下问他是不是其中某个"还有别的有效成员"的项目唯一的有效管理员：是则 `domain.ErrSoleAdmin`（409 `project.sole_admin`），什么都不写；否则一条语句结束他在这些项目的有效成员关系，在 `now`、由 `by`。一个项目都没有时不问、不写。`project.Cascade` 接口多这个方法（`module.go`）。
  - 存储的三条查询照原样写进 `cascade.sql`（spec 2.4 有全文）。
- 接线：`project.New` 的 `NewCascade(store, store, store)`；交错测试里自建的连带同样多一个参数。

**Tests:**
- `adapter/postgres/end_test.go`：`TestEndingAMembersProjectMemberships`（锁按 id 顺序返回 acme 和 gamma 里他有有效成员关系的未删除项目，已归档的也在；事务结束之前每个都持 `FOR NO KEY UPDATE`：`FOR SHARE` 等它，外键的 `FOR KEY SHARE` 不等；别的项目都不锁：他的成员关系已结束的 Docs，他的已删除而 alice 的有效的 HR，只有 alice 的 Free，已删除的 Gone，没有问到的 beta 的 Web。写结束他在这些项目的成员关系，各留角色，其余列不变；他在 Web 已删除的成员关系（两个行序）、alice 的、他已结束的、他在 Gone 和 beta 的一列不动；问到空的项目集合时什么都不写）；`TestSoleAdmin`（每个情形是自己的一个项目，在同一个工作区里，别的情形的项目各有管理员和成员，查错了项目的成员或管理员就答错：Plane 把这个集合查错过三种：工作区成员关系的 id 比项目成员的账户，只看一个成员的项目，只看他一个人的项目；含"他是成员、旁边还有一个成员、没有管理员"的一例）；`TestLockActiveMemberProjectsLocksInIDOrder`（Web 的 id 较小，但在表里、在名字和标识的索引里都排在 Alpha 之后；Alpha 被别的事务持有时，`LockActiveMemberProjects` 持着 Web 等它，`FOR SHARE` 于是等 Web；别的顺序会先到 Alpha、什么都不持地等）；`TestLockActiveMemberProjectsLeavesOutAProjectDeletedWhileItWaited`（等锁期间被删除的项目不在结果里）。
- `app/cascade_test.go`：`TestEndMemberships`（锁 → 查唯一管理员 → 结束，参数原样传下，锁返回的项目照原样往下传，顺序不是任何一种排序；唯一管理员时 `project.sole_admin`，不写；一个都没锁到时不问、不写；每个调用的失败原样返回，之后的不调用；全部在调用者的事务里）。`TestDeleteWorkspaceProjects`、`TestDemoteToGuest` 只多一个构造参数。

- [ ] **Step 1: 查询**

`server/internal/modules/project/adapter/postgres/queries/cascade.sql`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/queries/cascade.sql
  AND role <> 5;

````
````new server/internal/modules/project/adapter/postgres/queries/cascade.sql
  AND role <> 5;

-- name: LockActiveMemberProjects :many
-- The first step of ending an account's project memberships, EndMemberships' (M3 design 3.6 convention 6): run when
-- it is called, after the caller ended his membership of the workspaces, it finds their undeleted projects, archived
-- ones too, in which he has an active membership, and locks them FOR NO KEY UPDATE in id order. The lock is taken as
-- the sorted rows come, so the order is the ids'. After a wait, Postgres evaluates deleted_at IS NULL again on the
-- row's newest version: a project deleted meanwhile is left out.
SELECT p.id
FROM projects p
WHERE p.workspace_id = ANY (sqlc.arg(workspace_ids)::uuid[]) AND p.deleted_at IS NULL
  AND EXISTS (SELECT 1 FROM project_members m
              WHERE m.project_id = p.id AND m.member_id = sqlc.arg(member_id) AND m.is_active AND m.deleted_at IS NULL)
ORDER BY p.id
FOR NO KEY UPDATE;

-- name: SoleAdmin :one
-- The second step, under the projects' locks: whether the account is the only active admin of one of the projects
-- that has another active member, whom ending his membership would leave without an admin (M3 design 3.7 rule 2). A
-- project where he is alone, or that has another active admin, does not count.
SELECT EXISTS (
    SELECT 1 FROM project_members m
    WHERE m.project_id = ANY (sqlc.arg(project_ids)::uuid[]) AND m.member_id = sqlc.arg(member_id) AND m.role = 20
      AND m.is_active AND m.deleted_at IS NULL
      AND NOT EXISTS (SELECT 1 FROM project_members a
                      WHERE a.project_id = m.project_id AND a.member_id <> m.member_id AND a.role = 20 AND a.is_active
                        AND a.deleted_at IS NULL)
      AND EXISTS (SELECT 1 FROM project_members o
                  WHERE o.project_id = m.project_id AND o.member_id <> m.member_id AND o.is_active AND o.deleted_at IS NULL));

-- name: EndMemberships :exec
-- The last step, one statement under the projects' locks (convention 5): the account's active memberships of the
-- projects end, at the moment and by the account given; the rows stay, and an ended or deleted one keeps its columns.
UPDATE project_members
SET is_active = false, updated_at = sqlc.arg(now)::timestamptz, updated_by_id = sqlc.arg(ended_by)::uuid
WHERE project_id = ANY (sqlc.arg(project_ids)::uuid[]) AND member_id = sqlc.arg(member_id) AND is_active AND deleted_at IS NULL;

````

Run: `make gen-go`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `c5e2e9f27d9bb4988d8ddcf5dd163ea4f387ee38ae83c8eb5b7b4eac5dadee25` | 272 | `server/internal/modules/project/adapter/postgres/gen/cascade.sql.go` |

Run: `shasum -a 256 server/internal/modules/project/adapter/postgres/gen/cascade.sql.go`
Expected: 与上表相同。

- [ ] **Step 2: 领域的码、端口、存储**

`server/internal/modules/project/domain/errors.go`（修改，1 处）：

````old server/internal/modules/project/domain/errors.go
	ErrArchived = shared.NewError(shared.KindConflict, "project.archived", "The project is archived; unarchive it to change it.")
````
````new server/internal/modules/project/domain/errors.go
	ErrArchived = shared.NewError(shared.KindConflict, "project.archived", "The project is archived; unarchive it to change it.")
	// ErrSoleAdmin answers an ending of an account's project memberships
	// that would leave a project with other active members without an
	// active admin: he is its only one (M3 design 3.7 rule 2). The
	// workspace's removal and leaving declare it too (M3 design 5.1).
	ErrSoleAdmin = shared.NewError(shared.KindConflict, "project.sole_admin",
		"Ending the membership would leave a project that has other members without an admin; make another of its members an admin first.")
````

`server/internal/modules/project/app/ports.go`（修改，1 处）：

````old server/internal/modules/project/app/ports.go
}

// Deletion is what a deletion of projects deletes, and when and by whom:
````
````new server/internal/modules/project/app/ports.go
}

// MembershipEnder ends an account's memberships of workspaces' projects (M3
// design 3.6 convention 6, 3.7 rule 2): his projects locked first, then
// the check of their admins, then his memberships of them in one
// statement. Each method runs in the transaction ctx carries.
type MembershipEnder interface {
	// LockActiveMemberProjects locks FOR NO KEY UPDATE, in id order, the
	// undeleted projects of workspaceIDs in which userID has an active
	// membership, and returns their ids.
	LockActiveMemberProjects(ctx context.Context, workspaceIDs []uuid.UUID, userID uuid.UUID) ([]uuid.UUID, error)
	// SoleAdmin reports whether userID is the only active admin of one of
	// projectIDs that has another active member.
	SoleAdmin(ctx context.Context, projectIDs []uuid.UUID, userID uuid.UUID) (bool, error)
	// EndMemberships sets is_active false, updated_at now and updated_by_id
	// by on userID's active, undeleted memberships of projectIDs.
	EndMemberships(ctx context.Context, projectIDs []uuid.UUID, userID, by uuid.UUID, now time.Time) error
}

// Deletion is what a deletion of projects deletes, and when and by whom:
````

`server/internal/modules/project/adapter/postgres/cascade.go`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/cascade.go
		return fmt.Errorf("demote the member's project memberships: %w", err)
	}
	return nil
}

````
````new server/internal/modules/project/adapter/postgres/cascade.go
		return fmt.Errorf("demote the member's project memberships: %w", err)
	}
	return nil
}

// LockActiveMemberProjects locks FOR NO KEY UPDATE, in id order, the
// workspaces' undeleted projects in which userID has an active membership,
// and returns their ids (app.MembershipEnder).
func (s *Store) LockActiveMemberProjects(ctx context.Context, workspaceIDs []uuid.UUID, userID uuid.UUID) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).LockActiveMemberProjects(ctx, gen.LockActiveMemberProjectsParams{WorkspaceIds: workspaceIDs, MemberID: userID})
	if err != nil {
		return nil, fmt.Errorf("lock the member's active projects: %w", err)
	}
	return ids, nil
}

// SoleAdmin reports whether userID is the only active admin of one of
// projectIDs that has another active member (app.MembershipEnder).
func (s *Store) SoleAdmin(ctx context.Context, projectIDs []uuid.UUID, userID uuid.UUID) (bool, error) {
	sole, err := s.queries(ctx).SoleAdmin(ctx, gen.SoleAdminParams{ProjectIds: projectIDs, MemberID: userID})
	if err != nil {
		return false, fmt.Errorf("look for a project he is the only admin of: %w", err)
	}
	return sole, nil
}

// EndMemberships ends userID's active memberships of projectIDs, at now, by
// the account by (app.MembershipEnder).
func (s *Store) EndMemberships(ctx context.Context, projectIDs []uuid.UUID, userID, by uuid.UUID, now time.Time) error {
	err := s.queries(ctx).EndMemberships(ctx, gen.EndMembershipsParams{ProjectIds: projectIDs, MemberID: userID, EndedBy: by, Now: now})
	if err != nil {
		return fmt.Errorf("end the member's project memberships: %w", err)
	}
	return nil
}

````

`server/internal/modules/project/adapter/postgres/end_test.go`（新文件，303 行）：

````file server/internal/modules/project/adapter/postgres/end_test.go
package postgresadapter_test

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// endedRows is every membership as text, by id: one of ended without
// is_active, updated_at and updated_by_id, which ending it writes.
func endedRows(t *testing.T, pool *pgxpool.Pool, ended []uuid.UUID) string {
	t.Helper()
	var rows string
	if err := pool.QueryRow(context.Background(), `SELECT string_agg(CASE WHEN r.id = ANY ($1)
		THEN (to_jsonb(r) - 'is_active' - 'updated_at' - 'updated_by_id')::text ELSE r::text END, E'\n' ORDER BY r.id)
		FROM project_members r`, ended).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	return rows
}

// EndMemberships' first and last steps in one transaction (M3 design 3.6
// convention 6). The lock returns, in id order, the undeleted projects of
// acme and gamma, the archived one too, in which bob has an active
// membership; until the transaction ends each of them is held FOR NO KEY
// UPDATE, which a FOR SHARE waits for and a foreign key's FOR KEY SHARE
// does not, and no other project is: not Docs, where his membership ended;
// not HR, where his is deleted and alice's active; not Free, where only
// alice is; not Gone, deleted, though his membership of it is not; not
// beta's Web, beta not asked for. The write ends his memberships of those,
// at the moment and by the account given, each keeping its role, and leaves
// their other columns as they were; his deleted membership of Web, stored
// before his live one or after it, alice's, his ended one and his ones in
// Gone and beta keep every column. Asked to end his ended membership of
// Docs, it writes nothing.
func TestEndingAMembersProjectMemberships(t *testing.T) {
	for _, deletedFirst := range []bool{true, false} {
		t.Run(fmt.Sprintf("his deleted membership of Web stored first %v", deletedFirst), func(t *testing.T) {
			s, pool := newStore(t)
			alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
			acme, beta, gamma := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta"), newWorkspace(t, pool, "gamma")
			web, ops := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Ops", "OPS", alice)
			arch, docs := newProject(t, s, acme, "Arch", "ARCH", alice), newProject(t, s, acme, "Docs", "DOCS", alice)
			hr, free := newProject(t, s, acme, "HR", "HR", alice), newProject(t, s, acme, "Free", "FREE", alice)
			gone, site := newProject(t, s, acme, "Gone", "GONE", alice), newProject(t, s, gamma, "Site", "SITE", alice)
			betas := newProject(t, s, beta, "Web", "WEB", alice)
			exec(t, pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", arch, now)
			exec(t, pool, "UPDATE projects SET deleted_at = $2 WHERE id = $1", gone, now)
			// deleted stores bob's deleted membership of project, active as
			// a deletion leaves it, beside a live one or none.
			deleted := func(project uuid.UUID) {
				exec(t, pool, `INSERT INTO project_members (id, workspace_id, project_id, member_id, role, created_by_id, updated_by_id,
					created_at, updated_at, deleted_at) VALUES ($1, $2, $3, $4, 20, $4, $4, $5, $5, $5)`, uuid.NewV7(), acme, project, bob, now)
			}
			if deletedFirst {
				deleted(web)
			}
			ended := []uuid.UUID{seedMember(t, pool, acme, web, bob, 20, true)}
			if !deletedFirst {
				deleted(web)
			}
			ended = append(ended, seedMember(t, pool, acme, ops, bob, 15, true), seedMember(t, pool, acme, arch, bob, 5, true),
				seedMember(t, pool, gamma, site, bob, 15, true))
			deleted(hr)
			docsMembership := seedMember(t, pool, acme, docs, bob, 15, false)
			for _, p := range []uuid.UUID{web, hr, free} {
				seedMember(t, pool, acme, p, alice, 20, true)
			}
			seedMember(t, pool, acme, gone, bob, 20, true)
			seedMember(t, pool, beta, betas, bob, 20, true)
			before := endedRows(t, pool, ended)
			later := now.Add(time.Hour)

			var locked []uuid.UUID
			err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
				var err error
				if locked, err = s.LockActiveMemberProjects(ctx, []uuid.UUID{acme, gamma}, bob); err != nil {
					return err
				}
				for name, p := range map[string]struct {
					id   uuid.UUID
					held bool
				}{"Web": {web, true}, "Ops": {ops, true}, "Arch": {arch, true}, "gamma's Site": {site, true}, "Docs": {docs, false},
					"HR": {hr, false}, "Free": {free, false}, "Gone": {gone, false}, "beta's Web": {betas, false}} {
					if share, keyShare := waits(t, pool, p.id, "FOR SHARE"), waits(t, pool, p.id, "FOR KEY SHARE"); share != p.held || keyShare {
						t.Errorf("%s while locked: a FOR SHARE waits %v, a FOR KEY SHARE waits %v; want %v, false", name, share, keyShare, p.held)
					}
				}
				return s.EndMemberships(ctx, locked, bob, alice, later)
			})
			if err != nil {
				t.Fatal(err)
			}

			if want := slices.SortedFunc(slices.Values([]uuid.UUID{web, ops, arch, site}), uuid.UUID.Compare); !slices.Equal(locked, want) {
				t.Errorf("LockActiveMemberProjects() = %v, want %v", locked, want)
			}
			for i, id := range ended {
				var role int
				var active bool
				var updatedAt time.Time
				var updatedBy uuid.UUID
				if err := pool.QueryRow(context.Background(), "SELECT role, is_active, updated_at, updated_by_id FROM project_members WHERE id = $1",
					id).Scan(&role, &active, &updatedAt, &updatedBy); err != nil {
					t.Fatal(err)
				}
				if want := []int{20, 15, 5, 15}[i]; role != want || active || !updatedAt.Equal(later) || updatedBy != alice {
					t.Errorf("membership %s: role %d, active %v, at %v by %s; want %d, ended, at %v by alice", id, role, active, updatedAt, updatedBy,
						want, later)
				}
			}
			if after := endedRows(t, pool, ended); after != before {
				t.Errorf("the memberships, the ended ones without is_active, updated_at and updated_by_id:\n%s\nwant them as they were:\n%s", after,
					before)
			}
			before = endedRows(t, pool, nil)
			if err := s.EndMemberships(context.Background(), []uuid.UUID{docs}, bob, alice, later.Add(time.Hour)); err != nil {
				t.Fatal(err)
			}
			if after := endedRows(t, pool, nil); after != before {
				t.Errorf("ending his ended membership of Docs (%s) wrote:\n%s\nwant the memberships as they were:\n%s", docsMembership, after, before)
			}
		})
	}
}

// SoleAdmin reports whether bob is the only active admin of one of the
// projects asked about that has another active member (M3 design 3.7 rule
// 2): each case is a project of its own, asked about alone but where it
// says, in one workspace where the other cases' projects have admins and
// members, so that a check of another project's members or admins answers
// otherwise. Plane's checks got this set wrong: the workspace membership's
// id compared with a project member's account, the projects of one member
// only, those where he is alone.
func TestSoleAdmin(t *testing.T) {
	s, pool := newStore(t)
	alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	acme := newWorkspace(t, pool, "acme")
	type member struct {
		user   uuid.UUID
		role   int
		active bool
	}
	// project stores a project of acme with members, the last of them
	// deleted when deleteLast is set, and returns its id.
	n := 0
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
	alone := project(false, member{bob, 20, true})
	withAMember := project(false, member{bob, 20, true}, member{carol, 15, true})
	tests := []struct {
		name     string
		projects []uuid.UUID
		want     bool
	}{
		{"the only admin, with an active member", []uuid.UUID{withAMember}, true},
		{"the only admin, with an active guest", []uuid.UUID{project(false, member{bob, 20, true}, member{carol, 5, true})}, true},
		{"the only admin, alone", []uuid.UUID{alone}, false},
		{"the only admin, the other's membership ended", []uuid.UUID{project(false, member{bob, 20, true}, member{carol, 15, false})}, false},
		{"the only admin, the other's membership deleted", []uuid.UUID{project(true, member{bob, 20, true}, member{carol, 15, true})}, false},
		{"one of two active admins", []uuid.UUID{project(false, member{bob, 20, true}, member{alice, 20, true}, member{carol, 15, true})}, false},
		{"the other admin's membership ended", []uuid.UUID{project(false, member{bob, 20, true}, member{alice, 20, false},
			member{carol, 15, true})}, true},
		{"the other admin's membership deleted", []uuid.UUID{project(true, member{bob, 20, true}, member{carol, 15, true},
			member{alice, 20, true})}, true},
		{"a member, beside the only admin and another member", []uuid.UUID{project(false, member{bob, 15, true}, member{alice, 20, true},
			member{carol, 15, true})}, false},
		{"a member, beside another member and no admin", []uuid.UUID{project(false, member{bob, 15, true}, member{carol, 15, true})}, false},
		{"an admin whose membership ended", []uuid.UUID{project(false, member{bob, 20, false}, member{carol, 15, true})}, false},
		{"an admin whose membership is deleted", []uuid.UUID{project(true, member{carol, 15, true}, member{bob, 20, true})}, false},
		{"a member of the project asked about, the only admin of another", []uuid.UUID{project(false, member{bob, 15, true},
			member{alice, 20, true})}, false},
		{"the only admin of one of two asked about", []uuid.UUID{alone, withAMember}, true},
		{"none asked about", nil, false},
	}
	for _, tt := range tests {
		if got, err := s.SoleAdmin(context.Background(), tt.projects, bob); err != nil || got != tt.want {
			t.Errorf("%s: SoleAdmin() = %v, %v; want %v", tt.name, got, err, tt.want)
		}
	}
}

// LockActiveMemberProjects takes its locks in the projects' id order,
// whatever order the rows lie in: Web has the smaller id, but lies after
// Alpha in the table and in the indexes on the name and on the identifier.
// Alpha's row is held. LockActiveMemberProjects waits for it holding
// Web's, which a FOR SHARE then waits for; in any other order it would
// reach Alpha first and wait holding nothing.
func TestLockActiveMemberProjectsLocksInIDOrder(t *testing.T) {
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
			_, err := s.LockActiveMemberProjects(ctx, []uuid.UUID{acme}, bob)
			return err
		})
	}()
	pgtest.WaitForLockWaitOn(t, pool, "projects", 10*time.Second)

	if !waits(t, pool, web, "FOR SHARE") {
		t.Error("a FOR SHARE of Web while LockActiveMemberProjects waits for Alpha does not wait; want Web locked first")
	}
	if err := held.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("LockActiveMemberProjects() = %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("LockActiveMemberProjects() did not end within 10s")
	}
}

// A project deleted while LockActiveMemberProjects waits for its row is
// left out (M3 design 3.6): another transaction holds Web FOR NO KEY
// UPDATE, as deleteProject does, and soft-deletes it, its memberships too;
// LockActiveMemberProjects, which found Web undeleted, waits for it; once
// the deletion commits, it evaluates deleted_at again on the row's newest
// version and returns Ops alone.
func TestLockActiveMemberProjectsLeavesOutAProjectDeletedWhileItWaited(t *testing.T) {
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
	for _, sql := range []string{"SELECT 1 FROM projects WHERE id = $1 FOR NO KEY UPDATE", "UPDATE projects SET deleted_at = now() WHERE id = $1",
		"UPDATE project_members SET deleted_at = now() WHERE project_id = $1"} {
		if _, err := deletion.Exec(context.Background(), sql, web); err != nil {
			t.Fatal(err)
		}
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
			l.ids, err = s.LockActiveMemberProjects(ctx, []uuid.UUID{acme}, bob)
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
			t.Errorf("LockActiveMemberProjects() = %v, %v; want Ops (%s) alone, Web (%s) deleted while it waited", l.ids, l.err, ops, web)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("LockActiveMemberProjects() did not end within 10s")
	}
}
````

- [ ] **Step 3: 连带和接线**

`server/internal/modules/project/app/cascade.go`（修改，4 处）：

````old server/internal/modules/project/app/cascade.go
	"uuid"
````
````new server/internal/modules/project/app/cascade.go
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
````

````old server/internal/modules/project/app/cascade.go
	members  MemberDemoter
````
````new server/internal/modules/project/app/cascade.go
	members  MemberDemoter
	enders   MembershipEnder
````

````old server/internal/modules/project/app/cascade.go
// NewCascade returns the cascade over projects and members.
func NewCascade(projects ProjectsDeleter, members MemberDemoter) *Cascade {
	return &Cascade{projects: projects, members: members}
````
````new server/internal/modules/project/app/cascade.go
// NewCascade returns the cascade over projects, members and enders.
func NewCascade(projects ProjectsDeleter, members MemberDemoter, enders MembershipEnder) *Cascade {
	return &Cascade{projects: projects, members: members, enders: enders}
````

````old server/internal/modules/project/app/cascade.go
	return c.members.DemoteMemberships(ctx, projects, userID, by, now)
}

````
````new server/internal/modules/project/app/cascade.go
	return c.members.DemoteMemberships(ctx, projects, userID, by, now)
}

// EndMemberships ends userID's active memberships of the workspaces'
// projects (M3 design 3.6 convention 6, 3.7 rule 2), under the caller's FOR
// NO KEY UPDATE of each workspace, once the caller has ended his membership
// of it: the projects, found now, not given (a growth that committed before
// the caller's lock is among them), locked FOR NO KEY UPDATE in id order;
// then, were he the only active admin of one that has another active
// member, domain.ErrSoleAdmin, nothing written; else his memberships of
// them ended in one statement (convention 5), at now, by by. With no such
// project there is nothing to write.
func (c *Cascade) EndMemberships(ctx context.Context, workspaceIDs []uuid.UUID, userID, by uuid.UUID, now time.Time) error {
	projects, err := c.enders.LockActiveMemberProjects(ctx, workspaceIDs, userID)
	if err != nil || len(projects) == 0 {
		return err
	}
	sole, err := c.enders.SoleAdmin(ctx, projects, userID)
	switch {
	case err != nil:
		return err
	case sole:
		return domain.ErrSoleAdmin
	}
	return c.enders.EndMemberships(ctx, projects, userID, by, now)
}

````

`server/internal/modules/project/app/cascade_test.go`（修改，4 处）：

````old server/internal/modules/project/app/cascade_test.go
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
````
````new server/internal/modules/project/app/cascade_test.go
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
````

````old server/internal/modules/project/app/cascade_test.go
		err := app.NewCascade(f.store, &fakeDemoter{}).DeleteWorkspaceProjects(inTx, workspace, by, clockNow)
````
````new server/internal/modules/project/app/cascade_test.go
		err := app.NewCascade(f.store, &fakeDemoter{}, &fakeEnder{}).DeleteWorkspaceProjects(inTx, workspace, by, clockNow)
````

````old server/internal/modules/project/app/cascade_test.go
		err := app.NewCascade(newWrites().store, f).DemoteToGuest(inTx, workspace, user, by, now)
````
````new server/internal/modules/project/app/cascade_test.go
		err := app.NewCascade(newWrites().store, f, &fakeEnder{}).DemoteToGuest(inTx, workspace, user, by, now)
````

````old server/internal/modules/project/app/cascade_test.go
			t.Errorf("%s: DemoteToGuest() = %v, the calls\n%q\nwant %v,\n%q", tt.name, err, f.log.calls, tt.wantErr, tt.want)
		}
	}
}

````
````new server/internal/modules/project/app/cascade_test.go
			t.Errorf("%s: DemoteToGuest() = %v, the calls\n%q\nwant %v,\n%q", tt.name, err, f.log.calls, tt.wantErr, tt.want)
		}
	}
}

// fakeEnder is the memberships' repository of an ending: it records each
// call with its arguments, and " outside tx" when it ran outside the
// caller's transaction (callLog), answers LockActiveMemberProjects with
// locked and SoleAdmin with sole, and fails the call named in fail.
type fakeEnder struct {
	locked []uuid.UUID
	sole   bool
	log    callLog
	fail   string
}

func (f *fakeEnder) call(ctx context.Context, name, format string, args ...any) error {
	f.log.add(ctx, name+" "+format, args...)
	if name == f.fail {
		return errDisk
	}
	return nil
}

func (f *fakeEnder) LockActiveMemberProjects(ctx context.Context, workspaceIDs []uuid.UUID, userID uuid.UUID) ([]uuid.UUID, error) {
	if err := f.call(ctx, "LockActiveMemberProjects", "%v %s", workspaceIDs, userID); err != nil {
		return nil, err
	}
	return f.locked, nil
}

func (f *fakeEnder) SoleAdmin(ctx context.Context, projectIDs []uuid.UUID, userID uuid.UUID) (bool, error) {
	if err := f.call(ctx, "SoleAdmin", "%v %s", projectIDs, userID); err != nil {
		return false, err
	}
	return f.sole, nil
}

func (f *fakeEnder) EndMemberships(ctx context.Context, projectIDs []uuid.UUID, userID, by uuid.UUID, now time.Time) error {
	return f.call(ctx, "EndMemberships", "%v %s by %s at %s", projectIDs, userID, by, now.Format(time.RFC3339Nano))
}

// EndMemberships locks the account's projects with an active membership in
// the workspaces, found when it is called (M3 design 3.6 convention 6);
// asks whether he is the only admin of one of those it locked that has
// other members; then ends his memberships of them, by the caller's account
// at the caller's moment; all in the caller's transaction. The projects
// pass on as the lock returned them, in an order that is no sort's. Were he
// the only admin, project.sole_admin, and nothing is written (3.7 rule 2);
// with none locked it neither asks nor writes. A failing call comes back as
// itself, and nothing runs after it.
func TestEndMemberships(t *testing.T) {
	workspaces := []uuid.UUID{uuid.NewV7(), uuid.NewV7()}
	user, by := uuid.NewV7(), uuid.NewV7()
	now := time.Date(2026, 10, 2, 10, 0, 0, 123456000, time.UTC)
	first, second, third := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	projects := []uuid.UUID{second, first, third}
	lock := fmt.Sprintf("LockActiveMemberProjects %v %s", workspaces, user)
	ask := fmt.Sprintf("SoleAdmin %v %s", projects, user)
	end := fmt.Sprintf("EndMemberships %v %s by %s at %s", projects, user, by, now.Format(time.RFC3339Nano))
	tests := []struct {
		name    string
		locked  []uuid.UUID
		sole    bool
		fail    string
		wantErr error
		want    []string
	}{
		{"three projects", projects, false, "", nil, []string{lock, ask, end}},
		{"none", nil, false, "", nil, []string{lock}},
		{"the only admin of one", projects, true, "", domain.ErrSoleAdmin, []string{lock, ask}},
		{"the lock failing", projects, false, "LockActiveMemberProjects", errDisk, []string{lock}},
		{"the question failing", projects, false, "SoleAdmin", errDisk, []string{lock, ask}},
		{"the write failing", projects, false, "EndMemberships", errDisk, []string{lock, ask, end}},
	}
	for _, tt := range tests {
		f := &fakeEnder{locked: tt.locked, sole: tt.sole, fail: tt.fail}
		inTx := context.WithValue(context.Background(), inTxKey{}, true)
		err := app.NewCascade(newWrites().store, &fakeDemoter{}, f).EndMemberships(inTx, workspaces, user, by, now)
		if !errors.Is(err, tt.wantErr) || (tt.wantErr == nil) != (err == nil) || !slices.Equal(f.log.calls, tt.want) {
			t.Errorf("%s: EndMemberships() = %v, the calls\n%q\nwant %v,\n%q", tt.name, err, f.log.calls, tt.wantErr, tt.want)
		}
	}
}

````

`server/internal/modules/project/module.go`（修改，2 处）：

````old server/internal/modules/project/module.go
	DemoteToGuest(ctx context.Context, workspaceID, userID, by uuid.UUID, now time.Time) error
````
````new server/internal/modules/project/module.go
	DemoteToGuest(ctx context.Context, workspaceID, userID, by uuid.UUID, now time.Time) error
	// EndMemberships ends userID's active memberships of the workspaces'
	// projects, found when it is called; project.sole_admin, and nothing
	// ended, when he is the only active admin of one that has other active
	// members.
	EndMemberships(ctx context.Context, workspaceIDs []uuid.UUID, userID, by uuid.UUID, now time.Time) error
````

````old server/internal/modules/project/module.go
	return &Module{cascade: app.NewCascade(store, store), uc: httpadapter.UseCases{
````
````new server/internal/modules/project/module.go
	return &Module{cascade: app.NewCascade(store, store, store), uc: httpadapter.UseCases{
````

`server/internal/bootstrap/interleaving_growth_test.go`（修改，2 处）：

````old server/internal/bootstrap/interleaving_growth_test.go
			auth, cascade := r.authorizer(), projectapp.NewCascade(store, store)
````
````new server/internal/bootstrap/interleaving_growth_test.go
			auth, cascade := r.authorizer(), projectapp.NewCascade(store, store, store)
````

````old server/internal/bootstrap/interleaving_growth_test.go
				cascade = projectapp.NewCascade(store, gatedDemoter{store, g})
````
````new server/internal/bootstrap/interleaving_growth_test.go
				cascade = projectapp.NewCascade(store, gatedDemoter{store, g}, store)
````

`server/internal/bootstrap/interleaving_writes_test.go`（修改，2 处）：

````old server/internal/bootstrap/interleaving_writes_test.go
					demoted = run(func() error { return r.demote(ctx, workspacepg.New(r.pool), projectapp.NewCascade(store, store)) })
````
````new server/internal/bootstrap/interleaving_writes_test.go
					demoted = run(func() error {
						return r.demote(ctx, workspacepg.New(r.pool), projectapp.NewCascade(store, store, store))
					})
````

````old server/internal/bootstrap/interleaving_writes_test.go
						return r.demote(ctx, workspacepg.New(r.pool), projectapp.NewCascade(store, gatedDemoter{store, g}))
````
````new server/internal/bootstrap/interleaving_writes_test.go
						return r.demote(ctx, workspacepg.New(r.pool), projectapp.NewCascade(store, gatedDemoter{store, g}, store))
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
git add server/internal/bootstrap/interleaving_growth_test.go server/internal/bootstrap/interleaving_writes_test.go server/internal/modules/project/adapter/postgres/cascade.go server/internal/modules/project/adapter/postgres/end_test.go server/internal/modules/project/adapter/postgres/queries/cascade.sql server/internal/modules/project/app/cascade.go server/internal/modules/project/app/cascade_test.go server/internal/modules/project/app/ports.go server/internal/modules/project/domain/errors.go server/internal/modules/project/module.go server/internal/modules/project/adapter/postgres/gen/cascade.sql.go
```
```bash
git commit -m "feat(M3/P5a): the project cascade ends an account's project memberships

EndMemberships finds, when it is called, the workspaces' undeleted
projects in which the account has an active membership, locks them FOR
NO KEY UPDATE in id order, refuses with project.sole_admin when he is
the only active admin of one that has other active members, and else
ends his memberships of them in one statement, at the caller's moment,
by the caller's account (M3 design 3.6 convention 6, 3.7 rule 2).

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A）：

| 改坏 | 必须失败的测试 | 层 |
|---|---|---|
| `LockActiveMemberProjects` 去掉工作区 | `TestEndingAMembersProjectMemberships`；W12 | 存储；端到端 |
| `LockActiveMemberProjects` 的 `EXISTS` 去掉相关、成员、有效、`m.deleted_at`；整个 `EXISTS` 去掉 | `TestEndingAMembersProjectMemberships`；`TestEachLockOfAnEndingIsItsStrength`（相关、成员、整个 `EXISTS`：Site 被锁，Task 6 起）。故事看不到：只多锁，写由 `EndMemberships` 自己的谓词再筛一遍（spec 第 3 节第 9 条） | 存储；组合 |
| `LockActiveMemberProjects` 去掉 `p.deleted_at IS NULL` | `TestEndingAMembersProjectMemberships`、`TestLockActiveMemberProjectsLeavesOutAProjectDeletedWhileItWaited` | 存储 |
| 按行序而不是 id 顺序锁 | `TestLockActiveMemberProjectsLocksInIDOrder` | 存储 |
| 锁成 `FOR SHARE`、`FOR UPDATE`；不加锁 | `TestEndingAMembersProjectMemberships`；`TestEachLockOfAnEndingIsItsStrength`（Task 6 起） | 存储；组合 |
| `SoleAdmin` 去掉项目、成员、角色；另一个管理员的项目、成员、角色、有效、存在；另一个成员的项目、成员、有效、存在 | `TestSoleAdmin`；W12（项目），W7、W12（成员），W7（角色、另一个管理员的五条），W12（另一个成员的四条）（Task 13、14 起） | 存储；端到端 |
| `SoleAdmin` 去掉他的有效、已删除，另一个管理员的已删除，另一个成员的已删除 | `TestSoleAdmin`（故事看不到的理由见 spec 第 3 节第 9 条） | 存储 |
| `EndMemberships` 去掉项目、成员 | `TestEndingAMembersProjectMemberships`；W12（项目），W7、W12（成员） | 存储；端到端 |
| `EndMemberships` 去掉 `is_active`、`deleted_at IS NULL`；也改角色；不写结束者 | `TestEndingAMembersProjectMemberships`；W2、W7、W12 的写者（不写结束者，Task 13、14 起） | 存储；端到端 |
| 唯一管理员时照样结束 | `TestEndMemberships`；`TestAnEndingEndsTheMembershipsAndLeavesNoInvitation`（Task 5 起）；W7（Task 13 起） | 单元；组合；端到端 |
| 锁、查询、写的失败被吞掉；一个都没锁到也问、也写 | `TestEndMemberships` | 单元 |
| 锁住的项目按 id 排序之后再问、再写（清扫 16） | `TestEndMemberships`（假的锁按 second、first、third 回答，不是任何排序） | 单元 |
| `project.New` 的连带的第三个参数换成什么都不结束的 | `TestAnEndingEndsTheMembershipsAndLeavesNoInvitation`（Task 5 起） | 组合 |
| 三条语句经连接池执行 | `TestTheEndingsRunOnTheirTransactionsConnection`（Task 6 起） | 组合 |

**Done when:** 连带的第三个方法和三条语句通过存储和单元测试；锁的顺序在真实数据库上看得到。

---

### Task 3: 结束一步与 `removeWorkspaceMember` 的用例

**Files:**
- Create: `server/internal/modules/workspace/app/end_membership.go`、`server/internal/modules/workspace/app/remove_member.go`、`server/internal/modules/workspace/app/remove_member_test.go`
- Modify: `server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`、`server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go`、`server/internal/modules/workspace/app/clock_test.go`、`server/internal/modules/workspace/app/fakes_members_test.go`、`server/internal/modules/workspace/app/fakes_projects_test.go`、`server/internal/modules/workspace/app/fakes_workspaces_test.go`、`server/internal/modules/workspace/app/lock.go`、`server/internal/modules/workspace/app/ports.go`、`server/internal/modules/workspace/app/update_member.go`、`server/internal/modules/workspace/domain/actions.go`

**Interfaces:**
- Produces（spec 2.5、2.6，M3 设计 3.3、3.6 的加锁表、3.7 规则 2、3.8）：
  - `workspace/app.membershipEnd{members MembershipEnder, profiles MemberProfiles, projects ProjectCascade}`，`run(ctx, workspaceID, userID, by, now) error`：经 `MemberProfiles` 不加锁读他的地址（约定一；不是恰好一个时是一个错误，外键保证成员有账户）→ `DeletePendingInvitations` → `EndMember` → `ProjectCascade.EndMemberships([]uuid.UUID{workspaceID}, …)`。失败原样返回，之后的不执行；调用者的事务回滚，邀请的删除一起回滚。
  - `lockedMember(ctx, members MemberLocker, id)`：从 `UpdateWorkspaceMember` 提到 `lock.go`，两个用例共用：读成员行 → 锁它的工作区 `FOR NO KEY UPDATE` → 在锁下重读，须仍在那个工作区；不是时 `ErrMemberNotFound`。
  - `RemoveWorkspaceMember`，`NewRemoveWorkspaceMember(members MemberRemover, profiles, projects, auth, tx, clock)`，`Execute(ctx, id)`：一个事务里 `lockedMember` → 判定 `workspace_member.remove`（看不到时 `ErrMemberNotFound`）→ 已结束的 404 `workspace.member_not_found` → 自己的 409 `workspace.own_membership` → 读时钟（3.3）→ `membershipEnd.run`，由调用者。顺序照 G2：重读 → 判定 → 已结束 404 → 自己 409 → 时钟 → 结束一步。
  - 端口：`MemberLocker`（`MemberByID`、`LockWorkspace`，`MemberUpdater` 内嵌它）、`MembershipEnder`、`MemberRemover`；`ProjectCascade.EndMemberships`。
  - 操作名 `workspace_member.remove`；规则 `{Level: LevelWorkspace, Roles: [admin]}`（M3 设计 9.2）。

**Tests:**
- `app/remove_member_test.go`：`TestRemoveWorkspaceMemberLocksThenDecidesThenEnds`（调用的完整顺序：读、锁、重读、判定、地址、邀请、成员关系、项目，每个由 alice、在时钟的一个时刻，全部在一个事务里；bob 的成员关系结束）；`TestRemoveWorkspaceMemberRefusals`（不存在、等锁期间删除的成员关系，不存在、等锁期间删除、看不到的工作区都是 `workspace.member_not_found`；重读时到了另一个工作区的，在任何判定之前也是，尽管调用者也是那里的管理员；成员的 403 在检查目标之前，他什么都看不到；之后照 G2 的顺序：已结束的 404，自己的已结束的也是 404，自己的有效的 409；失败从来不是 404）；`TestRemoveWorkspaceMemberFailsWithinTheTransaction`（读地址失败、成员没有账户、结束一步的每一步失败、项目一侧的 `project.sole_admin`、提交被拒：答案是原样的错误，`project.sole_admin` 就是契约的 409；调用到失败的那一个为止，每个一次、都在事务里，之后的不调用、不重试）。
- `app/clock_test.go`：`TestEachWriteReadsTheClockUnderItsLock` 多移出一行（时钟在锁、判定、检查之后读一次）。
- `access/domain/rules_test.go`：`TestEveryRuleDecidesItsCells` 多 `workspace_member.remove` 的格子。

- [ ] **Step 1: 操作名、规则、端口**

`server/internal/modules/workspace/domain/actions.go`（修改，2 处）：

````old server/internal/modules/workspace/domain/actions.go
	ActionMemberUpdate shared.Action = "workspace_member.update"
````
````new server/internal/modules/workspace/domain/actions.go
	ActionMemberUpdate shared.Action = "workspace_member.update"
	// ActionMemberRemove is removing a member: removeWorkspaceMember.
	ActionMemberRemove shared.Action = "workspace_member.remove"
````

````old server/internal/modules/workspace/domain/actions.go
	return []shared.Action{ActionRead, ActionUpdate, ActionDelete, ActionMemberList, ActionMemberUpdate, ActionPreferencesRead, ActionPreferencesUpdate,
		ActionInvitationList, ActionInvitationCreate, ActionInvitationUpdate, ActionInvitationDelete}
````
````new server/internal/modules/workspace/domain/actions.go
	return []shared.Action{ActionRead, ActionUpdate, ActionDelete, ActionMemberList, ActionMemberUpdate, ActionMemberRemove, ActionPreferencesRead,
		ActionPreferencesUpdate, ActionInvitationList, ActionInvitationCreate, ActionInvitationUpdate, ActionInvitationDelete}
````

`server/internal/modules/access/domain/rules.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules.go
	"workspace_member.update": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin}},
````
````new server/internal/modules/access/domain/rules.go
	"workspace_member.update": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin}},
	// Removing another member: the workspace's admins (M3 design 9.2); one's
	// own membership is the use case's 409 (3.4).
	"workspace_member.remove": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin}},
````

`server/internal/modules/access/domain/rules_test.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules_test.go
	"workspace_member.update":      {allowed, forbidden, forbidden, invisible, invisible, invisible, forbidden},
````
````new server/internal/modules/access/domain/rules_test.go
	"workspace_member.update":      {allowed, forbidden, forbidden, invisible, invisible, invisible, forbidden},
	"workspace_member.remove":      {allowed, forbidden, forbidden, invisible, invisible, invisible, forbidden},
````

`server/internal/modules/workspace/app/ports.go`（修改，4 处）：

````old server/internal/modules/workspace/app/ports.go
// MemberUpdater changes a membership's role under its workspace's lock (M3
// design 3.6: read the row, lock the workspace, read the row again).
type MemberUpdater interface {
````
````new server/internal/modules/workspace/app/ports.go
// MemberLocker reads a membership and locks its workspace: the first steps
// of a write on a membership named by its id (M3 design 3.6: read the row,
// lock the workspace, read the row again).
type MemberLocker interface {
````

````old server/internal/modules/workspace/app/ports.go
	LockWorkspace(ctx context.Context, id uuid.UUID) error
````
````new server/internal/modules/workspace/app/ports.go
	LockWorkspace(ctx context.Context, id uuid.UUID) error
}

// MemberUpdater changes a membership's role under its workspace's lock.
type MemberUpdater interface {
	MemberLocker
````

````old server/internal/modules/workspace/app/ports.go
	UpdateMemberRole(ctx context.Context, id uuid.UUID, role shared.Role, by uuid.UUID, now time.Time) (domain.Membership, error)
````
````new server/internal/modules/workspace/app/ports.go
	UpdateMemberRole(ctx context.Context, id uuid.UUID, role shared.Role, by uuid.UUID, now time.Time) (domain.Membership, error)
}

// MembershipEnder ends a membership of a workspace under the workspace's
// FOR NO KEY UPDATE (M3 design 3.6, 3.8), in the transaction ctx carries.
type MembershipEnder interface {
	// DeletePendingInvitations soft-deletes the workspace's pending
	// invitation to email, if there is one, by the account by at now; a
	// declined one stays.
	DeletePendingInvitations(ctx context.Context, workspaceID uuid.UUID, email string, by uuid.UUID, now time.Time) error
	// EndMember ends userID's undeleted membership of the workspace, by the
	// account by at now; the row stays.
	EndMember(ctx context.Context, workspaceID, userID, by uuid.UUID, now time.Time) error
}

// MemberRemover removes a membership named by its id under its workspace's
// lock.
type MemberRemover interface {
	MemberLocker
	MembershipEnder
````

````old server/internal/modules/workspace/app/ports.go
	DemoteToGuest(ctx context.Context, workspaceID, userID, by uuid.UUID, now time.Time) error
````
````new server/internal/modules/workspace/app/ports.go
	DemoteToGuest(ctx context.Context, workspaceID, userID, by uuid.UUID, now time.Time) error
	// EndMemberships ends userID's active memberships of the workspaces'
	// projects, which it finds when it is called, at the moment and by the
	// account of the ending of his membership of those workspaces: an
	// admin's removal of him (removeWorkspaceMember), or his own leaving
	// (leaveWorkspace). It refuses with project.sole_admin, and ends none,
	// when he is the only active admin of one of them that has other active
	// members (M3 design 3.7 rule 2).
	EndMemberships(ctx context.Context, workspaceIDs []uuid.UUID, userID, by uuid.UUID, now time.Time) error
````

- [ ] **Step 2: 共用的锁和结束一步**

`server/internal/modules/workspace/app/lock.go`（修改，1 处）：

````old server/internal/modules/workspace/app/lock.go
}

// decide asks the Authorizer for action on the workspace for actor. A write
````
````new server/internal/modules/workspace/app/lock.go
}

// lockedMember reads the membership id, locks its workspace FOR NO KEY
// UPDATE, and reads it again under the lock, which must still be of the
// workspace locked (M3 design 3.6 convention 2): a membership or workspace
// deleted meanwhile, or a membership no longer of that workspace, is
// domain.ErrMemberNotFound.
func lockedMember(ctx context.Context, members MemberLocker, id uuid.UUID) (domain.Membership, error) {
	m, err := members.MemberByID(ctx, id)
	if err != nil {
		return domain.Membership{}, memberNotFound(err)
	}
	if err := members.LockWorkspace(ctx, m.WorkspaceID); err != nil {
		return domain.Membership{}, memberNotFound(err)
	}
	locked, err := members.MemberByID(ctx, id)
	if err != nil {
		return domain.Membership{}, memberNotFound(err)
	}
	if locked.WorkspaceID != m.WorkspaceID {
		return domain.Membership{}, domain.ErrMemberNotFound
	}
	return locked, nil
}

// memberNotFound turns ErrNotFound into domain.ErrMemberNotFound.
func memberNotFound(err error) error {
	if errors.Is(err, ErrNotFound) {
		return domain.ErrMemberNotFound
	}
	return err
}

// decide asks the Authorizer for action on the workspace for actor. A write
````

`server/internal/modules/workspace/app/update_member.go`（修改，3 处）：

````old server/internal/modules/workspace/app/update_member.go
	"context"
	"errors"
````
````new server/internal/modules/workspace/app/update_member.go
	"context"
````

````old server/internal/modules/workspace/app/update_member.go
		m, err := u.lockedMember(ctx, id)
````
````new server/internal/modules/workspace/app/update_member.go
		m, err := lockedMember(ctx, u.members, id)
````

````old server/internal/modules/workspace/app/update_member.go

// lockedMember reads the membership id, locks its workspace, and reads it
// again under the lock, which must still be of the workspace locked (M3
// design 3.6 convention 2): a membership or workspace deleted meanwhile,
// or a membership no longer of that workspace, is domain.ErrMemberNotFound.
func (u *UpdateWorkspaceMember) lockedMember(ctx context.Context, id uuid.UUID) (domain.Membership, error) {
	m, err := u.members.MemberByID(ctx, id)
	if err != nil {
		return domain.Membership{}, memberNotFound(err)
	}
	if err := u.members.LockWorkspace(ctx, m.WorkspaceID); err != nil {
		return domain.Membership{}, memberNotFound(err)
	}
	locked, err := u.members.MemberByID(ctx, id)
	if err != nil {
		return domain.Membership{}, memberNotFound(err)
	}
	if locked.WorkspaceID != m.WorkspaceID {
		return domain.Membership{}, domain.ErrMemberNotFound
	}
	return locked, nil
}

// memberNotFound turns ErrNotFound into domain.ErrMemberNotFound.
func memberNotFound(err error) error {
	if errors.Is(err, ErrNotFound) {
		return domain.ErrMemberNotFound
	}
	return err
}

````
````new server/internal/modules/workspace/app/update_member.go

````

`server/internal/modules/workspace/app/end_membership.go`（新文件，47 行）：

````file server/internal/modules/workspace/app/end_membership.go
package app

import (
	"context"
	"fmt"
	"time"
	"uuid"
)

// membershipEnd is the step removeWorkspaceMember and leaveWorkspace share
// (M3 design 3.6's lock table, 3.7 rule 2, 3.8): it ends an account's
// membership of a workspace, under the caller's FOR NO KEY UPDATE of it, at
// the caller's moment, read under that lock, by the caller's account.
type membershipEnd struct {
	members  MembershipEnder
	profiles MemberProfiles
	projects ProjectCascade
}

// run ends userID's membership of the workspace, in the caller's
// transaction: his address, read through MemberProfiles without a lock
// (convention 1); the workspace's pending invitation to it soft-deleted, so
// that the ended membership leaves no invitation (3.8), before his
// membership's row, as the global order has the invitations before the
// members; his membership ended, the row kept; then his memberships of the
// workspace's projects ended (ProjectCascade.EndMemberships), which refuses
// with project.sole_admin when he is the only admin of a project with other
// members (3.7 rule 2). A failure comes back as itself and nothing runs
// after it; the caller's transaction rolls back, the invitation's deletion
// with it.
func (e membershipEnd) run(ctx context.Context, workspaceID, userID, by uuid.UUID, now time.Time) error {
	profiles, err := e.profiles.PublicProfiles(ctx, []uuid.UUID{userID})
	if err != nil {
		return err
	}
	if len(profiles) != 1 {
		// The foreign key keeps every member's account: its absence is a bug.
		return fmt.Errorf("end the membership of %s in %s: no account", userID, workspaceID)
	}
	if err := e.members.DeletePendingInvitations(ctx, workspaceID, profiles[0].Email, by, now); err != nil {
		return err
	}
	if err := e.members.EndMember(ctx, workspaceID, userID, by, now); err != nil {
		return err
	}
	return e.projects.EndMemberships(ctx, []uuid.UUID{workspaceID}, userID, by, now)
}
````

- [ ] **Step 3: 用例和测试**

`server/internal/modules/workspace/app/remove_member.go`（新文件，61 行）：

````file server/internal/modules/workspace/app/remove_member.go
package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// RemoveWorkspaceMember removes a member from a workspace: DELETE
// /api/v0/workspace-members/{workspace_member_id}.
type RemoveWorkspaceMember struct {
	members MemberLocker
	end     membershipEnd
	auth    shared.Authorizer
	tx      shared.TxManager
	clock   Clock
}

// NewRemoveWorkspaceMember returns the use case.
func NewRemoveWorkspaceMember(members MemberRemover, profiles MemberProfiles, projects ProjectCascade, auth shared.Authorizer,
	tx shared.TxManager, clock Clock) *RemoveWorkspaceMember {
	return &RemoveWorkspaceMember{members: members, end: membershipEnd{members: members, profiles: profiles, projects: projects}, auth: auth,
		tx: tx, clock: clock}
}

// Execute removes the membership id, in one transaction (M3 design 3.6's
// lock table): the membership read for its workspace, the workspace row FOR
// NO KEY UPDATE, the membership read again under the lock, the decision on
// workspace_member.remove; then the checks on the target, which only a
// caller allowed to remove members gets to see, in updateWorkspaceMember's
// order: an ended membership is workspace.member_not_found, the caller's
// own workspace.own_membership (he leaves through leaveWorkspace); then the
// clock, read under the lock (3.3), and the ending: the workspace's pending
// invitation to his address, his membership, then his memberships of its
// projects (membershipEnd). Were he the only admin of a project with other
// members, project.sole_admin (3.7 rule 2), and the whole removal rolls
// back.
func (u *RemoveWorkspaceMember) Execute(ctx context.Context, id uuid.UUID) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	return u.tx.WithinTx(ctx, func(ctx context.Context) error {
		m, err := lockedMember(ctx, u.members, id)
		if err != nil {
			return err
		}
		if _, err := decide(ctx, u.auth, actor, domain.ActionMemberRemove, m.WorkspaceID, domain.ErrMemberNotFound); err != nil {
			return err
		}
		switch {
		case !m.IsActive:
			return domain.ErrMemberNotFound
		case m.MemberID == actor.UserID:
			return domain.ErrOwnMembership
		}
		return u.end.run(ctx, m.WorkspaceID, m.MemberID, actor.UserID, u.clock.Now())
	})
}
````

`server/internal/modules/workspace/app/fakes_members_test.go`（修改，1 处）：

````old server/internal/modules/workspace/app/fakes_members_test.go
	return domain.Membership{}, fmt.Errorf("update workspace member %s: no such row", id)
}

````
````new server/internal/modules/workspace/app/fakes_members_test.go
	return domain.Membership{}, fmt.Errorf("update workspace member %s: no such row", id)
}

// The ending's statements (app.MembershipEnder): each logs its call, fails
// with the error set for it, and changes what the fake holds as the store
// changes its rows.

func (f *fakeWorkspaces) DeletePendingInvitations(ctx context.Context, workspaceID uuid.UUID, email string, by uuid.UUID, now time.Time) error {
	f.log.add(ctx, "DeletePendingInvitations %s %s by %s at %s", workspaceID, email, by, now.Format(time.RFC3339Nano))
	if err := f.endErrs["DeletePendingInvitations"]; err != nil {
		return fmt.Errorf("delete the pending invitations: %w", err)
	}
	return nil
}

func (f *fakeWorkspaces) EndMember(ctx context.Context, workspaceID, userID, by uuid.UUID, now time.Time) error {
	f.log.add(ctx, "EndMember %s %s by %s at %s", workspaceID, userID, by, now.Format(time.RFC3339Nano))
	if err := f.endErrs["EndMember"]; err != nil {
		return fmt.Errorf("end workspace member: %w", err)
	}
	list := f.memberships[workspaceID]
	if i := slices.IndexFunc(list, func(m domain.Membership) bool { return m.MemberID == userID }); i >= 0 {
		list[i].IsActive = false
		return nil
	}
	return fmt.Errorf("end workspace member %s of %s: no such row", userID, workspaceID)
}

````

`server/internal/modules/workspace/app/fakes_workspaces_test.go`（修改，1 处）：

````old server/internal/modules/workspace/app/fakes_workspaces_test.go
	roleErr     error                             // for UpdateMemberRole
````
````new server/internal/modules/workspace/app/fakes_workspaces_test.go
	roleErr     error                             // for UpdateMemberRole
	endErrs     map[string]error                  // by method, for DeletePendingInvitations and EndMember
````

`server/internal/modules/workspace/app/fakes_projects_test.go`（修改，1 处）：

````old server/internal/modules/workspace/app/fakes_projects_test.go
		return fmt.Errorf("demote the member's project memberships: %w", err)
	}
	return nil
}

````
````new server/internal/modules/workspace/app/fakes_projects_test.go
		return fmt.Errorf("demote the member's project memberships: %w", err)
	}
	return nil
}

func (f *fakeProjects) EndMemberships(ctx context.Context, workspaceIDs []uuid.UUID, userID, by uuid.UUID, now time.Time) error {
	f.log.add(ctx, "EndMemberships %v %s by %s at %s", workspaceIDs, userID, by, now.Format(time.RFC3339Nano))
	if err := f.errs["EndMemberships"]; err != nil {
		return fmt.Errorf("end the member's project memberships: %w", err)
	}
	return nil
}

````

`server/internal/modules/workspace/app/remove_member_test.go`（新文件，202 行）：

````file server/internal/modules/workspace/app/remove_member_test.go
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

// newRemoveMember is RemoveWorkspaceMember over membersFixture's fakes: in
// acme alice is the admin, bob a member, carol's membership has ended.
func newRemoveMember() (*app.RemoveWorkspaceMember, *membersFixture, *fakeTx) {
	f := newMembers()
	tx := &fakeTx{}
	return app.NewRemoveWorkspaceMember(f.workspaces, f.profiles, f.projects, f.auth, tx, clockAt{at: clockNow}), f, tx
}

// removalCalls are the calls up to the decision on removing the membership
// m for user: lockedMemberCalls, deciding workspace_member.remove.
func removalCalls(user app.AccountState, m domain.Membership) []string {
	calls := lockedMemberCalls(user, m)
	calls[len(calls)-1] = "Authorize " + user.ID.String() + " workspace_member.remove on " + m.WorkspaceID.String() + "/" + uuid.Nil().String()
	return calls
}

// endingCalls are the calls of the ending of user's membership of
// workspace by by at the clock's time (membershipEnd): his address read,
// the pending invitation to it deleted, his membership ended, then his
// project memberships, in that order.
func endingCalls(workspace uuid.UUID, user app.AccountState, by uuid.UUID) []string {
	at := clockNow.Format(time.RFC3339Nano)
	return []string{
		fmt.Sprintf("PublicProfiles %v", []uuid.UUID{user.ID}),
		fmt.Sprintf("DeletePendingInvitations %s %s by %s at %s", workspace, user.Email, by, at),
		fmt.Sprintf("EndMember %s %s by %s at %s", workspace, user.ID, by, at),
		fmt.Sprintf("EndMemberships %v %s by %s at %s", []uuid.UUID{workspace}, user.ID, by, at),
	}
}

// RemoveWorkspaceMember reads the membership, locks its workspace FOR NO
// KEY UPDATE, reads it again, decides, then ends it: the pending
// invitation to the member's address, read through MemberProfiles, his
// membership, his memberships of acme's projects, each by alice at the
// clock's one time, all in one transaction (M3 design 3.6, 3.8). bob's
// membership is ended.
func TestRemoveWorkspaceMemberLocksThenDecidesThenEnds(t *testing.T) {
	uc, f, tx := newRemoveMember()

	err := uc.Execute(as(alice), bobInAcme.ID)

	want := slices.Concat(removalCalls(alice, bobInAcme), endingCalls(acme.ID, bob, alice.ID))
	if err != nil || !slices.Equal(f.log.calls, want) || tx.calls != 1 {
		t.Errorf("Execute() = %v, calls\n%q\nin %d transactions; want nil,\n%q\nin one", err, f.log.calls, tx.calls, want)
	}
	if m, _ := f.workspaces.MemberByID(context.Background(), bobInAcme.ID); m.IsActive {
		t.Errorf("bob's membership of acme after the removal: %+v, want it ended", m)
	}
}

// Each refusal is the answer, and nothing is ended. A membership that is
// not there or deleted meanwhile, of a workspace not there, deleted
// meanwhile or not visible is workspace.member_not_found, and so, before
// any decision, is one of another workspace when read again under the lock
// (M3 design 3.6 convention 2), though the caller is that one's admin too;
// a member's forbidden comes before any check of the target, so he learns
// nothing about it; then, in updateWorkspaceMember's order, an ended
// membership is workspace.member_not_found, also when it is the caller's
// own, and the caller's own active one workspace.own_membership; a failure
// is never a 404.
func TestRemoveWorkspaceMemberRefusals(t *testing.T) {
	failure := errors.New("connection reset")
	forbidBob := func(f *membersFixture) { f.auth.errs = map[grantKey]error{{bob.ID, acme.ID}: shared.Forbidden()} }
	stranger := domain.Membership{ID: uuid.NewV7(), WorkspaceID: uuid.NewV7(), MemberID: bob.ID, Role: shared.RoleMember, IsActive: true}
	decided := removalCalls(alice, bobInAcme)
	tests := []struct {
		name  string
		user  app.AccountState
		id    uuid.UUID
		set   func(f *membersFixture)
		want  error
		calls []string
	}{
		{"no such membership", alice, stranger.ID, nil, domain.ErrMemberNotFound, []string{"MemberByID " + stranger.ID.String()}},
		{"no such workspace", alice, stranger.ID,
			func(f *membersFixture) {
				f.workspaces.memberships[stranger.WorkspaceID] = []domain.Membership{stranger}
			},
			domain.ErrMemberNotFound, []string{"MemberByID " + stranger.ID.String(), "LockWorkspace " + stranger.WorkspaceID.String()}},
		{"acme deleted while the lock waited", alice, bobInAcme.ID,
			func(f *membersFixture) { f.workspaces.lockErrs = map[string]error{"acme": app.ErrNotFound} }, domain.ErrMemberNotFound, decided[:2]},
		{"deleted while the lock waited", alice, bobInAcme.ID,
			func(f *membersFixture) {
				f.workspaces.onLock = func() { f.workspaces.memberships[acme.ID] = []domain.Membership{aliceInAcme, carolInAcme} }
			},
			domain.ErrMemberNotFound, decided[:3]},
		{"of beta when read again under the lock", alice, bobInAcme.ID,
			func(f *membersFixture) {
				f.auth.grants[grantKey{alice.ID, beta.ID}] = shared.Grant{WorkspaceRole: shared.RoleAdmin}
				f.workspaces.onLock = func() { f.workspaces.memberships[acme.ID][1].WorkspaceID = beta.ID }
			},
			domain.ErrMemberNotFound, decided[:3]},
		{"not visible", carol, bobInAcme.ID, nil, domain.ErrMemberNotFound, removalCalls(carol, bobInAcme)},
		{"a member", bob, aliceInAcme.ID, forbidBob, shared.Forbidden(), removalCalls(bob, aliceInAcme)},
		{"a member, of an ended membership", bob, carolInAcme.ID, forbidBob, shared.Forbidden(), removalCalls(bob, carolInAcme)},
		{"a member, of his own", bob, bobInAcme.ID, forbidBob, shared.Forbidden(), removalCalls(bob, bobInAcme)},
		{"an ended membership", alice, carolInAcme.ID, nil, domain.ErrMemberNotFound, removalCalls(alice, carolInAcme)},
		{"ended while the lock waited", alice, bobInAcme.ID,
			func(f *membersFixture) {
				f.workspaces.onLock = func() { f.workspaces.memberships[acme.ID][1].IsActive = false }
			},
			domain.ErrMemberNotFound, decided},
		{"his own", alice, aliceInAcme.ID, nil, domain.ErrOwnMembership, removalCalls(alice, aliceInAcme)},
		{"his own, ended", alice, aliceInAcme.ID,
			func(f *membersFixture) { f.workspaces.memberships[acme.ID][0].IsActive = false }, domain.ErrMemberNotFound,
			removalCalls(alice, aliceInAcme)},
		{"the read failed", alice, bobInAcme.ID, func(f *membersFixture) { f.workspaces.membersErr = failure }, failure,
			[]string{"MemberByID " + bobInAcme.ID.String()}},
		{"the lock failed", alice, bobInAcme.ID, func(f *membersFixture) { f.workspaces.lockErrs = map[string]error{"acme": failure} },
			failure, decided[:2]},
		{"the read under the lock failed", alice, bobInAcme.ID,
			func(f *membersFixture) { f.workspaces.onLock = func() { f.workspaces.membersErr = failure } }, failure, decided[:3]},
		{"the Authorizer failed", alice, bobInAcme.ID,
			func(f *membersFixture) { f.auth.errs = map[grantKey]error{{alice.ID, acme.ID}: failure} }, failure, decided},
	}
	for _, tt := range tests {
		uc, f, tx := newRemoveMember()
		if tt.set != nil {
			tt.set(f)
		}
		err := uc.Execute(as(tt.user), tt.id)
		if !errors.Is(err, tt.want) {
			t.Errorf("%s: Execute() = %v; want %v", tt.name, err, tt.want)
		}
		if tt.want == failure && errors.Is(err, domain.ErrMemberNotFound) {
			t.Errorf("%s: Execute() = %v, which is also workspace.member_not_found", tt.name, err)
		}
		if !slices.Equal(f.log.calls, tt.calls) || tx.calls != 1 {
			t.Errorf("%s: calls = %q in %d transactions, want %q in one", tt.name, f.log.calls, tx.calls, tt.calls)
		}
	}
	uc, f, tx := newRemoveMember()
	if err := uc.Execute(context.Background(), bobInAcme.ID); !errors.Is(err, shared.Unauthenticated()) || len(f.log.calls) != 0 || tx.calls != 0 {
		t.Errorf("Execute() without an actor = %v, calls %q in %d transactions; want 401 unauthorized and no call", err, f.log.calls, tx.calls)
	}
}

// A failed read of the member's address, a member without an account, a
// failed step of the ending, the projects' refusal of the only admin of a
// project with other members, and a refused commit each fail the
// transaction, which the database then rolls back, the steps before with
// it: the answer is the error as it came, and project.sole_admin is itself,
// the 409 of the contract (M3 design 3.7 rule 2); the calls are the
// removal's own, each once and in the transaction, up to the failing one:
// nothing runs after it, and it is not tried again.
func TestRemoveWorkspaceMemberFailsWithinTheTransaction(t *testing.T) {
	failure := errors.New("connection reset")
	soleAdmin := shared.NewError(shared.KindConflict, "project.sole_admin", "Ending the membership would leave a project without an admin.")
	calls := slices.Concat(removalCalls(alice, bobInAcme), endingCalls(acme.ID, bob, alice.ID))
	decided := len(removalCalls(alice, bobInAcme))
	tests := []struct {
		name  string
		set   func(f *membersFixture, tx *fakeTx)
		want  error    // the error injected; nil for the use case's own, which is no *shared.Error
		calls []string // up to the failing call, the last one
	}{
		{"the address", func(f *membersFixture, _ *fakeTx) { f.profiles.err = failure }, failure, calls[:decided+1]},
		{"a member without an account", func(f *membersFixture, _ *fakeTx) { f.profiles.profiles = profiles[:2] }, nil, calls[:decided+1]},
		{"the invitations", func(f *membersFixture, _ *fakeTx) {
			f.workspaces.endErrs = map[string]error{"DeletePendingInvitations": failure}
		},
			failure, calls[:decided+2]},
		{"the membership", func(f *membersFixture, _ *fakeTx) { f.workspaces.endErrs = map[string]error{"EndMember": failure} }, failure,
			calls[:decided+3]},
		{"the projects' step", func(f *membersFixture, _ *fakeTx) { f.projects.errs = map[string]error{"EndMemberships": failure} }, failure,
			calls},
		{"the only admin of a project", func(f *membersFixture, _ *fakeTx) { f.projects.errs = map[string]error{"EndMemberships": soleAdmin} },
			soleAdmin, calls},
		{"the commit", func(_ *membersFixture, tx *fakeTx) { tx.commitErr = failure }, failure, calls},
	}
	for _, tt := range tests {
		uc, f, tx := newRemoveMember()
		tt.set(f, tx)
		err := uc.Execute(as(alice), bobInAcme.ID)
		var se *shared.Error
		switch {
		case tt.want == nil && (err == nil || errors.As(err, &se)):
			t.Errorf("%s failing: Execute() = %v; want an error that is no *shared.Error", tt.name, err)
		case tt.want != nil && !errors.Is(err, tt.want):
			t.Errorf("%s failing: Execute() = %v, want %v", tt.name, err, tt.want)
		}
		if !slices.Equal(f.log.calls, tt.calls) || tx.calls != 1 {
			t.Errorf("%s failing: calls\n%q\nin %d transactions; want\n%q\nin one", tt.name, f.log.calls, tx.calls, tt.calls)
		}
	}
}
````

`server/internal/modules/workspace/app/clock_test.go`（修改，2 处）：

````old server/internal/modules/workspace/app/clock_test.go
// every step, and a change to guest or a restoring as a guest for the
// projects' step. The clock logs its read among the fakes' calls.
````
````new server/internal/modules/workspace/app/clock_test.go
// every step, a change to guest or a restoring as a guest for the
// projects' step, and a removal for each step of the ending. The clock
// logs its read among the fakes' calls.
````

````old server/internal/modules/workspace/app/clock_test.go
			fmt.Sprintf("PublicProfiles %v", []uuid.UUID{bob.ID}))},
````
````new server/internal/modules/workspace/app/clock_test.go
			fmt.Sprintf("PublicProfiles %v", []uuid.UUID{bob.ID}))},
		{"removeWorkspaceMember", func() ([]string, error) {
			f := newMembers()
			err := app.NewRemoveWorkspaceMember(f.workspaces, f.profiles, f.projects, f.auth, &fakeTx{}, clockAt{clockNow, f.log}).
				Execute(as(alice), bobInAcme.ID)
			return f.log.calls, err
		}, slices.Concat(removalCalls(alice, bobInAcme), []string{"Now"}, endingCalls(acme.ID, bob, alice.ID))},
````

`server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go`（修改，1 处）：

````old server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
}

// allowAll allows every action, as the workspace's admin.
````
````new server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
}

func (noProjects) EndMemberships(context.Context, []uuid.UUID, uuid.UUID, uuid.UUID, time.Time) error {
	return errors.New("a deletion ended a member's project memberships")
}

// allowAll allows every action, as the workspace's admin.
````

- [ ] **Step 4: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/workspace/... ./internal/modules/access/...`
Expected: 全部 `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 5: 提交**

```bash
git add server/internal/modules/access/domain/rules.go server/internal/modules/access/domain/rules_test.go server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go server/internal/modules/workspace/app/clock_test.go server/internal/modules/workspace/app/end_membership.go server/internal/modules/workspace/app/fakes_members_test.go server/internal/modules/workspace/app/fakes_projects_test.go server/internal/modules/workspace/app/fakes_workspaces_test.go server/internal/modules/workspace/app/lock.go server/internal/modules/workspace/app/ports.go server/internal/modules/workspace/app/remove_member.go server/internal/modules/workspace/app/remove_member_test.go server/internal/modules/workspace/app/update_member.go server/internal/modules/workspace/domain/actions.go
```
```bash
git commit -m "feat(M3/P5a): the ending step and the removal of a workspace member

membershipEnd, which removeWorkspaceMember and leaveWorkspace share,
reads the member's address through MemberProfiles, soft-deletes the
workspace's pending invitation to it, ends his membership, then his
project memberships through ProjectCascade.EndMemberships. The removal
reads the membership, locks its workspace FOR NO KEY UPDATE, reads it
again, decides, answers an ended membership 404 and one's own 409,
then reads the clock and ends it. lockedMember moves out of the role
change, which shares it.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A）：

| 改坏 | 必须失败的测试 | 层 |
|---|---|---|
| 不判定 | `TestRemoveWorkspaceMemberRefusals`；`TestPermissionMatrix`（Task 4 起） | 单元；组合 |
| 判定成离开的操作；规则给成员 | `TestEveryRuleDecidesItsCells`（规则）；`TestPermissionMatrix`（Task 4、5 起） | 单元；组合 |
| 移出自己的成员关系 | `TestRemoveWorkspaceMemberRefusals`；`TestPermissionMatrix`（Task 4 起）；W7（Task 13 起） | 单元；组合；端到端 |
| 移出已结束的成员关系 | `TestRemoveWorkspaceMemberRefusals`；`TestPermissionMatrix`、`TestAnEndingFindsWhatEndedMeanwhile`（Task 4、6 起） | 单元；组合 |
| 已结束的 404 在判定之前 | `TestRemoveWorkspaceMemberRefusals`；`TestPermissionMatrix`（Task 4 起） | 单元；组合 |
| G2 的顺序反过来：自己的 409 在已结束的 404 之前 | `TestRemoveWorkspaceMemberRefusals`（组合一层看不到：通过判定的调用者是有效的管理员，"自己的"就是有效的，spec 第 3 节第 10 条） | 单元 |
| 不在锁下重读成员行 | `TestRemoveWorkspaceMemberRefusals`；`TestAnEndingFindsWhatEndedMeanwhile`（Task 6 起） | 单元；组合 |
| 在工作区的锁之前按第一次读到的成员关系判定 | `TestRemoveWorkspaceMemberLocksThenDecidesThenEnds`、`TestRemoveWorkspaceMemberRefusals`；`TestAnEndingFindsWhatEndedMeanwhile`（alice 的成员关系结束，Task 6 起） | 单元；组合 |
| 时钟在工作区的锁之前读 | `TestEachWriteReadsTheClockUnderItsLock`；`TestEachLockOfAnEndingIsItsStrength`（Task 6 起） | 单元；组合 |
| 结束一步：成员关系在邀请之前；项目在两者之前 | `TestRemoveWorkspaceMemberLocksThenDecidesThenEnds`（前一条）；`TestEachLockOfAnEndingIsItsStrength`（Task 6 起） | 单元；组合 |
| 结束一步跳过邀请；跳过项目 | `TestAnEndingEndsTheMembershipsAndLeavesNoInvitation`（Task 4、5 起）；`TestAnEndedMembershipLeavesNoInvitation`（邀请，Task 12 起） | 组合 |
| 读成员行和锁的失败被忽略；读地址、删邀请、结束成员关系、项目一步的失败被吞掉；删邀请调两次；拿到别的账户的地址 | `TestRemoveWorkspaceMemberRefusals`、`TestRemoveWorkspaceMemberFailsWithinTheTransaction`、`TestRemoveWorkspaceMemberLocksThenDecidesThenEnds` | 单元 |

**Done when:** 移出的用例和结束一步通过单元测试；改角色的用例照旧通过。

---

### Task 4: `removeWorkspaceMember` 的接口与组合

**Files:**
- Create: `server/internal/bootstrap/removal_test.go`
- Modify: `api/modules/workspace.yaml`、`server/internal/bootstrap/permission_matrix_coverage_test.go`、`server/internal/bootstrap/permission_matrix_workspace_test.go`、`server/internal/modules/workspace/adapter/http/handler.go`、`server/internal/modules/workspace/adapter/http/handler_test.go`、`server/internal/modules/workspace/adapter/http/members.go`、`server/internal/modules/workspace/adapter/http/members_test.go`、`server/internal/modules/workspace/module.go`、`web/apps/web/helpers/authentication.helper.ts`、`web/packages/i18n/src/locales/en/auth.json`、`web/packages/i18n/src/locales/zh-CN/auth.json`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/workspace/adapter/http/gen/server.gen.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Produces（spec 2.6，M3 设计 5.1、9.2、9.4）：`DELETE /api/v0/workspace-members/{workspace_member_id}`，204 无正文；码 `[workspace.member_not_found, forbidden, workspace.own_membership, project.sole_admin]`。`project.sole_admin` 是 `project` 模块的码，由工作区的操作声明：`workspace` 的 HTTP 测试为它答 409（9.4，M-2）。`PROBLEM_MESSAGES` 和两份 `auth.json` 加 `project.sole_admin` 的文案。
- `httpadapter.RemoveMemberUseCase`；`workspace.New` 的 `RemoveMember: app.NewRemoveWorkspaceMember(store, d.Profiles, d.Projects, d.Authorizer, d.Tx, d.Clock)`。
- 矩阵：`toMembership(method, target, body)`（改角色的行改用它，移出也用）；`endedMembership(c)`：每一列的目标工作区里一个已结束的成员关系（acme 的被移出的成员的；已删除的工作区那一列是 gone 的成员的，随 gone 删除）。移出三个变体：另一个成员 `ofMember(204, 403, 403)`；自己的 `ofMember(409 own_membership, 403, 403)`；已结束的 `ofMember(404 member_not_found, 403, 403)`；每个变体 6 格。
- `bootstrap/removal_test.go`（过渡版本，Task 5 换成 `ending_test.go`）：`endingWorld` 的准备和 `bystanders` 的前提。

**Tests:**
- `adapter/http/members_test.go`：`TestRemoveWorkspaceMember`（路径的成员关系交给用例，204 无正文）；`TestRemoveWorkspaceMemberRefusals`（用例的每个拒绝照契约答，`project.sole_admin` 原样是 409）。
- `bootstrap/removal_test.go`：`TestARemovalEndsTheMembershipsAndLeavesNoInvitation`（bob 是 Ops 唯一的管理员、carol 是它的成员：移出 409 `project.sole_admin`，任何表的任何行不变，发给他的待接受邀请仍待接受；alice 加入 Ops 之后，提交时被拒，对它写的每张表各一次，500 且一行不变；然后 204：他在 acme、Web、Ops 的成员关系结束，行留着、角色不变，acme 里发给他的待接受邀请删除，都由 alice、在不早于请求的一个时刻；他在 beta 的成员关系、beta 里发给他的邀请和其余每张表的每一行不变；结束 dave 的成员关系不动他已拒绝的邀请）。
- 矩阵：移出三行（18 格）；`TestMatrixViolationsCatchesEachGap` 的改角色一行改用 `toMembership`。

- [ ] **Step 1: 接口描述**

`api/modules/workspace.yaml`（修改，1 处）：

````old api/modules/workspace.yaml
                $ref: '#/components/schemas/WorkspaceMember'
````
````new api/modules/workspace.yaml
                $ref: '#/components/schemas/WorkspaceMember'
        default:
          $ref: '#/components/responses/Problem'
    delete:
      operationId: removeWorkspaceMember
      tags: [workspace]
      summary: Remove a member
      description: >-
        For the workspace's admins. A membership that does not exist or is
        deleted, or whose workspace the caller cannot see, answers
        workspace.member_not_found; a member or a guest, forbidden, whatever
        the membership. To a caller who may remove members, a membership that
        has ended answers workspace.member_not_found, and his own
        workspace.own_membership: one leaves a workspace through
        leaveWorkspace. The workspace's pending invitation to the member's
        address is deleted, so that he needs a new one to come back; a
        declined one stays. His membership ends, its row kept, and so do his
        memberships of the workspace's projects, all at the same moment, in
        one transaction. Were he the only admin of a project of the workspace
        that has other members, project.sole_admin, and nothing changes.
      security: [{bearer: []}]
      x-problem-codes: [workspace.member_not_found, forbidden, workspace.own_membership, project.sole_admin]
      responses:
        '204':
          description: The membership has ended.
````

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `239c909d2cce562c6ae94dd172e1e7d6712990a1ba6fef32185997d94973eaae` | 2392 | `api/dist/openapi.yaml` |
| `7186f287c82972e4c733eacbe767d82fc83c12d3d26692af484d6cef7afec0b8` | 2472 | `server/internal/modules/workspace/adapter/http/gen/server.gen.go` |
| `c71b8e839d8ccada04fadfae95d1a41025017e467c93ec02d0e48fada364970c` | 2578 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/workspace/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 2: handler、接线、文案**

`server/internal/modules/workspace/adapter/http/handler.go`（修改，2 处）：

````old server/internal/modules/workspace/adapter/http/handler.go
	Execute(ctx context.Context, id uuid.UUID, role shared.Role) (domain.Member, error)
````
````new server/internal/modules/workspace/adapter/http/handler.go
	Execute(ctx context.Context, id uuid.UUID, role shared.Role) (domain.Member, error)
}

// RemoveMemberUseCase is app.RemoveWorkspaceMember.
type RemoveMemberUseCase interface {
	Execute(ctx context.Context, id uuid.UUID) error
````

````old server/internal/modules/workspace/adapter/http/handler.go
	UpdateMember      UpdateMemberUseCase
````
````new server/internal/modules/workspace/adapter/http/handler.go
	UpdateMember      UpdateMemberUseCase
	RemoveMember      RemoveMemberUseCase
````

`server/internal/modules/workspace/adapter/http/members.go`（修改，1 处）：

````old server/internal/modules/workspace/adapter/http/members.go
}

// member is m as the API shows it.
````
````new server/internal/modules/workspace/adapter/http/members.go
}

// RemoveWorkspaceMember serves DELETE /api/v0/workspace-members/{workspace_member_id}.
func (h handler) RemoveWorkspaceMember(ctx context.Context, req gen.RemoveWorkspaceMemberRequestObject) (gen.RemoveWorkspaceMemberResponseObject, error) {
	if err := h.uc.RemoveMember.Execute(ctx, req.WorkspaceMemberID); err != nil {
		return nil, err
	}
	return gen.RemoveWorkspaceMember204Response{}, nil
}

// member is m as the API shows it.
````

`server/internal/modules/workspace/adapter/http/handler_test.go`（修改，4 处）：

````old server/internal/modules/workspace/adapter/http/handler_test.go
	role   *fakeUpdateMember
````
````new server/internal/modules/workspace/adapter/http/handler_test.go
	role   *fakeUpdateMember
	remove *fakeRemoveMember
````

````old server/internal/modules/workspace/adapter/http/handler_test.go
}

// fakePrefs is both preference use cases: each call is recorded as
````
````new server/internal/modules/workspace/adapter/http/handler_test.go
}

type fakeRemoveMember struct {
	calls []string // "caller id"
	err   error
}

func (f *fakeRemoveMember) Execute(ctx context.Context, id uuid.UUID) error {
	f.calls = append(f.calls, caller(ctx)+" "+id.String())
	return f.err
}

// fakePrefs is both preference use cases: each call is recorded as
````

````old server/internal/modules/workspace/adapter/http/handler_test.go
		f.role = &fakeUpdateMember{}
	}
````
````new server/internal/modules/workspace/adapter/http/handler_test.go
		f.role = &fakeUpdateMember{}
	}
	if f.remove == nil {
		f.remove = &fakeRemoveMember{}
	}
````

````old server/internal/modules/workspace/adapter/http/handler_test.go
		ListMembers: f.member, UpdateMember: f.role, GetPreferences: fakeGetPrefs{f.prefs}, UpdatePreferences: fakeUpdatePrefs{f.prefs},
		ListInvitations: fakeListInvitations{f.invitations}, CreateInvitations: fakeCreateInvitations{f.invitations},
````
````new server/internal/modules/workspace/adapter/http/handler_test.go
		ListMembers: f.member, UpdateMember: f.role, RemoveMember: f.remove, GetPreferences: fakeGetPrefs{f.prefs},
		UpdatePreferences: fakeUpdatePrefs{f.prefs},
		ListInvitations:   fakeListInvitations{f.invitations}, CreateInvitations: fakeCreateInvitations{f.invitations},
````

`server/internal/modules/workspace/adapter/http/members_test.go`（修改，2 处）：

````old server/internal/modules/workspace/adapter/http/members_test.go
import (
````
````new server/internal/modules/workspace/adapter/http/members_test.go
import (
	"fmt"
````

````old server/internal/modules/workspace/adapter/http/members_test.go
		t.Errorf("GET refused = %d %s, want 404 %s", res.StatusCode, body, want)
	}
}

````
````new server/internal/modules/workspace/adapter/http/members_test.go
		t.Errorf("GET refused = %d %s, want 404 %s", res.StatusCode, body, want)
	}
}

// DELETE removes the membership of the path for the caller and answers 204
// with no body.
func TestRemoveWorkspaceMember(t *testing.T) {
	remove := &fakeRemoveMember{}
	h := newServer(t, fakes{remove: remove})
	for _, token := range []string{"alice", "bob"} {
		if res, body := do(t, h, request(http.MethodDelete, "/api/v0/workspace-members/"+bobMember.ID.String(), token, "")); res.StatusCode != http.StatusNoContent ||
			body != "" {
			t.Errorf("%s's DELETE = %d %q, want 204 and no body", token, res.StatusCode, body)
		}
	}
	if want := []string{"alice " + bobMember.ID.String(), "bob " + bobMember.ID.String()}; !slices.Equal(remove.calls, want) {
		t.Errorf("calls = %q, want %q", remove.calls, want)
	}
}

// The use case's refusals, as the contract declares them: the project
// module's project.sole_admin comes through as it is, a 409 of the
// workspace's operation (M3 design 9.4).
func TestRemoveWorkspaceMemberRefusals(t *testing.T) {
	soleAdmin := shared.NewError(shared.KindConflict, "project.sole_admin", "The member is the only admin of a project.")
	tests := []struct {
		err    error
		status int
		want   string
	}{
		{domain.ErrMemberNotFound, http.StatusNotFound,
			`{"status":404,"code":"workspace.member_not_found","title":"Not Found","detail":"The member does not exist, or you cannot see the workspace."}`},
		{shared.Forbidden(), http.StatusForbidden, `{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
		{domain.ErrOwnMembership, http.StatusConflict,
			`{"status":409,"code":"workspace.own_membership","title":"Conflict","detail":"You cannot change your own membership."}`},
		{fmt.Errorf("end the member's project memberships: %w", soleAdmin), http.StatusConflict,
			`{"status":409,"code":"project.sole_admin","title":"Conflict","detail":"The member is the only admin of a project."}`},
	}
	for _, tt := range tests {
		h := newServer(t, fakes{remove: &fakeRemoveMember{err: tt.err}})
		res, body := do(t, h, request(http.MethodDelete, "/api/v0/workspace-members/"+bobMember.ID.String(), "alice", ""))
		if res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("DELETE refused with %v = %d %s, want %d %s", tt.err, res.StatusCode, body, tt.status, tt.want)
		}
	}
	remove := &fakeRemoveMember{}
	h := newServer(t, fakes{remove: remove})
	if res, _ := do(t, h, request(http.MethodDelete, "/api/v0/workspace-members/not-a-uuid", "alice", "")); res.StatusCode != http.StatusBadRequest ||
		len(remove.calls) != 0 {
		t.Errorf("DELETE of no membership id = %d, calls %q; want 400 and no call", res.StatusCode, remove.calls)
	}
}

````

`server/internal/modules/workspace/module.go`（修改，2 处）：

````old server/internal/modules/workspace/module.go
// listing the members and changing their roles, each member's display
// settings, and the invitations, and offers the other modules its reads
// through ports.
````
````new server/internal/modules/workspace/module.go
// listing the members, changing their roles and removing them, each
// member's display settings, and the invitations, and offers the other
// modules its reads through ports.
````

````old server/internal/modules/workspace/module.go
		UpdateMember:      app.NewUpdateWorkspaceMember(store, d.Projects, d.Profiles, d.Authorizer, d.Tx, d.Clock),
````
````new server/internal/modules/workspace/module.go
		UpdateMember:      app.NewUpdateWorkspaceMember(store, d.Projects, d.Profiles, d.Authorizer, d.Tx, d.Clock),
		RemoveMember:      app.NewRemoveWorkspaceMember(store, d.Profiles, d.Projects, d.Authorizer, d.Tx, d.Clock),
````

`web/apps/web/helpers/authentication.helper.ts`（修改，1 处）：

````old web/apps/web/helpers/authentication.helper.ts
  "project.archived": "auth.errors.project_archived",
````
````new web/apps/web/helpers/authentication.helper.ts
  "project.archived": "auth.errors.project_archived",
  "project.sole_admin": "auth.errors.project_sole_admin",
````

`web/packages/i18n/src/locales/en/auth.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/auth.json
      "project_archived": "The project is archived. Restore it to change it.",
````
````new web/packages/i18n/src/locales/en/auth.json
      "project_archived": "The project is archived. Restore it to change it.",
      "project_sole_admin": "A project that has other members would be left without an admin. Make another of its members an admin first.",
````

`web/packages/i18n/src/locales/zh-CN/auth.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/auth.json
      "project_archived": "项目已归档，恢复之后才能修改。",
````
````new web/packages/i18n/src/locales/zh-CN/auth.json
      "project_archived": "项目已归档，恢复之后才能修改。",
      "project_sole_admin": "一个还有别的成员的项目会因此没有管理员。请先把那个项目的另一位成员设为管理员。",
````

- [ ] **Step 3: 矩阵和组合出的移出**

`server/internal/bootstrap/permission_matrix_workspace_test.go`（修改，5 处）：

````old server/internal/bootstrap/permission_matrix_workspace_test.go
// toMembership is the request of a row whose callers each PATCH body to the
// membership target names for their column: a workspace's slug and whose
// membership of it.
func toMembership(target func(caller) (string, caller), body string) func(caller, seeded) (string, string, string) {
````
````new server/internal/bootstrap/permission_matrix_workspace_test.go
// toMembership is the request of a row whose callers each send method,
// with body, to the membership target names for their column: a
// workspace's slug and whose membership of it.
func toMembership(method string, target func(caller) (string, caller), body string) func(caller, seeded) (string, string, string) {
````

````old server/internal/bootstrap/permission_matrix_workspace_test.go
		return http.MethodPatch, "/api/v0/workspace-members/" + s.membership(slug, who).String(), body
````
````new server/internal/bootstrap/permission_matrix_workspace_test.go
		return method, "/api/v0/workspace-members/" + s.membership(slug, who).String(), body
````

````old server/internal/bootstrap/permission_matrix_workspace_test.go
	}
	return "acme", callerMember
````
````new server/internal/bootstrap/permission_matrix_workspace_test.go
	}
	return "acme", callerMember
}

// endedMembership is, for each column, an ended membership of the
// workspace the column targets: the removed member's of acme; for the
// deleted workspace's column, the member's of gone, deleted with it.
func endedMembership(c caller) (string, caller) {
	if c == callerDeleted {
		return "gone", callerMember
	}
	return "acme", callerRemoved
````

````old server/internal/bootstrap/permission_matrix_workspace_test.go
		{op: "updateWorkspaceMember", variant: "another member", write: true, request: toMembership(anotherMember, `{"role":5}`),
````
````new server/internal/bootstrap/permission_matrix_workspace_test.go
		{op: "updateWorkspaceMember", variant: "another member", write: true, request: toMembership(http.MethodPatch, anotherMember, `{"role":5}`),
````

````old server/internal/bootstrap/permission_matrix_workspace_test.go
		{op: "updateWorkspaceMember", variant: "one's own", write: true, request: toMembership(ownMembership, `{"role":15}`),
			cells: ofMember(cellOwnMembership, cellForbidden, cellForbidden)},
````
````new server/internal/bootstrap/permission_matrix_workspace_test.go
		{op: "updateWorkspaceMember", variant: "one's own", write: true, request: toMembership(http.MethodPatch, ownMembership, `{"role":15}`),
			cells: ofMember(cellOwnMembership, cellForbidden, cellForbidden)},
		// The admins remove another member (M3 design 9.2); to them alone an
		// ended membership is not found, and their own is the 409 of one's
		// own membership (3.4): a member or a guest is refused before any
		// check of the target.
		{op: "removeWorkspaceMember", variant: "another member", write: true, request: toMembership(http.MethodDelete, anotherMember, ""),
			cells: ofMember(cellNoContent, cellForbidden, cellForbidden)},
		{op: "removeWorkspaceMember", variant: "one's own", write: true, request: toMembership(http.MethodDelete, ownMembership, ""),
			cells: ofMember(cellOwnMembership, cellForbidden, cellForbidden)},
		{op: "removeWorkspaceMember", variant: "an ended membership", write: true, request: toMembership(http.MethodDelete, endedMembership, ""),
			cells: ofMember(cellMemberNotFound, cellForbidden, cellForbidden)},
````

`server/internal/bootstrap/permission_matrix_coverage_test.go`（修改，2 处）：

````old server/internal/bootstrap/permission_matrix_coverage_test.go
	demotes := matrixRow{op: "updateWorkspaceMember", write: true, request: toMembership(anotherMember, `{"role":5}`), cells: every(cellOK)}
````
````new server/internal/bootstrap/permission_matrix_coverage_test.go
	demotes := matrixRow{op: "updateWorkspaceMember", write: true, request: toMembership(http.MethodPatch, anotherMember, `{"role":5}`),
		cells: every(cellOK)}
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
		r.request = toMembership(func(c caller) (string, caller) {
````
````new server/internal/bootstrap/permission_matrix_coverage_test.go
		r.request = toMembership(http.MethodPatch, func(c caller) (string, caller) {
````

`server/internal/bootstrap/removal_test.go`（新文件，293 行）：

````file server/internal/bootstrap/removal_test.go
package bootstrap

import (
	"context"
	"encoding/json"
	"maps"
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// endingWorld is the wired app on a database of its own: acme, alice its
// admin; bob, carol and dave its members, each by accepting alice's
// invitation; Web, alice's, led by bob, so both are its admins, and carol
// its member by joining; Ops, bob's, his alone to administer, carol its
// member by his adding; beta, alice its admin, bob its member. Then,
// through the workspace store, the invitations no operation makes, 3.8
// refusing to invite an active member: one pending to bob's address in
// acme, stored as carol's, so that a claim of who deleted it can fail, and
// one in beta; one pending to carol's in acme; one to dave's in acme that
// he declined.
type endingWorld struct {
	contract       *apitest.Contract
	base           string
	pool           *pgxpool.Pool
	tokens         map[string]string    // access tokens, by name
	ids            map[string]uuid.UUID // accounts, by name
	web, ops       uuid.UUID
	bobsInvitation uuid.UUID // the pending one to bob's address in acme
	davesDeclined  uuid.UUID
}

func newEndingWorld(t *testing.T) endingWorld {
	t.Helper()
	contract := apitest.Load(t)
	dbURL := pgtest.NewDatabase(t)
	w := endingWorld{contract: contract, base: startApp(t, testConfig(t, dbURL, false), migrations.FS()), pool: openPool(t, dbURL),
		tokens: map[string]string{}, ids: map[string]uuid.UUID{}}
	for _, name := range []string{"alice", "bob", "carol", "dave"} {
		w.tokens[name] = registerAccount(t, contract, w.base, name+"@example.com").AccessToken
		w.ids[name] = accountID(t, contract, w.base, w.tokens[name])
	}
	for _, slug := range []string{"acme", "beta"} {
		if status, body := call(t, contract, http.MethodPost, w.base+"/api/v0/workspaces", w.tokens["alice"],
			`{"name":"`+slug+`","slug":"`+slug+`"}`); status != http.StatusCreated {
			t.Fatalf("creating %s = %d %s", slug, status, body)
		}
	}
	for _, m := range []struct{ slug, name string }{{"acme", "bob"}, {"acme", "carol"}, {"acme", "dave"}, {"beta", "bob"}} {
		answerInvitation(t, contract, w.base, w.tokens[m.name], "accept", invite(t, contract, w.base, w.tokens["alice"], m.slug, m.name+"@example.com"),
			http.StatusOK)
	}
	status, body := call(t, contract, http.MethodPost, w.base+"/api/v0/workspaces/acme/projects", w.tokens["alice"],
		`{"name":"Web","identifier":"WEB","project_lead_id":"`+w.ids["bob"].String()+`"}`)
	var web struct {
		ID uuid.UUID `json:"id"`
	}
	if status != http.StatusCreated {
		t.Fatalf("creating Web = %d %s", status, body)
	}
	decodeAnswer(t, body, &web)
	w.web, w.ops = web.ID, createdProject(t, contract, w.base, w.tokens["bob"], "acme", "Ops", "OPS")
	for _, step := range []struct{ token, path, body string }{
		{w.tokens["carol"], "/api/v0/projects/" + w.web.String() + "/join", ""},
		{w.tokens["bob"], "/api/v0/projects/" + w.ops.String() + "/members", `{"members":[{"member_id":"` + w.ids["carol"].String() + `","role":15}]}`},
	} {
		if status, body := call(t, contract, http.MethodPost, w.base+step.path, step.token, step.body); status != http.StatusOK &&
			status != http.StatusCreated {
			t.Fatalf("POST %s = %d %s", step.path, status, body)
		}
	}
	store, ctx := workspacepg.New(w.pool), context.Background()
	for _, inv := range []struct {
		slug, name string
		by         string
		id         *uuid.UUID
	}{{"acme", "bob", "carol", &w.bobsInvitation}, {"beta", "bob", "alice", nil}, {"acme", "carol", "alice", nil},
		{"acme", "dave", "alice", &w.davesDeclined}} {
		id := uuid.NewV7()
		if _, err := store.CreateInvitations(ctx, []workspaceapp.InvitationRow{{ID: id, WorkspaceID: w.workspace(t, inv.slug),
			Email: inv.name + "@example.com", Role: shared.RoleGuest, CreatedBy: w.ids[inv.by], Now: time.Now()}}); err != nil {
			t.Fatal(err)
		}
		if inv.id != nil {
			*inv.id = id
		}
	}
	if err := store.DeclineInvitation(ctx, w.davesDeclined, w.ids["dave"], time.Now()); err != nil {
		t.Fatal(err)
	}
	w.bystanders(t)
	return w
}

// bystanders checks the rows an ending of bob's membership of acme must
// leave as they were, each read by what makes it the one it is: his active
// membership of beta; the pending invitations to his address in beta and
// to carol's in acme; the one to dave's in acme, declined. Were one
// missing, an ending that also wrote it would pass.
func (w endingWorld) bystanders(t *testing.T) {
	t.Helper()
	var got string
	if err := w.pool.QueryRow(context.Background(), `SELECT concat_ws(', ',
		(SELECT 'bob in beta' FROM workspace_members m JOIN workspaces w ON w.id = m.workspace_id
			WHERE w.slug = 'beta' AND m.member_id = $1 AND m.is_active AND m.deleted_at IS NULL),
		(SELECT string_agg(w.slug || ' ' || i.email || CASE WHEN i.responded_at IS NULL THEN ' pending' WHEN i.accepted THEN ' accepted'
			ELSE ' declined' END, ', ' ORDER BY w.slug, i.email)
			FROM workspace_member_invites i JOIN workspaces w ON w.id = i.workspace_id WHERE i.deleted_at IS NULL))`, w.ids["bob"]).
		Scan(&got); err != nil {
		t.Fatal(err)
	}
	if want := "bob in beta, acme bob@example.com pending, acme carol@example.com pending, acme dave@example.com declined, " +
		"beta bob@example.com pending"; got != want {
		t.Fatalf("the rows an ending leaves: %s; want %s", got, want)
	}
}

// workspace is the id of the workspace slug.
func (w endingWorld) workspace(t *testing.T, slug string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := w.pool.QueryRow(context.Background(), "SELECT id FROM workspaces WHERE slug = $1 AND deleted_at IS NULL", slug).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// membership is the id of name's membership of acme.
func (w endingWorld) membership(t *testing.T, name string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := w.pool.QueryRow(context.Background(), `SELECT m.id FROM workspace_members m JOIN workspaces w ON w.id = m.workspace_id
		WHERE w.slug = 'acme' AND m.member_id = $1 AND m.deleted_at IS NULL`, w.ids[name]).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// remove is alice's removal of name from acme: its status and body.
func (w endingWorld) remove(t *testing.T, name string) (int, string) {
	t.Helper()
	return call(t, w.contract, http.MethodDelete, w.base+"/api/v0/workspace-members/"+w.membership(t, name).String(), w.tokens["alice"], "")
}

// rowJSON is the row id of table as JSON, its columns by name; nil when
// there is none.
func rowJSON(t *testing.T, pool *pgxpool.Pool, table string, id uuid.UUID) map[string]any {
	t.Helper()
	var text []byte
	if err := pool.QueryRow(context.Background(), "SELECT coalesce((SELECT row_to_json(r) FROM "+table+" r WHERE r.id = $1)::text, 'null')", id).
		Scan(&text); err != nil {
		t.Fatal(err)
	}
	var row map[string]any
	if err := json.Unmarshal(text, &row); err != nil {
		t.Fatal(err)
	}
	return row
}

// rowsBut is every table's rows as tableRows has them, River's left out,
// but the rows whose ids are in ids.
func rowsBut(t *testing.T, pool *pgxpool.Pool, ids []uuid.UUID) map[string]string {
	t.Helper()
	all := map[string]string{}
	for table := range tableRows(t, pool, riversOwn) {
		var text string
		if err := pool.QueryRow(context.Background(), `SELECT coalesce(string_agg(t, E'\n' ORDER BY t), '') FROM (SELECT row_to_json(r)::text AS t
			FROM `+table+` r WHERE NOT to_jsonb(r) ? 'id' OR (to_jsonb(r)->>'id') <> ALL ($1::text[])) s`, uuidTexts(ids)).Scan(&text); err != nil {
			t.Fatalf("%s: %v", table, err)
		}
		all[table] = text
	}
	return all
}

// uuidTexts are ids as text.
func uuidTexts(ids []uuid.UUID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id.String()
	}
	return out
}

// projectMemberships are the ids of user's memberships of the projects, in
// the order of projects.
func projectMemberships(t *testing.T, pool *pgxpool.Pool, user uuid.UUID, projects ...uuid.UUID) []uuid.UUID {
	t.Helper()
	ids := make([]uuid.UUID, len(projects))
	for i, p := range projects {
		if err := pool.QueryRow(context.Background(), "SELECT id FROM project_members WHERE project_id = $1 AND member_id = $2 AND deleted_at IS NULL",
			p, user).Scan(&ids[i]); err != nil {
			t.Fatal(err)
		}
	}
	return ids
}

// A removal ends a membership and leaves no invitation, in one transaction
// at one moment (M3 design 3.6, 3.7 rule 2, 3.8, 9.3), on the wired app.
//   - bob is Ops's only admin and carol its member: alice's removal of him
//     is 409 project.sole_admin, and no row of any table changes, the
//     pending invitation to his address, which the removal deletes before
//     the projects' step refuses, still pending.
//   - Once alice, acme's admin, has joined Ops, as its admin, the removal
//     refused at its commit, after every statement ran, for each table it
//     writes, is 500, and no row changes: no step wrote in a transaction of
//     its own.
//   - Then it is 204: bob's membership of acme, of Web and of Ops ended,
//     each row kept with its role; the pending invitation to his address in
//     acme deleted; each by alice, at one moment no earlier than the
//     request. His membership of beta, the invitation to him there, and
//     every other row of every table are as they were.
//   - Removing dave leaves his declined invitation as it was.
func TestARemovalEndsTheMembershipsAndLeavesNoInvitation(t *testing.T) {
	w := newEndingWorld(t)
	before := tableRows(t, w.pool, riversOwn)
	if status, body := w.remove(t, "bob"); status != http.StatusConflict || problemCode(t, []byte(body)) != "project.sole_admin" {
		t.Fatalf("removing bob, Ops's only admin = %d %s, want 409 project.sole_admin", status, body)
	}
	if after := tableRows(t, w.pool, riversOwn); !maps.Equal(after, before) {
		t.Errorf("the tables after the refused removal changed:\n%v\nwant them as they were:\n%v", after, before)
	}
	if status, body := call(t, w.contract, http.MethodPost, w.base+"/api/v0/projects/"+w.ops.String()+"/join", w.tokens["alice"], ""); status != http.StatusOK {
		t.Fatalf("alice's joining Ops = %d %s", status, body)
	}
	before = tableRows(t, w.pool, riversOwn)
	for _, table := range []string{"workspace_member_invites", "workspace_members", "project_members"} {
		restore := refusingCommits(t, w.pool, table)
		status, body := w.remove(t, "bob")
		restore()
		if after := tableRows(t, w.pool, riversOwn); status != http.StatusInternalServerError || !maps.Equal(after, before) {
			t.Errorf("the removal refused at its commit for %s = %d %s; want 500 and every table as it was", table, status, body)
		}
	}
	written := slices.Concat([]uuid.UUID{w.membership(t, "bob"), w.bobsInvitation}, projectMemberships(t, w.pool, w.ids["bob"], w.web, w.ops))
	tables := []string{"workspace_members", "workspace_member_invites", "project_members", "project_members"}
	rowsBefore := make([]map[string]any, len(written))
	for i, id := range written {
		rowsBefore[i] = rowJSON(t, w.pool, tables[i], id)
	}
	if by := []any{rowsBefore[0]["updated_by_id"], rowsBefore[1]["updated_by_id"], rowsBefore[3]["updated_by_id"]}; !slices.Equal(by,
		[]any{w.ids["bob"].String(), w.ids["carol"].String(), w.ids["bob"].String()}) {
		t.Fatalf("bob's membership of acme, the invitation to him, his membership of Ops last written by %v; want bob, carol, bob: "+
			"a claim of alice's writing must be able to fail", by)
	}
	others := rowsBut(t, w.pool, written)
	started := time.Now()

	if status, body := w.remove(t, "bob"); status != http.StatusNoContent {
		t.Fatalf("removing bob = %d %s, want 204", status, body)
	}

	moment, _ := rowJSON(t, w.pool, "workspace_member_invites", w.bobsInvitation)["deleted_at"].(string)
	if at, err := time.Parse(time.RFC3339Nano, moment); err != nil || at.Before(started.Truncate(time.Microsecond)) {
		t.Errorf("the invitation deleted at %q (%v); want a moment no earlier than the request, %v", moment, err, started)
	}
	for i, id := range written {
		after := rowJSON(t, w.pool, tables[i], id)
		want := maps.Clone(rowsBefore[i])
		want["updated_at"], want["updated_by_id"] = moment, w.ids["alice"].String()
		if tables[i] == "workspace_member_invites" {
			want["deleted_at"] = moment
		} else {
			want["is_active"] = false
		}
		if !maps.Equal(after, want) {
			t.Errorf("%s %s after the removal:\n%v\nwant\n%v", tables[i], id, after, want)
		}
	}
	if after := rowsBut(t, w.pool, written); !maps.Equal(after, others) {
		t.Errorf("every other row after the removal:\n%v\nwant them as they were:\n%v", after, others)
	}
	declined := rowJSON(t, w.pool, "workspace_member_invites", w.davesDeclined)
	if status, body := w.remove(t, "dave"); status != http.StatusNoContent {
		t.Fatalf("removing dave = %d %s, want 204", status, body)
	}
	if after := rowJSON(t, w.pool, "workspace_member_invites", w.davesDeclined); !maps.Equal(after, declined) {
		t.Errorf("dave's declined invitation after his removal:\n%v\nwant it as it was:\n%v", after, declined)
	}
}
````

- [ ] **Step 4: 测试、lint、前端检查**

Run: `go -C server test -count=1 ./internal/modules/workspace/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestPermissionMatrix$|TestThePermissionMatrixCoversEveryOperation|TestMatrixViolationsCatchesEach|TestARemovalEndsTheMembershipsAndLeavesNoInvitation|TestAPIRoutesAreTheContractsOperations' ./internal/bootstrap/`
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
git add api/modules/workspace.yaml server/internal/bootstrap/permission_matrix_coverage_test.go server/internal/bootstrap/permission_matrix_workspace_test.go server/internal/bootstrap/removal_test.go server/internal/modules/workspace/adapter/http/handler.go server/internal/modules/workspace/adapter/http/handler_test.go server/internal/modules/workspace/adapter/http/members.go server/internal/modules/workspace/adapter/http/members_test.go server/internal/modules/workspace/module.go web/apps/web/helpers/authentication.helper.ts web/packages/i18n/src/locales/en/auth.json web/packages/i18n/src/locales/zh-CN/auth.json api/dist/openapi.yaml server/internal/modules/workspace/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M3/P5a): removeWorkspaceMember

DELETE /api/v0/workspace-members/{workspace_member_id}, for the
workspace's admins; it answers the project module's project.sole_admin
as its own 409. The matrix gains its three rows, and the wired app's
removal ends the memberships, leaves no pending invitation and changes
nothing when refused, also at its commit.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A）：

| 改坏 | 必须失败的测试 | 层 |
|---|---|---|
| handler 吞掉用例的失败；把 `project.sole_admin` 答成别的 | `TestRemoveWorkspaceMember`、`TestRemoveWorkspaceMemberRefusals`（HTTP） | 单元 |
| `workspace.New` 的移出不带项目的连带、不开事务、用停在 2001 年的时钟 | `TestARemovalEndsTheMembershipsAndLeavesNoInvitation`；Task 5 起 `TestAnEndingEndsTheMembershipsAndLeavesNoInvitation` | 组合 |
| 规则给成员；不判定 | `TestPermissionMatrix` | 组合 |

**Done when:** 移出的三行矩阵（18 格）通过；`workspace` 的 `apitest.Main` 两个方向核对通过；组合出的移出拒绝时一行不改。

---

### Task 5: `leaveWorkspace`；组合出的结束改为移出和离开共用

**Files:**
- Create: `server/internal/bootstrap/ending_test.go`、`server/internal/modules/workspace/app/leave_workspace.go`、`server/internal/modules/workspace/app/leave_workspace_test.go`
- Modify: `api/modules/workspace.yaml`、`api/openapi.yaml`、`server/internal/bootstrap/permission_matrix_columns_test.go`、`server/internal/bootstrap/permission_matrix_coverage_test.go`、`server/internal/bootstrap/permission_matrix_seed_test.go`、`server/internal/bootstrap/permission_matrix_targets_test.go`、`server/internal/bootstrap/permission_matrix_test.go`、`server/internal/bootstrap/permission_matrix_workspace_test.go`、`server/internal/bootstrap/project_write_locks_test.go`、`server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`、`server/internal/modules/workspace/adapter/http/handler.go`、`server/internal/modules/workspace/adapter/http/handler_test.go`、`server/internal/modules/workspace/adapter/http/members.go`、`server/internal/modules/workspace/adapter/http/members_test.go`、`server/internal/modules/workspace/app/clock_test.go`、`server/internal/modules/workspace/app/fakes_members_test.go`、`server/internal/modules/workspace/app/fakes_workspaces_test.go`、`server/internal/modules/workspace/app/ports.go`、`server/internal/modules/workspace/domain/actions.go`、`server/internal/modules/workspace/domain/errors.go`、`server/internal/modules/workspace/module.go`、`web/apps/web/helpers/authentication.helper.ts`、`web/packages/i18n/src/locales/en/auth.json`、`web/packages/i18n/src/locales/zh-CN/auth.json`
- Delete: `server/internal/bootstrap/removal_test.go`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/workspace/adapter/http/gen/server.gen.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Produces（spec 2.7，M3 设计 3.7 规则 1、5.1、9.2）：`POST /api/v0/workspaces/{slug}/leave`，204 无正文；码 `[workspace.not_found, workspace.sole_admin, project.sole_admin]`。操作名 `workspace.leave`，规则 `{Level: LevelWorkspace, Roles: [admin, member, guest]}`：每个有效成员都可以离开。`workspace/domain.ErrSoleAdmin`（409 `workspace.sole_admin`，他是唯一的成员时也是）；文案进 `PROBLEM_MESSAGES` 和两份 `auth.json`。
- `LeaveWorkspace`，`NewLeaveWorkspace(workspaces WorkspaceLeaver, profiles, projects, auth, tx, clock)`，`Execute(ctx, slug)`：一个事务里工作区 `FOR NO KEY UPDATE`（按 slug）→ 判定 → 他是管理员时 `HasOtherAdmin`，没有则 `ErrSoleAdmin` → 读时钟 → `membershipEnd.run`，由他自己。`WorkspaceLeaver` = `LockWorkspaceBySlug` + `HasOtherAdmin` + `MembershipEnder`。
- 矩阵：唯一管理员的一张表 `soleAdminColumns = [callerSoleAdmin]`：other 的管理员（从来不是 acme 成员的那个账户），other 里还有被移出的成员作它的成员；离开两行：`inWorkspace(204, 204, 204)`（6 格；acme 的管理员旁边有 PM+WA）和"唯一管理员"一格 409 `workspace.sole_admin`。`projectTables` 之外的列（唯一管理员的表）里放项目的操作是一个缺口（`TestMatrixViolationsCatchesEachColumnGap`）；`preconditions` 核对 other 的管理员是它唯一的有效管理员、acme 的管理员有另一位、被移出的成员是 other 的有效成员。
- 写在项目一级的完整性核对（`writesOnAProject`）只数项目表的列：离开工作区的唯一管理员一行不是项目级的写（`TestWritesOnAProjectAreEachShape` 多它一例）。
- `bootstrap/ending_test.go` 取代 `removal_test.go`：`ending{name, request, by}`，`endings` 是移出和离开。

**Tests:**
- `app/leave_workspace_test.go`：`TestLeaveWorkspaceLocksDecidesThenEnds`（锁、判定、结束他自己的成员关系，由他自己、在时钟的一个时刻；成员和访客不再多问；管理员在工作区另有有效管理员时离开）；`TestLeaveWorkspaceRefusals`（不存在、等锁期间删除、他不是有效成员的工作区是 `workspace.not_found`；acme 唯一的有效管理员是 `workspace.sole_admin`，旁边有成员或只有他自己，在判定之后才问；失败从来不是 404；问另一个管理员失败原样返回）；`TestLeaveWorkspaceFailsWithinTheTransaction`（同移出）。
- `adapter/http/members_test.go`：`TestLeaveWorkspace`、`TestLeaveWorkspaceRefusals`（`project.sole_admin` 原样）。
- `bootstrap/ending_test.go`：`TestAnEndingEndsTheMembershipsAndLeavesNoInvitation`（Task 4 的测试，对移出和离开各一次；写者另由 dave 先写过每一行，"由结束者写"可以不成立）；`TestTheOnlyAdminCannotLeave`（alice 在只有她一人的 solo 和有成员的 acme 都是 409 `workspace.sole_admin`，任何行不变；carol 也成为 acme 的管理员之后 alice 离开）。
- 矩阵：离开两行（7 格）；`TestEveryColumnCallsAsARegisteredAccount`、`TestMatrixViolationsCatchesEachColumnGap`、`TestWritesOnAProjectAreEachShape` 各多一例。

- [ ] **Step 1: 接口描述**

`api/modules/workspace.yaml`（修改，1 处）：

````old api/modules/workspace.yaml
                $ref: '#/components/schemas/WorkspaceMemberList'
````
````new api/modules/workspace.yaml
                $ref: '#/components/schemas/WorkspaceMemberList'
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/workspaces/{slug}/leave:
    parameters:
      - $ref: '#/components/parameters/Slug'
    post:
      operationId: leaveWorkspace
      tags: [workspace]
      summary: Leave a workspace
      description: >-
        Every active member leaves a workspace himself. A workspace that does
        not exist, is deleted, or of which the caller is not an active member
        answers workspace.not_found. Its only active admin is refused
        workspace.sole_admin, also when he is its only member: he makes
        another member an admin first. The workspace's pending invitation to
        his address is deleted, so that he needs a new one to come back; a
        declined one stays. His membership ends, its row kept, and so do his
        memberships of the workspace's projects, all at the same moment, in
        one transaction. Were he the only admin of a project of the workspace
        that has other members, project.sole_admin, and nothing changes.
      security: [{bearer: []}]
      x-problem-codes: [workspace.not_found, workspace.sole_admin, project.sole_admin]
      responses:
        '204':
          description: The caller's membership has ended.
````

`api/openapi.yaml`（修改，1 处）：

````old api/openapi.yaml
    $ref: 'modules/workspace.yaml#/paths/~1api~1v0~1workspaces~1{slug}~1members'
````
````new api/openapi.yaml
    $ref: 'modules/workspace.yaml#/paths/~1api~1v0~1workspaces~1{slug}~1members'
  /api/v0/workspaces/{slug}/leave:
    $ref: 'modules/workspace.yaml#/paths/~1api~1v0~1workspaces~1{slug}~1leave'
````

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `b76e499e3dc8f5205dc423a27d5a2a4a61657a9191c1dbf2a5cd89ec3a317807` | 2412 | `api/dist/openapi.yaml` |
| `9abf666781db6e1be2189f52003606fba911730388537f330812dad8d6a8b829` | 2571 | `server/internal/modules/workspace/adapter/http/gen/server.gen.go` |
| `cfeef9c77c34f5e1703dc59c88544a12a0fdfbd031abe451b33d114a57ebc7df` | 2623 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/workspace/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 2: 领域、规则、用例**

`server/internal/modules/workspace/domain/actions.go`（修改，2 处）：

````old server/internal/modules/workspace/domain/actions.go
	ActionMemberRemove shared.Action = "workspace_member.remove"
````
````new server/internal/modules/workspace/domain/actions.go
	ActionMemberRemove shared.Action = "workspace_member.remove"
	// ActionLeave is leaving a workspace: leaveWorkspace.
	ActionLeave shared.Action = "workspace.leave"
````

````old server/internal/modules/workspace/domain/actions.go
	return []shared.Action{ActionRead, ActionUpdate, ActionDelete, ActionMemberList, ActionMemberUpdate, ActionMemberRemove, ActionPreferencesRead,
		ActionPreferencesUpdate, ActionInvitationList, ActionInvitationCreate, ActionInvitationUpdate, ActionInvitationDelete}
````
````new server/internal/modules/workspace/domain/actions.go
	return []shared.Action{ActionRead, ActionUpdate, ActionDelete, ActionLeave, ActionMemberList, ActionMemberUpdate, ActionMemberRemove,
		ActionPreferencesRead, ActionPreferencesUpdate, ActionInvitationList, ActionInvitationCreate, ActionInvitationUpdate, ActionInvitationDelete}
````

`server/internal/modules/workspace/domain/errors.go`（修改，1 处）：

````old server/internal/modules/workspace/domain/errors.go
	ErrOwnMembership = shared.NewError(shared.KindConflict, "workspace.own_membership", "You cannot change your own membership.")
````
````new server/internal/modules/workspace/domain/errors.go
	ErrOwnMembership = shared.NewError(shared.KindConflict, "workspace.own_membership", "You cannot change your own membership.")
	// ErrSoleAdmin answers the leaving of a workspace's only active admin,
	// also when he is its only member (M3 design 3.7 rule 1).
	ErrSoleAdmin = shared.NewError(shared.KindConflict, "workspace.sole_admin",
		"The workspace would be left without an admin; make another member an admin first.")
````

`server/internal/modules/access/domain/rules.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules.go
	"workspace.read":        {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	"workspace.update":      {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin}},
	"workspace.delete":      {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin}},
````
````new server/internal/modules/access/domain/rules.go
	"workspace.read":   {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	"workspace.update": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin}},
	"workspace.delete": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin}},
	// One's own membership: every active member; the only admin's 409 is
	// the use case's (M3 design 3.7 rule 1, 9.2).
	"workspace.leave":       {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
````

`server/internal/modules/access/domain/rules_test.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules_test.go
	"workspace.delete":             {allowed, forbidden, forbidden, invisible, invisible, invisible, forbidden},
````
````new server/internal/modules/access/domain/rules_test.go
	"workspace.delete":             {allowed, forbidden, forbidden, invisible, invisible, invisible, forbidden},
	"workspace.leave":              {allowed, allowed, allowed, invisible, invisible, invisible, forbidden},
````

`server/internal/modules/workspace/app/ports.go`（修改，1 处）：

````old server/internal/modules/workspace/app/ports.go
}

// MemberRemover removes a membership named by its id under its workspace's
````
````new server/internal/modules/workspace/app/ports.go
}

// WorkspaceLeaver ends the caller's own membership of a workspace named by
// its slug, under the workspace's lock.
type WorkspaceLeaver interface {
	WorkspaceLocker
	MembershipEnder
	// HasOtherAdmin reports whether the workspace has an active admin other
	// than userID.
	HasOtherAdmin(ctx context.Context, workspaceID, userID uuid.UUID) (bool, error)
}

// MemberRemover removes a membership named by its id under its workspace's
````

`server/internal/modules/workspace/app/leave_workspace.go`（新文件，58 行）：

````file server/internal/modules/workspace/app/leave_workspace.go
package app

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// LeaveWorkspace ends the caller's own membership of a workspace: POST
// /api/v0/workspaces/{slug}/leave.
type LeaveWorkspace struct {
	workspaces WorkspaceLeaver
	end        membershipEnd
	auth       shared.Authorizer
	tx         shared.TxManager
	clock      Clock
}

// NewLeaveWorkspace returns the use case.
func NewLeaveWorkspace(workspaces WorkspaceLeaver, profiles MemberProfiles, projects ProjectCascade, auth shared.Authorizer, tx shared.TxManager,
	clock Clock) *LeaveWorkspace {
	return &LeaveWorkspace{workspaces: workspaces, end: membershipEnd{members: workspaces, profiles: profiles, projects: projects}, auth: auth,
		tx: tx, clock: clock}
}

// Execute ends the caller's membership of the workspace slug, in one
// transaction (M3 design 3.6's lock table): the workspace row FOR NO KEY
// UPDATE, then the decision on workspace.leave, which every active member
// may take; then, the caller being its admin, whether the workspace has
// another active admin: if not, workspace.sole_admin, also when he is its
// only member (3.7 rule 1); then the clock, read under the lock (3.3), and
// the ending, by himself: the workspace's pending invitation to his
// address, his membership, then his memberships of its projects
// (membershipEnd). Were he the only admin of a project with other members,
// project.sole_admin (3.7 rule 2), and the whole leaving rolls back.
func (u *LeaveWorkspace) Execute(ctx context.Context, slug string) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	return u.tx.WithinTx(ctx, func(ctx context.Context) error {
		id, grant, err := lockAndDecide(ctx, u.workspaces.LockWorkspaceBySlug, u.auth, actor, slug, domain.ActionLeave)
		if err != nil {
			return err
		}
		if grant.WorkspaceRole == shared.RoleAdmin {
			other, err := u.workspaces.HasOtherAdmin(ctx, id, actor.UserID)
			switch {
			case err != nil:
				return err
			case !other:
				return domain.ErrSoleAdmin
			}
		}
		return u.end.run(ctx, id, actor.UserID, actor.UserID, u.clock.Now())
	})
}
````

`server/internal/modules/workspace/app/fakes_members_test.go`（修改，1 处）：

````old server/internal/modules/workspace/app/fakes_members_test.go
	return fmt.Errorf("end workspace member %s of %s: no such row", userID, workspaceID)
}

````
````new server/internal/modules/workspace/app/fakes_members_test.go
	return fmt.Errorf("end workspace member %s of %s: no such row", userID, workspaceID)
}

// HasOtherAdmin answers whether the workspace's memberships it holds have
// an active admin other than userID.
func (f *fakeWorkspaces) HasOtherAdmin(ctx context.Context, workspaceID, userID uuid.UUID) (bool, error) {
	f.log.add(ctx, "HasOtherAdmin %s %s", workspaceID, userID)
	if err := f.endErrs["HasOtherAdmin"]; err != nil {
		return false, fmt.Errorf("look for another admin: %w", err)
	}
	return slices.ContainsFunc(f.memberships[workspaceID], func(m domain.Membership) bool {
		return m.MemberID != userID && m.Role == shared.RoleAdmin && m.IsActive
	}), nil
}

````

`server/internal/modules/workspace/app/fakes_workspaces_test.go`（修改，1 处）：

````old server/internal/modules/workspace/app/fakes_workspaces_test.go
	endErrs     map[string]error                  // by method, for DeletePendingInvitations and EndMember
````
````new server/internal/modules/workspace/app/fakes_workspaces_test.go
	endErrs     map[string]error                  // by method, for HasOtherAdmin, DeletePendingInvitations and EndMember
````

`server/internal/modules/workspace/app/leave_workspace_test.go`（新文件，168 行）：

````file server/internal/modules/workspace/app/leave_workspace_test.go
package app_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// newLeave is LeaveWorkspace over membersFixture's fakes: in acme alice is
// the admin, bob a member, carol's membership has ended; in beta bob is a
// guest.
func newLeave() (*app.LeaveWorkspace, *membersFixture, *fakeTx) {
	f := newMembers()
	tx := &fakeTx{}
	return app.NewLeaveWorkspace(f.workspaces, f.profiles, f.projects, f.auth, tx, clockAt{at: clockNow}), f, tx
}

// leavingCalls are the calls up to the decision on user's leaving of w: the
// workspace's lock by its slug, the decision on workspace.leave.
func leavingCalls(user app.AccountState, w domain.Workspace) []string {
	return lockedDecision(user, w, "LockWorkspaceBySlug", domain.ActionLeave)
}

// otherAdmin is the call that asks whether w has an active admin other than
// user.
func otherAdmin(user app.AccountState, w domain.Workspace) string {
	return "HasOtherAdmin " + w.ID.String() + " " + user.ID.String()
}

// The caller leaves: the workspace locked FOR NO KEY UPDATE by its slug, the
// decision, then the ending of his own membership, by himself at the
// clock's one time, all in one transaction (M3 design 3.6, 3.8). A member
// and a guest are asked nothing more; an admin leaves when the workspace
// has another active admin: dave, made acme's admin for the case.
func TestLeaveWorkspaceLocksDecidesThenEnds(t *testing.T) {
	dave := app.AccountState{ID: uuid.NewV7(), Email: "dave@corp.com", Active: true}
	tests := []struct {
		name string
		user app.AccountState
		w    domain.Workspace
		set  func(f *membersFixture)
		want []string
	}{
		{"a member", bob, acme, nil, slices.Concat(leavingCalls(bob, acme), endingCalls(acme.ID, bob, bob.ID))},
		{"a guest", bob, beta, nil, slices.Concat(leavingCalls(bob, beta), endingCalls(beta.ID, bob, bob.ID))},
		{"an admin, beside another", alice, acme, func(f *membersFixture) {
			f.workspaces.memberships[acme.ID] = append(f.workspaces.memberships[acme.ID],
				domain.Membership{ID: uuid.NewV7(), WorkspaceID: acme.ID, MemberID: dave.ID, Role: shared.RoleAdmin, IsActive: true})
		}, slices.Concat(leavingCalls(alice, acme), []string{otherAdmin(alice, acme)}, endingCalls(acme.ID, alice, alice.ID))},
	}
	for _, tt := range tests {
		uc, f, tx := newLeave()
		if tt.set != nil {
			tt.set(f)
		}
		err := uc.Execute(as(tt.user), tt.w.Slug)
		if err != nil || !slices.Equal(f.log.calls, tt.want) || tx.calls != 1 {
			t.Errorf("%s: Execute() = %v, calls\n%q\nin %d transactions; want nil,\n%q\nin one", tt.name, err, f.log.calls, tx.calls, tt.want)
		}
	}
}

// Each refusal is the answer, and nothing is ended. A workspace not there,
// deleted while the lock waited, or of which the caller is not an active
// member is workspace.not_found; acme's only active admin is
// workspace.sole_admin, beside a member or alone (M3 design 3.7 rule 1),
// asked after the decision; a failure is never a 404.
func TestLeaveWorkspaceRefusals(t *testing.T) {
	failure := errors.New("connection reset")
	tests := []struct {
		name  string
		user  app.AccountState
		slug  string
		set   func(f *membersFixture)
		want  error
		calls []string
	}{
		{"no such workspace", bob, "gone", nil, domain.ErrNotFound, []string{"LockWorkspaceBySlug gone"}},
		{"deleted while the lock waited", bob, "acme", func(f *membersFixture) { f.workspaces.lockErrs = map[string]error{"acme": app.ErrNotFound} },
			domain.ErrNotFound, []string{"LockWorkspaceBySlug acme"}},
		{"not a member", carol, "acme", nil, domain.ErrNotFound, leavingCalls(carol, acme)},
		{"the only admin", alice, "acme", nil, domain.ErrSoleAdmin, append(leavingCalls(alice, acme), otherAdmin(alice, acme))},
		{"the only admin, alone", alice, "acme",
			func(f *membersFixture) { f.workspaces.memberships[acme.ID] = []domain.Membership{aliceInAcme} }, domain.ErrSoleAdmin,
			append(leavingCalls(alice, acme), otherAdmin(alice, acme))},
		{"the lock failed", bob, "acme", func(f *membersFixture) { f.workspaces.lockErrs = map[string]error{"acme": failure} }, failure,
			[]string{"LockWorkspaceBySlug acme"}},
		{"the Authorizer failed", bob, "acme", func(f *membersFixture) { f.auth.errs = map[grantKey]error{{bob.ID, acme.ID}: failure} }, failure,
			leavingCalls(bob, acme)},
		{"the question of another admin failed", alice, "acme",
			func(f *membersFixture) { f.workspaces.endErrs = map[string]error{"HasOtherAdmin": failure} }, failure,
			append(leavingCalls(alice, acme), otherAdmin(alice, acme))},
	}
	for _, tt := range tests {
		uc, f, tx := newLeave()
		if tt.set != nil {
			tt.set(f)
		}
		err := uc.Execute(as(tt.user), tt.slug)
		if !errors.Is(err, tt.want) {
			t.Errorf("%s: Execute() = %v; want %v", tt.name, err, tt.want)
		}
		if tt.want == failure && errors.Is(err, domain.ErrNotFound) {
			t.Errorf("%s: Execute() = %v, which is also workspace.not_found", tt.name, err)
		}
		if !slices.Equal(f.log.calls, tt.calls) || tx.calls != 1 {
			t.Errorf("%s: calls = %q in %d transactions, want %q in one", tt.name, f.log.calls, tx.calls, tt.calls)
		}
	}
	uc, f, tx := newLeave()
	if err := uc.Execute(context.Background(), "acme"); !errors.Is(err, shared.Unauthenticated()) || len(f.log.calls) != 0 || tx.calls != 0 {
		t.Errorf("Execute() without an actor = %v, calls %q in %d transactions; want 401 unauthorized and no call", err, f.log.calls, tx.calls)
	}
}

// A failed read of the leaver's address, a member without an account, a
// failed step of the ending, the projects' refusal of the only admin of a
// project with other members, and a refused commit each fail the
// transaction: the answer is the error as it came, project.sole_admin
// itself (M3 design 3.7 rule 2); the calls are the leaving's own, each
// once, up to the failing one: nothing runs after it.
func TestLeaveWorkspaceFailsWithinTheTransaction(t *testing.T) {
	failure := errors.New("connection reset")
	soleAdmin := shared.NewError(shared.KindConflict, "project.sole_admin", "Ending the membership would leave a project without an admin.")
	calls := slices.Concat(leavingCalls(bob, acme), endingCalls(acme.ID, bob, bob.ID))
	decided := len(leavingCalls(bob, acme))
	tests := []struct {
		name  string
		set   func(f *membersFixture, tx *fakeTx)
		want  error // nil for the use case's own, which is no *shared.Error
		calls []string
	}{
		{"the address", func(f *membersFixture, _ *fakeTx) { f.profiles.err = failure }, failure, calls[:decided+1]},
		{"a member without an account", func(f *membersFixture, _ *fakeTx) { f.profiles.profiles = profiles[:2] }, nil, calls[:decided+1]},
		{"the invitations", func(f *membersFixture, _ *fakeTx) {
			f.workspaces.endErrs = map[string]error{"DeletePendingInvitations": failure}
		},
			failure, calls[:decided+2]},
		{"the membership", func(f *membersFixture, _ *fakeTx) { f.workspaces.endErrs = map[string]error{"EndMember": failure} }, failure,
			calls[:decided+3]},
		{"the projects' step", func(f *membersFixture, _ *fakeTx) { f.projects.errs = map[string]error{"EndMemberships": failure} }, failure,
			calls},
		{"the only admin of a project", func(f *membersFixture, _ *fakeTx) { f.projects.errs = map[string]error{"EndMemberships": soleAdmin} },
			soleAdmin, calls},
		{"the commit", func(_ *membersFixture, tx *fakeTx) { tx.commitErr = failure }, failure, calls},
	}
	for _, tt := range tests {
		uc, f, tx := newLeave()
		tt.set(f, tx)
		err := uc.Execute(as(bob), "acme")
		var se *shared.Error
		switch {
		case tt.want == nil && (err == nil || errors.As(err, &se)):
			t.Errorf("%s failing: Execute() = %v; want an error that is no *shared.Error", tt.name, err)
		case tt.want != nil && !errors.Is(err, tt.want):
			t.Errorf("%s failing: Execute() = %v, want %v", tt.name, err, tt.want)
		}
		if !slices.Equal(f.log.calls, tt.calls) || tx.calls != 1 {
			t.Errorf("%s failing: calls\n%q\nin %d transactions; want\n%q\nin one", tt.name, f.log.calls, tx.calls, tt.calls)
		}
	}
}
````

`server/internal/modules/workspace/app/clock_test.go`（修改，2 处）：

````old server/internal/modules/workspace/app/clock_test.go
// projects' step, and a removal for each step of the ending. The clock
// logs its read among the fakes' calls.
````
````new server/internal/modules/workspace/app/clock_test.go
// projects' step, and a removal and a leaving for each step of the ending.
// The clock logs its read among the fakes' calls.
````

````old server/internal/modules/workspace/app/clock_test.go
		}, slices.Concat(removalCalls(alice, bobInAcme), []string{"Now"}, endingCalls(acme.ID, bob, alice.ID))},
````
````new server/internal/modules/workspace/app/clock_test.go
		}, slices.Concat(removalCalls(alice, bobInAcme), []string{"Now"}, endingCalls(acme.ID, bob, alice.ID))},
		{"leaveWorkspace", func() ([]string, error) {
			f := newMembers()
			err := app.NewLeaveWorkspace(f.workspaces, f.profiles, f.projects, f.auth, &fakeTx{}, clockAt{clockNow, f.log}).Execute(as(bob), "acme")
			return f.log.calls, err
		}, slices.Concat(leavingCalls(bob, acme), []string{"Now"}, endingCalls(acme.ID, bob, bob.ID))},
````

- [ ] **Step 3: handler、接线、文案**

`server/internal/modules/workspace/adapter/http/handler.go`（修改，2 处）：

````old server/internal/modules/workspace/adapter/http/handler.go
type DeleteWorkspaceUseCase interface {
````
````new server/internal/modules/workspace/adapter/http/handler.go
type DeleteWorkspaceUseCase interface {
	Execute(ctx context.Context, slug string) error
}

// LeaveUseCase is app.LeaveWorkspace.
type LeaveUseCase interface {
````

````old server/internal/modules/workspace/adapter/http/handler.go
	DeleteWorkspace   DeleteWorkspaceUseCase
````
````new server/internal/modules/workspace/adapter/http/handler.go
	DeleteWorkspace   DeleteWorkspaceUseCase
	Leave             LeaveUseCase
````

`server/internal/modules/workspace/adapter/http/members.go`（修改，1 处）：

````old server/internal/modules/workspace/adapter/http/members.go
	return out, nil
````
````new server/internal/modules/workspace/adapter/http/members.go
	return out, nil
}

// LeaveWorkspace serves POST /api/v0/workspaces/{slug}/leave.
func (h handler) LeaveWorkspace(ctx context.Context, req gen.LeaveWorkspaceRequestObject) (gen.LeaveWorkspaceResponseObject, error) {
	if err := h.uc.Leave.Execute(ctx, req.Slug); err != nil {
		return nil, err
	}
	return gen.LeaveWorkspace204Response{}, nil
````

`server/internal/modules/workspace/adapter/http/handler_test.go`（修改，4 处）：

````old server/internal/modules/workspace/adapter/http/handler_test.go
	del    *fakeDelete
````
````new server/internal/modules/workspace/adapter/http/handler_test.go
	del    *fakeDelete
	leave  *fakeLeave
````

````old server/internal/modules/workspace/adapter/http/handler_test.go
func (f *fakeDelete) Execute(ctx context.Context, slug string) error {
````
````new server/internal/modules/workspace/adapter/http/handler_test.go
func (f *fakeDelete) Execute(ctx context.Context, slug string) error {
	f.calls = append(f.calls, caller(ctx)+" "+slug)
	return f.err
}

type fakeLeave struct {
	calls []string // "caller slug"
	err   error
}

func (f *fakeLeave) Execute(ctx context.Context, slug string) error {
````

````old server/internal/modules/workspace/adapter/http/handler_test.go
		f.del = &fakeDelete{}
	}
````
````new server/internal/modules/workspace/adapter/http/handler_test.go
		f.del = &fakeDelete{}
	}
	if f.leave == nil {
		f.leave = &fakeLeave{}
	}
````

````old server/internal/modules/workspace/adapter/http/handler_test.go
		ListWorkspaces: f.list, CreateWorkspace: f.create, GetWorkspace: f.get, UpdateWorkspace: f.update, DeleteWorkspace: f.del, CheckSlug: f.check,
````
````new server/internal/modules/workspace/adapter/http/handler_test.go
		ListWorkspaces: f.list, CreateWorkspace: f.create, GetWorkspace: f.get, UpdateWorkspace: f.update, DeleteWorkspace: f.del, Leave: f.leave,
		CheckSlug:   f.check,
````

`server/internal/modules/workspace/adapter/http/members_test.go`（修改，1 处）：

````old server/internal/modules/workspace/adapter/http/members_test.go
		t.Errorf("DELETE of no membership id = %d, calls %q; want 400 and no call", res.StatusCode, remove.calls)
	}
}

````
````new server/internal/modules/workspace/adapter/http/members_test.go
		t.Errorf("DELETE of no membership id = %d, calls %q; want 400 and no call", res.StatusCode, remove.calls)
	}
}

// POST …/leave ends the caller's own membership of the workspace of the
// path and answers 204 with no body.
func TestLeaveWorkspace(t *testing.T) {
	leave := &fakeLeave{}
	h := newServer(t, fakes{leave: leave})
	for _, tt := range []struct{ token, slug string }{{"alice", "acme"}, {"bob", "beta"}} {
		if res, body := do(t, h, request(http.MethodPost, "/api/v0/workspaces/"+tt.slug+"/leave", tt.token, "")); res.StatusCode != http.StatusNoContent ||
			body != "" {
			t.Errorf("%s's leaving %s = %d %q, want 204 and no body", tt.token, tt.slug, res.StatusCode, body)
		}
	}
	if want := []string{"alice acme", "bob beta"}; !slices.Equal(leave.calls, want) {
		t.Errorf("calls = %q, want %q", leave.calls, want)
	}
}

// The use case's refusals, as the contract declares them: the project
// module's project.sole_admin comes through as it is (M3 design 9.4).
func TestLeaveWorkspaceRefusals(t *testing.T) {
	soleAdmin := shared.NewError(shared.KindConflict, "project.sole_admin", "The member is the only admin of a project.")
	tests := []struct {
		err    error
		status int
		want   string
	}{
		{domain.ErrNotFound, http.StatusNotFound,
			`{"status":404,"code":"workspace.not_found","title":"Not Found","detail":"The workspace does not exist, or you are not a member of it."}`},
		{domain.ErrSoleAdmin, http.StatusConflict, `{"status":409,"code":"workspace.sole_admin","title":"Conflict",` +
			`"detail":"The workspace would be left without an admin; make another member an admin first."}`},
		{fmt.Errorf("end the member's project memberships: %w", soleAdmin), http.StatusConflict,
			`{"status":409,"code":"project.sole_admin","title":"Conflict","detail":"The member is the only admin of a project."}`},
	}
	for _, tt := range tests {
		h := newServer(t, fakes{leave: &fakeLeave{err: tt.err}})
		res, body := do(t, h, request(http.MethodPost, "/api/v0/workspaces/acme/leave", "alice", ""))
		if res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("leaving refused with %v = %d %s, want %d %s", tt.err, res.StatusCode, body, tt.status, tt.want)
		}
	}
}

````

`server/internal/modules/workspace/module.go`（修改，2 处）：

````old server/internal/modules/workspace/module.go
// listing, reading, changing and deleting workspaces, checking a slug,
// listing the members, changing their roles and removing them, each
````
````new server/internal/modules/workspace/module.go
// listing, reading, changing, deleting and leaving workspaces, checking a
// slug, listing the members, changing their roles and removing them, each
````

````old server/internal/modules/workspace/module.go
		DeleteWorkspace:   app.NewDeleteWorkspace(store, d.Projects, d.Authorizer, d.Tx, d.Clock, d.Logger),
````
````new server/internal/modules/workspace/module.go
		DeleteWorkspace:   app.NewDeleteWorkspace(store, d.Projects, d.Authorizer, d.Tx, d.Clock, d.Logger),
		Leave:             app.NewLeaveWorkspace(store, d.Profiles, d.Projects, d.Authorizer, d.Tx, d.Clock),
````

`web/apps/web/helpers/authentication.helper.ts`（修改，1 处）：

````old web/apps/web/helpers/authentication.helper.ts
  "workspace.own_membership": "auth.errors.workspace_own_membership",
````
````new web/apps/web/helpers/authentication.helper.ts
  "workspace.own_membership": "auth.errors.workspace_own_membership",
  "workspace.sole_admin": "auth.errors.workspace_sole_admin",
````

`web/packages/i18n/src/locales/en/auth.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/auth.json
      "workspace_own_membership": "You cannot change your own membership.",
````
````new web/packages/i18n/src/locales/en/auth.json
      "workspace_own_membership": "You cannot change your own membership.",
      "workspace_sole_admin": "The workspace would be left without an admin. Make another member an admin first.",
````

`web/packages/i18n/src/locales/zh-CN/auth.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/auth.json
      "workspace_own_membership": "不能修改自己的成员身份。",
````
````new web/packages/i18n/src/locales/zh-CN/auth.json
      "workspace_own_membership": "不能修改自己的成员身份。",
      "workspace_sole_admin": "工作区会因此没有管理员。请先把另一位成员设为管理员。",
````

- [ ] **Step 4: 矩阵、完整性核对、组合出的结束**

`server/internal/bootstrap/permission_matrix_test.go`（修改，3 处）：

````old server/internal/bootstrap/permission_matrix_test.go
var workspaceColumns = []caller{callerAdmin, callerMember, callerGuest, callerNever, callerRemoved, callerDeleted}

````
````new server/internal/bootstrap/permission_matrix_test.go
var workspaceColumns = []caller{callerAdmin, callerMember, callerGuest, callerNever, callerRemoved, callerDeleted}

// callerSoleAdmin is the column of a workspace's only active admin, for the
// cells of 9.2 that acme, with two admins, cannot give (P4a review §6):
// other's admin, the account never a member of acme, beside other's member,
// the removed member.
const callerSoleAdmin caller = "the only admin of other"

// soleAdminColumns are the columns of the only admin's table: his alone.
var soleAdminColumns = []caller{callerSoleAdmin}

````

````old server/internal/bootstrap/permission_matrix_test.go
// prepared workspace, or the deleted one its caller was the admin of.
````
````new server/internal/bootstrap/permission_matrix_test.go
// prepared workspace; the deleted one its caller was the admin of; other,
// for its only admin.
````

````old server/internal/bootstrap/permission_matrix_test.go
	if c == callerDeleted {
		return "gone"
````
````new server/internal/bootstrap/permission_matrix_test.go
	switch c {
	case callerDeleted:
		return "gone"
	case callerSoleAdmin:
		return "other"
````

`server/internal/bootstrap/permission_matrix_workspace_test.go`（修改，2 处）：

````old server/internal/bootstrap/permission_matrix_workspace_test.go
	cellValidationFailed  = cell{http.StatusUnprocessableEntity, "validation_failed"}
````
````new server/internal/bootstrap/permission_matrix_workspace_test.go
	cellValidationFailed  = cell{http.StatusUnprocessableEntity, "validation_failed"}
	cellSoleAdmin         = cell{http.StatusConflict, "workspace.sole_admin"}
````

````old server/internal/bootstrap/permission_matrix_workspace_test.go
			check: listsTheMembers},
````
````new server/internal/bootstrap/permission_matrix_workspace_test.go
			check: listsTheMembers},
		// Every active member leaves; acme's admin beside PM+WA, its other
		// admin (M3 design 9.2). other's only admin is refused, beside its
		// member, as he would be alone (3.7 rule 1).
		{op: "leaveWorkspace", write: true, request: toWorkspace(http.MethodPost, "/leave", ""),
			cells: inWorkspace(cellNoContent, cellNoContent, cellNoContent)},
		{op: "leaveWorkspace", variant: "the only admin", write: true, columns: soleAdminColumns, request: toWorkspace(http.MethodPost, "/leave", ""),
			cells: map[caller]cell{callerSoleAdmin: cellSoleAdmin}},
````

`server/internal/bootstrap/permission_matrix_columns_test.go`（修改，4 处）：

````old server/internal/bootstrap/permission_matrix_columns_test.go
	case callerArchivedNever:
````
````new server/internal/bootstrap/permission_matrix_columns_test.go
	case callerArchivedNever, callerSoleAdmin:
````

````old server/internal/bootstrap/permission_matrix_columns_test.go
	for _, c := range slices.Concat(workspaceColumns, projectColumns, archivedColumns) {
````
````new server/internal/bootstrap/permission_matrix_columns_test.go
	for _, c := range slices.Concat(workspaceColumns, projectColumns, archivedColumns, soleAdminColumns) {
````

````old server/internal/bootstrap/permission_matrix_columns_test.go
				want = append(want, fmt.Sprintf("row getProject, %s: {project_id} from a column of no project table (matrixTables): "+
````
````new server/internal/bootstrap/permission_matrix_columns_test.go
				want = append(want, fmt.Sprintf("row getProject, %s: {project_id} from a column of no project table (projectTables): "+
````

````old server/internal/bootstrap/permission_matrix_columns_test.go
			return want
		}()},
		{"a {slug} of another column's workspace beside a listed {identifier}", listed, []matrixRow{row, func() matrixRow {
````
````new server/internal/bootstrap/permission_matrix_columns_test.go
			return want
		}()},
		// The only admin's table is a table a row may name, and no project
		// table: a project's operation in it is the same gap.
		{"a project operation in the only admin's row", listed, []matrixRow{with(func(r *matrixRow) {
			r.columns, r.cells = soleAdminColumns, map[caller]cell{callerSoleAdmin: cellOK}
		}), checks}, []string{fmt.Sprintf("row getProject, %s: {project_id} from a column of no project table (projectTables): "+
			"a project's row names its columns", callerSoleAdmin)}},
		{"a {slug} of another column's workspace beside a listed {identifier}", listed, []matrixRow{row, func() matrixRow {
````

`server/internal/bootstrap/permission_matrix_targets_test.go`（修改，2 处）：

````old server/internal/bootstrap/permission_matrix_targets_test.go
// column of a project table (matrixTables): a project's operation in a
// workspace-level row would leave the project level's own columns unasked.
````
````new server/internal/bootstrap/permission_matrix_targets_test.go
// column of a project table (projectTables): a project's operation in a
// workspace-level row, or the only admin's, would leave the project level's
// own columns unasked.
````

````old server/internal/bootstrap/permission_matrix_targets_test.go
			if !slices.ContainsFunc(matrixTables, func(table []caller) bool { return slices.Contains(table, c) }) {
				return "{project_id} from a column of no project table (matrixTables): a project's row names its columns"
````
````new server/internal/bootstrap/permission_matrix_targets_test.go
			if !slices.ContainsFunc(projectTables, func(table []caller) bool { return slices.Contains(table, c) }) {
				return "{project_id} from a column of no project table (projectTables): a project's row names its columns"
````

`server/internal/bootstrap/permission_matrix_coverage_test.go`（修改，1 处）：

````old server/internal/bootstrap/permission_matrix_coverage_test.go
)

// matrixTables are the columns a row may name besides the workspace level's
// (nil): each table of M3 design 9.2, whole.
var matrixTables = [][]caller{projectColumns, archivedColumns}
````
````new server/internal/bootstrap/permission_matrix_coverage_test.go
)

// projectTables are the tables of the project level (M3 design 9.2), each
// whole: a row that names a project names one of them.
var projectTables = [][]caller{projectColumns, archivedColumns}

// matrixTables are the columns a row may name besides the workspace level's
// (nil): each table of the project level, and the only admin's.
var matrixTables = append(slices.Clone(projectTables), soleAdminColumns)
````

`server/internal/bootstrap/permission_matrix_seed_test.go`（修改，1 处）：

````old server/internal/bootstrap/permission_matrix_seed_test.go
		s.t.Fatalf("the removed member's facts of %s = %+v, %v, %v; want him its active member", projectOf(callerRemoved), f, found, err)
	}
````
````new server/internal/bootstrap/permission_matrix_seed_test.go
		s.t.Fatalf("the removed member's facts of %s = %+v, %v, %v; want him its active member", projectOf(callerRemoved), f, found, err)
	}
	// other's admin is its only active admin, beside its active member, so
	// that his leaving's 409 is the rule's (3.7 rule 1); acme's admin has
	// another, PM+WA, so that his leaving's 204 is no other case.
	for _, tt := range []struct {
		slug  string
		c     caller
		other bool
	}{{"other", callerNever, false}, {"acme", callerAdmin, true}} {
		role, active, err := s.matrixSeed.store.ActiveRole(ctx, sd.workspace(tt.slug), s.ids[tt.c])
		if err != nil || !active || role != shared.RoleAdmin {
			s.t.Fatalf("%s's role in %s = %d, %v, %v; want its active admin", tt.c, tt.slug, role, active, err)
		}
		if other, err := s.matrixSeed.store.HasOtherAdmin(ctx, sd.workspace(tt.slug), s.ids[tt.c]); err != nil || other != tt.other {
			s.t.Fatalf("another admin of %s than %s: %v, %v; want %v", tt.slug, tt.c, other, err, tt.other)
		}
	}
	if role, active, err := s.matrixSeed.store.ActiveRole(ctx, sd.workspace("other"), s.ids[callerRemoved]); err != nil || !active ||
		role != shared.RoleMember {
		s.t.Fatalf("the removed member's role in other = %d, %v, %v; want its active member", role, active, err)
	}
````

`server/internal/bootstrap/project_write_locks_test.go`（修改，5 处）：

````old server/internal/bootstrap/project_write_locks_test.go
// level (one of no workspace's, any table of them). The second takes in a
// write on a project addressed by a row under it (P5's
// /project-members/{project_member_id}, P7's /states/{state_id}), and one
// whose rows ask a column set of their own (a self-only leaveProject).
func writesOnAProject(ops []apitest.Operation, rows []matrixRow) []string {
````
````new server/internal/bootstrap/project_write_locks_test.go
// level (a column of a project table, projectTables, that is no column of
// the workspace level). The second takes in a write on a project addressed
// by a row under it (P5's /project-members/{project_member_id}, P7's
// /states/{state_id}), and one whose rows ask a column set of their own (a
// self-only leaveProject); not a workspace's write that the only admin's
// table asks (leaveWorkspace).
func writesOnAProject(ops []apitest.Operation, rows []matrixRow) []string {
	ofTheProjectLevel := func(c caller) bool {
		return !slices.Contains(workspaceColumns, c) && slices.ContainsFunc(projectTables, func(table []caller) bool { return slices.Contains(table, c) })
	}
````

````old server/internal/bootstrap/project_write_locks_test.go
		if slices.ContainsFunc(r.columns, func(c caller) bool { return !slices.Contains(workspaceColumns, c) }) {
````
````new server/internal/bootstrap/project_write_locks_test.go
		if slices.ContainsFunc(r.columns, ofTheProjectLevel) {
````

````old server/internal/bootstrap/project_write_locks_test.go
// own; not a read of a project, nor a write at the workspace level.
````
````new server/internal/bootstrap/project_write_locks_test.go
// own; not a read of a project, nor a write at the workspace level, also
// when the only admin's table asks it.
````

````old server/internal/bootstrap/project_write_locks_test.go
		{ID: "updateProjectMember", Method: http.MethodPatch, Path: "/api/v0/project-members/{project_member_id}"},
````
````new server/internal/bootstrap/project_write_locks_test.go
		{ID: "updateProjectMember", Method: http.MethodPatch, Path: "/api/v0/project-members/{project_member_id}"},
		{ID: "leaveWorkspace", Method: http.MethodPost, Path: "/api/v0/workspaces/{slug}/leave"},
````

````old server/internal/bootstrap/project_write_locks_test.go
		{op: "updateProjectMember", write: true, columns: []caller{callerProjectAdmin, callerProjectMember}},
````
````new server/internal/bootstrap/project_write_locks_test.go
		{op: "updateProjectMember", write: true, columns: []caller{callerProjectAdmin, callerProjectMember}},
		{op: "leaveWorkspace", write: true, columns: soleAdminColumns},
````

`server/internal/bootstrap/removal_test.go`（删除）：

````delete server/internal/bootstrap/removal_test.go
````

`server/internal/bootstrap/ending_test.go`（新文件，351 行）：

````file server/internal/bootstrap/ending_test.go
package bootstrap

import (
	"context"
	"encoding/json"
	"maps"
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// endingWorld is the wired app on a database of its own: acme, alice its
// admin; bob, carol and dave its members, each by accepting alice's
// invitation; Web, alice's, led by bob, so both are its admins, and carol
// its member by joining; Ops, bob's, his alone to administer, carol its
// member by his adding; beta, alice its admin, bob its member. Then,
// through the workspace store, the invitations no operation makes, 3.8
// refusing to invite an active member: one pending to bob's address in
// acme, and one in beta; one pending to carol's in acme; one to dave's in
// acme that he declined.
type endingWorld struct {
	contract       *apitest.Contract
	base           string
	pool           *pgxpool.Pool
	tokens         map[string]string    // access tokens, by name
	ids            map[string]uuid.UUID // accounts, by name
	web, ops       uuid.UUID
	bobsInvitation uuid.UUID // the pending one to bob's address in acme
	davesDeclined  uuid.UUID
}

func newEndingWorld(t *testing.T) endingWorld {
	t.Helper()
	contract := apitest.Load(t)
	dbURL := pgtest.NewDatabase(t)
	w := endingWorld{contract: contract, base: startApp(t, testConfig(t, dbURL, false), migrations.FS()), pool: openPool(t, dbURL),
		tokens: map[string]string{}, ids: map[string]uuid.UUID{}}
	for _, name := range []string{"alice", "bob", "carol", "dave"} {
		w.tokens[name] = registerAccount(t, contract, w.base, name+"@example.com").AccessToken
		w.ids[name] = accountID(t, contract, w.base, w.tokens[name])
	}
	for _, slug := range []string{"acme", "beta"} {
		if status, body := call(t, contract, http.MethodPost, w.base+"/api/v0/workspaces", w.tokens["alice"],
			`{"name":"`+slug+`","slug":"`+slug+`"}`); status != http.StatusCreated {
			t.Fatalf("creating %s = %d %s", slug, status, body)
		}
	}
	for _, m := range []struct{ slug, name string }{{"acme", "bob"}, {"acme", "carol"}, {"acme", "dave"}, {"beta", "bob"}} {
		answerInvitation(t, contract, w.base, w.tokens[m.name], "accept", invite(t, contract, w.base, w.tokens["alice"], m.slug, m.name+"@example.com"),
			http.StatusOK)
	}
	status, body := call(t, contract, http.MethodPost, w.base+"/api/v0/workspaces/acme/projects", w.tokens["alice"],
		`{"name":"Web","identifier":"WEB","project_lead_id":"`+w.ids["bob"].String()+`"}`)
	var web struct {
		ID uuid.UUID `json:"id"`
	}
	if status != http.StatusCreated {
		t.Fatalf("creating Web = %d %s", status, body)
	}
	decodeAnswer(t, body, &web)
	w.web, w.ops = web.ID, createdProject(t, contract, w.base, w.tokens["bob"], "acme", "Ops", "OPS")
	for _, step := range []struct{ token, path, body string }{
		{w.tokens["carol"], "/api/v0/projects/" + w.web.String() + "/join", ""},
		{w.tokens["bob"], "/api/v0/projects/" + w.ops.String() + "/members", `{"members":[{"member_id":"` + w.ids["carol"].String() + `","role":15}]}`},
	} {
		if status, body := call(t, contract, http.MethodPost, w.base+step.path, step.token, step.body); status != http.StatusOK &&
			status != http.StatusCreated {
			t.Fatalf("POST %s = %d %s", step.path, status, body)
		}
	}
	store, ctx := workspacepg.New(w.pool), context.Background()
	for _, inv := range []struct {
		slug, name string
		id         *uuid.UUID
	}{{"acme", "bob", &w.bobsInvitation}, {"beta", "bob", nil}, {"acme", "carol", nil}, {"acme", "dave", &w.davesDeclined}} {
		id := uuid.NewV7()
		if _, err := store.CreateInvitations(ctx, []workspaceapp.InvitationRow{{ID: id, WorkspaceID: w.workspace(t, inv.slug),
			Email: inv.name + "@example.com", Role: shared.RoleGuest, CreatedBy: w.ids["alice"], Now: time.Now()}}); err != nil {
			t.Fatal(err)
		}
		if inv.id != nil {
			*inv.id = id
		}
	}
	if err := store.DeclineInvitation(ctx, w.davesDeclined, w.ids["dave"], time.Now()); err != nil {
		t.Fatal(err)
	}
	w.bystanders(t)
	return w
}

// bystanders checks the rows an ending of bob's membership of acme must
// leave as they were, each read by what makes it the one it is: his active
// membership of beta; the pending invitations to his address in beta and
// to carol's in acme; the one to dave's in acme, declined. Were one
// missing, an ending that also wrote it would pass.
func (w endingWorld) bystanders(t *testing.T) {
	t.Helper()
	var got string
	if err := w.pool.QueryRow(context.Background(), `SELECT concat_ws(', ',
		(SELECT 'bob in beta' FROM workspace_members m JOIN workspaces w ON w.id = m.workspace_id
			WHERE w.slug = 'beta' AND m.member_id = $1 AND m.is_active AND m.deleted_at IS NULL),
		(SELECT string_agg(w.slug || ' ' || i.email || CASE WHEN i.responded_at IS NULL THEN ' pending' WHEN i.accepted THEN ' accepted'
			ELSE ' declined' END, ', ' ORDER BY w.slug, i.email)
			FROM workspace_member_invites i JOIN workspaces w ON w.id = i.workspace_id WHERE i.deleted_at IS NULL))`, w.ids["bob"]).
		Scan(&got); err != nil {
		t.Fatal(err)
	}
	if want := "bob in beta, acme bob@example.com pending, acme carol@example.com pending, acme dave@example.com declined, " +
		"beta bob@example.com pending"; got != want {
		t.Fatalf("the rows an ending leaves: %s; want %s", got, want)
	}
}

// workspace is the id of the workspace slug.
func (w endingWorld) workspace(t *testing.T, slug string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := w.pool.QueryRow(context.Background(), "SELECT id FROM workspaces WHERE slug = $1 AND deleted_at IS NULL", slug).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// membership is the id of name's membership of acme.
func (w endingWorld) membership(t *testing.T, name string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := w.pool.QueryRow(context.Background(), `SELECT m.id FROM workspace_members m JOIN workspaces w ON w.id = m.workspace_id
		WHERE w.slug = 'acme' AND m.member_id = $1 AND m.deleted_at IS NULL`, w.ids[name]).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// leave is name's leaving of the workspace slug: its status and body.
func (w endingWorld) leave(t *testing.T, slug, name string) (int, string) {
	t.Helper()
	return call(t, w.contract, http.MethodPost, w.base+"/api/v0/workspaces/"+slug+"/leave", w.tokens[name], "")
}

// rowJSON is the row id of table as JSON, its columns by name; nil when
// there is none.
func rowJSON(t *testing.T, pool *pgxpool.Pool, table string, id uuid.UUID) map[string]any {
	t.Helper()
	var text []byte
	if err := pool.QueryRow(context.Background(), "SELECT coalesce((SELECT row_to_json(r) FROM "+table+" r WHERE r.id = $1)::text, 'null')", id).
		Scan(&text); err != nil {
		t.Fatal(err)
	}
	var row map[string]any
	if err := json.Unmarshal(text, &row); err != nil {
		t.Fatal(err)
	}
	return row
}

// rowsBut is every table's rows as tableRows has them, River's left out,
// but the rows whose ids are in ids.
func rowsBut(t *testing.T, pool *pgxpool.Pool, ids []uuid.UUID) map[string]string {
	t.Helper()
	all := map[string]string{}
	for table := range tableRows(t, pool, riversOwn) {
		var text string
		if err := pool.QueryRow(context.Background(), `SELECT coalesce(string_agg(t, E'\n' ORDER BY t), '') FROM (SELECT row_to_json(r)::text AS t
			FROM `+table+` r WHERE NOT to_jsonb(r) ? 'id' OR (to_jsonb(r)->>'id') <> ALL ($1::text[])) s`, uuidTexts(ids)).Scan(&text); err != nil {
			t.Fatalf("%s: %v", table, err)
		}
		all[table] = text
	}
	return all
}

// uuidTexts are ids as text.
func uuidTexts(ids []uuid.UUID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id.String()
	}
	return out
}

// projectMemberships are the ids of user's memberships of the projects, in
// the order of projects.
func projectMemberships(t *testing.T, pool *pgxpool.Pool, user uuid.UUID, projects ...uuid.UUID) []uuid.UUID {
	t.Helper()
	ids := make([]uuid.UUID, len(projects))
	for i, p := range projects {
		if err := pool.QueryRow(context.Background(), "SELECT id FROM project_members WHERE project_id = $1 AND member_id = $2 AND deleted_at IS NULL",
			p, user).Scan(&ids[i]); err != nil {
			t.Fatal(err)
		}
	}
	return ids
}

// An ending of name's membership of acme on the wired app: alice's removal
// of him, or his leaving.
type ending struct {
	name string
	// request is the ending's method, its path, and the access token it is
	// sent with.
	request func(w endingWorld, t *testing.T, name string) (method, path, token string)
	by      func(name string) string // the account it writes as
}

// end sends e's request on w's app: its status and body.
func (e ending) end(w endingWorld, t *testing.T, name string) (int, string) {
	t.Helper()
	method, path, token := e.request(w, t, name)
	return call(t, w.contract, method, w.base+path, token, "")
}

var endings = []ending{
	{name: "removal", request: func(w endingWorld, t *testing.T, name string) (string, string, string) {
		return http.MethodDelete, "/api/v0/workspace-members/" + w.membership(t, name).String(), w.tokens["alice"]
	}, by: func(string) string { return "alice" }},
	{name: "leaving", request: func(w endingWorld, _ *testing.T, name string) (string, string, string) {
		return http.MethodPost, "/api/v0/workspaces/acme/leave", w.tokens[name]
	}, by: func(name string) string { return name }},
}

// A removal and a leaving each end a membership and leave no invitation,
// in one transaction at one moment (M3 design 3.6, 3.7 rule 2, 3.8, 9.3),
// on the wired app.
//   - bob is Ops's only admin and carol its member: the ending of his
//     membership is 409 project.sole_admin, and no row of any table
//     changes, the pending invitation to his address, which the ending
//     deletes before the projects' step refuses, still pending.
//   - Once alice, acme's admin, has joined Ops, as its admin, the ending
//     refused at its commit, after every statement ran, for each table it
//     writes, is 500, and no row changes: no step wrote in a transaction of
//     its own.
//   - Then it is 204: bob's membership of acme, of Web and of Ops ended,
//     each row kept with its role; the pending invitation to his address in
//     acme deleted; each by the ender, at one moment no earlier than the
//     request. Each of those rows was last written by dave before, so that
//     the claim of the ender's writing can fail. His membership of beta,
//     the invitation to him there, and every other row of every table are
//     as they were.
//   - Ending dave's membership leaves his declined invitation as it was.
func TestAnEndingEndsTheMembershipsAndLeavesNoInvitation(t *testing.T) {
	for _, e := range endings {
		t.Run(e.name, func(t *testing.T) {
			w := newEndingWorld(t)
			before := tableRows(t, w.pool, riversOwn)
			if status, body := e.end(w, t, "bob"); status != http.StatusConflict || problemCode(t, []byte(body)) != "project.sole_admin" {
				t.Fatalf("ending bob's membership, Ops's only admin = %d %s, want 409 project.sole_admin", status, body)
			}
			if after := tableRows(t, w.pool, riversOwn); !maps.Equal(after, before) {
				t.Errorf("the tables after the refused ending changed:\n%v\nwant them as they were:\n%v", after, before)
			}
			if status, body := call(t, w.contract, http.MethodPost, w.base+"/api/v0/projects/"+w.ops.String()+"/join", w.tokens["alice"],
				""); status != http.StatusOK {
				t.Fatalf("alice's joining Ops = %d %s", status, body)
			}
			before = tableRows(t, w.pool, riversOwn)
			for _, table := range []string{"workspace_member_invites", "workspace_members", "project_members"} {
				restore := refusingCommits(t, w.pool, table)
				status, body := e.end(w, t, "bob")
				restore()
				if after := tableRows(t, w.pool, riversOwn); status != http.StatusInternalServerError || !maps.Equal(after, before) {
					t.Errorf("the ending refused at its commit for %s = %d %s; want 500 and every table as it was", table, status, body)
				}
			}
			written := slices.Concat([]uuid.UUID{w.membership(t, "bob"), w.bobsInvitation}, projectMemberships(t, w.pool, w.ids["bob"], w.web, w.ops))
			tables := []string{"workspace_members", "workspace_member_invites", "project_members", "project_members"}
			rowsBefore := make([]map[string]any, len(written))
			for i, id := range written {
				if tag, err := w.pool.Exec(context.Background(), "UPDATE "+tables[i]+" SET updated_by_id = $2 WHERE id = $1", id,
					w.ids["dave"]); err != nil || tag.RowsAffected() != 1 {
					t.Fatalf("%s %s last written by dave: %v, %v", tables[i], id, tag, err)
				}
				rowsBefore[i] = rowJSON(t, w.pool, tables[i], id)
			}
			others := rowsBut(t, w.pool, written)
			started := time.Now()

			if status, body := e.end(w, t, "bob"); status != http.StatusNoContent {
				t.Fatalf("ending bob's membership = %d %s, want 204", status, body)
			}

			moment, _ := rowJSON(t, w.pool, "workspace_member_invites", w.bobsInvitation)["deleted_at"].(string)
			if at, err := time.Parse(time.RFC3339Nano, moment); err != nil || at.Before(started.Truncate(time.Microsecond)) {
				t.Errorf("the invitation deleted at %q (%v); want a moment no earlier than the request, %v", moment, err, started)
			}
			for i, id := range written {
				after := rowJSON(t, w.pool, tables[i], id)
				want := maps.Clone(rowsBefore[i])
				want["updated_at"], want["updated_by_id"] = moment, w.ids[e.by("bob")].String()
				if tables[i] == "workspace_member_invites" {
					want["deleted_at"] = moment
				} else {
					want["is_active"] = false
				}
				if !maps.Equal(after, want) {
					t.Errorf("%s %s after the ending:\n%v\nwant\n%v", tables[i], id, after, want)
				}
			}
			if after := rowsBut(t, w.pool, written); !maps.Equal(after, others) {
				t.Errorf("every other row after the ending:\n%v\nwant them as they were:\n%v", after, others)
			}
			declined := rowJSON(t, w.pool, "workspace_member_invites", w.davesDeclined)
			if status, body := e.end(w, t, "dave"); status != http.StatusNoContent {
				t.Fatalf("ending dave's membership = %d %s, want 204", status, body)
			}
			if after := rowJSON(t, w.pool, "workspace_member_invites", w.davesDeclined); !maps.Equal(after, declined) {
				t.Errorf("dave's declined invitation after the ending:\n%v\nwant it as it was:\n%v", after, declined)
			}
		})
	}
}

// A workspace's only active admin cannot leave it (M3 design 3.7 rule 1,
// 9.3): alice is refused 409 workspace.sole_admin in solo, where she is
// alone, and in acme, beside its members, and no row of any table changes;
// once carol is acme's admin too, alice leaves it.
func TestTheOnlyAdminCannotLeave(t *testing.T) {
	w := newEndingWorld(t)
	if status, body := call(t, w.contract, http.MethodPost, w.base+"/api/v0/workspaces", w.tokens["alice"],
		`{"name":"solo","slug":"solo"}`); status != http.StatusCreated {
		t.Fatalf("creating solo = %d %s", status, body)
	}
	before := tableRows(t, w.pool, riversOwn)
	for _, slug := range []string{"solo", "acme"} {
		if status, body := w.leave(t, slug, "alice"); status != http.StatusConflict || problemCode(t, []byte(body)) != "workspace.sole_admin" {
			t.Errorf("alice's leaving %s, its only admin = %d %s, want 409 workspace.sole_admin", slug, status, body)
		}
	}
	if after := tableRows(t, w.pool, riversOwn); !maps.Equal(after, before) {
		t.Errorf("the tables after the refused leavings changed:\n%v\nwant them as they were:\n%v", after, before)
	}
	if status, body := call(t, w.contract, http.MethodPatch, w.base+"/api/v0/workspace-members/"+w.membership(t, "carol").String(),
		w.tokens["alice"], `{"role":20}`); status != http.StatusOK {
		t.Fatalf("carol made acme's admin = %d %s", status, body)
	}
	if status, body := w.leave(t, "acme", "alice"); status != http.StatusNoContent {
		t.Errorf("alice's leaving acme, carol its admin too = %d %s, want 204", status, body)
	}
}
````

- [ ] **Step 5: 测试、lint、前端检查**

Run: `go -C server test -count=1 ./internal/modules/workspace/... ./internal/modules/access/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestPermissionMatrix$|TestThePermissionMatrixCoversEveryOperation|TestMatrixViolationsCatchesEach|TestEveryColumnCallsAsARegisteredAccount|TestWritesOnAProjectAreEachShape|TestEachWriteOnAProjectSharesItsWorkspaceFirst|TestAnEndingEndsTheMembershipsAndLeavesNoInvitation|TestTheOnlyAdminCannotLeave' ./internal/bootstrap/`
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
git add api/modules/workspace.yaml api/openapi.yaml server/internal/bootstrap/ending_test.go server/internal/bootstrap/permission_matrix_columns_test.go server/internal/bootstrap/permission_matrix_coverage_test.go server/internal/bootstrap/permission_matrix_seed_test.go server/internal/bootstrap/permission_matrix_targets_test.go server/internal/bootstrap/permission_matrix_test.go server/internal/bootstrap/permission_matrix_workspace_test.go server/internal/bootstrap/project_write_locks_test.go server/internal/bootstrap/removal_test.go server/internal/modules/access/domain/rules.go server/internal/modules/access/domain/rules_test.go server/internal/modules/workspace/adapter/http/handler.go server/internal/modules/workspace/adapter/http/handler_test.go server/internal/modules/workspace/adapter/http/members.go server/internal/modules/workspace/adapter/http/members_test.go server/internal/modules/workspace/app/clock_test.go server/internal/modules/workspace/app/fakes_members_test.go server/internal/modules/workspace/app/fakes_workspaces_test.go server/internal/modules/workspace/app/leave_workspace.go server/internal/modules/workspace/app/leave_workspace_test.go server/internal/modules/workspace/app/ports.go server/internal/modules/workspace/domain/actions.go server/internal/modules/workspace/domain/errors.go server/internal/modules/workspace/module.go web/apps/web/helpers/authentication.helper.ts web/packages/i18n/src/locales/en/auth.json web/packages/i18n/src/locales/zh-CN/auth.json api/dist/openapi.yaml server/internal/modules/workspace/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M3/P5a): leaveWorkspace; the only admin cannot leave

POST /api/v0/workspaces/{slug}/leave ends the caller's own membership
under the workspace's FOR NO KEY UPDATE; a workspace's only active
admin is refused with workspace.sole_admin, also when he is alone (M3
design 3.7 rule 1). The matrix gains the only admin's table. The
wired app's ending test runs for the removal and the leaving, each row
it claims the ender wrote last written by another before.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A）：

| 改坏 | 必须失败的测试 | 层 |
|---|---|---|
| 唯一的管理员照样离开 | `TestLeaveWorkspaceRefusals`、`TestTheOnlyAdminCannotLeave`、`TestPermissionMatrix`；W2、W7、W12（Task 13、14 起） | 单元；组合；端到端 |
| 不问管理员有没有另一位 | `TestTheOnlyAdminCannotLeave` | 组合 |
| 问另一个管理员失败时当作没有 | `TestLeaveWorkspaceRefusals` | 单元 |
| 判定成移出的操作；规则只给管理员和成员 | `TestPermissionMatrix` | 组合 |
| 结束一步的失败被吞掉 | `TestLeaveWorkspaceFailsWithinTheTransaction`、`TestAnEndingEndsTheMembershipsAndLeavesNoInvitation` | 单元；组合 |
| 时钟在锁之前读 | `TestEachWriteReadsTheClockUnderItsLock`；`TestEachLockOfAnEndingIsItsStrength`（Task 6 起） | 单元；组合 |
| `workspace.New` 的离开不带项目的连带、不开事务、用停在 2001 年的时钟 | `TestAnEndingEndsTheMembershipsAndLeavesNoInvitation` | 组合 |
| 矩阵的种子：other 再有一位管理员；被移出的成员不在 other；PM+WA 不是 acme 的管理员 | `TestPermissionMatrix/prepare`（前提） | 组合 |
| `endingWorld` 少了 beta 的邀请、carol 的邀请、bob 在 beta 的成员关系，或 dave 的邀请没有拒绝 | 用它的每个测试（`bystanders` 的前提） | 组合 |

**Done when:** 离开的两行矩阵（7 格）通过；规则 1 在组合出的 app 上有正反例；组合出的结束对移出和离开都通过。

---

### Task 6: 结束的竞争、锁的强度、事务的连接（组合）

**Files:**
- Create: `server/internal/bootstrap/ending_connection_test.go`、`server/internal/bootstrap/ending_races_test.go`
- Modify: `server/internal/bootstrap/ending_test.go`、`server/internal/bootstrap/interleaving_not_found_test.go`、`server/internal/bootstrap/project_connection_test.go`

**Interfaces:** 没有新的产品代码。组合一层的测试（brief 清扫 8、9、12、13；spec 2.8）：
- `ending_test.go`：`endingWorld.joinsOps`（alice 以管理员加入 Ops，bob 不再是它唯一的管理员）；`rowJSON` 经 `querier` 读（连接池，或看得到自己的写的事务）；`ending.notFound`（移出的 404 是 `workspace.member_not_found`，离开的是 `workspace.not_found`）。
- `project_connection_test.go`：`projectRoute` 改成 `moduleRoute`：`newRoute(t, register)` 挂上一个模块的路由，`newProjectRoute` 经它；`answerWithin(t, contract, method, path, caller, body, want)` 在 5 秒内拿到回答并核对契约；`poolOfOne(t, url)`。`interleaving_not_found_test.go` 的 `webWrite.send` 改收 `moduleRoute`。
- `ending_connection_test.go`：`newWorkspaceRoute(t, pool)`：`workspace.New` 在一个池上，`access` 的 `Authorizer`、`identity` 的 `Accounts` 和 `MemberProfiles`（经转换）、`project` 的连带，照 `bootstrap` 的接法，各在这个池上。

**Tests:**
- `ending_races_test.go`：`TestAnEndingFindsWhatEndedMeanwhile`（另一个事务持 acme 的 `FOR NO KEY UPDATE`，结束 bob 的成员关系、结束移出者 alice 的、或删除 acme；alice 移出 bob 或 bob 离开，已过认证，在 acme 的行上等（`WaitForLockWaitOn` 证明）；另一个提交之后：移出 404 `workspace.member_not_found`，离开 404 `workspace.not_found`，从不是泄露存在的 403；那一行是另一个事务留下的样子，其余每一行不变；五个情形）；`TestEachLockOfAnEndingIsItsStrength`（移出和离开各一次：别的事务持 acme 的 `FOR NO KEY UPDATE`，发给 bob 的待接受邀请、他在 acme 的成员关系、他在 Ops 的成员关系各 `FOR SHARE`，逐个放开，`lockOn` 每次读出各行最强的锁：先等 acme，什么都不持；再等邀请，只持 acme；再等成员关系，持 acme 和邀请的 `FOR NO KEY UPDATE`；再等 Ops 的成员关系，持 acme、邀请、成员关系、Web、Ops，各 `FOR NO KEY UPDATE`，不强也不弱；从不锁他不是成员的 Site；从不以 `FOR SHARE` 或更强锁他的账户（地址不加锁读，约定一）；然后 204，写下的时刻不早于 acme 被放开）。
- `ending_connection_test.go`：`TestTheEndingsRunOnTheirTransactionsConnection`（工作区模块和项目的连带在只有一个连接的池上：alice 移出 bob、alice 离开 acme，各在 5 秒内 204；任何一条语句经连接池而不是事务，就等一个永远不来的连接，请求在期限时失败）。
- `TestAnEndingEndsTheMembershipsAndLeavesNoInvitation` 改用 `joinsOps`；`TestTheWritesOnAProjectRunOnTheirTransactionsConnection` 改用 `moduleRoute`，内容不变。

- [ ] **Step 1: 共用的路由和结束的辅助**

`server/internal/bootstrap/project_connection_test.go`（修改，8 处）：

````old server/internal/bootstrap/project_connection_test.go
// projectRoute is the project module as bootstrap wires it (project.New),
// on pool, with auth as its Authorizer and members as its WorkspaceMembers,
````
````new server/internal/bootstrap/project_connection_test.go
// moduleRoute is a module as bootstrap wires it, on a pool of the test's,
````

````old server/internal/bootstrap/project_connection_test.go
type projectRoute struct {
````
````new server/internal/bootstrap/project_connection_test.go
type moduleRoute struct {
````

````old server/internal/bootstrap/project_connection_test.go
func newProjectRoute(t *testing.T, pool *pgxpool.Pool, auth shared.Authorizer, members projectapp.WorkspaceMembers) projectRoute {
	t.Helper()
	module := project.New(project.Deps{Pool: pool, Tx: postgres.NewTxManager(pool, 2*time.Second), Clock: clock.System{}, Authorizer: auth,
		Workspaces: projectWorkspaces{directory: workspace.Provide(pool).WorkspaceDirectory}, Members: members})
````
````new server/internal/bootstrap/project_connection_test.go
// newRoute mounts the routes register registers so.
func newRoute(t *testing.T, register func(*httpserver.Router, *httpserver.API)) moduleRoute {
	t.Helper()
````

````old server/internal/bootstrap/project_connection_test.go
	module.Register(router, api)
	return projectRoute{router: router}
````
````new server/internal/bootstrap/project_connection_test.go
	register(router, api)
	return moduleRoute{router: router}
}

// newProjectRoute is the project module (project.New) on pool, with auth as
// its Authorizer and members as its WorkspaceMembers.
func newProjectRoute(t *testing.T, pool *pgxpool.Pool, auth shared.Authorizer, members projectapp.WorkspaceMembers) moduleRoute {
	t.Helper()
	return newRoute(t, project.New(project.Deps{Pool: pool, Tx: postgres.NewTxManager(pool, 2*time.Second), Clock: clock.System{}, Authorizer: auth,
		Workspaces: projectWorkspaces{directory: workspace.Provide(pool).WorkspaceDirectory}, Members: members}).Register)
````

````old server/internal/bootstrap/project_connection_test.go
func (p projectRoute) send(method, path string, caller uuid.UUID, body string) (*http.Request, *httptest.ResponseRecorder) {
````
````new server/internal/bootstrap/project_connection_test.go
func (p moduleRoute) send(method, path string, caller uuid.UUID, body string) (*http.Request, *httptest.ResponseRecorder) {
````

````old server/internal/bootstrap/project_connection_test.go
	return req, rec
````
````new server/internal/bootstrap/project_connection_test.go
	return req, rec
}

// answerWithin sends method path as caller, with body when it is not empty,
// checks the answer against contract and that it is want, and returns its
// body. A route on a pool of one connection answers within 5 seconds, or a
// statement waits for a second connection without the request's deadline:
// the test fails.
func (p moduleRoute) answerWithin(t *testing.T, contract *apitest.Contract, method, path string, caller uuid.UUID, body string, want int) string {
	t.Helper()
	type answer struct {
		req *http.Request
		rec *httptest.ResponseRecorder
	}
	done := make(chan answer, 1)
	go func() {
		req, rec := p.send(method, path, caller, body)
		done <- answer{req, rec}
	}()
	var a answer
	select {
	case a = <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("%s %s did not answer within 5s: a statement waits for the pool's one connection without the request's deadline", method, path)
	}
	contract.CheckResponse(t, a.req, a.rec.Result())
	if a.rec.Code != want {
		t.Fatalf("%s %s = %d %s, want %d", method, path, a.rec.Code, a.rec.Body, want)
	}
	return a.rec.Body.String()
}

// poolOfOne is a pool of one connection to url's database. A statement on a
// context without a deadline would wait for its connection for ever, and so
// would closing the pool: the closing has a deadline of its own.
func poolOfOne(t *testing.T, url string) *pgxpool.Pool {
	t.Helper()
	one, err := postgres.NewPool(context.Background(), config.DatabaseConfig{URL: url, MaxConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		closed := make(chan struct{})
		go func() { one.Close(); close(closed) }()
		select {
		case <-closed:
		case <-time.After(5 * time.Second):
			t.Error("the pool of one connection did not close within 5s: a transaction still holds its connection")
		}
	})
	return one
````

````old server/internal/bootstrap/project_connection_test.go
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
````
````new server/internal/bootstrap/project_connection_test.go
	one := poolOfOne(t, r.pool.Config().ConnString())
````

````old server/internal/bootstrap/project_connection_test.go
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
````
````new server/internal/bootstrap/project_connection_test.go
		return route.answerWithin(t, contract, method, path, caller, body, want)
````

`server/internal/bootstrap/interleaving_not_found_test.go`（修改，1 处）：

````old server/internal/bootstrap/interleaving_not_found_test.go
func (w webWrite) send(route projectRoute, r growthRace) (*http.Request, *httptest.ResponseRecorder) {
````
````new server/internal/bootstrap/interleaving_not_found_test.go
func (w webWrite) send(route moduleRoute, r growthRace) (*http.Request, *httptest.ResponseRecorder) {
````

`server/internal/bootstrap/ending_test.go`（修改，9 处）：

````old server/internal/bootstrap/ending_test.go
	"uuid"

````
````new server/internal/bootstrap/ending_test.go
	"uuid"

	"github.com/jackc/pgx/v5"
````

````old server/internal/bootstrap/ending_test.go
}

// leave is name's leaving of the workspace slug: its status and body.
````
````new server/internal/bootstrap/ending_test.go
}

// joinsOps has alice join Ops, as its admin, so that bob is no longer its
// only admin and the ending of his membership can go through.
func (w endingWorld) joinsOps(t *testing.T) {
	t.Helper()
	if status, body := call(t, w.contract, http.MethodPost, w.base+"/api/v0/projects/"+w.ops.String()+"/join", w.tokens["alice"],
		""); status != http.StatusOK {
		t.Fatalf("alice's joining Ops = %d %s", status, body)
	}
}

// leave is name's leaving of the workspace slug: its status and body.
````

````old server/internal/bootstrap/ending_test.go
}

// rowJSON is the row id of table as JSON, its columns by name; nil when
````
````new server/internal/bootstrap/ending_test.go
}

// querier is what rowJSON reads through: a pool, or a transaction, which
// sees its own writes.
type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// rowJSON is the row id of table as JSON, its columns by name; nil when
````

````old server/internal/bootstrap/ending_test.go
func rowJSON(t *testing.T, pool *pgxpool.Pool, table string, id uuid.UUID) map[string]any {
````
````new server/internal/bootstrap/ending_test.go
func rowJSON(t *testing.T, q querier, table string, id uuid.UUID) map[string]any {
````

````old server/internal/bootstrap/ending_test.go
	if err := pool.QueryRow(context.Background(), "SELECT coalesce((SELECT row_to_json(r) FROM "+table+" r WHERE r.id = $1)::text, 'null')", id).
````
````new server/internal/bootstrap/ending_test.go
	if err := q.QueryRow(context.Background(), "SELECT coalesce((SELECT row_to_json(r) FROM "+table+" r WHERE r.id = $1)::text, 'null')", id).
````

````old server/internal/bootstrap/ending_test.go
	by      func(name string) string // the account it writes as
````
````new server/internal/bootstrap/ending_test.go
	by      func(name string) string // the account it writes as
	// notFound is the code of its 404: the membership's, or the workspace's.
	notFound string
````

````old server/internal/bootstrap/ending_test.go
	}, by: func(string) string { return "alice" }},
````
````new server/internal/bootstrap/ending_test.go
	}, by: func(string) string { return "alice" }, notFound: "workspace.member_not_found"},
````

````old server/internal/bootstrap/ending_test.go
	}, by: func(name string) string { return name }},
````
````new server/internal/bootstrap/ending_test.go
	}, by: func(name string) string { return name }, notFound: "workspace.not_found"},
````

````old server/internal/bootstrap/ending_test.go
			if status, body := call(t, w.contract, http.MethodPost, w.base+"/api/v0/projects/"+w.ops.String()+"/join", w.tokens["alice"],
				""); status != http.StatusOK {
				t.Fatalf("alice's joining Ops = %d %s", status, body)
			}
````
````new server/internal/bootstrap/ending_test.go
			w.joinsOps(t)
````

- [ ] **Step 2: 竞争、锁、连接**

`server/internal/bootstrap/ending_races_test.go`（新文件，191 行）：

````file server/internal/bootstrap/ending_races_test.go
package bootstrap

import (
	"context"
	"maps"
	"net/http"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// An ending that waits for acme's row, against what another transaction
// changes or holds meanwhile, on the wired app (M3 design 3.6's lock table,
// convention 1, convention 2): what it finds once it has the lock, and
// each lock it takes, at its strength.

// sent sends e's ending of name's membership on w's app, checked against
// the contract, and hands over its answer.
func (e ending) sent(w endingWorld, t *testing.T, name string) (*http.Request, <-chan answer) {
	t.Helper()
	method, path, token := e.request(w, t, name)
	req := newRequest(t, method, w.base+path, token, nil)
	w.contract.CheckRequest(t, req)
	return req, sendInBackground(req)
}

// An ending that waits for acme's row reads, once it has it, what another
// transaction committed meanwhile, and answers its 404, as for a membership
// or a workspace never there, never the 403 that would tell its caller
// that it was there; and it changes no row (M3 design 3.6 convention 2,
// the lock table). The other transaction holds acme FOR NO KEY UPDATE and
// ends bob's membership, ends alice's, the remover's, or deletes acme; bob's
// removal by alice, or his leaving, has passed authentication and waits for
// acme's row. Once the other commits, the removal is 404
// workspace.member_not_found, the leaving 404 workspace.not_found; the row
// the other changed is as it left it, and every other row as it was. alice
// has joined Ops first, so that nothing else refuses the ending.
func TestAnEndingFindsWhatEndedMeanwhile(t *testing.T) {
	type change struct {
		name, sql string
		row       func(w endingWorld, t *testing.T) (table string, id uuid.UUID)
	}
	membershipOf := func(name string) func(w endingWorld, t *testing.T) (string, uuid.UUID) {
		return func(w endingWorld, t *testing.T) (string, uuid.UUID) {
			return "workspace_members", w.membership(t, name)
		}
	}
	ends := func(name string) change {
		return change{name + "'s membership ended", "UPDATE workspace_members SET is_active = false WHERE id = $1", membershipOf(name)}
	}
	deletesAcme := change{"acme deleted", "UPDATE workspaces SET deleted_at = now(), updated_at = now() WHERE id = $1",
		func(w endingWorld, t *testing.T) (string, uuid.UUID) { return "workspaces", w.workspace(t, "acme") }}
	for _, tt := range []struct {
		ending ending
		change change
	}{
		{endings[0], ends("bob")}, {endings[0], ends("alice")}, {endings[0], deletesAcme},
		{endings[1], ends("bob")}, {endings[1], deletesAcme},
	} {
		t.Run(tt.ending.name+", "+tt.change.name, func(t *testing.T) {
			w := newEndingWorld(t)
			w.joinsOps(t)
			table, id := tt.change.row(w, t)
			others := rowsBut(t, w.pool, []uuid.UUID{id})
			other := holding(t, w.pool, "SELECT 1 FROM workspaces WHERE slug = 'acme' FOR NO KEY UPDATE")
			if tag, err := other.Exec(context.Background(), tt.change.sql, id); err != nil || tag.RowsAffected() != 1 {
				t.Fatalf("%s: %v, %v; want one row changed", tt.change.sql, tag, err)
			}
			changed := rowJSON(t, other, table, id)
			req, answered := tt.ending.sent(w, t, "bob")
			pgtest.WaitForLockWaitOn(t, w.pool, "workspaces", 5*time.Second)
			if err := other.Commit(context.Background()); err != nil {
				t.Fatal(err)
			}

			a := receiveWithin(t, answered, 10*time.Second, "answer to the ending")
			if a.err != nil {
				t.Fatal(a.err)
			}
			w.contract.CheckResponse(t, req, a.res)
			if a.res.StatusCode != http.StatusNotFound || problemCode(t, a.body) != tt.ending.notFound {
				t.Errorf("the ending = %d %s, want 404 %s", a.res.StatusCode, a.body, tt.ending.notFound)
			}
			if after := rowJSON(t, w.pool, table, id); !maps.Equal(after, changed) {
				t.Errorf("%s %s after the ending:\n%v\nwant it as the other transaction left it:\n%v", table, id, after, changed)
			}
			if after := rowsBut(t, w.pool, []uuid.UUID{id}); !maps.Equal(after, others) {
				t.Errorf("every other row after the ending:\n%v\nwant them as they were:\n%v", after, others)
			}
		})
	}
}

// Each lock of an ending is taken in the lock table's order, at its
// strength, as bootstrap wires it (M3 design 3.6's lock table, convention
// 1, convention 6, the global order: the invitations before the members,
// the projects before their members). Other transactions hold acme's row
// FOR NO KEY UPDATE and, FOR SHARE, the pending invitation to bob's
// address, his membership of acme and his membership of Ops, which the
// ending's statements wait for in turn; they let go one at a time, and
// lockOn reads each row's strongest lock then. alice's removal of bob, or
// his leaving, waits for acme's row, holding nothing; then for the
// invitation, holding acme's row alone; then for his membership, holding
// acme's row and the invitation FOR NO KEY UPDATE; then for his membership
// of Ops, holding acme's row, the invitation, his membership, Web and Ops,
// each FOR NO KEY UPDATE, no stronger, no weaker; never Site, alice's
// project of acme, of which he is no member; and never his account at
// FOR SHARE or above, which a deactivation's FOR NO KEY UPDATE of it would
// wait for: the ending reads his address unlocked (convention 1). Then it
// is 204, and the moment it wrote is no earlier than acme's release: it
// read the clock under acme's lock (3.3).
func TestEachLockOfAnEndingIsItsStrength(t *testing.T) {
	for _, e := range endings {
		t.Run(e.name, func(t *testing.T) {
			w := newEndingWorld(t)
			w.joinsOps(t)
			acme, bobs := w.workspace(t, "acme"), w.membership(t, "bob")
			opsMembership := projectMemberships(t, w.pool, w.ids["bob"], w.ops)[0]
			site := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Site", "SITE")
			locks := func() string {
				t.Helper()
				return "acme " + lockOn(t, w.pool, "workspaces WHERE id = $1", acme) +
					", his membership " + lockOn(t, w.pool, "workspace_members WHERE id = $1", bobs) +
					", the invitation " + lockOn(t, w.pool, "workspace_member_invites WHERE id = $1", w.bobsInvitation) +
					", Web " + lockOn(t, w.pool, "projects WHERE id = $1", w.web) +
					", Ops " + lockOn(t, w.pool, "projects WHERE id = $1", w.ops) +
					", Site " + lockOn(t, w.pool, "projects WHERE id = $1", site)
			}
			holdsOps := holding(t, w.pool, "SELECT 1 FROM project_members WHERE id = $1 FOR SHARE", opsMembership)
			holdsMembership := holding(t, w.pool, "SELECT 1 FROM workspace_members WHERE id = $1 FOR SHARE", bobs)
			holdsInvitation := holding(t, w.pool, "SELECT 1 FROM workspace_member_invites WHERE id = $1 FOR SHARE", w.bobsInvitation)
			holdsAcme := holding(t, w.pool, "SELECT 1 FROM workspaces WHERE id = $1 FOR NO KEY UPDATE", acme)
			req, answered := e.sent(w, t, "bob")
			var released time.Time
			// Each step lets go of one holder and names the table the
			// ending then waits on, and its locks: acme's is the holder's
			// until the first step; the FOR SHARE ones are the holders'.
			for _, step := range []struct {
				release pgx.Tx
				waitsOn string
				want    string
			}{
				{nil, "workspaces", "acme FOR NO KEY UPDATE, his membership FOR SHARE, the invitation FOR SHARE, Web no lock, Ops no lock, " +
					"Site no lock"},
				{holdsAcme, "workspace_member_invites", "acme FOR NO KEY UPDATE, his membership FOR SHARE, the invitation FOR SHARE, Web no lock, " +
					"Ops no lock, Site no lock"},
				{holdsInvitation, "workspace_members", "acme FOR NO KEY UPDATE, his membership FOR SHARE, the invitation FOR NO KEY UPDATE, " +
					"Web no lock, Ops no lock, Site no lock"},
				{holdsMembership, "project_members", "acme FOR NO KEY UPDATE, his membership FOR NO KEY UPDATE, the invitation FOR NO KEY UPDATE, " +
					"Web FOR NO KEY UPDATE, Ops FOR NO KEY UPDATE, Site no lock"},
			} {
				if step.release != nil {
					if step.release == holdsAcme {
						released = time.Now()
					}
					if err := step.release.Rollback(context.Background()); err != nil {
						t.Fatal(err)
					}
				}
				pgtest.WaitForLockWaitOn(t, w.pool, step.waitsOn, 5*time.Second)
				if got := locks(); got != step.want {
					t.Errorf("the ending waiting on %s: %s; want %s", step.waitsOn, got, step.want)
				}
				if heldBy(t, w.pool, "SELECT 1 FROM users WHERE id = $1 FOR NO KEY UPDATE", w.ids["bob"]) {
					t.Errorf("the ending waiting on %s holds bob's account at FOR SHARE or above; want it unlocked, or held FOR KEY SHARE at most",
						step.waitsOn)
				}
			}
			if err := holdsOps.Rollback(context.Background()); err != nil {
				t.Fatal(err)
			}

			a := receiveWithin(t, answered, 10*time.Second, "answer to the ending")
			if a.err != nil {
				t.Fatal(a.err)
			}
			w.contract.CheckResponse(t, req, a.res)
			if a.res.StatusCode != http.StatusNoContent {
				t.Errorf("the ending = %d %s, want 204", a.res.StatusCode, a.body)
			}
			moment, _ := rowJSON(t, w.pool, "workspace_members", bobs)["updated_at"].(string)
			if at, err := time.Parse(time.RFC3339Nano, moment); err != nil || at.Before(released.Truncate(time.Microsecond)) {
				t.Errorf("the ending's moment %q (%v); want one no earlier than acme's release, %v", moment, err, released)
			}
		})
	}
}
````

`server/internal/bootstrap/ending_connection_test.go`（新文件，100 行）：

````file server/internal/bootstrap/ending_connection_test.go
package bootstrap

import (
	"context"
	"log/slog"
	"net/http"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	identitypg "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	projectpg "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres"
	projectapp "github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// newWorkspaceRoute is the workspace module (workspace.New) on pool, as
// bootstrap wires it for the endings: access's Authorizer, identity's
// Accounts and MemberProfiles, converted, and the project module's cascade,
// each on pool.
func newWorkspaceRoute(t *testing.T, pool *pgxpool.Pool) moduleRoute {
	t.Helper()
	tx, auth, identityPorts, provided := postgres.NewTxManager(pool, 2*time.Second), authorizerOn(pool), identity.Provide(pool), workspace.Provide(pool)
	projects := project.New(project.Deps{Pool: pool, Tx: tx, Clock: clock.System{}, Authorizer: auth,
		Workspaces: projectWorkspaces{directory: provided.WorkspaceDirectory}, Members: provided.WorkspaceMembers})
	return newRoute(t, workspace.New(workspace.Deps{Pool: pool, Tx: tx, Clock: clock.System{}, Logger: slog.New(slog.DiscardHandler),
		Authorizer: auth, Accounts: workspaceAccounts{accounts: identityPorts.Accounts},
		Profiles: workspaceProfiles{profiles: identityPorts.PublicProfiles}, Projects: projects.Cascade()}).Register)
}

// Each ending runs every statement on its transaction's connection (M3
// design 3.6 convention 2, 6.7): its locks, its reads (the membership, the
// decision's role, the other admin, the member's address through
// MemberProfiles, the projects' only admin), its writes and the projects'
// step. The workspace module and the project module's cascade are wired as
// bootstrap wires them, on a pool of one connection: a statement sent
// through the pool rather than the transaction would wait for a second
// connection that never comes, and its request fail at the request's
// deadline. In acme, alice and carol are the admins and bob a member;
// alice's Web has the three, carol its admin too; a pending invitation to
// each address of the three. alice removes bob, and leaves.
func TestTheEndingsRunOnTheirTransactionsConnection(t *testing.T) {
	r := newGrowthRace(t, false)
	carol := uuid.NewV7()
	ctx, now := context.Background(), time.Now()
	users := identitypg.New(r.pool)
	if err := users.CreateUser(ctx, identityapp.NewUser{ID: carol, Email: "carol@example.com", PasswordHash: "x", DisplayName: "carol",
		Now: now}); err != nil {
		t.Fatal(err)
	}
	if err := users.CreateDefaultProfile(ctx, uuid.NewV7(), carol, now); err != nil {
		t.Fatal(err)
	}
	var acme uuid.UUID
	if err := r.pool.QueryRow(ctx, "SELECT workspace_id FROM projects WHERE id = $1", r.web).Scan(&acme); err != nil {
		t.Fatal(err)
	}
	workspaces, projects := workspacepg.New(r.pool), projectpg.New(r.pool)
	if err := workspaces.CreateMember(ctx, workspaceapp.MemberRow{ID: uuid.NewV7(), WorkspaceID: acme, MemberID: carol, Role: shared.RoleAdmin,
		CreatedBy: r.alice, Now: now}); err != nil {
		t.Fatal(err)
	}
	for user, role := range map[uuid.UUID]shared.Role{r.bob: shared.RoleMember, carol: shared.RoleAdmin} {
		if err := projects.CreateMember(ctx, projectapp.MemberRow{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: r.web, MemberID: user, Role: role,
			CreatedBy: r.alice, Now: now}); err != nil {
			t.Fatal(err)
		}
	}
	for _, email := range []string{"alice@example.com", "bob@example.com", "carol@example.com"} {
		if _, err := workspaces.CreateInvitations(ctx, []workspaceapp.InvitationRow{{ID: uuid.NewV7(), WorkspaceID: acme, Email: email,
			Role: shared.RoleGuest, CreatedBy: r.alice, Now: now}}); err != nil {
			t.Fatal(err)
		}
	}
	route := newWorkspaceRoute(t, poolOfOne(t, r.pool.Config().ConnString()))
	contract := apitest.Load(t)

	route.answerWithin(t, contract, http.MethodDelete, "/api/v0/workspace-members/"+r.bobIn.String(), r.alice, "", http.StatusNoContent)
	route.answerWithin(t, contract, http.MethodPost, "/api/v0/workspaces/acme/leave", r.alice, "", http.StatusNoContent)

	var ended, pending int
	if err := r.pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM project_members WHERE project_id = $1 AND NOT is_active),
		(SELECT count(*) FROM workspace_member_invites WHERE workspace_id = $2 AND deleted_at IS NULL)`, r.web, acme).Scan(&ended, &pending); err != nil {
		t.Fatal(err)
	}
	if ended != 2 || pending != 1 {
		t.Errorf("after the removal and the leaving: %d ended memberships of Web, %d pending invitations; want 2, carol's alone", ended, pending)
	}
}
````

- [ ] **Step 3: 测试和 lint**

Run: `go -C server test -count=1 -run 'TestAnEndingFindsWhatEndedMeanwhile|TestEachLockOfAnEndingIsItsStrength|TestTheEndingsRunOnTheirTransactionsConnection|TestTheWritesOnAProjectRunOnTheirTransactionsConnection|TestAnEndingEndsTheMembershipsAndLeavesNoInvitation|TestAWriteOnAProject' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 4: 提交**

```bash
git add server/internal/bootstrap/ending_connection_test.go server/internal/bootstrap/ending_races_test.go server/internal/bootstrap/ending_test.go server/internal/bootstrap/interleaving_not_found_test.go server/internal/bootstrap/project_connection_test.go
```
```bash
git commit -m "test(M3/P5a): an ending against what changes meanwhile, its locks' strengths and its connection

On the wired app, a removal or a leaving that waits for the workspace's
row answers 404 for a membership ended or a workspace deleted meanwhile,
never 403, and changes no row; each lock it takes is pinned at its
strength, in the lock table's order, the clock read after the
workspace's; on a pool of one connection, every statement runs on the
transaction's. The project module's one-connection route becomes a
route of any module.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A，清扫 8、9、12、13）：

| 改坏 | 必须失败的测试 | 层 |
|---|---|---|
| `LockWorkspace`（移出）、`LockWorkspaceBySlug`（离开）锁成 `FOR SHARE` | `TestEachLockOfAnEndingIsItsStrength` | 组合 |
| `LockActiveMemberProjects` 锁成 `FOR SHARE`、`FOR UPDATE`、不加锁；去掉相关、成员 | `TestEachLockOfAnEndingIsItsStrength` | 组合 |
| 结束一步：成员关系在邀请之前；项目在两者之前 | `TestEachLockOfAnEndingIsItsStrength` | 组合 |
| 时钟在工作区的锁之前读（移出、离开） | `TestEachLockOfAnEndingIsItsStrength` | 组合 |
| 移出不在锁下重读成员行；移出已结束的成员关系；在工作区的锁之前判定 | `TestAnEndingFindsWhatEndedMeanwhile` | 组合 |
| 结束的十条语句（三把锁、成员行的读、另一个管理员、地址、邀请、成员关系、项目的锁、唯一管理员、项目成员关系）各经连接池执行 | `TestTheEndingsRunOnTheirTransactionsConnection` | 组合 |

**Done when:** 五个竞争情形都是 404、一行不改；两个结束的每把锁在组合一层钉住强度和顺序；每条语句在事务的连接上。

---

### Task 7: `ProjectMembershipCounts` 与恢复成员的用例

**Files:**
- Create: `server/internal/modules/workspace/adapter/postgres/reactivation.go`、`server/internal/modules/workspace/adapter/postgres/reactivation_test.go`、`server/internal/modules/workspace/app/reactivate_member.go`、`server/internal/modules/workspace/app/reactivate_member_test.go`
- Modify: `server/internal/modules/project/adapter/postgres/members.go`、`server/internal/modules/project/adapter/postgres/members_test.go`、`server/internal/modules/project/adapter/postgres/queries/members.sql`、`server/internal/modules/project/module.go`、`server/internal/modules/workspace/adapter/postgres/queries/members.sql`、`server/internal/modules/workspace/app/clock_test.go`、`server/internal/modules/workspace/app/create_workspace_test.go`、`server/internal/modules/workspace/app/fakes_members_test.go`、`server/internal/modules/workspace/app/fakes_projects_test.go`、`server/internal/modules/workspace/app/fakes_workspaces_test.go`、`server/internal/modules/workspace/app/ports.go`、`server/internal/modules/workspace/domain/errors.go`
- Generate: `server/internal/modules/project/adapter/postgres/gen/members.sql.go`、`server/internal/modules/workspace/adapter/postgres/gen/members.sql.go`

**Interfaces:**
- Produces（spec 2.9，M3 设计 3.6 的加锁表、约定六的例外，3.11，6.5）：
  - `project.ProjectMembershipCounts`（`CountInactive(ctx, workspaceID, userID) (int, error)`：他在这个工作区各项目已结束、未删除的成员关系的个数，不加锁读），进 `project.Provided`；存储方法 `CountInactive` 和查询 `CountInactiveMemberships`。
  - `workspace` 的存储 `ReactivateMember(ctx, workspaceID, userID, now) error`：他未删除的成员关系 `is_active = true`、`updated_at = now`，角色不变，`updated_by_id` 不动（照 Plane 的命令：没有哪个账户发起它，spec 第 3 节第 4 条）；不是恰好一行时是一个错误，不是 `ErrNotFound`。
  - `workspace/app.ProjectMembershipCounts`（同一个方法）、`MemberReactivator`（`LockWorkspaceBySlug`、`MemberOf`、`ReactivateMember`）；`Reactivation{Email, Role, AccountActive, AlreadyActive, EndedProjectMemberships}`；`NewReactivateMember(accounts Accounts, members MemberReactivator, counts ProjectMembershipCounts, tx, clock, logger)`，`Execute(ctx, slug, email) (Reactivation, error)`：地址照注册的规则规范化；一个事务里账户行 `FOR SHARE`（`ShareAccountByEmail`，没有时 `ErrAccountNotFound`；停用的照样往下）→ 工作区 `FOR NO KEY UPDATE`（没有时 `ErrSlugNotFound`）→ 读成员关系（没有时 `ErrNeverAMember`；有效的：报告，不读时钟、不写、不数、不记日志）→ 读时钟 → `ReactivateMember` → `CountInactive`；提交之后记一行日志。
  - `workspace/domain`：`ErrSlugNotFound`（`workspace.slug_not_found`，"No workspace has this slug."）、`ErrNeverAMember`（`workspace.never_a_member`，"The account has never been a member of this workspace."），只在命令行（spec 第 3 节第 3 条）。

**Tests:**
- `project/adapter/postgres/members_test.go`：`TestCountInactive`（bob 在 acme 的 Web、Ops 已结束：2；不数他在 Docs 有效的、在已删除的 Old 已结束的、在 beta 的项目已结束的（那是 beta 的数）；不数 carol 在 Web 已结束的（那是她的数）；alice 是 0）。
- `workspace/adapter/postgres/reactivation_test.go`：`TestReactivateMember`（恢复为有效，在给的时刻，角色不变，`updated_by_id` 仍是结束它的 alice；他已删除的（两个行序）、他在 beta 已结束的、carol 在 acme 已结束的一列不动；没有未删除成员关系的一对是不是 `ErrNotFound` 的错误，什么都不改）。
- `workspace/app/reactivate_member_test.go`：`TestReactivateMemberRestoresTheEndedMembership`（账户行按规范化的地址先锁，再锁工作区、读成员关系、在锁下读时钟、恢复、数，全部在一个事务里，记一行；carol 的账户已停用，照样恢复；角色不变）；`TestReactivateMemberLeavesAnActiveMembership`（有效的：报告，不读时钟、不写、不数、不记日志）；`TestReactivateMemberRefusals`（没有账户、没有工作区或等锁期间删除、从来不是成员：各自的码，什么都不改、不记日志；每个端口的失败原样返回，从不答成这几个码之一）。
- `workspace/app/clock_test.go`：`TestEachWriteReadsTheClockUnderItsLock` 多恢复一行；`create_workspace_test.go` 的 `logged` 改为共用。

- [ ] **Step 1: 查询**

`server/internal/modules/project/adapter/postgres/queries/members.sql`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/queries/members.sql
WHERE id = sqlc.arg(id);

````
````new server/internal/modules/project/adapter/postgres/queries/members.sql
WHERE id = sqlc.arg(id);

-- name: CountInactiveMemberships :one
-- ProjectMembershipCounts, for reactivate-member's report (M3 design 3.11, 6.5): the account's ended undeleted
-- memberships of the workspace's projects, read without a lock. A deleted project's memberships are deleted with it.
SELECT count(*)
FROM project_members
WHERE workspace_id = sqlc.arg(workspace_id) AND member_id = sqlc.arg(member_id) AND NOT is_active AND deleted_at IS NULL;

````

`server/internal/modules/workspace/adapter/postgres/queries/members.sql`（修改，1 处）：

````old server/internal/modules/workspace/adapter/postgres/queries/members.sql
                 AND deleted_at IS NULL);

````
````new server/internal/modules/workspace/adapter/postgres/queries/members.sql
                 AND deleted_at IS NULL);

-- name: ReactivateMember :execrows
-- reactivate-member, under the workspace's FOR NO KEY UPDATE (M3 design 3.11): the user's ended membership active
-- again, its role kept. As Plane's command, it writes is_active and updated_at alone: no account of the instance asks
-- for it, so updated_by_id stays whose it was.
UPDATE workspace_members
SET is_active = true, updated_at = sqlc.arg(now)
WHERE workspace_id = sqlc.arg(workspace_id) AND member_id = sqlc.arg(member_id) AND deleted_at IS NULL;

````

Run: `make gen-go`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `4b6c239c5833517e3846d4b6e064d9656ac0d2478ffee2cb091c11d9fd5aa50a` | 178 | `server/internal/modules/project/adapter/postgres/gen/members.sql.go` |
| `e0015915623e5ba82d69100a24df18520311673c95f6bec229b37ae142717bbc` | 331 | `server/internal/modules/workspace/adapter/postgres/gen/members.sql.go` |

Run: `shasum -a 256 server/internal/modules/project/adapter/postgres/gen/members.sql.go server/internal/modules/workspace/adapter/postgres/gen/members.sql.go`
Expected: 与上表相同。

- [ ] **Step 2: 两个存储和 `project.Provide`**

`server/internal/modules/project/adapter/postgres/members.go`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/members.go
		out[i] = domain.Member{ID: r.ID, ProjectID: r.ProjectID, MemberID: r.MemberID, Role: shared.Role(r.Role), CreatedAt: r.CreatedAt}
	}
	return out, nil
}

````
````new server/internal/modules/project/adapter/postgres/members.go
		out[i] = domain.Member{ID: r.ID, ProjectID: r.ProjectID, MemberID: r.MemberID, Role: shared.Role(r.Role), CreatedAt: r.CreatedAt}
	}
	return out, nil
}

// CountInactive is the number of userID's ended, undeleted memberships of
// the workspace's projects, read without a lock
// (project.ProjectMembershipCounts).
func (s *Store) CountInactive(ctx context.Context, workspaceID, userID uuid.UUID) (int, error) {
	n, err := s.queries(ctx).CountInactiveMemberships(ctx, gen.CountInactiveMembershipsParams{WorkspaceID: workspaceID, MemberID: userID})
	if err != nil {
		return 0, fmt.Errorf("count the ended project memberships of %s in workspace %s: %w", userID, workspaceID, err)
	}
	return int(n), nil
}

````

`server/internal/modules/project/adapter/postgres/members_test.go`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/members_test.go
		}
	}
}

````
````new server/internal/modules/project/adapter/postgres/members_test.go
		}
	}
}

// CountInactive counts the account's ended undeleted memberships of the
// workspace's projects: bob's of Web and of Ops in acme, 2; not his active
// one of Docs, nor his ended one of Old, deleted, nor his ended one of
// beta's project, which beta's count is; not carol's ended one of Web,
// which hers is. alice has none.
func TestCountInactive(t *testing.T) {
	s, pool := newStore(t)
	alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	web, ops := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Ops", "OPS", alice)
	docs, old := newProject(t, s, acme, "Docs", "DOC", alice), newProject(t, s, acme, "Old", "OLD", alice)
	betas := newProject(t, s, beta, "Web", "WEB", alice)
	seedMember(t, pool, acme, web, bob, 20, false)
	seedMember(t, pool, acme, ops, bob, 15, false)
	seedMember(t, pool, acme, docs, bob, 15, true)
	exec(t, pool, "UPDATE project_members SET deleted_at = $2 WHERE id = $1", seedMember(t, pool, acme, old, bob, 15, false), now)
	seedMember(t, pool, beta, betas, bob, 15, false)
	seedMember(t, pool, acme, web, carol, 5, false)
	seedMember(t, pool, acme, web, alice, 20, true)
	for _, tt := range []struct {
		name            string
		workspace, user uuid.UUID
		want            int
	}{{"bob in acme", acme, bob, 2}, {"bob in beta", beta, bob, 1}, {"carol in acme", acme, carol, 1}, {"alice in acme", acme, alice, 0}} {
		if got, err := s.CountInactive(context.Background(), tt.workspace, tt.user); err != nil || got != tt.want {
			t.Errorf("CountInactive() of %s = %d, %v; want %d", tt.name, got, err, tt.want)
		}
	}
}

````

`server/internal/modules/project/module.go`（修改，4 处）：

````old server/internal/modules/project/module.go
// access module its reads of a project (ProjectAccess).
````
````new server/internal/modules/project/module.go
// access module its reads of a project (ProjectAccess) and the workspace
// module its count of an account's ended project memberships
// (ProjectMembershipCounts).
````

````old server/internal/modules/project/module.go
type AccessFacts = app.AccessFacts

````
````new server/internal/modules/project/module.go
type AccessFacts = app.AccessFacts

// ProjectMembershipCounts counts an account's memberships of a workspace's
// projects, for the workspace module's reactivate-member (M3 design 3.11,
// 6.5): CountInactive is the number of userID's ended, undeleted
// memberships of the workspace's projects, read without a lock.
type ProjectMembershipCounts interface {
	CountInactive(ctx context.Context, workspaceID, userID uuid.UUID) (int, error)
}

````

````old server/internal/modules/project/module.go
	ProjectAccess ProjectAccess
````
````new server/internal/modules/project/module.go
	ProjectAccess           ProjectAccess
	ProjectMembershipCounts ProjectMembershipCounts
````

````old server/internal/modules/project/module.go
	return Provided{ProjectAccess: postgresadapter.New(pool)}
````
````new server/internal/modules/project/module.go
	store := postgresadapter.New(pool)
	return Provided{ProjectAccess: store, ProjectMembershipCounts: store}
````

`server/internal/modules/workspace/adapter/postgres/reactivation.go`（新文件，26 行）：

````file server/internal/modules/workspace/adapter/postgres/reactivation.go
package postgresadapter

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres/gen"
)

// ReactivateMember makes userID's membership of the workspace active again,
// its role kept, at now; updated_by_id stays whose it was, as in Plane's
// command (M3 design 3.11). The caller read the membership ended under the
// workspace's lock: a pair without exactly one undeleted membership is an
// error, not app.ErrNotFound.
func (s *Store) ReactivateMember(ctx context.Context, workspaceID, userID uuid.UUID, now time.Time) error {
	restored, err := s.queries(ctx).ReactivateMember(ctx, gen.ReactivateMemberParams{WorkspaceID: workspaceID, MemberID: userID, Now: now})
	switch {
	case err != nil:
		return fmt.Errorf("reactivate workspace member: %w", err)
	case restored != 1:
		return fmt.Errorf("reactivate workspace member %s of %s: %d undeleted memberships", userID, workspaceID, restored)
	}
	return nil
}
````

`server/internal/modules/workspace/adapter/postgres/reactivation_test.go`（新文件，81 行）：

````file server/internal/modules/workspace/adapter/postgres/reactivation_test.go
package postgresadapter_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ReactivateMember makes the user's undeleted membership of the workspace
// active again at the time given, its role kept, and writes nothing else:
// updated_by_id stays alice's, who ended it (M3 design 3.11). bob's
// deleted membership of acme, stored before his live one or after it, his
// ended membership of beta and carol's ended one of acme keep every column,
// and his reactivated one its other columns. A pair with no undeleted
// membership is an error that is not app.ErrNotFound, and changes nothing:
// carol's in beta, of which there is none, and bob's in gamma, deleted.
func TestReactivateMember(t *testing.T) {
	for _, deletedFirst := range []bool{true, false} {
		t.Run(fmt.Sprintf("the deleted membership stored first %v", deletedFirst), func(t *testing.T) {
			s, pool := newStore(t)
			alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
			acme, beta, gamma := newWorkspace(t, s, "Acme", "acme", alice), newWorkspace(t, s, "Beta", "beta", alice),
				newWorkspace(t, s, "Gamma", "gamma", alice)
			deleted := func(workspace uuid.UUID) {
				exec(t, pool, `INSERT INTO workspace_members (id, workspace_id, member_id, role, is_active, created_by_id, updated_by_id,
					created_at, updated_at, deleted_at) VALUES ($1, $2, $3, 15, false, $3, $3, $4, $4, $4)`, uuid.NewV7(), workspace, bob,
					now.Add(-time.Hour))
			}
			if deletedFirst {
				deleted(acme.ID)
			}
			bobIn := joinAt(t, s, acme.ID, bob, shared.RoleAdmin, now)
			if !deletedFirst {
				deleted(acme.ID)
			}
			joinAt(t, s, acme.ID, carol, shared.RoleGuest, now)
			joinAt(t, s, beta.ID, bob, shared.RoleMember, now)
			deleted(gamma.ID)
			for _, w := range []uuid.UUID{acme.ID, beta.ID} {
				if err := s.EndMember(context.Background(), w, bob, alice, now); err != nil {
					t.Fatal(err)
				}
			}
			if err := s.EndMember(context.Background(), acme.ID, carol, alice, now); err != nil {
				t.Fatal(err)
			}
			before := tableRows(t, pool, "workspace_members", []uuid.UUID{bobIn.ID}, "is_active", "updated_at")
			later := now.Add(time.Hour)

			if err := s.ReactivateMember(context.Background(), acme.ID, bob, later); err != nil {
				t.Fatal(err)
			}

			if got, want := stamp(t, pool, "workspace_members", bobIn.ID), fmt.Sprintf("active at %s by %s", at(t, pool, later), alice); got != want {
				t.Errorf("bob's membership of acme: %s, want %s", got, want)
			}
			if after := tableRows(t, pool, "workspace_members", []uuid.UUID{bobIn.ID}, "is_active", "updated_at"); after != before {
				t.Errorf("the memberships, bob's in acme without the columns written:\n%s\nwant them as they were:\n%s", after, before)
			}
			before = tableRows(t, pool, "workspace_members", nil)
			for _, pair := range []struct {
				name            string
				workspace, user uuid.UUID
			}{{"carol in beta", beta.ID, carol}, {"bob in gamma", gamma.ID, bob}} {
				if err := s.ReactivateMember(context.Background(), pair.workspace, pair.user, later.Add(time.Hour)); err == nil ||
					errors.Is(err, app.ErrNotFound) {
					t.Errorf("ReactivateMember() of %s = %v, want an error that is not app.ErrNotFound", pair.name, err)
				}
			}
			if after := tableRows(t, pool, "workspace_members", nil); after != before {
				t.Errorf("the memberships after reactivating none:\n%s\nwant them as they were:\n%s", after, before)
			}
		})
	}
}
````

- [ ] **Step 3: 用例**

`server/internal/modules/workspace/domain/errors.go`（修改，3 处）：

````old server/internal/modules/workspace/domain/errors.go
// last two are `nerve workspaces create`'s only (M3 design 3.11).
````
````new server/internal/modules/workspace/domain/errors.go
// last four are the administrator's commands' only, `nerve workspaces
// create`'s and `nerve workspaces reactivate-member`'s (M3 design 3.11).
````

````old server/internal/modules/workspace/domain/errors.go
	// ErrAccountNotFound answers `nerve workspaces create` for an address no
	// account has.
````
````new server/internal/modules/workspace/domain/errors.go
	// ErrAccountNotFound answers the administrator's commands for an address
	// no account has.
````

````old server/internal/modules/workspace/domain/errors.go
	ErrAccountDeactivated = shared.NewError(shared.KindForbidden, "workspace.account_deactivated", "The account is deactivated.")
````
````new server/internal/modules/workspace/domain/errors.go
	ErrAccountDeactivated = shared.NewError(shared.KindForbidden, "workspace.account_deactivated", "The account is deactivated.")
	// ErrSlugNotFound answers `nerve workspaces reactivate-member` for a slug
	// no undeleted workspace has.
	ErrSlugNotFound = shared.NewError(shared.KindNotFound, "workspace.slug_not_found", "No workspace has this slug.")
	// ErrNeverAMember answers `nerve workspaces reactivate-member` for an
	// account with no membership of the workspace, ended or active.
	ErrNeverAMember = shared.NewError(shared.KindNotFound, "workspace.never_a_member", "The account has never been a member of this workspace.")
````

`server/internal/modules/workspace/app/ports.go`（修改，1 处）：

````old server/internal/modules/workspace/app/ports.go
}

// PreferencesRow is a change of an account's display settings in a
````
````new server/internal/modules/workspace/app/ports.go
}

// ProjectMembershipCounts counts an account's memberships of a workspace's
// projects (M3 design 6.5): the project module implements it
// (project.Provide).
type ProjectMembershipCounts interface {
	// CountInactive is the number of userID's ended, undeleted memberships
	// of the workspace's projects, read without a lock.
	CountInactive(ctx context.Context, workspaceID, userID uuid.UUID) (int, error)
}

// MemberReactivator restores an account's ended membership of a workspace
// named by its slug, under the workspace's lock (M3 design 3.11).
type MemberReactivator interface {
	WorkspaceLocker
	// MemberOf returns userID's undeleted membership of the workspace,
	// active or ended; found is false when there is none.
	MemberOf(ctx context.Context, workspaceID, userID uuid.UUID) (m domain.Membership, found bool, err error)
	// ReactivateMember makes userID's undeleted membership of the
	// workspace active again, its role kept, at now.
	ReactivateMember(ctx context.Context, workspaceID, userID uuid.UUID, now time.Time) error
}

// PreferencesRow is a change of an account's display settings in a
````

`server/internal/modules/workspace/app/reactivate_member.go`（新文件，101 行）：

````file server/internal/modules/workspace/app/reactivate_member.go
package app

import (
	"context"
	"errors"
	"log/slog"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Reactivation is what `nerve workspaces reactivate-member` found and did.
type Reactivation struct {
	Email string      // the account's address, normalized
	Role  shared.Role // the membership's role, kept
	// AlreadyActive is set when the membership was active: nothing changed.
	AlreadyActive bool
	// EndedProjectMemberships is the number of the member's ended
	// memberships of the workspace's projects: he restores each by joining
	// its project (M3 design 3.5, 3.11).
	EndedProjectMemberships int
	// AccountActive is false for a deactivated account, which the command
	// reactivates all the same: `nerve users activate` is the next step.
	AccountActive bool
}

// ReactivateMember restores an ended membership, for the server's
// administrator: `nerve workspaces reactivate-member` (M3 design 3.11, as
// Plane's reactivate_workspace_member).
type ReactivateMember struct {
	accounts Accounts
	members  MemberReactivator
	counts   ProjectMembershipCounts
	tx       shared.TxManager
	clock    Clock
	logger   *slog.Logger
}

// NewReactivateMember returns the use case.
func NewReactivateMember(accounts Accounts, members MemberReactivator, counts ProjectMembershipCounts, tx shared.TxManager, clock Clock,
	logger *slog.Logger) *ReactivateMember {
	return &ReactivateMember{accounts: accounts, members: members, counts: counts, tx: tx, clock: clock, logger: logger}
}

// Execute makes the membership of the account of email in the workspace
// slug active again, its role kept, in one transaction (M3 design 3.6's
// lock table): the account row FOR SHARE first, its state read under the
// lock, a deactivated account allowed (3.6 convention 6, its one
// exception); then the workspace FOR NO KEY UPDATE, the membership read
// under it, the clock (3.3), the reactivation, and the count of his ended
// project memberships, read without a lock. The address is normalized as
// registration does it. No such account is workspace.account_not_found, no
// such workspace workspace.slug_not_found, no membership of it
// workspace.never_a_member; an active one is reported and changes nothing.
// A reactivation is logged once.
func (u *ReactivateMember) Execute(ctx context.Context, slug, email string) (Reactivation, error) {
	r := Reactivation{Email: shared.NormalizeEmail(email)}
	var workspaceID, userID uuid.UUID
	err := u.tx.WithinTx(ctx, func(ctx context.Context) error {
		state, found, err := u.accounts.ShareAccountByEmail(ctx, r.Email)
		switch {
		case err != nil:
			return err
		case !found:
			return domain.ErrAccountNotFound
		}
		r.AccountActive, userID = state.Active, state.ID
		workspaceID, err = u.members.LockWorkspaceBySlug(ctx, slug)
		switch {
		case errors.Is(err, ErrNotFound):
			return domain.ErrSlugNotFound
		case err != nil:
			return err
		}
		m, found, err := u.members.MemberOf(ctx, workspaceID, userID)
		switch {
		case err != nil:
			return err
		case !found:
			return domain.ErrNeverAMember
		}
		r.Role, r.AlreadyActive = m.Role, m.IsActive
		if m.IsActive {
			return nil
		}
		if err := u.members.ReactivateMember(ctx, workspaceID, userID, u.clock.Now()); err != nil {
			return err
		}
		r.EndedProjectMemberships, err = u.counts.CountInactive(ctx, workspaceID, userID)
		return err
	})
	if err != nil {
		return Reactivation{}, err
	}
	if !r.AlreadyActive {
		u.logger.InfoContext(ctx, "workspace member reactivated", slog.String("workspace_id", workspaceID.String()),
			slog.String("user_id", userID.String()), slog.String("by", byCLI))
	}
	return r, nil
}
````

`server/internal/modules/workspace/app/fakes_members_test.go`（修改，1 处）：

````old server/internal/modules/workspace/app/fakes_members_test.go
	}), nil
}

````
````new server/internal/modules/workspace/app/fakes_members_test.go
	}), nil
}

// The reactivation's repositories (app.MemberReactivator), besides the
// slug's lock.

func (f *fakeWorkspaces) MemberOf(ctx context.Context, workspaceID, userID uuid.UUID) (domain.Membership, bool, error) {
	f.log.add(ctx, "MemberOf %s %s", workspaceID, userID)
	if f.membersErr != nil {
		return domain.Membership{}, false, fmt.Errorf("read workspace member: %w", f.membersErr)
	}
	list := f.memberships[workspaceID]
	if i := slices.IndexFunc(list, func(m domain.Membership) bool { return m.MemberID == userID }); i >= 0 {
		return list[i], true, nil
	}
	return domain.Membership{}, false, nil
}

func (f *fakeWorkspaces) ReactivateMember(ctx context.Context, workspaceID, userID uuid.UUID, now time.Time) error {
	f.log.add(ctx, "ReactivateMember %s %s at %s", workspaceID, userID, now.Format(time.RFC3339Nano))
	if f.restoreErr != nil {
		return fmt.Errorf("reactivate workspace member: %w", f.restoreErr)
	}
	list := f.memberships[workspaceID]
	if i := slices.IndexFunc(list, func(m domain.Membership) bool { return m.MemberID == userID }); i >= 0 {
		list[i].IsActive = true
		return nil
	}
	return fmt.Errorf("reactivate workspace member %s of %s: no such row", userID, workspaceID)
}

````

`server/internal/modules/workspace/app/fakes_workspaces_test.go`（修改，1 处）：

````old server/internal/modules/workspace/app/fakes_workspaces_test.go
	memberships map[uuid.UUID][]domain.Membership // by workspace, for ListMembers and MemberByID
	membersErr  error                             // for ListMembers and MemberByID
	roleErr     error                             // for UpdateMemberRole
````
````new server/internal/modules/workspace/app/fakes_workspaces_test.go
	memberships map[uuid.UUID][]domain.Membership // by workspace, for ListMembers, MemberByID and MemberOf
	membersErr  error                             // for ListMembers, MemberByID and MemberOf
	roleErr     error                             // for UpdateMemberRole
	restoreErr  error                             // for ReactivateMember
````

`server/internal/modules/workspace/app/fakes_projects_test.go`（修改，1 处）：

````old server/internal/modules/workspace/app/fakes_projects_test.go
		return fmt.Errorf("end the member's project memberships: %w", err)
	}
	return nil
}

````
````new server/internal/modules/workspace/app/fakes_projects_test.go
		return fmt.Errorf("end the member's project memberships: %w", err)
	}
	return nil
}

// fakeCounts is the project module's ProjectMembershipCounts: it logs each
// call and answers the count it holds, or fails with its error, wrapped as
// the module wraps it.
type fakeCounts struct {
	log *callLog
	n   int
	err error
}

func (f *fakeCounts) CountInactive(ctx context.Context, workspaceID, userID uuid.UUID) (int, error) {
	f.log.add(ctx, "CountInactive %s %s", workspaceID, userID)
	if f.err != nil {
		return 0, fmt.Errorf("count the ended project memberships: %w", f.err)
	}
	return f.n, nil
}

````

`server/internal/modules/workspace/app/create_workspace_test.go`（修改，4 处）：

````old server/internal/modules/workspace/app/create_workspace_test.go
// logged is the log without each line's time.
func logged(f *createFixture) string {
````
````new server/internal/modules/workspace/app/create_workspace_test.go
// logged is a use case's log without each line's time.
func logged(logs *strings.Builder) string {
````

````old server/internal/modules/workspace/app/create_workspace_test.go
	for line := range strings.Lines(f.logs.String()) {
````
````new server/internal/modules/workspace/app/create_workspace_test.go
	for line := range strings.Lines(logs.String()) {
````

````old server/internal/modules/workspace/app/create_workspace_test.go
		if line, want := logged(f), createdLog(got.ID, user, "api"); line != want {
````
````new server/internal/modules/workspace/app/create_workspace_test.go
		if line, want := logged(f.logs), createdLog(got.ID, user, "api"); line != want {
````

````old server/internal/modules/workspace/app/create_workspace_test.go
		if line, want := logged(f), createdLog(got.ID, bob, "cli"); line != want {
````
````new server/internal/modules/workspace/app/create_workspace_test.go
		if line, want := logged(f.logs), createdLog(got.ID, bob, "cli"); line != want {
````

`server/internal/modules/workspace/app/reactivate_member_test.go`（新文件，149 行）：

````file server/internal/modules/workspace/app/reactivate_member_test.go
package app_test

import (
	"errors"
	"log/slog"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// reactivateFixture is ReactivateMember over membersFixture's fakes, with
// the accounts of alice, bob and carol, hers deactivated, and 2 ended
// project memberships to count.
type reactivateFixture struct {
	*membersFixture
	tx       *fakeTx
	accounts *fakeAccounts
	counts   *fakeCounts
	logs     *strings.Builder
}

func newReactivate() (*app.ReactivateMember, *reactivateFixture) {
	m := newMembers()
	f := &reactivateFixture{membersFixture: m, tx: &fakeTx{}, accounts: &fakeAccounts{log: m.log, accounts: []app.AccountState{alice, bob, carol}},
		counts: &fakeCounts{log: m.log, n: 2}, logs: &strings.Builder{}}
	return app.NewReactivateMember(f.accounts, f.workspaces, f.counts, f.tx, clockAt{clockNow, m.log}, slog.New(slog.NewTextHandler(f.logs, nil))), f
}

// reactivationCalls are the calls of user's reactivation in w up to the
// read of his membership: his account's lock by his address, the
// workspace's by its slug, the membership.
func reactivationCalls(user app.AccountState, w domain.Workspace) []string {
	return []string{"ShareAccountByEmail " + user.Email, "LockWorkspaceBySlug " + w.Slug, "MemberOf " + w.ID.String() + " " + user.ID.String()}
}

// reactivatedLog is the one line a reactivation logs.
func reactivatedLog(w domain.Workspace, user app.AccountState) string {
	return `level=INFO msg="workspace member reactivated" workspace_id=` + w.ID.String() + " user_id=" + user.ID.String() + " by=cli\n"
}

// The account row first, by the address normalized, then the workspace's
// lock, the membership, the clock under the lock, the reactivation and the
// count of his ended project memberships, all in one transaction (M3
// design 3.6's lock table, 3.11); one line logged. carol's account is
// deactivated, and is reactivated all the same; bob's membership of acme,
// ended for the case, is reactivated with his account active. Each keeps
// its role.
func TestReactivateMemberRestoresTheEndedMembership(t *testing.T) {
	at := clockNow.Format(time.RFC3339Nano)
	tests := []struct {
		name  string
		user  app.AccountState
		email string
		set   func(f *reactivateFixture)
		want  app.Reactivation
	}{
		{"carol, deactivated", carol, " Carol@Corp.COM ", nil,
			app.Reactivation{Email: carol.Email, Role: shared.RoleGuest, EndedProjectMemberships: 2, AccountActive: false}},
		{"bob, active", bob, bob.Email, func(f *reactivateFixture) { f.workspaces.memberships[acme.ID][1].IsActive = false },
			app.Reactivation{Email: bob.Email, Role: shared.RoleMember, EndedProjectMemberships: 2, AccountActive: true}},
	}
	for _, tt := range tests {
		uc, f := newReactivate()
		if tt.set != nil {
			tt.set(f)
		}
		got, err := uc.Execute(t.Context(), "acme", tt.email)
		want := append(reactivationCalls(tt.user, acme), "Now", "ReactivateMember "+acme.ID.String()+" "+tt.user.ID.String()+" at "+at,
			"CountInactive "+acme.ID.String()+" "+tt.user.ID.String())
		if err != nil || got != tt.want || !slices.Equal(f.log.calls, want) || f.tx.calls != 1 {
			t.Errorf("%s: Execute() = %+v, %v, calls\n%q\nin %d transactions; want %+v,\n%q\nin one", tt.name, got, err, f.log.calls, f.tx.calls,
				tt.want, want)
		}
		if logs := logged(f.logs); logs != reactivatedLog(acme, tt.user) {
			t.Errorf("%s: logs = %q, want %q", tt.name, logs, reactivatedLog(acme, tt.user))
		}
	}
}

// An active membership is reported and nothing changes: no clock, no
// write, no count, no log line (M3 design 3.11, as Plane's command).
func TestReactivateMemberLeavesAnActiveMembership(t *testing.T) {
	uc, f := newReactivate()
	got, err := uc.Execute(t.Context(), "acme", bob.Email)
	want := app.Reactivation{Email: bob.Email, Role: shared.RoleMember, AlreadyActive: true, AccountActive: true}
	if err != nil || got != want || !slices.Equal(f.log.calls, reactivationCalls(bob, acme)) || f.tx.calls != 1 || f.logs.Len() != 0 {
		t.Errorf("Execute() = %+v, %v, calls %q in %d transactions, logs %q; want %+v, %q in one, no log", got, err, f.log.calls, f.tx.calls,
			f.logs, want, reactivationCalls(bob, acme))
	}
}

// Each refusal and each failure is the answer, nothing is reactivated and
// nothing logged: no account of the address is workspace.account_not_found,
// a workspace not there, or deleted while the lock waited,
// workspace.slug_not_found, an account with no membership of it
// workspace.never_a_member; a failure is never one of those.
func TestReactivateMemberRefusals(t *testing.T) {
	failure := errors.New("connection reset")
	at := clockNow.Format(time.RFC3339Nano)
	reactivating := append(reactivationCalls(carol, acme), "Now", "ReactivateMember "+acme.ID.String()+" "+carol.ID.String()+" at "+at)
	tests := []struct {
		name        string
		slug, email string
		set         func(f *reactivateFixture)
		want        error
		calls       []string
	}{
		{"no such account", "acme", "nobody@corp.com", nil, domain.ErrAccountNotFound, []string{"ShareAccountByEmail nobody@corp.com"}},
		{"no such workspace", "gone", carol.Email, nil, domain.ErrSlugNotFound, []string{"ShareAccountByEmail " + carol.Email, "LockWorkspaceBySlug gone"}},
		{"deleted while the lock waited", "acme", carol.Email, func(f *reactivateFixture) {
			f.workspaces.lockErrs = map[string]error{"acme": app.ErrNotFound}
		}, domain.ErrSlugNotFound, []string{"ShareAccountByEmail " + carol.Email, "LockWorkspaceBySlug acme"}},
		{"never a member", "beta", alice.Email, nil, domain.ErrNeverAMember, reactivationCalls(alice, beta)},
		{"the account's lock failed", "acme", carol.Email, func(f *reactivateFixture) { f.accounts.err = failure }, failure,
			[]string{"ShareAccountByEmail " + carol.Email}},
		{"the workspace's lock failed", "acme", carol.Email, func(f *reactivateFixture) {
			f.workspaces.lockErrs = map[string]error{"acme": failure}
		}, failure, []string{"ShareAccountByEmail " + carol.Email, "LockWorkspaceBySlug acme"}},
		{"the membership's read failed", "acme", carol.Email, func(f *reactivateFixture) { f.workspaces.membersErr = failure }, failure,
			reactivationCalls(carol, acme)},
		{"the reactivation failed", "acme", carol.Email, func(f *reactivateFixture) { f.workspaces.restoreErr = failure }, failure, reactivating},
		{"the count failed", "acme", carol.Email, func(f *reactivateFixture) { f.counts.err = failure }, failure,
			append(reactivating, "CountInactive "+acme.ID.String()+" "+carol.ID.String())},
		{"the commit failed", "acme", carol.Email, func(f *reactivateFixture) { f.tx.commitErr = failure }, failure,
			append(reactivating, "CountInactive "+acme.ID.String()+" "+carol.ID.String())},
	}
	for _, tt := range tests {
		uc, f := newReactivate()
		if tt.set != nil {
			tt.set(f)
		}
		got, err := uc.Execute(t.Context(), tt.slug, tt.email)
		if !errors.Is(err, tt.want) || got != (app.Reactivation{}) {
			t.Errorf("%s: Execute() = %+v, %v; want %v", tt.name, got, err, tt.want)
		}
		if tt.want == failure && (errors.Is(err, domain.ErrAccountNotFound) || errors.Is(err, domain.ErrSlugNotFound) ||
			errors.Is(err, domain.ErrNeverAMember)) {
			t.Errorf("%s: Execute() = %v, which is also a refusal", tt.name, err)
		}
		if !slices.Equal(f.log.calls, tt.calls) || f.tx.calls != 1 || f.logs.Len() != 0 {
			t.Errorf("%s: calls = %q in %d transactions, logs %q; want %q in one, no log", tt.name, f.log.calls, f.tx.calls, f.logs, tt.calls)
		}
	}
}
````

`server/internal/modules/workspace/app/clock_test.go`（修改，2 处）：

````old server/internal/modules/workspace/app/clock_test.go
// projects' step, and a removal and a leaving for each step of the ending.
````
````new server/internal/modules/workspace/app/clock_test.go
// projects' step, and a removal and a leaving for each step of the ending.
// reactivate-member, which decides nothing, reads it after the workspace's
// lock and the membership's read.
````

````old server/internal/modules/workspace/app/clock_test.go
		}, slices.Concat(leavingCalls(bob, acme), []string{"Now"}, endingCalls(acme.ID, bob, bob.ID))},
````
````new server/internal/modules/workspace/app/clock_test.go
		}, slices.Concat(leavingCalls(bob, acme), []string{"Now"}, endingCalls(acme.ID, bob, bob.ID))},
		{"reactivate-member", func() ([]string, error) {
			uc, f := newReactivate()
			_, err := uc.Execute(t.Context(), "acme", carol.Email)
			return f.log.calls, err
		}, append(reactivationCalls(carol, acme), "Now", "ReactivateMember "+acme.ID.String()+" "+carol.ID.String()+" at "+at,
			"CountInactive "+acme.ID.String()+" "+carol.ID.String())},
````

- [ ] **Step 4: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/workspace/... ./internal/modules/project/...`
Expected: 全部 `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 5: 提交**

```bash
git add server/internal/modules/project/adapter/postgres/members.go server/internal/modules/project/adapter/postgres/members_test.go server/internal/modules/project/adapter/postgres/queries/members.sql server/internal/modules/project/module.go server/internal/modules/workspace/adapter/postgres/queries/members.sql server/internal/modules/workspace/adapter/postgres/reactivation.go server/internal/modules/workspace/adapter/postgres/reactivation_test.go server/internal/modules/workspace/app/clock_test.go server/internal/modules/workspace/app/create_workspace_test.go server/internal/modules/workspace/app/fakes_members_test.go server/internal/modules/workspace/app/fakes_projects_test.go server/internal/modules/workspace/app/fakes_workspaces_test.go server/internal/modules/workspace/app/ports.go server/internal/modules/workspace/app/reactivate_member.go server/internal/modules/workspace/app/reactivate_member_test.go server/internal/modules/workspace/domain/errors.go server/internal/modules/project/adapter/postgres/gen/members.sql.go server/internal/modules/workspace/adapter/postgres/gen/members.sql.go
```
```bash
git commit -m "feat(M3/P5a): ProjectMembershipCounts and the reactivation of a member

project.Provide gains ProjectMembershipCounts: an account's ended
memberships of a workspace's projects, counted without a lock. The
workspace's ReactivateMember locks the account's row FOR SHARE, a
deactivated one allowed, then the workspace FOR NO KEY UPDATE, reads
the membership, and makes an ended one active again, its role and its
last writer kept, at a moment read under the lock; an active one is
reported and left; it counts the project memberships still ended.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A）：

| 改坏 | 必须失败的测试 | 层 |
|---|---|---|
| `CountInactive` 去掉工作区、成员、`deleted_at IS NULL` | `TestCountInactive`；W12（Task 14 起） | 存储；端到端 |
| `CountInactive` 去掉 `NOT is_active` | `TestCountInactive`（故事看不到，spec 第 3 节第 9 条） | 存储 |
| `ReactivateMember` 去掉工作区、成员 | `TestReactivateMember`；W12（Task 14 起） | 存储；端到端 |
| `ReactivateMember` 去掉 `deleted_at IS NULL`；也改角色；把成员写成写者 | `TestReactivateMember` | 存储 |
| 已是有效的也写 | `TestReactivateMemberLeavesAnActiveMembership`；`TestWorkspacesReactivateMember`（Task 8 起） | 单元；组合 |
| 停用的账户被拒 | `TestReactivateMemberRestoresTheEndedMembership`；`TestWorkspacesReactivateMember`（Task 8 起） | 单元；组合 |
| 账户、工作区、成员关系的读的失败答成"没有"；恢复、计数的失败被吞掉 | `TestReactivateMemberRefusals` | 单元 |
| 先锁工作区再锁账户；时钟在工作区的锁之前读 | `TestReactivateMemberRestoresTheEndedMembership`；`TestEachLockOfAReactivationIsItsStrength`（Task 8 起） | 单元；组合 |
| `project.Provide` 的计数总是 0 | `TestWorkspacesReactivateMember`（Task 8 起） | 组合 |

**Done when:** 计数和恢复的存储测试、恢复的用例测试通过；`project.Provide` 提供计数。

---

### Task 8: `nerve workspaces reactivate-member`

**Files:**
- Create: `server/internal/bootstrap/reactivation_races_test.go`、`server/internal/bootstrap/reactivation_test.go`
- Modify: `server/cmd/nerve/workspaces.go`、`server/cmd/nerve/workspaces_test.go`、`server/internal/bootstrap/workspaces.go`、`server/internal/modules/workspace/admin.go`

**Interfaces:**
- Produces（spec 2.10，M3 设计 3.11、6.6、17.4）：
  - `workspace.AdminDeps.Counts app.ProjectMembershipCounts`；`(*Admin).ReactivateMember(ctx, slug, email) (app.Reactivation, error)`。`NewAdmin` 不建项目的连带（6.6：恢复不调用它）。
  - `bootstrap.Workspaces` 的组合多 `Counts: project.Provide(pool).ProjectMembershipCounts`；`bootstrap.ReactivateMember(slug, email) WorkspaceCommand` 输出一行：`reactivated <email> in <slug> as <admin|member|guest>; project memberships still ended: <n>, each restored when the member joins its project`；已是有效成员：`<email> is an active member of <slug> already; nothing changed`；账户已停用时行末加 `; the account is deactivated: run nerve users activate --email <email> next`。
  - `nerve workspaces reactivate-member --slug <slug> --email <address>`（两个都必填）。退出码 1 时 stderr 一行（`nerve: ` 加码的说明），stdout 为空。
  - 命令没有请求期限：等锁时一直等，中断（上下文结束，`cmd/nerve` 在 SIGINT、SIGTERM 时结束它）回滚（17.4 的 G1 (a)）。

**Tests:**
- `cmd/nerve/workspaces_test.go`：`TestWorkspacesReactivateMemberCommand`（恢复 lee，地址大写也规范化，输出一行、退出码 0、成员关系有效；没有账户、没有工作区、少了旗标：退出码 1，stderr 一行，stdout 为空）；`TestBareWorkspacesPrintsHelp` 的帮助列出 `reactivate-member`。
- `bootstrap/reactivation_test.go`（`endedMembers` 的准备：命令行建账户和工作区，存储写成员关系，然后照移出的语句结束 bob、carol 的，gone 删除，carol 的账户停用，dave 从来不是成员；准备完核对成员关系）：`TestWorkspacesReactivateMember`（bob 在 acme 恢复为有效，仍是管理员，他在 Web、Ops 的成员关系仍结束，输出说有 2 个；记一行日志；再跑一次说已是有效，什么都不改；carol 的账户已停用，照样恢复，输出说下一步）；`TestWorkspacesReactivateMemberErrors`（没有账户、没有工作区、已删除的工作区、从来不是成员的账户，以及在提交时被拒（每条语句都执行之后）：没有输出行、一行说明、每张表不变）。
- `bootstrap/reactivation_races_test.go`：`TestAReactivationFindsWhatChangedMeanwhile`（等锁时 acme 被删除："No workspace has this slug."，什么都不改；bob 的成员关系被恢复为有效（照接受邀请恢复的写法）：报告有效、什么都不改；等账户行时 bob 的账户被停用：照样恢复，输出说下一步；另一个事务改的行是它留下的样子，其余每一行不变）；`TestEachLockOfAReactivationIsItsStrength`（经命令的组合：等 acme 时只持账户的 `FOR SHARE`，不强也不弱；等成员关系时持账户和 acme 的 `FOR NO KEY UPDATE`；写下的时刻不早于 acme 被放开）；`TestAnInterruptedReactivationChangesNothing`（acme 被持有时命令一直等；中断让它失败、一行不改；行空出来之后再跑一次恢复 bob）；`TestTheReactivationRunsOnItsTransactionsConnection`（命令的组合在只有一个连接的池上 5 秒内完成）。

- [ ] **Step 1: `NewAdmin`、组合、命令**

`server/internal/modules/workspace/admin.go`（修改，5 处）：

````old server/internal/modules/workspace/admin.go
// AdminDeps are what the server administrator's commands need: the pool and
// identity's Accounts, and no Authorizer, signing key or jobs client (M3
// design 6.6).
````
````new server/internal/modules/workspace/admin.go
// AdminDeps are what the server administrator's commands need: the pool,
// identity's Accounts and project's ProjectMembershipCounts, and no
// Authorizer, signing key, project cascade or jobs client (M3 design 6.6).
````

````old server/internal/modules/workspace/admin.go
	Accounts app.Accounts
````
````new server/internal/modules/workspace/admin.go
	Accounts app.Accounts
	Counts   app.ProjectMembershipCounts
````

````old server/internal/modules/workspace/admin.go
	create *app.CreateWorkspace
````
````new server/internal/modules/workspace/admin.go
	create     *app.CreateWorkspace
	reactivate *app.ReactivateMember
````

````old server/internal/modules/workspace/admin.go
	return &Admin{create: app.NewCreateWorkspace(app.CreateWorkspaceDeps{
		Accounts: d.Accounts, Workspaces: postgresadapter.New(d.Pool), Tx: d.Tx, Clock: d.Clock, Logger: d.Logger,
	})}
````
````new server/internal/modules/workspace/admin.go
	store := postgresadapter.New(d.Pool)
	return &Admin{
		create: app.NewCreateWorkspace(app.CreateWorkspaceDeps{
			Accounts: d.Accounts, Workspaces: store, Tx: d.Tx, Clock: d.Clock, Logger: d.Logger,
		}),
		reactivate: app.NewReactivateMember(d.Accounts, store, d.Counts, d.Tx, d.Clock, d.Logger),
	}
````

````old server/internal/modules/workspace/admin.go
	return a.create.ExecuteForAdmin(ctx, adminEmail, domain.NewWorkspace{Name: name, Slug: slug})
}

````
````new server/internal/modules/workspace/admin.go
	return a.create.ExecuteForAdmin(ctx, adminEmail, domain.NewWorkspace{Name: name, Slug: slug})
}

// ReactivateMember is `nerve workspaces reactivate-member`: the ended
// membership of the account of email in the workspace slug active again,
// its role kept, also while the account is deactivated.
func (a *Admin) ReactivateMember(ctx context.Context, slug, email string) (app.Reactivation, error) {
	return a.reactivate.Execute(ctx, slug, email)
}

````

`server/internal/bootstrap/workspaces.go`（修改，5 处）：

````old server/internal/bootstrap/workspaces.go
	"context"
````
````new server/internal/bootstrap/workspaces.go
	"context"
	"fmt"
````

````old server/internal/bootstrap/workspaces.go
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
````
````new server/internal/bootstrap/workspaces.go
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
````

````old server/internal/bootstrap/workspaces.go
// identity's Accounts and workspace's administrator use cases; no HTTP
// server, no Authorizer, no signing key, no jobs client. The command's line
// goes to out, the logs to logOut. An error is one line for the
// administrator, and the database is unchanged.
````
````new server/internal/bootstrap/workspaces.go
// identity's Accounts, project's ProjectMembershipCounts and workspace's
// administrator use cases; no HTTP server, no Authorizer, no signing key,
// no project cascade, no jobs client. The command's line goes to out, the
// logs to logOut. An error is one line for the administrator, and the
// database is unchanged.
````

````old server/internal/bootstrap/workspaces.go
		Accounts: workspaceAccounts{accounts: identity.Provide(pool).Accounts},
````
````new server/internal/bootstrap/workspaces.go
		Accounts: workspaceAccounts{accounts: identity.Provide(pool).Accounts},
		Counts:   project.Provide(pool).ProjectMembershipCounts,
````

````old server/internal/bootstrap/workspaces.go
	}
}

````
````new server/internal/bootstrap/workspaces.go
	}
}

// roleNames are the workspace roles as the commands print them.
var roleNames = map[shared.Role]string{shared.RoleAdmin: "admin", shared.RoleMember: "member", shared.RoleGuest: "guest"}

// ReactivateMember is `nerve workspaces reactivate-member` (M3 design 3.11,
// as Plane's reactivate_workspace_member): the line says the role kept and
// how many of the member's project memberships stay ended, or that the
// membership was active already; for a deactivated account, that `nerve
// users activate` is next.
func ReactivateMember(slug, email string) WorkspaceCommand {
	return func(ctx context.Context, admin *workspace.Admin) (string, error) {
		r, err := admin.ReactivateMember(ctx, slug, email)
		line := fmt.Sprintf("reactivated %s in %s as %s; project memberships still ended: %d, each restored when the member joins its project",
			r.Email, slug, roleNames[r.Role], r.EndedProjectMemberships)
		if r.AlreadyActive {
			line = fmt.Sprintf("%s is an active member of %s already; nothing changed", r.Email, slug)
		}
		if !r.AccountActive {
			line += "; the account is deactivated: run nerve users activate --email " + r.Email + " next"
		}
		return line, err
	}
}

````

`server/cmd/nerve/workspaces.go`（修改，2 处）：

````old server/cmd/nerve/workspaces.go
	workspaces.AddCommand(createWorkspaceCommand(load))
````
````new server/cmd/nerve/workspaces.go
	workspaces.AddCommand(createWorkspaceCommand(load), reactivateMemberCommand(load))
````

````old server/cmd/nerve/workspaces.go
	return cmd
}

````
````new server/cmd/nerve/workspaces.go
	return cmd
}

// reactivateMemberCommand is `nerve workspaces reactivate-member`: an ended
// membership active again, its role kept; the member's project memberships
// stay ended until he joins each project.
func reactivateMemberCommand(load configLoader) *cobra.Command {
	var slug, email string
	cmd := &cobra.Command{
		Use:   "reactivate-member --slug <slug> --email <address>",
		Short: "Make an account's ended membership of a workspace active again, its role kept; its project memberships stay ended",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := load()
			if err != nil {
				return err
			}
			return bootstrap.Workspaces(cmd.Context(), cfg, cmd.ErrOrStderr(), cmd.OutOrStdout(), bootstrap.ReactivateMember(slug, email))
		},
	}
	cmd.Flags().StringVar(&slug, "slug", "", "the workspace's slug, its address")
	cmd.Flags().StringVar(&email, "email", "", "the e-mail address of the member's account")
	_ = cmd.MarkFlagRequired("slug") // the flags exist
	_ = cmd.MarkFlagRequired("email")
	return cmd
}

````

`server/cmd/nerve/workspaces_test.go`（修改，2 处）：

````old server/cmd/nerve/workspaces_test.go
}

func TestBareWorkspacesPrintsHelp(t *testing.T) {
````
````new server/cmd/nerve/workspaces_test.go
}

// `nerve workspaces reactivate-member` makes an ended membership active
// again and prints one line; a refused one exits 1 with one line on stderr
// and nothing on stdout (M3 design 3.11). That a refusal changes nothing is
// bootstrap's TestWorkspacesReactivateMemberErrors.
func TestWorkspacesReactivateMemberCommand(t *testing.T) {
	environ, pool := usersDatabase(t)
	for _, email := range []string{"nia@corp.com", "lee@corp.com"} {
		if code, _, stderr := executeWithInput(context.Background(), environ, "Tr0ub4dor&3\n", "users", "create", "--email", email); code != 0 {
			t.Fatalf("create the account of %s = %d: %s", email, code, stderr)
		}
	}
	if code, _, stderr := execute(context.Background(), environ,
		"workspaces", "create", "--slug", "acme", "--name", "Acme", "--admin-email", "nia@corp.com"); code != 0 {
		t.Fatalf("create acme = %d: %s", code, stderr)
	}
	// lee's ended membership: a fixture, which the store test and the
	// bootstrap test make as a removal does.
	if _, err := pool.Exec(context.Background(), `INSERT INTO workspace_members (id, workspace_id, member_id, role, is_active)
		SELECT gen_random_uuid(), w.id, u.id, 15, false FROM workspaces w, users u WHERE w.slug = 'acme' AND u.email = 'lee@corp.com'`); err != nil {
		t.Fatal(err)
	}

	code, stdout, stderr := execute(context.Background(), environ, "workspaces", "reactivate-member", "--slug", "acme", "--email", "LEE@corp.com")

	if want := "reactivated lee@corp.com in acme as member; project memberships still ended: 0, each restored when the member joins its project\n"; code != 0 ||
		stdout != want {
		t.Fatalf("nerve workspaces reactivate-member = %d %q (stderr %q), want 0 and %q", code, stdout, stderr, want)
	}
	var active bool
	if err := pool.QueryRow(context.Background(), `SELECT m.is_active FROM workspace_members m JOIN users u ON u.id = m.member_id
		WHERE u.email = 'lee@corp.com'`).Scan(&active); err != nil || !active {
		t.Errorf("lee's membership active = %v (%v), want true", active, err)
	}

	tests := []struct {
		name string
		args []string
		want string
	}{
		{"an unknown account", []string{"workspaces", "reactivate-member", "--slug", "acme", "--email", "may@corp.com"},
			"nerve: No account has this e-mail address.\n"},
		{"an unknown workspace", []string{"workspaces", "reactivate-member", "--slug", "beta", "--email", "lee@corp.com"},
			"nerve: No workspace has this slug.\n"},
		{"no email", []string{"workspaces", "reactivate-member", "--slug", "acme"}, "nerve: required flag(s) \"email\" not set\n"},
		{"no slug and no email", []string{"workspaces", "reactivate-member"}, "nerve: required flag(s) \"email\", \"slug\" not set\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stdout, stderr := execute(context.Background(), environ, tt.args...)
			if code != 1 || stdout != "" || !strings.HasSuffix(stderr, tt.want) {
				t.Errorf("nerve %s = %d, stdout %q, stderr %q; want 1 and %q", strings.Join(tt.args, " "), code, stdout, stderr, tt.want)
			}
		})
	}
}

func TestBareWorkspacesPrintsHelp(t *testing.T) {
````

````old server/cmd/nerve/workspaces_test.go
	if code != 0 || !strings.Contains(stdout, "create") || stderr != "" {
````
````new server/cmd/nerve/workspaces_test.go
	if code != 0 || !strings.Contains(stdout, "create") || !strings.Contains(stdout, "reactivate-member") || stderr != "" {
````

- [ ] **Step 2: 真实数据库上的命令**

`server/internal/bootstrap/reactivation_test.go`（新文件，183 行）：

````file server/internal/bootstrap/reactivation_test.go
package bootstrap

import (
	"context"
	"maps"
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
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// endedMembers prepares the database at url: through the command line,
// the accounts of alice, bob, carol and dave, and acme and gone, alice the
// admin of each; through the stores, bob acme's admin and carol its guest,
// bob the admin of Web and Ops, carol Web's guest; then bob's and carol's
// memberships of acme and of its projects ended by alice at one moment, as
// a removal ends them (M3 design 3.6, 3.7); gone deleted; carol's account
// deactivated. dave was never a member of acme.
func endedMembers(t *testing.T, url string) *pgxpool.Pool {
	t.Helper()
	pool := openPool(t, url)
	ids := map[string]uuid.UUID{}
	for _, name := range []string{"alice", "bob", "carol", "dave"} {
		ids[name] = uuid.MustParse(createdAccount(t, url, pool, name+"@corp.com"))
	}
	for _, slug := range []string{"acme", "gone"} {
		if _, _, err := runWorkspaces(t, url, CreateWorkspace(slug, slug, "alice@corp.com")); err != nil {
			t.Fatal(err)
		}
	}
	workspaces, projects, ctx, now := workspacepg.New(pool), projectpg.New(pool), context.Background(), time.Now()
	workspaceID := func(slug string) uuid.UUID {
		w, err := workspaces.WorkspaceBySlug(ctx, slug)
		if err != nil {
			t.Fatal(err)
		}
		return w.ID
	}
	acme, gone := workspaceID("acme"), workspaceID("gone")
	var acmes []uuid.UUID
	for _, p := range []struct{ name, identifier string }{{"Web", "WEB"}, {"Ops", "OPS"}} {
		id := uuid.NewV7()
		if err := projects.CreateProject(ctx, projectapp.ProjectRow{ID: id, WorkspaceID: acme, Name: p.name, Identifier: p.identifier,
			Network: projectdomain.NetworkPublic, Timezone: "UTC", CreatedBy: ids["alice"], Now: now}); err != nil {
			t.Fatal(err)
		}
		acmes = append(acmes, id)
	}
	for _, m := range []struct {
		name     string
		role     shared.Role
		projects []uuid.UUID
	}{{"bob", shared.RoleAdmin, acmes}, {"carol", shared.RoleGuest, acmes[:1]}} {
		if err := workspaces.CreateMember(ctx, workspaceapp.MemberRow{ID: uuid.NewV7(), WorkspaceID: acme, MemberID: ids[m.name], Role: m.role,
			CreatedBy: ids["alice"], Now: now}); err != nil {
			t.Fatal(err)
		}
		for _, p := range m.projects {
			if err := projects.CreateMember(ctx, projectapp.MemberRow{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: p, MemberID: ids[m.name],
				Role: m.role, CreatedBy: ids["alice"], Now: now}); err != nil {
				t.Fatal(err)
			}
		}
		if err := workspaces.EndMember(ctx, acme, ids[m.name], ids["alice"], now); err != nil {
			t.Fatal(err)
		}
		if err := projects.EndMemberships(ctx, m.projects, ids[m.name], ids["alice"], now); err != nil {
			t.Fatal(err)
		}
	}
	if err := workspaces.DeleteWorkspace(ctx, gone, ids["alice"], now); err != nil {
		t.Fatal(err)
	}
	if _, _, err := runUsers(t, url, DeactivateUser("carol@corp.com")); err != nil {
		t.Fatal(err)
	}
	return pool
}

// memberStates are the memberships of acme's members and of its projects,
// one line each, in byte order: the account, the workspace's or the
// project's, the role and whether it is active.
func memberStates(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var rows string
	if err := pool.QueryRow(context.Background(), `SELECT string_agg(s, E'\n' ORDER BY s COLLATE "C") FROM (
		SELECT u.email || ' acme ' || m.role || ' ' || m.is_active AS s FROM workspace_members m JOIN users u ON u.id = m.member_id
		JOIN workspaces w ON w.id = m.workspace_id WHERE w.slug = 'acme'
		UNION ALL SELECT u.email || ' ' || p.name || ' ' || m.role || ' ' || m.is_active FROM project_members m
		JOIN users u ON u.id = m.member_id JOIN projects p ON p.id = m.project_id) r`).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	return rows
}

// `nerve workspaces reactivate-member` runs on the minimal composition (M3
// design 3.11, 6.6), the address normalized: bob's membership of acme is
// active again, an admin's still, and his memberships of Web and Ops stay
// ended, as the line says; the reactivation is logged once. Run again, it
// says the membership is active and changes nothing. carol's, her account
// deactivated, is reactivated all the same, and the line says what is
// next.
func TestWorkspacesReactivateMember(t *testing.T) {
	url := pgtest.NewDatabase(t)
	pool := endedMembers(t, url)
	if got, want := memberStates(t, pool), strings.Join([]string{"alice@corp.com acme 20 true", "bob@corp.com Ops 20 false",
		"bob@corp.com Web 20 false", "bob@corp.com acme 20 false", "carol@corp.com Web 5 false", "carol@corp.com acme 5 false"}, "\n"); got != want {
		t.Fatalf("the memberships before:\n%s\nwant\n%s", got, want)
	}

	out, logs, err := runWorkspaces(t, url, ReactivateMember("acme", " Bob@Corp.COM "))

	if want := "reactivated bob@corp.com in acme as admin; project memberships still ended: 2, each restored when the member joins its project\n"; err != nil ||
		out != want {
		t.Fatalf("reactivate-member = %q, %v; want %q", out, err, want)
	}
	if strings.Count(logs, `msg="workspace member reactivated"`) != 1 || !strings.Contains(logs, "by=cli") {
		t.Errorf("logs = %s, want the reactivation logged once, by cli", logs)
	}
	if got, want := memberStates(t, pool), strings.Join([]string{"alice@corp.com acme 20 true", "bob@corp.com Ops 20 false",
		"bob@corp.com Web 20 false", "bob@corp.com acme 20 true", "carol@corp.com Web 5 false", "carol@corp.com acme 5 false"}, "\n"); got != want {
		t.Errorf("the memberships after:\n%s\nwant\n%s", got, want)
	}
	before := tableRows(t, pool, riversOwn)
	if out, _, err := runWorkspaces(t, url, ReactivateMember("acme", "bob@corp.com")); err != nil ||
		out != "bob@corp.com is an active member of acme already; nothing changed\n" {
		t.Errorf("reactivate-member again = %q, %v; want the membership reported active", out, err)
	}
	if after := tableRows(t, pool, riversOwn); !maps.Equal(after, before) {
		t.Errorf("the tables after reactivating an active membership changed:\n%v\nwant them as they were:\n%v", after, before)
	}
	out, _, err = runWorkspaces(t, url, ReactivateMember("acme", "carol@corp.com"))
	if want := "reactivated carol@corp.com in acme as guest; project memberships still ended: 1, each restored when the member joins its " +
		"project; the account is deactivated: run nerve users activate --email carol@corp.com next\n"; err != nil || out != want {
		t.Errorf("reactivate-member of carol = %q, %v; want %q", out, err, want)
	}
}

// A refused reactivation prints no line, says why in one line, and leaves
// every table as it was (M3 design 3.11): no such account, no such
// workspace, a deleted one, an account never its member; and a
// reactivation refused at its commit, after every statement ran, so that
// no step wrote in a transaction of its own.
func TestWorkspacesReactivateMemberErrors(t *testing.T) {
	url := pgtest.NewDatabase(t)
	pool := endedMembers(t, url)
	before := tableRows(t, pool, riversOwn)
	tests := []struct {
		name, slug, email string
		refuse            bool // the commit of a write of workspace_members
		want              string
	}{
		{"an unknown account", "acme", "nobody@corp.com", false, "No account has this e-mail address."},
		{"an unknown workspace", "beta", "bob@corp.com", false, "No workspace has this slug."},
		{"a deleted workspace", "gone", "alice@corp.com", false, "No workspace has this slug."},
		{"never a member", "acme", "dave@corp.com", false, "The account has never been a member of this workspace."},
		{"the commit refused", "acme", "bob@corp.com", true, "the commit is refused"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.refuse {
				defer refusingCommits(t, pool, "workspace_members")()
			}
			out, _, err := runWorkspaces(t, url, ReactivateMember(tt.slug, tt.email))
			if err == nil || !strings.Contains(err.Error(), tt.want) || out != "" {
				t.Errorf("= %q, %v; want no line and %q", out, err, tt.want)
			}
			if after := tableRows(t, pool, riversOwn); !maps.Equal(after, before) {
				t.Errorf("the tables changed:\n%v\nwant them as they were:\n%v", after, before)
			}
		})
	}
}
````

`server/internal/bootstrap/reactivation_races_test.go`（新文件，227 行）：

````file server/internal/bootstrap/reactivation_races_test.go
package bootstrap

import (
	"bytes"
	"context"
	"maps"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// A reactivation through the command's composition against what another
// transaction changes or holds meanwhile (M3 design 3.6's lock table,
// 3.11): what it finds once it has a lock, each lock it takes, at its
// strength, and the connection it runs on. endedMembers prepares each
// database: bob's membership of acme, an admin's, and his memberships of
// Web and Ops ended.

// commandRun is a command's line and error.
type commandRun struct {
	out string
	err error
}

// bobReactivated is the line of bob's reactivation in acme.
const bobReactivated = "reactivated bob@corp.com in acme as admin; project memberships still ended: 2, each restored when the member joins its project\n"

// reactivatingBob runs `nerve workspaces reactivate-member --slug acme
// --email bob@corp.com` on the database at url, on a pool of maxConns
// connections, in the background until ctx ends, and hands over its line
// and error.
func reactivatingBob(ctx context.Context, t *testing.T, url string, maxConns int32) <-chan commandRun {
	t.Helper()
	cfg := testConfig(t, url, false)
	cfg.Database.MaxConns = maxConns
	done := make(chan commandRun, 1)
	go func() {
		var out, logs bytes.Buffer
		err := Workspaces(ctx, cfg, &logs, &out, ReactivateMember("acme", "bob@corp.com"))
		done <- commandRun{out.String(), err}
	}()
	return done
}

// endedIDs are the ids of acme, of bob's account and of his membership of
// acme.
type endedIDs struct {
	acme, bob, bobs uuid.UUID
}

func idsOf(t *testing.T, pool *pgxpool.Pool) endedIDs {
	t.Helper()
	var ids endedIDs
	if err := pool.QueryRow(context.Background(), `SELECT w.id, u.id, m.id FROM workspace_members m JOIN workspaces w ON w.id = m.workspace_id
		JOIN users u ON u.id = m.member_id WHERE w.slug = 'acme' AND u.email = 'bob@corp.com'`).Scan(&ids.acme, &ids.bob, &ids.bobs); err != nil {
		t.Fatal(err)
	}
	return ids
}

// A reactivation that waits for a lock reads, once it has it, what another
// transaction committed meanwhile, and answers for that (M3 design 3.6,
// 3.11): acme deleted while it waits for acme's row is "No workspace has
// this slug.", and nothing changes; bob's membership made active again
// meanwhile, as accepting an invitation restores it, is reported active,
// and nothing changes; bob's account deactivated while it waits for his
// account's row is reactivated all the same, his membership with it, and
// the line says what is next. The row the other changed is as it left it,
// bob's membership as the case says, every other row as it was.
func TestAReactivationFindsWhatChangedMeanwhile(t *testing.T) {
	for _, tt := range []struct {
		name, holds, waitsOn, change string
		table                        string                       // the table of the row the other changes
		row                          func(ids endedIDs) uuid.UUID // that row
		out, err                     string
		active                       bool // bob's membership of acme after it
	}{
		{"acme deleted", "SELECT 1 FROM workspaces WHERE id = $1 AND $2::uuid IS NOT NULL FOR NO KEY UPDATE", "workspaces",
			"UPDATE workspaces SET deleted_at = now(), updated_at = now() WHERE id = $1 AND $2::uuid IS NOT NULL", "workspaces",
			func(ids endedIDs) uuid.UUID { return ids.acme }, "", "No workspace has this slug.", false},
		{"his membership active again", "SELECT 1 FROM workspaces WHERE id = $1 AND $2::uuid IS NOT NULL FOR NO KEY UPDATE", "workspaces",
			"UPDATE workspace_members SET is_active = true, updated_at = now() WHERE workspace_id = $1 AND member_id = $2", "workspace_members",
			func(ids endedIDs) uuid.UUID { return ids.bobs }, "bob@corp.com is an active member of acme already; nothing changed\n", "", true},
		{"his account deactivated", "SELECT 1 FROM users WHERE id = $2 AND $1::uuid IS NOT NULL FOR NO KEY UPDATE", "users",
			"UPDATE users SET is_active = false, updated_at = now() WHERE id = $2 AND $1::uuid IS NOT NULL", "users",
			func(ids endedIDs) uuid.UUID { return ids.bob },
			strings.TrimSuffix(bobReactivated, "\n") + "; the account is deactivated: run nerve users activate --email bob@corp.com next\n", "", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			url := pgtest.NewDatabase(t)
			pool := endedMembers(t, url)
			ids := idsOf(t, pool)
			table, id := tt.table, tt.row(ids)
			others := rowsBut(t, pool, []uuid.UUID{id, ids.bobs})
			other := holding(t, pool, tt.holds, ids.acme, ids.bob)
			if tag, err := other.Exec(context.Background(), tt.change, ids.acme, ids.bob); err != nil || tag.RowsAffected() != 1 {
				t.Fatalf("%s: %v, %v; want one row changed", tt.change, tag, err)
			}
			changed := rowJSON(t, other, table, id)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			done := reactivatingBob(ctx, t, url, 4)
			pgtest.WaitForLockWaitOn(t, pool, tt.waitsOn, 5*time.Second)
			if err := other.Commit(context.Background()); err != nil {
				t.Fatal(err)
			}

			run := receiveWithin(t, done, 10*time.Second, "reactivate-member's end")
			if run.out != tt.out || (tt.err == "") != (run.err == nil) || (run.err != nil && !strings.Contains(run.err.Error(), tt.err)) {
				t.Errorf("reactivate-member = %q, %v; want %q, %q", run.out, run.err, tt.out, tt.err)
			}
			if after := rowJSON(t, pool, table, id); !maps.Equal(after, changed) {
				t.Errorf("%s %s after it:\n%v\nwant it as the other transaction left it:\n%v", table, id, after, changed)
			}
			if active := rowJSON(t, pool, "workspace_members", ids.bobs)["is_active"]; active != tt.active {
				t.Errorf("bob's membership of acme after it: active %v, want %v", active, tt.active)
			}
			if after := rowsBut(t, pool, []uuid.UUID{id, ids.bobs}); !maps.Equal(after, others) {
				t.Errorf("every other row after it:\n%v\nwant them as they were:\n%v", after, others)
			}
		})
	}
}

// Each lock of a reactivation is taken in the lock table's order, at its
// strength, through the command's composition (M3 design 3.6's lock table,
// convention 6): bob's account row FOR SHARE, then acme's row FOR NO KEY
// UPDATE, then his membership. Other transactions hold acme's row FOR NO
// KEY UPDATE and his membership FOR SHARE, and let go one at a time;
// lockOn reads each row's strongest lock then. The command waits for
// acme's row holding his account FOR SHARE, no stronger, no weaker, and
// nothing else; then for his membership, holding his account and acme's
// row FOR NO KEY UPDATE. Then it reactivates him, at a moment no earlier
// than acme's release: it read the clock under acme's lock (3.3).
func TestEachLockOfAReactivationIsItsStrength(t *testing.T) {
	url := pgtest.NewDatabase(t)
	pool := endedMembers(t, url)
	ids := idsOf(t, pool)
	locks := func() string {
		t.Helper()
		return "his account " + lockOn(t, pool, "users WHERE id = $1", ids.bob) + ", acme " + lockOn(t, pool, "workspaces WHERE id = $1", ids.acme) +
			", his membership " + lockOn(t, pool, "workspace_members WHERE id = $1", ids.bobs)
	}
	holdsMembership := holding(t, pool, "SELECT 1 FROM workspace_members WHERE id = $1 FOR SHARE", ids.bobs)
	holdsAcme := holding(t, pool, "SELECT 1 FROM workspaces WHERE id = $1 FOR NO KEY UPDATE", ids.acme)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	done := reactivatingBob(ctx, t, url, 4)
	pgtest.WaitForLockWaitOn(t, pool, "workspaces", 5*time.Second)
	// acme's lock here is the holder's, and so is his membership's FOR SHARE.
	if got, want := locks(), "his account FOR SHARE, acme FOR NO KEY UPDATE, his membership FOR SHARE"; got != want {
		t.Errorf("the reactivation waiting for acme's row: %s; want %s", got, want)
	}
	released := time.Now()
	if err := holdsAcme.Rollback(context.Background()); err != nil {
		t.Fatal(err)
	}
	pgtest.WaitForLockWaitOn(t, pool, "workspace_members", 5*time.Second)
	if got, want := locks(), "his account FOR SHARE, acme FOR NO KEY UPDATE, his membership FOR SHARE"; got != want {
		t.Errorf("the reactivation waiting for his membership: %s; want %s", got, want)
	}
	if err := holdsMembership.Rollback(context.Background()); err != nil {
		t.Fatal(err)
	}

	if run := receiveWithin(t, done, 10*time.Second, "reactivate-member's end"); run.out != bobReactivated || run.err != nil {
		t.Fatalf("reactivate-member = %q, %v; want %q", run.out, run.err, bobReactivated)
	}
	moment, _ := rowJSON(t, pool, "workspace_members", ids.bobs)["updated_at"].(string)
	if at, err := time.Parse(time.RFC3339Nano, moment); err != nil || at.Before(released.Truncate(time.Microsecond)) {
		t.Errorf("the reactivation's moment %q (%v); want one no earlier than acme's release, %v", moment, err, released)
	}
}

// The command has no deadline of its own: while acme's row is held it
// waits, and an interruption, its context's end as SIGINT or SIGTERM ends
// it (cmd/nerve), rolls it back: it fails, and no row changes. Run again
// once the row is free, it reactivates bob (README, M3 design 17.4).
func TestAnInterruptedReactivationChangesNothing(t *testing.T) {
	url := pgtest.NewDatabase(t)
	pool := endedMembers(t, url)
	before := tableRows(t, pool, riversOwn)
	holdsAcme := holding(t, pool, "SELECT 1 FROM workspaces WHERE slug = 'acme' FOR NO KEY UPDATE")
	ctx, interrupt := context.WithCancel(context.Background())
	defer interrupt()
	done := reactivatingBob(ctx, t, url, 4)
	pgtest.WaitForLockWaitOn(t, pool, "workspaces", 5*time.Second)
	interrupt()

	if run := receiveWithin(t, done, 10*time.Second, "reactivate-member's end"); run.err == nil || run.out != "" {
		t.Errorf("reactivate-member interrupted = %q, %v; want no line and an error", run.out, run.err)
	}
	if err := holdsAcme.Rollback(context.Background()); err != nil {
		t.Fatal(err)
	}
	if after := tableRows(t, pool, riversOwn); !maps.Equal(after, before) {
		t.Errorf("the tables after the interrupted reactivation changed:\n%v\nwant them as they were:\n%v", after, before)
	}
	again, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if run := receiveWithin(t, reactivatingBob(again, t, url, 4), 10*time.Second, "reactivate-member's end"); run.out != bobReactivated ||
		run.err != nil {
		t.Errorf("reactivate-member again = %q, %v; want %q", run.out, run.err, bobReactivated)
	}
}

// The reactivation runs every statement on its transaction's connection
// (M3 design 3.6 convention 2): the account's lock, the workspace's, the
// membership's read, the reactivation and the count of his ended project
// memberships through ProjectMembershipCounts. The command's composition
// runs on a pool of one connection: a statement sent through the pool
// rather than the transaction would wait for a second connection until the
// command's context ends, after 5 seconds, and the command fail.
func TestTheReactivationRunsOnItsTransactionsConnection(t *testing.T) {
	url := pgtest.NewDatabase(t)
	endedMembers(t, url)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if run := receiveWithin(t, reactivatingBob(ctx, t, url, 1), 10*time.Second, "reactivate-member's end"); run.out != bobReactivated || run.err != nil {
		t.Errorf("reactivate-member on a pool of one connection = %q, %v; want %q", run.out, run.err, bobReactivated)
	}
}
````

- [ ] **Step 3: 测试和 lint**

Run: `go -C server test -count=1 ./cmd/... ./internal/modules/workspace/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestWorkspacesReactivateMember|TestAReactivationFindsWhatChangedMeanwhile|TestEachLockOfAReactivationIsItsStrength|TestAnInterruptedReactivationChangesNothing|TestTheReactivationRunsOnItsTransactionsConnection|TestWorkspaces' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 4: 提交**

```bash
git add server/cmd/nerve/workspaces.go server/cmd/nerve/workspaces_test.go server/internal/bootstrap/reactivation_races_test.go server/internal/bootstrap/reactivation_test.go server/internal/bootstrap/workspaces.go server/internal/modules/workspace/admin.go
```
```bash
git commit -m "feat(M3/P5a): nerve workspaces reactivate-member

The administrator's command makes an account's ended membership of a
workspace active again, its role kept, also while the account is
deactivated, and prints how many of its project memberships stay ended
and, for a deactivated account, that nerve users activate is next. It
runs on the minimal composition, project's ProjectMembershipCounts
added and no project cascade; an unknown workspace or account, or an
account never a member, exits 1 and changes nothing.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A）：

| 改坏 | 必须失败的测试 | 层 |
|---|---|---|
| `NewAdmin` 的计数总是 0；组合的计数总是 0 | `TestWorkspacesReactivateMember` | 组合 |
| 组合把每个账户都当作有效 | `TestWorkspacesReactivateMember` | 组合 |
| `NewAdmin` 不开事务 | `TestEachLockOfAReactivationIsItsStrength` | 组合 |
| `NewAdmin` 用停在 2001 年的时钟 | `TestEachLockOfAReactivationIsItsStrength` | 组合 |
| 账户行 `FOR KEY SHARE`、`FOR NO KEY UPDATE` | `TestEachLockOfAReactivationIsItsStrength` | 组合 |
| `LockWorkspaceBySlug` 锁成 `FOR SHARE`；不加锁 | `TestEachLockOfAReactivationIsItsStrength`；`TestAReactivationFindsWhatChangedMeanwhile`（不加锁） | 组合 |
| 先锁工作区再锁账户；时钟在工作区的锁之前读 | `TestEachLockOfAReactivationIsItsStrength` | 组合 |
| 账户的锁、工作区的锁、成员关系的读、恢复、计数各经连接池 | `TestTheReactivationRunsOnItsTransactionsConnection` | 组合 |

**Done when:** 命令在真实数据库上恢复、报告已是有效、提示停用的账户；退出码 1 的每种情形数据库不变；每把锁的强度和顺序在命令的组合上钉住。

---

### Task 9: 交错 1 的工作区一侧、交错 4

**Files:**
- Create: `server/internal/bootstrap/interleaving_endings_test.go`
- Modify: `server/internal/bootstrap/interleaving_growth_test.go`

**Interfaces:** 没有新的产品代码。`interleaving_endings_test.go`：`endedHolding`（结束一步写完成员关系之后停在 gate，持工作区行和成员行，在项目一步之前）；`adminRace.leave`；`growthRace.remove`（照 `bootstrap` 的接法，系统时钟）；`refusedAsNoMember`。`interleaving_growth_test.go` 的 `standing` 也写出他在 acme 的成员关系是否已结束。每个等待都有期限，第二方等在工作区行上由 `pgtest.WaitForLockWaitOn(…, "workspaces", …)` 证明（spec 2.11 列出每个顺序的探针和它的反例）。

**Tests:**
- `TestTwoAdminsLeavingLeaveAnAdmin`（交错 1 的工作区一侧，两种顺序：第一位持 acme 的 `FOR NO KEY UPDATE`、过了另一个管理员的检查、在 gate 处已结束自己的成员关系；第二位等 acme 的行；第一位提交之后第二位找不到另一个有效管理员：409 `workspace.sole_admin`；第一位的成员关系结束、第二位的有效：acme 留有一位管理员）。
- `TestARemovalAndTheProjectSidesGrowthSerialize`（交错 4，添加和加入、新的和已结束的成员关系、两种顺序，8 个子测试：增长先：它持 acme 和他在 acme 的成员关系的 `FOR SHARE`，不更强（`lockOn`），在锁 Web 之前停在 gate；移出等 acme 的行，什么都不持；增长提交之后，移出的项目一步找到他在 Web 的成员关系并结束它，时刻是移出持有 acme 之后读的，不早于 gate 放开（3.3）。移出先：它持 acme 和他的成员行；增长等 acme 的行，然后读到他的成员关系已结束：alice 添加他是 422 `members[0].member_id` `not_allowed`，他加入是 404；他在 Web 的成员关系如前。两种顺序里第二方等 acme 时没有谁持 Web（`FOR UPDATE NOWAIT`））。

- [ ] **Step 1: 交错测试**

`server/internal/bootstrap/interleaving_growth_test.go`（修改，2 处）：

````old server/internal/bootstrap/interleaving_growth_test.go
// standing is bob's role in acme, then his membership of Web: its role,
// "ended" when it is, "deleted" when Web is; "none" when he has none.
````
````new server/internal/bootstrap/interleaving_growth_test.go
// standing is bob's role in acme, "ended" when his membership is, then his
// membership of Web: its role, "ended" when it is, "deleted" when Web is;
// "none" when he has none.
````

````old server/internal/bootstrap/interleaving_growth_test.go
	if err := r.pool.QueryRow(context.Background(), `SELECT (SELECT role::text FROM workspace_members WHERE id = $1) || ', Web ' ||
````
````new server/internal/bootstrap/interleaving_growth_test.go
	if err := r.pool.QueryRow(context.Background(), `SELECT (SELECT role || CASE WHEN is_active THEN '' ELSE ' ended' END
		FROM workspace_members WHERE id = $1) || ', Web ' ||
````

`server/internal/bootstrap/interleaving_endings_test.go`（新文件，220 行）：

````file server/internal/bootstrap/interleaving_endings_test.go
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

	"github.com/open-nerve/NerveProject/server/internal/modules/access"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
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

// The interleavings 1 (its workspace side) and 4 of M3 design 9.3, on a
// real database, in both orders: an ending of a workspace membership, a
// leaving or a removal, through workspace's use case with project's
// cascade, identity's profiles and the Authorizer as bootstrap wires them,
// against another leaving, or against the project side's growth through
// the project module behind the API. A gate inside the first side's
// transaction holds it open at a lock the second side needs;
// pgtest.WaitForLockWaitOn proves that the second side waits on that
// table's row before the gate opens. Every wait has a deadline.

// endedHolding stops an ending after its write of the membership, holding
// the workspace's row and the membership's, before the projects' step.
type endedHolding struct {
	*workspacepg.Store
	gate *gate
}

func (m endedHolding) EndMember(ctx context.Context, workspaceID, userID, by uuid.UUID, now time.Time) error {
	if err := m.Store.EndMember(ctx, workspaceID, userID, by, now); err != nil {
		return err
	}
	return m.gate.wait(ctx)
}

// leave is user's leaving of acme, over workspaces, with project's cascade,
// identity's profiles and the Authorizer as bootstrap wires them.
func (r adminRace) leave(ctx context.Context, user uuid.UUID, workspaces workspaceapp.WorkspaceLeaver) error {
	return workspaceapp.NewLeaveWorkspace(workspaces, workspaceProfiles{profiles: identity.Provide(r.pool).PublicProfiles},
		project.New(project.Deps{Pool: r.pool}).Cascade(), access.New(access.Deps{WorkspaceRoles: workspace.Provide(r.pool).WorkspaceRoles}),
		postgres.NewTxManager(r.pool, 2*time.Second), clock.System{}).
		Execute(shared.WithActor(ctx, shared.Actor{UserID: user}), "acme")
}

// active is whether alice's and bob's memberships of acme are active.
func (r adminRace) active(t *testing.T) (alice, bob bool) {
	t.Helper()
	if err := r.pool.QueryRow(context.Background(), "SELECT (SELECT is_active FROM workspace_members WHERE id = $1), "+
		"(SELECT is_active FROM workspace_members WHERE id = $2)", r.aliceIn, r.bobIn).Scan(&alice, &bob); err != nil {
		t.Fatal(err)
	}
	return alice, bob
}

// Interleaving 1, the workspace's side: acme's two admins, alice and bob,
// leave it at once (M3 design 3.6, 3.7 rule 1). The first holds acme FOR NO
// KEY UPDATE and, past his check of another admin, has ended his
// membership at his gate; the second waits for acme's row. Once the first
// has committed, the second finds no other active admin: 409
// workspace.sole_admin. The first one's membership is ended, the second's
// active: acme keeps an admin.
func TestTwoAdminsLeavingLeaveAnAdmin(t *testing.T) {
	for _, aliceFirst := range []bool{true, false} {
		t.Run(fmt.Sprintf("alice first %v", aliceFirst), func(t *testing.T) {
			r := newAdminRace(t)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			first, second := r.alice, r.bob
			if !aliceFirst {
				first, second = second, first
			}
			g := newGate()
			left := run(func() error { return r.leave(ctx, first, endedHolding{workspacepg.New(r.pool), g}) })
			held(t, ctx, g, left, "the first leaving")
			refused := run(func() error { return r.leave(ctx, second, workspacepg.New(r.pool)) })
			pgtest.WaitForLockWaitOn(t, r.pool, "workspaces", 5*time.Second)
			if alice, bob := r.active(t); !alice || !bob {
				t.Errorf("while the first holds the lock: alice active %v, bob active %v; want both", alice, bob)
			}
			close(g.open)

			if err := result(t, ctx, left, "the first leaving"); err != nil {
				t.Errorf("the first leaving = %v, want it done", err)
			}
			if err := result(t, ctx, refused, "the second leaving"); !errors.Is(err, workspacedomain.ErrSoleAdmin) {
				t.Errorf("the second leaving = %v, want 409 workspace.sole_admin", err)
			}
			if alice, bob := r.active(t); alice != !aliceFirst || bob != aliceFirst {
				t.Errorf("alice active %v, bob active %v; want the first one's membership ended, the second's active", alice, bob)
			}
		})
	}
}

// remove is alice's removal of bob from acme, over members, with project's
// cascade, identity's profiles and the Authorizer as bootstrap wires them,
// on the system's clock: its time is read when it reads it.
func (r growthRace) remove(ctx context.Context, members workspaceapp.MemberRemover) error {
	return workspaceapp.NewRemoveWorkspaceMember(members, workspaceProfiles{profiles: identity.Provide(r.pool).PublicProfiles},
		project.New(project.Deps{Pool: r.pool}).Cascade(), r.authorizer(), postgres.NewTxManager(r.pool, 2*time.Second), clock.System{}).
		Execute(shared.WithActor(ctx, shared.Actor{UserID: r.alice}), r.bobIn)
}

// Interleaving 4: alice's removal of bob from acme and the project side's
// growth, alice's adding him to Web or his joining it, serialize on acme's
// row (M3 design 3.6 conventions 2, 3 and 6), in both orders, for a new
// membership of Web and for his ended one. The growth first: it holds acme
// and his membership of acme FOR SHARE, no stronger (lockOn), and waits at
// its gate before it locks Web; the removal waits for acme's row, holding
// nothing. Once the growth has committed, the removal's step over his
// projects, a statement run after its write of his membership, finds his
// membership of Web and ends it, at the removal's time, read once it held
// acme: after the gate opened (3.3). The removal first: it holds acme FOR
// NO KEY UPDATE and his membership's row after its write; the growth waits
// for acme's row, then reads his membership ended: alice's adding him is
// refused as one who is no active member of acme (422
// members[0].member_id not_allowed), his joining as one who does not see
// Web (404); his membership of Web is as it was. In either order, while
// the second side waits for acme's row, no transaction holds Web (FOR
// UPDATE NOWAIT): the growth takes his membership of acme before Web, and
// the removal his membership before his projects.
func TestARemovalAndTheProjectSidesGrowthSerialize(t *testing.T) {
	contract := apitest.Load(t)
	for _, add := range []bool{true, false} {
		for _, ended := range []bool{false, true} {
			for _, growthFirst := range []bool{true, false} {
				name := fmt.Sprintf("%s, ended %v, growth first %v", map[bool]string{true: "add", false: "join"}[add], ended, growthFirst)
				t.Run(name, func(t *testing.T) {
					r := newGrowthRace(t, ended)
					before := map[bool]string{false: "15, Web none", true: "15, Web 15 ended"}[ended]
					if got := r.standing(t); got != before {
						t.Fatalf("before: %s, want %s", got, before)
					}
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancel()
					g := newGate()
					var req *http.Request
					var rec *httptest.ResponseRecorder
					var grew, removed <-chan error
					if growthFirst {
						grow := r.growth(t, add, g)
						grew = run(func() error { req, rec = grow(); return nil })
						held(t, ctx, g, grew, "the growth")
						if got, want := "acme "+lockOn(t, r.pool, "workspaces WHERE slug = 'acme'")+", his membership "+lockOn(t, r.pool,
							"workspace_members WHERE id = $1", r.bobIn), "acme FOR SHARE, his membership FOR SHARE"; got != want {
							t.Errorf("the growth at its gate holds %s; want %s", got, want)
						}
						removed = run(func() error { return r.remove(ctx, workspacepg.New(r.pool)) })
					} else {
						removed = run(func() error { return r.remove(ctx, endedHolding{workspacepg.New(r.pool), g}) })
						held(t, ctx, g, removed, "the removal")
						grow := r.growth(t, add, nil)
						grew = run(func() error { req, rec = grow(); return nil })
					}
					pgtest.WaitForLockWaitOn(t, r.pool, "workspaces", 5*time.Second)
					if !r.webFree(t) {
						t.Error("Web is held while the second side waits for acme's row; want it locked after that row")
					}
					opened := time.Now()
					close(g.open)

					removal := result(t, ctx, removed, "the removal")
					if err := result(t, ctx, grew, "the growth"); err != nil {
						t.Fatal(err)
					}
					contract.CheckResponse(t, req, rec.Result())
					want, grown := "15 ended, Web 15 ended", rec.Code == map[bool]int{false: http.StatusOK, true: http.StatusCreated}[add]
					if !growthFirst {
						want, grown = "15 ended, "+before[len("15, "):], refusedAsNoMember(rec, add)
					}
					if got := r.standing(t); removal != nil || !grown || got != want {
						t.Errorf("the removal = %v, the growth = %d %s, bob %s; want the removal done, the growth %s, bob %s", removal, rec.Code,
							rec.Body, got, map[bool]string{true: "done", false: "refused"}[growthFirst], want)
					}
					if made, written := r.membershipTimes(t); growthFirst && (written.Before(made) || written.Before(opened)) {
						t.Errorf("bob's membership of Web made at %v, last written at %v, the gate opened at %v; want it ended last by the "+
							"removal, at a time read once it held acme", made, written, opened)
					}
				})
			}
		}
	}
}

// refusedAsNoMember reports whether rec is the growth's refusal of bob as
// no active member of acme: alice's adding him, 422 members[0].member_id
// not_allowed alone; his joining, 404 project.not_found.
func refusedAsNoMember(rec *httptest.ResponseRecorder, add bool) bool {
	if !add {
		return refusedAsAGuest(rec, false)
	}
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
	return rec.Code == http.StatusUnprocessableEntity && problem.Code == "validation_failed" && len(problem.Errors) == 1 &&
		problem.Errors[0].Field == "members[0].member_id" && problem.Errors[0].Code == "not_allowed"
}
````

- [ ] **Step 2: 测试和 lint**

Run: `go -C server test -count=5 -race -run 'TestTwoAdminsLeavingLeaveAnAdmin$|TestARemovalAndTheProjectSidesGrowthSerialize$' ./internal/bootstrap/`
Expected: `ok`；输出里没有 `40P01`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 3: 提交**

```bash
git add server/internal/bootstrap/interleaving_endings_test.go server/internal/bootstrap/interleaving_growth_test.go
```
```bash
git commit -m "test(M3/P5a): interleavings 1 (the workspace's side) and 4

Two admins leaving at once serialize on the workspace's row, and the
second is refused with workspace.sole_admin: the workspace keeps an
admin. A removal and the project side's growth, an add or a join, of
a new membership or an ended one, serialize on the workspace's row in
both orders; the second side waits there, holding no project.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A 的探针反例；每个都让第二方不再等工作区行）：

| 改坏 | 必须失败的测试 | 层 |
|---|---|---|
| `LockWorkspaceBySlug` 锁成 `FOR SHARE`；不加锁 | `TestTwoAdminsLeavingLeaveAnAdmin`（两种顺序，探针在期限时失败） | 组合 |
| `LockWorkspace`（移出）不加锁 | `TestARemovalAndTheProjectSidesGrowthSerialize`（8 个子测试，探针失败） | 组合 |
| 增长的 `ShareDirectoryWorkspaceByID` 不加锁 | `TestARemovalAndTheProjectSidesGrowthSerialize`（8 个子测试：增长先时 `lockOn` 看不到 acme 的 `FOR SHARE`，移出先时探针失败） | 组合 |

**Done when:** 两个交错 `-count=5 -race` 通过，没有 40P01。

---

### Task 10: 交错 5、6

**Files:**
- Create: `server/internal/bootstrap/interleaving_removal_test.go`

**Interfaces:** 没有新的产品代码。`interleaving_removal_test.go`：`growthRace.opsOf`、`opsWritten`；`adminRace.deleteAcme`、`removeAlice`、`acmeAndAlice`。建项目经项目模块的接口（`newProjectRoute`，`gatedShares` 在它取完他的成员行之后停住），删除工作区经 `workspace` 的用例（P4b 的 `gatedDeleter`）。

**Tests:**
- `TestARemovalAndTheRemovedMembersProjectSerialize`（交错 5，两种顺序：建项目先：它持 acme 和他在 acme 的成员行的 `FOR SHARE`（`lockOn`），在插入 Ops 之前停在 gate；移出等 acme 的行；建项目提交 201 之后，移出的项目一步找到他在 Ops 的成员关系（Ops 的管理员、唯一的成员）并结束它，时刻是移出持有 acme 之后读的。移出先：它持 acme 和他已结束的成员行；建项目等 acme 的行，然后判定时他已不是成员：404 `workspace.not_found`，没有 Ops）。
- `TestARemovalAndTheRemovedAdminsDeletionSerialize`（交错 6，两种顺序：删除先：判定之后持 acme 的 `FOR NO KEY UPDATE`；已读过她成员行的移出等 acme 的行，然后因为 acme 已删除而锁不到行：404 `workspace.member_not_found`，她的成员关系随 acme 删除。移出先：它持 acme 和她已结束的成员行；删除等 acme 的行，判定时她已不是成员：404 `workspace.not_found`；acme 留着，她的成员关系已结束）。

- [ ] **Step 1: 交错测试**

`server/internal/bootstrap/interleaving_removal_test.go`（新文件，205 行）：

````file server/internal/bootstrap/interleaving_removal_test.go
package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/access"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
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

// The interleavings 5 and 6 of M3 design 9.3, on a real database, in both
// orders: a removal against the removed member's own write, his creating a
// project, through the project module behind the API, or his deleting the
// workspace, through workspace's use case; each with the Authorizer as
// bootstrap wires it, and gated as interleaving_endings_test.go gates.

// opsOf is bob's standing in Ops, the project his creation makes: "no
// Ops" when there is none, else his membership's role, "ended" when it
// is.
func (r growthRace) opsOf(t *testing.T) string {
	t.Helper()
	var s string
	if err := r.pool.QueryRow(context.Background(), `SELECT coalesce((SELECT 'Ops ' || m.role || CASE WHEN m.is_active THEN '' ELSE ' ended' END
		FROM projects p JOIN project_members m ON m.project_id = p.id WHERE p.name = 'Ops' AND m.member_id = $1), 'no Ops')`, r.bob).
		Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

// opsWritten is when bob's membership of Ops was last written.
func (r growthRace) opsWritten(t *testing.T) time.Time {
	t.Helper()
	var written time.Time
	if err := r.pool.QueryRow(context.Background(), `SELECT m.updated_at FROM projects p JOIN project_members m ON m.project_id = p.id
		WHERE p.name = 'Ops' AND m.member_id = $1`, r.bob).Scan(&written); err != nil {
		t.Fatal(err)
	}
	return written
}

// Interleaving 5: alice's removal of bob from acme and his creating the
// project Ops in it serialize on acme's row (M3 design 3.6 conventions 2, 3
// and 6). The creation first: it holds acme FOR SHARE and his membership of
// acme FOR SHARE (lockOn), and waits at its gate before it inserts Ops;
// the removal waits for acme's row. Once the creation has committed, 201,
// the removal's step over his projects finds his membership of Ops, Ops's
// admin and only member, and ends it at the removal's time, read once it
// held acme. The removal first: it holds acme and his ended membership's
// row; the creation waits for acme's row, then decides once he is no
// member: 404 workspace.not_found, and there is no Ops.
func TestARemovalAndTheRemovedMembersProjectSerialize(t *testing.T) {
	contract := apitest.Load(t)
	for _, creationFirst := range []bool{true, false} {
		t.Run(fmt.Sprintf("creation first %v", creationFirst), func(t *testing.T) {
			r := newGrowthRace(t, false)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			g := newGate()
			var req *http.Request
			var rec *httptest.ResponseRecorder
			create := func(gated *gate) func() error {
				route := newProjectRoute(t, r.pool, r.authorizer(), gatedShares{workspace.Provide(r.pool).WorkspaceMembers, gated})
				return func() error {
					req, rec = route.send(http.MethodPost, "/api/v0/workspaces/acme/projects", r.bob, `{"name":"Ops","identifier":"OPS"}`)
					return nil
				}
			}
			var created, removed <-chan error
			if creationFirst {
				created = run(create(g))
				held(t, ctx, g, created, "the creation")
				if got, want := "acme "+lockOn(t, r.pool, "workspaces WHERE slug = 'acme'")+", his membership "+lockOn(t, r.pool,
					"workspace_members WHERE id = $1", r.bobIn), "acme FOR SHARE, his membership FOR SHARE"; got != want {
					t.Errorf("the creation at its gate holds %s; want %s", got, want)
				}
				removed = run(func() error { return r.remove(ctx, workspacepg.New(r.pool)) })
			} else {
				removed = run(func() error { return r.remove(ctx, endedHolding{workspacepg.New(r.pool), g}) })
				held(t, ctx, g, removed, "the removal")
				created = run(create(nil))
			}
			pgtest.WaitForLockWaitOn(t, r.pool, "workspaces", 5*time.Second)
			opened := time.Now()
			close(g.open)

			removal := result(t, ctx, removed, "the removal")
			if err := result(t, ctx, created, "the creation"); err != nil {
				t.Fatal(err)
			}
			contract.CheckResponse(t, req, rec.Result())
			var problem struct {
				Code string `json:"code"`
			}
			_ = json.Unmarshal(rec.Body.Bytes(), &problem) // a 201 has no code
			want, answered := "15 ended, Web none, Ops 20 ended", rec.Code == http.StatusCreated
			if !creationFirst {
				want, answered = "15 ended, Web none, no Ops", rec.Code == http.StatusNotFound && problem.Code == "workspace.not_found"
			}
			if got := r.standing(t) + ", " + r.opsOf(t); removal != nil || !answered || got != want {
				t.Errorf("the removal = %v, the creation = %d %s, bob %s; want the removal done, the creation %s, bob %s", removal, rec.Code,
					rec.Body, got, map[bool]string{true: "201", false: "404 workspace.not_found"}[creationFirst], want)
			}
			if creationFirst {
				if written := r.opsWritten(t); written.Before(opened) {
					t.Errorf("bob's membership of Ops last written at %v, the gate opened at %v; want it ended by the removal, at a time read "+
						"once it held acme", written, opened)
				}
			}
		})
	}
}

// deleteAcme is alice's deletion of acme, over workspaces, with project's
// cascade and the Authorizer as bootstrap wires them.
func (r adminRace) deleteAcme(ctx context.Context, workspaces workspaceapp.WorkspaceDeleter) error {
	return workspaceapp.NewDeleteWorkspace(workspaces, project.New(project.Deps{Pool: r.pool}).Cascade(),
		access.New(access.Deps{WorkspaceRoles: workspace.Provide(r.pool).WorkspaceRoles}), postgres.NewTxManager(r.pool, 2*time.Second),
		clock.System{}, slog.New(slog.DiscardHandler)).Execute(shared.WithActor(ctx, shared.Actor{UserID: r.alice}), "acme")
}

// removeAlice is bob's removal of alice from acme, over members, with
// project's cascade, identity's profiles and the Authorizer as bootstrap
// wires them.
func (r adminRace) removeAlice(ctx context.Context, members workspaceapp.MemberRemover) error {
	return workspaceapp.NewRemoveWorkspaceMember(members, workspaceProfiles{profiles: identity.Provide(r.pool).PublicProfiles},
		project.New(project.Deps{Pool: r.pool}).Cascade(), access.New(access.Deps{WorkspaceRoles: workspace.Provide(r.pool).WorkspaceRoles}),
		postgres.NewTxManager(r.pool, 2*time.Second), clock.System{}).
		Execute(shared.WithActor(ctx, shared.Actor{UserID: r.bob}), r.aliceIn)
}

// acmeAndAlice is whether acme is deleted, then alice's membership of it:
// "active", "ended" or "deleted".
func (r adminRace) acmeAndAlice(t *testing.T) string {
	t.Helper()
	var s string
	if err := r.pool.QueryRow(context.Background(), `SELECT CASE WHEN w.deleted_at IS NULL THEN 'acme' ELSE 'acme deleted' END || ', alice ' ||
		CASE WHEN m.deleted_at IS NOT NULL THEN 'deleted' WHEN m.is_active THEN 'active' ELSE 'ended' END
		FROM workspaces w JOIN workspace_members m ON m.workspace_id = w.id WHERE m.id = $1`, r.aliceIn).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

// Interleaving 6: bob's removal of alice, acme's other admin, and her
// deleting acme serialize on acme's row (M3 design 3.6 convention 2). The
// deletion first: it holds acme FOR NO KEY UPDATE after its decision
// (gatedDeleter); the removal, which has read her membership, waits for
// acme's row, then locks no row, acme being deleted: 404
// workspace.member_not_found. Her membership is deleted with acme. The
// removal first: it holds acme and her ended membership's row; the
// deletion waits for acme's row, then decides once she is no member: 404
// workspace.not_found. acme stays, her membership ended.
func TestARemovalAndTheRemovedAdminsDeletionSerialize(t *testing.T) {
	for _, deletionFirst := range []bool{true, false} {
		t.Run(fmt.Sprintf("deletion first %v", deletionFirst), func(t *testing.T) {
			r := newAdminRace(t)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			g := newGate()
			var deleted, removed <-chan error
			if deletionFirst {
				deleted = run(func() error { return r.deleteAcme(ctx, gatedDeleter{workspacepg.New(r.pool), g}) })
				held(t, ctx, g, deleted, "the deletion")
				removed = run(func() error { return r.removeAlice(ctx, workspacepg.New(r.pool)) })
			} else {
				removed = run(func() error { return r.removeAlice(ctx, endedHolding{workspacepg.New(r.pool), g}) })
				held(t, ctx, g, removed, "the removal")
				deleted = run(func() error { return r.deleteAcme(ctx, workspacepg.New(r.pool)) })
			}
			pgtest.WaitForLockWaitOn(t, r.pool, "workspaces", 5*time.Second)
			if got := r.acmeAndAlice(t); got != "acme, alice active" {
				t.Errorf("while the first holds acme: %s, want acme, alice active", got)
			}
			close(g.open)

			deletion, removal := result(t, ctx, deleted, "the deletion"), result(t, ctx, removed, "the removal")
			wantDeletion, wantRemoval, want := error(nil), error(workspacedomain.ErrMemberNotFound), "acme deleted, alice deleted"
			if !deletionFirst {
				wantDeletion, wantRemoval, want = workspacedomain.ErrNotFound, nil, "acme, alice ended"
			}
			if !errors.Is(deletion, wantDeletion) || !errors.Is(removal, wantRemoval) || r.acmeAndAlice(t) != want {
				t.Errorf("the deletion = %v, the removal = %v, %s; want %v, %v, %s", deletion, removal, r.acmeAndAlice(t), wantDeletion,
					wantRemoval, want)
			}
		})
	}
}
````

- [ ] **Step 2: 测试和 lint**

Run: `go -C server test -count=5 -race -run 'TestARemovalAndTheRemovedMembersProjectSerialize$|TestARemovalAndTheRemovedAdminsDeletionSerialize$' ./internal/bootstrap/`
Expected: `ok`；输出里没有 `40P01`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 3: 提交**

```bash
git add server/internal/bootstrap/interleaving_removal_test.go
```
```bash
git commit -m "test(M3/P5a): interleavings 5 and 6

A removal and the removed member's creating a project serialize on the
workspace's row: first, the creation's project membership is ended by
the removal; second, the creation is refused 404. A removal of the
workspace's other admin and her deleting the workspace serialize there
too: the second is refused 404 either way.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A 的探针反例）：

| 改坏 | 必须失败的测试 | 层 |
|---|---|---|
| `LockWorkspace`（移出）不加锁 | `TestARemovalAndTheRemovedMembersProjectSerialize`、`TestARemovalAndTheRemovedAdminsDeletionSerialize`（两种顺序，探针失败） | 组合 |
| 建项目的 `ShareDirectoryWorkspace` 不加锁 | `TestARemovalAndTheRemovedMembersProjectSerialize`（建项目先时 `lockOn` 看不到 acme 的锁，移出先时探针失败） | 组合 |
| `LockWorkspaceBySlug`（删除工作区）不加锁 | `TestARemovalAndTheRemovedAdminsDeletionSerialize`（删除先时探针失败；移出先时删除改写 acme 的行同样等在 acme 上，由结果发现：acme 被删除，spec 2.11） | 组合 |

**Done when:** 两个交错 `-count=5 -race` 通过，没有 40P01。

---

### Task 11: 矩阵："被移出的成员"由存储写出

**Files:**
- Modify: `server/internal/bootstrap/demotion_test.go`、`server/internal/bootstrap/interleaving_growth_test.go`、`server/internal/bootstrap/interleaving_writes_test.go`、`server/internal/bootstrap/permission_matrix_columns_test.go`、`server/internal/bootstrap/permission_matrix_members_test.go`、`server/internal/bootstrap/permission_matrix_project_test.go`、`server/internal/bootstrap/permission_matrix_seed_test.go`、`server/internal/bootstrap/permission_matrix_seeded_test.go`、`server/internal/bootstrap/permission_matrix_test.go`、`server/internal/bootstrap/project_members_test.go`、`server/internal/bootstrap/project_writes_test.go`

**Interfaces:** 没有新的产品代码（spec 2.12，M3 设计 9.2；P4b spec 第 5 节 P5 一行）：
- `projectSeed.removal(sd)`：经两个存储照 `removeWorkspaceMember` 的语句结束被移出的成员：`EndMember`（acme 的管理员、一个时刻），然后 `LockActiveMemberProjects` 在这时找出的项目、`EndMemberships`。邀请的一步不写：`prepareMatrix` 之后才建发给他地址的邀请（移出之后发的，3.8）。`standIns` 只剩 P5b 的一条（以前的成员在私有项目的成员关系结束），不再收 `seeded`。
- `preconditions`：被移出的成员在他那一列的公开项目的成员关系已结束、未删除（`ProjectFacts`、`Memberships`），他的 404 是被移出的成员的。
- 成员列表一行（`listsTheProjectMembers`）不再有被移出的成员：他的项目成员关系随移出结束（P4b spec 第 3 节第 4 条）。添加"工作区成员关系已结束的目标"和建项目"成员关系已结束的负责人"两个变体照旧各只答一个错误（P4b review M4）。
- `TestAcceptingAsAGuestAgainDemotesInTheWorkspacesProjects` 的已结束的成员关系经接口写出：alice 移出 bob（以前是 SQL 替身），他的两行由 alice 写，接受邀请"由他自己写"因此可以不成立。其余提到"P5 的移出"的替身注释改指 P5b 的移出项目成员。

**Tests:** 没有新测试；矩阵（25 格之外不变的格子照旧）、前提、完整性核对、降级测试照旧通过。

- [ ] **Step 1: 矩阵的种子**

`server/internal/bootstrap/permission_matrix_seed_test.go`（修改，7 处）：

````old server/internal/bootstrap/permission_matrix_seed_test.go
// runs the SQL that stands in for the stores P4b and P5 add.
````
````new server/internal/bootstrap/permission_matrix_seed_test.go
// runs the SQL that stands in for the stores P5b adds.
````

````old server/internal/bootstrap/permission_matrix_seed_test.go
// for the store that will end a membership (P5), and makes the two
````
````new server/internal/bootstrap/permission_matrix_seed_test.go
// for the store that will end one project membership (P5b), and makes the two
````

````old server/internal/bootstrap/permission_matrix_seed_test.go
}

// standIns writes, through SQL, the states no store writes yet, until the
````
````new server/internal/bootstrap/permission_matrix_seed_test.go
}

// removal ends the removed member's membership of acme, then his active
// memberships of acme's projects, found when they are ended, through the
// stores, as removeWorkspaceMember's statements end them (M3 design 3.6's
// lock table, convention 6): by acme's admin, at one moment. Its other
// statement, the pending invitations', is left out: prepareMatrix makes
// the invitation to his address after it, as one sent once he was removed
// (3.8).
func (s projectSeed) removal(sd seeded) {
	s.t.Helper()
	ctx, acme, removed, by := context.Background(), sd.workspace("acme"), s.ids[callerRemoved], s.ids[matrixAdmins["acme"]]
	if err := s.matrixSeed.store.EndMember(ctx, acme, removed, by, s.now); err != nil {
		s.t.Fatal(err)
	}
	projects, err := s.store.LockActiveMemberProjects(ctx, []uuid.UUID{acme}, removed)
	if err != nil {
		s.t.Fatal(err)
	}
	if err := s.store.EndMemberships(ctx, projects, removed, by, s.now); err != nil {
		s.t.Fatal(err)
	}
}

// standIns writes, through SQL, the states no store writes yet, until the
````

````old server/internal/bootstrap/permission_matrix_seed_test.go
// the private project ended (P5) and the removed member's membership of
// acme ended (P5). exec fails a statement that changes no row, and names
// it, which it checks first.
func (s projectSeed) standIns(pool *pgxpool.Pool, sd seeded) {
````
````new server/internal/bootstrap/permission_matrix_seed_test.go
// the private project ended (P5b, whose removal of a project member ends
// one). exec fails a statement that changes no row, and names it, which it
// checks first.
func (s projectSeed) standIns(pool *pgxpool.Pool) {
````

````old server/internal/bootstrap/permission_matrix_seed_test.go
		s.projects["acme/private"], s.ids[callerBefore])
	s.exec(s.t, pool, "UPDATE workspace_members SET is_active = false WHERE id = $1", sd.membership("acme", callerRemoved))
````
````new server/internal/bootstrap/permission_matrix_seed_test.go
		s.projects["acme/private"], s.ids[callerBefore])
````

````old server/internal/bootstrap/permission_matrix_seed_test.go
	// The removed member is still an active member of the project his
	// column aims at, so that only his ended membership of acme keeps him
	// out of it: his cell's 404 would not show which, were he none.
	if f, found, err := s.store.ProjectFacts(ctx, s.projects[projectOf(callerRemoved)], s.ids[callerRemoved]); err != nil || !found || !f.Member {
		s.t.Fatalf("the removed member's facts of %s = %+v, %v, %v; want him its active member", projectOf(callerRemoved), f, found, err)
````
````new server/internal/bootstrap/permission_matrix_seed_test.go
	// The removed member's membership of the project his column aims at,
	// public, is ended, as the removal left it, not deleted: his cell's 404
	// is a removed member's, which a member of acme would not get there.
	public, removed := s.projects[projectOf(callerRemoved)], s.ids[callerRemoved]
	if f, found, err := s.store.ProjectFacts(ctx, public, removed); err != nil || !found || f.Member {
		s.t.Fatalf("the removed member's facts of %s = %+v, %v, %v; want him no active member of it", projectOf(callerRemoved), f, found, err)
	}
	if ms, err := s.store.Memberships(ctx, public, []uuid.UUID{removed}); err != nil || len(ms) != 1 || ms[removed].Active {
		s.t.Fatalf("the removed member's memberships of %s = %+v, %v; want his ended one", projectOf(callerRemoved), ms, err)
````

````old server/internal/bootstrap/permission_matrix_seed_test.go
// of acme (the removed member's membership ended: standIns), WG-'s its
````
````new server/internal/bootstrap/permission_matrix_seed_test.go
// of acme (the removed member's membership ended: removal), WG-'s its
````

`server/internal/bootstrap/permission_matrix_test.go`（修改，3 处）：

````old server/internal/bootstrap/permission_matrix_test.go
// workspace store, the workspaces, memberships and invitations of
// matrixMemberships and matrixInvitations, with the ids newSeeded named,
// and acme's admin's display settings; other's admin and removed member
// are there so that a role read in the wrong workspace lets either into
// acme. Through the project store, the projects and project memberships of
// matrixProjects and matrixProjectMembers, and acme's archived project
// archived. Through SQL, the states no store writes yet (standIns,
// partingStates). Through the API, gone deleted by its admin, which
// soft-deletes its memberships and its project with it; then the checks
// that the rows the cells rest on are there (preconditions). Everything
// that connected to the database is closed when it returns, so that it can
// be copied. A -run that leaves out prepare fails here, not with a 401 in
// every cell.
````
````new server/internal/bootstrap/permission_matrix_test.go
// workspace store, the workspaces and memberships of matrixMemberships,
// with the ids newSeeded named, and acme's admin's display settings;
// other's admin and removed member are there so that a role read in the
// wrong workspace lets either into acme. Through the project store, the
// projects and project memberships of matrixProjects and
// matrixProjectMembers, and acme's archived project archived. Through both
// stores, the removed member's removal; then, through the workspace store,
// the invitations of matrixInvitations. Through SQL, the states no store
// writes yet (standIns, partingStates). Through the API, gone deleted by
// its admin, which soft-deletes its memberships and its project with it;
// then the checks that the rows the cells rest on are there
// (preconditions). Everything that connected to the database is closed
// when it returns, so that it can be copied. A -run that leaves out
// prepare fails here, not with a 401 in every cell.
````

````old server/internal/bootstrap/permission_matrix_test.go
		}
		for _, i := range matrixInvitations {
			seed.invite(s.invitation(i.slug, i.email), i.slug, i.email, i.role)
		}
````
````new server/internal/bootstrap/permission_matrix_test.go
		}
````

````old server/internal/bootstrap/permission_matrix_test.go
		projects.standIns(pool, s)
````
````new server/internal/bootstrap/permission_matrix_test.go
		projects.removal(s)
		for _, i := range matrixInvitations {
			seed.invite(s.invitation(i.slug, i.email), i.slug, i.email, i.role)
		}
		projects.standIns(pool)
````

`server/internal/bootstrap/permission_matrix_seeded_test.go`（修改，1 处）：

````old server/internal/bootstrap/permission_matrix_seeded_test.go
// the project level's members (projectColumns); the removed member, still
// an active member of the public one, so that only his membership of acme
// keeps him out; the member before in the private one, ended; WG- in both,
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
// the project level's members (projectColumns); the removed member in the
// public one, whose membership of it his removal ends; the member before
// in the private one, ended; WG- in both,
````

`server/internal/bootstrap/permission_matrix_members_test.go`（修改，3 处）：

````old server/internal/bootstrap/permission_matrix_members_test.go
		// member's membership of acme ended, his membership of the public
		// project still active (the stand-in); WG-'s is its guest, asked for
````
````new server/internal/bootstrap/permission_matrix_members_test.go
		// member's memberships of acme and of the public project ended (the
		// removal); WG-'s is its guest, asked for
````

````old server/internal/bootstrap/permission_matrix_members_test.go
// workspace's guest (PG's account), PM+WA and the removed member, whose
// membership of the project the stand-in left active (the list reads
// project_members alone, spec P4b §3 item 4): he stays listed until P5's
// removal ends his memberships of acme's projects with his membership of
// acme, and P5 drops him from here. Not WG-, whose membership
// partingStates ended.
````
````new server/internal/bootstrap/permission_matrix_members_test.go
// workspace's guest (PG's account) and PM+WA. Not the removed member,
// whose membership of the project his removal ended with his membership
// of acme (the list reads project_members alone, spec P4b §3 item 4); not
// WG-, whose membership partingStates ended.
````

````old server/internal/bootstrap/permission_matrix_members_test.go
	}{{callerProjectAdmin, 20}, {callerProjectMember, 15}, {callerGuest, 5}, {callerMemberAndAdmin, 15}, {callerRemoved, 15}}
````
````new server/internal/bootstrap/permission_matrix_members_test.go
	}{{callerProjectAdmin, 20}, {callerProjectMember, 15}, {callerGuest, 5}, {callerMemberAndAdmin, 15}}
````

`server/internal/bootstrap/permission_matrix_project_test.go`（修改，1 处）：

````old server/internal/bootstrap/permission_matrix_project_test.go
		// The removed member, whose membership of acme the stand-in ended, is
````
````new server/internal/bootstrap/permission_matrix_project_test.go
		// The removed member, whose membership of acme the removal ended, is
````

`server/internal/bootstrap/permission_matrix_columns_test.go`（修改，1 处）：

````old server/internal/bootstrap/permission_matrix_columns_test.go
		// admin; the public project, which only his ended membership of acme
		// keeps the removed member out of.
````
````new server/internal/bootstrap/permission_matrix_columns_test.go
		// admin; the public project, which any member of acme sees, and the
		// removed member, his memberships of acme and of it ended, does not.
````

- [ ] **Step 2: 降级测试和替身的注释**

`server/internal/bootstrap/demotion_test.go`（修改，2 处）：

````old server/internal/bootstrap/demotion_test.go
// project Web, so was its admin, and has left acme and Web, both his
// memberships ended as P5's leave ends them. alice invites him again, as a
// guest. While the projects' step fails, his acceptance answers 500 and
// changes nothing, the invitation still pending. So does an acceptance
````
````new server/internal/bootstrap/demotion_test.go
// project Web, so was its admin, beside alice, its creator, who has
// removed him from acme, which ended his memberships of acme and of Web,
// at one moment, by her. She invites him again, as a guest. While the
// projects' step fails, his acceptance answers 500 and changes nothing,
// the invitation still pending. So does an acceptance
````

````old server/internal/bootstrap/demotion_test.go
	for _, table := range []string{"workspace_members", "project_members"} {
		if _, err := pool.Exec(context.Background(), "UPDATE "+table+" SET is_active = false WHERE member_id = $1", bobID); err != nil {
			t.Fatal(err)
		}
````
````new server/internal/bootstrap/demotion_test.go
	var membership uuid.UUID
	if err := pool.QueryRow(context.Background(), "SELECT id FROM workspace_members WHERE member_id = $1", bobID).Scan(&membership); err != nil {
		t.Fatal(err)
	}
	if status, body := call(t, contract, http.MethodDelete, base+"/api/v0/workspace-members/"+membership.String(), alice, ""); status !=
		http.StatusNoContent {
		t.Fatalf("alice's removal of bob = %d %s", status, body)
````

`server/internal/bootstrap/interleaving_growth_test.go`（修改，1 处）：

````old server/internal/bootstrap/interleaving_growth_test.go
// membership as a member when ended is set: P5's removal ends one, and SQL
// stands in for it.
````
````new server/internal/bootstrap/interleaving_growth_test.go
// membership as a member when ended is set: P5b's removal of a project
// member ends one, and SQL stands in for it.
````

`server/internal/bootstrap/interleaving_writes_test.go`（修改，1 处）：

````old server/internal/bootstrap/interleaving_writes_test.go
// Web and is its admin (SQL stands in for P5's role change), and Ops is
````
````new server/internal/bootstrap/interleaving_writes_test.go
// Web and is its admin (SQL stands in for P5b's role change), and Ops is
````

`server/internal/bootstrap/project_members_test.go`（修改，2 处）：

````old server/internal/bootstrap/project_members_test.go
// membership is ended with a role, as P5's removal will end it (SQL stands
// in), and he joins again, or alice adds him with a role: the same row is
// active again with the role of 9.1's row, by the caller at the time of
````
````new server/internal/bootstrap/project_members_test.go
// membership is ended with a role, as P5b's removal of a project member
// will end it (SQL stands in), and he joins again, or alice adds him with
// a role: the same row is active again with the role of 9.1's row, by the
// caller at the time of
````

````old server/internal/bootstrap/project_members_test.go
// there (P5's removal ends it; SQL stands in): each time, with the commits
// of memberships refused, then those of display settings, which the growth
````
````new server/internal/bootstrap/project_members_test.go
// there (P5b's removal of a project member ends it; SQL stands in): each
// time, with the commits of memberships refused, then those of display
// settings, which the growth
````

`server/internal/bootstrap/project_writes_test.go`（修改，2 处）：

````old server/internal/bootstrap/project_writes_test.go
		// before (P5's removal ends one; SQL stands in), restored, and her
		// display settings made: two rows.
````
````new server/internal/bootstrap/project_writes_test.go
		// before (P5b's removal of a project member ends one; SQL stands
		// in), restored, and her display settings made: two rows.
````

````old server/internal/bootstrap/project_writes_test.go
// whom she adds as a guest, bob, whose membership ended (P5's removal ends
// one; SQL stands in), and dave, acme's member and none of Web's, are each
// refused as lead and as default assignee: 422 naming the field
````
````new server/internal/bootstrap/project_writes_test.go
// whom she adds as a guest, bob, whose membership ended (P5b's removal of a
// project member ends one; SQL stands in), and dave, acme's member and none
// of Web's, are each refused as lead and as default assignee: 422 naming
// the field
````

- [ ] **Step 3: 测试和 lint**

Run: `go -C server test -count=1 -run 'TestPermissionMatrix$|TestThePermissionMatrixCoversEveryOperation|TestMatrixViolationsCatchesEach|TestEveryColumnCallsAsARegisteredAccount|TestAcceptingAsAGuestAgainDemotesInTheWorkspacesProjects' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 4: 提交**

```bash
git add server/internal/bootstrap/demotion_test.go server/internal/bootstrap/interleaving_growth_test.go server/internal/bootstrap/interleaving_writes_test.go server/internal/bootstrap/permission_matrix_columns_test.go server/internal/bootstrap/permission_matrix_members_test.go server/internal/bootstrap/permission_matrix_project_test.go server/internal/bootstrap/permission_matrix_seed_test.go server/internal/bootstrap/permission_matrix_seeded_test.go server/internal/bootstrap/permission_matrix_test.go server/internal/bootstrap/project_members_test.go server/internal/bootstrap/project_writes_test.go
```
```bash
git commit -m "test(M3/P5a): the matrix's removed member is removed through the stores

prepareMatrix ends the removed member's membership of acme and his
project memberships as removeWorkspaceMember's statements do, instead
of SQL, and checks that his membership of the public project ended: he
is no longer listed among its members. The demotion test's ended
memberships come from alice's removal of bob through the API.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A，清扫 10）：

| 改坏 | 必须失败的测试 | 层 |
|---|---|---|
| `removal` 不结束他在 acme 的成员关系 | `TestPermissionMatrix/prepare`（`targets` 的前提：他在 acme 不是有效成员） | 组合 |
| `removal` 不结束他的项目成员关系 | `TestPermissionMatrix/prepare`（`preconditions`：他在公开项目不是有效成员） | 组合 |
| 成员列表把被移出的成员也列出（`ListMembers` 去掉 `is_active`） | `TestPermissionMatrix`；W7（Task 13 起） | 组合；端到端 |
| 降为访客不写降级者（P4a 的 `DemoteMemberships`） | `TestAcceptingAsAGuestAgainDemotesInTheWorkspacesProjects`（接受之前那两行由 alice 写）；W7（Task 13 起） | 组合；端到端 |

**Done when:** 矩阵里只剩 P5b 的一条替身；被移出的成员不在项目成员列表里。

---

### Task 12: S2 和 9d 顺序的集成测试

**Files:**
- Create: `server/internal/bootstrap/invitation_rules_test.go`

**Interfaces:** 没有新的产品代码（spec 2.13，M3 设计 3.8、9.3）。`adminsWorld`：acme 的管理员 alice（创建者）和 bob（接受她以管理员发的邀请），成员 carol，alice 的 Web；然后 alice 移出 bob、再以访客邀请他（`invitation`），`reactivate-member` 恢复他：他是有效的管理员，访客的邀请仍待接受。这是 9d 需要的状态（一个有效成员带着发给他的待接受邀请）唯一经接口和命令得到的方式（3.8 拒绝邀请有效成员）。

**Tests:**
- `TestAnInvitationNeverChangesAnActiveMembership`（S2 的顺序：bob 恢复之后加入 Web，以 acme 的管理员成为它的管理员；alice 离开 acme，bob 是唯一的管理员；bob 接受他被移出时 alice 发的访客邀请：200，回答 acme 和他的角色 20；邀请已接受并删除；他在 acme 仍是有效的管理员、在 Web 仍是管理员：acme 没有失去管理员，没有访客管理项目）。
- `TestAnEndedMembershipLeavesNoInvitation`（9d 的顺序，移出和离开各一次：bob 有效、访客的邀请待接受；他管理自己的 Ops，carol 是它的成员：结束 409 `project.sole_admin`，邀请仍待接受，链接答 200；alice 加入 Ops 之后结束 204；之后链接对查看和 bob 的接受都答 404 `workspace.invitation_not_found`，他没有回来，邀请删除的时刻就是他成员关系的 `updated_at`）。

- [ ] **Step 1: 集成测试**

`server/internal/bootstrap/invitation_rules_test.go`（新文件，208 行）：

````file server/internal/bootstrap/invitation_rules_test.go
package bootstrap

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// The two rules of M3 design 3.8 that keep an invitation from changing a
// membership, on the wired app, in the orders 9.3 gives: Codex's S2 and the
// review's spike 9d. Each starts as S2 does: acme's two admins, alice and
// bob; alice removes bob, invites him again as a guest, and
// `nerve workspaces reactivate-member` restores his membership, an
// admin's, while the invitation stays pending.

// adminsWorld is the wired app on a database of its own: acme, whose
// admins are alice, its creator, and bob, by accepting her invitation as
// an admin, and whose member is carol, by accepting hers; Web, alice's.
// Then alice has removed bob, invited him again as a guest (invitation),
// and reactivate-member has restored him.
type adminsWorld struct {
	url, base  string
	contract   *apitest.Contract
	pool       *pgxpool.Pool
	tokens     map[string]string    // access tokens, by name
	ids        map[string]uuid.UUID // accounts, by name
	web        uuid.UUID
	invitation invitationLink // the guest's invitation to bob's address
}

func newAdminsWorld(t *testing.T) adminsWorld {
	t.Helper()
	w := adminsWorld{url: pgtest.NewDatabase(t), contract: apitest.Load(t), tokens: map[string]string{}, ids: map[string]uuid.UUID{}}
	w.base, w.pool = startApp(t, testConfig(t, w.url, false), migrations.FS()), openPool(t, w.url)
	for _, name := range []string{"alice", "bob", "carol"} {
		w.tokens[name] = registerAccount(t, w.contract, w.base, name+"@example.com").AccessToken
		w.ids[name] = accountID(t, w.contract, w.base, w.tokens[name])
	}
	if status, body := call(t, w.contract, http.MethodPost, w.base+"/api/v0/workspaces", w.tokens["alice"],
		`{"name":"Acme","slug":"acme"}`); status != http.StatusCreated {
		t.Fatalf("creating acme = %d %s", status, body)
	}
	for name, role := range map[string]shared.Role{"bob": shared.RoleAdmin, "carol": shared.RoleMember} {
		answerInvitation(t, w.contract, w.base, w.tokens[name], "accept",
			inviteAs(t, w.contract, w.base, w.tokens["alice"], "acme", name+"@example.com", role), http.StatusOK)
	}
	w.web = createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Web", "WEB")
	if status, body := w.remove(t); status != http.StatusNoContent {
		t.Fatalf("alice's removal of bob = %d %s", status, body)
	}
	w.invitation = inviteAs(t, w.contract, w.base, w.tokens["alice"], "acme", "bob@example.com", shared.RoleGuest)
	if out, _, err := runWorkspaces(t, w.url, ReactivateMember("acme", "bob@example.com")); err != nil ||
		!strings.HasPrefix(out, "reactivated bob@example.com in acme as admin;") {
		t.Fatalf("reactivate-member of bob = %q, %v", out, err)
	}
	return w
}

// remove is alice's removal of bob from acme: its status and body.
func (w adminsWorld) remove(t *testing.T) (int, string) {
	t.Helper()
	var id uuid.UUID
	if err := w.pool.QueryRow(context.Background(), `SELECT m.id FROM workspace_members m JOIN workspaces s ON s.id = m.workspace_id
		WHERE s.slug = 'acme' AND m.member_id = $1`, w.ids["bob"]).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return call(t, w.contract, http.MethodDelete, w.base+"/api/v0/workspace-members/"+id.String(), w.tokens["alice"], "")
}

// joins has name join the project, wanting 200.
func (w adminsWorld) joins(t *testing.T, name string, project uuid.UUID) {
	t.Helper()
	if status, body := call(t, w.contract, http.MethodPost, w.base+"/api/v0/projects/"+project.String()+"/join", w.tokens[name],
		""); status != http.StatusOK {
		t.Fatalf("%s's joining %s = %d %s", name, project, status, body)
	}
}

// bobs is bob's membership of acme and of each project he has one of, each
// as its name, its role and whether it is active, in byte order.
func (w adminsWorld) bobs(t *testing.T) string {
	t.Helper()
	var s string
	if err := w.pool.QueryRow(context.Background(), `SELECT string_agg(r, '; ' ORDER BY r COLLATE "C") FROM (
		SELECT 'acme ' || role || ' ' || is_active AS r FROM workspace_members WHERE member_id = $1
		UNION ALL SELECT p.name || ' ' || m.role || ' ' || m.is_active FROM project_members m JOIN projects p ON p.id = m.project_id
		WHERE m.member_id = $1) s`, w.ids["bob"]).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

// look is the status and body of a look at the invitation through its
// link, as anyone holding it.
func (w adminsWorld) look(t *testing.T) (int, string) {
	t.Helper()
	return call(t, w.contract, http.MethodGet, w.base+"/api/v0/workspace-invitations/"+w.invitation.id.String()+"?token="+w.invitation.token, "",
		"")
}

// An invitation never changes an active membership (M3 design 3.8, 9.3),
// in Codex's S2 order: once reactivate-member has restored bob, an admin,
// he joins Web, its admin as acme's; alice leaves acme, which has bob as
// its only admin then; bob accepts the guest's invitation alice sent while
// he was removed: 200, its answer acme with his role, 20. The invitation is
// accepted and deleted; his membership of acme is an admin's, active, and
// his membership of Web an admin's: acme is not left without an admin, and
// no guest administers a project.
func TestAnInvitationNeverChangesAnActiveMembership(t *testing.T) {
	w := newAdminsWorld(t)
	w.joins(t, "bob", w.web)
	if status, body := call(t, w.contract, http.MethodPost, w.base+"/api/v0/workspaces/acme/leave", w.tokens["alice"], ""); status !=
		http.StatusNoContent {
		t.Fatalf("alice's leaving acme = %d %s", status, body)
	}
	if got, want := w.bobs(t), "Web 20 true; acme 20 true"; got != want {
		t.Fatalf("bob's memberships before his acceptance: %s, want %s", got, want)
	}

	status, body := call(t, w.contract, http.MethodPost, w.base+"/api/v0/workspace-invitations/"+w.invitation.id.String()+"/accept",
		w.tokens["bob"], `{"token":"`+w.invitation.token+`"}`)

	var answer struct {
		Slug string      `json:"slug"`
		Role shared.Role `json:"role"`
	}
	if status != http.StatusOK {
		t.Fatalf("bob's acceptance = %d %s, want 200", status, body)
	}
	decodeAnswer(t, body, &answer)
	if answer.Slug != "acme" || answer.Role != shared.RoleAdmin {
		t.Errorf("bob's acceptance answers %s; want acme with his role, 20", body)
	}
	var accepted, deleted bool
	if err := w.pool.QueryRow(context.Background(), "SELECT accepted, deleted_at IS NOT NULL FROM workspace_member_invites WHERE id = $1",
		w.invitation.id).Scan(&accepted, &deleted); err != nil || !accepted || !deleted {
		t.Errorf("the invitation accepted %v, deleted %v (%v); want both", accepted, deleted, err)
	}
	if got, want := w.bobs(t), "Web 20 true; acme 20 true"; got != want {
		t.Errorf("bob's memberships after his acceptance: %s, want %s, as they were", got, want)
	}
}

// An ended membership leaves no invitation (M3 design 3.8, 9.3), in the
// review's spike 9d order, for alice's removal of bob and for his own
// leaving: once reactivate-member has restored him, bob is active with the
// guest's invitation pending. He administers Ops, his own, and carol is
// its member: the ending is 409 project.sole_admin, and the invitation
// stays pending, its link answering 200. Once alice has joined Ops, the
// ending is 204; then the link answers 404 workspace.invitation_not_found
// to a look and to bob's acceptance, he has not come back, and the
// invitation was deleted at the ending's moment, his membership's
// updated_at.
func TestAnEndedMembershipLeavesNoInvitation(t *testing.T) {
	for _, e := range []struct {
		name string
		end  func(w adminsWorld, t *testing.T) (int, string)
	}{
		{"removal", adminsWorld.remove},
		{"leaving", func(w adminsWorld, t *testing.T) (int, string) {
			return call(t, w.contract, http.MethodPost, w.base+"/api/v0/workspaces/acme/leave", w.tokens["bob"], "")
		}},
	} {
		t.Run(e.name, func(t *testing.T) {
			w := newAdminsWorld(t)
			ops := createdProject(t, w.contract, w.base, w.tokens["bob"], "acme", "Ops", "OPS")
			w.joins(t, "carol", ops)
			if status, body := e.end(w, t); status != http.StatusConflict || problemCode(t, []byte(body)) != "project.sole_admin" {
				t.Fatalf("the ending, bob Ops's only admin = %d %s, want 409 project.sole_admin", status, body)
			}
			if status, body := w.look(t); status != http.StatusOK {
				t.Errorf("a look at the invitation after the refused ending = %d %s, want 200: still pending", status, body)
			}
			w.joins(t, "alice", ops)
			if status, body := e.end(w, t); status != http.StatusNoContent {
				t.Fatalf("the ending = %d %s, want 204", status, body)
			}

			if status, body := w.look(t); status != http.StatusNotFound || problemCode(t, []byte(body)) != "workspace.invitation_not_found" {
				t.Errorf("a look at the invitation = %d %s, want 404 workspace.invitation_not_found", status, body)
			}
			status, body := call(t, w.contract, http.MethodPost, w.base+"/api/v0/workspace-invitations/"+w.invitation.id.String()+"/accept",
				w.tokens["bob"], `{"token":"`+w.invitation.token+`"}`)
			if status != http.StatusNotFound || problemCode(t, []byte(body)) != "workspace.invitation_not_found" {
				t.Errorf("bob's acceptance = %d %s, want 404 workspace.invitation_not_found", status, body)
			}
			if got, want := w.bobs(t), "Ops 20 false; acme 20 false"; got != want {
				t.Errorf("bob's memberships: %s, want %s: ended, and not back", got, want)
			}
			var deletedAt, endedAt time.Time
			if err := w.pool.QueryRow(context.Background(), `SELECT i.deleted_at, m.updated_at FROM workspace_member_invites i
				JOIN workspace_members m ON m.workspace_id = i.workspace_id WHERE i.id = $1 AND m.member_id = $2`, w.invitation.id,
				w.ids["bob"]).Scan(&deletedAt, &endedAt); err != nil || !deletedAt.Equal(endedAt) {
				t.Errorf("the invitation deleted at %v, bob's membership ended at %v (%v); want the one moment", deletedAt, endedAt, err)
			}
		})
	}
}
````

- [ ] **Step 2: 测试和 lint**

Run: `go -C server test -count=1 -run 'TestAnInvitationNeverChangesAnActiveMembership$|TestAnEndedMembershipLeavesNoInvitation$' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 3: 提交**

```bash
git add server/internal/bootstrap/invitation_rules_test.go
```
```bash
git commit -m "test(M3/P5a): an invitation never changes an active membership; an ended one leaves none

In Codex's S2 order, an old guest invitation accepted by a reactivated
admin, the workspace's only one, is consumed and changes nothing. In
the review's spike 9d order, a removal and a leaving each delete the
pending invitation to the member's address at their moment, its link
answering 404 afterwards, and leave it pending when refused.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A）：

| 改坏 | 必须失败的测试 | 层 |
|---|---|---|
| 结束一步跳过邀请 | `TestAnEndedMembershipLeavesNoInvitation` | 组合 |
| 接受邀请时改有效成员的角色（P3 的规则） | `TestAnInvitationNeverChangesAnActiveMembership` | 组合 |

**Done when:** 两个集成测试通过，9d 对移出和离开各一次。

---

### Task 13: 端到端：W2、W7 的接口版本

**Files:**
- Create: `e2e/stories/workspace/w2-landing.spec.ts`、`e2e/stories/workspace/w7-member-management.spec.ts`
- Modify: `e2e/fixtures/api.ts`、`e2e/fixtures/assert/workspace.ts`、`e2e/stories/workspace/w3-workspace-settings.spec.ts`

**Interfaces:**（spec 2.14，M3 设计第 2 节 W2、W7，9.6）
- `e2e/fixtures/api.ts`：`listMembers(api, token, slug)`、`membershipOf(api, token, slug, memberId)`；类型 `WorkspaceMember`。
- `e2e/fixtures/assert/workspace.ts`：`expectMembershipEnded(db, slug, email, byEmail, role, projects)`（他在工作区的成员关系结束、行留着、角色不变，由 `byEmail`；他在 `projects` 里的项目成员关系与它同一个时刻、同一个写者结束，各留角色；他在这个工作区没有别的有效项目成员关系；没有发给他地址的待接受邀请）；`expectWrittenLastBy(db, slug, email, writers)`（结束之前，他的工作区成员关系和每个有效项目成员关系各由谁最后写：故事说"由 X 结束"之前，那些行由别人写过）；`expectWorkspaceDeleted` 的 `deletedAlone` 成为参数（`deletedAloneTables`，W3 照旧传它；W2 删除 First 时传空的：它的行都没有单独删除过）。

**Tests:**
- W2（API）：一个账户的工作区是它有效所在的那些，带它的角色、成员数和建立的时刻；删除一个之后少一个；把一个成员设为管理员之后离开另一个，就一个都没有了。First 有它删除时写的每张表的一行；Second 有 bob 作成员，他的 Ops 里 alice 由他添加为管理员；Second 唯一的管理员不能离开（409 `workspace.sole_admin`）；bob 成为管理员、又把她的角色原样设一次之后，她的成员关系由 bob 最后写，她离开：她的两行由她自己结束。
- W7（API）：管理员把一个成员降为访客、他在每个项目都成为访客；移出另一个，她的项目成员关系结束，不动别的邀请；再以访客邀请她回来；谁都不能改、移出自己的成员关系，唯一的管理员不能离开（Other 另有管理员 bob）。carol 是 Docs 唯一的管理员（erin 是它另一位管理员，已离开 acme），dave 是它的成员：移出 carol 409 `project.sole_admin`，管理员加入 Docs 之后 204；被删除的发给她的邀请、Other 发给她的、发给 eve 的一列不变；bob、carol 自己加入 Web，所以降级和移出前每一行都由他们自己写。
- W3：`expectWorkspaceDeleted` 传 `deletedAloneTables`。

- [ ] **Step 1: 夹具**

`e2e/fixtures/api.ts`（修改，2 处）：

````old e2e/fixtures/api.ts
export type WorkspaceInvitation = components["schemas"]["WorkspaceInvitation"];
````
````new e2e/fixtures/api.ts
export type WorkspaceInvitation = components["schemas"]["WorkspaceInvitation"];
export type WorkspaceMember = components["schemas"]["WorkspaceMember"];
````

````old e2e/fixtures/api.ts
}

/** Creates a project in the workspace of slug with the bearer token given, an admin's or a member's, and returns it. */
````
````new e2e/fixtures/api.ts
}

/** Lists the memberships of the workspace of slug, ended ones too, with the bearer token given, a member's. */
export async function listMembers(api: Api, token: string, slug: string): Promise<WorkspaceMember[]> {
  const { data, error, response } = await api.GET("/api/v0/workspaces/{slug}/members", {
    params: { path: { slug } },
    headers: bearer(token),
  });
  expect(response.status, `list the members of ${slug}: ${JSON.stringify(error)}`).toBe(200);
  if (!data) {
    throw new Error(`listMembers of ${slug} answered 200 without the members`);
  }
  return data.data;
}

/** The id of the membership of the account of memberId in the workspace of slug, as the caller of token lists it. */
export async function membershipOf(api: Api, token: string, slug: string, memberId: string): Promise<string> {
  const membership = (await listMembers(api, token, slug)).find((m) => m.member.id === memberId);
  if (!membership) {
    throw new Error(`no membership of ${memberId} in ${slug}`);
  }
  return membership.id;
}

/** Creates a project in the workspace of slug with the bearer token given, an admin's or a member's, and returns it. */
````

`e2e/fixtures/assert/workspace.ts`（修改，9 处）：

````old e2e/fixtures/assert/workspace.ts
  expect(rows, `the membership of ${email} in ${slug}`).toEqual(want === null ? [] : [want]);
}

/**
 * The tables whose rows belong to a workspace and are deleted with it (M3 design 3.6, 4.12); P7 adds the labels.
 * deletedAlone tells whether W3 deletes rows of the table on their own before the workspace, which keep that
 * moment: an invitation, when it is accepted or deleted (M3 design 3.8); a project, its memberships, its members'
 * display settings and its states, when the project is deleted (P4b). No row of the other tables is deleted alone.
````
````new e2e/fixtures/assert/workspace.ts
  expect(rows, `the membership of ${email} in ${slug}`).toEqual(want === null ? [] : [want]);
}

/** The tables whose rows belong to a workspace and are deleted with it (M3 design 3.6, 4.12); P7 adds the labels. */
const workspaceTables = [
  "workspace_members",
  "workspace_member_invites",
  "workspace_user_properties",
  "projects",
  "project_members",
  "project_user_properties",
  "states",
] as const;

/**
 * The tables whose rows a story can delete on their own before their workspace, which keep that moment: an
 * invitation, when it is accepted or deleted, or its address's membership ends (M3 design 3.8); a project, its
 * memberships, its members' display settings and its states, when the project is deleted (P4b).
````

````old e2e/fixtures/assert/workspace.ts
const workspaceTables: { table: string; deletedAlone: boolean }[] = [
  { table: "workspace_members", deletedAlone: false },
  { table: "workspace_member_invites", deletedAlone: true },
  { table: "workspace_user_properties", deletedAlone: false },
  { table: "projects", deletedAlone: true },
  { table: "project_members", deletedAlone: true },
  { table: "project_user_properties", deletedAlone: true },
  { table: "states", deletedAlone: true },
````
````new e2e/fixtures/assert/workspace.ts
export const deletedAloneTables: (typeof workspaceTables)[number][] = [
  "workspace_member_invites",
  "projects",
  "project_members",
  "project_user_properties",
  "states",
````

````old e2e/fixtures/assert/workspace.ts
 * W3: the workspace of slug is deleted by the account of adminEmail, and with it, at the same moment and by the
 * same account, every row under it that was not deleted before: its memberships, invitations and display
````
````new e2e/fixtures/assert/workspace.ts
 * W2, W3: the workspace of slug is deleted by the account of adminEmail, and with it, at the same moment and by
 * the same account, every row under it that was not deleted before: its memberships, invitations and display
````

````old e2e/fixtures/assert/workspace.ts
 * such a row; none is left undeleted; a table whose rows W3 deletes alone has rows deleted earlier, which kept
 * their moment, and every row of the others carries the workspace's.
````
````new e2e/fixtures/assert/workspace.ts
 * such a row; none is left undeleted; each table of deletedAlone, whose rows the story deleted alone, has rows
 * deleted earlier, which kept their moment, and every row of the others carries the workspace's.
````

````old e2e/fixtures/assert/workspace.ts
export async function expectWorkspaceDeleted(db: Database, slug: string, adminEmail: string): Promise<void> {
````
````new e2e/fixtures/assert/workspace.ts
export async function expectWorkspaceDeleted(
  db: Database,
  slug: string,
  adminEmail: string,
  deletedAlone: (typeof workspaceTables)[number][]
): Promise<void> {
````

````old e2e/fixtures/assert/workspace.ts
    workspaceTables.map(async ({ table }) => {
````
````new e2e/fixtures/assert/workspace.ts
    workspaceTables.map(async (table) => {
````

````old e2e/fixtures/assert/workspace.ts
    workspaceTables.map(({ table, deletedAlone }) => ({
````
````new e2e/fixtures/assert/workspace.ts
    workspaceTables.map((table) => ({
````

````old e2e/fixtures/assert/workspace.ts
      deletedEarlier: deletedAlone,
````
````new e2e/fixtures/assert/workspace.ts
      deletedEarlier: deletedAlone.includes(table),
````

````old e2e/fixtures/assert/workspace.ts
  );
}

````
````new e2e/fixtures/assert/workspace.ts
  );
}

/**
 * W2, W7, W12: the membership of the account of email in the workspace of slug has ended, by the account of byEmail:
 * its row kept, inactive, with its role; his memberships of the projects of projects (identifiers), each active
 * before, ended with it, at the same moment and by the same account, each row kept with its role; he has no other
 * membership of the workspace's projects that is active; and no invitation to his address in the workspace is
 * pending (M3 design 3.6, 3.8).
 */
export async function expectMembershipEnded(
  db: Database,
  slug: string,
  email: string,
  byEmail: string,
  role: number,
  projects: { identifier: string; role: number }[]
): Promise<void> {
  expect(
    await db.query(
      `SELECT m.role, m.is_active, m.deleted_at, b.email AS by
         FROM workspace_members m JOIN workspaces w ON w.id = m.workspace_id
         JOIN users u ON u.id = m.member_id JOIN users b ON b.id = m.updated_by_id
        WHERE w.slug = $1 AND u.email = $2`,
      [slug, email]
    ),
    `the membership of ${email} in ${slug}`
  ).toEqual([{ role, is_active: false, deleted_at: null, by: byEmail }]);
  // The database compares the moments: a Date holds milliseconds, a timestamptz microseconds.
  expect(
    await db.query(
      `SELECT p.identifier, pm.role, pm.is_active, pm.deleted_at, b.email AS by, pm.updated_at = m.updated_at AS with_it
         FROM project_members pm JOIN projects p ON p.id = pm.project_id
         JOIN workspace_members m ON m.workspace_id = pm.workspace_id AND m.member_id = pm.member_id
         JOIN workspaces w ON w.id = m.workspace_id JOIN users u ON u.id = m.member_id JOIN users b ON b.id = pm.updated_by_id
        WHERE w.slug = $1 AND u.email = $2 AND (pm.is_active OR pm.updated_at = m.updated_at)
        ORDER BY p.identifier COLLATE "C"`,
      [slug, email]
    ),
    `the memberships of ${email} of the projects of ${slug}, active or ended with it`
  ).toEqual(
    projects
      .toSorted((a, b) => (a.identifier < b.identifier ? -1 : 1))
      .map((p) => ({
        identifier: p.identifier,
        role: p.role,
        is_active: false,
        deleted_at: null,
        by: byEmail,
        with_it: true,
      }))
  );
  expect(
    await db.query(
      `SELECT i.email FROM workspace_member_invites i JOIN workspaces w ON w.id = i.workspace_id
        WHERE w.slug = $1 AND i.email = $2 AND i.responded_at IS NULL AND i.deleted_at IS NULL`,
      [slug, email]
    ),
    `the pending invitations to ${email} in ${slug}`
  ).toEqual([]);
}

/**
 * W2, W7, W12, before an ending: who wrote last the membership of the account of email in the workspace of slug
 * ("workspace") and each of his active memberships of its projects (by identifier), each the account of an address in
 * writers, so that a claim of the ending's writing them can fail.
 */
export async function expectWrittenLastBy(
  db: Database,
  slug: string,
  email: string,
  writers: Record<string, string>
): Promise<void> {
  const rows = await db.query<{ of: string; by: string }>(
    `SELECT 'workspace' AS of, b.email AS by
       FROM workspace_members m JOIN workspaces w ON w.id = m.workspace_id
       JOIN users u ON u.id = m.member_id JOIN users b ON b.id = m.updated_by_id
      WHERE w.slug = $1 AND u.email = $2 AND m.deleted_at IS NULL
     UNION ALL
     SELECT p.identifier, b.email
       FROM project_members pm JOIN projects p ON p.id = pm.project_id JOIN workspaces w ON w.id = p.workspace_id
       JOIN users u ON u.id = pm.member_id JOIN users b ON b.id = pm.updated_by_id
      WHERE w.slug = $1 AND u.email = $2 AND pm.is_active AND pm.deleted_at IS NULL`,
    [slug, email]
  );
  expect(
    Object.fromEntries(rows.map((r) => [r.of, r.by])),
    `who wrote last the memberships of ${email} in ${slug}`
  ).toEqual(writers);
}

````

`e2e/stories/workspace/w3-workspace-settings.spec.ts`（修改，2 处）：

````old e2e/stories/workspace/w3-workspace-settings.spec.ts
import { expectProjectCreated, expectProjectDeleted } from "../../fixtures/assert/project";
import {
````
````new e2e/stories/workspace/w3-workspace-settings.spec.ts
import { expectProjectCreated, expectProjectDeleted } from "../../fixtures/assert/project";
import {
  deletedAloneTables,
````

````old e2e/stories/workspace/w3-workspace-settings.spec.ts
  await expectWorkspaceDeleted(db, slug, adminEmail);
````
````new e2e/stories/workspace/w3-workspace-settings.spec.ts
  await expectWorkspaceDeleted(db, slug, adminEmail, deletedAloneTables);
````

- [ ] **Step 2: 故事**

`e2e/stories/workspace/w2-landing.spec.ts`（新文件，102 行）：

````file e2e/stories/workspace/w2-landing.spec.ts
import {
  addProjectMembers,
  createProject,
  createWorkspace,
  invite,
  inviteAndAccept,
  membershipOf,
  slugFor,
  type Workspace,
} from "../../fixtures/api";
import { expectMembershipEnded, expectWorkspaceDeleted, expectWrittenLastBy } from "../../fixtures/assert/workspace";
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// W2, where an account lands once signed in (M3 design 2). The landing rule is the web app's (P9); the API version
// checks the list it rests on.

/** A workspace as GET /api/v0/workspaces lists it for its member of role, with members active members. */
const entry = (w: Workspace, role: number, members: number) => ({
  slug: w.slug,
  role,
  total_members: members,
  created_at: w.created_at,
});

test("W2 (API): an account's workspaces are those it is an active member of, with its role, members and creation time; one fewer once it deletes one, none once it leaves the other after making a member its admin", async ({
  api,
  db,
}, testInfo) => {
  const aliceEmail = emailFor(testInfo, "alice");
  const alice = (await createPAT(api, (await register(api, aliceEmail)).access_token)).token;
  const bobEmail = emailFor(testInfo, "bob");
  const bob = (await createPAT(api, (await register(api, bobEmail)).access_token)).token;
  const bobId = await accountId(api, bob);
  const first = slugFor(testInfo, "first");
  const second = slugFor(testInfo, "second");
  // First holds a row of each table its deletion writes: alice's membership, a pending invitation, her display
  // settings, and her project Web with its own rows.
  const firstWorkspace = await createWorkspace(api, alice, { name: "First", slug: first });
  await invite(api, alice, first, [{ email: emailFor(testInfo, "invitee"), role: 15 }]);
  const settings = await api.PATCH("/api/v0/me/workspaces/{slug}/preferences", {
    params: { path: { slug: first } },
    body: { navigation_project_limit: 3 },
    headers: bearer(alice),
  });
  expect(settings.response.status).toBe(200);
  await createProject(api, alice, first, { name: "Web", identifier: "WEB" });
  // Second has bob as its member, and his project Ops, alice its admin by his adding, so both are its admins.
  const secondWorkspace = await createWorkspace(api, alice, { name: "Second", slug: second });
  await inviteAndAccept(api, alice, second, { email: bobEmail, token: bob }, 15);
  const ops = await createProject(api, bob, second, { name: "Ops", identifier: "OPS" });
  await addProjectMembers(api, bob, ops.id, [{ member_id: await accountId(api, alice), role: 20 }]);

  const listed = async (token: string) => {
    const { data, response } = await api.GET("/api/v0/workspaces", { headers: bearer(token) });
    expect(response.status).toBe(200);
    return data?.data.map((w) => ({
      slug: w.slug,
      role: w.role,
      total_members: w.total_members,
      created_at: w.created_at,
    }));
  };
  expect(await listed(alice)).toEqual([entry(firstWorkspace, 20, 1), entry(secondWorkspace, 20, 2)]);

  const deleted = await api.DELETE("/api/v0/workspaces/{slug}", {
    params: { path: { slug: first } },
    headers: bearer(alice),
  });
  expect(deleted.response.status).toBe(204);
  await expectWorkspaceDeleted(db, first, aliceEmail, []);
  expect(await listed(alice)).toEqual([entry(secondWorkspace, 20, 2)]);

  // Second's only admin cannot leave it; once bob is its admin too, she leaves.
  const alone = await api.POST("/api/v0/workspaces/{slug}/leave", {
    params: { path: { slug: second } },
    headers: bearer(alice),
  });
  expect([alone.response.status, alone.error?.code]).toEqual([409, "workspace.sole_admin"]);
  const promoted = await api.PATCH("/api/v0/workspace-members/{workspace_member_id}", {
    params: { path: { workspace_member_id: await membershipOf(api, alice, second, bobId) } },
    body: { role: 20 },
    headers: bearer(alice),
  });
  expect(promoted.response.status).toBe(200);
  // bob, an admin now, sets her role as it is: he wrote her memberships last, so that her leaving's writing shows.
  const rewritten = await api.PATCH("/api/v0/workspace-members/{workspace_member_id}", {
    params: { path: { workspace_member_id: await membershipOf(api, bob, second, await accountId(api, alice)) } },
    body: { role: 20 },
    headers: bearer(bob),
  });
  expect(rewritten.response.status).toBe(200);
  await expectWrittenLastBy(db, second, aliceEmail, { workspace: bobEmail, OPS: bobEmail });
  const left = await api.POST("/api/v0/workspaces/{slug}/leave", {
    params: { path: { slug: second } },
    headers: bearer(alice),
  });
  expect(left.response.status).toBe(204);
  await expectMembershipEnded(db, second, aliceEmail, aliceEmail, 20, [{ identifier: "OPS", role: 20 }]);
  expect(await listed(alice)).toEqual([]);
  expect(await listed(bob)).toEqual([entry(secondWorkspace, 20, 1)]);
});
````

`e2e/stories/workspace/w7-member-management.spec.ts`（新文件，217 行）：

````file e2e/stories/workspace/w7-member-management.spec.ts
import {
  accept,
  addProjectMembers,
  createProject,
  createWorkspace,
  invite,
  inviteAndAccept,
  listMembers,
  membershipOf,
  slugFor,
} from "../../fixtures/api";
import { expectMembership, expectMembershipEnded, expectWrittenLastBy } from "../../fixtures/assert/workspace";
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// W7, the members' management (M3 design 2). The page version comes with the members' page (P9).

test("W7 (API): the admin makes a member a guest in every project, removes another, ending her project memberships and no other invitation, and invites her back as a guest; nobody changes or removes his own membership, and the only admin cannot leave", async ({
  api,
  db,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = (await createPAT(api, (await register(api, adminEmail)).access_token)).token;
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin, { name: "Acme", slug });
  // carol's address had an invitation to acme, which the admin deleted before she joined.
  const carolEmail = emailFor(testInfo, "carol");
  const [revoked] = await invite(api, admin, slug, [{ email: carolEmail, role: 15 }]);
  if (!revoked) {
    throw new Error("the invitation to carol was not created");
  }
  const revoking = await api.DELETE("/api/v0/workspace-invitations/{invitation_id}", {
    params: { path: { invitation_id: revoked.id } },
    headers: bearer(admin),
  });
  expect(revoking.response.status).toBe(204);
  const people = Object.fromEntries(
    await Promise.all(
      ["bob", "carol", "dave", "erin"].map(async (name) => {
        const email = emailFor(testInfo, name);
        const token = (await createPAT(api, (await register(api, email)).access_token)).token;
        await inviteAndAccept(api, admin, slug, { email, token }, 15);
        return [name, { email, token, id: await accountId(api, token) }] as const;
      })
    )
  );
  const { bob, carol, dave, erin } = people as Record<
    "bob" | "carol" | "dave" | "erin",
    { email: string; token: string; id: string }
  >;
  // bob is the admin of a workspace of his own, Other, which invites carol; acme invites eve.
  const other = slugFor(testInfo, "other");
  await createWorkspace(api, bob.token, { name: "Other", slug: other });
  const [elsewhere] = await invite(api, bob.token, other, [{ email: carol.email, role: 15 }]);
  const [eves] = await invite(api, admin, slug, [{ email: emailFor(testInfo, "eve"), role: 5 }]);
  if (!elsewhere || !eves) {
    throw new Error("the invitations to carol in Other and to eve in acme were not created");
  }
  // bob and carol join Web; Ops is bob's own, carol its member; Docs is carol's own, dave its member, and erin was its
  // admin until she left acme.
  const join = async (projectId: string, token: string) =>
    (
      await api.POST("/api/v0/projects/{project_id}/join", {
        params: { path: { project_id: projectId } },
        headers: bearer(token),
      })
    ).response.status;
  const web = await createProject(api, admin, slug, { name: "Web", identifier: "WEB" });
  const ops = await createProject(api, bob.token, slug, { name: "Ops", identifier: "OPS" });
  const docs = await createProject(api, carol.token, slug, { name: "Docs", identifier: "DOCS" });
  expect(
    [
      await join(web.id, bob.token),
      await join(web.id, carol.token),
      await join(ops.id, carol.token),
      await join(docs.id, dave.token),
    ],
    "bob and carol join Web, carol Ops, dave Docs"
  ).toEqual([200, 200, 200, 200]);
  await addProjectMembers(api, carol.token, docs.id, [{ member_id: erin.id, role: 20 }]);
  const erinLeaves = await api.POST("/api/v0/workspaces/{slug}/leave", {
    params: { path: { slug } },
    headers: bearer(erin.token),
  });
  expect(erinLeaves.response.status, "erin leaves acme").toBe(204);
  const membership = (id: string) => membershipOf(api, admin, slug, id);
  const adminId = await accountId(api, admin);
  // The account of email's memberships of acme's projects, ended ones too: each one's role, whether it is active,
  // and who wrote it last.
  const projectRoles = (email: string) =>
    db.query(
      `SELECT p.identifier, m.role, m.is_active, b.email AS by
         FROM project_members m JOIN projects p ON p.id = m.project_id JOIN workspaces w ON w.id = p.workspace_id
         JOIN users u ON u.id = m.member_id JOIN users b ON b.id = m.updated_by_id
        WHERE w.slug = $1 AND u.email = $2 AND m.deleted_at IS NULL ORDER BY p.identifier COLLATE "C"`,
      [slug, email]
    );

  // Nobody changes or removes his own membership; the only admin cannot leave, though Other has an admin.
  const own = await membership(adminId);
  const ownChange = await api.PATCH("/api/v0/workspace-members/{workspace_member_id}", {
    params: { path: { workspace_member_id: own } },
    body: { role: 15 },
    headers: bearer(admin),
  });
  const ownRemoval = await api.DELETE("/api/v0/workspace-members/{workspace_member_id}", {
    params: { path: { workspace_member_id: own } },
    headers: bearer(admin),
  });
  const leaving = await api.POST("/api/v0/workspaces/{slug}/leave", {
    params: { path: { slug } },
    headers: bearer(admin),
  });
  expect(
    [ownChange, ownRemoval, leaving].map((r) => [r.response.status, r.error?.code]),
    "the admin's own change, his own removal, his leaving"
  ).toEqual([
    [409, "workspace.own_membership"],
    [409, "workspace.own_membership"],
    [409, "workspace.sole_admin"],
  ]);
  await expectMembership(db, slug, adminEmail, { role: 20, is_active: true });

  // bob becomes a guest, and a guest in each of his projects: Ops is left without an admin. He wrote each of those
  // rows last, so that the admin's writing them shows.
  await expectWrittenLastBy(db, slug, bob.email, { workspace: bob.email, OPS: bob.email, WEB: bob.email });
  const demoted = await api.PATCH("/api/v0/workspace-members/{workspace_member_id}", {
    params: { path: { workspace_member_id: await membership(bob.id) } },
    body: { role: 5 },
    headers: bearer(admin),
  });
  expect(demoted.response.status).toBe(200);
  await expectMembership(db, slug, bob.email, { role: 5, is_active: true });
  expect(await projectRoles(bob.email), "bob's memberships of the projects").toEqual([
    { identifier: "OPS", role: 5, is_active: true, by: adminEmail },
    { identifier: "WEB", role: 5, is_active: true, by: adminEmail },
  ]);

  // carol, Docs's only admin beside dave, cannot be removed until the admin joins Docs; then her memberships end,
  // their rows kept, and no invitation but a pending one to her address in acme is touched: none is.
  const carolsMembership = await membership(carol.id);
  const remove = () =>
    api.DELETE("/api/v0/workspace-members/{workspace_member_id}", {
      params: { path: { workspace_member_id: carolsMembership } },
      headers: bearer(admin),
    });
  const refused = await remove();
  expect([refused.response.status, refused.error?.code]).toEqual([409, "project.sole_admin"]);
  await expectMembership(db, slug, carol.email, { role: 15, is_active: true });
  expect(await join(docs.id, admin), "the admin joins Docs").toBe(200);
  const invitations = () =>
    db.query("SELECT to_jsonb(i) AS row FROM workspace_member_invites i WHERE i.id = ANY ($1) ORDER BY i.id", [
      [revoked.id, elsewhere.id, eves.id],
    ]);
  const untouched = await invitations();
  expect(
    await db.query(
      `SELECT email FROM workspace_member_invites
        WHERE id = ANY ($1) AND responded_at IS NULL AND deleted_at IS NULL ORDER BY email`,
      [[elsewhere.id, eves.id]]
    ),
    "the invitations to carol in Other and to eve in acme, pending before the removal"
  ).toEqual([{ email: carol.email }, { email: emailFor(testInfo, "eve") }]);
  await expectWrittenLastBy(db, slug, carol.email, {
    workspace: carol.email,
    DOCS: carol.email,
    OPS: carol.email,
    WEB: carol.email,
  });
  expect((await remove()).response.status).toBe(204);
  await expectMembershipEnded(db, slug, carol.email, adminEmail, 15, [
    { identifier: "DOCS", role: 20 },
    { identifier: "OPS", role: 15 },
    { identifier: "WEB", role: 15 },
  ]);
  expect(await invitations(), "the deleted invitation to carol, hers to Other, eve's").toEqual(untouched);
  expect(
    (await listMembers(api, admin, slug)).map((m) => [m.member.id, m.is_active]),
    "acme's members, carol's membership ended"
  ).toContainEqual([carol.id, false]);
  const webMembers = await api.GET("/api/v0/projects/{project_id}/members", {
    params: { path: { project_id: web.id } },
    headers: bearer(admin),
  });
  expect(webMembers.data?.data.map((m) => m.member_id).toSorted()).toEqual([adminId, bob.id].toSorted());

  // Invited back as a guest, she accepts: her membership's row is active again as a guest's, and her ended project
  // memberships stay ended, a guest's: she sees none of the projects.
  const [invitation] = await invite(api, admin, slug, [{ email: carol.email, role: 5 }]);
  if (!invitation) {
    throw new Error("the invitation to carol was not created");
  }
  expect(await accept(api, carol.token, invitation)).toMatchObject({ slug, role: 5 });
  expect(await membership(carol.id), "carol's membership, restored").toBe(carolsMembership);
  await expectMembership(db, slug, carol.email, { role: 5, is_active: true });
  expect(await projectRoles(carol.email), "carol's memberships of the projects").toEqual([
    { identifier: "DOCS", role: 5, is_active: false, by: carol.email },
    { identifier: "OPS", role: 5, is_active: false, by: carol.email },
    { identifier: "WEB", role: 5, is_active: false, by: carol.email },
  ]);
  const reads = await Promise.all(
    [web, ops, docs].map((project) =>
      api.GET("/api/v0/projects/{project_id}", {
        params: { path: { project_id: project.id } },
        headers: bearer(carol.token),
      })
    )
  );
  expect(
    reads.map((read) => [read.response.status, read.error?.code]),
    "carol reads Web, Ops and Docs"
  ).toEqual([
    [404, "project.not_found"],
    [404, "project.not_found"],
    [404, "project.not_found"],
  ]);
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
Expected: 64 个故事全部通过（此前的 62 个，加 W2、W7）。

- [ ] **Step 4: 提交**

```bash
git add e2e/fixtures/api.ts e2e/fixtures/assert/workspace.ts e2e/stories/workspace/w2-landing.spec.ts e2e/stories/workspace/w3-workspace-settings.spec.ts e2e/stories/workspace/w7-member-management.spec.ts
```
```bash
git commit -m "test(M3/P5a): the API versions of W2 and W7

W2 lists an account's workspaces until it deletes one and leaves the
other, refused while it is the only admin. W7 demotes a member to a
guest, removes another, refused while she is a project's only admin,
and invites her back. Each ending is checked against rows another
account wrote last, and expectWorkspaceDeleted takes the tables a story
deleted rows of alone.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A 清扫 1 的故事一半、清扫 14；每个都只运行它的故事）：

| 改坏 | 必须失败的故事 |
|---|---|
| `EndMember` 去掉成员 | W2、W7 |
| `HasOtherAdmin` 去掉成员、角色 | W2、W7；`HasOtherAdmin` 去掉工作区：W7 |
| `DeletePendingInvitations` 去掉工作区、地址、`deleted_at IS NULL` | W7 |
| `SoleAdmin` 去掉成员、角色，另一个管理员的项目、成员、角色、有效、存在 | W7（另一个管理员的存在：W2、W7） |
| `EndMemberships` 去掉成员 | W7 |
| 项目的 `ListMembers` 去掉 `is_active`；`ProjectFacts` 去掉 `m.is_active` | W7 |
| `EndMember`、`EndMemberships` 不写结束者；降为访客不写降级者 | W2、W7（降级者：W7） |
| 唯一的管理员照样离开；结束一步跳过项目 | W2、W7 |
| 移出自己的成员关系；唯一管理员时照样结束 | W7 |

**Done when:** 64 个故事全部通过。

---

### Task 14: 端到端：W12；文档

**Files:**
- Create: `e2e/stories/workspace/w12-reactivate-member.spec.ts`
- Modify: `README.md`、`docs/v0/plane-diff.md`

**Interfaces:**（spec 2.14、2.15，M3 设计第 2 节 W12，3.20、8.7）
- W12：`nerve workspaces reactivate-member` 从故事里运行（`fixtures/workspaces.ts` 的 `nerveWorkspaces`、`nerveWorkspacesFails`，P1）。
- `README.md` 部署一节的"恢复被移出的成员"（8.7；G1 (a)：命令没有请求期限，同一工作区里项目级的写连续重叠时一直等，中断之后什么都不改，可以重试）。
- `docs/v0/plane-diff.md` 第四节 P5a 的三行（4.11、3.20）：唯一管理员的检查、恢复被移出的成员、移出和离开时的待接受邀请。

**Tests:**
- W12：B 是 acme 的第二位管理员，管理 Ops 和 Old；carol 加入 Ops 后离开 acme。B 在 Other 是成员、在那里管理 Lab，carol 是 Lab 的成员；Spare 有一个发给 B 的待接受邀请。A 移出 B：他在 acme、Old、Ops 的成员关系结束（之前由他自己写），他在 Other、Lab 的和 Spare 的邀请不变；A 不能离开。A 加入 Old 并删除它；carol、B 离开 Other；B 的账户被停用；A 以访客邀请他回来（旧邀请）。不存在的工作区、没有账户的地址、从来不是成员的账户：退出码 1，`workspace_members` 不变。命令（地址大写）恢复 B 在 acme 的行，仍是管理员，`updated_by_id` 仍是 A，Ops 仍结束，输出说 1 个、下一步是 `nerve users activate`；B 激活之后再跑一次：已是有效，什么都不改。B 加入 Ops：同一行、管理员、由他；carol 在 Ops 的行不变。A 离开；B 接受旧邀请：角色 20，成员数 1，邀请已接受并在回答时删除。B 邀请 carol、她拒绝；命令恢复 carol；B 再移出她：已拒绝的邀请不变。

- [ ] **Step 1: 故事**

`e2e/stories/workspace/w12-reactivate-member.spec.ts`（新文件，213 行）：

````file e2e/stories/workspace/w12-reactivate-member.spec.ts
import {
  accept,
  createProject,
  createWorkspace,
  invite,
  inviteAndAccept,
  membershipOf,
  slugFor,
} from "../../fixtures/api";
import { expectMembership, expectMembershipEnded, expectWrittenLastBy } from "../../fixtures/assert/workspace";
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";
import { nerveUsers } from "../../fixtures/users";
import { nerveWorkspaces, nerveWorkspacesFails } from "../../fixtures/workspaces";

// W12, the administrator restores a removed member (M3 design 2, 3.11), with the old invitation of Codex S2: an
// invitation never changes an active membership (3.8). There is no page version.

/** The arguments of nerve workspaces reactivate-member for the account of email in the workspace of slug. */
const reactivate = (slug: string, email: string) => ["reactivate-member", "--slug", slug, "--email", email];

test("W12: nerve workspaces reactivate-member restores a removed admin's membership with its role, his project membership ended until he joins; an old guest invitation then changes nothing; an unknown workspace or account, or a never-member, changes nothing", async ({
  api,
  db,
}, testInfo) => {
  const aEmail = emailFor(testInfo, "a");
  const a = (await createPAT(api, (await register(api, aEmail)).access_token)).token;
  const bEmail = emailFor(testInfo, "b");
  const b = (await createPAT(api, (await register(api, bEmail)).access_token)).token;
  const carolEmail = emailFor(testInfo, "carol");
  const carol = (await createPAT(api, (await register(api, carolEmail)).access_token)).token;
  const strangerEmail = emailFor(testInfo, "stranger");
  await register(api, strangerEmail);
  const slug = slugFor(testInfo);
  await createWorkspace(api, a, { name: "Acme", slug });
  await inviteAndAccept(api, a, slug, { email: bEmail, token: b }, 20);
  await inviteAndAccept(api, a, slug, { email: carolEmail, token: carol }, 15);
  // B, acme's second admin, is the admin of Ops and of Old; carol joins Ops, then leaves acme, which ends her
  // membership of Ops.
  const ops = await createProject(api, b, slug, { name: "Ops", identifier: "OPS" });
  const old = await createProject(api, b, slug, { name: "Old", identifier: "OLD" });
  const join = async (projectId: string, token: string) =>
    (
      await api.POST("/api/v0/projects/{project_id}/join", {
        params: { path: { project_id: projectId } },
        headers: bearer(token),
      })
    ).response.status;
  expect(await join(ops.id, carol), "carol joins Ops").toBe(200);
  const leave = async (target: string, token: string) => {
    const left = await api.POST("/api/v0/workspaces/{slug}/leave", {
      params: { path: { slug: target } },
      headers: bearer(token),
    });
    return [left.response.status, left.error?.code];
  };
  expect(await leave(slug, carol), "carol leaves acme").toEqual([204, undefined]);
  // B is also Other's member and Lab's admin there, carol Lab's member; Spare, A's third workspace, has a pending
  // invitation to him.
  const other = slugFor(testInfo, "other");
  const spare = slugFor(testInfo, "spare");
  await createWorkspace(api, a, { name: "Other", slug: other });
  await inviteAndAccept(api, a, other, { email: bEmail, token: b }, 15);
  const lab = await createProject(api, b, other, { name: "Lab", identifier: "LAB" });
  await inviteAndAccept(api, a, other, { email: carolEmail, token: carol }, 15);
  expect(await join(lab.id, carol), "carol joins Lab").toBe(200);
  await createWorkspace(api, a, { name: "Spare", slug: spare });
  const [spares] = await invite(api, a, spare, [{ email: bEmail, role: 15 }]);
  if (!spares) {
    throw new Error("the invitation to B in Spare was not created");
  }
  const bMembership = await membershipOf(api, a, slug, await accountId(api, b));
  // The membership of the account of email of the project of projectId: the row's id, its role, whether it is
  // active, and who wrote it last.
  const projectRow = async (projectId: string, email: string) =>
    (
      await db.query<{ id: string }>(
        `SELECT m.id, m.role, m.is_active, w.email AS by FROM project_members m
           JOIN users u ON u.id = m.member_id JOIN users w ON w.id = m.updated_by_id WHERE m.project_id = $1 AND u.email = $2`,
        [projectId, email]
      )
    )[0];
  // B's membership of acme and his of Ops, as projectRow has them.
  const bs = async () => {
    const [workspace] = await db.query(
      `SELECT m.id, m.role, m.is_active, w.email AS by FROM workspace_members m JOIN workspaces s ON s.id = m.workspace_id
         JOIN users u ON u.id = m.member_id JOIN users w ON w.id = m.updated_by_id WHERE s.slug = $1 AND u.email = $2`,
      [slug, bEmail]
    );
    return { workspace, project: await projectRow(ops.id, bEmail) };
  };
  const opsMembership = (await bs()).project?.id;
  const carolsOps = await projectRow(ops.id, carolEmail);
  // The one table the command writes.
  const memberships = () => db.query("SELECT * FROM workspace_members ORDER BY id");

  // A removes B: his memberships of acme end, their rows kept, and nothing of his elsewhere changes. He wrote each of
  // them last, so that A's writing them shows.
  await expectWrittenLastBy(db, slug, bEmail, { workspace: bEmail, OLD: bEmail, OPS: bEmail });
  const removed = await api.DELETE("/api/v0/workspace-members/{workspace_member_id}", {
    params: { path: { workspace_member_id: bMembership } },
    headers: bearer(a),
  });
  expect(removed.response.status).toBe(204);
  await expectMembershipEnded(db, slug, bEmail, aEmail, 20, [
    { identifier: "OLD", role: 20 },
    { identifier: "OPS", role: 20 },
  ]);
  await expectMembership(db, other, bEmail, { role: 15, is_active: true });
  expect(await projectRow(lab.id, bEmail), "B's membership of Lab").toMatchObject({
    role: 20,
    is_active: true,
    by: bEmail,
  });
  expect(
    await db.query("SELECT responded_at, deleted_at FROM workspace_member_invites WHERE id = $1", [spares.id]),
    "the invitation to B in Spare"
  ).toEqual([{ responded_at: null, deleted_at: null }]);
  // A, now acme's only active admin, cannot leave it.
  expect(await leave(slug, a), "A leaves acme").toEqual([409, "workspace.sole_admin"]);
  // A joins Old and deletes it, B's membership with it. carol leaves Other, then B; his account is deactivated;
  // meanwhile A invites him back to acme as a guest: the old invitation.
  expect(await join(old.id, a), "A joins Old").toBe(200);
  const oldDeleted = await api.DELETE("/api/v0/projects/{project_id}", {
    params: { path: { project_id: old.id } },
    headers: bearer(a),
  });
  expect(oldDeleted.response.status, "A deletes Old").toBe(204);
  expect(await leave(other, carol), "carol leaves Other").toEqual([204, undefined]);
  expect(await leave(other, b), "B leaves Other").toEqual([204, undefined]);
  await nerveUsers(db, ["deactivate", "--email", bEmail]);
  const [oldInvitation] = await invite(api, a, slug, [{ email: bEmail, role: 5 }]);
  if (!oldInvitation) {
    throw new Error("the invitation to B was not created");
  }

  // An unknown workspace, an unknown account and an account that was never acme's member change nothing.
  const before = await memberships();
  await nerveWorkspacesFails(db, reactivate(slugFor(testInfo, "nowhere"), bEmail), "No workspace has this slug.");
  await nerveWorkspacesFails(db, reactivate(slug, emailFor(testInfo, "nobody")), "No account has this e-mail address.");
  await nerveWorkspacesFails(
    db,
    reactivate(slug, strangerEmail),
    "The account has never been a member of this workspace."
  );
  expect(await memberships(), "the memberships after the refusals").toEqual(before);

  // The command restores B's row of acme alone, with its role, though his account is deactivated, and says what is
  // next: his membership of Ops is the one of acme's projects still ended, Old's deleted.
  expect(await nerveWorkspaces(db, reactivate(slug, bEmail.toUpperCase()))).toBe(
    `reactivated ${bEmail} in ${slug} as admin; project memberships still ended: 1, each restored when the member joins its project; the account is deactivated: run nerve users activate --email ${bEmail} next\n`
  );
  expect(await bs(), "B's memberships, reactivated").toEqual({
    workspace: { id: bMembership, role: 20, is_active: true, by: aEmail },
    project: { id: opsMembership, role: 20, is_active: false, by: aEmail },
  });
  await expectMembership(db, slug, carolEmail, { role: 15, is_active: false });
  await expectMembership(db, other, bEmail, { role: 15, is_active: false });
  await nerveUsers(db, ["activate", "--email", bEmail]);
  const reactivated = await memberships();
  expect(await nerveWorkspaces(db, reactivate(slug, bEmail))).toBe(
    `${bEmail} is an active member of ${slug} already; nothing changed\n`
  );
  expect(await memberships(), "the memberships after the second run").toEqual(reactivated);

  // B joins Ops: his old row, an admin's, the lower of its role and his workspace role (3.5); carol's stays ended.
  expect(await join(ops.id, b), "B joins Ops").toBe(200);
  const restored = {
    workspace: { id: bMembership, role: 20, is_active: true, by: aEmail },
    project: { id: opsMembership, role: 20, is_active: true, by: bEmail },
  };
  expect(await bs(), "B's memberships, Ops joined").toEqual(restored);
  expect(await projectRow(ops.id, carolEmail), "carol's membership of Ops").toEqual(carolsOps);

  // A leaves, and B, acme's only admin, accepts the old guest invitation: it is consumed, and changes nothing.
  expect(await leave(slug, a), "A leaves acme").toEqual([204, undefined]);
  expect(await accept(api, b, oldInvitation)).toMatchObject({ slug, role: 20, total_members: 1 });
  expect(await bs(), "B's memberships, the old invitation accepted").toEqual(restored);
  // The database compares the moments: a Date holds milliseconds, a timestamptz microseconds.
  expect(
    await db.query(
      `SELECT accepted, responded_at IS NOT NULL AS responded, deleted_at = responded_at AS deleted_when_answered
         FROM workspace_member_invites WHERE id = $1`,
      [oldInvitation.id]
    ),
    "the old invitation"
  ).toEqual([{ accepted: true, responded: true, deleted_when_answered: true }]);

  // carol declines B's invitation back; reactivated by the command, she keeps the declined invitation, and so she
  // does when B removes her again: an ending deletes pending invitations alone.
  const [declined] = await invite(api, b, slug, [{ email: carolEmail, role: 5 }]);
  if (!declined) {
    throw new Error("the invitation to carol was not created");
  }
  const declining = await api.POST("/api/v0/workspace-invitations/{invitation_id}/decline", {
    params: { path: { invitation_id: declined.id } },
    body: { token: declined.token },
    headers: bearer(carol),
  });
  expect(declining.response.status).toBe(204);
  expect(await nerveWorkspaces(db, reactivate(slug, carolEmail))).toBe(
    `reactivated ${carolEmail} in ${slug} as member; project memberships still ended: 1, each restored when the member joins its project\n`
  );
  const declinedRow = () =>
    db.query("SELECT to_jsonb(i) AS row FROM workspace_member_invites i WHERE i.id = $1", [declined.id]);
  const kept = await declinedRow();
  const removedAgain = await api.DELETE("/api/v0/workspace-members/{workspace_member_id}", {
    params: { path: { workspace_member_id: await membershipOf(api, b, slug, await accountId(api, carol)) } },
    headers: bearer(b),
  });
  expect(removedAgain.response.status).toBe(204);
  expect(await declinedRow(), "the declined invitation to carol").toEqual(kept);
});
````

- [ ] **Step 2: 文档**

`README.md`（修改，1 处）：

````old README.md
- **建工作区**：`workspace.creation_enabled`（默认 `true`）决定用户能否经接口建工作区；关闭时，经过认证、格式正确的 `POST /api/v0/workspaces` 答 403 `workspace.creation_disabled`，前端隐藏入口。这时由服务器管理员建：`nerve workspaces create --slug <slug> --name <名称> --admin-email <邮箱>`，不受这个开关限制，建出的工作区与接口建的相同（规则、唯一性、管理员成员），这个邮箱的账户是它的管理员和唯一成员。它与 `nerve users` 一样直接连数据库，服务不用停；邮箱按注册时的规则规范化。slug 格式不对、已被占用或是保留名、名称不合规、邮箱没有账户、账户已停用时，退出码为 1，打印一行说明，数据库不变。
````
````new README.md
- **建工作区**：`workspace.creation_enabled`（默认 `true`）决定用户能否经接口建工作区；关闭时，经过认证、格式正确的 `POST /api/v0/workspaces` 答 403 `workspace.creation_disabled`，前端隐藏入口。这时由服务器管理员建：`nerve workspaces create --slug <slug> --name <名称> --admin-email <邮箱>`，不受这个开关限制，建出的工作区与接口建的相同（规则、唯一性、管理员成员），这个邮箱的账户是它的管理员和唯一成员。它与 `nerve users` 一样直接连数据库，服务不用停；邮箱按注册时的规则规范化。slug 格式不对、已被占用或是保留名、名称不合规、邮箱没有账户、账户已停用时，退出码为 1，打印一行说明，数据库不变。
- **恢复被移出的成员**：`nerve workspaces reactivate-member --slug <slug> --email <邮箱>` 把这个账户在这个工作区已结束的成员关系（被移出或自己离开）恢复为有效，角色不变。他在这个工作区的项目成员关系仍无效，由他经接口加入项目时恢复（角色取原来的项目角色与他的工作区角色中较低的一个），命令输出这样的项目成员关系还有几个。账户已停用时照样恢复，输出另外提示下一步执行 `nerve users activate --email <邮箱>`：`nerve users activate` 只恢复账户，不恢复成员关系。它与 `nerve workspaces create` 一样直接连数据库，服务不用停；邮箱按注册时的规则规范化。已是有效成员时输出说明，退出码为 0，什么都不改；工作区不存在、邮箱没有账户、账户从来不是这个工作区的成员时，退出码为 1，打印一行说明，数据库不变。它取工作区的锁：同一工作区里项目级的写连续重叠时一直等待（命令没有请求期限）；中断（SIGINT、SIGTERM）之后什么都不改，可以重试。
````

`docs/v0/plane-diff.md`（修改，1 处）：

````old docs/v0/plane-diff.md
| 接受邀请时已有成员行 | 不分有效还是已离开，都把角色改为邀请的角色 | 已是有效成员：只消费邀请，成员关系和角色不变；以前的成员行：恢复，角色取邀请的；取的是访客时，同一个事务里他在这个工作区的项目角色都改为访客，含已离开的项目（M3 设计 3.8） |
````
````new docs/v0/plane-diff.md
| 接受邀请时已有成员行 | 不分有效还是已离开，都把角色改为邀请的角色 | 已是有效成员：只消费邀请，成员关系和角色不变；以前的成员行：恢复，角色取邀请的；取的是访客时，同一个事务里他在这个工作区的项目角色都改为访客，含已离开的项目（M3 设计 3.8） |
| 移出成员、离开工作区时的唯一管理员检查 | 移出：拿工作区成员的 `id` 去比项目成员的 `member_id`，永远不会命中，查的又是"只有一个成员"的项目；离开：查"只有他一个成员、而他是管理员的项目"，与它的提示语相反 | 他是这个工作区里某个项目唯一的有效管理员、而那个项目还有别的有效成员时，移出和离开都答 409 `project.sole_admin`，什么都不改；那里只有他一人时允许。离开另有一条，与 Plane 相同：工作区唯一的有效管理员不能离开，哪怕只有他一人，409 `workspace.sole_admin`（M3 设计 3.7） |
| 恢复被移出的成员 | 管理命令 `reactivate_workspace_member`（位置参数） | `nerve workspaces reactivate-member --slug <slug> --email <邮箱>`，行为照搬：已结束的成员关系恢复为有效，角色不变，停用的账户照样恢复；项目成员关系仍无效，他经接口加入项目时恢复（M3 设计 3.5、3.11） |
| 移出成员、离开工作区时发给他的待接受邀请 | 不动：他凭旧链接就能回来 | 同一个事务里软删除这个工作区里发给他邮箱的待接受邀请（已忽略的不动），回来要新的邀请（M3 设计 3.8） |
````

- [ ] **Step 3: 检查**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

Run: `make lint-web`
Expected: 通过（关键词守卫也查文档）。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 65 个故事全部通过。

- [ ] **Step 4: 提交**

```bash
git add README.md docs/v0/plane-diff.md e2e/stories/workspace/w12-reactivate-member.spec.ts
```
```bash
git commit -m "test(M3/P5a): W12; the README's reactivation and plane-diff's P5a rows

W12 removes an admin, refuses the command for an unknown workspace or
account or a never-member without a change, reactivates the removed
admin while his account is deactivated, and shows that the old guest
invitation he accepts afterwards changes nothing. The README's
deployment section gains the command, which waits for the workspace's
lock without a deadline; plane-diff gains the sole-admin checks, the
command, and the invitations deleted on removal and leaving.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A；每个都只运行 W12）：

| 改坏 | 必须失败的故事 |
|---|---|
| `EndMember`、`ReactivateMember`、`CountInactive` 各去掉工作区、成员；`CountInactive` 去掉 `deleted_at IS NULL` | W12 |
| `HasOtherAdmin` 去掉成员、有效 | W12 |
| `DeletePendingInvitations` 去掉工作区、地址、`responded_at IS NULL` | W12 |
| `LockActiveMemberProjects` 去掉工作区 | W12 |
| `SoleAdmin` 去掉项目、成员；另一个成员的项目、成员、有效、存在 | W12 |
| `EndMemberships` 去掉项目、成员 | W12 |
| 项目的 `RestoreMember` 去掉 id | W12 |
| `EndMember`、`EndMemberships` 不写结束者 | W12 |
| 唯一的管理员照样离开；结束一步跳过项目；停用的账户被拒；已是有效的也写 | W12 |

**Done when:** 65 个故事全部通过；README 和差异清单写好。
