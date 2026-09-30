# M3/P3 邀请与凭邀请注册：设计说明（spec）

| 项 | 内容 |
|---|---|
| Phase | M3/P3 `invitations` |
| 日期 | 2026-09-30 |
| 状态 | 待控制者裁定第 3 节各条 |
| 上级文档 | [M3 设计文档](../M3-design.md) 第 2（W3–W6、W8）、3.3、3.4、3.6（加锁顺序、加锁表、约定一至六）、3.8、3.9、3.12、3.13、3.14、3.20（P3 各行）、4.4、4.11（P3 各行）、4.12、5.1–5.4、6.5、6.6、6.7、8.1、8.2、8.4、8.7（P3 一行）、9.1–9.4、9.6、10（决策点 1、2、4）、11.1、11.2、12（P3 与约束 3、4）、13.1、17 节；[v0 总体设计](../../v0-design.md) 1.1、4.2 节；[M2 设计](../../M2-auth/M2-design.md) 3.4、3.5、3.7、3.9、3.11 节 |
| 前置交接 | [M2 收尾交接](../handoffs/M2-closeout.md) 第 1 节的接口一侧；[P2 review](../reviews/P2-workspaces-review.md) 第 6 节交给 P3 的各件和 [P1 review](../reviews/P1-platform-review.md) 第 6 节的公开操作（落点见第 3 节第 2 条） |
| 计划 | [P3 plan](../plans/P3-invitations.md) |

本 spec 只写 M3 设计交给 P3 决定的东西：名字、签名、SQL、令牌的字节布局、测试名，以及原型证明了什么。规则本身以 M3 设计为准，这里引用节号，不重述。P3 依赖 P2（`1d2eaf7c` 的 `main`）。

P3 是评审敏感的一段（M3 设计 12 节约束 3）：令牌和"从邀请到成员关系"的路径是一个账户得到访问权的地方。令牌和接受的每个测试都按安全测试对待：附录 A 的变异表逐个核对它们在所说的性质去掉之后失败。

## 1. 目标

按 M3 设计 12 节 P3：邀请的全部操作；关闭注册时凭邀请注册；第二个成员从这里开始存在。具体是：

- 迁移 `00009`（`workspace_member_invites`）和它的运行时权限；删除工作区的连带在工作区行之后加上邀请（4.4、4.12）；
- 邀请令牌：`identity.LoadKeys` 移到 `bootstrap` 的第 1 步，`identity.Keys` 按用途派生 MAC（6.6、11.1）；令牌的格式、消息和比对在 `workspace` 的领域（3.8、8.1）；数据库不存令牌；
- `identity.Provide` 交出 `CredentialLock`，`workspace` 的 `CallerLock` 端口（6.5、6.6）；
- 七个操作：`listWorkspaceInvitations`、`createWorkspaceInvitations`（一批全有或全无，按规范化后的邮箱排序插入，23505 翻译为 422）、`updateWorkspaceInvitation`、`deleteWorkspaceInvitation`、公开的 `getWorkspaceInvitation`、`acceptWorkspaceInvitation`、`declineWorkspaceInvitation`（约定六的顺序）；
- 注册：`SignupPolicy.AllowSignup(ctx, email, invitation)`、`RegisterRequest.invitation`、`bootstrap` 的注册策略（3.8，决策点 1）；
- `apitest` 的请求体用例扩展到对象的数组（5.2、9.4）；
- 交错 3、9、12、18、19，两个顺序；`pgtest.WaitForKeyWaitOn`（唯一索引上的等待）；
- 矩阵：邀请的 6 行、账户级的 4 行（60 格），公开的查看以操作级的豁免加一个专门的测试；
- 端到端：`api.ts` 的邀请、接受、邀请并接受；W3、W4、W5、W6 的接口版本；W8 加第二个成员；
- 3.20、8.7 中 P3 的各行（总体设计 1.1、4.2，差异清单，README），M2 收尾交接的"处理结果（M3/P3）"。

## 2. 交付物

### 2.1 文件总览

路径相对于仓库根目录。"生成"表示由 `make gen`（或 `make gen-go`）生成并提交，不手改。"Task"是 plan 中负责它的任务（plan 有完整的文件表）。

| 路径 | 内容 | Task |
|---|---|---|
| `server/migrations/sql/00009_workspace_workspace_member_invites.sql`、`schema_test.go`；`deploy/runtime-grants.sql`；`server/sqlc.yaml` | 邀请的表；名字、种类、CHECK 的反例、部分唯一键；运行时权限 | 1 |
| `workspace/domain/invitation.go`、`app/invitation_ports.go`、`adapter/postgres/invitations.go`、`queries/invitations.sql` 及测试；`app/delete_workspace.go`；`bootstrap/workspace_deletion_test.go` | 邀请的领域、端口、存储的插入；删除工作区的连带 | 1 |
| `identity/keys.go`、`identity/adapter/signing/mac.go`、`keys.go`、`identity/module.go`；`bootstrap/app.go`；`workspace/domain/token.go` 及测试 | `LoadKeys` 移到 `bootstrap`；按用途派生的 MAC；令牌的领域 | 2 |
| `platform/httpserver/apitest/operations.go`、`bodycases.go` 及测试；`bootstrap/contract_test.go` | 对象的数组的请求体用例（请求体用例移到自己的文件）；必填的查询参数 | 3 |
| `identity/provide.go`、`provide_test.go` | `Provided.CredentialLock` | 4 |
| `workspace/app/list_invitations.go`、`app/tokens.go`、HTTP、存储、接口描述；`access/domain/rules.go`；`bootstrap/invitation_mac_test.go`、矩阵 | `listWorkspaceInvitations`；邀请的 MAC 接进 `workspace` | 5 |
| `workspace/app/create_invitations.go`、`domain/invitation.go`、HTTP、接口描述；规则；矩阵 | `createWorkspaceInvitations`；`CallerLock` 接进 `workspace` | 6 |
| `workspace/adapter/postgres/invitations.go`、`locks.go`、`queries/`、`update_invitation_test.go` | 按 id 读、锁邀请；按 id 的 `FOR SHARE`；改角色、删除的存储 | 7 |
| `workspace/app/update_invitation.go`、`delete_invitation.go`、`invitation_lock.go`、`domain/errors.go`、HTTP、接口描述；规则；矩阵；前端文案 | 修改、删除邀请；两个新码 | 8 |
| `workspace/app/get_invitation.go`、HTTP（`PublicOperations`）、存储、接口描述；`bootstrap/invitations_test.go`、`permission_matrix_coverage_test.go` | 公开的查看；访问日志；矩阵的豁免和"未知参数" | 9 |
| `workspace/adapter/postgres/responses.go`、`queries/` 及测试 | 接受、忽略要的存储：`MemberOf`、`RestoreMember`、`AcceptInvitation`、`DeclineInvitation`、`WorkspaceByID` | 10 |
| `workspace/app/respond_invitation.go`、`accept_invitation.go`、`decline_invitation.go`、`domain/errors.go`、HTTP、接口描述；矩阵；前端文案 | 接受、忽略；一个新码 | 11 |
| `identity/app/register.go`、`ports.go`、HTTP、`api/modules/identity.yaml`；`workspace/app/signup_invitations.go`、`module.go`；`bootstrap/signup_policy.go`、`app.go`、`invitations_test.go` | 凭邀请注册 | 12 |
| `platform/postgres/pgtest/lockwait.go` 及测试；`bootstrap/interleaving_answers_test.go`、`interleaving_invite_test.go`、`interleaving_test.go` | `WaitForKeyWaitOn`；交错 3、9、12、18、19；闸门的期限 | 13 |
| `e2e/fixtures/api.ts`、`assert/workspace.ts`；`e2e/stories/workspace/w3-…`、`w4-…`、`w5-…`、`w6-…`、`w8-…` | W3–W6 的接口版本；W8 的第二个成员 | 14 |
| `docs/v0/v0-design.md`、`plane-diff.md`、`README.md`、`handoffs/M2-closeout.md` | 3.20、8.7 中 P3 的各行；交接的处理结果 | 15 |
| `api/dist/openapi.yaml`、`web/packages/api-client/src/schema.gen.ts`、`workspace/adapter/http/gen/*.gen.go`、`identity/adapter/http/gen/*.gen.go`、`workspace/adapter/postgres/gen/*.go`（生成） | | 1、5–12 |

`workspace/…`、`identity/…`、`access/…` 指 `server/internal/modules/` 下的模块；`bootstrap/…`、`platform/…` 指 `server/internal/` 下。

### 2.2 依赖

没有新依赖（6.1）。`server/go.mod`、`server/tools/go.mod` 和两个 `go.sum` 不变，仍是 `go 1.27` / `toolchain go1.27.1`；不加 npm 包。HKDF 用标准库的 `crypto/hkdf`（M2 已用），base64url 用 `encoding/base64` 的 `RawURLEncoding.Strict()`。

### 2.3 迁移 `00009` 和删除的连带（4.4、4.12；P2 review 第 6 节）

- `workspace_member_invites` 的 11 列照 4.4。约束和索引的名字与种类（`schema_test.go` 同时钉住两者）：`_pkey`（`p`、`iu`）、`_workspace_id_fkey`（`ON DELETE CASCADE`）、`_created_by_id_fkey`、`_updated_by_id_fkey`（`SET NULL`）、`_email_check`（与 `users.email` 相同，另加不能为空）、`_role_check`（`role IN (5, 15, 20)`）、`_responded_check`（`responded_at IS NOT NULL OR NOT accepted`）、`_workspace_id_email_key`（`UNIQUE … WHERE deleted_at IS NULL`，`iuw`）、`_email_idx`（`WHERE deleted_at IS NULL`，`iw`：注册策略、停用按邮箱查）、`_workspace_id_idx`（物理级联）。
- 反例（`TestChecksRejectCounterexamples`）：邮箱的大写（ASCII 与非 ASCII）、结尾的制表符、中间的 U+3000、空串；角色 10、0；`accepted` 而没有 `responded_at`；去掉接受的时间。部分唯一键的行为（`TestUniqueKeysHoldAmongUndeletedRowsOnly`）：已忽略的邀请是未删除的行，仍占着邮箱，删除之后才空出来。
- 运行时角色的 `GRANT` 加上这张表。
- **删除的连带**：`cascade()` 在工作区行之后加 `DeleteWorkspaceInvitations`（全局加锁顺序：工作区 → 邀请 → 成员 → 显示设置），同一个 `now`、同一个删除者：

```sql
UPDATE workspace_member_invites
SET deleted_at = $now, updated_at = $now, updated_by_id = $deleted_by
WHERE workspace_id = $workspace_id AND deleted_at IS NULL;
```

  待接受的、已忽略的一起删除；此前已删除的（已接受的）保持原来的时间。组合的删除测试（`TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt`）从目录找出引用 `workspaces` 的每张表：它给两个工作区都准备了邀请，没有为邀请加任何豁免（P2 review 第 6 节第 1 件）。

### 2.4 签名密钥、按用途的 MAC 与令牌（3.8、6.6、8.1、11.1）

