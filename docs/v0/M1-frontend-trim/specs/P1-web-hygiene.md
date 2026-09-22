# M1/P1 工具链、遗留物与多语言：设计说明（spec）

| 项 | 内容 |
|---|---|
| Phase | M1/P1 `web-hygiene` |
| 日期 | 2026-09-22 |
| 状态 | 已批准（2026-09-23，控制者：第 3 节 12 项全部采纳，另补第 13 项） |
| 上级文档 | [M1 设计](../M1-design.md) 第 3.5、4.1（垫片中的死文件）、6、7.1、7.4、7.5（P1 一行）、7.6、8、9（P1）、10 节；[M0 设计](../../M0-foundation/M0-design.md) 2、5.2、6.1–6.3；[前端改动清单](../../frontend-changes.md) |
| 前置交接 | [M0-P5-frontend-trim-notes](../handoffs/M0-P5-frontend-trim-notes.md)、[M0-P6-knip-notes](../handoffs/M0-P6-knip-notes.md)：处理其中属于 P1 的事项，两份仍为 `open`（2.15） |

## 1. 目标
清掉迁入时带进来的工具链遗留和部署遗留，只留中英文，并把三条"靠人记着"的规则改成自动检查。P1 不删任何产品功能：
- **lint 上限**：警告数必须**等于**每个包的上限，多了、少了 `make lint-web` 都失败（M1 设计 7.1）；
- **关键词守卫**：进仓库的检查工具 `tools/keywords.mjs` 和规则文件 `tools/keywords.json`，`make lint-web` 和持续集成从 P1 起运行它；P1 删掉的每一样东西都在同一个提交里加上规则（M1 设计 7.4）；
- **前端单元测试**：新增 `make test-web`（turbo 的 `test` 任务，各包的 vitest），持续集成的 `web` 任务运行它；
- 删除部署遗留、没有调用方的 turbo 任务和脚本、Storybook、knip 报出的未使用依赖（`buffer` 除外）、`packages/services` 中没有调用方的 49 个文件、Next.js 垫片中没人用的文件；`pnpm-workspace.yaml` 中失效的条目和讲没有迁入内容的注释随之删除，每次都核对锁文件；
- 消除两条构建警告；每个包写自己的 TypeScript 增量编译文件；修复每次打开页面都有的 React #418；
- 多语言只留 `en`、`zh-CN`，不支持的语言回退到英文；删除没人用的翻译键生成；中英文键的双向一致性检查接进 `make lint-web`；删除企业版的死文案；
- `make web-dev` 同时监视各个包，改了包的源码，运行中的页面随之更新；
- 记录构建体积的基线（M1 设计 7.6）；
- 同步前端改动清单、README、M0 设计，在两份 M0 交接中写明 P1 的处理结果。

Plane 的旧地址重定向不在 P1（M1 设计 3.12：数据分析的一条随数据分析在 P2 删除，其余 10 条和仍然依赖它们的入口在 P4 一起处理）。

## 2. 交付物

### 2.1 文件总览
路径都相对于仓库根目录。

| 路径 | 内容 | Task |
|---|---|---|
| `tools/lint-cap.mjs`（新增） | 运行 oxlint，要求警告数等于上限（2.3） | 1 |
| 13 个包的 `package.json` | `check:lint` 改为调用 `lint-cap.mjs`（Task 1）；删掉没有调用方的脚本（Task 3）；删掉未使用的依赖（Task 2、3、4、6）；上限随警告减少调低（Task 3、4、5、9）；i18n 加 `test` 脚本和 `vitest`（Task 8） | 1–6、8、9 |
| `turbo.json` | `check:lint` 的输入加上 `lint-cap.mjs`（Task 1）；删掉 6 个没有调用方的任务，保留 `test`（Task 3）；新增 `check:sync`（Task 9） | 1、3、9 |
| `tools/keywords.mjs`、`tools/keywords.json`（新增） | 关键词守卫和它的规则（2.4）；之后每个删除 Task 加规则 | 2–5、8、9 |
| `Makefile` | `lint-web` 先运行关键词守卫（Task 2），再加 `check:sync`（Task 9）；新增 `test-web`（Task 3）；`web-dev` 监视各包（Task 10） | 2、3、9、10 |
| `.github/workflows/ci.yml` | `web` 任务加上 `make test-web` 一步 | 3 |
| `knip.jsonc` | 根工作区的 `entry` 加上 `tools/keywords.mjs`（Task 2）；删掉 i18n 的 `ignoreUnresolved`（Task 9） | 2、9 |
| `web/apps/web/` 的部署遗留、`public/` 中的无用文件、13 个 `.prettierignore` | 删除（2.5） | 2 |
| `.oxfmtrc.json` | 三条忽略规则代替 `.prettierignore`（Task 2）；删掉 `keys.generated.ts`（Task 9） | 2、9 |
| `pnpm-workspace.yaml`、`pnpm-lock.yaml` | 删掉失效条目和注释中讲没有迁入内容的句子；锁文件由 `pnpm install` 生成（2.6） | 2、3、4、6、8 |
| `.oxlintrc.json` | 删掉 Storybook 的两条忽略规则 | 3 |
| ui、propel 的 Storybook 配置、45 个 stories 和只为它们存在的文件与代码 | 删除（2.7） | 3 |
| 只为未使用依赖存在的文件；垫片中的 `image.tsx`、`script.tsx`、`next-script.d.ts`；`vite.config.ts` 中 `next/script` 的别名 | 删除（2.8） | 4 |
| `web/packages/services/` | 删掉 49 个文件，三个入口文件只导出剩下的部分；`url.test.ts` 保留（2.9） | 5 |
| `web/packages/tailwind-config/package.json`、`web/apps/web/vite.config.ts`、`web/packages/typescript-config/base.json` | 构建警告、增量编译文件（2.10） | 6 |
| `web/apps/web/app/root.tsx`、`core/components/common/logo-spinner.tsx` | React #418（2.11） | 7 |
| `web/packages/i18n/` 的 19 种语言、语言常量和类型、新的单元测试；web 中读取语言的三处 | 只留中英文，回退到英文（2.12） | 8 |
| `web/packages/i18n/` 的脚本、类型、8 个命名空间、`applications` 子树；`.gitignore` | 删除翻译键的生成，改写中英文键一致性检查，删除死文案（2.12） | 9 |
| `docs/v0/frontend-changes.md`、`README.md`、`docs/v0/M0-foundation/M0-design.md`、两份 M1 交接 | 文档同步（2.15） | 10 |

### 2.2 原型验证：结论与证据
在仓库外的临时克隆中，从本 worktree 的 `0f2b3e0`（`main` 的 `0e43771` 加上 M1 设计文档；代码与 `main` 相同）出发，按计划的 10 个 Task 完整执行了一遍，每个 Task 一个提交，每个提交都通过了 `make lint-web`、`make build-web`（Task 3 起还有 `make test-web`），并记录了 `make knip` 的报告。最后在两个全新的克隆（P1 之前、P1 之后）上各执行了一遍持续集成 `web` 任务的全部步骤。本 spec 和计划中的数值都来自这些运行。没有在原型中运行 `make e2e`（它要启动容器），计划的最后一个 Task 执行它。机器：Apple M 系列 18 核、macOS，Node 24.15.0、pnpm 11.10.0、turbo 2.10.11、oxlint 1.51.0、oxfmt 0.35.0、knip 6.37.0、vitest 4.1.11。

