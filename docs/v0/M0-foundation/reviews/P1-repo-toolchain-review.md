# M0/P1 仓库与工具链：评审记录

| 项 | 内容 |
|---|---|
| Phase | M0/P1 `repo-toolchain` |
| 日期 | 2026-09-22 |
| 结论 | **通过**（整分支评审提出的问题已在一轮修复中处理完） |
| spec / plan | [spec](../specs/P1-repo-toolchain.md) / [plan](../plans/P1-repo-toolchain.md) |
| 分支 | `worktree-m0-p1-repo-toolchain`，从 `main` 的 `d7c6c78` 分出 |

## 1. 评审方式

- **逐个 Task 评审**：plan 的 6 个 Task 各由一个实现子任务完成，随后由独立的评审子任务检查两项：是否符合 spec，代码质量如何。6 个 Task 全部一次通过。
- **整分支评审**：全部 Task 完成后，对整个分支做一次评审，结论为"修复后可合并"。修复集中在一轮中完成，之后对修复的提交做了一次范围评审，全部通过。
- **持续集成**：每次推送后通过 GitHub 公开接口确认结果，三次运行都通过：
  - `c8a49e9`：首次加入持续集成
  - `7b74e65`：文档
  - `627a7eb`：修复轮

## 2. 验收标准核对（spec 第 3 节）

| # | 标准 | 结果 |
|---|---|---|
| 1 | `make dev-db` 启动 PostgreSQL 18.6；停止后数据保留，重置后数据清空 | 通过（Task 3；修复轮在全新数据卷上重新验证） |
| 2 | `oapi-codegen -version` 输出 v2.8.0，并自动下载 Go 1.27.1 | 通过（Task 2；修复轮 tidy 之后复验） |
| 3 | `make tools` / `make lint` / `make test` 成功 | 通过（Task 4）。故意加入一个未使用的函数，lint 能够报错，门禁有效 |
| 4 | `corepack enable && pnpm install --frozen-lockfile` | 通过（Task 1、持续集成的 `web` 任务） |
| 5 | 持续集成的两个任务都通过 | 通过（见第 1 节的三次运行） |
| 6 | `make help` 列出全部命令 | 通过（7 个命令） |

**持续集成里的 Go 版本**：
- 持续集成的日志需要登录才能下载，无法直接看到 `Go version` 这一步的输出。
- 改为查阅 `actions/setup-go@v7` 的源码确认：只要没有设置 `GOTOOLCHAIN=local`，它就会安装 `go.mod` 中 `toolchain` 指令指定的版本，也就是 1.27.1。持续集成没有设置这个变量。

## 3. 执行中的决定

| 决定 | 原因 |
|---|---|
| 开发库的默认端口从 5432 改为 **55432**（spec 2.3 已同步） | 本机 5432 已被其他项目的数据库占用，P2 的开发配置会连错库；也避开了 Homebrew / Postgres.app 装的本地 Postgres |
| 同仓 PR 每次推送跑两次持续集成（`push` 和 `pull_request` 各一次），暂时接受 | spec 要求任意分支、任意 PR 都触发；合并两者的并发组会让 PR 上出现"已取消"的检查。P3 调整持续集成结构时重新评估，见 handoff |

## 4. 整分支评审的发现与处理

| 发现 | 级别 | 处理 |
|---|---|---|
| 开发库监听所有网卡，且使用公开的账号密码 | Important | 已修：只绑定 `127.0.0.1` |
| `make tools` 下载失败时仍返回成功（`curl \| sh` 没有 `pipefail`） | Important | 已修：配方开头加 `set -o pipefail`；已验证下载失败时命令会失败，且不会覆盖已安装的版本 |
| 提交 `27c3c28` 的署名与其他提交不一致 | Important（仅影响提交历史） | 已推送的提交不改写；合并时处理（见第 7 节） |
| `server/tools/go.sum` 不整洁 | Minor | 已修 |
| 健康检查只检查 Unix socket，新数据卷初始化期间会过早返回就绪 | Minor | 已修：`pg_isready -h 127.0.0.1` |
| golangci-lint 安装脚本没有锁定版本 | Minor | 已修：改用对应版本标签下的脚本 |
| M0 设计文档与实现不一致（镜像版本、buildinfo 的字段） | Minor | 已修 |
| `engines.node: ">=24"`：Node 25 起不再自带 corepack，README 中的安装步骤会失效 | Minor | 改为 `^24`（spec 2.1 已同步） |
| 部分 Linux 发行版自带的 Go 默认 `GOTOOLCHAIN=local` | Minor | README 已补充说明 |
| `.editorconfig` 让 `go.mod` 等文件使用空格缩进 | Minor | 已修 |
| 持续集成会取消 main 分支上较早的运行；`ubuntu-latest` 未锁定；任务没有超时 | Minor | 已修：main 分支的每次运行都保留；运行环境锁定为 `ubuntu-24.04`；任务超时 15 分钟（spec 2.5 已同步） |
| spec 状态、README 中的阶段描述过时 | Minor | 已修 |

**决定不处理**：
- 删除多余的 `.gitkeep`：其他目录也保留着，保持一致。
- 持续集成缓存 golangci-lint、把 actions 锁定到提交哈希：目前收益低，以后需要时再做。

## 5. 最终实现与 plan 的差异

plan 是执行记录，不回改。整分支评审后的修复使以下内容与 plan 中的代码片段不同，以本记录和 spec 为准：
- Task 1：`package.json` 的 `engines.node` 改为 `^24`；`.editorconfig` 增加 Go 模块文件使用 tab 的规则。
- Task 3：`compose.dev.yaml` 的端口只绑定 `127.0.0.1`；健康检查加了 `-h 127.0.0.1`。
- Task 4：`tools` 命令加 `set -o pipefail`；安装脚本的地址锁定到版本标签。
- Task 5：`ci.yml` 锁定运行环境为 `ubuntu-24.04`，加 `timeout-minutes: 15`；main 分支上的运行不会被取消。

## 6. 移交事项

| handoff | 交给 | 内容 |
|---|---|---|
| [P1-repo-toolchain-go-db-notes](../handoffs/P1-repo-toolchain-go-db-notes.md) | M0/P2 | `go` 指令保持 `go 1.27`；两个 go.mod 的 `toolchain` 保持一致；换开发库端口时，还要覆盖 `database.url` |
| [P1-repo-toolchain-ci-split](../handoffs/P1-repo-toolchain-ci-split.md) | M0/P3 | 按区域拆分持续集成的任务和 lint、gen-check 命令；重新评估同仓 PR 的重复运行 |
| [M0-P1-sqlc-cgo](../../M2-auth/handoffs/M0-P1-sqlc-cgo.md) | M2 | sqlc 依赖 cgo；它的解析器基于 PG 17 |

## 7. 已知限制

- **只支持 Node 24**：Node 26 预计在 2026 年 10 月成为 LTS。届时需要决定不依赖 corepack 的 pnpm 安装方式，并且不能违反"除 Docker、Go、Node 以外不要求全局安装工具"的约束。
- **提交 `27c3c28` 的署名**：它的 Co-Authored-By 与其他提交不一致。
  - 如果用 squash 方式合并进 main，这个问题自然消失。
  - 如果保留逐个提交的历史，就需要改写这个已推送的提交并强制推送。
