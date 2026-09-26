---
status: open
from: M0/P6
to: M2
created: 2026-09-22
---

# M2 扩展端到端测试骨架时的注意事项

## PAT 对等验收与认证 fixture

- 每个故事的 PAT 版本：用 PAT 调接口走完同样的流程，调用同一组数据库断言函数（总体设计 8.2）。
- 需要一个认证的 fixture：通过接口注册、登录，让页面使用登录状态；也要能只用 PAT，不经过页面。

## 数据库断言

- `fixtures/db.ts` 的 `query`（M0 每次调用新建一个连接，执行完关闭）在数据库断言变多以后，改为每个 worker 一个连接池，并建立断言函数的统一写法。
- 失败时保存本 worker 数据库的快照（`pg_dump`），和 trace、截图、nerve 日志一起归档。

## S1："迁移版本正确"

- 真正的迁移文件出现后，S1 加上"迁移版本正确"的断言：`nerve migrate status` 列出的全部迁移都是 applied，`goose_db_version` 的最新版本等于最后一个迁移文件。
- **`migrate status` 只有从 M2 起才是真正的数据库检查**：M0 没有迁移文件，`postgres.Migrator` 的 `provider` 为 `nil`，`Status` 直接返回、不连接数据库（P6 spec 2.5，Minor 5 修复）；S1 在 M0 即使连错数据库也能通过 `migrate status` 这一步。M2 加入迁移文件后，这一步才真正连接数据库、才能发现连错库的问题。
- **模板库的不变量继续保持**：只有全局准备（`global-setup.ts`）连接模板库、执行 `migrate up`；worker 只做 `CREATE DATABASE … TEMPLATE`，不再迁移。`CREATE DATABASE … TEMPLATE` 在模板库上有其他会话时最多等待 5 秒（Postgres 的行为），全局准备完成迁移并关闭连接后才应该让 worker 开始复制。`pool.Close()` 必须保持 `defer`，迁移完就释放连接，不要提前或遗漏。

## S3：`signup_enabled`

- `/api/v0/instance` 加入 `signup_enabled` 字段之后，S3（`s3-instance-info.spec.ts`）的 `toEqual` 断言要随之更新，加上这个字段的期望值。

## S2：接口不再 404

- 前端改调 `/api/v0/instance`（P5 交给 M2 的 [M0-P5-frontend-api-notes](M0-P5-frontend-api-notes.md)）之后，`/api/instances/` 的 404 消失：S2 可以加上"接口请求没有失败"的断言，并决定是否断言控制台没有错误（React #418 另由 M1 处理，和这里无关）。
- S2 目前用 `networkidle` 等待页面加载完成；一旦前端改为轮询（比如定期刷新数据），`networkidle` 永远不会触发，要改成等待具体的 UI 状态。

## 端口与停机

- 端口竞争的临时缓解（P6 最终评审 Minor 3，`a7d466d`）：`/readyz` 返回 200 之后再等一个轮询间隔，确认子进程仍在运行。根本解决办法留给 M2：nerve 在 `:0` 上监听并报告自己实际绑定的地址（比如写一个地址文件），fixture 直接读取，不用先猜端口再等待。
- fixture 里的每一次等待都要受 fixture 自己的期限约束，不能只在两次等待之间检查期限（M0 对抗性评审 Important 2，M0 加固已修）：`waitUntilReady` 的每个 `/readyz` 请求都用剩余时间做 `AbortSignal.timeout`；nerve 没有就绪时，先 SIGKILL 并等它退出，再报出带日志路径的错误。全局准备在任何 Playwright 超时生效之前运行，所以 `runNerve` 超过 60 秒就用 SIGKILL 结束命令；`db.ts` 连接数据库最多等 10 秒，每条查询最多 30 秒（pg 默认一直等）。M2 新增的等待（认证、快照、地址文件等）照此办理。
- River 的后台任务接入后，重新核对 nerve 的停机时间是否还在 fixture 的超时预算（`readyTimeoutMs + stopTimeoutMs + 10` 秒）之内；River 的 graceful shutdown 可能比 M0 单纯的 HTTP 优雅停机慢。

