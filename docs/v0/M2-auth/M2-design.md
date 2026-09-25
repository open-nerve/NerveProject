# M2 账户认证：设计与实施规划

| 项 | 内容 |
|---|---|
| 里程碑 | M2 账户认证（`docs/v0/M2-auth`） |
| 日期 | 2026-09-25 |
| 状态 | 第二稿，待控制者复核。第 10 节的决策点 1 已由负责人裁定（2026-09-25，选 B）；决策点 2、3、4 待裁定；第 11 节有 1 项请负责人确认 |
| 上级文档 | [v0 总体设计](../v0-design.md) 1.1、3、4、5、6、7、8、9 节；[差异清单](../plane-diff.md)；[前端改动清单](../frontend-changes.md) |
| 前置交接 | `handoffs/` 中的 10 份：M0-P1、M0-P2、M0-P3、M0-P4、M0-P5、M0-P6、M1-P2、M1-P3、M1-P4、M1-closeout。逐条落到 Phase，见第 13 节 |
| 设计评审 | 第一稿（`f6ea660`）经独立评审（opus），结论 Ready with fixes：Critical 0、Important 14、Minor 18。第二稿按评审和控制者的裁定修订，逐条落点见第 17 节 |

---

## 0. 目标与范围

### 0.1 目标
M2 是第一个做真实业务的里程碑，也是前端第一次对接新接口。M2 结束时：
1. 任何调用方都能通过接口注册、登录、续期、退出、修改密码，管理自己的资料、偏好和个人访问令牌（PAT）。用 PAT 能做的事和页面一样多。
2. 后端第一次有业务表（4 张）、第一批 sqlc 查询、第一个 River 定时任务（清理过期会话）。以下几件平台上的事一次定下，后续 M 照做：
   - 认证：默认拒绝，只放行声明为公开的操作；当前账户（Actor）的传递；
   - 限流；
   - 错误码：在接口描述中逐个操作声明；取值校验放在哪一层；
   - 事务；
   - 表结构约定，以及 sqlc 的模块边界；
   - oapi-codegen 的生成选项。
3. 前端用令牌管理器替换 Cookie 会话和 CSRF。认证、用户、资料、偏好、实例配置、PAT 这几块的 services、stores 改用 OpenAPI 生成的类型，没有转换层。
4. 本领域的每个用户故事都有端到端测试：页面、数据库、接口三层断言；接口版本用 PAT（第 2 节）。
5. 持续集成的全部门禁通过；本 M 的交接全部关闭；给 M3、M4、M5、M8 的交接写清接收条件。

### 0.2 范围
| 包含 | 不包含（留给后续 M） |
|---|---|
| 注册、登录、续期（刷新令牌轮换和重复使用检测）、退出、修改密码 | 工作区、成员、邀请（M3）。其中包括：关闭注册时持有邀请的人仍可注册；登录后落到哪个工作区；新手引导中创建或加入工作区、邀请成员三步 |
| 当前账户的资料（`/me`）和偏好（`/me/profile`，含新手引导的进度） | 头像和封面的存储与上传（M5）。M2 的接口中 `avatar_url`、`cover_image_url` 恒为 `null`，见 3.2 |
| PAT：创建、分页列出、撤销；用 PAT 调用所有接口 | 权限框架和权限矩阵（M3） |
| 实例配置（在 M0 的 `instance` 上扩展）和时区列表 | 接口调用日志（M8） |
| 管理命令：`nerve users reset-password`、`nerve users set-email`（决策点 1 已裁定），以及决策点 2、3 带来的命令（第 10 节） | 邮件、找回密码、邮箱验证（v0 不做，总体设计 1.2）；用户自助修改邮箱（决策点 1） |
| River 的第一个定时任务（清理过期会话）；sqlc 的第一批查询；第一次按快照建表的约定 | 实例管理员（v0 没有，见 3.16） |
| 前端：令牌管理器；登录页、注册页；认证包装；新手引导的资料步骤；个人设置的四个标签页；删除 CSRF、Cookie 和表单提交 | 刷新令牌改放 HttpOnly Cookie（只写下升级路径，见 8.5） |
| 端到端：认证 fixture、PAT 对等验收、数据库断言的写法 | 可注入的时钟（M4，M0-P6 交接） |

### 0.3 关于文中的 spike
文中的"spike"是写设计时在仓库之外做的验证（放在临时目录，不进仓库）。结论、实测的数字和关键的行号都已写在正文中，读者不需要那些文件。其中"spike `server.gen.go:…`"指用仓库锁定的 oapi-codegen v2.8.0 为一份试验用的接口描述生成的代码；P1 实现时以仓库中生成的代码为准，行号会不同，顺序不变。

---

## 1. 现状基线（2026-09-25 实测，`main` 为 `559c3c6`）

| 项 | 数值与出处 |
|---|---|
| 服务端的模块 | 只有 `instance`（`server/internal/modules/instance/`）。还没有 `internal/shared` |
| 迁移 | 0 个。`server/migrations/sql/` 只有 `.gitkeep` |
| 接口 | 1 个操作：`GET /api/v0/instance`（`api/modules/instance.yaml`） |
| Go 依赖 | `server/go.mod` 没有 River、JWT，`golang.org/x/crypto` v0.55.0 只是间接依赖。`server/tools/go.mod` 只有 oapi-codegen v2.8.0 |
| 平台错误码 | `not_found`、`bad_request`、`internal_error`、`not_ready`（`server/internal/platform/httpserver/problem.go`） |
| 架构测试 | 10 条规则（`server/internal/archtest/rules_test.go`）。规则 4 只禁止平台导入模块和 `bootstrap`；规则 6 只认 `adapter/http/gen` |
| 路由 | `httpserver.NewMux` 返回 `*http.ServeMux`，`/api/` 下没有模块处理的路径一律 404 problem（`routes.go`）。生成的代码按 `"METHOD /path"` 注册路由；路径参数在中间件之前绑定，请求体在中间件之后解码（spike 的生成代码 `server.gen.go:119-141`、`:264-267`、`:477`） |
| 前端的请求方式 | 两个 axios 基类：`web/apps/web/core/services/api.service.ts`（75 行，`withCredentials: true`，401 时 `window.location.replace`）和 `web/packages/services/src/api.service.ts`（106 行，同样 `withCredentials: true`）。继承它们的 service 共 33 个（`rg -n "extends APIService" web` → 32 + 1）。全仓库没有任何地方带 `Authorization` 请求头 |
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
- 命令行和后台任务的故事（A13、A14、A16）没有页面版本。

| 编号 | 故事 | 页面 | 数据库 | 接口版本 |
|---|---|---|---|---|
| A1 | 新用户注册 | 在 `/sign-up` 填邮箱、密码、确认密码，进入 `/onboarding` 的资料步骤。浏览器里没有任何 Cookie，localStorage 里有刷新令牌 | `users` 新增一行：邮箱已转小写；`password` 以 `$argon2id$` 开头；`display_name` 是邮箱 @ 之前的部分；`is_active`。`profiles` 新增一行，各列是默认值。`auth_sessions` 新增一行：`generation = 0`，记下 UA 和 IP，`expires_at` 约为 30 天后，未撤销 | `POST /api/v0/auth/register`，同一组断言 |
| A2 | 注册被拒绝 | 邮箱已存在：错误就地显示在表单上方，不跳转，邮箱仍在输入框里。密码不合规：字段下方显示规则。密码太常见（例如 `Password1!`）：字段下方显示"密码太常见"。关闭注册（`signup_enabled = false` 的独立 nerve）：页头没有"注册"链接；直接打开 `/sign-up` 提交，显示"注册已关闭" | 四种情况都没有新增账户、资料和会话 | 409 `identity.email_taken`；422 `validation_failed`（`errors[].field = password`，`code` 分别是 `weak_password`、`common_password`）；403 `identity.signup_disabled` |
| A3 | 登录与 `next_path` | 未登录打开 `/settings/profile/general?tab=x#y`，跳到登录页，地址带编码后的 `next_path`。登录后回到原地址，查询参数和片段都在。`next_path` 为 `//evil.example`、`/\evil`、`javascript:…`、带控制字符时，登录后落到默认页。错误密码和不存在的邮箱：页面显示同一句提示 | 成功时新增一行会话；失败时没有 | `POST /api/v0/auth/login`；两种失败都是 401 `identity.invalid_credentials` |
| A4 | 续期与多标签页 | 访问令牌有效期 3 秒的独立 nerve。同一个浏览器上下文开两个标签页，过期后同时操作：两边都成功，没有跳到登录页；两次续期请求在时间上不重叠。同一个测试再跑一遍，用 `addInitScript` 删掉 `navigator.locks`，走 localStorage 租约（7.1） | 会话未被撤销；`generation` 等于续期次数；`last_refreshed_at` 已更新；`expires_at` 不变（绝对期限，3.5） | 用同一个刷新令牌依次续期：每次返回新的一对令牌，`generation` 递增，`refresh_token_expires_at` 不变 |
| A5 | 刷新令牌被重复使用 | 测试从 localStorage 取出刷新令牌，先在接口上用它续期一次（模拟被盗）。页面下一次续期时，会话被作废，跳到登录页，`next_path` 是当前地址 | `revoked_at` 已填，`revoke_reason = 'reuse_detected'` | 纯接口复现：旧令牌再用一次得到 401 `identity.refresh_token_invalid`；之后这个会话的访问令牌和最新的刷新令牌也都是 401 |
| A6 | 退出 | 用户菜单点"退出"，回到登录页，localStorage 里没有刷新令牌。同一上下文的另一个标签页也回到登录页（`storage` 事件） | `revoke_reason = 'logout'`；这个会话的访问令牌在下一个请求就得到 401 | `POST /api/v0/auth/logout`，同一组断言。用上一代刷新令牌退出：204，会话不变（3.5） |
| A7 | 修改密码 | 在安全页输入当前密码和新密码，成功提示，页面保持登录。当前密码错误：字段错误，数据库不变。安全页列出账户的 PAT，并说明修改密码不会撤销它们 | `users.password` 已改变；其他会话 `revoke_reason = 'password_changed'`，当前会话未撤销；`api_tokens` 不变；旧密码登录 401，新密码 200 | PAT 调用 `POST /api/v0/me/change-password`：因为 PAT 没有"当前会话"，全部会话都被撤销；PAT 本身继续可用 |
| A8 | 修改资料 | 在 general 页改名、姓和显示名，在偏好页改时区；刷新页面后仍是新值 | `users` 对应列和 `updated_at` | PAT 调用 `PATCH /api/v0/me` |
| A9 | 修改偏好 | 主题、语言、每周第一天；刷新页面后仍生效；主题下拉框在按钮旁展开（中英文各一次，第 9.6 节） | `profiles` 对应列 | PAT 调用 `PATCH /api/v0/me/profile` |
| A10 | 新手引导的资料步骤 | 新注册的用户在 `/onboarding` 填名字，进入下一步。创建或加入工作区的界面出现即可，M3 之前不提交 | `users.first_name`；`profiles.onboarding_step` 中 `profile_complete = true`，其余三项仍是 `false` | PAT 调用 `PATCH /api/v0/me`，以及只带一个键的 `PATCH /api/v0/me/profile {onboarding_step: {profile_complete: true}}`：其余三个键不变 |
| A11 | PAT 的创建、使用和撤销 | 创建（名称、说明、有效期 1 周），令牌只显示一次；列表中出现，但没有令牌原文；撤销后从列表消失 | `token_hash` 等于令牌的 SHA-256，表中任何一列都不含令牌原文；`expired_at` 约为 7 天后。用这个 PAT 调用 `GET /api/v0/me` 成功，`last_used` 被写入。撤销后 `deleted_at` 已填，再用它得到 401。把另一个 PAT 的 `expired_at` 改到过去，也得到 401 | 用另一个 PAT 创建、列出（翻页）、撤销（`DELETE /api/v0/api-tokens/{token_id}`） |
| A12 | 停用账户 | 按决策点 3 的裁定写（第 10 节） | 按裁定 | 按裁定 |
| A13 | 管理员重置密码 | — | `nerve users reset-password --email …` 从标准输入读新密码：`users.password` 改变；全部会话 `revoke_reason = 'password_reset'`；全部 PAT 的 `deleted_at` 已填；输出一行，带撤销的会话数和 PAT 数 | 用接口核对：新密码能登录；旧会话的刷新令牌和旧 PAT 都得到 401 |
| A14 | 过期会话被清理 | — | 把一行会话的 `expires_at` 改到过去，限时轮询，直到这一行被删除；未过期的会话仍在 | — |
| A15 | 登录限流 | 限流很低的独立 nerve：同一 IP + 邮箱失败到上限后，页面显示"尝试次数过多"；换一个邮箱仍能尝试，直到同一 IP 的总次数到达按 IP 的上限 | 没有新增会话 | 两个桶各触发一次：429 `rate_limited`，带 `Retry-After` |
| A16 | 管理员修改登录邮箱（决策点 1，已裁定为 B） | — | `nerve users set-email --email <旧> --new-email <新，含大写>`：`users.email` 变为规范化后的新邮箱；该账户全部会话 `revoke_reason = 'email_changed'`；PAT 不变；输出一行，带撤销的会话数。新邮箱已被别的账户使用：退出码 1，输出说明，数据库不变 | 用接口核对：旧邮箱登录 401，新邮箱登录 200；旧会话的刷新令牌 401；PAT 调用 `GET /api/v0/me` 返回新邮箱 |

**冒烟故事的更新**：
- S1：加上"迁移版本正确"：`nerve migrate status` 的每一行都是 `applied`，`goose_db_version` 的最新版本等于最后一个迁移文件（M0-P6 交接）。
- S2：登录页没有失败的接口请求（没有刷新令牌时不请求 `/me`，7.1），控制台没有 CSP 违规（8.3）。
- S3：`toEqual` 加上三个新字段的期望值（5.3）。

---

## 3. 设计裁定

以下裁定都在总体设计定下的范围内；改写已批准规则的一处放在第 11 节请负责人确认。3.3、3.6、3.10–3.15 是后续所有 M 都要照做的平台约定。

### 3.1 范围边界：登录之后看到什么
工作区接口在 M3 才有，M2 不为它造假数据，也不为它临时改变跳转规则。

- **没有完成新手引导的用户**：进入 `/onboarding`。
  - 资料步骤完全可用（A10）。
  - 下一步是创建或加入工作区。界面照常出现，但它调用的是 Plane 的旧接口，在 M3 之前得到 404 problem，页面按已有的错误处理显示失败。
- **已完成新手引导的用户**：没有 `next_path` 时落到 `/create-workspace`（和现在的规则一样：找不到上次的工作区就去这里）。表单能打开，提交同样得到 404，直到 M3。
- **个人设置** `/settings/profile/*` 不在工作区之下，四个标签页在 M2 全部可用。
- **需要"已完成引导"的用户的测试**，由 fixture 通过接口 `PATCH /api/v0/me/profile {is_onboarded: true}` 准备。这符合总体设计 8.2："前置数据通过接口准备，只有被测的那一步走页面"。
- **登录不再依赖工作区数据**：
  - 现在 `fetchCurrentUser` 先取 `/api/users/me/`，再用 `Promise.all` 同时取三样：资料、`/api/users/me/settings/`、工作区列表（`web/apps/web/core/store/user/index.ts:112-118`）。其中任何一个失败，登录就失败。
  - M2 让用户 store 只取 M2 的数据（`/me`、`/me/profile`）。
  - "已登录的用户落到哪个工作区"由 `AuthenticationWrapper` 在需要时单独取数；取数失败时回落到 `/create-workspace`，不影响登录。
  - 取数的那两个调用（设置、工作区列表）仍是 Plane 的旧接口，由 M3 换掉（交接）。

### 3.2 后续 M 的字段何时进入接口
前端很多地方读 M2 的类型，但字段背后的数据属于后面的 M。规则有三条，M3 起同样适用：

1. **数据在本 M 能真实产生的字段**，进入本 M 的接口。
2. **本身就可以为空的引用字段**（头像、封面、图标这类"可能没有"的东西），随它所属的实体一起进入接口。在产生它的 M 到来之前，它的值是 `null`。
   - 这是真实的值（确实还没有头像），不是占位：接口描述里它本来就可以为空，调用方本来就要处理 `null`。
   - 显示它的代码保留，它们已经处理"没有图片"的情况。
   - 只删掉在本 M 无法工作的上传控件（它们调用 Plane 的上传接口，在 Nerve 中必然失败），由产生它的 M 按新的上传协议加回。
3. **其他属于后面 M 的字段**，本 M 不定义，删掉前端的读取。不为它返回恒为 `null` 的占位。

| 字段 | 规则 | 处理 |
|---|---|---|
| `avatar_url`、`cover_image_url`（`User`） | 2 | M2 定义为可为 `null` 的字符串，恒为 `null`。显示当前账户头像的 6 个文件、读封面的 2 个文件不改。删掉 general 页的头像上传弹窗和封面选择器、新手引导资料步骤的头像上传。M5 加入 `users.avatar_asset_id`、`cover_image_asset_id`，接口返回签名地址（总体设计 4.3），按新的 `{method, url, headers}` 协议加回上传控件（前端改动清单 3.2） |
| `last_workspace_id`（`Profile`） | 1 | M2 保留。它是客户端写入的 uuid，数据库不设外键，和 Plane 一样（Plane `serializers/user.py:90-138` 在读取时判断是不是成员）。保留它可以避免 M3 领域的 4 个文件（`invitations/page.tsx`、`create-workspace/page.tsx`、新手引导的 `create.tsx`、`workspace-menu-root.tsx`）先删后加。是否补外键由 M3 决定 |
| 实例配置的 `workspace_creation_enabled`、`file_size_limit` | 1 | M2 定义，取自配置文件。M1 设计 3.7 已约定"实例配置接口的最终字段在 M2 定义"。服务端的执行分别由 M3（创建工作区）、M5（上传）负责，写进交接 |

- **M3 照此处理**：工作区图标（web 中 12 个文件读 `logo_url`）、项目封面（10 个文件读 `cover_image_url`）、成员头像（46 个文件读 `avatar_url`，大多是 `IUserLite` 的成员头像）。它们随工作区、项目、成员一起进入接口，M5 之前为 `null`，读取它们的代码不动。
- **为什么改了第一稿的规则**：第一稿让这类字段"由产生它的 M 加入，本 M 删掉读取"。按它，M3 要删掉约 60 个文件的读取，M5 再加回来，白白改两遍。

### 3.3 模块划分与端口的位置
- **`identity`**：一个模块，按总体设计 6.2 负责账户、资料与偏好、会话、PAT、密码。它的用例包括注册、登录、续期、退出、认证、资料、偏好、修改密码、PAT、重置密码和修改邮箱（命令行）、清理过期会话（定时任务），以及决策点 2、3 带来的用例。
  - 不再拆出 `user` 或 `auth` 模块：资料和凭证共用 `users` 表，修改密码要同时改账户和会话，拆开以后这些操作都成了跨模块调用。
- **`instance`**：在 M0 的试点上扩展：三个配置字段（3.2、5.3）和时区列表。
- **`internal/shared`**：M2 第一次建立（M0 设计 3.1 预计在 M2）。只放**值会跨越模块边界**的东西，只依赖标准库：
  - `Actor`（当前账户，3.6）：由认证放进 `context`，每个模块的 handler 都要读；
  - 领域错误 `Error` 和它的种类（3.11）：每个模块都返回它，由同一处映射为 problem；
  - `TxManager` 端口：同一个事务经 `context` 穿过多个模块的仓储（总体设计 6.4、M0-P2 交接 1）；
  - 分页游标的编解码：游标格式对所有列表接口相同。
  - 一个包，按文件分；不建子包。`Authorizer` 端口由 M3 加入。
- **其余端口由使用方在自己的 `app` 层声明**（M0 试点 `instance/app/ports.go` 的写法）：
  - 包括时钟：`identity/app` 声明自己的 `Clock { Now() time.Time }`，`platform/clock` 的实现按结构满足它。一个方法的接口在每个模块各写一遍，比放进 `shared` 更符合"接口由使用方定义"（总体设计 6.1），时钟的值也不跨越模块。
  - 总体设计 6.2 写的是端口放在 `domain`。M0 起实际放在 `app`：用例是端口的使用方，`domain` 只有纯规则、没有 I/O。收尾时把 6.2 改成现在的做法（3.20）。
- **平台不导入 `internal/shared`**（控制者裁定，保持总体设计 6.2"平台与业务无关"）：
  - `httpserver` 声明自己需要的小接口：认证器（3.6）和"可以映射为 problem 的错误"（3.11）。`identity`、`shared` 按结构满足它们。
  - `platform/postgres` 的事务管理器按结构满足 `shared.TxManager`，不导入 `shared`；`bootstrap` 里写一行编译期断言。
  - 架构测试规则 4 加上这一条："平台不导入模块、`bootstrap` 和 `internal/shared`"。
