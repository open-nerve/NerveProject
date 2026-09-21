---
status: done
from: M0/P1
to: M0/P2
created: 2026-09-22
---

# P2 接入依赖和配置时的三个注意事项

1. **`server/go.mod` 的 `go` 指令保持 `go 1.27`，不要升到补丁版本。**
   - golangci-lint 2.13.2 的预编译二进制由 go1.27.0 构建，遇到 `go` 行高于自身构建版本的模块会拒绝运行。
   - P2 加入 pgx、cobra、koanf 等依赖时，`go get` / `go mod tidy` 可能把 `go` 行抬高，提交前检查一遍。
2. **`toolchain go1.27.1` 在两个模块里保持一致。**
   - `server/go.mod` 和 `server/tools/go.mod` 都写了这一行。持续集成的 `setup-go` 只读 `server/go.mod`。
   - 以后升级 Go 版本时，两处一起改。
3. **`config.dev.yaml` 中的数据库地址写死了端口 55432。**
   - 开发者如果通过 `NERVE_DEV_DB_PORT` 换了开发库的端口，还需要用环境变量覆盖 `database.url`。
   - 在 P2 的配置说明（或 README）里写明这一点。

## 处理结果（M0/P2）

1. P2 加入的全部依赖（见 [P2 spec](../specs/P2-server-platform.md) 2.2）都已核实不会抬高 `go` 行；P2 plan 的每个 Task 在 `go get` / `go mod tidy` 之后都检查了 `server/go.mod` 仍是 `go 1.27`。
2. P2 没有改动任何 `toolchain` 行，`server/go.mod` 和 `server/tools/go.mod` 都是 `toolchain go1.27.1`。
3. README 的"开发环境"一节和 `server/configs/config.dev.yaml` 的注释都写明了：用 `NERVE_DEV_DB_PORT` 换端口时，还要用 `NERVE_DATABASE__URL` 或 `config.local.yaml` 覆盖 `database.url`。

来源：[M0/P1 评审记录](../reviews/P1-repo-toolchain-review.md)。
