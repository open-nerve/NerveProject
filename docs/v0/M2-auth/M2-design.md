# M2 账户认证：设计与实施规划

| 项 | 内容 |
|---|---|
| 里程碑 | M2 账户认证（`docs/v0/M2-auth`） |
| 日期 | 2026-09-25 |
| 状态 | 第三稿。第 10 节的四个决策点都已由负责人裁定（2026-09-25：1 选 B，2 选 C，3 选 A，4 选 A）；第 11.1 节已由负责人批准 |
| 上级文档 | [v0 总体设计](../v0-design.md) 1.1、3、4、5、6、7、8、9 节；[差异清单](../plane-diff.md)；[前端改动清单](../frontend-changes.md) |
| 前置交接 | `handoffs/` 中的 10 份：M0-P1、M0-P2、M0-P3、M0-P4、M0-P5、M0-P6、M1-P2、M1-P3、M1-P4、M1-closeout。逐条落到 Phase，见第 13 节 |
| 设计评审 | 第一稿（`f6ea660`）经独立评审（opus），结论 Ready with fixes，第二稿（`498735d`）逐条落实。第二稿经 Codex 对抗性评审（[`reviews/M2-design-codex-adversarial-review.md`](reviews/M2-design-codex-adversarial-review.md)：Critical 0、Important 12、Minor 9）和控制者的复核（N1–N4、m1–m7）。第三稿按控制者的裁定修订，又经控制者复核（R1–R8 和几处细节）和一次核验（F1–F4 和几处细节）后补改，逐条落点见第 17 节 |

---

## 0. 目标与范围

### 0.1 目标
M2 是第一个做真实业务的里程碑，也是前端第一次对接新接口。M2 结束时：
1. 任何调用方都能通过接口注册、登录、续期、退出、修改密码、停用自己的账户，管理自己的资料、偏好和个人访问令牌（PAT）。用 PAT 能做的事和页面一样多。
2. 后端第一次有业务表（4 张）、第一批 sqlc 查询、第一个 River 定时任务（清理过期会话）。以下平台约定一次定下，后续 M 照做：
   - 认证：默认拒绝，只放行声明为公开的操作；认证之前按 IP 的失败闸门；当前账户（Actor）的传递；
   - 限流：每个桶都有速率和突发；
   - 错误码：在接口描述中逐个操作声明；**结构在接口边界，取值在领域**：请求体的未知字段、不合法的 `null`、缺少的必填字段在进入 handler 之前按契约拒绝（400），长度、格式、取值范围由领域层校验（422）；
   - 事务：提交不受请求期限的取消；凭证签发和变更时的账户行锁；
   - 表结构约定：审计时间列由用例的时钟写入；迁移归被改表的模块所有；sqlc 的模块边界；
   - 分页游标的封套；
   - oapi-codegen 的生成选项和请求体结构表的生成。
3. 前端用令牌管理器替换 Cookie 会话和 CSRF。认证、用户、资料、偏好、实例配置、PAT 这几块的 services、stores 改用 OpenAPI 生成的类型，没有转换层。
4. 本领域的每个用户故事都有端到端测试：页面、数据库、接口三层断言；接口版本用 PAT（第 2 节）。
5. 持续集成的全部门禁通过；本 M 的交接全部关闭；给 M3–M8 的交接写清接收条件。

### 0.2 范围
| 包含 | 不包含（留给后续 M） |
|---|---|
| 注册、登录、续期（刷新令牌轮换和重复使用检测）、退出、修改密码、自助停用账户 | 工作区、成员、邀请（M3）。其中包括：关闭注册时持有邀请的人仍可注册；登录后落到哪个工作区；新手引导中创建或加入工作区、邀请成员三步 |
| 当前账户的资料（`/me`）和偏好（`/me/profile`，含新手引导的进度） | 头像和封面的存储与上传（M5）。M2 的接口中 `avatar_url`、`cover_image_url` 恒为 `null`，见 3.2 |
| PAT：创建、分页列出、撤销；用 PAT 调用所有接口 | 权限框架和权限矩阵（M3）。停用时"唯一管理员"的检查（M3，13.2） |
| 实例配置（在 M0 的 `instance` 上扩展）和时区列表 | 接口调用日志（M8） |
| 管理命令：`nerve users reset-password`、`set-email`、`create`、`deactivate`、`activate`（3.17） | 邮件、找回密码、邮箱验证（v0 不做，总体设计 1.2）；用户自助修改邮箱（决策点 1） |
| River 的第一个定时任务（清理过期会话）；sqlc 的第一批查询；第一次按快照建表的约定 | 实例管理员（v0 没有，见 3.16） |
| 前端：令牌管理器；登录页、注册页；认证包装；新手引导的资料步骤；个人设置的四个标签页；删除 CSRF、Cookie 和表单提交 | 刷新令牌改放 HttpOnly Cookie（只写下升级路径，见 8.5） |
| 端到端：认证 fixture、PAT 对等验收、数据库断言的写法 | 可注入的时钟（M4，M0-P6 交接） |

### 0.3 关于文中的 spike
文中的"spike"是写设计时在仓库之外做的验证（放在临时目录，不进仓库）。结论、实测的数字和关键的行号都已写在正文中，读者不需要那些文件。其中"spike `server.gen.go:…`"指用仓库锁定的 oapi-codegen v2.8.0 为一份试验用的接口描述生成的代码；P1 实现时以仓库中生成的代码为准，行号会不同，顺序不变。第三稿新做的 spike：请求体结构校验的两种做法（3.11）、`shared.Error` 与平台接口的草图能编译（3.11）、表结构的 CHECK 和 sqlc（4.2、4.3、3.14）、常见密码名单的重新测量（3.8）、刷新令牌的 MAC 标签（3.4）、失败闸门在并发下的计数（3.6、3.10）、`golang.org/x/time/rate` 的 `CancelAt` 能否事后退回（3.10）、账户行锁的锁模式（3.5）、`oapi-codegen/runtime` 带进的传递依赖（3.12）、格式检查与解码器的一致（3.11，用仓库锁定的 Go 1.27.1）。

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
| Plane 的表 | `users` 40 列，`profiles` 29 列，`api_tokens` 17 列，`sessions` 5 列；`instances`、`instance_admins`、`instance_configurations` 已按差异清单一 B 不保留（`tools/plane-schema/plane-v1.4.2-schema.sql`）。快照中没有任何列默认值，CHECK 只有 Django 正整数字段生成的 `>= 0` |
| 新手引导挂载时的请求 | `/onboarding` 一打开就预取工作区列表和收到的邀请（`web/apps/web/app/(all)/onboarding/page.tsx:32-45`），两者都是 M3 的旧接口；工作区的取数在 SWR 的 fetcher 中没有 `return`/`await`（`web/apps/web/app/(all)/onboarding/page.tsx:33-37`；被调用的 `fetchWorkspaces` 在 `web/apps/web/core/store/workspace/index.ts:146-158`），失败会成为未处理的 Promise 拒绝 |

---

## 2. 用户故事（M2 的验收范围）

每个故事一个 Playwright 测试文件，放在 `e2e/stories/identity/`，按总体设计 8.2 在三个层面断言。

**接口版本的约定**：
- 认证本身的故事（A1–A6、A15）发生在拿到任何令牌之前，没有 PAT 可用。它们的接口版本直接调用认证接口，调用同一组数据库断言函数。
- 其余故事的接口版本一律用 PAT。页面版本和接口版本调用同一个断言函数；结果确实因凭证种类而不同的地方，由参数表达，并写在故事里（A7）。
- 命令行和后台任务的故事（A13、A14、A16、A17）没有页面版本。

| 编号 | 故事 | 页面 | 数据库 | 接口版本 |
|---|---|---|---|---|
| A1 | 新用户注册 | 在 `/sign-up` 填邮箱、密码、确认密码，进入 `/onboarding` 的资料步骤。浏览器里没有任何 Cookie；localStorage 的 `nerve.auth` 是 `{refresh_token, login_id}` | `users` 新增一行：邮箱已转小写；`password` 以 `$argon2id$` 开头；`display_name` 是邮箱 @ 之前的部分；`is_active`。`profiles` 新增一行，各列是默认值。`auth_sessions` 新增一行：`generation = 0`，`token_hash` 等于令牌中密文部分的 SHA-256，记下 UA 和 IP，`expires_at` 约为 30 天后，未撤销 | `POST /api/v0/auth/register`，同一组断言 |
| A2 | 注册被拒绝 | 邮箱已存在：错误就地显示在表单上方，不跳转，邮箱仍在输入框里。密码不合规：字段下方显示规则。密码太常见（例如 `Password1!`、`Password1!~`）：字段下方显示"密码太常见"。关闭注册的独立 nerve（用覆盖项 `NERVE_AUTH__SIGNUP_ENABLED=false`）：页头没有"注册"链接；直接打开 `/sign-up` 提交，显示"注册已关闭"，已存在的邮箱也是这一句 | 四种情况都没有新增账户、资料和会话 | 409 `identity.email_taken`；422 `validation_failed`（`errors[].field = password`，`code` 分别是 `weak_password`、`common_password`）；403 `identity.signup_disabled`，关闭时已存在的邮箱同样是 403。"prod 在没有任何覆盖时默认关闭"由 `platform/config` 的加载测试证明（决策点 2），端到端只证明覆盖项和界面 |
| A3 | 登录与 `next_path` | 未登录打开 `/settings/profile/general?tab=x#y`，跳到登录页，地址带编码后的 `next_path`。登录后回到原地址，查询参数和片段都在。`next_path` 为 `//evil.example`、`/\evil`、`javascript:…`、带控制字符时，登录后落到默认页。错误密码和不存在的邮箱：页面显示同一句提示 | 成功时新增一行会话（`generation = 0`，`token_hash` 等于令牌中密文部分的 SHA-256）；失败时没有 | `POST /api/v0/auth/login`；两种失败都是 401 `identity.invalid_credentials`；缺少 `password` 字段是 400 `bad_request`（`errors[{field: password, code: required}]`） |
| A4 | 续期与多标签页 | 访问令牌有效期 3 秒的独立 nerve。同一个浏览器上下文开两个标签页，过期后同时操作：两边都成功，没有跳到登录页；两次续期请求在时间上不重叠。同一个测试再跑一遍，用 `addInitScript` 删掉 `navigator.locks`，走 localStorage 租约（7.1） | 会话未被撤销；`generation` 等于续期次数；`token_hash` 等于最新令牌的密文的哈希；`last_refreshed_at` 已更新；`expires_at` 不变（绝对期限，3.5） | 每次用**上一次返回的**刷新令牌续期：每次返回新的一对令牌，`generation` 递增，`refresh_token_expires_at` 不变。旧令牌的重复使用只在 A5 测 |
| A5 | 刷新令牌被重复使用 | 测试从 localStorage 取出刷新令牌，先在接口上用它续期一次（模拟被盗）。页面下一次续期时，会话被作废，跳到登录页，`next_path` 是当前地址 | `revoked_at` 已填，`revoke_reason = 'reuse_detected'` | 纯接口复现：真实的旧令牌再用一次得到 401 `identity.refresh_token_invalid`，会话被撤销；之后这个会话的访问令牌和最新的刷新令牌也都是 401。伪造的旧代（会话 id 和代数是真的，密文和标签是随机的）：401，会话不变 |
| A6 | 退出与切换账户 | 用户菜单点"退出"，回到登录页，localStorage 里没有 `nerve.auth`。同一上下文的另一个标签页也回到登录页（`storage` 事件）。**切换账户**（两种走法各一次）：① 标签页甲以账户 X 登录并保持登录；测试在标签页乙中通过接口登录 Y，按令牌管理器的写入路径换上 Y 的记录（新的 `login_id`），中间不退出。标签页甲收到 `storage` 事件，丢掉内存中的访问令牌和 stores，重新取 `/me`，显示 Y；它此后发出的写请求都是 Y 的，页面不会仍显示 X。② 标签页乙先退出（甲随之回到登录页），再以 Y 登录：甲收到"记录出现"的事件，取 `/me`，以 Y 进入 | `revoke_reason = 'logout'`；这个会话的访问令牌在下一个请求就得到 401 | `POST /api/v0/auth/logout`，同一组断言。用上一代刷新令牌退出：204，会话不变（3.5） |
| A7 | 修改密码 | 在安全页输入当前密码和新密码，成功提示，页面保持登录。当前密码错误：字段错误，数据库不变。安全页列出账户的 PAT，并说明修改密码不会撤销它们 | 共用断言 `assertPasswordChanged(user, {survivingSession})`：`users.password` 已改变；除 `survivingSession` 外的会话 `revoke_reason = 'password_changed'`；`api_tokens` 不变；旧密码登录 401，新密码 200。页面版本传入当前会话 | PAT 调用 `POST /api/v0/me/change-password`，同一个断言，`survivingSession` 为空：PAT 没有"当前会话"，全部会话都被撤销；PAT 本身继续可用。这是有理由的差异：页面保留正在操作的那一处登录 |
| A8 | 修改资料 | 在 general 页改名、姓和显示名，在偏好页改时区；刷新页面后仍是新值 | `users` 对应列；`updated_at` 已更新 | PAT 调用 `PATCH /api/v0/me`；`{"first_name": null}` 是 400 `bad_request`（`code: invalid_format`），数据库不变 |
| A9 | 修改偏好 | 主题、语言、每周第一天；刷新页面后仍生效；主题下拉框在按钮旁展开（中英文各一次，9.6） | `profiles` 对应列 | PAT 调用 `PATCH /api/v0/me/profile` |
| A10 | 新手引导的资料步骤 | 新注册的用户第一次打开 `/onboarding`：没有失败的接口请求，没有发往 M3 旧接口的请求，没有未处理的 Promise 拒绝，控制台没有错误。填名字，进入下一步；创建或加入工作区的界面出现即可，M3 之前不提交 | `users.first_name`；`profiles.onboarding_step` 中 `profile_complete = true`，其余三项仍是 `false` | PAT 调用 `PATCH /api/v0/me`，以及只带一个键的 `PATCH /api/v0/me/profile {onboarding_step: {profile_complete: true}}`：其余三个键不变；未知的键（`{onboarding_step: {profile_completed: true}}`）是 400 |
| A11 | PAT 的创建、使用和撤销 | 创建（名称、说明、有效期 1 周），令牌只显示一次；列表中出现，但没有令牌原文；撤销后从列表消失 | `token_hash` 等于令牌的 SHA-256，表中任何一列都不含令牌原文；`expired_at` 约为 7 天后。用这个 PAT 调用 `GET /api/v0/me` 成功，`last_used` 被写入。撤销后 `deleted_at` 已填，再用它得到 401。把另一个 PAT 的 `expired_at` 改到过去，也得到 401 | 用另一个 PAT 创建、列出（翻页）、撤销（`DELETE /api/v0/api-tokens/{token_id}`） |
| A12 | 停用账户（决策点 3，已裁定为 A） | 在 general 页点"停用账户"，确认弹窗，回到登录页。再用原密码登录：页面显示"账户已停用" | `users.is_active = false`，`password` 不变；全部会话 `revoke_reason = 'deactivated'`；`profiles` 的 `onboarding_step` 四个键都是 `false`，`is_onboarded`、`is_tour_completed` 为 `false`，`last_workspace_id` 为空；`api_tokens` 不变（`deleted_at` 为空） | PAT 调用 `POST /api/v0/me/deactivate`：204，同一组断言；同一个 PAT 再调用 `GET /me` 得到 401；原密码登录 403 `identity.account_deactivated`。然后 `nerve users activate --email …`：输出里有重新可用的 PAT 数；这个 PAT 调用 `GET /me` 200，原密码登录 200。`nerve users deactivate --email …` 得到与自助停用相同的数据库结果 |
| A13 | 管理员重置密码 | — | `nerve users reset-password --email …` 从标准输入读新密码：`users.password` 改变；全部会话 `revoke_reason = 'password_reset'`；全部 PAT 的 `deleted_at` 已填；输出一行，带撤销的会话数和 PAT 数 | 用接口核对：新密码能登录；旧会话的刷新令牌和旧 PAT 都得到 401 |
| A14 | 过期会话被清理 | — | 把一行会话的 `expires_at` 改到过去，限时轮询，直到这一行被删除；未过期的会话仍在 | — |
| A15 | 登录限流 | 限流很低的独立 nerve：同一 IP + 邮箱失败到上限后，页面显示"尝试次数过多"；换一个邮箱仍能尝试，直到同一 IP 的总次数到达按 IP 的上限 | 没有新增会话 | 两个桶各触发一次：429 `rate_limited`，带 `Retry-After`。被 `login_ip_email` 拒绝的请求不消耗 `login_ip`（3.10） |
| A16 | 管理员修改登录邮箱（决策点 1，已裁定为 B） | — | `nerve users set-email --email <旧> --new-email <新，含大写>`：`users.email` 变为规范化后的新邮箱；该账户全部会话 `revoke_reason = 'email_changed'`；PAT 不变；输出一行，带撤销的会话数。新邮箱已被别的账户使用：退出码 1，输出说明，数据库不变 | 用接口核对：旧邮箱登录 401，新邮箱登录 200；旧会话的刷新令牌 401；PAT 调用 `GET /api/v0/me` 返回新邮箱 |
| A17 | 管理员创建账户（决策点 2，已裁定为 C） | — | `nerve users create --email <邮箱，含大写>` 从标准输入读密码：`users` 新增一行，邮箱已规范化；`profiles` 新增一行；没有会话；输出一行。邮箱已被使用、密码不合规：退出码 1，输出说明，数据库不变 | 用接口核对：新账户能登录 |

**冒烟故事的更新**：
- S1：加上"迁移版本正确"：`nerve migrate status` 的每一行都是 `applied`，`goose_db_version` 的最新版本等于最后一个迁移文件（M0-P6 交接）。
- S2：登录页没有失败的接口请求（没有刷新令牌时不请求 `/me`，7.1），控制台没有 CSP 违规（8.3）。
- S3：`toEqual` 加上三个新字段的期望值（5.3）。

---

## 3. 设计裁定

以下裁定都在总体设计定下的范围内；改写已批准规则的地方列在 3.20，按 Phase 同步到上级文档。3.3、3.5 的账户行锁、3.6、3.10–3.15 是后续所有 M 都要照做的平台约定。

### 3.1 范围边界：登录之后看到什么
工作区接口在 M3 才有，M2 不为它造假数据，也不为它临时改变跳转规则。**规则：M2 能到达的页面，挂载时不向 M3 及以后的旧接口发任何请求。** 只有用户自己提交一个 M3 的表单时才会遇到 404，这是已知的过渡。

- **M2 能到达的页面**：登录、注册、`/onboarding`、`/create-workspace`、`/settings/profile/*`。
- **没有完成新手引导的用户**：进入 `/onboarding`。
  - 资料步骤完全可用（A10）。
  - P4 删掉页面挂载时对工作区列表和邀请的预取（`web/apps/web/app/(all)/onboarding/page.tsx:32-45`）。`OnboardingRoot` 的 `invitations` 取默认的空数组；M3 在新接口上加回取数（13.2）。
  - 下一步是创建或加入工作区。界面照常出现；用户提交时调用的是 Plane 的旧接口，在 M3 之前得到 404 problem，页面按已有的错误处理显示失败。
- **已完成新手引导的用户**：没有 `next_path` 时直接落到 `/create-workspace`。M2 不取"上次的工作区"和工作区列表，M3 加回落点数据（13.2）。表单能打开，提交同样得到 404，直到 M3。
- **个人设置** `/settings/profile/*` 不在工作区之下，四个标签页在 M2 全部可用。
- **需要"已完成引导"的用户的测试**，由 fixture 通过接口 `PATCH /api/v0/me/profile {is_onboarded: true}` 准备。这符合总体设计 8.2："前置数据通过接口准备，只有被测的那一步走页面"。
- **登录不再依赖工作区数据**：
  - 现在 `fetchCurrentUser` 先取 `/api/users/me/`，再用 `Promise.all` 同时取三样：资料、`/api/users/me/settings/`、工作区列表（`web/apps/web/core/store/user/index.ts:112-118`）。其中任何一个失败，登录就失败。
  - M2 让用户 store 只取 M2 的数据（`/me`、`/me/profile`）。
- **核对**：A10 和 P4 的浏览器核对覆盖第一次打开 `/onboarding` 和登录后的落点：没有失败的接口请求，没有未处理的 Promise 拒绝，控制台没有错误（9.6）。

### 3.2 后续 M 的字段何时进入接口
前端很多地方读 M2 的类型，但字段背后的数据属于后面的 M。规则有三条，M3 起同样适用：

1. **数据在本 M 能真实产生的字段**，进入本 M 的接口。
2. **本身就可以为空的引用字段**（头像、封面、图标这类"可能没有"的东西），随它所属的实体一起进入接口。在产生它的 M 到来之前，它的值是 `null`。
   - 这是真实的值（确实还没有头像），不是占位：接口描述里它本来就可以为空，调用方本来就要处理 `null`。
   - 显示它的代码保留，它们已经处理"没有图片"的情况。
   - 只删掉在本 M 无法工作的上传控件（它们调用 Plane 的上传接口，在 Nerve 中必然失败），由产生它的 M 按新的上传协议加回。
   - 只适用于本来就允许为空的引用，不扩展到普通字段。
3. **其他属于后面 M 的字段**，本 M 不定义，删掉前端的读取。不为它返回恒为 `null` 的占位。

| 字段 | 规则 | 处理 |
|---|---|---|
| `avatar_url`、`cover_image_url`（`User`） | 2 | M2 定义为可为 `null` 的字符串，恒为 `null`。显示当前账户头像的 6 个文件、读封面的 2 个文件不改。删掉 general 页的头像上传弹窗和封面选择器、新手引导资料步骤的头像上传。M5 加入 `users.avatar_asset_id`、`cover_image_asset_id`（迁移的归属见 3.14），接口返回签名地址（总体设计 4.3），按新的 `{method, url, headers}` 协议加回上传控件（前端改动清单 3.2） |
| `last_workspace_id`（`Profile`） | 1 | M2 保留。它是客户端写入的 uuid，数据库不设外键，和 Plane 一样（Plane `serializers/user.py:90-138` 在读取时判断是不是成员）。保留它可以避免 M3 领域的 4 个文件（`invitations/page.tsx`、`create-workspace/page.tsx`、新手引导的 `create.tsx`、`workspace-menu-root.tsx`）先删后加。是否补外键由 M3 决定 |
| 实例配置的 `workspace_creation_enabled`、`file_size_limit` | 1 | M2 定义，取自配置文件。M1 设计 3.7 已约定"实例配置接口的最终字段在 M2 定义"。服务端的执行分别由 M3（创建工作区）、M5（上传）负责，写进交接 |

- **M3 照此处理**：工作区图标（web 中 12 个文件读 `logo_url`）、项目封面（10 个文件读 `cover_image_url`）、成员头像（46 个文件读 `avatar_url`，大多是 `IUserLite` 的成员头像）。它们随工作区、项目、成员一起进入接口，M5 之前为 `null`，读取它们的代码不动。
- **为什么改了第一稿的规则**：第一稿让这类字段"由产生它的 M 加入，本 M 删掉读取"。按它，M3 要删掉约 60 个文件的读取，M5 再加回来，白白改两遍。

### 3.3 模块划分与端口的位置
- **`identity`**：一个模块，按总体设计 6.2 负责账户、资料与偏好、会话、PAT、密码。它的用例包括注册、登录、续期、退出、认证、资料、偏好、修改密码、自助停用、PAT，命令行的创建账户、重置密码、修改邮箱、停用和恢复，以及清理过期会话（定时任务）。
  - 不再拆出 `user` 或 `auth` 模块：资料和凭证共用 `users` 表，修改密码要同时改账户和会话，拆开以后这些操作都成了跨模块调用。
- **`instance`**：在 M0 的试点上扩展：三个配置字段（3.2、5.3）和时区列表。
- **`internal/shared`**：M2 第一次建立（M0 设计 3.1 预计在 M2）。只放**值会跨越模块边界**的东西，只依赖标准库：
  - `Actor`（当前账户，3.6）：由认证放进 `context`，每个模块的 handler 都要读；
  - 领域错误 `Error` 和它的种类（3.11）：每个模块都返回它，由同一处映射为 problem；
  - `TxManager` 端口：同一个事务经 `context` 穿过多个模块的仓储（总体设计 6.4、M0-P2 交接 1）；
  - 分页游标的**封套**：版本和编码，以及解码失败时的错误。游标里装什么由每个列表自己定义（3.12）。
  - 一个包，按文件分；不建子包。`Authorizer` 端口由 M3 加入。
- **其余端口由使用方在自己的 `app` 层声明**（M0 试点 `instance/app/ports.go` 的写法）：
  - 包括时钟：`identity/app` 声明自己的 `Clock { Now() time.Time }`，`platform/clock` 的实现按结构满足它。一个方法的接口在每个模块各写一遍，比放进 `shared` 更符合"接口由使用方定义"（总体设计 6.1），时钟的值也不跨越模块。
  - 总体设计 6.2 写的是端口放在 `domain`。M0 起实际放在 `app`：用例是端口的使用方，`domain` 只有纯规则、没有 I/O。P1 把 6.2 改成现在的做法（3.20）。
- **平台不导入 `internal/shared`**（控制者裁定，保持总体设计 6.2"平台与业务无关"）：
  - `httpserver` 声明自己需要的小接口：认证器（3.6）和"可以映射为 problem 的错误"（3.11）。`identity`、`shared` 按结构满足它们。
  - `platform/postgres` 的事务管理器按结构满足 `shared.TxManager`，不导入 `shared`；`bootstrap` 里写一行编译期断言。
  - 架构测试规则 4 加上这一条："平台不导入模块、`bootstrap` 和 `internal/shared`"。
- **模块入口**（`module.go`）导出：`New(Deps)`、`Register(router, api)`、`PublicOperations()`（3.6）、`Authenticator()`、`Jobs()`、`Admin()`。
  - `Admin()` 返回命令行要调用的用例（创建账户、重置密码、修改邮箱、停用、恢复），命令的参数解析在 `cmd/nerve`，组合在 `bootstrap`（3.17）。
  - 请求体的结构表是模块 HTTP 适配器的生成代码，由适配器在 `Register` 中交给平台（3.11），不经过模块入口。

### 3.4 令牌格式
所有请求都用 `Authorization: Bearer <token>`。服务端按前缀区分令牌：`nrv_pat_` 开头的是 PAT；其余按 JWT 解析；把刷新令牌当作 Bearer 发来，得到 401。

| 令牌 | 形式 | 内容 | 服务端存什么 |
|---|---|---|---|
| 访问令牌 | JWT，头部 `{"alg":"EdDSA","typ":"JWT"}` | 只有 `sub`（用户 id）、`sid`（会话 id）、`exp`，没有任何权限信息（总体设计 4.1） | 不存 |
| 刷新令牌 | `nrv_rt_` + base64url（不补 `=`）编码的 68 字节：会话 id 16 字节 + 代数 4 字节（大端）+ 随机密文 32 字节 + 标签 16 字节；base64url 部分 91 个字符，整个令牌（`nrv_rt_` 加 91）98 个字符 | 对客户端是不透明的字符串 | 只存当前一代密文的 SHA-256（`auth_sessions.token_hash`，4.5） |
| PAT | `nrv_pat_` + base64url 编码的 32 字节随机数，base64url 部分 43 个字符，整个令牌 `nrv_pat_` + 43 = 51 个字符 | — | 只存整个令牌的 SHA-256（`api_tokens.token_hash`） |

- **JWT 库**：用 `github.com/golang-jwt/jwt/v5` v5.3.1。
  - 它没有任何传递依赖；解析时用 `WithValidMethods([]string{"EdDSA"})`、`WithExpirationRequired()`、`WithStrictDecoding()`。
  - 自己手写约 100 行也行（还能用 RFC 9864 的 `Ed25519` 算法名），但要自己写、自己测严格的 base64url 解码、算法固定和声明校验。v0 只有 nerve 自己验签，算法名是 `EdDSA` 还是 `Ed25519` 没有外部影响。
- **刷新令牌的标签**（Codex I-2 的选项②，3.5）：
  - `标签 = HMAC-SHA256(K, 会话 id ‖ 代数 ‖ 密文)` 的前 16 字节。
  - `K = HKDF-SHA256(签名私钥的种子, info = "nerve refresh-token mac v1")`，32 字节：从 3.7 的 Ed25519 密钥派生，不新增配置项和密钥文件；`info` 把它和签名的用途分开。
  - 标签让服务端不存历史也能认出"这个会话真的发过的某一代"。它由持有密钥材料的适配器计算（`identity/adapter/signing`，6.2），`identity/app` 声明一个小端口；`domain` 只管 68 字节的布局。比较用 `hmac.Equal`（常数时间）。
  - spike（Go 标准库的 `crypto/hkdf`、`crypto/hmac`）：令牌的 base64url 部分 91 个字符，整个令牌 98 个字符，符合 8.6 的正则；改动 68 字节中的任何一个，标签都不再成立；随机的标签不成立；换一把签名密钥后，旧令牌的标签不成立；派生出的 K 与种子不同。
- **随机数**：用 `crypto/rand`，不做成端口。Go 1.24 起它不会返回错误；测试只核对格式和唯一性，不需要控制具体的值。
- **有效期**：访问令牌 15 分钟（`auth.access_token_ttl`）；会话从登录起 30 天（`auth.session_ttl`），刷新令牌随会话一起到期（3.5）。

### 3.5 会话、轮换和重复使用检测
- **一次登录一行 `auth_sessions`**。行的 id 就是访问令牌里的 `sid`。`generation` 是当前的代数，`token_hash` 是当前这一代密文的 SHA-256。每个会话只有这一行，大小固定。
- **绝对期限**：登录时 `expires_at = 登录时刻 + auth.session_ttl`（默认 30 天），之后**永不延长**。
  - 续期只换令牌，不动 `expires_at`。30 天后无论是否活跃，都要重新登录。
  - 理由：这是总体设计 4.1"30 天"的字面意思；它给被盗的刷新令牌定了一个上限（PAT 另算，见 8.5）。第一稿每次续期都把期限推后 30 天，一个每月至少用一次的会话永不结束，而且没有登记这个差异。
  - Plane 的会话固定 7 天，请求不会延长它（`SESSION_COOKIE_AGE = 604800`，`SESSION_SAVE_EVERY_REQUEST` 默认关闭，`plane/apps/api/plane/settings/common.py:373-376`）。Nerve 是 30 天，登记为差异（4.6）。
- **业务判断用的时间都来自用例的时钟**：用例从 `Clock` 取当前时刻，作为参数传给 SQL（`expires_at > $now`），不在 SQL 中写 `now()`。审计列也由同一个时钟写入（3.13）。
- **续期**（一个事务）：
  1. 客户端交出刷新令牌。服务端解出会话 id、代数 g、密文和标签，不查库；格式不对直接 401。
  2. 按主键读出会话行。
  3. 按下表处理：

     | 情况 | 结果 |
     |---|---|
     | g 等于当前代数，`SHA-256(密文) = token_hash`，未撤销、未过期 | 轮换：`UPDATE auth_sessions SET generation = g + 1, token_hash = 新密文的哈希, last_refreshed_at = $now, updated_at = $now WHERE id = $sid AND generation = $g AND token_hash = $h AND revoked_at IS NULL AND expires_at > $now`，返回新的一对令牌。这一支不看标签 |
     | g 小于当前代数，标签成立（3.4），未撤销、未过期 | 这个会话真的发过的旧令牌被再次使用：撤销会话（`revoke_reason = 'reuse_detected'`），记一条 WARN 日志，返回 401 `identity.refresh_token_invalid` |
     | 其他：会话不存在；g 等于当前代数但哈希不符；g 大于当前代数；g 较旧但标签不成立；会话已撤销或已过期 | 401（码同上），**不撤销** |

  - 条件 `UPDATE` 没有命中时（并发的续期抢先轮换了，或并发的重置撤销了会话），在同一个事务里重读这一行，再按表处理。两个请求用同一个令牌同时续期，后到的那个看到的 g 已是旧代，而它的标签成立，判为重复使用；同一浏览器内由跨标签页的协调避免这种情况（7.1）。
  - **当前一代为什么不看标签**：存的哈希已经证明了它；而且换签名密钥时，正在用的令牌不会因此失效，不会让所有人被迫重新登录。
  - **为什么要确认旧令牌是真的**（Codex I-2）：第二稿只凭"g 小于当前代数"就撤销。会话 id 就写在访问令牌里，拿到一个早已过期的访问令牌的人，拼上 `g = 0` 和任意密文就能让会话下线。现在要撤销，必须交出这个会话真的发过的某一代令牌：只有服务端能算出它的标签。
  - **为什么用标签，不存历史**（Codex I-2 先取选项①，复核后改为选项②）：
    - 选项①给每一代存一行哈希（第三稿的 `auth_refresh_tokens`）。去掉按会话计数的续期桶以后（3.10），轮换只受按 IP 的 `anonymous` 约束（每分钟 600 次）：一个 IP 可以让它自己的会话在 30 天里积累约 2,600 万行、约 2.6 GB。第三稿写的"每个会话最多约 2,900 行"不成立。
    - 两种补救都不行：限制最短的轮换间隔，会误伤多个标签页（每个标签页有自己的内存访问令牌、各自续期，同时打开的标签页相隔几秒就会续期）；只留最近 K 代，攻击者快速轮换 K 次就能逃过检测。
    - 标签用密码学证明真实性，每个会话只存一行（O(1)）。
  - **换签名密钥之后**（3.7）：换钥之前签发的旧代令牌，标签无法验证，按伪造处理（401，不撤销）；当前一代不受影响。dev 的临时密钥在重启后同理。重复使用检测只对换钥之前的旧令牌有这个缺口（§16）。
