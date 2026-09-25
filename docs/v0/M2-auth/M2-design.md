# M2 账户认证：设计与实施规划

| 项 | 内容 |
|---|---|
| 里程碑 | M2 账户认证（`docs/v0/M2-auth`） |
| 日期 | 2026-09-25 |
| 状态 | 草稿，待控制者评审。第 10 节的 3 个决策点待项目负责人裁定；第 11 节没有需要负责人裁定的架构问题 |
| 上级文档 | [v0 总体设计](../v0-design.md) 1.1、3、4、5、6、7、8、9 节；[差异清单](../plane-diff.md)；[前端改动清单](../frontend-changes.md) |
| 前置交接 | `handoffs/` 中的 10 份：M0-P1、M0-P2、M0-P3、M0-P4、M0-P5、M0-P6、M1-P2、M1-P3、M1-P4、M1-closeout。逐条落到 Phase，见第 13 节 |
| 设计评审 | 未开始 |

---

## 0. 目标与范围

### 0.1 目标
M2 是第一个做真实业务的里程碑，也是前端第一次对接新接口。M2 结束时：
1. 任何调用方都能通过接口注册、登录、续期、退出、修改密码，管理自己的资料、偏好和个人访问令牌（PAT）。用 PAT 能做的事和页面一样多。
2. 后端第一次有业务表（4 张）、第一批 sqlc 查询、第一个 River 定时任务（清理过期会话）。以下几件平台上的事一次定下，后续 M 照做：
   - 认证中间件和当前账户（Actor）的传递；
   - 限流；
   - 错误码体系和取值校验放在哪一层；
   - 事务；
   - 表结构约定；
   - oapi-codegen 的生成选项。
3. 前端用令牌管理器替换 Cookie 会话和 CSRF。认证、用户、资料、偏好、实例配置、PAT 这几块的 services、stores 改用 OpenAPI 生成的类型，没有转换层。
4. 本领域的每个用户故事都有端到端测试：页面、数据库、接口三层断言；接口版本用 PAT（第 2 节）。
5. 持续集成的全部门禁通过；本 M 的交接全部关闭；给 M3、M5、M8 的交接写清接收条件。

### 0.2 范围
| 包含 | 不包含（留给后续 M） |
|---|---|
| 注册、登录、续期（刷新令牌轮换和重复使用检测）、退出、修改密码 | 工作区、成员、邀请（M3）。其中包括："关闭注册时被邀请的邮箱仍可注册"；登录后落到哪个工作区；新手引导中创建或加入工作区、邀请成员三步 |
| 当前账户的资料（`/me`）和偏好（`/me/profile`，含新手引导的进度） | 头像和封面（M5）：`avatar_url`、`cover_image_url` 和上传，见 3.2 |
| PAT：创建、分页列出、撤销；用 PAT 调用所有接口 | 权限框架和权限矩阵（M3） |
| 实例配置（在 M0 的 `instance` 上扩展）和时区列表 | 接口调用日志（M8） |
| 管理命令：`nerve users reset-password`，以及决策点带来的命令（第 10 节） | 邮件、找回密码、邮箱验证（v0 不做，总体设计 1.2） |
| River 的第一个定时任务（清理过期会话）；sqlc 的第一批查询；第一次按快照建表的约定 | 实例管理员（v0 没有，见 3.16） |
| 前端：令牌管理器；登录页、注册页；认证包装；新手引导的资料步骤；个人设置的四个标签页；删除 CSRF、Cookie 和表单提交 | 刷新令牌改放 HttpOnly Cookie（只写下升级路径，见 8.5） |
| 端到端：认证 fixture、PAT 对等验收、数据库断言的写法 | 可注入的时钟（M4，M0-P6 交接） |

---

## 1. 现状基线（2026-09-25 实测，`main` 为 `559c3c6`）

| 项 | 数值与出处 |
|---|---|
| 服务端的模块 | 只有 `instance`（`server/internal/modules/instance/`）。还没有 `internal/shared` |
| 迁移 | 0 个。`server/migrations/sql/` 只有 `.gitkeep` |
| 接口 | 1 个操作：`GET /api/v0/instance`（`api/modules/instance.yaml`） |
| Go 依赖 | `server/go.mod` 没有 River、JWT，`golang.org/x/crypto` v0.55.0 只是间接依赖。`server/tools/go.mod` 只有 oapi-codegen v2.8.0 |
| 平台错误码 | `not_found`、`bad_request`、`internal_error`、`not_ready`（`server/internal/platform/httpserver/problem.go`） |
| 架构测试 | 10 条规则（`server/internal/archtest/rules_test.go`）。规则 6 只认 `adapter/http/gen`。平台包导入 `internal/shared` 不违反任何一条 |
| 前端的请求方式 | 两个 axios 基类：`web/apps/web/core/services/api.service.ts`（75 行，`withCredentials: true`，401 时 `window.location.replace`）和 `web/packages/services/src/api.service.ts`（106 行）。继承它们的 service 共 33 个（`rg -n "extends APIService" web` → 32 + 1）。全仓库没有任何地方带 `Authorization` 请求头 |
| CSRF 与表单提交 | 登录、注册：原生表单 POST 到 `/auth/sign-in/`、`/auth/sign-up/`（`web/apps/web/core/components/account/auth-forms/password.tsx:118-142`）。退出：临时拼表单（`web/apps/web/core/services/auth.service.ts:21-42`）。修改密码：`X-CSRFTOKEN` 请求头（`web/apps/web/core/services/user.service.ts:73-83`）。`/auth/` 地址共 4 处 |
| 生成的客户端 | `@nerve/api-client` 还没有被 web 导入（`rg -n "@nerve/api-client" web` 只命中它自己的 `package.json`） |
| M2 领域的前端代码 | 约 8,467 行，口径见 7.10 |
| store 的使用方 | `useUser()` 74 个文件，`useUserProfile()` 20 个，`useUserSettings()` 6 个，`useInstance()` 9 个 |
| 死成员和死 prop | `deadsym.mjs` → `prop 452, member 898, export 2`；`domains.mjs --rows M2` → 66 行（成员 59、prop 7），与 M1-closeout 交接一致 |
| oxlint | 共 694 条警告；按 `domains.mjs` 的划分，M2 领域 32 条，分布在 19 个文件 |
| Plane 的表 | `users` 40 列，`profiles` 29 列，`api_tokens` 17 列，`sessions` 5 列；`instances`、`instance_admins`、`instance_configurations` 已按差异清单一 B 不保留（`tools/plane-schema/plane-v1.4.2-schema.sql`） |

---

## 2. 用户故事（M2 的验收范围）

每个故事一个 Playwright 测试文件，放在 `e2e/stories/identity/`，按总体设计 8.2 在三个层面断言。

**接口版本的约定**：
- 认证本身的故事（A1–A6、A15）发生在拿到任何令牌之前，没有 PAT 可用。它们的接口版本直接调用认证接口，调用同一组数据库断言函数。
- 其余故事的接口版本一律用 PAT。
- 命令行和后台任务的故事（A13、A14）没有页面版本。

| 编号 | 故事 | 页面 | 数据库 | 接口版本 |
|---|---|---|---|---|
| A1 | 新用户注册 | 在 `/sign-up` 填邮箱、密码、确认密码，进入 `/onboarding` 的资料步骤。浏览器里没有任何 Cookie，localStorage 里有刷新令牌 | `users` 新增一行：邮箱已转小写；`password` 以 `$argon2id$` 开头；`display_name` 是邮箱 @ 之前的部分；`is_active`。`profiles` 新增一行，各列是默认值。`auth_sessions` 新增一行：`generation = 0`，记下 UA 和 IP，`expires_at` 约为 30 天后，未撤销 | `POST /api/v0/auth/register`，同一组断言 |
| A2 | 注册被拒绝 | 邮箱已存在：错误就地显示在表单上方，不跳转，邮箱仍在输入框里。密码不合规：字段下方显示规则。关闭注册（`signup_enabled = false` 的独立 nerve）：页头没有"注册"链接；直接打开 `/sign-up` 提交，显示"注册已关闭" | 三种情况都没有新增账户、资料和会话 | 409 `identity.email_taken`；422 `validation_failed`（`errors[].field = password`）；403 `identity.signup_disabled` |
| A3 | 登录与 `next_path` | 未登录打开 `/settings/profile/general?tab=x#y`，跳到登录页，地址带编码后的 `next_path`。登录后回到原地址，查询参数和片段都在。`next_path` 为 `//evil.example`、`/\evil`、`javascript:…`、带控制字符时，登录后落到默认页。错误密码和不存在的邮箱：页面显示同一句提示 | 成功时新增一行会话；失败时没有 | `POST /api/v0/auth/login`；两种失败都是 401 `identity.invalid_credentials` |
| A4 | 续期与多标签页 | 访问令牌有效期 3 秒的独立 nerve。同一个浏览器上下文开两个标签页，过期后同时操作：两边都成功，没有跳到登录页；两次续期请求在时间上不重叠（`navigator.locks`） | 会话未被撤销；`generation` 等于续期次数；`last_refreshed_at` 已更新 | 用同一个刷新令牌依次续期：每次返回新的一对令牌，`generation` 递增 |
| A5 | 刷新令牌被重复使用 | 测试从 localStorage 取出刷新令牌，先在接口上用它续期一次（模拟被盗）。页面下一次续期时，会话被作废，跳到登录页，`next_path` 是当前地址 | `revoked_at` 已填，`revoke_reason = 'reuse_detected'` | 纯接口复现：旧令牌再用一次得到 401 `identity.refresh_token_invalid`；之后这个会话的访问令牌和最新的刷新令牌也都是 401 |
| A6 | 退出 | 用户菜单点"退出"，回到登录页，localStorage 里没有刷新令牌。同一上下文的另一个标签页也回到登录页（`storage` 事件） | `revoke_reason = 'logout'`；这个会话的访问令牌在下一个请求就得到 401 | `POST /api/v0/auth/logout`，同一组断言 |
| A7 | 修改密码 | 在安全页输入当前密码和新密码，成功提示，页面保持登录。当前密码错误：字段错误，数据库不变 | `users.password` 已改变；其他会话 `revoke_reason = 'password_changed'`，当前会话未撤销；旧密码登录 401，新密码 200 | PAT 调用 `POST /api/v0/me/change-password`：因为 PAT 没有"当前会话"，全部会话都被撤销；PAT 本身继续可用 |
| A8 | 修改资料 | 在 general 页改名、姓和显示名，在偏好页改时区；刷新页面后仍是新值 | `users` 对应列和 `updated_at` | PAT 调用 `PATCH /api/v0/me` |
| A9 | 修改偏好 | 主题、语言、每周第一天；刷新页面后仍生效；主题下拉框在按钮旁展开（中英文各一次，第 9.6 节） | `profiles` 对应列 | PAT 调用 `PATCH /api/v0/me/profile` |
| A10 | 新手引导的资料步骤 | 新注册的用户在 `/onboarding` 填名字，进入下一步。创建或加入工作区的界面出现即可，M3 之前不提交 | `users.first_name`；`profiles.onboarding_step` 中 `profile_complete = true`，其余三项仍是 `false` | PAT 调用 `PATCH /api/v0/me` 和 `PATCH /api/v0/me/profile` |
| A11 | PAT 的创建、使用和撤销 | 创建（名称、说明、有效期 1 周），令牌只显示一次；列表中出现，但没有令牌原文；撤销后从列表消失 | `token_hash` 等于令牌的 SHA-256，表中任何一列都不含令牌原文；`expired_at` 约为 7 天后。用这个 PAT 调用 `GET /api/v0/me` 成功，`last_used` 被写入。撤销后 `deleted_at` 已填，再用它得到 401。把另一个 PAT 的 `expired_at` 改到过去，也得到 401 | 用另一个 PAT 创建、列出（翻页）、撤销 |
| A12 | 停用账户 | 按决策点 3 的裁定写（第 10 节） | 按裁定 | 按裁定 |
| A13 | 管理员重置密码 | — | `nerve users reset-password --email …` 从标准输入读新密码：`users.password` 改变；全部会话 `revoke_reason = 'password_reset'` | 用接口核对：新密码能登录，旧会话的刷新令牌得到 401 |
| A14 | 过期会话被清理 | — | 把一行会话的 `expires_at` 改到过去，限时轮询，直到这一行被删除；未过期的会话仍在 | — |
| A15 | 登录限流 | 限流很低的独立 nerve：同一 IP + 邮箱失败到上限后，页面显示"尝试次数过多"；换一个邮箱不受影响 | 没有新增会话 | 429 `rate_limited`，带 `Retry-After` |

**冒烟故事的更新**：
- S1：加上"迁移版本正确"：`nerve migrate status` 的每一行都是 `applied`，`goose_db_version` 的最新版本等于最后一个迁移文件（M0-P6 交接）。
- S2：登录页没有失败的接口请求，控制台没有 CSP 违规（8.3）。
- S3：`toEqual` 加上三个新字段的期望值（5.3）。

---

## 3. 设计裁定

以下裁定都在总体设计定下的范围内。3.3–3.15 是后续所有 M 都要照做的平台约定。

### 3.1 范围边界：登录之后看到什么
工作区接口在 M3 才有，M2 不为它造假数据，也不为它临时改变跳转规则。

- **没有完成新手引导的用户**：进入 `/onboarding`。
  - 资料步骤完全可用（A10）。
  - 下一步是创建或加入工作区。界面照常出现，但它调用的是 Plane 的旧接口，在 M3 之前得到 404 problem，页面按已有的错误处理显示失败。
- **已完成新手引导的用户**：没有 `next_path` 时落到 `/create-workspace`（和现在的规则一样：找不到上次的工作区就去这里）。表单能打开，提交同样得到 404，直到 M3。
- **个人设置** `/settings/profile/*` 不在工作区之下，四个标签页在 M2 全部可用。
- **需要"已完成引导"的用户的测试**，由 fixture 通过接口 `PATCH /api/v0/me/profile {is_onboarded: true}` 准备。这符合总体设计 8.2："前置数据通过接口准备，只有被测的那一步走页面"。
- **登录不再依赖工作区数据**：
  - 现在 `fetchCurrentUser` 用 `Promise.all` 同时取 `/api/users/me/settings/` 和工作区列表（`web/apps/web/core/store/user/index.ts:114-118`），其中任何一个失败，登录就失败。
  - M2 让用户 store 只取 M2 的数据（`/me`、`/me/profile`）。
  - "已登录的用户落到哪个工作区"由 `AuthenticationWrapper` 在需要时单独取数；取数失败时回落到 `/create-workspace`，不影响登录。
  - 取数的那两个调用仍是 Plane 的旧接口，由 M3 换掉（交接）。

### 3.2 字段在"数据能产生"的那个 M 进入接口
前端很多地方读 M2 的类型，但字段背后的数据属于后面的 M。规则：**一个字段的数据在本 M 能真实产生，它就进入本 M 的接口；否则由产生它的 M 加入，本 M 删掉前端的读取。** 不返回恒为 `null` 的占位字段。

| 字段 | 处理 | 理由 |
|---|---|---|
| `avatar_url`、`cover_image_url`（`User`） | M2 不定义。前端删掉当前账户的头像和封面读取（6 个文件显示头像，2 个读封面），删掉 general 页和新手引导资料步骤里的上传控件。`Avatar` 组件在没有图片时显示首字母。M5 连同 `users.avatar_asset_id`、`cover_image_asset_id`、签名地址和新的上传流程一起加入 | 头像和封面依赖文件存储（总体设计 1.1"文件"、4.3 签名地址），M2 产生不了。M5 反正要按新的 `{method, url, headers}` 上传协议重写这些上传代码（前端改动清单 3.2），现在删掉并不浪费 |
| `last_workspace_id`（`Profile`） | M2 保留。它是客户端写入的 uuid，数据库不设外键，和 Plane 一样 | Plane 中它就不是外键，任何 uuid 都接受；"是不是成员"在读取时判断（Plane `serializers/user.py:90-138`）。保留它可以避免 M3 领域的 4 个文件（`invitations/page.tsx`、`create-workspace/page.tsx`、新手引导的 `create.tsx`、`workspace-menu-root.tsx`）先删后加。是否补外键由 M3 决定 |
| 实例配置的 `workspace_creation_enabled`、`file_size_limit` | M2 定义，取自配置文件 | 值来自配置，M2 就能真实产生。M1 设计 3.7 已约定"实例配置接口的最终字段在 M2 定义"。服务端的执行分别由 M3（创建工作区）、M5（上传）负责，写进交接 |

