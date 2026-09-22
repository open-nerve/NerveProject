# M0/P6 端到端测试骨架与持续集成：设计说明（spec）

| 项 | 内容 |
|---|---|
| Phase | M0/P6 `e2e-ci` |
| 日期 | 2026-09-22 |
| 状态 | 已批准 |
| 上级文档 | [M0 设计文档](../M0-design.md) 第 1、2、3.3、3.5、3.6、3.8、6.1–6.3、8、9、10、12 节；[v0 总体设计](../../v0-design.md) 6.8、7.6、8 节 |
| 前置交接 | [P2-server-platform-p6-e2e-notes](../handoffs/P2-server-platform-p6-e2e-notes.md)、[P3-api-contract-p6-notes](../handoffs/P3-api-contract-p6-notes.md)、[P5-web-import-p6-notes](../handoffs/P5-web-import-p6-notes.md)；都在本 Phase 处理，见 2.11 |

## 1. 目标
P6 是 M0 的最后一个 Phase：接上流水线的最后一环"端到端测试"，并放进持续集成：
- `make build` 注入版本号（控制者裁定）；
- Playwright 工作区包 `@nerve/e2e`：每个 worker 一个从模板库复制的数据库、一个 `bin/nerve serve`，以及生成的 TS 客户端；
- M0 设计第 9 节的四个冒烟故事 S1–S4；
- `make e2e`；
- 持续集成的 `e2e` 任务；
- knip 的配置和 `make knip`：M0 只出报告，M1 起作为门禁；
- README 的"端到端测试"一节；关闭交给 P6 的三个交接。

## 2. 交付物

### 2.1 文件总览
路径都相对于仓库根目录。"生成"表示由命令生成并提交，不手改。

| 路径 | 内容 |
|---|---|
| `Makefile` | `VERSION` 与版本号注入（2.7）；`e2e`（2.7）；`knip`（2.8） |
| `e2e/package.json`、`e2e/tsconfig.json`、`e2e/.gitignore` | 工作区包 `@nerve/e2e`（2.4） |
| `e2e/playwright.config.ts`、`e2e/global-setup.ts` | Playwright 配置；每次运行启动 Postgres、迁移模板库（2.5） |
| `e2e/fixtures/db.ts`、`server.ts`、`api.ts`、`test.ts` | fixtures（2.5） |
| `e2e/stories/smoke/s1-server-ready.spec.ts`、`s2-web-app.spec.ts`、`s3-instance-info.spec.ts`、`s4-unknown-api.spec.ts` | 冒烟故事（2.6） |
| `pnpm-workspace.yaml` | `allowBuilds` 拒绝三个安装脚本（2.4） |
| `package.json` | 开发依赖加入 `knip`（2.8） |
| `pnpm-lock.yaml` | 生成（2.4） |
| `knip.jsonc` | knip 的配置（2.8） |
| `.github/workflows/ci.yml` | `web` 任务输出 knip 报告；新增 `e2e` 任务（2.9） |
| `README.md` | "端到端测试"一节；前端一节加上 knip（2.10） |
| `docs/v0/M0-foundation/handoffs/` 中交给 P6 的三个文件 | 改为 `done`，写明处理结果（2.11） |

本 Phase 不改动 Go 代码和 `server/go.mod`。

新增的 TS 模块之间的导入：
```
stories/smoke/*.spec.ts ──→ fixtures/test.ts（S1 另外导入 fixtures/server.ts 的 runNerve、applicationName）
fixtures/test.ts ──→ fixtures/db.ts、fixtures/server.ts、fixtures/api.ts、@playwright/test
global-setup.ts ──→ fixtures/db.ts、fixtures/server.ts
fixtures/db.ts ──→ @testcontainers/postgresql、pg
fixtures/server.ts ──→ 只有 Node 标准库；运行 bin/nerve
fixtures/api.ts ──→ @nerve/api-client
```

### 2.2 原型验证：结论与证据
在仓库外的临时目录里先做原型（从 P5 的重放仓库出发，代码和锁文件与 P5 合并后的 `main` 相同），再在 P5 合并后的 `main`（`376f662`）的新克隆上按计划的 5 个 Task 完整执行一遍（每个 Task 一个提交），最后在干净的克隆上跑了持续集成三个任务的全部步骤。本 spec 和计划中的数值都来自这些运行。机器：Apple M 系列 18 核、macOS，Node 24.15.0、pnpm 11.10.0、Go 1.27.1、Docker Desktop 4.87（Engine 29.7.2）。

| 问题 | 结论 | 证据 |
|---|---|---|
| Playwright 1.63.0 是否存在、能否在 Node 24 上运行 | 能 | npm 的 `latest`，2026-09-04 发布；`pnpm exec playwright --version` 输出 `Version 1.63.0`。需要的浏览器是 Chromium 153（playwright chromium v1243） |
| Playwright 能否转译 `@nerve/api-client` 的 TS 源码 | 能，不需要配置 | `e2e/node_modules/@nerve/api-client` 是指向 `web/packages/api-client` 的符号链接，真实路径不在 `node_modules` 下。`e2e` 是 ESM 包（`type: module`），相对导入不写扩展名。S3 经 `fixtures/api.ts` 调用 `createClient`，运行通过；把路径改成 `/api/v0/instances`，`make lint-web` 在 `@nerve/e2e#check:types` 报 TS2345 |
| testcontainers-node 的版本 | `@testcontainers/postgresql` 12.1.0 | npm 的 `latest`（2026-08-04），依赖 `testcontainers` 12.1.0（`engines.node: >= 22.22`）；回收容器是 `testcontainers/ryuk:0.14.0`，保持启用 |
| 一个容器还是每个 worker 一个 | 每次运行一个容器，每个 worker 一个库 | 与 M0 设计第 8 节"从模板库复制出每个 worker 的数据库"一致。镜像已在本机时，启动容器 0.8–1.0 秒；建模板库并执行 `bin/nerve migrate up` 约 50 毫秒；4 个 worker 同时 `CREATE DATABASE … TEMPLATE`，各约 58 毫秒 |
| worker 的 nerve 需要什么配置 | 只需要数据库地址和监听地址 | `NERVE_ENV=test` 的配置中只有 `database.url` 没有默认值；`NERVE_DATABASE__URL`、`NERVE_SERVER__ADDR` 覆盖。从启动到 `/readyz` 返回 200 约 120 毫秒；SIGTERM 后约 2 毫秒以退出码 0 退出。test 配置的日志级别是 warn，正常运行时日志文件是空的 |
| 整个运行多久 | 本机约 5 秒 | 5 个测试、4 个 worker：Playwright 报告 3.4–5.0 秒；`make e2e`（含没有改动时约 1 秒的 `make build`）约 6 秒 |
| S1–S4 能否通过、该失败时是否失败 | 都能 | 2.6 列出的反证都按预期失败，并在恢复后通过（计划 Task 2、Task 3） |
| 本地的 `web/apps/web/.env` 会不会混进构建 | 会，S2 能发现 | 从 Plane 的 `.env.example` 复制出 `.env`（其中 `VITE_API_BASE_URL=http://localhost:8000`）后 `make build`：`.env*` 是 turbo 构建任务的输入，前端重新构建，页面把 `GET /api/instances/` 发到 `http://localhost:8000`，S2 的两个测试都因同源断言失败；删掉 `.env` 后 `make build` 从 turbo 缓存恢复原来的构建，S2 通过。干净的克隆中没有 `.env`，`make build` 和 S2 都通过（计划 Task 3 Step 10、Task 5 Step 5） |
| 版本号注入 | 可用 | `make build VERSION=9.9.9-p6` 之后 `bin/nerve version` 输出 `nerve 9.9.9-p6 commit=…`；`make e2e VERSION=0.0.0-p6` 中 S3 通过 |
| S2 看到的网络请求 | 静态资源全部成功，唯一的失败是接口 | 首页：1 个文档、114 个脚本、3 个样式表、1 个字体、2 张图片，全部 200；唯一的 404 是 `GET /api/instances/`（XHR）。深层路径：326 个静态资源全部 200，前端在本页内补上结尾的 `/`，没有第二次加载文档。网络空闲约在 650–780 毫秒。React #418 以页面错误（`pageerror`）出现 |
| knip 6.37.0 | 可用，报告 379 处 | npm 的 `latest`（2026-09-18）；本机 2–4 秒。构建过和没有构建过的克隆得到相同的报告（2.8） |
| 锁文件 | 可以复现 | 从 P5 的锁文件出发、加入 e2e 的依赖后执行 `pnpm install`，两次独立生成都是 14777 行、SHA-256 `8691dff8…3fe0`；再加入 knip 是 15393 行、`e47cdd6b…9c87`。原有的包一个都没有少（2.4） |
| 持续集成的步骤 | 在干净的克隆上通过 | 按 `ci.yml` 的顺序执行三个任务的全部步骤，都通过，之后 `git status` 为空（计划 Task 5 Step 5）。本机耗时：安装 6–8 秒，`make knip` 3–4 秒，`make lint-web`（49 个任务，没有缓存）32–41 秒，`make build-web` 6–8 秒，`make lint-go` 7–8 秒，`make test` 8–9 秒，`make e2e` 5–6 秒 |
| `make test`、`make lint-go` | 不受影响 | 本 Phase 不改动 Go 代码；16 个包都是 `ok`，golangci-lint `0 issues.` |

