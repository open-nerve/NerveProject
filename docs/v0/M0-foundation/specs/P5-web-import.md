# M0/P5 前端迁入与内嵌：设计说明（spec）

| 项 | 内容 |
|---|---|
| Phase | M0/P5 `web-import` |
| 日期 | 2026-09-22 |
| 状态 | 已批准 |
| 上级文档 | [M0 设计文档](../M0-design.md) 第 1、2、3.3、3.4、3.6、5.1、5.2、6.1–6.3、8、10、12 节；[v0 总体设计](../../v0-design.md) 2.2、2.3、6.8、7.1、7.6 节；[前端改动清单](../../frontend-changes.md) |
| 前置交接 | [P2-server-platform-p5-webui-mount](../handoffs/P2-server-platform-p5-webui-mount.md)、[P3-api-contract-p5-notes](../handoffs/P3-api-contract-p5-notes.md)、[P4-plane-schema-p5-notes](../handoffs/P4-plane-schema-p5-notes.md)（都在本 Phase 处理，见 2.14） |

## 1. 目标
把 Plane 的前端原样迁入 `web/`，能安装、检查、构建，并编进 `nerve`：
- 从 Plane 提交 `02c19e1` 复制 web 应用和它用到的 12 个包，不做功能改动；
- 根目录的 pnpm 工作区、turbo 配置和锁文件：与已有的 `@redocly/cli`、`@nerve/api-client` 合并；
- lint 警告基线（只降不升），类型检查、lint、格式检查统一由 turbo 驱动；
- Vite 开发服务器把 `/api` 转发给本地 Go 服务，`make web-dev`；
- `platform/webui`：内嵌 `dist/`，按缓存策略提供静态文件，单页应用回退到 `index.html`；
- `make build` 构建出内嵌前端的 `bin/nerve`；
- 持续集成的 `web` 任务加上 lint 基线、格式检查和前端构建；
- 在前端改动清单中登记来源提交和迁入时的改动。

## 2. 交付物

### 2.1 文件总览
路径都相对于仓库根目录。"复制"表示从 Plane 提交 `02c19e1341d93141e8ad7b3278298adce208bafc` 原样复制，"生成"表示由命令生成并提交。

| 路径 | 内容 |
|---|---|
| `web/apps/web/`、`web/packages/{constants,editor,hooks,i18n,propel,services,shared-state,tailwind-config,types,typescript-config,ui,utils}/` | 复制（2.3）；之后只改动 2.7、2.8 列出的几行 |
| `patches/react-color@2.19.3.patch` | 复制；`pnpm-workspace.yaml` 的 `patchedDependencies` 引用它 |
| `package.json` | 开发依赖加入 `oxfmt`、`oxlint`、`turbo`（2.4） |
| `pnpm-workspace.yaml` | 合并 Plane 的 catalog、overrides 等，只保留作用于迁入的包的条目（2.4） |
| `turbo.json` | 来自 Plane，删掉不适用的条目（2.4） |
| `pnpm-lock.yaml` | 生成（2.5） |
| `.oxlintrc.json`、`.oxfmtrc.json` | 来自 Plane，排除生成的文件；修正 Tailwind 样式表的路径（2.7） |
| `web/packages/api-client/package.json` | `typescript` 改用 `catalog:`；新增 `check:lint`、`check:format`（2.7） |
| `.gitignore` | 前端的生成物和构建产物；`webui/dist/` 中除 `.gitkeep` 外的文件 |
| `server/internal/platform/webui/` | `embed.go`、`handler.go` 及测试；`dist/.gitkeep`（2.9） |
| `server/internal/bootstrap/app.go`、`commands.go`、`app_test.go`、`webui_test.go` | 挂载 `webui`（2.10） |
| `Makefile` | `web-dev`、`build`、`build-web`；`lint-web` 改由 turbo 驱动（2.11） |
| `.github/workflows/ci.yml` | `web` 任务：pnpm 缓存、前端构建（2.12） |
| `docs/v0/frontend-changes.md`、`README.md` | 来源提交与迁入时的改动；前端的开发和构建（2.13） |
| `docs/v0/M0-foundation/handoffs/` 中交给 P5 的三个文件 | 改为 `done`，写明处理结果（2.14） |

新增的包依赖（箭头表示"导入"，均为非测试代码）：
```
bootstrap ──→ platform/webui（另有原来的 platform 包和 modules/instance）
platform/webui ──→ 只有标准库（embed、io/fs、net/http、path、strings）
```

### 2.2 原型验证：结论与证据
在仓库外的临时目录里，从 `main`（`83c8b49`，P4 已合并）出发，按计划的 6 个 Task 完整执行了一遍，再在一个干净的克隆上跑了持续集成两个任务的全部步骤。本 spec 和计划中的数值都来自这些运行。机器：Apple M 系列 18 核、macOS，Node 24.15.0、pnpm 11.10.0、Go 1.27.1。

