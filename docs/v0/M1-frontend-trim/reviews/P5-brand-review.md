# M1/P5 品牌：评审记录

| 项 | 内容 |
|---|---|
| Phase | M1/P5 `brand` |
| 日期 | 2026-09-24 |
| 结论 | **通过** |
| spec / plan | [spec](../specs/P5-brand.md) / [plan](../plans/P5-brand.md)（控制者评审补充在 plan 的 Tasks 之前，`4dd8d54`） |
| 分支 | `worktree-m1-p5-brand`，从 `main` 的 `96d8c1d` 分出，提交从 `5685811` 到本评审记录所在的提交 |
| 评审 | 各 Task 评审（sonnet，第 2 节）；整分支评审（opus，`96d8c1d..91cfb44`）：Ready to merge with fixes；修复轮后的范围复核（sonnet）：Approved，没有新的发现 |

## 1. 范围与结果

P5 让前端成为 Nerve 的（M1 设计 5）：工作区的包改名为 `@nerve/*`；界面上的图形、页面标题、文案、链接不再有 Plane；代码里的标识符、会话存储键、剪贴板类型、组件名和注释不再写 plane；关键词守卫让它保持这样；Nerve 自己写的文件带 OpenNerve 的版权声明；`@makeplane/propel` 的来源记进前端改动清单。

| Task | 去掉的 | 改成的 |
|---|---|---|
| 1 | 工作区 12 个包的 `@plane/*` 名字（1100 个文件，2922 处） | `@nerve/*`；锁文件只随改名变化；守卫 `plane-package` |
| 2 | 讲 Plane 的导入分组注释 788 行 | 706 行改为 `// nerve …`；Plane CE/EE 目录留下的 `// plane web …` 按下一行导入的来源改名（49 行），与相邻一组同名的合并（5 行）；28 行悬空的删除。跟进提交收掉本 Task 造成的两处相邻同名分组 |
| 3 | Plane 的 Logo、横版标志、网站图标、应用图标、分享图、两张加载动画 GIF（共约 1.4 MB）；从未生效的第二份清单和 348 px 图标；登录页"Join 10,000+ teams"的客户 Logo 页脚；propel 的 `icons/brand/`、`icons/sub-brand/`；维护页插图屏幕上的 Plane Logo；3 张没有引用的 Plane 图片 | `app/assets/brand/` 的三个 SVG 和由它们渲染的 PNG、ICO；`nerve-logo.tsx` 的 `NerveLogo`、`NerveLockup`；与主题无关的加载动画；导览图片的卡片标题 "API integration" |
| 4 | 页面标题、元数据、中英文文案、写死的文字里的 Plane；只为已删功能存在的 16 个文案键；Plane 收集箱机器人的特殊显示 | `SITE_NAME = "Nerve"`，页面元数据只读这些常量（跟进提交）；"Nerve" 的文案；收集箱的创建者按普通用户显示 |
| 5 | 指向 Plane 服务的链接：论坛、文档、状态页、支持邮箱、X 账号、条款和隐私、"Star us on GitHub"、"Learn more about projects" | 帮助菜单和命令面板的"文档"打开 Nerve 的仓库，"Report a bug" 打开它的 issues（`REPOSITORY_URL`，都带 `noopener,noreferrer`）；错误页、维护页只留给用户的说明 |
| 6 | `PlaneVersionNumber`、会话存储键 `__plane_chunk_reload`、剪贴板类型 `text/plane-editor-html`、13 个 `plane-ui-*` 的 `displayName`、注释、包的说明和 `tailwind-config/AGENTS.md` 里几何意义的 plane | `VersionNumber`、`__nerve_chunk_reload`、`text/nerve-editor-html`（读写方一起改，不做迁移）、`nerve-ui-*`、"stacking context" |
| 7 | P1–P4 新写的 12 个文件上 Plane 的版权声明 | 守卫 `brand`、`brand-files` 和 1 条例外；OpenNerve 的版权声明；前端改动清单记录 propel 的来源 |
| 修复轮 | 8 张随应用发布的图片里 Plane 的像素 Logo 和工作区名 "Plane Design"；5 处写死的站点名；会被当作网站文件发布的 `public/icons/SOURCES.md` | 图片上是 Nerve 的圆形标记和 "Acme Design"；站点名只读 `SITE_NAME`；`brand` 规则的排除更精确 |

数字（P4 结束 `96d8c1d` → P5 结束 `f697c05`）：

| 项 | P4 结束 | P5 结束 |
|---|---|---|
| 关键词守卫 | 45 条规则、4 条例外，顶层 `phase` 为 `M1/P4` | **48 条规则**（新增 `plane-package`、`brand`、`brand-files`）、**5 条例外**（新增 `brand` 在 `pnpm-workspace.yaml` 的 3 行，`until: M9`），顶层 `phase` 为 `M1/P5` |
| `make lint-web` / `make test-web` / `make build-web` | 52 / 16 / 11 | 52 / 16 / 11 |
| `pnpm exec turbo run check:types` | 23 | 23 |
| vitest 测试 | 77 | 77，stderr 仍只有编辑器的 5 行 |
| lint 上限合计 | 711 | 711，不变 |
| `make knip` | 门禁，零 | 零 |
| 每种语言的文案键 / 无引用的键 | 1617 / 498 | 1599 / 482 |
| `docs/` 以外的 `@plane/`（行 / 文件） | 2963 / 1101 | **6 / 1**（`tools/keywords.json` 里 `plane-package` 的样本） |
| 品牌命中（`brandhits.mjs`，与 `brand` 规则同一正则） | 3902 | **3**（登记的例外） |
| `web/` 下名字带 plane 的路径 | 5 | 0 |
| 带 OpenNerve 版权声明的文件 | 0 | 17（新写的组件、3 个 SVG、`brand/SOURCES.md`，改正的 12 个） |
| 没有引用的图片（`assets.mjs`） | 136 / 251 | 133 / 246 |
| 构建体积 js | 397 个 / 6,849,609 字节 | 397 个 / 6,818,887 字节 |
| 构建体积 css / fonts | 3 / 296,988；25 / 3,755,608 | 3 / 296,816；25 / 3,755,608 |
| 构建体积 other | 112 / 7,910,263 | 109 / 6,435,038（两张 GIF 共 1,404,364 字节） |
| 最大 chunk / 语言 chunk | `use-parse-editor-content` 1,379,322 / 34 | 1,379,320 / 34 |
| 页面上的 "plane"（`brand.mjs`） | 基点的构建上 171 项里 64 项失败 | 177 项全部通过 |

- **改动规模**：`96d8c1d..f697c05`（不含本评审记录所在的提交）共 **1225 个文件改动，+4946/−4585 行**（`git diff --shortstat`）。不算文档，是 1220 个文件、+4095/−4578 行；**29 个文件被删除、11 个新增**（`app/assets/brand/` 的 10 个文件和 `nerve-logo.tsx`），文档另新增 spec 和 plan。
- **依赖**：没有增删和升级；锁文件与"基点锁文件做同样替换"逐字节相同（`lockcheck.mjs` 0 行），Task 1 之后不再变化；`pnpm -r ls --depth -1` 只有 `nerve`、`web` 和 `@nerve/*`。
- **保留行为的核对**（M1 设计 7.5 的 P5 一行）：
  - 临时核对脚本 `brand.mjs`：A 组看得见的页面（登录页的深浅两种主题、工作区、帮助菜单、两种加载动画、错误页、维护页、创建工作区、新手引导、404、工作区不存在、导览），B 组行为（命令面板的帮助命令、陈旧资源的恢复与会话存储键、编辑器的复制粘贴与剪贴板类型、个人访问令牌与 Webhook 的设置页、收集箱创建者），C 组图片（整分支评审之后加入：页面实际显示的 8 张图不是 Plane 的原图，"工作区不存在"的插图是装饰性的）。每个场景还检查页面文字和属性里没有 plane、没有未列出的请求、没有未捕获的页面错误、没有 `navigate()` 的警告。在 `f697c05` 的构建上 177 项全部通过，截图经控制者目视（附录 6.1）；
  - P4 的 9 段核对脚本（当前项、地址、跳转、命令面板，重跑的 P2、P3 探测）在 `f697c05` 的构建上 403 项全部通过（附录 6.2）；
  - 反向对照：在基点 `96d8c1d` 的构建上跑 `brand.mjs`，失败的正好是品牌检查（附录 6.3）。
- **文档**：`docs/v0/frontend-changes.md` 第一节 propel 的来源一行、1.5 节、第四节（Task 7、修复轮）；README 的版权一节；`M0-P6-knip-notes` 关闭（Task 7）；spec 的几处说法（修复轮、本评审提交）；M1 设计 12 节 P5 一行、M4 和 M8 的交接在本评审提交中更新。

### spec 第 4 节验收标准

| # | 验收标准 | 结论 |
|---|---|---|
| 1 | 7 个 Task 各一个提交，每个提交结束时 `check:types` 23、`make lint-web` 52、`make test-web` 16、`make build-web` 11 通过，`make knip` 为零 | **满足**。7 个 Task 提交（`12889c2`、`94a124a`、`06987ee`、`82d630d`、`4a9c11b`、`7187d70`、`91cfb44`），Task 2、3、4 的跟进提交 `f32dfe6`、`c7dcca7`、`0118dcd`，修复轮的 3 个提交；每个提交结束时门禁都通过 |
| 2 | `keywords: 48 rules, 5 exceptions, no hits.`；警告数等于上限（711）；`alts.mjs M1/P5` 为 0 | **满足** |
| 3 | vitest 77 个；stderr 只有编辑器的 5 行 | **满足** |
| 4 | `docs/` 以外的 `@plane/` 只有 `tools/keywords.json:6`；`lockcheck.mjs` 0 行；Task 1 之后锁文件不变；`pnpm -r ls` 只有 `nerve`、`web`、`@nerve/*` | **满足**。整分支评审复跑确认 |
| 5 | `brandhits.mjs --summary` 最后一行 `brand hits: 3, file names: 0` | **满足** |
| 6 | 孤儿核对（基点 `96d8c1d`） | **满足**。`symref`、`keyref` 为 0，`headers.sh`、`dangling` 没有输出；`infile-orphans` 11 行都以 `defined now: 0)` 结尾（Task 3 的 8 个、Task 5 的 2 个、Task 6 的 1 个） |
| 7 | 新增的文件和改正的 12 个文件带 Nerve 的版权声明；PNG、ICO 登记在 `SOURCES.md` | **满足**。17 个文件带 OpenNerve 的声明（修复轮把 `public/icons/SOURCES.md` 并入 `brand/SOURCES.md` 之前是 18 个）；8 个 PNG、ICO 都登记在 `app/assets/brand/SOURCES.md` |
| 8 | 控制者的浏览器核对全部通过，截图经目视，脚本写进附录；S1–S4 通过；持续集成的 `server`、`web`、`e2e` 通过 | **满足**。浏览器核对见附录 6。持续集成 run 36015556286 在 `f697c05` 上通过（合并后在 `main` 上再跑一次）：`server` 43 秒，`web` 139 秒，`e2e` 106 秒（S1–S4 所在的 E2E 一步） |
| 9 | 前端改动清单、README、`M0-P6-knip-notes` 同步；M1 设计 12 节更新 | **满足** |

## 2. 各 Task 的评审

- 实现者和修复轮都用 Opus 5.5，各 Task 的评审者用 sonnet，整分支评审用 opus。实现者在前一个 Task 的评审进行时就开始下一个 Task，评审者的核对都钉在被评审的提交上（工作树的 HEAD 在前进时用干净的克隆）。
- 原型（`$P5TMP/proto`，spec 2.2）只作为对照答案，不整体搬运：每个 Task 逐步执行，报告与原型提交逐文件比较，每一处差异都说明原因。
- Task 1、2 是机械改动（控制者评审补充第 2 条）：控制者先在基点的干净克隆上重放脚本，确认提交的差异与脚本写出的差异相同，评审者只判断每一类替换的意思是否不变。
- 每份报告回答七个风险点：`@plane/` 只剩样本、锁文件只随改名变化、没有升级依赖、改到的页面上看不到 Plane（改了图片的写出大小和哈希，并打开看过）、存储键和剪贴板类型读写一致、没有新的共享可变状态、版权声明按来源。

