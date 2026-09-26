# M0 基础骨架：架构设计与实施规划

| 项 | 内容 |
|---|---|
| 里程碑 | M0 基础骨架（`docs/v0/M0-foundation`） |
| 日期 | 2026-09-22 |
| 状态 | 已批准（2026-09-22） |
| 上级文档 | [v0 总体设计](../v0-design.md) |
| 前置交接 | 无（`handoffs/` 为空） |

---

## 0. 目标与范围

### 0.1 目标
搭好后续所有里程碑都要用到的工程基础，并用一条最小的"试点链路"证明整条流水线是通的：

**OpenAPI 描述 → 生成 Go 接口层 → 模块（领域 / 用例 / 适配器）→ 组合根接线 → HTTP 服务 → 生成 TS 客户端 → 端到端测试**

M0 结束时，一个开发者克隆仓库后，用几条命令就能：
1. 启动开发数据库。
2. 运行后端和前端。
3. 跑完所有检查和测试。
4. 打出一个内嵌前端的 `nerve` 单个可执行文件。

### 0.2 范围
| 包含 | 不包含（留给后续 M） |
|---|---|
| 仓库布局、工具链与版本锁定、开发环境、Makefile | 任何业务表（见 7.1） |
| Go 服务端骨架：配置、日志、数据库连接、迁移机制、HTTP 服务与中间件、统一错误格式、健康检查、命令行 | River 任务队列、sqlc（第一次真正用到时，在 M2 接入，见 7.2） |
| 接口契约流水线，加上试点模块 `instance`（`GET /api/v0/instance`） | 认证、限流（M2） |
| 架构守护：架构测试和禁用依赖检查 | 前端删减、品牌替换、改用原生写法（M1） |
| Plane 表结构快照工具及快照文件 | 前端对接新接口（M2 起） |
| 前端代码原样迁入、能构建、内嵌进 Go 程序 | |
| 端到端测试骨架和冒烟故事；持续集成流水线 | |

---

## 1. 版本基线

以下版本均为 2026-09-22 的最新稳定版，在 P1 中锁定。

| 类别 | 选择 | 说明 |
|---|---|---|
| Go | **1.27.1** | `go.mod` 中写 `go 1.27` 和 `toolchain go1.27.1`。本机的 Go 1.26 会自动下载对应版本的工具链。goose、sqlc、River 都要求 Go 1.26 以上 |
| UUID | Go 1.27 **标准库 `uuid`**（`uuid.NewV7()`） | 不引入第三方 uuid 库。ID 由应用生成，数据库不设 `DEFAULT uuidv7()`，这样也绕开了 sqlc 解析器（基于 PG 17）不认识 `uuidv7()` 的问题 |
| PostgreSQL | **18.6**（镜像 `postgres:18.6`） | |
| 数据库驱动 | pgx v5（`pgxpool`） | |
| 迁移 | goose **v3.28.0** | 迁移文件通过 `embed.FS` 编进程序，用 Provider API 在代码中执行 |
| 接口代码生成 | oapi-codegen **v2.8.0** | `strict-server` + `std-http-server`（基于 Go 标准库的 `ServeMux`）；支持按 tag 生成，支持跨文件引用 |
| 接口校验（测试用） | kin-openapi **v0.149.0** | 支持 OpenAPI 3.1 |
| 配置 | koanf **v2.3.6** | 解析器用 yaml；数据来源用 rawbytes（读取内嵌的文件）、file、env/v2。没有用 koanf 的 fs provider，因为它还标着"实验性" |
| 命令行 | cobra | `nerve serve`、`nerve migrate`、`nerve version` |
| 静态检查 | golangci-lint **v2.13.2** | 包含 depguard |
| 集成测试 | testcontainers-go **v0.44.0**（Postgres 模块） | |
| Node | **24 LTS** | Plane 要求 ≥22.22 |
| pnpm | **11.10.0** | 通过 corepack，按 `packageManager` 字段锁定版本 |
| TypeScript | **5.8.3**（沿用 Plane 的版本） | TypeScript 7 已发布，但 openapi-typescript 仍声明只支持 `^5`。v0 期间不升级 |
| 前端框架 | React 19.2、React Router 8.3、Vite 8、turbo 2.10、oxlint 1.51、oxfmt 0.35 | 沿用 Plane 的版本 |
| TS 客户端生成 | openapi-typescript **7.13.0**、openapi-fetch **0.17.0** | |
| OpenAPI 打包 | Redocly CLI **2.53.3** | 版本已在 P3 锁定 |
| 端到端测试 | Playwright **1.63.0**；Node 端用 testcontainers-node **12.1.0**（`@testcontainers/postgresql`）启动 Postgres，`pg` **8.23.0** 建库和做数据库断言 | |
| 未使用代码检查 | knip **6.37.0** | M0 只出报告（`make knip --no-exit-code`，`web` 任务），M1 开始作为门禁 |

**工具怎么安装**：
- Go 的开发工具写在单独的 `server/tools/go.mod` 里，通过 `go tool -modfile=tools/go.mod <工具>` 调用，不污染主模块的依赖。M0 只需要 oapi-codegen；sqlc 在 M2 加入。goose 以库的形式在代码中调用，不需要命令行工具。
- golangci-lint 按官方建议使用预编译的二进制，由 `make tools` 下载到 `./bin`。lint 按区域拆分为 `lint-go`、`lint-web`（M0/P3，见 6.1、6.3）；本地和持续集成都执行同一套 `lint-go`、`lint-web`。
- 除了 Docker、Go、Node 以外，**不需要全局安装任何东西**。

---

## 2. 仓库布局（M0 完成时）

