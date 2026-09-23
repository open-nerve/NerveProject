# M1/P4 `router-native`：去掉 Next.js 兼容层 Spec

- 上级设计：[M1 设计](../M1-design.md)（3.12、4、7.4、7.5、9 节 P4、9.7）
- 前一 Phase：[P3 spec](P3-trim-platform.md)、[P3 plan](../plans/P3-trim-platform.md)、[P3 review](../reviews/P3-trim-platform-review.md)（第 4 节裁定、第 7 节"交给 P4"）
- 实施计划：[P4 plan](../plans/P4-router-native.md)
- 分支：`worktree-m1-p4-router-native`，基线 `9437a6e`（P3 合并后的 `main`）

---

## 1. 目标

web 应用只用 React Router 自己的写法，不再经过 Next.js 兼容层，也不再有前端环境变量：

- 各包不读取任何环境变量，接口一律用相对路径（同源部署）；删掉 dotenv、`define: { "process.env": … }` 和 `.env.example`；
- `next/link`、`next/navigation` 和包装层 `useAppRouter` 全部改为 React Router 的 `Link`、`NavLink`、`useLocation`、
  `useSearchParams`、`useNavigate`、`useParams`、`useMatch`、`matchPath`；命令面板上下文带 `navigate: NavigateFunction`；
- 路由参数按真实的 `string | undefined` 处理：路由组件用 `./+types/*` 的参数，共享组件用守卫，不用非空断言或 `as string`；
- `AuthenticationWrapper` 的渲染时跳转改为 `<Navigate replace />`；
- 去掉强制结尾 `/`：应用内部的地址一律不带结尾 `/`，"当前是哪一项"的判断对带和不带 `/` 的地址给出相同的结果；
- 删除 Plane 的 10 条旧地址重定向，先把仍然依赖它们的入口改为正式地址；
- 一个进仓库的路由匹配测试，核对应用里写出的每个内部跳转目标都落在真实的页面上。

同时处理交给 P4 的事项：P2 评审第 7 节（结尾 `/` 的重定向丢掉 `#` 片段）、P3 评审第 7 节（6 处渲染时跳转、10 条旧地址
重定向和 `/sign-in` 丢掉查询参数、`next/*` 垫片），以及 M0 的两份交接中 `.env`、dotenv、`define` 和深层路径结尾 `/` 的事项
（M1 设计 9.7）。

P4 不改认证和会话的方式（M2）、不改品牌（P5）、不处理基线就存在的 `.toString()` 空转换和死资源（收尾，第 7 节）。

---

## 2. 交付物

### 2.1 文件总览

6 个 Task，每个一个提交，顺序就是 M1 设计 9 节 P4 的四步（第 3、4 步各拆成两个 Task，见第 3 节第 1 条）。
原型（`9437a6e..270f8e6`，在 `$P4TMP/proto`）实测总计 `373 files changed, 1299 insertions(+), 1650 deletions(-)`：
删 20 个、新增 2 个、修改 351 个。下表是每个提交各自的数字（同一个文件会被几个 Task 修改）。

| Task | 标题 | 删 | 增 | 改 | 行（+/−） | 原型提交 |
|---|---|---:|---:|---:|---|---|
| 1 | 各包不再读取环境变量 | 2 | — | 47 | +35 / −291 | `479609f` |
| 2 | 删除 `process.env` 注入、dotenv 和 `.env.example`；守卫的 `phase` 改为 `M1/P4` | 1 | — | 7 | +43 / −45 | `6fad945` |
| 3 | `Link`、location、search params、`navigate`；渲染时跳转改为 `<Navigate>` | 3 | — | 136 | +492 / −557 | `2292eea` |
| 4 | 路由参数取自 React Router | 2 | — | 196 | +492 / −422 | `e168fdc` |
| 5 | 每个内部跳转落在页面上；删除旧地址重定向；路由匹配测试 | 10 | 2 | 9 | +195 / −208 | `a3a957b` |
| 6 | 去掉强制结尾 `/`，当前项用 React Router 匹配；垫片的最后部分 | 2 | — | 54 | +164 / −249 | `270f8e6` |

### 2.2 原型验证：结论与证据

在 `$P4TMP/proto`（从本分支的基线克隆）中逐个 Task 做过一遍，每个 Task 结束时 `pnpm exec turbo run check:types`、
`make lint-web`、`make test-web`、`make build-web`、`make knip` 都通过。Task 3–6 另外在一个干净的克隆上用脚本重放过
（`$P4TMP/replay`），得到的树与提交完全相同。下面是基线与终态的实测值，plan 的每一步都写了当步的预期输出和它来自哪个原型提交。