| 问题 | 结论 | 证据 |
|---|---|---|
| 上限怎样自动核对，而且只有一处来源 | 新增 `tools/lint-cap.mjs`；每个包的 `check:lint` 改为 `node <到仓库根目录的相对路径>/tools/lint-cap.mjs <上限>`（2.3） | 脚本在包目录中运行一次 `oxlint --format=json .`，运行目录、配置和忽略规则与原来的 `check:lint` 完全相同，按 `severity` 计数。13 个包的实测值与现在的上限逐个相等 |
| 有错误或进程失败时会不会被当成通过 | 不会 | 包里有一个无法解析的文件：列出 `src/lint-probe.ts:1:14  syntax  Unexpected token`，"errors are never allowed"，退出码 1；PATH 中没有 oxlint：`spawnSync oxlint ENOENT`，退出码 1；输出不是 JSON：打印原始输出，退出码 1；参数不是一个非负整数：退出码 2 |
| turbo 的缓存会不会重放错误的结果 | 不会：脚本列为 `check:lint` 的输入，上限在 `package.json` 中，本来就在包的哈希里 | 用单独的缓存目录执行 `turbo run check:lint`（23 个任务：13 个 `check:lint`、10 个包的构建），M1 设计 7.1 要求的六种情况：冷缓存 0 个命中；热缓存 23 个命中；少一条警告（删掉 types 中唯一带警告的空文件）types 失败并提示调低到 0，6 个命中；多一条警告（web 中加一个未使用的变量）web 失败，22 个命中；改上限（types 1 → 2）types 失败并提示调低到 1，6 个命中；改脚本 13 个 `check:lint` 全部重跑、构建命中，10 个命中；改依赖的包（hooks 的源码）hooks 及依赖它的 editor、propel、ui 的构建和 lint、web 的 lint 重跑，14 个命中。每次恢复原状后都是 23 个命中。不列脚本为输入时，只改脚本会 23 个全部命中（`FULL TURBO`），这就是要列的原因 |
| 关键词守卫能不能可靠地"报错即失败" | 能：Node 脚本，JavaScript 的 `RegExp`，显式 flags；规则带正例和反例样本，先自检（2.4） | P1 之前（只加 Task 2 的三条规则）：29 处命中，退出码 1，耗时约 0.2 秒。登记一条不再命中的例外：`stale exception … matches nothing now; delete it`，退出码 1；例外覆盖的处数与 `count` 不符：`exception count: … covers 2 hits, "count" says 1`，退出码 1。正则写错、某个正例样本不命中、flags 带 `g`、例外缺到期、规则文件不存在、文件不可读：各自给出原因，退出码 2。未跟踪但没有被忽略的文件也检查（本地加了新文件，提交前就能发现）；工作区中已经删掉、索引中还在的文件跳过 |
| 每个 Task 的规则在删除之前是否确实命中 | 是 | 每个 Task 先加规则，对删除之前的提交运行：Task 2 的 3 条规则 29 处，Task 3 的 2 条 356 处，Task 4 的 2 条 5 处，Task 5 的 1 条 49 处（正好是要删的 49 个文件），Task 8 的 1 条 532 处，Task 9 的 3 条 31 处；删除之后都是 0。P1 结束时 12 条规则、0 条例外 |
| 前端单元测试怎样进入持续集成 | `make test-web` = `turbo run test`；持续集成 `web` 任务在 `Lint` 之后执行它 | services 的 `url.test.ts` 13 个测试，Task 8 新增的 `language.test.ts` 10 个测试。把一个断言改错：`make test-web` 失败，`Failed: @plane/i18n#test`。任务数：Task 3–7 为 11 个（1 个测试加它依赖的 10 个包的构建），Task 8 起 12 个 |
| 13 个 `.prettierignore` 能不能直接删 | 不能直接删：oxfmt 会读取当前目录下的 `.prettierignore` | 只删文件时，各包的 `check:format` 检查到了 `dist/` 中的构建产物，失败。把其中实际起作用的三条改写为 `.oxfmtrc.json` 中相对仓库根目录的 `ignorePatterns` 之后，web 的格式检查覆盖的文件与改之前相同（1888 个）（2.5） |
| `public/favicon/` | 整个目录删除 | `site.webmanifest` 没有被引用（`root.tsx` 引用的是 `/site.webmanifest.json` 和 `/manifest.json`，都保留）；两个 PNG 只被它引用。`web/apps/web/manifest.json` 不在 `public/` 中，不会进入构建产物 |
| pnpm 工作区还有哪些失效条目 | 除随依赖删除的 catalog 条目外，还有 7 条覆盖项、1 条 allowBuilds、7 条 minimumReleaseAgeExclude；patchedDependencies 和 peerDependencyRules 都仍然有效 | 一次性脚本（计划 Global Constraints）逐条核对六类条目：catalog 被某个 `package.json` 或覆盖项引用；覆盖项和 peerDependencyRules 的每个包名在锁文件中；allowBuilds 的包在锁文件中；minimumReleaseAgeExclude 和 patchedDependencies 的"包名@版本"就在锁文件中。P1 之前没有输出。删掉 Storybook 之后它报出 `webpack`、`minimatch@3`、`ajv@6`、`ajv@8`、`flatted`、`fast-uri`、allowBuilds 的 `@swc/core` 和 7 个 Storybook 包的 minimumReleaseAgeExclude；删掉 `serve` 之后，`path-to-regexp: 0.1.13` 只剩更具体的 `router>path-to-regexp` 在决定版本（2.6） |
| 锁文件 | 只删除包；另有几处可以解释的副作用 | 一次性脚本逐项比较 importers，以及每个存活包的版本、完整性哈希和依赖边（2.6 的表）：5 次安装一共删掉 369 个包，没有新增包，没有任何存活包的版本或完整性哈希变化。`pnpm peers check` 前后都只报 `@tailwindcss/typography` 缺少对等依赖 `tailwindcss` 一条 |
| 哪些 turbo 任务和脚本有调用方 | `build`、`dev`、`check:*`、`fix:format`、`test` | Makefile 调用 `build`（`build-web`）、`dev`（`web-dev`）、`check:types`、`check:lint`、`check:format`（`lint-web`）、`test`（新的 `test-web`）；README 写明了 `pnpm exec turbo run fix:format`。`start`、`build-storybook`、`check`、`clean`、`fix`、`fix:lint` 没有调用方（2.7） |
| Storybook 能否整体删除 | 能 | ui、propel 各一套配置，共 45 个 stories，没有 Makefile 或持续集成入口。删掉 stories 后，knip 报出 propel 中 5 个只被 stories 导入的文件；ui 的 `styles/globals.css`、`postcss.config.js` 和 `postcss` 脚本，propel 的 `postcss.config.js`、`public/plane-lockup-light.svg` 也只为 Storybook 存在；propel 的 `ToastStatic` 组件注释写明"for Storybook and documentation"，没有其他调用方。删除后 propel 的警告 59 → 29，ui 32 → 31（2.7） |
| knip 报告的未使用依赖 | 除 `buffer` 外全部删除 | 删除之后 knip 的依赖、开发依赖、未列出的依赖三类只剩 editor 的 `buffer`。只为这些依赖存在的文件一起删除：`markdown-to-component.tsx`（react-markdown）、`use-font-face-observer.d.ts`、editor 的 `postcss.config.js`（未列出的 `@tailwindcss/postcss`；删掉它，editor 构建出的样式表逐字节相同）。`isbot`、`@react-router/node` 本来就不在报告中，保留（M1 设计 8）（2.8） |
| `packages/services` | 留 12 个文件 | web 只用到 `APITokenService`、`helpers` 中的地址规范化、`file/helper` 中的上传元数据工具。删掉其余 49 个文件后类型检查、构建、13 个地址测试都通过，services 的警告 6 → 0（2.9） |
| `tailwind-config` 的入口 | 删掉 `main`，入口只由 `exports` 声明 | 包里没有 JS 入口，`exports` 只导出 `./index.css` 和 `./postcss.config.js`，web 就是这样导入的，没有任何地方导入包名本身。加上 `"type": "module"`、删掉 `main` 之后，构建中的 `MODULE_TYPELESS_PACKAGE_JSON` 警告和 knip 的"Package entry file not found"提示都消失（2.10） |
| `vite-tsconfig-paths` | 换成 `resolve.tsconfigPaths: true` | 构建通过，Vite 的"检测到 vite-tsconfig-paths 插件"提示消失；此后构建只剩 Plane 自己的"有的代码块大于 500 kB"提示（2.10） |
| 增量编译文件 | `tsBuildInfoFile` 写成 `${configDir}/.turbo/tsconfig.tsbuildinfo` | TypeScript 5.5 起支持 `${configDir}`（仓库用 5.8.3）。改之前，继承 `base.json` 的包都写 `web/packages/typescript-config/.turbo/tsconfig.tsbuildinfo` 这一个文件；改之后 web 和 10 个包各写各的 `.turbo/`（`.gitignore` 已挡掉），原来那个文件不再生成（2.10） |
| React #418 的根因 | `HydrateFallback` 和 `LogoSpinner` 的输出取决于主题 | 预渲染时 `useTheme()` 没有主题，`HydrateFallback` 输出空的 `<div />`；浏览器中 next-themes 在水合之前已经设好主题，首次渲染输出加载动画，两者不一致。用静态服务器提供 `build/client`，Playwright 分别以亮色、暗色打开 `/` 和一个深层路径：修复前 4 次都有 #418；修复后都没有。拦截应用脚本、只看预渲染的 `index.html`：修复前看不到任何图片，修复后亮色下显示亮色的加载动画，暗色下显示暗色的（2.11） |
| 不支持的语言 | 在读取语言的三处换成英文 | i18next 因为配置了 `supportedLngs`，初始语言是 `fr` 时自己就解析成 `en`。问题在别处：用户资料中的 `fr` 经 `as TLanguage` 强转后交给 `setLanguage`，它把 `fr` 写回本地存储和 `<html lang>`；个人设置的语言选择框直接显示 `fr`。浏览器核对（拦截 Plane 的启动接口，登录一个资料语言为给定值的用户）：修复前 `fr` → `<html lang>=fr`、存储 `fr`；修复后 `fr` → `en`、`en`；对照 `zh-CN` → `zh-CN`、`zh-CN`（2.12） |
| 翻译键的生成 | 删除 | `keys.generated.ts`（3842 个键）只被 `TTranslationKeys` 使用，而这个类型没有任何导入。删除生成脚本后，knip 中 i18n 的 `ignoreUnresolved` 不再需要，删掉后也没有"Unresolved imports"（2.12） |
| 中英文键一致性检查 | 改写 `sync-check.ts`：两个目录都必须存在，按命名空间双向比较键集合，保留跨命名空间的重名和"叶子又是前缀"检查；作为 turbo 任务 `check:sync` 接进 `make lint-web` | P1 之前（原脚本）在 en 上没有重名和冲突。改写后逐项核对失败的情况：zh-CN 少一个键、zh-CN 多一个键、两个命名空间定义同一个键、一个键同时是另一个键的前缀、zh-CN 目录不存在，每种都失败并指出具体的键；`make lint-web` 失败时最后一行是 `Failed: @plane/i18n#check:sync`；没有改动时命中缓存（2.12） |
| 企业版的死文案 | 8 个命名空间加一个子树，引用都是 0 | 按"整键字面量 + 模板前缀"核对（计划 Task 9 的一次性脚本）：`automation` 141 个键、`editor` 29、`template` 137、`tour` 89、`update` 29、`wiki` 73、`work-item-type` 207、`workflow` 40，共 745 个；`workspace_settings.settings.applications` 161 个叶子键。对照：同一个 `workspace-settings.json` 的其余键中有 84 个被引用。en 和 zh-CN 的 JSON 经 `JSON.parse` 再 `JSON.stringify(…, null, 2)` 与原文件逐字节相同，所以删子树用脚本改写，不会带来格式改动 |
| `make web-dev` 能不能让包的改动出现在运行中的页面上 | 能，改为 `turbo run dev --filter=web... --concurrency=12` | web 和它依赖的 10 个包各有一个常驻的 `dev` 任务，共 11 个；`--concurrency=11` 时 turbo 拒绝启动。Playwright 打开开发服务器的首页，把 `@plane/constants` 中的 `SITE_NAME` 改成另一个值（`root.tsx` 把它写进 `<meta name="apple-mobile-web-app-title">`），不手动刷新：新值约 1.3 秒后出现在页面上（Vite 自动刷新一次），恢复源码后旧值同样回来；控制台没有 #418 和 `process is not defined`。对照：用 M0 的 `turbo run dev --filter=web` 做同样的修改，60 秒内页面没有变化（2.13） |
| 构建体积的基线 | 用同一个脚本在 P1 之前、P1 之后各测一次（2.14） | P1 之前 1239 个文件、32,232,132 字节；P1 之后 684 个文件、27,161,767 字节。两个全新克隆上的结果与原型一致 |
| 最终的检查结果 | 全部通过 | 全新克隆上的持续集成 `web` 任务：`make gen-check-web` 通过；`make lint-web` 50 个任务，冷缓存约 31 秒（P1 之前 49 个任务约 34 秒），有缓存时 1–2 秒；`make test-web` 12 个任务；`make knip` 2 秒；`make build-web` 11 个任务；之后工作区没有多出或改动的文件。knip：未使用的文件 131 → 123，依赖 19 → 1（`buffer`），开发依赖 4 → 0，未列出的依赖 2 → 0，导出 135 → 133，类型 83 → 81，枚举成员 4、重复导出 1 不变。`web/` 中跟踪的文件 4065 → 3375。警告合计 995 → 954 |

