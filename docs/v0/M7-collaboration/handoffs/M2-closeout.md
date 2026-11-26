---
status: open
from: M2/closeout
to: M7
created: 2026-09-28
---

# M2 交给 M7：保存视图的游标、stores 按会话分代、收集箱和视图的下拉框、删除关系图

下面这些来自 [M2 设计](../../M2-auth/M2-design.md) 13.2 中接收者含 M7 的各行（多个 M 的行只取 M7 的部分）。另有收尾找到的三项：第 2 节中最近访问的接口放在哪个文件（M2 设计 3.12），第 5 节的删除关系图（M2 设计 3.14 把它交给 M3、M4、M6、M7，13.2 那一行漏了 M7），第 6 节的 CSP（M2/P4 评审第 4 节）。

## 1. 保存视图的列表：每种排序一个游标载荷

- **做什么**：保存视图的列表有多种排序，照 M2 设计 3.12 为每种排序定义游标载荷，最后以 `id` 保证稳定；不复用 PAT 列表的 `(created_at, id)`。跨模块的联表同 M6：事先列为 `TestSQLCSchemaScope` 的例外并写明理由，或者用端口拆开（M2 设计 3.14）。
- **代码在哪**：游标的封套在 `server/internal/shared/cursor.go`（`EncodeCursor`、`DecodeCursor`，总体设计 3.4）。
- **游标不签名**：改成另一个合格的位置照样可用。载荷只决定从哪里接着读；游标内容会影响能看到什么的列表（视图会带筛选条件），要自己另加检查（M2/P3a 评审第 7 节）。
- **关闭条件**：每种排序有自己的载荷和测试（翻页不重复、不遗漏）；改过的游标只能在调用者看得到的数据里换起点，有测试；跨模块的联表有结论。

**处理结果（M7/P3a，工作区跨项目列表的部分）：** 工作区跨项目列表 `GET /workspaces/{slug}/issues` 用列表引擎的共享游标 `[order, key, id]`（与项目、成员列表相同，每种次序的键由比较器定），改过的游标只换起点、换不出看不到的行（组合测试 `TestWorkspaceIssueList` 的格，[P3a spec](../specs/P3a-workspace-list.md) §2.2）。视图的列表是集合型、不分页（设计 §3.7），本条的视图部分与跨模块联表的结论随 P3b。保持 open。

**处理结果（M7/P3b，视图的部分）：** 两种视图的列表都是集合型、不分页，按 `created_at` 倒序、同值按 `id`，没有游标载荷（“每种排序一个载荷”不适用，设计 §3.7）。跨模块的联表：navigation 的 SQL 不联表 `projects`、`workspaces`，项目的事实（工作区、`issue_views_view`）、项目与工作区的 `FOR SHARE` 都经端口（project 的 `navigation_ports.sql`、workspace 的 `WorkspaceDirectory`），`TestSQLCSchemaScope` 不加例外（[P3b spec](../specs/P3b-views.md) §2.1）。本条剩下收藏与最近访问的列表（P3c）。保持 open。

**处理结果（M7/P3c，收藏的部分）：** 收藏的列表是集合型、不分页，按 `sequence DESC, id DESC`；目标的名字、图标与是否看得到经端口取（project 的 `NavigationTargets`、planning 的 `NavigationCycles`/`NavigationModules`、access 的 `Readable`），navigation 的 SQL 不联表别的模块的表（[P3c spec](../specs/P3c-favorites.md) §2.4、§2.5）。最近访问的部分随 P3d，保持 open。

**处理结果（M7/P3d，最近访问的部分）：** 最近访问的列表是集合型、不分页，至多 20 条，按 `visited_at DESC, id DESC`，没有游标载荷；目标的摘要与是否看得到经端口取（issue 的 `NavigationIssues`、project 的 `NavigationTargets`、access 的 `Readable`），navigation 的 SQL 不联表别的模块的表，`TestSQLCSchemaScope` 不加例外（[P3d spec](../specs/P3d-recents.md) §2.4、§2.5）。M7 的列表至此都有了结论；照设计 §11 保持 open 至收尾关闭。

## 2. stores 按会话分代（规则在总体设计 7.7）

- **做什么**：M7 接上新接口的 stores、services 和页面照 [总体设计 7.7](../../v0-design.md) 写。M7 的范围（`d97c513` 上数出）：`git grep -n -E '^(export )?const [A-Za-z]+ = new [A-Za-z]+Service\(' -- web/apps/web` 中的 `WorkspaceNotificationService`、`IntakeWorkItemVersionService` 各 1 处，以及最近访问的 `WorkspaceService.fetchWorkspaceRecents` 1 处（`web/apps/web/core/components/home/widgets/recents/index.tsx`）。
- **最近访问的接口**：`/me/recent-visits` 这样的路径属于最近访问自己的模块文件，不放进 `api/modules/identity.yaml`（M2 设计 3.12 的组织规则：一个路径只属于一个模块文件）。
- **关闭条件**：这 3 处都已消失；M7 新加的 SWR 键带 `loginId`；最近访问的操作在它自己的模块文件里。