### 2.3 依赖版本（2026-09-22 通过 npm 和 GitHub 核实，写死）

| 包 | 版本 | 用途 | 位置 |
|---|---|---|---|
| `@playwright/test` | 1.63.0 | 测试运行器、浏览器自动化 | `e2e/package.json` 的 `devDependencies` |
| `@testcontainers/postgresql` | 12.1.0 | 启动 Postgres 容器（带来 `testcontainers` 12.1.0） | 同上 |
| `pg` | 8.23.0 | fixture 建库和数据库断言。它自带 ESM 入口，可以按名字导入 `Client`、`escapeIdentifier` | 同上 |
| `@types/pg` | 8.23.1 | `pg` 的类型 | 同上 |
| `typescript`、`@types/node` | `catalog:`（5.8.3、22.12.0） | 类型检查 | 同上 |
| `@nerve/api-client` | `workspace:*` | 生成的 TS 客户端 | 同上 |
| `knip` | 6.37.0 | 未使用代码的报告 | 根 `package.json` 的 `devDependencies` |
| `actions/upload-artifact` | v7（v7.0.1） | 失败时上传报告 | `ci.yml` |
| Postgres 镜像 | `postgres:18.6` | 与 `deploy/compose.dev.yaml`、`pgtest` 相同 | `fixtures/db.ts` |

- `@types/node` 用 catalog 中 Plane 的 22.12.0：e2e 用到的 Node 接口（`child_process`、`net`、`import.meta.dirname`、全局 `fetch`）在其中都有类型。
- 除 Docker、Go、Node 之外不需要全局安装任何东西；Playwright 的浏览器由 `pnpm exec playwright install` 下载到本机的缓存目录。

### 2.4 工作区包 `@nerve/e2e`
- **`package.json`**：`private`，`type: module`，AGPL-3.0-only；脚本只有 `check:types`（`tsc --noEmit`）、`check:lint`（`oxlint --max-warnings=0 .`）、`check:format`（`oxfmt --check .`），与 api-client 相同，`make lint-web` 由 turbo 自动包含它们（46 → 49 个任务）。这三个脚本必须写（P5 交接"turbo 与环境变量"）：`make lint-web` 是 `turbo run check:types check:lint check:format`，没有这些脚本的包被 turbo 悄悄跳过，不报错，e2e 的代码就不受任何检查。
- **不写 `test` 脚本**：Playwright 不经 turbo 运行，命令的入口是 Makefile（`make e2e`，理由见 2.7）；另外 `turbo.json` 中有 Plane 的 `test` 任务，以后执行 `turbo run test`（例如 vitest）时，e2e 的 `test` 脚本会被一起执行。
- **`tsconfig.json`**：与 api-client 相同的严格选项（`strict`、`noUncheckedIndexedAccess`、`moduleResolution: bundler`、`verbatimModuleSyntax`、`noEmit`）；`lib` 只有 ES2022，不含 DOM（测试代码都在 Node 中运行），`types: ["node"]`；包含包内所有 `.ts`。
- **lint**：新代码，上限为 0，不沿用 Plane 的基线。唯一的例外注释是 `fixtures/test.ts` 中的 `no-empty-pattern`：Playwright 从 fixture 函数的第一个参数读取依赖，没有依赖时必须写 `{}`。`playwright.config.ts` 被根目录 `.oxlintrc.json` 的 `*.config.{js,mjs,cjs,ts}` 排除，仍由 tsc 检查。
- **`e2e/.gitignore`**：`playwright-report/`、`test-results/`。放在包内：oxfmt 只读取当前目录下的 `.gitignore`（P5 spec 2.7），`check:format` 在 `e2e/` 中运行，这一个文件同时挡住 git 和格式检查，不需要再改 `.oxfmtrc.json`。
- **`pnpm-workspace.yaml` 的 `allowBuilds`**：testcontainers 经 dockerode 间接依赖 `cpu-features`、`ssh2`、`protobufjs`，它们带安装脚本。pnpm 11 遇到没有登记的安装脚本时，`pnpm install` 报 `ERR_PNPM_IGNORED_BUILDS`、退出码 1（并往 `allowBuilds` 中写入 `set this to true or false` 的占位）。三项都设为 `false`：前两个用 node-gyp 编译通过 SSH 连接 Docker 时用的可选扩展，后一个只检查依赖方写的版本号；testcontainers 连接本机的 Docker，用不到它们。它们放在 Plane 的条目之后，带中文注释说明不来自 Plane；之后的 `pnpm install` 不改动这个文件。
- **锁文件**：在 P5 的锁文件上执行 `pnpm install` 生成，不从头生成。
  - 加入 e2e 后新增 109 个包，原有的包一个都没有少。`debug` 有一个可选的对等依赖 `supports-color`：重新解析时 pnpm 把它解析为图中已有的 `supports-color@10.2.2`（P3 加入的 openapi-typescript 带来），依赖 `debug` 的包的键多了后缀 `(supports-color@10.2.2)`，包括 web 下 13 个包的部分依赖；去掉这个后缀，web 下的 importers 与 P5 逐行相同。
  - 加入 knip 后新增 56 个包，原有的包一个都没有少。knip 写死依赖 `oxc-resolver` 11.24.2；`tsdown` 所用的 `dts-resolver` 有可选的对等依赖 `oxc-resolver`，从 11.20.0 改为解析到 11.24.2（`@emnapi/core` 随之从 1.10.0 变为 1.11.2）。`tsdown` 等包自身的版本不变；之后前端的检查和构建都通过。
  - 同一组输入两次独立生成的锁文件逐字节相同（2.2）；计划给出每次生成的行数、SHA-256 和核对命令。

