# M2/P4 前端认证：设计说明（spec）

| 项 | 内容 |
|---|---|
| Phase | M2/P4 `web-auth` |
| 日期 | 2026-09-27 |
| 状态 | 进行中 |
| 上级文档 | [M2 设计文档](../M2-design.md) 第 2（A1–A6、A10、A15、S2）、3.1、3.2、3.18、3.19、3.20（P4 各行）、7.1–7.6、7.9、8.3、8.6、8.7（P4）、9.4–9.6、12（P4）、13.1、§16 节；[Codex 设计评审](../reviews/M2-design-codex-adversarial-review.md) I-5、I-11、M-5、R4；[v0 总体设计](../../v0-design.md) 4.3 节 |
| 前置交接 | [M0-P5-frontend-api-notes](../handoffs/M0-P5-frontend-api-notes.md)、[M0-P6-e2e-notes](../handoffs/M0-P6-e2e-notes.md) 的页面登录状态和 S2、[M1-P2-trim-content](../handoffs/M1-P2-trim-content.md)、[M1-P3-trim-platform](../handoffs/M1-P3-trim-platform.md) 的 CSRF、认证错误和实例字段、[M1-P4-router-native](../handoffs/M1-P4-router-native.md)；[P2 评审记录](../reviews/P2-sessions-review.md) 第 7 节的"真实反向代理后面的客户端 IP"（本 Phase 的处理见第 7 节） |
| 计划 | [P4 plan](../plans/P4-web-auth.md) |

本 spec 只写 M2 设计交给 P4 决定的东西：名字、签名、规则的细节、测试，以及原型证明了什么。规则本身以 M2 设计为准，这里引用节号，不重述。P4 依赖 P1–P3b 的接口（M2 设计 12 节）；M2 前端中属于 P5 的部分（个人设置的四个标签页、PAT、`@nerve/services`、剩下的清理）不拉进来（第 5 节）。

## 1. 目标

按 M2 设计 12 节 P4 的目标：浏览器通过令牌管理器注册、登录、续期、退出；Cookie 会话和 CSRF 从前端消失；用 HTTP 部署时多个标签页也能正常续期；另一个标签页登录别的账户时，本标签页不会以错误的身份写入；M2 能到达的页面挂载时不请求 M3 的旧接口。具体是：

- `@nerve/api-client` 用 `--root-types` 直接导出生成的类型，web 依赖它；
- 令牌管理器、跨标签页的锁（`navigator.locks`，没有它时用 localStorage 租约）、认证中间件，以及 9.4 的单元测试；
- web 的客户端：不带令牌的 `publicClient`；每一代 stores 一个挂着认证中间件的客户端，绑定这一代 stores 所属的会话；`ApiError` 与 `unwrap`；
- user、profile、instance 三个 store 改用生成的 `User`、`Profile`、`InstanceInfo`；会话变化时 stores 重建；删除 `IUser`、`TUserProfile`、`IUserTheme`、`TOnboardingSteps`、`IInstanceInfo`、`IInstanceConfig` 和 Plane 的认证类型；
- `AuthenticationWrapper` 重写；"会话暂不可用"；`next_path` 带查询和片段、拒绝控制字符；web 的 axios 基类不带 Cookie、不拦截 401；
- 登录页、注册页调用接口，错误就地显示；错误文案表由测试对着 `openapi.yaml` 核对；安全页修改密码的错误；密码 8–128 个字符；
- 新手引导只剩三步，挂载时不预取，资料步骤没有头像上传；
- `webui` 给页面加 CSP；
- 关键词规则 5 条；
- 端到端：`signedInPage`、`watchPage`；A1–A6、A10、A15 的页面版本（A4 两遍，A6 含两种切换账户）和 S2 的新断言；
- 3.20 中 P4 的各行、8.7 中 P4 的 README 内容、交接的处理记录。

## 2. 交付物

### 2.1 文件总览

路径相对于仓库根目录，`web/apps/web/` 简写为 `web:`。"生成"表示由命令生成并提交，不手改。"Task"是 plan 中负责它的任务；几个 Task 号表示先写过渡版本、最后一个 Task 写成最终版本。共改动或新增 102 个文件，删除 9 个；plan 的文件结构表逐个列出。

| 路径 | 内容 | Task |
|---|---|---|
| `web:core/lib/auth/refresh-lock.ts`、`refresh-lock.test.ts`、`fake-browser.ts` | 跨标签页的锁；测试替身 | 1、2 |
| `web/packages/api-client/package.json`、`src/index.ts`；`src/schema.gen.ts`（生成） | `--root-types`；`ApiClient`、`Middleware` | 2、4 |
| `web/apps/web/package.json`、`web/packages/constants/package.json`、`pnpm-lock.yaml` | 两个工作区内的依赖；web 的 oxlint 上限 | 2、5、7、8、10 |
| `web:core/lib/auth/token-manager.ts`、`token-manager.test.ts`、`token-manager.tabs.test.ts`、`fake-nerve.ts`、`fake-time.ts` | 令牌管理器 | 2、3 |
| `web:core/lib/auth/auth-middleware.ts`、`auth-middleware.test.ts`、`auth-middleware.session.test.ts` | 认证中间件；客户端绑定会话 | 4、10b |
| `web:app/(all)/onboarding/page.tsx`、`web:core/components/onboarding/`（`steps/role/`、`steps/usecase/` 删除）、`web/packages/types/src/workspace.ts` | 三步的新手引导 | 5、7、8 |
| `web:core/lib/api-error.ts`、`api-error.test.ts`；`web:core/lib/auth/api-client.ts` | `ApiError`、`unwrap`；`publicClient`、`apiFor` 和令牌管理器的组合 | 6、7、10b |
| `web:core/services/{instance,auth,user,api}.service.ts`；`web:core/store/{instance.store,root.store,root.store.test}.ts`、`web:core/store/user/{index,profile.store,settings.store,permissions.store}.ts`、`web:core/store/issue/{root.store,profile/issue.store}.ts`；`web:core/lib/store-context.tsx`、`store-context.test.ts` | services 和 stores；每一代 stores 绑定自己的会话 | 6–10、10b |
| `web/packages/types/src/`（`instance/`、`auth.ts` 删除；`users.ts`、`index.ts`、`project/projects.ts`、`search.ts` 修改）；`web/packages/constants/src/themes.ts` | Plane 的手写类型；`Theme` | 6、7、8、10 |
| 读实例配置、账户、资料的组件（plan 的文件结构表逐个列出）；`web:core/components/core/modals/user-image-upload-modal.tsx`（删除） | 生成的类型；没有上传 | 6、7、8 |
| `web:core/lib/wrappers/{authentication,store}-wrapper.tsx`、`web:core/lib/auth/use-session.ts`、`web:core/components/account/session-unavailable.tsx` | 认证包装；会话暂不可用 | 8、9 |
| `web/packages/utils/src/{url,auth}.ts`、`next-path.test.ts`、`auth.test.ts`；`web/packages/ui/src/form-fields/password/helper.tsx` | `next_path`；密码规则 | 9、10 |
| `web:core/components/account/auth-forms/{auth-root,auth-header,password}.tsx`、`web:helpers/authentication.helper.ts`、`authentication.helper.test.ts`（`.tsx` 删除）、`web:vitest.config.ts`、`security.tsx`、`web/packages/i18n/src/locales/{en,zh-CN}/auth.json` | 登录页、注册页、错误文案表、修改密码 | 9、10 |
| `tools/keywords.json` | 5 条规则 | 6、8、10 |
| `server/internal/platform/webui/csp.go`、`csp_test.go`、`handler.go`；`server/internal/bootstrap/headers_test.go` | 页面的 CSP | 11 |
| `e2e/tsconfig.json`、`e2e/fixtures/{test,auth,auth-pages,browser}.ts`、`e2e/fixtures/assert/identity.ts`、`e2e/stories/identity/a{1,2,3,4,5,6,10,15}-*.spec.ts`、`e2e/stories/smoke/s2-web-app.spec.ts` | fixture；页面版本；S2 | 12、13 |
| `docs/v0/v0-design.md`、`docs/v0/frontend-changes.md`、`README.md`、五份交接 | 文档同步、交接 | 14 |

### 2.2 依赖

- 不加任何 npm 注册表上的包，Go 依赖不变。
- 两个工作区内的依赖，都是 `workspace:*`：`web` 依赖 `@nerve/api-client`（Task 2：令牌管理器和 stores 用生成的类型和客户端）；`@nerve/constants` 依赖 `@nerve/api-client`（Task 8：`THEME_OPTIONS` 的 `value` 用生成的 `Theme`）。`pnpm install` 之后 `pnpm-lock.yaml` 各多一处 `link:`，plan 给出差异；复现时从前一个 lock 出发执行 `pnpm install`，得到的 lock 与 plan 逐字节相同。
- `openapi-fetch` 0.17.0、`openapi-typescript` 7.13.0 已在 `@nerve/api-client` 中锁定，只改生成命令的参数。
- 重新生成的 `web/packages/api-client/src/schema.gen.ts`：911 行，SHA-256 `6db7b4a9fa9c2aa05123db1b947a170a27de844a2c2ae7f6a8f6af60ad9d4ee3`；`api/dist/openapi.yaml` 不变。

### 2.3 `@nerve/api-client` 接入 web（M2 设计 3.12）

- 生成命令加 `--root-types --root-types-no-schema-prefix`：`components["schemas"]` 中的每个模式另有同名的顶层类型（`User`、`Profile`、`InstanceInfo`、`AuthTokens`、`Problem`、`FieldError`、`Theme`、`UserUpdate`、`ProfileUpdate`……）。`index.ts` 用 `export type * from "./schema.gen"` 原样导出，不写别名、不做转换。
- `type ApiClient = ReturnType<typeof createClient>`（令牌管理器的依赖）；Task 4 另导出 openapi-fetch 的 `Middleware` 类型。

### 2.4 跨标签页的锁（M2 设计 7.1）

```ts
interface RefreshLock { run<T>(task: () => Promise<T>): Promise<T> }
const LOCK_NAME = "nerve.auth.refresh";
function webLock(locks: Pick<LockManager, "request">): RefreshLock;
const LEASE_KEY = "nerve.auth.refresh_lease";
type LeaseDeps = {
  storage: Pick<Storage, "getItem" | "setItem" | "removeItem">;
  onStorage: (listener: (key: string | null) => void) => () => void;
  now: () => number;
  tabId: string;
};
function leaseLock(deps: LeaseDeps): RefreshLock;
```