| Task | 内容 | 提交 | 核实方式 |
|---|---|---|---|
| 1 | 包名 `@plane/*` → `@nerve/*` | `12889c2` | 实现者 DONE（1101 个文件，+2999/−2964），内容与原型 `40a54db` 相同。`rename.mjs` 替换 1100 个文件里的 2922 处；`lockcheck.mjs` 0 行（锁文件 +41/−41，只有 importer 的键）；`pnpm -r ls` 16 项，12 个 `@nerve/*@1.4.2`，都是 `private`；守卫 46 条规则，新增 `plane-package`；`docs/` 以外的 `@plane/` 只剩 `tools/keywords.json:6`；品牌命中 3902 → 964。<br>控制者重放：在基点的干净克隆上跑 `rename.mjs`，`git diff 12889c2 -- . ':!pnpm-lock.yaml' ':!tools/keywords.json' ':!docs'` 为空；`keywords.json` 与重放只差新规则和 `phase` 一行。<br>评审 Approved（钉在 `12889c2`）：`rename.mjs` 是对 `git grep -l -z -I -F "@plane/"` 结果的定长替换，没有转义的 `@plane\/`、分段拼出的包名或不带 `@` 的 `plane/<包名>`；分类 grep 的 623 处里 612 处是 `@makeplane/propel`（按设计不动）；tsconfig 的 `extends`、Vite、turbo、Makefile、CI、`knip.jsonc` 都能解析新包名；旧规则被改名的样本仍然各自命中、不命中 |
| 2 | 导入分组注释 | `94a124a`、`f32dfe6` | 实现者 DONE_WITH_CONCERNS（671 个文件，+755/−788），web 的树与原型 `be5666a` 相同。`labels.mjs`：A 类改名 706，A 类悬空删除 21，B 类悬空删除 7，B 类合并 5，B 类改名 49；`verify.mjs` `other 0`；`dangling` 0；品牌命中 176。实现者逐一核对了全部 61 行 B 类。<br>控制者重放：在 `12889c2` 的干净克隆上跑 `labels.mjs`，与提交逐字节相同。<br>实现者的疑问与**裁定**：两处相邻的同名分组——`use-workspace-issue-properties.ts` 的第二个 `// nerve imports` 标着 `./` 的导入（基点就标错了，A 类规则把它改了名），`permissions.store.ts` 的 B 类标签落在已有的 `// services` 正上方（脚本只向上合并）。这是本 Task 造成或暴露的退化结构，由同一位实现者在跟进提交里收掉（`// local imports`；删掉下面那行 `// services`）；基点其余的错标不动。<br>评审 Approved（钉在 `f32dfe6`）：两个标签的正则都锚定整行，散文、字符串、JSX 注释碰不到；`importSource` 处理 type 导入和多行导入；`labelAbove` 遇到第一个非导入行就停，不会跨代码合并；五类各抽 5 行以上，B 类合并 5 行、B 类删除 7 行全部核对。评审者自己写的相邻同名扫描在 `94a124a` 找到 6 对，2 对是本 Task 造成的（`f32dfe6` 已收掉），4 对在基点就有（交收尾）。Minor（一次性脚本，不改）：`labels.mjs` 对 `web` 大小写敏感（没有这样的行）；`verify.mjs` 的说明比代码做的多（全局计数已足够） |
| 3 | Nerve 的图形 | `06987ee`、`c7dcca7` | 实现者 DONE（54 个文件，+147/−417，与原型 `a559e05` 形状相同）。8 个 PNG、ICO 的哈希与 plan 相同（XML 注释不参与渲染）；`retext.mjs` 228,386 字节；`infile-orphans` 8 行都以 `defined now: 0)` 结尾（`AuthFooter`、4 个客户 Logo、`PlaneLockup`、`PlaneLogo`、`PlaneNewIcon`）；没有引用的图片 136 → 133，只少了 3 张没有引用的 Plane 图片；品牌命中 145，名字带 plane 的路径 0。与原型的差异：6 个文件的版权持有者是 OpenNerve（执行前的控制者裁定），构建的 other 因此小 36 字节（3 个 SVG 和会被复制进构建的 `public/icons/SOURCES.md`）。实现者用读图工具打开了分享图、应用图标、网站图标、三个 SVG（深浅背景和导览的底色上）、导览图片的裁切和维护页插图。<br>**裁定**：守卫 `deploy-files` 的不命中样本写着本 Task 删掉的 `public/manifest.json`；不命中样本要引用保留的代码，由删掉它的 Task 改（跟进提交 `c7dcca7`：改为 `public/site.webmanifest.json`）。`server/internal/platform/webui/handler_test.go` 的假文件名不依赖 web 的构建，交收尾。<br>控制者的浏览器核对第一轮（第 4 节裁定 11）：Task 3 的 10 项检查全部通过，它们在基点上都失败。<br>评审 Approved（钉在 `c7dcca7`）：评审者自己打开了所有新图片（16 px 网站图标放大 16 倍，N 和两个圆点仍可分辨）；`nerve-logo.tsx` 没有无用的分支或属性；`onboarding/header.tsx`、`tour/root.tsx` 为 `NerveLockup` 新建 `// components` 分组是对的（把 `@/` 导入放进包名的分组会标错它，其他导入没有移动）；删除的每个文件、路径和符号在 web、e2e、server、tools、Makefile、.github 里都没有残留 |
| 4 | 文案、页面标题和元数据 | `82d630d`、`0118dcd` | 实现者 DONE（35 个文件，+52/−131，与原型 `61538ba` 的 35 个文件逐字节相同）。文案键 1617 → 1601；品牌命中 71。实现者发现门禁的构建是 turbo 的缓存命中，另跑了不带缓存的构建：构建产物里没有 "New to Plane"、"Plane Pages"、`-intake`、"Plane logo"，预渲染的 `index.html` 里没有 plane。<br>实现者报告的基点问题与**裁定**：`root.tsx` 的 `og:description` 手写了 `SITE_DESCRIPTION` 的原文，`keywords` 手写了与没有读取方的 `SITE_KEYWORDS` 几乎相同的文字——同一个值两个来源，就在本 Task 改写的元数据里，所以由本 Task 的跟进提交收成一个来源（`0118dcd`，`SITE_KEYWORDS` 取页面实际显示的 "work item tracking"）；实现者证明构建是确定的（同一提交两次不带缓存的构建 538 个文件相同），改动之后 `<title>` 和每个 `<meta>` 逐字节不变，真正变了内容的只有 root 和常量两个 chunk。`root.tsx` 里悬空的 `// types` 在基点就有，交收尾。<br>评审 Approved，没有发现：每处新文字与 `t4/edits.mjs` 相同，中英文对应，占位符完整；删掉的常量和 7 棵文案子树没有读取方；`archived-at.tsx` 的系统归档显示 "Nerve"，恢复仍显示真实的用户 |
| 5 | Plane 服务的链接 | `4a9c11b` | 实现者 DONE_WITH_CONCERNS（19 个文件，+30/−210：原型 `6b440f0` 的 18 个文件逐字节相同，另加 `tools/keywords.json` 3 行）。`infile-orphans` 列出 `StarUsOnGitHubLink`、`TermsAndConditions`；文案键 1599；没有引用的图片 133 / 246；品牌命中 46；构建产物里有 `open-nerve/NerveProject`，没有 Plane 的服务地址和文案。<br>实现者的疑问与**裁定**：守卫 `integrations` 的两个不命中样本引用本 Task 删掉的 GitHub 图片导入和 "Star us on GitHub" 文案，规则的说明还说那个链接"保留"——与 `c7dcca7` 同一条裁定，由删掉代码的 Task 改为引用保留的两行（`help-commands.ts:7`、`metadata.ts:9`），说明随之改正。<br>评审 Approved，没有发现：`REPOSITORY_URL` 是仓库地址在 web 里唯一的字面量；每个打开外部地址的 `window.open` 都带 `noopener,noreferrer`（帮助菜单的一处原来没有）；维护页只剩一个元素的片段、项目设置页只包一个按钮的容器都收掉了 |
| 6 | 标识符、存储键、剪贴板类型、组件名和注释 | `7187d70` | 实现者 DONE（27 个文件，+44/−44：原型 `89ba562` 的 26 个文件，另加 `tools/keywords.json` 1 行）。品牌命中 3（`pnpm-workspace.yaml` 第 2、3、168 行）。`__nerve_chunk_reload` 经 `STALE_ASSET_RELOAD_KEY` 一个常量读写；`text/nerve-editor-html` 的两个写入方（`editor-ref.ts:90`、`markdown-clipboard.ts:41`）和读取方（`props.ts:41`）一致；测试、夹具、e2e 里没有旧名字。<br>**裁定**：守卫 `changelog` 的不命中样本引用本 Task 改名的 `PlaneVersionNumber` 导入，改为改名后的那一行（原型没有改，那个样本因此不再是真实的代码）。<br>评审 Approved，没有发现：改写的每条注释对它所在的代码都成立；没有代码读取 `displayName`；不做迁移的后果（spec 第 1 节）：旧的 `__plane_chunk_reload` 不再被读写，新键读到 `null`，相当于新的会话（刷新一次，再失败时显示错误页，不会循环）；只带旧剪贴板类型的内容粘贴时回退到 ProseMirror 按 `text/html` 粘贴，只少了附件复制那一步 |
| 7 | 守卫、版权、propel 的来源、文档 | `91cfb44` | 实现者 DONE_WITH_CONCERNS（16 个文件，+113/−29）。18 个文件的来源比例与 plan 相同；`headers.mjs` 改 12 个文件；`brand` 在基点命中 3902 处（1165 个文件），`brand-files` 5 个路径，在本分支是 3 处和 0；守卫 48 条规则、5 条例外；构建体积与原型的差别逐字节说清（other −36 是版权持有者，js −50 是 `0118dcd`）。实现者在跑完 `docs.mjs` 之后改了前端改动清单 1.5 节的四行，让它们符合本分支真正做的事（导入标签的跟进提交、GIF 共约 1.4 MB、元数据读常量、三条旧规则的样本改为保留的代码），并在线核对了 propel 的每项事实。<br>实现者的两处偏离与**裁定**：`brand` 规则的说明"本地存储键"改为"会话存储键、剪贴板类型"（它是 `sessionStorage`，规则也管剪贴板类型）——采纳；`brand` 的命中样本 `import { PlaneLogo } from "@plane/propel/icons";` 改为 `…"@nerve/propel/icons";`——原型的样本自己带着 `@plane/`，让 `docs/` 以外的 `@plane/` 计数成为 7（原型上也是 7），与 spec 的 6 不符；新样本是同一个导入改名后的写法，在 Task 3 删掉之前是真实的代码，命中样本可以指向已删的代码——采纳。<br>评审 Approved：两条规则用每个样本和一组对抗输入实测；web 里 `[a-z]plane\|plane[a-z]` 的 596 行都是 `@makeplane`，今天没有 airplane、planet、explanation 这类词；每个不命中样本在 `91cfb44` 上逐字存在；例外正好是 `pnpm-workspace.yaml` 里仅有的 3 处 Plane，理由属实；12 个文件只改了文件头；propel 一行的每项事实在线复核（npm 注册表、SLSA 证明、仓库 404、2699 个 source map 的来源 1376 个、带全文 1375 个）。Minor：`brand` 的第二个否定前瞻结尾的 `\b` 也会放过假想的 `@makeplane/propel-extra`，交修复轮 |
| 修复轮 | 整分支评审的 I1、I2、M1–M5（第 3 节） | `8ff2d88`、`a041cad`、`f697c05` | 实现者 DONE_WITH_CONCERNS（3 个提交，22 个文件，+39/−43）。每个提交都跑了 `check:types`、孤儿核对（基点为前一个提交）、守卫、品牌命中、`lintdiff`、上限、门禁和测试。<br>- 提交 1（图片）：两张"工作区不可用 / 不能创建"插图下方的蓝色圆章改为 Nerve 的圆形标记（`#155E75` 圆盘、白色 N、两个 `#5EEAD4` 圆点，取自 `mark.svg` 的几何，原来的蓝色像素 23214 → 0，阴影保留）；"No projects yet" 卡片上由格子拼成的 Plane Logo 用相邻的空白格子补齐，网格连续；停用模块、视图页插图面包屑里的 "Plane Design" 改为 "Acme Design"（按原字的颜色、字号、字重和基线拟合）；PNG 用无损的写法，没有改到的像素逐字节不变。"工作区不存在"的插图改为装饰性的 `alt=""`。实现者按能看清 20 px 细节的尺寸逐张看了代码引用的全部 113 张图片和 propel 的 25 个插图组件（深浅两种主题），带 Plane 的只有这 8 张；<br>- 提交 2（代码）：`root.tsx` 的两个图片 `alt`、登录页页头和注册页的标题、收集箱标题的回退值、动态里的系统操作者都读 `SITE_NAME`；句子里提到 Nerve 的文案仍写字面量（与文案文件一样，第 4 节裁定 5）；`brand` 的否定前瞻改为 `(?![\w-])`，加一个只命中 `github.com/makeplane` 的样本（基点邀请页第 117 行）；只命中 `plane/propel`、不带 `@plane/` 的基点行不存在（450 行都带 `@plane/`，由 `plane-package` 管）；`public/icons/SOURCES.md` 并入 `app/assets/brand/SOURCES.md`；<br>- 提交 3（文档）：README 和前端改动清单第四节写明 OpenNerve 的声明覆盖哪些文件（`api-client` 和 `web/` 以外以根目录 `LICENSE` 为准）；M2 交接、前端改动清单里还在教人用 `@plane/*` 的三处改为 `@nerve/*`；spec 第 1 节、2.2 结论 4、2.5、2.6、3.4 和第 6 节的图片一行按实际情况改写。<br>控制者目视了改过的图片（第 4 节裁定 9）。实现者的疑问与**裁定**：句子里的 Nerve 保留字面量——采纳；前端改动清单第 135 行（已完成的 P4 行，但告诉读者 `isValidNextPath` 在哪个包）改为 `@nerve/utils`——采纳；spec 2.2 表里 "18 个带 Nerve 声明的文件" 现在是 17 个——本评审提交补一句 |

## 3. 整分支评审（opus）：Ready to merge with fixes

评审范围 `96d8c1d..91cfb44`，结果为 0 Critical、2 Important、8 Minor。评审者逐行读了 Task 3–7 的差异，Task 1、2 的机械差异通过脚本和 grep 核对；复跑了 spec 第 4 节的全部工具，构建之后在 `build/client` 里查 plane（只有打包进来的 `package.json` 里的依赖名 `@makeplane/propel` 和表情标签 `plane`、`airplane`、`planet`，都不显示），并写了几个新的只读脚本：

- `misscheck.mjs`：旧规则里引用基点真实代码、而现在已不存在的不命中样本，结果为 0；
- `bluescan.mjs`、`halves.mjs`、`propel-illus.mjs`：按原尺寸重看图片，找到 I1、I2。

评审者的结论是：改名是纯机械的，锁文件逐字节对得上；`brand` 在基点的 3902 处上证明有效，每条旧规则的不命中样本仍引用真实的代码；Plane 的服务地址全部去掉，`REPOSITORY_URL` 是唯一的来源；会话存储键和剪贴板类型两端一致；版权声明依据逐行的来源分析；propel 的来源有记录。唯一的缺口在图片里——守卫和按文字检查的探测都看不见。

