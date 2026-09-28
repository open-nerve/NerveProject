---
status: open
from: M2/closeout
to: M8
created: 2026-09-28
---

# M2 交给 M8：接口调用日志、性能和内存的实测、部署、接口文档页

下面这些来自 [M2 设计](../../M2-auth/M2-design.md) 13.2 给 M8 的一行和"M3 及以后有 stores 的 M"一行中 M8 的部分；M0-P2、M0-P6 两份交接剩下的各一项（第 1、5 节）；以及收尾在 13.2 之外找到的几项：M2 设计 §16 中写明"M8 实测"的四项（第 2 节），River 在数据库恢复之后的重新启动（第 3 节）。

## 1. 接口调用日志挂在限流之后

- **做什么**：接口调用日志（总体设计 9.2 的 M8；差异清单第四节：记录所有写请求，不区分令牌类型，存储前去掉敏感请求头）作为按路由的中间件，挂在限流之后（M2 设计 3.6，总体设计 6.4）。
- **代码在哪**：`server/internal/platform/httpserver/api.go:121` 的 `Middlewares` 按顺序组装按路由的中间件；`httpserver.RequestID(ctx)`（`server/internal/platform/httpserver/middleware.go:62`）已导出，日志可以带上请求 ID。
- **M2 做了什么**：认证（P1）、失败闸门和限流（P2）都挂好了；这一项是 [M0-P2 交接](../../M2-auth/handoffs/M0-P2-platform-notes.md)第 2 条剩下的最后一件。
- **关闭条件**：中间件在限流之后，`API.Middlewares` 的顺序测试核对；被限流、认证失败的请求在它之前就结束、不进日志，M8 设计写明这与差异清单的"记录所有写请求"怎样对上。

## 2. 性能和内存的实测

M8 做 Go 进程的内存实测（总体设计 9.2：空闲时低于 50 MB）时，一并测下面几项。它们都是 M2 设计 §16 的应对里写明"M8 实测"的风险：

- **argon2 的内存**：并发上限 4 个（约 76 MiB，M2 设计 3.8）下的峰值。
- **argon2 的 CPU**：生产参数下登录、注册、修改密码的耗时；持续集成没有测过（测试环境用 m = 64 KiB、t = 1，M2/P1 评审第 7 节）。
- **常见密码名单的内存**：`server/internal/modules/identity/domain/common_passwords.txt`（33,904 行、278,161 字节）加载之后占用的内存。
- **失败闸门的并发**：认证之前的失败闸门按 IP 计数，先预留、后退回；同一个出口 IP 后同时在认证中的请求不能超过桶里当时剩下的单位（M2 设计 3.10）。实测同一 IP 的并发，看有效的调用方会不会得到 429。
- **请求体结构检查的开销**：请求体被解析三遍（codex-fixes 加了歧义扫描：歧义扫描、按表检查、生成代码解码，上限 1 MiB）。`bodyshape` 的 `TestCheckCostsAboutTheBody` 已守住最坏的几种请求体的内存（M2 设计 3.11），M8 实测常见请求体的耗时。
- **每个带令牌的请求多一次数据库查询**：访问令牌查它的会话行，PAT 查它的哈希（M2 设计 6.4、§16）。
- **与负责人的事项的关系**：调高 argon2 参数之后，登录耗时能区分休眠的账户和不存在的邮箱，是否加缓解由负责人以后决定，记在[总体设计第 10 节](../../v0-design.md)。它不是 M8 的任务；M8 的 argon2 数字供负责人参考。
- **关闭条件**：M8 的 review 有每一项的数字、测法和结论；超出预期的写明处理（调参、改默认值，或登记风险）。

## 3. 部署