| 问题 | 结论 | 证据 |
|---|---|---|
| web 的构建需要哪些包 | `apps/web` 加 12 个包 | web 的 `workspace:*` 依赖及其传递闭包正好是 constants、editor、hooks、i18n、propel、services、shared-state、types、ui、utils、tailwind-config、typescript-config；这些目录的源码中没有 `@plane/logger`、`@plane/decorators`、`@plane/codemods`；`turbo run build --filter=web` 共 11 个任务（两个 config 包没有 build 脚本） |
| 复制是否原样 | 是 | 13 个目录和 `patches/` 的 git 树对象与 `02c19e1` 中对应的树完全相同；共 4060 个文件（含 i18n 的符号链接 `locales → src/locales`），没有文件被 `.gitignore` 挡掉 |
| Node 24 | 可用 | 迁入的包都没有 `engines`；根目录的 `^24` 不变；安装、检查、构建全部通过 |
| Plane 的 `.npmrc` | 不迁入 | pnpm 11 只从 `.npmrc` 读取认证、仓库地址和网络设置（pnpm 源码 `isNpmrcReadableKey`）。实测复制它之后 `pnpm config list` 里没有它的任何设置，`node_modules/.modules.yaml` 中 `publicHoistPattern` 是 `[]`。不迁入时，13 个包的解析结果与 Plane 的锁文件逐行相同 |
| 锁文件能否复现 | 能 | Plane 的锁文件把 importers 的路径移到 `web/` 下，再执行一次 `pnpm install`：三次独立生成，结果都是 13835 行，SHA-256 `d677458d…18ca`；第二次 `pnpm install` 不再改动；`--frozen-lockfile` 通过 |
| 锁文件改变了什么 | 只加入 Nerve 自己的依赖 | 13 个 importers 与 Plane 逐行相同，Plane 的包没有任何版本变化；新增 21 个包（Redocly、openapi-typescript、openapi-fetch 及其依赖）。与 P3 的锁文件相比，Redocly 这一支只有 brace-expansion 被 Plane 的覆盖项从 2.1.7 换成 5.0.9，`make gen-check-web` 没有差异 |
| catalog 等能否修剪 | 能 | 删掉不作用于迁入的包的条目（2.4）之后，13 个包的解析结果仍与 Plane 相同；`pnpm-workspace.yaml` 与 Plane 的原文相比只有删除，没有新增的行 |
| 构建产物在哪里 | `web/apps/web/build/client/` | React Router 8 的 SPA 模式（`ssr: false`）；输出 1239 个文件、34 MB：`index.html`（6228 字节，构建时预渲染）、`assets/`（1226 个带哈希的文件）、从 `public/` 复制的 `favicon/`、`icons/`、`manifest.json` 等。另有 `build/server/`（预渲染用，不内嵌） |
| 构建需要环境变量吗 | 不需要 | `@plane/constants` 中 `API_BASE_URL = process.env.VITE_API_BASE_URL \|\| ""`，为空时所有请求都是同源的相对路径；turbo 默认的严格环境模式只把 `turbo.json` 列出的变量传给任务 |
| 构建时间 | 本机 18–20 秒 | 没有 turbo 缓存时 `make build` 20 秒（前端 18 秒，11 个任务）；`make lint-web` 之后再 `make build-web` 7 秒（10 个任务命中缓存）。`bin/nerve` 50.5 MB（不含前端时 18.0 MB） |
| 用户看到什么 | 前端能打开，显示 Plane 的错误页 | `bin/nerve serve` 后用浏览器打开首页：128 个静态资源全部 200；前端调用 Plane 的 `GET /api/instances/`，Nerve 返回 404，页面显示 Plane 的"🚧 Looks like Plane didn't start up correctly!"。深层路径同样：330 个静态资源全部 200。控制台另有一条 React #418（水合不匹配），`index.html` 与 Plane 自己的构建产物逐字节相同，是上游的行为 |
| lint 警告数 | 远低于 Plane 的上限 | 见 2.7 的表。Plane 的上限（web 11957、propel 3605、editor 416）是旧数据 |
| oxlint 从包目录运行时用哪份配置 | 根目录的 `.oxlintrc.json` | 在 web 中逐条规则统计，从包目录直接运行与显式 `-c ../../../.oxlintrc.json` 的结果完全相同（779 条，含 jsx-a11y 等插件的规则） |
| oxfmt | 需要改样式表路径 | `sortTailwindcss.stylesheet` 仍是 Plane 的 `packages/tailwind-config/index.css` 时，`oxfmt --check .` 在 web 中崩溃（SIGSEGV）；改为 `web/packages/tailwind-config/index.css` 后全部通过 |
| `platform/webui` | 按 2.9 实现，测试通过 | 9 个单元测试（`fstest.MapFS`）和 2 个 `bootstrap` 测试不需要构建前端；golangci-lint `0 issues.`；故意让 `webui` 导入 `httpserver`，架构测试报"platform packages do not import each other, except config" |
| `bin/nerve serve` | 符合 M0 设计 3.3、3.4 | curl 的结果见 2.9 的表；`/api/v0/instance` 200，`/api/v0/nope` 404 problem+json，`/healthz` 200 |
| 开发代理 | 可用 | `make run` 和 `make web-dev` 同时运行：`http://127.0.0.1:3000/api/v0/instance` 返回 Go 服务的 JSON，`/api/v0/nope` 返回 problem+json；不加代理时这两个路径都返回 `index.html` |
| 持续集成的步骤 | 在干净的克隆上通过 | `web`：`pnpm install --frozen-lockfile`、`make gen-check-web`、`make lint-web`（46 个任务）、`make build-web`，共 53 秒（pnpm 存储已有缓存）；`server`：`make gen-check-go`、`make lint-go`、`make test` 全部通过。执行后工作区没有任何改动 |
| 冷缓存的耗时 | 本机约 1 分钟 | 空的 pnpm 存储：安装 19 秒（下载 823 MB）；没有 turbo 缓存：`make lint-web` 32–35 秒，之后 `make build-web` 7 秒。GitHub 的 `ubuntu-24.04`（4 核）预计 3–5 分钟，推送后核实（2.12） |
| 内存 | 在持续集成的余量内 | web 的 `tsc --noEmit` 峰值 1.4 GB，`react-router build` 峰值 2.4 GB；仓库是公开的，`ubuntu-24.04` 有 16 GB 内存 |

### 2.3 迁入的内容
- **来源**：Plane `preview` 分支的提交 `02c19e1341d93141e8ad7b3278298adce208bafc`（P4 交接）。不用 `v1.4.2` 标签（`5f7d927`）：M0 设计第 1 节锁定的 React Router 8.3.0、pnpm 11.10.0 只在 `02c19e1` 中。
- **复制方式**：`git -C "$PLANE" archive <完整 SHA> -- <目录…> | tar -x -C web`，`$PLANE` 是本地参考源码 `plane/`（在工作区中用 `git rev-parse --git-common-dir` 找到主检出目录）。复制之前先确认提交存在；提交之前逐个比较 git 树对象，13 个目录和 `patches/` 都必须与 Plane 相同。
- **迁入**：

  | Plane | Nerve | 说明 |
  |---|---|---|
  | `apps/web` | `web/apps/web` | 2336 个文件 |
  | `packages/` 下的 constants、editor、hooks、i18n、propel、services、shared-state、tailwind-config、types、typescript-config、ui、utils | `web/packages/<包名>` | 共 1724 个文件；services 是"暂时使用"（前端改动清单） |
  | `patches/react-color@2.19.3.patch` | `patches/` | web 依赖 react-color，锁文件记录了这个补丁的哈希 |
  | `turbo.json`、`pnpm-workspace.yaml` 中的 catalog 等 | 根目录 | 合并，见 2.4 |
  | `.oxlintrc.json`、`.oxfmtrc.json` | 根目录 | 见 2.7 |

