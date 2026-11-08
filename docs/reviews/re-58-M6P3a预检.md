# M6 / P3a 预检

> 2026-11-08：预检通过，可以按 P3a 的 T1 开始实施。目标：[P3a spec](../v0/M6-cycles-modules/specs/P3a-issue-membership.md)、[P3a plan](../v0/M6-cycles-modules/plans/P3a-issue-membership.md)，依据 [M6 设计](../v0/M6-cycles-modules/M6-design.md)。基线 main `ddd79a9f5362aedfc33c940acecc2a5bf0e3d9b8`（P2 的根因修复 noff，main CI 42458454710 首次通过）。本轮只有文档，没有产品修改或测试运行。

## 1. 范围与方法

先由两个只读调研代理对照 main 的代码写出现状笔记（工作项的写入路径、草稿、事件、删除与清理守卫、自动归档、planning 与 project 的端口；列表引擎、排序夹具、偏好与 Plane 的列表行为），随阶段台账归档。据此把设计 §12 的 P3 拆为 P3a 与 P3b，写出 P3a 的 spec 与 plan（`ae0ef51f`）。随后一次 fresh 只读审查（`fable`）逐条对照分支上的实际代码核查：事实与计数是否准确、按次序执行时有没有守卫在本 Task 内变红而计划不修、R1–R10 的设计是否站得住（尤其“已结束”在读时钟之后判定与硬规则 §4 的关系）、是否有须负责人决定的事、A1–A8 的覆盖。原始报告没有另存文件，逐条内容与处理见 §3。

## 2. 起点事实

- `issue` 的建立与草稿发布共用 `IssueCreation`（`check` 收集全部 422，`write` 写行、动态与事件）；PATCH 由 `Diff`/`Changes` 驱动，标签读加入与移出两组、只检查加入的。三条路的时钟都来自资产锁之后（`LockContent`/`LockDraftAssets`），建立只读一次时钟由测试固定。
- `issue` 的语句不能连 `planning` 的表（`TestSQLCSchemaScope`），外键可以指向它们（先例 `issue_labels → labels`）；动态里的迭代、模块名字只能来自端口或事件。
- `planning` 只交出 `Provided{Purger}`，不发布事件；`planning.Provide` 在 bootstrap 里排在项目端口之后、`issue.New` 之前；项目的 `cycle_view`、`module_view` 默认 `false`。
- 两张新表名不以 `issue_` 开头，bootstrap 的 `projectIssueTable` 与 e2e 的 `asynchronousDeletionTable` 不改就会把它们当作同步删除；删除守卫的 `keysTo` 由目录推导，建表即要求它们有行。
- Plane：删迭代只写一条迭代动态；加入已结束迭代回答 400；`module_ids` 不含已归档模块；建立弹窗另调加入接口，历史里有加入的动态。

## 3. 预检结论与处理

结论：R1–R10 站得住。“已结束”在最后一把锁与读时钟之后判定，与 P1 改迭代的做法一致，也是建立只读一次时钟的约束所迫；planning 经未加锁的项目事实读取，在工作项写持有的项目行锁下稳定；事件在 planning 的删除事务里同步分发，所有写一个项目的工作项或归属的写者都先取项目行锁，不会交叉等待。没有须负责人决定的事项。处理（均已写入 spec/plan）：

| 编号 | 级别 | 问题 | 处理 |
|---|---|---|---|
| H1 | High | spec §2.5 写 `deletedWorkItem`、`fullIssue`、`furnishDeletionIssue` 经接口加归属，但归属的接口要到 T6/T7 才有；T3 的删除守卫（`issue_deletion_interleaving_test`、`issue_deletion_test`、`project_deletion_test`）建表后即要求两张表有行，照原文 T3 不能转绿 | Go 的夹具一律以 SQL 种归属，排在迭代与模块的种子之后；`seedIssues` 按项目查出 `seedCycles`/`seedModules` 种下的迭代与模块；测试里新建的项目各种一份 |
| M1 | Medium | `planning.Provide` 有五处测试调用，plan 只列三处（漏 `planning/adapter/postgres` 的 `module_rows_test`、`store_test`） | T5 Files 补全；`Provide` 只需项目事实的读取，清理不用它，可传 nil |
| M2 | Medium | e2e 的 `furnishIssueRows` 不只是调整次序：开关只在 `furnishPlanningRows` 里打开，它不回答 id；各删除故事都先 issue 后 planning 且令牌不同 | `furnishPlanningRows` 回答迭代与模块的 id，`furnishIssueRows` 以项目成员的令牌经接口加入；`furnishWorkspace` 与各删除故事改次序，T9 Files 列入 |
| M3 | Medium | `apitest.Main` 按（操作，码）核对，不是每个码一个 | spec §2.3 与 plan 写明：建立与 PATCH 各 4 个、发布草稿 4 个处理器测试 |
| M4 | Medium | 迭代动态的渲染器移出分支读 `new_value`，按 §2.4 的写法会显示空名（设计 §8 已规定改读 `old_value`） | T6 的前端一步改这一行并加单元测试；spec §1 与 §2.4 写明 |
| L | Low | R2 应写明为何不再读一次时钟（`issue_event_writes_test.go:120` 固定一次）并点名两个判定点；§6 的锁序说法（工作项写不显式锁工作项行）；R6 的先例写成了自己；R3 移出的目标不检查但要读名字；清理步骤在 `module.go` 的 `Provide`；手写字面量是 13 个文件；`purge_runtime_test` 的老化清单要同步；缺同一工作项两个并发 PATCH 的竞态用例；T5 的端口一时没有调用者可以接受 | 逐条改正；竞态用例加入 T6 |

## 4. 计划覆盖

P3a 共 13 个 Task：T1 P2 回填、拆分与阶段；T2 两张表与表级守卫；T3 删工作项、删除任务与清理的两步与删除守卫；T4 回答里的两个字段（含前端类型与字面量）；T5 planning 的目标端口与未结束的 id；T6 迭代归属；T7 模块归属；T8 草稿；T9 删迭代/模块的事件与订阅者，第一个检查点；T10 自动归档；T11 接口故事；T12 文档与交接回执；T13 门禁、整审与集成。spec §4 的 A1–A8 分别由 T6/T8（A1）、T7/T8（A2）、T2/T3/T9（A3）、T10（A4）、T6/T9（A5）、T4（A6）、T1/T6–T8/T12（A7）、T13（A8）覆盖。安全、加锁与访客性质在 T2–T7、T9、T10 各有变异。
