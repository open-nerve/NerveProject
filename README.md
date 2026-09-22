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
  - `make gen-go`：Go 接口层（`server/internal/modules/<模块>/adapter/http/gen/`、`server/internal/platform/httpserver/apigen/`），只需要 Go。
  - `make gen-web`：打包好的 `api/dist/openapi.yaml`，以及 TS 客户端 `web/packages/api-client` 的类型，需要 Node（先执行 `pnpm install`）。
- 生成的文件不要手改。`make gen-check` 会重新生成一遍，检查生成物已经提交、没有差异；持续集成也执行这项检查。
- `make lint` 依次执行 `make lint-go`（golangci-lint）和 `make lint-web`（前端的类型检查、oxlint、格式检查）。

## 前端

`web/apps/web` 和 `web/packages/*` 从 Plane 原样迁入，来源提交和之后的每一处改动见[前端改动清单](docs/v0/frontend-changes.md)；`web/packages/api-client` 是生成的 TS 客户端。前端任务由 turbo 按 `turbo.json` 编排，入口仍是 Makefile：

- **开发**：`make dev-db`、`make run` 启动后端，再在另一个终端执行 `make web-dev`，打开 http://127.0.0.1:3000 。Vite 把 `/api` 转发给 `127.0.0.1:8080`。改了 `web/packages/*` 下的代码，要重新执行 `make web-dev`。
- **不要建立 `web/apps/web/.env`**：`vite.config.ts` 用 dotenv 加载它；建了的话，构建会把其中的 `VITE_API_BASE_URL`（Plane 的 `.env.example` 里是 `http://localhost:8000`）打进产物，破坏同源部署。Nerve 不设置任何前端环境变量。
- **构建**：`make build` 构建前端，复制到 `server/internal/platform/webui/dist/`，编译出内嵌前端的 `bin/nerve`。运行时要在 `server/` 目录下，`config.local.yaml` 才能生效：`cd server && NERVE_ENV=dev ../bin/nerve serve`，之后打开 http://127.0.0.1:8080 。`dist/` 中只提交了 `.gitkeep`，没有构建过前端时页面上只有一句提示。
- **清掉旧的构建产物**：`make build` 之后，`make run` 和 `go test` 都会继续内嵌这份构建。要去掉它：`find server/internal/platform/webui/dist -mindepth 1 ! -name .gitkeep -delete`（与 Makefile 里 `make build` 自己的清理命令相同）。
- **M0 中看到的页面**：前端还在调用 Plane 的接口（例如 `/api/instances/`），Nerve 返回 404，页面显示 Plane 的"didn't start up correctly"。这是预期的，前端从 M2 起对接 Nerve 的接口。
- **lint 警告只降不升**：每个包的 `check:lint` 脚本用 `--max-warnings` 记着当前的警告数，警告多了 `make lint-web` 就失败；修掉警告后，在同一个提交里把这个数调低到新的警告数。`make lint-web` 用 `--output-logs=errors-only`，看不到具体的警告数；要看某个包当前的警告数，执行 `pnpm --filter <包名> run check:lint`，输出末尾的 `Found N warnings` 就是这个数。
- **修格式**：`pnpm exec turbo run fix:format` 用 oxfmt 就地格式化所有包。
- **未使用的代码**：`make knip` 用 knip 报告未使用的文件、导出和依赖，配置在 `knip.jsonc`。M0 只出报告：迁入的 Plane 代码有 379 处，退出码仍为 0，只有 knip 自身出错时才失败；M1 删减之后清零，改为门禁。持续集成的 `web` 任务执行它。

## 端到端测试

`e2e/` 是 Playwright 项目（工作区包 `@nerve/e2e`），一个用户故事一个测试文件，放在 `e2e/stories/` 下。被测对象是 `make build` 编译出的 `bin/nerve`（内嵌前端）加上 Postgres 18。

- **第一次运行之前**，安装 Playwright 用的 Chromium（下载到本机的缓存目录；升级 Playwright 之后再执行一次）：

  ```bash
  cd e2e && pnpm exec playwright install chromium
  ```

- **运行**：`make e2e`。它先执行 `make build`（没有改动时约 1 秒），再运行全部故事。需要 Docker：测试用 testcontainers 启动一个 Postgres 容器，运行结束后自动删除。
- **测试环境**：`e2e/global-setup.ts` 启动 Postgres，用 `bin/nerve migrate up` 迁移模板库 `nerve_template`。每个 Playwright worker 从模板复制出自己的库，在一个空闲的本机端口上用 test 配置启动自己的 `nerve serve`，`/readyz` 返回 200 之后才运行故事。故事从 `e2e/fixtures/test.ts` 导入 `test`：用 `api`（生成的 TS 客户端）、`request`、`page` 访问本 worker 的 nerve，用 `db` 查询它的数据库。
- **版本号**：`make build` 把 `VERSION`（默认 `0.1.0-dev`）写进 `bin/nerve`，例如 `make build VERSION=0.1.0`。`make e2e` 把同一个值放进环境变量 `NERVE_VERSION` 交给测试，S3 核对 `/api/v0/instance` 返回的版本号。
- **同源**：S2 断言页面的所有请求都发往 nerve 自身。构建时有 `web/apps/web/.env`（见"前端"一节的"不要建立"一条），S2 失败，失败信息列出发往别处的请求。
- **只运行部分故事、打开浏览器调试**：先 `make build`，再直接运行 Playwright，`NERVE_VERSION` 要与构建时的 `VERSION` 相同：

  ```bash
  cd e2e && NERVE_VERSION=0.1.0-dev pnpm exec playwright test s2 --headed
  ```

- **失败时**：报告在 `e2e/playwright-report/`（`cd e2e && pnpm exec playwright show-report` 打开），失败用例的操作记录（trace）和截图在 `e2e/test-results/`，每个 worker 的 nerve 日志是 `e2e/test-results/nerve-w<编号>.log`。持续集成的 `e2e` 任务失败时，把这两个目录上传为 `playwright-report`。

## Plane 表结构快照

`tools/plane-schema/plane-v1.4.2-schema.sql` 是 Plane v1.4.2 的完整表结构，各个 M 为自己的模块建表时以它为起点，再按[差异清单](docs/v0/plane-diff.md)修改。它是生成物，不要手改；只有升级 Plane 基线时才需要用 `make plane-schema` 重新生成（需要 Docker），说明见 [tools/plane-schema/README.md](tools/plane-schema/README.md)。

## 版权

Copyright © 2026 OpenNerve。以 [GNU AGPL-3.0](LICENSE) 协议发布。

前端的部分代码来自 [Plane](https://github.com/makeplane/plane)（Copyright © Plane Software, Inc. and contributors，AGPL-3.0），相关文件保留了原有的版权声明。
