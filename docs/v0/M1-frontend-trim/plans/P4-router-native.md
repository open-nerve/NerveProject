# M1/P4 去掉 Next.js 兼容层 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 前端不再有环境变量（各包不读取、删除 `define`、dotenv 和 `.env.example`）；`next/link`、`next/navigation`、`useAppRouter` 全部改为 React Router 的原生写法，路由参数按真实的 `string | undefined` 处理，渲染时跳转改为 `<Navigate replace />`；删除 Plane 的 10 条旧地址重定向，先改掉依赖它们的入口；去掉强制结尾 `/`，"当前是哪一项"一律用 React Router 匹配；加一个进仓库的路由匹配测试。

**Architecture:** 6 个 Task，每个一个提交，顺序是 M1 设计 9 节 P4 的四步：Task 1 各包的环境变量读取 → Task 2 `define` 和 dotenv → Task 3 `Link`、hooks、命令面板上下文和 `<Navigate>` → Task 4 参数守卫 → Task 5 跳转目标、旧地址重定向、路由匹配测试 → Task 6 强制结尾 `/`、当前项、垫片的最后部分。机械替换由 `$P4TMP` 下的一次性 ts-morph 脚本完成（不进仓库），逐处判断的部分每处都有挂载证据。共享文件（`app/routes/core.ts`、`tools/keywords.json`、`@plane/constants` 的导航常量、命令面板的上下文类型）由当前 Task 一次改完，不并行。

**Tech Stack:** Node 24、pnpm 11.10.0、turbo 2.10.11、oxlint 1.51.0、oxfmt 0.35.0、knip 6.37.0、vitest 4.1.11、React Router 8.3.0、Vite 8.0.16、TypeScript 5.8.3、ts-morph 27（只在 `$P4TMP`）。

**Spec:** `docs/v0/M1-frontend-trim/specs/P4-router-native.md`（上级：`docs/v0/M1-frontend-trim/M1-design.md`）

**原型：** `$P4TMP/proto`，分支 `worktree-m1-p4-router-native`，基点 `9437a6e`。每个 Task 的原型提交：
T1 `479609f`、T2 `6fad945`、T3 `2292eea`、T4 `e168fdc`、T5 `a3a957b`、T6 `270f8e6`。本计划的每个数字都注明来自哪个原型提交。

---

## Global Constraints

> 执行前先读本计划"## Tasks"之前的"控制者评审补充"：它新增了 Task 7，修改了 Task 1、3 的几处做法和浏览器核对的时机，与本节冲突时以它为准。

P3 计划的 Global Constraints 和"控制者评审补充"在本 Phase 继续有效，下面是它们在 P4 的写法；与 P3 冲突时以本节为准。

### 命令与环境

- **所有命令在 worktree 的根目录执行**，不要 `cd`。要在某个包里执行时用 `pnpm --dir <目录>` 或 `pnpm --filter <包名>`。
  开始之前执行一次 `pnpm install --frozen-lockfile`；不做任何全局安装。
- **`$P4TMP`** 是
  `/private/tmp/claude-501/-Users-xiaoruan-project-nerve-project/99d2bc1d-fdaf-4b92-a590-29b89514572b/scratchpad/nerve-p4`。
  原型留下的脚本都在那里；不要用裸的 `/tmp`。每个 Task 开始时核对一次 `ls $P4TMP/t1/services.mjs $P4TMP/node_modules/ts-morph`；
  不在时从备份恢复：`mkdir -p $P4TMP && tar -xf .superpowers/sdd/P4-router-native/p4tmp.tar -C $P4TMP`，再
  `pnpm --dir $P4TMP install`（只装 `ts-morph`，`$P4TMP/package.json`）。
- **基线副本** `$P4TMP/base`：`lintdiff.sh` 拿它比较新增的 oxlint 警告，Task 1 用它测构建体积的基线。没有时
  `git clone -q . $P4TMP/base`、`git -C $P4TMP/base checkout -q 9437a6e`（两次调用），再 `pnpm --dir $P4TMP/base install --frozen-lockfile`。
- **原型的提交**：先取进本仓库（只需一次，不建分支、不改引用）：
  `git fetch .superpowers/sdd/P4-router-native/proto.bundle worktree-m1-p4-router-native`。
  之后 `git show <原型提交> -- <路径>` 看某个文件的结果，`git diff --cached --stat <原型提交>` 在提交前比较整棵树。
- **一个 Bash 调用只跑一个 git 命令**：不把多个 git 命令、`cd` 和 git 串在一起，不在 git 命令里用花括号展开，不写调用 git 的
  shell 循环；多步逻辑写成 `$P4TMP` 下的脚本。不用 `git stash`、`git clean`、`git reset --hard`；不做浅拉取；不碰 `plane/`、
  `refer/`；不停、不改 Docker 容器，不运行 `make dev-db-down` / `make dev-db-reset`；不运行 `corepack enable`。
- **`$P4TMP/notes/per-task.mjs`、`orphans-at.mjs` 会在给定的克隆里 `git checkout`**，只在 `$P4TMP/proto` 上用，不要对 worktree 用。
- **生成的文件不手写、不手改**：`pnpm-lock.yaml` 只由 `pnpm install` 改写（Task 1、2、5 会改它）；`web/apps/web/.react-router/`
  由 `react-router typegen` 生成。改过 `package.json` 之后，第一条 `pnpm exec` 会多打印几行依赖核对，预期输出省略它们。
- **文件内容用编辑工具写**，不用带反引号的 heredoc。

### 沿用的裁定（P2、P3 评审，P3 计划的控制者评审补充）

1. **原型提交是对照答案，不能整体照搬。** 按 Task 的步骤一步步做，每一步都跑写明的命令，把实测输出和预期并排写进报告。
   - 可以用 `git show <原型提交> -- <路径>` 看一个文件的结果，可以运行 `$P4TMP/t<N>/` 的脚本（精确替换，原文不在就失败，
     什么都不写）。条件是：跑完一个脚本或照原型改完一个文件，都要读它的 diff，确认每一处改动都对应本 Task 的一条说明或一个
     类型错误。
   - **禁止以提交、目录或文件列表为单位取原型的内容**：`git checkout <原型提交> -- <目录或多个文件>`、`git cherry-pick`、
     `git read-tree`、`git diff … | git apply` 都不许用。
   - 与原型的差异可以有，每一处都写明原因；原型里你认为错的地方，照你认为对的做，并写进报告。
2. **基点是本分支上一个 Task 的实际提交**（Task 1 是 `9437a6e`）；`<基点>` 出现在 `symref.mjs orphaned`、`infile-orphans.mjs`、
   `dangling.mjs` 等命令里。对**整个 Phase 基线**的核对（`headers.sh 9437a6e`、`keyref.mjs orphaned 9437a6e`、
   `route-exports.mjs 9437a6e HEAD`）仍然用 `9437a6e`。
3. **数字以实测为准。** lint 上限只降不升；P4 预计不改任何上限（合计 711），实测低于 711 时说明多删了什么，高于时先找原因。
   `check:types` 23 个任务、`make lint-web` 52 个、`make build-web` 11 个；`make test-web` Task 1–4 是 15 个、Task 5 起 16 个。
4. **删除，不隐藏；退化结构收掉；孤儿由造成它的 Task 删**（包括同一文件内的使用者、对象成员和 prop）。不用非空断言、
   `as string`，不为了编译把参数改成可选。提交前在本 Task 新增的行里查退化结构：
   `git diff --cached -U0 | grep -E '^\+' | grep -F '={"'` 和 `git diff --cached -U0 | grep -E '^\+' | grep -F '${"'` 都没有输出；
   新增行里没有插值的模板字面量逐个看。原型整个 Phase 只有 3 处，都不是退化结构：Task 1 的停用账户提示（与同一张错误表的写法一致，
   Task 1 Step 5）；Task 3 的 codemod 把 `href=` 换成 `to=` 时原样带出的 `authentication.helper.tsx` 两个错误链接里的 `` : `` ``
   （P3 修复轮写的，不是本 Phase 造成的）。
5. **knip 是门禁**，每个 Task 结束时 `make knip` 为零。
6. **测试输出要干净**：`bash $P4TMP/tests.sh` 除 editor 包的 5 行 `prosemirror-codemark` 之外没有 `!` 开头的行。声明环境可以
   （Task 5 的 `vitest.config.ts`），过滤日志不行。
7. **守卫**：每条规则的每个顶层分支和 `(?:a|b)` 的每一支都有真实代码里的命中样本（`node $P4TMP/alts.mjs M1/P4` 输出
   `alternatives or variants without a hit sample: 0`）；本 Phase 不登记例外，出现命中就删掉它。
8. **报告放在最后一条回复里**（子 agent 不能写报告文件）：提交哈希；每一步的命令、实测输出和预期；与原型的差异及原因；
   下一节的风险点；没做到的事和疑问。状态写 `DONE`、`DONE_WITH_CONCERNS` 或 `BLOCKED`。

### 每个 Task 的固定节奏

1. **改代码**：按 Task 的步骤运行脚本或手改，读 diff。
2. **类型检查**：`pnpm exec turbo run check:types` → `Tasks:    23 successful, 23 total`。报到没动过的行上时（常见 TS2339），
   先 `rm -f web/apps/web/.turbo/tsconfig.tsbuildinfo web/packages/*/.turbo/tsconfig.tsbuildinfo` 再跑一次。
3. **孤儿核对**（`<基点>` 见裁定 2）：
   - `node $P4TMP/infile-orphans.mjs <基点>`：列出的每一行都必须以 `defined now: 0)` 结尾（符号随文件删了）；否则删掉它。
   - `node $P4TMP/symref.mjs orphaned <基点>` → `exports used elsewhere at <基点> and not now: 0`。
   - `node $P4TMP/dangling.mjs <基点>` → `new dangling import comments: 0`（删掉导入后留下的 `// hooks` 这类分组注释）。
   - `node $P4TMP/keyref.mjs orphaned 9437a6e` → `keys referenced at 9437a6e and not now: 0`；`bash $P4TMP/headers.sh 9437a6e` 没有输出。