- **服务端期限**（控制者复核 N2）：续期和退出有自己的服务端期限 `auth.refresh_deadline`（默认 4 秒）。
  - 顺序：服务端的语句期限 4 秒 + 提交的期限 2 秒（3.6）< 客户端超时 8 秒 < 租约 10 秒（7.1）。客户端放弃之前，服务端已经结束这次续期（提交、回滚，或者提交没有回应，见下面第 4 种情况），客户端不会在服务端仍可能提交时换一个标签页用旧令牌重试。启动校验这个不等式（6.5）。
  - 放在 `identity` 的 HTTP 适配器：handler 调用用例之前 `context.WithTimeout(ctx, refreshDeadline)`。这个期限是轮换协议的一部分（它必须短于前端的超时），不是传输层的策略；平台的请求期限对一个模块的全部操作一视同仁（m7）；`context` 的期限只能缩短，在 15 秒之内再缩短到 4 秒，适配器自己就能做到。
  - 需要**放宽**期限或请求体上限的是传输层的事（M5 的上传），由 M5 在平台加按操作的设置（13.2），与这里不重复。
  - 提交不受这个期限的取消（3.6 的全局规则）：语句在 4 秒内做完的轮换，不会因为这个期限在提交时被取消成 500。
  - 剩下的四种情况，结果都是一次重复使用检测，用户重新登录：
    1. 请求这一段：请求在网络上走得太久，到达服务端时客户端的 8 秒已所剩无几；服务端按时提交，响应回来时客户端已经放弃；
    2. 响应这一段：提交成功，响应在网络上走得太久或丢失；
    3. 持有租约的标签页被浏览器冻结，超过了租约期（7.1）；
    4. 提交的结果未知：`COMMIT` 已经发出，在 `database.commit_timeout` 内没有回应（3.6）。服务端答 500，客户端保留旧令牌、退避后重试；轮换其实已经提交时，这次重试交出的是旧代，判为重复使用。
- **已知代价**：上面四种情况是严格检测的固有代价。
- **退出**：只有**当前这一代、有效**的刷新令牌能撤销会话（`revoke_reason = 'logout'`），判定与续期表的第一行相同。
  - 未知、已过期、已撤销、上一代、伪造的令牌：204，什么都不做，不泄露它的状态，也不触发重复使用检测。重复使用只在续期时判定。
  - 前端退出时和续期共用同一把锁，所以交出的总是最新一代（7.1）。
- **撤销规则**：

  | 事件 | 撤销的会话 | 撤销的 PAT | `revoke_reason` |
  |---|---|---|---|
  | 退出 | 只有这一个会话（和 Plane 一样；负责人已批准，11.1） | 无 | `logout` |
  | 修改密码 | 除当前会话外的全部；用 PAT 修改时没有当前会话，全部撤销 | 无（和 Plane 一样；安全页列出 PAT，由用户决定） | `password_changed` |
  | 管理员重置密码 | 全部 | **全部**（和 Plane 不同） | `password_reset` |
  | 管理员修改邮箱 | 全部 | 无 | `email_changed` |
  | 停用账户（自助或管理员） | 全部 | 不删除。认证要求账户未停用，所以停用期间同样失效；`activate` 之后重新可用 | `deactivated` |
  | 重复使用 | 这个会话 | 无 | `reuse_detected` |

  - **管理员重置密码为什么撤销 PAT**：v0 没有邮件，重置是账户被盗后唯一的恢复手段（总体设计 4.2）。PAT 可以创建永不过期的新 PAT，只撤销会话赶不走持有 PAT 的攻击者（8.5）。命令输出撤销的会话数和 PAT 数，登记为差异（Plane 的 `reset_password` 只改密码，`plane/apps/api/plane/db/management/commands/reset_password.py:56-64`）。
  - **修改邮箱、停用不撤销 PAT**：两者都不是账户被盗后的恢复手段。怀疑账户被盗时，另外执行 `reset-password`；命令的说明和 README 写明（3.17）。
  - **用户自己修改密码不撤销 PAT**：和 Plane 相同。用户可能正在用 PAT 跑脚本，修改密码不该让它们突然失效；安全页在修改密码的表单下列出 PAT，需要时就地撤销（7.7）。
  - **创建 PAT 不要求输入密码**：和 Plane 相同。它带来的风险和恢复办法见 8.5。
- **每个请求都查一次数据库**：验签通过后，用一条按主键的查询确认"会话未撤销、未过期，账户未停用"。
  - 这样退出、修改密码、停用在下一个请求就生效，不用等访问令牌过期。
  - 总体设计 4.2 本来就要求每个请求从数据库读取成员关系；多一次主键查询的代价可以接受。
  - PAT 同理：按 `token_hash` 查询，要求未删除、未过期、账户未停用。
  - `last_used` 最多每分钟写一次（`UPDATE … WHERE last_used IS NULL OR last_used < $now - interval '1 minute'`）。Plane 每个请求都写一次（`plane/apps/api/plane/api/middleware/api_authentication.py:41-42`）；Agent 高频调用时，每次都写太重。登记为行为差异。
- **清理**：`expires_at` 已过的会话由 River 定时任务删除（3.15）。已撤销的行保留到它原本的过期时间，便于排查问题。

**凭证的签发与变更：账户行锁**（Codex I-1，后续 M 凡是签发或变更凭证的写入都照做）
- **问题**：登录先读出哈希，在事务外做 argon2 校验，再插入会话。管理员重置密码如果恰好在"校验"和"插入"之间提交，新会话就逃过了重置的撤销；登录时的重新哈希还可能把旧密码的哈希写回去。创建 PAT 同理：认证发生在用例之前，重置提交之后插入的 PAT 逃过了批量撤销。重置是账户被盗后唯一的恢复手段，它必须赶走所有凭证。
- **锁**：`SELECT … FROM users WHERE id = $1 FOR NO KEY UPDATE`，下文称"锁账户行"。
  - 它与自己、与 `FOR UPDATE` 冲突，所以同一个账户上签发和变更凭证的事务一个接一个执行。
  - 它不与外键检查取的 `FOR KEY SHARE` 冲突，所以别的事务插入引用这个账户的行（会话、PAT、资料）不会被它挡住。`FOR UPDATE` 会挡住它们。
  - spike（开发库）：一个事务持有 `FOR NO KEY UPDATE` 时，另一个事务插入引用这一行的记录，不到 1 毫秒完成；换成 `FOR UPDATE`，插入一直等到前者提交（2.4 秒）。
  - 改唯一列 `email` 的 `UPDATE`（`set-email`）在 Postgres 中算作改键，这条语句自己就把行锁升级为 `FOR UPDATE`；spike 中外键插入等到它提交为止。只改 `password` 这类非键列时，外键插入不受影响。`set-email` 只有几条语句，这样的等待无害。
- **协议**：

  | 操作 | 做法 |
  |---|---|
  | 用密码签发：登录（以及登录时的重新哈希） | 读出账户，记下哈希的快照 → 事务外做 argon2 校验；需要重新哈希时，新哈希也在事务外算好 → 事务：锁账户行；哈希仍等于快照（不等时见下）；账户未停用，否则 403 `identity.account_deactivated`；需要时写回新哈希；插入会话 |
  | 用凭证签发或变更：创建 PAT、修改密码、自助停用 | 事务：先锁账户行，再确认调用者的凭证仍然有效（会话未撤销、未过期；PAT 未删除、未过期）、账户未停用，否则 401 `unauthorized`；然后写入。修改密码的两次 argon2（校验当前密码、哈希新密码）在事务外做，事务内核对哈希仍等于快照（不等时见下） |
  | 管理员的变更：`reset-password`、`set-email`、`deactivate`、`activate` | 事务：先锁账户行，再改写、撤销 |
  | 续期 | 不锁账户行。会话行上的条件 `UPDATE` 在拿到行锁后重新求值，重置先提交时 `revoked_at IS NULL` 不再成立，续期失败 |

  注册和 `nerve users create` 插入的是新账户，没有可以竞争的旧凭证，靠邮箱的唯一约束。
- **哈希与快照不等时**：可能只是并发的登录做了重新哈希（密码没变，参数变了），也可能密码真的被改了。所以在事务外用新的哈希再校验一次密码：成立，就以新的哈希为快照，把锁内那一步重做一次；不成立，登录答 401 `identity.invalid_credentials`，修改密码答 422 `identity.current_password_incorrect`；重做时哈希又变了，同样失败。只在这种竞争中多做一次 argon2。
- **登录与 `set-email` 并发**：登录按旧邮箱找到账户、校验了正确的密码，`set-email` 在这之间提交。登录的插入等 `set-email` 提交后照常完成（哈希未变、账户未停用），新会话不在被撤销之列。这无害：签发给的是知道密码的人，与他稍后用新邮箱登录得到的会话相同；`set-email` 撤销会话是为了让大家改用新邮箱登录，不是账户被盗后的恢复手段（3.17）。
- **加锁顺序**：`users` → `profiles` → `auth_sessions` → `api_tokens`。所有事务按这个顺序取锁，不会互相等成环。
  - 外键检查（插入或改动引用 `users` 的行）对那一行 `users` 取 `FOR KEY SHARE`，按顺序规则算作一次 `users` 的锁：一个事务在锁住后三张表的行以后，不再插入引用另一个账户的行。M2 的事务都是先锁自己的账户行，再插入引用它的行。
  - `profiles` 的位置是这个全局顺序的一部分：同时写资料和会话的事务（M2 中是停用）先写资料、后撤销会话。顺序决定写入的先后，不是反过来由某个用例决定顺序。
  - 清理任务删会话时用 `FOR UPDATE SKIP LOCKED` 分批取行，跳过正被别的事务锁住的会话，下一轮再删（3.15）。
- **argon2 始终在锁外**：持锁的时间只有几条语句，锁不会因为哈希排队。
- **测试**（P3 的完成线）：在真实数据库上确定性地交错（`pgtest`），用会阻塞的假哈希器或钩子端口卡住一方，不只测顺序执行：
  1. 登录校验完成 → 重置提交 → 登录的插入失败（401），没有新会话；
  2. 登录的重新哈希不会覆盖重置写入的哈希；
  3. 用被并发重置撤销的凭证创建 PAT，失败，没有新 PAT；
  4. 修改密码与登录交错：登录用的是旧密码时失败；
  5. 两个登录交错，其中一个做了重新哈希：另一个重新校验一次后成功；
  6. 一个事务锁着账户行时，别的事务插入引用它的会话不必等待。

### 3.6 认证：默认拒绝
- **规则**：`/api/v0` 下的每个操作默认需要有效的令牌，只有模块声明为公开的操作例外。
  - Plane 同样默认要求登录（`DEFAULT_PERMISSION_CLASSES = IsAuthenticated`，`plane/apps/api/plane/settings/common.py:145`）。
  - 第一稿是"没有令牌就放行，由 handler 决定"，比它弱，而且整程序测试覆盖不到接口描述之外的路由和误标为公开的操作（评审 I1），已改掉。
- **公开操作由模块声明**：模块入口导出 `PublicOperations() []string`，内容是生成代码注册路由时用的模式字符串，例如 `"POST /api/v0/auth/login"`（spike `server.gen.go:264-267`）。
  - M2 的公开操作：`identity` 的 `register`、`login`、`refreshTokens`、`logout`；`instance` 的 `getInstance`、`listTimezones`。
- **中间件**：在 `platform/httpserver`，挂在生成代码的 `Middlewares` 里（按路由，M0-P2 交接 2）。它看到的请求已经由路由匹配过，`r.Pattern` 就是注册时的模式（Go 1.23 起 `ServeMux` 会填这个字段）。

  | 操作 | 没有令牌 | 带了令牌，本 IP 的失败闸门已空 | 带了令牌，认证失败 | 令牌有效 |
  |---|---|---|---|---|
  | 公开 | 放行 | 放行，不看令牌 | 放行，不认证 | 放行，不认证 |
  | 其他 | 401 `unauthorized` | 429 `rate_limited`，**不调用认证器** | 401 `unauthorized`；预留的一个单位留下（计数），签名有效而只是过期的 JWT 除外 | 认证器返回的 `context` 带着当前账户，交给下一层；预留的单位退回 |

  - 401 都带 `WWW-Authenticate: Bearer`；令牌无效时再加 `error="invalid_token"`。失败的原因只进 DEBUG 日志。
  - 公开操作不看 `Authorization`：页面在登录状态下读实例配置，不会因为访问令牌刚好过期而多一次 401。
- **认证之前的失败闸门**（Codex I-3）：
  - 第二稿让无效的令牌"计入按 IP 的桶"，但计数发生在认证之后：桶空了以后，每个请求仍然先验签或查一次数据库，429 限制不了这部分工作。
  - 现在由按 IP 的 `auth_failure` 桶（3.10）在认证之前把关：非公开操作带了令牌时，**先预留一个单位**，拿不到就直接 429，不调用认证器。认证失败时这个单位留下（计数），再返回 401；认证成功、或者认证器返回的不是"未认证"的错误时，把它退回。
  - **为什么先预留**（控制者复核 R5）：先查余额、失败后再扣，并发的请求都会在余额只剩 1 时通过检查，失败的次数可以超出额度任意多。先预留，同时在认证中的请求也占着额度，失败的次数不会超过额度。spike：额度 3，50 个并发的无效令牌，认证器只被调用 3 次，其余 47 个得到 429。
  - **预留的代价**：认证期间每个请求占着一个单位，同一个 IP 同时在认证中的请求超过桶里当时剩下的单位时，多出的得到 429。桶满时上限是突发（60）；已有失败扣掉一部分时，上限随之变小（spike：额度 3、认证 20 毫秒时，50 个并发的有效请求只有 3 个通过）。认证是一次验签加一次按主键的查询，正常负载下同一 IP 远到不了 60 个并发；数据库变慢时会更早碰到，§16 登记。
  - **计数的是什么**：不是真的、或已被撤销的凭证。也就是认证器返回的"未认证"错误（满足 3.11 的 `ProblemError`，状态为 401），包括：JWT 签名不对或格式不对；JWT 的会话不存在、已撤销或已过期；PAT 不存在、已撤销或已过期；账户已停用。
  - **不计数**：签名有效、只是 `exp` 已过的 JWT。它是协议里正常的续期信号（7.1），在查库之前就判定，不花数据库的工作；计入的话，共享出口 IP 后面一批标签页同时醒来，就会得到 429。认证器对它返回的 401 错误另外实现 `ExpiredCredential() bool`（`httpserver` 声明的可选接口），中间件据此退回单位。认证器返回的其他错误（例如数据库不可用）按 3.11 映射为 problem 或 500，同样退回。
  - **代价**：同一个出口 IP 后面的攻击流量会把额度用完，这个 IP 后面有效的调用方也会得到 429，直到额度恢复（每分钟 60 次，3.10）。这是按 IP 计数的固有代价；可信代理的配置和 IPv6 的前缀长度决定"IP"是谁（3.10）。
  - 预留和退回与 `AllowAll` 用同一套机制（3.10 的 `platform/ratelimit`）。
  - 测试：用一个计数的假认证器，超过额度以后调用次数不再增加；并发时也不超过额度；过期的 JWT、数据库错误、成功都退回单位。
- **认证器**（`httpserver` 声明，`identity` 实现，`bootstrap` 接上，M0-P3 交接 4）：
  ```go
  type Authenticator interface {
      // 成功时返回带当前账户的 context，以及这个凭证的限流键。
      // 令牌无效时返回 ProblemStatus() 为 401 的错误；签名有效而只是过期的 JWT，
      // 这个错误另外实现 ExpiredCredential() bool 并返回 true。其他错误是内部故障。
      Authenticate(ctx context.Context, token string) (context.Context, string, error)
  }
  ```
  - `identity` 的实现用 `shared.WithActor` 把 `Actor` 放进 `context`。平台不认识 `Actor`，也不导入 `shared`（3.3）。
  - 第二个返回值是限流键（`session:<会话 id>` 或 `pat:<PAT id>`）。平台读不到 `Actor`，限流中间件靠它按凭证计数（3.10）。
- **`shared.Actor`**：`{UserID, SessionID, APITokenID}`。通过会话认证时 `APITokenID` 为零值，通过 PAT 认证时 `SessionID` 为零值。这只区分凭证的种类，不区分账户的种类（总体设计 0.2 原则 1）。
  - handler 用 `shared.RequireActor(ctx)` 取当前账户。取不到时返回未认证错误（401 `unauthorized`）；有了默认拒绝，这只在接线出错时才会发生。
- **不用 oapi-codegen 的 `enable-auth-scopes-on-context`**：它是被标为"遗留"的兼容选项（`oapi-codegen/v2@v2.8.0/pkg/codegen/configuration.go:367-388`）。默认拒绝只需要"哪些操作公开"这一张表。
- **三个整程序测试**（`bootstrap`，P1；3.11 另有第四个）：
  1. 各模块声明的公开操作的并集，等于接口描述中 `security: []` 的操作集合。
  2. 注册在 `/api/v0` 下的路由模式，等于接口描述中的全部操作。
     - 接口描述之外的路由（例如 M5 本地存储的上传地址）会被发现：要么写进接口描述，要么不放在 `/api/v0` 下，并在那个 M 的设计中说明。
     - 为了拿到已注册的模式，`httpserver.NewMux` 改为返回 `*httpserver.Router`：它包着 `http.ServeMux`，在 `HandleFunc` 和 `Handle` 时都记下模式（`bootstrap` 用 `Handle("/", …)` 挂载 web 界面），只暴露这几个会记录的方法和 `ServeHTTP`。生成代码的 `BaseRouter` 只要求 `HandleFunc` 和 `ServeHTTP`（spike `server.gen.go:219-222`），`Router` 满足它。
  3. 每个非公开操作，不带令牌时得到 401 problem。路径参数和必填的查询参数填合法的示例值（例如全零的 uuid）：参数在中间件之前绑定，填错会先得到 400。
- **限流中间件**也在 `httpserver`（3.10），通过同样由 `httpserver` 声明的 `Limiter` 接口使用 `platform/ratelimit`。平台包之间不互相导入（规则 7）。
- **模块入口的演进**（M0-P3 交接 4）：
  - 平台交给模块的 HTTP 依赖合成一个值 `httpserver.API`：`Errors`（3.11 的映射），和方法 `Middlewares(bodies)`：按下面的顺序返回按路由的中间件，最里层是这个模块的请求体结构检查（3.11）。顺序只在平台里写一次，模块不需要知道。
  - 模块写成 `Register(router, api)`；`httpadapter.Register` 接收一个用例集合的结构体，不再逐个传指针。
- **按路由的中间件和顺序**：请求元信息（客户端 IP 和限流用的 IP 键、UA）→ 请求期限 → 请求体上限 → 失败闸门和认证 → 限流 → 请求体结构 → （生成的代码）解码请求体 → handler。
  - oapi-codegen 的 `Middlewares` 列表中最后一个在最外层，所以 `API.Middlewares` 按相反的顺序返回，并有测试核对。
  - 生成代码在这些中间件**之前**绑定路径参数和查询参数，在它们**之后**（strict handler 中）解码请求体（spike `server.gen.go:119-141`、`:477`）。所以参数格式错误的请求在认证和限流之前就得到 400。这类请求不碰数据库，不计入限流也没有代价。
  - 请求体的结构检查放在认证和限流之后：没有认证的请求不值得解析请求体。
  - **请求期限**：`server.request_timeout`（默认 15 秒）的 `context.WithTimeout`。handler 和用例里的数据库调用都带请求的 `context`，到期即取消。`server.write_timeout` 到期只让写出失败，不会取消请求的 `context`（M0-P2 交接 5），所以需要这一层。续期和退出在适配器里再缩短到 4 秒（3.5）。
  - **提交不受请求期限的取消**（控制者复核 R7，全局规则，在 `platform/postgres` 的 `TxManager`）：
    - 事务里的语句带请求的 `context`，期限到了就取消。`COMMIT` 和 `ROLLBACK` 都在 `context.WithoutCancel(ctx)` 下执行，另有自己的短期限 `database.commit_timeout`（默认 2 秒）。
    - 这样请求期限约束的是语句：语句都已做完的事务，不会因为请求期限在提交时被取消，`TxManager` 的事务不会因此"已提交却答 500"。回滚也不被取消，失败的事务不会把连接留在不确定的状态。`ROLLBACK` 本身失败（例如连接断了）是基础设施的故障：`WithinTx` 返回的错误包住这次失败，只保留领域错误的文字、不保留它的身份，所以接口答 500 并记 ERROR 日志，而不是用领域错误的答复把故障盖住。
    - `COMMIT` 已经发出、却在自己的期限内没有回应时，**结果未知**：数据库可能已经提交，也可能没有。接口答 500，记 ERROR 日志。这是数据库的故障，与请求期限无关；续期遇到它的后果见 3.5 的第 4 种情况。
    - 这条规则只管 `TxManager` 的事务。不开事务的单条写入（6.4：资料和偏好的 PATCH、退出、撤销 PAT、`last_used`）在语句执行中被取消时，数据库可能已经提交而接口答 500。这无害：PATCH 重试写的是同样的值；退出和撤销 PAT 重试时，要的状态已经达到（204 或 404）；`last_used` 本来就是尽力而为。
    - 测试：语句做完之后取消请求的 `context`，事务仍然提交；语句失败之后取消，回滚仍然完成，连接回到池里可以再用。

### 3.7 签名密钥
- **来源**：
  - `auth.jwt.private_key_file`：PKCS#8 PEM 格式的 Ed25519 私钥文件。这是总体设计 6.8 举的例子，环境变量是 `NERVE_AUTH__JWT__PRIVATE_KEY_FILE`。
  - prod 必须提供，否则启动失败，并指出这个配置项。
  - dev 和 test 可以留空：启动时生成一把临时密钥，记一条 WARN。
    - 当前一代刷新令牌由数据库中的哈希认出，与签名密钥无关。重启之后，旧的访问令牌验签失败，客户端按 401 续期一次就恢复，所以临时密钥不影响开发。
    - 端到端测试的每个 worker 有自己的 nerve，令牌不跨进程使用。
  - 生成密钥：`openssl genpkey -algorithm ed25519 -out nerve-jwt.pem`，写进 README。不另做命令。
- **两个用途**：这把密钥签访问令牌；刷新令牌标签用的 MAC 密钥也从它的种子派生（HKDF，`info` 分开两种用途，3.4）。所以没有第二个密钥文件和配置项。两者都由 `identity/adapter/signing` 持有（6.2）。
- **轮换**：换掉密钥文件，重启。
  - 旧的访问令牌验签失败，客户端续期一次，不需要新旧密钥并存。所以不引入 `kid`，也不提供 JWKS：v0 没有外部验签方。
  - 当前一代刷新令牌照常续期（3.5 的当前一代不看标签）。换钥之前签发的**旧代**刷新令牌，标签无法再验证，按伪造处理：401，不撤销会话。也就是说，换钥的那一刻起，更早的旧令牌被再次使用时不再能被发现（§16）。dev 的临时密钥每次重启都相当于换钥。
  - 另两个后果登记在 §16：换钥后旧的访问令牌验签失败，会暂时用空同一出口 IP 的失败闸门；刷新令牌正被盗用时换钥，先续期的一方留下会话。
  - 以后要多实例部署或让外部系统验签时，再加 `kid` 和公钥列表；那时 MAC 密钥也按 `kid` 保留上一把，旧代令牌在换钥之后仍能验证。
- **日志**：私钥的内容永远不进日志。`LogValue` 对所有 `*_file` 配置项只记"是否设置"（`private_key_file_set: true`），不记路径，按 M0-P2 交接 4 的原文关闭（Codex M-4）。

### 3.8 密码
- **哈希**：argon2id（`golang.org/x/crypto/argon2`），存 PHC 字符串（`$argon2id$v=19$m=…,t=…,p=…$盐$哈希`）。
  - 默认参数：m = 19456 KiB、t = 2、p = 1，盐 16 字节，输出 32 字节（OWASP 推荐的最低配置）。
  - test 环境降到 m = 64 KiB、t = 1（总体设计 6.8）。
  - 登录时发现参数和当前配置不同，就用新参数重新哈希。新哈希在事务外算好，在登录的事务里、账户行锁下写回（3.5）。
- **并发上限和等待上限**：
  - 同时进行的哈希计算最多 4 个（`auth.password.max_concurrent_hashes`），4 × 19 MiB ≈ 76 MiB。
  - 拿不到名额时最多等 `auth.password.max_wait`（默认 2 秒），然后返回 503 `server_busy`，带 `Retry-After: 1`。第一稿的通道没有等待上限，名额被占满时请求一直排队到超时（评审 I3）。
  - **为什么是 503，不是 429**：429 的意思是"你这个调用方发得太多了"（RFC 6585），界面会显示"尝试次数过多"。名额满是整个服务器暂时过载，与这个调用方无关，RFC 9110 为此规定的是 503 加 `Retry-After`。通用的 HTTP 客户端和 Agent 也会按 503 重试。
  - 能用多少次哈希由 3.10 的限流约束，这里只保证过载时尽快失败。
- **规则**：在注册、修改密码、命令行的创建账户和重置密码四处，服务端都执行下面两条。Plane 在注册、修改密码、重置命令三处用 zxcvbn 评分 ≥ 3（`plane/apps/api/plane/authentication/adapter/base.py:90-100`、`authentication/views/common.py:83`、`db/management/commands/reset_password.py:56`）。登录不检查（与 Plane 相同）。
  1. **组合规则**，与界面上已经显示的一致：
     - 长度 8–128 个字符；
     - 至少一个大写字母、一个小写字母、一个数字；
     - 至少一个特殊字符，取自 `!@#$%^&*()-_+=[]{}|;:'",.<>?/`。
     - 前四项与 `web/packages/utils/src/auth.ts:12-32` 的 `getPasswordStrength` 完全一致；128 的上限是新加的，界面同步加上。不合规时字段码是 `weak_password`。
  2. **常见密码名单**（评审 I2，控制者裁定；主干的定义按控制者复核 N1 修订）：
     - **主干**：把密码转小写，再从两端去掉所有**不是字母**的字符（Unicode 字母类 `\p{L}`，Go 的 `unicode.IsLetter`）。例如 `Password1!~`、`~Password1!`、`Password1! ` 的主干都是 `password`。
     - **判定**：小写的整个密码在名单中，或主干在名单中，就拒绝；主干等于邮箱 @ 之前那部分的主干，也拒绝（NIST SP 800-63B 建议排除与账户有关的词）。字段码是 `common_password`。
     - **为什么第二稿的主干不够**：第二稿只去掉两端的数字和组合规则里的特殊字符，两端多一个 `~`、反引号、`\` 或空格就绕过了。这些都满足组合规则，第二稿会接受：`Password1!~`、`Password1! `、`~Password1!`、`Summer2024!` 加反引号、`Welcome1!\`、`Qwerty123!~`、`Dragon#2026 `。新的主干把它们都挡住，单元测试逐个列出（9.1）。
     - **为什么要取主干**：组合规则会挡掉名单中几乎所有原样的条目，只比原样没有用。spike 对名单逐条实测（Codex M-9 要求写明口径）：
       - **原样通过**组合规则的条目：前 1 万个里 6 个，前 10 万个里 37 个；
       - **存在能通过的大小写变体**的条目（转小写后，至少有两个 ASCII 字母、一个数字、一个特殊字符，长度 8–128；某个密码转小写后等于它）：前 1 万个里 12 个，前 10 万个里 288 个。第二稿写的 294 用的是"至少一个字母"，多算了 6 个只有一个字母、任何大小写变体都通不过组合规则的条目（例如 `1.23457e+11`）。
       - 常见的写法是"常见词，首字母大写，前后加数字和符号"：`Password1!`、`Summer2024!`、`Qwerty123!`、`Welcome1!`、`P@ssw0rd1`、`Dragon#2026`、`Zxcvbnm1!` 以及上面七个绕过的写法都被拒绝；`Tr0ub4dor&3`、`Correct-Horse-9`、`Nerve2026!` 通过。
     - **来源**：英国国家网络安全中心（NCSC）发布的泄露最多的前 10 万个密码（`PwnedPasswordsTop100k.txt`，取自 Have I Been Pwned）。SecLists 收录为 `Passwords/Common-Credentials/100k-most-used-passwords-NCSC.txt`；spike 下载的文件 SHA-256 为 `c2e56968…c576e0`。
     - **许可**：NCSC 网站的内容是 Crown copyright，按 Open Government Licence v3.0 再使用，要求注明出处（NCSC 网站的条款页）。名单文件头和 README 的第三方声明写上"Contains public sector information licensed under the Open Government Licence v3.0"。第三方的权利不能只凭网站的总条款推定，P1 加入文件时对这个具体的数据文件再核对一次许可。
     - **生成时的过滤**：运行时的判定只做两种不区分大小写的查找（小写的整个密码、主干），所以名单只需要留下这两种查找可能命中的条目。一个条目 e（已转小写）被留下，当且仅当：
       - 某个满足组合规则的密码转小写后等于 e：长度 8–128，至少两个 ASCII 字母、一个数字、一个特殊字符；或者
       - 某个满足组合规则的密码的主干等于 e：e 的主干就是它自己（两端都是字母），至少两个 ASCII 字母，长度不超过 126（数字和特殊字符可以在去掉的两端）。
       - 去掉的条目不可能命中，结果与用全表相同。前 10 万个过滤后剩 33,887 条、277,141 字节；前 1 万个只剩 5,044 条，挡不住 `Summer`、`Welcome` 之外稍长一点的常见词，所以用 10 万。
     - **嵌入**：排好序、每行一个的文本文件，用 `//go:embed` 放进 `identity/domain`（`embed` 是标准库，符合规则 2）。启动时切成有序切片，约 0.5 MB，二分查找。没有新依赖。
     - **生成**：`tools/password-blocklist/build.mjs` 读取下载的原文件，核对 SHA-256，用与 Go 相同的主干规则（`/^[^\p{L}]+|[^\p{L}]+$/gu`）过滤、转小写、去重、排序，写出文件和文件头（来源、许可、SHA-256、过滤规则）。原文件不进仓库。Go 的单元测试核对：名单中每个条目的主干要么是它自己，要么它满足第一条过滤条件，确保两边的主干规则没有走样。
- **为什么不照搬 zxcvbn**：
  - Go 的 zxcvbn 移植都已多年不维护，评分和 Python 版不会逐字相同，"照搬"本来就做不到；它的字典还会让程序多出近 1 MB。
  - 名单加主干的判定是确定的，容易测，失败时能明确告诉用户原因。
  - 登记为差异（4.6）：Plane 在服务端用 zxcvbn，组合规则只在界面上；Nerve 在服务端执行组合规则和名单。
  - 第一稿只保留组合规则，还把它说成"修复不一致"。这是错的：Plane 的两层检查是叠加的，不是矛盾的，只留组合规则是削弱。
- **修改密码不检查"新旧相同"**：Plane 也不检查（`ChangePasswordSerializer` 中的检查是死代码）。界面上已有的"新旧必须不同"提示保留。

### 3.9 账户枚举与时间
- **登录**：
  - 邮箱不存在和密码错误，返回同一个 401 `identity.invalid_credentials`。
  - 邮箱不存在时，仍然用同样的参数对一个启动时生成的假哈希做一次校验，让两种情况耗时相同（存储的哈希都用当前参数时；调高参数之后的局限见 §16）。
  - Plane 会返回 `USER_DOES_NOT_EXIST`（`plane/apps/api/plane/authentication/views/app/email.py:90-104`），登记为差异。
- **停用的账户**：只在密码正确之后才返回 403 `identity.account_deactivated`（3.5 的登录事务里判定），所以只对知道密码的人暴露账户状态。
- **注册**：
  - 注册关闭时，先答 403 `identity.signup_disabled`，再做任何邮箱查询和哈希（m6）：关闭的实例对已注册和未注册的邮箱回答相同。
  - 注册开放时，邮箱已存在必然失败，没有邮件通道就没法做到"不暴露"（Plane 同样返回 `USER_ALREADY_EXIST`）。用按 IP 的注册限流（3.10）压低探测速度，并在 8.2 写明这个局限。
- **令牌的比较**：
  - 刷新令牌的当前一代按会话 id 读出会话行，在 SQL 的条件 `UPDATE` 中比较 `token_hash`；PAT 按 `token_hash` 查唯一索引。被比较的是 256 位随机数的 SHA-256，比较耗时最多泄露哈希的前几个字节，据此推不出令牌，也拼不出另一个能用的令牌，所以不需要常数时间比较。第一稿写"用 `ConstantTimeCompare`"，与实际的匹配位置不符，已改。
  - 旧代的标签是 MAC，比较耗时会一点点泄露正确的标签，所以用 `hmac.Equal`（常数时间，3.4）。

### 3.10 限流
"具体数值在 M2 确定，默认参考 Plane"（总体设计 3.6）。实现是 `platform/ratelimit` 自己的进程内按键令牌桶（约 100 行，只用标准库），闲置的键定期清掉。不用 `golang.org/x/time/rate`：M2 要"几个桶全扣或全不扣"和"事后退回一个单位"（3.6 的失败闸门），它的 `Reservation.CancelAt` 在预留的时刻过去之后什么也不退（spike：预留后 5 毫秒再取消，余额没有恢复），做不到后者。令牌桶需要速率和突发两个参数：一次页面加载会并行发出两三个请求，只有速率没有突发的桶会误伤它们。所以**每个桶都有 `per_minute` 和 `burst` 两个配置项**（6.5）。

**桶**：

| 桶 | 键 | 默认（每分钟 / 突发） | 位置 | Plane 的对应项 |
|---|---|---|---|---|
| `anonymous` | 客户端 IP 键 | 600 / 100 | `httpserver`，每个公开操作 | 匿名请求 30/minute（`DEFAULT_THROTTLE_RATES`，`settings/common.py:140-144`）。Nerve 的页面每次加载都会不带令牌调用实例配置、时区和续期，同一出口 IP 后的团队很快就会碰到 30。续期和退出只经过这一个桶 |
| `auth_failure` | 客户端 IP 键 | 60 / 60 | `httpserver`，认证之前的失败闸门（3.6）：先预留，只有认证失败才留下 | 无 |
| `authenticated` | 凭证：会话 id 或 PAT id（3.6 的限流键） | 1200 / 200 | `httpserver`，每个非公开操作 | 页面请求不限流，API Key 每分钟 60 次。Nerve 的页面和 Agent 用同一套令牌，60 次连一次项目页面的加载都撑不住 |
| `login_ip` | 客户端 IP 键 | 30 / 10 | `identity` 适配器 | 认证接口合计每 IP 10/minute（`AUTHENTICATION_RATE_LIMIT`，`plane/apps/api/plane/authentication/rate_limit.py:26-30`） |
| `login_ip_email` | 客户端 IP 键 + 规范化后的邮箱 | 10 / 5 | `identity` 适配器 | 同上 |
| `register_ip` | 客户端 IP 键 | 10 / 5 | `identity` 适配器 | 同上 |
| `password_user` | 账户 id | 5 / 5 | `identity` 适配器 | 无。所有需要校验密码的已认证操作都经过它；M2 中只有修改密码 |