- `webLock`：`locks.request(LOCK_NAME, task)`。
- `leaseLock`：租约 `{owner, expires}`，10 秒；没有、已过期或是自己的就写入，100 毫秒后读回，仍是自己的才算拿到；拿不到时等租约键（或 `null` 键，即 `localStorage.clear()`）的 `storage` 事件，或 200 毫秒后再看；用完只删除自己的；任务失败也释放，并把失败传出去。**同一个标签页的任务先在本标签页内排队**，再取租约（第 3 节第 3 条）。
- 标签页 id 和 `login_id` 都由 `crypto.getRandomValues` 生成：非安全上下文没有 `crypto.randomUUID`（附录 A 的 E4）。
- `api-client.ts` 中 `"locks" in navigator` 时用 `webLock(navigator.locks)`，否则用 `leaseLock`。
- 测试（`refresh-lock.test.ts`，9 个，假时钟和手动闸门，没有 sleep）：拿到空闲的租约要等读回、任务结束后删除；另一个标签页持有时等待，释放的 `storage` 事件唤醒它；没有 `storage` 事件时靠轮询发现租约已释放；接管过期的租约；只删除自己的租约；读回之前被另一个标签页盖掉时继续等；同一个标签页的任务一个接一个；任务失败时释放租约并传出失败；`webLock` 在 `nerve.auth.refresh` 下运行任务。

### 2.5 令牌管理器（M2 设计 7.1、9.4；Codex I-5、M-5、R4）

```ts
const AUTH_KEY = "nerve.auth";
type SessionState = Readonly<{
  status: "starting" | "signed-in" | "signed-out" | "unavailable";
  loginId?: string;
  retryAt?: number;
}>;
class SessionUnavailableError extends Error { readonly retryAt: number }
class SessionChangedError extends Error {}
type TokenManagerDeps = {
  storage: Pick<Storage, "getItem" | "setItem" | "removeItem">;
  lock: RefreshLock;
  client: ApiClient; // without the auth middleware
  now: () => number;
  randomHex: (bytes: number) => string;
};
class TokenManager {
  get state(): SessionState;
  subscribe(listener: () => void): () => void;
  start(): Promise<void>;
  retry(): Promise<void>;
  accessToken(): Promise<string | undefined>;
  renew(sent: string): Promise<string | undefined>;
  signIn(tokens: AuthTokens): Promise<void>;
  signOut(): Promise<void>;
  endSession(loginId: string | undefined): Promise<boolean>;
  handleStorageChange(): void;
}
```

- **记录**：`nerve.auth = {refresh_token, login_id}`，一次 `setItem` 写入。`login_id` 是 16 个随机字节的十六进制，每次 `signIn`（登录、注册）新生成，续期不变。访问令牌只在内存里。
- **改变会话的操作只作用于它被调用时的会话，读此刻的记录**：续期和 `signOut` 记下调用时标签页的 `login_id`，`endSession(loginId)` 由调用方给出被拒请求所属的会话；它们在锁下读出记录，是这个会话的才续期、退出、删除，不是的就跟随记录，不碰别的会话。`storage` 事件也一样：事件只说明记录变了，标签页读此刻的记录，不用事件带来的值，因为事件可能晚于之后的写入才到（例如晚于本标签页自己的登录）。
- **启动**（`start`，只执行一次）：没有记录 → `signed-out`，不发任何请求（S2 断言未登录时只请求 `GET /api/v0/instance`）；有记录 → `starting`，第一次续期决定状态：200 `signed-in`；401 `signed-out`，删除记录；429、5xx、400、网络错误、超时 → `unavailable`，保留记录，按 `Retry-After` 或 1、2、4……最多 30 秒退避，到时自动重试；`retry()` 立即重试；等待期间另一个标签页改了会话时不再重试。
- **访问令牌**：剩下不到 30 秒时先续期；剩余时间按本机收到响应的时刻加 `access_token_expires_in` 算，本机时钟的偏差不影响。
- **续期**：
  - 同一个标签页、同一个会话（按 `login_id`）一次只有一个续期，同时的请求共用它；会话换了之后发出的请求等新会话自己的续期（第 3 节第 2 条）；
  - 按上面的规则，本会话指续期开始时记下的 `login_id`，不是标签页此刻的会话：锁下的记录不是本会话的，续期就跟随它并以 `SessionChangedError` 结束，发给旧会话的请求到此为止，不会带着新会话的令牌发出（续期等锁期间标签页可能已经跟随了另一个标签页的登录）；
  - 续期请求 8 秒超时（`AbortController`），超时按网络错误处理；
  - 拿到响应后**再读一次记录**，`login_id` 变了就丢弃结果、不写回（不论 nerve 答 200 还是 401），跟随新记录（R4：没有 `navigator.locks` 时租约不是原子的，续期途中记录可能被另一个标签页的登录换掉）；
  - 200 → 写回新的刷新令牌，`login_id` 不变；**只有 401 结束会话**；429、5xx、400、网络错误、超时 → `SessionUnavailableError(retryAt)`，会话保留，退避期间不再请求。
- **`renew(sent)`**（中间件在 401 之后调用）：已退出时给 `undefined`（第 3 节第 4 条）；另一个请求已经换到新令牌时直接给它；否则丢掉被拒的令牌，续期。
- **写入都在锁下**：`signIn` 写新记录；`signOut` 在锁下读出最新的刷新令牌，用不挂中间件的客户端调 `POST /api/v0/auth/logout`（8 秒，尽力而为，失败也照样退出），再删除记录，记录已是别的会话时跟随它、谁也不登出（另一个标签页的登录已换掉本会话的刷新令牌，没有可登出的）；`endSession(loginId)`（重发仍 401 时，`loginId` 是请求所属的会话）在锁下删除记录，给出 `true`；记录已不是这个会话的（别的会话的，或已被删除）时跟随它、不删，给出 `false`。
- **`storage` 事件**（`handleStorageChange()`，读此刻的记录）：记录不在了且本标签页未退出 → 退出；`login_id` 变了（包括本标签页未登录时记录出现）→ 丢掉访问令牌，以新会话 `signed-in`，访问令牌由下一次续期取得；`login_id` 没变 → 不动。
- 订阅者（`useSession`、`store-context.tsx`）在每次状态变化后收到通知；`state` 每次是新对象。
- 测试：`token-manager.test.ts`（32 个，单标签页）、`token-manager.session-change.test.ts`（3 个 × 两种锁 = 6 个）、`token-manager.tabs.test.ts`（10 个 × 两种锁 = 20 个），各测试的内容见 plan 的 Task 2、3，此外：`token-manager.test.ts` 另有续期、退出时 fetch 被拒绝（断网）的两个测试，续期保留会话和记录、给出 `SessionUnavailableError`，退出照样在本地退出；`token-manager.session-change.test.ts` 核对上面的规则：另一个标签页以 Y 登录先拿到锁时，以 X 发出的请求的续期以 `SessionChangedError` 结束，拿不到 Y 的令牌；以 X 要求的退出谁也不登出，以 X 结束会话不删 Y 的记录，标签页都跟随 Y。`token-manager.tabs.test.ts` 的三个测试（登录等另一个标签页的续期、退出交出另一个标签页刚写的刷新令牌、另一个标签页只续期时不动）让事件晚于登录、退出、续期才送达，核对标签页读此刻的记录。写入 `nerve.auth` 的每一处（续期写回、续期 401 时删除、登录、退出、结束会话）都有测试核对写入时 `RecordingLock.held` 为真，以续期写入或删除为结果的测试核对全部写入。测试替身：`SharedStorage`（多个标签页共用一份数据，一个标签页的写入在微任务里通知其他标签页，事件只带键，和浏览器一样不通知自己；没有改变数据的写入（同样的值、删除不存在的键）不通知任何标签页；`hold()` 之后的事件留到 `deliver()` 才送达，像浏览器那样晚于之后的写入和锁的授予）、`RecordingLock`、`FakeNerve`（把每个请求停住，直到测试 `answer` 或 `fail`：`fail` 像断网时的浏览器那样让 fetch 以 `TypeError` 失败；`tokens()` 发出递增的 `at-n`、`rt-n`）、`gate()`、`settle`（假时钟上以 10 毫秒推进，20 秒为限）。

### 2.6 认证中间件（M2 设计 7.1；Codex M-5）

`authMiddleware(tokens: Pick<TokenManager, "state" | "accessToken" | "renew" | "endSession">, loginId: string | undefined): Middleware`。挂着它的客户端属于会话 `loginId`（`undefined` 是没有会话），即建这个客户端的那一代 stores 的会话（2.8）：

- `onRequest`：标签页此刻的 `login_id` 不是 `loginId` 时（另一个标签页登录或退出了，本标签页登录或退出了，或会话已经结束），请求以 `SessionChangedError` 失败，不发出；这个核对紧挨着取令牌，取到的是这个会话的令牌。有访问令牌就设 `Authorization: Bearer …`，并在 fetch 读走请求体之前用 `request.clone()` 留一份副本，存在 `WeakMap` 中，键是 openapi-fetch 交给 `onResponse` 的同一个请求对象；没有会话时请求原样发出，不留副本。
- `onResponse`：只处理带了令牌却得到 401 的请求。`renew(发出时的令牌)` 得到 `undefined`（续期 401 结束了会话，或会话已经结束）→ 把原来的 401 交给调用方；否则给副本换上新令牌，用 `options.fetch` 重发**一次**；重发仍是 401 → `endSession(loginId)`，结束了这个会话就把重发的 401 交给调用方。续期因 429、5xx、网络失败时请求以 `SessionUnavailableError` 失败，会话保留。403、404、500 不续期。没有会话时发出的请求，在它的 401 回来之前本标签页登录了，也不重发。请求属于客户端的会话 `loginId`（第 2.5 节的规则）；401 回来时标签页已不在这个会话，请求就以 `SessionChangedError` 失败，不续期、不重发。被会话变化打断的请求只有这一个信号：续期在锁下发现记录已是别的会话（2.5），或重发的 401 之后 `endSession` 发现记录已不是这个会话的（它跟随记录，给出 `false`），请求也都以 `SessionChangedError` 失败；调用方不会把别的会话的 401 当成标签页此刻的会话失败。
- M2 设计 7.1 要求原型先验证的一点：openapi-fetch 0.17.0 的中间件能这样重发。`auth-middleware.test.ts`（20 个，真实的令牌管理器、假 nerve）核对重发的 `PATCH` 方法、路径、请求体都与原来相同；"用已被读走的原请求重发""在设令牌之后才复制"的变异都让它失败（附录 A）。其中 5 个核对上面的会话规则：请求在外时另一个标签页以 Y 登录，X 的 401 回来时请求以 `SessionChangedError` 失败；标签页还没听到 Y 的登录时，续期在锁下发现它，请求同样以 `SessionChangedError` 失败；取令牌途中标签页跟随了 Y，请求仍属 X；重发途中跟随了 Y，重发的 401 不删 Y 的记录，请求以 `SessionChangedError` 失败；重发途中另一个标签页退出、本标签页还没听到时，`endSession` 在锁下发现记录不在了，请求同样以 `SessionChangedError` 失败，而不是交回重发的 401。另有 2 个核对 fetch 被拒绝（浏览器断网时的样子）：请求的 fetch 被拒绝时错误原样交给调用方，不续期、不重发；重发的 fetch 被拒绝时错误也交给调用方；两种情况都保留会话。这些测试的客户端属于 setUp 时标签页的会话。
- `auth-middleware.session.test.ts`（4 个，同样的真实令牌管理器和假 nerve）核对客户端绑定会话：标签页跟随了另一个标签页以别的账户登录、或跟随了另一个标签页退出之后，属于 X 的客户端的请求以 `SessionChangedError` 失败，假 nerve 什么也没收到（2 个）；在会话 X 开始时建的客户端，经过第一次续期、另一个标签页续期、本标签页自己续期，照常以当时的令牌发出；没有会话时建的客户端，本标签页登录之后请求同样失败、不发出。"去掉核对""核对比较的是标签页此刻的会话而不是客户端的""状态一变就拒绝"的变异都让它失败。