**处理结果（M7/P3d，最近访问的部分）：** `recordRecentVisit`（`/me/recent-visits`）与 `listRecentVisits` 都在 `api/modules/navigation.yaml`、标签 `navigation`，不在 `identity.yaml`（[P3d spec](../specs/P3d-recents.md) §2.4）。`WorkspaceService.fetchWorkspaceRecents` 已随 M3/P8a 的 `5c128973` 消失（`git grep 'fetchWorkspaceRecents' -- web` 为空）；另两处 `WorkspaceNotificationService`、`IntakeWorkItemVersionService` 仍在，随 P4、P5；最近访问新的 SWR 键随 P7。保持 open。

**处理结果（M7/P4，通知的部分）：** 模块级单例 `WorkspaceNotificationService`（`core/services/workspace-notification.service.ts`）随旧数据层在 T5 删除；新的 `NotificationService` 每一代以自己的客户端构造（`RootStore.notification`），取数只经 `useSessionSWR`，两个新键 `NOTIFICATION_UNREAD_COUNT`、`NOTIFICATIONS` 都带 `loginId`（以及工作区 id 与 slug）（[P4 spec](../specs/P4-web-notifications.md) §2.1、§2.3、§2.4）。上面那条 `git grep` 今天在 `web/apps/web` 只剩 `IntakeWorkItemVersionService`（`inbox/content/issue-root.tsx:36`，随 P5）与 M7 之外的 `TimezoneService`。保持 open。

**处理结果（M7/P5a，收集箱的部分）：** 模块级单例 `IntakeWorkItemVersionService`（`inbox/content/issue-root.tsx:36`）随旧详情树在 T6a 删除，旧 store 里的 `InboxIssueService` 实例（`store/inbox/project-inbox.store.ts:127`）随旧数据层在 T6c 删除；新的 `IntakeService` 每一代以自己的客户端构造（issue 根 store 的 `intakeApi`），取数只经 `useSessionSWR`，新键 `INTAKE_COUNTS`、`INTAKE_ISSUES` 与详情的读都带 `loginId`（[P5a spec](../specs/P5a-web-intake.md) §2.1、§2.3）。上面那条 `git grep` 今天在 `web/apps/web` 只剩 M7 之外的 `TimezoneService`（`core/hooks/use-timezone.tsx:12`），M7 的 3 处都已消失；最近访问新的 SWR 键随 P7，保持 open。

## 3. 下拉框（M7 的部分）

路径在 `web/apps/web/core/components/` 下，另写明的除外。根因和修法见 M2/P5 spec 2.3。

- **12 个同样写法的下拉框中 M7 的 2 个**：`dropdowns/intake-state/base.tsx`（收集箱），`web/packages/ui/src/dropdown/single-select.tsx`（`Dropdown`，经 `dropdowns/layout.tsx` 用在 `views/form.tsx`）。react-popper 的 ref 放在 `Combobox.Options` 唯一的子元素上，列表会停在页面左上角；popper 的 ref 改放在列表元素本身。打开状态只要 Headless UI 的一份：没有搜索框的选择用 `Listbox`，有搜索框的用 `Combobox`，列表打开时让输入框取得焦点。
- **P5 重建的 `CustomSelect`**（`Listbox`：按钮上的 Enter 提交所在的表单，输入字母跳到对应的选项，列表开着时页面其余部分 inert）在 M7 的调用方：`workspace-notifications/sidebar/notification-card/options/snooze/modal.tsx`。
- **`BreadcrumbNavigationSearchDropdown`**（`web/packages/ui/src/breadcrumbs/navigation-search-dropdown.tsx`，`onOpen` 每次打开调一次）在视图的页头上。
- **关闭条件**：M7 的 review 逐项写明：两个下拉框的列表在按钮旁展开；列出的调用方在浏览器中核对过。