| 项 | 基线（`9437a6e`） | P4 结束（`270f8e6`） |
|---|---|---|
| `make lint-web` | `keywords: 42 rules, 4 exceptions, no hits.` + 52 个 turbo 任务 | `keywords: 45 rules, 4 exceptions, no hits.` + 52 |
| `pnpm exec turbo run check:types` | 23 | 23 |
| `make test-web` | 15 个任务；vitest 63 个测试（constants 11、editor 16、i18n 10、services 13、utils 13） | 16 个任务；66 个（web 5、constants 11、editor 16、i18n 10、services 11、utils 13） |
| `make build-web` | 11 | 11 |
| `make knip` | 门禁，零 | 零 |
| lint 上限合计 | 711（web 566） | 711，不变 |
| 每种语言的文案键 / 无引用的键 | 1617 / 498 | 不变 |
| 环境变量的读取（`web/`、`turbo.json`、`pnpm-workspace.yaml` 中的 `process.env`、`VITE_*`、`dotenv`，行数） | 27 | 0 |
| 兼容层的引用（`next/link`、`next/navigation`、`useAppRouter`、`ensureTrailingSlash`、`compat/next`，行数） | 396（269 个文件） | 0 |
| 渲染时跳转（`render.mjs`） | 6（都在 `AuthenticationWrapper`） | 0 |
| 路由参数改为 React Router 的类型后的类型错误 | 102 处，43 个文件（设计写的 152 / 68 是 P2 之前的数字） | 0；没有新增的非空断言和 `as string` |
| 内部跳转目标（`targets.mjs`）/ 其中以 `/` 结尾的 | 351 / 39 | 343 / 0 |
| 路径字面量（`pathlits.mjs`）/ 落不到页面的 | 228 / 9 | 209 / 4（都是后面还要拼接的前缀，见结论 6） |
| 以 `/` 结尾的路径字面量（`slashes.mjs`，含只作后缀的） | 62 | 1（`utils/url.ts` 的 `"//"`，不是路径） |
| 按文本判断"当前是哪一项"的地方 | 17 处，另有侧边栏的 6 个 `highlight`、设置的 12 个没有调用方的 `highlight`、`PROFILE_TABS` 的 3 个 `selected` | 0（2.7） |
| 旧地址重定向 | 10 条路由、10 个模块 | 0 |
| 进仓库的路由匹配测试 | 无 | `app/routes/navigation.test.ts`，5 个测试；扫出 121 个文件里的 241 个路径（185 个带 `${…}`） |
| 构建体积 js | 422 个 / 6,881,280 字节 | 397 个 / 6,858,278 字节 |
| 构建体积 css / fonts | 3 / 296,988；25 / 3,755,608 | 不变 |
| 构建体积 other | 112 / 7,910,515 | 112 / 7,910,263 |
| 最大 chunk / 语言 chunk | `use-parse-editor-content` 1,379,329 / 34 | 1,379,330 / 34 |
| 锁文件 | — | catalog 删 `dotenv`；web 删 `dotenv`、加 `vitest`；i18n 加 `vite`；删 1 个包（`dotenv`） |
| `make web-dev` 首页的 "process is not defined" | 0（有 `define`） | 0（Task 2 之后实测，端口 3000，已停止） |

每个 Task 结束时的中间值（`$P4TMP/notes/per-task.mjs` 在各提交上重算）：

| 原型提交 | 环境读取 | 兼容层引用 | 渲染时跳转 | 参数类型错误 | 跳转目标 / 以 `/` 结尾 | 路径字面量 / 落空 | 守卫规则 | 测试任务 |
|---|---:|---:|---:|---|---|---|---:|---:|
| 基线 `9437a6e` | 27 | 396 | 6 | （102 / 43） | 351 / 39 | 228 / 9 | 42 | 15 |
| T1 `479609f` | 15 | 396 | 6 | — | 351 / 39 | 228 / 9 | 42 | 15 |
| T2 `6fad945` | 0 | 396 | 6 | — | 351 / 39 | 228 / 9 | 43 | 15 |
| T3 `2292eea` | 0 | 183 | 0 | — | 347 / 39 | 228 / 9 | 45 | 15 |
| T4 `e168fdc` | 0 | 3 | 0 | 102 → 63 → 0 | 347 / 39 | 228 / 9 | 45 | 15 |
| T5 `a3a957b` | 0 | 3 | 0 | 0 | 337 / 35 | 217 / 5 | 45 | 16 |
| T6 `270f8e6` | 0 | 0 | 0 | 0 | 343 / 0 | 209 / 4 | 45 | 16 |

结论：

1. **设计的顺序是必须的。** 各包用 tsdown 构建的 `dist/` 原样保留 `process.env.X`，靠 `define` 才不报错，所以先删读取
   （Task 1），再删 `define`（Task 2）；Task 2 之后用 `make web-dev` 打开首页，控制台没有 "process is not defined"。
   参数类型随 `next/navigation` 一起换（Task 4），所以 Task 3 把垫片削到只剩 `useParams`，而不是一次删光。
2. **垫片推迟了每一次跳转，所以渲染时跳转"能用"。** 垫片的 `push`、`replace`、`back` 都包在 `setTimeout` 里，并给目标补
   结尾 `/`。`render.mjs` 按"调用在哪个函数里、那个函数怎么被执行"逐个分类：基线有 82 个跳转变量、103 处调用，只有
   `AuthenticationWrapper` 的 6 处在渲染时执行，其余在 JSX 事件（21）、effect（5）、回调和局部函数里。这 6 处改为
   `<Navigate replace />`，之后为 0。
3. **参数类型错误的重新统计是 102 处、43 个文件。** 把垫片的手写类型换成 React Router 的类型后实测：`app/` 下 12 个文件
   39 处，都是路由自己的页头组件；`core/` 下 31 个文件 63 处。前者由布局用 `Route.ComponentProps` 的参数传进去（63 处剩下），
   后者逐个按真实情况处理（0 处剩下）。每一种处理都有挂载证据（2.5）。