| 编号 | 级别 | 问题 | 处理 |
|---|---|---|---|
| I1 | Important | Plane 的像素 Logo 还在 4 张随应用发布的图片里：`workspace/workspace-not-available.png`、`workspace-creation-disabled.png` 下方的蓝色圆章（"Workspace not found"、禁止创建工作区时显示；基点的代码给前一张写的 `alt` 就是 "Plane logo"），`empty-state/project-settings/no-projects-light.png`、`-dark.png` 卡片上由格子拼成的 Logo。spec 2.2 结论 4 说全部图片都看过，这句不成立 | 已修（`8ff2d88`）：圆章改为 Nerve 的圆形标记，格子补齐；插图 `alt=""`；spec 2.2 结论 4、第 6 节改写（`f697c05`） |
| I2 | Important | 停用模块、视图页的插图（`disabled-feature/modules-*.webp`、`views-*.webp`，最宽 960 px 显示，字约 9–10 px，看得清）面包屑里写着 "Plane Design" | 已修（`8ff2d88`）：改为 "Acme Design"（中性的示例名，不写 Nerve，免得像真实的 Nerve 工作区）；前端改动清单的图片一行补上（`f697c05`） |
| M1 | Minor | spec 3.4 说命中样本证明 `github.com/makeplane` 和 `@plane/propel` 照常命中，但没有一个样本单独证明其中任何一个 | 已修（`a041cad`、`f697c05`）：加基点邀请页只含 `github.com/makeplane` 的一行作为命中样本；`@plane/propel` 的样本不存在（见第 2 节），spec 3.4 改写为样本实际证明的内容 |
| M2 | Minor | 已有 `SITE_NAME`，`root.tsx` 的两个图片 `alt`、登录页页头的标题后缀、注册页标题、收集箱标题的回退值仍写字面量 "Nerve" | 已修（`a041cad`），动态里的系统操作者一并改为读 `SITE_NAME` |
| M3 | Minor | README 和前端改动清单第四节说"Nerve 写的前端文件都带 OpenNerve 的声明"，但 `web/packages/api-client` 按 spec 7.2 不加文件头 | 已修（`f697c05`） |
| M4 | Minor | 仍在用的文档里还写着旧包名：M2 交接的 `@plane/utils`、前端改动清单的 `@plane/services` | 已修（`f697c05`），另改前端改动清单第 135 行 |
| M5 | Minor | `public/icons/SOURCES.md` 会被复制进构建，发布在 `/icons/SOURCES.md` 并嵌进 Go 的二进制 | 已修（`a041cad`）：并入 `app/assets/brand/SOURCES.md` |
| M6 | Minor（基点就有） | `window.open` 没有 `noopener`：编辑器链接的点击处理（`custom-link/helpers/clickHandler.ts:53`，用户写的链接）、附件列表、图片下载、全屏弹窗。由别的成员贴的链接或上传的 HTML 附件因此拿得到 `window.opener` | 交收尾（第 7 节），P5 只改了它自己动过的链接 |
| M7 | Minor（基点就有） | `version-number.tsx` 导入整个 `package.json`，客户端包里因此带着全部依赖和版本 | 写进 M8 的版本号交接 |
| M8 | Minor | 导览截图（`onboarding/cycles.webp`、`modules.webp`、`views.webp`）里有真人头像和一位 Plane 联合创始人的名字 | 交收尾，与封面照片的来源一起查 |

评审者还指出：收尾的待办只记在不进仓库的 ledger 里，必须写进本评审记录第 7 节；133 张没有引用的图片（有的写着 "Plane Demo"）交收尾删除；`og:image`、`twitter:image` 是相对地址（基点如此），交 M8 与公开地址一起定。

修复轮合计（`91cfb44..f697c05`）：3 个提交，22 个文件，+39/−43 行。每个提交都通过 `check:types`、孤儿核对、守卫、`alts`、品牌命中和 lint 上限；最后 `make lint-web` 52、`make test-web` 16、`make build-web` 11 个任务通过，`make knip` 为零，vitest 77 个。

### 3.1 修复后的复核（sonnet，范围复核）

修复轮 `91cfb44..f697c05` 之后，范围复核的结论是 **Approved**：I1、I2、M1–M5 全部落实，没有新的发现。复核者对每一项都自己取证：

- **图片**：8 张改过的图都打开看过（深浅两种背景，放大 3 倍看边缘）。复核者自己写了一个无损的像素比较脚本（没有用实现者的）：两张圆章图改动的像素正好在圆章的外接框里（27074、27110 个），两张格子图在 (689,269)–(796,376) 里（4804、5092 个），框外的像素逐字节不变；四张 WebP 改动集中在面包屑的文字框里，框外只有一次有损重新编码的噪点（在高对比的边缘上，改前改后都有，正常大小下看不见）；文件大小变化都在 0.4% 以内；8 个文件改前改后的哈希与实现者的记录相同；圆章的颜色和几何与 `mark.svg` 一致。
- **代码**：5 处 `SITE_NAME` 都是站点名，导入放在原来相邻的组里；`git grep -w "Nerve"` 剩下的都是注释、图形的 `alt`、句子里的文案和 `SITE_NAME` 的定义本身；`alt=""` 在 `workspace-wrapper.tsx` 唯一的 `<img>` 上。
- **守卫**：新的命中样本与基点邀请页第 117 行逐字相同；`keywords: 48 rules, 5 exceptions, no hits.`，`alts` 为 0；`brandcheck.mjs`：当前 607 行 `@makeplane/propel` 都不命中，`@makeplane/propel-extra`、`@plane/propel` 命中。
- **`SOURCES.md`**：`public/icons/SOURCES.md` 已删，`brand/SOURCES.md` 列出全部 9 个品牌文件和 `public/icons/` 的两个图标，构建的 `client/icons/` 里只有两张 PNG。
- **文档**：spec、README、前端改动清单改过的每一句都核对过；`docs/` 里剩下的 `@plane/` 都是历史记录。
- **复跑**（`f697c05`）：品牌命中 3，`symref` 0，`infile-orphans` 11 行都以 `defined now: 0)` 结尾，`dangling` 0，`headers.sh` 没有输出，`docs/` 以外的 `@plane/` 只有 `tools/keywords.json:6`。复核者另外看了 propel 的 25 个插图和约 40 张抽样图片，包括与模块、视图同一版式的停用周期、停用收集箱插图（它们的截图没有工作区名），都没有 Plane。

修复轮之后，控制者在 `f697c05` 的构建上跑了 `brand.mjs` 的全部三组（177 项全部通过；C 组在修复之前的 `7187d70` 构建上失败 9 项：8 张图各一项、`alt` 一项，附录 6.1、6.3）和 P4 的 9 段脚本（403 项全部通过，附录 6.2），并目视了 C 组的截图。

## 4. 控制者的裁定

执行中的裁定都按"能自己定的就自己定"的原则做出：只要符合 SOLID、从根源解决、不打补丁，就不升级给用户。没有一项属于架构级、跨模块或意料之外的高风险。执行前向用户说明过一件事（不阻塞）：`@makeplane/propel` 以编译后的形式按 AGPL 随 Nerve 分发，它的上游仓库不公开；公开发布之前要取得对应源码，交 M8（第 7 节）。

1. **原型只是对照答案**（P3 裁定 1）。每个 Task 逐步执行，报告与原型逐文件比较。Task 1、2 由控制者重放脚本，确认提交就是脚本的输出；Task 4、5、6 的文件与原型逐字节相同；与原型不同的地方都来自执行前的控制者裁定（版权持有者）、跟进提交或实现者发现的原型问题（第 5 节）。
2. **版权持有者是 OpenNerve。** 控制者给架构师的默认值 "Nerve contributors" 与 README 的"Copyright © 2026 OpenNerve"不符，执行前改正，脚本和 plan 一起改（spec 7.2）。`web/` 以外的 Nerve 代码和 `api-client` 不加文件头，以根目录的 `LICENSE` 为准。
3. **退化结构由造成它的 Task 收掉**（P3 裁定 4）。Task 2 的脚本只向上合并，留下两处相邻的同名分组（其中一处把 `./` 的导入标成 `// nerve imports`），由同一位实现者的跟进提交收掉；基点就有的 4 对和其他错标不动，交收尾。
4. **不命中样本引用保留的代码，命中样本可以指向已删的代码**（P4 裁定）。这条规则由删掉或改名那段代码的 Task 执行：`deploy-files`（Task 3 的跟进提交）、`integrations`（Task 5）、`changelog`（Task 6）。原型没有做这三处。`brand` 的命中样本写成改名后的 `@nerve/propel/icons`，让 `docs/` 以外 `@plane/` 的计数保持 6（Task 7）。
5. **一个值只有一个来源。** Task 4 的跟进提交让页面元数据只读 `SITE_DESCRIPTION`、`SITE_KEYWORDS`；修复轮让写着站点名的地方读 `SITE_NAME`。句子里提到 Nerve 的文案仍写字面量，与文案文件一样——那是文字，不是站点名的引用。图形组件的 `alt="Nerve"` 描述的是图形本身，保留。
6. **Plane 收集箱机器人的特殊显示删除**（spec 第 3 节第 9 条）。Nerve 不区分人和智能体，也没有 Plane 的收集箱机器人；系统代为创建的工作项怎样显示交 M4。
7. **不做迁移**（spec 第 1 节）。会话存储键和剪贴板类型只在一个标签页的会话或一次复制粘贴里有效；Task 6 的评审者逐一推演了旧值留在浏览器里的后果：无害，不会循环。
8. **版本号和 propel 的对应源码交 M8。** 版本号随首次发布定；propel 以提交 `0a31b15` 的源码或包里的 source map 为准（缺 `src/internal/variant-props.ts`）。
9. **图片里的品牌由人看，守卫看不见。** 整分支评审证明，170 px 格子的联系表看不出 30 px 的 Logo，spec 2.2 结论 4 那句"全部看过"因此不成立。修复轮先按能看清 20 px 细节的尺寸重看代码引用的全部图片，再逐张修改；圆章改为 Nerve 自己的圆形标记（与方形标记同一几何），面包屑写中性的 "Acme Design"。控制者打开看过每张改过的图片：圆章干净、没有蓝边，网格连续，"Acme Design" 与原来的字一致。`no-projects-dark.png` 卡片外近黑色的 L 形三个格子在 Plane 的原图里就有，不是 Logo，在深色页面上几乎看不见，不改。
10. **导入分组不重排**（P4 裁定 10）。换掉的导入放在原来那一组；只有当原来的组是包名的组、新导入是 `@/` 时才新建 `// components` 组（Task 3 的两个文件），其他导入不动。
11. **控制者的探测发现了评审没有发现的问题**（P3 裁定 11）：
    - 在基点上做反向对照时，B12（收集箱创建者）只靠页面元数据里的 "Plane" 失败，根本没有走到 Plane 为收集箱机器人写的两处特殊显示：列表按邮箱 `intake@plane.so` 判断，工作项的"创建者"按名字里的 `-intake` 判断，而且完整页面只在窗口窄于 768 px 时才画那一块。重写之后两处都在基点上失败（列表的头像字母是 P，"Created by Plane"），在本分支上通过；
    - P4 的核对"每个收藏都有图标"在 Task 6 的构建上失败，原因在脚本：XPath 的 `svg` 在 HTML 文档里匹配不到 SVG 命名空间的元素，这项检查一直是靠页面上方某个 `<img>` 通过的——P4 时是 Task 5 删掉的 "Star us on GitHub" 图片。改用 CSS 选择器、只在这一行里找图标之后，基点和本分支上 5 个收藏都找到自己的 `svg`（附录 6.2）；
    - 整分支评审找到 I1、I2 之后，控制者加了 C 组：取页面实际显示的图片、算哈希、与基点的原图比较，在修复前的构建上 8 张图全部失败，修复后通过。探测在 Task 3、Task 6 之后和修复轮之后各跑一次，并在基点的构建上做了反向对照（附录 6.3）。
12. **基点就有、与品牌无关的问题交收尾**（spec 第 5 节）。实现者和评审者在执行中发现的这类问题都写进第 7 节的收尾清单，不在 P5 顺手修：它们不是 P5 造成的，也不在 P5 的范围里。
13. **执行过程。** 几位实现者各有一次把 `cd` 和只读命令写在一起（`cd … && ls` 之类），一位评审者把只读的正则测试脚本写到了 `/tmp` 而不是 `$P5TMP`；都没有改动仓库或历史。之后的派发都重申了"`cd` 不与任何命令连写"。

## 5. 计划缺陷

计划由原型写成，执行时发现以下缺陷，都已在对应 Task、跟进提交或修复轮中修正：

| Task | 缺陷 | 实际做法 |
|---|---|---|
| 2 | `labels.mjs` 只向上合并，留下两处相邻的同名分组 | 跟进提交 `f32dfe6` |
| 3 | 守卫 `deploy-files` 的不命中样本写着本 Task 删掉的 `public/manifest.json` | 跟进提交 `c7dcca7` |
| 3 | spec、plan 和提交信息说两张加载动画"约 1 MB" | 两张共 1,404,364 字节，文字改为"共约 1.4 MB" |
| 3 | `public/icons/SOURCES.md` 放在会原样发布的目录里 | 修复轮并入 `app/assets/brand/SOURCES.md`（M5） |
| 4 | `root.tsx` 的元数据手写 `SITE_DESCRIPTION` 的原文，`SITE_KEYWORDS` 没有读取方 | 跟进提交 `0118dcd` |
| 4 | 5 处写着站点名的字面量 | 修复轮读 `SITE_NAME`（M2） |
| 5 | 守卫 `integrations` 的不命中样本引用本 Task 删掉的代码，说明也过时 | 在 Task 5 的提交里改 |
| 6 | 守卫 `changelog` 的不命中样本引用本 Task 改名的导入 | 在 Task 6 的提交里改 |
| 7 | `brand` 的命中样本自己带着 `@plane/`，`docs/` 以外的计数是 7 而不是 spec 说的 6；说明写"本地存储键" | 在 Task 7 的提交里改 |
| 7 | `brand` 的第二个否定前瞻以 `\b` 结尾，放过 `@makeplane/propel-extra`；样本没有单独证明两个排除 | 修复轮（Task 7 评审的 Minor、整分支评审 M1） |
| spec 2.2 结论 4 | 联系表的格子太小，8 张图里 Plane 的 Logo 和名字没有被发现 | 修复轮（I1、I2） |
| 浏览器核对（plan 最后一节） | B12 走不到收集箱机器人的特殊显示；没有检查图片内容的项 | 控制者重写 B12，加 C 组 |