- **模块入口**（`module.go`）导出：`New(Deps)`、`Register(router, api)`、`PublicOperations()`（3.6）、`Authenticator()`、`Jobs()`、`Admin()`。
  - `Admin()` 返回命令行要调用的用例（重置密码、修改邮箱，以及决策点 2、3 带来的命令），命令的参数解析在 `cmd/nerve`，组合在 `bootstrap`（3.17）。

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
- **有效期**：访问令牌 15 分钟（`auth.access_token_ttl`）；会话从登录起 30 天（`auth.session_ttl`），刷新令牌随会话一起到期（3.5）。

### 3.5 会话、轮换和重复使用检测
- **一次登录一行 `auth_sessions`**。行的 id 就是访问令牌里的 `sid`。
- **绝对期限**：登录时 `expires_at = 登录时刻 + auth.session_ttl`（默认 30 天），之后**永不延长**。
  - 续期只换令牌，不动 `expires_at`。30 天后无论是否活跃，都要重新登录。
  - 理由：这是总体设计 4.1"30 天"的字面意思；它给被盗的刷新令牌定了一个上限。第一稿每次续期都把期限推后 30 天，一个每月至少用一次的会话永不结束，而且没有登记这个差异。
  - Plane 的会话固定 7 天，请求不会延长它（`SESSION_COOKIE_AGE = 604800`，`SESSION_SAVE_EVERY_REQUEST` 默认关闭，`plane/apps/api/plane/settings/common.py:373-376`）。Nerve 是 30 天，登记为差异（4.6）。
- **业务判断用的时间都来自用例的时钟**：用例从 `Clock` 取当前时刻，作为参数传给 SQL（`expires_at > $now`），不在 SQL 中写 `now()`。这样固定时钟的测试和数据库不会各说各话（3.13）。
- **续期**：
  - 客户端交出刷新令牌，服务端解出会话 id、代数 g 和密文。
  - 在一条 `UPDATE … WHERE id = $1 AND generation = $g AND token_hash = $h AND revoked_at IS NULL AND expires_at > $now RETURNING …` 中把代数加 1，换上新密文的哈希，写 `last_refreshed_at`。
  - 返回新的一对令牌。
- **判定**：`UPDATE` 没有命中时，读出这一行分情况处理：

  | 情况 | 结果 |
  |---|---|
  | 行不存在、已撤销、已过期 | 401 `identity.refresh_token_invalid` |
  | g 小于当前代数 | 旧令牌被再次使用。在同一个事务里撤销这个会话（`revoke_reason = 'reuse_detected'`），记一条 WARN 日志，返回 401（码同上） |
  | g 等于当前代数但哈希不符，或 g 大于当前代数 | 伪造的令牌：401，**不撤销**。否则任何人只要知道会话 id 就能把别人踢下线 |
- **已知代价**：
  - 知道会话 id（它在访问令牌里）的人，可以拼出一个"更旧的代数"让会话被撤销。但拿到访问令牌本身已经是更严重的泄露，所以接受。日志里只记会话 id 的哈希，不给看日志的人这个能力（8.4）。
  - 续期的响应在网络上丢失时，客户端重试会被判为重复使用，用户需要重新登录。这是严格检测的固有代价；同一浏览器内的并发续期由跨标签页的协调避免（7.1）。
- **退出**：只有**当前这一代、有效**的刷新令牌能撤销会话（`revoke_reason = 'logout'`）。
  - 未知、已过期、已撤销、上一代、伪造的令牌：204，什么都不做，不泄露它的状态，也不触发重复使用检测。
  - 前端退出时和续期共用同一把锁，所以交出的总是最新一代（7.1）。
- **撤销规则**：

  | 事件 | 撤销的会话 | 撤销的 PAT | `revoke_reason` |
  |---|---|---|---|
  | 退出 | 只有这一个会话（和 Plane 一样） | 无 | `logout` |
  | 修改密码 | 除当前会话外的全部；用 PAT 修改时没有当前会话，全部撤销 | 无（和 Plane 一样；安全页列出 PAT，由用户决定） | `password_changed` |
  | 管理员重置密码 | 全部 | **全部**（和 Plane 不同） | `password_reset` |
  | 管理员修改邮箱（决策点 1） | 全部 | 无 | `email_changed` |
  | 停用账户 | 全部 | 不删除，但认证要求账户未停用，所以同样失效 | `deactivated` |
  | 重复使用 | 这个会话 | 无 | `reuse_detected` |

  - "退出只结束当前会话"改写了总体设计 4.2 的原文，放在第 11 节请负责人确认。
  - **管理员重置密码为什么撤销 PAT**：v0 没有邮件，重置是账户被盗后唯一的恢复手段（总体设计 4.2）。PAT 可以创建永不过期的新 PAT，只撤销会话赶不走持有 PAT 的攻击者。命令输出撤销的会话数和 PAT 数，登记为差异（Plane 的 `reset_password` 只改密码，`plane/apps/api/plane/db/management/commands/reset_password.py:56-64`）。
  - **用户自己修改密码不撤销 PAT**：和 Plane 相同。用户可能正在用 PAT 跑脚本，修改密码不该让它们突然失效；安全页在修改密码的表单下列出 PAT，需要时就地撤销（7.7）。
  - **创建 PAT 不要求输入密码**：和 Plane 相同。
- **每个请求都查一次数据库**：验签通过后，用一条按主键的查询确认"会话未撤销、未过期，账户未停用"。
  - 这样退出、修改密码、停用在下一个请求就生效，不用等访问令牌过期。
  - 总体设计 4.2 本来就要求每个请求从数据库读取成员关系；多一次主键查询的代价可以接受。
  - PAT 同理：按 `token_hash` 查询，要求未删除、未过期、账户未停用。
  - `last_used` 最多每分钟写一次（`UPDATE … WHERE last_used IS NULL OR last_used < $now - interval '1 minute'`）。Plane 每个请求都写一次（`plane/apps/api/plane/api/middleware/api_authentication.py:41-42`）；Agent 高频调用时，每次都写太重。登记为行为差异。
- **清理**：`expires_at` 已过的行由 River 定时任务删除（3.15）。已撤销的行保留到它原本的过期时间，便于排查问题。

### 3.6 认证：默认拒绝
- **规则**：`/api/v0` 下的每个操作默认需要有效的令牌，只有模块声明为公开的操作例外。
  - Plane 同样默认要求登录（`DEFAULT_PERMISSION_CLASSES = IsAuthenticated`，`plane/apps/api/plane/settings/common.py:145`）。
  - 第一稿是"没有令牌就放行，由 handler 决定"，比它弱，而且整程序测试覆盖不到接口描述之外的路由和误标为公开的操作（评审 I1），已改掉。
- **公开操作由模块声明**：模块入口导出 `PublicOperations() []string`，内容是生成代码注册路由时用的模式字符串，例如 `"POST /api/v0/auth/login"`（spike `server.gen.go:264-267`）。
  - M2 的公开操作：`identity` 的 `register`、`login`、`refreshTokens`、`logout`；`instance` 的 `getInstance`、`listTimezones`。
- **中间件**：在 `platform/httpserver`，挂在生成代码的 `Middlewares` 里（按路由，M0-P2 交接 2）。它看到的请求已经由路由匹配过，`r.Pattern` 就是注册时的模式（Go 1.23 起 `ServeMux` 会填这个字段）。

  | 操作 | 没有令牌 | 令牌无效 | 令牌有效 |
  |---|---|---|---|
  | 公开 | 放行 | 放行，不认证 | 放行，不认证 |
  | 其他 | 401 `unauthorized` | 401 `unauthorized` | 认证器返回的 `context` 带着当前账户，交给下一层 |

  - 401 都带 `WWW-Authenticate: Bearer`；令牌无效时再加 `error="invalid_token"`。失败的原因只进 DEBUG 日志。
  - 公开操作不看 `Authorization`：页面在登录状态下读实例配置，不会因为访问令牌刚好过期而多一次 401。
- **认证器**（`httpserver` 声明，`identity` 实现，`bootstrap` 接上，M0-P3 交接 4）：
  ```go
  type Authenticator interface {
      // 成功时返回带当前账户的 context，以及这个凭证的限流键。
      Authenticate(ctx context.Context, token string) (context.Context, string, error)
  }
  ```
  - `identity` 的实现用 `shared.WithActor` 把 `Actor` 放进 `context`。平台不认识 `Actor`，也不导入 `shared`（3.3）。
  - 第二个返回值是限流键（`session:<会话 id>` 或 `pat:<PAT id>`）。平台读不到 `Actor`，限流中间件靠它按凭证计数（3.10）。
- **`shared.Actor`**：`{UserID, SessionID, APITokenID}`。通过会话认证时 `APITokenID` 为零值，通过 PAT 认证时 `SessionID` 为零值。这只区分凭证的种类，不区分账户的种类（总体设计 0.2 原则 1）。
  - handler 用 `shared.RequireActor(ctx)` 取当前账户。取不到时返回未认证错误（401 `unauthorized`）；有了默认拒绝，这只在接线出错时才会发生。
- **不用 oapi-codegen 的 `enable-auth-scopes-on-context`**：它是被标为"遗留"的兼容选项（`oapi-codegen/v2@v2.8.0/pkg/codegen/configuration.go:367-388`）。默认拒绝只需要"哪些操作公开"这一张表。
- **三个整程序测试**（`bootstrap`，P1）：
  1. 各模块声明的公开操作的并集，等于接口描述中 `security: []` 的操作集合。
  2. 注册在 `/api/v0` 下的路由模式，等于接口描述中的全部操作。
     - 接口描述之外的路由（例如 M5 本地存储的上传地址）会被发现：要么写进接口描述，要么不放在 `/api/v0` 下，并在那个 M 的设计中说明。
     - 为了拿到已注册的模式，`httpserver.NewMux` 改为返回 `*httpserver.Router`：它包着 `http.ServeMux`，在 `HandleFunc` 时记下模式。生成代码的 `BaseRouter` 只要求 `HandleFunc` 和 `ServeHTTP`（spike `server.gen.go:219-222`），`Router` 满足它。
  3. 每个非公开操作，不带令牌时得到 401 problem。路径参数填合法的示例值（例如全零的 uuid）：路径参数在中间件之前绑定，填错会先得到 400。
- **限流中间件**也在 `httpserver`（3.10），通过同样由 `httpserver` 声明的 `Limiter` 接口使用 `platform/ratelimit`。平台包之间不互相导入（规则 7）。
- **模块入口的演进**（M0-P3 交接 4）：
  - 平台交给模块的 HTTP 依赖合成一个值 `httpserver.API{Errors, Middlewares}`，模块写成 `Register(router, api)`。
  - `httpadapter.Register` 接收一个用例集合的结构体，不再逐个传指针。
- **按路由的中间件和顺序**：请求元信息（客户端 IP、UA）→ 请求期限 → 请求体上限 → 认证 → 限流 → handler。
  - oapi-codegen 的 `Middlewares` 列表中最后一个在最外层，所以列表要按相反的顺序写，并有测试核对。
  - 生成代码在这些中间件**之前**绑定路径参数和查询参数，在它们**之后**（strict handler 中）解码请求体（spike `server.gen.go:119-141`、`:477`）。所以参数格式错误的请求在认证和限流之前就得到 400。这类请求不碰数据库，不计入限流也没有代价。
  - **请求期限**：`server.request_timeout`（默认 15 秒）的 `context.WithTimeout`。handler 和用例里的数据库调用都带请求的 `context`，到期即取消。`server.write_timeout` 到期只让写出失败，不会取消请求的 `context`（M0-P2 交接 5），所以需要这一层。

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
- **并发上限和等待上限**：
  - 同时进行的哈希计算最多 4 个（`auth.password.max_concurrent_hashes`），4 × 19 MiB ≈ 76 MiB。
  - 拿不到名额时最多等 `auth.password.max_wait`（默认 2 秒），然后返回 503 `server_busy`，带 `Retry-After: 1`。第一稿的通道没有等待上限，名额被占满时请求一直排队到超时（评审 I3）。
  - **为什么是 503，不是 429**：429 的意思是"你这个调用方发得太多了"（RFC 6585），界面会显示"尝试次数过多"。名额满是整个服务器暂时过载，与这个调用方无关，RFC 9110 为此规定的是 503 加 `Retry-After`。通用的 HTTP 客户端和 Agent 也会按 503 重试。
  - 能用多少次哈希由 3.10 的限流约束，这里只保证过载时尽快失败。
- **规则**：在注册、修改密码、命令行重置三处，服务端都执行下面两条。Plane 在这三处用 zxcvbn 评分 ≥ 3（`plane/apps/api/plane/authentication/adapter/base.py:90-100`、`authentication/views/common.py:83`、`db/management/commands/reset_password.py:56`）。登录不检查（与 Plane 相同）。
  1. **组合规则**，与界面上已经显示的一致：
     - 长度 8–128 个字符；
     - 至少一个大写字母、一个小写字母、一个数字；
     - 至少一个特殊字符，取自 `!@#$%^&*()-_+=[]{}|;:'",.<>?/`。
     - 前四项与 `web/packages/utils/src/auth.ts:12-32` 的 `getPasswordStrength` 完全一致；128 的上限是新加的，界面同步加上。不合规时字段码是 `weak_password`。
  2. **常见密码名单**（评审 I2，控制者裁定）：
     - **判定**：把密码转小写，去掉开头和结尾的数字和特殊字符，得到"主干"。整个密码（小写）或主干在名单中，就拒绝；主干等于邮箱 @ 之前的部分，也拒绝（NIST SP 800-63B 建议排除与账户有关的词）。字段码是 `common_password`。
     - **为什么要取主干**：组合规则会挡掉名单中几乎所有原样的条目，只比原样没有用。spike 实测：前 10 万个里能原样通过组合规则的只有 294 个，前 1 万个里只有 12 个。常见的写法是"常见词，首字母大写，后面加数字和符号"：`Password1!`、`Summer2024!`、`Qwerty123!`、`Welcome1!`、`P@ssw0rd1`、`Dragon#2026` 的主干都在名单中，被拒绝；`Tr0ub4dor&3`、`Correct-Horse-9` 通过。
     - **来源**：英国国家网络安全中心（NCSC）发布的泄露最多的前 10 万个密码（`PwnedPasswordsTop100k.txt`，取自 Have I Been Pwned）。SecLists 收录为 `Passwords/Common-Credentials/100k-most-used-passwords-NCSC.txt`；spike 下载的文件 SHA-256 为 `c2e56968…c576e0`。
     - **许可**：NCSC 网站的内容是 Crown copyright，按 Open Government Licence v3.0 再使用，要求注明出处（NCSC 网站的条款页）。名单文件头和 README 的第三方声明写上"Contains public sector information licensed under the Open Government Licence v3.0"。P1 加入文件时再核对一次许可原文。
     - **规模**：前 10 万个。生成时只留下可能命中的条目：能原样通过组合规则的，以及本身可以作为主干的。剩 33,919 条、277 KB。去掉的条目不可能命中，结果与用全表相同。前 1 万个过滤后只有 5,066 条，挡不住 `Summer`、`Welcome` 之外稍长一点的常见词，所以用 10 万。
     - **嵌入**：排好序、每行一个的文本文件，用 `//go:embed` 放进 `identity/domain`（`embed` 是标准库，符合规则 2）。启动时切成有序切片，约 0.5 MB，二分查找。没有新依赖。
     - **生成**：`tools/password-blocklist/build.mjs` 读取下载的原文件，核对 SHA-256，过滤、转小写、去重、排序，写出文件和文件头（来源、许可、SHA-256、过滤规则）。原文件不进仓库。
- **为什么不照搬 zxcvbn**：
  - Go 的 zxcvbn 移植都已多年不维护，评分和 Python 版不会逐字相同，"照搬"本来就做不到；它的字典还会让程序多出近 1 MB。
  - 名单加主干的判定是确定的，容易测，失败时能明确告诉用户原因。
  - 登记为差异（4.6）：Plane 在服务端用 zxcvbn，组合规则只在界面上；Nerve 在服务端执行组合规则和名单。
  - 第一稿只保留组合规则，还把它说成"修复不一致"。这是错的：Plane 的两层检查是叠加的，不是矛盾的，只留组合规则是削弱。
- **修改密码不检查"新旧相同"**：Plane 也不检查（`ChangePasswordSerializer` 中的检查是死代码）。界面上已有的"新旧必须不同"提示保留。

### 3.9 账户枚举与时间
- **登录**：
  - 邮箱不存在和密码错误，返回同一个 401 `identity.invalid_credentials`。
  - 邮箱不存在时，仍然用同样的参数对一个启动时生成的假哈希做一次校验，让两种情况耗时相同。
  - Plane 会返回 `USER_DOES_NOT_EXIST`（`plane/apps/api/plane/authentication/views/app/email.py:90-104`），登记为差异。
- **停用的账户**：只在密码正确之后才返回 403 `identity.account_deactivated`，所以只对知道密码的人暴露账户状态。
- **注册**：邮箱已存在时必然失败，没有邮件通道就没法做到"不暴露"（Plane 同样返回 `USER_ALREADY_EXIST`）。用按 IP 的注册限流（3.10）压低探测速度，并在 8.2 写明这个局限。
- **令牌的比较**：刷新令牌由续期的 `UPDATE … WHERE token_hash = $h` 在数据库中匹配；PAT 按 `token_hash` 查唯一索引。被比较的是 256 位随机数的 SHA-256，比较耗时最多泄露哈希的前几个字节，据此推不出令牌，也拼不出另一个能用的令牌，所以不需要常数时间比较。第一稿写"用 `ConstantTimeCompare`"，与实际的匹配位置不符，已改。

### 3.10 限流
"具体数值在 M2 确定，默认参考 Plane"（总体设计 3.6）。实现是进程内按键的令牌桶（`golang.org/x/time/rate`），闲置的键定期清掉。

**桶**：

| 桶 | 键 | 默认 | Plane 的对应项 |
|---|---|---|---|
| `anonymous` | 客户端 IP | 每分钟 120 次 | 匿名请求 30/minute（`DEFAULT_THROTTLE_RATES`，`settings/common.py:140-144`）。Nerve 的页面每次加载都会不带令牌调用实例配置和续期，同一出口 IP 后的团队很快就会碰到 30 |
| `authenticated` | 凭证：会话 id 或 PAT id（3.6 的限流键） | 每分钟 1200 次，突发 200 | 页面请求不限流，API Key 每分钟 60 次。Nerve 的页面和 Agent 用同一套令牌，60 次连一次项目页面的加载都撑不住 |
| `login_ip` | 客户端 IP | 每分钟 30 次 | 认证接口合计每 IP 10/minute（`AUTHENTICATION_RATE_LIMIT`，`plane/apps/api/plane/authentication/rate_limit.py:26-30`） |
| `login_ip_email` | 客户端 IP + 规范化后的邮箱 | 每分钟 10 次 | 同上 |
| `register_ip` | 客户端 IP | 每分钟 10 次 | 同上 |
| `password_user` | 账户 id | 每分钟 5 次 | 无。所有需要校验密码的已认证操作都经过它；M2 中只有修改密码（改邮箱只在命令行，决策点 1） |

**每个操作经过哪些桶**（全部有余量才放行）：

| 操作 | 桶 | 哈希次数 |
|---|---|---|
| 登录 | `anonymous` → `login_ip` → `login_ip_email` | 1（邮箱不存在时是假哈希，同样 1 次） |
| 注册 | `anonymous` → `register_ip` | 1 |
| 修改密码 | `authenticated` → `password_user` | 2（校验旧密码，哈希新密码） |
| 续期、退出、实例配置、时区（其余公开操作） | `anonymous` | 0 |
| 其余需要登录的操作 | `authenticated` | 0 |
| 非公开操作带了无效的令牌 | `anonymous`（在返回 401 之前计数） | 0 |

- **为什么这样分**（评审 I3）：
  - 第一稿的登录只按 IP + 邮箱计数，换一个邮箱就是一个新桶；修改密码只受每凭证 1200 次的约束，而一个账户可以开任意多个会话和 PAT。任何人都能占满 4 个哈希名额，真正的登录排队到超时。
  - 现在同一个 IP 每分钟最多触发 40 次哈希（登录 30 + 注册 10），每个账户每分钟最多 5 次修改密码。4 个名额每秒能做约 100 次哈希（每次约 40 毫秒，P1 实测后写进 review），单个 IP 远远占不满。名额满了等 2 秒就答 503（3.8），不会无限排队。
  - 登录的"按账户"桶是 IP + 邮箱，不是只按邮箱：只按邮箱的话，任何人从任何地方都能把别人锁在门外。修改密码的请求已经认证过，按账户 id 计数没有这个问题。
  - 无效令牌计入按 IP 的桶：每次都要验签或查一次数据库，不计数就等于不限流。