- **`identity.LoadKeys(pem []byte, logger *slog.Logger) (*identity.Keys, error)`**：`bootstrap.New` 的第 1 步（6.6），在任何模块构造之前。`pem` 为 `nil` 时是临时密钥，`logger` 警告"访问令牌和邀请链接在重启之后失效（只用于 dev、test）"；解析失败的错误不引用密钥。`identity.Deps.SigningKeyPEM` 换成 `identity.Deps.Keys`，`identity.New` 不再读密钥文件。
- **`(*identity.Keys).MAC(purpose string) (identity.MAC, error)`**：`identity.MAC` 有 `Tag(message []byte) [16]byte`、`Verify(message []byte, tag [16]byte) bool`。`purpose` 为 `"refresh-token"` 时返回错误：刷新令牌的 MAC 只属于 `identity`（第 3 节第 5 条）。密钥不离开 `identity/adapter/signing`：`signing.Keys.MAC(purpose)` 按 `K = HKDF-SHA256(私钥的种子, salt = nil, info = "nerve " + purpose + " mac v1", 32 字节)` 派生，`Tag` 是 `HMAC-SHA256(K, message)` 的前 16 字节，`Verify` 用 `hmac.Equal`。刷新令牌的 MAC 改为 `signing.Keys.MAC(PurposeRefreshToken)`，派生与 M2 相同：已发出的刷新令牌仍然有效（已知答案 `65cc085217dccf9e602799b8e64a4468` 不变）。
- **`workspace.InvitationMACPurpose = "workspace-invitation"`**：`bootstrap` 用它向 `Keys` 要邀请的 MAC，经 `workspace.Deps.InvitationMAC`（`app.InvitationMAC`，与 `identity.MAC` 同形）交给 `workspace`。两个模块互不导入。
- **令牌的字节布局**（`workspace/domain/token.go`）：

| 部分 | 内容 |
|---|---|
| 消息 `InvitationMessage(id)` | ASCII `workspace-invitation`（20 字节）‖ 邀请 id 的 16 字节（`uuid.UUID` 的数组，RFC 9562 的字节序）：36 字节 |
| 标签 | `HMAC-SHA256(K_invitation, 消息)` 的前 16 字节（128 位） |
| 令牌 `FormatToken(tag)` | `nrv_inv_` ‖ 标签的 base64url（RFC 4648 第 5 节，不带 `=`）：22 个字符，最后一个字符的低 4 位是填充、必须为 0；共 30 个 ASCII 字符 |
| 解析 `ParseToken(token)` | 前缀不是 `nrv_inv_`、之后不是恰好 22 个字符、不是 `RawURLEncoding.Strict()` 能解出的 16 字节：不是令牌（`ok = false`，不交给 MAC） |

- **已知答案**（测试的签名密钥 `testKeyPEM`，附录 A 的 `kat.py` 按 RFC 5869、RFC 4648 手算，不经 Go 的代码）：id `0199a2b4-0000-7000-8000-000000000001` 的标签是 `92fab228187e6c034c4fa240218ce894`，令牌是 `nrv_inv_kvqyKBh-bANMT6JAIYzolA`。它钉在三处：`signing` 的 `TestMACKnownAnswers`（派生和 HMAC）、领域的 `TestTheTokenOfTheKnownAnswer`（消息和编码）、`bootstrap` 的 `TestTheInvitationMACIsTheDesigns`（组合根要的用途是设计的用途：用途换了，此前的链接全部失效）。
- **比对**：`app/tokens.go` 的 `invitationTokens.valid(id, token)` 先 `ParseToken`，再 `mac.Verify(InvitationMessage(id), tag)`；`Verify` 只答 `hmac.Equal`（`TestMACVerifyComparesInConstantTime` 读代码核对，时间测不出来）。`invitationTokens.of(id)` 在列出、创建、修改邀请时重新算出令牌；服务端从不存它。

### 2.5 `apitest`：对象的数组（5.2、9.4；P2 review 第 6 节）

- `Operation.BodyCases()` 加两类用例：11. 第一个对象数组的元素里有未声明的属性（有效的请求体在这个数组里放一个有效的元素）；12. 这个元素逐个缺少必填属性。10.（全部问题一起）加上第 11 类。字符串的数组没有这些用例；只取第一个对象数组。`TestBodyCasesOfAListOfObjects` 核对。
- `ParamCases` 给每个必填的查询参数一个"不带它"的用例：`getWorkspaceInvitation` 的 `token`，答 400 `bad_request`（`token` 的 `required`）；`ParamCase` 加 `Code`。整程序测试 `TestParametersThatDoNotBindAnswer400` 按契约逐个操作执行，新操作自动在其中。
- 加上这些之后 `operations.go` 从 359 行长到 411 行。请求体用例是单独的一件事（M2 设计 3.11 的第四个整程序测试）：`FieldProblem`、`BodyCase`、`BodyCases` 移到 `apitest/bodycases.go`（216 行），它们的测试移到 `bodycases_test.go`；`operations.go`（205 行）留下操作、参数用例和两边共用的 schema 辅助函数。只是移动，没有改写。
- 生成的 `bodyshape` 表对元素的检查由整程序测试 `TestBodiesThatBreakTheStructureAnswer400` 在真实的 app 上看到：让元素接受未声明的属性、不要求任何属性，两个变异都让它失败（附录 A）。

### 2.6 `CredentialLock` 与 `CallerLock`（3.8、6.5、6.6）

- `identity.Provided.CredentialLock`（`identity.CredentialLock` 接口）：`LockCaller(ctx, actor shared.Actor, now time.Time) error`，就是 M2 的凭证锁（M2 设计 3.5）：以 `FOR NO KEY UPDATE` 锁调用者的账户行到事务结束，在锁下复核账户有效、会话或 PAT 在 `now` 有效，否则 401 `unauthorized`。实现 `callerLock` 包着 `app.CredentialLock`，丢掉锁到的账户（别的模块拿不到密码的哈希）。它只用 `identity` 的存储，所以放在 `Provide`（6.6 的唯一例外）；`identity.New` 与它共用 `credentialLock(store)`。
- `workspace.Deps.CallerLock`（`app.CallerLock`，同形），`bootstrap` 直接接上，不需要转换。
- `TestProvidedCredentialLockIsTheCredentialLock`：有效的会话和 PAT 取得锁，账户行锁到事务结束（事务中另一个连接的 `FOR NO KEY UPDATE NOWAIT` 得到 `55P03`，事务结束之后得到这一行）；撤销的会话、撤销的 PAT、停用的账户、不存在的账户都是 401。

### 2.7 `listWorkspaceInvitations`（3.8、3.12、5.1）

- 接口：`GET /api/v0/workspaces/{slug}/invitations`，200 `WorkspaceInvitationList{data: [WorkspaceInvitation]}`；码 `[workspace.not_found, forbidden]`；规则 `workspace_invitation.list`：管理员（决策点 4）。
- `WorkspaceInvitation{id, workspace_id, email, role, accepted, responded_at, created_at, created_by_id, token}`（5.2）；`accepted` 总是 `false`（接受时软删除，不再列出）；`created_by_id` 可为 `null`（邀请人的账户被物理删除时 `SET NULL`，M4）。
- 用例：不开事务：`WorkspaceBySlug` → 判定 → `ListInvitations` → 每一条加上 `of(id)`。查询：

```sql
SELECT … FROM workspace_member_invites
WHERE workspace_id = $workspace_id AND deleted_at IS NULL
ORDER BY created_at DESC, id;
```

### 2.8 `createWorkspaceInvitations`（3.6、3.8、5.1）

- 接口：`POST /api/v0/workspaces/{slug}/invitations`，`WorkspaceInvitationsCreate{invitations: [InvitationCreate{email, role}]}`（两层都 `additionalProperties: false`），201 `WorkspaceInvitationList`（请求的顺序）；码 `[validation_failed, workspace.not_found, forbidden]`；规则 `workspace_invitation.create`：管理员。只有管理员邀请，邀请的角色自然不高于邀请人（3.8），没有单独的检查。
- **只看请求的检查**（`domain.CheckInvitations`，事务之前）：条数 1–100（`invitations` 的 `too_short`、`too_long`，单独回答）；每个邮箱按注册的规则规范化、校验（3.13；`invitations[i].email` 的 `invalid_format`）；规范化之后在本批中重复的，后出现的 `invitations[i].email` `duplicate`；角色按集合 `{5, 15, 20}`（`invitations[i].role` 的 `invalid_format`）。有问题时一个 422 列出全部，`i` 是请求中的下标（第 3 节第 3 条）。
- **一个事务**：`CallerLock.LockCaller`（邀请人的账户行 `FOR NO KEY UPDATE`，事务的第一把锁，约定一）→ `ShareWorkspaceBySlug`（`FOR SHARE`，约定五）→ 判定 → `checkAddresses`：`ListMembers` 取有效成员的 id，经不加锁的 `MemberProfiles.PublicProfiles` 换成邮箱（约定一；不经 `Accounts`，也不 JOIN `users`），与 `ListInvitations`（未删除的，含已忽略的）一起：是有效成员的邮箱 `not_allowed`，已有邀请的 `duplicate`，一个 422 列出全部 → `CreateInvitations(byAddress(rows))`：按规范化后的邮箱逐字节排序，一行一条语句：

```sql
INSERT INTO workspace_member_invites (id, workspace_id, email, role, created_by_id, updated_by_id, created_at, updated_at)
VALUES ($id, $workspace_id, $email, $role, $created_by, $created_by, $now, $now)
RETURNING …;
```

- **23505**：存储把 `workspace_member_invites_workspace_id_email_key` 的冲突答成 `*app.DuplicateInvitation{Email}`，之后的行不再插入；用例把它翻译为与事先检查相同的 422 `invitations[i].email` `duplicate`，`i` 是这个邮箱在**请求**中的下标（不是排序后的位置），事务回滚、整批都没有（`TestADuplicateFromTheUniqueKeyNamesTheRequestsIndex`、交错 18）。别的 CHECK 冲突是错误（值已由领域检查）。
- **时钟**：在事务之前读一次（第 3 节第 4 条，`TestCreatingInvitationsReadsTheClockBeforeItsTransaction`）；行的 id 是 `uuid.NewV7()`。回答按请求的顺序，每一条是存下的行（`RETURNING`）加令牌。

### 2.9 `updateWorkspaceInvitation`、`deleteWorkspaceInvitation`（3.6 约定二、3.8、5.1）

- 接口：`PATCH /api/v0/workspace-invitations/{invitation_id}`，`WorkspaceInvitationUpdate{role}`，200 `WorkspaceInvitation`；码 `[validation_failed, workspace.invitation_not_found, forbidden, workspace.invitation_responded]`。`DELETE` 同一路径，204；码 `[workspace.invitation_not_found, forbidden]`。规则 `workspace_invitation.update`、`workspace_invitation.delete`：管理员。
- 新码（Task 8，同一个 Task 加进 `PROBLEM_MESSAGES` 和两份 `auth.json`，12 节约束 4）：`workspace.invitation_not_found`（404，"The invitation does not exist, or its link is not valid."：不存在、已删除（含已接受）、看不到它的工作区；对持链接的人，还有令牌不对）、`workspace.invitation_responded`（409，"The invitation has been answered already."）。
- 存储（Task 7）：

