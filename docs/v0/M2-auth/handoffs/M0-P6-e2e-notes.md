---
status: closed
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

## 处理结果（M2/P3a）

- **PAT 对等验收和认证 fixture 的 PAT**（完成）：`e2e/fixtures/auth.ts` 加上 `createPAT` 和 `bearer`；A7–A11 的接口版本只用 PAT 调接口，调用 `e2e/fixtures/assert/identity.ts` 的断言函数（`accountOf`、`tokensOf`、`expectPasswordChanged`、`expectTokenStored`），页面版本以后调用同一组函数。
- **S3**（完成）：`toEqual` 加上 `signup_enabled`、`workspace_creation_enabled`、`file_size_limit` 的期望值。

仍未处理，状态保持 `open`：页面的登录状态（M2/P4）；S2 的断言（M2/P4）；River 停机与 fixture 的预算（M2/P3b）；fixture 写法的延伸（M4、M5、M8）。

来源：[M2/P3a spec](../specs/P3a-account-api.md) 第 7 节。

## 处理结果（M2/P3b）

- **River 停机与 fixture 的预算**（完成）：River 运行时实测 nerve 从收到 SIGTERM 到退出的时间（`bin/nerve serve`，test 配置，每种情形 10 次，依次是最小、中位、最大）。Task 11 的两轮（改前的 runner）：就绪后立即停机 3.5、4.3、4.6 毫秒，另一轮 1.5、2.6、4.6 毫秒；就绪 3 秒后停机 4.1、9.1、17.6 毫秒，另一轮 3.6、8.1、11.4 毫秒。3 秒后停机的那 10 次中，停机前清理任务完成过的只有 8 次：River 选出 leader 之后，它的维护服务逐个错开启动，定时任务的第一次投递可能晚于 3 秒。C1 调查的 210 次（改前的 runner 150 次、改后的 60 次，就绪后 0–30 秒停机）都在 3.3–21.7 毫秒之间，都以 0 退出。这些都远在 fixture 的 `stopTimeoutMs`（30 秒）和 worker 的预算（`nerveFixtureTimeoutMs = readyTimeoutMs + stopTimeoutMs + 10_000`，70 秒）之内。按配置算的最坏情况（HTTP 20 秒、任务 10 秒加 1 秒、连接池 5 秒，共 36 秒）超过 `stopTimeoutMs`：那时 fixture 在 30 秒时 SIGKILL，报出带日志路径的错误，不会挂住；停机要这么久本身就是缺陷。

  停机落在 River 启动后的最初几秒内时，River 会记 ERROR：`maintenance.PeriodicJobEnqueuer: Error starting transaction`（`context canceled`）；River 自己还没启动完时还有 `notifier.Notifier: Error running listener … conn closed`。这两处 River 不看取消的原因，runner 无从避免：改后的 runner 在 River 启动之后只调用 River 自己的 `Stop`，出现的比例不变。C1 调查中至少有一条 River ERROR 的运行，改前、改后依次是：就绪后立即停机 10/10、10/10；就绪 3 秒后 7/40、4/40（Fisher 检验 p = 0.52，是噪声）；就绪 10 秒后、`river_queue` 从 7 秒起被锁 1/10、0/10；运行 10 秒、30 秒后停机（test 配置和默认间隔各 20 次，只测了改前的 runner）0/40。所有运行都以 0 退出，没有任务停在 `running`，没有连接泄漏，没有写了一半的数据。所以 worker 的 nerve 日志里（尤其是 `nerveWith` 另起、很快又停的 nerve）出现这两条不是失败。
- **命令的标准输入**：`runNerve` 接受标准输入和额外的环境变量，写完就关闭标准输入，读密码的命令拿到文件结尾，不会一直等；`e2e/fixtures/users.ts` 的 `nerveUsers`、`nerveUsersFails` 在 worker 的库上运行 `nerve users`（A12、A13、A16、A17），日志开到 DEBUG：`nerveUsers` 核对命令确实记了日志，两者都核对输出和日志里没有密码（原文、十六进制和两种 base64）。
- **新等待的期限**：A14 用 `expect.poll` 等清理任务删掉过期的会话，以 15 秒为限（test 配置的间隔是 2 秒）。

仍未处理，状态保持 `open`：页面的登录状态（M2/P4）；S2 的断言（M2/P4）；fixture 写法的延伸（M4、M5、M8）。

来源：[M2/P3b spec](../specs/P3b-jobs-and-admin.md) 第 7 节。

