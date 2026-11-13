---
status: open
from: M2/closeout
to: M6
created: 2026-09-28
---

# M2 交给 M6：迭代的跨模块写入、删除关系图、stores 按会话分代、周期和模块的下拉框

下面这些来自 [M2 设计](../../M2-auth/M2-design.md) 13.2 中接收者含 M6 的各行（多个 M 的行只取 M6 的部分）。第 4 节在 13.2 里写给 M4，收尾改给 M6：它取的是周期，周期的接口在 M6（[M2 收尾 spec](../../M2-auth/specs/closeout.md) 第 4 节 D5）。

## 1. 迭代跨模块的写入和查询

- **做什么**：迭代（`cycles`、`cycle_issues`）跨 `planning` 与工作项模块的写入，用端口和共享事务；必须联表的查询，事先列为 `TestSQLCSchemaScope` 的例外并写明理由，或者用端口拆开（M2 设计 3.14）。
- **为什么**：sqlc 按模块限定 `schema`，一个模块的查询只能碰本模块的表（总体设计 6.3）；`TestSQLCSchemaScope`（`server/internal/archtest/sqlc_test.go`）核对这份配置。
- **关闭条件**：M6 设计列出每个跨模块的写入和查询，以及它走端口还是例外；例外在 `TestSQLCSchemaScope` 中写明理由。

## 2. 物理删除与跨模块外键的关系图（M6 的部分）

- **做什么**：M6 的每张新表照 M2 设计 3.13 搬 Django 模型的 `on_delete`，在 M2 设计 4.7 的图上延伸，写明物理删除时每条外键的去向（M3、M4、M7 各自延伸自己的表）。
- **为什么**：Plane `cycle.py:65-68` 的迭代负责人是 `CASCADE`：物理删除一个账户会连带删除迭代。M2 只停用、不删除账户。
- **关闭条件**：M6 设计有延伸后的图；迭代负责人的 `CASCADE` 有结论（照搬，或改为 `SET NULL`、`RESTRICT` 并登记差异）。

## 3. stores 按会话分代（规则在总体设计 7.7）

- **做什么**：M6 接上新接口的 stores、services 和页面照 [总体设计 7.7](../../v0-design.md) 写。M6 的范围（`d97c513` 上数出）：
  - **注册在沿用的 `router` 上的反应**：`web/apps/web/core/store/cycle_filter.store.ts:67`、`web/apps/web/core/store/module_filter.store.ts:74` 的 `reaction`，随退役的一代释放。
  - **模块级的 service 实例**：`git grep -n -E '^(export )?const [A-Za-z]+ = new [A-Za-z]+Service\(' -- web/apps/web` 中的 `CycleService` 2 处。
- **关闭条件**：两个反应随退役的一代释放，有单元测试；这 2 处都已消失；M6 新加的 SWR 键带 `loginId`。

## 4. 周期下拉框的取数

- **做什么**：`web/apps/web/core/components/dropdowns/cycle/cycle-options.tsx` 原来从不取周期（`!cycleIds` 对一个总是数组的值），M2/P5 清 lint 时改为打开时这个项目的周期还没取过（`getProjectCycleIds` 为 `null`）就取。同一处改动还带来三点（M2/P5 Task 10），M6 接上周期时核对：
  - 列表开着时 `workspaceSlug` 或 `projectId` 变了，打开时的 effect 再运行一次（重新聚焦，新项目没取过就取）；
  - 第一次取数还在途中时关上再打开，会再发一次 `GET`，两个回答都写 `cycleMap`、`fetchedMap`，没有先后的保护，旧的回答可能最后到；
  - `status` 是非字符串的假值（`false`、`0`、`NaN`）时，`status?.toLowerCase()` 现在抛出（原来返回 `true`）；这样的值不在它的类型 `TCycleGroups` 里。
- **关闭条件**：M6 的 review 写明三点各自的结论（保留、修正并有测试）；并发的两次取数不会让旧的回答覆盖新的。

## 5. 下拉框（M6 的部分）

路径在 `web/apps/web/core/components/` 下，另写明的除外。根因和修法见 M2/P5 spec 2.3。

