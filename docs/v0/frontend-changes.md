# 前端改动清单

本清单以 Plane v1.4.2（提交 `02c19e1`，`preview` 分支；`v1.4.2` 标签指向 `5f7d927`，两者的迁移文件逐字节相同，见 [`tools/plane-schema/README.md`](../../tools/plane-schema/README.md)）的前端为基线，记录 Nerve 前端（`web/`）的每一类改动。前端从 `02c19e1` 迁入，而不是 `v1.4.2` 标签：锁定的 React Router 8.3.0、pnpm 11.10.0 只在 `02c19e1` 中存在，标签中是 React Router 7.17/7.18、pnpm 11.3.0。有两个用途：
- 方便后续维护时查阅。
- 以后对照 Plane 的新版本挑选改进时，知道哪些地方已经和上游不一样了。

状态取值：`计划中` / `进行中` / `已完成`。执行时在"完成于"一栏填上对应的 M 或 Phase。

---

## 一、代码来源

| 项 | 内容 |
|---|---|
| 来源提交 | Plane `02c19e1341d93141e8ad7b3278298adce208bafc`（`preview` 分支，`package.json` 中的版本是 1.4.2） |
| 迁入方式 | M0/P5 用 `git archive` 按上面的完整提交复制。迁入时 13 个目录与 Plane 中对应目录的 git 树对象完全相同，之后的每一处改动都登记在本清单中（见 [P5 spec](M0-foundation/specs/P5-web-import.md) 2.3） |
| 使用 | `apps/web` → `web/apps/web`；packages 中的 types、constants、ui、propel、editor、i18n、hooks、utils、shared-state、tailwind-config、typescript-config → `web/packages/<包名>`；`patches/react-color@2.19.3.patch` → 仓库根目录的 `patches/` |
| 暂时使用 | `packages/services`：web 的令牌设置页和文件工具函数依赖它。M1/P1 删掉了其中没有调用方的 49 个文件，只剩令牌服务、地址规范化（带单元测试）和上传文件的元数据工具；M2（PAT）和 M5（文件）重写对应的接口调用后，将它删除 |
| 不使用 | `apps/admin`、`apps/space`、`apps/live`、`apps/api`、`apps/proxy`、`packages/logger`、`packages/decorators`、`packages/codemods`（已核实 web 及其依赖的包都不引用它们）；根目录的 `.npmrc`（pnpm 11 只从 `.npmrc` 读取认证和仓库地址，其中的其他设置都不起作用）；husky、lint-staged（Git 钩子）和 react-doctor |

### 1.1 迁入时的改动（M0/P5）

只做让前端能安装、检查、构建和开发的最小改动，没有功能改动。

