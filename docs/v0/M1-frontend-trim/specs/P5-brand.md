# M1/P5 `brand`：品牌与包名 Spec

- 上级设计：[M1 设计](../M1-design.md)（3.10、5、7.4、7.5、9 节 P5、9.7）
- 前一 Phase：[P4 spec](P4-router-native.md)、[P4 plan](../plans/P4-router-native.md)、[P4 review](../reviews/P4-router-native-review.md)（第 4 节裁定、第 5 节计划缺陷）
- 实施计划：[P5 plan](../plans/P5-brand.md)
- 分支：`worktree-m1-p5-brand`，基线 `96d8c1d`（P4 合并后的 `main`）

---

## 1. 目标

界面上不再有 Plane 的名称和 Logo，工作区的包名全部是 `@nerve/*`，关键词守卫看住这两点：

- 工作区的 12 个包 `@plane/*` 改名为 `@nerve/*`，一个纯机械的提交，锁文件只随改名变化；导入的分组注释 `// plane …` 随之改名；
- Nerve 的矢量图标、横版标志、网站图标、应用图标和分享图替换 Plane 的；约 1 MB 的两张加载动画 GIF 换成图标加 CSS 动画；
- 文案、页面标题和元数据里的 Plane 改为 Nerve，讲 Plane 公司或 Plane 服务的句子按含义改写或随功能删除；
- 指向 Plane 服务的链接删除，文档和问题反馈指向 Nerve 的仓库 `https://github.com/open-nerve/NerveProject`；
- 代码标识符、会话存储键、编辑器的剪贴板类型、组件的 `displayName` 和注释不再带 plane；
- 关键词守卫加入 `plane-package`、`brand`、`brand-files` 三条规则；Nerve 自己写的文件标注 Nerve 的版权；`@makeplane/propel` 的来源记进前端改动清单。

同时处理交给 P5 的事项：P1 评审第 6 节 (c)（两张加载动画 GIF）、P3 评审第 7 节的 P5 一行（`// plane web …` 注释、页面标题的 " - Plane"、品牌文案）、M0 交接 `M0-P6-knip-notes` 剩下的"S2 与包名"一节（M1 设计 9.7）。

**存储键不迁移**：会话存储键 `__plane_chunk_reload` 直接改名为 `__nerve_chunk_reload`，编辑器的剪贴板类型同样直接改名。Nerve 还没有用户，没有需要保留的旧值；这两个值本来也只在一个标签页的会话、一次复制粘贴里有效。

---

## 2. 交付物

### 2.1 文件总览

7 个 Task，每个一个提交，顺序是 M1 设计 9 节 P5 的三步：包名（Task 1、2）→ 品牌资源、文案、外部链接、存储键和组件名（Task 3–6）→ 可见品牌、版权和来源的检查（Task 7）。原型（`96d8c1d..36680de`，在 `$P5TMP/proto`）实测总计
`1215 files changed, 4123 insertions(+), 4566 deletions(-)`：删 28 个、新增 12 个、修改 1175 个。

| Task | 标题 | 删 | 增 | 改 | 行（+/−） | 原型提交 |
|---|---|---:|---:|---:|---|---|
| 1 | 包名 `@plane/*` 改为 `@nerve/*`，锁文件等价核对，规则 `plane-package` | — | — | 1101 | +2999 / −2964 | `40a54db` |
| 2 | 导入的分组注释 `// plane …` 改名，悬空的删除 | — | — | 671 | +755 / −788 | `be5666a` |
| 3 | Nerve 的图标、标志、网站图标和加载动画 | 24 | 12 | 18 | +147 / −417 | `a559e05` |
| 4 | 文案、页面标题和元数据 | — | — | 35 | +52 / −131 | `61538ba` |
| 5 | Plane 服务的链接 | 4 | — | 14 | +27 / −207 | `6b440f0` |
| 6 | 标识符、存储键、剪贴板类型、组件名和注释 | — | — | 26 | +43 / −43 | `89ba562` |
| 7 | 守卫规则、Nerve 文件的版权、propel 的来源、文档 | — | — | 16 | +113 / −29 | `36680de` |

### 2.2 原型验证：结论与证据

在 `$P5TMP/proto`（从本分支的基线克隆）中逐个 Task 做过一遍，每个 Task 结束时 `pnpm exec turbo run check:types`、`make lint-web`、`make test-web`、`make build-web`、`make knip` 都通过。之后在一个从基线新建的工作树上按 plan 的脚本步骤重放了全部 7 个 Task（`$P5TMP/tools/replay.mjs`），每个 Task 得到的树与原型提交完全相同（重放发现 Task 3、4 要在脚本之后跑一次 oxfmt，plan 已写进步骤）。下面是基线与终态的实测值。