- **12 个同样写法的下拉框中 M6 的 2 个**：`dropdowns/cycle/cycle-options.tsx`、`dropdowns/module/module-options.tsx`。react-popper 的 ref 放在 `Combobox.Options` 唯一的子元素上，列表会停在页面左上角；popper 的 ref 改放在列表元素本身。打开状态只要 Headless UI 的一份：没有搜索框的选择用 `Listbox`，有搜索框的用 `Combobox`，列表打开时让输入框取得焦点。
- **P5 重建的 `CustomSelect`**（`Listbox`：按钮上的 Enter 提交所在的表单，输入字母跳到对应的选项，列表开着时页面其余部分 inert）在 M6 的 3 个调用方：`modules/{module-status-dropdown,progress-sidebar/root,select/status}.tsx`。
- **`BreadcrumbNavigationSearchDropdown`**（`web/packages/ui/src/breadcrumbs/navigation-search-dropdown.tsx`，P5 重建的 `CustomSearchSelect`：`onOpen` 每次打开调一次）在周期和模块的页头上。
- M4 改好的 `dropdowns/date-range.tsx`、`dropdowns/project/base.tsx` 也在周期、模块的表单里（`cycles/form.tsx`、`modules/form.tsx`），接上时核对。
- **关闭条件**：M6 的 review 逐项写明：两个下拉框的列表在按钮旁展开；列出的调用方在浏览器中核对过。

## 6. 复制到剪贴板（M6 的 3 处）

- P5 起 `copyTextToClipboard`（`web/packages/utils/src/string.ts`）复制失败时拒绝，纯 http 下也是。M6 的 3 处没有处理拒绝，失败时没有提示，只有一个未处理的 Promise 拒绝：`web/apps/web/core/components/cycles/quick-actions.tsx`、`web/apps/web/core/components/modules/quick-actions.tsx`、`web/apps/web/core/components/modules/links/list-item.tsx`。
- **关闭条件**：3 处都改经 `useCopyLink`（`web/apps/web/core/hooks/use-copy-link.ts`，M3/P10；复制文字的是同一文件的 `useCopyText`，M3 收尾），失败时经 `t()` 提示；关键词规则 `copy-hook` 中它们的例外随之删除（M3 收尾，[M3 的交接](M3-closeout.md)）。

来源：[M2 设计](../../M2-auth/M2-design.md) 13.2 中接收者含 M6 的各行和"周期下拉框的取数"一行；[M2 收尾 spec](../../M2-auth/specs/closeout.md) 第 3 节。

## 处理结果（M6/P1）：第 2 节

M6 设计 §4.3 已在 M2 §4.7 的图上延伸到迭代、模块及其关联表。迭代负责人的结论是改为可空、`ON DELETE SET NULL`（Plane `CASCADE`，同 M3 对项目负责人的改法），差异登记在 [plane-diff](../../plane-diff.md) 按表的 `cycles` 行。P1 的 `cycles` 表（`00039_planning_cycles.sql`）照此落地，`server/migrations/planning_schema_test.go` 的 `TestCyclesSchema` 核外键去向；把负责人改回 `CASCADE` 的变异让迭代随账户消失而失败。模块的负责人与成员在 P2、关联表在 P3 落地；第 1、3–6 节也未到，本交接保持 open。证据：[P1 spec](../specs/P1-cycles-backend.md) 2.2。

## 处理结果（M6/P2）：第 2 节

P2 落地 §4.3 图中模块的三张表（`00040`–`00042`）：`modules.lead_id` 照 Plane 为 `ON DELETE SET NULL`，`module_members.member_id` 照 Plane 为 `ON DELETE CASCADE`（v0 只停用、不物理删除账户，这条不触发），两张子表的 `module_id` 与三张表的 `workspace_id`、`project_id` 都是 `CASCADE`，审计列 `SET NULL`；差异登记在 [plane-diff](../../plane-diff.md) 按表的 `modules`、`module_members`、`module_links` 行。`server/migrations/planning_modules_schema_test.go` 的 `TestModulesSchema`、`TestModuleMembersSchema`、`TestModuleLinksSchema` 核外键去向：物理删除负责人的账户，模块留下、负责人为空；删除成员的账户，带走他的成员行；删除模块，带走它的成员与链接。把 `member_id` 改为 `SET NULL` 的变异让 `TestModuleMembersSchema` 失败。关联表在 P3；第 1、3–6 节也未到，本交接保持 open。证据：[P2 spec](../specs/P2-modules-backend.md) 2.2。

## 处理结果（M6/P3a）：第 1 节