```
NerveProject/
  Makefile                      所有常用命令的统一入口（见 6.1）
  package.json                  pnpm 工作区的根（packageManager 字段锁定 pnpm 版本）
  pnpm-workspace.yaml           工作区：web/apps/*、web/packages/*、e2e；依赖版本表（catalog）
  pnpm-lock.yaml                锁文件，以 Plane 的锁文件为起点生成（见 5.1）
  turbo.json                    前端任务编排
  patches/                      react-color 的补丁，pnpm-workspace.yaml 的 patchedDependencies 引用它
  .oxlintrc.json                oxlint 配置，来自 Plane（见 5.1、5.2）
  .oxfmtrc.json                 oxfmt 配置，来自 Plane（见 5.1、5.2）
  knip.jsonc                    knip 的配置，带中文注释（M0 只出报告，见 6.1、8）
  .node-version                 24
  .editorconfig
  .github/workflows/ci.yml      持续集成
  api/
    openapi.yaml                打包入口：info、tags，每个路径用 $ref 指向模块文件
    common.yaml                 公共组件：Problem、FieldError
    modules/instance.yaml       各模块的接口描述，一个模块一个文件
    redocly.yaml                打包配置
    dist/openapi.yaml           打包后的完整描述（生成物，提交到仓库）
  server/                       Go 模块 github.com/open-nerve/NerveProject/server
    cmd/nerve/                  命令行入口（serve、migrate、version）
    configs/                    config.yaml、config.dev.yaml、config.test.yaml、config.prod.yaml
    migrations/                 embed.go（内嵌 sql/ 目录）；sql/ 中存放迁移文件，M0 只有一个 .gitkeep
    internal/
      bootstrap/                组合根
      platform/
        buildinfo/              版本号、提交号、提交时间（版本号构建时注入，提交信息来自 Go 工具链嵌入的 VCS 信息）
        config/                 配置加载与校验
        logging/                slog 初始化
        postgres/               连接池、迁移执行器；pgtest（测试工具，只在测试中使用）
        httpserver/             服务生命周期、中间件、problem+json、健康检查；apigen/ 是 common.yaml 生成的公共类型；apitest/ 是契约校验工具，只被测试导入
        webui/                  内嵌的前端静态文件与单页应用的路由回退；dist/.gitkeep（构建产物由 .gitignore 挡掉）
      modules/
        instance/               试点模块
      archtest/                 架构测试（只有测试文件）
    tools/go.mod                开发工具的版本锁定
    .golangci.yml
  web/
    apps/web/                   来自 Plane 的前端应用（原样迁入）
    packages/                   types、constants、ui、propel、editor、i18n、hooks、utils、
                                shared-state、services、tailwind-config、typescript-config、
                                api-client（新增：由 OpenAPI 生成）
  e2e/                          Playwright 端到端测试（工作区包 @nerve/e2e）
    fixtures/                   db.ts、server.ts、api.ts（普通函数，全局准备也用它们）；test.ts（接成 Playwright 的 fixture）
    global-setup.ts             每次运行启动一次 Postgres 容器，迁移出模板库
    playwright.config.ts
    stories/smoke/              冒烟故事 S1–S4（见第 9 节；M0 没有业务领域，故事都放在 smoke 下）
  deploy/compose.dev.yaml       开发环境（Postgres 18）
  tools/plane-schema/           Plane 表结构快照工具和快照文件
  tools/lint-cap.mjs            前端各包的 oxlint 警告数与上限的核对（M1/P1 加入，见 5.2）
  tools/keywords.mjs            关键词守卫，规则在 tools/keywords.json（M1/P1 加入，见 M1 设计 7.4）
  docs/
```

和总体设计 2.2 相比，根目录另有 pnpm 工作区文件（`package.json`、`pnpm-workspace.yaml`、`pnpm-lock.yaml`、`turbo.json`）、`patches/`、oxlint 和 oxfmt 的配置文件，以及 `tools/` 目录；没有 `.npmrc`（pnpm 11 只从中读取认证和仓库地址，其余设置都不起作用，M0/P5 已实测核实，见 5.1）。
- **pnpm 工作区放在仓库根目录**：这样端到端测试可以直接用生成的 TS 客户端（`web/packages/api-client`）准备测试数据。
- **`tools/` 目录**：放不属于任何运行时组件的一次性工具。

---

## 3. 服务端骨架设计

### 3.1 包的职责
| 包 | 职责 | 可以依赖 |
|---|---|---|
| `cmd/nerve` | 解析命令行参数，调用 `bootstrap` | `bootstrap`、`platform/config`、`platform/buildinfo`、`server/configs`（内嵌的配置数据） |
| `internal/bootstrap` | **唯一的组合根**：创建各个适配器，接到各模块上，调用各模块的 `Register` 把生成的路由挂到根路由上，启动服务 | 所有 `platform` 包和 `modules` 包 |
| `internal/platform/*` | 与业务无关的技术基础件，各包之间互不依赖（`config` 除外，它可以被任何包使用） | 标准库和第三方库；**不能依赖 `modules`、`bootstrap` 和 `internal/shared`**（M2/P1：平台声明自己需要的小接口，`shared` 的类型按结构满足） |
| `internal/modules/<m>/domain` | 领域模型与规则 | 只能依赖标准库和 `internal/shared`（从 M2/P1 起 `identity/domain` 依赖 `shared`） |
| `internal/modules/<m>/app` | 用例；声明本模块需要的端口 | 只能依赖本模块的 `domain` 和 `internal/shared`；不能依赖 `platform` 或第三方技术库。archtest 检查这条规则 |
| `internal/modules/<m>/adapter/*` | 端口的实现（http、postgres 等）；http 适配器的包名为 `httpadapter`，避免遮住标准库 `net/http` | 本模块的 `app` 和 `domain`、`platform`、生成的代码 |
| `internal/modules/<m>` 下的 `module.go` | 模块的对外入口：`New(依赖)`；`(*Module).Register(router, api)` 把模块生成的路由挂到根路由上，`api`（`httpserver.API`）提供错误映射和按路由的中间件；`PublicOperations()` 列出不需要令牌的操作（M2/P1） | 本模块内部的各个包 |

`internal/shared`（共享内核）在第一次真正需要跨模块共享类型时才建立，M0 不建空包。M2/P1 建立它：`Actor`、领域错误 `Error`、`TxManager` 端口（M2 设计 3.3）。

### 3.2 试点模块 `instance`
- **接口**：`GET /api/v0/instance`，不需要登录。返回 `{ "product": "Nerve", "version": "0.1.0-dev", "commit": "…", "api_version": "v0" }`。
- **后续扩展**：M2 加入 `signup_enabled`、`workspace_creation_enabled`、`file_size_limit`（M2 设计 5.3）。前端启动时用它读取公开配置，Agent 可以用它确认服务端的版本。
- **目录结构**（这就是以后所有模块的模板）：
  ```
  modules/instance/
    domain/info.go            Info 值对象
    app/get_info.go           GetInfo 用例，依赖端口 InfoSource
    app/ports.go              InfoSource 接口（由使用方声明）
    adapter/buildinfo/        InfoSource 的实现：从 buildinfo 和配置中读取
    adapter/http/handler.go   实现 oapi-codegen 生成的强类型接口
    adapter/http/gen/         生成的代码（不要手改）
    module.go                 New(deps) → *Module{Handler}
  ```
- **为什么简单也要分层**：这个接口很简单，分层看起来有点"仪式感"。但它的作用是**模块模板**，用来把"生成接口 → 用例 → 端口 → 适配器 → 组合根接线"这条路走通，并让架构测试有东西可查。每一层都只有一个很小的文件。

### 3.3 HTTP 服务
- **路由**：Go 标准库的 `ServeMux`（Go 1.22 以后支持按方法和路径匹配）。M2/P1 起由 `httpserver.Router` 包一层，记下注册的每个模式，供整程序测试与接口描述核对（M2 设计 3.6）。挂载规则如下：
  - `/api/v0/…`：各模块生成的路由，由 `bootstrap` 逐个挂载。
  - `/healthz`：存活检查，不访问数据库。
  - `/readyz`：就绪检查，检查数据库能否连通、迁移是否已完成。
  - `/api/` 下没有匹配到的路径：返回 **404 problem+json**，不会回退到前端页面。`/api/` 兜底不区分方法（不带方法注册），所以 `/api/` 下"路径存在但方法不对"也得到 404，而不是 405。
  - 其余路径：交给 `webui`（前端静态文件和单页应用回退）。