| 项 | 基线（`96d8c1d`） | P5 结束（`36680de`） |
|---|---|---|
| `make lint-web` | `keywords: 45 rules, 4 exceptions, no hits.` + 52 个 turbo 任务 | `keywords: 48 rules, 5 exceptions, no hits.` + 52 |
| `pnpm exec turbo run check:types` / `make build-web` | 23 / 11 | 23 / 11 |
| `make test-web` | 16 个任务；vitest 77 个测试（web 5、constants 11、editor 16、i18n 10、services 11、utils 24），stderr 只有 editor 的 5 行 `prosemirror-codemark` | 不变 |
| `make knip` | 零 | 零 |
| lint 上限合计 | 711 | 711，不变；`lintdiff.sh` 没有新增的警告 |
| `docs/` 以外的 `@plane/`（行 / 文件） | 2963 / 1101 | 6 / 1（`tools/keywords.json` 里 `plane-package` 规则的样本） |
| `pnpm -r ls --depth -1` | `nerve`、`@nerve/e2e`、`web`、`@nerve/api-client` 和 12 个 `@plane/*` | 同样 16 项，12 个改为 `@nerve/*`，版本都是 1.4.2，全部 `private` |
| 锁文件 | — | 对基线锁文件做同样的替换之后与新锁文件相同（`lockcheck.mjs`：0 行差异）；Task 1 之后不再变化 |
| 品牌规则的命中（`brandhits.mjs`，与 `brand` 规则同一正则） | 3902：包名 2945、导入分组注释 788、文案 en 21 / zh-CN 21、写死的文字 44、网址 23、标识符 20、`displayName` 13、Markdown 16、注释 7、剪贴板类型 3、存储键 1 | 3（`pnpm-workspace.yaml` 记录来源的 3 行注释，登记为例外） |
| `web/` 下名字带 plane 的路径（`brand-files`） | 5 | 0 |
| 规则排除的两种文本 | Plane 的版权声明行 2076 个文件；`@makeplane/propel` | 版权声明行 2051 个文件（12 个改为 Nerve 的声明，13 个随文件删除）；`@makeplane/propel` 454 个文件、607 处 |
| 带 Nerve 版权声明的文件 | 0 | 18（新写的 1 个组件、3 个 SVG、2 个 `SOURCES.md`，改正的 12 个） |
| 每种语言的文案键 / 无引用的键 | 1617 / 498 | 1599 / 482 |
| 没有引用的图片（`assets.mjs`） | 136 | 133（删掉 3 张没有引用的 Plane 图片，没有新的孤儿） |
| 构建体积 js | 397 个 / 6,849,609 字节 | 397 个 / 6,818,890 字节 |
| 构建体积 css / fonts | 3 / 296,988；25 / 3,755,608 | 3 / 296,816；不变 |
| 构建体积 other | 112 / 7,910,263 | 110 / 6,457,945（两张 GIF 共 1,404,364 字节） |
| 最大 chunk / 语言 chunk | `use-parse-editor-content` 1,379,322 / 34 | 1,379,320 / 34 |
| 页面上的 "plane"（`probe/visual.mjs`，10 个场景的文字节点和 `title`、`alt`、`href`、`content` 等属性） | 10 个场景都有 | 没有（52 项检查全过；基线上 16 项失败，是反向对照） |

每个 Task 结束时的中间值（`$P5TMP/tools/pertask.mjs` 在各原型提交上重算，不检出）：

| 原型提交 | 品牌命中 | 名字带 plane 的路径 | `@plane/`（行） | 守卫规则 / 例外 | 每种语言的键 |
|---|---:|---:|---:|---|---:|
| 基线 `96d8c1d` | 3902 | 5 | 2963 | 45 / 4 | 1617 |
| T1 `40a54db` | 964 | 5 | 6 | 46 / 4 | 1617 |
| T2 `be5666a` | 176 | 5 | 6 | 46 / 4 | 1617 |
| T3 `a559e05` | 145 | 0 | 6 | 46 / 4 | 1617 |
| T4 `61538ba` | 71 | 0 | 6 | 46 / 4 | 1601 |
| T5 `6b440f0` | 46 | 0 | 6 | 46 / 4 | 1599 |
| T6 `89ba562` | 3 | 0 | 6 | 46 / 4 | 1599 |
| T7 `36680de` | 3（例外） | 0 | 6 | 48 / 5 | 1599 |

结论：