### 2.3 lint 上限的自动核对（Task 1）
**做法**：新增 `tools/lint-cap.mjs`（Node 脚本，无依赖）。每个包的 `check:lint` 从 `oxlint --max-warnings=<上限> .` 改为 `node ../../../tools/lint-cap.mjs <上限>`（e2e 是 `node ../tools/lint-cap.mjs 0`）。脚本在包目录中运行一次 `oxlint --format=json .`，然后：
- oxlint 无法启动：抛出异常，退出码 1；输出不是 JSON：打印 oxlint 的原始输出，退出码 1；
- 有错误（`severity` 不是 `warning`，包括无法解析的文件）：列出错误，失败；错误不计入上限，任何错误都直接失败，与 oxlint 自己的行为一致（`.oxlintrc.json` 目前没有 `error` 级别的规则）；
- 警告比上限多：按文件和位置列出这个包的全部警告，提示修掉新增的警告、上限只能调低，**不给出调高后的数值**（M1 设计 7.1）；
- 警告比上限少：提示在同一个提交里把 `package.json` 中的上限调低到实测值；
- 相等：打印一行 `<包名>: N oxlint warnings, equal to the cap.`；
- 参数不是一个非负整数：退出码 2。

**为什么这样做**：
- 上限只有一处来源，就是 `check:lint` 脚本里的数字，和 M0 一样；不新增集中的上限文件。集中文件要么让每个包的 lint 任务都依赖它（任何一个上限变了，13 个任务全部重跑），要么和 `package.json` 各存一份。
- "运行 oxlint + 比较上限"是同一个进程、同一个 turbo 任务，只运行一次 oxlint；另写一个"数一数警告"的检查，要么再运行一次 oxlint，要么解析 oxlint 面向人的文本输出。
- oxlint 的 JSON 报告不区分新旧警告，所以"多了"时列出全部警告；`make lint-web` 只打印失败任务的输出，看到的就是这个包的列表。
- **局限**（M1 设计 7.1）：数值相等不能证明"没有新增一条、同时删掉另一条"，新增的警告仍靠评审发现。

**turbo 缓存**：`turbo.json` 中 `check:lint` 的 `inputs` 加上 `"$TURBO_ROOT$/tools/lint-cap.mjs"`。包外的文件不在任务的哈希里，不加的话改了脚本 turbo 会重放旧结果。上限在 `package.json` 中，改上限会让这个包（以及依赖它的包）的构建缓存失效；调低上限的提交本来就改了代码，构建反正要重跑，代价可以忽略。六种情况的实测见 2.2，计划 Task 1 把它们写成可以重跑的步骤。

**上限的变化**（每次都由检查逼出来，在减少警告的同一个 Task 中调低）：

| 包 | P1 之前 | P1 之后 | 在哪个 Task 变化 |
|---|---|---|---|
| web | 779 | 777 | Task 4（删掉的死文件带走 2 条） |
| editor | 75 | 75 | |
| utils | 34 | 34 | |
| ui | 32 | 31 | Task 3（stories） |
| propel | 59 | 29 | Task 3（stories 和只给它们用的文件） |
| services | 6 | 0 | Task 5 |
| hooks | 4 | 4 | |
| i18n | 3 | 1 | Task 9（删掉的生成脚本） |
| constants | 2 | 2 | |
| types | 1 | 1 | |
| shared-state、`@nerve/api-client`、`@nerve/e2e` | 0 | 0 | 保持 0（M1 设计 7.1、P6 交接） |
| 合计 | 995 | 954 | |