4. **守卫**：加规则的 Task 用 `node $P4TMP/kw.mjs add-rules|replace-rules $P4TMP/t<N>/rules.json`，然后
   `pnpm exec oxfmt tools`、`node tools/keywords.mjs`、`node $P4TMP/alts.mjs M1/P4`。
5. **lint**：`bash $P4TMP/lintdiff.sh $P4TMP/base` 列出比基线多出来的警告，预期见各 Task；`bash $P4TMP/caps.sh | tail -1` →
   `caps total: 711`。
6. **门禁**：`make lint-web`、`make test-web`（加 `bash $P4TMP/tests.sh`）、`make build-web`、`make knip` 都通过。
7. **与原型对比**：`git add -A`，`git diff --cached --stat <原型提交>` 为空或每处差异都有说明。
8. **提交**：一个 Task 一个提交，提交信息用英文（原型的在 `$P4TMP/t<N>/msg.txt`，可以照用或改写），最后一行是
   `Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>`。提交前 `git status --short` 只能有本 Task 的改动；
   **出现 `?? .superpowers/` 就停下来报告**。

### 风险点（每个 Task 报告必答）

1. **渲染时不跳转**：`node $P4TMP/tools/render.mjs .` 的最后一行。Task 1–2 是 `render-time navigations: 6`（都在
   `AuthenticationWrapper`，Task 3 改）；Task 3 起是 `render-time navigations: 0`。报告写出本 Task 新增或改动的每个跳转调用在哪类
   函数里（事件、effect、回调）。
2. **查询参数和 `#` 片段不丢失**：本 Task 改动的每个带 `?` 或 `#` 的跳转目标逐个列出，写明改后是否原样带上；不新增 `redirect()`、
   中间件或加载器重定向（`git diff --cached | grep -E '^\+.*(redirect\(|Middleware)'` 没有输出）。
3. **参数守卫不掩盖真实的缺参**（Task 4 起）：每个守卫写明种类（spec 2.5 的表）和挂载证据：
   `node $P4TMP/tools/mounts.mjs web/apps/web <文件相对 web/apps/web 的路径>` 列出组件挂载的路由和缺参数的路由。
   缺参数的路由上，改后的行为必须与基线相同，或者基线在那里本来就会出错（写出出错的那一行）。
4. **路由模块的约定导出不动**：`node $P4TMP/notes/route-exports.mjs <基点> HEAD`（提交之后跑）只列出本 Task 删除的文件；
   Task 6 另有 `app/layout.tsx	clientMiddleware,default -> default`。`default`、`meta`、`links`、`loader`、`clientLoader`、
   `action`、`clientAction`、`ErrorBoundary`、`HydrateFallback`、`handle`、`middleware`、`clientMiddleware` 以外的导出不在此列。
5. **没有新的共享或模块级可变状态**：`git diff --cached -U0 -- web | grep -E '^\+(export )?(let|var|const|class) '` 列出的每个新增
   顶层声明写明它是组件、函数还是只读数据；不把某个组件实例自己的状态搬到模块级（P2 的 `UniqueID` 回归）。
6. **当前项对带和不带结尾 `/` 的地址结果相同**（Task 6）：只用 `useMatch`、`matchPath`、不带 `end` 的 `NavLink`；
   `NavLink` 的 `end` 按文本比较地址，不能用（spec 第 3 节第 6 条）。

### 一次性脚本

都在 `$P4TMP`（备份 `.superpowers/sdd/P4-router-native/p4tmp.tar`），从仓库根目录运行，`<root>` 写 `.`。P3 的同名脚本
（`symref`、`infile-orphans`、`keyref`、`keycount`、`headers`、`kw`、`alts`、`caps`、`lintdiff`、`stats`）照搬，`kw.mjs` 加了
`replace-rules`；`lock-diff`、`web-size`、`pkg` 与 P2 相同。

| 脚本 | 用途 |
|---|---|
| `gates.sh <tag> [gate…]` | 五个门禁，每个一行摘要，日志在 `$P4TMP/logs/<tag>-<gate>.log` |
| `tests.sh` | 每个带 `test` 脚本的包的 vitest 数量，以及记录在案之外的 stderr（`!` 开头） |
| `caps.sh`、`lintdiff.sh <基线副本>` | oxlint 上限；比基线多出来的警告（规则 + 文件 + 消息，不带行号） |
| `symref.mjs orphaned <rev>`、`infile-orphans.mjs <rev>`、`dangling.mjs <rev>`、`headers.sh <rev>` | 孤儿、同文件孤儿、悬空的导入分组注释、许可证头 |
| `keyref.mjs orphaned <rev>`、`keycount.mjs` | 文案键（严格方法） |
| `kw.mjs add-rules \| replace-rules \| set-phase`、`alts.mjs <phase>` | 改 `tools/keywords.json`；规则分支的命中样本 |
| `lock-diff.mjs [<旧> <新>]`、`web-size.mjs`、`stats.sh <sha>…` | 锁文件除删除以外的变化；构建体积；提交的 shortstat 和 D/M/R/A |
| `devcheck.mjs [url]` | `make web-dev` 运行时打开首页，打印控制台，统计 "process is not defined" |
| `tools/targets.mjs <root> [--summary\|--trailing]` | 所有内部跳转目标表达式（`Link`、`NavLink`、`Navigate`、`navigate`、`redirect`、`href:` 等），以 `/` 结尾的标 `TRAILING` |
| `tools/render.mjs <root>` | 每个跳转变量的调用按执行时机分类，列出渲染时执行的 |
| `tools/pathlits.mjs <root> [--bad]` | 路径字面量（`${…}` 当作 `x`）用 `matchRoutes` 匹配真实路由表，`NONE` 是落不到页面的 |
| `tools/slashes.mjs <root>`、`tools/allpaths.mjs <root>` | 以 `/` 结尾的路径字面量；全部路径字面量 |
| `tools/mounts.mjs <web 应用目录> <文件>… [--routes]` | 组件挂载在哪些路由、缺哪些参数、挂载链 |
| `tools/inventory.mjs <root>`、`tools/ctx.mjs <tsc 日志> <正则> [n]` | 垫片使用者清单；类型错误附近的代码 |
| `notes/route-exports.mjs <base> <rev>` | `app/` 下路由模块约定导出的变化 |
| `notes/per-task.mjs`、`notes/orphans-at.mjs`、`notes/rules-at.mjs` | 在原型各提交上重算 spec 2.2 的中间值（只对原型克隆用） |
| `t1/services.mjs` | Task 1：service 去掉 `super(API_BASE_URL)` |
| `t3/codemod.mjs <root>` | Task 3：`Link`、`usePathname`、`useSearchParams`、`useRouter`/`useAppRouter` 的机械替换 |
| `t4/imports.mjs`、`t4/headers.mjs`、`t4/params.mjs <root>` | Task 4：`useParams` 的导入；页头取布局的参数；共享组件的守卫 |
| `t5/edits.mjs <root>`、`t5/mutate.mjs <root> break\|restore` | Task 5：跳转目标和旧地址重定向；把两个旧目标改回去、再恢复 |
| `t6/edits.mjs`、`t6/slashes.mjs`、`t6/docs.mjs <root>` | Task 6：强制结尾 `/` 和当前项；去掉字面量的结尾 `/`；文档 |
| `t<N>/rules.json`、`t<N>/msg.txt` | 守卫规则；原型的提交信息 |

### 原型的教训（执行前必读）

1. **`URL.canParse` 比 Vite 的默认构建目标新**（Task 1）：判断绝对地址用协议的正则；`try { new URL() }` 的写法会被 oxlint 的
   `no-new` 拦下（services 包的上限是 0）。
2. **缩进变了的 JSX 会让 oxfmt 把 `prop="x"` 写成 `prop={"x"}`**（Task 4 的 `extended-project-sidebar.tsx`）：`params.mjs` 已经处理；
   第 4 条裁定的 grep 守住。
3. **`NavLink` 的 `end` 不容忍结尾 `/`**（Task 6）：React Router 8.3 `lib/dom/lib.js` 第 387–388 行按文本比较；原型第一版用了它，
   已改为 `useMatch`。
4. **vitest 找到 `vite.config.ts` 就加载 React Router 的构建插件**（Task 5）：web 应用有自己的 `vitest.config.ts`，测试不加载
   应用的构建配置。（执行时更正：原型记的"否则失败""否则 knip 报 `navigation.test.ts` 未使用"在 vitest 4.1.11、React Router 8.3.0、
   knip 6.37.0 上都不成立，没有它测试和 knip 也通过；文件为隔离构建插件而保留。）
5. **路由表的初始值是 `satisfies` 表达式**：读 `routes/core.ts` 的脚本先解开它（`mounts.mjs`、`pathlits.mjs` 已处理）。
6. **页面没有参数时 `return null`，它上面定义的处理函数仍要各自守卫**：闭包不会随后面的 `return` 收窄类型（Task 4 的收藏菜单）。
7. **`lintdiff.sh` 按消息文本比较**：Task 6 改了 `project/root.tsx` 里 `isArchived` 的定义位置，两条 `exhaustive-deps` 的缺失依赖
   换了顺序，条数不变，不是新问题。

### 文件结构