M6 设计 §6 列出每个跨模块的写入与查询，全部走端口与事件，`TestSQLCSchemaScope` 不加例外（§4.1）。P3a 落地其中与工作项归属有关的三项：（1）`issue` → `planning` 的 `PlanningTargets`（由使用方在 `issue/app/planning_ports.go` 声明）：`Targets` 在工作项写已取得的项目行锁下回答项目的两个开关，以及所问的迭代、模块中本项目未删除的（含已归档）的名字与是否归档；`CycleCompleted` 按项目时区、用这次写在最后一把锁之后读的时钟回答迭代是否已结束。（2）自动归档用的 `OpenPlanning.OpenTargets`（设计 §6 写作 `OpenPlanningIDs`，[P3a spec](../specs/P3a-issue-membership.md) R9 定名）：在自动归档的项目锁与任务时钟下回答未结束的迭代与目标日未过的模块的 id，数组传入 `issue` 自己的静态语句。两个端口由 `planning/app/issue_targets.go` 实现（“已结束”“未结束”的口径只在 `planning`），`planning.Provide(pool, projects)` 交出，`bootstrap/ports.go` 的 `issuePlanning` 转换。（3）`planning` → `issue` 的事件 `planning.cycle_deleted`、`planning.module_deleted`：`planning` 在删除的事务里发布，`issue` 的订阅者 `NewMembershipRemovals`（`issue/app/membership_events.go`）在同一事务软删归属、写动态、记工作项的最后修改，`bootstrap/events.go` 的事件目录加入 `planning.EventTypes()`。两张归属表归 `issue`（负责人决定 A）：`server/sqlc.yaml` 的 issue 条目只加 `00043`、`00044`，`issue` 的语句不连 `cycles`、`modules`，名字与事实只经端口与事件；外键指向 `planning` 的表照 `issue_labels.label_id` 的先例在 `TestSQLCSchemaScope` 下允许，该测试与 archtest 都没有放宽（收据 `t2-red-sqlc-scope` 在 issue 条目未列两个迁移时失败，`t2-green-archtest` 通过）。测试：`planning/app` 的 `TestIssueTargetsAnswerTheTogglesAndTheTargets`、`TestCycleCompletedOnTheProjectsDay`、`TestOpenTargetsOnTheProjectsDay`，`planning` 适配器的 `TestIssueTargetsOfTheProject`、`TestOpenCycleIDs`、`TestOpenModuleIDs`，`bootstrap` 的 `TestIssuePlanningConvertsPlanningsAnswers`、`TestPlanningEventDirectory`、`TestDeletingACycleTakesItsWorkItemsOut`、`TestDeletingAModuleTakesItsWorkItemsOut` 与交错 `TestJoiningAndDeletingATargetInterleave`；以 UTC 的今天判定“已结束”、读到别的项目的迭代、删迭代不发布事件的变异（`t5-mut-completed-utc-today`、`t5-mut-targets-other-project-cycles`、`t9-mut-cycle-not-published`）都让测试失败。设计 §6 其余的 `PlanningDirectory`（P3b）、`PlanningIssueStats` 与 `TransferCycleIssues`（P4）未到；第 3–6 节也未到，本交接保持 open。证据：提交 `2ce5591cab00f32200502c05e54b7c1248d02d03`（表与 sqlc）、`52d9631d91a5f3f6879ce8ae8cc31e5a5d608bb3`（端口）、`a37bc6370ae651118bd9cb6204279679fdd5031c`（事件与订阅者）、`9e3220594912c9a9d419e8d13154b67473ae8a71`（自动归档）；[P3a spec](../specs/P3a-issue-membership.md) R1、R6、R9。

## 处理结果（M6/P3a）：第 2 节

P3a 落地 §4.3 图中的两张归属表 `cycle_issues`（`00043`）、`module_issues`（`00044`），归 `issue` 模块：`workspace_id`、`project_id`、`cycle_id`/`module_id`、`issue_id` 四条外键照 Plane 的模型都是 `ON DELETE CASCADE`，审计列 `SET NULL`，每条外键有不带条件的索引；差异登记在 [plane-diff](../../plane-diff.md) 按表的 `cycle_issues`、`module_issues` 行。`server/migrations/issue_membership_schema_test.go` 的 `TestCycleIssuesSchema`、`TestModuleIssuesSchema` 核外键去向与两个部分唯一：物理删除作者的账户，行留下、审计列为空；依次物理删除迭代或模块、工作项、项目、工作区，各自只带走自己的行。把两张表的 `issue_id` 改为 `SET NULL` 的变异（`t2-mut-cycle-issue-set-null`、`t2-mut-module-issue-set-null`）让它们失败。§4.3 图中只剩 `cycle_user_properties`、`module_user_properties`（P3b）未落表；第 3–6 节也未到，本交接保持 open。证据：提交 `2ce5591cab00f32200502c05e54b7c1248d02d03`；[P3a spec](../specs/P3a-issue-membership.md) 2.2。