### 2.5 fixtures 与全局准备
**运行过程**：
```
playwright test
  global-setup.ts（每次运行一次）
    startPostgres()                     postgres:18.6 容器；服务器地址放进 NERVE_E2E_POSTGRES_URL
    createDatabase("nerve_template")
    runNerve(["migrate", "up"], 模板库)  bin/nerve，test 配置
  每个 worker
    db     createDatabase("e2e_w<workerIndex>", "nerve_template")    CREATE DATABASE … TEMPLATE
    nerve  startNerve(db.url, test-results/nerve-w<workerIndex>.log) 空闲端口；轮询 /readyz；结束时 SIGTERM
  每个测试
    baseURL = nerve.baseURL（page、request 用它解析相对路径）；api = createClient({ baseUrl })
  全局收尾：停止容器（全局准备返回的函数）
```

**接口**：
```ts
// fixtures/db.ts
export const templateDatabase = "nerve_template";
export interface Database {
  readonly name: string;
  readonly url: string;
  query<Row extends QueryResultRow>(sql: string, params?: unknown[]): Promise<Row[]>;
}
export async function startPostgres(): Promise<{ stop: () => Promise<void> }>;
export async function createDatabase(name: string, template?: string): Promise<Database>;

// fixtures/server.ts
export const applicationName = "nerve";
export interface Nerve { readonly baseURL: string; stop(): Promise<void> }
export async function runNerve(args: string[], databaseUrl: string): Promise<{ stdout: string; stderr: string }>;
export async function startNerve(databaseUrl: string, logFile: string): Promise<Nerve>;

// fixtures/api.ts
export type Api = ReturnType<typeof createClient>;
export function createApi(baseURL: string): Api;

// fixtures/test.ts
export const test: TestType<{ api: Api } & …, { db: Database; nerve: Nerve } & …>;  // 另外覆盖 baseURL
export { expect } from "@playwright/test";
```

**设计要点**：
- **职责划分**：`db.ts`、`server.ts`、`api.ts` 是普通函数，不依赖 Playwright，全局准备也用它们；`test.ts` 只负责把它们接成 Playwright 的 fixture。故事只从 `test.ts` 导入 `test` 和 `expect`。
- **每次运行一个容器**：全局准备启动容器、迁移模板库，只做一次；worker 从模板复制，迁移不在每个 worker 中重复（M2 起迁移文件变多时仍然只执行一次）。容器由全局收尾停止；运行中途崩溃时由 Ryuk 删除（实测：全局准备在容器启动后失败，容器在约 10 秒内消失）。
- **模板库用 `bin/nerve migrate up` 迁移**：执行被测程序自己的命令，迁移文件内嵌在程序中，M2 加入迁移文件后不需要改 fixture；复制出的库带着 goose 的版本表。M0 没有迁移文件，输出 `no pending migrations`。服务器上的维护库 `postgres` 只用来建库，不被复制。
- **nerve 的环境**：`NERVE_ENV=test`、`NERVE_DATABASE__URL`（本 worker 的库，加 `sslmode=disable` 和 `application_name=nerve`）、`NERVE_SERVER__ADDR`（`127.0.0.1:<空闲端口>`）。调用方自己的 `NERVE_*` 变量都不传给 nerve：被测的是 test 配置本身，开发者 shell 中的 `NERVE_CONFIG_DIR` 之类不应改变结果。test 配置的 `auto_migrate` 是开启的，对已迁移的副本是空操作。
- **`application_name=nerve`**：nerve 自己不设置 application_name（pgx 不发送它）。fixture 把它写进交给 nerve 的地址，pgx 在连接时发送；只有 nerve 拿到这个地址，所以 S1 能在 `pg_stat_activity` 中确认"nerve 连上了本 worker 的库"。
- **端口**：在 `127.0.0.1` 上监听端口 0 取得一个空闲端口，关闭后交给 nerve（P2 交接第 1 条：不从日志中解析端口）。
- **就绪**：每 100 毫秒请求一次 `/readyz`，返回 200 即就绪；nerve 提前退出时立即失败，30 秒后仍未就绪也失败，这两种情况都先杀掉进程，错误信息给出日志文件的位置。
- **停机**：worker 结束时发送 SIGTERM，等待退出；退出码不是 0 就报错；30 秒内没有退出（`shutdown_timeout` 是 20 秒）就发送 SIGKILL 并报错。运行结束后没有遗留的 nerve 进程。
- **日志**：nerve 的标准输出和标准错误写进 `e2e/test-results/nerve-w<workerIndex>.log`，失败时和报告一起上传（2.9）。
- **库名**：`e2e_w<workerIndex>`。`workerIndex` 在一次运行中不重复，测试失败后新起的 worker 得到新的库。worker 的库不删除，容器在运行结束时整个删除。
- **`db.query`**：每次调用新建一个连接，执行完关闭。M0 只有 S1 的一条查询；数据库断言多了以后改为每个 worker 一个连接池（第 7 节）。
- **`baseURL`**：覆盖 Playwright 的 `baseURL` 选项，取本 worker 的 nerve 地址；`page.goto("/")`、`request.get("/healthz")` 都相对于它。

**Playwright 配置**：`testDir: "stories"`；`globalSetup`；持续集成中 `forbidOnly`；报告用 `list`（终端）和 `html`（`open: "never"`，失败时上传）；`trace: "retain-on-failure"`、`screenshot: "only-on-failure"`；一个项目 `chromium`（`Desktop Chrome`）。不设重试：M0 的故事没有不稳定的来源，重试会掩盖问题。worker 数用 Playwright 的默认值（CPU 核数的一半）；4 个测试文件在本机分给 4 个 worker。

### 2.6 冒烟故事（M0 设计第 9 节）