- **不迁入**：`apps/admin`、`apps/space`、`apps/live`、`apps/api`、`apps/proxy`、`packages/logger`、`packages/decorators`、`packages/codemods`（web 及其依赖都不引用）；根目录的 `.npmrc`（2.2：pnpm 11 不读取其中的设置）；husky、lint-staged（Git 钩子）、react-doctor；Plane 的持续集成、Docker 文件、部署脚本和文档。
- **迁入但留给 M1 删除**：web 中的 `Dockerfile.web`、`Dockerfile.dev`、`caddy/`、`.dockerignore`；`serve` 依赖以及 `start`、`preview` 脚本；`public/` 中从未注册过的 `sw.js` 和 workbox 文件。P5 只做让前端能安装、检查、构建和开发的最小改动（M0 设计 5.1），这些都属于功能删减（第 7 节）。
- **版权**：复制的文件原样保留 Plane 的版权声明；Plane 的 `LICENSE.txt` 与 Nerve 的 `LICENSE` 逐字节相同，不另外复制。

### 2.4 根目录的工作区：`package.json`、`pnpm-workspace.yaml`、`turbo.json`
**`package.json`**：保留 P1、P3 的内容（`engines.node: ^24`、`packageManager: pnpm@11.10.0`、`@redocly/cli` 2.53.3），开发依赖加入 `oxfmt`、`oxlint`、`turbo`，版本都写 `catalog:`（0.35.0、1.51.0、2.10.11）。不加 Plane 根目录的脚本：命令的入口是 Makefile。不加 husky、lint-staged（Plane 用它们装 Git 钩子，Nerve 不用）和 react-doctor。

**`pnpm-workspace.yaml`**：
- `packages` 不变：`web/apps/*`、`web/packages/*`、`e2e`。
- catalog、overrides、allowBuilds、minimumReleaseAgeExclude、patchedDependencies、peerDependencyRules 来自 Plane，**只保留作用于迁入的包的条目**：
  - catalog：183 → 144 条。删掉的 39 条都没有被迁入的 `package.json` 引用：apps/live 的 26 条（其中 2 条 logger、decorators 也用）、apps/space 的 1 条（`@react-router/serve`）、codemods 的 4 条、只有 logger 或 decorators 使用的 3 条、没有任何包引用的 3 条，以及根目录不再使用的 husky、lint-staged。
  - overrides：61 → 47 条。删掉 `express: "catalog:"`（引用的 catalog 条目已删除）；目标包不在依赖图中的 11 条（`@types/express`、`rollup`、`serialize-javascript`、`undici@7`、`yaml@1`、`tmp`、`ws@7`、三条 `@opentelemetry/*`、`toml`）；以及两条 `postcss-selector-parser`——包本身在依赖图中（6.0.10、6.1.3、7.1.3），但这两条的版本区间选择器（`>=6.1.0 <6.1.3`、`>=7.1.0 <7.1.3`）不落在图中任何一个版本上，等于不作用于任何包。
  - allowBuilds：删掉依赖图中不存在的 `@parcel/watcher`、`msgpackr-extract`、`sharp`。
  - 其余三节原样保留（其中的包都在依赖图中）。
  - 判断"作用于迁入的包"的依据是锁文件：修剪前后 13 个包的解析结果不变（2.2）。
- 保留的条目连同 Plane 的英文注释原样保留，文件开头用中文说明来源。与 Plane 的原文相比只有删除（61 行），计划中用 `diff` 核对。

**`turbo.json`**：来自 Plane，只做删除：
- `globalDependencies` 去掉 `.npmrc`（没有迁入）；
- `globalEnv` 去掉没有代码读取的 13 个变量：`APP_VERSION`、`LOG_LEVEL`（只有 live、logger 读取）、`VITE_APP_VERSION` 和 10 个 Sentry 变量（没有任何代码读取）。保留的 14 个都有迁入的代码读取，包括 admin、space、live 的地址（web 中的链接还在用，M1 删除）。
- `remoteCache.enabled: false` 和全部 `tasks` 不变。

### 2.5 锁文件
- **做法**：以 Plane 的 `pnpm-lock.yaml` 为起点：
  1. `git -C "$PLANE" show <SHA>:pnpm-lock.yaml`，用 `sed` 把 importers 的键 `apps/<名字>`、`packages/<名字>` 改为 `web/apps/<名字>`、`web/packages/<名字>`（包之间的 `link:../../packages/…` 相对路径不变）；
  2. 执行 `pnpm install`：pnpm 删除没有迁入的 importers（admin、space、live 等）和只被它们使用的包，保留迁入的包已锁定的版本，只为 Nerve 自己的依赖解析新版本。
- **为什么不从头生成**：从头生成会按 catalog 中的 `^` 范围取当时的最新版本，前端依赖就不再是 Plane 测试过的那一组，"原样迁入"就不成立了。
- **核对**：计划中用一条命令比较 13 个 importers 与 Plane 锁文件中对应的部分，要求逐行相同。锁文件的 SHA-256 是写作时的值；Nerve 自己的依赖（Redocly 等）的间接依赖取决于安装时 npm 上的版本，可能不同，以提交的锁文件为准。
- **Redocly 受到的影响**：Plane 的覆盖项作用于整个依赖图，Redocly 间接依赖的 brace-expansion 从 2.1.7 变为 5.0.9，其余版本与 P3 的锁文件相同；`make gen-check-web` 没有差异。