## 处理结果（M6/P3b）：第 1 节

P3b 加三处跨模块的读与校验，都走由使用方声明的端口，`TestSQLCSchemaScope` 不加例外；M6 设计 §6 的表已列出（`PlanningDirectory` 一行按 [P3b spec](../specs/P3b-list-engine.md) R2 收窄）。（1）`issue` → `planning` 的 `PlanningDirectory`（`issue/app/planning_ports.go`）：按迭代名、模块名排序时读迭代/模块的 id 与名字（含已归档、不含已删除），由 `planning/app/issue_directory.go` 实现，经 `planning.Provide` 的 `IssueDirectory` 交出、`bootstrap/ports.go` 的 `issuePlanning` 转换；`planning` 的语句只读自己的表（`queries/issue_directory.sql`），以数组传入 `issue` 的静态语句（`TestIssueDirectoryOfTheProjects`、`TestIssuePlanningConvertsPlanningsNames`；把模块名读成迭代名的变异 `t4-mut-bridge-swap` 让后者失败）。（2）`planning` → `project` 的 `PlanningProjects.Share`：改迭代/模块页的偏好时把项目行加 `FOR SHARE`，由 `project` 的 `SharePlanning`（`project/adapter/postgres/queries/planning_ports.sql`）实现（`TestSharePlanning`、`TestAPagePreferencesChangeSharesItsProject`；去掉 `FOR SHARE` 的变异 `t10-mut-share-no-lock` 让它们失败）。（3）`planning` → `issue` 的 `IssueListSettings`（`planning/app/list_settings.go`）：两页偏好的词由 `issue.ListSettings` 校验，纯函数、不读库，与项目偏好同一个；`bootstrap` 直接交出，删去项目偏好原有的逐字段转换。设计 §6 其余的 `PlanningIssueStats`、`TransferCycleIssues`（P4）未到；第 3–6 节也未到，本交接保持 open。证据：提交 `0d7caf28af607d9819edf543c5e786c2c6ff62ba`（目录端口）、`acc1f8a9fd0ae43513d5b81262edd744015c7323`（删去项目偏好的转换）、`ba883d875345e586fcee6e2936416771c85084e5`（planning 的列表设置端口与共享锁之路）；[P3b spec](../specs/P3b-list-engine.md) R2、R5、R6。

## 处理结果（M6/P3b）：第 2 节

P3b 落地 §4.3 图中最后两张表 `cycle_user_properties`（`00046`）、`module_user_properties`（`00047`），归 `planning`：`workspace_id`、`project_id`、`cycle_id`/`module_id`、`user_id` 四条外键照 Plane 的模型都是 `ON DELETE CASCADE`（`user_id` 这条在 v0 不触发：只停用、不物理删除账户），审计列 `SET NULL`，四条外键各有不带条件的索引；差异登记在 [plane-diff](../../plane-diff.md) 按表的 `cycle_user_properties`、`module_user_properties` 行。`server/migrations/planning_preferences_schema_test.go` 的 `TestPlanningPreferencesSchema` 核外键去向：物理删除写者的账户，行留下、审计列为空；依次物理删除迭代或模块、账户、项目、工作区，各自只带走自己的行。把 `cycle_id` 改为 `SET NULL` 的变异（`t8-mut-cycle-set-null`）让它失败。§4.3 图中的表至此都已落地；第 3–6 节未到，本交接保持 open。证据：提交 `e239a557471a048f56453f94676f2ecec22bf238`；[P3b spec](../specs/P3b-list-engine.md) 2.4。

## 处理结果（M6/P4）：第 1 节

