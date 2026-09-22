---
status: done
from: M0/P1
to: M0/P3
created: 2026-09-22
---

# 持续集成的任务划分需要在 P3 定下来

## 问题

M0 设计文档第 6 节的两处描述对不上：
- `make lint` 包含 golangci-lint、前端类型检查和 oxlint；`make gen-check` 需要 Node 工具（Redocly、openapi-typescript）。
- 但持续集成的 `server` 任务没有安装 Node，却要执行 `make gen-check` 和 `make lint`。

P1 只有 Go 代码，这个矛盾还没有暴露。P3 加入 `make gen-check`、P5 加入前端检查时就会出现。

## 建议

在 P3 的 spec 中定下来。例如：
- 按区域拆分命令：`lint-go` / `lint-web`、`gen-check` 按需拆分；`make lint` 依次执行两者，供本地使用。
- 持续集成的每个任务只调用自己区域的命令。
- 这样仍然满足"本地和持续集成执行同一套 Makefile 命令"。

## 顺带决定

**同仓 PR 每次推送跑两次持续集成。**
- 原因：`push` 和 `pull_request` 两个事件的 `github.ref` 不同，落在不同的并发组里。
- P1 接受了这一点：spec 要求任意分支和任意 PR 都触发，而且公开仓库的标准运行器免费。
- P3 调整持续集成结构时一并重新评估；最晚在 P6 加入端到端测试、持续集成变慢之前决定。

## 处理结果（M0/P3）

1. **命令按区域拆分**（[P3 spec](../specs/P3-api-contract.md) 2.10）：`lint-go` / `lint-web`、`gen-go` / `gen-web`、`gen-check-go` / `gen-check-web`。`*-go` 只需要 Go，`*-web` 只需要 Node；不带后缀的 `lint`、`gen`、`gen-check` 依次执行两个区域，供本地使用。
2. **持续集成的每个任务只调用自己区域的命令**：`server` 任务执行 `make gen-check-go` → `make lint-go` → `make test`；`web` 任务执行 `pnpm install --frozen-lockfile` → `make gen-check-web` → `make lint-web`。本地和持续集成仍然使用同一套 Makefile 命令。
3. **同仓 PR 的重复运行**：两个任务都加了条件 `github.event_name == 'push' || github.event.pull_request.head.repo.full_name != github.repository`。同仓库分支的 PR 已经由 push 事件在同一个提交上跑过，PR 页面显示的就是这次的结果；`pull_request` 事件只为来自 fork 的 PR 运行（fork 的推送不会在本仓库触发 push 事件）。
   - **必须通过检查的注意事项**：被 `if` 跳过的任务也会以同一个检查名字报告 Success。把 `server`/`web`（以及 P6 的 `e2e`）设为分支保护的必须检查之前，要先去掉这个跳过条件，或者另加一个 `if: always()` 的汇总任务并把它设为必须检查（详见 [M0 设计](../M0-design.md) 6.3）。

来源：[M0/P1 评审记录](../reviews/P1-repo-toolchain-review.md)。
