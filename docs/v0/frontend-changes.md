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
| 第三方依赖 | `@makeplane/propel` 0.3.0：Plane 发布在 npm 上的设计系统，AGPL-3.0-only，版本锁定（`pnpm-workspace.yaml` 的 catalog，不跟随发布更新）。tarball 的完整性哈希 `sha512-nGhiE42vLQVvv7NZOcQYKARJiTXyKDSSTcHOzPdtFpa+WkSz9B91jw6i+3Ikz5cpkv/aEKjOTJF+cByw2zv5ZQ==`（`pnpm-lock.yaml`）。源码：npm 上这个版本带 SLSA 来源证明，记录它由 `github.com/makeplane/propel` 的 `packages/propel`、提交 `0a31b1529c0f249a058e59ade313d8b34ea8f964` 经 `.github/workflows/release.yml` 构建；这个仓库不公开（M1/P5 时访问为 404）。包里的 source map 带着 1376 个源文件中 1375 个的全文。以后要自己修改设计令牌时，按 [M1 设计](M1-frontend-trim/M1-design.md) 3.10 评估并入源码 |

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

### 1.4 去掉 Next.js 兼容层（M1/P4）

路由、跳转和地址的改动见第二节"Next.js 兼容垫片"一行。详见 [M1/P4 spec](M1-frontend-trim/specs/P4-router-native.md)。

| 位置 | 改动 | 原因 |
|---|---|---|
| `web/apps/web` 的 `vite.config.ts`、`package.json`；`turbo.json`、`pnpm-workspace.yaml`（仓库根目录） | 删掉 dotenv 的加载、`define: { "process.env": … }` 和 `next/*` 的别名；`globalEnv` 只留 `NODE_ENV`，删掉 6 个 `VITE_*` 和 `DEV`；`build` 任务删掉只为 `.env*` 存在的 `inputs`；web 的开发依赖和 catalog 删掉 `dotenv` | 前端没有环境变量：与接口同源部署，接口一律用相对路径（M1 设计 4.1） |
| `packages/constants`、`packages/services`、web 的 services | 删掉 `endpoints.ts`（`API_BASE_URL` 等 8 个常量）；`APIService` 不再接收基础地址，`normalizeAPIRequestURL` 只给相对地址加结尾 `/`（带单元测试） | 同上 |
| `packages/i18n` | 开发环境的判断改为 `import.meta.env.DEV`；开发依赖加 `vite`，`tsconfig.json` 加 Vite 的环境类型 | 同上 |
| `web/apps/web/package.json`、新增 `vitest.config.ts` | 加 `test` 脚本和开发依赖 `vitest`；vitest 有自己的配置，不加载 React Router 的构建插件 | 路由匹配的单元测试 `app/routes/navigation.test.ts`（M1 设计 4.2、7.5） |
| `packages/typescript-config`、`.oxlintrc.json`、editor 的 `package.json`、`app/(all)/layout.preload.tsx` | 删除 `nextjs.json` 和内容全被注释掉的 `layout.preload.tsx`；删掉 `.next/**` 的忽略规则和 `nextjs` 关键词 | Next.js 的遗留（M1 设计 4.1） |
| `tools/keywords.json` | 新增规则 `frontend-env`、`next-shim-files`、`next-shims`；顶层 `phase` 改为 `M1/P4` | 删掉的东西不再长回来（M1 设计 7.4） |
| `.gitignore`（仓库根目录） | 删掉 `!.env.example`；`.env`、`.env.*` 仍被忽略 | 唯一的 `.env.example` 已删除 |
| web 的 `core/store/router.store.ts`、`use-project-issue-properties.ts`、`use-workspace-issue-properties.ts` | 路由参数的类型从 Node 的 `ParsedUrlQuery` 改为 React Router 给的 `Record<string, string \| undefined>`，两个 hook 的参数从 `string \| string[] \| undefined` 改为 `string \| undefined`；删掉没有读取方的 4 个 getter（`profileViewId`、`peekId`、`issueId`、`inboxId`） | Next.js 的 `useParams` 返回 `string \| string[]` 时留下的类型 |
| web 应用、`packages/constants`、`packages/utils` | 删掉对字符串的空转换 `.toString()`：web 699 处（Task 7 的 671 处，加上类型改正之后露出的 28 处），`fetch-keys.ts` 10 处、`emoji.ts` 2 处；由类型检查器按接收者的类型判断，留下的每一处都不是字符串 | 同上；空转换不改变值，删掉后类型如实 |