### 2.6 构建产物与运行时配置
- **构建**：`turbo run build --filter=web`，先构建 web 依赖的 10 个包（tsdown 输出到各自的 `dist/`，i18n 另外生成 `src/types/keys.generated.ts`），再执行 web 的 `react-router build`。产物在 `web/apps/web/build/client/`，就是 M0 设计 3.4 要复制的目录，Plane 的构建配置不需要改动。
- **环境变量**：不设置任何变量。前端与后端同源，所有接口请求都是相对路径（`/api/...`）。总体设计 6.8 提到的 Vite 模式文件（`.env.development` 等）不需要建立。
- **`.gitignore`**：新增 `web/apps/web/.react-router/`、`web/apps/web/build/`、`web/packages/*/dist/`、`web/packages/i18n/src/types/keys.generated.ts`，以及 `server/internal/platform/webui/dist/*`（`!…/.gitkeep`）。计划中在执行过所有 make 命令之后确认 `git status` 为空。
- **M0 中打开页面看到的内容**：前端仍在调用 Plane 的接口。启动时的 `GET /api/instances/` 得到 Nerve 的 404 problem+json，页面显示 Plane 的"didn't start up correctly"。这是预期的：前端从 M2 起对接 Nerve 的接口（总体设计 7.2）。M0 的验收只要求页面和静态资源能加载（没有 404）、深层路径返回 `index.html`（M0 设计第 9 节 S2）。

### 2.7 lint 警告基线、类型检查、格式检查
**基线存放在哪里**：沿用 Plane 的做法，每个包的 `check:lint` 脚本是 `oxlint --max-warnings=<上限> .`，超过上限 oxlint 退出码非 0（`Exceeded maximum number of warnings. Found N.`）。不另写脚本。

**上限定为实测的警告数**（M0 设计 5.2 的"只降不升"允许调低）：

| 包 | Plane 的上限 | 实测 | P5 的上限 |
|---|---|---|---|
| web | 11957 | 779 | 779 |
| editor | 416 | 75 | 75 |
| propel | 3605 | 59 | 59 |
| utils | 38 | 34 | 34 |
| ui | 66 | 32 | 32 |
| services | 6 | 6 | 6（不变） |
| hooks | 4 | 4 | 4（不变） |
| i18n | 9 | 3 | 3 |
| constants | 2 | 2 | 2（不变） |
| types | 1 | 1 | 1（不变） |
| shared-state | 0 | 0 | 0（不变） |
| api-client | — | 0 | 0（新增脚本） |

合计 995 条（原来误写为 1005，M1 设计时更正）。沿用 Plane 的上限等于允许 web 的警告再增加十几倍而不报错，"不升"就失去了意义。实测在根目录的 `.oxlintrc.json` 下进行（oxlint 从包目录运行时使用它，见 2.2），`.gitignore` 挡掉的构建产物不计入。

**规则**：警告数超过上限，`make lint-web` 失败；警告数下降后，在同一个提交里把上限调低到新的数值（README 写明）。"调低"靠评审把关，不做自动检查；M1 删减之后重新测出基线，并在 M1 的设计文档中决定是否加自动检查（第 7 节）。

**配置文件**：
- `.oxlintrc.json`：Plane 的原文，`ignorePatterns` 加入 `web/packages/api-client/src/schema.gen.ts`（P3 交接第 1 条）。oxlint 只检查 JS/TS 文件，不会碰 `api/dist/`。
- `.oxfmtrc.json`：`sortTailwindcss.stylesheet` 改为 `web/packages/tailwind-config/index.css`（不改时 oxfmt 在 web 中崩溃）；删掉只作用于 `packages/codemods` 的覆盖项；加入 `ignorePatterns`：`api/dist/**`、`web/packages/api-client/src/schema.gen.ts`（P3 交接第 1 条），以及 i18n 构建时生成的 `web/packages/i18n/src/types/keys.generated.ts`（oxfmt 只读取当前目录下的 `.gitignore`，不排除它时，构建或类型检查之后 i18n 的格式检查失败）。

**`make lint-web`** 改为 `turbo run check:types check:lint check:format --output-logs=errors-only`（P3 交接第 2 条）：
- `check:types` 依赖上游包的 `build`（`turbo.json`），turbo 按依赖顺序先构建各包；共 46 个任务。
- `--dry=json` 列出的 12 个 `check:types` 任务中有 `@nerve/api-client#check:types`。
- `--output-logs=errors-only`：只打印失败任务的输出。oxlint 会打印全部 995 条警告的详情，在持续集成的日志里没有用处。
- api-client 新增 `check:lint`（上限 0）和 `check:format`，与 Plane 的包一致；`package.json` 的键顺序按 oxfmt 调整（`license` 放在 `description` 之后）。
- **格式检查也是门禁**：Plane 的持续集成检查格式（`check:format`），迁入的代码全部符合；不纳入门禁的话，这些脚本和配置就是没人用的代码（第 3 节第 4 项）。

### 2.8 开发服务器代理与 `make web-dev`
- `web/apps/web/vite.config.ts` 的 `server` 中加上 `proxy: { "/api": "http://127.0.0.1:8080" }`，旁边注释说明是 Nerve 的改动。这是 Plane 文件中唯一的一处代码改动，登记在前端改动清单中。
- 只转发 `/api`：Nerve 的接口都在 `/api/v0/` 下；Plane 前端调用的 `/auth/...` 等路径由 M2 改为新接口（第 7 节）。
- `make web-dev` = `turbo run dev --filter=web`：先构建 web 依赖的包（`dev` 依赖上游的 `build`），再启动 `react-router dev --port 3000`（`127.0.0.1`）。只运行 web 自己的开发服务器，改了 `web/packages/*` 的代码要重新执行；Plane 用 `turbo run dev` 同时监视所有包，需要至少 11 个常驻任务的并发，M1 集中修改这些包时再评估。
- 开发流程（M0 设计 6.2）：`make dev-db` → `make run` → 另一个终端 `make web-dev` → 打开 http://127.0.0.1:3000 。

### 2.9 `platform/webui`
```go
package webui

// FS returns the embedded frontend, with index.html at its root once built.
func FS() fs.FS

// Handler serves the single-page app in files (GET and HEAD only).
func Handler(files fs.FS) http.Handler
```