**每个操作经过哪些桶**：

| 操作 | `httpserver` 的桶 | `identity` 的桶（全有或全无） | 哈希次数 |
|---|---|---|---|
| 登录 | `anonymous` | `login_ip` + `login_ip_email` | 1（邮箱不存在时是假哈希，同样 1 次） |
| 注册 | `anonymous` | `register_ip` | 1 |
| 续期、退出、实例配置、时区 | `anonymous` | — | 0 |
| 修改密码 | `authenticated` | `password_user` | 2（校验旧密码，哈希新密码） |
| 其余需要登录的操作 | `authenticated` | — | 0 |
| 非公开操作带了令牌 | 认证之前预留 `auth_failure` 的一个单位；认证失败时留下，其余情况退回 | — | 0 |

- **为什么这样分**（评审 I3、Codex I-3、控制者复核 N3）：
  - 第一稿的登录只按 IP + 邮箱计数，换一个邮箱就是一个新桶；修改密码只受每凭证 1200 次的约束，而一个账户可以开任意多个会话和 PAT。任何人都能占满 4 个哈希名额，真正的登录排队到超时。
  - 现在同一个 IP 每分钟最多触发 40 次哈希（登录 30 + 注册 10），突发时再多 15 次；每个账户每分钟最多 5 次修改密码。4 个名额每秒能做约 100 次哈希（每次约 40 毫秒，P1 实测后写进 review），单个 IP 远远占不满。名额满了等 2 秒就答 503（3.8），不会无限排队。
  - 登录的"按账户"桶是 IP + 邮箱，不是只按邮箱：只按邮箱的话，任何人从任何地方都能把别人锁在门外。修改密码的请求已经认证过，按账户 id 计数没有这个问题。
  - 续期、退出只经过按 IP 的 `anonymous`（每分钟 600、突发 100）。一个团队在同一出口 IP 后开很多标签页，每个页面每 15 分钟才续期一次，用不完这个额度。伪造的刷新令牌只花一次按主键的查找（3.5），同样受 `anonymous` 约束。
  - **限流键不取未经验证的输入**（控制者裁定）：第三稿的初稿曾按刷新令牌里的会话 id 另设一个桶。会话 id 没有经过验证，拿它做键，知道某个会话 id 的人就能用伪造的令牌耗尽这个会话的额度，让它续不了期。所以键只能是两类：认证过的凭证（`authenticated`、`password_user`），或者包含调用方自己的 IP（`anonymous`、`auth_failure`、`login_ip`、`register_ip`，以及 `login_ip_email`：邮箱同样没有验证，但键里有调用方的 IP，耗尽的只是调用方自己的额度）。
  - 无效的令牌由认证之前的闸门限制（3.6）。
- **全有或全无**（控制者复核 m6）：用 `Allow()` 依次检查几个桶，后面的桶拒绝时，前面的桶已经扣掉了。例如同一个邮箱反复失败，`login_ip_email` 拒绝的每一次都扣了 `login_ip`，这个 IP 很快连别的邮箱也登录不了。
  - `platform/ratelimit` 提供一次检查多个键的 `AllowAll(checks…)`：在同一把锁下先看每个桶是否都有余额，都有才一起扣；有一个没有，就都不扣，返回最长的等待时间。另有 `Reserve(check)`：扣一个单位，同时返回一个只能用一次的退回函数（3.6 的失败闸门用它）。spike：`login_ip_email` 拒绝 10 次之后，`login_ip` 的余额不变；失败闸门在 50 个并发请求下只放过额度内的 3 个。
  - `identity` 适配器声明的小接口只有 `AllowAll`，一个操作在适配器里的桶一次查完。`httpserver` 的桶在它之前扣：被适配器拒绝的请求仍然算作一次匿名或已认证的请求，这是有意的。
- **位置**：
  - `anonymous`、`authenticated`、`auth_failure` 在 `httpserver`（3.6）。有限流键时按凭证计数，没有时按 IP 计数。
  - 其余四个桶在 `identity` 的 HTTP 适配器里，在调用用例之前：登录的键里有邮箱，要先解码请求体；`password_user` 的键是当前账户。适配器通过自己声明的小接口使用 `platform/ratelimit`，超出时返回 `shared` 的限流错误，走"一条路"映射（3.11）。
- **超出**：429 `rate_limited`，带 `Retry-After`（秒，向上取整）。不加 `X-RateLimit-*`（登记为差异）。
- **客户端 IP**：
  - 默认取连接的对端地址。
  - `server.trusted_proxies`（CIDR 列表）非空，并且对端在列表中时，才从 `X-Forwarded-For` 从右往左取第一个不可信的地址。
  - 默认部署前面有 Caddy 时，要配置这一项。README 写明。
  - 对端不在可信列表中、请求却带着 `X-Forwarded-For` 时，进程记一次 WARN（只记一次）：多半是反向代理没有配置成可信，所有人都会落进代理地址这一个桶。
- **客户端 IP 键**（控制者复核 R6）：按 IP 计数的桶（`anonymous`、`auth_failure`、`login_ip`、`login_ip_email`、`register_ip`）都用同一个键，由请求元信息中间件算出：
  - IPv4 地址：就是这个地址；IPv4 映射的 IPv6 地址（`::ffff:a.b.c.d`）按 IPv4 处理。
  - IPv6 地址：取前缀，长度是 `ratelimit.ipv6_prefix_len`（默认 64）。一台主机通常拿到整个 /64，按完整地址计数时，它换一个地址就是一个新桶。
  - 日志和 `auth_sessions.ip` 仍记完整的地址。
  - 测试：同一个 /64 里的两个地址落进同一个桶，不同 /64 的不落进；映射的 IPv4 与原 IPv4 相同；前缀长度可配。
- **测试环境**：test 配置把上限都调得很高，否则同一个 IP 注册的大量测试账户会被限住。A15 用一个限流很低的独立 nerve。

### 3.11 错误码与取值校验
这是 M0-P3 交接 2 要求 M2 一次定下的体系。

- **领域错误**：`shared.Error{Kind, Code, Detail, Fields, RetryDelay}`。模块用 `shared` 的构造函数声明自己的错误，码带模块前缀，例如 `identity.email_taken`。

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

  - `shared.Error` 和 `shared.FieldError` 按结构满足它们，彼此不导入。`shared.Error` 的等待时长字段叫 `RetryDelay`，方法叫 `RetryAfter()`：第二稿两者同名，不能编译（Codex M-2）。spike 已编译并测试这份草图：逐个 `Kind` 的映射、`Retry-After` 向上取整、字段错误、包在 `errors.Join` 里的错误、不满足接口的错误映射为 500。
  - **一处小的取舍**：`ProblemStatus()` 让领域错误带上了 HTTP 状态。这不是完全与传输无关的写法；换来的是平台不需要导入 `shared` 就能映射，而且 problem 的 `status` 本来就是错误对外契约的一部分。以后出现第二种传输时，再把状态表移到那一侧。
  - 不满足 `ProblemError` 的错误一律 500 `internal_error`。`bootstrap` 的测试逐个 `Kind` 核对映射。
- **一条路**：handler 返回 `error`，由 `httpserver.APIErrors` 作为生成代码的 `ResponseErrorHandlerFunc` 统一映射为 problem+json。不用"每个操作声明带类型的 `default` 响应"那条路：那样每个 handler 都要自己拼 problem。
- **结构在接口边界，取值在领域**（Codex I-6，约束后续每个 M）：
  - **接口边界**按接口描述检查每个请求体的每一层，在 handler 之前：
    - JSON 语法和类型；
    - 未声明的字段（`additionalProperties: false`）：拒绝，字段码 `not_allowed`；
    - 不可为空的字段传了 `null`：拒绝，字段码 `invalid_format`；
    - 缺少必填字段：拒绝，字段码 `required`；
    - 生成为 Go 类型的字符串格式（M2 的模板下是 `date-time` 和 `uuid`）写错：拒绝，字段码 `invalid_format`。生成的类型装不下错误的值，这几种格式只能在解码之前检查。检查**与生成的解码器接受的完全相同**：用 `encoding/json` 把这个字段的原始 JSON 值解码进映射到的 Go 类型，即 `json.Unmarshal(raw, new(T))`，`T` 是 `time.Time` 或标准库的 `uuid.UUID`。生成的解码器对这个字段做的正是这件事，所以两者按构造一致，JSON 字符串里的转义也一样处理；不另写正则（控制者复核 R1，核验 F1）。
      - spike（仓库锁定的 Go 1.27.1）：16 种写法上检查器与解码器的结论全部相同。`date-time`：带 `Z` 和带时区偏移的 RFC 3339 接受；空格分隔、只有日期、数字、布尔、任意字符串拒绝；`Z` 写成 JSON 转义时同样接受。`uuid`：标准写法、大写、无连字符、带花括号、带 `urn:uuid:` 前缀、含 JSON 转义的都接受，非法字符串和数字拒绝。标准库的解析比 RFC 9562 的标准写法宽松，边界与解码器一致，领域拿到的是同一个值。
    - 以上一律 400 `bad_request`，`errors[{field, code}]`，一次收集全部问题，`field` 是 JSON 路径（`onboarding_step.profile_completed`、`tags[1].name`）。
  - **领域层**负责长度、其余格式（邮箱这类映射为 `string` 的格式）、枚举、取值范围和跨字段的规则：一次收集全部字段的问题，返回一个 422 `validation_failed`。handler 只做类型转换，从不自己解码请求体。
  - **做法**（spike 比较了两种，选第二种）：
    1. 候选一：覆盖 oapi-codegen 的 strict 模板，把请求体的解码换成平台的严格解码器，由生成的 Go 类型推断契约（`nullable.Nullable[T]` 表示可为空，没有 `omitempty` 表示必填，`DisallowUnknownFields`）。
    2. 候选二：构建时的生成器读接口描述，为每个模块生成一张紧凑的结构表；平台的校验器在解码之前按 `r.Pattern` 检查原始请求体。
    - spike 用同一份接口描述（嵌套对象、可为空的对象和字段、数组、开放的 map、部分更新的嵌套对象、`date-time`）跑了 25 个请求，两者在 23 个上结果相同。不同的两个都是 `date-time` 字段传了数字：候选一的反射遍历把有自己解码方法的类型当作不透明的叶子（spike 中是 `time.Time`；标准库的 `uuid.UUID` 实现了 `UnmarshalText`，按同样的道理也是如此），这一项漏出了一次收集，再由 Go 的解码器报错，错误信息 `Time.UnmarshalJSON: input is not a JSON string` 进了响应；候选二在同一次里报出 `when: invalid_format`。
    - 候选一还要复制上游 147 行的 strict 模板，升级 oapi-codegen 时模板的漂移 `gen-check` 发现不了；它从 Go 类型的写法推断契约，而 `omitempty` 也会因 `readOnly`、`writeOnly`、`x-omitempty` 出现（`oapi-codegen/v2@v2.8.0/pkg/codegen/schema.go:1427`）。候选二直接读契约里的 `required`、可为空和 `additionalProperties`，遇到不支持的组合就让生成失败。
    - 规模：候选二的运行时校验器约 170 行（加上格式检查约 200 行），生成器约 200 行；候选一约 190 行加 147 行复制的模板。
  - **候选二的组成**：
    - `platform/httpserver/bodyshape`：结构表的类型、校验器和中间件。中间件读出请求体（已受请求体上限约束），按表检查原始字节（逐层取 `json.RawMessage`，不解码成通用值，格式检查器拿到的就是生成代码解码时看到的字节），收集全部问题，按字段路径排序；然后把请求体原样放回，交给生成的 strict handler 解码。请求体被解析两次，上限 1 MiB，代价可以接受。
    - 格式检查器按**生成的 Go 类型**登记：`time.Time`、标准库的 `uuid.UUID` 各一个，都是上面那一行 `json.Unmarshal`。`bodyshape` 只依赖标准库。
      - 上一稿还为 `format: date` 登记了 `openapi_types.Date`，让 `bodyshape` 依赖 `oapi-codegen/runtime/types`，而那个包导入 `github.com/google/uuid`（核验 F1，3.12）。M2 没有 `date` 字段，这个检查器删去；生成器遇到 `date` 就失败（见下）。以后第一个用到 `date` 的 M 选定它的 Go 类型，带着测试登记检查器。
    - `server/tools/bodyshapegen`：生成器，放在工具模块（`server/tools/go.mod`），与 oapi-codegen 同一个模块。它用 oapi-codegen 自己的加载器读 `api/modules/<m>.yaml`（跨文件的 `$ref` 一并解析，与生成 `server.gen.go` 时读到的是同一份），为每个带 JSON 请求体的操作生成根节点，写出 `internal/modules/<m>/adapter/http/gen/bodyshape.gen.go`。它不在 `server/go.mod` 里，更不链接进 nerve；工具模块已有 oapi-codegen v2.8.0 和它用的 kin-openapi，不新增依赖。
      - **格式的清单取自同一份 `type-mapping`**：生成器读这个模块的 `oapi-codegen.yaml` 中的 `output-options.type-mapping`，与 oapi-codegen 的默认映射合并（`codegen.DefaultTypeMapping.Merge`，与 `oapi-codegen/v2@v2.8.0/pkg/codegen/codegen.go:162-165` 的做法相同），得到每个字段实际生成的 Go 类型。生成为 `string` 的格式（`email`）不检查，交给领域层；生成为已登记检查器的类型（`time.Time`、`uuid.UUID`）的，表中记下检查器；生成为其他类型的，生成失败并说明原因。数值同理：`bodyshape` 的 `Integer` 是 int64 容纳得下的字面量、`Number` 是 float64 容纳得下的，所以 `integer` 只能生成为 `int` 或 `int64`，`number` 只能生成为 `float64`（模板把 `number` 的默认映射改为 `float64`，3.12）；生成为更窄的整数、无符号整数、`float32`，或 `format` 不在映射里的，生成失败。模板改了映射，表随之改变，两边不会走样。
      - 字段带 `x-go-type`（或 `x-go-type-import`）时生成失败：它绕过 `type-mapping`，生成器推不出实际的类型。M2 的接口描述不用它。
      - 支持的写法：对象（属性、必填、`additionalProperties` 为 `false`、`true` 或一个 schema）、数组、标量、`type: [X, 'null']` 和 `anyOf: [X, {type: 'null'}]`，以及上一条的格式。遇到其他 `anyOf`/`oneOf`/`allOf`，或生成为没有检查器的类型的格式（例如 `date`、`byte`、`binary`、`duration`），生成失败并说明原因。
      - 输出是确定的：路径、方法、属性都排序后再编号。spike 中按 map 顺序遍历，两次生成的编号不同，`gen-check` 会误报。
    - 接线：模块的 HTTP 适配器在 `Register` 中调用 `api.Middlewares(gen.BodyShapes)`，结构检查就是最里层的中间件（3.6）。没有全局的表，每个模块的表跟着自己的路由。
    - `make gen-go` 在 oapi-codegen 之后逐个模块运行生成器（`go -C tools run ./bodyshapegen …`）；输出在 `GEN_GO_OUT` 已包含的目录里，`gen-check` 覆盖它。
    - **工具模块进持续集成**（核验 F2）：`server/tools` 是嵌套的独立模块，`cd server && go test ./...` 和 `golangci-lint run ./...` 都不会进入它，生成器的测试就没人跑。P1 让 `make test` 另跑 `go -C tools test -count=1 ./...`，`make lint-go` 另在工具模块跑一遍 golangci-lint（同一份 `.golangci.yml`，其中的 `standard` 含 govet）。持续集成调用的就是这两个目标。
    - **格式检查的第一个使用者在 P3**：M2 中请求体带格式字段的是 `createApiToken` 的 `expired_at`（`date-time`）和 `updateProfile` 的 `last_workspace_id`（`uuid`），都在 P3。P1 建好检查器并做单元测试，整程序测试中逐格式的情况从 P3 起有操作可测。
  - **第四个整程序测试**（`bootstrap`，P1；前三个见 3.6）：从接口描述中找出每个带请求体的操作，由它的 schema 造出一个合法的请求体，再逐项改坏，发给真实组合出来的程序（非公开操作带一个有效的令牌），断言 400 和对应的 `errors[].field`、`code`：
    1. 顶层多一个未知字段；
    2. 嵌套对象里多一个未知字段（有嵌套对象时）；
    3. 可省略、不可为空的字段传 `null`（有这种字段时）；
    4. 缺一个必填字段（有必填字段时）；
    5. 可为空的字段传 `null` 不得到 400（总体设计 3.1："传 `null` 表示清空"）；
    6. 每种带格式的字段传一个写错的字符串（有这种字段时）：400 `invalid_format`，`field` 是这个字段；
    7. 同一个请求里同时有未知字段、缺少的必填字段和写错的格式：一次返回全部问题。
  - **随之删除的规则**：第二稿给"零值也合法的必填字段"加 `writeOnly: true` 的规则删除：必填字段是否出现，现在在边界上检查。第二稿"已知的不一致"一段（未知字段被忽略、`null` 等同没传）随之删除。
  - `apitest` 的 `CheckRequest` 保留，让测试发出的请求也按契约校验。它是测试的辅助，不是服务端的保证；故意发不合契约请求的测试（例如 `limit=0`）绕过它。
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
  | `bad_request` | 400 | 已有。请求体不是合法 JSON：`detail` 是通用的一句话。请求体的结构不合契约：`errors[{field, code}]`（见上）。参数的类型不符：字段路径取自绑定错误，不再带出 Go 的类型名（M0-P3 交接 2）。游标不合法 |
  | `unauthorized` | 401 | 新增。认证中间件，以及 `RequireActor` |
  | `not_found` | 404 | 已有 |
  | `payload_too_large` | 413 | 新增。`http.MaxBytesError`；JSON 接口的请求体上限是 `server.max_body_bytes`（默认 1 MiB） |
  | `validation_failed` | 422 | 新增。取值校验 |
  | `rate_limited` | 429 | 新增 |
  | `internal_error` | 500 | 已有 |
  | `not_ready` | 503 | 已有，只用于 `/readyz` |
  | `server_busy` | 503 | 新增。密码哈希的名额等待超时（3.8） |
- **`context.Canceled`**：客户端断开，不记 500，也不记 ERROR 日志，只在 DEBUG 记一行。
- **查询参数**：类型不符在绑定时得到 400；取值超出范围是领域的规则，得到 422。例如 `limit` 为 `abc` 是 400，为 `0` 或 `101` 是 422 `validation_failed`（`errors[{field: limit, code: out_of_range}]`），`listApiTokens` 在 `x-problem-codes` 中声明 422（Codex M-1）。
- **错误出口的测试**：参数绑定、请求体解码（含结构检查）、handler 返回错误三个出口都要测，并对 problem 调 `CheckResponse`（M0-P3 交接 2）。
  - P1 只有 `register`、`getMe`，没有路径参数和查询参数：P1 测请求体解码和 handler 两个出口。
  - 参数绑定的出口由 P3 第一批带参数的操作测：`listApiTokens` 的 `limit=abc`、`revokeApiToken` 的非法 uuid。M0-P3 交接在 P3 整体关闭。
  - 不为了测试在生产的接口描述里加专门的操作。

### 3.12 接口描述与代码生成的约定
- **oapi-codegen 的模块模板**（M0-P3 交接 1，从 M2 起每个模块照抄；spike 已验证下面的选项能生成、能编译，0.3）：
  - `output-options.nullable-type: true`：可为空又可省略的字段生成 `nullable.Nullable[T]`，PATCH 能区分"没传"和"传 `null`"。依赖 `github.com/oapi-codegen/nullable` v1.2.0，写死。
  - `output-options.type-mapping.string.formats`：`uuid` → `{type: uuid.UUID, import: uuid}`（标准库）；`email` → `{type: string}`。默认的 `openapi_types.Email` 在解码时自己校验格式，会绕过 3.11 的分层；映射为 `string` 以后，邮箱格式由领域层校验（422）。
  - `output-options.type-mapping.number.default`：`{type: float64}`。默认的 `float32` 容纳不了的值会通过边界的检查、再被解码器拒绝，不带 `errors[]`（3.11）。
  - `prefer-skip-optional-pointer` 保持默认（`false`）：可省略的字段是 `*T`，`nil` 表示没传。
  - 第一个带路径参数的操作会让生成代码导入 `github.com/oapi-codegen/runtime`，写死为 v1.7.0。
- **`github.com/google/uuid` 的守卫改为直接检查**（核验 F1，控制者裁定；P3，与第一个带参数的操作一起）：
  - **事实**：`oapi-codegen/runtime` v1.7.0 自己导入 `github.com/google/uuid`，有两处：`runtime/types/uuid.go:4`，和 `runtime` 包本身的 `styleparam.go:30`；`bindparam.go:26` 等又导入 `runtime/types`。生成代码一绑定参数就导入 `runtime`（P3 的 `listApiTokens`、`revokeApiToken`），google/uuid 随之链接进 nerve，M0 的 `TestNerveBinaryLinksNoBannedModule` 必然失败。spike（Go 1.27.1，`go list -deps`）：导入 google/uuid 的恰好是这两个包。
  - **M0 为什么禁它**：M0 设计 3.7（`M0-design.md:248`）和 M0/P3 spec 2.8（`P3-api-contract.md:289`）禁止 google/uuid 进入程序，是一个**替代指标**。真正要防的是生成代码漏了 `format: uuid` 的映射、用上 `openapi_types.UUID`（M0/P3 spec 第 7 节 `:411`，M0-P3 交接 1）。`runtime` 一链接进来，无论映射漏没漏，这个指标都会报，它就失效了。
  - **改为直接检查三件事**：
    1. 传递依赖测试仍然禁止 `github.com/google/uuid`，只有一个例外：程序里导入它的每个包都属于 `github.com/oapi-codegen/runtime` 模块（目前是 `runtime` 和 `runtime/types`）。我们的代码或别的库导入它，测试失败并打印导入链。
    2. archtest 新规则：`adapter/*/gen` 下的生成文件引用 `openapi_types.UUID` 就失败。这才是原意：代码里只有一种 uuid 类型，漏了 `type-mapping` 的模块当场被发现。
    3. depguard 照旧禁止手写代码直接导入 google/uuid。
  - 其余禁止的模块（kin-openapi、testcontainers、docker）不变。3.20 在 P3 改 M0 设计和 M0/P3 spec 的相应文字。
  - 加依赖之后核对 `server/go.mod` 仍是 `go 1.27` / `toolchain go1.27.1`。
  - 每个模块另由 `bodyshapegen` 生成请求体结构表（3.11），与 `server.gen.go` 放在同一个 `gen` 目录。
- **`security` 的写法**（M0-P3 交接 3）：
  - 每个操作都显式声明 `security`：需要登录的写 `[{bearer: []}]`，公开的写 `[]`。
  - `securitySchemes.bearer` 同时写在 `api/openapi.yaml` 和每个模块文件里。
  - `apitest` 的写法检查加两条：每个操作都有 `security`；引用的 scheme 都存在。
- **`x-problem-codes`**：每个操作都写，写法见 3.11。
- **分页的公共组件由 M2 加入**：
  - `common.yaml` 加 `Limit`（1–100，默认 50）、`Cursor` 两个 parameter 和 `NextCursor`（`type: [string, 'null']`）。
  - M0/P3 原计划由 M3 的第一个列表接口加入；M2 的 PAT 列表先用到。
- **游标**（Codex I-9）：
  - `shared` 只定义**封套**：游标是 base64url（不补 `=`）编码的 JSON `{"v": 1, "p": <载荷>}`，`v` 是封套的版本。`shared` 负责编码、解码，解码失败、版本不认识时返回 400 `bad_request`（`errors[{field: cursor, code: invalid_format}]`）。
  - **载荷由每个列表自己定义**，按它自己的排序键，最后总是 `id`，保证顺序稳定、不重不漏。PAT 列表按 `created_at` 倒序，载荷是 `[created_at, id]`。载荷与这个列表的格式不符，同样是 400。
  - 第二稿把 `(created_at, id)` 写成所有列表的游标。M4 的工作项按 `sort_order`、优先级或日期排序，用创建时间截下一页会重复或跳过记录（交接 M4、M7）。
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
M0-P4 交接要求在第一次建表时一次定下。这些约定改变了 Plane 表结构的全局写法，P1 登记到差异清单二·全局（3.20）。

| 事项 | 约定 | 理由 |
|---|---|---|
| 外键的 `ON DELETE` | 照搬每个外键在 Django 模型中的 `on_delete`：`CASCADE` → `ON DELETE CASCADE`，`SET_NULL` → `ON DELETE SET NULL`，`PROTECT` → `ON DELETE RESTRICT`，`DO_NOTHING` → 不写。每个建表的 M 在设计中画出本 M 的表的删除关系图（M2 的见 4.7） | Plane 在 Python 里处理级联，快照中 474 个外键都没有 `ON DELETE`。把它的语义搬进数据库，符合总体设计 5.1 第 3 类改动。软删除的连带删除仍由用例在同一个事务里完成（总体设计 5.5） |
| `DEFERRABLE` | 不用 | Django 需要它来批量插入；Nerve 在同一个事务里按"先父后子"的顺序写 |
| 约束和索引的名字 | 主键、外键、唯一约束，以及只涉及一列、每列至多一个的 CHECK，用 Postgres 的默认名（`<表>_pkey`、`<表>_<列>_fkey`、`<表>_<列>_key`、`<表>_<列>_check`）。**涉及多列的 CHECK，或同一列的第二个 CHECK，必须显式命名**：`<表>_<含义>_check`。一列上的几个条件写成一个 CHECK（用 `AND` 连接）时，仍是这一列唯一的 CHECK，用默认名。索引必须命名：`<表>_<列>_idx`；部分唯一索引写成 `<表>_<列>_key` | 快照中的名字带 Django 的哈希后缀，而且有 29 张表的主键名与表名不对应（M0-P4 交接）。表级 CHECK 的默认名随创建顺序变化：在开发库中用回滚的临时表实测，多列的是 `<表>_check`、`<表>_check1`，同一列的第二个是 `<表>_<列>_check1`。按约束名映射错误、以后 `DROP CONSTRAINT` 都需要稳定的名字 |
| 默认值和取值范围 | 从 `plane/apps/api/plane/db/models/` 读取模型的默认值和取值范围，写进 `DEFAULT` 和 `CHECK`；`id` 不设默认值。这些都是**新加到数据库的**：快照中没有任何列默认值，CHECK 只有正整数字段的 `>= 0`。差异清单逐列写明，不写成"照搬" | 总体设计 5.3、5.5 |
| 审计时间列（Codex I-10） | `created_at`、`updated_at` 由应用在每次插入、每次业务更新时**显式写入**，取自用例的 `Clock`。`DEFAULT now()` 只是应用之外写入时的兜底；数据库不会在 `UPDATE` 时自动改 `updated_at`。仓储的集成测试断言这两列等于固定时钟的时刻；测试用的固定时钟截到微秒（`timestamptz` 的精度），读回的值才能与它逐位相等 | 总体设计 5.5 的自动归档按"超过 `archive_in × 30` 天未更新"判断，必然读 `updated_at`（交接 M4）。时间只有一个来源，固定时钟的测试才和数据库一致 |
| Django 的系统表 | 不照搬 | M0-P4 交接 |
| `*_like`（`varchar_pattern_ops`）索引 | 不建 | 它们只服务 Django 的 `LIKE 'x%'`，Nerve 按等值查找 |
| 外键列的索引 | 只给查询或级联真正用到的外键列建索引；`created_by_id`、`updated_by_id` 不建 | Django 给每个外键都建了 btree，多数用不上 |
| `varchar(n)` 还是 `text` | 照搬 Plane 的列类型；领域层的长度校验与 `n` 一致 | 两者在 Postgres 中性能相同；照搬就不用逐列讨论 |
| 语法 | 只用 sqlc 解析器能解析的 PG 17 语法：不用 PG 18 的 `VIRTUAL` 生成列、`RETURNING old/new`、`WITHOUT OVERLAPS`、`NOT ENFORCED`；不用 `MERGE`（sqlc v1.31.1 会不报错地生成错误的代码）；不写 `DEFAULT uuidv7()` | spike 实测（3.14）；M0-P1 交接 2 |
| JSON 列的 CHECK | 对象型的 `jsonb` 列用 CHECK 保证"是对象、键的集合、每个值的类型"。条件包在 `CASE WHEN jsonb_typeof(col) = 'object' THEN … ELSE false END` 里：`jsonb - text[]` 作用在标量上会报 `cannot delete from scalar`（22023），不包的话违反约束得到的不是 `check_violation`（23514），仓储就不能把它映射为领域错误 | spike 在开发库中实测（4.3） |
| 时间 | 连接池把 `timestamptz` 的扫描时区设为 UTC（pgx 的 `TimestamptzCodec.ScanLocation`，`pgx/v5@v5.11.0/pgtype/timestamptz.go:130-132`），接口输出的时间一律以 `Z` 结尾。做业务判断用的时刻由用例的 `Clock` 提供，作为参数传给 SQL | 总体设计 3.1 要求 UTC；spike 实测 pgx 默认按本地时区返回 |

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
  - **迁移归被改表的模块所有**（Codex I-7、控制者复核 N4）：给别的模块的表加列、加约束的迁移，属于那张表的模块，即使由后来的 M 编写。它是单独的一个文件，按表的所有者命名，可以 `REFERENCES` 编写它的 M 自己的表。
    - 原因：sqlc 处理 `ALTER TABLE` 时先找目标表。spike 实测（用一张代替 M5 文件表的 `assets`）：把 `ALTER TABLE users ADD COLUMN avatar_asset_id uuid REFERENCES assets ON DELETE SET NULL` 放进 `asset` 条目，得到 `relation "users" does not exist`；放进 `identity` 条目，生成通过，`User` 结构多出 `AvatarAssetID`，引用别的模块的表也通过。
    - 范例（M5 的头像，写进 M3、M5 的交接）：M5 先在自己的模块里建 `file_assets`（`<v>_<M5 的模块>_file_assets.sql`），再写 `<v+1>_identity_users_avatar_asset.sql` 给 `users` 加 `avatar_asset_id`、`cover_image_asset_id` 和外键。后者属于 `identity` 的条目，版本号更大，所以执行时被引用的表已经存在。`identity` 的查询由 M5 按需补充。
    - 总体设计 5.6"指向更晚建立的模块的外键，由后建的模块用 `ALTER TABLE` 补上"随之写明：由后建的 M 编写，文件归被改表的模块（3.20）。
    - 这条规则不能让后续 M 不加思考地照做：跨模块的联表查询、物理删除时的外键链，仍要由各 M 的设计逐个写明（交接 M3、M4、M6、M7）。
  - 由一个架构测试守住配置本身（`archtest` 中的 `TestSQLCSchemaScope`）：
    - 迁移文件名是 `<版本>_<模块>_<内容>.sql`；
    - 每个模块的迁移只出现在这个模块的条目里，不出现在别的条目里；
    - 迁移中每个 `ALTER TABLE` 的目标表，由文件名中的模块建立（由 `CREATE TABLE` 找到表的所有者）；
    - `river` 开头的迁移不属于任何模块的条目（模块通过 River 客户端使用它，不查它的表）；
    - 有 `adapter/postgres/queries` 的模块都有条目。
  - 需要读别的模块的数据时，按总体设计 6.3 规则 2 在自己的 `app` 层声明端口。确实需要在 SQL 中联表的，由那个 M 的设计写明理由，作为测试中列出的例外。M3 的 `access` 模块读成员表是第一个要回答的问题（交接）。
- **写法约定**（来自 spike）：
  - 行比较的游标条件 `(created_at, id) < ($1, $2)` 会让第二个参数被推断成时间类型，运行时报错。写成 `(created_at, id) < (sqlc.arg(cursor_created_at)::timestamptz, sqlc.arg(cursor_id)::uuid)`。
  - PATCH 只更新传入的字段（总体设计 3.6"同一字段并发修改时以后写入的为准"），写法是 `col = CASE WHEN sqlc.arg(set_col)::boolean THEN sqlc.arg(col) ELSE col END`。
  - **部分更新的 JSON 对象**在 SQL 中原子合并（Codex I-8）：`onboarding_step = onboarding_step || sqlc.arg(onboarding_step_patch)::jsonb … RETURNING *`，没传时 `patch` 是 `{}`。领域层先校验这个部分对象（键都是已知的四个，值都是布尔值）。spike 实测：一个事务改 `profile_complete` 后持有行锁，另一个同时改 `workspace_create`，后者等锁、在最新的行上重新求值，两个键都保留下来；"读出、在应用里合并、整体写回"的写法丢掉了先写的那个键。
  - 不用"读出整行、改完整行写回"：那会覆盖别人同时改的其他字段。
  - 时间比较用参数，不用 `now()`；审计列的时刻也是参数（3.13）。
  - 子查询引用同一张表时要起别名：清理任务的 `DELETE FROM auth_sessions WHERE id IN (SELECT … FROM auth_sessions …)` 不起别名，sqlc 报 `column reference "expires_at" is ambiguous`。
  - 以上查询（续期的查找和轮换、按账户撤销会话、`||` 合并、游标分页、行锁、`SKIP LOCKED` 的分批清理）都经 spike 用 `CGO_ENABLED=0` 的 sqlc 生成通过。