### 2.7 `ApiError`、`unwrap` 与客户端（M2 设计 5.3、7.1、7.2）

- `class ApiError extends Error { status: number; problem: Problem | undefined }`：消息取 `detail`，没有时取 `title`，都没有时 `HTTP <状态码>`。
- `unwrap<T>(result)`：2xx 给 `data`（204 是 `undefined`）；否则抛出 `ApiError`。只有带字符串 `code` 的 JSON 才算 problem：代理的 HTML 错误页、没有 `code` 的 JSON 都是没有 problem 的 `ApiError`，页面显示通用的文案（2.10）。被会话变化打断的请求（2.6）以 `SessionChangedError` 本身到达调用方，`unwrap` 和 `ApiError` 不包装它：它是客户端的调用（如 `GET(…)`）被拒绝，在 `unwrap` 运行之前。测试 `api-error.test.ts` 7 个，用真的 `createClient`，`fetch` 返回固定的响应；其中一个核对只有 `title` 的 problem 以 `title` 为消息，一个经真的认证中间件核对 `SessionChangedError` 不变成 `ApiError`。
- `api-client.ts`：
  - `publicClient = createClient()`：同源、相对地址、不带令牌，给实例、注册、登录和令牌管理器自己的续期、退出用；
  - `tokenManager`：`localStorage`、2.4 的锁、`publicClient`、`Date.now`、`getRandomValues`；
  - `storage` 事件中键为 `nerve.auth` 或 `null` 的交给 `handleStorageChange`；
  - 模块加载时 `void tokenManager.start()`，早于 stores 的创建；
  - `apiFor(loginId)`：新建一个客户端，挂上 `authMiddleware(tokenManager, loginId)`。其余所有调用都走这样的客户端，每一代 stores 一个（2.8）；没有模块级的带令牌客户端。

### 2.8 stores 与类型（M2 设计 7.1、7.5；M1-P2 交接）

- **会话变化时重建 stores，每一代 stores 属于一个会话**（第 3 节第 16 条）：应用加载时 `store-context.tsx` 为令牌管理器此刻的会话建 `new RootStore(apiFor(loginId))`（`start()` 在模块加载时已同步定下有没有记录、是哪个 `login_id`）；它订阅令牌管理器，`loginId` 一变（变成另一个值、变成没有值、从没有值变成一个值）就调用 `store.resetOnSignOut(apiFor(新的 loginId))`；同一会话内的状态变化（第一次续期、`unavailable` 和重试）不重建。本标签页退出、会话结束、跟随另一个标签页换成别的账户，页面上都不留旧账户的数据；本标签页登录、跟随另一个标签页登录，也换一代 stores。`RootStore` 把客户端交给建 `UserService` 的 stores：`UserStore` 自己和它建的 `ProfileStore`、`UserSettingsStore`、`UserPermissionStore`，`IssueRootStore` 建的 `ProfileIssues`，都在构造时 `new UserService(api)`。旧一代 stores 的引用（组件闭包里的动作、`await` 之后接着的写入）用的是旧会话的客户端：标签页已换了会话，请求就以 `SessionChangedError` 失败，不发出。`RootStore.resetOnSignOut()` 不再重建 `router` 和 `instance`（第 3 节第 1 条）。测试：`store-context.test.ts`（1 个，假的令牌管理器和 `RootStore`）核对加载时为 X 建、同一会话内的通知不重建、X→Y、→没有、没有→Z、Z→W 各重建一次，每次用新会话的客户端；`root.store.test.ts`（3 个，真的 stores、令牌管理器和中间件，假 nerve）核对 `resetOnSignOut` 保留 `instance`、`router` 和 `theme`，为 X 建的 stores 以 X 的令牌发出、重建给 Y 之后以 Y 的令牌发出，以及资料步骤的竞争：X 的保存在标签页跟随 Y 之后以 200 回来，旧 stores 接着的步骤写入以 `SessionChangedError` 失败，假 nerve 只收到那次保存。
- **`InstanceStore`**：`config: InstanceInfo | undefined`；`InstanceService.getInstanceInfo()` 经 `publicClient` 调 `GET /api/v0/instance`。`InstanceWrapper` 不改，请求失败时仍是维护页（附录 A 的 C7）。读实例配置的五处改读新字段：页头的注册链接读 `signup_enabled`；`/create-workspace`、新手引导的创建工作区、工作区菜单、命令面板读 `workspace_creation_enabled`。
- **`UserStore`**：`data: User | undefined`；`fetchCurrentUser()` 先等 `tokenManager.start()`，只在 `signed-in` 时并行取 `GET /api/v0/me` 和 `GET /api/v0/me/profile`（7.1：没有记录时不请求 `/me`；设置和工作区不再在登录时取，3.1）；`signIn(LoginRequest)`、`signUp(RegisterRequest)` 把得到的令牌交给 `tokenManager.signIn`，失败时抛出 `ApiError`，什么都不留；`signOut()` 就是 `tokenManager.signOut()`；`deactivateAccount()` 调 `POST /api/v0/me/deactivate` 后 `tokenManager.endSession()`；`updateCurrentUser(UserUpdate)`、`changePassword(ChangePasswordRequest)`。死成员（`reset`、`isAuthenticated`、`error` 等）删除。
- **`ProfileStore`**：`data: Profile | undefined`；`fetchUserProfile()` 按资料的语言切换界面语言；`updateUserProfile(ProfileUpdate)` **失败时抛出**（Plane 的版本吞掉错误，调用方的提示从不出现）；`finishUserOnboarding()` 一次 `PATCH /api/v0/me/profile`（四个步骤、`is_onboarded`，有工作区时带 `last_workspace_id`）；`updateTourCompleted()`、`updateUserTheme(theme: Theme)` 用同一个接口。新手引导的步骤是 `OnboardingStepsUpdate` 的部分更新，由服务端合并。
- **services**：`AuthService.register`、`login` 经 `publicClient`；`UserService` 在构造时接收建它的那一代 stores 的客户端（`constructor(api: ApiClient)`），M2 方法（`currentUser`、`updateCurrentUser`、`changePassword`、`deactivate`、`getCurrentUserProfile`、`updateCurrentUserProfile`）经它和 `unwrap`；Plane 的 `/api/users/me/profile/`、`/onboard/`、`/tour-completed/`、`/api/instances/` 删除；属于其他领域的 4 个方法原样留下（7.5）；Plane 的模块级实例（`export default userService`）删除，它唯一的使用者 `UserPermissionStore` 用自己建的实例。
- **删除的类型**：`IUser`、`TUserProfile`、`IUserTheme`、`TOnboardingSteps`、`IInstanceInfo`、`IInstanceConfig`，以及 `@nerve/types` 的 `auth.ts`（`ICsrfTokenData` 等）。`IUser` 的使用方按 7.5 的表改为 `User` 或 `IUserLite`；`StoreWrapper` 按账户 id 判断换人，按 `profile.theme` 设主题；`@nerve/constants` 的 `I_THEME_OPTION.value: Theme`，写错或多出一个主题值都过不了类型检查。
- **general 页**：表单只有名、姓、显示名和只读的邮箱（`UserUpdate` 的字段）；写回 `role` 默认值的逻辑删除（3.19）；头像、封面只显示，上传控件和 `UserImageUploadModal` 删除（第 3 节第 12 条）。停用账户、切换账户的弹窗不再自己跳转，跳转交给 `AuthenticationWrapper`。
- `TTimezones`、`IApiToken`、`settings.store` 的死成员、PAT store、`@nerve/services`：P5（第 5 节）。

### 2.9 认证包装、会话暂不可用、`next_path`、axios 基类（M2 设计 3.18、7.1、7.2、7.4；M1-P4 交接）

- `useSession(): SessionState`：`useSyncExternalStore(tokenManager.subscribe, () => tokenManager.state)`。
- **`AuthenticationWrapper`**：唯一发出跳转的地方，只读会话、账户、资料和 `next_path`。

  | 会话 | 页面 | 结果 |
  |---|---|---|
  | `starting` | 任何 | 加载图标 |
  | `unavailable` | 任何 | `SessionUnavailable`，说明页面会自动重试（令牌管理器按 `retryAt` 排定了重试），按钮调 `tokenManager.retry()`；不跳转，地址和记录都不变 |
  | `signed-out` | 公开页、登录页、注册页 | 照常显示 |
  | `signed-out` | 其他 | `<Navigate to={signInPath(pathname + search + hash)} replace />` |
  | `signed-in` | 任何 | 账户按 `["CURRENT_USER", loginId]` 用 SWR 取（换会话就重取；焦点变化时不重取，不自动重试）；取失败 → `SessionUnavailable`，说明不承诺自动重试，按钮重取；被会话变化打断的取（`SessionChangedError`）不是失败，`fetchCurrentUser` 给出 `undefined`，等新会话的键和新的 store；账户或资料还没到 → 加载图标 |
  | `signed-in` | 登录页、注册页 | 合格的 `next_path`；没有时，完成了新手引导去 `/create-workspace`，否则去 `/onboarding` |
  | `signed-in` | 新手引导页 | 完成了新手引导就去合格的 `next_path` 或 `/create-workspace` |
  | `signed-in` | 其他 | 没完成新手引导就去 `/onboarding` |

  "完成"是 `is_onboarded`，或 `onboarding_step` 的四个键都为真。
- **`SessionUnavailable({ onRetry, autoRetry })`**：`role="alert"`，标题、说明、"Try again"；文案键 `auth.session_unavailable.{title,description,description_auto_retry,retry}`（"Cannot reach the server for now" / "暂时无法连接服务器"）。只有排定了重试（`autoRetry`：`unavailable` 且有 `retryAt`）时说明才用 `description_auto_retry`（"页面会自动重试"），否则用不作此承诺的 `description`。
- **`next_path`**（3.18）：`signInPath(path) = "/?next_path=" + encodeURIComponent(path)`，路径、查询、片段整体编码；`isValidNextPath` 另外拒绝任何位置的控制字符（`\u0000`–`\u001f`、`\u007f`：浏览器会从地址里丢掉制表符和换行，`/\t/evil.example` 会变成 `//evil.example`）。服务端没有跳转，M1-P4 交接"服务端校验 `next_path`"按此关闭。测试 `next-path.test.ts` 11 → 20 个。
- **web 的 axios 基类**：删掉 `withCredentials: true` 和 401 拦截，`window.location.replace` 随之消失，恒为真的 `currentPath` 判断不照搬；此后只给还没对接的 M3–M8 领域用，不带令牌、不跳转。

### 2.10 登录页、注册页与错误文案表（M2 设计 3.8、3.11、7.2、7.3；M1-P3、M1-P4 交接）