- **中间件**（顺序固定）：
  1. 请求 ID：读取 `X-Request-Id`，只有是 1–128 个 `[A-Za-z0-9._:-]` 字符时才采用，否则生成 UUIDv7，并写回响应头。
  2. 异常恢复：捕获 panic，返回 500 problem+json，并记录日志。
  3. 访问日志：用 slog 记录方法、路径、状态码、耗时、请求 ID。`/healthz`、`/readyz` 的记录是 DEBUG 级别，其余是 INFO（M2/P1）。
  4. 安全响应头（M2/P2，M2 设计 8.3）：每个响应都带 `X-Content-Type-Options: nosniff`、`Referrer-Policy: same-origin`、`X-Frame-Options: DENY`；异常恢复清掉已设的响应头之后重新设上它们，panic 的 500 同样带着。
- **按路由的中间件**（M2/P1、P2，M2 设计 3.6）：`/api/v0` 的操作另有一串中间件，由 `httpserver.API.Middlewares` 交给每个模块的生成代码，在固定链之后、按这个顺序：请求元信息（客户端 IP 和限流用的 IP 键、UA）→ 请求期限（`server.request_timeout`）→ 请求体上限（`server.max_body_bytes`，超过是 413）→ 失败闸门和默认拒绝的认证（模块声明为公开的操作之外，没有有效令牌一律 401；带了令牌时先预留本 IP 的一个 `auth_failure` 单位，已空是 429）→ 限流（有凭证按凭证计数，没有按 IP 计数，超出是 429）→ 请求体结构检查（不合契约是 400）。生成代码在它们之前绑定路径参数和查询参数，在它们之后解码请求体。
- **`/readyz`**：按顺序执行各项检查，全部共用一个 2 秒的超时预算，遇到第一个失败就停止，返回通用的 `detail`（`<检查名> is not ready`）；具体错误只写进日志，不返回给客户端。
- **problem+json**：M0 定义统一的写出函数和 `Problem` 结构（与 `api/common.yaml` 中的定义一致），包含可选的 `detail`。平台自己的错误码不带模块前缀（`not_found`、`bad_request`、`internal_error`、`not_ready`）；M2/P1 加入 `unauthorized`、`payload_too_large`、`validation_failed`、`server_busy` 和领域错误码的体系（M2 设计 3.11），M2/P2 加入 `rate_limited`（429，带 `Retry-After`）。
- **生成代码的错误出口**（M0/P3，M2/P1 修改）：oapi-codegen 生成的代码在参数绑定、请求体解码、handler 返回错误三处默认输出纯文本；`httpserver.APIErrors` 把三处都接成 problem+json：参数绑定失败 → `BadRequest`（400 `bad_request`，`detail` 是失败原因）；请求体解码失败 → `BodyError`（400 `bad_request`，`detail` 是通用的一句话，不带出 Go 的类型名；超过请求体上限是 413 `payload_too_large`）；handler 出错或响应写出失败 → `Write`：满足 `httpserver.ProblemError` 的错误映射为它的状态、码、`detail` 和字段，其余是 500 `internal_error`（不带 `detail`，原因只进日志）；客户端断开（`context.Canceled`）不算 500；响应已经开始时改为记录日志并中断连接，不追加 problem。
- **非规范的 `/api/` 路径**：例如 `/api/v0//instance`、`/api/v0/./instance`、`/api`，Go 的 `ServeMux` 会先返回 307 跳转到规范路径，而不是直接落进平台的 404 兜底。M0/P3 评审后接受这个行为，不作特殊处理。
- **生命周期**：收到 SIGINT 或 SIGTERM 后停止接收新请求，在 `server.shutdown_timeout` 时间内处理完已有请求，然后退出。
- **连接上的读写都有上限**：读请求头（`server.read_header_timeout`）、读整个请求含请求体（`server.read_timeout`）、写响应（`server.write_timeout`），以及 2 分钟的空闲连接超时（包内常量）。只限制请求头不够：客户端发完请求头、声明了请求体却不发，服务端会一直等（M0 对抗性评审 Critical 2）。确实要更久的接口（M5 的文件上传、下载）在自己的 handler 里用 `http.ResponseController` 单独放宽，不调大全局值。这些上限管的是连接上的读写，不管 handler 自身的执行时间：`write_timeout` 到期只会让写出失败，既不停止 handler，也不取消它的 context；handler 里的数据库等阻塞调用要自带期限。
- 详见 [P2 spec](specs/P2-server-platform.md) 2.7 节。

### 3.4 内嵌前端（`platform/webui`）
- **打包方式**：`make build` 先构建前端，把 `web/apps/web/build/client/` 复制到 `server/internal/platform/webui/dist/`，再编译 Go 程序。`dist/` 只提交一个 `.gitkeep`，其余内容已加入 `.gitignore`。
- **没有构建前端时**：`dist/` 中没有 `index.html`，访问页面得到一句英文提示（程序输出的文字都用英文），建议执行 `make build`，开发时用 `make web-dev`；状态码 404，`Cache-Control: no-cache`。
- **只处理 GET、HEAD**：其他方法返回 405，带 `Allow: GET, HEAD`。
- **单页应用回退**（依次判断）：路径对应一个普通文件、且路径中没有以 `.` 开头的部分，就返回这个文件；`assets/` 或 `assets/` 下不存在的路径返回 404（避免脚本加载器把 HTML 当成 JS）；隐藏文件（如 `.gitkeep`）不提供；其余路径（包括 `/`、深层路径、目录）返回 `index.html`。
- **`/index.html`**：标准库 `http.ServeFileFS` 会把它 301 跳转到 `./`。
- **缓存策略**：`assets/` 下带哈希的资源文件设为 `Cache-Control: public, max-age=31536000, immutable`；`index.html` 和从 `public/` 复制来的文件（文件名不带哈希）都设为 `no-cache`。
- **路由注册**：必须注册成不带方法的 `/` 模式。`GET /` 会和 `/api/` 冲突，ServeMux 在注册时就会 panic。

### 3.5 数据库与迁移
- **连接**：`pgxpool`，参数来自配置（`database.url`、`database.max_conns`）。
- **迁移执行器**（`platform/postgres`）：用 goose Provider 执行 `server/migrations/sql/` 中的 SQL 文件。
- **迁移文件如何内嵌**：`server/migrations/embed.go` 用 `//go:embed sql/*.sql` 内嵌迁移文件。M0 没有迁移文件，当时写的是 `//go:embed all:sql`（`*.sql` 在一个文件都没有时会编译失败）；M2/P1 加入第一批迁移后改为现在的写法，删掉了占位的 `sql/.gitkeep`。
- **没有迁移文件时**：goose 会返回 `ErrNoMigrations`，迁移执行器把它当作"无事可做"，正常继续。
- **何时执行迁移**：
  - `nerve serve`：`database.auto_migrate=true` 时，启动前自动执行。dev 和 test 环境默认开启，prod 环境默认关闭。
  - `nerve migrate up | down | status`：手动执行。生产环境按"先迁移、再启动"的顺序操作。