- **Makefile 与持续集成**：
  - `make gen-go` 在 oapi-codegen 和 `bodyshapegen` 之后执行 `CGO_ENABLED=0 go tool -modfile=tools/go.mod sqlc generate`，先删掉旧的 `adapter/postgres/gen/*.go`。
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
  - 它删除 `expires_at` 早于用例时钟当前时刻的会话，每批最多 1000 行，行由 `SELECT … FOR UPDATE SKIP LOCKED` 取得：正被重置、续期锁住的会话跳过，下一轮再删，所以不会和 3.5 的加锁顺序形成死锁。
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
五个命令都在 `nerve users` 下，都按 3.5 的账户行锁执行；需要密码的，在终端上不回显地提示输入两次（`golang.org/x/term`），标准输入不是终端时读一行，供脚本和端到端测试使用。邮箱一律按注册时的规则规范化（Plane 的 `reset_password` 按原样匹配，这里修正）。账户不存在、邮箱已被使用、密码不合规时，退出码为 1，输出一句说明，数据库不变。

| 命令 | 做什么 | 输出 |
|---|---|---|
| `create --email <邮箱>`（决策点 2） | 建账户和资料，不建会话。与注册共用"建账户"这一步（校验邮箱、按 3.8 检查密码、哈希、插入 `users` 和 `profiles`），不经过 `SignupPolicy`：注册关闭时，服务器管理员仍能建账户 | `created user <email>` |
| `reset-password --email <邮箱>` | 按 3.8 检查新密码，改写哈希；撤销全部会话（`password_reset`）和**全部 PAT**（3.5） | `password reset for <email>: revoked <n> sessions, <m> API tokens` |
| `set-email --email <旧> --new-email <新>`（决策点 1） | 新邮箱按注册时的规则校验格式；改写 `users.email`，撤销全部会话（`email_changed`）。新旧规范化后相同也是退出码 1，说明没有变化。**PAT 不撤销**：它不是账户被盗后的恢复手段；如果改邮箱与安全有关，另外执行 `reset-password`（命令的说明和 README 写明） | `email changed from <旧> to <新>: revoked <n> sessions` |
| `deactivate --email <邮箱>`（决策点 3） | 与自助停用同一个用例：`is_active = false`，撤销全部会话（`deactivated`），重置新手引导，不改密码 | `deactivated <email>: revoked <n> sessions` |
| `activate --email <邮箱>`（决策点 3） | `is_active = true`。账户的 PAT 没有被删除，恢复后**重新可用**；怀疑账户被盗时，另外执行 `reset-password`（命令的说明和 README 写明） | `activated <email>: <m> API tokens are usable again` |

- 没有界面，也没有接口的命令：`create`、`reset-password`、`set-email`、`activate`。个人设置里的邮箱只读（7.7）。
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
  - 这条交接按"服务端没有跳转"关闭，写进 P4 的 review。
- **校验规则补上控制字符**：`isValidNextPath` 另外拒绝含控制字符（`\u0000`–`\u001f`、`\u007f`）的值。单元测试覆盖 `//`、`\`、协议、控制字符。
- **带上查询参数和片段**（M1-P4 交接"由 M2 决定"第 1 项）：
  - 跳到登录页时，`next_path` 取 `pathname + search + hash`，整体 `encodeURIComponent`。
  - 校验只看开头：以单个 `/` 开头，不是 `//`、`/\`。
  - 查询和片段在浏览器里只作用于本站的页面，不影响跳转目标的主机。
  - 现在 401 拦截器拼出的地址既不编码也丢掉查询参数（`web/apps/web/core/services/api.service.ts:42`），随令牌管理器一起重写。

### 3.19 新手引导的"角色""用途"两步：删除（决策点 4 已裁定为 A）
- **事实**：
  - Plane 把 `IS_SELF_MANAGED` 写死为 `True`（`plane/apps/api/plane/settings/common.py:54`），为真时新手引导跳过"角色""用途"两步（`web/apps/web/core/components/onboarding/root.tsx:74`，进度条和返回按钮在 `header.tsx:45,57`）。Plane 自托管版的用户从来看不到这两步。
  - Nerve 只有自托管一种形态。
- **做法**：
  - 前端（P4）：删除 `steps/role/`、`steps/usecase/`（共 338 行），以及 `onboarding/root.tsx`、`onboarding/header.tsx` 中读 `is_self_managed` 的分支；资料步骤固定走自托管的那条路。删除 general 页在提交时把 `role` 的默认值"Product / Project Manager"写回去的逻辑（`web/apps/web/core/components/settings/profile/content/pages/general/form.tsx:77,146-151`；那里没有输入框）。
  - 数据与接口（P1）：实例配置不定义 `is_self_managed`；`profiles.role`、`use_case` 不建，`Profile`、`ProfileUpdate` 中没有它们（4.3、5.2）。

### 3.20 需要同步到上级文档的地方
**规则**（Codex M-8）：每处偏离，在**实现它的那个 Phase** 的同一次合并中同步到上级文档和差异清单；建表、改表的 Phase 逐列更新差异清单。收尾只按本表逐行核对。

| Phase | 文档 | 位置 | 内容 |
|---|---|---|---|
| P1 | 总体设计 | 3.1 | 注册返回令牌，不返回新建的账户（账户由 `GET /me` 读取，5.1） |
| P1 | 总体设计 | 3.5 | 平台错误码加入 `unauthorized`、`payload_too_large`、`validation_failed`、`server_busy`；`FieldError` 带 `code`；结构不合契约是 400、取值不合规是 422（3.11）；每个操作用 `x-problem-codes` 声明错误码 |
| P1 | 总体设计 | 4.1 | 刷新令牌的格式：会话 id、代数、随机密文和 HMAC 标签，MAC 密钥从签名密钥派生；数据库只存当前一代密文的哈希（3.4、3.5） |
| P1 | 总体设计 | 4.2 | 是否开放注册：prod 默认关闭，dev、test 默认开放；第一个账户用 `nerve users create`（决策点 2） |
| P1 | 总体设计 | 5.5 | 审计时间列由用例的时钟显式写入（3.13） |
| P1 | 总体设计 | 5.6 | 改别的模块的表的迁移由后建的 M 编写，文件归被改表的模块（3.14） |
| P1 | 总体设计 | 6.2 | 端口由使用方在 `app` 层声明；`shared` 的内容是 Actor、领域错误、`TxManager`；时钟端口由各模块自己声明（3.3） |
| P1 | 总体设计 | 6.2（`module.go` 一行） | `Register(mux, apiErrors)` 改为 `Register(router, api)`，`api`（`httpserver.API`）提供错误映射和按路由的中间件；`PublicOperations()` 列出不需要令牌的操作；其他模块或 `bootstrap` 要用的能力由访问方法导出，例如 `Authenticator()`（3.3、3.6；P1 补登） |
| P1 | 总体设计 | 6.3 | 架构测试：平台不导入 `internal/shared`；sqlc 按模块限定 `schema`，`ALTER TABLE` 按表的所有者（3.3、3.14） |
| P1 | 总体设计 | 6.4 | 按路由挂载的中间件及其顺序（P1 的部分：请求元信息、期限、请求体上限、认证、请求体结构）；参数在它们之前绑定（3.6、3.11）；`TxManager` 的 `COMMIT` 不受请求期限的取消，有自己的期限（3.6） |
| P1 | 总体设计 | 8.2 | 失败时保存 trace、截图、nerve 日志和数据库快照，不录像（9.5） |
| P1 | 总体设计 | 8.2（fixture 目录） | `server.ts` 由"随机端口"改为在 `127.0.0.1:0` 上监听、从 `server.addr_file` 读出实际地址，需要另一种配置的故事在同一个库上另起一个 nerve；`db.ts` 每个进程一个连接池；断言函数移到 `assert/`，按表分文件；加上 `auth.ts`（9.5；P1 补登） |
| P1 | M0 设计 | 3.1 | 包职责表：`platform` 不能依赖 `modules`、`bootstrap` 和 `internal/shared`；`module.go` 的入口改为 `Register(router, api)` 和 `PublicOperations()`；`internal/shared` 不再是"预计在 M2"，由 P1 建立（`Actor`、`Error`、`TxManager`）（3.3、3.6；P1 补登） |
| P1 | M0 设计 | 3.3 | 按路由挂载的中间件及其顺序（3.6） |
| P1 | M0 设计 | 3.3（中间件以外） | 路由由 `httpserver.Router` 包一层，记下注册的模式（3.6）；`/healthz`、`/readyz` 的访问日志降为 DEBUG（6.1）；平台错误码加入 `unauthorized`、`payload_too_large`、`validation_failed`、`server_busy`；生成代码的错误出口不再是 `InternalError`：参数绑定失败 → `BadRequest`，请求体解码失败 → `BodyError`（超过上限是 413），handler 出错 → `APIErrors.Write`（按 `ProblemError` 映射，其余 500，`context.Canceled` 不算 500）（3.11；P1 补登） |
| P1 | M0 设计 | 3.5 | `//go:embed all:sql` 改为 `//go:embed sql/*.sql`，删掉占位的 `sql/.gitkeep`；"M0 不包含任何迁移文件"改为迁移文件从 M2 开始：P1 加入 `00001`–`00003`，`server/migrations/schema_test.go` 核对 up、down、再 up，以及约束名和 CHECK（4.1；P1 补登） |
| P1 | 差异清单 | 一 B | `sessions` 由 `auth_sessions` 替代（替换模型，不逐列继承） |
| P1 | 差异清单 | 二·全局 | 3.13 的全局约定：外键的 `ON DELETE` 写进数据库；不用 `DEFERRABLE`；约束和索引一律改名；不建 `*_like` 索引和多数外键列的索引；审计时间列由应用写入 |
| P1 | 差异清单 | 二·按表 | `users`、`profiles` 逐列（4.2、4.3）；`auth_sessions` 按实际的列改写原来那一行（4.5） |
| P1 | 差异清单 | 三 | 错误码在接口描述中逐个操作声明；请求体按契约拒绝未知字段、不合法的 `null` 和缺少的必填字段 |
| P1 | 差异清单 | 四 | 4.6 中标 P1 的行 |
| P2 | 总体设计 | 3.5 | `rate_limited` |
| P2 | 总体设计 | 3.6 | 限流的桶、突发和默认值；认证之前的失败闸门（3.10） |
| P2 | 总体设计 | 4.1 | 刷新令牌的 30 天是从登录起算的绝对期限，续期不延长（3.5） |
| P2 | 总体设计 | 4.2 | 退出只结束当前会话（负责人已批准，11.1）；重复使用检测只在交出的旧令牌确属这个会话时才撤销（3.5） |
| P2 | 总体设计 | 6.4 | 固定链由三个中间件变为四个（加上安全响应头，8.3）；按路由加上限流 |
| P2 | M0 设计 | 3.3 | 固定链由三个中间件变为四个 |
| P2 | 差异清单 | 四 | 4.6 中标 P2 的行 |
| P3 | 总体设计 | 3.1 | 修改密码、停用账户返回 204（5.1） |
| P3 | 总体设计 | 3.4、6.2 | 游标的封套由 `shared` 定义（6.2 中 `shared` 的内容随之加上它），载荷由各列表按自己的排序定义（3.12） |
| P3 | 总体设计 | 4.2 | 修改密码结束其他会话；停用结束全部会话，恢复后 PAT 重新可用；管理员重置密码同时撤销全部 PAT（3.5）；凭证的签发与变更按账户行锁（3.5） |
| P3 | M0 设计 | 3.2 | 原文"M2 加入 `signup_enabled`，M5 加入文件大小上限"改为：M2 加入 `signup_enabled`、`workspace_creation_enabled`、`file_size_limit`（5.3） |
| P3 | M0/P3 spec | 第 7 节 | 分页的公共组件由 M2 加入（3.12） |
| P3 | M0 设计 | 3.7（传递依赖测试，`M0-design.md:248`） | 禁止 `github.com/google/uuid` 的真实意图是"生成代码不漏 uuid 的映射"：google/uuid 只允许由 `oapi-codegen/runtime` 模块的包导入；另加规则，生成文件不得引用 `openapi_types.UUID`（3.12） |
| P3 | M0/P3 spec | 2.8（`:289`）、第 7 节（`:411`） | 同上；交接表中 `format: uuid` 一行"漏掉时 archtest 的传递依赖测试失败"改为"漏掉时生成代码引用 `openapi_types.UUID`，archtest 的新规则失败"（3.12） |
| P3 | 差异清单 | 二·按表 | `api_tokens` 逐列（4.4）；River 的表登记为新增的基础设施表 |
| P3 | 差异清单 | 三 | PAT 撤销的路径 `/api-tokens/{id}` |
| P3 | 差异清单 | 四 | 4.6 中标 P3 的行 |
| P4 | 总体设计 | 4.3 | 没有 `navigator.locks` 时用 localStorage 租约；`nerve.auth` 记录带 `login_id`；只有续期得到 401 才结束会话；"会话暂不可用"（7.1） |
| P4 | 前端改动清单 | 3.1 | CSRF 一行已完成 |
| P5 | 前端改动清单 | 3.2 | M2 一行已完成 |
| 收尾 | 总体设计 | 9.4 | M2 的状态 |
| 收尾 | 以上全部 | — | 按本表逐行核对；差异清单按第 4 节逐列总核对 |

---

## 4. 数据模型

起点是 `tools/plane-schema/plane-v1.4.2-schema.sql`，按 3.13 的约定修改。每一处改动在建表的 Phase 逐列登记到差异清单（3.20）。下面"与 Plane 的差异"一栏里，"照搬"只指类型、可空性和约束与快照相同；快照中没有任何列默认值，CHECK 只有正整数字段的 `>= 0`，所以写进数据库的默认值和 CHECK 都标为"新加"（Codex M-8）。

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
| `id` | `uuid PRIMARY KEY` | 类型照搬；主键名是默认的 `users_pkey`（Plane 是 `user_pkey`）；不设默认值，由应用生成 |
| `email` | `varchar(255) NOT NULL UNIQUE`，`CHECK (email = lower(email) AND email !~ '[[:space:]]')` | Plane 可以为空；唯一约束照搬（名字是 `users_email_key`，Plane 是 `user_email_key`）。CHECK 新加：规范化原来只在 Python 里做（`email.strip().lower()`，`plane/apps/api/plane/authentication/views/app/email.py:73`） |
| `password` | `varchar(128) NOT NULL` | 列名和类型照搬；内容改为 argon2id 的 PHC 字符串（约 97 个字符） |
| `first_name`、`last_name` | `varchar(255) NOT NULL DEFAULT ''` | 类型照搬；`DEFAULT ''` 新加（模型的空串默认值） |
| `display_name` | `varchar(255) NOT NULL`，`CHECK (display_name <> '')` | 类型照搬；CHECK 新加。Plane 的 `save()` 在它为空时用邮箱 @ 之前的部分填上（`plane/apps/api/plane/db/models/user.py:169-187`），数据库不约束。注册时同样取邮箱 @ 之前的部分 |
| `user_timezone` | `varchar(255) NOT NULL DEFAULT 'UTC'` | 类型照搬；默认值新加（模型默认值）。Plane 限定为 `pytz.common_timezones`；Nerve 接受 Go 的 `time.LoadLocation` 认得的 IANA 名称，`Local` 除外；程序内嵌 `time/tzdata`。登记为差异（4.6） |
| `is_active` | `boolean NOT NULL DEFAULT true` | 类型照搬；默认值新加 |
| `created_at`、`updated_at` | `timestamptz NOT NULL DEFAULT now()` | 类型照搬；由应用按时钟写入，`DEFAULT now()` 是新加的兜底（3.13） |

**邮箱的 CHECK 保证什么**（Codex M-3）：
- 领域层的规范化是 `strings.ToLower(strings.TrimSpace(s))`；格式按 Plane 使用的 Django `validate_email` 的规则（`plane/apps/api/plane/authentication/adapter/base.py:79`），不允许任何空白和控制字符，长度不超过 255。
- CHECK 保证：在数据库的排序规则下已是小写（`lower(email) = email`；开发库和 Postgres 官方镜像是 `en_US.utf8`，`lower` 按 Unicode 转换）；不含任何被数据库的字符类别认作空白的字符（`[[:space:]]`：ASCII 的空格、`\t`、`\n`、`\v`、`\f`、`\r`，以及 U+3000、U+2028 这类 Unicode 空白），首尾和中间都不行。第二稿只有 `email = lower(email)`，挡不住首尾空白；`btrim` 默认只去空格，挡不住制表符和换行，所以不用它。
- 领域层写入的值总能通过 CHECK；CHECK 挡住的是绕过应用直接写库的大写和空白。唯一的出入是 U+00A0（不换行空格）：glibc 不把它算作 `[[:space:]]`，而 Go 的 `TrimSpace` 会去掉它、格式校验也拒绝它，所以只影响绕过应用的写入。
- spike 在开发库（PG 18.6，`en_US.utf8`）的回滚事务中实测：`alice@corp.com`、`élodie@exämple.com` 通过；大写的 ASCII 和非 ASCII、首尾的空格、制表符、换行、回车、中间的空格、开头的 U+3000、结尾的 U+2028 都得到 `check_violation`。约束名是默认的 `users_email_check`。

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
- **由 M5 加入**（3.2）：`avatar_asset_id`、`cover_image_asset_id`。迁移归 `identity`，由 M5 编写（3.14）。在那之前，接口的 `avatar_url`、`cover_image_url` 是 `null`，不需要列。

名字不能含网址：Plane 对 `first_name`、`last_name` 用 `contains_url` 检查（`plane/apps/api/plane/app/serializers/user.py:16-24`，正则在 `plane/utils/url.py:12-53`），照搬到领域层。

### 4.3 `profiles`（Plane 29 列 → 11 列）
| 列 | 类型与约束 | 与 Plane 的差异 |
|---|---|---|
| `id` | `uuid PRIMARY KEY` | 类型照搬 |
| `user_id` | `uuid NOT NULL UNIQUE REFERENCES users ON DELETE CASCADE` | 类型和唯一约束照搬（`OneToOne`）；`ON DELETE CASCADE` 来自模型（3.13） |
| `theme` | `varchar(20) NOT NULL DEFAULT 'system'`，`CHECK (theme IN ('system','light','dark','light-contrast','dark-contrast'))` | Plane 是 `jsonb`，默认 `{}`，存自定义调色板。自定义主题已删除（M1-P2 交接），只剩一个值，改为字符串；默认值和 CHECK 新加 |
| `is_tour_completed` | `boolean NOT NULL DEFAULT false` | 类型照搬；默认值新加 |
| `onboarding_step` | `jsonb NOT NULL DEFAULT '{"profile_complete": false, "workspace_create": false, "workspace_invite": false, "workspace_join": false}'`，CHECK 见下 | 类型照搬；默认值新加（模型的 `get_default_onboarding()`）；CHECK 新加 |
| `is_onboarded` | `boolean NOT NULL DEFAULT false` | 类型照搬；默认值新加 |
| `last_workspace_id` | `uuid`（可空，不是外键） | 照搬（3.2） |
| `language` | `varchar(255) NOT NULL DEFAULT 'en'`，`CHECK (language IN ('en','zh-CN'))` | 类型照搬；Plane 接受任意字符串，Nerve 只剩两种语言（M1 设计第 6 节）；默认值和 CHECK 新加 |
| `start_of_the_week` | `smallint NOT NULL DEFAULT 0`，`CHECK (start_of_the_week BETWEEN 0 AND 6)` | 类型照搬；快照只有 `>= 0`，`<= 6` 来自模型的 choices，新加；默认值新加 |
| `created_at`、`updated_at` | `timestamptz NOT NULL DEFAULT now()` | 类型照搬；由应用写入（3.13） |

**`onboarding_step` 的 CHECK**（Codex M-3）：一列上的一个 CHECK，默认名 `profiles_onboarding_step_check`。
```sql
CHECK (CASE WHEN jsonb_typeof(onboarding_step) = 'object' THEN
    onboarding_step ?& array['profile_complete', 'workspace_create', 'workspace_invite', 'workspace_join']
    AND (onboarding_step - array['profile_complete', 'workspace_create', 'workspace_invite', 'workspace_join']) = '{}'::jsonb
    AND jsonb_typeof(onboarding_step -> 'profile_complete') = 'boolean'
    AND jsonb_typeof(onboarding_step -> 'workspace_create') = 'boolean'
    AND jsonb_typeof(onboarding_step -> 'workspace_invite') = 'boolean'
    AND jsonb_typeof(onboarding_step -> 'workspace_join') = 'boolean'
ELSE false END)
```
- 它保证：是对象；恰好这四个键（`?&` 保证都在，减去它们之后为空保证没有多余的键）；每个值都是布尔值。第二稿只保证四个键都在，数据库会接受 `null`、字符串和多余的键。
- 外层的 `CASE` 让数组、标量和 JSON `null` 也得到 `check_violation`，而不是 `cannot delete from scalar`（3.13）。
- spike 在开发库中实测：默认值、合并一个键、合并空对象通过；数组、标量 `5`、JSON `null`、缺一个键、多一个键、值为字符串、`null`、数字都得到 `check_violation`；`||` 合并后四个键齐全。sqlc（`CGO_ENABLED=0`）能解析这个 CHECK 和 `||` 的合并查询。
- 更新时由领域层先校验传入的部分对象，再在 SQL 中合并（3.14）。

删除的 18 列：
- 新手引导的"角色""用途"（决策点 4 已裁定为 A，3.19）：`role`、`use_case`。
- 账单与公司：`billing_address_country`、`billing_address`、`has_billing_address`、`company_name`。
- 移动端：`is_mobile_onboarded`、`mobile_onboarding_step`、`mobile_timezone_auto_set`。
- 前端类型 `TUserProfile` 中没有、Nerve 不用的 Plane 新功能：`is_smooth_cursor_enabled`、`is_app_rail_docked`（应用栏已在 M1 收尾删除）、`background_color`、`goals`、`is_navigation_tour_completed`、`product_tour`、`notification_view_mode`。
- 营销邮件、更新日志：`has_marketing_email_consent`、`is_subscribed_to_changelog`。

### 4.4 `api_tokens`（Plane 17 列 → 12 列）
| 列 | 类型与约束 | 与 Plane 的差异 |
|---|---|---|
| `id` | `uuid PRIMARY KEY` | 类型照搬 |
| `user_id` | `uuid NOT NULL REFERENCES users ON DELETE CASCADE` | 类型照搬；`ON DELETE` 来自模型 |
| `token_hash` | `bytea NOT NULL UNIQUE`，`CHECK (octet_length(token_hash) = 32)` | 原来的 `token` 存明文（差异清单已登记）；CHECK 新加 |
| `label` | `varchar(255) NOT NULL`，`CHECK (label <> '')` | 类型照搬；CHECK 新加。不传时由应用生成 32 位十六进制（Plane 的 `uuid4().hex`） |
| `description` | `text NOT NULL DEFAULT ''` | 类型照搬；默认值新加 |
| `expired_at` | `timestamptz`（可空，空表示永不过期） | 照搬。创建时必须晚于当前时间（Plane 不检查，属于缺陷） |
| `last_used` | `timestamptz` | 照搬；写入频率见 3.5 |
| `created_by_id`、`updated_by_id` | `uuid REFERENCES users ON DELETE SET NULL` | 类型照搬；`ON DELETE` 来自 `UserAuditModel` 的 `SET_NULL` |
| `created_at`、`updated_at` | `timestamptz NOT NULL DEFAULT now()` | 类型照搬；由应用写入（3.13） |
| `deleted_at` | `timestamptz` | 照搬（撤销就是软删除；管理员重置密码时一次软删除全部） |

- **删除的列**：
  - `user_type`、`workspace_id`：总体设计 5.3；Plane 自己也已把个人令牌的 `workspace_id` 置空（`plane/apps/api/plane/db/migrations/0115_auto_20260105_1406.py:31-38`）。
  - `is_active`：Plane 的列表把它算成"未过期"，撤销用软删除，这一列没有独立的写入方。
  - `is_service`：社区版没有代码创建它。
  - `allowed_rate_limit`：没有任何限流读取它。
- **索引**：`api_tokens_user_id_created_at_idx ON (user_id, created_at DESC, id DESC) WHERE deleted_at IS NULL`，用于列表。

### 4.5 `auth_sessions`（新增，替代 Plane 的 `sessions`）
Plane 的 `sessions`（`session_key`、`session_data`、`expire_date`、`device_info`、`user_id`）是 Django 的会话存储，不逐列继承。差异清单一 B 登记为"替换模型"：由 `auth_sessions` 替代（3.20）。

**`auth_sessions`**（12 列）：一次登录一行，大小固定。

| 列 | 类型与约束 | 说明 |
|---|---|---|
| `id` | `uuid PRIMARY KEY` | 访问令牌中的 `sid`，也写在刷新令牌里 |
| `user_id` | `uuid NOT NULL REFERENCES users ON DELETE CASCADE` | |
| `token_hash` | `bytea NOT NULL`，`CHECK (octet_length(token_hash) = 32)` | 当前这一代刷新令牌密文的 SHA-256 |
| `generation` | `integer NOT NULL DEFAULT 0`，`CHECK (generation >= 0)` | 当前这一代的代数：每续期一次加 1（3.5） |
| `user_agent` | `text NOT NULL DEFAULT ''` | 登录时的 UA，截断到 512 个字符 |
| `ip` | `inet` | 登录时的客户端 IP，完整的地址（3.10） |
| `expires_at` | `timestamptz NOT NULL` | 登录时设为登录时刻加 `auth.session_ttl`，之后不变（3.5） |
| `last_refreshed_at` | `timestamptz` | |
| `revoked_at` | `timestamptz` | |
| `revoke_reason` | `varchar(20)`，`CHECK (revoke_reason IN ('logout','password_changed','password_reset','email_changed','deactivated','reuse_detected'))` | 每个值的写入方见 3.5 的撤销规则 |
| `created_at`、`updated_at` | `timestamptz NOT NULL DEFAULT now()` | 由应用写入（3.13） |

- **表级约束**：`CONSTRAINT auth_sessions_revoked_consistent_check CHECK ((revoked_at IS NULL) = (revoke_reason IS NULL))`，显式命名（3.13）。
- **索引**：`auth_sessions_user_id_idx`（撤销某个账户的全部会话），`auth_sessions_expires_at_idx`（清理任务）。续期、退出按主键查找。
- **旧代令牌不存**：它们由令牌里的标签认出（3.4、3.5），所以一个会话无论续期多少次都只有这一行。第三稿一度为每一代另建一张 `auth_refresh_tokens` 表（Codex I-2 的选项①），它的行数只受按 IP 的限流约束，已按选项②删除（3.5）。
- **与差异清单原来那一行的对照**：差异清单写的是"刷新令牌的哈希、令牌轮换链（用于重复使用检测）、UA、IP、过期和撤销时间"。"轮换链"在这里是代数加上令牌里的标签，不是一串存下来的记录；P1 按实际的列改写这一行。

### 4.6 行为差异（登记到差异清单第四节）
"Phase"一栏是实现它、同时登记它的 Phase（3.20）。

| 行为 | Plane | Nerve | Phase |
|---|---|---|---|
| 密码规则 | 服务端在注册、修改密码、重置命令三处用 zxcvbn 评分 ≥ 3；组合规则只在界面上 | 服务端在注册、修改密码、创建账户和重置两个命令四处执行组合规则（长度 8–128）、NCSC 前 10 万常见密码名单和"主干不能是邮箱前缀的主干"（3.8） | P1 |
| 注册的前提 | 实例必须先由实例管理员完成设置 | 没有这一步 | P1 |
| 注册默认是否开放 | `ENABLE_SIGNUP` 默认开放 | prod 默认关闭，dev、test 默认开放；关闭时先答"注册已关闭"，不查邮箱；第一个账户用 `nerve users create`（决策点 2） | P1 |
| 请求中的未知字段和不合法的 `null` | DRF 的序列化器忽略未知字段 | 按契约返回 400（3.11） | P1 |
| 密码哈希过载 | 无并发上限 | 最多 4 个同时计算，等待 2 秒仍拿不到名额时 503 `server_busy`（3.8） | P1 |
| 登录时邮箱不存在 | 返回 `USER_DOES_NOT_EXIST` | 与密码错误相同的 401，耗时也相同（3.9） | P2 |
| 会话的期限 | Django 会话，从登录起固定 7 天（`SESSION_COOKIE_AGE`），请求不延长 | 访问令牌 15 分钟；会话从登录起 30 天，续期不延长；刷新令牌每次使用后换新，并检测重复使用（3.5） | P2 |
| 限流 | 认证接口合计每 IP 10/min，匿名 30/min，API Key 60/min；`/api/v1` 的响应带 `X-RateLimit-Remaining`、`X-RateLimit-Reset`（`plane/apps/api/plane/api/views/base.py:120-126`） | 3.10 的七个桶；超出时 429 带 `Retry-After`，不加 `X-RateLimit-*` | P2 |
| 无效的 PAT | 403（`AuthenticationFailed` 没有 `authenticate_header`） | 401 `unauthorized` | P3 |
| 退出、修改密码、停用后旧凭证失效 | 其他会话在下一个请求时失效 | 相同，由每个请求的会话检查做到（3.5） | P3 |
| 管理员重置密码 | 只改密码（会话随之失效），PAT 不动 | 撤销全部会话和全部 PAT，输出撤销的数量（3.5、3.17） | P3 |
| `reset_password` 命令的邮箱 | 按原样匹配 | 按注册时的规则规范化 | P3 |
| 修改登录邮箱 | 用户在个人设置中向新邮箱索取验证码后修改（M1 已删除这个流程） | 只能由服务器管理员用 `nerve users set-email` 修改，撤销该账户的全部会话，PAT 不撤销（决策点 1、3.17） | P3 |
| 停用账户 | 自助停用：撤销会话、重置新手引导、把密码改成随机值、发邮件；"唯一管理员"的检查从不拒绝；只有命令 `activate_user` 能恢复 | 自助停用（`POST /me/deactivate`）和管理员命令 `deactivate`、`activate`；撤销全部会话，重置新手引导，不改密码，PAT 不删除但停用期间认证失败；"唯一管理员"的检查由 M3 在同一个事务里实现（决策点 3） | P3 |
| 创建账户的命令 | 没有（第一个账户通过实例设置页创建） | `nerve users create`（决策点 2） | P3 |
| PAT 的管理 | 只能用 Cookie 会话管理 PAT | 任何凭证都能管理，包括 PAT 本身（总体设计 0.2 原则 2） | P3 |
| PAT 的 `last_used` | 每个请求都写 | 每分钟最多写一次 | P3 |
| PAT 的名称和过期时间 | 不校验：名称过长时变成 500，过期时间可以是过去 | 名称 1–255 个字符；过期时间必须在未来 | P3 |
| PAT 的编辑 | `PATCH` 可以改名称和说明，响应中带令牌原文 | 不提供（界面没有编辑入口）；令牌原文只在创建时返回一次 | P3 |
| 时区 | 只接受 `pytz.common_timezones` | 接受任何 IANA 名称（`Local` 除外）；时区列表接口给的仍是同一份常用列表（5.3） | P3 |

### 4.7 删除关系图（M2 的表）
按 3.13，每个建表的 M 画出本 M 的表的删除关系。
```
users
 ├── profiles.user_id                     ON DELETE CASCADE
 ├── auth_sessions.user_id                ON DELETE CASCADE
 ├── api_tokens.user_id                   ON DELETE CASCADE
 ├── api_tokens.created_by_id             ON DELETE SET NULL
 └── api_tokens.updated_by_id             ON DELETE SET NULL
```
- M2 从不物理删除账户：停用只改 `is_active`。M2 的物理删除只有一处：清理任务删除过期的会话。
- M4 的"物理删除软删除超过 60 天的数据"会删到 `api_tokens`（交接）。
- 以后引用 `users` 的表（成员、工作项的创建人和负责人、评论……）由各自的 M 在这张图上延伸，写明物理删除时每条外键的去向（交接 M3、M4、M6）。

---

## 5. 接口

### 5.1 操作
模块文件 `api/modules/identity.yaml`（新建）和 `api/modules/instance.yaml`（扩展）。路径都不带结尾 `/`（M1-P3 交接最后一条）。每个操作都写 `x-problem-codes`（3.11）；下表"主要错误"一栏就是它的内容，不再列出所有操作都可能返回的四个码。其中 `bad_request` 包括请求体的结构不合契约（未知字段、不合法的 `null`、缺少必填字段），每个带请求体的操作都可能返回。

| 方法与路径 | operationId | 认证 | 请求体 | 成功 | 主要错误 |
|---|---|---|---|---|---|
| `POST /api/v0/auth/register` | `register` | 公开 | `RegisterRequest {email, password}` | 201 `AuthTokens` | 403 `identity.signup_disabled`（结构合乎契约的请求在查看邮箱和密码之前就得到它；平台链的 400 结构错误和 413 可以在它之前，3.9）；422 `validation_failed`；409 `identity.email_taken`；503 `server_busy` |
| `POST /api/v0/auth/login` | `login` | 公开 | `LoginRequest {email, password}` | 200 `AuthTokens` | 401 `identity.invalid_credentials`；403 `identity.account_deactivated`；503 `server_busy`。缺字段是 400（结构） |
| `POST /api/v0/auth/refresh` | `refreshTokens` | 公开 | `RefreshRequest {refresh_token}` | 200 `AuthTokens` | 401 `identity.refresh_token_invalid` |
| `POST /api/v0/auth/logout` | `logout` | 公开 | `LogoutRequest {refresh_token}` | 204 | 无。令牌未知、已过期、已撤销、不是当前一代时也返回 204，不泄露它的状态（3.5） |
| `GET /api/v0/me` | `getMe` | bearer | — | 200 `User` | — |
| `PATCH /api/v0/me` | `updateMe` | bearer | `UserUpdate` | 200 `User` | 422 `validation_failed` |
| `POST /api/v0/me/change-password` | `changePassword` | bearer | `ChangePasswordRequest {current_password, new_password}` | 204 | 422 `validation_failed`（新密码不合规）；422 `identity.current_password_incorrect`；503 `server_busy` |
| `POST /api/v0/me/deactivate` | `deactivateMe` | bearer | — | 204 | 无。M3 加入"唯一管理员"时的错误码（13.2） |
| `GET /api/v0/me/profile` | `getProfile` | bearer | — | 200 `Profile` | — |
| `PATCH /api/v0/me/profile` | `updateProfile` | bearer | `ProfileUpdate` | 200 `Profile` | 422 `validation_failed` |
| `GET /api/v0/me/api-tokens` | `listApiTokens` | bearer | 查询参数 `limit`、`cursor` | 200 `ApiTokenPage` | 422 `validation_failed`（`limit` 超出 1–100）；400 `bad_request`（游标不合法；`limit` 不是整数） |
| `POST /api/v0/me/api-tokens` | `createApiToken` | bearer | `ApiTokenCreate {label?, description?, expired_at?}` | 201 `ApiTokenCreated` | 422 `validation_failed` |
| `DELETE /api/v0/api-tokens/{token_id}` | `revokeApiToken` | bearer | — | 204 | 404 `identity.api_token_not_found` |
| `GET /api/v0/instance` | `getInstance` | 公开 | — | 200 `InstanceInfo` | — |
| `GET /api/v0/timezones` | `listTimezones` | 公开 | — | 200 `TimezoneList` | — |