1. **改名不会弄坏已发布的包。** 改名的 12 个包都是 `private`，只在工作区里互相引用；`@makeplane/propel` 是 npm 上的第三方包，`@plane/` 的替换碰不到它（它的 `plane` 前面没有 `@`）。锁文件与"基线锁文件做同样替换"逐字节相同，没有依赖升级。`knip.jsonc` 里的工作区路径是目录，不用改。
2. **导入分组注释要逐行判断，不能随 `sed` 一起改。** 788 行里 706 行是 `// plane imports` 这类讲包名的，改成 `// nerve …`；54 行是 Plane 的 CE/EE 目录留下的 `// plane web …`，那个目录已经不存在，按下一行的导入改名（`// components`、`// hooks`、`// nerve imports` 等），与上一组同名时合并（5 行）；28 行没有标注任何导入（悬空），删除。`verify.mjs` 核对这个提交只改了注释行。
3. **图形都是文件，代码只按文件名读取。** `app/assets/brand/` 里是 `mark.svg`、`lockup.svg`、`lockup-on-dark.svg` 三个手画的矢量图（版权声明写在 XML 注释里），和由它们渲染出的 PNG、ICO（`rasterize.mjs`，同一台机器上重跑得到逐字节相同的文件）。换 Logo 只要替换 SVG、再跑一次脚本。
4. **图片里的品牌逐张看过。** 基线 `web/` 下 251 张图片：222 张拼成联系表逐张看，另 29 张封面照片单独看；带 Plane 品牌的只有这些：网站图标、应用图标、分享图、加载动画（替换）；维护页插图屏幕上的 Plane Logo（删掉那一条路径）；导览图片里标题为 "Plane integration" 的卡片（改为 "API integration"，`retext.mjs`）；propel 里 Plane 客户的 Logo（只给登录页页脚用，随页脚删除）；3 张没有引用的 Plane 图片（删除）。
5. **加载动画不再依赖主题。** 新的加载动画是图标加 Tailwind 的 `animate-pulse`：图标是深青色底的方块，在深浅两种背景上都看得清，所以只有一张图，预渲染的标记与主题无关，P1 的 #418 修复仍然成立（预渲染和首次渲染一致）。横版标志的字标颜色随主题变，用 CSS 的 `dark:` 在两张图之间切换（`dark` 变体按 `data-theme` 匹配，不读取主题）。
6. **维护页之外还有两个错误页。** 页面报错时的 `app/error/prod.tsx`（根 `ErrorBoundary`）和 404 页。前者带 Plane 的支持邮箱、状态页和 X 账号，Task 5 删掉；后者没有品牌。路由模块加载失败时 React Router 直接刷新页面（生产构建的 `routeModules.js`，每次都刷新），不进 `ErrorBoundary`；组件里 `lazy()` 的分块加载失败才进，并先经陈旧资源的恢复刷新一次。所以浏览器核对用渲染错误和 `lazy()` 分块失败两种方式触发错误页（plan 最后一节）。
7. **页面上已经没有 plane。** `probe/visual.mjs` 在终态的构建上打开登录页（浅色、深色）、两种加载动画、工作区首页和帮助菜单、维护页、错误页、陈旧资源的恢复、创建工作区、新手引导和 404 页，文字和属性里都没有 plane，52 项检查全过；在基线的构建上 16 项失败。陈旧资源的场景同时证明改名后的会话存储键读写一致：页面刷新一次、第二次失败时显示错误页，会话存储里只有 `__nerve_chunk_reload`。
8. **没有新的共享可变状态。** 全 Phase 新增的顶层声明只有 3 个字符串常量（`SITE_NAME`、`REPOSITORY_URL`、`STALE_ASSET_RELOAD_KEY`）和 `nerve-logo.tsx` 的两个组件；新增行里没有 `={"`、`${"`，也没有不带插值的模板字面量。

### 2.3 包名（Task 1，M1 设计 5）

- `rename.mjs` 在 `docs/` 和 `pnpm-lock.yaml` 以外，对 git 跟踪的每个含 `@plane/` 的文本文件做 `@plane/` → `@nerve/` 的替换：1100 个文件（`web/` 1099 个和 `tools/keywords.json` 里旧规则的样本）、2922 处；随后 `pnpm install` 改写锁文件（41 行）。
- 核对：`lockcheck.mjs 96d8c1d`（即设计写的 `git show 96d8c1d:pnpm-lock.yaml | sed 's#@plane/#@nerve/#g' | diff - pnpm-lock.yaml`，一条脚本、不用管道）为 0 行；`pnpm -r ls --depth -1` 只列出 `nerve`、`web`、`@nerve/*`；`docs/` 以外的 `@plane/` 只剩新规则的 6 行样本。
- 守卫规则 `plane-package`；顶层 `phase` 改为 `M1/P5`（没有 `until: M1/P5` 的例外）。

### 2.4 导入的分组注释（Task 2，M1 设计 5、P3 评审第 7 节）

见 2.2 结论 2。`labels.mjs` 只改注释行，不移动任何导入；`dangling.mjs` 为 0。

### 2.5 品牌图形（Task 3，M1 设计 5、P1 评审第 6 节 (c)）