**处理结果（M7/P4，`CustomSelect` 在暂缓弹窗里）：** 旧的 `snooze/modal.tsx` 随 T5 删除，M7 的这个调用方换成自定义暂缓弹窗的时间选择（`workspace-notifications/card/snooze-modal.tsx`，[P4 spec](../specs/P4-web-notifications.md) §2.7）。浏览器核对在 H20 (page)（`e2e/stories/collaboration/h20-notification-pages.spec.ts`，chromium、firefox、webkit）：按钮上 Enter 打开列表，列表在按钮旁（`expectListBesideButton`），方向键与 Enter 选中，列表关上、焦点回到按钮，按钮的名字是 “Time, 12:30 AM”；弹窗开着时后面的页面隐藏。收集箱的 `intake-state` 下拉框随 P5，视图的 `Dropdown` 与 `BreadcrumbNavigationSearchDropdown` 随 P6a/P6b，保持 open。

**处理结果（M7/P5a，`dropdowns/intake-state/base.tsx`）：** 这个下拉框已在 M3/P8b 删除（`ba8cb1f4`，连同 `dropdown.tsx`），P5a 不加回收集箱的分诊状态的前端（P2a 不开分诊状态的接口），收集箱的一半按不适用关闭（[P5a spec](../specs/P5a-web-intake.md) R24）。`Dropdown`（`single-select.tsx`）在视图的表单里，随 P6a；`BreadcrumbNavigationSearchDropdown` 随视图的页头。保持 open。

**处理结果（M7/P6a，`Dropdown` 与视图页头的切换器）：** `Dropdown` 一半不适用：`web/packages/ui/src/dropdown/single-select.tsx` 已在 M4/P16 删除（`a3af031b8`），它在视图里唯一的调用方 `dropdowns/layout.tsx`（`LayoutDropDown`）随视图表单不再有布局，在 T5 删除（`5a58db86`，[P6a spec](../specs/P6a-web-project-views.md) R28）；视图页的布局切换在 H20 (page) 三种引擎以键盘核过：宽屏页头的 `LayoutSelection` 以 Tab 与 Enter 选 Table，窄屏一行的 `MobileLayoutSelection` 菜单以键盘选 Calendar，只改草稿、不发请求（`e2e/stories/collaboration/h20-view-pages.spec.ts:331-348`）。`BreadcrumbNavigationSearchDropdown`（视图页的切换器）：P6a 给它加可选的 `onOpen`，在已有的打开回调里调（`web/packages/ui/src/breadcrumbs/navigation-search-dropdown.tsx:25`、`:67-69`），视图页以“打开过”为第一次读列表的门（R20）；三种引擎以键盘核过：焦点在切换器上 Enter 打开，搜索框取得焦点，打字、下箭头选中、Enter 换到另一个视图（`e2e/stories/collaboration/h20-view-pages.spec.ts:52-69`，在 `:125`、`:146` 用）。项目视图的部分关闭；工作区视图的页头（`app/(all)/[workspaceSlug]/(projects)/workspace-views/header.tsx`）随 P6b，保持 open。

## 4. 复制到剪贴板（M7 的 4 处）

- P5 起 `copyTextToClipboard`（`web/packages/utils/src/string.ts`）复制失败时拒绝，纯 http 下也是。M7 的 4 处没有处理拒绝，失败时没有提示，只有一个未处理的 Promise 拒绝（路径在 `web/apps/web/core/components/` 下）：`inbox/content/inbox-issue-header.tsx`、`views/quick-actions.tsx`、`workspace/views/{quick-action,default-view-quick-action}.tsx`。
- **关闭条件**：4 处都改经 `useCopyLink`（`web/apps/web/core/hooks/use-copy-link.ts`，M3/P10；复制文字的是同一文件的 `useCopyText`，M3 收尾），失败时经 `t()` 提示；关键词规则 `copy-hook` 中它们的例外随之删除（M3 收尾，[M3 的交接](M3-closeout.md)）。

**处理结果（M7/P5a，收集箱的页头）：** `inbox/content/inbox-issue-header.tsx` 在 T6a 删除，新的动作模型的“复制链接”经 `useCopyLink`（`inbox/content/header/use-intake-actions.ts:157`），失败时经 `t()` 提示，`copy-hook` 的这个例外同一提交删除（[P5a spec](../specs/P5a-web-intake.md) §2.7）。收集箱的一处关闭；视图的三处随 P6a/P6b，保持 open。

**处理结果（M7/P6a，项目视图的一处）：** `views/quick-actions.tsx` 的“复制链接”经 `useCopyLink`（`:58`），成功与失败各经 `t()` 提示，`copy-hook` 的这个例外同一提交删除（`e2a8ba42`，[P6a spec](../specs/P6a-web-project-views.md) §2.3）；H12 (page) 三种引擎核成功与浏览器拒绝剪贴板两种（`e2e/stories/collaboration/h12-project-views-list-page.spec.ts:351-362`）。项目视图的一处关闭；工作区视图的两处（`workspace/views/{quick-action,default-view-quick-action}.tsx`）随 P6b，保持 open。

