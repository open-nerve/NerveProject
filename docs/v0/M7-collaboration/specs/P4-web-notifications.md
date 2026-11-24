# M7 / P4：网页的通知与关注

> 状态：草稿；预检 [re-93](../../../reviews/re-93-M7P4预检.md) 修正后通过（处理见它的 §6），T1 提交。基线 main `3d9293bd`（P3d 的 noff）。设计见 [M7 设计](../M7-design.md) §2（H1–H7、H20）、§3.1–§3.4、§3.10、§7（P4 一条 `:242`）、§11、§12（P4 一行 `:303`）、§15；网页的长期规则见[总体设计 §7.2、§7.7](../../v0-design.md#77-stores-按会话分代长期有效m2-起)。接口合同见 [P1a spec](P1a-subscriptions-events.md)（关注、`is_subscribed`）与 [P1c spec](P1c-notification-api.md)（通知的四个接口）。现状调研在台账 `research/web.md`（网页现状、交接、契约，WP-n 指它 §11）与 `research/plane-pages.md`（Plane 的页面、e2e 约定，PP-n 指它 §8；契约陷阱 T1–T7 在它 §1.4；Plane 网页缺陷 W1–W8 在它 §1.10；页面故事在它 §9）。主控裁定在台账 `progress.md`。下文 `C/` = `web/apps/web/core/`，`A/` = `web/apps/web/app/(all)/[workspaceSlug]/(projects)/`，`WN/` = `C/components/workspace-notifications/`，`K` = `tools/keywords.json`，`NY` = `api/modules/notification.yaml`，`IY` = `api/modules/issue.yaml`，`SG` = `web/packages/api-client/src/schema.gen.ts`，`HO/` = `docs/v0/M7-collaboration/handoffs/`，`pw/` = 只读的 `plane/apps/web/`。本段不改 `server/`、`api/`。

## 1. 目标与边界

设计 §12 的 P4：网页的通知与关注；收起“保存为视图”；验收 H1–H7 的页面与 H20 的通知页。P4 是 M7 的第一个页面阶段。

- **通知的数据层**：service、store 与纯函数照总体设计 §7.7 重写，只调 `/api/v0`，类型只来自 `@nerve/api-client`；Plane 的旧 store、service、hook、手写类型、常量与工具整套删除（R2）。
- **通知页**：两个页签与计数、类型筛选（只在“全部”）、未读 / 已归档 / 已暂缓、卡片（文字、执行者、工作项、时间）、加载更多、刷新、全部已读、卡片的已读 / 归档 / 暂缓（预设与自定义；动作随视图收窄，R25）、右侧的预览（嵌入的工作项浮层）。
- **未读数**：顶栏的点与通知页页签上的数，共用一个取数（R5）。
- **关注**：工作项详情页头、浮层页头（含通知页里嵌入的浮层）与命令面板的“关注 / 取消关注”（R13、R14）。
- **收起“保存为视图”**：去掉两处 `enableSaveView`（R17）。
- **守卫与告警**：`old-notification-urls`、`workspace-address-route` 收入改写的文件、`.oxlintrc.json` 的 swr 名单、oxlint 上限 40 → 31、`no-empty-pattern` 升为 error；死成员的基线（M7 第一个页面阶段）。
- **e2e**：新的三引擎页面配置 `e2e/playwright.m7-pages.config.ts` 与六组页面故事（七个文件：H2–H4 的时钟一格单独成文件）；`e2e/fixtures/mounts.ts` 的 `WORKSPACE` 加未读数一条。
- **T1 另补 P3d 阶段记录 §9**（R1）。

**不做：**
- 收集箱的嵌入分诊（P5，设计 `:242`）：收集箱条目的通知在 P4 以普通浮层打开；收集箱组件上只给通知用的两个 prop（`isNotificationEmbed`、`embedRemoveCurrentNotification`，`C/components/inbox/content/{root,inbox-issue-header,inbox-issue-mobile-header}.tsx`）在 P4 之后没有传入者，到 P5 接回，P4 不碰这三个文件（`inbox-issue-header.tsx` 470 行，P5 先拆）。
- 视图（P6a、P6b）：“保存为视图”的弹窗、`C/components/rich-filters/filters-row.tsx:86-101` 的英文字面量（“Save view”“Update view”“Confirming”）、`workspace-level.tsx` 的分支、视图的嵌套按钮与复制。
- 收藏、最近访问（P7）：通知页不记最近访问。
- 应用侧栏的未读数：Plane 的侧栏收件箱入口与它的数从不渲染（见 R5 的证据），不加回。
- M3 交接 §2 的其余三项（P8）。
- 服务端与契约：`archived=true` 是否也列暂缓中的，是 P1c 已合入的合同的语义，跨模块，本段不改（R25）。

**P4 合入后 main 的样子**：通知页经新接口可用；每个工作区页面的顶栏多一次未读数的读；详情、浮层、命令面板有“关注”；项目页、迭代页、模块页的筛选行没有“保存为视图”；网页没有 Plane 的通知地址。

## 2. 交付合同

### 2.1 service

`C/services/notification.service.ts`（新）：构造函数收这一代的客户端（`ApiClient`），每个操作一个方法，经 `unwrap`，回答生成的类型，不转换（先例 `C/services/planning/modules.service.ts`）。

| 方法 | 操作（`NY`） |
|---|---|
| `list(slug, query)` | `listNotifications` `GET /api/v0/workspaces/{slug}/notifications`（`NY:10-87`；查询全是字符串，`SG:9117-9132`，`limit` 是数字） |
| `unreadCount(slug)` | `getNotificationUnreadCount` `GET …/unread-count`（`NY:88-113`） |
| `update(notificationId, body: NotificationUpdate)` | `updateNotification` `PATCH /api/v0/notifications/{id}`（`NY:156-199`） |
| `markAllRead(slug, body: NotificationMarkAllRead)` | `markAllNotificationsRead` `POST …/mark-all-read`（`NY:114-155`） |

`C/services/issue/issue-detail.service.ts`（68 行）加 `subscribe(issueId)`、`unsubscribe(issueId)`：`POST /api/v0/issues/{issue_id}/subscribe`、`/unsubscribe`，204（`IY:1292-1349`）。`createClient` 的 CSV 名单（`web/packages/api-client/src/index.ts:35-40`）不加：`type` 是一个字符串，由纯函数拼。

### 2.2 纯函数

全部有单测，逐条覆盖下面每个陷阱。

**`C/store/notification/filters.ts`**（筛选的模型与序列化）：
- `NotificationFilters = { tab: NotificationTab; types: readonly NotificationType[]; unread: boolean; archived: boolean; snoozed: boolean }`；`DEFAULT_FILTERS` 为“全部”、没有类型、三个都是 `false`。
- `filterKey(filters)` 与 `filtersOf(key)`：规范的字符串，可逆。类型按 `assigned, created, subscribed` 的次序；“提及”页签不含类型。`filtersOf(filterKey(f))` 等于规范化之后的 `f`：列表的 fetcher 只收键里的字符串（`C/lib/use-session-swr.ts:28-32`“what it fetches is what the key names”），由这一段算出筛选，不闭包当前的 `filters`（re-93 m5）。
- `listQuery(filters, cursor?)`：
  - 总带 `tab`、`archived`、`snoozed`（`"true"`/`"false"`）与 `limit: 100`（R21）。
  - `type` 只在“全部”页签且类型非空时带，逗号连接。“提及”页签带 `type` 答 422（T2，`NY:26-29`）；空的成员答 422（`NY:27`）。
  - “只看未读”是 `read: "false"`，不选时不带 `read`（T5：`read=true` 只列已读的）。
  - `cursor` 只取上一页回答的 `next_cursor`（T4：游标不透明，自造的答 400）。
- `markAllBody(filters)`：`{tab, archived, snoozed}`，“全部”页签有类型时加 `type` 数组；没有 `read`（`NY:460-480`）。类型为空时不发 `type`：`[]` 不合“一或多个”（`NY:464`，本 spec 加的陷阱 T8）。
- `snoozedNow(n, now)`：`snoozed_till !== null` 且晚于 `now`。过去的 `snoozed_till` 是普通的、计入未读的通知（T6，`NY:68-76`）。
- `matches(n, filters, now)`：页签（“提及” ⇔ `sender === "mentioned"`）、类型、`archived` 相等、`snoozedNow` 与 `snoozed` 相等。**不看 `read`**（R8：读状态的变化留在原处）。
- `viewOffers(filters)`：`{archive, snooze}`。“已暂缓”的视图里 `archive` 为假，“已归档”的视图里 `snooze` 为假，其余为真（R25：页面不造出既归档又在暂缓中的通知）。

**`WN/card/describe-change.ts`**：`describeChange(change, format)` 回答 `{key, params}`，见 §2.6。`format` 注入日期与优先级的显示，函数本身不读语言与时区。

**`WN/card/snooze-time.ts`**（R15）：
- `SNOOZE_PRESETS`：1、3、5、7、14 天，文案用已有的 `notification.snooze.{1_day,3_days,5_days,1_week,2_weeks}`（Plane 的 `plane/packages/constants/src/notification.ts:65-111`）。`presetMoment(days, now)` = 此刻 + N×24 小时。
- `snoozeSlots(day, now)`：一天 48 个半小时，值是 `"HH:MM"`（00:00–23:30）。`day` 是今天时只给晚于此刻的。
- `slotMoment(day, slot)`：浏览器时区的那一刻，ISO 串。
- `slotLabel(slot, locale)`：经 `Intl.DateTimeFormat` 按语言标注（en 为 12 小时制，zh-CN 为 24 小时制）。值是 24 小时制，所以 Plane 的 12 AM / 12 PM 颠倒（W1，`WN/sidebar/notification-card/options/snooze/modal.tsx:99-101`）在构造上不再存在。

### 2.3 store

**位置**：`C/store/notification/notification.store.ts` 放值、读与取数；`notification-changes.ts` 放写；`filters.ts` 见 §2.2。每个约 200 行以内（先例 P6a 的 `module.store.ts` + `module-changes.ts`）。

**接线**：`RootStore` 加 `notification: NotificationStore`，构造时传这一代的 `api`。旧的 `workspaceNotification` 在 T5 删除（R2）。`C/store/root.store.test.ts:106` 的 store 数在 T2 由 21 变 22，T5 回到 21。hook `useNotifications()` 在 `C/hooks/store/use-notifications.ts`。

**值**（`C/lib/reconciled.ts`，v0 §7.7“取数和修改的应答对齐”）：

| 值 | 键 | 来源 |
|---|---|---|
| `lists: ReconciledByKey<NotificationPage>` | `` `${workspaceId}|${filterKey}` `` | 第一页 `fetch`；后续页 `fetchPart`（§2.4） |
| `counts: ReconciledByKey<NotificationUnreadCount>` | 工作区 id | `unreadCount` |

store 另记“取过的列表”：键 → `{workspaceId, filters}`。它是普通的 `Map`，只给写的确认找这个工作区持有的每一份。

**读**：
- 收地址里的 slug，经调用者此刻的工作区列表（`workspaceRoot.getWorkspaceBySlug`）找到 id。不读 router store。
- `getList(slug, filters)`、`getUnreadCount(slug)`。没有 `hasMore`：列表读持有页的 `next_cursor`。
- 不在调用者工作区列表里的 slug 一律回答 `undefined`（v0 §7.7“按工作区的 id 存”）。

**取数**：
- `fetchList({id, slug}, filters)`：第一页，`Reconciled.fetch`，在外的后续页随之作废（R7）。
- `fetchMore({id, slug}, filters)`：持有的页有 `next_cursor` 时，以它为 part 经 `fetchPart` 取下一页。合并时接上本页没有的行，`next_cursor` 取这次的回答。没有 `next_cursor` 时不发请求。
- `fetchUnreadCount({id, slug})`。
- 失败是 SWR 的 `error` 或调用者收到的拒绝，不是未处理的拒绝。`SessionChangedError` 给 `undefined`。

**写**（`notification-changes.ts`，一个 `oneAtATime()`，请求体在轮到它时算）：
- `update(notification, patch)`：PATCH。nerve 确认后：
  - 回答写进这个工作区持有的每一份列表：仍 `matches` 的替换；`archived`、`snoozed` 不再相符的移出；读状态的变化原地替换（R8）。
  - 然后重取这个工作区的未读数（R6）。
  - `notification.not_found`：从这个工作区持有的每一份移出，重取未读数，再把拒绝交给调用者（R8）。
  - 其余拒绝交给调用者，值不变。
- `markAllRead({id, slug}, filters)`：POST `markAllBody(filters)`，回答 `{marked}`；不论成败，之后都重取这组筛选的第一页与未读数，再交出回答或拒绝（R9）。服务端分批提交，某一批失败时之前的批已经标了（`NY:131-134`），所以被拒之后也要重取（re-93 m2）。不在本地把行标成已读。
- 拒绝一律交给调用者（`ApiError`），store 不吞、不打日志。

**关注**（R13）：
- 新 `C/store/issue/issue-details/subscription.store.ts`（`IssueSubscriptionStore`），挂在详情根 store 上：`root.store.ts` 照 `reaction` 的接线（`:30-31`、`:69`、`:86`、`:110`、`:151-157`）加 `subscription`。
- 方法：`subscribe(ws, project, id)`、`unsubscribe(ws, project, id)`、`isPending(id)`。
- 写经工作项的队列 `root.writes.enqueueIssueChanges([id], …, {workspaceSlug, projectID})`（`C/store/issue/issue-writes.ts:56-60`）。POST 之后，不论成败，都在同一个排队的操作里 `root.issues.refreshAuthoritative(id, { afterWrite: true })`（先例 `C/store/issue/issue-details/reaction.store.ts:51-76`），`is_subscribed` 取自这次 `getIssue`。拒绝在重读之后再交给调用者。
- 不在本地设 `is_subscribed`。
- service 用 issue 根 store 已有的 `detailApi`（`C/store/issue/root.store.ts:153`）。

### 2.4 取数、门与 `ReconciledByKey.fetchPart`

| SWR 键（经 `useSessionSWR`，`sessionKey` 加 `loginId`） | 谁读 | 门 |
|---|---|---|
| `["NOTIFICATION_UNREAD_COUNT", workspaceId, slug]` | 顶栏（每个工作区页面）与通知页的页签：都经一个 hook `useUnreadNotificationCount()`（`C/hooks/use-unread-notification-count.ts`），键只写这一处 | `useRouteWorkspace()` 不为 `null`：工作区的有效成员，管理员、成员、访客都取（`notification.read`，设计 §3.10 `:143`） |
| `["NOTIFICATIONS", workspaceId, slug, filterKey]` | 通知页的列表 | 同上 |

- 列表的 fetcher 收键里的 `filterKey`，经 `filtersOf` 得到筛选，调 `fetchList`（§2.2）。
- “加载更多”、刷新与写的跟进直接调 store，不另开 SWR 键。
- 重新联网时 SWR 照常重读两个键，列表的这次重读就是刷新：丢掉已加载的后续页（R7）。不传 `revalidateOnReconnect: false`：`useSessionSWR` 只让能力的持有者关掉它（`C/lib/use-session-swr.ts:20-21`）。
- 只有 `useSessionSWR` 的一份配置（`C/lib/use-session-swr.ts:25`）：挂载与换键时取，聚焦时不取（R6）。通知页挂载时同一个计数键的挂载读就是“页面挂载时重取”。
- 新的取数文件（hook、页面、store 目录）加进 `.oxlintrc.json` 的 `no-restricted-imports` swr 名单（`.oxlintrc.json:119-215` 的 override）。

**`ReconciledByKey.fetchPart(key, part, read, merge)`**（R4）：委托给这个键的 `Reconciled.fetchPart`（`C/lib/reconciled.ts:85-107`）；键还没取过时回答 `undefined`、不建条目（`Reconciled.fetchPart` 在 `value === undefined` 时回答 `undefined`，`:86`）。单测：
- 同一 part 的新读占有；
- 整页的 `fetch` 作废在外的 part；
- 在外时确认的修改重放到 part 的回答上；
- 别的键不受影响；
- 没取过的键不发 `read`。

### 2.5 页面状态

`WN/page-state.tsx`（新）：一个 React context，由 `A/notifications/layout.tsx` 提供（R3）。
- **状态**：`filters: NotificationFilters` 与 `selected?: {notificationId, issueId, projectId}`。
- **改法**：`setTab`、`toggleType`、`clearTypes`、`toggleUnread`、`toggleArchived`（同时关掉 `snoozed`）、`toggleSnoozed`（同时关掉 `archived`，Plane 的互斥，`pw/core/components/workspace-notifications/sidebar/header/options/menu-option/root.tsx:36-93`）、`select`、`clearSelection`。
- **两个开关互斥，卡片的动作随之收窄**（R25）：服务端的两个筛选是相与的等式（`server/internal/modules/notification/adapter/postgres/queries/inbox.sql:19-20`，`NY:25-26`），页面发不出 `archived=true&snoozed=true`。所以“已暂缓”的视图里卡片不给“归档”、“已归档”的视图里卡片不给“暂缓”（`viewOffers`），页面造不出一条哪个视图都看不到的通知。
- **范围**：它在以路由的 slug 为 key 的 `WorkspaceContentWrapper` 之下（`web/apps/web/app/(all)/[workspaceSlug]/layout.tsx:21`），所以换工作区（后退、前进也是）随之结束。离开通知页同样结束（W6）。
- **选中的通知**：记它自己的工作项与项目，所以换页签、筛选之后预览照旧（Plane 换页签不清选中）。三种情况清掉：关掉浮层；预览的项目读被拒；选中的那一行因 `notification.not_found` 被移出（§2.6 的预览一行，re-93 I4）。

### 2.6 页面与组件

所有页面经 `useRouteWorkspace()`（`C/hooks/use-route-workspace.ts:16-20`）或路由参数得到工作区，不读 `currentWorkspace`。拒绝一律经 `useRefusalToast`（`C/hooks/use-refusal-toast.ts:28-34`），`await` 之后的跟进经 `followInSession`。`WN/` 改写为下表的文件，旧文件在 T5 全部删除（§2.9）。

| 页面 / 组件 | 合同 |
|---|---|
| **路由** `A/notifications/layout.tsx`、`page.tsx` | 组件改名为 `NotificationsLayout`、`NotificationsPage`（原名 `ProjectInboxIssuesLayout`、`WorkspaceDashboardPage`）。布局提供页面状态，渲染侧栏与 `<Outlet/>`。页面标题 `notification.page_label`，工作区名取自 `useRouteWorkspace()`（原读 `currentWorkspace`，`page.tsx:22-26`）。应用侧栏在通知页隐藏的现状不变（`A/_sidebar.tsx:27`）。 |
| **侧栏** `WN/sidebar/root.tsx` | 页头、页签、已选筛选、列表。窄屏选中之后只显示预览（Plane `w-full md:w-3/12`，`pw/.../sidebar/root.tsx:58-63`）。 |
| **页头** `WN/sidebar/header.tsx`、`header-options.tsx` | 面包屑 `notification.label`。<br>**全部已读**：请求体取页面状态，在途时停用并转圈；成败都重取（R9），被拒时提示。<br>**刷新**：重读第一页与计数，在途时转圈（R7）。<br>两个按钮的图标沿用外部依赖 `@makeplane/propel/icons` 的 `CheckDoneOutline`、`RefreshOutline`（旧页头 `WN/sidebar/header/options/root.tsx:8` 是它们在本仓唯一的读者，新页头接着读）。<br>**类型筛选菜单**（只在“全部”）与**视图菜单**（只看未读 / 已归档 / 已暂缓）：都是 `PopoverMenu`，按钮内是图标加 `sr-only` 的名字，不嵌套按钮；项是 `<button aria-pressed>`，Enter、Space 各切换一次，不提交外层的表单（先例 `C/components/issues/issue-detail/issue-activity/activity-filter.tsx:28-74`；旧 `menu-option-item.test.tsx` 钉的性质由新测试接上）。**项之间用 Tab**：`PopoverMenu` 是 Headless `Popover` 加一组普通按钮，没有方向键（`web/packages/ui/src/popovers/popover.tsx:17-90`、`popover-menu.tsx:14-51`），不给项加 `onKeyDown`（re-93 I3）；Escape 关上并把焦点还给按钮。 |
| **页签** `WN/sidebar/tabs.tsx` | Headless `Tab`（先例 `C/components/cycles/active-cycle/cycle-stats.tsx:67-93`）。名字含未读数：“全部”用 `all`，“提及”用 `mentions`（`NY:445-458`：`all` 不含提及），0 时不显示数，99 以上显示 `99+`（`getNumberCount` 留用，`web/packages/utils/src/string.ts:53`，读者由旧侧栏换成 `WN/sidebar/tabs.tsx`；经 `CountChip` 显示）。换页签时列表换键：SWR 没有这个键的回答时取，有时照 `revalidateIfStale` 重取（v0 §7.7）。 |
| **已选筛选** `WN/sidebar/applied-filters.tsx` | 只在“全部”：每个类型一个可去掉的小块，加“清除全部”（`common.clear_all`）。“提及”里隐藏，“全部”里的选择保留（R10）。 |
| **列表** `WN/sidebar/list.tsx` | 卡片。<br>“加载更多”是 `<button>`，有 `next_cursor` 时出现，在途时停用并显示 `common.loading`，失败时提示。持有的页零行而有 `next_cursor` 时（第一页的行被逐条归档、暂缓移出）显示“加载更多”，不显示空态（re-93 m6）。<br>首读时显示占位（`WN/sidebar/loader.tsx` 留用）。<br>读失败时显示可重试的错误态（复用 `C/components/planning/read-failure.tsx` 的 `PlanningReadFailure`，它只收 `error` 与 `onRetry`），不是永远加载。<br>空时按页签显示空态（`WN/sidebar/empty-state.tsx` 留用，文案 `workspace_empty_state.inbox_sidebar_{all,mentions}.title`）。<br>没有一个区域以 “Notifications” 为名：e2e 的提示定位器 `getByLabel("Notifications", { exact: true })` 认的是提示区（`e2e/fixtures/cycle-pages.ts:252-253`）。 |
| **卡片** `WN/card/item.tsx`、`sentence.tsx` | 一个容器，里面有一个主按钮和与它同级的动作按钮（R16）。<br>**主按钮**：头像、执行者与句子、第二行是快照的 `{identifier}-{sequence_id} {name}`、时间。暂缓中（`snoozedNow`）显示钟与 `notification.card.till`，否则显示“多久之前”（`calculateTimeAgo`）。未读的点带 `sr-only` 文字。按下时选中这条通知、打开预览；未读的同时排队 `update(n, {read: true})`，它的拒绝提示，预览照常打开（PP-13、R23）；拒绝是 `notification.not_found` 时这一行已被移出，选中随之清掉（re-93 I4）。<br>**执行者**：照 R22 的规则。<br>**摘录**：`excerpt` 不为 `null` 时（评论的建、改与评论里的提及）以纯文本显示，至多两行；不用 `LiteTextEditor`、不显示图片（M5 交接 M5-M7-03，设计 §11 `:282`）。<br>**附件**：不链到文件（M5-M7-02，`HO/M5-closeout.md:38`）。<br>以通知的 id 为 key。 |
| **卡片的动作** `WN/card/actions.tsx`、`snooze-menu.tsx` | 已读 / 未读、归档 / 取消归档、暂缓菜单：按钮以这一行命名（`notification.card.actions.*`，参数是工作项的编号与名字），悬停与 `focus-within` 时显示，键盘能到。<br>暂缓菜单是 `CustomMenu`：预设、“自定义”（打开 §2.7 的弹窗），暂缓中时首项为“取消暂缓”，发 `{snoozed_till: null}`（T1：Plane 发 `undefined`，nerve 里留空的字段不改）。<br>**随视图收窄**（R25，`viewOffers`）：“已暂缓”的视图里没有“归档”，有暂缓菜单（首项“取消暂缓”）；“已归档”的视图里没有暂缓菜单，有“取消归档”；默认视图三种都有。<br>每个动作在途时停用；成功提示用已有的 `notification.toasts.*`；被拒时提示；`notification.not_found` 时这一行随之消失，是选中的那一行时选中也清掉。 |
| **预览** `WN/root.tsx` | 没有选中时显示 `EmptyStateCompact`。选中时设浮层 `{workspaceSlug, projectId, issueId}`，渲染 `IssuePeekOverview`，带 `embedIssue`、`embedRemoveCurrentNotification`（关掉时清选中），`hostScope` 为 `JSON.stringify(["notification", slug, notificationId])`（这个 prop 是字符串，`C/components/issues/peek-overview/root.tsx:32-80`，现有的写法 `WN/root.tsx:99-104`）。<br>选中通知的项目不在调用者的项目里时（已归档的项目：包装层只读 `archived=false`，`e2e/fixtures/mounts.ts:25`），经 `useProjectRead(projectId)`（`C/hooks/use-project-read.ts:5-10`）读它；浮层只在角色已知时渲染（`peek-overview/root.tsx:38-44`，`C/store/user/permissions.store.ts:67-73`）。<br>**浮层渲染不出来的三种情况都要有交代**（re-93 I4；否则右侧一块空白，窄屏上只剩浏览器的后退）：项目读在路上时显示占位（`WN/sidebar/loader.tsx` 的样子）；项目读被拒（人已离开项目、没连上）时清掉选中并经 `useRefusalToast` 提示，回到没有选中的样子；选中的那一行因 `notification.not_found` 被移出时清掉选中（卡片一行）。<br>收集箱条目照普通工作项打开（R12）。<br>页面卸载时清掉浮层（Plane `pw/.../root.tsx:77-83`）。旧的收集箱分支与它的裸 `useSWR`（`WN/root.tsx:61-66`、`:81-97`）删除。 |
| **顶栏** `C/components/navigation/top-navigation-root.tsx` | 取未读数（§2.4）。`all + mentions > 0` 时显示点（Plane 的“先提及、再全部”判的是同一件事，`pw/core/components/navigation/top-navigation-root.tsx:30-45`）。<br>链接的名字是 `notification.label`，有未读时加 `notification.a11y.unread_count`。`AppSidebarItem`（`C/components/sidebar/sidebar-item.tsx:14-20`；链接 `:75-82`、按钮 `:84-90`）加可选的 `label`，作链接、按钮的 `aria-label`。<br>`Tooltip` 的英文 “Inbox”（`top-navigation-root.tsx:45`）改为 `t("notification.label")`。旧 store 的读（`:24-31`）在 T2 去掉。 |
| **关注按钮** `C/components/issues/issue-detail/subscription-button.tsx`（新） | 放在详情页头（`issue-detail-quick-actions.tsx` 复制链接之前，Plane `pw/.../issue-detail-quick-actions.tsx:146-148`）与浮层页头（`peek-overview/header.tsx`，Plane `:211-213`）。<br>**谁看得到**（R14）：在项目里有角色（`getProjectRoleByWorkspaceSlugAndProjectId` 不为 `undefined`，访客也算），持有的是详情（带 `is_subscribed`），工作项与项目都未归档。<br>**显示**：`common.actions.subscribe`/`unsubscribe` 随 `is_subscribed`；在途时显示 `common.loading` 并停用；文字在窄屏也是名字（Plane 窄屏只剩图标、没有名字，W7）。<br>**提示**：成功 `issue.subscription.actions.subscribed`/`unsubscribed`，被拒经 `useRefusalToast`。 |
| **命令面板** `C/components/power-k/ui/pages/context-based/work-item/subscription-command.ts`（新） | 加进 `commands.ts:7-12` 的数组。<br>`subscribe_work_item`：标题 `power_k.contextual_actions.work_item.subscribe`/`unsubscribe` 随状态，`modifierShortcut: "shift+s"`（现在没有人用，`cycle-command.test.tsx:202` 钉快捷键不重）。<br>可见与可用的条件同按钮；提示同按钮。<br>`mutation-commands.ts`（182 行）不改。 |

**卡片的文字**（`describeChange`，R11）：

- **规则**：每个 `(field, verb)` 一个键，加入与移出由新旧值区分（P1b `specs/P1b-notification-core.md:46-56`、`:142`）。
- **“有新值”是非空串**：移除时服务端给的 `new_value` 是空串 `""`，不是 `null`；设置时 `old_value` 也可能是 `""`（M4 动态的值原样进事件，`server/internal/modules/issue/app/issue_events_test.go:122-123`、`:141-144`；通知原样存，`server/internal/modules/notification/domain/changes.go:157-161`）。下表的“有新值”指 `new_value` 既不是 `null` 也不是 `""`。
- **父工作项与关联**：不读 `old_value`/`new_value`（服务端把父工作项的两个值置空，`server/internal/modules/notification/domain/changes.go:176-182`；关联只带 id，`:217-235`），另一端取 `new_issue`/`old_issue`。两者都为空时用不点名的说法（H4）。
- **键**：都在 `notification.card.*` 之下，en 与 zh-CN 同一提交加。表里写完整的键字面量，不用模板拼，`tools/i18n-keys.mjs` 才认得每一个。

| `field` / `verb` | 键（`notification.card.` 之后）与参数 |
|---|---|
| `issue` / `created` | `created` |
| `name` / `updated` | `renamed` `{value}` |
| `priority` / `updated` | `priority` `{value}`（`format.priority`） |
| `state` / `updated` | `state` `{value}` |
| `start_date`、`target_date` / `updated` | 有新值时 `start_date_set`、`due_date_set` `{value}`（`format.date`：按日历日显示，经 `renderFormattedDate` 的 `getDate`，`web/packages/utils/src/datetime.ts:19-40`、`:187-205`，不经 `new Date` 的时刻，负偏移的时区不差一天，re-93 m15）；否则 `start_date_removed`、`due_date_removed` |
| `labels`、`assignees` / `updated`（负责人另有 `created`） | 有新值时 `label_added`、`assignee_added` `{value}`；否则 `label_removed`、`assignee_removed` `{value: old_value}` |
| `parent` / `updated` | 有 `new_issue` 时 `parent_set` `{ref}`；只有 `old_issue` 时 `parent_removed` `{ref}`，文案说“不再以 {ref} 为父工作项”，不断言现在没有父工作项：新的父工作项读的人看不到时 `new_issue` 同样为空，客户端分不出这两种（`new_id` 也被置空，re-93 m14）；都没有时 `parent_changed` |
| `relates_to`、`duplicate`、`blocking`、`blocked_by` / `created`、`deleted` | `<类型>_added` / `<类型>_removed` `{ref}`；另一端看不到时 `<类型>_added_hidden` / `<类型>_removed_hidden` |
| `archived_at` / `updated` | 新值为 `restore` 时 `restored`，否则 `archived` |
| `comment` / `created`、`updated`、`deleted` | `commented`、`comment_edited`、`comment_deleted` |
| `link` / `created`、`updated`、`deleted` | `link_added`、`link_updated` `{value}`，`link_removed` `{value: old_value}` |
| `attachment` / `created`、`deleted` | `attachment_added`、`attachment_removed` |
| `description` / `mentioned`、`comment` / `mentioned` | `mentioned_description`、`mentioned_comment` |
| 其余 | `updated`（契约以后多出的字段） |

`{ref}` 为 `` `${identifier}-${sequence_id} ${name}` ``。

### 2.7 自定义暂缓的弹窗

`WN/card/snooze-modal.tsx`，状态归卡片所有，不靠弹出层关闭的定时器（Plane 的 50 ms 与 500 ms，`modal.tsx:52-63`）。

- **弹窗**：`ModalCore` 带 `handleClose`（Escape 与点外面都经它）。在途时关不掉（M3 交接 §3 `:36`）。标题经 `ModalTitle`（`notification.snooze_modal.title`，M3 交接 `:37`）。关上之后焦点回到这一行的暂缓按钮。
- **日期**：`DateDropdown`，最早今天。
- **时间**：`CustomSelect`，选项来自 `snoozeSlots`。日期是今天时只给将来的时段；没有时段时说明 `notification.snooze_modal.no_time`。保留 `CustomSelect` 是为了 M2 交接 §3 的浏览器核对（`HO/M2-closeout.md:40-42`）。Plane 的 AM/PM 两个可点的 `div`（`modal.tsx:174-196`，M3 交接 `:46`）不再有。
- **字段规则**：两个都必填，`useForm({ shouldFocusError: false })`。没过规则时 `FieldError` 显示提示，控件以它为描述并带 `aria-invalid`，经 `useRefusedFocus`（`C/lib/refused-focus.ts:29`）取得焦点（M3 交接 `:44`）。
- **提交**：`update(n, {snoozed_till: slotMoment(...)})`。服务端的 `must_be_future`（`snoozed_till` 的 422，`C/lib/error-messages.ts:122`）显示在时间一栏下；其余拒绝提示，弹窗留着。
- **文案**：都经 `t()`，键在 `notification.snooze_modal.*` 之下（Plane 的英文 “Customize Snooze Time” 等，`modal.tsx:118-221`）。

### 2.8 收起“保存为视图”

- 删去 `C/components/issues/issue-layouts/roots/project-layout-root.tsx:85` 与 `planning-page-layout-root.tsx:100` 的 `enableSaveView`。筛选行就不再给出会 404 的“保存视图”（设计 `:241`）。
- `ProjectLevelWorkItemFiltersHOC` 的保存与更新分支（`C/components/work-item-filters/filters-hoc/project-level.tsx:62-166`）、`workspace-level.tsx:37-202` 与 `C/components/rich-filters/filters-row.tsx:86-101` 留给 P6a/P6b 接回与改写（R17）。

### 2.9 删除

T5 一个 Task 删除（R2），以 `make knip`、`old-notification-urls` 与死成员前后的整份差异核对没有残留。`make knip` 查不出包的 barrel 导出（各包的入口是 `src/index.ts`，`knip.jsonc` 没有 `includeEntryExports`），所以下面的包导出由人对照列出（re-93 I2）：
- `WN/` 的旧文件：29 个、1,864 行（`research/web.md` §1.1；新件按 §2.6 重建，`sidebar/loader.tsx`、`sidebar/empty-state.tsx` 留用）。
- 旧数据层：`C/store/notifications/{workspace-notifications.store.ts,notification.ts}`、`C/services/workspace-notification.service.ts`（模块级单例 `:116`，M2 交接 §2）、`C/hooks/store/notifications/*`、`C/hooks/use-notification-preview.tsx`。
- 手写类型与常量：`web/packages/types/src/workspace-notifications.ts` 与 `types/src/index.ts:29` 的导出（M3 交接 §2 `:29` 的 17 行死成员随之消失）、`types/src/enums.ts:15-20` 的 `ENotificationFilterType`、`web/packages/utils/src/notification.ts` 与 `utils/src/index.ts:19` 的导出。
- 随旧树失去最后一个产品读者的 `@nerve/utils` 导出：`convertToEpoch`（`web/packages/utils/src/datetime.ts:231`，唯一的读者是旧 store `:22`）、`stripAndTruncateHTML`（`string.ts:80`，读者是旧卡片 `content.tsx:14` 与要删的 `utils/src/notification.ts:7`；新卡片的摘录本来就是纯文本，`NY:410-415`）与它在 `web/packages/utils/src/string.test.ts:13`、`:17-29` 的用例。`stripAndTruncateHTML` 删掉之后 `parseHTML`、`truncateText` 仍有别的读者（`string.ts:107` 与六处外部读者），不动。`getNumberCount` 留用（页签，§2.6）。`CheckDoneOutline`、`RefreshOutline` 是外部依赖 `@makeplane/propel`（`web/apps/web/package.json:25`，`catalog:`）的图标，不归本仓删；新页头接着读（§2.6）。
- `web/packages/constants/src/notification.ts` 改写：只留新页面读的项（页签、类型选项与它们的文案键，由 `NotificationTab`、`NotificationType` 推出，同 P6a 的 `ModuleStatus` 做法）。加载态、查询参数的枚举、12 小时制的时段与旧的暂缓选项删除；预设的天数归 `snooze-time.ts`。
- `RootStore.workspaceNotification` 与它的接线（`C/store/root.store.ts:37-38`、`:72`、`:106`）。
- 不再有读者的文案键：只在 T9 一处做（那时通知页的读者已齐）。`tools/i18n-keys.mjs` 只查“读了没定义的键”，查不出“定义了没人读的键”，所以是人工对照：`notification.*` 之下本段新页面不读的旧键逐个 `grep` 过，台账列出删了哪些（re-93 m3）。别的命名空间不动。

### 2.10 守卫与告警

- **`K`**：
  - `phase` 由 `M7/P3` 改为 `M7/P4`（T1，M3 交接 `:59`）。
  - 新规则 `old-notification-urls`：`web/` 的源文件（测试也算）不得出现 Plane 的通知地址，照 `old-module-urls`（`K:3026-3086`）以 `/api/workspaces/` 锚定：`/api/workspaces/(?:[^/"'\`\s]+/)+users/notifications(?![\w-])[^"'\`\s]*`。
    - 命中的样例是旧 service 的八种（`C/services/workspace-notification.service.ts:21-108`：列表、`unread/`、`{id}/`、`read/`、`archive/`、`mark-all-read/`）与无尾段的一例。
    - 不命中的样例是 `/api/v0/workspaces/{slug}/notifications`、`/api/v0/workspaces/{slug}/notifications/unread-count`、`/api/v0/notifications/{notification_id}`、页面链接 `/${workspaceSlug}/notifications`、组件导入 `@/components/workspace-notifications`。
    - 文件样例：命中旧 service，不命中 spec 与 `K` 自己。
  - `workspace-address-route`（`K:2785`）的文件表加入 `A/notifications/page.tsx` 与 `WN/root.tsx`（M3 交接 `:53` 的通知两个文件），以及本段新写的读工作区的文件，带样例。
- **oxlint**：
  - 本段改到的文件告警为零。告警全在 T5 删除的三个旧文件里，共 9 条：暂缓弹窗 5 条、暂缓根 1 条、菜单项 3 条（`research/web.md` §8）。
  - `lint-cap` 要求告警数恰好等于上限，所以 T5 同一提交把 `web/apps/web/package.json:10` 的上限 40 → 31，并在 `.oxlintrc.json` 把 `no-empty-pattern` 设为 `"error"`：全仓唯一的一条**告警**在 `WN/.../snooze/root.tsx:40`；产品代码里另有一处被行内 `oxlint-disable` 压住的空解构，`C/components/issues/issue-detail/issue-activity/activity/activity-list.tsx:46-47`（`comment: {}`，什么也不取），T5 同一提交删掉它与那行 disable，于是网页的产品代码里这一类清零，这是 M1 交接“按规则清一类”（`HO/M1-closeout.md:27-30`；re-93 m8）。e2e 里的十几处是 Playwright 夹具要求的 `({}, …)` 写法，各有行内 disable，不动。先例是 `no-unused-expressions`，`.oxlintrc.json:42`。
  - 两条 `oxlint-disable`（`WN/sidebar/root.tsx:71`、`WN/notification-card/root.tsx:51`）随文件删除。
- **死成员**（M1 交接 `HO/M1-closeout.md:14-22`，设计 §11 `:278` 首个页面阶段跑基线）：
  - T2 在台账跑基线：先 `react-router typegen`，再 `deadsym.mjs --tsv` 与 `domains.mjs --rows M7`。脚本取自 P6a 的归档 `.superpowers/sdd/M6-P6a-integration-20261114/payload/worktree/.superpowers/sdd/P6a-web-modules/dead-members/`，TypeScript 的路径是 `node_modules/.pnpm/typescript@5.8.3/`。
  - 另读 `--rows M3` 中通知的路径：`DOMAINS` 把 `…/workspace-notifications.ts` 与 `WN/` 直下的文件归到 M3，`docs/v0/M1-frontend-trim/plans/closeout.md:1526`。
  - 每个 Task 删去它改到的文件里列出的行。T5 删除之后重跑，比对 `dead-before` 与 `dead-after` 的**整份**差异：消失的行是通知的；新出现的行（读者随旧树删掉的成员）每一行都要有去处，删掉或写明留用的读者（re-93 I2）。T15 再跑一次，把剩下的（P5–P7 的）写进阶段记录。
- **i18n**：新文案 en 与 zh-CN 同一提交（`check:sync` 也不许一个键是另一个键的前缀）。没有新的问题码：`notification.not_found`、`must_be_future`、`issue.not_found`、`forbidden`、`server_busy`、`workspace.not_found` 都已有文案（`C/lib/error-messages.ts:43`、`:52`、`:83`、`:104`、`:122`）。

### 2.11 e2e

**页面配置** `e2e/playwright.m7-pages.config.ts`（新，照 `playwright.m6-pages.config.ts`）：
- `testMatch: /collaboration\/h\d+(?:-h\d+)?-[\w-]+-pages?\.spec\.ts$/`；
- `grep: /H\d+(?:–H\d+)? \(page\)/`（en dash，同 API 故事的标题）；
- chromium、firefox、webkit 三种引擎；
- `e2e/package.json` 加 `test:m7-pages`。

CI 照常只跑默认配置，每个页面故事在 chromium 跑一次。P5–P7 的页面故事加进同一个配置。`testMatch` 对下表七个文件都中，对现有的 API 故事都不中（re-93 §3 第 7 项核过）。

**挂载清单**：`e2e/fixtures/mounts.ts:22-27` 的 `WORKSPACE` 加 `GET /api/v0/workspaces/{slug}/notifications/unread-count`，头注释（`:5-14`）随之改写，与顶栏的取数同一个 Task（T2）。读它的六个文件只经常量跟随，断言不改：`stories/smoke/s2-web-app.spec.ts`（582 行，不加长）、`s2-issue-list-removal.spec.ts`、`project/p2-visibility.spec.ts`、`planning/g3-change-cycle-page.spec.ts`、`g8-archive-cycle-page.spec.ts`、`g13-archive-delete-module-page.spec.ts`。

**新夹具**：
- `e2e/fixtures/notification-pages.ts`：世界、地址、定位器。与迭代共用的 `toast`、`dismissToasts`、`recordApi`、`tabKey`、`tabStops`、`failedLoads` 从 `cycle-pages.ts` 导入，不复制；`cycle-pages.ts` 388 行，不加长。
- `e2e/fixtures/assert/notification.ts`：只读的库读。`subscriptionRows` 从 `stories/collaboration/h1-subscription.spec.ts:28-39` 移来，两个版本断言同一份行；另有通知行的 `read_at`、`archived_at`、`snoozed_till`。

**写法**：
- 世界经接口建。
- 每个故事 `watchPage` + `recordApi`，以 `expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []])` 与 `expectQuietConsole` 收尾。
- 描述不带图片（M6-M8-07 的浮层图片中止）。
- 不在重新载入之后核对控制台（M6-M8-09）。
- 应用内跳转之后以目标页独有的内容为锚。
- 提示用 `dismissToasts`。
- 精确的请求表只在直接打开通知页时断言：那时顶栏与页签同一次挂载，计数只取一次。应用内从别的页进来时，页签的挂载读会不会再发一次取决于 SWR 的去重窗口，那里用 `expect.poll` 等内容，不数计数的请求（re-93 m12）。

| 文件（`e2e/stories/collaboration/`） | 标题 | 覆盖 |
|---|---|---|
| `h1-subscription-page.spec.ts` | `H1 (page)` | 详情页与浮层里“关注”↔“取消关注”：`POST …/subscribe` 被扣住时停用并显示“Loading”（`sentHeld`），放开后先 POST 再 `GET /api/v0/issues/{id}`（R13），成功提示，库里的行同 API 故事（`subscriptionRows`）；取消同样。命令面板 Shift+S。访客（项目开着“访客看全部”）在别人的工作项上能关注（R14）。只看自己的访客在自己的工作项上能关注：访客不能建工作项（`server/internal/modules/access/domain/rules.go:133`），所以那一项是他作为成员时建的，之后被降为访客。已归档的工作项没有按钮。页面开着时被移出项目，点下去答 `issue.not_found`，提示、没有行。只有 `/api/v0` 的请求。 |
| `h2-h4-notification-cards-page.spec.ts` | `H2–H4 (page)` | §2.6 表中每种变化的卡片文字、执行者的名字与头像、编号与名字是快照、时间。<br>描述的修改与迭代、模块的变化没有卡片。<br>提及只在“提及”里，并且代替关注的那一张。<br>另一端看不到时不点名，看得到时 “WEB-n 名字”。<br>删除工作项后它的卡片不再列出。离开项目之后、刷新之前先点一次那个项目的一张未读卡片：标已读答 `notification.not_found`（`NY:170-176`），项目读也被拒，提示，这一行消失，预览回到没有选中的样子、不是空白（I4）；刷新之后那个项目的卡片不再列出。<br>摘录是纯文本。<br>收集箱条目的通知（负责人在提交时被指派）以普通浮层打开，没有分诊界面，页面安静（R12）。 |
| `h2-h4-auto-archive-page.spec.ts` | `H2–H4 (page)` | 自动归档的执行者显示 “Nerve”（R24）。<br>经 `clockTest` 的 `clockWorld`（`e2e/fixtures/test.ts:103`；先例 `stories/issue/i16-auto-archive.spec.ts:189`）。推进 744 小时之后旧令牌失效，以新进程重新登录（`:217-218`），浏览器的上下文指向新进程。<br>归档的时刻在浏览器时钟的 31 天之后，这一格不断言“多久之前”（re-93 m10）。<br>单独成文件，一个文件只用一种 `test`，标题同样合 `grep`（re-93 m13）。 |
| `h5-notification-list-page.spec.ts` | `H5 (page)` | 两个页签与计数；类型筛选与已选的小块、“清除全部”（“提及”里不发 `type`）；只看未读 / 已归档 / 已暂缓（已归档与已暂缓互斥），每种的精确请求（`recordApi`）；每个页签的空态。<br>“已暂缓”的视图里卡片没有“归档”，“已归档”的视图里卡片没有暂缓菜单（R25）。<br>“加载更多”按 `next_cursor` 取第二页：101 条，用一次 `updateIssue` 加 100 个标签造出（每个标签一条 `labels` 通知，P1b §2.3）。<br>刷新回到第一页。<br>直接打开通知页时的精确请求表：`APP`、`WORKSPACE` 与列表的读（m12）。 |
| `h6-notification-states-page.spec.ts` | `H6 (page)` | 每张卡片经悬停与键盘：已读 / 未读、归档 / 取消归档、预设的暂缓 / 取消暂缓（发 `{snoozed_till: null}`）。归档在默认视图里做、取消归档在“已归档”里做；暂缓在默认视图里做、取消暂缓在“已暂缓”里做（R25，不走“已暂缓”里的归档）。各有提示、在当前视图里留下或移出，库里的三个时刻跟着变。<br>自定义暂缓的弹窗：日期选明天（今天未必还有那个时段）、时间 00:30，库里存的是那个时区的 00:30（W1）。这个文件以 `test.use({ timezoneId: "America/Los_Angeles" })` 定一个非 UTC 的时区，期望值按它算（re-93 m11）。<br>经接口暂缓到两秒之后的一条，到期之后刷新，重新出现并计入未读。<br>打开卡片标为已读，并打开嵌入的浮层（`GET /api/v0/issues/{id}`）。<br>“全部”里的全部已读只标全部页签：提及与已归档的仍未读（D5），回答的 `marked` 与库一致。<br>被移出之后改一条答 `notification.not_found`：提示、这一行消失。 |
| `h7-unread-counts-page.spec.ts` | `H7 (page)` | 顶栏的点与两个页签的数。不计已归档与暂缓中的。<br>每次写与全部已读之后跟着变（重取）。<br>第二个工作区有自己的数。访客看到他自己的。<br>每个工作区页面的挂载请求里有未读数一条（直接打开时数；应用内跳转之后只等内容，m12）。 |
| `h20-notification-pages.spec.ts` | `H20 (page)` | <ol><li>换账号：PATCH 或列表被扣住时，另一个标签页登录别的账号（`anotherTabSignsIn`）；旧的回答不在新页面提示、不改数、不加行；库里的行照断言。</li><li>换工作区：A 的列表或计数被扣住时换到 B，A 迟到的回答不进 B 的列表、点与“加载更多”（W3、W5）；回到 A 重新取。</li><li>后退、前进：浮层开着、标为已读的回答被扣住时后退，回答不在新页面提示；前进回来没有残留的选中与浮层（W6）。</li><li>键盘与读屏：页签带数与选中状态；卡片 Enter 打开；动作 Tab 可达且有名字；类型筛选与视图两个 `PopoverMenu` 的项 Tab 可达、Enter 与 Space 各切换一次、读出按下与否，Escape 关上并把焦点还给按钮；暂缓菜单（`CustomMenu`）方向键与 Enter 可用，Escape 还焦点（re-93 I3）；“加载更多”是按钮；`expectOutlinesWhole`。</li><li>暂缓弹窗：有名字；焦点在里面；后面的页面 inert；Escape 与“取消”关上并还焦点；在途时关不掉；必填的提示显示并作描述；`CustomSelect` 的键盘与列表在按钮旁（`expectListBesideButton`）。</li><li>嵌入浮层的复制链接在剪贴板被拒时提示失败（`refuseClipboardWrites`）。</li><li>全程只有 `/api/v0`、控制台安静。</li></ol>文件过 400 行时按关注点拆成两个，标题同为 `H20 (page)`。 |

API 故事 `h1-subscription.spec.ts`、`h2-h4-notifications.spec.ts`、`h5-h7-notification-inbox.spec.ts` 不改，只有 `h1` 的 `subscriptionRows` 改为导入。

**时长**：
- 每个新文件先 chromium 聚焦跑，再三种引擎，再 `--repeat-each=3`，台账记下各引擎的时长。
- 先例的上限是本机 chromium 每个文件约 8 秒（M6/P7c，`docs/v0/M6-cycles-modules/M6-design.md:247`）。H20 与自动归档一格会超出，超出的写明原因。
- 推送检查点 2 之前，按“基线 main CI 44989539530 的 e2e 两片 + 新文件 chromium 时长 × CI 与本机之比”估出两片的时长。估出超过 30 分钟（设计 §9 `:258`）先报告，不推送。

### 2.12 文件与职责

| 路径 | 职责 |
|---|---|
| `C/services/notification.service.ts`、`C/services/issue/issue-detail.service.ts` | §2.1 |
| `C/store/notification/`（`notification.store.ts`、`notification-changes.ts`、`filters.ts` 与测试、替身 `fake-notifications.ts`） | §2.2、§2.3 |
| `C/store/issue/issue-details/subscription.store.ts`、`root.store.ts`（接线约 6 行，249 行） | §2.3 |
| `C/lib/reconciled.ts`（207 行，加约 10 行）与 `reconciled.test.ts` | §2.4 |
| `C/store/root.store.ts`、`root.store.test.ts` | 接线、store 数 |
| `C/hooks/store/use-notifications.ts`、`C/hooks/use-unread-notification-count.ts`、`C/hooks/use-actor-name.ts` | §2.3、§2.4、R22 |
| `A/notifications/{layout,page}.tsx`、`WN/**` | §2.5–§2.7 |
| `C/components/navigation/top-navigation-root.tsx`、`C/components/sidebar/sidebar-item.tsx` | 顶栏 |
| `C/components/issues/issue-detail/{subscription-button.tsx,issue-detail-quick-actions.tsx}`、`peek-overview/header.tsx`（211 行，加约 5 行）、`power-k/ui/pages/context-based/work-item/{subscription-command.ts,commands.ts}` | 关注 |
| `C/components/issues/issue-detail/issue-activity/activity/actions/helpers/issue-user.tsx` | 改读 `useActorName`，行为不变（R22） |
| `C/components/issues/issue-layouts/roots/{project-layout-root,planning-page-layout-root}.tsx` | §2.8 |
| `C/components/issues/issue-detail/issue-activity/activity/activity-list.tsx`（93 行，删两行） | §2.10 的 `no-empty-pattern` |
| `web/packages/utils/src/{datetime,string,string.test}.ts` | §2.9 的三个包导出 |
| `web/packages/{types,utils,constants}/src/…`、`web/packages/i18n/src/locales/{en,zh-CN}/` | §2.9、§2.10 |
| `K`、`.oxlintrc.json`、`web/apps/web/package.json` | §2.10 |
| `e2e/` | §2.11 |
| `docs/` | P3d 回填（T1）、plane-diff、frontend-changes、输入交接的“处理结果（M7/P4）”、设计 §12/§14、阶段记录 |

**接近 400 行的文件**：
- 不改、不加长：`e2e/stories/smoke/s2-web-app.spec.ts` 582、`C/components/inbox/content/inbox-issue-header.tsx` 470、`e2e/fixtures/cycle-pages.ts` 388、`…/work-item/module-command.test.tsx` 393、`cycle-command.test.tsx` 387、`C/components/issues/peek-overview/view.tsx` 303、`C/store/issue/issue-writes.ts` 302。
- `…/work-item/commands.test.tsx` 279：关注命令另写 `subscription-command.test.tsx`。
- 新文件都在 400 行以内，按关注点分（§2.6 的文件表）。最可能接近的是 `h20-notification-pages.spec.ts` 与 `h2-h4-notification-cards-page.spec.ts`（要盖 §2.6 表的每一行），过 400 行就按关注点拆，标题照旧、文件名仍合 `testMatch`（re-93 m13）。

## 3. 局部裁定

主控的裁定（台账 `progress.md`“主控裁定”）：WP-1 到 WP-18 都取 (a)；关注的写照 WP-12，不照 PP-11；W1–W8 都修、都登记。下面逐条写理由与代价。

- **R1 范围与 P3d 的回填**：§1。关键词 phase 改为 `M7/P4`。
  - T1 补 P3d 阶段记录 §9 与状态行，数字取自归档台账的末行（`.superpowers/sdd/M7-P3d-integration-20261122/payload/worktree/.superpowers/sdd/P3d-recents/progress.md`）：
    - 限定复审 Ready to merge、没有发现，观察 O1 由主控在 `abe2f209` 补测试（写或裁剪失败时记录随之失败，两个变异皆杀），O2 记为不改；
    - 终检 `final.sh fin` 全过，bootstrap 第 0 路 471.2、第 1 路 660.5 秒，e2e 494 passed；
    - 最终分支 CI 44987439140 全绿：server (0) Test 673 / bootstrap 493.0，server (1) Test 808 / bootstrap 786.8，期限的 65.6%；
    - noff `3d9293bd`（父 `9d26b1c3`、`abe2f209`）；
    - main CI 44989539530 全绿：server (0) Test 692 / 498.9，server (1) Test 757 / 736.3，61.4%；
    - 归档 `.superpowers/sdd/M7-P3d-integration-20261122`，3266 个文件、未知 0，清理之后核过；
    - 清理完成（`cleanup.log`）。
  - 设计 §14 的 P3d 一行改为已完成，加 P4 一行。
  - 理由：P3d 的阶段记录在合并之前提交，集成的数字只能由下一段补（P3d 的 R1 同一做法）。
- **R2 新的数据层，旧的整套删除（WP-1）**：
  - 新 store、service、hook、纯函数，类型只来自 `@nerve/api-client`。旧 store、service、hook、`TNotification` 等手写类型、预览 hook 与工具一并删除（§2.9）。
  - 理由：
    - 总体设计 §7.2 不设转换层；
    - 旧 store 读 router store 的 slug（`workspace-notifications.store.ts:162`、`:235`、`:251`、`:266`），本地做计数的算术（`:287-308`），按 Plane 的字段分页签（`:127-131`），处处违反 §7.7；
    - 旧 service 是模块级单例（`:116`）。
  - **新旧并存的窗口**：新 store 在 T2 进入 `RootStore`（顶栏是它的第一个读者），旧 store 留到 T5。T5 新写的页面替换 `WN/` 的整棵旧树，旧树、旧 store、旧 hook 在同一提交删除：它们一同失去读者，knip 会一同报出。并存只在本分支上，先例 P6a R2；中间的提交里通知页仍是坏的，与 main 现状相同（第一次请求就 404）。
- **R3 页签、筛选、选中是页面状态（WP-2）**：放在通知布局提供的 context 里，store 只存 nerve 的数据（工作区 id 与筛选键）。
  - 理由：M3 交接 B（`HO/M3-closeout.md:30`）与 v0 §7.7 的最后一条，界面状态在以工作区为 key 的布局之下。旧 store 把它们放在跨工作区的 store 里（`workspace-notifications.store.ts:72-84`），离开页面后选中还在（W6）。
  - 代价：离开通知页筛选复位（Plane 在同一会话里保留，重新载入复位）。登记差异。
- **R4 `ReconciledByKey.fetchPart`（WP-3）**：扩展这份实现，不复制（v0 §7.7 `docs/v0/v0-design.md:639`，`Reconciled.fetchPart` 已在 `C/lib/reconciled.ts:85-107`）。单测见 §2.4。
- **R5 未读数在顶栏，每个工作区页面取（WP-4）**：
  - 键带工作区 id 与 slug，修 Plane 的 W3（Plane 的键 `"WORKSPACE_UNREAD_NOTIFICATION_COUNT"` 不带工作区，`pw/core/components/workspace-notifications/notification-app-sidebar-option.tsx:20-45`）。
  - 顶栏的点与通知页页签的数共用一个 hook、一个键，取数门一处（re-65 M6 的教训）。
  - H7 的“侧栏”是通知页侧栏的页签数：Plane 的应用侧栏不渲染收件箱。
    - 它的数在 `SidebarUserMenuItem`（`pw/core/components/workspace/sidebar/user-menu-item.tsx:64`）里，而 `SidebarUserMenu`（`user-menu.tsx:19`）在 Plane 里没有导入者；
    - 静态入口 `inbox` 定义了（`plane/packages/constants/src/workspace.ts:238-244`），渲染的名单只有 `home`（`:276-278`）。
  - 代价：每个工作区页面多一次读。`mounts.ts` 的 `WORKSPACE` 与六个故事随之改（§2.11），T2 的检查点 0 用整套 e2e 与分支 CI 核这个共用的改动。
- **R6 计数何时重取；不在聚焦时重取（WP-5）**：每次本人确认的写之后、刷新时、通知页挂载时（同一个键的挂载读）重取。
  - 理由：
    - 计数只有服务端算得对（页签、已归档、暂缓中、可见），Plane 的本地算术错（D2、D5，`pw/core/store/notifications/notification.ts:194-238`）；
    - v0 §7.7 规定会话取数只有一份 SWR 配置、聚焦时不取（`docs/v0/v0-design.md:628`，`C/lib/use-session-swr.ts:25`）。为这个键开例外是改设计规则，是负责人的事，本段照规则走。
  - 代价：别人让你收到新通知，点不会自己亮。要等换工作区、进通知页、刷新或自己的写。Plane 聚焦时重取（`plane/packages/constants/src/swr.ts:16`）。登记为与 Plane 的差异；§6 列为给负责人看的一项，可选做法 (c)：为这个键开聚焦重取的例外。不等。
- **R7 刷新 = 重读第一页（WP-6）**：丢掉后续页，重取计数。
  - 理由：迟到的通知可能落在已读过的页上，第一页的游标看不出（`NY:22-26`，P1c `specs/P1c-notification-api.md:146`）。Plane 把第一页并进旧行，别处归档的行一直留着（W4）。
- **R8 自己 PATCH 之后的行（WP-7）**：
  - 写进持有的每一份。不再相符于 `archived`、`snoozed` 的移出（`matches`）。读状态的变化留在原处：在“只看未读”里打开一张未读的，它不从指针下跳走；Plane 留着，`workspace-notifications.store.ts:151` 的过滤被注释掉。
  - `notification.not_found` 处处移出并提示。
  - 代价：“只看未读”里刚读过的那张留到下次取数。
- **R9 全部已读（WP-8）**：请求体取页面状态；之后重读当前列表的第一页与计数。
  - 理由：服务端的时刻与可见决定标哪些（P1c §2.7）。Plane 把加载过的每一行都在本地标成已读，并把一个页签的数清零（D5）。
- **R10 “提及”里隐藏类型筛选、从不发送（WP-9）**：`tab=mentions` 带 `type` 答 422（`NY:26-29`）。“全部”里的选择保留。
- **R11 卡片的文字（WP-10）**：纯函数 `describeChange` 与新键 `notification.card.*`（§2.6 的表），逐行单测。
  - 理由：Plane 的映射按它自己的字段名，渲染出断句：
    - 创建是 “Alice the work item and assigned it to you.”；
    - 附件是 “Alice to the work item.”；
    - 关联是 “…related to to .”；
    - 删评论是 “commented”。

    并且全是英文字面量（`pw/.../notification-card/content.tsx:46-117`；W2）。
  - 关闭 P1b（`specs/P1b-notification-core.md:120`、`:142`）、P1c（`specs/P1c-notification-api.md:146-147`）与 M5-M7-03 的卡片一项。
- **R12 预览（WP-11）**：`IssuePeekOverview` 嵌入。项目不在手时 `useProjectRead`。收集箱条目以普通浮层打开。
  - 理由：设计 `:242`；只在需要时多一次读；关闭 M3 交接 §2 `:28` 的裸 `useSWR`。
  - 浮层渲染不出来时都有交代：项目读在路上显示占位，被拒时清选中并提示，选中的行被移出时清选中（§2.6，re-93 I4）。
  - 代价：条目在普通浮层里显示隐藏的 Triage 状态的 id，每个写答 `issue.in_triage`（`C/lib/error-messages.ts:87`），到 P5 为止，登记。
- **R13 关注的写不乐观（WP-12，不照 PP-11）**：在途时停用并显示“Loading”。经工作项的队列 POST，之后 `refreshAuthoritative` 取 `is_subscribed`（反应的先例）。
  - 理由：在本地以 `confirmChange` 设值要手造一个 `after`，它经 `canonical.receive` 与 `choose`（`C/store/issue/issue.store.ts:28-38`、`:60-77`）。与在途的详情读交错时，`choose` 可能整份留下任一边，旧的 `is_subscribed` 或旧的别的字段会赢。重读走已有的路径。
  - 代价：每次关注多一个 `GET /issues/{id}`。
- **R14 谁看得到关注、在哪里（WP-13）**：详情页头、浮层页头、命令面板（Shift+S）。在项目里有角色且持有详情的人看得到，**访客也是**；已归档的工作项或项目上隐藏。
  - 理由：设计 §3.10（`:142`，判定用 `issue.read`）与 H1 的“读得到才能关注”。Plane 的后端也让任何项目成员关注（`plane/apps/api/plane/app/views/issue/subscriber.py:22-28`），只是网页给访客停用（`pw/core/components/issues/issue-detail/subscription.tsx:45-50`、`:89`）。
  - 已归档的照 Plane 隐藏（`pw/core/components/issues/issue-detail/issue-detail-quick-actions.tsx:146-148`、`peek-overview/header.tsx:211-213` 的条件），虽然接口允许。
  - 登记差异（访客）。
- **R15 暂缓（WP-14）**：
  - 预设经键盘可用的菜单。
  - 自定义弹窗用日期加一个半小时一格的时间选择：值 24 小时制、按语言标注，修 W1；今天只给将来的时段。
  - `FieldError` + `useRefusedFocus`；服务端的 `must_be_future` 显示在时间一栏。
  - 取消暂缓发 `snoozed_till: null`。
  - 理由：一步关掉 M3 交接 §3 的两项（`:44`、`:46`），保留 `CustomSelect` 以完成 M2 交接 §3 的核对。
  - 代价：没有 AM/PM 的切换（en 的时段里直接写 AM/PM）；登记。
- **R16 可访问的形状（WP-15）**：
  - 卡片一个主按钮加同级的动作按钮，按行命名，悬停与 `focus-within` 时显示。
  - 页签用 Headless `Tab`，计数进名字。
  - 两个菜单是 `PopoverMenu`，不嵌套按钮（修 W7，M3 交接 `:39`）。
  - 每个失败都提示（修 W8）。
  - 理由：H20；M4 交接 M7-05（`HO/M4-closeout.md:86`）。
- **R17 收起“保存为视图”（WP-16）**：只去掉两个 `enableSaveView`。
  - 理由：设计 `:241` 只点名这两处，P6a 要“接上 `enableUpdateView`”（设计 `:244`），所以 HOC 的机器要留着；它按 Plane 的形状写（`project-level.tsx:13-14`、`:62-63`、`:104-121`），P6a 照生成的 `ProjectView` 重写。
  - 代价：从 P4 到 P6a/P6b，`enableSaveView`、`enableUpdateView` 与工作区 HOC 的分支没有传入者。照 M1 交接死 prop 一条的“写进本 M 的 review”登记在阶段记录，由 P6a/P6b 关闭。
- **R18 关键词与告警（WP-17）**：§2.10。上限在清掉告警的同一提交下调（re-64 H1 的教训）。
- **R19 页面配置与终检（WP-18）**：
  - 新的 `playwright.m7-pages.config.ts`（PP-21）。
  - 终检另跑 `m6-pages` 与 `m6-acceptance`（PP-22）：它们断言共用的外壳与 `WORKSPACE`，P4 改了顶栏、详情与浮层的页头、筛选行、挂载清单。
  - 检查点 1 另跑一次 `m4-detail`：详情、浮层页头与 S2。
  - CI 不变。
- **R20 Plane 网页缺陷 W1–W8 都修、都登记**：
  - W1 12 AM/PM 颠倒 → R15；
  - W2 卡片文字 → R11；
  - W3 计数的键不带工作区 → R5；
  - W4 刷新并入旧行 → R7；
  - W5 迟到的列表回答覆盖别的工作区的分页 → 按工作区 id 存（§2.3）；
  - W6 离开后选中还在 → R3；
  - W7 带点击的 `div`、没有名字的按钮与弹窗 → R16；
  - W8 失败只 `console.error` → R16。

  证据见 `research/plane-pages.md` §1.10。T15 逐条写进 plane-diff。
- **R21 每页 100 条（PP-1）**：Plane 300，nerve 只收 1–100（`NY:32-33`，T3）。取上限 100，与工作项列表的 `limit=100` 一致。“加载更多”的故事因此要 101 条（§2.11 的造法）。
- **R22 执行者的显示一处判定**：新 hook `useActorName(actorId)` 回答显示的名字与是否成员。规则照 `IssueUser`（`…/activity/actions/helpers/issue-user.tsx:20-25`）：
  - `null` 是 nerve 自己，显示 `SITE_NAME`（`web/packages/constants/src/metadata.ts:7`）；
  - 找不到的成员显示 `common.deactivated_user`；
  - 否则显示 `display_name`，经 `getUserDetails`（`C/store/member/index.ts:50`，`listWorkspaceMembers` 含已结束的成员）。

  `IssueUser` 改读它，行为不变。理由：同一条规则不写两份。代价：动一个 M4 的文件，它没有单测，本段给 hook 写单测。
- **R23 选中即打开，标已读排队**：按下卡片时立刻选中并打开预览，`{read: true}` 在 store 的队列里发。Plane 先等标已读的回答再打开（`pw/.../item.tsx:47-67`）。
  - 理由：浮层的 `getIssue` 与 PATCH 互不依赖。
  - 代价：无可见差异。
- **R24 自动归档的 “Nerve”**：H2 的页面证据单独成文件 `h2-h4-auto-archive-page.spec.ts`，用 `clockTest` 的 `clockWorld` 推进时钟、触发自动归档，推进之后重新登录，不断言“多久之前”（§2.11，re-93 m10、m13）。T12 记它在三种引擎的时长，单引擎超过 30 秒先报主控。执行者为空的显示另有 `useActorName` 与 `describeChange` 的单测。
- **R25 卡片的动作随视图收窄（主控裁定，re-93 I1）**：“已暂缓”的视图里卡片不给“归档”、给“取消暂缓”；“已归档”的视图里不给“暂缓”、给“取消归档”（`viewOffers`，§2.2、§2.6）。
  - 理由：服务端的 `archived`、`snoozed` 两个筛选是相与的等式（`server/internal/modules/notification/adapter/postgres/queries/inbox.sql:19-20`，`NY:25-26`），视图的两个开关照 Plane 互斥，页面永远发不出 `archived=true&snoozed=true`。在“已暂缓”里归档一张，它不再合 `archived=false`、被 `matches` 移出，“已归档”里也没有它（那里发 `snoozed=false`）；在“已归档”里暂缓同理。这条通知从此在哪个视图里都看不到，取消不了暂缓、也取消不了归档，直到暂缓到期。收窄动作之后页面造不出这种状态。Plane 没有这个洞：它的后端 `snoozed=false` 的方向是反的（D1），已归档的视图照样列出暂缓中的；洞是“改对方向 + 照搬互斥”合出来的。
  - 不改服务端与契约：`archived=true` 是否也列暂缓中的，是 P1c 已合入的合同的语义，跨模块，不是 P4 的事。
  - 代价：经接口造出的“既归档又在暂缓中”的通知，在网页里直到暂缓到期都看不到（到期之后出现在“已归档”里）。列为已知的缺口（§6），进主控的 M7 收官报告给负责人看；T15 写进 plane-diff。

本段的裁定都在已批准的设计与先例之内，不改接口、服务端，不跨后端模块。**没有须负责人决定才能动工的事**；R6 与 R25 的缺口是给负责人看的两项（§6）。

## 4. 验收标准

- [ ] **A1 数据层**：
  - service 只调 `/api/v0`；
  - store 按会话分代：客户端来自这一代，取数经 `useSessionSWR` 且键带 `loginId`，按工作区 id 存，写经 `oneAtATime`、确认写进持有的每一份；
  - `ReconciledByKey.fetchPart`；
  - 页面状态不在 store；
  - §2.2 的纯函数与陷阱 T1–T8 各有单测；
  - 每个门、迟到回答、拒绝、去重与重放的性质各有能让测试失败的变异。
- [ ] **A2 通知页**：
  - 页签、筛选、卡片、加载更多、刷新、全部已读、卡片动作、暂缓（预设、自定义、取消）、预览照 §2.5–§2.7；
  - 被拒时留着并提示；预览渲染不出来时有占位或提示（I4）；
  - 卡片的动作随视图收窄（R25）；
  - W1–W8 修正。
- [ ] **A3 未读数**：顶栏的点与页签的数共用一个取数；不计已归档与暂缓中的；写后重取；`WORKSPACE` 与六个故事跟随。
- [ ] **A4 关注**：详情、浮层、命令面板；访客可用；已归档的隐藏；不乐观、重读取 `is_subscribed`；被拒提示。
- [ ] **A5 页面故事**：H1、H2–H4、H5、H6、H7、H20（通知页）在三种引擎通过，`--repeat-each=3` 过；m6-pages、m6-acceptance、m4-detail 不回归。
- [ ] **A6 删除与守卫**：
  - §2.9 全部删除，含三个 `@nerve/utils` 导出中的两个与 `string.test.ts` 的用例；死成员前后整份比对；`old-notification-urls`；`workspace-address-route` 收入通知的文件；
  - swr 名单；上限 31；`no-empty-pattern` 为 error，`activity-list.tsx` 的空解构删掉；
  - 收起“保存为视图”；
  - 死成员的基线与结论；
  - `root.store.test.ts` 21 个 store。
- [ ] **A7 文档**：
  - plane-diff（R3、R6、R8、R14、R15、R20、R25 与 D1、D2、D5 的网页一侧）；
  - frontend-changes：3.1 的 M7 一行；3.2 加通知的数据层、通知页、未读数、关注、收起各一行；“挂载时的取数”一行补顶栏的未读数；
  - 输入交接的“处理结果（M7/P4）”（§5 的表）；
  - 设计 §12、§14；P3d 的回填（T1）。
- [ ] **A8 门禁、审查与集成**：七项本地门禁加三个页面配置（`final.sh`）、整分支审查与修复复审、准确的分支 CI 与 noff 之后的 main CI 全绿、归档清理。由下一段的 T1 勾选。

## 5. 交接与后续责任

**本段处理的输入交接**：都写“处理结果（M7/P4）”，照设计 §11 保持 open 至收尾，除非下表写关闭。T15 写。

| 交接 | 条款 | 本段 | Task |
|---|---|---|---|
| `HO/M1-P2-trim-content.md:14`、`:18` | 通知没有已删除功能的类型；全部已读的请求体 | 推进：卡片只渲染契约的字段（§2.6 的表）；全部已读发 `NotificationMarkAllRead`。首页的最近访问随 P7 | T6、T7 |
| `HO/M1-P3-trim-platform.md:11`、`:15` | 通知没有工时 | 推进：`NotificationChange` 没有这个字段；收藏的部分随 P7 | T6 |
| `HO/M1-closeout.md:14-22`、`:27-30` | 死成员与死 prop；改到的文件告警为零、清一类 | 推进：基线、本段改到的文件为零、上限 40 → 31、`no-empty-pattern` 为 error | T2、T5、T15 |
| `HO/M2-closeout.md:29-31` | 两个模块级单例；新键带 `loginId` | 推进：通知的单例删除；新键经 `useSessionSWR`。收集箱的随 P5 | T5 |
| `HO/M2-closeout.md:40-42` | `CustomSelect` 在暂缓弹窗里 | 暂缓弹窗的部分在 H20 (page) 的浏览器核对之后写明；视图页头的随 P6a | T9、T14 |
| `HO/M2-closeout.md:61-64` | CSP `frame-src` | 写明：通知页不嵌 iframe | T15 |
| `HO/M3-closeout.md:14`、`:22` | 未读通知数随接口加回、按权限启用，S2 的清单随之改 | 未读数一项关闭 | T2 |
| `HO/M3-closeout.md:28` | 通知页的收集箱项经裸 `useSWR` 读项目 | 关闭（R12） | T5 |
| `HO/M3-closeout.md:29` | `workspace-notifications.ts` 的 17 行死成员 | 关闭（文件删除） | T5 |
| `HO/M3-closeout.md:30` | 弹窗与选中在以 id 为 key 的布局之下 | 通知的部分关闭（R3） | T5 |
| `HO/M3-closeout.md:36`、`:37` | 通知的打盹弹窗的 Tab、Escape、点外面、焦点、名字 | 暂缓弹窗的部分关闭 | T9、T14 |
| `HO/M3-closeout.md:39` | 两个通知菜单的嵌套按钮；页头的通知链接没有名字 | 通知的三处关闭；视图的随 P6a | T2、T7 |
| `HO/M3-closeout.md:42` | `FiltersDropdown` 的按钮；M7 的 7 个调用方 | 本段不涉及：通知页用 `PopoverMenu`，不用 `FiltersDropdown`（re-93 m17） | T15 |
| `HO/M3-closeout.md:44`、`:46` | 暂缓的两条规则的提示；AM/PM 两个 `div` | 关闭（R15） | T9 |
| `HO/M3-closeout.md:53` | 通知页与 `WN/root.tsx` 改读 `useRouteWorkspace()`、加进规则 | 通知的两个关闭；其余六个随 P6b、P7 | T5 |
| `HO/M3-closeout.md:57` | 暂缓弹窗 5 条、筛选菜单 1 条告警 | 关闭：5 → 0；筛选菜单的那条已由 M4/P16 `c4881cf5` 清掉 | T5 |
| `HO/M3-closeout.md:59` | 每个阶段开始时改 phase | `M7/P4` | T1 |
| `HO/M4-closeout.md:86`（M7-05） | 新页面的弹窗名、字段提示、Escape、行名控件、角色与翻译 | 通知页的部分推进 | T5–T9 |
| `HO/M5-closeout.md:43`、`:47`（M5-M7-03） | 通知卡片的正文与图片 | 通知的部分关闭（摘录为纯文本）；视图的部分已在 P3b 写明（`:49`）。本条关闭 | T6 |
| `HO/M5-closeout.md:38`（M5-M7-02） | 附件通知“P4 的卡片若要链到文件再议” | 写明：不链 | T6 |
| P1a spec `:84`、P1b spec `:120`、`:142`、P1c spec `:146-147`、P3b spec `:9`、设计 `:240`、`:241` | 页面、卡片、单例、`TNotification`、`old-notification-urls`、收起 | 关闭各自的通知部分 | T2–T11 |

`HO/M6-closeout.md` 没有以 P4 为关闭阶段的条目（`research/web.md` §10），不写。

**后续**：
- **P5**：收集箱的嵌入分诊（通知预览里，以 `IssueDetail.intake` 或快照的 `state_group === "triage"` 判断，设计 §3.6 `:108`）；收集箱组件上通知的两个 prop 接回；收集箱的单例。
  - **处理结果（M7/P5a）：** 收集箱的单例删除：`inbox/content/issue-root.tsx:36` 的版本 service 在 T6a，`store/inbox/project-inbox.store.ts:127` 的 `InboxIssueService` 在 T6c（[P5a spec](P5a-web-intake.md) §2.10）。嵌入分诊与两个 prop 随 P5b（R32、Q13）。
  - **处理结果（M7/P5b）：** 嵌入分诊已做：预览读工作项之后按它的 `intake` 判定（不读通知的字段或快照的 `state_group`），未被接受的嵌入收集箱的详情与页头，已接受的开浮层；两个 prop 以选中的端口回来，`isNotificationEmbed` ↔ `selection.kind === "embed"`，`embedRemoveCurrentNotification` ↔ `selection.close`（[P5b spec](P5b-web-intake-files-embed.md) §2.5、R18）。本条的收集箱部分关闭。
- **P6a/P6b**：`enableSaveView`、`enableUpdateView` 与工作区 HOC 的分支接回或删除；`rich-filters/filters-row.tsx` 的字面量；视图的嵌套按钮与复制。
- **P7**：通知页不记最近访问（打开卡片不是访问工作项详情页）。
- **P8**：`PlanningReadFailure` 已是通知与规划共用的件，名字的去留随最终的收敛；顶栏未读数是否聚焦重取若负责人选 (c)，在 P8 之前任一页面阶段改 `useSessionSWR` 与 v0 §7.7。

## 6. 风险

- **给负责人看的一项：未读的点不是实时的**（R6）：
  - v0 §7.7 不让会话取数在聚焦时重取，顶栏在一个工作区里也不重新挂载，所以点只在进入工作区、进通知页、刷新与自己的写之后更新。Plane 聚焦时重取。
  - 本段照规则走、登记差异，P4 报告时向负责人提一句。负责人若要实时的点，可选 (c)：为这一个键开聚焦重取的例外，改 v0 §7.7 一句。
  - 不等这个决定。
- **已知的缺口：既归档又在暂缓中的通知**（R25，主控裁定）：页面不再造出它，但经接口造出的那一条在网页的每个视图里都看不到，直到暂缓到期（之后出现在“已归档”里）。是否让 `archived=true` 也列暂缓中的，是 P1c 合同的语义、跨模块，本段不改。进主控的 M7 收官报告给负责人看；T15 写进 plane-diff。
- **共用的挂载清单**：
  - `WORKSPACE` 一变，所有断言精确请求表的页面故事跟着变。读 `mounts.ts` 的是六个文件。
  - 别的页面故事都按条件过滤 `watch.apiRequests`（`grep` 过 `e2e/stories`：例如 `a9-preferences.spec.ts:100`、`p2-visibility.spec.ts:155`），不受影响。
  - 检查点 0 以整套默认 e2e、m6-pages 与分支 CI 确认。
- **大段的删除与新写**：T5 新写页面的骨架并删除约 2,700 行旧代码。knip 与 `old-notification-urls` 兜底；它是唯一的大 Task，在 plan 里分步。
- **收集箱条目在普通浮层里**（R12）：状态一栏是隐藏的 Triage 状态的 id，写都答 `issue.in_triage`，直到 P5。`research/web.md` §13 第 3 条：状态下拉框怎样显示一个不在目录里的 id 没有核实：T12 的 H2–H4 (page) 有一格打开收集箱条目的通知，断言是普通浮层、页面安静，台账记状态一栏的样子。
- **已归档项目的预览**：项目不在包装层的列表里时，浮层要等 `useProjectRead` 才渲染；不读则什么都不打开（`research/web.md` §9 风险 1）。读在路上显示占位，读被拒清选中并提示（I4）。T5 有单测，H6 (page) 与 H2–H4 (page) 各有一格。
- **网页 CI 的时长**：web 作业 20 分钟（负责人 2026-11-17）。P4 是 M7 第一个加大量 vitest 的阶段，每个检查点记 web 作业的时长，超过 16 分钟（八成）先报告。
- **e2e 的时长**：
  - 七个页面故事文件在 CI 的 chromium 各跑一次，每片 35 分钟期限、30 分钟报告线（设计 §9 `:258`）；
  - H20 与自动归档的 `clockWorld` 一格最慢；
  - 估算见 §2.11。
- **e2e 的已知间歇**：F11 page 的 `net::ERR_ABORTED`（M6-M8-07）只重跑那个文件一次并记录；53300 “too many clients”（M6-M8-03 已在 P1b 关闭）再出现时当作新问题查根因。
- **Go 两路**：本段不改 Go，不加顶层测试名，第 1 路的规则只写“不变”。分支 CI 的服务端两路照常核，本机两路因 M6-M8-05 失败时以分支 CI 为准（P1c 起负责人的规则）。