- **动作接口**：按总体设计 3.2"少数业务动作用 `POST /资源/动作名`"，修改密码写成 `POST /me/change-password`，停用写成 `POST /me/deactivate`，都返回 204。
- **注册返回令牌，不返回新建的账户**：总体设计 3.1 说"POST 返回改完之后的完整资源"。注册是"建账户并登录"的动作，调用方接下来需要的是令牌；账户用 `GET /me` 读取。P1 同步到 3.1（3.20）。
- **限流的 429**：所有操作都可能返回（顶层的 `x-problem-codes`）。登录、注册、修改密码另有自己的桶，见 3.10。
- **停用**：任何凭证都能调用，包括 PAT（与 Plane 一样不要求输入密码，风险见 8.6）。不需要请求体。

### 5.2 结构
所有对象都是 `additionalProperties: false`，服务端按契约拒绝未知字段（3.11）；时间是 `date-time`（UTC）；id 是 `uuid`。

| 结构 | 字段 |
|---|---|
| `AuthTokens` | `token_type`（`enum: [Bearer]`）、`access_token`、`access_token_expires_in`（整数，秒）、`refresh_token`、`refresh_token_expires_at`（会话的绝对期限），都必填。给秒数而不是时刻：客户端从收到响应的那一刻起算，不受本机时钟偏差影响（RFC 6749 5.1 的写法，7.1） |
| `User` | `id`、`email`、`first_name`、`last_name`、`display_name`、`user_timezone`、`avatar_url`、`cover_image_url`、`created_at`，都必填；`avatar_url`、`cover_image_url` 是 `type: [string, 'null']`，M5 之前恒为 `null`（3.2）。`is_active` 不返回：停用的账户根本认证不了 |
| `UserUpdate` | `first_name`、`last_name`、`display_name`、`user_timezone`，都可以省略，都不能为 `null`（传 `null` 是 400）。没有 `email`：邮箱只能由管理员用命令行修改（决策点 1、3.17） |
| `Profile` | `theme`（五个值的枚举）、`language`（`en`、`zh-CN`）、`start_of_the_week`（`0`–`6` 的整数枚举）、`onboarding_step`（`OnboardingSteps`，四个布尔值都必填）、`is_onboarded`、`is_tour_completed`、`last_workspace_id`（`uuid` 或 `null`）、`updated_at`。不返回 `id` 和 `user`：资料通过 `/me/profile` 访问，这两个 id 没有用处（前端的替代见 7.5） |
| `ProfileUpdate` | 上面除 `updated_at` 外的字段，都可以省略；`last_workspace_id` 可以传 `null` 清空，其余不能为 `null`。`onboarding_step` 是 `OnboardingStepsUpdate`：四个布尔值都可以省略，服务端把传入的键合并进现有的值（3.14）；未知的键是 400 |
| `ApiToken` | `id`、`label`、`description`、`expired_at`（可为 `null`）、`last_used`（可为 `null`）、`created_at` |
| `ApiTokenCreate` | `label`、`description`、`expired_at`，都可以省略；`expired_at` 是 `type: [string, 'null']` 的 `date-time`，省略或 `null` 表示永不过期 |
| `ApiTokenCreated` | `ApiToken` 的全部字段加上 `token`（令牌原文，只出现在这里）。单独写出所有字段，不用 `allOf`：几个都带 `additionalProperties: false` 的 schema 用 `allOf` 组合时会互相拒绝对方的字段 |
| `ApiTokenPage` | `data: ApiToken[]`、`next_cursor`（`NextCursor`），按 `created_at` 倒序，同一时刻按 `id` 倒序 |
| `InstanceInfo` | M0 的四个字段，加上 `signup_enabled`、`workspace_creation_enabled`、`file_size_limit`（5.3） |
| `Timezone` / `TimezoneList` | `label`、`value`（IANA 名称）、`utc_offset`、`gmt_offset`；列表是 `{data: Timezone[]}`，固定约 100 项，不分页 |

- 主题的五个值与 `web/packages/constants/src/themes.ts:18-69` 的 `THEME_OPTIONS` 相同，没有 `custom`，也没有调色板字段（M1-P2 交接）。
- `RegisterRequest`、`LoginRequest` 的 `email` 在接口描述里写 `format: email`，生成为 `string`（3.12），格式由领域层校验。

### 5.3 实例配置与时区
| 接口字段 | 配置项 | 默认值 | 谁来执行 | 前端的旧名字 |
|---|---|---|---|---|
| `signup_enabled` | `auth.signup_enabled` | prod 为 `false`，dev、test 为 `true`（决策点 2） | M2 的注册用例 | `enable_signup` |
| `workspace_creation_enabled` | `workspace.creation_enabled` | `true` | M3 的创建工作区（交接） | `is_workspace_creation_disabled`（取反） |
| `file_size_limit` | `files.size_limit`（字节） | `5242880` | M5 的上传（交接） | `file_size_limit` |

- **命名**：两个开关都用正面的 `…_enabled`，与 M0 设计 3.2 已写下的 `signup_enabled` 一致。前端读 `is_workspace_creation_disabled` 的 4 个文件改为读取反的值（`create-workspace/page.tsx:40`、新手引导的 `create.tsx:54`、`power-k/config/creation/command.ts:59`、`workspace-menu-root.tsx:45`）。
- **`is_self_managed`**：不定义（3.19）。
- **默认值**：`file_size_limit` 的默认值照搬 Plane 的 `FILE_SIZE_LIMIT`（`plane/apps/api/plane/license/api/views/instance.py:142`）。
- **时区列表**：
  - 照搬 Plane 的 `TimezoneEndpoint`（`plane/apps/api/plane/app/views/timezone/base.py`）：同一份"名称 + IANA"列表，偏移量在请求时用 Go 的时区数据计算。
  - 它服务于 M2 的偏好页，以及 M3 的工作区、项目设置（`web/apps/web/core/hooks/use-timezone.tsx` 的使用方）。
  - 放在 `instance` 模块：它是实例提供的公共参考数据，不属于任何一个账户。

### 5.4 `identity` 的错误码
| 码 | 状态 | 场景 |
|---|---|---|
| `identity.invalid_credentials` | 401 | 邮箱不存在或密码错误；登录的事务里发现密码已被并发修改（3.5） |
| `identity.account_deactivated` | 403 | 密码正确，但账户已停用 |
| `identity.signup_disabled` | 403 | 关闭注册时注册 |
| `identity.email_taken` | 409 | 注册时邮箱已被使用。`nerve users create`、`set-email` 的用例返回同一个错误，命令行把它显示为一句说明 |
| `identity.refresh_token_invalid` | 401 | 刷新令牌未知、已过期、已撤销、被重复使用、被伪造，都是这一个码 |
| `identity.current_password_incorrect` | 422 | 修改密码时当前密码错误，或校验之后密码已被并发修改；`errors[].field = current_password` |
| `identity.api_token_not_found` | 404 | 撤销不存在、已撤销或属于别人的 PAT（看不到的资源一律 404，总体设计 3.5） |

---

## 6. 后端结构

### 6.1 新增和修改的包
| 包 | 内容 | 依赖 |
|---|---|---|
| `internal/shared` | `Actor`、`Error`（含 `FieldError`）、`TxManager`、游标的封套（3.3、3.12） | 只有标准库 |
| `platform/postgres` | `TxManager` 的实现，按结构满足 `shared.TxManager`：事务放进 `context`，仓储用 `postgres.DB(ctx, pool)` 取当前事务或连接池；`COMMIT` 和 `ROLLBACK` 在 `context.WithoutCancel` 下执行，有自己的 `database.commit_timeout`（3.6）；连接池按 UTC 扫描 `timestamptz` | pgx、goose |
| `platform/clock` | `System` 时钟，按结构满足各模块声明的 `Clock` | 标准库 |
| `platform/ratelimit` | 自己实现的按键令牌桶（速率和突发），闲置的键定期清掉；`AllowAll` 一次检查多个键，全有或全无；`Reserve` 扣一个单位并返回退回函数（3.6、3.10） | 标准库 |
| `platform/jobs` | River 客户端的创建、启动、停止；服务用的客户端和命令行用的只投递客户端 | River、pgx |
| `platform/httpserver` | `Router`（在 `HandleFunc`、`Handle` 时记下模式）；`API` 值（`Errors`、`Middlewares(bodies)`）；默认拒绝的认证中间件和它前面的失败闸门；限流中间件；请求元信息（客户端 IP 和限流用的 IP 键；不可信的对端带 `X-Forwarded-For` 时记一次 WARN）；请求期限；请求体上限；3.11 的 `ProblemError` 映射和新平台码；导出 `RequestID`（M0-P2 交接 3）；固定链上加安全响应头（8.3） | 标准库 |
| `platform/httpserver/bodyshape` | 请求体结构表的类型、校验器（含按生成的 Go 类型登记的格式检查器）和中间件（3.11） | 标准库 |
| `platform/webui` | CSP：启动时算出 `index.html` 内联脚本的哈希（8.3） | 标准库 |
| `platform/config` | 6.5 的新配置项和校验；`LogValue` 对 `*_file` 只记是否设置（3.7，M0-P2 交接 4）；环境变量给布尔配置项传空值时报错，不再悄悄当作 `false`（M0-P2 交接 8） | koanf |
| `modules/identity` | 6.2 | `internal/shared`；适配器另外依赖平台、pgx、x/crypto、golang-jwt、River |
| `modules/instance` | 三个配置字段、时区列表 | 同 M0 |
| `bootstrap` | 接线；汇总各模块的公开操作；`run` 让 HTTP 和 River 一起运行，按 3.15 的顺序停机；`users` 命令的最小组合（3.17）；在创建 logger 之后出现的致命错误写一条结构化日志（M0-P2 交接 8）；编译期断言 `postgres` 的事务管理器满足 `shared.TxManager`；启动时的环境提醒（见下） | 全部 |
| `cmd/nerve` | `nerve users create`、`reset-password`、`set-email`、`deactivate`、`activate` 的参数解析 | 同 M0 |
| `tools/bodyshapegen`（工具模块） | 请求体结构表的生成器（3.11），只在 `make gen-go` 中运行，不在 `server/go.mod` 里 | oapi-codegen 的加载器和类型映射 |
| `internal/archtest` | 规则 4 加上 `internal/shared`；规则 6 推广到 `adapter/*/gen`；`TestSQLCSchemaScope`，含 `ALTER TABLE` 的所有者（3.14）；P3 改写传递依赖测试的 google/uuid 例外，加上生成文件不得引用 `openapi_types.UUID` 的规则（3.12） | — |

- **启动时的环境提醒**（控制者复核 m5）：`env` 不是 prod、而监听地址不是回环地址时，记一次 WARN，写明两个后果：注册默认开放（决策点 2），没有配置签名密钥时使用临时密钥（3.7）。部署时忘了设 `NERVE_ENV=prod` 就会这样；dev 的配置只监听 `127.0.0.1`，只有显式改了监听地址才会出现。M8 的镜像设置 `NERVE_ENV=prod`（交接）。
- **健康检查的访问日志**：`/healthz`、`/readyz` 的访问日志降为 DEBUG 级别（M0-P2 交接 8：生产环境中每次探测一条 INFO 太多）。
- **只读的迁移角色**：生产环境用单独的数据库角色执行迁移时，运行服务的角色需要 `goose_db_version` 的 SELECT 权限（M0-P2 交接 7），写进 README 的部署说明。

### 6.2 `identity` 模块的结构
```
modules/identity/
  domain/
    user.go                 User；邮箱规范化和格式；名字、显示名、时区的规则
    profile.go              Profile；Theme、Language、WeekStart；OnboardingSteps 的部分更新（只校验，合并在 SQL）
    password.go             密码规则：组合规则、常见密码名单、主干（3.8）
    common_passwords.txt    生成的名单，go:embed（3.8）
    session.go              Session；刷新令牌 68 字节的布局（编解码，不算标签）；续期的判定表（3.5）
    api_token.go            APIToken；PAT 的格式；名称和过期时间的规则；列表游标的载荷
    errors.go               本模块的错误码（5.4）
  app/
    ports.go                端口（6.3），包括 Clock
    credential_lock.go      账户行锁协议的共用部分：锁账户行、复核调用者的凭证（3.5）
    register.go  login.go  refresh.go  logout.go  authenticate.go
    get_me.go  update_me.go  change_password.go  deactivate.go  get_profile.go  update_profile.go
    create_api_token.go  list_api_tokens.go  revoke_api_token.go
    create_user.go  reset_password.go  set_email.go  activate.go  cleanup_sessions.go
  adapter/
    postgres/               仓储；queries/*.sql；gen/（sqlc 生成）
    http/                   handler，按资源分文件；操作级限流；续期、退出的服务端期限；公开操作的清单；
                            gen/（oapi-codegen 生成的 server.gen.go 和 bodyshapegen 生成的 bodyshape.gen.go）
    argon2/                 PasswordHasher（并发上限和等待上限）
    signing/                持有 Ed25519 密钥：AccessTokens（JWT）和 RefreshTokenMAC（HKDF 派生的 MAC 密钥，3.4）
    river/                  清理会话的 worker
    authn/                  Authenticator 的实现：调用认证用例，把 Actor 放进 context，返回限流键
  module.go                 New(Deps)；Register(router, api)；PublicOperations()；Authenticator()；Jobs()；Admin()
```
- 一个用例一个文件（总体设计 6.1），每个文件预计 40–120 行。注册和 `create_user` 共用"建账户"这一步，放在 `create_user.go`，注册先查 `SignupPolicy`、后签发会话。
- `domain` 的 Go 文件都在 400 行以内。

### 6.3 端口
| 端口 | 声明在 | 实现 | 测试 |
|---|---|---|---|
| `Users`（含锁账户行的 `LockForCredentials`）、`Profiles`、`Sessions`、`APITokens`（仓储，按用例需要拆成小接口） | `identity/app` | `identity/adapter/postgres`（sqlc） | 集成测试，每个测试一个独立的库（`pgtest`）；3.5 的交错测试 |
| `PasswordHasher`（`Hash`、`Verify` 返回"是否需要重新哈希"） | `identity/app` | `identity/adapter/argon2` | 往返、错误密码、参数变化后要求重新哈希、并发上限、等待超时返回 `server_busy`。交错测试用一个带闸门的假实现：`Verify` 停在闸门上，测试在这时提交另一个事务 |
| `AccessTokens`（`Issue`、`Verify`；过期而签名有效时返回可区分的错误） | `identity/app` | `identity/adapter/signing` | 往返；过期（与签名不对区分开）；算法不是 `EdDSA`；换了密钥；缺 `sub`、`sid`；篡改载荷 |
| `RefreshTokenMAC`（`Tag`、`Verify`） | `identity/app` | `identity/adapter/signing` | 往返；改动会话 id、代数、密文中任何一个字节都不成立；随机的标签不成立；换了签名密钥后不成立；比较用 `hmac.Equal` |
| `SignupPolicy`（是否允许注册） | `identity/app` | `bootstrap` 从配置传入的值 | 用例测试。M3 扩展它的实现（持有邀请的人仍可注册，13.2） |
| `Clock` | `identity/app` | `platform/clock` 的 `System`；测试用固定时钟，截到微秒（3.13） | 两种实现都跑同一个小的契约测试（总体设计 6.1 的里氏替换） |
| `RateLimiter`（`AllowAll`） | `identity/adapter/http` | `platform/ratelimit` | 适配器测试：登录、注册、修改密码的键和桶；续期、退出不经过适配器的桶；被后面的桶拒绝时前面的桶不扣 |
| `TxManager` | `internal/shared` | `platform/postgres`（按结构满足） | 提交、回滚、嵌套调用时复用同一个事务 |
| `Authenticator` | `platform/httpserver` | `identity/adapter/authn` | 中间件的单元测试用计数的假实现（含并发、过期的 JWT）；整程序测试用真实现 |
| `Limiter`（`Allow`、`Reserve`） | `platform/httpserver` | `platform/ratelimit` | 同上 |
| `ProblemError` | `platform/httpserver` | `shared.Error`（按结构满足） | `bootstrap` 的测试逐个 `Kind` 核对状态和码 |

没有"随机数"端口：理由见 3.4。

### 6.4 一个带令牌的请求
```
请求 → 请求 ID → 异常恢复 → 访问日志 → 安全响应头          （固定链，httpserver.NewServer）
     → 路由匹配：Router 找到模式，填进 r.Pattern
     → 生成的代码：绑定路径参数和查询参数（格式错误 → 400）
     → 请求元信息（客户端 IP、UA）→ 请求期限 → 请求体上限        （按路由，3.6）
     → 失败闸门：预留本 IP 键的一个单位；已空 → 429，不调用认证器
     → 认证：公开操作直接放行；其他操作要求 JWT（验签，再做会话和账户的主键查询）或 PAT（按哈希查询），
       通过后 Actor 进入 context，单位退回；失败 → 401，单位留下（只是过期的 JWT 退回）
     → 限流：有凭证按凭证计数，没有按 IP 计数
     → 请求体结构：未知字段、不合法的 null、缺少的必填字段 → 400（3.11）
     → 生成的代码：解码 JSON 请求体
     → handler：RequireActor；操作级限流（登录、注册、修改密码）；把生成的类型转成用例的入参
     → 用例：TxManager.WithinTx { 需要时先锁账户行 → 领域规则和校验 → 写数据 }
     → 响应；或者 error → APIErrors → problem+json
```
- 第一稿的图把参数绑定和 JSON 解码画在认证、限流之后，与生成代码的实际顺序不符（评审 M1），已按 spike 的生成代码改正。
- **事务**：以下各自在一个事务里完成；带"锁"的先锁账户行（3.5）：
  - 注册、`nerve users create`：账户、资料（注册另加会话）；
  - 登录（锁）：核对哈希快照和账户状态，需要时写回新哈希，插入会话；
  - 续期：条件轮换；发现重复使用时撤销；
  - 创建 PAT（锁）：复核调用者的凭证，插入；
  - 修改密码（锁）：复核凭证和哈希快照，改哈希，撤销其他会话；
  - 停用（锁）：按全局的加锁顺序（3.5）依次写：`users` 的 `is_active` → `profiles` 的新手引导重置 → `auth_sessions` 的撤销；PAT 不动。M3 加入的成员关系检查（13.2）另按 M3 定下的顺序；
  - 管理员重置密码（锁）：改哈希、撤销会话、撤销 PAT；
  - 管理员修改邮箱（锁）：改邮箱、撤销会话；
  - 恢复（锁）：`is_active`。
- **单条语句的写入**不开事务：资料和偏好的 PATCH、退出、撤销 PAT、`last_used`。它们在执行中被取消时可能已提交却答 500，这无害（3.6）。

### 6.5 配置
`server/configs/config.yaml` 新增（test、prod 的覆盖值写在注释中）：
```yaml
server:
  trusted_proxies: []          # CIDR；为空时客户端 IP 取连接的对端地址（3.10）
  max_body_bytes: 1048576      # JSON 接口的请求体上限
  request_timeout: 15s         # 每个接口请求的期限（3.6）
  addr_file: ""                # 非空时，监听成功后把实际地址写进这个文件（端到端测试用 :0 监听，9.5）
database:
  commit_timeout: 2s           # COMMIT、ROLLBACK 自己的期限，不受请求期限的取消（3.6）
auth:
  signup_enabled: false        # 基础配置关闭；config.dev.yaml、config.test.yaml 覆盖为 true（决策点 2）
  access_token_ttl: 15m
  session_ttl: 720h            # 30 天，从登录起算，续期不延长（3.5）
  refresh_deadline: 4s         # 续期和退出的语句期限；加上 commit_timeout 必须短于前端的 8 秒超时（3.5、7.1）
  session_cleanup_interval: 1h # test: 2s
  jwt:
    private_key_file: ""       # prod 必填；dev/test 为空时启动时生成临时密钥（3.7）
  password:
    argon2_memory_kib: 19456   # test: 64
    argon2_iterations: 2       # test: 1
    argon2_parallelism: 1
    max_concurrent_hashes: 4
    max_wait: 2s               # 拿不到名额时最多等这么久，然后 503 server_busy（3.8）
ratelimit:                     # 每个桶都有速率和突发；test 全部调高（3.10）
  ipv6_prefix_len: 64          # IPv6 客户端按这个长度的前缀计数（3.10）
  anonymous:       {per_minute: 600,  burst: 100}
  auth_failure:    {per_minute: 60,   burst: 60}
  authenticated:   {per_minute: 1200, burst: 200}
  login_ip:        {per_minute: 30,   burst: 10}
  login_ip_email:  {per_minute: 10,   burst: 5}
  register_ip:     {per_minute: 10,   burst: 5}
  password_user:   {per_minute: 5,    burst: 5}
jobs:
  shutdown_timeout: 10s
workspace:
  creation_enabled: true
files:
  size_limit: 5242880
```
- 环境变量照 M0 的规则，例如 `NERVE_RATELIMIT__LOGIN_IP__BURST`。
- **注册的默认值按环境**（决策点 2）：基础配置写 `false`，`config.dev.yaml`、`config.test.yaml` 覆盖为 `true`，prod 没有覆盖。`platform/config` 的加载测试断言：prod 在没有任何覆盖时 `signup_enabled` 为假，dev、test 为真。
- **启动校验**：
  - 时长为正；
  - `session_ttl` 大于 `access_token_ttl`；
  - `refresh_deadline` 加 `database.commit_timeout` 小于 8 秒（前端续期请求的超时，3.5、7.1）；
  - 每个桶的 `per_minute` 为正，`burst` 至少为 1；`ipv6_prefix_len` 在 1–128 之间；
  - CIDR 合法；
  - argon2 的参数在 x/crypto 允许的范围内；
  - `env = prod` 时 `auth.jwt.private_key_file` 必须提供，而且文件能读出一把 Ed25519 私钥。
- **`LogValue`**：列出新的配置项；`private_key_file` 只记 `private_key_file_set`（3.7）。

### 6.6 新的依赖（版本在 P1、P2、P3 写死）
| 依赖 | 版本 | 放在 | 用途 |
|---|---|---|---|
| `github.com/sqlc-dev/sqlc` | v1.31.1 | `server/tools/go.mod` | 代码生成（P1） |
| `github.com/golang-jwt/jwt/v5` | v5.3.1 | `server/go.mod` | 访问令牌（P1） |
| `golang.org/x/crypto` | v0.57.0 | `server/go.mod` | argon2id（P1） |
| `github.com/oapi-codegen/runtime`、`github.com/oapi-codegen/nullable` | v1.7.0、v1.2.0 | `server/go.mod` | 生成代码的依赖（3.12），各自随第一个用到它的生成代码加入；`runtime` 随第一个带参数的操作（P3），它带进 `github.com/google/uuid`（见下） |
| `github.com/riverqueue/river`、`riverdriver/riverpgxv5` | v0.47.0 | `server/go.mod` | 后台任务（P3） |
| River CLI（`github.com/riverqueue/river/cmd/river`） | v0.47.0 | 不进仓库 | 只在写迁移时导出 SQL；命令写进迁移文件的注释 |
| `golang.org/x/term` | v0.46.0（x/crypto v0.57.0 所需的版本） | `server/go.mod` | 命令行不回显地输入密码（P3） |

- 常见密码名单是嵌入的数据文件，不是依赖（3.8）。
- 这些依赖本身都不在架构测试的禁止链接名单里（`github.com/google/uuid`、kin-openapi、testcontainers、docker）。但传递依赖碰到了：`oapi-codegen/runtime` 自己导入 `github.com/google/uuid`（3.12）。所以 P3 按 3.12 改写 `TestNerveBinaryLinksNoBannedModule`：google/uuid 只允许由 `oapi-codegen/runtime` 模块的包导入，另加生成代码不得引用 `openapi_types.UUID` 的规则。其余名单不变，由这个测试在加入依赖的那个 Phase 核对。
- kin-openapi 在 `server/go.mod` 中只被测试（`apitest`）导入；请求体结构的生成器在工具模块，用的是 oapi-codegen 带的那一份。
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
- **存放**（Codex I-5）：
  - 访问令牌和它的过期时刻只在内存里。
  - localStorage 的一个键 `nerve.auth`，值是 `{"refresh_token": …, "login_id": …}`，一次写入，两者总是成对。
  - `login_id` 是 16 字节的随机数（`crypto.getRandomValues`，十六进制），在每次登录、注册时新生成，续期时保持不变。刷新令牌对客户端仍是不透明的，前端不解析它。
  - 租约用另一个键 `nerve.auth.refresh_lease`（见下）。
  - **每次写 `nerve.auth` 都在同一把锁（或租约）下**（控制者复核 R4）：登录、注册写入新记录，续期换令牌，退出删除记录，都先拿续期用的那把锁。否则登录写入新记录的同时，另一个标签页的续期可能把旧会话的令牌写回去，盖掉新记录。
- **取访问令牌**：离过期还有 30 秒以上就直接用，否则先续期。
  - 过期时刻 = 本地收到响应的时刻 + `access_token_expires_in` 秒（5.2）。前端不解析 JWT，也不拿本机时钟和服务端的时刻比较。
  - 理由：本机时钟快了十几分钟时（硬件时钟按本地时间设置的双系统电脑很常见），和服务端的时刻比较会让每个请求都先续期一次，很快碰到限流（评审 I14）。
- **启动**：
  - localStorage 里没有 `nerve.auth` 时，直接判定"未登录"，**不请求 `/me`**。所以登录页没有失败的接口请求（S2）。
  - 有记录时先续期拿到访问令牌，再取 `/me`。续期的结果决定三种状态之一：

    | 第一次续期的结果 | 状态 | 页面 |
    |---|---|---|
    | 200 | 已登录 | 照常 |
    | 401 | 未登录：清掉记录 | `AuthenticationWrapper` 跳到登录页 |
    | 429、5xx、网络错误 | **会话暂不可用**：保留记录 | 显示"暂时连不上服务器"和"重试"按钮，同时自动退避重试（有 `Retry-After` 就按它，没有就 1、2、4……秒，最多 30 秒）；成功后进入"已登录"。不跳到登录页（控制者复核 N3） |

- **续期**：
  - 同一个标签页内，同一时刻只有一次续期，其他调用共用同一个 Promise。
  - 跨标签页串行（见下一条）。拿到锁之后重新读 `nerve.auth`：另一个标签页可能刚换过令牌。
  - 续期请求用一个不挂认证中间件的客户端，避免递归。
  - **超时**：续期请求自己的超时是 8 秒，`navigator.locks` 和租约两条路径都一样。顺序是服务端的语句期限 4 秒加提交期限 2 秒（3.5、3.6）< 客户端超时 8 秒 < 租约 10 秒：客户端放弃时，服务端已经结束这次续期（提交、回滚，或者提交没有回应，3.5 的第 4 种情况）。
  - **写回之前核对 `login_id`**：续期在锁内读出记录，拿到响应后、写回之前再读一次；`login_id` 已经变了（租约非原子时另一个标签页登录了），就丢弃这次续期的结果，不写回，按"`login_id` 变了"处理（见"其他标签页"）。
  - 续期的结果：

    | 结果 | 处理 |
    |---|---|
    | 200 | 保存新的一对令牌，`login_id` 不变 |
    | 401 | 清掉记录，结束会话。只有这一种结果表示"需要重新登录" |
    | 429、5xx、网络错误 | 保留记录，退避后再试（同上）。这一次调用把错误交给调用方，页面按已有的错误提示显示；启动时则进入"会话暂不可用" |
- **跨标签页的协调**（评审 I4）：
  - `navigator.locks` 和 `crypto.randomUUID` 都只在安全上下文（HTTPS 或 localhost）中存在。总体设计 2.1 允许直接用 HTTP 部署（"需要 HTTPS 时加 Caddy"），例如在局域网里打开 `http://192.168.1.20:8080`，那里没有它们。
  - 第一稿只用 `navigator.locks`：在这种部署下，令牌管理器要么报错，要么几个标签页同时续期，触发重复使用检测（3.5），用户被踢下线。端到端测试跑在 `127.0.0.1` 上，那是安全上下文，发现不了。
  - 有 `navigator.locks.request` 时：`navigator.locks.request("nerve.auth.refresh", …)`。持有锁的标签页被冻结时，其他标签页等它恢复或被丢弃（锁随之释放），不会拿旧令牌续期。
  - 没有时，用 localStorage 租约：
    - 键 `nerve.auth.refresh_lease`，值 `{"owner": <标签页 id>, "expires": <毫秒时间戳>}`，租期 10 秒。
    - 标签页 id 用 `crypto.getRandomValues` 生成，它在非安全上下文中也能用。
    - 获取：没有租约、租约已过期、或者租约是自己的，就写入自己的租约；等 100 毫秒再读一次，仍是自己的才算拿到。
    - 等待：监听租约键的 `storage` 事件，同时每 200 毫秒轮询一次；租约被删除或过期后重新获取。
    - 用完：只删除自己的租约。
    - 局限：localStorage 没有原子的"比较并写入"。"写入后再读一次"只是让两个标签页同时拿到租约变得极不可能。万一发生，代价是一次重复使用检测，用户重新登录，与续期响应丢失的代价相同（3.5）。
    - 剩下的情况：持有租约的标签页发出续期后被浏览器冻结或节流，超过了 10 秒的租期。另一个标签页拿到租约，读到的还是旧令牌；服务端如果已经提交了轮换，这就是一次重复使用检测。它和响应丢失一样，只能观察（§16）。
    - 选租约而不选 BroadcastChannel 选主：租约只用刷新令牌所在的同一个 localStorage，不需要选主、心跳和换主。
  - README 写明：HTTP 部署可以用，多标签页靠租约协调；放在公网上应当用 HTTPS。
- **401**（openapi-fetch 中间件的 `onResponse`，Codex M-5）：
  - 请求带了访问令牌却得到 401：续期一次，用新令牌重发一次。重发需要请求的副本，在 `onRequest` 中保存（P4 先用原型验证 openapi-fetch 0.17.0 的中间件能这样做）。
  - **只有两种情况结束会话**：续期得到 401；续期成功、重发后仍是 401。续期得到 429、5xx 或网络错误时保留记录，把这次的错误交给调用方。
  - 没有刷新令牌时，401 原样交给调用方。
- **结束会话**：
  - 清掉内存和 `nerve.auth`，调用应用注册的回调：`rootStore.resetOnSignOut()`。
  - 当前账户变为空，`AuthenticationWrapper` 渲染 `<Navigate to="/?next_path=…" replace />`（3.18）。
  - 跳转只在包装层这一处发生；现在的 `window.location.replace` 删除。
- **退出**（评审 M7）：
  - 在续期用的同一把锁（或租约）下，读出当前的刷新令牌，用不挂认证中间件的客户端调用 `POST /auth/logout`。
  - 不能用挂了认证中间件的客户端：访问令牌恰好过期时，中间件会先续期、再重发，重发的请求体里还是续期前的刷新令牌，服务端看到的是上一代，退出什么也没做（3.5）。
  - 尽力而为：失败也清本地。然后结束会话，回到登录页。
- **其他标签页**（`nerve.auth` 的 `storage` 事件，Codex I-5）：

  | 事件 | 含义 | 处理 |
  |---|---|---|
  | 记录被删除 | 别的标签页退出了 | 结束会话（A6） |
  | 本标签页未登录时记录出现 | 别的标签页登录或注册了 | 续期拿到访问令牌，取 `/me`，由 `AuthenticationWrapper` 按规则跳转（A6 的第二种切换） |
  | `login_id` 变了 | 别的标签页登录了另一个会话（可能是另一个账户） | 丢掉内存中的访问令牌，重置 stores，重新取 `/me`，按新账户显示（A6 的切换账户） |
  | `login_id` 没变 | 只是续期换了令牌 | 什么都不做；下一次续期在锁内重新读记录 |

  - 第二稿只处理"被删除"：两个标签页先后登录不同的账户时，旧标签页仍显示第一个账户，下一次续期却用了第二个账户的刷新令牌，写入落到第二个账户。
- **可测试**：`storage`、`locks`、`fetch`、当前时刻、随机数都从构造函数传入，vitest 用假实现（9.4）。
- **不做**：定时主动续期（按需续期已经足够）；把访问令牌放进 sessionStorage。

### 7.2 传输层的清理
- **web 的 axios 基类**（`web/apps/web/core/services/api.service.ts`）：
  - 删掉 `withCredentials: true` 和 401 拦截（M1-P4 交接：恒为真的 `currentPath` 判断不照搬）。
  - 此后它只供 M3–M8 还没对接的领域使用。那些调用指向 Plane 的旧接口，在 Nerve 中得到 404 problem，由各领域对接时逐个删除。M2 能到达的页面挂载时不调用它们（3.1）。
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
  - 成功：令牌交给令牌管理器，它在续期的那把锁下写入新记录（新的 `login_id`，7.1），然后取当前账户，由 `AuthenticationWrapper` 按规则跳转。