### 3.3 模块划分
- **`identity`**：一个模块，按总体设计 6.2 负责账户、资料与偏好、会话、PAT、密码。它的用例包括注册、登录、续期、退出、认证、资料、偏好、修改密码、PAT、重置密码（命令行）、清理过期会话（定时任务），以及决策点带来的用例。
  - 不再拆出 `user` 或 `auth` 模块：资料和凭证共用 `users` 表，修改密码要同时改账户和会话，拆开以后这些操作都成了跨模块调用。
- **`instance`**：在 M0 的试点上扩展：三个配置字段（3.2、5.3）和时区列表。
- **`internal/shared`**：M2 第一次建立（M0 设计 3.1 预计在 M2），只放跨模块共用、只依赖标准库的东西：
  - `Actor`（当前账户，3.6）；
  - 领域错误 `Error` 和它的种类（3.11）；
  - `TxManager` 端口（M0-P2 交接 1）；
  - `Clock` 端口；
  - 分页游标的编解码。
  - 一个包，按文件分；不建子包。`Authorizer` 端口由 M3 加入。
- **端口放在 `app`**：沿用 M0 试点的写法（`instance/app/ports.go`），由使用方（用例）声明。

### 3.4 令牌格式
所有请求都用 `Authorization: Bearer <token>`。服务端按前缀区分令牌：`nrv_pat_` 开头的是 PAT；其余按 JWT 解析；把刷新令牌当作 Bearer 发来，得到 401。

| 令牌 | 形式 | 内容 | 服务端存什么 |
|---|---|---|---|
| 访问令牌 | JWT，头部 `{"alg":"EdDSA","typ":"JWT"}` | 只有 `sub`（用户 id）、`sid`（会话 id）、`exp`，没有任何权限信息（总体设计 4.1） | 不存 |
| 刷新令牌 | `nrv_rt_` + base64url（不补 `=`）编码的 52 字节：会话 id 16 字节 + 代数 4 字节（大端）+ 随机密文 32 字节 | 对客户端是不透明的字符串 | 只存密文部分的 SHA-256（`auth_sessions.token_hash`） |
| PAT | `nrv_pat_` + base64url 编码的 32 字节随机数 | — | 只存整个令牌的 SHA-256（`api_tokens.token_hash`） |

- **JWT 库**：用 `github.com/golang-jwt/jwt/v5` v5.3.1。
  - 它没有任何传递依赖；解析时用 `WithValidMethods([]string{"EdDSA"})`、`WithExpirationRequired()`、`WithStrictDecoding()`。
  - 自己手写约 100 行也行（还能用 RFC 9864 的 `Ed25519` 算法名），但要自己写、自己测严格的 base64url 解码、算法固定和声明校验。v0 只有 nerve 自己验签，算法名是 `EdDSA` 还是 `Ed25519` 没有外部影响。
- **随机数**：用 `crypto/rand`，不做成端口。Go 1.24 起它不会返回错误；测试只核对格式和唯一性，不需要控制具体的值。
- **有效期**：访问令牌 15 分钟，刷新令牌 30 天（总体设计 4.1），都是配置项（6.5）。

### 3.5 会话、轮换和重复使用检测
- **一次登录一行 `auth_sessions`**。行的 id 就是访问令牌里的 `sid`。
- **续期**：
  - 客户端交出刷新令牌，服务端解出会话 id、代数 g 和密文。
  - 在一条 `UPDATE … WHERE id = $1 AND generation = $g AND token_hash = $h AND revoked_at IS NULL AND expires_at > now() RETURNING …` 中把代数加 1，换上新密文的哈希，`expires_at` 改为此刻加 30 天（滑动过期）。
  - 返回新的一对令牌。
- **判定**：`UPDATE` 没有命中时，读出这一行分情况处理：

  | 情况 | 结果 |
  |---|---|
  | 行不存在、已撤销、已过期 | 401 `identity.refresh_token_invalid` |
  | g 小于当前代数 | 旧令牌被再次使用。在同一个事务里撤销这个会话（`revoke_reason = 'reuse_detected'`），记一条 WARN 日志，返回 401（码同上） |
  | g 等于当前代数但哈希不符，或 g 大于当前代数 | 伪造的令牌：401，**不撤销**。否则任何人只要知道会话 id 就能把别人踢下线 |
- **已知代价**：
  - 知道会话 id（它在访问令牌里）的人，可以拼出一个"更旧的代数"让会话被撤销。但拿到访问令牌本身已经是更严重的泄露，所以接受。
  - 续期的响应在网络上丢失时，客户端重试会被判为重复使用，用户需要重新登录。这是严格检测的固有代价；同一浏览器内的并发续期由 `navigator.locks` 避免（7.1）。
- **撤销规则**：

  | 事件 | 撤销范围 | `revoke_reason` |
  |---|---|---|
  | 退出 | 只撤销这一个会话（和 Plane 一样：只结束当前会话） | `logout` |
  | 修改密码 | 除当前会话外的全部会话；用 PAT 修改时没有当前会话，全部撤销 | `password_changed` |
  | 管理员重置密码 | 全部 | `password_reset` |
  | 停用账户 | 全部 | `deactivated` |
  | 重复使用 | 这个会话 | `reuse_detected` |
  - 总体设计 4.2 写的是"退出登录、修改密码、账户停用时，吊销该账户的刷新令牌"。上表按 Plane 的行为解释它：退出只结束当前会话，修改密码结束其他会话（Plane 靠 Django 的会话哈希做到，`plane/apps/api/plane/authentication/views/common.py:47-96`）。收尾时把这层意思写回总体设计 4.2。
  - 修改密码、重置密码都不撤销 PAT（与 Plane 一致）。停用后 PAT 不删除，但认证时要求账户未停用，所以同样失效。
- **每个请求都查一次数据库**：验签通过后，用一条按主键的查询确认"会话未撤销、账户未停用"。
  - 这样退出、修改密码、停用在下一个请求就生效，不用等访问令牌过期。
  - 总体设计 4.2 本来就要求每个请求从数据库读取成员关系；多一次主键查询的代价可以接受。
  - PAT 同理：按 `token_hash` 查询，要求未删除、未过期、账户未停用。
  - `last_used` 最多每分钟写一次（`UPDATE … WHERE last_used IS NULL OR last_used < now() - interval '1 minute'`）。Plane 每个请求都写一次（`plane/apps/api/plane/api/middleware/api_authentication.py:29-43`）；Agent 高频调用时，每次都写太重。登记为行为差异。
- **清理**：`expires_at` 已过的行由 River 定时任务删除（3.15）。已撤销的行保留到它原本的过期时间，便于排查问题。

### 3.6 认证中间件与 Actor
- **`shared.Actor`**：`{UserID, SessionID, APITokenID}`。通过会话认证时 `APITokenID` 为零值，通过 PAT 认证时 `SessionID` 为零值。这只区分凭证的种类，不区分账户的种类（总体设计 0.2 原则 1）。`shared.WithActor`、`shared.ActorFrom` 在 `context` 中存取。
- **中间件放在 `platform/httpserver`**：
  - 它解析 `Authorization`，调用一个由 `httpserver` 自己声明的小接口 `Authenticator { Authenticate(ctx, token) (shared.Actor, error) }`。
  - `identity` 模块导出这个能力，由 `bootstrap` 接上（M0-P3 交接 4）。
  - 行为：
    - 没有 `Authorization` 头：放行，不带 Actor。
    - 有，但无效：401 `unauthorized`，带 `WWW-Authenticate: Bearer error="invalid_token"`。
    - 有效：把 Actor 放进 `context`。
- **哪个操作需要登录，由 handler 决定**：
  - handler 用 `shared.RequireActor(ctx)` 取 Actor；取不到时返回 `shared` 的未认证错误，映射为 401 `unauthorized`。
  - 不用 oapi-codegen 的 `enable-auth-scopes-on-context`：它是被标为"遗留"的兼容选项（`oapi-codegen/v2@v2.8.0/pkg/codegen/configuration.go:367-388`），而且上下文的键在每个模块的生成包里，平台引用不到。
  - "声明了 `security` 的操作没有令牌时一定是 401"，由 `bootstrap` 的整程序契约测试逐个操作核对（9.3）。这样既不依赖遗留选项，漏写也会被发现。
- **依赖方向**：`platform/httpserver` 导入 `internal/shared`，用它的 `Actor` 和错误类型。
  - 架构测试的 10 条规则都允许这样做：规则 4 只禁止平台导入模块和 `bootstrap`；`shared` 只依赖标准库（规则 10）。
  - M0 设计 3.1 的表只写了"平台可以依赖标准库和第三方库"，收尾时补上"以及 `internal/shared`"。
- **限流中间件**也在 `httpserver`（3.10），通过同样由 `httpserver` 声明的 `Limiter` 接口使用 `platform/ratelimit`。平台包之间不互相导入（规则 7）。
- **模块入口的演进**（M0-P3 交接 4）：
  - 平台交给模块的 HTTP 依赖合成一个值 `httpserver.API{Errors, Middlewares}`，模块写成 `Register(mux, api)`。
  - `httpadapter.Register` 接收一个用例集合的结构体，不再逐个传指针。
  - 中间件的顺序是：请求元信息（客户端 IP、UA）→ 请求体上限 → 认证 → 限流 → handler。oapi-codegen 的 `Middlewares` 列表中最后一个在最外层，所以列表要按相反的顺序写，并有测试核对。

### 3.7 签名密钥
- **来源**：
  - `auth.jwt.private_key_file`：PKCS#8 PEM 格式的 Ed25519 私钥文件。这是总体设计 6.8 举的例子，环境变量是 `NERVE_AUTH__JWT__PRIVATE_KEY_FILE`。
  - prod 必须提供，否则启动失败，并指出这个配置项。
  - dev 和 test 可以留空：启动时生成一把临时密钥，记一条 WARN。
    - 刷新令牌存在数据库里，与签名密钥无关。重启之后，旧的访问令牌验签失败，客户端按 401 续期一次就恢复，所以临时密钥不影响开发。
    - 端到端测试的每个 worker 有自己的 nerve，令牌不跨进程使用。
  - 生成密钥：`openssl genpkey -algorithm ed25519 -out nerve-jwt.pem`，写进 README。不另做命令。
- **轮换**：换掉密钥文件，重启。
  - 旧的访问令牌验签失败，客户端续期一次，不需要新旧密钥并存。所以不引入 `kid`，也不提供 JWKS：v0 没有外部验签方。
  - 以后要多实例部署或让外部系统验签时，再加 `kid` 和公钥列表。
- **日志**：私钥的内容永远不进日志；文件路径不是机密，可以记（M0-P2 交接 4）。

### 3.8 密码
- **哈希**：argon2id（`golang.org/x/crypto/argon2`），存 PHC 字符串（`$argon2id$v=19$m=…,t=…,p=…$盐$哈希`）。
  - 默认参数：m = 19456 KiB、t = 2、p = 1，盐 16 字节，输出 32 字节（OWASP 推荐的最低配置）。
  - test 环境降到 m = 64 KiB、t = 1（总体设计 6.8）。
  - 登录时发现参数和当前配置不同，就用新参数重新哈希，在同一个事务里写回。
  - 同时进行的哈希计算最多 4 个（`auth.password.max_concurrent_hashes`），用一个带缓冲的通道限住。4 × 19 MiB ≈ 76 MiB，登录高峰不会把内存冲到失控。
- **规则：服务端执行界面上已经显示的那套规则**：
  - 长度 8–128 个字符；
  - 至少一个大写字母、一个小写字母、一个数字；
  - 至少一个特殊字符，取自 `!@#$%^&*()-_+=[]{}|;:'",.<>?/`。
  - 前四项与 `web/packages/utils/src/auth.ts:12-32` 的 `getPasswordStrength` 完全一致；128 的上限是新加的，界面同步加上。
  - 只在注册、修改密码、命令行重置时检查，登录不检查（与 Plane 相同）。
- **为什么不照搬 Plane**：
  - Plane 的服务端用 zxcvbn，要求评分 ≥ 3，没有长度规则（`plane/apps/api/plane/authentication/adapter/base.py:90-100`）。它的界面显示的却是上面这套组合规则。服务端和界面不一致，属于差异清单"修复明显的不一致"。
  - Go 的 zxcvbn 移植都已多年不维护，字典还会让二进制多出近 1 MB；评分和 Python 版也不会逐字相同，"照搬"本来就做不到。
  - 组合规则确定、容易测，与界面一字不差。
  - 登记为差异。
- **修改密码不检查"新旧相同"**：Plane 也不检查（`ChangePasswordSerializer` 中的检查是死代码）。界面上已有的"新旧必须不同"提示保留。

### 3.9 账户枚举与时间
- **登录**：
  - 邮箱不存在和密码错误，返回同一个 401 `identity.invalid_credentials`。
  - 邮箱不存在时，仍然用同样的参数对一个启动时生成的假哈希做一次校验，让两种情况耗时相同。
  - Plane 会返回 `USER_DOES_NOT_EXIST`（`plane/apps/api/plane/authentication/views/app/email.py:90-104`），登记为差异。
- **停用的账户**：只在密码正确之后才返回 403 `identity.account_deactivated`，所以只对知道密码的人暴露账户状态。
- **注册**：邮箱已存在时必然失败，没有邮件通道就没法做到"不暴露"（Plane 同样返回 `USER_ALREADY_EXIST`）。用按 IP 的注册限流（3.10）压低探测速度，并在 8.2 写明这个局限。
- **比较**：刷新令牌的哈希比较用 `crypto/subtle.ConstantTimeCompare`。PAT 按哈希查唯一索引，攻击者从查找时间里得不到另一个令牌的任何信息。

### 3.10 限流
"具体数值在 M2 确定，默认参考 Plane"（总体设计 3.6）。实现是进程内按键的令牌桶（`golang.org/x/time/rate`），闲置的键定期清掉。

| 对象 | 键 | 默认 | Plane 的对应项 |
|---|---|---|---|
| 登录 | 客户端 IP + 规范化后的邮箱 | 每分钟 10 次 | `AUTHENTICATION_RATE_LIMIT = 10/minute`，按 IP（`plane/apps/api/plane/authentication/rate_limit.py:26-39`） |
| 注册 | 客户端 IP | 每分钟 10 次 | 同上 |
| 其他未带令牌的请求（实例配置、时区、续期、退出） | 客户端 IP | 每分钟 120 次 | 匿名请求 30/minute。Nerve 的页面每次加载都会不带令牌调用实例配置和续期，同一出口 IP 后的团队很快就会碰到 30 |
| 带令牌的请求 | 会话 id 或 PAT id | 每分钟 1200 次，突发 200 | Plane 的页面请求不限流，API Key 每分钟 60 次。Nerve 的页面和 Agent 用同一套令牌，60 次连一次项目页面的加载都撑不住 |

- **位置**：
  - 登录的键里有邮箱，必须先解析请求体，所以放在 `identity` 的 HTTP 适配器里，在调用用例之前。
  - 其余三项是 `httpserver` 的限流中间件：有 Actor 时按令牌计数，没有时按 IP 计数。
- **超出**：429 `rate_limited`，带 `Retry-After`（秒，向上取整）。不加 `X-RateLimit-*`（登记为差异）。
- **客户端 IP**：
  - 默认取连接的对端地址。
  - `server.trusted_proxies`（CIDR 列表）非空，并且对端在列表中时，才从 `X-Forwarded-For` 从右往左取第一个不可信的地址。
  - 默认部署前面有 Caddy 时，要配置这一项。README 写明。
- **测试环境**：test 配置把上限都调得很高，否则同一个 IP 注册的大量测试账户会被限住。A15 用一个限流很低的独立 nerve。

### 3.11 错误码与取值校验
这是 M0-P3 交接 2 要求 M2 一次定下的体系。

- **领域错误**：`shared.Error{Kind, Code, Detail, Fields}`。
  - `Kind` 决定 HTTP 状态：`Invalid` → 422、`Unauthenticated` → 401、`Forbidden` → 403、`NotFound` → 404、`Conflict` → 409。
  - 模块用 `shared` 的构造函数声明自己的错误，码带模块前缀，例如 `identity.email_taken`。
- **一条路**：handler 返回 `error`，由 `httpserver.APIErrors` 作为生成代码的 `ResponseErrorHandlerFunc` 统一映射为 problem+json。不用"每个操作声明带类型的 `default` 响应"那条路：那样每个 handler 都要自己拼 problem。
- **取值校验放在领域层**：
  - 生成的代码只做类型绑定和 JSON 解码（M0/P3 已验证它不校验取值），`required` 也不检查。
  - 值对象的构造函数和用例入参的校验负责必填、长度、格式、枚举；一次收集全部字段的问题，返回一个 `validation_failed`。
  - handler 只做类型转换。
