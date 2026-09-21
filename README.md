# Nerve

一个轻量的多人协作项目管理系统，页面和接口能力完全对等：同一个账户，既可以由人在页面上使用，也可以由 Agent 通过接口使用。

- 后端：Go（单个可执行文件）+ PostgreSQL
- 前端：基于 [Plane](https://github.com/makeplane/plane) 前端分叉并裁剪
- 协议：[AGPL-3.0](LICENSE)

项目处于 v0 开发阶段，从 [docs/](docs/README.md) 开始阅读。

## 开发环境

需要安装：
- Docker（含 Compose v2）
- Go 1.26 或更高。第一次在 `server/` 下执行 Go 命令时，会自动下载 `server/go.mod` 指定的 Go 1.27.1（前提是 `GOTOOLCHAIN=auto`，这是 Go 官方安装包的默认值；部分 Linux 发行版自带的 Go 默认是 `local`，需要先执行 `go env -w GOTOOLCHAIN=auto`）
- Node.js 24，并执行一次 `corepack enable`（pnpm 的版本由 `package.json` 锁定）

第一次启动：

```bash
make dev-db   # 启动本地 Postgres 18（端口 55432，可用 NERVE_DEV_DB_PORT 修改）
make test     # 运行测试（需要 Docker：集成测试用 testcontainers 启动 Postgres）
make run      # 以 dev 配置运行后端，监听 :8080；Ctrl-C 停止
make          # 查看所有命令
```

- 只跑单元测试、跳过需要 Docker 的集成测试：在 `server/` 下执行 `go test -short ./...`。
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

## 版权

Copyright © 2026 OpenNerve。以 [GNU AGPL-3.0](LICENSE) 协议发布。

前端的部分代码来自 [Plane](https://github.com/makeplane/plane)（Copyright © Plane Software, Inc. and contributors，AGPL-3.0），相关文件保留了原有的版权声明。