| 故事 | 文件 | 断言 | 反证（计划中演示） |
|---|---|---|---|
| S1 运维人员用 test 配置启动 nerve，服务就绪 | `s1-server-ready.spec.ts` | `/healthz`、`/readyz` 都是 200 和 `{"status":"ok"}`；`pg_stat_activity` 中本 worker 的库上有 `application_name = 'nerve'` 的连接；`nerve migrate status`（同一个库、test 配置）退出码 0，输出 `no migrations` | 让 nerve 连接 `postgres` 库：`/healthz`、`/readyz` 仍然 200，连接数的断言失败（`Received: 0`） |
| S2 用户在浏览器中打开首页 | `s2-web-app.spec.ts`（两个测试） | 首页：文档 200、`text/html; charset=utf-8`，至少加载了一个 `/assets/…js`，所有静态资源都没有失败，页面的所有请求都发往 nerve 自身（同源）；深层路径 `/acme/projects/<uuid>/issues`：文档 200、`text/html`，内容与 `/` 逐字节相同，所有静态资源都没有失败，所有请求同源 | 从内嵌的 `dist/` 中删掉首页引用的样式表：两个测试都失败，失败项是这个样式表（`net::ERR_ABORTED`）。带着从 Plane 的 `.env.example` 复制的 `web/apps/web/.env` 构建：两个测试都失败，不同源的请求是 `http://localhost:8000/api/instances/` |
| S3 调用方查询实例信息 | `s3-instance-info.spec.ts` | 用生成的客户端 `api.GET("/api/v0/instance")`：200、没有 `error`，`data` 等于 `{product: "Nerve", version: <NERVE_VERSION>, commit: <40 位十六进制>, api_version: "v0"}`；类型检查由 `make lint-web` 完成 | 期望的版本号与构建时不同：`version` 的差异；不设 `NERVE_VERSION`：失败并说明原因；路径写错：TS2345 |
| S4 调用方访问不存在的接口 | `s4-unknown-api.spec.ts` | 直接请求 `GET /api/v0/nope`：404、`application/problem+json`，响应体等于 `{status: 404, code: "not_found", title: "Not Found", detail: "no API endpoint for GET /api/v0/nope"}` | 把路径改到 `/api/` 之外：得到前端页面，`Expected: 404, Received: 200` |

- **S1 的顺序**：先查 `pg_stat_activity`，再执行 `nerve migrate status`。命令行进程退出之后，它的服务端连接可能还会在 `pg_stat_activity` 中停留片刻；先查就不会把它误当成 nerve serve 的连接。
- **S2 怎样区分静态资源和接口**（P5 交接"断言时的注意事项"建议按资源类型区分）：按路径。`/api/` 以外的请求都是静态资源，这正是 nerve 的路由规则（M0 设计 3.3：`/api/` 由接口处理，其余交给前端）。监听页面的 `response`（状态码 ≥ 400 算失败）和 `requestfailed`（被浏览器拒绝或中断），`page.goto` 等到网络空闲。不按 Playwright 的资源类型区分：M2 起前端用 `fetch` 调用 `/api/v0/`，也可能用 `fetch` 取静态文件，路径才是稳定的分界。M0 的前端仍调用 Plane 的接口（`/api/instances/` 得到 404），这类请求不计入。
- **S2 为什么断言同源**：nerve 在同一个地址上提供前端和接口（M0 设计 3.3），前端不配置接口地址。Plane 的 `vite.config.ts` 用 dotenv 读取 `web/apps/web/.env`（不进仓库），从 Plane 的 `.env.example` 复制来的 `VITE_API_BASE_URL=http://localhost:8000` 会被编进前端，接口请求发往别处；页面照样加载，静态资源也都成功，只有同源断言能发现（P5 最终评审 Important 2；README 的"前端"一节和 P5 交接都写明不要建立这个文件）。断言覆盖页面的全部请求（包括接口），在 `request` 事件中记录，与请求是否成功无关。构建不依赖 `.env`：它不进仓库，干净的克隆中没有它，`make build` 和 S2 都通过（2.2）；持续集成的检出同样没有它，`e2e` 任务就是这项检查。
- **S2 不断言**页面文字（Plane 的品牌，M1 替换）和控制台（固定有 React #418 和上面的 404，P5 交接"断言时的注意事项"）。
- **S2 为什么是两个测试**：深层路径要在一个新的页面中"直接打开"，不能沿用首页已经加载过的缓存。
- **S3 的期望版本号**：由 `make e2e` 通过环境变量 `NERVE_VERSION` 交给测试（2.7）；没有设置时失败，不回落到默认值。`commit` 断言为 40 位十六进制：`make build` 在 git 检出目录中编译，Go 工具链写入了提交号。
- **S4 为什么不用生成的客户端**：客户端的类型不允许写不存在的路径（P3 交接第 1 条）。用 Playwright 的 `request` 直接请求，而不是全局的 `fetch`：它使用 `baseURL`，请求会出现在 trace 中。
- **测试名**用英文，以故事编号开头（`S1: …`），与代码注释一致；故事的中文描述在 M0 设计第 9 节。

### 2.7 版本号注入与 `make e2e`
```makefile
# nerve 的版本号：make build 把它写进 bin/nerve，端到端测试 S3 核对它。
# 默认值与 server/internal/platform/buildinfo 中的相同；发布时指定，例如 make build VERSION=0.1.0
VERSION ?= 0.1.0-dev
GO_LDFLAGS := -X github.com/open-nerve/NerveProject/server/internal/platform/buildinfo.version=$(VERSION)

build: build-web
	…
	cd server && go build -ldflags "$(GO_LDFLAGS)" -o ../bin/nerve ./cmd/nerve

e2e: build
	cd e2e && NERVE_VERSION=$(VERSION) pnpm exec playwright test
```
- **`VERSION ?=`**（控制者裁定）：命令行 `make build VERSION=0.1.0` 或环境变量 `VERSION` 都能指定；不指定时是 `0.1.0-dev`，与 `buildinfo` 的默认值相同，`make build` 的产物和 P5 一样。兼容 GNU Make 3.81。改变 `VERSION` 会让 Go 重新链接（链接参数是构建缓存的键的一部分）。
- **默认值写了两处**（Makefile 和 `buildinfo`）：本地 `make e2e` 用默认值，S3 看不出注入是否生效；持续集成注入与默认值不同的 `0.0.0-ci.<运行编号>`（2.9），由它发现注入失效。发布时版本号从哪里来，由 M8 决定（第 7 节）。
- **`make e2e` 依赖 `make build`**：没有改动时 `make build` 约 1 秒（turbo 命中缓存，Go 命中构建缓存），总是先构建，就不会拿旧的 `bin/nerve` 测试。
- **`make e2e` 不安装浏览器**：浏览器是一次性的准备工作（和 `pnpm install` 一样），写在 README；本地调试要完整的 Chromium（`--headed`、`--ui`），持续集成只要无界面的 Headless Shell 和系统库（`--with-deps`，需要 sudo），两者的命令不同。没有安装时 Playwright 报错并给出安装命令。
- **Playwright 不经 turbo 运行**（P5 交接"turbo 与环境变量"给出的两种做法之一）：turbo 2 默认的严格环境变量模式只把 `turbo.json` 中声明的变量交给任务，e2e 需要的 `CI`（`forbidOnly`）、`DOCKER_HOST` 和 `TESTCONTAINERS_*`（testcontainers 据此找到 Docker）、`PLAYWRIGHT_*`，以及每次运行不同的 `NERVE_VERSION` 都会被去掉。用 `passThroughEnv` 放行要维护一张跟着 testcontainers、Playwright 变化的名单，漏掉一个的症状是"找不到 Docker"或 S3 失败，很难联想到 turbo；而 turbo 在这里也没有用处：端到端测试的结果取决于 `bin/nerve` 和 Docker，不在 turbo 的输入中，不能缓存。所以 Makefile 直接在 `e2e/` 中执行 `pnpm exec playwright test`，Playwright 继承调用方的全部环境变量。
- **用 `cd e2e &&` 而不是 P5 交接中写的 `pnpm --filter e2e run <脚本>`**：`--filter` 写错时 pnpm 只打印 "No projects matched the filters"，退出码为 0，测试就被悄悄跳过。
- 命令从 20 个变为 22 个（`e2e`、`knip`），以 Tab 开头的行从 30 行变为 32 行。