| 位置 | 改动 | Task |
|---|---|---|
| `web/packages/constants/src/endpoints.ts`、各 service、`@plane/services` 的 `APIService` | 删除基础地址 | 1 |
| `web/packages/i18n`（`use-translation.ts`、`package.json`、`tsconfig.json`） | `import.meta.env.DEV` | 1 |
| `web/apps/web/vite.config.ts` | 删 dotenv、`define`（T2）；删别名（T3、T4） | 2, 3, 4 |
| `turbo.json`、`pnpm-workspace.yaml`、`web/apps/web/.env.example`、`README.md` | 环境变量 | 2 |
| `tools/keywords.json` | 新增 3 条规则，顶层 `phase` 改为 `M1/P4`（T2）；T4、T6 扩大两条规则 | 2, 3, 4, 6 |
| `web/apps/web/app/compat/next/*`、`app/types/next-*.d.ts`、`core/hooks/use-app-router.tsx` | 删除 | 3, 4, 6 |
| `web/apps/web/core/lib/wrappers/authentication-wrapper.tsx` | `<Navigate replace />` | 3 |
| `web/apps/web/core/components/power-k/` | 上下文 `navigate`（T3）、`Params`（T4）、"账户设置"（T5） | 3, 4, 5 |
| `web/apps/web/app/(all)/[workspaceSlug]/(projects)/**/{layout,header,mobile-header}.tsx` | 页头取布局的参数 | 4 |
| `web/apps/web/app/routes/core.ts`、`app/routes/redirects/core/*` | 删 10 条旧地址重定向 | 5 |
| `web/apps/web/app/routes/navigation.test.ts`、`vitest.config.ts`、`package.json` | 新增路由匹配测试 | 5, 6 |
| `web/apps/web/app/layout.tsx` | 删强制结尾 `/` 的中间件 | 6 |
| `web/packages/constants/src/{workspace,profile}.ts`、`settings/*.ts`、`types/src/settings.ts` | 侧边栏 `end`，删 `highlight`、`selected` | 6 |
| `docs/v0/frontend-changes.md`、`docs/v0/M1-frontend-trim/handoffs/M0-P{5,6}-*.md` | 记录本 Phase | 6 |

---

## 控制者评审补充（执行前必读）

控制者在执行前复核了 spec 和本计划（原型 `479609f..270f8e6` 的数字、每个 Task 的步骤、浏览器核对的分组）。本节与上文冲突时以本节为准。

### spec 第 3 节的裁定

- 第 1–14 条全部采纳。需要确认的几条：
  - 第 2 条（守卫跳转一律 `replace`）：采纳，设计 4.2 本来就是 `<Navigate replace />`；
  - 第 4 条（个人设置页不再打开工作项弹窗）：采纳。基线在那里提交会抛错，改成"没有入口"不损失功能；命令面板在那里是否还提供"创建工作项"交 M4；
  - 第 5 条（停用功能页的按钮改到各自的功能设置页）：采纳；
  - 第 6、7 条：采纳；
  - 第 8 条（路由匹配测试的宽松）：P4 采纳。两条反向断言和"改坏再恢复"的步骤是它的下限；是否收紧交收尾（spec 7.3）；
  - 第 9 条（`frontend-env` 按文件范围）：采纳。`e2e/`、`tools/` 全是 Node 端代码，按目录排除就是设计 7.4 "按执行环境"的精确写法，`web/` 下的 Node 端文件仍在规则范围内。
- 旧地址的书签和外部链接在 P4 之后落到"页面不存在"：这是设计 3.12 的裁定，确认。

### 对计划的修正

1. **对路由参数的空转换 `.toString()` 在 P4 删，新增 Task 7。** 这推翻 spec 第 5 节和 7.3 的第一条，不交收尾。
   - 理由：`x.toString()` 是 Next.js 的 `useParams` 返回 `string | string[]` 时留下的写法，属于 P4 要去掉的兼容层习惯。Task 4 之后参数有了真实的类型，才能用类型判断哪些调用是空操作。
   - 位置：Task 6 之后单独一个提交，放在最后，这样 Task 1–6 仍能与原型逐个对比。
   - 做法：一次性脚本 `$P4TMP/t7/tostring.mjs`（ts-morph 带类型检查器，不进仓库）只删两种调用：
     - 接收者的类型是 `string`、没有实参的 `.toString()`；
     - 接收者是 `string | undefined` 的 `?.toString()`。

     `any`、`unknown`、数字、对象和其他联合类型一律不动，列出来。改完 `pnpm exec oxfmt web/apps/web`。
   - 核对：
     - 用一个脚本确认 `git diff` 里只有"删掉 `.toString()`"这一种变化（oxfmt 因此重排的行除外）；
     - 门禁同其他 Task；
     - 报告删掉的行数和文件数（spec 按路由参数数的是 520 行、181 个文件，本 Task 按类型数，以实测为准），留下的每一处 `.toString()` 按接收者类型分组。
   - 风险点：第 1、4、5 条照答；第 3 条改为"没有改变任何表达式的值"。
   - 浏览器核对的 C 组改在 Task 7 之后跑，因为它改了一百多个文件里读参数的代码。
2. **`next_path` 用 `@plane/utils` 的 `isValidNextPath`（Task 3）。**
   - 问题：`AuthenticationWrapper` 自己的 `isValidURL` 只拒绝 `http`、`https`、`ftp` 开头的地址，`//host`（协议相对地址）、`javascript:`、反斜杠开头的地址都能通过。`@plane/utils` 的 `url.ts` 早就有严格的 `isValidNextPath`（只接受以单个 `/` 开头的站内路径），导出了但没有人用。
   - 做法：Task 3 让 `getWorkspaceRedirectionUrl` 改用它，删掉本地的 `isValidURL`；给 `isValidNextPath` 加单元测试，覆盖它文档注释里的 9 个例子（放在 `web/packages/utils` 现有的测试旁边）。
   - 这是开放重定向的根因修法，不留给 M2。spec 7.1 给 M2 的那一条随之改为"`next_path` 只带 `pathname`，不带查询参数和片段"。
   - 浏览器核对 A 组第 3 项加两种：`next_path=//example.com/x`、`next_path=javascript:alert(1)` 都被忽略，落在 `/probe-ws`。
3. **没有插值的模板字面量**（P3 修复轮的规则）：
   - Task 1：停用账户的提示写成普通字符串 `"User account deactivated. Please contact your administrator."`。这一行原来插入 `SUPPORT_EMAIL`，去掉插值后留下的外壳正是 P3 定下要收掉的形式；同一张表里其余项是基点代码，不动。
   - Task 3：codemod 把 `authentication.helper.tsx` 两个错误链接的 `href=` 换成 `to=` 时，同一行三元式里的 `` `` `` 写成 `""`。这是 P3 留下、本 Phase 改到的行。
   - 固定节奏第 4 条的检查照旧，原型记下的 3 处因此都不再存在。
4. **执行方式沿用 P3。**
   - 实现者不指定模型（继承 Opus 5.5），各 Task 评审用 sonnet，整分支评审用 opus。
   - Task 3、4 各改一百多个文件，评审包按"机械替换 / 逐处判断"拆成两份。
   - 数字以实测为准：utils 的测试数随第 2 条增加；Task 7 之后 spec 2.2 的数字由整分支评审前的核对重测。

---

## Tasks

### Task 1: 各包不再读取环境变量

M1 设计 9 节 P4 第 1 步、4.1。spec 2.3。原型提交 `479609f`（49 个文件：删 2、改 47，+35 / −291）。

**Files**

- 删除：`web/packages/constants/src/endpoints.ts`、`web/apps/web/app/(all)/layout.preload.tsx`
- 修改：`t1/services.mjs` 改的 33 个 service（web 的 `core/services/**` 32 个、`@plane/services` 的 `developer/api-token.service.ts`）；
  `web/apps/web/core/services/{api,auth,user}.service.ts`、`core/store/user/index.ts`、`core/components/account/auth-forms/password.tsx`、
  `helpers/authentication.helper.tsx`、`app/(all)/layout.tsx`；`web/packages/constants/src/index.ts`；
  `web/packages/services/src/{api.service.ts,helpers/url.ts,helpers/url.test.ts}`；`web/packages/utils/src/file.ts`；
  `web/packages/i18n/{package.json,tsconfig.json,src/hooks/use-translation.ts}`；`pnpm-lock.yaml`

**Steps**

- [ ] **Step 1: 准备脚本、原型和基线**

Run: `ls $P4TMP/t1/services.mjs $P4TMP/node_modules/ts-morph/package.json $P4TMP/base/package.json`（缺的按 Global Constraints 恢复）

Run: `git fetch .superpowers/sdd/P4-router-native/proto.bundle worktree-m1-p4-router-native`，再（另一次调用）`git log --oneline -1 270f8e6`
Expected: `270f8e6 refactor(web): drop the forced trailing slash, and match the current item with React Router`

Run: `pnpm install --frozen-lockfile`，然后 `bash $P4TMP/gates.sh t0`
Expected:

```
types  exit 0   Tasks: 23 successful, 23 total
lint   exit 0  keywords: 42 rules, 4 exceptions, no hits. Tasks: 52 successful, 52 total
test   exit 0   Tasks: 15 successful, 15 total
build  exit 0   Tasks: 11 successful, 11 total
knip   exit 0
```

Run: `node $P4TMP/web-size.mjs`（在 `9437a6e` 的构建上）
Expected: `js: 422 files, 6881280 bytes`、`css: 3 files, 296988 bytes`、`fonts: 25 files, 3755608 bytes`、`other: 112 files, 7910515 bytes`、
`largest chunk: assets/use-parse-editor-content-<hash>.js, 1379329 bytes`、`locale chunks: 34`

Run: `bash $P4TMP/tests.sh; bash $P4TMP/caps.sh | tail -1; node $P4TMP/keycount.mjs`
Expected: constants 11、editor 16（`recorded prosemirror-codemark sourcemap lines: 5`）、i18n 10、services 13、utils 13；`caps total: 711`；`en: 1617 keys`、`zh-CN: 1617 keys`