- **镜像设 `NERVE_ENV=prod`**（M2 设计决策点 2 的缓解，6.1）：不设时 nerve 按 dev 的默认值运行，注册开放，签名密钥是临时的（README"部署"一节的第一条）。
- **容器的停止宽限期**：默认配置下停机最坏约 36 秒（HTTP 20、任务 10+1、连接池 5，M2/P3b spec 第 3 节第 7 条），超过 Docker 默认的 10 秒。部署文件要设 `stop_grace_period`（或调小这几个期限），否则进程在收尾中被 SIGKILL。
- **数据库恢复之后 River 能重新启动**：M2 只读过代码（M2/P3b spec 附录 A 的 E1）并用假客户端测了重试；真实的客户端只测了"数据库不可达时启动失败、`Stop` 立即结束重试"（M2/P3b 评审第 7 节）。部署核对中停一次数据库、再恢复，看定时任务恢复运行（清理任务的日志，或 `river_job` 的新行）。
- **迁移和服务分用两个数据库角色**（M2 codex-fixes，M2 设计 6.1）：部署文件这样做时，每次 `nerve migrate up` 之后以表的所有者执行 `deploy/runtime-grants.sql`（README"部署"的"迁移"一条）。部署核对在这样的环境里看：`/readyz` 200；清理任务的日志，或 `river_job` 里 `identity.cleanup_expired_sessions` 的 `completed` 行；以服务角色执行一次 `REINDEX INDEX CONCURRENTLY river_job_pkey` 成功（River 每天 00:00 UTC 的索引重建要 `MAINTAIN`）；日志里没有 `permission denied`。自动化的证明是 `server/internal/bootstrap/runtime_role_test.go`。
- **关闭条件**：镜像的环境里有 `NERVE_ENV=prod`；部署文件的停止宽限期不短于停机的最坏时间（或写明调小了哪些期限）；数据库重启的核对结果写进 M8 的 review；分用两个角色的部署按 `deploy/runtime-grants.sql` 授权，核对结果写进 M8 的 review。

## 4. 对外接口文档页不从 CDN 加载脚本

- **做什么**：对外接口文档页的脚本和样式从本站提供，不从 CDN 加载（M2 设计 8.3：自托管的 Nerve 在没有外网时也能用，不把使用情况告诉第三方）。同一个页面的其他事项见 [M0-P3 的交接](M0-P3-public-api-docs.md)。
- **关闭条件**：文档页不请求第三方的来源；页面带 CSP 时没有违规。

## 5. e2e 的 `webhook.ts`（M0-P6 交接剩下的一项）

- **做什么**：本地的 Webhook 接收端，按 M0/P6 定下的 fixture 写法加 `e2e/fixtures/webhook.ts`：普通函数，再由 `e2e/fixtures/test.ts` 接成 Playwright fixture，全局准备也能直接调用普通函数（[M0-P6 的交接](../../M2-auth/handoffs/M0-P6-e2e-notes.md)"fixture 模式的延伸"）。
- **关闭条件**：`webhook.ts` 照这个写法加入，Webhook 的故事用它核对收到的负载和签名。

## 6. stores 按会话分代（规则在总体设计 7.7）

- **做什么**：Webhook 设置页和接口调用日志页的 stores（例如 `web/apps/web/core/store/workspace/webhook.store.ts`）、services 照 [总体设计 7.7](../../v0-design.md) 写。
- **关闭条件**：M8 领域没有模块级的带令牌 service 或客户端（`git grep -n -E '^(export )?const [A-Za-z]+ = new [A-Za-z]+Service\(' -- web/apps/web` 中没有 M8 的）；M8 新加的 SWR 键带 `loginId`。

来源：[M2 设计](../../M2-auth/M2-design.md) 13.2 中接收者含 M8 的各行和 §16；[M0-P2 的交接](../../M2-auth/handoffs/M0-P2-platform-notes.md)；[M0-P6 的交接](../../M2-auth/handoffs/M0-P6-e2e-notes.md)；[M2 收尾 spec](../../M2-auth/specs/closeout.md) 第 3 节。