4. **只有一个路由真的缺参数。** `mounts.mjs` 顺着引用把每个改动的共享组件追到它挂载的路由，再列出缺哪个参数：
   唯一真实的缺失是个人设置 `/settings/profile/:profileTabId` 没有 `workspaceSlug`（命令面板在那里挂载了工作项弹窗）。
   其余守卫在每个挂载路由上参数都存在，只是让类型成立。
5. **5 个入口到不了页面，或只靠旧地址重定向才到达。** 停用的迭代、模块、视图、收集箱页面上"去设置里开启"的按钮
   指向 `/:workspaceSlug/settings/projects/:projectId/features`，这个地址没有路由（Plane 也一样），点了就是"页面不存在"；
   命令面板的"账户设置"靠旧地址重定向才到达 `/settings/profile/general`。另有两处死代码指向没有路由的地址：`getTabUrl` 的
   `overview`（没有导航项用它）和 `tab-navigation-root.tsx` 从不执行的"项目根地址"跳转（Task 6）。Task 5 先改掉这些入口，再删重定向。
6. **路由匹配测试读源码和路由表，而不是一份清单。** 它用 TypeScript 的语法树找出 `app/`、`core/`、`helpers/` 里所有以 `/`
   开头的字符串和模板字面量（接口地址、文件除外），加上命令面板 `handlePowerKNavigate` 的各段，把每个 `${…}` 当作"同一段
   里的任意文本"，按每个页面的路径填进去，用 `matchRoutes` 匹配真实的路由表。以后新写的跳转、改掉的路由都会被它看到。
   `pathlits.mjs` 把 `${…}` 一律当成 `x`，所以还剩 4 个"落空"：它们都是后面还要拼接的前缀（例如
   `` `/${workspaceSlug}/projects/${projectId}/archives/${tab.key}` ``），测试按页面路径填入之后都能落到页面。
   故意改回两个旧目标，测试失败并列出这两处（plan Task 5）。
7. **`NavLink` 的 `end` 不能容忍结尾 `/`。** React Router 8.3 的 `NavLink` 在 `end` 时按文本比较地址
   （`lib/dom/lib.js` 第 387–388 行），`/acme/` 不算 `/acme`；`matchPath` 的精确匹配则允许结尾 `/`。所以侧边栏
   （首页、项目两项只在地址完全相同时算当前）用 `useMatch`，只有"本地址及以下都算当前"的标签页用 `NavLink`。
8. **没有新的共享可变状态，路由模块的约定导出没有动。** 新增的顶层声明只有组件、函数和测试文件里的只读数据；
   `route-exports.mjs` 核对 `app/` 下每个改动的文件：约定导出只有 `app/layout.tsx` 的 `clientMiddleware`（强制结尾 `/` 本身）
   被删掉，其余变化都是随文件删除的旧重定向和垫片。

### 2.3 环境变量（Task 1、2，M1 设计 4.1）

- **Task 1：各包不再读取环境变量**
  - `constants/src/endpoints.ts` 整个删除：8 个常量中只有 `API_BASE_URL`（35 个文件，70 行）和 `SUPPORT_EMAIL` 还有读取方。
  - `APIService`（web 和 `@plane/services` 各一份）不再接收基础地址；33 个 service 去掉只调用 `super(API_BASE_URL)` 的
    构造函数（`t1/services.mjs`）；登录、注册表单和退出直接提交到 `/auth/…`；`AuthService.signOut()` 不再带参数。
  - `normalizeAPIRequestURL` 只给相对地址补结尾 `/`，以协议开头的绝对地址（签名上传地址）原样返回；单元测试 5 → 3 个。
    判断用正则 `^[a-z][a-z\d+\-.]*:`，不用 `URL.canParse`（比 Vite 默认的构建目标新）。
  - `getFileURL` 变成 `path || undefined`：资源地址是同源的相对路径，调用方依赖"空路径得到 `undefined`"给 `img` 的 `src`。
  - 停用账户的错误提示改为"请联系管理员"（原来读 `VITE_SUPPORT_EMAIL`），与同一张表里"注册已关闭"的写法一致。
  - i18n 的开发提示改读 `import.meta.env.DEV`；i18n 加开发依赖 `vite`（catalog 版本），`tsconfig.json` 加 `vite/client` 类型。
  - 随之没有读取方的 `UserService.currentUserConfig`（用基础地址拼地址）和整个被注释掉的 `app/(all)/layout.preload.tsx` 删除。
- **Task 2：注入和配置**
  - `vite.config.ts` 删掉 dotenv 加载和 `define`；`.env.example` 删除；dotenv 从 web 的开发依赖和 catalog 删除。
  - `turbo.json` 的 `globalEnv` 只留 `NODE_ENV`（它决定 Vite 构建的模式），删掉 6 个 `VITE_*` 和没人读的 `DEV`；
    `build` 任务的 `inputs` 只为 `.env*` 存在，删除。
  - README 的两处说明（P6 交接）：前端一节改为"没有前端环境变量"，端到端测试一节的"同源"不再引用 `.env`。
  - 守卫规则 `frontend-env`；顶层 `phase` 改为 `M1/P4`（没有 `until: M1/P4` 的例外）。