**README 和 M0 设计 5.2** 改写规则的说明：警告数必须等于上限；看某个包的警告详情用 `pnpm --filter <包名> exec oxlint .`。

### 2.4 关键词守卫（Task 2 起）
**工具**：`tools/keywords.mjs`（Node 脚本，无依赖），规则在 `tools/keywords.json`。`make lint-web` 的第一行运行它（`node tools/keywords.mjs`，约 0.2 秒），所以本地和持续集成的 `Lint` 一步都经过它。它检查整个仓库、不属于任何工作区包，所以不经过 turbo。

- **文件范围**：`git ls-files --cached --others --exclude-standard`，即跟踪的文件加上未跟踪但没有被忽略的文件，按工作区中的样子检查：本地新加、还没 `git add` 的文件同样受检查；工作区中已经删掉的文件跳过。根目录的 `pnpm-lock.yaml`、`pnpm-workspace.yaml`、`turbo.json` 等也在范围内。内容规则跳过二进制文件（前 8000 个字节中有 NUL，与 git 的判断相同）。
- **规则**：路径规则（`path`）检查每个文件的路径；内容规则（`files` 选出文件、`content` 检查文本）逐处报告行号。正则都是 JavaScript 的 `RegExp`，`source` 和 `flags` 分开写，flags 只允许 `i`、`m`、`s`、`u`（全局匹配由工具自己加），在 macOS 和 Linux 上行为一致。每条规则有 `id`、`phase`（加入的 Phase）、`why`（中文理由）和样本：至少一个应命中的 `hit`、一个不应命中的 `miss`。工具先用样本自检，任何一条样本不符就退出。
- **例外**：`{ rule, path, match, reason, until }`，另有可选的 `count`（不小于 1 的整数，默认 1）。`match` 是命中的原文（路径规则就是路径本身），`until` 是到期的 M 或 Phase（如 `M3`、`M1/P4`）。一条例外只覆盖这一个文件里这一段原文，而且处数必须正好等于 `count`：同一个文件里同一段原文出现多处时（例如锁文件里 `serve@14.2.5:` 出现两处），写明 `"count": 2`；实际处数与 `count` 不符时工具失败并给出实际处数，所以一条例外不会把后来新增的同样一处顺带盖住。不再命中任何内容的例外报为过期（stale），工具同样失败，提醒删掉它。P1 没有例外。
- **退出码**：0 表示没有命中；1 表示有未登记的命中、过期的例外或处数不符的例外，逐条列出；2 表示规则文件、某个正则、样本、git 或读取文件出了问题。任何错误都不会被当成"零命中"。
- **knip**：它只由 Makefile 调用，`knip.jsonc` 的根工作区把它列为 `entry`，否则会被报为未使用的文件。

**P1 的 12 条规则**（每条在删掉对应内容的同一个 Task 中加入）：

| Task | 规则 | 类型 | 要点 |
|---|---|---|---|
| 2 | `deploy-files` | 路径 | `web/` 下的 `Dockerfile.web`、`Dockerfile.dev`、`.dockerignore`、`caddy/`、`sw.js`、`workbox-*.js` 及 source map |
| 2 | `prettierignore` | 路径 | 任何位置的 `.prettierignore` |
| 2 | `serve` | 内容 | `package.json`、`pnpm-workspace.yaml`、`pnpm-lock.yaml` 中的 `serve` 依赖和 `serve -s` 命令；不命中 `@react-router/serve`、`serve-static` |
| 3 | `storybook-files` | 路径 | `.storybook/`、`storybook-static/`、`*.stories.*` |
| 3 | `storybook` | 内容（不区分大小写） | `web/`、`e2e/`、根目录的 `package.json`、pnpm 的两个文件、`turbo.json`、`.oxlintrc.json`、`knip.jsonc` 中的 `storybook`、`chromatic` |
| 4 | `next-shim-dead-files` | 路径 | 垫片中的 `image`、`script` 和它们的类型声明 |
| 4 | `next-script-image` | 内容 | `web/` 的代码和 JSON 中带引号的 `next/script`、`next/image`；`next/link`、`next/navigation` 由 P4 加规则 |
| 5 | `services-files` | 路径 | `web/packages/services/` 下只允许保留的 12 个文件 |
| 8 | `locales` | 路径 | `web/packages/i18n/src/locales/` 下只允许 `en/` 和 `zh-CN/` |
| 9 | `i18n-key-generator` | 内容 | `keys.generated`、`TTranslationKeys`、`generate:types` |
| 9 | `i18n-dead-namespaces` | 路径 | 8 个企业版命名空间的 JSON |
| 9 | `i18n-applications` | 内容 | `workspace-settings.json` 中的 `"applications":` |

M1 设计 7.4 给 P1 的两行种子规则都在其中（`compat/next/(image|script)` 拆成了路径和内容两条）。其余几条同样是"P1 删掉、以后不应长回来"的东西：工具链的选择（oxfmt 而不是 prettier、nerve 而不是 serve）、Storybook、services 的范围、翻译键生成。删掉的依赖名（`react-markdown`、`zod` 等）不加规则：它们以后可能被正当地重新使用，是否再引入由评审决定。

**顺序**：守卫在 Task 2 加入，早于所有删除，这样每个删除 Task 都能先加规则、看到它命中，再删到零（2.2）。M1 设计 9 的建议顺序是把它放在 `packages/services` 之后，这里提前（第 3 节第 9 项）。

**它不检查自己**：`tools/` 下的脚本不属于任何工作区包，`make lint-web` 的格式和 lint 检查不覆盖它们（`tools/lint-cap.mjs` 也一样）。计划中每次改动它们，都单独执行一次 `pnpm exec oxfmt --check` 和 `pnpm exec oxlint`。

### 2.5 部署遗留（Task 2）
删除：
- `web/apps/web/` 中的 `Dockerfile.web`、`Dockerfile.dev`、`caddy/Caddyfile`、`.dockerignore`、`.gitignore`（只有 Sentry 的条目）、`manifest.json`（不在 `public/` 中，没有引用）；
- `public/` 中从未注册的 `sw.js`、`sw.js.map`、`workbox-9f2f79cf.js`、`workbox-9f2f79cf.js.map`，以及整个 `public/favicon/`（2.2）；
- web 的 `preview`、`start` 脚本和 `serve` 依赖（运行即崩溃，根因是全局覆盖 `path-to-regexp: 0.1.13`，见 2.6）；
- 13 个内容相同的 `.prettierignore`（web 和 12 个 Plane 包）。

**`.prettierignore` 的替代**：它们的内容是 `.next/ .react-router/ .turbo/ .vite/ build/ dist/ node_modules/ out/ pnpm-lock.yaml storybook-static/`，oxfmt 会读取当前目录下的这个文件（M1 设计以为不起作用）。其中真正会出现的只有 web 的 `.react-router/`、`build/` 和各包的 `dist/`（`node_modules/` oxfmt 本来就跳过；`.turbo/` 中没有 oxfmt 处理的文件类型；其余目录不会生成）。`.oxfmtrc.json` 的 `ignorePatterns` 加上 `web/apps/web/.react-router/**`、`web/apps/web/build/**`、`web/packages/*/dist/**`。它们相对 `.oxfmtrc.json` 所在的仓库根目录解析，从包目录运行也生效。

### 2.6 pnpm 工作区与锁文件（Task 2、3、4、6、8）
**删除的条目**（都用 2.2 的一次性脚本核对，删完之后没有输出）：

| Task | catalog | overrides | 其他 |
|---|---|---|---|
| 2 | `serve` | `path-to-regexp: 0.1.13`（及其两段注释） | |
| 3 | `@chromatic-com/storybook`、9 个 `@storybook/*`、`storybook`、`autoprefixer`、`postcss-cli`、`postcss-nested` | `webpack`、`minimatch@3`、`ajv@6`、`ajv@8`、`flatted`、`fast-uri`（及其 5 行注释） | allowBuilds 的 `@swc/core`；minimumReleaseAgeExclude 中 7 个 Storybook 包 |
| 4 | `@tiptap/extension-list-item`、`comlink`、`emoji-picker-react`、`react-fast-compare`、`react-markdown`、`zod` | | |
| 6 | `vite-tsconfig-paths` | | |