## 失败时的录像

- 是否录像（v0 §8.2 列出了这一项）还没有决定：M0 只保存 trace、截图和 nerve 日志，trace 里已经有每一步的截屏，录像还要多下载 ffmpeg。M2 开始有业务数据和更长的用户流程时，视情况决定是否加上，或者干脆修改 v0 §8.2 的措辞。

## fixture 模式的延伸

- `storage.ts`（M5，检查文件是否真的写入本地存储）、`webhook.ts`（M8，本地 Webhook 接收端）都沿用 P6 定下的 fixture 写法：普通函数 + `test.ts` 接成 Playwright fixture，全局准备也能直接调用普通函数。
- 可注入的时钟（M4，用于"时间流逝"类故事，比如自动归档）同样按这个模式加一个 `clock.ts`。

来源：[M0/P6 评审记录](../../M0-foundation/reviews/P6-e2e-ci-review.md)。

## 处理结果（M2/P1）

- **认证 fixture**（注册完成）：`e2e/fixtures/auth.ts` 提供 `password`、`emailFor`、`register`；A1、A2 的接口版本调用 `e2e/fixtures/assert/identity.ts` 的断言函数。
- **数据库断言**（完成）：每个 worker 一个 `pg` 连接池（`openDatabase`），断言函数按表放在 `e2e/fixtures/assert/`。测试失败时，自动 fixture `databaseSnapshot` 用 `docker exec … pg_dump` 导出本 worker 的库到 `database.sql`，作为附件和 trace、截图、nerve 日志放在一起。
- **S1**（完成）：核对 `nerve migrate status` 的每一行都是 `applied`、来源依次等于迁移文件，`goose_db_version` 的最大版本等于最后一个文件的序号。模板库仍只由全局准备连接。
- **端口与等待**（完成）：nerve 在 `127.0.0.1:0` 上监听，把实际地址写进 `server.addr_file`，fixture 读取；空闲端口的猜测和"就绪后再等一个轮询间隔"的缓解一起删除。读地址文件和轮询 `/readyz` 共用 30 秒的期限，每个请求只用剩余时间；`pg_dump` 超过 60 秒就 SIGKILL；`nerveWith` 另起的 nerve 把自己的预算加到测试的超时上。
- **录像**（完成）：不录像，总体设计 8.2 已改为"trace、截图、nerve 日志和数据库快照"。

仍未处理，状态保持 `open`：PAT 对等验收，认证 fixture 的登录、PAT 和页面的登录状态（M2/P2–P4）；S3 的 `signup_enabled`（M2/P3）；S2 的断言（M2/P4）；River 停机与 fixture 的预算（M2/P3）；fixture 写法的延伸（M4、M5、M8）。

来源：[M2/P1 spec](../specs/P1-platform-core.md) 2.16。

## 处理结果（M2/P2）

- **认证 fixture 的登录**（完成）：`e2e/fixtures/auth.ts` 加上 `login` 和 `refresh`；A3、A4、A5、A6、A15 的接口版本调用 `e2e/fixtures/assert/identity.ts` 的断言函数（`expectSignedIn`、`expectRefreshed`、`expectRevoked`、`sessionOf`）；A15 用 `nerveWith` 另起一个限流很低的 nerve。
- **新等待的期限**：P2 没有新增等待；A15 另起的 nerve 沿用 `nerveWith` 的预算。

仍未处理，状态保持 `open`：PAT 对等验收和认证 fixture 的 PAT（M2/P3）；页面的登录状态（M2/P4）；S3 的 `signup_enabled`（M2/P3）；S2 的断言（M2/P4）；River 停机与 fixture 的预算（M2/P3）；fixture 写法的延伸（M4、M5、M8）。

来源：[M2/P2 spec](../specs/P2-sessions.md) 第 7 节。