## 6. 附录（M1 设计 7.5）

临时核对脚本不进仓库（M1 设计 7.5），全文、假数据、运行命令、断言和输出写在这里。脚本在 `$P5TMP`（`/private/tmp/claude-501/-Users-xiaoruan-project-nerve-project/99d2bc1d-fdaf-4b92-a590-29b89514572b/scratchpad/nerve-p5`）下，跑在 `$P5TMP/probe-app` 里：那是本仓库的一个克隆，检出到被测的提交，执行过 `pnpm install --frozen-lockfile` 和 `make build-web`。做法沿用 P3、P4 评审的附录：node 起一个静态服务器提供 `web/apps/web/build/client`（单页应用的回退到 `index.html`），用 e2e 包里的 Playwright 驱动 Chromium，路径以 `/api/`、`/auth/` 开头的请求全部由脚本里的桩回答；**当前场景没有列出的请求一律算失败**，每个场景还检查"没有未捕获的页面错误"和"控制台没有 `You should call navigate() in a React.useEffect()`"。下面的输出来自修复轮之后 `f697c05` 的构建，反向对照的输出来自基点 `96d8c1d` 的构建（`$P5TMP/base`）。

### 6.1 `brand.mjs`：看得见的页面、行为和图片

#### 1. 目的

核对 P5 改到的每一处在页面上的样子（plan 最后一节"控制者的浏览器核对"，M1 设计 7.5 的 P5 一行）：

- **A 组**（截图给控制者目视，同时断言文字）：登录页的浅色、深色主题（标题 `… - Nerve`、只有一份清单、网站图标是品牌文件、横版标志的字标随主题、"New to Nerve?"、没有条款和隐私、没有客户 Logo）；工作区首页（没有 "Star us on GitHub"）和帮助菜单（Documentation、Keyboard shortcuts、版本，没有 Forum；Documentation 以 `noopener,noreferrer` 打开 Nerve 的仓库）；预渲染的和等待实例配置时的加载动画（是一张脉动的标记，不是 GIF，两种主题相同）；错误页（新的建议、"Go to home"，没有支持邮箱、状态页和 X 账号）；维护页（Nerve 的标题，没有支持团队和邮箱）；创建工作区、新手引导、404、工作区不存在和导览。
- **B 组**（行为）：命令面板的帮助命令（"Open documentation" 打开仓库）；陈旧资源的恢复（刷新一次，第二次失败显示错误页，会话存储里只有 `__nerve_chunk_reload`）；编辑器的复制写入 `text/nerve-editor-html`、没有带 plane 的类型，只带这一种类型的粘贴能把描述放进评论编辑器；个人访问令牌的创建和删除、Webhook 的列表（文案说 Nerve，没有 "Plane Pages"）；收集箱创建者（邮箱 `intake@plane.so`、名字 `zeta-intake` 的用户：列表里是它自己的头像字母 Z，不是 Plane 的 P；窗口窄于 768 px 时工作项的"Created by" 写它的名字，不是 "Plane"）。
- **C 组**（图片，整分支评审之后加入）：对显示 8 张图之一的每个页面，取页面上文件名对得上的每个 `<img>`，按页面拿到的字节算 SHA-256，与基点的原图（`$P5TMP/base` 里的文件）相同即失败；按主题选图的页面深浅各一次；"Workspace not found" 的插图必须是装饰性的（`alt=""`）。

每个场景都检查页面的文字节点和 `title`、`alt`、`aria-label`、`placeholder`、`href`、`content`、`src` 属性里没有 plane（`data:` 地址除外）。

#### 2. 共用部分

`$P5TMP/probe/lib.mjs` 与 P4 的 `$P4TMP/probe-a/lib.mjs` 逐字节相同，全文见 [P4 评审记录](P4-router-native-review.md)附录 6.2。工作项、收集箱的假数据取自 P4 C 组的 `p3data.mjs`、`stubs.mjs`，全文见 P4 评审记录附录 6.4。

#### 3. 脚本全文（`$P5TMP/probe/brand.mjs`）

从架构师的 `probe/visual.mjs` 起步；在原型的构建上调试（`fix1.py`：图标的正则、命令面板的搜索请求、令牌对话框的点击），在基点上做反向对照时重写 B12（`fix2.py`、`fix3.py`），整分支评审之后加 C 组（`fix4.py`、`fix5.py`）。下面是最终的全文：

```js
// One-off (M1/P5 browser checks, plan "控制者的浏览器核对", M1 design 5, 7.5): the pages the brand phase changes, on
// the build of the repository at the current directory. Group A takes screenshots for the controller's visual check
// and asserts the page text; group B checks behaviour (the help links, the palette's help commands, the renamed
// session key and clipboard type, the token and webhook settings, the intake creator). Every scenario also fails on a
// "plane" in any text node or in the title, alt, aria-label, placeholder, href, content or src of any element
// (data: URLs aside), on an unlisted request, an uncaught page error, and React Router's navigate warning (lib.mjs).
// Starts from the architect's visual.mjs; the server, stubs and checks are P4's (lib.mjs, copied), the work-item and
// intake data P4's group C data (../../nerve-p4/probe-c/).
// usage: OUT=<screenshot dir> node brand.mjs [scenario name filter]   (from the repository root of the build)
import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { Reply, WS, anonymousStubs, check, goto, probe, scenario, settle, signedInStubs, visible, workspaceStubs } from "./lib.mjs";
import { ISSUES, P, W, issue, project, users } from "../../nerve-p4/probe-c/p3data.mjs";
import { stubs } from "../../nerve-p4/probe-c/stubs.mjs";

const REPO = "https://github.com/open-nerve/NerveProject";
const out = path.resolve(process.env.OUT ?? "shots");
fs.mkdirSync(out, { recursive: true });
const shot = (page, name) => page.screenshot({ path: path.join(out, `${name}.png`) });

// every text node and listed attribute in the document that says "plane"
const planeInPage = (page) =>
  page.evaluate(() => {
    const found = [];
    const walker = document.createTreeWalker(document.documentElement, NodeFilter.SHOW_TEXT);
    while (walker.nextNode()) if (/plane/i.test(walker.currentNode.nodeValue)) found.push(walker.currentNode.nodeValue.trim());
    for (const el of document.querySelectorAll("*"))
      for (const a of ["title", "alt", "aria-label", "placeholder", "href", "content", "src"]) {
        const v = el.getAttribute(a);
        if (v && /plane/i.test(v) && !/^data:/.test(v)) found.push(`${el.tagName.toLowerCase()}[${a}]=${v}`);
      }
    return found;
  });
const noPlane = async (page, name) => {
  const found = await planeInPage(page);
  check(`${name}: no "plane" in the page`, found.length === 0, JSON.stringify(found.slice(0, 12)));
};
const head = (page) =>
  page.evaluate(() => ({
    title: document.title,
    icons: [...document.querySelectorAll("link[rel*=icon]")].map((l) => l.getAttribute("href")),
    manifests: [...document.querySelectorAll("link[rel=manifest]")].map((l) => l.getAttribute("href")),
  }));
const bodyText = (page) => page.locator("body").innerText();
// C: the pictures' base files (Plane's originals) and what a page shows
const BASE = "/private/tmp/claude-501/-Users-xiaoruan-project-nerve-project/99d2bc1d-fdaf-4b92-a590-29b89514572b/scratchpad/nerve-p5/base/web/apps/web/app/assets";
const baseSha = (rel) => crypto.createHash("sha256").update(fs.readFileSync(`${BASE}/${rel}`)).digest("hex");
// every <img> on the page whose src names `stem`: its src, alt and the SHA-256 of the bytes the page was served
const shownImages = (page, stem) =>
  page.evaluate(async (stem) => {
    const out = [];
    for (const img of document.querySelectorAll("img")) {
      const src = img.getAttribute("src") ?? "";
      if (!src.includes(stem)) continue;
      const buf = await (await fetch(img.src)).arrayBuffer();
      const hash = [...new Uint8Array(await crypto.subtle.digest("SHA-256", buf))].map((b) => b.toString(16).padStart(2, "0")).join("");
      out.push({ src, alt: img.getAttribute("alt"), sha: hash });
    }
    return out;
  }, stem);
// the page shows the picture `rel` (by its file stem), and not Plane's original bytes
const checkPicture = async (page, name, rel) => {
  const stem = rel.split("/").pop().replace(/\.[a-z]+$/, "");
  let shown = [];
  for (let i = 0; i < 20 && !shown.length; i++) {
    shown = await shownImages(page, stem);
    if (!shown.length) await page.waitForTimeout(500);
  }
  check(`${name}: shows ${stem}`, shown.length > 0, JSON.stringify(shown));
  const original = baseSha(rel);
  check(`${name}: ${stem} is not Plane's original`, shown.length > 0 && shown.every((i) => i.sha !== original), JSON.stringify(shown.map((i) => i.sha.slice(0, 16))));
  return shown;
};
// window.open is recorded instead of opening a tab: [url, target, features] of every call
const recordOpen = (page) =>
  page.addInitScript(() => {
    window.__opened = [];
    window.open = (...args) => {
      window.__opened.push(args);
      return null;
    };
  });
const opened = (page) => page.evaluate(() => window.__opened ?? []);

