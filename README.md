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
make test     # 运行测试
make          # 查看所有命令
```

## 版权

Copyright © 2026 OpenNerve。以 [GNU AGPL-3.0](LICENSE) 协议发布。

前端的部分代码来自 [Plane](https://github.com/makeplane/plane)（Copyright © Plane Software, Inc. and contributors，AGPL-3.0），相关文件保留了原有的版权声明。