- **错误就地显示**（M1-P3 交接）：不跳转，邮箱留在输入框里。
  - `problem.code` 映射到 `t()` 的文案；`validation_failed` 和 `bad_request` 的字段错误显示在对应字段下方（`weak_password` 显示规则，`common_password` 显示"密码太常见"，`required` 显示"必填"）；429 显示"尝试次数过多，请稍后再试"；503 `server_busy` 显示"服务器繁忙，请稍后再试"；403 `signup_disabled` 显示"注册已关闭"。
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
  - 判断只用 M2 的数据：令牌管理器的状态，当前账户，资料中的 `is_onboarded`、`onboarding_step`，`next_path`。
  - 未登录访问受保护的页面：跳到 `/?next_path=${encodeURIComponent(pathname + search + hash)}`。
  - 已登录：`next_path` 有效就去 `next_path`；未完成引导去 `/onboarding`；否则去 `/create-workspace`。M2 不取工作区落点的数据，M3 加回（3.1）。
  - "会话暂不可用"（7.1）：渲染等待和重试的界面，不跳转。
- **`InstanceWrapper`**：改调 `GET /api/v0/instance`（M0-P5 交接）。请求失败时仍显示维护页。

### 7.5 类型、services 与 stores
| 位置 | 现状 | M2 |
|---|---|---|
| `@nerve/types` 的 `IUser`、`TUserProfile`、`IUserTheme`、`TOnboardingSteps`、`IInstanceInfo`、`IInstanceConfig`、`IApiToken`、`ICsrfTokenData`、`TTimezones`、`TTimezoneObject` | 手写的 Plane 类型（`web/packages/types/src/users.ts` 等） | 删除。使用方改从 `@nerve/api-client` 导入 `User`、`Profile`、`OnboardingSteps`、`InstanceInfo`、`ApiToken`、`Timezone`（3.12） |
| `IUserLite`、`IUserSettings` | 成员信息（M3）；`/users/me/settings/` 中的工作区数据 | 不动，留给 M3。`IUserLite.avatar_url` 按 3.2 规则 2 由 M3 改为可为 `null` |
| `EStartOfTheWeek` | 日历等 9 个文件使用的数字枚举 | 保留：纯界面常量，取值与生成的 0–6 一致 |
| `core/services/auth.service.ts` | CSRF 和表单 | 重写：`register`、`login`。`refresh`、`logout` 由令牌管理器调用 |
| `core/services/user.service.ts` | Plane 的 `/api/users/me/…` | M2 的方法改为生成客户端的薄封装，加上 `deactivate`。属于其他领域的 4 个方法（`getUserProfileIssues`、`leaveWorkspace`、`joinProject`、`leaveProject`）原样留下，由 M3、M4 对接时迁走 |
| `core/services/instance.service.ts`、`timezone.service.ts` | Plane 的地址 | 生成的客户端 |
| `@nerve/services` 的 `api.service.ts`、`developer/` | `APITokenService`（Plane 的 `/api/users/api-tokens/`） | 删除（P5）。包里只剩地址工具和文件工具，整个包由 M5 删除（总体设计 7.1） |
| `store/user/index.ts`（`UserStore`） | `IUser`；登录时先取账户，再并行取资料、设置、工作区三样 | `data: User`；没有 `nerve.auth` 时不取数（7.1）；`fetchCurrentUser` 只取 `/me` 和 `/me/profile`；`signIn`、`signUp`、`signOut`、`deactivate` 通过令牌管理器；删除死成员（`reset`、`isAuthenticated`、`error` 等） |
| `store/user/profile.store.ts` | `TUserProfile`；`updateUserProfile` 吞掉错误（`:138-148`），调用方的错误提示从不出现 | `data: Profile`；失败时抛出。`finishUserOnboarding` 合并为一次 `PATCH /me/profile`（部分的 `onboarding_step` + `is_onboarded` + `last_workspace_id`）；`updateTourCompleted`、`updateUserTheme` 同样用这一个接口 |
| `store/user/settings.store.ts` | `/users/me/settings/` 加上侧边栏状态 | 从登录流程中拿掉（3.1）；删除死成员（`isScrolled`、`toggleIsScrolled` 等）；工作区数据留给 M3 |
| `store/instance.store.ts` | `IInstanceConfig` | `config: InstanceInfo` |
| `store/workspace/api-token.store.ts` | 没有任何代码读它（`store/workspace/index.ts:93` 只是创建）；页面用 SWR 直接调用 service | 移到 `store/user/api-token.store.ts`，用生成的客户端重写，翻页直到取完；页面和弹窗改为经 store 读写（总体设计 7.2：组件不直接调用接口） |
| `app/(all)/onboarding/page.tsx` | 挂载时预取工作区列表和邀请（`:32-45`） | 删除两次预取（P4，3.1）；`user` 取到后渲染 `OnboardingRoot` |

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
- 新手引导的"角色""用途"两步（338 行）和 `is_self_managed`（3.19）。
- 新手引导页挂载时对工作区和邀请的预取（3.1）。
- `@nerve/services` 的 `api.service.ts`、`developer/`（共 178 行）。
- 上传控件（3.2）：general 页的头像上传弹窗（`UserImageUploadModal`）和封面选择器（`ImagePickerPopover`），新手引导资料步骤的头像上传。显示头像和封面的代码保留。
- general 页写 `role` 默认值的逻辑（3.19）。
- 死成员和死 prop（7.8）。
- 以上各项的文案键：中英文一起删，`sync-check` 和"整键 + 模板前缀"的核对沿用 M1 的做法。

### 7.7 个人设置的四个标签页
- **general**：
  - 名、姓、显示名可以修改；邮箱只读，只能由管理员用命令行修改（决策点 1）；
  - 头像和封面照常显示（M5 之前是首字母头像和默认封面），上传控件按 3.2 删除；
  - "停用账户"（决策点 3，已裁定为 A）：保留现有的按钮和确认弹窗（`web/apps/web/core/components/settings/profile/content/pages/general/form.tsx:398-408`、`web/apps/web/core/components/account/deactivate-account-modal.tsx`），改调 `POST /api/v0/me/deactivate`；成功后结束会话，回到登录页。弹窗文案按 Nerve 的行为改写：不发邮件，由服务器管理员恢复。
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
  - 列表、撤销。列表显示创建时间和最后使用时间，便于认出不认识的令牌（8.5）。
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
- **界面文案都走 `t()`**，中英文的键一致（`sync-check` 是门禁）。新写的错误提示、令牌管理器的提示（包括"会话暂不可用"）都在其中。

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
| 删除 | 约 1,050 行 | 7.6 的各项。头像和封面的显示代码不再删除 |
| 重写 | 约 3,000 行 | stores 约 700 行；services 约 250 行；两个包装层约 180 行；认证表单约 450 行；个人设置和 PAT 组件约 900 行；新手引导约 450 行 |
| 新增 | 约 1,000 行 | 令牌管理器约 300 行（含 `login_id` 和"会话暂不可用"），跨标签页的协调约 100 行，它们的测试约 450 行，其余约 150 行 |
| 净变化 | 减少约 50 行 | |

**后端**：
- 生产代码约 5,450 行（不含生成的代码和名单文件；其中请求体结构的校验器、格式检查器和生成器约 400 行，账户行锁约 150 行，自己的令牌桶约 100 行，刷新令牌的 MAC 约 50 行），测试约 6,700 行（含六个交错测试、失败闸门的并发测试和第四个整程序测试）；
- 手写的 SQL 约 350 行，加上 River 的迁移（318 + 189 行）；
- `identity.yaml` 约 900 行；
- 常见密码名单 33,887 行（生成的数据，277 KB）。

**端到端**：fixture 约 500 行，17 个故事约 2,150 行。

---

## 8. 安全

### 8.1 密码与哈希
见 3.8：组合规则加常见密码名单（主干从两端去掉所有非字母）；argon2id 的并发上限和等待上限；argon2 始终在账户行锁之外（3.5）。

### 8.2 账户枚举
- 登录不暴露账户是否存在，时间也一致（3.9）。
- 注册关闭时，已注册和未注册的邮箱得到同一个 403（3.9）。
- 注册开放时必然暴露：没有邮件通道，就无法做到"已注册和未注册的回答相同"。用每个 IP 每分钟 10 次的注册限流压低探测速度。
- 这是 v0 已知的局限，写进 README 的安全说明（8.7）。

### 8.3 安全响应头与 CSP
- **两层，分开放**（评审 M4）：
  - **固定链上的安全响应头**（所有响应，P2）：`X-Content-Type-Options: nosniff`、`Referrer-Policy: same-origin`、`X-Frame-Options: DENY`。它们对接口的响应同样有意义，所以放在固定链上。固定链由三个中间件变为四个，M0 设计 3.3、总体设计 6.4 的说明在 P2 随之更新。
  - **CSP 只加在 HTML 响应上，由 `webui` 负责**（P4）：它只对页面有意义，而且要用 `index.html` 里内联脚本的哈希，只有 `webui` 知道这些脚本。
  - M0-P5 交接说"和 CSP 放在同一层"，本意是两者都由服务端在 M2 加入。按上面的理由分在两层，是有意的安排。这一项在 P4（CSP 落地时）正式关闭，review 写下这条理由。
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
  - `Authorization` 请求头；
  - 认证接口和创建 PAT 的请求体、响应体；
  - JWT 私钥的内容和密钥文件的路径（3.7）；
  - 不存在的账户的邮箱。
- **会话 id 直接记为 `session_id`**（第三稿重新推导，Codex I-2 和控制者的裁定）：
  - 第二稿只记它的摘要 `session_ref`，理由是知道会话 id 就能拼出"更旧的代数"让会话被撤销。现在撤销要交出这个会话真的发过的旧令牌（3.5），也没有哪个限流键取自会话 id（3.10）。知道会话 id 不再带来任何能力。
  - 它和 `user_id`、`token_id` 一样是标识，不是机密。刷新令牌中的会话 id 和代数本来就不是机密，令牌的安全只靠其中 32 字节的随机密文。
  - 所以直接记，排查时按它在数据库中找到会话。`session_ref` 这个概念删除。
- **INFO**：
  - 注册、`nerve users create`：`user_id`；
  - 登录成功：`user_id`、`session_id`、IP；
  - 登录失败：原因（`invalid_credentials` 或 `deactivated`）、IP，已知账户时加 `user_id`；
  - 退出：`session_id`；
  - 修改密码：`user_id`；重置密码：`user_id`、撤销的会话数和 PAT 数；修改邮箱：`user_id`、撤销的会话数（不记新旧邮箱）；
  - 停用、恢复：`user_id`，撤销的会话数或重新可用的 PAT 数，自助还是命令行；
  - 创建、撤销 PAT：`user_id`、`token_id`；
  - 触发限流：桶的名字、IP；
  - 密码哈希名额等待超时：当时排队的数量。
- **WARN**：
  - 发现刷新令牌被重复使用：`user_id`、`session_id`、IP；
  - 不可信的对端带着 `X-Forwarded-For`（每个进程一次，3.10）；
  - 非 prod 环境监听在非回环地址上（启动时一次，6.1）。
- **访问日志**不变：方法、路径、状态码、耗时、请求 ID。令牌从不出现在地址里，路径中没有机密。
- **测试**：登录、注册、续期、创建 PAT 的测试捕获日志输出，断言其中不含密码、令牌和令牌的哈希。

### 8.5 刷新令牌放在 localStorage 的风险与升级路径
- **风险**（Codex I-4）：任何 XSS 都能读出 localStorage 中的刷新令牌。拿到它的人能做的，不止是用这个会话：
  - 刷新令牌换来访问令牌，访问令牌可以创建一个**永不过期的 PAT**（创建 PAT 不要求输入密码，3.5）。
  - 这个 PAT 不受退出、重复使用检测、会话 30 天期限和用户自己修改密码的影响，而且还能再创建新的 PAT。
  - 所以泄露的影响**不以 30 天为限**：直到恶意的 PAT 被撤销，或管理员重置密码为止。
- **现有的防线**：
  - 严格的 CSP（8.3）；
  - 服务端清洗 HTML（M4，bluemonday）；
  - 刷新令牌每次使用后换新，并检测重复使用（3.5）；
  - 会话有绝对期限，续期不延长（3.5）；
  - 访问令牌只在内存里，15 分钟过期；
  - 退出、修改密码在下一个请求就生效（3.5）。
- **怀疑泄露时的恢复步骤**（README 的安全说明，8.7）：
  1. 在个人设置的 api-tokens 页或安全页查看 PAT 列表，按创建时间和最后使用时间认出不认识的令牌，逐个撤销；
  2. 或者请服务器管理员执行 `nerve users reset-password`：撤销全部会话和全部 PAT（3.5）。
- **负责人以后可以选的产品选项**（记录在此，v0 **不采用**）：
  - 创建 PAT 时重新输入密码。这改变已定下的"创建 PAT 不要求密码"（与 Plane 相同），也影响自动化的调用方；
  - 不允许用 PAT 创建 PAT，或给 PAT 的有效期设上限。两者要一起做：只限有效期、仍允许无限派生，解决不了问题。
- **升级路径**（总体设计 4.3，M2 只写下来，不做）：
  - 注册、登录、续期的响应把刷新令牌写进 `HttpOnly; Secure; SameSite=Strict; Path=/api/v0/auth` 的 Cookie；
  - 续期和退出在请求体里没有令牌时，从这个 Cookie 读；
  - 其余接口仍然只认 `Bearer`，不受影响。
  - 前端的令牌管理器不再碰 localStorage 中的令牌（`nerve.auth` 只剩 `login_id`），续期请求带上 `credentials: "same-origin"`；多标签页共享同一个 Cookie，跨标签页的协调（7.1）仍然需要。
  - 跨站请求伪造：`SameSite=Strict`，加上令牌只在响应体中返回、跨站读不到，最坏的情况是对方让令牌换了一次新。
  - 改动只在 `identity` 的 HTTP 适配器和令牌管理器两处。它只保护刷新令牌；页面上的 XSS 仍能用内存中的访问令牌创建 PAT，上面的风险不因此消失。

### 8.6 其他
- JSON 接口的请求体上限 1 MiB（3.11）；每个接口请求有 15 秒的期限（3.6），续期和退出 4 秒（3.5）。
- 客户端 IP 只在对端是可信代理时才取自 `X-Forwarded-For`（3.10）。认证之前的失败闸门按 IP 计数：同一个出口 IP 后的攻击流量会让这个 IP 后面有效的调用方也得到 429，直到额度恢复（3.6）。
- PAT 拥有账户的全部能力，v0 不做权限范围（与 Plane 相同）。它可以派生新的 PAT，风险和恢复见 8.5。管理员重置密码会撤销全部 PAT，恢复账户时能赶走持有 PAT 的攻击者（3.5）。
- **停用**（决策点 3）：任何凭证都能停用自己的账户，**包括 PAT**，不只是访问令牌；与 Plane 一样不要求输入密码。拿到一个 PAT 的人可以让账户停用，服务器管理员用 `nerve users activate` 恢复。恢复后账户的 PAT 重新可用；怀疑账户被盗时，恢复之后再执行 `reset-password`（3.17）。
- **密钥扫描**（Codex M-7）：`nrv_pat_`、`nrv_rt_` 前缀让代码托管平台的密钥扫描**可以配置**识别泄露的令牌的自定义规则；前缀本身不会让平台自动识别。精确的正则：
  - PAT：`nrv_pat_[A-Za-z0-9_-]{43}`（32 字节，base64url 不补 `=`）；
  - 刷新令牌：`nrv_rt_[A-Za-z0-9_-]{91}`（68 字节，含 16 字节的标签；spike 生成的令牌与之匹配）。
  - README 写明在哪里启用：例如 GitHub 仓库或组织设置中的 Secret scanning → Custom patterns（需要平台提供这项功能）。
- 直接用 HTTP 部署时，浏览器不在安全上下文中：跨标签页的续期用租约（7.1），复制令牌用剪贴板工具已有的 `execCommand` 退路（`web/packages/utils/src/string.ts:201-206`）。README 的部署说明建议公网部署用 HTTPS。

### 8.7 README 的运维与安全说明
随实现它的 Phase 写进 README：

| 内容 | Phase |
|---|---|
| 部署时设 `NERVE_ENV=prod`；不设时注册默认开放、签名密钥是临时的，启动日志会提醒（6.1） | P1 |
| 生成签名密钥：`openssl genpkey -algorithm ed25519 -out nerve-jwt.pem`；换密钥的后果：换钥之前的旧代刷新令牌不再能被认出重复使用，刷新令牌正被盗用时受害者只是被登出（恢复靠修改密码或 `reset-password`），同一出口 IP 后的大量标签页会短暂得到 429；所以在低峰时换（3.7、§16） | P1 |
| 常见密码名单的第三方声明（3.8） | P1 |
| 迁移角色的权限：用单独的角色执行迁移时，运行服务的角色要能读 `goose_db_version`（6.1）。第一批迁移在 P1，所以放在 P1 | P1 |
| 前面有反向代理时配置 `server.trusted_proxies`（3.10） | P2 |
| 第一个账户：`nerve users create --email …`（决策点 2） | P3 |
| 管理命令：`reset-password` 撤销全部会话和 PAT；`set-email` 和 `activate` 不撤销 PAT，怀疑账户被盗时另外执行 `reset-password`（3.17） | P3 |
| 注册会暴露邮箱是否已注册（8.2）；刷新令牌泄露时可能派生 PAT，以及恢复步骤（8.5） | P3 |
| 密钥扫描的自定义规则和启用位置（8.6） | P3 |
| HTTP 部署时多标签页靠租约；公网部署用 HTTPS（7.1、8.6） | P4 |

---

## 9. 测试

### 9.1 后端单元测试
| 对象 | 内容 |
|---|---|
| `identity/domain` | 邮箱规范化和格式；名字中的网址；显示名；时区；主题、语言、每周第一天；新手引导的部分对象（未知的键、值不是布尔值时拒绝）；密码规则（表格驱动：8、128 的边界，缺各类字符，非 ASCII 字符；主干的提取；名单命中和不命中的例子，包括 3.8 列出的那些和七个绕过第二稿的写法；主干等于邮箱前缀的主干；名单中每个条目都满足生成时的过滤条件）；刷新令牌 68 字节布局的编码和解码（往返；长度不对、前缀不对、base64 非法；字段的偏移）；续期的判定表（3.5 的每一行：当前代且哈希相符；当前代但哈希不符；旧代且标签成立；旧代但标签不成立；g 大于当前代；查不到、已撤销、已过期；标签的结果由调用方传入）；PAT 的格式、名称、过期时间；PAT 列表游标的载荷 |
| `identity/app` | 每个用例用内存里的假端口测试：注册（关闭注册时先答 403、不查邮箱；邮箱已存在；三行在一个事务里）；登录（邮箱不存在时仍做一次假校验；事务里哈希已变：新哈希下密码仍成立（并发的重新哈希）时重做一次锁内那一步后成功，不成立时 401，重做时又变了也是 401；停用只在密码正确后报出；参数变化后重新哈希）；续期（轮换，`token_hash` 换成新密文的哈希；真实的旧令牌（标签成立）撤销；伪造的旧代（随机密文加随机标签）、过期访问令牌里的 `sid` 加 `g = 0` 都 401 且不撤销；当前代而哈希不符时不撤销；换了签名密钥之后，换钥之前的旧代令牌 401 且不撤销，当前一代照常续期；`expires_at` 不变；用固定时钟推到期限之后续期失败；条件更新没有命中时重读）；退出（只有当前一代有效的令牌才撤销，其余都是 204 且不变）；修改密码（撤销其他会话；用 PAT 修改时撤销全部；PAT 不变；凭证已被撤销时 401；哈希只因并发的重新哈希而变时重新校验一次后成功）；停用（撤销全部会话、重置新手引导、密码和 PAT 不变）；恢复；创建账户（不经过 `SignupPolicy`，不建会话）；重置密码（撤销全部会话和全部 PAT，返回数量）；修改邮箱（两个邮箱都规范化；新邮箱已被使用时 `identity.email_taken`；新旧相同时报错；撤销全部会话，PAT 不变）；PAT 的创建（凭证已被撤销时 401）、列出（`limit` 为 0、101 时 422）、撤销；认证（JWT 加会话检查；账户停用时 401；PAT 的过期和 `last_used` 的写入频率）；清理过期会话 |
| 适配器 | 6.3 表中各端口的测试，包括 `signing` 的 MAC 标签（往返、改一个字节不成立、随机标签不成立、换密钥后不成立）；`authn` 把 `Actor` 放进 `context` 并返回限流键，令牌无效时返回 401 的错误，签名有效而只是过期的 JWT 返回的 401 错误实现 `ExpiredCredential()` 且为真，其他 401（签名不对、会话已撤销或已过期、PAT 不存在、账户停用）不实现或为假，数据库出错时返回别的错误；HTTP 适配器：登录、注册、修改密码的限流键和桶，续期和退出不经过适配器的桶（只受 `anonymous` 约束，3.10），续期和退出的 `context` 带 4 秒的期限 |
| `platform` | 认证中间件（公开操作不看令牌；非公开操作没有令牌、令牌无效、令牌有效；闸门已空时 429 且计数的假认证器没有被调用；先预留：认证失败时单位留下，成功、签名有效而只是过期的 JWT（`ExpiredCredential()`）、认证器返回非 401 的错误（500）时退回；并发：额度 3、50 个并发的无效令牌，假认证器只被调用 3 次，其余 429）；`Router` 在 `HandleFunc` 和 `Handle` 时都记下模式；`bodyshape` 的校验器（spike 的 25 个请求：未知字段、`null`、必填、数组、可为空的对象、开放的 map、类型、一次收集多个问题；格式：`date-time`、`uuid` 各取 3.11 spike 的写法（含 JSON 转义和宽松的 uuid 写法），检查器的结论与把同一个值解码进生成的结构体一致，不合法时 `invalid_format`）和中间件（把请求体原样放回）；`bodyshapegen`（两次生成的输出相同；格式来自 `type-mapping`；遇到没有检查器的 Go 类型、`format: date`、`x-go-type` 或不支持的组合时失败；这些测试在工具模块，由 `make test` 运行）；限流（自己的令牌桶：突发、`AllowAll` 被拒时一个都不扣、`Reserve` 的退回只生效一次且不超过突发、清理闲置的键、`Retry-After`）；请求期限；可信代理下的客户端 IP，不可信的对端带 `X-Forwarded-For` 时只记一次 WARN；客户端 IP 键（IPv4；IPv4 映射的地址；同一个 /64 的两个 IPv6 地址同键、不同 /64 不同键；前缀长度可配）；413；`ProblemError` 的映射（字段错误、`RetryAfter`、不满足接口时 500），包括不带 Go 类型名的解码错误和 `context.Canceled`；`webui` 的 CSP 哈希；配置的新校验、`LogValue` 不含密钥文件的路径、prod 在没有覆盖时 `signup_enabled` 为假 |
| `bootstrap` | 非 prod 且监听在非回环地址时记一次 WARN；`API.Middlewares` 的顺序 |

### 9.2 集成测试（连接真实的 Postgres，`pgtest`）
- 仓储的每条查询，包括：
  - 续期的查找和条件轮换（时刻作为参数传入；轮换后 `generation` 加一、`token_hash` 换成新值；代数或哈希不符时不命中）；
  - 游标分页（`created_at` 相同的两行不重不漏）；
  - 违反约束：只有 `users_email_key` 的唯一冲突映射为领域错误 `identity.email_taken`；其余约束违反，包括每一个 CHECK，都是缺陷（领域已先校验过），按 3.11 成为 500 `internal_error`。identity 的 postgres 适配器的 `TestCreateUserBreakingACheckIsInternal` 核对 CHECK 触发时仓储返回原样的数据库错误，不是领域错误，所以答 500；
  - CHECK 的反例：邮箱的大写和空白、`onboarding_step` 的八种反例（4.3）、`start_of_the_week = 7`、`revoked_at` 与 `revoke_reason` 不一致；
  - 审计列：插入和业务更新之后，`created_at`、`updated_at` 等于固定时钟的时刻（3.13）；
  - 重置密码在一个事务里撤销会话和 PAT；
  - 两个请求同时合并 `onboarding_step` 的不同键，两个键都保留（3.14）；
  - 清理任务跳过被锁住的会话，下一轮再删。
- **账户行锁的交错测试**（3.5，P3 的完成线）：用带闸门的假哈希器确定性地交错：
  1. 登录校验完成 → 重置提交 → 登录的插入失败，没有新会话；
  2. 登录的重新哈希不会覆盖重置写入的哈希；
  3. 用被并发重置撤销的凭证创建 PAT，失败，没有新 PAT；
  4. 修改密码与登录交错：登录用的是旧密码时失败；
  5. 两个登录交错，其中一个做了重新哈希：另一个重新校验一次后成功；
  6. 一个事务以 `FOR NO KEY UPDATE` 锁着账户行时，别的事务插入引用它的会话不必等待（有语句超时，超时即失败）。
- `TxManager`：提交与回滚；语句都做完之后取消请求的 `context`，事务仍然提交，数据在库里；语句失败之后取消，回滚仍然完成，连接回到池里可以再用（3.6 的提交规则）。
- 5 个迁移都能 up、down、再 up。
- River worker 的 `Work()` 只删除过期的会话。
- `archtest` 的 `TestSQLCSchemaScope`（3.14），包括用人造的迁移文件核对"`ALTER TABLE` 放在不属于表所有者的文件里时失败"。
- `archtest` 的 uuid 守卫（3.12，P3）：
  - 传递依赖测试：用合成的导入图核对，`github.com/google/uuid` 只由 `oapi-codegen/runtime`、`runtime/types` 导入时通过；再有一个别的包（例如某个模块的适配器）导入它时失败，并打印那条导入链；真实的 `./cmd/nerve` 通过；
  - 生成文件的规则：一个人造的、引用 `openapi_types.UUID` 的生成文件让规则失败；仓库里现有的生成文件都通过。

### 9.3 契约测试
- `identity`、`instance` 的 handler 测试对每种状态（包括每种 problem）调用 `CheckResponse`。
- `bootstrap` 的四个整程序测试：3.6 的三个（公开集合等于 `security: []`；注册的 `/api/v0` 路由等于接口描述的全部操作；每个非公开操作不带令牌得到 401 problem），和 3.11 的一个（每个带请求体的操作，结构不合契约时 400，可为空的字段传 `null` 不是 400；从 P3 起，每个带格式的字段各传一个不合法的字符串得到 400 `invalid_format`，另有一个同时有未知字段、缺少的必填字段和写错的格式的请求，一次返回全部问题）。另外，带 PAT 调用 `GET /me` 成功。
- `apitest`：
  - 写法检查加上 `security` 的两条规则（3.12）和 `x-problem-codes` 的写法（3.11）；
  - `CheckResponse` 核对 problem 的码在这个操作声明的集合里；每个模块的 handler 测试跑完时，核对声明的码都被返回过（3.11）；
  - 新增 `CheckRequest`，测试中发出的请求也按契约校验（3.11）。
- 错误出口的测试（3.11）：P1 测请求体解码和 handler 两个出口；P3 测参数绑定的出口（`limit=abc`、非法的 `token_id`），以及 `limit` 为 0、101 时的 422。

### 9.4 前端单元测试（vitest）
- 令牌管理器：
  - 同一标签页内只续期一次；
  - 用假的 `locks` 验证跨标签页串行；没有 `locks` 时用假的 storage 和假时钟验证租约的获取、等待、过期后重新获取、只删除自己的租约；
  - 续期请求在两条路径上都是 8 秒超时；
  - 过期时刻由 `access_token_expires_in` 和本地收到响应的时刻算出，本机时钟偏差不影响；
  - `nerve.auth` 一次写入刷新令牌和 `login_id`；登录、注册时生成新的 `login_id`，续期时不变；
  - 登录、注册、续期、退出写 `nerve.auth` 时都持有同一把锁（或租约）：假的 `locks` 记录每次写入时锁是否被持有；
  - 续期的响应回来之前 `login_id` 被换掉（假的 storage 在续期途中写入另一个账户的记录）：丢弃这次续期的结果，不写回，记录仍是新账户的，按"`login_id` 变了"处理；
  - `storage` 事件的四种情况：记录被删除时结束会话；本标签页未登录时记录出现，续期、取 `/me`；`login_id` 变了时丢掉访问令牌、重置 stores、重新取 `/me`；只换了令牌时不动；
  - 启动时第一次续期的三种结果：200 已登录；401 未登录；429、5xx、网络错误进入"会话暂不可用"，保留记录并退避重试，不跳到登录页；
  - 401 → 续期 → 重发一次；续期得到 401、或重发后仍是 401 时结束会话；续期得到 429、5xx、网络错误时保留记录并退避；
  - 没有 `nerve.auth` 时不请求 `/me`；
  - 退出在锁内用不挂认证中间件的客户端，交出的是最新一代。
- `isValidNextPath` 的控制字符；`next_path` 的拼接（路径 + 查询 + 片段，整体编码）。
- 错误码到文案键的表与 `api/dist/openapi.yaml` 中的 `x-problem-codes` 相等；字段码的表覆盖 `FieldError.code` 的全部取值。
- 密码规则的 128 上限。

### 9.5 端到端
- **fixture 的扩展**（M0-P6 交接）：
  - **`auth.ts`**：
    - 通过接口注册、登录，每个测试用自己的邮箱（由测试 id 生成），返回令牌；
    - `signedInPage`：用 `context.addInitScript` 在页面加载前把 `nerve.auth`（刷新令牌和一个 `login_id`）写进 localStorage，页面自己续期得到访问令牌；
    - `pat`：通过接口创建 PAT。
  - **`db.ts`**：每个 worker 一个 `pg` 连接池。断言函数按表放在 `e2e/fixtures/assert/identity.ts`，页面版本和接口版本调用同一个函数；因凭证种类而不同的预期用参数传入（A7）。
  - **`server.ts`**：
    - `startNerve` 接受额外的环境变量，用来启动"关闭注册""访问令牌很短""限流很低"几种独立的 nerve，只在需要它的测试中存在；
    - nerve 在 `127.0.0.1:0` 上监听，把实际地址写进 `server.addr_file`，fixture 直接读取，不再先猜端口（M0-P6 交接：端口竞争的根本解决）；
    - `runNerve` 支持标准输入（A13、A17）。
  - **失败时的产物**：
    - 在全局容器里对本 worker 的库执行 `pg_dump`，快照和 trace、截图、nerve 日志放在一起；
    - 不录像：trace 已经有每一步的截屏，录像还要多下载 ffmpeg。P1 把总体设计 8.2 的措辞改为"trace、截图、日志和数据库快照"（M0-P6 交接）。
  - **新加的等待**都受 fixture 自己的期限约束（M0-P6 交接）。
- **故事**：A1–A17，以及 S1–S3 的更新（第 2 节）。A4 另跑一遍删掉 `navigator.locks` 的版本。
- **耗时**：M0 的 `e2e` 任务约 120 秒，超时 20 分钟。M2 加入 17 个故事文件，P5 结束时实测，写进 review。

### 9.6 控制者的浏览器核对
沿用 M1 设计 7.5 的做法：临时脚本不进仓库，脚本全文、假数据、运行命令和断言写进该 Phase review 的附录。浏览器核对是单元测试和端到端测试之外的证据，不代替确定性的并发测试和数据库测试。
- **P4**：
  - 登录页、注册页的就地错误（包括"密码太常见"）；
  - `next_path` 的各种取值；
  - 两个标签页的续期和退出；
  - **两个账户、两个标签页**（Codex I-5、控制者复核 R4），两种走法：
    - 标签页甲以 X 登录并保持登录，不退出；在标签页乙中通过接口登录 Y，按令牌管理器的写入路径换上 Y 的记录。标签页甲切换为 Y，此后它发出的写请求都是 Y 的；甲在切换前后正好续期时，也不会把 X 的令牌写回去；
    - 标签页乙先退出，再以 Y 登录：标签页甲收到"记录出现"，以 Y 进入；
  - **用 HTTP 在局域网 IP 上打开**（nerve 监听 `0.0.0.0`，浏览器打开 `http://<本机局域网 IP>:<端口>`，不是 localhost，所以不是安全上下文）：确认 `window.isSecureContext` 为假、`navigator.locks` 不存在；开两个标签页，等访问令牌过期后同时操作，两边都成功，数据库中这个会话未被撤销；启动日志有非回环地址的提醒；
  - **会话暂不可用**：已登录时停掉 nerve、刷新页面，显示等待和重试，没有跳到登录页；启动 nerve 后自动恢复；
  - 实例请求失败时的维护页；
  - **第一次打开 `/onboarding` 和登录后的落点**：没有失败的接口请求，没有发往 M3 旧接口的请求，没有未处理的 Promise 拒绝，控制台没有错误（Codex I-11）；资料步骤和下一步的出现；
  - 登录页、新手引导页、设置页没有 CSP 违规；
  - 控制台没有"navigate() 应在 useEffect 中调用"的警告。
- **P5**：
  - 个人设置的四个标签页；
  - 主题下拉框在按钮旁展开（zh-CN、en 各一次）；
  - PAT 创建后只显示一次、CSV 下载；
  - 安全页的字段错误，以及安全页的 PAT 列表；
  - 停用账户：确认弹窗、回到登录页、再登录显示"账户已停用"；`nerve users activate` 之后能登录。

---

## 10. 负责人决策点

下面四项会改变用户能做什么，由项目负责人决定。四项都已由负责人裁定（2026-09-25）；选项和代价留在下面备查，正文各节都已按裁定写定。

### 决策点 1：怎样修改登录邮箱（M1 设计 3.13）
**背景**：
- M1 删掉了"向新邮箱发验证码"的旧流程。v0 没有邮件服务，没法证明新邮箱属于这个人。
- 邮箱是登录名，M3 的工作区邀请也按邮箱发出。