- **`FieldError` 加上 `code`**：
  - `{field, code, message}`。前端按 `problem.code` 和 `errors[].code` 查 `t()` 的文案，不直接显示服务端的英文 `message`（M1 收尾的规则：界面文案都走 `t()`）。
  - 字段码是一个封闭的小集合：`required`、`invalid_format`、`too_short`、`too_long`、`out_of_range`、`not_allowed`、`weak_password`、`must_be_future`、`contains_url`。
  - `api/common.yaml` 的 `FieldError` 和 `httpserver.FieldError` 同步修改。
- **平台错误码**：

  | 码 | 状态 | 来源 |
  |---|---|---|
  | `bad_request` | 400 | 已有。请求体不是合法 JSON：`detail` 是通用的一句话。类型不符：`errors[{field, code: "invalid_format"}]`，字段路径取自 `json.UnmarshalTypeError`，不再带出 Go 的类型名（M0-P3 交接 2） |
  | `unauthorized` | 401 | 新增。认证中间件，以及 `RequireActor` |
  | `not_found` | 404 | 已有 |
  | `payload_too_large` | 413 | 新增。`http.MaxBytesError`；JSON 接口的请求体上限是 `server.max_body_bytes`（默认 1 MiB） |
  | `validation_failed` | 422 | 新增。取值校验 |
  | `rate_limited` | 429 | 新增 |
  | `internal_error` | 500 | 已有 |
  | `not_ready` | 503 | 已有 |
- **`context.Canceled`**：客户端断开，不记 500，也不记 ERROR 日志，只在 DEBUG 记一行。
- **错误出口的测试**：参数绑定、请求体解码、handler 返回错误三个出口都要测，并对 problem 调 `CheckResponse`（M0-P3 交接 2）。
- **请求中未声明的字段**：生成的解码器忽略它们，而且无法配置。契约里写的是 `additionalProperties: false`，服务端却接受，这是已知的不一致。
  - M2 在 `apitest` 加 `CheckRequest`，让契约测试也校验测试发出的请求。
  - 前端的请求由生成的 TS 类型在编译时约束。
  - 服务端不引入运行时的请求校验：kin-openapi 被架构测试禁止链接进生产程序。

### 3.12 接口描述与代码生成的约定
- **oapi-codegen 的模块模板**（M0-P3 交接 1，从 M2 起每个模块照抄；spike 已在 `$M2TMP/oapi-spike` 验证能生成、能编译）：
  - `output-options.nullable-type: true`：可为空又可省略的字段生成 `nullable.Nullable[T]`，PATCH 能区分"没传"和"传 `null`"。依赖 `github.com/oapi-codegen/nullable` v1.2.0，写死。
  - `output-options.type-mapping.string.formats.uuid: {type: uuid.UUID, import: uuid}`：映射到标准库。
  - `prefer-skip-optional-pointer` 保持默认（`false`）：可省略的字段是 `*T`，`nil` 表示没传。
  - 第一个带路径参数的操作会让生成代码导入 `github.com/oapi-codegen/runtime`，写死为 v1.7.0。
  - 加依赖之后核对 `server/go.mod` 仍是 `go 1.27` / `toolchain go1.27.1`。
- **`security` 的写法**（M0-P3 交接 3）：
  - 每个操作都显式声明 `security`：需要登录的写 `[{bearer: []}]`，公开的写 `[]`。
  - `securitySchemes.bearer` 同时写在 `api/openapi.yaml` 和每个模块文件里。
  - `apitest` 的写法检查加两条：每个操作都有 `security`；引用的 scheme 都存在。
- **分页的公共组件由 M2 加入**：
  - `common.yaml` 加 `Limit`（1–100，默认 50）、`Cursor` 两个 parameter 和 `NextCursor`（`type: [string, 'null']`）。
  - M0/P3 原计划由 M3 的第一个列表接口加入；M2 的 PAT 列表先用到。
  - 游标是 base64url 编码的 `(created_at, id)`，编解码在 `shared`。游标不合法时返回 400 `bad_request`，`errors[].field = cursor`。
- **组织规则**（M0-P3 交接 5）照旧：
  - 跨模块共用的类型放 `common.yaml`；
  - 一个路径只属于一个模块文件（`/me/*` 属于 `identity`，以后的 `/me/recent-visits` 属于它自己的模块）；
  - 模块文件名等于模块目录名。
  - M2 没有 map 型对象，不需要扩展 `closedObject` 的例外。
- **TS 类型**：
  - `@nerve/api-client` 的生成命令加 `--root-types --root-types-no-schema-prefix`：`User`、`Profile` 等直接从 `@nerve/api-client` 导出（openapi-typescript 7.13.0 的 `bin/cli.js:34-36` 支持这两个选项）。
  - stores 和组件直接导入这些名字，`@nerve/types` 中对应的 Plane 类型删除（7.5）。

### 3.13 表结构约定（第一次按快照建表，以后每个 M 照做）
M0-P4 交接要求在第一次建表时一次定下。

| 事项 | 约定 | 理由 |
|---|---|---|
| 外键的 `ON DELETE` | 照搬每个外键在 Django 模型中的 `on_delete`：`CASCADE` → `ON DELETE CASCADE`，`SET_NULL` → `ON DELETE SET NULL`，`PROTECT` → `ON DELETE RESTRICT`，`DO_NOTHING` → 不写 | Plane 在 Python 里处理级联，快照中 474 个外键都没有 `ON DELETE`。把它的语义搬进数据库，符合总体设计 5.1 第 3 类改动。软删除的连带删除仍由用例在同一个事务里完成（总体设计 5.5） |
| `DEFERRABLE` | 不用 | Django 需要它来批量插入；Nerve 在同一个事务里按"先父后子"的顺序写 |
| 约束和索引的名字 | 主键、外键、唯一约束、CHECK 不写名字，用 Postgres 的默认名（`<表>_pkey`、`<表>_<列>_fkey`、`<表>_<列>_key`、`<表>_<列>_check`）；索引必须命名，写成 `<表>_<列>_idx`，部分唯一索引写成 `<表>_<列>_key` | 快照中的名字带 Django 的哈希后缀，而且有 29 张表的主键名与表名不对应（M0-P4 交接） |
| 默认值和取值范围 | 从 `plane/apps/api/plane/db/models/` 读取，写进 `DEFAULT` 和 `CHECK`；`created_at`、`updated_at` 的默认值是 `now()`；`id` 不设默认值 | 总体设计 5.3、5.5 |
| Django 的系统表 | 不照搬 | M0-P4 交接 |
| `*_like`（`varchar_pattern_ops`）索引 | 不建 | 它们只服务 Django 的 `LIKE 'x%'`，Nerve 按等值查找 |
| 外键列的索引 | 只给查询或级联真正用到的外键列建索引；`created_by_id`、`updated_by_id` 不建 | Django 给每个外键都建了 btree，多数用不上 |
| `varchar(n)` 还是 `text` | 照搬 Plane 的列类型；领域层的长度校验与 `n` 一致 | 两者在 Postgres 中性能相同；照搬就不用逐列讨论 |
| 语法 | 只用 sqlc 解析器能解析的 PG 17 语法：不用 PG 18 的 `VIRTUAL` 生成列、`RETURNING old/new`、`WITHOUT OVERLAPS`、`NOT ENFORCED`；不用 `MERGE`（sqlc v1.31.1 会不报错地生成错误的代码）；不写 `DEFAULT uuidv7()` | spike 实测（3.14）；M0-P1 交接 2 |
| 时间 | 连接池把 `timestamptz` 的扫描时区设为 UTC（pgx 的 `TimestamptzCodec.ScanLocation`，`pgx/v5@v5.11.0/pgtype/timestamptz.go:130-132`），接口输出的时间一律以 `Z` 结尾 | 总体设计 3.1 要求 UTC；spike 实测 pgx 默认按本地时区返回 |

### 3.14 sqlc 接入
- **版本**：sqlc v1.31.1，2026-04-22 发布，是目前最新的正式版。放进 `server/tools/go.mod`，与 oapi-codegen 一起用 `go tool -modfile=tools/go.mod` 调用。
  - 加入后要核对 oapi-codegen 的输出没有变化（`make gen-check-go`）。两者共用一个工具模块，依赖的版本会被一起抬高。
  - sqlc 自己依赖 pgx v5.9.2，但工具模块和主模块分开，不影响主模块的 pgx v5.11.0。
- **cgo（M0-P1 交接 1）**：一律用 `CGO_ENABLED=0` 运行 sqlc。
  - 这时它用编译成 wasm 的同一个 libpg_query（`wasilibs/go-pgquery`），不需要 C 编译器。
  - spike 在本机试了两种构建：cgo 版能编译（Apple clang 21，冷构建 17.65 秒）；`CGO_ENABLED=0` 版也能编译，`sqlc compile` 通过。
  - 这样"除了 Docker、Go、Node 以外不装任何东西"（M0 设计第 1 节）继续成立，也就不需要 M0-P1 交接提的 Docker 镜像这条退路。
  - P1 用两种构建各生成一次，确认输出相同。
- **PG 17 解析器（M0-P1 交接 2）**：v1.31.1 基于 libpg_query 17.7。PG 18 的四种语法解析失败（见 3.13）。
  - 未发布的主干已经换成纯 Go、对应 PG 18 的解析器（`sqlc-dev/oliphant`）。它正式发布后再评估升级，v0 期间不依赖未发布的版本。
- **配置**：`server/sqlc.yaml`，每个模块一个 `sql` 条目。
  - 输入：`schema: migrations/sql`（goose 的 SQL，含 River 的迁移），`queries: internal/modules/<m>/adapter/postgres/queries`。
  - 输出：`out: internal/modules/<m>/adapter/postgres/gen`，`sql_package: pgx/v5`。
  - 类型：`emit_pointers_for_null_types: true`；`omit_unused_structs: true`，否则 River 的表会生成进每个模块的 `models.go`。
  - 覆盖：`uuid` → 标准库 `uuid.UUID`（可空的是 `*uuid.UUID`），`timestamptz` → `time.Time`（可空的是 `*time.Time`）。注意 `db_type` 要写 `timestamptz`，写 `pg_catalog.timestamptz` 不会匹配。
  - 以上都经 spike 验证：映射到没有点号的标准库包 `uuid` 可以生成、编译，pgx v5.11.0 能读写。
- **写法约定**（来自 spike）：
  - 行比较的游标条件 `(created_at, id) < ($1, $2)` 会让第二个参数被推断成时间类型，运行时报错。写成 `(created_at, id) < (sqlc.arg(cursor_created_at)::timestamptz, sqlc.arg(cursor_id)::uuid)`。
  - PATCH 只更新传入的字段（总体设计 3.6"同一字段并发修改时以后写入的为准"），写法是 `col = CASE WHEN sqlc.arg(set_col)::boolean THEN sqlc.arg(col) ELSE col END`。
  - 不用"读出整行、改完整行写回"：那会覆盖别人同时改的其他字段。
- **Makefile 与持续集成**：
  - `make gen-go` 在 oapi-codegen 之后执行 `CGO_ENABLED=0 go tool -modfile=tools/go.mod sqlc generate`，先删掉旧的 `adapter/postgres/gen/*.go`。
  - `GEN_GO_OUT` 加上 `server/internal/modules/*/adapter/postgres/gen`。
- **架构测试**（M0-P2 交接 6）：规则 6 从 `adapter/http/gen` 推广到 `adapter/*/gen`：一个适配器的生成代码只能被这个适配器导入。

### 3.15 River 接入
- **版本**：`github.com/riverqueue/river` 和 `riverdriver/riverpgxv5` 都用 v0.47.0，2026-09-01 发布。
  - 它支持 Go 1.26、1.27；持续集成在 PG 18 上测试。
  - 主线迁移到第 7 版（v0.40.0 加入）。
  - 过去一年的破坏性修改（v0.39.0 的 `Validate` 签名、v0.40.0 的第 7 版迁移）都在 v0.47.0 之前，与 M2 的用法无关。
- **迁移**（M0-P2 交接 5）：
  - 用锁定版本的 CLI 导出：`river migrate-get --line main --all --exclude-version 1 --up` / `--down`，得到第 2–7 版，建出 `river_job`、`river_leader`、`river_queue`、`river_notification` 和枚举 `river_job_state`。
  - 写成一个 goose 迁移，**整段 Up、整段 Down 各包在一对 `-- +goose StatementBegin` / `StatementEnd` 里**。spike 实测：原样粘贴时，goose 按 `;` 切分，把 `$$` 函数体切断，迁移失败；包起来以后 up、down、再 up 都通过。
  - 第 5 版自己能容忍 `river_migration` 表不存在。
  - 客户端的 `Start` 只做 `SELECT 1`，不检查迁移版本（`river@v0.47.0/client.go:1076`），所以不建 `river_migration` 没有问题。spike 已用这种库启动客户端，跑通了定时任务和事务内插入。
  - 以后升级 River，用 `--version N` 导出新增的版本，另写一份迁移，不改已发布的文件。
- **位置**：
  - `platform/jobs` 负责创建和启动 River 客户端，接收连接池、worker 和定时任务列表，只依赖 River 和 pgx，不导入别的平台包。
  - 模块的 worker 是一个适配器（`identity/adapter/river`），只包一层用例。模块入口导出自己的 worker 和定时任务，由 `bootstrap` 汇总后交给 `platform/jobs`。
- **第一个定时任务**：`identity.cleanup_expired_sessions`。
  - 用 `river.NewPeriodicJob(river.PeriodicInterval(auth.session_cleanup_interval), …, &river.PeriodicJobOpts{ID: …, RunOnStart: true})`。间隔默认 1 小时，test 配置为 2 秒。
  - 它删除 `expires_at < now()` 的会话。
  - River 没有内置 cron（spike：`periodic_job.go` 只有 `PeriodicInterval` 和一个 `Next(time.Time)` 接口），M2 也不需要 cron，不引入 `robfig/cron`。
- **生命周期**（M0-P2 交接 5）：
  - `bootstrap` 的 `run` 让 HTTP 和 River 一起运行。
  - 停机顺序：HTTP 优雅停机 → River 停止（`jobs.shutdown_timeout`，默认 10 秒）→ 迁移执行器 → 连接池。
  - `pool.Close()` 放在一个带时限（5 秒）的协程里：不理会 `ctx` 的 handler 可能还占着连接。
  - handler 里的数据库调用都带请求的 `context`；续期、登录这类用例另设 5 秒的期限。
  - 端到端 fixture 的停机预算是 `readyTimeoutMs + stopTimeoutMs + 10` 秒，P2 实测停机时间，写进 review（M0-P6 交接）。
- **测试怎么用 River**：
  - `pgtest` 的模板库已经包含 River 的表（同一条迁移链）。
  - worker 的集成测试直接调用 `Work()`。
  - `bootstrap` 的测试核对定时任务已注册。
  - 端到端 A14 验证它真的按时运行。
- **给以后的 M**：事务内投递任务用 `InsertTx(ctx, tx pgx.Tx, …)`，spike 已验证回滚的事务不会留下任务。事件订阅者的写法在第一个需要它的 M（M4）定。

### 3.16 实例管理员：v0 没有
差异清单一 B 已经不保留 `instance_admins`，"服务器管理员通过命令行管理"。所以：
- 账户之间没有"实例管理员"这一级。
- 服务器管理员的能力就是配置文件和命令行。
- 工作区的管理员由 M3 的成员角色表示。
- `workspace_creation_enabled = false` 时，页面上谁都不能创建工作区。Plane 中只有实例管理员能通过管理后台绕过这个开关（`plane/apps/api/plane/license/api/views/workspace.py:71-99`）；Nerve 是否提供一个创建工作区的命令，由 M3 决定（交接）。
- 这和总体设计 0.2 原则 1"账户不区分使用者、权限由管理员分配"一致：这里的"管理员"指工作区和项目的管理员。

### 3.17 管理命令
- **`nerve users reset-password --email <邮箱>`**：
  - 邮箱按注册时的规则规范化（Plane 的命令不规范化，按原样匹配，这里修正）。
  - 在终端上不回显地提示输入两次（`golang.org/x/term`）；标准输入不是终端时读一行，供脚本和端到端测试使用。
  - 按 3.8 的规则检查密码，改写哈希，撤销全部会话（`password_reset`），只输出一行 `password reset for <email>`。
  - 账户不存在时退出码为 1，并说明原因。
- **决策点带来的命令**（第 10 节）：`nerve users activate`、`nerve users set-email`，按裁定加入。
- **命令的实现**：
  - `cmd/nerve` 只解析参数。
  - `bootstrap` 建一个只含连接池和 `identity` 用例的最小组合，不启动 HTTP 和 River。
  - 用例和 HTTP 接口共用，规则只写一份。