`vitest` 的 catalog 条目保留（services、i18n 的测试用它）。`@react-router/serve>express`、`router>path-to-regexp`、`body-parser`、`morgan`、`qs` 仍作用于锁文件中的包（`@react-router/serve` 是 `@react-router/dev` 的可选对等依赖，锁文件中有它），保留（M1 设计 8）。patchedDependencies（`react-color`）和 peerDependencyRules 的三条仍然作用于锁文件中的包，不动。

**注释**（Task 2）：删掉 Plane 英文注释中讲没有迁入内容的句子，其余原文不动：
- React 那段的最后一句（Vite 的 `resolve.dedupe` 只覆盖"三个应用"，不覆盖 SSR、tsdown 和 Storybook）；
- "The catalog stays on Express 4 for apps/live…"；
- brace-expansion 注释中的"reachable via serve>serve-handler>minimatch"；
- `path-to-regexp` 的两段注释（随条目删除）；
- browserslist 注释中"the runtime images copy node_modules wholesale…"两行（讲部署镜像）；
- react-popper 注释中"hidden behind the global strict-peer-dependencies=false"一句（讲没有迁入的 `.npmrc`）。

文件开头的中文说明补一句：M1/P1 删掉了其中讲没有迁入内容的句子。

**锁文件核对**（M1 设计 8）：每次安装之后，用一次性脚本 `lock-diff.mjs`（计划 Global Constraints）把新锁文件与提交中的比较：importers 逐个依赖比较说明符和解析结果，`packages` 和 `snapshots` 逐项比较版本、完整性哈希和依赖边，其余各节逐行比较；只打印删除以外的差异。要求：没有新增的包，没有存活包的版本或完整性哈希变化，其余差异与下表逐条相同；随后 `pnpm install --frozen-lockfile` 通过，六类条目的核对没有输出。

| Task | 删掉的包 | importers | 快照中的其他差异（都已核实原因） |
|---|---|---|---|
| 2 | 43 | 删掉 web 的 `serve` | `bytes`、`compressible`、`compression`、`debug@2.6.9`、`mime-db`、`ms@2.0.0`、`negotiator@0.6.4`、`on-headers`、`vary` 加上 `optional: true`：它们原来也经 `serve` 可达，现在只经可选的 `@react-router/serve` 可达。`tsx@4.20.6` 依赖的 `get-tsconfig` 从 4.13.7 变为图中已有的 4.14.3（4.13.7 随之删除）：pnpm 重新解析时向已有版本去重，只要重新解析就会发生 |
| 3 | 300 | 删掉 18 个依赖；constants 的 `tsdown` 的对等后缀 `oxc-resolver@11.20.0` 变为 `@11.24.2`，与其余 9 个包一致 | 11.20.0 是 Storybook 带进来的，随之删除；`tsdown`、`rolldown-plugin-dts` 各多一个带 `oxc-resolver@11.24.2` 后缀的快照键，替换原来的。`terser`、`acorn`、`commander@2.20.3`、`source-map-support`、`buffer-from`、`@jridgewell/source-map`、`range-parser`、`has-flag@4.0.0`、`supports-color@7.2.0` 加上 `optional: true`（原来经 webpack 可达，现在只经 Vite 的可选对等依赖） |
| 4 | 23 | 删掉 21 个依赖；utils 的 `remark-gfm` 带上对等后缀 `(supports-color@10.2.2)`，与 P6 中其他依赖 `debug` 的包一致 | `remark-gfm` 这一支的 4 个快照键带上同一后缀，5 个快照中对 `mdast-util-from-markdown` 的引用随之带上后缀 |
| 6 | 3 | 删掉 web 的 `vite-tsconfig-paths` | 无 |
| 8 | 0 | 新增 i18n 的 `vitest`，解析结果与 services 的完全相同 | 无 |

这些副作用都不是新版本：变化后的版本或后缀原本就在锁文件中。M1 设计 8 只允许"由删除直接引起的差异"，`tsx` 的 `get-tsconfig` 是重新解析时的去重，不完全属于这一类，见第 3 节第 7 项。锁文件在每次安装后的行数和 SHA-256 写在计划中（写作时的值）。

### 2.7 没有调用方的任务和脚本、Storybook；前端单元测试的入口（Task 3）
**原则**（M1 设计 8）：任务或脚本只有被 Makefile、持续集成、另一个任务调用，或者是 README 写明的开发入口，才保留。
- `turbo.json` 删掉 `build-storybook`、`check`、`clean`、`fix`、`fix:lint`、`start`；保留 `build`、`check:format`、`check:lint`、`check:types`、`dev`、`fix:format`、`test`（Task 9 再加 `check:sync`）。
- **`test` 保留并接上调用方**：新增 `make test-web`（`$(TURBO) run test $(TURBO_QUIET)`），持续集成 `web` 任务在 `Lint` 之后加一步 `Unit tests: make test-web`。services 的 `test` 脚本、`vitest` 和 `url.test.ts` 保留（M1 设计 3.5、8）。`make test` 仍然只运行 Go 测试：持续集成的 `server` 任务调用它，不需要 Node（第 3 节第 10 项）。
- **`fix:format` 保留**：README 写明 `pnpm exec turbo run fix:format` 是修格式的入口。M1 设计把它列在 `fix*` 中要删，这里按设计自己的原则保留；api-client 和 e2e 补上 `"fix:format": "oxfmt ."`，README 说的"所有包"才成立。
- 各包删掉 `clean`、`fix:lint`；i18n 另删 `sync:check`（与 `check:sync` 重复，后者 Task 9 改写）；propel 另删 `storybook`、`build-storybook`；ui 另删 `storybook`、`build-storybook`、`postcss`。
- **Storybook**：删掉 ui 的 `.storybook/`、propel 的 `.storybook/`、45 个 stories，以及只为 Storybook 存在的文件和代码：propel 的 `src/empty-state/assets/{horizontal-stack,illustration,vertical-stack}/constant.tsx`、`src/icons/constants.tsx`、`src/separator/separator.tsx`（只被 stories 导入）、`postcss.config.js`、`public/plane-lockup-light.svg`、`src/toast/toast.tsx` 中的 `ToastStatic`；ui 的 `styles/globals.css`、`postcss.config.js`。依赖：propel 的 3 个 `@storybook/*`、`storybook`、`@plane/tailwind-config`；ui 的 `@chromatic-com/storybook`、7 个 `@storybook/*`、`storybook`、`postcss-cli`、`autoprefixer`、`postcss-nested`、`@plane/tailwind-config`。`.oxlintrc.json` 删掉 `.storybook/**`、`storybook-static/**`。

### 2.8 未使用的依赖与死文件（Task 4）
- **依赖**：web 的 `clsx`、`comlink`、`emoji-picker-react`、`react-fast-compare`、`react-is`、`react-markdown`、`recharts`、`use-font-face-observer`；editor 的 `@tiptap/extension-list-item`；propel 的 `@plane/hooks`、`@plane/utils`、`@tanstack/react-table`；shared-state 的 `zod`；ui 的 `clsx`、`lucide-react`、`react-day-picker`、`tailwind-merge`、`use-font-face-observer`；types 的开发依赖 `@types/react-dom`。`react-is`、`recharts` 等仍被其他包使用，catalog 条目保留（2.6 的脚本核对）。
- **只为它们存在的文件**：`web/apps/web/core/components/ui/markdown-to-component.tsx`（react-markdown，没有导入）、`web/apps/web/use-font-face-observer.d.ts`（已删依赖的类型声明）、`web/packages/editor/postcss.config.js`（2.2），以及 editor 随之没有使用者的开发依赖 `postcss`、`@plane/tailwind-config`。
- **Next.js 垫片中的死文件**（M1 设计 4.1）：`app/compat/next/image.tsx`、`app/compat/next/script.tsx`、`app/types/next-script.d.ts`，`vite.config.ts` 中 `next/script` 的别名。`next/link`、`next/navigation` 的垫片留给 P4。
- **`buffer` 保留**：knip 把它当成 Node 的内置模块，实际上 Yjs 的工具代码还在导入它，随 Yjs 在 P2 删除（M1 设计 8）。