- **迁移文件的命名**：`NNNNN_<模块>_<说明>.sql`（goose 按序号排序），文件名能看出归属哪个模块。
- **外键方向约定**：每个模块的迁移只建自己的表，以及指向更早建立的模块的外键。指向更晚建立的模块的外键（比如 `users.avatar_asset_id → file_assets`），由后建的那个模块的迁移用 `ALTER TABLE` 补上。
- **迁移文件从 M2 开始**：M0 不包含任何迁移文件，迁移机制（执行、回滚、查看状态、没有迁移文件时的处理）由集成测试验证，测试使用专门的测试迁移集，只存在于测试中。M2/P1 加入第一批迁移（`users`、`profiles`、`auth_sessions`），`server/migrations/schema_test.go` 核对它们能 up、down、再 up，以及约束名和 CHECK。
- **`goose_db_version` 的产生**：第一次对一个从未迁移过的库执行 `/readyz`，会顺带建出 goose 的 `goose_db_version` 表，这是 goose 自身的行为，无害。
- **不启用 goose 的会话锁**：v0 是单实例部署，暂不需要。
- **部分失败时**：某次迁移失败，`migrate up` 仍会先报出这之前已经执行成功的迁移，再报错。
- 详见 [P2 spec](specs/P2-server-platform.md) 2.5 节。

### 3.6 配置（落实总体设计 6.8）
- **M0 的配置项**：
  ```yaml
  server:
    addr: ":8080"
    read_header_timeout: 5s
    read_timeout: 30s     # 不小于 read_header_timeout
    write_timeout: 60s
    shutdown_timeout: 20s
  database:
    url: ""              # 必须提供
    max_conns: 10
    auto_migrate: true   # prod 中为 false
  log:
    level: info          # dev 中为 debug，test 中为 warn
    format: json         # dev 和 test 中为 text
  ```
- **dev 环境**：`server.addr` 是 `127.0.0.1:8080`（只监听本机）；数据库地址是本地开发库（`postgres://nerve:nerve@localhost:55432/nerve?sslmode=disable`）。这是只在本机使用的开发账号，可以提交。test 和 prod 的数据库地址都通过环境变量提供。
- **校验**：启动时逐项校验，有错误就退出，并指出是哪个配置项出了问题。启动日志打印生效的配置，其中 `database.url` 整体打码：pgx 用自己的 libpq 兼容语法解析这个地址，别的解析器只能认出其中一部分密码写法，只遮认出的部分会漏掉其余写法。连接目标（主机、端口、库名、用户）在创建连接池后按 pgx 的解析结果单独记一条日志，其中没有密码。
- 未知的配置键直接报错，不会悄悄回落到默认值。
- 时长类配置项只接受字符串（例如 `5s`），不接受纯数字。
- 只有包含 `__` 的 `NERVE_*` 变量才是配置键；`NERVE_ENV`、`NERVE_CONFIG_DIR` 和换开发库端口用的 `NERVE_DEV_DB_PORT` 都不含 `__`，不当作配置键。
- `NERVE_CONFIG_DIR` 是内置配置文件之上的一层，按键覆盖，不是整体替换。
- `config.local.yaml` 相对当前工作目录解析；`make run` 在 `server/` 目录下运行。
- 详见 [P2 spec](specs/P2-server-platform.md) 2.3 节。

### 3.7 架构守护
- **架构测试**（`internal/archtest`，写法类似 Java 的 ArchUnit）：用 `golang.org/x/tools/go/packages` 读取所有包的导入关系，一共 10 条规则，包括：
  1. 模块内的依赖只能向内：`adapter → app → domain`。
  2. `domain` 和 `app` 都只能依赖标准库、本模块的内层包和 `internal/shared`，不能依赖 `platform`、数据库驱动、HTTP 等技术库。
  3. 模块之间不能互相导入。
  4. `platform` 不能导入 `modules` 和 `bootstrap`。
  5. 只有 `bootstrap` 能导入各个模块。
  6. 生成的代码只能被本模块的 http 适配器导入。
  7. `platform` 的各个包之间互不导入（`config` 除外）。
  8. 测试工具（`pgtest`、`apitest`）只能被测试代码导入。
  9. 模块内的包只能放在 `domain`、`app`、`adapter`（含子目录）或模块根目录（`module.go`）。放在别处的包，第 1 条排不出它的层次，第 2 条又把模块内的导入交给第 1 条判断，`domain → 模块内其他目录 → net/http` 就能两条都绕过（M0 对抗性评审 Important 1）。
  10. `internal/shared` 只能依赖标准库（不含 `net/http`、`database/sql`）和它自己：`domain`、`app` 可以导入它，它不干净，技术依赖就会经它带进这两层。M0 还没有 `internal/shared`，这条规则先用合成的导入关系测试。
- **传递依赖测试**（`TestNerveBinaryLinksNoBannedModule`，M0/P3；M2/P3a 改写）：规则 8 只挡住测试工具包本身，挡不住生成的代码或其他途径间接引入的依赖（例如内嵌的接口描述），depguard 也不检查生成的文件。这个测试用 `golang.org/x/tools/go/packages` 读取 `./cmd/nerve` 的全部传递依赖（不含测试），出现 `github.com/getkin/kin-openapi`、`github.com/testcontainers/`、`github.com/docker/` 开头的包就失败，并打印导入链。`github.com/google/uuid` 只允许由 `github.com/oapi-codegen/runtime` 模块的包导入（它的参数绑定自己用），别的包导入它同样失败。
- **生成代码的 uuid 规则**（`TestGeneratedCodeUsesTheStandardUUID`，M2/P3a）：禁止 google/uuid 的真实意图是"生成代码不漏 uuid 的映射"，所以直接检查：生成文件不得引用 `oapi-codegen/runtime/types` 的 `UUID`（不论导入时用什么名字；生成代码默认叫它 `openapi_types`）（M2 设计 3.12）。
- **纯净性的传递检查**（`TestPureLayersReachNoInfrastructure`，M0 加固）：第 2、10 条只看直接导入，而标准库里的 `expvar`、`net/rpc` 自己就导入 `net/http`，`domain → expvar` 能过这两条，却把 `net/http` 链接了进来。这个测试沿全部传递依赖检查每个 `domain`、`app`、`internal/shared` 包：不能碰到 `net/http`、`database/sql` 或本模块以外的库。发现时打印完整的导入链。
- **depguard**：只管"整个项目都禁止使用的库"，例如：
  - 第三方 uuid 库（`github.com/google/uuid`、`github.com/gofrs/uuid`、`github.com/satori/go.uuid`）：用标准库。
  - viper：用 koanf。
  - 标准库 `log`（精确匹配 `log$`，因为前缀匹配的 `log` 会连 `log/slog` 一起禁掉）：用 slog。
  - `github.com/pkg/errors`：用标准库的 errors。
- golangci-lint 还启用 `gochecknoinits`，禁止 `init()`。
- 以上都是持续集成的门禁。架构测试随 `go test ./...` 一起运行，不需要额外安装工具。
- 详见 [P2 spec](specs/P2-server-platform.md) 2.10 节。

