# Nerve

一个轻量的多人协作项目管理系统，页面和接口能力完全对等：同一个账户，既可以由人在页面上使用，也可以由 Agent 通过接口使用。

- 后端：Go（单个可执行文件）+ PostgreSQL
- 前端：基于 [Plane](https://github.com/makeplane/plane) 前端分叉并裁剪
- 协议：[AGPL-3.0](LICENSE)

项目处于 v0 开发阶段，从 [docs/](docs/README.md) 开始阅读。

## 开发环境

需要安装：
- Docker（含 Compose v2；`make plane-schema` 需要 Compose 2.22 或更高）
- Go 1.26 或更高。第一次在 `server/` 下执行 Go 命令时，会自动下载 `server/go.mod` 指定的 Go 1.27.1（前提是 `GOTOOLCHAIN=auto`，这是 Go 官方安装包的默认值；部分 Linux 发行版自带的 Go 默认是 `local`，需要先执行 `go env -w GOTOOLCHAIN=auto`）
- Node.js 24，并执行一次 `corepack enable`（pnpm 的版本由 `package.json` 锁定）。代码生成，以及前端的检查、开发和构建需要它

第一次启动：

```bash
pnpm install  # 安装 Node 依赖（代码生成和前端检查要用）
make dev-db   # 启动本地 Postgres 18（端口 55432，可用 NERVE_DEV_DB_PORT 修改）
make test     # 运行测试（需要 Docker：集成测试用 testcontainers 启动 Postgres）
make run      # 以 dev 配置运行后端，监听 127.0.0.1:8080；Ctrl-C 停止
make          # 查看所有命令
```

- 只跑单元测试、跳过需要 Docker 的集成测试：在 `server/` 下执行 `go test -short ./...`。
- `make test` 不用 Go 的测试缓存：缓存不跟踪 `server/` 之外的文件，契约测试读取的 `api/` 改了也会重放旧结果。直接执行 `go test` 时，改了 `api/` 要加 `-count=1`。
- `server/tools` 是独立的 Go 模块（代码生成工具），`server/` 下的 `go test ./...` 不进入它；`make test` 和 `make lint-go` 都另外在它里面跑一遍。
- 后端的配置文件在 `server/configs/`，任何一项都可以用环境变量覆盖：`NERVE_` 加上配置路径，层级之间用双下划线，例如 `database.url` 对应 `NERVE_DATABASE__URL`。个人的本地覆盖写在 `server/configs/config.local.yaml`（不进仓库，只在 dev 环境生效）。
- **换了开发库的端口时**：`NERVE_DEV_DB_PORT` 只改变 compose 映射的端口，`server/configs/config.dev.yaml` 中的数据库地址仍然是 55432。还需要覆盖 `database.url`，例如：

  ```bash
  NERVE_DEV_DB_PORT=55433 make dev-db
  NERVE_DATABASE__URL=postgres://nerve:nerve@localhost:55433/nerve?sslmode=disable make run
  ```

  或者在 `server/configs/config.local.yaml` 中写：

  ```yaml
  database:
    url: postgres://nerve:nerve@localhost:55433/nerve?sslmode=disable
  ```

## 接口与代码生成

接口先写描述（OpenAPI 3.1），再写实现：

- `api/openapi.yaml` 是入口，列出所有路径；各模块的描述在 `api/modules/<模块>.yaml`，公共组件在 `api/common.yaml`。
- 改完描述后执行 `make gen`，把生成的文件和描述一起提交：
  - `make gen-go`：只需要 Go。生成 Go 接口层（`server/internal/modules/<模块>/adapter/http/gen/server.gen.go`、`server/internal/platform/httpserver/apigen/`）；每个模块的请求体结构表（同一目录的 `bodyshape.gen.go`，由 `server/tools/bodyshapegen` 读同一份接口描述生成，服务端据此在 handler 之前拒绝不合契约的请求体）；以及 sqlc 按 `server/sqlc.yaml` 从 `server/internal/modules/<模块>/adapter/postgres/queries/*.sql` 生成的查询代码（`adapter/postgres/gen/`；sqlc 以 `CGO_ENABLED=0` 运行，不需要 C 编译器）。
  - `make gen-web`：打包好的 `api/dist/openapi.yaml`，以及 TS 客户端 `web/packages/api-client` 的类型，需要 Node（先执行 `pnpm install`）。
- 生成的文件不要手改。`make gen-check` 会重新生成一遍，检查生成物已经提交、没有差异；持续集成也执行这项检查。
- 每个操作都写明 `security`（需要令牌的写 `[{bearer: []}]`，公开的写 `[]`）和 `x-problem-codes`（它可能返回的错误码；所有操作都可能返回的写在 `api/openapi.yaml` 顶层）。`apitest` 的测试核对这些写法，模块的 handler 测试要把声明的每个错误码都返回一次。
- `make lint` 依次执行 `make lint-go`（golangci-lint）和 `make lint-web`（关键词守卫，前端的类型检查、oxlint 警告数核对、格式检查、中英文翻译键一致性检查，以及 `tools/` 下脚本的 lint、`tools/` 和根目录配置文件的格式检查）。

## 前端

`web/apps/web` 和 `web/packages/*` 从 Plane 原样迁入，来源提交和之后的每一处改动见[前端改动清单](docs/v0/frontend-changes.md)；`web/packages/api-client` 是生成的 TS 客户端。前端任务由 turbo 按 `turbo.json` 编排，入口仍是 Makefile：

- **开发**：`make dev-db`、`make run` 启动后端，再在另一个终端执行 `make web-dev`，打开 http://127.0.0.1:3000 。Vite 把 `/api` 转发给 `127.0.0.1:8080`。`make web-dev` 同时监视 `web/packages/*`：改了某个包的代码，这个包重新构建，页面随之更新。第一次启动时 Vite 要预构建依赖，页面如果报"Outdated Optimize Dep"，刷新一次即可。
- **单元测试**：`make test-web` 运行各包 `test` 脚本中的 vitest（经 turbo，持续集成的 `web` 任务也执行它）。只给以后仍然有效的稳定逻辑写小测试，测试文件 `*.test.ts` 放在被测代码旁边；包里还没有 `test` 脚本时，加上 `"test": "vitest run"` 和开发依赖 `vitest`（`catalog:`）。`make test` 只运行 Go 测试。
- **没有前端环境变量**：前端与接口同源部署，接口一律用相对路径（`/api/…`、`/auth/…`），代码不读取 `process.env` 或 `import.meta.env.VITE_*`，关键词守卫看住这一点。`import.meta.env.DEV`、`PROD` 是 Vite 按构建模式给出的常量，不是环境变量。
- **构建**：`make build` 构建前端，复制到 `server/internal/platform/webui/dist/`，编译出内嵌前端的 `bin/nerve`。运行时要在 `server/` 目录下，`config.local.yaml` 才能生效：`cd server && NERVE_ENV=dev ../bin/nerve serve`，之后打开 http://127.0.0.1:8080 。`dist/` 中只提交了 `.gitkeep`，没有构建过前端时页面上只有一句提示。
- **清掉旧的构建产物**：`make build` 之后，`make run` 和 `go test` 都会继续内嵌这份构建。要去掉它：`find server/internal/platform/webui/dist -mindepth 1 ! -name .gitkeep -delete`（与 Makefile 里 `make build` 自己的清理命令相同）。
- **M0 中看到的页面**：前端还在调用 Plane 的接口（例如 `/api/instances/`），Nerve 返回 404，页面显示"Looks like Nerve didn't start up correctly!"。这是预期的，前端从 M2 起对接 Nerve 的接口。
- **lint 警告数等于上限**：每个包的 `check:lint` 脚本是 `node <到仓库根目录的相对路径>/tools/lint-cap.mjs <上限>`，它运行 oxlint，要求警告数正好等于上限，有任何错误都失败。警告多了，`make lint-web` 失败并列出这个包的全部警告：修掉新增的那几条，上限只能调低。警告少了（修掉了警告，或者删掉了带警告的代码），同样失败，并给出应调低到的数值：在同一个提交里把上限改成这个数。`make lint-web` 只打印失败任务的输出；要看某个包的全部警告，执行 `pnpm --filter <包名> exec oxlint .`。
- **关键词守卫**：`make lint-web` 的第一步是 `node tools/keywords.mjs`，规则在 `tools/keywords.json`（M1 设计 7.4）。它检查 git 列出的文件（包括还没 `git add` 的新文件），命中规则、又没有登记例外就失败，并列出规则、文件、行号和命中的原文；规则或文件读取有问题时以 2 退出。删掉一个功能时，在同一个提交里加上它的规则（每条规则带理由和命中、不命中的样本）。确实要保留的命中登记为例外：规则、文件、命中的原文、理由和到期的 M 或 Phase，一条例外只覆盖一处；例外不再命中任何内容时守卫会提醒删掉它。`tools/` 下的脚本本身也由 `make lint-web` 检查：oxlint 不允许警告（根目录 `package.json` 的 `check:lint`）；oxfmt 检查它们和根目录的工具链配置的格式，文件列表只写在根目录 `package.json` 的 `fix:format` 里，`check:format` 就是带 `--check` 运行它。
- **修格式**：`pnpm exec turbo run fix:format` 用 oxfmt 就地格式化所有包，以及根目录 `fix:format` 列出的文件。
- **多语言**：只有 `en` 和 `zh-CN`（`web/packages/i18n/src/locales/`）。两种语言的命名空间文件和键必须完全一致，同一个键不能出现在两个命名空间里，`make lint-web` 检查（i18n 包的 `check:sync`）；加、删文案时两种语言一起改。本地存储或用户资料中的其他语言按英文处理。
- **未使用的代码**：`make knip` 先生成 web 的路由类型，再用 knip 检查未使用的文件、导出和依赖，配置在 `knip.jsonc`。它是门禁（M1/P3 清零之后）：有任何未使用的代码，或配置本身过时，都会失败。持续集成的 `web` 任务执行它。

## 端到端测试

`e2e/` 是 Playwright 项目（工作区包 `@nerve/e2e`），一个用户故事一个测试文件，放在 `e2e/stories/` 下。被测对象是 `make build` 编译出的 `bin/nerve`（内嵌前端）加上 Postgres 18。

- **第一次运行之前**，安装 Playwright 用的 Chromium（下载到本机的缓存目录；升级 Playwright 之后再执行一次）：

  ```bash
  cd e2e && pnpm exec playwright install chromium
  ```

- **运行**：`make e2e`。它先执行 `make build`（没有改动时约 1 秒），再运行全部故事。需要 Docker：测试用 testcontainers 启动一个 Postgres 容器，运行结束后自动删除。
- **测试环境**：`e2e/global-setup.ts` 启动 Postgres，用 `bin/nerve migrate up` 迁移模板库 `nerve_template`。每个 Playwright worker 从模板复制出自己的库，用 test 配置启动自己的 `nerve serve`：nerve 在 `127.0.0.1:0` 上监听，把实际地址写进地址文件（`server.addr_file`），fixture 读出地址，`/readyz` 返回 200 之后才运行故事。故事从 `e2e/fixtures/test.ts` 导入 `test`：用 `api`（生成的 TS 客户端）、`request`、`page` 访问本 worker 的 nerve，用 `db` 查询它的数据库（每个 worker 一个连接池；断言函数按表放在 `e2e/fixtures/assert/`）。需要另一种配置的故事（例如关闭注册）用 `nerveWith` 在同一个库上另起一个 nerve。
- **版本号**：`make build` 把 `VERSION`（默认 `0.1.0-dev`）写进 `bin/nerve`，例如 `make build VERSION=0.1.0`。`make e2e` 把同一个值放进环境变量 `NERVE_VERSION` 交给测试，S3 核对 `/api/v0/instance` 返回的版本号。
- **同源**：S2 断言页面的所有请求都发往 nerve 自身（见"前端"一节的"没有前端环境变量"一条）；有请求发往别处时 S2 失败，失败信息列出这些请求。
- **只运行部分故事、打开浏览器调试**：先 `make build`，再直接运行 Playwright，`NERVE_VERSION` 要与构建时的 `VERSION` 相同：

  ```bash
  cd e2e && NERVE_VERSION=0.1.0-dev pnpm exec playwright test s2 --headed
  ```

- **失败时**：报告在 `e2e/playwright-report/`（`cd e2e && pnpm exec playwright show-report` 打开），失败用例的操作记录（trace）、截图和数据库快照（`database.sql`，本 worker 数据库的 `pg_dump`，也是报告中的附件）在 `e2e/test-results/`，每个 worker 的 nerve 日志是 `e2e/test-results/nerve-w<编号>.log`。持续集成的 `e2e` 任务失败时，把这两个目录上传为 `playwright-report`。

## 部署

- **设 `NERVE_ENV=prod`**。不设时 nerve 按 dev 的默认值运行：注册默认开放；没有配置签名密钥时用临时密钥，重启后已签发的访问令牌全部失效。nerve 不是 prod、却监听在本机回环地址之外时，启动日志会记一条 WARN 提醒。
- **签名密钥**：prod 必须提供 Ed25519 私钥（PKCS#8 PEM 文件），用 `auth.jwt.private_key_file`（环境变量 `NERVE_AUTH__JWT__PRIVATE_KEY_FILE`）指向它；没有提供或读不出来时 nerve 拒绝启动，错误指出这个配置项。生成：

  ```bash
  openssl genpkey -algorithm ed25519 -out nerve-jwt.pem
  chmod 600 nerve-jwt.pem
  ```

  访问令牌由它签名，刷新令牌的 MAC 密钥也从它派生。换密钥（替换文件后重启）的后果：已签发的访问令牌验签失败，客户端续期一次即可；换钥之前的旧代刷新令牌不再能被认出是重复使用，所以刷新令牌正被盗用时换钥，受害者只是被登出（恢复靠修改密码或 `nerve users reset-password`）；同一出口 IP 后面的大量标签页同时续期，会短暂得到 429。所以在低峰时换。
- **注册**：prod 默认关闭注册（`auth.signup_enabled: false`）。第一个账户用 `nerve users create` 创建（M2/P3 加入；在那之前临时设 `NERVE_AUTH__SIGNUP_ENABLED=true`）。
- **迁移**：prod 默认不在启动时迁移（`database.auto_migrate: false`），先执行 `nerve migrate up`，再 `nerve serve`。用单独的数据库角色执行迁移时，运行服务的角色除了读写业务表，还要能读 `goose_db_version`（`GRANT SELECT ON goose_db_version TO <服务的角色>`）：`/readyz` 靠它判断迁移是否已完成。

## Plane 表结构快照

`tools/plane-schema/plane-v1.4.2-schema.sql` 是 Plane v1.4.2 的完整表结构，各个 M 为自己的模块建表时以它为起点，再按[差异清单](docs/v0/plane-diff.md)修改。它是生成物，不要手改；只有升级 Plane 基线时才需要用 `make plane-schema` 重新生成（需要 Docker），说明见 [tools/plane-schema/README.md](tools/plane-schema/README.md)。

## 版权

Copyright © 2026 OpenNerve。以 [GNU AGPL-3.0](LICENSE) 协议发布。

前端的部分代码来自 [Plane](https://github.com/makeplane/plane)（Copyright © Plane Software, Inc. and contributors，AGPL-3.0），相关文件保留了原有的版权声明。`web/apps` 和 `web/packages` 里与 Plane 的文件放在一起的 Nerve 文件带 `Copyright (c) 2026-present OpenNerve` 和 `SPDX-License-Identifier: AGPL-3.0-only`；放不下文件头的资源（例如 PNG）登记在 `web/apps/web/app/assets/brand/SOURCES.md` 里。`web/packages/api-client` 和 `web/` 以外 Nerve 自己的代码不加文件头，以本仓库的 [LICENSE](LICENSE) 为准。

服务端的常见密码名单 `server/internal/modules/identity/domain/common_passwords.txt` 由 `tools/password-blocklist/build.mjs` 从英国国家网络安全中心（NCSC）发布的 `PwnedPasswordsTop100k.txt`（泄露最多的前 10 万个密码，数据来自 Have I Been Pwned 的 Pwned Passwords）过滤生成。NCSC 的原地址已失效，SecLists 收录的是同一个文件（SHA-256 相同）：[`Passwords/Common-Credentials/100k-most-used-passwords-NCSC.txt`](https://raw.githubusercontent.com/danielmiessler/SecLists/master/Passwords/Common-Credentials/100k-most-used-passwords-NCSC.txt)。Contains public sector information licensed under the [Open Government Licence v3.0](https://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/). Pwned Passwords 本身没有许可和署名要求。