Run: `node $P4TMP/tools/render.mjs . | tail -7; node $P4TMP/tools/targets.mjs . --summary | tail -1; node $P4TMP/tools/pathlits.mjs . --bad | tail -1`
Expected: `render-time navigations: 6`，其后 6 行都是 `web/apps/web/core/lib/wrappers/authentication-wrapper.tsx:<行>  RENDER (component body)  router.push(…)`
（第 4 行是 `router.replace`）；`total 351, trailing 39`；`path literals 228, not resolving to a page 9`

- [ ] **Step 2: 记下环境变量的读取**

Run: `git grep -n -E "process\.env|(^|[^A-Za-z0-9_])VITE_[A-Z]|dotenv" -- web turbo.json pnpm-workspace.yaml ':!*.md'`
Expected（27 行）：`layout.preload.tsx` 5 行（注释里的 `process.env.VITE_API_BASE_URL`）、`vite.config.ts` 5 行、`endpoints.ts` 6 行、
`i18n/src/hooks/use-translation.ts` 1 行、`turbo.json` 6 行、`.env.example` 2 行、`web/apps/web/package.json` 1 行、
`pnpm-workspace.yaml` 1 行。本 Task 删 `layout.preload.tsx`、`endpoints.ts` 和 i18n 的 12 行；剩下的 15 行（`vite.config.ts`、
`turbo.json`、`.env.example`、两个清单）由 Task 2 删。

- [ ] **Step 3: service 不再传基础地址**

Run: `node $P4TMP/t1/services.mjs | wc -l` → `33`（打印改过的文件）

读 diff：只调用 `super(API_BASE_URL)` 的构造函数整个删掉（连同其后的空行），做更多事的构造函数里改成 `super()`
（`file.service.ts` 等），`API_BASE_URL` 从 `@plane/constants` 的导入里去掉。

- [ ] **Step 4: 手改其余的读取方**（每个文件改完的样子见 `git show 479609f -- <路径>`）

- `web/apps/web/core/services/api.service.ts`、`web/packages/services/src/api.service.ts`：删 `baseURL` 字段、构造参数和 axios 的
  `baseURL`；拦截器调用 `normalizeAPIRequestURL(config.url)`；后者的文档注释改为"requests go to relative paths on this origin"。
- `web/packages/services/src/helpers/url.ts`：`normalizeAPIRequestURL(url)` 只剩一个参数，以协议开头的绝对地址原样返回
  （`if (/^[a-z][a-z\d+\-.]*:/i.test(url)) return url; // an absolute URL starts with its scheme`），其余交给
  `ensureAPITrailingSlash`；`isForeignAbsoluteURL` 删除。`url.test.ts` 的 `normalizeAPIRequestURL` 组：两个"签名地址"合成一个，
  删掉"同源绝对地址加 `/`"，其余去掉第二个实参（5 → 3 个）。
- `core/services/auth.service.ts`：`signOut()` 不带参数，`form.action = "/auth/sign-out/"`；`core/store/user/index.ts` 的调用随之改。
- `core/components/account/auth-forms/password.tsx`：`action={\`/auth/${mode === EAuthModes.SIGN_IN ? "sign-in" : "sign-up"}/\`}`，
  导入去掉 `API_BASE_URL`。
- `core/services/user.service.ts`：删没有调用方的 `currentUserConfig`。
- `helpers/authentication.helper.tsx`：停用账户的 `message: () => \`User account deactivated. Please contact your administrator.\``，
  删 `SUPPORT_EMAIL` 的导入和它上面的 `// plane imports`。
- `web/packages/utils/src/file.ts`：`export const getFileURL = (path: string): string | undefined => path || undefined;`，文档注释写
  "asset paths are relative to this origin … an empty path means there is no file"，删 `@plane/constants` 的导入。
- 删 `web/packages/constants/src/endpoints.ts` 和 `constants/src/index.ts` 里它的一行；删 `app/(all)/layout.preload.tsx`，
  `app/(all)/layout.tsx` 直接 `return <Outlet />;`。
- i18n：`use-translation.ts` 的 `process.env.NODE_ENV !== "production"` → `import.meta.env.DEV`；`tsconfig.json` 的 `compilerOptions`
  加 `"types": ["vite/client"]`；`package.json` 的 `devDependencies` 加 `"vite": "catalog:"`（字母序在 `typescript` 之后）。

Run: `pnpm install`，然后 `node $P4TMP/lock-diff.mjs`
Expected:

```
importers: 0 dependencies removed, 1 added or changed
  + web/packages/i18n devDependencies vite: specifier: 8.0.16  version: 8.0.16(@types/node@22.12.0)(esbuild@0.28.1)(jiti@2.7.0)(terser@5.43.1)(tsx@4.20.6)(yaml@2.8.3)
```

（`specifier: 8.0.16` 是 `pnpm-workspace.yaml` 的 `overrides` 决定的，与 web 应用的 `vite` 一样。）

- [ ] **Step 5: 核对**

Run: 固定节奏第 2 步 → 23/23。

Run: `pnpm --dir web/packages/services exec oxlint .` → `Found 0 warnings and 0 errors.`；
`pnpm --dir web/packages/services exec vitest run` → `Tests  11 passed (11)`

Run: `node $P4TMP/infile-orphans.mjs 9437a6e`
Expected（7 行，都是随文件删掉的符号）：`layout.preload.tsx` 的 `usePreloadResources`、`PreloadResources`；`endpoints.ts` 的
`API_BASE_URL`、`API_BASE_PATH`、`WEB_BASE_URL`、`WEB_BASE_PATH`、`SUPPORT_EMAIL`，每行以 `defined now: 0)` 结尾。

Run: 固定节奏第 3 步其余三条 → 都为 0 / 没有输出。

Run: 第 2 步的 `git grep` → 15 行（`vite.config.ts` 5、`turbo.json` 6、`.env.example` 2、web 的 `package.json` 1、`pnpm-workspace.yaml` 1）。

Run: `bash $P4TMP/lintdiff.sh $P4TMP/base` → 没有输出；`bash $P4TMP/gates.sh t1` → 与 Step 1 相同（`keywords: 42 rules`）；
`bash $P4TMP/tests.sh` → services 11，其余不变。

新增行里没有插值的模板字面量：只有 `authentication.helper.tsx` 的停用账户提示，它与同一张 `errorCodeMessages` 表里的每一项写法相同
（例如"Sign up disabled. Please contact your administrator."），保留。

- [ ] **Step 6: 提交**

`git add -A`，`git diff --cached --stat 479609f` 为空（或说明差异），提交信息见 `$P4TMP/t1/msg.txt`。

---

### Task 2: 删除 `process.env` 注入、dotenv 和 `.env.example`；守卫的 `phase` 改为 `M1/P4`

M1 设计 9 节 P4 第 2 步、4.1、9.7。spec 2.3、2.8。原型提交 `6fad945`（8 个文件：删 1、改 7，+43 / −45）。

**Files**

- 删除：`web/apps/web/.env.example`
- 修改：`web/apps/web/vite.config.ts`、`web/apps/web/package.json`、`pnpm-workspace.yaml`、`pnpm-lock.yaml`、`turbo.json`、`README.md`、
  `tools/keywords.json`

**Steps**

- [ ] **Step 1: 删注入和配置**（见 `git show 6fad945 -- <路径>`）

- `vite.config.ts`：删 `import * as dotenv from "dotenv";`、`dotenv.config(...)`、`viteEnv` 的整段和 `define` 块。`node:path` 的导入
  这时还在用（别名），Task 4 删。
- `git rm -q web/apps/web/.env.example`
- web 的 `package.json` 删开发依赖 `"dotenv": "catalog:"`；`pnpm-workspace.yaml` 的 catalog 删 `"dotenv": "16.4.7"`。
- `turbo.json`：`"globalEnv": ["NODE_ENV"]`；`build` 任务删 `"inputs": ["$TURBO_DEFAULT$", ".env*"]`。
- `README.md`：前端一节"不要建立 `web/apps/web/.env`"一条改为"**没有前端环境变量**：前端与接口同源部署，接口一律用相对路径
  （`/api/…`、`/auth/…`），代码不读取 `process.env` 或 `import.meta.env.VITE_*`，关键词守卫看住这一点。`import.meta.env.DEV`、
  `PROD` 是 Vite 按构建模式给出的常量，不是环境变量。"；端到端测试一节的"同源"一条改为"S2 断言页面的所有请求都发往 nerve 自身
  （见"前端"一节的"没有前端环境变量"一条）；有请求发往别处时 S2 失败，失败信息列出这些请求。"

Run: `pnpm install`，然后 `node $P4TMP/lock-diff.mjs`
Expected:

```
catalogs: 3 lines removed, 0 added
importers: 1 dependencies removed, 0 added or changed
packages: 1 removed, 0 added or changed
snapshots: 1 removed, 0 added or changed
```

Run: Task 1 Step 2 的 `git grep` → 没有输出。

- [ ] **Step 2: 守卫**

Run: `node $P4TMP/kw.mjs add-rules $P4TMP/t2/rules.json`、`node $P4TMP/kw.mjs set-phase M1/P4`、`pnpm exec oxfmt tools`、`node tools/keywords.mjs`
Expected: `keywords: 43 rules, 4 exceptions, no hits.`（没有 `until: M1/P4` 的例外，改 `phase` 不会让例外过期）

Run: `node $P4TMP/alts.mjs M1/P4` → `alternatives or variants without a hit sample: 0`

- [ ] **Step 3: 开发服务器**

在后台运行 `make web-dev`（端口 3000），等到日志出现 `Local:   http://127.0.0.1:3000/`，然后
Run: `node $P4TMP/devcheck.mjs`
Expected: 最后一行 `"process is not defined": 0`（接口没有运行，页面停在实例错误页，控制台有一条 502，这是预期的）。
停掉开发服务器，报告端口，确认 `lsof -nP -iTCP:3000 -sTCP:LISTEN` 没有输出。