| 位置 | 改动 | 原因 |
|---|---|---|
| `pnpm-workspace.yaml`（仓库根目录） | 工作区的路径改为 `web/apps/*`、`web/packages/*`、`e2e`；catalog、overrides、allowBuilds 只保留作用于迁入的包的条目（删掉 39 个 catalog 条目、14 个 overrides、3 个 allowBuilds） | 工作区的路径；删掉的条目只属于没有迁入的应用和包，或者不作用于迁入的包的依赖 |
| `pnpm-lock.yaml`（仓库根目录） | 以 Plane 的锁文件为起点，把 importers 的路径移到 `web/` 下，再由 pnpm 加入 Nerve 自己的依赖 | 迁入的 13 个包的依赖版本与 Plane 完全相同 |
| `package.json`（仓库根目录） | 只加入开发依赖 `oxfmt`、`oxlint`、`turbo`（`catalog:`）；不加 husky、lint-staged、react-doctor，不加脚本 | 命令的入口是 Makefile |
| `turbo.json`（仓库根目录） | `globalDependencies` 去掉 `.npmrc`；`globalEnv` 去掉没有代码读取的 `APP_VERSION`、`LOG_LEVEL`、`VITE_APP_VERSION` 和 10 个 Sentry 变量；`check:format`、`fix:format` 不缓存（`cache: false`），`check:lint`、`fix:lint` 依赖 `^build`（M0 加固） | 没有迁入 `.npmrc`；这些变量只属于没有迁入的应用。oxfmt 按 `.oxfmtrc.json` 指定的样式表排序 Tailwind 类名：样式表在另一个工作区包（`tailwind-config`）里，又经 `@import` 引入 npm 包的样式，这些文件都不在各个包格式检查任务的哈希里，改了样式表会重放旧的通过结果（M0 对抗性评审 Minor 1）；不缓存时，全部包的格式检查约 2.5 秒。oxlint 的 import 规则会读取依赖包构建出的 `dist/`，依赖 `^build` 之后，这些包变了会让 lint 的缓存失效，`dist/` 也保证已经生成；`check:types` 本来就依赖 `^build`，`lint-web` 不会因此多花时间 |
| `.oxlintrc.json`（仓库根目录） | `ignorePatterns` 加入 `web/packages/api-client/src/schema.gen.ts` | 生成的文件不做 lint |
| `.oxfmtrc.json`（仓库根目录） | `sortTailwindcss.stylesheet` 改为 `web/packages/tailwind-config/index.css`；删掉 `packages/codemods` 的覆盖项；加入 `ignorePatterns`：`api/dist/**`、`web/packages/api-client/src/schema.gen.ts`、`web/packages/i18n/src/types/keys.generated.ts` | 工作区的路径（路径不对时，oxfmt 检查 web 会崩溃）；codemods 没有迁入；生成的文件不检查格式 |
| `web/apps/web/package.json`，editor、i18n、propel、ui、utils 的 `package.json` | `check:lint` 的 `--max-warnings` 调低到实测的警告数：web 11957→779，editor 416→75，i18n 9→3，propel 3605→59，ui 66→32，utils 38→34 | lint 警告基线只降不升（M0 设计 5.2） |
| `web/apps/web/vite.config.ts` | 开发服务器加上代理：`/api` 转发到 `http://127.0.0.1:8080` | 开发时由 Go 后端（`make run`）回答接口请求 |

### 1.2 端到端测试加入时的改动（M0/P6）

只涉及 pnpm 工作区和依赖锁定，不改动迁入的 Plane 代码。

| 位置 | 改动 | 原因 |
|---|---|---|
| `pnpm-workspace.yaml`（仓库根目录） | `allowBuilds` 新增三项，都是 `false`：`cpu-features`、`protobufjs`、`ssh2` | testcontainers 经 dockerode 间接依赖它们，带安装脚本；pnpm 11 遇到没有登记的安装脚本会报错退出。这三项编译通过 SSH 连接 Docker 时用的可选扩展，或只检查版本号，用不到，不运行（[P6 spec](M0-foundation/specs/P6-e2e-ci.md) 2.4） |
| `pnpm-lock.yaml`（仓库根目录） | 在 P5 的锁文件上先加入 e2e 的依赖（新增 109 个包，14777 行），再加入 knip（新增 56 个包，15393 行）；原有的包一个都没有少，迁入的 Plane 包本身的版本号都不变。依赖 `debug` 的包（含 web 下 13 个包的部分依赖）解析出对等依赖 `supports-color@10.2.2` 后缀；`tsdown` 的可选对等依赖 `oxc-resolver` 从 11.20.0 改为解析到 11.24.2，`@emnapi/core` 随之从 1.10.0 变为 1.11.2 | 加入 e2e、knip 后 pnpm 重新解析可选的对等依赖（[P6 spec](M0-foundation/specs/P6-e2e-ci.md) 2.4） |
| `package.json`（仓库根目录） | 开发依赖新增 `knip` 6.37.0 | 未使用代码检查（M0 只出报告，M1 起作为门禁；[P6 spec](M0-foundation/specs/P6-e2e-ci.md) 2.8） |

### 1.3 工具链、遗留物与多语言（M1/P1）

