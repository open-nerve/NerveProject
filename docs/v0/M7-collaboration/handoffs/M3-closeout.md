---
status: open
from: M3/closeout
to: M7
created: 2026-10-13
---

# M3 交给 M7：收藏、通知、"最近"、视图和收集箱的取数，跨项目的标签，这些页面上留下的前端

M3（工作区与项目）的页面挂载时不再取 M7 的数据（收藏、未读通知数、首页的"最近"、视图、分诊状态），工作区级的标签删除，这些页面在 M3 里走不到（[M3 设计](../../M3-workspace-project/M3-design.md) 3.1）。下面每一节来自 M3 设计 13.2 的 M7 一行，和 M3 各 Phase、M3 收尾交给 M7 的各条（[收尾 spec](../../M3-workspace-project/specs/closeout.md) 第 2 节的分诊表写明来处）。路径在 `web/apps/web/core/components/` 下的，只写这之后的一段。

## 1. 接口和挂载时的取数

- 侧边栏的收藏、未读通知数、首页的"最近"小部件（连同它的工作项预览 `peek-overviews.tsx`，M3/P8a 随小部件删除，M1 设计 3.14）、项目的视图和分诊状态的取数、项目侧边栏的 `intake_count`（侧边栏的数字和收集箱 store 写它的两段），随各自的接口加回，照 M3 设计 7.1 由权限启用（M3 设计 3.1；M3/P8a、P8b spec 第 5 节）。
- 打开 `intake_view` 时建默认收集箱、分诊状态的接口（M3 设计 3.17）：分诊状态由 `"group" = 'triage'` 识别，每个项目至多一个（`states_project_id_triage_key`），它的名称也占用项目内的状态名（M3/P7a spec 第 3 节第 6 条）；收集箱的分诊状态的前端（store、下拉框、类型、`@nerve/propel` 的两个图标）在 M3/P8b 的 Task 7 删除，随接口加回。
- 跨项目的标签列表是各项目列表的并集，没有工作区级的标签（M3 设计 3.16、11.5）；工作区级标签的前端（store 的四个成员、筛选的标签一项、按标签分组）在 M3/P8b 删除，随它加回，连同工作区一级显示的工作项（所有工作项、视图、草稿、通知的预览）上的标签（M3/P8b spec 第 3 节第 21 条）。
- 项目的收藏：`is_favorite`、卡片上的收藏按钮、建项目之后加入收藏、`favorite.store.ts` 的项目分支、`favoriteProjectIds` getter（M3 设计 3.2），和归档项目时的收藏处理。
- 收集箱查重 `inbox/modals/select-duplicate.tsx` 改用 M4 的工作项搜索（M3 设计 7.3）；工作区视图的筛选列（M3 设计 3.18）。
- **关闭条件**：每一项的接口版本和页面版本的故事；挂载时的取数按权限启用，S2 的清单随之改（M3/P8b 的裁定 D5）。

## 2. stores 和会话（总体设计 7.7）

- 读工作区列表而没有取数的三处：新手引导的 `hasWorkspaces`、个人设置侧边栏的工作区、power-K 在工作区页面之外打开工作区的菜单；直接打开 `/settings/profile/*` 时三处都没有工作区。创建成功之后再取列表被拒绝，被当成"创建失败"，重试时遇到"slug 已占用"，根在 `createWorkspace` 只写进已经取到的列表（M3/P8a 评审第 6 节）。
- 已挂载的工作区包装层换 slug 时不重取列表，在别的标签页加入的工作区要到下一次挂载才显示；要改，改 `useWorkspaceFetch` 这一个判断（M3/P9 spec 第 5 节，P19）。
- `ProjectViewStore.loader` 和 `fetchedMap` 没有写者了：视图的取数设置它们，否则删除它们（M3/P8a 评审的 P30）。通知页的收集箱项经原来的 `useSWR` 读项目（`workspace-notifications/root.tsx`），写项目自己的读，可以盖过包装层的读：照 7.1 改成按权限的取数（M3/P8b 的 PF-L3）。收藏的 `workspace/sidebar/favorites/favorite-items/common/helper.tsx` 的 `console.error` 改成提示（M3/P9 spec 第 5 节）。
- `@nerve/types` 的 `workspace-notifications.ts` 的 17 行死成员（`domains.mjs` 按路径误归到 M3，M3/P11 spec 附录 A.9）。
- **界面状态在以 id 为 key 的布局之下**（M3 收尾，负责人 2026-10-13 的 B，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.7）：工作区的内容（`app/(all)/[workspaceSlug]/layout.tsx` 的 `WorkspaceContentWrapper`）以路由的 slug 为 key，项目的页面和项目的设置（两个 `[projectId]/layout.tsx`）以路由的项目为 key，换工作区、换项目（后退、前进也是）时其下重新挂载，组件状态里的弹窗和状态不带过去。store 持有的弹窗开关由渲染它的宿主在卸载时关上（`useLayoutEffect` 的清理，像 `WorkspaceLevelModals`）；宿主在 key 之上时还要在范围结束时关上。收集箱、视图、通知、收藏的弹窗和选中放在这些布局之下，不要放到它们之上（路由的根、`WorkspaceAuthWrapper`、跨工作区的 store 或 `localStorage`）。
- **嵌套的 id 没有 key**（M3 收尾）：视图的页面 `views/(detail)/[viewId]`、`workspace-views/[globalViewId]`只在工作区、项目的 key 之下，换一个视图（后退、前进也是）时页面留着，开着的弹窗和组件的状态跟过去。M7 接上这些页面时以页面自己的 id 给它的子树 key，并加一个后退的检查。
- **关闭条件**：上面每一处有 vitest 或故事；这 17 行在 M7 的评审中有结论。

