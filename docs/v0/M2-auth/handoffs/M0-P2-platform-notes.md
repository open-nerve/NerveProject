---
status: open
from: M0/P2
to: M2
created: 2026-09-22
---

# M2 在 P2 平台层上扩展时的注意事项

M2 第一次加入认证、事务、后台任务和真正的迁移文件，届时处理以下事项：

1. **TxManager 端口由使用方声明。**
   - 按 archtest 规则 2，`app` 层不能导入 `platform/*` 或 pgx。
   - `TxManager` 接口放在 `internal/shared`，实现放在 `platform/postgres`，由 `bootstrap` 接上。总体设计 6.4 已按此更新。
2. **认证、限流、接口调用日志按路由挂载，不改 `httpserver.NewServer`。**
   - `NewServer` 固定套上三个平台中间件：请求 ID → 异常恢复 → 访问日志。
   - 认证等中间件通过 oapi-codegen 的 `Middlewares` 和 strict 中间件挂在接口路由上。这样它们位于访问日志之后，也不会作用于 `/healthz`、`/readyz` 和前端页面。
3. **导出请求 ID。** 接口调用日志或 500 的记录第一次需要读取请求 ID 时，把 `httpserver` 包内的 `requestID(ctx)` 导出为 `RequestID(ctx)`。
4. **新的密钥类配置项要写进 `LogValue`。**
   - 日志只输出 `LogValue` 明确列出的字段。
   - JWT 私钥、S3 密钥等新配置项，以及 `*_file` 形式的配置项，都要在对应的 `LogValue` 中打码或省略。
5. **River 与停机顺序。**
   - `bootstrap` 的 `app.run` 需要调整，让 River 和 HTTP 服务一起运行。
   - 停机顺序：HTTP 停机 → River 停止（有自己的超时）→ 迁移执行器 → 连接池。
   - `pool.Close()` 要设置时间上限：强制关闭 HTTP 连接后，不理会 ctx 的 handler 可能仍然占着数据库连接。
   - River 自己的表结构用 goose 的 SQL 迁移建立（而不是 River 的迁移命令），并纳入迁移命名规则。SQL 取自锁定版本的 `river migrate-get`，`--up` 的输出写进 goose 的 Up 段，`--down` 的输出写进 Down 段。第一次导出用 `--line main --all --exclude-version 1`：版本 1 只建 `river_migration` 表，那是 River 自己的迁移命令用的，改用 goose 后不需要。以后升级 River，用 `--version N` 导出新增的版本，另写一份新迁移，不改已经发布的迁移文件。总体设计 5.2 已统一为这个做法（M0 对抗性评审 Minor 2）。
6. **archtest 的补充。**
   - 规则 6 从 `adapter/http/gen` 推广到所有 `adapter/*/gen`，覆盖 sqlc 的生成代码。
   - ~~建立 `internal/shared` 时，为它加一条"只能依赖标准库"的规则。~~ M0 加固已完成：archtest 第 10 条要求 `internal/shared` 只依赖标准库（不含 `net/http`、`database/sql`）和它自己，`domain → shared → net/http` 会被拦下，建立 `internal/shared` 时不用再加。
7. **迁移与就绪检查。**
   - 第一个迁移文件出现后，`/readyz` 的迁移检查才真正生效。
   - 如果生产环境用单独的数据库角色执行迁移，运行服务的角色需要有 `goose_db_version` 表的 SELECT 权限。
   - 在从未迁移过的库上第一次访问 `/readyz`，会建出这张表（goose 的行为，P2 spec 2.5）。
8. **P2 暂不处理的小问题**（M2 碰到时再决定）：
   - 环境变量传入空值时，布尔配置项会被悄悄解码为 `false`。
   - `nerve serve` 在创建 logger 之后出现致命错误时，只在 stderr 打印一行，没有结构化的日志记录。
   - `/healthz` 和 `/readyz` 的每次探测都会写一条 info 级别的访问日志，在生产环境中可能过多。

来源：[M0/P2 评审记录](../../M0-foundation/reviews/P2-server-platform-review.md)。