### 1.5 品牌与包名（M1/P5）

界面上不再有 Plane 的名称和 Logo，工作区的包名全部是 `@nerve/*`。详见 [M1/P5 spec](M1-frontend-trim/specs/P5-brand.md)。

| 位置 | 改动 | 原因 |
|---|---|---|
| `web/` 的源码、样式和各包的 `package.json`、`tsconfig.json`；`tools/keywords.json`；`pnpm-lock.yaml` | 包名 `@plane/*` 改为 `@nerve/*`：1100 个文件里的 2922 处（`docs/` 除外，那里是历史记录），`pnpm install` 重写锁文件；对基线锁文件做同样的替换之后与新锁文件逐字节相同，没有依赖变化 | M1 设计 5；`@makeplane/propel` 是第三方包，不改（设计 3.10） |
| 导入的分组注释（671 个文件） | `// plane imports` 这类 706 行改为 `// nerve …`（其中一行标注的是本目录 `./` 的导入，改为 `// local imports`）；Plane 的 CE/EE 目录留下的 `// plane web …` 按下一行的导入改名（`// components`、`// hooks`、`// nerve imports` 等），与相邻一组同名的合并；没有标注任何导入的 28 行删除 | 随包名 |
| 新增 `app/assets/brand/`、`core/components/common/nerve-logo.tsx`；`public/icons/`、`site.webmanifest.json`；`app/root.tsx`；propel 的 `icons/brand/`、`icons/sub-brand/` | Nerve 的图标、横版标志（浅色和深色背景各一）、网站图标、应用图标和分享图，版权声明写在 SVG 和 `app/assets/brand/SOURCES.md` 里（`public/icons/` 的两个应用图标也登记在那里，因为 `public/` 的文件原样发布）；`NerveLogo`、`NerveLockup` 按文件名读取它们，换 Logo 只换文件。删除 propel 的 `PlaneLogo`、`PlaneLockup`、`PlaneWordmark`、`PlaneNewIcon` 和只给登录页页脚用的客户 Logo（页脚一起删除）、从未被读取的第二份应用清单 `manifest.json`、没有引用的三张 Plane 图片 | M1 设计 5 |
| `core/components/common/logo-spinner.tsx` | 加载动画从两张共约 1.4 MB 的 GIF 改为 Nerve 的图标加 CSS 的 `animate-pulse`；图标在深浅两种背景上都看得清，标记与主题无关，预渲染和首次渲染仍然一致 | P1 评审的交接；保留 #418 的修复 |
| `app/assets/instance/maintenance-mode-*.svg`、`app/assets/onboarding/issues.webp`；`app/assets/workspace/workspace-not-available.png`、`workspace-creation-disabled.png`；`app/assets/empty-state/project-settings/no-projects-light.png`、`-dark.png`；`app/assets/empty-state/disabled-feature/modules-light.webp`、`modules-dark.webp`、`views-light.webp`、`views-dark.webp`；`core/layouts/auth-layout/workspace-wrapper.tsx` | 维护页插图里屏幕上的 Plane Logo 删掉；导览图片里标题为 "Plane integration" 的卡片改为 "API integration"；"工作区不存在"和"不允许创建工作区"两张图底部蓝色圆徽上的 Plane 像素 Logo 改为圆形的 Nerve 图标（`#155E75` 的圆盘、白色的 N 和两端的圆点，取自 `mark.svg`，投影保留）；项目设置空状态两张图的卡片网格上由着色方块拼成的 Plane Logo 去掉，着色的格子用旁边未着色的格子填回；模块、视图关闭时的四张图面包屑里的工作区名 "Plane Design" 改为中性的演示名 "Acme Design"（字色、字号、基线不变）；"工作区不存在"那张图是装饰，`alt` 为空，下面的标题说的是同一句话 | 图片里的品牌关键词守卫看不到，逐张目视；后 8 张是整分支评审找到的，修复轮按原尺寸重看了代码引用的全部图片和 propel 的插图，没有别的 |
| en、zh-CN 的文案；页面标题和元数据（`@nerve/constants` 的 `SITE_NAME`、`app/root.tsx`）；登录、注册、引导、导览、邀请页等写死的文案 | 讲产品的 "Plane" 改为 Nerve；讲 Plane 公司、Plane 服务的句子按含义改写（例如复制令牌的提示不再建议存进 "Plane Pages"）；没有引用、又写着 Plane 的 7 组文案删除（每种语言 16 个键）；`SITE_TITLE`、`SITE_URL`、`TWITTER_USER_NAME` 和 `og:url`、`twitter:site` 删除；页面的元数据读取这些常量，`og:description`、`keywords` 改为读取 `SITE_DESCRIPTION`、`SITE_KEYWORDS`，不再在 `app/root.tsx` 里重复它们的文字；Plane 收集箱机器人（`intake@plane.so`、名字带 `-intake`）的特殊显示删除，创建者按普通用户显示 | M1 设计 5 |
| 帮助菜单、命令面板的帮助命令、错误页、维护页、登录表单、顶部栏、邀请页、空的项目设置页 | 文档和问题反馈指向 Nerve 的仓库（`@nerve/constants` 的 `REPOSITORY_URL`）；论坛、支持邮箱、状态页、X 账号、服务条款和隐私政策、"Star us on GitHub"、plane.so 的介绍链接连同显示它们的组件、图片和文案删除 | 同上 |
| 代码标识符、存储键、剪贴板类型、组件名、注释 | `PlaneVersionNumber` → `VersionNumber`；会话存储键 `__plane_chunk_reload` → `__nerve_chunk_reload`（不迁移旧值：只在一个标签页的会话里有效）；编辑器的剪贴板类型 `text/plane-editor-html` → `text/nerve-editor-html`（两个写入方和读取方一起改）；13 个 `displayName` 从 `plane-ui-*` 改为 `nerve-ui-*`；注释、editor 包的描述和 Readme、`tailwind-config/AGENTS.md`（"plane" 在那里指层次，改为 "stacking context"）不再出现 plane | 同上 |
| Nerve 在迁入之后新写的 12 个文件（测试、测试配置、`use-profile-member.ts`） | 版权声明从 Plane 的改为 `Copyright (c) 2026-present OpenNerve` 和 `SPDX-License-Identifier: AGPL-3.0-only`；迁入后新增、但内容来自 Plane 代码的 6 个文件保留 Plane 的声明 | 按实际来源标注（M1 设计 5） |
| `tools/keywords.json` | 新增规则 `plane-package`、`brand`、`brand-files`，例外 1 条（`pnpm-workspace.yaml` 里记录来源的 3 行注释，到 M9）；顶层 `phase` 改为 `M1/P5`；三条旧规则的不命中样本原来引用本 Phase 删除或改名的代码（`deploy-files` 的 `public/manifest.json`，`integrations` 的 GitHub 图片导入和 "Star us on GitHub" 文案，`changelog` 的 `PlaneVersionNumber` 导入），改为引用保留的代码，`integrations` 的说明随之改正 | 删掉的东西不再长回来（M1 设计 7.4）；不命中样本要引用保留的代码 |