### 3.8 测试工具
- **`platform/postgres/pgtest`**：
  1. 用 testcontainers 启动一个 Postgres 18 容器，每个包的测试进程共用一个（Go 为每个包单独启动一个测试进程，容器不跨进程共用）。
  2. 执行迁移，得到一个模板库。
  3. 每个测试用 `CREATE DATABASE … TEMPLATE` 复制出自己独立的数据库，测试结束后删除；`NewEmptyDatabase` 给出一个没有执行过任何迁移的库。
  4. 容器由 testcontainers 的回收容器（Ryuk）清理，所以不能设置 `TESTCONTAINERS_RYUK_DISABLED`。
- `go test -short` 跳过集成测试，没有 Docker 时也能跑单元测试。
- **接口契约校验**（`platform/httpserver/apitest`，M0/P3）：只被测试导入，读取并校验 `api/dist/openapi.yaml`，提供 `CheckResponse`、`CheckSchema` 给各处的契约测试使用：平台写出的每一种 problem、试点模块 `instance` 的 handler、`bootstrap` 接线后的整个程序，都用它校验响应是否符合契约。
- **`make test` 不用测试缓存**（`-count=1`）：Go 的测试缓存不跟踪 `server/` 模块根目录之外的文件，只改了 `api/` 时缓存的 `go test` 会重放旧结果，所以 `make test`（持续集成也调用它）总是加 `-count=1`。

---

## 4. 接口契约流水线

```
api/common.yaml + api/modules/*.yaml
      │
      ├─(oapi-codegen，每个模块单独生成)─→ server/internal/modules/<m>/adapter/http/gen/
      │     common.yaml 中的公共组件通过 import-mapping 生成到一个共享包里，
      │     放在 platform/httpserver/apigen，避免每个模块各生成一份
      │
      └─(redocly 打包)─→ api/dist/openapi.yaml ─(openapi-typescript)─→ web/packages/api-client
                               └→ 服务端测试中的契约校验；M8 的对外接口文档
```
- **按模块拆分描述文件**：每个模块的接口描述由该模块自己负责，生成的 Go 代码也放在该模块内部，符合"模块自己的东西自己管"的原则。
- **OpenAPI 版本：已定为 3.1**（`openapi: 3.1.0`，M0/P3 验证通过）。验证方法、结论和证据见 [P3 spec](specs/P3-api-contract.md) 2.2；接口描述的写法约定（绕开验证中发现的 oapi-codegen 不足）见同一份 spec 2.4。
- **生成物的管理**：生成的代码提交到仓库，这样不装生成工具也能直接编译。持续集成中会重新生成一遍，用 `git status --porcelain` 检查生成物和描述文件是否一致（不用 `git diff --exit-code`：新模块尚未提交的生成文件是未跟踪状态，`git diff` 看不到）。
- **TS 客户端**：`web/packages/api-client` 导出生成的类型，以及一个用 openapi-fetch 创建客户端的函数。M0 里只有端到端测试使用它，前端应用从 M2 开始使用。

---

## 5. 前端迁入与 Plane 表结构快照

### 5.1 前端迁入（原样迁入，不做功能改动）
- **复制的内容**：`plane/apps/web` 复制到 `web/apps/web`；web 用到的 packages（见总体设计 7.1）复制到 `web/packages/`；`.oxlintrc.json`、`.oxfmtrc.json` 复制到仓库根目录；`patches/react-color@2.19.3.patch` 复制到根目录的 `patches/`（web 依赖 react-color，锁文件记录这个补丁的哈希）。**不复制 `.npmrc`**：pnpm 11 只从 `.npmrc` 读取认证、仓库地址和网络设置，Plane 写在里面的其他设置都不起作用（M0/P5 实测核实）。
- **锁文件**：以 Plane 的 `pnpm-lock.yaml` 为起点生成——把 importers 的路径移到 `web/` 下，再执行一次 `pnpm install`，保证迁入的包的依赖版本与 Plane 完全相同。
- **`turbo.json` 和依赖版本表（catalog、overrides、allowBuilds 等）的处理**：只保留作用于迁入的包的条目，以锁文件为证据——修剪前后，13 个迁入的包的解析结果不变。
- **M0 只做让前端能跑起来的最小改动**：
  - 工作区的路径。
  - 开发服务器的代理：Vite 把 `/api` 转发到 `http://127.0.0.1:8080`。
- 功能删减、品牌替换、改用 React Router 原生写法、包名改为 `@nerve/*`，**全部留到 M1**。这样 M0 引入的改动和 M1 的删减可以分开审查。
- **验收**：`pnpm install` 成功，类型检查通过，lint 检查通过（按下面的警告基线），前端能构建。
- **版权**：所有来自 Plane 的文件保留原有的版权声明。

### 5.2 lint 警告基线：只降不升
- **现状**：Plane 本身就给每个包设了"最多允许多少条警告"的上限，但这些上限远高于实际警告数（比如 web 的上限是 11957 条，实测只有 779 条）。
- **做法**：上限改为实测的警告数，并定下一条规则：**上限只能往下调，不能往上调**。M0/P5 实测的上限（在根目录的 `.oxlintrc.json` 下测量）：web 779、editor 75、propel 59、utils 34、ui 32、services 6、hooks 4、i18n 3、constants 2、types 1、shared-state 0、api-client 0（新增脚本），合计 995 条（原来误写为 1005，M1 设计时更正）。
  - 任何提交如果让警告数超过上限，持续集成就失败。
  - 警告数下降后，要在同一个提交里把上限调低到新的数值。
- **自动核对（M1/P1 起）**：每个包的 `check:lint` 是 `node <到仓库根目录的相对路径>/tools/lint-cap.mjs <上限>`，警告数必须**等于**上限，多了或少了 `make lint-web` 都失败；少了时提示应调低到的数值，所以上限总是实测值，不需要单独重新测量。M1/P1 之后的上限：web 777、editor 75、utils 34、ui 31、propel 29、hooks 4、constants 2、types 1、i18n 1，其余为 0，合计 954 条（[M1/P1 spec](../M1-frontend-trim/specs/P1-web-hygiene.md) 2.3）。逐步清零的计划见 [M1 设计](../M1-frontend-trim/M1-design.md) 7.3。
- 这一条是对总体设计 7.6"oxlint 警告即报错"的修正，见 7.3。

### 5.3 Plane 表结构快照（`tools/plane-schema/`）
- **`extract.sh` 做什么**：
  1. 用 docker compose 启动 Postgres 15.7 和 Plane v1.4.2 的后端镜像，两个镜像都写成"标签@摘要"（`tag@digest`），保证重新运行拿到同样的输入。
  2. `up --detach --wait db` 启动 Postgres 并等到健康检查通过；`run -T migrator` 在后端镜像里执行 `python manage.py migrate --no-input`，`REDIS_URL` 设为一个不存在的主机（一个非空的虚拟值：迁移过程只需要数据库，从未真正连接 Redis）。
  3. 在 `db` 容器内部执行 `pg_dump --schema-only --no-owner --no-privileges`，导出表结构，生成 `plane-v1.4.2-schema.sql`。