### 3.18 `next_path`
- **服务端不再接触 `next_path`**：
  - M1-P4 交接要求"Go 的登录、注册处理器按同一条规则校验 `next_path`"。前提是 Plane 的做法：表单提交给服务端，由服务端发出最后的跳转。
  - M2 的登录、注册是 JSON 接口，服务端不发出任何跳转，也不接收 `next_path`。跳转只发生在浏览器里，唯一的关口是前端的 `isValidNextPath`。
  - 在一个不跳转的接口上再加一个"校验后原样返回"的字段，只给网页用，对 Agent 没有意义，还违背"没有只给前端用的接口"。
  - 这条交接按"服务端没有跳转"关闭，写进 review。
- **校验规则补上控制字符**：`isValidNextPath` 另外拒绝含控制字符（`\u0000`–`\u001f`、`\u007f`）的值。单元测试覆盖 `//`、`\`、协议、控制字符。
- **带上查询参数和片段**（M1-P4 交接"由 M2 决定"第 1 项）：
  - 跳到登录页时，`next_path` 取 `pathname + search + hash`，整体 `encodeURIComponent`。
  - 校验只看开头：以单个 `/` 开头，不是 `//`、`/\`。
  - 查询和片段在浏览器里只作用于本站的页面，不影响跳转目标的主机。
  - 现在 401 拦截器拼出的地址既不编码也丢掉查询参数（`web/apps/web/core/services/api.service.ts:42`），随令牌管理器一起重写。

### 3.19 新手引导：删掉"角色""用途"两步和 `is_self_managed`
- **Plane 中这两步本来就不会出现**：
  - Plane 把 `IS_SELF_MANAGED` 写死为 `True`（`plane/apps/api/plane/settings/common.py:54`），为真时新手引导跳过"角色""用途"两步（`web/apps/web/core/components/onboarding/root.tsx:74`，进度条和返回按钮在 `header.tsx:45,57`）。
  - Nerve 只有自托管一种形态。
- **裁定**：删除这两步（`steps/role/`、`steps/usecase/`，共 338 行）、实例配置的 `is_self_managed`，以及 `onboarding/root.tsx`、`onboarding/header.tsx` 中读它的分支。资料步骤固定走自托管的那条路。
- **连带删除**：
  - `profiles.role`、`use_case` 两列也不再有写入方。general 页只是在提交时把 `role` 的默认值"Product / Project Manager"写回去，没有输入框（`web/apps/web/core/components/settings/profile/content/pages/general/form.tsx:77,146-151`）。
  - 两列一起删除，登记差异。
- **这不是产品变化**：用户看到的与 Plane 自托管版完全相同。M1-P3 交接把它列为"产品决定"，这里的依据是"照搬 Plane 的行为"。负责人如果希望 Nerve 显示这两步，可以在评审时改回。

### 3.20 需要同步到上级文档的地方（收尾时同步）
| 文档 | 位置 | 内容 |
|---|---|---|
| 总体设计 | 3.5 | 平台错误码加入 `unauthorized`、`payload_too_large`、`validation_failed`、`rate_limited`；`FieldError` 带 `code` |
| 总体设计 | 3.6 | 限流的默认数值（3.10） |
| 总体设计 | 4.2 | 退出只结束当前会话，修改密码结束其他会话（3.5） |
| 总体设计 | 8.2 | 失败时保存 trace、截图、nerve 日志和数据库快照，不录像（9.5） |
| 总体设计 | 9.4 | M2 的状态 |
| M0 设计 | 3.1 | 平台包可以依赖 `internal/shared`（3.6） |
| M0 设计 | 3.2 | 原文"M2 加入 `signup_enabled`，M5 加入文件大小上限"改为：M2 加入 `signup_enabled`、`workspace_creation_enabled`、`file_size_limit`（5.3） |
| M0 设计 | 3.3 | 固定链由三个中间件变为四个，加上安全响应头（8.3）；认证、限流等按路由挂载的中间件及其顺序（3.6） |
| M0/P3 spec | 第 7 节 | 分页的公共组件由 M2 加入（3.12） |
| 差异清单 | 二、三、四 | 第 4 节的列和第 8 节的行为差异 |
| 前端改动清单 | 3.1、3.2 | M2 一行改为已完成；CSRF 一行已完成 |

---

## 4. 数据模型

起点是 `tools/plane-schema/plane-v1.4.2-schema.sql`，按 3.13 的约定修改。每一处改动都登记到差异清单。

### 4.1 迁移文件
| 文件 | 内容 | Phase |
|---|---|---|
| `00001_identity_users.sql` | `users` | P1 |
| `00002_identity_profiles.sql` | `profiles` | P1 |
| `00003_identity_auth_sessions.sql` | `auth_sessions`（新增） | P1 |
| `00004_identity_api_tokens.sql` | `api_tokens` | P2 |
| `00005_river_main_v2_to_v7.sql` | River 主线第 2–7 版（3.15），`StatementBegin`/`End` 包住整段 | P2 |

`00005` 的 Down 段是 `migrate-get --down` 的原样输出。每个迁移都要能 up、down、再 up（`platform/postgres` 的迁移测试沿用 M0 的写法）。

### 4.2 `users`（Plane 40 列 → 10 列）
| 列 | 类型与约束 | 与 Plane 的差异 |
|---|---|---|
| `id` | `uuid PRIMARY KEY` | 照搬 |
| `email` | `varchar(255) NOT NULL UNIQUE`，`CHECK (email = lower(email))` | Plane 可以为空；规范化（去掉首尾空白、转小写）原来只在 Python 里做，现在由 CHECK 保证 |
| `password` | `varchar(128) NOT NULL` | 列名照搬；内容改为 argon2id 的 PHC 字符串（约 97 个字符） |
| `first_name`、`last_name` | `varchar(255) NOT NULL DEFAULT ''` | 照搬 |
| `display_name` | `varchar(255) NOT NULL`，`CHECK (display_name <> '')` | 照搬（Plane 的 `blank=False`）。注册时取邮箱 @ 之前的部分（`plane/apps/api/plane/db/models/user.py:169-187`） |
| `user_timezone` | `varchar(255) NOT NULL DEFAULT 'UTC'` | 照搬默认值。Plane 限定为 `pytz.common_timezones`；Nerve 接受 Go 的 `time.LoadLocation` 认得的 IANA 名称，`Local` 除外；程序内嵌 `time/tzdata` |
| `is_active` | `boolean NOT NULL DEFAULT true` | 照搬 |
| `created_at`、`updated_at` | `timestamptz NOT NULL DEFAULT now()` | 照搬 |

删除的 30 列：
- **Django 与管理后台**：`last_login`、`is_superuser`、`is_staff`、`username`（Plane 在注册时填一个 uuid 十六进制，前端从不读取）。
- **与 `created_at` 重复**：`date_joined`。
- **登录记录，改由 `auth_sessions` 承担**：`last_login_time`、`last_logout_time`、`last_login_ip`、`last_logout_ip`、`last_login_medium`、`last_login_uagent`、`last_active`、`token`、`token_updated_at`。
- **砍掉的功能或废弃的列**：
  - 第三方登录、验证码登录：`is_password_autoset`；
  - 邮箱验证：`is_email_verified`、`is_email_valid`；
  - Plane 代码中已不使用：`mobile_number`、`last_location`、`created_location`、`is_managed`、`is_password_expired`、`is_password_reset_required`、`masked_at`；
  - 机器人（总体设计 5.3）：`is_bot`、`bot_type`。
- **旧的 URL 列**（差异清单已登记）：`avatar`、`cover_image`。
- **由 M5 加入**（3.2）：`avatar_asset_id`、`cover_image_asset_id`。

名字不能含网址：Plane 对 `first_name`、`last_name` 用 `contains_url` 检查（`plane/apps/api/plane/app/serializers/user.py:16-24`，正则在 `plane/utils/url.py:12-53`），照搬。

### 4.3 `profiles`（Plane 29 列 → 11 列）
| 列 | 类型与约束 | 与 Plane 的差异 |
|---|---|---|
| `id` | `uuid PRIMARY KEY` | 照搬 |
| `user_id` | `uuid NOT NULL UNIQUE REFERENCES users ON DELETE CASCADE` | 照搬（`OneToOne`、`CASCADE`） |
| `theme` | `varchar(20) NOT NULL DEFAULT 'system'`，`CHECK (theme IN ('system','light','dark','light-contrast','dark-contrast'))` | Plane 是 `jsonb`，默认 `{}`，存自定义调色板。自定义主题已删除（M1-P2 交接），只剩一个值，改为字符串 |
| `is_tour_completed` | `boolean NOT NULL DEFAULT false` | 照搬 |
| `onboarding_step` | `jsonb NOT NULL DEFAULT '{"profile_complete":false,"workspace_create":false,"workspace_invite":false,"workspace_join":false}'`，`CHECK (jsonb_typeof(onboarding_step) = 'object')` | 照搬默认值（`get_default_onboarding()`）。四个键都必须有、都是布尔值，由领域层校验 |
| `is_onboarded` | `boolean NOT NULL DEFAULT false` | 照搬 |
| `last_workspace_id` | `uuid`（可空，不是外键） | 照搬（3.2） |
| `language` | `varchar(255) NOT NULL DEFAULT 'en'`，`CHECK (language IN ('en','zh-CN'))` | Plane 接受任意字符串；Nerve 只剩两种语言（M1 设计第 6 节） |
| `start_of_the_week` | `smallint NOT NULL DEFAULT 0`，`CHECK (start_of_the_week BETWEEN 0 AND 6)` | 照搬（Plane 的取值 0–6） |
| `created_at`、`updated_at` | `timestamptz NOT NULL DEFAULT now()` | 照搬 |

删除的 18 列：
- 3.19：`role`、`use_case`。
- 账单与公司：`billing_address_country`、`billing_address`、`has_billing_address`、`company_name`。
- 移动端：`is_mobile_onboarded`、`mobile_onboarding_step`、`mobile_timezone_auto_set`。
- 前端类型 `TUserProfile` 中没有、Nerve 不用的 Plane 新功能：`is_smooth_cursor_enabled`、`is_app_rail_docked`（应用栏已在 M1 收尾删除）、`background_color`、`goals`、`is_navigation_tour_completed`、`product_tour`、`notification_view_mode`。
- 营销邮件、更新日志：`has_marketing_email_consent`、`is_subscribed_to_changelog`。

### 4.4 `api_tokens`（Plane 17 列 → 12 列）
| 列 | 类型与约束 | 与 Plane 的差异 |
|---|---|---|
| `id` | `uuid PRIMARY KEY` | 照搬 |
| `user_id` | `uuid NOT NULL REFERENCES users ON DELETE CASCADE` | 照搬 |
| `token_hash` | `bytea NOT NULL UNIQUE`，`CHECK (octet_length(token_hash) = 32)` | 原来的 `token` 存明文（差异清单已登记） |
| `label` | `varchar(255) NOT NULL`，`CHECK (label <> '')` | 照搬。不传时由应用生成 32 位十六进制（Plane 的 `uuid4().hex`） |
| `description` | `text NOT NULL DEFAULT ''` | 照搬 |
| `expired_at` | `timestamptz`（可空，空表示永不过期） | 照搬。创建时必须晚于当前时间（Plane 不检查，属于缺陷） |
| `last_used` | `timestamptz` | 照搬；写入频率见 3.5 |
| `created_by_id`、`updated_by_id` | `uuid REFERENCES users ON DELETE SET NULL` | 照搬（`UserAuditModel` 的 `SET_NULL`） |
| `created_at`、`updated_at` | `timestamptz NOT NULL DEFAULT now()` | 照搬 |
| `deleted_at` | `timestamptz` | 照搬（撤销就是软删除） |

- **删除的列**：
  - `user_type`、`workspace_id`：总体设计 5.3；Plane 自己也已把个人令牌的 `workspace_id` 置空（`plane/apps/api/plane/db/migrations/0115_auto_20260105_1406.py:31-38`）。
  - `is_active`：Plane 的列表把它算成"未过期"，撤销用软删除，这一列没有独立的写入方。
  - `is_service`：社区版没有代码创建它。
  - `allowed_rate_limit`：没有任何限流读取它。
- **索引**：`api_tokens_user_id_created_at_idx ON (user_id, created_at DESC, id DESC) WHERE deleted_at IS NULL`，用于列表。

### 4.5 `auth_sessions`（新增，12 列）
| 列 | 类型与约束 | 说明 |
|---|---|---|
| `id` | `uuid PRIMARY KEY` | 访问令牌中的 `sid`，也写在刷新令牌里 |
| `user_id` | `uuid NOT NULL REFERENCES users ON DELETE CASCADE` | |
| `token_hash` | `bytea NOT NULL`，`CHECK (octet_length(token_hash) = 32)` | 当前这一代刷新令牌密文的 SHA-256 |
| `generation` | `integer NOT NULL DEFAULT 0`，`CHECK (generation >= 0)` | 令牌的代数：每续期一次加 1，用来认出旧令牌（3.5） |
| `user_agent` | `text NOT NULL DEFAULT ''` | 登录时的 UA，截断到 512 个字符 |
| `ip` | `inet` | 登录时的客户端 IP（3.10） |
| `expires_at` | `timestamptz NOT NULL` | 每次续期后改为当时加 `auth.refresh_token_ttl` |
| `last_refreshed_at` | `timestamptz` | |
| `revoked_at` | `timestamptz` | |
| `revoke_reason` | `varchar(20)`，`CHECK (revoke_reason IN ('logout','password_changed','password_reset','deactivated','reuse_detected'))`，另有 `CHECK ((revoked_at IS NULL) = (revoke_reason IS NULL))` | 取值随决策点的裁定增加（例如 `email_changed`） |
| `created_at`、`updated_at` | `timestamptz NOT NULL DEFAULT now()` | |

- **索引**：`auth_sessions_user_id_idx`（撤销某个账户的全部会话），`auth_sessions_expires_at_idx`（清理任务）。
- **与差异清单的写法对照**：差异清单写的是"刷新令牌的哈希、令牌轮换链（用于重复使用检测）、UA、IP、过期和撤销时间"。"轮换链"在这里用代数表示：一次登录只有一行，不为每个刷新令牌建一行（3.5）。收尾时按实际的列改写差异清单。

### 4.6 行为差异（登记到差异清单第四节）
| 行为 | Plane | Nerve |
|---|---|---|
| 登录时邮箱不存在 | 返回 `USER_DOES_NOT_EXIST` | 与密码错误相同的 401，耗时也相同（3.9） |
| 密码规则 | zxcvbn 评分 ≥ 3 | 界面显示的组合规则，长度 8–128（3.8） |
| 会话 | Django 会话，固定 7 天（`SESSION_COOKIE_AGE`） | 访问令牌 15 分钟；刷新令牌 30 天，每次使用后换新并检测重复使用 |
| 退出、修改密码、停用后旧凭证失效 | 其他会话在下一个请求时失效 | 相同，由每个请求的会话检查做到（3.5） |
| PAT 的管理 | 只能用 Cookie 会话管理 PAT | 任何凭证都能管理，包括 PAT 本身（总体设计 0.2 原则 2） |
| 无效的 PAT | 403（`AuthenticationFailed` 没有 `authenticate_header`） | 401 `unauthorized` |
| PAT 的 `last_used` | 每个请求都写 | 每分钟最多写一次 |
| PAT 的名称和过期时间 | 不校验：名称过长时变成 500，过期时间可以是过去 | 名称 1–255 个字符；过期时间必须在未来 |
| PAT 的编辑 | `PATCH` 可以改名称和说明，响应中带令牌原文 | 不提供（界面没有编辑入口）；令牌原文只在创建时返回一次 |
| 限流 | 认证接口按 IP 10/min，匿名 30/min，API Key 60/min | 3.10 的表 |
| 注册的前提 | 实例必须先由实例管理员完成设置 | 没有这一步 |
| `reset_password` 命令 | 邮箱按原样匹配 | 按注册时的规则规范化 |

---

## 5. 接口

### 5.1 操作
模块文件 `api/modules/identity.yaml`（新建）和 `api/modules/instance.yaml`（扩展）。路径都不带结尾 `/`（M1-P3 交接最后一条）。