```sql
-- InvitationByID：锁工作区之前按 id 读，为了知道它的工作区
SELECT … FROM workspace_member_invites WHERE id = $id AND deleted_at IS NULL;
-- ShareWorkspace：按 id 的 FOR SHARE（P2 的 ShareWorkspaceBySlug 按 slug）
SELECT id FROM workspaces WHERE id = $id AND deleted_at IS NULL FOR SHARE;
-- LockInvitation：工作区的锁之下，邀请行 FOR UPDATE、重读
SELECT … FROM workspace_member_invites WHERE id = $id AND deleted_at IS NULL FOR UPDATE;
-- UpdateInvitationRole
UPDATE workspace_member_invites SET role = $role, updated_by_id = $updated_by, updated_at = $now WHERE id = $id RETURNING …;
-- DeleteInvitation：邮箱立即空出来（部分唯一索引）
UPDATE workspace_member_invites SET deleted_at = $now, updated_at = $now, updated_by_id = $deleted_by WHERE id = $id;
```

  `LockInvitation` 等锁期间邀请被删除（或被接受）时，Postgres 在提交的新版本上重新求值 `deleted_at IS NULL`，读到 0 行：`app.ErrNotFound`（`TestLockInvitationSkipsAnInvitationDeletedWhileItWaits`）。`FOR UPDATE` 而不是 `FOR NO KEY UPDATE`：挡住外键检查的 `FOR KEY SHARE`（`TestLockInvitationLocksTheRowForUpdate`），邀请的行没有子行，这把锁只在邀请之间竞争。
- 按资源寻址的写（`app/invitation_lock.go`）：`readInvitation`（`InvitationByID`）→ `lockInvitation`（锁工作区 → `LockInvitation`）；两处的 `ErrNotFound` 都换成 `domain.ErrInvitationNotFound`，别的失败原样。
- 修改：`CheckMemberRole`（事务之前）→ 一个事务：读 → `ShareWorkspace` → `LockInvitation` → `decide(workspace_invitation.update, invitation_not_found)` → 已忽略：409 `invitation_responded`（判定之后：成员对任何邀请都得到 403）→ 读时钟 → `UpdateInvitationRole`。回答加令牌。
- 删除：同样的锁和判定 → 读时钟 → `DeleteInvitation`。已忽略的邀请照样删除（3.8：删掉它之后才能再邀请这个邮箱）。
- 两者都在锁之后读时钟（P2 spec 2.6），`clock_test.go` 的 `TestEachWriteReadsTheClockUnderItsLock` 加这两行。

### 2.10 公开的 `getWorkspaceInvitation`（3.8、8.1、8.2；P1 review 第 6 节）

- 接口：`GET /api/v0/workspace-invitations/{invitation_id}?token=…`，`security: []`，查询参数 `token` 必填（不带是 400）；200 `InvitationPreview{id, role, declined, workspace_name, workspace_slug, workspace_logo_url}`（`workspace_logo_url` 在 M5 之前总是 `null`）；码 `[workspace.invitation_not_found]`。按 IP 计数（`anonymous`，5.4），`workspace` 的 HTTP 适配器由 `PublicOperations()` 交出这个操作，`bootstrap` 把它并进公开操作的列表。
- 用例：`valid(id, token)` 为假时直接 404，**不读任何东西**；之后才 `InvitationPreview`：

```sql
SELECT i.id, i.role, i.responded_at, w.name AS workspace_name, w.slug AS workspace_slug
FROM workspace_member_invites i JOIN workspaces w ON w.id = i.workspace_id
WHERE i.id = $id AND i.deleted_at IS NULL AND w.deleted_at IS NULL;
```

  （同一个模块的两张表，可以 JOIN。）没有这一行也是 `invitation_not_found`，失败是错误。令牌只由 id 算出、不要行，所以"令牌不对"与"邀请不存在"的回答不可能因邀请是否存在而不同（8.2）；回答里没有邮箱，也不因调用者是否登录而不同（路由不读凭证）。
- **访问日志**：`TestTheInvitationLinksTokenIsNotLogged` 在组合出的 app 上以 debug 级别记全部日志，令牌对（200）、改成大写（404）、不带（400）三个请求之后：三行访问日志的 `path` 都是不带查询的路径；日志里没有 `token=`、令牌、令牌的 22 个字符、`nrv_inv_`。
- **矩阵**：见 2.14；这个操作以操作级的豁免 `matrixExempt.public` 加 `TestTheInvitationLinkAnswersEveryCallerAlike` 覆盖（第 3 节第 6 条）。

### 2.11 `acceptWorkspaceInvitation`、`declineWorkspaceInvitation`（3.6 约定一、六，3.8，9.1）

- 接口：`POST /api/v0/workspace-invitations/{invitation_id}/accept`、`/decline`，bearer，请求体 `InvitationResponse{token}`（必填，`additionalProperties: false`）；接受 200 `Workspace`（`role` 是调用者现在的角色），忽略 204；码 `[workspace.invitation_not_found, workspace.invitation_email_mismatch, workspace.invitation_responded]`。账户级：不经 `Authorizer`（6.4）。
- 新码（Task 11，同时加文案）：`workspace.invitation_email_mismatch`（403，"The invitation was sent to another e-mail address."，不含那个邮箱）。
- 共用的步骤（`app/respond_invitation.go` 的 `responder`）：
  1. `checkToken`：令牌不对 404，**在事务之前、不读任何东西**；
  2. 一个事务：`Accounts.ShareAccount(调用者)`（`FOR SHARE`，事务的第一把锁；锁下读 `is_active` 和邮箱）：没有或已停用 401 `unauthorized`；
  3. `readInvitation` → 锁工作区（接受 `LockWorkspace` 即 `FOR NO KEY UPDATE`，成员关系的写；忽略 `ShareWorkspace` 即 `FOR SHARE`）→ `LockInvitation`（`FOR UPDATE`，重读）；
  4. 锁下的邮箱不等于邀请的邮箱：403 `invitation_email_mismatch`（什么都不写）；已忽略：409 `invitation_responded`。邮箱的比较在"已回应"之前：别的邮箱不会得知邀请是否被回应过。
- 接受，锁之后读时钟，`MemberOf(工作区, 调用者)`：
  - 有效的成员关系：不改（角色、`is_active`），回答的 `role` 是他原来的角色；
  - 已结束的：`RestoreMember`（`is_active = true`，角色取邀请的；P4 在这里、同一个事务里加 `DemoteToGuest`，当角色是访客时）；
  - 没有：`CreateMember`（新 id，角色取邀请的，调用者是 `created_by`）；
  - 然后 `AcceptInvitation`（`accepted = true`、`responded_at = deleted_at = now`，接受者是 `updated_by`），最后在同一个事务里 `WorkspaceByID` 读回答（`total_members` 含他）。
- 忽略，锁之后读时钟，`DeclineInvitation`（`responded_at = now`，不删除：仍占着邮箱）。不读、不写成员关系：有效成员的邀请照样可以忽略。
- 存储（Task 10，`adapter/postgres/responses.go`）：

```sql
-- MemberOf：他在这个工作区未删除的成员关系，有效的或已结束的（部分唯一索引保证至多一行）
SELECT id, workspace_id, member_id, role, is_active, created_at FROM workspace_members
WHERE workspace_id = $workspace_id AND member_id = $member_id AND deleted_at IS NULL;
-- RestoreMember
UPDATE workspace_members SET is_active = true, role = $role, updated_by_id = $restored_by, updated_at = $now WHERE id = $id;
-- AcceptInvitation
UPDATE workspace_member_invites SET accepted = true, responded_at = $now, deleted_at = $now, updated_at = $now,
       updated_by_id = $accepted_by WHERE id = $id;
-- DeclineInvitation
UPDATE workspace_member_invites SET responded_at = $now, updated_at = $now, updated_by_id = $declined_by WHERE id = $id;
-- WorkspaceByID：接受的回答，事务里在成员关系改了之后读
SELECT w.id, w.name, w.slug, w.organization_size, w.timezone, w.created_at, w.updated_at,
       (SELECT count(*) FROM workspace_members c WHERE c.workspace_id = w.id AND c.is_active AND c.deleted_at IS NULL) AS total_members
FROM workspaces w WHERE w.id = $id AND w.deleted_at IS NULL;
```

  `WorkspaceByID` 没有这一行是错误，不是 404：调用者持着它的锁。
- 用例测试（9.1）：`TestAcceptWorkspaceInvitation` 的 bob 是 `acme` 的成员、`beta` 的访客，邀请的角色高、低、相同都不改他的角色，只消费邀请；erin 的已结束的成员关系以邀请的角色恢复；frank 新建。`responseRefusals` 与 `responseFailures` 是接受、忽略共用的拒绝和失败表：每一种都不写任何东西，失败原样返回（不是 404）。

### 2.12 凭邀请注册（3.8，决策点 1）

- `identity`：`app.SignupInvitation{ID uuid.UUID; Token string}`（`identity.SignupInvitation` 是它的别名）；`SignupPolicy.AllowSignup(ctx, email string, invitation *SignupInvitation) (bool, error)`；`RegisterInput.Invitation`。`Register` 的第 1 步以**规范化后**的邮箱和请求原样的邀请问策略，在任何别的检查之前；拒绝一律 403 `identity.signup_disabled`。`RegisterRequest` 加可选的 `invitation`（结构 `RegisterInvitation {id: uuid, token: string}`，都必填，`additionalProperties: false`）；错误码不变。
- `workspace`：`app.SignupInvitations.Allows(ctx, email, id, token) (bool, error)`：先 `valid(id, token)`（不对就答 `false`，不读）→ `InvitationByID`（未删除的；没有就 `false`）→ `!Responded() && inv.Email == email`。读失败是错误，不是拒绝。只读，不接受邀请。`workspace.Module.SignupInvitations()` 交出它（`workspace.SignupInvitations` 接口）。
- `bootstrap/signup_policy.go` 的 `signupPolicy{enabled bool; invitations workspace.SignupInvitations}`：开放时一律允许、不问邀请；关闭时不带邀请拒绝，带邀请照 `workspace` 的回答。它代替 M2 的 `signupSwitch`（删除）。6.6 的顺序不变：`workspace.New` 在 `identity.New` 之前。
- `TestRegisteringWithAnInvitationWhileSignupIsOff`：同一个库、同一个密钥上的第二个 app 关闭注册；带有效邀请、邮箱相同（大小写、空白不同也算）的注册 201，邀请仍待接受；不带邀请、令牌不对、id 不存在、已删除、已接受、已忽略、邮箱不同，七种都是**逐字节相同**的 403 `identity.signup_disabled`。

### 2.13 探针与交错 3、9、12、18、19（9.3；P2 review 第 6 节）