- **位置**：
  - `anonymous`、`authenticated` 是 `httpserver` 的限流中间件。有限流键（3.6）时按凭证计数，没有时按 IP 计数。
  - 其余四个桶在 `identity` 的 HTTP 适配器里，在调用用例之前：登录的键里有邮箱，要先解析请求体；`password_user` 的键是当前账户。适配器通过自己声明的小接口使用 `platform/ratelimit`，超出时返回 `shared` 的限流错误，走"一条路"映射（3.11）。
- **超出**：429 `rate_limited`，带 `Retry-After`（秒，向上取整）。不加 `X-RateLimit-*`（登记为差异）。
- **客户端 IP**：
  - 默认取连接的对端地址。
  - `server.trusted_proxies`（CIDR 列表）非空，并且对端在列表中时，才从 `X-Forwarded-For` 从右往左取第一个不可信的地址。
  - 默认部署前面有 Caddy 时，要配置这一项。README 写明。
- **测试环境**：test 配置把上限都调得很高，否则同一个 IP 注册的大量测试账户会被限住。A15 用一个限流很低的独立 nerve。

### 3.11 错误码与取值校验
这是 M0-P3 交接 2 要求 M2 一次定下的体系。

- **领域错误**：`shared.Error{Kind, Code, Detail, Fields, RetryAfter}`。模块用 `shared` 的构造函数声明自己的错误，码带模块前缀，例如 `identity.email_taken`。

  | Kind | 状态 | 码 |
  |---|---|---|
  | `Invalid` | 422 | `validation_failed`，或模块码 |
  | `BadRequest` | 400 | `bad_request`（例如游标不合法，`errors[].field = cursor`） |
  | `Unauthenticated` | 401 | `unauthorized`，或模块码 |
  | `Forbidden` | 403 | 模块码 |
  | `NotFound` | 404 | `not_found`，或模块码 |
  | `Conflict` | 409 | 模块码 |
  | `RateLimited` | 429 | `rate_limited`，带 `Retry-After` |
  | `Unavailable` | 503 | `server_busy`，带 `Retry-After` |

- **平台怎样认出它**（平台不导入 `shared`，3.3）：`httpserver` 声明

  ```go
  type ProblemError interface {
      error                   // Error() 作为 problem 的 detail
      ProblemStatus() int
      ProblemCode() string
  }
  // 可选的两个方法：
  //   ProblemFields() []error   每个元素另有 ProblemField() string、ProblemCode() string，Error() 是 message
  //   RetryAfter() time.Duration
  ```

  - `shared.Error` 和 `shared.FieldError` 按结构满足它们，彼此不导入。
  - 状态码放在 `shared` 里不算越层：problem 的 `status` 是响应体的一个字段，本来就是错误对外契约的一部分。
  - 不满足 `ProblemError` 的错误一律 500 `internal_error`。`bootstrap` 的测试逐个 `Kind` 核对映射。
- **一条路**：handler 返回 `error`，由 `httpserver.APIErrors` 作为生成代码的 `ResponseErrorHandlerFunc` 统一映射为 problem+json。不用"每个操作声明带类型的 `default` 响应"那条路：那样每个 handler 都要自己拼 problem。
- **取值校验放在领域层**：
  - 生成的代码只做类型绑定和 JSON 解码，不校验取值，`required` 也不检查（M0/P3 已验证）。
  - 值对象的构造函数和用例入参的校验负责必填、长度、格式、枚举；一次收集全部字段的问题，返回一个 `validation_failed`。handler 只做类型转换。
- **零值也合法的必填字段**（评审 I6）：`required` 不检查，必填字段又生成为非指针，缺了这个字段和传了零值无法区分。规则：
  - **请求中的必填字段，如果零值也是合法的取值**（布尔、整数，以及允许空串的字符串），在接口描述中加 `writeOnly: true`。
    - oapi-codegen 对 `writeOnly` 的字段一律生成指针（`oapi-codegen/v2@v2.8.0/pkg/codegen/schema.go:156-168`）。spike 实测：`required` 加 `writeOnly` 的布尔、整数生成 `*bool`、`*int`。领域层把 `nil` 判为缺字段（字段码 `required`）。
    - 只用于只出现在请求中的结构。TS 类型不受影响：openapi-typescript 只在开启读写标记时才看 `writeOnly`。
  - **部分更新的嵌套对象**，写成全部字段都可省略的"部分对象"，由服务端合并。例如 `ProfileUpdate.onboarding_step`（5.2）。
  - 零值本身就不合法的必填字段（邮箱、密码这类不能为空的字符串）不需要处理：零值会被校验拒绝。
  - 改完 `onboarding_step` 以后，M2 的请求结构中没有需要 `writeOnly` 的字段。这条规则给后续 M。
- **已知的不一致**：
  - 请求中未声明的字段：生成的解码器忽略它们，无法配置。契约写的是 `additionalProperties: false`，服务端却接受。
  - 可省略、不可为空的字段传了 `null`（例如 `PATCH /me {"first_name": null}`）：解码为 `nil`，等同"没传"，返回 200 而不是 422。
  - 两者的处理相同：`apitest` 的 `CheckRequest` 让测试发出的请求也按契约校验；前端的请求由生成的 TS 类型在编译时约束；服务端不引入运行时的请求校验，因为 kin-openapi 被架构测试禁止链接进生产程序。
- **`FieldError` 加上 `code`**：
  - `{field, code, message}`。前端按 `problem.code` 和 `errors[].code` 查 `t()` 的文案，不直接显示服务端的英文 `message`（M1 收尾的规则：界面文案都走 `t()`）。
  - 字段码是一个封闭的小集合：`required`、`invalid_format`、`too_short`、`too_long`、`out_of_range`、`not_allowed`、`weak_password`、`common_password`、`must_be_future`、`contains_url`。
  - `api/common.yaml` 的 `FieldError` 和 `httpserver.FieldError` 同步修改。
- **错误码写进接口描述**（评审 I13，约束后续每个 M）：
  - 每个操作用扩展字段 `x-problem-codes` 列出它可能返回的错误码。
    - 所有操作都可能返回的四个码（`bad_request`、`payload_too_large`、`rate_limited`、`internal_error`）只写一次，放在 `api/openapi.yaml` 顶层的 `x-problem-codes`。
    - 声明了 `bearer` 的操作隐含 `unauthorized`。
  - `apitest` 核对三件事：
    1. 写法：码的格式是 `^([a-z]+\.)?[a-z_]+$`；带前缀的码，前缀等于所在模块文件的名字；不带前缀的码必须是下面表中的平台码。
    2. `CheckResponse` 遇到 problem 时，核对它的码在这个操作允许的集合里（顶层、操作自己、`unauthorized`）。
    3. 一个模块的 handler 测试跑完时，这个模块每个操作声明的码都至少被一个测试返回过。声明了却从不返回的码，是写错了的契约。
  - 前端的"错误码 → 文案"表（7.3）由 vitest 核对：从 `api/dist/openapi.yaml` 读出全部码，表的键必须与之相等。
  - **为什么用扩展字段，不用每个模块一个枚举**：
    - `Problem` 是 `common.yaml` 里的一个公共结构。要给 `code` 按模块缩窄，就得为每个模块派生一个 `Problem`，而几个 `additionalProperties: false` 的结构用 `allOf` 组合会互相拒绝对方的字段（5.2）。
    - 枚举只能说明"这个模块有哪些码"，说明不了"这个操作会返回哪些码"。Agent 调用一个操作时需要的是后者。
    - 生成器不认识扩展字段，生成的 Go 代码和 TS 类型都不变，"一条路"的错误处理不用改。
- **平台错误码**：

  | 码 | 状态 | 来源 |
  |---|---|---|
  | `bad_request` | 400 | 已有。请求体不是合法 JSON：`detail` 是通用的一句话。类型不符：`errors[{field, code: "invalid_format"}]`，字段路径取自 `json.UnmarshalTypeError`，不再带出 Go 的类型名（M0-P3 交接 2）。游标不合法 |
  | `unauthorized` | 401 | 新增。认证中间件，以及 `RequireActor` |
  | `not_found` | 404 | 已有 |
  | `payload_too_large` | 413 | 新增。`http.MaxBytesError`；JSON 接口的请求体上限是 `server.max_body_bytes`（默认 1 MiB） |
  | `validation_failed` | 422 | 新增。取值校验 |
  | `rate_limited` | 429 | 新增 |
  | `internal_error` | 500 | 已有 |
  | `not_ready` | 503 | 已有，只用于 `/readyz` |
  | `server_busy` | 503 | 新增。密码哈希的名额等待超时（3.8） |
- **`context.Canceled`**：客户端断开，不记 500，也不记 ERROR 日志，只在 DEBUG 记一行。
- **错误出口的测试**：参数绑定、请求体解码、handler 返回错误三个出口都要测，并对 problem 调 `CheckResponse`（M0-P3 交接 2）。

### 3.12 接口描述与代码生成的约定
- **oapi-codegen 的模块模板**（M0-P3 交接 1，从 M2 起每个模块照抄；spike 已验证下面的选项能生成、能编译，0.3）：
  - `output-options.nullable-type: true`：可为空又可省略的字段生成 `nullable.Nullable[T]`，PATCH 能区分"没传"和"传 `null`"。依赖 `github.com/oapi-codegen/nullable` v1.2.0，写死。
  - `output-options.type-mapping.string.formats.uuid: {type: uuid.UUID, import: uuid}`：映射到标准库。
  - `prefer-skip-optional-pointer` 保持默认（`false`）：可省略的字段是 `*T`，`nil` 表示没传。
  - 第一个带路径参数的操作会让生成代码导入 `github.com/oapi-codegen/runtime`，写死为 v1.7.0。
  - 加依赖之后核对 `server/go.mod` 仍是 `go 1.27` / `toolchain go1.27.1`。
- **`security` 的写法**（M0-P3 交接 3）：
  - 每个操作都显式声明 `security`：需要登录的写 `[{bearer: []}]`，公开的写 `[]`。
  - `securitySchemes.bearer` 同时写在 `api/openapi.yaml` 和每个模块文件里。
  - `apitest` 的写法检查加两条：每个操作都有 `security`；引用的 scheme 都存在。
- **`x-problem-codes`**：每个操作都写，写法见 3.11。
- **分页的公共组件由 M2 加入**：
  - `common.yaml` 加 `Limit`（1–100，默认 50）、`Cursor` 两个 parameter 和 `NextCursor`（`type: [string, 'null']`）。
  - M0/P3 原计划由 M3 的第一个列表接口加入；M2 的 PAT 列表先用到。
  - 游标是 base64url 编码的 `(created_at, id)`，编解码在 `shared`。游标不合法时返回 400 `bad_request`，`errors[].field = cursor`。
- **路径**按总体设计 3.2："列表挂在父资源下，单个资源用短路径"。PAT 的列表和创建是 `/me/api-tokens`，撤销是 `DELETE /api-tokens/{token_id}`。第一稿写的 `/me/api-tokens/{token_id}` 不符合 3.2，已改（评审 M17）。
- **组织规则**（M0-P3 交接 5）照旧：
  - 跨模块共用的类型放 `common.yaml`；
  - 一个路径只属于一个模块文件（`/me/*`、`/api-tokens/*`、`/auth/*` 属于 `identity`，以后的 `/me/recent-visits` 属于它自己的模块）；
  - 模块文件名等于模块目录名。
  - M2 没有 map 型对象，不需要扩展 `closedObject` 的例外。
- **TS 类型**：
  - `@nerve/api-client` 的生成命令加 `--root-types --root-types-no-schema-prefix`：`User`、`Profile` 等直接从 `@nerve/api-client` 导出（openapi-typescript 7.13.0 的 `bin/cli.js:34-36` 支持这两个选项）。
  - stores 和组件直接导入这些名字，`@nerve/types` 中对应的 Plane 类型删除（7.5）。

### 3.13 表结构约定（第一次按快照建表，以后每个 M 照做）
M0-P4 交接要求在第一次建表时一次定下。这些约定改变了 Plane 表结构的全局写法，收尾时登记到差异清单二·全局（3.20）。

| 事项 | 约定 | 理由 |
|---|---|---|
| 外键的 `ON DELETE` | 照搬每个外键在 Django 模型中的 `on_delete`：`CASCADE` → `ON DELETE CASCADE`，`SET_NULL` → `ON DELETE SET NULL`，`PROTECT` → `ON DELETE RESTRICT`，`DO_NOTHING` → 不写 | Plane 在 Python 里处理级联，快照中 474 个外键都没有 `ON DELETE`。把它的语义搬进数据库，符合总体设计 5.1 第 3 类改动。软删除的连带删除仍由用例在同一个事务里完成（总体设计 5.5） |
| `DEFERRABLE` | 不用 | Django 需要它来批量插入；Nerve 在同一个事务里按"先父后子"的顺序写 |
| 约束和索引的名字 | 主键、外键、唯一约束，以及只涉及一列、每列至多一个的 CHECK，用 Postgres 的默认名（`<表>_pkey`、`<表>_<列>_fkey`、`<表>_<列>_key`、`<表>_<列>_check`）。**涉及多列的 CHECK，或同一列的第二个 CHECK，必须显式命名**：`<表>_<含义>_check`。索引必须命名：`<表>_<列>_idx`；部分唯一索引写成 `<表>_<列>_key` | 快照中的名字带 Django 的哈希后缀，而且有 29 张表的主键名与表名不对应（M0-P4 交接）。表级 CHECK 的默认名随创建顺序变化：在开发库中用回滚的临时表实测，多列的是 `<表>_check`、`<表>_check1`，同一列的第二个是 `<表>_<列>_check1`。按约束名映射错误、以后 `DROP CONSTRAINT` 都需要稳定的名字 |
| 默认值和取值范围 | 从 `plane/apps/api/plane/db/models/` 读取，写进 `DEFAULT` 和 `CHECK`；`created_at`、`updated_at` 的默认值是 `now()`；`id` 不设默认值 | 总体设计 5.3、5.5 |
| Django 的系统表 | 不照搬 | M0-P4 交接 |
| `*_like`（`varchar_pattern_ops`）索引 | 不建 | 它们只服务 Django 的 `LIKE 'x%'`，Nerve 按等值查找 |
| 外键列的索引 | 只给查询或级联真正用到的外键列建索引；`created_by_id`、`updated_by_id` 不建 | Django 给每个外键都建了 btree，多数用不上 |
| `varchar(n)` 还是 `text` | 照搬 Plane 的列类型；领域层的长度校验与 `n` 一致 | 两者在 Postgres 中性能相同；照搬就不用逐列讨论 |
| 语法 | 只用 sqlc 解析器能解析的 PG 17 语法：不用 PG 18 的 `VIRTUAL` 生成列、`RETURNING old/new`、`WITHOUT OVERLAPS`、`NOT ENFORCED`；不用 `MERGE`（sqlc v1.31.1 会不报错地生成错误的代码）；不写 `DEFAULT uuidv7()` | spike 实测（3.14）；M0-P1 交接 2 |
| 时间 | 连接池把 `timestamptz` 的扫描时区设为 UTC（pgx 的 `TimestamptzCodec.ScanLocation`，`pgx/v5@v5.11.0/pgtype/timestamptz.go:130-132`），接口输出的时间一律以 `Z` 结尾。**做业务判断用的时刻由用例的 `Clock` 提供，作为参数传给 SQL**；`DEFAULT now()` 只填 `created_at`、`updated_at` 这类审计列，没有逻辑拿它们做判断 | 总体设计 3.1 要求 UTC；spike 实测 pgx 默认按本地时区返回。时间只有一个来源，固定时钟的测试才和数据库一致（评审 M3） |

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
  - 输入：`schema:` **只列本模块的迁移文件**；`queries: internal/modules/<m>/adapter/postgres/queries`。
  - 输出：`out: internal/modules/<m>/adapter/postgres/gen`，`sql_package: pgx/v5`。
  - 类型：`emit_pointers_for_null_types: true`。
  - 覆盖：`uuid` → 标准库 `uuid.UUID`（可空的是 `*uuid.UUID`），`timestamptz` → `time.Time`（可空的是 `*time.Time`）。注意 `db_type` 要写 `timestamptz`，写 `pg_catalog.timestamptz` 不会匹配。
  - 以上都经 spike 验证：映射到没有点号的标准库包 `uuid` 可以生成、编译，pgx v5.11.0 能读写。
- **模块边界：一个模块的查询只能碰本模块的表**（评审 M18，控制者裁定）：
  - 由 sqlc 自己执行：`schema:` 里没有的表，查询编译时就报错。spike 实测：`workspace` 条目的查询读 `users`，得到 `relation "users" does not exist`；`workspace` 自己的迁移里 `REFERENCES users` 可以通过。所以跨模块的外键照写，跨模块的读取写不出来。
  - 由一个架构测试守住配置本身（`archtest` 中的 `TestSQLCSchemaScope`）：
    - 迁移文件名是 `<版本>_<模块>_<内容>.sql`；
    - 每个模块的迁移只出现在这个模块的条目里，不出现在别的条目里；
    - `river` 开头的迁移不属于任何模块的条目（模块通过 River 客户端使用它，不查它的表）；
    - 有 `adapter/postgres/queries` 的模块都有条目。
  - 需要读别的模块的数据时，按总体设计 6.3 规则 2 在自己的 `app` 层声明端口。确实需要在 SQL 中联表的，由那个 M 的设计写明理由，作为测试中列出的例外。M3 的 `access` 模块读成员表是第一个要回答的问题（交接）。
- **写法约定**（来自 spike）：
  - 行比较的游标条件 `(created_at, id) < ($1, $2)` 会让第二个参数被推断成时间类型，运行时报错。写成 `(created_at, id) < (sqlc.arg(cursor_created_at)::timestamptz, sqlc.arg(cursor_id)::uuid)`。
  - PATCH 只更新传入的字段（总体设计 3.6"同一字段并发修改时以后写入的为准"），写法是 `col = CASE WHEN sqlc.arg(set_col)::boolean THEN sqlc.arg(col) ELSE col END`。
  - 不用"读出整行、改完整行写回"：那会覆盖别人同时改的其他字段。
  - 时间比较用参数，不用 `now()`（3.13）。
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
  - `platform/jobs` 能创建两种客户端：服务用的（配队列，调用 `Start`），命令行用的"只投递"客户端（不配队列，不调用 `Start`，`river@v0.47.0/client.go:89-95`）。见 3.17。
- **第一个定时任务**：`identity.cleanup_expired_sessions`。
  - 用 `river.NewPeriodicJob(river.PeriodicInterval(auth.session_cleanup_interval), …, &river.PeriodicJobOpts{ID: …, RunOnStart: true})`。间隔默认 1 小时，test 配置为 2 秒。
  - 它删除 `expires_at` 早于用例时钟当前时刻的会话。
  - River 没有内置 cron（spike：`periodic_job.go` 只有 `PeriodicInterval` 和一个 `Next(time.Time)` 接口），M2 也不需要 cron，不引入 `robfig/cron`。
- **生命周期**（M0-P2 交接 5）：
  - `bootstrap` 的 `run` 让 HTTP 和 River 一起运行。
  - 停机顺序：HTTP 优雅停机 → River 停止（`jobs.shutdown_timeout`，默认 10 秒）→ 迁移执行器 → 连接池。
  - `pool.Close()` 放在一个带时限（5 秒）的协程里：不理会 `ctx` 的 handler 可能还占着连接。
  - handler 里的阻塞调用由请求期限约束（3.6）。
  - 端到端 fixture 的停机预算是 `readyTimeoutMs + stopTimeoutMs + 10` 秒，P3 实测停机时间，写进 review（M0-P6 交接）。
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
  - 按 3.8 的规则检查密码，改写哈希。
  - 撤销全部会话（`password_reset`）和**全部 PAT**（3.5），在一个事务里完成。
  - 输出一行：`password reset for <email>: revoked <n> sessions, <m> API tokens`。
  - 账户不存在时退出码为 1，并说明原因。
- **`nerve users set-email --email <旧邮箱> --new-email <新邮箱>`**（决策点 1，负责人 2026-09-25 裁定为 B）：
  - 两个邮箱都按注册时的规则规范化；新邮箱也按注册时的规则校验格式。
  - 新邮箱已被别的账户使用：退出码为 1，输出一句说明（`email <新邮箱> is already used by another account`），数据库不变。新旧邮箱规范化后相同：同样退出码为 1，说明没有变化。旧邮箱对应的账户不存在：退出码为 1。
  - 在一个事务里改写 `users.email`，撤销该账户的全部会话（`email_changed`）。PAT 不撤销：改邮箱不是账户被盗后的恢复手段。
  - 输出一行：`email changed from <旧> to <新>: revoked <n> sessions`。
  - 没有界面，也没有接口：个人设置里的邮箱只读（7.7）。