- [ ] **Step 4: 核对与提交**

固定节奏第 2、3、5、6 步：23/23；孤儿核对都为 0；lintdiff 没有输出；`bash $P4TMP/gates.sh t2` 的 lint 行是
`keywords: 43 rules, 4 exceptions, no hits.`。`git diff --cached --stat 6fad945` 为空，提交信息见 `$P4TMP/t2/msg.txt`。

---

### Task 3: `Link`、location、search params 和 `navigate`；渲染时跳转改为 `<Navigate>`

M1 设计 9 节 P4 第 3 步的前半、4.2。spec 2.4。原型提交 `2292eea`（139 个文件：删 3、改 136，+492 / −557）。

**Files**

- 删除：`web/apps/web/app/compat/next/link.tsx`、`web/apps/web/app/types/next-link.d.ts`、`web/apps/web/core/hooks/use-app-router.tsx`
- 修改：`t3/codemod.mjs` 改的 129 个文件（清单 `$P4TMP/logs/t3-codemod.txt`）；手改 `app/compat/next/navigation.ts`、
  `app/types/next-navigation.d.ts`、`vite.config.ts`、`core/components/power-k/{core/types.ts,utils/navigation.ts,config/creation/command.ts}`，
  以及 codemod 改过之后再手改的 `core/lib/wrappers/authentication-wrapper.tsx`、`core/components/cycles/active-cycle/use-cycles-details.ts`、
  `app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/issues/(detail)/[issueId]/page.tsx`；`tools/keywords.json`

**Steps**

- [ ] **Step 1: 机械替换**

Run: `node $P4TMP/t3/codemod.mjs . | wc -l` → `129`（与 `$P4TMP/logs/t3-codemod.txt` 相同）

它做的替换（脚本头部有全文）：`next/link` 的默认导入 `Link` → `react-router` 的 `Link`，`<Link href=…>` → `<Link to=…>`；
`const pathname = usePathname()` → `const { pathname } = useLocation()`；`const searchParams = useSearchParams()` →
`const [searchParams] = useSearchParams()`；`const router = useRouter()`（或 `useAppRouter()`）→ `const navigate = useNavigate()`，
`router.push(x)` → `navigate(x)`，`router.replace(x)` → `navigate(x, { replace: true })`，`router.back()` → `navigate(-1)`，
作为回调传出的 `router.back` → `() => navigate(-1)`，依赖数组和对象简写里的 `router` → `navigate`。`useParams` 留在
`next/navigation`（Task 4）。遇到不认识的用法它停下，打印文件和行，什么都不写。

读 diff：抽查每一类替换至少一处。然后
Run: `git grep -n -E "router\.(push|replace|back)|(ctx|context)\.router|useRouter[^P]|useAppRouter|usePathname|next/link" -- web/apps/web ':!web/apps/web/core/hooks/store/use-router-params.ts'`
Expected（12 行，都是 Step 2 要手改或删除的）：`app/compat/next/navigation.ts` 2 行、`app/types/next-link.d.ts` 1 行、
`app/types/next-navigation.d.ts` 2 行、`power-k/config/creation/command.ts:136`、`power-k/core/types.ts` 2 行、
`power-k/utils/navigation.ts:25`、`core/hooks/use-app-router.tsx` 2 行、`vite.config.ts:14`。（`use-router-params.ts` 是 MobX 的路由
store，与垫片无关。）

Run: `node $P4TMP/tools/render.mjs . | tail -7`
Expected: `render-time navigations: 6`，6 行都在 `authentication-wrapper.tsx`（`navigate(currentRedirectRoute)`、`navigate("/onboarding")`、
`` navigate(`/${pathname ? `?next_path=${pathname}` : ``}`) ``、`navigate(currentRedirectRoute, { replace: true })`、
``navigate(`/onboarding`)``、第二个 `next_path`）——codemod 只换了写法，Step 2 改掉它们。

- [ ] **Step 2: 手改**（见 `git show 2292eea -- <路径>`）

- `authentication-wrapper.tsx`：6 处 `navigate(…); return <></>;` 都改为 `return <Navigate to=… replace />;`：
  - 非登录页已登录且完成引导 → `getWorkspaceRedirectionUrl()`；未完成引导 → `"/onboarding"`；
  - 引导页未登录 → `` `/?next_path=${pathname}` ``（原来的 `pathname ? … : \`\`` 收掉，`pathname` 不会为空）；引导页已完成引导 →
    `getWorkspaceRedirectionUrl()`；
  - 需登录页未完成引导 → `"/onboarding"`；未登录 → `` `/?next_path=${pathname}` ``。
  - 导入改为 `import { Navigate, useLocation, useSearchParams } from "react-router";`，不再有 `useNavigate`。
- `use-cycles-details.ts`：返回值里的 `navigate`（原来的对象简写 `router`）没有读取方，删掉。
- `issues/(detail)/[issueId]/page.tsx`：`use-app-router` 的导入没了之后留下的 `// hooks` 注释删掉（`dangling.mjs` 会报它）。
- 命令面板：`core/types.ts` 的 `router: ReturnType<typeof useRouter>` → `navigate: NavigateFunction`（注释 `// Navigation`，
  `import type { NavigateFunction } from "react-router";`）；`utils/navigation.ts` 的 `context.router.push(route)` →
  `context.navigate(route)`；`config/creation/command.ts` 的 `ctx.router.push("/create-workspace")` → `ctx.navigate("/create-workspace")`。
- `git rm -q web/apps/web/app/compat/next/link.tsx web/apps/web/app/types/next-link.d.ts web/apps/web/core/hooks/use-app-router.tsx`；
  `vite.config.ts` 删 `next/link` 的别名；`app/compat/next/navigation.ts` 只剩
  `import { useParams as useParamsRR } from "react-router";` 和 `useParams`；`next-navigation.d.ts` 只剩 `useParams` 的声明。
- `pnpm exec oxfmt web/apps/web`（codemod 合并导入后的格式）。

- [ ] **Step 3: 核对**

固定节奏第 2 步 → 23/23。

Run: `node $P4TMP/infile-orphans.mjs <基点>`
Expected（5 行，都以 `defined now: 0)` 结尾）：`navigation.ts` 和 `next-navigation.d.ts` 各自的 `useRouter`、`usePathname`，
`use-app-router.tsx` 的 `useAppRouter`。其余三条孤儿核对为 0 / 没有输出。

Run: `node $P4TMP/tools/render.mjs . | tail -8`
Expected: `navigate variables: 81`，分类 `27 JSX handler`、`3 callback`、`5 effect`、`49 local helper`、`16 object member / other`、
`3 useCallback`，最后一行 `render-time navigations: 0`（Task 6 删掉 `tab-navigation-root.tsx` 的 effect 之后是 80 个变量、4 个 effect）。

Run: `node $P4TMP/tools/targets.mjs . --summary | tail -1` → `total 347, trailing 39`（结尾 `/` 由 Task 6 去掉；这期间 `app/layout.tsx`
的中间件仍会补 `/`）。

- [ ] **Step 4: 守卫**

Run: `node $P4TMP/kw.mjs add-rules $P4TMP/t3/rules.json`、`pnpm exec oxfmt tools`、`node tools/keywords.mjs`
Expected: `keywords: 45 rules, 4 exceptions, no hits.`（加规则之前在 `6fad945` 上它们会命中 184 行、107 个文件，清单
`$P4TMP/logs/t3-rule-before.txt`）

Run: `node $P4TMP/alts.mjs M1/P4` → 0。

- [ ] **Step 5: 门禁与提交**

lintdiff 没有输出；`bash $P4TMP/gates.sh t3` 全部通过（`keywords: 45 rules`、测试 15 个任务）；`git diff --cached --stat 2292eea` 为空；
提交信息见 `$P4TMP/t3/msg.txt`。

**控制者在本 Task 之后跑浏览器核对的 A 组（见最后一节）。**

---

### Task 4: 路由参数取自 React Router

M1 设计 9 节 P4 第 3 步的后半、4.2。spec 2.5。原型提交 `e168fdc`（198 个文件：删 2、改 196，+492 / −422）。

**Files**

- 删除：`web/apps/web/app/compat/next/navigation.ts`、`web/apps/web/app/types/next-navigation.d.ts`
- 修改：`t4/imports.mjs` 的 178 个文件（清单 `$P4TMP/logs/t4-imports.txt`）；`t4/headers.mjs` 的 24 个 `app/` 文件；
  `t4/params.mjs` 的 38 个文件；`web/apps/web/vite.config.ts`；`tools/keywords.json`（三组有重叠，合计 196）

**Steps**

- [ ] **Step 1: `useParams` 的导入**

Run: `node $P4TMP/t4/imports.mjs . 2>&1 | tail -1` → `changed 178 files`

Run: `git rm -q web/apps/web/app/compat/next/navigation.ts web/apps/web/app/types/next-navigation.d.ts`；
`vite.config.ts` 删整个 `alias` 块（连同 `// Next.js compatibility shims used within web` 注释）和 `import path from "node:path";`。

Run: `pnpm --dir web/apps/web exec react-router typegen`，然后
`pnpm --dir web/apps/web exec tsc --noEmit -p . > $P4TMP/logs/t4-tsc-1.log; grep -c 'error TS' $P4TMP/logs/t4-tsc-1.log`
Expected: `102`；`app/` 12 个文件 39 处、`core/` 31 个文件 63 处（TS18048 39、TS2322 38、TS2345 25）。

- [ ] **Step 2: 路由自己的页头从布局取参数**

Run: `node $P4TMP/t4/headers.mjs . | wc -l` → `24`；`pnpm exec oxfmt web/apps/web/app`