- **`pgtest.WaitForKeyWaitOn(t, pool, table, limit)`**：只在有连接"在本事务里写过 `table`（持有它已授予的 `RowExclusiveLock`），在等另一个事务结束（`wait_event = 'transactionid'`），而且不持有任何 tuple 锁"时返回；表不存在时立即失败，到期失败。这是唯一索引检查的等待：插入与一个未结束的事务插入的键相同，等那个事务提交或回滚，没有行可锁。`WaitForLockWaitOn` 看不到它（没有 tuple 锁）。它分不清的，文档写明：同一个写过 `table` 的事务在别的表上等键，或者 Postgres 不取 tuple 锁的两种行等待（`WaitForLockWaitOn` 的文档所列）。只用在等待的事务不写别的带唯一键的表、不升级共享的行锁的地方。`TestWaitForKeyWaitOnSeesOnlyAKeysWaitOnItsTable` 的正例：等同一个键的 `INSERT`（`WaitForLockWait` 证明它在等、`WaitForLockWaitOn` 看不到）；反例（各在自己的库里，都由 `WaitForLockWait` 证明确实在等）：写过这张表的事务等这张表的一行（有 tuple 锁）；只读过这张表的事务在另一张表上等键；写过这张表的事务等咨询锁。
- **每个交错用哪个探针、为什么别的等待满足不了它**：每个测试有自己的库（`pgtest.NewDatabase`），用例照 `bootstrap` 的接法手工组装，库上没有 River，只有两方在跑；先的一方停在闸门上（它的事务里、持着锁），探针确认后的一方等在设计说的那一处，然后打开闸门。

| 交错 | 测试 | 先的一方停在 | 后的一方等在 | 探针 |
|---|---|---|---|---|
| 3 接受与删除工作区 | `TestAcceptingAndDeletingTheWorkspace` | 接受：`MemberOf` 之前（持账户 S、工作区 N、邀请 U）；删除：`DeleteWorkspace` 之前（持工作区 N、已判定） | `acme` 的行 | `WaitForLockWaitOn "workspaces"`：删除不锁账户，两方只在工作区行上相遇 |
| 9 创建邀请与重置密码 | `TestInvitingAndResettingThePassword` | 重置：写密码哈希之前（持 alice 的账户行）；创建：插入之后、提交之前（持 alice 的账户行和工作区 S） | alice 的账户行 | `WaitForLockWaitOn "users"`：两方都锁的只有这一行 |
| 12 接受与改邮箱 | `TestAcceptingAndChangingTheAddress` | 改邮箱：写入之后、撤销会话之前（持 bob 的账户行 `FOR UPDATE`）；接受：`MemberOf` 之前（持 `FOR SHARE`） | bob 的账户行 | `WaitForLockWaitOn "users"` |
| 18 重叠的两批 | `TestInvitingOverlappingBatches` | 先的一批：插入之后、提交之前 | 先的一批插入的键 | `WaitForKeyWaitOn "workspace_member_invites"`：后的一批持工作区 S（与先的一方的 S 相容）、锁自己的账户行，不等任何行 |
| 19 忽略与停用 | `TestDecliningAndDeactivating` | 忽略：写入之前（持账户 S、工作区 S、邀请 U）；停用：撤销会话之前（持账户行和 `acme` 的 N） | bob 的账户行 | `WaitForLockWaitOn "users"`：两方都先锁账户行 |

- 结果：3 接受先：删除之后连带删除 bob 新的成员关系；删除先：接受 404 `invitation_not_found`，bob 不是成员。9 重置先：创建 401，没有邀请；创建先：邀请留下（属于工作区，不属于凭证）。12 改邮箱先：403 `invitation_email_mismatch`，什么都没变；接受先：bob 按原邮箱成为成员，之后改邮箱照常完成。18 alice `[x, y]`、carol `[y, x]`：后的一方 422 `invitations[i].email` `duplicate`，`i` 是 `x` 在它自己请求里的下标，它的整批都没有插入；单个邮箱的竞争（两批都是 `[x]`）下标 0；两个顺序都没有 40P01。19 忽略先：停用照常；停用先：忽略在锁下读到已停用，401，邀请仍待接受；都没有 40P01。
- 19 的停用一方：P6 之前真实的停用不锁工作区；测试用 `nerve users deactivate` 的用例，包一层会话撤销器：在撤销会话之前按 3.9 的顺序取 `acme` 的 `FOR NO KEY UPDATE`（账户行已由停用锁住）。P6 换成真实的停用再跑一次（9.3）。
- **闸门的期限**（第 3 节第 9 条）：P2 的 `gate.wait` 只等 ctx；一个把创建挪到事务之外的变异让闸门的 ctx 变成 `context.Background()`，测试挂到 `go test` 的 10 分钟超时。`newGate` 记下 10 秒的期限，`wait` 到期返回错误"the gate was not opened within 10s"：测试失败而不挂住。

### 2.14 矩阵（9.2；P1、P2 review 第 6 节）

- 准备数据：`matrixInvitations` 在准备之前定名（P2 的 P10/C4）：`acme`、`gone` 各一份 `newcomer@example.com`；每一列一份发给自己邮箱的邀请：`acme` 的管理员、成员、访客在 `other`，另三列在 `acme`（"已被移出"的一列接受时恢复他已结束的成员关系）。`gone` 的邀请随 `gone` 的删除一起删除。`seeded.invitation(slug, email)` 取准备好的 id；`invitationToken` 用矩阵的签名密钥、经 `identity.LoadKeys` 和 `Keys.MAC(workspace.InvitationMACPurpose)` 算令牌，与组合出的 app 相同。
- 本 Phase 的 10 行（60 格；P2 的 72 格加起来共 132 格）：

| 行 | 管理员 | 成员 | 访客 | 从来不是成员 | 已被移出 | 工作区已删除 | 答案的核对 |
|---|---|---|---|---|---|---|---|
| `acceptWorkspaceInvitation`，自己的 | 200 | 200 | 200 | 200 | 200 | 200 | `joinsAsAMember`：邀请的工作区、角色 15、人数多一 |
| `acceptWorkspaceInvitation`，newcomer 的 | 403 `invitation_email_mismatch` × 6 | | | | | | |
| `declineWorkspaceInvitation`，自己的 | 204 × 6 | | | | | | |
| `declineWorkspaceInvitation`，newcomer 的 | 403 `invitation_email_mismatch` × 6 | | | | | | |
| `listWorkspaceInvitations` | 200 | 403 | 403 | 404 | 404 | 404 | `listsTheInvitations`：`acme` 的、不含 `gone` 的，令牌是各自 id 的 |
| `createWorkspaceInvitations` | 201 | 403 | 403 | 404 | 404 | 404 | `invitesTheInvitee` |
| 同上，有效成员的邮箱 | 422 | 403 | 403 | 404 | 404 | 404 | |
| 同上，已邀请的邮箱 | 422 | 403 | 403 | 404 | 404 | 404 | |
| `updateWorkspaceInvitation` | 200 | 403 | 403 | 404 `invitation_not_found` | 同左 | 同左 | `promotesTheNewcomer`：角色 20，令牌是它 id 的 |
| `deleteWorkspaceInvitation` | 204 | 403 | 403 | 404 `invitation_not_found` | 同左 | 同左 | |

  （"工作区已删除"一列的修改、删除指名 `gone` 里的邀请；404 不带码的是 `workspace.not_found`。）"有效成员的邮箱""已邀请的邮箱"两行经组合出的 `MemberProfiles` 和存储得到 422：接线没人看的缺陷在这里看得到。
- **公开的查看**：没有行，`matrixExempt.public = {"getWorkspaceInvitation": "TestTheInvitationLinkAnswersEveryCallerAlike"}`。完整性核对要求：豁免的操作存在、确实公开、没有行。这个测试代替那一行：`acme` 的链接和四种坏链接（改一个字符、别的邀请的令牌、`gone` 的邀请带它自己的令牌、不存在的 id），各以不带令牌和六列的令牌请求：每个调用者得到相同的答案，好链接 200、没有邮箱，四种坏链接是**逐字节相同**的同一个 404。
- **`targetViolation`**（P2 re-review Minor 2）：除了 `workspaces` 之后的 `{slug}` 和以 `_id}` 结尾的参数，任何别的 `{…}` 都报告，除非它的路径列在 `matrixExempt.notTargets` 里（带理由）：`/api/v0/workspace-slugs/{slug}`（问的是 slug，不是工作区）、`…/{invitation_id}/accept`、`…/{invitation_id}/decline`（账户级：每一列答自己的邀请，或 `acme` 的 newcomer 的，与它那一列指向的工作区无关）。列出而不是任何操作的路径也报告。邀请的路径参数按 5.1 叫 `{invitation_id}`。`TestMatrixViolationsCatchesEachGap` 加上这些反例。
- 耗时见附录 A。

### 2.15 端到端（2 的 W3–W6、W8，9.6）

- **fixture**（`e2e/fixtures/api.ts`）：`invite(api, token, slug, invitations)`（答请求顺序的邀请，带令牌）、`accept(api, token, invitation)`（答新成员读到的工作区）、`inviteAndAccept(api, adminToken, slug, {email, token}, role)`（第二个成员的唯一来路，12 节约束 1），类型 `WorkspaceInvitation`、`InvitationCreate` 取自生成的客户端。`assert/workspace.ts`：`InvitationRow`、`invitationColumns`（11 列，钉住表里没有令牌列）、`expectInvitations(db, slug, inviterEmail, want)`（按 `COLLATE "C"` 排序比较；接受的 `deleted_at` 等于 `responded_at`；任何一行都不含任何一个令牌）、`expectMembership(db, slug, email, want | null)`、`expectWorkspaceDeleted`（每张属于工作区的表：至少一行随工作区在同一时刻删除，没有未删除的，也没有更晚删除的）。
- **W3 (API)**：管理员改名、规模、时区；成员 `PATCH`、`DELETE` 403；带 `slug` 400；删除：工作区、成员、邀请（待接受的和已忽略的）、显示设置在同一时刻删除；原成员 404。
- **W4 (API)**：管理员一批邀请（邮箱规范化）；改一条的角色为 20；删除一条；列表带令牌；erin 忽略她的；两次被拒：本批重复的邮箱（`[["invitations[1].email","duplicate"]]`，只看请求的问题单独回答），以及有效成员的邮箱 `not_allowed`、已邀请的、已忽略的 `duplicate`（`[1]`、`[2]`、`[3]`）；成员、访客 `POST`、`GET`、`PATCH`、`DELETE` 都是 403 `forbidden`；最后 `expectInvitations`。
- **W5 (API)**：查看带不带 bearer 相同；不带令牌 400；改动的令牌与不存在的 id 同一个 404；别的邮箱接受 403、回答里没有被邀请的邮箱、数据库不变；接受的请求体 `{}` 是 400；令牌不对的接受 404；接受 200（角色 5，`total_members` 2）；忽略 204，之后查看显示已忽略；已回应的再接受、再忽略 409。
- **W6 (API)**：在关闭注册的第二个 nerve 上（9.6：它没有密钥文件，令牌只在它那里有效），工作区、邀请、回应都经它、用 PAT；六种无效的注册都是 403 `identity.signup_disabled`、数据库不变；carol 以 ` CAROL@… `（大写、空白）带有效邀请注册 201，邀请仍待接受；她经这个 nerve 接受，成为成员。
- **W8 (API)**：第二个工作区之外加一个经 `inviteAndAccept` 加入的成员：他读到默认值、库里没有他的行（P2 review 第 7 节：`Preferences` 不看 `user_id` 在 W8 单独运行时也失败）。