probe(async () => {
  // ---------------------------------------------------------------- A: visual, with text assertions
  // A1. The sign-in page, light and dark: the lockup's wordmark follows the theme; no terms, privacy, customer logos.
  for (const theme of ["light", "dark"]) {
    await scenario(`A1 sign-in (${theme})`, anonymousStubs(), async (page) => {
      if (theme === "dark") await page.addInitScript(() => localStorage.setItem("theme", "dark"));
      await goto(page, "/");
      check(`A1 sign-in (${theme}): the form shows`, await visible(page.locator("form input#email"), 20000));
      await page.waitForTimeout(800);
      const h = await head(page);
      console.log(`  ${JSON.stringify(h)}`);
      check(`A1 sign-in (${theme}): the title is "… - Nerve"`, /(^|\s-\s)Nerve$/.test(h.title), h.title);
      check(`A1 sign-in (${theme}): one manifest, site.webmanifest.json`, h.manifests.length === 1 && h.manifests[0].endsWith("site.webmanifest.json"), JSON.stringify(h.manifests));
      check(`A1 sign-in (${theme}): the icons are the brand files`, h.icons.length > 0 && h.icons.every((i) => /\/assets\/(favicon|icon-\d+x\d+)/.test(i)), JSON.stringify(h.icons));
      const lockup = page.locator("img[alt=Nerve]:visible");
      const src = (await lockup.count()) === 1 ? await lockup.getAttribute("src") : `${await lockup.count()} visible`;
      check(`A1 sign-in (${theme}): the visible lockup is the ${theme === "dark" ? "light" : "dark"} wordmark`,
        theme === "dark" ? /lockup-on-dark/.test(src) : /lockup-(?!on-dark)|lockup\./.test(src) && !/on-dark/.test(src), src);
      const text = await bodyText(page);
      check(`A1 sign-in (${theme}): "New to Nerve?" and no terms, privacy or "Join 10,000+"`,
        /New to Nerve\?/.test(text) && !/terms|privacy|10,000/i.test(text), text.slice(0, 400));
      await noPlane(page, `A1 sign-in (${theme})`);
      await shot(page, `sign-in-${theme}`);
    });
  }

  // A2, A3. The workspace home, and the help menu: Documentation opens the repository without an opener; no Forum.
  await scenario("A2 workspace and A3 help menu", stubs(), async (page) => {
    await recordOpen(page);
    await goto(page, `/${WS}`);
    await settle(page, `/${WS}`);
    await page.waitForTimeout(1500);
    const text = await bodyText(page);
    check("A2 workspace: no \"Star us on GitHub\"", !/star us/i.test(text), text.slice(0, 300));
    await noPlane(page, "A2 workspace");
    await shot(page, "workspace");
    // the help button is the top bar's icon button whose menu holds "Keyboard shortcuts"
    const buttons = page.locator("button:has(svg)");
    let open = false;
    for (let i = 0; i < (await buttons.count()) && !open; i++) {
      const b = buttons.nth(i);
      if (!(await b.isVisible())) continue;
      const box = await b.boundingBox();
      if (!box || box.y > 80 || box.x < 1000) continue;
      await b.click();
      open = await visible(page.getByText("Keyboard shortcuts"), 1000);
      if (!open) await page.keyboard.press("Escape");
    }
    check("A3 help menu: opens", open);
    const items = (await page.locator("[role=menuitem]").allInnerTexts()).map((t) => t.trim()).filter(Boolean);
    console.log(`  help menu: ${JSON.stringify(items)}`);
    check("A3 help menu: Documentation, Keyboard shortcuts and the version, no Forum",
      items.some((t) => /^Documentation/.test(t)) && items.some((t) => /Keyboard shortcuts/.test(t)) &&
        /Version/.test(await bodyText(page)) && !items.some((t) => /forum/i.test(t)), JSON.stringify(items));
    await noPlane(page, "A3 help menu");
    await shot(page, "help-menu");
    await page.getByRole("menuitem", { name: /Documentation/ }).first().click();
    await page.waitForTimeout(500);
    const calls = await opened(page);
    check("A3 help menu: Documentation opens the repository, without an opener",
      calls.length === 1 && calls[0][0] === REPO && /noopener/.test(calls[0][2] ?? ""), JSON.stringify(calls));
  });

  // A4. The loaders: prerendered (no script runs) and while the instance request is pending, in both themes: the mark.
  await scenario("A4 loader, prerendered", {}, async (page) => {
    await page.context().route("**/*.js", (route) => route.abort());
    await goto(page, "/");
    await page.waitForTimeout(500);
    const mark = page.locator("img[alt=Nerve]");
    check("A4 loader, prerendered: the pulsing mark, not a GIF",
      (await visible(mark, 3000)) && /animate-pulse/.test(await mark.getAttribute("class")) && !/\.gif/.test(await mark.getAttribute("src")),
      `${await mark.getAttribute("class").catch(() => "")} ${await mark.getAttribute("src").catch(() => "none")}`);
    await noPlane(page, "A4 loader, prerendered");
    await shot(page, "loader-prerendered");
  });
  const pendingSrc = {};
  for (const theme of ["light", "dark"]) {
    await scenario(`A4 loader, instance pending (${theme})`, anonymousStubs(), async (page) => {
      if (theme === "dark") await page.addInitScript(() => localStorage.setItem("theme", "dark"));
      // a page route wins over the scenario's context route; it never answers, so the instance request stays pending
      await page.route("**/api/instances/", () => {});
      await goto(page, "/");
      await page.waitForTimeout(1500);
      const mark = page.locator("img[alt=Nerve]");
      check(`A4 loader, instance pending (${theme}): the mark shows`, await visible(mark, 3000));
      pendingSrc[theme] = await mark.getAttribute("src").catch(() => null);
      await noPlane(page, `A4 loader, instance pending (${theme})`);
      await shot(page, `loader-pending-${theme}`);
    });
  }
  if ("light" in pendingSrc && "dark" in pendingSrc)
    check("A4 the loader is the same image in both themes", pendingSrc.light && pendingSrc.light === pendingSrc.dark, JSON.stringify(pendingSrc));

  // A5. The error page (a render error: the home's recent visits answer an object, and the widget calls .filter).
  await scenario("A5 error page", stubs({ [`GET ${W}/recent-visits/`]: {} }), async (page) => {
    await goto(page, `/${WS}`);
    const shown = await visible(page.getByText("Looks like something went wrong"), 20000);
    const text = await bodyText(page);
    check("A5 error page: the message, the new advice and \"Go to home\"",
      shown && /Try refreshing the page\. If the problem persists, contact your administrator\./.test(text) && /Go to home/i.test(text), text.slice(0, 400));
    check("A5 error page: no support email, status page or X account",
      !/support@|status\.|twitter|x\.com|we track these errors/i.test(await page.content()), "found one");
    await noPlane(page, "A5 error page");
    await shot(page, "error");
  });

  // A6. The maintenance page.
  await scenario("A6 maintenance", { "GET /api/instances/": new Reply(500, {}) }, async (page) => {
    await goto(page, "/");
    const shown = await visible(page.getByText("Looks like Nerve didn't start up correctly!"), 20000);
    const text = await bodyText(page);
    check("A6 maintenance: the Nerve heading, no support team or email", shown && !/support team|mailto|support@/i.test(text + (await page.content())), text.slice(0, 300));
    await noPlane(page, "A6 maintenance");
    await shot(page, "maintenance");
  });

  // A7. More screenshots: create-workspace, onboarding, 404, workspace not found, the product tour's first screen.
  await scenario("A7 create workspace", { ...signedInStubs({ workspaces: [] }), "GET /api/users/me/workspaces/invitations/": [] }, async (page) => {
    await goto(page, "/create-workspace");
    check("A7 create workspace: the mark shows", await visible(page.locator("img[alt=Nerve]"), 20000));
    await page.waitForTimeout(800);
    await noPlane(page, "A7 create workspace");
    await shot(page, "create-workspace");
  });
  await scenario("A7 onboarding", { ...signedInStubs({ onboarded: false, workspaces: [] }), "GET /api/users/me/workspaces/invitations/": [] }, async (page) => {
    await goto(page, "/onboarding");
    check("A7 onboarding: the lockup shows", await visible(page.locator("img[alt=Nerve]:visible"), 20000));
    await page.waitForTimeout(800);
    await noPlane(page, "A7 onboarding");
    await shot(page, "onboarding");
  });
  await scenario("A7 not found", stubs(), async (page) => {
    await goto(page, `/${WS}/no-such-page/x`);
    check("A7 not found: shows", await visible(page.getByText("Sorry, the page you are looking for cannot be found."), 20000));
    await noPlane(page, "A7 not found");
    await shot(page, "not-found");
  });
  await scenario("A7 workspace not found", stubs({
    "GET /api/workspaces/nope/states/": new Reply(404, { error: "Workspace not found" }),
    "GET /api/workspaces/nope/user-properties/": new Reply(404, { error: "Workspace not found" }),
  }), async (page) => {
    await goto(page, "/nope");
    check("A7 workspace not found: shows", await visible(page.getByText("Workspace not found").first(), 20000));
    await page.waitForTimeout(800);
    await noPlane(page, "A7 workspace not found");
    await shot(page, "workspace-not-found");
  });
  await scenario("A7 product tour", stubs({
    "GET /api/users/me/profile/": { id: "pr1", user: "u1", language: "en", is_onboarded: true, is_tour_completed: false, theme: {} },
  }), async (page) => {
    await goto(page, `/${WS}`);
    const tour = await visible(page.locator("img[alt=Nerve]:visible").first(), 20000);
    await page.waitForTimeout(1000);
    console.log(`  tour visible: ${tour}; ${(await bodyText(page)).slice(0, 200).replace(/\n/g, " | ")}`);
    check("A7 product tour: the lockup shows", tour);
    await noPlane(page, "A7 product tour");
    await shot(page, "tour");
  });

  // ---------------------------------------------------------------- B: behaviour
  // B8. The palette's help commands: documentation and bug reports go to the repository; no forum.
  const noResults = { results: { workspace: [], project: [], issue: [], cycle: [], module: [], issue_view: [] } };
  await scenario("B8 palette help commands", stubs({ "GET /api/users/me/workspaces/invitations/": [], [`GET ${W}/search/`]: noResults }), async (page) => {
    await recordOpen(page);
    await goto(page, `/${WS}/projects`);
    await settle(page, `/${WS}/projects`);
    await page.waitForTimeout(1000);
    const offered = async (query) => {
      await page.keyboard.press("ControlOrMeta+k");
      await page.locator("[cmdk-input]").fill(query);
      await page.waitForTimeout(500);
      const items = (await page.locator("[cmdk-item]").allInnerTexts()).map((t) => t.trim());
      return items;
    };
    const docs = await offered("documentation");
    check("B8 \"Open documentation\" is offered", docs.some((t) => /Open documentation/i.test(t)), JSON.stringify(docs));
    await page.locator("[cmdk-item]", { hasText: /Open documentation/i }).first().click();
    await page.waitForTimeout(500);
    await page.keyboard.press("Escape");
    const bug = await offered("bug");
    check("B8 \"Report a bug\" is offered", bug.some((t) => /Report a bug/i.test(t)), JSON.stringify(bug));
    await page.locator("[cmdk-item]", { hasText: /Report a bug/i }).first().click();
    await page.waitForTimeout(500);
    await page.keyboard.press("Escape");
    const forum = await offered("forum");
    check("B8 no forum command", !forum.some((t) => /forum/i.test(t)), JSON.stringify(forum));
    await page.keyboard.press("Escape");
    const calls = await opened(page);
    console.log(`  window.open calls: ${JSON.stringify(calls)}`);
    check("B8 documentation opens the repository, bug reports its issues", calls.length === 2 && calls[0][0] === REPO && calls[1][0] === `${REPO}/issues`, JSON.stringify(calls));
    await noPlane(page, "B8 palette");
  });

  // B9. A stale asset: the page reloads once under __nerve_chunk_reload, the second failure shows the error page.
  await scenario("B9 stale asset", { ...signedInStubs(), ...workspaceStubs() }, async (page) => {
    await page.context().route("**/assets/api-tokens-*.js", (route) => route.abort());
    let loads = 0;
    page.on("load", () => loads++);
    await goto(page, "/settings/profile/api-tokens");
    const shown = await visible(page.getByText("Looks like something went wrong"), 20000);
    const keys = await page.evaluate(() => Object.keys(sessionStorage));
    console.log(`  page loads: ${loads}; session storage keys: ${JSON.stringify(keys)}`);
    check("B9 stale asset: the error page shows after one reload", shown && loads === 2, `shown ${shown}, loads ${loads}`);
    check("B9 stale asset: remembered under __nerve_chunk_reload only", keys.includes("__nerve_chunk_reload") && !keys.some((k) => /plane/i.test(k)), JSON.stringify(keys));
    await noPlane(page, "B9 stale asset");
  });

  // B10. The editor's clipboard type: copying the description writes text/nerve-editor-html; a paste that carries only
  // that type puts the content into the comment editor (the reader and the writers agree on the key).
  await scenario("B10 editor clipboard", stubs(), async (page) => {
    await goto(page, `/${WS}/browse/PRB-1`);
    const desc = page.locator("#editor-container-i1 .ProseMirror");
    check("B10 the description editor shows", await visible(desc, 20000));
    await desc.click();
    await page.waitForTimeout(300);
    await page.keyboard.press("ControlOrMeta+a");
    const copied = await desc.evaluate((el) => {
      const dt = new DataTransfer();
      el.dispatchEvent(new ClipboardEvent("copy", { clipboardData: dt, bubbles: true, cancelable: true }));
      const types = [...dt.types];
      const branded = types.find((t) => /-editor-html$/.test(t));
      return { types, branded, html: branded ? dt.getData(branded) : "" };
    });
    console.log(`  copied types: ${JSON.stringify(copied.types)}`);
    check("B10 copy writes text/nerve-editor-html and no plane type", copied.types.includes("text/nerve-editor-html") && !copied.types.some((t) => /plane/i.test(t)), JSON.stringify(copied.types));
    const box = page.locator("#editor-container-add_comment_i1 .ProseMirror");
    check("B10 the comment editor shows", await visible(box, 10000));
    await box.click();
    await page.waitForTimeout(300);
    await box.evaluate((el, html) => {
      const dt = new DataTransfer();
      dt.setData("text/nerve-editor-html", html);
      el.dispatchEvent(new ClipboardEvent("paste", { clipboardData: dt, bubbles: true, cancelable: true }));
    }, copied.html || "<p>nothing was copied</p>");
    await page.waitForTimeout(500);
    check("B10 a paste carrying only text/nerve-editor-html puts the description into the comment editor",
      /Alpha description text/.test(await box.innerText()), await box.innerText());
    await noPlane(page, "B10 editor clipboard");
  });

  // B11. The token and webhook settings stay reachable, and their copy says Nerve.
  const TOKEN = {
    id: "t1", label: "Probe token", description: "", expired_at: null, is_active: true, last_used: null,
    created_at: "2026-09-01T00:00:00Z", updated_at: "2026-09-01T00:00:00Z", created_by: "u1", updated_by: "u1",
    user: "u1", user_type: 0, workspace: "w1",
  };
  await scenario("B11 tokens", {
    ...signedInStubs(), ...workspaceStubs(),
    "GET /api/users/api-tokens/": [TOKEN],
    "POST /api/users/api-tokens/": { ...TOKEN, id: "t2", label: "New token", token: "nerve_secret_probe" },
    "DELETE /api/users/api-tokens/t1": new Reply(204),
  }, async (page) => {
    await goto(page, "/settings/profile/api-tokens");
    check("B11 the token page lists the token", await visible(page.getByText("Probe token"), 20000));
    await page.getByRole("button", { name: /Add access token/i }).first().click();
    const dialog = page.getByRole("dialog");
    await dialog.locator("input").first().fill("New token");
    await dialog.getByText("Never expires").click();
    await dialog.getByRole("button", { name: "Generate token" }).click();
    const copyNote = await visible(page.getByText(/Copy and save this secret key/i), 10000);
    const text = await bodyText(page);
    check("B11 the new-token note: \"Copy and save this secret key\", no \"Plane Pages\"", copyNote && !/Plane Pages/.test(text), text.slice(0, 400));
    await noPlane(page, "B11 tokens (created)");
    await shot(page, "token-created");
  });
  await scenario("B11 token delete", {
    ...signedInStubs(), ...workspaceStubs(),
    "GET /api/users/api-tokens/": [TOKEN],
  }, async (page) => {
    await goto(page, "/settings/profile/api-tokens");
    await visible(page.getByText("Probe token"), 20000);
    // the row's delete button shows on hover
    const row = page.locator("div.group", { hasText: "Probe token" }).first();
    await row.hover();
    await row.locator("button").first().click();
    const dialogShown = await visible(page.getByText(/Delete access token/i), 3000);
    const text = await bodyText(page);
    console.log(`  delete dialog shown: ${dialogShown}`);
    check("B11 the delete dialog says \"access to Nerve data\"", dialogShown && /access to Nerve data/.test(text), text.slice(0, 400));
    await noPlane(page, "B11 token delete");
  });
  await scenario("B11 webhooks", stubs({
    [`GET ${W}/webhooks/`]: [{ id: "wh1", url: "https://hooks.example.com/probe", is_active: true, project: true, issue: true, module: false, cycle: false, issue_comment: false, created_at: "2026-09-01T00:00:00Z", updated_at: "2026-09-01T00:00:00Z" }],
  }), async (page) => {
    await goto(page, `/${WS}/settings/webhooks`);
    check("B11 the webhook page lists the webhook", await visible(page.getByText("hooks.example.com/probe"), 20000));
    await noPlane(page, "B11 webhooks");
  });


  // ---------------------------------------------------------------- C: pictures (the final review's I1, I2)
  await scenario("C13 workspace not found picture", stubs({
    "GET /api/workspaces/nope/states/": new Reply(404, { error: "Workspace not found" }),
    "GET /api/workspaces/nope/user-properties/": new Reply(404, { error: "Workspace not found" }),
  }), async (page) => {
    await goto(page, "/nope");
    check("C13 workspace not found: shows", await visible(page.getByText("Workspace not found").first(), 20000));
    const shown = await checkPicture(page, "C13 workspace not found", "workspace/workspace-not-available.png");
    check("C13 the picture is decorative (alt=\"\")", shown.length > 0 && shown.every((i) => i.alt === ""), JSON.stringify(shown.map((i) => i.alt)));
    await shot(page, "c-workspace-not-found");
  });
  const disabledConfig = { enable_signup: true, is_workspace_creation_disabled: true, file_size_limit: 5242880, is_self_managed: true };
  await scenario("C14 workspace creation disabled picture", {
    ...signedInStubs({ workspaces: [] }),
    "GET /api/users/me/workspaces/invitations/": [],
    "GET /api/instances/": { config: disabledConfig },
  }, async (page) => {
    await goto(page, "/create-workspace");
    await checkPicture(page, "C14 creation disabled", "workspace/workspace-creation-disabled.png");
    await shot(page, "c-creation-disabled");
  });
  for (const theme of ["light", "dark"]) {
    for (const [feature, flag, path] of [["modules", "module_view", "modules"], ["views", "issue_views_view", "views"]]) {
      const off = { ...project, [flag]: false };
      await scenario(`C15 ${feature} disabled picture (${theme})`, stubs({ [`GET ${P}/`]: off, [`GET ${W}/projects/`]: [off] }), async (page) => {
        if (theme === "dark") await page.addInitScript(() => localStorage.setItem("theme", "dark"));
        await goto(page, `/${WS}/projects/p1/${path}`);
        await checkPicture(page, `C15 ${feature} disabled (${theme})`, `empty-state/disabled-feature/${feature}-${theme}.webp`);
        await noPlane(page, `C15 ${feature} disabled (${theme})`);
        await shot(page, `c-${feature}-disabled-${theme}`);
      });
    }
    // with a project the layout sends settings/projects on to the first project's settings: no projects here
    await scenario(`C16 no projects picture (${theme})`, stubs({ [`GET ${W}/projects/`]: [], "GET /api/users/me/workspaces/probe-ws/project-roles/": {} }), async (page) => {
      if (theme === "dark") await page.addInitScript(() => localStorage.setItem("theme", "dark"));
      await goto(page, `/${WS}/settings/projects`);
      await checkPicture(page, `C16 no projects (${theme})`, `empty-state/project-settings/no-projects-${theme}.png`);
      await shot(page, `c-no-projects-${theme}`);
    });
  }

  // B12. What Plane showed as its intake bot shows as an ordinary creator. Plane keyed the intake list's avatar on the
  // creator's email (intake@plane.so: its own avatar, letter P) and the work item's "Created by" on a name with
  // "-intake" (the text "Plane", no avatar); this creator has both, and its own letter is Z.
  const bot = { id: "u9", email: "intake@plane.so", display_name: "zeta-intake", first_name: "Zeta", last_name: "Bot", avatar_url: "", is_active: true };
  const members = [...Object.values(users), bot].map((member, i) => ({ id: `wm${i + 1}`, member, role: 20, is_active: true, created_at: "2026-09-01T00:00:00Z" }));
  const intakeProject = { ...project, inbox_view: true };
  const i9 = issue("i9", 9, "Intake item iota", { created_by: "u9" });
  const inboxIssue = { id: "ii9", status: -2, issue: i9, snoozed_till: null, duplicate_to: null, source: "IN_APP", created_by: "u9" };
  await scenario("B12 intake creator (list)", stubs({
    [`GET ${P}/`]: intakeProject,
    [`GET ${W}/projects/`]: [intakeProject],
    [`GET ${W}/members/`]: members,
    [`GET ${P}/inbox-issues/`]: { next_cursor: "10:1:0", prev_cursor: "10:-1:1", next_page_results: false, prev_page_results: false, total_count: 1, count: 1, total_pages: 1, total_results: 1, results: [inboxIssue] },
    [`GET ${P}/inbox-issues/i9/`]: inboxIssue,
    [`GET ${P}/intake-work-items/i9/description-versions/`]: { next_page_results: false, prev_page_results: false, results: [], total_pages: 0, cursor: "", next_cursor: null, prev_cursor: null, page_count: 0 },
    [`GET ${P}/issues/i9/reactions/`]: [],
    [`GET ${P}/issues/i9/history/`]: [],
    [`GET /api/assets/v2/workspaces/probe-ws/projects/p1/issues/i9/attachments/`]: [],
  }), async (page) => {
    await goto(page, "/probe-ws/projects/p1/intake?currentTab=open&inboxIssueId=i9");
    const item = page.locator("a", { hasText: "Intake item iota" }).first();
    check("B12 the intake item is listed", await visible(item, 20000));
    await page.waitForTimeout(1500);
    // the single capital letters in the list item: the avatars' fallbacks
    const letters = await item.evaluate((el) =>
      [...el.querySelectorAll("*")].filter((e) => !e.children.length && /^[A-Z]$/.test(e.textContent.trim())).map((e) => e.textContent.trim()));
    check("B12 the intake list shows the creator's own avatar (Z), not Plane's (P)", letters.includes("Z") && !letters.includes("P"), JSON.stringify(letters));
    await noPlane(page, "B12 intake creator (list)");
    await shot(page, "intake");
  });
  const botItem = { ...ISSUES[0], created_by: "u9" };
  await scenario("B12 intake creator (work item)", stubs({
    [`GET ${W}/members/`]: members,
    [`GET ${P}/issues/i1/`]: botItem,
    [`GET ${W}/work-items/PRB-1/`]: botItem,
  }), async (page) => {
    // below 768px the page draws its properties under the description, where Plane had its "-intake" branch
    await page.setViewportSize({ width: 700, height: 1400 });
    await goto(page, `/${WS}/browse/PRB-1`);
    check("B12 the work item opens", await visible(page.locator("#editor-container-i1 .ProseMirror"), 20000));
    await page.waitForTimeout(1500);
    // the visible "Created by" label's row: the nearest ancestor that holds more than the label
    const row = await page.evaluate(() => {
      const label = [...document.querySelectorAll("*")].find(
        (e) => !e.children.length && e.textContent.trim() === "Created by" && e.getClientRects().length);
      let el = label;
      while (el && el.innerText.trim() === "Created by") el = el.parentElement;
      return el ? el.innerText.replace(/\s+/g, " ").trim() : null;
    });
    check("B12 the work item's \"Created by\" names the creator, not Plane", /zeta-intake/.test(row ?? "") && !/plane/i.test(row), JSON.stringify(row));
    await noPlane(page, "B12 intake creator (work item)");
    await shot(page, "intake-work-item");
  });
});
```

#### 4. 运行命令

```bash
cd $P5TMP/probe-app   # 检出到 f697c05，已 make build-web
OUT=$P5TMP/shots/probe-final node $P5TMP/probe/brand.mjs
cd $P5TMP/base        # 基点 96d8c1d，已 make build-web（反向对照，6.3）
OUT=$P5TMP/shots/probe-base-final node $P5TMP/probe/brand.mjs
```

截图由控制者目视：登录页深浅两种主题、加载动画、导览、创建工作区、维护页、错误页、帮助菜单、收集箱列表、窄窗口的工作项、令牌、C 组的全部页面（工作区不存在、禁止创建工作区、停用的模块和视图、没有项目的项目设置）。

#### 5. 输出（`f697c05` 的构建）

```