- **内嵌**：`embed.go` 中 `//go:embed all:dist`。`all:` 让只有 `.gitkeep` 的 `dist/` 也能匹配（不带 `all:` 时隐藏文件被排除，模式匹配不到任何文件，编译失败），写法与 `server/migrations` 相同。`FS()` 用 `fs.Sub` 去掉 `dist/` 前缀。
- **handler 只依赖 `fs.FS`**：生产中传入 `webui.FS()`，测试中传入 `fstest.MapFS`，不需要先构建前端。
- **规则**（依次判断）：

  | 请求 | 响应 | 缓存 |
  |---|---|---|
  | 方法不是 GET、HEAD | 405，`Allow: GET, HEAD`（P2 交接第 3 条） | — |
  | `files` 中没有 `index.html`（没有构建前端） | 404，`text/plain`，提示运行 `make build` 或使用 `make web-dev` | `no-cache` |
  | 路径对应一个普通文件（不是目录），并且路径中没有以 `.` 开头的部分 | 这个文件 | `assets/` 下：`public, max-age=31536000, immutable`；其他：`no-cache` |
  | `assets` 或 `assets/` 下不存在的路径 | 404 | — |
  | 其余路径（包括 `/`、深层路径、目录、隐藏文件） | `index.html` | `no-cache` |

- **为什么缺失的 `assets/` 返回 404**：`assets/` 中只有带哈希的文件，缺失通常是旧版本页面请求已不存在的代码块。返回 `index.html` 会让脚本加载器拿到 HTML（MIME 类型错误），返回 404 则是明确的"文件不存在"。其他缺失路径都当作页面路径，交给前端路由。
- **隐藏文件**：`dist/.gitkeep`、误复制进来的 `.DS_Store` 都不提供，当作页面路径处理。
- **缓存**：Vite 只给 `assets/` 下的文件名加哈希；`index.html` 和从 `public/` 复制的文件名字不变，所以都用 `no-cache`（每次重新验证）。
- **Content-Type 和 HEAD**：由 `http.ServeFileFS` 处理：先按扩展名（`mime.TypeByExtension`），查不到时按内容嗅探（woff2、woff、ttf、ico 都能嗅探出正确的类型）；它同时处理 HEAD、Range，并把 `/index.html` 301 跳转到 `./`（标准库 `FileServer` 的行为）。嵌入的文件没有修改时间，所以不发 `Last-Modified`。
- **实测**（`make build` 之后的 `bin/nerve serve`）：

  | 请求 | 结果 |
  |---|---|
  | `GET /` | 200，`text/html; charset=utf-8`，`no-cache`，6228 字节 |
  | `GET /acme/projects/<uuid>/issues` | 与 `/` 相同 |
  | `GET /assets/entry.client-<hash>.js` | 200，`text/javascript; charset=utf-8`，`public, max-age=31536000, immutable` |
  | `GET /assets/globals-<hash>.css` | 200，`text/css; charset=utf-8`，immutable |
  | `GET /assets/inter-latin-wght-normal-<hash>.woff2` | 200，`font/woff2`，immutable |
  | `GET /manifest.json` | 200，`application/json`，`no-cache` |
  | `GET /assets/entry.client-gone.js` | 404，`404 page not found` |
  | `GET /index.html` | 301，`Location: ./` |
  | `GET /.gitkeep` | 200，`index.html` |
  | `POST /` | 405，`Allow: GET, HEAD` |
  | `HEAD /` | 200，`Content-Length: 6228`，没有响应体 |
  | `GET /api/v0/instance` | 200，JSON |
  | `GET /api/v0/nope` | 404，`application/problem+json` |
  | 没有构建前端时 `GET /` | 404，提示文字，`no-cache` |

- **测试**（`handler_test.go`、`embed_test.go`，都用 `fstest.MapFS`）：首页；深层路径、带结尾 `/` 的路径、目录、隐藏文件、不存在的非 `assets` 文件、带查询参数的路径都回退到 `index.html`；`assets/` 下的 JS、CSS 与 `public/` 下的 JSON、PNG 的类型和缓存；缺失的 `assets/` 路径（包括 `/assets`、`/assets/` 和 `assets/` 下的隐藏文件）返回 404 且不含 `index.html`；`/index.html` 的 301；HEAD；405（已构建和没有构建两种情况）；没有构建时的提示；`FS()` 的根目录就是 `dist/`。
- **架构规则**：`webui` 是 platform 包，只导入标准库。规则 7（platform 包之间互不导入）不需要修改；计划中故意让它导入 `httpserver`，确认架构测试报错。

### 2.10 挂载与接线（`bootstrap`）
- `newApp(ctx, cfg, logger, migrationFiles, webFiles fs.FS)`：在挂上各模块之后执行 `mux.Handle("/", webui.Handler(webFiles))`，注册的是不带方法的 `/`（P2 交接第 1 条）。`/api/`（平台的 404 兜底）、`/healthz`、`/readyz` 和模块的路由都比 `/` 更具体，不受影响。
- `Serve` 传入 `webui.FS()`；`bootstrap` 的测试传入一个只有 `index.html` 的 `testWebUI`（`startApp` 内部使用，调用处不变）。
- **测试**：`webui_test.go`：`/` 和深层路径返回 `testWebUI` 的 `index.html`；`/healthz` 仍返回 `application/json`（`startApp` 靠轮询 `/healthz` 判断服务已启动，被前端吞掉时它也会得到 200，所以单独断言类型）。已有的 `TestUnknownAPIRequestsStillAnswerProblem404` 在挂上前端之后继续通过。
- 没有加 `web.enabled` 开关：见第 3 节第 5 项。

### 2.11 Makefile
| 命令 | 作用 |
|---|---|
| `make web-dev` | `turbo run dev --filter=web`：前端开发服务器 http://127.0.0.1:3000，`/api` 转发给 `make run` 的后端 |
| `make lint-web` | `turbo run check:types check:lint check:format`（原来是 `pnpm -r run check:types`） |
| `make build-web` | `turbo run build --filter=web`：只需要 Node；持续集成的 `web` 任务调用 |
| `make build` | 先 `build-web`；清空 `server/internal/platform/webui/dist/`（保留 `.gitkeep`）；复制 `web/apps/web/build/client/.`；`cd server && go build -o ../bin/nerve ./cmd/nerve` |