### 2.16 文档（3.20、8.7 的 P3 各行）

| 文档 | 位置 | 内容 |
|---|---|---|
| 总体设计 | 1.1 | 成员邀请"在系统内接受"改为"只有工作区管理员发出邀请，由他复制邀请链接交给对方，对方凭链接接受；不发邮件"（决策点 2、4）；登录方式一段："被邀请的邮箱始终可以注册"改为注册关闭时持有效邀请链接、邮箱一致的人仍可注册，各种无效与不带邀请同一个回答，公开的查看不显示邮箱（决策点 1） |
| 总体设计 | 4.2 | 约定六的接受邀请一段从 M3/P3 起落地 |
| 差异清单 | 二·按表 | `workspace_member_invites` 的七行（保留的列、外键的删除行为、`email` 的 CHECK、角色和默认值与回应的 CHECK、索引、删除的两列） |
| 差异清单 | 四 | 4.11 中标 P3 的行：谁能管理邀请、令牌、链接、重复的邀请、接受之后、接受时已有成员行；删除工作区一行加上邀请 |
| README | 部署、安全 | 8.7 的 P3 一行：密钥文件派生邀请链接的 MAC；全部进程同一个密钥文件；换钥、dev 重启之后链接失效；让泄露的链接失效；反向代理的日志；注册关闭时凭邀请注册；账户被盗之后核对待接受的邀请 |
| M2 收尾交接 | 文末 | "处理结果（M3/P3）"：第 1 节的接口一侧 |

（原计划的"总体设计 4.2 的注册"一行：总体设计中那句话在 1.1 的"登录方式"里，4.2 没有；Task 15 在 1.1 改，第 3 节第 13 条。）

## 3. 与设计的差异和补充（待控制者裁定）

以下都没有改变 M3 设计的架构。每条是一个待裁定的事项，除非标着"说明"。

1. **任务的划分：15 个，不是 14 个**。设计的任务与 plan 的对应：1 → 1；2 → 2；3 → 4（`Provide`）、6（`CallerLock` 接上）；4 → 2（令牌的领域与 MAC 一起：它的已知答案要 MAC 的已知答案）；5 → 6；6 → 5（列出）、7（存储）、8（修改、删除）；7 → 9；8 → 10（存储）、11（用例）；9 → 12；10 → 3；11 → 13；12 → 每个操作的 Task 加自己的行（5、6、8、9、11）；13 → 14；14 → 15（review 是控制者的）。列出、修改删除、接受忽略的存储各拆出来，因为合在一起时 plan 的一个 Task 超过约 1,500 行。最大的是 Task 11（接受、忽略，1,415 行）、Task 5（1,355 行）和 Task 9（1,332 行）；全部 15 个 Task 共 13,188 行。
2. **P2 review 第 6 节和 P1 review 第 6 节的落点**（说明）：

   | 移交 | 落点 |
   |---|---|
   | 1. 连带加上邀请；组合的删除测试不加豁免 | Task 1（`cascade()`、两个工作区的准备行、存储和用例的删除测试） |
   | 2. 交错 3 用 `WaitForLockWaitOn(…, "workspaces", …)` | Task 13 |
   | 3. 交错 18 等在唯一索引上：选或加一个只看得到这种等待的探针，正例、反例都证明；9、12、19 说明用哪个 | Task 13：`WaitForKeyWaitOn`（2.13）；9、12、19 用 `WaitForLockWaitOn "users"`，理由见 2.13 的表 |
   | 4. W8 的第二个成员 | Task 14 |
   | 5. 修改、删除、接受、忽略在锁之后读时钟；创建说明前后 | Task 8、11（`clock_test.go` 各加两行）；创建在事务之前读（第 4 条），Task 6 |
   | 6. 公开的查看进矩阵：一列"不带令牌"或操作级的豁免加理由；令牌不对、别的邀请的令牌 | Task 9：操作级的豁免加 `TestTheInvitationLinkAnswersEveryCallerAlike`（第 6 条） |
   | 7. `targetViolation` 报告不认识的参数；列出的例外带理由 | Task 9（规则、反例、`workspace-slugs/{slug}`）、Task 11（`accept`、`decline`） |
   | 8. `apitest` 的对象的数组 | Task 3 |

   不属于 P3 的：`DeleteWorkspaceProjects`、`DemoteToGuest`（P4；接受的用例在恢复成员关系之后写明 P4 的接入点）；"已被移出"的准备数据和交错 1（P5）；交错 19 的真实停用（P6）；`PROBLEM_MESSAGES` 的搬移（P8）；M4 的物理删除。
3. **创建：只看请求的问题先单独回答**。3.8 写"任何一个邮箱不合规，整批都不插入，回答列出每个不合规的 `invitations[i]`"。本 Phase 分两步：只看请求的问题（条数、邮箱格式、本批重复、角色）在事务之前，有就一个 422 列出它们；依赖数据库的问题（有效成员的邮箱、已有邀请）在锁和判定之后，一个 422 列出它们。一批同时有两类问题时只得到第一类。理由与 P2 的裁定 (d) 相同（M3 设计 6.7、3.6 约定二已照它改）：值的校验在事务之前，不透露工作区的任何事；依赖行的检查在判定之后，没有权限的人得不到目标的任何信息。每一步里全部问题一起回答，"整批全有或全无"不变。W4 因此把"本批重复"放在单独的一次请求里。若接受，建议 M3 设计 3.8 的那一句加上"只看请求的问题先单独回答（6.7 第 0 步）"。
4. **创建在事务之前读时钟**（P2 review FW-2）。P2 spec 2.6 的规则"时钟在锁之后读"是为了修改已有行的写：排在别的写后面的写，存下的 `updated_at`、`deleted_at` 不会早于它等的那一次。创建插入的是新行，没有它等的写：与之竞争的另一批要么在它之前提交（它的插入 23505，整批回滚），要么在它之后；凭证按请求的时刻复核，与 M2 签发令牌相同。`TestCreatingInvitationsReadsTheClockBeforeItsTransaction` 钉住：时钟读一次、在事务的第一个调用（凭证锁）之前。若要按锁之后读，改 `create_invitations.go` 一处和这个测试。
5. **`Keys.MAC` 拒绝刷新令牌的用途**；`identity.Deps.SigningKeyPEM` 换成 `Deps.Keys`。6.6 第 1 步让 `LoadKeys` 返回的值"只提供 `identity.New` 要的 JWT 和刷新令牌的 MAC，以及按用途派生的 MAC"。按用途派生的 MAC 对别的模块开放，本 Phase 让它拒绝 `"refresh-token"`：否则拿到 `Keys` 的代码能算出刷新令牌的标签。刷新令牌的 MAC 由 `identity.New` 从未导出的 `signing.Keys` 直接取。派生本身（11.1）不变，已知答案证明刷新令牌的标签与 M2 相同。
6. **公开的查看在矩阵中：操作级的豁免加一个测试**（P1 review 第 6 节）。矩阵的列都是带凭证的身份，加一列"不带令牌"要给其余 22 行都填这一格（全是 401，整程序测试已覆盖）。本 Phase 改为：`matrixExemptions.public` 列出操作和代替它的测试名，完整性核对要求它存在于契约、确实公开、没有行；`TestTheInvitationLinkAnswersEveryCallerAlike` 对不带令牌和六列的令牌逐个请求，答案都相同（2.14）。豁免里的测试名只是指向，不被核对（第 6 节）。
7. **`notTargets`**（P2 re-review Minor 2）：见 2.14。接受、忽略是账户级的操作，列在 `notTargets` 里而不是豁免：它们仍有行，只是格子不按列指向工作区。
8. **`pgtest.WaitForKeyWaitOn`**：`platform` 的测试工具加一个函数（只给测试用，`testHelpersOnlyInTests`）；`WaitForLockWait`、`WaitForLockWaitOn` 不变。它分不清的两种情形写在文档里（2.13），只在交错 18 用。M3 设计 9.3 的交错引言说"`ON CONFLICT` 等同一个键的插入"用不分表的 `WaitForLockWait`（只用在没有别的语句能等的库上）；那句话仍然成立，P3 按 brief 用了只看这种等待、分表的探针。若接受，建议 9.3 的引言在那里写出 `WaitForKeyWaitOn`。
9. **P2 的闸门加期限**（附录 A 的 A4）：`bootstrap/interleaving_test.go` 的 `gate` 是 P2 的共用测试工具，本 Phase 给它 10 秒的期限。没有期限时，把创建挪到事务之外的变异让交错 18 挂住 10 分钟（缺陷类别"测试挂住"）。
10. **原型的变异发现、本 plan 已补上的测试**（附录 A 的修订）：A1 `TestTheInvitationMACIsTheDesigns`（组合根要的用途换了没有测试看得到）；A2 矩阵的相同性测试按链接的种类而不是令牌的值分组（去掉消息里的 id 时每个邀请的令牌相同，原来的分组把它们当成同一个链接）；A3 创建的时钟测试数事务的调用（原来只看时钟读了几次）；A5 `LockInvitation` 的失败测试在 `BEGIN` 之后才取消 ctx（原来 `BEGIN` 先失败，锁的语句没有运行）；A4 见第 9 条。另外改正了 W6 的注释引用的设计节号（3.9 → 3.8、决策点 1）。
11. **一个等价变异**："修改答锁到的那一行加上新角色，而不是 `RETURNING` 的行"：`WorkspaceInvitation` 没有 `updated_at`、`updated_by_id`，锁着的行除了角色没有别的列变化，两者在真实数据库上相同。没有测试区分它们，也不需要。
12. **表里没有令牌列只由 W4 看到**：迁移的测试钉住约束和索引的名字与种类，不钉列；`expectInvitations` 核对 11 列恰好是 4.4 的列、任何一行都不含令牌。加一个令牌列的变异只让 W4 失败（附录 A）。
13. **3.20 的"总体设计 4.2"一行**：设计写"4.2：'被邀请的邮箱始终可以注册'改为……"，这句话在总体设计 1.1 的"登录方式"一段，不在 4.2。Task 15 在 1.1 改它；4.2 另写约定六的接受一段（3.20 的 P2 一行要求 P3 核对）。
14. **`apitest` 的请求体用例移到 `bodycases.go`**（说明，2.5）：P3 让 `operations.go` 长到 411 行；请求体用例是单独的一件事，移到自己的文件，只移动不改写。`workspace/app/fakes_test.go`（P2 的共用假实现）因 P3 的 10 行到 401 行，留给 P4（它还要加项目的连带）按用例拆开。

## 4. 验收标准（完成线，M3 设计 12 节 P3）