== A1 sign-in (light)
PASS A1 sign-in (light): the form shows
  {"title":"Sign in - Nerve","icons":["/assets/favicon-32x32-USpJWITc.png","/assets/favicon-16x16-D0oeb8Jw.png","/assets/favicon-BvJTCFNT.ico","/assets/icon-512x512-DomL59Hd.png","/assets/icon-180x180-EzizVSsG.png","/assets/icon-512x512-DomL59Hd.png"],"manifests":["/site.webmanifest.json"]}
PASS A1 sign-in (light): the title is "… - Nerve"
PASS A1 sign-in (light): one manifest, site.webmanifest.json
PASS A1 sign-in (light): the icons are the brand files
PASS A1 sign-in (light): the visible lockup is the dark wordmark
PASS A1 sign-in (light): "New to Nerve?" and no terms, privacy or "Join 10,000+"
PASS A1 sign-in (light): no "plane" in the page
PASS A1 sign-in (light): no request outside the scenario's list
PASS A1 sign-in (light): no uncaught page error
PASS A1 sign-in (light): no "call navigate() in a React.useEffect()" warning

== A1 sign-in (dark)
PASS A1 sign-in (dark): the form shows
  {"title":"Sign in - Nerve","icons":["/assets/favicon-32x32-USpJWITc.png","/assets/favicon-16x16-D0oeb8Jw.png","/assets/favicon-BvJTCFNT.ico","/assets/icon-512x512-DomL59Hd.png","/assets/icon-180x180-EzizVSsG.png","/assets/icon-512x512-DomL59Hd.png"],"manifests":["/site.webmanifest.json"]}
PASS A1 sign-in (dark): the title is "… - Nerve"
PASS A1 sign-in (dark): one manifest, site.webmanifest.json
PASS A1 sign-in (dark): the icons are the brand files
PASS A1 sign-in (dark): the visible lockup is the light wordmark
PASS A1 sign-in (dark): "New to Nerve?" and no terms, privacy or "Join 10,000+"
PASS A1 sign-in (dark): no "plane" in the page
PASS A1 sign-in (dark): no request outside the scenario's list
PASS A1 sign-in (dark): no uncaught page error
PASS A1 sign-in (dark): no "call navigate() in a React.useEffect()" warning

== A2 workspace and A3 help menu
  (at /probe-ws)
PASS A2 workspace: no "Star us on GitHub"
PASS A2 workspace: no "plane" in the page
PASS A3 help menu: opens
  help menu: ["Documentation","Keyboard shortcuts"]
PASS A3 help menu: Documentation, Keyboard shortcuts and the version, no Forum
PASS A3 help menu: no "plane" in the page
PASS A3 help menu: Documentation opens the repository, without an opener
PASS A2 workspace and A3 help menu: no request outside the scenario's list
PASS A2 workspace and A3 help menu: no uncaught page error
PASS A2 workspace and A3 help menu: no "call navigate() in a React.useEffect()" warning

== A4 loader, prerendered
PASS A4 loader, prerendered: the pulsing mark, not a GIF
PASS A4 loader, prerendered: no "plane" in the page
PASS A4 loader, prerendered: no request outside the scenario's list
PASS A4 loader, prerendered: no uncaught page error
PASS A4 loader, prerendered: no "call navigate() in a React.useEffect()" warning

== A4 loader, instance pending (light)
PASS A4 loader, instance pending (light): the mark shows
PASS A4 loader, instance pending (light): no "plane" in the page
PASS A4 loader, instance pending (light): no request outside the scenario's list
PASS A4 loader, instance pending (light): no uncaught page error
PASS A4 loader, instance pending (light): no "call navigate() in a React.useEffect()" warning

== A4 loader, instance pending (dark)
PASS A4 loader, instance pending (dark): the mark shows
PASS A4 loader, instance pending (dark): no "plane" in the page
PASS A4 loader, instance pending (dark): no request outside the scenario's list
PASS A4 loader, instance pending (dark): no uncaught page error
PASS A4 loader, instance pending (dark): no "call navigate() in a React.useEffect()" warning
PASS A4 the loader is the same image in both themes

== A5 error page
PASS A5 error page: the message, the new advice and "Go to home"
PASS A5 error page: no support email, status page or X account
PASS A5 error page: no "plane" in the page
PASS A5 error page: no request outside the scenario's list
PASS A5 error page: no uncaught page error
PASS A5 error page: no "call navigate() in a React.useEffect()" warning

== A6 maintenance
PASS A6 maintenance: the Nerve heading, no support team or email
PASS A6 maintenance: no "plane" in the page
PASS A6 maintenance: no request outside the scenario's list
PASS A6 maintenance: no uncaught page error
PASS A6 maintenance: no "call navigate() in a React.useEffect()" warning

== A7 create workspace
PASS A7 create workspace: the mark shows
PASS A7 create workspace: no "plane" in the page
PASS A7 create workspace: no request outside the scenario's list
PASS A7 create workspace: no uncaught page error
PASS A7 create workspace: no "call navigate() in a React.useEffect()" warning

== A7 onboarding
PASS A7 onboarding: the lockup shows
PASS A7 onboarding: no "plane" in the page
PASS A7 onboarding: no request outside the scenario's list
PASS A7 onboarding: no uncaught page error
PASS A7 onboarding: no "call navigate() in a React.useEffect()" warning

== A7 not found
PASS A7 not found: shows
PASS A7 not found: no "plane" in the page
PASS A7 not found: no request outside the scenario's list
PASS A7 not found: no uncaught page error
PASS A7 not found: no "call navigate() in a React.useEffect()" warning

== A7 workspace not found
PASS A7 workspace not found: shows
PASS A7 workspace not found: no "plane" in the page
PASS A7 workspace not found: no request outside the scenario's list
PASS A7 workspace not found: no uncaught page error
PASS A7 workspace not found: no "call navigate() in a React.useEffect()" warning

== A7 product tour
  tour visible: true; P | Probe WS | P | Projects | New work item |  | Home |  | Your work |  | Drafts |  | Workspace |  | Projects |  | Views |  | Archives |  | Projects | 🚀 |  | Probe Project |  | Projects | New work item |  | Home |  | Your work |  | Drafts |  | Workspace |  | Projects |  | Vi
PASS A7 product tour: the lockup shows
PASS A7 product tour: no "plane" in the page
PASS A7 product tour: no request outside the scenario's list
PASS A7 product tour: no uncaught page error
PASS A7 product tour: no "call navigate() in a React.useEffect()" warning

== B8 palette help commands
  (at /probe-ws/projects)