### 1.6 收尾（M1/closeout）

没有删除产品功能；删掉的是从不渲染、没有读取方的代码和资源（第二节最后两行）。详见 [M1 收尾 spec](M1-frontend-trim/specs/closeout.md)。

| 位置 | 改动 | 原因 |
|---|---|---|
| `tools/keywords.mjs`、`tools/keywords.json` | Phase 可以写成 `M<n>/closeout`，排在该 M 的所有编号 Phase 之后；顶层 `phase` 改为 `M1/closeout`，到期的例外由工具报出，只剩跨 M 的 3 条（`analytics` 到 M6、`project-invitations` 到 M3、`brand` 到 M9）。每条规则的每个分支（内容、路径、文件范围）都有命中样本，不命中样本都引用保留的代码（新增 41 条命中样本；不命中样本替换 60 条，其中 26 条是从未存在的 `app/assets/logo.svg`，另删除 5 条）。新增规则 `window-open`、`app-rail`、`app-rail-files`，共 51 条 | M1 设计 7.4；P5 评审 |
| 12 处 `window.open`（共 16 处） | 都传 `"noopener,noreferrer"`：编辑器打开成员写的链接、附件列表和图片工具栏打开可能由别的源提供的文件，以及同源的"在新标签页打开" | 打开的页面拿不到 `window.opener`（P5 评审）；`target="_blank"` 的 18 个元素都带 `rel="noopener noreferrer"` |
| 应用栏 | 删除 `core/lib/app-rail/`、`navigation/app-rail-*.tsx`、`items-root.tsx`、只有它用的 `use-workspace-paths.ts`、它的显示偏好和 `@nerve/types` 里的类型，以及只为它存在的 propel 右键菜单分隔线和两个 `className` 属性；顶部栏永远走不到的 `px-2` 分支删除 | `AppRailVisibilityProvider` 默认关闭，唯一的使用处不打开它：藏在开关后面的功能（P4 评审裁定 9） |
| 工作区包的导出 | 没有其他文件读取的包导出，三轮删完（512 个）：274 个连同声明删除，238 个在自己的文件里还有人用，只去掉导出（220 个 `export` 关键字、18 个 `export { … }` 里的说明符），另外去掉桶文件里转出它们的 34 个说明符；166 个文件删除（propel 的 accordion、avatar、badge、banner、collapsible、combobox、command、dialog、input、skeleton、switch、tabs、toolbar 组件，图标注册表连同只有它提到的图标（111 个文件），ui 的 avatar、collapsible、input、tag、textarea 等）；没有导入方的 16 个包入口；propel 的 `cmdk` 依赖；utils 的 `tlds.ts` 和守卫里它的 2 条例外（到 M9）；只有被删的 `EmptyState` 传入的 `asset` 属性；只被这些代码提到的 22 个文案键 | knip 把包的入口都当作已使用，看不到它们（P3、P4 评审） |
| 点名的死代码 | `useProjectIssueProperties`（5 个 fetcher 只有一个有调用方）和工作项表单挂载时什么也不做的重置，表单直接用迭代 store 的 `fetchAllCycles`；propel `Menu` 和 `ContextMenu` 的子菜单、ui `CustomMenu` 的静态成员 `Portal`、`SubMenuTrigger`、`SubMenuContent`；`EmptySpace` 没人传的 `Icon`、`description` | P3、P4、P5 评审 |
| propel 的 `MenuItem`；`.oxlintrc.json` | 点菜单项时不再调用全局的 `close()`：它就是 `window.close()`，由脚本打开的标签页会被关掉。oxlint 规则 `no-restricted-globals` 按 confusing-browser-globals 的列表禁用这类全局名，`use-reload-confirmation` 的 `confirm` 改为 `window.confirm` | 缺陷修复 |
| `.oxlintrc.json` 和 58 个文件；四处别名、模板字面量、导入分组注释、插图的 `alt`、`list-view-types.d.ts` | `react/jsx-curly-brace-presence` 设为错误，基线的 80 处 `={"…"}` 等由 oxlint 修掉；只给导入常量起别名的 4 个 `const`、没有插值的模板字面量、`t(\`${…}\`)` 外面多余的模板删除；悬空的 204 行、重复的 4 行导入分组注释删除，还有 3 行只有 `//` 的注释和叠在另一个标签上的 `//hooks`（不重排导入）；三张装饰插图的 `alt="ProjectSettingImg"` 改为空；`TPlacement` 从 propel 不导出的子路径导入，在 `skipLibCheck` 下悄悄成了 `any`，改为 `CustomMenu` 的 `placement` 类型 | 退化结构（P4、P5 评审） |
| 编辑器的标注块（callout） | 属性按 HTML 里的字符串读取（`parseHTML`），emoji 的码点不再被 TipTap 转成数字，`logo-selector.tsx` 的 `.toString()` 删除；带单元测试 | P4 评审 |
| `@nerve/utils` 的 HTML 工具；编辑器的标注块；`pnpm-workspace.yaml` | 删除 `sanitize-html` 和 `@types/sanitize-html`：文本和"是否为空"改由浏览器的 `DOMParser` 读取（应用是纯客户端的构建），通知预览不再显示 `&amp;`，标注块不再转义本地存储里的 `&`；postcss 不再进入浏览器端的依赖，开发环境的 "externalized for browser compatibility" 警告消失（每加载一次页面，浏览器控制台和开发服务器各 22 条；P4 评审记下 66 条）；带单元测试（utils 的开发依赖加 `jsdom`） | P4 评审 |
| `patches/prosemirror-codemark@0.4.2.patch`、`pnpm-workspace.yaml` | 删掉 12 个构建文件末尾的 `sourceMappingURL` 注释：它们指向的 source map 列出的源文件没有发布；编辑器测试的输出里不再有警告 | P3 评审裁定 6；0.4.2 是最新版本（2022 年） |
| `tailwind-config`、`typescript-config` 的 `package.json`；仓库根目录的 `package.json` | 两个包加 `check:format`、`fix:format`（oxlint 在它们里面没有可读的文件）；根目录的格式检查加上 `package.json`、`pnpm-workspace.yaml`、`turbo.json`、`knip.jsonc`、`.oxlintrc.json`、`.oxfmtrc.json`；`typescript-config` 删掉只在打包发布时起作用、又只列了 4 个配置中 3 个的 `files` | P1 评审第 6 节 (b) |
| en、zh-CN 的文案 | 删掉没有代码引用的 500 个键（严格方法下的 482 个，加上 18 个只因名字与无关的字面量相同而算作有引用的键），以及它们留下的 149 个空对象；每种语言 1577 → 1077 个键；主题选项的标签只剩一个来源（`i18n_label` 是键），Power K 的主题菜单在中文界面下不再显示英文 | M1 设计 6、9 节 |
| `app/assets/` | 删掉 133 张没有被导入的图片（6.1 MiB）：126 张空状态插图（画的是 Plane 的界面）、认证页的三张、项目 emoji、命令键图形和两张图库人像 | M1 设计 9 节；P5 评审 |
| `app/assets/cover-images/`、`helpers/cover-image.helper.ts` | 29 张预设封面从来源和许可查不到的照片换成 Nerve 自己画的抽象图（色块、波纹、圆盘、条带、切面、点阵、等高线，中到深色，画面中部有细节：封面会被裁成宽条，上面写白字）。SVG 是源文件，带 Nerve 的版权声明；应用导入由它渲染的 WebP（1920 × 1080，质量 0.9），因为项目或个人资料选用预设封面时应用会上传一份副本，上传只接受 JPEG、PNG、WebP。同目录的 `SOURCES.md` 写明做法和每张画的是什么。旧的 29 张 JPEG 删除，封面辅助函数导入新文件，上传的兜底文件名随格式改为 `image.webp` | 收尾 spec 第 9 节第 1 条的裁定：来源和许可查不到的图片不随 Nerve 发布，功能保留 |
| `app/assets/onboarding/`、`app/assets/empty-state/`；`cycles/active-cycle/root.tsx` | 导览的三张截图和"功能未开启"的八张图里，显示照片的头像（人像、卡通人物和盖在照片上的计数徽章，11 张图共 77 个）改为字母头像，真实的人名改为中性的名字（"vera"；"Robin"、"Robin Park"），用应用的字体 Inter 写在原来的位置，字号、字重、颜色和基线按原文拟合；此外没有改动，只是多了一次有损的 WebP 编码（取文件大小最接近原图的质量）；同目录的 `SOURCES.md` 保留上游的版权声明，写明每张改了什么。删掉不显示的 8 张图：按路径没有被引用的 6 张（与在用的图同名，或是另一张图文件名的结尾，按文件名查找时被当作在用）；活动迭代的两张，根组件按主题选出它们，传给内层组件一个不读的属性，这个属性和只为它存在的主题 hook 一起删除 | 收尾 spec 第 9 节第 1 条的裁定：保留的图片不含第三方的肖像和个人信息；保留的其余图片按能看清 20 px 细节的尺寸复看过（收尾 plan 的 Task 15） |
| propel 右键菜单的 `Trigger`、`Content` | 属性类型去掉 `className`（`Omit`）：T3 删掉了它们自己声明的 `className`，可它们继承的 base-ui 属性里还有，传进来时 `Trigger` 会用它换掉 `outline-none`，`Content` 会不声不响地丢掉；现在传它是类型错误 | 收尾的任务评审（修复轮） |
| 5 个文件的导入分组注释 | 4 行只有 `//` 的标签：`count-chip.tsx` 的删除，迭代工作项 store 的删除（下面的本地导入并入上面的本地一组），工作项基类 store 和模块工作项 store 的改为 `// local imports`（删掉会把本地导入归到 `// services`、`// helpers` 下）；模块链接列表项盖在 toast、tooltip 和类型上的 `// nerve types` 改为 `// nerve imports`，盖在 `@nerve/utils` 上的 `// nerve ui` 改为 `// nerve utils`。只改注释行，不移动导入 | 收尾的任务评审（修复轮） |
| 编辑器的标注块（callout） | 插入标注块时从本地存储读上次用的图标，现在先检查形状：`in_use` 是 `emoji` 或 `icon`，对应的 `value` 或 `name` 是非空字符串，`url`、`color` 有的话是字符串；不符合的值写日志、删掉、改用默认值，与存的不是 JSON 时一样。原来 `as TLogoProps` 直接信任：存着 `null` 时插入就抛错，T7 删掉 `.toString()` 之后，不是字符串的 emoji 值会传到 `stringToEmoji(…).split`；带单元测试 | 收尾的任务评审（修复轮） |
| 草稿弹窗；`@nerve/utils` 的 `isEmptyHtmlString` | 只有图片或只有提及的草稿，关闭时不再不问就丢掉：弹窗判断描述是否为空时只把 `img` 算作内容，编辑器写的却是 `<image-component>`、`<mention-component>`。算作内容的标签只写在 `isEmptyHtmlString` 里（`img`、`image-component`、`mention-component`），它不再接受标签列表，弹窗和 `isCommentEmpty` 一样调用它；带单元测试 | 控制者的浏览器核对（基线和本分支都能重现；修复轮） |
| 个人主页的顶部栏 | 窗口窄于 768 px 时，菜单按钮显示 `profile.tabs.assigned` 这样的文案键：布局把当前标签的 `i18n_label` 当作 `type` 传给顶部栏，顶部栏原样显示（Plane 原有）。顶部栏改为接收当前标签本身，用 `t(currentTab.i18n_label)` 显示，与它的菜单项同一种写法 | 控制者的浏览器核对（`a3-profile.mjs`，480 px；修复轮） |
| 仓库根目录的 `package.json`、`turbo.json` | 根目录加 `fix:format`，文件列表与 `check:format` 相同；`turbo.json` 登记 `//#fix:format`（不缓存）。原来 `pnpm exec turbo run fix:format` 不格式化根目录的文件，与 README 说的"格式化每个包"不符 | 收尾的任务评审；P1 spec 的保证（修复轮） |
| `@nerve/utils` 的 `isCommentEmpty`；`@nerve/types` 的 `Content` | `isCommentEmpty` 只接受评论的 HTML（`string` 或 `undefined`），三个调用方传的都是它；从来走不到的 JSON 分支删除（它列的节点类型写的是 HTML 标签名，编辑器的节点叫 `mention`、`imageComponent`），连同只有它用的 `Content`、`HTMLContent`；utils 的 oxlint 上限 19 → 18。`JSONContent` 保留：它描述 TipTap 的 JSON，只写一半的字段不如不写；它唯一的使用者 `TIssueComment.comment_json` 没有读写方，交 M4 一起处理 | 修复草稿弹窗时发现（修复轮）：删除，不隐藏 |
| 编辑器的粘贴处理（`helpers/paste-asset.ts`、`helpers/asset-duplication.ts`） | 粘贴编辑器自己的剪贴板类型 `text/nerve-editor-html` 时，要复制的图片改在 `DOMParser` 解析出的惰性文档里标记。原来是在活动文档里新建一个 `div` 来解析这段 HTML：`div` 没有挂到页面上，但它的元素属于页面，别的网页在复制事件里写进这个类型的 `<img onerror>` 会执行（Plane 原有，P5 只改了类型名）。标记之后的 HTML 照旧交给 ProseMirror，它按编辑器的 schema 解析，只留声明过的节点和属性；编辑器之间复制已上传的图片仍然复制资源。带单元测试（在旧代码上失败）；浏览器核对 `copy-probe.mjs` 在本分支 22 项通过、0 项失败，在基线 21 项通过、1 项失败（C3 的 `window.__xss` 变为 `true`） | Codex 对抗评审 Critical 1（收尾 T17） |
| `Makefile` 的 `build-web`、`build` | `build-web` 在 Turbo 之前 `rm -rf $(WEB_CLIENT)`（`web/apps/web/build/client`）：命中缓存时 Turbo 把记录下来的文件写回原处，不删目录里多出来的文件，`make build` 会把它们嵌进 Go 程序。跟进提交让 `make build` 的 `cp` 也读 `$(WEB_CLIENT)`，这个路径只有一个来源；`build-web` 的 `##` 说明仍写字面的路径，因为 `make help` 不展开变量 | Codex 对抗评审 Important 1（收尾 T18） |
| 活动迭代卡片（`cycles/active-cycle/progress.tsx`）；en、zh-CN 的文案 | 进度条改用迭代列表也在用的 `calculateCycleProgress`（`@nerve/utils`）：完成 / (总数 − 取消)。原来是 (完成 + 取消) / (总数 − 取消)，会显示 125% 和 "10/8 closed"。上方的文字由 `t("project_cycles.active_cycle.work_items_completed", { completed, total })` 写出（例如 "3/8 work items completed"），取消的说明由 `t("…cancelled_excluded", { count })` 写出：两个新键，en 用 ICU 复数，每种语言 1077 → 1079 个键；取消说明外面一层只包着文字的 `<span>` 删除。跟进提交：分组行的 `map` 原来返回一个不带 key、只有一个子元素的片段，key 写在里面的 `div` 上，React 报 key 的警告；现在每一行的 `div` 就是列表项（`key={group}`），外面那层没有属性的 `div` 删除；web 的 oxlint 上限 566 → 565（少了一个 `react/no-array-index-key`） | Codex 对抗评审 Important 2（收尾 T19） |
| `app/assets/onboarding/`、`app/assets/empty-state/disabled-feature/` | 保留的 9 张截图里涂掉 10 个已删功能的控件：导览 `cycles.webp` 的数据分析按钮和甘特图布局图标，`views.webp` 的甘特图布局图标，`issues.webp` 的 "Public" 可见性标签（项目发布）；"功能未开启"的模块、迭代各两张的甘特图/时间线布局图标，收集箱两张的 "Estimate: 2 Months" 一行。每个框用旁边平的背景色填平，只有 `intake-light.webp` 的框重复它左边一列（x=1699）的像素：那一行被卡片的下边缘截断，填纯色会抹掉卡片 650 px 长的一段底边，还把白色涂进卡片下面透明的边距，在应用里看得见（控制者的裁定）。之后按"大小最接近原图"的质量重新编码一次；两个 `SOURCES.md` 写明去掉了什么 | Codex 对抗评审 Important 4（收尾 T20） |
| zh-CN 的 `common` 文案 | 个人、项目、工作区三个设置侧边栏的分组标题 `your_profile`、`developer`、`work_structure`、`execution`、`administration` 在 zh-CN 里仍是英文，改为"您的个人资料""开发者""工作结构""执行""管理"（zh-CN 已有的说法）；zh-CN 与英文相同的值只剩不用翻译的 URL、ID、Webhooks、`name@company.com` | 控制者的浏览器核对（收尾 spec 2.10 的 I2；整分支评审之后的修复轮） |
| `@nerve/services` 的 `helpers/index.ts` | 只按名字转出 `normalizeAPIRequestURL`。原来是 `export * from "./url"`，包入口又转出 `./helpers`，没有包外读取方的 `ensureAPITrailingSlash` 也成了包的公开接口；它留在 `url.ts`，由 `normalizeAPIRequestURL` 调用、由测试按名字导入。包现在公开 `APITokenService`、`generateFileUploadPayload`、`getFileMetaDataForUpload`、`normalizeAPIRequestURL` | Codex 对抗评审 Minor 1（整分支评审之后的修复轮） |
| 工作区内容区（`workspace/content-wrapper.tsx`）的导入分组注释 | T3 删掉 `cn` 和应用栏的导入之后，`// nerve imports` 盖在唯一剩下的本地组件导入上，改为 `// components`；只改注释行 | 整分支评审（其后的修复轮） |
| 仓库根目录的 `package.json`；README | 根目录格式检查的文件列表只写在 `fix:format` 里，`check:format` 是 `pnpm run fix:format --check`（同一条 oxfmt 命令的检查模式）。原来两个脚本各写一遍 7 个路径，README 再写一遍：一处加了路径另一处没加，这个路径就只检查不格式化，或者反过来，而且什么都不报错。README 改为指向这个脚本 | 整分支评审（其后的修复轮） |

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
| Plane 的旧地址重定向（`routes/core.ts` 末尾的 11 条；数据分析的一条随数据分析删除，其余 10 条随 Next.js 兼容层一起删除）。先改掉仍依赖它们的入口：命令面板的"转到账号设置"直接去 `/settings/profile/general`；同时修好停用的迭代、模块、视图、收集箱页面上的"管理功能"按钮，改为去各自的功能设置页（原来指向没有页面的 `/:workspaceSlug/settings/projects/:projectId/features`，Plane 也是如此）。路由匹配的单元测试 `app/routes/navigation.test.ts` 核对源码里写出的内部路径和导航常量都落在真实的页面上 | 已完成 | M1/P4 |
| 邮件通知偏好设置页，连同营销邮件同意 | 已完成 | M1/P3 |
| 企业版残留中的"活跃迭代"推广页（工作区级；项目迭代列表中的"当前迭代"区块保留） | 已完成 | M1/P2 |
| 企业版残留：Epic、团队、工作项类型、计费和升级提示、批量操作及其工作项多选、工作项模板、工时记录、重复工作项、工作流和项目更新的空壳；企业版扩展点（`extended`、`additional` 空壳，只为企业版子类存在的 Base 类加别名，富文本筛选中空的扩展一半）；收藏的实体类型改为联合类型 | 已完成 | M1/P3 |
| Plane 自身的死代码：IndexedDB 和同步代码、调用不存在接口的 service 方法、集成与导入器的残留（Jira 图标、集成和导入的空状态图、设置和计费页中的文案）、knip 报告的未使用文件和导出；knip 改为门禁 | 已完成 | M1/P3 |
| 多语言：只保留 `zh-CN` 和 `en` | 已完成 | M1/P1 |
| Next.js 兼容垫片（`app/compat/next/*` 及 Vite 别名）：`next/link`、`next/navigation` 的引用和包装层 `useAppRouter` 全部改为 React Router 原生写法（`Link`、`NavLink`、`useParams`、`useLocation`、`useSearchParams`、`useNavigate`、`useMatch`）；路由参数按真实的 `string \| undefined` 处理，路由组件用 `./+types/*` 的参数，共享组件用守卫；去掉延迟跳转，`AuthenticationWrapper` 的渲染时跳转改为 `<Navigate replace />`，登录后跳回的 `next_path` 改用 `@nerve/utils` 的 `isValidNextPath` 校验（只接受以单个 `/` 开头的站内路径，不再放过 `//host`、`javascript:`；带单元测试），包装只读一次、修剪后校验，跳到校验过的那个值；页面到达时自动做的跳转（项目设置到第一个项目、收集箱到第一项、收集箱里的工作项到收集箱）改为替换当前地址，后退不再回到会再次跳走的地址；去掉强制结尾 `/`，应用内部的地址一律不带结尾 `/`，"当前是哪一项"的判断改用 React Router 的匹配；删除垫片和 `typescript-config/nextjs.json`（两个未使用的文件 `script.tsx`、`image.tsx` 已在 M1/P1 删除） | 已完成 | M1/P4 |
| web 中的部署遗留：`Dockerfile.web`、`Dockerfile.dev`、`caddy/`、`.dockerignore` | 已完成 | M1/P1 |
| `serve` 依赖及其 `start`、`preview` 脚本（当前运行即崩溃） | 已完成 | M1/P1 |
| `public/` 中从未注册的 `sw.js` 及 workbox 相关文件 | 已完成 | M1/P1 |
| 前端环境变量：`.env.example`、dotenv 和 `process.env` 的注入、各包对 `process.env` 和 `VITE_*` 的读取；停用账户的提示不再给出 Plane 的支持邮箱，改为联系管理员 | 已完成 | M1/P4 |
| 从不渲染的应用栏（`AppRailVisibilityProvider` 默认关闭，唯一的使用处不打开它），连同它的显示偏好和只有它用的 `use-workspace-paths.ts` | 已完成 | M1/收尾 |
| Plane 自身的死代码（续）：没有其他文件读取的工作区包导出、没有代码引用的文案键、没有被导入的图片 | 已完成 | M1/收尾 |

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
| 删除对 `@nerve/services` 的依赖 | M5 | 计划中 |

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
| 替换 Logo 和网站图标 | 已完成 | M1/P5 |
| 替换页面标题和文案中的"Plane"，改为 Nerve | 已完成 | M1/P5 |
| 内部包名 `@plane/*` 改为 `@nerve/*` | 已完成 | M1/P5 |
| 指向 Plane 服务的链接删除，文档和问题反馈指向 Nerve 的仓库 | 已完成 | M1/P5 |
| 代码标识符、存储键、剪贴板类型、组件名和注释不再带 plane | 已完成 | M1/P5 |
| `web/` 中来自 Plane 的文件保留原有的版权声明 | 已完成 | M0/P5 |
| `web/apps`、`web/packages` 里与 Plane 的文件放在一起的 Nerve 文件和图形标注 Nerve 的版权；放不下文件头的资源登记在 `app/assets/brand/SOURCES.md`。`web/packages/api-client` 和 `web/` 以外 Nerve 自己的代码不加文件头，以仓库根目录的 `LICENSE` 为准（[M1/P5 spec](M1-frontend-trim/specs/P5-brand.md) 7.2） | 已完成 | M1/P5 |
