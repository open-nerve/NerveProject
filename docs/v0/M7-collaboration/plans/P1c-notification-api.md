# M7 / P1c plan：通知的接口与读

> spec：[P1c spec](../specs/P1c-notification-api.md)（预检 [re-84](../../../reviews/re-84-M7P1c预检.md) 修正后通过）。基线 main `1a23f6f8`。worktree `.codex/worktress/codex/m7-p1c`，分支 `codex/m7-p1c`。台账 `.superpowers/sdd/P1c-notification-api/progress.md`（ignored），现状调研 `research/p1c-facts.md`。

## Global Constraints

- spec §2 是合同，§3 的 R1–R7 是裁定；超出 spec 的契约或架构改动先停下报告。P1c 不加迁移与索引；要加先停下报告。
- 契约先行：`api/modules/notification.yaml` 改了就 `make gen`（Go 与网页），`make gen-check` 无差异；逐操作的码各有处理器测试回答；查询参数都是字符串（spec §2.3），不改网页客户端的写法。
- 模块之间只经端口：notification 不 import 别的模块；access、project、issue 交出的端口由 bootstrap 转换（`notification_ports.go`）。SQL 只经各模块自己的 sqlc，`TestSQLCSchemaScope` 不加例外；夹具不进产品包；手写文件约 400 行以内。
- 访客规则：判定只有 `domain.Issue.HiddenFrom` 一个谓词；通知 SQL 里的那一个过滤（spec R2）只在列表、未读数、全部已读三条语句里，由六格等价测试钉住；单条修改走 `GuestRule` 端口。
- 允许改的已有测试（改之前在台账逐条写明，每处只照这里写的改法）：
  - `bootstrap/actions_test.go`：`notification` 移出 `actionlessModules`、加进 `moduleActions()`。
  - `access/domain/rules_test.go` 的 `tableCells`：两个新动作各一行。
  - 权限矩阵：`matrixRows()` 加 `notificationMatrixRows()`；种子（`newSeeded` 的按列 id、`prepareMatrix` 插通知行、`workspaceOfRow` 认得 `notifications`）；不改已有行与断言。
  - `access_decide_many_test.go` 的第二遍造状态：抽成与 T2 的等价测试共用的辅助，断言不改。
  - P1b 的通知夹具与 `notificationDeps`（`Deps` 加字段）；`notification_deletion_test.go` 直接构造 `notification.Deps` 的那一处补新字段或改经 `notificationDeps`。
  - 生成物（Go 的 `gen/`、网页的 `schema.gen.ts`）；`error-messages.ts` 与两份 `errors.json` 加新码（`error-messages.test.ts` 要求与契约的码集相等）。

  其余 M4–M6 与 P1a、P1b 的断言是回归，不改。
- 红的窗口（其间这几项不作为证据，台账记下）：路由与契约的守卫（`TestAPIRoutesAreTheContractsOperations`）、处理器逐码的完整性（`apitest.Main`、`TestEveryModuleRunsItsHTTPTestsThroughMain`）T6 → T8；权限矩阵的覆盖守卫（`TestThePermissionMatrixCoversEveryOperation`）T6 → T9。
- 证据：每次检查经 `run-check.py`，变异经 `mutate.py`，失败的运行也留着；每个可见、幂等、锁的性质至少一个能让测试失败的变异。
- 测试时间：CI 第 0 路 Test 步的总增量 ≤ 90 秒（折合本机约 60 秒，含 bootstrap 以外的包）；两个检查点记本机两路与分支 CI 两路；第 0 路超过 870 秒时在阶段记录里向负责人提出加一路（设计 §9）。

## Tasks

### T1 基线与文档（主控）
P1b 阶段记录 §9 回填集成、设计 §14 的 P1b 一行；spec、plan、预检 re-84；设计 §14 加 P1c 进行中。台账写基线。

### T2 读的人看得到的项目（access、project）
- project：调用者在一个工作区里有有效项目成员资格的、未删除的项目（含已归档）的事实与成员资格，一次查询（sqlc）。
- access：`NewReadableProjects(deps)` 的 `Readable(ctx, actor, workspaceID)`，逐项目以 `domain.Decide`（`issue.read`）判定，只接受不放行非成员的规则。
- 等价测试两遍（spec §2.5），第二遍的造状态与 `TestDecideManyIsAuthorizeOnTheMatrixSeed` 共用辅助；每遍三种答案都出现。变异：漏已归档的项目、漏成员资格的 `is_active`、漏 `guest_view_all_features`、把已删除的项目算进来（第二遍杀）。

### T3 另一端的解析（issue、project）
- issue 交出 `IssueRefs(ctx, reader, grants, ids)`（spec §2.6）：`GetIssues` 之上按 `grants` 与 `HiddenFrom` 过滤，已归档的照给。
- project 加按 id 数组读未删除项目标识的一条查询；bootstrap 的转换一页一次。
- 测试：读得到、只看自己的访客看不到别人建的、已删除的、不在集合里的项目、别的工作区；关联类型全在引用集合里（bootstrap 测试，去掉一种即红）。变异：漏 `HiddenFrom`、漏已删除、把已归档的工作项去掉。

