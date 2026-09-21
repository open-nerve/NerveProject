# M0/P1 仓库与工具链：设计说明（spec）

| 项 | 内容 |
|---|---|
| Phase | M0/P1 `repo-toolchain` |
| 日期 | 2026-09-22 |
| 状态 | 待审阅 |
| 上级文档 | [M0 设计文档](../M0-design.md) 第 1、2、6 节 |

## 1. 目标
建立仓库的工程骨架：锁定工具链版本，提供统一的命令入口和本地开发数据库，搭起持续集成。P1 只写一个 Go 包 `platform/buildinfo`（见 2.2）；之后各 Phase 在这个骨架上逐步增加内容。

## 2. 交付物

### 2.1 根目录
| 文件 | 内容 |
|---|---|
| `package.json` | `"private": true`；`"packageManager": "pnpm@11.10.0"`；`engines.node: ">=24"`。P1 中没有依赖和脚本（前端在 P5 迁入） |
| `pnpm-workspace.yaml` | `packages: [web/apps/*, web/packages/*, e2e]`。这些目录在 P5 和 P6 才会出现，目前匹配不到任何包，pnpm 允许这种情况 |
| `pnpm-lock.yaml` | 由 `pnpm install` 生成 |
| `.node-version` | `24` |
| `.editorconfig` | UTF-8、LF 换行；Go 文件用 tab 缩进；其他文件用 2 个空格缩进；去掉行尾空白；文件末尾保留换行 |
| `.gitignore` | 追加：`bin/`、`node_modules/`、`.turbo/`、`coverage/`、`*.test`、`*.out` |
| `Makefile` | 见 2.4 |
| `README.md` | 追加"开发环境"一节：需要安装什么、第一次怎么启动 |

### 2.2 Go 模块
| 文件 | 内容 |
|---|---|
| `server/go.mod` | `module github.com/open-nerve/NerveProject/server`；`go 1.27`；`toolchain go1.27.1`。P1 中没有依赖 |
| `server/tools/go.mod` | 独立的模块，只用来锁定开发工具的版本。P1 只锁定 `oapi-codegen v2.8.0`（通过 `tool` 指令） |

- **工具的调用方式**：在 `server/` 目录下执行 `go tool -modfile=tools/go.mod oapi-codegen …`。Makefile 会把这个细节封装起来。
- **Go 版本**：本机的 Go 1.26 会按 `toolchain` 指令自动下载 1.27.1（本机的 `GOTOOLCHAIN=auto`，已核实）。

**`server/internal/platform/buildinfo`**（从 P2 提前到 P1）：
- **为什么提前**：已核实，在一个没有 Go 代码的模块上，`go test ./...` 会以退出码 1 结束，golangci-lint 会报"no go files to analyze"。P2 的 `nerve version` 和 P3 的试点模块本来就要用这个包，把它提前，lint 和 test 从一开始就有真实的检查对象，不需要为"空项目"写特殊判断。
- **接口**：`buildinfo.Get() Info`，其中 `Info{Version, Commit, CommitTime string; Modified bool}`。
- **数据来源**：
  - 版本号是包内变量，默认值为 `0.1.0-dev`，发布构建时通过 `-ldflags -X` 覆盖。
  - 提交号、提交时间、工作区是否有未提交的改动，来自 Go 工具链在 git 仓库中构建时自动嵌入的 VCS 信息（`runtime/debug.ReadBuildInfo`）。拿不到时（测试、`go run`）为 `unknown`。

### 2.3 开发数据库：`deploy/compose.dev.yaml`
- **镜像**：`postgres:18.6`（写死小版本，保证每个人的环境一致）。
- **账号和库名**：`POSTGRES_USER`、`POSTGRES_PASSWORD`、`POSTGRES_DB` 都是 `nerve`。这是只在本机使用的开发账号。
- **端口**：`${NERVE_DEV_DB_PORT:-55432}:5432`。默认不用 5432，是为了避开本机已有的其他 Postgres（包括其他项目的容器），避免连错数据库；如果 55432 也被占用，可以通过 `NERVE_DEV_DB_PORT` 环境变量换一个。
- **数据卷**：挂载到 `/var/lib/postgresql`。**注意**：从 18 版开始，Postgres 官方镜像的数据目录改成了 `/var/lib/postgresql/18/docker`（已核实），挂载到旧路径 `/var/lib/postgresql/data` 的话，数据不会写进卷里。
- **健康检查**：`pg_isready -U nerve -d nerve`。
- **Compose 项目名**：固定为 `nerve-dev`，避免和其他项目冲突。