### 2.8 knip
**配置**（`knip.jsonc`，带中文注释；jsonc 是 knip 支持的格式）：

| 工作区 | 设置 | 原因 |
|---|---|---|
| 根目录 | `ignoreDependencies: ["@redocly/cli", "turbo"]` | 它们由 Makefile 调用（`make gen-web`、`make lint-web` 等），knip 不读 Makefile，会误报为未使用 |
| `web/apps/web` | `ignoreUnresolved: ["\\+types/"]` | 路由文件导入 `./+types/…`：`react-router typegen` 把它们生成在 `.react-router/types/`（不进仓库），经 tsconfig 的 `rootDirs` 导入。knip 不按 `rootDirs` 解析，无论是否生成过都报 61 条"Unresolved imports" |
| `web/packages/i18n` | `ignoreUnresolved: ["^\\./keys\\.generated$"]` | `src/types/keys.generated.ts` 由 i18n 的构建生成（不进仓库）；没有构建过时 knip 报 1 条"Unresolved imports"，构建过之后能找到 |
| `web/packages/api-client` | `entry: ["test/*.typecheck.ts"]` | 类型测试只由 tsc 检查（P3 交接第 3 条）；作为入口文件，它和它用到的导出都不再被报告 |
| `web/packages/api-client` | `ignoreIssues: {"src/schema.gen.ts": ["types"]}` | 生成的类型 `webhooks`、`$defs`、`operations` 没有被使用（P3 交接第 3 条） |

- **构建产物不需要忽略**（P5 交接"断言时的注意事项"的 knip 一条）：knip 默认读取 `.gitignore`，`web/apps/web/build/`、`.react-router/`、`web/packages/*/dist/`、`server/internal/platform/webui/dist/` 都已在其中。在 `make build` 之后运行，报告中没有这些路径。
- **e2e 被分析**：knip 的 Playwright 插件从配置找到 `global-setup.ts` 和 `stories/`；在 `fixtures/api.ts` 中临时加一个未使用的导出，knip 报告它。
- **报告**（构建过和没有构建过的克隆相同）：

  | 类别 | 数量 |
  |---|---|
  | 未使用的文件 | 131 |
  | 未使用的依赖 | 19 |
  | 未使用的开发依赖 | 4 |
  | 未声明的依赖 | 2 |
  | 未使用的导出 | 135 |
  | 未使用的导出类型 | 83 |
  | 未使用的枚举成员 | 4 |
  | 重复的导出 | 1 |
  | 合计 | 379 |

  全部在迁入的 Plane 代码中，Nerve 自己的代码（api-client、e2e、根目录）没有。另有配置提示：构建过的克隆上 2 条（i18n 的 `ignoreUnresolved` 暂时用不上；`tailwind-config` 的 `main` 指向不存在的 `tailwind.config.js`），没有构建过时只有后一条。
- **`make knip`** = `pnpm exec knip --no-exit-code`：发现问题时退出码仍为 0；knip 自身出错（例如配置中有未知的键）时退出码 2，make 失败。M1 去掉 `--no-exit-code`，就成为门禁。本机 2–4 秒。
- **持续集成执行它**：`web` 任务在 `make lint-web` 之后执行 `make knip`，作为普通的一步（不是 `continue-on-error`）：报告出现在日志中，配置出错会让任务失败，M1 之前配置不会悄悄失效。`continue-on-error` 会把配置错误也变成一个黄色警告。

### 2.9 持续集成
**`e2e` 任务**：
```
needs: [server, web]；if 与另外两个任务相同；ubuntu-24.04；timeout-minutes: 20
env: VERSION=0.0.0-ci.<github.run_number>
checkout → setup-go（与 server 任务相同的缓存键）→ setup-node → corepack enable
→ 缓存 pnpm 存储（与 web 任务相同的键）→ pnpm install --frozen-lockfile
→ playwright install --with-deps --only-shell chromium
→ make build → make e2e
→ 失败时上传 e2e/playwright-report/ 和 e2e/test-results/（名为 playwright-report）
```
- **`needs` 和 `if`**（P3 交接第 4 条）：`server`、`web` 通过后才运行；`if` 条件与两者相同。同仓 PR 上两者被跳过，`e2e` 也被跳过；设为必须通过的检查之前的注意事项不变（M0 设计 6.3）。
- **版本号**：任务级的环境变量 `VERSION` 同时作用于 `make build` 和 `make e2e`。`0.0.0-ci.<运行编号>` 是合法的语义化版本，明显不是正式版本，并且与默认值不同，S3 由此确认注入生效。
- **浏览器**：只装 Chromium 的 Headless Shell（`--only-shell`，测试以无界面方式运行）和它需要的系统库（`--with-deps`）。不缓存 `~/.cache/ms-playwright`：Playwright 的文档不建议缓存浏览器——Linux 上的系统库缓存不了，每次仍要安装；下载与恢复缓存的耗时相当。
- **先 `make build` 再 `make e2e`**：分成两步，失败时一眼能看出是构建还是测试；`make e2e` 中的 `make build` 命中上一步留下的 turbo 和 Go 缓存，约 1 秒。
- **上传**：`if: failure()`。HTML 报告中带着失败测试的 trace 和截图；`test-results/` 中另有每个 worker 的 nerve 日志。
- **`web` 任务**：`make lint-web` 之后加一步 `Unused code (report only)`：`make knip`（2.8）。

**耗时估计**：依据 P5 分支在持续集成上的实测（M0 设计第 12 节），`web` 任务冷运行 142 秒，其中 `pnpm install` 12 秒、`make gen-check-web` 2 秒、`make lint-web` 87 秒、`make build-web` 24 秒；`server` 任务 45 秒。同样的 `make lint-web` 本机 34–41 秒，runner 约是本机的 2.3 倍。推送后由控制者记录实际耗时，写进 P6 的 review。

| 步骤 | 本机 | `ubuntu-24.04` 估计 |
|---|---|---|
| 准备（checkout、Go、Node、corepack） | — | 15–25 秒 |
| 恢复 pnpm 缓存 + `pnpm install` | 7–8 秒（本机存储已有） | 10–20 秒（P5：12 秒） |
| 安装浏览器和系统库 | Headless Shell 下载 13 秒 | 30–60 秒（apt 安装系统库占大半） |
| `make build`（没有 turbo 缓存，11 个前端构建任务全部执行） | 前端 25 秒；Go 冷编译 5 秒 | 75–90 秒（前端 55–60 秒） |
| `make e2e` | 6 秒（镜像已在本机） | 20–40 秒（拉取 `postgres:18.6` 和 Ryuk 10–20 秒） |
| 合计 | — | 约 3–4 分钟 |