## 处理结果（M2/P4）

- **页面的登录状态**（完成）：fixture `signedInPage(tokens, baseURL?)` 给出一个已经登录、还没有加载任何东西的页面：页面第一次加载时，在它的任何脚本运行之前，`e2e/fixtures/auth.ts` 的 `signInContext` 把一条新的 `nerve.auth` 记录写进 localStorage；之后的加载、刷新和同一上下文里的其他标签页不再写，页面自己续期、退出。A4、A5、A6 的页面版本和 A10 整页加载的那个页面测试用它；A1–A3、A15 的页面版本和 A10 的另一个页面测试（注册之后在应用内进入 `/onboarding`）从登录页、注册页开始。A6 的切换 ① 用同一文件的 `writeRecord` 像令牌管理器一样在锁下换上另一个账户的记录。页面版本调用接口版本的同一组断言函数（`expectRegistered`、`expectSignedIn`、`expectRefreshed`、`expectRevoked`、`accountOf` 等）。
- **S2**（完成）：`e2e/fixtures/browser.ts` 的 `watchPage` 记录页面的接口请求、失败的接口请求、发往旧接口（`/api/v0` 之外）的请求、未处理的异常、控制台的错误和警告、CSP 违规；同一文件的 `expectQuietConsole` 先让页面写一条探针错误和一条探针警告、要求收到（`watchPage` 没有接上控制台时不会空过），再断言控制台没有别的错误和警告。S2 断言未登录的页面只请求 `GET /api/v0/instance`，失败的接口请求、未处理的异常和 CSP 违规都为空，控制台没有错误，也没有警告（原文"是否断言控制台没有错误"：断言；深链接点名允许一条第三方的警告：它的页面模块加载编辑器，`is-emoji-supported` 反复读画布，Chromium 提示 `willReadFrequently`）；首页带 `Content-Security-Policy`；深链接跳到带 `next_path` 的登录页。S2 不再用 `networkidle`，改为等待具体的状态（原文的建议）：`GET /api/v0/instance` 的回答和登录页的"Go to workspace"按钮。页面不读回答的请求一直算在途中，`networkidle` 会等到测试超时，而不是在断言上失败（P4 Task 12 实测）。A2、A4、A5、A6 和 A10 的两个页面测试也用 `watchPage`。
- **新等待的期限**：页面故事的 `waitForResponse` 都带 `{ timeout: 10_000 }`（S2；A2 的 `showSignIn`；`e2e/fixtures/auth-pages.ts` 的 `submitSignIn`、`submitSignUp`；`e2e/fixtures/onboarding-pages.ts` 的 `saveProfileStep`）；A4 的 `holdFirstRefresh` 最多扣住第一个续期请求 1 秒，等访问令牌过期的 `expect.poll` 以 10 秒为限；`expect` 的断言（包括 `expectQuietConsole` 的 `expect.poll`）受 Playwright `expect` 的超时约束（默认 5 秒）；导航、点击和 `page.evaluate` 没有自己的期限，由测试的超时（默认 30 秒，`nerveWith` 每起一个 nerve 再加它的预算）兜底，超时就失败，不会挂住。

仍未处理，状态保持 `open`：fixture 写法的延伸（M4、M5、M8），由收尾转交给这些 M。

来源：[M2/P4 spec](../specs/P4-web-auth.md) 第 7 节。

## 处理结果（M2/收尾）

逐节的结论：

- **PAT 对等验收与认证 fixture**：M2/P1–P4 完成（`e2e/fixtures/auth.ts`）。
- **数据库断言**：M2/P1 完成（`e2e/fixtures/assert/`）。
- **S1、S3**：M2/P1、P3a 完成。
- **S2**：M2/P4 完成。
- **端口与停机**：M2/P1、P3b 完成。
- **录像**：M2/P1 定为不录像。
- **fixture 模式的延伸**：不属于 M2，转交：`clock.ts` 给 M4（[M4 的交接](../../M4-issue-core/handoffs/M2-closeout.md)第 13 节），`storage.ts` 给 M5（[M5 的交接](../../M5-files/handoffs/M2-closeout.md)第 5 节），`webhook.ts` 给 M8（[M8 的交接](../../M8-open-release/handoffs/M2-closeout.md)第 5 节）。

收尾的头上 `make e2e` 48 个测试通过（收尾 spec 附录 A）。全部有结论，状态改为 `closed`。

来源：[M2 收尾 spec](../specs/closeout.md) 2.1。