## 5. 物理删除与跨模块外键的关系图（M7 的部分）

- **做什么**：M7 的每张新表（评论、通知、保存视图、收集箱等）照 M2 设计 3.13 搬 Django 模型的 `on_delete`，在 M2 设计 4.7 的图上延伸，写明物理删除时每条指向 `users` 和工作项的外键的去向（M3、M4、M6 各自延伸自己的表）。跨模块的联表见第 1 节。
- **为什么**：M2 设计 3.14 把"跨模块的联表查询、物理删除时的外键链由各 M 的设计逐个写明"交给 M3、M4、M6、M7；13.2 的关系图一行只写了 M3、M4、M6。
- **关闭条件**：M7 设计有延伸后的图，每条指向 `users`、工作项的外键写明去向。

**处理结果（M7/P3b，`issue_views` 的部分）：** `owned_by_id → users` 照 Plane `ON DELETE CASCADE`，`created_by_id`、`updated_by_id → users` `SET NULL`；不指向工作项；`workspace_id`、`project_id` `CASCADE`，删除项目、工作区经 `navigation.delete_rows` 任务软删（设计 §4 的表与删除图，[P3b spec](../specs/P3b-views.md) §2.3、§2.7）。收藏与最近访问的两张表随 P3c。保持 open。

**处理结果（M7/P3c，`user_favorites` 的部分）：** `user_id → users` 照 Plane `ON DELETE CASCADE`（v0 不物理删除账户，不触发），`created_by_id`、`updated_by_id → users` `SET NULL`；`parent_id` 自引用 `CASCADE`（清理物理删文件夹时带走与它同一时刻软删过的子行）；目标不是外键（`entity_identifier`），读时只给看得到的目标；`workspace_id`、`project_id` `CASCADE`，删除项目、工作区经 `navigation.delete_rows` 任务软删（[P3c spec](../specs/P3c-favorites.md) §2.3、§2.6）。最近访问的表随 P3d，保持 open。

**处理结果（M7/P3d，`user_recent_visits` 的部分）：** `user_id → users` 照 Plane `ON DELETE CASCADE`（v0 不物理删除账户，不触发），`created_by_id`、`updated_by_id → users` `SET NULL`；不指向工作项：目标不是外键（`entity_identifier`），工作项删除时 navigation 的订阅者在删除的事务里软删所有人指向它的行；`workspace_id`、`project_id`（非空）`CASCADE`，删除项目、工作区经 `navigation.delete_rows` 任务的第三步软删（[P3d spec](../specs/P3d-recents.md) §2.3、§2.6）。navigation 的三张表至此都写明了去向；照设计 §11 保持 open 至收尾关闭。

## 6. CSP 的 `frame-src`

- 见 [M4 的交接](../../M4-issue-core/handoffs/M2-closeout.md)第 8 节：页面的 CSP 没有 `frame-src`，落回 `default-src 'self'`。
- **关闭条件**：M7 的 review 写明收集箱或视图是否嵌入 iframe；嵌入的话，`server/internal/platform/webui/csp.go` 只为它加那个来源，有测试。

**处理结果（M7/P4，通知页）：** 通知页不嵌 iframe：新的 `workspace-notifications/` 与它嵌入的工作项浮层里都没有（`git grep -i iframe -- web/apps/web/core web/apps/web/app web/packages/editor/src` 为空），CSP 不改（[P4 阶段记录](../reviews/P4-web-notifications-review.md) §8）。收集箱随 P5、视图随 P6a/P6b 写明，保持 open。

**处理结果（M7/P5a，收集箱）：** 收集箱页不嵌 iframe：新的 `inbox/` 与它的详情里都没有（`git grep -i iframe -- web/apps/web/core web/apps/web/app web/packages/editor/src` 为空），CSP 不改（[P5a 阶段记录](../reviews/P5a-web-intake-review.md) §8）。视图随 P6a/P6b 写明，保持 open。

**处理结果（M7/P6a，项目视图）：** 项目视图的页面不嵌 iframe：视图列表页、视图页与它们的组件里都没有（`git grep -i iframe -- web/apps/web/core web/apps/web/app web/packages/editor/src` 为空），CSP 不改（[P6a 阶段记录](../reviews/P6a-web-project-views-review.md) §8）。工作区视图随 P6b 写明，保持 open。

来源：[M2 设计](../../M2-auth/M2-design.md) 13.2 中接收者含 M7 的各行、3.12、3.14；[M2/P4 评审](../../M2-auth/reviews/P4-web-auth-review.md)第 4 节；[M2 收尾 spec](../../M2-auth/specs/closeout.md) 第 3 节。
