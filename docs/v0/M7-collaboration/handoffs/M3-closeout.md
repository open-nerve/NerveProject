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
  - **处理结果（M7/P4，未读通知数）：** 顶栏的未读通知数加回：每个工作区页面取一次，门是路由的工作区（调用者是它的有效成员即取，管理员、成员、访客都有 `notification.read`），键带 `loginId`、工作区 id 与 slug，与通知页的页签共用一个取数（`useUnreadNotificationCount`）；S2 的清单随之改：`e2e/fixtures/mounts.ts` 的 `WORKSPACE` 加 `GET /api/v0/workspaces/{slug}/notifications/unread-count`，读它的 S2 的两个、G3、G8、G13 五个旧故事只经常量跟随（P4 新的 H2–H4、H5 的页面故事也读它）（[P4 spec](../specs/P4-web-notifications.md) §2.4、§2.11、R5）。应用侧栏不加数（Plane 的侧栏收件箱从不渲染，R5）。接口的故事是 P1c 的 H5–H7，页面的是 H7 (page)。未读数一项关闭；本节其余各项照旧待办。
  - **处理结果（M7/P5a，侧栏的待处理数）：** 项目侧栏的收集箱数字加回为工作区级的待处理数：侧栏的项目列表经 `useIntakeCounts()`（`core/hooks/use-intake-counts.ts`，`workspace/sidebar/projects-list.tsx` 调一次）读 `GET /workspaces/{slug}/intake-counts`，门是路由的工作区与至少一个开着收集箱、未归档、调用者在职的项目，键带 `loginId`、工作区 id、slug 与这些项目的 id；导航项只读 store，大于 0 时显示在 “Intake” 旁，链接名带数（`project-navigation.tsx:159`、`:170`）；收集箱 store 不在本地加减，本人的提交、分诊、删除之后重取（[P5a spec](../specs/P5a-web-intake.md) §2.5、R7、R8）。挂载清单随之改：`e2e/fixtures/mounts.ts` 加条件化的 `INTAKE_COUNTS`（只在有这样的项目时读，读它的是 H9 (page)），`WORKSPACE` 不变；新建的项目收集箱是关着的，旧故事不受影响。分诊状态的前端不加回（上面 P2a 的结果）。收集箱的部分关闭；视图、收藏与首页随 P6a、P6b、P7。
- 打开 `intake_view` 时建默认收集箱、分诊状态的接口（M3 设计 3.17）：分诊状态由 `"group" = 'triage'` 识别，每个项目至多一个（`states_project_id_triage_key`），它的名称也占用项目内的状态名（M3/P7a spec 第 3 节第 6 条）；收集箱的分诊状态的前端（store、下拉框、类型、`@nerve/propel` 的两个图标）在 M3/P8b 的 Task 7 删除，随接口加回。
  - **处理结果（M7/P2a）：** 不建默认收集箱（没有 `intakes` 表，收集箱就是项目的开关 `intake_view`）；不开分诊状态的接口（M7 设计 §3.6）：提交放进项目已有的 Triage 状态，缺了是内部错误，不现建；工作项的修改不能以它为目标，状态接口碰不到它（[P2a spec](../specs/P2a-intake-submit-triage.md) §2.3、§2.4）。分诊状态的前端不加回；本节其余各项照旧待办。
- 跨项目的标签列表是各项目列表的并集，没有工作区级的标签（M3 设计 3.16、11.5）；工作区级标签的前端（store 的四个成员、筛选的标签一项、按标签分组）在 M3/P8b 删除，随它加回，连同工作区一级显示的工作项（所有工作项、视图、草稿、通知的预览）上的标签（M3/P8b spec 第 3 节第 21 条）。
- 项目的收藏：`is_favorite`、卡片上的收藏按钮、建项目之后加入收藏、`favorite.store.ts` 的项目分支、`favoriteProjectIds` getter（M3 设计 3.2），和归档项目时的收藏处理。
  - **处理结果（M7/P3c，项目的收藏）：** 项目收藏的接口在 `/api/v0`（`entity_type: project`，`project_id` 即项目自己）；“归档项目时的收藏处理”= 项目归档在原事务里移除这个项目里的全部收藏（项目自己、它的迭代、模块、视图），恢复不加回（[P3c spec](../specs/P3c-favorites.md) §2.6）。资源不带 `is_favorite`，页面由收藏列表算出（设计 §3.8）；卡片上的按钮、建项目之后加入收藏、store 与 getter 随 P7。本节其余各项照旧待办。