| 文件 | 内容 | 大小（字节） |
|---|---|---:|
| `app/assets/brand/mark.svg` | 图标：深青色（`#155E75`）圆角方块上，一条白色的线从左下的突触折到右上的突触，写成一个 N | 580 |
| `app/assets/brand/lockup.svg`、`lockup-on-dark.svg` | 横版标志：半尺寸的图标加小写字标 "nerve"（描边路径），字标分别是深色 `#1F2937` 和浅色 `#F9FAFB` | 1267、1194 |
| `app/assets/brand/favicon-16x16.png`、`favicon-32x32.png`、`favicon.ico`（16/32/48） | 网站图标 | 489、927、2897 |
| `app/assets/brand/icon-180x180.png`、`icon-512x512.png` | 苹果触屏图标 | 5095、14873 |
| `app/assets/brand/og-image.png` | 1200×630 的分享图：横版标志和一行 "Open-source project management"（Inter） | 27992 |
| `public/icons/icon-192x192.png`、`icon-512x512.png` | `site.webmanifest.json` 的图标（清单要固定地址，所以放在 `public/`） | 5055、14873 |
| `app/assets/brand/SOURCES.md` | 放不下文件头的 PNG、ICO 的来源和版权，`public/icons/` 的两个图标也登记在这里（`public/` 的文件原样发布，Task 3 放在那里的 `SOURCES.md` 会出现在 `/icons/SOURCES.md`，修复轮并入这里、删除） | — |

- **代码**：新增 `core/components/common/nerve-logo.tsx`（`NerveLogo`：图标；`NerveLockup`：横版标志，浅色和深色两张图用 `dark:` 切换，`onColor` 用于彩色背景）。7 处调用方从 propel 的 `PlaneLogo`、`PlaneLockup`、`PlaneNewIcon` 改为它们；propel 的 `icons/brand/`（Plane 的 3 个图形组件和 4 个客户 Logo）、`icons/sub-brand/` 和注册表里的 `sub-brand.plane` 删除。登录页页脚 "Join 10,000+ teams building with Plane"（客户 Logo 的唯一使用者）删除。
- **加载动画**：`logo-spinner.tsx` 渲染 `NerveLogo` 加 `animate-pulse`；两张 GIF 删除（2.2 结论 5）。
- **清单和网站图标**：`app/root.tsx` 从 `app/assets/brand/` 读取；从未被读取的第二份清单 `public/manifest.json`（页面只用第一个 `<link rel="manifest">`，即 `site.webmanifest.json`）连同它独有的 348 px 图标删除；`site.webmanifest.json` 的名字和描述改为 Nerve。
- **图片**：维护页的两张插图删掉屏幕上的 Plane Logo（每张一条 `<path>`）；`onboarding/issues.webp` 的 "Plane integration" 改为 "API integration"（202,198 → 228,386 字节）；没有引用的 `favicon/apple-touch-icon.png`、`plane-takeoff.png`、`users/user-profile-cover-default-img.png` 删除。

### 2.6 文案与元数据（Task 4，M1 设计 5）

- **en、zh-CN 在用的文案**：`auth.common.new_to_plane` 改名为 `new_to_nerve`（"New to Nerve?" / "首次使用 Nerve？"）；首页引导的两条、自动归档的说明、令牌删除的说明改为 Nerve；复制令牌的提示去掉"存进 Plane Pages"（那是不在 Nerve 里的产品）。
- **没有引用、又写着 Plane 的文案**（7 组，每种语言 16 个键）删除：`self_hosted_maintenance_message`、`common_empty_state.not_found`、`project_settings.features.intake.email`、三处 `primary_button.comic`、`workspace_settings.empty_state.api_tokens`。
- **标题和元数据**：`SITE_NAME` 改为 "Nerve"，是页面标题、`og:title` 和 `application-name`；只装着 Plane 的名称、网址、账号的 `SITE_TITLE`、`SITE_URL`、`TWITTER_USER_NAME`（没有读取方）、`og:url`、`twitter:site` 删除；注册页和登录页的标题是 "… - Nerve"。
- **写死的文案**：登录和注册的副标题、新手引导的 4 步、导览、邀请页的 3 处说明、收集箱的页面标题、归档动态的操作者都改为 Nerve；工作区不存在时图片的 `alt` 改为 "Workspace not found"。
- **退化结构**：`PageHead` 里 `if (title)` 内部永远不用的默认标题、动态的 `customUserName || "Plane"`（只在 `customUserName` 为真的分支里渲染）收掉。
- **Plane 收集箱机器人**：收集箱列表和工作项详情对 `intake@plane.so`、名字带 `-intake` 的用户显示 Plane 的头像和名字，这是 Plane 后端的约定；删除，创建者按普通用户显示（第 7 节交 M4）。

### 2.7 外部链接（Task 5，M1 设计 5）

| 位置 | 基线 | 改为 |
|---|---|---|
| 帮助菜单 "Documentation"、命令面板 "Open documentation"（`open_plane_documentation` → `open_documentation`） | Plane 的文档站 | `REPOSITORY_URL`（新增在 `@nerve/constants` 的 `metadata.ts`） |
| 命令面板 "Report a bug" | `github.com/makeplane/plane/issues/new/choose` | `${REPOSITORY_URL}/issues` |
| 帮助菜单 "Forum"、命令面板 "Join the forum" | forum.plane.so | 删除（连同 `join_forum` 键） |
| 错误页 | 支持邮箱、状态页、X 账号；"we track these errors automatically…" | 删除链接；说明改为 "Try refreshing the page. If the problem persists, contact your administrator." |
| 维护页 | 支持邮箱；"reach out to our support team" | 删除；标题改为 "Looks like Nerve didn't start up correctly!"；只剩一个元素的片段收掉 |
| 登录表单下的服务条款、隐私政策 | plane.so/legals | 删除，连同 `terms-and-conditions.tsx` |
| 顶部栏 "Star us on GitHub" | github.com/makeplane | 删除，连同 `star-us-link.tsx`、两张 GitHub 图片、`home.star_us_on_github` 键 |
| 邀请页的 "Star us on GitHub"、"Join our community" 卡片 | github.com/makeplane、forum.plane.so | 删除 |
| 空的项目设置页 "Learn more about projects" | plane.so | 删除；只剩一个按钮的容器收掉 |