| 方法与路径 | operationId | 认证 | 请求体 | 成功 | 主要错误 |
|---|---|---|---|---|---|
| `POST /api/v0/auth/register` | `register` | 无 | `RegisterRequest {email, password}` | 201 `AuthTokens` | 422 `validation_failed`；409 `identity.email_taken`；403 `identity.signup_disabled`；429 |
| `POST /api/v0/auth/login` | `login` | 无 | `LoginRequest {email, password}` | 200 `AuthTokens` | 422 `validation_failed`（缺字段）；401 `identity.invalid_credentials`；403 `identity.account_deactivated`；429 |
| `POST /api/v0/auth/refresh` | `refreshTokens` | 无 | `RefreshRequest {refresh_token}` | 200 `AuthTokens` | 401 `identity.refresh_token_invalid`；429 |
| `POST /api/v0/auth/logout` | `logout` | 无 | `LogoutRequest {refresh_token}` | 204 | 429。令牌未知、已过期、已撤销时也返回 204，不泄露它的状态 |
| `GET /api/v0/me` | `getMe` | bearer | — | 200 `User` | 401 |
| `PATCH /api/v0/me` | `updateMe` | bearer | `UserUpdate` | 200 `User` | 422 |
| `POST /api/v0/me/change-password` | `changePassword` | bearer | `ChangePasswordRequest {current_password, new_password}` | 204 | 422 `validation_failed`（新密码不合规）；422 `identity.current_password_incorrect`；429 |
| `GET /api/v0/me/profile` | `getProfile` | bearer | — | 200 `Profile` | 401 |
| `PATCH /api/v0/me/profile` | `updateProfile` | bearer | `ProfileUpdate` | 200 `Profile` | 422 |
| `GET /api/v0/me/api-tokens` | `listApiTokens` | bearer | 查询参数 `limit`、`cursor` | 200 `ApiTokenPage` | 400（游标不合法） |
| `POST /api/v0/me/api-tokens` | `createApiToken` | bearer | `ApiTokenCreate {label?, description?, expired_at?}` | 201 `ApiTokenCreated` | 422 |
| `DELETE /api/v0/me/api-tokens/{token_id}` | `revokeApiToken` | bearer | — | 204 | 404 `identity.api_token_not_found` |
| 决策点 1、3 带来的操作 | — | bearer | — | — | 见第 10 节 |
| `GET /api/v0/instance` | `getInstance` | 无 | — | 200 `InstanceInfo` | — |
| `GET /api/v0/timezones` | `listTimezones` | 无 | — | 200 `TimezoneList` | — |

- **修改密码用动作接口**：按总体设计 3.2"少数业务动作用 `POST /资源/动作名`"，写成 `POST /me/change-password`，返回 204。
- **登录限流的位置**：登录的 429 在调用用例之前由适配器返回（3.10）；其余 429 来自限流中间件。

### 5.2 结构
所有对象都是 `additionalProperties: false`；时间是 `date-time`（UTC）；id 是 `uuid`。

| 结构 | 字段 |
|---|---|
| `AuthTokens` | `token_type`（`enum: [Bearer]`）、`access_token`、`access_token_expires_at`、`refresh_token`、`refresh_token_expires_at`，都必填 |
| `User` | `id`、`email`、`first_name`、`last_name`、`display_name`、`user_timezone`、`created_at`，都必填。`is_active` 不返回：停用的账户根本认证不了 |
| `UserUpdate` | `first_name`、`last_name`、`display_name`、`user_timezone`，都可以省略，都不能为 `null`。改邮箱见决策点 1 |
| `Profile` | `theme`（五个值的枚举）、`language`（`en`、`zh-CN`）、`start_of_the_week`（`0`–`6` 的整数枚举）、`onboarding_step`（`OnboardingSteps`，四个布尔值都必填）、`is_onboarded`、`is_tour_completed`、`last_workspace_id`（`uuid` 或 `null`）、`updated_at`。不返回 `id` 和 `user`：资料通过 `/me/profile` 访问，这两个 id 没有用处 |
| `ProfileUpdate` | 上面除 `updated_at` 外的字段，都可以省略；`last_workspace_id` 可以传 `null` 清空。`onboarding_step` 整体替换：前端本来就发送合并后的完整对象（`web/apps/web/core/components/onboarding/root.tsx:59-62`） |
| `ApiToken` | `id`、`label`、`description`、`expired_at`（可为 `null`）、`last_used`（可为 `null`）、`created_at` |
| `ApiTokenCreated` | `ApiToken` 的全部字段加上 `token`（令牌原文，只出现在这里）。单独写出所有字段，不用 `allOf`：几个都带 `additionalProperties: false` 的 schema 用 `allOf` 组合时会互相拒绝对方的字段 |
| `ApiTokenPage` | `data: ApiToken[]`、`next_cursor`（`NextCursor`），按 `created_at` 倒序 |
| `InstanceInfo` | M0 的四个字段，加上 `signup_enabled`、`workspace_creation_enabled`、`file_size_limit`（5.3） |
| `Timezone` / `TimezoneList` | `label`、`value`（IANA 名称）、`utc_offset`、`gmt_offset`；列表是 `{data: Timezone[]}`，固定约 100 项，不分页 |

主题的五个值与 `web/packages/constants/src/themes.ts:18-69` 的 `THEME_OPTIONS` 相同，没有 `custom`，也没有调色板字段（M1-P2 交接）。

### 5.3 实例配置与时区
| 接口字段 | 配置项 | 默认值 | 谁来执行 | 前端的旧名字 |
|---|---|---|---|---|
| `signup_enabled` | `auth.signup_enabled` | `true`（决策点 2） | M2 的注册用例 | `enable_signup` |
| `workspace_creation_enabled` | `workspace.creation_enabled` | `true` | M3 的创建工作区（交接） | `is_workspace_creation_disabled`（取反） |
| `file_size_limit` | `files.size_limit`（字节） | `5242880` | M5 的上传（交接） | `file_size_limit` |

- **命名**：两个开关都用正面的 `…_enabled`，与 M0 设计 3.2 已写下的 `signup_enabled` 一致。前端读 `is_workspace_creation_disabled` 的 4 个文件改为读取反的值（`create-workspace/page.tsx:40`、新手引导的 `create.tsx:54`、`power-k/config/creation/command.ts:59`、`workspace-menu-root.tsx:45`）。
- **`is_self_managed`**：删除（3.19）。
- **默认值**：`file_size_limit` 的默认值照搬 Plane 的 `FILE_SIZE_LIMIT`（`plane/apps/api/plane/license/api/views/instance.py:142`）。
- **时区列表**：
  - 照搬 Plane 的 `TimezoneEndpoint`（`plane/apps/api/plane/app/views/timezone/base.py`）：同一份"名称 + IANA"列表，偏移量在请求时用 Go 的时区数据计算。
  - 它服务于 M2 的偏好页，以及 M3 的工作区、项目设置（`web/apps/web/core/hooks/use-timezone.tsx` 的使用方）。
  - 放在 `instance` 模块：它是实例提供的公共参考数据，不属于任何一个账户。

### 5.4 `identity` 的错误码
| 码 | 状态 | 场景 |
|---|---|---|
| `identity.invalid_credentials` | 401 | 邮箱不存在或密码错误 |
| `identity.account_deactivated` | 403 | 密码正确，但账户已停用 |
| `identity.signup_disabled` | 403 | 关闭注册时注册 |
| `identity.email_taken` | 409 | 注册（以及决策点 1 的改邮箱）时邮箱已被使用 |
| `identity.refresh_token_invalid` | 401 | 刷新令牌未知、已过期、已撤销、被重复使用、被伪造，都是这一个码 |
| `identity.current_password_incorrect` | 422 | 修改密码时当前密码错误；`errors[].field = current_password` |
| `identity.api_token_not_found` | 404 | 撤销不存在、已撤销或属于别人的 PAT（看不到的资源一律 404，总体设计 3.5） |

---

## 6. 后端结构

### 6.1 新增和修改的包
| 包 | 内容 | 依赖 |
|---|---|---|
| `internal/shared` | `Actor`、`Error`、`TxManager`、`Clock`、游标（3.3） | 只有标准库 |
| `platform/postgres` | 加上 `TxManager` 的实现：事务放进 `context`；仓储用 `postgres.DB(ctx, pool)` 取当前事务或连接池；连接池按 UTC 扫描 `timestamptz` | pgx、goose、`internal/shared` |
| `platform/clock` | `System` 时钟 | 标准库 |
| `platform/ratelimit` | 按键的令牌桶，闲置的键定期清掉 | `golang.org/x/time/rate` |
| `platform/jobs` | River 客户端的创建、启动、停止 | River、pgx |
| `platform/httpserver` | 3.6 的 `API` 值、认证中间件、限流中间件、请求元信息（客户端 IP、UA）、请求体上限；3.11 的错误映射和新平台码；导出 `RequestID`（M0-P2 交接 3）；固定链上加安全响应头（8.3） | `internal/shared` |
| `platform/webui` | CSP：启动时算出 `index.html` 内联脚本的哈希（8.3） | 标准库 |
| `platform/config` | 6.5 的新配置项、校验、`LogValue` 打码（M0-P2 交接 4）；环境变量给布尔配置项传空值时报错，不再悄悄当作 `false`（M0-P2 交接 8） | koanf |
| `modules/identity` | 6.2 | `internal/shared`；适配器另外依赖平台、pgx、x/crypto、golang-jwt、River |
| `modules/instance` | 三个配置字段、时区列表 | 同 M0 |
| `bootstrap` | 接线；`run` 让 HTTP 和 River 一起运行，按 3.15 的顺序停机；`users` 命令；在创建 logger 之后出现的致命错误写一条结构化日志（M0-P2 交接 8） | 全部 |
| `cmd/nerve` | `nerve users reset-password` 等命令的参数解析 | 同 M0 |

- **健康检查的访问日志**：`/healthz`、`/readyz` 的访问日志降为 DEBUG 级别（M0-P2 交接 8：生产环境中每次探测一条 INFO 太多）。
- **只读的迁移角色**：生产环境用单独的数据库角色执行迁移时，运行服务的角色需要 `goose_db_version` 的 SELECT 权限（M0-P2 交接 7），写进 README 的部署说明。

### 6.2 `identity` 模块的结构
```
modules/identity/
  domain/
    user.go            User；邮箱规范化；名字、显示名、时区的规则
    profile.go         Profile；Theme、Language、WeekStart、OnboardingSteps
    password.go        密码规则（3.8）
    session.go         Session；刷新令牌的编解码；轮换与重复使用的判定（3.5）
    api_token.go       APIToken；PAT 的格式；名称和过期时间的规则
    errors.go          本模块的错误码（5.4）
  app/
    ports.go           端口（6.3）
    register.go  login.go  refresh.go  logout.go  authenticate.go
    get_me.go  update_me.go  change_password.go  get_profile.go  update_profile.go
    create_api_token.go  list_api_tokens.go  revoke_api_token.go
    reset_password.go  cleanup_sessions.go  （以及决策点的用例）
  adapter/
    postgres/          仓储；queries/*.sql；gen/（sqlc 生成）
    http/              handler，按资源分文件；gen/（oapi-codegen 生成）
    argon2/            PasswordHasher
    jwt/               AccessTokens（Ed25519）
    river/             清理会话的 worker
  module.go            New(Deps) *Module；Register(mux, api)；Authenticator()；Jobs()；Commands()
```
- 一个用例一个文件（总体设计 6.1），每个文件预计 40–120 行。
- `domain` 的文件都在 400 行以内。

### 6.3 端口
| 端口 | 声明在 | 实现 | 测试 |
|---|---|---|---|
| `Users`、`Profiles`、`Sessions`、`APITokens`（仓储，按用例需要拆成小接口） | `identity/app` | `identity/adapter/postgres`（sqlc） | 集成测试，每个测试一个独立的库（`pgtest`） |
| `PasswordHasher`（`Hash`、`Verify` 返回"是否需要重新哈希"） | `identity/app` | `identity/adapter/argon2` | 往返、错误密码、参数变化后要求重新哈希、并发上限 |
| `AccessTokens`（`Issue`、`Verify`） | `identity/app` | `identity/adapter/jwt` | 往返；过期；算法不是 `EdDSA`；换了密钥；缺 `sub`、`sid`；篡改载荷 |
| `SignupPolicy`（是否开放注册） | `identity/app` | `bootstrap` 从配置传入的值 | 用例测试 |
| `TxManager` | `internal/shared` | `platform/postgres` | 提交、回滚、嵌套调用时复用同一个事务 |
| `Clock` | `internal/shared` | `platform/clock`；测试用固定时钟 | 两种实现都跑同一个小的契约测试（总体设计 6.1 的里氏替换） |
| `Authenticator` | `platform/httpserver` | `identity` 导出的认证用例 | 中间件的单元测试用假实现；整程序测试用真实现 |
| `Limiter` | `platform/httpserver` | `platform/ratelimit` | 同上 |

没有"随机数"端口：理由见 3.4。

### 6.4 一个带令牌的请求
```
请求 → 请求 ID → 异常恢复 → 访问日志 → 安全响应头          （固定链，httpserver.NewServer）
     → 请求元信息（客户端 IP、UA）→ 请求体上限
     → 认证（JWT：验签 + 会话和账户的主键查询；PAT：按哈希查询）→ Actor 进入 context
     → 限流（按会话 id 或 PAT id）
     → 生成的代码：参数绑定、JSON 解码
     → handler：RequireActor；把生成的类型转成用例的入参
     → 用例：TxManager.WithinTx { 领域规则和校验 → 写数据 }
     → 响应；或者 error → APIErrors → problem+json
```
- **事务**：注册（账户 + 资料 + 会话）、修改密码（改哈希 + 撤销会话）、续期时发现重复使用（撤销）、决策点 3 的停用（账户 + 会话 + 资料），各自在一个事务里完成。
- **单条语句的写入**不开事务：登录只插入一行会话；续期的正常路径是一条条件 `UPDATE`。

### 6.5 配置
`server/configs/config.yaml` 新增（test、prod 的覆盖值写在注释中）：
```yaml
server:
  trusted_proxies: []          # CIDR；为空时客户端 IP 取连接的对端地址（3.10）
  max_body_bytes: 1048576      # JSON 接口的请求体上限
  addr_file: ""                # 非空时，监听成功后把实际地址写进这个文件（端到端测试用 :0 监听，9.5）
auth:
  signup_enabled: true
  access_token_ttl: 15m
  refresh_token_ttl: 720h      # 30 天
  session_cleanup_interval: 1h # test: 2s
  jwt:
    private_key_file: ""       # prod 必填；dev/test 为空时启动时生成临时密钥（3.7）
  password:
    argon2_memory_kib: 19456   # test: 64
    argon2_iterations: 2       # test: 1
    argon2_parallelism: 1
    max_concurrent_hashes: 4
ratelimit:                     # test 全部调高（3.10）
  login_per_minute: 10
  register_per_minute: 10
  anonymous_per_minute: 120
  authenticated_per_minute: 1200
  authenticated_burst: 200
jobs:
  shutdown_timeout: 10s
workspace:
  creation_enabled: true
files:
  size_limit: 5242880
```
- **启动校验**：
  - 时长为正；
  - `refresh_token_ttl` 大于 `access_token_ttl`；
  - CIDR 合法；
  - argon2 的参数在 x/crypto 允许的范围内；
  - `env = prod` 时 `auth.jwt.private_key_file` 必须提供，而且文件能读出一把 Ed25519 私钥。
- **`LogValue`**：列出新的配置项；`private_key_file` 只记路径。

### 6.6 新的依赖（版本在 P1、P2 写死）
| 依赖 | 版本 | 放在 | 用途 |
|---|---|---|---|
| `github.com/riverqueue/river`、`riverdriver/riverpgxv5` | v0.47.0 | `server/go.mod` | 后台任务（P2） |
| River CLI（`github.com/riverqueue/river/cmd/river`） | v0.47.0 | 不进仓库 | 只在写迁移时导出 SQL；命令写进迁移文件的注释 |
| `github.com/sqlc-dev/sqlc` | v1.31.1 | `server/tools/go.mod` | 代码生成（P1） |
| `github.com/golang-jwt/jwt/v5` | v5.3.1 | `server/go.mod` | 访问令牌 |
| `golang.org/x/crypto` | v0.57.0 | `server/go.mod` | argon2id |
| `golang.org/x/term` | v0.46.0（x/crypto v0.57.0 所需的版本） | `server/go.mod` | 命令行不回显地输入密码 |
| `golang.org/x/time` | 当时的最新版 | `server/go.mod` | 限流 |
| `github.com/oapi-codegen/runtime`、`github.com/oapi-codegen/nullable` | v1.7.0、v1.2.0 | `server/go.mod` | 生成代码的依赖（3.12） |

- 这些依赖都不在架构测试的禁止链接名单里（`github.com/google/uuid`、kin-openapi、testcontainers、docker）。它们的传递依赖同样不能碰到这个名单，由 `TestNerveBinaryLinksNoBannedModule` 在加入依赖的那个 Phase 核对。
- River 让程序大约增加 0.96 MB（spike 实测，未去符号表时）。

---

## 7. 前端

### 7.1 令牌管理器
落实总体设计 4.3 和 7.3。