### 2.4 Link、hooks、命令面板上下文与渲染时跳转（Task 3，M1 设计 4.2）

- **机械部分**（`t3/codemod.mjs`，ts-morph 只读语法树、按文本范围替换，129 个文件）：
  `next/link` 的 `Link href` → React Router 的 `Link to`；`usePathname()` → `useLocation().pathname`；
  `useSearchParams()` → `const [searchParams] = useSearchParams()`；`useRouter()` / `useAppRouter()` → `useNavigate()`，
  `push(x)` → `navigate(x)`，`replace(x)` → `navigate(x, { replace: true })`，`back()` → `navigate(-1)`；
  依赖数组和对象简写里的 `router` → `navigate`。脚本遇到不认识的用法就停下并指出文件和行。
- **手工部分**：
  - `AuthenticationWrapper` 的 6 处渲染时跳转改为 `return <Navigate to=… replace />`；`next_path` 的写法收成
    `` `/?next_path=${pathname}` ``（`pathname` 不会为空）。原来 4 处是 `push`，现在守卫跳转一律不留下可以后退回来的记录（设计 4.2）。
  - 命令面板上下文 `router` → `navigate: NavigateFunction`（`core/types.ts`、`utils/navigation.ts`、创建命令）。
  - `use-cycles-details.ts` 返回值里的 `router`（对象简写换名后是 `navigate`）没有读取方，删除。
  - 删除 `next/link` 的垫片、类型声明和别名、`use-app-router.tsx`；`next/navigation` 的垫片只剩 `useParams`（Task 4 删）。
- 守卫规则 `next-shim-files`、`next-shims` 在这个提交加入（覆盖已删的部分：184 行、107 个文件，之后为零）。

### 2.5 路由参数（Task 4，M1 设计 4.2）

`useParams` 从 `react-router` 导入（`t4/imports.mjs`，178 个文件），删除 `next/navigation` 的垫片、类型声明、Vite 别名
（`vite.config.ts` 的整个 `alias` 块和 `node:path` 导入）。102 处类型错误按下面的方式处理：

- **路由自己的页头**（`app/` 下 12 个页头 / 移动端页头，`t4/headers.mjs` 改 24 个文件）：布局组件取
  `{ params }: Route.ComponentProps`（参数在该布局下的每个页面都存在），传给页头；页头改收 `TProps`，不再调用 `useParams`。
- **共享组件**（`core/` 下，`t4/params.mjs` 改 38 个文件），每处写出种类：

| 种类 | 含义 | 文件 |
|---|---|---|
| guard | 处理函数或 effect 没有参数时提前返回（周围代码本来就用的写法） | 迭代的转移弹窗、日历选项、标签下拉、项目列表根组件、视图列表、收藏菜单（4 个处理函数）、新建收藏文件夹 |
| render | 组件没有参数时不渲染：最后一个 hook 之后 `return null`，或在渲染回调里 `x && <C/>` | 工作项页头、工作项弹窗（`base.tsx`、`form.tsx`、`draft-issue-layout.tsx`）、项目标签设置、模块列表项、个人工作项、收集箱设置页头、视图列表项、邀请弹窗、成员列、收藏文件夹、收藏菜单、侧边栏项目项及渲染它的两个列表 |
| pass | 被调用方的参数本来可选，值原样传入（去掉无效的 `.toString()`） | 4 个项目视图布局根、2 个个人工作项布局根、父工作项选择弹窗 |
| widen | 被调用方的函数体本来就处理缺失的值，把类型改成实话 | `isUserSuspended`（成员下拉）、`workspaceInfoBySlug`（邀请列表项） |
| delete | 组件从来不读传给它的 prop | 命令面板状态菜单的 `projectId`、`workspaceSlug` |
| type | 命令面板上下文带 React Router 的 `Params` | `core/types.ts`、两个命令面板 provider（工作项的 `project_id` 可能为 `null`） |

- **唯一真实的缺参**（2.2 结论 4）：个人设置页没有工作区。工作项弹窗在那里不再打开（`base.tsx` 在"没有项目"的判断里加上
  "没有工作区"），表单和草稿布局改从弹窗取 `workspaceSlug`，不再各自读地址；成员下拉的 `isUserSuspended` 在没有工作区时
  本来就回答 `false`。基线在这里如果打开了弹窗（项目 store 里还留着上一个工作区的项目时），提交会在
  `workspaceSlug.toString()` 上抛错。
- 守卫让后面的判断恒为真时，那个判断一起收掉（例如 `project-setting-label-list.tsx` 的 `onDrop`、收集箱页头第 72 行）。

### 2.6 内部跳转目标、旧地址重定向与路由匹配测试（Task 5，M1 设计 3.12、4.2）

- **先改掉落不到页面的目标**（`t5/edits.mjs`）：
  - 命令面板的"账户设置"直接去 `/settings/profile/general`；
  - 停用的迭代、模块、视图、收集箱页面的按钮分别去 `/settings/projects/:projectId/features/{cycles,modules,views,intake}`；
  - `getTabUrl` 里没有导航项、也没有路由的 `overview` 删除。