它做的事：12 个布局 `export default function X({ params }: Route.ComponentProps)`，把页头用到的参数作为 prop 传进去
（`import type { Route } from "./+types/layout";` 放在最后一条导入之后）；12 个页头改收 `props: TProps`（`type TProps = { … string }`），
删掉 `useParams`。例子：`cycles/(list)/layout.tsx` 与 `cycles/(list)/header.tsx`（`git show e168fdc -- <路径>`）。

Run: 再跑 Step 1 的 `tsc`（日志写到 `t4-tsc-2.log`）→ `63`，全部在 `core/` 的 31 个文件。

- [ ] **Step 3: 挂载证据（改共享组件之前）**

Run: `grep -oE '^core/[^(]+' $P4TMP/logs/t4-tsc-2.log | sort -u > $P4TMP/logs/t4-errfiles.txt`（31 个文件），然后
`node $P4TMP/tools/mounts.mjs web/apps/web @$P4TMP/logs/t4-errfiles.txt > $P4TMP/logs/t4-mounts.txt`
Expected（与备份里的 `logs/t4-mounts.txt` 相同）：每个读路由参数的组件一行"读哪些参数、挂载在几个路由"，缺参数时再列出路由和
一条挂载链。缺参数的路由只有下面这些，其余组件在每个挂载路由上参数都存在：

| 组件 | 缺的参数和路由 | 改后的行为 |
|---|---|---|
| `MemberOptions`、`ParentIssuesListModal`、`PowerKProjectStatesMenu` | `workspaceSlug`：`/settings/profile/:profileTabId`（经命令面板的工作项弹窗） | `isUserSuspended` 没有工作区时回答 `false`（widen）；父工作项链接的 `workspaceSlug` 本来可选（pass）；状态菜单从不读这两个 prop（delete） |
| `CreateUpdateIssueModalBase`、`IssueFormRoot` | `workspaceSlug`：同上；`projectId`：16 个工作区级路由；`cycleId`、`moduleId`、`workItem`：37 个 | 没有工作区时弹窗不打开（render），表单从弹窗取 `workspaceSlug`；`projectId` 仍按"数据 → 路由 → 编号推出"取；`cycleId`、`moduleId` 可选的判断写成 `!(moduleId && payload.module_ids?.includes(moduleId))`，与基线的真值表相同 |
| `SidebarProjectsListItem` | `projectId`：10 个路由 | 只用来判断"当前项目"，本来可选；`workspaceSlug` 改由渲染它的两个列表传入 |

- [ ] **Step 4: 共享组件按真实情况处理**

Run: `node $P4TMP/t4/params.mjs .`
Expected: 38 行 `<种类>\t<文件>`，stderr `changed 38 files`。种类和文件见 spec 2.5 的表。然后 `pnpm exec oxfmt web/apps/web`。

每一处都要能说出理由，报告里按文件列出"种类 → 理由 → Step 3 的挂载证据"。读 diff 时核对：
- 守卫让后面的判断恒为真时，那个判断一起收掉（`project-setting-label-list.tsx` 的 `onDrop`、收集箱页头第 72 行附近）；
- `x && <C/>` 只用在渲染回调里（hook 之后才能 `return null`）；
- `extended-project-sidebar.tsx` 的列表缩进变了，`projectListType={"JOINED"}` 写回 `projectListType="JOINED"`（原型的教训 2）。

Run: 再跑 `tsc`（`t4-tsc-3.log`）→ `0`；固定节奏第 2 步 → 23/23。

Run: `git diff -U0 HEAD -- web | grep -E '^\+' | grep -cE '( as string)|([A-Za-z0-9_)]![.),;])'` → `0`（没有新增的 `as string` 和非空断言）

- [ ] **Step 5: 守卫**

Run: `node $P4TMP/kw.mjs replace-rules $P4TMP/t4/rules.json`、`pnpm exec oxfmt tools`、`node tools/keywords.mjs`
Expected: `keywords: 45 rules, 4 exceptions, no hits.`（扩大后的两条规则在 `2292eea` 上命中 180 行、180 个文件，清单
`$P4TMP/logs/t4-rule-before.txt`）；`alts.mjs M1/P4` → 0。

- [ ] **Step 6: 核对、门禁与提交**

孤儿核对：`infile-orphans.mjs <基点>` → `…: 0`，其余为 0 / 没有输出；lintdiff 没有输出；`render.mjs` → 0；
`bash $P4TMP/gates.sh t4` 全部通过；`git diff --cached --stat e168fdc` 为空；提交信息见 `$P4TMP/t4/msg.txt`。

**控制者在本 Task 之后跑浏览器核对的 B 组。**

---

### Task 5: 每个内部跳转落在页面上；删除旧地址重定向；路由匹配测试

M1 设计 9 节 P4 第 4 步的前半、3.12、4.2、7.5。spec 2.6。原型提交 `a3a957b`（21 个文件：删 10、新增 2、改 9，+195 / −208）。

**Files**

- 删除：`web/apps/web/app/routes/redirects/core/{accounts-signup,api-tokens,inbox,login,profile-settings,project-settings,register,sign-in,signin,workspace-account-settings}.tsx`
- 新增：`web/apps/web/app/routes/navigation.test.ts`、`web/apps/web/vitest.config.ts`
- 修改：`web/apps/web/app/routes/core.ts`、`web/apps/web/package.json`、`pnpm-lock.yaml`、
  `core/components/power-k/config/navigation/commands.ts`、`core/components/navigation/tab-navigation-utils.ts`、
  `app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/{intake/page.tsx,cycles/(list)/page.tsx,modules/(list)/page.tsx,views/(list)/page.tsx}`

**Steps**

- [ ] **Step 1: 找出落不到页面的目标**

Run: `node $P4TMP/tools/pathlits.mjs . --bad`
Expected（9 行 + 汇总）：4 个 `/${workspaceSlug}/settings/projects/${projectId}/features`（收集箱、迭代、模块、视图页）；
`tab-navigation-root.tsx:125`、`tab-navigation-utils.ts:22` 的 `/${workspaceSlug}/projects/${projectId}`；
`archive-tabs-list.tsx`、`sub-issue-column.tsx`、`favorites/…/common/helper.tsx` 三个只作前缀的字面量；
`path literals 228, not resolving to a page 9`。命令面板的"账户设置"是 `handlePowerKNavigate` 的数组，这个工具看不到，测试看得到。

- [ ] **Step 2: 改目标，删旧地址重定向**

Run: `node $P4TMP/t5/edits.mjs .`
Expected: 打印 8 个写过的文件和 10 行 `deleted web/apps/web/app/routes/redirects/core/<名字>.tsx`。它做的事：
- `commands.ts`：`handlePowerKNavigate(ctx, [ctx.params.workspaceSlug?.toString(), "settings", "account"])` →
  `handlePowerKNavigate(ctx, ["settings", "profile", "general"])`；
- 4 个停用功能页的按钮 → `` `/${workspaceSlug}/settings/projects/${projectId}/features/<intake|cycles|modules|views>` ``；
- `tab-navigation-utils.ts` 删 `overview: \`${baseUrl}/overview\`,`；
- web 的 `package.json` 加脚本 `"test": "vitest run"` 和开发依赖 `"vitest": "catalog:"`；
- `routes/core.ts` 从 `// REDIRECT ROUTES` 的横幅到路由表末尾整段删除（正好 10 个 `route(`），删 10 个模块。

`profile-index.tsx`（P2 的个人主页重定向）保留。

Run: `pnpm install`，`node $P4TMP/lock-diff.mjs`
Expected:

```
importers: 0 dependencies removed, 1 added or changed
  + web/apps/web devDependencies vitest: specifier: 'catalog:'  version: 4.1.11(@opentelemetry/api@1.9.1)(@types/node@22.12.0)(@vitest/coverage-v8@4.1.11)(jsdom@30.1.1)(vite@8.0.16(@types/node@22.12.0)(esbuild@0.28.1)(jiti@2.7.0)(terser@5.43.1)(tsx@4.20.6)(yaml@2.8.3))
```

- [ ] **Step 3: 路由匹配测试**

写 `web/apps/web/vitest.config.ts`：

```ts
import { defineConfig } from "vitest/config";

// The tests read the route table and the source, in Node. Without this file vitest would load vite.config.ts,
// whose React Router plugin builds the app and does not run under vitest.
export default defineConfig({ test: { environment: "node" } });
```

写 `web/apps/web/app/routes/navigation.test.ts`（带许可证头；`git show a3a957b -- web/apps/web/app/routes/navigation.test.ts` 是
原型的全文，照它写，逐段读懂）。它的结构：
- `toRouteObjects`：把 `app/routes.ts` 的 `RouteConfigEntry[]` 转成 `matchRoutes` 用的 `RouteObject[]`（`index` → `{ index: true }`）；
- `lands(url)`：`matchRoutes` 的最深一层存在且不是 `"*"`（"页面不存在"）；
- `pagePaths`：每个页面的完整路径；`reaches(target)`：目标里的 `*`（源码的 `${…}`）是同一段里的任意文本，按每个页面的路径填入
  （参数段填 `x`，静态段在匹配时填该段）再 `lands`；
- `targetsIn(file)`：TypeScript 语法树里以 `/` 开头的字符串和模板字面量（`/api/`、`/auth/`、带扩展名的文件、带空白的文本除外；
  导入和导出语句不看），`${prefix}/…` 的前缀取同一文件里以 `/` 开头的常量，另加 `handlePowerKNavigate` 的各段；
- `sourceFiles`：`app/`、`core/`、`helpers/` 下的 `.ts`、`.tsx`（测试和声明文件除外）；
- 4 个测试：扫到的路径多于 200 个；每个路径 `reaches`；导航常量的每一项（侧边栏两组、`WORKSPACE_SETTINGS`、`PROJECT_SETTINGS`、
  `PROFILE_TABS`、`generateWorkItemLink` 的两种）`lands`；`/acme/settings/account` 和 `/*/settings/projects/*/features` 落空。