### 2.4 Makefile
**只包含在 P1 中就能正常工作的命令**，不写空的占位命令。其余命令由各 Phase 在实现对应功能时加入。

| 命令 | 作用 |
|---|---|
| `make help`（默认） | 列出所有命令及说明。说明写在每个命令后面的 `## 说明` 注释里，由 help 自动提取 |
| `make dev-db` | 启动开发数据库，等到健康检查通过才返回 |
| `make dev-db-down` | 停止开发数据库，保留数据 |
| `make dev-db-reset` | 停止开发数据库并删除数据卷 |
| `make tools` | 把锁定版本的 golangci-lint（v2.13.2）下载到 `./bin`；如果已经是这个版本，就跳过 |
| `make lint` | `bin/golangci-lint run`（在 `server/` 目录下执行） |
| `make test` | `go test ./...`（在 `server/` 目录下执行） |

兼容性：必须能在 macOS 自带的 GNU Make 3.81 上运行，不使用 4.x 才有的语法。

### 2.5 持续集成：`.github/workflows/ci.yml`
- **触发条件**：推送到任意分支、任意 PR。
- **并发控制**：同一个分支有新的运行时，取消旧的运行。

| 任务 | 步骤 |
|---|---|
| `server` | checkout → `actions/setup-go`（用 `server/go.mod` 指定版本）→ `make lint`（下载锁定版本的 golangci-lint 并运行）→ `make test` |
| `web` | checkout → `actions/setup-node`（按 `.node-version`）→ `corepack enable` → `pnpm install --frozen-lockfile` |

持续集成和本地执行的是同一套 Makefile 命令，不单独使用 golangci-lint 的官方 Action，避免本地和持续集成的版本不一致。P1 中两个任务的内容都还很少。之后各 Phase 往里面加内容：P2 加测试，P3 加生成物一致性检查，P5 加前端检查和构建，P6 加 `e2e` 任务。

## 3. 验收标准
1. 在只装了 Docker、Go（1.26 或更高）、Node 24 的机器上：
   - 执行 `make dev-db` 后数据库启动，`docker compose … exec db psql -U nerve -c 'select version()'` 输出 PostgreSQL 18.6。
   - `make dev-db-down` 后再执行 `make dev-db`，之前写入的数据还在。`make dev-db-reset` 之后，数据被清空。
2. 在 `server/` 目录下执行 `go tool -modfile=tools/go.mod oapi-codegen -version`，输出 v2.8.0；过程中自动下载了 Go 1.27.1 的工具链。
3. `make tools`、`make lint`、`make test` 都能成功执行；`buildinfo` 的单元测试通过。
4. `corepack enable && pnpm install --frozen-lockfile` 成功。
5. 推送到 GitHub 后，持续集成的两个任务都通过。
6. `make help` 列出上面所有的命令和说明。

## 4. 不在 P1 范围内
除 `buildinfo` 以外的 Go 源码、`.golangci.yml`（P2 加入，包含 depguard 规则；P1 使用 golangci-lint 的默认规则）、前端代码、`turbo.json`（P5）、端到端测试（P6）。

## 5. 风险
| 风险 | 应对 |
|---|---|
| GitHub Actions 的运行结果需要在网页上查看（本机没有安装 gh 命令行工具） | 如果仓库是公开的，就用 GitHub 的公开接口查询结果；否则请你在 Actions 页面确认 |
| sqlc 依赖 cgo（M2 才接入） | P1 中写一份 handoff，放进 `M2-auth/handoffs/`，M2 接入时处理 |
| GitHub Actions 各官方 Action 的最新大版本号未经核实 | 实现时通过 GitHub 接口确认 |