- `AuthRoot` 只读地址里的 `invitation_id`、`slug`（M3 的邀请标题），不再读 `error_code`、`email`。
- `AuthPasswordForm({ mode })`：受控表单（`noValidate`；`autoComplete` 为 `email`、`current-password`、`new-password`），提交时调 `signIn` 或 `signUp`；没有原生表单 POST，没有 CSRF 预取和隐藏字段。nerve 拒绝时：字段错误显示在字段下方（`fieldErrorKeys`）；没有字段错误时在表单上方显示一句（`errorMessageKey`，`role="alert"`）；不跳转，输入的内容都留着；再次编辑时清掉上一次的提示，包括"密码强度不够"的横幅（第 3 节第 6 条）；成功之后的跳转交给 `AuthenticationWrapper`。
- `helpers/authentication.helper.ts`（`.tsx` 删除，连同 Plane 的 `EAuthenticationErrorCodes`、`authErrorHandler`）：
  - `EPageTypes`、`EAuthModes`；
  - `PROBLEM_MESSAGES`：`openapi.yaml` 中全部 13 个问题码 → 文案键；
  - `FIELD_ERROR_MESSAGES: Record<FieldError["code"], string>`：`FieldError.code` 的全部 10 个取值 → 文案键，少一个过不了类型检查；
  - `errorMessageKey(error)`：`ApiError` 按码取；不认识的码和没有 problem 的回答是 `auth.errors.unknown`；`SessionUnavailableError` 是 `auth.errors.unreachable`；
  - `fieldErrorKeys(error)`：字段 → 文案键。
- 测试 `authentication.helper.test.ts`（9 个）从 `api/dist/openapi.yaml` 读出全部 `x-problem-codes` 和 `FieldError.code` 的取值，核对两张表的键与之相等、指向的文案在英文文件中都存在（`sync-check` 保证中文一致）。web 的 vitest 配置加 `resolve.tsconfigPaths`，测试能解析 `@/`。
- **安全页修改密码**：`identity.current_password_incorrect` 显示在当前密码下方；`new_password` 的字段错误显示在新密码下方；其他错误用提示框，内容按错误码取文案。`Error & { error_code?: string }` 的断言、`toString()` 和笼统的 `change_password.error.message` 键删除（M1-P4 交接"由 M2 决定"第 2 项）。
- **密码规则**：`MIN_PASSWORD_LENGTH = 8`、`MAX_PASSWORD_LENGTH = 128`，按 UTF-16 码元计，与服务端的 `domain.MaxPasswordLength` 相同；长度规则的标签是"8–128 characters"。测试 `@nerve/utils` 的 `auth.test.ts` 5 个。
- 完成线：`git grep -n -E "csrfmiddlewaretoken|X-CSRFTOKEN" -- web` 没有输出。原型中用 `grep -rniE` 连 `csrf`、`EAuthenticationErrorCodes`、`error_code` 一起查 `web/`，也没有输出。

### 2.11 新手引导（M2 设计 3.1、3.2、3.19；Codex I-11）

- 删除 `steps/role/`、`steps/usecase/`；`EOnboardingSteps` 只剩 `PROFILE_SETUP`、`WORKSPACE_CREATE_OR_JOIN`、`INVITE_MEMBERS`；页头的进度只有这三步，"返回"只从工作区一步回到资料一步；不再读 `is_self_managed`。
- `/onboarding` 挂载时不再取工作区列表和邀请（M3 的接口）；`OnboardingRoot` 的 `invitations` 默认为空（13.2：M3 加回）。
- 资料步骤：没有头像上传，已有头像时显示，没有时显示名字的首字母；表单只有名、姓；**名字保存成功后才进入下一步**（第 3 节第 5 条）；步骤按 `OnboardingStepsUpdate` 部分更新，失败时提示。
- A10 的页面版本核对第一次打开（2.14）；浏览器核对 C8（附录 A）。

### 2.12 `webui` 的 CSP（M2 设计 8.3；M0-P5 交接）

- `contentSecurityPolicy(index []byte) string`：`index.html` 中每个没有 `src` 的 `<script>`（标签不区分大小写，结束标签可以带空白）的内容算 SHA-256、base64 编码，去重后按出现的顺序放进 `script-src`。策略：

  ```text
  default-src 'self'; script-src 'self' 'sha256-…' …; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; font-src 'self' data:; connect-src 'self'; object-src 'none'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'
  ```

  `style-src` 允许内联：React 的 `style` 属性和 popper 的定位要它（第 6 节）。
- `webui.Handler` 构造时读 `index.html`、算出策略；只有 `.html` 的响应带 `Content-Security-Policy`，资源文件、找不到的资源、没有构建前端时的回答都不带。前端重新构建后不需要手工更新（构建出的 `index.html` 有 5 个内联脚本，附录 A 的 E3）。
- 安全响应头仍在 `httpserver` 的固定链上（M2/P2）。两层分开的理由见 8.3；M0-P5 交接"和 CSP 放在同一层"一项按此正式关闭，理由写进 review。
- 测试：`csp_test.go` 的 `TestPolicyAllowsEachInlineScriptByItsHash`（一个像 React Router 构建出的页面：内联脚本、模块脚本、带 `src` 的脚本、大写的 `<SCRIPT>`、重复的脚本；哈希在包外用 Python 的 `hashlib`、`base64` 算出，策略逐字相同）、`TestPolicyOfAPageWithoutInlineScriptsAllowsOnlyNervesScripts`、`TestOnlyPagesCarryThePolicy`；`headers_test.go` 的 `TestOnlyPagesHaveAContentSecurityPolicy`（经整个服务：页面带策略，资源文件、`/healthz`、`/api/v0/instance` 和 `/api/` 下的 404 都不带）。

### 2.13 关键词规则（M2 设计 7.9 中 P4 的部分）

规则 51 → 56 条，例外仍是 3 条；文件范围都是 `^web/.*\.(?:[cm]?[jt]sx?|json)$`：

| 规则 | 内容 | Task | 7.9 的条目 |
|---|---|---|---|
| `is-self-managed` | `is_self_managed\|isSelfManaged` | 6 | `is_self_managed` |
| `plane-user-urls` | `/api/users/me/(?:profile\|onboard\|tour-completed)\|/api/instances/` | 8 | 用户、实例地址中 P4 的部分 |
| `csrf` | `csrf`（不区分大小写） | 10 | `csrf`、`X-CSRFTOKEN`、`csrfmiddlewaretoken`、`get-csrf-token` |
| `plane-auth-urls` | `(?<![\w.@-])/auth/` | 10 | `/auth/sign-`、`/auth/change-password` |
| `auth-error-code` | `error_code\|EAuthenticationErrorCodes` | 10 | `error_code` |

- 每条规则在删除的同一个 Task 加入（M1 设计 7.4）。
- `plane-auth-urls` 比 7.9 宽，禁止根路径 `/auth/` 下的一切：Plane 的认证接口都在那里，Nerve 的都在 `/api/v0/auth`；前面的否定断言放过 `/api/v0/auth/`、`assets/auth/` 这样的路径。
- 不禁止整个 `/api/users/me/`：M3 的 `joinProject` 等旧调用还在用它。`/api/users/api-tokens`、`/api/timezones/`、`withCredentials:\s*true` 的规则留给 P5（7.9：偏好页、api-tokens 页和 `@nerve/services` 在 P5 改写之前还用它们）。

### 2.14 端到端（M2 设计 2、9.5；M0-P6 交接）

- **fixture**：
  - `e2e/tsconfig.json` 的 `lib` 加 `DOM`（`page.evaluate` 和 `addInitScript` 里的代码在浏览器中运行）；
  - `auth.ts`：`signInContext(context, baseURL, tokens)` 用 `context.addInitScript` 在页面的任何脚本之前把一条新记录（`login_id` 是 16 个随机字节）写进这个 nerve 源的 localStorage，只在第一次加载时写（标记 `nerve.auth.e2e-seeded`），之后的加载、刷新和其他标签页看到的是页面自己续期或删除之后的记录；`recordOf(page)`；`newRecord(tokens)`；`writeRecord(page, record)` 像令牌管理器一样在 `nerve.auth.refresh` 锁下一次写入，其他标签页收到 `storage` 事件；
  - `test.ts` 的 fixture `signedInPage(tokens, baseURL?)`：返回还没有加载任何东西的页面；
  - `auth-pages.ts`：`signInPath`（测试自己的一份，不依赖前端的实现）、`formAlert`、`submitSignIn`、`fillSignUp`、`submitSignUp`；
  - `browser.ts`：`watchPage(page)`，在页面第一次打开之前调用，记下接口请求、失败的接口请求、发往 `/api/v0` 之外的接口的请求（M3 的旧接口）、未处理的异常和拒绝、控制台的错误和警告、CSP 违规（`securitypolicyviolation` 事件经 `exposeBinding` 送回测试）；
  - `assert/identity.ts`：`generationOf(refreshToken)`。
- **故事的页面版本**：接口版本不变，页面版本与接口版本调用同一组断言函数。plan 的 Task 12、13 逐条写明，要点：
  - **A1**：注册后到资料步骤；没有 Cookie；记录只有 `refresh_token`、`login_id`；两个令牌的主体（刷新令牌去掉前缀、访问令牌的签名段）不出现在 sessionStorage、localStorage 的其他键、页面地址、任何请求的地址和任何控制台消息中（第 3 节第 10 条）；
  - **A2**：邮箱已存在（大写输入）409，提示在表单上方，地址和输入框不变；不合规的密码不发请求（故事结束时数注册请求：5 个，不合规的密码一个也没有）；`common_password` 在字段下方；关闭注册的 nerve 没有"Sign up"链接（开放的 nerve 有）、两个邮箱都是 403；两个 nerve 的页面都没有 CSP 违规；数据库没有新增；
  - **A3**（2 个）：未登录打开 `/settings/profile/general?tab=x#y` 到带 `next_path` 的登录页，登录后原样回来；`//evil.example`、`/\evil.example`、`javascript:alert(1)`、`/\t/evil.example` 登录后都落到 `/create-workspace`；错误密码与不存在的邮箱是同一句提示，没有新会话；
  - **A4**（2 个：有 `navigator.locks`；`addInitScript` 删掉 `Navigator.prototype.locks`，走租约）：访问令牌 3 秒的 nerve，同一个上下文的两个标签页都在资料步骤；第一个续期被扣住最多 1 秒；两个标签页同时点"Continue"，都等到这一步最后一个请求的 200、进入下一步；每次续期 200，发出的刷新令牌代数依次是 0、1、2……各一次；`expectRefreshed`（第 3 节第 8 条）；
  - **A5**：页面的刷新令牌先被"别人"在接口上用过一次；页面重新加载时续期被拒，到带 `next_path` 的登录页，记录删除，会话 `reuse_detected`；
  - **A6**（3 个）：在一个标签页通过账户菜单退出，两个标签页都回到登录页，记录删除，会话 `logout`，这个会话的访问令牌下一次请求就是 401（第 3 节第 7 条）；切换 ①：标签页乙用 `writeRecord` 换上 Y 的记录、不退出，标签页甲显示 Y，此后保存的名字写进 Y，X 的名字不变、会话未被撤销；切换 ②：标签页乙先退出（甲随之到登录页），再以 Y 登录，两个标签页都以 Y 回到 `/onboarding`；
  - **A10**（2 个）：新账户进入 `/onboarding` 的两条路：整页加载（`signedInPage`，页面挂载时还没有账户），和注册之后应用内的跳转（页面挂载时账户已经取到，文档仍是 `/sign-up` 的那一个）；每条路上都没有失败的接口请求、发往旧接口的请求、未处理的异常、控制台错误、CSP 违规；填名字后等到 `PATCH /api/v0/me/profile` 的 200，下一步出现；数据库中只有 `profile_complete` 为真，名字已保存；
  - **A15**：限流很低的 nerve：同一个邮箱 401、401、429，另一个邮箱 401，第三个邮箱 429（按 IP）；两句提示分别是"The email or the password is wrong."和"Too many attempts. Please try again later."；
  - **S2**（2 个，改写）：未登录时只请求 `GET /api/v0/instance`，没有失败的请求、CSP 违规、未处理的异常、控制台错误；首页带 `Content-Security-Policy`，"Go to workspace"按钮出现；深链接跳到带 `next_path` 的登录页。S2 不再等 `networkidle`，而是等 `GET /api/v0/instance` 的回答和登录页出现：页面不读回答的请求一直算在途中，`networkidle` 就等到测试超时，而不是在断言上失败（Task 12 实测）。