- 新变量：`TURBO := TURBO_TELEMETRY_DISABLED=1 pnpm exec turbo`（关闭 turbo 的匿名使用数据上报，与 Redocly 的处理一致）、`TURBO_QUIET := --output-logs=errors-only`、`WEBUI_DIST`。
- `make build` 不注入版本号：`bin/nerve` 的版本是 `platform/buildinfo` 中的默认值 `0.1.0-dev`，提交号来自 Go 工具链嵌入的 VCS 信息，与 `make run` 相同。需要注入时用 buildinfo 注释中写明的 `-ldflags -X`（第 7 节，P6）。
- 清空 `dist/` 用 `find … -mindepth 1 ! -name .gitkeep -delete`，旧构建留下的文件不会被嵌入。
- 命令总数从 17 个变为 20 个；以 Tab 开头的行从 25 行变为 30 行；兼容 GNU Make 3.81。

### 2.12 持续集成
**`web` 任务**：checkout → setup-node → `corepack enable` → 缓存 pnpm 存储 → `pnpm install --frozen-lockfile` → `make gen-check-web` → `make lint-web` → `make build-web`。
- **pnpm 存储的缓存**：用 `pnpm store path --silent` 取得路径，`actions/cache@v6` 按 `hashFiles('pnpm-lock.yaml')` 缓存。只用精确的键：锁文件变了就换一个新的缓存，旧版本的包不会在缓存里越积越多。不用 `setup-node` 的 `cache: pnpm`：它要求 pnpm 在 `setup-node` 之前就已安装，而 pnpm 由 corepack 在 `setup-node` 之后启用。
- **turbo 的缓存**：只在任务内部使用（`lint-web` 构建的包被 `build-web` 复用），不跨运行保存。跨运行保存需要按提交存一份、用前缀恢复，恢复的缓存会不断累积旧的条目；M1 集中修改前端时，这会让每次恢复越来越慢。先看实际耗时，不够再考虑只在 `main` 上保存（第 6 节）。
- **不在 `web` 任务中运行 `make build`**：它需要 Go。嵌入的逻辑由 `server` 任务中的 `webui`、`bootstrap` 测试覆盖（`dist/` 中只有 `.gitkeep`，测试用 `fstest`）；完整的 `make build` 由 P6 的 `e2e` 任务执行（M0 设计 6.3），这样不在两个任务里各编译一次。
- **`server` 任务不变**：不需要前端的构建产物。
- 两个任务的 `if` 条件保持不变（M0/P3 的同仓 PR 跳过）；`timeout-minutes: 15` 不变。
- 推送后由控制者记录 `web` 任务第一次运行（冷缓存）和第二次运行（命中 pnpm 缓存）的耗时，写进 P5 的 review。

### 2.13 文档：前端改动清单、README
- **前端改动清单**"一、代码来源"：加上完整的来源提交 SHA（P4 交接第 3 条）和迁入方式；"不使用"一行补上 `.npmrc`、husky、lint-staged、react-doctor；新增"1.1 迁入时的改动（M0/P5）"表，逐项登记 2.4、2.5、2.7、2.8 的改动和原因。"四、品牌"中"保留原有的版权声明"改为已完成（M0/P5）。
- **README**："开发环境"说明 Node 用于前端的开发和构建；`make lint` 一行写明 `lint-web` 的内容；新增"前端"一节：开发（`make web-dev`）、构建（`make build`）、M0 中看到的页面、警告基线的规则。

### 2.14 处理交给 P5 的交接
- [P2-server-platform-p5-webui-mount](../handoffs/P2-server-platform-p5-webui-mount.md)：不带方法的 `/`（2.10）；只处理 GET 和 HEAD（2.9）；`web.enabled` 没有加入（第 3 节第 5 项）。
- [P3-api-contract-p5-notes](../handoffs/P3-api-contract-p5-notes.md)：
  1. oxlint、oxfmt 排除 `schema.gen.ts`，oxfmt 另外排除 `api/dist/**`（2.7）；
  2. `make lint-web` 由 turbo 驱动，api-client 仍在类型检查的范围内（2.7）；
  3. 保留 `@redocly/cli`（2.4），`make gen-check-web` 没有差异（2.5）；
  4. api-client 的 `typescript` 改为 `catalog:`；
  5. 包名两种前缀并存，照旧；
  6. 可选的 tsconfig 继承不做：api-client 已经是 `strict` 加 `noUncheckedIndexedAccess`，继承 `@plane/typescript-config` 会让新包依赖一个 M1 要改名的 Plane 包。
- [P4-plane-schema-p5-notes](../handoffs/P4-plane-schema-p5-notes.md)：从 `02c19e1` 迁入；复制命令用完整 SHA 并先确认提交存在；树对象核对（2.3）；前端改动清单记下完整 SHA（2.13）。

三个 handoff 在计划的最后一个 Task 中改为 `done`，并写明处理结果。

## 3. 与上级设计的差异和补充（请控制者裁定）