PASS B8 "Open documentation" is offered
PASS B8 "Report a bug" is offered
PASS B8 no forum command
  window.open calls: [["https://github.com/open-nerve/NerveProject","_blank","noopener,noreferrer"],["https://github.com/open-nerve/NerveProject/issues","_blank","noopener,noreferrer"]]
PASS B8 documentation opens the repository, bug reports its issues
PASS B8 palette: no "plane" in the page
PASS B8 palette help commands: no request outside the scenario's list
PASS B8 palette help commands: no uncaught page error
PASS B8 palette help commands: no "call navigate() in a React.useEffect()" warning

== B9 stale asset
  page loads: 2; session storage keys: ["__nerve_chunk_reload"]
PASS B9 stale asset: the error page shows after one reload
PASS B9 stale asset: remembered under __nerve_chunk_reload only
PASS B9 stale asset: no "plane" in the page
PASS B9 stale asset: no request outside the scenario's list
PASS B9 stale asset: no uncaught page error
PASS B9 stale asset: no "call navigate() in a React.useEffect()" warning

== B10 editor clipboard
PASS B10 the description editor shows
  copied types: ["text/plain","text/html","text/nerve-editor-html"]
PASS B10 copy writes text/nerve-editor-html and no plane type
PASS B10 the comment editor shows
PASS B10 a paste carrying only text/nerve-editor-html puts the description into the comment editor
PASS B10 editor clipboard: no "plane" in the page
PASS B10 editor clipboard: no request outside the scenario's list
PASS B10 editor clipboard: no uncaught page error
PASS B10 editor clipboard: no "call navigate() in a React.useEffect()" warning

== B11 tokens
PASS B11 the token page lists the token
PASS B11 the new-token note: "Copy and save this secret key", no "Plane Pages"
PASS B11 tokens (created): no "plane" in the page
PASS B11 tokens: no request outside the scenario's list
PASS B11 tokens: no uncaught page error
PASS B11 tokens: no "call navigate() in a React.useEffect()" warning

== B11 token delete
  delete dialog shown: true
PASS B11 the delete dialog says "access to Nerve data"
PASS B11 token delete: no "plane" in the page
PASS B11 token delete: no request outside the scenario's list
PASS B11 token delete: no uncaught page error
PASS B11 token delete: no "call navigate() in a React.useEffect()" warning

== B11 webhooks
PASS B11 the webhook page lists the webhook
PASS B11 webhooks: no "plane" in the page
PASS B11 webhooks: no request outside the scenario's list
PASS B11 webhooks: no uncaught page error
PASS B11 webhooks: no "call navigate() in a React.useEffect()" warning

== C13 workspace not found picture
PASS C13 workspace not found: shows
PASS C13 workspace not found: shows workspace-not-available
PASS C13 workspace not found: workspace-not-available is not Plane's original
PASS C13 the picture is decorative (alt="")
PASS C13 workspace not found picture: no request outside the scenario's list
PASS C13 workspace not found picture: no uncaught page error
PASS C13 workspace not found picture: no "call navigate() in a React.useEffect()" warning

== C14 workspace creation disabled picture
PASS C14 creation disabled: shows workspace-creation-disabled
PASS C14 creation disabled: workspace-creation-disabled is not Plane's original
PASS C14 workspace creation disabled picture: no request outside the scenario's list
PASS C14 workspace creation disabled picture: no uncaught page error
PASS C14 workspace creation disabled picture: no "call navigate() in a React.useEffect()" warning

== C15 modules disabled picture (light)
PASS C15 modules disabled (light): shows modules-light
PASS C15 modules disabled (light): modules-light is not Plane's original
PASS C15 modules disabled (light): no "plane" in the page
PASS C15 modules disabled picture (light): no request outside the scenario's list
PASS C15 modules disabled picture (light): no uncaught page error
PASS C15 modules disabled picture (light): no "call navigate() in a React.useEffect()" warning

== C15 views disabled picture (light)
PASS C15 views disabled (light): shows views-light
PASS C15 views disabled (light): views-light is not Plane's original
PASS C15 views disabled (light): no "plane" in the page
PASS C15 views disabled picture (light): no request outside the scenario's list
PASS C15 views disabled picture (light): no uncaught page error
PASS C15 views disabled picture (light): no "call navigate() in a React.useEffect()" warning

== C16 no projects picture (light)
PASS C16 no projects (light): shows no-projects-light
PASS C16 no projects (light): no-projects-light is not Plane's original
PASS C16 no projects picture (light): no request outside the scenario's list
PASS C16 no projects picture (light): no uncaught page error
PASS C16 no projects picture (light): no "call navigate() in a React.useEffect()" warning

== C15 modules disabled picture (dark)
PASS C15 modules disabled (dark): shows modules-dark
PASS C15 modules disabled (dark): modules-dark is not Plane's original
PASS C15 modules disabled (dark): no "plane" in the page
PASS C15 modules disabled picture (dark): no request outside the scenario's list
PASS C15 modules disabled picture (dark): no uncaught page error
PASS C15 modules disabled picture (dark): no "call navigate() in a React.useEffect()" warning

== C15 views disabled picture (dark)
PASS C15 views disabled (dark): shows views-dark
PASS C15 views disabled (dark): views-dark is not Plane's original
PASS C15 views disabled (dark): no "plane" in the page
PASS C15 views disabled picture (dark): no request outside the scenario's list
PASS C15 views disabled picture (dark): no uncaught page error
PASS C15 views disabled picture (dark): no "call navigate() in a React.useEffect()" warning

== C16 no projects picture (dark)
PASS C16 no projects (dark): shows no-projects-dark
PASS C16 no projects (dark): no-projects-dark is not Plane's original
PASS C16 no projects picture (dark): no request outside the scenario's list
PASS C16 no projects picture (dark): no uncaught page error
PASS C16 no projects picture (dark): no "call navigate() in a React.useEffect()" warning

== B12 intake creator (list)
PASS B12 the intake item is listed
PASS B12 the intake list shows the creator's own avatar (Z), not Plane's (P)
PASS B12 intake creator (list): no "plane" in the page
PASS B12 intake creator (list): no request outside the scenario's list
PASS B12 intake creator (list): no uncaught page error
PASS B12 intake creator (list): no "call navigate() in a React.useEffect()" warning

== B12 intake creator (work item)
PASS B12 the work item opens
PASS B12 the work item's "Created by" names the creator, not Plane
PASS B12 intake creator (work item): no "plane" in the page
PASS B12 intake creator (work item): no request outside the scenario's list
PASS B12 intake creator (work item): no uncaught page error
PASS B12 intake creator (work item): no "call navigate() in a React.useEffect()" warning

177 passed, 0 failed
```

### 6.2 P4 的核对脚本重跑

P5 改了登录页、帮助菜单、顶部栏、收集箱和编辑器，P4 的 9 段脚本（当前项、地址、跳转和 `next_path`、命令面板，P2、P3 探测的副本）在 Task 6 之后和修复轮之后重跑，结果应与 P4 结束时相同。脚本全文见 [P4 评审记录](P4-router-native-review.md)附录 6.2–6.4，它们都不断言 Plane 的文字。

```bash
bash $P4TMP/probe-c/run-c.sh p5final $P5TMP/probe-app   # 检出到 f697c05
```

```
build at f697c05
current: exit 0, PASS 106, FAIL 0, last: 106 passed, 0 failed
addresses: exit 0, PASS 69, FAIL 0, last: 69 passed, 0 failed
redirects: exit 0, PASS 61, FAIL 0, last: 61 passed, 0 failed
palette: exit 0, PASS 15, FAIL 0, last: 15 passed, 0 failed
p2probe1: exit 0, PASS 37, FAIL 0, last: 37 passed, 0 failed
p2probe2: exit 0, PASS 17, FAIL 0, last: 17 passed, 0 failed
p2probe3: exit 0, PASS 24, FAIL 0, last: all checks passed
p3auth: exit 0, PASS 46, FAIL 0, last: 46 passed, 0 failed
p3lists: exit 0, PASS 28, FAIL 0, last: all checks passed
```

P3 探测 1 的"每个收藏都有图标"在 Task 6 的构建上失败（第 4 节裁定 11）。原来的写法是 `item.locator("xpath=ancestor::*[.//svg or .//img or .//span[contains(@class,'emoji')]][1]")`：XPath 的 `svg` 在 HTML 文档里匹配不到 SVG 命名空间的元素，这项检查一直靠页面上方某个 `<img>` 通过，P4 时是 Task 5 删掉的 "Star us on GitHub" 图片。改写的脚本如下，它把检查换成从标题向上找、到第一个也包含别的收藏的祖先为止，用 CSS 选择器找图标并打印它的标签：

```py
# One-off (M1/P5 probe round 2): P4's p3lists.mjs check "each favourite has an icon" looked for the nearest ancestor
# holding `.//svg or .//img or …emoji` by XPath. In an HTML document XPath's `svg` never matches an SVG-namespace
# element, so the check could only pass on an <img> somewhere above the row — on P4's build the top bar's "Star us on
# GitHub" picture, which P5 Task 5 deleted. The check now walks up from the title with CSS (namespace-aware) and stops
# at the first ancestor that also holds another favourite, so the icon it finds is this row's own; it reports the
# icon's tag.
import pathlib

f = pathlib.Path(
    "/private/tmp/claude-501/-Users-xiaoruan-project-nerve-project/99d2bc1d-fdaf-4b92-a590-29b89514572b/scratchpad/"
    "nerve-p4/probe-c/p3lists.mjs"
)
s = f.read_text()
old = '''        const row = item.locator("xpath=ancestor::*[.//svg or .//img or .//span[contains(@class,'emoji')]][1]");
        if (!(await row.count())) missing.push(`${name}: no icon`);
'''
new = '''        // this row's icon: walk up from the title (CSS sees SVG elements, XPath's `svg` does not) and stop at the first
        // ancestor that also holds another favourite
        const icon = await item.evaluate((el, names) => {
          const own = el.textContent.trim();
          for (let node = el.parentElement; node; node = node.parentElement) {
            if (names.some((n) => n !== own && (node.textContent ?? "").includes(n))) return null;
            const found = node.querySelector("svg, img, [class*='emoji']");
            if (found) return found.tagName.toLowerCase();
          }
          return null;
        }, FAVOURITES.map((f) => f.name));
        console.log(`  ${name}: icon ${icon}`);
        if (!icon) missing.push(`${name}: no icon`);
'''
assert s.count(old) == 1
s = s.replace(old, new)
f.write_text(s)
print("fixed")
```

改写之后，这一项在基点和 `f697c05` 的构建上的输出：

```
# 96d8c1d
  Probe Project: icon svg
  Probe View: icon svg
  Sprint One: icon svg
  Module One: icon svg
  Probe Folder: icon svg
PASS favourites (P3): the sidebar lists a project, view, cycle, module and folder favourite, each with an icon
  favourites: 0 (from unstubbed requests: 0, other: 0)
# f697c05
  Probe Project: icon svg
  Probe View: icon svg
  Sprint One: icon svg
  Module One: icon svg
  Probe Folder: icon svg
PASS favourites (P3): the sidebar lists a project, view, cycle, module and folder favourite, each with an icon
  favourites: 0 (from unstubbed requests: 0, other: 0)