### T4 领域规则（纯函数）
筛选参数的解析与校验（字符串的 `tab`、`type` 子集与重复与 `tab=mentions` 的 422、`read`、`archived`、`snoozed`、未知参数）；游标 `[created_at, id]`；修改的规则（已读、归档的幂等、`snoozed_till` 的将来与截到微秒、`{}`）；回答里 id 的白名单与引用字段（spec §2.4、R7）。表驱动单元测试，每条规则一个变异。

### T5 存储（notification 的 sqlc）
列表（可见的数组、筛选、游标、`limit + 1`）、未读数（两个页签各一个数）、单行的不加锁读与 `FOR NO KEY UPDATE` 的锁读、单行更新、全部已读的一批（带锁子查询 `id > after ORDER BY id LIMIT 1000 FOR NO KEY UPDATE SKIP LOCKED`，条件在子查询里，`RETURNING id`，语句头写不变式）。adapter 测试：筛选与游标、被锁的行被跳过、`created_at <= now` 的界、记一次 1 万条未读时单批的耗时（不设门槛）。变异：游标比较反向、漏 `deleted_at`、去掉 `SKIP LOCKED`、条件移出子查询。

### T6 契约、动作与网页的码
`api/modules/notification.yaml`（spec §2.2–§2.4，参数都是字符串，逐操作的码）、`api/openapi.yaml` 的标签与路径、`make gen`；`notification.read`、`notification.write` 进规则表与 `rules_test`；`actions_test` 的移动；`notification.not_found` 进 `error-messages.ts` 与两份 `errors.json`。红的窗口从这里开始（Global Constraints）。

### T7 用例
四个用例与 `app/ports.go`：列表与未读数（按 slug 找工作区 → `Authorize` → 一个 `ReadTx` 里取可见的集合、读行、解析引用）；单条修改照 spec §2.7 的七步次序；全部已读的批循环（请求开始时取一次可见的集合与 `now`）。单元测试钉住调用次序、404 的逐值相同与 404 先于 422。

### T8 处理器与接线
`adapter/http`（处理器、`Register`、`given` 式的三态解码、参数的严格解析）、`TestMain` 的 `apitest.Main(m, "notification")`、bootstrap 的接线（`Deps` 的事务（单条修改带锁的期限与 503 的映射）、时钟、授权、三个读端口）。逐码的处理器测试。路由与逐码的守卫转绿。

### T9 权限矩阵
spec §2.8 的行与种子；覆盖守卫转绿。列表、未读数、全部已读的 `check`；单条修改的两行（带与不带 `prepare`）。

### T10a 组合测试：列出与筛选
经真实的 app：H2–H4 的各类写以 `Notification` 列出；H5 的页签、`type`、`read`、`archived`、`snoozed`、跨页无重无漏、`limit`、坏游标、未知参数。

### T10b 组合测试：可见、另一端与 R2
可见的各种情形（spec §4）；另一端的给出与 `null`、白名单之外不给 id；R2 的六格乘三条 SQL 对 `HiddenFrom`。

### T11 组合测试：修改、暂缓、全部已读与未读数 → **检查点 1**
H6、H7（spec §4）；四种 404 与随机 id 逐字节相同；404 先于 422；暂缓到期用业务时钟的偏移或可推进的时钟；全部已读的跨批、页签与筛选、事件晚于 `now` 的不标、迟到的行写死。**检查点 1**：七项本地门禁、分支 CI、两路时间写进台账。

### T12 交错
子树删除持有通知行时全部已读立即返回（`marked` 不含被持有的行、没有 40P01）；全部已读与项目删除的分批软删；单条修改与工作项删除（两种先后）；单条修改等锁超时 503。语句名的锁探针（P1b 的 `WaitForStatementLockWaitOn`）。变异：去掉 `SKIP LOCKED`、单条修改不带锁的期限。

### T13 e2e → **检查点 2**
H2–H7 的接口故事（`e2e/stories/collaboration/`，H1 的写法；等分发写出用 `expect.poll`，导出 P1b 的 `notified` 或写同样的等待）。**检查点 2**：七项本地门禁、两路时间。

### T14 文档与交接（主控）
plane-diff（地址与动词：单条修改合并了 Plane 的四个已读 / 归档端点，没有单条的读与删除；页签与 `{all, mentions}`；全部已读的范围与 `{marked}`；暂缓的方向与未读数计入已到期的；排序与分页；回答里没有 `triggered_by_details` 等；另一端的可见）；硬规则 03 §4 登记 R2 的 SQL 例外与 `notifications` 各个写者的锁序（R5）；M4 交接 M7-02、M5 交接 M5-M7-02 的“处理结果（M7/P1c）”与关闭；设计 §3.4、§5 的 `{all, mentions}`、H7 的措辞与 §14；阶段记录（向负责人说明改名）。

之后：整分支审查（fable，只读）→ 修复轮 → 限定复审 → 最终七项 → 分支 CI → `--no-ff` 合入 main → main CI → 归档、清理、记忆。

## Review Focus

- 可见的集合与 `Authorize` 的等价（含已归档的项目，两遍）；只看自己的 SQL 过滤与 `HiddenFrom` 的六格等价。
- 不透露：四种 404 与随机 id 逐字节相同、404 先于 422；看不到的另一端连 id 也不给；回答里的 id 按白名单。
- 全部已读的 `SKIP LOCKED` 与子树删除；单条修改的锁期限与 503。
- 修改的三态解码与幂等；暂缓按业务时钟。
- 测试时间的预算（CI 第 0 路）。