- **快照文件提交到仓库**：这是之后每个 M 建表时"照搬 Plane"的依据。建表的 M 以快照为起点，再按[差异清单](../plane-diff.md)中的规则修改。
- **不过滤**：快照是 `pg_dump` 的原样输出，未保留的表和 Django、Celery Beat 的系统表都留在里面；保留哪些表属于差异清单会随设计调整，工具只负责如实导出。
- **README 写明**：Plane 的版本、提交号、两个镜像的"标签@摘要"、生成时间、内容概要，以及如何重新生成。**生成时间**指快照内容最后一次变化的日期：快照本身不含时间戳，重新生成不会改变它。
- **官方镜像已验证可用**；恢复办法：在对应标签上用 `apps/api/Dockerfile.api` 构建（未验证）。
- 详见 [P4 spec](specs/P4-plane-schema.md)。

---

## 6. 开发体验与持续集成

### 6.1 Makefile 命令
| 命令 | 作用 |
|---|---|
| `make dev-db` / `make dev-db-down` | 启动或停止开发数据库（`deploy/compose.dev.yaml`） |
| `make run` | 以 dev 配置运行后端（`go run ./cmd/nerve serve`） |
| `make web-dev` | 启动前端开发服务器（Vite，把 `/api` 转发给后端），同时监视 `web/packages/*`（M1/P1） |
| `make gen` | 重新生成所有代码：Go 接口层、打包后的 OpenAPI 描述、TS 客户端 |
| `make gen-check` | 重新生成，并检查生成物是否已提交且没有差异 |
| `make lint` | golangci-lint；关键词守卫，前端的类型检查、oxlint（警告数等于上限）、格式检查、中英文翻译键一致性，`tools/` 下脚本的 lint 和格式检查（守卫和后三项的改动来自 M1/P1） |
| `make test` | Go 单元测试、集成测试、架构测试。需要 Docker（集成测试用 testcontainers） |
| `make test-web` | 前端单元测试（各包的 vitest，经 turbo；需要 Node；M1/P1 加入） |
| `make build-web` | 构建前端，产物在 `web/apps/web/build/client/`（只需要 Node） |
| `make build` | 依赖 `build-web`；把构建产物嵌入 Go 程序，编译出 `bin/nerve` |
| `make e2e` | 构建产物并运行端到端测试；需要 Docker，浏览器要先单独安装一次（README）；`VERSION` 写进构建产物，`NERVE_VERSION` 把同一个值交给测试核对 |
| `make knip` | 报告未使用的文件、导出和依赖（需要 Node；M0 只出报告，M1 起作为门禁） |
| `make plane-schema` | 重新生成 Plane 表结构快照（需要 Docker，Compose 2.22 或更高，见 [P4 spec](specs/P4-plane-schema.md)） |

`gen`、`gen-check`、`lint` 按区域拆分出 `-go`（只需要 Go）和 `-web`（需要 Node，先执行 `pnpm install`）两个后缀（`gen-go`/`gen-web`、`gen-check-go`/`gen-check-web`、`lint-go`/`lint-web`）；不带后缀的命令依次执行两个区域，供本地使用（M0/P3，见 [P3 spec](specs/P3-api-contract.md) 2.10）。`test-web` 是前端单独的入口；`make test` 仍只运行 Go 测试，持续集成的 `server` 任务调用它，不需要 Node（M1/P1）。

### 6.2 开发流程
1. 执行 `make dev-db`，启动本地的 Postgres 18。
2. 执行 `make run`，后端监听 `:8080`；dev 环境下会自动执行迁移。
3. 执行 `make web-dev`，前端运行在 http://127.0.0.1:3000 ，接口请求会被转发到后端。它同时监视 `web/packages/*`：改了某个包的代码，这个包的 `dev` 任务（tsdown）重新构建，页面随之更新（M1/P1；M0 中只运行 web 自己的开发服务器）。

### 6.3 持续集成（`.github/workflows/ci.yml`）
| 任务 | 内容 |
|---|---|
| `server` | 安装 Go 1.27.1 → `make gen-check-go` → `make lint-go`（锁定版本的 golangci-lint）→ `make test`（包含集成测试和架构测试；GitHub 提供的 Linux 运行环境自带 Docker） |
| `web` | `corepack enable` → 缓存 pnpm 存储（按锁文件的哈希）→ `pnpm install --frozen-lockfile` → `make gen-check-web` → `make lint-web`（关键词守卫、类型检查、oxlint 警告数等于上限、格式检查、中英文翻译键一致性、`tools/` 下脚本的 lint 和格式检查；守卫和后三项的改动来自 M1/P1）→ `make test-web`（前端单元测试，M1/P1 加入）→ `make knip`（未使用代码报告，M0/P6 加入）→ `make build-web`（M0/P5 加入） |
| `e2e` | 在 `server` 和 `web` 通过后运行：安装 Playwright 的 Chromium Headless Shell（`--with-deps`）→ `make build` → `make e2e`；`VERSION=0.0.0-ci.<运行编号>`；超时 20 分钟；失败或被取消时上传 `playwright-report`、`test-results`（P6 加入） |

持续集成不运行 Plane 表结构快照的提取（`make plane-schema`）：两个输入都按摘要写死，快照不会自己变化，见 [P4 spec](specs/P4-plane-schema.md) 2.8。

触发条件：每次推送代码和每个 PR；**同仓库分支的 PR 跳过 `server` 和 `web`**（M0/P3）：两个任务都加了 `if: github.event_name == 'push' || github.event.pull_request.head.repo.full_name != github.repository`。同仓库分支的每个提交已经由 push 事件跑过，PR 页面显示的就是这次的结果；来自 fork 的 PR 没有对应的 push 事件，仍然由 `pull_request` 事件运行。P6 的 `e2e` 任务用 `needs: [server, web]` 依赖这两个任务，这个跳过条件也会经 `needs` 传导过去。

**必须通过检查（required checks）前的注意事项**：GitHub 把被 `if` 跳过的任务也标记为 Success，且用的是同一个检查名字。把 `server`/`web`/`e2e` 设为分支保护的必须检查之前，要先去掉上面的跳过条件，或者另加一个 `if: always()` 的汇总任务，把这个汇总任务设为必须检查，否则一次被跳过的运行就能满足必须检查的要求。汇总任务只解决"跳过算通过"的问题：push 事件测的是分支头，不是 PR 合并进 `main` 之后的结果。保留跳过条件时，还要在分支保护里打开"合并前分支必须与 `main` 同步"（require branches to be up to date），否则 `main` 前进之后，分支头通过不代表合并结果能通过（M0 对抗性评审 K16）。

---

## 7. 对总体设计的调整（已确认，已同步）

写 M0 设计时，发现总体设计中有几处需要修正。以下 5 项已于 2026-09-22 确认，并已同步到 `v0-design.md` 和 `plane-diff.md`。