| 选项 | 做法 | 好处 | 代价 |
|---|---|---|---|
| A. 自助修改 | 个人设置里输入当前密码和新邮箱（`POST /api/v0/me/change-email`），撤销其他会话 | 用户自己就能改 | 多一个接口、一个表单和它的文案。新邮箱不经验证就生效：填错了也会生效；可以改成别人还没注册的邮箱，对方之后无法注册；如果 M3 按邮箱匹配邀请，改过去的人还会收到发给对方的工作区邀请（与决策点 2 的①相同） |
| B. 只能由服务器管理员修改 | `nerve users set-email --email <旧> --new-email <新>`，撤销该账户的全部会话 | 代码最少：一个命令，复用同一个用例。管理员可以先核实身份，和"忘记密码由管理员重置"是同一种信任方式 | 用户要找管理员 |
| C. v0 不提供 | — | 没有代码 | 邮箱写错或公司换了域名，只能新建账户，数据留在旧账户上 |

**建议：B**。改邮箱很少发生；v0 没有邮箱验证，让管理员经手更稳妥；成本最低。以后接入邮件服务时，再做带验证的自助修改。

**负责人裁定（2026-09-25）：B**。
- 只能由服务器管理员用 `nerve users set-email --email <旧> --new-email <新>` 修改，撤销该账户的全部会话。没有界面，也没有接口。
- 实现在 P3：命令和它的用例；两个邮箱都按注册时的规则规范化；新邮箱已被使用时，命令失败并给出明确的说明（3.17、A16）。
- 命令的说明和 README 写明：`set-email` 不撤销 PAT，不是账户被盗后的恢复手段；改邮箱与安全有关时，另外执行 `reset-password`（Codex §4）。

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
  - **C 比 B 少一层保护**（第二稿说选 B"让开发和端到端测试每次都要先建账户，却没有安全上的收益"，不对）：部署时忘了设 `NERVE_ENV=prod`，程序按 dev 的默认值运行，注册是开放的，签名密钥也是临时的；B 在这种情况下注册仍然关闭。缓解：dev 的配置只监听回环地址，要显式改监听地址才会暴露；非 prod 而监听在非回环地址时，启动日志提醒这两个后果（6.1）；M8 的镜像设 `NERVE_ENV=prod`（13.2）。换来的是开发和端到端测试不用先建账户。

**负责人裁定（2026-09-25）：C**。
- prod 默认关闭注册，dev 和 test 默认开放（6.5）。`platform/config` 的加载测试证明 prod 在没有任何覆盖时是关闭的；端到端"关闭注册"的 nerve 只证明覆盖项和界面（A2）。
- 命令 `nerve users create --email …`：密码从终端或标准输入读，与 `reset-password` 相同；复用注册的"建账户"一步，在 P3 实现（3.17、A17）。README 写明第一个账户的创建步骤（8.7）。
- 上面的缓解（启动提醒、M8 的镜像）照做。
- 给 M3 的邀请交接不变，仍由 M3 的设计交负责人确认。prod 默认关闭注册，不能代替接受邀请时的身份证明（Codex §4）。

### 决策点 3：停用账户
**背景**：
- Plane 在个人设置里提供自助停用（`DELETE /api/users/me/`，`plane/apps/api/plane/app/views/user/base.py:252-348`）：
  - **本意**是：账户是某个项目或工作区唯一的管理员、而且那里还有别的成员时，拒绝停用（`:265-305`）。
  - **实际上这个检查从不拒绝**（按代码推导，未运行 Plane）：两段查询都只取这个用户自己的成员记录，按每一条分组计数。`Count(Case(…, default=0))` 对每一行都计 1，所以"存在其他管理员"恒为真；`total_members` 也恒为 1。停用后可能留下没有管理员的项目或工作区。
  - 其余步骤：停用这个用户的全部成员关系；删除发给这个邮箱的全部工作区邀请（`:313`）；删除全部会话（`:316`）；重置新手引导（`last_workspace_id`、`is_tour_completed`、`is_onboarded`、`onboarding_step`，`:318-331`）；把密码改成随机值；发一封邮件。
  - 不要求输入密码。只有命令 `activate_user` 能恢复。
- Nerve 的前端保留了这个按钮和弹窗（`web/apps/web/core/components/settings/profile/content/pages/general/form.tsx:398-408`、`web/apps/web/core/components/account/deactivate-account-modal.tsx`）。
- 总体设计 4.2 提到"账户停用时让所有会话失效"。
- 成员关系和邀请属于 M3。"唯一管理员"的检查必须能**拒绝**停用，所以不能是停用之后再发出的事件：
  - 停用用例声明一个端口，由 M3 实现，在停用的同一个事务里调用；它返回错误时，整个停用回滚。
  - M2 没有成员数据，不写空实现。端口和它的实现在 M3 一起加入；M2 的停用只处理账户、会话和资料。

| 选项 | 做法 | 好处 | 代价 |
|---|---|---|---|
| A. 照搬 Plane 的自助停用，另加管理员命令 | `POST /api/v0/me/deactivate`：账户不能再登录，全部会话撤销，PAT 随之失效，新手引导的进度重置。`nerve users deactivate / activate --email`：管理员停用离职的人，以及恢复。M3 实现上面的端口：唯一管理员时拒绝（按 Plane 的本意，修正它的查询缺陷，登记差异），停用成员关系，删除发给这个邮箱的邀请。不把密码改成随机值：Nerve 用 `is_active` 挡住登录，恢复后原密码仍可用 | 保留现有的界面，和 Plane 一致；离职的人也有办法停用 | 一个接口、两个命令、一个给 M3 的端口。和 Plane 一样不要求输入密码：拿到**任何凭证（包括 PAT）**的人都可以停用这个账户（管理员能恢复） |
| B. 只能由管理员停用和恢复 | `nerve users deactivate / activate --email`；删除界面上的按钮和弹窗。M3 的端口同 A | 不会误操作，也不会被盗用的凭证停用；离职由管理员处理，贴近团队的用法 | 用户不能自己注销；删掉一个保留功能的界面 |
| C. v0 不提供 | 删除按钮和弹窗，不做接口和命令 | 没有代码 | 总体设计 4.2 的"账户停用"没有入口；员工离职后只能由 M3 把他移出工作区，账户本身仍能登录，PAT 仍然有效 |

**建议：A**。它是保留的功能，照搬 Plane；管理员命令补上了离职的情况。A、B 都需要 M3 的端口，A 只比 B 多一个接口。

**负责人裁定（2026-09-25）：A**。
- `POST /api/v0/me/deactivate`，以及 `nerve users deactivate`、`nerve users activate --email`（3.17、5.1）。
- 停用撤销全部会话（`deactivated`）。PAT 不删除，但认证要求账户未停用，所以停用期间失效。
- 重置新手引导的四项，不改密码。
- 自助停用、管理员停用和恢复都按账户行锁执行（3.5）。
- M3 的端口在同一个事务里拒绝停用唯一的管理员（13.2）。
- 风险写明：任何凭证（包括 PAT）都能停用自己的账户（8.6）。
- `activate` 的说明和 README 写明：恢复后 PAT 重新可用；怀疑账户被盗时，另外执行 `reset-password`。
- A12 按页面、数据库、接口（PAT）三层写定（第 2 节）。

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

**负责人裁定（2026-09-25）：A**。删除两步和 `is_self_managed`；`profiles` 为 11 列，不建 `role`、`use_case`（3.19、4.3、5.2）。

---

## 11. 负责人确认的事项与架构问题

### 11.1 退出登录只结束当前这一处登录（负责人已批准，2026-09-25）
- **总体设计 4.2 原来的原文**："退出登录、修改密码、账户停用时，吊销该账户的刷新令牌。"按字面读，在一处点"退出"，这个账户在所有浏览器、所有设备上的登录都会结束。
- **M2 的做法**（3.5）：
  - 在一处点"退出"，只结束这一处的登录；其他浏览器、手机上的登录不受影响。
  - 修改密码时，结束其他各处的登录，保留正在操作的这一处。
  - 账户停用时，结束全部登录。
  - 这和 Plane 的行为相同（Plane 的退出只结束当前的会话，`plane/apps/api/plane/authentication/views/app/signout.py:25`）。
- **另一种做法**："退出"就是"在所有地方退出"。好处是规则简单，丢了设备时有办法结束那里的登录。代价是在公司电脑上点退出，手机上也被迫重新登录，而且与 Plane 不同。以后需要"在所有设备上退出"时，可以作为单独的按钮加入。
- **负责人批准 M2 的做法**。总体设计 4.2 按 3.20 的规则在实现它的 Phase 改写：退出（P2）；修改密码和停用（P3）。改写后的原文："退出只结束当前的登录；修改密码结束其他各处的登录；账户停用时结束全部登录。"

### 11.2 架构问题
- 没有需要负责人裁定的架构问题。
- 评审提出的"平台包能否导入 `internal/shared`"，已由控制者裁定为不导入（3.3）：平台自己声明认证器和错误接口，`identity`、`shared` 按结构满足它们。这保持了总体设计 6.2"平台与业务无关"，M0 设计 3.1 不用改。代价是 `shared.Error` 带着 HTTP 状态（3.11）。
- 第三稿新定的平台约定（按控制者的裁定，经 spike 选定做法）：
  1. **请求体的结构校验**（3.11）：新增一个平台子包 `httpserver/bodyshape` 和一个构建时的生成器 `tools/bodyshapegen`。生成器在工具模块，用 oapi-codegen 自己的加载器和 `type-mapping` 读契约，不链接进 nerve；运行时只用标准库和 oapi-codegen 的运行时类型。
  2. **凭证签发和变更的账户行锁**（3.5）：`FOR NO KEY UPDATE` 和加锁顺序是全局约定，以后签发或变更凭证的写入（例如 M3 的邀请令牌）照做。
  3. **迁移归被改表的模块所有**（3.14），由 `TestSQLCSchemaScope` 守住。
  4. **提交不受请求期限的取消**（3.6）：`TxManager` 的 `COMMIT` 在 `context.WithoutCancel` 下执行，有自己的期限。
- 以下几处看起来像与上级设计冲突，核对后都在现有规则内，已作为裁定写在第 3、5 节：
  1. **M1-P4 交接要求服务端校验 `next_path`**（3.18）：它的前提（服务端发出跳转）在 JSON 接口下不存在了。
  2. **后续 M 的字段**（3.2）：本身可为空的引用字段随实体进入接口，值为 `null`；实例配置的字段按 M1 设计 3.7 在 M2 定义。
  3. **端口放在 `app` 而不是 `domain`**（3.3）：M0 起就是这样，P1 改写总体设计 6.2。
  4. **注册返回令牌**（5.1），与总体设计 3.1"POST 返回完整资源"的字面不同，P1 写进 3.1。
  5. **续期和退出的服务端期限放在 `identity` 的适配器**（3.5），不在平台：它是轮换协议的一部分，平台的请求期限对一个模块一视同仁。

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

每个 Phase 合并时：持续集成的全部门禁、S1–S4 和此前已加入的 M2 故事都必须通过；3.20 中标着这个 Phase 的上级文档和差异清单都已在同一次合并中同步；8.7 中标着这个 Phase 的 README 内容已写好。

**为什么这样分**：
- **先后端，后前端**：前端要对接真实的接口，M2 不写假后端。
- **后端分三段**：平台约定和第一个能用的认证（P1）→ 会话的安全逻辑（P2）→ 账户的其余接口（P3）。
  - 平台约定约束后续每个 M；会话的轮换、重复使用检测和限流是安全上最敏感的代码。两者放在同一个 Phase 里，评审时会互相稀释（评审 I10；M1 的教训）。第一稿的 P1 约有 20 个任务，两者都在里面。
  - P1 的完成线包含默认拒绝和请求体结构的整程序测试：需要令牌的 `getMe` 合并时，安全网同时合并（评审 I9）。
- **前端内先做令牌管理器和登录页**：设置页要靠它们才能访问。
- **每个 Phase 合并的都是能用、有测试的代码**：后端 Phase 用接口版本的故事验收，页面版本随前端 Phase 加入。
- **比较过的其他拆法**：
  - 按功能前后端一起切（"认证""账户"各一个 Phase）：一个 Phase 同时评审 Go 和 TS，令牌管理器还要等后端做到一半。
  - 单独的"基础设施"Phase：会合并一批暂时没人用的代码（M0 设计 7.2 不这样做的理由）。P1 用 `register` 和 `getMe` 让平台约定当场有使用者。

### P1 `platform-core`：平台约定与第一个认证（后端）
- **目标**：任何调用方都能注册，并用注册得到的令牌访问 `GET /me`；不带令牌访问非公开操作一律 401；不合契约的请求体一律 400；后续 M 照做的平台约定全部定下。
- **交付物**：
  1. 表结构约定（3.13，含审计列和 JSON 列的 CHECK）；迁移 `00001`–`00003`（`users`、`profiles`、`auth_sessions`，4.1）；sqlc 接入（3.14），按模块限定 `schema`，`TestSQLCSchemaScope` 含 `ALTER TABLE` 的所有者；架构测试规则 4 加上 `internal/shared`，规则 6 推广到 `adapter/*/gen`。
  2. 平台：
     - `internal/shared`（游标的封套到 P3 随第一个列表加入）、`platform/postgres` 的事务（`COMMIT` 不受请求期限的取消，3.6）和 UTC、`platform/clock`（固定时钟截到微秒，3.13）；
     - `httpserver` 的 `Router`、`API` 值、默认拒绝的认证中间件、请求元信息、请求期限、请求体上限、`ProblemError` 的映射和新平台码、导出 `RequestID`（3.6、3.11）；
     - 请求体的结构校验：`httpserver/bodyshape`（只用标准库；含 `date-time`、`uuid` 的格式检查器，第一个使用者在 P3）、`tools/bodyshapegen`、`make gen-go` 的接入（3.11）；
     - 工具模块进持续集成：`make test` 另跑 `go -C tools test`，`make lint-go` 另在工具模块跑 golangci-lint（3.11）；
     - 配置（6.5 中 P1 用到的部分，含按环境的注册默认值和 `LogValue`），M0-P2 交接 8 的三个小问题，启动时的环境提醒（6.1）。
  3. 接口描述：oapi-codegen 的模块模板（含 `email` 映射为 `string`）；`security` 的写法和 `apitest` 的两条规则；`x-problem-codes` 和 `apitest` 的三项核对；`FieldError.code`；`CheckRequest`（3.11、3.12）。
  4. `identity`：
     - `register`、`getMe`；
     - argon2id，含并发上限和等待上限；密码规则（新的主干）、常见密码名单和生成它的脚本（3.8）；
     - `signing` 适配器：Ed25519 JWT、刷新令牌的 MAC 标签和密钥（3.4、3.7）；注册时建会话、发令牌；
     - 认证用例和 `authn` 适配器（JWT 加会话检查）。
  5. `bootstrap` 的四个整程序测试（3.6、3.11）。
  6. 端到端：
     - fixture 的扩展（9.5，页面的登录状态除外；`auth.ts` 这时只有注册）；
     - S1 的迁移断言；
     - A1、A2 的接口版本。
  7. 3.20 中 P1 的各行（含差异清单：一 B、二·全局，`users`、`profiles`、`auth_sessions` 逐列）；8.7 中 P1 的 README 内容。
- **关闭**：
  - M0-P1；
  - M0-P2 的第 1、3、4、6、7、8 条和第 2 条中认证的部分（第 4 条按原文：`*_file` 不记路径）；
  - M0-P3 的第 1、3、4、5 条，以及第 2 条中请求体解码和 handler 两个错误出口。M0-P3 交接整体在 P3 关闭（参数绑定的出口）；
  - M0-P4；
  - M0-P6 中与 fixture、数据库断言、S1、端口、等待期限、录像有关的几项。
- **完成线**：
  - A1、A2 的接口版本和 S1–S4 在持续集成中通过；`platform/config` 证明 prod 默认关闭注册的测试通过；
  - 四个整程序测试和 `apitest` 的新核对通过；
  - `make gen-check` 覆盖 sqlc 和请求体结构表的输出，`bodyshapegen` 两次生成的输出相同；
  - 架构测试和传递依赖测试通过；
  - 审计列等于固定时钟的集成测试、CHECK 的反例测试通过；
  - `TxManager` 在请求的 `context` 被取消后仍然提交、仍然回滚的集成测试通过；
  - `make test`、`make lint-go` 覆盖工具模块，持续集成的日志里能看到 `bodyshapegen` 的测试。

### P2 `sessions`：登录与会话（后端）
- **目标**：登录、续期、退出可用；刷新令牌的轮换和重复使用检测、会话的绝对期限、限流、安全响应头全部到位。
- **交付物**：
  1. `login`（按账户行锁签发；假哈希；停用只在密码正确后报出；重新哈希）；`refreshTokens`（按会话 id 取出会话行；当前一代比对 `token_hash` 后条件轮换；旧代由 MAC 标签确认是这个会话真的发过的才撤销，其余 401 不撤销；绝对期限）；`logout`（只认当前一代）；续期和退出的 4 秒语句期限，提交另有期限（3.5、3.6、3.9）。
  2. 限流：`platform/ratelimit`（自己的令牌桶：突发、`AllowAll`、`Reserve` 和退回）；`httpserver` 的限流中间件和认证之前的失败闸门（先预留；签名有效而只是过期的 JWT 不计数）；`identity` 适配器中登录、注册的桶（续期、退出只经过 `anonymous`）；客户端 IP 键（IPv6 取前缀）；`server.trusted_proxies` 和不可信 `X-Forwarded-For` 的提醒（3.6、3.10）。
  3. 固定链上的安全响应头（8.3）。
  4. 端到端：`auth.ts` 加上登录；A3、A4、A5、A6、A15 的接口版本。
  5. 3.20 中 P2 的各行；8.7 中 P2 的 README 内容。
- **关闭**：M0-P2 第 2 条中限流的部分；M0-P5 中安全响应头的部分。
- **完成线**：
  - A3–A6、A15 的接口版本通过，P1 的故事仍然通过；
  - 登录的耗时测试（邮箱存在与否，耗时相同）通过；
  - 续期的四个测试通过：伪造的旧代（随机密文加随机标签）401、不撤销；过期访问令牌里的 `sid` 加 `g = 0` 401、不撤销；真实的旧令牌撤销；换了签名密钥之后，换钥之前的旧代令牌 401、不撤销，当前一代照常续期；
  - 3.10 表中除 `password_user`（随修改密码在 P3 加入）外的每个桶都有测试，续期、退出只经过 `anonymous`；客户端 IP 键的 IPv6 前缀有测试；
  - 失败闸门用计数的假认证器证明：超额后不再调用认证器；50 个并发的无效令牌也不超过额度；签名有效而只是过期的 JWT、成功、数据库错误都退回单位；
  - `platform/ratelimit` 的测试覆盖突发、`AllowAll`、`Reserve` 的退回和闲置键的清理。

### P3 `account-api`：账户接口（后端）
- **目标**：账户的其余接口都可用，而且都能用 PAT 完成；管理命令可用；River 的第一个定时任务运行；凭证的签发与变更在并发下仍然正确。
- **交付物**：
  1. `updateMe`、`getProfile`、`updateProfile`（`onboarding_step` 在 SQL 中合并）、`changePassword`（账户行锁、3.5 的撤销规则、`password_user` 桶）、`deactivateMe`（账户行锁）。
  2. PAT：
     - 迁移 `00004`；
     - 创建（账户行锁下复核凭证）、分页列表（`shared` 的游标封套、`common.yaml` 的分页组件、PAT 列表的游标载荷、`limit` 的 422）、撤销（`DELETE /api-tokens/{token_id}`）；
     - PAT 认证和 `last_used`。
  3. `instance` 的三个字段和 `listTimezones`；嵌入 `time/tzdata`。
  4. 管理命令：`nerve users create`、`reset-password`、`set-email`、`deactivate`、`activate`；`Admin()`（3.17）。
  5. River（3.15）：
     - 迁移 `00005`；
     - `platform/jobs`（服务用和只投递两种客户端）；
     - `bootstrap` 的运行和停机顺序；命令行的最小组合（3.17）；
     - 清理过期会话的定时任务（`SKIP LOCKED` 分批）。
  6. 账户行锁的六个交错测试（3.5、9.2）。
  7. uuid 守卫的改写（3.12）：传递依赖测试只允许 `oapi-codegen/runtime` 模块导入 `github.com/google/uuid`；archtest 加上"生成文件不得引用 `openapi_types.UUID`"；与第一个带参数的操作同一次合并。
  8. 请求体格式检查的第一批使用者（`createApiToken` 的 `expired_at`、`updateProfile` 的 `last_workspace_id`）和第四个整程序测试中逐格式、一次收集多种问题的情况（3.11）。
  9. 端到端：
     - A7–A14、A16、A17 的接口版本（需要登录的都用 PAT）；
     - S3；
     - 实测 nerve 的停机时间。
  10. 3.20 中 P3 的各行（含差异清单的 `api_tokens` 逐列和 River 的表）；8.7 中 P3 的 README 内容。
- **关闭**：
  - M0-P2 第 5 条；
  - M0-P3 整体（参数绑定的出口由 `listApiTokens`、`revokeApiToken` 测过）；
  - M0-P6 的 S3 和 River 停机两项；
  - M1-P2、M1-P3 中接口描述的部分：主题字段；实例字段；不再读的字段和不再调用的地址都不出现在接口描述里；令牌地址不带结尾 `/`；
  - M1-P3 中修改登录邮箱的一项（决策点 1）。
- **完成线**：
  - A7–A14、A16、A17 的接口版本通过，P1、P2 的故事仍然通过；
  - 每个需要登录的操作都有 PAT 的测试；
  - 四个整程序测试覆盖新加的全部操作，包括每个带格式的字段写错时 400 `invalid_format`；
  - 账户行锁的六个交错测试通过；
  - 带着 `oapi-codegen/runtime` 的 nerve 通过改写后的传递依赖测试，生成文件的 uuid 规则通过（9.2）；
  - `onboarding_step` 的并发合并测试通过；`limit` 为 0、101、`abc` 的测试通过。

### P4 `web-auth`：前端认证
- **目标**：浏览器通过令牌管理器登录、续期、退出；Cookie 和 CSRF 从前端消失；用 HTTP 部署时多标签页也能正常续期；切换账户时标签页不会以错误的身份写入；M2 能到达的页面挂载时不请求 M3 的旧接口。
- **交付物**：
  1. `@nerve/api-client` 接入 web（`--root-types`）；令牌管理器（`nerve.auth` 和 `login_id`，每次写入都在同一把锁或租约下，续期写回之前核对 `login_id`；四种 `storage` 事件，含本标签页未登录时记录出现；"会话暂不可用"；两条路径上的 8 秒超时；只有续期 401 才结束会话）、跨标签页的协调（`navigator.locks` 和租约）和它们的测试（7.1）。
  2. 传输层的清理：CSRF 4 处、表单提交、web 的 axios 基类（7.2）。
  3. 登录页、注册页、错误文案表（7.3）；`next_path`（3.18）。
  4. `AuthenticationWrapper`（已完成引导的用户直接去 `/create-workspace`；"会话暂不可用"的界面）、`InstanceWrapper`（7.4）；user、profile、instance 三个 store 和相关类型，以及删除 `IUser` 牵连的地方（7.5 中与认证有关的部分）。
  5. 新手引导：删除"角色""用途"两步和 `is_self_managed`（3.19）；删除挂载时对工作区和邀请的预取（3.1）；资料步骤，删掉它的头像上传（3.2）。
  6. `webui` 的 CSP（8.3）。
  7. 关键词规则（7.9 中 P4 的部分）。
  8. 端到端：
     - `signedInPage` fixture；
     - A1–A6（A6 含切换账户）、A10（含第一次打开 `/onboarding` 的核对）、A15 的页面版本（A4 含删掉 `navigator.locks` 的一遍）；
     - S2。
  9. 3.20 中 P4 的各行；8.7 中 P4 的 README 内容。
- **关闭**：M0-P5 中 CSP 和前端的部分，以及"安全响应头与 CSP 放在同一层"一项（按 8.3 的理由正式关闭，写进 review）；M1-P3 中 CSRF、认证错误、实例字段的部分；M1-P4；M0-P6 的 S2。
- **完成线**：
  - `git grep -n -E "csrfmiddlewaretoken|X-CSRFTOKEN" -- web` 没有输出；
  - 上述故事的页面版本通过；
  - 9.6 中 P4 的浏览器核对（含局域网 HTTP、两个账户两个标签页、会话暂不可用、第一次打开新手引导和登录后的落点）写进 review。

### P5 `web-account`：前端账户设置
- **目标**：个人设置的四个标签页对接新接口；M2 领域的前端清理完毕。
- **交付物**：
  1. general（含停用账户）、preferences、security（含 PAT 列表）、api-tokens（7.7）；PAT store（7.5）；删掉 general 页的头像和封面上传控件（3.2）；主题下拉框的定位。
  2. `@nerve/services` 删到只剩地址工具和文件工具；删除剩下的 Plane 类型（7.5）。
  3. 清理：
     - 死成员和死 prop 的 66 行；
     - oxlint：改到的文件清零，`no-unneeded-ternary` 一类清零，上限调低（7.8）。
  4. 关键词规则（7.9 中 P5 的部分，包括 `withCredentials:\s*true`）。
  5. 端到端：A7–A9、A11、A12 的页面版本。
  6. 3.20 中 P5 的一行。
- **关闭**：M1-P2；M1-closeout；M1-P3、M1-P4 剩下的事项。
- **完成线**：
  - 全部故事的页面版本和接口版本通过；
  - `domains.mjs --rows M2` 没有输出，或者剩下的每一行都在 review 中写明是误报；
  - 9.6 中 P5 的浏览器核对写进 review。

### 收尾 `closeout`
- 10 份交接逐项写下结论，`status` 改为 `closed`。
- 按 3.20 逐行核对上级文档和差异清单都已在各 Phase 同步；差异清单按第 4 节逐列总核对。收尾不再做首次同步。
- 写好给 M3–M8 的交接（13.2）。
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
| | 2 认证、限流、接口调用日志按路由挂载 | P1 认证（3.6）；P2 限流和失败闸门（3.10）。接口调用日志由 M8 挂在限流之后（13.2） |
| | 3 导出 `RequestID` | P1 |
| | 4 新的密钥类配置写进 `LogValue` | P1。按原文关闭：`*_file` 只记是否设置，不记路径（3.7） |
| | 5 River 与停机顺序、连接池关闭的时限、handler 的期限、River 的迁移 | P1 请求期限（3.6）；P3 River、停机和迁移（3.15） |
| | 6 规则 6 推广到 `adapter/*/gen` | P1（3.14） |
| | 7 迁移与就绪检查、迁移角色的权限 | P1：S1；README 的迁移角色一行（8.7），与第一批迁移同时 |
| | 8 布尔配置的空值、logger 之后的致命错误、健康检查的日志 | P1（6.1） |
| M0-P3-api-codegen-notes | 1 生成选项 | P1（3.12） |
| | 1 中"P3 新增的架构测试会拦住对 google/uuid 的传递依赖" | P3。`runtime` 本身就带进 google/uuid，这个拦法与引入 `runtime` 冲突；守卫改为直接检查：google/uuid 只允许经由 `runtime` 模块，生成文件不得引用 `openapi_types.UUID`（3.12） |
| | 2 错误映射、校验层、解码错误、413、`context.Canceled`、错误出口的测试、`CheckRequest` | P1：映射、分层（结构在边界，取值在领域）、请求体解码和 handler 两个出口；P3：参数绑定的出口（`listApiTokens`、`revokeApiToken`）。交接在 P3 整体关闭；不为测试在生产的接口描述里加操作（3.11） |
| | 3 `security` 的写法 | P1（3.12） |
| | 4 模块入口的演进 | P1（3.6） |
| | 5 接口描述的组织规则、map 型对象 | P1（3.12；M2 没有 map 型对象） |
| M0-P4-schema-conventions | 1 外键的 `DEFERRABLE` 和 `ON DELETE`、命名、默认值和 CHECK、系统表、Django 的索引、`varchar` | P1（3.13；删除关系图 4.7） |
| | 2 主键与 ID | P1（照旧：`uuid`，应用生成） |
| M0-P5-frontend-api-notes | 改调 `/api/v0/instance`；认证接口在 `/api/v0/` 下；同源 | P1、P2（接口路径）；P4（前端） |
| | 安全响应头与 CSP 放在同一层 | P2（固定链上的响应头）；P4（CSP）。分在两层的理由见 8.3，P4 正式关闭这一项 |
| M0-P6-e2e-notes | PAT 对等验收与认证 fixture | P1（注册）、P2（登录）、P3（PAT）、P4（页面） |
| | `db.ts` 的连接池、断言的写法、失败时的数据库快照 | P1（9.5） |
| | S1 的迁移版本 | P1 |
| | S3 的新字段 | P3 |
| | S2 的"没有失败的接口请求"、控制台、`networkidle` | P4。没有刷新令牌时不请求 `/me`（7.1）；登录页没有轮询，`networkidle` 继续可用；有轮询的页面不进 S2 |
| | 端口竞争的根本解决 | P1（`server.addr_file`） |
| | 新等待的期限 | P1–P5 |
| | River 停机是否仍在预算内 | P3 |
| | 是否录像 | P1 定为不录像，同时同步总体设计 8.2（9.5） |
| | fixture 写法的延伸（`storage.ts`、`webhook.ts`、`clock.ts`） | 不属于 M2：M5、M8、M4 各自按 P6 的写法加入 |
| M1-P2-trim-content | 主题只剩一个值，五个取值，没有 `custom` 和调色板 | P3（接口，5.2）、P4（`IUserTheme` 删除，改用生成的类型） |
| M1-P3-trim-platform | 删除 Cookie 会话和 CSRF | P4（7.2） |
| | 认证错误就地显示 | P4（7.3） |
| | 修改登录邮箱 | 决策点 1，负责人 2026-09-25 裁定为 B：`nerve users set-email`，在 P3 实现（3.17、A16）；没有界面和接口 |
| | `IInstanceConfig` 的 4 个字段；`is_self_managed` 和新手引导的两步；`enable_signup` 的名字 | P3（接口，5.3）、P4（前端）；两步按决策点 4（已裁定为 A）删除 |
| | 不再读的用户字段、不再调用的地址不出现在接口描述里 | P3；收尾再核对一次 |
| | 令牌地址统一不带结尾 `/` | P3（5.1） |
| | 新手引导挂载时请求 M3 的旧接口（交接没有列出，Codex I-11 发现） | P4（3.1）：删除预取；A10 和浏览器核对第一次打开 |
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
| M3 | **邀请**（负责人确认的产品改动）：邮箱未经验证期间，接受邀请不能只靠邮箱匹配，要凭邀请链接中的令牌（M1 设计 3.15 留下的路径）；关闭注册时，持有有效邀请的人仍可注册。后者由 M3 扩展 `SignupPolicy` 的实现和注册请求（加上邀请令牌），不另加端口（评审 M16）。这改变总体设计 1.1"系统内接受邀请"和 4.2"被邀请的邮箱始终可以注册"的做法，由 M3 的设计交负责人确认。prod 默认关闭注册（决策点 2）不能代替接受邀请时的身份证明。参考：Plane 把任何未删除的工作区邀请都算上（`plane/apps/api/plane/authentication/adapter/base.py:102-120`）。签发邀请令牌按 3.5 的账户行锁 |
| M3 | **登录后的落点与新手引导的取数**：M2 让已完成引导的用户直接去 `/create-workspace`，删掉了新手引导页对工作区和邀请的预取（3.1）。M3 在新接口上加回：`AuthenticationWrapper` 的落点数据（"上次的工作区"、工作区列表）、新手引导页的工作区和邀请。加回时，工作区取数的 SWR fetcher 要 `return`（或 `await`）`fetchWorkspaces()` 的 Promise：原来的 fetcher（`web/apps/web/app/(all)/onboarding/page.tsx:33-37`，M2 随预取一起删掉）没有，失败会成为未处理的 Promise 拒绝。被调用的 `fetchWorkspaces` 本身在 `web/apps/web/core/store/workspace/index.ts:146-158` |
| M3 | `profiles.last_workspace_id` 是否补外键（`ON DELETE SET NULL`）。补的话，迁移归 `identity`（`<v>_identity_profiles_last_workspace_fk.sql`，3.14），版本号大于建 `workspaces` 的迁移 |
| M3 | `workspace_creation_enabled` 的执行，以及关闭时是否提供创建工作区的命令（3.16） |
| M3 | **停用的端口**（决策点 3 已裁定为 A）：给停用用例加上它声明的端口并实现，在停用的同一个事务里调用。唯一管理员时拒绝（按 Plane 的本意，修正它查询的缺陷，登记差异），并给 `deactivateMe` 声明对应的错误码；停用成员关系；删除发给这个邮箱的邀请 |
| M3 | 3.2 的字段规则：工作区图标、项目封面、成员头像随实体进入接口，M5 之前为 `null`；`IUserLite.avatar_url` 改为可为 `null`；读取它们的代码不动 |
| M3 | sqlc 的模块边界（3.14）：`access` 模块读成员表，是在 `app` 层声明端口，还是作为 `TestSQLCSchemaScope` 中写明理由的例外 |
| M3 | `TestSQLCSchemaScope` 的所有者规则用正则识别 `ALTER TABLE`，漏掉带引号的标识符，以及别的模块的表上 `ALTER` 以外的 DDL（`CREATE INDEX … ON users`、`CREATE TRIGGER`、`DROP TABLE`）；M3 是第一个有跨模块外键的 M，补上这几种写法和它们的反例（P1 评审 M5） |
| M3、M4、M6 | **物理删除与跨模块外键的关系图**：每张新表按 3.13 照搬 `on_delete`，并在 4.7 的图上延伸，写明物理删除时每条外键的去向。Plane `project.py:77-89` 的项目负责人、`cycle.py:65-68` 的迭代负责人都是 `CASCADE`：物理删除一个账户会连带删除项目或迭代。M2 只停用、不删除账户；各 M 写明允许物理删除的范围 |
| M3 | CSP：表情选择器从 `cdn.jsdelivr.net` 下载 `emojibase-data`，改为随前端一起构建、从本站提供（8.3） |
| M3 | 新手引导的创建工作区、加入工作区、邀请成员三步；`user.service.ts` 中留下的 `leaveWorkspace`、`joinProject`、`leaveProject`；`IUserLite` 的 `is_bot`；时区接口也供工作区和项目设置使用 |
| M4 | **游标**：工作项按 `sort_order`、优先级或日期排序，每种排序定义自己的游标载荷，最后以 `id` 保证稳定（3.12）；不复用 PAT 列表的 `(created_at, id)` |
| M4 | **自动归档读 `updated_at`**：它由用例的时钟显式写入（3.13），测试用固定时钟 |
| M4 | **请求体检查的两处延伸**（P1 评审）：字段错误的路径现在按字典序排序（`tags[10]` 在 `tags[2]` 之前），改为按数组下标的数值排序；不限类型的节点（`{}`、开放对象、没有 `items` 的数组）不看数的范围，`1e400` 这类 float64 放不下的数仍然得到解码器笼统的 400，改为在边界上报出。两者都在第一个带数组或开放对象请求体的操作到来时处理 |
| M4 | **草稿发布**复用同一个结构检查：发布时，草稿的 `payload` 按"创建工作项"的请求 schema 走 `bodyshape` 的校验（3.11），与创建工作项走同一条路 |
| M4 | **60 天物理清理**：把软删除的 `api_tokens` 纳入；`issue_activities` 对工作项、评论的外键是 `DO_NOTHING`（不写 `ON DELETE`），先删工作项会被它挡住：先删除或置空这些引用，或者登记为 `SET NULL` 的差异 |
| M4 | `user.service.ts` 中的 `getUserProfileIssues`；事件订阅者的写法（3.15）；CSP：编辑器 callout 的默认表情图来自 `cdn.jsdelivr.net`，改为本站资源或原生表情，并核对表情回应（8.3） |
| M5 | **头像和封面**：`users.avatar_asset_id`、`cover_image_asset_id`。迁移的范例（3.14）：先在 M5 的模块里建 `file_assets`（`<v>_<模块>_file_assets.sql`），再写 `<v+1>_identity_users_avatar_asset.sql` 给 `users` 加列和外键，后者归 `identity` 的 sqlc 条目。`User.avatar_url`、`cover_image_url` 开始返回签名地址；按新的上传协议加回 general 页和新手引导资料步骤的上传控件（3.2） |
| M5 | **上传与按路由的中间件**：模块级的 `Middlewares`（1 MiB 请求体上限、15 秒期限）会让 `/api/v0` 下的上传失败。M5 在平台加按操作的放宽设置，或者把上传放在 `/api/v0` 之外，并按 3.6 的整程序测试处理：写进接口描述，或在设计中说明（控制者复核 m7）。`file_size_limit` 的执行；CSP 的 `img-src`、`connect-src` 加上存储的来源 |
| M6 | 迭代（`cycles`、`cycle_issues`）跨 `planning` 与工作项模块的写入用端口和共享事务；必须联表的查询，事先列为 `TestSQLCSchemaScope` 的例外并写明理由，或者用端口拆开 |
| M7 | 保存视图的列表有多种排序：按 3.12 为每种排序定义游标载荷；跨模块的联表同 M6 |
| M8 | 接口调用日志挂在限流之后（3.6）；Go 进程空闲内存实测时，一并测 argon2 并发上限下的峰值和常见密码名单占用的内存（3.8）；对外接口文档页不从 CDN 加载脚本（8.3）；**镜像设置 `NERVE_ENV=prod`**（决策点 2 的缓解，6.1） |