- **位置**：
  - `web/apps/web/core/lib/auth/token-manager.ts`：`TokenManager` 类。
  - `web/apps/web/core/lib/auth/api-client.ts`：web 使用的生成客户端实例，挂上认证中间件。
  - 端到端测试直接用 `@nerve/api-client`，自己带 `Authorization`，不经过令牌管理器。
- **存放**：访问令牌和它的过期时间只在内存里；刷新令牌在 localStorage 的 `nerve.refresh_token`。
- **取访问令牌**：离过期还有 30 秒以上就直接用，否则先续期。过期时间取自接口返回的 `access_token_expires_at`，前端不解析 JWT。
- **续期**：
  - 同一个标签页内，同一时刻只有一次续期，其他调用共用同一个 Promise。
  - 跨标签页用 `navigator.locks.request("nerve.auth.refresh", …)`。拿到锁之后重新读 localStorage：另一个标签页可能刚换过令牌。
  - 续期请求用一个不挂认证中间件的客户端，避免递归。
  - 续期得到 401：清掉刷新令牌，结束会话。
  - 网络错误：保留刷新令牌，把错误交给调用方。
- **401**（openapi-fetch 中间件的 `onResponse`）：
  - 请求带了访问令牌却得到 401：续期一次，用新令牌重发一次。重发需要请求的副本，在 `onRequest` 中保存（P3 先用原型验证 openapi-fetch 0.17.0 的中间件能这样做）。
  - 续期失败或重发后仍是 401：结束会话。
  - 没有刷新令牌时，401 原样交给调用方：未登录时加载页面，`GET /me` 得到 401 是正常情况，不应触发跳转。
- **结束会话**：
  - 清掉内存和 localStorage，调用应用注册的回调：`rootStore.resetOnSignOut()`。
  - 当前账户变为空，`AuthenticationWrapper` 渲染 `<Navigate to="/?next_path=…" replace />`（3.18）。
  - 跳转只在包装层这一处发生；现在的 `window.location.replace` 删除。
- **退出**：调用 `POST /auth/logout`，尽力而为：失败也清本地。然后结束会话，回到登录页。
- **其他标签页**：监听 `storage` 事件，刷新令牌被别的标签页删除时，本标签页也结束会话（A6）。
- **可测试**：`storage`、`locks`、`fetch` 都从构造函数传入，vitest 用假实现（9.4）。
- **不做**：定时主动续期（按需续期已经足够）；把访问令牌放进 sessionStorage。

### 7.2 传输层的清理
- **web 的 axios 基类**（`web/apps/web/core/services/api.service.ts`）：
  - 删掉 `withCredentials: true` 和 401 拦截（M1-P4 交接：恒为真的 `currentPath` 判断不照搬）。
  - 此后它只供 M3–M8 还没对接的领域使用。那些调用指向 Plane 的旧接口，在 Nerve 中得到 404 problem，由各领域对接时逐个删除。
  - 它不带访问令牌：对接新接口的代码都走生成的客户端。
- **CSRF**（M1 设计 3.6、M1-P3 交接，不能再延期）：
  - 删除 `AuthService.requestCSRFToken`、`ICsrfTokenData`；
  - 删除 `password.tsx` 的 CSRF 预取和隐藏字段；
  - 删除 `security.tsx` 取 CSRF 的调用、`user.service.ts` 的 `X-CSRFTOKEN`；
  - 删除 `UserStore.changePassword` 的 `csrfToken` 参数。
  - 关闭条件：`git grep -n -E "csrfmiddlewaretoken|X-CSRFTOKEN" -- web` 没有输出。
- **表单提交**：`password.tsx` 的原生表单 POST 和 `auth.service.ts` 临时拼的退出表单，都改为 store 调用生成的客户端。
- **开发代理**：Vite 只转发 `/api`（`web/apps/web/vite.config.ts:16-18`）。新接口都在 `/api/v0` 下，不用改。

### 7.3 登录页与注册页
- **路由**不变：`/` 是登录，`/sign-up` 是注册（M1/P3）。
- **提交**：`password.tsx` 的 `onSubmit` 调用 store 的 `signIn` / `signUp`。
  - 成功：令牌交给令牌管理器，取当前账户，由 `AuthenticationWrapper` 按规则跳转。
- **错误就地显示**（M1-P3 交接）：不跳转，邮箱留在输入框里。
  - `problem.code` 映射到 `t()` 的文案；`validation_failed` 的字段错误显示在对应字段下方；429 显示"尝试次数过多，请稍后再试"。
  - `auth-root.tsx` 不再读地址里的 `error_code`、`email`。
  - 保留 `next_path`，以及工作区邀请的 `invitation_id`、`slug`：它们属于 M3 的"加入某个工作区"标题。
- **`authentication.helper.tsx`**：
  - 删除 `EAuthenticationErrorCodes`（18 个 Plane 数字码）和它们的文案映射。
  - 保留 `EPageTypes`、`EAuthModes`。
  - 新写一张"problem 码 → 文案键"的表，覆盖 5.4 的 7 个模块码和 `validation_failed`、`rate_limited`、`unauthorized`、`internal_error`；有单元测试核对它覆盖了接口描述中的每个码。
- **注册链接**只在 `signup_enabled` 为真时显示（`web/apps/web/core/components/auth-screens/header.tsx:39-47`）。
- **密码强度提示**照旧（`getPasswordStrength`），加上 128 个字符的上限，与服务端一致。

### 7.4 认证包装与实例包装
- **`AuthenticationWrapper` 重写**（M1-P4 交接）：
  - 判断只用 M2 的数据：当前账户，资料中的 `is_onboarded`、`onboarding_step`，`next_path`。
  - 未登录访问受保护的页面：跳到 `/?next_path=${encodeURIComponent(pathname + search + hash)}`。
  - 已登录：`next_path` 有效就去 `next_path`；未完成引导去 `/onboarding`；否则去工作区落点。
  - 工作区落点的数据单独取，失败时回落到 `/create-workspace`（3.1）。
- **`InstanceWrapper`**：改调 `GET /api/v0/instance`（M0-P5 交接）。请求失败时仍显示维护页。

### 7.5 类型、services 与 stores
| 位置 | 现状 | M2 |
|---|---|---|
| `@nerve/types` 的 `IUser`、`TUserProfile`、`IUserTheme`、`TOnboardingSteps`、`IInstanceInfo`、`IInstanceConfig`、`IApiToken`、`ICsrfTokenData`、`TTimezones`、`TTimezoneObject` | 手写的 Plane 类型（`web/packages/types/src/users.ts` 等） | 删除。使用方改从 `@nerve/api-client` 导入 `User`、`Profile`、`OnboardingSteps`、`InstanceInfo`、`ApiToken`、`Timezone`（3.12） |
| `IUserLite`、`IUserSettings` | 成员信息（M3）；`/users/me/settings/` 中的工作区数据 | 不动，留给 M3 |
| `EStartOfTheWeek` | 日历等 9 个文件使用的数字枚举 | 保留：纯界面常量，取值与生成的 0–6 一致 |
| `core/services/auth.service.ts` | CSRF 和表单 | 重写：`register`、`login`。`refresh`、`logout` 由令牌管理器调用 |
| `core/services/user.service.ts` | Plane 的 `/api/users/me/…` | M2 的方法改为生成客户端的薄封装。属于其他领域的 4 个方法（`getUserProfileIssues`、`leaveWorkspace`、`joinProject`、`leaveProject`）原样留下，由 M3、M4 对接时迁走 |
| `core/services/instance.service.ts`、`timezone.service.ts` | Plane 的地址 | 生成的客户端 |
| `@nerve/services` 的 `api.service.ts`、`developer/` | `APITokenService`（Plane 的 `/api/users/api-tokens/`） | 删除。包里只剩地址工具和文件工具，整个包由 M5 删除（总体设计 7.1） |
| `store/user/index.ts`（`UserStore`） | `IUser`；登录时并行取三样 | `data: User`；`fetchCurrentUser` 只取 `/me`；`signIn`、`signUp`、`signOut` 通过令牌管理器；删除死成员（`reset`、`isAuthenticated`、`error` 等） |
| `store/user/profile.store.ts` | `TUserProfile`；`updateUserProfile` 吞掉错误（`:138-148`），调用方的错误提示从不出现 | `data: Profile`；失败时抛出。`finishUserOnboarding` 合并为一次 `PATCH /me/profile`（`onboarding_step` + `is_onboarded` + `last_workspace_id`）；`updateTourCompleted`、`updateUserTheme` 同样用这一个接口 |
| `store/user/settings.store.ts` | `/users/me/settings/` 加上侧边栏状态 | 从登录流程中拿掉（3.1）；删除死成员（`isScrolled`、`toggleIsScrolled` 等）；工作区数据留给 M3 |
| `store/instance.store.ts` | `IInstanceConfig` | `config: InstanceInfo` |
| `store/workspace/api-token.store.ts` | 没有任何代码读它（`store/workspace/index.ts:93` 只是创建）；页面用 SWR 直接调用 service | 移到 `store/user/api-token.store.ts`，用生成的客户端重写，翻页直到取完；页面和弹窗改为经 store 读写（总体设计 7.2：组件不直接调用接口） |

**跟随类型修改的使用方**：
- 当前账户的头像、封面：6 个文件（3.2）。
- `is_workspace_creation_disabled`：4 个文件（5.3）。
- `profile.theme.theme` → `profile.theme`：`store-wrapper.tsx:52-72`、`theme-switcher.tsx`、`power-k/config/preferences-commands.ts`。
- PAT 列表项的"有效 / 已过期"：改为由 `expired_at` 计算（`web/apps/web/core/components/api-token/token-list-item.tsx`）。

### 7.6 删除的 Plane 代码
- CSRF：4 处，加上它的类型（7.2）。
- Plane 的 18 个认证错误码和它们的文案（7.3）。
- 新手引导的"角色""用途"两步（338 行）和 `is_self_managed`（3.19）。
- `@nerve/services` 的 `api.service.ts`、`developer/`（共 178 行）。
- general 页和新手引导资料步骤里的头像、封面上传控件（3.2）；general 页写 `role` 默认值的逻辑（3.19）。
- 死成员和死 prop（7.8）。
- 以上各项的文案键：中英文一起删，`sync-check` 和"整键 + 模板前缀"的核对沿用 M1 的做法。

### 7.7 个人设置的四个标签页
- **general**：
  - 名、姓、显示名可以修改，邮箱只读；
  - 头像和封面按 3.2 删除；
  - "停用账户"按决策点 3 处理。
- **preferences**：
  - 主题：`PATCH /me/profile {theme}`，保留切换后刷新页面的现有行为；
  - 语言：`PATCH /me/profile {language}`；
  - 时区：`PATCH /me {user_timezone}`，下拉框的数据来自 `GET /api/v0/timezones`；
  - 每周第一天：`PATCH /me/profile {start_of_the_week}`。
- **security**：
  - 修改密码。错误用生成的 `Problem` 类型处理，去掉 `Error & { error_code?: string }` 的断言和 `toString()` 转换（M1-P4 交接"由 M2 决定"第 2 项）。
  - `current_password_incorrect` 显示在当前密码字段下；`validation_failed` 的密码错误显示在新密码字段下。
- **api-tokens**：
  - 创建（名称、说明、有效期：1 周、1 个月、3 个月、1 年、自定义日期、永不过期）；
  - 创建后令牌只显示一次，同时下载 CSV（照搬现有行为）；
  - 列表、撤销。
- **主题下拉框**（M1-closeout 交接）：界面语言为 zh-CN 时，主题下拉框画在页面左上角。
  - P4 查明 `web/packages/ui/src/dropdowns/custom-select.tsx` 中 react-popper 定位加 `createPortal` 的根因，在组件里修复，不在调用处打补丁。
  - 浏览器核对中英文各一次。

### 7.8 约束 M2 的 M1 收尾规则
- **死成员和死 prop**：`domains.mjs --rows M2` 的 66 行全部处理。
  - 改到的文件随改随删。
  - 没改到的（例如 `store/issue/profile/*` 的 7 行、`auth-layout/*-wrapper.tsx` 的 2 个 `isLoading` prop）数量很少，在 P4 集中删掉，一次关闭交接。
  - 确认是误报的写进 review（文件、名称、谁在读它或传它）。
- **oxlint**：
  - 改到的文件在 M2 结束时没有警告。M2 领域现有 32 条，分布在 19 个文件。
  - 另外按规则清掉一类：`eslint(no-unneeded-ternary)`，全仓 56 条，可以机械修复。
  - 各包的上限随之调低（`tools/lint-cap.mjs` 要求警告数等于上限）。
- **界面文案都走 `t()`**，中英文的键一致（`sync-check` 是门禁）。新写的错误提示、令牌管理器的提示都在其中。

### 7.9 关键词守卫的新规则
每个 Phase 在删除的同一个提交里加入规则（M1 设计 7.4 的做法）：

| 规则 | Phase |
|---|---|
| `csrf`（不区分大小写）、`X-CSRFTOKEN`、`csrfmiddlewaretoken`、`get-csrf-token` | P3 |
| `withCredentials:\s*true` | P3 |
| Plane 认证地址 `/auth/sign-`、`/auth/change-password` | P3 |
| Plane 的认证错误码参数 `error_code` | P3 |
| `is_self_managed` | P3 |
| Plane 的用户、实例、时区地址：`/api/users/me/profile`、`/api/users/me/onboard`、`/api/users/me/tour-completed`、`/api/users/api-tokens`、`/api/instances/`、`/api/timezones/` | P3、P4 |

- 不禁止整个 `/api/users/me/`：`joinProject` 等 M3 的旧调用还在用它。
- `file-upload.service.ts` 的 `withCredentials: false` 属于 M5，不会命中上面的正则。

### 7.10 规模估计
**口径**：M2 领域的前端代码约 8,467 行（2026-09-25 实测，按文件组统计）：

| 文件组 | 行数 |
|---|---|
| services | 300 |
| `packages/services` | 400 |
| stores | 1,276 |
| store 的 hooks | 203 |
| wrappers | 246 |
| 认证页和表单 | 1,122 |
| 个人设置 | 2,405 |
| 新手引导 | 2,315 |
| 类型 | 200 |

**估计**（M2 结束时按实际数字校准，写进收尾 review，供 M3 以后的估算参考，总体设计第 10 节）：

| 项 | 估计 | 依据 |
|---|---|---|
| 改动的文件 | 80–95 个 | 上面这些文件组中约 60 个；跟随类型修改的使用方约 15 个（7.5）；新文件约 8 个（令牌管理器、客户端实例、错误文案表和它们的测试） |
| 删除 | 约 1,100 行 | 7.6 的各项 |
| 重写 | 约 3,000 行 | stores 约 700 行；services 约 250 行；两个包装层约 160 行；认证表单约 450 行；个人设置和 PAT 组件约 900 行；新手引导约 450 行 |
| 新增 | 约 700 行 | 令牌管理器约 250 行，它的测试约 300 行，其余约 150 行 |
| 净变化 | 减少约 400 行 | |

**后端**：
- 生产代码约 4,000 行（不含生成的代码），测试约 5,000 行；
- 手写的 SQL 约 300 行，加上 River 的迁移（318 + 189 行）；
- `identity.yaml` 约 800 行。

**端到端**：fixture 约 500 行，15 个故事约 1,800 行。

---

## 8. 安全

### 8.1 密码与哈希
见 3.8。

### 8.2 账户枚举
- 登录不暴露账户是否存在，时间也一致（3.9）。
- 注册必然暴露：没有邮件通道，就无法做到"已注册和未注册的回答相同"。用每个 IP 每分钟 10 次的注册限流压低探测速度。
- 这是 v0 已知的局限，写进 README 的安全说明。

### 8.3 安全响应头与 CSP
- **固定链上的安全响应头**（所有响应）：`X-Content-Type-Options: nosniff`、`Referrer-Policy: same-origin`、`X-Frame-Options: DENY`。
  - 这就是 M0-P5 交接说的"和 CSP 放在同一层"：两者都在 M2 加入。
  - 固定链由三个中间件变为四个，M0 设计 3.3 的说明随之更新。
- **`webui` 给 HTML 响应加上 CSP**：
  ```
  default-src 'self'; script-src 'self' 'sha256-…'（每个内联脚本一项）;
  style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; font-src 'self' data:;
  connect-src 'self'; object-src 'none'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'
  ```
- **为什么脚本用哈希**：
  - 构建出的 `index.html` 有 5 个内联脚本：next-themes 的主题初始化、React Router 的上下文、模块导入，以及两段数据流的写入和关闭（`make build-web` 实测）。
  - 允许 `'unsafe-inline'` 会让 CSP 对 XSS 失去作用。
  - `webui` 在启动时从内嵌的 `index.html` 中取出内联脚本、计算 SHA-256；前端重新构建后不需要手工更新。
  - 没有构建前端时没有 `index.html`，也就不设 CSP。