### 7.1 业务表由各个 M 自己建，M0 不一次性建全部表
- **原来的写法**：M0 生成包含全部 45 张表的 `0001_init.sql`。
- **建议改为**：
  - M0 只提供 Plane 表结构快照（5.3）和迁移机制（3.5）。
  - 每个 M 在实现自己的功能时，以快照为起点，为自己的模块写迁移文件。
- **理由**：
  1. 很多列级别的决定，要到对应的 M 才有足够的信息做出，比如 `auth_sessions` 的结构、筛选条件怎么存、草稿的 `payload`。一次性建全部表会把这些决定提前，之后还得再写迁移来改。
  2. 符合"不写用不上的代码"的原则：M0 建的表在 M2 到 M8 之前都没有代码使用。
  3. 迁移文件的归属更清楚：谁建的表谁负责。
- **影响**：差异清单里"列级别的细节在 M0 补全"改为"由建表的 M 补全"。

### 7.2 River 和 sqlc 推迟到第一次真正使用时（M2）再接入
- **理由**：M0 没有任何后台任务，也没有任何 SQL 查询。现在接入 River，只会得到一个"启动了但什么都不做"的任务客户端；现在接入 sqlc，也没有查询可以生成代码。两者都属于用不上的代码。
- **M2 的情况**：M2 会有第一个定时任务（清理过期会话），也会有第一批查询（账户和会话），届时和真实的用法一起接入并测试。
- **对 M0 的影响**：M0 的依赖列表里不再包含 River 和 sqlc；它们的版本已经在第 1 节核实过了。

### 7.3 前端 lint 采用"警告基线只降不升"
- **原来的写法**：总体设计 7.6 写的是"oxlint 警告即报错"。
- **问题**：Plane 现有的代码带着上万条警告，这个要求在迁入时做不到。
- **建议改为**："警告数不得超过基线，基线只能调低"，并在 M1 制定逐步清零的计划（见 5.2）。

### 7.4 架构规则主要由架构测试来检查
- **原来的写法**：总体设计 6.3 写的是"用 depguard（必要时加 go-arch-lint）检查依赖方向和模块边界"。
- **建议改为**：
  - 模块边界和分层方向，用仓库内的架构测试来检查（3.7）。规则写成代码，没有额外的工具依赖，也容易扩展。
  - depguard 只负责"禁止使用的库"。
  - 不引入 go-arch-lint。它虽然还在维护，但主要靠社区贡献，而我们的规则用架构测试更容易准确表达。

### 7.5 版本相关的补充
- **UUIDv7 由 Go 1.27 的标准库生成**，数据库不设默认值。
- **v0 期间 TypeScript 保持 5.8**。
- **OpenAPI 的版本**：按第 4 节的方法验证，已定为 3.1（M0/P3）。

---

## 8. Phase 划分与实施规划

所有 Phase 按顺序依次推进。每个 Phase 都要依次产出 spec、plan、实现和 review，文件放在对应的子目录中。

### P1 `repo-toolchain`：仓库与工具链
- **交付物**：
  - 根目录的 `package.json`、`pnpm-workspace.yaml`（先只包含工作区的骨架）、`.node-version`、`.editorconfig`。
  - `server/go.mod`（锁定 `toolchain go1.27.1`）和 `server/tools/go.mod`。
  - Makefile 的骨架。
  - `deploy/compose.dev.yaml`（Postgres 18）。
  - `.github/workflows/ci.yml` 的骨架：几个任务都能跑通，此时还没有实际内容。
  - `.gitignore` 的补充项。
  - `platform/buildinfo`（从 P2 提前：没有任何 Go 代码时 `go test` 和 golangci-lint 都会失败，详见 P1 spec）。
- **验收**：
  - 在一台干净的机器上，只装了 Docker、Go 和 Node，执行 `make dev-db` 能启动数据库。
  - `go tool -modfile=tools/go.mod oapi-codegen -version` 能运行。
  - 持续集成通过。
- **风险**：sqlc 需要 cgo（它依赖 pg_query 的 C 代码）。M0 不使用 sqlc，但这个风险要在 P1 记录下来，M2 接入时再验证。

### P2 `server-platform`：服务端平台层
- **交付物**：
  - `platform/config`：分环境加载配置、环境变量覆盖、配置文件内嵌进程序、启动时校验。
  - `platform/logging`。
  - `platform/postgres`：连接池、goose 迁移执行器、测试工具 `pgtest`。
  - `platform/httpserver`：`ServeMux`、三个中间件、problem+json、`/healthz` 和 `/readyz`、优雅停机。
  - `bootstrap`。
  - `cmd/nerve`：`serve`、`migrate up|down|status`、`version` 三个命令。
  - `archtest` 和 `.golangci.yml`（包含 depguard 规则）。
- **验收**：
  - 单元测试：配置的加载顺序和覆盖规则、配置校验、中间件、problem+json。
  - 集成测试：迁移执行器（up、down、status）、`/readyz` 在数据库不可用时返回 503。
  - 架构测试通过。故意写一个违规的导入，架构测试必须报错（验证后删除这个违规）。
  - 持续集成通过。

### P3 `api-contract`：接口契约流水线与试点模块
- **交付物**：
  - OpenAPI 3.1 的链路验证，结论写进 P3 的 review。
  - `api/` 目录结构和 Redocly 打包配置。
  - 为每个模块单独运行 oapi-codegen 的生成配置，以及公共组件的共享生成包。
  - `modules/instance` 的完整实现，并在 `bootstrap` 中接线。
  - `web/packages/api-client`。
  - `make gen` 和 `make gen-check`。
- **验收**：
  - `GET /api/v0/instance` 的 handler 测试通过，响应通过契约校验。
  - `GET /api/v0/不存在的路径` 返回 404 problem+json（P2 已在平台层实现并测试，P3 挂上模块后再验证一次）。
  - `make gen-check` 在持续集成中生效：故意修改描述文件但不重新生成，持续集成必须失败。

### P4 `plane-schema`：Plane 表结构快照
- **交付物**：`tools/plane-schema/` 下的 `compose.yaml`、提取脚本、说明文档、快照文件 `plane-v1.4.2-schema.sql`；`make plane-schema`。
- **验收**：
  - 重新运行脚本，得到的快照与提交的版本一致。
  - 快照中能找到总体设计 5.2 列出的全部 44 张 Plane 表。
  - 差异清单中的相关说明已更新。

### P5 `web-import`：前端迁入与内嵌
- **交付物**：
  - `web/`（原样迁入）。
  - 根目录的工作区和 turbo 配置。
  - Vite 开发服务器的代理配置。
  - `platform/webui`。
  - `make build` 能构建出内嵌前端的 `bin/nerve`。
  - 在前端改动清单中登记迁入时做的最小改动。
- **验收**：
  - `pnpm install`、类型检查、按基线的 lint 检查、构建全部通过。
  - `bin/nerve serve` 能打开前端首页和任意深层路径，静态资源全部加载成功；页面内容是 Plane 的"didn't start up correctly"（前端调用的 `/api/instances/` 在 Nerve 中还不存在），这是预期的，直到 M2 对接新接口才会改变。
  - 持续集成通过。