- 每个断言内容的检查之前，先断言元素或请求存在（缺陷类别"页面没有渲染被检查的东西"）。
- 测试数：22 → 29（Task 12；A10 的应用内进入由 Task 5 的评审加入）→ 35（Task 13）。新增的等待都有期限：`waitForResponse` 10 秒，A4 的扣住最多 1 秒。

### 2.15 文档、README 与交接（M2 设计 3.20、8.7）

plan 的 Task 14 逐行给出文字。要点：

- **总体设计 4.3**：记录带 `login_id`；所有写入在同一把锁或租约下，续期写回之前核对 `login_id`；没有 `navigator.locks` 时用租约；只有续期 401 或重发仍 401 才结束会话；"会话暂不可用"；CSP 只加在页面上，内联脚本按哈希放行。
- **前端改动清单**：3.2 的"用户和认证相关的 store"和"登录、注册、退出、修改密码的提交方式"（CSRF）两行改为已完成（M2/P4）；3.1 的 M2 一行改为进行中（第 3 节第 13 条）。
- **README**：HTTP 部署时多标签页靠租约，公网部署用 HTTPS（8.7）；另外写上页面的登录状态、M2 能看到的页面、CSP、e2e 的 `signedInPage` 和 `watchPage`；删掉 M0 时"页面报错是预期的"一条和相对地址里的 `/auth/…`。
- **交接**：五份追加"处理结果（M2/P4）"（第 7 节）。

## 3. 与设计的差异和补充（请控制者裁定）

以下都没有改变 M2 设计的架构和前后端契约。第 11、12 条动到设计分给 P5 的事项，第 13、14 条是设计文本的错误，请控制者裁定。第 16 条记下控制者加的 Task 10b。

1. **`resetOnSignOut` 不重建 `instance` 和 `router`**：7.1 写"结束会话……调用 `rootStore.resetOnSignOut()`"。Plane 的 `resetOnSignOut` 重建全部 store，包括 `instance` 和 `router`；它们不是账户的数据，重建之后在下一次整页加载之前没有代码会再填它们。原型中照原样重建时，A5 停在 `InstanceWrapper` 的加载图标上；"重建 `instance`"的变异让 A5 失败（附录 A）。重建由 `store-context.tsx` 订阅令牌管理器触发（`loginId` 变成别的值或没有值；第 16 条起也包括从没有值变成一个值），不由令牌管理器回调：令牌管理器不认识 stores。
2. **续期按会话去重**：9.4 写"同一标签页内只续期一次"。原型的第一稿按标签页去重：另一个标签页登录 Y 之后，本标签页新发出的请求加入了 X 那次还在途中的续期；X 的续期写回前发现 `login_id` 变了，以 `SessionChangedError` 结束，这个本该以 Y 发出的请求也随之失败（浏览器核对 C4b 发现）。改为按 `login_id` 去重：换了会话之后发出的请求等新会话自己的续期。测试 "gives a request made after another tab's sign-in a refresh of its own, not the old session's"；"续期在会话之间共用""旧续期结束时清掉新续期"两个变异都让它失败。
3. **租约先在本标签页内排队**：7.1 的"租约是自己的就写入"对同一标签页的第二个任务等于没有锁（两个任务的 owner 相同）。`leaseLock` 先让本标签页的任务排队，再取租约；去掉队列的变异让 "runs the tasks of one tab one after the other" 失败。
4. **`renew` 在已退出时给 `undefined`**：会话已经结束时（另一个请求的续期刚得到 401，或另一个标签页退出了），被拒的请求再调 `renew`，第一稿会以没有 `login_id` 的状态续期、在锁下读到空记录时抛出 `TypeError`（`tsc` 报 `Object is possibly 'undefined'` 时发现）。改为直接给 `undefined`，中间件把原来的 401 交给调用方。另外，中间件只对自己带了令牌的请求续期：未登录时发出的请求，即使 401 回来之前本标签页登录了，也不会以新会话的身份重发。两条各有测试和变异（附录 A）。
5. **资料步骤只在保存成功后前进**：Plane 的资料步骤不看保存结果就进入下一步（它的 store 吞掉错误，7.5）。store 改为抛出之后，资料步骤等名字保存成功才前进，失败时留在这一步并提示。切换账户的竞争中这也是对的：C4b 中 X 的保存因 `SessionChangedError` 被放弃，页面留在资料步骤，名字没有写进任何一个账户。
6. **再次编辑时清掉"密码强度不够"的横幅**：7.3 写"密码强度提示照旧"。Plane 的横幅在提交被拦下之后一直显示，改为与其他提示一样在再次编辑时清掉（C1）。
7. **A6 在新手引导页头的账户菜单退出**（"Wrong e-mail address?" → "Switch account"）：这是 M2 页面上唯一看得见的退出入口：`/create-workspace` 和个人设置页没有退出按钮（个人设置页只能经命令面板的退出命令），工作区侧边栏的两个菜单在工作区页面里（M3）。
8. **A4 的"一次一个"用刷新令牌的代数证明**：不扣住时两个标签页的续期很少真的重叠，没有锁也常常通过；而 Playwright 对被 `route` 扣住的请求，计时从放行时算起，不能拿来判断重叠（原型实测，E2）。A4 因此扣住第一个续期最多 1 秒（没有锁时另一个标签页的续期在这期间发出，带同一个刷新令牌），用 nerve 收到的刷新令牌代数证明一次一个；"`webLock` 不加锁""不取租约"的变异让对应的一遍失败。另外 `waitForLoadState("networkidle")` 在客户端导航之后立即返回（E2），原型中 A4、A10 和浏览器核对 C5 因此偶发地在最后一个请求完成之前断言；改为等这一步最后一个请求（`PATCH /api/v0/me/profile`）的回答。
9. **访问令牌的有效期不到 30 秒时，每个请求都先续期**：30 秒的余量（7.1）大于有效期时，内存中的令牌永远"快要过期"。只有测试配置会这样（A4 和 C3、C5 的 3 秒）；A4 靠它在两个标签页同时触发续期。
10. **A1 查令牌的编码形式**：9.5 的 A1 没有写"令牌不出现在别处"。按缺陷类别 F，A1 核对两个令牌的主体不出现在 sessionStorage、其他 localStorage 键、页面地址、请求地址和控制台消息中；四个把令牌放到这些地方的变异都让 A1 失败。
11. **M1-P2 在 P4 关闭**：12 节把 M1-P2 放在 P5 的"关闭"中，13.1 写"P4（`IUserTheme` 删除，改用生成的类型）"。这份交接只有主题这一项，P4 做完（`TUserProfile`、`IUserTheme` 删除，`THEME_OPTIONS` 的值是生成的 `Theme`），所以在 P4 标为 `done`。主题下拉框的定位是 M1-closeout 的事项，仍在 P5。
12. **general 页的头像、封面上传控件在 P4 删除**：12 节把它放在 P5 的交付物 1。P4 删除 `IUser` 时 general 页的表单改用生成的 `User` 和 `UserUpdate`，而 `UserUpdate` 没有 `avatar_url`、`cover_image_url`（M5 加入），上传控件已无处可写，留着只能先写一层转换或 `any`。所以 P4 一起删掉（`UserImageUploadModal` 删除，general 页不再用 `ImagePickerPopover` 和 `handleCoverImageChange`），显示保留。P5 的这一项因此已经完成。
13. **3.20 中前端改动清单的节号写反**：3.20 写"P4：前端改动清单 3.1，CSRF 一行已完成""P5：3.2，M2 一行已完成"，但 CSRF 一行在 3.2（"登录、注册、退出、修改密码的提交方式"），M2 一行在 3.1。P4 按内容处理：3.2 的 CSRF 一行和"用户和认证相关的 store"一行（P4 做完）改为已完成，3.1 的 M2 一行改为进行中；P5 应把 3.1 的 M2 一行改为已完成。设计文本请随之更正。
14. **9.6 的"会话暂不可用"写法与实际不符**：9.6 写"已登录时停掉 nerve、刷新页面，显示等待和重试"。nerve 同时提供页面：生产构建下停掉 nerve 再刷新，页面本身加载不了，浏览器显示自己的错误页；用 Vite 开发服务器时页面能加载，但先失败的是实例请求，显示的是维护页（C6a 实测：维护页，地址和记录都在，nerve 恢复后 4.9 秒自动回到原页面）。"会话暂不可用"出现在 nerve 回答、但续期失败时：数据库停掉（续期 500，C6b）或续期得到 503、429（C6c）。建议 9.6 改为"停掉 nerve 的数据库（或让续期得到 503）后刷新页面"，并另列"停掉 nerve：维护页，恢复后自动回来"。维护页的文案"Looks like Nerve didn't start up correctly!"在 nerve 只是暂时不可达时有误导，P4 不改（7.4 只要求"仍显示维护页"），建议 P5 或 M8 改写。
15. **`token-manager.test.ts` 472 行**：超过约 400 行的规则。它是一个类的单标签页测试，按 `describe` 分三组，拆开要重复假实现的搭建；与 P3b 的 `load_test.go` 同样处理，plan 的 Global Constraints 写明。
16. **每一代 stores 一个绑定会话的客户端**（控制者加的 Task 10b）：7.1、7.5 要求标签页不以错误的身份写入，但原来所有 stores 共用一个模块级的 `api`，它的中间件取标签页此刻会话的令牌。在会话 X 中开始、`await` 之后才发下一个请求的操作，标签页在这中间跟随另一个标签页以 Y 登录之后，下一个请求带着 Y 的令牌发出：资料步骤以 X 的令牌保存名字（200）之后改 `onboarding_step`，写进了 Y 的资料；新手引导的创建工作区、`/create-workspace`、`/invitations` 之后写 `last_workspace_id`，新手引导根组件的后续写入，也是这个形状（Task 7、Task 8 发现）。逐处比较 `loginId` 只能补一处，所以改成一个机制：中间件属于一个会话，标签页已不在这个会话时请求在发出之前以 `SessionChangedError` 失败（2.6）；`api-client.ts` 不再有模块级的带令牌客户端，`apiFor(loginId)` 为一代 stores 建一个；`RootStore` 在建立和重建时把它交给 stores，`UserService` 从构造函数接收它（2.8）。为此 `store-context.tsx` 在从没有会话变成有会话时（本标签页登录，或跟随另一个标签页登录）也重建 stores：没有会话时建的 stores 属于"没有会话"，不重建的话登录之后它们的请求都会被拒绝（`auth-middleware.session.test.ts` 的最后一个测试）；Task 7 的"登录时 stores 已是新的，不重建"因此不再成立。这次重建也照 `resetOnSignOut` 把主题设为跟随系统、语言设为默认；登录之后 `fetchUserProfile` 按资料设语言，`StoreWrapper` 在包装层跳转之后按资料设主题。Plane 的模块级 `userService` 删除，唯一的使用者 `UserPermissionStore` 改为自己建实例。7.1 的位置一项写 `api-client.ts` 是"web 使用的生成客户端实例，挂上认证中间件"，现在是 `publicClient`、令牌管理器和 `apiFor`，设计文本请随之更正。