- 收集箱查重 `inbox/modals/select-duplicate.tsx` 改用 M4 的工作项搜索（M3 设计 7.3）；工作区视图的筛选列（M3 设计 3.18）。
  - **处理结果（M7/P3b，工作区视图的筛选列）：** 已存工作区视图的筛选存在 `issue_views.filters`（11 个键，含 `project_id`），不给 `workspace_user_properties` 加列；四个默认视图的显示设置照 Plane 存在浏览器（设计 §3.7，[P3b spec](../specs/P3b-views.md) §2.3）。页面随 P6b；本节其余各项照旧待办。
  - **处理结果（M7/P5a，收集箱查重）：** 选择器经 M4 的工作项搜索 `useIssueSearch`（M4/P14 `4c0ed5ea` 接上，`inbox/modals/select-duplicate.tsx:57`，按项目、带搜索词），P5a 让它以 prop 收条目 id、不列条目自己（`bc0eeadf`）；H10 (page) 三种引擎在真实页面上走过。收集箱查重一项关闭；本节其余各项照旧待办。
- **关闭条件**：每一项的接口版本和页面版本的故事；挂载时的取数按权限启用，S2 的清单随之改（M3/P8b 的裁定 D5）。

## 2. stores 和会话（总体设计 7.7）

- 读工作区列表而没有取数的三处：新手引导的 `hasWorkspaces`、个人设置侧边栏的工作区、power-K 在工作区页面之外打开工作区的菜单；直接打开 `/settings/profile/*` 时三处都没有工作区。创建成功之后再取列表被拒绝，被当成"创建失败"，重试时遇到"slug 已占用"，根在 `createWorkspace` 只写进已经取到的列表（M3/P8a 评审第 6 节）。
- 已挂载的工作区包装层换 slug 时不重取列表，在别的标签页加入的工作区要到下一次挂载才显示；要改，改 `useWorkspaceFetch` 这一个判断（M3/P9 spec 第 5 节，P19）。
- `ProjectViewStore.loader` 和 `fetchedMap` 没有写者了：视图的取数设置它们，否则删除它们（M3/P8a 评审的 P30）。通知页的收集箱项经原来的 `useSWR` 读项目（`workspace-notifications/root.tsx`），写项目自己的读，可以盖过包装层的读：照 7.1 改成按权限的取数（M3/P8b 的 PF-L3）。收藏的 `workspace/sidebar/favorites/favorite-items/common/helper.tsx` 的 `console.error` 改成提示（M3/P9 spec 第 5 节）。
  - **处理结果（M7/P4，通知页的收集箱项）：** 通知页的裸 `useSWR` 随旧的 `workspace-notifications/root.tsx` 删除；新的预览只在调用者对选中通知的项目没有角色时经 `useProjectRead` 读项目（会话的取数，键 `["PROJECT", id]`），读在路上显示占位，被拒时清选中并提示；收集箱条目以普通浮层打开（[P4 spec](../specs/P4-web-notifications.md) §2.6、R12）。通知的部分关闭；视图 store 的 `loader`、`fetchedMap` 随 P6a，收藏的 `console.error` 随 P7。
- `@nerve/types` 的 `workspace-notifications.ts` 的 17 行死成员（`domains.mjs` 按路径误归到 M3，M3/P11 spec 附录 A.9）。
  - **处理结果（M7/P4）：** 文件连同 `index.ts` 的导出与 `enums.ts` 的 `ENotificationFilterType` 随 T5 删除，17 行死成员随之消失（T2 的基线与 T15 的终表整份比对，[P4 阶段记录](../reviews/P4-web-notifications-review.md) §5）。本条关闭。
- **界面状态在以 id 为 key 的布局之下**（M3 收尾，负责人 2026-10-13 的 B，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.7）：工作区的内容（`app/(all)/[workspaceSlug]/layout.tsx` 的 `WorkspaceContentWrapper`）以路由的 slug 为 key，项目的页面和项目的设置（两个 `[projectId]/layout.tsx`）以路由的项目为 key，换工作区、换项目（后退、前进也是）时其下重新挂载，组件状态里的弹窗和状态不带过去。store 持有的弹窗开关由渲染它的宿主在卸载时关上（`useLayoutEffect` 的清理，像 `WorkspaceLevelModals`）；宿主在 key 之上时还要在范围结束时关上。收集箱、视图、通知、收藏的弹窗和选中放在这些布局之下，不要放到它们之上（路由的根、`WorkspaceAuthWrapper`、跨工作区的 store 或 `localStorage`）。
  - **处理结果（M7/P4，通知）：** 通知页的页签、筛选与选中在通知布局 `NotificationsLayout` 提供的 context 里（`workspace-notifications/page-state.tsx`），布局在以 slug 为 key 的 `WorkspaceContentWrapper` 之下，离开通知页或换工作区（后退、前进也是）即结束；自定义暂缓弹窗的开关在卡片的动作里，随卡片卸载；页面卸载时清掉浮层（[P4 spec](../specs/P4-web-notifications.md) §2.5、R3；单测 `page-state.test.tsx` 与变异 `t5-mut-state-outlives-page`，H20 (page) 的后退与前进一格）。通知的部分关闭；收集箱、视图、收藏随 P5–P7。
  - **处理结果（M7/P5a，收集箱）：** 收集箱页的筛选与排序在收集箱布局提供的 context 里（`inbox/page-state.tsx` 的 `IntakePageStateProvider`，以页签为 key），布局在以项目为 key 的 `[projectId]/layout.tsx` 之下，离开页面或换项目、换工作区（后退、前进也是）即结束；页签与选中在地址里；分诊的弹窗画在详情区域里（`inbox/content/root.tsx:192` 的 `IntakeModals`），开关在动作模型，详情以 `[slug, 项目, 条目, 权限身份, loginId]` 为 key（`:77`），换选中、换项目即卸下（[P5a spec](../specs/P5a-web-intake.md) §2.6、R5；单测 `inbox/page-state.test.tsx`，H20 (page) ⑧ 的后退、⑨ 的换项目与换工作区）。收集箱的部分关闭；视图、收藏随 P6a–P7。