### 2.9 `packages/services` 修剪（Task 5）
保留 12 个文件：`package.json`、`tsconfig.json`、`tsdown.config.ts`、`src/index.ts`、`src/api.service.ts`、`src/helpers/{index,url,url.test}.ts`、`src/developer/{index,api-token.service}.ts`、`src/file/{index,helper}.ts`。其余 49 个文件（`ai/`、`auth/`、`cycle/`、`dashboard/`、`instance/`、`intake/`、`issue/`、`label/`、`module/`、`project/`、`state/`、`user/`、`workspace/` 整个目录，`developer/webhook.service.ts`，`file/` 中的三个服务，`indexedDB.service.ts`、`live.service.ts`）用 `git ls-files` 按保留名单反选后删除，计划给出命令和数量；守卫的 `services-files` 规则用同一份名单。三个入口文件保留 Plane 的版权声明，只导出剩下的部分。包本身留到 M5（M1 设计 3.5）。

### 2.10 构建警告与增量编译文件（Task 6）
- `tailwind-config/package.json`：加 `"type": "module"`（它的 `postcss.config.js` 是 ES 模块）；删掉 `"main": "tailwind.config.js"`：这个文件不存在，包也没有 JS 入口，真实的入口 `./index.css` 和 `./postcss.config.js` 已经由 `exports` 声明（M1 设计 7.2："按它真实的 `index.css` 入口和 `exports` 写，不为了消除提示造一个空的 JS 文件"）。
- `vite.config.ts`：删掉 `vite-tsconfig-paths` 插件，`resolve` 中加 `tsconfigPaths: true`（Vite 8 内置，读取的就是 web 自己的 `tsconfig.json`）；web 的开发依赖和 catalog 删掉它。
- `typescript-config/base.json`：`tsBuildInfoFile` 改为 `${configDir}/.turbo/tsconfig.tsbuildinfo`。`${configDir}` 是继承链最末端（各包自己的）`tsconfig.json` 所在的目录，写法不需要每个包各配一遍。

### 2.11 React #418（Task 7）
- **根因**（2.2）：SPA 模式下 `index.html` 在构建时预渲染 `HydrateFallback`。它按 `useTheme()` 决定输出：服务端没有主题，输出空的 `<div />`；浏览器中 next-themes 的内联脚本在水合之前就设好了主题，首次渲染输出加载动画。`LogoSpinner` 同样按 `resolvedTheme` 选图片。
- **修复**：`HydrateFallback` 不再读取主题，总是输出加载动画的容器；`LogoSpinner` 同时输出亮色、暗色两张图片，用 Tailwind 的 `dark:` 变体决定显示哪一张（变体按 `data-theme` 属性判断，next-themes 在首次绘制前设置它）。这样预渲染和首次渲染的标记完全相同，而且预渲染的页面在脚本加载之前就能显示正确颜色的加载动画。
- **影响范围**：`LogoSpinner` 在 web 中有 26 处使用，行为只有一处变化：`dark-contrast` 主题下原来显示亮色的动画（`resolvedTheme` 不等于 `dark`），现在显示暗色的，与这个主题一致（第 3 节第 8 项）。
- **核对**（M1 设计 7.5 的 P1 一行）：计划 Task 7 用一次性的静态服务器和 Playwright 脚本，修复前后各跑一次，输出写进 review 的附录。

### 2.12 多语言（Task 8、9）
**只留中英文**（Task 8）：删除 `web/packages/i18n/src/locales/` 下 19 种语言的目录（532 个文件，剩 56 个）；`SUPPORTED_LANGUAGES` 只剩 English、简体中文两项；`TLanguage` 改为 `"en" | "zh-CN"`。

**回退到英文**：新增并导出 `toSupportedLanguage(language)`：支持的语言原样返回，否则返回 `FALLBACK_LANGUAGE`（`en`）。本地存储和用户资料中的语言都是没有类型保证的字符串，凡是读取它们的地方都经过这个函数：
- web 的 `profile.store.ts` 取到或更新用户资料之后调用 `setLanguage`（原来用 `as TLanguage` 强转，`fr` 会被写回本地存储和 `<html lang>`）；
- 个人设置中的语言选择框：显示回退后的语言（"English"），不再显示原始值 `fr`；
- `core/instance.ts` 读取本地存储中的语言作为初始语言。这一处 i18next 自己也会解析成 `en`（2.2），经过同一个函数是为了让规则只有一种写法、类型是 `TLanguage`。

用户资料中存的 `fr` 不改写：前端只决定怎么显示，资料接口由 M2 定义。

**单元测试**（M1 设计 7.5："只给以后仍然有效的稳定逻辑写小测试"）：`src/constants/language.test.ts` 核对支持的语言正好是 `en`、`zh-CN`，两者原样返回，`fr`、`zh-TW`、`zh`、`EN`、空字符串、`null`、`undefined` 都得到 `en`（10 个测试）。i18n 加 `"test": "vitest run"` 和开发依赖 `vitest`（catalog 中已有，锁文件只多 i18n 的一条 importer）。浏览器中的核对见 2.2，脚本在计划 Task 8。

**翻译键的生成**（Task 9）：删除 `scripts/generate-types.ts`、`scripts/lib/locale-io.ts`、`generate:types` 脚本和 `TTranslationKeys` 的两处导出；`build` 改为 `tsdown`，`check:types` 改为 `tsc --noEmit`；`.gitignore`、`.oxfmtrc.json` 删掉 `keys.generated.ts`，`knip.jsonc` 删掉 i18n 的 `ignoreUnresolved`（P6 交接），knip 的注释改为只说 web 这一处。

**中英文键一致性检查**（Task 9，M1 设计 6）：`scripts/sync-check.ts` 重写：
- `src/locales/en`、`src/locales/zh-CN` 两个目录都必须存在；
- 两边的命名空间文件相同，每个命名空间展开后的键集合完全相同，多键、缺键都失败（按命名空间比较，键从一个命名空间挪到另一个也能发现）；
- 保留原脚本的跨命名空间检查，并对两种语言各做一次：同一个键定义在两个命名空间文件里，或者一个键同时是另一个键的前缀，都失败。所有命名空间共用一个键空间（`core/instance.ts` 把每个命名空间都设为后备），这两种情况会让键的含义不确定。

失败时逐条列出问题并以 1 退出。原脚本还比较其他语言、打印覆盖率，并且只在 `--ci` 时失败；只剩两种语言之后这些都不需要。i18n 的 `check:sync` 改为 `tsx scripts/sync-check.ts`；`turbo.json` 新增任务 `check:sync`（默认输入就是包内的全部文件，脚本和语言文件都在其中；不需要构建）；`make lint-web` 加上它，任务数 49 → 50。它只能保证中英文同步，两边一起删错要靠"整键 + 模板前缀"的引用核对（M1 设计 2.3、6）。

**企业版的死文案**（Task 9）：先用计划中的一次性脚本重新核对引用为 0，再删除 `automation`、`editor`、`template`、`tour`、`update`、`wiki`、`work-item-type`、`workflow` 8 个命名空间（en、zh-CN 各 8 个文件，`NAMESPACES` 中的 8 项），以及两种语言 `workspace-settings.json` 中的 `workspace_settings.settings.applications`（161 个叶子键）。`editor` 不在 M1 设计第 6 节的名单里，核对结果同样是 0 引用（第 3 节第 4 项）。`tour` 命名空间与保留的新手导览组件 `TourRoot` 无关（M1 设计 6），`TourRoot` 不动。

**不在 P1**：P2、P3 删功能之后才失去引用的键由那两个 Phase 删除；各处写死的 `"en-US"` 不变（M1 设计 6）。

### 2.13 `make web-dev`（Task 10）
`$(TURBO) run dev --filter=web... --concurrency=12`：`web...` 是 web 和它依赖的包，turbo 先构建这些包（`dev` 依赖 `^build`），再同时运行 11 个常驻任务（10 个包的 `tsdown --watch --no-clean` 和 web 的 `react-router dev`）。包的监视进程重新构建 `dist/` 之后，Vite 发现依赖变了，刷新页面。turbo 要求并发数大于常驻任务数，Makefile 中用中文注释写明 12 的来源；以后常驻任务数变了，turbo 会直接报错并给出需要的数值。

**核对**（M1 设计 7.5 的 P1 一行、第 8 节）：计划 Task 10 用一次性的 Playwright 脚本在运行中的页面上核对（2.2）：改 `@plane/constants` 的 `SITE_NAME`，不手动刷新，页面上的值随之变化；恢复后变回来。