## 4. 验收标准（完成线，M2 设计 12 节 P4）

- [ ] `git grep -n -E "csrfmiddlewaretoken|X-CSRFTOKEN" -- web` 没有输出。
- [ ] A1–A6、A10、A15 的页面版本（A4 有、没有 `navigator.locks` 两遍，A6 含两种切换账户）和 S2 通过；此前的故事仍然通过（`make e2e` 共 35 个测试）。
- [ ] 令牌管理器、锁、中间件的 72 个单元测试通过；去掉锁、去掉写回前的 `login_id` 核对、429 或 5xx 结束会话、去掉 8 秒超时、去掉租约的锁，每个变异都让至少一个测试失败（附录 A）。
- [ ] 9.6 中 P4 的浏览器核对（局域网 HTTP、两个账户两个标签页的两种走法、会话暂不可用、第一次打开新手引导和登录后的落点、CSP）在合并前的代码上重跑，写进 review；M0-P5"同一层"的关闭理由写进 review。
- [ ] `make lint`、`make test`、`make gen-check`、`make knip`、`make test-web`、`make e2e` 通过；web 的 oxlint 上限 551。
- [ ] 3.20 中 P4 的各行、8.7 中 P4 的 README 内容在同一次合并中写好；交接按第 7 节处理。

## 5. 不在 P4 范围内

- **P5**：个人设置的 preferences（时区 `GET /api/v0/timezones`）、security 的 PAT 列表、api-tokens；PAT store；`@nerve/services` 删到只剩地址工具和文件工具（M1-P3 交接剩下的 API 令牌旧地址）；剩下的 Plane 类型（`IApiToken`、`TTimezones` 等）；主题下拉框的定位；死成员和死 prop 的其余部分（含 `settings.store`）；oxlint 在改到的文件中清零（P4 改到的 6 个文件还有 12 条，第 6 节）和 `no-unneeded-ternary` 一类；关键词规则 `withCredentials:\s*true`、`/api/users/api-tokens`、`/api/timezones/`；A7–A9、A11、A12 的页面版本；9.6 中 P5 的浏览器核对。
- **M3**：工作区的落点数据；新手引导的邀请和工作区列表；工作区页面里的退出入口；`IUserLite.avatar_url` 可为 `null`。
- **M5**：头像、封面的上传。
- 7.1 的"不做"：定时主动续期；把访问令牌放进 sessionStorage。

## 6. 风险

| 风险 | 应对 |
|---|---|
| 租约不是原子的：两个没有 `navigator.locks` 的标签页可能同时拿到租约（7.1、§16） | 写回前核对 `login_id`，另一个账户的记录不会被覆盖（多标签页测试、C4b）；同一个会话的两次同时续期的代价是一次重复使用检测、重新登录，与续期响应丢失相同（3.5）。原型中 A4 的租约一遍、C5 各跑多次，续期全部 200、代数不重复 |
| 持有租约的标签页被冻结超过 10 秒 | 原型没有复现（要让浏览器真的冻结一个标签页）；续期 8 秒超时早于 10 秒的租期（多标签页测试"8 秒放开锁"）；剩下的情况与响应丢失一样只能观察（§16） |
| 生产构建下停掉 nerve，页面本身加载不了 | 浏览器显示自己的错误页，nerve 恢复后用户刷新即可；"会话暂不可用"只在 nerve 回答、续期失败时出现（第 3 节第 14 条，C6b、C6c） |
| 维护页的文案在 nerve 暂时不可达时有误导 | P4 不改；建议 P5 或 M8 改写（第 3 节第 14 条） |
| `style-src 'unsafe-inline'` | 脚本仍只允许 nerve 的文件和按哈希放行的内联脚本；React 的 `style` 属性和 popper 的定位需要内联样式；C9 在 8 个页面上没有违规 |
| A4 依赖 1 秒的扣住和 3 秒的访问令牌 | 原型中 A4 的页面版本重复 6 次、页面故事重复 4 次都通过；扣住有期限，放行后测试照常断言 |
| P4 改到的 6 个文件还有 12 条 oxlint 警告 | 7.8 要求 M2 结束时清零；P5 清（第 5 节） |
| 偏好页、api-tokens 页挂载时仍请求 Plane 的旧接口（404） | P5 改写这两页；C9 记下；它们不在 S2、A10 的页面上 |
| 持续集成（ubuntu）上的运行 | 原型只在 macOS 上运行；合并后看持续集成 |

## 7. 交接的处理

| 交接 | P4 处理的条目 | 留下的条目 | 状态 |
|---|---|---|---|
| M0-P5-frontend-api-notes | 前端改调 `/api/v0/instance`；认证接口在 `/api/v0/` 下；同源（Task 6、7、10）。CSP（Task 11）；"安全响应头与 CSP 放在同一层"按 8.3 的理由正式关闭 | — | done |
| M0-P6-e2e-notes | 页面的登录状态：`signedInPage`（Task 12）；S2 的"没有失败的接口请求"、控制台、CSP：`watchPage`；新加的等待都有期限 | fixture 写法的延伸（M4、M5、M8） | open |
| M1-P2-trim-content | 主题：`IUserTheme` 删除，改用生成的 `Theme`（Task 8） | — | done（第 3 节第 11 条） |
| M1-P3-trim-platform | Cookie 会话和 CSRF（Task 7、9、10）；认证错误就地显示（Task 10）；前端改读实例字段，`is_self_managed` 和两步（Task 5、6）；新手引导挂载时的旧接口（Task 5） | `@nerve/services` 的 API 令牌旧地址（P5） | open |
| M1-P4-router-native | 服务端校验 `next_path` 按"服务端没有跳转"关闭（3.18），前端补上控制字符（Task 9）；`AuthenticationWrapper` 重写（Task 9）；401 不照搬恒为真的判断（Task 9）；由 M2 决定的三项：查询和片段（Task 9）、`Error & { error_code }`（Task 10）、表单不再提交到 `/auth/…`（Task 10） | — | done |
| M1-closeout（不追加处理记录，P5 关闭它时一并写） | 改到的文件中的死成员随改随删（`UserStore` 的 `reset`、`isAuthenticated`、`error` 等） | 其余死成员；oxlint 在改到的文件中清零、`no-unneeded-ternary`；主题下拉框（P5） | open |

评审交给 P4 的事项：

| 事项 | 落在 |
|---|---|
| Codex I-5、R4：另一个标签页换了账户；`login_id` 的写入竞争 | 2.5；Task 2、3 的测试；A6 的两种切换；浏览器核对 C4a–C4c |
| Codex I-11：新手引导挂载时请求 M3 旧接口 | 2.11；A10；C8 |
| Codex M-5：续期非 401 失败时不退出 | 2.5、2.6；令牌管理器和中间件的测试；C6b、C6c |
| 7.1：原型先验证 openapi-fetch 0.17.0 的中间件能重发 | 2.6；中间件测试（附录 A 的 E1） |
| §16：重复使用检测是否频繁 | C3、C5 共 15 次页面续期全部 200，会话未被撤销；A4 各遍同样 |
| P2 评审第 7 节：真实反向代理后面的客户端 IP（"P4 或 M8"） | 浏览器核对 C11（Caddy）：不信任代理时记下代理的地址并记 WARN，`server.trusted_proxies` 设为代理的地址后记下客户端的地址。代价很小，在 P4 做完，不留给 M8 |

## 附录 A：原型验证记录（2026-09-27）

原型在 `$M2TMP/p4proto`（`f27434c` 的副本；Node 24.15.0、pnpm 11.10.0、Go 1.27.1、Docker 29.7.2、Playwright 1.63.0 的 Chromium）。plan 中的代码就是原型中运行过的代码，由脚本从原型文件原样拼入 plan：新文件和改动大的文件给完整内容，改动小的给对前一个版本的统一差异；拼接脚本用 `patch -p1` 把 72 个差异块逐个应用到前一个版本上，结果都与原型的文件逐字节相同；plan 提交前重新拼了一次，与提交的 plan 逐字节相同。另一个脚本核对每个 Task 的文件快照：`f27434c` 与原型之间的每一处差异（102 个改动或新增的文件、9 个删除的文件）都由某个 Task 写出或删除。

**逐 Task 复现**：在 `$M2TMP/p4stage`（`f27434c` 的另一个副本，`pnpm install --frozen-lockfile`）按 plan 的 14 个 Task 依次执行：放入这个 Task 写的文件（最终版本或过渡版本，即 plan 中的内容），删除它删除的文件；执行 plan 中的命令（Task 2、8 从前一个 lock 出发的 `pnpm install`，得到的 lock 与 plan 逐字节相同；Task 2 的 `make gen-web`，生成物与 plan 表中的 SHA-256 所对应的文件逐字节相同；Task 5–10、14 的 `grep` 核对，都没有输出）；核对这个 Task 写的文件没有新的 `&lt;`、`&gt;`、`&amp;`；然后关键词守卫、前端检查、`make knip`、web 单元测试；Task 11 另跑 `make lint-go`（两段都是 `0 issues.`）和 `make test`（32 个包全部 `ok`）；Task 11、12、13 跑 `make e2e`。每个 Task 7–47 秒。14 个 Task 之后，复现的目录与原型逐文件相同（2626 个文件，不含依赖和构建产物）。