- **决策点 2、3 带来的命令**（第 10 节）：`nerve users create`（决策点 2 选 B 或 C）、`nerve users deactivate` 和 `activate`（决策点 3 选 A 或 B），按裁定加入。
- **命令的实现**：
  - `cmd/nerve` 只解析参数。
  - `bootstrap` 建一个最小组合：连接池、`identity` 的 `Admin()` 用例，以及一个只投递的 River 客户端（3.15）；不启动 HTTP，也不处理任务。
  - M2 的命令还不投递任务。组合里放上只投递的客户端，是为了以后的用例在事务中投递任务时（例如 M4 的领域事件），命令行不会因为缺少客户端而坏掉。它和服务共用同一段组合代码，不是专门为命令行多写的。
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

### 3.19 新手引导的"角色""用途"两步：见决策点 4
- **事实**：
  - Plane 把 `IS_SELF_MANAGED` 写死为 `True`（`plane/apps/api/plane/settings/common.py:54`），为真时新手引导跳过"角色""用途"两步（`web/apps/web/core/components/onboarding/root.tsx:74`，进度条和返回按钮在 `header.tsx:45,57`）。Plane 自托管版的用户从来看不到这两步。
  - Nerve 只有自托管一种形态。
  - M1-P3 交接把它列为产品决定。第一稿直接裁定删除，没有交给负责人（评审 M15），现在列为决策点 4。
- **两种裁定各自的做法**：

  | 裁定 | 前端 | 数据与接口 |
  |---|---|---|
  | A 删除（建议） | 删除 `steps/role/`、`steps/usecase/`（共 338 行），以及 `onboarding/root.tsx`、`onboarding/header.tsx` 中读 `is_self_managed` 的分支；资料步骤固定走自托管的那条路 | 实例配置不定义 `is_self_managed`。`profiles.role`、`use_case` 不建：它们没有别的写入方，general 页只是在提交时把 `role` 的默认值"Product / Project Manager"写回去，没有输入框（`web/apps/web/core/components/settings/profile/content/pages/general/form.tsx:77,146-151`） |
  | B 显示这两步 | 删除 `is_self_managed` 的分支，两步总是显示；删除 general 页写回默认 `role` 的逻辑 | 实例配置同样不定义 `is_self_managed`（恒为真的配置没有意义）。`profiles.role`、`use_case` 保留，进入 `Profile` 和 `ProfileUpdate` |

- **裁定晚于 P1 时**：P1 的 `profiles` 迁移按建议 A 建表；选 B 就在 P3 加一个迁移补上两列。前端的改动在 P4，不受影响。

### 3.20 需要同步到上级文档的地方（收尾时同步）
| 文档 | 位置 | 内容 |
|---|---|---|
| 总体设计 | 3.1 | 动作接口的返回值：注册返回令牌，不返回新建的账户（账户由 `GET /me` 读取）；修改密码返回 204（5.1） |
| 总体设计 | 3.5 | 平台错误码加入 `unauthorized`、`payload_too_large`、`validation_failed`、`rate_limited`、`server_busy`；`FieldError` 带 `code`；每个操作在接口描述中用 `x-problem-codes` 声明错误码（3.11） |
| 总体设计 | 3.6 | 限流的桶和默认数值（3.10） |
| 总体设计 | 4.1 | 刷新令牌的 30 天是从登录起算的绝对期限，续期不延长（3.5） |
| 总体设计 | 4.2 | 退出只结束当前会话（负责人确认后，第 11 节）；管理员重置密码同时撤销全部 PAT（3.5） |
| 总体设计 | 4.3 | 浏览器没有 `navigator.locks` 时，用 localStorage 租约协调续期（7.1） |
| 总体设计 | 6.2 | 端口由使用方在 `app` 层声明（M0 起的做法）；`shared` 的内容是 Actor、领域错误、`TxManager`、分页游标，时钟端口由各模块自己声明（3.3） |
| 总体设计 | 6.3 | 架构测试：平台不导入 `internal/shared`；sqlc 按模块限定 `schema`（3.3、3.14） |
| 总体设计 | 6.4 | 固定链由三个中间件变为四个（加上安全响应头）；按路由挂载的中间件及其顺序，路径参数在它们之前绑定（3.6） |
| 总体设计 | 8.2 | 失败时保存 trace、截图、nerve 日志和数据库快照，不录像（9.5） |
| 总体设计 | 9.4 | M2 的状态 |
| M0 设计 | 3.2 | 原文"M2 加入 `signup_enabled`，M5 加入文件大小上限"改为：M2 加入 `signup_enabled`、`workspace_creation_enabled`、`file_size_limit`（5.3） |
| M0 设计 | 3.3 | 固定链由三个中间件变为四个，加上安全响应头（8.3）；认证、限流等按路由挂载的中间件及其顺序（3.6） |
| M0/P3 spec | 第 7 节 | 分页的公共组件由 M2 加入（3.12） |
| 差异清单 | 二·全局 | 3.13 的全局约定：外键的 `ON DELETE` 写进数据库；不用 `DEFERRABLE`；约束和索引一律改名（默认名，表级 CHECK 显式命名）；不建 `*_like` 索引和多数外键列的索引 |
| 差异清单 | 二·按表 | 第 4 节各表的列；`auth_sessions` 一行按 4.5 的实际列改写 |
| 差异清单 | 三 | PAT 撤销的路径 `/api-tokens/{id}`；错误码在接口描述中逐个操作声明 |
| 差异清单 | 四 | 4.6 的行为差异 |
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
| `00004_identity_api_tokens.sql` | `api_tokens` | P3 |
| `00005_river_main_v2_to_v7.sql` | River 主线第 2–7 版（3.15），`StatementBegin`/`End` 包住整段 | P3 |

`00005` 的 Down 段是 `migrate-get --down` 的原样输出。每个迁移都要能 up、down、再 up（`platform/postgres` 的迁移测试沿用 M0 的写法）。

### 4.2 `users`（Plane 40 列 → 10 列）
| 列 | 类型与约束 | 与 Plane 的差异 |
|---|---|---|
| `id` | `uuid PRIMARY KEY` | 照搬 |
| `email` | `varchar(255) NOT NULL UNIQUE`，`CHECK (email = lower(email))` | Plane 可以为空；规范化（去掉首尾空白、转小写）原来只在 Python 里做，现在由 CHECK 保证 |
| `password` | `varchar(128) NOT NULL` | 列名照搬；内容改为 argon2id 的 PHC 字符串（约 97 个字符） |
| `first_name`、`last_name` | `varchar(255) NOT NULL DEFAULT ''` | 照搬 |
| `display_name` | `varchar(255) NOT NULL`，`CHECK (display_name <> '')` | 照搬（Plane 的 `blank=False`）。注册时取邮箱 @ 之前的部分（`plane/apps/api/plane/db/models/user.py:169-187`） |
| `user_timezone` | `varchar(255) NOT NULL DEFAULT 'UTC'` | 照搬默认值。Plane 限定为 `pytz.common_timezones`；Nerve 接受 Go 的 `time.LoadLocation` 认得的 IANA 名称，`Local` 除外；程序内嵌 `time/tzdata`。登记为差异（4.6） |
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
- **由 M5 加入**（3.2）：`avatar_asset_id`、`cover_image_asset_id`。在那之前，接口的 `avatar_url`、`cover_image_url` 是 `null`，不需要列。

名字不能含网址：Plane 对 `first_name`、`last_name` 用 `contains_url` 检查（`plane/apps/api/plane/app/serializers/user.py:16-24`，正则在 `plane/utils/url.py:12-53`），照搬。

### 4.3 `profiles`（Plane 29 列 → 11 列）
| 列 | 类型与约束 | 与 Plane 的差异 |
|---|---|---|
| `id` | `uuid PRIMARY KEY` | 照搬 |
| `user_id` | `uuid NOT NULL UNIQUE REFERENCES users ON DELETE CASCADE` | 照搬（`OneToOne`、`CASCADE`） |
| `theme` | `varchar(20) NOT NULL DEFAULT 'system'`，`CHECK (theme IN ('system','light','dark','light-contrast','dark-contrast'))` | Plane 是 `jsonb`，默认 `{}`，存自定义调色板。自定义主题已删除（M1-P2 交接），只剩一个值，改为字符串 |
| `is_tour_completed` | `boolean NOT NULL DEFAULT false` | 照搬 |
| `onboarding_step` | `jsonb NOT NULL DEFAULT '{"profile_complete":false,"workspace_create":false,"workspace_invite":false,"workspace_join":false}'`，`CHECK (jsonb_typeof(onboarding_step) = 'object' AND onboarding_step ?& array['profile_complete','workspace_create','workspace_invite','workspace_join'])` | 照搬默认值（`get_default_onboarding()`）。四个键都在，由 CHECK 保证；更新时由领域层把传入的部分合并进去，值都必须是布尔值 |
| `is_onboarded` | `boolean NOT NULL DEFAULT false` | 照搬 |
| `last_workspace_id` | `uuid`（可空，不是外键） | 照搬（3.2） |
| `language` | `varchar(255) NOT NULL DEFAULT 'en'`，`CHECK (language IN ('en','zh-CN'))` | Plane 接受任意字符串；Nerve 只剩两种语言（M1 设计第 6 节） |
| `start_of_the_week` | `smallint NOT NULL DEFAULT 0`，`CHECK (start_of_the_week BETWEEN 0 AND 6)` | 照搬（Plane 的取值 0–6） |
| `created_at`、`updated_at` | `timestamptz NOT NULL DEFAULT now()` | 照搬 |

删除的 18 列：
- 按决策点 4 的建议 A（3.19）：`role`、`use_case`。选 B 则保留这两列，`profiles` 为 13 列。
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
| `deleted_at` | `timestamptz` | 照搬（撤销就是软删除；管理员重置密码时一次软删除全部） |

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
| `expires_at` | `timestamptz NOT NULL` | 登录时设为登录时刻加 `auth.session_ttl`，之后不变（3.5） |
| `last_refreshed_at` | `timestamptz` | |
| `revoked_at` | `timestamptz` | |
| `revoke_reason` | `varchar(20)`，`CHECK (revoke_reason IN ('logout','password_changed','password_reset','email_changed','deactivated','reuse_detected'))` | `email_changed` 来自决策点 1 的命令；`deactivated` 只在决策点 3 选 A 或 B 时有写入方，选 C 就从列表中去掉 |
| `created_at`、`updated_at` | `timestamptz NOT NULL DEFAULT now()` | |

- **表级约束**：`CONSTRAINT auth_sessions_revoked_consistent_check CHECK ((revoked_at IS NULL) = (revoke_reason IS NULL))`，显式命名（3.13）。
- **索引**：`auth_sessions_user_id_idx`（撤销某个账户的全部会话），`auth_sessions_expires_at_idx`（清理任务）。
- **与差异清单的写法对照**：差异清单写的是"刷新令牌的哈希、令牌轮换链（用于重复使用检测）、UA、IP、过期和撤销时间"。"轮换链"在这里用代数表示：一次登录只有一行，不为每个刷新令牌建一行（3.5）。收尾时按实际的列改写差异清单。

### 4.6 行为差异（登记到差异清单第四节）
| 行为 | Plane | Nerve |
|---|---|---|
| 登录时邮箱不存在 | 返回 `USER_DOES_NOT_EXIST` | 与密码错误相同的 401，耗时也相同（3.9） |
| 密码规则 | 服务端在注册、修改密码、重置命令三处用 zxcvbn 评分 ≥ 3；组合规则只在界面上 | 服务端在三处执行组合规则（长度 8–128）、NCSC 前 10 万常见密码名单和"主干不能是邮箱前缀"（3.8） |
| 会话的期限 | Django 会话，从登录起固定 7 天（`SESSION_COOKIE_AGE`），请求不延长 | 访问令牌 15 分钟；会话从登录起 30 天，续期不延长；刷新令牌每次使用后换新，并检测重复使用（3.5） |
| 退出、修改密码、停用后旧凭证失效 | 其他会话在下一个请求时失效 | 相同，由每个请求的会话检查做到（3.5） |
| 管理员重置密码 | 只改密码（会话随之失效），PAT 不动 | 撤销全部会话和全部 PAT，输出撤销的数量（3.5、3.17） |
| `reset_password` 命令的邮箱 | 按原样匹配 | 按注册时的规则规范化 |
| 修改登录邮箱 | 用户在个人设置中向新邮箱索取验证码后修改（M1 已删除这个流程） | 只能由服务器管理员用 `nerve users set-email` 修改，撤销该账户的全部会话（决策点 1、3.17） |
| PAT 的管理 | 只能用 Cookie 会话管理 PAT | 任何凭证都能管理，包括 PAT 本身（总体设计 0.2 原则 2） |
| 无效的 PAT | 403（`AuthenticationFailed` 没有 `authenticate_header`） | 401 `unauthorized` |
| PAT 的 `last_used` | 每个请求都写 | 每分钟最多写一次 |
| PAT 的名称和过期时间 | 不校验：名称过长时变成 500，过期时间可以是过去 | 名称 1–255 个字符；过期时间必须在未来 |
| PAT 的编辑 | `PATCH` 可以改名称和说明，响应中带令牌原文 | 不提供（界面没有编辑入口）；令牌原文只在创建时返回一次 |
| 时区 | 只接受 `pytz.common_timezones` | 接受任何 IANA 名称（`Local` 除外）；时区列表接口给的仍是同一份常用列表（5.3） |
| 限流 | 认证接口合计每 IP 10/min，匿名 30/min，API Key 60/min；`/api/v1` 的响应带 `X-RateLimit-Remaining`、`X-RateLimit-Reset`（`plane/apps/api/plane/api/views/base.py:120-126`） | 3.10 的六个桶；超出时 429 带 `Retry-After`，不加 `X-RateLimit-*` |
| 密码哈希过载 | 无并发上限 | 最多 4 个同时计算，等待 2 秒仍拿不到名额时 503 `server_busy`（3.8） |
| 注册的前提 | 实例必须先由实例管理员完成设置 | 没有这一步 |
| 停用账户 | 见决策点 3 | 按裁定登记 |

---

## 5. 接口

### 5.1 操作
模块文件 `api/modules/identity.yaml`（新建）和 `api/modules/instance.yaml`（扩展）。路径都不带结尾 `/`（M1-P3 交接最后一条）。每个操作都写 `x-problem-codes`（3.11）；下表"主要错误"一栏就是它的内容，不再列出所有操作都可能返回的四个码。

| 方法与路径 | operationId | 认证 | 请求体 | 成功 | 主要错误 |
|---|---|---|---|---|---|
| `POST /api/v0/auth/register` | `register` | 公开 | `RegisterRequest {email, password}` | 201 `AuthTokens` | 422 `validation_failed`；409 `identity.email_taken`；403 `identity.signup_disabled`；503 `server_busy` |
| `POST /api/v0/auth/login` | `login` | 公开 | `LoginRequest {email, password}` | 200 `AuthTokens` | 422 `validation_failed`（缺字段）；401 `identity.invalid_credentials`；403 `identity.account_deactivated`；503 `server_busy` |
| `POST /api/v0/auth/refresh` | `refreshTokens` | 公开 | `RefreshRequest {refresh_token}` | 200 `AuthTokens` | 401 `identity.refresh_token_invalid` |
| `POST /api/v0/auth/logout` | `logout` | 公开 | `LogoutRequest {refresh_token}` | 204 | 无。令牌未知、已过期、已撤销、不是当前一代时也返回 204，不泄露它的状态（3.5） |
| `GET /api/v0/me` | `getMe` | bearer | — | 200 `User` | — |
| `PATCH /api/v0/me` | `updateMe` | bearer | `UserUpdate` | 200 `User` | 422 `validation_failed` |
| `POST /api/v0/me/change-password` | `changePassword` | bearer | `ChangePasswordRequest {current_password, new_password}` | 204 | 422 `validation_failed`（新密码不合规）；422 `identity.current_password_incorrect`；503 `server_busy` |
| `GET /api/v0/me/profile` | `getProfile` | bearer | — | 200 `Profile` | — |
| `PATCH /api/v0/me/profile` | `updateProfile` | bearer | `ProfileUpdate` | 200 `Profile` | 422 `validation_failed` |
| `GET /api/v0/me/api-tokens` | `listApiTokens` | bearer | 查询参数 `limit`、`cursor` | 200 `ApiTokenPage` | 400 `bad_request`（游标不合法） |
| `POST /api/v0/me/api-tokens` | `createApiToken` | bearer | `ApiTokenCreate {label?, description?, expired_at?}` | 201 `ApiTokenCreated` | 422 `validation_failed` |
| `DELETE /api/v0/api-tokens/{token_id}` | `revokeApiToken` | bearer | — | 204 | 404 `identity.api_token_not_found` |
| 决策点 3 带来的操作 | — | bearer | — | — | 见第 10 节 |
| `GET /api/v0/instance` | `getInstance` | 公开 | — | 200 `InstanceInfo` | — |
| `GET /api/v0/timezones` | `listTimezones` | 公开 | — | 200 `TimezoneList` | — |

- **动作接口**：按总体设计 3.2"少数业务动作用 `POST /资源/动作名`"，修改密码写成 `POST /me/change-password`，返回 204。
- **注册返回令牌，不返回新建的账户**：总体设计 3.1 说"POST 返回改完之后的完整资源"。注册是"建账户并登录"的动作，调用方接下来需要的是令牌；账户用 `GET /me` 读取。这与 3.1 的字面不同，收尾时写进 3.1（3.20）。
- **限流的 429**：所有操作都可能返回（顶层的 `x-problem-codes`）。登录、注册、修改密码另有自己的桶，见 3.10。

### 5.2 结构
所有对象都是 `additionalProperties: false`；时间是 `date-time`（UTC）；id 是 `uuid`。

| 结构 | 字段 |
|---|---|
| `AuthTokens` | `token_type`（`enum: [Bearer]`）、`access_token`、`access_token_expires_in`（整数，秒）、`refresh_token`、`refresh_token_expires_at`（会话的绝对期限），都必填。给秒数而不是时刻：客户端从收到响应的那一刻起算，不受本机时钟偏差影响（RFC 6749 5.1 的写法，7.1） |
| `User` | `id`、`email`、`first_name`、`last_name`、`display_name`、`user_timezone`、`avatar_url`、`cover_image_url`、`created_at`，都必填；`avatar_url`、`cover_image_url` 是 `type: [string, 'null']`，M5 之前恒为 `null`（3.2）。`is_active` 不返回：停用的账户根本认证不了 |
| `UserUpdate` | `first_name`、`last_name`、`display_name`、`user_timezone`，都可以省略，都不能为 `null`。没有 `email`：邮箱只能由管理员用命令行修改（决策点 1、3.17） |
| `Profile` | `theme`（五个值的枚举）、`language`（`en`、`zh-CN`）、`start_of_the_week`（`0`–`6` 的整数枚举）、`onboarding_step`（`OnboardingSteps`，四个布尔值都必填）、`is_onboarded`、`is_tour_completed`、`last_workspace_id`（`uuid` 或 `null`）、`updated_at`。不返回 `id` 和 `user`：资料通过 `/me/profile` 访问，这两个 id 没有用处（前端的替代见 7.5） |
| `ProfileUpdate` | 上面除 `updated_at` 外的字段，都可以省略；`last_workspace_id` 可以传 `null` 清空。`onboarding_step` 是 `OnboardingStepsUpdate`：四个布尔值都可以省略，服务端把传入的键合并进现有的值（3.11） |
| `ApiToken` | `id`、`label`、`description`、`expired_at`（可为 `null`）、`last_used`（可为 `null`）、`created_at` |
| `ApiTokenCreated` | `ApiToken` 的全部字段加上 `token`（令牌原文，只出现在这里）。单独写出所有字段，不用 `allOf`：几个都带 `additionalProperties: false` 的 schema 用 `allOf` 组合时会互相拒绝对方的字段 |
| `ApiTokenPage` | `data: ApiToken[]`、`next_cursor`（`NextCursor`），按 `created_at` 倒序 |
| `InstanceInfo` | M0 的四个字段，加上 `signup_enabled`、`workspace_creation_enabled`、`file_size_limit`（5.3） |
| `Timezone` / `TimezoneList` | `label`、`value`（IANA 名称）、`utc_offset`、`gmt_offset`；列表是 `{data: Timezone[]}`，固定约 100 项，不分页 |