P4 落地设计 §6 余下的跨模块查询与写入，都走由使用方声明的端口，`TestSQLCSchemaScope` 与 archtest 不放宽。（1）`planning` → `issue` 的 `IssueStats`（`planning/app/issue_stats.go`，设计原写 `PlanningIssueStats`）：按状态 id 回答一个迭代/模块的工作项事实（状态、完成时刻、未删除负责人与标签的 id）、一页迭代/模块按状态的计数与迭代的负责人 id；`viewer` 为只看自己的访客时只算他建立的；由 `issue` 的 `queries/planning_stats.sql` 实现（不连 `cycles`、`modules`、`states`），经 `issue.Provide` 的 `PlanningStats` 交出、`bootstrap/planning_issues.go` 的 `planningIssues` 转换（`TestPlanningItems`、`TestPlanningStateCounts`、`TestPlanningCycleAssignees`、`TestPlanningIssuesConvertsIssuesStats`）。（2）`planning` → `project` 的 `StateGroups`：复用项目的 `IssueStates`，不加语句，`bootstrap/ports.go` 的 `planningProjects` 转换（`TestPlanningProjectsConvertsStateGroups`）；状态组到数字的计算在 `planning/domain/progress.go` 的纯函数。（3）`planning` → `issue` 的 `IssueTransfer.TransferCycleIssues`（`planning/app/issue_transfer.go`）：在 planning 的转移写事务里经 ctx 调用，锁源迭代的归属行（`FOR NO KEY UPDATE OF r`）、软删、插入目标的新行、每个工作项一条 `CycleMoved` 动态、`MarkChanged`，不发布事件；由 `issue/app/cycle_transfer.go` 实现，经 `issue.Provide` 的 `PlanningTransfer` 交出（`TestTransferringACyclesMemberships`、`TestPlanningTransferConvertsTheTransfer`，后者也核 ctx 的同一性）。锁序与交错见 [P4 spec](../specs/P4-progress-transfer.md) §2.6 与 `bootstrap/planning_transfer_interleaving_test.go`。设计 §6 的表至此都已落地；第 3–6 节（网页）未到，本交接保持 open。证据：提交 `a4b56a308dea04783100c41b98f87c284b0e89e1`、`52d3400e4cab4cc16960156990a74c02735f5eb8`（统计端口）、`c76eab7d4459a9028bac60eb8260e2faae640c4e`、`d08ee622e23903928577db6bd49862737052585c`（转移端口）、`4404ba2889c4693c180f2415c7dad17596f7730f`（转移的用例）。

## 处理结果（M6/P5a）：第 3–6 节（迭代）

- **第 3 节**：迭代的筛选 store 改为按项目 id 存取、不跟随 `router`，没有要随退役的一代释放的反应（[P5a spec](../specs/P5a-web-cycles.md) R4），`store-context.test.ts` 断言换代之后旧一代的筛选不随 router 变；两处模块级的 `CycleService`（新建弹窗、侧栏页头，都为 `date-check`）随 `date-check` 删除；新的取数都经 `useSessionSWR`，键带 `loginId`（`PROJECT_CYCLES`、`PROJECT_ARCHIVED_CYCLES`、`CYCLE`、`CYCLE_PROGRESS`）。模块的反应在 P6。
- **第 4 节**：`dropdowns/cycle/cycle-options.tsx` 已被 M4 删除，P7 恢复工作项的迭代下拉框。三点的结论：并发两次取数不让旧的回答覆盖新的，由迭代 store 的 `Reconciled` 保证（较旧的取数不写，T2 的测试与变异 `t2-mut-older-fetch-writes`）；`status` 是生成的 `CycleStatus`，网页不再对它 `toLowerCase()`；列表开着时换项目重新取数一点随 P7 的下拉框核对。
- **第 5 节**：(c) `BreadcrumbNavigationSearchDropdown` 在迭代页头切换迭代，G19 页面故事在三种引擎经它打开另一个迭代；(d) 迭代表单里的 `dropdowns/date-range.tsx` 在 G1、G3 页面故事中于三种引擎在弹窗里选日期、确认，浮层在弹窗之内可用；同一表单的 `dropdowns/project/base.tsx` 只给调用者能写迭代的项目（组件测试与变异 `t6-mut-project-any`），它的浏览器核对随 P8 的最终矩阵。(a) 在 P7，(b) 在 P6。
- **第 6 节**：`cycles/quick-actions.tsx` 与命令面板迭代页的复制改经 `useCopyLink`，失败经 `t()` 提示（G19 页面故事拒绝剪贴板时核对）；`copy-hook` 中这两条例外随之删除。模块的两处在 P6。
- 证据：提交 `b4911394dc2a92d4bd0224e572902e6d45275941`、`a21686c4a9827c1fb98d28eafa67c35a5b1150f7`、`5cdd61c28bba772e2b69332c1b96fb26cb900e0a`、`11ef5b52ab5b8497d887c0fe2c4d250fd0ce55b9`、`a415c86eb74295cd132d449c0ad49b512f93e91f`、`f497055055f00e0cdbfa74e08f5186d40a1297d3`、`5524a44dd6576da362a2588ec4079d307d565747`。本交接保持 open。

