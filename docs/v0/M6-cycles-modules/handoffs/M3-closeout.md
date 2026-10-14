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
- **弹窗的名字**：Headless UI v2 的 `Dialog` 只以它的 `DialogTitle` 为名（`aria-labelledby`），普通的标题不给弹窗名字，读屏读出一个没有名字的对话框；M3 收尾的修复轮在 `@nerve/ui` 的 `ModalCore` 旁边导出 `ModalTitle`（它就是 `DialogTitle`，`h3`），`AlertModalCore` 的标题经它，M2、M3 的弹窗的标题都经它（[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.5）。M6 的确认框（`AlertModalCore`）因此都有名字；别的弹窗的标题改经 `ModalTitle`，没有看得见的标题的给一个 `sr-only` 的（不加第二个看得见的），标题的文字经 `t()`。以普通标题的：`cycles/form.tsx:100`（`cycles/modal.tsx`）、`modules/form.tsx:111`（`modules/modal.tsx`）、`cycles/archived-cycles/modal.tsx:67`、`modules/archived-modules/modal.tsx:67`、`cycles/transfer-issues-modal.tsx:92`（`h4`）、`modules/links/create-update-modal.tsx:90`；没有标题的：`core/modals/existing-issues-list-modal.tsx`（搜索的列表，`:222` 的 `h5` 是一组的说明）。M6 的故事打开它们时断言 `expect(modalDialog(page)).toHaveAccessibleName(…)`。
- **正的 `tabIndex`**：迭代、模块、视图的表单（`PROJECT_CYCLE`、`PROJECT_MODULE`、`PROJECT_VIEW`）同 M4 的交接第 7 节：Tab 离开它们的弹窗，照样删去；迭代、模块的归档弹窗的归档按钮（`cycles/archived-cycles/modal.tsx`、`modules/archived-modules/modal.tsx`）是 `tabIndex={1}`（M3/P10 的裁定 F-18）。模块的状态 `ModuleStatusSelect`（`modules/select/status.tsx`）给 `CustomSelect` 的 `tabIndex` 落在它的按钮上（裁定 F-13），删去模块表单的正的 `tabIndex` 之后，它不再收 `tabIndex`，`CustomSelect` 不再读它（裁定 F-16）。M4、M6、M7 中最后删去正的 `tabIndex` 的那一个删除 `getTabIndex` 和两份 `tab-indices.ts`。
- **嵌套的按钮**：菜单自己的按钮里又放了 `IconButton` 的两处（M3 收尾找到）：`cycles/quick-actions.tsx`、`modules/quick-actions.tsx`；改法照 M3 收尾的项目一行（`CustomMenu` 的 `customButtonClassName`，内容是图标，不再嵌一个按钮）。
- **右键菜单**：迭代、模块、视图的列表的右键菜单关着时 `inert`（裁定 F-14），在 M6 的页面上核对右键打开它、点它的项。
- **下拉框**：C7 没有核对 M6 页面上的 `CustomSearchSelect` 调用方（成员下拉框的 2 个文件，迭代、模块、视图的页头），对接时用键盘核对。共用组件的焦点轮廓是否被 `overflow-hidden` 裁掉，在 M6 的页面上核对。
- **筛选的按钮**（M3 收尾 Task 12，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.9）：`FiltersDropdown` 默认的按钮现在就是弹层的 `Popover.Button`（以 `title` 为名、`aria-expanded`、Escape 之后焦点回到它，指针点开之后焦点留在它上面，同 M4 的交接第 7 节）；M6 的 8 个调用方（6 个用默认的按钮）在收尾 spec 附录 A.4 逐个核对过，没有故事走到，M6 的筛选的故事落地时断言按钮的名字、`aria-expanded` 和 Escape 之后焦点回到它。
- **每一行的控件以它那一行为名**（M3 收尾的规则，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.9：一行一个、每行都一样的控件以那一行的东西为名，同 F6 的项目一行、"{name} settings"、"Remove X"）：迭代一行的菜单按钮 `cycles/quick-actions.tsx:141` 和模块一行的 `modules/quick-actions.tsx:135`（都叫"Options"）以迭代、模块的名字为名：两处已有的 `cycleDetails?.name`、`moduleDetails?.name`；M3 收尾已加的带 `{name}` 的键 `common.item_options`（"{name} options"、"{name} 的选项"，项目成员页、标签页一行的菜单读它）。
- **关闭条件**：M6 的评审写明上面每一项的浏览器核对。

## 4. 地址、复制和 oxlint

- **地址只从路由参数**：迭代、模块、视图的页面从 router store 读地址的 22 个文件（M3/P10 spec 第 5 节 M6 一行列出：11 个，和裁定 F-12 补上的 11 个），改成读路由参数，并加进关键词规则 `project-address-route` 的范围。
- **复制只经 hook**：`cycles/quick-actions.tsx`、`modules/links/list-item.tsx`、`modules/quick-actions.tsx`、`power-k/ui/pages/context-based/cycle/commands.ts`、`power-k/ui/pages/context-based/module/commands.tsx` 改经 `useCopyLink`（`core/hooks/use-copy-link.ts`），失败时经 `t()` 提示；关键词规则 `copy-hook` 中这 5 个例外（`until: "M6/closeout"`）随之删除。两个 quick-actions 今天把复制交给菜单的 `action`（`() => void`），丢下 promise，拒绝没有处理（M3/P11 spec 附录 A.6）。
- **oxlint**：`cycles/modal.tsx` 1 条（`no-shadow`）、`modules/module-card-item.tsx` 5 条（M3/P11 spec 第 5 节），M6 改它们时清零。
- **守卫的阶段**：收尾开始时把 `tools/keywords.json` 的 `phase` 改为 `M6/closeout`，`copy-hook` 的 5 个例外（`until: "M6/closeout"`）那时到期，守卫要求它们已经删除（`tools/keywords.mjs` 的说明、M1 设计 7.4）；每个 Phase 开始时改为 `M6/P<k>`。`analytics` 规则对 `core/services/cycle.service.ts` 的例外写的是 `until: "M6"`，`phase` 改为 `M6/P1` 时就到期（M1 设计 3.4）。
- **关闭条件**：`project-address-route` 的范围含这些文件，`copy-hook` 没有 `M6/closeout` 的例外。

来源：[M3 设计](../../M3-workspace-project/M3-design.md) 13.2 的 M6 一行；M3/P8a、P8b、P9、P10、P11 的 spec 第 5 节和评审第 6 节中交给 M6 的各条；[M3 收尾 spec](../../M3-workspace-project/specs/closeout.md) 第 2 节。