没有删除产品功能（删除的功能见第二节）。详见 [M1/P1 spec](M1-frontend-trim/specs/P1-web-hygiene.md)。

| 位置 | 改动 | 原因 |
|---|---|---|
| 各包 `package.json` 的 `check:lint`；新增 `tools/lint-cap.mjs`；`turbo.json` | `oxlint --max-warnings=N .` 改为 `node …/tools/lint-cap.mjs N`，警告数必须等于上限；`check:lint` 的输入加上这个脚本。上限：web 777、editor 75、utils 34、ui 31、propel 29、hooks 4、constants 2、types 1、i18n 1，其余为 0 | "只降不升"由检查强制（M1 设计 7.1） |
| 新增 `tools/keywords.mjs`、`tools/keywords.json`；`Makefile` 的 `lint-web`；`knip.jsonc` | 关键词守卫：`make lint-web` 先运行它，P1 的 12 条规则看住本表删掉的东西 | 删掉的东西不再长回来（M1 设计 7.4） |
| 仓库根目录 `package.json` 的 `check:lint`、`check:format`；`turbo.json` 的根任务 `//#check:lint`、`//#check:format` | `make lint-web` 也检查 `tools/` 下的脚本：oxlint 不允许警告，oxfmt 检查格式 | 门禁本身也要受门禁检查（P1 spec 第 3 节第 13 项） |
| `turbo.json`、各包 `package.json` 的脚本；`Makefile`、`.github/workflows/ci.yml` | 删掉没有调用方的任务 `build-storybook`、`check`、`clean`、`fix`、`fix:lint`、`start`，以及各包的 `clean`、`fix:lint`、`sync:check`、`storybook`、`build-storybook`、`postcss` 和 web 的 `preview`、`start`；api-client、e2e 补上 `fix:format`；`test` 任务保留，新增 `make test-web`，持续集成的 `web` 任务运行它；i18n 加 `test` 脚本和 `vitest`；新增任务 `check:sync` | 只保留 Makefile、持续集成或 README 用到的入口；前端单元测试进入持续集成（M1 设计 7.5、8） |
| ui、propel 的 Storybook | 删除配置、45 个 stories、只为它们存在的文件和代码（propel 中 5 个只被 stories 导入的文件和 `ToastStatic`、两个包的 PostCSS 配置和样式）及依赖 | 没有入口 |
| 未使用的依赖 | 删除 knip 报出的未使用依赖（editor 的 `buffer` 除外），以及只为它们存在的 `markdown-to-component.tsx`、`use-font-face-observer.d.ts`、editor 的 `postcss.config.js` | knip 的报告 |
| `pnpm-workspace.yaml`、`pnpm-lock.yaml`（仓库根目录） | 删掉随依赖失效的 catalog 条目、不再作用于任何包的 7 条 overrides（含 `path-to-regexp: 0.1.13`）、allowBuilds 的 `@swc/core`、minimumReleaseAgeExclude 中的 7 个 Storybook 包；删掉英文注释中讲没有迁入内容的句子。锁文件一共删掉 369 个包，没有新增包，存活包的版本和完整性哈希都不变 | 锁文件和一次性脚本逐条核对 |
| 13 个 `.prettierignore`、`.oxfmtrc.json`（仓库根目录） | 删除 `.prettierignore`；`ignorePatterns` 加上 `web/apps/web/.react-router/**`、`web/apps/web/build/**`、`web/packages/*/dist/**`，删掉 `keys.generated.ts` | oxfmt 会读取 `.prettierignore`，改为统一放在一处 |
| `.oxlintrc.json`、`knip.jsonc`、`.gitignore`（仓库根目录） | 删掉 Storybook 的忽略规则、i18n 的 `ignoreUnresolved`、`keys.generated.ts` | 对应的东西已删除 |
| `web/apps/web/.gitignore`、`manifest.json`、`public/favicon/` | 删除 | 只有 Sentry 的条目；没有被引用 |
| `packages/services` | 删掉没有调用方的 49 个文件 | M1 设计 3.5 |
| `packages/tailwind-config/package.json` | 加 `"type": "module"`；删掉指向不存在文件的 `main`，入口由 `exports` 声明 | 构建警告 `MODULE_TYPELESS_PACKAGE_JSON`；knip 的提示 |
| `web/apps/web/vite.config.ts` | `vite-tsconfig-paths` 插件换成 `resolve.tsconfigPaths: true`；删掉 `next/script` 的别名，连同垫片中没人用的 `image.tsx`、`script.tsx`、`next-script.d.ts` | Vite 8 内置；死代码 |
| `packages/typescript-config/base.json` | `tsBuildInfoFile` 改为 `${configDir}/.turbo/tsconfig.tsbuildinfo` | 原来各包写同一个文件 |
| `app/root.tsx`、`core/components/common/logo-spinner.tsx` | `HydrateFallback` 不再读取主题；加载动画用 CSS 的 `dark:` 变体选图片 | 修复 React #418：预渲染与首次渲染的标记不一致 |
| `packages/i18n`；web 的 `profile.store.ts` 和语言选择框 | 只留 `en`、`zh-CN`；新增 `toSupportedLanguage`（带单元测试），不支持的语言按英文处理；删除翻译键的生成；`sync-check.ts` 改为双向核对 en 与 zh-CN，保留跨命名空间的冲突检查；删除 8 个企业版命名空间和 `workspace_settings.settings.applications` | 只留中英文（M1 设计 6） |
| `Makefile` 的 `web-dev` | `turbo run dev --filter=web... --concurrency=12` | 改了 `web/packages/*` 的源码，运行中的页面随之更新 |