### 2.14 构建体积的基线（Task 1、10）
M1 设计 7.6 要求 P1 和收尾用同一条命令测量。命令是计划 Global Constraints 中的一次性脚本 `web-size.mjs`（全文在计划中，收尾时照原样使用）：先 `rm -rf web/apps/web/build` 再 `make build-web`（turbo 从缓存恢复产物时不会删除目录中多出的旧文件，不先删掉会把旧产物也算进去），然后在仓库根目录执行 `node web-size.mjs`。它按扩展名把 `build/client` 分为 JS、CSS、字体（`woff2`、`woff`、`ttf`、`otf`、`eot`）和其他，统计文件数和字节数，找出最大的 JS chunk，并数出语言 chunk（以某个命名空间命名、只默认导出数据、不导入别的 chunk 的 JS 文件）。

| 项 | P1 之前（`0f2b3e0`） | P1 之后 |
|---|---|---|
| JS | 1055 个文件，15,151,327 字节 | 505 个文件，10,270,415 字节 |
| CSS | 3 个文件，326,068 字节 | 3 个文件，322,306 字节 |
| 字体 | 34 个文件，6,849,620 字节 | 34 个文件，6,849,620 字节 |
| 其他 | 147 个文件，9,905,117 字节 | 142 个文件，9,719,426 字节 |
| 最大的 chunk | `assets/toolbar-Bnxh0Ws1.js`，1,821,937 字节 | `assets/toolbar-Dhce5n2w.js`，1,821,937 字节 |
| 语言 chunk | 588 个（21 种语言 × 28 个命名空间） | 40 个（2 × 20） |
| 合计 | 1239 个文件，32,232,132 字节（30.7 MiB） | 684 个文件，27,161,767 字节（25.9 MiB） |

文件数只作记录，不作为门禁（M1 设计 7.6）。字节数是全部产物的总和，不等于首次加载的开销。

### 2.15 文档与交接（Task 10）
- **前端改动清单**："暂时使用"一行补上 services 的修剪；新增"1.3 工具链、遗留物与多语言（M1/P1）"表，逐项登记本 Phase 对 Plane 文件和根目录配置的改动及原因；第二节中多语言、部署遗留、`serve`、`sw.js` 四行改为"已完成 / M1/P1"，Next.js 垫片一行注明两个死文件已删除。旧地址重定向一行不动（P2、P4）。
- **README**：`make lint`、`make test-web`、`make web-dev`、lint 上限的规则、关键词守卫、knip 的现状（343 处）、新增"多语言"一条。
- **M0 设计**：第 2 节布局加上 `tools/lint-cap.mjs`、`tools/keywords.mjs` 和 `tools/keywords.json`；5.2 写明自动核对和 P1 之后的上限；6.1 中 `web-dev`、`lint` 两行，新增 `test-web` 一行；6.2 第 3 步；6.3 中 `web` 任务的步骤。
- **两份 M1 交接**各加"处理结果（M1/P1）"一节，状态仍为 `open`：M0-P5 交接中 `.env.example`、dotenv、`process.env` 和结尾 `/` 属于 P4；M0-P6 交接中 knip 门禁和配置提示属于 P3，S2 与包名属于 P5，`.env` 属于 P4。

## 3. 与上级设计的差异和补充（请控制者裁定）
都是局部的，没有架构层面的变化，也不涉及后端。

| # | 上级设计 | P1 的做法 | 理由 |
|---|---|---|---|
| 1 | M1 8："13 个 `.prettierignore`（格式化工具是 oxfmt）"，意思是直接删 | 删除，同时在 `.oxfmtrc.json` 加三条 `ignorePatterns` | oxfmt 会读取 `.prettierignore`，直接删会让格式检查扫到构建产物而失败（2.2、2.5） |
| 2 | M1 8、M0-P5 交接：删掉 `turbo.json` 中的 `fix*` | `fix`、`fix:lint` 删除；`fix:format` 保留，api-client、e2e 补上这个脚本 | README 写明的修格式入口；按 M1 设计 8 自己的原则（README 写明的入口保留）（2.7） |
| 3 | M1 7.2、8：`tailwind-config` 的入口按真实的 `index.css` 和 `exports` 写 | 删除 `main`，入口只由已有的 `exports` 声明 | 包没有 JS 入口，`main` 只能指向 JS；`exports` 已经列出 `./index.css`，web 也按它导入（2.10） |
| 4 | M1 6：企业版命名空间 `automation`、`workflow`、`template`、`work-item-type`、`tour`、`update`、`wiki` | 另删 `editor` 命名空间（29 个键，0 引用） | 按同一方法核对的结果（2.2、2.12） |
| 5 | M1 8：Storybook 连同配置、stories 和依赖删除；删 knip 报出的依赖及只被未使用文件引用的依赖 | 另删只为 Storybook 或已删依赖存在的文件和代码：propel 5 个只被 stories 导入的文件和 `ToastStatic`、ui 和 propel 的 PostCSS 配置与样式、editor 的 `postcss.config.js` 及其两个开发依赖、`use-font-face-observer.d.ts`；web 中没有被引用的 `.gitignore`、`manifest.json` 和整个 `public/favicon/`（`site.webmanifest` 和只被它引用的两个 PNG） | 删掉之后它们都成了死代码或没有使用者的依赖（2.5、2.7、2.8） |
| 6 | M1 8：删掉已不作用于任何包的条目 | 列出并删除了 7 条覆盖项、1 条 allowBuilds、7 条 minimumReleaseAgeExclude（2.6） | 用一次性脚本按六类逐条核对，不靠人工判断 |
| 7 | M1 8："只允许由删除直接引起的差异（对等依赖后缀、孤立的包）" | 另有一处：`tsx` 的 `get-tsconfig` 4.13.7 → 4.14.3 | pnpm 重新解析时向图中已有版本去重，没有新增包或版本；`tsx` 只用于 i18n 的 `check:sync`，由 `make lint-web` 覆盖。`optional: true` 的变化是删除直接引起的（2.6） |
| 8 | M1 8："改为预渲染和首次渲染输出同样的结构" | 同时把共用的 `LogoSpinner` 改为用 CSS 选图片（26 处使用） | 只改 `HydrateFallback` 不够：`LogoSpinner` 自己也按主题输出不同的 `src`，仍会不一致。副作用是 `dark-contrast` 主题改为显示暗色动画（2.11） |
| 9 | M1 9 P1 的建议顺序：lint 上限、关键词守卫排在 services 之后；M1 7.4 的 P1 种子规则两行；文件范围"来自 `git ls-files`" | 守卫在 Task 2 加入，早于所有删除；P1 共 12 条规则（2.4 的表）；文件范围是跟踪的文件加上未跟踪、未忽略的文件，跳过工作区中已删除的文件 | 先有守卫，每个删除 Task 才能在同一个提交里加规则并看到它从命中到零（M1 7.4"规则随删除加入"）；多出的规则都是 P1 删掉、不应长回来的东西；检查工作区而不只是索引，本地提交前就能发现问题（2.4） |
| 10 | M1 8："新增 `make` 入口（经 turbo 的 `test` 任务运行各包的 vitest）"；7.5："以后的 Phase 加小测试" | 入口叫 `make test-web`，`make test` 仍只运行 Go 测试；P1 同时给 `toSupportedLanguage` 加了 10 个测试 | Makefile 的约定是 `-web` 后缀需要 Node，持续集成的 `server` 任务调用 `make test`、没有 Node；回退规则是 7.5 所说的"以后仍然有效的稳定逻辑"（2.7、2.12） |
| 11 | M1 6：保留跨命名空间冲突的检查 | 重名和"叶子又是前缀"两种检查都保留，两种语言各做一次；比较按命名空间进行 | 两种都是共用键空间下的歧义；按命名空间比较能发现键在命名空间之间挪动（2.12） |
| 12 | M1 7.1："具体做法在 P1 spec 中通过原型确定" | `tools/lint-cap.mjs` + 各包 `check:lint` 中的上限 + turbo 输入（2.3） | 上限仍只有一处来源；六种缓存情况都已实测；改上限会让这个包的构建缓存失效，代价可以忽略 |

