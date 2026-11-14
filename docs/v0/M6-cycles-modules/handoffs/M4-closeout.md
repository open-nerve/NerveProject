---
status: open
from: M4/closeout
to: M6
created: 2026-11-02
---

# M4 交给 M6：迭代与模块的新列表、领域页面验收、删除和键盘边界

这是接收方尚待完成的工作。M4 输入的“转交关闭”只表示责任和验收条件已落实到此处，不表示本页功能已经实现或通过验收。
来源为 [M4 设计 §13](../../M4-issue-core/M4-design.md#13-交接的落点)、[逐条交接索引](../../M4-issue-core/reviews/closeout-handoffs.md)和该索引中的各 Phase spec §5 / review §6。
M4 的最终 C12、准确分支 CI 和未来 main 集成分别由[验收台账](../../M4-issue-core/reviews/closeout-acceptance.md)记录；不以阶段历史绿灯替代最终输入。

<a id="M6-01"></a>
## M6-01 领域自身页面的验收归属

负责人在 P10A-S1 选择 A：P10a 只删除工作项旧列表，迭代/模块自身详情和归档目录保留；它们的完整可用及整页零失败归 M6 接通时验收。当前代价是**没有工作项列表，领域自身接口也未接通**，不是两个已经完成的功能。

- 原始逐请求方法、URL/query、触发、响应、源文件、记录和期限见 [P10a review §6–7](../../M4-issue-core/reviews/P10a-web-issue-types-review.md#6-移交事项)，原件随该 review 的归档入口可查。
- T8 r1 六入口合计 **75 条 API＝69 条父资源 200＋6 条领域 404**，其中包含 M7 视图入口；不是 M6 有效实体通过 75 条。
- 归档 feature flags 默认关闭会让 tabs 不出现；404 后未设置 fetchedMap / ids 为 null 的持续 loader 不能算目录成功。
- analytics/progress 在该次页面观察中 **0 条网络请求、页面状态未验**。保留的真实 store/组件/service 层证明 analytics 带 `?type=issues`、progress 不带该 query，不能升级成有效实体的页面验收。
- T8 合法归档 fixture 暴露的 catch 非 action 写 loader 与嵌套 button，由 M6 的有效实体和失败刷新回归负责。

**何时验收：** 每类领域页面接通时替换自身旧接口和到期例外，删除对应过渡记录；用真实存在、可访问的迭代/模块覆盖详情、统计、归档目录、空态、拒绝和重试，整页零非预期失败。不能永久沿用 P10a 的未实现允许。

<a id="M6-02"></a>
## M6-02 在新引擎恢复工作项列表

- 依据只读 Plane 实现，在 `paged-groups.ts` 和原生列表 API 上恢复迭代/模块列表 store、筛选 store、两个布局根、详情和归档根的列表部分，以及 `EIssuesStoreType.CYCLE` / `MODULE` 的实际分支。
- 恢复分组头“添加已有”、日历 `addIssuesToView`、两域空态、活动迭代/进度侧栏对列表的读取、迭代转移弹窗和 `existing-issues-list-modal.tsx` 的领域调用方。
- M4 已把选择器接到 `issue-search` 并去除正 `tabIndex`；新调用方保留其权限、取消、会话和异步边界。普通 Base 的旧 cycle/module 写链、nested 空壳和无入口 distribution-update/wrapper 已删除；不能恢复这些死算法或 Plane 旧列表引擎。
- 随真实数据加入 `Issue.cycle_id` / `module_ids` 及写入；扩 `issue/domain/listvocab.go`、SQL、共享 `issue-order.json`、`IssueListSettings`、筛选/分组/排序及显示属性“迭代”“模块”。
- 自动归档随实际关系加入迭代和模块条件；保持锁下重读、固定业务时钟、每项目事务及访客可见性。

**何时验收：** 对应列表第一次开放前完成。Go 与前端同 fixture、双方向分页/重复和空页、跨分组写后计数、OwnOnly/管理员兼访客、失权迟到答、归档范围和换会话均走真实引擎；不得通过保留旧 DTO 转换层验收。

<a id="M6-03"></a>
## M6-03 新关系表与删除 owner

`cycle_issues`、`module_issues` 的外键和无条件父索引延伸 M4 §4.16；每次加表同时扩 `furnishIssueRows`、工作项删除语句、M3/M4 删除故事与目录守卫。订阅 `project.deleted` / `workspace.deleted` 在原事务投递自己的清理任务；父范围立即隐藏，任务沿用请求 At/By。普通工作项删除与父项目/工作区删除入口都必须覆盖，不能假定后两者逐项发工作项事件。60 天物理清理增加所属清理者，过期/未过期/邻域、运行角色和回滚重试都有证明。

**何时验收：** 每张关系表合入同步验；M6 收尾前完成全图与有限批次清理。

<a id="M6-04"></a>
## M6-04 表单、焦点和最后一个旧工具读者

- M4 已修共用 `dropdowns/date-range.tsx` 与 `dropdowns/project/base.tsx`；M6 在真实迭代/模块表单三引擎核原生 Tab、日期确认/取消、输入法、Escape、焦点返回、拒绝保留输入。
- `getTabIndex` 现余 7 文件、4 组：M6 的 `cycles/form`、`modules/form`，M7 的 `views/form` 及 inbox create-root / issue-properties / issue-title / issue-description；分组为 PROJECT_CYCLE、PROJECT_MODULE、PROJECT_VIEW、INTAKE_ISSUE_FORM。M4 的 ISSUE_FORM 已删除。
- 迭代/模块归档按钮的正 `tabIndex` 一起清理。M6/M7 **最后一个消费者**改完时，删除 `getTabIndex`、两份 `tab-indices.ts`、barrel、枚举及映射；菜单的非正 -1 不在这个删除范围。
- 本次整期审查在未启用领域留下的具体观察：cycle progress-sidebar 日期提交未等待；cycle peek 表单失败关闭/旧 Escape；module 表单旧日期确认及 progress tab/panel 次序。接通对应页面前，用真实失败/慢答/取消核对并根因修复，不能沿用 M4 页面通过结论。
- `AreaChart` / `ProgressChart` 的单点除零风险只剩 M6 统计消费者；在开放统计前覆盖 0/1/多点，绘图坐标必须有限，空态/单点仍可读。

**何时验收：** 对应 M6 页面启用前；跨 M6/M7 的最后读者在最后一个消费者的阶段删除，并同步另一份交接。原始数量和读者见 [P16 inventory](../../M4-issue-core/reviews/P16-web-cleanup-inventory.md)。

## 处理结果（M6/P1）：M6-03

P1 还没有关系表（`cycle_issues`、`module_issues` 在 P3），`cycles` 本身照本条的协议：订阅 `project.deleted`、`workspace.deleted`，在原事务投递 `planning.delete_rows`，父范围立即隐藏，任务沿用请求的 At/By、有界续跑；`furnishPlanningRows` 进了 `furnishWorkspace` 和 P4、W3、I21 的删除故事，目录 `asynchronousDeletionTable` 加 `cycles`，删除与清理的守卫（`TestDeletingAProject…`、`TestPurgeCatalog…`、`TestPurgeParentCascades…`）都覆盖它；60 天清理的 `planning` 清理者有过期、未过期、邻域、运行角色与回滚重试的证明。工作项删除入口对关系表的覆盖与全图在 P3 及收尾，本交接保持 open。证据：[P1 spec](../specs/P1-cycles-backend.md) 2.4、A6。

## 处理结果（M6/P2）：M6-03

P2 也还没有关系表（在 P3）。模块的三张表照本条的协议加入：`planning.delete_rows` 由单表改为命名步骤 `cycles → modules → module_members → module_links`（照 `issue` 的 `DeletionRowSteps`），沿用请求的 At/By、有界续跑，预算边界按当前步骤判断剩余；`furnishPlanningRows` 再打开 `module_view`，以管理员为负责人和成员建一个带一条链接的模块，P4、W3、I21 的删除故事随之覆盖三张表，目录 `asynchronousDeletionTable` 加 `modules`、`module_members`、`module_links`；删除与清理的守卫（`TestDeletingAProjectLeavesNoUndeletedRowUnderIt`、`TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt`、`TestPurgeCatalogCoversTablesAndIncomingKeys`、`TestPurgeParentCascadesUnprocessedIssueRows`、`TestLateDeleteRowsAfterPurge`）以 SQL 种三张表的行并核软删。60 天清理的新步骤有 `TestPurgeModuleTables`（过期的按页删除，截止时刻的、近期的、未删除的留下）与 `TestModulePurgeChildrenFirst`（子先父），运行角色可执行（`TestPurgeDeletedRunsWithRuntimeRole` 的 planning 组含三张表）。删一个模块时，它未删除的成员与链接在同一事务、同一时刻、同一写者软删（G13 的数据库断言）。工作项删除入口对关系表的覆盖与全图在 P3 及收尾，本交接保持 open。证据：[P2 spec](../specs/P2-modules-backend.md) 2.4、A6。

## 处理结果（M6/P3a）：M6-02

P3a 做了本条中不依赖列表引擎的两项；列表引擎的筛选、分组、排序、显示属性与 `issue-order.json`、偏好在 P3b，页面在 P5–P7，本条其余各项未到。

- **`Issue.cycle_id` / `module_ids` 及写入**：`Issue`、`IssueDetail` 必带两个字段，详情、列表的三种形态、关系、子工作项、搜索的七处投影逐列相同，访客与只看自己的调用者读到的一样（`TestEveryReadAnswersTheMemberships`、`TestIssueMembershipsAcrossRepresentations`；`module_ids` 不排序、读到已离开的迭代、访客读不到迭代的变异都让它们失败）。建立、PATCH、发布草稿经这两个字段加入、移动、移出迭代与模块，每次变化一条动态；接口与数据库故事 `e2e/stories/planning/g4-issue-cycle.spec.ts`、`g11-issue-modules.spec.ts`。前端只重新生成类型、补 `fake-issues.ts` 的字面量、迭代动态的移出分支改读 `old_value`、草稿编辑保留两个字段，列表没有动。
- **自动归档的迭代与模块条件**：经 `planning` 的端口在项目锁与固定业务时钟下读出未结束的迭代（结束日为空或不早于今天，含草稿、未开始的）与目标日未过的模块的 id，语句以 `NOT EXISTS` 排除仍有未删除归属的工作项；仍是每项目一个事务，固定的调用次序为 `Lock → States → Now → OpenTargets → Update`（`TestAutoArchiveProjectTransactionsAndSystemIdentity`），锁与时钟的既有测试照常通过（`TestAutoArchiveReadsClockAfterLock`）；组合测试 `TestAutoArchiveKeepsTheWorkOfOpenCyclesAndModules` 按项目时区跨日，适配器测试 `TestAutoArchiveKeepsTheOpenTargetsWorkItems`。以 UTC 的今天、把今天结束的迭代算作已结束、排除条件漏掉模块的变异（`t10-mut-utc-today`、`t10-mut-cycle-ending-today-ended`、`t10-mut-exclusion-forgets-modules`）都让测试失败。

本交接保持 open。证据：提交 `12bac611efd49c7aa0477194b61aef64d13de169`（回答）、`54d9284892b166bfee5ac91ba6c5b8bd81f0fede`、`650662bd02e9b66e4829bbc156095f3eab8c848f`（迭代与模块的写入）、`d7d1cdb54371960dd2f2d9c98234cf6685961167`、`333e9ba87574eb80c7b682fe73d29453ebaec487`（草稿）、`9e3220594912c9a9d419e8d13154b67473ae8a71`（自动归档）、`dfa04ae47bae651d767f1e33f86aa5c8a890ff60`（故事）；[P3a spec](../specs/P3a-issue-membership.md) 2.3、R9。

## 处理结果（M6/P3a）：M6-03

P3a 加入本条的两张关系表 `cycle_issues`、`module_issues`（归 `issue` 模块；外键与不带条件的父索引延伸 M4 §4.16，见 M6 设计 §4.3），各删除入口与清理都覆盖：

- **普通工作项删除**：`issueRowSteps` 在 `issue_labels` 之后加两步，随子树以原 At/By 软删，不写归属的动态（`TestDeleteIssueDeletesTheWorkItemsSubtreeAndItsRows`、`TestTheCycleStatementOfADeletedSubtreeDeletesItsRows`、`TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt`）。
- **项目、工作区删除**：两张表是 `issue` 的，走 `issue` 已有的 `issue.delete_rows`（订阅 `project.deleted`/`workspace.deleted`，原事务投递，父范围立即隐藏，沿用请求的 At/By）；`DeletionRowSteps` 加两步、剩余探查加两个分支，按批有界、可续跑（`TestDeleteRowsCommitsEachBatch`、`TestDeletionRemainingUsesCurrentCursor`、`TestABatchPassesOverARowDeletedWhileItWaits`）；守卫 `TestDeletingAProjectLeavesNoUndeletedRowUnderIt`、`TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt`、`TestAProjectDeletionRefusedAtItsCommitChangesNoRow`、`TestProjectDeletionPublishesAndEnqueuesAtomically`、`TestLateDeleteRowsAfterPurge`、`TestDeletionInterleaving25` 都要求两张表有行并核软删。
- **删迭代、删模块**：`planning` 在原事务发布 `planning.cycle_deleted`/`planning.module_deleted`，`issue` 的订阅者在同一事务移除归属、写动态（`TestDeletingACycleTakesItsWorkItemsOut`、`TestDeletingAModuleTakesItsWorkItemsOut`，交错 `TestJoiningAndDeletingATargetInterleave`）。
- **夹具与目录**：Go 的守卫以 SQL 种归属（`seedIssues`、`deletedWorkItem`、`furnishDeletionIssue`、`seedPurgeSparseRows`），`projectIssueTable` 认得两张表；e2e 的 `furnishPlanningRows` 回答所建迭代与模块的 id，`furnishIssueRows` 以项目成员的令牌经 PATCH 把工作项加入，`furnishWorkspace` 与 P4、W3、I21 的删除故事改为先 planning 后 issue，`asynchronousDeletionTable` 加两张表；I15、I17、I21 的工作项删除故事同样要求两张表有行。第一个检查点（收据 `t9-gate1`，七项串行全过，e2e 426 条通过）时这些故事转绿；`furnishIssueRows` 不经 PATCH 加入的变异（`t9-mut-e2e-no-join`）让 P4、W2 的删除断言失败。
- **60 天清理**：`issue` 的清理步骤在 `issues` 之前加 `module_issues`、`cycle_issues`；`TestIssuePurgeAllTables` 的 `cycle_issues`、`module_issues` 两个子测试覆盖过期的删除，截止时刻的、近期的、未删除的留下，游标邻域，调用者回滚，失败时已提交的批次保留，`child_order_and_cascades` 核子先父；运行角色可执行（`TestPurgeDeletedRunsWithRuntimeRole` 的 issue 组含两张表）。

删工作项漏掉 `cycle_issues`、删除任务漏掉 `module_issues`、清理把 `issues` 放到两张表之前的变异（`t3-mut-issue-deletion-no-cycle`、`t3-mut-delete-rows-no-module`、`t3-mut-purge-issues-first`）都让测试失败。全图与有限批次清理的规模重跑在 M6 收尾，本交接保持 open。证据：提交 `2ce5591cab00f32200502c05e54b7c1248d02d03`（表）、`f9e62906607ba16f2b0a7ee21ced6db539ce03c8`（删除与清理）、`a37bc6370ae651118bd9cb6204279679fdd5031c`（删迭代/模块的事件与 e2e 夹具）、`dfa04ae47bae651d767f1e33f86aa5c8a890ff60`（G14 故事）；[P3a spec](../specs/P3a-issue-membership.md) 2.5、A3。

## 处理结果（M6/P3b）：M6-02

P3b 做了本条中服务端的列表引擎一项（“扩 `issue/domain/listvocab.go`、SQL、共享 `issue-order.json`、`IssueListSettings`、筛选/分组/排序及显示属性‘迭代’‘模块’”），并给迭代页、模块页各加调用者自己的列表显示设置；页面、store 与网页的词表在 P5–P7，本条其余各项未到。

- **词汇**（`issue/domain/listvocab.go`）：筛选 `cycle_id`、`module_id`，值为 uuid 或 `none`，同一字段取并、不同字段取交，`none` 指没有未删除的归属，已归档的迭代、模块照常算；分组与子分组 `cycle_id`（单值）、`module_id`（多值，同 `label_id`）；这两种只限项目列表，“我的工作项”回答 422；排序 `±cycle_name`、`±module_name` 两种列表都可用。`appendID` 改为按字段分派，`none` 不再落到负责人（变异 `t2-mut-appendid-else` 让 `TestParseListQuery` 失败）。
- **SQL**：三条列表语句（平铺、分组、子分组）各加两个筛选、两个名字键与分组的 `CASE`，三条语句的列与参数复制逐一相同；`issue` 的语句不连 `cycles`、`modules`，名字目录以数组传入（`TestSQLCSchemaScope` 未放宽）。适配器的 `TestIssueListPlanningFilters`（15 个场景 × 三种形态）、`TestIssueListPlanningGroups`、`TestFlatIssueListOrders`，bootstrap 的 `TestIssueListPlanningFilters`、`TestIssueListPlanningGroups`、`TestIssueListAllOrders` 经 HTTP 覆盖成员与只看自己的工作区管理员兼项目访客（OwnOnly）、组与格的续页、两种方向与“我的工作项”。`none` 与 id 取交、筛选的 OR 越出合取而绕过 OwnOnly、OwnOnly 只管行不管组总数、模块只取第一个的变异（`t2-mut-none-intersect`、`t2-mut-cycle-or-escapes`、`t3-mut-ownonly-totals`、`t3-mut-first-module`）都让测试失败。
- **名字目录**：`issue/app` 声明的 `PlanningDirectory` 由 `planning/app/issue_directory.go` 实现（含已归档、不含已删除，`TestIssueDirectoryOfTheProjects`）；`Listing` 只在 `±cycle_name`/`±module_name` 时、在读行的同一只读快照里取（`TestListPlanningNamesSnapshot`），其余排序不取（`TestListOtherOrdersReadNoPlanningNames`；每种排序都取的变异 `t5-mut-every-order-loads` 让它失败）。
- **共用夹具**：`api/testdata/issue-order.json` 加 `cycle_name`、`module_name` 两个场景（共 13 个），同一个提交改 Go 的三个消费者、网页的比较器 `lib/issue-order.ts`（`ordersAllSharedFixtureScenarios`）与 e2e 的 `i4`（27 种排序）；网页生产的目录在 P7 之前是空的，按名字排序时大声失败（`planningDirectoryIsUnavailableUntilP7`）。
- **`IssueListSettings` 与显示属性**：显示筛选、日历与 14 个显示属性（加 `cycle`、`modules`）三页共用（`common.yaml` 的 `IssueListDisplayProperties`，Go 在 `internal/shared`）；项目偏好经迁移 `00045` 认得 `cycle_id`、`module_id` 筛选与两个属性，旧行补 `true`（`TestPlanningDisplayMigration`、`TestIssuePreferencesAPI`）；迭代页、模块页的偏好 `GET/PATCH /api/v0/me/cycles|modules/{id}/preferences` 用同一个 `issue.ListSettings` 校验（`TestPlanningPreferencesAPI`、`TestPageSettingsAndTheirTargetsDeletionInterleave`）。
- **接口故事**：`e2e/stories/planning/g15-issue-lists.spec.ts`、`g15-planning-preferences.spec.ts`，经公共客户端（`cycle_id`、`module_id` 按契约以 CSV 发送，`@nerve/api-client` 的测试对照契约守着），带数据库断言。

本条其余各项留给 P5–P7：迭代/模块列表 store 与筛选 store、两个布局根与详情/归档根的列表部分、`EIssuesStoreType.CYCLE`/`MODULE`；网页的筛选、分组、排序、显示属性的词表与界面（`filter.store.ts`、`issue-filters.ts`、`issue-filter-helper.store.ts`、常量与 `group-columns.tsx`）；目录取数（换掉 `store/issue/helpers/issue-list-binding.ts` 生产目录的空 `Map`）；分组头“添加已有”、日历、两域空态、转移弹窗等调用方。“何时验收”中服务端的部分（同 fixture、两向分页、OwnOnly/管理员兼访客、归档范围）有上列证据；跨分组写后计数、失权迟到答与换会话随页面在 P5–P7 验收。本交接保持 open。证据：提交 `89436a29d92447fc35aaad4530c6ff7b56dc0127`（筛选）、`73fd604d989eb7e264b06250ee121d04eb9241d8`（分组）、`0d7caf28af607d9819edf543c5e786c2c6ff62ba`（目录端口）、`e1ce8ea24bc4467919f4012ff1c7d2400bb88ee6`（排序与共用夹具）、`acc1f8a9fd0ae43513d5b81262edd744015c7323`（共用的显示设置）、`28f7bbb92b3d3f6837580732289fe0f5635df5bf`（项目偏好）、`ba883d875345e586fcee6e2936416771c85084e5`、`79a33c7c83c078ac4664eb0379c0decca4795226`（迭代/模块页偏好）、`b975cabb6e136cb722616a17cb65fce6db910b5c`、`3e94ba1c08c33f96cc3aa357f164697a1cddcc00`（公共客户端按契约发送 CSV 及其守卫）、`55e0398b6312230afdae2631eacb0da333c16542`（G15 故事）；[P3b spec](../specs/P3b-list-engine.md) 2.1、R1–R9。

## 处理结果（M6/P3b）：M6-03

P3b 加两张表 `cycle_user_properties`（`00046`）、`module_user_properties`（`00047`），归 `planning`，照本条的协议覆盖各删除入口与清理：

- **删迭代、删模块**：在删除事务里、同一时刻、同一写者软删它未删除的偏好行，先前已删的保留自己的删除（`TestDeleteCycleThroughTheApp`、`TestDeleteModuleThroughTheApp`，适配器 `TestDeleteCyclePreferenceRows`、`TestDeleteModuleWithItsRows`）；改偏好以项目行 `FOR SHARE` 与删除串行，两种次序的交错见 `TestPageSettingsAndTheirTargetsDeletionInterleave`。跳过这一步、改写已删行的变异（`t9-mut-cycle-skips-preferences`、`t9-mut-module-skips-preferences`、`t9-mut-cycle-rewrites-deleted`）让测试失败。
- **项目、工作区删除**：走 `planning` 已有的 `planning.delete_rows`（订阅 `project.deleted`/`workspace.deleted`，原事务投递，父范围立即隐藏，沿用请求的 At/By）：`DeletionRowSteps` 在 `module_links` 之后加两步，剩余探查加两个分支，按批有界、可续跑（`TestDeletionRowStepsParentsFirst`、`TestDeleteRowsProbesTheStepItStoppedIn`、`TestProjectDeletionRemainingFromItsStep`、`TestDeleteProjectPlanningRows`）；守卫 `TestDeletingAProjectLeavesNoUndeletedRowUnderIt`、`TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt` 以 SQL 种的偏好行（`seedCycles`、`seedModules`）核软删。漏掉 `module_user_properties` 一步的变异（`t8-mut-steps-no-module-preferences`）让它们失败。
- **e2e 夹具**：`furnishPlanningRows` 以管理员的令牌经接口 PATCH 两种偏好，`asynchronousDeletionTable` 加两张表，P4、W2、W3、I21 等删除故事随之要求两张表有行（收据 `t11-e2e-deletion`，25 条通过）；不写偏好的变异（`t11-mut-furnish-no-preferences`）让六个故事失败。
- **60 天清理**：`planning` 的清理步骤子先父，`module_user_properties → module_links → module_members → modules → cycle_user_properties → cycles`（`TestPurgePlanningTables`、`TestModulePurgeChildrenFirst`；把 `cycles` 放到 `cycle_user_properties` 之前的变异 `t8-mut-purge-cycles-first` 让它们失败）；运行角色可执行（`TestPurgeDeletedRunsWithRuntimeRole`，授权文件 `TestTheGrantsFileCoversEveryRelationAndFunction`，变异 `t8-mut-grants-missing`）；清理目录 `TestPurgeCatalogCoversTablesAndIncomingKeys` 认得两张表。

全图与有限批次清理的规模重跑在 M6 收尾，本交接保持 open。证据：提交 `e239a557471a048f56453f94676f2ecec22bf238`（表、删除任务、清理与守卫）、`8fd6f9972b887824d4524f2a7dbf63444e3ec48e`（删迭代/模块时软删）、`79a33c7c83c078ac4664eb0379c0decca4795226`（接口、e2e 夹具与交错）；[P3b spec](../specs/P3b-list-engine.md) 2.4、2.5、A6。

## 处理结果（M6/P4）：M6-04

统计的消费者要用的燃尽序列在 P4 定形：`Burndown {start_date, end_date, scope, points: [{date, remaining}]}`，点覆盖开始到结束（模块到目标日）的每一天，今天之后 `remaining` 为 `null`；一天的迭代只有 1 个点，草稿、缺日期或跨度超过 3660 天时整个燃尽为 `null`，所以 0、1、多个点与 `null` 都是合法的回答（[P4 spec](../specs/P4-progress-transfer.md) §2.1、R4，`planning/domain/burndown_test.go` 的 `TestProgressBurndown`、`TestProgressBurndownNeedsBothDays`、`TestProgressBurndownSpan` 覆盖这些形状）。`ProgressChart`/`AreaChart` 按它画、`null` 不画成 0、坐标有限，在 P5 的迭代页面与 P6 的模块页面启用前完成；本条保持 open。证据：提交 `735cac67f04519adee7d6a046051cc9dcefabe05`、`a92a292387db1105e97a7a52b8e4b9ac6d7d70bb`。

## 处理结果（M6/P5a）：M6-01、M6-04、M6-02

- **M6-01**：迭代自身的页面接上 nerve：列表（进行中卡片、即将开始、已结束）、预览、归档目录、详情页与进度侧栏都只调 `/api/v0`；真实迭代覆盖详情、统计、归档目录、空态、拒绝与重试（G1、G2、G3、G5–G8 页面故事），归档目录取数失败可重试、不再永远加载（`t9-mut-archive-forever-loader`）。M4 冒烟故事 S2 中迭代的两条移出（`b36fe3b1ce130ecba8d97d46a804b4ded9bb2b91`），原覆盖由 G3（不存在的迭代只有 `/api/v0` 的读、404、没有旧地址）与 G8（空归档目录是空态）接上。模块的页面在 P6。
- **M6-04**：迭代表单在三种引擎（`e2e/playwright.m6-pages.config.ts`）核对 Tab、日期的选择与确认、Escape、焦点返回、被拒保留输入（G1、G3）；输入法：组字中确认组字的 Enter 不提交、组字之后的 Enter 照常提交（G1 的输入法故事；chromium 经 DevTools 协议真组字，三种引擎都派发 M4 的 `composingKey` 两种次序，firefox、webkit 无法驱动真实组字，提交 `9831341cd2de3952f78db0778724840dde207a6c` 的裁定）。`cycles/form.tsx` 迁到自然次序、`PROJECT_CYCLE` 删除，迭代归档按钮的正 `tabIndex` 删除；侧栏的日期提交等 nerve 确认、被拒回到 nerve 的值（G3）；修改弹窗只在确认后关闭、Escape 经 `ModalCore` 的 `handleClose`；`BurndownChart` 覆盖 0、1、多个点，`null` 不画、坐标有限（单测与 G5 的一天的迭代）。`getTabIndex` 的最后读者是模块（P6）与 M7。模块的各项在 P6。
- **M6-02**：迭代的部分在 P5b（迭代页的工作项列表）。
- 证据：提交 `523c3d92e5d2d7fa3c57855e099dfbd87234ef1e`、`5cdd61c28bba772e2b69332c1b96fb26cb900e0a`、`a415c86eb74295cd132d449c0ad49b512f93e91f`、`b6a0b7dfe45aeb2fce97abda9b3e6ed14f1e4a10`、`1523da8682d55866e8c08f6da5eac0db34d784ba`、`5524a44dd6576da362a2588ec4079d307d565747`。本交接保持 open。

## 处理结果（M6/P5b）：M6-01、M6-02（迭代），及 P5b 修好的一个 M4 缺陷

- **M6-01**（迭代页的列表部分）：P5a 的迭代详情页正文只有转移横幅与进度侧栏；P5b 在横幅之下接上工作项列表（`CycleLayoutRoot`），整页仍只调 `/api/v0`。真实迭代覆盖四种布局、本页偏好的读写、已归档迭代的页面（照常列出、只读）、迭代关闭时的停用空态（不取本页偏好与列表），空态、本页偏好取数失败的说明与重试由组件测试覆盖（`roots/cycle-layout-root.test.tsx`、`roots/cycle-layout-access.test.tsx`）；进行中卡片的优先级页签取数失败时说明并可重试，不再一直占位。G15 的三个页面故事每种布局以 `expectListPageQuiet` 收尾（整页零非预期失败），G19 的失权故事放行旧回答后不报失败。模块页在 P6。
- **M6-02**（迭代的部分）：在原生列表 API 与 M4 的列表绑定（`IssueListBinding`，分组分页照旧）上恢复了迭代页的列表：
  - 列表 store `CycleIssues`（`core/store/issue/cycle/issue.store.ts`，继承新抽出的 `PagePreferencesIssues`，再继承 `BaseIssuesStore`）：查询是本页偏好得出的项目列表查询加 `cycle_id: [id]`，scope 仍是 `project`（[P5b spec](../specs/P5b-cycle-list.md) R1），每个迭代一个绑定，挂载键带迭代 id；
  - 筛选 store `CycleIssuesFilter`（按迭代 id 存取、不读 `router`，与项目页共用按页的基类 `PagePreferencesFilter`，R3）；
  - 布局根 `CycleLayoutRoot` 与列表、看板、日历、表格的四个 `cycle-root.tsx`；`EIssuesStoreType.CYCLE` 在 `useIssues`、`useIssuesActions`（迭代页的动作在 `core/hooks/use-cycle-issue-actions.ts`）、`ISSUE_STORE_TO_FILTERS_MAP`、`TCreateModalStoreTypes`、拖拽、空态与路由映射中的实际分支；迭代的空态（`empty-states/cycle.tsx`，只有“新建工作项”与“清除筛选”）；
  - 列表的成员判断认得 `cycle_id` 与 `module_id`，`IssueRootStore.fields()` 认得 `cycle_id` 与 `module_ids`，创建与更新的请求体放行 `cycle_id`（R2）；
  - 活动迭代与进度侧栏对列表的读取：进行中卡片的优先级页签（另一个 store 实例 `CyclePriorityIssues`，显式的完整查询，不改迭代页列表的当前绑定，R7），卡片与侧栏的行点击设迭代页的筛选（R8）；迭代转移弹窗已在 P5a 重建；
  - 没有恢复 Plane 的旧写链与已删的算法：本页新建是一个带 `cycle_id` 的请求（不是建立后再 `POST cycle-issues`，R4）；移出迭代经 `PATCH /issues/{id}` 的 `cycle_id: null`（R6）；写之后不在网页上推算分布，迭代页列表的写经 nerve 确认后重读本迭代的 `/progress` 一次（R10），`distribution-update` 没有恢复。
  - **在 P7 的几项**：分组头“添加已有”、日历的 `addIssuesToView`、`existing-issues-list-modal.tsx` 的迭代调用方本段没有做。本条“对应列表第一次开放前完成”字面上包含它们；迭代页的列表在 P5b 先开放，这几项依 [M6 设计](../M6-design.md) §12 的 P7 一行与 §15 第 12 条（“添加已有”逐个 `PATCH`）随 P7 的迭代下拉框与列表词表一起完成（P5b spec R11），本段的空态与分组头没有“添加已有”。
  - **“何时验收”逐项**（真实引擎；页面故事 `e2e/stories/planning/g15-cycle-list-page.spec.ts` 三个、`g19-cycle-list-page.spec.ts` 四个，chromium、三种引擎的页面配置、三种引擎 `--repeat-each=3` 都通过）：
    - Go 与前端同 fixture：服务端与网页比较器共用 `issue-order.json` 见 P3b 的回执；迭代页的列表走同一个绑定，不另写一份排序。
    - 双方向分页、重复与空页：G15 故事二，迭代里 51 个工作项按附件数两个方向，接口经 `expectPages` 每项一次、最后一页没有游标，页面按状态分组（每组 50）滚到底续页，每项一次、之后没有“Load more”。
    - 跨分组写后计数：G15 故事二，按状态分组时把一行从 Todo 改到 Done，两组的数（50、1）与页头的数随 nerve 的回答变化。
    - OwnOnly 与管理员兼访客：G19 故事三，只看自己的工作区管理员兼项目访客（wanda）与纯访客（gus）各只看到自己建立的一项、页头数目 1；wanda 照项目页可新建、可移出（实际移出她的一项，nerve 接受），gus 两者都没有，行菜单只有新标签页打开与复制链接。
    - 失权迟到答：G19 故事二，列表的重读被扣住时管理员把调用者移出项目，页面说他没有权限；他再加入后页面列出 nerve 现在的工作项，之后放行的旧回答不回来、不报失败。
    - 归档范围：G15 故事二，在迭代页归档一行，行离开、分组与页头的数随 nerve 变化，接口的总数同为 50。
    - 换会话：G19 故事三（另一标签页登录别人，页面跟随新会话、旧会话的迟到答不进来）、G15 故事二（重新登录后附件数照 nerve 现在的给出）；换迭代（面包屑）、后退前进、换项目时旧迭代的列表与迟到答不进新页面：G19 故事一。
    - 不经 DTO 转换层：类型直接来自 `@nerve/api-client`。
  - e2e 变异（台账）：列表不带 `cycle_id`、已结束仍可写、新建不带 `cycle_id`、附件列恒为 0、迭代关闭时仍取偏好、弹窗放回行内、挂载键不带迭代 id、迟到的回答设为当前列表、失权的两道守卫一起去掉、访客可新建、换会话不建新的根、卡片的行合并而非替换、确认前就跳转、不打开预览、优先级页签不续页、侧栏的行替换而非切换，各让故事失败；单门的“访客可移出”“访客行可改”与“已归档可写”是等价变异（被另一道门挡住、或 nerve 只归档已结束的迭代），由嵌套变异与 T7、T8、T10 的单测覆盖。
- **M4 的缺陷（P5b 发现并修复）**：列表行的快捷操作（`WorkItemQuickActions`，项目页、迭代页、个人页的四种布局共用）把“修改”与“复制一份”的弹窗渲染在行内。打开弹窗本身经 `fetchIssue → loadIssueResources → fetchSubIssues → readChildren` 对该工作项做父项核验，M4 引擎按设计在核验期间把行移出页面（`getIssueById` 不给核验未决的工作项，`IssueLayoutHOC` 以骨架替换布局）并重读第一页，行与弹窗一起卸下，弹窗一打开就关上；项目页与迭代页在真浏览器都复现（T13 的探针，诊断变异证明触发者是子项读取的父项核验、卸载来自引擎的两道显示门）。M4 已知子项读取的父项核验会卸载行（[P12 spec](../../M4-issue-core/specs/P12-web-list-pages.md) D22、[re-17](../../../reviews/re-17-M4P12整体审核.md) R1、[re-29](../../../reviews/re-29-M4整体审核.md) R08），做法是把须在行卸下后存活的界面放进加载门之上的布局宿主（[re-19](../../../reviews/re-19-M4P13整体审核.md) R2 的 `IssueListCommandHost`），这两个弹窗漏掉了，M4 也没有从列表行菜单打开修改的故事。修复照 P13 R2：弹窗归 `IssueListCommandHost`（`useWorkItemDialogRequest`，请求时定下弹窗的数据，换路由或调用者在该项目的权限身份变了即结束），行只提交目标；核验语义与引擎的显示门不变（主控裁定不改 D22、R08）。项目页、个人页随之修好。证据：提交 `a4735ee7ca206aca7f34382b5d68436b1c86712c`（单测 `roots/row-dialog-lifetime.test.tsx` RED 7 例；弹窗放回行内、宿主恒以 PROJECT 打开、宿主不开弹窗、弹窗不看权限身份、修改也以复制打开，5 个能编译的变异各让测试失败）；G15 故事三在迭代页与项目页核对行在核验中卸下又回来时弹窗仍在、标题不变（e2e 变异“弹窗放回行内”让它失败）。
- 证据：提交 `a91035eaf940fb50e0a3c2bc588c371c80adf788`（成员判断与写入的请求体）、`b205756f2789f57126fa5d1dcc7ade87141c89f2`（偏好的筛选 store 抽出基类）、`781ffefc719c93cb0f30324de88408d1dfc39823`（迭代页偏好的筛选 store）、`5f530b8b11419816136bfe2da745fdc023cc1458`（列表引擎的 CYCLE）、`a46cf5e9cf9694192579670bb81fbd8f253f113c`（按 store 类型读本页的筛选）、`fd26d88e9076fc62a2850ce82acc846349ec6209`（整页只读）、`77d52b4bad905ca9b3de2077d83e88a524376a3a`（列表、空态与只读）、`641f507bcc5c2e39559ee260e31f72f6a08b26fc`（页头与本页新建）、`0dcdfe5f827f7e28a92164c1c370784bf60f1d39`（快捷操作与写后重读进度）、`7051ee4d5f11076de8535ffa180c5cac2f73bc2f`（优先级页签）、`99bf286bfa0ba0025700d9b1b27b14b5413e7cc8`（行点击）、`a4735ee7ca206aca7f34382b5d68436b1c86712c`（M4 缺陷的修复）、`f4a3700732eeda348ebf160b6aa8062fb168b1f1`（G15 页面故事）、`b90dda8156b17fd911d4380f61407456dc23e477`（G19 页面故事）；[P5b spec](../specs/P5b-cycle-list.md) §2、R1–R11。本交接保持 open。

## 处理结果（M6/P6a）：M6-01、M6-04（模块），M6-02 的模块部分在 P6b

- **M6-01**（模块自身的页面）：模块的列表（列表与卡片）、预览、归档目录、详情页的页头与侧栏都只调 `/api/v0`；真实模块覆盖详情、统计（计数、百分比、燃尽、分布）、归档目录、空态、拒绝与重试（G9、G10、G12、G13、G19 页面故事，三种引擎，每个故事以整页安静的断言收尾）。详情页把“模块不存在”（404 `planning.module_not_found`）与其他读失败分开，后者可重试（变异 `t8-mut-not-found-any-error`）；列表页、归档目录与预览取数失败时说明并可重试，不再一直加载（`t10-mut-failure-loader`、`t11-mut-archive-forever-loader`）。M4 冒烟故事 S2 中模块的两条移出（`c09d4f8d9d2544724887ad1ad34a5c6cf0b44ffc` 移出 `module`，`e2d967a18fc3ba0f02210e51f9e407d4f1893c5f` 移出 `archived modules`），原覆盖由 G13 接上：删除了的模块的地址显示“模块不存在”，请求只有 `/api/v0` 的读（模块、进度、链接各 404），没有旧地址；没有已归档的模块时归档目录立即显示空态，不是加载占位。模块页的工作项列表在 P6b。
- **M6-04**（模块）：
  - 模块表单在三种引擎（`e2e/playwright.m6-pages.config.ts`）核对原生 Tab 的次序、弹窗里选日期、Escape、焦点回到打开它的按钮、名字的字段错误与焦点、被拒（409 重名）保留输入并说明（G9 故事一）；修改弹窗只发改了的字段、只在 nerve 确认后关闭（G9 故事二）。输入法在整审 re-69 之后的修复轮 F2 核对（G9 的输入法页面故事 `e2e/stories/planning/g9-module-name-page.spec.ts`，同 P5a 的 G1）：组字中确认组字的 Enter 不提交、弹窗留着、名字是组好的“模块”，组字之后的 Enter 经浏览器的隐式提交发出 `{name, status}`；chromium 经 DevTools 协议真组字，三种引擎都派发 M4 的 `composingKey` 两种次序，firefox、webkit 无法驱动真实组字。名字的 Enter 不看组字即提交、只看 `isComposing` 两个产品变异（`f2-mut-enter-ignores-composition`、`f2-mut-enter-reads-iscomposing-only`）各让它失败。
  - `modules/form.tsx` 迁到自然次序，`PROJECT_MODULE` 删除；模块归档按钮的正 `tabIndex` 删除。`getTabIndex` 余下的读者是 M7 的 `views/form.tsx` 与收集箱新建弹窗的四个文件，M7 是最后的消费者。
  - 本条点名的模块观察：表单的日期在弹窗里选、随表单提交；侧栏、列表行与卡片的日期只在两天都选了或清除时发一次，等 nerve 确认，被拒回到 nerve 的值（G9 故事二；组件测试与变异 `t8-mut-one-day-sends`、`t8-mut-refused-days-kept`、`t10-mut-refused-days-kept`）；进度的页签与面板出自同一张表、次序一致（共用的 `ProgressTabs`，变异 `t5-mut-panels-plane-order`；G12 核对三个页签各自的面板）。
  - 图表：模块侧栏改用 P5a 的 `BurndownChart`（0、1、多个点，`null` 不画，坐标有限）；G12 核对燃尽从开始日画到今天、之后为空，没有目标日的模块没有燃尽、分布照常；`ProgressChart` 与 `AreaChart` 的 `comparisonLine` 删除，单点的除零随之不存在。
- **M6-02**（模块的部分）在 P6b（[P6a spec](../specs/P6a-web-modules.md) §1、R15、§5）：`EIssuesStoreType.MODULE`、模块页偏好的筛选 store（P5b 的按页基类）、`module_ids` 整集合的写入（本页新建、“移出模块”）、四种布局与空态、附件列、侧栏的行切换本页筛选；“何时验收”的各项（同 fixture、双向分页、跨分组写后计数、OwnOnly 与工作区管理员兼访客、失权迟到答、归档范围、换会话）由 P6b 的 G15、G19 页面故事核对；P6b 在加入之前先按职责拆开工作项根 store（R16，M6-M8-08）。分组头“添加已有”、日历 `addIssuesToView`、`existing-issues-list-modal.tsx` 的模块调用方在 P7。
- 证据：提交 `3f46edea4510ebdee8789114a01d28809f1ead8f`（共用的进度页签、`comparisonLine` 删除）、`cd366811af816d1ed317aa7491dfa46cdfdc6f54`（表单与弹窗）、`e9fc041b4a1cae06173fa29ec9b03f9ad0c109ca`（归档与删除）、`c09d4f8d9d2544724887ad1ad34a5c6cf0b44ffc`（详情页与侧栏）、`82dac1efd9dac3fe7b7d99bdcac18410cd12bb0a`（列表页与预览）、`e2d967a18fc3ba0f02210e51f9e407d4f1893c5f`（归档目录）、`b1d2578e2aab5a9769ea7f71cbd7dc6f6b9e232d`（G9、G10、G13 页面故事）、`b8ed29df59bcb59fc30adfbafb8c79748af992b9`（G12、G19 页面故事）、`dbbefbd60765a849a388e89e48b7f92b22fb8316`（整审 re-69 之后的修复轮 F2：模块名字的输入法）。本交接保持 open。

## 处理结果（M6/P6b）：M6-01、M6-02（模块）

- **M6-01**（模块页的列表部分）：P6a 的模块详情页正文只有侧栏；P6b 在侧栏左边接上工作项列表（两域共用的布局根 `PlanningPageLayoutRoot`，域为模块），整页仍只调 `/api/v0`。真实模块覆盖四种布局、本页偏好的读写、已归档模块的页面（照常列出、只读）、模块关闭时的停用空态（不取本页偏好与列表）（G15 故事一）；空态、本页偏好第一次读失败的说明与重试、读到过之后的重读失败（正文沿用已读的偏好、提示并可重试）由组件测试覆盖（`roots/module-layout-root.test.tsx`、`roots/module-layout-access.test.tsx`、`roots/preferences-reread.test.tsx`）。G15 的三个模块页故事每种布局以 `expectListPageQuiet` 收尾（整页零非预期失败），G19 的模块页故事同样，失权故事放行旧回答后不报失败。
- **M6-02**（模块的部分）：在原生列表 API 与 M4 的列表绑定（`IssueListBinding`，分组分页照旧）上恢复了模块页的列表。加入之前先按职责把工作项根 store 拆出写入单元（M6-M8-08，[P6b spec](../specs/P6b-module-list.md) R1），并把 P5b 迭代页列表中只差领域的件改为两域共用（R2）：
  - 列表 store `ModuleIssues`（`core/store/issue/module/issue.store.ts`，继承两域共用的 `PlanningPageIssues`，再继承 `PagePreferencesIssues` 与 `BaseIssuesStore`）：查询是本页偏好得出的项目列表查询加 `module_id: [id]`，scope 仍是 `project`，每个模块一个绑定，挂载键带模块 id；
  - 筛选 store：`PagePreferencesFilter` 以 `ModulePreferencesService` 与 MODULE 直接构造（按模块 id、不读 `router`、随代退役；迭代页的 `CycleIssuesFilter` 只差这两个参数，一并删除）；
  - 布局根与四种布局、空态、行的快捷操作与迭代页共用（`PlanningPageLayoutRoot`、`PlanningPage{List,KanBan,Calendar,Spreadsheet}Layout`、`PlanningPageEmptyState`、`PlanningPageIssueQuickActions`）；`EIssuesStoreType.MODULE` 在 `useIssues`、`useIssuesActions`（两域共用的 `usePlanningPageIssueActions`）、`ISSUE_STORE_TO_FILTERS_MAP`（`module_issues`）、`TCreateModalStoreTypes`、拖拽、四个 base root、空态与路由映射中的实际分支；
  - 成员判断与成员的快照认得 `module_id`、`module_ids` 已在 P5b；写入单元的创建与更新放行 `module_ids`；
  - 进度侧栏对列表的读取：模块页列表的写经 nerve 确认后重读本模块的 `/progress`（R13），状态组、负责人、标签行切换本页筛选（R14）；
  - 没有恢复 Plane 的旧写链与已删的算法：本页新建是一个带 `module_ids`（给定的集合并上本模块）的请求，不是建好后再 `POST …/issues/{id}/modules/`（R6）；移出模块经 `PATCH /issues/{id}` 的 `module_ids`，请求体在写队列轮到时由所持的集合去掉本模块算出、保留已归档与别的模块，不是按对 `DELETE`（R5）；修改与复制不覆写模块集合（R7）；`distribution-update` 没有恢复。
  - **在 P7 的几项**：分组头“添加已有”、日历的 `addIssuesToView`、`existing-issues-list-modal.tsx` 的模块调用方本段没有做。本条“对应列表第一次开放前完成”字面上包含它们；模块页的列表在 P6b 先开放，这几项依 [M6 设计](../M6-design.md) §12 的 P7 一行与 §15 第 12 条（“添加已有”逐个 `PATCH`）随 P7 的模块下拉框与列表词表一起完成（[P6b spec](../specs/P6b-module-list.md) R10），本段的空态与分组头没有“添加已有”。
  - **“何时验收”逐项**（真实引擎；页面故事 `e2e/stories/planning/g15-module-list-page.spec.ts` 三个、`g19-module-list-page.spec.ts` 四个，chromium、三种引擎的页面配置、三种引擎 `--repeat-each=3` 都通过）：
    - Go 与前端同 fixture：服务端与网页比较器共用 `issue-order.json` 见 P3b 的回执；模块页的列表走项目页同一个绑定，不另写一份排序。
    - 双方向分页、重复与空页：G15 故事二，模块里 51 个工作项按附件数两个方向，接口经 `expectPages`（带 `module_id`）每项一次、最后一页没有游标，页面按状态分组（每组 50）两个方向滚到底续页，每项一次、之后没有“Load more”，续页的查询带 `module_id` 与组的游标。
    - 跨分组写后计数：G15 故事二，按状态分组时把一行从 Todo 改到 Done，两组的数（50、1）与页头的数（51）随 nerve 的回答变化。
    - OwnOnly 与管理员兼访客：G19 故事三，只看自己的工作区管理员兼项目访客（wanda）与纯访客（gus）各只看到自己建立的一项、页头数目 1，与 nerve 给他们的进度总数、列表总数及数据库对他们所建的计数一致；wanda 照项目页可新建、可移出她的一项（数据库里只有它在本模块的归属被移除），gus 两者都没有，行菜单只有新标签页打开与复制链接。
    - 失权迟到答：G19 故事二，列表的重读被扣住时管理员把调用者移出项目（同时把一项移出本模块、在本模块建一项），页面说他没有权限；他再加入后页面列出 nerve 现在的工作项，之后放行的旧回答不回来、不报失败。挡住旧回答的有两道守卫：被取代的部分读（`core/lib/reconciled.ts:93`）与权限水位（`IssueRootStore.isReadCurrent`）。两道一起去掉的嵌套变异让故事失败；单去部分读的检查等价存活（旧读在失权之前发出，水位先拒绝它）；单去权限水位本身就让故事失败，迭代页的 G19 故事二同样失败（`t14-e2e-mut-late-guard-inner-alone-cycle`），因为水位还管列表回答里写进工作项 store 的每一项（`core/store/issue/issue.store.ts:40`），部分读的检查只管绑定。P5b 的 T14 台账写“两道守卫单去任一道都存活”，只对部分读的检查成立，在此更正（P5b 的记录不改）。
    - 归档范围：G15 故事二，在模块页归档一行，行与 Done 组离开、页头的数变为 50，接口的总数同为 50。
    - 换会话：G19 故事三（另一标签页登录别人，页面跟随新会话、旧会话的迟到答不进来）、G15 故事二（另一标签页重新登录后附件数照 nerve 现在的给出）；换模块（面包屑）、后退前进、换到别的项目的模块页时旧模块的列表与迟到答不进新页面：G19 故事一。
    - 不经 DTO 转换层：类型直接来自 `@nerve/api-client`。
  - e2e 变异（台账）：列表不带 `module_id`、看板当作列表、已归档仍可写、新建不带本模块、移出时只保留页面读到的未归档模块、去掉已归档的（在数据库的归属处失败）、修改带 `module_ids`、模块关闭时仍取偏好、附件列恒为 0、弹窗随行离开、挂载键不带模块 id、页头控件不等偏好、迟到的回答设为当前列表、“新建工作项”不可 Tab 到达、失权后仍当成员、失权的两道守卫一起去掉、访客可新建、换会话不建新的根、访客的两道移出门一起去掉、行切换用迭代的实例、预览里行可点、布局根卸下后筛选实例留着，各让故事失败；单门的“访客可移出”“访客行可改”是等价变异（被另一道门挡住），由嵌套变异与 T10 的单测覆盖；“移出时去掉 store 里已归档的模块”（T5 单测的写法）在页面上等价（模块页只读到项目未归档的模块），由 T5 的单测覆盖。
- 证据：提交 `c331aa9d5c044ec0c878a82d364bdde9c2111281`、`691302633b51d4b35e645efe46b0ec862824a905`（拆出写入单元）、`85d3a0cc435c7e248cce06b0b63ed35f403f2fb0`、`6a174b7c7bd32bb48d3a141bf28ff2f7c7337caf`（迭代页列表的件改为两域共用）、`c09f81134fc29b9454b255fba48f681bfd42ab05`（模块页的偏好与列表 store，`module_ids` 的新建与移出）、`64ac0312411b1c4b2480acdac23e660e9d5d07ff`（列表引擎的 MODULE）、`d4eb1b082db200be6a2419160663fed00e8fccd5`（正文、空态、只读与停用）、`18a5ff8f8de7ed983072aea6ade690764c051172`（页头与本页新建）、`3a5c3cf6c3c6651052395d830ef22d615131eeb0`（移出模块、修改与复制、写后重读进度）、`c3a300cf7180939e78b9496a0b27751828e187e5`（侧栏的行切换）、`b2ffe170a053aa8a67adc41a151319d449959ac7`（重读失败时沿用已读的偏好）、`8200f3473e1633b12bffde5ae2b3108d99e4fcb7`（列表页的夹具改为按域）、`c65f7d1bca6fdff29a44a397c089fc4e67a4623e`（G15 页面故事）、`61e6075e5afd7872317020914a22a1b4e2d2fd94`（G19 页面故事）；[P6b spec](../specs/P6b-module-list.md) §2、R1–R16。本交接保持 open。