- [ ] W3、W4、W5、W6 的接口版本和加了第二个成员的 W8 通过，此前的 53 个故事仍然通过（共 57 个）。
- [ ] 交错 3、9、12、18、19 在真实数据库上两个顺序都通过，`-count=5 -race` 也通过，都没有 40P01；去掉锁、锁在事务之外的变异都让它们失败，不挂住。
- [ ] 接受的用例测试覆盖已是有效成员（角色高、低、相同）、以前的成员行、没有成员行三种（9.1）。
- [ ] 邀请的唯一冲突翻译为 422、下标是请求中的下标的测试通过（用例测试和交错 18）。
- [ ] 对象的数组的请求体用例通过（`apitest` 和整程序测试）。
- [ ] 本 Phase 的矩阵格子（60 个）通过，共 132 格，耗时记下；公开的查看由它的测试覆盖；完整性核对、答案的计数通过。
- [ ] 令牌：已知答案（三处）、每一位都算、别的邀请和别的用途的令牌不通过、常量时间的比较、访问日志没有 `token=` 的测试通过。
- [ ] 9 个迁移 up、down、再 up 通过；新表的名字、种类、CHECK、部分唯一键由测试核对。
- [ ] `make lint`、`make test`、`make gen-check`、`make knip`、`make test-web`、`make e2e` 通过。
- [ ] 3.20、8.7 中 P3 的各行（总体设计 1.1、4.2，差异清单，README）写好；M2 收尾交接有"处理结果（M3/P3）"。

## 5. 不在 P3 范围内

- 邀请页、注册页、成员页的邀请界面（W4–W6 的页面版本）：P9。
- 恢复成员关系时的 `DemoteToGuest`，删除工作区连带项目：P4。
- 移出、离开时删除发给他的待接受邀请（3.8"结束的成员关系不留下邀请"），`EndMemberships`，交错 1、4、5、6，W12 的旧邀请：P5。
- 停用删除发给他的全部邀请（含已忽略的），交错 7，交错 19 的真实停用：P6。
- `PROBLEM_MESSAGES` 移到通用的位置：P8。

**留给后面的 Phase**（各 Phase 的 spec 接过去）：

| Phase | 条目 |
|---|---|
| P4 | `AcceptWorkspaceInvitation.Execute`：恢复已结束的成员关系、邀请的角色是访客时，在 `RestoreMember` 之后、`AcceptInvitation` 之前调 `DemoteToGuest`（同一个事务，注释写明位置）；`cascade()` 在最后加项目；矩阵每行带自己的列 |
| P5 | `EndMemberships` 在同一个事务里、工作区 N 之下、改成员行之前删除发给他邮箱的待接受邀请（邮箱经 `MemberProfiles`）；矩阵"已被移出"一列的 SQL 换成存储；9.3 的"邀请从不改变有效的成员关系"（Codex S2）与"结束的成员关系不留下邀请"（spike 9d）两个集成测试；W12 |
| P6 | 交错 19 换成真实的停用再跑一次（`gatedDeactivation` 删除）；停用删除邀请用 `workspace_member_invites_email_idx`；交错 7 |
| P9 | W5、W6 的页面版本（M2 收尾交接第 1 节的页面一侧）；W4 的页面版本（复制链接） |

## 6. 风险

| 风险 | 应对 |
|---|---|
| `WaitForKeyWaitOn` 分不清同一个事务在别的表上等键、以及不取 tuple 锁的两种行等待 | 文档写明；只用在交错 18，那里等待的一方只写邀请这一张带唯一键的表、不升级共享锁；四个削弱它的变异都被 `TestWaitForKeyWaitOnSeesOnlyAKeysWaitOnItsTable` 发现 |
| 常量时间的比较测不出来 | `TestMACVerifyComparesInConstantTime` 解析 `mac.go`，要求 `Verify` 的最后一句是 `return hmac.Equal(…)`；把比较挪进辅助函数的重构要同时改这个测试 |
| 豁免里的测试名不被核对：改名之后留下过时的指向 | 名字只是指向，删除那个测试会让 `getWorkspaceInvitation` 失去它的覆盖而没有测试失败。P3 之后若再有公开操作，考虑让完整性核对解析测试文件、要求这个名字存在（需要解析源文件，本 Phase 不做） |
| 表里加一个令牌列只有 W4 看得到（第 3 节第 12 条） | W4 在持续集成的 `make e2e` 里；迁移测试若以后钉列的清单，可以改由 Go 测试看到 |
| 没有密钥文件的 nerve 各有自己的临时密钥：一个进程发的链接在别的进程上无效 | 部署要求全部进程共用一个密钥文件，写在 README（8.7）；W6 全部经第二个 nerve（9.6）；`LoadKeys` 的警告提到邀请链接 |
| 令牌在链接的查询参数里，反向代理的访问日志会记下 | nerve 的访问日志只记路径（测试核对）；README 写明代理的日志要像密钥一样保管或不记查询参数 |
| 交错 19 的停用一方是测试按 3.9 的顺序取锁，不是 P6 的实现 | P6 换成真实的停用再跑一次（9.3、第 5 节） |
| plan 的最大 Task 接近上限 | Task 11 是 1,415 行；它的存储已拆成 Task 10，再拆会把接受的用例和它的 HTTP、矩阵行分到两个 Task，前一个的操作在契约里没有 handler，不能各自绿 |
| 持续集成（ubuntu）上的运行 | 原型和复现都在 macOS 上；P2 在两处的结果相同 |

## 7. 交接和关闭条件

| 交接 | P3 处理的条目 | 留下的条目 |
|---|---|---|
| M2 收尾交接第 1 节 邀请与注册 | 接口一侧完成（Task 6–12、14）；"处理结果（M3/P3）"（Task 15） | 页面一侧：P9；本节保持 `open` |
| P2 review 第 6 节（P3 的各件） | 全部，落点见第 3 节第 2 条 | — |
| P1 review 第 6 节 公开操作进矩阵 | 操作级的豁免加 `TestTheInvitationLinkAnswersEveryCallerAlike`（Task 9） | — |

**M3 设计 13.1 的关闭条件**（M2 收尾交接第 1 节，P3 的接口一侧），逐条核对：

| 条件 | 落点 |
|---|---|
| 负责人已裁定（第 10 节） | 决策点 1、2、4 照裁定实现（2.8–2.12） |
| 只凭邮箱的接受被拒绝、凭有效令牌且邮箱一致的成功 | 接受要令牌：请求体不带是 400（W5、`TestAnsweringAWorkspaceInvitation`），令牌不对 404（W5、`TestAcceptWorkspaceInvitationRefusals`、矩阵的相同性测试），邮箱不一致 403（W5、矩阵 12 格）；令牌对、邮箱一致 200（W5、矩阵 6 格） |
| 接受最先锁住接收账户的行 | `TestAcceptWorkspaceInvitation` 的调用记录（账户 S 在一切之前）；交错 12 两个顺序；交错 19 |
| 已是有效成员时不改成员关系 | `TestAcceptWorkspaceInvitation`（9.1 的三种） |
| 注册关闭时带有效邀请的注册成功、不带的 `identity.signup_disabled` | `TestRegisteringWithAnInvitationWhileSignupIsOff`、W6 |
| W5、W6 两个版本（P3 接口、P9 页面） | 接口版本：Task 14；页面版本：P9 |

接口一侧的条件都有落点，没有放不下的。

## 附录 A：原型验证记录（2026-09-30）

原型在 `$M3TMP/p3proto`（`1d2eaf7c` 的副本，Go 1.27.1、Node 24、pnpm 11.10.0、Docker；`bin/golangci-lint` 2.13.2）。做法照 P2：每个 Task 做完时存一份源文件的快照（`$M3TMP/p3snapF/T1`…`T15`），plan 的代码块由脚本从相邻两份快照的差异生成，脚本把每个 Task 的块应用到前一份快照上，结果与这一份逐字节相同；生成的文件不进块，按 SHA-256 核对。原来 14 个 Task 中两个超过约 1,500 行，拆开时在原型上重建了两份中间状态（Task 7：修改、删除的存储；Task 10：接受、忽略的存储），各自 `make gen-go`、`make lint-go`、`make test` 通过之后才存快照。

**逐 Task 复现**（`$M3TMP/p3tools/replay.py`，日志在 `replay-logs`）：在 `$M3TMP/p3replay`（`1d2eaf7c` 的另一个副本，`pnpm install --frozen-lockfile`）按 plan 的 15 个 Task 依次执行：`planapply.mjs` 从 plan 的文本中取出这个 Task 的块写入，然后按顺序执行这个 Task 的每一条 `Run:` 命令，原样照 plan。例外只有：`make gen`、`make gen-go` 之后核对全部生成物（21 个）与这个 Task 的快照逐字节相同；`shasum -a 256` 的输出与 plan 表中的 SHA-256 和行数核对；副本不是 git 仓库时 `make lint-web` 的关键词守卫由 `kwcheck.mjs` 代替（同样的规则、同样的文件）；`make e2e` 之前把副本初始化为 git 仓库并提交（F8）；提交之后的 `make gen-check` 由生成物的核对代替。每个 Task 另核对 `go.mod`、`go.sum`、`server/tools` 和 `pnpm-lock.yaml` 不变。

| Task | 复现的结果 |
|---|---|
| 1 | 写入 16 个文件；`make gen-go`：21 个生成物与快照相同，2 个 SHA-256 与表相同；`./migrations/`、`workspace/...`、`TestTheGrantsFileCoversEveryRelationAndFunction`、`TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt` ok；lint 2 × `0 issues.`；`make test` 38 个 `ok` |
| 2 | 10 个文件；`identity/...`、`workspace/domain` ok；lint；`make test` 38 `ok` |
| 3 | 5 个文件；`apitest` ok；`TestParametersThatDoNotBindAnswer400`、`TestBodiesThatBreakTheStructureAnswer400` ok；lint；`make test` 38 `ok` |
| 4 | 3 个文件；`identity` ok；lint；`make test` 38 `ok` |
| 5 | 26 个文件；`make gen`：21 个生成物相同，4 个 SHA-256 相同；`workspace/...`、`access/...` ok；`TestTheInvitationMACIsTheDesigns`、矩阵、完整性和规则表 ok；lint；`make test` 38 `ok`；`make lint-web`（关键词守卫、turbo 54 个任务）、`make knip`、`make test-web`（16 个任务）通过 |
| 6 | 19 个文件；`make gen`：4 个 SHA-256 相同；各包、矩阵和完整性、`TestBodiesThatBreakTheStructureAnswer400` ok；lint；`make test` 38 `ok`；前端检查通过 |
| 7 | 6 个文件；`make gen-go`：2 个 SHA-256 相同；存储 ok；lint；`make test` 38 `ok` |
| 8 | 26 个文件；`make gen`：4 个 SHA-256 相同；各包、矩阵和完整性 ok；lint；`make test` 38 `ok`；前端检查通过 |
| 9 | 20 个文件；`make gen`：4 个 SHA-256 相同；`workspace/...` ok；访问日志、相同性、矩阵和完整性、公开操作、查询参数的整程序测试 ok；lint；`make test` 38 `ok`；前端检查通过 |
| 10 | 5 个文件；`make gen-go`：3 个 SHA-256 相同；存储 ok；lint；`make test` 38 `ok` |
| 11 | 25 个文件；`make gen`：4 个 SHA-256 相同；`workspace/...` ok；矩阵、完整性、相同性、请求体 ok；`-v`：`TestPermissionMatrix` 1.33 秒、`prepare` 0.07 秒；lint；`make test` 38 `ok`；前端检查通过 |
| 12 | 16 个文件；`make gen`：4 个 SHA-256 相同；`identity/...`、`workspace/...` ok；注册、策略、请求体、访问日志的整程序测试 ok；lint；`make test` 38 `ok`；前端检查通过 |
| 13 | 5 个文件；`pgtest` ok；交错 3、9、12、18、19 和 P2 的交错 2 `-count=5 -race` ok（10 秒）；lint；`make test` 38 `ok` |
| 14 | 7 个文件；lint；`make test` 38 `ok`；前端检查通过；`make e2e` 57 个通过 |
| 15 | 4 个文件；lint；`make test` 38 `ok`；`make lint-web`（副本此时已是仓库，照原样执行）通过 |
| 结束 | 复现的树与原型逐文件相同（2,836 个文件，0 个差异）；提交之后的副本上 `make gen-check`、`make lint-web` 通过；`planapply.mjs check` 从 `1d2eaf7c` 起 380 个块全部通过 |