```

### 6.3 反向对照

为了证明 `brand.mjs` 能发现 P5 改变的东西，控制者在基点 `96d8c1d` 的构建上跑了全部三组。失败的正好是品牌检查：标题、清单、横版标志、页面上的 plane、帮助菜单和它打开的地址、加载动画、错误页和维护页、命令面板的命令、会话存储键、剪贴板类型、令牌的文案、收集箱创建者的两处特殊显示、C 组 8 张图是原图和插图的 `alt`；表单、页面能否打开、令牌和 Webhook 的列表这类保留行为在基点上也通过。有的场景在基点上中途失败（B8 找不到 "Open documentation" 命令，点击超时），之后的检查没有执行，所以基点的总数是 171 项。C 组在修复轮之前的 `7187d70` 构建上也跑过：8 张图各一项失败（页面显示的仍是原图）、`alt` 一项失败，其余通过。基点上的失败项：

```
FAIL A1 sign-in (light): the title is "… - Nerve": Sign in - Plane
FAIL A1 sign-in (light): one manifest, site.webmanifest.json: ["/site.webmanifest.json","/manifest.json"]
FAIL A1 sign-in (light): the visible lockup is the dark wordmark: 0 visible
FAIL A1 sign-in (light): "New to Nerve?" and no terms, privacy or "Join 10,000+": New to Plane?
FAIL A1 sign-in (light): no "plane" in the page: ["Sign in - Plane","New to Plane?","Welcome back to Plane.","Join 10,000+ teams building with Plane","meta[content]=Plane","meta[content]=Plane | Simple, extensible, open-source project management tool.","a[href]=https://plane.so/legals/terms-and-cond
FAIL A1 sign-in (dark): the title is "… - Nerve": Sign in - Plane
FAIL A1 sign-in (dark): one manifest, site.webmanifest.json: ["/site.webmanifest.json","/manifest.json"]
FAIL A1 sign-in (dark): the visible lockup is the light wordmark: 0 visible
FAIL A1 sign-in (dark): "New to Nerve?" and no terms, privacy or "Join 10,000+": New to Plane?
FAIL A1 sign-in (dark): no "plane" in the page: ["Sign in - Plane","New to Plane?","Welcome back to Plane.","Join 10,000+ teams building with Plane","meta[content]=Plane","meta[content]=Plane | Simple, extensible, open-source project management tool.","a[href]=https://plane.so/legals/terms-and-condi
FAIL A2 workspace: no "Star us on GitHub": P
FAIL A2 workspace: no "plane" in the page: ["Welcome to Plane,","re glad that you decided to try out Plane. You can now manage your projects with ease. Get started by creating a project.","Most things start with a project in Plane.","Make Plane yours.","meta[content]=Plane","meta[content]=Plane | Si
FAIL A3 help menu: Documentation, Keyboard shortcuts and the version, no Forum: ["Documentation","Keyboard shortcuts","Forum"]
FAIL A3 help menu: no "plane" in the page: ["Welcome to Plane,","re glad that you decided to try out Plane. You can now manage your projects with ease. Get started by creating a project.","Most things start with a project in Plane.","Make Plane yours.","meta[content]=Plane","meta[content]=Plane | Si
FAIL A3 help menu: Documentation opens the repository, without an opener: [["https://go.plane.so/p-docs","_blank"]]
FAIL A4 loader, prerendered: the pulsing mark, not a GIF:  none
FAIL A4 loader, prerendered: no "plane" in the page: ["Plane | Simple, extensible, open-source project management tool.","meta[content]=Plane","meta[content]=Plane | Simple, extensible, open-source project management tool.","meta[content]=Plane | Simple, extensible, open-source project management to
FAIL A4 loader, instance pending (light): the mark shows: 
FAIL A4 loader, instance pending (light): no "plane" in the page: ["meta[content]=Plane","meta[content]=Plane | Simple, extensible, open-source project management tool."]
FAIL A4 loader, instance pending (dark): the mark shows: 
FAIL A4 loader, instance pending (dark): no "plane" in the page: ["meta[content]=Plane","meta[content]=Plane | Simple, extensible, open-source project management tool."]
FAIL A4 the loader is the same image in both themes: {"light":null,"dark":null}
FAIL A5 error page: the message, the new advice and "Go to home": 🚧 Looks like something went wrong!
FAIL A5 error page: no support email, status page or X account: found one
FAIL A5 error page: no "plane" in the page: ["@planepowers","meta[content]=Plane","meta[content]=Plane | Simple, extensible, open-source project management tool.","a[href]=mailto:support@plane.so","a[href]=https://status.plane.so/","a[href]=https://x.com/planepowers"]
FAIL A6 maintenance: the Nerve heading, no support team or email: 🚧 Looks like Plane didn't start up correctly!
FAIL A6 maintenance: no "plane" in the page: ["🚧 Looks like Plane didn't start up correctly!","meta[content]=Plane","meta[content]=Plane | Simple, extensible, open-source project management tool.","a[href]=mailto:support@plane.so"]
FAIL A7 create workspace: the mark shows: 
FAIL A7 create workspace: no "plane" in the page: ["meta[content]=Plane","meta[content]=Plane | Simple, extensible, open-source project management tool."]
FAIL A7 onboarding: the lockup shows: 
FAIL A7 onboarding: no "plane" in the page: ["This is how you will appear in Plane.","meta[content]=Plane","meta[content]=Plane | Simple, extensible, open-source project management tool."]
FAIL A7 not found: no "plane" in the page: ["meta[content]=Plane","meta[content]=Plane | Simple, extensible, open-source project management tool."]
FAIL A7 workspace not found: no "plane" in the page: ["meta[content]=Plane","meta[content]=Plane | Simple, extensible, open-source project management tool.","img[alt]=Plane logo"]
FAIL A7 product tour: the lockup shows: 
FAIL A7 product tour: no "plane" in the page: ["Welcome to Plane,","re glad that you decided to try out Plane. You can now manage your projects with ease. Get started by creating a project.","Most things start with a project in Plane.","Make Plane yours.","meta[content]=Plane","meta[content]=Plane |
FAIL B8 "Open documentation" is offered: ["No results found\nClear search","Open Plane documentation"]
FAIL B8 palette help commands: ran to the end: locator.click: Timeout 30000ms exceeded.
FAIL B9 stale asset: remembered under __nerve_chunk_reload only: ["__plane_chunk_reload"]
FAIL B9 stale asset: no "plane" in the page: ["@planepowers","meta[content]=Plane","meta[content]=Plane | Simple, extensible, open-source project management tool.","a[href]=mailto:support@plane.so","a[href]=https://status.plane.so/","a[href]=https://x.com/planepowers"]
FAIL B10 copy writes text/nerve-editor-html and no plane type: ["text/plain","text/html","text/plane-editor-html"]
FAIL B10 a paste carrying only text/nerve-editor-html puts the description into the comment editor: 
FAIL B10 editor clipboard: no "plane" in the page: ["meta[content]=Plane","meta[content]=Plane | Simple, extensible, open-source project management tool.","a[href]=https://github.com/makeplane/plane"]
FAIL B11 the new-token note: "Copy and save this secret key", no "Plane Pages": P
FAIL B11 tokens (created): no "plane" in the page: ["Copy and save this secret key in Plane Pages. You can't see this key after you hit Close. A CSV file containing the key has been downloaded.","meta[content]=Plane","meta[content]=Plane | Simple, extensible, open-source project management tool."]
FAIL B11 the delete dialog says "access to Nerve data": P
FAIL B11 token delete: no "plane" in the page: ["Any application using this token will no longer have the access to Plane data. This action cannot be undone.","meta[content]=Plane","meta[content]=Plane | Simple, extensible, open-source project management tool."]
FAIL B11 webhooks: no "plane" in the page: ["meta[content]=Plane","meta[content]=Plane | Simple, extensible, open-source project management tool.","a[href]=https://github.com/makeplane/plane"]
FAIL C13 workspace not found: workspace-not-available is not Plane's original: ["43f745bfc0842b24"]
FAIL C13 the picture is decorative (alt=""): ["Plane logo"]
FAIL C14 creation disabled: workspace-creation-disabled is not Plane's original: ["b0f772ebcb4beb26"]
FAIL C15 modules disabled (light): modules-light is not Plane's original: ["5e8804f74aab1249"]
FAIL C15 modules disabled (light): no "plane" in the page: ["meta[content]=Plane","meta[content]=Plane | Simple, extensible, open-source project management tool.","a[href]=https://github.com/makeplane/plane"]
FAIL C15 views disabled (light): views-light is not Plane's original: ["ab700af52407b558"]
FAIL C15 views disabled (light): no "plane" in the page: ["meta[content]=Plane","meta[content]=Plane | Simple, extensible, open-source project management tool.","a[href]=https://github.com/makeplane/plane"]
FAIL C16 no projects (light): no-projects-light is not Plane's original: ["2f86d193b353d39f"]
FAIL C15 modules disabled (dark): modules-dark is not Plane's original: ["55be6db04eb55699"]
FAIL C15 modules disabled (dark): no "plane" in the page: ["meta[content]=Plane","meta[content]=Plane | Simple, extensible, open-source project management tool.","a[href]=https://github.com/makeplane/plane"]
FAIL C15 views disabled (dark): views-dark is not Plane's original: ["7431cddf974168eb"]
FAIL C15 views disabled (dark): no "plane" in the page: ["meta[content]=Plane","meta[content]=Plane | Simple, extensible, open-source project management tool.","a[href]=https://github.com/makeplane/plane"]
FAIL C16 no projects (dark): no-projects-dark is not Plane's original: ["f2fcb81d75e80781"]
FAIL B12 the intake list shows the creator's own avatar (Z), not Plane's (P): ["P"]
FAIL B12 intake creator (list): no "plane" in the page: ["meta[content]=Plane","meta[content]=Plane | Simple, extensible, open-source project management tool.","a[href]=https://github.com/makeplane/plane","span[aria-label]=Plane"]
FAIL B12 the work item's "Created by" names the creator, not Plane: "Created by Plane"
FAIL B12 intake creator (work item): no "plane" in the page: ["Plane","meta[content]=Plane","meta[content]=Plane | Simple, extensible, open-source project management tool.","a[href]=https://github.com/makeplane/plane"]
107 passed, 64 failed
```

## 7. 交接与延后项

交给后续 M 的事项写进对应 M 的 `handoffs/M1-P5-brand.md`：

| M | 事项 |
|---|---|
| M4 | - 收集箱列表和工作项详情不再为 Plane 的收集箱机器人（邮箱 `intake@plane.so`、名字带 `-intake`）显示特殊的头像和名字，创建者按普通用户显示；由系统代为创建的工作项怎样显示，随工作项和收集箱的接口一起定；<br>- 动态里自动归档的操作者显示为站点名（`SITE_NAME`，基点写死 "Plane"）；系统操作者的显示同上 |
| M8 | - **propel 的对应源码**：`@makeplane/propel` 0.3.0 以编译后的形式随 Nerve 分发（AGPL-3.0-only），上游仓库不公开；公开发布之前取得提交 `0a31b15` 的源码，或从包里的 source map 还原（1376 个源文件中缺 `src/internal/variant-props.ts`）；<br>- **版本号**：12 个包和 web 应用的 `version` 仍是 Plane 的 1.4.2，帮助菜单显示它，随首次发布一起定；`version-number.tsx` 导入整个 `package.json`，客户端包因此带着全部依赖和版本，改为只取 `version`（具名导入或构建时注入）；<br>- **分享图的地址**：`og:image`、`twitter:image` 是相对地址（基点如此），抓取方不会解析；有了公开地址之后写成绝对地址 |

交给 M1 收尾（基点就有、不属于 P5 范围的问题；spec 7.3 的两项也在这里）：

| 事项 | 说明 |
|---|---|
| **`window.open` 没有 `noopener`**（安全，优先） | `web/packages/editor/src/extensions/custom-link/helpers/clickHandler.ts:53` 用 `window.open(href, target)` 打开用户写的链接，绕过了锚点上的 `rel="noopener noreferrer nofollow"`；另有 `issues/attachment/attachment-list-item.tsx:57`、编辑器 `custom-image/components/toolbar/download.tsx:22`、`full-screen/modal.tsx:276`、`:286`。别的成员贴的链接、从另一个源提供的 HTML 附件因此拿得到 `window.opener`。都传 `"noopener,noreferrer"` |
| **133 张没有引用的图片** | 删除；其中有的写着 Plane 的名字（如 `all-issues/all-issues-light.webp` 的 "Plane Demo"），它们不随应用发布，但不该留在仓库里 |
| **照片和示例数据的来源** | 29 张封面照片（spec 7.3）；导览截图 `onboarding/cycles.webp`、`modules.webp`、`views.webp` 里的真人头像和一位 Plane 联合创始人的名字。查不到许可的替换或删除 |
| **`brand` 在 `pnpm-workspace.yaml` 的例外**（`until: M9`） | 按 M1 设计 11 节重新核对理由（spec 7.3） |
| **导入分组的错标** | `app/root.tsx` 悬空的 `// types`；基点就有的 4 对相邻同名分组：`extended-sidebar-wrapper.tsx`、`base-list-root.tsx`、`modules-list-view.tsx`、`use-notification.ts` |
| **守卫里从未存在过的样本路径** | `changelog` 等约 20 条规则的文件不命中样本 `web/apps/web/app/assets/logo.svg`，这个文件在本仓库的历史里从未存在 |
| **没有调用方的代码和文案** | `ui/empty-space.tsx` 没人传的属性（`EmptySpace` 的 `Icon`、`EmptySpaceItem` 的 `description`）和只有一个子元素的片段；482 个无引用的文案键（包括 `power_k.help_actions.chat_with_us`、`common.forum`）；错误页插图的 `alt="ProjectSettingImg"`（spec 第 5 节） |
| **Go 测试的假文件名** | `server/internal/platform/webui/handler_test.go` 的夹具（`manifest.json`、`favicon/android-192.png`）注释说"对应 `web/apps/web/build/client` 的布局"，名字却是编的；`handler.go` 不依赖任何具体文件名。改成构建真实产出的名字，或改写注释 |

## 8. spec 第 3 节的裁定

第 3 节的 15 项在执行前全部采纳（plan 的"控制者评审补充"），执行中的落实情况：

1. **导入分组注释单独一个 Task**：照此执行。Task 1 的提交就是 `rename.mjs` 的输出，Task 2 的 `verify.mjs` 证明只改了注释行；两处相邻的同名分组由跟进提交收掉（第 4 节裁定 3）。
2. **Nerve 的图形组件放在 web 应用**：照此执行，`nerve-logo.tsx` 按文件名读取 `app/assets/brand/` 的图片。
3. **加载动画不用 `dark:`**：照此执行，A4 在两种主题下断言同一张图，预渲染的加载动画没有 GIF。
4. **版权声明行和 `@makeplane/propel` 写在正则里排除**：照此执行；修复轮把第二个排除收紧到后面不接字母、数字、下划线或连字符，并加了单独证明 `github.com/makeplane` 命中的样本（第 3 节 M1）。
5. **`plane-package` 的范围比 `brand` 宽**：照此执行。
6. **规则 `brand` 在最后一个 Task 加入**：照此执行，每个 Task 用 `brandhits.mjs` 报告进度：3902 → 964 → 176 → 145 → 71 → 46 → 3。
7. **只留一份应用清单**：照此执行，A1 断言只有 `site.webmanifest.json`。
8. **没有引用、又写着 Plane 的文案在 Task 4 删除**：照此执行，16 个键；其余 482 个交收尾。
9. **Plane 收集箱机器人的特殊显示删除**：照此执行，B12 在两处特殊显示上各断言一次（第 4 节裁定 11），交 M4。
10. **"Report a bug" 指向 `/issues`**：照此执行。
11. **导览图片改字，不删图**：照此执行；修复轮用同样的办法改了另外 4 张图的 "Plane Design"。
12. **`tailwind-config/AGENTS.md` 改写，不开整词例外**：照此执行，15 处改为文档自己用过的 "stacking context"。
13. **Nerve 自己写的文件改为 Nerve 的版权声明**：照此执行，持有者为 OpenNerve（第 4 节裁定 2）；Task 7 评审复核了有疑问的文件，复述 Plane 代码的 6 个保留 Plane 的声明。
14. **propel 的源码提交能找到**：照此执行，Task 7 和它的评审者都在线核对了来源证明和 source map；对应源码交 M8。
15. **`dangling.mjs` 只数像分组标签的行**：照此执行，各 Task 和整分支的输出都是 0。