- **再删 10 条旧地址重定向**：`routes/core.ts` 末尾的整段（`/:workspaceSlug/projects/:projectId/settings/*`、
  `/:workspaceSlug/settings/api-tokens`、`/:workspaceSlug/projects/:projectId/inbox`、`/accounts/sign-up`、`/sign-in`、
  `/signin`、`/login`、`/register`、`/profile/*`、`/:workspaceSlug/settings/account/*`）和它们的 10 个模块。
  P2 的 `profile-index.tsx`（个人主页重定向到第一个标签页）是正式路由的一部分，保留。
- **路由匹配测试** `web/apps/web/app/routes/navigation.test.ts`（Task 5 加 4 个，Task 6 加 1 个，并给常量那一个加结尾 `/` 的断言）：
  - 扫到的路径多于 200 个（防止扫描悄悄失效）；
  - 源码里写出的每个路径都落在页面上；
  - 源码里写出的路径都不以 `/` 结尾（Task 6）；
  - 导航常量的每一项（侧边栏、工作区设置、项目设置、个人主页标签、`generateWorkItemLink`）都落在页面上、不以 `/` 结尾；
  - 没有页面的地址落到"页面不存在"：旧的账户设置地址、原来那几个按钮的 `/features`。
- web 应用第一次有单元测试：`package.json` 加 `test` 脚本和开发依赖 `vitest`；新增 `vitest.config.ts`（`environment: "node"`），
  否则 vitest 会加载 `vite.config.ts` 里 React Router 的构建插件；它带 `test` 键，knip 才把测试文件算作入口。

### 2.7 强制结尾 `/`、当前项判断与垫片的最后部分（Task 6，M1 设计 4.1、4.2）

- **强制结尾 `/`**：`app/layout.tsx` 的 `clientMiddleware`（308 到带 `/` 的地址，用的是 `request.url`，丢掉 `#` 片段）和
  `app/compat/next/helper.ts`（`ensureTrailingSlash`）删除。
- **地址不带结尾 `/`**：30 个字面量目标（24 个文件，`t6/slashes.mjs`）；侧边栏常量的 `href` 不带 `/`，首页的 `href` 为 `""`；
  顶部通知 `/${workspaceSlug}/notifications`。
- **"当前是哪一项"一律用 React Router 匹配**（`t6/edits.mjs`）：

| 位置 | 基线的判断 | 改为 |
|---|---|---|
| 工作区侧边栏 `sidebar-item.tsx` 和常量的 6 个 `highlight` | `pathname === url` / `includes(url)` | `useMatch({ path, end })`；首页、项目两项 `end: true`（2.2 结论 7） |
| 个人主页标签 `profile/[userId]/navbar.tsx` | `pathname === …${tab.selected}`（带 `/`） | `NavLink` |
| 个人主页布局 `profile/[userId]/layout.tsx` | `includes("assigned")…`、`=== …${tab.selected}` | 路由参数 `profileViewId`；`PROFILE_TABS` 删 `selected` |
| 归档标签 `archive-tabs-list.tsx` | `includes(tab.key)` | `NavLink` |
| 工作区设置侧边栏 `item-categories.tsx` | `=== \`${href}/\`` / `RegExp` | `matchPath`，"常规"一项精确匹配 |
| 项目设置侧边栏 `item-categories.tsx` | 同上 | `matchPath`，"常规"一项精确匹配 |
| 通知页 `_sidebar.tsx`、`use-workspace-paths.ts`（3 个） | `includes` | `useMatch`；`isProjectsPath` 改为"在工作区里且不在设置里"（基线靠补上的 `/` 在工作区首页也成立） |
| 顶部通知 `top-navigation-root.tsx` | `includes("/notifications/")`（不带 `/` 时不成立） | `useMatch` |
| 应用栏的设置 `app-rail-root.tsx` | `includes(\`/${slug}/settings\`)` | `useMatch({ end: false })` |
| 归档项目页 `project/header.tsx`、`project/root.tsx` | `includes("/archives")` | `useMatch` / `matchPath` |
| 侧边栏项目列表展开 `projects-list.tsx` | `includes("projects")` | `matchPath({ path: "/:workspaceSlug/projects", end: false })` |
| 项目导航 `project-navigation.tsx`、`use-active-tab.ts` | `includes(href)` / `=== \|\| startsWith` | `matchPath({ path: href, end: false })` |
| `tab-navigation-root.tsx` 的"项目根地址跳到默认标签页" | `pathname === root \|\| === root + "/"` | 删除：没有路由服务项目根地址，组件从不在那里挂载 |
| 设置的 12 个 `highlight`（工作区 3、项目 9）和类型字段 | 没有调用方 | 删除 |

  设置布局的 `settings/helper.ts`（按段切分地址，取移动端页头的标题和权限键）不按文本比较整个地址，结尾 `/` 不影响结果，保留。
- **Next.js 的其他遗留**：`typescript-config/nextjs.json`（及其 `files` 条目）、`.oxlintrc.json` 的 `.next/**`、editor 包的
  `nextjs` 关键词。守卫规则 `next-shim-files`、`next-shims` 扩大到整个 `app/compat/next/`、`nextjs.json`、`ensureTrailingSlash`、
  `compat/next`（之前 3 行、2 个文件和 2 个文件，之后为零）。