- **样式允许 `'unsafe-inline'`**：React 的 `style` 属性和 popper 的定位都靠它；样式注入的风险远低于脚本。
- **图片、字体**：都来自本站（构建产物中的 CSS 只引用本站的字体文件）。M5 接入 S3 时，把存储的来源加进 `img-src` 和 `connect-src`（交接）。
- **HSTS** 由前面的反向代理设置：nerve 不知道自己是否在 TLS 之后。
- **核对**：S2 统计 `securitypolicyviolation` 事件为 0；P3 的浏览器核对再覆盖新手引导页和设置页。

### 8.4 日志
- **永远不记**：
  - 密码；
  - 任何令牌（访问、刷新、PAT）和它们的哈希；
  - `Authorization` 请求头；
  - 认证接口和创建 PAT 的请求体、响应体；
  - JWT 私钥的内容；
  - 不存在的账户的邮箱。
- **INFO**：
  - 注册：`user_id`；
  - 登录成功：`user_id`、`session_id`、IP；
  - 登录失败：原因（`invalid_credentials` 或 `deactivated`）、IP，已知账户时加 `user_id`；
  - 退出：`session_id`；
  - 修改密码、重置密码：`user_id`；
  - 创建、撤销 PAT：`user_id`、`token_id`；
  - 触发限流：桶的种类、IP。
- **WARN**：发现刷新令牌被重复使用：`user_id`、`session_id`、IP。
- **访问日志**不变：方法、路径、状态码、耗时、请求 ID。令牌从不出现在地址里，路径中没有机密。
- **测试**：登录、注册、续期、创建 PAT 的测试捕获日志输出，断言其中不含密码和令牌。

### 8.5 刷新令牌放在 localStorage 的风险与升级路径
- **风险**：任何 XSS 都能读出 localStorage 中的刷新令牌，得到一个长期可用的会话，直到发现重复使用或用户退出。
- **现有的防线**：
  - 严格的 CSP（8.3）；
  - 服务端清洗 HTML（M4，bluemonday）；
  - 刷新令牌每次使用后换新，并检测重复使用（3.5）；
  - 访问令牌只在内存里，15 分钟过期；
  - 退出、修改密码在下一个请求就生效（3.5）。
- **升级路径**（总体设计 4.3，M2 只写下来，不做）：
  - 注册、登录、续期的响应把刷新令牌写进 `HttpOnly; Secure; SameSite=Strict; Path=/api/v0/auth` 的 Cookie；
  - 续期和退出在请求体里没有令牌时，从这个 Cookie 读；
  - 其余接口仍然只认 `Bearer`，不受影响。
  - 前端的令牌管理器不再碰 localStorage，续期请求带上 `credentials: "same-origin"`；多标签页共享同一个 Cookie，`navigator.locks` 仍然需要。
  - 跨站请求伪造：`SameSite=Strict`，加上令牌只在响应体中返回、跨站读不到，最坏的情况是对方让令牌换了一次新。
  - 改动只在 `identity` 的 HTTP 适配器和令牌管理器两处。

### 8.6 其他
- JSON 接口的请求体上限 1 MiB（3.11）。
- 客户端 IP 只在对端是可信代理时才取自 `X-Forwarded-For`（3.10）。
- PAT 拥有账户的全部能力，v0 不做权限范围（与 Plane 相同）。
- `nrv_pat_`、`nrv_rt_` 前缀让代码托管平台的密钥扫描能认出泄露的令牌。

---

## 9. 测试

### 9.1 后端单元测试
| 对象 | 内容 |
|---|---|
| `identity/domain` | 邮箱规范化；名字中的网址；显示名；时区；主题、语言、每周第一天；新手引导的四个键；密码规则（表格驱动：8、128 的边界，缺各类字符，非 ASCII 字符）；刷新令牌的编码和解码（长度不对、前缀不对、base64 非法）；轮换的判定（当前代、更旧的代、伪造）；PAT 的格式、名称、过期时间 |
| `identity/app` | 每个用例用内存里的假端口测试：注册（关闭注册、邮箱已存在、三行在一个事务里）；登录（邮箱不存在时仍做一次假校验、停用只在密码正确后报出、参数变化后重新哈希）；续期（轮换、重复使用时撤销、伪造时不撤销）；退出（幂等）；修改密码（撤销其他会话；用 PAT 修改时撤销全部）；PAT 的创建、列出、撤销；认证（JWT 加会话检查；PAT 的过期和 `last_used` 的写入频率）；清理过期会话 |
| 适配器 | 6.3 表中各端口的测试 |
| `platform` | 限流（桶、清理闲置的键、`Retry-After`）；中间件（没有头、头无效、头有效；限流的键；可信代理下的客户端 IP；413；错误映射，包括不带 Go 类型名的解码错误和 `context.Canceled`）；`webui` 的 CSP 哈希；配置的新校验和打码 |

### 9.2 集成测试（连接真实的 Postgres，`pgtest`）
- 仓储的每条查询，包括：
  - 续期的条件 `UPDATE`；
  - 游标分页（`created_at` 相同的两行不重不漏）；
  - 违反唯一约束和 CHECK 时映射为领域错误。
- `TxManager`。
- 5 个迁移都能 up、down、再 up。
- River worker 的 `Work()` 只删除过期的会话。

### 9.3 契约测试
- `identity`、`instance` 的 handler 测试对每种状态（包括每种 problem）调用 `CheckResponse`。
- `bootstrap` 的整程序测试：
  - 遍历 `api/dist/openapi.yaml`，每个声明了 `security: [{bearer: []}]` 的操作，不带令牌时得到 401 problem；声明 `security: []` 的操作不会得到 401（3.6）。
  - 带 PAT 调用 `GET /me` 成功。
- `apitest`：
  - 写法检查加上 `security` 的两条规则（3.12）；
  - 新增 `CheckRequest`，测试中发出的请求也按契约校验（3.11）。
- 错误出口的测试（3.11）。

### 9.4 前端单元测试（vitest）
- 令牌管理器：
  - 同一标签页内只续期一次；
  - 用假的 `locks` 验证跨标签页串行；
  - 401 → 续期 → 重发一次；续期得到 401 时结束会话；网络错误时保留刷新令牌；
  - `storage` 事件。
- `isValidNextPath` 的控制字符；`next_path` 的拼接（路径 + 查询 + 片段，整体编码）。
- problem 码到文案键的表覆盖接口中的每个码。
- 密码规则的 128 上限。

### 9.5 端到端
- **fixture 的扩展**（M0-P6 交接）：
  - **`auth.ts`**：
    - 通过接口注册、登录，每个测试用自己的邮箱（由测试 id 生成），返回令牌；
    - `signedInPage`：用 `context.addInitScript` 在页面加载前把刷新令牌写进 localStorage，页面自己续期得到访问令牌；
    - `pat`：通过接口创建 PAT。
  - **`db.ts`**：每个 worker 一个 `pg` 连接池。断言函数按表放在 `e2e/fixtures/assert/identity.ts`，页面版本和接口版本调用同一个函数。
  - **`server.ts`**：
    - `startNerve` 接受额外的环境变量，用来启动"关闭注册""访问令牌很短""限流很低"几种独立的 nerve，只在需要它的测试中存在；
    - nerve 在 `127.0.0.1:0` 上监听，把实际地址写进 `server.addr_file`，fixture 直接读取，不再先猜端口（M0-P6 交接：端口竞争的根本解决）；
    - `runNerve` 支持标准输入（A13）。
  - **失败时的产物**：
    - 在全局容器里对本 worker 的库执行 `pg_dump`，快照和 trace、截图、nerve 日志放在一起；
    - 不录像：trace 已经有每一步的截屏，录像还要多下载 ffmpeg。收尾时把总体设计 8.2 的措辞改为"trace、截图、日志和数据库快照"（M0-P6 交接）。
  - **新加的等待**都受 fixture 自己的期限约束（M0-P6 交接）。
- **故事**：A1–A15，以及 S1–S3 的更新（第 2 节）。
- **耗时**：M0 的 `e2e` 任务约 120 秒，超时 20 分钟。M2 加入约 15 个故事文件，P4 结束时实测，写进 review。

### 9.6 控制者的浏览器核对
沿用 M1 设计 7.5 的做法：临时脚本不进仓库，脚本全文、假数据、运行命令和断言写进该 Phase review 的附录。
- **P3**：
  - 登录页、注册页的就地错误；
  - `next_path` 的各种取值；
  - 两个标签页的续期和退出；
  - 实例请求失败时的维护页；
  - 新手引导的资料步骤和下一步的出现；
  - 登录页、新手引导页、设置页没有 CSP 违规；
  - 控制台没有"navigate() 应在 useEffect 中调用"的警告。
- **P4**：
  - 个人设置的四个标签页；
  - 主题下拉框在按钮旁展开（zh-CN、en 各一次）；
  - PAT 创建后只显示一次、CSV 下载；
  - 安全页的字段错误；
  - 停用账户（按裁定）。

---

## 10. 负责人决策点

下面三项会改变用户能做什么，由项目负责人决定。每项给出选项、代价和建议。裁定之前，P1 不受影响；它们的接口和命令放在 P2 的最后几个任务里。

### 决策点 1：怎样修改登录邮箱（M1 设计 3.13）
**背景**：
- M1 删掉了"向新邮箱发验证码"的旧流程。v0 没有邮件服务，没法证明新邮箱属于这个人。
- 邮箱是登录名，M3 的工作区邀请也按邮箱发出。

| 选项 | 做法 | 好处 | 代价 |
|---|---|---|---|
| A. 自助修改 | 个人设置里输入当前密码和新邮箱（`POST /api/v0/me/change-email`），撤销其他会话 | 用户自己就能改 | 多一个接口、一个表单和它的文案。新邮箱不经验证就生效：填错了也会生效；占用了别人尚未注册的邮箱时，对方之后无法注册，还会收到本该给对方的工作区邀请。注册时本来就有同样的问题（v0 没有邮箱验证） |
| B. 只能由服务器管理员修改 | `nerve users set-email --email <旧> --new-email <新>`，撤销该账户的全部会话 | 代码最少：一个命令，复用同一个用例。和"忘记密码由管理员重置"是同一种信任方式 | 用户要找管理员 |
| C. v0 不提供 | — | 没有代码 | 邮箱写错或公司换了域名，只能新建账户，数据留在旧账户上 |

**建议：B**。
- 改邮箱很少发生；v0 没有邮箱验证，让管理员经手更稳妥；成本最低。
- 以后接入邮件服务时，再做带验证的自助修改。

### 决策点 2：注册默认是否开放，第一个账户从哪里来
**背景**：
- Plane 默认开放注册（`ENABLE_SIGNUP` 默认 `"1"`）。但 Plane 要先由实例管理员在管理后台完成设置，之后才有人能注册。
- Nerve 没有这一步，也没有实例管理员（3.16）。开放注册的服务器一旦放在公网上，任何人都能注册，并创建自己的工作区（除非关闭了创建工作区）；他们看不到别人的工作区（M3 的权限）。

| 选项 | 做法 | 好处 | 代价 |
|---|---|---|---|
| A. 默认开放（照搬 Plane 的默认值） | 第一个人直接注册。团队都注册之后，运维把 `auth.signup_enabled` 改为 `false`，以后靠 M3 的邀请加人 | 部署完就能用；与 Plane 一致 | 部署在公网又忘了关时，任何人都能注册 |
| B. 默认关闭 | 第一个账户用新命令 `nerve users create --email …` 创建（密码从标准输入读）；其他人靠 M3 的邀请注册 | 默认安全 | 多一个命令；M3 的邀请做好之前，每个账户都要用命令行创建，或者临时打开注册 |
| C. 按环境不同：prod 关闭，dev/test 开放 | 同 B，只在 prod 生效 | 生产默认安全，开发不受影响 | 同 B；另外 dev 和 prod 的行为不一样 |

**建议：A**。
- v0 面向想先试用的小团队；关闭注册只要改一行配置，README 的部署清单写明这一步。
- 陌生人注册后看不到任何已有的工作区。

### 决策点 3：停用账户
**背景**：
- Plane 在个人设置里提供自助停用（`DELETE /api/users/me/`，`plane/apps/api/plane/app/views/user/base.py:252-348`）：
  - 停用全部成员关系、重置新手引导、删除全部会话、把密码改成随机值；
  - 不要求输入密码；
  - 只有命令 `activate_user` 能恢复。
- Nerve 的前端保留了这个按钮和弹窗（`web/apps/web/core/components/settings/profile/content/pages/general/form.tsx:398-408`、`web/apps/web/core/components/account/deactivate-account-modal.tsx`）。
- 总体设计 4.2 提到"账户停用时让所有会话失效"。
- 其中"停用成员关系"要用到 M3 的数据。

| 选项 | 做法 | 好处 | 代价 |
|---|---|---|---|
| A. 照搬 Plane 的自助停用 | `POST /api/v0/me/deactivate`：账户不能再登录，全部会话撤销，PAT 失效，新手引导的进度重置；另加 `nerve users activate --email` 恢复。停用成员关系由 M3 通过领域事件补上。不把密码改成随机值：Nerve 用 `is_active` 挡住登录，恢复后原密码仍可用 | 保留现有的界面，和 Plane 一致 | 一个接口、一个命令、一条给 M3 的跨模块交接。和 Plane 一样不要求输入密码：拿到访问令牌的人可以停用这个账户（管理员能恢复） |
| B. 只能由管理员停用和恢复 | `nerve users deactivate / activate --email`；删除界面上的按钮和弹窗 | 不会误操作，也不会被恶意停用；员工离职由管理员处理，更贴近团队的用法 | 用户不能自己注销 |
| C. v0 不提供 | 删除按钮和弹窗，不做接口和命令 | 没有代码 | 总体设计 4.2 的"账户停用"没有入口；员工离职后只能由 M3 把他移出工作区，账户本身仍能登录 |

**建议：A**。它是保留的功能，照搬 Plane。负责人如果更看重团队的离职管理，B 的代码量相近。

---

## 11. 需要负责人裁定的架构问题

**没有。** 以下几处看起来像是与上级设计冲突，核对后都能在现有规则内解决，已作为裁定写在第 3 节，请控制者评审时确认：
1. **平台包依赖 `internal/shared`**（3.6）：
   - 架构测试的 10 条规则都允许，`shared` 本身只依赖标准库；
   - 这样错误映射和 Actor 只需实现一次；
   - M0 设计 3.1 的表补一句即可。
2. **M1-P4 交接要求服务端校验 `next_path`**（3.18）：它的前提（服务端发出跳转）在 JSON 接口下不存在了。
3. **后续 M 的字段**（3.2）：按"数据能否在本 M 产生"区分；实例配置的字段按 M1 设计 3.7 在 M2 定义。
4. **总体设计 4.2 的"退出时吊销该账户的刷新令牌"**（3.5）：按 Plane 的行为解释为只结束当前会话，收尾时改写原文。

---

## 12. Phase 划分与实施规划

所有 Phase 依次推进。每个 Phase 按 M0、M1 的节奏：
1. worktree；
2. spec 和 plan（先做原型验证）；
3. 按任务实现，逐个评审；
4. 整分支评审（opus）；
5. 修复和限定范围的复审；
6. review、交接；
7. `--no-ff` 合并，推送；
8. 持续集成通过；
9. 清理 worktree 和分支。

每个 Phase 合并时，持续集成的全部门禁、S1–S4 和此前已加入的 M2 故事都必须通过。

**为什么这样分**：
- **先后端，后前端**：前端要对接真实的接口，M2 不写假后端。
- **后端内先做认证核心**：其余每个接口都需要先认出当前账户。
- **前端内先做令牌管理器和登录页**：设置页要靠它们才能访问。
- **每个 Phase 合并的都是能用、有测试的代码**：后端 Phase 用接口版本的故事验收，页面版本随前端 Phase 加入。
- **比较过的其他拆法**：
  - 只有一个后端 Phase：太大，评审和定位失败的成本高（M1 的教训）。
  - 按功能前后端一起切（"认证""账户"各一个 Phase）：一个 Phase 同时评审 Go 和 TS，令牌管理器还要等后端做到一半。
  - 单独的"基础设施"Phase：会合并一批暂时没人用的代码（M0 设计 7.2 不这样做的理由）。