- **嵌套的 id 没有 key**（M3 收尾）：视图的页面 `views/(detail)/[viewId]`、`workspace-views/[globalViewId]`只在工作区、项目的 key 之下，换一个视图（后退、前进也是）时页面留着，开着的弹窗和组件的状态跟过去。M7 接上这些页面时以页面自己的 id 给它的子树 key，并加一个后退的检查。
- **关闭条件**：上面每一处有 vitest 或故事；这 17 行在 M7 的评审中有结论。

## 3. 键盘、弹窗和下拉框

- **弹窗**：M3 收尾把 `ModalCore` 改成 Headless UI v2 的 `Dialog`（[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.5）：打开时后面的页面 `inert`、`aria-hidden`，请求在路上时关不掉。收集箱、视图、通知的弹窗（通知的打盹 `workspace-notifications/sidebar/notification-card/options/snooze/modal.tsx` 在内）对接时逐个核对 Tab、Escape、点外面、关上之后的焦点和弹窗里经 portal 渲染的东西（收尾 spec 附录 A.3 只证明了 M3 页面上的几种）；`AlertModalCore` 的确认框照 A-M2，按钮的文字经 `t()` 传入。
- **弹窗的名字**：Headless UI v2 的 `Dialog` 只以它的 `DialogTitle` 为名（`aria-labelledby`），普通的标题不给弹窗名字，读屏读出一个没有名字的对话框；M3 收尾的修复轮在 `@nerve/ui` 的 `ModalCore` 旁边导出 `ModalTitle`（它就是 `DialogTitle`，`h3`），`AlertModalCore` 的标题经它，M2、M3 的弹窗的标题都经它（[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.5）。M7 的确认框（`AlertModalCore`）因此都有名字；别的弹窗的标题改经 `ModalTitle`，没有看得见的标题的给一个 `sr-only` 的（不加第二个看得见的），标题的文字经 `t()`。以普通标题的：`inbox/modals/create-modal/create-root.tsx:184`（`create-modal/modal.tsx`）、`views/form.tsx:111`（`views/modal.tsx`）、`workspace/views/form.tsx:80`（`workspace/views/modal.tsx`）、`workspace-notifications/sidebar/notification-card/options/snooze/modal.tsx:118`（标题是写死的英文"Customize Snooze Time"）；没有标题的：`inbox/modals/select-duplicate.tsx`（`:90` 的 `h2` 只在搜索为空时显示）、`inbox/modals/snooze-issue-modal.tsx`。M7 的故事打开它们时断言 `expect(modalDialog(page)).toHaveAccessibleName(…)`。共用的自定义日期的弹窗（`core/filters/date-filter-modal.tsx`）M3 收尾已在根上改，以调用方给的 `title` 为名；M7 的两个调用方（收集箱的 `inbox/inbox-filter/filters/date.tsx`，视图的筛选经 `common/filters/created-at.tsx`）传的是写死的英文"Created date"，改经 `t()`。收集箱的 `FilterDate`（`date.tsx:66`）是创建日期和最后更新日期两个筛选共用的（`filter-selection.tsx:85,89`），所以它把自己的 `label` 经 `t()` 传给弹窗（调用方传来的两个 `label` 也是写死的英文），不传写死的"Created date"：今天"Last updated date"筛选的弹窗也以"Created date"为名，下拉框读"Created date range"。
  - **处理结果（M7/P4，通知的暂缓弹窗，上面两条）：** 旧的打盹弹窗随 T5 删除；新的自定义暂缓弹窗（`workspace-notifications/card/snooze-modal.tsx`）是 `ModalCore` 带 `handleClose`，标题经 `ModalTitle`（“Customize snooze time” / “自定义暂缓时间”，经 `t()`）。H20 (page) 在三种引擎里核对：弹窗以标题为名（`toHaveAccessibleName`），焦点在里面，后面的页面隐藏；Escape 与“取消”关上之后焦点回到这一行的暂缓按钮；在途时 Escape、点外面与“取消”都关不掉；经 portal 渲染的日期日历与时间列表可用（[P4 spec](../specs/P4-web-notifications.md) §2.7）。暂缓弹窗的部分关闭；收集箱、视图的弹窗随 P5、P6a/P6b。
  - **处理结果（M7/P5a，收集箱的四个分诊弹窗，上面两条）：** 拒绝、删除（`AlertModalCore`，标题经 `ModalTitle`）、暂缓与标重复的选择器（`ModalCore` 带 `handleClose`，标题经 `ModalTitle`：`inbox/modals/snooze-issue-modal.tsx:82`、`select-duplicate.tsx:98`），文字都经 `t()`。H20 (page) 三种引擎核对（`h20-intake-pages.spec.ts`、`h20-intake-keys-pages.spec.ts`）：弹窗以标题为名，焦点在里面，后面的页面隐藏；键盘在弹窗里以 Tab 移到按钮、Enter 或 Space 按下（H20 (page) ⑦）；Escape 与取消关上之后焦点回打开它的按钮（窄屏是窄页头的菜单按钮）；请求在途时 Escape、点外面都关不掉、取消停用；做成之后焦点照落点表（[P5a spec](../specs/P5a-web-intake.md) §2.8、R35）。这四个关闭；新建弹窗（`inbox/modals/create-modal/modal.tsx`）随 P5b，视图的弹窗随 P6a/P6b，保持 open。
- **正的 `tabIndex`**：收集箱的创建表单（`INTAKE_ISSUE_FORM`）同 M4 的交接第 7 节，其中给父工作项的 `CustomMenu` 的一处（`inbox/modals/create-modal/issue-properties.tsx`）落在菜单的按钮上（M3/P10 的裁定 F-17；M3 收尾 Task 8 的修复轮 1 把它从 Menu 的根移到按钮），它的 `customButton` 是一个 `<button>`，放在菜单的按钮里（按钮里有按钮）。M4、M6、M7 中最后删去正的 `tabIndex` 的那一个删除 `getTabIndex` 和两份 `tab-indices.ts`。
- **嵌套的按钮和看不到的轮廓**（M3/P11 spec 第 5 节、M3 收尾找到）：菜单自己的按钮里又放了 `IconButton` 的三处（`views/quick-actions.tsx`、`workspace-notifications/sidebar/filters/menu/root.tsx`、`workspace-notifications/sidebar/header/options/menu-option/root.tsx`）；收藏的菜单 `workspace/sidebar/favorites/favorites-menu.tsx` 的 `IconButton` 外层 `opacity-0`、没有 `focus-within`，焦点看不到；收藏夹的外层手柄按钮（`workspace/sidebar/favorites/favorite-folder.tsx`）显示时是一个没有名字、按 Enter 什么都不做的停留点。改法照 M3 收尾的项目一行（`CustomMenu` 的 `customButtonClassName`；行的 `focus-within:opacity-100`；手柄不是停留点）。页头的通知链接（`navigation/top-navigation-root.tsx`）没有名字，提示是写死的"Inbox"：随未读数一起给它经 `t()` 的名字。
  - **处理结果（M7/P4，通知的三处）：** 两个通知菜单（类型筛选、视图）改为 `PopoverMenu`，按钮里是图标与 `sr-only` 的名字，不再嵌套 `IconButton`，项是 `<button aria-pressed>`（`workspace-notifications/sidebar/{type-filter-menu,view-menu}.tsx`，旧的两个文件随 T5 删除；H20 (page) 三种引擎核对 Tab、Enter、Space 与 Escape）；页头的通知链接以 `notification.label` 为名，有未读时带数（“Inbox, 5 unread”），`Tooltip` 的文字改经 `t()`（[P4 spec](../specs/P4-web-notifications.md) §2.6、R5、R16）。这个 `Tooltip` 在 Plane 里打不开（`AppSidebarItem` 不转发 ref 与事件），P4 修复轮让 `AppSidebarItem` 的链接收下触发器给的 ref 与事件，悬停与聚焦时打开（真的 Propel Tooltip 的单测 `core/components/sidebar/sidebar-item.test.tsx`）；名字仍靠 `aria-label`，见 [P4 阶段记录](../reviews/P4-web-notifications-review.md) §7、§9。通知的三处关闭；视图的 `views/quick-actions.tsx` 随 P6a，收藏的两处随 P7。
- **每一行的控件以它那一行为名**（M3 收尾的规则，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.9：一行一个、每行都一样的控件以那一行的东西为名，同 F6 的项目一行、"{name} settings"、"Remove X"）：视图一行的菜单按钮 `views/quick-actions.tsx:94`（"Options"）以视图的名字为名：已有的 `view.name`；M3 收尾已加的带 `{name}` 的键 `common.item_options`（"{name} options"、"{name} 的选项"，项目成员页、标签页一行的菜单读它）。
- **下拉框**：收集箱的两个指派人的成员下拉框（`inbox/content/issue-properties.tsx`、`inbox/modals/create-modal/issue-properties.tsx`）以 `CustomSearchSelect` 为底之后看得到的不同同 M4 的交接第 7 节，在 M7 的页面上核对（C7 没有核对）。右键菜单关着时 `inert`，在 M7 的页面上核对右键打开。共用组件的焦点轮廓是否被 `overflow-hidden` 裁掉，在 M7 的页面上核对。
  - **处理结果（M7/P5a）：** 详情的负责人下拉框（`inbox/content/issue-properties.tsx` 的 `MemberDropdown`）只有单测核对：键盘从按钮打开，读项目的成员，搜索框取得焦点（`issue-properties.test.tsx:194`）；页面故事没有打开它（H11 (page) 只断言访客与已处理时 “Add assignees” 停用，`h11-intake-content-page.spec.ts:53`、`:314`），所以 spec 表中“详情的一个关闭”不成立，本条不关闭。新建弹窗的那一个随 P5b；右键菜单与焦点轮廓 P5a 没有涉及。保持 open。
- **筛选的按钮**（M3 收尾 Task 12，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.9）：`FiltersDropdown` 默认的按钮现在就是弹层的 `Popover.Button`（以 `title` 为名、`aria-expanded`、Escape 之后焦点回到它，指针点开之后焦点留在它上面，同 M4 的交接第 7 节）；M7 的 7 个调用方（5 个用默认的按钮）在收尾 spec 附录 A.4 逐个核对过，没有故事走到，M7 的筛选的故事落地时断言按钮的名字、`aria-expanded` 和 Escape 之后焦点回到它。
  - **处理结果（M7/P4）：** 本段不涉及：通知页的两个菜单用 `PopoverMenu`，不用 `FiltersDropdown`（`git grep FiltersDropdown -- web/apps/web/core/components/workspace-notifications` 为空，re-93 m17）。M7 的 7 个调用方随后续页面阶段的筛选故事，保持 open。
  - **处理结果（M7/P5a，收集箱的筛选）：** 收集箱页的筛选按钮是 `FiltersDropdown` 默认的按钮（`inbox/filters/menu.tsx`）。三种引擎核对：H20 (page) ⑤ 断言按钮的名字（“Filters”）、`aria-expanded`、Escape 关上之后焦点回到它（`h20-intake-keys-pages.spec.ts:106-131`）；H9 (page) ③ 每次打开都断言 `aria-expanded` 与浮层在按钮旁（`expectPopupBesideButton`，1280 与 800 两种宽度，`e2e/fixtures/intake-pages.ts:223-230`；修复 `b1aeccb1` 之前浮层在视口之外，[P5a 阶段记录](../reviews/P5a-web-intake-review.md) §3）。收集箱这一个调用方关闭；其余调用方随视图（P6a/P6b），保持 open。
- **字段的提示、弹窗里的列表、列表里的控件**（M3 收尾的浏览器核对之后的修复，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.17 第 2、3、5 条；写法同 M4 的交接第 7 节）。
  - 视图、工作区视图的表单的名字的提示（`views/form.tsx:186`、`workspace/views/form.tsx:110`）照 M2、M3 的表单：提示是字段的 propel `FieldError`，控件以它为描述、`aria-invalid`，被拒绝的字段在提示显示之后取得焦点（`core/lib/refused-focus.ts` 的 `useRefusedFocus`，`useForm` 写 `shouldFocusError: false`；写法和时机见 M4 交接第 7 节）。通知的稍后提醒的两条规则（`snooze/modal.tsx:133`、`:156`）带写死的英文的提示，却不显示；收藏夹的新建（`workspace/sidebar/favorites/new-fav-folder.tsx`）的规则没有提示：没过规则时读屏什么也读不到，照同一个写法给它们经 `t()` 的提示。
  - 收集箱的查重（`inbox/modals/select-duplicate.tsx`）的列表本身就是弹窗的内容，核对它打开时后面的页面隐藏着；在弹窗里打开的别的列表读 `useInsideModal()`。
  - 稍后提醒的时间的 `CustomSelect` 列表里，AM、PM 是两个可点的 `div`（`snooze/modal.tsx:176`、`:187`），不是选项，方向键到不了：改为选项，或移到列表之外。
  - **处理结果（M7/P4，通知的暂缓弹窗）：** 上面第一条里通知的两条规则与这一条：自定义暂缓弹窗的两个字段必填，提示经 `t()`（`common.errors.required`）以 `FieldError` 显示在字段下，控件以它为描述并带 `aria-invalid`，第一个被拒的字段取得焦点（`useForm({ shouldFocusError: false })` 与 `useRefusedFocus`），nerve 的 `must_be_future` 显示在时间一栏下；AM/PM 两个 `div` 不再有：时段的值是 24 小时制、按页面的语言标注，列表里只有选项，方向键可达（[P4 spec](../specs/P4-web-notifications.md) §2.7、R15；H20 (page) 三种引擎核对必填的提示与描述、时间选择的键盘）。通知的这两处关闭；视图与收藏夹的表单、收集箱的查重随 P5–P7。
  - **处理结果（M7/P5a，收集箱的查重，上面第二条）：** 选择器的列表就是弹窗的内容（`inbox/modals/select-duplicate.tsx`，以 prop 收条目 id、不列条目自己）；H10 (page) 与 H20 (page) ① 三种引擎核对它打开时后面的页面隐藏、焦点在里面，键盘、指针与窄页头的菜单三条路都选得到（[P5a spec](../specs/P5a-web-intake.md) §2.8）。这一条关闭。
- **收集箱新建的标题的长度提示**（M3 收尾，浏览器核对之后）：`inbox/modals/create-modal/issue-title.tsx:48` 超过 255 个字时显示一个红字的 `span`，输入框（`required`）不在 propel 的 `Field` 里，没有以它为描述，也不 `aria-invalid`（[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.17 第 2 条的第三次扫描找到）。它是收集箱的表单，照上面视图的表单的写法改：提示是 `Field` 里的 `FieldError`，控件恰在提示显示时 `aria-invalid`，被拒绝时取得焦点（`useRefusedFocus`）。
- **收集箱新建的弹窗自己的 Escape 监听**（M3 收尾，浏览器核对之后）：新建的弹窗（`inbox/modals/create-modal/modal.tsx:37` 的 `ModalCore`，没有 `handleClose`）以 `useKeypress("Escape")`（`:29`，监听整个 `document`）关上自己。它里面打开选父工作项（`create-modal/issue-properties.tsx:192` 的 `ParentIssuesListModal`，有 `handleClose`）时，最里层的 `ModalCore` 认领 Escape、选父工作项关上，但同一个 Escape 也到这个监听，把新建的弹窗一起关上（[收尾评审](../../M3-workspace-project/reviews/closeout-review.md)第 4 节，最后一轮的窄复审列出的嵌套的调用方第 6 行）。这是 M7 的代码，与 Plane 相同；修法同 M4 的交接第 7 节描述编辑器的一条，在这个监听里，由 M7 决定（例如不理会目标不在它自己的弹窗里的 Escape），加一个故事核对。
- **关闭条件**：M7 的评审写明上面每一项的浏览器核对。

## 4. 地址、复制、文案和其余

- **地址只从路由参数**：收集箱的页面从 router store 读地址的 4 个文件（`app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/intake/page.tsx`、`inbox/content/issue-properties.tsx`、`inbox/sidebar/root.tsx`、`projects/settings/intake/header.tsx`）改成读路由参数，加进关键词规则 `project-address-route` 的范围（M3/P10 spec 第 5 节）。只读工作区的 8 个（`app/(all)/[workspaceSlug]/(projects)/notifications/page.tsx`、`workspace-views/[globalViewId]/page.tsx`、`workspace-views/page.tsx`，`issues/issue-layouts/empty-states/global-view.tsx`、`workspace-notifications/root.tsx`、`workspace/views/quick-action.tsx`，`web/apps/web/core/store/favorite.store.ts`、`global-view.store.ts`）改用 `useRouteWorkspace()` 或路由的 slug（store 的由调用者传入），加进关键词规则 `workspace-address-route`（M3 收尾，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.12）。
  - **处理结果（M7/P4，通知的两个文件）：** `notifications/page.tsx`（页面标题）与 `workspace-notifications/root.tsx`（预览）改读 `useRouteWorkspace()`，连同 P4 新写的读工作区的文件（顶栏、`use-unread-notification-count.ts`、`use-notification-list.ts`、卡片、卡片的动作、暂缓弹窗、页头）都在 `workspace-address-route` 的范围里，各有命中的样例（[P4 spec](../specs/P4-web-notifications.md) §2.10；关键词变异 `t5-mut-title-router-keywords` 等）。通知的两个关闭；其余六个随 P6b、P7，保持 open。
  - **处理结果（M7/P5a，收集箱的 4 个文件）：** 这 4 个文件都已改写或移走：`inbox/content/issue-properties.tsx`、`inbox/sidebar/root.tsx` 与路由页 `intake/page.tsx` 重写，`projects/settings/intake/header.tsx` 移为 `inbox/page-header.tsx`；新文件从路由参数取地址（路由页的 `params`、页头的 `useParams`，往下经 prop 传；列表的读经 `useRouteWorkspace`），收入 `project-address-route` 与 `workspace-address-route` 的范围（T6a、T6b、T7、T8、T9；写的接缝的三个文件在 T5、T5c 收入）（[P5a spec](../specs/P5a-web-intake.md) §2.11）。收集箱的 4 个关闭；只读工作区的其余随 P6b、P7，保持 open。
- **复制只经 hook**：`inbox/content/inbox-issue-header.tsx`、`views/quick-actions.tsx`、`workspace/views/default-view-quick-action.tsx`、`workspace/views/quick-action.tsx` 改经 `useCopyLink`，失败时经 `t()` 提示；关键词规则 `copy-hook` 中这 4 个例外（`until: "M7/closeout"`）随之删除。默认视图的复制今天没有处理拒绝（M3/P11 spec 附录 A.6）。
  - **处理结果（M7/P5a，收集箱的页头）：** 旧的 `inbox/content/inbox-issue-header.tsx` 在 T6a 删除；新的动作模型的“复制链接”经 `useCopyLink`（`inbox/content/header/use-intake-actions.ts:157`），失败时经 `t()` 提示；关键词 `copy-hook` 的这个例外同一提交删除（[P5a spec](../specs/P5a-web-intake.md) §2.7）。收集箱的一处关闭；视图的三处随 P6a/P6b，保持 open。
- **文案**：收集箱的页头（面包屑的"Intake"）、收藏、视图的页面仍有写死的英文（M3 收尾的扫描），对接时经 `t()`。
- **收集箱的上一个、下一个工作项**：4 个 `IconButton` 的名字是写死的英文（`inbox/content/inbox-issue-header.tsx:315`、`:322`，`inbox/content/inbox-issue-mobile-header.tsx:123`、`:130`："Previous work item""Next work item"）。`IconButton` 的名字自 M3 收尾必传，它们有名字，只是不经 `t()`（[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.9）：随收集箱的页面经 `t()`，en 与 zh-CN 同时加。
  - **处理结果（M7/P5a，上面两条的收集箱部分）：** 面包屑的 “Intake” 经 `t("sidebar.intake")`（`inbox/page-header.tsx:59`）；上一条、下一条的两个 `IconButton`（桌面与窄页头共用 `inbox/content/header/navigation.tsx`）以 `inbox_issue.a11y.previous`、`inbox_issue.a11y.next` 为名，en 与 zh-CN 同加（[P5a spec](../specs/P5a-web-intake.md) §2.7）。收集箱的部分关闭；收藏、视图的文案随 P6a–P7。
- **oxlint**：通知打盹的弹窗（第 3 节）5 条，M7 改它时清零。通知的筛选菜单（`workspace-notifications/sidebar/filters/menu/root.tsx`）1 条（`no-map-spread`）：收尾只为 `IconButton` 的名字改到它（控制者的裁定，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 第 3 节第 2 条），M7 改它时清零。收藏的文件夹（`workspace/sidebar/favorites/favorite-folder.tsx`）2 条（`no-shadow`：`getData` 的参数 `element`、`onDrag` 里的 `instruction` 遮住外层的同名变量）：收尾只为 `CustomMenu` 改到它（菜单打开时设为打开、关上时设为关上，Task 8 的修复轮 1；控制者 2026-10-14），M7 改它时清零。
  - **处理结果（M7/P4，通知的两条）：** 打盹弹窗的 5 条随旧文件在 T5 删除（新的暂缓弹窗 0 条）；筛选菜单的那条 `no-map-spread` 已由 M4/P16 的 `c4881cf5` 清掉（`...filter` 改为 `value: filter.value`），文件随 T5 删除；web 的上限 40 → 31（[P4 spec](../specs/P4-web-notifications.md) §2.10）。通知的两条关闭；收藏夹的 2 条随 P7，保持 open。
- **共用的 getter**：成员 store 的 `projectMemberIds` 在 M7 的读者是视图的两处和收集箱的筛选，M4 的是 `issues/issue-layouts/utils.tsx` 的两处；M4、M7 中最后改完读者的那一个删除它（M3/P10 spec 第 3 节，裁定 F-2）。
  - **处理结果（M7/P5a）：** 收集箱的筛选这个读者随旧筛选树在 T6b 删除，新的筛选读显式的 `getProjectMemberIds(projectId, true)`（`inbox/filters/menu.tsx:106`）；M4 的两处已在 M4/P16 迁走（§7）。现余两个读者都是视图的，getter 随 P6a 删除，保持 open。
- **守卫的阶段**：收尾开始时把 `tools/keywords.json` 的 `phase` 改为 `M7/closeout`，`copy-hook` 的 4 个例外（`until: "M7/closeout"`）那时到期，守卫要求它们已经删除（`tools/keywords.mjs` 的说明、M1 设计 7.4）；每个 Phase 开始时改为 `M7/P<k>`。
  - **处理结果（M7/P4）：** T1 把 `phase` 改为 `M7/P4`（`8f718622`），`node tools/keywords.mjs` 没有到期的例外。保持 open 至收尾。
- **关闭条件**：两条地址规则的范围含这些文件，`copy-hook` 没有 `M7/closeout` 的例外。

来源：[M3 设计](../../M3-workspace-project/M3-design.md) 13.2 的 M7 一行；M3/P7a、P7b、P8a、P8b、P9、P10、P11 的 spec 第 5 节和评审第 6 节中交给 M7 的各条；[M3 收尾 spec](../../M3-workspace-project/specs/closeout.md) 第 2 节。

## 5. M4/P14 共用编辑器与收集箱的接续边界（2026-10-31）

P14 的 rich/lite 入口改为 lazy，公开 props/ref 保持；收集箱查重已适配原生搜索。收集箱 create-modal/issue-properties 的 isVisible 默认 false、唯一真实调用者不传，开始日期与父项分支没有可达入口，P14 整审已删除这些死分支及专属状态/导入；原优先级、成员、标签、截止日期仍在。§3 的旧父项嵌套按钮行是历史输入，M7 若重新设计可用父项入口，仍须按自己的交互合同实现。

明确保留的未完成项：`inbox/modals/create-modal/create-root.tsx` 的 Discard 在 editor 尚未挂载时仍按现有 readiness 返回提示，外层 `create-modal/modal.tsx` 的 document Escape 可退出。慢加载会延长按钮不可用时间，不能将它说成已经沿 M4 创建窗统一关闭，也不能将共享 ref 兼容视为 M7 自身业务验收。M7 应结合原有 Escape/dirty/保存草稿规则统一关闭策略，并加真实 lazy pending、busy、内层 Escape、输入保留回归。按已批准 P10a 裁定 A，M7 自身页面和接口仍由 M7 验收，本期未修改其协议/关闭政策。

证据：[P14 整审](../../../reviews/re-21-M4P14整体审核.md)、[P14 review §13](../../M4-issue-core/reviews/P14-web-issue-detail-review.md#13-交接实际决定与保留边界)。本 handoff 顶层保持 open。

## 6. M4/P16 焦点索引最后读者回执（2026-11-01）

M4 原创建表单的 `ISSUE_FORM` 已无读者，本期删除其枚举、独占数组和映射项；其余四组及 `getTabIndex`、两份 barrel 保留。当前七个调用文件：M6 的 `cycles/form.tsx`、`modules/form.tsx`，M7 的 `views/form.tsx` 与 `inbox/modals/create-modal/{create-root,issue-properties,issue-title,issue-description}.tsx`。视图表单虽由 M7 实现，M6 修改共用项目弹窗宿主时仍按本交接既有分工处理。

全 web 44 处直接 `CustomMenu` JSX 无 `tabIndex` 属性/展开属性；M4 的 `issues/issue-layouts/spreadsheet/columns/header-column.tsx` 仍传 `customButtonTabIndex={-1}`，用于表格列头控件，不是正索引残留。P14 已删除收集箱父项死入口，§3 所记该 `CustomMenu` 正索引与嵌套按钮现属历史；不能据此删除整个菜单焦点接口。`tabIndex ?? customButtonTabIndex` 本期保持，后续按真实最后读者再收口。

最后删除条件：M6/M7 迁移上述表单与两处归档弹窗 `tabIndex={1}` 到自然 Tab 顺序，并完成各自弹窗/portal/回焦回归；全仓确认 `getTabIndex`、枚举/映射及各共享控件的属性读者都迁完后，最后迁移者删除两份 `tab-indices.ts` 和 barrel。共享菜单的非正索引用途须独立核实保留，不以 M4 零正索引等同全部焦点属性可删。未改未来领域键盘顺序、接口或顶部 open 状态。

证据：[P16 review §9](../../M4-issue-core/reviews/P16-web-cleanup-review.md#9-t1-焦点索引与实施入口)、本期 `t1-focus-inventory.json`（逐调用、直接属性、M01 静态输入反例）。

## 7. M4/P16 成员 getter 最后读者回执（2026-11-01）

§4的M4历史两读者已迁移；`issues/issue-layouts/utils.tsx`不再读隐式getter。当前M4子项工具栏与工作项命令的同名局部数组来自 `getProjectMemberIds(projectId, false)`，不能因为局部变量同名判为未迁移。TypeScript checker沿属性声明确认生产读者只有以下三文件，全部归M7：

| 实际文件（web/apps/web/ 下） | 用法 |
|---|---|
| app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/views/(list)/mobile-header.tsx | 解构getter，交ViewFiltersSelection.memberIds |
| core/components/views/view-list-header.tsx | 解构getter，交桌面ViewFiltersSelection.memberIds |
| core/components/inbox/inbox-filter/filters/filter-selection.tsx | 解构getter，共给负责人/创建人两个FilterMember |

M7最后迁移这三处后，重新核全仓读者，再一起删除 `IProjectMemberStore.projectMemberIds`、makeObservable的computed注册、类getter及其专属测试断言。显式参数的getProjectMemberIds/getFilteredProjectMemberIds和其它成员权限/排序测试保留；不得只删类getter留半条合同。本期保留三消费者与getter，未提前接通M7页面或改变目录权限。

证据：[P16 review §16](../../M4-issue-core/reviews/P16-web-cleanup-review.md#16-t7-成员getter最后读者)、t7-member-readers-restored（3生产/2测试读取、0未分类/0M4），M17恢复一个M4隐式读取准确失败、同名显式参数正例通过，成员store21测试与web typegen/tsc通过。顶部status继续open。

**处理结果（M7/P5a）：** 表中第三行 `inbox/inbox-filter/filters/filter-selection.tsx` 随旧筛选树在 T6b 删除（`2ac1928c`），新的负责人与创建者筛选读 `getProjectMemberIds(projectId, true)`（`inbox/filters/menu.tsx:106`，T7）。重新核过：`git grep projectMemberIds -- web/apps/web` 在产品代码里解构 getter 的只剩表中前两行（视图）与 store 本身，`issue-detail-widgets/sub-issues/title-actions.tsx` 是同名的局部变量。getter、computed 注册与专属测试的删除随 P6a（[P5a spec](../specs/P5a-web-intake.md) R23）。保持 open。