- `constants` 的 `navigation.test.ts`：`href` 以 `/` 开头的断言和 `selected` 的断言删除（落点由 web 的路由测试核对），测试改名。
- **文档**：`docs/v0/frontend-changes.md` 新增 1.4 节，第二节 3 行改为"已完成 / M1/P4"；M0 的两份交接写"处理结果（M1/P4）"，
  `M0-P5-frontend-trim-notes` 改为 `closed`（第 7.4 节）。

### 2.8 关键词守卫（M1 设计 7.4）

P3 的 42 条规则之上新增 3 条，全部 `"phase": "M1/P4"`；每个顶层分支和 `(?:a|b)` 的每一支都有真实代码里的命中样本
（`alts.mjs M1/P4` 输出 0）。例外仍是 P3 的 4 条，本 Phase 不登记新的例外。

| 规则 | 要点 | 文件范围 | Task |
|---|---|---|---|
| `frontend-env` | `process\.env`、`\bVITE_[A-Z]`、`\bdotenv\b` | `web/` 的源码和 JSON、`web/**/.env*`、`turbo.json`、`pnpm-workspace.yaml`；`e2e/`、`tools/` 是 Node 端代码，不在范围内（第 3 节第 9 条） | 2 |
| `next-shim-files`（文件名） | `app/compat/next/` 整个目录、`app/types/next-*.d.ts`、`core/hooks/use-app-router.tsx`、`typescript-config/nextjs.json` | — | 3（Task 6 扩大） |
| `next-shims` | `["'\`]next/(?:link\|navigation)["'\`]`、`useAppRouter`、`ensureTrailingSlash`、`compat/next`（不含接口地址的 `ensureAPITrailingSlash`） | `web/` 的源码和 JSON | 3（Task 4、6 扩大） |

`next/script`、`next/image` 已由 P1 的 `next-script-image`、`next-shim-dead-files` 看住。

### 2.9 保留行为的核对（M1 设计 7.5 的 P4 一行、4.2）

| 手段 | 核对 | 落点 |
|---|---|---|
| **进仓库**的 `app/routes/navigation.test.ts` | 2.6 的 5 个测试；Task 5 用 `t5/mutate.mjs` 改回两个旧目标，测试失败并列出它们 | Task 5、6 |
| 构建与静态检查 | `check:types`、`make knip`、`make build-web`、守卫的 3 条规则；Task 2 之后 `make web-dev` 首页没有 "process is not defined" | 每个 Task |
| 临时核对脚本（控制者在 Task 3、4、6 之后写和跑，plan 的"控制者的浏览器核对"一节） | 4.2 的"运行时核对"：几种登录状态的跳转、`next_path`、后退和替换、带和不带结尾 `/` 的当前项、查询参数和 `#` 片段、控制台没有 "navigate() 应在 useEffect 中调用" 的警告；旧地址落到"页面不存在"；重跑 P2 的 3 个、P3 的 2 个核对脚本（地址不再带结尾 `/`），并在基线的构建上做反向对照 | P4 review 附录 |

---

## 3. 与上级设计的差异和补充

每一条都已按"能自己定的就自己定"的原则决定，这里列出决定和理由，供控制者复核。

1. **第 3、4 步各拆成两个 Task。** 第 3 步拆成 Task 3（`Link`、hooks、命令面板上下文、`<Navigate>`）和 Task 4（参数守卫），
   第 4 步拆成 Task 5（跳转目标、旧地址重定向、路由匹配测试）和 Task 6（强制结尾 `/`、当前项、垫片的最后部分）。
   理由：Task 3、4 各改 139、198 个文件，一个是机械替换、一个是逐处判断，分开评审才看得清；参数类型随 `useParams` 一起换
   才不会中途留下假类型。Task 5 满足 3.12 的"先改入口、同一个提交删重定向"；Task 6 把最后一个垫片文件和强制结尾 `/` 一起删，
   因为当前项的判断要在 `/` 不再自动补上时才能核对。
2. **守卫跳转一律 `replace`。** 基线 6 处里 4 处是 `push`。设计 4.2 要求 `<Navigate replace />`：被守卫挡住的地址不应留在历史里，
   否则后退又会触发同一个跳转。
3. **路由自己的页头从布局取参数（Task 4）。** 设计说"路由组件用 `./+types/*` 的 `params`"。`app/` 下的页头不是路由模块，
   但只挂在各自的布局下，所以由布局取 `Route.ComponentProps` 的参数传入；`core/` 下的共享组件统一用守卫（2.5）。
4. **工作项弹窗在没有工作区的页面不打开（Task 4，行为变化）。** 见 2.5。基线在那里打开的弹窗提交时抛错；让它不打开，
   是把一个坏掉的入口变成没有入口。命令面板在个人设置页是否还应提供"创建工作项"，由 M4 决定（第 7 节）。
5. **修好停用功能页的"去设置里开启"按钮（Task 5，行为变化）。** 它们指向没有路由的地址（Plane 的问题），路由匹配测试
   不允许这样的目标存在；改为各自的功能设置页，这是"落在页面上"的最小改法。
6. **侧边栏用 `useMatch`，不用 `NavLink` 的 `end`（Task 6）。** 设计说"`NavLink` 的 `isActive`、`matchPath`"。实测
   `NavLink` 在 `end` 时不容忍结尾 `/`（2.2 结论 7），而 P4 之前所有的书签和外部链接都带结尾 `/`。只有不需要 `end` 的标签页用
   `NavLink`。
