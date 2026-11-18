# M7 / P1c 阶段记录：通知的接口与读

> 状态：实施完成，待整分支审查与集成。spec 与 plan 见 [P1c spec](../specs/P1c-notification-api.md)、[P1c plan](../plans/P1c-notification-api.md)，预检 [re-84](../../../reviews/re-84-M7P1c预检.md)（修正后通过，8 处重要、9 处次要全部采纳）。基线 main `1a23f6f8`（P1b 的 noff）。

## 1. 交付

- 四个接口：`GET /workspaces/{slug}/notifications`（两个页签、`type`、`read`、`archived`、`snoozed`、键集分页）、`GET …/unread-count`（`{all, mentions}`）、`POST …/mark-all-read`（`{marked}`，每批 1000 行、`SKIP LOCKED`）、`PATCH /notifications/{id}`（`read`、`archived`、`snoozed_till`）。动作 `notification.read`、`notification.write`（工作区级，三种角色），码 `notification.not_found`。
- 读时的可见：access 的 `ReadableProjects`（含已归档的项目，与 `Authorize` 同一判定，两遍等价测试）；只看自己在三条 SQL 里（六格等价测试钉在 `HiddenFrom` 上），单条修改经 `GuestRule` 端口。
- 另一端：issue 的 `IssueRefs`（读得到才给，含已归档的）与 project 按 id 数组读标识，一页一次；回答里的 id 按白名单；关联类型的守卫。
- 权限矩阵：四个操作五行 30 格，每列一条种好的通知与访客看不到的那一条，列表等三行带 `check`。
- 组合测试、交错（I1 的子树删除与全部已读）、三条 e2e 故事（H2–H7，PAT 与浏览器会话对等）。

## 2. Task 与提交

| Task | 提交 | 内容 |
|---|---|---|
| T1 | `86d6d6a4` | spec、plan、预检 re-84；P1b 阶段记录 §9 与设计 §14 |
| T2 | `375da17e` | project 的 `MemberProjectFacts`、access 的 `ReadableProjects`、两遍等价测试 |
| T3 | `176a3a3b` | issue 的 `IssueRefs`、project 的 `NotificationIdentifiers`、bootstrap 的一页一次 |
| T4 | `00f25cd7` | 领域：筛选、游标、修改的三态、回答里 id 的白名单 |
| T5 | `9b797cd4` | 存储：列表、未读数、单行的读、锁与写、全部已读的一批 |
| T6 | `af4b4aae` | 契约、动作、网页的码 |
| T7 | `bf26e0e8` | 四个用例 |
| T8 | `9d6576b5` | 处理器、`Register`、接线 |
| T9 | `e6db0075` | 权限矩阵 |
| T10a | `a892b091` | 组合测试：列出与筛选 |
| T10b | `0c3d2ccf` | 组合测试：可见、另一端、R2 的六格、关联类型的守卫 |
| T11 | `950558ff` | 组合测试：单条修改、暂缓、全部已读与未读数 → 检查点 1 |
| T12 | `7bde59f1` | 交错 |
| T13 | `16b64cb0`、`db6e8e4e` | e2e 故事；主控把暂缓自行到期的余量放宽到 2 秒 → 检查点 2 |
| T14 | 本提交 | 文档与交接 |

## 3. 局部裁定（实施中）

