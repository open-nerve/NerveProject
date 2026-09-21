---
status: open
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

来源：[M0/P1 评审记录](../reviews/P1-repo-toolchain-review.md)。