7. **`settings/helper.ts` 保留（Task 6）。** 它按段切分地址（`split("/").filter(Boolean)`、去掉首尾 `/`），取的是移动端页头的
   标题和权限键，不是"哪一项高亮"；对带和不带结尾 `/` 的地址结果相同。换成 `matchPath` 要为每个设置页写一遍模式，没有收益。
8. **路由匹配测试的口径（Task 5）。** 一个 `${…}` 可以是同一段里的任意文本，所以它按每个页面的路径填入（参数段填 `x`，静态段
   填该段的文本）再匹配。这样不会漏掉真实存在的目标，代价是宽松：例如 `` `/${slug}/projects/${id}` `` 能借
   `/x/projects/archives` 通过。更严格的做法要推断每个表达式的取值，已超出"只依赖路由表"的范围（设计 4.2）。
   测试另有两条反向断言，确认"没有页面的地址"确实判为落空。
9. **`frontend-env` 按文件范围排除 Node 端代码，不登记例外（Task 2）。** 设计 7.4 说 Node 端工具合法的 `process.env` 读取
   "按执行环境开精确例外"。`e2e/`、`tools/` 里全是 Node 代码，读取环境变量永远合法；为它们逐个登记 `M9` 例外只是噪声。
   规则的 `files` 只选 `web/`、`turbo.json`、`pnpm-workspace.yaml`，样本里 `e2e/playwright.config.ts`、`tools/keywords.mjs`
   是不命中的样本。`web/` 里的 Node 端文件（`vite.config.ts` 等）在 Task 2 之后没有读取。
10. **`turbo.json` 的 `globalEnv` 留 `NODE_ENV`（Task 2）。** 设计只说删 `VITE_*` 和 `.env*`；`NODE_ENV` 决定 Vite 构建的模式，
    缓存必须按它区分，所以保留；没人读的 `DEV` 删除。
11. **`getFileURL` 保留为 `path || undefined`（Task 1）。** 它不再拼基础地址，但"空路径得到 `undefined`"仍是调用方（头像、
    封面、图标的 `src`）依赖的行为；把它内联到几十个调用方不是 P4 的范围。
12. **`normalizeAPIRequestURL` 的签名变了（Task 1）。** 基础地址参数删除，绝对地址一律原样返回：没有接口地址是绝对的，
    绝对地址只会是签名上传地址。它仍然给相对的 Django 地址补结尾 `/`（设计 4.1：接口地址的 `/` 随各领域退出）。
13. **收掉的死代码**：`tab-navigation-root.tsx` 从不执行的跳转、`getTabUrl` 的 `overview`、`use-cycles-details` 返回值里没人读的
    `router`、命令面板状态菜单没人读的两个 prop、`UserService.currentUserConfig`、设置的 12 个 `highlight`。都是本 Phase 的
    改动让它们露出来或变成孤儿的。
14. **M0 交接的收尾写在 Task 6 的文档里**：`M0-P5-frontend-trim-notes` 剩下的两项（`.env`、深层路径的结尾 `/`）都已完成，
    改为 `closed`；`M0-P6-knip-notes` 的 `.env` 一节已完成，S2 和包名仍在 P5，保持 `open`。

---

## 4. 验收标准

- [ ] 6 个 Task 各一个提交，每个提交结束时 `pnpm exec turbo run check:types` 23 个任务、`make lint-web` 52 个、`make build-web`
      11 个通过，`make knip` 为零。
- [ ] `make lint-web`：`keywords: 45 rules, 4 exceptions, no hits.`；每个包的 oxlint 警告数等于上限（合计 711，不变）；
      `node $P4TMP/alts.mjs M1/P4` 输出 `alternatives or variants without a hit sample: 0`。
- [ ] `make test-web` 16 个任务通过；vitest 66 个测试；除 editor 的 5 行 `prosemirror-codemark` 之外没有 stderr。
- [ ] `web/`、`turbo.json`、`pnpm-workspace.yaml` 里没有 `process.env`、`VITE_*`、`dotenv`；`.env.example` 不存在；
      `make web-dev` 打开首页，控制台没有 "process is not defined"。
- [ ] `app/compat/next/`、`next-*.d.ts`、`use-app-router.tsx`、`nextjs.json` 不存在；没有 `next/link`、`next/navigation`、
      `useAppRouter`、`ensureTrailingSlash`、`compat/next` 的引用。
- [ ] `node $P4TMP/tools/render.mjs .` 输出 `render-time navigations: 0`；`AuthenticationWrapper` 只用 `<Navigate replace />`。
- [ ] 没有新增的非空断言和 `as string`；每个参数守卫在报告里写明种类和挂载证据。
- [ ] `routes/core.ts` 没有旧地址重定向；`app/routes/navigation.test.ts` 的 5 个测试通过，并且改回旧目标时失败。
- [ ] `node $P4TMP/tools/targets.mjs . --summary` 的 `trailing 0`；"当前是哪一项"没有按文本比较整个地址的判断（2.7 的表）。
- [ ] `symref.mjs orphaned 9437a6e`、`keyref.mjs orphaned 9437a6e` 为 0；`headers.sh 9437a6e`、`dangling.mjs 9437a6e` 没有输出；
      `route-exports.mjs 9437a6e HEAD` 只列出删除的文件和 `app/layout.tsx` 的 `clientMiddleware`。