| 核对 | 命令 | 结果 |
|---|---|---|
| 生成物 | Task 2 的 `make gen-web`（副本不是 git 仓库，`make gen-check` 用不了；原型上用等价脚本重新生成到临时目录再比较） | `schema.gen.ts` 与 plan 的 SHA-256 相同，其余生成物不变 |
| 前端检查 | `turbo run check:types check:lint check:format check:sync`（54 个任务）；关键词守卫（真实的 `tools/keywords.mjs`，`git ls-files` 由遍历目录的替身代替）；`make knip`；`turbo run test`（16 个任务） | 每个 Task 都通过；web 的 oxlint 警告依次是 565、560（Task 5）、555（Task 7）、554（Task 8）、551（Task 10），各等于上限；关键词规则 51、52、53、56 条，没有命中 |
| web 单元测试 | vitest | `core/lib/auth` 72 个（锁 9、令牌管理器 30、多标签页 20、中间件 13）；`core/lib/api-error` 5 个；`helpers` 9 个；`@nerve/utils` 的 `next-path` 20 个、`auth` 5 个 |
| Go | `make lint-go`；`make test`（testcontainers，`postgres:18.6`） | 0 issues ×2；32 个包全部通过 |
| 端到端 | `make e2e`（Task 11、12、13） | 22、28、34 个测试中除 S3 外全部通过（Playwright 计时 5.6、4.2、12.3 秒）。S3 失败的原因与 P1–P3b 相同：副本不是 git 仓库，`commit` 是 `unknown` |
| 页面故事重复 | `make build`，然后 `NERVE_VERSION=0.1.0-dev pnpm -C e2e exec playwright test stories/identity stories/smoke/s2-web-app.spec.ts --grep "page\|S2" --repeat-each=4` | 60 个全部通过（17.9 秒） |
| A1、A4、A10 重复 | A1 的页面版本 `--repeat-each=3`；A4、A10 的页面版本 `--repeat-each=6` | 6 个、18 个全部通过 |
| 完成线 | `grep -rniE "csrfmiddlewaretoken\|X-CSRFTOKEN\|csrf\|EAuthenticationErrorCodes\|error_code" web`（排除 `node_modules`、`dist`、`build`、`.react-router`、`.turbo`） | 没有输出 |

**原型中定下的事实**：

- **E1** openapi-fetch 0.17.0 的中间件：`onRequest` 中 `request.clone()` 的副本在 fetch 读走原请求的请求体之后仍可读；`onResponse` 收到的 `request` 与 `onRequest` 的是同一个对象，可以做 `WeakMap` 的键；`options.fetch` 可以用来重发。中间件测试中重发的 `PATCH` 与原来的方法、路径、请求体都相同；"用原请求重发"的变异失败（请求体已被读走）。7.1 的备选方案（在 service 层包一层"401 后续期重试"）不需要。
- **E2** Playwright 1.63.0：`waitForLoadState("networkidle")` 在客户端导航之后立即返回（页面已处于 idle），不等导航之后的请求；被 `route` 扣住的请求，`request.timing()` 从放行时算起，不能用来判断两个请求是否重叠。
- **E3** `make build` 构建出的 `index.html` 有 5 个内联脚本（React Router 的）；CSP 按哈希放行后，C9 的 8 个页面、S2 和 A10 都没有 `securitypolicyviolation`。
- **E4** 非安全上下文（Chromium 打开 `http://10.10.20.104:<端口>`）：`window.isSecureContext` 为假，没有 `navigator.locks`，`crypto.getRandomValues` 可用，`crypto.randomUUID` 是 `undefined`。
- **E5** 页面续期的次数：C3 两个标签页 8 次续期，代数 0–7 各一次，结束时会话的代数是 8；C5 局域网 HTTP 7 次，代数 7；没有一次重复使用检测。

**9.6 的浏览器核对**：`make build`，然后 `node $M2TMP/p4tools/checks/checks.mjs 10.10.20.104`（脚本在 `$M2TMP/p4tools/checks/`，不进仓库；review 附录按 9.6 收录执行时重跑的脚本全文和结果）。脚本对原型的 `bin/nerve` 运行，每个核对在开发库 `nerve-dev-db-1` 上建一个独立的库、结束时删除；C6b 另起一个 Postgres 容器，C11 另起一个 Caddy 容器，结束时都删除。结果是最后一次完整运行（`f8da093d`）；C3、C5 另外连续重跑 3 次，都通过。

| 核对 | 9.6 的条目 | 做法 | 结果 |
|---|---|---|---|
| C1 | 登录页、注册页的就地错误 | 注册已存在的邮箱（大写）；常见密码；弱密码；错误密码；不存在的邮箱 | 通过：409，表单上方"An account with this email already exists."，邮箱留在输入框；422，字段下方"This password is too common"，表单上方没有提示；弱密码显示规则和横幅，一个请求都不发；错误密码、不存在的邮箱都是 401、同一句"The email or the password is wrong."；没有 CSP 违规 |
| C2 | `next_path` 的各种取值 | 10 个取值，各自新的上下文，登录后看落点 | 通过：`/settings/profile/general?tab=x#y` 原样回来；` /settings/profile/preferences` 去掉空格后回来；`/%2F%2Fevil.example` 是站内路径；`//evil.example`、`/\evil.example`、`javascript:alert(1)`、`/\t/evil.example`、`/\u0000/x`、`https://evil.example/x` 都落到 `/create-workspace`；未完成新手引导的账户落到 `/onboarding`；都在同源 |
| C3 | 两个标签页的续期和退出 | 访问令牌 3 秒的 nerve，两个标签页交替操作，然后一个标签页退出 | 通过：8 次续期全部 200，代数 0–7 各一次，一次一个；两个标签页都到 `/?next_path=%2Fonboarding`；会话撤销原因 `logout` |
| C4a | 两个账户两个标签页，切换 ①（`navigator.locks`） | 甲以 X 登录；乙在锁下换上 Y 的记录，不退出 | 通过：甲显示 Y；此后保存的名字写进 Y（"Yvonne"），X 的名字不变，X 的会话未被撤销 |
| C4b | 切换 ①，没有 `navigator.locks`，正好在续期时 | 删掉 `locks`；X 的续期发出后被扣住，其间换上 Y 的记录，再放行 | 通过：nerve 把 X 从第 2 代轮换到第 3 代，但第 3 代从未写进记录；记录是 Y 的；资料步骤没有前进，两个账户的名字都没有变；没有失败的接口请求 |
| C4c | 切换 ②（先退出再登录） | 乙退出，再以 Y 登录 | 通过：甲随之到 `?next_path=%2Fonboarding`；乙登录 200；两个标签页都以 Y 显示 |
| C5 | 局域网 HTTP | nerve 监听 `0.0.0.0`，浏览器打开 `http://10.10.20.104:<端口>`；注册；两个标签页在访问令牌过期后同时操作 | 通过：`isSecureContext` 为假，没有 `locks`，`getRandomValues` 可用，`randomUUID` 没有；注册 201；两个标签页都成功，7 次续期全部 200、代数 0–6 各一次，会话的代数 7，未被撤销；启动日志有非回环地址的 WARN |
| C6a | 会话暂不可用（照 9.6 的字面：停掉 nerve、刷新） | Vite 开发服务器提供页面，nerve 在 8080；停掉 nerve，刷新，再启动 | 显示的是维护页，不是"会话暂不可用"（第 3 节第 14 条）；地址和记录都在；nerve 启动后 4.9 秒自动回到原页面 |
| C6b | 会话暂不可用（nerve 回答，续期失败） | nerve 用自己的 Postgres 容器；停掉数据库，刷新，再启动数据库 | 通过："Cannot reach the server for now"和"Try again"；续期 500；地址和记录都在，没有跳到登录页；数据库启动后 1.4 秒自动恢复，回到原页面 |
| C6c | 会话暂不可用（503 和 `Retry-After`） | 续期答 503 `server_busy`、`Retry-After: 2`，然后放开 | 通过：显示"会话暂不可用"；放开后 2.3 秒自己恢复，地址不变 |
| C7 | 实例请求失败时的维护页 | 实例请求被拒绝连接 | 通过：维护页 |
| C8 | 第一次打开 `/onboarding`、登录后的落点 | 在注册页注册，到新手引导，保存名字；另一个完成新手引导的账户登录 | 通过：两段都没有失败的接口请求、旧接口请求、未处理的异常、控制台错误、CSP 违规；新手引导的请求只有 `GET /api/v0/instance`、`POST /api/v0/auth/register`、`GET /api/v0/me`、`GET /api/v0/me/profile`、`PATCH /api/v0/me`、`PATCH /api/v0/me/profile`；资料步骤和下一步出现；登录后落到 `/create-workspace`，请求只有实例、登录、`/me`、`/me/profile` |
| C9 | 登录页、新手引导页、设置页没有 CSP 违规 | `/`、`/sign-up`、`/onboarding`、`/create-workspace` 和个人设置的四页 | 通过：8 个页面都带策略，没有违规。偏好页请求 `/api/timezones/`、api-tokens 页请求 `/api/users/api-tokens/`，都是 404（P5 改写这两页） |
| C10 | 控制台没有"navigate() 应在 useEffect 中调用"的警告 | 收集以上所有页面的控制台警告 | 通过：没有这类警告；只有 6 条 Chromium 的 Canvas2D `willReadFrequently` 提示 |
| C11 | 真实反向代理后面的客户端 IP（P2 评审第 7 节） | Caddy 2.10（容器）把请求转给 nerve；先不信任、再把 `server.trusted_proxies` 设为 `127.0.0.1/32`；另从本机直接注册 | 通过：不信任时会话记下代理的地址 `127.0.0.1` 并记 WARN "ignored X-Forwarded-For from a peer that is not a trusted proxy…"；信任后记下客户端的地址 `172.17.0.1`；直接访问记下 `127.0.0.1` |

**变异核对**（`$M2TMP/p4tools/m_*.py`，由 `mutall.sh` 依次运行：改一处代码，跑相关的单元测试，或重新构建前端和 `bin/nerve` 后跑相关的故事，再恢复）。共 119 个变异，按缺陷类别归类；每个变异按最后一次运行计，117 个被发现，2 个没有（见下）：

| Task | A | B | C | D | F | H | I | K | 合计 |
|---|---|---|---|---|---|---|---|---|---|
| 1 锁 | 6 | | | 9 | | 1 | 3 | 2 | 21 |
| 2 令牌管理器 | 24 | | 1 | 8 | | 3 | 6 | | 42 |
| 4 中间件 | 10 | | 2 | | | | | | 12 |
| 5 新手引导 | 1 | 3 | | | | | | | 4 |
| 6 `unwrap` | 3 | | | | | | | | 3 |
| 7 stores | | 1 | | | | | | 2 | 3 |
| 9 `next_path` | 7 | | | | | | | | 7 |
| 10 错误文案、密码规则 | 11 | | | | | | | | 11 |
| 11 CSP | 9 | 1 | | | | | | 2 | 12 |
| 12 A1 | | | | | 4 | | | | 4 |
| 合计 | 71 | 5 | 3 | 17 | 4 | 4 | 9 | 6 | 119 |

类别：A 属性去掉了测试仍通过；B 断言不可能失败；C 假实现忽略参数；D 只有一个标签页或一个账户，看不出跨标签页、跨账户的问题；F 令牌的编码形式出现在存储、地址或控制台中；H 测试挂住而不是失败；I 读写挪到锁外（测试的闸门在锁内）；K 接线没人看。页面上的变异按它改的产品代码所属的 Task 计（例如"`webLock` 不加锁"让 A4 失败，计在 Task 1）。