控制者裁定后，评审阶段把 M1 设计 6（`editor` 命名空间）、7.4（P1 的规则）、8（`fix:format`、`tailwind-config` 的入口、锁文件差异的表述）、9（P1 的顺序）同步更新；M0 设计 2、5.2、6.1–6.3 在 Task 10 中已经更新。

**控制者裁定（2026-09-23）**：
- 第 1–12 项全部采纳。M1 设计的第 6、7.4、8、9 节已在提交本 spec 时同步。
- 另补第 13 项：`tools/` 下的两个脚本本身就是门禁，却不受 `make lint-web` 的 lint 和格式检查，第 6 节风险表中"每次改动手工检查"只能管住 P1 自己。计划新增 Task 9A：根目录 `package.json` 加 `check:lint`、`check:format`，覆盖 `tools/`；`turbo.json` 注册根任务 `//#check:lint`、`//#check:format`。`make lint-web` 的任务数由 50 变为 52，此后手工检查不再需要（计划"控制者评审补充"第 2 条）。

## 4. 验收标准
1. **检查**：每个 Task 结束时 `make lint-web` 通过（Task 1–8 为 49 个任务，Task 9 为 50 个，Task 9A 起 52 个，包括覆盖 `tools/` 的两个根任务；Task 2 起前面都有关键词守卫的一行），`make build-web` 通过（11 个任务），Task 3 起 `make test-web` 通过（Task 3–7 为 11 个任务，Task 8 起 12 个），`make knip` 的报告与计划给出的数值一致。
2. **lint 上限**：13 个包的警告数等于上限（2.3 的表）；计划 Task 1 Step 9 的核对（无法解析的文件、参数错误，以及六种缓存情况，其中少一条、多一条、改上限三种失败）与计划给出的输出逐行一致。
3. **关键词守卫**：12 条规则、0 条例外、没有命中；每条规则加入时，对删除之前的提交有命中（数量见 2.2）；过期的例外、处数不符的例外、错误的正则、不符的样本分别以 1、1、2、2 退出。
4. **前端单元测试**：`make test-web` 运行 services 的 13 个、i18n 的 10 个测试；持续集成 `web` 任务有这一步。
5. **knip**：未使用的依赖只剩 editor 的 `buffer`，开发依赖和未列出的依赖为 0；配置提示最多剩 web 的 `+types/` 一条。
6. **锁文件**：每次安装后 `lock-diff.mjs` 的输出与计划逐行相同（没有新增包，没有存活包的版本或完整性哈希变化）；`pnpm install --frozen-lockfile` 通过；六类条目的核对没有输出。
7. **构建**：没有 `MODULE_TYPELESS_PACKAGE_JSON` 和 `vite-tsconfig-paths` 的提示；构建体积与 2.14 的表一致。
8. **7.5 的 P1 一行**（一次性脚本，输出写进 review 的附录）：亮色、暗色下打开首页和深层路径都没有 #418，预渲染的 `index.html` 显示与主题一致的加载动画；资料语言为 `fr` 时 `<html lang>` 和本地存储都是 `en`；en、zh-CN 的键集合相同（`check:sync`）；`make web-dev` 中改 `@plane/constants` 的符号，运行中的页面随之更新。
9. **端到端**：`make e2e` 通过（S1–S4，5 个测试）；持续集成的三个任务通过。
10. 前端改动清单、README、M0 设计已同步；两份 M1 交接写明了 P1 的处理结果。

## 5. 不在 P1 范围内
- 删除产品功能（M1 设计 2.1、2.2，P2、P3），包括 `buffer`、只在删功能之后才失去引用的文案。
- Plane 的旧地址重定向（M1 设计 3.12）：数据分析的一条在 P2，其余 10 条和仍然依赖它们的入口（命令面板的"账户设置"、两处指向 `/sign-in` 的链接）在 P4。
- Next.js 兼容层（M1 设计第 4 节，P4），P1 只删其中的死文件；`.env.example`、dotenv 和 `process.env` 注入、`turbo.json` 中的 `VITE_*` 环境变量也在 P4。
- 品牌和包名（P5）。
- knip 改为门禁、`--treat-config-hints-as-errors`、web 的 `ignoreUnresolved`（P3）。
- 清零 oxlint 警告本身（M1 设计 7.3）。

## 6. 风险

| 风险 | 应对 |
|---|---|
| 本地构建过的克隆里留着旧的 `keys.generated.ts`：Task 9 之后它不再被 `.gitignore` 挡掉，会出现在 `git status` 中，关键词守卫（`TTranslationKeys`）、格式检查和 knip 也会扫到它 | Task 9 的步骤删除它；review 和合并说明提醒其他工作区执行 `rm -f web/packages/i18n/src/types/keys.generated.ts`。守卫在本地直接指出这个文件，不会悄悄通过 |
| 守卫检查未跟踪的文件，本地随手放的文件可能让 `make lint-web` 在本地失败而持续集成通过 | 输出写明路径和规则；`.gitignore` 中的文件不检查。这是有意的：提交之前就能发现 |
| oxlint 的 JSON 格式随版本变化（`diagnostics`、`severity`、`labels[].span`） | 版本由 catalog 锁定为 1.51.0；解析失败时脚本打印原始输出并失败，不会误判为通过；升级 oxlint 时核对 |
| `tools/lint-cap.mjs`、`tools/keywords.mjs` 不属于任何工作区包，`make lint-web` 原本不检查它们自己的格式和 lint | Task 9A 加上根任务 `//#check:lint`、`//#check:format`，由 `make lint-web` 覆盖（第 3 节第 13 项）；在那之前的 Task 中，每次改动都用 `pnpm exec oxfmt --check`、`pnpm exec oxlint` 单独检查 |
| 警告"多了"时列出全部警告（web 有 777 条），新增的那条不好找 | oxlint 的报告不区分新旧；列表按文件和位置排序，可以与 `git diff` 涉及的文件对照。清零计划推进后列表会越来越短（M1 设计 7.3） |
| `make web-dev` 的并发数写死为 12：包的 `dev` 脚本增减后会不对 | 太小时 turbo 拒绝启动并给出需要的数值；太大没有影响。Makefile 的注释写明来源 |
| 第一次冷启动 `make web-dev` 时 Vite 预构建依赖，短暂出现"504 Outdated Optimize Dep" | 刷新一次即可；README 写明 |
| 锁文件的副作用（2.6）改变了 `tsx` 的一个间接依赖 | `tsx` 只用于 i18n 的 `check:sync`，由 `make lint-web` 覆盖 |
| `LogoSpinner` 的改动影响 26 处加载动画 | 标记只多一张隐藏的图片；两张图片都很小并且会被缓存；主题切换由 CSS 完成，不再依赖 React 重新渲染 |
| turbo 从缓存恢复构建产物时不删除目录中多出的旧文件，直接测体积会偏大 | 体积的命令先删掉 `web/apps/web/build`（2.14） |

## 7. 移交事项

| 交给 | 事项 |
|---|---|
| P2 | 删除 `/:workspaceSlug/analytics` 的重定向，与侧边栏的"数据分析"入口一起删（M1 设计 3.12） |
| P2、P3 | 删依赖之后用计划中的 `lock-diff.mjs` 和六类条目的核对脚本核对锁文件（M1 设计 8、9.7）；删文案用 Task 9 的引用核对脚本；每个功能的删除在同一个提交里往 `tools/keywords.json` 加规则，写法见 2.4 |
| P2、P3、P4、P5 | 每个减少警告的提交都要同时调低上限，`make lint-web` 会给出数值；稳定的逻辑往 `make test-web` 里加小测试（M1 设计 7.5） |
| P3 | knip 的配置提示只剩 web 的 `+types/`，按 M1 设计 7.2 处理 |
| P4 | 其余 10 条旧地址重定向和仍然依赖它们的入口（M1 设计 3.12）；垫片剩下的 `link.tsx`、`navigation.ts` 及其别名，加 `next/(link\|navigation)` 的守卫规则 |
| 收尾 | 用计划中的 `web-size.mjs`、按 2.14 的步骤再测一次构建体积，与 2.14 的表对比（M1 设计 7.6） |
| 控制者 | 提交本 spec 和计划时，把 M1 设计第 12 节中 P1 的状态改为"进行中"并加上链接 |