## 3. 键盘、弹窗和下拉框

- **弹窗**：M3 收尾把 `ModalCore` 改成 Headless UI v2 的 `Dialog`（[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.5）：打开时后面的页面 `inert`、`aria-hidden`，请求在路上时关不掉。收集箱、视图、通知的弹窗（通知的打盹 `workspace-notifications/sidebar/notification-card/options/snooze/modal.tsx` 在内）对接时逐个核对 Tab、Escape、点外面、关上之后的焦点和弹窗里经 portal 渲染的东西（收尾 spec 附录 A.3 只证明了 M3 页面上的几种）；`AlertModalCore` 的确认框照 A-M2，按钮的文字经 `t()` 传入。
- **弹窗的名字**：Headless UI v2 的 `Dialog` 只以它的 `DialogTitle` 为名（`aria-labelledby`），普通的标题不给弹窗名字，读屏读出一个没有名字的对话框；M3 收尾的修复轮在 `@nerve/ui` 的 `ModalCore` 旁边导出 `ModalTitle`（它就是 `DialogTitle`，`h3`），`AlertModalCore` 的标题经它，M2、M3 的弹窗的标题都经它（[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.5）。M7 的确认框（`AlertModalCore`）因此都有名字；别的弹窗的标题改经 `ModalTitle`，没有看得见的标题的给一个 `sr-only` 的（不加第二个看得见的），标题的文字经 `t()`。以普通标题的：`inbox/modals/create-modal/create-root.tsx:184`（`create-modal/modal.tsx`）、`views/form.tsx:111`（`views/modal.tsx`）、`workspace/views/form.tsx:80`（`workspace/views/modal.tsx`）、`workspace-notifications/sidebar/notification-card/options/snooze/modal.tsx:118`（标题是写死的英文"Customize Snooze Time"）；没有标题的：`inbox/modals/select-duplicate.tsx`（`:90` 的 `h2` 只在搜索为空时显示）、`inbox/modals/snooze-issue-modal.tsx`。M7 的故事打开它们时断言 `expect(modalDialog(page)).toHaveAccessibleName(…)`。
- **正的 `tabIndex`**：收集箱的创建表单（`INTAKE_ISSUE_FORM`）同 M4 的交接第 7 节，其中给父工作项的 `CustomMenu` 的一处（`inbox/modals/create-modal/issue-properties.tsx`）落在菜单的按钮上（M3/P10 的裁定 F-17；M3 收尾 Task 8 的修复轮 1 把它从 Menu 的根移到按钮），它的 `customButton` 是一个 `<button>`，放在菜单的按钮里（按钮里有按钮）。M4、M6、M7 中最后删去正的 `tabIndex` 的那一个删除 `getTabIndex` 和两份 `tab-indices.ts`。
- **嵌套的按钮和看不到的轮廓**（M3/P11 spec 第 5 节、M3 收尾找到）：菜单自己的按钮里又放了 `IconButton` 的三处（`views/quick-actions.tsx`、`workspace-notifications/sidebar/filters/menu/root.tsx`、`workspace-notifications/sidebar/header/options/menu-option/root.tsx`）；收藏的菜单 `workspace/sidebar/favorites/favorites-menu.tsx` 的 `IconButton` 外层 `opacity-0`、没有 `focus-within`，焦点看不到；收藏夹的外层手柄按钮（`workspace/sidebar/favorites/favorite-folder.tsx`）显示时是一个没有名字、按 Enter 什么都不做的停留点。改法照 M3 收尾的项目一行（`CustomMenu` 的 `customButtonClassName`；行的 `focus-within:opacity-100`；手柄不是停留点）。页头的通知链接（`navigation/top-navigation-root.tsx`）没有名字，提示是写死的"Inbox"：随未读数一起给它经 `t()` 的名字。
- **每一行的控件以它那一行为名**（M3 收尾的规则，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.9：一行一个、每行都一样的控件以那一行的东西为名，同 F6 的项目一行、"{name} settings"、"Remove X"）：视图一行的菜单按钮 `views/quick-actions.tsx:94`（"Options"）以视图的名字为名：已有的 `view.name`；M3 收尾已加的带 `{name}` 的键 `common.item_options`（"{name} options"、"{name} 的选项"，项目成员页、标签页一行的菜单读它）。
- **下拉框**：收集箱的两个指派人的成员下拉框（`inbox/content/issue-properties.tsx`、`inbox/modals/create-modal/issue-properties.tsx`）以 `CustomSearchSelect` 为底之后看得到的不同同 M4 的交接第 7 节，在 M7 的页面上核对（C7 没有核对）。右键菜单关着时 `inert`，在 M7 的页面上核对右键打开。共用组件的焦点轮廓是否被 `overflow-hidden` 裁掉，在 M7 的页面上核对。
- **筛选的按钮**（M3 收尾 Task 12，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.9）：`FiltersDropdown` 默认的按钮现在就是弹层的 `Popover.Button`（以 `title` 为名、`aria-expanded`、Escape 之后焦点回到它，指针点开之后焦点留在它上面，同 M4 的交接第 7 节）；M7 的 7 个调用方（5 个用默认的按钮）在收尾 spec 附录 A.4 逐个核对过，没有故事走到，M7 的筛选的故事落地时断言按钮的名字、`aria-expanded` 和 Escape 之后焦点回到它。
- **关闭条件**：M7 的评审写明上面每一项的浏览器核对。

## 4. 地址、复制、文案和其余

- **地址只从路由参数**：收集箱的页面从 router store 读地址的 4 个文件（`app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/intake/page.tsx`、`inbox/content/issue-properties.tsx`、`inbox/sidebar/root.tsx`、`projects/settings/intake/header.tsx`）改成读路由参数，加进关键词规则 `project-address-route` 的范围（M3/P10 spec 第 5 节）。只读工作区的 8 个（`app/(all)/[workspaceSlug]/(projects)/notifications/page.tsx`、`workspace-views/[globalViewId]/page.tsx`、`workspace-views/page.tsx`，`issues/issue-layouts/empty-states/global-view.tsx`、`workspace-notifications/root.tsx`、`workspace/views/quick-action.tsx`，`web/apps/web/core/store/favorite.store.ts`、`global-view.store.ts`）改用 `useRouteWorkspace()` 或路由的 slug（store 的由调用者传入），加进关键词规则 `workspace-address-route`（M3 收尾，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.12）。
- **复制只经 hook**：`inbox/content/inbox-issue-header.tsx`、`views/quick-actions.tsx`、`workspace/views/default-view-quick-action.tsx`、`workspace/views/quick-action.tsx` 改经 `useCopyLink`，失败时经 `t()` 提示；关键词规则 `copy-hook` 中这 4 个例外（`until: "M7/closeout"`）随之删除。默认视图的复制今天没有处理拒绝（M3/P11 spec 附录 A.6）。
- **文案**：收集箱的页头（面包屑的"Intake"）、收藏、视图的页面仍有写死的英文（M3 收尾的扫描），对接时经 `t()`。
- **收集箱的上一个、下一个工作项**：4 个 `IconButton` 的名字是写死的英文（`inbox/content/inbox-issue-header.tsx:315`、`:322`，`inbox/content/inbox-issue-mobile-header.tsx:123`、`:130`："Previous work item""Next work item"）。`IconButton` 的名字自 M3 收尾必传，它们有名字，只是不经 `t()`（[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.9）：随收集箱的页面经 `t()`，en 与 zh-CN 同时加。
- **oxlint**：通知打盹的弹窗（第 3 节）5 条，M7 改它时清零。通知的筛选菜单（`workspace-notifications/sidebar/filters/menu/root.tsx`）1 条（`no-map-spread`）：收尾只为 `IconButton` 的名字改到它（控制者的裁定，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 第 3 节第 2 条），M7 改它时清零。收藏的文件夹（`workspace/sidebar/favorites/favorite-folder.tsx`）2 条（`no-shadow`：`getData` 的参数 `element`、`onDrag` 里的 `instruction` 遮住外层的同名变量）：收尾只为 `CustomMenu` 改到它（菜单打开时设为打开、关上时设为关上，Task 8 的修复轮 1；控制者 2026-10-14），M7 改它时清零。
- **共用的 getter**：成员 store 的 `projectMemberIds` 在 M7 的读者是视图的两处和收集箱的筛选，M4 的是 `issues/issue-layouts/utils.tsx` 的两处；M4、M7 中最后改完读者的那一个删除它（M3/P10 spec 第 3 节，裁定 F-2）。
- **守卫的阶段**：收尾开始时把 `tools/keywords.json` 的 `phase` 改为 `M7/closeout`，`copy-hook` 的 4 个例外（`until: "M7/closeout"`）那时到期，守卫要求它们已经删除（`tools/keywords.mjs` 的说明、M1 设计 7.4）；每个 Phase 开始时改为 `M7/P<k>`。
- **关闭条件**：两条地址规则的范围含这些文件，`copy-hook` 没有 `M7/closeout` 的例外。

来源：[M3 设计](../../M3-workspace-project/M3-design.md) 13.2 的 M7 一行；M3/P7a、P7b、P8a、P8b、P9、P10、P11 的 spec 第 5 节和评审第 6 节中交给 M7 的各条；[M3 收尾 spec](../../M3-workspace-project/specs/closeout.md) 第 2 节。