Run: `pnpm --dir web/apps/web exec vitest run`
Expected: `Test Files  1 passed (1)`、`Tests  4 passed (4)`，没有 stderr。

- [ ] **Step 4: 改坏再恢复，证明测试有用**

Run: `node $P4TMP/t5/mutate.mjs . break` → `break: 2 targets`；`pnpm --dir web/apps/web exec vitest run`
Expected: `× lands every path written in the web app on a page`，差异里列出
`"at": "app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/intake/page.tsx:55"`、`"path": "/*/settings/projects/*/features"`
和 `"at": "core/components/power-k/config/navigation/commands.ts:147"`、`"path": "/*/settings/account"`；`Tests  1 failed | 3 passed (4)`。

Run: `node $P4TMP/t5/mutate.mjs . restore` → `restore: 2 targets`；`git diff --stat -- web/apps/web/core/components/power-k web/apps/web/app` 与
Step 2 之后相同。

- [ ] **Step 5: 核对、门禁与提交**

Run: `node $P4TMP/tools/pathlits.mjs . --bad | tail -1` → `path literals 217, not resolving to a page 5`（剩下的 5 个见 Step 1 的
前缀和 `tab-navigation-root.tsx:125`，后者 Task 6 删）。

knip 必须为零（`vitest.config.ts` 的 `test` 键让它把测试文件算作入口，原型的教训 4）。孤儿核对都为 0 / 没有输出；lintdiff 没有输出；
`bash $P4TMP/gates.sh t5` → 测试 `Tasks: 16 successful, 16 total`；`bash $P4TMP/tests.sh` → `apps/web  Test Files 1 passed (1) | Tests 4 passed (4)`。
`git diff --cached --stat a3a957b` 为空；提交信息见 `$P4TMP/t5/msg.txt`。

---

### Task 6: 去掉强制结尾 `/`，当前项用 React Router 匹配；垫片的最后部分

M1 设计 9 节 P4 第 4 步的后半、4.1、4.2。spec 2.7。原型提交 `270f8e6`（56 个文件：删 2、改 54，+164 / −249）。

**Files**

- 删除：`web/apps/web/app/compat/next/helper.ts`、`web/packages/typescript-config/nextjs.json`
- 修改：`t6/edits.mjs` 的文件（`app/layout.tsx`、spec 2.7 表中的组件、导航常量和类型、两个 `navigation.test.ts`、
  `typescript-config/package.json`、`.oxlintrc.json`、editor 的 `package.json`）；`t6/slashes.mjs` 的 24 个文件；
  `docs/v0/frontend-changes.md`、`docs/v0/M1-frontend-trim/handoffs/M0-P5-frontend-trim-notes.md`、`M0-P6-knip-notes.md`；`tools/keywords.json`

**Steps**

- [ ] **Step 1: 记下以 `/` 结尾的目标**

Run: `node $P4TMP/tools/targets.mjs . --trailing | wc -l` → `35`；`node $P4TMP/tools/slashes.mjs . | tail -1` → `total 57`

- [ ] **Step 2: 强制结尾 `/`、当前项、Next.js 的遗留**

Run: `node $P4TMP/t6/edits.mjs .`
Expected: 打印写过的文件，最后两行 `deleted web/apps/web/app/compat/next/helper.ts`、`deleted web/packages/typescript-config/nextjs.json`。

它做的事（每一处见 spec 2.7 的表，结果见 `git show 270f8e6 -- <路径>`）：
- `app/layout.tsx` 删 `trailingSlashRedirectMiddleware`、`clientMiddleware` 和随之不用的 `redirect`、`ensureTrailingSlash`、`Route` 导入；
- 侧边栏常量：`IWorkspaceSidebarNavigationItem` 的 `highlight` → `/** Current only on this exact address, not below it. */ end?: boolean;`，
  首页 `href: ""` 加 `end: true`，项目加 `end: true`，其余 4 个 `highlight` 删掉，`href` 不带结尾 `/`；
  `sidebar-item.tsx` 在权限判断**之前**算 `itemHref` 和 `const isActive = useMatch({ path: itemHref, end: item.end ?? false }) !== null;`
  （hook 不能在 `return null` 之后），渲染 `Link` + `SidebarNavItem isActive={isActive}`；
- 个人主页标签、归档标签：`NavLink` 的渲染函数 `({ isActive }) => …`；个人主页布局用 `params.profileViewId` 找当前标签，
  `PROFILE_TABS` 删 3 个 `selected`，删掉那条 `TODO` 注释；
- 两个设置侧边栏：`matchPath({ path: \`/:workspaceSlug${item.href}\`, end: item.href === "/settings" }, pathname)`、
  `matchPath({ path: \`/:workspaceSlug/settings/projects/:projectId${item.href}\`, end: item.href === "" }, pathname)`，链接不带结尾 `/`；
  设置的 12 个 `highlight` 和 `types/src/settings.ts` 的两个字段删除；
- `useMatch`：`_sidebar.tsx`、`app-rail-root.tsx`、`top-navigation-root.tsx`（`href` 改为 `` `/${workspaceSlug}/notifications` ``）、
  `use-workspace-paths.ts`（`isProjectsPath = workspaceSlug !== undefined && !isSettingsPath`）、`project/header.tsx`；
  `matchPath`：`project/root.tsx`、`projects-list.tsx` 的 effect、`project-navigation.tsx`、`use-active-tab.ts`；
- `tab-navigation-root.tsx` 删从不执行的"项目根地址跳到默认标签页"的 effect 和随之不用的 `useEffect`、`useNavigate`、`DEFAULT_TAB_KEY`；
- `typescript-config/package.json` 的 `files` 删 `nextjs.json`；`.oxlintrc.json` 删 `".next/**"`；editor 的 `keywords` 删 `"nextjs"`；
- `constants/src/navigation.test.ts`：`href` 以 `/` 开头、`tab.selected` 两条断言删除，测试改名"gives every item a label and at least
  one role"，上面加注释说明落点由 web 的路由测试核对；
- web 的 `navigation.test.ts`：加"writes every in-app path without a trailing slash"（`target.path.length > 1 && endsWith("/")` 的为空），
  常量那一个改名"… on a page, without a trailing slash"并断言没有以 `/` 结尾的项。

Run: `node $P4TMP/t6/slashes.mjs . | tail -1` → `30 literals in 24 files`（Step 1 的 57 个里，26 个随上一步的删除和常量改写消失；
`utils/url.ts` 的 `"//"` 不是路径，脚本不动它）

Run: `node $P4TMP/t6/docs.mjs .` → 打印 `docs/v0/frontend-changes.md` 和两份交接的路径（内容见 spec 2.7、7.4）。

Run: `pnpm exec oxfmt web/apps/web`，并在 `web/packages/constants`、`types`、`utils` 各跑一次 `pnpm --dir <包> exec oxfmt .`

- [ ] **Step 3: 守卫**

Run: `node $P4TMP/kw.mjs replace-rules $P4TMP/t6/rules.json`、`pnpm exec oxfmt tools`、`node tools/keywords.mjs`
Expected: `keywords: 45 rules, 4 exceptions, no hits.`（扩大后的两条规则在 `a3a957b` 上命中 3 行、2 个文件和 2 个文件名）；
`alts.mjs M1/P4` → 0。

- [ ] **Step 4: 核对**

固定节奏第 2 步 → 23/23。

Run: `node $P4TMP/tools/targets.mjs . --summary | tail -1` → `total 343, trailing 0`；`node $P4TMP/tools/slashes.mjs .` → 只有
`web/packages/utils/src/url.ts:299	"//"` 和 `total 1`；`node $P4TMP/tools/pathlits.mjs . --bad | tail -1` →
`path literals 209, not resolving to a page 4`（Step 1 所说的 4 个前缀）。

Run: `pnpm --dir web/apps/web exec vitest run` → `Tests  5 passed (5)`；`t5/mutate.mjs` 的 break / restore 再做一次，结果与 Task 5 Step 4 相同
（`Tests  1 failed | 4 passed (5)`）。

Run: `grep -rn -E "NavLink[^>]*\bend\b" web/apps/web/app web/apps/web/core` → 没有输出（风险点 6）。

Run: `node $P4TMP/infile-orphans.mjs <基点>` → 1 行 `app/compat/next/helper.ts	ensureTrailingSlash	(base uses: 2, defined now: 0)`；
其余孤儿核对为 0 / 没有输出。

Run: `bash $P4TMP/lintdiff.sh $P4TMP/base`
Expected（原型的教训 7，不是新问题）：

```
== web/apps/web
eslint-plugin-react-hooks(exhaustive-deps)	core/components/project/root.tsx	React Hook useCallback has missing dependencies: 'updateDisplayFilters', and 'isArchived'
eslint-plugin-react-hooks(exhaustive-deps)	core/components/project/root.tsx	React Hook useEffect has missing dependencies: 'workspaceSlug', 'updateDisplayFilters', and 'isArchived'
```

- [ ] **Step 5: 门禁、整个 Phase 的核对与提交**

Run: `bash $P4TMP/gates.sh t6` 全部通过；`bash $P4TMP/tests.sh` → web 5、constants 11、editor 16（5 行记录在案）、i18n 10、services 11、utils 13；
`bash $P4TMP/caps.sh | tail -1` → `caps total: 711`；`node $P4TMP/keycount.mjs` → 1617；`node $P4TMP/web-size.mjs` →
`js: 397 files, 6858278 bytes`、`other: 112 files, 7910263 bytes`，其余与 Task 1 Step 1 相同（最大 chunk 1379330）。

Run: `node $P4TMP/lock-diff.mjs $P4TMP/base/pnpm-lock.yaml pnpm-lock.yaml` → catalog 删 3 行；importers 删 1（dotenv）、加 2（web 的 vitest、i18n 的 vite）；
packages、snapshots 各删 1。