- 主题的五个值与 `web/packages/constants/src/themes.ts:18-69` 的 `THEME_OPTIONS` 相同，没有 `custom`，也没有调色板字段（M1-P2 交接）。
- 决策点 4 选 B 时，`Profile` 和 `ProfileUpdate` 加上 `role`、`use_case`。

### 5.3 实例配置与时区
| 接口字段 | 配置项 | 默认值 | 谁来执行 | 前端的旧名字 |
|---|---|---|---|---|
| `signup_enabled` | `auth.signup_enabled` | 按决策点 2 的裁定 | M2 的注册用例 | `enable_signup` |
| `workspace_creation_enabled` | `workspace.creation_enabled` | `true` | M3 的创建工作区（交接） | `is_workspace_creation_disabled`（取反） |
| `file_size_limit` | `files.size_limit`（字节） | `5242880` | M5 的上传（交接） | `file_size_limit` |

- **命名**：两个开关都用正面的 `…_enabled`，与 M0 设计 3.2 已写下的 `signup_enabled` 一致。前端读 `is_workspace_creation_disabled` 的 4 个文件改为读取反的值（`create-workspace/page.tsx:40`、新手引导的 `create.tsx:54`、`power-k/config/creation/command.ts:59`、`workspace-menu-root.tsx:45`）。
- **`is_self_managed`**：两种裁定下都不定义（3.19）。
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
| `identity.email_taken` | 409 | 注册时邮箱已被使用。`nerve users set-email` 的用例返回同一个错误，命令行把它显示为一句说明 |
| `identity.refresh_token_invalid` | 401 | 刷新令牌未知、已过期、已撤销、被重复使用、被伪造，都是这一个码 |
| `identity.current_password_incorrect` | 422 | 修改密码时当前密码错误；`errors[].field = current_password` |
| `identity.api_token_not_found` | 404 | 撤销不存在、已撤销或属于别人的 PAT（看不到的资源一律 404，总体设计 3.5） |

---

## 6. 后端结构

### 6.1 新增和修改的包
| 包 | 内容 | 依赖 |
|---|---|---|
| `internal/shared` | `Actor`、`Error`（含 `FieldError`）、`TxManager`、分页游标（3.3） | 只有标准库 |
| `platform/postgres` | `TxManager` 的实现，按结构满足 `shared.TxManager`：事务放进 `context`，仓储用 `postgres.DB(ctx, pool)` 取当前事务或连接池；连接池按 UTC 扫描 `timestamptz` | pgx、goose |
| `platform/clock` | `System` 时钟，按结构满足各模块声明的 `Clock` | 标准库 |
| `platform/ratelimit` | 按键的令牌桶，闲置的键定期清掉 | `golang.org/x/time/rate` |
| `platform/jobs` | River 客户端的创建、启动、停止；服务用的客户端和命令行用的只投递客户端 | River、pgx |
| `platform/httpserver` | `Router`（记下注册的模式）；3.6 的 `API` 值、认证中间件（默认拒绝）、限流中间件、请求元信息、请求期限、请求体上限；3.11 的 `ProblemError` 映射和新平台码；导出 `RequestID`（M0-P2 交接 3）；固定链上加安全响应头（8.3） | 标准库 |
| `platform/webui` | CSP：启动时算出 `index.html` 内联脚本的哈希（8.3） | 标准库 |
| `platform/config` | 6.5 的新配置项、校验、`LogValue` 打码（M0-P2 交接 4）；环境变量给布尔配置项传空值时报错，不再悄悄当作 `false`（M0-P2 交接 8） | koanf |
| `modules/identity` | 6.2 | `internal/shared`；适配器另外依赖平台、pgx、x/crypto、golang-jwt、River |
| `modules/instance` | 三个配置字段、时区列表 | 同 M0 |
| `bootstrap` | 接线；汇总各模块的公开操作；`run` 让 HTTP 和 River 一起运行，按 3.15 的顺序停机；`users` 命令的最小组合（3.17）；在创建 logger 之后出现的致命错误写一条结构化日志（M0-P2 交接 8）；编译期断言 `postgres` 的事务管理器满足 `shared.TxManager` | 全部 |
| `cmd/nerve` | `nerve users reset-password` 等命令的参数解析 | 同 M0 |
| `internal/archtest` | 规则 4 加上 `internal/shared`；规则 6 推广到 `adapter/*/gen`；`TestSQLCSchemaScope`（3.14） | — |

- **健康检查的访问日志**：`/healthz`、`/readyz` 的访问日志降为 DEBUG 级别（M0-P2 交接 8：生产环境中每次探测一条 INFO 太多）。
- **只读的迁移角色**：生产环境用单独的数据库角色执行迁移时，运行服务的角色需要 `goose_db_version` 的 SELECT 权限（M0-P2 交接 7），写进 README 的部署说明。

### 6.2 `identity` 模块的结构
```
modules/identity/
  domain/
    user.go                 User；邮箱规范化；名字、显示名、时区的规则
    profile.go              Profile；Theme、Language、WeekStart、OnboardingSteps（含部分更新的合并）
    password.go             密码规则：组合规则、常见密码名单、主干（3.8）
    common_passwords.txt    生成的名单，go:embed（3.8）
    session.go              Session；刷新令牌的编解码；轮换与重复使用的判定（3.5）
    api_token.go            APIToken；PAT 的格式；名称和过期时间的规则
    errors.go               本模块的错误码（5.4）
  app/
    ports.go                端口（6.3），包括 Clock
    register.go  login.go  refresh.go  logout.go  authenticate.go
    get_me.go  update_me.go  change_password.go  get_profile.go  update_profile.go
    create_api_token.go  list_api_tokens.go  revoke_api_token.go
    reset_password.go  set_email.go  cleanup_sessions.go  （以及决策点 2、3 的用例）
  adapter/
    postgres/               仓储；queries/*.sql；gen/（sqlc 生成）
    http/                   handler，按资源分文件；操作级限流；公开操作的清单；gen/（oapi-codegen 生成）
    argon2/                 PasswordHasher（并发上限和等待上限）
    jwt/                    AccessTokens（Ed25519）
    river/                  清理会话的 worker
    authn/                  Authenticator 的实现：调用认证用例，把 Actor 放进 context，返回限流键
  module.go                 New(Deps)；Register(router, api)；PublicOperations()；Authenticator()；Jobs()；Admin()
```
- 一个用例一个文件（总体设计 6.1），每个文件预计 40–120 行。
- `domain` 的 Go 文件都在 400 行以内。

### 6.3 端口
| 端口 | 声明在 | 实现 | 测试 |
|---|---|---|---|
| `Users`、`Profiles`、`Sessions`、`APITokens`（仓储，按用例需要拆成小接口） | `identity/app` | `identity/adapter/postgres`（sqlc） | 集成测试，每个测试一个独立的库（`pgtest`） |
| `PasswordHasher`（`Hash`、`Verify` 返回"是否需要重新哈希"） | `identity/app` | `identity/adapter/argon2` | 往返、错误密码、参数变化后要求重新哈希、并发上限、等待超时返回 `server_busy` |
| `AccessTokens`（`Issue`、`Verify`） | `identity/app` | `identity/adapter/jwt` | 往返；过期；算法不是 `EdDSA`；换了密钥；缺 `sub`、`sid`；篡改载荷 |
| `SignupPolicy`（是否允许注册） | `identity/app` | `bootstrap` 从配置传入的值 | 用例测试。M3 扩展它的实现（持有邀请的人仍可注册，13.2） |
| `Clock` | `identity/app` | `platform/clock` 的 `System`；测试用固定时钟 | 两种实现都跑同一个小的契约测试（总体设计 6.1 的里氏替换） |
| `TxManager` | `internal/shared` | `platform/postgres`（按结构满足） | 提交、回滚、嵌套调用时复用同一个事务 |
| `Authenticator` | `platform/httpserver` | `identity/adapter/authn` | 中间件的单元测试用假实现；整程序测试用真实现 |
| `Limiter` | `platform/httpserver` | `platform/ratelimit` | 同上 |
| `ProblemError` | `platform/httpserver` | `shared.Error`（按结构满足） | `bootstrap` 的测试逐个 `Kind` 核对状态和码 |

没有"随机数"端口：理由见 3.4。

### 6.4 一个带令牌的请求
```
请求 → 请求 ID → 异常恢复 → 访问日志 → 安全响应头          （固定链，httpserver.NewServer）
     → 路由匹配：Router 找到模式，填进 r.Pattern
     → 生成的代码：绑定路径参数和查询参数（格式错误 → 400）
     → 请求元信息（客户端 IP、UA）→ 请求期限 → 请求体上限        （按路由，3.6）
     → 认证：公开操作直接放行；其他操作要求 JWT（验签 + 会话和账户的主键查询）或 PAT（按哈希查询），
       通过后 Actor 进入 context
     → 限流：有凭证按凭证计数，没有按 IP 计数
     → 生成的代码：解码 JSON 请求体（格式错误 → 400）
     → handler：RequireActor；操作级限流（登录、注册、修改密码）；把生成的类型转成用例的入参
     → 用例：TxManager.WithinTx { 领域规则和校验 → 写数据 }
     → 响应；或者 error → APIErrors → problem+json
```
- 第一稿的图把参数绑定和 JSON 解码画在认证、限流之后，与生成代码的实际顺序不符（评审 M1），已按 spike 的生成代码改正。
- **事务**：以下各自在一个事务里完成：
  - 注册：账户、资料、会话；
  - 修改密码：改哈希、撤销会话；
  - 续期时发现重复使用：撤销；
  - 管理员重置密码：改哈希、撤销会话、撤销 PAT；
  - 管理员修改邮箱：改邮箱、撤销会话；
  - 决策点 3 的停用：账户、会话、资料，以及 M3 加入的成员关系（第 10 节）；
  - 登录时需要重新哈希：写回哈希、插入会话。
- **单条语句的写入**不开事务：不需要重新哈希的登录只插入一行会话；续期的正常路径是一条条件 `UPDATE`。

### 6.5 配置
`server/configs/config.yaml` 新增（test、prod 的覆盖值写在注释中）：
```yaml
server:
  trusted_proxies: []          # CIDR；为空时客户端 IP 取连接的对端地址（3.10）
  max_body_bytes: 1048576      # JSON 接口的请求体上限
  request_timeout: 15s         # 每个接口请求的期限（3.6）
  addr_file: ""                # 非空时，监听成功后把实际地址写进这个文件（端到端测试用 :0 监听，9.5）
auth:
  signup_enabled: true         # 默认值按决策点 2 的裁定；选 C 时 prod 覆盖为 false
  access_token_ttl: 15m
  session_ttl: 720h            # 30 天，从登录起算，续期不延长（3.5）
  session_cleanup_interval: 1h # test: 2s
  jwt:
    private_key_file: ""       # prod 必填；dev/test 为空时启动时生成临时密钥（3.7）
  password:
    argon2_memory_kib: 19456   # test: 64
    argon2_iterations: 2       # test: 1
    argon2_parallelism: 1
    max_concurrent_hashes: 4
    max_wait: 2s               # 拿不到名额时最多等这么久，然后 503 server_busy（3.8）
ratelimit:                     # test 全部调高（3.10）
  anonymous_per_minute: 120
  authenticated_per_minute: 1200
  authenticated_burst: 200
  login_ip_per_minute: 30
  login_ip_email_per_minute: 10
  register_ip_per_minute: 10
  password_user_per_minute: 5
jobs:
  shutdown_timeout: 10s
workspace:
  creation_enabled: true
files:
  size_limit: 5242880
```
- **启动校验**：
  - 时长为正；
  - `session_ttl` 大于 `access_token_ttl`；
  - CIDR 合法；
  - argon2 的参数在 x/crypto 允许的范围内；
  - `env = prod` 时 `auth.jwt.private_key_file` 必须提供，而且文件能读出一把 Ed25519 私钥。
- **`LogValue`**：列出新的配置项；`private_key_file` 只记路径。

### 6.6 新的依赖（版本在 P1、P3 写死）
| 依赖 | 版本 | 放在 | 用途 |
|---|---|---|---|
| `github.com/sqlc-dev/sqlc` | v1.31.1 | `server/tools/go.mod` | 代码生成（P1） |
| `github.com/golang-jwt/jwt/v5` | v5.3.1 | `server/go.mod` | 访问令牌（P1） |
| `golang.org/x/crypto` | v0.57.0 | `server/go.mod` | argon2id（P1） |
| `github.com/oapi-codegen/runtime`、`github.com/oapi-codegen/nullable` | v1.7.0、v1.2.0 | `server/go.mod` | 生成代码的依赖（3.12，P1） |
| `golang.org/x/time` | 当时的最新版 | `server/go.mod` | 限流（P2） |
| `github.com/riverqueue/river`、`riverdriver/riverpgxv5` | v0.47.0 | `server/go.mod` | 后台任务（P3） |
| River CLI（`github.com/riverqueue/river/cmd/river`） | v0.47.0 | 不进仓库 | 只在写迁移时导出 SQL；命令写进迁移文件的注释 |
| `golang.org/x/term` | v0.46.0（x/crypto v0.57.0 所需的版本） | `server/go.mod` | 命令行不回显地输入密码（P3） |

- 常见密码名单是嵌入的数据文件，不是依赖（3.8）。
- 这些依赖都不在架构测试的禁止链接名单里（`github.com/google/uuid`、kin-openapi、testcontainers、docker）。它们的传递依赖同样不能碰到这个名单，由 `TestNerveBinaryLinksNoBannedModule` 在加入依赖的那个 Phase 核对。
- River 让程序大约增加 0.96 MB（spike 实测，未去符号表时）。

---

## 7. 前端

### 7.1 令牌管理器
落实总体设计 4.3 和 7.3。

- **位置**：
  - `web/apps/web/core/lib/auth/token-manager.ts`：`TokenManager` 类。
  - `web/apps/web/core/lib/auth/refresh-lock.ts`：跨标签页的协调（`navigator.locks` 或 localStorage 租约）。
  - `web/apps/web/core/lib/auth/api-client.ts`：web 使用的生成客户端实例，挂上认证中间件。
  - 端到端测试直接用 `@nerve/api-client`，自己带 `Authorization`，不经过令牌管理器。
- **存放**：访问令牌和它的过期时刻只在内存里；刷新令牌在 localStorage 的 `nerve.refresh_token`。
- **取访问令牌**：离过期还有 30 秒以上就直接用，否则先续期。
  - 过期时刻 = 本地收到响应的时刻 + `access_token_expires_in` 秒（5.2）。前端不解析 JWT，也不拿本机时钟和服务端的时刻比较。
  - 理由：本机时钟快了十几分钟时（硬件时钟按本地时间设置的双系统电脑很常见），和服务端的时刻比较会让每个请求都先续期一次，很快碰到限流（评审 I14）。
- **启动**：localStorage 里没有刷新令牌时，直接判定"未登录"，**不请求 `/me`**。所以登录页没有失败的接口请求（S2）。有刷新令牌时先续期拿到访问令牌，再取 `/me`。
- **续期**：
  - 同一个标签页内，同一时刻只有一次续期，其他调用共用同一个 Promise。
  - 跨标签页串行（见下一条）。拿到锁之后重新读 localStorage：另一个标签页可能刚换过令牌。
  - 续期请求用一个不挂认证中间件的客户端，避免递归。
  - 续期的结果：

    | 结果 | 处理 |
    |---|---|
    | 200 | 保存新的一对令牌 |
    | 401 | 清掉刷新令牌，结束会话。只有这一种结果表示"需要重新登录" |
    | 429、5xx、网络错误 | 保留刷新令牌，退避后再试：有 `Retry-After` 就按它，没有就 1、2、4……秒，最多 30 秒。这一次调用把错误交给调用方，页面按已有的错误提示显示 |
- **跨标签页的协调**（评审 I4）：
  - `navigator.locks` 和 `crypto.randomUUID` 都只在安全上下文（HTTPS 或 localhost）中存在。总体设计 2.1 允许直接用 HTTP 部署（"需要 HTTPS 时加 Caddy"），例如在局域网里打开 `http://192.168.1.20:8080`，那里没有它们。
  - 第一稿只用 `navigator.locks`：在这种部署下，令牌管理器要么报错，要么几个标签页同时续期，触发重复使用检测（3.5），用户被踢下线。端到端测试跑在 `127.0.0.1` 上，那是安全上下文，发现不了。
  - 有 `navigator.locks.request` 时：`navigator.locks.request("nerve.auth.refresh", …)`。
  - 没有时，用 localStorage 租约：
    - 键 `nerve.auth.refresh_lease`，值 `{"owner": <标签页 id>, "expires": <毫秒时间戳>}`，租期 10 秒。续期请求自己的超时是 8 秒，短于租期。
    - 标签页 id 用 `crypto.getRandomValues` 生成，它在非安全上下文中也能用。
    - 获取：没有租约、租约已过期、或者租约是自己的，就写入自己的租约；等 100 毫秒再读一次，仍是自己的才算拿到。
    - 等待：监听租约键的 `storage` 事件，同时每 200 毫秒轮询一次；租约被删除或过期后重新获取。
    - 用完：只删除自己的租约。
    - 局限：localStorage 没有原子的"比较并写入"。"写入后再读一次"只是让两个标签页同时拿到租约变得极不可能。万一发生，代价是一次重复使用检测，用户重新登录，与续期响应丢失的代价相同（3.5）。
    - 选租约而不选 BroadcastChannel 选主：租约只用刷新令牌所在的同一个 localStorage，不需要选主、心跳和换主。
  - README 写明：HTTP 部署可以用，多标签页靠租约协调；放在公网上应当用 HTTPS。
- **401**（openapi-fetch 中间件的 `onResponse`）：
  - 请求带了访问令牌却得到 401：续期一次，用新令牌重发一次。重发需要请求的副本，在 `onRequest` 中保存（P4 先用原型验证 openapi-fetch 0.17.0 的中间件能这样做）。
  - 续期失败或重发后仍是 401：结束会话。
  - 没有刷新令牌时，401 原样交给调用方。
- **结束会话**：
  - 清掉内存和 localStorage，调用应用注册的回调：`rootStore.resetOnSignOut()`。
  - 当前账户变为空，`AuthenticationWrapper` 渲染 `<Navigate to="/?next_path=…" replace />`（3.18）。
  - 跳转只在包装层这一处发生；现在的 `window.location.replace` 删除。
- **退出**（评审 M7）：
  - 在续期用的同一把锁（或租约）下，读出当前的刷新令牌，用不挂认证中间件的客户端调用 `POST /auth/logout`。
  - 不能用挂了认证中间件的客户端：访问令牌恰好过期时，中间件会先续期、再重发，重发的请求体里还是续期前的刷新令牌，服务端看到的是上一代，退出什么也没做（3.5）。
  - 尽力而为：失败也清本地。然后结束会话，回到登录页。
- **其他标签页**：监听 `storage` 事件，刷新令牌被别的标签页删除时，本标签页也结束会话（A6）。
- **可测试**：`storage`、`locks`、`fetch`、当前时刻都从构造函数传入，vitest 用假实现（9.4）。
- **不做**：定时主动续期（按需续期已经足够）；把访问令牌放进 sessionStorage。

### 7.2 传输层的清理
- **web 的 axios 基类**（`web/apps/web/core/services/api.service.ts`）：
  - 删掉 `withCredentials: true` 和 401 拦截（M1-P4 交接：恒为真的 `currentPath` 判断不照搬）。
  - 此后它只供 M3–M8 还没对接的领域使用。那些调用指向 Plane 的旧接口，在 Nerve 中得到 404 problem，由各领域对接时逐个删除。
  - 它不带访问令牌：对接新接口的代码都走生成的客户端。
- **`@nerve/services` 的 axios 基类**（`web/packages/services/src/api.service.ts:23`，同样 `withCredentials: true`）：只被 PAT 的 `APITokenService` 使用，随 PAT store 的重写在 P5 删除（7.5）。
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
  - `problem.code` 映射到 `t()` 的文案；`validation_failed` 的字段错误显示在对应字段下方（`weak_password` 显示规则，`common_password` 显示"密码太常见"）；429 显示"尝试次数过多，请稍后再试"；503 `server_busy` 显示"服务器繁忙，请稍后再试"。
  - `auth-root.tsx` 不再读地址里的 `error_code`、`email`。
  - 保留 `next_path`，以及工作区邀请的 `invitation_id`、`slug`：它们属于 M3 的"加入某个工作区"标题。