---

## 二、删除的功能（M1）

每个功能都要删到这些层面：路由、导航和菜单入口 → 组件、store、services、hooks → 类型和字段 → 常量和枚举 → 多语言文案 → 不再使用的依赖。

| 功能 | 状态 | 完成于 |
|---|---|---|
| 文档页（Pages）及协作编辑模式（Yjs、Hocuspocus）；随它一起删除的更新日志（"what's new"）、新手导览中的文档页一步、编辑器内核中只为协作和文档页存在的部分（`getDocument`、AI 处理器、AI 菜单、文档信息与标题回调、页面专用的排版变量） | 已完成 | M1/P2 |
| 估算（Estimates）；迭代和模块的进度改为只按工作项数计算，`calculateCycleProgress` 和乐观更新有单元测试 | 已完成 | M1/P2 |
| 甘特图与时间线（包括模块的时间线视图）；保留的 `REVERSE_RELATIONS` 从 `constants/gantt-chart.ts` 移到 `constants/issue/relation.ts` | 已完成 | M1/P2 |
| "自动化"设置页中的自动关闭（自动归档保留） | 已完成 | M1/P2 |
| 数据分析，连同 `:workspaceSlug/analytics` 旧地址重定向、侧边栏入口、power-k 命令和 propel 中只有它用到的图表和表格（`line-chart`、`radar-chart`、`scatter-chart`、`tree-map`、`table`）；保留的迭代、模块进度代码由 analytics 改名为 progress | 已完成 | M1/P2 |
| 导出，连同导出插图和导出格式图标，以及它借用的、从未创建过的集成服务（集成与导入器的 service、类型和 `integration` 文案命名空间）；集成与导入器的其余残留见下面 Plane 死代码一行 | 已完成 | M1/P2 |
| 便签，连同 `react-masonry-component`；工具栏随之只剩一组（去掉 `TEditorTypes` 和只属于文档页的排版、表格两项） | 已完成 | M1/P2 |
| 首页快捷链接和首页个性化；自定义主题（首页固定显示问候、无项目空状态和最近访问，见 [M1 设计](M1-frontend-trim/M1-design.md) 3.14）；已经没有入口的旧首页仪表盘 | 已完成 | M1/P2 |
| 侧边栏的自定义导航（固定、排序、隐藏菜单项；数据在已砍掉的 `workspace_user_preferences` 表），侧边栏改为固定列表；保留的项目导航偏好改由 `ProjectNavigationDialog` 配置 | 已完成 | M1/P2 |
| AI 助手，连同 Plane AI 的侧边栏入口和实例的 AI 开关；遥测残留（实例的遥测开关，应用里没有遥测 SDK） | 已完成 | M1/P3 |
| Unsplash 封面图（封面选择只留静态图和上传，打开选择器不再请求 `/api/unsplash/`） | 已完成 | M1/P3 |
| 公开发布（发布弹窗、指向 space 的链接），连同评论的内部 / 外部可见范围 | 已完成 | M1/P3 |
| 管理后台（god-mode）入口，连同"实例未完成设置"页（实例请求失败时的维护页保留） | 已完成 | M1/P3 |
| 个人主页的统计和动态，连同 propel 中最后只有它用到的 `bar-chart`、`pie-chart`；个人主页只剩用户卡片和工作项分页，`/profile/:userId` 重定向到"分配给他的" | 已完成 | M1/P2 |
| 项目邀请：web 里没有按邮件邀请进项目的流程，"邀请成员"弹窗改名为添加成员，离开项目、私有项目的文案不再说邀请 | 已完成 | M1/P3 |
| 第三方登录、验证码登录、找回 / 重置 / 设置密码、登录前的"检查邮箱"步骤；登录和注册各剩一个"邮箱 + 密码"表单，模式由路由决定（CSRF 移到 M2，见第三节 3.2） | 已完成 | M1/P3 |
| 修改登录邮箱（靠邮件验证码完成） | 已完成 | M1/P3 |
| Plane 的旧地址重定向（`routes/core.ts` 末尾的 11 条；数据分析的一条随数据分析删除，其余随 Next.js 兼容层一起删除，先改掉仍依赖它们的入口） | 计划中 | |
| 邮件通知偏好设置页，连同营销邮件同意 | 已完成 | M1/P3 |
| 企业版残留中的"活跃迭代"推广页（工作区级；项目迭代列表中的"当前迭代"区块保留） | 已完成 | M1/P2 |
| 企业版残留：Epic、团队、工作项类型、计费和升级提示、批量操作及其工作项多选、工作项模板、工时记录、重复工作项、工作流和项目更新的空壳；企业版扩展点（`extended`、`additional` 空壳，只为企业版子类存在的 Base 类加别名，富文本筛选中空的扩展一半）；收藏的实体类型改为联合类型 | 已完成 | M1/P3 |
| Plane 自身的死代码：IndexedDB 和同步代码、调用不存在接口的 service 方法、集成与导入器的残留（Jira 图标、集成和导入的空状态图、设置和计费页中的文案）、knip 报告的未使用文件和导出；knip 改为门禁 | 已完成 | M1/P3 |
| 多语言：只保留 `zh-CN` 和 `en` | 已完成 | M1/P1 |
| Next.js 兼容垫片（`app/compat/next/*` 及 Vite 别名）：`next/link`、`next/navigation` 的约 330 处引用全部改为 React Router 原生写法（`Link`、`useParams`、`useLocation`、`useSearchParams`、`useNavigate`）；去掉强制结尾 `/` 和延迟跳转，修复因此暴露出的"渲染时跳转"问题；删除垫片（两个未使用的文件 `script.tsx`、`image.tsx` 已在 M1/P1 删除） | 计划中 | |
| web 中的部署遗留：`Dockerfile.web`、`Dockerfile.dev`、`caddy/`、`.dockerignore` | 已完成 | M1/P1 |
| `serve` 依赖及其 `start`、`preview` 脚本（当前运行即崩溃） | 已完成 | M1/P1 |
| `public/` 中从未注册的 `sw.js` 及 workbox 相关文件 | 已完成 | M1/P1 |
| `.env.example`（整个文件） | 计划中 | |