Run: `git grep -n -E "process\.env|(^|[^A-Za-z0-9_])VITE_[A-Z]|dotenv" -- web turbo.json pnpm-workspace.yaml ':!*.md'` → 没有输出；
`git grep -n -E "[\"'\`]next/(link|navigation)[\"'\`]|useAppRouter|ensureTrailingSlash|compat/next" -- web ':!*.md'` → 没有输出。

提交之后：`node $P4TMP/notes/route-exports.mjs 9437a6e HEAD | grep -v "(deleted)"` →
`web/apps/web/app/layout.tsx	clientMiddleware,default -> default` 和汇总行；`node $P4TMP/symref.mjs orphaned 9437a6e`、
`node $P4TMP/keyref.mjs orphaned 9437a6e` → 0；`bash $P4TMP/headers.sh 9437a6e`、`node $P4TMP/dangling.mjs 9437a6e | tail -1` → 空、0。

`git diff --cached --stat 270f8e6` 为空；提交信息见 `$P4TMP/t6/msg.txt`。

**控制者在 Task 7 之后跑浏览器核对的 C 组，并重跑 A、B 两组（控制者评审补充第 1 条）。**

---

### Task 7: 删掉对字符串的空转换 `.toString()`（控制者评审补充第 1 条）

没有原型提交。基点是 Task 6 的提交。

**Steps**

- [ ] **Step 1: 记下现状**

Run: `git grep -c -E "\.toString\(\)" -- web/apps/web ':!*.test.ts'`，把每个文件的数目存到 `$P4TMP/logs/t7-before.txt`；报告合计。

- [ ] **Step 2: 写并运行脚本**

写 `$P4TMP/t7/tostring.mjs`（ts-morph，用 `web/apps/web/tsconfig.json` 建项目，先跑过 `react-router typegen`）：遍历 `web/apps/web` 的
`.ts`、`.tsx`（测试和声明文件除外），找所有没有实参的 `<表达式>.toString()` 和 `<表达式>?.toString()`；用类型检查器取接收者的类型：

- 类型是 `string`（或字符串字面量类型）→ 整个调用换成接收者的原文；
- `?.toString()` 且类型是 `string | undefined` → 同上；
- 其他一律不动，打印 `KEEP <文件>:<行> <类型文本>`。

脚本只按文本范围替换，最后打印 `removed <n> in <m> files`。然后 `pnpm exec oxfmt web/apps/web`。

- [ ] **Step 3: 核对只有这一种变化**

写并运行 `$P4TMP/t7/verify.mjs`：对 `git diff -U0` 的每一对删除行、新增行，把删除行里的 `.toString()` 和 `?.toString()` 去掉之后，
与新增行去掉空白后相同；oxfmt 合并或拆开的行按整段比较。输出 `pairs <n>, mismatches 0`，有不相同的就列出来逐个说明。

- [ ] **Step 4: 核对与门禁**

固定节奏第 2–6 步（孤儿核对的 `<基点>` 是 Task 6 的提交）；`node $P4TMP/tools/render.mjs .` → `render-time navigations: 0`；
`pnpm --dir web/apps/web exec vitest run` → 5 个测试通过。报告：删掉的行数和文件数、Step 2 的 `KEEP` 清单按类型分组、
lint 上限（如有变化说明原因）。

- [ ] **Step 5: 提交**

提交信息示例：`refactor(web): drop the no-op toString calls on strings`，正文说明它们来自 Next.js 的 `useParams`
返回 `string | string[]` 的时代，以及删除的范围由类型决定。

---

## 控制者的浏览器核对（M1 设计 4.2 "运行时核对"、7.5）

由控制者写脚本、跑、把全文和输出写进 P4 review 的附录；实现者不写。做法沿用 P3 review 附录 6：在 `$P4TMP/probe-app`
（本仓库的克隆，检出到被测提交，`make build-web`）上用 node 起静态服务器提供 `web/apps/web/build/client`（单页应用回退到
`index.html`），Playwright 取自 `e2e/`，`/api/`、`/auth/` 的请求全由桩回答，场景没有列出的请求一律算失败，每个场景检查"没有未捕获的
页面错误"和"控制台没有 `You should call navigate() in a React.useEffect()`"。地址一律按"不带结尾 `/`"断言。

**A 组：几种登录状态的跳转（Task 3 之后）**

1. 未登录打开 `/probe-ws/projects` → 落在 `/?next_path=/probe-ws/projects`，显示登录表单；`history.back()` 不再回到 `/probe-ws/projects`
   （`replace`）。
2. 已登录、未完成引导，打开 `/probe-ws/projects` → `/onboarding`；未登录打开 `/onboarding` → `/?next_path=/onboarding`。
3. 已登录、已完成引导，打开 `/` → 上次的工作区 `/probe-ws`；打开 `/?next_path=/probe-ws/projects` → `/probe-ws/projects`；
   `next_path=https://example.com/x` 被忽略，落在 `/probe-ws`。
4. 已完成引导、没有工作区（`last_workspace_slug` 为空、工作区列表为空）→ `/create-workspace`。
5. 后退和替换：面包屑的返回（`navigate(-1)`）回到上一页；删除 Webhook 之后落在 `/probe-ws/settings/webhooks`，后退不回到被删的
   Webhook 页。
6. 退出登录仍以表单提交 `/auth/sign-out/`（带 CSRF 令牌），落在 `/`。

**B 组：参数（Task 4 之后）**

7. 重跑 P2 review 附录的核对脚本 1（首页、个人主页、侧边栏）、2（应用中的编辑器）、3（动态、通知、工作项列表），P3 review 附录的
   核对脚本 1（认证与实例）、2（动态、通知、工作项列表与详情）：Task 4 改了这些页面的参数来源。
8. 个人设置 `/settings/profile/general` 上打开命令面板：没有未捕获的错误；如果它提供"创建工作项"，选它不打开弹窗
   （spec 第 3 节第 4 条）。

**C 组：地址、当前项、片段和旧地址（Task 6 之后）**

9. 当前项，每个地址都分别用不带和带结尾 `/` 两种形式打开，结果相同：
   - 侧边栏：`/probe-ws` → 只有"首页"是当前项；`/probe-ws/projects` → 只有"项目"；`/probe-ws/projects/archives` → 只有"归档"
     （"项目"不是）；`/probe-ws/profile/u1/assigned` → "你的工作"；`/probe-ws/drafts` → "草稿"；
   - 个人主页标签：`/probe-ws/profile/u1/created` → "创建的"一个标签是当前项，页面列出"创建的"工作项；
   - 工作区设置：`/probe-ws/settings` → 只有"常规"；`/probe-ws/settings/members` → 只有"成员"；
   - 项目设置：`/probe-ws/settings/projects/p1` → 只有"常规"；`/probe-ws/settings/projects/p1/labels` → 只有"标签"；
   - 应用栏：`/probe-ws/settings/members` 时"设置"是当前项；顶部通知：`/probe-ws/notifications` 时收件箱图标是当前项；
   - 项目导航：`/probe-ws/projects/p1/cycles/c1` 时"迭代"是当前项；归档标签：`/probe-ws/projects/p1/archives/modules` 时"模块"是当前项。
10. 应用内的跳转不带结尾 `/`：点侧边栏、设置侧边栏、顶部通知、面包屑之后的 `location.pathname` 都不以 `/` 结尾；没有任何 308 或
    客户端重定向（观察 `framenavigated` 只有一次）。
11. 查询参数和片段：打开 `/probe-ws/browse/PRB-1#comment-c1`（和带结尾 `/` 的形式）→ 地址保留 `#comment-c1`，评论卡片 `#comment-c1`
    带 `border-accent-strong`；`/probe-ws/projects/p1/intake?currentTab=open&inboxIssueId=i9` → 地址保留查询参数，打开的是 `i9`；
    `/probe-ws/profile/u1?x=1` → `/probe-ws/profile/u1/assigned?x=1`。
12. 旧地址落到"页面不存在"：`/sign-in`、`/login`、`/probe-ws/settings/account`、`/probe-ws/projects/p1/inbox`；停用迭代时迭代页的按钮
    去 `/probe-ws/settings/projects/p1/features/cycles`；命令面板的"账户设置"去 `/settings/profile/general`。
13. 重跑 A 组和 B 组第 7 项，P2、P3 脚本里带结尾 `/` 的预期地址改为不带（例如 P3 的 `/probe-ws/projects/`、`/browse/PRB-1/`、
    `/sign-up/?email=…`）。
14. **反向对照**：在基线 `9437a6e` 的构建上跑 C 组。第 10 项（地址以 `/` 结尾，多一次客户端重定向）、第 11 项的评论片段
    （中间件用 `request.url` 重定向，丢掉 `#`）、第 12 项（旧地址被重定向而不是"页面不存在"，停用页的按钮落到"页面不存在"）应当失败，
    证明脚本能发现 P4 改变的行为；第 9 项在基线上预计也通过（中间件把地址统一成带 `/` 的形式，`highlight` 按那个形式写），它核对的是
    P4 没有改坏当前项，基线上不通过的项要查明是不是基线本身的问题。
    C 组在 Task 7 之后跑（Task 6 之前中间件还在，地址总会被补上 `/`；Task 7 又改了读参数的代码）。

---

## 完成后

- 整分支评审之前，控制者在 `9437a6e..HEAD` 上跑一次 `node $P4TMP/notes/route-exports.mjs 9437a6e HEAD`、`render.mjs`、
  `targets.mjs --summary`、`pathlits.mjs --bad` 和 C 组核对。
- review 写明第 7 节交给 M2、M4 和收尾的事项（spec 第 7 节），把 M1 设计 12 节 P4 一行改为"已完成"并加上 review 的链接。