- 主控：关联类型的守卫从 T3 移到 T10b（集合由 T4 定义）；回答里 `change` 的可选键一律 required + nullable（看不到的引用与原本没有的逐字节相同）。
- T2：`ReadableProjects` 的动作固定为 `issue.read`，`memberRule` 只接受项目级、不带 `AllowCreator` 的规则；只读调用者有有效成员资格的未删除项目（含已归档）。access 的转换器搬到 `bootstrap/access_ports.go`（约 400 行的上限）。
- T3：专用查询代替 `GetIssues`（一页只要五列）；可见只在 issue 的 app 层判定，SQL 只管未删除。
- T4：`Filter` 与 `ListFilter` 分开（全部已读没有 `read`），两个入口一套规则；`type` 的重复、空成员都是 `invalid_format`。
- T5：全部已读用 `MATERIALIZED` 的 CTE 而不是 `IN` 子查询：重扫计划下 `IN` 的子查询每扫一次再锁一批，一批会越过 `LIMIT`（探针为证，强制计划的测试钉住）。
- T6：`state_group` 用字符串（`triage` 不在项目的状态组枚举里，快照是历史的）。
- T7：参数校验先于找工作区（与 `listDrafts`、`searchIssues` 一致，不读工作区故不透露）；批长是常量，跨批的测试造 1000 行以上。
- T8：单条修改用 `busyTx{issueTx}`（锁期限 `issue.write_lock_timeout`，超时 503），批用普通事务（`SKIP LOCKED`）；查询参数的检查在 strict 中间件，PATCH 不查查询（照各模块）。
- T9：访客多一条看不到的通知，矩阵就看得见三条 SQL 的只看自己；不为列表等三行加 `prepare` 变体（省 18 个写格，T10b 覆盖）。
- T10a：`apitest.CheckRequest` 会把契约的默认值写回请求（见 §7），P1c 的测试校验副本、发原请求。
- T11：边界（`snoozed_till` 等于此刻、差 1 微秒）用停住的时钟的模块路由；迟到的行在请求读完时刻之后插入仍被标，写死为已知性质（预检 I2）。
- T12：I1 用真实写造出交错的 id 次序；“从不等锁”以语句名的探针加 2 秒的界判断。

## 4. 向负责人说明

- **未读数改名**：设计 §3.4、§5 的 `{total, mentions}` 改为 `{all, mentions}`，只改名字，行为照 Plane：两个页签各自的数，“全部”不含提及（Plane 的 `total` 也不含提及，名字有歧义）。设计的 §2 H7、§3.4、§5 已随之改。

## 5. 本地门禁与 CI

- 检查点 1（`950558ff`）：七项全过；分支 CI 44282836041 attempt 1 全绿。
- 检查点 2（`db6e8e4e`）：七项全过，e2e 484 passed、没有 53300。

## 6. 测试时间与两路

| 时点 | 本机 bootstrap 第 0 路 | 第 1 路 | 合计 | CI server (0) Test | CI server (1) Test |
|---|---|---|---|---|---|
| P1b 基线（`fin2`、main 44179902213） | 523.3 秒 | 500.0 秒 | 1023.3 | 749 秒 | 517 秒 |
| 检查点 1 | 559.2 | 479.2 | 1038.4 | 794 | 541 |
| 检查点 2 | 468.1 | 550.2 | 1018.3 | — | — |

P1c 新增的 bootstrap 测试本机合计约 15 秒；两路的轻重随按名字轮流的分路互换。CI 第 0 路 794 秒，在 870 秒的提醒线以下。

检查点 1 的分支 CI 上 web 任务 903 秒（20 分钟的 75%），Unit tests 761 秒（P1b 时约 470 秒）：turbo 三次都是 16 个任务缓存 9 个，与 P1c 无关的 Lint 步骤也同样慢了约 65%，判为那一次的运行机器慢；最终 CI 再看。

## 7. 交给 M8（M7 收官时写进 M8 的交接）

- **分批锁的写法**：“`IN (SELECT … ORDER BY id LIMIT … FOR [NO KEY] UPDATE)`”在重扫计划下一批可越过上限（结果仍对，每个事务的锁数随计划）。现有 23 条：identity `DeleteExpiredSessions`；issue `delete_rows.sql` 15 条；planning `delete_rows.sql` 6 条；notification `DeleteProjectNotifications`。统一改为 `MATERIALIZED` 的 CTE，并照 P1c T5 的做法加“强制重扫计划下一批不超过上限”的测试（探针与测试的源码在 P1c 台账的归档里）。
- **测试工具的默认值**：`apitest.Contract.CheckRequest` 调 kin-openapi 的 `ValidateRequest` 时没有设 `SkipSettingDefaults`，契约的默认值被写回请求再发出，经 `w.send`、`call` 的 bootstrap 测试从不走服务端的默认值（各列表的 `limit=50`、issue 列表的两个字符串默认等）。改为对副本校验，并补“省略参数走服务端默认”的测试。

## 8. 文档与交接（T14）

`plane-diff.md` 六行（接口、列表、未读数、全部已读、回答、单条修改）；硬规则 03 §4 登记 R2 的 SQL 例外与 `notifications` 各个写者的锁序；M4 交接 M7-02、M5 交接 M5-M7-02 的“处理结果（M7/P1c）”，两条关闭；设计 §2 H7、§3.4、§5 的 `{all, mentions}` 与 §14。

## 9. 整分支审查与集成

（待填。）