README"M0 中看到的页面"一条引用的是维护页的原文，随之改为新标题。

### 2.8 标识符、存储键、剪贴板类型、组件名与注释（Task 6，M1 设计 5）

- `PlaneVersionNumber` → `VersionNumber`。
- 陈旧资源重新加载的会话存储键 `__plane_chunk_reload` → `__nerve_chunk_reload`（读和写都是同一个常量）。
- 编辑器的剪贴板类型 `text/plane-editor-html` → `text/nerve-editor-html`：两个写入方（`editor-ref.ts`、`markdown-clipboard.ts`）和读取方（`props.ts`）一起改。
- 13 个 `displayName` 从 `plane-ui-*` 改为 `nerve-ui-*`（propel 7 个、ui 6 个；没有代码读取它们）。
- 注释：路由测试里讲 Plane 历史的两处改为讲这份代码；Plane 的两条 TODO 写明它们指的 Nerve 包（`@nerve/hooks` 的 `useLocalStorage`、`@nerve/constants`）；邮箱示例改为 `user@example.com`；`GlobalModals` 的说明去掉 "across Plane applications"。
- editor 包的描述和 Readme 不再写 Plane；`tailwind-config/AGENTS.md` 用 "plane" 表示层次（15 行），改为同一文档已经在用的 "stacking context"。

### 2.9 关键词守卫、版权与来源（Task 7，M1 设计 5、3.10、7.4）

P4 的 45 条规则之上新增 3 条，全部 `"phase": "M1/P5"`；每个顶层分支都有真实代码里的命中样本（`alts.mjs M1/P5` 输出 0）。

| 规则 | 要点 | 文件范围 | Task |
|---|---|---|---|
| `plane-package` | `@plane/` | `docs/` 和 `tools/keywords.json` 以外的全部文件 | 1 |
| `brand` | `plane`，不区分大小写；两个否定前瞻精确排除 Plane 的版权声明行（`Copyright (c) 2023-present Plane Software, Inc.`）和包名 `@makeplane/propel` | `web/`、`pnpm-lock.yaml`、`pnpm-workspace.yaml`、`turbo.json` | 7 |
| `brand-files`（文件名） | `web/` 下路径里带 plane（不区分大小写） | — | 7 |

- **例外**（1 条）：`brand`、`pnpm-workspace.yaml`、`Plane`、`count: 3`、`until: M9`：文件开头两行和 `allowBuilds` 里一行中文注释记录这份配置来自 Plane 的哪个提交、哪些不来自 Plane（M0/P5 spec 2.4、M0/P6），在 v0 内不会消失。
- **规则先在基线上核对**：`rulehits.mjs` 在基线副本上报 `brand: 3902 hits in 1165 files`、`brand-files: 5 paths`，在 Task 6 之后的树上只有 `pnpm-workspace.yaml` 的 3 处。
- **Nerve 文件的版权**：迁入之后新增的 `web/` 文件里（`newfiles.mjs`，30 个，其中 P5 自己的 12 个），P1–P4 写的 18 个照抄了 Plane 的声明。`origin.mjs` 逐行在迁入的代码里查找：12 个是 Nerve 自己写的（测试、测试配置、`use-profile-member.ts`，至多 2 行短代码与迁入的代码相同），改为 `Copyright (c) 2026-present OpenNerve` 和 `SPDX-License-Identifier: AGPL-3.0-only`（持有者与 README 的"Copyright © 2026 OpenNerve"一致，控制者评审）；6 个复述 Plane 的代码，保留 Plane 的声明：`sidebar-chart.tsx`、`links/types.ts`、`project-navigation-dialog.tsx`、`utils/src/theme.ts`（搬过来的代码）、`home-body.tsx`（Plane 的 `home-dashboard-widgets.tsx` 删减而来）、`profile-index.tsx`（照 Plane 的重定向模块写成）。
- **`@makeplane/propel` 的来源**（M1 设计 3.10）：0.3.0，AGPL-3.0-only；tarball 完整性 `sha512-nGhiE42vLQVvv7NZOcQYKARJiTXyKDSSTcHOzPdtFpa+WkSz9B91jw6i+3Ikz5cpkv/aEKjOTJF+cByw2zv5ZQ==`；npm 的 SLSA 来源证明记录它由 `github.com/makeplane/propel` 的 `packages/propel`、提交 `0a31b1529c0f249a058e59ade313d8b34ea8f964` 经 `.github/workflows/release.yml` 构建；这个仓库不公开（访问为 404）；包里的 source map 带着 1376 个源文件中 1375 个的全文。记在前端改动清单第一节。
- **文档**：前端改动清单新增 1.5 节、第四节全部 `已完成 / M1/P5`；README 的"版权"一节说明 Nerve 文件的声明和 `SOURCES.md`；`M0-P6-knip-notes` 写"处理结果（M1/P5）"，改为 `closed`（第 7.4 节）。