- **`authentication.helper.tsx`**：
  - 删除 `EAuthenticationErrorCodes`（18 个 Plane 数字码）和它们的文案映射。
  - 保留 `EPageTypes`、`EAuthModes`。
  - 新写一张"错误码 → 文案键"的表。vitest 从 `api/dist/openapi.yaml` 读出全部 `x-problem-codes`，核对表的键与之相等（3.11）；字段码另有一张表，核对它覆盖 `FieldError.code` 的全部取值。
- **注册链接**只在 `signup_enabled` 为真时显示（`web/apps/web/core/components/auth-screens/header.tsx:39-47`）。
- **密码强度提示**照旧（`getPasswordStrength`），加上 128 个字符的上限，与服务端一致。常见密码只在服务端判定，名单不下发到前端。

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
| `IUserLite`、`IUserSettings` | 成员信息（M3）；`/users/me/settings/` 中的工作区数据 | 不动，留给 M3。`IUserLite.avatar_url` 按 3.2 规则 2 由 M3 改为可为 `null` |
| `EStartOfTheWeek` | 日历等 9 个文件使用的数字枚举 | 保留：纯界面常量，取值与生成的 0–6 一致 |
| `core/services/auth.service.ts` | CSRF 和表单 | 重写：`register`、`login`。`refresh`、`logout` 由令牌管理器调用 |
| `core/services/user.service.ts` | Plane 的 `/api/users/me/…` | M2 的方法改为生成客户端的薄封装。属于其他领域的 4 个方法（`getUserProfileIssues`、`leaveWorkspace`、`joinProject`、`leaveProject`）原样留下，由 M3、M4 对接时迁走 |
| `core/services/instance.service.ts`、`timezone.service.ts` | Plane 的地址 | 生成的客户端 |
| `@nerve/services` 的 `api.service.ts`、`developer/` | `APITokenService`（Plane 的 `/api/users/api-tokens/`） | 删除（P5）。包里只剩地址工具和文件工具，整个包由 M5 删除（总体设计 7.1） |
| `store/user/index.ts`（`UserStore`） | `IUser`；登录时先取账户，再并行取资料、设置、工作区三样 | `data: User`；没有刷新令牌时不取数（7.1）；`fetchCurrentUser` 只取 `/me` 和 `/me/profile`；`signIn`、`signUp`、`signOut` 通过令牌管理器；删除死成员（`reset`、`isAuthenticated`、`error` 等） |
| `store/user/profile.store.ts` | `TUserProfile`；`updateUserProfile` 吞掉错误（`:138-148`），调用方的错误提示从不出现 | `data: Profile`；失败时抛出。`finishUserOnboarding` 合并为一次 `PATCH /me/profile`（部分的 `onboarding_step` + `is_onboarded` + `last_workspace_id`）；`updateTourCompleted`、`updateUserTheme` 同样用这一个接口 |
| `store/user/settings.store.ts` | `/users/me/settings/` 加上侧边栏状态 | 从登录流程中拿掉（3.1）；删除死成员（`isScrolled`、`toggleIsScrolled` 等）；工作区数据留给 M3 |
| `store/instance.store.ts` | `IInstanceConfig` | `config: InstanceInfo` |
| `store/workspace/api-token.store.ts` | 没有任何代码读它（`store/workspace/index.ts:93` 只是创建）；页面用 SWR 直接调用 service | 移到 `store/user/api-token.store.ts`，用生成的客户端重写，翻页直到取完；页面和弹窗改为经 store 读写（总体设计 7.2：组件不直接调用接口） |

**删除 `IUser` 和 `Profile.id` 牵连的地方**（评审 M11，逐个核对过）：

| 位置 | 现在 | 改为 |
|---|---|---|
| `web/packages/types/src/workspace.ts:20` | `owner: IUser` | `IUserLite`（M3 对接工作区时按接口再定） |
| `web/packages/types/src/project/projects.ts:51` | `default_assignee?: IUser \| string \| null` | `IUserLite \| string \| null` |
| `web/packages/types/src/search.ts:17-19` | `IUser["avatar_url"]`、`["display_name"]`、`["id"]` | 同名的 `IUserLite` 字段 |
| `home/user-greetings.tsx`、`workspace/settings/member-columns.tsx`、`project/settings/member-columns.tsx`、`issues/issue-detail/reactions/issue.tsx` | `currentUser: IUser` 这类 prop | `User` |
| `web/apps/web/core/lib/wrappers/store-wrapper.tsx:53` | 用 `userProfile?.id` 判断"换了账户"，据此重新同步主题 | 改用 `currentUser?.id` |

**跟随类型修改的其他使用方**：
- `is_workspace_creation_disabled`：4 个文件（5.3）。
- `profile.theme.theme` → `profile.theme`：`store-wrapper.tsx:52-72`、`theme-switcher.tsx`、`power-k/config/preferences-commands.ts`。
- PAT 列表项的"有效 / 已过期"：改为由 `expired_at` 计算（`web/apps/web/core/components/api-token/token-list-item.tsx`）。
- 当前账户的头像、封面：显示代码不改；类型从 `string` 变为 `string | null`，现有写法（`?? ""`、先判断是否为空）已经兼容，类型检查会列出例外。

### 7.6 删除的 Plane 代码
- CSRF：4 处，加上它的类型（7.2）。
- Plane 的 18 个认证错误码和它们的文案（7.3）。
- 新手引导的"角色""用途"两步（338 行）和 `is_self_managed`（按决策点 4 的裁定，3.19）。
- `@nerve/services` 的 `api.service.ts`、`developer/`（共 178 行）。
- 上传控件（3.2）：general 页的头像上传弹窗（`UserImageUploadModal`）和封面选择器（`ImagePickerPopover`），新手引导资料步骤的头像上传。显示头像和封面的代码保留。
- general 页写 `role` 默认值的逻辑（3.19）。
- 死成员和死 prop（7.8）。
- 以上各项的文案键：中英文一起删，`sync-check` 和"整键 + 模板前缀"的核对沿用 M1 的做法。

### 7.7 个人设置的四个标签页
- **general**：
  - 名、姓、显示名可以修改；邮箱只读，只能由管理员用命令行修改（决策点 1）；
  - 头像和封面照常显示（M5 之前是首字母头像和默认封面），上传控件按 3.2 删除；
  - "停用账户"按决策点 3 处理。
- **preferences**：
  - 主题：`PATCH /me/profile {theme}`，保留切换后刷新页面的现有行为；
  - 语言：`PATCH /me/profile {language}`；
  - 时区：`PATCH /me {user_timezone}`，下拉框的数据来自 `GET /api/v0/timezones`；
  - 每周第一天：`PATCH /me/profile {start_of_the_week}`。
- **security**：
  - 修改密码。错误用生成的 `Problem` 类型处理，去掉 `Error & { error_code?: string }` 的断言和 `toString()` 转换（M1-P4 交接"由 M2 决定"第 2 项）。
  - `current_password_incorrect` 显示在当前密码字段下；`validation_failed` 的密码错误显示在新密码字段下。
  - 表单下方列出账户的 PAT（复用 api-tokens 页的列表组件，可以就地撤销），并说明"修改密码不会撤销这些令牌"（3.5）。
- **api-tokens**：
  - 创建（名称、说明、有效期：1 周、1 个月、3 个月、1 年、自定义日期、永不过期）；
  - 创建后令牌只显示一次，同时下载 CSV（照搬现有行为）；
  - 列表、撤销。
- **主题下拉框**（M1-closeout 交接）：界面语言为 zh-CN 时，主题下拉框画在页面左上角。
  - P5 查明 `web/packages/ui/src/dropdowns/custom-select.tsx` 中 react-popper 定位加 `createPortal` 的根因，在组件里修复，不在调用处打补丁。
  - 浏览器核对中英文各一次。

### 7.8 约束 M2 的 M1 收尾规则
- **死成员和死 prop**：`domains.mjs --rows M2` 的 66 行全部处理。
  - 改到的文件随改随删。
  - 没改到的（例如 `store/issue/profile/*` 的 7 行、`auth-layout/*-wrapper.tsx` 的 2 个 `isLoading` prop）数量很少，在 P5 集中删掉，一次关闭交接。
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
| `csrf`（不区分大小写）、`X-CSRFTOKEN`、`csrfmiddlewaretoken`、`get-csrf-token` | P4 |
| Plane 认证地址 `/auth/sign-`、`/auth/change-password` | P4 |
| Plane 的认证错误码参数 `error_code` | P4 |
| `is_self_managed` | P4 |
| `withCredentials:\s*true` | P5：`web/packages/services/src/api.service.ts:23` 要到 P5 才随 PAT 的重写删除，放在 P4 会让守卫在 P4 失败（评审 M5） |
| Plane 的用户、实例、时区地址：`/api/users/me/profile`、`/api/users/me/onboard`、`/api/users/me/tour-completed`、`/api/users/api-tokens`、`/api/instances/`、`/api/timezones/` | P4、P5 |

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
| 改动的文件 | 85–100 个 | 上面这些文件组中约 60 个；跟随类型修改的使用方约 20 个（7.5）；新文件约 10 个（令牌管理器、跨标签页的协调、客户端实例、错误文案表和它们的测试） |
| 删除 | 约 1,000 行 | 7.6 的各项。头像和封面的显示代码不再删除 |
| 重写 | 约 3,000 行 | stores 约 700 行；services 约 250 行；两个包装层约 160 行；认证表单约 450 行；个人设置和 PAT 组件约 900 行；新手引导约 450 行 |
| 新增 | 约 850 行 | 令牌管理器约 250 行，跨标签页的协调约 100 行，它们的测试约 350 行，其余约 150 行 |
| 净变化 | 减少约 150 行 | |

**后端**：
- 生产代码约 4,600 行（不含生成的代码和名单文件），测试约 5,600 行；
- 手写的 SQL 约 300 行，加上 River 的迁移（318 + 189 行）；
- `identity.yaml` 约 850 行；
- 常见密码名单 33,919 行（生成的数据，277 KB）。

**端到端**：fixture 约 500 行，16 个故事约 2,000 行。

---

## 8. 安全

### 8.1 密码与哈希
见 3.8：组合规则加常见密码名单；argon2id 的并发上限和等待上限。

### 8.2 账户枚举
- 登录不暴露账户是否存在，时间也一致（3.9）。
- 注册必然暴露：没有邮件通道，就无法做到"已注册和未注册的回答相同"。用每个 IP 每分钟 10 次的注册限流压低探测速度。
- 这是 v0 已知的局限，写进 README 的安全说明。

### 8.3 安全响应头与 CSP
- **两层，分开放**（评审 M4）：
  - **固定链上的安全响应头**（所有响应，P2）：`X-Content-Type-Options: nosniff`、`Referrer-Policy: same-origin`、`X-Frame-Options: DENY`。它们对接口的响应同样有意义，所以放在固定链上。固定链由三个中间件变为四个，M0 设计 3.3、总体设计 6.4 的说明随之更新。
  - **CSP 只加在 HTML 响应上，由 `webui` 负责**（P4）：它只对页面有意义，而且要用 `index.html` 里内联脚本的哈希，只有 `webui` 知道这些脚本。
  - M0-P5 交接说"和 CSP 放在同一层"，本意是两者都由服务端在 M2 加入。按上面的理由分在两层，是有意的安排。
- **CSP 的内容**：
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
- **外部来源的清点**（评审 M12）：在保留的前端代码中查找绝对地址（排除测试和示例），并查看运行时使用的第三方库的默认地址，得到下表。M2 的页面（登录、注册、新手引导的资料步骤、个人设置）一个都不用，P4 的浏览器核对确认这些页面没有 CSP 违规。

  | 来源 | 用在哪里 | 被 CSP 挡住的类别 | 交给 |
  |---|---|---|---|
  | `cdn.jsdelivr.net/npm/emojibase-data` | `frimousse` 表情选择器在运行时下载表情数据（`@nerve/propel` 的 `emoji-icon-picker`，`frimousse` 的默认地址）。项目图标、视图、评论和工作项的表情回应都用它 | `connect-src` | M3（项目图标最先用到）；M4 核对表情回应 |
  | `cdn.jsdelivr.net/npm/emoji-datasource-apple` | 编辑器 callout 的默认表情图（`web/packages/editor/src/extensions/callout/utils.ts:19`） | `img-src` | M4 |
  | 对象存储（S3）的来源 | 签名地址的图片和附件 | `img-src`、`connect-src` | M5 |
  | 对外接口文档页 | 常见的文档页组件从 CDN 加载脚本 | `script-src` | M8 |

  - 处理原则写进交接：优先把资源打进程序、从本站提供（例如把 `emojibase-data` 随前端一起构建，设置 `frimousse` 的 `emojibaseUrl`），而不是在 CSP 中放开外部来源。自托管的 Nerve 应当在没有外网时也能用，也不应把使用情况泄露给第三方。只有对象存储这种本来就在外部的来源才加进 CSP。
- **HSTS** 由前面的反向代理设置：nerve 不知道自己是否在 TLS 之后。
- **核对**：S2 统计 `securitypolicyviolation` 事件为 0；P4 的浏览器核对再覆盖新手引导页和设置页。

### 8.4 日志
- **永远不记**：
  - 密码；
  - 任何令牌（访问、刷新、PAT）和它们的哈希；
  - 会话 id 的原文：知道会话 id 就能拼出"更旧的代数"让会话被撤销（3.5），看日志的人不该有这个能力。日志里记 `session_ref`，即会话 id 的 SHA-256 的前 8 个字节（十六进制），足够把同一个会话的几条日志串起来（评审 M14）；
  - `Authorization` 请求头；
  - 认证接口和创建 PAT 的请求体、响应体；
  - JWT 私钥的内容；
  - 不存在的账户的邮箱。
- **INFO**：
  - 注册：`user_id`；
  - 登录成功：`user_id`、`session_ref`、IP；
  - 登录失败：原因（`invalid_credentials` 或 `deactivated`）、IP，已知账户时加 `user_id`；
  - 退出：`session_ref`；
  - 修改密码：`user_id`；重置密码：`user_id`、撤销的会话数和 PAT 数；修改邮箱：`user_id`、撤销的会话数（不记新旧邮箱）；
  - 创建、撤销 PAT：`user_id`、`token_id`；
  - 触发限流：桶的名字、IP；
  - 密码哈希名额等待超时：当时排队的数量。
- **WARN**：发现刷新令牌被重复使用：`user_id`、`session_ref`、IP。
- **访问日志**不变：方法、路径、状态码、耗时、请求 ID。令牌从不出现在地址里，路径中没有机密。
- **测试**：登录、注册、续期、创建 PAT 的测试捕获日志输出，断言其中不含密码、令牌和会话 id 的原文。

### 8.5 刷新令牌放在 localStorage 的风险与升级路径
- **风险**：任何 XSS 都能读出 localStorage 中的刷新令牌，得到一个会话，直到发现重复使用、用户退出，或者会话到达 30 天的绝对期限。
- **现有的防线**：
  - 严格的 CSP（8.3）；
  - 服务端清洗 HTML（M4，bluemonday）；
  - 刷新令牌每次使用后换新，并检测重复使用（3.5）；
  - 会话有绝对期限，续期不延长（3.5）；
  - 访问令牌只在内存里，15 分钟过期；
  - 退出、修改密码在下一个请求就生效（3.5）。
- **升级路径**（总体设计 4.3，M2 只写下来，不做）：
  - 注册、登录、续期的响应把刷新令牌写进 `HttpOnly; Secure; SameSite=Strict; Path=/api/v0/auth` 的 Cookie；
  - 续期和退出在请求体里没有令牌时，从这个 Cookie 读；
  - 其余接口仍然只认 `Bearer`，不受影响。
  - 前端的令牌管理器不再碰 localStorage 中的令牌，续期请求带上 `credentials: "same-origin"`；多标签页共享同一个 Cookie，跨标签页的协调（7.1）仍然需要。
  - 跨站请求伪造：`SameSite=Strict`，加上令牌只在响应体中返回、跨站读不到，最坏的情况是对方让令牌换了一次新。
  - 改动只在 `identity` 的 HTTP 适配器和令牌管理器两处。

### 8.6 其他
- JSON 接口的请求体上限 1 MiB（3.11）；每个接口请求有 15 秒的期限（3.6）。
- 客户端 IP 只在对端是可信代理时才取自 `X-Forwarded-For`（3.10）。
- PAT 拥有账户的全部能力，v0 不做权限范围（与 Plane 相同）。管理员重置密码会撤销全部 PAT，恢复账户时能赶走持有 PAT 的攻击者（3.5）。
- `nrv_pat_`、`nrv_rt_` 前缀让代码托管平台的密钥扫描能认出泄露的令牌。
- 直接用 HTTP 部署时，浏览器不在安全上下文中：跨标签页的续期用租约（7.1），复制令牌用剪贴板工具已有的 `execCommand` 退路（`web/packages/utils/src/string.ts:201-206`）。README 的部署说明建议公网部署用 HTTPS。

---

## 9. 测试

### 9.1 后端单元测试
| 对象 | 内容 |
|---|---|
| `identity/domain` | 邮箱规范化；名字中的网址；显示名；时区；主题、语言、每周第一天；新手引导的部分更新（只传一个键，其余不变；值不是布尔值时拒绝）；密码规则（表格驱动：8、128 的边界，缺各类字符，非 ASCII 字符；主干的提取；名单命中和不命中的例子，包括 3.8 列出的那些；主干等于邮箱前缀）；刷新令牌的编码和解码（长度不对、前缀不对、base64 非法）；轮换的判定（当前代、更旧的代、伪造）；PAT 的格式、名称、过期时间 |
| `identity/app` | 每个用例用内存里的假端口测试：注册（关闭注册、邮箱已存在、三行在一个事务里）；登录（邮箱不存在时仍做一次假校验、停用只在密码正确后报出、参数变化后重新哈希）；续期（轮换、重复使用时撤销、伪造时不撤销、`expires_at` 不变、用固定时钟推到期限之后续期失败）；退出（只有当前一代有效的令牌才撤销，其余都是 204 且不变）；修改密码（撤销其他会话；用 PAT 修改时撤销全部；PAT 不变）；重置密码（撤销全部会话和全部 PAT，返回数量）；修改邮箱（两个邮箱都规范化；新邮箱已被使用时 `identity.email_taken`；新旧相同时报错；撤销全部会话，PAT 不变）；PAT 的创建、列出、撤销；认证（JWT 加会话检查；PAT 的过期和 `last_used` 的写入频率）；清理过期会话 |
| 适配器 | 6.3 表中各端口的测试；`authn` 把 `Actor` 放进 `context` 并返回限流键；操作级限流的键和桶 |
| `platform` | 认证中间件（公开操作不看令牌；非公开操作没有令牌、令牌无效、令牌有效；无效令牌计入按 IP 的桶）；`Router` 记下注册的模式；限流（桶、清理闲置的键、`Retry-After`）；请求期限；可信代理下的客户端 IP；413；`ProblemError` 的映射（字段错误、`RetryAfter`、不满足接口时 500），包括不带 Go 类型名的解码错误和 `context.Canceled`；`webui` 的 CSP 哈希；配置的新校验和打码 |

### 9.2 集成测试（连接真实的 Postgres，`pgtest`）
- 仓储的每条查询，包括：
  - 续期的条件 `UPDATE`（时刻作为参数传入）；
  - 游标分页（`created_at` 相同的两行不重不漏）；
  - 违反唯一约束和 CHECK 时映射为领域错误；
  - `onboarding_step` 缺键时被 CHECK 拒绝；
  - 重置密码在一个事务里撤销会话和 PAT。
- `TxManager`。
- 5 个迁移都能 up、down、再 up。
- River worker 的 `Work()` 只删除过期的会话。
- `archtest` 的 `TestSQLCSchemaScope`（3.14）。

### 9.3 契约测试
- `identity`、`instance` 的 handler 测试对每种状态（包括每种 problem）调用 `CheckResponse`。
- `bootstrap` 的三个整程序测试（3.6）：公开集合等于 `security: []`；注册的 `/api/v0` 路由等于接口描述的全部操作；每个非公开操作不带令牌得到 401 problem。另外，带 PAT 调用 `GET /me` 成功。
- `apitest`：
  - 写法检查加上 `security` 的两条规则（3.12）和 `x-problem-codes` 的写法（3.11）；
  - `CheckResponse` 核对 problem 的码在这个操作声明的集合里；每个模块的 handler 测试跑完时，核对声明的码都被返回过（3.11）；
  - 新增 `CheckRequest`，测试中发出的请求也按契约校验（3.11）。
- 错误出口的测试（3.11）。