**验收标准**（详见 [M1 设计](M1-frontend-trim/M1-design.md) 第 7、11 节）：
- TypeScript 类型检查通过，knip 为零；oxlint 不超过新的警告基线（M8 发布前清零，见 M1 设计 7.3）。
- 关键词守卫没有未登记的命中。
- 保留功能的行为按 M1 设计 7.5 核对。
- 能正常构建。

---

## 三、对接新接口（M2 起按领域推进）

原则见 [v0-design 7.2](v0-design.md#72-对接新接口不设转换层前后端数据结构统一)：**不设转换层，前后端数据结构统一。**

- `packages/api-client`：由 `api/openapi.yaml` 生成（openapi-typescript + openapi-fetch）。
- `packages/types`：实体类型直接使用生成的类型；只有纯界面用的类型（显示设置、布局参数等）才手写。原来手写的 Plane 实体类型，随各领域对接新接口时删除。
- `core/services`：改为只调用生成客户端的薄封装，不做任何数据转换。
- stores 和组件：直接使用新接口的数据结构，不保留任何为兼容 Plane 旧接口、旧字段而存在的代码。

### 3.1 各领域的对接进度

| 领域 | 所属 M | 状态 |
|---|---|---|
| 认证、用户、实例配置、PAT；令牌管理器 | M2 | 计划中 |
| 工作区、成员、邀请、项目、项目成员、项目归档、状态、标签、显示设置 | M3 | 计划中 |
| 工作项、列表（分页和分组的新结构）、子任务、关联、链接、评论、表情回应、操作动态、搜索、历史版本、草稿、工作项归档 | M4 | 计划中 |
| 文件、附件、编辑器图片（上传改为 `{method, url, headers}` 形式的 PUT） | M5 | 计划中 |
| 迭代、模块（归属改为工作项字段）、迭代和模块归档 | M6 | 计划中 |
| 通知、收集箱、视图、收藏、最近访问 | M7 | 计划中 |
| Webhook 设置、接口调用日志 | M8 | 计划中 |
| 删除对 `@plane/services` 的依赖 | M5 | 计划中 |

### 3.2 已知的结构性改动

| 位置 | 改动 | 原因 | 状态 | 完成于 |
|---|---|---|---|---|
| `core/store/issue/helpers/base-issues.store.ts` 等列表相关 store | 使用新接口的分页结构（不透明游标 `next_cursor`）和分组结构（`groups` 数组），不再按"已加载条数 ÷ 每页条数"拼页码游标 | 新接口的分页和分组设计 | 计划中 | |
| 用户和认证相关的 store | 登录、退出、续期改走令牌管理器 | 认证改为 Bearer 令牌 | 计划中 | |
| 登录、注册、退出、修改密码的提交方式 | 删除 CSRF 令牌和 Django 会话的表单提交，改走令牌管理器 | 认证改为 Bearer 令牌；CSRF 是传输方式的一部分，和它的替代品一起删除（[M1 设计](M1-frontend-trim/M1-design.md) 3.6） | 计划中 | |
| 所有处理接口错误的地方 | 统一按 RFC 9457 的 problem+json 读取 `code`、`title`、`errors` | 错误格式统一 | 计划中 | |
| 文件上传相关的 store 和调用方 | 预签名 POST 改为 `{method, url, headers}` 形式的 PUT | 文件存储改为 PUT 上传 | 计划中 | |
| 迭代和模块的归属 | 通过工作项的 `cycle_id`、`module_ids` 字段修改，不再调用单独的接口 | 接口设计 | 计划中 | |

---

## 四、品牌

| 项 | 状态 | 完成于 |
|---|---|---|
| 替换 Logo 和网站图标 | 计划中 | |
| 替换页面标题和文案中的"Plane"，改为 Nerve | 计划中 | |
| 内部包名 `@plane/*` 改为 `@nerve/*` | 计划中 | |
| `web/` 中来自 Plane 的文件保留原有的版权声明 | 已完成 | M0/P5 |