### 2.10 保留行为的核对（M1 设计 7.5 的 P5 一行）

| 手段 | 核对 | 落点 |
|---|---|---|
| 构建与静态检查 | `check:types`、`make knip`、`make build-web`、守卫的 3 条规则；每个 Task 的 `brandhits.mjs` 数字；Task 1 的锁文件核对 | 每个 Task |
| 进仓库的测试 | 77 个测试不变；路由匹配测试照常扫到全部路径 | 每个 Task |
| 临时核对脚本（控制者写和跑，plan 最后一节） | 目视核对（登录页、工作区页、加载动画、错误页、帮助菜单的截图）；页面文字里没有 plane；帮助链接和命令；陈旧资源的存储键；编辑器复制粘贴；个人访问令牌、Webhook 的设置页仍然可达；重跑 P4 的 A、B、C 组；在基线的构建上做反向对照 | P5 review 附录 |
| S1–S4 | S2 仍然不断言页面文字（M1 设计 9.7） | 合并前 |

---

## 3. 与上级设计的差异和补充

每一条都已按"能自己定的就自己定"的原则决定，这里列出决定和理由，供控制者复核。

1. **导入分组注释单独一个 Task（Task 2）。** 设计说它们"随包名一起改为 `// nerve imports`"。决定：Task 1 只做 `sed` 式的替换，注释在紧接着的 Task 2 改。理由：`// plane web …` 的 54 行不能机械改名（2.2 结论 2），放进 Task 1 会让"纯机械的提交"不再纯；分开之后 Task 1 用锁文件和 `@plane/` 计数核对，Task 2 用 `verify.mjs` 证明只改了注释行。
2. **Nerve 的图形组件放在 web 应用，不放在 propel（Task 3）。** 设计说"propel 中的 `PlaneLogo`、`PlaneLockup`、`PlaneWordmark` 等改为 Nerve 的对应组件"。决定：删掉 propel 的 `brand/`、`sub-brand/`，新组件是 `web/apps/web/core/components/common/nerve-logo.tsx`。理由：propel 的图标是写在代码里的 SVG 组件，放在那里换 Logo 就要改代码，违反设计"之后可以直接替换文件，不需要改代码"；按文件名读取的图片由 Vite 打包，属于应用的资源；只有 web 应用用它们（7 处），`PlaneWordmark` 没有使用者。
3. **加载动画不用 `dark:`（Task 3）。** P1 的修复用 `dark:` 在浅色、深色两张 GIF 之间选择。新的加载动画在两种背景上是同一张图，写成 `dark:` 的一对就是两张相同的图片（退化结构）。P1 修复的实质——标记与主题无关、预渲染不读取主题——保留（2.2 结论 5）。横版标志的两张图不同，仍用 `dark:`。
4. **版权声明行和 `@makeplane/propel` 写在正则里排除，不登记例外（Task 7）。** 设计 7.4 说对它们开"精确例外"。例外按（规则、路径、原文）登记，这两种文本分布在 2051 个和 454 个文件里，要登记约 2500 条。两个否定前瞻只排除这两段确切的文本（版权声明行的整个前缀、作为整词的 `@makeplane/propel`），精确程度相同；规则的不命中样本证明两者被排除，命中样本证明其他写法（`github.com/makeplane`、`@plane/propel`）照常命中。
5. **`plane-package` 的范围比 `brand` 宽（Task 1）。** 它检查 `docs/` 以外的全部文件（`e2e/`、`tools/`、README 等也不许出现旧包名），只排除 `docs/`（历史记录）和规则文件本身（样本必须写出旧包名）。`brand` 的范围与设计 7.4 的守卫范围一致；`e2e/`、`tools/`、README 里讲 Plane 来源的文字是合法的。
6. **规则 `brand` 在最后一个 Task 加入。** P1 定下"删除的同一个提交加规则"。品牌的命中分散在 Task 1–6，在 Task 1 加规则要为约 960 处登记本 Phase 内到期的例外。决定：每个 Task 用同一正则的 `brandhits.mjs` 报告进度（2.2 的表），Task 7 在命中只剩 3 处时加入规则。`plane-package` 在 Task 1 随改名一起加入。
7. **只留一份应用清单（Task 3）。** 基线有两个 `<link rel="manifest">`，页面只用第一个（`site.webmanifest.json`）；第二个 `manifest.json` 从未生效，删除，连同只有它引用的 348 px 图标。
8. **没有引用、又写着 Plane 的文案在 Task 4 删除。** 死文案原本交收尾（设计 6、9 节）。这 16 个键会让 `brand` 规则命中，改写它们没有意义；删除后 `keyref.mjs orphaned` 为 0。其余 482 个无引用的键仍交收尾。
9. **Plane 收集箱机器人的特殊显示删除（Task 4）。** 见 2.6。Nerve 不区分人和智能体，也没有 Plane 的收集箱机器人；由系统代为创建的工作项怎样显示，由 M4 定（第 7 节）。
10. **"Report a bug" 指向 `/issues`，不是 `/issues/new/choose`（Task 5）。** 仓库的问题模板还没有定，`/issues` 总是有效。
11. **导览图片改字，不删图（Task 3）。** 设计 5 要求"图片中的 Logo 按使用位置逐个替换"。这张图只有一张卡片的标题写着 Plane，改字是最小的改动；新标题 "API integration" 与卡片的内容相符。
12. **`tailwind-config/AGENTS.md` 改写，不开整词例外（Task 6）。** 设计 7.4 说 airplane、planet 这类普通单词开确切的例外。这里的 plane 是"层次"的意思，15 行；登记永久的例外只是噪声，改用文档自己的术语 "stacking context" 更清楚。
13. **Nerve 自己写的文件改为 Nerve 的版权声明（Task 7）。** 设计只说"新文件"按实际来源标注。P1–P4 新写的 12 个文件照抄了 Plane 的声明，这是标错了来源，一并改正；复述 Plane 代码的 6 个文件保留 Plane 的声明（2.9）。判断有疑问时保留 Plane 的声明：多写一个来源不会错，少写会。
14. **propel 的源码提交能找到（Task 7）。** 设计 3.10 说"npm 元数据里没有对应的提交号"。npm 上这个版本的 SLSA 来源证明记录了仓库、目录、提交和工作流；仓库不公开，但包里的 source map 带着几乎全部源文件的全文。所以"以后并入源码"的前提是拿到这个提交或从 source map 还原，记在前端改动清单里。
15. **`dangling.mjs` 只数像分组标签的行。** P4 的版本在"后面不是导入"的情况下把整个注释块都算作悬空，Task 6 改写路由测试开头的说明时它报了一条假阳性。P5 的副本在这种情况下也只数形如 `// hooks` 的标签行（不分大小写）；各 Task 的输出因此都是 0。
16. **README 随页面文字修改。** "M0 中看到的页面"一条引用维护页的原文，Task 5 改了原文，同一个提交改 README；"版权"一节在 Task 7 补上 Nerve 文件的声明方式。

