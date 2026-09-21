---
status: open
from: M0/P1
to: M2
created: 2026-09-22
---

# sqlc 接入时的两个注意事项

M2 首次接入 sqlc（v1.31.1，放进 `server/tools/go.mod`）时处理：

1. **sqlc 依赖 cgo**：它通过 pg_query_go 解析 SQL，需要 C 编译器。
   - 验证本机（macOS，需要 Xcode Command Line Tools）和持续集成（ubuntu）上，`go tool -modfile=tools/go.mod sqlc version` 都能编译运行。
   - 如果 cgo 构建有问题，改用 sqlc 官方 Docker 镜像（`sqlc/sqlc:1.31.1`）运行代码生成。
2. **sqlc 的解析器基于 PG 17**：它不认识 PG 18 新增的函数，例如 `uuidv7()`。我们的 ID 由应用生成（Go 1.27 标准库的 `uuid.NewV7()`），迁移中不要写 `DEFAULT uuidv7()`；如果需要用到其他 PG 18 特有的语法，先验证 sqlc 能否解析。

来源：[M0/P1 spec](../../M0-foundation/specs/P1-repo-toolchain.md) 第 5 节。