整个工作流约 6 分钟：`web` 任务约 2.5 分钟（P5 的 142 秒加上 `make knip`），`e2e` 任务在它之后。

- **`e2e` 任务自己执行 `make build`**：turbo 的缓存不跨任务保存（P5 spec 2.12），`web` 任务构建过的前端在这里要再构建一次，约多花 1 分钟。P5 交接请 P6 评估的替代办法是由 `web` 任务把 `web/apps/web/build/client`（34 MB、1239 个文件）作为产物上传，`e2e` 任务下载后只编译 Go。没有采用：Makefile 要拆出一个跳过前端构建的目标，`e2e` 任务不再执行完整的 `make build`；而持续集成中只有这里完整执行 `make build`，它是 M0 完成标准"`make build` 能构建出单个可执行文件"在持续集成中的证据；上传、下载这么多小文件本身也要十几秒，省下的不到 1 分钟。
- **超过 5 分钟时的备选办法**：在同一次运行中，由 `web` 任务把 turbo 的本地缓存 `.turbo/cache` 作为产物上传（`include-hidden-files: true`），`e2e` 任务在 `make build` 之前下载到原处。`make build` 仍然完整执行，只是前端命中缓存。
- **缓存按分支隔离**（P5 交接"构建、缓存与版本号"）：GitHub Actions 的缓存只在保存它的分支（和从默认分支继承）中可见。分支上保存的 pnpm、Go 缓存 `main` 读不到，合并后 `main` 的第一次运行是冷的；P6 改了锁文件，分支上的第一次运行也是冷的。控制者分别记录分支冷、分支热、合并后 `main` 冷三次运行的耗时（计划 Task 5 Step 8）。

### 2.10 README
- 新增"端到端测试"一节：第一次运行前安装 Chromium；`make e2e` 做什么、需要 Docker；测试环境（全局准备、每个 worker 的库和 nerve、故事从哪里导入）；版本号（`VERSION`、`NERVE_VERSION`）；不要有 `web/apps/web/.env`（2.6）；只运行部分故事和打开浏览器调试的命令；失败时到哪里看报告、trace、截图和 nerve 日志，持续集成上传什么。
- "前端"一节加上"未使用的代码"：`make knip`，M0 只出报告（379 处），M1 改为门禁，持续集成执行它。

### 2.11 处理交给 P6 的交接
- [P2-server-platform-p6-e2e-notes](../handoffs/P2-server-platform-p6-e2e-notes.md)：
  1. 端口由 fixture 选定，经 `NERVE_SERVER__ADDR` 传入，轮询 `/readyz`；不解析日志（2.5）；
  2. 数据库从模板库复制，地址经 `NERVE_DATABASE__URL` 传入（2.5）；
  3. SIGTERM 后等待退出，检查退出码 0，超时发送 SIGKILL（2.5）。
- [P3-api-contract-p6-notes](../handoffs/P3-api-contract-p6-notes.md)：
  1. 通过 `createClient({ baseUrl })` 调用，e2e 参与 `make lint-web` 的类型检查；S4 直接请求（2.6）；
  2. Playwright 能转译 api-client 的 TS 源码，不需要配置（2.2）；
  3. knip：`test/*.typecheck.ts` 作为入口，`schema.gen.ts` 忽略未使用的类型（2.8）；
  4. `e2e` 任务的 `needs` 和 `if`（2.9）。
- [P5-web-import-p6-notes](../handoffs/P5-web-import-p6-notes.md)：
  - 构建、缓存与版本号：`e2e` 任务安装 Go 和 Node，沿用 `web` 任务缓存 pnpm 存储的步骤；评估了上传 `build/client`，不采用，`e2e` 任务自己执行 `make build`，超过 5 分钟时改为交接 turbo 缓存；按 ref 隔离的缓存分三次记录耗时（2.9）；`make build` 注入 `VERSION`，S3 核对（2.7）。
  - turbo 与环境变量：e2e 定义三项检查（2.4）；Playwright 不经 turbo，也不用 `pnpm --filter`（2.7）。
  - 断言：S2 按路径（不按资源类型）区分静态资源和接口，不断言页面文字和控制台；e2e 不碰 `web/apps/web/.env`，S2 的同源断言能发现混进构建的 `.env`（2.6）；knip 读取 `.gitignore`，构建产物不需要另外忽略（2.8）。

三个 handoff 在计划的最后一个 Task 中改为 `done`，并写明处理结果。之后 `docs/v0/M0-foundation/handoffs/` 中没有 `open` 的事项。

## 3. 与上级设计的差异和补充（请控制者裁定）