---

## 4. 验收标准

- [ ] 7 个 Task 各一个提交，每个提交结束时 `pnpm exec turbo run check:types` 23 个任务、`make lint-web` 52 个、`make test-web` 16 个、`make build-web` 11 个通过，`make knip` 为零。
- [ ] `make lint-web`：`keywords: 48 rules, 5 exceptions, no hits.`；每个包的 oxlint 警告数等于上限（合计 711，不变）；`node $P5TMP/alts.mjs M1/P5` 输出 `alternatives or variants without a hit sample: 0`。
- [ ] vitest 77 个测试；除 editor 的 5 行 `prosemirror-codemark` 之外没有 stderr。
- [ ] `git grep -c "@plane/" -- . ':!docs'` 只有 `tools/keywords.json:6`；`node $P5TMP/t1/lockcheck.mjs 96d8c1d` 为 0 行；Task 1 之后锁文件不再变化；`pnpm -r ls --depth -1` 只有 `nerve`、`web`、`@nerve/*`。
- [ ] `node $P5TMP/tools/brandhits.mjs --summary` 最后一行 `brand hits: 3, file names: 0`（3 处都是登记的例外）。
- [ ] `symref.mjs orphaned 96d8c1d`、`keyref.mjs orphaned 96d8c1d` 为 0；`headers.sh 96d8c1d`、`dangling.mjs 96d8c1d` 没有输出或为 0；`infile-orphans.mjs 96d8c1d` 列出的每一行都以 `defined now: 0)` 结尾。
- [ ] 新增的文件和改正的 12 个文件带 Nerve 的版权声明；PNG、ICO 登记在同目录的 `SOURCES.md`。
- [ ] 控制者的浏览器核对（plan 最后一节）全部通过，截图经控制者目视，脚本写进 review 附录；S1–S4 通过；持续集成的 `server`、`web`、`e2e` 通过。
- [ ] 前端改动清单（第一节的 propel 来源、1.5 节、第四节）、README、`M0-P6-knip-notes` 同步；M1 设计 12 节 P5 一行更新。

---

## 5. 不在 P5 范围内