brief 要求的五个，和有代表性的几个：

| 变异 | 结果 |
|---|---|
| 去掉锁（令牌管理器的续期和写入都不经过锁） | 50 个令牌管理器测试中 17 个失败：写入时 `held` 为假；两种锁的"续期一个接一个"失败 |
| 去掉写回前的 `login_id` 核对 | 5 个测试失败：两种锁的"续期途中记录换成另一个账户"（nerve 答 200、答 401 各一个），以及"另一个标签页登录之后发出的请求" |
| 429 结束会话 / 5xx 结束会话 / 网络错误结束会话 / 400 结束会话 | 分别 2、7、3、1 个测试失败 |
| 去掉 8 秒超时 / 超时改为 10 秒、5 秒 / 不把 `signal` 交给 fetch | 分别 4、3、5、4 个测试失败（"8 秒放弃续期"、两种锁的"8 秒放开锁，早于租约"、退出的 8 秒）；在假时钟上失败，不挂住 |
| 租约的锁去掉（`acquire` 直接返回） | 锁的 9 个测试中 7 个失败；A4 的租约一遍另有"不取租约"的变异，失败 |
| 页面上 `webLock` 不加锁 / 不取租约 | A4 的对应一遍失败：两个标签页发出同一代刷新令牌，nerve 当成被盗 |
| 删掉 `navigator.locks`、代码不变（A4 的第二遍） | 通过（租约接手） |
| 续期、登录、退出、结束会话各自挪到锁外；锁下不读记录；退出在锁外读刷新令牌 | 各有测试失败（I 类，6 个） |
| 续期在会话之间共用；旧续期结束时清掉新续期；已退出时 `renew` 仍续期 | 各 1 个测试失败（第 3 节第 2、4 条） |
| 中间件：不带令牌、不重发、用旧令牌重发、用已读走的请求重发、设令牌之后才复制、重发 401 不结束会话、任何错误都续期、未登录时发出的请求在会话开始后重发、续期 429 或 5xx 时结束会话或交回 401 | 中间件的对应测试失败 |
| 租约不读回、读回不等、不排本标签页的队、不轮询、删除别人的租约、忽略过期、租期 20 秒、读回等 50 毫秒、轮询 400 毫秒、改键名、改锁名、`webLock` 不独占 | 锁的对应测试失败 |
| `unwrap` 把 4xx 当数据 / 任何 JSON 都当 problem / 消息不取 `detail` | `api-error.test.ts` 失败 |
| `next_path` 放过控制字符 / 放过 DEL / 不编码 | `next-path.test.ts` 失败；页面上不带 `next_path`、忽略它、不校验它，A3 失败 |
| 文案表少一个码 / 多一个码 / 指向不存在的文案 / 字段错误不分字段 / 会话暂不可用当作未知错误 | `authentication.helper.test.ts` 失败；页面上字段错误不显示、提示都是通用的，A2、A15 失败 |
| 密码不设上限 / 按码点计 / 规则只查下限 | `auth.test.ts` 失败 |
| CSP：哈希脚本文件、不去重、标签区分大小写、结束标签不许空白、`'unsafe-inline'` 脚本、十六进制哈希、所有文件都带、不设、没有构建时也带、少一条指令、接口也带 | `csp_test.go`、`headers_test.go` 的对应测试失败；页面上漏掉一个内联脚本的哈希，S2 的两个测试失败（CSP 违规） |
| 重置 stores 时重建 `instance` / 不订阅 `storage` 事件 / 切换账户时保留旧的访问令牌 | A5、A6 的三个测试、A6 的切换 ① 失败 |
| 未登录时取 `/me`（两处守卫都去掉） | S2 的两个测试失败 |
| 新手引导的预取放回根组件 / 挂载时一个未处理的拒绝 / 一个控制台错误 | A10 失败 |
| 新手引导的预取原样放回 `page.tsx`（Task 5 删除的两个：工作区列表、邀请） / 只放回工作区列表的一个（Task 12 实测） | A10 的两个页面测试都失败 / 应用内进入的一个失败，整页加载的一个通过（见下） |
| 令牌写进 sessionStorage / 另一个 localStorage 键 / 控制台 / 请求地址 | A1 失败 |

**没被发现的**：

- **未登录时取 `/me`，只去掉 `AuthenticationWrapper` 的守卫**（未登录时也按 `["CURRENT_USER", loginId]` 取账户）：`UserStore.fetchCurrentUser` 未登录时不请求，这第二道守卫挡住了它，S2 通过。两道守卫都去掉的变异让 S2 失败（上表）。两道守卫各有用处（store 对任何调用方都守住 7.1 的规则，包装不为未登录的页面取数），不删其中一道。
- **只把工作区列表的预取放回 `page.tsx`，整页加载进入**：整页加载时页面挂载在账户取到之前，`if (user?.id)` 使这个键不变的预取从不执行，整页加载的 A10 通过，在这条路上是等价的变异。注册之后应用内进入时账户已经取到，A10 的另一个页面测试发现它（上表；下面"新手引导的预取"一条）。

**原型中最初没被发现、改了测试或代码之后才被发现的**：

- **`unwrap` 把没有 `code` 的 JSON 当作 problem**：第一稿的测试只有 problem+json 和 HTML 页面；加上 `{message}` 的用例后被发现。
- **旧续期结束时清掉新续期**（第 3 节第 2 条的实现细节）：第一稿的去重在 `finally` 中无条件清空，新会话的续期会被旧会话的结束清掉；加上"另一个标签页登录之后发出的请求"的测试后被发现。
- **A4 没有锁也通过**（D 类）：第一稿不扣住续期，两个标签页的续期很少真的重叠，"`webLock` 不加锁"的变异通过；扣住第一个续期、改用代数断言之后，两遍都被发现（第 3 节第 8 条）。
- **未登录时取 `/me`**：第一稿只有一道守卫，变异的判断见上。
- **新手引导的预取**：原型的第一稿把放回 `page.tsx` 的变异算作等价（页面挂载时账户还没取到），改为放回根组件后被 A10 发现。这只对整页加载、只对工作区列表的预取成立（Task 12 实测更正）：邀请的预取以带 `user?.id` 的键请求，账户取到后键一变就请求，整页加载也会发出；注册之后应用内跳到 `/onboarding` 时账户已经取到，两个预取在挂载时都会请求。A10 因此有两个页面测试（2.14）：Task 5 删除的代码原样放回 `page.tsx`，两个都失败（整页加载：`404 GET /api/users/me/workspaces/invitations/`；应用内进入：邀请和 `/api/users/me/workspaces/` 两个 404）；只放回工作区列表的预取，应用内进入的一个失败（`404 GET /api/users/me/workspaces/`）。
- **中间件对未登录时发出的请求重发**：`renew` 加上"已退出时给 `undefined`"之后，这个变异不再被发现；加上"本标签页在 401 回来之前登录了"的测试后重新被发现（第 3 节第 4 条）。
- **C5 和 A4、A10 的等待**：`networkidle` 立即返回（E2），C5 一次运行中续期数少于代数（最后一个续期和 `PATCH` 还在途中）；改为等 `PATCH /api/v0/me/profile` 的回答后，C3、C5 连续 3 次通过，A4、A10 重复 6 次通过。
- **`renew` 的空记录**：类型检查发现（第 3 节第 4 条），加上测试和变异。
- **F 类没有覆盖**：第一稿的 A1 只看 `nerve.auth` 的两个键；加上令牌主体的查找后，4 个 F 类变异都被发现（第 3 节第 10 条）。

**缺陷类别**（brief 列出的类别，对本 plan 逐类核对）：

| 类别 | 核对了什么 | 结果 |
|---|---|---|
| 测试不失败（A） | 每条规则和边界（30 秒余量、8 秒超时、10 秒租期、100 毫秒读回、200 毫秒轮询、退避的上限、8 和 128 个字符、UTF-16 码元） | 71 个 A 类变异都被发现；发现并修正三处（`unwrap`、旧续期清掉新续期、`renew` 的空记录） |
| 断言不可能失败（B） | "没有请求 `/me`""没有旧接口的请求""没有违规"都先断言页面确实渲染、确实发出了别的请求；A10 核对未处理的拒绝和控制台错误；S2 的 CSP 断言由漏掉哈希的变异证明能失败 | 5 个 B 类变异都被发现；预取的等价变异见上 |
| 假实现忽略参数（C） | `FakeNerve` 记下方法、路径、`Authorization`、请求体，测试核对续期和退出交出的刷新令牌、重发的请求体；`RecordingLock` 记下写入时是否持锁 | 3 个 C 类变异都被发现 |
| 一个标签页或一个账户（D） | 多标签页测试两种锁各 10 个，每个至少两个标签页或"另一个标签页"的写入；A4、A6、C3–C5 两个标签页；A6、C4 两个账户，另一个账户的名字和会话都核对 | 17 个 D 类变异都被发现；发现并修正一处（A4 的扣住） |
| 秘密的编码形式（F） | A1 查两个令牌的主体在 sessionStorage、其他 localStorage 键、地址、请求地址、控制台中都不出现；e2e 的 `recordOf` 只读 `nerve.auth` | 4 个 F 类变异都被发现（最初没有覆盖，见上） |
| 测试挂住（H） | 单元测试都在假时钟上，`settle` 20 秒为限；e2e 的等待都有期限；浏览器核对每一步有期限 | 4 个 H 类变异（去掉超时、不传 `signal`、不轮询、不自动重试）在期限内失败 |
| 闸门在争用区段之外（I） | `FakeNerve` 把续期停在锁内（请求发出之后、写回之前），测试在这时换记录、写入或登录；租约的读回前后各有闸门 | 9 个 I 类变异都被发现 |
| 说明与代码不符（J） | README、总体设计 4.3、交接的处理结果、界面文案（会话暂不可用、错误文案表）、代码注释 | 逐句与代码核对；错误文案表由测试对着 `openapi.yaml` 核对；9.6 的一处设计文本与实际不符（第 3 节第 14 条） |
| 接线没人看（K） | 锁名、租约键、`storage` 事件的订阅、stores 的重建、CSP 的接线（handler 和整个服务）、`signup_enabled` 的链接、有无 `navigator.locks` 两条路 | 6 个 K 类变异都被发现；注册链接由 A2 两种 nerve 核对；两条锁的路由 A4 两遍和 C5 核对 |
| 页面没有渲染被检查的东西 | e2e 每个内容断言之前先断言元素或请求存在（资料步骤、"Go to workspace"按钮、表单上方的提示、`PATCH` 的回答） | 由上面各类的页面变异间接证明：断言都曾因变异而失败 |

**没有证明的**：

- 持续集成（ubuntu）上的运行；S3 在 git 仓库中的结果。合并后看持续集成。
- 生产构建下停掉 nerve 时页面的表现：浏览器自己的错误页，没有实测（第 3 节第 14 条）。
- 持有租约的标签页被真的冻结超过 10 秒（第 6 节）。
- 主题下拉框、偏好页、api-tokens 页（P5）。