---

## 14. 完成标准
- [ ] P1–P5 和收尾全部完成，每个 Phase 都有 spec、plan 和 review。
- [ ] A1–A17 的页面版本（有页面的）和接口版本全部通过，S1–S4 通过。需要登录的每个操作都有 PAT 版本，调用同一组数据库断言；因凭证种类而不同的预期由参数表达，并写在故事里（A7）。
- [ ] 后端：单元测试、集成测试（含账户行锁的交错测试）、契约测试（四个整程序测试、`apitest` 对错误码的核对）、架构测试（含规则 4 的扩展和 `TestSQLCSchemaScope`）和 depguard 全部通过；`make gen-check` 覆盖 oapi-codegen、请求体结构表、sqlc 和 TS 客户端。
- [ ] 前端：类型检查通过，knip 为零；oxlint 等于上限，上限已按 7.8 调低；前端单元测试通过；关键词守卫没有未登记的命中。
- [ ] `git grep -n -E "csrfmiddlewaretoken|X-CSRFTOKEN" -- web` 没有输出；前端不再调用 `/auth/…` 和 Plane 的用户、实例、令牌、时区地址（7.9 的规则守着）；M2 能到达的页面挂载时不请求 M3 的旧接口（3.1）。
- [ ] 4 张业务表（`users`、`profiles`、`auth_sessions`、`api_tokens`）由 M2 的迁移创建，以 Plane 表结构快照为起点；River 的表由它自己的迁移创建（3.15）；差异清单在建表的 Phase 登记了全局的建表约定（3.13）、每一列的改动和每一条行为差异（4.6），`sessions` 登记为替换模型。
- [ ] 第 10 节的四个决策点已由负责人裁定（2026-09-25）并已实现；第 11.1 节已由负责人批准，总体设计 4.2 已改写。
- [ ] 浏览器核对（9.6，含局域网 HTTP、两个账户两个标签页、会话暂不可用）的脚本全文写在各 Phase review 的附录中。
- [ ] `handoffs/` 中没有 `open` 的事项；交给 M3–M8 的交接已写好（13.2）。
- [ ] 总体设计、M0 设计、M0/P3 spec、差异清单、前端改动清单已在各 Phase 按 3.20 同步，收尾逐行核对过；README 已按 8.7 写好；总体设计中 M2 的状态改为"已完成"。
- [ ] 收尾 review 写明规模估计与实际的对比（7.10）。

---

## 15. Phase 进度表

| Phase | 名称 | 状态 | spec | plan | review |
|---|---|---|---|---|---|
| P1 | platform-core | 已完成 | [spec](specs/P1-platform-core.md) | [plan](plans/P1-platform-core.md) | [review](reviews/P1-platform-core-review.md) |
| P2 | sessions | 已完成 | [spec](specs/P2-sessions.md) | [plan](plans/P2-sessions.md) | [review](reviews/P2-sessions-review.md) |
| P3 | account-api | 未开始 | — | — | — |
| P4 | web-auth | 未开始 | — | — | — |
| P5 | web-account | 未开始 | — | — | — |
| 收尾 | closeout | 未开始 | — | — | — |

---

## 16. 风险

| 风险 | 应对 |
|---|---|
| 重复使用检测过严：续期的响应在网络上丢失，客户端重试会被当成重复使用，用户被迫重新登录 | 同一浏览器内由跨标签页的协调避免并发续期；服务端期限短于客户端超时（3.5）；发现重复使用时记 WARN 日志，P4、P5 的核对中观察它是否频繁出现。真的频繁时，可以加"上一代令牌在几秒内仍可用"的宽限期（会话行上要多存上一代的哈希和轮换时刻），但要重新审视安全语义，留到以后 |
| 租约不是原子的：没有 `navigator.locks` 时，两个标签页极小概率同时续期；持有租约的标签页被冻结、超过租期时，另一个标签页会用旧令牌续期 | "写入后再读一次"；代价与上一行相同。A4 的无 `locks` 版本和局域网 HTTP 的浏览器核对观察它 |
| 认证之前的失败闸门按 IP 计数：同一个出口 IP 后的攻击流量会让有效的调用方也得到 429；先预留的做法让同一个 IP 同时在认证中的请求不能超过桶里当时剩下的单位（满时是突发 60，已有失败扣掉一部分时更少），数据库变慢时更早碰到 | 额度每分钟 60 次、突发 60；签名有效而只是过期的 JWT 不计数；认证成功立即退回；可信代理配好以后按真实的客户端 IP 计数，IPv6 按前缀（默认 /64，可配，3.10）；配置项可调；M8 做性能实测时观察同一 IP 的并发 |
| 续期和退出只受按 IP 的 `anonymous` 约束：同一出口 IP 后的大量伪造刷新令牌会用掉这个 IP 的匿名额度 | 伪造的令牌每次只花一次按主键的查找；额度每分钟 600 次、突发 100，正常页面每 15 分钟才续期一次；续期得到 429 时前端保留令牌、退避重试（7.1）。不按令牌里的会话 id 另设桶：那是未经验证的输入，会让知道会话 id 的人耗尽别人的额度（3.10） |
| 泄露的刷新令牌可以派生永不过期的 PAT，影响超过 30 天 | 8.5 写明风险和恢复步骤；PAT 列表显示创建时间和最后使用时间；管理员重置密码撤销全部 PAT。重新输入密码、限制派生是负责人以后可选的产品选项 |
| 任何凭证（包括 PAT）都能停用账户 | 与 Plane 相同的产品取舍（决策点 3）；管理员用 `nerve users activate` 恢复；停用写 INFO 日志 |
| 部署时忘了设 `NERVE_ENV=prod`：注册开放、签名密钥是临时的 | dev 的配置只监听回环地址；非 prod 而监听在非回环地址时启动日志提醒；README 写明；M8 的镜像设 `NERVE_ENV=prod` |
| 账户行锁：持锁太久或死锁 | argon2 始终在锁外；锁用 `FOR NO KEY UPDATE`，不挡别的事务的外键检查；全局的加锁顺序（外键检查也算一次 `users` 的锁）；清理任务用 `SKIP LOCKED`；六个交错测试（3.5） |
| 默认拒绝漏声明公开操作：公开的接口被 401 | 整程序测试要求公开集合等于接口描述的 `security: []`，漏一个就失败 |
| 请求体的结构检查：请求体被解析两次；以后的 M 用到生成器不支持的 schema 写法 | 请求体上限 1 MiB，M8 实测开销；生成器遇到不支持的写法时失败并说明原因，由那个 M 带着测试扩展生成器，不会悄悄放过 |
| 每个带令牌的请求多一次数据库查询 | 按主键查询；M8 做性能和内存实测时一起观察 |
| 换签名密钥之后，换钥之前签发的旧代刷新令牌被再次使用时不再能被发现：标签无法验证，按伪造处理（401，不撤销） | 当前一代照常续期，换钥不让用户重新登录；换钥是少见的运维动作，README 写明这个后果（3.7、8.7）；需要时再加 `kid`，过渡期保留旧的 MAC 密钥（3.7） |
| 换签名密钥清空失败闸门：换钥之后，每个标签页手里的访问令牌都验签失败，计入 `auth_failure`；同一个出口 IP 后超过 60 个标签页时，闸门暂时用空，有效的请求也得到 429 | 这些 401 让标签页去续期，续期是公开操作，不经过闸门；当前一代照常续期，新的访问令牌验签通过。闸门每分钟恢复 60 次，前端遇到 429 退避重试（7.1）。换钥在低峰时做，README 写明（8.7） |
| 刷新令牌正被盗用时换钥：谁先用换钥前的当前一代续期，谁就留下这个会话；另一方交出的已是换钥前的旧代，标签无法验证，按伪造处理（401，不撤销）。受害者只是被登出，会话留在攻击者手里 | 恢复：用户修改密码（撤销其他会话），或管理员执行 `reset-password`（撤销全部会话和 PAT）。不为此加 `prev_token_hash` 之类的列：为一个少见、由运维发起的事件多存一列不值得。README 写明（8.7） |
| 会话表的大小 | 每个会话只有一行，续期多少次都不增加（3.5、4.5）；行数只随登录、注册增长，受 3.10 的桶约束；过期的行由清理任务删除（3.15） |
| openapi-fetch 0.17.0 的中间件能否重发请求 | P4 先做原型；不行时在薄 service 层统一包一层"401 后续期重试" |
| 生成的类型替换 `IUser` 等之后，牵连的使用方比 7.5 列出的多 | 类型检查会列出全部；超出估计时按领域拆成更小的任务 |
| 常见密码名单误伤或漏网 | 判定规则确定，单元测试列出命中和不命中的例子（含绕过第二稿的七个写法）；被拒绝时提示明确；名单可以用脚本重新生成 |
| `x-problem-codes` 与实现不一致 | `apitest` 两个方向都核对：返回的码必须已声明，声明的码必须被返回过 |
| sqlc 和 oapi-codegen 共用一个工具模块，依赖被一起抬高后 oapi-codegen 的输出变化 | P1 加入 sqlc 后立即运行 `make gen-check-go`；有变化时把 sqlc 放进单独的工具模块 |
| sqlc 的解析器只认 PG 17 | 3.13 的语法约定；真正需要 PG 18 的语法时，评估升级到已发布的新版 sqlc |
| River 仍是 0.x | 锁定 v0.47.0；升级时按 3.15 另写迁移，不改已发布的迁移 |
| argon2 占用内存和 CPU | 并发上限 4，约 76 MiB；等待上限 2 秒（3.8）；3.10 的桶限制每个 IP 和每个账户能触发的次数；M8 实测 |
| 调高 argon2 参数（`argon2_memory_kib`、`argon2_iterations`）之后，登录耗时能区分休眠的账户（调高之后还没有登录过）和不存在的邮箱：休眠账户的哈希仍是旧参数，密码错误时按旧参数校验，不存在的邮箱按新参数校验假哈希（3.9）；每个账户成功登录一次、重新哈希（3.8）之后才一致 | 与 Django 的 argon2 哈希器的局限相同（它的 `harden_runtime` 什么也不做；Django 的 PBKDF2 哈希器在密码错误时补算新旧迭代次数之差）。Plane 用 Django 默认的 PBKDF2，但邮箱不存在时直接答 `USER_DOES_NOT_EXIST`（3.9），暴露得更多。参数少调；prod 默认关闭注册，登录按 IP 和按 IP + 邮箱限流（3.10）。是否加缓解，留给负责人以后决定 |
| CSP 挡住某个库的内联脚本或外部资源 | 内联脚本的哈希在启动时计算；外部来源已清点并交给 M3、M4、M5、M8（8.3）；S2 统计违规事件 |
| 端到端的耗时增加 | 独立的 nerve 只用于 A2、A4、A15 三个故事；P5 实测并写进 review |
| 新手引导的工作区步骤在 M3 之前失败 | 预期行为（3.1）：只有用户自己提交 M3 的表单时才会遇到 404；页面挂载时不请求旧接口，A10 和浏览器核对守着 |

---

## 17. 评审的落实

### 17.1 第一稿的独立评审（第二稿落实）
第一稿 `f6ea660` 经独立评审（opus）：Ready with fixes，Critical 0、Important 14、Minor 18。控制者对每一条作了裁定，第二稿逐条落实。第三稿又改动了其中几条的做法，括号里注明。

| 编号 | 问题 | 落点 |
|---|---|---|
| I1 | 认证默认放行，整程序测试覆盖不全 | 3.6 默认拒绝和三个整程序测试；6.4；9.3；P1 |
| I2 | 只留组合规则是削弱，却写成"修复不一致" | 3.8 常见密码名单（第三稿改了主干的定义）；4.6；A2 |
| I3 | argon2 的名额可以被占满，挡住所有登录 | 3.8 等待上限和 503；3.10 的桶和每个操作的桶（第三稿加了突发和失败闸门）；6.5 |
| I4 | 非安全上下文中没有 `navigator.locks` | 7.1 租约；8.6；9.4；9.6 局域网 HTTP；A4 |
| I5 | 刷新令牌的期限滑动、没有上限，也没有登记 | 3.4、3.5 绝对期限；4.5；4.6；6.5 |
| I6 | 零值也合法的必填字段无法校验 | 第二稿用 `writeOnly` 规则；第三稿改为在接口边界检查必填字段是否出现，`writeOnly` 规则删除（3.11） |
| I7 | 决策点 2 的选项 A 漏了最大的代价 | 第 10 节决策点 2；13.2 给 M3 的邀请交接 |
| I8 | Plane 的停用描述有误，M3 的部分规划不足 | 第 10 节决策点 3；6.4；13.2 |
| I9 | P1 合并了放行的中间件，安全网晚一个 Phase | 第 12 节 P1 的交付物和完成线 |
| I10 | P1 太大 | 第 12 节拆成 P1 `platform-core`、P2 `sessions`，后面顺延为 P3–P5 |
| I11 | 字段规则让 M3、M5 先删后加 | 3.2 规则 2；7.5；7.6；7.7；13.2 |
| I12 | 差异登记不全；"第 8 节"的引用错误 | 3.13；3.20；4.6 |
| I13 | 错误码不在契约里 | 3.11 `x-problem-codes`；3.12；7.3；9.3；9.4 |
| I14 | 访问令牌的过期拿本机时钟和服务端时刻比较 | 5.2 `access_token_expires_in`；7.1 退避 |
| M1 | 请求流程图的顺序不对 | 3.6；6.4 |
| M2 | 不用 scopes-on-context 的第二个理由不准 | 3.6 只留"遗留选项" |
| M3 | 时钟端口的位置；两个时间来源；端口放在 `app` | 3.3；3.5；3.13（第三稿：审计列也由时钟写入）；3.20 |
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
| M14 | 3.9 的比较位置；日志中的会话 id | 3.9；8.4（第二稿记 `session_ref`；第三稿会话 id 不再带来能力，直接记 `session_id`） |
| M15 | 新手引导两步是产品决定 | 3.19；决策点 4 |
| M16 | 给 M3 的交接说要"加端口"，其实已有 `SignupPolicy` | 13.2 |
| M17 | 与总体设计 3.1、3.2 的约定不符 | 3.12 撤销路径改为 `/api-tokens/{id}`；5.1 注册的返回值；3.20 |
| M18 | 模块入口的 `Commands()`；命令行没有 River；sqlc 没有模块边界 | 3.3 `Admin()`；3.15、3.17 只投递的客户端；3.14 按模块限定 `schema` |

- 评审的负责人问题 2（退出的语义）放在 11.1，负责人已批准；问题 3（平台能否导入 `shared`）由控制者裁定为不导入（3.3）。
- 核对评审引用时发现的出入：Plane 重置命令中 zxcvbn 的位置是 `reset_password.py:56`，不是评审写的 `:122`；Plane 停用时的"唯一管理员"检查确实写在代码里，但按代码推导它从不拒绝（决策点 3）。

### 17.2 第二稿的 Codex 评审与控制者复核（第三稿落实）
第二稿 `498735d` 经 Codex 对抗性评审（Critical 0、Important 12、Minor 9，另有四个决策点和偏离核对表），控制者又复核出 N1–N4、m1–m7。控制者对每一条作了裁定，负责人裁定了全部决策点。第三稿逐条落实；"Phase"一栏是实现并验证它的 Phase。Codex 报告第 8 节记录同样的处理结果。控制者复核第三稿后又提出 R1–R8 和几处细节，复核修订之后的核验又提出 F1–F4（都在本节末）；下面两张表已按最后的做法更新，其中 I-2 由选项①改为选项②。

**负责人的裁定**（2026-09-25）：

| 事项 | 裁定 | 落点 |
|---|---|---|
| 决策点 1 | B：`nerve users set-email`；说明写明它不撤销 PAT、不是恢复手段 | 3.5、3.17、8.7、第 10 节、A16 |
| 决策点 2 | C：prod 默认关闭，dev、test 默认开放；`nerve users create`；配置测试证明 prod 的默认值；启动提醒；更正"C 与 B 的安全差别" | 3.17、5.3、6.1、6.5、8.7、第 10 节、13.2（M3、M8）、A2、A17 |
| 决策点 3 | A：自助停用和 `deactivate`、`activate` 命令；PAT 不删除；按账户行锁；任何凭证都能停用 | 3.5、3.17、4.6、5.1、7.7、8.6、第 10 节、13.2（M3）、A12 |
| 决策点 4 | A：删除两步和 `is_self_managed`；`profiles` 11 列 | 3.19、4.3、5.2、5.3、7.6、第 10 节 |
| 11.1 | 批准：退出只结束当前会话 | 11.1、3.20（P2、P3） |

**Codex 的发现**：

| 编号 | 问题 | 落点 | Phase |
|---|---|---|---|
| I-1 | 管理员重置赶不走并发签发的凭证 | 3.5 的账户行锁协议（`FOR NO KEY UPDATE`，spike；哈希只因重新哈希而变时重新校验一次）和加锁顺序（`users` → `profiles` → `auth_sessions` → `api_tokens`，外键检查算一次 `users` 的锁）；3.15 `SKIP LOCKED`；6.3；6.4；9.2 的六个交错测试；P3 的完成线；§16 | P2（登录）、P3（其余和交错测试） |
| I-2 | 只凭旧代数就撤销，没有证明旧令牌是真的 | 第三稿先取选项①（`auth_refresh_tokens` 存每一代），复核（R3）指出它的行数只受按 IP 的限流约束，改为选项②：3.4 刷新令牌带 16 字节的 HMAC 标签，MAC 密钥由 HKDF 从签名密钥派生；3.5 的判定表（当前代比对 `token_hash`，旧代要标签成立才撤销）、为什么改、换钥的后果；3.7；4.1、4.5 每个会话一行；6.2、6.3 `RefreshTokenMAC`；8.4 日志直接记 `session_id`，`session_ref` 删除；8.6 正则；9.1；A1、A4、A5；§16 | P1（令牌格式和表）、P2（逻辑和测试） |
| I-3 | 无效令牌的限流发生在认证之后 | 3.6 认证之前的失败闸门：先预留，只有认证失败才留下；签名有效而只是过期的 JWT 不计数；计数的确切范围；3.10 `auth_failure`、自己的令牌桶（`AllowAll`、`Reserve`）、客户端 IP 键；6.3；6.4；8.6；9.1 的并发测试；§16 | P2 |
| I-4 | 刷新令牌泄露可派生长期 PAT | 8.5 的风险链、恢复步骤、不采用的产品选项；8.6；8.7；7.7 | P3 |
| I-5 | 另一个标签页换了账户，旧标签页身份错乱 | 7.1 `nerve.auth` 和 `login_id`、四种 `storage` 事件、每次写入都在同一把锁下、续期写回之前核对 `login_id`；A6 的两种走法；9.4；9.6；P4 | P4 |
| I-6 | 服务端接受契约禁止的请求 | 0.1；3.11 结构在边界、取值在领域，两种做法的 spike，格式检查把原始值 `json.Unmarshal` 进映射的类型（`bodyshape` 只用标准库），`tools/bodyshapegen`，第四个整程序测试，删除 `writeOnly` 规则和"已知的不一致"；3.12 `email` 映射；5.1、5.2；6.1；6.4；A3、A8、A10；P1、P3 | P1（P3 起有格式字段） |
| I-7 | sqlc 的跨模块 `ALTER` 没有归属 | 3.14 迁移归被改表的模块，spike，`TestSQLCSchemaScope`；4.2；13.2（M3、M5）；3.20（总体设计 5.6） | P1 |
| I-8 | `onboarding_step` 的合并会丢更新 | 3.14 `||` 合并和 spike；4.3；6.2；9.2；P3 | P3 |
| I-9 | 共享游标固定为 `(created_at, id)` | 3.3；3.12 封套与载荷；13.2（M4、M7） | P3 |
| I-10 | 业务时钟与 `updated_at` 被切开 | 3.13 审计列由时钟写入；4.2–4.5；9.2；13.2（M4） | P1 |
| I-11 | 新手引导挂载时请求 M3 旧接口 | 3.1；7.4；7.5；7.6；A10；9.6；13.1；13.2（M3） | P4 |
| I-12 | A4 按字面必定失败 | A4 | P2 |
| M-1 | `limit` 的取值错误没有契约落点 | 3.11；5.1；9.3 | P3 |
| M-2 | `RetryAfter` 字段与方法同名 | 3.11 `RetryDelay`，草图已编译 | P1 |
| M-3 | 邮箱 CHECK 的说明不成立；JSON 约束偏弱 | 4.2、4.3 的 CHECK 和它保证的内容；3.13 JSON 列的 CHECK；9.2 反例 | P1 |
| M-4 | 密钥文件路径进了日志 | 3.7；6.5；13.1 | P1 |
| M-5 | 续期非 401 失败时是否退出，两处说法相反 | 7.1；9.4 | P4 |
| M-6 | A7 的页面版和 PAT 版不能用同一个会话预期 | 第 2 节的约定；A7；9.5 | P3（接口）、P5（页面） |
| M-7 | 前缀不会让托管平台自动识别 | 3.4；8.6 的正则（刷新令牌的 base64url 部分 91 个字符，整个令牌 98 个字符）；8.7 | P3 |
| M-8 | 差异登记的时点；"照搬"的用词 | 3.13；3.20 改为按 Phase；第 4 节的说明和各表；4.5 替换模型；4.6 的 Phase 一栏；4.7 删除关系图；第 12 节的合并条件 | P1、P3 |
| M-9 | 密码名单的数量口径 | 3.8 | P1 |
| §4 | 四个决策点和 11.1 | 第 10 节；11.1 | — |
| §5 | `ProblemStatus()` 是一处取舍 | 3.11 | P1 |
| §5 | M0-P5 的"同一层"要正式关闭 | 8.3；12 P4；13.1 | P4 |
| §5 | M0-P3 的参数绑定出口 | 3.11；12 P1、P3；13.1 | P3 |
| §5 | 物理删除与跨模块外键的关系图；M8 的 `NERVE_ENV` | 3.13；4.7；13.2（M3、M4、M6、M8） | — |

**控制者复核的发现**：

| 编号 | 问题 | 落点 | Phase |
|---|---|---|---|
| N1 | 主干一个字符就能绕过 | 3.8 的主干、过滤条件和重新测量；9.1；A2 | P1 |
| N2 | 期限的先后顺序 | 3.5 服务端期限和放在适配器的理由；6.5；7.1 两条路径上的超时和剩下的情况；9.1；9.4 | P2（服务端）、P4（前端） |
| N3 | 匿名桶 | 3.10 突发、提高 `anonymous`（续期、退出只经过它；不按未经验证的会话 id 另设桶，限流键不取未经验证的输入）、`X-Forwarded-For` 的提醒；6.5；7.1、7.4"会话暂不可用"；9.4；9.6；§16。最后一条（401 优先于 429）由 I-3 取代 | P2、P4 |
| N4 | 同 I-7 | 3.14 | P1 |
| m1 | `Router` 只记 `HandleFunc`；测试 3 没有填查询参数 | 3.6 | P1 |
| m2 | 只有"未认证"的错误计数 | 3.6；9.1 | P2 |
| m3 | 同 I-8 | 3.14 | P3 |
| m4 | 草稿发布的结构检查 | 由 I-6 解决；13.2（M4） | — |
| m5 | 忘了设 `NERVE_ENV=prod` | 6.1 启动提醒；8.4；第 10 节决策点 2；13.2（M8） | P1 |
| m6 | 依次 `Allow()` 会白扣前面的桶；注册先查邮箱 | 3.9；3.10 `AllowAll` 和 spike；6.3 | P1（注册的顺序）、P2 |
| m7 | 模块级中间件会挡住 M5 的上传 | 3.5；13.2（M5） | — |

**控制者对第三稿的复核**（`5ea72f9`，Ready with fixes）：

| 编号 | 问题 | 落点 | Phase |
|---|---|---|---|
| R1 | 边界上的格式检查要与生成的解码器一致 | 3.11 把原始值 `json.Unmarshal` 进映射的类型（核验 F1 之后的做法；这一稿原先直接调用类型自己的解析），格式清单来自 `type-mapping`，生成器移到工具模块，第一个使用者在 P3，第四个整程序测试的第 6、7 项；6.1；9.1；9.3；12 P1、P3 | P1（检查器）、P3（使用者和整程序测试） |
| R2 | `FOR UPDATE` 挡住外键插入；加锁顺序漏了 `profiles` | 3.5 `FOR NO KEY UPDATE` 和 spike，`set-email` 改唯一列时自己升级，`profiles` 进入顺序，外键检查算一次 `users` 的锁；6.3；9.2 第 5、6 项；§16 | P2、P3 |
| R3 | 每一代一行的存储没有上界 | 由 I-2 改为选项②解决：每个会话一行（3.5、4.5）；§16 | P1、P2 |
| R4 | `login_id` 的写入竞争 | 7.1 每次写 `nerve.auth` 都在同一把锁下、写回之前核对 `login_id`、"记录出现"一行；A6 的两种走法；9.4；9.6；12 P4 | P4 |
| R5 | 先查后扣的失败闸门在并发下超额 | 3.6 先预留、失败留下、其余退回，spike 和预留的代价；3.10 `Reserve`；9.1 的并发测试；§16 | P2 |
| R6 | IPv6 一台主机拿到整个 /64 | 3.10 客户端 IP 键；6.5 `ratelimit.ipv6_prefix_len`；9.1；§16 | P2 |
| R7 | 请求期限会在 `COMMIT` 时取消已做完的事务；剩下的情况漏了请求这一段 | 3.5 剩下的情况（核验 F4 之后是四种）；3.6 `COMMIT`、`ROLLBACK` 在 `context.WithoutCancel` 下执行、有自己的期限（全局规则）；6.1；6.5 `database.commit_timeout` 和启动时的不等式；7.1；9.2；3.20（总体设计 6.4） | P1（`TxManager`）、P2（续期） |
| R8 | 迁移角色的 README 说明放错了 Phase | 8.7 移到 P1；13.1 M0-P2 第 7 条在 P1 关闭 | P1 |
| 细节 | 判定表中重复使用那一行写明"未撤销、未过期" | 3.5 | P2 |
| 细节 | 哈希只因并发的重新哈希而变 | 3.5 在事务外重新校验一次、重做一次锁内那一步；9.1；9.2 第 5 项 | P2、P3 |
| 细节 | 登录与 `set-email` 并发 | 3.5 说明它无害的原因 | — |
| 细节 | M3 交接的引用 | 第 1 节；13.2 改为 `web/apps/web/app/(all)/onboarding/page.tsx:33-37` | — |
| 细节 | 游标的封套随第一个列表加入 | 3.20；12 P1、P3 | P3 |
| 细节 | 测试用的固定时钟截到微秒 | 3.13；6.3 | P1 |
| 细节 | 换签名密钥的风险 | 3.5；3.7；8.7；§16 | — |
| 细节 | 失败闸门计数的是什么 | 3.6 计数和不计数的清单；3.10；9.1 | P2 |

**控制者对复核修订的核验**（`8cebc20`，Ready with fixes；I-2 的选项②、R1–R8 和偏离 (a)–(c) 经核验成立）：

| 编号 | 问题 | 落点 | Phase |
|---|---|---|---|
| F1 | `oapi-codegen/runtime` 把 `github.com/google/uuid` 带进 nerve，M0 的传递依赖测试禁止它；`bodyshape` 依赖 `runtime/types` 也会带进它 | 3.11 格式检查改为 `json.Unmarshal(raw, new(T))`，`bodyshape` 只用标准库，删去 `date` 的检查器，生成器遇到 `date`、`x-go-type` 失败，spike；3.12 守卫改为直接检查（google/uuid 只允许由 `runtime` 模块的包导入；生成文件不得引用 `openapi_types.UUID`；depguard 照旧）；6.1；6.6；3.20（M0 设计 3.7，M0/P3 spec 2.8、第 7 节）；13.1；9.1；9.2；12 P1、P3 | P1（`bodyshape`）、P3（守卫） |
| F2 | 工具模块不在持续集成里 | 3.11；12 P1（`make test`、`make lint-go` 覆盖 `server/tools`）；9.1 | P1 |
| F3 | 遗留的旧说法 | A3 一行会话；6.4 删去"第 0 代刷新令牌"，停用按全局的加锁顺序列出；3.5 `profiles` 位置的理由；4.7 的缩进 | P2、P3 |
| F4 | 提交的措辞 | 3.6 提交无回应时结果未知、只管 `TxManager` 的事务、单条写入为什么无害、`ROLLBACK` 同样不被取消；3.5 第 4 种情况；6.1；6.4；6.5；7.1；9.2；12 P1 | P1、P2 |
| 细节 | 换钥暂时用空失败闸门；刷新令牌正被盗用时换钥 | §16 两行；3.7；8.7 | — |
| 细节 | 预留时在认证中的上限是桶里当时剩下的单位 | 3.6；§16 | P2 |
| 细节 | 闲置键的清理要有测试 | 9.1；12 P2 的完成线 | P2 |

**第三稿及其复核、核验时发现的出入**：
- 第二稿的"294"用的是"至少一个字母"的近似条件；精确的条件（至少两个 ASCII 字母）是 288（3.8）。
- oapi-codegen 默认把 `format: email` 生成为 `openapi_types.Email`，它在解码时自己校验，会绕过分层；模板改为映射到 `string`（3.12）。
- `jsonb - text[]` 作用在标量上报 22023 而不是 CHECK 违例，JSON 列的 CHECK 要包在 `CASE` 里（3.13）。
- glibc 的 `[[:space:]]` 不含 U+00A0（4.2）。
- sqlc 要求引用同一张表的子查询起别名（3.14）。
- `golang.org/x/time/rate` 的 `Reservation.CancelAt` 在预留的时刻已过之后什么也不退回（spike）。复核要的"与 `AllowAll` 同一套预留和退回"建不到它上面：认证之后才知道要不要退。`platform/ratelimit` 改为自己的令牌桶，这个依赖删除（3.10、6.6）。
- 改唯一列的 `UPDATE` 在 Postgres 中算改键：即使事务先取的是 `FOR NO KEY UPDATE`，`set-email` 的那条语句也会把行锁升级，外键插入要等它提交（spike，3.5）。
- 导入 `github.com/google/uuid` 的不只是 `runtime/types`（`uuid.go:4`），`runtime` 包自己的 `styleparam.go:30` 也导入它（spike，`go list -deps`）。所以传递依赖测试的例外按模块写：导入者都属于 `github.com/oapi-codegen/runtime` 模块，而不是只认 `runtime/types` 一个包（3.12）。
- 标准库的 `uuid.UUID` 除了标准写法，还接受无连字符、带花括号和带 `urn:uuid:` 前缀的写法；边界的检查与解码器一致，照样接受（3.11）。