| 核对 | 命令 | 结果 |
|---|---|---|
| 生成物 | 原型上 `make gen-check` | 没有差异 |
| Go 静态检查 | `make lint-go` | server 0 issues；server/tools 0 issues |
| Go 测试 | `make test`（testcontainers，`postgres:18.6`） | 全部通过（37 个包加 `server/tools` 的 1 个） |
| 前端检查 | `make lint-web`（关键词守卫 60 条规则、turbo 54 个任务）；`make knip`；`make test-web`（16 个任务） | 全部通过 |
| 端到端 | `make e2e` | 57 个全部通过（此前的 53 个、W3、W4、W5、W6；W8 加了第二个成员） |
| 迁移 | `TestMigrationsGoUpDownAndUpAgain`（9 个迁移） | up、down、再 up 通过 |
| 矩阵 | `go test -count=3 -v -run 'TestPermissionMatrix$'` | 132 格；整个矩阵 1.31、0.23、0.24 秒（第一次最慢），准备 0.06–0.08 秒；9.2 的预算是 20–30 秒 |
| 交错 | `go test -count=5 -race -run 'TestAcceptingAndDeletingTheWorkspace\|TestInvitingAndResettingThePassword\|TestAcceptingAndChangingTheAddress\|TestInvitingOverlappingBatches\|TestDecliningAndDeactivating' ./internal/bootstrap/` | 通过（7.8 秒），没有 40P01 |
| 已知答案 | `$M3TMP/p3tools/kat.py`：Python 的 `hmac`、`hashlib` 按 RFC 5869 手写 HKDF，取 `testKeyPEM` 的种子 | 刷新令牌 `65cc0852…4468`（与 M2 的代码相同），邀请 `92fab228…e894`、`nrv_inv_kvqyKBh-bANMT6JAIYzolA` |
| 变异 | `$M3TMP/p3tools/mutants.py`（139 个，Go 测试）、`e2e_mutants.py`（2 个，单独运行一个故事） | 全部被点名的测试发现（下表） |

**原型中定下的事实**：

- **F1** 唯一索引上的等待没有 tuple 锁：后到的 `INSERT` 等先到的事务的 `transactionid`，`pg_locks` 里它只有表的 `RowExclusiveLock` 和对那个事务 id 的等待。`WaitForLockWaitOn` 看不到，`WaitForLockWait` 看得到但不分种类。所以加 `WaitForKeyWaitOn`（2.13），正例和三个反例由测试证明。
- **F2** 端到端的每个 nerve 在没有密钥文件时各有临时密钥：主 nerve 算出的令牌在第二个 nerve 上无效（9.6 预见了）。W6 的工作区、邀请、回应都经第二个 nerve，用 PAT（令牌在库里，两个 nerve 都认）。
- **F3** 只看请求的问题先单独回答（第 3 节第 3 条）：W4 原来把本批重复和已邀请的邮箱放在一次请求里，只得到前者；拆成两次。
- **F4** 已接受的邀请在接受时软删除，早于工作区的删除：W3 原来要求"每一行随工作区同一时刻删除"而失败。`expectWorkspaceDeleted` 改为：每张表至少一行随工作区删除，没有未删除的，也没有更晚删除的。
- **F5** 把创建挪到事务之外的变异让交错 18 的闸门拿到 `context.Background()`，测试挂到 10 分钟的超时；P2 的 `gate` 加期限之后它在 33 秒内失败（第 3 节第 9 条）。
- **F6** 已取消的 ctx 让 `BEGIN` 先失败：`LockInvitation` 的失败测试原来到不了锁的语句，"失败答成没有"的变异留下。改为在 `BEGIN` 之后取消（A5）。
- **F7** `RawURLEncoding` 不带 `Strict()` 时，最后一个字符的四个填充位被忽略：同一个标签有 16 种写法。`Strict()` 之后一个标签恰好一个令牌（`TestEveryBitOfATokenCounts` 逐位翻转）。
- **F8** 原型不是 git 仓库时 S3（实例信息里的 `commit`）失败（P2 附录 A 的 F2）；复现在 `make e2e` 之前把副本初始化为仓库并提交。

**变异核对**（`mutants.py`：改一处代码，跑点名的测试，恢复；期望这些测试失败并且输出含点名的失败行；生成的查询常量代表它的查询）。第一轮（按类别分批跑）130 个中 115 个按期望被发现。其余 15 个：5 个是测试的缺口，补上测试之后都被发现（A1–A5，第 3 节第 10 条；其中 A4 是测试挂住）；2 个是期望写错（公开查看的"令牌不对答 403"由另一个测试名发现；`Verify` 只比一半的期望多列了一个看不到它的包）；6 个变异本身编译不过（去掉一个用法留下未用的变量），改写成能编译的同一个缺陷之后都被发现；1 个是等价变异（第 3 节第 11 条），从表中去掉；1 个（加令牌列）只有 W4 看得到，移到 `e2e_mutants.py`。之后补了 3 个（角色按大小、创建答自己拼的行、组合根的策略总是开放）和 Task 3 的 8 个（`apitest`、生成的 `bodyshape`）。修订之后在最终的原型（的副本）上一次跑完：**Go 139 个全部被发现（398 秒；`logs/mutants-final2.txt`、`mutants-all-final2.json`），端到端 2 个全部被发现**；请求体用例移到 `bodycases.go` 之后，Task 3 的 8 个在新文件上再跑一次，全部被发现。下表按 brief 的缺陷类别：