| # | 上级设计 | P5 的做法 | 理由 |
|---|---|---|---|
| 1 | M0 2、5.1：`.npmrc` 复制到仓库根目录 | 不复制 | pnpm 11 只从 `.npmrc` 读取认证和仓库地址，Plane 写在里面的设置都不起作用（2.2）；复制过来就是不起作用的配置 |
| 2 | M0 5.1：`turbo.json` 和 catalog 只删掉明显只属于 admin、space、live 的条目 | catalog、overrides、allowBuilds 只保留作用于迁入的包的条目；`turbo.json` 删掉 `.npmrc` 和没有代码读取的 13 个环境变量；根目录不加 husky、lint-staged、react-doctor | 删掉的条目都不作用于迁入的包，锁文件证明 13 个包的解析结果不变（2.4）。现在删有锁文件可以核对；留到 M1，就要在删减功能的同时再核对一遍 |
| 3 | M0 5.2：沿用 Plane 的警告上限（web 11957、propel 3605、editor 416） | 上限改为实测的警告数（web 779、propel 59、editor 75 等，2.7） | Plane 的上限远高于实际，沿用就等于允许警告成倍增加；调低本身符合"只降不升" |
| 4 | M0 6.1、6.3：`lint-web` 是类型检查加 oxlint | 另外检查格式（`check:format`） | Plane 的持续集成也检查格式，迁入的代码全部符合；P3 交接已经假定会加入 oxfmt |
| 5 | M0 3.4、P2 交接第 4 条：P5 加入配置项 `web.enabled` | 不加入 | v0 的部署方式只有"一个 `nerve` 加一个 Postgres"，没有需要关闭内嵌前端的场景，开关没有使用者；与 P2 对 `app.name` 的处理一致（P2 spec 第 3 节第 1 项） |
| 6 | M0 3.4：没有构建前端时提示"前端未构建，请运行 make build" | 英文提示，另外提到开发时用 `make web-dev`；状态码 404，`no-cache` | 程序输出的文字（日志、错误、problem）都用英文；404 表示"没有可提供的页面"，不是服务故障 |
| 7 | M0 3.4：能对应到静态文件就返回文件，否则返回 `index.html` | 另外：`assets/` 下缺失的路径返回 404；隐藏文件不提供；`/index.html` 301 到 `./` | 缺失的带哈希文件不应变成 HTML（2.9）；`.gitkeep` 不是前端文件；301 是 `http.ServeFileFS` 的行为 |
| 8 | M0 3.4：带哈希的资源 `immutable`，`index.html` `no-cache` | 另外：`public/` 复制来的文件（名字不带哈希）也是 `no-cache` | 名字不变的文件不能长期缓存 |
| 9 | M0 6.1：`make build` 构建前端并嵌入 | 拆出 `make build-web`（只构建前端，只需要 Node），`make build` 依赖它 | 与 P3 的 `-go`/`-web` 拆分一致；持续集成的 `web` 任务没有 Go |
| 10 | M0 6.3：`web` 任务是 install → `gen-check-web` → `lint-web`（P5 加入 oxlint 基线） | 另外缓存 pnpm 存储，最后执行 `make build-web`；完整的 `make build` 留给 P6 的 `e2e` 任务 | M0 第 10 节的门禁包括前端构建；`make build` 需要 Go，`e2e` 任务反正要执行它 |
| 11 | M0 12：持续集成使用 pnpm 缓存和 turbo 的本地缓存 | pnpm 缓存按锁文件；turbo 缓存只在任务内部使用，不跨运行保存 | 跨运行保存的 turbo 缓存会不断累积（2.12）；先看实际耗时 |
| 12 | M0 2：仓库布局 | 根目录另有 `patches/`；没有 `.npmrc` | `patches/` 是 react-color 的补丁，锁文件引用它 |
| 13 | 总体设计 6.8：前端使用 Vite 的模式文件（`.env.development` 等） | 不建立 | 同源部署，接口地址为空即相对路径，没有需要配置的值（2.6） |
| 14 | M0 5.1：最小改动包括"构建产物的位置" | Plane 的构建配置不改，`make build` 从 `web/apps/web/build/client/` 复制 | 产物本来就在 M0 设计 3.4 要求的位置 |
| 15 | M0 8 P5 验收："`bin/nerve serve` 能打开前端首页" | 首页和全部静态资源都能加载；页面显示 Plane 的"didn't start up correctly" | 前端调用的 Plane 接口（`/api/instances/`）在 Nerve 中不存在，M2 起对接新接口（2.6） |
| 16 | — | 锁文件以 Plane 的锁文件为起点生成（2.5） | 保证迁入的包的依赖版本与 Plane 相同 |

控制者裁定后，评审阶段把 M0 设计第 2 节（布局）、3.4 节（`web.enabled`、提示文字、回退规则、缓存）、3.6 节（`web.enabled`）、5.1 节（`.npmrc`、修剪规则）、5.2 节（新的基线数值）、6.1 节（`build-web`、`lint-web` 的内容）、6.3 节（`web` 任务）、第 12 节（持续集成的缓存），以及总体设计 6.8 节（前端的环境变量）同步更新。

## 4. 验收标准
1. **迁入**：13 个目录和 `patches/` 与 `02c19e1` 中对应的 git 树对象相同（计划 Task 1 在提交前逐个比较）。
2. **安装**：`pnpm install --frozen-lockfile` 在干净的克隆上成功；13 个 importers 与 Plane 的锁文件逐行相同；`make gen-check-web` 没有差异。
3. **检查**：`make lint-web` 通过（46 个任务），其中包括 `@nerve/api-client#check:types`；在 web 中加一条警告后，`make lint-web` 失败并报 `Exceeded maximum number of warnings. Found 780.`，删除后恢复。
4. **构建**：`make build-web` 通过；`make build` 产出 `bin/nerve`，执行后 `git status` 为空（产物都被 `.gitignore` 挡掉）。
5. **Go**：`make lint-go` 输出 `0 issues.`；`make test` 全部通过，包括 `webui` 的 9 个测试和 `bootstrap` 的 2 个新测试；故意让 `webui` 导入 `httpserver` 时架构测试失败，删除后恢复。
6. **手工验证**（`make dev-db` 已启动）：
   - `make build` 之前，`make run` 的 `GET /` 返回 404 和提示文字；
   - `make build` 之后，`NERVE_ENV=dev bin/nerve serve`：`GET /` 和深层路径返回同一个 `index.html`（`no-cache`）；`assets/` 下的 JS 带 `immutable`；缺失的 `assets/` 文件 404；`POST /` 405；`GET /api/v0/instance` 200；`GET /api/v0/nope` 404 problem+json；
   - `make run` 和 `make web-dev` 同时运行：`http://127.0.0.1:3000/api/v0/instance` 返回 Go 服务的 JSON。
7. **持续集成通过**：`server` 和 `web` 两个任务都通过；`web` 任务执行了 `make build-web`，第二次运行命中 pnpm 缓存。
8. 前端改动清单登记了完整的来源 SHA 和 1.1 中的全部改动；三个交给 P5 的 handoff 为 `done`。