## 处理结果（M6/P6a）：第 3、5、6 节（模块）

- **第 3 节**：模块的筛选 store（`core/store/planning/module_filter.store.ts`）按项目 id 存取、不跟随 `router`，没有要随退役的一代释放的反应，也不再写 `localStorage`（[P6a spec](../specs/P6a-web-modules.md) R4）；`store-context.test.ts` 加一条，断言换代之后退役的一代的模块筛选不随 router 变（变异 `t4-mut-filters-follow-router`、`t4-mut-filters-persist` 各让测试失败），`RootStore.dispose()` 与 `store-context.tsx` 的说明改为迭代与模块的筛选都没有反应。模块的 service 由每一代以自己的客户端构造，没有模块级的实例；新加的 SWR 键 `PROJECT_MODULES`、`PROJECT_ARCHIVED_MODULES`、`MODULE`、`MODULE_PROGRESS`、`MODULE_LINKS` 都带 `loginId`。迭代的部分在 P5a，本节的关闭条件至此都已满足。
- **第 5 节**：
  - (b) P5 重建的 `CustomSelect` 在模块的调用方现在是 `modules/module-status-dropdown.tsx`（收模块 id、自己读 store，侧栏的页头、列表行与卡片共用）与 `modules/select/status.tsx`（表单）；`progress-sidebar/root.tsx` 不再直接用它。侧栏改状态在 G9 页面故事二中于三种引擎核对（等 nerve 确认、在途停用、被拒回到 nerve 的值）；表单的状态选择以真 `CustomSelect` 的组件测试核对（`select/status.test.tsx`：没过规则时有经 `t()` 的提示、`aria-invalid`、被拒时取得焦点）。
  - (c) 模块页头切换模块的 `BreadcrumbNavigationSearchDropdown`：G19 页面故事三在三种引擎经它从 Alpha 打开 Beta，它列出本项目未归档的模块、服务端的次序。
  - (d) 模块表单里的 `dropdowns/date-range.tsx`：G9 页面故事一在三种引擎于弹窗里选日期并提交，浮层在弹窗之内可用；`dropdowns/project/base.tsx` 只列调用者能写模块的项目（组件测试与变异 `t6-mut-project-picker-create-permissions`），它的浏览器核对随 P8 的最终矩阵。
  - (a) 的 `dropdowns/module/module-options.tsx` 在 P7。
- **第 6 节**：`modules/quick-actions.tsx` 的复制经 `useCopyLink`，`modules/links/list-item.tsx` 的复制经 `useCopyText`，命令面板模块页的复制也经 `useCopyLink`；失败经 `t()` 提示（G19 页面故事三拒绝剪贴板时核对快捷操作与链接一行两处）；`copy-hook` 中模块的三条例外随之删除，M6 的例外已全部删除（关键词的例外 10 → 7）。迭代的两处在 P5a，本节的关闭条件至此都已满足。
- 证据：提交 `d71ab782f11899b7ae45452ac8cf7e0d47a73c89`（service、store 与 SWR 键）、`6a088615629ba2c7d127e53cea25cc0376c1e2b5`（筛选 store）、`cd366811af816d1ed317aa7491dfa46cdfdc6f54`（表单、状态选择与项目下拉）、`e9fc041b4a1cae06173fa29ec9b03f9ad0c109ca`（快捷操作的复制）、`c09d4f8d9d2544724887ad1ad34a5c6cf0b44ffc`（侧栏的状态与页头）、`d8c87d1adfceb28b9200a446f7c5748551190e4a`（链接一行的复制）、`dc9172b0fecf86c716bf50394b03c2a4e4a6fd6d`（命令面板、`dispose()` 的说明）、`b1d2578e2aab5a9769ea7f71cbd7dc6f6b9e232d`、`b8ed29df59bcb59fc30adfbafb8c79748af992b9`（页面故事）。本交接保持 open。