| # | 上级设计 | P6 的做法 | 理由 |
|---|---|---|---|
| 1 | M0 6.1：`make e2e`"等同于 `pnpm e2e`"；总体设计 8.2："`pnpm e2e` 一条命令完成编译、启动数据库和运行全部故事" | 只有 `make e2e`；根 `package.json` 不加脚本 | P5 已定下命令的入口是 Makefile（P5 spec 2.4）；`make e2e` 完成构建、启动数据库和运行全部故事 |
| 2 | 同上："一条命令完成" | 浏览器单独安装一次（README），`make e2e` 不安装 | 和 `pnpm install` 一样是一次性准备；本地调试和持续集成需要的浏览器不同（2.7） |
| 3 | M0 8 P6、总体设计 8.2：fixtures 是 `server.ts`、`db.ts`、`api.ts` | 另有 `fixtures/test.ts`（接成 Playwright 的 fixture）和 `e2e/global-setup.ts`（每次运行启动容器、迁移模板库） | 三个文件只放普通函数，全局准备也能用；容器每次运行一个，不属于任何 worker（2.5） |
| 4 | 总体设计 8.2：`stories/<领域>/*.spec.ts` | M0 的故事放在 `stories/smoke/` | M0 没有业务领域；M2 起按领域建目录 |
| 5 | 总体设计 8.2：每个故事对应一个测试文件 | S2 的文件中有两个测试（首页、深层路径） | 深层路径要在新页面中直接打开（2.6）；仍然是一个故事一个文件 |
| 6 | 总体设计 8.2：失败时保存操作记录、截图、录像和数据库快照 | M0 保存 trace 和截图，另有 nerve 的日志；不录像，不保存数据库快照 | trace 已含每一步的截屏，录像还要多下载 ffmpeg；M0 没有业务表，数据库快照从 M2 起（第 7 节） |
| 7 | M0 9 S1："`pg_stat_activity` 中能看到 nerve 的连接" | fixture 在交给 nerve 的地址中加 `application_name=nerve`，S1 查本 worker 的库上这个名字的连接 | nerve 自己不设置 application_name；只有 nerve 拿到这个地址，断言因此是精确的（2.5）。是否让 nerve 默认设置它，属于产品行为，不在 P6 决定 |
| 8 | M0 9 S2："所有静态资源都加载成功（没有 404）"；P5 交接：按资源类型区分静态资源和接口 | 静态资源按路径定义：`/api/` 以外的全部请求；除 404 外，被浏览器拒绝或中断的请求也算失败；不断言页面文字和控制台 | 与 nerve 的路由规则一致（M0 设计 3.3）；M2 起接口和静态文件都可能用 `fetch` 取，资源类型分不开（2.6） |
| 9 | M0 9 S3："`version` 与构建时注入的版本号一致" | `make build` 用 `VERSION ?= 0.1.0-dev` 注入；`make e2e` 经 `NERVE_VERSION` 交给测试；持续集成注入 `0.0.0-ci.<运行编号>`；另外断言 `commit` 是 40 位十六进制 | 控制者裁定在 `make build` 中注入；默认值与 `buildinfo` 相同，所以由持续集成的不同值发现注入失效（2.7） |
| 10 | P3 交接第 1 条：S4 用 `fetch` 直接请求 | 用 Playwright 的 `request` 直接请求 | 同样不经过生成的客户端；它使用 `baseURL`，请求记录在 trace 中 |
| 11 | M0 6.3：`e2e` 任务是 `make build` → 安装浏览器 → 运行测试 | 安装浏览器 → `make build` → `make e2e`；浏览器只装 Headless Shell 和系统库，不缓存；`timeout-minutes: 20` | 顺序不影响结果；`make e2e` 中的构建命中缓存；不缓存的理由见 2.9；`e2e` 任务要拉取镜像和安装系统库，比另外两个任务多留 5 分钟 |
| 12 | M0 8 P6、总体设计 8.3：knip"M0 只出报告" | `make knip` 带 `--no-exit-code`；持续集成的 `web` 任务执行它，作为普通的一步 | 发现问题不失败，配置出错失败，M1 之前配置不会失效；M1 只需去掉一个参数（2.8） |
| 13 | P3 交接第 3 条：knip 忽略 `test/**` | `test/*.typecheck.ts` 作为 api-client 的入口文件 | 它们确实被使用（由 tsc 检查），作为入口比忽略更准确，它们用到的导出也不会被误报 |
| 14 | P5 交接：knip 忽略构建产物 | 不另外配置 | knip 读取 `.gitignore`，构建产物已被排除（2.8） |
| 15 | — | knip 另有三项配置：根目录的 `ignoreDependencies`、web 和 i18n 的 `ignoreUnresolved` | 前者是 Makefile 调用的工具；后两者是生成的、不进仓库的文件。没有它们，报告多出 62 条"Unresolved imports"，并随本地是否构建过而变化（2.8） |
| 16 | M0 2：仓库布局 | 根目录另有 `knip.jsonc`；`e2e/` 中有自己的 `.gitignore` | jsonc 可以写注释，说明每一项配置的原因；`.gitignore` 放在包内的理由见 2.4 |
| 17 | M0 5.1、P5 spec 2.4：catalog、overrides、allowBuilds 等"只保留作用于迁入的包的条目"；P5 spec 2.2、2.5：13 个 importers 与 Plane 的锁文件"逐行相同" | allowBuilds 加入三项 Nerve 的条目（`false`）；锁文件中依赖 `debug` 的包多了对等后缀 `(supports-color@10.2.2)`，`dts-resolver` 的可选对等依赖 `oxc-resolver` 解析为 11.24.2 | 前者是 pnpm 11 安装 testcontainers 的前提；后者是 pnpm 重新解析锁文件的结果，所有包的版本不变，前端的检查和构建都通过（2.4） |
| 18 | M0 1：端到端测试"Node 端用 testcontainers 启动 Postgres"，没有写版本 | testcontainers-node 12.1.0（`@testcontainers/postgresql`）；另用 `pg` 8.23.0 建库和断言 | 2.3 |
| 19 | M0 9 S2：只要求页面加载、静态资源没有 404 | S2 另外断言页面的所有请求都发往 nerve 自身（同源） | nerve 同源提供前端和接口（M0 设计 3.3）；本地 `.env` 混进构建时页面照样加载，只有这条断言能发现（P5 评审提出，2.6） |

控制者裁定后，评审阶段把 M0 设计第 1 节（testcontainers-node、pg 的版本）、第 2 节（`e2e/` 的内容、`knip.jsonc`）、6.1 节（`make e2e`、`make knip`、`VERSION`）、6.3 节（`e2e` 任务的步骤、`web` 任务的 knip 一步）、第 9 节（S1–S3 的断言写法）、第 12 节（持续集成的耗时），总体设计 8.2 节（`pnpm e2e`、fixtures、失败时保存的内容）、8.3 节（knip 在 M0 只出报告），以及前端改动清单中 `pnpm-workspace.yaml`、`pnpm-lock.yaml` 两行同步更新。

## 4. 验收标准
1. **本地**：`make e2e` 通过（5 个测试：S1、S2 两个、S3、S4）；之后没有遗留的 nerve 进程，testcontainers 的容器在约 10 秒内消失。
2. **反证**（计划 Task 2、Task 3）：S1（nerve 连错库）、S2（缺少样式表；带着 `web/apps/web/.env` 构建，请求不同源）、S3（版本号不一致、没有 `NERVE_VERSION`、路径写错时类型检查失败）、S4（路径不在 `/api/` 下）都按预期失败，恢复后通过。
3. **版本号**：`make build VERSION=9.9.9-p6` 之后 `bin/nerve version` 输出 `nerve 9.9.9-p6`；`make e2e VERSION=0.0.0-p6` 通过；不指定时 `bin/nerve version` 是 `0.1.0-dev`。
4. **检查**：`make lint-web` 49 个任务全部成功，其中包括 `@nerve/e2e` 的三项检查；`make lint-go` 输出 `0 issues.`；`make test` 全部通过；`server/go.mod` 没有改动。
5. **knip**：`make knip` 退出码 0，报告 379 处，全部在 Plane 的代码中；配置出错时退出码非 0；构建过和没有构建过的克隆报告相同。
6. **锁文件**：`pnpm install --frozen-lockfile` 在干净的克隆上成功；原有的包一个都没有少（计划 Task 2 Step 3、Task 4 Step 2 的核对命令）。
7. **干净的克隆**：按 `ci.yml` 的顺序执行三个任务的全部步骤都通过，之后 `git status` 为空。
8. **持续集成**：`server`、`web`、`e2e` 三个任务都通过，`web` 任务的日志中有 knip 的报告；推送一个让 S4 失败的临时分支，`e2e` 任务失败并上传 `playwright-report`。记录分支冷、分支热、合并后 `main` 冷三次运行的耗时。
9. **交接**：交给 P6 的三个 handoff 为 `done`，`docs/v0/M0-foundation/handoffs/` 中没有 `open` 的事项。

## 5. 不在 P6 范围内
- PAT 对等验收、认证相关的 fixture（M2）。
- 数据库断言函数库、失败时的数据库快照（M2 起，有业务表之后）。
- 总体设计 8.2 中的 `webhook.ts`、`storage.ts` 和可注入的时钟（需要它们的 M 再加入）。
- knip 作为门禁、清零 Plane 代码的报告（M1）。
- Chromium 以外的浏览器。
- 让 nerve 自己设置 `application_name`（第 3 节第 7 项）。
- 发布版本号的来源和发布流程（M8）。