### P6 `e2e-ci`：端到端测试骨架与冒烟故事
- **交付物**：
  - `e2e/` 下的 Playwright 项目。
  - fixtures：`server.ts`（每个 worker 启动一个 `nerve`）、`db.ts`（Node 端用 testcontainers 启动 Postgres，从模板库复制出每个 worker 的数据库）、`api.ts`（生成的 TS 客户端）、`test.ts`（把以上三个接成 Playwright 的 fixture，故事从这里导入 `test`、`expect`）；`global-setup.ts`（每次运行启动一次 Postgres 容器、迁移出模板库）。
  - 第 9 节中的冒烟故事。
  - 持续集成中的 `e2e` 任务。
  - knip 的配置：M0 只出报告，M1 开始作为门禁。
- **验收**：本地执行 `make e2e` 和持续集成中的 `e2e` 任务，全部通过。

---

## 9. 用户故事（M0 冒烟）

M0 没有业务功能，所以这里的故事只验证"系统能启动、能访问、行为正确"。每个故事都按总体设计 8.2 的要求编写。

| 编号 | 故事 | 页面 / 接口断言 | 数据库断言 |
|---|---|---|---|
| S1 | 运维人员用 test 配置启动 nerve，服务就绪 | `/healthz` 返回 200；`/readyz` 返回 200；`nerve migrate status` 能正常执行，M0 没有迁移文件时输出 `no migrations` | `pg_stat_activity` 中能看到 `application_name = 'nerve'` 的连接，确认服务连上了为本 worker 准备的数据库（fixture 把 `application_name=nerve` 写进交给 nerve 的数据库地址）。从 M2 起加上"迁移版本正确"的断言 |
| S2 | 用户在浏览器中打开首页 | 返回的是前端页面；按路径把请求分类为静态资源（`/api/` 以外）和接口，静态资源全部加载成功；页面的所有请求都发往 nerve 自身（同源）；直接打开一个深层路径，同样返回前端页面，内容与首页逐字节相同 | — |
| S3 | 调用方查询实例信息 | `GET /api/v0/instance` 返回 200，`api_version` 为 `v0`，`version` 与构建时注入的版本号一致，`commit` 是 40 位十六进制；用生成的 TS 客户端调用，类型检查通过 | — |
| S4 | 调用方访问不存在的接口 | 返回 404，`Content-Type` 为 `application/problem+json`，而不是前端页面 | — |

M0 还没有认证，所以不涉及 PAT 对等验收。从 M2 开始，每个故事都要有 PAT 版本。

---

## 10. 完成标准

- [x] P1 到 P6 全部完成，每个 Phase 都有 spec、plan 和 review。
- [x] 持续集成中的全部门禁通过：生成物一致性检查、golangci-lint（含 depguard）、Go 测试（含架构测试和集成测试）、前端类型检查、oxlint（按基线）、前端构建、端到端冒烟故事。
- [x] `make build` 能构建出单个可执行文件 `bin/nerve`；它加上一个 Postgres，就能完成 S1 到 S4。
- [x] 第 7 节的调整建议已确认，并已同步更新到总体设计和差异清单。
- [x] 前端改动清单中已登记迁入时的改动。
- [x] `handoffs/` 中没有 `open` 状态的事项；需要移交给后续 M 的事项，已放进对应 M 的 `handoffs/` 目录。
- [x] 总体设计中 M0 的状态改为"已完成"。

---

## 11. Phase 进度表

| Phase | 名称 | 状态 | spec | plan | review |
|---|---|---|---|---|---|
| P1 | repo-toolchain | 已完成 | [spec](specs/P1-repo-toolchain.md) | [plan](plans/P1-repo-toolchain.md) | [review](reviews/P1-repo-toolchain-review.md) |
| P2 | server-platform | 已完成 | [spec](specs/P2-server-platform.md) | [plan](plans/P2-server-platform.md) | [review](reviews/P2-server-platform-review.md) |
| P3 | api-contract | 已完成 | [spec](specs/P3-api-contract.md) | [plan](plans/P3-api-contract.md) | [review](reviews/P3-api-contract-review.md) |
| P4 | plane-schema | 已完成 | [spec](specs/P4-plane-schema.md) | [plan](plans/P4-plane-schema.md) | [review](reviews/P4-plane-schema-review.md) |
| P5 | web-import | 已完成 | [spec](specs/P5-web-import.md) | [plan](plans/P5-web-import.md) | [review](reviews/P5-web-import-review.md) |
| P6 | e2e-ci | 已完成 | [spec](specs/P6-e2e-ci.md) | [plan](plans/P6-e2e-ci.md) | [review](reviews/P6-e2e-ci-review.md) |

---

## 12. 风险

| 风险 | 应对 |
|---|---|
| oapi-codegen 和 kin-openapi 对 OpenAPI 3.1 的支持还比较新 | **已由 M0/P3 验证并解除**：3.1 在整条链路上可用（[P3 spec](specs/P3-api-contract.md) 2.2），不需要改用 3.0.3 |
| 按模块拆分描述文件后，oapi-codegen 的跨文件引用（`import-mapping`）表现不符合预期 | **已由 M0/P3 验证并解除**：表现符合预期，公共组件通过 import-mapping 生成到共享包 `apigen`，不需要退回"单个描述文件 + `include-tags`"的备选方案 |
| Plane 后端镜像无法获取，或者在当前环境中跑不起来 | **已由 M0/P4 验证并解除**：官方镜像可用；恢复办法：在对应标签上用 `apps/api/Dockerfile.api` 构建（未验证） |
| Plane 前端的依赖很多，构建比较慢，会拖慢持续集成 | **已由 M0/P5 验证并解除**：冷缓存的 `web` 任务耗时 142 秒，远低于 15 分钟的超时；pnpm 存储按锁文件的哈希缓存；turbo 缓存只在任务内部使用（`lint-web` 构建的包被 `build-web` 复用），不跨运行保存 |
| River 仍是 0.x 版本，小版本之间可能有行为变化 | M2 接入时锁定具体的版本号 |
| 持续集成拉取 Postgres 镜像（`postgres:18.6`）和 testcontainers 的回收镜像（`testcontainers/ryuk:0.14.0`）受 Docker Hub 匿名拉取频率限制 | 出现限流时，在持续集成中登录 Docker Hub 或改用镜像缓存；M8 之前没有负责人跟进（M0/P6 交给 M8 的 handoff） |
| 每个包一个测试容器，模块增多后持续集成变慢 | M2 之后评估 testcontainers 的跨进程复用，或限制 `go test -p` |
| 持续集成的 `e2e` 任务比估计的慢（前端在这个任务里再构建一次、安装系统库、拉取镜像） | **已由 M0/P6 验证并解除**：分支冷运行（`35702932216`）`e2e` 任务 120 秒，分支热运行（`35703813772`）122 秒，都远低于 20 分钟的超时；整条流水线（`e2e` 在 `web` 完成后才开始）冷运行约 270 秒、热运行约 233 秒（P6 spec 2.9） |
