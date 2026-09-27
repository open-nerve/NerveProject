# M2/P4 前端认证 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 浏览器通过令牌管理器注册、登录、续期、退出；Cookie 会话和 CSRF 从前端消失；用 HTTP 部署时多个标签页也能正常续期；另一个标签页登录别的账户时，本标签页不会以错误的身份写入；M2 能到达的页面挂载时不请求 M3 的旧接口；页面带 CSP。A1–A6、A10、A15 的页面版本（A4 含没有 `navigator.locks` 的一遍，A6 含两种切换账户）和 S2 的新断言通过。

**Architecture:** 令牌管理器（`web/apps/web/core/lib/auth/`）是认证状态的唯一来源：访问令牌只在内存里，localStorage 只有一条 `nerve.auth = {refresh_token, login_id}`，每次写入都在跨标签页的锁（`navigator.locks`，没有它时用 localStorage 租约）之下；它的依赖（storage、锁、客户端、时钟、随机数）都从构造函数传入，单元测试用假实现强制顺序。`@nerve/api-client` 用 `--root-types` 直接导出生成的类型；web 有两个客户端：不带令牌的 `publicClient`（实例、注册、登录、令牌管理器自己的续期和退出）和挂着认证中间件的 `api`（带上访问令牌，401 时续期、重发一次）。stores 直接用生成的 `User`、`Profile`、`InstanceInfo`，Plane 的手写类型删除；`AuthenticationWrapper` 只读令牌管理器的状态、账户、资料和 `next_path`，是唯一发出跳转的地方；stores 在会话换人时整体重建。`webui` 在构造时算出 `index.html` 内联脚本的哈希，只给页面设 CSP。

**Tech Stack:** React 19.2.8、React Router 8.3.0、MobX 6.12.0、SWR 2.4.2、react-hook-form 7.84、openapi-fetch 0.17.0、openapi-typescript 7.13.0、TypeScript 5.8.3、vitest 4.1.11、oxlint 1.51.0、oxfmt 0.35.0、knip 6.37.0、turbo 2.10.11；Node 24、pnpm 11.10.0、Playwright 1.63.0；Go 1.27.1、golangci-lint 2.13.2。

**Spec:** `docs/v0/M2-auth/specs/P4-web-auth.md`（上级：`docs/v0/M2-auth/M2-design.md`）

## Global Constraints

- **依赖**：不加任何 npm 注册表上的包，Go 依赖不变。只加两个工作区内的依赖，都是 `workspace:*`：Task 2 让 `web`（`web/apps/web/package.json`）依赖 `@nerve/api-client`，Task 8 让 `@nerve/constants` 依赖它（`THEME_OPTIONS` 的 `value` 用生成的 `Theme`）。改完 `package.json` 之后执行 `pnpm install`，`pnpm-lock.yaml` 的变化只有本 plan 给出的那几行 `link:`（Task 2、Task 8 的差异块）；不一致时停下来。`openapi-fetch` 0.17.0、`openapi-typescript` 7.13.0 已在 `@nerve/api-client` 中锁定，本 plan 只改生成命令的参数（Task 2）。
- **每个 Task 提交前**：`make lint-web`（关键词守卫；各包的类型检查、oxlint 警告数等于上限、格式、中英文翻译键一致）、`make knip`、`make test-web` 都通过。web 的 oxlint 上限（`web/apps/web/package.json` 的 `check:lint`）在 Task 5、7、8、10 依次调低为 560、555、554、551：`tools/lint-cap.mjs` 要求警告数正好等于上限，这几个数就是删掉带警告的代码之后的结果；对不上时停下来，说明某处与本 plan 不一致。Task 2 另执行 `make gen-web`，核对生成物的 SHA-256 和行数，**提交之后**执行 `make gen-check`（它用 `git status` 判断生成物是否已提交）。Task 11 另执行 `make lint-go`（两段都是 `0 issues.`）和 `make test`。Task 11、12、13 执行 `make e2e`（Task 11 跑的是还没改的故事，证明前面的改动没有弄坏它们）。
- **生成的文件不手写、不从本 plan 复制**：执行生成命令，提交它的输出。表中是生成物的 SHA-256 和行数；对不上时停下来。
- **容器**：`make test` 和 `make e2e` 用自己的 testcontainers。开发库 `nerve-dev-db-1` 可以用，但不要停止或重建它，不要执行 `make dev-db-down`、`make dev-db-reset`。不要碰其他项目的容器（`agentforge-*`、`plane-app-*`、`opennerve-*`）。
- **git**：每次 Bash 调用只执行一个 git 命令，不用 `;`、`&&`、`|` 串联 git；不用 `git -C`、`stash`、`clean`、`reset --hard`。`cd` 不与别的命令组合；用绝对路径或 `pnpm -C <目录>`。
- **安装**：除了 Docker、Go、Node 不做任何全局安装；不执行 `corepack enable`（pnpm 已在 PATH 上）。
- **规则**（负责人的长期规则）：
  - 前端没有兼容层：接口的数据结构直接用 `@nerve/api-client` 生成的类型，不写转换；Plane 的手写实体类型随对接删除；
  - 删掉的东西删干净：没有开关、占位、注释掉的代码；关键词守卫在删除的同一个提交里加上规则（M1 设计 7.4）；
  - 沿用现有的写法：MobX store、M1 留下的 React Router 写法、`t()` 取文案（中英文的键一致）；
  - 一个文件只做一件事，不超过约 400 行（`token-manager.test.ts` 472 行：一个类的单标签页测试，按 `describe` 分三组，拆开反而要重复假实现的搭建；与 P3b 的 `load_test.go` 同样处理）；
  - 单元测试凡是声称顺序或竞争的，都用假时钟、可控的锁或手动的闸门强制顺序，不用 sleep；每个等待都有期限；
  - Go、TS 代码的注释用英文；中文文档照本 plan 原样。
- **工具的坑**：Edit、Write 工具曾把字面的 `<` 写成 `&lt;`。写完 TSX、TS 文件后 `grep -rn "&lt;\|&gt;\|&amp;"` 这次改过的文件，应当没有输出。
- **代码块**：标为"新文件"或"完整内容"的块是**完整的文件内容**，照原样写入，不要改动（原型中逐字节运行过）；标为"差异"的块是对这个文件当前版本（`f27434c` 或前一个 Task 写的版本）的统一差异，照差异修改，改完的文件与原型逐字节相同。差异块可以存成文件后在仓库根目录用 `patch -p1 < <文件>` 应用（块里的路径是 `a/<路径>`、`b/<路径>`；拼 plan 的脚本已用 `patch -p1` 逐个核对过，72 个差异块都得出原型中的文件）。
- **删除**：标为"删除"的文件用 `git rm` 删掉，之后用给出的 `grep` 核对没有残留的引用。
- **过渡版本**：少数文件先在较早的 Task 写成过渡版本，较晚的 Task 再写成最终版本；每个过渡版本都在原型的逐 Task 复现中运行过（spec 附录 A）。
- **提交**：提交信息用英文，末尾加一行：`Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>`
- 所有命令在仓库根目录下执行，除非步骤中另有说明。

## 文件结构

路径相对于仓库根目录，`web/apps/web/` 简写为 `web:`。"生成"表示由命令生成并提交；"过渡"表示这个 Task 写过渡版本，后面的 Task 写最终版本；"删除"表示这个 Task 删掉它。

| 文件 | 职责 | Task |
|---|---|---|
| `web:core/lib/auth/refresh-lock.ts`、`refresh-lock.test.ts` | 跨标签页的锁：`navigator.locks` 和 localStorage 租约 | 1 |
| `web:core/lib/auth/fake-browser.ts` | 测试替身：多个标签页共用的 localStorage、看得见持有者的锁、手动闸门 | 1（过渡）、2 |
| `web/packages/api-client/package.json`、`src/index.ts` | `--root-types`；`ApiClient`；`Middleware` | 2（`index.ts` 过渡）、4 |
| `web/packages/api-client/src/schema.gen.ts`（生成） | | 2 |
| `web/apps/web/package.json`（修改） | 依赖 `@nerve/api-client`；oxlint 上限 | 2、5、7、8、10 |
| `pnpm-lock.yaml`（修改） | 两处 `link:` | 2（过渡）、8 |
| `web:core/lib/auth/token-manager.ts`、`token-manager.test.ts`、`fake-nerve.ts`、`fake-time.ts` | 令牌管理器；假的 nerve 和时间工具 | 2 |
| `web:core/lib/auth/token-manager.tabs.test.ts` | 多个标签页（两种锁各一遍） | 3 |
| `web:core/lib/auth/auth-middleware.ts`、`auth-middleware.test.ts` | 带上访问令牌；401 时续期、重发一次 | 4 |
| `web:core/components/onboarding/steps/role/`、`usecase/`（删除） | 新手引导的"角色""用途"两步 | 5 |
| `web:app/(all)/onboarding/page.tsx`、`web:core/components/onboarding/header.tsx`、`steps/root.tsx`（修改） | 三步的新手引导；挂载时不预取 | 5 |
| `web:core/components/onboarding/root.tsx`（修改） | 新手引导的根：三步；步骤的部分更新 | 5（过渡）、8 |
| `web:core/components/onboarding/steps/profile/root.tsx`（修改） | 资料步骤：没有头像上传；保存成功才进入下一步 | 5（过渡）、7 |
| `web/packages/types/src/workspace.ts`（修改） | `EOnboardingSteps` 只剩三步；`IUser` 的引用 | 5（过渡）、7 |
| `web:core/lib/api-error.ts`、`api-error.test.ts` | `ApiError`、`unwrap` | 6 |
| `web:core/lib/auth/api-client.ts` | web 的两个客户端和令牌管理器的组合 | 6（过渡）、7 |
| `web:core/services/instance.service.ts`、`web:core/store/instance.store.ts`（修改） | `GET /api/v0/instance`；`InstanceInfo` | 6 |
| `web/packages/types/src/instance/`（删除）、`web/packages/types/src/index.ts`（修改） | `IInstanceInfo`、`IInstanceConfig`；`auth.ts` 的导出 | 6（过渡）、10 |
| `web:app/(all)/create-workspace/page.tsx`、`web:core/components/auth-screens/header.tsx`、`power-k/config/creation/command.ts`、`workspace/sidebar/workspace-menu-root.tsx`（修改） | 读 `signup_enabled`、`workspace_creation_enabled` | 6 |
| `web:core/components/onboarding/steps/workspace/create.tsx`（修改） | 同上；`User` | 6（过渡）、7 |
| `tools/keywords.json`（修改） | P4 的五条规则 | 6（过渡）、8（过渡）、10 |
| `web:core/lib/store-context.tsx`、`web:core/store/root.store.ts`（修改） | 会话换人时重建 stores；`resetOnSignOut` 不动实例和路由 | 7 |
| `web:core/store/user/index.ts`（修改） | `UserStore`：`User`；经令牌管理器登录、注册、退出、停用 | 7（过渡）、10 |
| `web:core/services/auth.service.ts`（修改） | `register`、`login` | 7（过渡）、10 |
| `web:core/services/user.service.ts`（修改） | 生成客户端的薄封装 | 7（过渡）、8 |
| `web/packages/types/src/users.ts`（修改） | 删除 `IUser`、`TUserProfile`、`IUserTheme`、`TOnboardingSteps` | 7（过渡）、8 |
| `web/packages/types/src/project/projects.ts`、`search.ts`（修改）；`web:core/components/{account/deactivate-account-modal,home/user-greetings,issues/issue-detail/reactions/issue,onboarding/switch-account-modal,project/settings/member-columns,workspace/settings/member-columns}.tsx`（修改） | `IUser` 的引用改为 `User` 或 `IUserLite` | 7 |
| `web:core/components/core/modals/user-image-upload-modal.tsx`（删除） | 头像上传（M5 加回） | 7 |
| `web:core/components/settings/profile/content/pages/general/form.tsx`（修改） | 头像和封面只显示；`User`、`UserUpdate` | 7（过渡）、8 |
| `web:core/components/settings/profile/content/pages/security.tsx`（修改） | 修改密码的错误 | 7（过渡）、10 |
| `web:core/store/user/profile.store.ts`（修改） | `ProfileStore`：`Profile`；失败时抛出 | 8 |
| `web/packages/constants/package.json`、`src/themes.ts`（修改） | `Theme` | 8 |
| `web:core/components/{appearance/theme-switcher,power-k/config/preferences-commands,profile/start-of-week-preference}`、`settings/profile/content/pages/general/root.tsx`、`preferences/language-and-timezone-list.tsx`、`web:core/lib/wrappers/store-wrapper.tsx`（修改） | 资料的新类型 | 8 |
| `web:core/lib/wrappers/authentication-wrapper.tsx`（修改） | 认证包装 | 8（过渡）、9 |
| `web:core/lib/auth/use-session.ts`、`web:core/components/account/session-unavailable.tsx` | 会话状态的 hook；"会话暂不可用" | 9 |
| `web:core/services/api.service.ts`（修改） | axios 基类：没有 Cookie，不拦截 401 | 9 |
| `web/packages/utils/src/url.ts`、`next-path.test.ts`（修改） | 控制字符；`signInPath` | 9 |
| `web/packages/i18n/src/locales/{en,zh-CN}/auth.json`（修改） | 会话暂不可用；错误文案表 | 9（过渡）、10 |
| `web:core/components/account/auth-forms/{auth-root,auth-header,password}.tsx`（修改） | 登录页、注册页 | 10 |
| `web:helpers/authentication.helper.tsx`（删除）、`authentication.helper.ts`、`authentication.helper.test.ts` | 页面类型；错误文案表 | 10 |
| `web:vitest.config.ts`（修改） | 测试解析 `@/` | 10 |
| `web/packages/types/src/auth.ts`（删除） | Plane 的认证类型 | 10 |
| `web/packages/utils/src/auth.ts`、`auth.test.ts`；`web/packages/ui/src/form-fields/password/helper.tsx`（修改） | 密码 8–128 个字符 | 10 |
| `server/internal/platform/webui/csp.go`、`csp_test.go`、`handler.go`（修改）；`server/internal/bootstrap/headers_test.go`（修改） | 页面的 CSP | 11 |
| `e2e/tsconfig.json`、`e2e/fixtures/test.ts`（修改）；`e2e/fixtures/auth-pages.ts`、`browser.ts` | `signedInPage`；登录页、注册页的操作；页面的记录 | 12 |
| `e2e/fixtures/auth.ts`（修改） | `signInContext`、`recordOf`；`newRecord`、`writeRecord` | 12（过渡）、13 |
| `e2e/stories/identity/a{1,2,3,10,15}-*.spec.ts`、`e2e/stories/smoke/s2-web-app.spec.ts`（修改） | 页面版本；S2 | 12 |
| `e2e/fixtures/assert/identity.ts`（修改）；`e2e/stories/identity/a{4,5,6}-*.spec.ts`（修改） | `generationOf`；页面版本 | 13 |
| `docs/v0/v0-design.md`、`docs/v0/frontend-changes.md`、`README.md`、`docs/v0/M2-auth/handoffs/{M0-P5,M0-P6,M1-P2,M1-P3,M1-P4}-*.md`（修改） | 文档同步、交接 | 14 |

---

### Task 1: 跨标签页的锁：`navigator.locks` 与 localStorage 租约

**Files:**
- Create: `web/apps/web/core/lib/auth/refresh-lock.ts`、`refresh-lock.test.ts`
- Create（过渡）: `web/apps/web/core/lib/auth/fake-browser.ts`

**Interfaces:**
- Produces（spec 2.4，M2 设计 7.1）：
  - `interface RefreshLock { run<T>(task: () => Promise<T>): Promise<T> }`：同一个浏览器的所有标签页、同一个标签页的所有任务，一次只有一个在锁下运行；
  - `LOCK_NAME = "nerve.auth.refresh"`；`webLock(locks: Pick<LockManager, "request">): RefreshLock`；
  - `LEASE_KEY = "nerve.auth.refresh_lease"`；`leaseLock(deps: LeaseDeps): RefreshLock`，`LeaseDeps = { storage, onStorage, now, tabId }`。租约 `{owner, expires}`：没有、已过期或是自己的就写入，10 秒有效；写入 100 毫秒后读回，仍是自己的才算拿到；拿不到时等租约键的 `storage` 事件，或 200 毫秒后再看；用完只删除自己的；同一个标签页的任务先在本标签页内排队（租约是"本标签页的"，否则第二个任务会直接通过）。
- 测试替身（过渡）：`SharedStorage`（多个标签页的视图共用一份数据；一个标签页的写入在微任务里通知其他标签页，和浏览器的 `storage` 事件一样不通知自己）和 `gate()`（测试手动打开的 Promise）。Task 2 加上 `RecordingLock`。
- 使用者：Task 2 的令牌管理器；Task 7 的 `api-client.ts` 按 `"locks" in navigator` 选一种。

**Tests:**（`refresh-lock.test.ts`，9 个，假时钟）
- 租约：拿到空闲的租约要等读回，任务结束后删除；另一个标签页持有时等待，释放的 `storage` 事件唤醒它；没有 `storage` 事件时靠 200 毫秒的轮询发现租约已释放；接管过期的租约；只删除自己的租约；读回之前被另一个标签页盖掉时继续等；同一个标签页的任务一个接一个；任务失败时释放租约、把失败传出去。
- `webLock`：任务在名为 `nerve.auth.refresh` 的锁下运行，结果原样传出。

- [ ] **Step 1: 测试替身**

`web/apps/web/core/lib/auth/fake-browser.ts`（新文件）：

```ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// Test doubles of the browser for the auth tests: one localStorage shared by several tabs, which tells
// the other tabs about every change as the browser does (a storage event, after the writing task).

type StorageListener = (key: string | null, newValue: string | null) => void;

/** The localStorage of one browser: every tab's view writes the same data. */
export class SharedStorage {
  readonly data = new Map<string, string>();
  private readonly listeners = new Map<string, Set<StorageListener>>();

  /** The view of the tab `tab`: its writes reach the other tabs' listeners, never its own. */
  tab(tab: string) {
    return {
      getItem: (key: string) => this.data.get(key) ?? null,
      setItem: (key: string, value: string) => {
        this.data.set(key, String(value));
        this.notify(tab, key, String(value));
      },
      removeItem: (key: string) => {
        if (!this.data.delete(key)) return;
        this.notify(tab, key, null);
      },
      /** Subscribes this tab to the storage events of the other tabs; returns the unsubscribe. */
      onStorage: (listener: StorageListener) => {
        const set = this.listeners.get(tab) ?? new Set<StorageListener>();
        this.listeners.set(tab, set);
        set.add(listener);
        return () => {
          set.delete(listener);
        };
      },
    };
  }

  /** A write by a tab outside the tests' tabs, e.g. a test playing another tab by hand. */
  write(key: string, value: string | null): void {
    if (value === null) this.data.delete(key);
    else this.data.set(key, value);
    this.notify("elsewhere", key, value);
  }

  private notify(writer: string, key: string, value: string | null): void {
    for (const [tab, set] of this.listeners) {
      if (tab === writer) continue;
      // The browser fires the event later, never inside the write.
      queueMicrotask(() => {
        for (const listener of set) listener(key, value);
      });
    }
  }
}

/** A promise the test resolves or rejects by hand. */
export function gate<T = void>() {
  let open!: (value: T) => void;
  let fail!: (reason: unknown) => void;
  const promise = new Promise<T>((resolve, reject) => {
    open = resolve;
    fail = reject;
  });
  return { promise, open, fail };
}
```

- [ ] **Step 2: 锁**

`web/apps/web/core/lib/auth/refresh-lock.ts`（新文件）：

```ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// The lock the tabs of one browser share to refresh and to write the stored session one at a time (M2
// design 7.1): two tabs refreshing the same refresh token at once would look like a stolen token, and a
// tab writing an old session back could overwrite a newer sign-in.

/** Runs a task while no other tab of this browser, and no other task of this tab, holds the lock. */
export interface RefreshLock {
  run<T>(task: () => Promise<T>): Promise<T>;
}

/** The name of the Web Locks lock. */
export const LOCK_NAME = "nerve.auth.refresh";

/**
 * navigator.locks, which only secure contexts (HTTPS, localhost) have. The browser queues the tasks of
 * every tab, and a frozen tab keeps the lock until it thaws or is discarded.
 */
export function webLock(locks: Pick<LockManager, "request">): RefreshLock {
  return { run: (task) => locks.request(LOCK_NAME, task) };
}

/** The localStorage key of the lease. */
export const LEASE_KEY = "nerve.auth.refresh_lease";
/** How long a lease lasts: longer than a refresh may take (8 s, M2 design 3.5). */
const LEASE_MS = 10_000;
/** How long a tab waits after writing its lease before it reads it back. */
const LEASE_SETTLE_MS = 100;
/** How often a waiting tab looks at the lease, besides the storage events. */
const LEASE_POLL_MS = 200;

type Lease = { owner: string; expires: number };

export type LeaseDeps = {
  storage: Pick<Storage, "getItem" | "setItem" | "removeItem">;
  /** Calls the listener with the key of every storage event from another tab; returns the unsubscribe. */
  onStorage: (listener: (key: string | null) => void) => () => void;
  now: () => number;
  /** This tab's id: random, so it is unique without crypto.randomUUID, which non-secure contexts lack. */
  tabId: string;
};

/**
 * A lease in localStorage, for a page without navigator.locks: plain HTTP on a LAN address. A tab takes
 * the lease when it is free, expired or its own, and holds it after reading it back LEASE_SETTLE_MS later;
 * the others wait for a storage event or poll. localStorage has no compare-and-set, so two tabs taking the
 * lease at the same moment is made unlikely, not impossible (M2 design 7.1, §16).
 */
export function leaseLock(deps: LeaseDeps): RefreshLock {
  // The lease is the tab's, so a second task of the same tab would pass it: the tab's tasks queue first.
  let queue: Promise<unknown> = Promise.resolve();
  return {
    run<T>(task: () => Promise<T>): Promise<T> {
      const turn = queue.then(async () => {
        await acquire(deps);
        try {
          return await task();
        } finally {
          release(deps);
        }
      });
      queue = turn.catch(() => undefined);
      return turn;
    },
  };
}

function readLease(deps: LeaseDeps): Lease | undefined {
  try {
    const lease = JSON.parse(deps.storage.getItem(LEASE_KEY) ?? "null") as Partial<Lease> | null;
    return typeof lease?.owner === "string" && typeof lease.expires === "number"
      ? { owner: lease.owner, expires: lease.expires }
      : undefined;
  } catch {
    return undefined;
  }
}

/** Takes the lease: at once when no other tab holds it, else once the holder lets it go or it expires. */
async function acquire(deps: LeaseDeps): Promise<void> {
  if (await take(deps)) return;
  await changeOrPoll(deps);
  return acquire(deps);
}

/** Writes this tab's lease unless another tab holds one, and tells whether it is still this tab's later. */
async function take(deps: LeaseDeps): Promise<boolean> {
  const lease = readLease(deps);
  if (lease !== undefined && lease.expires > deps.now() && lease.owner !== deps.tabId) return false;
  deps.storage.setItem(LEASE_KEY, JSON.stringify({ owner: deps.tabId, expires: deps.now() + LEASE_MS }));
  await sleep(LEASE_SETTLE_MS);
  return readLease(deps)?.owner === deps.tabId;
}

function release(deps: LeaseDeps): void {
  if (readLease(deps)?.owner === deps.tabId) deps.storage.removeItem(LEASE_KEY);
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/** Resolves on the next storage event about the lease, or after LEASE_POLL_MS. */
function changeOrPoll(deps: LeaseDeps): Promise<void> {
  let unsubscribe: (() => void) | undefined;
  let timer: ReturnType<typeof setTimeout> | undefined;
  return new Promise<void>((resolve) => {
    unsubscribe = deps.onStorage((key) => {
      if (key === LEASE_KEY || key === null) resolve();
    });
    timer = setTimeout(resolve, LEASE_POLL_MS);
  }).finally(() => {
    clearTimeout(timer);
    unsubscribe?.();
  });
}
```

- [ ] **Step 3: 测试**

`web/apps/web/core/lib/auth/refresh-lock.test.ts`（新文件）：

```ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { SharedStorage, gate } from "./fake-browser";
import { LEASE_KEY, LOCK_NAME, leaseLock, webLock } from "./refresh-lock";

// The lease (M2 design 7.1) with fake timers: every wait is a timer the test advances, never a sleep.

function tab(storage: SharedStorage, tabId: string) {
  const view = storage.tab(tabId);
  return leaseLock({ storage: view, onStorage: (listener) => view.onStorage(listener), now: () => Date.now(), tabId });
}

/** Runs a task that records when it starts and ends and waits on a gate the test opens. */
function task(log: string[], name: string) {
  const g = gate<string>();
  const run = async () => {
    log.push(`${name} start ${Date.now()}`);
    const value = await g.promise;
    log.push(`${name} end ${Date.now()}`);
    return value;
  };
  return { run, finish: () => g.open(name) };
}

const lease = (storage: SharedStorage) => JSON.parse(storage.data.get(LEASE_KEY) ?? "null") as unknown;

describe("leaseLock", () => {
  beforeEach(() => {
    vi.useFakeTimers({ now: 0 });
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("takes a free lease once it reads it back, and deletes it after the task", async () => {
    const storage = new SharedStorage();
    const log: string[] = [];
    const a = task(log, "a");
    const done = tab(storage, "A").run(a.run);

    await vi.advanceTimersByTimeAsync(99);
    expect(log).toEqual([]);
    expect(LEASE_KEY).toBe("nerve.auth.refresh_lease");
    expect(lease(storage)).toEqual({ owner: "A", expires: 10_000 });
    await vi.advanceTimersByTimeAsync(1);
    expect(log).toEqual(["a start 100"]);

    a.finish();
    await expect(done).resolves.toBe("a");
    expect(storage.data.has(LEASE_KEY)).toBe(false);
  });

  it("makes another tab wait while it holds the lease, and wakes it on the release", async () => {
    const storage = new SharedStorage();
    const log: string[] = [];
    const [a, b] = [task(log, "a"), task(log, "b")];
    const first = tab(storage, "A").run(a.run);
    await vi.advanceTimersByTimeAsync(100);
    const second = tab(storage, "B").run(b.run);

    await vi.advanceTimersByTimeAsync(5_000);
    expect(log).toEqual(["a start 100"]);

    // The release is a storage event for B, which takes the lease at once instead of at its next poll.
    await vi.advanceTimersByTimeAsync(50);
    a.finish();
    await first;
    await vi.advanceTimersByTimeAsync(100);
    expect(log).toEqual(["a start 100", "a end 5150", "b start 5250"]);
    b.finish();
    await second;
  });

  it("finds a released lease by polling when no storage event comes", async () => {
    const storage = new SharedStorage();
    const log: string[] = [];
    const b = task(log, "b");
    // A tab outside the test holds the lease; it expires in 10 s but goes away sooner, silently.
    storage.data.set(LEASE_KEY, JSON.stringify({ owner: "A", expires: 10_000 }));
    const waiting = tab(storage, "B").run(b.run);

    await vi.advanceTimersByTimeAsync(1_250);
    storage.data.delete(LEASE_KEY);
    // The next poll, at 1400 ms, finds it free; B takes it and starts 100 ms later.
    await vi.advanceTimersByTimeAsync(249);
    expect(log).toEqual([]);
    await vi.advanceTimersByTimeAsync(1);
    expect(log).toEqual(["b start 1500"]);
    b.finish();
    await waiting;
  });

  it("takes over an expired lease", async () => {
    const storage = new SharedStorage();
    const log: string[] = [];
    // A tab took the lease and was frozen: its lease is never released.
    storage.data.set(LEASE_KEY, JSON.stringify({ owner: "A", expires: 10_000 }));
    const b = task(log, "b");
    const waiting = tab(storage, "B").run(b.run);

    await vi.advanceTimersByTimeAsync(10_000 - 1);
    expect(log).toEqual([]);
    await vi.advanceTimersByTimeAsync(200 + 100);
    expect(log).toHaveLength(1);
    expect(lease(storage)).toMatchObject({ owner: "B" });
    b.finish();
    await waiting;
  });

  it("deletes only its own lease", async () => {
    const storage = new SharedStorage();
    const log: string[] = [];
    const [a, b] = [task(log, "a"), task(log, "b")];
    const first = tab(storage, "A").run(a.run);
    await vi.advanceTimersByTimeAsync(100);
    // A holds the lease longer than it lasts; B takes it over.
    const second = tab(storage, "B").run(b.run);
    await vi.advanceTimersByTimeAsync(10_000 + 300);
    expect(lease(storage)).toMatchObject({ owner: "B" });

    a.finish();
    await first;
    expect(lease(storage)).toMatchObject({ owner: "B" });
    b.finish();
    await second;
    expect(storage.data.has(LEASE_KEY)).toBe(false);
  });

  it("waits when another tab wrote its lease over this tab's before the read-back", async () => {
    const storage = new SharedStorage();
    const log: string[] = [];
    const a = task(log, "a");
    const waiting = tab(storage, "A").run(a.run);

    // Another tab saw the lease free at the same moment and wrote its own after A's.
    await vi.advanceTimersByTimeAsync(50);
    storage.write(LEASE_KEY, JSON.stringify({ owner: "B", expires: 50 + 10_000 }));
    await vi.advanceTimersByTimeAsync(3_000);
    expect(log).toEqual([]);
    expect(lease(storage)).toMatchObject({ owner: "B" });

    storage.write(LEASE_KEY, null);
    await vi.advanceTimersByTimeAsync(100);
    expect(log).toEqual(["a start 3150"]);
    a.finish();
    await waiting;
  });

  it("runs the tasks of one tab one after the other", async () => {
    const storage = new SharedStorage();
    const lock = tab(storage, "A");
    const log: string[] = [];
    const [a, b] = [task(log, "a"), task(log, "b")];
    const first = lock.run(a.run);
    const second = lock.run(b.run);

    await vi.advanceTimersByTimeAsync(1_000);
    expect(log).toEqual(["a start 100"]);
    a.finish();
    await first;
    await vi.advanceTimersByTimeAsync(100);
    expect(log).toEqual(["a start 100", "a end 1000", "b start 1100"]);
    b.finish();
    await second;
  });

  it("releases the lease when the task fails, and passes the failure on", async () => {
    const storage = new SharedStorage();
    const lock = tab(storage, "A");
    const failed = lock.run(async () => {
      throw new Error("refresh failed");
    });
    const caught = expect(failed).rejects.toThrow("refresh failed");
    await vi.advanceTimersByTimeAsync(100);
    await caught;
    expect(storage.data.has(LEASE_KEY)).toBe(false);

    const next = lock.run(async () => "next");
    await vi.advanceTimersByTimeAsync(100);
    await expect(next).resolves.toBe("next");
  });
});

describe("webLock", () => {
  it("runs the task under the lock nerve.auth.refresh and passes its result on", async () => {
    const names: string[] = [];
    const locks = {
      request: (async (name: string, granted: () => Promise<unknown>) => {
        names.push(name);
        return granted();
      }) as LockManager["request"],
    };

    await expect(webLock(locks).run(async () => 42)).resolves.toBe(42);
    expect(names).toEqual([LOCK_NAME]);
    expect(LOCK_NAME).toBe("nerve.auth.refresh");
  });
});
```

- [ ] **Step 4: 检查**

Run: `pnpm -C web/apps/web exec vitest run core/lib/auth`
Expected: `refresh-lock.test.ts` 9 个测试通过。

Run: `make lint-web`、`make knip`、`make test-web`
Expected: 都通过（web 的 oxlint 上限仍是 565）。

- [ ] **Step 5: 提交**

```bash
git add web/apps/web/core/lib/auth
```
```bash
git commit -m "feat(M2/P4): the refresh lock: navigator.locks, or a lease in localStorage

One refresh or write of the stored session at a time across the tabs of a
browser. Without navigator.locks (plain HTTP on a LAN address) a lease:
taken when free, expired or ours, held once read back 100 ms later,
awaited through storage events or a 200 ms poll, released only by its
owner; a tab's own tasks queue first.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** 9 个测试通过；它们用假时钟和手动闸门决定每一步的先后，没有 sleep。

---

### Task 2: 令牌管理器；`@nerve/api-client` 的生成类型接入 web

**Files:**
- Modify: `web/packages/api-client/package.json`、`src/index.ts`（过渡）
- Regenerate: `web/packages/api-client/src/schema.gen.ts`
- Modify: `web/apps/web/package.json`（过渡）、`pnpm-lock.yaml`（过渡）
- Modify: `web/apps/web/core/lib/auth/fake-browser.ts`
- Create: `web/apps/web/core/lib/auth/fake-nerve.ts`、`fake-time.ts`、`token-manager.ts`、`token-manager.test.ts`

**Interfaces:**
- Produces（spec 2.3、2.5，M2 设计 3.12、7.1）：
  - `@nerve/api-client`：生成命令加 `--root-types --root-types-no-schema-prefix`，`export type * from "./schema.gen"`（`User`、`Profile`、`AuthTokens`、`Problem` 等直接导出）；`type ApiClient = ReturnType<typeof createClient>`。
  - `web` 依赖 `@nerve/api-client`（`workspace:*`）。
  - `token-manager.ts`：`AUTH_KEY = "nerve.auth"`；`SessionState = { status: "starting" | "signed-in" | "signed-out" | "unavailable"; loginId?; retryAt? }`；`SessionUnavailableError(retryAt)`、`SessionChangedError`；`TokenManagerDeps = { storage, lock, client, now, randomHex }`；`class TokenManager` 的 `state`、`subscribe`、`start`、`retry`、`accessToken`、`renew(sent)`、`signIn(tokens)`、`signOut`、`endSession`、`handleStorageChange(newValue)`。规则见 spec 2.5，要点：
    - 记录 `{refresh_token, login_id}` 一次写入；`login_id` 是 16 个随机字节的十六进制，每次 `signIn` 新生成，续期不变；
    - 启动：没有记录就是 `signed-out`，不发任何请求；有记录时第一次续期决定状态：200 `signed-in`，401 `signed-out`（删除记录），其他（429、5xx、400、网络、超时）`unavailable`，按 `Retry-After` 或 1、2、4……最多 30 秒退避，到时自动重试，`retry()` 立即重试；
    - 访问令牌在剩下不到 30 秒时先续期，剩余时间按本机收到响应的时刻算；
    - 续期：同一个标签页、同一个会话（按 `login_id`）一次只有一个；在锁下读出记录，不是本会话的就跟随它（`SessionChangedError`）；续期请求 8 秒超时；拿到响应后再读一次记录，`login_id` 变了就丢弃结果、不写回；只有 401 结束会话；
    - `renew(sent)`：已退出时给 `undefined`；另一个请求已经换到新令牌时直接用它；否则丢掉被拒的令牌、续期；
    - `signIn`、`signOut`、`endSession` 的写入都在锁下；退出在锁下读出最新的刷新令牌，调 `POST /api/v0/auth/logout`（8 秒，尽力而为），再删除记录；
    - `storage` 事件：记录被删除（本标签页未退出时）→ 退出；`login_id` 变了（包括本标签页未登录时记录出现）→ 丢掉访问令牌，以新会话登录；`login_id` 没变 → 不动。
  - 测试替身：`RecordingLock`（一次一个任务，`held` 告诉测试写入时锁是否被持有）；`FakeNerve`（`fetch` 把每个请求停住，直到测试 `answer`；记下方法、路径、`Authorization`、请求体、到达时刻、是否被放弃；`tokens(expiresIn)` 发出递增的 `at-n`、`rt-n`）；`until`、`track`、`settle`（假时钟上以 10 毫秒推进，20 秒的期限）。
- 使用者：Task 3、4 的测试；Task 7 的 `api-client.ts`、stores；Task 9 的 `useSession`。

**Tests:**（`token-manager.test.ts`，30 个，一个标签页，假时钟，假 nerve 按测试的顺序回答）
- `start`：没有记录时未登录、不请求；第一次续期 200、401；429（`Retry-After: 5`）、500、503 时保留记录、`unavailable`、到时自己重试；没有网络时按 1、2、4……30 秒退避；按钮立即重试；另一个标签页改了会话时不再重试。
- `refresh`：剩 30 秒之前用内存中的令牌，时长从收到时算；同一时刻的请求只续期一次；`login_id` 不变、记录一次写入；429、500、503、400、没有网络时保留会话、把错误交给调用方；退避期间不再请求；只有续期 401 结束会话；8 秒放弃续期（7999 毫秒时仍在等）；锁下读到别的登录的记录时不续期它；另一个标签页登录之后发出的请求得到新会话自己的续期，而不是旧会话那次（旧的那次以 `SessionChangedError` 结束）；401 之后续期一次，另一个请求已换到的令牌直接用；会话结束之后被拒的请求不再续期。
- 登录与退出：登录在锁下写入新的 `login_id`；每次登录的 `login_id` 都不同；退出在锁下用最新的刷新令牌；退出失败或超过 8 秒时照样在本地退出；重发被拒时在锁下结束会话；订阅者收到每一次变化。
- 每个写入 `nerve.auth` 的测试都核对写入时 `RecordingLock.held` 为真。

- [ ] **Step 1: 生成的类型直接导出**

`web/packages/api-client/package.json`（对 `f27434c` 的差异）：

```diff
--- a/web/packages/api-client/package.json
+++ b/web/packages/api-client/package.json
@@ -8,7 +8,7 @@
     ".": "./src/index.ts"
   },
   "scripts": {
-    "gen": "openapi-typescript ../../../api/dist/openapi.yaml --output src/schema.gen.ts",
+    "gen": "openapi-typescript ../../../api/dist/openapi.yaml --root-types --root-types-no-schema-prefix --output src/schema.gen.ts",
     "check:types": "tsc --noEmit",
     "check:lint": "node ../../../tools/lint-cap.mjs 0",
     "check:format": "oxfmt --check .",
```

Run: `make gen-web`
Expected: 只有 `web/packages/api-client/src/schema.gen.ts` 改变（`api/dist/openapi.yaml` 不变）：

| 生成的文件 | SHA-256 | 行数 |
|---|---|---|
| `web/packages/api-client/src/schema.gen.ts` | `6db7b4a9fa9c2aa05123db1b947a170a27de844a2c2ae7f6a8f6af60ad9d4ee3` | 911 |

`web/packages/api-client/src/index.ts`（完整内容）：

```ts
import createFetchClient, { type ClientOptions } from "openapi-fetch";

import type { paths } from "./schema.gen";

// The schemas' own names (User, Profile, Problem …): openapi-typescript's --root-types (M2 design 3.12).
export type * from "./schema.gen";

/**
 * Creates a client for the Nerve API. Paths, parameters, request bodies and
 * responses are typed from api/dist/openapi.yaml; error bodies are
 * problem+json, Problem.
 */
export function createClient(options?: ClientOptions) {
  return createFetchClient<paths>(options);
}

/** A client of the Nerve API, as createClient makes it. */
export type ApiClient = ReturnType<typeof createClient>;
```

- [ ] **Step 2: web 依赖 `@nerve/api-client`**

`web/apps/web/package.json`（对 `f27434c` 的差异）：

```diff
--- a/web/apps/web/package.json
+++ b/web/apps/web/package.json
@@ -23,6 +23,7 @@
     "@fontsource/material-symbols-rounded": "catalog:",
     "@headlessui/react": "catalog:",
     "@makeplane/propel": "catalog:",
+    "@nerve/api-client": "workspace:*",
     "@nerve/constants": "workspace:*",
     "@nerve/editor": "workspace:*",
     "@nerve/hooks": "workspace:*",
```

Run: `pnpm install`
Expected: `pnpm-lock.yaml` 的变化只有下面这一处：

`pnpm-lock.yaml`（对 `f27434c` 的差异）：

```diff
--- a/pnpm-lock.yaml
+++ b/pnpm-lock.yaml
@@ -392,6 +392,9 @@
       '@makeplane/propel':
         specifier: 'catalog:'
         version: 0.3.0(@date-fns/tz@1.4.1)(@types/react@19.2.17)(date-fns@4.1.0)(react-dom@19.2.8(react@19.2.8))(react@19.2.8)(tailwindcss@4.1.17)
+      '@nerve/api-client':
+        specifier: workspace:*
+        version: link:../../packages/api-client
       '@nerve/constants':
         specifier: workspace:*
         version: link:../../packages/constants
```

- [ ] **Step 3: 测试替身**

`web/apps/web/core/lib/auth/fake-browser.ts`（对 Task 1 版本的差异）：

```diff
--- a/web/apps/web/core/lib/auth/fake-browser.ts
+++ b/web/apps/web/core/lib/auth/fake-browser.ts
@@ -4,8 +4,11 @@
  */
 
 // Test doubles of the browser for the auth tests: one localStorage shared by several tabs, which tells
-// the other tabs about every change as the browser does (a storage event, after the writing task).
+// the other tabs about every change as the browser does (a storage event, after the writing task), and a
+// lock whose holder the tests can see.
 
+import type { RefreshLock } from "./refresh-lock";
+
 type StorageListener = (key: string | null, newValue: string | null) => void;
 
 /** The localStorage of one browser: every tab's view writes the same data. */
@@ -55,6 +58,25 @@
   }
 }
 
+/** A lock that runs one task at a time, in order, and tells whether a task holds it now. */
+export class RecordingLock implements RefreshLock {
+  held = false;
+  private queue: Promise<unknown> = Promise.resolve();
+
+  run<T>(task: () => Promise<T>): Promise<T> {
+    const turn = this.queue.then(async () => {
+      this.held = true;
+      try {
+        return await task();
+      } finally {
+        this.held = false;
+      }
+    });
+    this.queue = turn.catch(() => undefined);
+    return turn;
+  }
+}
+
 /** A promise the test resolves or rejects by hand. */
 export function gate<T = void>() {
   let open!: (value: T) => void;
```

`web/apps/web/core/lib/auth/fake-nerve.ts`（新文件）：

```ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// A test double of nerve's API behind fetch, for the auth tests: every request waits until the test
// answers it, so the tests set the order of events themselves.

import { createClient } from "@nerve/api-client";
import type { AuthTokens } from "@nerve/api-client";

/** A request the fake received and has not answered yet. */
export type Call = {
  method: string;
  path: string;
  authorization: string | null;
  body: unknown;
  /** When the request arrived, on the fake clock. */
  at: number;
  /** Whether the client gave the request up (its signal aborted). */
  aborted: () => boolean;
  answer: (response: Response) => void;
};

const BASE_URL = "http://nerve.test";

export class FakeNerve {
  readonly calls: Call[] = [];
  private issued = 0;

  /** The fetch of the client under test: reads the body as fetch does, parks the request until answered. */
  fetch = async (request: Request): Promise<Response> => {
    const text = await request.text();
    return new Promise<Response>((resolve, reject) => {
      const onAbort = () => reject(new DOMException("The operation was aborted.", "AbortError"));
      if (request.signal.aborted) return onAbort();
      request.signal.addEventListener("abort", onAbort);
      this.calls.push({
        method: request.method,
        path: new URL(request.url).pathname,
        authorization: request.headers.get("Authorization"),
        body: text === "" ? undefined : (JSON.parse(text) as unknown),
        at: Date.now(),
        aborted: () => request.signal.aborted,
        answer: resolve,
      });
    });
  };

  /** A client of this fake, as the app makes one. */
  client() {
    return createClient({ baseUrl: BASE_URL, fetch: this.fetch });
  }

  /** The requests to path so far. */
  to(path: string): Call[] {
    return this.calls.filter((c) => c.path === path);
  }

  /** The next pair of tokens, numbered: rt-1 with at-1, rt-2 with at-2 … */
  tokens(expiresIn = 900): AuthTokens {
    this.issued++;
    return {
      token_type: "Bearer",
      access_token: `at-${this.issued}`,
      access_token_expires_in: expiresIn,
      refresh_token: `rt-${this.issued}`,
      refresh_token_expires_at: "2026-10-27T00:00:00Z",
    };
  }
}

export function json(status: number, body: unknown, headers: Record<string, string> = {}): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": status < 400 ? "application/json" : "application/problem+json", ...headers },
  });
}

export function problem(status: number, code: string, headers: Record<string, string> = {}): Response {
  return json(status, { status, code, title: "", detail: code }, headers);
}

export const noContent = () => new Response(null, { status: 204 });
```

`web/apps/web/core/lib/auth/fake-time.ts`（新文件）：

```ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// Waiting under vitest's fake timers, for the auth tests: time moves only when a test moves it, and every
// wait has a deadline in fake time, so a test that would hang fails instead.

import { vi } from "vitest";

/** Moves the fake time on in 10 ms steps until done() holds; fails after `deadline` ms of fake time. */
export async function until(done: () => boolean, what: string, deadline = 20_000): Promise<void> {
  if (done()) return;
  if (deadline <= 0) throw new Error(`timed out waiting for ${what}`);
  await vi.advanceTimersByTimeAsync(10);
  return until(done, what, deadline - 10);
}

/** Records what a promise resolves to or throws, so a test can look without awaiting it. */
export function track<T>(promise: Promise<T>) {
  const result: { settled: boolean; value?: T; error?: unknown } = { settled: false };
  promise.then(
    (value) => Object.assign(result, { settled: true, value }),
    (error: unknown) => Object.assign(result, { settled: true, error })
  );
  return result;
}

/** What a promise resolves to or throws, once it settles; fails, instead of hanging, when it never does. */
export async function settle<T>(promise: Promise<T>, what: string) {
  const result = track(promise);
  await until(() => result.settled, what);
  return result;
}
```

- [ ] **Step 4: 令牌管理器**

`web/apps/web/core/lib/auth/token-manager.ts`（新文件）：

```ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ApiClient, AuthTokens } from "@nerve/api-client";
import type { RefreshLock } from "./refresh-lock";

// The browser's tokens (M2 design 7.1): the access token lives in memory only; localStorage keeps one
// record, nerve.auth = {refresh_token, login_id}, written in one piece and only under the refresh lock.
// login_id is new at every sign-in and kept by refreshes, so a tab can tell another tab's refresh (same
// login_id) from another tab's sign-in (a new one), which may be another account.

/** The localStorage key of the session's record. */
export const AUTH_KEY = "nerve.auth";
/** An access token closer than this to its end is refreshed before it is used. */
const REFRESH_MARGIN_MS = 30_000;
/** How long a refresh or a logout may take: longer than nerve's own 4 s + 2 s (M2 design 3.5). */
const REQUEST_TIMEOUT_MS = 8_000;
/** The longest wait between two refreshes that failed for a passing reason. */
const MAX_BACKOFF_MS = 30_000;

type AuthRecord = { refresh_token: string; login_id: string };

/**
 * starting: the first refresh is under way; signed-in: loginId's record is in use; signed-out: no record;
 * unavailable: the first refresh failed for a passing reason (429, 5xx, no network), the record is kept and
 * the refresh is tried again at retryAt.
 */
type SessionStatus = "starting" | "signed-in" | "signed-out" | "unavailable";
export type SessionState = Readonly<{ status: SessionStatus; loginId?: string; retryAt?: number }>;

/** The session cannot be used for now: a refresh failed for a passing reason; try again from retryAt. */
export class SessionUnavailableError extends Error {
  constructor(readonly retryAt: number) {
    super("The session is unavailable for now.");
    this.name = "SessionUnavailableError";
  }
}

/** Another tab signed in or out: this tab follows it, and the request it meant for the old session stops. */
export class SessionChangedError extends Error {
  constructor() {
    super("Another tab changed the session.");
    this.name = "SessionChangedError";
  }
}

export type TokenManagerDeps = {
  storage: Pick<Storage, "getItem" | "setItem" | "removeItem">;
  lock: RefreshLock;
  /** A client without the auth middleware: a refresh must not wait on itself. */
  client: ApiClient;
  now: () => number;
  /** n random bytes as hexadecimal (crypto.getRandomValues: non-secure contexts have it too). */
  randomHex: (bytes: number) => string;
};

type Outcome =
  | { kind: "tokens"; tokens: AuthTokens; receivedAt: number }
  | { kind: "unauthorized" }
  | { kind: "unavailable"; retryAfterMs?: number };

export class TokenManager {
  #state: SessionState = { status: "starting" };
  #listeners = new Set<() => void>();
  #access: { token: string; expiresAt: number } | undefined;
  #refreshing: { loginId: string | undefined; promise: Promise<string | undefined> } | undefined;
  #failures = 0;
  #retryAt = 0;
  #retryTimer: ReturnType<typeof setTimeout> | undefined;
  #started: Promise<void> | undefined;

  constructor(private readonly deps: TokenManagerDeps) {}

  /** The session as the tab shows it; a new object on every change. */
  get state(): SessionState {
    return this.#state;
  }

  /** Calls listener after every change of state; returns the unsubscribe. */
  subscribe = (listener: () => void): (() => void) => {
    this.#listeners.add(listener);
    return () => {
      this.#listeners.delete(listener);
    };
  };

  /**
   * Decides the session when the app starts, once. Without a record the tab is signed out and asks
   * nothing, so the sign-in page makes no request that fails (S2); with one, the first refresh decides.
   */
  start(): Promise<void> {
    this.#started ??= this.#start();
    return this.#started;
  }

  /** Tries the first refresh again now, e.g. from the "try again" button. */
  retry = async (): Promise<void> => {
    clearTimeout(this.#retryTimer);
    this.#retryAt = 0;
    await this.#firstRefresh();
  };

  /**
   * The access token for a request: the one in memory while more than 30 s of it are left, else a
   * refreshed one; undefined when signed out. The time left is counted on this computer's clock from when
   * the token arrived, so a wrong clock changes nothing (M2 design 7.1).
   */
  async accessToken(): Promise<string | undefined> {
    if (this.#state.status === "signed-out") return undefined;
    const access = this.#access;
    if (access !== undefined && access.expiresAt - this.deps.now() > REFRESH_MARGIN_MS) return access.token;
    return this.#refresh();
  }

  /**
   * A new access token after nerve refused `sent` with 401: the one another request got meanwhile, else a
   * refreshed one; undefined when the refresh ended the session, or it had ended already.
   */
  async renew(sent: string): Promise<string | undefined> {
    if (this.#state.status === "signed-out") return undefined;
    if (this.#access !== undefined && this.#access.token !== sent) return this.accessToken();
    this.#access = undefined;
    return this.#refresh();
  }

  /** Keeps the tokens of a sign-in or a sign-up, with a new login_id, under the lock (M2 design 7.3). */
  async signIn(tokens: AuthTokens): Promise<void> {
    const receivedAt = this.deps.now();
    await this.deps.lock.run(async () => {
      const record = { refresh_token: tokens.refresh_token, login_id: this.deps.randomHex(16) };
      this.deps.storage.setItem(AUTH_KEY, JSON.stringify(record));
      this.#keep(tokens, receivedAt);
      this.#set({ status: "signed-in", loginId: record.login_id });
    });
  }

  /**
   * Signs out: under the lock, so the refresh token handed over is the latest, logs out with it (best
   * effort) and removes the record (M2 design 7.1, review M7).
   */
  async signOut(): Promise<void> {
    await this.deps.lock.run(async () => {
      const record = this.#read();
      if (record !== undefined) {
        await this.#call("/api/v0/auth/logout", record.refresh_token);
        this.deps.storage.removeItem(AUTH_KEY);
      }
      this.#signedOut();
    });
  }

  /** Ends the session after nerve refused a request again with the refreshed token. */
  async endSession(): Promise<void> {
    await this.deps.lock.run(async () => {
      const record = this.#read();
      if (!this.#isOurs(record)) {
        this.#switchTo(record);
        return;
      }
      this.deps.storage.removeItem(AUTH_KEY);
      this.#signedOut();
    });
  }

  /**
   * Another tab changed nerve.auth (the storage event): removed, it signed out; a new login_id, it signed
   * in, maybe as another account; the same login_id, it only refreshed.
   */
  handleStorageChange(newValue: string | null): void {
    const record = parse(newValue);
    if (record === undefined ? this.#state.status !== "signed-out" : record.login_id !== this.#state.loginId) {
      this.#switchTo(record);
    }
  }

  async #start(): Promise<void> {
    const record = this.#read();
    if (record === undefined) {
      this.#set({ status: "signed-out" });
      return;
    }
    this.#set({ status: "starting", loginId: record.login_id });
    await this.#firstRefresh();
  }

  /** The refresh that decides a starting or unavailable session; a passing failure makes it unavailable. */
  async #firstRefresh(): Promise<void> {
    try {
      await this.#refresh();
    } catch (error) {
      if (error instanceof SessionChangedError) return;
      if (!(error instanceof SessionUnavailableError)) throw error;
      this.#set({ status: "unavailable", loginId: this.#state.loginId, retryAt: error.retryAt });
      this.#retryTimer = setTimeout(() => void this.retry(), error.retryAt - this.deps.now());
    }
  }

  /**
   * One refresh at a time for the tab's session: a request made after another tab signed in waits for a
   * refresh of the new record, not for the old session's refresh, which ends in SessionChangedError.
   */
  #refresh(): Promise<string | undefined> {
    const loginId = this.#state.loginId;
    const current = this.#refreshing;
    if (current !== undefined && current.loginId === loginId) return current.promise;
    const promise = this.#refreshUnderLock().finally(() => {
      if (this.#refreshing?.promise === promise) this.#refreshing = undefined;
    });
    this.#refreshing = { loginId, promise };
    return promise;
  }

  async #refreshUnderLock(): Promise<string | undefined> {
    if (this.deps.now() < this.#retryAt) throw new SessionUnavailableError(this.#retryAt);
    return this.deps.lock.run(async () => {
      // Read under the lock: another tab may have refreshed, signed out or signed in meanwhile.
      const record = this.#read();
      if (!this.#isOurs(record)) this.#follow(record);
      const outcome = await this.#call("/api/v0/auth/refresh", record.refresh_token);
      // Read again before writing: without navigator.locks another tab can sign in while the lease is held.
      const now = this.#read();
      if (now?.login_id !== record.login_id) this.#follow(now);
      switch (outcome.kind) {
        case "tokens":
          this.deps.storage.setItem(
            AUTH_KEY,
            JSON.stringify({ refresh_token: outcome.tokens.refresh_token, login_id: record.login_id })
          );
          this.#keep(outcome.tokens, outcome.receivedAt);
          if (this.#state.status !== "signed-in") this.#set({ status: "signed-in", loginId: record.login_id });
          return outcome.tokens.access_token;
        case "unauthorized":
          // Only a 401 to a refresh ends the session.
          this.deps.storage.removeItem(AUTH_KEY);
          this.#signedOut();
          return undefined;
        case "unavailable":
          throw new SessionUnavailableError(this.#backOff(outcome.retryAfterMs));
      }
    });
  }

  /** POSTs refresh_token to path, with its own timeout; never throws. A logout's answer does not matter. */
  async #call(path: "/api/v0/auth/refresh" | "/api/v0/auth/logout", refreshToken: string): Promise<Outcome> {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);
    try {
      const { data, response } = await this.deps.client.POST(path, {
        body: { refresh_token: refreshToken },
        signal: controller.signal,
      });
      if (data !== undefined) return { kind: "tokens", tokens: data, receivedAt: this.deps.now() };
      if (response.status === 401) return { kind: "unauthorized" };
      const retryAfter = Number(response.headers.get("Retry-After"));
      return { kind: "unavailable", retryAfterMs: retryAfter > 0 ? retryAfter * 1000 : undefined };
    } catch {
      // No network, or the timeout.
      return { kind: "unavailable" };
    } finally {
      clearTimeout(timer);
    }
  }

  /** Waits Retry-After, else 1, 2, 4 … up to 30 s after each failure in a row; returns when to try again. */
  #backOff(retryAfterMs: number | undefined): number {
    const delay = retryAfterMs ?? Math.min(1000 * 2 ** this.#failures, MAX_BACKOFF_MS);
    this.#failures++;
    this.#retryAt = this.deps.now() + delay;
    return this.#retryAt;
  }

  #keep(tokens: AuthTokens, receivedAt: number): void {
    this.#access = { token: tokens.access_token, expiresAt: receivedAt + tokens.access_token_expires_in * 1000 };
    this.#failures = 0;
    this.#retryAt = 0;
  }

  #isOurs(record: AuthRecord | undefined): record is AuthRecord {
    return record !== undefined && record.login_id === this.#state.loginId;
  }

  /** Follows another tab's change of the record, and stops the request that was meant for the old session. */
  #follow(record: AuthRecord | undefined): never {
    this.#switchTo(record);
    throw new SessionChangedError();
  }

  /** Signed out when the record is gone, else the record's session, whose access token comes by a refresh. */
  #switchTo(record: AuthRecord | undefined): void {
    if (record === undefined) {
      this.#signedOut();
      return;
    }
    this.#access = undefined;
    this.#set({ status: "signed-in", loginId: record.login_id });
  }

  #signedOut(): void {
    this.#access = undefined;
    this.#set({ status: "signed-out" });
  }

  #set(state: SessionState): void {
    if (state.status !== "unavailable") clearTimeout(this.#retryTimer);
    this.#state = state;
    for (const listener of this.#listeners) listener();
  }

  #read(): AuthRecord | undefined {
    return parse(this.deps.storage.getItem(AUTH_KEY));
  }
}

function parse(value: string | null): AuthRecord | undefined {
  try {
    const record = JSON.parse(value ?? "null") as Partial<AuthRecord> | null;
    return typeof record?.refresh_token === "string" && typeof record.login_id === "string"
      ? { refresh_token: record.refresh_token, login_id: record.login_id }
      : undefined;
  } catch {
    return undefined;
  }
}
```

- [ ] **Step 5: 测试**

`web/apps/web/core/lib/auth/token-manager.test.ts`（新文件）：

```ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { RecordingLock, SharedStorage } from "./fake-browser";
import { FakeNerve, json, noContent, problem } from "./fake-nerve";
import { settle, track, until } from "./fake-time";
import { AUTH_KEY, SessionChangedError, SessionUnavailableError, TokenManager } from "./token-manager";

// One tab of the token manager (M2 design 7.1, 9.4), with fake timers: nerve answers when the test says
// so, and time moves only when the test moves it. The tabs' coordination is token-manager.tabs.test.ts.

const REFRESH = "/api/v0/auth/refresh";
const LOGOUT = "/api/v0/auth/logout";

function setUp(record?: { refresh_token: string; login_id: string }) {
  const storage = new SharedStorage();
  if (record) storage.data.set(AUTH_KEY, JSON.stringify(record));
  const nerve = new FakeNerve();
  const lock = new RecordingLock();
  const view = storage.tab("A");
  // Every write of the record must happen while the lock is held (M2 design 7.1, review R4).
  const writes: { key: string; held: boolean }[] = [];
  const tm = new TokenManager({
    storage: {
      getItem: view.getItem,
      setItem: (key, value) => {
        writes.push({ key, held: lock.held });
        view.setItem(key, value);
      },
      removeItem: (key) => {
        writes.push({ key, held: lock.held });
        view.removeItem(key);
      },
    },
    lock,
    client: nerve.client(),
    now: () => Date.now(),
    randomHex: (bytes) => `${bytes}`.padStart(bytes * 2, "a"),
  });
  const stored = () => JSON.parse(storage.data.get(AUTH_KEY) ?? "null") as unknown;
  return { storage, nerve, lock, tm, writes, stored };
}

const loginId = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa16";
const other = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb";

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("start", () => {
  it("is signed out without a record, and asks nerve nothing", async () => {
    const { tm, nerve } = setUp();
    await tm.start();
    expect(tm.state).toEqual({ status: "signed-out" });
    expect(await settle(tm.accessToken(), "the token")).toEqual({ settled: true, value: undefined });
    expect(nerve.calls).toEqual([]);
  });

  it("is signed in when the first refresh succeeds", async () => {
    const { tm, nerve, stored } = setUp({ refresh_token: "rt-0", login_id: loginId });
    const started = tm.start();
    expect(tm.state).toEqual({ status: "starting", loginId });
    await until(() => nerve.to(REFRESH).length === 1, "the refresh");
    expect(nerve.calls[0]?.body).toEqual({ refresh_token: "rt-0" });
    nerve.calls[0]?.answer(json(200, nerve.tokens()));
    await started;

    expect(tm.state).toEqual({ status: "signed-in", loginId });
    expect(stored()).toEqual({ refresh_token: "rt-1", login_id: loginId });
    expect((await settle(tm.accessToken(), "the token")).value).toBe("at-1");
    expect(nerve.calls).toHaveLength(1);
  });

  it("is signed out, the record gone, when the first refresh answers 401", async () => {
    const { tm, nerve, storage } = setUp({ refresh_token: "rt-0", login_id: loginId });
    const started = tm.start();
    await until(() => nerve.calls.length === 1, "the refresh");
    nerve.calls[0]?.answer(problem(401, "identity.refresh_token_invalid"));
    await started;

    expect(tm.state).toEqual({ status: "signed-out" });
    expect(storage.data.has(AUTH_KEY)).toBe(false);
  });

  it.each([
    ["429", () => problem(429, "rate_limited", { "Retry-After": "5" }), 5_000],
    ["500", () => problem(500, "internal_error"), 1_000],
    ["503", () => problem(503, "server_busy"), 1_000],
  ])("keeps the record and is unavailable on a %s, then tries again by itself", async (_, answer, delay) => {
    const { tm, nerve, stored } = setUp({ refresh_token: "rt-0", login_id: loginId });
    const started = tm.start();
    await until(() => nerve.calls.length === 1, "the refresh");
    nerve.calls[0]?.answer(answer());
    await started;

    const retryAt = Date.now() + delay;
    expect(tm.state).toEqual({ status: "unavailable", loginId, retryAt });
    expect(stored()).toEqual({ refresh_token: "rt-0", login_id: loginId });
    await vi.advanceTimersByTimeAsync(delay - 1);
    expect(nerve.calls).toHaveLength(1);
    await vi.advanceTimersByTimeAsync(1);
    await until(() => nerve.calls.length === 2, "the second refresh");
    expect(nerve.calls[1]?.body).toEqual({ refresh_token: "rt-0" });
    nerve.calls[1]?.answer(json(200, nerve.tokens()));
    await until(() => tm.state.status === "signed-in", "the session");
  });

  it("keeps the record and is unavailable without a network, backing off 1, 2, 4 … up to 30 s", async () => {
    const { tm, nerve, stored } = setUp({ refresh_token: "rt-0", login_id: loginId });
    void tm.start();
    // Fails refreshes n … 7 as they come, and returns how long the manager waited after each.
    const fail = async (n: number, waits: number[] = []): Promise<number[]> => {
      if (n > 7) return waits;
      await until(() => nerve.calls.length === n, `refresh ${n}`);
      const before = tm.state.retryAt;
      const failedAt = Date.now();
      nerve.calls[n - 1]?.answer(Response.error());
      await until(() => tm.state.status === "unavailable" && tm.state.retryAt !== before, "unavailable");
      const wait = (tm.state.retryAt ?? 0) - failedAt;
      await vi.advanceTimersByTimeAsync(wait);
      return fail(n + 1, [...waits, wait]);
    };
    expect(await fail(1)).toEqual([1_000, 2_000, 4_000, 8_000, 16_000, 30_000, 30_000]);
    expect(stored()).toEqual({ refresh_token: "rt-0", login_id: loginId });

    // A success starts the count again: the next failure waits 1 s.
    await until(() => nerve.calls.length === 8, "refresh 8");
    nerve.calls[7]?.answer(json(200, nerve.tokens()));
    await until(() => tm.state.status === "signed-in", "the session");
    const renewed = track(tm.renew("at-1"));
    await until(() => nerve.calls.length === 9, "refresh 9");
    const failedAt = Date.now();
    nerve.calls[8]?.answer(Response.error());
    await until(() => renewed.settled, "the refresh's end");
    expect(renewed.error).toEqual(new SessionUnavailableError(failedAt + 1_000));
  });

  it("tries again at once from the button", async () => {
    const { tm, nerve } = setUp({ refresh_token: "rt-0", login_id: loginId });
    void tm.start();
    await until(() => nerve.calls.length === 1, "the refresh");
    nerve.calls[0]?.answer(problem(503, "server_busy", { "Retry-After": "20" }));
    await until(() => tm.state.status === "unavailable", "unavailable");

    void tm.retry();
    await until(() => nerve.calls.length === 2, "the retry");
    nerve.calls[1]?.answer(json(200, nerve.tokens()));
    await until(() => tm.state.status === "signed-in", "the session");
    // The timer of the automatic retry is gone with the unavailability.
    await vi.advanceTimersByTimeAsync(30_000);
    expect(nerve.calls).toHaveLength(2);
  });

  it("stops trying again when another tab changes the session meanwhile", async () => {
    const { tm, nerve, storage } = setUp({ refresh_token: "rt-0", login_id: loginId });
    storage.tab("A").onStorage((key, value) => {
      if (key === AUTH_KEY) tm.handleStorageChange(value);
    });
    void tm.start();
    await until(() => nerve.calls.length === 1, "the refresh");
    nerve.calls[0]?.answer(problem(503, "server_busy", { "Retry-After": "20" }));
    await until(() => tm.state.status === "unavailable", "unavailable");

    storage.write(AUTH_KEY, JSON.stringify({ refresh_token: "rt-y", login_id: other }));
    await until(() => tm.state.loginId === other, "the switch");
    expect(tm.state).toEqual({ status: "signed-in", loginId: other });
    await vi.advanceTimersByTimeAsync(30_000);
    expect(nerve.calls).toHaveLength(1);
  });
});

describe("refresh", () => {
  async function signedIn(expiresIn = 900) {
    const s = setUp({ refresh_token: "rt-0", login_id: loginId });
    const started = s.tm.start();
    await until(() => s.nerve.calls.length === 1, "the first refresh");
    s.nerve.calls[0]?.answer(json(200, s.nerve.tokens(expiresIn)));
    await started;
    return s;
  }

  it("uses the access token until 30 s before its end, counted from when it arrived", async () => {
    const s = setUp({ refresh_token: "rt-0", login_id: loginId });
    const started = s.tm.start();
    await until(() => s.nerve.calls.length === 1, "the first refresh");
    // The answer takes 5 s; the token's 900 s count from its arrival, not from the request.
    await vi.advanceTimersByTimeAsync(5_000);
    const arrived = Date.now();
    // The token is not a JWT: the manager never reads it, nor nerve's clock (M2 design 7.1).
    s.nerve.calls[0]?.answer(json(200, { ...s.nerve.tokens(900), access_token: "opaque" }));
    await started;

    await vi.advanceTimersByTimeAsync(arrived + 870_000 - 1 - Date.now());
    expect((await settle(s.tm.accessToken(), "the token")).value).toBe("opaque");
    expect(s.nerve.calls).toHaveLength(1);

    await vi.advanceTimersByTimeAsync(1);
    const next = track(s.tm.accessToken());
    await until(() => s.nerve.calls.length === 2, "the refresh");
    expect(s.nerve.calls[1]?.body).toEqual({ refresh_token: "rt-1" });
    s.nerve.calls[1]?.answer(json(200, s.nerve.tokens()));
    await until(() => next.settled, "the token");
    expect(next.value).toBe("at-2");
  });

  it("refreshes once for the requests of a tab that need a token at the same time", async () => {
    const s = await signedIn(20);
    const tokens = [track(s.tm.accessToken()), track(s.tm.accessToken()), track(s.tm.accessToken())];
    await until(() => s.nerve.calls.length === 2, "the refresh");
    await vi.advanceTimersByTimeAsync(100);
    expect(s.nerve.to(REFRESH)).toHaveLength(2);
    s.nerve.calls[1]?.answer(json(200, s.nerve.tokens()));
    await until(() => tokens.every((t) => t.settled), "the tokens");
    expect(tokens.map((t) => t.value)).toEqual(["at-2", "at-2", "at-2"]);
  });

  it("keeps login_id and writes the record in one piece", async () => {
    const s = await signedIn(20);
    const next = track(s.tm.accessToken());
    await until(() => s.nerve.calls.length === 2, "the refresh");
    s.nerve.calls[1]?.answer(json(200, s.nerve.tokens()));
    await until(() => next.settled, "the token");
    expect(s.stored()).toEqual({ refresh_token: "rt-2", login_id: loginId });
    expect(s.writes.filter((w) => w.key === AUTH_KEY)).toHaveLength(2);
  });

  it.each([
    ["429", () => problem(429, "rate_limited")],
    ["500", () => problem(500, "internal_error")],
    ["503", () => problem(503, "server_busy")],
    ["a 400", () => problem(400, "bad_request")],
    ["no network", () => Response.error()],
  ])("keeps the session on %s, and gives the caller the error", async (_, answer) => {
    const s = await signedIn(20);
    const next = track(s.tm.accessToken());
    await until(() => s.nerve.calls.length === 2, "the refresh");
    s.nerve.calls[1]?.answer(answer());
    await until(() => next.settled, "the refresh's end");

    expect(next.error).toBeInstanceOf(SessionUnavailableError);
    expect(s.tm.state).toEqual({ status: "signed-in", loginId });
    expect(s.stored()).toEqual({ refresh_token: "rt-1", login_id: loginId });
  });

  it("does not ask again before the back-off ends", async () => {
    const s = await signedIn(20);
    const first = track(s.tm.accessToken());
    await until(() => s.nerve.calls.length === 2, "the refresh");
    const answeredAt = Date.now();
    s.nerve.calls[1]?.answer(problem(503, "server_busy", { "Retry-After": "3" }));
    await until(() => first.settled, "the refresh's end");
    expect(first.error).toEqual(new SessionUnavailableError(answeredAt + 3_000));

    await vi.advanceTimersByTimeAsync(answeredAt + 3_000 - 1 - Date.now());
    const early = track(s.tm.accessToken());
    await vi.advanceTimersByTimeAsync(0);
    expect(early.error).toBeInstanceOf(SessionUnavailableError);
    expect(s.nerve.calls).toHaveLength(2);
    await vi.advanceTimersByTimeAsync(1);
    const again = track(s.tm.accessToken());
    await until(() => s.nerve.calls.length === 3, "the next refresh");
    s.nerve.calls[2]?.answer(json(200, s.nerve.tokens()));
    await until(() => again.settled, "the token");
    expect(again.value).toBe("at-2");
  });

  it("ends the session only when the refresh answers 401", async () => {
    const s = await signedIn(20);
    const next = track(s.tm.accessToken());
    await until(() => s.nerve.calls.length === 2, "the refresh");
    s.nerve.calls[1]?.answer(problem(401, "identity.refresh_token_invalid"));
    await until(() => next.settled, "the refresh's end");

    expect(next.value).toBeUndefined();
    expect(next.error).toBeUndefined();
    expect(s.tm.state).toEqual({ status: "signed-out" });
    expect(s.storage.data.has(AUTH_KEY)).toBe(false);
  });

  it("gives up a refresh after 8 s, keeping the session", async () => {
    const s = await signedIn(20);
    const next = track(s.tm.accessToken());
    await until(() => s.nerve.calls.length === 2, "the refresh");
    const call = s.nerve.calls[1];

    await vi.advanceTimersByTimeAsync((call?.at ?? 0) + 7_999 - Date.now());
    expect(next.settled).toBe(false);
    expect(call?.aborted()).toBe(false);
    await vi.advanceTimersByTimeAsync(1);
    expect(call?.aborted()).toBe(true);
    await until(() => next.settled, "the refresh's end");
    expect(next.error).toBeInstanceOf(SessionUnavailableError);
    expect(s.tm.state.status).toBe("signed-in");
    expect(s.stored()).toEqual({ refresh_token: "rt-1", login_id: loginId });
  });

  it("does not refresh a record of another sign-in it has not heard of yet", async () => {
    const s = await signedIn(20);
    // Another tab signed in; its storage event has not reached this tab when the lock is granted.
    s.storage.data.set(AUTH_KEY, JSON.stringify({ refresh_token: "rt-y", login_id: other }));
    const next = await settle(s.tm.accessToken(), "the token");

    expect(next.error).toBeInstanceOf(SessionChangedError);
    expect(s.nerve.calls).toHaveLength(1);
    expect(s.tm.state).toEqual({ status: "signed-in", loginId: other });
    expect(s.stored()).toEqual({ refresh_token: "rt-y", login_id: other });
  });

  it("gives a request made after another tab's sign-in a refresh of its own, not the old session's", async () => {
    const s = await signedIn(20);
    s.storage.tab("A").onStorage((key, value) => {
      if (key === AUTH_KEY) s.tm.handleStorageChange(value);
    });
    // The old session's refresh is out when another tab's sign-in lands (the lease's race window).
    const old = track(s.tm.accessToken());
    await until(() => s.nerve.calls.length === 2, "the old session's refresh");
    s.storage.write(AUTH_KEY, JSON.stringify({ refresh_token: "rt-y", login_id: other }));
    await until(() => s.tm.state.loginId === other, "the switch");
    const mine = track(s.tm.accessToken());
    s.nerve.calls[1]?.answer(json(200, s.nerve.tokens()));
    await until(() => old.settled, "the old session's request");
    expect(old.error).toBeInstanceOf(SessionChangedError);

    // The new session's request refreshes the new record once the old refresh has let the lock go.
    await until(() => s.nerve.calls.length === 3, "the new session's refresh");
    expect(s.nerve.calls[2]?.body).toEqual({ refresh_token: "rt-y" });
    // Another request meanwhile shares it: the old refresh's end does not start a second one.
    const also = track(s.tm.accessToken());
    s.nerve.calls[2]?.answer(json(200, s.nerve.tokens()));
    await until(() => mine.settled && also.settled, "the new session's tokens");
    expect([mine.value, also.value]).toEqual(["at-3", "at-3"]);
    expect(s.nerve.to(REFRESH)).toHaveLength(3);
    expect(s.stored()).toEqual({ refresh_token: "rt-3", login_id: other });
  });

  it("renews once after a 401, with the token another request got meanwhile if there is one", async () => {
    const s = await signedIn();
    const renewed = track(s.tm.renew("at-1"));
    await until(() => s.nerve.calls.length === 2, "the refresh");
    s.nerve.calls[1]?.answer(json(200, s.nerve.tokens()));
    await until(() => renewed.settled, "the token");
    expect(renewed.value).toBe("at-2");

    // A second request refused with the old token takes the new one, without another refresh.
    expect((await settle(s.tm.renew("at-1"), "the token")).value).toBe("at-2");
    expect(s.nerve.to(REFRESH)).toHaveLength(2);
  });

  it("gives a request refused after the session ended no token, asking nerve nothing", async () => {
    const s = await signedIn();
    // Two requests with at-1 were refused; the first one's refresh answers 401 and ends the session.
    const first = track(s.tm.renew("at-1"));
    await until(() => s.nerve.calls.length === 2, "the refresh");
    s.nerve.calls[1]?.answer(problem(401, "identity.refresh_token_invalid"));
    await until(() => first.settled, "the refresh's end");
    expect(first.value).toBeUndefined();
    expect(s.tm.state).toEqual({ status: "signed-out" });

    expect(await settle(s.tm.renew("at-1"), "the token")).toEqual({ settled: true, value: undefined });
    expect(s.nerve.calls).toHaveLength(2);
  });
});

describe("sign-in and sign-out", () => {
  it("keeps the tokens of a sign-in with a new login_id, under the lock", async () => {
    const { tm, nerve, stored, writes } = setUp();
    await tm.start();
    await tm.signIn(nerve.tokens());
    const first = stored() as { login_id: string };
    expect(first).toEqual({ refresh_token: "rt-1", login_id: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa16" });
    expect(tm.state).toEqual({ status: "signed-in", loginId: first.login_id });
    expect((await settle(tm.accessToken(), "the token")).value).toBe("at-1");
    expect(nerve.calls).toEqual([]);
    expect(writes).toEqual([{ key: AUTH_KEY, held: true }]);
  });

  it("makes a new login_id at every sign-in", async () => {
    const storage = new SharedStorage();
    const nerve = new FakeNerve();
    let n = 0;
    const tm = new TokenManager({
      storage: storage.tab("A"),
      lock: new RecordingLock(),
      client: nerve.client(),
      now: () => Date.now(),
      randomHex: (bytes) => `${++n}`.padStart(bytes * 2, "0"),
    });
    await tm.start();
    await tm.signIn(nerve.tokens());
    const first = tm.state.loginId;
    await tm.signIn(nerve.tokens());
    expect(first).toMatch(/^[0-9a-f]{32}$/);
    expect(tm.state.loginId).toMatch(/^[0-9a-f]{32}$/);
    expect(tm.state.loginId).not.toBe(first);
    expect(JSON.parse(storage.data.get(AUTH_KEY) ?? "null")).toEqual({
      refresh_token: "rt-2",
      login_id: tm.state.loginId,
    });
  });

  it("signs out with the latest refresh token, under the lock, and forgets the tokens", async () => {
    const { tm, nerve, storage, writes } = setUp();
    await tm.start();
    await tm.signIn(nerve.tokens());
    const out = track(tm.signOut());
    await until(() => nerve.calls.length === 1, "the logout");
    expect(nerve.calls[0]).toMatchObject({ path: LOGOUT, body: { refresh_token: "rt-1" } });
    nerve.calls[0]?.answer(noContent());
    await until(() => out.settled, "the sign-out");

    expect(tm.state).toEqual({ status: "signed-out" });
    expect(storage.data.has(AUTH_KEY)).toBe(false);
    expect(await settle(tm.accessToken(), "the token")).toEqual({ settled: true, value: undefined });
    expect(nerve.calls).toHaveLength(1);
    expect(writes).toEqual([
      { key: AUTH_KEY, held: true },
      { key: AUTH_KEY, held: true },
    ]);
  });

  it("signs out locally when the logout fails or takes over 8 s", async () => {
    const { tm, nerve, storage } = setUp();
    await tm.start();
    await tm.signIn(nerve.tokens());
    const out = track(tm.signOut());
    await until(() => nerve.calls.length === 1, "the logout");
    await vi.advanceTimersByTimeAsync((nerve.calls[0]?.at ?? 0) + 7_999 - Date.now());
    expect(nerve.calls[0]?.aborted()).toBe(false);
    expect(tm.state.status).toBe("signed-in");
    await vi.advanceTimersByTimeAsync(1);
    await until(() => out.settled, "the sign-out");
    expect(nerve.calls[0]?.aborted()).toBe(true);
    expect(out.error).toBeUndefined();
    expect(tm.state).toEqual({ status: "signed-out" });
    expect(storage.data.has(AUTH_KEY)).toBe(false);
  });

  it("ends the session after a refused replay, under the lock", async () => {
    const { tm, nerve, storage, writes } = setUp();
    await tm.start();
    await tm.signIn(nerve.tokens());
    await tm.endSession();
    expect(tm.state).toEqual({ status: "signed-out" });
    expect(storage.data.has(AUTH_KEY)).toBe(false);
    expect(writes).toEqual([
      { key: AUTH_KEY, held: true },
      { key: AUTH_KEY, held: true },
    ]);
    expect(nerve.calls).toEqual([]);
  });

  it("tells its subscribers of every change of state", async () => {
    const { tm, nerve } = setUp();
    const seen: string[] = [];
    const unsubscribe = tm.subscribe(() => seen.push(tm.state.status));
    await tm.start();
    await tm.signIn(nerve.tokens());
    unsubscribe();
    const out = track(tm.signOut());
    await until(() => nerve.calls.length === 1, "the logout");
    nerve.calls[0]?.answer(noContent());
    await until(() => out.settled, "the sign-out");
    expect(seen).toEqual(["signed-out", "signed-in"]);
  });
});
```

- [ ] **Step 6: 检查**

Run: `pnpm -C web/apps/web exec vitest run core/lib/auth`
Expected: 2 个文件、39 个测试通过。

Run: `make lint-web`、`make knip`、`make test-web`
Expected: 都通过。

- [ ] **Step 7: 提交**

```bash
git add web/packages/api-client web/apps/web/package.json pnpm-lock.yaml web/apps/web/core/lib/auth
```
```bash
git commit -m "feat(M2/P4): the token manager

The access token in memory; nerve.auth = {refresh_token, login_id} written
in one piece under the refresh lock. The first refresh decides the
session: signed in, signed out, or unavailable with back-off. A refresh
takes 8 s at most, one at a time per session; it re-reads the record
before writing back and drops its result when login_id changed. Only a
refresh answered 401 ends the session. @nerve/api-client exports the
generated types by name (--root-types), and web depends on it.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

- [ ] **Step 8: 生成物已提交**

Run: `make gen-check`
Expected: 没有输出差异，以 0 退出。

**Done when:** 39 个测试通过；spec 附录 A 列出的变异（去掉锁、去掉写回前的 `login_id` 核对、429 或 5xx 结束会话、去掉 8 秒超时……）各让至少一个测试失败。

---

### Task 3: 令牌管理器的多标签页测试

**Files:**
- Create: `web/apps/web/core/lib/auth/token-manager.tabs.test.ts`

**Interfaces:**
- 只有测试。几个标签页共用一份 `SharedStorage` 和一个 `FakeNerve`，每个标签页有自己的 `TokenManager`，`storage` 事件按浏览器的方式送到其他标签页；锁分两种各跑一遍：一个所有标签页共用的队列（`navigator.locks`），或每个标签页自己的 `leaseLock`（没有 `navigator.locks` 时）。

**Tests:**（`token-manager.tabs.test.ts`，10 个 × 两种锁 = 20 个）
- 两个标签页同时启动、令牌过期后同时要令牌：续期一个接一个，每次用另一个留下的刷新令牌（发出的依次是 `rt-0` 到 `rt-3`，从不重复，一次续期在途时 2 秒内没有第二个）；
- 登录要等另一个标签页的续期结束，续期之后不会把旧会话写回去；
- 续期在途中记录被换成另一个账户时（租约不是原子的，这可能发生），不论 nerve 回答新令牌还是 401，都丢弃结果、不写回，标签页以新账户继续，下一次续期用新账户的刷新令牌；
- 一个标签页退出，所有标签页退出；退出交出的是另一个标签页刚写入的刷新令牌；
- 另一个标签页登录时，未登录的标签页随之登录；另一个标签页以别的账户登录时，所有标签页切换过去；另一个标签页只是续期时，本标签页不动；
- 续期没有回应时，8 秒放开锁，早于租约的 10 秒。

- [ ] **Step 1: 测试**

`web/apps/web/core/lib/auth/token-manager.tabs.test.ts`（新文件）：

```ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { RecordingLock, SharedStorage } from "./fake-browser";
import { FakeNerve, json, noContent, problem } from "./fake-nerve";
import { track, until } from "./fake-time";
import { leaseLock, webLock } from "./refresh-lock";
import { AUTH_KEY, SessionChangedError, SessionUnavailableError, TokenManager } from "./token-manager";

// Several tabs of one browser (M2 design 7.1, 9.4): one localStorage, one nerve, and the tabs' lock,
// either navigator.locks or, where the page has none (plain HTTP on a LAN address), the lease. Each tab
// hears the others' writes of nerve.auth as the browser's storage event. Time is fake throughout.

const REFRESH = "/api/v0/auth/refresh";
const LOGOUT = "/api/v0/auth/logout";
const X = "0000000000000000000000000000000a";
const Y = "0000000000000000000000000000000b";

type Kind = "navigator.locks" | "the lease";

function browser(kind: Kind) {
  const storage = new SharedStorage();
  const nerve = new FakeNerve();
  // navigator.locks: one queue for the whole browser, as the browser keeps it.
  const shared = new RecordingLock();
  const locks = {
    request: ((_name: string, task: () => Promise<unknown>) => shared.run(task)) as LockManager["request"],
  };
  let logins = 0;
  function tab(id: string) {
    const view = storage.tab(id);
    const lock =
      kind === "navigator.locks"
        ? webLock(locks)
        : leaseLock({ storage: view, onStorage: view.onStorage, now: () => Date.now(), tabId: id });
    const tm = new TokenManager({
      storage: view,
      lock,
      client: nerve.client(),
      now: () => Date.now(),
      randomHex: (bytes) => `${++logins}`.padStart(bytes * 2, "c"),
    });
    const changes: string[] = [];
    tm.subscribe(() => changes.push(`${tm.state.status} ${tm.state.loginId ?? "-"}`));
    view.onStorage((key, value) => {
      if (key === AUTH_KEY) tm.handleStorageChange(value);
    });
    return { tm, changes };
  }
  const stored = () => JSON.parse(storage.data.get(AUTH_KEY) ?? "null") as unknown;
  return { storage, nerve, tab, stored };
}

/** Tabs that share the record of X, each with an access token that lasts expiresIn seconds. */
async function signedIn(b: ReturnType<typeof browser>, ids: string[], expiresIn = 20) {
  b.storage.data.set(AUTH_KEY, JSON.stringify({ refresh_token: "rt-0", login_id: X }));
  const tabs = ids.map((id) => b.tab(id));
  const started = tabs.map((t) => track(t.tm.start()));
  // The tabs' first refreshes, one after the other.
  const answer = async (n: number): Promise<void> => {
    if (n > tabs.length) return;
    await until(() => b.nerve.to(REFRESH).length === n, `refresh ${n}`);
    b.nerve.to(REFRESH)[n - 1]?.answer(json(200, b.nerve.tokens(expiresIn)));
    return answer(n + 1);
  };
  await answer(1);
  await until(() => started.every((s) => s.settled), "the starts");
  b.nerve.calls.length = 0;
  for (const t of tabs) t.changes.length = 0;
  return tabs;
}

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe.each<Kind>(["navigator.locks", "the lease"])("tabs with %s", (kind) => {
  it("refresh one at a time, each with the refresh token the other left", async () => {
    const b = browser(kind);
    b.storage.data.set(AUTH_KEY, JSON.stringify({ refresh_token: "rt-0", login_id: X }));
    const [a, c] = [b.tab("A"), b.tab("C")];
    const starts = [track(a.tm.start()), track(c.tm.start())];

    await until(() => b.nerve.calls.length === 1, "the first refresh");
    await vi.advanceTimersByTimeAsync(2_000);
    expect(b.nerve.calls).toHaveLength(1);
    expect(b.nerve.calls[0]?.body).toEqual({ refresh_token: "rt-0" });
    b.nerve.calls[0]?.answer(json(200, b.nerve.tokens(20)));
    await until(() => b.nerve.calls.length === 2, "the second refresh");
    expect(b.nerve.calls[1]?.body).toEqual({ refresh_token: "rt-1" });
    b.nerve.calls[1]?.answer(json(200, b.nerve.tokens(20)));
    await until(() => starts.every((s) => s.settled), "the starts");
    expect([a.tm.state, c.tm.state]).toEqual([
      { status: "signed-in", loginId: X },
      { status: "signed-in", loginId: X },
    ]);

    // Both tokens run out; both tabs need one at once.
    await vi.advanceTimersByTimeAsync(10_000);
    const tokens = [track(a.tm.accessToken()), track(c.tm.accessToken())];
    await until(() => b.nerve.calls.length === 3, "a refresh");
    await vi.advanceTimersByTimeAsync(2_000);
    expect(b.nerve.calls).toHaveLength(3);
    expect(b.nerve.calls[2]?.body).toEqual({ refresh_token: "rt-2" });
    b.nerve.calls[2]?.answer(json(200, b.nerve.tokens(20)));
    await until(() => b.nerve.calls.length === 4, "the other refresh");
    expect(b.nerve.calls[3]?.body).toEqual({ refresh_token: "rt-3" });
    b.nerve.calls[3]?.answer(json(200, b.nerve.tokens(20)));
    await until(() => tokens.every((t) => t.settled), "the tokens");
    expect(new Set(tokens.map((t) => t.value))).toEqual(new Set(["at-3", "at-4"]));
    expect(b.stored()).toEqual({ refresh_token: "rt-4", login_id: X });
  });

  it("make a sign-in wait for another tab's refresh, which then leaves the new record alone", async () => {
    const b = browser(kind);
    const [a] = await signedIn(b, ["A"]);
    const other = b.tab("C");
    await vi.advanceTimersByTimeAsync(10_000);
    const token = track(a!.tm.accessToken());
    await until(() => b.nerve.calls.length === 1, "the refresh");

    const signIn = track(other.tm.signIn({ ...b.nerve.tokens(), refresh_token: "rt-y" }));
    await vi.advanceTimersByTimeAsync(2_000);
    expect(signIn.settled).toBe(false);
    expect(b.stored()).toEqual({ refresh_token: "rt-1", login_id: X });

    b.nerve.calls[0]?.answer(json(200, b.nerve.tokens()));
    await until(() => signIn.settled && token.settled, "the sign-in");
    const y = other.tm.state.loginId;
    expect(b.stored()).toEqual({ refresh_token: "rt-y", login_id: y });
    expect(a!.tm.state).toEqual({ status: "signed-in", loginId: y });
  });

  it.each([
    [
      "answers",
      () =>
        json(200, {
          token_type: "Bearer",
          access_token: "at-x",
          access_token_expires_in: 900,
          refresh_token: "rt-x",
          refresh_token_expires_at: "2026-10-27T00:00:00Z",
        }),
    ],
    ["refuses", () => problem(401, "identity.refresh_token_invalid")],
  ])("drop a refresh whose record changed to another account on the way, when nerve %s it", async (_, answer) => {
    const b = browser(kind);
    const [a] = await signedIn(b, ["A"]);
    await vi.advanceTimersByTimeAsync(10_000);
    const token = track(a!.tm.accessToken());
    await until(() => b.nerve.calls.length === 1, "the refresh");
    expect(b.nerve.calls[0]?.body).toEqual({ refresh_token: "rt-1" });

    // Another tab signs in as Y while the refresh is out: without navigator.locks, the lease is not
    // atomic, so this can happen (M2 design 7.1).
    b.storage.write(AUTH_KEY, JSON.stringify({ refresh_token: "rt-y", login_id: Y }));
    b.nerve.calls[0]?.answer(answer());
    await until(() => token.settled, "the refresh's end");

    expect(token.error).toBeInstanceOf(SessionChangedError);
    expect(b.stored()).toEqual({ refresh_token: "rt-y", login_id: Y });
    expect(a!.tm.state).toEqual({ status: "signed-in", loginId: Y });

    // The tab goes on as Y: its next token comes from Y's refresh token.
    const next = track(a!.tm.accessToken());
    await until(() => b.nerve.calls.length === 2, "Y's refresh");
    expect(b.nerve.calls[1]?.body).toEqual({ refresh_token: "rt-y" });
    b.nerve.calls[1]?.answer(json(200, b.nerve.tokens()));
    await until(() => next.settled, "Y's token");
    expect(b.stored()).toEqual({ refresh_token: "rt-2", login_id: Y });
  });

  it("end the session in every tab when one signs out", async () => {
    const b = browser(kind);
    const [a, c] = await signedIn(b, ["A", "C"], 900);
    const out = track(a!.tm.signOut());
    await until(() => b.nerve.calls.length === 1, "the logout");
    expect(b.nerve.calls[0]).toMatchObject({ path: LOGOUT, body: { refresh_token: "rt-2" } });
    b.nerve.calls[0]?.answer(noContent());
    await until(() => out.settled && c!.tm.state.status === "signed-out", "the other tab's sign-out");

    expect(c!.changes).toEqual(["signed-out -"]);
    expect(await c!.tm.accessToken()).toBeUndefined();
    expect(b.nerve.calls).toHaveLength(1);
  });

  it("sign out with the refresh token another tab's refresh just wrote", async () => {
    const b = browser(kind);
    const [a, c] = await signedIn(b, ["A", "C"], 900);
    const token = track(c!.tm.renew("at-2"));
    await until(() => b.nerve.calls.length === 1, "the refresh");
    const out = track(a!.tm.signOut());
    await vi.advanceTimersByTimeAsync(1_000);
    expect(b.nerve.calls).toHaveLength(1);

    b.nerve.calls[0]?.answer(json(200, b.nerve.tokens()));
    await until(() => b.nerve.calls.length === 2, "the logout");
    expect(b.nerve.calls[1]).toMatchObject({ path: LOGOUT, body: { refresh_token: "rt-3" } });
    b.nerve.calls[1]?.answer(noContent());
    await until(() => out.settled && token.settled, "the sign-out");
    expect(b.storage.data.has(AUTH_KEY)).toBe(false);
    expect([a!.tm.state, c!.tm.state]).toEqual([{ status: "signed-out" }, { status: "signed-out" }]);
  });

  it("sign a signed-out tab in when another tab signs in", async () => {
    const b = browser(kind);
    const [a, c] = [b.tab("A"), b.tab("C")];
    await Promise.all([a.tm.start(), c.tm.start()]);
    expect(c.tm.state).toEqual({ status: "signed-out" });

    const signIn = track(a.tm.signIn(b.nerve.tokens()));
    await until(() => signIn.settled, "the sign-in");
    const x = a.tm.state.loginId;
    await until(() => c.tm.state.status === "signed-in", "the other tab's sign-in");
    expect(c.tm.state).toEqual({ status: "signed-in", loginId: x });

    // It has no access token yet: it refreshes with the record, under the lock.
    const token = track(c.tm.accessToken());
    await until(() => b.nerve.calls.length === 1, "its refresh");
    expect(b.nerve.calls[0]?.body).toEqual({ refresh_token: "rt-1" });
    b.nerve.calls[0]?.answer(json(200, b.nerve.tokens()));
    await until(() => token.settled, "its token");
    expect(token.value).toBe("at-2");
    expect(b.stored()).toEqual({ refresh_token: "rt-2", login_id: x });
  });

  it("switch every tab to the account another tab signs in as", async () => {
    const b = browser(kind);
    const [a, c] = await signedIn(b, ["A", "C"], 900);
    const other = b.tab("D");
    const signIn = track(other.tm.signIn({ ...b.nerve.tokens(), refresh_token: "rt-y" }));
    await until(() => signIn.settled, "the sign-in");
    const y = other.tm.state.loginId;
    expect(y).not.toBe(X);
    await until(() => a!.tm.state.loginId === y && c!.tm.state.loginId === y, "the switch");
    expect(a!.changes).toEqual([`signed-in ${y}`]);

    // The old access token is gone: the next request of either tab uses Y's session.
    const token = track(a!.tm.accessToken());
    await until(() => b.nerve.calls.length === 1, "Y's refresh");
    expect(b.nerve.calls[0]?.body).toEqual({ refresh_token: "rt-y" });
    b.nerve.calls[0]?.answer(json(200, b.nerve.tokens()));
    await until(() => token.settled, "Y's token");
    expect(token.value).toBe("at-4");
  });

  it("leave a tab alone when another tab only refreshed", async () => {
    const b = browser(kind);
    const [a, c] = await signedIn(b, ["A", "C"], 900);
    const before = c!.tm.state;
    const ownToken = track(c!.tm.accessToken());
    await until(() => ownToken.settled, "the token");

    const token = track(a!.tm.renew("at-1"));
    await until(() => b.nerve.calls.length === 1, "the refresh");
    b.nerve.calls[0]?.answer(json(200, b.nerve.tokens()));
    await until(() => token.settled, "the token");
    await vi.advanceTimersByTimeAsync(100);

    expect(b.stored()).toEqual({ refresh_token: "rt-3", login_id: X });
    expect(c!.tm.state).toBe(before);
    expect(c!.changes).toEqual([]);
    const again = track(c!.tm.accessToken());
    await vi.advanceTimersByTimeAsync(0);
    expect(again.value).toBe(ownToken.value);
    expect(b.nerve.calls).toHaveLength(1);
  });

  it("free the lock 8 s into a refresh that does not come back, before a lease would expire", async () => {
    const b = browser(kind);
    const [a, c] = await signedIn(b, ["A", "C"], 20);
    await vi.advanceTimersByTimeAsync(10_000);
    const first = track(a!.tm.accessToken());
    await until(() => b.nerve.calls.length === 1, "the refresh");
    const sentAt = b.nerve.calls[0]?.at ?? 0;
    const second = track(c!.tm.accessToken());

    await vi.advanceTimersByTimeAsync(sentAt + 8_000 - 1 - Date.now());
    expect(b.nerve.calls[0]?.aborted()).toBe(false);
    expect(b.nerve.calls).toHaveLength(1);
    await vi.advanceTimersByTimeAsync(1);
    expect(b.nerve.calls[0]?.aborted()).toBe(true);
    await until(() => first.settled, "the first refresh's end");
    expect(first.error).toBeInstanceOf(SessionUnavailableError);

    // The other tab's refresh goes out at once, not when the 10 s lease would run out.
    await until(() => b.nerve.calls.length === 2, "the other tab's refresh");
    expect(Date.now() - sentAt).toBeLessThan(8_500);
    b.nerve.calls[1]?.answer(json(200, b.nerve.tokens()));
    await until(() => second.settled, "the other tab's token");
    expect(second.value).toBe("at-3");
  });
});
```

- [ ] **Step 2: 检查**

Run: `pnpm -C web/apps/web exec vitest run core/lib/auth`
Expected: 3 个文件、59 个测试通过。

Run: `make lint-web`、`make knip`、`make test-web`
Expected: 都通过。

- [ ] **Step 3: 提交**

```bash
git add web/apps/web/core/lib/auth/token-manager.tabs.test.ts
```
```bash
git commit -m "test(M2/P4): the token manager across tabs, with either lock

Two or three tabs share one localStorage and one fake nerve, with
navigator.locks or with the lease: refreshes take turns, a sign-in waits
for a refresh, a refresh whose record changed to another account is
dropped, sign-out and sign-in reach every tab.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** 20 个测试通过；两种锁的每个测试都至少有两个标签页，或一个标签页加上"另一个标签页"的写入。

---

### Task 4: 认证中间件：带上访问令牌，401 时续期并重发一次

**Files:**
- Modify: `web/packages/api-client/src/index.ts`
- Create: `web/apps/web/core/lib/auth/auth-middleware.ts`、`auth-middleware.test.ts`

**Interfaces:**
- Produces（spec 2.6，M2 设计 7.1，Codex M-5）：
  - `@nerve/api-client` 另导出 openapi-fetch 的 `Middleware` 类型；
  - `authMiddleware(tokens: Pick<TokenManager, "accessToken" | "renew" | "endSession">): Middleware`：
    - `onRequest`：有访问令牌就设 `Authorization: Bearer …`，并在 fetch 读走请求体之前用 `request.clone()` 留一份副本（`WeakMap`，键是 openapi-fetch 交给 `onResponse` 的同一个请求对象）；没有会话时请求原样发出；
    - `onResponse`：只处理带了令牌却得到 401 的请求：`renew(发出时的令牌)`；得到 `undefined`（续期 401 结束了会话，或会话已经结束）就把原来的 401 交给调用方；否则给副本换上新令牌、用 `options.fetch` 重发一次；重发仍是 401 时 `endSession()`。续期因 429、5xx、网络失败时，请求以 `SessionUnavailableError` 失败，会话保留。
- 使用者：Task 7 的 `api` 客户端。

**Tests:**（`auth-middleware.test.ts`，13 个：真实的令牌管理器、假 nerve）
- 带上访问令牌；没有会话时不带令牌，401 原样交回；没有会话时发出的请求，在它的 401 回来之前本标签页登录了，也不重发（否则一个未登录时发出的请求会以新会话的身份重发）；401 之后续期、用新令牌重发同一个 `PATCH`（方法、路径、请求体都相同），只一次；同时被拒的两个请求只续期一次；续期 401 时结束会话、交回第一个 401；重发仍 401 时结束会话；续期得到 429、503、没有网络时保留会话、请求失败；403、404、500 不续期。

- [ ] **Step 1: 导出 `Middleware`**

`web/packages/api-client/src/index.ts`（对 Task 2 版本的差异）：

```diff
--- a/web/packages/api-client/src/index.ts
+++ b/web/packages/api-client/src/index.ts
@@ -4,6 +4,7 @@
 
 // The schemas' own names (User, Profile, Problem …): openapi-typescript's --root-types (M2 design 3.12).
 export type * from "./schema.gen";
+export type { Middleware } from "openapi-fetch";
 
 /**
  * Creates a client for the Nerve API. Paths, parameters, request bodies and
```

- [ ] **Step 2: 中间件和测试**

`web/apps/web/core/lib/auth/auth-middleware.ts`（新文件）：

```ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { Middleware } from "@nerve/api-client";
import type { TokenManager } from "./token-manager";

/**
 * Puts the access token on every request of the client, and answers a 401 to it (M2 design 7.1, Codex
 * M-5): renews the token once and sends a copy of the request again, once. Two outcomes end the session:
 * the refresh answers 401 (the token manager ends it), or the request sent again is refused as well. A
 * refresh that fails for a passing reason (429, 5xx, no network) rejects the request with that error and
 * keeps the session. Without a session the request goes out without a token, and its 401 comes back as is.
 */
export function authMiddleware(tokens: Pick<TokenManager, "accessToken" | "renew" | "endSession">): Middleware {
  // A copy of each request made before fetch reads its body, and the token it went with; openapi-fetch
  // hands onResponse the request object onRequest returned.
  const sent = new WeakMap<Request, { copy: Request; token: string }>();
  return {
    async onRequest({ request }) {
      const token = await tokens.accessToken();
      if (token === undefined) return undefined;
      request.headers.set("Authorization", `Bearer ${token}`);
      sent.set(request, { copy: request.clone(), token });
      return request;
    },
    async onResponse({ request, response, options }) {
      const first = sent.get(request);
      if (response.status !== 401 || first === undefined) return undefined;
      const token = await tokens.renew(first.token);
      if (token === undefined) return undefined;
      first.copy.headers.set("Authorization", `Bearer ${token}`);
      const again = await options.fetch(first.copy);
      if (again.status === 401) await tokens.endSession();
      return again;
    },
  };
}
```

`web/apps/web/core/lib/auth/auth-middleware.test.ts`（新文件）：

```ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { authMiddleware } from "./auth-middleware";
import { RecordingLock, SharedStorage } from "./fake-browser";
import { FakeNerve, json, problem } from "./fake-nerve";
import { track, until } from "./fake-time";
import { AUTH_KEY, SessionUnavailableError, TokenManager } from "./token-manager";

// The client the app uses, with the middleware on a real token manager, against a fake nerve that answers
// when the test says (M2 design 7.1, 9.4).

const REFRESH = "/api/v0/auth/refresh";
const ME = "/api/v0/me";
const PROFILE = "/api/v0/me/profile";
const loginId = "0123456789abcdef0123456789abcdef";
const user = {
  id: "5f0c1b1e-8a6d-4d0e-9d0b-0a8f5f5f5f5f",
  email: "ada@example.com",
  first_name: "Ada",
  last_name: "",
  display_name: "ada",
  user_timezone: "UTC",
  avatar_url: null,
  cover_image_url: null,
  created_at: "2026-09-27T00:00:00Z",
};

async function setUp(record = true) {
  const storage = new SharedStorage();
  if (record) storage.data.set(AUTH_KEY, JSON.stringify({ refresh_token: "rt-0", login_id: loginId }));
  const nerve = new FakeNerve();
  const tm = new TokenManager({
    storage: storage.tab("A"),
    lock: new RecordingLock(),
    client: nerve.client(),
    now: () => Date.now(),
    randomHex: () => loginId,
  });
  const started = track(tm.start());
  if (record) {
    await until(() => nerve.calls.length === 1, "the first refresh");
    nerve.calls[0]?.answer(json(200, nerve.tokens()));
  }
  await until(() => started.settled, "the start");
  nerve.calls.length = 0;
  const api = nerve.client();
  api.use(authMiddleware(tm));
  return { storage, nerve, tm, api };
}

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("authMiddleware", () => {
  it("sends the access token", async () => {
    const { nerve, api } = await setUp();
    const me = track(api.GET(ME));
    await until(() => nerve.calls.length === 1, "the request");
    expect(nerve.calls[0]).toMatchObject({ method: "GET", path: ME, authorization: "Bearer at-1" });
    nerve.calls[0]?.answer(json(200, user));
    await until(() => me.settled, "the answer");
    expect(me.value?.data).toEqual(user);
  });

  it("sends no token without a session, and hands a 401 back as it is", async () => {
    const { nerve, api, tm } = await setUp(false);
    expect(tm.state.status).toBe("signed-out");
    const me = track(api.GET(ME));
    await until(() => nerve.calls.length === 1, "the request");
    expect(nerve.calls[0]?.authorization).toBeNull();
    nerve.calls[0]?.answer(problem(401, "unauthorized"));
    await until(() => me.settled, "the answer");
    expect(me.value?.response.status).toBe(401);
    expect(me.value?.error).toMatchObject({ code: "unauthorized" });
    await vi.advanceTimersByTimeAsync(1_000);
    expect(nerve.calls).toHaveLength(1);
  });

  it("does not send a request made without a session again when a session began before its 401", async () => {
    const { nerve, api, tm } = await setUp(false);
    const me = track(api.GET(ME));
    await until(() => nerve.calls.length === 1, "the request");
    expect(nerve.calls[0]?.authorization).toBeNull();
    // A sign-in while the request is out: the request stays the signed-out one it was.
    await tm.signIn(nerve.tokens());
    nerve.calls[0]?.answer(problem(401, "unauthorized"));
    await until(() => me.settled, "the answer");
    expect(me.value?.response.status).toBe(401);
    await vi.advanceTimersByTimeAsync(1_000);
    expect(nerve.calls).toHaveLength(1);
  });

  it("renews the token after a 401 and sends the same request again, once", async () => {
    const { nerve, api } = await setUp();
    const saved = track(api.PATCH(PROFILE, { body: { theme: "dark" } }));
    await until(() => nerve.calls.length === 1, "the request");
    nerve.calls[0]?.answer(problem(401, "unauthorized"));
    await until(() => nerve.calls.length === 2, "the refresh");
    expect(nerve.calls[1]).toMatchObject({ path: REFRESH, body: { refresh_token: "rt-1" }, authorization: null });
    nerve.calls[1]?.answer(json(200, nerve.tokens()));
    await until(() => nerve.calls.length === 3, "the request again");
    expect(nerve.calls[2]).toMatchObject({
      method: "PATCH",
      path: PROFILE,
      body: { theme: "dark" },
      authorization: "Bearer at-2",
    });
    nerve.calls[2]?.answer(json(200, { theme: "dark" }));
    await until(() => saved.settled, "the answer");
    expect(saved.value?.data).toEqual({ theme: "dark" });
    expect(nerve.calls).toHaveLength(3);
  });

  it("renews once for requests refused at the same time", async () => {
    const { nerve, api } = await setUp();
    const me = track(api.GET(ME));
    const saved = track(api.PATCH(PROFILE, { body: { language: "en" } }));
    await until(() => nerve.calls.length === 2, "the requests");
    nerve.calls[0]?.answer(problem(401, "unauthorized"));
    nerve.calls[1]?.answer(problem(401, "unauthorized"));
    await until(() => nerve.to(REFRESH).length === 1, "the refresh");
    await vi.advanceTimersByTimeAsync(100);
    expect(nerve.to(REFRESH)).toHaveLength(1);
    nerve.to(REFRESH)[0]?.answer(json(200, nerve.tokens()));
    await until(() => nerve.calls.length === 5, "the requests again");
    const again = nerve.calls.slice(3);
    expect(again.map((c) => [c.method, c.authorization])).toEqual([
      ["GET", "Bearer at-2"],
      ["PATCH", "Bearer at-2"],
    ]);
    again[0]?.answer(json(200, user));
    again[1]?.answer(json(200, { language: "en" }));
    await until(() => me.settled && saved.settled, "the answers");
    expect([me.value?.response.status, saved.value?.response.status]).toEqual([200, 200]);
  });

  it("ends the session when the refresh answers 401, and hands the first 401 back", async () => {
    const { nerve, api, tm, storage } = await setUp();
    const me = track(api.GET(ME));
    await until(() => nerve.calls.length === 1, "the request");
    nerve.calls[0]?.answer(problem(401, "unauthorized"));
    await until(() => nerve.calls.length === 2, "the refresh");
    nerve.calls[1]?.answer(problem(401, "identity.refresh_token_invalid"));
    await until(() => me.settled, "the answer");

    expect(me.value?.response.status).toBe(401);
    expect(me.value?.error).toMatchObject({ code: "unauthorized" });
    expect(tm.state).toEqual({ status: "signed-out" });
    expect(storage.data.has(AUTH_KEY)).toBe(false);
    expect(nerve.calls).toHaveLength(2);
  });

  it("ends the session when the request sent again is refused too", async () => {
    const { nerve, api, tm, storage } = await setUp();
    const me = track(api.GET(ME));
    await until(() => nerve.calls.length === 1, "the request");
    nerve.calls[0]?.answer(problem(401, "unauthorized"));
    await until(() => nerve.calls.length === 2, "the refresh");
    nerve.calls[1]?.answer(json(200, nerve.tokens()));
    await until(() => nerve.calls.length === 3, "the request again");
    nerve.calls[2]?.answer(problem(401, "unauthorized"));
    await until(() => me.settled, "the answer");

    expect(me.value?.response.status).toBe(401);
    expect(tm.state).toEqual({ status: "signed-out" });
    expect(storage.data.has(AUTH_KEY)).toBe(false);
    await vi.advanceTimersByTimeAsync(1_000);
    expect(nerve.calls).toHaveLength(3);
  });

  it.each([
    ["429", () => problem(429, "rate_limited")],
    ["503", () => problem(503, "server_busy")],
    ["no network", () => Response.error()],
  ])("keeps the session when the refresh gets a %s, and fails the request with it", async (_, answer) => {
    const { nerve, api, tm, storage } = await setUp();
    const me = track(api.GET(ME));
    await until(() => nerve.calls.length === 1, "the request");
    nerve.calls[0]?.answer(problem(401, "unauthorized"));
    await until(() => nerve.calls.length === 2, "the refresh");
    nerve.calls[1]?.answer(answer());
    await until(() => me.settled, "the answer");

    expect(me.error).toBeInstanceOf(SessionUnavailableError);
    expect(tm.state).toEqual({ status: "signed-in", loginId });
    expect(JSON.parse(storage.data.get(AUTH_KEY) ?? "null")).toEqual({ refresh_token: "rt-1", login_id: loginId });
    expect(nerve.calls).toHaveLength(2);
  });

  it.each([403, 404, 500])("hands a %s back without renewing", async (status) => {
    const { nerve, api } = await setUp();
    const me = track(api.GET(ME));
    await until(() => nerve.calls.length === 1, "the request");
    nerve.calls[0]?.answer(problem(status, "some_code"));
    await until(() => me.settled, "the answer");
    expect(me.value?.response.status).toBe(status);
    await vi.advanceTimersByTimeAsync(1_000);
    expect(nerve.calls).toHaveLength(1);
  });
});
```

- [ ] **Step 3: 检查**

Run: `pnpm -C web/apps/web exec vitest run core/lib/auth`
Expected: 4 个文件、72 个测试通过。

Run: `make lint-web`、`make knip`、`make test-web`
Expected: 都通过。

- [ ] **Step 4: 提交**

```bash
git add web/packages/api-client/src/index.ts web/apps/web/core/lib/auth/auth-middleware.ts web/apps/web/core/lib/auth/auth-middleware.test.ts
```
```bash
git commit -m "feat(M2/P4): the auth middleware: the access token on every request, a 401 renewed once

onRequest keeps a copy of the request before fetch reads its body;
onResponse renews the token after a 401 and sends the copy again, once.
A refresh answered 401, or a copy refused again, ends the session; a
passing failure keeps it and fails the request.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** 13 个测试通过；这证明 openapi-fetch 0.17.0 的中间件能在 `onRequest` 留下请求的副本、在 `onResponse` 重发一次（M2 设计 7.1 要求原型先验证的一点）。

---

### Task 5: 新手引导：删除"角色""用途"两步、挂载时的预取和资料步骤的头像上传

**Files:**
- Delete: `web/apps/web/core/components/onboarding/steps/role/`、`steps/usecase/`（各两个文件）
- Modify: `web/apps/web/app/(all)/onboarding/page.tsx`、`web/apps/web/core/components/onboarding/header.tsx`、`steps/root.tsx`
- Modify（过渡）: `web/apps/web/core/components/onboarding/root.tsx`、`steps/profile/root.tsx`、`web/packages/types/src/workspace.ts`、`web/apps/web/package.json`

**Interfaces:**
- Produces（spec 2.11，M2 设计 3.19、3.1、3.2）：
  - `EOnboardingSteps` 只剩 `PROFILE_SETUP`、`WORKSPACE_CREATE_OR_JOIN`、`INVITE_MEMBERS`；页头的进度只有这三步，"返回"只从工作区一步回到资料一步；新手引导不再读 `is_self_managed`（字段在 Task 6 随 `IInstanceConfig` 删除）；
  - `/onboarding` 挂载时不再取工作区列表和邀请（M3 的接口）；`OnboardingRoot` 的 `invitations` 默认为空；
  - 资料步骤没有头像上传：已有头像时显示，没有时显示名字的首字母；表单只有名、姓。
- web 的 oxlint 上限 565 → 560（删掉的文件带走 5 条警告）。
- 使用者：Task 7 让资料步骤只在保存成功后前进；Task 8 让根组件按步骤部分更新资料；Task 12 的 A10 核对第一次打开 `/onboarding`。

**Tests:** 没有单元测试（界面）。A10 的页面版本（Task 12）断言第一次打开 `/onboarding` 没有失败的接口请求、没有发往 M3 旧接口的请求、没有未处理的异常、控制台没有错误；把预取加回根组件的变异让它失败（spec 附录 A）。

- [ ] **Step 1: 删除两个步骤**

```bash
git rm -r web/apps/web/core/components/onboarding/steps/role web/apps/web/core/components/onboarding/steps/usecase
```

- [ ] **Step 2: 三步的新手引导**

`web/packages/types/src/workspace.ts`（对 `f27434c` 的差异）：

```diff
--- a/web/packages/types/src/workspace.ts
+++ b/web/packages/types/src/workspace.ts
@@ -130,8 +130,6 @@
 
 export enum EOnboardingSteps {
   PROFILE_SETUP = "PROFILE_SETUP",
-  ROLE_SETUP = "ROLE_SETUP",
-  USE_CASE_SETUP = "USE_CASE_SETUP",
   WORKSPACE_CREATE_OR_JOIN = "WORKSPACE_CREATE_OR_JOIN",
   INVITE_MEMBERS = "INVITE_MEMBERS",
 }
```

`web/apps/web/core/components/onboarding/steps/root.tsx`（对 `f27434c` 的差异）：

```diff
--- a/web/apps/web/core/components/onboarding/steps/root.tsx
+++ b/web/apps/web/core/components/onboarding/steps/root.tsx
@@ -10,9 +10,7 @@
 import { EOnboardingSteps } from "@nerve/types";
 // local components
 import { ProfileSetupStep } from "./profile";
-import { RoleSetupStep } from "./role";
 import { InviteTeamStep } from "./team";
-import { UseCaseSetupStep } from "./usecase";
 import { WorkspaceSetupStep } from "./workspace";
 
 type Props = {
@@ -25,10 +23,6 @@
   switch (currentStep) {
     case EOnboardingSteps.PROFILE_SETUP:
       return <ProfileSetupStep handleStepChange={handleStepChange} />;
-    case EOnboardingSteps.ROLE_SETUP:
-      return <RoleSetupStep handleStepChange={handleStepChange} />;
-    case EOnboardingSteps.USE_CASE_SETUP:
-      return <UseCaseSetupStep handleStepChange={handleStepChange} />;
     case EOnboardingSteps.WORKSPACE_CREATE_OR_JOIN:
       return <WorkspaceSetupStep invitations={invitations ?? []} handleStepChange={handleStepChange} />;
     case EOnboardingSteps.INVITE_MEMBERS:
```

`web/apps/web/core/components/onboarding/header.tsx`（对 `f27434c` 的差异）：

```diff
--- a/web/apps/web/core/components/onboarding/header.tsx
+++ b/web/apps/web/core/components/onboarding/header.tsx
@@ -14,7 +14,6 @@
 // components
 import { NerveLockup } from "@/components/common/nerve-logo";
 // hooks
-import { useInstance } from "@/hooks/store/use-instance";
 import { useUser } from "@/hooks/store/user";
 // local imports
 import { SwitchAccountDropdown } from "./switch-account-dropdown";
@@ -29,22 +28,10 @@
   const { currentStep, updateCurrentStep, hasInvitations } = props;
   // store hooks
   const { data: user } = useUser();
-  const { config: instanceConfig } = useInstance();
-  const isSelfManaged = instanceConfig?.is_self_managed;
 
   // handle step back
   const handleStepBack = () => {
-    switch (currentStep) {
-      case EOnboardingSteps.ROLE_SETUP:
-        updateCurrentStep(EOnboardingSteps.PROFILE_SETUP);
-        break;
-      case EOnboardingSteps.USE_CASE_SETUP:
-        updateCurrentStep(EOnboardingSteps.ROLE_SETUP);
-        break;
-      case EOnboardingSteps.WORKSPACE_CREATE_OR_JOIN:
-        updateCurrentStep(isSelfManaged ? EOnboardingSteps.PROFILE_SETUP : EOnboardingSteps.USE_CASE_SETUP);
-        break;
-    }
+    if (currentStep === EOnboardingSteps.WORKSPACE_CREATE_OR_JOIN) updateCurrentStep(EOnboardingSteps.PROFILE_SETUP);
   };
 
   // can go back
@@ -54,7 +41,6 @@
   const showInviteStep = !hasInvitations || currentStep === EOnboardingSteps.INVITE_MEMBERS;
   const stepOrder: TOnboardingStep[] = [
     EOnboardingSteps.PROFILE_SETUP,
-    ...(isSelfManaged ? [] : [EOnboardingSteps.ROLE_SETUP, EOnboardingSteps.USE_CASE_SETUP]),
     EOnboardingSteps.WORKSPACE_CREATE_OR_JOIN,
     ...(showInviteStep ? [EOnboardingSteps.INVITE_MEMBERS] : []),
   ];
```

`web/apps/web/core/components/onboarding/root.tsx`（对 `f27434c` 的差异）：

```diff
--- a/web/apps/web/core/components/onboarding/root.tsx
+++ b/web/apps/web/core/components/onboarding/root.tsx
@@ -11,7 +11,6 @@
 import type { IWorkspaceMemberInvitation, TOnboardingStep, TOnboardingSteps, TUserProfile } from "@nerve/types";
 import { EOnboardingSteps } from "@nerve/types";
 // hooks
-import { useInstance } from "@/hooks/store/use-instance";
 import { useWorkspace } from "@/hooks/store/use-workspace";
 import { useUser, useUserProfile } from "@/hooks/store/user";
 // local components
@@ -28,10 +27,8 @@
   const { data: user } = useUser();
   const { data: userProfile, updateUserProfile, finishUserOnboarding } = useUserProfile();
   const { workspaces } = useWorkspace();
-  const { config: instanceConfig } = useInstance();
 
   const workspacesList = Object.values(workspaces ?? {});
-  const isSelfManaged = instanceConfig?.is_self_managed;
 
   // Calculate total steps based on whether invitations are available
   const hasInvitations = invitations.length > 0;
@@ -71,19 +68,6 @@
     (step: EOnboardingSteps, skipInvites?: boolean) => {
       switch (step) {
         case EOnboardingSteps.PROFILE_SETUP:
-          if (isSelfManaged) {
-            // Skip role & use case steps for self-hosted
-            stepChange({ profile_complete: true });
-            if (workspacesList.length > 0) finishOnboarding();
-            else setCurrentStep(EOnboardingSteps.WORKSPACE_CREATE_OR_JOIN);
-          } else {
-            setCurrentStep(EOnboardingSteps.ROLE_SETUP);
-          }
-          break;
-        case EOnboardingSteps.ROLE_SETUP:
-          setCurrentStep(EOnboardingSteps.USE_CASE_SETUP);
-          break;
-        case EOnboardingSteps.USE_CASE_SETUP:
           stepChange({ profile_complete: true });
           if (workspacesList.length > 0) finishOnboarding();
           else setCurrentStep(EOnboardingSteps.WORKSPACE_CREATE_OR_JOIN);
@@ -101,7 +85,7 @@
           break;
       }
     },
-    [stepChange, finishOnboarding, workspacesList, isSelfManaged]
+    [stepChange, finishOnboarding, workspacesList]
   );
 
   const updateCurrentStep = (step: EOnboardingSteps) => setCurrentStep(step);
```

`web/apps/web/app/(all)/onboarding/page.tsx`（完整内容）：

```text
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// components
import { LogoSpinner } from "@/components/common/logo-spinner";
import { OnboardingRoot } from "@/components/onboarding";
// helpers
import { EPageTypes } from "@/helpers/authentication.helper";
// hooks
import { useUser } from "@/hooks/store/user";
// wrappers
import { AuthenticationWrapper } from "@/lib/wrappers/authentication-wrapper";

// The workspaces and the invitations come back with the workspace API (M3): until then the page asks for
// nothing of them when it opens (M2 design 3.1).
function OnboardingPage() {
  // store hooks
  const { data: user } = useUser();

  return (
    <AuthenticationWrapper pageType={EPageTypes.ONBOARDING}>
      <div className="relative flex size-full overflow-hidden rounded-lg bg-canvas transition-all duration-300 ease-in-out">
        <div className="size-full flex-grow overflow-hidden p-2 transition-all duration-300 ease-in-out">
          <div className="shadow-md relative flex h-full w-full flex-col overflow-hidden rounded-lg border border-subtle bg-surface-1">
            {user ? (
              <OnboardingRoot />
            ) : (
              <div className="grid h-full w-full place-items-center">
                <LogoSpinner />
              </div>
            )}
          </div>
        </div>
      </div>
    </AuthenticationWrapper>
  );
}

export default observer(OnboardingPage);
```

`web/apps/web/core/components/onboarding/steps/profile/root.tsx`（完整内容）：

```text
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { Controller, useForm } from "react-hook-form";
// nerve imports
import { Button } from "@nerve/propel/button";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import type { IUser } from "@nerve/types";
import { EOnboardingSteps } from "@nerve/types";
import { cn, getFileURL, validatePersonName } from "@nerve/utils";
// hooks
import { useUser } from "@/hooks/store/user";
// local components
import { CommonOnboardingHeader } from "../common";

type Props = {
  handleStepChange: (step: EOnboardingSteps, skipInvites?: boolean) => void;
};

type TProfileSetupFormValues = {
  first_name: string;
  last_name: string;
};

export const ProfileSetupStep = observer(function ProfileSetupStep({ handleStepChange }: Props) {
  // store hooks
  const { data: user, updateCurrentUser } = useUser();
  // form info
  const {
    handleSubmit,
    control,
    watch,
    formState: { errors, isSubmitting, isValid },
  } = useForm<TProfileSetupFormValues>({
    defaultValues: {
      first_name: user?.first_name ?? "",
      last_name: user?.last_name ?? "",
    },
    mode: "onChange",
  });

  const handleSubmitUserDetail = async (formData: TProfileSetupFormValues) => {
    const userDetailsPayload: Partial<IUser> = {
      first_name: formData.first_name,
      last_name: formData.last_name,
    };
    try {
      await updateCurrentUser(userDetailsPayload);
    } catch {
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Error",
        message: "User details update failed. Please try again!",
      });
    }
  };

  const onSubmit = async (formData: TProfileSetupFormValues) => {
    if (!user) return;
    await handleSubmitUserDetail(formData);
    handleStepChange(EOnboardingSteps.PROFILE_SETUP);
  };

  const isButtonDisabled = isSubmitting || !isValid;

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col gap-10">
      {/* Header */}
      <CommonOnboardingHeader title="Create your profile." description="This is how you will appear in Nerve." />

      {/* Profile picture: shown, not uploaded, until the file storage (M5) */}
      <div className="flex size-12 items-center justify-center rounded-full bg-accent-primary text-18 font-semibold text-on-color">
        {user?.avatar_url ? (
          <img
            src={getFileURL(user.avatar_url)}
            alt={user.display_name}
            className="h-full w-full rounded-full object-cover"
          />
        ) : (
          <>{watch("first_name")[0] ?? "R"}</>
        )}
      </div>

      <div className="flex w-full flex-col gap-6">
        {/* Name Input */}
        <div className="flex flex-col gap-2">
          <label
            className="block text-13 font-medium text-tertiary after:ml-0.5 after:text-danger-primary after:content-['*']"
            htmlFor="first_name"
          >
            Name
          </label>
          <Controller
            control={control}
            name="first_name"
            rules={{
              required: "Name is required",
              validate: validatePersonName,
              maxLength: {
                value: 50,
                message: "Name must be within 50 characters.",
              },
            }}
            render={({ field: { value, onChange, ref } }) => (
              <input
                ref={ref}
                id="first_name"
                name="first_name"
                type="text"
                value={value}
                onChange={(e) => onChange(e.target.value)}
                autoFocus
                className={cn(
                  "w-full rounded-md border border-strong bg-surface-1 px-3 py-2 text-secondary transition-all duration-200 placeholder:text-placeholder focus:border-transparent focus:ring-2 focus:ring-accent-strong focus:outline-none",
                  {
                    "border-strong": !errors.first_name,
                    "border-danger-strong": errors.first_name,
                  }
                )}
                placeholder="Enter your full name"
                autoComplete="on"
              />
            )}
          />
          {errors.first_name && <span className="text-13 text-danger-primary">{errors.first_name.message}</span>}
        </div>
      </div>
      {/* Continue Button */}
      <Button variant="primary" type="submit" className="w-full" size="xl" disabled={isButtonDisabled}>
        Continue
      </Button>
    </form>
  );
});
```

`web/apps/web/package.json`（对 Task 2 版本的差异）：

```diff
--- a/web/apps/web/package.json
+++ b/web/apps/web/package.json
@@ -7,7 +7,7 @@
   "scripts": {
     "dev": "react-router dev --port 3000",
     "build": "react-router build",
-    "check:lint": "node ../../../tools/lint-cap.mjs 565",
+    "check:lint": "node ../../../tools/lint-cap.mjs 560",
     "check:types": "react-router typegen && tsc --noEmit",
     "check:format": "oxfmt --check .",
     "fix:format": "oxfmt .",
```

- [ ] **Step 3: 没有残留**

Run: `grep -rn -e ROLE_SETUP -e USE_CASE_SETUP -e steps/role -e steps/usecase web/apps/web/app web/apps/web/core web/packages/types/src`
Expected: 没有输出。

Run: `grep -rn -e USER_WORKSPACES_LIST -e userWorkspaceInvitations "web/apps/web/app/(all)/onboarding" web/apps/web/core/components/onboarding`
Expected: 没有输出（`/invitations` 页面仍用它们，属于 M3）。

- [ ] **Step 4: 检查**

Run: `make lint-web`、`make knip`、`make test-web`
Expected: 都通过；web 的 oxlint 警告正好 560 个。

- [ ] **Step 5: 提交**

```bash
git add -A web/apps/web/core/components/onboarding "web/apps/web/app/(all)/onboarding/page.tsx" web/packages/types/src/workspace.ts web/apps/web/package.json
```
```bash
git commit -m "feat(M2/P4): onboarding without the role and use-case steps, prefetches or avatar upload

Three steps remain: profile, workspace, invites (M2 design 3.19). The page
fetches no workspaces or invitations as it opens (they are M3's), and the
profile step shows the avatar but has no upload until the file storage.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** 新手引导只有三步；Step 3 的 `grep` 没有输出；上限 560。

---

### Task 6: 实例配置改用生成的类型；`ApiError` 与 `unwrap`

**Files:**
- Create: `web/apps/web/core/lib/api-error.ts`、`api-error.test.ts`
- Create（过渡）: `web/apps/web/core/lib/auth/api-client.ts`
- Modify: `web/apps/web/core/services/instance.service.ts`、`web/apps/web/core/store/instance.store.ts`
- Modify: `web/apps/web/app/(all)/create-workspace/page.tsx`、`web/apps/web/core/components/auth-screens/header.tsx`、`power-k/config/creation/command.ts`、`workspace/sidebar/workspace-menu-root.tsx`
- Modify（过渡）: `web/apps/web/core/components/onboarding/steps/workspace/create.tsx`、`web/packages/types/src/index.ts`、`tools/keywords.json`
- Delete: `web/packages/types/src/instance/`（`base.ts`、`index.ts`）

**Interfaces:**
- Produces（spec 2.7、2.8，M2 设计 5.3、7.2、7.4、7.5）：
  - `class ApiError extends Error { status; problem: Problem | undefined }`：消息取 `detail`，没有时取 `title`，都没有时 `HTTP <状态码>`；
  - `unwrap<T>(result: { data?; error?; response }): T`：成功（2xx）时给 `data`（204 是 `undefined`）；否则抛出 `ApiError`，只有带字符串 `code` 的 JSON 才算 problem（代理的 HTML 页面、没有 `code` 的 JSON 都不算）；
  - `api-client.ts`（过渡）：`publicClient = createClient()`，同源、相对地址，不带令牌；
  - `InstanceService.getInstanceInfo(): Promise<InstanceInfo>` 调 `GET /api/v0/instance`；`InstanceStore.config: InstanceInfo | undefined`；请求失败时 `InstanceWrapper` 仍显示维护页（`InstanceWrapper` 本身不改）；
  - 读实例配置的五处改读新字段：页头的注册链接读 `signup_enabled`；`/create-workspace`、新手引导的创建工作区、工作区菜单、命令面板读 `workspace_creation_enabled`（`is_workspace_creation_disabled` 的反义）；
  - 删除 `IInstanceInfo`、`IInstanceConfig`（`@nerve/types` 不再导出 `instance`）；
  - 关键词规则 `is-self-managed`（`is_self_managed|isSelfManaged`，`web/` 下的源码和 JSON）。
- 使用者：Task 7 起的 services 都用 `unwrap`；Task 10 的错误文案表读 `ApiError.problem`。

**Tests:**（`api-error.test.ts`，5 个；用真的 `createClient`，`fetch` 返回固定的响应）成功时的数据；204 没有数据；problem+json 变成带 problem 的 `ApiError`（消息是 `detail`）；代理的 HTML 页面变成没有 problem 的 `ApiError`（`HTTP 502`）；没有 `code` 的 JSON（`{message}`）不算 problem。

- [ ] **Step 1: `ApiError` 和 `unwrap`**

`web/apps/web/core/lib/api-error.ts`（新文件）：

```ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { Problem } from "@nerve/api-client";

/** An answer of nerve that is not a success; `problem` is its problem+json body, when it has one. */
export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly problem: Problem | undefined
  ) {
    super(problem?.detail ?? problem?.title ?? `HTTP ${status}`);
    this.name = "ApiError";
  }
}

/** The data of an answer of the generated client, or the answer as an ApiError when it is not a success. */
export function unwrap<T>(result: { data?: T; error?: unknown; response: Response }): T {
  if (result.response.ok) return result.data as T;
  throw new ApiError(result.response.status, isProblem(result.error) ? result.error : undefined);
}

function isProblem(body: unknown): body is Problem {
  return typeof body === "object" && body !== null && typeof (body as Partial<Problem>).code === "string";
}
```

`web/apps/web/core/lib/api-error.test.ts`（新文件）：

```ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import { createClient } from "@nerve/api-client";
import { ApiError, unwrap } from "./api-error";

// unwrap on real answers of the generated client, from a fetch that answers what each test gives it.
function clientAnswering(response: Response) {
  return createClient({ baseUrl: "http://nerve.test", fetch: async () => response });
}

const problem = { status: 409, code: "identity.email_taken", title: "Conflict", detail: "The email is taken." };

describe("unwrap", () => {
  it("returns the data of a success", async () => {
    const info = { product: "nerve", signup_enabled: true };
    const answer = new Response(JSON.stringify(info), { status: 200, headers: { "Content-Type": "application/json" } });
    expect(unwrap(await clientAnswering(answer).GET("/api/v0/instance"))).toEqual(info);
  });

  it("returns nothing for a 204", async () => {
    const answer = new Response(null, { status: 204 });
    const result = await clientAnswering(answer).POST("/api/v0/auth/logout", { body: { refresh_token: "rt" } });
    expect(unwrap(result)).toBeUndefined();
  });

  it("throws a problem+json answer as an ApiError that carries the problem", async () => {
    const answer = new Response(JSON.stringify(problem), {
      status: 409,
      headers: { "Content-Type": "application/problem+json" },
    });
    const result = await clientAnswering(answer).POST("/api/v0/auth/register", {
      body: { email: "a@example.com", password: "x" },
    });
    expect(() => unwrap(result)).toThrow(ApiError);
    expect(() => unwrap(result)).toThrow(expect.objectContaining({ status: 409, problem, message: problem.detail }));
  });

  it("throws an answer without a problem, e.g. a proxy's HTML page, as an ApiError without one", async () => {
    const answer = new Response("<html>Bad Gateway</html>", { status: 502, headers: { "Content-Type": "text/html" } });
    const result = await clientAnswering(answer).GET("/api/v0/instance");
    expect(() => unwrap(result)).toThrow(
      expect.objectContaining({ status: 502, problem: undefined, message: "HTTP 502" })
    );
  });

  it("does not take a JSON answer without a problem code for a problem", async () => {
    const answer = new Response(JSON.stringify({ message: "upstream timed out" }), {
      status: 504,
      headers: { "Content-Type": "application/json" },
    });
    const result = await clientAnswering(answer).GET("/api/v0/instance");
    expect(() => unwrap(result)).toThrow(
      expect.objectContaining({ status: 504, problem: undefined, message: "HTTP 504" })
    );
  });
});
```

- [ ] **Step 2: 不带令牌的客户端；实例配置**

`web/apps/web/core/lib/auth/api-client.ts`（新文件）：

```ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { createClient } from "@nerve/api-client";

// The web app's clients of nerve's API (M2 design 7.1). They send to the page's own origin, where nerve
// serves both the app and the API.

/** The client of the operations that need no token: the instance, sign-in and sign-up. */
export const publicClient = createClient();
```

`web/apps/web/core/services/instance.service.ts`（完整内容）：

```ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { InstanceInfo } from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";
import { publicClient } from "@/lib/auth/api-client";

export class InstanceService {
  async getInstanceInfo(): Promise<InstanceInfo> {
    return unwrap(await publicClient.GET("/api/v0/instance"));
  }
}
```

`web/apps/web/core/store/instance.store.ts`（完整内容）：

```ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observable, action, makeObservable, runInAction } from "mobx";
// types
import type { InstanceInfo } from "@nerve/api-client";
// services
import { InstanceService } from "@/services/instance.service";

export interface IInstanceStore {
  isLoading: boolean;
  config: InstanceInfo | undefined;
  // action
  fetchInstanceInfo: () => Promise<void>;
}

export class InstanceStore implements IInstanceStore {
  isLoading: boolean = true;
  config: InstanceInfo | undefined = undefined;
  // services
  instanceService;

  constructor() {
    makeObservable(this, {
      // observable
      isLoading: observable.ref,
      config: observable,
      // actions
      fetchInstanceInfo: action,
    });
    // services
    this.instanceService = new InstanceService();
  }

  /**
   * @description fetching instance information
   */
  fetchInstanceInfo = async () => {
    try {
      this.isLoading = true;
      const instanceInfo = await this.instanceService.getInstanceInfo();
      runInAction(() => {
        this.isLoading = false;
        this.config = instanceInfo;
      });
    } catch (error) {
      runInAction(() => {
        this.isLoading = false;
      });
      throw error;
    }
  };
}
```

- [ ] **Step 3: 读新字段的地方**

`web/apps/web/core/components/auth-screens/header.tsx`（对 `f27434c` 的差异）：

```diff
--- a/web/apps/web/core/components/auth-screens/header.tsx
+++ b/web/apps/web/core/components/auth-screens/header.tsx
@@ -38,7 +38,7 @@
   // store
   const { config } = useInstance();
   // derived values
-  const enableSignUpConfig = config?.enable_signup ?? false;
+  const enableSignUpConfig = config?.signup_enabled ?? false;
 
   return (
     <AuthHeaderBase
```

`web/apps/web/app/(all)/create-workspace/page.tsx`（对 `f27434c` 的差异）：

```diff
--- a/web/apps/web/app/(all)/create-workspace/page.tsx
+++ b/web/apps/web/app/(all)/create-workspace/page.tsx
@@ -37,7 +37,7 @@
     organization_size: "",
   });
   // derived values
-  const isWorkspaceCreationDisabled = config?.is_workspace_creation_disabled ?? false;
+  const isWorkspaceCreationDisabled = config?.workspace_creation_enabled === false;
 
   // methods
   const getMailtoHref = () => {
```

`web/apps/web/core/components/onboarding/steps/workspace/create.tsx`（对 `f27434c` 的差异）：

```diff
--- a/web/apps/web/core/components/onboarding/steps/workspace/create.tsx
+++ b/web/apps/web/core/components/onboarding/steps/workspace/create.tsx
@@ -51,7 +51,7 @@
   const { fetchCurrentUserSettings } = useUserSettings();
   const { createWorkspace, fetchWorkspaces } = useWorkspace();
 
-  const isWorkspaceCreationDisabled = config?.is_workspace_creation_disabled ?? false;
+  const isWorkspaceCreationDisabled = config?.workspace_creation_enabled === false;
 
   // form info
   const {
```

`web/apps/web/core/components/power-k/config/creation/command.ts`（对 `f27434c` 的差异）：

```diff
--- a/web/apps/web/core/components/power-k/config/creation/command.ts
+++ b/web/apps/web/core/components/power-k/config/creation/command.ts
@@ -56,7 +56,7 @@
       ctx.params.workspaceSlug,
       ctx.params.projectId
     );
-  const isWorkspaceCreationDisabled = config?.is_workspace_creation_disabled ?? false;
+  const isWorkspaceCreationDisabled = config?.workspace_creation_enabled === false;
 
   const getProjectDetails = (ctx: TPowerKContext) =>
     ctx.params.projectId ? getPartialProjectById(ctx.params.projectId) : undefined;
```

`web/apps/web/core/components/workspace/sidebar/workspace-menu-root.tsx`（对 `f27434c` 的差异）：

```diff
--- a/web/apps/web/core/components/workspace/sidebar/workspace-menu-root.tsx
+++ b/web/apps/web/core/components/workspace/sidebar/workspace-menu-root.tsx
@@ -42,7 +42,7 @@
   const { updateUserProfile } = useUserProfile();
   const { currentWorkspace: activeWorkspace, workspaces } = useWorkspace();
   // derived values
-  const isWorkspaceCreationDisabled = config?.is_workspace_creation_disabled ?? false;
+  const isWorkspaceCreationDisabled = config?.workspace_creation_enabled === false;
   // translation
   const { t } = useTranslation();
   // local state
```

- [ ] **Step 4: 删除 Plane 的实例类型；关键词规则**

```bash
git rm -r web/packages/types/src/instance
```

`web/packages/types/src/index.ts`（对 `f27434c` 的差异）：

```diff
--- a/web/packages/types/src/index.ts
+++ b/web/packages/types/src/index.ts
@@ -18,7 +18,6 @@
 export * from "./file";
 export * from "./home";
 export * from "./inbox";
-export * from "./instance";
 export * from "./issues";
 export * from "./issues/base"; // TODO: Remove this after development and the refactor/mobx-store-issue branch is stable
 export * from "./issues/issue-identifier";
```

`tools/keywords.json`（对 `f27434c` 的差异）：

```diff
--- a/tools/keywords.json
+++ b/tools/keywords.json
@@ -1,5 +1,5 @@
 {
-  "phase": "M1/closeout",
+  "phase": "M2/P4",
   "rules": [
     {
       "id": "deploy-files",
@@ -1748,6 +1748,27 @@
           "web/apps/web/core/hooks/store/use-workspace.ts"
         ]
       }
+    },
+    {
+      "id": "is-self-managed",
+      "phase": "M2/P4",
+      "why": "Plane 实例配置的 is_self_managed 和它控制的新手引导\"角色\"\"用途\"两步（M2 设计 3.19，决策点 4 裁定为 A）；Nerve 的实例信息 GET /api/v0/instance 没有这个字段，新手引导只有资料、工作区、邀请三步",
+      "files": {
+        "source": "^web/.*\\.(?:[cm]?[jt]sx?|json)$",
+        "flags": ""
+      },
+      "content": {
+        "source": "is_self_managed|isSelfManaged",
+        "flags": ""
+      },
+      "samples": {
+        "hit": ["  is_self_managed: boolean;", "  const isSelfManaged = instanceConfig?.is_self_managed;"],
+        "miss": ["  const isWorkspaceCreationDisabled = config?.workspace_creation_enabled === false;"],
+        "files": {
+          "hit": ["web/apps/web/core/components/onboarding/root.tsx", "web/packages/types/src/index.ts"],
+          "miss": ["docs/v0/M2-auth/M2-design.md", "api/dist/openapi.yaml"]
+        }
+      }
     }
   ],
   "exceptions": [
```

Run: `grep -rn -e IInstanceInfo -e IInstanceConfig -e enable_signup -e is_workspace_creation_disabled -e is_self_managed web/apps/web/app web/apps/web/core web/packages --exclude-dir=node_modules --exclude-dir=dist`
Expected: 没有输出。

- [ ] **Step 5: 检查**

Run: `pnpm -C web/apps/web exec vitest run core/lib`
Expected: 5 个文件、77 个测试通过。

Run: `make lint-web`、`make knip`、`make test-web`
Expected: 都通过（关键词守卫 52 条规则，没有命中）。

- [ ] **Step 6: 提交**

```bash
git add -A web/apps/web/core/lib web/apps/web/core/services/instance.service.ts web/apps/web/core/store/instance.store.ts web/apps/web/core/components "web/apps/web/app/(all)/create-workspace/page.tsx" web/packages/types/src tools/keywords.json
```
```bash
git commit -m "feat(M2/P4): the instance's settings from GET /api/v0/instance; ApiError and unwrap

The instance store holds the generated InstanceInfo; the sign-up link reads
signup_enabled and the four ways to create a workspace read
workspace_creation_enabled. unwrap throws a non-2xx answer as an ApiError
that carries the problem, when the body is one. IInstanceInfo and
IInstanceConfig are gone, and the keyword guard keeps is_self_managed out.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** 5 个测试通过；Step 4 的 `grep` 没有输出；未登录打开首页时，前端只请求 `GET /api/v0/instance`（Task 12 的 S2 断言）。

---

### Task 7: 账户与会话：`UserStore` 用 `User`；stores 随会话重建；删除 `IUser`

**Files:**
- Modify: `web/apps/web/core/lib/auth/api-client.ts`、`web/apps/web/core/lib/store-context.tsx`、`web/apps/web/core/store/root.store.ts`
- Modify（过渡）: `web/apps/web/core/store/user/index.ts`、`web/apps/web/core/services/auth.service.ts`、`user.service.ts`、`web/packages/types/src/users.ts`、`web/apps/web/core/components/settings/profile/content/pages/general/form.tsx`、`security.tsx`、`web/apps/web/package.json`
- Modify: `web/apps/web/core/components/onboarding/steps/profile/root.tsx`、`steps/workspace/create.tsx`、`switch-account-modal.tsx`、`account/deactivate-account-modal.tsx`、`home/user-greetings.tsx`、`issues/issue-detail/reactions/issue.tsx`、`project/settings/member-columns.tsx`、`workspace/settings/member-columns.tsx`；`web/packages/types/src/project/projects.ts`、`search.ts`、`workspace.ts`
- Delete: `web/apps/web/core/components/core/modals/user-image-upload-modal.tsx`

**Interfaces:**
- Produces（spec 2.8，M2 设计 7.1、7.5）：
  - `api-client.ts`：`tokenManager`（`localStorage`；`"locks" in navigator` 时 `webLock(navigator.locks)`，否则 `leaseLock`，标签页 id 由 `crypto.getRandomValues` 生成，非安全上下文也有；`publicClient`；`Date.now`）；`storage` 事件中 `nerve.auth` 的变化（以及 `localStorage.clear()` 的 `null` 键）交给 `handleStorageChange`；模块加载时 `void tokenManager.start()`；`api = createClient()` 挂上 `authMiddleware(tokenManager)`；
  - `store-context.tsx`：订阅令牌管理器，`loginId` 从一个值变成另一个值或没有值时 `store.resetOnSignOut()`：本标签页退出、或跟随另一个标签页换成别的账户时，页面上不留旧账户的任何数据；
  - `RootStore.resetOnSignOut()` 不再重建 `router` 和 `instance`：它们不是账户的数据，重建之后在下一次整页加载之前没有代码会再填它们（原型中重建 `instance` 让 A5 停在加载图标上，spec 第 3 节）；
  - `UserStore`（过渡）：`data: User | undefined`；`fetchCurrentUser()` 先等 `tokenManager.start()`，只在 `signed-in` 时并行取 `GET /api/v0/me` 和 `GET /api/v0/me/profile`（资料进 `userProfile`）；`updateCurrentUser(UserUpdate)`、`changePassword(ChangePasswordRequest)`；`deactivateAccount()` 调 `POST /api/v0/me/deactivate` 后 `tokenManager.endSession()`；`signOut()` 就是 `tokenManager.signOut()`。死成员（`reset`、`isAuthenticated`、`error` 等）删除；
  - `AuthService`（过渡）：删掉 CSRF 表单的 `signOut`；`UserService`（过渡）：`currentUser`、`updateCurrentUser`、`changePassword`、`deactivate` 经 `api` 和 `unwrap`；
  - 删除 `IUser` 和上传头像的弹窗（M5 加回）：用到 `IUser` 的地方改为生成的 `User`，成员类的类型改为 `IUserLite`；general 页的头像和封面只显示、不上传；资料步骤只在名字保存成功后前进（保存失败时不前进，也不会在账户已被另一个标签页换掉时继续）；停用、切换账户的弹窗不再自己跳转，跳转交给 `AuthenticationWrapper`；
  - web 的 oxlint 上限 560 → 555。
- 使用者：Task 8 改写资料 store；Task 9 的包装读会话和账户；Task 10 加上登录、注册。

**Tests:** 没有新的单元测试。stores 的重建和跨标签页的切换由 A5、A6 的页面版本（Task 13）和浏览器核对 C4a–C4c 证明；"重置时重建 `instance`""不订阅 `storage` 事件"的变异让 A5、A6 失败（spec 附录 A）。

- [ ] **Step 1: 令牌管理器和两个客户端的组合**

`web/apps/web/core/lib/auth/api-client.ts`（完整内容）：

```ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { createClient } from "@nerve/api-client";
import { authMiddleware } from "./auth-middleware";
import { leaseLock, webLock } from "./refresh-lock";
import type { RefreshLock } from "./refresh-lock";
import { AUTH_KEY, TokenManager } from "./token-manager";

// The web app's clients of nerve's API and its token manager (M2 design 7.1). The clients send to the
// page's own origin, where nerve serves both the app and the API.

/** The client of the operations that need no token: the instance, sign-in and sign-up, and the token manager's own. */
export const publicClient = createClient();

/** n random bytes as hexadecimal, from crypto.getRandomValues, which pages on plain HTTP have too. */
function randomHex(bytes: number): string {
  return Array.from(crypto.getRandomValues(new Uint8Array(bytes)), (b) => b.toString(16).padStart(2, "0")).join("");
}

/** Calls listener on every change another tab makes to localStorage (a null key: it was cleared). */
function onStorage(listener: (key: string | null, newValue: string | null) => void): () => void {
  const handle = (event: StorageEvent) => {
    if (event.storageArea === localStorage) listener(event.key, event.newValue);
  };
  window.addEventListener("storage", handle);
  return () => window.removeEventListener("storage", handle);
}

/**
 * navigator.locks where the page has it (HTTPS, localhost); else, as on plain HTTP at a LAN address, the
 * lease in localStorage.
 */
function refreshLock(): RefreshLock {
  if ("locks" in navigator) return webLock(navigator.locks);
  return leaseLock({ storage: localStorage, onStorage, now: Date.now, tabId: randomHex(16) });
}

export const tokenManager = new TokenManager({
  storage: localStorage,
  lock: refreshLock(),
  client: publicClient,
  now: Date.now,
  randomHex,
});

// Another tab signed in, signed out or refreshed.
onStorage((key, newValue) => {
  if (key === AUTH_KEY || key === null) tokenManager.handleStorageChange(newValue);
});

// The session is decided once, as the app loads: before the stores exist, so they start with it.
void tokenManager.start();

/** The client of every other operation: the access token on each request, and a 401 renewed once. */
export const api = createClient();
api.use(authMiddleware(tokenManager));
```

`web/apps/web/core/lib/store-context.tsx`（完整内容）：

```text
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactElement } from "react";
import { createContext } from "react";
// lib
import { tokenManager } from "@/lib/auth/api-client";
// store
import { RootStore } from "@/store/root.store";

export let rootStore = new RootStore();

export const StoreContext = createContext<RootStore>(rootStore);

const initializeStore = () => {
  const newRootStore = rootStore ?? new RootStore();
  if (typeof window === "undefined") return newRootStore;
  if (!rootStore) rootStore = newRootStore;
  return newRootStore;
};

export const store = initializeStore();

// A tab that signs out, or follows another tab's sign-in as another account, starts again with new stores
// (M2 design 7.1): nothing of the account it showed stays on screen. A sign-in after a sign-out finds them
// new already.
let loginId = tokenManager.state.loginId;
tokenManager.subscribe(() => {
  const next = tokenManager.state.loginId;
  if (loginId !== undefined && next !== loginId) store.resetOnSignOut();
  loginId = next;
});

export function StoreProvider({ children }: { children: ReactElement }) {
  return <StoreContext.Provider value={store}>{children}</StoreContext.Provider>;
}
```

`web/apps/web/core/store/root.store.ts`（对 `f27434c` 的差异）：

```diff
--- a/web/apps/web/core/store/root.store.ts
+++ b/web/apps/web/core/store/root.store.ts
@@ -109,13 +109,17 @@
     this.powerK = new PowerKStore();
   }
 
+  /**
+   * Forgets the account's data when the session ends or changes to another account, while the page stays
+   * (M2 design 7.1). The instance's information and the address's parameters are not the account's: the
+   * page goes on with them, and nothing would fetch or set them again until the next page load or
+   * navigation.
+   */
   resetOnSignOut() {
     // handling the system theme when user logged out from the app
     localStorage.setItem("theme", "system");
     void setLanguage(FALLBACK_LANGUAGE);
-    this.router = new RouterStore();
     this.commandPalette = new CommandPaletteStore();
-    this.instance = new InstanceStore();
     this.user = new UserStore(this);
     this.workspaceRoot = new WorkspaceRootStore(this);
     this.projectRoot = new ProjectRootStore(this);
```

- [ ] **Step 2: 账户的 store 和 services**

`web/apps/web/core/services/auth.service.ts`（完整内容）：

```ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// types
import type { ICsrfTokenData } from "@nerve/types";
// services
import { APIService } from "@/services/api.service";

export class AuthService extends APIService {
  async requestCSRFToken(): Promise<ICsrfTokenData> {
    return this.get("/auth/get-csrf-token/")
      .then((response) => response.data)
      .catch((error) => {
        throw error;
      });
  }
}
```

`web/apps/web/core/services/user.service.ts`（完整内容）：

```ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// services
import type { ChangePasswordRequest, User, UserUpdate } from "@nerve/api-client";
import type { IUserSettings, TIssuesResponse, TUserProfile } from "@nerve/types";
import { unwrap } from "@/lib/api-error";
import { api } from "@/lib/auth/api-client";
import { APIService } from "@/services/api.service";

export class UserService extends APIService {
  async currentUser(): Promise<User> {
    return unwrap(await api.GET("/api/v0/me"));
  }

  async updateCurrentUser(data: UserUpdate): Promise<User> {
    return unwrap(await api.PATCH("/api/v0/me", { body: data }));
  }

  async changePassword(data: ChangePasswordRequest): Promise<void> {
    unwrap(await api.POST("/api/v0/me/change-password", { body: data }));
  }

  async deactivate(): Promise<void> {
    unwrap(await api.POST("/api/v0/me/deactivate"));
  }

  async getCurrentUserProfile(): Promise<TUserProfile> {
    return this.get("/api/users/me/profile/")
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response;
      });
  }
  async updateCurrentUserProfile(data: any): Promise<any> {
    return this.patch("/api/users/me/profile/", data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response;
      });
  }

  async currentUserSettings(bustCache: boolean = false): Promise<IUserSettings> {
    const url = bustCache ? `/api/users/me/settings/?t=${Date.now()}` : "/api/users/me/settings/";
    return this.get(url)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response;
      });
  }

  async updateUserOnBoard(): Promise<any> {
    return this.patch("/api/users/me/onboard/", {
      is_onboarded: true,
    })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async updateUserTourCompleted(): Promise<any> {
    return this.patch("/api/users/me/tour-completed/", {
      is_tour_completed: true,
    })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async getUserProfileIssues(
    workspaceSlug: string,
    userId: string,
    params: any,
    config = {}
  ): Promise<TIssuesResponse> {
    return this.get(
      `/api/workspaces/${workspaceSlug}/user-issues/${userId}/`,
      {
        params,
      },
      config
    )
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async leaveWorkspace(workspaceSlug: string) {
    return this.post(`/api/workspaces/${workspaceSlug}/members/leave/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async joinProject(workspaceSlug: string, project_ids: string[]): Promise<any> {
    return this.post(`/api/users/me/workspaces/${workspaceSlug}/projects/invitations/`, { project_ids })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async leaveProject(workspaceSlug: string, projectId: string) {
    return this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/members/leave/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }
}

const userService = new UserService();

export default userService;
```

`web/apps/web/core/store/user/index.ts`（完整内容）：

```ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { action, makeObservable, observable, runInAction, computed } from "mobx";
// nerve imports
import type { ChangePasswordRequest, User, UserUpdate } from "@nerve/api-client";
import { EUserPermissions } from "@nerve/constants";
import type { TUserPermissions } from "@nerve/types";
// lib
import { tokenManager } from "@/lib/auth/api-client";
// store
import type { RootStore } from "@/store/root.store";
import type { IUserPermissionStore } from "@/store/user/permissions.store";
import { UserPermissionStore } from "@/store/user/permissions.store";
// services
import { UserService } from "@/services/user.service";
// stores
import type { IUserProfileStore } from "@/store/user/profile.store";
import { ProfileStore } from "@/store/user/profile.store";
// local imports
import type { IUserSettingsStore } from "./settings.store";
import { UserSettingsStore } from "./settings.store";

export interface IUserStore {
  // observables
  isLoading: boolean;
  data: User | undefined;
  // store observables
  userProfile: IUserProfileStore;
  userSettings: IUserSettingsStore;
  permission: IUserPermissionStore;
  // actions
  fetchCurrentUser: () => Promise<User | undefined>;
  updateCurrentUser: (data: UserUpdate) => Promise<User>;
  deactivateAccount: () => Promise<void>;
  changePassword: (payload: ChangePasswordRequest) => Promise<void>;
  signOut: () => Promise<void>;
  // computed
  canPerformAnyCreateAction: boolean;
  projectsWithCreatePermissions: { [projectId: string]: number } | null;
}

export class UserStore implements IUserStore {
  // observables
  isLoading: boolean = false;
  data: User | undefined = undefined;
  // store observables
  userProfile: IUserProfileStore;
  userSettings: IUserSettingsStore;
  permission: IUserPermissionStore;
  // service
  userService: UserService;

  constructor(private store: RootStore) {
    // stores
    this.userProfile = new ProfileStore(store);
    this.userSettings = new UserSettingsStore();
    this.permission = new UserPermissionStore(store);
    // service
    this.userService = new UserService();
    // observables
    makeObservable(this, {
      // observables
      isLoading: observable.ref,
      // model observables
      data: observable,
      userProfile: observable,
      userSettings: observable,
      permission: observable,
      // actions
      fetchCurrentUser: action,
      updateCurrentUser: action,
      deactivateAccount: action,
      changePassword: action,
      signOut: action,
      // computed
      canPerformAnyCreateAction: computed,
      projectsWithCreatePermissions: computed,
    });
  }

  /**
   * @description fetches the account and its profile, once the session is decided: without one it asks
   * nerve nothing (M2 design 7.1). The workspaces come with M3 (M2 design 3.1).
   * @returns {Promise<User | undefined>}
   */
  fetchCurrentUser = async (): Promise<User | undefined> => {
    await tokenManager.start();
    if (tokenManager.state.status !== "signed-in") return undefined;
    runInAction(() => {
      this.isLoading = true;
    });
    try {
      const [user] = await Promise.all([this.userService.currentUser(), this.userProfile.fetchUserProfile()]);
      runInAction(() => {
        this.data = user;
      });
      return user;
    } finally {
      runInAction(() => {
        this.isLoading = false;
      });
    }
  };

  /**
   * @description updates the account's names or time zone
   * @returns {Promise<User>}
   */
  updateCurrentUser = async (data: UserUpdate): Promise<User> => {
    const user = await this.userService.updateCurrentUser(data);
    runInAction(() => {
      this.data = user;
    });
    return user;
  };

  changePassword = async (payload: ChangePasswordRequest): Promise<void> => {
    await this.userService.changePassword(payload);
  };

  /**
   * @description deactivates the account; nerve ends all its sessions, and this browser forgets its own
   * @returns {Promise<void>}
   */
  deactivateAccount = async (): Promise<void> => {
    await this.userService.deactivate();
    await tokenManager.endSession();
  };

  /**
   * @description signs out this browser's session; the stores start again when the session ends
   * @returns {Promise<void>}
   */
  signOut = async (): Promise<void> => {
    await tokenManager.signOut();
  };

  // helper actions
  /**
   * @description fetches the projects with write permissions
   * @returns {{[projectId: string]: number} || null}
   */
  fetchProjectsWithCreatePermissions = (): { [key: string]: TUserPermissions } => {
    const { workspaceSlug } = this.store.router;

    const allWorkspaceProjectRoles = this.permission.getProjectRolesByWorkspaceSlug(workspaceSlug || "");

    const userPermissions =
      (allWorkspaceProjectRoles &&
        Object.keys(allWorkspaceProjectRoles)
          .filter((key) => allWorkspaceProjectRoles[key] >= EUserPermissions.MEMBER)
          .reduce(
            (res: { [projectId: string]: number }, key: string) => ((res[key] = allWorkspaceProjectRoles[key]), res),
            {}
          )) ||
      null;

    return userPermissions;
  };

  /**
   * @description returns projects where user has permissions
   * @returns {{[projectId: string]: number} || null}
   */
  get projectsWithCreatePermissions() {
    return this.fetchProjectsWithCreatePermissions();
  }

  /**
   * @description returns true if user has permissions to write in any project
   * @returns {boolean}
   */
  get canPerformAnyCreateAction() {
    const filteredProjects = this.fetchProjectsWithCreatePermissions();
    return filteredProjects ? Object.keys(filteredProjects).length > 0 : false;
  }
}
```

- [ ] **Step 3: 删除 `IUser` 和头像上传**

```bash
git rm web/apps/web/core/components/core/modals/user-image-upload-modal.tsx
```

`web/packages/types/src/users.ts`（对 `f27434c` 的差异）：

```diff
--- a/web/packages/types/src/users.ts
+++ b/web/packages/types/src/users.ts
@@ -30,23 +30,6 @@
   last_name: string;
   joining_date?: string;
 }
-export interface IUser extends IUserLite {
-  // only for uploading the cover image
-  cover_image_asset?: string | null;
-  cover_image?: string | null;
-  // only for rendering the cover image
-  cover_image_url: string | null;
-  date_joined: string;
-  email: string;
-  is_active: boolean;
-  is_email_verified: boolean;
-  is_tour_completed: boolean;
-  mobile_number: string | null;
-  last_workspace_id: string;
-  user_timezone: string;
-  username: string;
-  theme: IUserTheme;
-}
 
 export type TUserProfile = {
   id: string | undefined;
```

`web/packages/types/src/project/projects.ts`（对 `f27434c` 的差异）：

```diff
--- a/web/packages/types/src/project/projects.ts
+++ b/web/packages/types/src/project/projects.ts
@@ -7,7 +7,7 @@
 import type { TLogoProps } from "../common";
 import type { TUserPermissions } from "../enums";
 import type { TStateGroups } from "../state";
-import type { IUser, IUserLite } from "../users";
+import type { IUserLite } from "../users";
 import type { IWorkspace } from "../workspace";
 
 export enum EUserProjectRoles {
@@ -48,7 +48,7 @@
   cover_image?: string;
   // only for rendering the cover image
   readonly cover_image_url?: string;
-  default_assignee?: IUser | string | null;
+  default_assignee?: IUserLite | string | null;
   description?: string;
   is_favorite?: boolean;
   members?: string[];
```

`web/packages/types/src/search.ts`（对 `f27434c` 的差异）：

```diff
--- a/web/packages/types/src/search.ts
+++ b/web/packages/types/src/search.ts
@@ -8,15 +8,15 @@
 import type { TIssue } from "./issues/issue";
 import type { IModule } from "./module";
 import type { IProject } from "./project";
-import type { IUser } from "./users";
+import type { IUserLite } from "./users";
 import type { IWorkspace } from "./workspace";
 
 export type TSearchEntities = "user_mention" | "issue" | "project" | "cycle" | "module";
 
 export type TUserSearchResponse = {
-  member__avatar_url: IUser["avatar_url"];
-  member__display_name: IUser["display_name"];
-  member__id: IUser["id"];
+  member__avatar_url: IUserLite["avatar_url"];
+  member__display_name: IUserLite["display_name"];
+  member__id: IUserLite["id"];
 };
 
 type TProjectSearchResponse = {
```

`web/packages/types/src/workspace.ts`（对 Task 5 版本的差异）：

```diff
--- a/web/packages/types/src/workspace.ts
+++ b/web/packages/types/src/workspace.ts
@@ -6,7 +6,7 @@
 
 import type { TUserPermissions } from "./enums";
 import type { TProjectMembership } from "./project";
-import type { IUser, IUserLite } from "./users";
+import type { IUserLite } from "./users";
 import type { IWorkspaceViewProps } from "./view-props";
 
 export enum EUserWorkspaceRoles {
@@ -17,7 +17,7 @@
 
 export interface IWorkspace {
   readonly id: string;
-  readonly owner: IUser;
+  readonly owner: IUserLite;
   readonly created_at: Date;
   readonly updated_at: Date;
   name: string;
```

`web/apps/web/core/components/home/user-greetings.tsx`（对 `f27434c` 的差异）：

```diff
--- a/web/apps/web/core/components/home/user-greetings.tsx
+++ b/web/apps/web/core/components/home/user-greetings.tsx
@@ -6,12 +6,12 @@
 
 // nerve types
 import { useTranslation } from "@nerve/i18n";
-import type { IUser } from "@nerve/types";
+import type { User } from "@nerve/api-client";
 // hooks
 import { useCurrentTime } from "@/hooks/use-current-time";
 
 export interface IUserGreetingsView {
-  user: IUser;
+  user: User;
 }
 
 export function UserGreetingsView(props: IUserGreetingsView) {
```

`web/apps/web/core/components/issues/issue-detail/reactions/issue.tsx`（对 `f27434c` 的差异）：

```diff
--- a/web/apps/web/core/components/issues/issue-detail/reactions/issue.tsx
+++ b/web/apps/web/core/components/issues/issue-detail/reactions/issue.tsx
@@ -10,7 +10,7 @@
 import { EmojiReactionGroup, EmojiReactionPicker } from "@nerve/propel/emoji-reaction";
 import type { EmojiReactionType } from "@nerve/propel/emoji-reaction";
 import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
-import type { IUser } from "@nerve/types";
+import type { User } from "@nerve/api-client";
 // ui
 import { cn } from "@nerve/utils";
 // helpers
@@ -21,7 +21,7 @@
   workspaceSlug: string;
   projectId: string;
   issueId: string;
-  currentUser: IUser;
+  currentUser: User;
   disabled?: boolean;
   className?: string;
 };
```

`web/apps/web/core/components/project/settings/member-columns.tsx`（对 `f27434c` 的差异）：

```diff
--- a/web/apps/web/core/components/project/settings/member-columns.tsx
+++ b/web/apps/web/core/components/project/settings/member-columns.tsx
@@ -12,7 +12,8 @@
 // nerve imports
 import { ROLE, EUserPermissions } from "@nerve/constants";
 import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
-import type { EUserProjectRoles, IUser, IWorkspaceMember, TProjectMembership } from "@nerve/types";
+import type { User } from "@nerve/api-client";
+import type { EUserProjectRoles, IWorkspaceMember, TProjectMembership } from "@nerve/types";
 import { CustomMenu, CustomSelect } from "@nerve/ui";
 import { getFileURL } from "@nerve/utils";
 // hooks
@@ -27,7 +28,7 @@
   rowData: RowData;
   workspaceSlug: string;
   isAdmin: boolean;
-  currentUser: IUser | undefined;
+  currentUser: User | undefined;
   setRemoveMemberModal: (rowData: RowData) => void;
 };
 
```

`web/apps/web/core/components/workspace/settings/member-columns.tsx`（对 `f27434c` 的差异）：

```diff
--- a/web/apps/web/core/components/workspace/settings/member-columns.tsx
+++ b/web/apps/web/core/components/workspace/settings/member-columns.tsx
@@ -14,7 +14,8 @@
 import { DeactivatedUserOutline, DeleteOutline } from "@makeplane/propel/icons";
 import { Pill, EPillVariant, EPillSize } from "@nerve/propel/pill";
 import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
-import type { IUser, IWorkspaceMember } from "@nerve/types";
+import type { User } from "@nerve/api-client";
+import type { IWorkspaceMember } from "@nerve/types";
 // nerve ui
 import { CustomSelect, PopoverMenu } from "@nerve/ui";
 // helpers
@@ -33,7 +34,7 @@
   rowData: RowData;
   workspaceSlug: string;
   isAdmin: boolean;
-  currentUser: IUser | undefined;
+  currentUser: User | undefined;
   setRemoveMemberModal: (rowData: RowData) => void;
 };
 
```

`web/apps/web/core/components/onboarding/steps/workspace/create.tsx`（对 Task 6 版本的差异）：

```diff
--- a/web/apps/web/core/components/onboarding/steps/workspace/create.tsx
+++ b/web/apps/web/core/components/onboarding/steps/workspace/create.tsx
@@ -13,7 +13,8 @@
 import { useTranslation } from "@nerve/i18n";
 import { Button } from "@nerve/propel/button";
 import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
-import type { IUser, IWorkspace } from "@nerve/types";
+import type { User } from "@nerve/api-client";
+import type { IWorkspace } from "@nerve/types";
 import { Spinner } from "@nerve/ui";
 import { cn, validateWorkspaceName, validateSlug } from "@nerve/utils";
 // hooks
@@ -26,7 +27,7 @@
 import { CommonOnboardingHeader } from "../common";
 
 type Props = {
-  user: IUser | undefined;
+  user: User | undefined;
   onComplete: (skipInvites?: boolean) => void;
   handleCurrentViewChange: () => void;
   hasInvitations?: boolean;
```

`web/apps/web/core/components/onboarding/steps/profile/root.tsx`（对 Task 5 版本的差异）：

```diff
--- a/web/apps/web/core/components/onboarding/steps/profile/root.tsx
+++ b/web/apps/web/core/components/onboarding/steps/profile/root.tsx
@@ -7,9 +7,9 @@
 import { observer } from "mobx-react";
 import { Controller, useForm } from "react-hook-form";
 // nerve imports
+import type { UserUpdate } from "@nerve/api-client";
 import { Button } from "@nerve/propel/button";
 import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
-import type { IUser } from "@nerve/types";
 import { EOnboardingSteps } from "@nerve/types";
 import { cn, getFileURL, validatePersonName } from "@nerve/utils";
 // hooks
@@ -43,26 +43,28 @@
     mode: "onChange",
   });
 
-  const handleSubmitUserDetail = async (formData: TProfileSetupFormValues) => {
-    const userDetailsPayload: Partial<IUser> = {
+  /** Saves the names; false when nerve did not, so the step stays (M2 design 7.1: nor for another account). */
+  const handleSubmitUserDetail = async (formData: TProfileSetupFormValues): Promise<boolean> => {
+    const userDetailsPayload: UserUpdate = {
       first_name: formData.first_name,
       last_name: formData.last_name,
     };
     try {
       await updateCurrentUser(userDetailsPayload);
+      return true;
     } catch {
       setToast({
         type: TOAST_TYPE.ERROR,
         title: "Error",
         message: "User details update failed. Please try again!",
       });
+      return false;
     }
   };
 
   const onSubmit = async (formData: TProfileSetupFormValues) => {
     if (!user) return;
-    await handleSubmitUserDetail(formData);
-    handleStepChange(EOnboardingSteps.PROFILE_SETUP);
+    if (await handleSubmitUserDetail(formData)) handleStepChange(EOnboardingSteps.PROFILE_SETUP);
   };
 
   const isButtonDisabled = isSubmitting || !isValid;
```

`web/apps/web/core/components/onboarding/switch-account-modal.tsx`（对 `f27434c` 的差异）：

```diff
--- a/web/apps/web/core/components/onboarding/switch-account-modal.tsx
+++ b/web/apps/web/core/components/onboarding/switch-account-modal.tsx
@@ -14,7 +14,6 @@
 import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
 // hooks
 import { useUser } from "@/hooks/store/user";
-import { useNavigate } from "react-router";
 
 type Props = {
   isOpen: boolean;
@@ -25,8 +24,6 @@
   const { isOpen, onClose } = props;
   // states
   const [switchingAccount, setSwitchingAccount] = useState(false);
-  // router
-  const navigate = useNavigate();
   // store hooks
   const { data: userData, signOut } = useUser();
 
@@ -43,7 +40,6 @@
     await signOut()
       .then(() => {
         setTheme("system");
-        navigate("/");
         handleClose();
       })
       .catch(() =>
```

`web/apps/web/core/components/account/deactivate-account-modal.tsx`（对 `f27434c` 的差异）：

```diff
--- a/web/apps/web/core/components/account/deactivate-account-modal.tsx
+++ b/web/apps/web/core/components/account/deactivate-account-modal.tsx
@@ -13,7 +13,6 @@
 import { EModalPosition, EModalWidth, ModalCore } from "@nerve/ui";
 // hooks
 import { useUser } from "@/hooks/store/user";
-import { useNavigate } from "react-router";
 
 type Props = {
   isOpen: boolean;
@@ -21,11 +20,10 @@
 };
 
 export function DeactivateAccountModal(props: Props) {
-  const navigate = useNavigate();
   const { isOpen, onClose } = props;
   // hooks
   const { t } = useTranslation();
-  const { deactivateAccount, signOut } = useUser();
+  const { deactivateAccount } = useUser();
 
   // states
   const [isDeactivating, setIsDeactivating] = useState(false);
@@ -45,8 +43,6 @@
           title: "Success!",
           message: "Account deactivated successfully.",
         });
-        signOut();
-        navigate("/");
         handleClose();
         return;
       })
```

`web/apps/web/core/components/settings/profile/content/pages/general/form.tsx`（完整内容）：

```text
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
import { Controller, useForm } from "react-hook-form";
import { UserOutline } from "@makeplane/propel/icons";
// nerve imports
import { Field } from "@makeplane/propel/components/field";
import { Input, InputGroup } from "@makeplane/propel/components/input";
import type { User, UserUpdate } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { setPromiseToast } from "@nerve/propel/toast";
import type { TUserProfile } from "@nerve/types";

import { getFileURL } from "@nerve/utils";
// components
import { DeactivateAccountModal } from "@/components/account/deactivate-account-modal";
import { CoverImage } from "@/components/common/cover-image";
import { SettingsBoxedControlItem } from "@/components/settings/boxed-control-item";
// hooks
import { useUser, useUserProfile } from "@/hooks/store/user";
// utils
import { validatePersonName, validateDisplayName } from "@nerve/utils";

type TUserProfileForm = {
  first_name: string;
  last_name: string;
  display_name: string;
  email: string;
  role: string;
  language: string;
  user_timezone: string;
};

type Props = {
  user: User;
  profile: TUserProfile;
};

export const GeneralProfileSettingsForm = observer(function GeneralProfileSettingsForm(props: Props) {
  const { user, profile } = props;
  // states
  const [isLoading, setIsLoading] = useState(false);
  const [deactivateAccountModal, setDeactivateAccountModal] = useState(false);
  // language support
  const { t } = useTranslation();
  // form info
  const {
    handleSubmit,
    watch,
    control,
    formState: { errors },
  } = useForm<TUserProfileForm>({
    defaultValues: {
      first_name: user.first_name || "",
      last_name: user.last_name || "",
      display_name: user.display_name || "",
      email: user.email || "",
      role: profile.role || "Product / Project Manager",
      language: profile.language || "en",
      user_timezone: user.user_timezone || "Asia/Kolkata",
    },
  });
  // store hooks
  const { data: currentUser, updateCurrentUser } = useUser();
  const { updateUserProfile } = useUserProfile();

  const onSubmit = async (formData: TUserProfileForm) => {
    setIsLoading(true);
    const userPayload: UserUpdate = {
      first_name: formData.first_name,
      last_name: formData.last_name,
      display_name: formData?.display_name,
    };

    const profilePayload: Partial<TUserProfile> = {
      role: formData.role,
    };

    const updateCurrentUserDetail = updateCurrentUser(userPayload);
    const promises: Promise<User | TUserProfile | undefined>[] = [updateCurrentUserDetail];
    if (profilePayload.role !== profile.role) {
      const updateCurrentUserProfile = updateUserProfile(profilePayload);
      promises.push(updateCurrentUserProfile);
    }

    const updatePromise = Promise.allSettled(promises)
      .then((results) => {
        const rejectedResult = results.find((result) => result.status === "rejected") as
          | PromiseRejectedResult
          | undefined;
        if (rejectedResult) {
          throw rejectedResult.reason ?? new Error("Failed to update profile");
        }
        const values = results.map(
          (result) => (result as PromiseFulfilledResult<User | TUserProfile | undefined>).value
        );
        if (values.some((v) => v === undefined)) {
          throw new Error("Failed to update profile");
        }
        return values;
      })
      .finally(() => setIsLoading(false));

    setPromiseToast(updatePromise, {
      loading: "Updating...",
      success: {
        title: "Success!",
        message: () => `Profile updated successfully.`,
      },
      error: {
        title: "Error!",
        message: () => `There was some error in updating your profile. Please try again.`,
      },
    });
  };

  return (
    <>
      <DeactivateAccountModal isOpen={deactivateAccountModal} onClose={() => setDeactivateAccountModal(false)} />
      <form onSubmit={handleSubmit(onSubmit)} className="w-full">
        <div className="flex w-full flex-col gap-7">
          {/* The picture and the cover are shown, not uploaded, until the file storage (M5, M2 design 3.2) */}
          <div className="relative h-44 w-full">
            <CoverImage
              src={user.cover_image_url ?? undefined}
              className="h-44 w-full rounded-lg"
              alt={currentUser?.first_name ?? "Cover image"}
            />
            <div className="absolute -bottom-6 left-6 flex items-end justify-between">
              <div className="flex gap-3">
                <div className="flex h-16 w-16 items-center justify-center rounded-lg bg-surface-2">
                  {user.avatar_url ? (
                    <div className="relative h-16 w-16 overflow-hidden">
                      <img
                        src={getFileURL(user.avatar_url)}
                        className="absolute top-0 left-0 h-full w-full rounded-lg object-cover"
                        alt={currentUser?.display_name}
                      />
                    </div>
                  ) : (
                    <div className="h-16 w-16 rounded-md bg-layer-1 p-2">
                      <UserOutline className="h-full w-full text-secondary" />
                    </div>
                  )}
                </div>
              </div>
            </div>
          </div>
          <div className="item-center mt-6 flex justify-between">
            <div className="flex flex-col">
              <div className="item-center flex text-16 font-medium text-secondary">
                <span>{`${watch("first_name")} ${watch("last_name")}`}</span>
              </div>
              <span className="text-13 tracking-tight text-tertiary">{watch("email")}</span>
            </div>
          </div>
          <div className="flex flex-col gap-2">
            <div className="grid grid-cols-1 gap-x-6 gap-y-4 sm:grid-cols-2 xl:grid-cols-3">
              <div className="flex flex-col gap-1">
                <h4 className="text-13 font-medium text-secondary">
                  {t("first_name")}&nbsp;
                  <span className="text-danger-primary">*</span>
                </h4>
                <Controller
                  control={control}
                  name="first_name"
                  rules={{
                    required: "Please enter first name",
                    validate: validatePersonName,
                  }}
                  render={({ field: { value, onChange, ref } }) => (
                    <Field name="first_name" invalid={Boolean(errors.first_name)}>
                      <InputGroup size="2xl">
                        <Input
                          size="2xl"
                          id="first_name"
                          name="first_name"
                          type="text"
                          value={value}
                          onChange={onChange}
                          ref={ref}
                          placeholder="Enter your first name"
                          maxLength={50}
                          autoComplete="on"
                        />
                      </InputGroup>
                    </Field>
                  )}
                />
                {errors.first_name && <span className="text-11 text-danger-primary">{errors.first_name.message}</span>}
              </div>
              <div className="flex flex-col gap-1">
                <h4 className="text-13 font-medium text-secondary">{t("last_name")}</h4>
                <Controller
                  control={control}
                  name="last_name"
                  rules={{
                    validate: validatePersonName,
                  }}
                  render={({ field: { value, onChange, ref } }) => (
                    <Field name="last_name" invalid={Boolean(errors.last_name)}>
                      <InputGroup size="2xl">
                        <Input
                          size="2xl"
                          id="last_name"
                          name="last_name"
                          type="text"
                          value={value}
                          onChange={onChange}
                          ref={ref}
                          placeholder="Enter your last name"
                          maxLength={50}
                          autoComplete="on"
                        />
                      </InputGroup>
                    </Field>
                  )}
                />
                {errors.last_name && <span className="text-11 text-danger-primary">{errors.last_name.message}</span>}
              </div>
              <div className="flex flex-col gap-1">
                <h4 className="text-13 font-medium text-secondary">
                  {t("display_name")}&nbsp;
                  <span className="text-danger-primary">*</span>
                </h4>
                <Controller
                  control={control}
                  name="display_name"
                  rules={{
                    required: "Display name is required.",
                    validate: validateDisplayName,
                  }}
                  render={({ field: { value, onChange, ref } }) => (
                    <Field name="display_name" invalid={Boolean(errors?.display_name)}>
                      <InputGroup size="2xl">
                        <Input
                          size="2xl"
                          id="display_name"
                          name="display_name"
                          type="text"
                          value={value}
                          onChange={onChange}
                          ref={ref}
                          placeholder="Enter your display name"
                          maxLength={50}
                        />
                      </InputGroup>
                    </Field>
                  )}
                />
                {errors?.display_name && (
                  <span className="text-11 text-danger-primary">{errors?.display_name?.message}</span>
                )}
              </div>
              <div className="flex flex-col gap-1">
                <h4 className="text-13 font-medium text-secondary">
                  {t("auth.common.email.label")}&nbsp;
                  <span className="text-danger-primary">*</span>
                </h4>
                <Controller
                  control={control}
                  name="email"
                  rules={{
                    required: "Email is required.",
                  }}
                  render={({ field: { value, ref } }) => (
                    <Field name="email" invalid={Boolean(errors.email)}>
                      <InputGroup size="2xl">
                        <Input
                          size="2xl"
                          id="email"
                          name="email"
                          type="email"
                          value={value}
                          ref={ref}
                          placeholder="Enter your email"
                          autoComplete="on"
                          disabled
                        />
                      </InputGroup>
                    </Field>
                  )}
                />
              </div>
            </div>
          </div>
          <div>
            <Button variant="primary" type="submit" loading={isLoading}>
              {isLoading ? t("saving") : t("save_changes")}
            </Button>
          </div>
        </div>
      </form>
      <div className="mt-10">
        <SettingsBoxedControlItem
          title={t("deactivate_account")}
          description={t("deactivate_account_description")}
          control={
            <Button variant="error-outline" onClick={() => setDeactivateAccountModal(true)}>
              {t("deactivate_account")}
            </Button>
          }
        />
      </div>
    </>
  );
});
```

`web/apps/web/core/components/settings/profile/content/pages/security.tsx`（对 `f27434c` 的差异）：

```diff
--- a/web/apps/web/core/components/settings/profile/content/pages/security.tsx
+++ b/web/apps/web/core/components/settings/profile/content/pages/security.tsx
@@ -23,8 +23,6 @@
 import { authErrorHandler, EAuthenticationErrorCodes, passwordErrors } from "@/helpers/authentication.helper";
 // hooks
 import { useUser } from "@/hooks/store/user";
-// services
-import { AuthService } from "@/services/auth.service";
 
 export interface FormValues {
   old_password: string;
@@ -38,8 +36,6 @@
   confirm_password: "",
 };
 
-const authService = new AuthService();
-
 const defaultShowPassword = {
   oldPassword: false,
   password: false,
@@ -78,11 +74,8 @@
   const handleChangePassword = async (formData: FormValues) => {
     const { old_password, new_password } = formData;
     try {
-      const csrfToken = await authService.requestCSRFToken().then((data) => data?.csrf_token);
-      if (!csrfToken) throw new Error("csrf token not found");
+      await changePassword({ current_password: old_password, new_password });
 
-      await changePassword(csrfToken, { old_password, new_password });
-
       reset(defaultValues);
       setShowPassword(defaultShowPassword);
       setToast({
```

`web/apps/web/package.json`（对 Task 5 版本的差异）：

```diff
--- a/web/apps/web/package.json
+++ b/web/apps/web/package.json
@@ -7,7 +7,7 @@
   "scripts": {
     "dev": "react-router dev --port 3000",
     "build": "react-router build",
-    "check:lint": "node ../../../tools/lint-cap.mjs 560",
+    "check:lint": "node ../../../tools/lint-cap.mjs 555",
     "check:types": "react-router typegen && tsc --noEmit",
     "check:format": "oxfmt --check .",
     "fix:format": "oxfmt .",
```

Run: `grep -rnw -e IUser -e UserImageUploadModal web/apps/web/app web/apps/web/core web/packages --exclude-dir=node_modules --exclude-dir=dist`
Expected: 没有输出。

- [ ] **Step 4: 检查**

Run: `make lint-web`、`make knip`、`make test-web`
Expected: 都通过；web 的 oxlint 警告正好 555 个。

- [ ] **Step 5: 提交**

```bash
git add -A web/apps/web/core web/packages/types/src web/apps/web/package.json
```
```bash
git commit -m "feat(M2/P4): the account store on the generated User; the stores start again with the session

The web app's token manager and clients: navigator.locks or the lease,
storage events, the auth middleware on every other call. The user store
fetches /me and /me/profile only when signed in, and signs out and
deactivates through the token manager. When the session ends or changes
to another account the stores start again; the instance and the router
stay. IUser and the avatar upload modal are gone, and the profile step
moves on only once the name is saved.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** Step 3 的 `grep` 没有输出；上限 555；检查都通过。

---

### Task 8: 资料 store、主题和偏好用 `Profile`、`Theme`；删除 `TUserProfile`、`IUserTheme`

**Files:**
- Modify: `web/apps/web/core/store/user/profile.store.ts`、`web/apps/web/core/services/user.service.ts`、`web/packages/types/src/users.ts`
- Modify: `web/packages/constants/package.json`、`src/themes.ts`、`pnpm-lock.yaml`
- Modify: `web/apps/web/core/components/appearance/theme-switcher.tsx`、`power-k/config/preferences-commands.ts`、`profile/start-of-week-preference.tsx`、`settings/profile/content/pages/general/form.tsx`、`general/root.tsx`、`preferences/language-and-timezone-list.tsx`、`onboarding/root.tsx`、`web/apps/web/core/lib/wrappers/store-wrapper.tsx`
- Modify（过渡）: `web/apps/web/core/lib/wrappers/authentication-wrapper.tsx`、`tools/keywords.json`、`web/apps/web/package.json`

**Interfaces:**
- Produces（spec 2.8，M2 设计 7.5；M1-P2 交接）：
  - `ProfileStore.data: Profile | undefined`；`fetchUserProfile()`（按资料的语言切换界面语言）；`updateUserProfile(ProfileUpdate)` 失败时抛出（Plane 版本吞掉错误，调用方的提示从不出现）；`finishUserOnboarding()` 一次 `PATCH /api/v0/me/profile`（四个步骤、`is_onboarded`，有工作区时带 `last_workspace_id`）；`updateTourCompleted()`、`updateUserTheme(theme: Theme)` 用同一个接口；
  - `UserService.getCurrentUserProfile()`、`updateCurrentUserProfile(ProfileUpdate)`；Plane 的 `/api/users/me/profile/`、`/onboard/`、`/tour-completed/` 删除；
  - 删除 `TUserProfile`、`IUserTheme`、`TOnboardingSteps`；新手引导的步骤改为 `OnboardingStepsUpdate` 的部分更新（服务端合并，不再先拼整个对象），失败时提示；
  - `@nerve/constants` 依赖 `@nerve/api-client`，`I_THEME_OPTION.value: Theme`：写错或多出一个主题值都过不了类型检查；
  - general 页的表单只有名、姓、显示名和只读的邮箱（`role` 不在 `Profile` 里，语言、时区在偏好页）；
  - `StoreWrapper` 按账户 id（不再是资料的 id）判断换人，按 `profile.theme` 设主题；
  - 关键词规则 `plane-user-urls`（`/api/users/me/(?:profile|onboard|tour-completed)|/api/instances/`）；
  - web 的 oxlint 上限 555 → 554。
- 使用者：Task 9 的包装读 `Profile` 的 `is_onboarded` 和 `onboarding_step`；P5 的偏好页。

**Tests:** 没有新的单元测试；A10（Task 12）核对资料步骤只把 `profile_complete` 设为真、其余三个键不变。

- [ ] **Step 1: 资料的 store 和 service**

`web/apps/web/core/services/user.service.ts`（完整内容）：

```ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// services
import type { ChangePasswordRequest, Profile, ProfileUpdate, User, UserUpdate } from "@nerve/api-client";
import type { IUserSettings, TIssuesResponse } from "@nerve/types";
import { unwrap } from "@/lib/api-error";
import { api } from "@/lib/auth/api-client";
import { APIService } from "@/services/api.service";

export class UserService extends APIService {
  async currentUser(): Promise<User> {
    return unwrap(await api.GET("/api/v0/me"));
  }

  async updateCurrentUser(data: UserUpdate): Promise<User> {
    return unwrap(await api.PATCH("/api/v0/me", { body: data }));
  }

  async changePassword(data: ChangePasswordRequest): Promise<void> {
    unwrap(await api.POST("/api/v0/me/change-password", { body: data }));
  }

  async deactivate(): Promise<void> {
    unwrap(await api.POST("/api/v0/me/deactivate"));
  }

  async getCurrentUserProfile(): Promise<Profile> {
    return unwrap(await api.GET("/api/v0/me/profile"));
  }

  async updateCurrentUserProfile(data: ProfileUpdate): Promise<Profile> {
    return unwrap(await api.PATCH("/api/v0/me/profile", { body: data }));
  }

  async currentUserSettings(bustCache: boolean = false): Promise<IUserSettings> {
    const url = bustCache ? `/api/users/me/settings/?t=${Date.now()}` : "/api/users/me/settings/";
    return this.get(url)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response;
      });
  }

  async getUserProfileIssues(
    workspaceSlug: string,
    userId: string,
    params: any,
    config = {}
  ): Promise<TIssuesResponse> {
    return this.get(
      `/api/workspaces/${workspaceSlug}/user-issues/${userId}/`,
      {
        params,
      },
      config
    )
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async leaveWorkspace(workspaceSlug: string) {
    return this.post(`/api/workspaces/${workspaceSlug}/members/leave/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async joinProject(workspaceSlug: string, project_ids: string[]): Promise<any> {
    return this.post(`/api/users/me/workspaces/${workspaceSlug}/projects/invitations/`, { project_ids })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async leaveProject(workspaceSlug: string, projectId: string) {
    return this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/members/leave/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }
}

const userService = new UserService();

export default userService;
```

`web/apps/web/core/store/user/profile.store.ts`（完整内容）：

```ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { action, makeObservable, observable, runInAction } from "mobx";
// nerve imports
import type { Profile, ProfileUpdate, Theme } from "@nerve/api-client";
import { setLanguage } from "@nerve/i18n";
// services
import { UserService } from "@/services/user.service";
// store
import type { RootStore } from "../root.store";

export interface IUserProfileStore {
  // observables
  data: Profile | undefined;
  // actions
  fetchUserProfile: () => Promise<Profile>;
  updateUserProfile: (data: ProfileUpdate) => Promise<Profile>;
  finishUserOnboarding: () => Promise<void>;
  updateTourCompleted: () => Promise<Profile>;
  updateUserTheme: (theme: Theme) => Promise<Profile>;
}

export class ProfileStore implements IUserProfileStore {
  data: Profile | undefined = undefined;

  // services
  userService: UserService;

  constructor(public store: RootStore) {
    makeObservable(this, {
      // observables
      data: observable,
      // actions
      fetchUserProfile: action,
      updateUserProfile: action,
      finishUserOnboarding: action,
      updateTourCompleted: action,
      updateUserTheme: action,
    });
    // services
    this.userService = new UserService();
  }

  /**
   * @description fetches the account's profile, and shows the app in its language
   * @returns {Promise<Profile>}
   */
  fetchUserProfile = async (): Promise<Profile> => {
    const profile = await this.userService.getCurrentUserProfile();
    runInAction(() => {
      this.data = profile;
    });
    void setLanguage(profile.language);
    return profile;
  };

  /**
   * @description changes the given fields of the profile (onboarding_step key by key); fails when nerve refuses
   * @returns {Promise<Profile>}
   */
  updateUserProfile = async (data: ProfileUpdate): Promise<Profile> => {
    if (data.language) void setLanguage(data.language);
    const profile = await this.userService.updateCurrentUserProfile(data);
    runInAction(() => {
      this.data = profile;
    });
    return profile;
  };

  /**
   * @description finishes the onboarding in one change of the profile
   * @returns {Promise<void>}
   */
  finishUserOnboarding = async (): Promise<void> => {
    const firstWorkspace = Object.values(this.store.workspaceRoot.workspaces ?? {})[0];
    await this.updateUserProfile({
      onboarding_step: {
        profile_complete: true,
        workspace_join: true,
        workspace_create: true,
        workspace_invite: true,
      },
      is_onboarded: true,
      ...(firstWorkspace ? { last_workspace_id: firstWorkspace.id } : {}),
    });
  };

  /**
   * @description marks the product tour as seen
   * @returns {Promise<Profile>}
   */
  updateTourCompleted = (): Promise<Profile> => this.updateUserProfile({ is_tour_completed: true });

  /**
   * @description changes the theme
   * @returns {Promise<Profile>}
   */
  updateUserTheme = (theme: Theme): Promise<Profile> => this.updateUserProfile({ theme });
}
```

`web/packages/types/src/users.ts`（完整内容）：

```ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TUserPermissions } from "./enums";

/**
 * @description The start of the week for the user
 * @enum {number}
 */
export enum EStartOfTheWeek {
  SUNDAY = 0,
  MONDAY = 1,
  TUESDAY = 2,
  WEDNESDAY = 3,
  THURSDAY = 4,
  FRIDAY = 5,
  SATURDAY = 6,
}

export interface IUserLite {
  avatar_url: string;
  display_name: string;
  email?: string;
  first_name: string;
  id: string;
  is_bot: boolean;
  last_name: string;
  joining_date?: string;
}

export interface IUserSettings {
  id: string | undefined;
  email: string | undefined;
  workspace: {
    last_workspace_id: string | undefined;
    last_workspace_slug: string | undefined;
    last_workspace_name: string | undefined;
    last_workspace_logo: string | undefined;
    fallback_workspace_id: string | undefined;
    fallback_workspace_slug: string | undefined;
    invites: number | undefined;
  };
}

export interface IUserProjectsRole {
  [projectId: string]: TUserPermissions;
}

export type TProfileViews = "assigned" | "created" | "subscribed";
```

- [ ] **Step 2: 主题用生成的 `Theme`**

`web/packages/constants/package.json`（对 `f27434c` 的差异）：

```diff
--- a/web/packages/constants/package.json
+++ b/web/packages/constants/package.json
@@ -21,6 +21,7 @@
     "test": "vitest run"
   },
   "dependencies": {
+    "@nerve/api-client": "workspace:*",
     "@nerve/types": "workspace:*"
   },
   "devDependencies": {
```

Run: `pnpm install`
Expected: `pnpm-lock.yaml` 的变化只有下面这一处：

`pnpm-lock.yaml`（对 Task 2 版本的差异）：

```diff
--- a/pnpm-lock.yaml
+++ b/pnpm-lock.yaml
@@ -547,6 +547,9 @@
 
   web/packages/constants:
     dependencies:
+      '@nerve/api-client':
+        specifier: workspace:*
+        version: link:../api-client
       '@nerve/types':
         specifier: workspace:*
         version: link:../types
```

`web/packages/constants/src/themes.ts`（对 `f27434c` 的差异）：

```diff
--- a/web/packages/constants/src/themes.ts
+++ b/web/packages/constants/src/themes.ts
@@ -4,8 +4,10 @@
  * See the LICENSE file for details.
  */
 
+import type { Theme } from "@nerve/api-client";
+
 export interface I_THEME_OPTION {
-  value: string;
+  value: Theme;
   i18n_label: string;
   type: string;
   icon: {
```

- [ ] **Step 3: 读资料的地方**

`web/apps/web/core/components/appearance/theme-switcher.tsx`（对 `f27434c` 的差异）：

```diff
--- a/web/apps/web/core/components/appearance/theme-switcher.tsx
+++ b/web/apps/web/core/components/appearance/theme-switcher.tsx
@@ -34,16 +34,16 @@
   // derived values
   const currentTheme = useMemo(() => {
     // oxlint-disable-next-line no-shadow
-    const userThemeOption = THEME_OPTIONS.find((t) => t.value === userProfile?.theme?.theme);
+    const userThemeOption = THEME_OPTIONS.find((t) => t.value === userProfile?.theme);
     return userThemeOption || null;
-  }, [userProfile?.theme?.theme]);
+  }, [userProfile?.theme]);
 
   const handleThemeChange = useCallback(
     async (themeOption: I_THEME_OPTION) => {
       try {
         setTheme(themeOption.value);
 
-        const updatePromise = updateUserTheme({ theme: themeOption.value });
+        const updatePromise = updateUserTheme(themeOption.value);
         setPromiseToast(updatePromise, {
           loading: "Updating theme...",
           success: {
```

`web/apps/web/core/components/power-k/config/preferences-commands.ts`（对 `f27434c` 的差异）：

```diff
--- a/web/apps/web/core/components/power-k/config/preferences-commands.ts
+++ b/web/apps/web/core/components/power-k/config/preferences-commands.ts
@@ -9,9 +9,9 @@
 import { Languages } from "lucide-react";
 import { CalendarOutline, GlobeOutline, PaletteOutline } from "@makeplane/propel/icons";
 // nerve imports
+import type { Language, ProfileUpdate, StartOfTheWeek, Theme } from "@nerve/api-client";
 import { useTranslation } from "@nerve/i18n";
 import { setToast, TOAST_TYPE } from "@nerve/propel/toast";
-import type { EStartOfTheWeek, TUserProfile } from "@nerve/types";
 // components
 import type { TPowerKCommandConfig } from "@/components/power-k/core/types";
 // hooks
@@ -29,9 +29,9 @@
   const { t } = useTranslation();
 
   const handleUpdateTheme = useCallback(
-    async (newTheme: string) => {
+    async (newTheme: Theme) => {
       setTheme(newTheme);
-      return updateUserTheme({ theme: newTheme })
+      return updateUserTheme(newTheme)
         .then(() => {
           setToast({
             type: TOAST_TYPE.SUCCESS,
@@ -80,7 +80,7 @@
   );
 
   const handleUpdateUserProfile = useCallback(
-    (payload: Partial<TUserProfile>) => {
+    (payload: ProfileUpdate) => {
       updateUserProfile(payload)
         .then(() => {
           setToast({
@@ -112,7 +112,7 @@
       i18n_title: "power_k.preferences_actions.update_theme",
       icon: PaletteOutline,
       onSelect: (data) => {
-        const theme = data as string;
+        const theme = data as Theme;
         void handleUpdateTheme(theme);
       },
       isEnabled: () => true,
@@ -142,7 +142,7 @@
       i18n_title: "power_k.preferences_actions.update_start_of_week",
       icon: CalendarOutline,
       onSelect: (data) => {
-        const startOfWeek = data as EStartOfTheWeek;
+        const startOfWeek = data as StartOfTheWeek;
         handleUpdateUserProfile({ start_of_the_week: startOfWeek });
       },
       isEnabled: () => true,
@@ -157,7 +157,7 @@
       i18n_title: "power_k.preferences_actions.update_language",
       icon: Languages,
       onSelect: (data) => {
-        const language = data as string;
+        const language = data as Language;
         handleUpdateUserProfile({ language });
       },
       isEnabled: () => true,
```

`web/apps/web/core/components/profile/start-of-week-preference.tsx`（完整内容）：

```text
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import type { StartOfTheWeek } from "@nerve/api-client";
import { START_OF_THE_WEEK_OPTIONS } from "@nerve/constants";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { CustomSelect } from "@nerve/ui";
// components
import { SettingsControlItem } from "@/components/settings/control-item";
// hooks
import { useUserProfile } from "@/hooks/store/user";

const getStartOfWeekLabel = (startOfWeek: StartOfTheWeek | undefined) =>
  START_OF_THE_WEEK_OPTIONS.find((option) => option.value === startOfWeek)?.label;

export const StartOfWeekPreference = observer(function StartOfWeekPreference(props: {
  option: { title: string; description: string };
}) {
  // hooks
  const { data: userProfile, updateUserProfile } = useUserProfile();

  const handleStartOfWeekChange = async (val: StartOfTheWeek) => {
    try {
      await updateUserProfile({ start_of_the_week: val });
      setToast({ type: TOAST_TYPE.SUCCESS, title: "Success", message: "First day of the week updated successfully" });
    } catch (_error) {
      setToast({ type: TOAST_TYPE.ERROR, title: "Update failed", message: "Please try again later." });
    }
  };

  return (
    <SettingsControlItem
      title={props.option.title}
      description={props.option.description}
      control={
        <CustomSelect
          value={userProfile?.start_of_the_week}
          label={getStartOfWeekLabel(userProfile?.start_of_the_week)}
          onChange={handleStartOfWeekChange}
          buttonClassName="border border-subtle-1"
          input
          maxHeight="lg"
          placement="bottom-end"
        >
          <>
            {START_OF_THE_WEEK_OPTIONS.map((day) => (
              <CustomSelect.Option key={day.value} value={day.value}>
                {day.label}
              </CustomSelect.Option>
            ))}
          </>
        </CustomSelect>
      }
    />
  );
});
```

`web/apps/web/core/components/settings/profile/content/pages/general/root.tsx`（完整内容）：

```text
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { useTranslation } from "@nerve/i18n";
// components
import { PageHead } from "@/components/core/page-title";
// hooks
import { useUser } from "@/hooks/store/user";
// local imports
import { GeneralProfileSettingsForm } from "./form";

export const GeneralProfileSettings = observer(function GeneralProfileSettings() {
  const { t } = useTranslation();
  // store hooks
  const { data: currentUser } = useUser();

  if (!currentUser) return null;

  return (
    <>
      <PageHead title={`${t("profile.label")} - ${t("general_settings")}`} />
      <GeneralProfileSettingsForm user={currentUser} />
    </>
  );
});
```

`web/apps/web/core/components/settings/profile/content/pages/general/form.tsx`（对 Task 7 版本的差异）：

```diff
--- a/web/apps/web/core/components/settings/profile/content/pages/general/form.tsx
+++ b/web/apps/web/core/components/settings/profile/content/pages/general/form.tsx
@@ -15,7 +15,6 @@
 import { useTranslation } from "@nerve/i18n";
 import { Button } from "@nerve/propel/button";
 import { setPromiseToast } from "@nerve/propel/toast";
-import type { TUserProfile } from "@nerve/types";
 
 import { getFileURL } from "@nerve/utils";
 // components
@@ -23,7 +22,7 @@
 import { CoverImage } from "@/components/common/cover-image";
 import { SettingsBoxedControlItem } from "@/components/settings/boxed-control-item";
 // hooks
-import { useUser, useUserProfile } from "@/hooks/store/user";
+import { useUser } from "@/hooks/store/user";
 // utils
 import { validatePersonName, validateDisplayName } from "@nerve/utils";
 
@@ -32,18 +31,14 @@
   last_name: string;
   display_name: string;
   email: string;
-  role: string;
-  language: string;
-  user_timezone: string;
 };
 
 type Props = {
   user: User;
-  profile: TUserProfile;
 };
 
 export const GeneralProfileSettingsForm = observer(function GeneralProfileSettingsForm(props: Props) {
-  const { user, profile } = props;
+  const { user } = props;
   // states
   const [isLoading, setIsLoading] = useState(false);
   const [deactivateAccountModal, setDeactivateAccountModal] = useState(false);
@@ -61,14 +56,10 @@
       last_name: user.last_name || "",
       display_name: user.display_name || "",
       email: user.email || "",
-      role: profile.role || "Product / Project Manager",
-      language: profile.language || "en",
-      user_timezone: user.user_timezone || "Asia/Kolkata",
     },
   });
   // store hooks
   const { data: currentUser, updateCurrentUser } = useUser();
-  const { updateUserProfile } = useUserProfile();
 
   const onSubmit = async (formData: TUserProfileForm) => {
     setIsLoading(true);
@@ -78,35 +69,8 @@
       display_name: formData?.display_name,
     };
 
-    const profilePayload: Partial<TUserProfile> = {
-      role: formData.role,
-    };
-
-    const updateCurrentUserDetail = updateCurrentUser(userPayload);
-    const promises: Promise<User | TUserProfile | undefined>[] = [updateCurrentUserDetail];
-    if (profilePayload.role !== profile.role) {
-      const updateCurrentUserProfile = updateUserProfile(profilePayload);
-      promises.push(updateCurrentUserProfile);
-    }
+    const updatePromise = updateCurrentUser(userPayload).finally(() => setIsLoading(false));
 
-    const updatePromise = Promise.allSettled(promises)
-      .then((results) => {
-        const rejectedResult = results.find((result) => result.status === "rejected") as
-          | PromiseRejectedResult
-          | undefined;
-        if (rejectedResult) {
-          throw rejectedResult.reason ?? new Error("Failed to update profile");
-        }
-        const values = results.map(
-          (result) => (result as PromiseFulfilledResult<User | TUserProfile | undefined>).value
-        );
-        if (values.some((v) => v === undefined)) {
-          throw new Error("Failed to update profile");
-        }
-        return values;
-      })
-      .finally(() => setIsLoading(false));
-
     setPromiseToast(updatePromise, {
       loading: "Updating...",
       success: {
```

`web/apps/web/core/components/settings/profile/content/pages/preferences/language-and-timezone-list.tsx`（对 `f27434c` 的差异）：

```diff
--- a/web/apps/web/core/components/settings/profile/content/pages/preferences/language-and-timezone-list.tsx
+++ b/web/apps/web/core/components/settings/profile/content/pages/preferences/language-and-timezone-list.tsx
@@ -6,6 +6,7 @@
 
 import { observer } from "mobx-react";
 // nerve imports
+import type { Language } from "@nerve/api-client";
 import { SUPPORTED_LANGUAGES, toSupportedLanguage, useTranslation } from "@nerve/i18n";
 import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
 import { CustomSelect } from "@nerve/ui";
@@ -45,7 +46,7 @@
       }
     };
 
-    const handleLanguageChange = async (value: string) => {
+    const handleLanguageChange = async (value: Language) => {
       try {
         await updateUserProfile({ language: value });
         setToast({
```

`web/apps/web/core/components/onboarding/root.tsx`（对 Task 5 版本的差异）：

```diff
--- a/web/apps/web/core/components/onboarding/root.tsx
+++ b/web/apps/web/core/components/onboarding/root.tsx
@@ -7,8 +7,9 @@
 import { useCallback, useEffect, useState } from "react";
 import { observer } from "mobx-react";
 // nerve imports
+import type { OnboardingStepsUpdate } from "@nerve/api-client";
 import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
-import type { IWorkspaceMemberInvitation, TOnboardingStep, TOnboardingSteps, TUserProfile } from "@nerve/types";
+import type { IWorkspaceMemberInvitation, TOnboardingStep } from "@nerve/types";
 import { EOnboardingSteps } from "@nerve/types";
 // hooks
 import { useWorkspace } from "@/hooks/store/use-workspace";
@@ -47,21 +48,21 @@
     }
   }, [user, finishUserOnboarding]);
 
-  // handle step change
+  // handle step change: nerve merges the steps it is given into the profile's
   const stepChange = useCallback(
-    async (steps: Partial<TOnboardingSteps>) => {
+    async (steps: OnboardingStepsUpdate) => {
       if (!user) return;
-
-      const payload: Partial<TUserProfile> = {
-        onboarding_step: {
-          ...userProfile.onboarding_step,
-          ...steps,
-        },
-      };
-
-      await updateUserProfile(payload);
+      try {
+        await updateUserProfile({ onboarding_step: steps });
+      } catch {
+        setToast({
+          type: TOAST_TYPE.ERROR,
+          title: "Failed",
+          message: "Failed to save your progress, Please try again later.",
+        });
+      }
     },
-    [user, userProfile, updateUserProfile]
+    [user, updateUserProfile]
   );
 
   const handleStepChange = useCallback(
```

`web/apps/web/core/lib/wrappers/store-wrapper.tsx`（对 `f27434c` 的差异）：

```diff
--- a/web/apps/web/core/lib/wrappers/store-wrapper.tsx
+++ b/web/apps/web/core/lib/wrappers/store-wrapper.tsx
@@ -12,7 +12,7 @@
 // hooks
 import { useAppTheme } from "@/hooks/store/use-app-theme";
 import { useRouterParams } from "@/hooks/store/use-router-params";
-import { useUserProfile } from "@/hooks/store/user";
+import { useUser, useUserProfile } from "@/hooks/store/user";
 
 type TStoreWrapper = {
   children: ReactNode;
@@ -27,6 +27,7 @@
   // store hooks
   const { setQuery } = useRouterParams();
   const { sidebarCollapsed, toggleSidebar } = useAppTheme();
+  const { data: currentUser } = useUser();
   const { data: userProfile } = useUserProfile();
   // Track if we've initialized theme from server (one-time only)
   const hasInitializedThemeRef = useRef(false);
@@ -50,7 +51,7 @@
    * This prevents a feedback loop where server updates trigger UI updates in a cycle.
    */
   useEffect(() => {
-    const userId = userProfile?.id;
+    const userId = currentUser?.id;
 
     // Reset initialization flag when user changes (logout/login)
     // This handles both logout (userId becomes undefined) and login (userId changes)
@@ -60,16 +61,16 @@
     }
 
     // Only initialize theme from server on FIRST load for this user
-    if (!userProfile?.theme?.theme || hasInitializedThemeRef.current) {
+    if (!userProfile?.theme || hasInitializedThemeRef.current) {
       return; // Skip if already initialized or no profile data
     }
 
     // Apply theme from server profile (one-time only)
-    setTheme(userProfile?.theme?.theme || "system");
+    setTheme(userProfile.theme);
 
     // Mark as initialized - prevents future syncs from server
     hasInitializedThemeRef.current = true;
-  }, [userProfile?.theme?.theme, setTheme]);
+  }, [currentUser?.id, userProfile?.theme, setTheme]);
 
   useEffect(() => {
     if (!params) return;
```

`web/apps/web/core/lib/wrappers/authentication-wrapper.tsx`（对 `f27434c` 的差异）：

```diff
--- a/web/apps/web/core/lib/wrappers/authentication-wrapper.tsx
+++ b/web/apps/web/core/lib/wrappers/authentication-wrapper.tsx
@@ -85,7 +85,7 @@
   if (pageType === EPageTypes.NON_AUTHENTICATED) {
     if (!currentUser?.id) return <>{children}</>;
     else {
-      if (currentUserProfile?.id && isUserOnboard) {
+      if (currentUserProfile && isUserOnboard) {
         const currentRedirectRoute = getWorkspaceRedirectionUrl();
         return <Navigate to={currentRedirectRoute} replace />;
       } else {
@@ -98,7 +98,7 @@
     if (!currentUser?.id) {
       return <Navigate to={`/?next_path=${pathname}`} replace />;
     } else {
-      if (currentUser && currentUserProfile?.id && isUserOnboard) {
+      if (currentUser && currentUserProfile && isUserOnboard) {
         const currentRedirectRoute = getWorkspaceRedirectionUrl();
         return <Navigate to={currentRedirectRoute} replace />;
       } else return <>{children}</>;
@@ -107,7 +107,7 @@
 
   if (pageType === EPageTypes.AUTHENTICATED) {
     if (currentUser?.id) {
-      if (currentUserProfile && currentUserProfile?.id && isUserOnboard) return <>{children}</>;
+      if (currentUserProfile && isUserOnboard) return <>{children}</>;
       else {
         return <Navigate to="/onboarding" replace />;
       }
```

- [ ] **Step 4: 关键词规则；上限**

`tools/keywords.json`（对 Task 6 版本的差异）：

```diff
--- a/tools/keywords.json
+++ b/tools/keywords.json
@@ -1769,6 +1769,37 @@
           "miss": ["docs/v0/M2-auth/M2-design.md", "api/dist/openapi.yaml"]
         }
       }
+    },
+    {
+      "id": "plane-user-urls",
+      "phase": "M2/P4",
+      "why": "Plane 的资料、新手引导、导览和实例地址（M2 设计 7.9）；前端改调 /api/v0 下的 /me/profile 和 /instance。不禁止整个 /api/users/me/：M3 的 joinProject 等旧调用还在用它",
+      "files": {
+        "source": "^web/.*\\.(?:[cm]?[jt]sx?|json)$",
+        "flags": ""
+      },
+      "content": {
+        "source": "/api/users/me/(?:profile|onboard|tour-completed)|/api/instances/",
+        "flags": ""
+      },
+      "samples": {
+        "hit": [
+          "    return this.get(\"/api/users/me/profile/\")",
+          "    return this.patch(\"/api/users/me/onboard/\", {",
+          "    return this.patch(\"/api/users/me/tour-completed/\", {",
+          "    return this.get(\"/api/instances/\")"
+        ],
+        "miss": [
+          "    return unwrap(await api.GET(\"/api/v0/me/profile\"));",
+          "    return unwrap(await publicClient.GET(\"/api/v0/instance\"));",
+          "    return this.post(`/api/users/me/workspaces/${workspaceSlug}/projects/invitations/`, { project_ids })",
+          "    const url = bustCache ? `/api/users/me/settings/?t=${Date.now()}` : \"/api/users/me/settings/\";"
+        ],
+        "files": {
+          "hit": ["web/apps/web/core/services/user.service.ts", "web/apps/web/core/services/instance.service.ts"],
+          "miss": ["e2e/stories/smoke/s2-web-app.spec.ts", "docs/v0/M2-auth/M2-design.md"]
+        }
+      }
     }
   ],
   "exceptions": [
```

`web/apps/web/package.json`（对 Task 7 版本的差异）：

```diff
--- a/web/apps/web/package.json
+++ b/web/apps/web/package.json
@@ -7,7 +7,7 @@
   "scripts": {
     "dev": "react-router dev --port 3000",
     "build": "react-router build",
-    "check:lint": "node ../../../tools/lint-cap.mjs 555",
+    "check:lint": "node ../../../tools/lint-cap.mjs 554",
     "check:types": "react-router typegen && tsc --noEmit",
     "check:format": "oxfmt --check .",
     "fix:format": "oxfmt .",
```

Run: `grep -rnw -e TUserProfile -e IUserTheme -e TOnboardingSteps web/apps/web/app web/apps/web/core web/packages --exclude-dir=node_modules --exclude-dir=dist`
Expected: 没有输出。

- [ ] **Step 5: 检查**

Run: `make lint-web`、`make knip`、`make test-web`
Expected: 都通过；web 的 oxlint 警告正好 554 个；关键词守卫 53 条规则，没有命中。

- [ ] **Step 6: 提交**

```bash
git add web/apps/web/core web/packages/types/src/users.ts web/packages/constants pnpm-lock.yaml tools/keywords.json web/apps/web/package.json
```
```bash
git commit -m "feat(M2/P4): the profile store on the generated Profile, themes on Theme

The profile store throws when nerve refuses, finishes the onboarding in
one PATCH and saves the theme and the steps through /me/profile. THEME_OPTIONS
values are the generated Theme; TUserProfile, IUserTheme and
TOnboardingSteps are gone, and the keyword guard keeps Plane's profile,
onboarding, tour and instance addresses out.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** Step 4 的 `grep` 没有输出；上限 554；检查都通过。

---

### Task 9: `AuthenticationWrapper`、"会话暂不可用"、`next_path`；axios 基类

**Files:**
- Create: `web/apps/web/core/lib/auth/use-session.ts`、`web/apps/web/core/components/account/session-unavailable.tsx`
- Modify: `web/apps/web/core/lib/wrappers/authentication-wrapper.tsx`、`web/apps/web/core/services/api.service.ts`
- Modify: `web/packages/utils/src/url.ts`、`next-path.test.ts`
- Modify（过渡）: `web/packages/i18n/src/locales/en/auth.json`、`zh-CN/auth.json`

**Interfaces:**
- Produces（spec 2.9，M2 设计 3.18、7.1、7.2、7.4；M1-P4 交接）：
  - `useSession(): SessionState`：`useSyncExternalStore(tokenManager.subscribe, () => tokenManager.state)`；
  - `AuthenticationWrapper`（唯一发出跳转的地方，只读会话、账户、资料和 `next_path`）：
    - `starting` → 加载图标；`unavailable` → `SessionUnavailable`，按钮调 `tokenManager.retry()`；
    - `signed-out`：公开页和登录、注册页照常显示，其他页面 `<Navigate to={signInPath(pathname + search + hash)} replace />`；
    - `signed-in`：账户按 `["CURRENT_USER", loginId]` 用 SWR 取（换会话就重取，不在焦点变化时重取，不自动重试）；取失败 → `SessionUnavailable`，按钮重取；账户或资料还没到 → 加载图标；登录、注册页 → 合格的 `next_path`，没有时按是否完成新手引导去 `/create-workspace` 或 `/onboarding`；新手引导页 → 完成了就去合格的 `next_path` 或 `/create-workspace`；其他页面 → 没完成新手引导就去 `/onboarding`。"完成"是 `is_onboarded`，或四个步骤都为真；
  - `SessionUnavailable({ onRetry })`：`role="alert"`，标题、说明、"重试"，文案键 `auth.session_unavailable.{title,description,retry}`（"Cannot reach the server for now" / "暂时无法连接服务器"）；不跳转，地址和记录都不变；
  - `@nerve/utils`：`isValidNextPath` 另外拒绝任何位置的控制字符（`\u0000`–`\u001f`、`\u007f`：浏览器会从地址里丢掉制表符和换行，`/\t/evil.example` 会变成 `//evil.example`）；`signInPath(path) = "/?next_path=" + encodeURIComponent(path)`；
  - web 的 axios 基类：删掉 `withCredentials: true` 和 401 拦截（恒为真的 `currentPath` 判断不照搬），此后只给还没对接的 M3–M8 领域用，不带令牌、不跳转。
- 使用者：Task 10 的登录页、注册页；Task 12、13 的故事；浏览器核对 C2、C6。

**Tests:**（`next-path.test.ts`，11 → 20 个）合格：站内路径、多段路径、带查询和片段、首尾有空格；不合格：制表符、换行在前，斜杠之间的制表符，NUL、U+001F、DEL；控制字符两边的字符（空格、`~`、U+0080、`é`）合格；原有的协议、`//`、`\`、`/\`、空串、不以 `/` 开头。`signInPath`：路径、查询、片段整体编码；从地址里取回 `next_path` 与原值相同，且合格。

- [ ] **Step 1: `next_path`**

`web/packages/utils/src/url.ts`（对 `f27434c` 的差异）：

```diff
--- a/web/packages/utils/src/url.ts
+++ b/web/packages/utils/src/url.ts
@@ -21,10 +21,15 @@
  * isValidNextPath("dashboard") // false (must start with /)
  * isValidNextPath("\\malicious") // false (backslash)
  * isValidNextPath("  /dashboard  ") // true (trimmed)
+ * isValidNextPath("\t/dashboard") // false (control character)
  */
 export function isValidNextPath(url: string): boolean {
   if (!url || typeof url !== "string") return false;
 
+  // No control characters (U+0000–U+001F, U+007F) anywhere: browsers drop some of them from an address,
+  // which can turn what was checked into something else (M2 design 3.18).
+  if ([...url].some((c) => c.charCodeAt(0) <= 0x1f || c.charCodeAt(0) === 0x7f)) return false;
+
   // Trim leading/trailing whitespace
   const trimmedUrl = url.trim();
 
@@ -66,3 +71,14 @@
     return false;
   }
 }
+
+/**
+ * The address of the sign-in page that comes back to `path` after signing in: `path` is the page's
+ * pathname, search and hash together, encoded as one value (M2 design 3.18).
+ *
+ * @example
+ * signInPath("/settings/profile/general?tab=x#y") // "/?next_path=%2Fsettings%2Fprofile%2Fgeneral%3Ftab%3Dx%23y"
+ */
+export function signInPath(path: string): string {
+  return `/?next_path=${encodeURIComponent(path)}`;
+}
```

`web/packages/utils/src/next-path.test.ts`（完整内容）：

```ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import { isValidNextPath, signInPath } from "./url";

// After signing in, the app goes to the `next_path` query parameter. Anyone can put a link with any
// `next_path` in front of a user, so only a path on this site may pass; everything else would be an
// open redirect. These are the examples from the function's own documentation.

describe("isValidNextPath", () => {
  it("accepts a path on this site", () => {
    expect(isValidNextPath("/dashboard")).toBe(true);
  });

  it("accepts a path with several segments", () => {
    expect(isValidNextPath("/workspace/123")).toBe(true);
  });

  it("accepts a path with a query and a fragment", () => {
    expect(isValidNextPath("/settings/profile/general?tab=x#y")).toBe(true);
  });

  it("accepts a path surrounded by spaces", () => {
    expect(isValidNextPath("  /dashboard  ")).toBe(true);
  });

  // `%09` and `%0A` in the address arrive as a tab and a newline. Control characters are refused wherever
  // they are (M2 design 3.18): the browser drops tabs and newlines from an address, so "/\t/evil.example"
  // would be the protocol-relative "//evil.example".
  it.each([
    ["a tab before the path", "\t/dashboard"],
    ["a newline before the path", "\n/dashboard"],
    ["a tab between the slashes", "/\t/evil.example"],
    ["a NUL", "/dash\u0000board"],
    ["a U+001F", "/dashboard\u001f"],
    ["a DEL", "/dash\u007fboard"],
  ])("rejects a path with %s", (_, path) => {
    expect(isValidNextPath(path)).toBe(false);
  });

  it("accepts the characters around the control ranges", () => {
    expect(isValidNextPath("/a b~")).toBe(true);
    expect(isValidNextPath("/\u0080é")).toBe(true);
  });

  it("rejects an absolute address", () => {
    expect(isValidNextPath("https://malicious.com")).toBe(false);
  });

  it("rejects a protocol-relative address", () => {
    expect(isValidNextPath("//malicious.com")).toBe(false);
  });

  it("rejects a javascript: address", () => {
    expect(isValidNextPath("javascript:alert(1)")).toBe(false);
  });

  it("rejects an empty string", () => {
    expect(isValidNextPath("")).toBe(false);
  });

  it("rejects a path that does not start with a slash", () => {
    expect(isValidNextPath("dashboard")).toBe(false);
  });

  it("rejects a path that starts with a backslash", () => {
    expect(isValidNextPath("\\malicious")).toBe(false);
  });

  it("rejects a slash and a backslash, which browsers read as a protocol-relative address", () => {
    expect(isValidNextPath("/\\evil.example")).toBe(false);
  });
});

describe("signInPath", () => {
  it("encodes the path, its query and its fragment as one value", () => {
    expect(signInPath("/settings/profile/general?tab=x#y")).toBe(
      "/?next_path=%2Fsettings%2Fprofile%2Fgeneral%3Ftab%3Dx%23y"
    );
  });

  it("gives back the path, query and fragment unchanged as next_path", () => {
    const path = "/settings/profile/general?tab=a&b=c d#y%20z";
    const next = new URL(signInPath(path), "http://nerve.test").searchParams.get("next_path");
    expect(next).toBe(path);
    expect(isValidNextPath(next ?? "")).toBe(true);
  });
});
```

- [ ] **Step 2: 会话状态和"会话暂不可用"**

`web/apps/web/core/lib/auth/use-session.ts`（新文件）：

```ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useSyncExternalStore } from "react";
import { tokenManager } from "./api-client";
import type { SessionState } from "./token-manager";

const snapshot = () => tokenManager.state;

/** The session as the token manager has it; the component renders again whenever it changes. */
export function useSession(): SessionState {
  return useSyncExternalStore(tokenManager.subscribe, snapshot);
}
```

`web/packages/i18n/src/locales/en/auth.json`（对 `f27434c` 的差异）：

```diff
--- a/web/packages/i18n/src/locales/en/auth.json
+++ b/web/packages/i18n/src/locales/en/auth.json
@@ -62,6 +62,11 @@
           "message": "Failed to sign out. Please try again."
         }
       }
+    },
+    "session_unavailable": {
+      "title": "Cannot reach the server for now",
+      "description": "You are still signed in. The page tries again by itself, or you can try now.",
+      "retry": "Try again"
     }
   }
 }
```

`web/packages/i18n/src/locales/zh-CN/auth.json`（对 `f27434c` 的差异）：

```diff
--- a/web/packages/i18n/src/locales/zh-CN/auth.json
+++ b/web/packages/i18n/src/locales/zh-CN/auth.json
@@ -62,6 +62,11 @@
           "message": "登出失败。请重试。"
         }
       }
+    },
+    "session_unavailable": {
+      "title": "暂时连不上服务器",
+      "description": "你仍然处于登录状态。页面会自动重试，也可以现在重试。",
+      "retry": "重试"
     }
   }
 }
```

`web/apps/web/core/components/account/session-unavailable.tsx`（新文件）：

```text
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
// layouts
import DefaultLayout from "@/layouts/default-layout";

type Props = {
  /** Tries again at once; the page also tries again by itself. */
  onRetry: () => void;
};

/**
 * The page while the session is kept but nerve cannot be reached (M2 design 7.1, 7.4): a refresh failed
 * with 429, 5xx or no network. It stays on the page the user asked for, never on the sign-in page.
 */
export function SessionUnavailable({ onRetry }: Props) {
  const { t } = useTranslation();
  return (
    <DefaultLayout>
      <div
        role="alert"
        className="relative container mx-auto flex h-full w-full max-w-xl flex-col items-center justify-center gap-4 text-center"
      >
        <h1 className="text-h4-semibold text-primary">{t("auth.session_unavailable.title")}</h1>
        <p className="text-body-sm-regular text-secondary">{t("auth.session_unavailable.description")}</p>
        <Button variant="primary" onClick={onRetry}>
          {t("auth.session_unavailable.retry")}
        </Button>
      </div>
    </DefaultLayout>
  );
}
```

- [ ] **Step 3: 认证包装**

`web/apps/web/core/lib/wrappers/authentication-wrapper.tsx`（完整内容）：

```text
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
import { observer } from "mobx-react";
import { Navigate, useLocation, useSearchParams } from "react-router";
import useSWR from "swr";
// nerve imports
import type { Profile } from "@nerve/api-client";
import { isValidNextPath, signInPath } from "@nerve/utils";
// components
import { SessionUnavailable } from "@/components/account/session-unavailable";
import { LogoSpinner } from "@/components/common/logo-spinner";
// helpers
import { EPageTypes } from "@/helpers/authentication.helper";
// hooks
import { useUser, useUserProfile } from "@/hooks/store/user";
// lib
import { tokenManager } from "@/lib/auth/api-client";
import { useSession } from "@/lib/auth/use-session";

type TAuthenticationWrapper = {
  children: ReactNode;
  pageType?: EPageTypes;
};

const isOnboarded = (profile: Profile) =>
  profile.is_onboarded ||
  (profile.onboarding_step.profile_complete &&
    profile.onboarding_step.workspace_create &&
    profile.onboarding_step.workspace_invite &&
    profile.onboarding_step.workspace_join);

function Loading() {
  return (
    <div className="relative flex h-screen w-full items-center justify-center">
      <LogoSpinner />
    </div>
  );
}

/**
 * Decides who may see a page (M2 design 7.4), from the session, the account, its profile and next_path
 * only, and is the one place that sends the user elsewhere (7.1): signed out on a page that needs an
 * account, to the sign-in page with the page as next_path (3.18); signed in on the sign-in or sign-up page,
 * to a valid next_path, else to /onboarding until onboarded, else to /create-workspace. While nerve cannot
 * be reached the session is kept, and the page says so instead of moving.
 */
export const AuthenticationWrapper = observer(function AuthenticationWrapper(props: TAuthenticationWrapper) {
  const { children, pageType = EPageTypes.AUTHENTICATED } = props;
  const { pathname, search, hash } = useLocation();
  const [searchParams] = useSearchParams();
  const nextPath = searchParams.get("next_path")?.trim();
  // store hooks
  const session = useSession();
  const { data: currentUser, fetchCurrentUser } = useUser();
  const { data: currentProfile } = useUserProfile();

  // The account is fetched for each session: a sign-in here or in another tab, or another account (7.1).
  const { error, mutate } = useSWR(
    session.status === "signed-in" ? ["CURRENT_USER", session.loginId] : null,
    () => fetchCurrentUser(),
    { revalidateOnFocus: false, shouldRetryOnError: false }
  );

  if (session.status === "starting") return <Loading />;
  if (session.status === "unavailable") return <SessionUnavailable onRetry={() => void tokenManager.retry()} />;

  if (session.status === "signed-out") {
    if (pageType === EPageTypes.PUBLIC || pageType === EPageTypes.NON_AUTHENTICATED) return <>{children}</>;
    return <Navigate to={signInPath(pathname + search + hash)} replace />;
  }

  if (error) return <SessionUnavailable onRetry={() => void mutate()} />;
  if (!currentUser || !currentProfile) return <Loading />;
  if (pageType === EPageTypes.PUBLIC) return <>{children}</>;

  const validNextPath = nextPath && isValidNextPath(nextPath) ? nextPath : undefined;
  const onboarded = isOnboarded(currentProfile);
  switch (pageType) {
    case EPageTypes.NON_AUTHENTICATED:
      return <Navigate to={validNextPath ?? (onboarded ? "/create-workspace" : "/onboarding")} replace />;
    case EPageTypes.ONBOARDING:
      return onboarded ? <Navigate to={validNextPath ?? "/create-workspace"} replace /> : <>{children}</>;
    default:
      return onboarded ? <>{children}</> : <Navigate to="/onboarding" replace />;
  }
});
```

- [ ] **Step 4: axios 基类**

`web/apps/web/core/services/api.service.ts`（对 `f27434c` 的差异）：

```diff
--- a/web/apps/web/core/services/api.service.ts
+++ b/web/apps/web/core/services/api.service.ts
@@ -9,13 +9,14 @@
 import { create } from "axios";
 import { normalizeAPIRequestURL } from "@nerve/services";
 
+// The domains that do not use nerve's API yet (M3–M8) still call their old addresses through this class, and
+// nerve answers them 404 problem; each domain moves to the generated client when it gets its API (M2 design
+// 7.2). It sends no token, and never navigates: sending the user to sign in is AuthenticationWrapper's.
 export abstract class APIService {
   private axiosInstance: AxiosInstance;
 
   constructor() {
-    this.axiosInstance = create({
-      withCredentials: true,
-    });
+    this.axiosInstance = create();
 
     this.setupInterceptors();
   }
@@ -33,17 +34,6 @@
       }
       return config;
     });
-
-    this.axiosInstance.interceptors.response.use(
-      (response) => response,
-      (error) => {
-        if (error.response && error.response.status === 401) {
-          const currentPath = window.location.pathname;
-          window.location.replace(`/${currentPath ? `?next_path=${currentPath}` : ``}`);
-        }
-        return Promise.reject(error);
-      }
-    );
   }
 
   get(url: string, params = {}, config: AxiosRequestConfig = {}) {
```

Run: `grep -rn -e withCredentials -e "window.location.replace" web/apps/web/core/services web/apps/web/core/lib`
Expected: 只有 `web/apps/web/core/services/file-upload.service.ts` 的 `withCredentials: false`（上传到对象存储，M5）。

- [ ] **Step 5: 检查**

Run: `pnpm -C web/packages/utils exec vitest run src/next-path.test.ts`
Expected: 20 个测试通过。

Run: `make lint-web`、`make knip`、`make test-web`
Expected: 都通过。

- [ ] **Step 6: 提交**

```bash
git add web/apps/web/core web/packages/utils/src web/packages/i18n/src/locales
```
```bash
git commit -m "feat(M2/P4): AuthenticationWrapper on the token manager; session unavailable; next_path with query and hash

The wrapper reads only the session, the account, its profile and
next_path, and is the one place that navigates. While a refresh fails for
a passing reason it shows 'cannot reach the server' with a retry, and
keeps the page and the session. next_path carries pathname, search and
hash, encoded as one value, and refuses control characters. The axios
base class sends no cookie and no longer redirects on 401.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** 20 个测试通过；Step 4 的 `grep` 只有那一处。

---

### Task 10: 登录页、注册页和错误文案表；修改密码的错误；密码 8–128 个字符

**Files:**
- Modify: `web/apps/web/core/components/account/auth-forms/auth-root.tsx`、`auth-header.tsx`、`password.tsx`
- Delete: `web/apps/web/helpers/authentication.helper.tsx`；Create: `web/apps/web/helpers/authentication.helper.ts`、`authentication.helper.test.ts`
- Modify: `web/apps/web/vitest.config.ts`、`web/apps/web/core/store/user/index.ts`、`web/apps/web/core/services/auth.service.ts`、`web/apps/web/core/components/settings/profile/content/pages/security.tsx`
- Delete: `web/packages/types/src/auth.ts`；Modify: `web/packages/types/src/index.ts`
- Modify: `web/packages/utils/src/auth.ts`；Create: `web/packages/utils/src/auth.test.ts`；Modify: `web/packages/ui/src/form-fields/password/helper.tsx`
- Modify: `web/packages/i18n/src/locales/en/auth.json`、`zh-CN/auth.json`、`tools/keywords.json`、`web/apps/web/package.json`

**Interfaces:**
- Produces（spec 2.10，M2 设计 3.8、3.11、7.2、7.3；M1-P3、M1-P4 交接）：
  - `AuthService.register(RegisterRequest)`、`login(LoginRequest)` 经 `publicClient` 和 `unwrap`；`UserStore.signIn`、`signUp` 把得到的令牌交给 `tokenManager.signIn`，失败时抛出 `ApiError`，什么都不留；
  - `AuthRoot` 只读地址里的 `invitation_id`、`slug`（M3 的邀请）；`AuthPasswordForm({ mode })` 是受控表单（`noValidate`，`autoComplete` 为 `email`、`current-password`、`new-password`），提交时调 `signIn` 或 `signUp`；nerve 拒绝时：字段错误显示在字段下方（`fieldErrorKeys`），没有字段错误时在表单上方显示一句（`errorMessageKey`，`role="alert"`）；不跳转，输入的内容都留着；再次编辑时清掉上一次的提示（包括"密码强度不够"的提示）；成功之后的跳转交给 `AuthenticationWrapper`；
  - `helpers/authentication.helper.ts`：`EPageTypes`、`EAuthModes`；`PROBLEM_MESSAGES`（`api/dist/openapi.yaml` 中全部 13 个问题码 → 文案键）、`FIELD_ERROR_MESSAGES`（`FieldError.code` 的全部 10 个取值 → 文案键）；`errorMessageKey(error)`（`ApiError` 按码，不认识的码和没有 problem 的回答是 `auth.errors.unknown`；`SessionUnavailableError` 是 `auth.errors.unreachable`）；`fieldErrorKeys(error)`（字段 → 文案键）。Plane 的 `EAuthenticationErrorCodes`、`authErrorHandler` 等随 `.tsx` 删除；
  - 安全页修改密码：`identity.current_password_incorrect` 显示在当前密码下方；`new_password` 的字段错误显示在新密码下方；其他错误用提示框，内容按错误码取文案（`change_password.error.message` 这个笼统的键删除）；Plane 的 `Error & { error_code?: string }` 断言和 `toString()` 删除；
  - `@nerve/types` 的 `auth.ts`（`ICsrfTokenData` 等）删除；
  - 密码规则：`MIN_PASSWORD_LENGTH = 8`、`MAX_PASSWORD_LENGTH = 128`，按 UTF-16 码元计（与服务端的 `domain.MaxPasswordLength` 相同），长度规则的标签是"8–128 characters"；
  - web 的 vitest 配置加 `resolve.tsconfigPaths`，测试能解析 `@/`；
  - 关键词规则 `csrf`（不区分大小写）、`plane-auth-urls`（`(?<![\w.@-])/auth/`）、`auth-error-code`（`error_code|EAuthenticationErrorCodes`）；
  - web 的 oxlint 上限 554 → 551。
- 使用者：Task 12 的 A1–A3、A15 的页面版本。

**Tests:**
- `authentication.helper.test.ts`（9 个）：文案表读到了契约（两张表在 `openapi.yaml` 中都找得到）；有契约中每个问题码的文案、没有别的码；有 `FieldError.code` 每个取值的文案、没有别的；指向的文案在英文文件中都存在（`sync-check` 保证中文相同）；`errorMessageKey` 按码给文案，不认识的码或没有 problem 时给通用文案，会话暂不可用时给"无法连接"；`fieldErrorKeys` 按字段给文案，没有字段的 problem 和别的错误给空。
- `auth.test.ts`（`@nerve/utils`，5 个）：8、128 个字符合格；7、129 个不合格；64 个 emoji 是 128 个 UTF-16 码元，再多一个字符就不合格；缺一类字符不合格、空串；长度规则在 129 个字符时显示未满足。

- [ ] **Step 1: 错误文案表**

```bash
git rm web/apps/web/helpers/authentication.helper.tsx
```

`web/apps/web/helpers/authentication.helper.ts`（新文件）：

```ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { FieldError } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { SessionUnavailableError } from "@/lib/auth/token-manager";

export enum EPageTypes {
  PUBLIC = "PUBLIC",
  NON_AUTHENTICATED = "NON_AUTHENTICATED",
  ONBOARDING = "ONBOARDING",
  AUTHENTICATED = "AUTHENTICATED",
}

export enum EAuthModes {
  SIGN_IN = "SIGN_IN",
  SIGN_UP = "SIGN_UP",
}

/**
 * The message of every problem code nerve's API answers: the codes of every x-problem-codes in
 * api/dist/openapi.yaml, which a test keeps equal to the keys (M2 design 3.11, 7.3).
 */
export const PROBLEM_MESSAGES: Readonly<Record<string, string>> = {
  bad_request: "auth.errors.bad_request",
  payload_too_large: "auth.errors.payload_too_large",
  rate_limited: "auth.errors.rate_limited",
  internal_error: "auth.errors.internal_error",
  validation_failed: "auth.errors.validation_failed",
  server_busy: "auth.errors.server_busy",
  "identity.signup_disabled": "auth.errors.signup_disabled",
  "identity.email_taken": "auth.errors.email_taken",
  "identity.invalid_credentials": "auth.errors.invalid_credentials",
  "identity.account_deactivated": "auth.errors.account_deactivated",
  "identity.refresh_token_invalid": "auth.errors.refresh_token_invalid",
  "identity.current_password_incorrect": "auth.errors.current_password_incorrect",
  "identity.api_token_not_found": "auth.errors.api_token_not_found",
};

/** The message of every FieldError.code, shown under the field it names. */
export const FIELD_ERROR_MESSAGES: Readonly<Record<FieldError["code"], string>> = {
  required: "auth.errors.field.required",
  invalid_format: "auth.errors.field.invalid_format",
  too_short: "auth.errors.field.too_short",
  too_long: "auth.errors.field.too_long",
  out_of_range: "auth.errors.field.out_of_range",
  not_allowed: "auth.errors.field.not_allowed",
  weak_password: "auth.errors.field.weak_password",
  common_password: "auth.errors.field.common_password",
  must_be_future: "auth.errors.field.must_be_future",
  contains_url: "auth.errors.field.contains_url",
};

/** The i18n key of the message for an error of a call to nerve. */
export function errorMessageKey(error: unknown): string {
  if (error instanceof ApiError) return PROBLEM_MESSAGES[error.problem?.code ?? ""] ?? "auth.errors.unknown";
  if (error instanceof SessionUnavailableError) return "auth.errors.unreachable";
  return "auth.errors.unknown";
}

/** The i18n keys of the messages for the fields a problem names, by field. */
export function fieldErrorKeys(error: unknown): Partial<Record<string, string>> {
  if (!(error instanceof ApiError)) return {};
  return Object.fromEntries((error.problem?.errors ?? []).map((e) => [e.field, FIELD_ERROR_MESSAGES[e.code]]));
}
```

`web/apps/web/helpers/authentication.helper.test.ts`（新文件）：

```ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";
import { ApiError } from "@/lib/api-error";
import { SessionChangedError, SessionUnavailableError } from "@/lib/auth/token-manager";
import { FIELD_ERROR_MESSAGES, PROBLEM_MESSAGES, errorMessageKey, fieldErrorKeys } from "./authentication.helper";

// The message tables against the API's contract (M2 design 3.11, 7.3, 9.4): api/dist/openapi.yaml is the
// bundled description the server is checked against, so a code added there fails here until it has a message.

const spec = readFileSync(new URL("../../../../api/dist/openapi.yaml", import.meta.url), "utf8").split("\n");
const en = JSON.parse(
  readFileSync(new URL("../../../packages/i18n/src/locales/en/auth.json", import.meta.url), "utf8")
) as Record<string, unknown>;

/** The items of every list under a line that matches `key` (block lists: "- item", deeper than the key). */
function listsUnder(key: RegExp): string[] {
  const items: string[] = [];
  spec.forEach((line, i) => {
    const head = key.exec(line);
    if (!head) return;
    const indent = line.length - line.trimStart().length;
    for (const next of spec.slice(i + 1)) {
      const item = /^(\s*)- (\S+)$/.exec(next);
      if (!item || item[1].length < indent) break;
      items.push(item[2]);
    }
  });
  return items;
}

const problemCodes = new Set(listsUnder(/^\s*x-problem-codes:\s*$/));
// FieldError.code's enum: the only "enum:" list under the FieldError schema.
const fieldErrorStart = spec.findIndex((line) => /^ {4}FieldError:$/.test(line));
const fieldCodes = (() => {
  const enumLine = spec.findIndex((line, i) => i > fieldErrorStart && /^\s+enum:$/.test(line));
  const codes: string[] = [];
  for (const line of spec.slice(enumLine + 1)) {
    const item = /^\s+- (\S+)$/.exec(line);
    if (!item) break;
    codes.push(item[1]);
  }
  return new Set(codes);
})();

/** The value at a dotted i18n key of en's auth namespace, or undefined. */
function english(key: string): unknown {
  return key.split(".").reduce<unknown>((node, part) => (node as Record<string, unknown> | undefined)?.[part], en);
}

describe("the message tables", () => {
  it("read the contract: both lists are found in openapi.yaml", () => {
    expect(problemCodes).toContain("identity.email_taken");
    expect(problemCodes).toContain("bad_request");
    expect(fieldCodes).toContain("common_password");
  });

  it("have a message for every problem code of the contract, and no other", () => {
    expect(new Set(Object.keys(PROBLEM_MESSAGES))).toEqual(problemCodes);
  });

  it("have a message for every FieldError.code of the contract, and no other", () => {
    expect(new Set(Object.keys(FIELD_ERROR_MESSAGES))).toEqual(fieldCodes);
  });

  it("point at messages that exist in English (sync-check keeps zh-CN the same)", () => {
    const keys = [...Object.values(PROBLEM_MESSAGES), ...Object.values(FIELD_ERROR_MESSAGES)];
    for (const key of [...keys, "auth.errors.unknown", "auth.errors.unreachable"]) {
      expect(typeof english(key), key).toBe("string");
    }
  });
});

const problem = (code: string) => ({ status: 400, code, title: "" });

describe("errorMessageKey", () => {
  it("gives the message of the problem's code", () => {
    expect(errorMessageKey(new ApiError(409, problem("identity.email_taken")))).toBe("auth.errors.email_taken");
    expect(errorMessageKey(new ApiError(429, problem("rate_limited")))).toBe("auth.errors.rate_limited");
  });

  it("gives the general message for a code it does not know, or an answer without a problem", () => {
    expect(errorMessageKey(new ApiError(418, problem("teapot")))).toBe("auth.errors.unknown");
    expect(errorMessageKey(new ApiError(502, undefined))).toBe("auth.errors.unknown");
    expect(errorMessageKey(new TypeError("Failed to fetch"))).toBe("auth.errors.unknown");
    expect(errorMessageKey(new SessionChangedError())).toBe("auth.errors.unknown");
  });

  it("says the server cannot be reached when the session is unavailable", () => {
    expect(errorMessageKey(new SessionUnavailableError(0))).toBe("auth.errors.unreachable");
  });
});

describe("fieldErrorKeys", () => {
  it("gives each named field the message of its code", () => {
    const error = new ApiError(422, {
      status: 422,
      code: "validation_failed",
      title: "",
      errors: [
        { field: "password", code: "common_password", message: "is too common" },
        { field: "email", code: "required", message: "is required" },
      ],
    });
    expect(fieldErrorKeys(error)).toEqual({
      password: "auth.errors.field.common_password",
      email: "auth.errors.field.required",
    });
  });

  it("gives nothing for a problem without fields, or for another error", () => {
    expect(fieldErrorKeys(new ApiError(409, { status: 409, code: "identity.email_taken", title: "" }))).toEqual({});
    expect(fieldErrorKeys(new TypeError("Failed to fetch"))).toEqual({});
  });
});
```

`web/apps/web/vitest.config.ts`（完整内容）：

```ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { defineConfig } from "vitest/config";

// The tests read the route table and the source, in Node. Without this file vitest would load vite.config.ts,
// the app's build configuration, and run its React Router plugin, which the tests do not use. Imports that
// start with "@/" resolve through tsconfig.json's paths, as in the build.
export default defineConfig({ resolve: { tsconfigPaths: true }, test: { environment: "node" } });
```

`web/packages/i18n/src/locales/en/auth.json`（对 Task 9 版本的差异）：

```diff
--- a/web/packages/i18n/src/locales/en/auth.json
+++ b/web/packages/i18n/src/locales/en/auth.json
@@ -39,8 +39,7 @@
               "message": "Password changed successfully."
             },
             "error": {
-              "title": "Error!",
-              "message": "Something went wrong. Please try again."
+              "title": "Error!"
             }
           }
         }
@@ -67,6 +66,35 @@
       "title": "Cannot reach the server for now",
       "description": "You are still signed in. The page tries again by itself, or you can try now.",
       "retry": "Try again"
+    },
+    "errors": {
+      "bad_request": "The request was not understood. Check the fields and try again.",
+      "payload_too_large": "The request is too large.",
+      "rate_limited": "Too many attempts. Please try again later.",
+      "internal_error": "Something went wrong on the server. Please try again.",
+      "validation_failed": "Some fields are not valid.",
+      "server_busy": "The server is busy. Please try again later.",
+      "signup_disabled": "Sign-up is closed.",
+      "email_taken": "An account with this email already exists.",
+      "invalid_credentials": "The email or the password is wrong.",
+      "account_deactivated": "This account is deactivated.",
+      "refresh_token_invalid": "You are signed out. Please sign in again.",
+      "current_password_incorrect": "The current password is wrong.",
+      "api_token_not_found": "The token does not exist or has been revoked.",
+      "unreachable": "Cannot reach the server for now. Please try again.",
+      "unknown": "Something went wrong. Please try again.",
+      "field": {
+        "required": "Required",
+        "invalid_format": "The format is not valid",
+        "too_short": "Too short",
+        "too_long": "Too long",
+        "out_of_range": "Out of range",
+        "not_allowed": "Not allowed",
+        "weak_password": "Use 8–128 characters with an upper-case letter, a lower-case letter, a digit and a special character",
+        "common_password": "This password is too common",
+        "must_be_future": "Must be in the future",
+        "contains_url": "Must not contain a web address"
+      }
     }
   }
 }
```

`web/packages/i18n/src/locales/zh-CN/auth.json`（对 Task 9 版本的差异）：

```diff
--- a/web/packages/i18n/src/locales/zh-CN/auth.json
+++ b/web/packages/i18n/src/locales/zh-CN/auth.json
@@ -39,8 +39,7 @@
               "message": "密码修改成功。"
             },
             "error": {
-              "title": "错误！",
-              "message": "出现错误。请重试。"
+              "title": "错误！"
             }
           }
         }
@@ -67,6 +66,35 @@
       "title": "暂时连不上服务器",
       "description": "你仍然处于登录状态。页面会自动重试，也可以现在重试。",
       "retry": "重试"
+    },
+    "errors": {
+      "bad_request": "请求无法识别，请检查填写的内容后重试。",
+      "payload_too_large": "请求过大。",
+      "rate_limited": "尝试次数过多，请稍后再试。",
+      "internal_error": "服务器出错，请重试。",
+      "validation_failed": "部分内容不符合要求。",
+      "server_busy": "服务器繁忙，请稍后再试。",
+      "signup_disabled": "注册已关闭。",
+      "email_taken": "该邮箱已注册。",
+      "invalid_credentials": "邮箱或密码错误。",
+      "account_deactivated": "账户已停用。",
+      "refresh_token_invalid": "登录已失效，请重新登录。",
+      "current_password_incorrect": "当前密码错误。",
+      "api_token_not_found": "令牌不存在或已被撤销。",
+      "unreachable": "暂时连不上服务器，请稍后再试。",
+      "unknown": "出错了，请重试。",
+      "field": {
+        "required": "必填",
+        "invalid_format": "格式不正确",
+        "too_short": "太短",
+        "too_long": "太长",
+        "out_of_range": "超出范围",
+        "not_allowed": "不允许",
+        "weak_password": "密码需要 8–128 个字符，包含大写字母、小写字母、数字和特殊字符",
+        "common_password": "密码太常见",
+        "must_be_future": "必须是将来的时间",
+        "contains_url": "不能包含网址"
+      }
     }
   }
 }
```

- [ ] **Step 2: 登录、注册**

`web/apps/web/core/services/auth.service.ts`（完整内容）：

```ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { AuthTokens, LoginRequest, RegisterRequest } from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";
import { publicClient } from "@/lib/auth/api-client";

// Sign-in and sign-up; the token manager refreshes and signs out itself (M2 design 7.5).
export class AuthService {
  async register(body: RegisterRequest): Promise<AuthTokens> {
    return unwrap(await publicClient.POST("/api/v0/auth/register", { body }));
  }

  async login(body: LoginRequest): Promise<AuthTokens> {
    return unwrap(await publicClient.POST("/api/v0/auth/login", { body }));
  }
}
```

`web/apps/web/core/store/user/index.ts`（对 Task 7 版本的差异）：

```diff
--- a/web/apps/web/core/store/user/index.ts
+++ b/web/apps/web/core/store/user/index.ts
@@ -6,7 +6,7 @@
 
 import { action, makeObservable, observable, runInAction, computed } from "mobx";
 // nerve imports
-import type { ChangePasswordRequest, User, UserUpdate } from "@nerve/api-client";
+import type { ChangePasswordRequest, LoginRequest, RegisterRequest, User, UserUpdate } from "@nerve/api-client";
 import { EUserPermissions } from "@nerve/constants";
 import type { TUserPermissions } from "@nerve/types";
 // lib
@@ -16,6 +16,7 @@
 import type { IUserPermissionStore } from "@/store/user/permissions.store";
 import { UserPermissionStore } from "@/store/user/permissions.store";
 // services
+import { AuthService } from "@/services/auth.service";
 import { UserService } from "@/services/user.service";
 // stores
 import type { IUserProfileStore } from "@/store/user/profile.store";
@@ -37,6 +38,8 @@
   updateCurrentUser: (data: UserUpdate) => Promise<User>;
   deactivateAccount: () => Promise<void>;
   changePassword: (payload: ChangePasswordRequest) => Promise<void>;
+  signIn: (credentials: LoginRequest) => Promise<void>;
+  signUp: (credentials: RegisterRequest) => Promise<void>;
   signOut: () => Promise<void>;
   // computed
   canPerformAnyCreateAction: boolean;
@@ -53,6 +56,7 @@
   permission: IUserPermissionStore;
   // service
   userService: UserService;
+  authService: AuthService;
 
   constructor(private store: RootStore) {
     // stores
@@ -61,6 +65,7 @@
     this.permission = new UserPermissionStore(store);
     // service
     this.userService = new UserService();
+    this.authService = new AuthService();
     // observables
     makeObservable(this, {
       // observables
@@ -75,6 +80,8 @@
       updateCurrentUser: action,
       deactivateAccount: action,
       changePassword: action,
+      signIn: action,
+      signUp: action,
       signOut: action,
       // computed
       canPerformAnyCreateAction: computed,
@@ -132,6 +139,23 @@
   };
 
   /**
+   * @description signs in; the token manager keeps the session, and AuthenticationWrapper fetches the
+   * account and moves on (M2 design 7.3). Fails, with nothing kept, when nerve refuses.
+   * @returns {Promise<void>}
+   */
+  signIn = async (credentials: LoginRequest): Promise<void> => {
+    await tokenManager.signIn(await this.authService.login(credentials));
+  };
+
+  /**
+   * @description creates the account and signs it in, as signIn does
+   * @returns {Promise<void>}
+   */
+  signUp = async (credentials: RegisterRequest): Promise<void> => {
+    await tokenManager.signIn(await this.authService.register(credentials));
+  };
+
+  /**
    * @description signs out this browser's session; the stores start again when the session ends
    * @returns {Promise<void>}
    */
```

`web/apps/web/core/components/account/auth-forms/auth-root.tsx`（完整内容）：

```text
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { useSearchParams } from "react-router";
// helpers
import type { EAuthModes } from "@/helpers/authentication.helper";
// local imports
import { AuthHeader } from "./auth-header";
import { AuthPasswordForm } from "./password";

type TAuthRoot = {
  authMode: EAuthModes;
};

export const AuthRoot = observer(function AuthRoot(props: TAuthRoot) {
  const { authMode } = props;
  //router
  const [searchParams] = useSearchParams();
  // query params: a workspace invitation's, for M3's "join the workspace" title (M2 design 7.3)
  const invitation_id = searchParams.get("invitation_id");
  const workspaceSlug = searchParams.get("slug");

  return (
    <div className="mt-10 flex w-full flex-grow flex-col items-center justify-center py-6">
      <div className="relative flex w-full max-w-[22.5rem] flex-col gap-6">
        <AuthHeader
          workspaceSlug={workspaceSlug || undefined}
          invitationId={invitation_id || undefined}
          authMode={authMode}
        />
        <AuthPasswordForm mode={authMode} />
      </div>
    </div>
  );
});
```

`web/apps/web/core/components/account/auth-forms/auth-header.tsx`（对 `f27434c` 的差异）：

```diff
--- a/web/apps/web/core/components/account/auth-forms/auth-header.tsx
+++ b/web/apps/web/core/components/account/auth-forms/auth-header.tsx
@@ -19,7 +19,6 @@
 type TAuthHeader = {
   workspaceSlug: string | undefined;
   invitationId: string | undefined;
-  invitationEmail: string | undefined;
   authMode: EAuthModes;
 };
 
@@ -37,7 +36,7 @@
 const workSpaceService = new WorkspaceService();
 
 export const AuthHeader = observer(function AuthHeader(props: TAuthHeader) {
-  const { workspaceSlug, invitationId, invitationEmail, authMode } = props;
+  const { workspaceSlug, invitationId, authMode } = props;
   // nerve imports
   const { t } = useTranslation();
 
@@ -50,13 +49,9 @@
     }
   );
 
-  const getHeaderSubHeader = (
-    mode: EAuthModes,
-    invitation: IWorkspaceMemberInvitation | undefined,
-    email: string | undefined
-  ) => {
-    if (invitation && email && invitation.email === email && invitation.workspace) {
-      const workspace = invitation.workspace;
+  const getHeaderSubHeader = (mode: EAuthModes, current: IWorkspaceMemberInvitation | undefined) => {
+    if (current?.workspace) {
+      const workspace = current.workspace;
       return {
         header: (
           <div className="relative inline-flex items-center gap-2">
@@ -75,7 +70,7 @@
     return Titles[mode];
   };
 
-  const { header, subHeader } = getHeaderSubHeader(authMode, invitation || undefined, invitationEmail);
+  const { header, subHeader } = getHeaderSubHeader(authMode, invitation || undefined);
 
   if (isLoading)
     return (
```

`web/apps/web/core/components/account/auth-forms/password.tsx`（完整内容）：

```text
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { FormEvent } from "react";
import { useMemo, useRef, useState } from "react";
import { observer } from "mobx-react";
// icons
import { CloseCircleOutline, HideOutline, ShowOutline, WarningCircleOutline } from "@makeplane/propel/icons";
// nerve imports
import { Banner } from "@makeplane/propel/components/banner";
import { Field } from "@makeplane/propel/components/field";
import { Input, InputGroup } from "@makeplane/propel/components/input";
import { E_PASSWORD_STRENGTH } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { PasswordStrengthIndicator, Spinner } from "@nerve/ui";
import { checkEmailValidity, getPasswordStrength } from "@nerve/utils";
// helpers
import { EAuthModes, errorMessageKey, fieldErrorKeys } from "@/helpers/authentication.helper";
// hooks
import { useUser } from "@/hooks/store/user";

type Props = {
  mode: EAuthModes;
};

type TPasswordFormValues = {
  email: string;
  password: string;
  confirm_password: string;
};

const defaultValues: TPasswordFormValues = {
  email: "",
  password: "",
  confirm_password: "",
};

/**
 * The sign-in and sign-up form (M2 design 7.3). It calls nerve through the store; on success the token
 * manager keeps the session and AuthenticationWrapper moves on. A refusal shows here, the email kept: the
 * problem's message above the form, a field's message under the field.
 */
export const AuthPasswordForm = observer(function AuthPasswordForm(props: Props) {
  const { mode } = props;
  const { t } = useTranslation();
  const { signIn, signUp } = useUser();
  // ref
  const emailInputRef = useRef<HTMLInputElement>(null);
  // states
  const [passwordFormData, setPasswordFormData] = useState<TPasswordFormValues>(defaultValues);
  const [showPassword, setShowPassword] = useState({
    password: false,
    retypePassword: false,
  });
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isPasswordInputFocused, setIsPasswordInputFocused] = useState(false);
  const [isRetryPasswordInputFocused, setIsRetryPasswordInputFocused] = useState(false);
  const [isBannerMessage, setBannerMessage] = useState(false);
  const [isEmailInputFocused, setIsEmailInputFocused] = useState(false);
  // the last refusal of nerve, until the form changes
  const [submitError, setSubmitError] = useState<unknown>(undefined);

  const handleShowPassword = (key: keyof typeof showPassword) =>
    setShowPassword((prev) => ({ ...prev, [key]: !prev[key] }));

  // an edit answers the last message, whichever it was
  const handleFormChange = (key: keyof TPasswordFormValues, value: string) => {
    setSubmitError(undefined);
    setBannerMessage(false);
    setPasswordFormData((prev) => ({ ...prev, [key]: value }));
  };

  const isEmailInvalid = passwordFormData.email.length > 0 && !checkEmailValidity(passwordFormData.email);

  const isButtonDisabled = useMemo(
    () =>
      isSubmitting ||
      passwordFormData.email.length === 0 ||
      isEmailInvalid ||
      !passwordFormData.password ||
      (mode === EAuthModes.SIGN_UP && passwordFormData.password !== passwordFormData.confirm_password),
    [
      isSubmitting,
      isEmailInvalid,
      mode,
      passwordFormData.confirm_password,
      passwordFormData.email,
      passwordFormData.password,
    ]
  );

  const password = passwordFormData.password;
  const confirmPassword = passwordFormData.confirm_password;
  const renderPasswordMatchError = !isRetryPasswordInputFocused || confirmPassword.length >= password.length;
  const fieldErrors = fieldErrorKeys(submitError);
  const hasFieldErrors = Object.keys(fieldErrors).length > 0;

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (mode === EAuthModes.SIGN_UP && getPasswordStrength(password) !== E_PASSWORD_STRENGTH.STRENGTH_VALID) {
      setBannerMessage(true);
      return;
    }
    setIsSubmitting(true);
    setSubmitError(undefined);
    const credentials = { email: passwordFormData.email, password };
    try {
      await (mode === EAuthModes.SIGN_IN ? signIn(credentials) : signUp(credentials));
    } catch (error) {
      setSubmitError(error);
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <>
      {isBannerMessage && mode === EAuthModes.SIGN_UP && (
        <Banner
          placement="inline"
          variant="danger"
          description={t("auth.sign_up.errors.password.strength")}
          onDismiss={() => setBannerMessage(false)}
        />
      )}
      {submitError !== undefined && !hasFieldErrors && (
        <Banner
          placement="inline"
          variant="danger"
          description={t(errorMessageKey(submitError))}
          onDismiss={() => setSubmitError(undefined)}
        />
      )}
      <form className="space-y-4" noValidate onSubmit={(event) => void handleSubmit(event)}>
        <div className="space-y-1">
          <label htmlFor="email" className="text-13 font-medium text-tertiary">
            {t("auth.common.email.label")}
          </label>
          <Field name="email" invalid={(!isEmailInputFocused && isEmailInvalid) || Boolean(fieldErrors.email)}>
            <InputGroup
              size="2xl"
              onFocus={() => setIsEmailInputFocused(true)}
              onBlur={() => setIsEmailInputFocused(false)}
            >
              <Input
                size="2xl"
                id="email"
                name="email"
                type="email"
                value={passwordFormData.email}
                onChange={(e) => handleFormChange("email", e.target.value)}
                placeholder={t("auth.common.email.placeholder")}
                autoComplete="email"
                ref={emailInputRef}
              />
              {passwordFormData.email.length > 0 && (
                <button
                  type="button"
                  className="grid size-5 place-items-center"
                  onClick={() => {
                    handleFormChange("email", "");
                    emailInputRef.current?.focus();
                  }}
                  aria-label={t("aria_labels.auth_forms.clear_email")}
                  tabIndex={-1}
                >
                  <CloseCircleOutline className="size-5 text-placeholder" />
                </button>
              )}
            </InputGroup>
          </Field>
          {isEmailInvalid && !isEmailInputFocused && (
            <p className="flex items-center gap-1 px-0.5 text-11 text-danger-primary">
              <WarningCircleOutline height={12} width={12} />
              {t("auth.common.email.errors.invalid")}
            </p>
          )}
          {fieldErrors.email && <p className="px-0.5 text-11 text-danger-primary">{t(fieldErrors.email)}</p>}
        </div>

        <div className="space-y-1">
          <label htmlFor="password" className="text-13 font-medium text-tertiary">
            {mode === EAuthModes.SIGN_IN ? t("auth.common.password.label") : t("auth.common.password.set_password")}
          </label>
          <Field name="password" invalid={Boolean(fieldErrors.password)}>
            <InputGroup size="2xl">
              <Input
                size="2xl"
                type={showPassword.password ? "text" : "password"}
                id="password"
                name="password"
                value={passwordFormData.password}
                onChange={(e) => handleFormChange("password", e.target.value)}
                placeholder={t("auth.common.password.placeholder")}
                onFocus={() => setIsPasswordInputFocused(true)}
                onBlur={() => setIsPasswordInputFocused(false)}
                autoComplete={mode === EAuthModes.SIGN_IN ? "current-password" : "new-password"}
              />
              <button
                type="button"
                onClick={() => handleShowPassword("password")}
                className="grid size-5 place-items-center"
                aria-label={t(
                  showPassword.password
                    ? "aria_labels.auth_forms.hide_password"
                    : "aria_labels.auth_forms.show_password"
                )}
              >
                {showPassword.password ? (
                  <HideOutline className="size-5 text-placeholder" />
                ) : (
                  <ShowOutline className="size-5 text-placeholder" />
                )}
              </button>
            </InputGroup>
          </Field>
          {fieldErrors.password && <p className="px-0.5 text-11 text-danger-primary">{t(fieldErrors.password)}</p>}
          {mode === EAuthModes.SIGN_UP &&
            password.length > 0 &&
            getPasswordStrength(password) !== E_PASSWORD_STRENGTH.STRENGTH_VALID && (
              <PasswordStrengthIndicator password={password} isFocused={isPasswordInputFocused} />
            )}
        </div>

        {mode === EAuthModes.SIGN_UP && (
          <div className="space-y-1">
            <label htmlFor="confirm-password" className="text-13 font-medium text-tertiary">
              {t("auth.common.password.confirm_password.label")}
            </label>
            <InputGroup size="2xl">
              <Input
                size="2xl"
                type={showPassword.retypePassword ? "text" : "password"}
                id="confirm-password"
                name="confirm_password"
                value={passwordFormData.confirm_password}
                onChange={(e) => handleFormChange("confirm_password", e.target.value)}
                placeholder={t("auth.common.password.confirm_password.placeholder")}
                onFocus={() => setIsRetryPasswordInputFocused(true)}
                onBlur={() => setIsRetryPasswordInputFocused(false)}
                autoComplete="new-password"
              />
              <button
                type="button"
                className="grid size-5 place-items-center"
                aria-label={t(
                  showPassword.retypePassword
                    ? "aria_labels.auth_forms.hide_password"
                    : "aria_labels.auth_forms.show_password"
                )}
                onClick={() => handleShowPassword("retypePassword")}
              >
                {showPassword.retypePassword ? (
                  <HideOutline className="size-5 text-placeholder" />
                ) : (
                  <ShowOutline className="size-5 text-placeholder" />
                )}
              </button>
            </InputGroup>
            {!!confirmPassword && password !== confirmPassword && renderPasswordMatchError && (
              <span className="text-13 text-danger-primary">{t("auth.common.password.errors.match")}</span>
            )}
          </div>
        )}

        <Button type="submit" variant="primary" className="w-full" size="xl" disabled={isButtonDisabled}>
          {isSubmitting ? (
            <Spinner height="20px" width="20px" />
          ) : mode === EAuthModes.SIGN_IN ? (
            t("common.go_to_workspace")
          ) : (
            "Create account"
          )}
        </Button>
      </form>
    </>
  );
});
```

- [ ] **Step 3: 修改密码的错误**

`web/apps/web/core/components/settings/profile/content/pages/security.tsx`（对 Task 7 版本的差异）：

```diff
--- a/web/apps/web/core/components/settings/profile/content/pages/security.tsx
+++ b/web/apps/web/core/components/settings/profile/content/pages/security.tsx
@@ -20,9 +20,11 @@
 // components
 import { ProfileSettingsHeading } from "@/components/settings/profile/heading";
 // helpers
-import { authErrorHandler, EAuthenticationErrorCodes, passwordErrors } from "@/helpers/authentication.helper";
+import { errorMessageKey, fieldErrorKeys } from "@/helpers/authentication.helper";
 // hooks
 import { useUser } from "@/hooks/store/user";
+// lib
+import { ApiError } from "@/lib/api-error";
 
 export interface FormValues {
   old_password: string;
@@ -84,25 +86,21 @@
         message: t("auth.common.password.toast.change_password.success.message"),
       });
     } catch (error: unknown) {
-      const err = error as Error & { error_code?: string };
-      const code = err.error_code?.toString();
-      const errorInfo = code ? authErrorHandler(code as EAuthenticationErrorCodes) : undefined;
-
+      // A refusal shows under the field it is about, anything else in a toast (M2 design 7.3).
+      if (error instanceof ApiError && error.problem?.code === "identity.current_password_incorrect") {
+        setError("old_password", { type: "manual", message: t(errorMessageKey(error)) });
+        return;
+      }
+      const newPassword = fieldErrorKeys(error).new_password;
+      if (newPassword !== undefined) {
+        setError("new_password", { type: "manual", message: t(newPassword) });
+        return;
+      }
       setToast({
         type: TOAST_TYPE.ERROR,
-        title: errorInfo?.title ?? t("auth.common.password.toast.change_password.error.title"),
-        message:
-          typeof errorInfo?.message === "string"
-            ? errorInfo.message
-            : t("auth.common.password.toast.change_password.error.message"),
+        title: t("auth.common.password.toast.change_password.error.title"),
+        message: t(errorMessageKey(error)),
       });
-
-      if (code && passwordErrors.includes(code as EAuthenticationErrorCodes)) {
-        setError("new_password", {
-          type: "manual",
-          message: errorInfo?.message?.toString() || t("auth.common.password.toast.change_password.error.message"),
-        });
-      }
     }
   };
 
```

- [ ] **Step 4: 密码规则**

`web/packages/utils/src/auth.ts`（对 `f27434c` 的差异）：

```diff
--- a/web/packages/utils/src/auth.ts
+++ b/web/packages/utils/src/auth.ts
@@ -6,6 +6,14 @@
 
 import { E_PASSWORD_STRENGTH } from "@nerve/constants";
 
+// The server's password lengths, in UTF-16 code units like password.length
+// (server/internal/modules/identity/domain/password.go).
+const MIN_PASSWORD_LENGTH = 8;
+const MAX_PASSWORD_LENGTH = 128;
+
+const isLengthValid = (password: string) =>
+  password.length >= MIN_PASSWORD_LENGTH && password.length <= MAX_PASSWORD_LENGTH;
+
 /**
  * Calculate password strength based on various criteria
  */
@@ -14,7 +22,7 @@
     return E_PASSWORD_STRENGTH.EMPTY;
   }
 
-  if (password.length < 8) {
+  if (!isLengthValid(password)) {
     return E_PASSWORD_STRENGTH.LENGTH_NOT_VALID;
   }
 
@@ -43,8 +51,8 @@
 export const getPasswordCriteria = (password: string): PasswordCriteria[] => [
   {
     key: "length",
-    label: "Min 8 characters",
-    isValid: password.length >= 8,
+    label: "8–128 characters",
+    isValid: isLengthValid(password),
   },
   {
     key: "uppercase",
```

`web/packages/utils/src/auth.test.ts`（新文件）：

```ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import { E_PASSWORD_STRENGTH } from "@nerve/constants";
import { getPasswordCriteria, getPasswordStrength } from "./auth";

// The strength hint follows the server's composition rules (M2 design 3.8, 7.3): 8–128 UTF-16 code units,
// an upper-case letter, a lower-case letter, a digit and a special character. The common-password list is
// the server's alone.

const strong = (length: number) => "Aa1!" + "x".repeat(length - 4);

describe("getPasswordStrength", () => {
  it("takes 8 to 128 characters", () => {
    expect(getPasswordStrength(strong(8))).toBe(E_PASSWORD_STRENGTH.STRENGTH_VALID);
    expect(getPasswordStrength(strong(128))).toBe(E_PASSWORD_STRENGTH.STRENGTH_VALID);
  });

  it("refuses fewer than 8 or more than 128 characters, as the server does", () => {
    expect(getPasswordStrength(strong(7))).toBe(E_PASSWORD_STRENGTH.LENGTH_NOT_VALID);
    expect(getPasswordStrength(strong(129))).toBe(E_PASSWORD_STRENGTH.LENGTH_NOT_VALID);
  });

  it("counts UTF-16 code units: 64 emoji are 128 of them", () => {
    expect(getPasswordStrength("Aa1!" + "😀".repeat(62))).toBe(E_PASSWORD_STRENGTH.STRENGTH_VALID);
    expect(getPasswordStrength("Aa1!x" + "😀".repeat(62))).toBe(E_PASSWORD_STRENGTH.LENGTH_NOT_VALID);
  });

  it("needs every class of character", () => {
    expect(getPasswordStrength("aa1!xxxx")).toBe(E_PASSWORD_STRENGTH.STRENGTH_NOT_VALID);
    expect(getPasswordStrength("")).toBe(E_PASSWORD_STRENGTH.EMPTY);
  });
});

describe("getPasswordCriteria", () => {
  it("shows the length rule as unmet above 128 characters", () => {
    const length = (password: string) => getPasswordCriteria(password).find((c) => c.key === "length");
    expect(length(strong(128))).toMatchObject({ label: "8–128 characters", isValid: true });
    expect(length(strong(129))?.isValid).toBe(false);
    expect(length(strong(7))?.isValid).toBe(false);
  });
});
```

`web/packages/ui/src/form-fields/password/helper.tsx`（对 `f27434c` 的差异）：

```diff
--- a/web/packages/ui/src/form-fields/password/helper.tsx
+++ b/web/packages/ui/src/form-fields/password/helper.tsx
@@ -25,7 +25,7 @@
       };
     case E_PASSWORD_STRENGTH.LENGTH_NOT_VALID:
       return {
-        message: "Password is too short",
+        message: "Password must be 8–128 characters",
         textColor: "text-danger-primary",
         activeFragments: 1,
       };
```

- [ ] **Step 5: 删除 Plane 的认证类型；关键词规则；上限**

```bash
git rm web/packages/types/src/auth.ts
```

`web/packages/types/src/index.ts`（对 Task 6 版本的差异）：

```diff
--- a/web/packages/types/src/index.ts
+++ b/web/packages/types/src/index.ts
@@ -5,7 +5,6 @@
  */
 
 export * from "./api_token";
-export * from "./auth";
 export * from "./calendar";
 export * from "./charts";
 export * from "./common";
```

`tools/keywords.json`（对 Task 8 版本的差异）：

```diff
--- a/tools/keywords.json
+++ b/tools/keywords.json
@@ -1798,8 +1798,94 @@
         "files": {
           "hit": ["web/apps/web/core/services/user.service.ts", "web/apps/web/core/services/instance.service.ts"],
           "miss": ["e2e/stories/smoke/s2-web-app.spec.ts", "docs/v0/M2-auth/M2-design.md"]
+        }
+      }
+    },
+    {
+      "id": "csrf",
+      "phase": "M2/P4",
+      "why": "Plane 的会话 cookie 要 CSRF 令牌（取令牌、表单隐藏字段、X-CSRFTOKEN 头）；Nerve 的请求带 Authorization: Bearer，不用 cookie，也就没有 CSRF（M2 设计 7.2，M1-P3 交接）",
+      "files": {
+        "source": "^web/.*\\.(?:[cm]?[jt]sx?|json)$",
+        "flags": ""
+      },
+      "content": {
+        "source": "csrf",
+        "flags": "i"
+      },
+      "samples": {
+        "hit": [
+          "        <input type=\"hidden\" name=\"csrfmiddlewaretoken\" />",
+          "      headers: { \"X-CSRFTOKEN\": csrfToken },",
+          "  requestCSRFToken = async (): Promise<ICsrfTokenData> =>"
+        ],
+        "miss": ["  request.headers.set(\"Authorization\", `Bearer ${token}`);"],
+        "files": {
+          "hit": ["web/apps/web/core/services/auth.service.ts", "web/packages/types/src/auth.ts"],
+          "miss": ["docs/v0/M2-auth/M2-design.md", "server/internal/modules/identity/adapter/http/auth.go"]
+        }
+      }
+    },
+    {
+      "id": "plane-auth-urls",
+      "phase": "M2/P4",
+      "why": "Plane 的认证接口在根路径 /auth/ 下（登录、注册、退出、改密码、取 CSRF 令牌），登录页用原生表单 POST 过去；Nerve 的认证接口都在 /api/v0/auth 和 /api/v0/me 下，由生成的客户端调用（M2 设计 7.2、7.3）",
+      "files": {
+        "source": "^web/.*\\.(?:[cm]?[jt]sx?|json)$",
+        "flags": ""
+      },
+      "content": {
+        "source": "(?<![\\w.@-])/auth/",
+        "flags": ""
+      },
+      "samples": {
+        "hit": [
+          "        action={`/auth/${mode === EAuthModes.SIGN_IN ? \"sign-in\" : \"sign-up\"}/`}",
+          "      form.action = \"/auth/sign-out/\";",
+          "    return this.post(`/auth/change-password/`, data, {"
+        ],
+        "miss": [
+          "    return unwrap(await publicClient.POST(\"/api/v0/auth/login\", { body }));",
+          "import Unauthorized from \"@/app/assets/auth/unauthorized.svg?url\";"
+        ],
+        "files": {
+          "hit": [
+            "web/apps/web/core/components/account/auth-forms/password.tsx",
+            "web/apps/web/core/services/user.service.ts"
+          ],
+          "miss": ["docs/v0/M2-auth/M2-design.md", "e2e/stories/auth/a1-sign-up.spec.ts"]
         }
       }
+    },
+    {
+      "id": "auth-error-code",
+      "phase": "M2/P4",
+      "why": "Plane 登录失败时跳回登录页，把数字错误码放在地址的 error_code 里，前端按 EAuthenticationErrorCodes 查文案；Nerve 的错误是 problem+json 的 code，登录页就地显示，文案表由测试对着 openapi.yaml 核对（M2 设计 3.11、7.3）",
+      "files": {
+        "source": "^web/.*\\.(?:[cm]?[jt]sx?|json)$",
+        "flags": ""
+      },
+      "content": {
+        "source": "error_code|EAuthenticationErrorCodes",
+        "flags": ""
+      },
+      "samples": {
+        "hit": [
+          "  const error_code = searchParams.get(\"error_code\");",
+          "      const err = error as Error & { error_code?: string };",
+          "  if (code && passwordErrors.includes(code as EAuthenticationErrorCodes)) {"
+        ],
+        "miss": [
+          "      if (error instanceof ApiError && error.problem?.code === \"identity.current_password_incorrect\") {"
+        ],
+        "files": {
+          "hit": [
+            "web/apps/web/core/components/account/auth-forms/auth-root.tsx",
+            "web/apps/web/helpers/authentication.helper.ts"
+          ],
+          "miss": ["server/internal/shared/error.go", "docs/v0/M2-auth/M2-design.md"]
+        }
+      }
     }
   ],
   "exceptions": [
```

`web/apps/web/package.json`（对 Task 8 版本的差异）：

```diff
--- a/web/apps/web/package.json
+++ b/web/apps/web/package.json
@@ -7,7 +7,7 @@
   "scripts": {
     "dev": "react-router dev --port 3000",
     "build": "react-router build",
-    "check:lint": "node ../../../tools/lint-cap.mjs 554",
+    "check:lint": "node ../../../tools/lint-cap.mjs 551",
     "check:types": "react-router typegen && tsc --noEmit",
     "check:format": "oxfmt --check .",
     "fix:format": "oxfmt .",
```

Run: `grep -rniE "csrfmiddlewaretoken|X-CSRFTOKEN|csrf|EAuthenticationErrorCodes|error_code" web --exclude-dir=node_modules --exclude-dir=dist --exclude-dir=build --exclude-dir=.react-router --exclude-dir=.turbo`
Expected: 没有输出（M2 的完成线：`git grep -n -E "csrfmiddlewaretoken|X-CSRFTOKEN" -- web` 没有输出）。

- [ ] **Step 6: 检查**

Run: `pnpm -C web/apps/web exec vitest run helpers`、`pnpm -C web/packages/utils exec vitest run src/auth.test.ts`
Expected: 9 个、5 个测试通过。

Run: `make lint-web`、`make knip`、`make test-web`
Expected: 都通过；web 的 oxlint 警告正好 551 个；关键词守卫 56 条规则、3 条例外，没有命中。

- [ ] **Step 7: 提交**

```bash
git add -A web/apps/web/core web/apps/web/helpers web/apps/web/vitest.config.ts web/apps/web/package.json web/packages/types/src web/packages/utils/src web/packages/ui/src/form-fields/password/helper.tsx web/packages/i18n/src/locales tools/keywords.json
```
```bash
git commit -m "feat(M2/P4): sign-in and sign-up pages on the token manager; errors in place

The forms call the API and hand the tokens to the token manager; a refusal
shows under its field or above the form, from message tables that a test
keeps equal to the codes of openapi.yaml. The change-password page shows
its errors by code. Passwords are 8-128 UTF-16 code units, as on the
server. Plane's CSRF, form posts to /auth/ and error_code are gone, and
the keyword guard keeps them out.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** 14 个新测试通过；Step 5 的 `grep` 没有输出；上限 551。

---

### Task 11: `webui` 的 CSP

**Files:**
- Create: `server/internal/platform/webui/csp.go`、`csp_test.go`
- Modify: `server/internal/platform/webui/handler.go`、`server/internal/bootstrap/headers_test.go`

**Interfaces:**
- Produces（spec 2.12，M2 设计 8.3；M0-P5 交接）：
  - `contentSecurityPolicy(index []byte) string`：`index.html` 中每个没有 `src` 的 `<script>`（标签不区分大小写，结束标签可以带空白）的内容算 SHA-256，base64 编码，去重后按出现的顺序放进 `script-src 'self' 'sha256-…'`；其余指令固定：`default-src 'self'`、`style-src 'self' 'unsafe-inline'`、`img-src 'self' data: blob:`、`font-src 'self' data:`、`connect-src 'self'`、`object-src 'none'`、`base-uri 'none'`、`form-action 'self'`、`frame-ancestors 'none'`；
  - `webui.Handler` 构造时读 `index.html`、算出 CSP；只有 `.html` 的响应带 `Content-Security-Policy`；资源文件、找不到的资源、没有构建前端时的回答都不带。前端重新构建后不需要手工更新。
- 安全响应头仍在 `httpserver` 的固定链上（M2/P2）；两层分开的理由见 M2 设计 8.3。

**Tests:**
- `csp_test.go`（3 个）：一个像 React Router 构建出的页面（内联脚本、模块脚本、带 `src` 的脚本、大写的 `<SCRIPT>`、重复的脚本），哈希在包外用 Python 的 `hashlib`、`base64` 算出，策略逐字相同；没有内联脚本的页面只有 `script-src 'self'`；`GET /`、带查询的页面、`HEAD` 页面带策略，资源文件、`site.webmanifest.json`、找不到的资源、没有构建的前端都不带。
- `headers_test.go` 的 `TestOnlyPagesHaveAContentSecurityPolicy`：经整个服务，页面带策略，资源文件、`/healthz`、`/api/v0/instance` 和 `/api/` 下的 404 都不带。

- [ ] **Step 1: 策略和处理器**

`server/internal/platform/webui/csp.go`（新文件）：

```go
package webui

import (
	"crypto/sha256"
	"encoding/base64"
	"regexp"
	"slices"
	"strings"
)

// Every script element of a page, with its attributes and its text. A script's
// text is raw: it ends at the first "</script", as the browser ends it, and
// React Router escapes "<" in the data it writes inline.
var (
	scriptElement = regexp.MustCompile(`(?is)<script\b([^>]*)>(.*?)</script\s*>`)
	srcAttribute  = regexp.MustCompile(`(?i)(?:^|\s)src\s*=`)
)

// contentSecurityPolicy is the CSP of the app's pages (M2 design 8.3), for
// the page index: scripts come from nerve only, or are the page's own inline
// scripts, each allowed by the SHA-256 of its text, so a rebuilt frontend
// needs no change here. Styles may be inline (React's style attributes and
// popper's positions); nothing else is allowed from anywhere but nerve.
func contentSecurityPolicy(index []byte) string {
	scripts := []string{"'self'"}
	for _, m := range scriptElement.FindAllSubmatch(index, -1) {
		if srcAttribute.Match(m[1]) {
			continue // a script file: 'self' covers it
		}
		sum := sha256.Sum256(m[2])
		source := "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
		if !slices.Contains(scripts, source) {
			scripts = append(scripts, source)
		}
	}
	return strings.Join([]string{
		"default-src 'self'",
		"script-src " + strings.Join(scripts, " "),
		"style-src 'self' 'unsafe-inline'",
		"img-src 'self' data: blob:",
		"font-src 'self' data:",
		"connect-src 'self'",
		"object-src 'none'",
		"base-uri 'none'",
		"form-action 'self'",
		"frame-ancestors 'none'",
	}, "; ")
}
```

`server/internal/platform/webui/handler.go`（对 `f27434c` 的差异）：

```diff
--- a/server/internal/platform/webui/handler.go
+++ b/server/internal/platform/webui/handler.go
@@ -27,17 +27,24 @@
 //     as a missing file instead of turning into HTML;
 //   - any other path: index.html, and the client-side router renders the page.
 //
+// An HTML file goes out with the Content-Security-Policy made from index.html
+// when the handler is made (M2 design 8.3); other files and errors have none.
+//
 // Hidden files (any name part starting with "."), such as dist/.gitkeep, are
 // never served. Without index.html every request answers 404 with a hint.
 // Mount it on the pattern "/" so that /api/ and the probes keep their routes.
 func Handler(files fs.FS) http.Handler {
-	_, err := fs.Stat(files, indexFile)
-	return &handler{files: files, built: err == nil}
+	index, err := fs.ReadFile(files, indexFile)
+	if err != nil {
+		return &handler{files: files}
+	}
+	return &handler{files: files, built: true, csp: contentSecurityPolicy(index)}
 }
 
 type handler struct {
 	files fs.FS
 	built bool
+	csp   string // the pages' Content-Security-Policy, once built
 }
 
 func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
@@ -80,6 +87,9 @@
 		cache = cacheImmutable
 	}
 	w.Header().Set("Cache-Control", cache)
+	if path.Ext(name) == ".html" {
+		w.Header().Set("Content-Security-Policy", h.csp)
+	}
 	// ServeFileFS sets Content-Type from the extension (sniffing the content
 	// otherwise), answers HEAD and Range, and redirects /index.html to ./.
 	http.ServeFileFS(w, r, h.files, name)
```

- [ ] **Step 2: 测试**

`server/internal/platform/webui/csp_test.go`（新文件）：

```go
package webui

import (
	"net/http"
	"testing"
	"testing/fstest"
)

// A page like the one React Router builds: inline scripts, one of them a
// module, a script file, and a script repeated. The hashes were computed
// apart from this package (Python's hashlib and base64).
const scriptedIndex = `<!DOCTYPE html><html><head>` +
	`<script>document.documentElement.dataset.theme = "dark";</script>` +
	`<link rel="modulepreload" href="/assets/entry.client-a1.js"/>` +
	`<script src="/assets/legacy-c3.js"></script>` +
	`</head><body>` +
	`<SCRIPT>window.__ctx = {"a":1};</SCRIPT >` +
	"<script type=\"module\" async=\"\">import \"/assets/manifest-1.js\";\n  window.x = 1;</script>" +
	`<script>window.__ctx = {"a":1};</script>` +
	`</body></html>`

const scriptedPolicy = "default-src 'self'; " +
	"script-src 'self' 'sha256-RH/36EFn2TnZtn39Gf6aq1rnKqEXnEQbfAXoYkwhRP0=' " +
	"'sha256-CACaKUMP3bUaeK7JgU+grVR6sA9VUsIgZNwdQnbXMCA=' 'sha256-p61WDS2Nz6pOwhZMG1Gv1r9QDC4titM1f6gnicWbCZU='; " +
	"style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; font-src 'self' data:; connect-src 'self'; " +
	"object-src 'none'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'"

func TestPolicyAllowsEachInlineScriptByItsHash(t *testing.T) {
	if got := contentSecurityPolicy([]byte(scriptedIndex)); got != scriptedPolicy {
		t.Errorf("policy =\n%s\nwant\n%s", got, scriptedPolicy)
	}
}

func TestPolicyOfAPageWithoutInlineScriptsAllowsOnlyNervesScripts(t *testing.T) {
	want := "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; " +
		"font-src 'self' data:; connect-src 'self'; object-src 'none'; base-uri 'none'; form-action 'self'; " +
		"frame-ancestors 'none'"
	if got := contentSecurityPolicy([]byte(indexHTML)); got != want {
		t.Errorf("policy =\n%s\nwant\n%s", got, want)
	}
}

// The policy goes on the pages, made from the index.html being served, and on
// nothing else: files, missing assets and the answers of an unbuilt frontend
// have none.
func TestOnlyPagesCarryThePolicy(t *testing.T) {
	scripted := fstest.MapFS{
		"index.html":                {Data: []byte(scriptedIndex)},
		"site.webmanifest.json":     {Data: []byte(`{"name":"Nerve"}`)},
		"assets/entry.client-a1.js": {Data: []byte("export {};")},
	}
	h := Handler(scripted)
	for _, tt := range []struct{ method, target, want string }{
		{http.MethodGet, "/", scriptedPolicy},
		{http.MethodGet, "/sign-up?next_path=%2F", scriptedPolicy},
		{http.MethodHead, "/onboarding", scriptedPolicy},
		{http.MethodGet, "/assets/entry.client-a1.js", ""},
		{http.MethodGet, "/site.webmanifest.json", ""},
		{http.MethodGet, "/assets/entry.client-old.js", ""},
	} {
		rec := serve(h, tt.method, tt.target)
		if got := rec.Result().Header.Get("Content-Security-Policy"); got != tt.want {
			t.Errorf("%s %s (%d): Content-Security-Policy = %q, want %q", tt.method, tt.target, rec.Code, got, tt.want)
		}
	}

	unbuilt := serve(Handler(fstest.MapFS{".gitkeep": {}}), http.MethodGet, "/")
	if got := unbuilt.Result().Header.Get("Content-Security-Policy"); got != "" {
		t.Errorf("unbuilt GET / (%d): Content-Security-Policy = %q, want none", unbuilt.Code, got)
	}
}
```

`server/internal/bootstrap/headers_test.go`（对 `f27434c` 的差异）：

```diff
--- a/server/internal/bootstrap/headers_test.go
+++ b/server/internal/bootstrap/headers_test.go
@@ -2,6 +2,7 @@
 
 import (
 	"net/http"
+	"strings"
 	"testing"
 	"testing/fstest"
 )
@@ -26,6 +27,34 @@
 	}
 }
 
+// The CSP is the web UI's, on its pages only (M2 design 8.3): the probes and
+// the API's answers have none.
+func TestOnlyPagesHaveAContentSecurityPolicy(t *testing.T) {
+	base := startApp(t, testConfig(t, unreachableDB, false), fstest.MapFS{})
+	for _, tt := range []struct {
+		path string
+		page bool
+	}{
+		{"/", true},
+		{"/settings/profile/general", true},
+		{"/" + testAsset, false},
+		{"/healthz", false},
+		{"/api/v0/instance", false},
+		{"/api/v0/nope", false},
+	} {
+		res, err := client.Get(base + tt.path)
+		if err != nil {
+			t.Fatal(err)
+		}
+		_ = res.Body.Close()
+
+		csp := res.Header.Get("Content-Security-Policy")
+		if got := strings.HasPrefix(csp, "default-src 'self'; script-src 'self';"); got != tt.page {
+			t.Errorf("GET %s (%d): Content-Security-Policy = %q, want a page's policy: %v", tt.path, res.StatusCode, csp, tt.page)
+		}
+	}
+}
+
 // No cache may store an API response (M2 design 8.3): an answer, the /api/
 // fallback's 404 and a 401 alike. The web UI's files keep their own
 // caching: index.html is revalidated, a hashed asset kept for good.
```

- [ ] **Step 3: 检查**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`（32 个包）。

Run: `make lint-web`、`make knip`、`make test-web`
Expected: 都通过。

Run: `make e2e`
Expected: 22 个故事照旧通过（故事还没改；这一步证明 Task 1–11 没有弄坏它们，也证明 CSP 没有挡住页面的脚本）。

- [ ] **Step 4: 提交**

```bash
git add server/internal/platform/webui server/internal/bootstrap/headers_test.go
```
```bash
git commit -m "feat(M2/P4): a Content-Security-Policy on the web UI's pages

webui hashes the inline scripts of the embedded index.html when the
handler is made and sets the policy on HTML answers only: scripts from
nerve or with those hashes, connections to nerve only, no objects, no
base, no framing. Assets, API answers and an unbuilt frontend carry none.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** 4 个新测试通过；`make e2e` 通过。

---

### Task 12: 端到端：`signedInPage`、`watchPage`；A1–A3、A10、A15 的页面版本和 S2

**Files:**
- Modify: `e2e/tsconfig.json`、`e2e/fixtures/test.ts`
- Modify（过渡）: `e2e/fixtures/auth.ts`
- Create: `e2e/fixtures/auth-pages.ts`、`e2e/fixtures/browser.ts`
- Modify: `e2e/stories/identity/a1-sign-up.spec.ts`、`a2-sign-up-refused.spec.ts`、`a3-sign-in.spec.ts`、`a10-onboarding-profile.spec.ts`、`a15-sign-in-limits.spec.ts`、`e2e/stories/smoke/s2-web-app.spec.ts`

**Interfaces:**
- Produces（spec 2.14，M2 设计 9.5；M0-P6 交接）：
  - `e2e/tsconfig.json` 的 `lib` 加 `DOM`（`page.evaluate` 和 `addInitScript` 里的代码在浏览器中运行）；
  - `auth.ts`（过渡）：`AuthRecord`；`signInContext(context, baseURL, tokens)`：`context.addInitScript` 在页面的任何脚本之前把一条新记录（`login_id` 是 16 个随机字节）写进这个 nerve 的 localStorage，只在第一次加载时写（标记 `nerve.auth.e2e-seeded`），之后的加载、刷新和其他标签页看到的是页面自己续期或删除之后的记录；只作用于这个 nerve 的源；`recordOf(page)`；
  - `test.ts`：fixture `signedInPage(tokens, baseURL?)`：对测试的页面调用 `signInContext`（默认是本 worker 的 nerve），返回还没有加载任何东西的页面；
  - `auth-pages.ts`：`signInPath(path)`（测试自己的一份，不依赖前端的实现）、`formAlert(page)`（表单上方的 `role="alert"`）、`submitSignIn`（先清空邮箱再填，核对上一次的提示已消失，返回 `POST /api/v0/auth/login` 的状态码）、`fillSignUp`、`submitSignUp`；
  - `browser.ts`：`watchPage(page): PageWatch`，在页面第一次打开之前调用：接口请求（`<方法> <路径>`）、失败的接口请求、发往 `/api/v0` 之外的接口的请求（M3 的旧接口）、未处理的异常和拒绝、控制台的错误和警告、CSP 违规（`securitypolicyviolation` 事件经 `exposeBinding` 送回测试）。
- 故事的页面版本（接口版本不变，页面版本调用同一组断言函数）：
  - **A1**：注册后到 `/onboarding` 的资料步骤；没有 Cookie；记录只有 `refresh_token`、`login_id`，`login_id` 是 32 位十六进制；两个令牌的主体（刷新令牌去掉前缀，访问令牌的签名段）不出现在 sessionStorage、localStorage 的其他键、页面地址、任何请求的地址和任何控制台消息中；`expectRegistered`（UA、IP）；
  - **A2**：邮箱已存在（大写输入）：409，提示在表单上方，地址和输入框不变；不合规的密码：字段下方显示规则、表单上方的提示，一个请求都不发；`Password1!`、`Password1!~`：422，字段下方"This password is too common"，表单上方没有提示；关闭注册的 nerve：页头没有"Sign up"链接（开放的 nerve 有），两个邮箱都是 403"Sign-up is closed."；数据库没有新增；
  - **A3**：未登录打开 `/settings/profile/general?tab=x#y` → 登录页，`next_path` 整体编码；登录后原样回到这个地址；`//evil.example`、`/\evil.example`、`javascript:alert(1)`、`/\t/evil.example` 四个值（各自新的浏览器上下文）登录后都落到 `/create-workspace`；错误密码和不存在的邮箱：同一句提示，留在登录页，输入框不变，没有新会话；
  - **A10**：新账户第一次打开 `/onboarding`：填名字，等到这一步最后一个请求（`PATCH /api/v0/me/profile`）的 200，下一步出现；数据库中只有 `profile_complete` 为真，名字已保存；没有失败的接口请求、没有发往旧接口的请求、没有未处理的异常、控制台没有错误、没有 CSP 违规；
  - **A15**：限流很低的 nerve：同一个邮箱 401、401、429，另一个邮箱 401，第三个邮箱 429（按 IP）；每次的提示分别是"The email or the password is wrong."和"Too many attempts. Please try again later."；
  - **S2**：两个测试都用 `watchPage`：未登录时只请求 `GET /api/v0/instance`，没有失败的请求、CSP 违规、未处理的异常、控制台错误；首页带 `Content-Security-Policy`（`default-src 'self'; script-src 'self' 'sha256-…`），"Go to workspace"按钮出现；深链接跳到带 `next_path` 的登录页。

**Tests:** 页面版本 6 个（A1、A2、A10、A15 各 1 个，A3 2 个），S2 的 2 个改写。

- [ ] **Step 1: fixture**

`e2e/tsconfig.json`（完整内容）：

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "lib": ["ES2023", "DOM"],
    "module": "ESNext",
    "moduleResolution": "bundler",
    "verbatimModuleSyntax": true,
    "strict": true,
    "noUncheckedIndexedAccess": true,
    "skipLibCheck": true,
    "noEmit": true,
    "types": ["node"]
  },
  "include": ["**/*.ts"]
}
```

`e2e/fixtures/auth.ts`（对 `f27434c` 的差异）：

```diff
--- a/e2e/fixtures/auth.ts
+++ b/e2e/fixtures/auth.ts
@@ -1,11 +1,53 @@
+import { randomBytes } from "node:crypto";
+
 import type { components } from "@nerve/api-client";
-import { expect, type TestInfo } from "@playwright/test";
+import { expect, type BrowserContext, type Page, type TestInfo } from "@playwright/test";
 
 import type { Api } from "./api";
 
 export type AuthTokens = components["schemas"]["AuthTokens"];
 export type ApiTokenCreated = components["schemas"]["ApiTokenCreated"];
 
+/** The key of the token manager's record in localStorage (M2 design 7.1). */
+const authKey = "nerve.auth";
+
+/** The token manager's record: the refresh token, and the login_id of the sign-in it came from. */
+export interface AuthRecord {
+  refresh_token: string;
+  login_id: string;
+}
+
+/** The record a sign-in with tokens writes: a new login_id of 16 random bytes in hexadecimal. */
+function newRecord(tokens: AuthTokens): AuthRecord {
+  return { refresh_token: tokens.refresh_token, login_id: randomBytes(16).toString("hex") };
+}
+
+/**
+ * Signs the pages of context in with tokens at the nerve of baseURL (M2 design 9.5): before the first
+ * page there loads, its localStorage gets the record a sign-in writes, and the page refreshes it itself.
+ * Only that first load writes it: later loads, reloads and other tabs find the record the pages keep,
+ * refreshed or removed, as in a browser.
+ */
+export async function signInContext(context: BrowserContext, baseURL: string, tokens: AuthTokens): Promise<void> {
+  await context.addInitScript(
+    ({ origin, key, record }) => {
+      const seeded = `${key}.e2e-seeded`;
+      if (window.location.origin !== origin || localStorage.getItem(seeded) !== null) {
+        return;
+      }
+      localStorage.setItem(seeded, "1");
+      localStorage.setItem(key, record);
+    },
+    { origin: new URL(baseURL).origin, key: authKey, record: JSON.stringify(newRecord(tokens)) }
+  );
+}
+
+/** The record in the localStorage of page, or null when it has none. */
+export async function recordOf(page: Page): Promise<AuthRecord | null> {
+  const text = await page.evaluate((key) => localStorage.getItem(key), authKey);
+  return text === null ? null : (JSON.parse(text) as AuthRecord);
+}
+
 /** A password that meets the rules and is not common. */
 export const password = "Tr0ub4dor&3";
 
```

`e2e/fixtures/test.ts`（对 `f27434c` 的差异）：

```diff
--- a/e2e/fixtures/test.ts
+++ b/e2e/fixtures/test.ts
@@ -1,8 +1,9 @@
 import path from "node:path";
 
-import { test as base } from "@playwright/test";
+import { test as base, type Page } from "@playwright/test";
 
 import { createApi, type Api } from "./api";
+import { signInContext, type AuthTokens } from "./auth";
 import { createDatabase, openDatabase, templateDatabase, type Database } from "./db";
 import { nerveFixtureTimeoutMs, startNerve, type Nerve } from "./server";
 
@@ -25,6 +26,11 @@
    * fixture's own timeouts fire first.
    */
   nerveWith: (env: Record<string, string>) => Promise<Nerve>;
+  /**
+   * Signs the test's browser context in with tokens, at the worker's nerve or at baseURL (M2 design 9.5),
+   * and returns the test's page, which has loaded nothing yet: its first page refreshes the session.
+   */
+  signedInPage: (tokens: AuthTokens, baseURL?: string) => Promise<Page>;
   /** When the test fails, a pg_dump of the worker's database joins its trace, screenshot and nerve log. */
   databaseSnapshot: void;
 }
@@ -71,6 +77,12 @@
     });
     await Promise.all(started.map((nerve) => nerve.stop()));
   },
+  signedInPage: async ({ page, nerve }, use) => {
+    await use(async (tokens, baseURL = nerve.baseURL) => {
+      await signInContext(page.context(), baseURL, tokens);
+      return page;
+    });
+  },
   databaseSnapshot: [
     async ({ db }, use, testInfo) => {
       await use();
```

`e2e/fixtures/auth-pages.ts`（新文件）：

```ts
import { expect, type Page } from "@playwright/test";

// The sign-in and sign-up forms (M2 design 7.3), as a person fills them.

/** The address of the sign-in page that comes back to path after signing in (M2 design 3.18). */
export function signInPath(path: string): string {
  return `/?next_path=${encodeURIComponent(path)}`;
}

/** The message above a form, which says why nerve refused it. */
export function formAlert(page: Page) {
  return page.getByRole("alert");
}

/**
 * Fills the sign-in form, which page shows, and submits it: resolves with the status of nerve's answer to
 * the one login it sends. The address is typed anew, so the message of the last try is gone before this
 * one is sent, even when both have the same values.
 */
export async function submitSignIn(page: Page, email: string, password: string): Promise<number> {
  await page.getByLabel("Email", { exact: true }).clear();
  await page.getByLabel("Email", { exact: true }).fill(email);
  await page.getByLabel("Password", { exact: true }).fill(password);
  await expect(formAlert(page)).toHaveCount(0);
  const [response] = await Promise.all([
    page.waitForResponse((res) => new URL(res.url()).pathname === "/api/v0/auth/login"),
    page.getByRole("button", { name: "Go to workspace" }).click(),
  ]);
  return response.status();
}

/** Fills the sign-up form, which page shows, with password typed twice. */
export async function fillSignUp(page: Page, email: string, password: string): Promise<void> {
  await page.getByLabel("Email", { exact: true }).fill(email);
  await page.getByLabel("Set a password", { exact: true }).fill(password);
  await page.getByLabel("Confirm password", { exact: true }).fill(password);
  await expect(formAlert(page)).toHaveCount(0);
}

/** Fills the sign-up form and submits it: resolves with the status of nerve's answer to the registration. */
export async function submitSignUp(page: Page, email: string, password: string): Promise<number> {
  await fillSignUp(page, email, password);
  const [response] = await Promise.all([
    page.waitForResponse((res) => new URL(res.url()).pathname === "/api/v0/auth/register"),
    page.getByRole("button", { name: "Create account" }).click(),
  ]);
  return response.status();
}
```

`e2e/fixtures/browser.ts`（新文件）：

```ts
import type { Page, Request } from "@playwright/test";

/** What a page did that a story checks: its API calls, and what went wrong in it. */
export interface PageWatch {
  /** Every API request, as "<method> <path>". */
  readonly apiRequests: string[];
  /** API requests that failed: "<status> <method> <path>", or the browser's error for one without an answer. */
  readonly apiFailures: string[];
  /** Requests to the API of an older frontend, outside /api/v0 (M2 design 3.1): M3's domains, for instance. */
  readonly oldApiRequests: string[];
  /** Uncaught exceptions and unhandled rejections. */
  readonly pageErrors: string[];
  /** Console messages of type error. */
  readonly consoleErrors: string[];
  /** Console messages of type warning. */
  readonly consoleWarnings: string[];
  /** Content-Security-Policy violations, as "<directive> <blocked URI>" (M2 design 8.3). */
  readonly cspViolations: string[];
}

function apiPath(request: Request): string | undefined {
  const { pathname } = new URL(request.url());
  return pathname.startsWith("/api/") ? pathname : undefined;
}

/**
 * Starts watching page, and every document it loads from now on: call it before the page's first
 * navigation, so that nothing the app does as it starts goes unseen.
 */
export async function watchPage(page: Page): Promise<PageWatch> {
  const watch: PageWatch = {
    apiRequests: [],
    apiFailures: [],
    oldApiRequests: [],
    pageErrors: [],
    consoleErrors: [],
    consoleWarnings: [],
    cspViolations: [],
  };
  page.on("request", (request) => {
    const path = apiPath(request);
    if (path === undefined) {
      return;
    }
    watch.apiRequests.push(`${request.method()} ${path}`);
    if (!path.startsWith("/api/v0/")) {
      watch.oldApiRequests.push(`${request.method()} ${path}`);
    }
  });
  page.on("response", (response) => {
    const path = apiPath(response.request());
    if (path !== undefined && response.status() >= 400) {
      watch.apiFailures.push(`${response.status()} ${response.request().method()} ${path}`);
    }
  });
  page.on("requestfailed", (request) => {
    const path = apiPath(request);
    if (path !== undefined) {
      watch.apiFailures.push(`${request.failure()?.errorText} ${request.method()} ${path}`);
    }
  });
  page.on("pageerror", (error) => {
    watch.pageErrors.push(error.message);
  });
  page.on("console", (message) => {
    if (message.type() === "error") {
      watch.consoleErrors.push(message.text());
    } else if (message.type() === "warning") {
      watch.consoleWarnings.push(message.text());
    }
  });
  await page.exposeBinding("__nerveE2eCspViolation", (_source, violation: string) => {
    watch.cspViolations.push(violation);
  });
  await page.addInitScript(() => {
    document.addEventListener("securitypolicyviolation", (event) => {
      const report = (window as unknown as { __nerveE2eCspViolation: (violation: string) => void })
        .__nerveE2eCspViolation;
      report(`${event.effectiveDirective} ${event.blockedURI}`);
    });
  });
  return watch;
}
```

- [ ] **Step 2: 故事**

`e2e/stories/identity/a1-sign-up.spec.ts`（完整内容）：

```ts
import { expectRegistered } from "../../fixtures/assert/identity";
import { submitSignUp } from "../../fixtures/auth-pages";
import { emailFor, password, recordOf, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// A1, a new account (M2 design 2).

test("A1 (page): a visitor signs up and lands on the profile step of onboarding", async ({ page, db }, testInfo) => {
  const email = emailFor(testInfo, "Alice");
  // What the page shows or sends where a token could leak: the console, and the addresses it asks for.
  const shown: string[] = [];
  let accessToken = "";
  page.on("console", (message) => shown.push(message.text()));
  page.on("request", (request) => {
    shown.push(request.url());
    accessToken = request.headers().authorization?.replace(/^Bearer /, "") ?? accessToken;
  });

  await page.goto("/sign-up");
  expect(await submitSignUp(page, email, password)).toBe(201);

  await expect(page).toHaveURL("/onboarding");
  await expect(page.getByText("Create your profile.")).toBeVisible();
  // The session is the token manager's record, and nothing else: no cookie (M2 design 7.1).
  expect(await page.context().cookies()).toEqual([]);
  const record = await recordOf(page);
  expect(Object.keys(record ?? {}).toSorted()).toEqual(["login_id", "refresh_token"]);
  expect(record?.login_id).toMatch(/^[0-9a-f]{32}$/);
  // The tokens are nowhere else: not in sessionStorage, another localStorage key, an address or the
  // console. Their bodies are looked for, so that a copy under another prefix or in quotes counts too.
  expect(accessToken, "a request carried the access token").not.toBe("");
  const bodies = [record?.refresh_token.replace(/^nrv_rt_/, "") ?? "", accessToken.split(".")[2] ?? ""];
  expect(bodies.map((body) => body.length > 20)).toEqual([true, true]);
  const stored = await page.evaluate(() =>
    [...Object.entries(sessionStorage), ...Object.entries(localStorage).filter(([key]) => key !== "nerve.auth")].map(
      ([key, value]) => `${key}=${value}`
    )
  );
  const leaks = [...stored, page.url(), ...shown].filter((text) => bodies.some((body) => text.includes(body)));
  expect(leaks).toEqual([]);
  const userAgent = await page.evaluate(() => navigator.userAgent);
  await expectRegistered(db, { email, refreshToken: record?.refresh_token ?? "", userAgent, ip: "127.0.0.1" });
});

test("A1 (API): a caller signs up and gets a session", async ({ api, db }, testInfo) => {
  const email = emailFor(testInfo, "Alice");
  const userAgent = "nerve-e2e/A1";

  const tokens = await register(api, email, { "User-Agent": userAgent });

  expect(tokens.token_type).toBe("Bearer");
  expect(tokens.access_token_expires_in).toBe(15 * 60);
  await expectRegistered(db, { email, refreshToken: tokens.refresh_token, userAgent, ip: "127.0.0.1" });

  // The access token works at once.
  const me = await api.GET("/api/v0/me", { headers: { Authorization: `Bearer ${tokens.access_token}` } });
  expect(me.response.status).toBe(200);
  expect(me.data?.email).toBe(email.toLowerCase());
});
```

`e2e/stories/identity/a2-sign-up-refused.spec.ts`（完整内容）：

```ts
import type { Page } from "@playwright/test";

import { createApi, type Api } from "../../fixtures/api";
import { countIdentity, expectNothingAdded } from "../../fixtures/assert/identity";
import { fillSignUp, formAlert, submitSignUp } from "../../fixtures/auth-pages";
import { emailFor, password, register } from "../../fixtures/auth";
import { watchPage } from "../../fixtures/browser";
import { expect, test } from "../../fixtures/test";

// A2, sign-up refused (M2 design 2).

test("A2 (page): a refused sign-up says why in place, keeps the address and adds nothing", async ({
  page,
  api,
  db,
  nerve,
  nerveWith,
}, testInfo) => {
  const email = emailFor(testInfo);
  await register(api, email);
  const before = await countIdentity(db);
  const newEmail = emailFor(testInfo, "new");
  const emailInput = page.getByLabel("Email", { exact: true });
  const watch = await watchPage(page);

  // The address is taken: the message is above the form, the page and the address stay.
  await page.goto("/sign-up");
  expect(await submitSignUp(page, email.toUpperCase(), password)).toBe(409);
  await expect(formAlert(page)).toHaveText("An account with this email already exists.");
  await expect(page).toHaveURL("/sign-up");
  await expect(emailInput).toHaveValue(email.toUpperCase());

  // A weak password: the rules show under the field, and the page sends nothing.
  await fillSignUp(page, newEmail, "password");
  await expect(page.getByText("8–128 characters")).toBeVisible();
  await expect(page.getByText("Min 1 upper-case letter")).toBeVisible();
  const sent = watch.apiRequests.length;
  await page.getByRole("button", { name: "Create account" }).click();
  await expect(formAlert(page)).toHaveText("Try setting-up a strong password to proceed");
  expect(watch.apiRequests.slice(sent)).toEqual([]);

  // Common passwords, which only nerve knows: the message is under the field, none above the form.
  const expectTooCommon = async (common: string) => {
    expect(await submitSignUp(page, newEmail, common), common).toBe(422);
    await expect(page.getByText("This password is too common")).toBeVisible();
    await expect(formAlert(page)).toHaveCount(0);
    await expect(emailInput).toHaveValue(newEmail);
  };
  await expectTooCommon("Password1!");
  await expectTooCommon("Password1!~");

  // With sign-up off the header has no sign-up link, which it has with sign-up on; and nerve refuses
  // every address alike, a taken one too.
  const signUpLink = page.getByRole("link", { name: "Sign up" });
  await showSignIn(page, nerve.baseURL);
  await expect(signUpLink).toBeVisible();
  const closed = await nerveWith({ NERVE_AUTH__SIGNUP_ENABLED: "false" });
  await showSignIn(page, closed.baseURL);
  await expect(signUpLink).toHaveCount(0);
  await page.goto(`${closed.baseURL}/sign-up`);
  const expectClosed = async (address: string) => {
    expect(await submitSignUp(page, address, password), address).toBe(403);
    await expect(formAlert(page)).toHaveText("Sign-up is closed.");
  };
  await expectClosed(newEmail);
  await expectClosed(email);

  await expectNothingAdded(db, before);
});

/** Opens the sign-in page of the nerve at baseURL, and waits until it has the instance's settings. */
async function showSignIn(page: Page, baseURL: string): Promise<void> {
  await Promise.all([
    page.waitForResponse((res) => res.url() === `${baseURL}/api/v0/instance` && res.ok()),
    page.goto(`${baseURL}/`),
  ]);
  await expect(page.getByRole("button", { name: "Go to workspace" })).toBeVisible();
}

async function expectRefused(
  api: Api,
  body: { email: string; password: string },
  want: { status: number; code: string; fields?: { field: string; code: string }[] }
): Promise<void> {
  const { response, error } = await api.POST("/api/v0/auth/register", { body });
  const label = `${body.email} ${body.password}`;
  expect(response.status, label).toBe(want.status);
  expect(error?.code, label).toBe(want.code);
  if (want.fields) {
    expect(
      error?.errors?.map((e) => ({ field: e.field, code: e.code })),
      label
    ).toEqual(want.fields);
  }
}

test("A2 (API): a refused sign-up answers why and adds nothing", async ({ api, db, nerveWith }, testInfo) => {
  const email = emailFor(testInfo);
  await register(api, email);
  const before = await countIdentity(db);
  const newEmail = emailFor(testInfo, "new");
  const invalid = (pw: string, code: string) =>
    expectRefused(
      api,
      { email: newEmail, password: pw },
      { status: 422, code: "validation_failed", fields: [{ field: "password", code }] }
    );

  await Promise.all([
    // The address is taken, whatever its case.
    expectRefused(api, { email: email.toUpperCase(), password }, { status: 409, code: "identity.email_taken" }),
    // A weak password, and common ones.
    invalid("password", "weak_password"),
    invalid("Password1!", "common_password"),
    invalid("Password1!~", "common_password"),
  ]);

  // With sign-up off, every address gets the same answer, a taken one too.
  const closed = createApi((await nerveWith({ NERVE_AUTH__SIGNUP_ENABLED: "false" })).baseURL);
  await Promise.all(
    [newEmail, email].map((address) =>
      expectRefused(closed, { email: address, password }, { status: 403, code: "identity.signup_disabled" })
    )
  );

  await expectNothingAdded(db, before);
});
```

`e2e/stories/identity/a3-sign-in.spec.ts`（完整内容）：

```ts
import type { Browser, Page } from "@playwright/test";

import { countIdentity, expectNothingAdded, expectSignedIn } from "../../fixtures/assert/identity";
import { formAlert, signInPath, submitSignIn } from "../../fixtures/auth-pages";
import { bearer, emailFor, login, password, recordOf, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// A3, signing in (M2 design 2), with next_path (3.18).

/** A page in a browser context of its own: signed out, whatever the other pages did. */
async function freshPage(browser: Browser, baseURL: string): Promise<Page> {
  const context = await browser.newContext({ baseURL });
  return context.newPage();
}

test("A3 (page): signing in comes back to the page asked for, and only to a page of this site", async ({
  browser,
  api,
  db,
  nerve,
}, testInfo) => {
  const email = emailFor(testInfo, "Alice");
  const tokens = await register(api, email);
  // Onboarded, so that the settings open once signed in.
  const onboarded = await api.PATCH("/api/v0/me/profile", {
    body: { is_onboarded: true },
    headers: bearer(tokens.access_token),
  });
  expect(onboarded.response.status).toBe(200);

  // Signed out, a page behind the sign-in goes to the sign-in page; the page comes back after it, with
  // its query and fragment.
  const asked = "/settings/profile/general?tab=x#y";
  const page = await freshPage(browser, nerve.baseURL);
  await page.goto(asked);
  await expect(page).toHaveURL(signInPath(asked));
  expect(signInPath(asked)).toBe("/?next_path=%2Fsettings%2Fprofile%2Fgeneral%3Ftab%3Dx%23y");
  expect(await submitSignIn(page, email, password)).toBe(200);
  await expect(page).toHaveURL(asked);
  const userAgent = await page.evaluate(() => navigator.userAgent);
  await expectSignedIn(db, {
    email,
    refreshToken: (await recordOf(page))?.refresh_token ?? "",
    userAgent,
    ip: "127.0.0.1",
  });
  await page.context().close();

  // A next_path that could lead elsewhere is dropped: the account's default page instead.
  await Promise.all(
    ["//evil.example", "/\\evil.example", "javascript:alert(1)", "/\t/evil.example"].map(async (nextPath) => {
      const other = await freshPage(browser, nerve.baseURL);
      await other.goto(`/?next_path=${encodeURIComponent(nextPath)}`);
      expect(await submitSignIn(other, email, password), nextPath).toBe(200);
      await expect(other, nextPath).toHaveURL("/create-workspace");
      await other.context().close();
    })
  );
});

test("A3 (page): a wrong password and an unknown address get the same message, and no session", async ({
  page,
  api,
  db,
}, testInfo) => {
  const email = emailFor(testInfo, "Alice");
  await register(api, email);
  const before = await countIdentity(db);

  await page.goto("/");
  const expectRefused = async (address: string, pw: string) => {
    expect(await submitSignIn(page, address, pw), address).toBe(401);
    await expect(formAlert(page)).toHaveText("The email or the password is wrong.");
    await expect(page).toHaveURL("/");
    await expect(page.getByLabel("Email", { exact: true })).toHaveValue(address);
  };
  await expectRefused(email, "Wr0ng-password");
  await expectRefused(emailFor(testInfo, "nobody"), password);
  await expectNothingAdded(db, before);
});

test("A3 (API): a caller signs in; a wrong password and an unknown address answer alike", async ({
  api,
  db,
}, testInfo) => {
  const email = emailFor(testInfo, "Alice");
  await register(api, email);
  const userAgent = "nerve-e2e/A3";

  // The address in another case and with blanks around it still signs in.
  const tokens = await login(api, ` ${email.toUpperCase()} `, { "User-Agent": userAgent });

  expect(tokens.token_type).toBe("Bearer");
  await expectSignedIn(db, { email, refreshToken: tokens.refresh_token, userAgent, ip: "127.0.0.1" });
  const me = await api.GET("/api/v0/me", { headers: { Authorization: `Bearer ${tokens.access_token}` } });
  expect(me.response.status).toBe(200);

  const before = await countIdentity(db);
  const refusals = await Promise.all([
    api.POST("/api/v0/auth/login", { body: { email, password: "Wr0ng-password" } }),
    api.POST("/api/v0/auth/login", { body: { email: emailFor(testInfo, "nobody"), password } }),
  ]);
  for (const { response, error } of refusals) {
    expect(response.status).toBe(401);
    expect(error?.code).toBe("identity.invalid_credentials");
    expect(error?.detail).toBe("The e-mail address or the password is incorrect.");
  }

  // A body without a password breaks the contract: the platform's 400.
  const missing = await api.POST("/api/v0/auth/login", {
    // @ts-expect-error -- the request leaves out a required field on purpose
    body: { email },
  });
  expect(missing.response.status).toBe(400);
  expect(missing.error?.code).toBe("bad_request");
  expect(missing.error?.errors?.map((e) => ({ field: e.field, code: e.code }))).toEqual([
    { field: "password", code: "required" },
  ]);
  await expectNothingAdded(db, before);
});
```

`e2e/stories/identity/a10-onboarding-profile.spec.ts`（完整内容）：

```ts
import { accountOf } from "../../fixtures/assert/identity";
import { bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { watchPage } from "../../fixtures/browser";
import type { Database } from "../../fixtures/db";
import { expect, test } from "../../fixtures/test";

// A10, the profile step of onboarding (M2 design 2).

/** The onboarding steps of the account of email, a lowercased address. */
async function stepsOf(db: Database, email: string): Promise<unknown> {
  const { id } = await accountOf(db, email);
  const rows = await db.query<{ onboarding_step: unknown }>("SELECT onboarding_step FROM profiles WHERE user_id = $1", [
    id,
  ]);
  return rows[0]?.onboarding_step;
}

test("A10 (page): a new account's first visit of /onboarding goes well, and its profile step saves the name", async ({
  api,
  db,
  signedInPage,
}, testInfo) => {
  const email = emailFor(testInfo);
  const page = await signedInPage(await register(api, email));
  const watch = await watchPage(page);

  await page.goto("/onboarding");
  await expect(page.getByText("Create your profile.")).toBeVisible();
  await page.getByLabel("Name").fill("Ada");
  // The step's last request saves the step done, after the name.
  const [saved] = await Promise.all([
    page.waitForResponse(
      (response) =>
        response.request().method() === "PATCH" && new URL(response.url()).pathname === "/api/v0/me/profile",
      { timeout: 10_000 }
    ),
    page.getByRole("button", { name: "Continue" }).click(),
  ]);
  expect(saved.status()).toBe(200);

  // The next step shows; nerve has the name and the one step done.
  await expect(page.getByText("Create your workspace")).toBeVisible();
  expect(await stepsOf(db, email)).toEqual({
    profile_complete: true,
    workspace_create: false,
    workspace_invite: false,
    workspace_join: false,
  });
  expect((await accountOf(db, email)).first_name).toBe("Ada");
  // Nothing went wrong on the way (M2 design 3.1): no API call failed, none went to an older API, no
  // exception or rejection was left unhandled, the console has no error, the CSP blocked nothing.
  expect(watch.apiFailures).toEqual([]);
  expect(watch.oldApiRequests).toEqual([]);
  expect(watch.pageErrors).toEqual([]);
  expect(watch.consoleErrors).toEqual([]);
  expect(watch.cspViolations).toEqual([]);
});

test("A10 (API): the profile step sets the name and one step, which the others keep beside", async ({
  api,
  db,
}, testInfo) => {
  const email = emailFor(testInfo);
  const pat = await createPAT(api, (await register(api, email)).access_token);
  const { id } = await accountOf(db, email);
  const steps = async () =>
    (await db.query<{ onboarding_step: unknown }>("SELECT onboarding_step FROM profiles WHERE user_id = $1", [id]))[0]
      ?.onboarding_step;

  const named = await api.PATCH("/api/v0/me", { body: { first_name: "Ada" }, headers: bearer(pat.token) });
  expect(named.response.status).toBe(200);
  expect((await accountOf(db, email)).first_name).toBe("Ada");

  // Another step is done already, so keeping it differs from resetting it
  // to its default.
  const joined = await api.PATCH("/api/v0/me/profile", {
    body: { onboarding_step: { workspace_join: true } },
    headers: bearer(pat.token),
  });
  expect(joined.response.status).toBe(200);

  // One key: it is merged in, the other three keep their values (M2 design 3.14).
  const stepped = await api.PATCH("/api/v0/me/profile", {
    body: { onboarding_step: { profile_complete: true } },
    headers: bearer(pat.token),
  });
  expect(stepped.response.status).toBe(200);
  const merged = { profile_complete: true, workspace_create: false, workspace_invite: false, workspace_join: true };
  expect(await steps()).toEqual(merged);

  // An unknown key, here misspelt, breaks the contract: the platform's 400,
  // and nothing changes.
  const unknown = await api.PATCH("/api/v0/me/profile", {
    body: { onboarding_step: { profile_completed: true } },
    headers: bearer(pat.token),
  });
  expect(unknown.response.status).toBe(400);
  expect(unknown.error?.errors?.map((e) => ({ field: e.field, code: e.code }))).toEqual([
    { field: "onboarding_step.profile_completed", code: "not_allowed" },
  ]);
  expect(await steps()).toEqual(merged);
});
```

`e2e/stories/identity/a15-sign-in-limits.spec.ts`（完整内容）：

```ts
import { createApi } from "../../fixtures/api";
import { countIdentity, expectNothingAdded } from "../../fixtures/assert/identity";
import { formAlert, submitSignIn } from "../../fixtures/auth-pages";
import { emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// A15, login limits (M2 design 2, 3.10).

/** A nerve with low login limits: login_ip 3, login_ip_email 2, each regaining one unit a minute. */
const lowLoginLimits = {
  NERVE_RATELIMIT__LOGIN_IP__PER_MINUTE: "1",
  NERVE_RATELIMIT__LOGIN_IP__BURST: "3",
  NERVE_RATELIMIT__LOGIN_IP_EMAIL__PER_MINUTE: "1",
  NERVE_RATELIMIT__LOGIN_IP_EMAIL__BURST: "2",
};

test("A15 (page): the sign-in page says when there were too many attempts", async ({
  page,
  db,
  nerveWith,
}, testInfo) => {
  const limited = await nerveWith(lowLoginLimits);
  const email = emailFor(testInfo);
  await register(createApi(limited.baseURL), email);
  const before = await countIdentity(db);
  const attempt = async (address: string, want: number) => {
    expect(await submitSignIn(page, address, "Wr0ng-password"), address).toBe(want);
    await expect(formAlert(page), address).toHaveText(
      want === 429 ? "Too many attempts. Please try again later." : "The email or the password is wrong."
    );
  };

  await page.goto(`${limited.baseURL}/`);
  // One address fails up to login_ip_email's burst, then is refused; another address may still try, until
  // login_ip refuses every address.
  await attempt(email, 401);
  await attempt(email, 401);
  await attempt(email, 429);
  await attempt(emailFor(testInfo, "other"), 401);
  await attempt(emailFor(testInfo, "third"), 429);

  await expectNothingAdded(db, before);
});

test("A15 (API): logins are limited per client IP and address, then per client IP", async ({
  db,
  nerveWith,
}, testInfo) => {
  const limited = createApi((await nerveWith(lowLoginLimits)).baseURL);
  const email = emailFor(testInfo);
  await register(limited, email);
  const before = await countIdentity(db);
  const attempt = async (address: string, want: number) => {
    const { response, error } = await limited.POST("/api/v0/auth/login", {
      body: { email: address, password: "Wr0ng-password" },
    });
    expect(response.status, address).toBe(want);
    if (want === 429) {
      expect(error?.code).toBe("rate_limited");
      expect(response.headers.get("Retry-After")).toBe("60");
    }
  };

  // One address fails up to login_ip_email's burst, then is refused.
  await attempt(email, 401);
  await attempt(email, 401);
  await attempt(email, 429);
  // That refusal took nothing from login_ip: another address gets its last
  // unit, and then login_ip refuses every address.
  await attempt(emailFor(testInfo, "other"), 401);
  await attempt(emailFor(testInfo, "third"), 429);

  await expectNothingAdded(db, before);
});
```

`e2e/stories/smoke/s2-web-app.spec.ts`（完整内容）：

```ts
import type { Page, Response } from "@playwright/test";

import { signInPath } from "../../fixtures/auth-pages";
import { watchPage, type PageWatch } from "../../fixtures/browser";
import { expect, test } from "../../fixtures/test";

/** A page of the frontend's router, not a file: nerve answers it with index.html. */
const deepLink = "/acme/projects/0199f1c2-7a1b-7c3d-8e4f-5a6b7c8d9e0f/issues";

interface Visit {
  document: Response;
  watch: PageWatch;
  /** Static resources that loaded, as "<status> <url>". */
  loaded: string[];
  /** Static resources that failed: an HTTP error, or refused or aborted by the browser. */
  failed: string[];
  /** Requests to any origin other than nerve's; the frontend is served same-origin. */
  elsewhere: string[];
}

function isStatic(url: string): boolean {
  return !new URL(url).pathname.startsWith("/api/");
}

/** Opens path, signed out, and waits until the network is idle. */
async function open(page: Page, path: string): Promise<Visit> {
  const watch = await watchPage(page);
  const requested: string[] = [];
  const loaded: string[] = [];
  const failed: string[] = [];
  page.on("request", (req) => {
    requested.push(req.url());
  });
  page.on("response", (res) => {
    if (isStatic(res.url())) {
      (res.status() < 400 ? loaded : failed).push(`${res.status()} ${res.url()}`);
    }
  });
  page.on("requestfailed", (req) => {
    if (isStatic(req.url())) {
      failed.push(`${req.failure()?.errorText} ${req.url()}`);
    }
  });
  const document = await page.goto(path, { waitUntil: "networkidle" });
  if (!document) {
    throw new Error(`no document response for ${path}`);
  }
  const origin = new URL(document.url()).origin;
  const elsewhere = requested.filter((url) => new URL(url).origin !== origin);
  return { document, watch, loaded, failed, elsewhere };
}

/**
 * Signed out, the app asks nerve for the instance's settings only: without a refresh token it neither
 * refreshes nor asks for /me (M2 design 7.1), so no API call fails; and the page's Content-Security-Policy
 * blocks nothing of it (8.3).
 */
function expectQuietSignedOut(watch: PageWatch): void {
  expect(watch.apiRequests).toEqual(["GET /api/v0/instance"]);
  expect(watch.apiFailures).toEqual([]);
  expect(watch.cspViolations).toEqual([]);
  expect(watch.pageErrors).toEqual([]);
  expect(watch.consoleErrors).toEqual([]);
}

test("S2: a user opens the home page in a browser", async ({ page }) => {
  const { document, watch, loaded, failed, elsewhere } = await open(page, "/");

  expect(document.status()).toBe(200);
  expect(document.headers()["content-type"]).toBe("text/html; charset=utf-8");
  expect(document.headers()["content-security-policy"]).toMatch(/^default-src 'self'; script-src 'self' 'sha256-/);
  expect(loaded).toContainEqual(expect.stringMatching(/^200 .*\/assets\/[^/]+\.js$/));
  await expect(page.getByRole("button", { name: "Go to workspace" })).toBeVisible();
  expect(failed).toEqual([]);
  expect(elsewhere).toEqual([]);
  expectQuietSignedOut(watch);
});

test("S2: a user opens a deep link directly", async ({ page, request }) => {
  const { document, watch, failed, elsewhere } = await open(page, deepLink);

  expect(document.status()).toBe(200);
  expect(document.headers()["content-type"]).toBe("text/html; charset=utf-8");
  expect(await document.body()).toEqual(await (await request.get("/")).body());
  // Signed out, the page behind the sign-in goes to the sign-in page, which comes back to it.
  await expect(page).toHaveURL(signInPath(deepLink));
  expect(failed).toEqual([]);
  expect(elsewhere).toEqual([]);
  expectQuietSignedOut(watch);
});
```

- [ ] **Step 3: 检查**

Run: `make lint-web`、`make knip`、`make test-web`
Expected: 都通过（`@nerve/e2e` 的 oxlint 上限仍是 0）。

Run: `make e2e`
Expected: 28 个测试全部通过。

Run: `make build`，然后 `NERVE_VERSION=0.1.0-dev pnpm -C e2e exec playwright test stories/identity stories/smoke/s2-web-app.spec.ts --grep "page|S2" --repeat-each=4`
Expected: 全部通过。

- [ ] **Step 4: 提交**

```bash
git add e2e
```
```bash
git commit -m "test(M2/P4): the page versions of A1-A3, A10 and A15; S2 watches the signed-out page

signedInPage writes the token manager's record before a page's first
script, once per context; watchPage records a page's API calls and
failures, exceptions, console errors and CSP violations. S2 asserts the
signed-out page asks for the instance only, fails nothing and carries a
Content-Security-Policy; A1 finds the tokens nowhere but nerve.auth.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** `make e2e` 通过；重复 4 次也通过。

---

### Task 13: 端到端：A4–A6 的页面版本（A4 含没有 `navigator.locks` 的一遍，A6 含两种切换账户）

**Files:**
- Modify: `e2e/fixtures/auth.ts`、`e2e/fixtures/assert/identity.ts`
- Modify: `e2e/stories/identity/a4-refresh.spec.ts`、`a5-refresh-reuse.spec.ts`、`a6-sign-out.spec.ts`

**Interfaces:**
- Produces（spec 2.14，M2 设计 2、7.1、9.5）：
  - `auth.ts`：`newRecord(tokens)` 导出；`writeRecord(page, record)` 像令牌管理器一样写入记录：一次写入，持有 `nerve.auth.refresh` 锁（其他标签页收到 `storage` 事件）；
  - `assert/identity.ts`：`generationOf(refreshToken)`（刷新令牌里的代数）。
- 故事：
  - **A4**（两个测试：有 `navigator.locks`；用 `addInitScript` 删掉 `Navigator.prototype.locks`，走租约）：访问令牌 3 秒的 nerve，同一个上下文的两个标签页都在资料步骤；等到 nerve 拒绝两个标签页拿到的最后一个访问令牌；第一个续期请求被扣住最多 1 秒（`holdFirstRefresh`：没有锁时另一个标签页的续期会在这期间发出，用同一个刷新令牌）；两个标签页同时点"Continue"，都等到这一步最后一个请求的 200，都进入下一步、都没有去登录页；每次续期都是 200，发出的刷新令牌的代数依次是 0、1、2……各一次（同时的两次会发出同一个令牌，nerve 会当成被盗）；没有失败的接口请求；`expectRefreshed`（代数等于续期次数，`expires_at` 不变）。续期是否重叠不用浏览器的计时判断：被扣住的请求，Playwright 的计时从放行时算起；
  - **A5**：页面的刷新令牌先被"别人"在接口上用了一次；页面重新加载时续期被拒，页面到带 `next_path` 的登录页，记录被删除，会话 `reuse_detected`；
  - **A6**（三个测试）：在一个标签页通过账户菜单退出（M2 只有新手引导页头的账户菜单："Wrong e-mail address?" → "Switch account"），两个标签页都回到登录页，记录被删除，会话 `logout`，这个会话的访问令牌下一次请求就是 401；**切换 ①**：标签页乙在站内任一页面用 `writeRecord` 换上 Y 的记录（新的 `login_id`），不退出：标签页甲显示 Y，此后保存的名字写进 Y，X 的名字不变，X 的会话没有被撤销；**切换 ②**：标签页乙先退出（甲随之到登录页），再以 Y 登录：两个标签页都以 Y 回到 `/onboarding`。

**Tests:** 页面版本 6 个（A4 2 个，A5 1 个，A6 3 个）。

- [ ] **Step 1: fixture**

`e2e/fixtures/auth.ts`（对 Task 12 版本的差异）：

```diff
--- a/e2e/fixtures/auth.ts
+++ b/e2e/fixtures/auth.ts
@@ -18,7 +18,7 @@
 }
 
 /** The record a sign-in with tokens writes: a new login_id of 16 random bytes in hexadecimal. */
-function newRecord(tokens: AuthTokens): AuthRecord {
+export function newRecord(tokens: AuthTokens): AuthRecord {
   return { refresh_token: tokens.refresh_token, login_id: randomBytes(16).toString("hex") };
 }
 
@@ -48,6 +48,22 @@
   return text === null ? null : (JSON.parse(text) as AuthRecord);
 }
 
+/**
+ * Writes record into the localStorage of page as the token manager does (M2 design 7.1): in one piece,
+ * holding the refresh lock, so that no refresh of another tab writes in between. The other tabs get the
+ * storage event.
+ */
+export async function writeRecord(page: Page, record: AuthRecord): Promise<void> {
+  await page.evaluate(
+    async ({ key, text }) => {
+      await navigator.locks.request("nerve.auth.refresh", () => {
+        localStorage.setItem(key, text);
+      });
+    },
+    { key: authKey, text: JSON.stringify(record) }
+  );
+}
+
 /** A password that meets the rules and is not common. */
 export const password = "Tr0ub4dor&3";
 
```

`e2e/fixtures/assert/identity.ts`（对 `f27434c` 的差异）：

```diff
--- a/e2e/fixtures/assert/identity.ts
+++ b/e2e/fixtures/assert/identity.ts
@@ -29,6 +29,11 @@
   };
 }
 
+/** The generation a refresh token carries: 0 at the sign-in, one more at each refresh. */
+export function generationOf(refreshToken: string): number {
+  return parseRefreshToken(refreshToken).generation;
+}
+
 function secretHash(refreshToken: string): Buffer {
   return createHash("sha256").update(parseRefreshToken(refreshToken).secret).digest();
 }
```

- [ ] **Step 2: 故事**

`e2e/stories/identity/a4-refresh.spec.ts`（完整内容）：

```ts
import type { BrowserContext, Page, Request, Response } from "@playwright/test";

import { createApi } from "../../fixtures/api";
import { expectRefreshed, generationOf } from "../../fixtures/assert/identity";
import { bearer, emailFor, login, recordOf, refresh, register } from "../../fixtures/auth";
import { watchPage } from "../../fixtures/browser";
import { expect, test } from "../../fixtures/test";

// A4, refreshing (M2 design 2, 7.1). Reusing an old token is A5.

/** A refresh a tab sent: the generation of the refresh token it sent, and nerve's answer. */
interface Refresh {
  generation: number;
  status: number;
}

function isRefresh(request: Request): boolean {
  return new URL(request.url()).pathname === "/api/v0/auth/refresh";
}

/** What the tabs of a context send: every refresh, and the access token of the last API request. */
class TabRecorder {
  readonly #refreshes: Refresh[] = [];
  readonly #pending: Promise<void>[] = [];
  lastAccessToken: string | undefined;

  record(tab: Page): void {
    tab.on("request", (request) => {
      const authorization = request.headers().authorization;
      if (authorization?.startsWith("Bearer ")) {
        this.lastAccessToken = authorization.slice("Bearer ".length);
      }
    });
    tab.on("requestfinished", (request) => {
      if (isRefresh(request)) {
        this.#pending.push(this.#add(request));
      }
    });
    tab.on("requestfailed", (request) => {
      if (isRefresh(request)) {
        this.#refreshes.push({ generation: sentGeneration(request), status: 0 });
      }
    });
  }

  /** The refreshes, once every answer is in. */
  async refreshes(): Promise<Refresh[]> {
    await Promise.all(this.#pending);
    return this.#refreshes;
  }

  async #add(request: Request): Promise<void> {
    const response = await request.response();
    this.#refreshes.push({ generation: sentGeneration(request), status: response?.status() ?? 0 });
  }
}

/**
 * The answer to the last request the profile step sends: the step done, saved after the name. It needs a
 * token, so every refresh of the tab comes before it.
 */
function stepSaved(tab: Page): Promise<Response> {
  return tab.waitForResponse(
    (response) => response.request().method() === "PATCH" && new URL(response.url()).pathname === "/api/v0/me/profile",
    { timeout: 10_000 }
  );
}

function sentGeneration(request: Request): number {
  return generationOf((request.postDataJSON() as { refresh_token: string }).refresh_token);
}

/**
 * Holds the first refresh the tabs of context send from now on, until another refresh goes out or holdMs
 * pass. With the refresh lock no other refresh can go out meanwhile, and the hold ends at holdMs; without
 * it, the other tab's refresh goes out while this one is held, with the same refresh token (M2 design 7.1).
 */
async function holdFirstRefresh(context: BrowserContext, holdMs: number): Promise<void> {
  let release: (() => void) | undefined;
  await context.route("**/api/v0/auth/refresh", async (route) => {
    if (release === undefined) {
      await new Promise<void>((resolve) => {
        release = resolve;
        setTimeout(resolve, holdMs);
      });
    } else {
      release();
    }
    await route.continue();
  });
}

for (const locks of [true, false]) {
  const how = locks ? "navigator.locks" : "the localStorage lease, without navigator.locks";
  test(`A4 (page): two tabs whose access tokens expired both act at once, refreshing one at a time (${how})`, async ({
    context,
    db,
    nerveWith,
    signedInPage,
  }, testInfo) => {
    // Access tokens of 3 s: every request the tabs send refreshes first (M2 design 7.1: 30 s before the end).
    const shortLived = await nerveWith({ NERVE_AUTH__ACCESS_TOKEN_TTL: "3s" });
    const api = createApi(shortLived.baseURL);
    const signedUp = await register(api, emailFor(testInfo));
    if (!locks) {
      await context.addInitScript(() => {
        Reflect.deleteProperty(Navigator.prototype, "locks");
      });
    }
    const recorder = new TabRecorder();
    const tabA = await signedInPage(signedUp, shortLived.baseURL);
    const tabB = await context.newPage();
    const watches = [await watchPage(tabA), await watchPage(tabB)];
    // One tab after the other, so that the second finds the record the first refreshed.
    const openProfileStep = async (tab: Page) => {
      recorder.record(tab);
      await tab.goto(`${shortLived.baseURL}/onboarding`);
      await expect(tab.getByText("Create your profile.")).toBeVisible();
      expect(await tab.evaluate(() => "locks" in navigator)).toBe(locks);
      await tab.getByLabel("Name").fill("Ada");
    };
    await openProfileStep(tabA);
    await openProfileStep(tabB);

    // Wait until nerve refuses the last access token either tab got: every token in the tabs has expired.
    const lastToken = recorder.lastAccessToken;
    expect(lastToken, "the tabs sent an access token").toBeDefined();
    await expect
      .poll(async () => (await api.GET("/api/v0/me", { headers: bearer(lastToken ?? "") })).response.status, {
        timeout: 10_000,
      })
      .toBe(401);

    // Both tabs save the profile step at once: both go on to the next step, neither to the sign-in page.
    // The first refresh is held a while, so that the other tab's would overlap it if nothing kept them
    // apart.
    await holdFirstRefresh(context, 1_000);
    const [saved] = await Promise.all([
      Promise.all([tabA, tabB].map((tab) => stepSaved(tab))),
      Promise.all([tabA, tabB].map((tab) => tab.getByRole("button", { name: "Continue" }).click())),
    ]);
    expect(saved.map((response) => response.status())).toEqual([200, 200]);
    await Promise.all(
      [tabA, tabB].map(async (tab) => {
        await expect(tab.getByText("Create your workspace")).toBeVisible();
        await expect(tab).toHaveURL(`${shortLived.baseURL}/onboarding`);
      })
    );

    // Every refresh succeeded, and no two of them overlapped: each sent the token the one before it got,
    // so the generations sent are 0, 1, 2 … once each. Two at the same time would send the same token, and
    // nerve would take the second for a stolen copy (A5). The browser's timings cannot show it: a held
    // request's timing counts from when it was let go.
    const refreshes = await recorder.refreshes();
    expect(refreshes.length).toBeGreaterThanOrEqual(4);
    expect(refreshes.map((r) => r.status)).toEqual(refreshes.map(() => 200));
    expect(refreshes.map((r) => r.generation).toSorted((a, b) => a - b)).toEqual(refreshes.map((_, i) => i));
    expect(watches.flatMap((watch) => watch.apiFailures)).toEqual([]);
    const record = await recordOf(tabA);
    await expectRefreshed(db, record?.refresh_token ?? "", refreshes.length, signedUp.refresh_token_expires_at);
  });
}

test("A4 (API): each refresh uses the last token; the generation counts up and the session end stays", async ({
  api,
  db,
}, testInfo) => {
  const email = emailFor(testInfo);
  await register(api, email);
  const first = await login(api, email);

  const second = await refresh(api, first.refresh_token);
  const third = await refresh(api, second.refresh_token);
  const fourth = await refresh(api, third.refresh_token);

  // The access token holds only sub, sid and exp in seconds (M2 design 3.4):
  // within one second it can come out the same, so only the refresh token
  // must differ.
  for (const [before, after] of [
    [first, second],
    [second, third],
    [third, fourth],
  ] as const) {
    expect(after.refresh_token).not.toBe(before.refresh_token);
    expect(after.refresh_token_expires_at).toBe(first.refresh_token_expires_at);
  }
  await expectRefreshed(db, fourth.refresh_token, 3, first.refresh_token_expires_at);
  const me = await api.GET("/api/v0/me", { headers: { Authorization: `Bearer ${fourth.access_token}` } });
  expect(me.response.status).toBe(200);
});
```

`e2e/stories/identity/a5-refresh-reuse.spec.ts`（完整内容）：

```ts
import { randomBytes } from "node:crypto";

import { expectRevoked, sessionOf } from "../../fixtures/assert/identity";
import { signInPath } from "../../fixtures/auth-pages";
import { emailFor, recordOf, refresh, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// A5, a reused refresh token (M2 design 2).

test("A5 (page): when a copy of the page's refresh token was used, the page's next refresh ends the session", async ({
  api,
  db,
  signedInPage,
}, testInfo) => {
  const page = await signedInPage(await register(api, emailFor(testInfo)));
  await page.goto("/onboarding");
  await expect(page.getByText("Create your profile.")).toBeVisible();

  // Someone with a copy of the page's refresh token uses it first: nerve rotates the session to them.
  const held = (await recordOf(page))?.refresh_token ?? "";
  await refresh(api, held);

  // The page's next refresh, as it loads again, presents the retired token: nerve ends the session, and
  // the page goes to the sign-in page, which comes back here.
  await page.reload();
  await expect(page).toHaveURL(signInPath("/onboarding"));
  await expect(page.getByRole("button", { name: "Go to workspace" })).toBeVisible();
  expect(await recordOf(page)).toBeNull();
  await expectRevoked(db, held, "reuse_detected");
});

/** token with its secret and tag replaced by random bytes: the session and the generation are real, the rest is not. */
function forgedFrom(token: string): string {
  const prefix = "nrv_rt_";
  const raw = Buffer.from(token.slice(prefix.length), "base64url");
  randomBytes(48).copy(raw, 20);
  return prefix + raw.toString("base64url");
}

test("A5 (API): a retired refresh token revokes its session; a forged older generation does not", async ({
  api,
  db,
}, testInfo) => {
  const first = await register(api, emailFor(testInfo));
  const second = await refresh(api, first.refresh_token);
  const refused = async (token: string) => {
    const { response, error } = await api.POST("/api/v0/auth/refresh", { body: { refresh_token: token } });
    expect(response.status).toBe(401);
    expect(error?.code).toBe("identity.refresh_token_invalid");
  };

  // A forged older generation proves nothing: 401, the session goes on.
  const before = await sessionOf(db, second.refresh_token);
  await refused(forgedFrom(first.refresh_token));
  expect(await sessionOf(db, second.refresh_token)).toEqual(before);

  // The real retired token comes back: someone else holds a copy.
  await refused(first.refresh_token);
  await expectRevoked(db, first.refresh_token, "reuse_detected");

  // Every token of the session fails from now on.
  await refused(second.refresh_token);
  const me = await api.GET("/api/v0/me", { headers: { Authorization: `Bearer ${second.access_token}` } });
  expect(me.response.status).toBe(401);
});
```

`e2e/stories/identity/a6-sign-out.spec.ts`（完整内容）：

```ts
import type { Page } from "@playwright/test";

import { accountOf, expectRevoked, sessionOf } from "../../fixtures/assert/identity";
import { signInPath, submitSignIn } from "../../fixtures/auth-pages";
import {
  bearer,
  emailFor,
  login,
  newRecord,
  password,
  recordOf,
  refresh,
  register,
  writeRecord,
} from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// A6, signing out and switching accounts (M2 design 2, 7.1).

/** Opens /onboarding in tab, which shows the account of email in its header. */
async function openOnboarding(tab: Page, email: string): Promise<void> {
  await tab.goto("/onboarding");
  await expect(tab.getByText("Create your profile.")).toBeVisible();
  await expect(accountMenu(tab, email)).toBeVisible();
}

/** The account menu of the onboarding header, which shows the display name of the account signed in. */
function accountMenu(tab: Page, email: string) {
  // A new account's display name is its address before the "@" (M2 design 2, A1).
  return tab.getByRole("button", { name: email.toLowerCase().split("@")[0] });
}

/** Signs out through the account menu: "Wrong e-mail address?", then "Switch account" (M2 has no workspace menu). */
async function signOutThroughMenu(tab: Page, email: string): Promise<void> {
  await accountMenu(tab, email).click();
  await tab.getByRole("menuitem", { name: "Wrong e-mail address?" }).click();
  await tab.getByRole("button", { name: "Switch account" }).click();
}

test("A6 (page): signing out in one tab signs every tab out", async ({ api, db, context, signedInPage }, testInfo) => {
  const email = emailFor(testInfo);
  const tabA = await signedInPage(await register(api, email));
  let accessToken = "";
  tabA.on("request", (request) => {
    accessToken = request.headers().authorization?.replace(/^Bearer /, "") ?? accessToken;
  });
  await openOnboarding(tabA, email);
  const tabB = await context.newPage();
  await openOnboarding(tabB, email);
  const held = (await recordOf(tabA))?.refresh_token ?? "";

  await signOutThroughMenu(tabA, email);

  // Both tabs are on the sign-in page, which comes back to where they were; the record is gone.
  await Promise.all(
    [tabA, tabB].map(async (tab) => {
      await expect(tab).toHaveURL(signInPath("/onboarding"));
      await expect(tab.getByRole("button", { name: "Go to workspace" })).toBeVisible();
    })
  );
  expect(await recordOf(tabB)).toBeNull();
  await expectRevoked(db, held, "logout");
  // The session's access token fails on the next request.
  const me = await api.GET("/api/v0/me", { headers: bearer(accessToken) });
  expect(me.response.status).toBe(401);
});

test("A6 (page): another tab signs another account in without signing out: every tab goes on as that account", async ({
  api,
  db,
  context,
  signedInPage,
}, testInfo) => {
  const x = emailFor(testInfo, "x");
  const y = emailFor(testInfo, "y");
  const tabA = await signedInPage(await register(api, x));
  await register(api, y);
  await openOnboarding(tabA, x);
  const xHeld = (await recordOf(tabA))?.refresh_token ?? "";

  // Tab B, any page of the site, keeps a sign-in of Y as the token manager does: a new record, written
  // under the refresh lock, with a new login_id.
  const tabB = await context.newPage();
  await tabB.goto("/site.webmanifest.json");
  await writeRecord(tabB, newRecord(await login(api, y)));

  // Tab A follows: it shows Y, and what it writes from now on is Y's.
  await expect(accountMenu(tabA, y)).toBeVisible();
  await expect(accountMenu(tabA, x)).toHaveCount(0);
  await tabA.getByLabel("Name").fill("Yvonne");
  await tabA.getByRole("button", { name: "Continue" }).click();
  await expect(tabA.getByText("Create your workspace")).toBeVisible();
  await expect.poll(async () => (await accountOf(db, y.toLowerCase())).first_name).toBe("Yvonne");
  expect((await accountOf(db, x.toLowerCase())).first_name).toBe("");
  // X's session is left as it was: nobody signed it out.
  expect((await sessionOf(db, xHeld)).revoked_at).toBeNull();
});

test("A6 (page): another tab signs out, then signs another account in: every tab comes in as that account", async ({
  api,
  db,
  context,
  signedInPage,
}, testInfo) => {
  const x = emailFor(testInfo, "x");
  const y = emailFor(testInfo, "y");
  const tabA = await signedInPage(await register(api, x));
  await register(api, y);
  await openOnboarding(tabA, x);
  const tabB = await context.newPage();
  await openOnboarding(tabB, x);
  const xHeld = (await recordOf(tabB))?.refresh_token ?? "";

  await signOutThroughMenu(tabB, x);
  await expect(tabA).toHaveURL(signInPath("/onboarding"));
  await expect(tabB).toHaveURL(signInPath("/onboarding"));
  await expectRevoked(db, xHeld, "logout");

  // Tab B signs Y in; tab A sees the record appear and comes in as Y too, back to where it was.
  expect(await submitSignIn(tabB, y, password)).toBe(200);
  await Promise.all(
    [tabB, tabA].map(async (tab) => {
      await expect(tab).toHaveURL("/onboarding");
      await expect(accountMenu(tab, y)).toBeVisible();
    })
  );
});

test("A6 (API): logout ends the session; the previous generation's logout changes nothing", async ({
  api,
  db,
}, testInfo) => {
  const first = await register(api, emailFor(testInfo));
  const second = await refresh(api, first.refresh_token);
  const logout = async (token: string) => {
    const { response } = await api.POST("/api/v0/auth/logout", { body: { refresh_token: token } });
    expect(response.status).toBe(204);
  };

  // The previous generation: 204, and the session goes on (M2 design 3.5).
  const before = await sessionOf(db, second.refresh_token);
  await logout(first.refresh_token);
  expect(await sessionOf(db, second.refresh_token)).toEqual(before);

  // The current one ends the session; its access token fails on the next request.
  await logout(second.refresh_token);
  await expectRevoked(db, second.refresh_token, "logout");
  const me = await api.GET("/api/v0/me", { headers: { Authorization: `Bearer ${second.access_token}` } });
  expect(me.response.status).toBe(401);

  // Once more: the same answer, nothing changes.
  const ended = await sessionOf(db, second.refresh_token);
  await logout(second.refresh_token);
  expect(await sessionOf(db, second.refresh_token)).toEqual(ended);
});
```

- [ ] **Step 3: 检查**

Run: `make lint-web`、`make knip`、`make test-web`
Expected: 都通过。

Run: `make e2e`
Expected: 34 个测试全部通过。

Run: `make build`，然后 `NERVE_VERSION=0.1.0-dev pnpm -C e2e exec playwright test stories/identity stories/smoke/s2-web-app.spec.ts --grep "page|S2" --repeat-each=4`
Expected: 全部通过。

- [ ] **Step 4: 提交**

```bash
git add e2e
```
```bash
git commit -m "test(M2/P4): the page versions of A4-A6, with and without navigator.locks

A4 holds the first refresh a moment so that an unlocked second one would
overlap it, and proves one refresh at a time by the generations sent. A6
signs out through the account menu and switches accounts both ways: a
new record written under the lock in another tab, and sign-out then
sign-in.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** `make e2e` 通过；重复 4 次也通过。

---

### Task 14: 上级文档、README 与交接

**Files:**
- Modify: `docs/v0/v0-design.md`、`docs/v0/frontend-changes.md`、`README.md`
- Modify: `docs/v0/M2-auth/handoffs/M0-P5-frontend-api-notes.md`、`M0-P6-e2e-notes.md`、`M1-P2-trim-content.md`、`M1-P3-trim-platform.md`、`M1-P4-router-native.md`

**Interfaces:**
- 3.20 中 P4 的各行（spec 2.15、第 3 节）：
  - 总体设计 4.3：记录带 `login_id`；每次写入都在同一把锁或租约下，续期写回之前核对 `login_id`；没有 `navigator.locks` 时用租约；只有续期 401 或重发仍 401 才结束会话；"会话暂不可用"；CSP 只加在页面上；
  - 前端改动清单：3.2 的"登录、注册、退出、修改密码的提交方式"和"用户和认证相关的 store"两行已完成（M2/P4），3.1 的 M2 一行改为进行中（设计的 3.20 把两节的编号写反了，spec 第 3 节）。
- 8.7 中 P4 的 README 内容：HTTP 部署时多标签页靠租约，公网部署用 HTTPS；另外写上页面的登录状态、M2 能看到的页面、CSP、e2e 的 `signedInPage` 和 `watchPage`，删掉 M0 时"页面报错是预期的"一条和相对地址里的 `/auth/…`。
- 交接：五份都追加"处理结果（M2/P4）"；M0-P5、M1-P2、M1-P4 全部处理完，`status` 改为 `done`；M0-P6（fixture 写法的延伸，M4、M5、M8）和 M1-P3（`packages/services` 的 API 令牌旧地址，M2/P5）仍为 `open`（spec 第 7 节）。

- [ ] **Step 1: 上级文档**

`docs/v0/v0-design.md`（对 `f27434c` 的差异）：

```diff
--- a/docs/v0/v0-design.md
+++ b/docs/v0/v0-design.md
@@ -235,10 +235,10 @@
 - **忘记密码**：没有邮件服务，由服务器管理员通过命令行重置：`nerve users reset-password --email <email>`，它同时结束该账户的全部会话、撤销全部 PAT。
 
 ### 4.3 浏览器端
-- **令牌存放**：访问令牌存在内存；刷新令牌存在 localStorage。
-- **防范 XSS**：HTML 在服务端清洗；配置严格的内容安全策略（CSP）；刷新令牌每次使用后换新，并做重复使用检测。以后如需加强，只需把刷新令牌移进一个仅限续期接口使用的 HttpOnly Cookie，其他接口不受影响。
-- **多标签页续期**：用 `navigator.locks` 保证同一时间只有一个标签页在续期。
-- **401 处理**：先续期，再重试一次；续期失败就跳转到登录页。
+- **令牌存放**：访问令牌存在内存；刷新令牌存在 localStorage 的 `nerve.auth` 记录中，记录还带 `login_id`：每次登录、注册新生成，续期不变。其他标签页从 `login_id` 看出别处登录了另一个会话（可能是另一个账户），丢掉内存中的令牌，按新会话重新读取（M2 设计 7.1）。
+- **防范 XSS**：HTML 在服务端清洗；配置严格的内容安全策略（CSP，只加在页面上，内联脚本按哈希放行，M2 设计 8.3）；刷新令牌每次使用后换新，并做重复使用检测。以后如需加强，只需把刷新令牌移进一个仅限续期接口使用的 HttpOnly Cookie，其他接口不受影响。
+- **多标签页续期**：用 `navigator.locks` 保证同一时间只有一个标签页在续期；没有它时（用 HTTP 部署，浏览器不在安全上下文中）用 localStorage 的租约。登录、注册、续期、退出写 `nerve.auth` 都在这同一把锁或租约下；续期在锁内读出记录，写回之前再核对一次 `login_id`，变了就丢弃这次的结果（M2 设计 7.1）。
+- **401 处理**：先续期，再重发一次。只有续期得到 401、或者重发后仍是 401 才结束会话，跳转到登录页。续期超时（8 秒）、得到 429 或 5xx 时会话保留，页面显示"会话暂不可用"，按 `Retry-After` 或退避自动重试（M2 设计 7.1）。
 - **`<img src>` 没法带令牌**：
   - 编辑器图片：通过接口换取短期签名地址（Plane 编辑器本来就是异步获取图片地址的）。
   - 头像、图标、封面、附件：接口直接返回带签名的地址。
```

`docs/v0/frontend-changes.md`（对 `f27434c` 的差异）：

```diff
--- a/docs/v0/frontend-changes.md
+++ b/docs/v0/frontend-changes.md
@@ -198,7 +198,7 @@
 
 | 领域 | 所属 M | 状态 |
 |---|---|---|
-| 认证、用户、实例配置、PAT；令牌管理器 | M2 | 计划中 |
+| 认证、用户、实例配置、PAT；令牌管理器 | M2 | 进行中 |
 | 工作区、成员、邀请、项目、项目成员、项目归档、状态、标签、显示设置 | M3 | 计划中 |
 | 工作项、列表（分页和分组的新结构）、子任务、关联、链接、评论、表情回应、操作动态、搜索、历史版本、草稿、工作项归档 | M4 | 计划中 |
 | 文件、附件、编辑器图片（上传改为 `{method, url, headers}` 形式的 PUT） | M5 | 计划中 |
@@ -212,8 +212,8 @@
 | 位置 | 改动 | 原因 | 状态 | 完成于 |
 |---|---|---|---|---|
 | `core/store/issue/helpers/base-issues.store.ts` 等列表相关 store | 使用新接口的分页结构（不透明游标 `next_cursor`）和分组结构（`groups` 数组），不再按"已加载条数 ÷ 每页条数"拼页码游标 | 新接口的分页和分组设计 | 计划中 | |
-| 用户和认证相关的 store | 登录、退出、续期改走令牌管理器 | 认证改为 Bearer 令牌 | 计划中 | |
-| 登录、注册、退出、修改密码的提交方式 | 删除 CSRF 令牌和 Django 会话的表单提交，改走令牌管理器 | 认证改为 Bearer 令牌；CSRF 是传输方式的一部分，和它的替代品一起删除（[M1 设计](M1-frontend-trim/M1-design.md) 3.6） | 计划中 | |
+| 用户和认证相关的 store | 登录、退出、续期改走令牌管理器 | 认证改为 Bearer 令牌 | 已完成 | M2/P4 |
+| 登录、注册、退出、修改密码的提交方式 | 删除 CSRF 令牌和 Django 会话的表单提交，改走令牌管理器 | 认证改为 Bearer 令牌；CSRF 是传输方式的一部分，和它的替代品一起删除（[M1 设计](M1-frontend-trim/M1-design.md) 3.6） | 已完成 | M2/P4 |
 | 所有处理接口错误的地方 | 统一按 RFC 9457 的 problem+json 读取 `code`、`title`、`errors` | 错误格式统一 | 计划中 | |
 | 文件上传相关的 store 和调用方 | 预签名 POST 改为 `{method, url, headers}` 形式的 PUT | 文件存储改为 PUT 上传 | 计划中 | |
 | 迭代和模块的归属 | 通过工作项的 `cycle_id`、`module_ids` 字段修改，不再调用单独的接口 | 接口设计 | 计划中 | |
```

- [ ] **Step 2: README**

`README.md`（对 `f27434c` 的差异）：

````diff
--- a/README.md
+++ b/README.md
@@ -61,10 +61,11 @@
 
 - **开发**：`make dev-db`、`make run` 启动后端，再在另一个终端执行 `make web-dev`，打开 http://127.0.0.1:3000 。Vite 把 `/api` 转发给 `127.0.0.1:8080`。`make web-dev` 同时监视 `web/packages/*`：改了某个包的代码，这个包重新构建，页面随之更新。第一次启动时 Vite 要预构建依赖，页面如果报"Outdated Optimize Dep"，刷新一次即可。
 - **单元测试**：`make test-web` 运行各包 `test` 脚本中的 vitest（经 turbo，持续集成的 `web` 任务也执行它）。只给以后仍然有效的稳定逻辑写小测试，测试文件 `*.test.ts` 放在被测代码旁边；包里还没有 `test` 脚本时，加上 `"test": "vitest run"` 和开发依赖 `vitest`（`catalog:`）。`make test` 只运行 Go 测试。
-- **没有前端环境变量**：前端与接口同源部署，接口一律用相对路径（`/api/…`、`/auth/…`），代码不读取 `process.env` 或 `import.meta.env.VITE_*`，关键词守卫看住这一点。`import.meta.env.DEV`、`PROD` 是 Vite 按构建模式给出的常量，不是环境变量。
+- **没有前端环境变量**：前端与接口同源部署，接口一律用相对路径（`/api/…`），代码不读取 `process.env` 或 `import.meta.env.VITE_*`，关键词守卫看住这一点。`import.meta.env.DEV`、`PROD` 是 Vite 按构建模式给出的常量，不是环境变量。
 - **构建**：`make build` 构建前端，复制到 `server/internal/platform/webui/dist/`，编译出内嵌前端的 `bin/nerve`。运行时要在 `server/` 目录下，`config.local.yaml` 才能生效：`cd server && NERVE_ENV=dev ../bin/nerve serve`，之后打开 http://127.0.0.1:8080 。`dist/` 中只提交了 `.gitkeep`，没有构建过前端时页面上只有一句提示。
 - **清掉旧的构建产物**：`make build` 之后，`make run` 和 `go test` 都会继续内嵌这份构建。要去掉它：`find server/internal/platform/webui/dist -mindepth 1 ! -name .gitkeep -delete`（与 Makefile 里 `make build` 自己的清理命令相同）。
-- **M0 中看到的页面**：前端还在调用 Plane 的接口（例如 `/api/instances/`），Nerve 返回 404，页面显示"Looks like Nerve didn't start up correctly!"。这是预期的，前端从 M2 起对接 Nerve 的接口。
+- **登录状态**：页面用令牌管理器（`web/apps/web/core/lib/auth/`）注册、登录、续期和退出，没有 Cookie。访问令牌只在内存里；刷新令牌和 `login_id` 存在 localStorage 的 `nerve.auth` 中，同一个浏览器的标签页共用这一个会话，一次只有一个标签页在续期（`navigator.locks`，没有它时用 localStorage 的租约，见"部署"一节的"HTTPS"一条）。只有续期得到 401 才结束会话；续期超时、得到 429 或 5xx 时，页面显示"会话暂不可用"，按退避自动重试，也可以手动重试。
+- **M2 中看到的页面**：登录、注册、新手引导（只剩资料一步）、登录后的落点 `/create-workspace`、个人设置的 general 和 security。工作区相关的页面和操作还在调用 Plane 的接口，Nerve 返回 404，从 M3 起对接；个人设置的 preferences（时区列表）和 api-tokens 在 M2/P5 对接。
 - **lint 警告数等于上限**：每个包的 `check:lint` 脚本是 `node <到仓库根目录的相对路径>/tools/lint-cap.mjs <上限>`，它运行 oxlint，要求警告数正好等于上限，有任何错误都失败。警告多了，`make lint-web` 失败并列出这个包的全部警告：修掉新增的那几条，上限只能调低。警告少了（修掉了警告，或者删掉了带警告的代码），同样失败，并给出应调低到的数值：在同一个提交里把上限改成这个数。`make lint-web` 只打印失败任务的输出；要看某个包的全部警告，执行 `pnpm --filter <包名> exec oxlint .`。
 - **关键词守卫**：`make lint-web` 的第一步是 `node tools/keywords.mjs`，规则在 `tools/keywords.json`（M1 设计 7.4）。它检查 git 列出的文件（包括还没 `git add` 的新文件），命中规则、又没有登记例外就失败，并列出规则、文件、行号和命中的原文；规则或文件读取有问题时以 2 退出。删掉一个功能时，在同一个提交里加上它的规则（每条规则带理由和命中、不命中的样本）。确实要保留的命中登记为例外：规则、文件、命中的原文、理由和到期的 M 或 Phase，一条例外只覆盖一处；例外不再命中任何内容时守卫会提醒删掉它。`tools/` 下的脚本本身也由 `make lint-web` 检查：oxlint 不允许警告（根目录 `package.json` 的 `check:lint`）；oxfmt 检查它们和根目录的工具链配置的格式，文件列表只写在根目录 `package.json` 的 `fix:format` 里，`check:format` 就是带 `--check` 运行它。
 - **修格式**：`pnpm exec turbo run fix:format` 用 oxfmt 就地格式化所有包，以及根目录 `fix:format` 列出的文件。
@@ -83,8 +84,9 @@
 
 - **运行**：`make e2e`。它先执行 `make build`（没有改动时约 1 秒），再运行全部故事。需要 Docker：测试用 testcontainers 启动一个 Postgres 容器，运行结束后自动删除。
 - **测试环境**：`e2e/global-setup.ts` 启动 Postgres，用 `bin/nerve migrate up` 迁移模板库 `nerve_template`。每个 Playwright worker 从模板复制出自己的库，用 test 配置启动自己的 `nerve serve`：nerve 在 `127.0.0.1:0` 上监听，把实际地址写进地址文件（`server.addr_file`），fixture 读出地址，`/readyz` 返回 200 之后才运行故事。故事从 `e2e/fixtures/test.ts` 导入 `test`：用 `api`（生成的 TS 客户端）、`request`、`page` 访问本 worker 的 nerve，用 `db` 查询它的数据库（每个 worker 一个连接池；断言函数按表放在 `e2e/fixtures/assert/`）。需要另一种配置的故事（例如关闭注册）用 `nerveWith` 在同一个库上另起一个 nerve。
+- **页面的登录状态**：`signedInPage(tokens)` 给出一个已经登录的页面，`tokens` 来自 `login` 或 `register`（`nerveWith` 另起的 nerve 把它的地址作为第二个参数）。页面第一次加载时，在它的任何脚本运行之前，一条新的 `nerve.auth` 记录写进 localStorage；之后的加载、刷新和同一上下文里的其他标签页不再写，看到的是页面自己续期或删除之后的记录，和真实的浏览器一样。`e2e/fixtures/browser.ts` 的 `watchPage` 记录页面的接口请求和失败的接口请求、未处理的异常、控制台的错误和警告、CSP 违规，要在页面第一次打开之前调用。
 - **版本号**：`make build` 把 `VERSION`（默认 `0.1.0-dev`）写进 `bin/nerve`，例如 `make build VERSION=0.1.0`。`make e2e` 把同一个值放进环境变量 `NERVE_VERSION` 交给测试，S3 核对 `/api/v0/instance` 返回的版本号。
-- **同源**：S2 断言页面的所有请求都发往 nerve 自身（见"前端"一节的"没有前端环境变量"一条）；有请求发往别处时 S2 失败，失败信息列出这些请求。
+- **同源**：S2 断言页面的所有请求都发往 nerve 自身（见"前端"一节的"没有前端环境变量"一条）；有请求发往别处时 S2 失败，失败信息列出这些请求。S2 还断言：未登录的页面只请求 `GET /api/v0/instance`，没有失败的请求、未处理的异常、控制台错误和 CSP 违规；页面带 `Content-Security-Policy`。
 - **只运行部分故事、打开浏览器调试**：先 `make build`，再直接运行 Playwright，`NERVE_VERSION` 要与构建时的 `VERSION` 相同：
 
   ```bash
@@ -105,6 +107,8 @@
 
   访问令牌由它签名，刷新令牌的 MAC 密钥也从它派生。换密钥（替换文件后重启）的后果：已签发的访问令牌验签失败，客户端续期一次即可；换钥之前的旧代刷新令牌不再能被认出是重复使用，所以刷新令牌正被盗用时换钥，受害者只是被登出（恢复靠修改密码或 `nerve users reset-password`）；同一出口 IP 后面的大量标签页同时续期，会短暂得到 429。所以在低峰时换。
 - **反向代理**：nerve 前面有反向代理（例如 Caddy）时，把代理的地址写进 `server.trusted_proxies`（CIDR 列表；环境变量用逗号分隔，例如 `NERVE_SERVER__TRUSTED_PROXIES=10.0.0.0/8,fd00::/8`）。只有连接的对端在这个列表中时，nerve 才从 `X-Forwarded-For` 自右向左取第一个不可信的地址作为客户端 IP。不配置时，所有请求都算作代理的地址：按 IP 的限流（匿名请求、登录、注册、认证失败）让所有人共用一份额度；nerve 第一次收到不可信对端带来的 `X-Forwarded-For` 时记一条 WARN 提醒。代理必须往 `X-Forwarded-For` 里写不带端口的 IP 地址：某一项带端口或是主机名时，nerve 在转发它的那个代理处停下，这个代理后面的客户端都算作代理的地址（第一次遇到时同样记一条 WARN）。只信任你自己的代理的地址：`0.0.0.0/0`、`::/0` 这样信任所有地址的前缀让任何客户端都能自己选 IP，启动时被拒绝。IPv6 客户端按前缀计数（`ratelimit.ipv6_prefix_len`，默认 64）。
+- **HTTPS**：放在公网上的部署用 HTTPS（例如在前面加 Caddy，按上一条配置 `server.trusted_proxies`）。直接用 HTTP 部署也可以，例如在局域网里打开 `http://192.168.1.20:8080`：这时浏览器不在安全上下文中，没有 `navigator.locks`，同一个浏览器的多个标签页改用 localStorage 的租约轮流续期。租约不是原子的：极少数情况下两个标签页同时续期，或者持有租约的标签页被浏览器冻结超过 10 秒、另一个标签页接着续期，都会触发刷新令牌的重复使用检测，用户要重新登录（M2 设计 7.1、§16）。
+- **CSP**：nerve 给每个页面（HTML 响应）设 `Content-Security-Policy`：内联脚本按 `index.html` 里各自的 SHA-256 放行，请求只能发往本站（`connect-src 'self'`）。前端和接口必须同源；反向代理不要改写或另加这个响应头。
 - **注册**：prod 默认关闭注册（`auth.signup_enabled: false`）。第一个账户用 `nerve users create --email <邮箱>` 创建（见下一条），注册关闭时也能用。开放注册时，注册接口会暴露一个邮箱是否已经注册：没有邮件通道，就无法让两种回答相同。注册按客户端 IP 限流（默认每分钟 10 次），只能压低探测的速度。登录不暴露账户是否存在；存储的哈希都用当前的 argon2 参数时，耗时也相同。调高 `auth.password.argon2_memory_kib`、`argon2_iterations` 之后，还没有重新登录过的账户能从耗时上与不存在的邮箱区分开，直到它们各登录一次，所以这两个参数少调（M2 设计 §16）。关闭注册时，已注册和未注册的邮箱得到同一个 403。
 - **管理命令**：`nerve users` 下的五个命令直接连数据库执行，用与 `nerve serve` 相同的配置（`NERVE_ENV`、`NERVE_DATABASE__URL` 等），服务不用停。都用 `--email` 指定账户，邮箱按注册时的规则规范化（去掉首尾空白、转小写）。账户不存在、邮箱已被使用、密码不合规时，退出码为 1，打印一行说明，数据库不变。需要密码的 `create`、`reset-password` 在终端上不回显地提示输入两次；标准输入不是终端时读一行，供脚本使用（例如 `printf '%s\n' "$PASSWORD" | nerve users create --email ada@example.com`）。
   - `create --email <邮箱>`：建账户，不建会话。
````

- [ ] **Step 3: 交接**

`docs/v0/M2-auth/handoffs/M0-P5-frontend-api-notes.md`（对 `f27434c` 的差异）：

```diff
--- a/docs/v0/M2-auth/handoffs/M0-P5-frontend-api-notes.md
+++ b/docs/v0/M2-auth/handoffs/M0-P5-frontend-api-notes.md
@@ -1,5 +1,5 @@
 ---
-status: open
+status: done
 from: M0/P5
 to: M2
 created: 2026-09-22
@@ -29,3 +29,14 @@
 仍未处理，状态保持 `open`：CSP（M2/P4）；前端改调 `/api/v0/instance` 和认证接口，以及同源部署的核对（M2/P4）。
 
 来源：[M2/P2 spec](../specs/P2-sessions.md) 第 7 节。
+
+## 处理结果（M2/P4）
+
+- **CSP**（完成，M2 设计 8.3）：`webui.Handler` 构造时从内嵌的 `index.html` 取出内联脚本（不带 `src` 的 `<script>`），各算一个 SHA-256，只在 `.html` 的响应上设 `Content-Security-Policy`：`script-src 'self'` 加每个内联脚本的哈希，`connect-src 'self'`、`object-src 'none'`、`base-uri 'none'`、`form-action 'self'`、`frame-ancestors 'none'`；没有构建前端时不设。`webui` 的单元测试用另外算出的哈希核对；`bootstrap` 的测试核对页面带它，接口、健康检查和静态文件不带。S2 核对页面带它，S2、A10 核对没有 CSP 违规；浏览器核对逐个打开 M2 能到达的 8 个页面，都没有违规。
+- **安全响应头与 CSP 分在两层**（关闭，M2 设计 8.3）：三个安全响应头对接口的响应同样有意义，放在 `httpserver` 的固定链上（M2/P2）；CSP 只对页面有意义，而且要用 `index.html` 里内联脚本的哈希，只有 `webui` 知道这些脚本，所以由 `webui` 设置。交接原文"大概率是同一层中间件"的本意是两者都由服务端在 M2 加入，这一点已经做到；分在两层是有意的安排，P4 的 review 写下这条理由。
+- **前端改调新接口**（完成）：启动时的 `GET /api/instances/` 改为 `GET /api/v0/instance`；注册、登录、续期、退出走 `/api/v0/auth/`，当前账户、资料、修改密码走 `/api/v0/me`。前端不再调用 `/auth/…`，关键词规则 `plane-auth-urls` 看住。未登录的页面只请求 `GET /api/v0/instance`（S2）。
+- **同源部署**（核对）：前端没有环境变量（M1/P4），`@nerve/api-client` 的两个客户端都不设 `baseUrl`，接口一律用相对路径；S2 断言页面的请求都发往 nerve 自身；局域网 HTTP 和 Caddy 反向代理的浏览器核对也都是同源。
+
+全部处理完，状态改为 `done`。
+
+来源：[M2/P4 spec](../specs/P4-web-auth.md) 第 7 节。
```

`docs/v0/M2-auth/handoffs/M0-P6-e2e-notes.md`（对 `f27434c` 的差异）：

```diff
--- a/docs/v0/M2-auth/handoffs/M0-P6-e2e-notes.md
+++ b/docs/v0/M2-auth/handoffs/M0-P6-e2e-notes.md
@@ -90,3 +90,13 @@
 仍未处理，状态保持 `open`：页面的登录状态（M2/P4）；S2 的断言（M2/P4）；fixture 写法的延伸（M4、M5、M8）。
 
 来源：[M2/P3b spec](../specs/P3b-jobs-and-admin.md) 第 7 节。
+
+## 处理结果（M2/P4）
+
+- **页面的登录状态**（完成）：fixture `signedInPage(tokens, baseURL?)` 给出一个已经登录的页面：页面第一次加载时，在它的任何脚本运行之前，`e2e/fixtures/auth.ts` 的 `signInContext` 把一条新的 `nerve.auth` 记录写进 localStorage；之后的加载、刷新和同一上下文里的其他标签页不再写，页面自己续期、退出。A4、A5、A6、A10 的页面版本用它，A1–A3、A15 的页面版本从登录页、注册页开始；页面版本调用接口版本的同一组断言函数（`expectRegistered`、`expectSignedIn`、`expectRefreshed`、`expectRevoked`、`accountOf` 等）。
+- **S2**（完成）：`e2e/fixtures/browser.ts` 的 `watchPage` 记录页面的接口请求、失败的接口请求、未处理的异常、控制台的错误和警告、CSP 违规。S2 断言未登录的页面只请求 `GET /api/v0/instance`，失败的接口请求、未处理的异常、控制台错误和 CSP 违规都为空（原文"是否断言控制台没有错误"：断言），页面带 `Content-Security-Policy`；深链接跳到带 `next_path` 的登录页。前端没有轮询，S2 仍用 `networkidle`。A10 用同一个 `watchPage` 核对第一次打开 `/onboarding`。
+- **新等待的期限**：页面故事的等待都受 Playwright `expect` 的超时约束；A4 的 `holdFirstRefresh` 最多扣住第一个续期请求 1 秒，等访问令牌过期的 `expect.poll` 以 10 秒为限。
+
+仍未处理，状态保持 `open`：fixture 写法的延伸（M4、M5、M8），由收尾转交给这些 M。
+
+来源：[M2/P4 spec](../specs/P4-web-auth.md) 第 7 节。
```

`docs/v0/M2-auth/handoffs/M1-P2-trim-content.md`（对 `f27434c` 的差异）：

```diff
--- a/docs/v0/M2-auth/handoffs/M1-P2-trim-content.md
+++ b/docs/v0/M2-auth/handoffs/M1-P2-trim-content.md
@@ -1,5 +1,5 @@
 ---
-status: open
+status: done
 from: M1/P2
 to: M2
 created: 2026-09-23
@@ -33,3 +33,11 @@
 仍未处理，状态保持 `open`：前端的 `IUserTheme` 改用生成的类型（M2/P4、P5）。
 
 来源：[M2/P3a spec](../specs/P3a-account-api.md) 第 7 节。
+
+## 处理结果（M2/P4）
+
+- **前端的类型**（完成）：`IUserTheme` 连同 `IUser`、`TUserProfile`、`TOnboardingSteps` 一起删除，资料 store 和个人设置、新手引导、主题切换直接用生成的 `Profile`、`ProfileUpdate`，主题是生成的枚举 `Theme`；`THEME_OPTIONS` 各项的 `value` 类型是 `Theme`（`@nerve/constants` 为此依赖 `@nerve/api-client`），写错或写出第六个值都过不了类型检查。前端不再有 `custom` 主题的分支。设计原把这一项分给 P4、P5，资料 store 在 P4 整个改写，所以在 P4 一次完成。
+
+全部处理完，状态改为 `done`。
+
+来源：[M2/P4 spec](../specs/P4-web-auth.md) 第 7 节。
```

`docs/v0/M2-auth/handoffs/M1-P3-trim-platform.md`（对 `f27434c` 的差异）：

```diff
--- a/docs/v0/M2-auth/handoffs/M1-P3-trim-platform.md
+++ b/docs/v0/M2-auth/handoffs/M1-P3-trim-platform.md
@@ -79,3 +79,13 @@
 仍未处理，状态保持 `open`：Cookie 会话和 CSRF、认证错误就地显示、前端改读新的实例字段（M2/P4）。
 
 来源：[M2/P3b spec](../specs/P3b-jobs-and-admin.md) 第 7 节。
+
+## 处理结果（M2/P4）
+
+- **Cookie 会话和 CSRF**（完成）：四处都已换掉。登录、注册由 `useUser().signIn`、`signUp` 用 JSON 调 `POST /api/v0/auth/login`、`register`，令牌交给令牌管理器；退出由令牌管理器在续期用的那把锁下调 `POST /api/v0/auth/logout`；修改密码调 `POST /api/v0/me/change-password`，访问令牌由认证中间件加上。web 的 axios 基类删掉 `withCredentials` 和 401 拦截。`git grep -n -E "csrfmiddlewaretoken|X-CSRFTOKEN" -- web` 没有输出，关键词规则 `csrf`（不区分大小写）和 `plane-auth-urls` 看住。
+- **认证错误就地显示**（完成，M2 设计 7.3）：接口直接返回 problem，登录、注册表单把它显示在表单上方或对应字段下方（`weak_password` 显示规则，`common_password` 显示"密码太常见"），不跳转，输入的内容留在表单里；文案按 `problem.code` 取（`helpers/authentication.helper.ts`，带单元测试）。前端不再读 `/?error_code=…`，Plane 的错误码枚举 `EAuthenticationErrorCodes`（原在 `helpers/authentication.helper.tsx`）和 `@nerve/types` 的 `auth.ts` 删除，关键词规则 `auth-error-code` 看住。A2、A3、A15 的页面版本覆盖。
+- **实例配置**（完成）：`IInstanceConfig`、`IInstanceInfo` 删除，instance store 直接用生成的 `InstanceInfo`；注册链接读 `signup_enabled`，创建工作区的四处读 `workspace_creation_enabled`，`use-file-size` 读 `file_size_limit`。新手引导的"角色""用途"两步连同 `is_self_managed` 删除（M2 设计 3.19），关键词规则 `is-self-managed` 看住。
+
+仍未处理，状态保持 `open`：`packages/services` 的 API 令牌服务仍调旧地址 `/api/users/api-tokens/…`（没有结尾斜杠的 `retrieve`、`destroy` 也在其中），M2/P5 把 api-tokens 标签页改接 `/api/v0/me/api-tokens`、`/api/v0/api-tokens/{token_id}` 时删除。
+
+来源：[M2/P4 spec](../specs/P4-web-auth.md) 第 7 节。
```

`docs/v0/M2-auth/handoffs/M1-P4-router-native.md`（对 `f27434c` 的差异）：

```diff
--- a/docs/v0/M2-auth/handoffs/M1-P4-router-native.md
+++ b/docs/v0/M2-auth/handoffs/M1-P4-router-native.md
@@ -1,5 +1,5 @@
 ---
-status: open
+status: done
 from: M1/P4
 to: M2
 created: 2026-09-24
@@ -32,3 +32,17 @@
 逐项的结论写进该 M 的 review，然后 `status` 改为 `closed`。
 
 来源：[M1/P4 评审记录](../../M1-frontend-trim/reviews/P4-router-native-review.md)第 7 节。
+
+## 处理结果（M2/P4）
+
+- **服务端校验 `next_path`**（按"服务端没有跳转"关闭，M2 设计 3.18）：M2 的登录、注册是 JSON 接口，服务端不发出跳转，也不接收 `next_path`；表单不再提交它（隐藏字段随表单的重写删除）。跳转只发生在浏览器里，唯一的关口是 `@nerve/utils` 的 `isValidNextPath`：修剪首尾空白后以单个 `/` 开头，不是 `//`、`/\`，任何位置都没有控制字符（`\u0000`–`\u001f`、`\u007f`；浏览器会从地址里丢掉制表符和换行，`/\t/evil.example` 会变成 `//evil.example`）。单元测试覆盖 `//`、`\`、`/\`、协议和控制字符；A3 的页面版本打开四个不合格的 `next_path`，登录后都落到 `/create-workspace`。
+- **`AuthenticationWrapper`**（完成，M2 设计 7.4）：随令牌管理器重写。会话状态来自令牌管理器（`useSession`）；未登录时 `<Navigate to={signInPath(…)} replace />`；已登录而在登录页、注册页时，去合格的 `next_path`，没有时按是否完成新手引导去 `/onboarding` 或 `/create-workspace`（需要账户的页面仍先要求完成新手引导）；"会话暂不可用"时显示可重试的界面，地址和记录都不变。跳转只在这一处发生。
+- **401 处理**（完成，M2 设计 7.1、7.2）：web 的 axios 基类删掉 401 拦截和 `withCredentials`，恒为真的 `currentPath` 判断没有照搬；401 由 `@nerve/api-client` 的认证中间件处理：续期一次、重发一次，只有续期得到 401 或重发后仍是 401 才结束会话。
+- **"由 M2 决定"的三项**：
+  - `next_path` 带上查询参数和片段（M2 设计 3.18）：`signInPath` 取 `pathname + search + hash`，整体 `encodeURIComponent`；单元测试和 A3 的页面版本（`/settings/profile/general?tab=x#y` 登录后原样回到）覆盖。
+  - 修改密码页的错误断言（完成）：`Error & { error_code?: string }` 和 `toString()` 删除；错误是 `ApiError`，按 `problem.code`（`identity.current_password_incorrect` 显示在当前密码下方）和 `problem.errors`（`new_password` 的规则）显示，类型来自接口描述。
+  - 表单提交的地址（完成）：登录、注册、退出不再提交到 `/auth/…`，改为 `@nerve/api-client` 调 `/api/v0/auth/…`，关键词规则 `plane-auth-urls` 看住。
+
+全部处理完，状态改为 `done`。
+
+来源：[M2/P4 spec](../specs/P4-web-auth.md) 第 7 节。
```

- [ ] **Step 4: 检查**

Run: `make lint-web`
Expected: 通过。

Run: `grep -nF '/auth/…' README.md`
Expected: 没有输出（前端不再有 Plane 的 `/auth/…` 地址）。

- [ ] **Step 5: 提交**

```bash
git add docs/v0/v0-design.md docs/v0/frontend-changes.md README.md docs/v0/M2-auth/handoffs
```
```bash
git commit -m "docs(M2/P4): the browser side in the v0 design, the README and the handoffs

v0 design 4.3 gets login_id, the lease and when a session ends; the
frontend change list marks the CSRF and store rows done; the README
covers the token manager, the pages of M2, HTTPS and the lease, the CSP
and the e2e helpers. M0-P5, M1-P2 and M1-P4 are done; M0-P6 and M1-P3
keep what is left for later.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** 文档与代码一致；交接的状态如上。