### P1 `auth-core`：认证核心（后端）
- **目标**：任何调用方都能通过接口注册、登录、续期、退出，带令牌访问 `GET /me`；平台约定一次定下。
- **交付物**：
  1. 表结构约定（3.13）；迁移 `00001`–`00003`；sqlc 接入（3.14），架构测试规则 6 推广到 `adapter/*/gen`。
  2. 平台：
     - `internal/shared`、`platform/postgres` 的事务和 UTC、`platform/clock`、`platform/ratelimit`；
     - `httpserver` 的 `API` 值、认证和限流中间件、请求元信息、请求体上限、错误体系和新平台码、导出 `RequestID`、固定链上的安全响应头（3.6、3.10、3.11、8.3）；
     - 配置（6.5 中 P1 用到的部分），以及 M0-P2 交接 8 的三个小问题。
  3. 接口描述：oapi-codegen 的模块模板、`security` 的写法和 `apitest` 的两条规则、`FieldError.code`、`CheckRequest`（3.11、3.12）。
  4. `identity`：
     - `register`、`login`、`refreshTokens`、`logout`、`getMe`；
     - argon2id；Ed25519 JWT 和密钥（3.7）；
     - 认证用例（JWT 部分）；
     - 登录和注册的限流。
  5. 端到端：
     - fixture 的扩展（9.5，页面的登录状态除外）；
     - S1 的迁移断言；
     - A1、A2、A3、A5、A6、A15 的接口版本。
- **关闭**：M0-P1；M0-P2 的第 1–4、6–8 条；M0-P3；M0-P4；M0-P6 中与 fixture、数据库断言、S1、端口、等待期限、录像有关的几项。
- **完成线**：
  - 上述 6 个故事的接口版本和 S1–S4 在持续集成中通过；
  - `make gen-check` 覆盖 sqlc 的输出；
  - 架构测试和传递依赖测试通过。

### P2 `account-api`：账户接口（后端）
- **目标**：账户的其余接口都可用，而且都能用 PAT 完成；River 的第一个定时任务运行。
- **交付物**：
  1. `updateMe`、`getProfile`、`updateProfile`、`changePassword`（3.5 的撤销规则）。
  2. PAT：
     - 迁移 `00004`；
     - 创建、分页列表（`common.yaml` 的分页组件）、撤销；
     - PAT 认证和 `last_used`。
  3. `instance` 的三个字段和 `listTimezones`；嵌入 `time/tzdata`。
  4. `nerve users reset-password`；决策点 1、3 的接口和命令。
  5. River（3.15）：
     - 迁移 `00005`；
     - `platform/jobs`；
     - `bootstrap` 的运行和停机顺序；
     - 清理过期会话的定时任务。
  6. 端到端：
     - A4、A7–A14 的接口版本（需要登录的都用 PAT）；
     - S3；
     - 实测 nerve 的停机时间。
- **关闭**：
  - M0-P2 第 5 条；
  - M0-P6 的 S3 和 River 停机两项；
  - M1-P2、M1-P3 中接口描述的部分：主题字段；实例字段；不再读的字段和不再调用的地址都不出现在接口描述里；令牌地址不带结尾 `/`。
- **完成线**：
  - A4、A7–A14 的接口版本通过，P1 的故事仍然通过；
  - 每个需要登录的操作都有 PAT 的测试；
  - 契约测试核对了全部需要登录的操作不带令牌时得到 401。

### P3 `web-auth`：前端认证
- **目标**：浏览器通过令牌管理器登录、续期、退出；Cookie 和 CSRF 从前端消失。
- **交付物**：
  1. `@nerve/api-client` 接入 web（`--root-types`）；令牌管理器和它的测试（7.1）。
  2. 传输层的清理：CSRF 4 处、表单提交、axios 基类（7.2）。
  3. 登录页、注册页、错误文案表（7.3）；`next_path`（3.18）。
  4. `AuthenticationWrapper`、`InstanceWrapper`（7.4）；user、profile、instance 三个 store 和相关类型（7.5 中与认证有关的部分）。
  5. 新手引导：删除"角色""用途"两步和 `is_self_managed`；资料步骤（3.19）；当前账户的头像读取（3.2）。
  6. `webui` 的 CSP（8.3）。
  7. 关键词规则（7.9 中 P3 的部分）。
  8. 端到端：
     - `signedInPage` fixture；
     - A1–A6、A10、A15 的页面版本；
     - S2。
- **关闭**：M0-P5；M1-P3 中 CSRF、认证错误、实例字段的部分；M1-P4；M0-P6 的 S2。
- **完成线**：
  - `git grep -n -E "csrfmiddlewaretoken|X-CSRFTOKEN" -- web` 没有输出；
  - 上述故事的页面版本通过；
  - 9.6 中 P3 的浏览器核对写进 review。

### P4 `web-account`：前端账户设置
- **目标**：个人设置的四个标签页对接新接口；M2 领域的前端清理完毕。
- **交付物**：
  1. general、preferences、security、api-tokens（7.7）；PAT store（7.5）；主题下拉框的定位。
  2. 决策点 3 的界面；决策点 1 的界面（选 A 时）。
  3. `@nerve/services` 删到只剩地址工具和文件工具；删除剩下的 Plane 类型（7.5）。
  4. 清理：
     - 死成员和死 prop 的 66 行；
     - oxlint：改到的文件清零，`no-unneeded-ternary` 一类清零，上限调低（7.8）。
  5. 关键词规则（7.9 中 P4 的部分）。
  6. 端到端：A7–A9、A11、A12 的页面版本。
- **关闭**：M1-P2；M1-closeout；M1-P3、M1-P4 剩下的事项。
- **完成线**：
  - 全部故事的页面版本和接口版本通过；
  - `domains.mjs --rows M2` 没有输出，或者剩下的每一行都在 review 中写明是误报；
  - 9.6 中 P4 的浏览器核对写进 review。

### 收尾 `closeout`
- 10 份交接逐项写下结论，`status` 改为 `closed`。
- 同步文档：差异清单（第 4 节的列和行为差异）、前端改动清单（3.1 中 M2 一行、3.2 中认证和 CSRF 两行）、总体设计和 M0 设计（3.20）。
- 写好给 M3、M4、M5、M8 的交接（13.2）。
- 规模估计与实际的对比（7.10）。
- 关键词守卫的例外只剩明确跨 M 的。
- 总体设计中 M2 的状态改为"已完成"。

---

## 13. 交接的落点

### 13.1 交给 M2 的 10 份交接
| 交接 | 事项 | 落点 |
|---|---|---|
| M0-P1-sqlc-cgo | sqlc 需要 cgo | P1。一律用 `CGO_ENABLED=0` 运行，不需要 C 编译器，也不需要 Docker 退路（3.14） |
| | sqlc 的解析器基于 PG 17 | P1。语法约定（3.13）；不写 `uuidv7()` |
| M0-P2-platform-notes | 1 `TxManager` 由使用方声明 | P1（`internal/shared`、`platform/postgres`） |
| | 2 认证、限流、接口调用日志按路由挂载 | P1（3.6）。接口调用日志由 M8 挂在限流之后（13.2） |
| | 3 导出 `RequestID` | P1 |
| | 4 新的密钥类配置写进 `LogValue` | P1（6.5） |
| | 5 River 与停机顺序、连接池关闭的时限、handler 的期限、River 的迁移 | P2（3.15） |
| | 6 规则 6 推广到 `adapter/*/gen` | P1（3.14） |
| | 7 迁移与就绪检查、迁移角色的权限 | P1（S1；README，见 6.1） |
| | 8 布尔配置的空值、logger 之后的致命错误、健康检查的日志 | P1（6.1） |
| M0-P3-api-codegen-notes | 1 生成选项 | P1（3.12） |
| | 2 错误映射、校验层、解码错误、413、`context.Canceled`、错误出口的测试、`CheckRequest` | P1（3.11） |
| | 3 `security` 的写法 | P1（3.12） |
| | 4 模块入口的演进 | P1（3.6） |
| | 5 接口描述的组织规则、map 型对象 | P1（3.12；M2 没有 map 型对象） |
| M0-P4-schema-conventions | 1 外键的 `DEFERRABLE` 和 `ON DELETE`、命名、默认值和 CHECK、系统表、Django 的索引、`varchar` | P1（3.13） |
| | 2 主键与 ID | P1（照旧：`uuid`，应用生成） |
| M0-P5-frontend-api-notes | 改调 `/api/v0/instance`；认证接口在 `/api/v0/` 下；同源 | P1（接口路径）、P3（前端） |
| | 安全响应头与 CSP 放在同一层 | P1（固定链）、P3（CSP，8.3） |
| M0-P6-e2e-notes | PAT 对等验收与认证 fixture | P1（接口）、P2（PAT）、P3（页面） |
| | `db.ts` 的连接池、断言的写法、失败时的数据库快照 | P1（9.5） |
| | S1 的迁移版本 | P1 |
| | S3 的新字段 | P2 |
| | S2 的"没有失败的接口请求"、控制台、`networkidle` | P3。登录页没有轮询，`networkidle` 继续可用；有轮询的页面不进 S2 |
| | 端口竞争的根本解决 | P1（`server.addr_file`） |
| | 新等待的期限 | P1–P4 |
| | River 停机是否仍在预算内 | P2 |
| | 是否录像 | P1 定为不录像，收尾同步总体设计 8.2（9.5） |
| | fixture 写法的延伸（`storage.ts`、`webhook.ts`、`clock.ts`） | 不属于 M2：M5、M8、M4 各自按 P6 的写法加入 |
| M1-P2-trim-content | 主题只剩一个值，五个取值，没有 `custom` 和调色板 | P2（接口，5.2）、P3（`IUserTheme` 删除，改用生成的类型） |
| M1-P3-trim-platform | 删除 Cookie 会话和 CSRF | P3（7.2） |
| | 认证错误就地显示 | P3（7.3） |
| | 修改登录邮箱 | 决策点 1；实现在 P2、P4 |
| | `IInstanceConfig` 的 4 个字段；`is_self_managed` 和新手引导的两步；`enable_signup` 的名字 | P2（接口，5.3）、P3（前端，3.19） |
| | 不再读的用户字段、不再调用的地址不出现在接口描述里 | P2；收尾再核对一次 |
| | 令牌地址统一不带结尾 `/` | P2（5.1） |
| M1-P4-router-native | 服务端校验 `next_path` | 3.18：服务端没有跳转；前端的校验补上控制字符（P3） |
| | 重写 `AuthenticationWrapper` | P3（7.4） |
| | 401 处理不照搬恒为真的判断 | P3（7.1、7.2） |
| | 由 M2 决定：`next_path` 带查询和片段 | 3.18：带上 |
| | 由 M2 决定：`security.tsx` 的错误断言 | P4（7.7） |
| | 由 M2 决定：表单不再提交到 `/auth/…` | P3（7.2） |
| M1-closeout | 死成员和死 prop（66 行） | P3、P4（7.8） |
| | oxlint：改到的文件清零，另清一类规则 | P3、P4（7.8）：`no-unneeded-ternary` |
| | 主题下拉框的位置 | P4（7.7） |

**没有落点的事项：无。** 唯一不在 M2 实现的是 M0-P6 最后一项（其他 fixture 的写法），它本来就写给 M4、M5、M8。

### 13.2 M2 交给后续 M 的事项（收尾时写成交接）
| 接收 | 事项 |
|---|---|
| M3 | 关闭注册时被邀请的邮箱仍可注册：`identity` 的注册用例加一个由 M3 实现的端口；Plane 把任何未删除的工作区邀请都算上（`plane/apps/api/plane/authentication/adapter/base.py:102-120`） |
| M3 | 登录后的工作区落点：`/users/me/settings/` 的数据和 `AuthenticationWrapper` 的落点取数（3.1）；`profiles.last_workspace_id` 是否补外键（`ON DELETE SET NULL`） |
| M3 | `workspace_creation_enabled` 的执行，以及关闭时是否提供创建工作区的命令（3.16） |
| M3 | 停用账户时一并停用成员关系（决策点 3 选 A 时） |
| M3 | 新手引导的创建工作区、加入工作区、邀请成员三步；`user.service.ts` 中留下的 `leaveWorkspace`、`joinProject`、`leaveProject`；`IUserLite` 的 `is_bot`；时区接口也供工作区和项目设置使用 |
| M4 | `user.service.ts` 中的 `getUserProfileIssues`；事件订阅者的写法（3.15）；建"物理删除软删除超过 60 天的数据"定时任务时，把 `api_tokens` 纳入 |
| M5 | `users.avatar_asset_id`、`cover_image_asset_id`，`User` 的 `avatar_url`、`cover_image_url`；general 页和新手引导资料步骤的上传控件；6 个文件的头像显示（3.2）；`file_size_limit` 的执行；CSP 的 `img-src`、`connect-src` 加上存储的来源 |
| M8 | 接口调用日志挂在限流之后（3.6）；Go 进程空闲内存实测时，一并测 argon2 并发上限下的峰值（3.8） |

---

## 14. 完成标准
- [ ] P1–P4 和收尾全部完成，每个 Phase 都有 spec、plan 和 review。
- [ ] A1–A15 的页面版本（有页面的）和接口版本全部通过，S1–S4 通过。需要登录的每个操作都有 PAT 版本，调用同一组数据库断言。
- [ ] 后端：单元测试、集成测试、契约测试、架构测试和 depguard 全部通过；`make gen-check` 覆盖 oapi-codegen、sqlc 和 TS 客户端。
- [ ] 前端：类型检查通过，knip 为零；oxlint 等于上限，上限已按 7.8 调低；前端单元测试通过；关键词守卫没有未登记的命中。
- [ ] `git grep -n -E "csrfmiddlewaretoken|X-CSRFTOKEN" -- web` 没有输出；前端不再调用 `/auth/…` 和 Plane 的用户、实例、令牌、时区地址（7.9 的规则守着）。
- [ ] 4 张表由 M2 的迁移创建，以 Plane 表结构快照为起点；差异清单登记了每一列的改动和每一条行为差异（4.6）。
- [ ] 第 10 节的三个决策点都有负责人的裁定，并已实现。
- [ ] 浏览器核对（9.6）的脚本全文写在各 Phase review 的附录中。
- [ ] `handoffs/` 中没有 `open` 的事项；交给 M3、M4、M5、M8 的交接已写好（13.2）。
- [ ] 总体设计、M0 设计、差异清单、前端改动清单已按 3.20 同步；总体设计中 M2 的状态改为"已完成"。
- [ ] 收尾 review 写明规模估计与实际的对比（7.10）。

---

## 15. Phase 进度表

| Phase | 名称 | 状态 | spec | plan | review |
|---|---|---|---|---|---|
| P1 | auth-core | 未开始 | — | — | — |
| P2 | account-api | 未开始 | — | — | — |
| P3 | web-auth | 未开始 | — | — | — |
| P4 | web-account | 未开始 | — | — | — |
| 收尾 | closeout | 未开始 | — | — | — |

---

## 16. 风险

| 风险 | 应对 |
|---|---|
| 重复使用检测过严：续期的响应在网络上丢失，客户端重试会被当成重复使用，用户被迫重新登录 | 同一浏览器内用 `navigator.locks` 避免并发续期；发现重复使用时记 WARN 日志，P3、P4 的核对中观察它是否频繁出现。真的频繁时，可以加"上一代令牌在几秒内仍可用"的宽限期，但要多存一代哈希，留到以后 |
| 每个带令牌的请求多一次数据库查询 | 按主键查询；M8 做性能和内存实测时一起观察 |
| openapi-fetch 0.17.0 的中间件能否重发请求 | P3 先做原型；不行时在薄 service 层统一包一层"401 后续期重试" |
| 生成的类型替换 `IUser` 等之后，牵连的使用方比 7.5 列出的多 | 类型检查会列出全部；超出估计时按领域拆成更小的任务 |
| sqlc 和 oapi-codegen 共用一个工具模块，依赖被一起抬高后 oapi-codegen 的输出变化 | P1 加入 sqlc 后立即运行 `make gen-check-go`；有变化时把 sqlc 放进单独的工具模块 |
| sqlc 的解析器只认 PG 17 | 3.13 的语法约定；真正需要 PG 18 的语法时，评估升级到已发布的新版 sqlc |
| River 仍是 0.x | 锁定 v0.47.0；升级时按 3.15 另写迁移，不改已发布的迁移 |
| argon2 占用内存 | 并发上限 4，约 76 MiB（3.8）；M8 实测 |
| CSP 挡住某个库的内联脚本或 `eval` | 内联脚本的哈希在启动时计算；P3 的浏览器核对覆盖主要页面；S2 统计违规事件 |
| 端到端的耗时增加 | 独立的 nerve 只用于 A2、A4、A15 三个故事；P4 实测并写进 review |
| 新手引导的工作区步骤在 M3 之前失败 | 预期行为（3.1）；页面版本只断言到"下一步出现" |
| 决策点的裁定晚于 P2 开始 | P1 不受影响；决策点的接口和命令放在 P2 的最后几个任务，界面放在 P4 |