- 版权声明行本身（总体设计 2.3：来自 Plane 的文件保留原有的声明）。
- `@makeplane/propel` 的替换或并入源码（设计 3.10：保留）；它的图标名、样式里的 Plane 痕迹看不到，不改。
- 包的版本号：12 个包和 web 应用的版本仍是 Plane 的 1.4.2，帮助菜单显示 "Version: v1.4.2"（第 7 节：版本号随发布定，交 M8）。
- `web/` 以外 Nerve 自己的代码（`server/`、`e2e/`、`tools/`、`api-client`）没有文件头，以仓库根目录的 `LICENSE` 为准（第 7 节）。
- 基线就有、与品牌无关的问题：482 个无引用的文案键、错误页插图的 `alt="ProjectSettingImg"`、从不渲染的应用栏（P4 裁定 9，本 Phase 只把其中的 `PlaneNewIcon` 换成 `NerveLogo`）。
- 29 张封面照片（`app/assets/cover-images/`）没有品牌，来源和许可不在本 Phase 核对（第 7 节）。

---

## 6. 风险

| 风险 | 应对 |
|---|---|
| 包名替换误伤第三方包或改到生成的文件 | 只替换 `@plane/`（碰不到 `@makeplane/`）；锁文件由 `pnpm install` 生成，再与"基线做同样替换"逐字节比较；`pnpm -r ls` 核对 |
| 注释改名时移动或删错了导入 | `labels.mjs` 只改注释行；`verify.mjs` 证明差异里只有注释行（`other 0`） |
| 图形替换后某处引用了不存在的文件，或留下没有引用的图片 | 构建失败会暴露缺失；`assets.mjs` 前后对比（136 → 133）；`infile-orphans` 列出随文件删除的符号 |
| 加载动画重新依赖主题，#418 回归 | 加载动画只有一张图、标记与主题无关；浏览器核对预渲染的加载动画（禁用脚本时）和客户端的加载动画 |
| 图片里的 Plane 品牌漏看 | 全部 251 张图片目视（2.2 结论 4）；控制者对实际页面截图目视 |
| 会话存储键、剪贴板类型只改了一侧 | 读写方在同一个 Task 一起改；浏览器核对陈旧资源的恢复和编辑器的复制粘贴 |
| 删链接时删掉了保留功能的入口 | 只删指向 Plane 服务的项；帮助菜单、命令面板的保留项和 PAT、Webhook 设置页由浏览器核对 |
| 规则为了通过而放宽 | 排除写成两段确切文本的否定前瞻，有样本证明；唯一的例外带 `count: 3`，多一处就失败；基线上 3902 处命中证明规则有效 |
| 版权来源标错 | `origin.mjs` 逐行对照迁入的代码；有疑问的保留 Plane 的声明 |

---

## 7. 移交事项

### 7.1 交给后续 M（写进对应 M 的 `handoffs/`）

| M | 事项 |
|---|---|
| M4 | - 收集箱列表和工作项详情不再为 Plane 的收集箱机器人（`intake@plane.so`、名字带 `-intake`）显示特殊的头像和名字，创建者按普通用户显示。由系统代为创建的工作项怎样显示，随工作项和收集箱的接口一起定。<br>- 工作项动态里自动归档的操作者显示为写死的 "Nerve"（基线是 "Plane"）；系统操作者的显示同上。 |
| M8 | - **propel 的对应源码**：`@makeplane/propel` 以编译后的形式随 Nerve 分发（AGPL-3.0-only），它的上游仓库不公开。公开发布之前，取得提交 `0a31b15` 的源码，或从包里的 source map 还原（1376 个源文件中缺 `src/internal/variant-props.ts`），以便提供对应源码（2.9）。<br>- **版本号**：12 个包和 web 应用的 `version` 仍是 Plane 的 1.4.2，帮助菜单显示它；Nerve 的版本号随首次发布定，届时一起改。 |

### 7.2 控制者的裁定（执行前）

- 版本号：P5 不改；Nerve 的版本号随首次发布定，交 M8（7.1）。
- 版权持有者：新文件头写 `Copyright (c) 2026-present OpenNerve`，与 README 的"Copyright © 2026 OpenNerve"一致（原稿写的 "Nerve contributors" 是控制者给的默认值，与 README 不符，执行前改正）。
- `web/` 以外的 Nerve 代码（Go 服务、`e2e/`、`tools/`）和 `web/packages/api-client` 不加文件头，以仓库根目录的 `LICENSE` 为准；只有 `web/` 里与 Plane 文件混在一起的地方按来源标注。

### 7.3 交给 M1 收尾

- `brand` 在 `pnpm-workspace.yaml` 的例外（`until: M9`）：按设计 11 节重新核对理由。
- 29 张封面照片的来源和许可（看起来是图库照片，Plane 随代码分发）：收尾查明来源；查不到许可的替换或删除。
- 其余基线就有的事项见第 5 节。

### 7.4 M0 交接的落点

- `M0-P6-knip-notes`：S2 与包名一节在 P5 完成（包名已改，`knip.jsonc` 的路径是目录，未改；S2 仍不断言页面文字），改为 `closed`。