| 类别 | 变异 | 被哪些测试发现 |
|---|---|---|
| 令牌：别的邀请、别的工作区、别的用途 | 消息不含 id；只含 id 的一半 | `TestTheTokenOfTheKnownAnswer`、`TestInvitationMessagesDifferByEveryByteOfTheID`（前者另有 `TestTheInvitationLinkAnswersEveryCallerAlike`、查看和注册的用例测试） |
| | 消息不含标签 | `TestTheTokenOfTheKnownAnswer` |
| | 列出时给每个邀请它工作区 id 的令牌 | `TestListWorkspaceInvitationsListsEachWithItsToken`、`TestTheTokensAreTheMACs`、矩阵 |
| | 每个用途一个密钥；HKDF 只做 extract | `TestMACKnownAnswers`（前者另有 `TestMACsOfPurposesDiffer`、`TestKeysMACIsThePurposes`） |
| | 别的模块拿得到刷新令牌的 MAC | `TestKeysMACRefusesTheRefreshTokensPurpose` |
| | 组合根要的用途不是设计的 | `TestTheInvitationMACIsTheDesigns` |
| 令牌：改一位也接受 | 不带 `Strict()`；不看前缀；接受更长的 | `TestParseTokenRefuses`（第一个另有 `TestEveryBitOfATokenCounts`） |
| | 格式对的令牌都有效 | `TestGetWorkspaceInvitationRefusals`、`TestAcceptWorkspaceInvitationRefusals`、`TestSignupInvitationsRefuseEveryOtherCase`、`TestTheInvitationLinkAnswersEveryCallerAlike`、`TestRegisteringWithAnInvitationWhileSignupIsOff` |
| | `Verify` 只比一半 | `TestMACVerify` |
| 令牌：不是常量时间 | `Verify` 用 `==` | `TestMACVerifyComparesInConstantTime` |
| 令牌：存、记、回显 | 访问日志记查询参数 | `TestTheInvitationLinksTokenIsNotLogged` |
| | 表里加令牌列 | W4（`e2e_mutants.py`：只运行 W4 时失败） |
| 枚举 | 查看对令牌不对答 403；接受、忽略同样 | `TestGetWorkspaceInvitationRefusesAWrongTokenBeforeReading`、`TestTheInvitationLinkAnswersEveryCallerAlike`；`TestAcceptWorkspaceInvitationRefusals`、`TestDeclineWorkspaceInvitationRefusals` |
| | 查看先读邀请再核对令牌；回应在锁之后核对令牌 | `TestGetWorkspaceInvitationRefusesAWrongTokenBeforeReading`；`TestAcceptWorkspaceInvitationRefusals` |
| | 注册说出是哪一种无效（邮箱不同答 500）；已忽略的、任何邮箱、不要令牌的邀请让注册通过 | `TestSignupInvitationsRefuseEveryOtherCase`、`TestRegisteringWithAnInvitationWhileSignupIsOff` |
| | 读邀请失败当作拒绝 | `TestSignupInvitationsRefuseEveryOtherCase` |
| | `InvitationByID` 读已删除的（已接受、已删除的邀请让注册通过） | `TestRegisteringWithAnInvitationWhileSignupIsOff`、`TestTheInvitationReadsFindOnlyAnUndeletedInvitation` |
| | 开放时也问邀请；关闭时不带邀请也通过 | `TestSignupPolicy`（后者另有整程序测试） |
| | 问策略用输入原样的邮箱；handler 丢掉邀请 | `TestRegisterAsksThePolicyAboutTheAddressAndTheInvitation`、`TestRegisterHandsOnTheInvitation`，都另有整程序测试 |
| | 组合根的策略总是开放 | `TestRegisteringWithAnInvitationWhileSignupIsOff` |
| 公开的查看 | 组合根漏掉公开操作 | `TestPublicOperationsAreTheContractsPublicOperations`、`TestTheInvitationLinkAnswersEveryCallerAlike` |
| | 显示已接受或已删除的邀请、已删除工作区的邀请；从不说已忽略；读失败答 404 | `TestInvitationPreview` |
| 接受：账户行没有最先锁 | 账户在工作区和邀请之后锁 | `TestAcceptWorkspaceInvitation`、`TestDeclineWorkspaceInvitation`、`TestDecliningAndDeactivating` |
| | 账户在事务之外读（不持锁）；读账户不加 `FOR SHARE` | `TestAcceptingAndChangingTheAddress`、`TestDecliningAndDeactivating` |
| 接受：停用的账户、别的邮箱、再次回应 | 停用的账户可以回应 | 两个用例的拒绝测试、`TestDecliningAndDeactivating` |
| | 不比较邮箱 | 两个用例的拒绝测试、`TestAcceptingAndChangingTheAddress`、矩阵 |
| | 已忽略的可以再回应；别的邮箱被告知"已回应" | 两个用例的拒绝测试 |
| | 锁下不重读邀请 | `TestAcceptWorkspaceInvitationRefusals`、`TestUpdateWorkspaceInvitationRefusals` |
| 接受：有效的成员关系、恢复的行 | 给有效成员邀请的角色；回答邀请的角色 | `TestAcceptWorkspaceInvitation` |
| | 以原来的角色恢复；给以前的成员再插一行 | `TestAcceptWorkspaceInvitation` |
| | 新成员关系记成邀请人的；接受写到工作区的 id | `TestAcceptWorkspaceInvitation` |
| | `RestoreMember` 保留原角色、不恢复、恢复每一行 | `TestRestoreMember` |
| | `MemberOf` 读已删除的、别的工作区的 | `TestMemberOf` |
| | 接受的邀请不删除；`AcceptInvitation`、`DeclineInvitation` 改每一行；忽略时删除 | `TestAnsweringAnInvitation`（第一个另有交错 12） |
| | 回答的人数含已结束的 | `TestWorkspaceByID` |
| 接受：锁的强度、锁在事务之外 | 忽略用 N；接受用 S | 两个用例测试 |
| | 接受不锁工作区；接受、忽略的锁在事务之前 | 用例测试、`TestAcceptingAndDeletingTheWorkspace`（接受）、`TestAcceptingAndChangingTheAddress`、`TestDecliningAndDeactivating` |
| 创建 | 按请求的顺序插入 | `TestCreateWorkspaceInvitationsLocksDecidesChecksThenInserts`、`TestInvitingOverlappingBatches` |
| | 23505 答 409；下标取排序后的位置 | `TestADuplicateFromTheUniqueKeyNamesTheRequestsIndex`、`TestInvitingOverlappingBatches` |
| | 存储跳过被占的邮箱、插入其余的 | `TestCreateInvitationsRefusesAnAddressTaken`、`TestInvitingOverlappingBatches` |
| | 插入在事务之外 | `TestInvitingOverlappingBatches`（闸门的期限之内失败） |
| | 已忽略的邀请不占邮箱（用例一侧；唯一索引一侧） | `TestCreateWorkspaceInvitationsRefusesMembersAndInvitedAddresses`；`TestUniqueKeysHoldAmongUndeletedRowsOnly` |
| | 不查有效成员的邮箱；把已结束的也算上 | `TestCreateWorkspaceInvitationsRefusesMembersAndInvitedAddresses` |
| | 不锁邀请人的账户；在事务之外锁 | `TestInvitingAndResettingThePassword`（前者另有用例测试） |
| | 工作区用 N（两批在行上等而不是在键上） | 用例测试、`TestInvitingOverlappingBatches` |
| | 角色按大小（5 到 20） | `TestCheckInvitationsRefuses` |
| | 提供的凭证锁什么都不锁、不查 | `TestProvidedCredentialLockIsTheCredentialLock` |
| 先判定、再锁 | 修改、删除按锁之前读到的判定 | 两个用例测试 |
| | 修改不锁；工作区用 N | `TestUpdateWorkspaceInvitationLocksThenDecidesThenWrites` |
| | 已忽略的邀请可以改角色；成员被告知已忽略 | `TestUpdateWorkspaceInvitationRefusals` |
| 锁在事务之外 | 修改、删除的读和锁在事务之前 | 两个用例测试 |
| 锁没有 `deleted_at IS NULL`、强度不对 | `LockInvitation` 去掉条件 | `TestTheInvitationReadsFindOnlyAnUndeletedInvitation`、`TestLockInvitationSkipsAnInvitationDeletedWhileItWaits` |
| | `LockInvitation` 用 `FOR SHARE`、`FOR NO KEY UPDATE` | `TestLockInvitationLocksTheRowForUpdate` |
| | `InvitationByID` 读已删除的 | `TestTheInvitationReadsFindOnlyAnUndeletedInvitation` |
| | `ShareWorkspace` 去掉条件；用 N；用 `FOR UPDATE` | P2 的三个锁测试（最后一个另有 `TestAForeignKeyCheckDoesNotWaitForTheWorkspaceLocks`） |
| 存储的失败分支 | 改角色、删除、接受、忽略、恢复吞掉错误；`LockInvitation`、`MemberOf`、`WorkspaceByID` 把失败答成"没有" | 各自的存储测试（`TestUpdateInvitationRole`、`TestDeleteInvitation`、`TestAnsweringAnInvitation`、`TestRestoreMember`、`TestTheInvitationReadsFindOnlyAnUndeletedInvitation`、`TestMemberOf`、`TestWorkspaceByID`） |
| | 失败的读、锁答成邀请的 404 | 三个用例的拒绝测试（另有忽略、查看的） |
| 只有一行 | `UpdateInvitationRole`、`DeleteInvitation` 改每一行 | `TestUpdateInvitationRole`、`TestDeleteInvitation` |
| | `ListInvitations` 列已删除的、每个工作区的、只按 id | `TestListInvitations` |
| 假实现 | 创建答它自己拼的行（时钟的时间），不是存下的 | `TestCreateWorkspaceInvitationsLocksDecidesChecksThenInserts`（假存储答的时间与时钟不同） |
| | 修改按工作区的 id 锁邀请 | 修改、删除、接受、忽略的用例测试（假存储按参数回答） |
| 时钟 | 修改、删除、接受、忽略在锁之前读 | `TestEachWriteReadsTheClockUnderItsLock` |
| | 创建在凭证锁之后读 | `TestCreatingInvitationsReadsTheClockBeforeItsTransaction` |
| 连带 | 不删邀请；删每个工作区的；再删已删除的 | `TestDeleteWorkspaceLocksThenDecidesThenCascades`、`TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt`；`TestDeletingAWorkspaceSoftDeletesItsRows`（后两个） |
| 目录只钉名字 | 唯一索引去掉 `WHERE`；去掉回应的 CHECK；角色的 CHECK 放宽；邮箱允许空白；外键去掉 `CASCADE` | `TestConstraintAndIndexNames`、`TestUniqueKeysHoldAmongUndeletedRowsOnly`、`TestChecksRejectCounterexamples`（各自的） |
| 规则表 | 列出、创建、修改给成员（放宽） | `TestEveryRuleDecidesItsCells`、矩阵 |
| | 加一行而没有格子 | `TestEveryRuleDecidesItsCells`、`TestEveryActionHasARuleAndEveryRuleAnAction` |
| 矩阵 | `targetViolation` 放过不认识的参数 | `TestMatrixViolationsCatchesEachGap` |
| 码只在别的包里返回（9.4） | HTTP 测试不答 `workspace.invitation_email_mismatch` | `apitest.Main`：`declares problem codes that no test answered` |
| 探针被削弱 | 数任何表的键等待；数行等待；数任何等待；数只读过表的事务 | `TestWaitForKeyWaitOnSeesOnlyAKeysWaitOnItsTable` |
| `apitest` 被削弱 | 不给对象数组的元素用例；不给元素缺少必填属性的用例；"全部问题一起"不含元素；字符串的数组也算 | `TestBodyCasesOfAListOfObjects` |
| | 必填的查询参数没有"不带它"的用例；可选的也有 | `TestParamCases`（后者另有 `TestParametersThatDoNotBindAnswer400`） |
| 接线没人看（请求体的结构） | 生成的 `bodyshape` 让邀请的元素接受未声明的属性；不要求任何属性 | `TestBodiesThatBreakTheStructureAnswer400` |
| 端到端单独运行看得到 | `Preferences` 读工作区的第一行，不看账户（P2 的风险） | W8（`e2e_mutants.py`：只运行 W8 时失败） |

**缺陷类别**（brief 列出的，逐类核对）：

| 类别 | 核对了什么 | 结果 |
|---|---|---|
| 测试在它说的性质去掉之后仍通过 | 上表每一类 | 139 + 2 个变异全部被发现 |
| 令牌的四种 | 别的邀请、别的用途、改一位、非常量时间、存储和日志 | 全部被发现；已知答案不经 Go 的代码算出 |
| 枚举 | 查看、回应、注册的每种无效都是同一个回答，逐字节比较 | 全部被发现 |
| 接受的七种 | 账户最先锁、锁下读、有效成员不变、恢复取邀请的角色、邮箱不一致不写、已回应不再接受、忽略的锁强度 | 全部被发现 |
| 创建的六种 | 部分插入、排序、23505 的码和下标、已忽略的占邮箱、经 `MemberProfiles`、角色 | 全部被发现；"角色高于邀请人"由规则表的变异代表（只有管理员邀请） |
| 存储的失败分支 | 每个新的读写在失败时答错误；锁的失败在 `BEGIN` 之后强制 | 第一轮留下 1 个（A5），补测试之后全部被发现 |
| 假实现忽略参数、答时钟的时间 | 假实现按参数回答，存下的时间与时钟不同 | 两个变异被发现 |
| 精确的错误 | 用例、HTTP、存储测试用 `errors.Is` 比较具体的错误；失败不是 404、不是 422 另外断言 | 失败答成 404 的变异被发现 |
| 断言不可能失败 | 矩阵每格断言状态码和码；答案的核对计数；相同性测试逐字节比较 | 相同性测试原来按令牌的值分组，第一轮留下"消息不含 id"（A2），改正之后被发现 |
| 只有一行 | 存储测试都有第二个工作区、第二份邀请、一行已删除的；W8 有第二个成员 | `WHERE` 的变异全部被发现；W8 单独运行看得到 `user_id` |
| 目录只钉名字 | 名字加种类；部分唯一键的行为测试 | 全部被发现 |
| 按大小比较角色 | `CheckInvitations` 的表含 0、10、16、25 | 被发现 |
| 测试挂住、闸门在争用区段之外 | 每个等待 10 秒为限（闸门加期限）；闸门在持锁的事务里；探针确认对方等在设计说的那一处 | 去掉锁、锁在事务之外都让交错失败，最慢 33 秒 |
| 说明与代码不符 | 接口描述、代码注释中的设计节号、文档 | 原型中改正：W6 的节号；"创建在事务之前读时钟"的注释引用 P2 spec 2.6 并说明它为什么不适用 |
| 接线没人看 | 组合根的 MAC 用途、策略、公开操作、`MemberProfiles` 在创建里的接线 | 四个变异被发现（`TestTheInvitationMACIsTheDesigns`、整程序测试、矩阵的两行） |
| 码只在别的模块的测试包里返回 | `workspace` 的 HTTP 测试返回它声明的每个码 | 删掉一个的变异被 `apitest.Main` 发现 |

**没有证明的**：

- 持续集成（ubuntu）上的运行：原型和复现都在 macOS 上。
- 交错 19 的真实停用（P6）；恢复时的 `DemoteToGuest`（P4）。
- W4–W6 的页面版本（P9）。
- 常量时间的比较只由读代码核对。