## 5. 不在 P5 范围内
- 功能删减、品牌替换、改用 React Router 原生写法、包名改为 `@nerve/*`（M1）。
- 前端对接 Nerve 的接口：实例信息、认证、令牌管理器（M2 起）。
- knip 的配置、Playwright 端到端测试、持续集成的 `e2e` 任务（P6）。
- 安全相关的响应头（CSP 等，总体设计 4.3），见第 7 节。

## 6. 风险

| 风险 | 应对 |
|---|---|
| 持续集成的 `web` 任务变慢：冷缓存预计 3–5 分钟（本机约 1 分钟，未在 GitHub 上实测） | pnpm 存储按锁文件缓存；turbo 在任务内部复用包的构建。推送后记录耗时；超过约 5 分钟时，再评估只在 `main` 上保存 turbo 缓存，或把 `lint-web` 和 `build-web` 拆到并行的任务中 |
| 前端构建内存不足：`react-router build` 峰值 2.4 GB | `ubuntu-24.04` 有 16 GB，Node 默认的堆上限足够；出现 `JavaScript heap out of memory` 时，像 Plane 的持续集成一样设置 `NODE_OPTIONS=--max-old-space-size=4096` |
| 警告数取决于 oxlint 的版本和配置 | 版本由 catalog 写死（1.51.0）；升级 oxlint 或修改 `.oxlintrc.json` 时在同一个提交里重新测量上限 |
| 基线只拦住"升"，不强制"降"：修掉警告后忘了调低上限，之后警告又能涨回去 | README 和评审把关；M1 重新测出基线时决定是否加自动检查 |
| Nerve 自己的依赖在重新安装时解析出不同的间接版本 | 以提交的锁文件为准，持续集成用 `--frozen-lockfile`；13 个 Plane 包的版本由核对命令守住 |
| 构建产物不完全确定：两次构建中个别代码块的哈希文件名不同 | 验收只核对文件数量和行为，不核对产物的哈希 |
| `make build` 之后，`dist/` 中 34 MB 的文件会被 `webui`、`bootstrap` 的测试程序一起编译 | 只影响本机、只多几秒；持续集成中 `dist/` 只有 `.gitkeep` |
| M1 改动 Vite 的 `build.assetsDir` | `webui` 的缓存和 404 规则依赖 `assets/` 这个名字；改动时同步修改 `webui` 的常量和测试 |
| Plane 上游的 Express、path-to-regexp 相关覆盖项让 web 的 `serve`（`start` 脚本）无法运行（`pathToRegExp.compile is not a function`） | Nerve 不使用 `serve`；M1 删除它和相关的覆盖项（第 7 节） |

## 7. 移交给后续阶段的事项（评审时建立 handoff）

| 交给 | 事项 |
|---|---|
| M1 | 删除迁入的部署遗留：web 中的 `Dockerfile.web`、`Dockerfile.dev`、`caddy/`、`.dockerignore`；`serve` 依赖和 `start`、`preview` 脚本（当前运行即崩溃）；`public/` 中从未注册的 `sw.js`、`workbox-*.js` 及其 source map；`.env.example` 中 admin、space、live 的地址 |
| M1 | 删掉 `serve` 之后复查 `pnpm-workspace.yaml` 中与 Express 相关的覆盖项（`@react-router/serve>express`、`path-to-regexp`、`router>path-to-regexp`、`body-parser`、`morgan`、`qs`），以及 `turbo.json` 中 admin、space、live 的环境变量；Plane 注释中提到 apps/live 的地方随之删除 |
| M1 | 删减之后重新测出每个包的警告上限（M0 设计 5.2），并决定是否加"警告减少后必须调低上限"的自动检查 |
| M1 | 每次打开页面控制台都有 React #418（水合不匹配，来自 Plane 构建时预渲染的 `index.html`）；深层路径被前端补上结尾的 `/`（Next.js 兼容垫片的行为） |
| M1 | 构建时的两条警告：`tailwind-config` 的 `package.json` 缺少 `"type": "module"`（Node 的 `MODULE_TYPELESS_PACKAGE_JSON`）；Vite 8 已支持 `resolve.tsconfigPaths`，`vite-tsconfig-paths` 插件可以去掉 |
| M1 | `make web-dev` 只运行 web 的开发服务器，改了 `web/packages/*` 要重启；需要同时监视各包时改为 `turbo run dev --filter=web...`，并把并发数设为 11 以上 |
| M2 | 前端调用的是 Plane 的接口：启动时的 `GET /api/instances/` 改为 `GET /api/v0/instance`；`/auth/...` 等不在 `/api` 下的路径，在内嵌的前端中会得到 `index.html`（200，`text/html`），而不是 404，新的认证接口必须放在 `/api/v0/` 下 |
| M2 | 保持同源部署：不设置 `VITE_API_BASE_URL`，接口一律用相对路径；Vite 的代理只转发 `/api` |
| M2 | 安全相关的响应头：Plane 的 Caddyfile 设置了 `X-Frame-Options`、`X-Content-Type-Options`；总体设计 4.3 要求严格的 CSP。`webui` 目前不设置，随认证和令牌管理器一起决定放在哪一层 |
| P6 | `make build` 需要 Node 和 Go，产出约 50 MB 的 `bin/nerve`；`e2e` 任务可以沿用 `web` 任务的 pnpm 缓存步骤 |
| P6 | S2 的"静态资源没有 404"：M0 中唯一的 404 是前端调用的 `/api/instances/`（接口，不是静态资源），断言时按资源类型区分；页面文字仍是 Plane 的，M1 会替换品牌，不要断言"Plane"字样；控制台固定有 React #418 和这个 404，M0 中不要断言"控制台没有错误" |
| P6 | S3 要求 `version` 与构建时注入的版本号一致：`make build` 目前不注入版本号（默认 `0.1.0-dev`），需要时加上 `-ldflags "-X github.com/open-nerve/NerveProject/server/internal/platform/buildinfo.version=<版本>"` |
| P6 | knip 忽略构建产物：`web/apps/web/build/`、`web/apps/web/.react-router/`、`web/packages/*/dist/`、`server/internal/platform/webui/dist/`（P3 交给 P6 的事项仍然有效） |