### 9.4 前端单元测试（vitest）
- 令牌管理器：
  - 同一标签页内只续期一次；
  - 用假的 `locks` 验证跨标签页串行；没有 `locks` 时用假的 storage 和假时钟验证租约的获取、等待、过期后重新获取、只删除自己的租约；
  - 过期时刻由 `access_token_expires_in` 和本地收到响应的时刻算出，本机时钟偏差不影响；
  - 401 → 续期 → 重发一次；续期得到 401 时结束会话；续期得到 429、5xx、网络错误时保留刷新令牌并退避；
  - 没有刷新令牌时不请求 `/me`；
  - 退出在锁内用不挂认证中间件的客户端，交出的是最新一代；
  - `storage` 事件。
- `isValidNextPath` 的控制字符；`next_path` 的拼接（路径 + 查询 + 片段，整体编码）。
- 错误码到文案键的表与 `api/dist/openapi.yaml` 中的 `x-problem-codes` 相等；字段码的表覆盖 `FieldError.code` 的全部取值。
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
- **故事**：A1–A16，以及 S1–S3 的更新（第 2 节）。A4 另跑一遍删掉 `navigator.locks` 的版本。
- **耗时**：M0 的 `e2e` 任务约 120 秒，超时 20 分钟。M2 加入 16 个故事文件，P5 结束时实测，写进 review。

### 9.6 控制者的浏览器核对
沿用 M1 设计 7.5 的做法：临时脚本不进仓库，脚本全文、假数据、运行命令和断言写进该 Phase review 的附录。
- **P4**：
  - 登录页、注册页的就地错误（包括"密码太常见"）；
  - `next_path` 的各种取值；
  - 两个标签页的续期和退出；
  - **用 HTTP 在局域网 IP 上打开**（nerve 监听 `0.0.0.0`，浏览器打开 `http://<本机局域网 IP>:<端口>`，不是 localhost，所以不是安全上下文）：确认 `window.isSecureContext` 为假、`navigator.locks` 不存在；开两个标签页，等访问令牌过期后同时操作，两边都成功，数据库中这个会话未被撤销；
  - 实例请求失败时的维护页；
  - 新手引导的资料步骤和下一步的出现；
  - 登录页、新手引导页、设置页没有 CSP 违规；
  - 控制台没有"navigate() 应在 useEffect 中调用"的警告。
- **P5**：
  - 个人设置的四个标签页；
  - 主题下拉框在按钮旁展开（zh-CN、en 各一次）；
  - PAT 创建后只显示一次、CSV 下载；
  - 安全页的字段错误，以及安全页的 PAT 列表；
  - 停用账户（按裁定）。

---

## 10. 负责人决策点

下面四项会改变用户能做什么，由项目负责人决定。每项给出选项、代价和建议。
- 决策点 1 已由负责人裁定（2026-09-25，选 B）。选项和代价留在下面备查。
- 决策点 2、3：裁定之前 P1、P2 不受影响；它们的接口和命令放在 P3 的最后几个任务里，界面放在 P5。
- 决策点 4：影响 P1 建的 `profiles` 表。裁定晚于 P1 时按建议建表，选另一项就在 P3 补一个迁移（3.19）。

### 决策点 1：怎样修改登录邮箱（M1 设计 3.13）
**背景**：
- M1 删掉了"向新邮箱发验证码"的旧流程。v0 没有邮件服务，没法证明新邮箱属于这个人。
- 邮箱是登录名，M3 的工作区邀请也按邮箱发出。

| 选项 | 做法 | 好处 | 代价 |
|---|---|---|---|
| A. 自助修改 | 个人设置里输入当前密码和新邮箱（`POST /api/v0/me/change-email`），撤销其他会话 | 用户自己就能改 | 多一个接口、一个表单和它的文案。新邮箱不经验证就生效：填错了也会生效；可以改成别人还没注册的邮箱，对方之后无法注册；如果 M3 按邮箱匹配邀请，改过去的人还会收到发给对方的工作区邀请（与决策点 2 的①相同） |
| B. 只能由服务器管理员修改 | `nerve users set-email --email <旧> --new-email <新>`，撤销该账户的全部会话 | 代码最少：一个命令，复用同一个用例。管理员可以先核实身份，和"忘记密码由管理员重置"是同一种信任方式 | 用户要找管理员 |
| C. v0 不提供 | — | 没有代码 | 邮箱写错或公司换了域名，只能新建账户，数据留在旧账户上 |

**建议：B**。
- 改邮箱很少发生；v0 没有邮箱验证，让管理员经手更稳妥；成本最低。
- 以后接入邮件服务时，再做带验证的自助修改。

**负责人裁定（2026-09-25）：B**。只能由服务器管理员用 `nerve users set-email --email <旧> --new-email <新>` 修改，撤销该账户的全部会话。没有界面，也没有接口。实现在 P3：命令和它的用例；两个邮箱都按注册时的规则规范化；新邮箱已被使用时，命令失败并给出明确的说明（3.17、A16）。

### 决策点 2：注册默认是否开放，第一个账户从哪里来
**背景**：
- Plane 默认开放注册（`ENABLE_SIGNUP` 默认 `"1"`）。但 Plane 要先由实例管理员在管理后台完成设置，之后才有人能注册。
- Nerve 没有这一步，也没有实例管理员（3.16）。
- v0 没有邮箱验证：注册时填的邮箱不必属于注册的人。
- M3 的邀请：总体设计 1.1 保留"系统内接受邀请"，也就是被邀请的人登录后，按邮箱看到并接受发给他的邀请；M1 设计 3.15 把"凭邀请链接接受"这条路留给 M3 决定。

| 选项 | 做法 | 好处 | 代价 |
|---|---|---|---|
| A. 默认开放（照搬 Plane 的默认值） | 第一个人直接注册。团队都注册之后，运维把 `auth.signup_enabled` 改为 `false`，以后靠 M3 的邀请加人 | 部署完就能用；与 Plane 一致 | 服务器能从公网访问、运维又没关注册时：① **抢注邮箱，截走邀请**：任何人都能用别人的邮箱（例如 `alice@corp.com`）注册。如果 M3 按邮箱匹配邀请，之后发给 Alice 的工作区邀请会出现在抢注者的账户里，他可以接受并进入工作区；Alice 本人反而注册不了。② 陌生人可以创建自己的工作区（除非同时关闭 `workspace.creation_enabled`）。③ M5 之后，陌生人可以上传文件，占用服务器的磁盘或对象存储：只有单个文件的大小上限，没有总量配额。④ 注册接口会暴露某个邮箱是否已注册（8.2） |
| B. 默认关闭 | 第一个账户用新命令 `nerve users create --email …` 创建（密码从标准输入读）；其他人靠 M3 的邀请注册 | 默认安全：①–④ 都要运维主动打开注册才会出现 | 多一个命令（复用注册的用例，约 40 行）和 README 的一步；开发环境也要先跑命令建账户；M3 的邀请做好之前，每个账户都要用命令行创建，或者临时打开注册 |
| C. 按环境：prod 默认关闭，dev 和 test 默认开放 | 同 B，只在 prod 生效 | 生产环境默认安全；开发和端到端测试不受影响 | 同 B 的命令；dev 与 prod 的默认值不同（签名密钥、argon2 参数本来就按环境不同） |

- **无论选哪一项，都给 M3 一条交接**：邮箱未经验证期间，接受邀请不能只靠邮箱匹配，要凭邀请链接中的令牌（M1 设计 3.15 留下的那条路）；关闭注册时"被邀请的人仍可注册"也改为凭邀请令牌。
  - 这条交接关掉的是①。②③ 只能靠关闭注册或关闭创建工作区。
  - 它会改变总体设计 1.1"系统内接受邀请"和 4.2"被邀请的邮箱始终可以注册"的做法，由 M3 的设计交负责人确认。
- **建议：C**。
  - 第一稿建议 A，理由是"陌生人注册后看不到任何已有的工作区"。M3 之后这句话不成立（①），而且第一稿没有列出 ②③。
  - 放在公网上的自托管服务，默认开放注册的风险落在不知情的运维身上；默认关闭的代价只是部署时多跑一条命令。
  - 选 C 而不选 B：开发和端到端测试每次都要先建账户，却没有安全上的收益。

### 决策点 3：停用账户
**背景**：
- Plane 在个人设置里提供自助停用（`DELETE /api/users/me/`，`plane/apps/api/plane/app/views/user/base.py:252-348`）：
  - **本意**是：账户是某个项目或工作区唯一的管理员、而且那里还有别的成员时，拒绝停用（`:265-305`）。
  - **实际上这个检查从不拒绝**（按代码推导，未运行 Plane）：两段查询都只取这个用户自己的成员记录，按每一条分组计数。`Count(Case(…, default=0))` 对每一行都计 1，所以"存在其他管理员"恒为真；`total_members` 也恒为 1。停用后可能留下没有管理员的项目或工作区。
  - 其余步骤：停用这个用户的全部成员关系；删除发给这个邮箱的全部工作区邀请（`:313`）；删除全部会话（`:316`）；重置新手引导；把密码改成随机值；发一封邮件。
  - 不要求输入密码。只有命令 `activate_user` 能恢复。
- Nerve 的前端保留了这个按钮和弹窗（`web/apps/web/core/components/settings/profile/content/pages/general/form.tsx:398-408`、`web/apps/web/core/components/account/deactivate-account-modal.tsx`）。
- 总体设计 4.2 提到"账户停用时让所有会话失效"。
- 成员关系和邀请属于 M3。"唯一管理员"的检查必须能**拒绝**停用，所以不能是停用之后再发出的事件：
  - 停用用例声明一个端口，由 M3 实现，在停用的同一个事务里调用；它返回错误时，整个停用回滚。
  - M2 没有成员数据，不写空实现。端口和它的实现在 M3 一起加入；M2 的停用只处理账户、会话和资料。

| 选项 | 做法 | 好处 | 代价 |
|---|---|---|---|
| A. 照搬 Plane 的自助停用，另加管理员命令 | `POST /api/v0/me/deactivate`：账户不能再登录，全部会话撤销，PAT 随之失效，新手引导的进度重置。`nerve users deactivate / activate --email`：管理员停用离职的人，以及恢复。M3 实现上面的端口：唯一管理员时拒绝（按 Plane 的本意，修正它的查询缺陷，登记差异），停用成员关系，删除发给这个邮箱的邀请。不把密码改成随机值：Nerve 用 `is_active` 挡住登录，恢复后原密码仍可用 | 保留现有的界面，和 Plane 一致；离职的人也有办法停用 | 一个接口、两个命令、一个给 M3 的端口。和 Plane 一样不要求输入密码：拿到访问令牌的人可以停用这个账户（管理员能恢复） |
| B. 只能由管理员停用和恢复 | `nerve users deactivate / activate --email`；删除界面上的按钮和弹窗。M3 的端口同 A | 不会误操作，也不会被盗用的令牌停用；离职由管理员处理，贴近团队的用法 | 用户不能自己注销；删掉一个保留功能的界面 |
| C. v0 不提供 | 删除按钮和弹窗，不做接口和命令 | 没有代码 | 总体设计 4.2 的"账户停用"没有入口；员工离职后只能由 M3 把他移出工作区，账户本身仍能登录，PAT 仍然有效 |

**建议：A**。
- 它是保留的功能，照搬 Plane；管理员命令补上了离职的情况。
- A、B 都需要 M3 的端口，A 只比 B 多一个接口。
- 负责人如果更担心"被盗用的令牌停用账户"，选 B，代码量相近。

### 决策点 4：新手引导的"角色""用途"两步（M1-P3 交接）
**背景**：
- Plane 把"是否自托管"写死为真，自托管版的新手引导因此跳过"你的角色""你打算怎么用"两步（3.19）。Plane 自托管版的用户从来看不到这两步。
- Nerve 只有自托管一种形态。
- M1-P3 交接把它列为产品决定。

| 选项 | 做法 | 好处 | 代价 |
|---|---|---|---|
| A. 删除（与 Plane 自托管版的实际表现相同） | 删除这两步（338 行）、实例配置的 `is_self_managed`，以及资料中的 `role`、`use_case` 两列（3.19） | 用户看到的与 Plane 自托管版完全相同；代码和数据都更少 | 以后想了解用户的角色和用途，要重新加回 |
| B. 显示这两步 | 两步总是显示；`role`、`use_case` 保留在资料里，进入接口 | 可以收集用户的角色和用途 | v0 没有任何功能读这两项，收集了也用不上；与 Plane 自托管版的流程不同，是一处产品改动，要登记差异；新手引导多两步 |

**建议：A**。Nerve 只有自托管一种形态；B 收集的数据在 v0 没有使用者。

---

## 11. 需要负责人确认的事项与架构问题

### 11.1 请负责人确认：退出登录只结束当前这一处登录
- **总体设计 4.2 已批准的原文**："退出登录、修改密码、账户停用时，吊销该账户的刷新令牌。"按字面读，在一处点"退出"，这个账户在所有浏览器、所有设备上的登录都会结束。
- **M2 的做法**（3.5）：
  - 在一处点"退出"，只结束这一处的登录；其他浏览器、手机上的登录不受影响。
  - 修改密码时，结束其他各处的登录，保留正在操作的这一处。
  - 账户停用时，结束全部登录。
  - 这和 Plane 的行为相同（Plane 的退出只结束当前的会话，`plane/apps/api/plane/authentication/views/app/signout.py:25`）。
- **另一种做法**："退出"就是"在所有地方退出"。好处是规则简单，丢了设备时有办法结束那里的登录。代价是在公司电脑上点退出，手机上也被迫重新登录，而且与 Plane 不同。
- **建议按 M2 的做法**，收尾时把 4.2 改写为"退出只结束当前的登录；修改密码结束其他各处的登录；账户停用时结束全部登录"。负责人如果选另一种，只改 3.5 表中的一行，代价很小。以后需要"在所有设备上退出"时，也可以作为单独的按钮加入。

### 11.2 架构问题
- 没有需要负责人裁定的架构问题。
- 评审提出的"平台包能否导入 `internal/shared`"，已由控制者裁定为不导入（3.3）：平台自己声明认证器和错误接口，`identity`、`shared` 按结构满足它们。这保持了总体设计 6.2"平台与业务无关"，M0 设计 3.1 不用改。
- 以下几处看起来像与上级设计冲突，核对后都在现有规则内，已作为裁定写在第 3、5 节，请控制者复核时确认：
  1. **M1-P4 交接要求服务端校验 `next_path`**（3.18）：它的前提（服务端发出跳转）在 JSON 接口下不存在了。
  2. **后续 M 的字段**（3.2）：本身可为空的引用字段随实体进入接口，值为 `null`；实例配置的字段按 M1 设计 3.7 在 M2 定义。
  3. **端口放在 `app` 而不是 `domain`**（3.3）：M0 起就是这样，收尾时改写总体设计 6.2。
  4. **注册返回令牌**（5.1），与总体设计 3.1"POST 返回完整资源"的字面不同，收尾时写进 3.1。

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
- **后端分三段**：平台约定和第一个能用的认证（P1）→ 会话的安全逻辑（P2）→ 账户的其余接口（P3）。
  - 平台约定约束后续每个 M；会话的轮换、重复使用检测和限流是安全上最敏感的代码。两者放在同一个 Phase 里，评审时会互相稀释（评审 I10；M1 的教训）。第一稿的 P1 约有 20 个任务，两者都在里面。
  - P1 的完成线包含默认拒绝的整程序测试：需要令牌的 `getMe` 合并时，安全网同时合并（评审 I9）。
- **前端内先做令牌管理器和登录页**：设置页要靠它们才能访问。
- **每个 Phase 合并的都是能用、有测试的代码**：后端 Phase 用接口版本的故事验收，页面版本随前端 Phase 加入。
- **比较过的其他拆法**：
  - 按功能前后端一起切（"认证""账户"各一个 Phase）：一个 Phase 同时评审 Go 和 TS，令牌管理器还要等后端做到一半。
  - 单独的"基础设施"Phase：会合并一批暂时没人用的代码（M0 设计 7.2 不这样做的理由）。P1 用 `register` 和 `getMe` 让平台约定当场有使用者。

### P1 `platform-core`：平台约定与第一个认证（后端）
- **目标**：任何调用方都能注册，并用注册得到的令牌访问 `GET /me`；不带令牌访问非公开操作一律 401；后续 M 照做的平台约定全部定下。
- **交付物**：
  1. 表结构约定（3.13）；迁移 `00001`–`00003`；sqlc 接入（3.14），按模块限定 `schema` 和 `TestSQLCSchemaScope`；架构测试规则 4 加上 `internal/shared`，规则 6 推广到 `adapter/*/gen`。
  2. 平台：
     - `internal/shared`、`platform/postgres` 的事务和 UTC、`platform/clock`；
     - `httpserver` 的 `Router`、`API` 值、默认拒绝的认证中间件、请求元信息、请求期限、请求体上限、`ProblemError` 的映射和新平台码、导出 `RequestID`（3.6、3.11）；
     - 配置（6.5 中 P1 用到的部分），以及 M0-P2 交接 8 的三个小问题。
  3. 接口描述：oapi-codegen 的模块模板；`security` 的写法和 `apitest` 的两条规则；`x-problem-codes` 和 `apitest` 的三项核对；`FieldError.code`；`CheckRequest`；`writeOnly` 的规则（3.11、3.12）。
  4. `identity`：
     - `register`、`getMe`；
     - argon2id，含并发上限和等待上限；密码规则、常见密码名单和生成它的脚本（3.8）；
     - Ed25519 JWT 和密钥（3.7）；注册时建会话、发令牌；
     - 认证用例和 `authn` 适配器（JWT 加会话检查）。
  5. `bootstrap` 的三个整程序测试（3.6）。
  6. 端到端：
     - fixture 的扩展（9.5，页面的登录状态除外；`auth.ts` 这时只有注册）；
     - S1 的迁移断言；
     - A1、A2 的接口版本。
- **关闭**：M0-P1；M0-P2 的第 1、3、4、6、7、8 条和第 2 条中认证的部分；M0-P3；M0-P4；M0-P6 中与 fixture、数据库断言、S1、端口、等待期限、录像有关的几项。
- **完成线**：
  - A1、A2 的接口版本和 S1–S4 在持续集成中通过；
  - 三个整程序测试和 `apitest` 的新核对通过；
  - `make gen-check` 覆盖 sqlc 的输出；
  - 架构测试和传递依赖测试通过。

### P2 `sessions`：登录与会话（后端）
- **目标**：登录、续期、退出可用；刷新令牌的轮换和重复使用检测、会话的绝对期限、限流、安全响应头全部到位。
- **交付物**：
  1. `login`（假哈希、停用只在密码正确后报出、重新哈希）；`refreshTokens`（轮换、重复使用检测、绝对期限）；`logout`（只认当前一代）（3.5、3.9）。
  2. 限流：`platform/ratelimit`；`httpserver` 的限流中间件（按凭证、按 IP，无效令牌计入按 IP 的桶）；`identity` 适配器中登录、注册的桶；`server.trusted_proxies`（3.10）。
  3. 固定链上的安全响应头（8.3）。
  4. 端到端：`auth.ts` 加上登录；A3、A4、A5、A6、A15 的接口版本。
- **关闭**：M0-P2 第 2 条中限流的部分；M0-P5 中安全响应头的部分。
- **完成线**：
  - A3–A6、A15 的接口版本通过，P1 的故事仍然通过；
  - 登录的耗时测试（邮箱存在与否，耗时相同）通过；
  - 3.10 表中的每个桶都有测试。

### P3 `account-api`：账户接口（后端）
- **目标**：账户的其余接口都可用，而且都能用 PAT 完成；River 的第一个定时任务运行。
- **交付物**：
  1. `updateMe`、`getProfile`、`updateProfile`（部分的 `onboarding_step`）、`changePassword`（3.5 的撤销规则、`password_user` 桶）。
  2. PAT：
     - 迁移 `00004`；
     - 创建、分页列表（`common.yaml` 的分页组件）、撤销（`DELETE /api-tokens/{token_id}`）；
     - PAT 认证和 `last_used`。
  3. `instance` 的三个字段和 `listTimezones`；嵌入 `time/tzdata`。
  4. 管理命令：`nerve users reset-password`（撤销会话和 PAT）；`nerve users set-email`（决策点 1，已裁定为 B，3.17）；`Admin()`；决策点 2、3 的接口和命令；决策点 4 选 B 时补迁移。
  5. River（3.15）：
     - 迁移 `00005`；
     - `platform/jobs`（服务用和只投递两种客户端）；
     - `bootstrap` 的运行和停机顺序；命令行的最小组合（3.17）；
     - 清理过期会话的定时任务。
  6. 端到端：
     - A7–A14、A16 的接口版本（需要登录的都用 PAT）；
     - S3；
     - 实测 nerve 的停机时间。
