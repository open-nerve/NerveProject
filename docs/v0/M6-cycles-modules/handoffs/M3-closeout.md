---
status: open
from: M3/closeout
to: M6
created: 2026-10-13
---

# M3 交给 M6：项目包装层取迭代和模块、筛选的反应、迭代和模块页面上留下的前端

M3（工作区与项目）的项目包装层不再取迭代、模块，迭代和模块的页面在 M3 里走不到（[M3 设计](../../M3-workspace-project/M3-design.md) 3.1）。下面每一节来自 M3 设计 13.2 的 M6 一行，和 M3 各 Phase、M3 收尾交给 M6 的各条（[收尾 spec](../../M3-workspace-project/specs/closeout.md) 第 2 节的分诊表写明来处）。路径在 `web/apps/web/core/components/` 下的，只写这之后的一段。

## 1. 取数和 stores（总体设计 7.7）

- `ProjectAuthWrapper`（`web/apps/web/core/layouts/auth-layout/project-wrapper.tsx`）取迭代、模块，照 M3 设计 7.1 由权限启用（项目的有效成员才取）；`ModulesStore.fetchModulesSlim` 照新接口加回（M3/P8a spec 第 5 节）。
- `cycle_filter`、`module_filter` 跟随 `router` 的反应由 `RootStore.dispose()` 随退役的一代释放（M3 设计 7.1）；在那之前，退役的模块筛选的反应在每次换路由时仍写 localStorage（M3/P8a 评审第 6 节）。
- 迭代的 `user-properties` 不在关键词规则（旧 `ProjectService` 的 `/user-properties/`）的模式里，不命中它的样例（M3/P8b spec 第 5 节）。
- `core/modals/existing-issues-list-modal.tsx`（添加已有工作项）改用 M4 的工作项搜索接口（M3 设计 7.3）。
- 迭代、模块的弹窗读 `error?.error` 的三处（`cycles/transfer-issues-modal.tsx`、`modules/modal.tsx` 两处）改经文案表（`errorMessageKey`）；视图的修改之后的跟进经 `followInSession`（M3/P9 spec 第 5 节）。
- **界面状态在以 id 为 key 的布局之下**（M3 收尾，负责人 2026-10-13 的 B，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.7）：工作区的内容（`app/(all)/[workspaceSlug]/layout.tsx` 的 `WorkspaceContentWrapper`）以路由的 slug 为 key，项目的页面和项目的设置（两个 `[projectId]/layout.tsx`）以路由的项目为 key，换工作区、换项目（后退、前进也是）时其下重新挂载，组件状态里的弹窗和状态不带过去。store 持有的弹窗开关由渲染它的宿主在卸载时关上（`useLayoutEffect` 的清理，像 `WorkspaceLevelModals`）；宿主在 key 之上时还要在范围结束时关上。迭代、模块、视图的弹窗和选中放在这些布局之下，不要放到它们之上（路由的根、`WorkspaceAuthWrapper`、跨工作区的 store 或 `localStorage`）。
- **项目级弹窗的宿主**（M3 收尾；控制者的裁定：store 持有的弹窗开关属于渲染它的宿主，由宿主结束）：`modals/project-level.tsx` 的 `ProjectLevelModals`（创建迭代 `isCreateCycleModalOpen`、模块 `isCreateModuleModalOpen`、视图 `isCreateViewModalOpen`）。开关在命令面板的 store（`web/apps/web/core/store/command-palette.store.ts`）里，宿主由 `power-k/projects-app-provider.tsx` 渲染，在 `(projects)/layout.tsx` 和 `(settings)/layout.tsx` 的 `ProjectsAppPowerKProvider` 里，在项目的 key 之上：换项目（后退、前进也是）时开着的弹窗留着，作用于路由新指向的项目。M6 接上迭代、模块的页面时两件都做：把宿主移到项目的 key 之下，或在项目的范围结束时关上它的开关；并像 `modals/workspace-level.tsx` 的 `WorkspaceLevelModals` 那样在卸载时关上它渲染的每个开关（`useLayoutEffect` 的清理，在下一个页面绘制之前）。加一个像 P3、P8 那样的后退检查（[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.7）。视图的开关 `isCreateViewModalOpen` 只是随这个宿主一起，M6 改这个宿主时它的开关一并照做；本交接中写到视图的各条（第 1 节视图的跟进和弹窗、第 3 节、第 4 节视图的页面读地址）照 M3 各 Phase 的分派（P9、P10）交给 M6，保存的视图自己的页面是 M7 的（[M7 的交接](../../M7-collaboration/handoffs/M3-closeout.md)第 2 节）。
- **嵌套的 id 没有 key**（M3 收尾）：迭代、模块的页面 `cycles/(detail)/[cycleId]`、`modules/(detail)/[moduleId]`只在工作区、项目的 key 之下，换一个迭代或模块（后退、前进也是）时页面留着，开着的弹窗和组件的状态跟过去。M6 接上这些页面时以页面自己的 id 给它的子树 key，并加一个后退的检查。
- **关闭条件**：包装层的取数有按权限的 vitest；`dispose()` 释放两个反应，`store-context.test.ts` 核对换代之后它们不再运行。

## 2. 数据

- 迭代、模块的外键在 M3 设计 4.12 的删除关系图上延伸。
- `core/sidebar/progress-stats/` 的 8 行死成员（`label.tsx` 5 行、`state_group.tsx` 3 行，`domains.mjs` 按路径误归到 M3，M3/P11 spec 附录 A.9）。
- **关闭条件**：4.12 的图延伸到迭代、模块的表；这 8 行在 M6 的评审中有结论。

## 3. 键盘、弹窗和下拉框

- **弹窗**：M3 收尾把 `ModalCore` 改成 Headless UI v2 的 `Dialog`（[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.5）：打开时后面的页面 `inert`、`aria-hidden`，请求在路上时关不掉。迭代、模块、视图的弹窗在 M3 的故事里走不到：对接时逐个核对 Tab 在弹窗里转、Escape 和点外面、关上之后的焦点，以及弹窗里经 portal 渲染的日期选择、下拉框能用（收尾 spec 附录 A.3 只证明了 M3 页面上的几种）。`AlertModalCore` 的确认框照 A-M2：被拒绝时留着，只在成功的跟进中关上；按钮的文字经 `t()` 传入。
- **弹窗的名字**：Headless UI v2 的 `Dialog` 只以它的 `DialogTitle` 为名（`aria-labelledby`），普通的标题不给弹窗名字，读屏读出一个没有名字的对话框；M3 收尾的修复轮在 `@nerve/ui` 的 `ModalCore` 旁边导出 `ModalTitle`（它就是 `DialogTitle`，`h3`），`AlertModalCore` 的标题经它，M2、M3 的弹窗的标题都经它（[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.5）。M6 的确认框（`AlertModalCore`）因此都有名字；别的弹窗的标题改经 `ModalTitle`，没有看得见的标题的给一个 `sr-only` 的（不加第二个看得见的），标题的文字经 `t()`。以普通标题的：`cycles/form.tsx:100`（`cycles/modal.tsx`）、`modules/form.tsx:111`（`modules/modal.tsx`）、`cycles/archived-cycles/modal.tsx:67`、`modules/archived-modules/modal.tsx:67`、`cycles/transfer-issues-modal.tsx:92`（`h4`）、`modules/links/create-update-modal.tsx:90`；没有标题的：`core/modals/existing-issues-list-modal.tsx`（搜索的列表，`:222` 的 `h5` 是一组的说明）。M6 的故事打开它们时断言 `expect(modalDialog(page)).toHaveAccessibleName(…)`。共用的自定义日期的弹窗（`core/filters/date-filter-modal.tsx`）M3 收尾已在根上改，以调用方给的 `title` 为名；M6 的四个调用方（`cycles/dropdowns/filters/start-date.tsx`、`end-date.tsx`，`modules/dropdowns/filters/start-date.tsx`、`target-date.tsx`）传的是写死的英文（"Start date""Due date"），改经 `t()`。
- **正的 `tabIndex`**：迭代、模块、视图的表单（`PROJECT_CYCLE`、`PROJECT_MODULE`、`PROJECT_VIEW`）同 M4 的交接第 7 节：Tab 离开它们的弹窗，照样删去；迭代、模块的归档弹窗的归档按钮（`cycles/archived-cycles/modal.tsx`、`modules/archived-modules/modal.tsx`）是 `tabIndex={1}`（M3/P10 的裁定 F-18）。模块的状态 `ModuleStatusSelect`（`modules/select/status.tsx`）给 `CustomSelect` 的 `tabIndex` 落在它的按钮上（裁定 F-13），删去模块表单的正的 `tabIndex` 之后，它不再收 `tabIndex`，`CustomSelect` 不再读它（裁定 F-16）。M4、M6、M7 中最后删去正的 `tabIndex` 的那一个删除 `getTabIndex` 和两份 `tab-indices.ts`。
- **嵌套的按钮**：菜单自己的按钮里又放了 `IconButton` 的两处（M3 收尾找到）：`cycles/quick-actions.tsx`、`modules/quick-actions.tsx`；改法照 M3 收尾的项目一行（`CustomMenu` 的 `customButtonClassName`，内容是图标，不再嵌一个按钮）。
- **右键菜单**：迭代、模块、视图的列表的右键菜单关着时 `inert`（裁定 F-14），在 M6 的页面上核对右键打开它、点它的项。
- **下拉框**：C7 没有核对 M6 页面上的 `CustomSearchSelect` 调用方（成员下拉框的 2 个文件，迭代、模块、视图的页头），对接时用键盘核对。共用组件的焦点轮廓是否被 `overflow-hidden` 裁掉，在 M6 的页面上核对。
- **筛选的按钮**（M3 收尾 Task 12，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.9）：`FiltersDropdown` 默认的按钮现在就是弹层的 `Popover.Button`（以 `title` 为名、`aria-expanded`、Escape 之后焦点回到它，指针点开之后焦点留在它上面，同 M4 的交接第 7 节）；M6 的 8 个调用方（6 个用默认的按钮）在收尾 spec 附录 A.4 逐个核对过，没有故事走到，M6 的筛选的故事落地时断言按钮的名字、`aria-expanded` 和 Escape 之后焦点回到它。
- **每一行的控件以它那一行为名**（M3 收尾的规则，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.9：一行一个、每行都一样的控件以那一行的东西为名，同 F6 的项目一行、"{name} settings"、"Remove X"）：迭代一行的菜单按钮 `cycles/quick-actions.tsx:141` 和模块一行的 `modules/quick-actions.tsx:135`（都叫"Options"）以迭代、模块的名字为名：两处已有的 `cycleDetails?.name`、`moduleDetails?.name`；M3 收尾已加的带 `{name}` 的键 `common.item_options`（"{name} options"、"{name} 的选项"，项目成员页、标签页一行的菜单读它）。
- **字段的提示、弹窗里的列表、列表里的控件**（M3 收尾的浏览器核对之后的修复，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.17 第 2、3、5 条；写法同 M4 的交接第 7 节）。
  - 迭代、模块的表单的名字的提示（`cycles/form.tsx:134`、`modules/form.tsx:145`）照 M2、M3 的表单：提示是字段的 propel `FieldError`，控件以它为描述、`aria-invalid`，被拒绝的字段在提示显示之后取得焦点（`core/lib/refused-focus.ts` 的 `useRefusedFocus`，`useForm` 写 `shouldFocusError: false`；写法和时机见 M4 交接第 7 节）。模块链接的弹窗（`modules/links/create-update-modal.tsx`）的地址的规则带一句英文的提示（"URL is required"），却不显示它，输入框只是 `aria-invalid`：照同一个写法显示它、以它为描述。
  - 在弹窗里打开的列表读 `useInsideModal()`，在弹窗里 `modal={false}`：`dropdowns/cycle/cycle-options.tsx`、`dropdowns/module/module-options.tsx`（工作项的弹窗里）、`dropdowns/date-range.tsx`（迭代的弹窗里）的 `Combobox.Options`；`core/modals/existing-issues-list-modal.tsx` 的列表本身就是弹窗的内容，核对它打开时后面的页面隐藏着。
  - `core/modals/existing-issues-list-modal.tsx` 每个选项里只在指针经过时显示的链接（`:290`），方向键到不了；已选的工作项的移除按钮（`:175`）没有名字。
- **模块的状态选择没过规则时只变红**（M3 收尾，浏览器核对之后）：`modules/select/status.tsx` 的 `Controller` 带 `rules={{ required: true }}`，没过时按钮的字和图标变红（`:40`、`:45`、`:48`，读 `error`），没有提示，按钮也不 `aria-invalid`（[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.17 第 2 条的第三次扫描找到；上面"正的 `tabIndex`"一条只说到它的 `tabIndex`）。它是 M6 的模块表单。照 M2、M3 的写法：给它经 `t()` 的提示，`CustomSelect` 收 `describedBy`、`invalid`，以提示为描述、恰在提示显示时 `aria-invalid`，被拒绝时取得焦点（写法同 M4 的交接第 7 节）。
- **关闭条件**：M6 的评审写明上面每一项的浏览器核对。

## 4. 地址、复制和 oxlint

- **地址只从路由参数**：迭代、模块、视图的页面从 router store 读地址的 22 个文件（M3/P10 spec 第 5 节 M6 一行列出：11 个，和裁定 F-12 补上的 11 个），改成读路由参数，并加进关键词规则 `project-address-route` 的范围。
- **复制只经 hook**：`cycles/quick-actions.tsx`、`modules/links/list-item.tsx`、`modules/quick-actions.tsx`、`power-k/ui/pages/context-based/cycle/commands.ts`、`power-k/ui/pages/context-based/module/commands.tsx` 改经 `useCopyLink`（`core/hooks/use-copy-link.ts`），失败时经 `t()` 提示；关键词规则 `copy-hook` 中这 5 个例外（`until: "M6/closeout"`）随之删除。两个 quick-actions 今天把复制交给菜单的 `action`（`() => void`），丢下 promise，拒绝没有处理（M3/P11 spec 附录 A.6）。
- **oxlint**：`cycles/modal.tsx` 1 条（`no-shadow`）、`modules/module-card-item.tsx` 5 条（M3/P11 spec 第 5 节），M6 改它们时清零。
- **守卫的阶段**：收尾开始时把 `tools/keywords.json` 的 `phase` 改为 `M6/closeout`，`copy-hook` 的 5 个例外（`until: "M6/closeout"`）那时到期，守卫要求它们已经删除（`tools/keywords.mjs` 的说明、M1 设计 7.4）；每个 Phase 开始时改为 `M6/P<k>`。`analytics` 规则对 `core/services/cycle.service.ts` 的例外写的是 `until: "M6"`，`phase` 改为 `M6/P1` 时就到期（M1 设计 3.4）。
- **关闭条件**：`project-address-route` 的范围含这些文件，`copy-hook` 没有 `M6/closeout` 的例外。

来源：[M3 设计](../../M3-workspace-project/M3-design.md) 13.2 的 M6 一行；M3/P8a、P8b、P9、P10、P11 的 spec 第 5 节和评审第 6 节中交给 M6 的各条；[M3 收尾 spec](../../M3-workspace-project/specs/closeout.md) 第 2 节。

## 5. M4/P16 焦点索引最后读者回执（2026-11-01）

M4 原创建表单的 `ISSUE_FORM` 已无读者，本期删除其枚举、独占数组和映射项；其余四组及 `getTabIndex`、两份 barrel 保留。当前七个调用文件：M6 的 `cycles/form.tsx`、`modules/form.tsx`，M7 的 `views/form.tsx` 与 `inbox/modals/create-modal/{create-root,issue-properties,issue-title,issue-description}.tsx`。视图表单虽由 M7 实现，M6 修改共用项目弹窗宿主时仍按本交接既有分工处理。

全 web 44 处直接 `CustomMenu` JSX 无 `tabIndex` 属性/展开属性；M4 的 `issues/issue-layouts/spreadsheet/columns/header-column.tsx` 仍传 `customButtonTabIndex={-1}`，用于表格列头控件，不是正索引残留。P14 已删除收集箱父项死入口，§3 所记该 `CustomMenu` 正索引与嵌套按钮现属历史；不能据此删除整个菜单焦点接口。`tabIndex ?? customButtonTabIndex` 本期保持，后续按真实最后读者再收口。

最后删除条件：M6/M7 迁移上述表单与两处归档弹窗 `tabIndex={1}` 到自然 Tab 顺序，并完成各自弹窗/portal/回焦回归；全仓确认 `getTabIndex`、枚举/映射及各共享控件的属性读者都迁完后，最后迁移者删除两份 `tab-indices.ts` 和 barrel。共享菜单的非正索引用途须独立核实保留，不以 M4 零正索引等同全部焦点属性可删。未改未来领域键盘顺序、接口或顶部 open 状态。

证据：[P16 review §9](../../M4-issue-core/reviews/P16-web-cleanup-review.md#9-t1-焦点索引与实施入口)、本期 `t1-focus-inventory.json`（逐调用、直接属性、M01 静态输入反例）。

## 处理结果（M6/P1）：第 2 节

4.12 的图已在 M6 设计 §4.3 延伸。P1 落地其中的 `cycles`：`workspace_id`、`project_id` 两条 `CASCADE` 父外键各有不带条件的索引；删项目、删工作区时 `planning` 订阅 `project.deleted`、`workspace.deleted`，在原事务投递 `planning.delete_rows`（沿用请求的 At/By），父范围立即隐藏；60 天物理清理加入 `planning` 的步骤（asset → issue → planning → project → workspace → identity）。模块的表在 P2、关联表在 P3；`core/sidebar/progress-stats/` 的 8 行死成员属前端，结论在 M6/P5 的评审。本交接保持 open。证据：[P1 spec](../specs/P1-cycles-backend.md) 2.4。

## 处理结果（M6/P2）：第 2 节

P2 落地 M6 设计 §4.3 中模块的三张表：`modules`、`module_members`、`module_links` 的 `workspace_id`、`project_id`，子表的 `module_id` 和 `module_members.member_id` 都是 `CASCADE`，各有不带条件的索引。删项目、删工作区沿用 P1 的 `planning.delete_rows`（任务参数不变），由单表改为命名步骤 `cycles → modules → module_members → module_links`，每步按项目、id 升序有界，沿用请求的 At/By，可续跑；60 天物理清理的 `planning` 步骤为 `module_links → module_members → modules → cycles`（子先父）。删除与清理的守卫随之覆盖三张表；漏掉链接步骤的变异让 `TestDeletionRowStepsParentsFirst` 等删除任务的测试失败，清理把 `modules` 放到 `module_links` 之前的变异让 `TestModulePurgeChildrenFirst` 失败。关联表在 P3；`core/sidebar/progress-stats/` 的 8 行死成员属前端，结论在 M6/P5 的评审。本交接保持 open。证据：[P2 spec](../specs/P2-modules-backend.md) 2.4。

## 处理结果（M6/P3a）：第 2 节

P3a 落地 M6 设计 §4.3 中的两张归属表 `cycle_issues`、`module_issues`（归 `issue` 模块）：`workspace_id`、`project_id` 两条 `CASCADE` 父外键各有不带条件的索引，`TestCycleIssuesSchema`、`TestModuleIssuesSchema` 依次物理删除项目、工作区，核它们的行随之而去。它们是 `issue` 的表，删项目、删工作区走 `issue` 已有的 `issue.delete_rows`（订阅 `project.deleted`、`workspace.deleted`，在原事务投递，父范围立即隐藏，沿用请求的 At/By）：`DeletionRowSteps` 在 `issue_labels` 之后加 `cycle_issues`、`module_issues` 两步，剩余探查加两个分支，按批有界、可续跑；`planning.delete_rows` 不变。60 天物理清理的 `issue` 步骤在 `issues` 之前加 `module_issues`、`cycle_issues`（模块次序 asset → issue → planning 使它们先于 `cycles`、`modules`）。项目与工作区的删除守卫（`TestDeletingAProjectLeavesNoUndeletedRowUnderIt`、`TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt`、`TestAProjectDeletionRefusedAtItsCommitChangesNoRow`）与清理目录（`TestPurgeCatalogCoversTablesAndIncomingKeys`）随之要求两张表有行并核软删；删除任务漏掉 `module_issues` 一步的变异（`t3-mut-delete-rows-no-module`）让项目与工作区的删除守卫失败。`core/sidebar/progress-stats/` 的 8 行死成员属前端，结论在 M6/P5 的评审。本交接保持 open。证据：提交 `2ce5591cab00f32200502c05e54b7c1248d02d03`（表）、`f9e62906607ba16f2b0a7ee21ced6db539ce03c8`（删除任务、清理与守卫）；[P3a spec](../specs/P3a-issue-membership.md) 2.5。

## 处理结果（M6/P3b）：第 2 节

P3b 落地 M6 设计 §4.3 中的 `cycle_user_properties`、`module_user_properties`（归 `planning`）：`workspace_id`、`project_id` 两条 `CASCADE` 父外键各有不带条件的索引，`TestPlanningPreferencesSchema` 依次物理删除项目、工作区，核它们的行随之而去。删项目、删工作区走 `planning` 已有的 `planning.delete_rows`：`DeletionRowSteps` 在 `module_links` 之后加两步，剩余探查加两个分支，按批有界、可续跑，沿用请求的 At/By；60 天物理清理的 `planning` 步骤为 `module_user_properties → module_links → module_members → modules → cycle_user_properties → cycles`（子先父）。项目与工作区的删除守卫（`TestDeletingAProjectLeavesNoUndeletedRowUnderIt`、`TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt`）与清理目录（`TestPurgeCatalogCoversTablesAndIncomingKeys`）随之要求两张表有行并核软删；删除任务漏掉 `module_user_properties` 一步的变异（`t8-mut-steps-no-module-preferences`）让删除守卫失败。4.12 的图至此延伸到 M6 的全部表；`core/sidebar/progress-stats/` 的 8 行死成员属前端，结论在 M6/P5 的评审。本交接保持 open。证据：提交 `e239a557471a048f56453f94676f2ecec22bf238`；[P3b spec](../specs/P3b-list-engine.md) 2.4、2.5。

## 处理结果（M6/P5a）：第 1–5 节（迭代）

- **第 1 节**：项目包装层在调用者是有效成员、项目未归档且 `cycle_view` 打开时取迭代（`use-project-fetch.ts`，vitest 核对三种不取的情形）；迭代的筛选 store 不跟随 `router`（R4），没有要释放的反应，`store-context.test.ts` 核对（模块的在 P6）；转移弹窗重建，拒绝经 `useRefusalToast`/`errorMessageKey`；`ProjectLevelModals` 以 `projectId` 为 key 并在卸载时关上三个开关，G19 页面故事核对新建弹窗开着时后退、换项目、换工作区都结束它；迭代详情的子树以 `cycleId` 为 key，G19 核对经面包屑换迭代后旧迭代迟到的进度不写进新页面。迭代 `user-properties` 在 P5b，`existing-issues-list-modal.tsx` 在 P7。
- **第 2 节**：`core/sidebar/progress-stats/` 的 8 行：`state_group.tsx` 的 `total` 已在 M4/P10a 删除（`fdd3ef2e`）；其余是误报，而且这些文件在 P5a 之后只剩模块的读者（迭代改用 `distribution-stats.tsx`），交 P6。
- **第 3 节**（浏览器核对在 G1、G3、G8、G19 页面故事，三种引擎）：新建/修改弹窗的 Tab 在弹窗里转、Escape 关闭、焦点回到打开它的按钮、弹窗里的日期选择可用；删除与归档的确认框被拒绝时留着并说明、只在成功后关闭；新建/修改、归档、转移三个弹窗的标题经 `ModalTitle`（故事断言弹窗的名字）；自定义日期筛选的标题经 `t()`；迭代表单的 `PROJECT_CYCLE` 正 `tabIndex` 与归档按钮的 `tabIndex={1}` 删除；快捷操作的菜单按钮不再嵌按钮、以迭代命名（`common.item_options`）；筛选按钮与其各节有名字与 `aria-expanded`；名字字段的提示是 `FieldError`、`aria-invalid`、被拒时取得焦点；页头的切换迭代可用键盘。右键菜单：关着时不可交互（读屏看不到、各项取不到焦点），右键一行打开与按钮菜单同一组项、Escape 关闭后焦点回到该行、选“修改”打开修改弹窗（G3，提交 `9831341cd2de3952f78db0778724840dde207a6c`）。模块的各项在 P6，`existing-issues-list-modal.tsx` 在 P7。
- **第 4 节**：迭代领域的文件改读路由参数，加入 `project-address-route`（含 P10 名单之外的读者；P10 名单中的 `cycles/(detail)/mobile-header.tsx` 已不存在）；迭代的两处复制经 `useCopyLink`，`copy-hook` 例外 12 → 10；`cycles/modal.tsx` 的 `no-shadow` 已不存在（M4/P16 起全仓为错误）；`phase` 已为 `M6/P5`。
- **第 5 节**：`cycles/form.tsx` 改为自然的 Tab 次序，`ETabIndices.PROJECT_CYCLE` 与它的表删除；迭代归档按钮的 `tabIndex={1}` 删除；`getTabIndex` 保留给模块（P6）与 M7。
- 证据：提交 `b4911394dc2a92d4bd0224e572902e6d45275941`、`a21686c4a9827c1fb98d28eafa67c35a5b1150f7`、`5cdd61c28bba772e2b69332c1b96fb26cb900e0a`、`11ef5b52ab5b8497d887c0fe2c4d250fd0ce55b9`、`a415c86eb74295cd132d449c0ad49b512f93e91f`、`b6a0b7dfe45aeb2fce97abda9b3e6ed14f1e4a10`、`3104995cc63d45b1eacfdf4c09cc02fbfb2391e5`、`f497055055f00e0cdbfa74e08f5186d40a1297d3`、`1523da8682d55866e8c08f6da5eac0db34d784ba`、`5524a44dd6576da362a2588ec4079d307d565747`。本交接保持 open。

## 处理结果（M6/P5b）：第 1 节（迭代的 user-properties）

- **迭代的 `user-properties`**：迭代页的显示设置改走 nerve 的 `GET`/`PATCH /api/v0/me/cycles/{cycle_id}/preferences`（P3b 的接口）。网页经 `CyclePreferencesService`（`core/services/planning/cycle-preferences.service.ts`，只持本代的 `ApiClient`，由每一代的 `IssueRootStore` 建立的 `CycleIssuesFilter` 构造，没有模块级的实例）与 `CycleIssuesFilter`（`core/store/issue/cycle/filter.store.ts`）：按迭代 id 存取，不读 `router`（store 没有 `rootIssueStore.cycleId`，没有反应），随所在的一代退役，退役后迟到的读与写都不写入（`assertActive`）；读的先后、每个字段一个 `oneAtATime`、整字段替换与显示映射在与项目页共用的基类 `PagePreferencesFilter`（[P5b spec](../specs/P5b-cycle-list.md) §2.4、R3）。取数在迭代页的 `CycleLayoutRoot` 经 `useSessionSWR`，键带迭代 id，项目的迭代关闭时不取；列表的挂载键同样带迭代 id。web 中已没有 `/user-properties/` 的地址，Plane 的 `…/cycles/{id}/user-properties/` 会被 P5a 的关键词规则 `old-cycle-urls`（以 `/api/workspaces/…/cycles` 锚定）拦住。单测 `core/store/issue/cycle/filter.store.test.ts` 10 例（读到 nerve 的默认值、只发改的字段且确认后才显示、被拒不改并交给调用者、较旧的读不盖较新的修改、两个迭代互不影响、换代后迟到的读与写都不写入、没读过的整字段替换只发一个 `PATCH` 并以回答种下、不随路由、看板折叠按 CYCLE 存取），8 个能编译的变异（较旧的读覆盖、整字段改为合并、退役后照写、退役后照读、版本与队列的键去掉迭代 id、以 PROJECT 存折叠、用项目偏好的 service、加读 router 的 getter）各让它失败；G15 页面故事一在真浏览器核对改布局、分组与显示属性后重载仍在、项目页的偏好不变，迭代关闭时页面不读本页偏好。`existing-issues-list-modal.tsx` 仍在 P7。
- 证据：提交 `b205756f2789f57126fa5d1dcc7ade87141c89f2`、`781ffefc719c93cb0f30324de88408d1dfc39823`、`77d52b4bad905ca9b3de2077d83e88a524376a3a`、`f4a3700732eeda348ebf160b6aa8062fb168b1f1`。本交接保持 open。

## 处理结果（M6/P6a）：第 1–5 节（模块）

- **第 1 节**：
  - 项目包装层在调用者是有效成员、项目未归档且 `module_view` 打开时取本项目未归档的模块（`PROJECT_MODULES`，`use-project-fetch.ts`；开关经与取数门共用的谓词 `planningOn`），vitest 核对非成员、已归档、开关关闭时不取（变异 `t2-mut-wrapper-module-view`、`t2-mut-wrapper-module-member`、`t5-mut-wrapper-own-switch`）。Plane 的 `fetchModulesSlim` 取的是工作区级的列表，nerve 没有这个接口；通知页对它的取数随 `use-workspace-issue-properties.ts` 删除（工作区级的模块列表属 M7）。
  - 模块的筛选 store 不跟随 `router`、没有反应、不写 `localStorage`（[P6a spec](../specs/P6a-web-modules.md) R4），“退役的模块筛选的反应在每次换路由时仍写 localStorage”随之不再存在，`store-context.test.ts` 核对；关闭条件里的两个反应至此都已不存在（迭代的在 P5a）。
  - `modules/modal.tsx` 读 `error?.detail ?? error?.error` 的两处删除：字段错误经 `useFormRefusal` 显示在字段下（`member_ids[i]` 归到成员一栏），其余经 `useRefusalToast`/`errorMessageKey`。
  - 新建模块的弹窗仍由 `ProjectLevelModals`（P5a 已以 `projectId` 为 key、卸载时关上三个开关）渲染，改读新 store；G19 页面故事二核对新建弹窗开着、模块已发出时后退、换项目、换工作区都结束它，迟到的回答建了模块而不在新页面提示。修改、归档、删除的弹窗归页面级的 `ModuleActionsHost`，挂在模块的列表、归档目录、详情三个布局上（项目的 key 之下）。
  - `modules/(detail)/[moduleId]` 的子树以 `moduleId` 为 key，宿主在 key 之内（变异 `t8-mut-subtree-unkeyed`）；预览的侧栏以所预览的模块为 key（`t10-mut-peek-unkeyed`）；G19 页面故事三经页头的切换从 Alpha 打开 Beta，Alpha 迟到的进度不写进 Beta 的页面。
  - 模块页偏好的 `user-properties` 在 P6b，`existing-issues-list-modal.tsx` 在 P7。
- **第 2 节**：`core/sidebar/progress-stats/` 的 8 行（`label.tsx` 5 行、`state_group.tsx` 3 行）随模块侧栏改用共用的分布与页签、`progress-stats/{assignee,label,state_group}.tsx` 删除而了结；模块的表在删除关系图上的延伸已在 P2、P3a、P3b。
- **第 3 节**（浏览器核对在 G9、G10、G13、G19 页面故事，三种引擎）：
  - 新建/修改弹窗以标题为名（“Create Module”，经 `ModalTitle`），打开时名字取得焦点，Tab 在弹窗里按次序转，Escape 关闭，焦点回到打开它的按钮，弹窗里选日期可用（G9）；归档与删除的确认框被拒绝时留着并说明、只在成功后关闭（G13）；归档确认框与链接弹窗的标题经 `ModalTitle`。
  - 自定义日期筛选：模块自己的 `dropdowns/filters/{start-date,target-date}.tsx` 删除，与迭代共用 `core/components/planning/filter-date.tsx`，标题经 `t()`。
  - 正的 `tabIndex`：模块表单的 `PROJECT_MODULE` 删除，归档按钮的 `tabIndex={1}` 删除（变异 `t7-mut-archive-positive-tabindex`）；`ModuleStatusSelect` 不再收 `tabIndex`；`CustomSelect` 的 `tabIndex` 在 `@nerve/ui` 共用的 `IDropdownProps` 上，随第 5 节的最后迁移者收口。
  - 嵌套的按钮：`modules/quick-actions.tsx` 的菜单按钮不再嵌一个按钮，以模块命名（`common.item_options`），G19 核对它的名字与 `aria-expanded`。
  - 右键菜单：关着时 `inert`，右键一行打开、首项取得焦点，Escape 关上、焦点回到行的链接（G19 故事三）。
  - 下拉框：负责人与成员的下拉（`CustomSearchSelect`）在 G9 中选人，只列项目里不是访客的成员；页头切换模块的下拉在 G19 打开另一个模块。共用组件的焦点轮廓在整审 re-69 之后的修复轮 F2 核对（G19 故事三，三种引擎；量法是与 M4 C12 共用的 `focusOutline`：轮廓画出，视口与隐藏溢出的祖先都不裁它）：页头的搜索（`IconButton`）、排序（`CustomMenu` 与 `getButtonStyling`）、筛选与“Add Module”（`Button`），行与卡片的菜单（`CustomMenu`），详情页页头的详情开关（`IconButton`）与菜单，Tab 到达时都完整；菜单按钮放进隐藏溢出的盒子、去掉轮廓两个产品变异（`f2-mut-menu-in-hidden-box`、`f2-mut-menu-no-outline`）各让它失败。逐个 Tab 量下来，这四种以外的控件（行的链接、侧栏链接一行的链接与“Add link”、面包屑的链接被裁，日期范围按钮与进度页签没有轮廓）迭代页同样，交 P8（[P6a spec](../specs/P6a-web-modules.md) §5）。
  - 筛选的按钮：页头的筛选按钮以 “Filters” 为名、有 `aria-expanded`，各节也有，Escape 关上（G19）；筛选面板的搜索不再 `autoFocus`（变异 `t11-mut-filter-search-autofocus`）；排序的按钮以所选的排序为名。
  - 每一行的控件以行为名：模块一行的菜单按钮（`common.item_options`），链接一行的改、删、复制按钮以链接为名。
  - 字段的提示：名字的提示是 `FieldError`、`aria-invalid`、被拒时取得焦点（G9：空的名字与只有空格的名字）；链接弹窗的地址为空时提示显示在字段下、不发出（G10）；模块的状态选择没过规则时有经 `t()` 的提示、`aria-invalid`、被拒时取得焦点（组件测试 `select/status.test.tsx`；界面上走不到：新建默认“积压”，修改带着模块的状态）。
  - 弹窗里的列表：模块表单的日期范围在弹窗里可用（G9）；`dropdowns/module/module-options.tsx` 与 `existing-issues-list-modal.tsx` 在 P7。
- **第 4 节**：模块领域的路由页面、组件与 hook 改读路由参数，随各自的 Task 加入 `project-address-route`（带样例）；它的内容模式加了嵌套一层的解构分支（模块的两个页头原以 `const { workspace: { workspaceMemberIds } } = useMember()` 读 router 驱动的 getter），负责人、成员的筛选改读 `getWorkspaceMemberIds(地址的 slug)`；模块的三处复制（快捷操作、链接一行、命令面板的模块页）经 `useCopyLink`/`useCopyText`，`copy-hook` 没有 `M6/closeout` 的例外了；`modules/module-card-item.tsx` 的 5 条 oxlint 随卡片的改写清掉；`phase` 为 `M6/P6`。本节的关闭条件至此已满足。
- **第 5 节**：`modules/form.tsx` 改为自然的 Tab 次序，`ETabIndices.PROJECT_MODULE` 与它的表删除；模块归档按钮的 `tabIndex={1}` 删除。`getTabIndex` 余下的读者只在 M7：`views/form.tsx`（`PROJECT_VIEW`）与收集箱新建弹窗的四个文件（`INTAKE_ISSUE_FORM`），M7 是最后的迁移者。
- 证据：提交 `d71ab782f11899b7ae45452ac8cf7e0d47a73c89`、`6a088615629ba2c7d127e53cea25cc0376c1e2b5`、`3f46edea4510ebdee8789114a01d28809f1ead8f`、`cd366811af816d1ed317aa7491dfa46cdfdc6f54`、`e9fc041b4a1cae06173fa29ec9b03f9ad0c109ca`、`c09d4f8d9d2544724887ad1ad34a5c6cf0b44ffc`、`d8c87d1adfceb28b9200a446f7c5748551190e4a`、`82dac1efd9dac3fe7b7d99bdcac18410cd12bb0a`、`e2d967a18fc3ba0f02210e51f9e407d4f1893c5f`、`dc9172b0fecf86c716bf50394b03c2a4e4a6fd6d`、`b1d2578e2aab5a9769ea7f71cbd7dc6f6b9e232d`、`b8ed29df59bcb59fc30adfbafb8c79748af992b9`、`dbbefbd60765a849a388e89e48b7f92b22fb8316`（整审 re-69 之后的修复轮 F2：共用组件的焦点轮廓）。本交接保持 open。

## 处理结果（M6/P6b）：第 1 节（模块的 user-properties）

- **模块的 `user-properties`**：模块页的显示设置改走 nerve 的 `GET`/`PATCH /api/v0/me/modules/{module_id}/preferences`（P3b 的接口）。网页经 `ModulePreferencesService`（`core/services/planning/module-preferences.service.ts`，只持本代的 `ApiClient`，由每一代的 `IssueRootStore` 在建立 `moduleIssuesFilter` 时构造，没有模块级的实例）与按页的基类 `PagePreferencesFilter`（以这个 service 与 `EIssuesStoreType.MODULE` 直接构造，[P6b spec](../specs/P6b-module-list.md) §2.4、R2）：按模块 id 存取，不读 `router`（没有 `rootIssueStore.moduleId`，没有反应），随所在的一代退役，退役后迟到的读与写都不写入（`assertActive`）；读的先后、每个字段一个 `oneAtATime`、整字段替换与显示映射是与项目页、迭代页共用的基类的。迭代页的 `CycleIssuesFilter`（上文 P5b 的回执）只差 service 与 store 类型两个参数，P6b 一并删去，两域都直接构造基类，迭代页的测试只改构造处、断言不改。取数在两域共用的布局根经 `useSessionSWR`，键带模块 id（`["MODULE_ISSUE_FILTERS", 工作区, 模块]`），项目的模块关闭时不取；列表的挂载键同样带模块 id。web 中已没有 `/user-properties/` 的地址，Plane 的 `…/modules/{id}/user-properties/` 会被 P6a 的关键词规则 `old-module-urls` 拦住。单测 `core/store/issue/module/page-preferences.test.ts` 11 例（按模块 id 读到 nerve 的值并照项目页显示词、经接口存下的迭代条件不显示且整字段替换时只发网页的条件、只发改的字段且确认后才显示、被拒不改并交给调用者、较旧的读不盖较新的修改、两个模块互不影响、换代后迟到的读与写都不写入、没读过的整字段替换只发一个 `PATCH` 并以回答种下、在它之后发出的读答得更旧也不盖它、不随路由、看板折叠按 MODULE 存取）与 `module-preferences.service.test.ts` 3 例，变异（较旧的读覆盖、以 CYCLE 建模块页的偏好、退役后迟到的读照写，台账 T5）各让它失败；G15 页面故事一在真浏览器核对每种布局写进本页偏好、分组与显示属性改后由同一人另开的浏览器载入仍在、项目页的偏好不变，模块关闭时页面不读本页偏好与列表。`existing-issues-list-modal.tsx` 仍在 P7。
- 证据：提交 `85d3a0cc435c7e248cce06b0b63ed35f403f2fb0`（`CycleIssuesFilter` 删除，迭代页偏好直接构造基类）、`c09f81134fc29b9454b255fba48f681bfd42ab05`（模块页偏好的 service 与筛选）、`d4eb1b082db200be6a2419160663fed00e8fccd5`（布局根取模块页偏好）、`c65f7d1bca6fdff29a44a397c089fc4e67a4623e`（G15 页面故事）。本交接保持 open。

## 处理结果（M6/P7a）：第 3 节（工作项的迭代、模块下拉框）

- **弹窗里的列表**：Plane 的 `dropdowns/cycle/cycle-options.tsx`、`dropdowns/module/module-options.tsx` 已被 M4 删除；P7a 重建的两个下拉框（`web/apps/web/core/components/dropdowns/{cycle,module}/`）建在共用的 `CustomSearchSelect` 上，它的列表 `modal={false}`，在工作项的弹窗里可用。浏览器核对在 G4、G11 的页面故事，三种引擎（`t14-three-engines-2`、`t14-repeat3-three-engines-2`、`t14-full-pages-2`）：G4 在迭代页新建的弹窗里打开迭代一栏，列表在按钮旁，选未开始的迭代后提交，只发一个创建请求、带所选的迭代；G11 在行的修改弹窗里打开模块一栏，列表在按钮旁，去掉一个模块之后 Escape 只关上列表、弹窗留着，保存。`core/modals/existing-issues-list-modal.tsx` 在 P7b。
- **每一行的控件以它那一行为名**：列表与看板的行、表格的格、草稿的行里的迭代、模块下拉框以 sr-only 的“字段: 值, 工作项”为名（不加新键，照标签下拉框与子工作项按钮的拼接；已归档的值带说明，如 “Cycle: Old, Archived, Legacy”）。组件测试按行的名字找控件（`issue-layouts/properties/row-planning-properties.test.tsx`、`issue-layouts/spreadsheet/planning-columns.test.tsx`、`workspace-draft/draft-planning.test.tsx`），去掉行名的变异 `t8-mut-row-unnamed`、`t9-mut-cell-unnamed` 各让它们失败；G4 在项目页的列表里以这个名字找到那一行的迭代控件。
- **嵌套的按钮**：模块下拉框的按钮里不放 Plane 的带 “x” 的移除按钮，已归档的模块经列表去掉（[P7a spec](../specs/P7a-work-item-planning.md) R12）；`dropdowns/module/dropdown.test.tsx` 以 DOM 核对按钮里没有按钮（无字的嵌套按钮的变异 `t7-mut-nested-remove-2` 让它失败）。
- **字段的提示**：工作项弹窗与草稿弹窗里的迭代、模块两栏照 M2、M3 的写法：nerve 的 422 `cycle_id`、`module_ids[i]` 与 409 `planning.cycle_completed` 显示在所指的一栏下（`FieldError`），控件以它为描述、`aria-invalid`，被拒的一栏取得焦点（`useRefusedFocus`）；两份未归档的列表读被拒时两栏下说明并可重试（`issue-layouts/roots/planning-edit.test.tsx`、`roots/planning-copy.test.tsx`、`issue-modal/draft-planning.test.tsx`；变异 `t11-mut-module-error-toast`、`t11-mut-cycle-completed-toast`、`t11-mut-field-not-invalid`、`t12-readfail-mut-no-refusal-shown`、`t13-mut-publish-toast-only` 各让它们失败）。
- **下拉框**：两个新的 `CustomSearchSelect` 调用方以键盘核对（G4：焦点在按钮上按 Enter 打开、搜索框取得焦点、Escape 关上后焦点回到按钮）；它们的焦点轮廓没有量，随 P8 的最终矩阵。
- 其余各项：第 1 节与本节的 `existing-issues-list-modal.tsx`（搜索接口、打开时后面的页面、选项里只在指针经过时显示的链接、移除按钮的名字）在 P7b；第 1 节视图的跟进与第 4 节四个视图文件的地址在 P7c（P7a spec R1 的 X1）。
- 证据：提交 `43d53c3521185f5de26b3138090dd9d49d8a8d3c`、`45889a0b2c330bbc1cc731d19cd94e97816e59ae`（两个下拉框）、`adb53eb61358209b5673abcad8ec4730c4f7f13c`、`f1646370f1396b755be0a03c5326f7d7e8a8d4fc`、`ca0f10be804552848684e6f850fff4a66724c819`（行与表格的格）、`2a8faa016131fbb4d2fb1aa6da6b604264191101`、`12070947d270fc93a47de19d8681a3f54afd9b0a`、`cc5dafeab6cd8a9a8f637d080baedfbdc9d61f1c`、`16473c1bc469203eb837a3088d1bc1ae0077f213`、`45b4c2ab31dac12fdd9634c572de4e0c36703d65`（弹窗与草稿的两栏）、`40cef888032479ddc8708857fb1789642b19701d`（G4、G11 的页面故事）。本交接保持 open。