## 6. 风险

| 风险 | 应对 |
|---|---|
| 持续集成的 `e2e` 任务比估计的慢（前端再构建一次、安装系统库、拉取镜像） | 推送后记录分支冷、分支热、合并后 `main` 冷三次运行的耗时；超过 5 分钟时，按 2.9 的备选办法把 `web` 任务的 turbo 缓存交给 `e2e` 任务 |
| 开发者本地有 `web/apps/web/.env`（从 Plane 的 `.env.example` 复制），构建出的前端把接口请求发往 `http://localhost:8000` | S2 的同源断言发现它，失败信息列出不同源的请求；README 写明不要有这个文件；M1 去掉读取它的来源（第 7 节） |
| 合并到 `main` 后第一次运行没有缓存（GitHub 的缓存按分支隔离） | 预期内，耗时按冷运行记录（2.9） |
| Docker Hub 匿名拉取的频率限制（`postgres:18.6`、`testcontainers/ryuk`） | 与 M0 设计第 12 节相同：出现限流时在持续集成中登录 Docker Hub 或改用镜像缓存 |
| 取空闲端口和 nerve 监听之间有一个很短的窗口，端口可能被别的进程占用 | nerve 监听失败会立即退出，fixture 报"nerve exited"并给出日志位置；M0 中没有遇到过 |
| `VERSION` 的默认值写在 Makefile 和 `buildinfo` 两处，改版本号时可能只改了一处 | Makefile 的注释写明两者相同；持续集成注入不同的值，S3 能发现注入失效；M8 定下发布时版本号的来源 |
| 环境中恰好有一个无关的 `VERSION` 变量，`make build` 会用它 | `?=` 是控制者裁定的写法，也是持续集成传入版本号的方式；`bin/nerve version` 能看出实际写入的值 |
| 任何依赖的增减都会让 pnpm 重新解析可选的对等依赖，Plane 包的锁文件条目随之出现后缀的变化（2.4） | 计划中的核对命令确认没有包被删掉、版本不变；M1 删减依赖时沿用这组命令（第 7 节） |
| knip 在构建过的克隆上提示 i18n 的 `ignoreUnresolved` 可以删掉（"Remove from ignoreUnresolved"） | knip.jsonc 的注释写明不要删：没有构建过时它才起作用，删掉之后报告随本地是否构建过而变化 |
| M2 起前端改调 `/api/v0/`，S2 不再有接口的 404，但控制台和页面的断言仍未加入 | 交给 M2（第 7 节） |
| Playwright 升级后浏览器版本变化，本地要重新安装 | Playwright 的报错给出安装命令；README 写明升级后重新执行 |

## 7. 移交给后续阶段的事项（评审时建立 handoff）

| 交给 | 事项 |
|---|---|
| M1 | knip 改为门禁：`make knip` 去掉 `--no-exit-code`，报告清零；决定是否把配置提示也作为错误（`--treat-config-hints-as-errors`）；`tailwind-config` 的 `main` 指向不存在的 `tailwind.config.js`（knip 的配置提示）。M1 删掉 react-router 的 typegen 或 i18n 的生成步骤时，同步删掉 `knip.jsonc` 中对应的 `ignoreUnresolved` |
| M1 | 重新测出 lint 基线时，`@nerve/api-client`、`@nerve/e2e` 的上限保持 0（它们是 Nerve 的新代码） |
| M1 | 删减依赖后锁文件的核对：沿用 P6 计划 Task 2 Step 3 的命令（没有意外删掉的包、web 下 importers 只有对等后缀的变化） |
| M1 | 品牌替换之后，S2 仍然不应断言页面文字（页面在 M2 前仍是错误页）；前端的包名改为 `@nerve/*` 时，`knip.jsonc` 中的工作区路径不变 |
| M1 | `.env`：删除整个 `.env.example`、复查 `vite.config.ts` 的 dotenv 加载，已在 P5 交给 M1 的 [M0-P5-frontend-trim-notes](../../M1-frontend-trim/handoffs/M0-P5-frontend-trim-notes.md) 中，P6 不另建。P6 只补充：S2 的同源断言保留，作为回归检查；去掉 dotenv 加载之后，README"前端"一节"不要建立 `web/apps/web/.env`"一条和"端到端测试"一节"同源"一条中对它的引用随之修改 |
| M2 | 每个故事的 PAT 版本：用 PAT 调接口走完同样的流程，调用同一组数据库断言函数（总体设计 8.2）；认证的 fixture（通过接口注册、登录，页面使用登录状态） |
| M2 | `fixtures/db.ts`：数据库断言多了以后改为每个 worker 一个连接池，并建立断言函数的写法；失败时保存本 worker 数据库的快照（`pg_dump`） |
| M2 | S1 加上"迁移版本正确"：`nerve migrate status` 列出全部迁移为 applied，`goose_db_version` 的最新版本等于最后一个迁移文件 |
| M2 | S3：`/api/v0/instance` 加入 `signup_enabled` 之后，S3 的 `toEqual` 要随之更新 |
| M2 | 前端改调 `/api/v0/instance`（P5 交给 M2 的 [M0-P5-frontend-api-notes](../../M2-auth/handoffs/M0-P5-frontend-api-notes.md)）之后，`/api/instances/` 的 404 消失：S2 可以加上"接口请求没有失败"，并决定是否断言控制台（React #418 另由 M1 处理） |
| M8 | 发布时版本号的来源（`make build VERSION=…`、git 标签）；Makefile 与 `buildinfo` 两处默认值合一；把三个任务设为必须通过的检查时，按 M0 设计 6.3 处理跳过条件或加汇总任务 |

## 8. 与 M0 完成标准的对照（M0 设计第 10 节）
完成标准由控制者在 M0 结束时逐项核对并勾选；下表是 P6 提供的证据。

| 完成标准 | P6 的证据 |
|---|---|
| P1 到 P6 全部完成，每个 Phase 都有 spec、plan 和 review | 本 spec、计划；review 在代码评审后建立 |
| 持续集成中的全部门禁通过（生成物一致性、golangci-lint、Go 测试、前端类型检查、oxlint 按基线、前端构建、端到端冒烟故事） | 计划 Task 5 Step 8：`server`、`web`、`e2e` 三个任务通过；`e2e` 任务运行 S1–S4 |
| `make build` 能构建出单个可执行文件 `bin/nerve`；它加上一个 Postgres 就能完成 S1 到 S4 | `make e2e` 正是这样运行的：被测对象只有 `bin/nerve` 和 testcontainers 启动的 `postgres:18.6` |
| 第 7 节的调整建议已确认并同步 | 已勾选（P6 不涉及） |
| 前端改动清单中已登记迁入时的改动 | P5 完成；P6 的两处补充见第 3 节第 17 项 |
| `handoffs/` 中没有 `open` 的事项；交给后续 M 的事项已放进对应 M 的 `handoffs/` | 计划 Task 5 Step 4：M0 的交接全部为 `done`；交给 M1、M2、M8 的事项在评审时按第 7 节建立 |
| 总体设计中 M0 的状态改为"已完成" | 由控制者在 M0 结束时修改 |