- **关闭**：
  - M0-P2 第 5 条；
  - M0-P6 的 S3 和 River 停机两项；
  - M1-P2、M1-P3 中接口描述的部分：主题字段；实例字段；不再读的字段和不再调用的地址都不出现在接口描述里；令牌地址不带结尾 `/`；
  - M1-P3 中修改登录邮箱的一项（决策点 1 已裁定，命令在本 Phase 实现）。
- **完成线**：
  - A7–A14、A16 的接口版本通过，P1、P2 的故事仍然通过；
  - 每个需要登录的操作都有 PAT 的测试；
  - 三个整程序测试覆盖新加的全部操作。

### P4 `web-auth`：前端认证
- **目标**：浏览器通过令牌管理器登录、续期、退出；Cookie 和 CSRF 从前端消失；用 HTTP 部署时多标签页也能正常续期。
- **交付物**：
  1. `@nerve/api-client` 接入 web（`--root-types`）；令牌管理器、跨标签页的协调（`navigator.locks` 和租约）和它们的测试（7.1）。
  2. 传输层的清理：CSRF 4 处、表单提交、web 的 axios 基类（7.2）。
  3. 登录页、注册页、错误文案表（7.3）；`next_path`（3.18）。
  4. `AuthenticationWrapper`、`InstanceWrapper`（7.4）；user、profile、instance 三个 store 和相关类型，以及删除 `IUser` 牵连的地方（7.5 中与认证有关的部分）。
  5. 新手引导：按决策点 4 处理两步和 `is_self_managed`；资料步骤，删掉它的头像上传（3.2、3.19）。
  6. `webui` 的 CSP（8.3）。
  7. 关键词规则（7.9 中 P4 的部分）。
  8. 端到端：
     - `signedInPage` fixture；
     - A1–A6、A10、A15 的页面版本（A4 含删掉 `navigator.locks` 的一遍）；
     - S2。
- **关闭**：M0-P5 中 CSP 和前端的部分；M1-P3 中 CSRF、认证错误、实例字段的部分；M1-P4；M0-P6 的 S2。
- **完成线**：
  - `git grep -n -E "csrfmiddlewaretoken|X-CSRFTOKEN" -- web` 没有输出；
  - 上述故事的页面版本通过；
  - 9.6 中 P4 的浏览器核对（含局域网 HTTP）写进 review。

### P5 `web-account`：前端账户设置
- **目标**：个人设置的四个标签页对接新接口；M2 领域的前端清理完毕。
- **交付物**：
  1. general、preferences、security（含 PAT 列表）、api-tokens（7.7）；PAT store（7.5）；删掉 general 页的头像和封面上传控件（3.2）；主题下拉框的定位。
  2. 决策点 3 的界面（选 A 时保留按钮和弹窗并对接接口；选 B、C 时删除）。
  3. `@nerve/services` 删到只剩地址工具和文件工具；删除剩下的 Plane 类型（7.5）。
  4. 清理：
     - 死成员和死 prop 的 66 行；
     - oxlint：改到的文件清零，`no-unneeded-ternary` 一类清零，上限调低（7.8）。
  5. 关键词规则（7.9 中 P5 的部分，包括 `withCredentials:\s*true`）。
  6. 端到端：A7–A9、A11、A12 的页面版本。
- **关闭**：M1-P2；M1-closeout；M1-P3、M1-P4 剩下的事项。
- **完成线**：
  - 全部故事的页面版本和接口版本通过；
  - `domains.mjs --rows M2` 没有输出，或者剩下的每一行都在 review 中写明是误报；
  - 9.6 中 P5 的浏览器核对写进 review。

### 收尾 `closeout`
- 10 份交接逐项写下结论，`status` 改为 `closed`。
- 同步文档（3.20）：总体设计、M0 设计、M0/P3 spec、差异清单（二·全局、二·按表、三、四）、前端改动清单。总体设计 4.2 按负责人对第 11 节的确认改写。
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
| M0-P2-platform-notes | 1 `TxManager` 由使用方声明 | P1（`internal/shared`、`platform/postgres`，按结构满足） |
| | 2 认证、限流、接口调用日志按路由挂载 | P1 认证（3.6）；P2 限流（3.10）。接口调用日志由 M8 挂在限流之后（13.2） |
| | 3 导出 `RequestID` | P1 |
| | 4 新的密钥类配置写进 `LogValue` | P1（6.5） |
| | 5 River 与停机顺序、连接池关闭的时限、handler 的期限、River 的迁移 | P1 请求期限（3.6）；P3 River、停机和迁移（3.15） |
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
| M0-P5-frontend-api-notes | 改调 `/api/v0/instance`；认证接口在 `/api/v0/` 下；同源 | P1、P2（接口路径）；P4（前端） |
| | 安全响应头与 CSP 放在同一层 | P2（固定链上的响应头）；P4（CSP）。分在两层的理由见 8.3 |
| M0-P6-e2e-notes | PAT 对等验收与认证 fixture | P1（注册）、P2（登录）、P3（PAT）、P4（页面） |
| | `db.ts` 的连接池、断言的写法、失败时的数据库快照 | P1（9.5） |
| | S1 的迁移版本 | P1 |
| | S3 的新字段 | P3 |
| | S2 的"没有失败的接口请求"、控制台、`networkidle` | P4。没有刷新令牌时不请求 `/me`（7.1）；登录页没有轮询，`networkidle` 继续可用；有轮询的页面不进 S2 |
| | 端口竞争的根本解决 | P1（`server.addr_file`） |
| | 新等待的期限 | P1–P5 |
| | River 停机是否仍在预算内 | P3 |
| | 是否录像 | P1 定为不录像，收尾同步总体设计 8.2（9.5） |
| | fixture 写法的延伸（`storage.ts`、`webhook.ts`、`clock.ts`） | 不属于 M2：M5、M8、M4 各自按 P6 的写法加入 |
| M1-P2-trim-content | 主题只剩一个值，五个取值，没有 `custom` 和调色板 | P3（接口，5.2）、P4（`IUserTheme` 删除，改用生成的类型） |
| M1-P3-trim-platform | 删除 Cookie 会话和 CSRF | P4（7.2） |
| | 认证错误就地显示 | P4（7.3） |
| | 修改登录邮箱 | 决策点 1，负责人 2026-09-25 裁定为 B：`nerve users set-email`，在 P3 实现（3.17、A16）；没有界面和接口 |
| | `IInstanceConfig` 的 4 个字段；`is_self_managed` 和新手引导的两步；`enable_signup` 的名字 | P3（接口，5.3）、P4（前端）；两步见决策点 4 |
| | 不再读的用户字段、不再调用的地址不出现在接口描述里 | P3；收尾再核对一次 |
| | 令牌地址统一不带结尾 `/` | P3（5.1） |
| M1-P4-router-native | 服务端校验 `next_path` | 3.18：服务端没有跳转；前端的校验补上控制字符（P4） |
| | 重写 `AuthenticationWrapper` | P4（7.4） |
| | 401 处理不照搬恒为真的判断 | P4（7.1、7.2） |
| | 由 M2 决定：`next_path` 带查询和片段 | 3.18：带上 |
| | 由 M2 决定：`security.tsx` 的错误断言 | P5（7.7） |
| | 由 M2 决定：表单不再提交到 `/auth/…` | P4（7.2） |
| M1-closeout | 死成员和死 prop（66 行） | P4、P5（7.8） |
| | oxlint：改到的文件清零，另清一类规则 | P4、P5（7.8）：`no-unneeded-ternary` |
| | 主题下拉框的位置 | P5（7.7） |

**没有落点的事项：无。** 唯一不在 M2 实现的是 M0-P6 最后一项（其他 fixture 的写法），它本来就写给 M4、M5、M8。

### 13.2 M2 交给后续 M 的事项（收尾时写成交接）
| 接收 | 事项 |
|---|---|
| M3 | **邀请**：邮箱未经验证期间，接受邀请不能只靠邮箱匹配，要凭邀请链接中的令牌（M1 设计 3.15 留下的路径）；关闭注册时，持有有效邀请的人仍可注册。后者由 M3 扩展 `SignupPolicy` 的实现和注册请求（加上邀请令牌），不另加端口（评审 M16）。这改变总体设计 1.1"系统内接受邀请"和 4.2"被邀请的邮箱始终可以注册"的做法，由 M3 的设计交负责人确认（决策点 2）。参考：Plane 把任何未删除的工作区邀请都算上（`plane/apps/api/plane/authentication/adapter/base.py:102-120`） |
| M3 | 登录后的工作区落点：`/users/me/settings/` 的数据和 `AuthenticationWrapper` 的落点取数（3.1）；`profiles.last_workspace_id` 是否补外键（`ON DELETE SET NULL`） |
| M3 | `workspace_creation_enabled` 的执行，以及关闭时是否提供创建工作区的命令（3.16） |
| M3 | 决策点 3 选 A 或 B 时：给停用用例加上它声明的端口并实现，在停用的同一个事务里调用。唯一管理员时拒绝（按 Plane 的本意，修正它查询的缺陷，登记差异）；停用成员关系；删除发给这个邮箱的邀请 |
| M3 | 3.2 的字段规则：工作区图标、项目封面、成员头像随实体进入接口，M5 之前为 `null`；`IUserLite.avatar_url` 改为可为 `null`；读取它们的代码不动 |
| M3 | sqlc 的模块边界（3.14）：`access` 模块读成员表，是在 `app` 层声明端口，还是作为 `TestSQLCSchemaScope` 中写明理由的例外 |
| M3 | CSP：表情选择器从 `cdn.jsdelivr.net` 下载 `emojibase-data`，改为随前端一起构建、从本站提供（8.3） |
| M3 | 新手引导的创建工作区、加入工作区、邀请成员三步；`user.service.ts` 中留下的 `leaveWorkspace`、`joinProject`、`leaveProject`；`IUserLite` 的 `is_bot`；时区接口也供工作区和项目设置使用 |
| M4 | `user.service.ts` 中的 `getUserProfileIssues`；事件订阅者的写法（3.15）；建"物理删除软删除超过 60 天的数据"定时任务时，把 `api_tokens` 纳入；CSP：编辑器 callout 的默认表情图来自 `cdn.jsdelivr.net`，改为本站资源或原生表情，并核对表情回应（8.3） |
| M5 | `users.avatar_asset_id`、`cover_image_asset_id`，`User.avatar_url`、`cover_image_url` 开始返回签名地址；按新的上传协议加回 general 页和新手引导资料步骤的上传控件（3.2）；`file_size_limit` 的执行；CSP 的 `img-src`、`connect-src` 加上存储的来源；本地存储的上传地址按 3.6 的整程序测试处理：写进接口描述，或不放在 `/api/v0` 下 |
| M8 | 接口调用日志挂在限流之后（3.6）；Go 进程空闲内存实测时，一并测 argon2 并发上限下的峰值和常见密码名单占用的内存（3.8）；对外接口文档页不从 CDN 加载脚本（8.3） |

---

## 14. 完成标准
- [ ] P1–P5 和收尾全部完成，每个 Phase 都有 spec、plan 和 review。
- [ ] A1–A16 的页面版本（有页面的）和接口版本全部通过，S1–S4 通过。需要登录的每个操作都有 PAT 版本，调用同一组数据库断言。
- [ ] 后端：单元测试、集成测试、契约测试（三个整程序测试、`apitest` 对错误码的核对）、架构测试（含规则 4 的扩展和 `TestSQLCSchemaScope`）和 depguard 全部通过；`make gen-check` 覆盖 oapi-codegen、sqlc 和 TS 客户端。
- [ ] 前端：类型检查通过，knip 为零；oxlint 等于上限，上限已按 7.8 调低；前端单元测试通过；关键词守卫没有未登记的命中。
- [ ] `git grep -n -E "csrfmiddlewaretoken|X-CSRFTOKEN" -- web` 没有输出；前端不再调用 `/auth/…` 和 Plane 的用户、实例、令牌、时区地址（7.9 的规则守着）。
- [ ] 4 张表由 M2 的迁移创建，以 Plane 表结构快照为起点；差异清单登记了全局的建表约定（3.13）、每一列的改动和每一条行为差异（4.6）。
- [ ] 第 10 节的四个决策点都有负责人的裁定（决策点 1 已于 2026-09-25 裁定为 B），并已实现；第 11.1 节有负责人的确认。
- [ ] 浏览器核对（9.6，含局域网 HTTP）的脚本全文写在各 Phase review 的附录中。
- [ ] `handoffs/` 中没有 `open` 的事项；交给 M3、M4、M5、M8 的交接已写好（13.2）。
- [ ] 总体设计、M0 设计、差异清单、前端改动清单已按 3.20 同步；总体设计中 M2 的状态改为"已完成"。
- [ ] 收尾 review 写明规模估计与实际的对比（7.10）。

---

## 15. Phase 进度表

| Phase | 名称 | 状态 | spec | plan | review |
|---|---|---|---|---|---|
| P1 | platform-core | 未开始 | — | — | — |
| P2 | sessions | 未开始 | — | — | — |
| P3 | account-api | 未开始 | — | — | — |
| P4 | web-auth | 未开始 | — | — | — |
| P5 | web-account | 未开始 | — | — | — |
| 收尾 | closeout | 未开始 | — | — | — |

---

## 16. 风险

| 风险 | 应对 |
|---|---|
| 重复使用检测过严：续期的响应在网络上丢失，客户端重试会被当成重复使用，用户被迫重新登录 | 同一浏览器内由跨标签页的协调避免并发续期；发现重复使用时记 WARN 日志，P4、P5 的核对中观察它是否频繁出现。真的频繁时，可以加"上一代令牌在几秒内仍可用"的宽限期，但要多存一代哈希，留到以后 |
| 租约不是原子的：没有 `navigator.locks` 时，两个标签页极小概率同时续期 | "写入后再读一次"；代价与上一行相同。A4 的无 `locks` 版本和局域网 HTTP 的浏览器核对观察它 |
| 默认拒绝漏声明公开操作：公开的接口被 401 | 整程序测试要求公开集合等于接口描述的 `security: []`，漏一个就失败 |
| 每个带令牌的请求多一次数据库查询 | 按主键查询；M8 做性能和内存实测时一起观察 |
| openapi-fetch 0.17.0 的中间件能否重发请求 | P4 先做原型；不行时在薄 service 层统一包一层"401 后续期重试" |
| 生成的类型替换 `IUser` 等之后，牵连的使用方比 7.5 列出的多 | 类型检查会列出全部；超出估计时按领域拆成更小的任务 |
| 常见密码名单误伤或漏网 | 判定规则确定，单元测试列出命中和不命中的例子；被拒绝时提示明确；名单可以用脚本重新生成 |
| `x-problem-codes` 与实现不一致 | `apitest` 两个方向都核对：返回的码必须已声明，声明的码必须被返回过 |
| sqlc 和 oapi-codegen 共用一个工具模块，依赖被一起抬高后 oapi-codegen 的输出变化 | P1 加入 sqlc 后立即运行 `make gen-check-go`；有变化时把 sqlc 放进单独的工具模块 |
| sqlc 的解析器只认 PG 17 | 3.13 的语法约定；真正需要 PG 18 的语法时，评估升级到已发布的新版 sqlc |
| River 仍是 0.x | 锁定 v0.47.0；升级时按 3.15 另写迁移，不改已发布的迁移 |
| argon2 占用内存和 CPU | 并发上限 4，约 76 MiB；等待上限 2 秒（3.8）；3.10 的桶限制每个 IP 和每个账户能触发的次数；M8 实测 |
| CSP 挡住某个库的内联脚本或外部资源 | 内联脚本的哈希在启动时计算；外部来源已清点并交给 M3、M4、M5、M8（8.3）；S2 统计违规事件 |
| 端到端的耗时增加 | 独立的 nerve 只用于 A2、A4、A15 三个故事；P5 实测并写进 review |
| 新手引导的工作区步骤在 M3 之前失败 | 预期行为（3.1）；页面版本只断言到"下一步出现" |
| 决策点的裁定晚于开工 | 决策点 1 已裁定。决策点 2、3 不影响 P1、P2，它们的接口和命令放在 P3 的最后几个任务，界面放在 P5；决策点 4 晚于 P1 时按建议建表，另一种裁定在 P3 补迁移 |

---

## 17. 独立评审的落实

第一稿 `f6ea660` 经独立评审（opus）：Ready with fixes，Critical 0、Important 14、Minor 18。控制者对每一条作了裁定，第二稿逐条落实：

| 编号 | 问题 | 落点 |
|---|---|---|
| I1 | 认证默认放行，整程序测试覆盖不全 | 3.6 默认拒绝和三个整程序测试；6.4；9.3；P1 |
| I2 | 只留组合规则是削弱，却写成"修复不一致" | 3.8 常见密码名单；4.6；A2 |
| I3 | argon2 的名额可以被占满，挡住所有登录 | 3.8 等待上限和 503；3.10 的六个桶和每个操作的桶；6.5 |
| I4 | 非安全上下文中没有 `navigator.locks` | 7.1 租约；8.6；9.4；9.6 局域网 HTTP；A4 |
| I5 | 刷新令牌的期限滑动、没有上限，也没有登记 | 3.4、3.5 绝对期限；4.5；4.6；6.5 |
| I6 | 零值也合法的必填字段无法校验 | 3.11 的 `writeOnly` 规则和部分对象；4.3；5.2；A10 |
| I7 | 决策点 2 的选项 A 漏了最大的代价 | 第 10 节决策点 2 重写，建议改为 C；13.2 给 M3 的邀请交接 |
| I8 | Plane 的停用描述有误，M3 的部分规划不足 | 第 10 节决策点 3 重写；6.4；13.2 |
| I9 | P1 合并了放行的中间件，安全网晚一个 Phase | 第 12 节 P1 的交付物和完成线 |
| I10 | P1 太大 | 第 12 节拆成 P1 `platform-core`、P2 `sessions`，后面顺延为 P3–P5 |
| I11 | 字段规则让 M3、M5 先删后加 | 3.2 规则 2；7.5；7.6；7.7；13.2 |
| I12 | 差异登记不全；"第 8 节"的引用错误 | 3.13；3.20；4.6 |
| I13 | 错误码不在契约里 | 3.11 `x-problem-codes`；3.12；7.3；9.3；9.4 |
| I14 | 访问令牌的过期拿本机时钟和服务端时刻比较 | 5.2 `access_token_expires_in`；7.1 退避 |
| M1 | 请求流程图的顺序不对 | 3.6；6.4 |
| M2 | 不用 scopes-on-context 的第二个理由不准 | 3.6 只留"遗留选项" |
| M3 | 时钟端口的位置；两个时间来源；端口放在 `app` | 3.3；3.5；3.13；3.20 |
| M4 | 响应头与 CSP 的层；总体设计 6.4 未列入同步 | 8.3；3.20 |
| M5 | `withCredentials` 守卫加得太早 | 7.2；7.9 |
| M6 | 没有刷新令牌时仍请求 `/me` | 7.1；S2 |
| M7 | 退出的客户端和服务端都没写清 | 3.5；7.1；A6 |
| M8 | 账户恢复后 PAT 仍然有效 | 3.5；3.17；4.6；7.7；A7；A13 |
| M9 | handler 的阻塞调用没有期限 | 3.6 请求期限；6.5 |
| M10 | 缺 400 和 429 的错误种类 | 3.11 |
| M11 | 删除 `IUser`、`Profile.id` 的牵连 | 7.5 |
| M12 | CSP 会挡住保留功能的外部来源 | 8.3 的清点；13.2 |
| M13 | 表级 CHECK 的默认名不稳定 | 3.13；4.5 |
| M14 | 3.9 的比较位置；日志中的会话 id | 3.9；8.4 |
| M15 | 新手引导两步是产品决定 | 3.19；决策点 4 |
| M16 | 给 M3 的交接说要"加端口"，其实已有 `SignupPolicy` | 13.2 |
| M17 | 与总体设计 3.1、3.2 的约定不符 | 3.12 撤销路径改为 `/api-tokens/{id}`；5.1 注册的返回值；3.20 |
| M18 | 模块入口的 `Commands()`；命令行没有 River；sqlc 没有模块边界 | 3.3 `Admin()`；3.15、3.17 只投递的客户端；3.14 按模块限定 `schema` |

- 评审的负责人问题 2（退出的语义）放在 11.1 请负责人确认；问题 3（平台能否导入 `shared`）由控制者裁定为不导入（3.3）；其余问题分别落在上表和第 10 节。
- **核对评审引用时发现的出入**：
  - Plane 重置命令中 zxcvbn 的位置是 `reset_password.py:56`，不是评审写的 `:122`。
  - Plane 停用时的"唯一管理员"检查确实写在代码里，但按代码推导它从不拒绝（第 10 节决策点 3）。第二稿照实描述，并让 M3 按 Plane 的本意实现。
