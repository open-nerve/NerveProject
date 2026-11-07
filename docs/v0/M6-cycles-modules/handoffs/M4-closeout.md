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
