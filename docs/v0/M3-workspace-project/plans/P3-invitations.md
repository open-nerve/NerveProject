# M3/P3 邀请与凭邀请注册 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 邀请的全部操作可用：工作区管理员批量邀请邮箱（一批全有或全无）、列出（带链接的令牌）、改角色、删除；持链接的人公开地查看邀请（没有邮箱）；登录的账户凭链接接受、忽略（约定六的加锁顺序：账户行 `FOR SHARE` 最先）；注册关闭时凭有效邀请注册。令牌由签名密钥按用途派生的 MAC 算出，不存库。删除工作区连带邀请。交错 3、9、12、18、19 两个顺序都在真实数据库上测过。矩阵加 10 行（60 格），公开的查看由它自己的测试覆盖；W3–W6 的接口版本和 W8 的第二个成员通过；总体设计 1.1、4.2、差异清单、README 和 M2 收尾交接中 P3 的各行写好。

**Architecture:** 新迁移 `00009`（`workspace_member_invites`）。`identity`：`LoadKeys` 移到 `bootstrap` 的第 1 步，`identity.Keys.MAC(purpose)` 按用途派生 MAC（拒绝刷新令牌的用途）；`Provide` 交出 `CredentialLock`；`SignupPolicy.AllowSignup(ctx, email, invitation)`、`RegisterRequest.invitation`。`workspace`：领域的 `Invitation`、`InvitationPreview`、`CheckInvitations`、令牌（`InvitationMessage`、`FormatToken`、`ParseToken`）；端口 `InvitationMAC`、`CallerLock` 和按用例分的邀请端口；存储的邀请读写、`ShareWorkspace`（按 id 的 `FOR SHARE`）、`LockInvitation`（`FOR UPDATE`）、回应要的 `MemberOf`、`RestoreMember`、`WorkspaceByID`；七个用例和它们共用的 `invitationTokens`、`readInvitation`/`lockInvitation`、`responder`；`SignupInvitations`（注册的检查）。`bootstrap`：邀请的 MAC、`CallerLock` 接进 `workspace`，`signupPolicy` 组合开关和 `workspace` 的检查，公开操作并进列表。`access` 的规则表加 4 行。测试一侧：`apitest` 的对象的数组，`pgtest.WaitForKeyWaitOn`（唯一索引上的等待），矩阵的"未知参数"和公开操作的豁免，P2 的闸门加期限。

**Tech Stack:** Go 1.27.1、pgx v5.11.0、sqlc v1.31.1（`CGO_ENABLED=0`）、oapi-codegen v2.8.0、goose v3.28.0、River v0.47.0、golangci-lint 2.13.2、PostgreSQL 18.6（testcontainers）；Node 24、pnpm 11.10.0、Playwright 1.63.0。

**Spec:** `docs/v0/M3-workspace-project/specs/P3-invitations.md`（上级：`docs/v0/M3-workspace-project/M3-design.md`）

## Global Constraints

- **Go 版本和依赖**：本 plan 不执行 `go get`，不加任何 Go 模块或 npm 包。`server/go.mod` 和 `server/tools/go.mod` 保持 `go 1.27` 和 `toolchain go1.27.1`（golangci-lint 2.13.2 由 go1.27.0 构建，`go` 行更高就拒绝运行）。每个 Task 提交前执行 `grep -n "^go \|^toolchain" server/go.mod server/tools/go.mod`，四行必须是 `go 1.27` 和 `toolchain go1.27.1`；`git diff --stat 1d2eaf7c -- server/go.mod server/go.sum server/tools pnpm-lock.yaml` 在本 plan 的任何时刻都没有输出。
- **每个 Task 提交前**：`make lint-go` 两段都输出 `0 issues.`，`make test` 全部通过（没有 `FAIL`）。
- **生成的文件不手写、不从本 plan 复制**：有生成物的 Task（1、5–12）执行 Task 中的生成命令（改了接口描述的 Task 5、6、8、9、11、12 执行 `make gen`，只动了 sqlc 的 Task 1、7、10 执行 `make gen-go`），用 `shasum -a 256` 和 `wc -l` 核对表中的 SHA-256 和行数，对不上时停下来：说明某个输入与本 plan 不一致。这些 Task **提交之后**执行 `make gen-check`（它用 `git status` 判断生成物是否已提交，提交之前必然报差异）。
- **前端检查**：改了接口描述、前端或 `e2e/` 的 Task（5、6、8、9、11、12、14）另执行 `make lint-web`、`make knip`、`make test-web`；声明新错误码的 Task 8、11 在同一个 Task 里把码加进 `PROBLEM_MESSAGES` 和两份 `auth.json`（M3 设计 12 节约束 4）；Task 15 只改文档，执行 `make lint-web`（关键词守卫也查文档）；Task 14 执行 `make e2e`。
- **容器**：`make test` 和 `make e2e` 用自己的 testcontainers；机器忙时偶尔起不来，等 Docker 空闲之后重跑一次再当作失败。开发库 `nerve-dev-db-1` 可以用，但不要停止或重建它，不要执行 `make dev-db-down`、`make dev-db-reset`。不要碰其他项目的容器（`agentforge-*`、`plane-app-*`、`opennerve-*`）。
- **git**：每次 Bash 调用只执行一个 git 命令，不用 `;`、`&&`、`|` 串联 git；不用 `git -C`、`stash`、`clean`、`reset --hard`。`cd` 不与别的命令组合。不碰 `plane/`、`refer/`。
- **安装**：除了 Docker、Go、Node 不做任何全局安装；不执行 `corepack enable`（pnpm 已在 PATH 上）。
- **规则**：不写 `init()`，不用全局可变状态，构造函数显式传入依赖；不建 `utils`、`common`、`helpers` 包；一个文件只做一件事，不超过约 400 行；依赖只能向内；模块之间不互相导入，值在 `bootstrap` 中转换或直接接上（M3 设计 6.5、6.6）；模块的 SQL 只经 sqlc（`TestModulesRunSQLOnlyThroughSQLC`）；不留没有使用者的代码。**例外**：接口描述按模块一个文件（M0-P3 交接 5），`api/modules/*.yaml` 不受约 400 行的限制。本 plan 的其余文件都在约 400 行以内：最长的是 `workspace/app/fakes_test.go`（401 行，P2 的共用假实现，本 plan 加 10 行）和 `bootstrap/interleaving_answers_test.go`（398 行）；`apitest/operations.go` 因为对象的数组要长到 411 行，Task 3 把请求体用例移到 `bodycases.go`。
- **注释**：Go、TS 代码、SQL 查询和接口描述用英文；迁移文件的中文注释和中文文档照本 plan 原样。
- **代码块**：每个改动都写成四个反引号围起来的块，块的第一行写明种类和路径，照原样使用（原型中逐字节运行过）：
  - ````` ````file <路径> ````` 新文件，块的内容加一个结尾换行就是整个文件；
  - ````` ````whole <路径> ````` 已有文件的完整新内容（同样加结尾换行）；
  - ````` ````old <路径> ````` 与紧跟着的 ````` ````new <路径> `````：`old` 的文字在文件当前版本中恰好出现一次，把它换成 `new` 的文字；
  - ````` ````delete <路径> ````` 删除这个文件（块是空的）。

  一个文件的几个块按出现的顺序依次应用。拼 plan 的脚本已从 `1d2eaf7c` 起按顺序核对过全部块：每个 `old` 恰好出现一次（在它之前的块应用之后的文件中），每个新文件原来不存在，逐 Task 应用之后的文件与原型逐字节相同（spec 附录 A）。可以用 `node <planapply.mjs> <本 plan> apply <仓库根> <n>` 写入第 n 个 Task 的块，也可以手工照抄。
- **过渡版本**：一些文件先在较早的 Task 写成过渡版本，较晚的 Task 再修改（`api/modules/workspace.yaml`、`workspace/module.go`、`app/invitation_ports.go`、`app/fakes_invitations_test.go`、`adapter/http/handler.go`、`adapter/http/invitations.go`、`adapter/postgres/invitations.go`、`queries/invitations.sql`、`domain/invitation.go`、`access/domain/rules.go`、`bootstrap/app.go`、矩阵的四个文件、生成物）；每个过渡版本都在逐 Task 复现中运行过。
- **变异**：每个 Task 末尾的"变异"表列出：把代码改坏的方式，和必须因此失败的测试。它们在原型上逐个跑过（`$M3TMP/p3tools/mutants.py`、`e2e_mutants.py`，spec 附录 A）；实现者可以照表抽查，改坏之后必须恢复。表中"（Task n 起）"标出的测试在较晚的 Task 才有。
- **评审敏感**（M3 设计 12 节约束 3）：令牌和从邀请到成员关系的路径是一个账户得到访问权的地方。改动令牌、接受、注册的测试之前，先照"变异"表确认它在所说的性质去掉之后失败。
- **提交**：提交信息用英文，末尾加一行：`Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>`
- 所有命令在仓库根目录下执行，除非步骤中另有说明。

## 文件结构

路径相对于仓库根目录。"生成"表示由命令生成并提交；一个文件由多个 Task 修改时，"Task"一列都列出。

| 文件 | 职责 | Task |
|---|---|---|
| `server/migrations/sql/00009_workspace_workspace_member_invites.sql` | 邀请的表（M3 设计 4.4） | 1 |
| `deploy/runtime-grants.sql`、`server/sqlc.yaml`（修改） | 运行时角色的权限；sqlc 读新迁移 | 1 |
| `server/migrations/schema_test.go`（修改） | 9 个迁移；名字和种类；CHECK 的反例；部分唯一键 | 1 |
| `server/internal/modules/workspace/adapter/postgres/gen/models.go`（生成） | | 1 |
| `server/internal/modules/workspace/domain/invitation.go` | `Invitation`、`InvitationPreview`、`InvitationWithToken`、`NewInvitation`、`CheckInvitations` 和两个下标的问题 | 1、5、6、8、9 |
| `server/internal/modules/workspace/domain/invitation_test.go` | 批量的检查 | 6 |
| `server/internal/modules/workspace/app/invitation_ports.go` | 邀请的端口：`InvitationMAC`、`CallerLock`、各用例的存储端口、`InvitationRow`、`DuplicateInvitation` | 1、5、6、8、9、11、12 |
| `server/internal/modules/workspace/app/ports.go`（修改） | `WorkspaceDeleter` 加上邀请的一步 | 1 |
| `server/internal/modules/workspace/adapter/postgres/queries/invitations.sql`、`server/internal/modules/workspace/adapter/postgres/invitations.go`、`server/internal/modules/workspace/adapter/postgres/invitations_test.go` | 邀请的查询和存储 | 1、5、7、9、10 |
| `server/internal/modules/workspace/adapter/postgres/gen/invitations.sql.go`（生成） | | 1、5、7、9、10 |
| `server/internal/modules/workspace/adapter/postgres/failures_test.go`（修改） | 读写失败时答错误 | 1、5 |
| `server/internal/modules/workspace/app/delete_workspace.go`、`server/internal/modules/workspace/app/delete_workspace_test.go`、`server/internal/modules/workspace/app/fakes_test.go`、`server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go`、`server/internal/bootstrap/workspace_deletion_test.go`（修改） | 删除工作区连带邀请 | 1（`fakes_test.go` 另有 8） |
| `server/internal/modules/identity/adapter/signing/keys.go`、`server/internal/modules/identity/adapter/signing/mac.go`、`server/internal/modules/identity/adapter/signing/signing_test.go`（修改） | 按用途派生的 MAC | 2 |
| `server/internal/modules/identity/keys.go`、`server/internal/modules/identity/keys_test.go` | `identity.LoadKeys`、`Keys.MAC` | 2 |
| `server/internal/modules/identity/module.go`、`server/internal/modules/identity/interleavings_test.go`（修改） | `Deps.Keys`；`SignupInvitation` | 2、4、12 |
| `server/internal/modules/workspace/domain/token.go`、`server/internal/modules/workspace/domain/token_test.go` | 令牌的消息、格式和解析 | 2 |
| `server/internal/bootstrap/app.go`（修改） | 载入密钥；邀请的 MAC、`CallerLock`、公开操作、注册策略的接线 | 2、5、6、9、12 |
| `server/internal/platform/httpserver/apitest/operations.go`、`server/internal/platform/httpserver/apitest/operations_test.go`（修改）；`server/internal/platform/httpserver/apitest/bodycases.go`、`server/internal/platform/httpserver/apitest/bodycases_test.go` | 参数的用例（必填的查询参数）；请求体的用例（对象的数组）移到自己的文件 | 3 |
| `server/internal/bootstrap/contract_test.go`（修改） | 必填的查询参数不带时 400 | 3 |
| `server/internal/modules/identity/provide.go`（修改）、`server/internal/modules/identity/provide_test.go` | `Provided.CredentialLock` | 4 |
| `api/modules/workspace.yaml`、`api/openapi.yaml`（修改） | 7 个操作和它们的结构 | 5、6、8、9、11（`openapi.yaml`：5、8、11） |
| `api/dist/openapi.yaml`、`web/packages/api-client/src/schema.gen.ts`（生成） | | 5、6、8、9、11、12 |
| `server/internal/modules/workspace/adapter/http/gen/server.gen.go`（生成） | | 5、6、8、9、11 |
| `server/internal/modules/workspace/adapter/http/gen/bodyshape.gen.go`（生成） | | 6、8、11 |
| `server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`（修改） | 规则表的 4 行和它们的格子 | 5、6、8 |
| `server/internal/modules/workspace/domain/actions.go`（修改） | 4 个操作名 | 5、6、8 |
| `server/internal/modules/workspace/app/tokens.go` | `invitationTokens`：算出、核对令牌 | 5、9 |
| `server/internal/modules/workspace/app/list_invitations.go`、`server/internal/modules/workspace/app/list_invitations_test.go` | `listWorkspaceInvitations` | 5（测试另有 6、11） |
| `server/internal/modules/workspace/app/fakes_invitations_test.go` | 邀请的假实现 | 5、6、8、9、11 |
| `server/internal/modules/workspace/adapter/http/handler.go`、`server/internal/modules/workspace/adapter/http/handler_test.go`（修改）；`server/internal/modules/workspace/adapter/http/invitations.go`、`server/internal/modules/workspace/adapter/http/invitations_test.go` | 用例的接口；邀请的 handler | 5、6、8、9、11 |
| `server/internal/modules/workspace/module.go`（修改） | 接上用例；`InvitationMACPurpose`；`PublicOperations`、`SignupInvitations` | 5、6、8、9、11、12 |
| `server/internal/bootstrap/invitation_mac_test.go` | 组合根要的用途是设计的 | 5 |
| `server/internal/bootstrap/permission_matrix_test.go`、`server/internal/bootstrap/permission_matrix_seeded_test.go`、`server/internal/bootstrap/permission_matrix_workspace_test.go`（修改）；`server/internal/bootstrap/permission_matrix_invitations_test.go` | 矩阵的邀请行、准备数据、公开操作的豁免和它的测试 | 5、6、8、9、11 |
| `server/internal/bootstrap/permission_matrix_coverage_test.go`（修改） | 未知的路径参数；公开操作的豁免 | 9、11 |
| `server/internal/modules/workspace/app/create_invitations.go`、`server/internal/modules/workspace/app/create_invitations_test.go` | `createWorkspaceInvitations` | 6 |
| `server/internal/modules/workspace/adapter/postgres/locks.go`、`server/internal/modules/workspace/adapter/postgres/locks_test.go`（修改）；`server/internal/modules/workspace/adapter/postgres/queries/workspaces.sql`（修改） | 按 id 的 `FOR SHARE`；`WorkspaceByID` | 7、10 |
| `server/internal/modules/workspace/adapter/postgres/gen/workspaces.sql.go`（生成） | | 7、10 |
| `server/internal/modules/workspace/adapter/postgres/update_invitation_test.go` | 邀请的读、锁、改角色、删除的存储测试 | 7 |
| `server/internal/modules/workspace/app/invitation_lock.go` | 按 id 寻址的邀请：读 → 锁工作区 → 锁邀请 | 8 |
| `server/internal/modules/workspace/app/update_invitation.go`、`server/internal/modules/workspace/app/update_invitation_test.go`、`server/internal/modules/workspace/app/delete_invitation.go`、`server/internal/modules/workspace/app/delete_invitation_test.go` | 修改、删除邀请 | 8 |
| `server/internal/modules/workspace/app/clock_test.go`（修改） | 写在锁之后读时钟 | 8、11 |
| `server/internal/modules/workspace/domain/errors.go`（修改） | 三个新码 | 8、11 |
| `web/apps/web/helpers/authentication.helper.ts`、`web/packages/i18n/src/locales/en/auth.json`、`web/packages/i18n/src/locales/zh-CN/auth.json`（修改） | 新码的文案 | 8、11 |
| `server/internal/modules/workspace/app/get_invitation.go`、`server/internal/modules/workspace/app/get_invitation_test.go` | 公开的查看 | 9 |
| `server/internal/bootstrap/invitations_test.go` | 访问日志；凭邀请注册 | 9、12 |
| `server/internal/modules/workspace/adapter/postgres/queries/members.sql`（修改）；`server/internal/modules/workspace/adapter/postgres/responses.go`、`server/internal/modules/workspace/adapter/postgres/responses_test.go` | 回应要的存储 | 10 |
| `server/internal/modules/workspace/adapter/postgres/gen/members.sql.go`（生成） | | 10 |
| `server/internal/modules/workspace/app/respond_invitation.go`、`server/internal/modules/workspace/app/accept_invitation.go`、`server/internal/modules/workspace/app/accept_invitation_test.go`、`server/internal/modules/workspace/app/decline_invitation.go`、`server/internal/modules/workspace/app/decline_invitation_test.go` | 接受、忽略 | 11 |
| `api/modules/identity.yaml`（修改） | `RegisterRequest.invitation` | 12 |
| `server/internal/modules/identity/adapter/http/gen/server.gen.go`、`server/internal/modules/identity/adapter/http/gen/bodyshape.gen.go`（生成） | | 12 |
| `server/internal/modules/identity/app/ports.go`、`server/internal/modules/identity/app/register.go`、`server/internal/modules/identity/app/register_test.go`、`server/internal/modules/identity/app/fakes_test.go`、`server/internal/modules/identity/adapter/http/auth.go`、`server/internal/modules/identity/adapter/http/handler_test.go`（修改） | 注册带着邀请问策略 | 12 |
| `server/internal/modules/workspace/app/signup_invitations.go`、`server/internal/modules/workspace/app/signup_invitations_test.go` | 注册的邀请检查 | 12 |
| `server/internal/bootstrap/signup_policy.go`、`server/internal/bootstrap/signup_policy_test.go` | 注册策略：开关加检查 | 12 |
| `server/internal/platform/postgres/pgtest/lockwait.go`、`server/internal/platform/postgres/pgtest/lockwait_test.go`（修改） | `WaitForKeyWaitOn` | 13 |
| `server/internal/bootstrap/interleaving_test.go`（修改）；`server/internal/bootstrap/interleaving_answers_test.go`、`server/internal/bootstrap/interleaving_invite_test.go` | 闸门的期限；交错 3、12、19 和 9、18 | 13 |
| `e2e/fixtures/api.ts`、`e2e/fixtures/assert/workspace.ts`、`e2e/stories/workspace/w8-navigation-preferences.spec.ts`（修改）；`e2e/stories/workspace/w3-workspace-settings.spec.ts`、`e2e/stories/workspace/w4-invite-members.spec.ts`、`e2e/stories/workspace/w5-invitation-link.spec.ts`、`e2e/stories/workspace/w6-sign-up-by-invitation.spec.ts` | W3–W6 的接口版本；W8 的第二个成员 | 14 |
| `docs/v0/v0-design.md`、`docs/v0/plane-diff.md`、`README.md`、`docs/v0/M3-workspace-project/handoffs/M2-closeout.md`（修改） | M3 设计 3.20、8.7 中 P3 的各行；交接的处理结果 | 15 |

---

### Task 1: 迁移 `00009`、邀请的插入与删除的连带

**Files:**
- Create: `server/internal/modules/workspace/adapter/postgres/invitations.go`、`server/internal/modules/workspace/adapter/postgres/invitations_test.go`、`server/internal/modules/workspace/adapter/postgres/queries/invitations.sql`、`server/internal/modules/workspace/app/invitation_ports.go`、`server/internal/modules/workspace/domain/invitation.go`、`server/migrations/sql/00009_workspace_workspace_member_invites.sql`
- Modify: `deploy/runtime-grants.sql`、`server/internal/bootstrap/workspace_deletion_test.go`、`server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go`、`server/internal/modules/workspace/adapter/postgres/failures_test.go`、`server/internal/modules/workspace/app/delete_workspace.go`、`server/internal/modules/workspace/app/delete_workspace_test.go`、`server/internal/modules/workspace/app/fakes_test.go`、`server/internal/modules/workspace/app/ports.go`、`server/migrations/schema_test.go`、`server/sqlc.yaml`
- Generate: `server/internal/modules/workspace/adapter/postgres/gen/invitations.sql.go`、`server/internal/modules/workspace/adapter/postgres/gen/models.go`

**Interfaces:**
- Produces（spec 2.3，M3 设计 4.4、4.12）：`domain.Invitation{ID, WorkspaceID, Email, Role, Accepted, RespondedAt, CreatedAt, CreatedByID}`、`(Invitation).Responded() bool`；`app.InvitationRow{ID, WorkspaceID, Email, Role, CreatedBy, Now}`；`*app.DuplicateInvitation{Email}`（唯一键拒绝的那一行）；存储 `CreateInvitations(ctx, rows) ([]domain.Invitation, error)`（一行一条语句，按给的顺序）、`DeleteWorkspaceInvitations(ctx, workspaceID, by, now) error`；`WorkspaceDeleter` 加 `DeleteWorkspaceInvitations`，`cascade()` 在工作区行之后加这一步。
- 使用者：Task 6（创建）；Task 13 的交错准备数据；P4 在 `cascade()` 的最后加项目。

**Tests:**
- `server/migrations/schema_test.go`：迁移数 9；`TestConstraintAndIndexNames` 加新表的 11 个名字和种类（`iuw`、`iw`、`f c`、`f n`）；`TestChecksRejectCounterexamples` 加 9 个反例（邮箱的大写 ASCII、非 ASCII、结尾制表符、中间 U+3000、空串；角色 10、0；接受而没有回应的时刻；去掉接受的时刻）；`TestUniqueKeysHoldAmongUndeletedRowsOnly` 加邀请：已忽略的邀请仍占着邮箱，删除之后空出来。
- `adapter/postgres/invitations_test.go`：`TestCreateInvitationsStoresTheRows`（按给的值存、按给的顺序答存下的行：待接受、时钟的时间、两列审计都是邀请人；两个工作区、两个邀请人）；`TestCreateInvitationsRefusesAnAddressTaken`（一个未删除的邀请占着的邮箱，待接受或已忽略：`*app.DuplicateInvitation` 指名它，之后的行不插入；在事务里，之前的行随之回滚；别的工作区的、已删除的邀请不占）。
- `adapter/postgres/delete_workspace_test.go`：`TestDeletingAWorkspaceSoftDeletesItsRows`、`TestAFailedDeletionLeavesTheWorkspace` 加上邀请（待接受和已忽略的一起删除，此前已删除的保持原来的时间，另一个工作区的不动）；`failures_test.go` 加两个写的失败。
- `app/delete_workspace_test.go`：`TestDeleteWorkspaceLocksThenDecidesThenCascades` 的调用记录在工作区行之后多一步 `DeleteWorkspaceInvitations`，同一个时刻、同一个删除者。
- `bootstrap/workspace_deletion_test.go`：两个工作区各准备一份邀请；`TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt` 从目录找到新表，不加豁免。

- [ ] **Step 1: 迁移和运行时权限**

`server/migrations/sql/00009_workspace_workspace_member_invites.sql`（新文件，32 行）：

````file server/migrations/sql/00009_workspace_workspace_member_invites.sql
-- workspace_member_invites：Plane 的 workspace_member_invites 表（13 列）按 M3 设计 4.4 保留 11 列。
-- 不存令牌（3.8）：链接里的令牌由签名密钥派生的 MAC 从邀请的 id 算出，删除 token 列；message 没有写入方，删除。
-- 接受时记 accepted、responded_at 并软删除这一行；忽略时只记 responded_at，这一行留着、仍占着这个邮箱（3.8）。

-- +goose Up
CREATE TABLE workspace_member_invites (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    -- 存规范化之后的邮箱（3.13），与 users.email 的 CHECK 相同，另外不能为空
    email varchar(255) NOT NULL CHECK (email <> '' AND email = lower(email) AND email !~ '[[:space:]]'),
    -- 访客 5、成员 15、管理员 20，同 workspace_members.role
    role smallint NOT NULL DEFAULT 5 CHECK (role IN (5, 15, 20)),
    accepted boolean NOT NULL DEFAULT false,
    responded_at timestamptz,
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    -- 接受的邀请一定有回应的时刻
    CONSTRAINT workspace_member_invites_responded_check CHECK (responded_at IS NOT NULL OR NOT accepted)
);
-- 一个工作区里一个邮箱至多一份未删除的邀请；已忽略的没有删除，仍占着这个邮箱（3.8）
CREATE UNIQUE INDEX workspace_member_invites_workspace_id_email_key ON workspace_member_invites (workspace_id, email)
    WHERE deleted_at IS NULL;
-- 注册策略、停用按邮箱查（4.4）
CREATE INDEX workspace_member_invites_email_idx ON workspace_member_invites (email) WHERE deleted_at IS NULL;
-- 物理级联（M4 的 60 天清理）按它找子行（4 节开头）
CREATE INDEX workspace_member_invites_workspace_id_idx ON workspace_member_invites (workspace_id);

-- +goose Down
DROP TABLE workspace_member_invites;
````

`server/sqlc.yaml`（修改，1 处）：

````old server/sqlc.yaml
      - migrations/sql/00008_workspace_workspace_user_properties.sql
````

````new server/sqlc.yaml
      - migrations/sql/00008_workspace_workspace_user_properties.sql
      - migrations/sql/00009_workspace_workspace_member_invites.sql
````

`deploy/runtime-grants.sql`（修改，1 处）：

````old deploy/runtime-grants.sql
GRANT SELECT, INSERT, UPDATE, DELETE ON workspaces, workspace_members, workspace_user_properties TO nerve_runtime;
````

````new deploy/runtime-grants.sql
GRANT SELECT, INSERT, UPDATE, DELETE ON workspaces, workspace_members, workspace_user_properties, workspace_member_invites
    TO nerve_runtime;
````

- [ ] **Step 2: 查询**

`server/internal/modules/workspace/adapter/postgres/queries/invitations.sql`（新文件，14 行）：

````file server/internal/modules/workspace/adapter/postgres/queries/invitations.sql
-- name: CreateInvitation :one
-- createWorkspaceInvitations inserts a batch one row a statement, in the order of the normalized addresses, under the
-- workspace's FOR SHARE (M3 design 3.6 convention 5). RETURNING gives the row as stored.
INSERT INTO workspace_member_invites (id, workspace_id, email, role, created_by_id, updated_by_id, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(email), sqlc.arg(role),
        sqlc.arg(created_by), sqlc.arg(created_by), sqlc.arg(now), sqlc.arg(now))
RETURNING id, workspace_id, email, role, accepted, responded_at, created_by_id, updated_by_id, created_at, updated_at, deleted_at;

-- name: DeleteWorkspaceInvitations :exec
-- deleteWorkspace's cascade: every undeleted invitation of the workspace, pending or declined, one statement in scan
-- order under the workspace's FOR NO KEY UPDATE (M3 design 3.6 convention 5). A row deleted before keeps its time.
UPDATE workspace_member_invites
SET deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE workspace_id = sqlc.arg(workspace_id) AND deleted_at IS NULL;
````

- [ ] **Step 3: 生成**

Run: `make gen-go`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `42047c679458e62cb2dedb228c31e34d04999ddc07bdd28edb5bdc2d080a21af` | 76 | `server/internal/modules/workspace/adapter/postgres/gen/invitations.sql.go` |
| `9785322a0ff91421b2351d2289857a0127bc38fe1f5620b58df40526c9024123` | 64 | `server/internal/modules/workspace/adapter/postgres/gen/models.go` |

Run: `shasum -a 256 server/internal/modules/workspace/adapter/postgres/gen/invitations.sql.go server/internal/modules/workspace/adapter/postgres/gen/models.go`
Expected: 与上表相同。

- [ ] **Step 4: 迁移的测试**

`server/migrations/schema_test.go`（修改，10 处）：

````old server/migrations/schema_test.go
	if err != nil || len(up) != 8 {
		t.Fatalf("Up() = %d migrations, %v; want 8", len(up), err)
````

````new server/migrations/schema_test.go
	if err != nil || len(up) != 9 {
		t.Fatalf("Up() = %d migrations, %v; want 9", len(up), err)
````

````old server/migrations/schema_test.go
			"workspace_members", "workspace_user_properties", "workspaces"}},
````

````new server/migrations/schema_test.go
			"workspace_member_invites", "workspace_members", "workspace_user_properties", "workspaces"}},
````

````old server/migrations/schema_test.go
	if again, err := m.Up(ctx); err != nil || len(again) != 8 {
		t.Errorf("Up() again = %d migrations, %v; want 8", len(again), err)
````

````new server/migrations/schema_test.go
	if again, err := m.Up(ctx); err != nil || len(again) != 9 {
		t.Errorf("Up() again = %d migrations, %v; want 9", len(again), err)
````

````old server/migrations/schema_test.go
		"users_pkey p",
````

````new server/migrations/schema_test.go
		"users_pkey p",
		"workspace_member_invites_created_by_id_fkey f n",
		"workspace_member_invites_email_check c",
		"workspace_member_invites_email_idx iw",
		"workspace_member_invites_pkey iu",
		"workspace_member_invites_pkey p",
		"workspace_member_invites_responded_check c",
		"workspace_member_invites_role_check c",
		"workspace_member_invites_updated_by_id_fkey f n",
		"workspace_member_invites_workspace_id_email_key iuw",
		"workspace_member_invites_workspace_id_fkey f c",
		"workspace_member_invites_workspace_id_idx i",
````

````old server/migrations/schema_test.go
// (M2 design 4.2, 4.3, 4.5; M3 design 4.2, 4.3, 4.5).
````

````new server/migrations/schema_test.go
// (M2 design 4.2, 4.3, 4.5; M3 design 4.2, 4.3, 4.4, 4.5).
````

````old server/migrations/schema_test.go
			"VALUES ('0199a2b4-0000-7000-8000-000000000007', '0199a2b4-0000-7000-8000-000000000005', " + user + ", 0, 'TABBED')",
````

````new server/migrations/schema_test.go
			"VALUES ('0199a2b4-0000-7000-8000-000000000007', '0199a2b4-0000-7000-8000-000000000005', " + user + ", 0, 'TABBED')",
		// An accepted invitation and a declined one, each with its response's
		// time, and a pending one.
		"INSERT INTO workspace_member_invites (id, workspace_id, email, role, accepted, responded_at) VALUES " +
			"('0199a2b4-0000-7000-8000-000000000008', '0199a2b4-0000-7000-8000-000000000005', 'élodie@exämple.com', 20, true, now()), " +
			"('0199a2b4-0000-7000-8000-000000000009', '0199a2b4-0000-7000-8000-000000000005', 'bob@corp.com', 5, false, now())",
		"INSERT INTO workspace_member_invites (id, workspace_id, email) VALUES " +
			"('0199a2b4-0000-7000-8000-00000000000a', '0199a2b4-0000-7000-8000-000000000005', 'carol@corp.com')",
````

````old server/migrations/schema_test.go
		{"lower-case navigation control", "UPDATE workspace_user_properties SET navigation_control_preference = 'tabbed'",
			"workspace_user_properties_navigation_control_preference_check"},
````

````new server/migrations/schema_test.go
		{"lower-case navigation control", "UPDATE workspace_user_properties SET navigation_control_preference = 'tabbed'",
			"workspace_user_properties_navigation_control_preference_check"},
		{"an invitation's upper-case ASCII e-mail", "UPDATE workspace_member_invites SET email = 'Carol@corp.com' WHERE email = 'carol@corp.com'",
			"workspace_member_invites_email_check"},
		{"an invitation's upper-case non-ASCII e-mail", "UPDATE workspace_member_invites SET email = 'Élodie@exämple.com' WHERE role = 20",
			"workspace_member_invites_email_check"},
		{"an invitation's e-mail with a trailing tab", `UPDATE workspace_member_invites SET email = E'carol@corp.com\t' WHERE email = 'carol@corp.com'`,
			"workspace_member_invites_email_check"},
		{"an invitation's e-mail with an inner U+3000", `UPDATE workspace_member_invites SET email = U&'carol\3000@corp.com' WHERE email = 'carol@corp.com'`,
			"workspace_member_invites_email_check"},
		{"an invitation's empty e-mail", "UPDATE workspace_member_invites SET email = '' WHERE email = 'carol@corp.com'",
			"workspace_member_invites_email_check"},
		{"an invitation's role 10", "UPDATE workspace_member_invites SET role = 10 WHERE email = 'carol@corp.com'", "workspace_member_invites_role_check"},
		{"an invitation's role 0", "UPDATE workspace_member_invites SET role = 0 WHERE email = 'carol@corp.com'", "workspace_member_invites_role_check"},
		{"accepted without a response", "UPDATE workspace_member_invites SET accepted = true WHERE email = 'carol@corp.com'",
			"workspace_member_invites_responded_check"},
		{"an acceptance's time removed", "UPDATE workspace_member_invites SET responded_at = NULL WHERE accepted",
			"workspace_member_invites_responded_check"},
````

````old server/migrations/schema_test.go
// (M3 design 3.10, 4.2, 4.3, 4.5).
````

````new server/migrations/schema_test.go
// (M3 design 3.10, 4.2, 4.3, 4.4, 4.5). A declined invitation is an
// undeleted row: it holds its address until it is deleted (M3 design 3.8).
````

````old server/migrations/schema_test.go
		"INSERT INTO workspace_user_properties (id, workspace_id, user_id) VALUES (gen_random_uuid(), " + beta + ", " + user + ")",
````

````new server/migrations/schema_test.go
		"INSERT INTO workspace_user_properties (id, workspace_id, user_id) VALUES (gen_random_uuid(), " + beta + ", " + user + ")",
		// carol's invitation to the workspace, declined; another address in
		// the workspace and carol in another workspace hold keys of their own.
		"INSERT INTO workspace_member_invites (id, workspace_id, email, responded_at) VALUES (gen_random_uuid(), " + workspace + ", 'carol@corp.com', now())",
		"INSERT INTO workspace_member_invites (id, workspace_id, email) VALUES (gen_random_uuid(), " + workspace + ", 'dave@corp.com')",
		"INSERT INTO workspace_member_invites (id, workspace_id, email) VALUES (gen_random_uuid(), " + beta + ", 'carol@corp.com')",
````

````old server/migrations/schema_test.go
			"workspace_user_properties_workspace_id_user_id_key",
````

````new server/migrations/schema_test.go
			"workspace_user_properties_workspace_id_user_id_key",
		},
		{
			"a declined invitation's address in a workspace",
			"INSERT INTO workspace_member_invites (id, workspace_id, email) VALUES (gen_random_uuid(), " + workspace + ", 'carol@corp.com')",
			"UPDATE workspace_member_invites SET deleted_at = now() WHERE workspace_id = " + workspace + " AND email = 'carol@corp.com'",
			"workspace_member_invites_workspace_id_email_key",
````

- [ ] **Step 5: 领域、端口、存储**

`server/internal/modules/workspace/domain/invitation.go`（新文件，29 行）：

````file server/internal/modules/workspace/domain/invitation.go
package domain

import (
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Invitation is an undeleted row of workspace_member_invites (M3 design
// 4.4, 5.2): an invitation of an address to a workspace, with a role,
// pending or declined. An accepted invitation is deleted as it is accepted,
// so the stores never answer one.
type Invitation struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	Email       string // normalized (M3 design 3.13)
	Role        shared.Role
	Accepted    bool
	RespondedAt *time.Time
	CreatedAt   time.Time
	CreatedByID *uuid.UUID
}

// Responded reports whether the invitation has been answered: declined,
// for an undeleted one (M3 design 3.8).
func (i Invitation) Responded() bool {
	return i.RespondedAt != nil
}
````

`server/internal/modules/workspace/app/invitation_ports.go`（新文件，32 行）：

````file server/internal/modules/workspace/app/invitation_ports.go
package app

import (
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The ports of the invitations' use cases (M3 design 3.8, 6.5).

// InvitationRow is an invitation to insert: a checked, normalized address
// and a role, its id, its inviter and the time of the use case's clock.
type InvitationRow struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	Email       string
	Role        shared.Role
	CreatedBy   uuid.UUID
	Now         time.Time
}

// DuplicateInvitation is a store's answer to an insert that the unique key
// of (workspace, address) refused: an undeleted invitation of the
// workspace, pending or declined, has Email (M3 design 3.8).
type DuplicateInvitation struct {
	Email string
}

func (e *DuplicateInvitation) Error() string {
	return "an undeleted invitation of the workspace has the address"
}
````

`server/internal/modules/workspace/adapter/postgres/invitations.go`（新文件，55 行）：

````file server/internal/modules/workspace/adapter/postgres/invitations.go
package postgresadapter

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// CreateInvitations inserts rows, one statement each, in the order given,
// and returns them as stored, in that order. The first row whose address an
// undeleted invitation of its workspace has is *app.DuplicateInvitation,
// naming that address; nothing after it is inserted, and the caller's
// transaction, aborted, rolls back the rows before it. The domain checked
// every value, so a CHECK violation is a bug: an internal error.
func (s *Store) CreateInvitations(ctx context.Context, rows []app.InvitationRow) ([]domain.Invitation, error) {
	q := s.queries(ctx)
	out := make([]domain.Invitation, 0, len(rows))
	for _, r := range rows {
		row, err := q.CreateInvitation(ctx, gen.CreateInvitationParams{
			ID: r.ID, WorkspaceID: r.WorkspaceID, Email: r.Email, Role: int16(r.Role), CreatedBy: &r.CreatedBy, Now: r.Now,
		})
		switch {
		case uniqueViolation(err, "workspace_member_invites_workspace_id_email_key"):
			return nil, &app.DuplicateInvitation{Email: r.Email}
		case err != nil:
			return nil, fmt.Errorf("create workspace invitation: %w", err)
		}
		out = append(out, invitation(row))
	}
	return out, nil
}

// DeleteWorkspaceInvitations soft-deletes the undeleted invitations of the
// workspace, pending or declined, by the account by at now.
func (s *Store) DeleteWorkspaceInvitations(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	err := s.queries(ctx).DeleteWorkspaceInvitations(ctx, gen.DeleteWorkspaceInvitationsParams{WorkspaceID: workspaceID, DeletedBy: by, Now: now})
	if err != nil {
		return fmt.Errorf("delete workspace invitations: %w", err)
	}
	return nil
}

// invitation is a stored row as the domain's value.
func invitation(r gen.WorkspaceMemberInvite) domain.Invitation {
	return domain.Invitation{
		ID: r.ID, WorkspaceID: r.WorkspaceID, Email: r.Email, Role: shared.Role(r.Role), Accepted: r.Accepted,
		RespondedAt: r.RespondedAt, CreatedAt: r.CreatedAt, CreatedByID: r.CreatedByID,
	}
}
````

`server/internal/modules/workspace/adapter/postgres/invitations_test.go`（新文件，131 行）：

````file server/internal/modules/workspace/adapter/postgres/invitations_test.go
package postgresadapter_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// invite stores a pending invitation of email to workspace with role, by
// inviter at now, and returns it as stored.
func invite(t *testing.T, s *postgresadapter.Store, workspace uuid.UUID, email string, role shared.Role, inviter uuid.UUID) domain.Invitation {
	t.Helper()
	got, err := s.CreateInvitations(context.Background(), []app.InvitationRow{
		{ID: uuid.NewV7(), WorkspaceID: workspace, Email: email, Role: role, CreatedBy: inviter, Now: now},
	})
	if err != nil {
		t.Fatal(err)
	}
	return got[0]
}

// pendingEmails are the addresses of workspace's undeleted invitations, sorted.
func pendingEmails(t *testing.T, pool *pgxpool.Pool, workspace uuid.UUID) []string {
	t.Helper()
	var emails []string
	if err := pool.QueryRow(context.Background(), `SELECT coalesce(array_agg(email ORDER BY email), '{}') FROM workspace_member_invites
		WHERE workspace_id = $1 AND deleted_at IS NULL`, workspace).Scan(&emails); err != nil {
		t.Fatal(err)
	}
	return emails
}

// CreateInvitations stores each row with the use case's values and answers
// them as stored, in the order given: pending, the time the clock's, the
// inviter in both audit columns. Two workspaces, two inviters.
func TestCreateInvitationsStoresTheRows(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme, beta := newWorkspace(t, s, "Acme", "acme", alice), newWorkspace(t, s, "Beta", "beta", bob)
	rows := []app.InvitationRow{
		{ID: uuid.NewV7(), WorkspaceID: acme.ID, Email: "erin@corp.com", Role: shared.RoleMember, CreatedBy: alice, Now: now},
		{ID: uuid.NewV7(), WorkspaceID: acme.ID, Email: "dave@corp.com", Role: shared.RoleGuest, CreatedBy: alice, Now: now},
		{ID: uuid.NewV7(), WorkspaceID: beta.ID, Email: "dave@corp.com", Role: shared.RoleAdmin, CreatedBy: bob, Now: now.Add(time.Hour)},
	}

	got, err := s.CreateInvitations(context.Background(), rows)

	if err != nil || len(got) != len(rows) {
		t.Fatalf("CreateInvitations() = %+v, %v; want the %d rows", got, err, len(rows))
	}
	for i, r := range rows {
		want := domain.Invitation{ID: r.ID, WorkspaceID: r.WorkspaceID, Email: r.Email, Role: r.Role, CreatedAt: r.Now, CreatedByID: &r.CreatedBy}
		if !sameInvitation(got[i], want) || got[i].CreatedAt.Location() != time.UTC {
			t.Errorf("row %d: %+v, want %+v", i, got[i], want)
		}
		var createdBy, updatedBy uuid.UUID
		var updated time.Time
		var responded, deleted *time.Time
		if err := pool.QueryRow(context.Background(), `SELECT created_by_id, updated_by_id, updated_at, responded_at, deleted_at
			FROM workspace_member_invites WHERE id = $1`, r.ID).Scan(&createdBy, &updatedBy, &updated, &responded, &deleted); err != nil {
			t.Fatal(err)
		}
		if createdBy != r.CreatedBy || updatedBy != r.CreatedBy || !updated.Equal(r.Now) || responded != nil || deleted != nil {
			t.Errorf("row %d: by %s, %s at %v, responded %v, deleted %v; want by the inviter at %v, pending", i, createdBy, updatedBy, updated,
				responded, deleted, r.Now)
		}
	}
}

// sameInvitation compares two invitations, the pointers by value.
func sameInvitation(a, b domain.Invitation) bool {
	sameTime := (a.RespondedAt == nil) == (b.RespondedAt == nil) && (a.RespondedAt == nil || a.RespondedAt.Equal(*b.RespondedAt))
	sameBy := (a.CreatedByID == nil) == (b.CreatedByID == nil) && (a.CreatedByID == nil || *a.CreatedByID == *b.CreatedByID)
	a.RespondedAt, b.RespondedAt, a.CreatedByID, b.CreatedByID = nil, nil, nil, nil
	return sameTime && sameBy && a.ID == b.ID && a.WorkspaceID == b.WorkspaceID && a.Email == b.Email && a.Role == b.Role &&
		a.Accepted == b.Accepted && a.CreatedAt.Equal(b.CreatedAt)
}

// An address with an undeleted invitation of the workspace, pending or
// declined, refuses the row that repeats it: *app.DuplicateInvitation
// naming it, and nothing after it is inserted; in a transaction, the rows
// before it roll back with it (M3 design 3.8). An address invited to
// another workspace, or whose invitation is deleted, is free.
func TestCreateInvitationsRefusesAnAddressTaken(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme, beta := newWorkspace(t, s, "Acme", "acme", alice), newWorkspace(t, s, "Beta", "beta", alice)
	invite(t, s, acme.ID, "carol@corp.com", shared.RoleMember, alice)
	declined := invite(t, s, acme.ID, "dave@corp.com", shared.RoleMember, alice)
	exec(t, pool, "UPDATE workspace_member_invites SET responded_at = $1 WHERE id = $2", now, declined.ID)
	deleted := invite(t, s, acme.ID, "erin@corp.com", shared.RoleMember, alice)
	exec(t, pool, "UPDATE workspace_member_invites SET deleted_at = $1 WHERE id = $2", now, deleted.ID)
	invite(t, s, beta.ID, "frank@corp.com", shared.RoleMember, alice)
	row := func(email string) app.InvitationRow {
		return app.InvitationRow{ID: uuid.NewV7(), WorkspaceID: acme.ID, Email: email, Role: shared.RoleGuest, CreatedBy: alice, Now: now}
	}
	tx := postgres.NewTxManager(pool, 2*time.Second)

	for _, taken := range []string{"carol@corp.com", "dave@corp.com"} {
		var got []domain.Invitation
		err := tx.WithinTx(context.Background(), func(ctx context.Context) error {
			var err error
			got, err = s.CreateInvitations(ctx, []app.InvitationRow{row("erin@corp.com"), row(taken), row("zoe@corp.com")})
			return err
		})
		var dup *app.DuplicateInvitation
		if !errors.As(err, &dup) || dup.Email != taken || got != nil {
			t.Errorf("a batch repeating %s: %+v, %v; want *app.DuplicateInvitation of it", taken, got, err)
		}
		if want := []string{"carol@corp.com", "dave@corp.com"}; !slices.Equal(pendingEmails(t, pool, acme.ID), want) {
			t.Errorf("after the batch repeating %s, acme's invitations are %q, want %q", taken, pendingEmails(t, pool, acme.ID), want)
		}
	}
	if _, err := s.CreateInvitations(context.Background(), []app.InvitationRow{row("erin@corp.com"), row("frank@corp.com")}); err != nil {
		t.Errorf("addresses free in acme: %v", err)
	}
	if want := []string{"carol@corp.com", "dave@corp.com", "erin@corp.com", "frank@corp.com"}; !slices.Equal(pendingEmails(t, pool, acme.ID), want) {
		t.Errorf("acme's invitations are %q, want %q", pendingEmails(t, pool, acme.ID), want)
	}
}
````

`server/internal/modules/workspace/adapter/postgres/failures_test.go`（修改，2 处）：

````old server/internal/modules/workspace/adapter/postgres/failures_test.go
		"DeleteWorkspace": s.DeleteWorkspace, "DeleteWorkspaceMembers": s.DeleteWorkspaceMembers,
		"DeleteWorkspacePreferences": s.DeleteWorkspacePreferences,
````

````new server/internal/modules/workspace/adapter/postgres/failures_test.go
		"DeleteWorkspace": s.DeleteWorkspace, "DeleteWorkspaceInvitations": s.DeleteWorkspaceInvitations,
		"DeleteWorkspaceMembers": s.DeleteWorkspaceMembers, "DeleteWorkspacePreferences": s.DeleteWorkspacePreferences,
````

````old server/internal/modules/workspace/adapter/postgres/failures_test.go
		t.Errorf("UpdateMemberRole() = %+v, %v; want context.Canceled", got, err)
	}
````

````new server/internal/modules/workspace/adapter/postgres/failures_test.go
		t.Errorf("UpdateMemberRole() = %+v, %v; want context.Canceled", got, err)
	}
	var dup *app.DuplicateInvitation
	if got, err := s.CreateInvitations(cancelled, []app.InvitationRow{
		{ID: uuid.NewV7(), WorkspaceID: w.ID, Email: "carol@corp.com", Role: shared.RoleGuest, CreatedBy: alice, Now: now},
	}); !failed(err) || errors.As(err, &dup) || got != nil {
		t.Errorf("CreateInvitations() = %+v, %v; want context.Canceled, not a duplicate", got, err)
	}
````

- [ ] **Step 6: 删除的连带**

`server/internal/modules/workspace/app/ports.go`（修改，1 处）：

````old server/internal/modules/workspace/app/ports.go
	DeleteWorkspace(ctx context.Context, id, by uuid.UUID, now time.Time) error
````

````new server/internal/modules/workspace/app/ports.go
	DeleteWorkspace(ctx context.Context, id, by uuid.UUID, now time.Time) error
	// DeleteWorkspaceInvitations soft-deletes its invitations, pending or
	// declined.
	DeleteWorkspaceInvitations(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error
````

`server/internal/modules/workspace/app/delete_workspace.go`（修改，2 处）：

````old server/internal/modules/workspace/app/delete_workspace.go
// statement at the same moment. Later phases add theirs here: P3 the
// invitations after the workspace row, P4 the projects at the end, through
````

````new server/internal/modules/workspace/app/delete_workspace.go
// statement at the same moment. P4 adds the projects at the end, through
````

````old server/internal/modules/workspace/app/delete_workspace.go
		u.workspaces.DeleteWorkspace,
````

````new server/internal/modules/workspace/app/delete_workspace.go
		u.workspaces.DeleteWorkspace,
		u.workspaces.DeleteWorkspaceInvitations,
````

`server/internal/modules/workspace/app/fakes_test.go`（修改，1 处）：

````old server/internal/modules/workspace/app/fakes_test.go
}

func (f *fakeWorkspaces) DeleteWorkspaceMembers(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error {
````

````new server/internal/modules/workspace/app/fakes_test.go
}

func (f *fakeWorkspaces) DeleteWorkspaceInvitations(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	return f.deleteStep(ctx, "DeleteWorkspaceInvitations", workspaceID, by, now)
}

func (f *fakeWorkspaces) DeleteWorkspaceMembers(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error {
````

`server/internal/modules/workspace/app/delete_workspace_test.go`（修改，4 处）：

````old server/internal/modules/workspace/app/delete_workspace_test.go
	for _, step := range []string{"DeleteWorkspace", "DeleteWorkspaceMembers", "DeleteWorkspacePreferences"} {
````

````new server/internal/modules/workspace/app/delete_workspace_test.go
	for _, step := range []string{"DeleteWorkspace", "DeleteWorkspaceInvitations", "DeleteWorkspaceMembers", "DeleteWorkspacePreferences"} {
````

````old server/internal/modules/workspace/app/delete_workspace_test.go
// soft-deletes the workspace, its members and their settings, by the caller
// at one moment, in one transaction (M3 design 3.6), and logs it. Two
// callers, two workspaces.
````

````new server/internal/modules/workspace/app/delete_workspace_test.go
// soft-deletes the workspace, its invitations, its members and their
// settings, by the caller at one moment, in one transaction (M3 design
// 3.6), and logs it. Two callers, two workspaces.
````

````old server/internal/modules/workspace/app/delete_workspace_test.go
			failure, append(slices.Clone(decided), steps[0])},
````

````new server/internal/modules/workspace/app/delete_workspace_test.go
			failure, append(slices.Clone(decided), steps[0])},
		{"the invitations failed", alice, "acme",
			func(f *deleteFixture) {
				f.workspaces.deleteErrs = map[string]error{"DeleteWorkspaceInvitations": failure}
			},
			failure, append(slices.Clone(decided), steps[:2]...)},
````

````old server/internal/modules/workspace/app/delete_workspace_test.go
			failure, append(slices.Clone(decided), steps[:2]...)},
		{"the settings failed", alice, "acme",
````

````new server/internal/modules/workspace/app/delete_workspace_test.go
			failure, append(slices.Clone(decided), steps[:3]...)},
		{"the settings failed", alice, "acme",
````

`server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go`（修改，12 处）：

````old server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
	"log/slog"
````

````new server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
	"log/slog"
	"slices"
````

````old server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
// The three steps, in one transaction, soft-delete the workspace, every
// membership of it, active or not, and every member's settings in it, at
// the same moment and by the same account; a membership and a settings row
// deleted before keep their time, and another workspace keeps everything.
// Running the steps again changes nothing, nobody's last_workspace_id is
// cleared, and the slug is free again.
````

````new server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
// The four steps, in one transaction, soft-delete the workspace, every
// invitation to it, pending or declined, every membership of it, active or
// not, and every member's settings in it, at the same moment and by the
// same account; an invitation, a membership and a settings row deleted
// before keep their time, and another workspace keeps everything. Running
// the steps again changes nothing, nobody's last_workspace_id is cleared,
// and the slug is free again.
````

````old server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
		upsert(t, s, r)
	}
	earlier := now.Add(-time.Hour)
````

````new server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
		upsert(t, s, r)
	}
	invite(t, s, acme.ID, "erin@corp.com", shared.RoleMember, alice)
	declined := invite(t, s, acme.ID, "frank@corp.com", shared.RoleGuest, alice)
	accepted := invite(t, s, acme.ID, "gina@corp.com", shared.RoleGuest, alice)
	invite(t, s, beta.ID, "erin@corp.com", shared.RoleMember, alice)
	earlier := now.Add(-time.Hour)
	exec(t, pool, "UPDATE workspace_member_invites SET responded_at = $1 WHERE id = $2", earlier, declined.ID)
	exec(t, pool, "UPDATE workspace_member_invites SET accepted = true, responded_at = $1, deleted_at = $1 WHERE id = $2", earlier, accepted.ID)
````

````old server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
		s.DeleteWorkspace, s.DeleteWorkspaceMembers, s.DeleteWorkspacePreferences,
````

````new server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
		s.DeleteWorkspace, s.DeleteWorkspaceInvitations, s.DeleteWorkspaceMembers, s.DeleteWorkspacePreferences,
````

````old server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
			WHERE workspace_id = $1 AND user_id <> $2`, acme.ID, carol),
````

````new server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
			WHERE workspace_id = $1 AND user_id <> $2`, acme.ID, carol),
		"invitations": deletions(t, pool, `SELECT id, deleted_at, updated_at, updated_by_id FROM workspace_member_invites
			WHERE workspace_id = $1 AND id <> $2`, acme.ID, accepted.ID),
````

````old server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
		want := map[string]int{"acme": 1, "members": 3, "settings": 2}[what]
````

````new server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
		want := map[string]int{"acme": 1, "members": 3, "settings": 2, "invitations": 2}[what]
````

````old server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
	var carolDeleted, daveDeleted time.Time
````

````new server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
	var carolDeleted, daveDeleted, ginaDeleted time.Time
````

````old server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
		(SELECT deleted_at FROM workspace_members WHERE member_id = $2)`, carol, dave).
		Scan(&carolDeleted, &daveDeleted); err != nil || !carolDeleted.Equal(earlier) || !daveDeleted.Equal(earlier) {
		t.Errorf("carol's settings deleted at %v, dave's membership at %v, %v; want %v, as before", carolDeleted, daveDeleted, err, earlier)
````

````new server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
		(SELECT deleted_at FROM workspace_members WHERE member_id = $2), (SELECT deleted_at FROM workspace_member_invites WHERE id = $3)`,
		carol, dave, accepted.ID).Scan(&carolDeleted, &daveDeleted, &ginaDeleted); err != nil || !carolDeleted.Equal(earlier) ||
		!daveDeleted.Equal(earlier) || !ginaDeleted.Equal(earlier) {
		t.Errorf("carol's settings deleted at %v, dave's membership at %v, gina's accepted invitation at %v, %v; want %v, as before",
			carolDeleted, daveDeleted, ginaDeleted, err, earlier)
	}
	var respondedAt time.Time
	if err := pool.QueryRow(context.Background(), "SELECT responded_at FROM workspace_member_invites WHERE id = $1", declined.ID).
		Scan(&respondedAt); err != nil || !respondedAt.Equal(earlier) {
		t.Errorf("frank's declined invitation responded at %v, %v; want %v, as before", respondedAt, err, earlier)
````

````old server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
		UNION ALL SELECT id, deleted_at, updated_at, updated_by_id FROM workspace_user_properties WHERE workspace_id = $1`, beta.ID)
	if len(betaRows) != 4 {
		t.Errorf("beta's rows: %d, want 4: the workspace, alice's and bob's memberships, alice's settings", len(betaRows))
````

````new server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
		UNION ALL SELECT id, deleted_at, updated_at, updated_by_id FROM workspace_user_properties WHERE workspace_id = $1
		UNION ALL SELECT id, deleted_at, updated_at, updated_by_id FROM workspace_member_invites WHERE workspace_id = $1`, beta.ID)
	if len(betaRows) != 5 {
		t.Errorf("beta's rows: %d, want 5: the workspace, alice's and bob's memberships, alice's settings, erin's invitation", len(betaRows))
````

````old server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
// step fails, the workspace and its members are not deleted either.
````

````new server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
// step fails, the workspace, its invitations and its members are not
// deleted either.
````

````old server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
	acme := newWorkspace(t, s, "Acme", "acme", alice)
	join(t, s, acme.ID, bob, shared.RoleMember)
````

````new server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
	acme := newWorkspace(t, s, "Acme", "acme", alice)
	join(t, s, acme.ID, bob, shared.RoleMember)
	invite(t, s, acme.ID, "carol@corp.com", shared.RoleGuest, alice)
````

````old server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
		t.Errorf("deleted memberships after the failure: %+v, want none", got)
	}
````

````new server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
		t.Errorf("deleted memberships after the failure: %+v, want none", got)
	}
	if got := pendingEmails(t, pool, acme.ID); !slices.Equal(got, []string{"carol@corp.com"}) {
		t.Errorf("acme's invitations after the failure: %q, want carol's", got)
	}
````

`server/internal/bootstrap/workspace_deletion_test.go`（修改，4 处）：

````old server/internal/bootstrap/workspace_deletion_test.go
// the cascade deletes it (P3 the invitations, P4 the projects' tables
// through ProjectCascade).
````

````new server/internal/bootstrap/workspace_deletion_test.go
// the cascade deletes it (P4 the projects' tables through ProjectCascade).
````

````old server/internal/bootstrap/workspace_deletion_test.go
// membership, which the creation writes; the member's, through the
// workspace store; the admin's display settings, through the API. A phase
// that adds a table under workspaces seeds a row of it here. It returns the
// workspace's id.
````

````new server/internal/bootstrap/workspace_deletion_test.go
// membership, which the creation writes; the member's, and an invitation
// the admin sent, through the workspace store; the admin's display
// settings, through the API. A phase that adds a table under workspaces
// seeds a row of it here. It returns the workspace's id.
````

````old server/internal/bootstrap/workspace_deletion_test.go
	var id, member uuid.UUID
	if err := pool.QueryRow(context.Background(), "SELECT (SELECT id FROM workspaces WHERE slug = $1), (SELECT id FROM users WHERE email = $2)",
		slug, "member@example.com").Scan(&id, &member); err != nil {
````

````new server/internal/bootstrap/workspace_deletion_test.go
	var id, member, admin uuid.UUID
	if err := pool.QueryRow(context.Background(), `SELECT (SELECT id FROM workspaces WHERE slug = $1), (SELECT id FROM users WHERE email = $2),
		(SELECT created_by_id FROM workspaces WHERE slug = $1)`, slug, "member@example.com").Scan(&id, &member, &admin); err != nil {
````

````old server/internal/bootstrap/workspace_deletion_test.go
	if err := workspacepg.New(pool).CreateMember(context.Background(), workspaceapp.MemberRow{
		ID: uuid.NewV7(), WorkspaceID: id, MemberID: member, Role: shared.RoleMember, CreatedBy: member, Now: time.Now(),
````

````new server/internal/bootstrap/workspace_deletion_test.go
	store := workspacepg.New(pool)
	if err := store.CreateMember(context.Background(), workspaceapp.MemberRow{
		ID: uuid.NewV7(), WorkspaceID: id, MemberID: member, Role: shared.RoleMember, CreatedBy: member, Now: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateInvitations(context.Background(), []workspaceapp.InvitationRow{
		{ID: uuid.NewV7(), WorkspaceID: id, Email: "invitee@example.com", Role: shared.RoleGuest, CreatedBy: admin, Now: time.Now()},
````

- [ ] **Step 7: 测试、lint**

Run: `go -C server test -count=1 ./migrations/`
Expected: `ok`。

Run: `go -C server test -count=1 ./internal/modules/workspace/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestTheGrantsFileCoversEveryRelationAndFunction|TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 8: 提交**

```bash
git add deploy/runtime-grants.sql server/internal/bootstrap/workspace_deletion_test.go server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go server/internal/modules/workspace/adapter/postgres/failures_test.go server/internal/modules/workspace/adapter/postgres/invitations.go server/internal/modules/workspace/adapter/postgres/invitations_test.go server/internal/modules/workspace/adapter/postgres/queries/invitations.sql server/internal/modules/workspace/app/delete_workspace.go server/internal/modules/workspace/app/delete_workspace_test.go server/internal/modules/workspace/app/fakes_test.go server/internal/modules/workspace/app/invitation_ports.go server/internal/modules/workspace/app/ports.go server/internal/modules/workspace/domain/invitation.go server/migrations/schema_test.go server/migrations/sql/00009_workspace_workspace_member_invites.sql server/sqlc.yaml server/internal/modules/workspace/adapter/postgres/gen/invitations.sql.go server/internal/modules/workspace/adapter/postgres/gen/models.go
```
```bash
git commit -m "feat(M3/P3): the workspace_member_invites table; deleting a workspace deletes its invitations

Migration 00009 keeps 11 of Plane's 13 columns, without the token (M3
design 3.8, 4.4): the address normalized, the role one of three, an
acceptance with its time, a partial unique key per workspace and address
that a declined invitation still holds. The store inserts a batch in the
order given and names the address the unique key refuses. deleteWorkspace's
cascade deletes the invitations after the workspace row, at its moment.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A）：

| 改坏 | 必须失败的测试 |
|---|---|
| 唯一索引去掉 `WHERE deleted_at IS NULL` | `TestConstraintAndIndexNames`、`TestUniqueKeysHoldAmongUndeletedRowsOnly` |
| 唯一索引只管未回应的（`WHERE deleted_at IS NULL AND responded_at IS NULL`） | `TestUniqueKeysHoldAmongUndeletedRowsOnly` |
| 去掉回应的 CHECK；角色的 CHECK 放宽为 `role >= 0`；邮箱的 CHECK 允许空白 | `TestConstraintAndIndexNames`（第一个）、`TestChecksRejectCounterexamples` |
| `workspace_id` 的外键去掉 `ON DELETE CASCADE` | `TestConstraintAndIndexNames` |
| 存储跳过被占的邮箱、插入其余的 | `TestCreateInvitationsRefusesAnAddressTaken`（Task 13 起另有 `TestInvitingOverlappingBatches`） |
| `cascade()` 少邀请一步 | `TestDeleteWorkspaceLocksThenDecidesThenCascades`、`TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt` |
| 连带的查询不看 `workspace_id`；不看 `deleted_at IS NULL` | `TestDeletingAWorkspaceSoftDeletesItsRows` |

**Done when:** 9 个迁移 up、down、再 up 都通过；11 个名字和种类、9 个反例、部分唯一键由测试核对；删除工作区在同一个事务、同一时刻连带邀请，组合的删除测试不加豁免而通过。

---

### Task 2: 签名密钥移到 `bootstrap`；按用途的 MAC；令牌的领域

**Files:**
- Create: `server/internal/modules/identity/keys.go`、`server/internal/modules/identity/keys_test.go`、`server/internal/modules/workspace/domain/token.go`、`server/internal/modules/workspace/domain/token_test.go`
- Modify: `server/internal/bootstrap/app.go`、`server/internal/modules/identity/adapter/signing/keys.go`、`server/internal/modules/identity/adapter/signing/signing_test.go`、`server/internal/modules/identity/interleavings_test.go`、`server/internal/modules/identity/module.go`
- Modify（完整内容）: `server/internal/modules/identity/adapter/signing/mac.go`

**Interfaces:**
- Produces（spec 2.4，M3 设计 3.8、6.6 第 1 步、8.1、11.1）：`identity.LoadKeys(pem []byte, logger *slog.Logger) (*identity.Keys, error)`；`(*identity.Keys).MAC(purpose string) (identity.MAC, error)`（`"refresh-token"` 是错误）；`identity.MAC{Tag(message []byte) [16]byte; Verify(message []byte, tag [16]byte) bool}`；`identity.Deps.Keys` 代替 `SigningKeyPEM`。`signing`：`PurposeRefreshToken`、`(*Keys).MAC(purpose) (*MAC, error)`（`HKDF-SHA256(seed, nil, "nerve "+purpose+" mac v1", 32)`）、`(*MAC).Tag`、`(*MAC).Verify`（`hmac.Equal`）；M2 的 `RefreshTokenMAC`、`NewRefreshTokenMAC` 由它代替（删除），`identity.New` 取 `d.Keys.signing.MAC(PurposeRefreshToken)`。`workspace/domain`：`InvitationMessage(id) []byte`、`FormatToken(tag [16]byte) string`、`ParseToken(token string) (tag [16]byte, ok bool)`。
- 使用者：Task 5（`InvitationMACPurpose`、`invitationTokens`）；Task 5、9、13 的测试用 `identity.LoadKeys` 算令牌。

**Tests:**
- `signing_test.go`（M2 的刷新令牌 MAC 的测试改为按用途）：`TestMAC`（同一消息同一标签；改任何一个字节、换密钥都换标签）、`TestMACVerify`（自己的标签通过；随机的、最后一位翻转的、全零的不通过；改消息的任何一个字节不通过）；`TestMACKnownAnswers`（刷新令牌 `65cc085217dccf9e602799b8e64a4468` 与 M2 的代码相同；邀请 `92fab228187e6c034c4fa240218ce894`，都由 spec 附录 A 的 `kat.py` 按 RFC 5869 手算）；`TestMACsOfPurposesDiffer`；`TestMACVerifyComparesInConstantTime`（解析 `mac.go`：`Verify` 的最后一句是 `return hmac.Equal(…)`）。
- `identity/keys_test.go`：`TestKeysMACIsThePurposes`（文件的密钥给已知答案；同一个文件相同，别的密钥不同）；`TestLoadKeysWithoutAFileWarns`（警告提到邀请链接）；`TestKeysMACRefusesTheRefreshTokensPurpose`。
- `workspace/domain/token_test.go`：`TestTheTokenOfTheKnownAnswer`（`nrv_inv_kvqyKBh-bANMT6JAIYzolA`）；`TestInvitationMessagesDifferByEveryByteOfTheID`；`TestATokenParsesBackToItsTag`（30 个字符，没有 `+`、`/`、`=`）；`TestEveryBitOfATokenCounts`（逐位翻转：不能解析，或解析成别的标签；最后一个字符的四个填充位）；`TestParseTokenRefuses`。

- [ ] **Step 1: 按用途的 MAC**

`server/internal/modules/identity/adapter/signing/mac.go`（完整内容，47 行）：

````whole server/internal/modules/identity/adapter/signing/mac.go
package signing

import (
	"crypto/hkdf"
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
)

// PurposeRefreshToken is the purpose of the refresh tokens' MAC (M2 design
// 3.4): its key's HKDF info is "nerve refresh-token mac v1".
const PurposeRefreshToken = "refresh-token"

// MAC tags messages for one purpose: the first 16 bytes of HMAC-SHA256 under
// K = HKDF-SHA256(the signing key's seed, info "nerve <purpose> mac v1")
// (M2 design 3.4, M3 design 3.8). A key derived for one purpose is no other
// purpose's and not the signing key, so a tag made for one purpose verifies
// for no other. It implements identity's app.RefreshTokenMAC and, through
// identity.Keys, the workspace module's InvitationMAC.
type MAC struct {
	key []byte
}

// MAC returns the MAC of purpose, its key derived from the signing key's
// seed.
func (k *Keys) MAC(purpose string) (*MAC, error) {
	key, err := hkdf.Key(sha256.New, k.private.Seed(), nil, "nerve "+purpose+" mac v1", 32)
	if err != nil {
		return nil, fmt.Errorf("derive the %s MAC key: %w", purpose, err)
	}
	return &MAC{key: key}, nil
}

// Tag returns the tag of message.
func (m *MAC) Tag(message []byte) [16]byte {
	h := hmac.New(sha256.New, m.key)
	h.Write(message)
	return [16]byte(h.Sum(nil)[:16])
}

// Verify reports whether tag is the tag of message. It compares in constant
// time: the time of a comparison would tell a forger how much of a tag is
// right (M2 design 3.9).
func (m *MAC) Verify(message []byte, tag [16]byte) bool {
	want := m.Tag(message)
	return hmac.Equal(want[:], tag[:])
}
````

`server/internal/modules/identity/adapter/signing/keys.go`（修改，7 处）：

````old server/internal/modules/identity/adapter/signing/keys.go
// the access tokens and, through a key derived from it, tags the refresh
// tokens (3.4). The key never leaves this package and is never logged.
````

````new server/internal/modules/identity/adapter/signing/keys.go
// the access tokens and, through a key derived from it for each purpose,
// tags the refresh tokens (M2 design 3.4) and the workspace invitations
// (M3 design 3.8). The key never leaves this package and is never logged.
````

````old server/internal/modules/identity/adapter/signing/keys.go
	"crypto/ed25519"
	"crypto/hkdf"
	"crypto/sha256"
````

````new server/internal/modules/identity/adapter/signing/keys.go
	"crypto/ed25519"
````

````old server/internal/modules/identity/adapter/signing/keys.go
// macInfo separates the MAC key from the signing key (HKDF info, M2 design 3.4).
const macInfo = "nerve refresh-token mac v1"

// Keys are the signing key and the MAC key derived from its seed.
````

````new server/internal/modules/identity/adapter/signing/keys.go
// Keys are the signing key and its public half. Each purpose's MAC key is
// derived from its seed (MAC).
````

````old server/internal/modules/identity/adapter/signing/keys.go
	public  ed25519.PublicKey
	mac     []byte
````

````new server/internal/modules/identity/adapter/signing/keys.go
	public  ed25519.PublicKey
````

````old server/internal/modules/identity/adapter/signing/keys.go
	return newKeys(private)
````

````new server/internal/modules/identity/adapter/signing/keys.go
	return newKeys(private), nil
````

````old server/internal/modules/identity/adapter/signing/keys.go
	k, _ := newKeys(private)                  // cannot fail for a generated key
	return k
````

````new server/internal/modules/identity/adapter/signing/keys.go
	return newKeys(private)
````

````old server/internal/modules/identity/adapter/signing/keys.go
func newKeys(private ed25519.PrivateKey) (*Keys, error) {
	mac, err := hkdf.Key(sha256.New, private.Seed(), nil, macInfo, 32)
	if err != nil {
		return nil, fmt.Errorf("derive the MAC key: %w", err)
	}
	return &Keys{private: private, public: private.Public().(ed25519.PublicKey), mac: mac}, nil
````

````new server/internal/modules/identity/adapter/signing/keys.go
func newKeys(private ed25519.PrivateKey) *Keys {
	return &Keys{private: private, public: private.Public().(ed25519.PublicKey)}
````

`server/internal/modules/identity/adapter/signing/signing_test.go`（修改，11 处）：

````old server/internal/modules/identity/adapter/signing/signing_test.go
	"encoding/base64"
````

````new server/internal/modules/identity/adapter/signing/signing_test.go
	"encoding/base64"
	"encoding/hex"
````

````old server/internal/modules/identity/adapter/signing/signing_test.go
	"errors"
````

````new server/internal/modules/identity/adapter/signing/signing_test.go
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
````

````old server/internal/modules/identity/adapter/signing/signing_test.go
}

func TestParseKeysReadsAnOpenSSLKey(t *testing.T) {
````

````new server/internal/modules/identity/adapter/signing/signing_test.go
}

// testMAC is keys' MAC of purpose.
func testMAC(t *testing.T, keys *Keys, purpose string) *MAC {
	t.Helper()
	m, err := keys.MAC(purpose)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestParseKeysReadsAnOpenSSLKey(t *testing.T) {
````

````old server/internal/modules/identity/adapter/signing/signing_test.go
	if len(k.private) != 64 || len(k.public) != 32 || len(k.mac) != 32 {
		t.Errorf("key sizes = %d, %d, %d; want 64, 32, 32", len(k.private), len(k.public), len(k.mac))
````

````new server/internal/modules/identity/adapter/signing/signing_test.go
	if len(k.private) != 64 || len(k.public) != 32 {
		t.Errorf("key sizes = %d, %d; want 64, 32", len(k.private), len(k.public))
````

````old server/internal/modules/identity/adapter/signing/signing_test.go
	if bytes.Equal(k.mac, k.private.Seed()) {
		t.Error("the MAC key equals the seed, want a derived key")
````

````new server/internal/modules/identity/adapter/signing/signing_test.go
	if m := testMAC(t, k, PurposeRefreshToken); len(m.key) != 32 || bytes.Equal(m.key, k.private.Seed()) {
		t.Errorf("the MAC key is %d bytes, equal to the seed %v; want 32 bytes derived from it", len(m.key), bytes.Equal(m.key, k.private.Seed()))
````

````old server/internal/modules/identity/adapter/signing/signing_test.go
	if bytes.Equal(a.private, b.private) || bytes.Equal(a.mac, b.mac) {
````

````new server/internal/modules/identity/adapter/signing/signing_test.go
	if bytes.Equal(a.private, b.private) || bytes.Equal(testMAC(t, a, PurposeRefreshToken).key, testMAC(t, b, PurposeRefreshToken).key) {
````

````old server/internal/modules/identity/adapter/signing/signing_test.go
func TestRefreshTokenMAC(t *testing.T) {
````

````new server/internal/modules/identity/adapter/signing/signing_test.go
func TestMAC(t *testing.T) {
````

````old server/internal/modules/identity/adapter/signing/signing_test.go
	m := NewRefreshTokenMAC(keys)
````

````new server/internal/modules/identity/adapter/signing/signing_test.go
	m := testMAC(t, keys, PurposeRefreshToken)
````

````old server/internal/modules/identity/adapter/signing/signing_test.go
	if NewRefreshTokenMAC(EphemeralKeys()).Tag(msg) == tag {
````

````new server/internal/modules/identity/adapter/signing/signing_test.go
	if testMAC(t, EphemeralKeys(), PurposeRefreshToken).Tag(msg) == tag {
````

````old server/internal/modules/identity/adapter/signing/signing_test.go
func TestRefreshTokenMACVerify(t *testing.T) {
	m := NewRefreshTokenMAC(testKeys(t))
````

````new server/internal/modules/identity/adapter/signing/signing_test.go
func TestMACVerify(t *testing.T) {
	m := testMAC(t, testKeys(t), PurposeRefreshToken)
````

````old server/internal/modules/identity/adapter/signing/signing_test.go
	if NewRefreshTokenMAC(EphemeralKeys()).Verify(msg, tag) {
		t.Error("Verify() under another key = true")
	}
}

````

````new server/internal/modules/identity/adapter/signing/signing_test.go
	if testMAC(t, EphemeralKeys(), PurposeRefreshToken).Verify(msg, tag) {
		t.Error("Verify() under another key = true")
	}
}

// The MACs' keys are HKDF-SHA256 of testKeyPEM's seed with the info of M2
// design 3.4 and M3 design 3.8, and a tag the first 16 bytes of
// HMAC-SHA256: the answers below were computed from RFC 5869 by hand, not
// by this package (spec P3, appendix). The refresh-token tag is the one the
// code before M3 gave, so the refresh tokens a deployment issued stay
// valid; the invitation's is the tag of "workspace-invitation" and the id
// 0199a2b4-0000-7000-8000-000000000001.
func TestMACKnownAnswers(t *testing.T) {
	keys := testKeys(t)
	id := [16]byte{0x01, 0x99, 0xa2, 0xb4, 0, 0, 0x70, 0, 0x80, 0, 0, 0, 0, 0, 0, 0x01}
	for _, tt := range []struct {
		purpose string
		message []byte
		want    string
	}{
		{PurposeRefreshToken, bytes.Repeat([]byte{7}, 52), "65cc085217dccf9e602799b8e64a4468"},
		{"workspace-invitation", append([]byte("workspace-invitation"), id[:]...), "92fab228187e6c034c4fa240218ce894"},
	} {
		if tag := testMAC(t, keys, tt.purpose).Tag(tt.message); hex.EncodeToString(tag[:]) != tt.want {
			t.Errorf("the %s tag = %x, want %s", tt.purpose, tag, tt.want)
		}
	}
}

// A tag made for one purpose is no tag of another: the keys differ.
func TestMACsOfPurposesDiffer(t *testing.T) {
	keys := testKeys(t)
	refresh, invitation := testMAC(t, keys, PurposeRefreshToken), testMAC(t, keys, "workspace-invitation")
	msg := bytes.Repeat([]byte{7}, 36)
	if bytes.Equal(refresh.key, invitation.key) || refresh.Tag(msg) == invitation.Tag(msg) {
		t.Error("the refresh-token and workspace-invitation MACs are the same")
	}
	if invitation.Verify(msg, refresh.Tag(msg)) || refresh.Verify(msg, invitation.Tag(msg)) {
		t.Error("a tag of one purpose verifies for the other")
	}
}

// Verify answers hmac.Equal of the two tags and nothing else: a comparison
// that stops at the first difference tells a forger, by its time, how much
// of a tag is right (M2 design 3.9, M3 design 8.1). No test can time it, so
// this one reads the code.
func TestMACVerifyComparesInConstantTime(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "mac.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var verify *ast.FuncDecl
	for _, d := range file.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok && fn.Name.Name == "Verify" && fn.Recv != nil {
			verify = fn
		}
	}
	if verify == nil {
		t.Fatal("mac.go has no method Verify")
	}
	last, _ := verify.Body.List[len(verify.Body.List)-1].(*ast.ReturnStmt)
	if last == nil || len(last.Results) != 1 || !isCallOf(last.Results[0], "hmac", "Equal") {
		t.Error("Verify does not end in return hmac.Equal(…)")
	}
}

// isCallOf reports whether e calls pkg.name.
func isCallOf(e ast.Expr, pkg, name string) bool {
	call, ok := e.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	x, ok := sel.X.(*ast.Ident)
	return ok && x.Name == pkg && sel.Sel.Name == name
}

````

- [ ] **Step 2: `identity.Keys`**

`server/internal/modules/identity/keys.go`（新文件，58 行）：

````file server/internal/modules/identity/keys.go
package identity

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/signing"
)

// Keys are the instance's signing key, loaded once, before any module is
// built (M3 design 6.6 step 1). They never give the key out: identity.New
// signs the access tokens and tags the refresh tokens with it, and MAC
// gives another module a MAC of its own purpose.
type Keys struct {
	signing *signing.Keys
}

// LoadKeys loads the signing key from pemData, the content of
// auth.jwt.private_key_file. nil is no file: the key is then ephemeral, for
// dev and test only (M2 design 3.7), and logger warns. A key that cannot be
// parsed is an error that never quotes the key.
func LoadKeys(pemData []byte, logger *slog.Logger) (*Keys, error) {
	if pemData == nil {
		logger.Warn("auth.jwt.private_key_file is not set: signing with an ephemeral key; " +
			"access tokens and invitation links stop verifying at restart (dev and test only)")
		return &Keys{signing: signing.EphemeralKeys()}, nil
	}
	keys, err := signing.ParseKeys(pemData)
	if err != nil {
		return nil, fmt.Errorf("auth.jwt.private_key_file: %w", err)
	}
	return &Keys{signing: keys}, nil
}

// MAC authenticates the messages of one purpose: Tag gives the first 16
// bytes of HMAC-SHA256 under the purpose's key, and Verify compares a tag
// in constant time.
type MAC interface {
	Tag(message []byte) [16]byte
	Verify(message []byte, tag [16]byte) bool
}

// MAC returns the MAC of purpose, whose key is derived from the signing key
// for that purpose alone, with HKDF info "nerve <purpose> mac v1" (M3
// design 3.8, 11.1): a tag of one purpose verifies for no other. purpose
// names another module's use, e.g. "workspace-invitation"; the refresh
// tokens' purpose is identity's own and no other module's.
func (k *Keys) MAC(purpose string) (MAC, error) {
	if purpose == signing.PurposeRefreshToken {
		return nil, errors.New("the refresh-token MAC is identity's own")
	}
	m, err := k.signing.MAC(purpose)
	if err != nil {
		return nil, err
	}
	return m, nil
}
````

`server/internal/modules/identity/keys_test.go`（新文件，76 行）：

````file server/internal/modules/identity/keys_test.go
package identity_test

import (
	"bytes"
	"encoding/hex"
	"log/slog"
	"strings"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
)

// keyPEM is what `openssl genpkey -algorithm ed25519` writes: the key of
// the signing package's tests, for tests only.
const keyPEM = `-----BEGIN PRIVATE KEY-----
MC4CAQAwBQYDK2VwBCIEIGqen6oN2FFQjS+yPQPHLVBIW0B2O9faCmNwftOWxqyE
-----END PRIVATE KEY-----
`

// invitationMessage is "workspace-invitation" and the invitation id
// 0199a2b4-0000-7000-8000-000000000001, the message of the signing
// package's known answer.
var invitationMessage = append([]byte("workspace-invitation"), 0x01, 0x99, 0xa2, 0xb4, 0, 0, 0x70, 0, 0x80, 0, 0, 0, 0, 0, 0, 0x01)

// A purpose's MAC is the one derived for that purpose from the key loaded:
// the known answer of the signing package's test for the key of the file,
// the same for the same file, another for another key.
func TestKeysMACIsThePurposes(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	tagOf := func(pem []byte) [16]byte {
		t.Helper()
		keys, err := identity.LoadKeys(pem, logger)
		if err != nil {
			t.Fatal(err)
		}
		mac, err := keys.MAC("workspace-invitation")
		if err != nil {
			t.Fatal(err)
		}
		return mac.Tag(invitationMessage)
	}
	tag := tagOf([]byte(keyPEM))
	if hex.EncodeToString(tag[:]) != "92fab228187e6c034c4fa240218ce894" {
		t.Errorf("the workspace-invitation tag = %x, want the known answer", tag)
	}
	if tagOf([]byte(keyPEM)) != tag {
		t.Error("the same key file gives another tag")
	}
	if ephemeral := tagOf(nil); ephemeral == tag || tagOf(nil) == ephemeral {
		t.Error("an ephemeral key gives the file's tag, or two ephemeral keys the same")
	}
}

// Without a file the key is ephemeral, and LoadKeys warns that links stop
// verifying at restart (M3 design 3.8).
func TestLoadKeysWithoutAFileWarns(t *testing.T) {
	var logs bytes.Buffer
	if _, err := identity.LoadKeys(nil, slog.New(slog.NewTextHandler(&logs, nil))); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(logs.String(), "access tokens and invitation links stop verifying at restart") {
		t.Errorf("logs: %s; want the ephemeral key's warning", logs.String())
	}
}

// The refresh tokens' MAC is identity's own: no other module is given its
// key, so none can make a refresh token's tag.
func TestKeysMACRefusesTheRefreshTokensPurpose(t *testing.T) {
	keys, err := identity.LoadKeys([]byte(keyPEM), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	if mac, err := keys.MAC("refresh-token"); err == nil || mac != nil {
		t.Errorf("MAC(refresh-token) = %v, %v; want an error", mac, err)
	}
}
````

`server/internal/modules/identity/module.go`（修改，6 处）：

````old server/internal/modules/identity/module.go
	// SigningKeyPEM is the content of auth.jwt.private_key_file; nil for
	// none, then the key is ephemeral (dev and test only, M2 design 3.7).
	SigningKeyPEM   []byte
````

````new server/internal/modules/identity/module.go
	// Keys are the signing key, as LoadKeys loaded it.
	Keys            *Keys
````

````old server/internal/modules/identity/module.go
// New wires the module. A signing key that cannot be parsed is an error
// that never quotes the key.
````

````new server/internal/modules/identity/module.go
// New wires the module.
````

````old server/internal/modules/identity/module.go
	keys, err := signingKeys(d)
````

````new server/internal/modules/identity/module.go
	refreshMAC, err := d.Keys.signing.MAC(signing.PurposeRefreshToken)
````

````old server/internal/modules/identity/module.go
	tokens := signing.NewAccessTokens(keys)
````

````new server/internal/modules/identity/module.go
	tokens := signing.NewAccessTokens(d.Keys.signing)
````

````old server/internal/modules/identity/module.go
		MAC:        signing.NewRefreshTokenMAC(keys),
````

````new server/internal/modules/identity/module.go
		MAC:        refreshMAC,
````

````old server/internal/modules/identity/module.go

func signingKeys(d Deps) (*signing.Keys, error) {
	if d.SigningKeyPEM == nil {
		d.Logger.Warn("auth.jwt.private_key_file is not set: signing with an ephemeral key; " +
			"access tokens stop verifying at restart (dev and test only)")
		return signing.EphemeralKeys(), nil
	}
	keys, err := signing.ParseKeys(d.SigningKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("auth.jwt.private_key_file: %w", err)
	}
	return keys, nil
}

````

````new server/internal/modules/identity/module.go

````

`server/internal/modules/identity/interleavings_test.go`（修改，2 处）：

````old server/internal/modules/identity/interleavings_test.go
	keys := signing.EphemeralKeys()
````

````new server/internal/modules/identity/interleavings_test.go
	keys := signing.EphemeralKeys()
	mac, err := keys.MAC(signing.PurposeRefreshToken)
	if err != nil {
		panic(err) // 32 bytes of HKDF-SHA256 never fail
	}
````

````old server/internal/modules/identity/interleavings_test.go
		Issuance: app.Issuance{Tokens: signing.NewAccessTokens(keys), MAC: signing.NewRefreshTokenMAC(keys), AccessTTL: time.Minute, SessionTTL: time.Hour},
````

````new server/internal/modules/identity/interleavings_test.go
		Issuance: app.Issuance{Tokens: signing.NewAccessTokens(keys), MAC: mac, AccessTTL: time.Minute, SessionTTL: time.Hour},
````

- [ ] **Step 3: `bootstrap` 先载入密钥**

`server/internal/bootstrap/app.go`（修改，2 处）：

````old server/internal/bootstrap/app.go
		return nil, err
	}
	pool, err := postgres.NewPool(ctx, cfg.Database)
````

````new server/internal/bootstrap/app.go
		return nil, err
	}
	// The signing key first, before any module (M3 design 6.6 step 1).
	keys, err := identity.LoadKeys(signingKey, logger)
	if err != nil {
		return nil, err
	}
	pool, err := postgres.NewPool(ctx, cfg.Database)
````

````old server/internal/bootstrap/app.go
		SigningKeyPEM:   signingKey,
````

````new server/internal/bootstrap/app.go
		Keys:            keys,
````

- [ ] **Step 4: 令牌的领域**

`server/internal/modules/workspace/domain/token.go`（新文件，46 行）：

````file server/internal/modules/workspace/domain/token.go
package domain

import (
	"encoding/base64"
	"strings"
	"uuid"
)

// An invitation's token (M3 design 3.8, 8.1) is "nrv_inv_" and 22 base64url
// characters without padding: the 16-byte tag of the invitation MAC on
// InvitationMessage. The server never stores it; it computes it again from
// the invitation's id.
const (
	tokenPrefix = "nrv_inv_"
	// tokenLabel begins every message the invitation MAC tags.
	tokenLabel = "workspace-invitation"
)

// tokenEncoding is base64url without padding, strict: a last character with
// any of its four low bits set is refused, so that one tag has one token.
var tokenEncoding = base64.RawURLEncoding.Strict()

// InvitationMessage is the message an invitation's token tags:
// "workspace-invitation", then the id's 16 bytes.
func InvitationMessage(id uuid.UUID) []byte {
	return append([]byte(tokenLabel), id[:]...)
}

// FormatToken is the token that carries tag.
func FormatToken(tag [16]byte) string {
	return tokenPrefix + tokenEncoding.EncodeToString(tag[:])
}

// ParseToken returns the tag token carries. ok is false for anything but
// "nrv_inv_" and the 22 characters FormatToken writes for some tag.
func ParseToken(token string) (tag [16]byte, ok bool) {
	encoded, found := strings.CutPrefix(token, tokenPrefix)
	if !found || len(encoded) != 22 {
		return tag, false
	}
	b, err := tokenEncoding.DecodeString(encoded)
	if err != nil || len(b) != len(tag) {
		return tag, false
	}
	return [16]byte(b), true
}
````

`server/internal/modules/workspace/domain/token_test.go`（新文件，116 行）：

````file server/internal/modules/workspace/domain/token_test.go
package domain_test

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
)

// The message is the label and the id's bytes, and a token the prefix and
// the tag in base64url: the known answer of the signing package's test
// (spec P3, appendix), computed by hand from RFC 5869 and RFC 4648.
func TestTheTokenOfTheKnownAnswer(t *testing.T) {
	id := uuid.MustParse("0199a2b4-0000-7000-8000-000000000001")
	want := append([]byte("workspace-invitation"), 0x01, 0x99, 0xa2, 0xb4, 0, 0, 0x70, 0, 0x80, 0, 0, 0, 0, 0, 0, 0x01)
	if got := domain.InvitationMessage(id); !bytes.Equal(got, want) {
		t.Errorf("InvitationMessage() = %q, want %q", got, want)
	}
	tag, err := hex.DecodeString("92fab228187e6c034c4fa240218ce894")
	if err != nil {
		t.Fatal(err)
	}
	if got := domain.FormatToken([16]byte(tag)); got != "nrv_inv_kvqyKBh-bANMT6JAIYzolA" {
		t.Errorf("FormatToken() = %q, want nrv_inv_kvqyKBh-bANMT6JAIYzolA", got)
	}
}

// Two ids, two messages: the message holds the whole id.
func TestInvitationMessagesDifferByEveryByteOfTheID(t *testing.T) {
	id := uuid.NewV7()
	for i := range id {
		other := id
		other[i] ^= 0x80
		if bytes.Equal(domain.InvitationMessage(id), domain.InvitationMessage(other)) {
			t.Errorf("byte %d of the id is not in the message", i)
		}
	}
}

// randomTag is 16 random bytes.
func randomTag(t *testing.T) [16]byte {
	t.Helper()
	var tag [16]byte
	if _, err := rand.Read(tag[:]); err != nil {
		t.Fatal(err)
	}
	return tag
}

// A token is 30 characters, parses back to its tag, and is written in
// base64url: no "+", "/" or "=".
func TestATokenParsesBackToItsTag(t *testing.T) {
	for range 1000 {
		tag := randomTag(t)
		token := domain.FormatToken(tag)
		if got, ok := domain.ParseToken(token); !ok || got != tag || len(token) != 30 || strings.ContainsAny(token, "+/=") {
			t.Fatalf("FormatToken(%x) = %q, which parses to %x, %v", tag, token, got, ok)
		}
	}
}

// Changing any one bit of a token gives no token of the same tag: it does
// not parse, or it parses to another tag, which the MAC then refuses. The
// last character carries four bits of padding: a decoder that ignored them
// would take four other characters for it (M3 design 8.1).
func TestEveryBitOfATokenCounts(t *testing.T) {
	for range 64 {
		tag := randomTag(t)
		token := domain.FormatToken(tag)
		for i := range len(token) {
			for bit := range 8 {
				changed := []byte(token)
				changed[i] ^= 1 << bit
				if got, ok := domain.ParseToken(string(changed)); ok && got == tag {
					t.Fatalf("%q with bit %d of character %d changed, %q, parses to the same tag", token, bit, i, changed)
				}
			}
		}
	}
}

// Anything but "nrv_inv_" and 22 base64url characters of 16 bytes is no
// token.
func TestParseTokenRefuses(t *testing.T) {
	valid := domain.FormatToken(randomTag(t))
	encoded := strings.TrimPrefix(valid, "nrv_inv_")
	for name, token := range map[string]string{
		"empty":                       "",
		"the prefix alone":            "nrv_inv_",
		"without the prefix":          encoded,
		"another prefix":              "nrv_pat_" + encoded,
		"an upper-case prefix":        "NRV_INV_" + encoded,
		"one character short":         valid[:len(valid)-1],
		"one character more":          valid + "A",
		"padded":                      valid[:len(valid)-2] + "==",
		"standard base64's +":         valid[:12] + "+" + valid[13:],
		"standard base64's /":         valid[:12] + "/" + valid[13:],
		"a new line inside":           valid[:12] + "\n" + valid[13:],
		"a space inside":              valid[:12] + " " + valid[13:],
		"padding bits set":            "nrv_inv_AAAAAAAAAAAAAAAAAAAAAB",
		"a space before":              " " + valid,
		"fifteen bytes":               "nrv_inv_" + base64.RawURLEncoding.EncodeToString(make([]byte, 15)),
		"seventeen bytes":             "nrv_inv_" + base64.RawURLEncoding.EncodeToString(make([]byte, 17)),
		"the token twice":             valid + valid,
		"a multi-byte character last": valid[:len(valid)-1] + "é",
	} {
		if tag, ok := domain.ParseToken(token); ok {
			t.Errorf("%s: ParseToken(%q) = %x, want no token", name, token, tag)
		}
	}
}
````

- [ ] **Step 5: 测试、lint**

Run: `go -C server test -count=1 ./internal/modules/identity/... ./internal/modules/workspace/domain/`
Expected: 全部 `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 6: 提交**

```bash
git add server/internal/bootstrap/app.go server/internal/modules/identity/adapter/signing/keys.go server/internal/modules/identity/adapter/signing/mac.go server/internal/modules/identity/adapter/signing/signing_test.go server/internal/modules/identity/interleavings_test.go server/internal/modules/identity/keys.go server/internal/modules/identity/keys_test.go server/internal/modules/identity/module.go server/internal/modules/workspace/domain/token.go server/internal/modules/workspace/domain/token_test.go
```
```bash
git commit -m "feat(M3/P3): the signing key loads in bootstrap and derives a MAC per purpose; the invitation token

bootstrap loads the signing key before any module (M3 design 6.6 step 1).
identity.Keys gives another module a MAC of its own purpose, its key
HKDF-derived from the seed with the purpose in the info, as the refresh
tokens' MAC has always been (their tag is unchanged); the refresh-token
purpose stays identity's own. An invitation's token is nrv_inv_ and the
base64url of the first 16 bytes of the invitation MAC of
\"workspace-invitation\" and the id, strictly decoded: one tag, one token
(M3 design 3.8, 8.1).

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**变异**（spec 附录 A）：

| 改坏 | 必须失败的测试 |
|---|---|
| 消息不含 id；只含 id 的一半 | `TestTheTokenOfTheKnownAnswer`、`TestInvitationMessagesDifferByEveryByteOfTheID` |
| 消息不含标签 | `TestTheTokenOfTheKnownAnswer` |
| 解码不带 `Strict()` | `TestParseTokenRefuses`、`TestEveryBitOfATokenCounts` |
| 不看前缀；接受更长的令牌 | `TestParseTokenRefuses` |
| 每个用途一个密钥 | `TestMACKnownAnswers`、`TestMACsOfPurposesDiffer`、`TestKeysMACIsThePurposes` |
| HKDF 只做 extract（没有 info） | `TestMACKnownAnswers` |
| `Verify` 用 `==` | `TestMACVerifyComparesInConstantTime` |
| `Verify` 只比一半 | `TestMACVerify` |
| `Keys.MAC` 给出刷新令牌的用途 | `TestKeysMACRefusesTheRefreshTokensPurpose` |

**Done when:** 两个已知答案与手算的相同；刷新令牌的标签与 M2 相同；`bootstrap` 在任何模块之前载入密钥，`identity.New` 不再读密钥文件。

---

### Task 3: `apitest`：对象的数组、必填的查询参数

**Files:**
- Create: `server/internal/platform/httpserver/apitest/bodycases.go`、`server/internal/platform/httpserver/apitest/bodycases_test.go`
- Modify: `server/internal/bootstrap/contract_test.go`
- Modify（完整内容）: `server/internal/platform/httpserver/apitest/operations.go`、`server/internal/platform/httpserver/apitest/operations_test.go`

**Interfaces:**
- Produces（spec 2.5，M3 设计 5.2、9.4；P2 review 第 6 节第 8 件）：`Operation.BodyCases()` 加两类用例（第一个对象数组的元素里的未声明属性；元素逐个缺少必填属性），"全部问题一起"加上第一类；`validValue` 给对象数组一个有效元素；`ParamCase` 加 `Code`（`invalid_format`，或不带它时的 `required`），每个必填的查询参数一个"不带它"的用例，`Target` 是不带它的例子（`Operation.without(name)`）。`FieldProblem`、`BodyCase`、`BodyCases` 从 `operations.go` 移到 `bodycases.go`（测试移到 `bodycases_test.go`）：`operations.go` 会长到 411 行，请求体用例是单独的一件事。
- 使用者：`bootstrap` 的 `TestBodiesThatBreakTheStructureAnswer400`、`TestParametersThatDoNotBindAnswer400`（Task 6 的 `invitations[]`、Task 9 的 `token`、P4 的 `members[]`、M9）。

**Tests:**
- `bodycases_test.go`：`TestBodyCasesOfAListOfObjects`（有效的请求体在数组里放一个有效元素；元素里的未声明属性、每个必填属性缺少各一个用例；"全部问题一起"含第一类；字符串的数组没有这些用例；只取第一个对象数组）；此前的 `TestBodyCases` 等移来不改。
- `operations_test.go`：`TestParamCases` 改为：只给 Go 类型会拒绝某些字符串的参数错误值；每个必填的查询参数一个"不带它"的用例，枚举的也算；可选的和路径参数没有。
- `bootstrap/contract_test.go`：`TestParametersThatDoNotBindAnswer400` 的核对加上字段和码（`[{field, code}]`），不带必填的查询参数答 400 `bad_request`、`required`。

- [ ] **Step 1: 参数的用例**

`server/internal/platform/httpserver/apitest/operations.go`（完整内容，205 行）：

````whole server/internal/platform/httpserver/apitest/operations.go
package apitest

import (
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// Operation is one operation of the contract, as the whole-program tests of
// bootstrap see it (M2 design 3.6, 3.11).
type Operation struct {
	ID     string // operationId
	Tags   []string
	Method string // upper case
	Path   string
	Public bool // security: []
	// ProblemHeaders are the headers its default response, the problem,
	// declares, sorted.
	ProblemHeaders []string
	params         openapi3.Parameters
	body           *openapi3.Schema
}

// Pattern is the route pattern the generated code registers, e.g.
// "POST /api/v0/auth/register".
func (o Operation) Pattern() string { return o.Method + " " + o.Path }

// HasJSONBody reports whether the operation takes an application/json body.
func (o Operation) HasJSONBody() bool { return o.body != nil }

// Target is the request target of an example call: the path with each path
// parameter, and each required query parameter, set to a valid value.
// Parameters bind before the middlewares, so a wrong one would answer 400
// before anything else runs (M2 design 3.6).
func (o Operation) Target() string {
	return o.target("", "")
}

// target is Target with parameter name set to value, when name is not "".
func (o Operation) target(name, value string) string {
	path, query := o.Path, url.Values{}
	for _, ref := range o.params {
		p := ref.Value
		v, set := fmt.Sprint(validValue(p.Schema.Value)), p.Required
		if p.Name == name {
			v, set = value, true
		}
		switch {
		case p.In == openapi3.ParameterInPath:
			path = strings.ReplaceAll(path, "{"+p.Name+"}", url.PathEscape(v))
		case p.In == openapi3.ParameterInQuery && set:
			query.Set(p.Name, v)
		}
	}
	if len(query) == 0 {
		return path
	}
	return path + "?" + query.Encode()
}

// ParamCase is a request target with one parameter that cannot bind, and
// the parameter and field code the 400 must name.
type ParamCase struct {
	Name   string
	Target string
	Field  string
	Code   string // invalid_format, or required for a parameter left out
}

// ParamCases derives the cases of the parameter binding whole-program test
// (M2 design 3.11): for each path or query parameter whose Go type rejects
// some strings, a number, a boolean, or a string whose format is generated
// as a Go type, the example target with that parameter wrong; for each
// required query parameter, the example target without it. Other strings,
// enums too, bind whatever they are.
func (o Operation) ParamCases() []ParamCase {
	var cases []ParamCase
	for _, ref := range o.params {
		p := ref.Value
		if p.In != openapi3.ParameterInPath && p.In != openapi3.ParameterInQuery {
			continue
		}
		wrong := ""
		switch s := p.Schema.Value; {
		case s.Type.Includes("integer"), s.Type.Includes("number"):
			wrong = "not-a-number"
		case s.Type.Includes("boolean"):
			wrong = "not-a-boolean"
		case checkedFormat(s):
			wrong = "not-a-" + s.Format
		}
		if wrong != "" {
			cases = append(cases, ParamCase{Name: "wrong " + p.Name, Target: o.target(p.Name, wrong), Field: p.Name, Code: "invalid_format"})
		}
		if p.In == openapi3.ParameterInQuery && p.Required {
			cases = append(cases, ParamCase{Name: "missing " + p.Name, Target: o.without(p.Name), Field: p.Name, Code: "required"})
		}
	}
	return cases
}

// without is Target without the query parameter name.
func (o Operation) without(name string) string {
	path, query, _ := strings.Cut(o.Target(), "?")
	values, _ := url.ParseQuery(query)
	values.Del(name)
	if len(values) == 0 {
		return path
	}
	return path + "?" + values.Encode()
}

// Operations lists every operation of the contract, sorted by pattern.
func (c *Contract) Operations() []Operation {
	var ops []Operation
	for path, item := range c.doc.Paths.Map() {
		for method, op := range item.Operations() {
			o := Operation{ID: op.OperationID, Tags: op.Tags, Method: strings.ToUpper(method), Path: path, Public: !needsToken(op),
				params: slices.Concat(item.Parameters, op.Parameters)}
			if rb := op.RequestBody; rb != nil && rb.Value != nil {
				if media := rb.Value.Content.Get("application/json"); media != nil && media.Schema != nil {
					o.body = media.Schema.Value
				}
			}
			if problem := op.Responses.Default(); problem != nil && problem.Value != nil {
				o.ProblemHeaders = slices.Sorted(maps.Keys(problem.Value.Headers))
			}
			ops = append(ops, o)
		}
	}
	slices.SortFunc(ops, func(a, b Operation) int { return strings.Compare(a.Pattern(), b.Pattern()) })
	return ops
}

// validValue is a value that s's structure accepts: every required
// property, the first enum value, a valid string for each checked format,
// one valid item in a list of objects and none in another array.
func validValue(s *openapi3.Schema) any {
	if len(s.AnyOf) > 0 {
		for _, alt := range s.AnyOf {
			if !alt.Value.Type.Is("null") {
				return validValue(alt.Value)
			}
		}
	}
	if len(s.Enum) > 0 {
		return s.Enum[0]
	}
	switch {
	case isObject(s):
		obj := map[string]any{}
		for _, name := range s.Required {
			obj[name] = validValue(s.Properties[name].Value)
		}
		return obj
	case isList(s):
		return []any{validValue(s.Items.Value)}
	case s.Type.Includes("array"):
		return []any{}
	case s.Type.Includes("integer"), s.Type.Includes("number"):
		return 1
	case s.Type.Includes("boolean"):
		return false
	}
	switch s.Format {
	case "date-time":
		return "2026-01-01T00:00:00Z"
	case "uuid":
		return "00000000-0000-0000-0000-000000000000"
	case "email":
		return "someone@example.com"
	}
	return "x"
}

func isObject(s *openapi3.Schema) bool {
	return s.Type.Includes("object") && len(s.AnyOf) == 0
}

// isList reports whether s is an array of objects.
func isList(s *openapi3.Schema) bool {
	return s.Type.Includes("array") && s.Items != nil && s.Items.Value != nil && isObject(s.Items.Value)
}

func isNullable(s *openapi3.Schema) bool {
	if s.Type.IncludesNull() {
		return true
	}
	for _, alt := range s.AnyOf {
		if alt.Value.Type.Is("null") {
			return true
		}
	}
	return false
}

// checkedFormat reports a string format that bodyshape checks: those the
// module template generates as Go types (M2 design 3.12).
func checkedFormat(s *openapi3.Schema) bool {
	return s.Type.Includes("string") && (s.Format == "date-time" || s.Format == "uuid")
}
````

`server/internal/platform/httpserver/apitest/operations_test.go`（完整内容，146 行）：

````whole server/internal/platform/httpserver/apitest/operations_test.go
package apitest

import (
	"slices"
	"testing"
)

const bodiesContract = `
openapi: 3.1.0
info: {title: bodies, version: v0}
x-problem-codes: [bad_request]
paths:
  /api/v0/things:
    post:
      operationId: createThing
      security: [{bearer: []}]
      x-problem-codes: []
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              additionalProperties: false
              required: [name, owner_id]
              properties:
                name: {type: string}
                owner_id: {type: string, format: uuid}
                note: {type: [string, 'null']}
                count: {type: integer}
                kind: {type: string, enum: [a, b]}
                settings:
                  type: object
                  additionalProperties: false
                  properties:
                    notify: {type: boolean}
      responses:
        '204': {description: created}
    get:
      operationId: listThings
      security: []
      x-problem-codes: []
      responses:
        '204': {description: none}
        default:
          description: problem
          headers:
            WWW-Authenticate: {schema: {type: string}}
            Retry-After: {schema: {type: integer}}
  /api/v0/things/{thing_id}:
    parameters:
      - {name: thing_id, in: path, required: true, schema: {type: string, format: uuid}}
    get:
      operationId: getThing
      tags: [things]
      security: [{bearer: []}]
      x-problem-codes: []
      parameters:
        - {name: limit, in: query, required: true, schema: {type: integer}}
        - {name: view, in: query, required: true, schema: {type: string, enum: [full, short]}}
        - {name: q, in: query, schema: {type: string}}
      responses:
        '204': {description: none}
components:
  securitySchemes:
    bearer: {type: http, scheme: bearer}
`

func TestOperations(t *testing.T) {
	ops := contractFrom(t, bodiesContract).Operations()

	if len(ops) != 3 || ops[0].Pattern() != "GET /api/v0/things" || !ops[0].Public || ops[0].HasJSONBody() ||
		ops[1].Pattern() != "GET /api/v0/things/{thing_id}" || ops[1].Public || ops[1].HasJSONBody() ||
		ops[2].Pattern() != "POST /api/v0/things" || ops[2].Public || !ops[2].HasJSONBody() {
		t.Errorf("Operations() = %+v", ops)
	}
	if ops[1].ID != "getThing" || !slices.Equal(ops[1].Tags, []string{"things"}) || ops[0].ID != "listThings" || ops[0].Tags != nil {
		t.Errorf("IDs and tags = %q %q, %q %q; want each operation's", ops[0].ID, ops[0].Tags, ops[1].ID, ops[1].Tags)
	}
	if !slices.Equal(ops[0].ProblemHeaders, []string{"Retry-After", "WWW-Authenticate"}) || ops[1].ProblemHeaders != nil {
		t.Errorf("ProblemHeaders = %q, %q; want the default response's, sorted, and none without one", ops[0].ProblemHeaders, ops[1].ProblemHeaders)
	}
}

func TestTarget(t *testing.T) {
	ops := contractFrom(t, bodiesContract).Operations()

	if got := ops[0].Target(); got != "/api/v0/things" {
		t.Errorf("Target() without parameters = %q", got)
	}
	// The path-level parameter is filled; only the required query parameters
	// are set.
	if got, want := ops[1].Target(), "/api/v0/things/00000000-0000-0000-0000-000000000000?limit=1&view=full"; got != want {
		t.Errorf("Target() = %q, want %q", got, want)
	}
}

// Only the parameters whose Go type rejects some strings get a wrong value:
// the uuid path parameter and the integer, not the enum or the free string.
// Each required query parameter gets a case without it, the enum too; the
// optional one and the path parameter get none. The others keep their
// example values.
func TestParamCases(t *testing.T) {
	ops := contractFrom(t, bodiesContract).Operations()

	got := ops[1].ParamCases()

	const thing = "/api/v0/things/00000000-0000-0000-0000-000000000000"
	want := []ParamCase{
		{"wrong thing_id", "/api/v0/things/not-a-uuid?limit=1&view=full", "thing_id", "invalid_format"},
		{"wrong limit", thing + "?limit=not-a-number&view=full", "limit", "invalid_format"},
		{"missing limit", thing + "?view=full", "limit", "required"},
		{"missing view", thing + "?limit=1", "view", "required"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("ParamCases() =\n%q\nwant\n%q", got, want)
	}
	if got := ops[0].ParamCases(); got != nil {
		t.Errorf("ParamCases() without parameters = %q, want none", got)
	}
}

// An optional parameter gets its case too, set only there. A header
// parameter gets none: a target cannot carry it.
func TestParamCasesOfAnOptionalParameter(t *testing.T) {
	op := contractFrom(t, `
openapi: 3.1.0
info: {title: params, version: v0}
paths:
  /api/v0/x:
    get:
      parameters:
        - {name: flag, in: query, schema: {type: boolean}}
        - {name: at, in: query, schema: {type: string, format: date-time}}
        - {name: X-Page, in: header, schema: {type: integer}}
      responses: {'204': {description: none}}
`).Operations()[0]

	want := []ParamCase{
		{"wrong flag", "/api/v0/x?flag=not-a-boolean", "flag", "invalid_format"},
		{"wrong at", "/api/v0/x?at=not-a-date-time", "at", "invalid_format"},
	}
	if got := op.ParamCases(); !slices.Equal(got, want) || op.Target() != "/api/v0/x" {
		t.Errorf("ParamCases() = %q, Target() = %q; want %q and no query", got, op.Target(), want)
	}
}
````

- [ ] **Step 2: 请求体的用例**

`server/internal/platform/httpserver/apitest/bodycases.go`（新文件，216 行）：

````file server/internal/platform/httpserver/apitest/bodycases.go
package apitest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// FieldProblem is one entry of a problem's errors: field and code.
type FieldProblem struct {
	Field string `json:"field"`
	Code  string `json:"code"`
}

// BodyCase is a request body that breaks the operation's structure in one
// way, and what it must get: 400 bad_request with exactly Fields, in order;
// or, when Accepted, anything but 400.
type BodyCase struct {
	Name     string
	Body     []byte
	Fields   []FieldProblem
	Accepted bool
}

// unknownField is the property no schema declares.
const unknownField = "nerve_undeclared"

// rawMark stands for a value that json.Marshal cannot write, until BodyCases
// puts the JSON text in its place.
const rawMark = "nerve_raw_value"

// BodyCases derives the cases of M2 design 3.11's fourth whole-program test
// from the operation's body schema: a valid body, broken one way at a time.
// Cases whose kind of field the schema lacks are left out.
//
//  1. an undeclared top-level property;
//  2. an undeclared property in a nested object;
//  3. null for an optional property that is not nullable;
//  4. each required property missing;
//  5. null for a nullable property: not 400;
//  6. each format property with a wrong string;
//  7. the first property twice: duplicate (a body read two ways);
//  8. the first property of the nested object twice, in it;
//  9. bytes that are not UTF-8 in the first string property: invalid_format;
//  10. every kind of 1–4, 6 and 11 that the schema has, each on a property
//     of its own, together: every problem in one answer. Left out when the
//     schema has only the first kind. 7–9 and 12 are not in it: a body read
//     two ways gets only those answers, before its structure is checked, and
//     12 would take the list 11 takes;
//  11. an undeclared property in the item of the first array of objects
//     (M3 design 5.2), which the valid body lists once;
//  12. each required property of that item missing.
func (o Operation) BodyCases() []BodyCase {
	s := o.body
	valid := validValue(s).(map[string]any)
	with := func(change func(body map[string]any)) []byte {
		body := maps.Clone(valid)
		change(body)
		out, _ := json.Marshal(body)
		return out
	}
	names := slices.Sorted(maps.Keys(s.Properties))
	var nested, listed string
	var optional, required, formatted []string
	for _, name := range names {
		p := s.Properties[name].Value
		if nested == "" && isObject(p) {
			nested = name
		}
		if listed == "" && isList(p) {
			listed = name
		}
		if !isNullable(p) && !slices.Contains(s.Required, name) {
			optional = append(optional, name)
		}
		if checkedFormat(p) {
			formatted = append(formatted, name)
		}
	}
	required = slices.Sorted(slices.Values(s.Required))
	undeclaredIn := func(name string) map[string]any {
		v := validValue(s.Properties[name].Value).(map[string]any)
		v[unknownField] = 1
		return v
	}
	// item is the valid item of the list with change made to it, listed as
	// the list's only item.
	item := func(change func(item map[string]any)) []any {
		v := validValue(s.Properties[listed].Value.Items.Value).(map[string]any)
		change(v)
		return []any{v}
	}
	wrong := func(name string) string { return "not-a-" + s.Properties[name].Value.Format }
	// raw is the valid body with name's value written as text, which json.Marshal would not write: a
	// second member of the same name, bytes that are not UTF-8. The mark must be the body's only one, or
	// the text would land elsewhere, or nowhere, and the case would test something else.
	raw := func(name, text string) []byte {
		body := maps.Clone(valid)
		body[name] = rawMark
		out, _ := json.Marshal(body)
		mark := []byte(`"` + rawMark + `"`)
		if n := bytes.Count(out, mark); n != 1 {
			panic(fmt.Sprintf("apitest: the body %s has the raw mark %d times, want once", out, n))
		}
		return bytes.Replace(out, mark, []byte(text), 1)
	}
	// twice is the text of name's value v, then of a second member name: v.
	twice := func(name string, v any) string {
		value, _ := json.Marshal(v)
		return string(value) + `,"` + name + `":` + string(value)
	}
	// valueOf is name's value in values, else a valid one for its schema.
	valueOf := func(schema *openapi3.Schema, name string, values map[string]any) any {
		if v, ok := values[name]; ok {
			return v
		}
		return validValue(schema.Properties[name].Value)
	}

	cases := []BodyCase{{Name: "undeclared property", Body: with(func(b map[string]any) { b[unknownField] = 1 }),
		Fields: []FieldProblem{{unknownField, "not_allowed"}}}}
	if nested != "" {
		cases = append(cases, BodyCase{Name: "undeclared property in " + nested, Body: with(func(b map[string]any) { b[nested] = undeclaredIn(nested) }),
			Fields: []FieldProblem{{nested + "." + unknownField, "not_allowed"}}})
	}
	for _, name := range names {
		switch {
		case isNullable(s.Properties[name].Value):
			cases = append(cases, BodyCase{Name: "null for nullable " + name, Body: with(func(b map[string]any) { b[name] = nil }), Accepted: true})
		case slices.Contains(optional, name):
			cases = append(cases, BodyCase{Name: "null for optional " + name, Body: with(func(b map[string]any) { b[name] = nil }),
				Fields: []FieldProblem{{name, "invalid_format"}}})
		}
	}
	for _, name := range required {
		cases = append(cases, BodyCase{Name: "missing " + name, Body: with(func(b map[string]any) { delete(b, name) }),
			Fields: []FieldProblem{{name, "required"}}})
	}
	for _, name := range formatted {
		cases = append(cases, BodyCase{Name: "wrong " + s.Properties[name].Value.Format + " in " + name,
			Body: with(func(b map[string]any) { b[name] = wrong(name) }), Fields: []FieldProblem{{name, "invalid_format"}}})
	}
	if len(names) > 0 {
		name := names[0]
		cases = append(cases, BodyCase{Name: name + " twice", Body: raw(name, twice(name, valueOf(s, name, valid))),
			Fields: []FieldProblem{{name, "duplicate"}}})
	}
	if nested != "" {
		n := s.Properties[nested].Value
		if inner := slices.Sorted(maps.Keys(n.Properties)); len(inner) > 0 {
			text := `{"` + inner[0] + `":` + twice(inner[0], valueOf(n, inner[0], nil)) + `}`
			cases = append(cases, BodyCase{Name: nested + "." + inner[0] + " twice", Body: raw(nested, text),
				Fields: []FieldProblem{{nested + "." + inner[0], "duplicate"}}})
		}
	}
	if i := slices.IndexFunc(names, func(name string) bool { return s.Properties[name].Value.Type.Includes("string") }); i >= 0 {
		cases = append(cases, BodyCase{Name: "not UTF-8 in " + names[i], Body: raw(names[i], "\"\xff\""),
			Fields: []FieldProblem{{names[i], "invalid_format"}}})
	}
	if listed != "" {
		cases = append(cases, BodyCase{Name: "undeclared property in an item of " + listed,
			Body:   with(func(b map[string]any) { b[listed] = item(func(i map[string]any) { i[unknownField] = 1 }) }),
			Fields: []FieldProblem{{listed + "[0]." + unknownField, "not_allowed"}}})
		for _, name := range slices.Sorted(slices.Values(s.Properties[listed].Value.Items.Value.Required)) {
			cases = append(cases, BodyCase{Name: "missing " + name + " in an item of " + listed,
				Body:   with(func(b map[string]any) { b[listed] = item(func(i map[string]any) { delete(i, name) }) }),
				Fields: []FieldProblem{{listed + "[0]." + name, "required"}}})
		}
	}

	// Case 10: each kind takes the first property no earlier kind took.
	body := maps.Clone(valid)
	body[unknownField] = 1
	all := []FieldProblem{{unknownField, "not_allowed"}}
	used := map[string]bool{}
	first := func(names []string) string {
		for _, name := range names {
			if !used[name] {
				used[name] = true
				return name
			}
		}
		return ""
	}
	if name := first([]string{nested}); name != "" {
		body[name] = undeclaredIn(name)
		all = append(all, FieldProblem{name + "." + unknownField, "not_allowed"})
	}
	if name := first([]string{listed}); name != "" {
		body[name] = item(func(i map[string]any) { i[unknownField] = 1 })
		all = append(all, FieldProblem{name + "[0]." + unknownField, "not_allowed"})
	}
	if name := first(optional); name != "" {
		body[name] = nil
		all = append(all, FieldProblem{name, "invalid_format"})
	}
	if name := first(required); name != "" {
		delete(body, name)
		all = append(all, FieldProblem{name, "required"})
	}
	if name := first(formatted); name != "" {
		body[name] = wrong(name)
		all = append(all, FieldProblem{name, "invalid_format"})
	}
	if len(all) >= 2 {
		slices.SortFunc(all, func(a, b FieldProblem) int { return strings.Compare(a.Field, b.Field) })
		out, _ := json.Marshal(body)
		cases = append(cases, BodyCase{Name: "every problem at once", Body: out, Fields: all})
	}
	return cases
}
````

`server/internal/platform/httpserver/apitest/bodycases_test.go`（新文件，146 行）：

````file server/internal/platform/httpserver/apitest/bodycases_test.go
package apitest

import (
	"slices"
	"strings"
	"testing"
)

func TestBodyCases(t *testing.T) {
	post := contractFrom(t, bodiesContract).Operations()[2]

	var got []string
	for _, c := range post.BodyCases() {
		got = append(got, c.Name+" "+string(c.Body))
		if c.Accepted != (len(c.Fields) == 0) {
			t.Errorf("case %s: accepted %v with fields %v", c.Name, c.Accepted, c.Fields)
		}
	}
	const owner = `"owner_id":"00000000-0000-0000-0000-000000000000"`
	valid := `"name":"x",` + owner
	want := []string{
		`undeclared property {"name":"x","nerve_undeclared":1,` + owner + `}`,
		`undeclared property in settings {` + valid + `,"settings":{"nerve_undeclared":1}}`,
		`null for optional count {"count":null,` + valid + `}`,
		`null for optional kind {"kind":null,` + valid + `}`,
		`null for nullable note {"name":"x","note":null,` + owner + `}`,
		`null for optional settings {` + valid + `,"settings":null}`,
		`missing name {"owner_id":"00000000-0000-0000-0000-000000000000"}`,
		`missing owner_id {"name":"x"}`,
		`wrong uuid in owner_id {"name":"x","owner_id":"not-a-uuid"}`,
		`count twice {"count":1,"count":1,` + valid + `}`,
		`settings.notify twice {` + valid + `,"settings":{"notify":false,"notify":false}}`,
		"not UTF-8 in kind {\"kind\":\"\xff\"," + valid + `}`,
		`every problem at once {"count":null,"nerve_undeclared":1,"owner_id":"not-a-uuid","settings":{"nerve_undeclared":1}}`,
	}
	if !slices.Equal(got, want) {
		t.Errorf("BodyCases() =\n%q\nwant\n%q", got, want)
	}
	last := post.BodyCases()[len(want)-1]
	wantFields := []FieldProblem{{"count", "invalid_format"}, {"name", "required"}, {"nerve_undeclared", "not_allowed"},
		{"owner_id", "invalid_format"}, {"settings.nerve_undeclared", "not_allowed"}}
	if !slices.Equal(last.Fields, wantFields) {
		t.Errorf("every problem at once expects %v, want %v", last.Fields, wantFields)
	}
}

// The case with every problem at once combines whichever kinds the schema
// has, each on a property no other kind took, as soon as there are two; a
// schema with only undeclared properties to offer has no such case.
func TestBodyCasesCombineEveryKindTheSchemaHas(t *testing.T) {
	tests := []struct {
		name, schema string
		want         string // the body of "every problem at once", or "" for none
		fields       []FieldProblem
	}{
		{"no required property", "{type: object, properties: {label: {type: string}, due: {type: [string, 'null'], format: date-time}}}",
			`{"due":"not-a-date-time","label":null,"nerve_undeclared":1}`,
			[]FieldProblem{{"due", "invalid_format"}, {"label", "invalid_format"}, {"nerve_undeclared", "not_allowed"}}},
		{"only a nested object", "{type: object, properties: {step: {type: object, properties: {a: {type: boolean}}}}}",
			`{"nerve_undeclared":1,"step":{"nerve_undeclared":1}}`,
			[]FieldProblem{{"nerve_undeclared", "not_allowed"}, {"step.nerve_undeclared", "not_allowed"}}},
		{"the required property is the only formatted one", "{type: object, required: [id], properties: {id: {type: string, format: uuid}}}",
			`{"nerve_undeclared":1}`, []FieldProblem{{"id", "required"}, {"nerve_undeclared", "not_allowed"}}},
		{"only nullable properties", "{type: object, properties: {note: {type: [string, 'null']}}}", "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cases := bodyOperation(t, tt.schema).BodyCases()
			var got *BodyCase
			for i := range cases {
				if cases[i].Name == "every problem at once" {
					got = &cases[i]
				}
			}
			switch {
			case tt.want == "" && got != nil:
				t.Errorf("every problem at once = %s, want no such case", got.Body)
			case tt.want != "" && (got == nil || string(got.Body) != tt.want || !slices.Equal(got.Fields, tt.fields)):
				t.Errorf("every problem at once = %+v, want %s with %v", got, tt.want, tt.fields)
			}
		})
	}
}

// A list of objects (M3 design 5.2) has one valid item in the valid body,
// and its cases: an undeclared property in the item, each required
// property of the item missing, and the first in the case with every
// problem at once. An array of strings has none, and none but the first
// list of objects is taken.
func TestBodyCasesOfAListOfObjects(t *testing.T) {
	const item = "{type: object, additionalProperties: false, required: [role, email], " +
		"properties: {email: {type: string, format: email}, role: {type: integer, enum: [5, 15, 20]}, note: {type: string}}}"
	op := bodyOperation(t, "{type: object, required: [invitations, tags], properties: {invitations: {type: array, items: "+item+
		"}, tags: {type: array, items: {type: string}}, zones: {type: array, items: "+item+"}}}")

	byName := map[string]BodyCase{}
	for _, c := range op.BodyCases() {
		byName[c.Name] = c
	}
	const listed = `"invitations":[{"email":"someone@example.com","role":5}]`
	for _, want := range []BodyCase{
		{Name: "undeclared property in an item of invitations", Body: []byte(`{"invitations":[{"email":"someone@example.com",` +
			`"nerve_undeclared":1,"role":5}],"tags":[]}`), Fields: []FieldProblem{{"invitations[0].nerve_undeclared", "not_allowed"}}},
		{Name: "missing email in an item of invitations", Body: []byte(`{"invitations":[{"role":5}],"tags":[]}`),
			Fields: []FieldProblem{{"invitations[0].email", "required"}}},
		{Name: "missing role in an item of invitations", Body: []byte(`{"invitations":[{"email":"someone@example.com"}],"tags":[]}`),
			Fields: []FieldProblem{{"invitations[0].role", "required"}}},
		{Name: "missing tags", Body: []byte("{" + listed + "}"), Fields: []FieldProblem{{"tags", "required"}}},
		{Name: "every problem at once", Body: []byte(`{"invitations":[{"email":"someone@example.com","nerve_undeclared":1,"role":5}],` +
			`"nerve_undeclared":1,"zones":null}`), Fields: []FieldProblem{{"invitations[0].nerve_undeclared", "not_allowed"},
			{"nerve_undeclared", "not_allowed"}, {"tags", "required"}, {"zones", "invalid_format"}}},
	} {
		got, ok := byName[want.Name]
		if !ok || string(got.Body) != string(want.Body) || !slices.Equal(got.Fields, want.Fields) {
			t.Errorf("%s = %s %v, want %s %v", want.Name, got.Body, got.Fields, want.Body, want.Fields)
		}
	}
	for name := range byName {
		if strings.Contains(name, "item of tags") || strings.Contains(name, "item of zones") {
			t.Errorf("case %q: only the first list of objects has item cases", name)
		}
	}
}

// A case written as text puts the text in place of the one mark it wrote
// into the body. A valid value that is the mark as well would make the case
// test something else, so BodyCases fails loudly.
func TestBodyCasesPanicWhenAValidValueIsTheRawMark(t *testing.T) {
	op := bodyOperation(t, "{type: object, required: [kind], properties: {a: {type: string}, kind: {type: string, enum: [nerve_raw_value]}}}")
	defer func() {
		if recover() == nil {
			t.Error("BodyCases() returned, want a panic: the body holds the raw mark twice")
		}
	}()
	op.BodyCases()
}

// bodyOperation returns the one operation of a contract whose JSON body has
// the given schema.
func bodyOperation(t *testing.T, schema string) Operation {
	t.Helper()
	src := "openapi: 3.1.0\ninfo: {title: body, version: v0}\npaths:\n  /api/v0/x:\n    post:\n" +
		"      requestBody:\n        content:\n          application/json:\n            schema: " + schema + "\n" +
		"      responses: {'204': {description: none}}\n"
	return contractFrom(t, src).Operations()[0]
}
````

- [ ] **Step 3: 整程序测试**

`server/internal/bootstrap/contract_test.go`（修改，2 处）：

````old server/internal/bootstrap/contract_test.go
// answers a wrong value with 400 that names it (M2 design 3.11, M0-P3
// handoff 2), before authentication: parameters bind first (3.6), so no
// token is sent.
````

````new server/internal/bootstrap/contract_test.go
// answers a wrong value with 400 that names it, and every required query
// parameter answers its absence with 400 required (M2 design 3.11, M0-P3
// handoff 2, M3 design 9.4), before authentication: parameters bind first
// (3.6), so no token is sent.
````

````old server/internal/bootstrap/contract_test.go
				want := []apitest.FieldProblem{{Field: c.Field, Code: "invalid_format"}}
````

````new server/internal/bootstrap/contract_test.go
				want := []apitest.FieldProblem{{Field: c.Field, Code: c.Code}}
````

- [ ] **Step 4: 测试、lint**

Run: `go -C server test -count=1 ./internal/platform/httpserver/apitest/`
Expected: `ok`。

Run: `go -C server test -count=1 -run 'TestParametersThatDoNotBindAnswer400|TestBodiesThatBreakTheStructureAnswer400' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 5: 提交**

```bash
git add server/internal/bootstrap/contract_test.go server/internal/platform/httpserver/apitest/bodycases.go server/internal/platform/httpserver/apitest/bodycases_test.go server/internal/platform/httpserver/apitest/operations.go server/internal/platform/httpserver/apitest/operations_test.go
```
```bash
git commit -m "test(M3/P3): apitest derives the cases of a list of objects and of a required query parameter

The body cases break the item of the first array of objects too: an
undeclared property in it, each required property missing (M3 design
5.2); the case with every problem holds the item's. Each required query
parameter gets a case without it, answered 400 required. The body cases
move to bodycases.go, a thing of their own.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**变异**（spec 附录 A）：

| 改坏 | 必须失败的测试 |
|---|---|
| 不给对象数组的元素任何用例 | `TestBodyCasesOfAListOfObjects` |
| 不给元素缺少必填属性的用例 | `TestBodyCasesOfAListOfObjects` |
| "全部问题一起"不含元素的问题 | `TestBodyCasesOfAListOfObjects` |
| 字符串的数组也算对象数组 | `TestBodyCasesOfAListOfObjects` |
| 必填的查询参数没有"不带它"的用例 | `TestParamCases` |
| 可选的查询参数也有"不带它"的用例 | `TestParamCases`、`TestParametersThatDoNotBindAnswer400` |

**Done when:** 两个整程序测试照契约生成的用例通过；`apitest` 的每个文件在 400 行以内。

---

### Task 4: `identity.Provide` 交出 `CredentialLock`

**Files:**
- Create: `server/internal/modules/identity/provide_test.go`
- Modify: `server/internal/modules/identity/module.go`、`server/internal/modules/identity/provide.go`

**Interfaces:**
- Produces（spec 2.6，M3 设计 6.5、6.6）：`identity.CredentialLock{LockCaller(ctx, actor shared.Actor, now time.Time) error}`；`identity.Provided.CredentialLock`；`credentialLock(store)` 是 `identity.New` 和 `Provide` 共用的 `app.CredentialLock`；`callerLock` 丢掉锁到的账户（别的模块拿不到密码的哈希）。
- 使用者：Task 6（`workspace` 的 `CallerLock`）。

**Tests:**
- `identity/provide_test.go`：`TestProvidedCredentialLockIsTheCredentialLock`：有效的会话和 PAT 取得锁，账户行锁到事务结束（事务中另一个连接的 `FOR NO KEY UPDATE NOWAIT` 得到 `55P03`，之后得到这一行）；撤销的会话、撤销的 PAT、停用的账户、不存在的账户都是 401 `unauthorized`。

- [ ] **Step 1: `Provide`**

`server/internal/modules/identity/provide.go`（修改，6 处）：

````old server/internal/modules/identity/provide.go
	"context"
````

````new server/internal/modules/identity/provide.go
	"context"
	"time"
````

````old server/internal/modules/identity/provide.go
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
````

````new server/internal/modules/identity/provide.go
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/shared"
````

````old server/internal/modules/identity/provide.go
}

// Provided are the adapters identity offers the other modules. They depend
````

````new server/internal/modules/identity/provide.go
}

// CredentialLock is M2's credential lock (M2 design 3.5) as another module
// takes it: LockCaller locks the caller's account row FOR NO KEY UPDATE
// until the transaction ends, then checks under the lock that the account
// is active and the caller's session or personal access token valid at now;
// 401 unauthorized otherwise. It is the first lock of a transaction that
// issues something with the caller's credential: creating invitations (M3
// design 3.8), so that a password reset committed first leaves none.
type CredentialLock interface {
	LockCaller(ctx context.Context, actor shared.Actor, now time.Time) error
}

// Provided are the adapters identity offers the other modules. They depend
````

````old server/internal/modules/identity/provide.go
// 6.6, step 2).
````

````new server/internal/modules/identity/provide.go
// 6.6, step 2); CredentialLock is identity's own protocol step, on its
// store alone.
````

````old server/internal/modules/identity/provide.go
	PublicProfiles PublicProfiles
````

````new server/internal/modules/identity/provide.go
	PublicProfiles PublicProfiles
	CredentialLock CredentialLock
````

````old server/internal/modules/identity/provide.go
	return Provided{Accounts: store, PublicProfiles: store}
}

````

````new server/internal/modules/identity/provide.go
	return Provided{Accounts: store, PublicProfiles: store, CredentialLock: callerLock{lock: credentialLock(store)}}
}

// credentialLock is the credential lock over the store, identity's use
// cases' and the other modules' alike.
func credentialLock(store *postgresadapter.Store) app.CredentialLock {
	return app.CredentialLock{Locker: store, Sessions: store, APITokens: store}
}

// callerLock is the credential lock without the locked account: another
// module learns only whether the caller's credential holds, never the
// account's password hash.
type callerLock struct {
	lock app.CredentialLock
}

func (l callerLock) LockCaller(ctx context.Context, actor shared.Actor, now time.Time) error {
	_, err := l.lock.Lock(ctx, actor, now)
	return err
}

````

`server/internal/modules/identity/module.go`（修改，1 处）：

````old server/internal/modules/identity/module.go
	lock := app.CredentialLock{Locker: store, Sessions: store, APITokens: store}
````

````new server/internal/modules/identity/module.go
	lock := credentialLock(store)
````

`server/internal/modules/identity/provide_test.go`（新文件，85 行）：

````file server/internal/modules/identity/provide_test.go
package identity_test

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// rowLockedByAnother reports whether alice's account row is locked against
// FOR NO KEY UPDATE by another transaction: NOWAIT answers lock_not_available
// (55P03) at once instead of waiting.
func (a *account) rowLockedByAnother(t *testing.T) bool {
	t.Helper()
	_, err := a.pool.Exec(context.Background(), "SELECT 1 FROM users WHERE id = $1 FOR NO KEY UPDATE NOWAIT", a.id)
	var pgErr *pgconn.PgError
	if err != nil && (!errors.As(err, &pgErr) || pgErr.Code != "55P03") {
		t.Fatal(err)
	}
	return err != nil
}

// Provide's CredentialLock is M2's credential lock (M3 design 6.5, 6.6): a
// valid session or personal access token takes it, and the account row stays
// locked until the transaction ends; a revoked session, a revoked token, a
// deactivated account and an unknown one are 401 unauthorized.
func TestProvidedCredentialLockIsTheCredentialLock(t *testing.T) {
	a := newAccount(t, "hashed:Tr0ub4dor&3:s")
	lock := identity.Provide(a.pool).CredentialLock
	ctx, now := context.Background(), time.Now()
	session := shared.Actor{UserID: a.id, SessionID: a.session}
	pat := shared.Actor{UserID: a.id, APITokenID: uuid.NewV7()}
	if err := a.store.CreateAPIToken(ctx, app.NewAPIToken{ID: pat.APITokenID, UserID: a.id, TokenHash: make([]byte, 32), Label: "ci", Now: now}); err != nil {
		t.Fatal(err)
	}

	for name, actor := range map[string]shared.Actor{"a session": session, "a personal access token": pat} {
		var held bool
		err := a.tx.WithinTx(ctx, func(ctx context.Context) error {
			if err := lock.LockCaller(ctx, actor, now); err != nil {
				return err
			}
			held = a.rowLockedByAnother(t)
			return nil
		})
		if err != nil || !held || a.rowLockedByAnother(t) {
			t.Errorf("%s: LockCaller() = %v, row held %v; want the row locked until the transaction ended", name, err, held)
		}
	}

	if _, err := a.store.RevokeAPIToken(ctx, pat.APITokenID, a.id, now); err != nil {
		t.Fatal(err)
	}
	if err := lock.LockCaller(ctx, pat, now); !errors.Is(err, shared.Unauthenticated()) {
		t.Errorf("a revoked token: LockCaller() = %v, want 401 unauthorized", err)
	}
	if _, err := a.store.RevokeSessions(ctx, a.id, uuid.Nil(), domain.RevokePasswordReset, now); err != nil {
		t.Fatal(err)
	}
	if err := lock.LockCaller(ctx, session, now); !errors.Is(err, shared.Unauthenticated()) {
		t.Errorf("a revoked session: LockCaller() = %v, want 401 unauthorized", err)
	}
	live := shared.Actor{UserID: a.id, SessionID: uuid.NewV7()}
	if err := a.store.CreateSession(ctx, app.NewSession{ID: live.SessionID, UserID: a.id, TokenHash: make([]byte, 32),
		ExpiresAt: now.Add(time.Hour), Now: now}); err != nil {
		t.Fatal(err)
	}
	if err := a.store.DeactivateUser(ctx, a.id, now); err != nil {
		t.Fatal(err)
	}
	if err := lock.LockCaller(ctx, live, now); !errors.Is(err, shared.Unauthenticated()) {
		t.Errorf("a deactivated account: LockCaller() = %v, want 401 unauthorized", err)
	}
	if err := lock.LockCaller(ctx, shared.Actor{UserID: uuid.NewV7(), SessionID: live.SessionID}, now); !errors.Is(err, shared.Unauthenticated()) {
		t.Errorf("an unknown account: LockCaller() = %v, want 401 unauthorized", err)
	}
}
````

- [ ] **Step 2: 测试、lint**

Run: `go -C server test -count=1 ./internal/modules/identity/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 3: 提交**

```bash
git add server/internal/modules/identity/module.go server/internal/modules/identity/provide.go server/internal/modules/identity/provide_test.go
```
```bash
git commit -m "feat(M3/P3): identity.Provide offers the credential lock

M2's credential lock (M2 design 3.5) as another module takes it: the
caller's account row FOR NO KEY UPDATE until the transaction ends, his
session or token checked under it, 401 otherwise; the locked account
stays inside identity. It uses identity's store alone, so Provide gives
it (M3 design 6.6).

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**变异**（spec 附录 A）：

| 改坏 | 必须失败的测试 |
|---|---|
| 提供的凭证锁什么都不锁、不查 | `TestProvidedCredentialLockIsTheCredentialLock` |

**Done when:** 凭证锁的四种拒绝和持锁到事务结束由测试核对。

---

### Task 5: `listWorkspaceInvitations`；邀请的 MAC 接进 `workspace`

**Files:**
- Create: `server/internal/bootstrap/invitation_mac_test.go`、`server/internal/bootstrap/permission_matrix_invitations_test.go`、`server/internal/modules/workspace/adapter/http/invitations.go`、`server/internal/modules/workspace/adapter/http/invitations_test.go`、`server/internal/modules/workspace/app/fakes_invitations_test.go`、`server/internal/modules/workspace/app/list_invitations.go`、`server/internal/modules/workspace/app/list_invitations_test.go`、`server/internal/modules/workspace/app/tokens.go`
- Modify: `api/modules/workspace.yaml`、`api/openapi.yaml`、`server/internal/bootstrap/app.go`、`server/internal/bootstrap/permission_matrix_seeded_test.go`、`server/internal/bootstrap/permission_matrix_test.go`、`server/internal/bootstrap/permission_matrix_workspace_test.go`、`server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`、`server/internal/modules/workspace/adapter/http/handler.go`、`server/internal/modules/workspace/adapter/http/handler_test.go`、`server/internal/modules/workspace/adapter/postgres/failures_test.go`、`server/internal/modules/workspace/adapter/postgres/invitations.go`、`server/internal/modules/workspace/adapter/postgres/invitations_test.go`、`server/internal/modules/workspace/adapter/postgres/queries/invitations.sql`、`server/internal/modules/workspace/app/invitation_ports.go`、`server/internal/modules/workspace/domain/actions.go`、`server/internal/modules/workspace/domain/invitation.go`、`server/internal/modules/workspace/module.go`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/workspace/adapter/http/gen/server.gen.go`、`server/internal/modules/workspace/adapter/postgres/gen/invitations.sql.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Produces（spec 2.7，M3 设计 3.8、3.12、5.1、5.2）：接口 `GET /api/v0/workspaces/{slug}/invitations`，`WorkspaceInvitationList`、`WorkspaceInvitation`；`domain.InvitationWithToken{Invitation, Token}`；`domain.ActionInvitationList`；规则 `workspace_invitation.list`：管理员；`app.InvitationMAC`、`app.InvitationLister`、`app.NewListWorkspaceInvitations(invitations, auth, mac)`；`invitationTokens{mac}` 的 `of(id)`、`withTokens(list)`；存储 `ListInvitations(ctx, workspaceID)`（未删除的，`created_at DESC, id`）；`workspace.InvitationMACPurpose`、`workspace.Deps.InvitationMAC`；`bootstrap` 以 `keys.MAC(workspace.InvitationMACPurpose)` 接上。矩阵：`matrixInvitations`（准备之前定名）、`seeded.invitation(slug, email)`、`invitationToken`、`listsTheInvitations`。
- 使用者：Task 6、8、9、11、12 的用例用 `invitationTokens`；Task 6 起的矩阵行。

**Tests:**
- `app/list_invitations_test.go`：`TestListWorkspaceInvitationsListsEachWithItsToken`（存储的顺序，每一条带 MAC 给它 id 的令牌，不开事务，判定在读之前；两位管理员、两个工作区）；`TestTheTokensAreTheMACs`（别的密钥给别的令牌）；`TestListWorkspaceInvitationsRefusals`（工作区不在、看不到 404；`forbidden` 在读之前；失败原样、不是 404；没有调用者 401、什么都不读）。
- `adapter/postgres/invitations_test.go`：`TestListInvitations`（待接受和已忽略的，新的在前、再按 id；已接受、已删除的和别的工作区的不在）；`failures_test.go` 加这个读。
- `adapter/http/invitations_test.go`：`TestListWorkspaceInvitations`（路径的 slug、调用者交给用例；一条待接受、一条已忽略且邀请人已不在的）；`TestListWorkspaceInvitationsRefusals`（契约声明的码）。
- `bootstrap/invitation_mac_test.go`：`TestTheInvitationMACIsTheDesigns`（组合根要的用途给出 spec 的已知答案）。
- 矩阵：`listWorkspaceInvitations` 一行（管理员 200、成员和访客 403、另三列 404），答案由 `listsTheInvitations` 核对；`access` 的 `TestEveryRuleDecidesItsCells` 加这一行的格子。

- [ ] **Step 1: 接口描述、查询**

`api/modules/workspace.yaml`（修改，3 处）：

````old api/modules/workspace.yaml
        For the workspace's admins. The workspace, its memberships and the
        members' display settings are soft-deleted in one transaction, at the
        same moment; the members' accounts stay. The slug can name a new
````

````new api/modules/workspace.yaml
        For the workspace's admins. The workspace, its invitations, its
        memberships and the members' display settings are soft-deleted in one
        transaction, at the same moment; the members' accounts stay. The slug can name a new
````

````old api/modules/workspace.yaml
                $ref: '#/components/schemas/WorkspaceMemberList'
````

````new api/modules/workspace.yaml
                $ref: '#/components/schemas/WorkspaceMemberList'
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/workspaces/{slug}/invitations:
    parameters:
      - $ref: '#/components/parameters/Slug'
    get:
      operationId: listWorkspaceInvitations
      tags: [workspace]
      summary: List a workspace's invitations
      description: >-
        The workspace's invitations, pending and declined, newest first, then
        by id, each with the token of its link: for the workspace's admins
        alone. An accepted invitation is no longer listed. A workspace that
        does not exist, is deleted, or of which the caller is not an active
        member answers workspace.not_found; a member or a guest, forbidden.
        The whole collection at once: collections are not paginated.
      security: [{bearer: []}]
      x-problem-codes: [workspace.not_found, forbidden]
      responses:
        '200':
          description: The workspace's invitations.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/WorkspaceInvitationList'
````

````old api/modules/workspace.yaml
        navigation_project_limit:
          $ref: '#/components/schemas/NavigationProjectLimit'
    SlugAvailability:
````

````new api/modules/workspace.yaml
        navigation_project_limit:
          $ref: '#/components/schemas/NavigationProjectLimit'
    WorkspaceInvitation:
      description: >-
        An invitation of an address to the workspace, pending or declined
        (responded_at set). Only who may manage the workspace's invitations
        is given one, with the token of its link:
        /workspace-invitations?invitation_id={id}&token={token}.
      type: object
      additionalProperties: false
      required: [id, workspace_id, email, role, accepted, responded_at, created_at, created_by_id, token]
      properties:
        id:
          description: The invitation's id, the link's invitation_id.
          type: string
          format: uuid
        workspace_id:
          type: string
          format: uuid
        email:
          description: The address invited, normalized.
          type: string
        role:
          $ref: '#/components/schemas/WorkspaceRole'
        accepted:
          description: False; an invitation is deleted as it is accepted, and never listed again.
          type: boolean
        responded_at:
          description: When the invitation was declined; null while it is pending.
          type: [string, 'null']
          format: date-time
        created_at:
          type: string
          format: date-time
        created_by_id:
          description: The account that invited; null once that account is gone.
          type: [string, 'null']
          format: uuid
        token:
          description: >-
            nrv_inv_ and 22 base64url characters. The server does not store
            it: it computes it again from the invitation's id with a key
            derived from the instance's signing key, so a new signing key
            gives every invitation a new token and voids the old links.
          type: string
    WorkspaceInvitationList:
      type: object
      additionalProperties: false
      required: [data]
      properties:
        data:
          type: array
          items:
            $ref: '#/components/schemas/WorkspaceInvitation'
    SlugAvailability:
````

`api/openapi.yaml`（修改，1 处）：

````old api/openapi.yaml
    $ref: 'modules/workspace.yaml#/paths/~1api~1v0~1workspaces~1{slug}~1members'
````

````new api/openapi.yaml
    $ref: 'modules/workspace.yaml#/paths/~1api~1v0~1workspaces~1{slug}~1members'
  /api/v0/workspaces/{slug}/invitations:
    $ref: 'modules/workspace.yaml#/paths/~1api~1v0~1workspaces~1{slug}~1invitations'
````

`server/internal/modules/workspace/adapter/postgres/queries/invitations.sql`（修改，1 处）：

````old server/internal/modules/workspace/adapter/postgres/queries/invitations.sql
RETURNING id, workspace_id, email, role, accepted, responded_at, created_by_id, updated_by_id, created_at, updated_at, deleted_at;

````

````new server/internal/modules/workspace/adapter/postgres/queries/invitations.sql
RETURNING id, workspace_id, email, role, accepted, responded_at, created_by_id, updated_by_id, created_at, updated_at, deleted_at;

-- name: ListInvitations :many
-- listWorkspaceInvitations: the workspace's undeleted invitations, pending or declined, newest first, then by id
-- (M3 design 3.12).
SELECT id, workspace_id, email, role, accepted, responded_at, created_by_id, updated_by_id, created_at, updated_at, deleted_at
FROM workspace_member_invites
WHERE workspace_id = sqlc.arg(workspace_id) AND deleted_at IS NULL
ORDER BY created_at DESC, id;

````

- [ ] **Step 2: 生成**

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `7423feed5b0fa41b36ac3f28f2249921a3cef6377cc5958f4fc6207f0e6ede64` | 1435 | `api/dist/openapi.yaml` |
| `bbba87a0d9586cf70a2d11ec428f36fa9bb8a94555490c76976e105e349242b8` | 1641 | `server/internal/modules/workspace/adapter/http/gen/server.gen.go` |
| `48527700a9d860cb679dbf0ebcd333dcf4c491bf27c3a12d84ae94cca3abe869` | 117 | `server/internal/modules/workspace/adapter/postgres/gen/invitations.sql.go` |
| `d7aea09c2e7d1c0c8a6ec9a80fa10710f77f55bc77db143bd47ee49c7c4fe484` | 1515 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/workspace/adapter/http/gen/server.gen.go server/internal/modules/workspace/adapter/postgres/gen/invitations.sql.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 3: 领域、端口、令牌、用例**

`server/internal/modules/workspace/domain/invitation.go`（修改，1 处）：

````old server/internal/modules/workspace/domain/invitation.go
// Responded reports whether the invitation has been answered: declined,
// for an undeleted one (M3 design 3.8).
func (i Invitation) Responded() bool {
	return i.RespondedAt != nil
````

````new server/internal/modules/workspace/domain/invitation.go
// InvitationWithToken is an invitation and the token of its link, which
// only who may manage the workspace's invitations is given (M3 design 5.2).
type InvitationWithToken struct {
	Invitation
	Token string
````

`server/internal/modules/workspace/domain/actions.go`（修改，2 处）：

````old server/internal/modules/workspace/domain/actions.go
	ActionPreferencesUpdate shared.Action = "workspace_preferences.update"
````

````new server/internal/modules/workspace/domain/actions.go
	ActionPreferencesUpdate shared.Action = "workspace_preferences.update"
	// ActionInvitationList is listing a workspace's invitations, with their
	// tokens: listWorkspaceInvitations.
	ActionInvitationList shared.Action = "workspace_invitation.list"
````

````old server/internal/modules/workspace/domain/actions.go
	return []shared.Action{ActionRead, ActionUpdate, ActionDelete, ActionMemberList, ActionMemberUpdate, ActionPreferencesRead, ActionPreferencesUpdate}
````

````new server/internal/modules/workspace/domain/actions.go
	return []shared.Action{ActionRead, ActionUpdate, ActionDelete, ActionMemberList, ActionMemberUpdate, ActionPreferencesRead, ActionPreferencesUpdate,
		ActionInvitationList}
````

`server/internal/modules/workspace/app/invitation_ports.go`（修改，3 处）：

````old server/internal/modules/workspace/app/invitation_ports.go
import (
````

````new server/internal/modules/workspace/app/invitation_ports.go
import (
	"context"
````

````old server/internal/modules/workspace/app/invitation_ports.go
	"uuid"

````

````new server/internal/modules/workspace/app/invitation_ports.go
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
````

````old server/internal/modules/workspace/app/invitation_ports.go
// The ports of the invitations' use cases (M3 design 3.8, 6.5).
````

````new server/internal/modules/workspace/app/invitation_ports.go
// The ports of the invitations' use cases (M3 design 3.8, 6.5).

// InvitationMAC tags the messages of the invitations' tokens: identity's MAC
// of the purpose "workspace-invitation", whose key is derived from the
// signing key (M3 design 3.8, 6.5, 11.1). Verify compares in constant time.
type InvitationMAC interface {
	Tag(message []byte) [16]byte
	Verify(message []byte, tag [16]byte) bool
}

// InvitationLister reads a workspace and lists its invitations.
type InvitationLister interface {
	WorkspaceFinder
	// ListInvitations returns the workspace's undeleted invitations,
	// pending or declined, newest first, then by id (M3 design 3.12).
	ListInvitations(ctx context.Context, workspaceID uuid.UUID) ([]domain.Invitation, error)
}
````

`server/internal/modules/workspace/app/tokens.go`（新文件，28 行）：

````file server/internal/modules/workspace/app/tokens.go
package app

import (
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
)

// invitationTokens computes and checks the invitations' tokens with the
// invitation MAC (M3 design 3.8). The server stores no token: it computes
// one again from the invitation's id.
type invitationTokens struct {
	mac InvitationMAC
}

// of is the token of the invitation id.
func (t invitationTokens) of(id uuid.UUID) string {
	return domain.FormatToken(t.mac.Tag(domain.InvitationMessage(id)))
}

// withTokens is invitations, each with its token.
func (t invitationTokens) withTokens(invitations []domain.Invitation) []domain.InvitationWithToken {
	out := make([]domain.InvitationWithToken, len(invitations))
	for i, inv := range invitations {
		out[i] = domain.InvitationWithToken{Invitation: inv, Token: t.of(inv.ID)}
	}
	return out
}
````

`server/internal/modules/workspace/app/list_invitations.go`（新文件，49 行）：

````file server/internal/modules/workspace/app/list_invitations.go
package app

import (
	"context"
	"errors"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListWorkspaceInvitations lists a workspace's invitations: GET
// /api/v0/workspaces/{slug}/invitations.
type ListWorkspaceInvitations struct {
	invitations InvitationLister
	auth        shared.Authorizer
	tokens      invitationTokens
}

// NewListWorkspaceInvitations returns the use case.
func NewListWorkspaceInvitations(invitations InvitationLister, auth shared.Authorizer, mac InvitationMAC) *ListWorkspaceInvitations {
	return &ListWorkspaceInvitations{invitations: invitations, auth: auth, tokens: invitationTokens{mac: mac}}
}

// Execute returns the workspace's undeleted invitations, pending or
// declined, in the store's order, each with its token, which the caller
// needs to hand out the link (M3 design 3.8). Only the workspace's admins
// may list them (M3 decision 4). A read opens no transaction and decides
// directly (M3 design 3.4).
func (u *ListWorkspaceInvitations) Execute(ctx context.Context, slug string) ([]domain.InvitationWithToken, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	w, err := u.invitations.WorkspaceBySlug(ctx, slug)
	switch {
	case errors.Is(err, ErrNotFound):
		return nil, domain.ErrNotFound
	case err != nil:
		return nil, err
	}
	if _, err := decide(ctx, u.auth, actor, domain.ActionInvitationList, w.ID, domain.ErrNotFound); err != nil {
		return nil, err
	}
	invitations, err := u.invitations.ListInvitations(ctx, w.ID)
	if err != nil {
		return nil, err
	}
	return u.tokens.withTokens(invitations), nil
}
````

`server/internal/modules/workspace/app/fakes_invitations_test.go`（新文件，61 行）：

````file server/internal/modules/workspace/app/fakes_invitations_test.go
package app_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
)

// fakeMAC tags a message with the first 16 bytes of HMAC-SHA256 under a key
// of its own, as identity's does, and logs each Verify with the invitation
// id its message names. Two fakeMACs of other keys tag differently.
type fakeMAC struct {
	log *callLog
	key string
}

func (m fakeMAC) Tag(message []byte) [16]byte {
	h := hmac.New(sha256.New, []byte(m.key))
	h.Write(message)
	return [16]byte(h.Sum(nil)[:16])
}

func (m fakeMAC) Verify(message []byte, tag [16]byte) bool {
	id := uuid.UUID(message[len(message)-16:])
	m.log.calls = append(m.log.calls, "Verify "+id.String())
	want := m.Tag(message)
	return hmac.Equal(want[:], tag[:])
}

// tokenOf is the token of the invitation id under mac.
func tokenOf(mac fakeMAC, id uuid.UUID) string {
	return domain.FormatToken(mac.Tag(domain.InvitationMessage(id)))
}

// fakeInvitations is the invitations' repositories over fakeWorkspaces,
// which answers the workspaces and their locks: it logs every call with its
// arguments, answers from the undeleted invitations it holds, and fails a
// call with the error set for it, wrapped as the store wraps it.
type fakeInvitations struct {
	*fakeWorkspaces
	invitations []domain.Invitation
	listErr     error // for ListInvitations
}

func (f *fakeInvitations) ListInvitations(ctx context.Context, workspaceID uuid.UUID) ([]domain.Invitation, error) {
	f.log.add(ctx, "ListInvitations %s", workspaceID)
	if f.listErr != nil {
		return nil, fmt.Errorf("list workspace invitations: %w", f.listErr)
	}
	var out []domain.Invitation
	for _, inv := range f.invitations {
		if inv.WorkspaceID == workspaceID {
			out = append(out, inv)
		}
	}
	return out, nil
}
````

`server/internal/modules/workspace/app/list_invitations_test.go`（新文件，151 行）：

````file server/internal/modules/workspace/app/list_invitations_test.go
package app_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

var (
	declined = now.Add(time.Hour)
	// acme's invitations: carol's pending, dave's declined; beta's: erin's.
	carolToAcme = domain.Invitation{ID: uuid.NewV7(), WorkspaceID: acme.ID, Email: "carol@corp.com", Role: shared.RoleMember, CreatedAt: now,
		CreatedByID: &alice.ID}
	daveToAcme = domain.Invitation{ID: uuid.NewV7(), WorkspaceID: acme.ID, Email: "dave@corp.com", Role: shared.RoleGuest, RespondedAt: &declined,
		CreatedAt: now}
	erinToBeta = domain.Invitation{ID: uuid.NewV7(), WorkspaceID: beta.ID, Email: "erin@corp.com", Role: shared.RoleAdmin, CreatedAt: now,
		CreatedByID: &bob.ID}
)

// invitationsFixture is the invitations' use cases over fakes sharing one
// log: acme and beta with carolToAcme, daveToAcme and erinToBeta; alice is
// acme's admin, bob acme's member and beta's admin.
type invitationsFixture struct {
	log         *callLog
	tx          *fakeTx
	invitations *fakeInvitations
	auth        *fakeAuthorizer
	mac         fakeMAC
}

func newInvitations() *invitationsFixture {
	log := &callLog{}
	return &invitationsFixture{log: log, tx: &fakeTx{},
		invitations: &fakeInvitations{fakeWorkspaces: &fakeWorkspaces{log: log, workspaces: []domain.Workspace{acme, beta}},
			invitations: []domain.Invitation{carolToAcme, daveToAcme, erinToBeta}},
		auth: &fakeAuthorizer{log: log, grants: map[grantKey]shared.Grant{
			{alice.ID, acme.ID}: {WorkspaceRole: shared.RoleAdmin},
			{bob.ID, acme.ID}:   {WorkspaceRole: shared.RoleMember},
			{bob.ID, beta.ID}:   {WorkspaceRole: shared.RoleAdmin},
		}}, mac: fakeMAC{log: log, key: "the instance's key"}}
}

// withToken is inv with its token under mac.
func withToken(mac fakeMAC, inv domain.Invitation) domain.InvitationWithToken {
	return domain.InvitationWithToken{Invitation: inv, Token: tokenOf(mac, inv.ID)}
}

// sameInvitations compares two lists, the pointers by value.
func sameInvitations(a, b []domain.InvitationWithToken) bool {
	return slices.EqualFunc(a, b, func(x, y domain.InvitationWithToken) bool {
		rx, ry, cx, cy := x.RespondedAt, y.RespondedAt, x.CreatedByID, y.CreatedByID
		x.RespondedAt, y.RespondedAt, x.CreatedByID, y.CreatedByID = nil, nil, nil, nil
		return x == y && (rx == nil) == (ry == nil) && (rx == nil || rx.Equal(*ry)) && (cx == nil) == (cy == nil) && (cx == nil || *cx == *cy)
	})
}

// The list is the workspace's invitations in the store's order, each with
// the token the MAC gives its id, read without a transaction after the
// decision on workspace_invitation.list for the caller. Two admins, two
// workspaces.
func TestListWorkspaceInvitationsListsEachWithItsToken(t *testing.T) {
	for _, tt := range []struct {
		user app.AccountState
		w    domain.Workspace
		want []domain.Invitation
	}{{alice, acme, []domain.Invitation{carolToAcme, daveToAcme}}, {bob, beta, []domain.Invitation{erinToBeta}}} {
		f := newInvitations()
		got, err := app.NewListWorkspaceInvitations(f.invitations, f.auth, f.mac).Execute(as(tt.user), tt.w.Slug)
		var want []domain.InvitationWithToken
		for _, inv := range tt.want {
			want = append(want, withToken(f.mac, inv))
		}
		if err != nil || !sameInvitations(got, want) {
			t.Errorf("%s lists %s: %+v, %v; want %+v", tt.user.Email, tt.w.Slug, got, err, want)
		}
		wantCalls := []string{"WorkspaceBySlug " + tt.w.Slug + " outside tx",
			"Authorize " + tt.user.ID.String() + " workspace_invitation.list on " + tt.w.ID.String() + "/" + uuid.Nil().String() + " outside tx",
			"ListInvitations " + tt.w.ID.String() + " outside tx"}
		if !slices.Equal(f.log.calls, wantCalls) {
			t.Errorf("%s lists %s: calls = %q, want %q", tt.user.Email, tt.w.Slug, f.log.calls, wantCalls)
		}
	}
}

// Another key gives other tokens: the token is the MAC's, of the
// invitation's id alone.
func TestTheTokensAreTheMACs(t *testing.T) {
	f := newInvitations()
	other := fakeMAC{log: f.log, key: "another instance's key"}
	got, err := app.NewListWorkspaceInvitations(f.invitations, f.auth, other).Execute(as(alice), "acme")
	if err != nil || len(got) != 2 || got[0].Token != tokenOf(other, carolToAcme.ID) || got[0].Token == tokenOf(f.mac, carolToAcme.ID) ||
		got[1].Token != tokenOf(other, daveToAcme.ID) {
		t.Errorf("Execute() under another key = %+v, %v; want its tokens", got, err)
	}
}

// Each refusal and failure is the answer, and no invitation is read after
// it: a workspace not there or not visible is workspace.not_found, the
// Authorizer's forbidden comes before the list. A failure is itself, never a
// 404; without a caller it is 401 and nothing is read.
func TestListWorkspaceInvitationsRefusals(t *testing.T) {
	failure := errors.New("connection reset")
	decided := func(user app.AccountState) []string {
		return []string{"WorkspaceBySlug acme outside tx",
			"Authorize " + user.ID.String() + " workspace_invitation.list on " + acme.ID.String() + "/" + uuid.Nil().String() + " outside tx"}
	}
	tests := []struct {
		name  string
		user  app.AccountState
		slug  string
		set   func(f *invitationsFixture)
		want  error
		calls []string
	}{
		{"no such workspace", alice, "nothing", nil, domain.ErrNotFound, []string{"WorkspaceBySlug nothing outside tx"}},
		{"not visible", carol, "acme", nil, domain.ErrNotFound, decided(carol)},
		{"forbidden", bob, "acme", func(f *invitationsFixture) { f.auth.errs = map[grantKey]error{{bob.ID, acme.ID}: shared.Forbidden()} },
			shared.Forbidden(), decided(bob)},
		{"the workspace's read failed", alice, "acme", func(f *invitationsFixture) { f.invitations.slugErrs = map[string]error{"acme": failure} },
			failure, []string{"WorkspaceBySlug acme outside tx"}},
		{"the Authorizer failed", alice, "acme", func(f *invitationsFixture) { f.auth.errs = map[grantKey]error{{alice.ID, acme.ID}: failure} },
			failure, decided(alice)},
		{"the list failed", alice, "acme", func(f *invitationsFixture) { f.invitations.listErr = failure }, failure,
			append(decided(alice), "ListInvitations "+acme.ID.String()+" outside tx")},
	}
	for _, tt := range tests {
		f := newInvitations()
		if tt.set != nil {
			tt.set(f)
		}
		got, err := app.NewListWorkspaceInvitations(f.invitations, f.auth, f.mac).Execute(as(tt.user), tt.slug)
		if !errors.Is(err, tt.want) || got != nil || (tt.want == failure && errors.Is(err, domain.ErrNotFound)) {
			t.Errorf("%s: Execute() = %+v, %v; want %v", tt.name, got, err, tt.want)
		}
		if !slices.Equal(f.log.calls, tt.calls) {
			t.Errorf("%s: calls = %q, want %q", tt.name, f.log.calls, tt.calls)
		}
	}
	f := newInvitations()
	if _, err := app.NewListWorkspaceInvitations(f.invitations, f.auth, f.mac).Execute(context.Background(), "acme"); !errors.Is(err, shared.Unauthenticated()) ||
		len(f.log.calls) != 0 {
		t.Errorf("Execute() without an actor = %v, calls %q; want 401 unauthorized and nothing", err, f.log.calls)
	}
}
````

- [ ] **Step 4: 存储**

`server/internal/modules/workspace/adapter/postgres/invitations.go`（修改，1 处）：

````old server/internal/modules/workspace/adapter/postgres/invitations.go
}

// DeleteWorkspaceInvitations soft-deletes the undeleted invitations of the
````

````new server/internal/modules/workspace/adapter/postgres/invitations.go
}

// ListInvitations returns the workspace's undeleted invitations, pending or
// declined, newest first, then by id.
func (s *Store) ListInvitations(ctx context.Context, workspaceID uuid.UUID) ([]domain.Invitation, error) {
	rows, err := s.queries(ctx).ListInvitations(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list workspace invitations: %w", err)
	}
	out := make([]domain.Invitation, len(rows))
	for i, r := range rows {
		out[i] = invitation(r)
	}
	return out, nil
}

// DeleteWorkspaceInvitations soft-deletes the undeleted invitations of the
````

`server/internal/modules/workspace/adapter/postgres/invitations_test.go`（修改，2 处）：

````old server/internal/modules/workspace/adapter/postgres/invitations_test.go
import (
````

````new server/internal/modules/workspace/adapter/postgres/invitations_test.go
import (
	"bytes"
````

````old server/internal/modules/workspace/adapter/postgres/invitations_test.go
		t.Errorf("acme's invitations are %q, want %q", pendingEmails(t, pool, acme.ID), want)
	}
}

````

````new server/internal/modules/workspace/adapter/postgres/invitations_test.go
		t.Errorf("acme's invitations are %q, want %q", pendingEmails(t, pool, acme.ID), want)
	}
}

// ListInvitations is the workspace's undeleted invitations, pending and
// declined, newest first, then by id: an accepted or deleted invitation and
// another workspace's are not in it.
func TestListInvitations(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme, beta := newWorkspace(t, s, "Acme", "acme", alice), newWorkspace(t, s, "Beta", "beta", alice)
	at := func(email string, created time.Time) domain.Invitation {
		t.Helper()
		got, err := s.CreateInvitations(context.Background(), []app.InvitationRow{
			{ID: uuid.NewV7(), WorkspaceID: acme.ID, Email: email, Role: shared.RoleMember, CreatedBy: alice, Now: created},
		})
		if err != nil {
			t.Fatal(err)
		}
		return got[0]
	}
	oldest := at("carol@corp.com", now.Add(-time.Hour))
	declined := at("dave@corp.com", now)
	sameTime := at("erin@corp.com", now)
	newest := at("frank@corp.com", now.Add(time.Hour))
	exec(t, pool, "UPDATE workspace_member_invites SET responded_at = $1 WHERE id = $2", now, declined.ID)
	declined.RespondedAt = &now
	for _, gone := range []string{"UPDATE workspace_member_invites SET accepted = true, responded_at = $1, deleted_at = $1 WHERE id = $2",
		"UPDATE workspace_member_invites SET deleted_at = $1 WHERE id = $2"} {
		exec(t, pool, gone, now, at("gina+"+uuid.NewV7().String()+"@corp.com", now.Add(2*time.Hour)).ID)
	}
	invite(t, s, beta.ID, "carol@corp.com", shared.RoleMember, alice)

	got, err := s.ListInvitations(context.Background(), acme.ID)

	sameTimes := []domain.Invitation{declined, sameTime}
	slices.SortFunc(sameTimes, func(a, b domain.Invitation) int { return bytes.Compare(a.ID[:], b.ID[:]) })
	want := append([]domain.Invitation{newest}, append(sameTimes, oldest)...)
	if err != nil || !slices.EqualFunc(got, want, sameInvitation) {
		t.Errorf("ListInvitations() = %+v, %v; want %+v", got, err, want)
	}
}

````

`server/internal/modules/workspace/adapter/postgres/failures_test.go`（修改，1 处）：

````old server/internal/modules/workspace/adapter/postgres/failures_test.go
		t.Errorf("MemberByID() = %+v, %v; want context.Canceled, not app.ErrNotFound", got, err)
	}
````

````new server/internal/modules/workspace/adapter/postgres/failures_test.go
		t.Errorf("MemberByID() = %+v, %v; want context.Canceled, not app.ErrNotFound", got, err)
	}
	invite(t, s, w.ID, "carol@corp.com", shared.RoleGuest, alice)
	if list, err := s.ListInvitations(cancelled, w.ID); !failed(err) || list != nil {
		t.Errorf("ListInvitations() = %v, %v; want context.Canceled, no list", list, err)
	}
````

- [ ] **Step 5: HTTP**

`server/internal/modules/workspace/adapter/http/handler.go`（修改，2 处）：

````old server/internal/modules/workspace/adapter/http/handler.go
}

// CheckSlugUseCase is app.CheckSlug.
````

````new server/internal/modules/workspace/adapter/http/handler.go
}

// ListInvitationsUseCase is app.ListWorkspaceInvitations.
type ListInvitationsUseCase interface {
	Execute(ctx context.Context, slug string) ([]domain.InvitationWithToken, error)
}

// CheckSlugUseCase is app.CheckSlug.
````

````old server/internal/modules/workspace/adapter/http/handler.go
	UpdatePreferences UpdatePreferencesUseCase
````

````new server/internal/modules/workspace/adapter/http/handler.go
	UpdatePreferences UpdatePreferencesUseCase
	ListInvitations   ListInvitationsUseCase
````

`server/internal/modules/workspace/adapter/http/invitations.go`（新文件，54 行）：

````file server/internal/modules/workspace/adapter/http/invitations.go
package httpadapter

import (
	"context"
	"time"
	"uuid"

	"github.com/oapi-codegen/nullable"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
)

// ListWorkspaceInvitations serves GET /api/v0/workspaces/{slug}/invitations.
func (h handler) ListWorkspaceInvitations(ctx context.Context, req gen.ListWorkspaceInvitationsRequestObject) (gen.ListWorkspaceInvitationsResponseObject, error) {
	list, err := h.uc.ListInvitations.Execute(ctx, req.Slug)
	if err != nil {
		return nil, err
	}
	return gen.ListWorkspaceInvitations200JSONResponse{Data: invitations(list)}, nil
}

// invitations is list as the API shows it.
func invitations(list []domain.InvitationWithToken) []gen.WorkspaceInvitation {
	out := make([]gen.WorkspaceInvitation, len(list))
	for i, inv := range list {
		out[i] = invitation(inv)
	}
	return out
}

// invitation is inv as the API shows it. Its nullable fields are required:
// the zero Nullable is "unspecified", so null is set explicitly.
func invitation(inv domain.InvitationWithToken) gen.WorkspaceInvitation {
	respondedAt := nullable.NewNullNullable[time.Time]()
	if inv.RespondedAt != nil {
		respondedAt = nullable.NewNullableWithValue(*inv.RespondedAt)
	}
	createdBy := nullable.NewNullNullable[uuid.UUID]()
	if inv.CreatedByID != nil {
		createdBy = nullable.NewNullableWithValue(*inv.CreatedByID)
	}
	return gen.WorkspaceInvitation{
		ID:          inv.ID,
		WorkspaceID: inv.WorkspaceID,
		Email:       inv.Email,
		Role:        gen.WorkspaceRole(inv.Role),
		Accepted:    inv.Accepted,
		RespondedAt: respondedAt,
		CreatedAt:   inv.CreatedAt,
		CreatedByID: createdBy,
		Token:       inv.Token,
	}
}
````

`server/internal/modules/workspace/adapter/http/handler_test.go`（修改，3 处）：

````old server/internal/modules/workspace/adapter/http/handler_test.go
	prefs  *fakePrefs
````

````new server/internal/modules/workspace/adapter/http/handler_test.go
	prefs  *fakePrefs
	// invitations are the invitations' use cases (invitations_test.go).
	invitations *fakeInvitations
````

````old server/internal/modules/workspace/adapter/http/handler_test.go
		f.prefs = &fakePrefs{}
	}
````

````new server/internal/modules/workspace/adapter/http/handler_test.go
		f.prefs = &fakePrefs{}
	}
	if f.invitations == nil {
		f.invitations = &fakeInvitations{}
	}
````

````old server/internal/modules/workspace/adapter/http/handler_test.go
		ListMembers: f.member, UpdateMember: f.role, GetPreferences: fakeGetPrefs{f.prefs}, UpdatePreferences: fakeUpdatePrefs{f.prefs},
````

````new server/internal/modules/workspace/adapter/http/handler_test.go
		ListMembers: f.member, UpdateMember: f.role, GetPreferences: fakeGetPrefs{f.prefs}, UpdatePreferences: fakeUpdatePrefs{f.prefs},
		ListInvitations: fakeListInvitations{f.invitations},
````

`server/internal/modules/workspace/adapter/http/invitations_test.go`（新文件，88 行）：

````file server/internal/modules/workspace/adapter/http/invitations_test.go
package httpadapter_test

import (
	"context"
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeInvitations is the invitations' use cases: each records its call as
// "<operation> <caller> <arguments>" and answers what it is given.
type fakeInvitations struct {
	calls []string
	lists map[string][]domain.InvitationWithToken // by slug
	err   error
}

type fakeListInvitations struct{ *fakeInvitations }

func (f fakeListInvitations) Execute(ctx context.Context, slug string) ([]domain.InvitationWithToken, error) {
	f.calls = append(f.calls, "list "+caller(ctx)+" "+slug)
	return f.lists[slug], f.err
}

var (
	declinedAt = time.Date(2026, 9, 30, 8, 0, 0, 0, time.UTC)
	// carolInvited is pending, alice's; daveDeclined declined, its inviter
	// gone.
	carolInvited = domain.InvitationWithToken{Invitation: domain.Invitation{ID: uuid.MustParse("0199a2b4-0000-7000-8000-0000000000c1"),
		WorkspaceID: acmeID, Email: "carol@corp.com", Role: shared.RoleMember, CreatedAt: created, CreatedByID: &aliceID},
		Token: "nrv_inv_kvqyKBh-bANMT6JAIYzolA"}
	daveDeclined = domain.InvitationWithToken{Invitation: domain.Invitation{ID: uuid.MustParse("0199a2b4-0000-7000-8000-0000000000d1"),
		WorkspaceID: acmeID, Email: "dave@corp.com", Role: shared.RoleGuest, RespondedAt: &declinedAt, CreatedAt: created},
		Token: "nrv_inv_AAAAAAAAAAAAAAAAAAAAAA"}
)

const (
	carolInvitedJSON = `{"accepted":false,"created_at":"2026-09-29T10:00:00.123456Z","created_by_id":"0199a2b4-0000-7000-8000-000000000001",` +
		`"email":"carol@corp.com","id":"0199a2b4-0000-7000-8000-0000000000c1","responded_at":null,"role":15,` +
		`"token":"nrv_inv_kvqyKBh-bANMT6JAIYzolA","workspace_id":"0199a2b4-0000-7000-8000-00000000000a"}`
	daveDeclinedJSON = `{"accepted":false,"created_at":"2026-09-29T10:00:00.123456Z","created_by_id":null,"email":"dave@corp.com",` +
		`"id":"0199a2b4-0000-7000-8000-0000000000d1","responded_at":"2026-09-30T08:00:00Z","role":5,` +
		`"token":"nrv_inv_AAAAAAAAAAAAAAAAAAAAAA","workspace_id":"0199a2b4-0000-7000-8000-00000000000a"}`
)

// The list is the use case's for the caller and the slug of the path, each
// invitation with its token: a pending one and a declined one whose inviter
// is gone.
func TestListWorkspaceInvitations(t *testing.T) {
	inv := &fakeInvitations{lists: map[string][]domain.InvitationWithToken{"acme": {carolInvited, daveDeclined}}}
	h := newServer(t, fakes{invitations: inv})
	for _, tt := range []struct{ token, slug, want string }{
		{"alice", "acme", `{"data":[` + carolInvitedJSON + `,` + daveDeclinedJSON + `]}`},
		{"bob", "beta", `{"data":[]}`},
	} {
		res, body := do(t, h, request(http.MethodGet, "/api/v0/workspaces/"+tt.slug+"/invitations", tt.token, ""))
		if res.StatusCode != http.StatusOK || body != tt.want+"\n" {
			t.Errorf("%s GET %s's invitations = %d %s, want 200 %s", tt.token, tt.slug, res.StatusCode, body, tt.want)
		}
	}
	if want := []string{"list alice acme", "list bob beta"}; !slices.Equal(inv.calls, want) {
		t.Errorf("calls = %q, want %q", inv.calls, want)
	}
}

// The use case's refusals, as the contract declares them.
func TestListWorkspaceInvitationsRefusals(t *testing.T) {
	for _, tt := range []struct {
		err    error
		status int
		want   string
	}{
		{domain.ErrNotFound, http.StatusNotFound,
			`{"status":404,"code":"workspace.not_found","title":"Not Found","detail":"The workspace does not exist, or you are not a member of it."}`},
		{shared.Forbidden(), http.StatusForbidden, `{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
	} {
		h := newServer(t, fakes{invitations: &fakeInvitations{err: tt.err}})
		if res, body := do(t, h, request(http.MethodGet, "/api/v0/workspaces/acme/invitations", "bob", "")); res.StatusCode != tt.status ||
			body != tt.want+"\n" {
			t.Errorf("GET refused with %v = %d %s, want %d %s", tt.err, res.StatusCode, body, tt.status, tt.want)
		}
	}
}
````

- [ ] **Step 6: 规则、接线**

`server/internal/modules/access/domain/rules.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules.go
	"workspace_preferences.update": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
````

````new server/internal/modules/access/domain/rules.go
	"workspace_preferences.update": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	// The invitations and their tokens: the workspace's admins alone (M3
	// decision 4).
	"workspace_invitation.list": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin}},
````

`server/internal/modules/access/domain/rules_test.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules_test.go
	"workspace_preferences.update": {allowed, allowed, allowed, invisible, invisible, invisible, forbidden},
````

````new server/internal/modules/access/domain/rules_test.go
	"workspace_preferences.update": {allowed, allowed, allowed, invisible, invisible, invisible, forbidden},
	"workspace_invitation.list":    {allowed, forbidden, forbidden, invisible, invisible, invisible, forbidden},
````

`server/internal/modules/workspace/module.go`（修改，4 处）：

````old server/internal/modules/workspace/module.go
// workspaces and their members. It brings creating, listing, reading,
// changing and deleting workspaces, checking a slug, listing the members
// and changing their roles, and each member's display settings, and offers
// the other modules its reads through ports.
````

````new server/internal/modules/workspace/module.go
// workspaces, their members and their invitations. It brings creating,
// listing, reading, changing and deleting workspaces, checking a slug,
// listing the members and changing their roles, each member's display
// settings, and the invitations, and offers the other modules its reads
// through ports.
````

````old server/internal/modules/workspace/module.go
type PublicProfile = app.PublicProfile

````

````new server/internal/modules/workspace/module.go
type PublicProfile = app.PublicProfile

// InvitationMACPurpose is the purpose of the invitation MAC that bootstrap
// asks identity's keys for: its key's HKDF info is "nerve
// workspace-invitation mac v1" (M3 design 3.8).
const InvitationMACPurpose = "workspace-invitation"

````

````old server/internal/modules/workspace/module.go
	Profiles app.MemberProfiles
````

````new server/internal/modules/workspace/module.go
	Profiles app.MemberProfiles
	// InvitationMAC is identity's MAC of InvitationMACPurpose.
	InvitationMAC app.InvitationMAC
````

````old server/internal/modules/workspace/module.go
		UpdatePreferences: app.NewUpdateWorkspacePreferences(store, d.Authorizer, d.Tx, d.Clock),
````

````new server/internal/modules/workspace/module.go
		UpdatePreferences: app.NewUpdateWorkspacePreferences(store, d.Authorizer, d.Tx, d.Clock),
		ListInvitations:   app.NewListWorkspaceInvitations(store, d.Authorizer, d.InvitationMAC),
````

`server/internal/bootstrap/app.go`（修改，2 处）：

````old server/internal/bootstrap/app.go
	// The signing key first, before any module (M3 design 6.6 step 1).
	keys, err := identity.LoadKeys(signingKey, logger)
````

````new server/internal/bootstrap/app.go
	// The signing key first, before any module, and the MACs derived from
	// it for the other modules (M3 design 6.6 step 1).
	keys, err := identity.LoadKeys(signingKey, logger)
	if err != nil {
		return nil, err
	}
	invitationMAC, err := keys.MAC(workspace.InvitationMACPurpose)
````

````old server/internal/bootstrap/app.go
		Profiles:        workspaceProfiles{profiles: identityPorts.PublicProfiles},
````

````new server/internal/bootstrap/app.go
		Profiles:        workspaceProfiles{profiles: identityPorts.PublicProfiles},
		InvitationMAC:   invitationMAC,
````

`server/internal/bootstrap/invitation_mac_test.go`（新文件，32 行）：

````file server/internal/bootstrap/invitation_mac_test.go
package bootstrap

import (
	"log/slog"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
)

// bootstrap derives the invitation MAC from the signing key for
// workspace.InvitationMACPurpose (M3 design 3.8, 6.6). Its token of the id
// below is the known answer computed by hand from the design's derivation,
// HKDF info "nerve workspace-invitation mac v1" (spec P3, appendix): the
// purpose is part of the key, and a purpose other than the design's would
// void every link made before it.
func TestTheInvitationMACIsTheDesigns(t *testing.T) {
	keys, err := identity.LoadKeys([]byte(testKeyPEM), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	mac, err := keys.MAC(workspace.InvitationMACPurpose)
	if err != nil {
		t.Fatal(err)
	}
	id := uuid.MustParse("0199a2b4-0000-7000-8000-000000000001")
	if got := domain.FormatToken(mac.Tag(domain.InvitationMessage(id))); got != "nrv_inv_kvqyKBh-bANMT6JAIYzolA" {
		t.Errorf("the token of %s = %s, want nrv_inv_kvqyKBh-bANMT6JAIYzolA", id, got)
	}
}
````

- [ ] **Step 7: 矩阵**

`server/internal/bootstrap/permission_matrix_test.go`（修改，3 处）：

````old server/internal/bootstrap/permission_matrix_test.go
// store, the workspaces and memberships of matrixMemberships, with the ids
// newSeeded named, and acme's admin's display settings; other's admin and
// removed member are there so that a role read in the wrong workspace lets
// either into acme. Through the API, gone deleted by its admin, which
````

````new server/internal/bootstrap/permission_matrix_test.go
// store, the workspaces, memberships and invitations of matrixMemberships
// and matrixInvitations, with the ids newSeeded named, and acme's admin's
// display settings; other's admin and removed member are there so that a
// role read in the wrong workspace lets either into acme. Through the API, gone deleted by its admin, which
````

````old server/internal/bootstrap/permission_matrix_test.go
			seed.join(s.membership(m.slug, m.c), m.slug, m.c, m.role)
		}
````

````new server/internal/bootstrap/permission_matrix_test.go
			seed.join(s.membership(m.slug, m.c), m.slug, m.c, m.role)
		}
		for _, i := range matrixInvitations {
			seed.invite(s.invitation(i.slug, i.email), i.slug, i.email, i.role)
		}
````

````old server/internal/bootstrap/permission_matrix_test.go
		// membership goes with the workspace row, so every cell of the column
		// is asked about a workspace deleted the one way there is. A
		// membership left active in a deleted workspace is ActiveRole's
		// store test (P1).
````

````new server/internal/bootstrap/permission_matrix_test.go
		// membership and invitations go with the workspace row, so every
		// cell of the column is asked about a workspace deleted the one way
		// there is. A membership left active in a deleted workspace is
		// ActiveRole's store test (P1).
````

`server/internal/bootstrap/permission_matrix_seeded_test.go`（修改，8 处）：

````old server/internal/bootstrap/permission_matrix_seeded_test.go
}

// seeded are the ids of the rows prepareMatrix seeds that a request can
// name: each membership, by the workspace's slug and the column. t is the
// test that asks for them (in).
````

````new server/internal/bootstrap/permission_matrix_seeded_test.go
}

// matrixInvitations are the invitations prepareMatrix seeds, each sent by
// its workspace's admin: in acme and in gone, one to an address no account
// has. gone's are deleted with it.
var matrixInvitations = []struct {
	slug, email string
	role        shared.Role
}{
	{"acme", "newcomer@example.com", shared.RoleMember},
	{"gone", "newcomer@example.com", shared.RoleMember},
}

// seeded are the ids of the rows prepareMatrix seeds that a request can
// name: each membership, by the workspace's slug and the column, and each
// invitation, by the workspace's slug and the address. t is the test that
// asks for them (in).
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
	memberships map[string]uuid.UUID
````

````new server/internal/bootstrap/permission_matrix_seeded_test.go
	memberships map[string]uuid.UUID
	invitations map[string]uuid.UUID
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
// newSeeded names an id for each of matrixMemberships before prepareMatrix
// writes them, so that matrixViolations, without a database, sees the keys
// and the workspaces the cells will.
````

````new server/internal/bootstrap/permission_matrix_seeded_test.go
// newSeeded names an id for each of matrixMemberships and
// matrixInvitations before prepareMatrix writes them, so that
// matrixViolations, without a database, sees the keys and the workspaces
// the cells will.
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
	s := seeded{memberships: map[string]uuid.UUID{}}
````

````new server/internal/bootstrap/permission_matrix_seeded_test.go
	s := seeded{memberships: map[string]uuid.UUID{}, invitations: map[string]uuid.UUID{}}
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
		s.memberships[m.slug+"/"+string(m.c)] = uuid.NewV7()
````

````new server/internal/bootstrap/permission_matrix_seeded_test.go
		s.memberships[m.slug+"/"+string(m.c)] = uuid.NewV7()
	}
	for _, i := range matrixInvitations {
		s.invitations[i.slug+"/"+i.email] = uuid.NewV7()
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
}

// workspaceOfRow is the slug of the workspace the seeded row id is under,
````

````new server/internal/bootstrap/permission_matrix_seeded_test.go
}

// invitation is the id of the invitation of email to the workspace slug. A
// key that was never seeded fails the test at once, as membership's does.
func (s seeded) invitation(slug, email string) uuid.UUID {
	id, ok := s.invitations[slug+"/"+email]
	if !ok {
		s.t.Helper()
		s.t.Fatalf("no invitation of %s to %s is seeded", email, slug)
	}
	return id
}

// workspaceOfRow is the slug of the workspace the seeded row id is under,
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
	for key, seededID := range s.memberships {
		if seededID == id {
			slug, _, _ := strings.Cut(key, "/")
			return slug, true
````

````new server/internal/bootstrap/permission_matrix_seeded_test.go
	for _, rows := range []map[string]uuid.UUID{s.memberships, s.invitations} {
		for key, seededID := range rows {
			if seededID == id {
				slug, _, _ := strings.Cut(key, "/")
				return slug, true
			}
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
}

func (s matrixSeed) exec(pool *pgxpool.Pool, sql string, args ...any) {
````

````new server/internal/bootstrap/permission_matrix_seeded_test.go
}

// invite stores the invitation id of email to the workspace slug, by its
// admin.
func (s matrixSeed) invite(id uuid.UUID, slug, email string, role shared.Role) {
	s.t.Helper()
	admin := callerAdmin
	if slug == "gone" {
		admin = callerDeleted
	}
	if _, err := s.store.CreateInvitations(context.Background(), []workspaceapp.InvitationRow{
		{ID: id, WorkspaceID: s.workspaces[slug], Email: email, Role: role, CreatedBy: s.ids[admin], Now: s.now},
	}); err != nil {
		s.t.Fatal(err)
	}
}

func (s matrixSeed) exec(pool *pgxpool.Pool, sql string, args ...any) {
````

`server/internal/bootstrap/permission_matrix_workspace_test.go`（修改，1 处）：

````old server/internal/bootstrap/permission_matrix_workspace_test.go
			cells: ofMember(cellOwnMembership, cellForbidden, cellForbidden)},
````

````new server/internal/bootstrap/permission_matrix_workspace_test.go
			cells: ofMember(cellOwnMembership, cellForbidden, cellForbidden)},
		{op: "listWorkspaceInvitations", request: toWorkspace(http.MethodGet, "/invitations", ""),
			cells: inWorkspace(cellOK, cellForbidden, cellForbidden), check: listsTheInvitations},
````

`server/internal/bootstrap/permission_matrix_invitations_test.go`（新文件，60 行）：

````file server/internal/bootstrap/permission_matrix_invitations_test.go
package bootstrap

import (
	"log/slog"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	workspacedomain "github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
)

// The invitations' part of the workspace module's rows (M3 design 9.2).

// invitationToken is the token of the invitation id under the matrix's
// signing key, as the app wired on it computes it: identity's MAC of the
// workspace module's purpose.
func invitationToken(t *testing.T, id uuid.UUID) string {
	t.Helper()
	keys, err := identity.LoadKeys([]byte(testKeyPEM), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	mac, err := keys.MAC(workspace.InvitationMACPurpose)
	if err != nil {
		t.Fatal(err)
	}
	return workspacedomain.FormatToken(mac.Tag(workspacedomain.InvitationMessage(id)))
}

// listsTheInvitations: the admin's list is acme's invitations, not gone's,
// each with the token of its id.
func listsTheInvitations(t *testing.T, c caller, answer string) {
	var list struct {
		Data []struct {
			ID    uuid.UUID `json:"id"`
			Email string    `json:"email"`
			Token string    `json:"token"`
		} `json:"data"`
	}
	decodeAnswer(t, answer, &list)
	var got, want []string
	for _, i := range list.Data {
		got = append(got, i.Email)
		if i.Token != invitationToken(t, i.ID) {
			t.Errorf("%s: the invitation of %s has the token %s, want its id's", c, i.Email, i.Token)
		}
	}
	for _, i := range matrixInvitations {
		if i.slug == "acme" {
			want = append(want, i.email)
		}
	}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("%s lists the invitations of %q, want %q", c, got, want)
	}
}
````

- [ ] **Step 8: 测试、lint**

Run: `go -C server test -count=1 ./internal/modules/workspace/... ./internal/modules/access/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestTheInvitationMACIsTheDesigns|TestPermissionMatrix$|TestThePermissionMatrixCoversEveryOperation|TestMatrixViolationsCatchesEachGap|TestEveryActionHasARuleAndEveryRuleAnAction' ./internal/bootstrap/`
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
git add api/modules/workspace.yaml api/openapi.yaml server/internal/bootstrap/app.go server/internal/bootstrap/invitation_mac_test.go server/internal/bootstrap/permission_matrix_invitations_test.go server/internal/bootstrap/permission_matrix_seeded_test.go server/internal/bootstrap/permission_matrix_test.go server/internal/bootstrap/permission_matrix_workspace_test.go server/internal/modules/access/domain/rules.go server/internal/modules/access/domain/rules_test.go server/internal/modules/workspace/adapter/http/handler.go server/internal/modules/workspace/adapter/http/handler_test.go server/internal/modules/workspace/adapter/http/invitations.go server/internal/modules/workspace/adapter/http/invitations_test.go server/internal/modules/workspace/adapter/postgres/failures_test.go server/internal/modules/workspace/adapter/postgres/invitations.go server/internal/modules/workspace/adapter/postgres/invitations_test.go server/internal/modules/workspace/adapter/postgres/queries/invitations.sql server/internal/modules/workspace/app/fakes_invitations_test.go server/internal/modules/workspace/app/invitation_ports.go server/internal/modules/workspace/app/list_invitations.go server/internal/modules/workspace/app/list_invitations_test.go server/internal/modules/workspace/app/tokens.go server/internal/modules/workspace/domain/actions.go server/internal/modules/workspace/domain/invitation.go server/internal/modules/workspace/module.go api/dist/openapi.yaml server/internal/modules/workspace/adapter/http/gen/server.gen.go server/internal/modules/workspace/adapter/postgres/gen/invitations.sql.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M3/P3): list a workspace's invitations, each with its link's token

GET /api/v0/workspaces/{slug}/invitations: the undeleted invitations,
pending or declined, newest first, for the workspace's admins only
(decision 4). The server stores no token: it computes each from the
invitation's id with the invitation MAC, which bootstrap asks identity's
keys for under workspace.InvitationMACPurpose (M3 design 3.8, 6.6).

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A）：

| 改坏 | 必须失败的测试 |
|---|---|
| 给每个邀请它工作区 id 的令牌 | `TestListWorkspaceInvitationsListsEachWithItsToken`、`TestTheTokensAreTheMACs`、`TestPermissionMatrix` |
| 组合根要的用途不是 `"workspace-invitation"` | `TestTheInvitationMACIsTheDesigns` |
| `ListInvitations` 列已删除的；列每个工作区的；只按 id 排 | `TestListInvitations` |
| `workspace_invitation.list` 给成员 | `TestEveryRuleDecidesItsCells`、`TestPermissionMatrix` |

**Done when:** 用例、存储、HTTP、矩阵（6 格）通过；组合根的 MAC 给出设计的已知答案；前端检查通过。

---

### Task 6: `createWorkspaceInvitations`；`CallerLock` 接进 `workspace`

**Files:**
- Create: `server/internal/modules/workspace/app/create_invitations.go`、`server/internal/modules/workspace/app/create_invitations_test.go`、`server/internal/modules/workspace/domain/invitation_test.go`
- Modify: `api/modules/workspace.yaml`、`server/internal/bootstrap/app.go`、`server/internal/bootstrap/permission_matrix_invitations_test.go`、`server/internal/bootstrap/permission_matrix_workspace_test.go`、`server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`、`server/internal/modules/workspace/adapter/http/handler.go`、`server/internal/modules/workspace/adapter/http/handler_test.go`、`server/internal/modules/workspace/adapter/http/invitations.go`、`server/internal/modules/workspace/adapter/http/invitations_test.go`、`server/internal/modules/workspace/app/fakes_invitations_test.go`、`server/internal/modules/workspace/app/invitation_ports.go`、`server/internal/modules/workspace/app/list_invitations_test.go`、`server/internal/modules/workspace/domain/actions.go`、`server/internal/modules/workspace/domain/invitation.go`、`server/internal/modules/workspace/module.go`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/workspace/adapter/http/gen/bodyshape.gen.go`、`server/internal/modules/workspace/adapter/http/gen/server.gen.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Produces（spec 2.8，M3 设计 3.6、3.8、5.1）：接口 `POST /api/v0/workspaces/{slug}/invitations`，`WorkspaceInvitationsCreate{invitations: [InvitationCreate{email, role}]}`，201 `WorkspaceInvitationList`；`domain.NewInvitation`、`MaxInvitations = 100`、`CheckInvitations(batch) ([]NewInvitation, error)`、`MemberAddress(i)`、`InvitedAddress(i)`；`domain.ActionInvitationCreate`，规则 `workspace_invitation.create`：管理员；`app.CallerLock`、`app.InvitationCreator`、`app.CreateInvitationsDeps{Caller, Invitations, Profiles, Auth, Tx, Clock, MAC}`、`NewCreateWorkspaceInvitations`；`byAddress(rows)`；`workspace.Deps.CallerLock`（`bootstrap` 接 `identity.Provided.CredentialLock`）。
- 使用者：Task 13 的交错 9、18；Task 14 的 `invite`。

**Tests:**
- `domain/invitation_test.go`：`TestCheckInvitationsNormalizes`；`TestCheckInvitationsRefuses`（无效的邮箱、规范化之后重复的（后出现的）、三种以外的角色（按集合，含 0、10、16、25），都在一个回答里，下标是请求中的；条数的限制先单独回答）；`TestTheProblemsOfAnAddressNameItsIndex`。
- `app/create_invitations_test.go`：`TestCreateWorkspaceInvitationsLocksDecidesChecksThenInserts`（事务里：凭证锁、工作区 `FOR SHARE`、判定、有效成员的邮箱和邀请、按邮箱排序插入；回答按请求的顺序，是存下的行加令牌；别的工作区的邀请不占；两位管理员、两个工作区）；`TestCreatingInvitationsReadsTheClockBeforeItsTransaction`（时钟读一次，在事务的第一个调用之前）；`TestCreateWorkspaceInvitationsChecksTheBatchFirst`（只看请求的拒绝不开事务）；`TestCreateWorkspaceInvitationsRefusesMembersAndInvitedAddresses`（有效成员的 `not_allowed`、未删除的邀请（待接受或已忽略）的 `duplicate`，一个 422，什么都不插入；已结束的成员关系、别的工作区的邀请不拒绝）；`TestADuplicateFromTheUniqueKeyNamesTheRequestsIndex`（23505 的下标是请求中的，不是插入的顺序）；`TestCreateWorkspaceInvitationsRefusals`（撤销的凭证 401、404、`forbidden`、每个读和插入的失败原样，之后什么都不运行；没有调用者 401）。
- `adapter/http/invitations_test.go`：`TestCreateWorkspaceInvitations`（批量原样交给用例，201）；`TestCreateWorkspaceInvitationsRefusals`（元素缺少角色在用例之前 400）。
- 矩阵：`createWorkspaceInvitations` 三行（新邮箱 201；有效成员的邮箱、已邀请的邮箱 422；成员、访客 403；另三列 404），`invitesTheInvitee` 核对答案；`TestBodiesThatBreakTheStructureAnswer400` 自动加上 `invitations[]` 的用例（Task 3）。

- [ ] **Step 1: 接口描述**

`api/modules/workspace.yaml`（修改，2 处）：

````old api/modules/workspace.yaml
          description: The workspace's invitations.
````

````new api/modules/workspace.yaml
          description: The workspace's invitations.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/WorkspaceInvitationList'
        default:
          $ref: '#/components/responses/Problem'
    post:
      operationId: createWorkspaceInvitations
      tags: [workspace]
      summary: Invite addresses to a workspace
      description: >-
        For the workspace's admins. Each address is normalized as at
        registration (surrounding white space removed, lower case) and the
        batch is checked first: 1–100 invitations, each address valid
        (invalid_format) and listed once (duplicate), each role one of the
        three (invalid_format). Then, the workspace looked at: an active
        member's address is not_allowed, and an address with an invitation
        to the workspace, pending or declined, duplicate, also when another
        request invites it at the same time. The batch is refused as a whole,
        one validation_failed naming each invitations[i] of the request as
        sent. A declined invitation holds its address until it is deleted.
        The answer is the new invitations in the request's order, each with
        the token of its link. A workspace that does not exist, is deleted,
        or of which the caller is not an active member answers
        workspace.not_found; a member or a guest, forbidden. A credential
        revoked meanwhile, by a password reset, answers unauthorized, and
        nothing is invited.
      security: [{bearer: []}]
      x-problem-codes: [validation_failed, workspace.not_found, forbidden]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/WorkspaceInvitationsCreate'
      responses:
        '201':
          description: The new invitations, in the request's order.
````

````old api/modules/workspace.yaml
            $ref: '#/components/schemas/WorkspaceInvitation'
````

````new api/modules/workspace.yaml
            $ref: '#/components/schemas/WorkspaceInvitation'
    InvitationCreate:
      type: object
      additionalProperties: false
      required: [email, role]
      properties:
        email:
          description: An e-mail address; it is normalized as at registration.
          type: string
        role:
          $ref: '#/components/schemas/WorkspaceRole'
    WorkspaceInvitationsCreate:
      type: object
      additionalProperties: false
      required: [invitations]
      properties:
        invitations:
          description: 1–100 invitations, created all together or not at all.
          type: array
          items:
            $ref: '#/components/schemas/InvitationCreate'
````

- [ ] **Step 2: 生成**

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `2d834a33c222d557458e570e12576265135e96db441bc0e827ccdf686c59b658` | 1485 | `api/dist/openapi.yaml` |
| `99cbc9a9bc95a30c0af7f622ae77f1966c97c872769ed31e4c3a8572ea4aa7d5` | 39 | `server/internal/modules/workspace/adapter/http/gen/bodyshape.gen.go` |
| `4c0bc2ee6fa85428a9f9877f3f612b545975aae70419a9a275418538210d5b8f` | 1772 | `server/internal/modules/workspace/adapter/http/gen/server.gen.go` |
| `c5946295852eaae4b7e29c47cfbe1e08a3dd8448dd93e2737cb15e76e8953cc9` | 1558 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/workspace/adapter/http/gen/bodyshape.gen.go server/internal/modules/workspace/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 3: 批量的检查**

`server/internal/modules/workspace/domain/invitation.go`（修改，2 处）：

````old server/internal/modules/workspace/domain/invitation.go
import (
````

````new server/internal/modules/workspace/domain/invitation.go
import (
	"fmt"
	"slices"
````

````old server/internal/modules/workspace/domain/invitation.go
	Token string
}

````

````new server/internal/modules/workspace/domain/invitation.go
	Token string
}

// NewInvitation is an invitation as a request asks for it: an address and a
// role.
type NewInvitation struct {
	Email string
	Role  shared.Role
}

// MaxInvitations is how many invitations one request makes at most (M3
// design 5.1).
const MaxInvitations = 100

// CheckInvitations normalizes each address as registration does (M3 design
// 3.13) and checks what the request alone decides, before anything is read
// (M3 design 3.6 convention 2): 1 to MaxInvitations invitations; each
// address valid, and not listed twice; each role one of the three. The
// batch is refused as a whole, one 422 validation_failed with every problem,
// each on invitations[i] of the request as sent (M3 design 3.8). It returns
// the batch in the request's order, the addresses normalized.
func CheckInvitations(batch []NewInvitation) ([]NewInvitation, error) {
	switch {
	case len(batch) == 0:
		return nil, shared.Invalid(shared.FieldError{Field: "invitations", Code: shared.FieldTooShort, Message: "must list an invitation"})
	case len(batch) > MaxInvitations:
		return nil, shared.Invalid(shared.FieldError{Field: "invitations", Code: shared.FieldTooLong,
			Message: fmt.Sprintf("must list at most %d invitations", MaxInvitations)})
	}
	var fields []shared.FieldError
	listed := map[string]bool{}
	out := make([]NewInvitation, len(batch))
	for i, inv := range batch {
		email := shared.NormalizeEmail(inv.Email)
		switch {
		case !shared.ValidEmail(email):
			fields = append(fields, invitationField(i, "email", shared.FieldInvalidFormat, "is not a valid e-mail address"))
		case listed[email]:
			fields = append(fields, invitationField(i, "email", shared.FieldDuplicate, "is listed twice"))
		}
		listed[email] = true
		if !slices.Contains(roles, inv.Role) {
			fields = append(fields, invitationField(i, "role", shared.FieldInvalidFormat, "is not 5, 15 or 20"))
		}
		out[i] = NewInvitation{Email: email, Role: inv.Role}
	}
	if len(fields) > 0 {
		return nil, shared.Invalid(fields...)
	}
	return out, nil
}

// MemberAddress is the problem of the address of invitation i of a request
// that an active member of the workspace has: an invitation never changes an
// active membership (M3 design 3.8).
func MemberAddress(i int) shared.FieldError {
	return invitationField(i, "email", shared.FieldNotAllowed, "is an active member's address")
}

// InvitedAddress is the problem of the address of invitation i of a request
// that an undeleted invitation of the workspace has, pending or declined
// (M3 design 3.8), however the use case learned it: before it inserted, or
// from the unique key of a concurrent request's insert.
func InvitedAddress(i int) shared.FieldError {
	return invitationField(i, "email", shared.FieldDuplicate, "has an invitation to the workspace already")
}

// invitationField is a problem with field of invitation i of a request.
func invitationField(i int, field, code, message string) shared.FieldError {
	return shared.FieldError{Field: fmt.Sprintf("invitations[%d].%s", i, field), Code: code, Message: message}
}

````

`server/internal/modules/workspace/domain/invitation_test.go`（新文件，93 行）：

````file server/internal/modules/workspace/domain/invitation_test.go
package domain_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fieldsOf are err's field problems as "field code", or nil when err is no
// 422 validation_failed.
func fieldsOf(err error) []string {
	var e *shared.Error
	if !errors.As(err, &e) || !errors.Is(err, shared.Invalid()) {
		return nil
	}
	var out []string
	for _, f := range e.Fields {
		out = append(out, f.Field+" "+f.Code)
	}
	return out
}

// A valid batch comes back in the request's order, each address
// normalized as registration does it (M3 design 3.13), each role as sent.
func TestCheckInvitationsNormalizes(t *testing.T) {
	batch := []domain.NewInvitation{{" Zoe@Corp.COM\t", shared.RoleAdmin}, {"elodie@exämple.com", shared.RoleGuest}, {"bob@corp.com", shared.RoleMember}}

	got, err := domain.CheckInvitations(batch)

	want := []domain.NewInvitation{{"zoe@corp.com", shared.RoleAdmin}, {"elodie@exämple.com", shared.RoleGuest}, {"bob@corp.com", shared.RoleMember}}
	if err != nil || !slices.Equal(got, want) {
		t.Errorf("CheckInvitations() = %+v, %v; want %+v", got, err, want)
	}
}

// Every problem the request alone shows is in one answer, each on its
// invitation's index in the request: an invalid address, one listed again
// after normalization (the later ones), a role outside the three, compared
// by set. The limits of the batch come first, alone.
func TestCheckInvitationsRefuses(t *testing.T) {
	many := make([]domain.NewInvitation, domain.MaxInvitations+1)
	for i := range many {
		many[i] = domain.NewInvitation{Email: "someone" + strings.Repeat("x", i) + "@corp.com", Role: shared.RoleGuest}
	}
	tests := []struct {
		name  string
		batch []domain.NewInvitation
		want  []string
	}{
		{"no invitation", nil, []string{"invitations too_short"}},
		{"one too many", many, []string{"invitations too_long"}},
		{"an invalid address", []domain.NewInvitation{{"bob@corp.com", 5}, {"not an address", 5}}, []string{"invitations[1].email invalid_format"}},
		{"an empty address", []domain.NewInvitation{{" ", 15}}, []string{"invitations[0].email invalid_format"}},
		{"an address too long", []domain.NewInvitation{{strings.Repeat("a", 250) + "@corp.com", 15}}, []string{"invitations[0].email invalid_format"}},
		{"an address twice, once in upper case", []domain.NewInvitation{{"bob@corp.com", 5}, {"carol@corp.com", 5}, {" BOB@corp.com", 15},
			{"bob@corp.com", 20}}, []string{"invitations[2].email duplicate", "invitations[3].email duplicate"}},
		{"role 10, between two roles", []domain.NewInvitation{{"bob@corp.com", 10}}, []string{"invitations[0].role invalid_format"}},
		{"role 0", []domain.NewInvitation{{"bob@corp.com", 0}}, []string{"invitations[0].role invalid_format"}},
		{"role 25, above the admin", []domain.NewInvitation{{"bob@corp.com", 25}}, []string{"invitations[0].role invalid_format"}},
		{"every problem at once", []domain.NewInvitation{{"x", 1}, {"bob@corp.com", 5}, {"Bob@corp.com", 21}},
			[]string{"invitations[0].email invalid_format", "invitations[0].role invalid_format", "invitations[2].email duplicate",
				"invitations[2].role invalid_format"}},
	}
	for _, tt := range tests {
		got, err := domain.CheckInvitations(tt.batch)
		if got != nil || !slices.Equal(fieldsOf(err), tt.want) {
			t.Errorf("%s: CheckInvitations() = %+v, %v (%q); want %q", tt.name, got, err, fieldsOf(err), tt.want)
		}
	}
	if _, err := domain.CheckInvitations(many[:domain.MaxInvitations]); err != nil {
		t.Errorf("CheckInvitations() of %d invitations = %v, want them allowed", domain.MaxInvitations, err)
	}
}

// The problems found after the request is read name the request's index.
func TestTheProblemsOfAnAddressNameItsIndex(t *testing.T) {
	for _, tt := range []struct {
		got  shared.FieldError
		want string
	}{
		{domain.MemberAddress(3), "invitations[3].email not_allowed"},
		{domain.InvitedAddress(0), "invitations[0].email duplicate"},
		{domain.InvitedAddress(12), "invitations[12].email duplicate"},
	} {
		if got := tt.got.Field + " " + tt.got.Code; got != tt.want {
			t.Errorf("%s, want %s", got, tt.want)
		}
	}
}
````

`server/internal/modules/workspace/domain/actions.go`（修改，2 处）：

````old server/internal/modules/workspace/domain/actions.go
	ActionInvitationList shared.Action = "workspace_invitation.list"
````

````new server/internal/modules/workspace/domain/actions.go
	ActionInvitationList shared.Action = "workspace_invitation.list"
	// ActionInvitationCreate is inviting addresses to a workspace:
	// createWorkspaceInvitations.
	ActionInvitationCreate shared.Action = "workspace_invitation.create"
````

````old server/internal/modules/workspace/domain/actions.go
		ActionInvitationList}
````

````new server/internal/modules/workspace/domain/actions.go
		ActionInvitationList, ActionInvitationCreate}
````

- [ ] **Step 4: 端口、用例**

`server/internal/modules/workspace/app/invitation_ports.go`（修改，2 处）：

````old server/internal/modules/workspace/app/invitation_ports.go
}

// InvitationLister reads a workspace and lists its invitations.
````

````new server/internal/modules/workspace/app/invitation_ports.go
}

// CallerLock is identity's credential lock (M2 design 3.5), which
// identity.Provide offers (M3 design 6.5, 6.6): LockCaller locks the
// caller's account row FOR NO KEY UPDATE until the transaction ends, then
// checks under the lock that his account is active and his session or
// personal access token valid at now; 401 unauthorized otherwise. It is the
// first lock of a transaction that issues something with the caller's
// credential (M3 design 3.6 convention 1, 3.8).
type CallerLock interface {
	LockCaller(ctx context.Context, actor shared.Actor, now time.Time) error
}

// InvitationLister reads a workspace and lists its invitations.
````

````old server/internal/modules/workspace/app/invitation_ports.go
}

// DuplicateInvitation is a store's answer to an insert that the unique key
````

````new server/internal/modules/workspace/app/invitation_ports.go
}

// InvitationCreator inserts a batch of invitations under the workspace's
// FOR SHARE, after it read what the batch must not repeat.
type InvitationCreator interface {
	WorkspaceSharer
	// ListMembers returns the undeleted memberships of the workspace,
	// active or not.
	ListMembers(ctx context.Context, workspaceID uuid.UUID) ([]domain.Membership, error)
	// ListInvitations returns the workspace's undeleted invitations.
	ListInvitations(ctx context.Context, workspaceID uuid.UUID) ([]domain.Invitation, error)
	// CreateInvitations inserts rows, one statement each, in the order
	// given, and returns them as stored, in that order; the first row whose
	// address an undeleted invitation of the workspace has is
	// *DuplicateInvitation.
	CreateInvitations(ctx context.Context, rows []InvitationRow) ([]domain.Invitation, error)
}

// DuplicateInvitation is a store's answer to an insert that the unique key
````

`server/internal/modules/workspace/app/create_invitations.go`（新文件，154 行）：

````file server/internal/modules/workspace/app/create_invitations.go
package app

import (
	"context"
	"errors"
	"slices"
	"strings"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// CreateWorkspaceInvitations invites addresses to a workspace: POST
// /api/v0/workspaces/{slug}/invitations.
type CreateWorkspaceInvitations struct {
	d      CreateInvitationsDeps
	tokens invitationTokens
}

// CreateInvitationsDeps are the use case's ports.
type CreateInvitationsDeps struct {
	Caller      CallerLock
	Invitations InvitationCreator
	Profiles    MemberProfiles
	Auth        shared.Authorizer
	Tx          shared.TxManager
	Clock       Clock
	MAC         InvitationMAC
}

// NewCreateWorkspaceInvitations returns the use case.
func NewCreateWorkspaceInvitations(d CreateInvitationsDeps) *CreateWorkspaceInvitations {
	return &CreateWorkspaceInvitations{d: d, tokens: invitationTokens{mac: d.MAC}}
}

// Execute checks the batch, then, in one transaction, in the order of M3
// design 3.6: the caller's account row, his credential checked under the
// lock (CallerLock), so that a password reset committed first leaves no
// invitation (3.8); the workspace FOR SHARE; the decision on
// workspace_invitation.create; the addresses of active members and of
// undeleted invitations refused; then the rows inserted in the order of
// their addresses, so that two batches of overlapping addresses under the
// shared lock wait for each other in one direction and never deadlock (3.6
// convention 5). All of the batch or none of it. The rows are new, so the
// clock is read once, before the transaction (P2 spec 2.6 is for rows that
// exist): the credential is checked at the request's time, as M2's token
// creation does. The answer is the batch in the request's order, each
// invitation with its token.
func (u *CreateWorkspaceInvitations) Execute(ctx context.Context, slug string, batch []domain.NewInvitation) ([]domain.InvitationWithToken, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	batch, err = domain.CheckInvitations(batch)
	if err != nil {
		return nil, err
	}
	now := u.d.Clock.Now()
	rows := make([]InvitationRow, len(batch))
	for i, inv := range batch {
		rows[i] = InvitationRow{ID: uuid.NewV7(), Email: inv.Email, Role: inv.Role, CreatedBy: actor.UserID, Now: now}
	}
	var stored []domain.Invitation
	err = u.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := u.d.Caller.LockCaller(ctx, actor, now); err != nil {
			return err
		}
		id, _, err := lockAndDecide(ctx, u.d.Invitations.ShareWorkspaceBySlug, u.d.Auth, actor, slug, domain.ActionInvitationCreate)
		if err != nil {
			return err
		}
		if err := u.checkAddresses(ctx, id, batch); err != nil {
			return err
		}
		for i := range rows {
			rows[i].WorkspaceID = id
		}
		stored, err = u.d.Invitations.CreateInvitations(ctx, byAddress(rows))
		var taken *DuplicateInvitation
		if errors.As(err, &taken) {
			// A concurrent batch committed the address after the check: the
			// same answer as the check's (3.8).
			return shared.Invalid(domain.InvitedAddress(slices.IndexFunc(batch, func(inv domain.NewInvitation) bool { return inv.Email == taken.Email })))
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	byID := make(map[uuid.UUID]domain.Invitation, len(stored))
	for _, inv := range stored {
		byID[inv.ID] = inv
	}
	out := make([]domain.Invitation, len(rows))
	for i, r := range rows {
		out[i] = byID[r.ID]
	}
	return u.tokens.withTokens(out), nil
}

// checkAddresses refuses, as one 422, each address of the batch that an
// active member of the workspace has (not_allowed: an invitation never
// changes an active membership) or an undeleted invitation of it has,
// pending or declined (duplicate) (M3 design 3.8). The members' addresses
// come through MemberProfiles, without a lock: Accounts is only ever a
// transaction's first lock (3.6 convention 1).
func (u *CreateWorkspaceInvitations) checkAddresses(ctx context.Context, workspaceID uuid.UUID, batch []domain.NewInvitation) error {
	members, err := u.d.Invitations.ListMembers(ctx, workspaceID)
	if err != nil {
		return err
	}
	var active []uuid.UUID
	for _, m := range members {
		if m.IsActive {
			active = append(active, m.MemberID)
		}
	}
	profiles, err := u.d.Profiles.PublicProfiles(ctx, active)
	if err != nil {
		return err
	}
	memberAddresses := map[string]bool{}
	for _, p := range profiles {
		memberAddresses[p.Email] = true
	}
	invitations, err := u.d.Invitations.ListInvitations(ctx, workspaceID)
	if err != nil {
		return err
	}
	invited := map[string]bool{}
	for _, inv := range invitations {
		invited[inv.Email] = true
	}
	var fields []shared.FieldError
	for i, inv := range batch {
		switch {
		case memberAddresses[inv.Email]:
			fields = append(fields, domain.MemberAddress(i))
		case invited[inv.Email]:
			fields = append(fields, domain.InvitedAddress(i))
		}
	}
	if len(fields) > 0 {
		return shared.Invalid(fields...)
	}
	return nil
}

// byAddress is rows in the order of their addresses, bytewise: the one
// order every batch inserts in (M3 design 3.6 convention 5).
func byAddress(rows []InvitationRow) []InvitationRow {
	return slices.SortedFunc(slices.Values(rows), func(a, b InvitationRow) int { return strings.Compare(a.Email, b.Email) })
}
````

`server/internal/modules/workspace/app/fakes_invitations_test.go`（修改，3 处）：

````old server/internal/modules/workspace/app/fakes_invitations_test.go
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
)
````

````new server/internal/modules/workspace/app/fakes_invitations_test.go
	"fmt"
	"strings"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeCallerLock logs each lock with the caller's credential and the time,
// and answers err.
type fakeCallerLock struct {
	log *callLog
	err error
}

func (f *fakeCallerLock) LockCaller(ctx context.Context, actor shared.Actor, now time.Time) error {
	f.log.add(ctx, "LockCaller %s session %s at %s", actor.UserID, actor.SessionID, now.Format(time.RFC3339Nano))
	return f.err
}
````

````old server/internal/modules/workspace/app/fakes_invitations_test.go
	listErr     error // for ListInvitations
````

````new server/internal/modules/workspace/app/fakes_invitations_test.go
	listErr     error  // for ListInvitations
	createErr   error  // for CreateInvitations
	taken       string // an address CreateInvitations finds taken, as the unique key would
````

````old server/internal/modules/workspace/app/fakes_invitations_test.go
	return out, nil
}

````

````new server/internal/modules/workspace/app/fakes_invitations_test.go
	return out, nil
}

// CreateInvitations logs the rows' addresses in the order given, and their
// workspace, inviter and time; it stores them and answers them as stored,
// at the stored time, unless an address is taken.
func (f *fakeInvitations) CreateInvitations(ctx context.Context, rows []app.InvitationRow) ([]domain.Invitation, error) {
	var emails []string
	for _, r := range rows {
		emails = append(emails, fmt.Sprintf("%s as %d", r.Email, r.Role))
	}
	f.log.add(ctx, "CreateInvitations %s in %s by %s at %s", strings.Join(emails, ", "), rows[0].WorkspaceID, rows[0].CreatedBy,
		rows[0].Now.Format(time.RFC3339Nano))
	if f.createErr != nil {
		return nil, fmt.Errorf("create workspace invitation: %w", f.createErr)
	}
	var out []domain.Invitation
	for _, r := range rows {
		if r.Email == f.taken {
			return nil, &app.DuplicateInvitation{Email: r.Email}
		}
		by := r.CreatedBy
		out = append(out, domain.Invitation{ID: r.ID, WorkspaceID: r.WorkspaceID, Email: r.Email, Role: r.Role, CreatedAt: stored(r.Now),
			CreatedByID: &by})
	}
	f.invitations = append(f.invitations, out...)
	return out, nil
}

````

`server/internal/modules/workspace/app/create_invitations_test.go`（新文件，242 行）：

````file server/internal/modules/workspace/app/create_invitations_test.go
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

// session is the caller's session: the credential CallerLock checks.
var session = uuid.NewV7()

// withSession is user as the caller, signed in with session.
func withSession(user app.AccountState) context.Context {
	return shared.WithActor(context.Background(), shared.Actor{UserID: user.ID, SessionID: session})
}

// create is CreateWorkspaceInvitations over f's fakes, its clock at
// clockNow.
func (f *invitationsFixture) create(clock app.Clock) *app.CreateWorkspaceInvitations {
	return app.NewCreateWorkspaceInvitations(app.CreateInvitationsDeps{Caller: f.caller, Invitations: f.invitations, Profiles: f.profiles,
		Auth: f.auth, Tx: f.tx, Clock: clock, MAC: f.mac})
}

// createCalls are the calls of a creation by user in w up to the inserts:
// the credential, the workspace's lock, the decision, the members, their
// addresses, the invitations.
func createCalls(user app.AccountState, w domain.Workspace, active ...uuid.UUID) []string {
	return []string{
		fmt.Sprintf("LockCaller %s session %s at %s", user.ID, session, clockNow.Format(time.RFC3339Nano)),
		"ShareWorkspaceBySlug " + w.Slug,
		"Authorize " + user.ID.String() + " workspace_invitation.create on " + w.ID.String() + "/" + uuid.Nil().String(),
		"ListMembers " + w.ID.String(),
		fmt.Sprintf("PublicProfiles %v", active),
		"ListInvitations " + w.ID.String(),
	}
}

// fieldsOf are err's field problems as "field code".
func fieldsOf(err error) []string {
	var e *shared.Error
	if !errors.As(err, &e) {
		return nil
	}
	var out []string
	for _, f := range e.Fields {
		out = append(out, f.Field+" "+f.Code)
	}
	return out
}

// A creation checks the batch, then in one transaction takes the caller's
// credential lock, the workspace's FOR SHARE and the decision, reads the
// active members' addresses and the invitations, and inserts the rows in
// the order of their addresses, by the caller at the clock's time. The
// answer is the batch in the request's order, as stored, each with its
// token. An address invited to another workspace is free. Two admins, two
// workspaces.
func TestCreateWorkspaceInvitationsLocksDecidesChecksThenInserts(t *testing.T) {
	for _, tt := range []struct {
		user   app.AccountState
		w      domain.Workspace
		batch  []domain.NewInvitation
		active []uuid.UUID
		insert string
		want   []domain.NewInvitation
	}{
		{alice, acme, []domain.NewInvitation{{Email: " Zoe@corp.com ", Role: shared.RoleMember}, {Email: "erin@corp.com", Role: shared.RoleGuest}}, []uuid.UUID{alice.ID, bob.ID},
			"erin@corp.com as 5, zoe@corp.com as 15", []domain.NewInvitation{{Email: "zoe@corp.com", Role: shared.RoleMember}, {Email: "erin@corp.com", Role: shared.RoleGuest}}},
		{bob, beta, []domain.NewInvitation{{Email: "carol@corp.com", Role: shared.RoleAdmin}}, []uuid.UUID{bob.ID},
			"carol@corp.com as 20", []domain.NewInvitation{{Email: "carol@corp.com", Role: shared.RoleAdmin}}},
	} {
		f := newInvitations()
		got, err := f.create(clockAt{at: clockNow}).Execute(withSession(tt.user), tt.w.Slug, tt.batch)
		if err != nil || len(got) != len(tt.want) {
			t.Fatalf("%s invites to %s: %+v, %v; want %d invitations", tt.user.Email, tt.w.Slug, got, err, len(tt.want))
		}
		for i, inv := range got {
			want := domain.InvitationWithToken{Invitation: domain.Invitation{ID: inv.ID, WorkspaceID: tt.w.ID, Email: tt.want[i].Email,
				Role: tt.want[i].Role, CreatedAt: now, CreatedByID: &tt.user.ID}, Token: tokenOf(f.mac, inv.ID)}
			if !sameInvitations([]domain.InvitationWithToken{inv}, []domain.InvitationWithToken{want}) || inv.ID == uuid.Nil() {
				t.Errorf("%s invites to %s: invitation %d = %+v, want %+v", tt.user.Email, tt.w.Slug, i, inv, want)
			}
		}
		wantCalls := append(createCalls(tt.user, tt.w, tt.active...),
			fmt.Sprintf("CreateInvitations %s in %s by %s at %s", tt.insert, tt.w.ID, tt.user.ID, clockNow.Format(time.RFC3339Nano)))
		if !slices.Equal(f.log.calls, wantCalls) || f.tx.calls != 1 {
			t.Errorf("%s invites to %s: calls = %q in %d transactions, want %q in one", tt.user.Email, tt.w.Slug, f.log.calls, f.tx.calls, wantCalls)
		}
	}
}

// txClock is clockNow; at each read it notes how many transactions tx had
// begun.
type txClock struct {
	tx    *fakeTx
	reads []int
}

func (c *txClock) Now() time.Time {
	c.reads = append(c.reads, c.tx.calls)
	return clockNow
}

// The clock is read once, before the transaction, whose first call is the
// credential lock: the rows are new, and the credential is checked at the
// request's time (spec P3 2.8).
func TestCreatingInvitationsReadsTheClockBeforeItsTransaction(t *testing.T) {
	f := newInvitations()
	clock := &txClock{tx: f.tx}
	if _, err := f.create(clock).Execute(withSession(alice), "acme", []domain.NewInvitation{{Email: "zoe@corp.com", Role: 5}}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(clock.reads, []int{0}) || f.tx.calls != 1 {
		t.Errorf("the clock was read with %v transactions begun, of %d; want once, before the one", clock.reads, f.tx.calls)
	}
}

// A batch the request alone refuses is refused before anything is read or
// locked: no transaction.
func TestCreateWorkspaceInvitationsChecksTheBatchFirst(t *testing.T) {
	f := newInvitations()
	_, err := f.create(clockAt{at: clockNow}).Execute(withSession(alice), "acme",
		[]domain.NewInvitation{{Email: "zoe@corp.com", Role: 5}, {Email: "ZOE@corp.com", Role: 15}, {Email: "not an address", Role: 20}, {Email: "yan@corp.com", Role: 10}})
	want := []string{"invitations[1].email duplicate", "invitations[2].email invalid_format", "invitations[3].role invalid_format"}
	if !errors.Is(err, shared.Invalid()) || !slices.Equal(fieldsOf(err), want) || len(f.log.calls) != 0 || f.tx.calls != 0 {
		t.Errorf("Execute() = %v (%q), calls %q in %d transactions; want 422 %q and nothing", err, fieldsOf(err), f.log.calls, f.tx.calls, want)
	}
}

// Under the lock, the addresses of active members are not_allowed and
// those of undeleted invitations, pending or declined, duplicate, each on
// its index in the request, in one 422; nothing is inserted. An ended
// membership refuses nothing, nor does another workspace's invitation.
func TestCreateWorkspaceInvitationsRefusesMembersAndInvitedAddresses(t *testing.T) {
	f := newInvitations()
	batch := []domain.NewInvitation{{Email: "bob@corp.com", Role: 5}, {Email: "frank@corp.com", Role: 5}, {Email: "dave@corp.com", Role: 5}, {Email: "CAROL@corp.com", Role: 5}, {Email: "alice@corp.com", Role: 20},
		{Email: "erin@corp.com", Role: 5}}

	_, err := f.create(clockAt{at: clockNow}).Execute(withSession(alice), "acme", batch)

	want := []string{"invitations[0].email not_allowed", "invitations[2].email duplicate", "invitations[3].email duplicate",
		"invitations[4].email not_allowed"}
	if !errors.Is(err, shared.Invalid()) || !slices.Equal(fieldsOf(err), want) {
		t.Errorf("Execute() = %v (%q), want 422 %q", err, fieldsOf(err), want)
	}
	if wantCalls := createCalls(alice, acme, alice.ID, bob.ID); !slices.Equal(f.log.calls, wantCalls) {
		t.Errorf("calls = %q, want %q: no insert", f.log.calls, wantCalls)
	}
	// carol's membership of acme ended: without her invitation, she can be
	// invited again.
	f = newInvitations()
	f.invitations.invitations = []domain.Invitation{daveToAcme, erinToBeta}
	if got, err := f.create(clockAt{at: clockNow}).Execute(withSession(alice), "acme", []domain.NewInvitation{{Email: "carol@corp.com", Role: 5}}); err != nil ||
		len(got) != 1 {
		t.Errorf("inviting carol, whose membership ended: %+v, %v; want her invitation", got, err)
	}
}

// When a concurrent batch committed an address after the check, the
// unique key refuses it: the answer is the check's, on the address's index
// in the request as sent, not in the order the rows were inserted.
func TestADuplicateFromTheUniqueKeyNamesTheRequestsIndex(t *testing.T) {
	f := newInvitations()
	f.invitations.taken = "zoe@corp.com"
	_, err := f.create(clockAt{at: clockNow}).Execute(withSession(alice), "acme",
		[]domain.NewInvitation{{Email: "yan@corp.com", Role: 5}, {Email: "zoe@corp.com", Role: 5}, {Email: "abe@corp.com", Role: 5}})
	if want := []string{"invitations[1].email duplicate"}; !errors.Is(err, shared.Invalid()) || !slices.Equal(fieldsOf(err), want) {
		t.Errorf("Execute() = %v (%q), want 422 %q", err, fieldsOf(err), want)
	}
	if last := f.log.calls[len(f.log.calls)-1]; last != fmt.Sprintf("CreateInvitations abe@corp.com as 5, yan@corp.com as 5, zoe@corp.com as 5 in %s by %s at %s",
		acme.ID, alice.ID, clockNow.Format(time.RFC3339Nano)) {
		t.Errorf("the insert = %q, want the rows in the order of their addresses", last)
	}
}

// Each refusal and failure is the answer, and nothing after it runs: a
// revoked credential is identity's 401; a workspace not there or not
// visible, workspace.not_found; the Authorizer's forbidden; a failure of
// any read or of the insert is itself, never a 404 or a 422. Without a
// caller it is 401 and nothing runs.
func TestCreateWorkspaceInvitationsRefusals(t *testing.T) {
	failure := errors.New("connection reset")
	all := append(createCalls(alice, acme, alice.ID, bob.ID),
		fmt.Sprintf("CreateInvitations zoe@corp.com as 5 in %s by %s at %s", acme.ID, alice.ID, clockNow.Format(time.RFC3339Nano)))
	tests := []struct {
		name  string
		user  app.AccountState
		slug  string
		set   func(f *invitationsFixture)
		want  error
		calls int // how many of the calls of all, or of the calls up to the decision
	}{
		{"the credential revoked", alice, "acme", func(f *invitationsFixture) { f.caller.err = shared.Unauthenticated() }, shared.Unauthenticated(), 1},
		{"the credential lock failed", alice, "acme", func(f *invitationsFixture) { f.caller.err = failure }, failure, 1},
		{"no such workspace", alice, "nothing", nil, domain.ErrNotFound, 2},
		{"the workspace lock failed", alice, "acme", func(f *invitationsFixture) { f.invitations.lockErrs = map[string]error{"acme": failure} },
			failure, 2},
		{"not visible", carol, "acme", nil, domain.ErrNotFound, 3},
		{"forbidden", bob, "acme", func(f *invitationsFixture) { f.auth.errs = map[grantKey]error{{bob.ID, acme.ID}: shared.Forbidden()} },
			shared.Forbidden(), 3},
		{"the Authorizer failed", alice, "acme", func(f *invitationsFixture) { f.auth.errs = map[grantKey]error{{alice.ID, acme.ID}: failure} },
			failure, 3},
		{"the members failed", alice, "acme", func(f *invitationsFixture) { f.invitations.membersErr = failure }, failure, 4},
		{"their profiles failed", alice, "acme", func(f *invitationsFixture) { f.profiles.err = failure }, failure, 5},
		{"the invitations failed", alice, "acme", func(f *invitationsFixture) { f.invitations.listErr = failure }, failure, 6},
		{"the insert failed", alice, "acme", func(f *invitationsFixture) { f.invitations.createErr = failure }, failure, 7},
		{"the commit failed", alice, "acme", func(f *invitationsFixture) { f.tx.commitErr = failure }, failure, 7},
	}
	for _, tt := range tests {
		f := newInvitations()
		if tt.set != nil {
			tt.set(f)
		}
		got, err := f.create(clockAt{at: clockNow}).Execute(withSession(tt.user), tt.slug, []domain.NewInvitation{{Email: "zoe@corp.com", Role: 5}})
		if !errors.Is(err, tt.want) || got != nil || (tt.want == failure && (errors.Is(err, domain.ErrNotFound) || errors.Is(err, shared.Invalid()))) {
			t.Errorf("%s: Execute() = %+v, %v; want %v", tt.name, got, err, tt.want)
		}
		want := slices.Clone(all[:tt.calls])
		if tt.user != alice {
			want = createCalls(tt.user, acme)[:tt.calls]
		}
		if tt.slug != "acme" {
			want[1] = "ShareWorkspaceBySlug " + tt.slug
		}
		if !slices.Equal(f.log.calls, want) || f.tx.calls != 1 {
			t.Errorf("%s: calls = %q in %d transactions, want %q in one", tt.name, f.log.calls, f.tx.calls, want)
		}
	}
	f := newInvitations()
	if _, err := f.create(clockAt{at: clockNow}).Execute(context.Background(), "acme", []domain.NewInvitation{{Email: "zoe@corp.com", Role: 5}}); !errors.Is(err,
		shared.Unauthenticated()) || len(f.log.calls) != 0 || f.tx.calls != 0 {
		t.Errorf("Execute() without an actor = %v, calls %q; want 401 unauthorized and nothing", err, f.log.calls)
	}
}
````

`server/internal/modules/workspace/app/list_invitations_test.go`（修改，3 处）：

````old server/internal/modules/workspace/app/list_invitations_test.go
// log: acme and beta with carolToAcme, daveToAcme and erinToBeta; alice is
// acme's admin, bob acme's member and beta's admin.
````

````new server/internal/modules/workspace/app/list_invitations_test.go
// log: acme and beta with carolToAcme, daveToAcme and erinToBeta; acme's
// memberships alice's (admin), bob's (member) and carol's, ended; beta's
// bob's (guest). The Authorizer gives alice admin in acme, bob member in
// acme and admin in beta.
````

````old server/internal/modules/workspace/app/list_invitations_test.go
	tx          *fakeTx
	invitations *fakeInvitations
````

````new server/internal/modules/workspace/app/list_invitations_test.go
	tx          *fakeTx
	caller      *fakeCallerLock
	invitations *fakeInvitations
	profiles    *fakeProfiles
````

````old server/internal/modules/workspace/app/list_invitations_test.go
	return &invitationsFixture{log: log, tx: &fakeTx{},
		invitations: &fakeInvitations{fakeWorkspaces: &fakeWorkspaces{log: log, workspaces: []domain.Workspace{acme, beta}},
			invitations: []domain.Invitation{carolToAcme, daveToAcme, erinToBeta}},
````

````new server/internal/modules/workspace/app/list_invitations_test.go
	return &invitationsFixture{log: log, tx: &fakeTx{}, caller: &fakeCallerLock{log: log},
		invitations: &fakeInvitations{fakeWorkspaces: &fakeWorkspaces{log: log, workspaces: []domain.Workspace{acme, beta},
			memberships: map[uuid.UUID][]domain.Membership{acme.ID: {aliceInAcme, bobInAcme, carolInAcme}, beta.ID: {bobInBeta}}},
			invitations: []domain.Invitation{carolToAcme, daveToAcme, erinToBeta}},
		profiles: &fakeProfiles{log: log, profiles: profiles},
````

- [ ] **Step 5: HTTP、规则、接线**

`server/internal/modules/workspace/adapter/http/handler.go`（修改，2 处）：

````old server/internal/modules/workspace/adapter/http/handler.go
}

// CheckSlugUseCase is app.CheckSlug.
````

````new server/internal/modules/workspace/adapter/http/handler.go
}

// CreateInvitationsUseCase is app.CreateWorkspaceInvitations.
type CreateInvitationsUseCase interface {
	Execute(ctx context.Context, slug string, batch []domain.NewInvitation) ([]domain.InvitationWithToken, error)
}

// CheckSlugUseCase is app.CheckSlug.
````

````old server/internal/modules/workspace/adapter/http/handler.go
	ListInvitations   ListInvitationsUseCase
````

````new server/internal/modules/workspace/adapter/http/handler.go
	ListInvitations   ListInvitationsUseCase
	CreateInvitations CreateInvitationsUseCase
````

`server/internal/modules/workspace/adapter/http/invitations.go`（修改，2 处）：

````old server/internal/modules/workspace/adapter/http/invitations.go
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
````

````new server/internal/modules/workspace/adapter/http/invitations.go
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
````

````old server/internal/modules/workspace/adapter/http/invitations.go
	return gen.ListWorkspaceInvitations200JSONResponse{Data: invitations(list)}, nil
````

````new server/internal/modules/workspace/adapter/http/invitations.go
	return gen.ListWorkspaceInvitations200JSONResponse{Data: invitations(list)}, nil
}

// CreateWorkspaceInvitations serves POST
// /api/v0/workspaces/{slug}/invitations.
func (h handler) CreateWorkspaceInvitations(ctx context.Context, req gen.CreateWorkspaceInvitationsRequestObject) (gen.CreateWorkspaceInvitationsResponseObject, error) {
	batch := make([]domain.NewInvitation, len(req.Body.Invitations))
	for i, inv := range req.Body.Invitations {
		batch[i] = domain.NewInvitation{Email: inv.Email, Role: shared.Role(inv.Role)}
	}
	list, err := h.uc.CreateInvitations.Execute(ctx, req.Slug, batch)
	if err != nil {
		return nil, err
	}
	return gen.CreateWorkspaceInvitations201JSONResponse{Data: invitations(list)}, nil
````

`server/internal/modules/workspace/adapter/http/handler_test.go`（修改，1 处）：

````old server/internal/modules/workspace/adapter/http/handler_test.go
		ListInvitations: fakeListInvitations{f.invitations},
````

````new server/internal/modules/workspace/adapter/http/handler_test.go
		ListInvitations: fakeListInvitations{f.invitations}, CreateInvitations: fakeCreateInvitations{f.invitations},
````

`server/internal/modules/workspace/adapter/http/invitations_test.go`（修改，4 处）：

````old server/internal/modules/workspace/adapter/http/invitations_test.go
	"context"
````

````new server/internal/modules/workspace/adapter/http/invitations_test.go
	"context"
	"fmt"
````

````old server/internal/modules/workspace/adapter/http/invitations_test.go
	"slices"
````

````new server/internal/modules/workspace/adapter/http/invitations_test.go
	"slices"
	"strings"
````

````old server/internal/modules/workspace/adapter/http/invitations_test.go
	err   error
````

````new server/internal/modules/workspace/adapter/http/invitations_test.go
	err   error
}

type fakeCreateInvitations struct{ *fakeInvitations }

func (f fakeCreateInvitations) Execute(ctx context.Context, slug string, batch []domain.NewInvitation) ([]domain.InvitationWithToken, error) {
	f.calls = append(f.calls, fmt.Sprintf("create %s %s %+v", caller(ctx), slug, batch))
	return f.lists[slug], f.err
````

````old server/internal/modules/workspace/adapter/http/invitations_test.go
		}
	}
}

````

````new server/internal/modules/workspace/adapter/http/invitations_test.go
		}
	}
}

// POST hands the batch to the use case as sent, the addresses as they are
// and every role, for the caller and the slug of the path, and answers the
// invitations it creates with 201.
func TestCreateWorkspaceInvitations(t *testing.T) {
	inv := &fakeInvitations{lists: map[string][]domain.InvitationWithToken{"acme": {carolInvited}}}
	h := newServer(t, fakes{invitations: inv})
	res, body := do(t, h, request(http.MethodPost, "/api/v0/workspaces/acme/invitations", "alice",
		`{"invitations":[{"email":" Carol@corp.com ","role":15},{"email":"dave","role":10}]}`))
	if want := `{"data":[` + carolInvitedJSON + `]}`; res.StatusCode != http.StatusCreated || body != want+"\n" {
		t.Errorf("POST = %d %s, want 201 %s", res.StatusCode, body, want)
	}
	if want := []string{"create alice acme [{Email: Carol@corp.com  Role:15} {Email:dave Role:10}]"}; !slices.Equal(inv.calls, want) {
		t.Errorf("calls = %q, want %q", inv.calls, want)
	}
}

// The use case's refusals, as the contract declares them; a body whose
// invitation lacks a role is refused before it.
func TestCreateWorkspaceInvitationsRefusals(t *testing.T) {
	for _, tt := range []struct {
		err    error
		status int
		want   string
	}{
		{shared.Invalid(shared.FieldError{Field: "invitations[1].email", Code: shared.FieldDuplicate, Message: "has an invitation to the workspace already"}),
			http.StatusUnprocessableEntity, `{"status":422,"code":"validation_failed","title":"Unprocessable Entity",` +
				`"detail":"The request has invalid values.","errors":[{"field":"invitations[1].email","code":"duplicate",` +
				`"message":"has an invitation to the workspace already"}]}`},
		{domain.ErrNotFound, http.StatusNotFound,
			`{"status":404,"code":"workspace.not_found","title":"Not Found","detail":"The workspace does not exist, or you are not a member of it."}`},
		{shared.Forbidden(), http.StatusForbidden, `{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
	} {
		h := newServer(t, fakes{invitations: &fakeInvitations{err: tt.err}})
		if res, body := do(t, h, request(http.MethodPost, "/api/v0/workspaces/acme/invitations", "bob",
			`{"invitations":[{"email":"carol@corp.com","role":5}]}`)); res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("POST refused with %v = %d %s, want %d %s", tt.err, res.StatusCode, body, tt.status, tt.want)
		}
	}
	inv := &fakeInvitations{}
	h := newServer(t, fakes{invitations: inv})
	if res, body := do(t, h, request(http.MethodPost, "/api/v0/workspaces/acme/invitations", "alice",
		`{"invitations":[{"email":"carol@corp.com"}]}`)); res.StatusCode != http.StatusBadRequest || len(inv.calls) != 0 ||
		!strings.Contains(body, `"field":"invitations[0].role","code":"required"`) {
		t.Errorf("POST without a role = %d %s, calls %q; want 400 on invitations[0].role and no call", res.StatusCode, body, inv.calls)
	}
}

````

`server/internal/modules/access/domain/rules.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules.go
	"workspace_invitation.list": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin}},
````

````new server/internal/modules/access/domain/rules.go
	"workspace_invitation.list": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin}},
	// Only admins invite, so an invitation's role is never above its
	// inviter's (M3 design 3.8): no check of its own.
	"workspace_invitation.create": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin}},
````

`server/internal/modules/access/domain/rules_test.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules_test.go
	"workspace_invitation.list":    {allowed, forbidden, forbidden, invisible, invisible, invisible, forbidden},
````

````new server/internal/modules/access/domain/rules_test.go
	"workspace_invitation.list":    {allowed, forbidden, forbidden, invisible, invisible, invisible, forbidden},
	"workspace_invitation.create":  {allowed, forbidden, forbidden, invisible, invisible, invisible, forbidden},
````

`server/internal/modules/workspace/module.go`（修改，2 处）：

````old server/internal/modules/workspace/module.go
	InvitationMAC app.InvitationMAC
````

````new server/internal/modules/workspace/module.go
	InvitationMAC app.InvitationMAC
	// CallerLock is identity's credential lock (identity.Provide).
	CallerLock app.CallerLock
````

````old server/internal/modules/workspace/module.go
		ListInvitations:   app.NewListWorkspaceInvitations(store, d.Authorizer, d.InvitationMAC),
````

````new server/internal/modules/workspace/module.go
		ListInvitations:   app.NewListWorkspaceInvitations(store, d.Authorizer, d.InvitationMAC),
		CreateInvitations: app.NewCreateWorkspaceInvitations(app.CreateInvitationsDeps{
			Caller: d.CallerLock, Invitations: store, Profiles: d.Profiles, Auth: d.Authorizer, Tx: d.Tx, Clock: d.Clock, MAC: d.InvitationMAC,
		}),
````

`server/internal/bootstrap/app.go`（修改，1 处）：

````old server/internal/bootstrap/app.go
		InvitationMAC:   invitationMAC,
````

````new server/internal/bootstrap/app.go
		InvitationMAC:   invitationMAC,
		CallerLock:      identityPorts.CredentialLock,
````

- [ ] **Step 6: 矩阵**

`server/internal/bootstrap/permission_matrix_workspace_test.go`（修改，2 处）：

````old server/internal/bootstrap/permission_matrix_workspace_test.go
	cellOwnMembership     = cell{http.StatusConflict, "workspace.own_membership"}
````

````new server/internal/bootstrap/permission_matrix_workspace_test.go
	cellOwnMembership     = cell{http.StatusConflict, "workspace.own_membership"}
	cellValidationFailed  = cell{http.StatusUnprocessableEntity, "validation_failed"}
````

````old server/internal/bootstrap/permission_matrix_workspace_test.go
			cells: inWorkspace(cellOK, cellForbidden, cellForbidden), check: listsTheInvitations},
````

````new server/internal/bootstrap/permission_matrix_workspace_test.go
			cells: inWorkspace(cellOK, cellForbidden, cellForbidden), check: listsTheInvitations},
		{op: "createWorkspaceInvitations", write: true, request: toWorkspace(http.MethodPost, "/invitations", inviting("invitee@example.com")),
			cells: inWorkspace(cellCreated, cellForbidden, cellForbidden), check: invitesTheInvitee},
		// The addresses the workspace refuses, read through the wired
		// MemberProfiles and the store: an active member's, an invited one's.
		{op: "createWorkspaceInvitations", variant: "an active member's address", write: true,
			request: toWorkspace(http.MethodPost, "/invitations", inviting("member@example.com")),
			cells:   inWorkspace(cellValidationFailed, cellForbidden, cellForbidden)},
		{op: "createWorkspaceInvitations", variant: "an invited address", write: true,
			request: toWorkspace(http.MethodPost, "/invitations", inviting("newcomer@example.com")),
			cells:   inWorkspace(cellValidationFailed, cellForbidden, cellForbidden)},
````

`server/internal/bootstrap/permission_matrix_invitations_test.go`（修改，1 处）：

````old server/internal/bootstrap/permission_matrix_invitations_test.go
	}
}

````

````new server/internal/bootstrap/permission_matrix_invitations_test.go
	}
}

// inviting is the body of a creation that invites email as a guest.
func inviting(email string) string {
	return `{"invitations":[{"email":"` + email + `","role":5}]}`
}

// invitesTheInvitee: the admin's answer is the one new invitation, of
// invitee@example.com as a guest, with the token of its id.
func invitesTheInvitee(t *testing.T, c caller, answer string) {
	var list struct {
		Data []struct {
			ID    uuid.UUID `json:"id"`
			Email string    `json:"email"`
			Role  int       `json:"role"`
			Token string    `json:"token"`
		} `json:"data"`
	}
	decodeAnswer(t, answer, &list)
	if len(list.Data) != 1 || list.Data[0].Email != "invitee@example.com" || list.Data[0].Role != 5 ||
		list.Data[0].Token != invitationToken(t, list.Data[0].ID) {
		t.Errorf("%s's invitation answers %+v, want invitee@example.com's, as a guest, with its token", c, list.Data)
	}
}

````

- [ ] **Step 7: 测试、lint**

Run: `go -C server test -count=1 ./internal/modules/workspace/... ./internal/modules/access/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestPermissionMatrix$|TestThePermissionMatrixCoversEveryOperation|TestMatrixViolationsCatchesEachGap|TestEveryActionHasARuleAndEveryRuleAnAction|TestBodiesThatBreakTheStructureAnswer400' ./internal/bootstrap/`
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
git add api/modules/workspace.yaml server/internal/bootstrap/app.go server/internal/bootstrap/permission_matrix_invitations_test.go server/internal/bootstrap/permission_matrix_workspace_test.go server/internal/modules/access/domain/rules.go server/internal/modules/access/domain/rules_test.go server/internal/modules/workspace/adapter/http/handler.go server/internal/modules/workspace/adapter/http/handler_test.go server/internal/modules/workspace/adapter/http/invitations.go server/internal/modules/workspace/adapter/http/invitations_test.go server/internal/modules/workspace/app/create_invitations.go server/internal/modules/workspace/app/create_invitations_test.go server/internal/modules/workspace/app/fakes_invitations_test.go server/internal/modules/workspace/app/invitation_ports.go server/internal/modules/workspace/app/list_invitations_test.go server/internal/modules/workspace/domain/actions.go server/internal/modules/workspace/domain/invitation.go server/internal/modules/workspace/domain/invitation_test.go server/internal/modules/workspace/module.go api/dist/openapi.yaml server/internal/modules/workspace/adapter/http/gen/bodyshape.gen.go server/internal/modules/workspace/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M3/P3): invite a batch of addresses to a workspace, all or nothing

POST /api/v0/workspaces/{slug}/invitations checks the batch as the request
alone decides it, then in one transaction locks the inviter's account
row through identity's credential lock, so that a password reset
committed first leaves no invitation, shares the workspace, decides,
refuses active members' and invited addresses through MemberProfiles and
the store, and inserts in the order of the normalized addresses (M3
design 3.6 convention 5, 3.8). A concurrent batch's address the unique
key refuses gets the check's 422, on its index in the request.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A）：

| 改坏 | 必须失败的测试 |
|---|---|
| 按请求的顺序插入 | `TestCreateWorkspaceInvitationsLocksDecidesChecksThenInserts`（Task 13 起另有 `TestInvitingOverlappingBatches`） |
| 23505 答 409；下标取排序后的位置 | `TestADuplicateFromTheUniqueKeyNamesTheRequestsIndex`（同上） |
| 已忽略的邀请不占邮箱 | `TestCreateWorkspaceInvitationsRefusesMembersAndInvitedAddresses` |
| 不查有效成员的邮箱；把已结束的也算上 | `TestCreateWorkspaceInvitationsRefusesMembersAndInvitedAddresses` |
| 不锁邀请人的账户 | `TestCreateWorkspaceInvitationsLocksDecidesChecksThenInserts`（Task 13 起另有 `TestInvitingAndResettingThePassword`） |
| 工作区用 `FOR NO KEY UPDATE` | `TestCreateWorkspaceInvitationsLocksDecidesChecksThenInserts`（同上，`TestInvitingOverlappingBatches`） |
| 在凭证锁之后读时钟 | `TestCreatingInvitationsReadsTheClockBeforeItsTransaction` |
| 角色按大小（5 到 20） | `TestCheckInvitationsRefuses` |
| 回答用例自己拼的行（时钟的时间），不是存下的 | `TestCreateWorkspaceInvitationsLocksDecidesChecksThenInserts` |
| `workspace_invitation.create` 给成员 | `TestEveryRuleDecidesItsCells`、`TestPermissionMatrix` |
| 生成的 bodyshape 让元素接受未声明的属性；元素不要求任何属性 | `TestBodiesThatBreakTheStructureAnswer400` |

**Done when:** 批量的检查、用例、HTTP、矩阵（18 格）通过；23505 翻译为 422、下标是请求中的；前端检查通过。

---

### Task 7: 邀请的读、锁、改角色、删除的存储；按 id 的 `FOR SHARE`

**Files:**
- Create: `server/internal/modules/workspace/adapter/postgres/update_invitation_test.go`
- Modify: `server/internal/modules/workspace/adapter/postgres/invitations.go`、`server/internal/modules/workspace/adapter/postgres/locks.go`、`server/internal/modules/workspace/adapter/postgres/locks_test.go`、`server/internal/modules/workspace/adapter/postgres/queries/invitations.sql`、`server/internal/modules/workspace/adapter/postgres/queries/workspaces.sql`
- Generate: `server/internal/modules/workspace/adapter/postgres/gen/invitations.sql.go`、`server/internal/modules/workspace/adapter/postgres/gen/workspaces.sql.go`

**Interfaces:**
- Produces（spec 2.9，M3 设计 3.6 约定二）：存储 `InvitationByID(ctx, id)`（未删除的；`app.ErrNotFound`）、`LockInvitation(ctx, id)`（`FOR UPDATE`、重读；等锁期间被删除读到 0 行）、`UpdateInvitationRole(ctx, id, role, by, now) (domain.Invitation, error)`、`DeleteInvitation(ctx, id, by, now) error`、`ShareWorkspace(ctx, id) error`（按 id 的 `FOR SHARE`，与 `ShareWorkspaceBySlug` 同一把锁）；`lockedWorkspace(id, err)` 是三把按 id、按 slug 的锁共用的翻译。
- 使用者：Task 8（修改、删除）、Task 11（回应）、Task 12（注册的检查）。

**Tests:**（真实数据库）
- `adapter/postgres/update_invitation_test.go`：`TestTheInvitationReadsFindOnlyAnUndeletedInvitation`（待接受、已忽略的照存下的读；已接受、已删除的、没有的 id 是 `app.ErrNotFound`；失败是它的错误，锁的失败在 `BEGIN` 之后强制）；`TestLockInvitationLocksTheRowForUpdate`（另一个 `LockInvitation` 等，`FOR KEY SHARE` 也等（`FOR NO KEY UPDATE` 会放过它）；别的邀请的锁不等；`lock_timeout` 之下得到 `55P03`）；`TestLockInvitationSkipsAnInvitationDeletedWhileItWaits`（`WaitForLockWaitOn` 看到它等在邀请行上）；`TestUpdateInvitationRole`、`TestDeleteInvitation`（只改那一份；同一个工作区的另一份、别的工作区的不变；删除之后邮箱立即空出来；失败是错误）。
- `adapter/postgres/locks_test.go`：`locks` 加上 `ShareWorkspace`，于是 P2 的只找未删除的、等锁期间被删除、外键检查不等它、失败是错误四个测试都覆盖它；`TestTheWorkspaceLocksConflictAsConvention2Says` 加 6 行（与两把 N 互等，与按 slug 的 S 不等，别的工作区不等）。

- [ ] **Step 1: 查询**

`server/internal/modules/workspace/adapter/postgres/queries/invitations.sql`（修改，1 处）：

````old server/internal/modules/workspace/adapter/postgres/queries/invitations.sql
ORDER BY created_at DESC, id;

````

````new server/internal/modules/workspace/adapter/postgres/queries/invitations.sql
ORDER BY created_at DESC, id;

-- name: InvitationByID :one
-- A write on an invitation reads it before its workspace's lock, for the workspace (M3 design 3.6 convention 2).
SELECT id, workspace_id, email, role, accepted, responded_at, created_by_id, updated_by_id, created_at, updated_at, deleted_at
FROM workspace_member_invites
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

-- name: LockInvitation :one
-- Then, under the workspace's lock, the invitation row FOR UPDATE, read again: a response, a change or a deletion that
-- committed while the write waited is seen, and none commits before it ends. After a wait, Postgres evaluates
-- deleted_at IS NULL again on the row's newest version, so an invitation deleted meanwhile reads no row.
SELECT id, workspace_id, email, role, accepted, responded_at, created_by_id, updated_by_id, created_at, updated_at, deleted_at
FROM workspace_member_invites
WHERE id = sqlc.arg(id) AND deleted_at IS NULL
FOR UPDATE;

-- name: UpdateInvitationRole :one
-- updateWorkspaceInvitation, under the workspace's FOR SHARE and the invitation's FOR UPDATE.
UPDATE workspace_member_invites
SET role = sqlc.arg(role), updated_by_id = sqlc.arg(updated_by), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id)
RETURNING id, workspace_id, email, role, accepted, responded_at, created_by_id, updated_by_id, created_at, updated_at, deleted_at;

-- name: DeleteInvitation :exec
-- deleteWorkspaceInvitation, under the workspace's FOR SHARE and the invitation's FOR UPDATE: the address is free again
-- at once (the partial unique index).
UPDATE workspace_member_invites
SET deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE id = sqlc.arg(id);

````

`server/internal/modules/workspace/adapter/postgres/queries/workspaces.sql`（修改，1 处）：

````old server/internal/modules/workspace/adapter/postgres/queries/workspaces.sql
FOR SHARE;

````

````new server/internal/modules/workspace/adapter/postgres/queries/workspaces.sql
FOR SHARE;

-- name: ShareWorkspace :one
-- ShareWorkspace takes ShareWorkspaceBySlug's lock by the workspace's id: for a write addressed by a row under the
-- workspace that adds or changes a row under it (M3 design 3.6 convention 2).
SELECT id
FROM workspaces
WHERE id = sqlc.arg(id) AND deleted_at IS NULL
FOR SHARE;

````

- [ ] **Step 2: 生成**

Run: `make gen-go`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `272e13d61e9f9209e18c5bb3187d7c02ec4389457f03c3dbdcd0747658888c62` | 230 | `server/internal/modules/workspace/adapter/postgres/gen/invitations.sql.go` |
| `20660dec1ae598952faf2b02820bcd24684b4bc641c27b967a4bb965384f9fde` | 315 | `server/internal/modules/workspace/adapter/postgres/gen/workspaces.sql.go` |

Run: `shasum -a 256 server/internal/modules/workspace/adapter/postgres/gen/invitations.sql.go server/internal/modules/workspace/adapter/postgres/gen/workspaces.sql.go`
Expected: 与上表相同。

- [ ] **Step 3: 存储和它的测试**

`server/internal/modules/workspace/adapter/postgres/invitations.go`（修改，1 处）：

````old server/internal/modules/workspace/adapter/postgres/invitations.go
}

// DeleteWorkspaceInvitations soft-deletes the undeleted invitations of the
````

````new server/internal/modules/workspace/adapter/postgres/invitations.go
}

// InvitationByID returns the undeleted invitation id; app.ErrNotFound when
// there is none.
func (s *Store) InvitationByID(ctx context.Context, id uuid.UUID) (domain.Invitation, error) {
	r, err := s.queries(ctx).InvitationByID(ctx, id)
	if err != nil {
		return domain.Invitation{}, notFound(err)
	}
	return invitation(r), nil
}

// LockInvitation locks the undeleted invitation id FOR UPDATE until the
// transaction ends and returns it; app.ErrNotFound when there is none, also
// when it was deleted while the lock waited.
func (s *Store) LockInvitation(ctx context.Context, id uuid.UUID) (domain.Invitation, error) {
	r, err := s.queries(ctx).LockInvitation(ctx, id)
	if err != nil {
		return domain.Invitation{}, notFound(err)
	}
	return invitation(r), nil
}

// UpdateInvitationRole sets the role of the invitation id, by the account by
// at now, and returns it as stored. The caller holds the invitation's lock:
// its absence is an error.
func (s *Store) UpdateInvitationRole(ctx context.Context, id uuid.UUID, role shared.Role, by uuid.UUID, now time.Time) (domain.Invitation, error) {
	r, err := s.queries(ctx).UpdateInvitationRole(ctx, gen.UpdateInvitationRoleParams{ID: id, Role: int16(role), UpdatedBy: &by, Now: now})
	if err != nil {
		return domain.Invitation{}, fmt.Errorf("update workspace invitation: %w", err)
	}
	return invitation(r), nil
}

// DeleteInvitation soft-deletes the invitation id, by the account by at
// now.
func (s *Store) DeleteInvitation(ctx context.Context, id, by uuid.UUID, now time.Time) error {
	if err := s.queries(ctx).DeleteInvitation(ctx, gen.DeleteInvitationParams{ID: id, DeletedBy: by, Now: now}); err != nil {
		return fmt.Errorf("delete workspace invitation: %w", err)
	}
	return nil
}

// DeleteWorkspaceInvitations soft-deletes the undeleted invitations of the
````

`server/internal/modules/workspace/adapter/postgres/locks.go`（修改，2 处）：

````old server/internal/modules/workspace/adapter/postgres/locks.go
// carries ends; the slug locks return the workspace's id, LockWorkspace,
// which is given it, only an error. A workspace that is not there is
````

````new server/internal/modules/workspace/adapter/postgres/locks.go
// carries ends; the slug locks return the workspace's id, the id locks,
// which are given it, only an error. A workspace that is not there is
````

````old server/internal/modules/workspace/adapter/postgres/locks.go
}

func lockedWorkspace(id uuid.UUID, err error) (uuid.UUID, error) {
````

````new server/internal/modules/workspace/adapter/postgres/locks.go
}

// ShareWorkspace locks the workspace id FOR SHARE: for a write addressed by
// a row under the workspace that adds or changes rows under it.
func (s *Store) ShareWorkspace(ctx context.Context, id uuid.UUID) error {
	_, err := lockedWorkspace(s.queries(ctx).ShareWorkspace(ctx, id))
	return err
}

func lockedWorkspace(id uuid.UUID, err error) (uuid.UUID, error) {
````

`server/internal/modules/workspace/adapter/postgres/locks_test.go`（修改，4 处）：

````old server/internal/modules/workspace/adapter/postgres/locks_test.go
// workspace's id, the one the lock returned or, for LockWorkspace, which
// returns none, the one it was given.
````

````new server/internal/modules/workspace/adapter/postgres/locks_test.go
// workspace's id, the one the lock returned or, for the id locks, which
// return none, the one they were given.
````

````old server/internal/modules/workspace/adapter/postgres/locks_test.go
	locks = []lock{noKeyUpdate, forShare, noKeyUpdateByID}
````

````new server/internal/modules/workspace/adapter/postgres/locks_test.go
	forShareByID = lock{"ShareWorkspace", func(ctx context.Context, s *postgresadapter.Store, w named) (uuid.UUID, error) {
		if err := s.ShareWorkspace(ctx, w.id); err != nil {
			return uuid.UUID{}, err
		}
		return w.id, nil
	}}
	locks = []lock{noKeyUpdate, forShare, noKeyUpdateByID, forShareByID}
````

````old server/internal/modules/workspace/adapter/postgres/locks_test.go
		{noKeyUpdateByID, forShare, "acme", true},
````

````new server/internal/modules/workspace/adapter/postgres/locks_test.go
		{noKeyUpdateByID, forShare, "acme", true},
		{noKeyUpdate, forShareByID, "acme", true},
		{forShareByID, noKeyUpdate, "acme", true},
		{forShareByID, noKeyUpdateByID, "acme", true},
		{forShare, forShareByID, "acme", false},
		{forShareByID, forShare, "acme", false},
````

````old server/internal/modules/workspace/adapter/postgres/locks_test.go
		{noKeyUpdate, noKeyUpdateByID, "beta", false},
````

````new server/internal/modules/workspace/adapter/postgres/locks_test.go
		{noKeyUpdate, noKeyUpdateByID, "beta", false},
		{noKeyUpdate, forShareByID, "beta", false},
````

`server/internal/modules/workspace/adapter/postgres/update_invitation_test.go`（新文件，250 行）：

````file server/internal/modules/workspace/adapter/postgres/update_invitation_test.go
package postgresadapter_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// invitationReads are the two reads of an invitation by its id, the lock
// in a transaction of its own.
func invitationReads(s *postgresadapter.Store, tx *postgres.TxManager) map[string]func(ctx context.Context, id uuid.UUID) (domain.Invitation, error) {
	return map[string]func(ctx context.Context, id uuid.UUID) (domain.Invitation, error){
		"InvitationByID": s.InvitationByID,
		"LockInvitation": func(ctx context.Context, id uuid.UUID) (got domain.Invitation, err error) {
			// The transaction begins on a live context and the lock runs in it
			// on one that is cancelled when ctx is: a cancelled ctx fails the
			// lock's own statement, not the BEGIN.
			err = tx.WithinTx(context.Background(), func(inTx context.Context) error {
				lockCtx, cancel := context.WithCancel(inTx)
				defer cancel()
				if ctx.Err() != nil {
					cancel()
				}
				got, err = s.LockInvitation(lockCtx, id)
				return err
			})
			return got, err
		},
	}
}

// InvitationByID and LockInvitation read an undeleted invitation, pending
// or declined, as stored; app.ErrNotFound for an accepted or deleted one
// and an id no row has. A failed read is its error, never app.ErrNotFound.
func TestTheInvitationReadsFindOnlyAnUndeletedInvitation(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, s, "Acme", "acme", alice)
	pending := invite(t, s, acme.ID, "carol@corp.com", shared.RoleMember, alice)
	declined := invite(t, s, acme.ID, "dave@corp.com", shared.RoleGuest, alice)
	exec(t, pool, "UPDATE workspace_member_invites SET responded_at = $1 WHERE id = $2", now, declined.ID)
	declined.RespondedAt = &now
	accepted := invite(t, s, acme.ID, "erin@corp.com", shared.RoleMember, alice)
	exec(t, pool, "UPDATE workspace_member_invites SET accepted = true, responded_at = $1, deleted_at = $1 WHERE id = $2", now, accepted.ID)
	deleted := invite(t, s, acme.ID, "frank@corp.com", shared.RoleMember, alice)
	exec(t, pool, "UPDATE workspace_member_invites SET deleted_at = $1 WHERE id = $2", now, deleted.ID)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	for name, read := range invitationReads(s, postgres.NewTxManager(pool, 2*time.Second)) {
		for _, want := range []domain.Invitation{pending, declined} {
			if got, err := read(context.Background(), want.ID); err != nil || !sameInvitation(got, want) {
				t.Errorf("%s(%s) = %+v, %v; want %+v", name, want.Email, got, err, want)
			}
		}
		for _, id := range []uuid.UUID{accepted.ID, deleted.ID, uuid.NewV7()} {
			if got, err := read(context.Background(), id); !errors.Is(err, app.ErrNotFound) || got.ID != (uuid.UUID{}) {
				t.Errorf("%s(%s) = %+v, %v; want app.ErrNotFound", name, id, got, err)
			}
		}
		if got, err := read(cancelled, pending.ID); !errors.Is(err, context.Canceled) || errors.Is(err, app.ErrNotFound) || got.ID != (uuid.UUID{}) {
			t.Errorf("%s() on a cancelled context = %+v, %v; want context.Canceled, not app.ErrNotFound", name, got, err)
		}
	}
}

// LockInvitation holds the row FOR UPDATE: another LockInvitation of it
// waits, and so does a FOR KEY SHARE, which a FOR NO KEY UPDATE would let
// pass; another invitation's lock does not wait. A wait ends with
// lock_not_available under a lock_timeout.
func TestLockInvitationLocksTheRowForUpdate(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, s, "Acme", "acme", alice)
	carol := invite(t, s, acme.ID, "carol@corp.com", shared.RoleMember, alice)
	dave := invite(t, s, acme.ID, "dave@corp.com", shared.RoleMember, alice)
	tx := postgres.NewTxManager(pool, 2*time.Second)
	hold(t, tx, func(ctx context.Context) error {
		_, err := s.LockInvitation(ctx, carol.ID)
		return err
	})
	var pgErr *pgconn.PgError
	for name, take := range map[string]func(ctx context.Context) error{
		"LockInvitation": func(ctx context.Context) error {
			_, err := s.LockInvitation(ctx, carol.ID)
			return err
		},
		"FOR KEY SHARE": func(ctx context.Context) error {
			_, err := postgres.DB(ctx, pool).Exec(ctx, "SELECT id FROM workspace_member_invites WHERE id = $1 FOR KEY SHARE", carol.ID)
			return err
		},
	} {
		if err := withLockTimeout(tx, pool, take); !errors.As(err, &pgErr) || pgErr.Code != "55P03" {
			t.Errorf("%s of carol's while it is locked: %v; want lock_not_available after waiting", name, err)
		}
	}
	var got domain.Invitation
	err := withLockTimeout(tx, pool, func(ctx context.Context) error {
		var err error
		got, err = s.LockInvitation(ctx, dave.ID)
		return err
	})
	if err != nil || got.ID != dave.ID {
		t.Errorf("LockInvitation of dave's = %+v, %v; want it without a wait", got, err)
	}
}

// A LockInvitation that waits for the transaction deleting the invitation
// reads no row once that one commits: app.ErrNotFound. WaitForLockWaitOn
// sees it waiting on the invitation's row before the deletion commits.
func TestLockInvitationSkipsAnInvitationDeletedWhileItWaits(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, s, "Acme", "acme", alice)
	carol := invite(t, s, acme.ID, "carol@corp.com", shared.RoleMember, alice)
	tx := postgres.NewTxManager(pool, 2*time.Second)
	commit := hold(t, tx, func(ctx context.Context) error {
		if _, err := s.LockInvitation(ctx, carol.ID); err != nil {
			return err
		}
		return s.DeleteInvitation(ctx, carol.ID, alice, now)
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	answered := make(chan error, 1)
	go func() {
		answered <- tx.WithinTx(ctx, func(ctx context.Context) error {
			_, err := s.LockInvitation(ctx, carol.ID)
			return err
		})
	}()
	pgtest.WaitForLockWaitOn(t, pool, "workspace_member_invites", 5*time.Second)
	if err := commit(); err != nil {
		t.Fatalf("the deletion: %v", err)
	}
	select {
	case err := <-answered:
		if !errors.Is(err, app.ErrNotFound) {
			t.Errorf("LockInvitation() after the deletion = %v; want app.ErrNotFound", err)
		}
	case <-ctx.Done():
		t.Fatal("the lock did not end within 10s")
	}
}

// audit is an invitation's audit columns and deletion time.
type audit struct {
	updatedBy uuid.UUID
	updatedAt time.Time
	deletedAt *time.Time
}

func auditOf(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) audit {
	t.Helper()
	var a audit
	if err := pool.QueryRow(context.Background(), "SELECT updated_by_id, updated_at, deleted_at FROM workspace_member_invites WHERE id = $1", id).
		Scan(&a.updatedBy, &a.updatedAt, &a.deletedAt); err != nil {
		t.Fatal(err)
	}
	return a
}

// UpdateInvitationRole sets the role of that invitation only, with the
// updater and the time, and answers it as stored: another invitation of the
// workspace and one in another workspace keep their role and audit columns.
// A missing row is an error, not app.ErrNotFound; a failed write is its
// error.
func TestUpdateInvitationRole(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme, beta := newWorkspace(t, s, "Acme", "acme", alice), newWorkspace(t, s, "Beta", "beta", alice)
	carol := invite(t, s, acme.ID, "carol@corp.com", shared.RoleMember, alice)
	others := []domain.Invitation{invite(t, s, acme.ID, "dave@corp.com", shared.RoleMember, alice),
		invite(t, s, beta.ID, "carol@corp.com", shared.RoleMember, alice)}
	later := now.Add(time.Hour)

	got, err := s.UpdateInvitationRole(context.Background(), carol.ID, shared.RoleAdmin, bob, later)

	want := carol
	want.Role = shared.RoleAdmin
	if err != nil || !sameInvitation(got, want) {
		t.Errorf("UpdateInvitationRole() = %+v, %v; want %+v", got, err, want)
	}
	if a := auditOf(t, pool, carol.ID); a.updatedBy != bob || !a.updatedAt.Equal(later) || a.deletedAt != nil {
		t.Errorf("carol's: %+v; want updated by bob at %v, not deleted", a, later)
	}
	for _, o := range others {
		if got, err := s.InvitationByID(context.Background(), o.ID); err != nil || !sameInvitation(got, o) {
			t.Errorf("%s in %s: %+v, %v; want it unchanged", o.Email, o.WorkspaceID, got, err)
		}
		if a := auditOf(t, pool, o.ID); a.updatedBy != alice || !a.updatedAt.Equal(now) {
			t.Errorf("%s in %s: %+v; want updated by alice at %v", o.Email, o.WorkspaceID, a, now)
		}
	}
	if _, err := s.UpdateInvitationRole(context.Background(), uuid.NewV7(), shared.RoleGuest, bob, later); err == nil || errors.Is(err, app.ErrNotFound) {
		t.Errorf("UpdateInvitationRole() of no row = %v, want an error that is not app.ErrNotFound", err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if got, err := s.UpdateInvitationRole(cancelled, carol.ID, shared.RoleGuest, bob, later); !errors.Is(err, context.Canceled) || got.ID != (uuid.UUID{}) {
		t.Errorf("UpdateInvitationRole() on a cancelled context = %+v, %v; want context.Canceled", got, err)
	}
}

// DeleteInvitation soft-deletes that invitation only, with the deleter and
// the time; its address is free again at once. Another invitation of the
// workspace and one in another workspace stay. A failed write is its
// error.
func TestDeleteInvitation(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme, beta := newWorkspace(t, s, "Acme", "acme", alice), newWorkspace(t, s, "Beta", "beta", alice)
	carol := invite(t, s, acme.ID, "carol@corp.com", shared.RoleMember, alice)
	invite(t, s, acme.ID, "dave@corp.com", shared.RoleMember, alice)
	invite(t, s, beta.ID, "carol@corp.com", shared.RoleMember, alice)
	later := now.Add(time.Hour)

	if err := s.DeleteInvitation(context.Background(), carol.ID, bob, later); err != nil {
		t.Fatalf("DeleteInvitation() = %v", err)
	}

	if a := auditOf(t, pool, carol.ID); a.updatedBy != bob || !a.updatedAt.Equal(later) || a.deletedAt == nil || !a.deletedAt.Equal(later) {
		t.Errorf("carol's: %+v; want deleted by bob at %v", a, later)
	}
	if got := pendingEmails(t, pool, acme.ID); !slices.Equal(got, []string{"dave@corp.com"}) {
		t.Errorf("acme's invitations: %q; want dave's", got)
	}
	if got := pendingEmails(t, pool, beta.ID); !slices.Equal(got, []string{"carol@corp.com"}) {
		t.Errorf("beta's invitations: %q; want carol's", got)
	}
	invite(t, s, acme.ID, "carol@corp.com", shared.RoleGuest, alice)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := s.DeleteInvitation(cancelled, carol.ID, bob, later); !errors.Is(err, context.Canceled) {
		t.Errorf("DeleteInvitation() on a cancelled context = %v; want context.Canceled", err)
	}
}
````

- [ ] **Step 4: 测试、lint**

Run: `go -C server test -count=1 ./internal/modules/workspace/adapter/postgres/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 5: 提交**

```bash
git add server/internal/modules/workspace/adapter/postgres/invitations.go server/internal/modules/workspace/adapter/postgres/locks.go server/internal/modules/workspace/adapter/postgres/locks_test.go server/internal/modules/workspace/adapter/postgres/queries/invitations.sql server/internal/modules/workspace/adapter/postgres/queries/workspaces.sql server/internal/modules/workspace/adapter/postgres/update_invitation_test.go server/internal/modules/workspace/adapter/postgres/gen/invitations.sql.go server/internal/modules/workspace/adapter/postgres/gen/workspaces.sql.go
```
```bash
git commit -m "feat(M3/P3): the store reads, locks, changes and deletes an invitation

InvitationByID reads an invitation for its workspace; LockInvitation
locks it FOR UPDATE under the workspace's lock and reads it again, none
once it was deleted while the lock waited; ShareWorkspace takes the
workspace's FOR SHARE by id (M3 design 3.6 convention 2). Changing the
role and deleting touch that invitation only.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（改生成的查询常量，等于改 `.sql` 再生成；spec 附录 A）：

| 改坏 | 必须失败的测试 |
|---|---|
| `LockInvitation` 去掉 `deleted_at IS NULL` | `TestTheInvitationReadsFindOnlyAnUndeletedInvitation`、`TestLockInvitationSkipsAnInvitationDeletedWhileItWaits` |
| `LockInvitation` 用 `FOR SHARE`；用 `FOR NO KEY UPDATE` | `TestLockInvitationLocksTheRowForUpdate` |
| `InvitationByID` 读已删除的 | `TestTheInvitationReadsFindOnlyAnUndeletedInvitation` |
| `ShareWorkspace` 去掉 `deleted_at IS NULL` | `TestTheWorkspaceLocksFindOnlyAnUndeletedWorkspace`、`TestTheWorkspaceLocksSkipAWorkspaceDeletedWhileTheyWait` |
| `ShareWorkspace` 用 `FOR NO KEY UPDATE`；用 `FOR UPDATE` | `TestTheWorkspaceLocksConflictAsConvention2Says`（后者另有 `TestAForeignKeyCheckDoesNotWaitForTheWorkspaceLocks`） |
| `UpdateInvitationRole`、`DeleteInvitation` 改每一行 | `TestUpdateInvitationRole`、`TestDeleteInvitation` |
| `UpdateInvitationRole`、`DeleteInvitation` 吞掉错误 | `TestUpdateInvitationRole`、`TestDeleteInvitation` |
| `LockInvitation` 把失败答成"没有" | `TestTheInvitationReadsFindOnlyAnUndeletedInvitation` |

**Done when:** 5 个存储测试和扩展的锁测试在真实数据库上通过，每种冲突都在 `lock_timeout` 之下确定地得到；生成物的 SHA-256 与表相同。

---

### Task 8: `updateWorkspaceInvitation`、`deleteWorkspaceInvitation`

**Files:**
- Create: `server/internal/modules/workspace/app/delete_invitation.go`、`server/internal/modules/workspace/app/delete_invitation_test.go`、`server/internal/modules/workspace/app/invitation_lock.go`、`server/internal/modules/workspace/app/update_invitation.go`、`server/internal/modules/workspace/app/update_invitation_test.go`
- Modify: `api/modules/workspace.yaml`、`api/openapi.yaml`、`server/internal/bootstrap/permission_matrix_invitations_test.go`、`server/internal/bootstrap/permission_matrix_workspace_test.go`、`server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`、`server/internal/modules/workspace/adapter/http/handler.go`、`server/internal/modules/workspace/adapter/http/handler_test.go`、`server/internal/modules/workspace/adapter/http/invitations.go`、`server/internal/modules/workspace/adapter/http/invitations_test.go`、`server/internal/modules/workspace/app/clock_test.go`、`server/internal/modules/workspace/app/fakes_invitations_test.go`、`server/internal/modules/workspace/app/fakes_test.go`、`server/internal/modules/workspace/app/invitation_ports.go`、`server/internal/modules/workspace/domain/actions.go`、`server/internal/modules/workspace/domain/errors.go`、`server/internal/modules/workspace/domain/invitation.go`、`server/internal/modules/workspace/module.go`、`web/apps/web/helpers/authentication.helper.ts`、`web/packages/i18n/src/locales/en/auth.json`、`web/packages/i18n/src/locales/zh-CN/auth.json`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/workspace/adapter/http/gen/bodyshape.gen.go`、`server/internal/modules/workspace/adapter/http/gen/server.gen.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Produces（spec 2.9，M3 设计 3.6、3.8、5.1、5.3）：接口 `PATCH`、`DELETE /api/v0/workspace-invitations/{invitation_id}`，`WorkspaceInvitationUpdate{role}`；新码 `workspace.invitation_not_found`（404）、`workspace.invitation_responded`（409）和它们的文案；`domain.ActionInvitationUpdate`、`ActionInvitationDelete`，规则 `workspace_invitation.update`、`.delete`：管理员；`app.InvitationLocker`、`InvitationUpdater`、`InvitationDeleter`；`readInvitation`、`lockInvitation`、`invitationNotFound`（`app/invitation_lock.go`）；`NewUpdateWorkspaceInvitation(invitations, auth, tx, clock, mac)`、`NewDeleteWorkspaceInvitation(invitations, auth, tx, clock)`。
- 使用者：Task 11 的回应用 `readInvitation`、`lockInvitation`；Task 14 的 W4。

**Tests:**
- `app/update_invitation_test.go`：`TestUpdateWorkspaceInvitationLocksThenDecidesThenWrites`（读 → 工作区 `FOR SHARE` → 锁邀请 → 判定 → 写角色，一个事务；回答是存下的行加令牌；两位管理员、两个工作区）；`TestUpdateWorkspaceInvitationRefusals`（角色的检查最先；不在、期间被删除、期间工作区被删除、看不到都是 `invitation_not_found`；成员的 `forbidden` 在邀请的检查之前；之后已忽略的（等锁期间才忽略的也算）`invitation_responded`；失败原样、不是 404）。
- `app/delete_invitation_test.go`：`TestDeleteWorkspaceInvitationLocksThenDecidesThenDeletes`（已忽略的照样删除；别的邀请不动）；`TestDeleteWorkspaceInvitationRefusals`。
- `app/clock_test.go`：`TestEachWriteReadsTheClockUnderItsLock` 加修改、删除两行。
- `adapter/http/invitations_test.go`：`TestUpdateWorkspaceInvitation`、`TestUpdateWorkspaceInvitationRefusals`（请求体没有角色、id 不是 UUID 在用例之前 400）、`TestDeleteWorkspaceInvitation`（204 没有正文）。
- 矩阵：修改、删除两行（管理员 200/204，成员、访客 403，另三列 404 `invitation_not_found`），`promotesTheNewcomer` 核对答案。

- [ ] **Step 1: 接口描述**

`api/modules/workspace.yaml`（修改，3 处）：

````old api/modules/workspace.yaml
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/workspace-slugs/{slug}:
````

````new api/modules/workspace.yaml
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/workspace-invitations/{invitation_id}:
    parameters:
      - $ref: '#/components/parameters/InvitationID'
    patch:
      operationId: updateWorkspaceInvitation
      tags: [workspace]
      summary: Change an invitation's role
      description: >-
        For the workspace's admins. The role is checked first
        (validation_failed). An invitation that does not exist, is deleted
        or accepted, or whose workspace the caller cannot see, answers
        workspace.invitation_not_found; a member or a guest, forbidden,
        whatever the invitation. To an admin, a declined invitation answers
        workspace.invitation_responded: delete it, then invite the address
        again. The link keeps its token.
      security: [{bearer: []}]
      x-problem-codes: [validation_failed, workspace.invitation_not_found, forbidden, workspace.invitation_responded]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/WorkspaceInvitationUpdate'
      responses:
        '200':
          description: The invitation with its new role, and the token of its link.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/WorkspaceInvitation'
        default:
          $ref: '#/components/responses/Problem'
    delete:
      operationId: deleteWorkspaceInvitation
      tags: [workspace]
      summary: Delete an invitation
      description: >-
        For the workspace's admins. The invitation, pending or declined, is
        soft-deleted: its link stops working, and its address can be invited
        again at once. An invitation that does not exist, is deleted or
        accepted, or whose workspace the caller cannot see, answers
        workspace.invitation_not_found; a member or a guest, forbidden.
      security: [{bearer: []}]
      x-problem-codes: [workspace.invitation_not_found, forbidden]
      responses:
        '204':
          description: The invitation is deleted.
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/workspace-slugs/{slug}:
````

````old api/modules/workspace.yaml
      schema:
        type: string
````

````new api/modules/workspace.yaml
      schema:
        type: string
    InvitationID:
      name: invitation_id
      in: path
      required: true
      description: An invitation's id (WorkspaceInvitation.id), the link's invitation_id.
      schema:
        type: string
        format: uuid
````

````old api/modules/workspace.yaml
        role:
          $ref: '#/components/schemas/WorkspaceRole'
    WorkspaceInvitationsCreate:
````

````new api/modules/workspace.yaml
        role:
          $ref: '#/components/schemas/WorkspaceRole'
    WorkspaceInvitationUpdate:
      type: object
      additionalProperties: false
      required: [role]
      properties:
        role:
          $ref: '#/components/schemas/WorkspaceRole'
    WorkspaceInvitationsCreate:
````

`api/openapi.yaml`（修改，1 处）：

````old api/openapi.yaml
    $ref: 'modules/workspace.yaml#/paths/~1api~1v0~1workspace-members~1{workspace_member_id}'
````

````new api/openapi.yaml
    $ref: 'modules/workspace.yaml#/paths/~1api~1v0~1workspace-members~1{workspace_member_id}'
  /api/v0/workspace-invitations/{invitation_id}:
    $ref: 'modules/workspace.yaml#/paths/~1api~1v0~1workspace-invitations~1{invitation_id}'
````

- [ ] **Step 2: 生成**

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `f1dbf5869d4023afe2a644e2d99a9c83fa071906ecea68f1c4544fe39602ea46` | 1548 | `api/dist/openapi.yaml` |
| `96d0f59a97c4c0d4ad3cc37597156d2bcb4a5fac0d583a4ff5e8bb1c73abc6bb` | 41 | `server/internal/modules/workspace/adapter/http/gen/bodyshape.gen.go` |
| `66c0d71f6aa60080dc8cdee5718c4f8d0b19eb973396eff708c3e4fccecd6473` | 1996 | `server/internal/modules/workspace/adapter/http/gen/server.gen.go` |
| `08bac7a0fc56689a6ccc57ba3a0e6d7d9e4044a57b0897a19b6c1bc7dc79a31f` | 1642 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/workspace/adapter/http/gen/bodyshape.gen.go server/internal/modules/workspace/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 2b: 新码和文案**

`server/internal/modules/workspace/domain/errors.go`（修改，1 处）：

````old server/internal/modules/workspace/domain/errors.go
	ErrOwnMembership = shared.NewError(shared.KindConflict, "workspace.own_membership", "You cannot change your own membership.")
````

````new server/internal/modules/workspace/domain/errors.go
	ErrOwnMembership = shared.NewError(shared.KindConflict, "workspace.own_membership", "You cannot change your own membership.")
	// ErrInvitationNotFound answers an invitation that does not exist or is
	// deleted, as an accepted one is, or whose workspace the caller cannot
	// see; for the invitee, also a token that is not the invitation's: the
	// same 404 for all (M3 design 5.3, 8.2).
	ErrInvitationNotFound = shared.NewError(shared.KindNotFound, "workspace.invitation_not_found",
		"The invitation does not exist, or its link is not valid.")
	// ErrInvitationResponded answers a response to, or a change of, an
	// invitation that has been declined (M3 design 3.8).
	ErrInvitationResponded = shared.NewError(shared.KindConflict, "workspace.invitation_responded", "The invitation has been answered already.")
````

`web/apps/web/helpers/authentication.helper.ts`（修改，1 处）：

````old web/apps/web/helpers/authentication.helper.ts
  "workspace.own_membership": "auth.errors.workspace_own_membership",
````

````new web/apps/web/helpers/authentication.helper.ts
  "workspace.own_membership": "auth.errors.workspace_own_membership",
  "workspace.invitation_not_found": "auth.errors.workspace_invitation_not_found",
  "workspace.invitation_responded": "auth.errors.workspace_invitation_responded",
````

`web/packages/i18n/src/locales/en/auth.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/auth.json
      "workspace_own_membership": "You cannot change your own membership.",
````

````new web/packages/i18n/src/locales/en/auth.json
      "workspace_own_membership": "You cannot change your own membership.",
      "workspace_invitation_not_found": "The invitation does not exist, or its link is not valid.",
      "workspace_invitation_responded": "The invitation has been answered already.",
````

`web/packages/i18n/src/locales/zh-CN/auth.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/auth.json
      "workspace_own_membership": "不能修改自己的成员身份。",
````

````new web/packages/i18n/src/locales/zh-CN/auth.json
      "workspace_own_membership": "不能修改自己的成员身份。",
      "workspace_invitation_not_found": "邀请不存在，或链接无效。",
      "workspace_invitation_responded": "这份邀请已经回应过了。",
````

- [ ] **Step 3: 端口、按 id 寻址的锁、用例**

`server/internal/modules/workspace/domain/invitation.go`（修改，1 处）：

````old server/internal/modules/workspace/domain/invitation.go
	CreatedByID *uuid.UUID
````

````new server/internal/modules/workspace/domain/invitation.go
	CreatedByID *uuid.UUID
}

// Responded reports whether the invitation has been answered: declined,
// for an undeleted one (M3 design 3.8).
func (i Invitation) Responded() bool {
	return i.RespondedAt != nil
````

`server/internal/modules/workspace/domain/actions.go`（修改，2 处）：

````old server/internal/modules/workspace/domain/actions.go
	ActionInvitationCreate shared.Action = "workspace_invitation.create"
````

````new server/internal/modules/workspace/domain/actions.go
	ActionInvitationCreate shared.Action = "workspace_invitation.create"
	// ActionInvitationUpdate is changing an invitation's role:
	// updateWorkspaceInvitation.
	ActionInvitationUpdate shared.Action = "workspace_invitation.update"
	// ActionInvitationDelete is deleting an invitation:
	// deleteWorkspaceInvitation.
	ActionInvitationDelete shared.Action = "workspace_invitation.delete"
````

````old server/internal/modules/workspace/domain/actions.go
		ActionInvitationList, ActionInvitationCreate}
````

````new server/internal/modules/workspace/domain/actions.go
		ActionInvitationList, ActionInvitationCreate, ActionInvitationUpdate, ActionInvitationDelete}
````

`server/internal/modules/workspace/app/invitation_ports.go`（修改，1 处）：

````old server/internal/modules/workspace/app/invitation_ports.go
}

// DuplicateInvitation is a store's answer to an insert that the unique key
````

````new server/internal/modules/workspace/app/invitation_ports.go
}

// InvitationLocker reads an invitation by its id, then locks it under its
// workspace's lock (M3 design 3.6 convention 2).
type InvitationLocker interface {
	// InvitationByID returns the undeleted invitation id; ErrNotFound when
	// there is none.
	InvitationByID(ctx context.Context, id uuid.UUID) (domain.Invitation, error)
	// LockInvitation locks the undeleted invitation id FOR UPDATE until the
	// transaction ends and returns it; ErrNotFound when there is none, also
	// when it was deleted while the lock waited.
	LockInvitation(ctx context.Context, id uuid.UUID) (domain.Invitation, error)
}

// InvitationUpdater changes an invitation's role under its workspace's FOR
// SHARE.
type InvitationUpdater interface {
	InvitationLocker
	// ShareWorkspace locks the undeleted workspace id FOR SHARE until the
	// transaction ends; ErrNotFound when there is none, also when it was
	// deleted while the lock waited.
	ShareWorkspace(ctx context.Context, id uuid.UUID) error
	// UpdateInvitationRole sets the invitation's role, by the account by at
	// now, and returns it as stored.
	UpdateInvitationRole(ctx context.Context, id uuid.UUID, role shared.Role, by uuid.UUID, now time.Time) (domain.Invitation, error)
}

// InvitationDeleter deletes an invitation under its workspace's FOR SHARE.
type InvitationDeleter interface {
	InvitationLocker
	// ShareWorkspace is InvitationUpdater's.
	ShareWorkspace(ctx context.Context, id uuid.UUID) error
	// DeleteInvitation soft-deletes the invitation, by the account by at
	// now.
	DeleteInvitation(ctx context.Context, id, by uuid.UUID, now time.Time) error
}

// DuplicateInvitation is a store's answer to an insert that the unique key
````

`server/internal/modules/workspace/app/invitation_lock.go`（新文件，50 行）：

````file server/internal/modules/workspace/app/invitation_lock.go
package app

import (
	"context"
	"errors"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
)

// A write on an invitation named by its id (M3 design 3.6 convention 2)
// reads the invitation for its workspace (readInvitation), locks the
// workspace, then locks the invitation row FOR UPDATE and reads it again
// (lockInvitation): its decisions and checks see what committed before, and
// nothing changes the row until the write ends. Between the two, a response
// locks the caller's account row (3.6 conventions 1 and 6).

// readInvitation reads the undeleted invitation id, without a lock;
// domain.ErrInvitationNotFound when there is none.
func readInvitation(ctx context.Context, invitations InvitationLocker, id uuid.UUID) (domain.Invitation, error) {
	inv, err := invitations.InvitationByID(ctx, id)
	if err != nil {
		return domain.Invitation{}, invitationNotFound(err)
	}
	return inv, nil
}

// lockInvitation locks inv's workspace with lockWorkspace, then inv's row
// FOR UPDATE, and returns the row read under the lock. A workspace deleted,
// or an invitation deleted (by an acceptance too) while the locks waited,
// is domain.ErrInvitationNotFound.
func lockInvitation(ctx context.Context, invitations InvitationLocker, lockWorkspace func(ctx context.Context, id uuid.UUID) error,
	inv domain.Invitation) (domain.Invitation, error) {
	if err := lockWorkspace(ctx, inv.WorkspaceID); err != nil {
		return domain.Invitation{}, invitationNotFound(err)
	}
	locked, err := invitations.LockInvitation(ctx, inv.ID)
	if err != nil {
		return domain.Invitation{}, invitationNotFound(err)
	}
	return locked, nil
}

// invitationNotFound turns ErrNotFound into domain.ErrInvitationNotFound.
func invitationNotFound(err error) error {
	if errors.Is(err, ErrNotFound) {
		return domain.ErrInvitationNotFound
	}
	return err
}
````

`server/internal/modules/workspace/app/update_invitation.go`（新文件，64 行）：

````file server/internal/modules/workspace/app/update_invitation.go
package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// UpdateWorkspaceInvitation changes an invitation's role: PATCH
// /api/v0/workspace-invitations/{invitation_id}.
type UpdateWorkspaceInvitation struct {
	invitations InvitationUpdater
	auth        shared.Authorizer
	tx          shared.TxManager
	clock       Clock
	tokens      invitationTokens
}

// NewUpdateWorkspaceInvitation returns the use case.
func NewUpdateWorkspaceInvitation(invitations InvitationUpdater, auth shared.Authorizer, tx shared.TxManager, clock Clock,
	mac InvitationMAC) *UpdateWorkspaceInvitation {
	return &UpdateWorkspaceInvitation{invitations: invitations, auth: auth, tx: tx, clock: clock, tokens: invitationTokens{mac: mac}}
}

// Execute checks role, then in one transaction (M3 design 3.6): the
// invitation read for its workspace, the workspace FOR SHARE, the invitation
// FOR UPDATE read again, the decision on workspace_invitation.update, then,
// for a caller allowed to change it, a declined invitation is
// workspace.invitation_responded; then the change, at the time the clock
// gives under the locks. Only admins change invitations, so the role is
// never above the changer's (3.8). The answer carries the token.
func (u *UpdateWorkspaceInvitation) Execute(ctx context.Context, id uuid.UUID, role shared.Role) (domain.InvitationWithToken, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.InvitationWithToken{}, err
	}
	if err := domain.CheckMemberRole(role); err != nil {
		return domain.InvitationWithToken{}, err
	}
	var updated domain.Invitation
	err = u.tx.WithinTx(ctx, func(ctx context.Context) error {
		inv, err := readInvitation(ctx, u.invitations, id)
		if err != nil {
			return err
		}
		if inv, err = lockInvitation(ctx, u.invitations, u.invitations.ShareWorkspace, inv); err != nil {
			return err
		}
		if _, err := decide(ctx, u.auth, actor, domain.ActionInvitationUpdate, inv.WorkspaceID, domain.ErrInvitationNotFound); err != nil {
			return err
		}
		if inv.Responded() {
			return domain.ErrInvitationResponded
		}
		updated, err = u.invitations.UpdateInvitationRole(ctx, inv.ID, role, actor.UserID, u.clock.Now())
		return err
	})
	if err != nil {
		return domain.InvitationWithToken{}, err
	}
	return domain.InvitationWithToken{Invitation: updated, Token: u.tokens.of(updated.ID)}, nil
}
````

`server/internal/modules/workspace/app/delete_invitation.go`（新文件，49 行）：

````file server/internal/modules/workspace/app/delete_invitation.go
package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// DeleteWorkspaceInvitation deletes an invitation: DELETE
// /api/v0/workspace-invitations/{invitation_id}.
type DeleteWorkspaceInvitation struct {
	invitations InvitationDeleter
	auth        shared.Authorizer
	tx          shared.TxManager
	clock       Clock
}

// NewDeleteWorkspaceInvitation returns the use case.
func NewDeleteWorkspaceInvitation(invitations InvitationDeleter, auth shared.Authorizer, tx shared.TxManager, clock Clock) *DeleteWorkspaceInvitation {
	return &DeleteWorkspaceInvitation{invitations: invitations, auth: auth, tx: tx, clock: clock}
}

// Execute deletes the invitation in one transaction (M3 design 3.6): read
// for its workspace, the workspace FOR SHARE, the invitation FOR UPDATE read
// again, the decision on workspace_invitation.delete, then the soft
// deletion, at the time the clock gives under the locks. A declined
// invitation is deleted as a pending one is, and its address is free again
// (3.8); deleting and inviting again is how a leaked link is voided.
func (u *DeleteWorkspaceInvitation) Execute(ctx context.Context, id uuid.UUID) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	return u.tx.WithinTx(ctx, func(ctx context.Context) error {
		inv, err := readInvitation(ctx, u.invitations, id)
		if err != nil {
			return err
		}
		if inv, err = lockInvitation(ctx, u.invitations, u.invitations.ShareWorkspace, inv); err != nil {
			return err
		}
		if _, err := decide(ctx, u.auth, actor, domain.ActionInvitationDelete, inv.WorkspaceID, domain.ErrInvitationNotFound); err != nil {
			return err
		}
		return u.invitations.DeleteInvitation(ctx, inv.ID, actor.UserID, u.clock.Now())
	})
}
````

`server/internal/modules/workspace/app/fakes_test.go`（修改，1 处）：

````old server/internal/modules/workspace/app/fakes_test.go
	f.log.add(ctx, "LockWorkspace %s", id)
````

````new server/internal/modules/workspace/app/fakes_test.go
	f.log.add(ctx, "LockWorkspace %s", id)
	return f.lockByID(id)
}

// lockByID answers as lock does, for the workspace with id; once locked, it
// runs onLock.
func (f *fakeWorkspaces) lockByID(id uuid.UUID) error {
````

`server/internal/modules/workspace/app/fakes_invitations_test.go`（修改，2 处）：

````old server/internal/modules/workspace/app/fakes_invitations_test.go
	"fmt"
````

````new server/internal/modules/workspace/app/fakes_invitations_test.go
	"fmt"
	"slices"
````

````old server/internal/modules/workspace/app/fakes_invitations_test.go
	taken       string // an address CreateInvitations finds taken, as the unique key would
````

````new server/internal/modules/workspace/app/fakes_invitations_test.go
	taken       string // an address CreateInvitations finds taken, as the unique key would
	readErr     error  // for InvitationByID and LockInvitation
	writeErr    error  // for UpdateInvitationRole and DeleteInvitation
}

func (f *fakeInvitations) InvitationByID(ctx context.Context, id uuid.UUID) (domain.Invitation, error) {
	f.log.add(ctx, "InvitationByID %s", id)
	return f.invitation(id)
}

func (f *fakeInvitations) LockInvitation(ctx context.Context, id uuid.UUID) (domain.Invitation, error) {
	f.log.add(ctx, "LockInvitation %s", id)
	return f.invitation(id)
}

// invitation answers the invitation id it holds, app.ErrNotFound when it
// holds none, and readErr wrapped as the store wraps it.
func (f *fakeInvitations) invitation(id uuid.UUID) (domain.Invitation, error) {
	if f.readErr != nil {
		return domain.Invitation{}, fmt.Errorf("read workspace invitation: %w", f.readErr)
	}
	i := slices.IndexFunc(f.invitations, func(inv domain.Invitation) bool { return inv.ID == id })
	if i < 0 {
		return domain.Invitation{}, app.ErrNotFound
	}
	return f.invitations[i], nil
}

// ShareWorkspace answers as LockWorkspace does.
func (f *fakeInvitations) ShareWorkspace(ctx context.Context, id uuid.UUID) error {
	f.log.add(ctx, "ShareWorkspace %s", id)
	return f.lockByID(id)
}

// UpdateInvitationRole sets the role of the invitation it holds and answers
// it as stored.
func (f *fakeInvitations) UpdateInvitationRole(ctx context.Context, id uuid.UUID, role shared.Role, by uuid.UUID, now time.Time) (domain.Invitation, error) {
	f.log.add(ctx, "UpdateInvitationRole %s to %d by %s at %s", id, role, by, now.Format(time.RFC3339Nano))
	if f.writeErr != nil {
		return domain.Invitation{}, fmt.Errorf("update workspace invitation: %w", f.writeErr)
	}
	i := slices.IndexFunc(f.invitations, func(inv domain.Invitation) bool { return inv.ID == id })
	if i < 0 {
		return domain.Invitation{}, fmt.Errorf("update workspace invitation %s: no such row", id)
	}
	f.invitations[i].Role = role
	return f.invitations[i], nil
}

// DeleteInvitation drops the invitation it holds.
func (f *fakeInvitations) DeleteInvitation(ctx context.Context, id, by uuid.UUID, now time.Time) error {
	f.log.add(ctx, "DeleteInvitation %s by %s at %s", id, by, now.Format(time.RFC3339Nano))
	if f.writeErr != nil {
		return fmt.Errorf("delete workspace invitation: %w", f.writeErr)
	}
	f.invitations = slices.DeleteFunc(f.invitations, func(inv domain.Invitation) bool { return inv.ID == id })
	return nil
````

`server/internal/modules/workspace/app/update_invitation_test.go`（新文件，140 行）：

````file server/internal/modules/workspace/app/update_invitation_test.go
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

// lockedInvitationCalls are the calls up to the decision on action over
// the invitation inv for user: the read, the workspace's FOR SHARE, the
// invitation's lock, the decision.
func lockedInvitationCalls(user app.AccountState, inv domain.Invitation, action shared.Action) []string {
	return []string{
		"InvitationByID " + inv.ID.String(),
		"ShareWorkspace " + inv.WorkspaceID.String(),
		"LockInvitation " + inv.ID.String(),
		"Authorize " + user.ID.String() + " " + string(action) + " on " + inv.WorkspaceID.String() + "/" + uuid.Nil().String(),
	}
}

func (f *invitationsFixture) update() *app.UpdateWorkspaceInvitation {
	return app.NewUpdateWorkspaceInvitation(f.invitations, f.auth, f.tx, clockAt{at: clockNow}, f.mac)
}

// UpdateWorkspaceInvitation reads the invitation, locks its workspace FOR
// SHARE, locks the invitation, decides, then writes the role, all in one
// transaction (M3 design 3.6); the answer is the row as stored, with its
// token. Two admins, two workspaces.
func TestUpdateWorkspaceInvitationLocksThenDecidesThenWrites(t *testing.T) {
	for _, tt := range []struct {
		user app.AccountState
		inv  domain.Invitation
		role shared.Role
	}{{alice, carolToAcme, shared.RoleGuest}, {alice, carolToAcme, shared.RoleAdmin}, {bob, erinToBeta, shared.RoleMember}} {
		f := newInvitations()
		got, err := f.update().Execute(as(tt.user), tt.inv.ID, tt.role)
		want := tt.inv
		want.Role = tt.role
		if err != nil || !sameInvitations([]domain.InvitationWithToken{got}, []domain.InvitationWithToken{withToken(f.mac, want)}) {
			t.Errorf("%s to %d: Execute() = %+v, %v; want %+v", tt.inv.Email, tt.role, got, err, withToken(f.mac, want))
		}
		wantCalls := append(lockedInvitationCalls(tt.user, tt.inv, domain.ActionInvitationUpdate),
			fmt.Sprintf("UpdateInvitationRole %s to %d by %s at %s", tt.inv.ID, tt.role, tt.user.ID, clockNow.Format(time.RFC3339Nano)))
		if !slices.Equal(f.log.calls, wantCalls) || f.tx.calls != 1 {
			t.Errorf("%s to %d: calls = %q in %d transactions, want %q in one", tt.inv.Email, tt.role, f.log.calls, f.tx.calls, wantCalls)
		}
	}
}

// Each refusal and failure is the answer, and no role is written. The
// role's check comes first. An invitation not there, deleted meanwhile, of
// a workspace deleted meanwhile or not visible is
// workspace.invitation_not_found; a member's forbidden comes before any
// check of the invitation; then a declined invitation, also declined while
// the lock waited, is workspace.invitation_responded. A failure is itself,
// never a 404.
func TestUpdateWorkspaceInvitationRefusals(t *testing.T) {
	failure := errors.New("connection reset")
	forbidBob := func(f *invitationsFixture) { f.auth.errs = map[grantKey]error{{bob.ID, acme.ID}: shared.Forbidden()} }
	decided := lockedInvitationCalls(alice, carolToAcme, domain.ActionInvitationUpdate)
	stranger := uuid.NewV7()
	tests := []struct {
		name  string
		user  app.AccountState
		id    uuid.UUID
		role  shared.Role
		set   func(f *invitationsFixture)
		want  error
		calls []string
	}{
		{"a role outside the three", alice, carolToAcme.ID, 10, nil,
			shared.Invalid(shared.FieldError{Field: "role", Code: shared.FieldInvalidFormat, Message: "is not 5, 15 or 20"}), nil},
		{"no such invitation", alice, stranger, shared.RoleGuest, nil, domain.ErrInvitationNotFound, []string{"InvitationByID " + stranger.String()}},
		{"acme deleted while the lock waited", alice, carolToAcme.ID, shared.RoleGuest,
			func(f *invitationsFixture) { f.invitations.lockErrs = map[string]error{"acme": app.ErrNotFound} }, domain.ErrInvitationNotFound,
			decided[:2]},
		{"deleted while the lock waited", alice, carolToAcme.ID, shared.RoleGuest,
			func(f *invitationsFixture) {
				f.invitations.onLock = func() { f.invitations.invitations = []domain.Invitation{daveToAcme, erinToBeta} }
			},
			domain.ErrInvitationNotFound, decided[:3]},
		{"not visible", carol, carolToAcme.ID, shared.RoleGuest, nil, domain.ErrInvitationNotFound,
			lockedInvitationCalls(carol, carolToAcme, domain.ActionInvitationUpdate)},
		{"a member", bob, carolToAcme.ID, shared.RoleGuest, forbidBob, shared.Forbidden(),
			lockedInvitationCalls(bob, carolToAcme, domain.ActionInvitationUpdate)},
		{"a member, of a declined one", bob, daveToAcme.ID, shared.RoleGuest, forbidBob, shared.Forbidden(),
			lockedInvitationCalls(bob, daveToAcme, domain.ActionInvitationUpdate)},
		{"a declined one", alice, daveToAcme.ID, shared.RoleMember, nil, domain.ErrInvitationResponded,
			lockedInvitationCalls(alice, daveToAcme, domain.ActionInvitationUpdate)},
		{"declined while the lock waited", alice, carolToAcme.ID, shared.RoleGuest,
			func(f *invitationsFixture) {
				f.invitations.onLock = func() { f.invitations.invitations[0].RespondedAt = &declined }
			},
			domain.ErrInvitationResponded, decided},
		{"the read failed", alice, carolToAcme.ID, shared.RoleGuest, func(f *invitationsFixture) { f.invitations.readErr = failure }, failure,
			decided[:1]},
		{"the workspace's lock failed", alice, carolToAcme.ID, shared.RoleGuest,
			func(f *invitationsFixture) { f.invitations.lockErrs = map[string]error{"acme": failure} }, failure, decided[:2]},
		{"the invitation's lock failed", alice, carolToAcme.ID, shared.RoleGuest,
			func(f *invitationsFixture) { f.invitations.onLock = func() { f.invitations.readErr = failure } }, failure, decided[:3]},
		{"the Authorizer failed", alice, carolToAcme.ID, shared.RoleGuest,
			func(f *invitationsFixture) { f.auth.errs = map[grantKey]error{{alice.ID, acme.ID}: failure} }, failure, decided},
		{"the write failed", alice, carolToAcme.ID, shared.RoleGuest, func(f *invitationsFixture) { f.invitations.writeErr = failure }, failure,
			append(decided, fmt.Sprintf("UpdateInvitationRole %s to %d by %s at %s", carolToAcme.ID, shared.RoleGuest, alice.ID,
				clockNow.Format(time.RFC3339Nano)))},
	}
	for _, tt := range tests {
		f := newInvitations()
		if tt.set != nil {
			tt.set(f)
		}
		got, err := f.update().Execute(as(tt.user), tt.id, tt.role)
		if !errors.Is(err, tt.want) || got != (domain.InvitationWithToken{}) {
			t.Errorf("%s: Execute() = %+v, %v; want no invitation and %v", tt.name, got, err, tt.want)
		}
		if tt.want == failure && errors.Is(err, domain.ErrInvitationNotFound) {
			t.Errorf("%s: Execute() = %v, which is also workspace.invitation_not_found", tt.name, err)
		}
		wantTx := 1
		if tt.calls == nil {
			wantTx = 0
		}
		if !slices.Equal(f.log.calls, tt.calls) || f.tx.calls != wantTx {
			t.Errorf("%s: calls = %q in %d transactions, want %q in %d", tt.name, f.log.calls, f.tx.calls, tt.calls, wantTx)
		}
	}
	f := newInvitations()
	if _, err := f.update().Execute(context.Background(), carolToAcme.ID, shared.RoleGuest); !errors.Is(err, shared.Unauthenticated()) ||
		len(f.log.calls) != 0 {
		t.Errorf("Execute() without an actor = %v, calls %q; want 401 unauthorized and no call", err, f.log.calls)
	}
}
````

`server/internal/modules/workspace/app/delete_invitation_test.go`（新文件，105 行）：

````file server/internal/modules/workspace/app/delete_invitation_test.go
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

func (f *invitationsFixture) delete() *app.DeleteWorkspaceInvitation {
	return app.NewDeleteWorkspaceInvitation(f.invitations, f.auth, f.tx, clockAt{at: clockNow})
}

// DeleteWorkspaceInvitation reads the invitation, locks its workspace FOR
// SHARE, locks the invitation, decides, then deletes it, all in one
// transaction (M3 design 3.6). A declined invitation is deleted as a
// pending one is (3.8). Two admins, two workspaces; the other invitations
// stay.
func TestDeleteWorkspaceInvitationLocksThenDecidesThenDeletes(t *testing.T) {
	for _, tt := range []struct {
		user app.AccountState
		inv  domain.Invitation
		left []domain.Invitation
	}{
		{alice, carolToAcme, []domain.Invitation{daveToAcme, erinToBeta}},
		{alice, daveToAcme, []domain.Invitation{carolToAcme, erinToBeta}},
		{bob, erinToBeta, []domain.Invitation{carolToAcme, daveToAcme}},
	} {
		f := newInvitations()
		err := f.delete().Execute(as(tt.user), tt.inv.ID)
		wantCalls := append(lockedInvitationCalls(tt.user, tt.inv, domain.ActionInvitationDelete),
			fmt.Sprintf("DeleteInvitation %s by %s at %s", tt.inv.ID, tt.user.ID, clockNow.Format(time.RFC3339Nano)))
		if err != nil || !slices.Equal(f.log.calls, wantCalls) || f.tx.calls != 1 {
			t.Errorf("%s: Execute() = %v, calls = %q in %d transactions; want %q in one", tt.inv.Email, err, f.log.calls, f.tx.calls, wantCalls)
		}
		if !slices.EqualFunc(f.invitations.invitations, tt.left, func(a, b domain.Invitation) bool { return a.ID == b.ID }) {
			t.Errorf("%s: left %+v, want %+v", tt.inv.Email, f.invitations.invitations, tt.left)
		}
	}
}

// Each refusal and failure is the answer, and nothing is deleted: an
// invitation not there, deleted meanwhile, of a workspace deleted meanwhile
// or not visible is workspace.invitation_not_found; a member's forbidden.
// A failure is itself, never a 404.
func TestDeleteWorkspaceInvitationRefusals(t *testing.T) {
	failure := errors.New("connection reset")
	decided := lockedInvitationCalls(alice, carolToAcme, domain.ActionInvitationDelete)
	stranger := uuid.NewV7()
	tests := []struct {
		name  string
		user  app.AccountState
		id    uuid.UUID
		set   func(f *invitationsFixture)
		want  error
		calls []string
	}{
		{"no such invitation", alice, stranger, nil, domain.ErrInvitationNotFound, []string{"InvitationByID " + stranger.String()}},
		{"acme deleted while the lock waited", alice, carolToAcme.ID,
			func(f *invitationsFixture) { f.invitations.lockErrs = map[string]error{"acme": app.ErrNotFound} }, domain.ErrInvitationNotFound,
			decided[:2]},
		{"deleted while the lock waited", alice, carolToAcme.ID,
			func(f *invitationsFixture) {
				f.invitations.onLock = func() { f.invitations.invitations = []domain.Invitation{daveToAcme, erinToBeta} }
			},
			domain.ErrInvitationNotFound, decided[:3]},
		{"not visible", carol, carolToAcme.ID, nil, domain.ErrInvitationNotFound,
			lockedInvitationCalls(carol, carolToAcme, domain.ActionInvitationDelete)},
		{"a member", bob, daveToAcme.ID, func(f *invitationsFixture) { f.auth.errs = map[grantKey]error{{bob.ID, acme.ID}: shared.Forbidden()} },
			shared.Forbidden(), lockedInvitationCalls(bob, daveToAcme, domain.ActionInvitationDelete)},
		{"the read failed", alice, carolToAcme.ID, func(f *invitationsFixture) { f.invitations.readErr = failure }, failure, decided[:1]},
		{"the workspace's lock failed", alice, carolToAcme.ID,
			func(f *invitationsFixture) { f.invitations.lockErrs = map[string]error{"acme": failure} }, failure, decided[:2]},
		{"the invitation's lock failed", alice, carolToAcme.ID,
			func(f *invitationsFixture) { f.invitations.onLock = func() { f.invitations.readErr = failure } }, failure, decided[:3]},
		{"the Authorizer failed", alice, carolToAcme.ID,
			func(f *invitationsFixture) { f.auth.errs = map[grantKey]error{{alice.ID, acme.ID}: failure} }, failure, decided},
		{"the write failed", alice, carolToAcme.ID, func(f *invitationsFixture) { f.invitations.writeErr = failure }, failure,
			append(decided, fmt.Sprintf("DeleteInvitation %s by %s at %s", carolToAcme.ID, alice.ID, clockNow.Format(time.RFC3339Nano)))},
	}
	for _, tt := range tests {
		f := newInvitations()
		if tt.set != nil {
			tt.set(f)
		}
		err := f.delete().Execute(as(tt.user), tt.id)
		if !errors.Is(err, tt.want) || (tt.want == failure && errors.Is(err, domain.ErrInvitationNotFound)) {
			t.Errorf("%s: Execute() = %v, want %v", tt.name, err, tt.want)
		}
		if !slices.Equal(f.log.calls, tt.calls) || f.tx.calls != 1 {
			t.Errorf("%s: calls = %q in %d transactions, want %q in one", tt.name, f.log.calls, f.tx.calls, tt.calls)
		}
	}
	f := newInvitations()
	if err := f.delete().Execute(context.Background(), carolToAcme.ID); !errors.Is(err, shared.Unauthenticated()) || len(f.log.calls) != 0 {
		t.Errorf("Execute() without an actor = %v, calls %q; want 401 unauthorized and no call", err, f.log.calls)
	}
}
````

`server/internal/modules/workspace/app/clock_test.go`（修改，1 处）：

````old server/internal/modules/workspace/app/clock_test.go
			fmt.Sprintf("PublicProfiles %v", []uuid.UUID{bob.ID}))},
````

````new server/internal/modules/workspace/app/clock_test.go
			fmt.Sprintf("PublicProfiles %v", []uuid.UUID{bob.ID}))},
		{"updateWorkspaceInvitation", func() ([]string, error) {
			f := newInvitations()
			_, err := app.NewUpdateWorkspaceInvitation(f.invitations, f.auth, f.tx, clockAt{clockNow, f.log}, f.mac).
				Execute(as(alice), carolToAcme.ID, shared.RoleGuest)
			return f.log.calls, err
		}, append(lockedInvitationCalls(alice, carolToAcme, domain.ActionInvitationUpdate), "Now",
			fmt.Sprintf("UpdateInvitationRole %s to %d by %s at %s", carolToAcme.ID, shared.RoleGuest, alice.ID, at))},
		{"deleteWorkspaceInvitation", func() ([]string, error) {
			f := newInvitations()
			err := app.NewDeleteWorkspaceInvitation(f.invitations, f.auth, f.tx, clockAt{clockNow, f.log}).Execute(as(alice), daveToAcme.ID)
			return f.log.calls, err
		}, append(lockedInvitationCalls(alice, daveToAcme, domain.ActionInvitationDelete), "Now",
			fmt.Sprintf("DeleteInvitation %s by %s at %s", daveToAcme.ID, alice.ID, at))},
````

- [ ] **Step 4: HTTP、规则、接线**

`server/internal/modules/workspace/adapter/http/handler.go`（修改，2 处）：

````old server/internal/modules/workspace/adapter/http/handler.go
}

// CheckSlugUseCase is app.CheckSlug.
````

````new server/internal/modules/workspace/adapter/http/handler.go
}

// UpdateInvitationUseCase is app.UpdateWorkspaceInvitation.
type UpdateInvitationUseCase interface {
	Execute(ctx context.Context, id uuid.UUID, role shared.Role) (domain.InvitationWithToken, error)
}

// DeleteInvitationUseCase is app.DeleteWorkspaceInvitation.
type DeleteInvitationUseCase interface {
	Execute(ctx context.Context, id uuid.UUID) error
}

// CheckSlugUseCase is app.CheckSlug.
````

````old server/internal/modules/workspace/adapter/http/handler.go
	CreateInvitations CreateInvitationsUseCase
````

````new server/internal/modules/workspace/adapter/http/handler.go
	CreateInvitations CreateInvitationsUseCase
	UpdateInvitation  UpdateInvitationUseCase
	DeleteInvitation  DeleteInvitationUseCase
````

`server/internal/modules/workspace/adapter/http/invitations.go`（修改，1 处）：

````old server/internal/modules/workspace/adapter/http/invitations.go
}

// invitations is list as the API shows it.
````

````new server/internal/modules/workspace/adapter/http/invitations.go
}

// UpdateWorkspaceInvitation serves PATCH
// /api/v0/workspace-invitations/{invitation_id}.
func (h handler) UpdateWorkspaceInvitation(ctx context.Context, req gen.UpdateWorkspaceInvitationRequestObject) (gen.UpdateWorkspaceInvitationResponseObject, error) {
	inv, err := h.uc.UpdateInvitation.Execute(ctx, req.InvitationID, shared.Role(req.Body.Role))
	if err != nil {
		return nil, err
	}
	return gen.UpdateWorkspaceInvitation200JSONResponse(invitation(inv)), nil
}

// DeleteWorkspaceInvitation serves DELETE
// /api/v0/workspace-invitations/{invitation_id}.
func (h handler) DeleteWorkspaceInvitation(ctx context.Context, req gen.DeleteWorkspaceInvitationRequestObject) (gen.DeleteWorkspaceInvitationResponseObject, error) {
	if err := h.uc.DeleteInvitation.Execute(ctx, req.InvitationID); err != nil {
		return nil, err
	}
	return gen.DeleteWorkspaceInvitation204Response{}, nil
}

// invitations is list as the API shows it.
````

`server/internal/modules/workspace/adapter/http/handler_test.go`（修改，1 处）：

````old server/internal/modules/workspace/adapter/http/handler_test.go
		ListInvitations: fakeListInvitations{f.invitations}, CreateInvitations: fakeCreateInvitations{f.invitations},
````

````new server/internal/modules/workspace/adapter/http/handler_test.go
		ListInvitations: fakeListInvitations{f.invitations}, CreateInvitations: fakeCreateInvitations{f.invitations},
		UpdateInvitation: fakeUpdateInvitation{f.invitations}, DeleteInvitation: fakeDeleteInvitation{f.invitations},
````

`server/internal/modules/workspace/adapter/http/invitations_test.go`（修改，3 处）：

````old server/internal/modules/workspace/adapter/http/invitations_test.go
	lists map[string][]domain.InvitationWithToken // by slug
````

````new server/internal/modules/workspace/adapter/http/invitations_test.go
	lists map[string][]domain.InvitationWithToken // by slug
	one   domain.InvitationWithToken              // the answer of an update
````

````old server/internal/modules/workspace/adapter/http/invitations_test.go
	f.calls = append(f.calls, "list "+caller(ctx)+" "+slug)
	return f.lists[slug], f.err
````

````new server/internal/modules/workspace/adapter/http/invitations_test.go
	f.calls = append(f.calls, "list "+caller(ctx)+" "+slug)
	return f.lists[slug], f.err
}

type fakeUpdateInvitation struct{ *fakeInvitations }

func (f fakeUpdateInvitation) Execute(ctx context.Context, id uuid.UUID, role shared.Role) (domain.InvitationWithToken, error) {
	f.calls = append(f.calls, fmt.Sprintf("update %s %s %d", caller(ctx), id, role))
	return f.one, f.err
}

type fakeDeleteInvitation struct{ *fakeInvitations }

func (f fakeDeleteInvitation) Execute(ctx context.Context, id uuid.UUID) error {
	f.calls = append(f.calls, fmt.Sprintf("delete %s %s", caller(ctx), id))
	return f.err
````

````old server/internal/modules/workspace/adapter/http/invitations_test.go
		t.Errorf("POST without a role = %d %s, calls %q; want 400 on invitations[0].role and no call", res.StatusCode, body, inv.calls)
	}
}

````

````new server/internal/modules/workspace/adapter/http/invitations_test.go
		t.Errorf("POST without a role = %d %s, calls %q; want 400 on invitations[0].role and no call", res.StatusCode, body, inv.calls)
	}
}

// PATCH hands the invitation of the path and the role as sent to the use
// case, for the caller, and answers the invitation it gives.
func TestUpdateWorkspaceInvitation(t *testing.T) {
	inv := &fakeInvitations{one: carolInvited}
	h := newServer(t, fakes{invitations: inv})
	for _, body := range []string{`{"role":20}`, `{"role":10}`} {
		res, got := do(t, h, request(http.MethodPatch, "/api/v0/workspace-invitations/"+carolInvited.ID.String(), "alice", body))
		if res.StatusCode != http.StatusOK || got != carolInvitedJSON+"\n" {
			t.Errorf("PATCH %s = %d %s, want 200 %s", body, res.StatusCode, got, carolInvitedJSON)
		}
	}
	id := carolInvited.ID.String()
	if want := []string{"update alice " + id + " 20", "update alice " + id + " 10"}; !slices.Equal(inv.calls, want) {
		t.Errorf("calls = %q, want %q", inv.calls, want)
	}
}

// The use case's refusals, as the contract declares them; a body without a
// role and an id that is no UUID are refused before it.
func TestUpdateWorkspaceInvitationRefusals(t *testing.T) {
	path := "/api/v0/workspace-invitations/" + carolInvited.ID.String()
	for _, tt := range []struct {
		err    error
		status int
		want   string
	}{
		{shared.Invalid(shared.FieldError{Field: "role", Code: shared.FieldInvalidFormat, Message: "is not 5, 15 or 20"}), http.StatusUnprocessableEntity,
			`{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
				`"errors":[{"field":"role","code":"invalid_format","message":"is not 5, 15 or 20"}]}`},
		{domain.ErrInvitationNotFound, http.StatusNotFound, invitationNotFoundJSON},
		{shared.Forbidden(), http.StatusForbidden, `{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
		{domain.ErrInvitationResponded, http.StatusConflict,
			`{"status":409,"code":"workspace.invitation_responded","title":"Conflict","detail":"The invitation has been answered already."}`},
	} {
		h := newServer(t, fakes{invitations: &fakeInvitations{err: tt.err}})
		if res, body := do(t, h, request(http.MethodPatch, path, "bob", `{"role":5}`)); res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("PATCH refused with %v = %d %s, want %d %s", tt.err, res.StatusCode, body, tt.status, tt.want)
		}
	}
	inv := &fakeInvitations{}
	h := newServer(t, fakes{invitations: inv})
	for _, req := range []*http.Request{request(http.MethodPatch, path, "alice", `{}`),
		request(http.MethodPatch, "/api/v0/workspace-invitations/carol", "alice", `{"role":5}`)} {
		if res, _ := do(t, h, req); res.StatusCode != http.StatusBadRequest || len(inv.calls) != 0 {
			t.Errorf("PATCH %s = %d, calls %q; want 400 and no call", req.URL.Path, res.StatusCode, inv.calls)
		}
	}
}

// invitationNotFoundJSON is workspace.invitation_not_found's problem.
const invitationNotFoundJSON = `{"status":404,"code":"workspace.invitation_not_found","title":"Not Found",` +
	`"detail":"The invitation does not exist, or its link is not valid."}`

// DELETE hands the invitation of the path to the use case, for the caller,
// and answers 204 without a body; its refusals are the contract's.
func TestDeleteWorkspaceInvitation(t *testing.T) {
	inv := &fakeInvitations{}
	h := newServer(t, fakes{invitations: inv})
	path := "/api/v0/workspace-invitations/" + daveDeclined.ID.String()
	if res, body := do(t, h, request(http.MethodDelete, path, "alice", "")); res.StatusCode != http.StatusNoContent || body != "" {
		t.Errorf("DELETE = %d %q, want 204 and no body", res.StatusCode, body)
	}
	if want := []string{"delete alice " + daveDeclined.ID.String()}; !slices.Equal(inv.calls, want) {
		t.Errorf("calls = %q, want %q", inv.calls, want)
	}
	for _, tt := range []struct {
		err    error
		status int
		want   string
	}{
		{domain.ErrInvitationNotFound, http.StatusNotFound, invitationNotFoundJSON},
		{shared.Forbidden(), http.StatusForbidden, `{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
	} {
		h := newServer(t, fakes{invitations: &fakeInvitations{err: tt.err}})
		if res, body := do(t, h, request(http.MethodDelete, path, "bob", "")); res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("DELETE refused with %v = %d %s, want %d %s", tt.err, res.StatusCode, body, tt.status, tt.want)
		}
	}
}

````

`server/internal/modules/access/domain/rules.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules.go
	"workspace_invitation.create": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin}},
````

````new server/internal/modules/access/domain/rules.go
	"workspace_invitation.create": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin}},
	"workspace_invitation.update": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin}},
	"workspace_invitation.delete": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin}},
````

`server/internal/modules/access/domain/rules_test.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules_test.go
	"workspace_invitation.create":  {allowed, forbidden, forbidden, invisible, invisible, invisible, forbidden},
````

````new server/internal/modules/access/domain/rules_test.go
	"workspace_invitation.create":  {allowed, forbidden, forbidden, invisible, invisible, invisible, forbidden},
	"workspace_invitation.update":  {allowed, forbidden, forbidden, invisible, invisible, invisible, forbidden},
	"workspace_invitation.delete":  {allowed, forbidden, forbidden, invisible, invisible, invisible, forbidden},
````

`server/internal/modules/workspace/module.go`（修改，1 处）：

````old server/internal/modules/workspace/module.go
			Caller: d.CallerLock, Invitations: store, Profiles: d.Profiles, Auth: d.Authorizer, Tx: d.Tx, Clock: d.Clock, MAC: d.InvitationMAC,
		}),
````

````new server/internal/modules/workspace/module.go
			Caller: d.CallerLock, Invitations: store, Profiles: d.Profiles, Auth: d.Authorizer, Tx: d.Tx, Clock: d.Clock, MAC: d.InvitationMAC,
		}),
		UpdateInvitation: app.NewUpdateWorkspaceInvitation(store, d.Authorizer, d.Tx, d.Clock, d.InvitationMAC),
		DeleteInvitation: app.NewDeleteWorkspaceInvitation(store, d.Authorizer, d.Tx, d.Clock),
````

- [ ] **Step 5: 矩阵**

`server/internal/bootstrap/permission_matrix_workspace_test.go`（修改，1 处）：

````old server/internal/bootstrap/permission_matrix_workspace_test.go
			request: toWorkspace(http.MethodPost, "/invitations", inviting("newcomer@example.com")),
			cells:   inWorkspace(cellValidationFailed, cellForbidden, cellForbidden)},
````

````new server/internal/bootstrap/permission_matrix_workspace_test.go
			request: toWorkspace(http.MethodPost, "/invitations", inviting("newcomer@example.com")),
			cells:   inWorkspace(cellValidationFailed, cellForbidden, cellForbidden)},
		{op: "updateWorkspaceInvitation", write: true, request: toInvitation(http.MethodPatch, "newcomer@example.com", `{"role":20}`),
			cells: ofInvitation(cellOK, cellForbidden, cellForbidden), check: promotesTheNewcomer},
		{op: "deleteWorkspaceInvitation", write: true, request: toInvitation(http.MethodDelete, "newcomer@example.com", ""),
			cells: ofInvitation(cellNoContent, cellForbidden, cellForbidden)},
````

`server/internal/bootstrap/permission_matrix_invitations_test.go`（修改，2 处）：

````old server/internal/bootstrap/permission_matrix_invitations_test.go
	"log/slog"
````

````new server/internal/bootstrap/permission_matrix_invitations_test.go
	"log/slog"
	"net/http"
````

````old server/internal/bootstrap/permission_matrix_invitations_test.go
// The invitations' part of the workspace module's rows (M3 design 9.2).
````

````new server/internal/bootstrap/permission_matrix_invitations_test.go
// The invitations' part of the workspace module's rows (M3 design 9.2).

var cellInvitationNotFound = cell{http.StatusNotFound, "workspace.invitation_not_found"}

// ofInvitation are the cells of a row that names an invitation: the answers
// of the workspace's admin, member and guest, and
// workspace.invitation_not_found for the callers the workspace is not
// visible to.
func ofInvitation(admin, member, guest cell) map[caller]cell {
	return map[caller]cell{callerAdmin: admin, callerMember: member, callerGuest: guest,
		callerNever: cellInvitationNotFound, callerRemoved: cellInvitationNotFound, callerDeleted: cellInvitationNotFound}
}

// toInvitation is the request of a row whose callers each send method and
// body to the invitation of email seeded in the workspace their column
// targets.
func toInvitation(method, email, body string) func(caller, seeded) (string, string, string) {
	return func(c caller, s seeded) (string, string, string) {
		return method, "/api/v0/workspace-invitations/" + s.invitation(workspaceOf(c), email).String(), body
	}
}

// promotesTheNewcomer: the admin's answer is acme's invitation of
// newcomer@example.com, now as an admin, with the token of its id.
func promotesTheNewcomer(t *testing.T, c caller, answer string) {
	var inv struct {
		ID          uuid.UUID `json:"id"`
		WorkspaceID uuid.UUID `json:"workspace_id"`
		Email       string    `json:"email"`
		Role        int       `json:"role"`
		Token       string    `json:"token"`
	}
	decodeAnswer(t, answer, &inv)
	if inv.Email != "newcomer@example.com" || inv.Role != 20 || inv.Token != invitationToken(t, inv.ID) {
		t.Errorf("%s's change answers %+v, want newcomer@example.com's invitation as an admin, with its token", c, inv)
	}
}
````

- [ ] **Step 6: 测试、lint**

Run: `go -C server test -count=1 ./internal/modules/workspace/... ./internal/modules/access/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestPermissionMatrix$|TestThePermissionMatrixCoversEveryOperation|TestMatrixViolationsCatchesEachGap|TestEveryActionHasARuleAndEveryRuleAnAction' ./internal/bootstrap/`
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
git add api/modules/workspace.yaml api/openapi.yaml server/internal/bootstrap/permission_matrix_invitations_test.go server/internal/bootstrap/permission_matrix_workspace_test.go server/internal/modules/access/domain/rules.go server/internal/modules/access/domain/rules_test.go server/internal/modules/workspace/adapter/http/handler.go server/internal/modules/workspace/adapter/http/handler_test.go server/internal/modules/workspace/adapter/http/invitations.go server/internal/modules/workspace/adapter/http/invitations_test.go server/internal/modules/workspace/app/clock_test.go server/internal/modules/workspace/app/delete_invitation.go server/internal/modules/workspace/app/delete_invitation_test.go server/internal/modules/workspace/app/fakes_invitations_test.go server/internal/modules/workspace/app/fakes_test.go server/internal/modules/workspace/app/invitation_lock.go server/internal/modules/workspace/app/invitation_ports.go server/internal/modules/workspace/app/update_invitation.go server/internal/modules/workspace/app/update_invitation_test.go server/internal/modules/workspace/domain/actions.go server/internal/modules/workspace/domain/errors.go server/internal/modules/workspace/domain/invitation.go server/internal/modules/workspace/module.go web/apps/web/helpers/authentication.helper.ts web/packages/i18n/src/locales/en/auth.json web/packages/i18n/src/locales/zh-CN/auth.json api/dist/openapi.yaml server/internal/modules/workspace/adapter/http/gen/bodyshape.gen.go server/internal/modules/workspace/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M3/P3): change an invitation's role and delete an invitation

PATCH and DELETE /api/v0/workspace-invitations/{invitation_id}: in one
transaction the invitation is read for its workspace, the workspace
locked FOR SHARE, the invitation locked FOR UPDATE and read again, then
decided (M3 design 3.6 convention 2); a declined invitation's role no
longer changes (409), and the clock is read under the locks. New codes
workspace.invitation_not_found and workspace.invitation_responded, with
their messages.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A）：

| 改坏 | 必须失败的测试 |
|---|---|
| 修改、删除按锁之前读到的判定 | `TestUpdateWorkspaceInvitationLocksThenDecidesThenWrites`；`TestDeleteWorkspaceInvitationLocksThenDecidesThenDeletes` |
| 修改不锁；工作区用 `FOR NO KEY UPDATE` | `TestUpdateWorkspaceInvitationLocksThenDecidesThenWrites` |
| 修改、删除的读和锁在事务之前 | 两个用例测试（假事务记下 `outside tx`） |
| 按工作区的 id 锁邀请 | `TestUpdateWorkspaceInvitationLocksThenDecidesThenWrites`、`TestDeleteWorkspaceInvitationLocksThenDecidesThenDeletes`（假存储按参数回答） |
| 已忽略的邀请可以改角色；成员被告知已忽略 | `TestUpdateWorkspaceInvitationRefusals` |
| 修改、删除在锁之前读时钟 | `TestEachWriteReadsTheClockUnderItsLock` |
| `workspace_invitation.update` 给成员 | `TestEveryRuleDecidesItsCells`、`TestPermissionMatrix` |
| 规则表加一行（`workspace_invitation.resend`）而没有格子 | `TestEveryRuleDecidesItsCells`、`TestEveryActionHasARuleAndEveryRuleAnAction` |

**Done when:** 两个用例、HTTP、矩阵的 12 格通过；两个新码有文案，`make test-web` 通过。

---

### Task 9: 公开的 `getWorkspaceInvitation`；访问日志；矩阵的豁免和未知参数

**Files:**
- Create: `server/internal/bootstrap/invitations_test.go`、`server/internal/modules/workspace/app/get_invitation.go`、`server/internal/modules/workspace/app/get_invitation_test.go`
- Modify: `api/modules/workspace.yaml`、`server/internal/bootstrap/app.go`、`server/internal/bootstrap/permission_matrix_coverage_test.go`、`server/internal/bootstrap/permission_matrix_invitations_test.go`、`server/internal/bootstrap/permission_matrix_test.go`、`server/internal/modules/workspace/adapter/http/handler.go`、`server/internal/modules/workspace/adapter/http/handler_test.go`、`server/internal/modules/workspace/adapter/http/invitations.go`、`server/internal/modules/workspace/adapter/http/invitations_test.go`、`server/internal/modules/workspace/adapter/postgres/invitations.go`、`server/internal/modules/workspace/adapter/postgres/invitations_test.go`、`server/internal/modules/workspace/adapter/postgres/queries/invitations.sql`、`server/internal/modules/workspace/app/fakes_invitations_test.go`、`server/internal/modules/workspace/app/invitation_ports.go`、`server/internal/modules/workspace/app/tokens.go`、`server/internal/modules/workspace/domain/invitation.go`、`server/internal/modules/workspace/module.go`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/workspace/adapter/http/gen/server.gen.go`、`server/internal/modules/workspace/adapter/postgres/gen/invitations.sql.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Produces（spec 2.10、2.14，M3 设计 3.8、8.1、8.2）：接口 `GET /api/v0/workspace-invitations/{invitation_id}?token=…`（`security: []`，`token` 必填），200 `InvitationPreview`；`domain.InvitationPreview{ID, Role, Declined, WorkspaceName, WorkspaceSlug}`；`app.InvitationPreviewer`、`NewGetWorkspaceInvitation(invitations, mac)`；`invitationTokens.valid(id, token)`；存储 `InvitationPreview(ctx, id)`；`httpadapter.PublicOperations()`、`(*workspace.Module).PublicOperations()`，`bootstrap` 并进公开操作。矩阵：`matrixExemptions.public`（操作 → 代替它的测试）、`notTargets`（路径 → 理由），`targetViolation` 报告任何不认识的参数。
- 使用者：Task 11、12 用 `valid`；P9 的邀请页。

**Tests:**
- `app/get_invitation_test.go`：`TestGetWorkspaceInvitationShowsTheLinksInvitation`（角色、是否已忽略、工作区的名称和 slug；先 MAC 后读，不开事务，没有调用者；两个工作区，一份待接受、一份已忽略）；`TestGetWorkspaceInvitationRefusesAWrongTokenBeforeReading`（别的邀请的、别的密钥的、改一个字符的、每种格式不对的，都是 `invitation_not_found` 且什么都不读；格式不对的不交给 MAC）；`TestGetWorkspaceInvitationRefusals`（存储找不到时同一个 404；失败原样）。
- `adapter/postgres/invitations_test.go`：`TestInvitationPreview`（已接受、已删除、工作区已删除、没有的 id 都是 `ErrNotFound`；失败是错误）。
- `adapter/http/invitations_test.go`：`TestGetWorkspaceInvitation`（带不带 bearer、带无效的 bearer，路由都不读凭证）；`TestGetWorkspaceInvitationRefusals`（不带令牌、id 不是 UUID 在用例之前 400；回答不重复令牌）。
- `bootstrap/invitations_test.go`：`TestTheInvitationLinksTokenIsNotLogged`（debug 级别；令牌对、改成大写、不带三个请求；三行访问日志的 `path` 不带查询；日志里没有 `token=`、令牌、令牌的 22 个字符、`nrv_inv_`）。
- `bootstrap/permission_matrix_invitations_test.go`：`TestTheInvitationLinkAnswersEveryCallerAlike`（代替这个操作的一行：好链接和四种坏链接，各以不带令牌和六列的令牌请求，每个调用者相同；四种坏链接逐字节相同）。
- `bootstrap/permission_matrix_coverage_test.go`：`TestMatrixViolationsCatchesEachGap` 加反例：公开的操作既没有行也没有豁免；公开的豁免指名不存在的操作、需要令牌的操作、另有一行的操作；矩阵不认识的参数（每一列一条）；列为不是目标的路径不报告；列出而不是任何操作的路径被报告。

- [ ] **Step 1: 接口描述、查询**

`api/modules/workspace.yaml`（修改，2 处）：

````old api/modules/workspace.yaml
      - $ref: '#/components/parameters/InvitationID'
````

````new api/modules/workspace.yaml
      - $ref: '#/components/parameters/InvitationID'
    get:
      operationId: getWorkspaceInvitation
      tags: [workspace]
      summary: Show an invitation to whoever holds its link
      description: >-
        Public: the link's token stands for a credential, and the answer is
        the same with a bearer token or without one. It shows the
        invitation's role, whether it was declined, and its workspace's name
        and slug; never the address invited. A token that is not the
        invitation's, and an invitation that does not exist, is deleted or
        accepted, answer the same workspace.invitation_not_found. Requests
        are limited per client IP.
      security: []
      parameters:
        - name: token
          in: query
          required: true
          description: The token of the invitation's link (WorkspaceInvitation.token).
          schema:
            type: string
      x-problem-codes: [workspace.invitation_not_found]
      responses:
        '200':
          description: The invitation as its link shows it.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/InvitationPreview'
        default:
          $ref: '#/components/responses/Problem'
````

````old api/modules/workspace.yaml
        role:
          $ref: '#/components/schemas/WorkspaceRole'
    WorkspaceInvitationUpdate:
````

````new api/modules/workspace.yaml
        role:
          $ref: '#/components/schemas/WorkspaceRole'
    InvitationPreview:
      description: >-
        What an invitation's link shows whoever holds it. The workspace's
        fields are flat: the holder cannot read the workspace itself. There
        is no address: the holder of a link does not learn which address to
        register with.
      type: object
      additionalProperties: false
      required: [id, role, declined, workspace_name, workspace_slug, workspace_logo_url]
      properties:
        id:
          type: string
          format: uuid
        role:
          $ref: '#/components/schemas/WorkspaceRole'
        declined:
          description: True once the invitation was declined; it can no longer be answered.
          type: boolean
        workspace_name:
          type: string
        workspace_slug:
          type: string
        workspace_logo_url:
          description: Null until uploads arrive (M5).
          type: [string, 'null']
    WorkspaceInvitationUpdate:
````

`server/internal/modules/workspace/adapter/postgres/queries/invitations.sql`（修改，1 处）：

````old server/internal/modules/workspace/adapter/postgres/queries/invitations.sql
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;
````

````new server/internal/modules/workspace/adapter/postgres/queries/invitations.sql
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

-- name: InvitationPreview :one
-- getWorkspaceInvitation: what the link shows, never the address (M3 design 3.8). One module's tables, so one JOIN.
SELECT i.id, i.role, i.responded_at, w.name AS workspace_name, w.slug AS workspace_slug
FROM workspace_member_invites i
JOIN workspaces w ON w.id = i.workspace_id
WHERE i.id = sqlc.arg(id) AND i.deleted_at IS NULL AND w.deleted_at IS NULL;
````

- [ ] **Step 2: 生成**

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `2cba67156c23f75bc068609ae48a3807cc3254938de8d2c09b3b7fbccf53cc8c` | 1602 | `api/dist/openapi.yaml` |
| `079b53be43b7fc54cea5ae70d8f470c37a8a84e4bc0d9b920cbaaf9cc76faee5` | 2141 | `server/internal/modules/workspace/adapter/http/gen/server.gen.go` |
| `4448b39e4f3f64ef3de9a70199793747c1c09816bbbae7e28107307d0e2b0b51` | 259 | `server/internal/modules/workspace/adapter/postgres/gen/invitations.sql.go` |
| `c19580811f9c8ec487434d6de75a1e36498b9fd64d46286720e0c66da8385bb1` | 1686 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/workspace/adapter/http/gen/server.gen.go server/internal/modules/workspace/adapter/postgres/gen/invitations.sql.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 3: 领域、端口、用例**

`server/internal/modules/workspace/domain/invitation.go`（修改，1 处）：

````old server/internal/modules/workspace/domain/invitation.go
	return i.RespondedAt != nil
````

````new server/internal/modules/workspace/domain/invitation.go
	return i.RespondedAt != nil
}

// InvitationPreview is what an invitation's link shows whoever holds it (M3
// design 3.8, 5.2): the invitation's role, whether it was declined, and its
// workspace's name and slug. Never the address invited (decision 1).
type InvitationPreview struct {
	ID            uuid.UUID
	Role          shared.Role
	Declined      bool
	WorkspaceName string
	WorkspaceSlug string
````

`server/internal/modules/workspace/app/invitation_ports.go`（修改，1 处）：

````old server/internal/modules/workspace/app/invitation_ports.go
}

// InvitationLocker reads an invitation by its id, then locks it under its
````

````new server/internal/modules/workspace/app/invitation_ports.go
}

// InvitationPreviewer reads what an invitation's link shows.
type InvitationPreviewer interface {
	// InvitationPreview returns the undeleted invitation id of an undeleted
	// workspace as its link shows it; ErrNotFound when there is none.
	InvitationPreview(ctx context.Context, id uuid.UUID) (domain.InvitationPreview, error)
}

// InvitationLocker reads an invitation by its id, then locks it under its
````

`server/internal/modules/workspace/app/tokens.go`（修改，1 处）：

````old server/internal/modules/workspace/app/tokens.go
}

// withTokens is invitations, each with its token.
````

````new server/internal/modules/workspace/app/tokens.go
}

// valid reports whether token is the invitation id's: the MAC checks the
// tag in constant time (M3 design 8.1). A token not of the format has no
// tag to check.
func (t invitationTokens) valid(id uuid.UUID, token string) bool {
	tag, ok := domain.ParseToken(token)
	return ok && t.mac.Verify(domain.InvitationMessage(id), tag)
}

// withTokens is invitations, each with its token.
````

`server/internal/modules/workspace/app/get_invitation.go`（新文件，38 行）：

````file server/internal/modules/workspace/app/get_invitation.go
package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
)

// GetWorkspaceInvitation shows an invitation to whoever holds its link: the
// public GET /api/v0/workspace-invitations/{invitation_id}?token=….
type GetWorkspaceInvitation struct {
	invitations InvitationPreviewer
	tokens      invitationTokens
}

// NewGetWorkspaceInvitation returns the use case.
func NewGetWorkspaceInvitation(invitations InvitationPreviewer, mac InvitationMAC) *GetWorkspaceInvitation {
	return &GetWorkspaceInvitation{invitations: invitations, tokens: invitationTokens{mac: mac}}
}

// Execute checks the token, then reads. A token that is not the
// invitation's, and an invitation that does not exist, is deleted or
// accepted, or whose workspace is deleted, are the same
// workspace.invitation_not_found (M3 design 3.8, 8.2). A wrong token reads
// nothing: the token is the invitation id's MAC, which needs no row, so the
// answer to it cannot depend on whether the invitation exists. Nothing
// depends on the caller: the operation is public.
func (u *GetWorkspaceInvitation) Execute(ctx context.Context, id uuid.UUID, token string) (domain.InvitationPreview, error) {
	if !u.tokens.valid(id, token) {
		return domain.InvitationPreview{}, domain.ErrInvitationNotFound
	}
	p, err := u.invitations.InvitationPreview(ctx, id)
	if err != nil {
		return domain.InvitationPreview{}, invitationNotFound(err)
	}
	return p, nil
}
````

`server/internal/modules/workspace/app/fakes_invitations_test.go`（修改，1 处）：

````old server/internal/modules/workspace/app/fakes_invitations_test.go
}

// ShareWorkspace answers as LockWorkspace does.
````

````new server/internal/modules/workspace/app/fakes_invitations_test.go
}

// InvitationPreview answers the invitation it holds with its workspace's
// name and slug; app.ErrNotFound when it holds neither.
func (f *fakeInvitations) InvitationPreview(ctx context.Context, id uuid.UUID) (domain.InvitationPreview, error) {
	f.log.add(ctx, "InvitationPreview %s", id)
	inv, err := f.invitation(id)
	if err != nil {
		return domain.InvitationPreview{}, err
	}
	i := slices.IndexFunc(f.workspaces, func(w domain.Workspace) bool { return w.ID == inv.WorkspaceID })
	if i < 0 {
		return domain.InvitationPreview{}, app.ErrNotFound
	}
	return domain.InvitationPreview{ID: inv.ID, Role: inv.Role, Declined: inv.Responded(), WorkspaceName: f.workspaces[i].Name,
		WorkspaceSlug: f.workspaces[i].Slug}, nil
}

// ShareWorkspace answers as LockWorkspace does.
````

`server/internal/modules/workspace/app/get_invitation_test.go`（新文件，119 行）：

````file server/internal/modules/workspace/app/get_invitation_test.go
package app_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func (f *invitationsFixture) get() *app.GetWorkspaceInvitation {
	return app.NewGetWorkspaceInvitation(f.invitations, f.mac)
}

// The link shows the invitation's role, whether it was declined, and its
// workspace's name and slug: the token checked by the MAC, then the
// invitation read, without a transaction and without a caller. Two
// workspaces; a pending and a declined invitation.
func TestGetWorkspaceInvitationShowsTheLinksInvitation(t *testing.T) {
	for _, tt := range []struct {
		inv  domain.Invitation
		want domain.InvitationPreview
	}{
		{carolToAcme, domain.InvitationPreview{ID: carolToAcme.ID, Role: shared.RoleMember, WorkspaceName: "Acme", WorkspaceSlug: "acme"}},
		{daveToAcme, domain.InvitationPreview{ID: daveToAcme.ID, Role: shared.RoleGuest, Declined: true, WorkspaceName: "Acme", WorkspaceSlug: "acme"}},
		{erinToBeta, domain.InvitationPreview{ID: erinToBeta.ID, Role: shared.RoleAdmin, WorkspaceName: "Beta", WorkspaceSlug: "beta"}},
	} {
		f := newInvitations()
		got, err := f.get().Execute(context.Background(), tt.inv.ID, tokenOf(f.mac, tt.inv.ID))
		if err != nil || got != tt.want {
			t.Errorf("%s: Execute() = %+v, %v; want %+v", tt.inv.Email, got, err, tt.want)
		}
		if want := []string{"Verify " + tt.inv.ID.String(), "InvitationPreview " + tt.inv.ID.String() + " outside tx"}; !slices.Equal(f.log.calls, want) {
			t.Errorf("%s: calls = %q, want %q", tt.inv.Email, f.log.calls, want)
		}
	}
}

// Every token that is not the invitation's is workspace.invitation_not_found
// and reads nothing: another invitation's, one under another key, one with
// a character changed, and every string not of the format. A token not of
// the format is not even given to the MAC.
func TestGetWorkspaceInvitationRefusesAWrongTokenBeforeReading(t *testing.T) {
	f := newInvitations()
	right := tokenOf(f.mac, carolToAcme.ID)
	// A character of the tag's middle, whose six bits are all the tag's
	// (the last character's low bits are padding the format refuses).
	changed := []byte(right)
	changed[len("nrv_inv_")+10] = 'A'
	if string(changed) == right {
		changed[len("nrv_inv_")+10] = 'B'
	}
	tests := []struct {
		name, token string
		verified    bool
	}{
		{"another invitation's", tokenOf(f.mac, daveToAcme.ID), true},
		{"another workspace's invitation's", tokenOf(f.mac, erinToBeta.ID), true},
		{"under another key", tokenOf(fakeMAC{log: f.log, key: "another instance's key"}, carolToAcme.ID), true},
		{"a character changed", string(changed), true},
		{"empty", "", false},
		{"without its prefix", strings.TrimPrefix(right, "nrv_inv_"), false},
		{"the prefix in upper case", "NRV_INV_" + strings.TrimPrefix(right, "nrv_inv_"), false},
		{"one character short", right[:len(right)-1], false},
		{"one character long", right + "A", false},
		{"padded", right + "==", false},
		{"of the standard alphabet", strings.NewReplacer("-", "+", "_", "/").Replace(right) + "+", false},
	}
	for _, tt := range tests {
		f := newInvitations()
		got, err := f.get().Execute(context.Background(), carolToAcme.ID, tt.token)
		if !errors.Is(err, domain.ErrInvitationNotFound) || got != (domain.InvitationPreview{}) {
			t.Errorf("%s: Execute() = %+v, %v; want workspace.invitation_not_found", tt.name, got, err)
		}
		var want []string
		if tt.verified {
			want = []string{"Verify " + carolToAcme.ID.String()}
		}
		if !slices.Equal(f.log.calls, want) {
			t.Errorf("%s: calls = %q, want %q", tt.name, f.log.calls, want)
		}
	}
}

// An invitation the store does not find, with the token of its id, is the
// same workspace.invitation_not_found; a failed read is itself, never a
// 404.
func TestGetWorkspaceInvitationRefusals(t *testing.T) {
	failure := errors.New("connection reset")
	stranger := uuid.NewV7()
	for _, tt := range []struct {
		name string
		id   uuid.UUID
		set  func(f *invitationsFixture)
		want error
	}{
		{"no such invitation", stranger, nil, domain.ErrInvitationNotFound},
		{"deleted", carolToAcme.ID, func(f *invitationsFixture) { f.invitations.invitations = []domain.Invitation{daveToAcme} },
			domain.ErrInvitationNotFound},
		{"the read failed", carolToAcme.ID, func(f *invitationsFixture) { f.invitations.readErr = failure }, failure},
	} {
		f := newInvitations()
		if tt.set != nil {
			tt.set(f)
		}
		got, err := f.get().Execute(context.Background(), tt.id, tokenOf(f.mac, tt.id))
		if !errors.Is(err, tt.want) || got != (domain.InvitationPreview{}) || (tt.want == failure && errors.Is(err, domain.ErrInvitationNotFound)) {
			t.Errorf("%s: Execute() = %+v, %v; want %v", tt.name, got, err, tt.want)
		}
		if want := []string{"Verify " + tt.id.String(), "InvitationPreview " + tt.id.String() + " outside tx"}; !slices.Equal(f.log.calls, want) {
			t.Errorf("%s: calls = %q, want %q", tt.name, f.log.calls, want)
		}
	}
}
````

- [ ] **Step 4: 存储**

`server/internal/modules/workspace/adapter/postgres/invitations.go`（修改，1 处）：

````old server/internal/modules/workspace/adapter/postgres/invitations.go
}

// LockInvitation locks the undeleted invitation id FOR UPDATE until the
````

````new server/internal/modules/workspace/adapter/postgres/invitations.go
}

// InvitationPreview returns the undeleted invitation id of an undeleted
// workspace as its link shows it; app.ErrNotFound when there is none.
func (s *Store) InvitationPreview(ctx context.Context, id uuid.UUID) (domain.InvitationPreview, error) {
	r, err := s.queries(ctx).InvitationPreview(ctx, id)
	if err != nil {
		return domain.InvitationPreview{}, notFound(err)
	}
	return domain.InvitationPreview{ID: r.ID, Role: shared.Role(r.Role), Declined: r.RespondedAt != nil, WorkspaceName: r.WorkspaceName,
		WorkspaceSlug: r.WorkspaceSlug}, nil
}

// LockInvitation locks the undeleted invitation id FOR UPDATE until the
````

`server/internal/modules/workspace/adapter/postgres/invitations_test.go`（修改，1 处）：

````old server/internal/modules/workspace/adapter/postgres/invitations_test.go
		t.Errorf("ListInvitations() = %+v, %v; want %+v", got, err, want)
	}
}

````

````new server/internal/modules/workspace/adapter/postgres/invitations_test.go
		t.Errorf("ListInvitations() = %+v, %v; want %+v", got, err, want)
	}
}

// InvitationPreview is the undeleted invitation of an undeleted workspace
// as its link shows it, pending or declined, with its own workspace's name
// and slug; app.ErrNotFound for an accepted or deleted invitation, one
// whose workspace is deleted (here without its invitations, which the
// deletion never leaves), and an id no row has. A failed read is its
// error, never app.ErrNotFound.
func TestInvitationPreview(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme, beta := newWorkspace(t, s, "Acme", "acme", alice), newWorkspace(t, s, "Beta", "beta", alice)
	gone := newWorkspace(t, s, "Gone", "gone", alice)
	pending := invite(t, s, acme.ID, "carol@corp.com", shared.RoleMember, alice)
	declined := invite(t, s, beta.ID, "carol@corp.com", shared.RoleGuest, alice)
	exec(t, pool, "UPDATE workspace_member_invites SET responded_at = $1 WHERE id = $2", now, declined.ID)
	accepted := invite(t, s, acme.ID, "dave@corp.com", shared.RoleMember, alice)
	exec(t, pool, "UPDATE workspace_member_invites SET accepted = true, responded_at = $1, deleted_at = $1 WHERE id = $2", now, accepted.ID)
	deleted := invite(t, s, acme.ID, "erin@corp.com", shared.RoleMember, alice)
	exec(t, pool, "UPDATE workspace_member_invites SET deleted_at = $1 WHERE id = $2", now, deleted.ID)
	orphan := invite(t, s, gone.ID, "carol@corp.com", shared.RoleMember, alice)
	exec(t, pool, "UPDATE workspaces SET deleted_at = $1 WHERE id = $2", now, gone.ID)

	for _, want := range []domain.InvitationPreview{
		{ID: pending.ID, Role: shared.RoleMember, WorkspaceName: "Acme", WorkspaceSlug: "acme"},
		{ID: declined.ID, Role: shared.RoleGuest, Declined: true, WorkspaceName: "Beta", WorkspaceSlug: "beta"},
	} {
		if got, err := s.InvitationPreview(context.Background(), want.ID); err != nil || got != want {
			t.Errorf("InvitationPreview(%s) = %+v, %v; want %+v", want.ID, got, err, want)
		}
	}
	for _, id := range []uuid.UUID{accepted.ID, deleted.ID, orphan.ID, uuid.NewV7()} {
		if got, err := s.InvitationPreview(context.Background(), id); !errors.Is(err, app.ErrNotFound) || got != (domain.InvitationPreview{}) {
			t.Errorf("InvitationPreview(%s) = %+v, %v; want app.ErrNotFound", id, got, err)
		}
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if got, err := s.InvitationPreview(cancelled, pending.ID); !errors.Is(err, context.Canceled) || errors.Is(err, app.ErrNotFound) ||
		got != (domain.InvitationPreview{}) {
		t.Errorf("InvitationPreview() on a cancelled context = %+v, %v; want context.Canceled, not app.ErrNotFound", got, err)
	}
}

````

- [ ] **Step 5: HTTP、公开操作、接线**

`server/internal/modules/workspace/adapter/http/handler.go`（修改，4 处）：

````old server/internal/modules/workspace/adapter/http/handler.go
}

// UpdateInvitationUseCase is app.UpdateWorkspaceInvitation.
````

````new server/internal/modules/workspace/adapter/http/handler.go
}

// GetInvitationUseCase is app.GetWorkspaceInvitation.
type GetInvitationUseCase interface {
	Execute(ctx context.Context, id uuid.UUID, token string) (domain.InvitationPreview, error)
}

// UpdateInvitationUseCase is app.UpdateWorkspaceInvitation.
````

````old server/internal/modules/workspace/adapter/http/handler.go
	CreateInvitations CreateInvitationsUseCase
````

````new server/internal/modules/workspace/adapter/http/handler.go
	CreateInvitations CreateInvitationsUseCase
	GetInvitation     GetInvitationUseCase
````

````old server/internal/modules/workspace/adapter/http/handler.go
}

// Register mounts the module's routes on router behind api's per-route
````

````new server/internal/modules/workspace/adapter/http/handler.go
}

// PublicOperations are the module's routes that need no token (M3 design
// 3.8), as the generated code registers them.
func PublicOperations() []string {
	return []string{"GET /api/v0/workspace-invitations/{invitation_id}"}
}

// Register mounts the module's routes on router behind api's per-route
````

````old server/internal/modules/workspace/adapter/http/handler.go
// Every operation needs a token.
````

````new server/internal/modules/workspace/adapter/http/handler.go
// Every operation but PublicOperations needs a token.
````

`server/internal/modules/workspace/adapter/http/invitations.go`（修改，1 处）：

````old server/internal/modules/workspace/adapter/http/invitations.go
	return gen.CreateWorkspaceInvitations201JSONResponse{Data: invitations(list)}, nil
````

````new server/internal/modules/workspace/adapter/http/invitations.go
	return gen.CreateWorkspaceInvitations201JSONResponse{Data: invitations(list)}, nil
}

// GetWorkspaceInvitation serves the public GET
// /api/v0/workspace-invitations/{invitation_id}.
func (h handler) GetWorkspaceInvitation(ctx context.Context, req gen.GetWorkspaceInvitationRequestObject) (gen.GetWorkspaceInvitationResponseObject, error) {
	p, err := h.uc.GetInvitation.Execute(ctx, req.InvitationID, req.Params.Token)
	if err != nil {
		return nil, err
	}
	return gen.GetWorkspaceInvitation200JSONResponse{
		ID:               p.ID,
		Role:             gen.WorkspaceRole(p.Role),
		Declined:         p.Declined,
		WorkspaceName:    p.WorkspaceName,
		WorkspaceSlug:    p.WorkspaceSlug,
		WorkspaceLogoURL: nullable.NewNullNullable[string](), // M5
	}, nil
````

`server/internal/modules/workspace/adapter/http/handler_test.go`（修改，2 处）：

````old server/internal/modules/workspace/adapter/http/handler_test.go
		Logger:         logger,
		Authenticator:  fakeAuth{},
		MaxBodyBytes:   1024,
		RequestTimeout: 5 * time.Second,
		IPv6PrefixLen:  64,
		Anonymous:      limit,
		Authenticated:  limit,
		AuthFailure:    limit,
````

````new server/internal/modules/workspace/adapter/http/handler_test.go
		Logger:           logger,
		Authenticator:    fakeAuth{},
		PublicOperations: httpadapter.PublicOperations(),
		MaxBodyBytes:     1024,
		RequestTimeout:   5 * time.Second,
		IPv6PrefixLen:    64,
		Anonymous:        limit,
		Authenticated:    limit,
		AuthFailure:      limit,
````

````old server/internal/modules/workspace/adapter/http/handler_test.go
		UpdateInvitation: fakeUpdateInvitation{f.invitations}, DeleteInvitation: fakeDeleteInvitation{f.invitations},
````

````new server/internal/modules/workspace/adapter/http/handler_test.go
		GetInvitation: fakeGetInvitation{f.invitations}, UpdateInvitation: fakeUpdateInvitation{f.invitations},
		DeleteInvitation: fakeDeleteInvitation{f.invitations},
````

`server/internal/modules/workspace/adapter/http/invitations_test.go`（修改，2 处）：

````old server/internal/modules/workspace/adapter/http/invitations_test.go
	calls []string
	lists map[string][]domain.InvitationWithToken // by slug
	one   domain.InvitationWithToken              // the answer of an update
	err   error
````

````new server/internal/modules/workspace/adapter/http/invitations_test.go
	calls   []string
	lists   map[string][]domain.InvitationWithToken // by slug
	one     domain.InvitationWithToken              // the answer of an update
	preview domain.InvitationPreview                // the answer of a get
	err     error
}

type fakeGetInvitation struct{ *fakeInvitations }

func (f fakeGetInvitation) Execute(ctx context.Context, id uuid.UUID, token string) (domain.InvitationPreview, error) {
	f.calls = append(f.calls, fmt.Sprintf("get %s %s %s", caller(ctx), id, token))
	return f.preview, f.err
````

````old server/internal/modules/workspace/adapter/http/invitations_test.go
			t.Errorf("DELETE refused with %v = %d %s, want %d %s", tt.err, res.StatusCode, body, tt.status, tt.want)
		}
	}
}

````

````new server/internal/modules/workspace/adapter/http/invitations_test.go
			t.Errorf("DELETE refused with %v = %d %s, want %d %s", tt.err, res.StatusCode, body, tt.status, tt.want)
		}
	}
}

// carolsLink is the path of carol's invitation's link, with its token.
var carolsLink = "/api/v0/workspace-invitations/" + carolInvited.ID.String() + "?token=" + carolInvited.Token

// The public GET hands the invitation of the path and the token of the
// query to the use case and answers what it shows, without the address.
// The route needs no bearer token and reads none: with one, even one that
// is not valid, the use case is called by nobody all the same.
func TestGetWorkspaceInvitation(t *testing.T) {
	inv := &fakeInvitations{preview: domain.InvitationPreview{ID: carolInvited.ID, Role: shared.RoleMember, Declined: true,
		WorkspaceName: "Acme", WorkspaceSlug: "acme"}}
	h := newServer(t, fakes{invitations: inv})
	want := `{"declined":true,"id":"0199a2b4-0000-7000-8000-0000000000c1","role":15,"workspace_logo_url":null,` +
		`"workspace_name":"Acme","workspace_slug":"acme"}`
	for _, token := range []string{"", "alice", "mallory"} {
		if res, body := do(t, h, request(http.MethodGet, carolsLink, token, "")); res.StatusCode != http.StatusOK || body != want+"\n" {
			t.Errorf("GET with the bearer %q = %d %s, want 200 %s", token, res.StatusCode, body, want)
		}
	}
	call := "get nobody " + carolInvited.ID.String() + " " + carolInvited.Token
	if want := []string{call, call, call}; !slices.Equal(inv.calls, want) {
		t.Errorf("calls = %q, want %q", inv.calls, want)
	}
}

// The use case's refusal, as the contract declares it; a link without its
// token and an id that is no UUID are refused before it. No answer repeats
// the token.
func TestGetWorkspaceInvitationRefusals(t *testing.T) {
	h := newServer(t, fakes{invitations: &fakeInvitations{err: domain.ErrInvitationNotFound}})
	if res, body := do(t, h, request(http.MethodGet, carolsLink, "", "")); res.StatusCode != http.StatusNotFound || body != invitationNotFoundJSON+"\n" {
		t.Errorf("GET refused = %d %s, want 404 %s", res.StatusCode, body, invitationNotFoundJSON)
	}
	inv := &fakeInvitations{}
	h = newServer(t, fakes{invitations: inv})
	for _, tt := range []struct{ path, field string }{
		{"/api/v0/workspace-invitations/" + carolInvited.ID.String(), `"field":"token","code":"required"`},
		{"/api/v0/workspace-invitations/carol?token=" + carolInvited.Token, `"field":"invitation_id","code":"invalid_format"`},
	} {
		res, body := do(t, h, request(http.MethodGet, tt.path, "", ""))
		if res.StatusCode != http.StatusBadRequest || !strings.Contains(body, tt.field) || strings.Contains(body, carolInvited.Token) || len(inv.calls) != 0 {
			t.Errorf("GET %s = %d %s, calls %q; want 400 on %s without the token, and no call", tt.path, res.StatusCode, body, inv.calls, tt.field)
		}
	}
}

````

`server/internal/modules/workspace/module.go`（修改，2 处）：

````old server/internal/modules/workspace/module.go
			Caller: d.CallerLock, Invitations: store, Profiles: d.Profiles, Auth: d.Authorizer, Tx: d.Tx, Clock: d.Clock, MAC: d.InvitationMAC,
		}),
````

````new server/internal/modules/workspace/module.go
			Caller: d.CallerLock, Invitations: store, Profiles: d.Profiles, Auth: d.Authorizer, Tx: d.Tx, Clock: d.Clock, MAC: d.InvitationMAC,
		}),
		GetInvitation:    app.NewGetWorkspaceInvitation(store, d.InvitationMAC),
````

````old server/internal/modules/workspace/module.go
	httpadapter.Register(router, api, m.uc)
````

````new server/internal/modules/workspace/module.go
	httpadapter.Register(router, api, m.uc)
}

// PublicOperations are the module's routes that need no token.
func (m *Module) PublicOperations() []string {
	return httpadapter.PublicOperations()
````

`server/internal/bootstrap/app.go`（修改，1 处）：

````old server/internal/bootstrap/app.go
	a.publicOperations = slices.Concat(ident.PublicOperations(), inst.PublicOperations())
````

````new server/internal/bootstrap/app.go
	a.publicOperations = slices.Concat(ident.PublicOperations(), inst.PublicOperations(), ws.PublicOperations())
````

- [ ] **Step 6: 访问日志、矩阵**

`server/internal/bootstrap/invitations_test.go`（新文件，77 行）：

````file server/internal/bootstrap/invitations_test.go
package bootstrap

import (
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// The invitations on the wired app and a real database: what the matrix
// and the modules' tests cannot see.

// invitationLink is an invitation's id and its link's token.
type invitationLink struct {
	id    uuid.UUID
	token string
}

// invite has admin, with the bearer token, invite email to the workspace
// slug as a member, and returns the invitation's link.
func invite(t *testing.T, contract *apitest.Contract, base, admin, slug, email string) invitationLink {
	t.Helper()
	status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces/"+slug+"/invitations", admin,
		`{"invitations":[{"email":"`+email+`","role":15}]}`)
	var list struct {
		Data []struct {
			ID    uuid.UUID `json:"id"`
			Token string    `json:"token"`
		} `json:"data"`
	}
	if status != http.StatusCreated {
		t.Fatalf("inviting %s = %d %s", email, status, body)
	}
	decodeAnswer(t, body, &list)
	return invitationLink{list.Data[0].ID, list.Data[0].Token}
}

// The link's token never reaches the logs, at any level (M3 design 8.1):
// the access log has the path of the public view and no query, whether the
// token is right, wrong or missing.
func TestTheInvitationLinksTokenIsNotLogged(t *testing.T) {
	var logs lockedBuffer
	base := startAppLogging(t, testConfig(t, pgtest.NewDatabase(t), false), migrations.FS(),
		slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	contract := apitest.Load(t)
	admin := registerAccount(t, contract, base, "admin@example.com").AccessToken
	if status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces", admin, `{"name":"Acme","slug":"acme"}`); status != http.StatusCreated {
		t.Fatalf("creating acme = %d %s", status, body)
	}
	link := invite(t, contract, base, admin, "acme", "carol@example.com")
	token, path := link.token, "/api/v0/workspace-invitations/"+link.id.String()

	for _, tt := range []struct {
		query string
		want  int
	}{{"?token=" + token, http.StatusOK}, {"?token=" + strings.ToUpper(token), http.StatusNotFound}, {"", http.StatusBadRequest}} {
		req := newRequest(t, http.MethodGet, base+path+tt.query, "", nil)
		if res, body := send(t, req); res.StatusCode != tt.want {
			t.Errorf("GET %s = %d %s, want %d", path+tt.query, res.StatusCode, body, tt.want)
		}
	}

	got := logs.String()
	if strings.Count(got, `msg="http request"`+" request_id=") == 0 || strings.Count(got, "path="+path+" ") != 3 {
		t.Fatalf("the logs lack the three requests' access lines:\n%s", got)
	}
	for _, leak := range []string{"token=", token, token[len("nrv_inv_"):], "nrv_inv_"} {
		if strings.Contains(got, leak) {
			t.Errorf("the logs hold %q:\n%s", leak, got)
		}
	}
}
````

`server/internal/bootstrap/permission_matrix_test.go`（修改，1 处）：

````old server/internal/bootstrap/permission_matrix_test.go
// matrixExempt are the modules whose operations have no row, each for its
// reason. Every other operation of the contract has one, so a module that
// adds operations is in the matrix unless it is added here (M3 design 9.2:
// every operation but the account-level and the public ones). An entry that
// no operation carries is reported, so a misspelled one fails. P3's public
// getWorkspaceInvitation is tagged workspace: P3 gives the matrix a column
// for a caller without a token, or an exemption by operation. This list
// exempts modules, and workspace on it would exempt all of its operations
// (spec P1 3 item 10).
var matrixExempt = []string{
	// Account-level (M2): each operation acts on the caller's own account,
	// sessions or tokens, and no workspace or project role decides it.
	"identity",
	// Public: it describes this instance to anyone, with a token or without.
	"instance",
````

````new server/internal/bootstrap/permission_matrix_test.go
// matrixExempt is what has no row, each entry for its reason (M3 design
// 9.2: every operation but the account-level and the public ones).
var matrixExempt = matrixExemptions{
	modules: []string{
		// Account-level (M2): each operation acts on the caller's own
		// account, sessions or tokens, and no workspace or project role
		// decides it.
		"identity",
		// Public: it describes this instance to anyone, with a token or
		// without.
		"instance",
	},
	public: map[string]string{
		// The link's token stands for a credential (M3 design 3.8).
		"getWorkspaceInvitation": "TestTheInvitationLinkAnswersEveryCallerAlike",
	},
}

// matrixExemptions are the operations without a row. modules exempts every
// operation of a module: every other operation of the contract has a row,
// so a module that adds operations is in the matrix unless it is listed,
// and an entry that no operation carries is reported, so a misspelled one
// fails. public exempts one public operation of a module the matrix
// covers, by its operationId, naming the test that stands for its row: its
// route runs no authentication (httpserver's PublicOperations), so every
// column would call it as nobody and the cells could not tell the columns
// apart. The test calls it with every column's token and without one, and
// wants one answer. An operation that needs a token cannot be listed.
type matrixExemptions struct {
	modules []string
	public  map[string]string // operationId → the test that stands for its row
````

`server/internal/bootstrap/permission_matrix_coverage_test.go`（修改，27 处）：

````old server/internal/bootstrap/permission_matrix_coverage_test.go
	"fmt"
````

````new server/internal/bootstrap/permission_matrix_coverage_test.go
	"fmt"
	"maps"
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
// operation without a row, unless every tag it has is on exempt, so that a
// new module's operations need rows without anyone listing the module; an
// exempt module that no operation carries, which a misspelling would be; a
// row that names no operation; a row without a cell for a column; a cell
````

````new server/internal/bootstrap/permission_matrix_coverage_test.go
// operation without a row, unless every tag it has is on exempt.modules, so
// that a new module's operations need rows without anyone listing the
// module, or it is on exempt.public; an exempt module that no operation
// carries, which a misspelling would be; a public exemption that names no
// operation, one that needs a token, or one that has a row; a row that
// names no operation; a row without a cell for a column; a cell
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
func matrixViolations(ops []apitest.Operation, exempt []string, rows []matrixRow, s seeded) []string {
````

````new server/internal/bootstrap/permission_matrix_coverage_test.go
func matrixViolations(ops []apitest.Operation, exempt matrixExemptions, rows []matrixRow, s seeded) []string {
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
		exempted := len(op.Tags) > 0 && !slices.ContainsFunc(op.Tags, func(tag string) bool { return !slices.Contains(exempt, tag) })
		if !exempted && !inMatrix[op.ID] {
````

````new server/internal/bootstrap/permission_matrix_coverage_test.go
		exempted := len(op.Tags) > 0 && !slices.ContainsFunc(op.Tags, func(tag string) bool { return !slices.Contains(exempt.modules, tag) })
		if _, public := exempt.public[op.ID]; !exempted && !public && !inMatrix[op.ID] {
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
	for _, module := range exempt {
````

````new server/internal/bootstrap/permission_matrix_coverage_test.go
	for _, id := range slices.Sorted(maps.Keys(exempt.public)) {
		op, named := byID[id]
		switch {
		case !named:
			found = append(found, fmt.Sprintf("the public exemption %s names no operation of the contract", id))
		case !op.Public:
			found = append(found, fmt.Sprintf("operation %s is exempt as public, but needs a token", id))
		case inMatrix[id]:
			found = append(found, fmt.Sprintf("operation %s is exempt as public, and has a row", id))
		}
	}
	for _, module := range exempt.modules {
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
// exempt unless a case says otherwise.
````

````new server/internal/bootstrap/permission_matrix_coverage_test.go
// exempt unless a case says otherwise, and so is the public
// getWorkspaceInvitation, which each ops holds.
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
		{ID: "getMe", Tags: []string{"identity"}, Method: http.MethodGet, Path: "/api/v0/me"}}
	exempt := []string{"identity"}
````

````new server/internal/bootstrap/permission_matrix_coverage_test.go
		{ID: "getMe", Tags: []string{"identity"}, Method: http.MethodGet, Path: "/api/v0/me"},
		{ID: "getWorkspaceInvitation", Tags: []string{"workspace"}, Method: http.MethodGet, Path: "/api/v0/workspace-invitations/{invitation_id}",
			Public: true}}
	exempt := matrixExemptions{modules: []string{"identity"}, public: map[string]string{"getWorkspaceInvitation": "TestOfItsOwn"}}
	// exemptPublic is exempt, but with the public exemptions public.
	exemptPublic := func(public ...string) matrixExemptions {
		e := matrixExemptions{modules: exempt.modules, public: map[string]string{}}
		for _, id := range public {
			e.public[id] = "TestOfItsOwn"
		}
		return e
	}
	getInvitation := matrixRow{op: "getWorkspaceInvitation", cells: every(cellOK), request: func(c caller, s seeded) (string, string, string) {
		return http.MethodGet, "/api/v0/workspace-invitations/" + s.invitation(workspaceOf(c), "newcomer@example.com").String() + "?token=t", ""
	}}
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
		name string
		ops  []apitest.Operation
		rows []matrixRow
		want []string
````

````new server/internal/bootstrap/permission_matrix_coverage_test.go
		name   string
		ops    []apitest.Operation
		exempt matrixExemptions
		rows   []matrixRow
		want   []string
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
		{"an operation without a row", append(ops, apitest.Operation{ID: "updateWorkspace", Tags: []string{"workspace"}}),
````

````new server/internal/bootstrap/permission_matrix_coverage_test.go
		{"an operation without a row", append(ops, apitest.Operation{ID: "updateWorkspace", Tags: []string{"workspace"}}), exempt,
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
		{"an operation of a module no list names", append(ops, apitest.Operation{ID: "listProjects", Tags: []string{"project"}}),
````

````new server/internal/bootstrap/permission_matrix_coverage_test.go
		{"a public operation without a row or an exemption", ops, exemptPublic(), []matrixRow{row},
			[]string{"operation getWorkspaceInvitation, tagged [workspace], has no row"}},
		{"a public exemption of no operation", ops, exemptPublic("getWorkspaceInvitation", "getInvitation"), []matrixRow{row},
			[]string{"the public exemption getInvitation names no operation of the contract"}},
		{"a public exemption of an operation that needs a token",
			append(ops, apitest.Operation{ID: "deleteWorkspace", Tags: []string{"workspace"}, Method: http.MethodDelete, Path: "/api/v0/workspaces/{slug}"}),
			exemptPublic("getWorkspaceInvitation", "deleteWorkspace"), []matrixRow{row},
			[]string{"operation deleteWorkspace is exempt as public, but needs a token"}},
		{"a public exemption with a row", ops, exempt, []matrixRow{row, getInvitation},
			[]string{"operation getWorkspaceInvitation is exempt as public, and has a row"}},
		{"an operation of a module no list names", append(ops, apitest.Operation{ID: "listProjects", Tags: []string{"project"}}), exempt,
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
		{"an operation without a tag", append(ops, apitest.Operation{ID: "getHealth"}),
````

````new server/internal/bootstrap/permission_matrix_coverage_test.go
		{"an operation without a tag", append(ops, apitest.Operation{ID: "getHealth"}), exempt,
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
		{"an operation with an exempt tag and another", append(ops, apitest.Operation{ID: "getMyWorkspaces", Tags: []string{"identity", "workspace"}}),
````

````new server/internal/bootstrap/permission_matrix_coverage_test.go
		{"an operation with an exempt tag and another", append(ops, apitest.Operation{ID: "getMyWorkspaces", Tags: []string{"identity", "workspace"}}), exempt,
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
		{"a row of no operation", ops, []matrixRow{row, {op: "renameWorkspace", request: get, cells: every(cellOK)}},
````

````new server/internal/bootstrap/permission_matrix_coverage_test.go
		{"a row of no operation", ops, exempt, []matrixRow{row, {op: "renameWorkspace", request: get, cells: every(cellOK)}},
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
		{"a row without a cell", ops, []matrixRow{partial}, []string{
````

````new server/internal/bootstrap/permission_matrix_coverage_test.go
		{"a row without a cell", ops, exempt, []matrixRow{partial}, []string{
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
		{"a row that sends another operation", ops, []matrixRow{{op: "getWorkspace", request: sameRequest(http.MethodGet, "/api/v0/workspaces", ""),
````

````new server/internal/bootstrap/permission_matrix_coverage_test.go
		{"a row that sends another operation", ops, exempt, []matrixRow{{op: "getWorkspace", request: sameRequest(http.MethodGet, "/api/v0/workspaces", ""),
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
		{"a cell of another method", ops, []matrixRow{posting},
````

````new server/internal/bootstrap/permission_matrix_coverage_test.go
		{"a cell of another method", ops, exempt, []matrixRow{posting},
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
		{"a cell of another path", ops, []matrixRow{guestSends(http.MethodGet, "/api/v0/workspace-slugs/acme")},
````

````new server/internal/bootstrap/permission_matrix_coverage_test.go
		{"a cell of another path", ops, exempt, []matrixRow{guestSends(http.MethodGet, "/api/v0/workspace-slugs/acme")},
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
		{"a cell with an empty parameter", ops, []matrixRow{guestSends(http.MethodGet, "/api/v0/workspaces/")},
````

````new server/internal/bootstrap/permission_matrix_coverage_test.go
		{"a cell with an empty parameter", ops, exempt, []matrixRow{guestSends(http.MethodGet, "/api/v0/workspaces/")},
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
		{"a cell with a longer path", ops, []matrixRow{guestSends(http.MethodGet, "/api/v0/workspaces/acme/members")},
````

````new server/internal/bootstrap/permission_matrix_coverage_test.go
		{"a cell with a longer path", ops, exempt, []matrixRow{guestSends(http.MethodGet, "/api/v0/workspaces/acme/members")},
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
		{"a write without write", append(ops, creates), []matrixRow{row,
````

````new server/internal/bootstrap/permission_matrix_coverage_test.go
		{"a write without write", append(ops, creates), exempt, []matrixRow{row,
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
		{"a cell that targets another column's workspace", ops,
````

````new server/internal/bootstrap/permission_matrix_coverage_test.go
		{"a cell that targets another column's workspace", ops, exempt,
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
		{"a cell of one's settings in another column's workspace", append(ops, preferences), []matrixRow{row, otherPreferences},
````

````new server/internal/bootstrap/permission_matrix_coverage_test.go
		{"a cell of one's settings in another column's workspace", append(ops, preferences), exempt, []matrixRow{row, otherPreferences},
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
		{"a membership of another column's workspace", append(ops, membership), []matrixRow{row, deletedNames("acme", callerMember)},
````

````new server/internal/bootstrap/permission_matrix_coverage_test.go
		{"a membership of another column's workspace", append(ops, membership), exempt, []matrixRow{row, deletedNames("acme", callerMember)},
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
		{"an id no seeded row has", append(ops, membership), []matrixRow{row, guestNames("/api/v0/workspace-members/" + uuid.Nil().String())},
````

````new server/internal/bootstrap/permission_matrix_coverage_test.go
		{"an id no seeded row has", append(ops, membership), exempt, []matrixRow{row, guestNames("/api/v0/workspace-members/" + uuid.Nil().String())},
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
		if got := matrixViolations(tt.ops, exempt, tt.rows, s); !slices.Equal(got, tt.want) {
````

````new server/internal/bootstrap/permission_matrix_coverage_test.go
		if got := matrixViolations(tt.ops, tt.exempt, tt.rows, s); !slices.Equal(got, tt.want) {
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
		name   string
		exempt []string
		want   []string
````

````new server/internal/bootstrap/permission_matrix_coverage_test.go
		name    string
		modules []string
		want    []string
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
		if got := matrixViolations(ops, tt.exempt, []matrixRow{row}, s); !slices.Equal(got, tt.want) {
````

````new server/internal/bootstrap/permission_matrix_coverage_test.go
		if got := matrixViolations(ops, matrixExemptions{modules: tt.modules, public: exempt.public}, []matrixRow{row}, s); !slices.Equal(got, tt.want) {
````

`server/internal/bootstrap/permission_matrix_invitations_test.go`（修改，4 处）：

````old server/internal/bootstrap/permission_matrix_invitations_test.go
	"log/slog"
````

````new server/internal/bootstrap/permission_matrix_invitations_test.go
	"log/slog"
	"maps"
````

````old server/internal/bootstrap/permission_matrix_invitations_test.go
	"slices"
````

````new server/internal/bootstrap/permission_matrix_invitations_test.go
	"slices"
	"strings"
````

````old server/internal/bootstrap/permission_matrix_invitations_test.go
	workspacedomain "github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
````

````new server/internal/bootstrap/permission_matrix_invitations_test.go
	workspacedomain "github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
````

````old server/internal/bootstrap/permission_matrix_invitations_test.go
		t.Errorf("%s's invitation answers %+v, want invitee@example.com's, as a guest, with its token", c, list.Data)
	}
}

````

````new server/internal/bootstrap/permission_matrix_invitations_test.go
		t.Errorf("%s's invitation answers %+v, want invitee@example.com's, as a guest, with its token", c, list.Data)
	}
}

// wholeAnswer is an answer as a caller could compare two: the status, the
// body and every header but the two that differ by request (Date,
// X-Request-Id).
type wholeAnswer struct {
	status  int
	body    string
	headers string
}

// wholeAnswerOf is res, whose body is body, as a wholeAnswer.
func wholeAnswerOf(res *http.Response, body []byte) wholeAnswer {
	header := res.Header.Clone()
	header.Del("Date")
	header.Del("X-Request-Id")
	var lines []string
	for _, name := range slices.Sorted(maps.Keys(header)) {
		lines = append(lines, name+": "+strings.Join(header[name], ", "))
	}
	return wholeAnswer{res.StatusCode, string(body), strings.Join(lines, "\n")}
}

// The public getWorkspaceInvitation has no row (matrixExempt): its route
// reads no credential. This test stands for the row. Each link is asked
// with no bearer token and with each column's, and every caller gets the
// same answer: acme's invitation with its token, 200, without the address;
// and one 404 workspace.invitation_not_found, the same byte for byte for a
// character of the token changed, another invitation's token, gone's
// invitation, deleted with gone, with its own token, and an id no
// invitation has (M3 design 8.2): nothing tells a wrong token from a
// missing invitation.
func TestTheInvitationLinkAnswersEveryCallerAlike(t *testing.T) {
	d := prepareMatrix(t)
	contract := apitest.Load(t)
	base := startApp(t, d.config(t, pgtest.NewDatabaseFrom(t, d.url), nil), migrations.FS())
	s := d.seeded.in(t)
	acme, gone, nobodys := s.invitation("acme", "newcomer@example.com"), s.invitation("gone", "newcomer@example.com"), uuid.NewV7()
	right := invitationToken(t, acme)
	changed := []byte(right)
	changed[len("nrv_inv_")+10] = 'A'
	if string(changed) == right {
		changed[len("nrv_inv_")+10] = 'B'
	}
	links := []struct {
		name  string
		id    uuid.UUID
		token string
		shows bool // the link that shows its invitation; every other is the one 404
	}{
		{"acme's invitation", acme, right, true},
		{"a character changed", acme, string(changed), false},
		{"another invitation's token", acme, invitationToken(t, gone), false},
		{"gone's invitation", gone, invitationToken(t, gone), false},
		{"no invitation", nobodys, invitationToken(t, nobodys), false},
	}
	callers := append([]caller{"nobody"}, workspaceColumns...)
	var notFound *wholeAnswer
	for _, l := range links {
		var first wholeAnswer
		for i, c := range callers {
			req := newRequest(t, http.MethodGet, base+"/api/v0/workspace-invitations/"+l.id.String()+"?token="+l.token, d.tokens[c], nil)
			contract.CheckRequest(t, req)
			res, body := send(t, req)
			contract.CheckResponse(t, req, res)
			got := wholeAnswerOf(res, body)
			if i == 0 {
				first = got
			} else if got != first {
				t.Errorf("%s asked by %s = %+v, want what %s got: %+v", l.name, c, got, callers[0], first)
			}
		}
		switch {
		case l.shows:
			var p struct {
				ID            uuid.UUID `json:"id"`
				Role          int       `json:"role"`
				Declined      bool      `json:"declined"`
				WorkspaceSlug string    `json:"workspace_slug"`
			}
			decodeAnswer(t, first.body, &p)
			if first.status != http.StatusOK || p.ID != acme || p.Role != 15 || p.Declined || p.WorkspaceSlug != "acme" ||
				strings.Contains(first.body, "email") || strings.Contains(first.body, "newcomer") {
				t.Errorf("%s = %d %s; want 200, acme's pending invitation as a member, without the address", l.name, first.status, first.body)
			}
		case notFound == nil:
			if first.status != http.StatusNotFound || problemCode(t, []byte(first.body)) != "workspace.invitation_not_found" {
				t.Errorf("%s = %d %s; want 404 workspace.invitation_not_found", l.name, first.status, first.body)
			}
			notFound = &first
		case first != *notFound:
			t.Errorf("%s = %+v; want the same answer as %s: %+v", l.name, first, links[1].name, *notFound)
		}
	}
}

````

- [ ] **Step 7: 测试、lint**

Run: `go -C server test -count=1 ./internal/modules/workspace/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestTheInvitationLinksTokenIsNotLogged|TestTheInvitationLinkAnswersEveryCallerAlike|TestPermissionMatrix$|TestThePermissionMatrixCoversEveryOperation|TestMatrixViolationsCatchesEachGap|TestPublicOperationsAreTheContractsPublicOperations|TestParametersThatDoNotBindAnswer400' ./internal/bootstrap/`
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
git add api/modules/workspace.yaml server/internal/bootstrap/app.go server/internal/bootstrap/invitations_test.go server/internal/bootstrap/permission_matrix_coverage_test.go server/internal/bootstrap/permission_matrix_invitations_test.go server/internal/bootstrap/permission_matrix_test.go server/internal/modules/workspace/adapter/http/handler.go server/internal/modules/workspace/adapter/http/handler_test.go server/internal/modules/workspace/adapter/http/invitations.go server/internal/modules/workspace/adapter/http/invitations_test.go server/internal/modules/workspace/adapter/postgres/invitations.go server/internal/modules/workspace/adapter/postgres/invitations_test.go server/internal/modules/workspace/adapter/postgres/queries/invitations.sql server/internal/modules/workspace/app/fakes_invitations_test.go server/internal/modules/workspace/app/get_invitation.go server/internal/modules/workspace/app/get_invitation_test.go server/internal/modules/workspace/app/invitation_ports.go server/internal/modules/workspace/app/tokens.go server/internal/modules/workspace/domain/invitation.go server/internal/modules/workspace/module.go api/dist/openapi.yaml server/internal/modules/workspace/adapter/http/gen/server.gen.go server/internal/modules/workspace/adapter/postgres/gen/invitations.sql.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M3/P3): the invitation link's public view

GET /api/v0/workspace-invitations/{invitation_id}?token=: the role,
whether it was declined, and the workspace's name and slug, never the
address (decision 1). A wrong token reads nothing and is the one 404 of
an invitation that does not exist (M3 design 8.2); the answer does not
depend on the caller; the access log has the path alone. The matrix
exempts the public operation for the test that stands for its row, and
reports any path parameter it does not know.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A）：

| 改坏 | 必须失败的测试 |
|---|---|
| 令牌不对答 403 | `TestGetWorkspaceInvitationRefusesAWrongTokenBeforeReading`、`TestTheInvitationLinkAnswersEveryCallerAlike` |
| 先读邀请再核对令牌 | `TestGetWorkspaceInvitationRefusesAWrongTokenBeforeReading` |
| 格式对的令牌都有效 | `TestGetWorkspaceInvitationRefusals`、`TestTheInvitationLinkAnswersEveryCallerAlike` |
| 消息不含 id（每个邀请同一个令牌） | `TestTheInvitationLinkAnswersEveryCallerAlike`（另有 Task 2 的测试） |
| 组合根漏掉 `workspace` 的公开操作 | `TestPublicOperationsAreTheContractsPublicOperations`、`TestTheInvitationLinkAnswersEveryCallerAlike` |
| 访问日志记查询参数 | `TestTheInvitationLinksTokenIsNotLogged` |
| 显示已接受或已删除的邀请；显示已删除工作区的邀请；从不说已忽略；读失败答 404 | `TestInvitationPreview` |
| `targetViolation` 放过不认识的参数 | `TestMatrixViolationsCatchesEachGap` |

**Done when:** 用例、存储、HTTP 通过；公开的查看对每个调用者相同、四种坏链接同一个 404；访问日志里没有令牌；矩阵的完整性核对通过。

---

### Task 10: 回应要的存储

**Files:**
- Create: `server/internal/modules/workspace/adapter/postgres/responses.go`、`server/internal/modules/workspace/adapter/postgres/responses_test.go`
- Modify: `server/internal/modules/workspace/adapter/postgres/queries/invitations.sql`、`server/internal/modules/workspace/adapter/postgres/queries/members.sql`、`server/internal/modules/workspace/adapter/postgres/queries/workspaces.sql`
- Generate: `server/internal/modules/workspace/adapter/postgres/gen/invitations.sql.go`、`server/internal/modules/workspace/adapter/postgres/gen/members.sql.go`、`server/internal/modules/workspace/adapter/postgres/gen/workspaces.sql.go`

**Interfaces:**
- Produces（spec 2.11，M3 设计 3.8）：存储 `MemberOf(ctx, workspaceID, userID) (domain.Membership, bool, error)`（未删除的，有效或已结束）、`RestoreMember(ctx, id, role, by, now) error`、`AcceptInvitation(ctx, id, by, now) error`（`accepted`、`responded_at = deleted_at = now`）、`DeclineInvitation(ctx, id, by, now) error`、`WorkspaceByID(ctx, id) (domain.Workspace, error)`（有效成员数，事务里读；没有这一行是错误）。
- 使用者：Task 11。

**Tests:**（`adapter/postgres/responses_test.go`，真实数据库）
- `TestMemberOf`（有效的、已结束的都找到；已删除的、别的工作区的、没有的都是"没有"；失败是错误，不是"没有"）。
- `TestRestoreMember`（那一行恢复为有效、角色、恢复者、时刻；同一个工作区另一份已结束的和他在别的工作区已结束的不变；失败是错误）。
- `TestAnsweringAnInvitation`（接受：记下并在同一时刻删除，邮箱空出来；忽略：记下、不删除、仍占着邮箱；同一个工作区的另一份和别的工作区发给同一个邮箱的仍待接受；失败是错误）。
- `TestWorkspaceByID`（只数有效的成员关系，照读它的事务所见；已删除的工作区、失败都是错误，不是 `ErrNotFound`）。

- [ ] **Step 1: 查询**

`server/internal/modules/workspace/adapter/postgres/queries/invitations.sql`（修改，1 处）：

````old server/internal/modules/workspace/adapter/postgres/queries/invitations.sql
WHERE id = sqlc.arg(id);

````

````new server/internal/modules/workspace/adapter/postgres/queries/invitations.sql
WHERE id = sqlc.arg(id);

-- name: AcceptInvitation :exec
-- acceptWorkspaceInvitation, under the workspace's FOR NO KEY UPDATE and the invitation's FOR UPDATE: accepted, and
-- deleted at the same moment (M3 design 3.8).
UPDATE workspace_member_invites
SET accepted = true, responded_at = sqlc.arg(now)::timestamptz, deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now),
    updated_by_id = sqlc.arg(accepted_by)::uuid
WHERE id = sqlc.arg(id);

-- name: DeclineInvitation :exec
-- declineWorkspaceInvitation, under the workspace's FOR SHARE and the invitation's FOR UPDATE: it stays, and holds its
-- address (M3 design 3.8).
UPDATE workspace_member_invites
SET responded_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(declined_by)::uuid
WHERE id = sqlc.arg(id);

````

`server/internal/modules/workspace/adapter/postgres/queries/members.sql`（修改，1 处）：

````old server/internal/modules/workspace/adapter/postgres/queries/members.sql
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;
````

````new server/internal/modules/workspace/adapter/postgres/queries/members.sql
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

-- name: MemberOf :one
-- acceptWorkspaceInvitation, under the workspace's FOR NO KEY UPDATE: the user's undeleted membership, active or
-- ended; the partial unique index holds at most one.
SELECT id, workspace_id, member_id, role, is_active, created_at
FROM workspace_members
WHERE workspace_id = sqlc.arg(workspace_id) AND member_id = sqlc.arg(member_id) AND deleted_at IS NULL;

-- name: RestoreMember :exec
-- acceptWorkspaceInvitation, under the workspace's FOR NO KEY UPDATE: an ended membership active again, with the
-- invitation's role (M3 design 3.8).
UPDATE workspace_members
SET is_active = true, role = sqlc.arg(role), updated_by_id = sqlc.arg(restored_by), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);
````

`server/internal/modules/workspace/adapter/postgres/queries/workspaces.sql`（修改，1 处）：

````old server/internal/modules/workspace/adapter/postgres/queries/workspaces.sql
WHERE w.slug = sqlc.arg(slug) AND w.deleted_at IS NULL;
````

````new server/internal/modules/workspace/adapter/postgres/queries/workspaces.sql
WHERE w.slug = sqlc.arg(slug) AND w.deleted_at IS NULL;

-- name: WorkspaceByID :one
-- acceptWorkspaceInvitation's answer, read in its transaction after the membership changed.
SELECT w.id, w.name, w.slug, w.organization_size, w.timezone, w.created_at, w.updated_at,
       (SELECT count(*) FROM workspace_members c
        WHERE c.workspace_id = w.id AND c.is_active AND c.deleted_at IS NULL) AS total_members
FROM workspaces w
WHERE w.id = sqlc.arg(id) AND w.deleted_at IS NULL;
````

- [ ] **Step 2: 生成**

Run: `make gen-go`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `a718a9fa6f053f234ba4409ecc90c726ce4dda5d905eacf751c948db8dc95255` | 298 | `server/internal/modules/workspace/adapter/postgres/gen/invitations.sql.go` |
| `cd34adf38d229814f5526555074060dd38c309a05c4b4c4248b43d678587789a` | 259 | `server/internal/modules/workspace/adapter/postgres/gen/members.sql.go` |
| `fd6f2f209d0d947b080533b28d882c62e50158d6b6151b3a8305eec3d96d3f63` | 351 | `server/internal/modules/workspace/adapter/postgres/gen/workspaces.sql.go` |

Run: `shasum -a 256 server/internal/modules/workspace/adapter/postgres/gen/invitations.sql.go server/internal/modules/workspace/adapter/postgres/gen/members.sql.go server/internal/modules/workspace/adapter/postgres/gen/workspaces.sql.go`
Expected: 与上表相同。

- [ ] **Step 3: 存储和它的测试**

`server/internal/modules/workspace/adapter/postgres/responses.go`（新文件，73 行）：

````file server/internal/modules/workspace/adapter/postgres/responses.go
package postgresadapter

import (
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The writes of the answers to an invitation (M3 design 3.8), each under
// the locks the use case holds.

// MemberOf returns userID's undeleted membership of the workspace, active
// or ended; found is false when he has none.
func (s *Store) MemberOf(ctx context.Context, workspaceID, userID uuid.UUID) (domain.Membership, bool, error) {
	r, err := s.queries(ctx).MemberOf(ctx, gen.MemberOfParams{WorkspaceID: workspaceID, MemberID: userID})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.Membership{}, false, nil
	case err != nil:
		return domain.Membership{}, false, fmt.Errorf("read workspace member: %w", err)
	}
	return domain.Membership{ID: r.ID, WorkspaceID: r.WorkspaceID, MemberID: r.MemberID, Role: shared.Role(r.Role),
		IsActive: r.IsActive, CreatedAt: r.CreatedAt}, true, nil
}

// RestoreMember makes the membership id active again with role, by the
// account by at now.
func (s *Store) RestoreMember(ctx context.Context, id uuid.UUID, role shared.Role, by uuid.UUID, now time.Time) error {
	if err := s.queries(ctx).RestoreMember(ctx, gen.RestoreMemberParams{ID: id, Role: int16(role), RestoredBy: &by, Now: now}); err != nil {
		return fmt.Errorf("restore workspace member: %w", err)
	}
	return nil
}

// AcceptInvitation records the invitation id as accepted by the account by
// at now, and deletes it.
func (s *Store) AcceptInvitation(ctx context.Context, id, by uuid.UUID, now time.Time) error {
	if err := s.queries(ctx).AcceptInvitation(ctx, gen.AcceptInvitationParams{ID: id, AcceptedBy: by, Now: now}); err != nil {
		return fmt.Errorf("accept workspace invitation: %w", err)
	}
	return nil
}

// DeclineInvitation records the invitation id as declined by the account by
// at now.
func (s *Store) DeclineInvitation(ctx context.Context, id, by uuid.UUID, now time.Time) error {
	if err := s.queries(ctx).DeclineInvitation(ctx, gen.DeclineInvitationParams{ID: id, DeclinedBy: by, Now: now}); err != nil {
		return fmt.Errorf("decline workspace invitation: %w", err)
	}
	return nil
}

// WorkspaceByID returns the undeleted workspace id and its number of active
// members, without a role. The caller holds the workspace's lock: its
// absence is an error, not app.ErrNotFound.
func (s *Store) WorkspaceByID(ctx context.Context, id uuid.UUID) (domain.Workspace, error) {
	r, err := s.queries(ctx).WorkspaceByID(ctx, id)
	if err != nil {
		return domain.Workspace{}, fmt.Errorf("read workspace: %w", err)
	}
	return domain.Workspace{
		ID: r.ID, Name: r.Name, Slug: r.Slug, OrganizationSize: r.OrganizationSize, Timezone: r.Timezone,
		TotalMembers: int(r.TotalMembers), CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}, nil
}
````

`server/internal/modules/workspace/adapter/postgres/responses_test.go`（新文件，202 行）：

````file server/internal/modules/workspace/adapter/postgres/responses_test.go
package postgresadapter_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// MemberOf finds the user's undeleted membership of that workspace, active
// or ended; none for a deleted row, another workspace's row, or no row. A
// failed read is its error, never "none".
func TestMemberOf(t *testing.T) {
	s, pool := newStore(t)
	alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	acme, beta := newWorkspace(t, s, "Acme", "acme", alice), newWorkspace(t, s, "Beta", "beta", alice)
	bobIn := joinAt(t, s, acme.ID, bob, shared.RoleMember, now)
	carolIn := joinAt(t, s, acme.ID, carol, shared.RoleGuest, now)
	exec(t, pool, "UPDATE workspace_members SET is_active = false WHERE id = $1", carolIn.ID)
	carolIn.IsActive = false
	gone := joinAt(t, s, beta.ID, carol, shared.RoleMember, now)
	exec(t, pool, "UPDATE workspace_members SET deleted_at = $2 WHERE id = $1", gone.ID, now)

	for _, want := range []domain.Membership{bobIn, carolIn} {
		if got, found, err := s.MemberOf(context.Background(), acme.ID, want.MemberID); err != nil || !found || got != want {
			t.Errorf("MemberOf(acme, %s) = %+v, %v, %v; want %+v", want.MemberID, got, found, err, want)
		}
	}
	for _, tt := range []struct {
		name            string
		workspace, user uuid.UUID
	}{{"a deleted row", beta.ID, carol}, {"another workspace's", beta.ID, bob}, {"no row", acme.ID, uuid.NewV7()}} {
		if got, found, err := s.MemberOf(context.Background(), tt.workspace, tt.user); err != nil || found || got != (domain.Membership{}) {
			t.Errorf("%s: MemberOf() = %+v, %v, %v; want none", tt.name, got, found, err)
		}
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if got, found, err := s.MemberOf(cancelled, acme.ID, bob); !errors.Is(err, context.Canceled) || found || got != (domain.Membership{}) {
		t.Errorf("MemberOf() on a cancelled context = %+v, %v, %v; want context.Canceled", got, found, err)
	}
}

// memberAudit is a membership's state and audit columns.
type memberAudit struct {
	role      shared.Role
	active    bool
	updatedBy uuid.UUID
	updatedAt time.Time
}

func memberAuditOf(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) memberAudit {
	t.Helper()
	var a memberAudit
	if err := pool.QueryRow(context.Background(), "SELECT role, is_active, updated_by_id, updated_at FROM workspace_members WHERE id = $1", id).
		Scan(&a.role, &a.active, &a.updatedBy, &a.updatedAt); err != nil {
		t.Fatal(err)
	}
	return a
}

// RestoreMember makes that membership active again with the role, the
// restorer and the time; another ended membership of the workspace and the
// member's ended one in another workspace stay ended. A failed write is its
// error.
func TestRestoreMember(t *testing.T) {
	s, pool := newStore(t)
	alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	acme, beta := newWorkspace(t, s, "Acme", "acme", alice), newWorkspace(t, s, "Beta", "beta", alice)
	bobIn, carolIn, bobInBeta := joinAt(t, s, acme.ID, bob, shared.RoleAdmin, now), joinAt(t, s, acme.ID, carol, shared.RoleMember, now),
		joinAt(t, s, beta.ID, bob, shared.RoleMember, now)
	exec(t, pool, "UPDATE workspace_members SET is_active = false WHERE id = ANY($1)", []uuid.UUID{bobIn.ID, carolIn.ID, bobInBeta.ID})
	later := now.Add(time.Hour)

	if err := s.RestoreMember(context.Background(), bobIn.ID, shared.RoleGuest, bob, later); err != nil {
		t.Fatalf("RestoreMember() = %v", err)
	}

	if got, want := memberAuditOf(t, pool, bobIn.ID), (memberAudit{shared.RoleGuest, true, bob, later}); got.role != want.role ||
		got.active != want.active || got.updatedBy != want.updatedBy || !got.updatedAt.Equal(want.updatedAt) {
		t.Errorf("bob's in acme: %+v, want %+v", got, want)
	}
	for _, m := range []domain.Membership{carolIn, bobInBeta} {
		if got := memberAuditOf(t, pool, m.ID); got.active || got.role != m.Role || !got.updatedAt.Equal(now) {
			t.Errorf("%s in %s: %+v; want it ended, unchanged", m.MemberID, m.WorkspaceID, got)
		}
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := s.RestoreMember(cancelled, carolIn.ID, shared.RoleGuest, bob, later); !errors.Is(err, context.Canceled) {
		t.Errorf("RestoreMember() on a cancelled context = %v; want context.Canceled", err)
	}
}

// AcceptInvitation records that invitation accepted and deletes it, at the
// same moment, with the accepter; its address is free again. DeclineInvitation
// records it declined, undeleted: it holds its address. Another invitation
// of the workspace and one to the same address in another workspace stay
// pending. A failed write is its error.
func TestAnsweringAnInvitation(t *testing.T) {
	for _, tt := range []struct {
		name     string
		accepted bool
	}{{"AcceptInvitation", true}, {"DeclineInvitation", false}} {
		t.Run(tt.name, func(t *testing.T) {
			s, pool := newStore(t)
			answer := s.DeclineInvitation
			if tt.accepted {
				answer = s.AcceptInvitation
			}
			alice, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "carol@corp.com")
			acme, beta := newWorkspace(t, s, "Acme", "acme", alice), newWorkspace(t, s, "Beta", "beta", alice)
			answered := invite(t, s, acme.ID, "carol@corp.com", shared.RoleMember, alice)
			invite(t, s, acme.ID, "dave@corp.com", shared.RoleMember, alice)
			invite(t, s, beta.ID, "carol@corp.com", shared.RoleMember, alice)
			later := now.Add(time.Hour)

			if err := answer(context.Background(), answered.ID, carol, later); err != nil {
				t.Fatalf("%s() = %v", tt.name, err)
			}

			var accepted bool
			var responded, deleted *time.Time
			var updatedBy uuid.UUID
			var updatedAt time.Time
			if err := pool.QueryRow(context.Background(), `SELECT accepted, responded_at, deleted_at, updated_by_id, updated_at
				FROM workspace_member_invites WHERE id = $1`, answered.ID).Scan(&accepted, &responded, &deleted, &updatedBy, &updatedAt); err != nil {
				t.Fatal(err)
			}
			wantDeleted := tt.accepted
			if accepted != tt.accepted || responded == nil || !responded.Equal(later) || (deleted != nil) != wantDeleted ||
				(deleted != nil && !deleted.Equal(later)) || updatedBy != carol || !updatedAt.Equal(later) {
				t.Errorf("the answered invitation: accepted %v, responded %v, deleted %v, by %s at %v; want accepted %v at %v, deleted %v",
					accepted, responded, deleted, updatedBy, updatedAt, tt.accepted, later, wantDeleted)
			}
			wantAcme := []string{"dave@corp.com"}
			if !tt.accepted {
				wantAcme = []string{"carol@corp.com", "dave@corp.com"}
			}
			if got := pendingEmails(t, pool, acme.ID); !slices.Equal(got, wantAcme) {
				t.Errorf("acme's undeleted invitations: %q, want %q", got, wantAcme)
			}
			var pending int
			if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM workspace_member_invites
				WHERE id <> $1 AND responded_at IS NULL AND NOT accepted AND deleted_at IS NULL AND updated_at = created_at`, answered.ID).
				Scan(&pending); err != nil || pending != 2 {
				t.Errorf("the other invitations still pending, unchanged: %d, %v; want 2", pending, err)
			}
			_, err := s.CreateInvitations(context.Background(), []app.InvitationRow{
				{ID: uuid.NewV7(), WorkspaceID: acme.ID, Email: "carol@corp.com", Role: shared.RoleGuest, CreatedBy: alice, Now: later},
			})
			var dup *app.DuplicateInvitation
			if tt.accepted == errors.As(err, &dup) {
				t.Errorf("inviting carol@corp.com again = %v; want it free again only once accepted", err)
			}
			cancelled, cancel := context.WithCancel(context.Background())
			cancel()
			if err := answer(cancelled, answered.ID, carol, later); !errors.Is(err, context.Canceled) {
				t.Errorf("%s() on a cancelled context = %v; want context.Canceled", tt.name, err)
			}
		})
	}
}

// WorkspaceByID reads the undeleted workspace, counting its active
// memberships only, as the transaction that reads it sees them; a deleted
// workspace, which its caller's lock rules out, and a failed read are
// errors, never app.ErrNotFound.
func TestWorkspaceByID(t *testing.T) {
	s, pool := newStore(t)
	alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	acme, beta := newWorkspace(t, s, "Acme", "acme", alice), newWorkspace(t, s, "Beta", "beta", alice)
	joinAt(t, s, acme.ID, bob, shared.RoleMember, now)
	ended := joinAt(t, s, acme.ID, carol, shared.RoleMember, now)
	exec(t, pool, "UPDATE workspace_members SET is_active = false WHERE id = $1", ended.ID)
	exec(t, pool, "UPDATE workspaces SET deleted_at = $2 WHERE id = $1", beta.ID, now)

	got, err := s.WorkspaceByID(context.Background(), acme.ID)
	want := acme
	want.Role, want.TotalMembers = 0, 2
	if err != nil || !sameWorkspace(got, want) || got.TotalMembers != 2 {
		t.Errorf("WorkspaceByID(acme) = %+v, %v; want %+v with 2 members", got, err, want)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	for name, tt := range map[string]struct {
		ctx context.Context
		id  uuid.UUID
	}{"deleted": {context.Background(), beta.ID}, "cancelled": {cancelled, acme.ID}} {
		if got, err := s.WorkspaceByID(tt.ctx, tt.id); err == nil || errors.Is(err, app.ErrNotFound) || got.ID != (uuid.UUID{}) {
			t.Errorf("WorkspaceByID(), %s = %+v, %v; want an error that is not app.ErrNotFound", name, got, err)
		}
	}
}
````

- [ ] **Step 4: 测试、lint**

Run: `go -C server test -count=1 ./internal/modules/workspace/adapter/postgres/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 5: 提交**

```bash
git add server/internal/modules/workspace/adapter/postgres/queries/invitations.sql server/internal/modules/workspace/adapter/postgres/queries/members.sql server/internal/modules/workspace/adapter/postgres/queries/workspaces.sql server/internal/modules/workspace/adapter/postgres/responses.go server/internal/modules/workspace/adapter/postgres/responses_test.go server/internal/modules/workspace/adapter/postgres/gen/invitations.sql.go server/internal/modules/workspace/adapter/postgres/gen/members.sql.go server/internal/modules/workspace/adapter/postgres/gen/workspaces.sql.go
```
```bash
git commit -m "feat(M3/P3): the store's writes of an answer to an invitation

MemberOf finds a user's membership of a workspace, active or ended;
RestoreMember makes an ended one active again with a role; an accepted
invitation is recorded and deleted at one moment, a declined one only
recorded, holding its address (M3 design 3.8); WorkspaceByID reads the
workspace with its active members for the acceptance's answer.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（改生成的查询常量；spec 附录 A）：

| 改坏 | 必须失败的测试 |
|---|---|
| `MemberOf` 读已删除的；读别的工作区的 | `TestMemberOf` |
| `MemberOf` 把失败答成"没有" | `TestMemberOf` |
| `RestoreMember` 保留原角色；不恢复；恢复每一行 | `TestRestoreMember` |
| 接受的邀请不删除；`AcceptInvitation`、`DeclineInvitation` 改每一行；忽略时删除 | `TestAnsweringAnInvitation` |
| 回答的人数含已结束的 | `TestWorkspaceByID` |
| `RestoreMember`、`AcceptInvitation`、`DeclineInvitation`、`WorkspaceByID` 吞掉错误 | 各自的测试 |

**Done when:** 4 个存储测试在真实数据库上通过（两个工作区、已结束的和已删除的行）；生成物的 SHA-256 与表相同。

---

### Task 11: `acceptWorkspaceInvitation`、`declineWorkspaceInvitation`

**Files:**
- Create: `server/internal/modules/workspace/app/accept_invitation.go`、`server/internal/modules/workspace/app/accept_invitation_test.go`、`server/internal/modules/workspace/app/decline_invitation.go`、`server/internal/modules/workspace/app/decline_invitation_test.go`、`server/internal/modules/workspace/app/respond_invitation.go`
- Modify: `api/modules/workspace.yaml`、`api/openapi.yaml`、`server/internal/bootstrap/permission_matrix_coverage_test.go`、`server/internal/bootstrap/permission_matrix_invitations_test.go`、`server/internal/bootstrap/permission_matrix_seeded_test.go`、`server/internal/bootstrap/permission_matrix_test.go`、`server/internal/bootstrap/permission_matrix_workspace_test.go`、`server/internal/modules/workspace/adapter/http/handler.go`、`server/internal/modules/workspace/adapter/http/handler_test.go`、`server/internal/modules/workspace/adapter/http/invitations.go`、`server/internal/modules/workspace/adapter/http/invitations_test.go`、`server/internal/modules/workspace/app/clock_test.go`、`server/internal/modules/workspace/app/fakes_invitations_test.go`、`server/internal/modules/workspace/app/invitation_ports.go`、`server/internal/modules/workspace/app/list_invitations_test.go`、`server/internal/modules/workspace/domain/errors.go`、`server/internal/modules/workspace/module.go`、`web/apps/web/helpers/authentication.helper.ts`、`web/packages/i18n/src/locales/en/auth.json`、`web/packages/i18n/src/locales/zh-CN/auth.json`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/workspace/adapter/http/gen/bodyshape.gen.go`、`server/internal/modules/workspace/adapter/http/gen/server.gen.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Produces（spec 2.11，M3 设计 3.6 约定一、六，3.8，9.1）：接口 `POST /api/v0/workspace-invitations/{invitation_id}/accept`（200 `Workspace`）、`/decline`（204），`InvitationResponse{token}`；新码 `workspace.invitation_email_mismatch`（403）和它的文案；`app.InvitationAccepter`、`InvitationDecliner`；`responder{accounts, invitations, tokens}` 的 `checkToken`、`lock`（`app/respond_invitation.go`）；`AcceptInvitationDeps{Accounts, Invitations, Tx, Clock, MAC}`、`NewAcceptWorkspaceInvitation`、`NewDeclineWorkspaceInvitation(accounts, invitations, tx, clock, mac)`。矩阵：四行账户级的格子，`toOwnInvitation`、`toNewcomersInvitation`、`joinsAsAMember`；`notTargets` 加两条回应的路径。
- 使用者：Task 13 的交错 3、12、19；Task 14；P4 在接受恢复成员关系之后加 `DemoteToGuest`（`Execute` 的注释写明位置）。

**Tests:**
- `app/accept_invitation_test.go`：`TestAcceptWorkspaceInvitation`（9.1 的三种：bob 是 `acme` 的成员、`beta` 的访客，邀请的角色高、低、相同都不改，只消费邀请；erin 的已结束的成员关系以邀请的角色恢复；frank 新建；每一种在账户、工作区、邀请的锁之后，一个事务；回答是工作区和调用者现在的角色）；`TestAcceptWorkspaceInvitationRefusals`（`responseRefusals`、`responseFailures`：令牌不对在事务之前、什么都不读；停用的账户 401；不在、期间被删除、工作区期间被删除 404；别的邮箱 403、什么都不写；已忽略的 409；每个失败原样；写和回答的读失败也原样；没有调用者 401）。
- `app/decline_invitation_test.go`：`TestDeclineWorkspaceInvitation`（账户 `FOR SHARE`、工作区 `FOR SHARE`、邀请的锁之后记下回应；有效成员的邀请照样；不读、不写成员关系）；`TestDeclineWorkspaceInvitationRefusals`。
- `app/clock_test.go`：接受、忽略两行。
- `adapter/http/invitations_test.go`：`TestAnsweringAWorkspaceInvitation`（路径的邀请和请求体的令牌交给用例；接受答工作区，忽略 204；请求体没有令牌或多一个字段在用例之前 400；回答不重复令牌）。
- 矩阵：接受、忽略各两行（自己的：每一列 200/204；newcomer 的：每一列 403 `invitation_email_mismatch`），`joinsAsAMember` 核对答案；`-v` 记下矩阵的耗时。

- [ ] **Step 1: 接口描述**

`api/modules/workspace.yaml`（修改，2 处）：

````old api/modules/workspace.yaml
          description: The invitation is deleted.
````

````new api/modules/workspace.yaml
          description: The invitation is deleted.
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/workspace-invitations/{invitation_id}/accept:
    parameters:
      - $ref: '#/components/parameters/InvitationID'
    post:
      operationId: acceptWorkspaceInvitation
      tags: [workspace]
      summary: Accept an invitation to one's own address
      description: >-
        For the account the invitation was sent to: its address, as it is
        when the request runs, must be the invitation's; another answers
        workspace.invitation_email_mismatch, which does not say whose, and
        nothing changes. The caller becomes a member with the invitation's
        role, or a former member has his membership back with it; an active
        member stays as he is, his role too, and only the invitation is
        used. The invitation is then accepted and deleted: its link stops
        working. A token that is not the invitation's, and an invitation
        that does not exist, is deleted or accepted, answer the same
        workspace.invitation_not_found; a declined one,
        workspace.invitation_responded. An account deactivated meanwhile
        answers unauthorized. The answer is the workspace, with the caller's
        role in it.
      security: [{bearer: []}]
      x-problem-codes: [workspace.invitation_not_found, workspace.invitation_email_mismatch, workspace.invitation_responded]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/InvitationResponse'
      responses:
        '200':
          description: The workspace the caller is a member of, with his role.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Workspace'
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/workspace-invitations/{invitation_id}/decline:
    parameters:
      - $ref: '#/components/parameters/InvitationID'
    post:
      operationId: declineWorkspaceInvitation
      tags: [workspace]
      summary: Decline an invitation to one's own address
      description: >-
        For the account the invitation was sent to, as accepting it is, with
        the same answers. The invitation stays, declined: its link shows
        so, and it holds its address until an admin deletes it.
      security: [{bearer: []}]
      x-problem-codes: [workspace.invitation_not_found, workspace.invitation_email_mismatch, workspace.invitation_responded]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/InvitationResponse'
      responses:
        '204':
          description: The invitation is declined.
````

````old api/modules/workspace.yaml
          description: Null until uploads arrive (M5).
          type: [string, 'null']
    WorkspaceInvitationUpdate:
````

````new api/modules/workspace.yaml
          description: Null until uploads arrive (M5).
          type: [string, 'null']
    InvitationResponse:
      type: object
      additionalProperties: false
      required: [token]
      properties:
        token:
          description: The token of the invitation's link (WorkspaceInvitation.token).
          type: string
    WorkspaceInvitationUpdate:
````

`api/openapi.yaml`（修改，1 处）：

````old api/openapi.yaml
    $ref: 'modules/workspace.yaml#/paths/~1api~1v0~1workspace-invitations~1{invitation_id}'
````

````new api/openapi.yaml
    $ref: 'modules/workspace.yaml#/paths/~1api~1v0~1workspace-invitations~1{invitation_id}'
  /api/v0/workspace-invitations/{invitation_id}/accept:
    $ref: 'modules/workspace.yaml#/paths/~1api~1v0~1workspace-invitations~1{invitation_id}~1accept'
  /api/v0/workspace-invitations/{invitation_id}/decline:
    $ref: 'modules/workspace.yaml#/paths/~1api~1v0~1workspace-invitations~1{invitation_id}~1decline'
````

- [ ] **Step 2: 生成**

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `273050ec12d402f34dca33cb92ff6b58b153590df2060315c4c0124c60a53801` | 1667 | `api/dist/openapi.yaml` |
| `46950bd52d1bce8c7a3d232e49528dc29442cd43c5db5c7282988596e4d28488` | 45 | `server/internal/modules/workspace/adapter/http/gen/bodyshape.gen.go` |
| `c5b7bb150b7ed6a926aa19ac0eb8ae09cf5d80fbc6fbb39adf1e0f7cabe118fa` | 2373 | `server/internal/modules/workspace/adapter/http/gen/server.gen.go` |
| `d636544687557f7ed0503a2b7c073a651784f272e394b8f438570b4d37121a79` | 1791 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/workspace/adapter/http/gen/bodyshape.gen.go server/internal/modules/workspace/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 3: 新码和文案**

`server/internal/modules/workspace/domain/errors.go`（修改，1 处）：

````old server/internal/modules/workspace/domain/errors.go
		"The invitation does not exist, or its link is not valid.")
````

````new server/internal/modules/workspace/domain/errors.go
		"The invitation does not exist, or its link is not valid.")
	// ErrInvitationEmailMismatch answers a response to an invitation by an
	// account whose address, read under its lock, is not the invitation's;
	// it does not say which address the invitation is for (M3 design 3.8).
	ErrInvitationEmailMismatch = shared.NewError(shared.KindForbidden, "workspace.invitation_email_mismatch",
		"The invitation was sent to another e-mail address.")
````

`web/apps/web/helpers/authentication.helper.ts`（修改，1 处）：

````old web/apps/web/helpers/authentication.helper.ts
  "workspace.invitation_responded": "auth.errors.workspace_invitation_responded",
````

````new web/apps/web/helpers/authentication.helper.ts
  "workspace.invitation_responded": "auth.errors.workspace_invitation_responded",
  "workspace.invitation_email_mismatch": "auth.errors.workspace_invitation_email_mismatch",
````

`web/packages/i18n/src/locales/en/auth.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/auth.json
      "workspace_invitation_responded": "The invitation has been answered already.",
````

````new web/packages/i18n/src/locales/en/auth.json
      "workspace_invitation_responded": "The invitation has been answered already.",
      "workspace_invitation_email_mismatch": "This invitation was sent to another email address.",
````

`web/packages/i18n/src/locales/zh-CN/auth.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/auth.json
      "workspace_invitation_responded": "这份邀请已经回应过了。",
````

````new web/packages/i18n/src/locales/zh-CN/auth.json
      "workspace_invitation_responded": "这份邀请已经回应过了。",
      "workspace_invitation_email_mismatch": "这份邀请发给了另一个邮箱。",
````

- [ ] **Step 4: 端口、回应、用例**

`server/internal/modules/workspace/app/invitation_ports.go`（修改，1 处）：

````old server/internal/modules/workspace/app/invitation_ports.go
}

// DuplicateInvitation is a store's answer to an insert that the unique key
````

````new server/internal/modules/workspace/app/invitation_ports.go
}

// InvitationAccepter accepts an invitation under its workspace's FOR NO KEY
// UPDATE, the lock of a write of a membership.
type InvitationAccepter interface {
	InvitationLocker
	// LockWorkspace locks the undeleted workspace id FOR NO KEY UPDATE until
	// the transaction ends; ErrNotFound when there is none, also when it was
	// deleted while the lock waited.
	LockWorkspace(ctx context.Context, id uuid.UUID) error
	// MemberOf returns userID's undeleted membership of the workspace,
	// active or ended; found is false when he has none.
	MemberOf(ctx context.Context, workspaceID, userID uuid.UUID) (m domain.Membership, found bool, err error)
	// RestoreMember makes the ended membership id active again with role,
	// by the account by at now.
	RestoreMember(ctx context.Context, id uuid.UUID, role shared.Role, by uuid.UUID, now time.Time) error
	// CreateMember inserts m.
	CreateMember(ctx context.Context, m MemberRow) error
	// AcceptInvitation records the invitation as accepted by the account by
	// at now, and deletes it.
	AcceptInvitation(ctx context.Context, id, by uuid.UUID, now time.Time) error
	// WorkspaceByID returns the undeleted workspace id and its number of
	// active members, without a role.
	WorkspaceByID(ctx context.Context, id uuid.UUID) (domain.Workspace, error)
}

// InvitationDecliner declines an invitation under its workspace's FOR
// SHARE.
type InvitationDecliner interface {
	InvitationLocker
	// ShareWorkspace is InvitationUpdater's.
	ShareWorkspace(ctx context.Context, id uuid.UUID) error
	// DeclineInvitation records the invitation as declined by the account
	// by at now; it stays undeleted.
	DeclineInvitation(ctx context.Context, id, by uuid.UUID, now time.Time) error
}

// DuplicateInvitation is a store's answer to an insert that the unique key
````

`server/internal/modules/workspace/app/respond_invitation.go`（新文件，66 行）：

````file server/internal/modules/workspace/app/respond_invitation.go
package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// responder is what accepting and declining an invitation share (M3 design
// 3.8): the caller answers for himself, account level, without the
// Authorizer (6.4). The link's token is checked before anything is read, so
// the answer to a wrong one cannot depend on the invitation's existence.
// Then, in the transaction, in the order of 3.6: the caller's account row
// FOR SHARE, the first lock (conventions 1 and 6), its state and address
// read under that lock; the invitation read for its workspace; the
// workspace, by the lock the response passes; the invitation FOR UPDATE,
// read again. Only then are the addresses compared, the one read under the
// account's lock with the invitation's, then whether the invitation was
// answered: an address that is not the invitation's writes nothing.
type responder struct {
	accounts    Accounts
	invitations InvitationLocker
	tokens      invitationTokens
}

// checkToken is workspace.invitation_not_found unless token is the
// invitation id's.
func (r responder) checkToken(id uuid.UUID, token string) error {
	if !r.tokens.valid(id, token) {
		return domain.ErrInvitationNotFound
	}
	return nil
}

// lock takes the locks in the transaction ctx carries and returns the
// caller's account and the invitation as read under them: 401 unauthorized
// for an account deactivated or gone, workspace.invitation_not_found for an
// invitation not there or deleted meanwhile, or whose workspace is,
// workspace.invitation_email_mismatch for another address,
// workspace.invitation_responded for a declined invitation.
func (r responder) lock(ctx context.Context, actor shared.Actor, id uuid.UUID,
	lockWorkspace func(ctx context.Context, id uuid.UUID) error) (AccountState, domain.Invitation, error) {
	account, found, err := r.accounts.ShareAccount(ctx, actor.UserID)
	switch {
	case err != nil:
		return AccountState{}, domain.Invitation{}, err
	case !found || !account.Active:
		return AccountState{}, domain.Invitation{}, shared.Unauthenticated()
	}
	inv, err := readInvitation(ctx, r.invitations, id)
	if err != nil {
		return AccountState{}, domain.Invitation{}, err
	}
	if inv, err = lockInvitation(ctx, r.invitations, lockWorkspace, inv); err != nil {
		return AccountState{}, domain.Invitation{}, err
	}
	switch {
	case account.Email != inv.Email:
		return AccountState{}, domain.Invitation{}, domain.ErrInvitationEmailMismatch
	case inv.Responded():
		return AccountState{}, domain.Invitation{}, domain.ErrInvitationResponded
	}
	return account, inv, nil
}
````

`server/internal/modules/workspace/app/accept_invitation.go`（新文件，89 行）：

````file server/internal/modules/workspace/app/accept_invitation.go
package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// AcceptInvitationDeps are AcceptWorkspaceInvitation's collaborators.
type AcceptInvitationDeps struct {
	Accounts    Accounts
	Invitations InvitationAccepter
	Tx          shared.TxManager
	Clock       Clock
	MAC         InvitationMAC
}

// AcceptWorkspaceInvitation makes the caller a member by an invitation to
// his address: POST /api/v0/workspace-invitations/{invitation_id}/accept.
type AcceptWorkspaceInvitation struct {
	invitations InvitationAccepter
	tx          shared.TxManager
	clock       Clock
	responder   responder
}

// NewAcceptWorkspaceInvitation returns the use case.
func NewAcceptWorkspaceInvitation(d AcceptInvitationDeps) *AcceptWorkspaceInvitation {
	return &AcceptWorkspaceInvitation{invitations: d.Invitations, tx: d.Tx, clock: d.Clock,
		responder: responder{accounts: d.Accounts, invitations: d.Invitations, tokens: invitationTokens{mac: d.MAC}}}
}

// Execute takes responder's locks, the workspace FOR NO KEY UPDATE, the lock
// of a write of a membership (M3 design 3.6), then reads the clock and
// writes (M3 design 3.8). An invitation never changes an active
// membership: an active member's invitation is consumed, his role kept. A
// former member's row is restored with the invitation's role; P4 adds, when
// that role is a guest's, DemoteToGuest of his project memberships here, in
// the same transaction. Anyone else is inserted with it. Then the
// invitation is accepted, and deleted. The answer is the workspace with the
// caller's role as it now is.
func (u *AcceptWorkspaceInvitation) Execute(ctx context.Context, id uuid.UUID, token string) (domain.Workspace, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Workspace{}, err
	}
	if err := u.responder.checkToken(id, token); err != nil {
		return domain.Workspace{}, err
	}
	var joined domain.Workspace
	err = u.tx.WithinTx(ctx, func(ctx context.Context) error {
		account, inv, err := u.responder.lock(ctx, actor, id, u.invitations.LockWorkspace)
		if err != nil {
			return err
		}
		now := u.clock.Now()
		m, member, err := u.invitations.MemberOf(ctx, inv.WorkspaceID, account.ID)
		if err != nil {
			return err
		}
		role := inv.Role
		switch {
		case member && m.IsActive:
			role = m.Role
		case member:
			err = u.invitations.RestoreMember(ctx, m.ID, inv.Role, account.ID, now)
		default:
			err = u.invitations.CreateMember(ctx, MemberRow{ID: uuid.NewV7(), WorkspaceID: inv.WorkspaceID, MemberID: account.ID, Role: inv.Role,
				CreatedBy: account.ID, Now: now})
		}
		if err != nil {
			return err
		}
		if err := u.invitations.AcceptInvitation(ctx, inv.ID, account.ID, now); err != nil {
			return err
		}
		if joined, err = u.invitations.WorkspaceByID(ctx, inv.WorkspaceID); err != nil {
			return err
		}
		joined.Role = role
		return nil
	})
	if err != nil {
		return domain.Workspace{}, err
	}
	return joined, nil
}
````

`server/internal/modules/workspace/app/decline_invitation.go`（新文件，47 行）：

````file server/internal/modules/workspace/app/decline_invitation.go
package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// DeclineWorkspaceInvitation answers an invitation to the caller's address
// with no: POST /api/v0/workspace-invitations/{invitation_id}/decline.
type DeclineWorkspaceInvitation struct {
	invitations InvitationDecliner
	tx          shared.TxManager
	clock       Clock
	responder   responder
}

// NewDeclineWorkspaceInvitation returns the use case.
func NewDeclineWorkspaceInvitation(accounts Accounts, invitations InvitationDecliner, tx shared.TxManager, clock Clock,
	mac InvitationMAC) *DeclineWorkspaceInvitation {
	return &DeclineWorkspaceInvitation{invitations: invitations, tx: tx, clock: clock,
		responder: responder{accounts: accounts, invitations: invitations, tokens: invitationTokens{mac: mac}}}
}

// Execute takes responder's locks, the workspace FOR SHARE: declining adds
// no membership, and the account's lock comes first all the same, so the
// two answers check the address alike and neither locks the account after
// a workspace (M3 design 3.8, 3.6 convention 1). Then it reads the clock
// and records the answer. The invitation stays, declined, and holds its
// address until an admin deletes it.
func (u *DeclineWorkspaceInvitation) Execute(ctx context.Context, id uuid.UUID, token string) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	if err := u.responder.checkToken(id, token); err != nil {
		return err
	}
	return u.tx.WithinTx(ctx, func(ctx context.Context) error {
		account, inv, err := u.responder.lock(ctx, actor, id, u.invitations.ShareWorkspace)
		if err != nil {
			return err
		}
		return u.invitations.DeclineInvitation(ctx, inv.ID, account.ID, u.clock.Now())
	})
}
````

`server/internal/modules/workspace/app/fakes_invitations_test.go`（修改，1 处）：

````old server/internal/modules/workspace/app/fakes_invitations_test.go
	writeErr    error  // for UpdateInvitationRole and DeleteInvitation
````

````new server/internal/modules/workspace/app/fakes_invitations_test.go
	writeErr    error  // for UpdateInvitationRole and DeleteInvitation
	// failing fails the answers' steps by name: MemberOf, RestoreMember,
	// AcceptInvitation, DeclineInvitation, WorkspaceByID.
	failing map[string]error
}

// step logs a step of an answer to an invitation and fails it with the
// error set for it, wrapped as the store wraps it.
func (f *fakeInvitations) step(ctx context.Context, name, format string, args ...any) error {
	f.log.add(ctx, name+" "+format, args...)
	if err := f.failing[name]; err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func (f *fakeInvitations) MemberOf(ctx context.Context, workspaceID, userID uuid.UUID) (domain.Membership, bool, error) {
	if err := f.step(ctx, "MemberOf", "%s %s", workspaceID, userID); err != nil {
		return domain.Membership{}, false, err
	}
	i := slices.IndexFunc(f.memberships[workspaceID], func(m domain.Membership) bool { return m.MemberID == userID })
	if i < 0 {
		return domain.Membership{}, false, nil
	}
	return f.memberships[workspaceID][i], true, nil
}

func (f *fakeInvitations) RestoreMember(ctx context.Context, id uuid.UUID, role shared.Role, by uuid.UUID, now time.Time) error {
	return f.step(ctx, "RestoreMember", "%s as %d by %s at %s", id, role, by, now.Format(time.RFC3339Nano))
}

// AcceptInvitation drops the invitation it holds: an accepted one is
// deleted.
func (f *fakeInvitations) AcceptInvitation(ctx context.Context, id, by uuid.UUID, now time.Time) error {
	if err := f.step(ctx, "AcceptInvitation", "%s by %s at %s", id, by, now.Format(time.RFC3339Nano)); err != nil {
		return err
	}
	f.invitations = slices.DeleteFunc(f.invitations, func(inv domain.Invitation) bool { return inv.ID == id })
	return nil
}

func (f *fakeInvitations) DeclineInvitation(ctx context.Context, id, by uuid.UUID, now time.Time) error {
	return f.step(ctx, "DeclineInvitation", "%s by %s at %s", id, by, now.Format(time.RFC3339Nano))
}

// WorkspaceByID answers the workspace it holds, as stored.
func (f *fakeInvitations) WorkspaceByID(ctx context.Context, id uuid.UUID) (domain.Workspace, error) {
	if err := f.step(ctx, "WorkspaceByID", "%s", id); err != nil {
		return domain.Workspace{}, err
	}
	i := slices.IndexFunc(f.workspaces, func(w domain.Workspace) bool { return w.ID == id })
	if i < 0 {
		return domain.Workspace{}, fmt.Errorf("read workspace %s: no such row", id)
	}
	return f.workspaces[i], nil
````

`server/internal/modules/workspace/app/accept_invitation_test.go`（新文件，222 行）：

````file server/internal/modules/workspace/app/accept_invitation_test.go
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

var (
	// The invitees: dave (declined in acme, daveToAcme), erin, whose
	// membership of acme, an admin's, has ended, and frank, never a member.
	dave       = app.AccountState{ID: uuid.NewV7(), Email: "dave@corp.com", Active: true}
	erin       = app.AccountState{ID: uuid.NewV7(), Email: "erin@corp.com", Active: true}
	frank      = app.AccountState{ID: uuid.NewV7(), Email: "frank@corp.com", Active: true}
	erinInAcme = domain.Membership{ID: uuid.NewV7(), WorkspaceID: acme.ID, MemberID: erin.ID, Role: shared.RoleAdmin, CreatedAt: now}
	// frankToAcme is the invitation the refusals answer, unless a case
	// names another.
	frankToAcme = invitationTo(frank, acme, shared.RoleMember)
)

// invitationTo is a pending invitation of user's address to w as role.
func invitationTo(user app.AccountState, w domain.Workspace, role shared.Role) domain.Invitation {
	return domain.Invitation{ID: uuid.NewV7(), WorkspaceID: w.ID, Email: user.Email, Role: role, CreatedAt: now, CreatedByID: &alice.ID}
}

// responding is invitationsFixture holding inv too, erin's ended membership
// of acme, and the accounts: alice, bob, carol (deactivated), dave, erin
// and frank.
func responding(inv ...domain.Invitation) *invitationsFixture {
	f := newInvitations()
	f.invitations.invitations = append(f.invitations.invitations, inv...)
	f.invitations.memberships[acme.ID] = append(f.invitations.memberships[acme.ID], erinInAcme)
	f.accounts = &fakeAccounts{log: f.log, accounts: []app.AccountState{alice, bob, carol, dave, erin, frank}}
	return f
}

func (f *invitationsFixture) accept() *app.AcceptWorkspaceInvitation {
	return app.NewAcceptWorkspaceInvitation(app.AcceptInvitationDeps{Accounts: f.accounts, Invitations: f.invitations, Tx: f.tx,
		Clock: clockAt{at: clockNow}, MAC: f.mac})
}

// respondedCalls are the calls up to the answer to inv by user: the token,
// the account's lock, the invitation's read, the workspace's lock by
// lockName, the invitation's lock.
func respondedCalls(user app.AccountState, inv domain.Invitation, lockName string) []string {
	return []string{"Verify " + inv.ID.String(), "ShareAccount " + user.ID.String(), "InvitationByID " + inv.ID.String(),
		lockName + " " + inv.WorkspaceID.String(), "LockInvitation " + inv.ID.String()}
}

// Accepting never changes an active membership (M3 design 3.8, 9.1): bob,
// acme's member and beta's guest, keeps his role whatever the invitation's,
// higher, lower or the same, and only the invitation is consumed; erin's
// ended membership is restored with the invitation's role; frank is
// inserted with it. Each in one transaction, after the account's, the
// workspace's and the invitation's locks; the answer is the workspace with
// the caller's role as it now is, and the invitation is gone.
func TestAcceptWorkspaceInvitation(t *testing.T) {
	at := clockNow.Format(time.RFC3339Nano)
	tests := []struct {
		name  string
		user  app.AccountState
		inv   domain.Invitation
		w     domain.Workspace
		role  shared.Role // the answer's
		write string      // the membership's, "" for none
	}{
		{"an active member, invited higher", bob, invitationTo(bob, acme, shared.RoleAdmin), acme, shared.RoleMember, ""},
		{"an active member, invited lower", bob, invitationTo(bob, acme, shared.RoleGuest), acme, shared.RoleMember, ""},
		{"an active member, invited the same", bob, invitationTo(bob, acme, shared.RoleMember), acme, shared.RoleMember, ""},
		{"an active guest of another workspace, invited higher", bob, invitationTo(bob, beta, shared.RoleAdmin), beta, shared.RoleGuest, ""},
		{"a former admin, invited as a guest", erin, invitationTo(erin, acme, shared.RoleGuest), acme, shared.RoleGuest,
			fmt.Sprintf("RestoreMember %s as %d by %s at %s", erinInAcme.ID, shared.RoleGuest, erin.ID, at)},
		{"a former member elsewhere, new here", erin, invitationTo(erin, beta, shared.RoleMember), beta, shared.RoleMember,
			fmt.Sprintf("CreateMember %s in %s as %d by %s at %s", erin.ID, beta.ID, shared.RoleMember, erin.ID, at)},
		{"never a member", frank, invitationTo(frank, acme, shared.RoleAdmin), acme, shared.RoleAdmin,
			fmt.Sprintf("CreateMember %s in %s as %d by %s at %s", frank.ID, acme.ID, shared.RoleAdmin, frank.ID, at)},
	}
	for _, tt := range tests {
		f := responding(tt.inv)
		got, err := f.accept().Execute(as(tt.user), tt.inv.ID, tokenOf(f.mac, tt.inv.ID))
		want := tt.w
		want.Role = tt.role
		if err != nil || got != want {
			t.Errorf("%s: Execute() = %+v, %v; want %+v", tt.name, got, err, want)
		}
		wantCalls := append(respondedCalls(tt.user, tt.inv, "LockWorkspace"), "MemberOf "+tt.w.ID.String()+" "+tt.user.ID.String())
		if tt.write != "" {
			wantCalls = append(wantCalls, tt.write)
		}
		wantCalls = append(wantCalls, fmt.Sprintf("AcceptInvitation %s by %s at %s", tt.inv.ID, tt.user.ID, at), "WorkspaceByID "+tt.w.ID.String())
		if !slices.Equal(f.log.calls, wantCalls) || f.tx.calls != 1 {
			t.Errorf("%s: calls = %q in %d transactions, want %q in one", tt.name, f.log.calls, f.tx.calls, wantCalls)
		}
		if slices.ContainsFunc(f.invitations.invitations, func(inv domain.Invitation) bool { return inv.ID == tt.inv.ID }) {
			t.Errorf("%s: the invitation is still there", tt.name)
		}
	}
}

// responseRefusals are the refusals accepting and declining share, each
// with the calls it makes (M3 design 3.8, 8.2): a wrong token reads nothing
// and opens no transaction; an account deactivated or gone, read under its
// lock, is 401 before the invitation is read; an invitation not there,
// deleted meanwhile or of a workspace deleted meanwhile is
// workspace.invitation_not_found; another address is
// workspace.invitation_email_mismatch, also for a declined invitation; a
// declined one, also declined meanwhile, workspace.invitation_responded.
// None writes. lockName is the workspace's lock.
func responseRefusals(lockName string) []responseRefusal {
	stranger := app.AccountState{ID: uuid.NewV7(), Email: "stranger@corp.com", Active: true}
	decided := respondedCalls(frank, frankToAcme, lockName)
	nobodys := uuid.NewV7()
	return []responseRefusal{
		{"another invitation's token", frank, frankToAcme.ID, tokenOf(fakeMAC{key: "the instance's key"}, carolToAcme.ID), nil,
			domain.ErrInvitationNotFound, decided[:1]},
		{"a deactivated account", carol, carolToAcme.ID, "", nil, shared.Unauthenticated(), respondedCalls(carol, carolToAcme, lockName)[:2]},
		{"an account gone", stranger, frankToAcme.ID, "", nil, shared.Unauthenticated(), respondedCalls(stranger, frankToAcme, lockName)[:2]},
		{"no such invitation", frank, nobodys, "", nil, domain.ErrInvitationNotFound,
			[]string{"Verify " + nobodys.String(), "ShareAccount " + frank.ID.String(), "InvitationByID " + nobodys.String()}},
		{"acme deleted while the lock waited", frank, frankToAcme.ID, "",
			func(f *invitationsFixture) { f.invitations.lockErrs = map[string]error{"acme": app.ErrNotFound} }, domain.ErrInvitationNotFound, decided[:4]},
		{"deleted while the lock waited", frank, frankToAcme.ID, "",
			func(f *invitationsFixture) { f.invitations.onLock = func() { f.invitations.invitations = nil } }, domain.ErrInvitationNotFound, decided},
		{"another address", alice, frankToAcme.ID, "", nil, domain.ErrInvitationEmailMismatch, respondedCalls(alice, frankToAcme, lockName)},
		{"another address, declined", alice, daveToAcme.ID, "", nil, domain.ErrInvitationEmailMismatch, respondedCalls(alice, daveToAcme, lockName)},
		{"declined", dave, daveToAcme.ID, "", nil, domain.ErrInvitationResponded, respondedCalls(dave, daveToAcme, lockName)},
		{"declined while the lock waited", frank, frankToAcme.ID, "",
			func(f *invitationsFixture) {
				f.invitations.onLock = func() { f.invitations.invitations[len(f.invitations.invitations)-1].RespondedAt = &declined }
			},
			domain.ErrInvitationResponded, decided},
	}
}

type responseRefusal struct {
	name  string
	user  app.AccountState
	id    uuid.UUID
	token string // "" for the invitation's
	set   func(f *invitationsFixture)
	want  error
	calls []string
}

// responseFailures are the failures accepting and declining share before
// they write, each the error injected, never a problem of the contract.
func responseFailures(failure error, lockName string) []responseRefusal {
	decided := respondedCalls(frank, frankToAcme, lockName)
	return []responseRefusal{
		{"the account's lock failed", frank, frankToAcme.ID, "", func(f *invitationsFixture) { f.accounts.err = failure }, failure, decided[:2]},
		{"the read failed", frank, frankToAcme.ID, "", func(f *invitationsFixture) { f.invitations.readErr = failure }, failure, decided[:3]},
		{"the workspace's lock failed", frank, frankToAcme.ID, "",
			func(f *invitationsFixture) { f.invitations.lockErrs = map[string]error{"acme": failure} }, failure, decided[:4]},
		{"the invitation's lock failed", frank, frankToAcme.ID, "",
			func(f *invitationsFixture) { f.invitations.onLock = func() { f.invitations.readErr = failure } }, failure, decided},
	}
}

// Each refusal and failure is the answer, and nothing is written: see
// responseRefusals and responseFailures; a failure of a write, or of the
// answer's read, is itself too. Without a caller it is 401 and nothing is
// read.
func TestAcceptWorkspaceInvitationRefusals(t *testing.T) {
	failure := errors.New("connection reset")
	cases := append(responseRefusals("LockWorkspace"), responseFailures(failure, "LockWorkspace")...)
	decided := append(respondedCalls(frank, frankToAcme, "LockWorkspace"), "MemberOf "+acme.ID.String()+" "+frank.ID.String())
	at := clockNow.Format(time.RFC3339Nano)
	created := fmt.Sprintf("CreateMember %s in %s as %d by %s at %s", frank.ID, acme.ID, shared.RoleMember, frank.ID, at)
	accepted := fmt.Sprintf("AcceptInvitation %s by %s at %s", frankToAcme.ID, frank.ID, at)
	for _, step := range []struct {
		name  string
		set   func(f *invitationsFixture)
		calls []string
	}{
		{"MemberOf", func(f *invitationsFixture) { f.invitations.failing = map[string]error{"MemberOf": failure} }, decided},
		{"CreateMember", func(f *invitationsFixture) { f.invitations.memberErr = failure }, append(slices.Clone(decided), created)},
		{"AcceptInvitation", func(f *invitationsFixture) { f.invitations.failing = map[string]error{"AcceptInvitation": failure} },
			append(slices.Clone(decided), created, accepted)},
		{"WorkspaceByID", func(f *invitationsFixture) { f.invitations.failing = map[string]error{"WorkspaceByID": failure} },
			append(slices.Clone(decided), created, accepted, "WorkspaceByID "+acme.ID.String())},
	} {
		cases = append(cases, responseRefusal{step.name + " failed", frank, frankToAcme.ID, "", step.set, failure, step.calls})
	}
	for _, tt := range cases {
		f := responding(frankToAcme)
		if tt.set != nil {
			tt.set(f)
		}
		token := tt.token
		if token == "" {
			token = tokenOf(f.mac, tt.id)
		}
		got, err := f.accept().Execute(as(tt.user), tt.id, token)
		if !errors.Is(err, tt.want) || got != (domain.Workspace{}) {
			t.Errorf("%s: Execute() = %+v, %v; want no workspace and %v", tt.name, got, err, tt.want)
		}
		var se *shared.Error
		if tt.want == failure && errors.As(err, &se) {
			t.Errorf("%s: Execute() = %v, which is also a problem of the contract", tt.name, err)
		}
		wantTx := 1
		if len(tt.calls) == 1 {
			wantTx = 0
		}
		if !slices.Equal(f.log.calls, tt.calls) || f.tx.calls != wantTx {
			t.Errorf("%s: calls = %q in %d transactions, want %q in %d", tt.name, f.log.calls, f.tx.calls, tt.calls, wantTx)
		}
	}
	f := responding(frankToAcme)
	if _, err := f.accept().Execute(context.Background(), frankToAcme.ID, tokenOf(f.mac, frankToAcme.ID)); !errors.Is(err, shared.Unauthenticated()) ||
		len(f.log.calls) != 0 {
		t.Errorf("Execute() without an actor = %v, calls %q; want 401 unauthorized and nothing", err, f.log.calls)
	}
}
````

`server/internal/modules/workspace/app/decline_invitation_test.go`（新文件，76 行）：

````file server/internal/modules/workspace/app/decline_invitation_test.go
package app_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func (f *invitationsFixture) decline() *app.DeclineWorkspaceInvitation {
	return app.NewDeclineWorkspaceInvitation(f.accounts, f.invitations, f.tx, clockAt{at: clockNow}, f.mac)
}

// Declining records the answer after the account's lock, the workspace's
// FOR SHARE and the invitation's lock, in one transaction (M3 design 3.8,
// 3.6): an active member's invitation as anyone else's; no membership is
// read or written, and the invitation stays.
func TestDeclineWorkspaceInvitation(t *testing.T) {
	for _, tt := range []struct {
		user app.AccountState
		inv  domain.Invitation
	}{{frank, frankToAcme}, {bob, invitationTo(bob, beta, shared.RoleAdmin)}} {
		f := responding(frankToAcme, tt.inv)
		err := f.decline().Execute(as(tt.user), tt.inv.ID, tokenOf(f.mac, tt.inv.ID))
		wantCalls := append(respondedCalls(tt.user, tt.inv, "ShareWorkspace"),
			fmt.Sprintf("DeclineInvitation %s by %s at %s", tt.inv.ID, tt.user.ID, clockNow.Format(time.RFC3339Nano)))
		if err != nil || !slices.Equal(f.log.calls, wantCalls) || f.tx.calls != 1 {
			t.Errorf("%s: Execute() = %v, calls = %q in %d transactions; want %q in one", tt.inv.Email, err, f.log.calls, f.tx.calls, wantCalls)
		}
	}
}

// Declining refuses as accepting does (responseRefusals, responseFailures),
// under the workspace's FOR SHARE; a failed write is itself. Without a
// caller it is 401 and nothing is read.
func TestDeclineWorkspaceInvitationRefusals(t *testing.T) {
	failure := errors.New("connection reset")
	cases := append(responseRefusals("ShareWorkspace"), responseFailures(failure, "ShareWorkspace")...)
	cases = append(cases, responseRefusal{"the write failed", frank, frankToAcme.ID, "",
		func(f *invitationsFixture) { f.invitations.failing = map[string]error{"DeclineInvitation": failure} }, failure,
		append(respondedCalls(frank, frankToAcme, "ShareWorkspace"),
			fmt.Sprintf("DeclineInvitation %s by %s at %s", frankToAcme.ID, frank.ID, clockNow.Format(time.RFC3339Nano)))})
	for _, tt := range cases {
		f := responding(frankToAcme)
		if tt.set != nil {
			tt.set(f)
		}
		token := tt.token
		if token == "" {
			token = tokenOf(f.mac, tt.id)
		}
		err := f.decline().Execute(as(tt.user), tt.id, token)
		var se *shared.Error
		if !errors.Is(err, tt.want) || (tt.want == failure && errors.As(err, &se)) {
			t.Errorf("%s: Execute() = %v, want %v", tt.name, err, tt.want)
		}
		wantTx := 1
		if len(tt.calls) == 1 {
			wantTx = 0
		}
		if !slices.Equal(f.log.calls, tt.calls) || f.tx.calls != wantTx {
			t.Errorf("%s: calls = %q in %d transactions, want %q in %d", tt.name, f.log.calls, f.tx.calls, tt.calls, wantTx)
		}
	}
	f := responding(frankToAcme)
	if err := f.decline().Execute(context.Background(), frankToAcme.ID, tokenOf(f.mac, frankToAcme.ID)); !errors.Is(err, shared.Unauthenticated()) ||
		len(f.log.calls) != 0 {
		t.Errorf("Execute() without an actor = %v, calls %q; want 401 unauthorized and nothing", err, f.log.calls)
	}
}
````

`server/internal/modules/workspace/app/clock_test.go`（修改，1 处）：

````old server/internal/modules/workspace/app/clock_test.go
			fmt.Sprintf("DeleteInvitation %s by %s at %s", daveToAcme.ID, alice.ID, at))},
````

````new server/internal/modules/workspace/app/clock_test.go
			fmt.Sprintf("DeleteInvitation %s by %s at %s", daveToAcme.ID, alice.ID, at))},
		{"acceptWorkspaceInvitation", func() ([]string, error) {
			f := responding(frankToAcme)
			_, err := app.NewAcceptWorkspaceInvitation(app.AcceptInvitationDeps{Accounts: f.accounts, Invitations: f.invitations, Tx: f.tx,
				Clock: clockAt{clockNow, f.log}, MAC: f.mac}).Execute(as(frank), frankToAcme.ID, tokenOf(f.mac, frankToAcme.ID))
			return f.log.calls, err
		}, append(respondedCalls(frank, frankToAcme, "LockWorkspace"), "Now", "MemberOf "+acme.ID.String()+" "+frank.ID.String(),
			fmt.Sprintf("CreateMember %s in %s as %d by %s at %s", frank.ID, acme.ID, shared.RoleMember, frank.ID, at),
			fmt.Sprintf("AcceptInvitation %s by %s at %s", frankToAcme.ID, frank.ID, at), "WorkspaceByID "+acme.ID.String())},
		{"declineWorkspaceInvitation", func() ([]string, error) {
			f := responding(frankToAcme)
			err := app.NewDeclineWorkspaceInvitation(f.accounts, f.invitations, f.tx, clockAt{clockNow, f.log}, f.mac).
				Execute(as(frank), frankToAcme.ID, tokenOf(f.mac, frankToAcme.ID))
			return f.log.calls, err
		}, append(respondedCalls(frank, frankToAcme, "ShareWorkspace"), "Now",
			fmt.Sprintf("DeclineInvitation %s by %s at %s", frankToAcme.ID, frank.ID, at))},
````

`server/internal/modules/workspace/app/list_invitations_test.go`（修改，1 处）：

````old server/internal/modules/workspace/app/list_invitations_test.go
	caller      *fakeCallerLock
````

````new server/internal/modules/workspace/app/list_invitations_test.go
	caller      *fakeCallerLock
	accounts    *fakeAccounts // responding's
````

- [ ] **Step 5: HTTP、接线**

`server/internal/modules/workspace/adapter/http/handler.go`（修改，2 处）：

````old server/internal/modules/workspace/adapter/http/handler.go
}

// CheckSlugUseCase is app.CheckSlug.
````

````new server/internal/modules/workspace/adapter/http/handler.go
}

// AcceptInvitationUseCase is app.AcceptWorkspaceInvitation.
type AcceptInvitationUseCase interface {
	Execute(ctx context.Context, id uuid.UUID, token string) (domain.Workspace, error)
}

// DeclineInvitationUseCase is app.DeclineWorkspaceInvitation.
type DeclineInvitationUseCase interface {
	Execute(ctx context.Context, id uuid.UUID, token string) error
}

// CheckSlugUseCase is app.CheckSlug.
````

````old server/internal/modules/workspace/adapter/http/handler.go
	DeleteInvitation  DeleteInvitationUseCase
````

````new server/internal/modules/workspace/adapter/http/handler.go
	DeleteInvitation  DeleteInvitationUseCase
	AcceptInvitation  AcceptInvitationUseCase
	DeclineInvitation DeclineInvitationUseCase
````

`server/internal/modules/workspace/adapter/http/invitations.go`（修改，1 处）：

````old server/internal/modules/workspace/adapter/http/invitations.go
}

// invitations is list as the API shows it.
````

````new server/internal/modules/workspace/adapter/http/invitations.go
}

// AcceptWorkspaceInvitation serves POST
// /api/v0/workspace-invitations/{invitation_id}/accept.
func (h handler) AcceptWorkspaceInvitation(ctx context.Context, req gen.AcceptWorkspaceInvitationRequestObject) (gen.AcceptWorkspaceInvitationResponseObject, error) {
	w, err := h.uc.AcceptInvitation.Execute(ctx, req.InvitationID, req.Body.Token)
	if err != nil {
		return nil, err
	}
	return gen.AcceptWorkspaceInvitation200JSONResponse(workspace(w)), nil
}

// DeclineWorkspaceInvitation serves POST
// /api/v0/workspace-invitations/{invitation_id}/decline.
func (h handler) DeclineWorkspaceInvitation(ctx context.Context, req gen.DeclineWorkspaceInvitationRequestObject) (gen.DeclineWorkspaceInvitationResponseObject, error) {
	if err := h.uc.DeclineInvitation.Execute(ctx, req.InvitationID, req.Body.Token); err != nil {
		return nil, err
	}
	return gen.DeclineWorkspaceInvitation204Response{}, nil
}

// invitations is list as the API shows it.
````

`server/internal/modules/workspace/adapter/http/handler_test.go`（修改，1 处）：

````old server/internal/modules/workspace/adapter/http/handler_test.go
		DeleteInvitation: fakeDeleteInvitation{f.invitations},
````

````new server/internal/modules/workspace/adapter/http/handler_test.go
		DeleteInvitation: fakeDeleteInvitation{f.invitations}, AcceptInvitation: fakeAcceptInvitation{f.invitations},
		DeclineInvitation: fakeDeclineInvitation{f.invitations},
````

`server/internal/modules/workspace/adapter/http/invitations_test.go`（修改，2 处）：

````old server/internal/modules/workspace/adapter/http/invitations_test.go
	preview domain.InvitationPreview                // the answer of a get
	err     error
````

````new server/internal/modules/workspace/adapter/http/invitations_test.go
	preview domain.InvitationPreview                // the answer of a get
	joined  domain.Workspace                        // the answer of an acceptance
	err     error
}

type fakeAcceptInvitation struct{ *fakeInvitations }

func (f fakeAcceptInvitation) Execute(ctx context.Context, id uuid.UUID, token string) (domain.Workspace, error) {
	f.calls = append(f.calls, fmt.Sprintf("accept %s %s %s", caller(ctx), id, token))
	return f.joined, f.err
}

type fakeDeclineInvitation struct{ *fakeInvitations }

func (f fakeDeclineInvitation) Execute(ctx context.Context, id uuid.UUID, token string) error {
	f.calls = append(f.calls, fmt.Sprintf("decline %s %s %s", caller(ctx), id, token))
	return f.err
````

````old server/internal/modules/workspace/adapter/http/invitations_test.go
			t.Errorf("GET %s = %d %s, calls %q; want 400 on %s without the token, and no call", tt.path, res.StatusCode, body, inv.calls, tt.field)
		}
	}
}

````

````new server/internal/modules/workspace/adapter/http/invitations_test.go
			t.Errorf("GET %s = %d %s, calls %q; want 400 on %s without the token, and no call", tt.path, res.StatusCode, body, inv.calls, tt.field)
		}
	}
}

// The two answers to an invitation hand the invitation of the path and the
// token of the body to their use case, for the caller: accepting answers
// the workspace it gives, declining 204 without a body. Each refuses as
// the contract declares; a body without its token, or with a field it does
// not have, is refused before the use case, and no answer repeats the
// token.
func TestAnsweringAWorkspaceInvitation(t *testing.T) {
	path := "/api/v0/workspace-invitations/" + carolInvited.ID.String()
	body := `{"token":"` + carolInvited.Token + `"}`
	for _, tt := range []struct {
		name, path string
		status     int
		answer     string
	}{{"accept", path + "/accept", http.StatusOK, acmeJSON + "\n"}, {"decline", path + "/decline", http.StatusNoContent, ""}} {
		inv := &fakeInvitations{joined: acme}
		h := newServer(t, fakes{invitations: inv})
		if res, got := do(t, h, request(http.MethodPost, tt.path, "bob", body)); res.StatusCode != tt.status || got != tt.answer {
			t.Errorf("POST %s = %d %q, want %d %q", tt.path, res.StatusCode, got, tt.status, tt.answer)
		}
		if want := []string{tt.name + " bob " + carolInvited.ID.String() + " " + carolInvited.Token}; !slices.Equal(inv.calls, want) {
			t.Errorf("%s: calls = %q, want %q", tt.name, inv.calls, want)
		}
		for _, refusal := range []struct {
			err    error
			status int
			want   string
		}{
			{domain.ErrInvitationNotFound, http.StatusNotFound, invitationNotFoundJSON},
			{domain.ErrInvitationEmailMismatch, http.StatusForbidden, `{"status":403,"code":"workspace.invitation_email_mismatch",` +
				`"title":"Forbidden","detail":"The invitation was sent to another e-mail address."}`},
			{domain.ErrInvitationResponded, http.StatusConflict,
				`{"status":409,"code":"workspace.invitation_responded","title":"Conflict","detail":"The invitation has been answered already."}`},
		} {
			h := newServer(t, fakes{invitations: &fakeInvitations{err: refusal.err}})
			if res, got := do(t, h, request(http.MethodPost, tt.path, "bob", body)); res.StatusCode != refusal.status || got != refusal.want+"\n" {
				t.Errorf("%s refused with %v = %d %s, want %d %s", tt.name, refusal.err, res.StatusCode, got, refusal.status, refusal.want)
			}
		}
		idle := &fakeInvitations{}
		h = newServer(t, fakes{invitations: idle})
		for _, bad := range []string{`{}`, `{"token":"` + carolInvited.Token + `","email":"carol@corp.com"}`} {
			res, got := do(t, h, request(http.MethodPost, tt.path, "bob", bad))
			if res.StatusCode != http.StatusBadRequest || strings.Contains(got, carolInvited.Token) || len(idle.calls) != 0 {
				t.Errorf("%s with %s = %d %s, calls %q; want 400 without the token, and no call", tt.name, bad, res.StatusCode, got, idle.calls)
			}
		}
	}
}

````

`server/internal/modules/workspace/module.go`（修改，1 处）：

````old server/internal/modules/workspace/module.go
		DeleteInvitation: app.NewDeleteWorkspaceInvitation(store, d.Authorizer, d.Tx, d.Clock),
````

````new server/internal/modules/workspace/module.go
		DeleteInvitation: app.NewDeleteWorkspaceInvitation(store, d.Authorizer, d.Tx, d.Clock),
		AcceptInvitation: app.NewAcceptWorkspaceInvitation(app.AcceptInvitationDeps{
			Accounts: d.Accounts, Invitations: store, Tx: d.Tx, Clock: d.Clock, MAC: d.InvitationMAC,
		}),
		DeclineInvitation: app.NewDeclineWorkspaceInvitation(d.Accounts, store, d.Tx, d.Clock, d.InvitationMAC),
````

- [ ] **Step 6: 矩阵**

`server/internal/bootstrap/permission_matrix_test.go`（修改，4 处）：

````old server/internal/bootstrap/permission_matrix_test.go
		"getWorkspaceInvitation": "TestTheInvitationLinkAnswersEveryCallerAlike",
	},
````

````new server/internal/bootstrap/permission_matrix_test.go
		"getWorkspaceInvitation": "TestTheInvitationLinkAnswersEveryCallerAlike",
	},
	notTargets: map[string]string{
		"/api/v0/workspace-slugs/{slug}": "a slug asked about, not a workspace: the answer is the same for every caller",
		"/api/v0/workspace-invitations/{invitation_id}/accept": "account level: each column answers an invitation to its own " +
			"address (ownInvitation), or acme's newcomer's, whatever workspace its column targets",
		"/api/v0/workspace-invitations/{invitation_id}/decline": "account level, as accept",
	},
````

````old server/internal/bootstrap/permission_matrix_test.go
// wants one answer. An operation that needs a token cannot be listed.
````

````new server/internal/bootstrap/permission_matrix_test.go
// wants one answer. An operation that needs a token cannot be listed.
// notTargets are the paths whose parameters name nothing a column's cell
// must aim at its workspace, each with its reason: targetViolation passes
// them over, and reports any other parameter it does not know.
````

````old server/internal/bootstrap/permission_matrix_test.go
	modules []string
	public  map[string]string // operationId → the test that stands for its row
````

````new server/internal/bootstrap/permission_matrix_test.go
	modules    []string
	public     map[string]string // operationId → the test that stands for its row
	notTargets map[string]string // path → why its parameters are no column's target
````

````old server/internal/bootstrap/permission_matrix_test.go
			email := strings.ReplaceAll(string(c), " ", "-") + "@example.com"
````

````new server/internal/bootstrap/permission_matrix_test.go
			email := emailOf(c)
````

`server/internal/bootstrap/permission_matrix_seeded_test.go`（修改，3 处）：

````old server/internal/bootstrap/permission_matrix_seeded_test.go
// has. gone's are deleted with it.
````

````new server/internal/bootstrap/permission_matrix_seeded_test.go
// has; and one to each column's own address (ownInvitation), as a member.
// gone's are deleted with it.
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
	{"gone", "newcomer@example.com", shared.RoleMember},
````

````new server/internal/bootstrap/permission_matrix_seeded_test.go
	{"gone", "newcomer@example.com", shared.RoleMember},
	{"other", emailOf(callerAdmin), shared.RoleMember},
	{"other", emailOf(callerMember), shared.RoleMember},
	{"other", emailOf(callerGuest), shared.RoleMember},
	{"acme", emailOf(callerNever), shared.RoleMember},
	{"acme", emailOf(callerRemoved), shared.RoleMember},
	{"acme", emailOf(callerDeleted), shared.RoleMember},
}

// ownInvitation is the workspace of the invitation to c's own address: one
// he is not an active member of, so accepting it makes him one. acme's
// admin, member and guest are invited to other; the others to acme, where
// the removed member's ended membership is restored. An active member's
// address stays uninvited in acme, so that createWorkspaceInvitations'
// row of an active member's address is refused as that, not as an
// invited one.
func ownInvitation(c caller) string {
	switch c {
	case callerAdmin, callerMember, callerGuest:
		return "other"
	}
	return "acme"
}

// emailOf is the address of c's account.
func emailOf(c caller) string {
	return strings.ReplaceAll(string(c), " ", "-") + "@example.com"
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
	admin := callerAdmin
	if slug == "gone" {
		admin = callerDeleted
	}
````

````new server/internal/bootstrap/permission_matrix_seeded_test.go
	admin := map[string]caller{"acme": callerAdmin, "gone": callerDeleted, "other": callerNever}[slug]
````

`server/internal/bootstrap/permission_matrix_workspace_test.go`（修改，1 处）：

````old server/internal/bootstrap/permission_matrix_workspace_test.go
			request: sameRequest(http.MethodPost, "/api/v0/workspaces", `{"name":"New","slug":"new"}`), cells: every(cellCreationDisabled)},
````

````new server/internal/bootstrap/permission_matrix_workspace_test.go
			request: sameRequest(http.MethodPost, "/api/v0/workspaces", `{"name":"New","slug":"new"}`), cells: every(cellCreationDisabled)},
		// The answers to an invitation: each column's own, and acme's
		// newcomer's, to another address (M3 design 9.2).
		{op: "acceptWorkspaceInvitation", variant: "one's own", write: true, request: toOwnInvitation("accept"), cells: every(cellOK),
			check: joinsAsAMember},
		{op: "acceptWorkspaceInvitation", variant: "another's", write: true, request: toNewcomersInvitation("accept"),
			cells: every(cellEmailMismatch)},
		{op: "declineWorkspaceInvitation", variant: "one's own", write: true, request: toOwnInvitation("decline"), cells: every(cellNoContent)},
		{op: "declineWorkspaceInvitation", variant: "another's", write: true, request: toNewcomersInvitation("decline"),
			cells: every(cellEmailMismatch)},
````

`server/internal/bootstrap/permission_matrix_invitations_test.go`（修改，2 处）：

````old server/internal/bootstrap/permission_matrix_invitations_test.go
var cellInvitationNotFound = cell{http.StatusNotFound, "workspace.invitation_not_found"}
````

````new server/internal/bootstrap/permission_matrix_invitations_test.go
var (
	cellInvitationNotFound = cell{http.StatusNotFound, "workspace.invitation_not_found"}
	cellEmailMismatch      = cell{http.StatusForbidden, "workspace.invitation_email_mismatch"}
)

// toOwnInvitation is the request of a row whose callers each answer, with
// its token, the invitation to their own address (ownInvitation).
func toOwnInvitation(answer string) func(caller, seeded) (string, string, string) {
	return func(c caller, s seeded) (string, string, string) {
		id := s.invitation(ownInvitation(c), emailOf(c))
		return http.MethodPost, "/api/v0/workspace-invitations/" + id.String() + "/" + answer, `{"token":"` + invitationToken(s.t, id) + `"}`
	}
}

// toNewcomersInvitation is the request of a row whose callers each answer,
// with its token, acme's invitation of newcomer@example.com: another
// address than any caller's.
func toNewcomersInvitation(answer string) func(caller, seeded) (string, string, string) {
	return func(_ caller, s seeded) (string, string, string) {
		id := s.invitation("acme", "newcomer@example.com")
		return http.MethodPost, "/api/v0/workspace-invitations/" + id.String() + "/" + answer, `{"token":"` + invitationToken(s.t, id) + `"}`
	}
}

// joinsAsAMember: each caller's acceptance answers the workspace of his
// invitation, where he is now an active member as the invitation's
// member: one member more than it had (other: its admin and the removed
// member; acme: its admin, member and guest).
func joinsAsAMember(t *testing.T, c caller, answer string) {
	var w struct {
		Slug         string `json:"slug"`
		Role         int    `json:"role"`
		TotalMembers int    `json:"total_members"`
	}
	decodeAnswer(t, answer, &w)
	want := map[string]int{"other": 3, "acme": 4}[ownInvitation(c)]
	if w.Slug != ownInvitation(c) || w.Role != 15 || w.TotalMembers != want {
		t.Errorf("%s's acceptance answers %+v, want %s with role 15 and %d members", c, w, ownInvitation(c), want)
	}
}
````

````old server/internal/bootstrap/permission_matrix_invitations_test.go
func invitationToken(t *testing.T, id uuid.UUID) string {
````

````new server/internal/bootstrap/permission_matrix_invitations_test.go
func invitationToken(t testing.TB, id uuid.UUID) string {
````

`server/internal/bootstrap/permission_matrix_coverage_test.go`（修改，9 处）：

````old server/internal/bootstrap/permission_matrix_coverage_test.go
// operation, one that needs a token, or one that has a row; a row that
// names no operation; a row without a cell for a column; a cell
````

````new server/internal/bootstrap/permission_matrix_coverage_test.go
// operation, one that needs a token, or one that has a row; a not-target
// path that is no operation's; a row that names no operation; a row without a cell for a column; a cell
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
			default:
````

````new server/internal/bootstrap/permission_matrix_coverage_test.go
			default:
				if _, listed := exempt.notTargets[op.Path]; listed {
					break
				}
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
			found = append(found, fmt.Sprintf("operation %s is exempt as public, and has a row", id))
````

````new server/internal/bootstrap/permission_matrix_coverage_test.go
			found = append(found, fmt.Sprintf("operation %s is exempt as public, and has a row", id))
		}
	}
	for _, path := range slices.Sorted(maps.Keys(exempt.notTargets)) {
		if !slices.ContainsFunc(ops, func(op apitest.Operation) bool { return op.Path == path }) {
			found = append(found, fmt.Sprintf("the not-target path %s is no operation's", path))
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
// deleted workspaces are hidden or not.
````

````new server/internal/bootstrap/permission_matrix_coverage_test.go
// deleted workspaces are hidden or not. Any other parameter is reported: a
// path whose parameters are no column's target is listed as such, with its
// reason (matrixExemptions.notTargets), and not given here.
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
	for i, segment := range want {
		switch {
````

````new server/internal/bootstrap/permission_matrix_coverage_test.go
	for i, segment := range want {
		switch {
		case !strings.HasPrefix(segment, "{") || !strings.HasSuffix(segment, "}"):
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
		case strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "_id}"):
````

````new server/internal/bootstrap/permission_matrix_coverage_test.go
		case strings.HasSuffix(segment, "_id}"):
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
				return fmt.Sprintf("%s %s is no row seeded under its column's workspace %s", segment, got[i], workspaceOf(c))
			}
````

````new server/internal/bootstrap/permission_matrix_coverage_test.go
				return fmt.Sprintf("%s %s is no row seeded under its column's workspace %s", segment, got[i], workspaceOf(c))
			}
		default:
			return fmt.Sprintf("%s is no target the matrix knows: list %s as not a target, with its reason", segment, pattern)
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
		return e
````

````new server/internal/bootstrap/permission_matrix_coverage_test.go
		return e
	}
	// checkSlug is a row of checkWorkspaceSlug, whose {slug} is no
	// workspace; notTarget is exempt with its path listed as not a target.
	checkSlug := apitest.Operation{ID: "checkWorkspaceSlug", Tags: []string{"workspace"}, Method: http.MethodGet,
		Path: "/api/v0/workspace-slugs/{slug}"}
	checks := matrixRow{op: "checkWorkspaceSlug", request: sameRequest(http.MethodGet, "/api/v0/workspace-slugs/acme", ""), cells: every(cellOK)}
	notTarget := exemptPublic("getWorkspaceInvitation")
	notTarget.notTargets = map[string]string{checkSlug.Path: "a slug asked about"}
	var unknown []string
	for _, c := range workspaceColumns {
		unknown = append(unknown, fmt.Sprintf("row checkWorkspaceSlug, %s: {slug} is no target the matrix knows: "+
			"list /api/v0/workspace-slugs/{slug} as not a target, with its reason", c))
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
			[]string{"operation getWorkspaceInvitation is exempt as public, and has a row"}},
````

````new server/internal/bootstrap/permission_matrix_coverage_test.go
			[]string{"operation getWorkspaceInvitation is exempt as public, and has a row"}},
		{"a parameter the matrix does not know", append(ops, checkSlug), exempt, []matrixRow{row, checks}, unknown},
		{"a path listed as not a target", append(ops, checkSlug), notTarget, []matrixRow{row, checks}, nil},
		{"a not-target path of no operation", ops, notTarget, []matrixRow{row},
			[]string{"the not-target path /api/v0/workspace-slugs/{slug} is no operation's"}},
````

- [ ] **Step 7: 测试、lint**

Run: `go -C server test -count=1 ./internal/modules/workspace/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestPermissionMatrix$|TestThePermissionMatrixCoversEveryOperation|TestMatrixViolationsCatchesEachGap|TestTheInvitationLinkAnswersEveryCallerAlike|TestBodiesThatBreakTheStructureAnswer400' ./internal/bootstrap/`
Expected: `ok`。

Run: `go -C server test -count=1 -v -run 'TestPermissionMatrix$' ./internal/bootstrap/`
Expected: `ok`；132 格，`TestPermissionMatrix` 约 0.2–1.5 秒（写进 review）。

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
git add api/modules/workspace.yaml api/openapi.yaml server/internal/bootstrap/permission_matrix_coverage_test.go server/internal/bootstrap/permission_matrix_invitations_test.go server/internal/bootstrap/permission_matrix_seeded_test.go server/internal/bootstrap/permission_matrix_test.go server/internal/bootstrap/permission_matrix_workspace_test.go server/internal/modules/workspace/adapter/http/handler.go server/internal/modules/workspace/adapter/http/handler_test.go server/internal/modules/workspace/adapter/http/invitations.go server/internal/modules/workspace/adapter/http/invitations_test.go server/internal/modules/workspace/app/accept_invitation.go server/internal/modules/workspace/app/accept_invitation_test.go server/internal/modules/workspace/app/clock_test.go server/internal/modules/workspace/app/decline_invitation.go server/internal/modules/workspace/app/decline_invitation_test.go server/internal/modules/workspace/app/fakes_invitations_test.go server/internal/modules/workspace/app/invitation_ports.go server/internal/modules/workspace/app/list_invitations_test.go server/internal/modules/workspace/app/respond_invitation.go server/internal/modules/workspace/domain/errors.go server/internal/modules/workspace/module.go web/apps/web/helpers/authentication.helper.ts web/packages/i18n/src/locales/en/auth.json web/packages/i18n/src/locales/zh-CN/auth.json api/dist/openapi.yaml server/internal/modules/workspace/adapter/http/gen/bodyshape.gen.go server/internal/modules/workspace/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M3/P3): accept or decline an invitation by its link

POST /api/v0/workspace-invitations/{invitation_id}/accept and /decline,
with the link's token in the body. The token is checked before anything
is read. Then, in one transaction, the caller's account row FOR SHARE
first, its state and address read under that lock; the workspace, FOR NO
KEY UPDATE to accept and FOR SHARE to decline; the invitation FOR UPDATE,
read again (M3 design 3.6 conventions 1 and 6). Another address is 403
workspace.invitation_email_mismatch and writes nothing. Accepting never
changes an active membership; it restores an ended one with the
invitation's role, or inserts one (M3 design 3.8, 9.1).

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A）：

| 改坏 | 必须失败的测试 |
|---|---|
| 令牌不对答 403；在锁之后核对令牌 | `TestAcceptWorkspaceInvitationRefusals`（前者另有 `TestDeclineWorkspaceInvitationRefusals`） |
| 账户在工作区和邀请之后锁 | `TestAcceptWorkspaceInvitation`、`TestDeclineWorkspaceInvitation`（Task 13 起另有 `TestDecliningAndDeactivating`） |
| 停用的账户可以回应；不比较邮箱；已忽略的可以再回应 | 两个用例的拒绝测试（第二个另有矩阵） |
| 别的邮箱被告知"已回应"；锁下不重读邀请 | `TestAcceptWorkspaceInvitationRefusals` |
| 给有效成员邀请的角色；回答邀请的角色 | `TestAcceptWorkspaceInvitation` |
| 以原来的角色恢复；给以前的成员再插一行 | `TestAcceptWorkspaceInvitation` |
| 新成员关系记成邀请人的；接受写到工作区的 id | `TestAcceptWorkspaceInvitation` |
| 忽略用 `FOR NO KEY UPDATE`；接受用 `FOR SHARE`；接受不锁工作区 | 各自的用例测试（最后一个 Task 13 起另有 `TestAcceptingAndDeletingTheWorkspace`） |
| 接受、忽略的锁在事务之前 | 各自的用例测试 |
| 接受、忽略在锁之前读时钟 | `TestEachWriteReadsTheClockUnderItsLock` |
| 失败的读、锁答成邀请的 404 | `TestAcceptWorkspaceInvitationRefusals`、`TestUpdateWorkspaceInvitationRefusals`、`TestDeleteWorkspaceInvitationRefusals` |
| HTTP 测试不答 `workspace.invitation_email_mismatch` | `apitest.Main`：`declares problem codes that no test answered` |

**Done when:** 两个用例、HTTP、矩阵的 24 格通过；矩阵共 132 格，耗时记下；新码有文案，`make test-web` 通过。

---

### Task 12: 凭邀请注册

**Files:**
- Create: `server/internal/bootstrap/signup_policy.go`、`server/internal/bootstrap/signup_policy_test.go`、`server/internal/modules/workspace/app/signup_invitations.go`、`server/internal/modules/workspace/app/signup_invitations_test.go`
- Modify: `api/modules/identity.yaml`、`server/internal/bootstrap/app.go`、`server/internal/bootstrap/invitations_test.go`、`server/internal/modules/identity/adapter/http/auth.go`、`server/internal/modules/identity/adapter/http/handler_test.go`、`server/internal/modules/identity/app/fakes_test.go`、`server/internal/modules/identity/app/ports.go`、`server/internal/modules/identity/app/register.go`、`server/internal/modules/identity/app/register_test.go`、`server/internal/modules/identity/module.go`、`server/internal/modules/workspace/app/invitation_ports.go`、`server/internal/modules/workspace/module.go`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/identity/adapter/http/gen/bodyshape.gen.go`、`server/internal/modules/identity/adapter/http/gen/server.gen.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Produces（spec 2.12，M3 设计 3.8，决策点 1）：`identityapp.SignupInvitation{ID, Token}`（`identity.SignupInvitation` 是别名）；`SignupPolicy.AllowSignup(ctx, email string, invitation *SignupInvitation) (bool, error)`；`RegisterInput.Invitation`；`RegisterRequest.invitation`（`RegisterInvitation{id, token}`）；`workspace/app.SignupInvitations`、`NewSignupInvitations(invitations InvitationFinder, mac)`、`Allows(ctx, email, id, token) (bool, error)`；`workspace.SignupInvitations` 接口、`(*Module).SignupInvitations()`；`bootstrap.signupPolicy{enabled, invitations}` 代替 `signupSwitch`（删除）。
- 使用者：Task 14 的 W6；P9 的注册页。

**Tests:**
- `workspace/app/signup_invitations_test.go`：`TestSignupInvitationsAllowTheInvitedAddress`（carol 到 `acme`、erin 到 `beta`，先核对令牌，不开事务）；`TestSignupInvitationsRefuseEveryOtherCase`（令牌不对什么都不读；不在、已删除（已接受的也是）、已忽略、别的邮箱、没有规范化的邮箱都是 `false` 而没有错误；读失败是错误）。
- `bootstrap/signup_policy_test.go`：`TestSignupPolicy`（开放时一律允许、不问邀请，坏的也不问；关闭时不带邀请拒绝，带邀请照 `workspace` 的回答，它的失败原样）。
- `identity/app/register_test.go`：`TestRegisterAsksThePolicyAboutTheAddressAndTheInvitation`（规范化后的邮箱和原样的邀请，只问一次、在一切之前；拒绝答 `identity.signup_disabled`，带邀请也是）；`identity/adapter/http/handler_test.go`：`TestRegisterHandsOnTheInvitation`。
- `bootstrap/invitations_test.go`：`TestRegisteringWithAnInvitationWhileSignupIsOff`（同一个库、同一个密钥上关闭注册的第二个 app：带有效邀请、邮箱相同（` Carol@Example.com `）201，邀请仍待接受；不带邀请、令牌不对、id 不存在、已删除、已接受、已忽略、别的邮箱，七种逐字节相同的 403）。

- [ ] **Step 1: 接口描述**

`api/modules/identity.yaml`（修改，2 处）：

````old api/modules/identity.yaml
        whether the address is registered. The password needs
````

````new api/modules/identity.yaml
        whether the address is registered; unless it names an invitation
        whose link it holds, pending, to the address it registers
        (normalized). Every other invitation answers the same
        identity.signup_disabled, whatever is wrong with it. Registering
        does not accept the invitation. The password needs
````

````old api/modules/identity.yaml
        password:
          type: string
    LoginRequest:
````

````new api/modules/identity.yaml
        password:
          type: string
        invitation:
          $ref: '#/components/schemas/RegisterInvitation'
    RegisterInvitation:
      description: >-
        The invitation a registration names, from its link
        (/workspace-invitations?invitation_id=…&token=…): while sign-up is
        off, it lets the address it was sent to register.
      type: object
      additionalProperties: false
      required: [id, token]
      properties:
        id:
          description: The link's invitation_id.
          type: string
          format: uuid
        token:
          description: The link's token.
          type: string
    LoginRequest:
````

- [ ] **Step 2: 生成**

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `333bc8249ada8f26db35d0d53a542fa23c4cc1ec9b91daf1b5572de704f3aac5` | 1684 | `api/dist/openapi.yaml` |
| `16121159dcf19d0d8757df211504af72ea20c9be55584e6371556f4280c01a90` | 62 | `server/internal/modules/identity/adapter/http/gen/bodyshape.gen.go` |
| `02e821378917d28e75758f61d8d34c5eb1ca7c8cf5afbb42c2ab7d9f612eba6d` | 1807 | `server/internal/modules/identity/adapter/http/gen/server.gen.go` |
| `c71f1506f121b6ea823fa41ed9d7c1156bd2ef52dce1554c20fc7e1be3779316` | 1803 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/identity/adapter/http/gen/bodyshape.gen.go server/internal/modules/identity/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 3: `identity` 的注册带着邀请问策略**

`server/internal/modules/identity/app/ports.go`（修改，1 处）：

````old server/internal/modules/identity/app/ports.go
// SignupPolicy decides whether registration is open (M2 design 3.9). From
// bootstrap it is auth.signup_enabled; M3 extends it to invitations.
type SignupPolicy interface {
	AllowSignup(ctx context.Context) (bool, error)
}

````

````new server/internal/modules/identity/app/ports.go
// SignupInvitation is the invitation a registration names (M3 design 3.8):
// its link's invitation id and token, as the request sent them.
type SignupInvitation struct {
	ID    uuid.UUID
	Token string
}

// SignupPolicy decides whether a registration may go on (M2 design 3.9, M3
// design 3.8): for email, normalized, and the invitation the request names,
// nil when none. From bootstrap it is auth.signup_enabled and, while that
// is off, workspace's check of the invitation. Every refusal is false: the
// registration answers identity.signup_disabled, whatever the reason.
type SignupPolicy interface {
	AllowSignup(ctx context.Context, email string, invitation *SignupInvitation) (bool, error)
}

````

`server/internal/modules/identity/app/register.go`（修改，3 处）：

````old server/internal/modules/identity/app/register.go
	Email     string
	Password  string
	UserAgent string
	IP        netip.Addr
````

````new server/internal/modules/identity/app/register.go
	Email      string
	Password   string
	Invitation *SignupInvitation // the invitation the request names; nil when none
	UserAgent  string
	IP         netip.Addr
````

````old server/internal/modules/identity/app/register.go
//  1. closed sign-up answers 403 before any other check (M2 design 3.9);
````

````new server/internal/modules/identity/app/register.go
//  1. the policy decides on the address, normalized, and the invitation,
//     before any other check: a refusal answers 403 for every address and
//     every invitation alike (M2 design 3.9, M3 design 3.8);
````

````old server/internal/modules/identity/app/register.go
	allowed, err := r.d.Policy.AllowSignup(ctx)
````

````new server/internal/modules/identity/app/register.go
	allowed, err := r.d.Policy.AllowSignup(ctx, shared.NormalizeEmail(in.Email), in.Invitation)
````

`server/internal/modules/identity/app/fakes_test.go`（修改，4 处）：

````old server/internal/modules/identity/app/fakes_test.go
	"errors"
````

````new server/internal/modules/identity/app/fakes_test.go
	"errors"
	"fmt"
````

````old server/internal/modules/identity/app/fakes_test.go
}

type fixedPolicy struct {
````

````new server/internal/modules/identity/app/fakes_test.go
}

// fixedPolicy answers allow and err, and records what it was asked in
// asked, when set.
type fixedPolicy struct {
````

````old server/internal/modules/identity/app/fakes_test.go
	err   error
````

````new server/internal/modules/identity/app/fakes_test.go
	err   error
	asked *[]string
````

````old server/internal/modules/identity/app/fakes_test.go
func (p fixedPolicy) AllowSignup(context.Context) (bool, error) { return p.allow, p.err }
````

````new server/internal/modules/identity/app/fakes_test.go
func (p fixedPolicy) AllowSignup(_ context.Context, email string, invitation *app.SignupInvitation) (bool, error) {
	if p.asked != nil {
		*p.asked = append(*p.asked, fmt.Sprintf("%s %+v", email, invitation))
	}
	return p.allow, p.err
}
````

`server/internal/modules/identity/app/register_test.go`（修改，3 处）：

````old server/internal/modules/identity/app/register_test.go
	"errors"
````

````new server/internal/modules/identity/app/register_test.go
	"errors"
	"fmt"
````

````old server/internal/modules/identity/app/register_test.go
	"net/netip"
````

````new server/internal/modules/identity/app/register_test.go
	"net/netip"
	"slices"
````

````old server/internal/modules/identity/app/register_test.go
}

func TestRegisterValidatesBeforeHashing(t *testing.T) {
````

````new server/internal/modules/identity/app/register_test.go
}

// The policy is asked about the address as registration normalizes it and
// the invitation as the request named it, once, before anything else: a
// refusal answers identity.signup_disabled for an invitation too, and an
// allowed one registers the address.
func TestRegisterAsksThePolicyAboutTheAddressAndTheInvitation(t *testing.T) {
	invitation := &app.SignupInvitation{ID: uuid.NewV7(), Token: "nrv_inv_AAAAAAAAAAAAAAAAAAAAAA"}
	for _, tt := range []struct {
		invitation *app.SignupInvitation
		allow      bool
	}{{nil, true}, {invitation, true}, {invitation, false}} {
		var asked []string
		f := newRegister(fixedPolicy{allow: tt.allow, asked: &asked})
		in := input
		in.Invitation = tt.invitation

		_, err := f.uc.Execute(context.Background(), in)

		if want := []string{fmt.Sprintf("alice@corp.com %+v", tt.invitation)}; !slices.Equal(asked, want) {
			t.Errorf("the policy was asked %q, want %q", asked, want)
		}
		registered := err == nil && len(f.store.users) == 1 && f.store.users[0].Email == "alice@corp.com"
		if tt.allow != registered || (!tt.allow && (!errors.Is(err, domain.ErrSignupDisabled) || f.hasher.calls != 0)) {
			t.Errorf("allowed %v with %+v: Execute() = %v, users %d, hashes %d", tt.allow, tt.invitation, err, len(f.store.users), f.hasher.calls)
		}
	}
}

func TestRegisterValidatesBeforeHashing(t *testing.T) {
````

`server/internal/modules/identity/adapter/http/auth.go`（修改，1 处）：

````old server/internal/modules/identity/adapter/http/auth.go
		return nil, err
	}
	tokens, err := h.uc.Register.Execute(ctx, app.RegisterInput{
		Email:     req.Body.Email,
		Password:  req.Body.Password,
		UserAgent: meta.UserAgent,
		IP:        meta.ClientIP,
````

````new server/internal/modules/identity/adapter/http/auth.go
		return nil, err
	}
	var invitation *app.SignupInvitation
	if inv := req.Body.Invitation; inv != nil {
		invitation = &app.SignupInvitation{ID: inv.ID, Token: inv.Token}
	}
	tokens, err := h.uc.Register.Execute(ctx, app.RegisterInput{
		Email:      req.Body.Email,
		Password:   req.Body.Password,
		Invitation: invitation,
		UserAgent:  meta.UserAgent,
		IP:         meta.ClientIP,
````

`server/internal/modules/identity/adapter/http/handler_test.go`（修改，1 处）：

````old server/internal/modules/identity/adapter/http/handler_test.go
}

// The handler exit: every error the use case returns becomes its problem.
````

````new server/internal/modules/identity/adapter/http/handler_test.go
}

// The invitation a registration names reaches the use case as sent.
func TestRegisterHandsOnTheInvitation(t *testing.T) {
	register := &fakeRegister{}
	req := registerRequest(`{"email":"alice@corp.com","password":"Tr0ub4dor&3",` +
		`"invitation":{"id":"0199a2b4-0000-7000-8000-0000000000c1","token":"nrv_inv_kvqyKBh-bANMT6JAIYzolA"}}`)
	apitest.Load(t).CheckRequest(t, req)

	if res, body := do(t, newServer(t, fakes{register: register}), req); res.StatusCode != http.StatusCreated {
		t.Fatalf("POST /auth/register = %d %s, want 201", res.StatusCode, body)
	}
	want := app.SignupInvitation{ID: uuid.MustParse("0199a2b4-0000-7000-8000-0000000000c1"), Token: "nrv_inv_kvqyKBh-bANMT6JAIYzolA"}
	if register.got.Invitation == nil || *register.got.Invitation != want {
		t.Errorf("use case got the invitation %+v, want %+v", register.got.Invitation, want)
	}
}

// The handler exit: every error the use case returns becomes its problem.
````

`server/internal/modules/identity/module.go`（修改，2 处）：

````old server/internal/modules/identity/module.go
)

// Deps are what bootstrap builds for the module.
````

````new server/internal/modules/identity/module.go
)

// SignupInvitation is the invitation a registration names, as SignupPolicy
// is asked about it (M3 design 3.8).
type SignupInvitation = app.SignupInvitation

// Deps are what bootstrap builds for the module.
````

````old server/internal/modules/identity/module.go
	// SignupPolicy is auth.signup_enabled (M2 decision 2).
````

````new server/internal/modules/identity/module.go
	// SignupPolicy is auth.signup_enabled (M2 decision 2) and, while that is
	// off, the check of the invitation a registration names (M3 design
	// 3.8).
````

- [ ] **Step 4: `workspace` 的检查**

`server/internal/modules/workspace/app/invitation_ports.go`（修改，2 处）：

````old server/internal/modules/workspace/app/invitation_ports.go
}

// InvitationLocker reads an invitation by its id, then locks it under its
````

````new server/internal/modules/workspace/app/invitation_ports.go
}

// InvitationFinder reads an invitation by its id.
type InvitationFinder interface {
	// InvitationByID returns the undeleted invitation id; ErrNotFound when
	// there is none.
	InvitationByID(ctx context.Context, id uuid.UUID) (domain.Invitation, error)
}

// InvitationLocker reads an invitation by its id, then locks it under its
````

````old server/internal/modules/workspace/app/invitation_ports.go
	// InvitationByID returns the undeleted invitation id; ErrNotFound when
	// there is none.
	InvitationByID(ctx context.Context, id uuid.UUID) (domain.Invitation, error)
	// LockInvitation locks the undeleted invitation id FOR UPDATE until the
````

````new server/internal/modules/workspace/app/invitation_ports.go
	InvitationFinder
	// LockInvitation locks the undeleted invitation id FOR UPDATE until the
````

`server/internal/modules/workspace/app/signup_invitations.go`（新文件，40 行）：

````file server/internal/modules/workspace/app/signup_invitations.go
package app

import (
	"context"
	"errors"
	"uuid"
)

// SignupInvitations checks the invitation a registration names while
// sign-up is closed (M3 design 3.8, decision 1): bootstrap's signup policy
// asks it for identity's registration. It reads; it accepts nothing.
type SignupInvitations struct {
	invitations InvitationFinder
	tokens      invitationTokens
}

// NewSignupInvitations returns the check.
func NewSignupInvitations(invitations InvitationFinder, mac InvitationMAC) *SignupInvitations {
	return &SignupInvitations{invitations: invitations, tokens: invitationTokens{mac: mac}}
}

// Allows reports whether token is the link of the invitation id, pending,
// to email, normalized as registration normalizes it. The token is checked
// first and reads nothing. Every other case is the same false, which the
// registration answers with the same identity.signup_disabled (M3 design
// 8.2): a wrong token, an invitation not there, deleted, accepted or
// declined, another address. A failed read is an error, never a refusal.
func (s *SignupInvitations) Allows(ctx context.Context, email string, id uuid.UUID, token string) (bool, error) {
	if !s.tokens.valid(id, token) {
		return false, nil
	}
	inv, err := s.invitations.InvitationByID(ctx, id)
	switch {
	case errors.Is(err, ErrNotFound):
		return false, nil
	case err != nil:
		return false, err
	}
	return !inv.Responded() && inv.Email == email, nil
}
````

`server/internal/modules/workspace/app/signup_invitations_test.go`（新文件，77 行）：

````file server/internal/modules/workspace/app/signup_invitations_test.go
package app_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
)

// A registration's invitation allows the address it was sent to, with its
// link's token, while it is pending: carol's to acme, erin's to beta. The
// token is checked first; the invitation is read without a transaction.
func TestSignupInvitationsAllowTheInvitedAddress(t *testing.T) {
	for _, tt := range []struct {
		email string
		id    uuid.UUID
	}{{"carol@corp.com", carolToAcme.ID}, {"erin@corp.com", erinToBeta.ID}} {
		f := newInvitations()
		allowed, err := app.NewSignupInvitations(f.invitations, f.mac).Allows(context.Background(), tt.email, tt.id, tokenOf(f.mac, tt.id))
		if err != nil || !allowed {
			t.Errorf("%s: Allows() = %v, %v; want true", tt.email, allowed, err)
		}
		if want := []string{"Verify " + tt.id.String(), "InvitationByID " + tt.id.String() + " outside tx"}; !slices.Equal(f.log.calls, want) {
			t.Errorf("%s: calls = %q, want %q", tt.email, f.log.calls, want)
		}
	}
}

// Every other case is false, without an error: a token not the
// invitation's reads nothing; an invitation not there or deleted (as an
// accepted one is), a declined one, another address, the address not
// normalized. A failed read is an error.
func TestSignupInvitationsRefuseEveryOtherCase(t *testing.T) {
	failure := errors.New("connection reset")
	nobodys := uuid.NewV7()
	for _, tt := range []struct {
		name, email string
		id          uuid.UUID
		token       string // "" for the invitation's
		set         func(f *invitationsFixture)
		read        bool
		err         error
	}{
		{"another invitation's token", "carol@corp.com", carolToAcme.ID, "other", nil, false, nil},
		{"no such invitation", "carol@corp.com", nobodys, "", nil, true, nil},
		{"deleted or accepted", "carol@corp.com", carolToAcme.ID, "",
			func(f *invitationsFixture) { f.invitations.invitations = []domain.Invitation{daveToAcme, erinToBeta} }, true, nil},
		{"declined", "dave@corp.com", daveToAcme.ID, "", nil, true, nil},
		{"another address", "erin@corp.com", carolToAcme.ID, "", nil, true, nil},
		{"the address not normalized", "Carol@corp.com", carolToAcme.ID, "", nil, true, nil},
		{"the read failed", "carol@corp.com", carolToAcme.ID, "", func(f *invitationsFixture) { f.invitations.readErr = failure }, true, failure},
	} {
		f := newInvitations()
		if tt.set != nil {
			tt.set(f)
		}
		token := tokenOf(f.mac, tt.id)
		if tt.token == "other" {
			token = tokenOf(f.mac, daveToAcme.ID)
		}
		allowed, err := app.NewSignupInvitations(f.invitations, f.mac).Allows(context.Background(), tt.email, tt.id, token)
		if allowed || !errors.Is(err, tt.err) || (tt.err == nil && err != nil) {
			t.Errorf("%s: Allows() = %v, %v; want false, %v", tt.name, allowed, err, tt.err)
		}
		want := []string{"Verify " + tt.id.String()}
		if tt.read {
			want = append(want, "InvitationByID "+tt.id.String()+" outside tx")
		}
		if !slices.Equal(f.log.calls, want) {
			t.Errorf("%s: calls = %q, want %q", tt.name, f.log.calls, want)
		}
	}
}
````

`server/internal/modules/workspace/module.go`（修改，3 处）：

````old server/internal/modules/workspace/module.go
	uc httpadapter.UseCases
````

````new server/internal/modules/workspace/module.go
	uc     httpadapter.UseCases
	signup *app.SignupInvitations
}

// SignupInvitations checks the invitation a registration names while
// sign-up is closed (M3 design 3.8): bootstrap's signup policy asks it.
type SignupInvitations interface {
	// Allows reports whether token is the link of the invitation id,
	// pending, to email, normalized. Every other case is the same false.
	Allows(ctx context.Context, email string, id uuid.UUID, token string) (bool, error)
````

````old server/internal/modules/workspace/module.go
	return &Module{uc: httpadapter.UseCases{
````

````new server/internal/modules/workspace/module.go
	return &Module{signup: app.NewSignupInvitations(store, d.InvitationMAC), uc: httpadapter.UseCases{
````

````old server/internal/modules/workspace/module.go
}

// Actions lists the module's actions: bootstrap's test holds the union of
````

````new server/internal/modules/workspace/module.go
}

// SignupInvitations is the check of a registration's invitation, for
// identity's SignupPolicy (M3 design 6.6 step 6).
func (m *Module) SignupInvitations() SignupInvitations {
	return m.signup
}

// Actions lists the module's actions: bootstrap's test holds the union of
````

- [ ] **Step 5: `bootstrap` 的策略**

`server/internal/bootstrap/signup_policy.go`（新文件，30 行）：

````file server/internal/bootstrap/signup_policy.go
package bootstrap

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
)

// signupPolicy is identity's SignupPolicy (M3 design 3.8, 6.5):
// auth.signup_enabled; while it is off, a registration goes on only with an
// invitation workspace allows for its address. Neither module knows the
// other: the switch and the check meet here.
type signupPolicy struct {
	enabled     bool
	invitations workspace.SignupInvitations
}

// AllowSignup allows every registration while sign-up is on, without
// asking about the invitation; while it is off, none without one, and one
// with one as workspace answers.
func (p signupPolicy) AllowSignup(ctx context.Context, email string, invitation *identity.SignupInvitation) (bool, error) {
	switch {
	case p.enabled:
		return true, nil
	case invitation == nil:
		return false, nil
	}
	return p.invitations.Allows(ctx, email, invitation.ID, invitation.Token)
}
````

`server/internal/bootstrap/signup_policy_test.go`（新文件，59 行）：

````file server/internal/bootstrap/signup_policy_test.go
package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
)

// fakeSignupInvitations answers allowed and err, and records each question.
type fakeSignupInvitations struct {
	allowed bool
	err     error
	asked   []string
}

func (f *fakeSignupInvitations) Allows(_ context.Context, email string, id uuid.UUID, token string) (bool, error) {
	f.asked = append(f.asked, fmt.Sprintf("%s %s %s", email, id, token))
	return f.allowed, f.err
}

// While sign-up is on, every registration goes on and no invitation is
// asked about, a bad one neither; while it is off, none without an
// invitation, and one with an invitation as workspace answers, its failure
// the failure.
func TestSignupPolicy(t *testing.T) {
	inv := &identity.SignupInvitation{ID: uuid.NewV7(), Token: "nrv_inv_AAAAAAAAAAAAAAAAAAAAAA"}
	asked := []string{"carol@corp.com " + inv.ID.String() + " " + inv.Token}
	failure := errors.New("connection reset")
	for _, tt := range []struct {
		name       string
		enabled    bool
		invitation *identity.SignupInvitation
		workspace  fakeSignupInvitations
		want       bool
		err        error
		asked      []string
	}{
		{"on, without an invitation", true, nil, fakeSignupInvitations{}, true, nil, nil},
		{"on, with one workspace refuses", true, inv, fakeSignupInvitations{}, true, nil, nil},
		{"off, without an invitation", false, nil, fakeSignupInvitations{allowed: true}, false, nil, nil},
		{"off, with one workspace allows", false, inv, fakeSignupInvitations{allowed: true}, true, nil, asked},
		{"off, with one workspace refuses", false, inv, fakeSignupInvitations{}, false, nil, asked},
		{"off, the check failed", false, inv, fakeSignupInvitations{err: failure}, false, failure, asked},
	} {
		invitations := tt.workspace
		got, err := signupPolicy{enabled: tt.enabled, invitations: &invitations}.AllowSignup(context.Background(), "carol@corp.com", tt.invitation)
		if got != tt.want || !errors.Is(err, tt.err) || (tt.err == nil && err != nil) {
			t.Errorf("%s: AllowSignup() = %v, %v; want %v, %v", tt.name, got, err, tt.want, tt.err)
		}
		if !slices.Equal(invitations.asked, tt.asked) {
			t.Errorf("%s: workspace was asked %q, want %q", tt.name, invitations.asked, tt.asked)
		}
	}
}
````

`server/internal/bootstrap/app.go`（修改，2 处）：

````old server/internal/bootstrap/app.go
		SignupPolicy:    signupSwitch(cfg.Auth.SignupEnabled),
````

````new server/internal/bootstrap/app.go
		SignupPolicy:    signupPolicy{enabled: cfg.Auth.SignupEnabled, invitations: ws.SignupInvitations()},
````

````old server/internal/bootstrap/app.go

// signupSwitch is auth.signup_enabled as identity's SignupPolicy.
type signupSwitch bool

func (s signupSwitch) AllowSignup(context.Context) (bool, error) { return bool(s), nil }

````

````new server/internal/bootstrap/app.go

````

`server/internal/bootstrap/invitations_test.go`（修改，2 处）：

````old server/internal/bootstrap/invitations_test.go
	"uuid"

````

````new server/internal/bootstrap/invitations_test.go
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
````

````old server/internal/bootstrap/invitations_test.go
	return invitationLink{list.Data[0].ID, list.Data[0].Token}
````

````new server/internal/bootstrap/invitations_test.go
	return invitationLink{list.Data[0].ID, list.Data[0].Token}
}

// answerInvitation has the bearer accept or decline the invitation of
// link, wanting status.
func answerInvitation(t *testing.T, contract *apitest.Contract, base, bearer, answer string, link invitationLink, status int) {
	t.Helper()
	got, body := call(t, contract, http.MethodPost, base+"/api/v0/workspace-invitations/"+link.id.String()+"/"+answer, bearer,
		`{"token":"`+link.token+`"}`)
	if got != status {
		t.Fatalf("%s %s = %d %s, want %d", answer, link.id, got, body, status)
	}
}

// While sign-up is off, a registration goes on only with the link of a
// pending invitation to its address, normalized (M3 design 3.8): on a
// second app on the same database and key, with sign-up off. Every other
// case answers the one 403 identity.signup_disabled, byte for byte, whatever
// is wrong (8.2): no invitation, a token not the invitation's, an id no
// invitation has, a deleted invitation, an accepted one (its address has an
// account, and that is not what it says), a declined one, another address.
// Registering does not accept the invitation.
func TestRegisteringWithAnInvitationWhileSignupIsOff(t *testing.T) {
	url, keyFile := pgtest.NewDatabase(t), writeFile(t, testKeyPEM)
	configured := func(signup bool) config.Config {
		cfg := testConfig(t, url, false)
		cfg.Auth.JWT.PrivateKeyFile, cfg.Auth.SignupEnabled = keyFile, signup
		return cfg
	}
	contract := apitest.Load(t)
	open := startApp(t, configured(true), migrations.FS())
	admin := registerAccount(t, contract, open, "admin@example.com").AccessToken
	if status, body := call(t, contract, http.MethodPost, open+"/api/v0/workspaces", admin, `{"name":"Acme","slug":"acme"}`); status != http.StatusCreated {
		t.Fatalf("creating acme = %d %s", status, body)
	}
	links := map[string]invitationLink{}
	for _, name := range []string{"carol", "dave", "erin", "frank", "gina"} {
		links[name] = invite(t, contract, open, admin, "acme", name+"@example.com")
	}
	dave := registerAccount(t, contract, open, "dave@example.com").AccessToken
	answerInvitation(t, contract, open, dave, "decline", links["dave"], http.StatusNoContent)
	erin := registerAccount(t, contract, open, "erin@example.com").AccessToken
	answerInvitation(t, contract, open, erin, "accept", links["erin"], http.StatusOK)
	if status, body := call(t, contract, http.MethodDelete, open+"/api/v0/workspace-invitations/"+links["frank"].id.String(), admin, ""); status != http.StatusNoContent {
		t.Fatalf("deleting frank's invitation = %d %s", status, body)
	}
	closed := startApp(t, configured(false), migrations.FS())
	register := func(email, invitation string) wholeAnswer {
		t.Helper()
		req := newRequest(t, http.MethodPost, closed+"/api/v0/auth/register", "",
			[]byte(`{"email":"`+email+`","password":"Tr0ub4dor&3"`+invitation+`}`))
		contract.CheckRequest(t, req)
		res, body := send(t, req)
		contract.CheckResponse(t, req, res)
		return wholeAnswerOf(res, body)
	}
	naming := func(l invitationLink) string {
		return `,"invitation":{"id":"` + l.id.String() + `","token":"` + l.token + `"}`
	}
	nobodys := uuid.NewV7()

	refused := register("zoe@example.com", "")
	if refused.status != http.StatusForbidden || problemCode(t, []byte(refused.body)) != "identity.signup_disabled" {
		t.Fatalf("registering without an invitation = %+v, want 403 identity.signup_disabled", refused)
	}
	for _, tt := range []struct{ name, email, invitation string }{
		{"a token not the invitation's", "carol@example.com", naming(invitationLink{links["carol"].id, links["gina"].token})},
		{"an id no invitation has", "carol@example.com", naming(invitationLink{nobodys, invitationToken(t, nobodys)})},
		{"a deleted invitation", "frank@example.com", naming(links["frank"])},
		{"an accepted invitation", "erin@example.com", naming(links["erin"])},
		{"a declined invitation", "dave@example.com", naming(links["dave"])},
		{"another address", "zoe@example.com", naming(links["gina"])},
	} {
		if got := register(tt.email, tt.invitation); got != refused {
			t.Errorf("registering with %s = %+v, want what one without an invitation gets: %+v", tt.name, got, refused)
		}
	}
	if got := register(" Carol@Example.com ", naming(links["carol"])); got.status != http.StatusCreated {
		t.Errorf("registering carol with her invitation = %+v, want 201", got)
	}
	carol := links["carol"]
	if status, body := call(t, contract, http.MethodGet, closed+"/api/v0/workspace-invitations/"+carol.id.String()+"?token="+carol.token, "", ""); status != http.StatusOK ||
		strings.Contains(body, `"declined":true`) {
		t.Errorf("carol's invitation after she registered = %d %s, want it pending", status, body)
	}
````

- [ ] **Step 6: 测试、lint**

Run: `go -C server test -count=1 ./internal/modules/identity/... ./internal/modules/workspace/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestRegisteringWithAnInvitationWhileSignupIsOff|TestSignupPolicy|TestBodiesThatBreakTheStructureAnswer400|TestTheInvitationLinksTokenIsNotLogged' ./internal/bootstrap/`
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
git add api/modules/identity.yaml server/internal/bootstrap/app.go server/internal/bootstrap/invitations_test.go server/internal/bootstrap/signup_policy.go server/internal/bootstrap/signup_policy_test.go server/internal/modules/identity/adapter/http/auth.go server/internal/modules/identity/adapter/http/handler_test.go server/internal/modules/identity/app/fakes_test.go server/internal/modules/identity/app/ports.go server/internal/modules/identity/app/register.go server/internal/modules/identity/app/register_test.go server/internal/modules/identity/module.go server/internal/modules/workspace/app/invitation_ports.go server/internal/modules/workspace/app/signup_invitations.go server/internal/modules/workspace/app/signup_invitations_test.go server/internal/modules/workspace/module.go api/dist/openapi.yaml server/internal/modules/identity/adapter/http/gen/bodyshape.gen.go server/internal/modules/identity/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M3/P3): while sign-up is off, an invitation's link lets its address register

RegisterRequest names an invitation {id, token}; the signup policy is
asked about the normalized address and the invitation before anything
else. bootstrap's policy is auth.signup_enabled and, while it is off,
workspace's check: the link's token, then a pending invitation to that
address. Every other case is the same 403 identity.signup_disabled
(decision 1, M3 design 8.2). Registering does not accept the invitation.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A）：

| 改坏 | 必须失败的测试 |
|---|---|
| 注册说出是哪一种无效（邮箱不同答 500） | `TestSignupInvitationsRefuseEveryOtherCase`、`TestRegisteringWithAnInvitationWhileSignupIsOff` |
| 已忽略的、任何邮箱的、不要令牌的邀请让注册通过 | `TestSignupInvitationsRefuseEveryOtherCase`、`TestRegisteringWithAnInvitationWhileSignupIsOff` |
| 读邀请失败当作拒绝 | `TestSignupInvitationsRefuseEveryOtherCase` |
| `InvitationByID` 读已删除的（已接受、已删除的邀请让注册通过） | `TestRegisteringWithAnInvitationWhileSignupIsOff`、`TestTheInvitationReadsFindOnlyAnUndeletedInvitation` |
| 开放时也问邀请 | `TestSignupPolicy`、`TestRegisteringWithAnInvitationWhileSignupIsOff` |
| 关闭时不带邀请也通过 | `TestSignupPolicy`、`TestRegisteringWithAnInvitationWhileSignupIsOff` |
| 问策略用输入原样的邮箱 | `TestRegisterAsksThePolicyAboutTheAddressAndTheInvitation`、`TestRegisteringWithAnInvitationWhileSignupIsOff` |
| handler 丢掉邀请 | `TestRegisterHandsOnTheInvitation`、`TestRegisteringWithAnInvitationWhileSignupIsOff` |
| 组合根的策略总是开放 | `TestRegisteringWithAnInvitationWhileSignupIsOff` |

**Done when:** 策略、检查、注册和整程序测试通过；七种无效逐字节相同；注册不接受邀请。

---

### Task 13: `WaitForKeyWaitOn`；交错 3、9、12、18、19

**Files:**
- Create: `server/internal/bootstrap/interleaving_answers_test.go`、`server/internal/bootstrap/interleaving_invite_test.go`
- Modify: `server/internal/bootstrap/interleaving_test.go`、`server/internal/platform/postgres/pgtest/lockwait.go`、`server/internal/platform/postgres/pgtest/lockwait_test.go`

**Interfaces:**
- Produces（spec 2.13，M3 设计 9.3；P2 review 第 6 节第 2、3 件）：`pgtest.WaitForKeyWaitOn(t, pool, table, limit)`：只数"写过 `table`、等另一个事务结束、没有 tuple 锁"的连接（`WaitForLockWait`、`WaitForLockWaitOn` 不变）。`interleaving_test.go` 的 `gate` 加 10 秒的期限（`newGate` 记下，`wait` 到期返回错误）。`interleaving_answers_test.go`：`answerRace`（`acme` 的管理员 alice、被邀请的 bob）、`testInvitationMAC`、`gatedAccepter`、`gatedDeleter`、`gatedEmails`、`gatedDeactivation`（按 3.9 在撤销会话之前取 `acme` 的 N）、`gatedDecliner`；`interleaving_invite_test.go`：`inviteRace`（加上第二位管理员 carol 和两人的会话）、`gatedInviter`（插入之后停下）、`gatedPasswords`（写哈希之前停下）。
- 使用者：P5 的交错 1、4、5、6，P6 的交错 7、19（换成真实的停用）。

**Tests:**
- `pgtest/lockwait_test.go`：`TestWaitForKeyWaitOnSeesOnlyAKeysWaitOnItsTable`（正例：等同一个键的 `INSERT`，`WaitForLockWait` 证明它在等、`WaitForLockWaitOn` 看不到；反例各在自己的库里：写过这张表的事务等这张表的一行；只读过这张表的事务在另一张表上等键；写过这张表的事务等咨询锁）。
- `TestAcceptingAndDeletingTheWorkspace`（交错 3，`WaitForLockWaitOn "workspaces"`）；`TestAcceptingAndChangingTheAddress`（交错 12，`"users"`）；`TestDecliningAndDeactivating`（交错 19，`"users"`）；`TestInvitingAndResettingThePassword`（交错 9，`"users"`）；`TestInvitingOverlappingBatches`（交错 18，`WaitForKeyWaitOn "workspace_member_invites"`；`[x, y]` 对 `[y, x]` 和 `[x]` 对 `[x]`，两个顺序；后到的一方 422、下标是它请求中的、什么都没插入；没有 40P01）。每个都两个顺序，结果见 spec 2.13。

- [ ] **Step 1: 探针**

`server/internal/platform/postgres/pgtest/lockwait.go`（修改，1 处）：

````old server/internal/platform/postgres/pgtest/lockwait.go
}

// waitFor polls query, a count of the waiting backends, until it is
````

````new server/internal/platform/postgres/pgtest/lockwait.go
}

// WaitForKeyWaitOn returns once a backend connected to pool's database that
// has written table in its transaction waits for another transaction to
// end without holding a tuple lock, and fails the test when none has within
// limit, or at once when the database has no such table. That is the wait
// of a unique index's check: an INSERT whose key a live transaction has
// inserted too waits for that transaction's transactionid until it commits
// or rolls back, with no row to lock (M3 design 9.3, interleaving 18).
// WaitForLockWaitOn does not see it, having no tuple lock to look for. A
// wait for a row of the table holds its tuple lock and does not count; nor
// does a wait of a transaction that has not written the table, nor one for
// a lock of another kind, such as an advisory lock. By a transaction that
// has written table, it cannot tell a key's wait on table from one on
// another table, or from the two row waits PostgreSQL makes without a
// tuple lock (WaitForLockWaitOn): use it where the waiting transaction
// writes no other table with a unique key and upgrades no shared row lock.
func WaitForKeyWaitOn(t testing.TB, pool *pgxpool.Pool, table string, limit time.Duration) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()
	what := "a key of " + table
	var relation *uint32
	if err := pool.QueryRow(ctx, "SELECT to_regclass($1)::oid", table).Scan(&relation); err != nil {
		failPoll(ctx, t, limit, what, err)
	}
	if relation == nil {
		t.Fatalf("pgtest: no table %q", table)
	}
	waitFor(ctx, t, pool, limit, what, `
		SELECT count(*) FROM pg_stat_activity a
		WHERE a.datname = current_database() AND a.wait_event_type = 'Lock' AND a.wait_event = 'transactionid'
			AND EXISTS (SELECT 1 FROM pg_locks l WHERE l.pid = a.pid AND l.locktype = 'relation' AND l.relation = $1
				AND l.mode = 'RowExclusiveLock' AND l.granted)
			AND NOT EXISTS (SELECT 1 FROM pg_locks l WHERE l.pid = a.pid AND l.locktype = 'tuple')`, *relation)
}

// waitFor polls query, a count of the waiting backends, until it is
````

`server/internal/platform/postgres/pgtest/lockwait_test.go`（修改，3 处）：

````old server/internal/platform/postgres/pgtest/lockwait_test.go
}

// WaitForLockWaitOn fails at once for a table the database does not have: a
````

````new server/internal/platform/postgres/pgtest/lockwait_test.go
}

// WaitForKeyWaitOn sees an INSERT that waits for the transaction that
// inserted the same key into its table: a wait WaitForLockWaitOn does not
// see, having no tuple lock. It does not count a wait for a row of its
// table, though the waiter has written the table; a key's wait on another
// table by a transaction that has read its table; or a wait for an advisory
// lock by a transaction that has written its table. Each case waits in a
// database of its own, and WaitForLockWait proves it waits, so it fails for
// the kind of its wait.
func TestWaitForKeyWaitOnSeesOnlyAKeysWaitOnItsTable(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prepared := pgtest.NewDatabase(t)
	setup := connect(t, prepared)
	if _, err := setup.Exec(ctx, "CREATE TABLE a (id int PRIMARY KEY, n int); CREATE TABLE b (id int PRIMARY KEY); INSERT INTO a VALUES (1, 0)"); err != nil {
		t.Fatal(err)
	}
	if err := setup.Close(ctx); err != nil {
		t.Fatal(err)
	}
	key, row, other, advisory := pgtest.NewDatabaseFrom(t, prepared), pgtest.NewDatabaseFrom(t, prepared), pgtest.NewDatabaseFrom(t, prepared),
		pgtest.NewDatabaseFrom(t, prepared)
	holdAndWaitIn(t, key, "INSERT INTO a VALUES (2, 0)", "INSERT INTO a VALUES (2, 0)")
	holdAndWaitIn(t, row, "UPDATE a SET n = 1 WHERE id = 1", "INSERT INTO a VALUES (3, 0); UPDATE a SET n = 2 WHERE id = 1")
	holdAndWaitIn(t, other, "INSERT INTO b VALUES (2)", "SELECT count(*) FROM a; INSERT INTO b VALUES (2)")
	holdAndWaitIn(t, advisory, "SELECT pg_advisory_xact_lock(1)", "INSERT INTO a VALUES (3, 0); SELECT pg_advisory_xact_lock(1)")

	keyPool := newPool(t, key)
	pgtest.WaitForKeyWaitOn(t, keyPool, "a", 10*time.Second)
	if failed := fatalOf(func(tb testing.TB) { pgtest.WaitForLockWaitOn(tb, keyPool, "a", 300*time.Millisecond) }); failed !=
		"no statement waited for a row lock of a within 300ms" {
		t.Errorf("WaitForLockWaitOn on a key's wait failed with %q, want it to fail at its deadline", failed)
	}
	for _, tt := range []struct{ name, url string }{
		{"a row of the table, by a writer of it", row},
		{"a key of another table, by a reader of it", other},
		{"an advisory lock, by a writer of it", advisory},
	} {
		pool := newPool(t, tt.url)
		pgtest.WaitForLockWait(t, pool, 10*time.Second)
		if failed := fatalOf(func(tb testing.TB) { pgtest.WaitForKeyWaitOn(tb, pool, "a", 300*time.Millisecond) }); failed !=
			"no statement waited for a key of a within 300ms" {
			t.Errorf("%s: WaitForKeyWaitOn failed with %q, want it to fail at its deadline", tt.name, failed)
		}
	}
	start := time.Now()
	failed := fatalOf(func(tb testing.TB) { pgtest.WaitForKeyWaitOn(tb, keyPool, "workspace", 10*time.Second) })
	if took := time.Since(start); failed != `pgtest: no table "workspace"` || took > 5*time.Second {
		t.Errorf("WaitForKeyWaitOn(workspace) failed with %q after %v, want pgtest: no table \"workspace\" at once", failed, took)
	}
}

// WaitForLockWaitOn fails at once for a table the database does not have: a
````

````old server/internal/platform/postgres/pgtest/lockwait_test.go
			"no statement waited for a row lock of workspaces within 300ms"},
````

````new server/internal/platform/postgres/pgtest/lockwait_test.go
			"no statement waited for a row lock of workspaces within 300ms"},
		{"WaitForKeyWaitOn", func(tb testing.TB) { pgtest.WaitForKeyWaitOn(tb, pool, "workspaces", 300*time.Millisecond) },
			"no statement waited for a key of workspaces within 300ms"},
````

````old server/internal/platform/postgres/pgtest/lockwait_test.go
}

func newPool(t *testing.T, url string) *pgxpool.Pool {
````

````new server/internal/platform/postgres/pgtest/lockwait_test.go
}

// holdAndWaitIn makes a second connection to url wait: a first one runs
// held in a transaction it keeps open; the second runs waits in a
// transaction of its own, whose last statement waits for the first, until
// the test ends and the first rolls back.
func holdAndWaitIn(t *testing.T, url, held, waits string) {
	t.Helper()
	holder, waiter := connect(t, url), connect(t, url)
	tx, err := holder.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(context.Background(), held); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	done := make(chan error, 1)
	go func() {
		wtx, err := waiter.Begin(ctx)
		if err == nil {
			_, err = wtx.Exec(ctx, waits)
		}
		if err == nil {
			err = wtx.Rollback(ctx)
		}
		done <- err
	}()
	// Runs before the connections close: the first rolls back, and the
	// waiter goes on.
	t.Cleanup(func() {
		defer cancel()
		if err := tx.Rollback(context.Background()); err != nil {
			t.Error(err)
		}
		if err := <-done; err != nil {
			t.Errorf("the waiting connection: %v", err)
		}
	})
}

func newPool(t *testing.T, url string) *pgxpool.Pool {
````

- [ ] **Step 2: 闸门的期限**

`server/internal/bootstrap/interleaving_test.go`（修改，5 处）：

````old server/internal/bootstrap/interleaving_test.go
// waits until open is closed.
````

````new server/internal/bootstrap/interleaving_test.go
// waits until open is closed, the caller's context ends, or 10 seconds
// after the gate was made, whichever comes first. The gate's own deadline
// is for a side that lost the test's context, such as a use case that runs
// a statement on context.Background(): its wait would never end, and the
// test's cleanup, closing the pool, would wait for its connection forever.
````

````old server/internal/bootstrap/interleaving_test.go
	held, open chan struct{}
````

````new server/internal/bootstrap/interleaving_test.go
	held, open chan struct{}
	deadline   time.Time
````

````old server/internal/bootstrap/interleaving_test.go
func newGate() *gate { return &gate{held: make(chan struct{}), open: make(chan struct{})} }
````

````new server/internal/bootstrap/interleaving_test.go
func newGate() *gate {
	return &gate{held: make(chan struct{}), open: make(chan struct{}), deadline: time.Now().Add(10 * time.Second)}
}
````

````old server/internal/bootstrap/interleaving_test.go
	close(g.held)
````

````new server/internal/bootstrap/interleaving_test.go
	close(g.held)
	expired := time.NewTimer(time.Until(g.deadline))
	defer expired.Stop()
````

````old server/internal/bootstrap/interleaving_test.go
		return ctx.Err()
````

````new server/internal/bootstrap/interleaving_test.go
		return ctx.Err()
	case <-expired.C:
		return errors.New("the gate was not opened within 10s")
````

- [ ] **Step 3: 交错 3、12、19**

`server/internal/bootstrap/interleaving_answers_test.go`（新文件，398 行）：

````file server/internal/bootstrap/interleaving_answers_test.go
package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/access"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	identitypg "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	identitydomain "github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	workspacedomain "github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Interleavings 3, 12 and 19 of M3 design 9.3: an answer to an invitation
// against the deletion of its workspace, against the change of the
// invitee's address, and against his deactivation; each in both orders, on
// a real database, through the real use cases as bootstrap wires them. The
// side that goes first stops at a gate inside its transaction, holding its
// locks; the test waits until pgtest shows the other waiting on the table
// the design says it waits on, then opens the gate. Only the two sides run
// on the database, so the one wait each probe sees is the one meant: the
// deletion's and the acceptance's lock of the workspace row
// (WaitForLockWaitOn "workspaces"), and the lock of the invitee's account
// row (WaitForLockWaitOn "users"), which the answer takes first. Every wait
// has a deadline.

// answerRace is a database with acme, whose admin is alice, and bob, who is
// invited to acme at his address, each account with its profile;
// invitation is the link of bob's invitation, as a member.
type answerRace struct {
	pool       *pgxpool.Pool
	alice, bob uuid.UUID
	acme       uuid.UUID
	invitation invitationLink
	mac        workspaceapp.InvitationMAC
}

func newAnswerRace(t *testing.T) answerRace {
	t.Helper()
	pool := openPool(t, pgtest.NewDatabase(t))
	r := answerRace{pool: pool, alice: uuid.NewV7(), bob: uuid.NewV7(), acme: uuid.NewV7(), mac: testInvitationMAC(t)}
	now := time.Now()
	users := identitypg.New(pool)
	for id, email := range map[uuid.UUID]string{r.alice: "alice@example.com", r.bob: "bob@example.com"} {
		if err := errors.Join(users.CreateUser(context.Background(), identityapp.NewUser{ID: id, Email: email, PasswordHash: "x", DisplayName: "x", Now: now}),
			users.CreateDefaultProfile(context.Background(), uuid.NewV7(), id, now)); err != nil {
			t.Fatal(err)
		}
	}
	store := workspacepg.New(pool)
	if _, err := store.CreateWorkspace(context.Background(), workspaceapp.WorkspaceRow{ID: r.acme, Name: "Acme", Slug: "acme", Timezone: "UTC",
		CreatedBy: r.alice, Now: now}); err != nil {
		t.Fatal(err)
	}
	r.join(t, r.alice, shared.RoleAdmin)
	id := uuid.NewV7()
	if _, err := store.CreateInvitations(context.Background(), []workspaceapp.InvitationRow{
		{ID: id, WorkspaceID: r.acme, Email: "bob@example.com", Role: shared.RoleMember, CreatedBy: r.alice, Now: now},
	}); err != nil {
		t.Fatal(err)
	}
	r.invitation = invitationLink{id, workspacedomain.FormatToken(r.mac.Tag(workspacedomain.InvitationMessage(id)))}
	return r
}

// testInvitationMAC is the invitation MAC of the tests' signing key, as
// bootstrap derives it.
func testInvitationMAC(t *testing.T) workspaceapp.InvitationMAC {
	t.Helper()
	keys, err := identity.LoadKeys([]byte(testKeyPEM), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	mac, err := keys.MAC(workspace.InvitationMACPurpose)
	if err != nil {
		t.Fatal(err)
	}
	return mac
}

// join makes user a member of acme with role.
func (r answerRace) join(t *testing.T, user uuid.UUID, role shared.Role) {
	t.Helper()
	if err := workspacepg.New(r.pool).CreateMember(context.Background(), workspaceapp.MemberRow{
		ID: uuid.NewV7(), WorkspaceID: r.acme, MemberID: user, Role: role, CreatedBy: r.alice, Now: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}
}

func (r answerRace) tx() *postgres.TxManager { return postgres.NewTxManager(r.pool, 2*time.Second) }

func (r answerRace) accounts() workspaceapp.Accounts {
	return workspaceAccounts{accounts: identity.Provide(r.pool).Accounts}
}

// accept is bob's acceptance of his invitation, over invitations.
func (r answerRace) accept(ctx context.Context, invitations workspaceapp.InvitationAccepter) error {
	_, err := workspaceapp.NewAcceptWorkspaceInvitation(workspaceapp.AcceptInvitationDeps{
		Accounts: r.accounts(), Invitations: invitations, Tx: r.tx(), Clock: clocktest.At(time.Now()), MAC: r.mac,
	}).Execute(shared.WithActor(ctx, shared.Actor{UserID: r.bob}), r.invitation.id, r.invitation.token)
	return err
}

// decline is bob's answer no, over invitations.
func (r answerRace) decline(ctx context.Context, invitations workspaceapp.InvitationDecliner) error {
	return workspaceapp.NewDeclineWorkspaceInvitation(r.accounts(), invitations, r.tx(), clocktest.At(time.Now()), r.mac).
		Execute(shared.WithActor(ctx, shared.Actor{UserID: r.bob}), r.invitation.id, r.invitation.token)
}

// deleteAcme is alice's deletion of acme, over workspaces.
func (r answerRace) deleteAcme(ctx context.Context, workspaces workspaceapp.WorkspaceDeleter) error {
	return workspaceapp.NewDeleteWorkspace(workspaces, access.New(access.Deps{WorkspaceRoles: workspace.Provide(r.pool).WorkspaceRoles}),
		r.tx(), clocktest.At(time.Now()), slog.New(slog.DiscardHandler)).Execute(shared.WithActor(ctx, shared.Actor{UserID: r.alice}), "acme")
}

// bobIn is bob's membership of acme as it stands: whether there is an
// undeleted one, and whether it is active; and whether acme is deleted.
func (r answerRace) bobIn(t *testing.T) (member, active, acmeDeleted bool) {
	t.Helper()
	if err := r.pool.QueryRow(context.Background(), `SELECT
		EXISTS (SELECT 1 FROM workspace_members WHERE workspace_id = $1 AND member_id = $2 AND deleted_at IS NULL),
		EXISTS (SELECT 1 FROM workspace_members WHERE workspace_id = $1 AND member_id = $2 AND deleted_at IS NULL AND is_active),
		(SELECT deleted_at IS NOT NULL FROM workspaces WHERE id = $1)`, r.acme, r.bob).Scan(&member, &active, &acmeDeleted); err != nil {
		t.Fatal(err)
	}
	return member, active, acmeDeleted
}

// answered is the invitation's state: accepted, answered, deleted.
func (r answerRace) answered(t *testing.T) (accepted, responded, deleted bool) {
	t.Helper()
	if err := r.pool.QueryRow(context.Background(), `SELECT accepted, responded_at IS NOT NULL, deleted_at IS NOT NULL
		FROM workspace_member_invites WHERE id = $1`, r.invitation.id).Scan(&accepted, &responded, &deleted); err != nil {
		t.Fatal(err)
	}
	return accepted, responded, deleted
}

// gatedAccepter stops an acceptance after its locks, before it reads the
// membership.
type gatedAccepter struct {
	*workspacepg.Store
	gate *gate
}

func (a gatedAccepter) MemberOf(ctx context.Context, workspaceID, userID uuid.UUID) (workspacedomain.Membership, bool, error) {
	if err := a.gate.wait(ctx); err != nil {
		return workspacedomain.Membership{}, false, err
	}
	return a.Store.MemberOf(ctx, workspaceID, userID)
}

// gatedDeleter stops a deletion after its lock and decision, before its
// first write.
type gatedDeleter struct {
	*workspacepg.Store
	gate *gate
}

func (d gatedDeleter) DeleteWorkspace(ctx context.Context, id, by uuid.UUID, now time.Time) error {
	if err := d.gate.wait(ctx); err != nil {
		return err
	}
	return d.Store.DeleteWorkspace(ctx, id, by, now)
}

// Interleaving 3: the acceptance first holds acme's row; the deletion waits
// on it, then deletes acme with bob's new membership. The deletion first
// holds it; the acceptance waits on it, then finds no workspace: 404, and
// bob is no member.
func TestAcceptingAndDeletingTheWorkspace(t *testing.T) {
	for _, acceptFirst := range []bool{true, false} {
		name := map[bool]string{true: "the acceptance first", false: "the deletion first"}[acceptFirst]
		t.Run(name, func(t *testing.T) {
			r := newAnswerRace(t)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			g := newGate()
			store := workspacepg.New(r.pool)
			var accepted, deleted <-chan error
			if acceptFirst {
				accepted = run(func() error { return r.accept(ctx, gatedAccepter{store, g}) })
				held(t, ctx, g, accepted, "the acceptance")
				deleted = run(func() error { return r.deleteAcme(ctx, store) })
			} else {
				deleted = run(func() error { return r.deleteAcme(ctx, gatedDeleter{store, g}) })
				held(t, ctx, g, deleted, "the deletion")
				accepted = run(func() error { return r.accept(ctx, store) })
			}
			pgtest.WaitForLockWaitOn(t, r.pool, "workspaces", 5*time.Second)
			close(g.open)

			wantAccept := error(nil)
			if !acceptFirst {
				wantAccept = workspacedomain.ErrInvitationNotFound
			}
			if err := result(t, ctx, accepted, "the acceptance"); !errors.Is(err, wantAccept) || (wantAccept == nil && err != nil) {
				t.Errorf("the acceptance = %v, want %v", err, wantAccept)
			}
			if err := result(t, ctx, deleted, "the deletion"); err != nil {
				t.Errorf("the deletion = %v, want it done", err)
			}
			var gone int
			if err := r.pool.QueryRow(context.Background(), `SELECT count(*) FROM workspace_members m JOIN workspaces w ON w.id = m.workspace_id
				WHERE m.member_id = $1 AND m.deleted_at = w.deleted_at`, r.bob).Scan(&gone); err != nil {
				t.Fatal(err)
			}
			wantGone := map[bool]int{true: 1, false: 0}[acceptFirst]
			if member, _, acmeDeleted := r.bobIn(t); member || !acmeDeleted || gone != wantGone {
				t.Errorf("bob a member %v, acme deleted %v, bob's memberships deleted with acme %d; want none left, acme deleted, %d",
					member, acmeDeleted, gone, wantGone)
			}
			if acc, _, deleted := r.answered(t); acc != acceptFirst || !deleted {
				t.Errorf("the invitation accepted %v, deleted %v; want accepted %v, deleted", acc, deleted, acceptFirst)
			}
		})
	}
}

// gatedEmails stops a change of address after its write, holding the
// account row, before it revokes the sessions.
type gatedEmails struct {
	identityapp.SessionRevoker
	gate *gate
}

func (s gatedEmails) RevokeSessions(ctx context.Context, userID, keep uuid.UUID, reason identitydomain.RevokeReason, now time.Time) (int, error) {
	if err := s.gate.wait(ctx); err != nil {
		return 0, err
	}
	return s.SessionRevoker.RevokeSessions(ctx, userID, keep, reason, now)
}

// setBobsEmail is `nerve users set-email` of bob's address, over sessions.
func (r answerRace) setBobsEmail(ctx context.Context, sessions identityapp.SessionRevoker) error {
	store := identitypg.New(r.pool)
	_, err := identityapp.NewSetEmail(identityapp.SetEmailDeps{Accounts: store, Users: store, Sessions: sessions, Tx: r.tx(),
		Clock: clocktest.At(time.Now()), Logger: slog.New(slog.DiscardHandler)}).Execute(ctx, "bob@example.com", "robert@example.com")
	return err
}

// Interleaving 12: the change of bob's address first holds his account row;
// the acceptance waits on it, then reads the new address under its lock:
// 403 workspace.invitation_email_mismatch, and nothing changes. The
// acceptance first holds the row FOR SHARE; the change waits, the
// acceptance makes bob a member by his address as it was, then the change
// goes on.
func TestAcceptingAndChangingTheAddress(t *testing.T) {
	for _, acceptFirst := range []bool{true, false} {
		name := map[bool]string{true: "the acceptance first", false: "the change first"}[acceptFirst]
		t.Run(name, func(t *testing.T) {
			r := newAnswerRace(t)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			g := newGate()
			store, users := workspacepg.New(r.pool), identitypg.New(r.pool)
			var accepted, changed <-chan error
			if acceptFirst {
				accepted = run(func() error { return r.accept(ctx, gatedAccepter{store, g}) })
				held(t, ctx, g, accepted, "the acceptance")
				changed = run(func() error { return r.setBobsEmail(ctx, users) })
			} else {
				changed = run(func() error { return r.setBobsEmail(ctx, gatedEmails{users, g}) })
				held(t, ctx, g, changed, "the change")
				accepted = run(func() error { return r.accept(ctx, store) })
			}
			pgtest.WaitForLockWaitOn(t, r.pool, "users", 5*time.Second)
			close(g.open)

			wantAccept := error(nil)
			if !acceptFirst {
				wantAccept = workspacedomain.ErrInvitationEmailMismatch
			}
			if err := result(t, ctx, accepted, "the acceptance"); !errors.Is(err, wantAccept) || (wantAccept == nil && err != nil) {
				t.Errorf("the acceptance = %v, want %v", err, wantAccept)
			}
			if err := result(t, ctx, changed, "the change"); err != nil {
				t.Errorf("the change = %v, want it done", err)
			}
			member, _, _ := r.bobIn(t)
			acc, responded, deleted := r.answered(t)
			if member != acceptFirst || acc != acceptFirst || responded != acceptFirst || deleted != acceptFirst {
				t.Errorf("bob a member %v; the invitation accepted %v, answered %v, deleted %v; want all %v", member, acc, responded, deleted, acceptFirst)
			}
		})
	}
}

// gatedDeactivation takes, before it revokes the sessions, the lock of acme
// that P6's deactivation takes (M3 design 3.9: the account row, then each
// workspace FOR NO KEY UPDATE), and stops there when it has a gate.
type gatedDeactivation struct {
	identityapp.SessionRevoker
	pool *pgxpool.Pool
	acme uuid.UUID
	gate *gate // nil: never stops
}

func (d gatedDeactivation) RevokeSessions(ctx context.Context, userID, keep uuid.UUID, reason identitydomain.RevokeReason, now time.Time) (int, error) {
	if _, err := postgres.DB(ctx, d.pool).Exec(ctx, "SELECT id FROM workspaces WHERE id = $1 FOR NO KEY UPDATE", d.acme); err != nil {
		return 0, err
	}
	if d.gate != nil {
		if err := d.gate.wait(ctx); err != nil {
			return 0, err
		}
	}
	return d.SessionRevoker.RevokeSessions(ctx, userID, keep, reason, now)
}

// deactivateBob is `nerve users deactivate` of bob, over sessions.
func (r answerRace) deactivateBob(ctx context.Context, sessions identityapp.SessionRevoker) error {
	store := identitypg.New(r.pool)
	_, err := identityapp.NewDeactivate(identityapp.DeactivateDeps{Accounts: store, Users: store, Profiles: store, Sessions: sessions, Tx: r.tx(),
		Clock: clocktest.At(time.Now()), Logger: slog.New(slog.DiscardHandler)}).ExecuteByEmail(ctx, "bob@example.com")
	return err
}

// gatedDecliner stops a decline after its locks, before its write.
type gatedDecliner struct {
	*workspacepg.Store
	gate *gate
}

func (d gatedDecliner) DeclineInvitation(ctx context.Context, id, by uuid.UUID, now time.Time) error {
	if err := d.gate.wait(ctx); err != nil {
		return err
	}
	return d.Store.DeclineInvitation(ctx, id, by, now)
}

// Interleaving 19: bob is acme's member and has an invitation to it, as
// after reactivate-member or a change of address. The decline first holds
// his account row FOR SHARE, then acme's; the deactivation waits on the
// account row, then goes on through acme. The deactivation first holds the
// account row and acme's; the decline waits on the account row, then reads
// it deactivated under its lock: 401, and the invitation stays pending.
// Neither deadlocks: both lock the account row first (3.6 convention 1).
func TestDecliningAndDeactivating(t *testing.T) {
	for _, declineFirst := range []bool{true, false} {
		name := map[bool]string{true: "the decline first", false: "the deactivation first"}[declineFirst]
		t.Run(name, func(t *testing.T) {
			r := newAnswerRace(t)
			r.join(t, r.bob, shared.RoleMember)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			g := newGate()
			store, users := workspacepg.New(r.pool), identitypg.New(r.pool)
			var declined, deactivated <-chan error
			if declineFirst {
				declined = run(func() error { return r.decline(ctx, gatedDecliner{store, g}) })
				held(t, ctx, g, declined, "the decline")
				deactivated = run(func() error { return r.deactivateBob(ctx, gatedDeactivation{users, r.pool, r.acme, nil}) })
			} else {
				deactivated = run(func() error { return r.deactivateBob(ctx, gatedDeactivation{users, r.pool, r.acme, g}) })
				held(t, ctx, g, deactivated, "the deactivation")
				declined = run(func() error { return r.decline(ctx, store) })
			}
			pgtest.WaitForLockWaitOn(t, r.pool, "users", 5*time.Second)
			close(g.open)

			wantDecline := error(nil)
			if !declineFirst {
				wantDecline = shared.Unauthenticated()
			}
			if err := result(t, ctx, declined, "the decline"); !errors.Is(err, wantDecline) || (wantDecline == nil && err != nil) {
				t.Errorf("the decline = %v, want %v", err, wantDecline)
			}
			if err := result(t, ctx, deactivated, "the deactivation"); err != nil {
				t.Errorf("the deactivation = %v, want it done", err)
			}
			var active bool
			if err := r.pool.QueryRow(context.Background(), "SELECT is_active FROM users WHERE id = $1", r.bob).Scan(&active); err != nil {
				t.Fatal(err)
			}
			acc, responded, deleted := r.answered(t)
			if active || acc || responded != declineFirst || deleted {
				t.Errorf("bob active %v; the invitation accepted %v, answered %v, deleted %v; want deactivated, answered %v, undeleted",
					active, acc, responded, deleted, declineFirst)
			}
		})
	}
}
````

- [ ] **Step 4: 交错 9、18**

`server/internal/bootstrap/interleaving_invite_test.go`（新文件，262 行）：

````file server/internal/bootstrap/interleaving_invite_test.go
package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/access"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	identitypg "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	identitydomain "github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	workspacedomain "github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Interleavings 9 and 18 of M3 design 9.3: creating invitations against
// the administrator's reset of the inviter's password, and two admins
// inviting overlapping batches; in both orders, on a real database, through
// the real use cases and identity's credential lock as bootstrap wires
// them. The reset and the creation wait on the inviter's account row
// (WaitForLockWaitOn "users"): the only row both lock. The later batch
// waits on no row: it inserts a key the earlier one has inserted and waits
// for that transaction to end, which WaitForKeyWaitOn sees and nothing else
// in these tests does; its transaction writes no other table and upgrades
// no row lock.

// inviteRace is answerRace with carol, acme's second admin, and a live
// session of each admin, which the credential lock checks.
type inviteRace struct {
	answerRace
	carol                      uuid.UUID
	aliceSession, carolSession uuid.UUID
}

func newInviteRace(t *testing.T) inviteRace {
	t.Helper()
	r := inviteRace{answerRace: newAnswerRace(t), carol: uuid.NewV7(), aliceSession: uuid.NewV7(), carolSession: uuid.NewV7()}
	now := time.Now()
	users := identitypg.New(r.pool)
	if err := errors.Join(
		users.CreateUser(context.Background(), identityapp.NewUser{ID: r.carol, Email: "carol@example.com", PasswordHash: "x", DisplayName: "carol", Now: now}),
		users.CreateDefaultProfile(context.Background(), uuid.NewV7(), r.carol, now),
		users.CreateSession(context.Background(), identityapp.NewSession{ID: r.aliceSession, UserID: r.alice, TokenHash: make([]byte, 32),
			ExpiresAt: now.Add(time.Hour), Now: now}),
		users.CreateSession(context.Background(), identityapp.NewSession{ID: r.carolSession, UserID: r.carol, TokenHash: make([]byte, 32),
			ExpiresAt: now.Add(time.Hour), Now: now}),
	); err != nil {
		t.Fatal(err)
	}
	r.join(t, r.carol, shared.RoleAdmin)
	return r
}

// invite is the admin's creation of batch in acme, as his session, over
// invitations.
func (r inviteRace) invite(ctx context.Context, admin, session uuid.UUID, invitations workspaceapp.InvitationCreator, emails ...string) error {
	var batch []workspacedomain.NewInvitation
	for _, email := range emails {
		batch = append(batch, workspacedomain.NewInvitation{Email: email, Role: shared.RoleMember})
	}
	_, err := workspaceapp.NewCreateWorkspaceInvitations(workspaceapp.CreateInvitationsDeps{
		Caller: identity.Provide(r.pool).CredentialLock, Invitations: invitations, Profiles: workspaceProfiles{profiles: identity.Provide(r.pool).PublicProfiles},
		Auth: access.New(access.Deps{WorkspaceRoles: workspace.Provide(r.pool).WorkspaceRoles}), Tx: r.tx(), Clock: clocktest.At(time.Now()), MAC: r.mac,
	}).Execute(shared.WithActor(ctx, shared.Actor{UserID: admin, SessionID: session}), "acme", batch)
	return err
}

// invited are the addresses of acme's undeleted invitations but bob's,
// sorted, each with the admin who sent it.
func (r inviteRace) invited(t *testing.T) []string {
	t.Helper()
	rows, err := r.pool.Query(context.Background(), `SELECT i.email || ' by ' || u.email FROM workspace_member_invites i
		JOIN users u ON u.id = i.created_by_id WHERE i.workspace_id = $1 AND i.deleted_at IS NULL AND i.email <> 'bob@example.com'`, r.acme)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		got = append(got, s)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	slices.Sort(got)
	return got
}

// gatedInviter stops a creation after its inserts, holding the keys it
// inserted, the workspace's FOR SHARE and the inviter's account row.
type gatedInviter struct {
	*workspacepg.Store
	gate *gate
}

func (i gatedInviter) CreateInvitations(ctx context.Context, rows []workspaceapp.InvitationRow) ([]workspacedomain.Invitation, error) {
	created, err := i.Store.CreateInvitations(ctx, rows)
	if err != nil {
		return nil, err
	}
	return created, i.gate.wait(ctx)
}

// fixedHasher hashes password as "hashed:<password>".
type fixedHasher struct{}

func (fixedHasher) Hash(_ context.Context, password string) (string, error) {
	return "hashed:" + password, nil
}

func (fixedHasher) Verify(_ context.Context, password, hash string) (bool, bool, error) {
	return hash == "hashed:"+password, false, nil
}

// gatedPasswords stops a reset before it writes the hash, holding the
// account row.
type gatedPasswords struct {
	identityapp.PasswordHashWriter
	gate *gate
}

func (p gatedPasswords) UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string, now time.Time) error {
	if err := p.gate.wait(ctx); err != nil {
		return err
	}
	return p.PasswordHashWriter.UpdatePasswordHash(ctx, id, hash, now)
}

// resetAlice is `nerve users reset-password` of alice, over passwords.
func (r inviteRace) resetAlice(ctx context.Context, passwords identityapp.PasswordHashWriter) error {
	store := identitypg.New(r.pool)
	_, err := identityapp.NewResetPassword(identityapp.ResetPasswordDeps{Accounts: store, Passwords: passwords, Sessions: store, APITokens: store,
		Hasher: fixedHasher{}, Rules: identitydomain.NewPasswordRules(), Tx: r.tx(), Clock: clocktest.At(time.Now()),
		Logger: slog.New(slog.DiscardHandler)}).Execute(ctx, "alice@example.com", "N3w-Passw0rd!")
	return err
}

// Interleaving 9: the reset first holds alice's account row; her creation
// waits on it, then finds her session revoked under the lock: 401, and no
// invitation. The creation first holds the row; the reset waits, and the
// invitation stays: it belongs to acme, not to her credential (3.8).
func TestInvitingAndResettingThePassword(t *testing.T) {
	for _, inviteFirst := range []bool{true, false} {
		name := map[bool]string{true: "the creation first", false: "the reset first"}[inviteFirst]
		t.Run(name, func(t *testing.T) {
			r := newInviteRace(t)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			g := newGate()
			store := workspacepg.New(r.pool)
			var invited, reset <-chan error
			if inviteFirst {
				invited = run(func() error {
					return r.invite(ctx, r.alice, r.aliceSession, gatedInviter{store, g}, "dave@example.com")
				})
				held(t, ctx, g, invited, "the creation")
				reset = run(func() error { return r.resetAlice(ctx, identitypg.New(r.pool)) })
			} else {
				reset = run(func() error { return r.resetAlice(ctx, gatedPasswords{identitypg.New(r.pool), g}) })
				held(t, ctx, g, reset, "the reset")
				invited = run(func() error { return r.invite(ctx, r.alice, r.aliceSession, store, "dave@example.com") })
			}
			pgtest.WaitForLockWaitOn(t, r.pool, "users", 5*time.Second)
			close(g.open)

			wantInvite := error(nil)
			if !inviteFirst {
				wantInvite = shared.Unauthenticated()
			}
			if err := result(t, ctx, invited, "the creation"); !errors.Is(err, wantInvite) || (wantInvite == nil && err != nil) {
				t.Errorf("the creation = %v, want %v", err, wantInvite)
			}
			if err := result(t, ctx, reset, "the reset"); err != nil {
				t.Errorf("the reset = %v, want it done", err)
			}
			var want []string
			if inviteFirst {
				want = []string{"dave@example.com by alice@example.com"}
			}
			if got := r.invited(t); !slices.Equal(got, want) {
				t.Errorf("acme's invitations: %q, want %q", got, want)
			}
		})
	}
}

// Interleaving 18: alice invites [x, y] and carol [y, x], each under acme's
// FOR SHARE, inserting in the order of the addresses. The later waits on
// the key the earlier inserted, not on a row, and once the earlier commits
// is refused 422 duplicate on the address's index in its own request, with
// nothing inserted: no deadlock (40P01), whichever goes first. The same for
// one address in both.
func TestInvitingOverlappingBatches(t *testing.T) {
	x, y := "xavier@example.com", "yvonne@example.com"
	for _, tt := range []struct {
		name         string
		aliceFirst   bool
		alice, carol []string
		field        string // the later one's
	}{
		{"alice first", true, []string{x, y}, []string{y, x}, "invitations[1].email"},
		{"carol first", false, []string{x, y}, []string{y, x}, "invitations[0].email"},
		{"one address, alice first", true, []string{x}, []string{x}, "invitations[0].email"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := newInviteRace(t)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			g := newGate()
			store := workspacepg.New(r.pool)
			first, second := func(inv workspaceapp.InvitationCreator) error {
				return r.invite(ctx, r.alice, r.aliceSession, inv, tt.alice...)
			}, func(inv workspaceapp.InvitationCreator) error {
				return r.invite(ctx, r.carol, r.carolSession, inv, tt.carol...)
			}
			winner := "alice@example.com"
			if !tt.aliceFirst {
				first, second, winner = second, first, "carol@example.com"
			}
			earlier := run(func() error { return first(gatedInviter{store, g}) })
			held(t, ctx, g, earlier, "the earlier batch")
			later := run(func() error { return second(store) })
			pgtest.WaitForKeyWaitOn(t, r.pool, "workspace_member_invites", 5*time.Second)
			close(g.open)

			if err := result(t, ctx, earlier, "the earlier batch"); err != nil {
				t.Errorf("the earlier batch = %v, want it done", err)
			}
			err := result(t, ctx, later, "the later batch")
			var se *shared.Error
			if !errors.As(err, &se) || se.Code != shared.CodeValidationFailed || len(se.Fields) != 1 || se.Fields[0].Field != tt.field ||
				se.Fields[0].Code != shared.FieldDuplicate {
				t.Errorf("the later batch = %v, want 422 duplicate on %s alone", err, tt.field)
			}
			batch := tt.alice
			if !tt.aliceFirst {
				batch = tt.carol
			}
			var want []string
			for _, email := range batch {
				want = append(want, email+" by "+winner)
			}
			slices.Sort(want)
			if got := r.invited(t); !slices.Equal(got, want) {
				t.Errorf("acme's invitations: %q, want %q: the earlier batch's, none of the later", got, want)
			}
		})
	}
}
````

- [ ] **Step 5: 测试、lint**

Run: `go -C server test -count=1 ./internal/platform/postgres/pgtest/`
Expected: `ok`。

Run: `go -C server test -count=5 -race -run 'TestAcceptingAndDeletingTheWorkspace|TestInvitingAndResettingThePassword|TestAcceptingAndChangingTheAddress|TestInvitingOverlappingBatches|TestDecliningAndDeactivating|TestTwoAdminsDemotingEachOtherLeaveAnAdmin' ./internal/bootstrap/`
Expected: `ok`（约 10 秒）；输出里没有 `40P01`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 6: 提交**

```bash
git add server/internal/bootstrap/interleaving_answers_test.go server/internal/bootstrap/interleaving_invite_test.go server/internal/bootstrap/interleaving_test.go server/internal/platform/postgres/pgtest/lockwait.go server/internal/platform/postgres/pgtest/lockwait_test.go
```
```bash
git commit -m "test(M3/P3): interleavings 3, 9, 12, 18 and 19

The five interleavings of M3 design 9.3 that P3 brings, both orders, on
a real database through the use cases as bootstrap wires them: accepting
against deleting the workspace, inviting against a password reset,
accepting against a change of address, overlapping batches, declining
against a deactivation. pgtest.WaitForKeyWaitOn sees the one wait no row
lock shows, an insert waiting on the key another transaction inserted;
the gates of the interleavings fail at a deadline instead of hanging.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**变异**（spec 附录 A）：

| 改坏 | 必须失败的测试 |
|---|---|
| 探针数任何表的键等待；数行的等待；数任何等待；数只读过表的事务 | `TestWaitForKeyWaitOnSeesOnlyAKeysWaitOnItsTable` |
| 接受不锁工作区（交错 3 没有锁） | `TestAcceptingAndDeletingTheWorkspace` |
| 接受的锁在事务之前 | `TestAcceptingAndDeletingTheWorkspace`、`TestAcceptingAndChangingTheAddress` |
| 账户在事务之外读（不持锁）；读账户不加 `FOR SHARE` | `TestAcceptingAndChangingTheAddress`、`TestDecliningAndDeactivating` |
| 账户在工作区和邀请之后锁 | `TestDecliningAndDeactivating` |
| 忽略的锁在事务之前 | `TestDecliningAndDeactivating` |
| 创建不锁邀请人的账户；在事务之外锁 | `TestInvitingAndResettingThePassword` |
| 按请求的顺序插入；工作区用 N；23505 答 409；下标取排序后的位置；存储跳过被占的邮箱 | `TestInvitingOverlappingBatches` |
| 插入在事务之外 | `TestInvitingOverlappingBatches`（闸门的期限之内失败，不挂住） |

**Done when:** 五个交错两个顺序在真实数据库上 `-count=5 -race` 通过，没有 40P01；探针的正例、三个反例通过；上表的变异都让它们失败，最慢的在 35 秒之内。

---

### Task 14: 端到端：W3–W6 的接口版本，W8 的第二个成员

**Files:**
- Create: `e2e/stories/workspace/w3-workspace-settings.spec.ts`、`e2e/stories/workspace/w4-invite-members.spec.ts`、`e2e/stories/workspace/w5-invitation-link.spec.ts`、`e2e/stories/workspace/w6-sign-up-by-invitation.spec.ts`
- Modify: `e2e/fixtures/api.ts`、`e2e/fixtures/assert/workspace.ts`、`e2e/stories/workspace/w8-navigation-preferences.spec.ts`

**Interfaces:**
- Produces（spec 2.15，M3 设计 2、9.6）：`e2e/fixtures/api.ts`：`invite(api, token, slug, invitations)`、`accept(api, token, invitation)`、`inviteAndAccept(api, adminToken, slug, {email, token}, role)`，类型 `WorkspaceInvitation`、`InvitationCreate`；`e2e/fixtures/assert/workspace.ts`：`InvitationRow`、`expectInvitations(db, slug, inviterEmail, want)`（按 `COLLATE "C"` 排序比较；表的列恰好是模块内 `invitationColumns` 的 11 列；接受的 `deleted_at` 等于 `responded_at`；没有一行含令牌）、`expectMembership(db, slug, email, want | null)`、`expectWorkspaceDeleted(db, slug, adminEmail)`（模块内 `workspaceTables` 的每张表：至少一行随工作区在同一时刻删除，没有未删除的，也没有更晚删除的）。
- 使用者：P4 起凡是要第二个成员的故事；P9 的页面版本调用同一组断言。

**Tests:**
- `W3 (API): the admin changes the workspace and deletes it with its members, invitations and settings at one moment; a member may do neither, and the slug never changes`。
- `W4 (API): the admin invites a batch, changes a role and deletes an invitation; an active member's address, a repeated or declined one is refused; members and guests may not invite`。
- `W5 (API): the link shows the workspace and the role without the address; the invitee accepts or declines; another address, a wrong token and an answered invitation change nothing`。
- `W6 (API): with sign-up off, the address an invitation was sent to registers with its link and then accepts it; without a link, with a wrong token, another address, a declined or a deleted invitation it may not`（全部经关闭注册的第二个 nerve，9.6）。
- `W8 (API)` 加一个经 `inviteAndAccept` 加入的成员：他读到默认值、库里没有他的行。
- 此前的 53 个故事不改而通过（共 57 个）。

- [ ] **Step 1: fixture 和断言**

`e2e/fixtures/api.ts`（修改，2 处）：

````old e2e/fixtures/api.ts
export type WorkspacePreferencesUpdate = components["schemas"]["WorkspacePreferencesUpdate"];
````

````new e2e/fixtures/api.ts
export type WorkspacePreferencesUpdate = components["schemas"]["WorkspacePreferencesUpdate"];
export type WorkspaceInvitation = components["schemas"]["WorkspaceInvitation"];
export type InvitationCreate = components["schemas"]["InvitationCreate"];
````

````old e2e/fixtures/api.ts
  return data;
}

````

````new e2e/fixtures/api.ts
  return data;
}

/** Invites the addresses of invitations to the workspace of slug with the bearer token given, an admin's, and returns the invitations in that order. */
export async function invite(
  api: Api,
  token: string,
  slug: string,
  invitations: InvitationCreate[]
): Promise<WorkspaceInvitation[]> {
  const { data, error, response } = await api.POST("/api/v0/workspaces/{slug}/invitations", {
    params: { path: { slug } },
    body: { invitations },
    headers: bearer(token),
  });
  expect(response.status, `invite to ${slug}: ${JSON.stringify(error)}`).toBe(201);
  if (!data) {
    throw new Error(`invite to ${slug} answered 201 without the invitations`);
  }
  return data.data;
}

/** Accepts invitation with the bearer token given, the invitee's, and returns the workspace as the new member reads it. */
export async function accept(api: Api, token: string, invitation: WorkspaceInvitation): Promise<Workspace> {
  const { data, error, response } = await api.POST("/api/v0/workspace-invitations/{invitation_id}/accept", {
    params: { path: { invitation_id: invitation.id } },
    body: { token: invitation.token },
    headers: bearer(token),
  });
  expect(response.status, `accept the invitation of ${invitation.email}: ${JSON.stringify(error)}`).toBe(200);
  if (!data) {
    throw new Error(`accept answered 200 without the workspace`);
  }
  return data;
}

/**
 * Makes the account of email, whose bearer token is memberToken, a member of the workspace of slug with role:
 * its admin, with adminToken, invites the address and the account accepts. The one way a workspace gets a
 * second member (M3 design 12 constraint 1). Returns the workspace as the new member reads it.
 */
export async function inviteAndAccept(
  api: Api,
  adminToken: string,
  slug: string,
  member: { email: string; token: string },
  role: InvitationCreate["role"]
): Promise<Workspace> {
  const [invitation] = await invite(api, adminToken, slug, [{ email: member.email, role }]);
  if (!invitation) {
    throw new Error(`invite ${member.email} to ${slug} answered no invitation`);
  }
  return accept(api, member.token, invitation);
}

````

`e2e/fixtures/assert/workspace.ts`（修改，1 处）：

````old e2e/fixtures/assert/workspace.ts
  return rows[0]?.id ?? null;
}

````

````new e2e/fixtures/assert/workspace.ts
  return rows[0]?.id ?? null;
}

/** An invitation as a story expects it in the database. */
export interface InvitationRow {
  email: string;
  role: number;
  accepted: boolean;
  responded: boolean;
  deleted: boolean;
}

/** The columns of workspace_member_invites (M3 design 4.4): no token among them, the server keeps none (3.8). */
const invitationColumns = [
  "accepted",
  "created_at",
  "created_by_id",
  "deleted_at",
  "email",
  "id",
  "responded_at",
  "role",
  "updated_at",
  "updated_by_id",
  "workspace_id",
];

/**
 * W4, W5, W6: the invitations of the workspace of slug, deleted ones too, are want, in any order. The table
 * holds no token: its rows have the columns of the design, and none holds any of tokens. Every one was made by
 * the account of inviterEmail; an acceptance deletes the invitation at the moment of the answer.
 */
export async function expectInvitations(
  db: Database,
  slug: string,
  inviterEmail: string,
  want: InvitationRow[],
  tokens: string[] = []
): Promise<void> {
  const rows = await db.query<
    Record<string, unknown> & { email: string; accepted: boolean; responded_at: Date | null; deleted_at: Date | null }
  >(
    `SELECT i.* FROM workspace_member_invites i JOIN workspaces w ON w.id = i.workspace_id
      WHERE w.slug = $1 ORDER BY i.email COLLATE "C"`,
    [slug]
  );
  for (const row of rows) {
    expect(Object.keys(row).toSorted(), "the columns of workspace_member_invites").toEqual(invitationColumns);
    const values = new Set(Object.values(row).map(String));
    expect(
      tokens.filter((token) => values.has(token)),
      `the invitation of ${row.email} holds no token`
    ).toEqual([]);
  }
  expect(
    rows.map((r) => ({
      email: r.email,
      role: r.role,
      accepted: r.accepted,
      responded: r.responded_at !== null,
      deleted: r.deleted_at !== null,
    })),
    `the invitations of ${slug}`
  ).toEqual(want.toSorted((a, b) => (a.email < b.email ? -1 : 1)));
  const [inviter] = await db.query<{ id: string }>("SELECT id FROM users WHERE email = $1", [inviterEmail]);
  expect(
    rows.map((r) => r.created_by_id),
    `the invitations of ${slug}, made by ${inviterEmail}`
  ).toEqual(rows.map(() => inviter?.id));
  const accepted = rows.filter((r) => r.accepted);
  expect(
    accepted.map((r) => r.deleted_at?.getTime()),
    `the accepted invitations of ${slug}, deleted when answered`
  ).toEqual(accepted.map((r) => r.responded_at?.getTime()));
}

/**
 * W5, W6: the membership of the account of email in the workspace of slug: none while want is null, else one,
 * undeleted, holding want.
 */
export async function expectMembership(
  db: Database,
  slug: string,
  email: string,
  want: { role: number; is_active: boolean } | null
): Promise<void> {
  const rows = await db.query(
    `SELECT m.role, m.is_active FROM workspace_members m JOIN workspaces w ON w.id = m.workspace_id JOIN users u ON u.id = m.member_id
      WHERE w.slug = $1 AND u.email = $2 AND m.deleted_at IS NULL`,
    [slug, email]
  );
  expect(rows, `the membership of ${email} in ${slug}`).toEqual(want === null ? [] : [want]);
}

/** The tables whose rows belong to a workspace and are deleted with it (M3 design 4.12); P4 adds the projects'. */
const workspaceTables = ["workspace_members", "workspace_member_invites", "workspace_user_properties"];

/**
 * W3: the workspace of slug is deleted by the account of adminEmail, and with it, at the same moment, every row
 * under it that was not deleted before: its memberships, invitations and display settings. Each table has
 * such a row; none is left undeleted.
 */
export async function expectWorkspaceDeleted(db: Database, slug: string, adminEmail: string): Promise<void> {
  const [w] = await db.query<{ id: string; deleted_at: Date | null; updated_by_id: string; admin: string }>(
    `SELECT w.id, w.deleted_at, w.updated_by_id, (SELECT id FROM users WHERE email = $2) AS admin FROM workspaces w WHERE w.slug = $1`,
    [slug, adminEmail]
  );
  expect(w?.deleted_at, `${slug} deleted`).toBeInstanceOf(Date);
  expect(w?.updated_by_id, `${slug} deleted by ${adminEmail}`).toBe(w?.admin);
  const at = w?.deleted_at?.getTime() ?? 0;
  const tables = await Promise.all(
    workspaceTables.map((table) =>
      db.query<{ deleted_at: Date | null }>(`SELECT deleted_at FROM ${table} WHERE workspace_id = $1`, [w?.id])
    )
  );
  expect(
    tables.map((rows, i) => ({
      table: workspaceTables[i],
      deletedWithIt: rows.some((r) => r.deleted_at?.getTime() === at),
      undeletedOrLater: rows.filter((r) => r.deleted_at === null || r.deleted_at.getTime() > at).length,
    })),
    `the rows under ${slug}`
  ).toEqual(workspaceTables.map((table) => ({ table, deletedWithIt: true, undeletedOrLater: 0 })));
}

````

- [ ] **Step 2: 故事**

`e2e/stories/workspace/w3-workspace-settings.spec.ts`（新文件，84 行）：

````file e2e/stories/workspace/w3-workspace-settings.spec.ts
import { createWorkspace, invite, inviteAndAccept, slugFor } from "../../fixtures/api";
import { expectWorkspaceDeleted } from "../../fixtures/assert/workspace";
import { bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// W3, the workspace's settings (M3 design 2). The page version, with the
// session switch of 7.1, comes with the general page (P9); P4 adds the
// projects to the deletion's assertions.

test("W3 (API): the admin changes the workspace and deletes it with its members, invitations and settings at one moment; a member may do neither, and the slug never changes", async ({
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
  await invite(api, admin, slug, [{ email: emailFor(testInfo, "invitee"), role: 5 }]);
  const settings = await api.PATCH("/api/v0/me/workspaces/{slug}/preferences", {
    params: { path: { slug } },
    body: { navigation_project_limit: 3 },
    headers: bearer(member),
  });
  expect(settings.response.status).toBe(200);

  const renamed = await api.PATCH("/api/v0/workspaces/{slug}", {
    params: { path: { slug } },
    body: { name: "Acme Corp", organization_size: "11-50", timezone: "Europe/Berlin" },
    headers: bearer(admin),
  });
  expect(renamed.response.status).toBe(200);
  expect(renamed.data).toMatchObject({
    slug,
    name: "Acme Corp",
    organization_size: "11-50",
    timezone: "Europe/Berlin",
    role: 20,
    total_members: 2,
  });
  expect(
    await db.query(
      `SELECT w.name, w.organization_size, w.timezone, w.updated_by_id = u.id AS updated_by_the_admin
         FROM workspaces w JOIN users u ON u.email = $2 WHERE w.slug = $1`,
      [slug, adminEmail]
    )
  ).toEqual([{ name: "Acme Corp", organization_size: "11-50", timezone: "Europe/Berlin", updated_by_the_admin: true }]);

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

  const deleted = await api.DELETE("/api/v0/workspaces/{slug}", { params: { path: { slug } }, headers: bearer(admin) });
  expect(deleted.response.status).toBe(204);
  await expectWorkspaceDeleted(db, slug, adminEmail);
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

`e2e/stories/workspace/w4-invite-members.spec.ts`（新文件，149 行）：

````file e2e/stories/workspace/w4-invite-members.spec.ts
import { createWorkspace, invite, inviteAndAccept, slugFor } from "../../fixtures/api";
import { expectInvitations } from "../../fixtures/assert/workspace";
import { bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// W4, invite members (M3 design 2, 3.8; decision 4: admins only). The page
// version comes with the members page (P9).

test("W4 (API): the admin invites a batch, changes a role and deletes an invitation; an active member's address, a repeated or declined one is refused; members and guests may not invite", async ({
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
  const accepted = [
    { email: memberEmail, role: 15, accepted: true, responded: true, deleted: true },
    { email: guestEmail, role: 5, accepted: true, responded: true, deleted: true },
  ];

  // A batch of two, the addresses as a person types them: stored normalized, pending, in the request's order.
  const carol = emailFor(testInfo, "carol");
  const dave = emailFor(testInfo, "dave");
  const created = await invite(api, admin, slug, [
    { email: ` ${carol.toUpperCase()} `, role: 15 },
    { email: dave, role: 5 },
  ]);
  expect(created.map((i) => [i.email, i.role, i.accepted, i.responded_at])).toEqual([
    [carol, 15, false, null],
    [dave, 5, false, null],
  ]);
  const tokens = created.map((i) => i.token);
  await expectInvitations(
    db,
    slug,
    adminEmail,
    [
      ...accepted,
      { email: carol, role: 15, accepted: false, responded: false, deleted: false },
      { email: dave, role: 5, accepted: false, responded: false, deleted: false },
    ],
    tokens
  );

  // One role changed, the other invitation deleted; the list shows what is left, with its link.
  const [carolsInvitation, davesInvitation] = created;
  const changed = await api.PATCH("/api/v0/workspace-invitations/{invitation_id}", {
    params: { path: { invitation_id: carolsInvitation?.id ?? "" } },
    body: { role: 20 },
    headers: bearer(admin),
  });
  expect(changed.response.status).toBe(200);
  expect(changed.data).toMatchObject({ email: carol, role: 20, token: carolsInvitation?.token });
  const deleted = await api.DELETE("/api/v0/workspace-invitations/{invitation_id}", {
    params: { path: { invitation_id: davesInvitation?.id ?? "" } },
    headers: bearer(admin),
  });
  expect(deleted.response.status).toBe(204);
  const listed = await api.GET("/api/v0/workspaces/{slug}/invitations", {
    params: { path: { slug } },
    headers: bearer(admin),
  });
  expect(listed.data?.data.map((i) => [i.email, i.role, i.token])).toEqual([[carol, 20, carolsInvitation?.token]]);

  // A declined invitation holds its address until it is deleted.
  const erinEmail = emailFor(testInfo, "erin");
  const erin = (await register(api, erinEmail)).access_token;
  const [erinsInvitation] = await invite(api, admin, slug, [{ email: erinEmail, role: 15 }]);
  const declined = await api.POST("/api/v0/workspace-invitations/{invitation_id}/decline", {
    params: { path: { invitation_id: erinsInvitation?.id ?? "" } },
    body: { token: erinsInvitation?.token ?? "" },
    headers: bearer(erin),
  });
  expect(declined.response.status).toBe(204);

  // Refused as a whole, each address at its index. A batch that repeats an address is refused as it is sent,
  // before the workspace is read; then an active member's address, a pending and a declined invitation's.
  const frank = emailFor(testInfo, "frank");
  const refuse = (invitations: { email: string; role: 5 | 15 }[]) =>
    api.POST("/api/v0/workspaces/{slug}/invitations", {
      params: { path: { slug } },
      body: { invitations },
      headers: bearer(admin),
    });
  const repeated = await refuse([
    { email: frank, role: 15 },
    { email: frank.toUpperCase(), role: 5 },
  ]);
  expect([repeated.response.status, repeated.error?.errors?.map((e) => [e.field, e.code])]).toEqual([
    422,
    [["invitations[1].email", "duplicate"]],
  ]);
  const taken = await refuse([
    { email: frank, role: 15 },
    { email: memberEmail, role: 15 },
    { email: carol, role: 5 },
    { email: erinEmail, role: 5 },
  ]);
  expect([taken.response.status, taken.error?.errors?.map((e) => [e.field, e.code])]).toEqual([
    422,
    [
      ["invitations[1].email", "not_allowed"],
      ["invitations[2].email", "duplicate"],
      ["invitations[3].email", "duplicate"],
    ],
  ]);

  // Members and guests may not invite, list, change or delete.
  const invitationId = { invitation_id: carolsInvitation?.id ?? "" };
  const byOthers = await Promise.all(
    [member, guest].flatMap((token) => [
      api.POST("/api/v0/workspaces/{slug}/invitations", {
        params: { path: { slug } },
        body: { invitations: [{ email: emailFor(testInfo, "gina"), role: 5 }] },
        headers: bearer(token),
      }),
      api.GET("/api/v0/workspaces/{slug}/invitations", { params: { path: { slug } }, headers: bearer(token) }),
      api.PATCH("/api/v0/workspace-invitations/{invitation_id}", {
        params: { path: invitationId },
        body: { role: 5 },
        headers: bearer(token),
      }),
      api.DELETE("/api/v0/workspace-invitations/{invitation_id}", {
        params: { path: invitationId },
        headers: bearer(token),
      }),
    ])
  );
  expect(byOthers.map((r) => [r.response.status, r.error?.code])).toEqual(byOthers.map(() => [403, "forbidden"]));
  await expectInvitations(
    db,
    slug,
    adminEmail,
    [
      ...accepted,
      { email: carol, role: 20, accepted: false, responded: false, deleted: false },
      { email: dave, role: 5, accepted: false, responded: false, deleted: true },
      { email: erinEmail, role: 15, accepted: false, responded: true, deleted: false },
    ],
    tokens
  );
});
````

`e2e/stories/workspace/w5-invitation-link.spec.ts`（新文件，112 行）：

````file e2e/stories/workspace/w5-invitation-link.spec.ts
import { createWorkspace, invite, slugFor, type Api, type WorkspaceInvitation } from "../../fixtures/api";
import { expectInvitations, expectMembership } from "../../fixtures/assert/workspace";
import { bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// W5, accept or decline an invitation by its link (M3 design 2, 3.8; decision 1). The page version comes with
// /workspace-invitations (P9).

/** Accepts or declines invitation with token, as the account of bearerToken. */
async function answer(
  api: Api,
  bearerToken: string,
  which: "accept" | "decline",
  invitation: WorkspaceInvitation,
  token = invitation.token
) {
  return api.POST(`/api/v0/workspace-invitations/{invitation_id}/${which}`, {
    params: { path: { invitation_id: invitation.id } },
    body: { token },
    headers: bearer(bearerToken),
  });
}

test("W5 (API): the link shows the workspace and the role without the address; the invitee accepts or declines; another address, a wrong token and an answered invitation change nothing", async ({
  api,
  db,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = (await createPAT(api, (await register(api, adminEmail)).access_token)).token;
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin, { name: "Acme", slug });
  const carolEmail = emailFor(testInfo, "carol");
  const carol = (await createPAT(api, (await register(api, carolEmail)).access_token)).token;
  const daveEmail = emailFor(testInfo, "dave");
  const dave = (await createPAT(api, (await register(api, daveEmail)).access_token)).token;
  const [toCarol, toDave] = await invite(api, admin, slug, [
    { email: carolEmail, role: 5 },
    { email: daveEmail, role: 15 },
  ]);
  if (!toCarol || !toDave) {
    throw new Error("the invitations were not created");
  }

  // The public view: with the token, the workspace and the role, never the address; the same with a bearer
  // token; without the token 400; with a changed token or of no invitation, the same 404.
  const view = (id: string, token?: string, headers: Record<string, string> = {}) =>
    api.GET("/api/v0/workspace-invitations/{invitation_id}", {
      params: { path: { invitation_id: id }, query: { token } as { token: string } },
      headers,
    });
  const shown = await view(toCarol.id, toCarol.token);
  expect(shown.response.status).toBe(200);
  expect(shown.data).toEqual({
    id: toCarol.id,
    role: 5,
    declined: false,
    workspace_name: "Acme",
    workspace_slug: slug,
    workspace_logo_url: null,
  });
  expect(JSON.stringify(shown.data)).not.toContain(carolEmail);
  expect((await view(toCarol.id, toCarol.token, bearer(dave))).data).toEqual(shown.data);
  expect((await view(toCarol.id)).response.status).toBe(400);
  const changed = toCarol.token.slice(0, 10) + (toCarol.token[10] === "A" ? "B" : "A") + toCarol.token.slice(11);
  const wrong = await view(toCarol.id, changed);
  const missing = await view(crypto.randomUUID(), toCarol.token);
  expect([wrong.response.status, wrong.error]).toEqual([404, missing.error]);
  expect(wrong.error?.code).toBe("workspace.invitation_not_found");

  // Another account's answer: 403 without the address, and nothing changes.
  const refused = await Promise.all([answer(api, dave, "accept", toCarol), answer(api, dave, "decline", toCarol)]);
  expect(refused.map((r) => [r.response.status, r.error?.code])).toEqual([
    [403, "workspace.invitation_email_mismatch"],
    [403, "workspace.invitation_email_mismatch"],
  ]);
  expect(JSON.stringify(refused.map((r) => r.error))).not.toContain(carolEmail);
  // The invited address without the link's token: 400, nothing read. A wrong token: the 404 of an invitation
  // that does not exist.
  const noToken = await api.POST("/api/v0/workspace-invitations/{invitation_id}/accept", {
    params: { path: { invitation_id: toCarol.id } },
    body: {} as never,
    headers: bearer(carol),
  });
  expect(noToken.response.status).toBe(400);
  const badToken = await answer(api, carol, "accept", toCarol, changed);
  expect([badToken.response.status, badToken.error]).toEqual([404, wrong.error]);
  await expectMembership(db, slug, carolEmail, null);

  // carol accepts: a member with the invitation's role; the invitation is used up.
  const accepted = await answer(api, carol, "accept", toCarol);
  expect(accepted.response.status).toBe(200);
  expect(accepted.data).toMatchObject({ slug, role: 5, total_members: 2 });
  await expectMembership(db, slug, carolEmail, { role: 5, is_active: true });
  // dave declines: no membership; the link shows it declined.
  expect((await answer(api, dave, "decline", toDave)).response.status).toBe(204);
  await expectMembership(db, slug, daveEmail, null);
  expect((await view(toDave.id, toDave.token)).data?.declined).toBe(true);
  await expectInvitations(db, slug, adminEmail, [
    { email: carolEmail, role: 5, accepted: true, responded: true, deleted: true },
    { email: daveEmail, role: 15, accepted: false, responded: true, deleted: false },
  ]);

  // Answered already: carol's is gone, 404; dave's is declined, 409 for either answer.
  expect((await answer(api, carol, "accept", toCarol)).error?.code).toBe("workspace.invitation_not_found");
  expect((await view(toCarol.id, toCarol.token)).response.status).toBe(404);
  const again = await Promise.all([answer(api, dave, "accept", toDave), answer(api, dave, "decline", toDave)]);
  expect(again.map((r) => [r.response.status, r.error?.code])).toEqual([
    [409, "workspace.invitation_responded"],
    [409, "workspace.invitation_responded"],
  ]);
  await expectMembership(db, slug, daveEmail, null);
});
````

`e2e/stories/workspace/w6-sign-up-by-invitation.spec.ts`（新文件，85 行）：

````file e2e/stories/workspace/w6-sign-up-by-invitation.spec.ts
import { accept, createApi, createWorkspace, invite, slugFor } from "../../fixtures/api";
import { countIdentity, expectNothingAdded } from "../../fixtures/assert/identity";
import { expectInvitations, expectMembership } from "../../fixtures/assert/workspace";
import { bearer, createPAT, emailFor, password, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// W6, sign up by an invitation while sign-up is off (M3 design 2, 3.8; decision 1). The page version comes with
// /workspace-invitations (P9).

test("W6 (API): with sign-up off, the address an invitation was sent to registers with its link and then accepts it; without a link, with a wrong token, another address, a declined or a deleted invitation it may not", async ({
  api,
  db,
  nerveWith,
}, testInfo) => {
  // The links are of the nerve that made them: without a key file each nerve has its own key (README,
  // deployment), so the workspace, the invitations and the answers all go through the closed nerve, with
  // personal access tokens, which every nerve on the database accepts.
  const closed = createApi((await nerveWith({ NERVE_AUTH__SIGNUP_ENABLED: "false" })).baseURL);
  const adminEmail = emailFor(testInfo, "admin");
  const admin = (await createPAT(api, (await register(api, adminEmail)).access_token)).token;
  const daveEmail = emailFor(testInfo, "dave");
  const dave = (await createPAT(api, (await register(api, daveEmail)).access_token)).token;
  const slug = slugFor(testInfo);
  await createWorkspace(closed, admin, { name: "Acme", slug });
  const carolEmail = emailFor(testInfo, "carol");
  const erinEmail = emailFor(testInfo, "erin");
  const [toCarol, toDave, toErin] = await invite(closed, admin, slug, [
    { email: carolEmail, role: 15 },
    { email: daveEmail, role: 15 },
    { email: erinEmail, role: 5 },
  ]);
  if (!toCarol || !toDave || !toErin) {
    throw new Error("the invitations were not created");
  }
  const declined = await closed.POST("/api/v0/workspace-invitations/{invitation_id}/decline", {
    params: { path: { invitation_id: toDave.id } },
    body: { token: toDave.token },
    headers: bearer(dave),
  });
  expect(declined.response.status).toBe(204);
  const deleted = await closed.DELETE("/api/v0/workspace-invitations/{invitation_id}", {
    params: { path: { invitation_id: toErin.id } },
    headers: bearer(admin),
  });
  expect(deleted.response.status).toBe(204);

  const signUp = (email: string, invitation?: { id: string; token: string }) =>
    closed.POST("/api/v0/auth/register", { body: { email, password, invitation } });
  const changed = toCarol.token.slice(0, 10) + (toCarol.token[10] === "A" ? "B" : "A") + toCarol.token.slice(11);
  const before = await countIdentity(db);
  // Each is refused as sign-up being off. dave's address is taken: an invitation that let it through would
  // answer 409.
  const refusals = {
    "no invitation": signUp(carolEmail),
    "a wrong token": signUp(carolEmail, { id: toCarol.id, token: changed }),
    "another address": signUp(emailFor(testInfo, "mallory"), { id: toCarol.id, token: toCarol.token }),
    "a declined invitation": signUp(daveEmail, { id: toDave.id, token: toDave.token }),
    "a deleted invitation": signUp(erinEmail, { id: toErin.id, token: toErin.token }),
    "no such invitation": signUp(carolEmail, { id: crypto.randomUUID(), token: toCarol.token }),
  };
  const answers = await Promise.all(
    Object.entries(refusals).map(async ([label, refused]) => {
      const r = await refused;
      return [label, r.response.status, r.error?.code];
    })
  );
  expect(answers).toEqual(Object.keys(refusals).map((label) => [label, 403, "identity.signup_disabled"]));
  await expectNothingAdded(db, before);

  // carol registers with her link, the address as she types it; the invitation stays to be answered.
  const registered = await signUp(` ${carolEmail.toUpperCase()} `, { id: toCarol.id, token: toCarol.token });
  expect(registered.response.status).toBe(201);
  await expectInvitations(db, slug, adminEmail, [
    { email: carolEmail, role: 15, accepted: false, responded: false, deleted: false },
    { email: daveEmail, role: 15, accepted: false, responded: true, deleted: false },
    { email: erinEmail, role: 5, accepted: false, responded: false, deleted: true },
  ]);
  await expectMembership(db, slug, carolEmail, null);
  expect(await accept(closed, registered.data?.access_token ?? "", toCarol)).toMatchObject({
    slug,
    role: 15,
    total_members: 2,
  });
  await expectMembership(db, slug, carolEmail, { role: 15, is_active: true });
});
````

`e2e/stories/workspace/w8-navigation-preferences.spec.ts`（修改，3 处）：

````old e2e/stories/workspace/w8-navigation-preferences.spec.ts
  createWorkspace,
````

````new e2e/stories/workspace/w8-navigation-preferences.spec.ts
  createWorkspace,
  inviteAndAccept,
````

````old e2e/stories/workspace/w8-navigation-preferences.spec.ts
test("W8 (API): the settings are the defaults and nothing is stored until the first change, which stores one row; later changes change it; each workspace has its own", async ({
````

````new e2e/stories/workspace/w8-navigation-preferences.spec.ts
test("W8 (API): the settings are the defaults and nothing is stored until the first change, which stores one row; later changes change it; each member and each workspace has its own", async ({
````

````old e2e/stories/workspace/w8-navigation-preferences.spec.ts
  expect(await expectPreferences(db, slug, email, { ...tabbed, navigation_project_limit: 0 })).toBe(row);

````

````new e2e/stories/workspace/w8-navigation-preferences.spec.ts
  expect(await expectPreferences(db, slug, email, { ...tabbed, navigation_project_limit: 0 })).toBe(row);

  // Another member of the workspace reads the defaults, and nothing is stored for him.
  const memberEmail = emailFor(testInfo, "member");
  const member = (await createPAT(api, (await register(api, memberEmail)).access_token)).token;
  await inviteAndAccept(api, pat, slug, { email: memberEmail, token: member }, 15);
  expect(await read(api, member, slug)).toEqual(defaults);
  await expectPreferences(db, slug, memberEmail, null);

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
Expected: 57 个全部通过。

- [ ] **Step 4: 提交**

```bash
git add e2e/fixtures/api.ts e2e/fixtures/assert/workspace.ts e2e/stories/workspace/w3-workspace-settings.spec.ts e2e/stories/workspace/w4-invite-members.spec.ts e2e/stories/workspace/w5-invitation-link.spec.ts e2e/stories/workspace/w6-sign-up-by-invitation.spec.ts e2e/stories/workspace/w8-navigation-preferences.spec.ts
```
```bash
git commit -m "test(M3/P3): the API versions of W3, W4, W5 and W6; W8 has a second member

W3 deletes a workspace with its members, invitations and settings at one
moment; W4 invites, changes and deletes, and is refused as the design
says; W5 answers a link; W6 registers by an invitation on a nerve with
sign-up off, all through that nerve, whose key its links need (M3 design
9.6). W8's second member joins by an invitation, so a settings read that
ignores the account fails W8 run alone.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**变异**（只运行一个故事；spec 附录 A 的 `e2e_mutants.py`）：

| 改坏 | 必须失败的故事 |
|---|---|
| 表里加一个 `token` 列 | W4（`expectInvitations` 核对 11 列恰好是设计的列） |
| `Preferences` 读工作区的第一行，不看账户（P2 的风险） | W8（第二个成员读到管理员的设置） |

**Done when:** `make e2e` 57 个全部通过；两个变异各让它的故事单独运行时失败。

---

### Task 15: 上级文档和交接

**Files:**
- Modify: `README.md`、`docs/v0/M3-workspace-project/handoffs/M2-closeout.md`、`docs/v0/plane-diff.md`、`docs/v0/v0-design.md`

**Interfaces:**
- Produces（spec 2.16，M3 设计 3.20、8.7）：总体设计 1.1（成员邀请、登录方式）、4.2（约定六的接受一段）；差异清单二·按表（`workspace_member_invites` 七行）、四（邀请的六行，删除工作区一行）；README 的"部署""安全"（邀请链接、注册、账户被盗之后的第三步）；M2 收尾交接"处理结果（M3/P3）"。

**Tests:** 关键词守卫（`make lint-web`）也查文档。

- [ ] **Step 1: 总体设计**

`docs/v0/v0-design.md`（修改，3 处）：

````old docs/v0/v0-design.md
| 工作区 | 工作区、成员与角色（管理员 / 成员 / 访客）、成员邀请（在系统内接受，不发邮件） |
````

````new docs/v0/v0-design.md
| 工作区 | 工作区、成员与角色（管理员 / 成员 / 访客）、成员邀请（只有工作区管理员发出邀请，由他复制邀请链接交给对方，对方凭链接接受；不发邮件） |
````

````old docs/v0/v0-design.md
  "先锁父行，再判定"从 M3/P2 起落地（修改、删除工作区，改成员的角色，修改显示设置）：两位管理员互相降级时，先拿到工作区锁的一方成功，后到的一方判定时已是成员而被拒绝。约定六的接受邀请一段在 M3/P3、停用一段在 M3/P6 随实现核对。
````

````new docs/v0/v0-design.md
  "先锁父行，再判定"从 M3/P2 起落地（修改、删除工作区，改成员的角色，修改显示设置）：两位管理员互相降级时，先拿到工作区锁的一方成功，后到的一方判定时已是成员而被拒绝。约定六的接受邀请一段从 M3/P3 起落地：接受、忽略邀请最先以 `FOR SHARE` 锁住调用者的账户行，在锁下重读 `is_active` 和邮箱，再锁工作区（接受 `FOR NO KEY UPDATE`，忽略 `FOR SHARE`）和邀请行（`FOR UPDATE`）；与改邮箱并发时，接受比较的是对方提交之后的邮箱。停用一段在 M3/P6 随实现核对。
````

````old docs/v0/v0-design.md
- **登录方式**：v0 只有邮箱加密码。是否开放注册由配置 `auth.signup_enabled` 控制：prod 默认关闭，dev、test 默认开放；关闭时注册先答"注册已关闭"，不查邮箱。prod 的第一个账户由服务器管理员用 `nerve users create` 创建（M2 设计决策点 2）。被邀请的邮箱始终可以注册。
````

````new docs/v0/v0-design.md
- **登录方式**：v0 只有邮箱加密码。是否开放注册由配置 `auth.signup_enabled` 控制：prod 默认关闭，dev、test 默认开放；关闭时注册先答"注册已关闭"，不查邮箱。prod 的第一个账户由服务器管理员用 `nerve users create` 创建（M2 设计决策点 2）。注册关闭时，带着有效的邀请链接、注册邮箱（规范化后）与邀请的邮箱相同的人仍可注册；邀请无效的各种情况（令牌不对、邀请不存在、已删除、已回应、邮箱不同）与不带邀请得到同一个"注册已关闭"。公开的邀请查看不显示被邀请的邮箱（M3 设计 3.8、决策点 1）。
````

- [ ] **Step 2: 差异清单**

`docs/v0/plane-diff.md`（修改，2 处）：

````old docs/v0/plane-diff.md
| `workspace_user_properties` | 删除 `filters`、`display_filters`、`display_properties`、`rich_filters` | 工作项列表的筛选和显示列由它们的使用者 M4 按自己的格式加回（M3 设计 3.18） |
````

````new docs/v0/plane-diff.md
| `workspace_user_properties` | 删除 `filters`、`display_filters`、`display_properties`、`rich_filters` | 工作项列表的筛选和显示列由它们的使用者 M4 按自己的格式加回（M3 设计 3.18） |
| `workspace_member_invites` | 13 列保留 11 列（M3/P3，`00009_workspace_workspace_member_invites.sql`） | M3 设计 4.4 |
| `workspace_member_invites` | `workspace_id`：加上 `ON DELETE CASCADE`；`created_by_id`、`updated_by_id`：加上 `ON DELETE SET NULL` | 二·全局（模型的 `on_delete`） |
| `workspace_member_invites` | `email`：新加 `CHECK (email <> '' AND email = lower(email) AND email !~ '[[:space:]]')`，存规范化之后的邮箱 | 与 `users.email` 的规则相同（M3 设计 3.13）；Plane 按原样存 |
| `workspace_member_invites` | `role`：`CHECK (role >= 0)` 收紧为 `CHECK (role IN (5, 15, 20))`，新加 `DEFAULT 5`；`accepted`：新加 `DEFAULT false`；新加 `workspace_member_invites_responded_check CHECK (responded_at IS NOT NULL OR NOT accepted)`；`created_at`、`updated_at`：新加 `DEFAULT now()`（兜底，见二·全局） | 只有三种角色（M3 设计 3.4）；接受的邀请一定有回应的时刻 |
| `workspace_member_invites` | 部分唯一索引改为 `workspace_member_invites_workspace_id_email_key ON (workspace_id, email) WHERE deleted_at IS NULL`（Plane 是 `workspace_member_invite_unique_email_workspace_when_deleted_at_ ON (email, workspace_id)`）；新加 `workspace_member_invites_email_idx ON (email) WHERE deleted_at IS NULL`；`workspace_id` 的索引改为 `workspace_member_invites_workspace_id_idx` | 已忽略的邀请没有删除，仍占着它的邮箱（M3 设计 3.8）；注册策略按邮箱查；物理级联要不带条件的索引（M3 设计 4） |
| `workspace_member_invites` | 删除 `token`、`message` | 不存令牌：链接里的令牌是由签名密钥派生的 MAC 从邀请的 id 算出的，数据库泄露时待接受的链接不泄露（M3 设计 3.8、8.1）；`message` 没有写入方 |
````

````old docs/v0/plane-diff.md
| 删除工作区 | 清掉别人的 `last_workspace_id`；成员、显示设置等的连带软删除由 Celery 异步完成 | 不清 `last_workspace_id`（登录后的落点规则让它无害，M3 设计 3.14）；工作区、成员、显示设置在同一个事务里、同一时刻软删除，删除之后 slug 立即可以重用。邀请由 M3/P3、项目由 M3/P4、标签由 M3/P7 加入这个事务 |
````

````new docs/v0/plane-diff.md
| 删除工作区 | 清掉别人的 `last_workspace_id`；成员、显示设置等的连带软删除由 Celery 异步完成 | 不清 `last_workspace_id`（登录后的落点规则让它无害，M3 设计 3.14）；工作区、成员、邀请、显示设置在同一个事务里、同一时刻软删除，删除之后 slug 立即可以重用。项目由 M3/P4、标签由 M3/P7 加入这个事务 |
| 邀请的列出、创建、修改、删除 | 工作区管理员和成员；修改不限制角色，成员能把邀请改成管理员 | 只有工作区管理员（M3 设计决策点 4）；邀请的角色因此不高于邀请人（M3 设计 3.8） |
| 邀请令牌 | JWT，原文存库；公开的查看不要令牌，返回被邀请的邮箱；关闭注册时，有任何一份未删除的邀请的邮箱就能注册 | 由签名密钥派生的 MAC（`nrv_inv_` 加 22 个字符），不存库，管理员列出时重新算出；公开的查看要令牌、不返回邮箱；关闭注册时要有效的令牌、且注册邮箱与邀请的相同（M3 设计 3.8、决策点 1） |
| 邀请的链接 | `/workspace-invitations/?invitation_id=…&slug=…&token=…`；另有系统内的接受：`/invitations` 页和新手引导的"加入工作区"一步按账户的邮箱列出发给他的邀请，批量接受 | 只有链接一条路：`/workspace-invitations?invitation_id=…&token=…`；接受要登录，账户的邮箱须与邀请的相同（M3 设计 3.8、决策点 2） |
| 重复的邀请 | 静默忽略 | 422 `duplicate`，整批不插入，`invitations[i]` 是它在请求中的下标；已忽略的邀请仍占着这个邮箱，删除之后才能再邀请（M3 设计 3.8） |
| 接受邀请之后 | 服务端写 `last_workspace_id` | 前端写（M3 设计 3.14） |
| 接受邀请时已有成员行 | 不分有效还是已离开，都把角色改为邀请的角色 | 已是有效成员：只消费邀请，成员关系和角色不变；以前的成员行：恢复，角色取邀请的（访客时项目角色的连带由 M3/P4 加入）（M3 设计 3.8） |
````

- [ ] **Step 3: README**

`README.md`（修改，3 处）：

````old README.md
  访问令牌由它签名，刷新令牌的 MAC 密钥也从它派生。换密钥（替换文件后重启）的后果：已签发的访问令牌验签失败，客户端续期一次即可；换钥之前的旧代刷新令牌不再能被认出是重复使用，所以刷新令牌正被盗用时换钥，受害者只是被登出（恢复靠修改密码或 `nerve users reset-password`）；同一出口 IP 后面的大量标签页同时续期，会短暂得到 429。所以在低峰时换。
````

````new README.md
  访问令牌由它签名，刷新令牌和邀请链接的 MAC 密钥也从它派生（下面"邀请链接"一条）。换密钥（替换文件后重启）的后果：已签发的访问令牌验签失败，客户端续期一次即可；换钥之前的旧代刷新令牌不再能被认出是重复使用，所以刷新令牌正被盗用时换钥，受害者只是被登出（恢复靠修改密码或 `nerve users reset-password`）；同一出口 IP 后面的大量标签页同时续期，会短暂得到 429。所以在低峰时换。
````

````old README.md
- **注册**：prod 默认关闭注册（`auth.signup_enabled: false`）。第一个账户用 `nerve users create --email <邮箱>` 创建（见下一条），注册关闭时也能用。开放注册时，注册接口会暴露一个邮箱是否已经注册：没有邮件通道，就无法让两种回答相同。注册按客户端 IP 限流（默认每分钟 10 次），只能压低探测的速度。登录不暴露账户是否存在；存储的哈希都用当前的 argon2 参数时，耗时也相同。调高 `auth.password.argon2_memory_kib`、`argon2_iterations` 之后，还没有重新登录过的账户能从耗时上与不存在的邮箱区分开，直到它们各登录一次，所以这两个参数少调（M2 设计 §16）。关闭注册时，已注册和未注册的邮箱得到同一个 403。
````

````new README.md
- **注册**：prod 默认关闭注册（`auth.signup_enabled: false`）。第一个账户用 `nerve users create --email <邮箱>` 创建（见下一条），注册关闭时也能用。开放注册时，注册接口会暴露一个邮箱是否已经注册：没有邮件通道，就无法让两种回答相同。注册按客户端 IP 限流（默认每分钟 10 次），只能压低探测的速度。登录不暴露账户是否存在；存储的哈希都用当前的 argon2 参数时，耗时也相同。调高 `auth.password.argon2_memory_kib`、`argon2_iterations` 之后，还没有重新登录过的账户能从耗时上与不存在的邮箱区分开，直到它们各登录一次，所以这两个参数少调（M2 设计 §16）。关闭注册时，已注册和未注册的邮箱得到同一个 403；只有带着有效的邀请链接、注册邮箱与邀请的邮箱相同时能注册，邀请无效的各种情况也是这同一个 403（M3 设计 3.8）。
- **邀请链接**：工作区管理员在成员页邀请邮箱，复制每份邀请的链接交给对方（nerve 不发邮件）；对方用这个邮箱的账户登录后凭链接接受。链接里的令牌由签名密钥派生（M3 设计 3.8），数据库里不存：
  - 同一部署的全部 nerve 进程用同一个密钥文件，否则一个进程发的链接在别的进程上无效（访问令牌本来也这样要求）。
  - 换签名密钥之后，待接受的链接全部失效，管理员在成员页重新复制即可；dev、test 没有配置密钥时，nerve 重启也让链接失效。
  - 链接没有有效期。让一个泄露的链接失效：删除这份邀请，重新邀请（新的链接）。
  - 令牌在链接的查询参数里。nerve 的访问日志只记路径，不记查询参数；反向代理的访问日志通常记下整个地址，这样的日志要像密钥一样保管，或者让代理不记查询参数。
````

````old README.md
  2. **再撤销不认识的 PAT**：在 api-tokens 页或 security 页的列表中（或 `GET /api/v0/me/api-tokens`），按创建时间和最后使用时间认出不认识的令牌，全部撤销，拿不准的也撤销；然后重新打开列表核对。PAT 能创建 PAT：撤销的同时，对方可能用还没撤销的 PAT 建了新的，列表里不再出现不认识的令牌才算完成。
````

````new README.md
  2. **再撤销不认识的 PAT**：在 api-tokens 页或 security 页的列表中（或 `GET /api/v0/me/api-tokens`），按创建时间和最后使用时间认出不认识的令牌，全部撤销，拿不准的也撤销；然后重新打开列表核对。PAT 能创建 PAT：撤销的同时，对方可能用还没撤销的 PAT 建了新的，列表里不再出现不认识的令牌才算完成。

  3. **再核对待接受的邀请**：修改密码和 `reset-password` 都不撤销账户以前发出的邀请，链接属于邀请，不属于发出它的凭证。在各工作区的成员页核对待接受的邀请，删除不认识的。
````

- [ ] **Step 4: 交接**

`docs/v0/M3-workspace-project/handoffs/M2-closeout.md`（修改，1 处）：

````old docs/v0/M3-workspace-project/handoffs/M2-closeout.md
来源：[M3/P2 spec](../specs/P2-workspaces.md) 第 7 节。

````

````new docs/v0/M3-workspace-project/handoffs/M2-closeout.md
来源：[M3/P2 spec](../specs/P2-workspaces.md) 第 7 节。

## 处理结果（M3/P3）

- **第 1 节 邀请与注册**（接口一侧完成）：负责人的裁定写在 M3 设计第 10 节（决策点 1、2、4）。接受、忽略要登录，请求体带链接的令牌（`nrv_inv_` 加 22 个字符，由签名密钥派生，不存库，`server/internal/modules/workspace/domain/token.go`）；不带令牌 400，令牌不对与邀请不存在同一个 404，邮箱不一致 403 `workspace.invitation_email_mismatch`，回答不含被邀请的邮箱。接受最先以 `FOR SHARE` 锁住调用者的账户行，锁下重读 `is_active` 和邮箱（`server/internal/modules/workspace/app/respond_invitation.go`）；已是有效成员时只消费邀请，成员关系和角色不变（`TestAcceptWorkspaceInvitation`，交错测试 12 `TestAcceptingAndChangingTheAddress`）。`auth.signup_enabled = false` 时，带有效邀请、邮箱相同的注册成功，不带的和邀请无效的各种情况都是 403 `identity.signup_disabled`（`server/internal/bootstrap/signup_policy.go`，`TestRegisteringWithAnInvitationWhileSignupIsOff`）。W5、W6 的接口版本核对这些。页面一侧（邀请页、注册页带着邀请回到邀请页，W5、W6 的页面版本）在 P9，本节保持 `open`。

仍未处理，状态保持 `open`：第 1 节的页面一侧（P9）；第 2、3、6 节，第 7 节的其余部分，第 9–11、13、14 节，随 M3 设计 13.1 中各自的 Phase；第 12 节等 M3 的收尾。

来源：[M3/P3 spec](../specs/P3-invitations.md) 第 7 节。

````

- [ ] **Step 5: 检查**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

Run: `make lint-web`
Expected: 通过。

- [ ] **Step 6: 提交**

```bash
git add README.md docs/v0/M3-workspace-project/handoffs/M2-closeout.md docs/v0/plane-diff.md docs/v0/v0-design.md
```
```bash
git commit -m "docs(M3/P3): the design, plane-diff, README and handoff rows of P3

v0 design 1.1: invitations by link, sent by a workspace's admins; with
sign-up off, a valid link lets its address register. 4.2: convention 6's
acceptance holds from P3. plane-diff: workspace_member_invites column by
column and the invitations' behaviour rows. README: the invitation links
and the signing key, sign-up by invitation, reviewing pending invitations
after an account was taken over. M2-closeout: what P3 did of section 1.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** M3 设计 3.20 中 P3 的各行（总体设计 1.1、4.2，差异清单）和 8.7 的 P3 一行写好；M2 收尾交接有"处理结果（M3/P3）"。