- [ ] 控制者的浏览器核对（plan 最后一节）全部通过，脚本写进 review 附录；S1–S4 通过；持续集成的 `server`、`web`、`e2e` 通过。
- [ ] `docs/v0/frontend-changes.md`、M0 的两份交接同步；M1 设计 12 节 P4 一行更新。

---

## 5. 不在 P4 范围内

- 认证和会话的传输方式、`AuthenticationWrapper` 的重写（M2，设计 4.2："M1 只做让它在原生路由下正确工作的最小改动"）。
- 接口地址的结尾 `/`（`ensureAPITrailingSlash`、`normalizeAPIRequestURL`），随各领域对接新接口时退出（设计 4.1）。
- 基线就存在的、对路由参数的无效 `.toString()`：`app/`、`core/` 里 181 个文件 520 行（基线 199 个文件 575 行，P4 改到的行顺手去掉了）。
  它们对字符串是空操作，不影响类型和行为；统一去掉要再改 181 个文件（第 7 节，交收尾）。
- 两处保留的加载器重定向不带 `#` 片段：工作项的项目内地址 `/projects/:projectId/issues/:issueId` 重定向到 `/browse/…`
  （Plane 的行为，查询参数也不带）；个人主页重定向到第一个标签页（P2，带查询参数）。React Router 给加载器的请求本来就不含片段。
  应用自己的工作项链接大多用 `generateWorkItemLink` 直接生成 `/browse/…`（评论链接的 `#comment-…` 因此保留），只有表格的
  "子工作项"列经过前一个重定向（第 7 节，交 M4）。
- 品牌（P5）；oxlint 警告清零（M2–M8）。

---

## 6. 风险

| 风险 | 应对 |
|---|---|
| 机械替换改错语义（`replace`、`back`、对象简写、依赖数组） | 脚本只认识设计列出的几种用法，其他一律停下；重放与原型一致；`render.mjs` 逐个分类跳转调用 |
| 渲染时跳转换成原生跳转后被 React Router 丢掉 | `<Navigate replace />`；`render.mjs` 为 0；浏览器核对几种登录状态和控制台警告 |
| 参数守卫掩盖真实的缺参，保留功能在某个路由上悄悄不出现 | `mounts.mjs` 为每个改动的共享组件列出挂载路由和缺的参数；只有个人设置页真实缺参，逐个说明（2.5） |
| 删掉强制结尾 `/` 之后，旧书签和外部链接（都带 `/`）上的当前项不对 | 一律用 `matchPath` / `useMatch` / 不带 `end` 的 `NavLink`；浏览器核对每个位置带和不带 `/` 两种地址 |
| 删掉旧地址重定向之后，还有入口依赖它们 | 路由匹配测试覆盖源码里写出的全部路径和导航常量；反向断言；改回旧目标时测试失败 |
| 查询参数和 `#` 片段在跳转中丢失 | 删掉 308 中间件（它丢片段）和 `/sign-in` 重定向（它丢查询）；浏览器核对评论链接 `#comment-…` 的定位 |
| 删 `define` 之后某个包的 `dist/` 仍读 `process.env`，开发服务器报错 | 先删读取、再删 `define`（设计 4.1 的顺序）；守卫看住；`make web-dev` 核对 |
| 路由测试随时间变成快照，或扫描悄悄失效 | 测试读源码和路由表，不存清单；"多于 200 个路径"的断言；两条反向断言 |

---

## 7. 移交事项

### 7.1 交给后续 M（写进对应 M 的 `handoffs/`）

| M | 事项 |
|---|---|
| M2 | - `AuthenticationWrapper` 只做了最小改动：6 处跳转改为 `<Navigate replace />`；令牌管理器重写时一并重做。<br>- `next_path` 只带 `pathname`，不带原地址的查询参数和片段（基线如此）；`isValidURL` 只拒绝 `http`、`https`、`ftp` 开头的地址，`//host` 这样的协议相对地址能通过。M2 应只接受以单个 `/` 开头的站内路径。<br>- 登录、注册表单和退出直接提交到 `/auth/…`（相对地址）。 |
| M4 | - 工作项弹窗在没有工作区的页面（个人设置）不打开；命令面板在那里是否还提供"创建工作项"由 M4 定。<br>- 表格的"子工作项"列跳到 `/projects/:projectId/issues/:id#sub-issues`，经加载器重定向到 `/browse/…` 时片段丢失；没有元素读取 `#sub-issues`，这个片段本来就不起作用。M4 重做工作项地址时一并决定。 |

### 7.2 交给 P5

- 无新增事项。

### 7.3 交给 M1 收尾

- 对路由参数的无效 `.toString()`：181 个文件 520 行（第 5 节）。
- 路由匹配测试的宽松之处（第 3 节第 8 条）：如果收尾时要收紧，需要给跳转目标的表达式加上取值推断。
- 构建体积对比（2.2 的表）写进收尾 review。

### 7.4 M0 交接的落点

- `M0-P5-frontend-trim-notes`：剩下的 `.env`、dotenv、`define` 和深层路径的结尾 `/` 都在 P4 完成，改为 `closed`。
- `M0-P6-knip-notes`：`.env` 一节（README 的两处说明、S2 的同源断言保留）在 P4 完成；S2 与包名在 P5，保持 `open`。
