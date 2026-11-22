# M7 / P4 plan：网页的通知与关注

> spec：[P4 spec](../specs/P4-web-notifications.md)（§2 是合同，§3 的 R1–R25 是裁定）；预检 [re-93](../../../reviews/re-93-M7P4预检.md) 修正后通过（处理见它的 §6）。基线 main `3d9293bd`（P3d 的 noff）。worktree `.codex/worktress/codex/m7-p4`，分支 `codex/m7-p4`。台账 `.superpowers/sdd/P4-web-notifications/progress.md`（ignored）。实现代理每个 Task 先读同目录的 `implementer-brief.md`。16 个 Task。下文 `C/` = `web/apps/web/core/`，`A/` = `web/apps/web/app/(all)/[workspaceSlug]/(projects)/`，`WN/` = `C/components/workspace-notifications/`，`K` = `tools/keywords.json`。

## Global Constraints

**分支与提交**
- 主目录永远在 main。提交用详细中文，前缀 `feat/fix/docs/test/refactor(M7/P4)`，结尾用派发时会话给出的署名行。提交只在本 Task 的 lint 过了之后，不与它串在一条命令里，用显式路径（`git commit -F <msg> -- <paths>`）。
- 实现代理不推送、不合并；检查点由主控推送。
- 关键词的 phase 由 T1 改为 `M7/P4`；不延长任何 `until`。
- 本段不改 `server/`、`api/`、`deploy/`。超出 spec 的契约或架构改动先停下报告。

**只读与禁区**：`plane/`、`refer/` 只读，不在其内跑 git。不 stash、`git clean`、`reset --hard`、强推。不动别的项目的容器。容器套件一次一个：`make test`、`make test-shard`、`make e2e` 与 Playwright 的配置不并发。

**代码规则**
- 手写文件约 400 行以内，新代码进新文件，按关注点分（spec §2.12 的表）。新文件的版权头 `Copyright (c) 2026-present OpenNerve`，改写的 Plane 文件保留原头。
- 类型只来自 `@nerve/api-client`（只给界面用的形状：`NotificationFilters` 在 `C/store/notification/filters.ts`，页签、类型选项在 `@nerve/constants`）。
- service 不转换；没有模块级的带令牌 service；store 只经自己的 `RootStore` 找兄弟；SWR 只经 `useSessionSWR`；新代码不读 router store 的地址（`useRouteWorkspace()` 或路由参数）。
- 页签、筛选、选中不进 store（spec R3）。
- `await` 之后的提示与跟进一律经 `followInSession`（`C/lib/in-session.ts:40`）。
- 删掉不用的代码，不加开关、桩或兼容别名。新旧 store 只在 T2–T5 并存（spec R2）。
- 不放宽 `K` 的规则、`lint-cap`、knip、`error-messages.test.ts`、i18n 的 `check:sync`。
- **不再加长的文件**：
  - `e2e/stories/smoke/s2-web-app.spec.ts` 582
  - `C/components/inbox/content/inbox-issue-header.tsx` 470（不碰）
  - `e2e/fixtures/cycle-pages.ts` 388
  - `C/components/power-k/ui/pages/context-based/work-item/module-command.test.tsx` 393
  - `cycle-command.test.tsx` 387
  - `C/components/issues/peek-overview/view.tsx` 303
  - `C/store/issue/issue-writes.ts` 302
  - `…/work-item/mutation-commands.ts` 182（不改）

**knip 与第一个读者**（re-93 m7）
- 新文件、新导出在它第一个产品读者出现的 Task 里建立：
  - T2：`NotificationService.unreadCount`、`NotificationStore` 的计数、`use-notifications.ts`、`use-unread-notification-count.ts`，读者是顶栏。
  - T3：`NotificationService.list`；`filters.ts` 的 `NotificationFilters`、`filterKey`、`filtersOf`、`listQuery`，读者是 store 的列表。
  - T4：`NotificationService.update`、`markAllRead`；`filters.ts` 的 `snoozedNow`、`matches`、`markAllBody`，读者是 `notification-changes.ts`。
  - T5：`DEFAULT_FILTERS`（读者是页面状态）、页面状态、新的 `WN/` 骨架、`use-notification-list.ts`，读者是路由。
  - T6：`describe-change.ts`、`use-actor-name.ts`。
  - T7：两个菜单。
  - T8：卡片的动作、预设、`filters.ts` 的 `viewOffers`。
  - T9：时段与弹窗。
  - T10：关注的件。
- **例外，只有一类**：store 的成员先于页面。`getList`、`fetchList`、`fetchMore`（T3）与 `update`（T4）的第一个产品读者是 T5 的页面，`markAllRead`（T4）的是 T7 的页头。它们是 store 的成员、不是导出，同 P6a 的分法（store 先、页面后）。T5、T7 结束时各核一次这些成员都有了产品读者，台账记下。
- knip 把 vitest 文件当入口，只被测试读的导出它查不出；“只给测试读不算读者”由逐 Task 的审查执行。knip 也查不出包的 barrel 导出（spec §2.9），那里靠人对照与死成员的整份比对。
- 删掉某个文件、导出的最后一个读者的 Task，同一提交删去它（T5 一次删整棵旧树、旧数据层与三个包里随之失去读者的导出，spec §2.9）。

**告警上限**：`lint-cap` 要求 oxlint 告警数恰好等于 `web/apps/web/package.json:10` 的上限（现 40）。T5 删掉全部 9 条（spec §2.10），同一提交 40 → 31、`no-empty-pattern` 设为 `"error"`，并删掉 `activity-list.tsx:46-47` 被压住的空解构与它的 disable。之后的新文件不得新增告警，T15 只核对。

**文案**：新文案 en 与 zh-CN 同一提交，不许一个键是另一个键的前缀。`describeChange` 的表写完整的键字面量（`tools/i18n-keys.mjs` 才认得）。不再有读者的旧键只在 T9 删一次：只看 `notification.*`，人工逐个 `grep`，台账列出删了哪些（`tools/i18n-keys.mjs` 查不出定义了没人读的键，re-93 m3）。

**守卫**
- `workspace-address-route` 随 T5 收入 `A/notifications/page.tsx`、`WN/root.tsx` 与新写的读工作区的文件，带样例。
- `old-notification-urls` 在 T5 加：先看它命中旧 service（红的收据，不提交），再与删除同一提交（re-93 m1）。
- `.oxlintrc.json` 的 swr 名单（`:119-215`）随各 Task 收入新的取数文件。
- 每次改 `K` 跑 `node tools/keywords.mjs`。

**允许改的已有测试与夹具**（改之前在台账逐条写明，只照这里写的改法；其余断言是回归，不改；编译跟随只为编译、不改断言，逐处记台账）：
- `C/store/root.store.test.ts:106` 的 store 数：T2 21 → 22，T5 22 → 21。
- `WN/sidebar/filters/menu/menu-option-item.test.tsx`：T5 随它的组件删除。它钉的性质（Enter、Space 各切换一次，不提交外层表单）由 T7 的类型筛选菜单测试接上。**性质的空窗 T5 → T7**，台账记下（空窗里那个组件已不存在，没有产品代码可以回归）。
- `web/packages/utils/src/string.test.ts`：T5 随 `stripAndTruncateHTML` 删它的用例（`:13` 的导入与 `:17-29` 的说明和 `describe`），其余用例不改（re-93 I2）。
- 渲染详情或浮层页头的测试：`C/components/issues/detail-copy.test.tsx`、`archive-issue-modal.test.tsx`、`peek-overview/restore-menu.test.tsx`；以及命令面板的 `…/work-item/commands.test.tsx`、`cycle-command.test.tsx`、`module-command.test.tsx`。T10 编译跟随：替身加 `subscription`，断言不改。`cycle-command.test.tsx:202` 的“快捷键不重”是 T10 的回归保护。
- 不改、不受影响：`C/components/issues/detail-entry.test.tsx`、`body-images.test.tsx`、`issue-detail-widgets/attachments/host.test.tsx`（把页头 mock 掉）；`peek-overview/host-actions.test.tsx`（嵌入浮层的晚关闭保护，回归）；`C/store/issue/issue.store.types.test.ts`（`is_subscribed` 在列表行落下时保留，回归）。
- e2e：
  - `e2e/fixtures/mounts.ts`：T2 改 `WORKSPACE` 与头注释 `:5-14`。读它的六个文件（spec §2.11）只经常量跟随，不改。
  - `e2e/stories/collaboration/h1-subscription.spec.ts`：T12 改为从 `fixtures/assert/notification.ts` 导入 `subscriptionRows`，断言不改。
  - `e2e/package.json`：T12 加 `test:m7-pages`。
- 产品代码里动到的 M4 文件：
  - `…/activity/actions/helpers/issue-user.tsx`（T6，改读 `useActorName`，行为不变，没有单测，T6 给 hook 写）；
  - `C/components/sidebar/sidebar-item.tsx`（T2，加可选 `label`，另两个读者 `help-section/root.tsx`、`user-menu-root.tsx` 不传）；
  - `C/components/issues/issue-detail/issue-activity/activity/activity-list.tsx`（T5，删 `:46-47` 的 `comment: {}` 与它的 `oxlint-disable`，什么也不取的解构，行为不变，re-93 m8）。

**红的窗口**：不预期有，也没有红的提交（re-93 m1）。
- 旧通知页在 main 上本来就坏（第一次请求就 404），T2–T5 之间仍是坏的，没有故事访问它（`grep` 过 `e2e/stories`）。
- `WORKSPACE` 与顶栏的取数在 T2 同一提交，S2 与 G3、G8、G13 不会单独变红。
- T5 的守卫先看红（只留收据），与让它变绿的删除同一提交。
- 名单之外的任何红先停下报告。

**检查点**（七项用 `sh .superpowers/sdd/P4-web-notifications/gates.sh <前缀>`）：
- **0（T2 之后）**：
  - 本机七项，另以 `playwright.m6-pages.config.ts` 三种引擎跑一次（`WORKSPACE` 变了）。
  - 主控推送，核分支 CI。
  - 台账记基线：web 作业的 Lint 与 Unit tests、e2e 两片、Go 两路（以 `gh` 读 main CI 44989539530 的作业时长作对照）。
- **1（T11 之后，产品代码完成）**：本机七项，加 `playwright.m4-detail.config.ts` 与 `m6-pages` 三种引擎各一次，加分支 CI。
- **2（T14 之后，故事完成）**：
  - `final.sh cp2`：七项加 m7-pages、m6-pages、m6-acceptance。
  - 新故事文件另跑 `--repeat-each=3`。
  - 推送之前照 spec §2.11 估出 e2e 两片的时长，估出超过 30 分钟先报告、不推送；之后核分支 CI。
- **终检（T16）**：`final.sh fin`。
- 本机 Go 两路因 M6-M8-05 失败且重跑一次仍不过时，以分支 CI 的服务端两路为准并记录。

**证据**
- 每次检查经 `run-check.py`，变异经 `mutate.py`（每个旧串恰出现一次，工具恢复并核 sha256）；标签 `t<N>-…`。
- 下表每个性质至少一个能编译、能让测试失败的变异。两层都在查同一件事时两层一起改，并留只改一层的对照（硬规则 §5）。只有竞态杀得死的变异不算。
- 页面故事先写一个期望错误的版本（或以 e2e 变异），看到断言失败，再改正（P6a 的写法）。

**测试时间**
- Go 不变，不加顶层名，第 1 路的规则只写“不变”。
- web 作业期限 20 分钟（负责人 2026-11-17），每个检查点记 Lint 与 Unit tests 两步，作业超过 16 分钟先报告。
- e2e 每片 35 分钟、报告线 30 分钟（设计 §9）。T12–T14 记每个新文件在三种引擎的时长，本机 chromium 每个约 8 秒为先例的上限。H20 与时钟一格会超，写明原因；时钟一格单引擎超过 30 秒先报主控（spec R24）。

## 性质与变异

| 性质 | 变异（标签） | Task |
|---|---|---|
| 计数按工作区 id 存、读按地址的 slug | 计数按 slug 存（`t2-mut-count-key-slug`） | T2 |
| 顶栏的点 = `all + mentions > 0` | 只看 `all`（`t2-mut-dot-all-only`）；只看 `mentions` | T2 |
| 顶栏读路由的工作区 | 读 `currentWorkspace`（`t2-mut-top-nav-router`；router 的工作区与路由不同的一格） | T2 |
| 链接有名字 | `label` 不传给 `aria-label` | T2 |
| `ReconciledByKey.fetchPart` | 键没取过时也建条目、发 `read`（`t3-mut-fetchpart-new-entry`）；不委托、另起一个 `Reconciled` | T3 |
| 列表的查询 | “提及”带 `type`（`t3-mut-type-in-mentions`）；空的类型发 `type=`（`t3-mut-empty-type`）；“只看未读”发 `read=true`（`t3-mut-unread-read-true`）；`limit` 300 | T3 |
| 筛选键可逆 | `filtersOf` 丢掉 `unread`（`t3-mut-filtersof-lossy`，往返一格失败） | T3 |
| 刷新替换第一页 | 第一页并进旧行（W4，`t3-mut-refresh-merge`） | T3 |
| 列表按工作区 id | 列表按 slug（W5，`t3-mut-list-key-slug`） | T3 |
| 加载更多去重、取回答的游标 | 合并不去重；`next_cursor` 留旧的 | T3 |
| 全部已读的请求体 | 空的类型发 `type: []`（`t4-mut-empty-type-array`）；“提及”带 `type`；带 `read` | T4 |
| 暂缓按此刻 | `snoozedNow` 不看 `now`（Plane 的 T6，`t4-mut-snoozed-any`） | T4 |
| `matches` 不看读状态 | `matches` 比较 `read`（`t4-mut-matches-read`） | T4 |
| 写一个接一个 | 绕过 `oneAtATime`（`t4-mut-concurrent`） | T4 |
| 确认写进持有的每一份 | 只写当前的一份（`t4-mut-current-list-only`） | T4 |
| 不再相符的移出、读状态留下 | 归档后不移出（`t4-mut-keep-archived`）；读之后移出（`t4-mut-drop-on-read`） | T4 |
| `notification.not_found` | 不移出（`t4-mut-not-found-kept`）；之后不重取计数（`t4-mut-not-found-no-count`）；吞掉拒绝 | T4 |
| 写后重取计数 | 不重取（`t4-mut-no-count-refetch`） | T4 |
| 全部已读不在本地标、成败都重取 | 本地把加载的行标成已读（D5，`t4-mut-mark-all-local`）；不重取第一页；被拒之后不重取（`t4-mut-mark-all-refusal-stale`，re-93 m2） | T4 |
| 拒绝时值不变 | 拒绝时照写 | T4 |
| 页面状态随页面结束 | 状态放在模块级的变量（`t5-mut-state-outlives-page`） | T5 |
| 列表的键带筛选 | `use-notification-list.ts` 的键去掉 `filterKey`（`t5-mut-list-key-no-filter`：换筛选不再取） | T5 |
| 选中跨页签保留、关掉才清 | 换页签时清选中（`t5-mut-selection-cleared-on-tab`） | T5 |
| 打开未读卡片：预览不等标已读 | 预览等 PATCH 的回答才打开（`t5-mut-open-waits-for-read`：PATCH 被扣住或被拒时预览不出现） | T5 |
| 已归档项目的预览 | 不读项目（`t5-mut-no-project-read`） | T5 |
| 预览渲染不出来时有交代 | 项目读被拒时保留选中（`t5-mut-keep-selection-on-read-failure`）；选中的行因 `notification.not_found` 移出时保留选中（`t5-mut-keep-selection-on-not-found`）；项目读在路上不显示占位 | T5 |
| 离开时清浮层 | 不清（W6，`t5-mut-peek-kept`） | T5 |
| 读失败可重试 | 失败时显示占位（`t5-mut-forever-loading`） | T5 |
| 零行而有游标时给“加载更多” | 显示空态、不给“加载更多”（`t5-mut-empty-state-over-more`，re-93 m6） | T5 |
| “Till” 只在暂缓中 | 过去的 `snoozed_till` 也显示（`t5-mut-till-past`） | T5 |
| 打开即标已读 | 已读的也发 PATCH；未读的不发 | T5 |
| 地址只从路由 | 页面标题读 `currentWorkspace`（规则 `workspace-address-route` 命中，加单测一格） | T5 |
| 卡片文字 | 父工作项读 `new_value`（`t6-mut-parent-new-value`）；看不到的另一端点名（`t6-mut-hidden-end-named`）；加入与移出颠倒（`t6-mut-added-removed-swapped`）；空串当作有值（`t6-mut-empty-is-value`：移除的标签与清掉的日期被说成加入、设为空）；日期按时刻解析（`t6-mut-date-as-instant`：`TZ=America/Los_Angeles` 下差一天，re-93 m15）；未知字段抛错 | T6 |
| 执行者 | `null` 显示“已停用的用户”（`t6-mut-actor-null-gone`）；找不到的显示 id | T6 |
| 表里每个键在两种语言里都有 | 删一个 zh-CN 键（测试遍历表） | T6 |
| “提及”里不发、不显示类型 | 全部已读在“提及”里带 `type`（`t7-mut-mentions-sends-type`）；换页签清掉类型（`t7-mut-clear-types-on-tab`） | T7 |
| 已归档与已暂缓互斥 | 两个都开（`t7-mut-archived-snoozed-both`） | T7 |
| 在途停用 | 全部已读在途可再发（`t7-mut-mark-all-twice`） | T7 |
| 菜单项切换一次、不嵌套按钮、项之间 Tab | 加一个 `onKeyDown` 的切换（Enter 切两次，`t7-mut-enter-twice`）；按钮里放 `IconButton` | T7 |
| 迟到的回答不在新会话里跟进 | 全部已读的提示直接写在 `await` 之后、不经 `followInSession`（`t7-mut-no-follow`） | T7 |
| 动作随视图收窄（R25） | “已暂缓”的视图里仍给“归档”（`t8-mut-archive-in-snoozed-view`）；“已归档”的视图里仍给暂缓菜单（`t8-mut-snooze-in-archived-view`） | T8 |
| 取消暂缓发 `null` | 发 `{}` 或 `undefined`（T1，`t8-mut-unsnooze-undefined`） | T8 |
| 动作键盘可达、有名字 | 动作 `hidden` 到悬停为止（`t8-mut-actions-hover-only`）；去掉行名 | T8 |
| 被拒不提示成功 | 失败也提示成功（`t8-mut-success-on-refusal`） | T8 |
| 迟到的回答不在新会话里跟进 | 动作的提示不经 `followInSession`（`t8-mut-no-follow`） | T8 |
| 预设从此刻算 | 从当天零点算 | T8 |
| 时段 24 小时制 | Plane 的 AM/PM 换算（W1，`t9-mut-plane-ampm`） | T9 |
| 今天只给将来的时段 | 给全部（`t9-mut-today-past-slots`） | T9 |
| 在途关不掉、焦点回到按钮 | 在途可关（`t9-mut-close-in-flight`）；关上之后不还焦点 | T9 |
| `must_be_future` 在字段下 | 作为提示（`t9-mut-must-be-future-toast`） | T9 |
| 迟到的回答不在新会话里跟进 | 弹窗的跟进不经 `followInSession`（`t9-mut-no-follow`） | T9 |
| 关注不乐观 | 204 之后在本地设值、不等重读（PP-11，`t10-mut-optimistic`：重读被扣住时标签已翻） | T10 |
| 关注被拒之后仍重读 | 被拒就不重读（`t10-mut-no-reread-on-refusal`） | T10 |
| 访客可关注 | 门要管理员或成员（`t10-mut-member-only`） | T10 |
| 已归档隐藏 | 已归档也显示（`t10-mut-archived-shown`） | T10 |
| 关注经工作项的队列 | 不经 `enqueueIssueChanges`（`t10-mut-no-queue`：同一工作项在途的写之前就发出） | T10 |
| 迟到的回答不在新会话里跟进 | 按钮或命令的提示不经 `followInSession`（`t10-mut-no-follow`） | T10 |
| 收起“保存为视图” | 加回 `enableSaveView`（`t11-mut-save-view-back`） | T11 |
| 页面故事 | 每个文件先跑一个期望错误的版本，看到失败 | T12–T14 |

## Tasks

### T1 基线与文档（主控）

- **提交**：spec、plan、预检 re-93（含 §6 的处理）。
- **P3d 阶段记录**（`docs/v0/M7-collaboration/reviews/P3d-recents-review.md`）：
  - 状态行改为已完成（noff `3d9293bd`，main CI 44989539530 全绿）；
  - §2 加限定复审 O1 一行（`abe2f209`）；
  - §5、§6 的表各加最终分支 CI 与 main CI 两行；
  - §9 末尾补记（spec R1 的数字，取自归档台账的末行）：限定复审（`95c8dbd3..fd0fc6de`，Ready to merge、没有发现，O1 由主控在 `abe2f209` 补测试、两个变异皆杀，O2 记为不改）、终检、最终分支 CI、合并（父 `9d26b1c3`、`abe2f209`）、main CI、归档（3266 个文件、未知 0，清理之后核过）、清理（worktree、本地与远端分支已删，`cleanup.log` 2026-11-22T05:55:14Z）。
- **设计**：§14 的 P3d 一行改为已完成（noff、main CI、阶段记录），加 P4 一行（进行中，re-93，基线 `3d9293bd`，16 个 Task，spec、plan 的链接）。
- **关键词**：`K:2` 的 `phase` 改为 `M7/P4`，跑 `node tools/keywords.mjs`（没有到期的例外：M7 的四条是 `until: "M7/closeout"`）。
- **依赖**：worktree 没有 `node_modules`，在根目录跑一次 `pnpm install --frozen-lockfile`（收据 `t1-install`）。
- **检查**：文档链接（`doc-links.py`）与关键词，然后提交 `docs(M7/P4): …`。

### T2 未读数的取数、顶栏与挂载清单 → **检查点 0**

**Files**：
- 新：`C/services/notification.service.ts`（本 Task 只有 `unreadCount`）与测试；`C/store/notification/notification.store.ts`（本 Task 只有计数）与测试；`C/store/notification/fake-notifications.ts`（测试数据：`notificationOf`、`pageOf`、`countOf`）；`C/hooks/store/use-notifications.ts`；`C/hooks/use-unread-notification-count.ts`；`C/components/navigation/top-navigation-root.test.tsx`。
- 改：`C/store/root.store.ts`（`notification` 与旧的 `workspaceNotification` 并存，spec R2）、`root.store.test.ts`（21 → 22）；`C/components/navigation/top-navigation-root.tsx`；`C/components/sidebar/sidebar-item.tsx`（`label`）；en、zh-CN 的 `notification.a11y.unread_count`；`.oxlintrc.json`（swr 名单）；`e2e/fixtures/mounts.ts`（`WORKSPACE` 加 `GET /api/v0/workspaces/{slug}/notifications/unread-count`，头注释改写）。

**Steps**：
- [ ] Step 0：死成员基线（spec §2.10）。把 P6a 归档的 `dead-members/` 三个脚本复制进台账目录 `dead-members/`，先 `pnpm --filter web exec react-router typegen`，再 `deadsym.mjs --tsv > dead-before.tsv`、`domains.mjs dead-before.tsv --rows M7` 与 `--rows M3`。整份 `dead-before.tsv` 留在台账，通知路径的行另记一段（收据 `t2-dead-before`）。
- [ ] Step 1：先写测试（`@/lib/auth/fake-nerve` 的 `FakeNerve`、`json`、`problem`，`@/store/fake-queue`），看到它们因所测性质失败：
  - service：`unreadCount` 的路径；拒绝成 `ApiError`。
  - store 的计数：
    - 按工作区 id 存，经 slug 读；不在调用者工作区列表里的 slug 答 `undefined`；
    - 两个工作区互不影响；
    - 较旧的取数被超过时不写；
    - `SessionChangedError` 给 `undefined`。
  - 顶栏：
    - 点在 `all > 0` 或 `mentions > 0` 时有，两者都是 0 时没有；
    - 键经路由的工作区（router store 指向别的工作区的一格）；路由的工作区为 `null` 时不取；
    - 链接的名字是 `notification.label`，有未读时带数；
    - `Tooltip` 不再是英文字面量。
- [ ] Step 2：实现到通过；顶栏不再读旧 store。核对 `mounts.ts` 的六个读者只经常量跟随。
- [ ] Step 3：变异照上表 T2 的四行。`make lint-web`、聚焦 vitest、`make knip`、`make test-web`。提交 `feat(M7/P4): 未读数的 service、store 与顶栏的点——每个工作区页面取一次，键带工作区`。
- [ ] Step 4（**检查点 0**）：
  - `gates.sh cp0`；`m6-pages` 三种引擎一次（收据 `cp0-m6-pages`）。
  - 主控推送、核分支 CI，台账记基线（见 Global Constraints）。
  - S2 与 G3、G8、G13 由整套 e2e 盖住；任何一条红先停下，查是不是 `WORKSPACE` 之外的精确请求表。

### T3 `fetchPart`、筛选的键与查询、列表的读

**Files**：`C/lib/reconciled.ts`（`ReconciledByKey.fetchPart`，约 10 行）与 `reconciled.test.ts`（新的用例，既有的不改）；`notification.service.ts` 加 `list` 与测试；新 `C/store/notification/filters.ts`（`NotificationFilters`、`filterKey`、`filtersOf`、`listQuery`）与 `filters.test.ts`；`notification.store.ts` 加列表（`lists`、取过的列表、`getList`、`fetchList`、`fetchMore`；没有 `hasMore`，spec §2.3）与测试。

**Steps**：
- [ ] Step 1：先写测试，看到失败：
  - `fetchPart`：spec §2.4 的五条。
  - service：`list` 的路径与查询串，查询全是字符串、`limit` 是数字。
  - `filters`：
    - `filterKey` 与类型的先后无关，“提及”不含类型；
    - `filtersOf(filterKey(f))` 等于规范化之后的 `f`，每个字段各一格（re-93 m5）；
    - `listQuery` 的 T2–T5 与 R21（“提及”不带 `type`；空的不带；“只看未读”是 `read=false`；`limit` 100；`cursor` 只在给了时带）。
  - store：
    - 第一页按 `${workspaceId}|${filterKey}` 存；
    - `fetchMore` 以持有页的 `next_cursor` 取，接上本页没有的行、`next_cursor` 取回答的；没有 `next_cursor` 时不发请求；
    - 在外的 `fetchMore` 被 `fetchList`（刷新）作废；
    - 刷新之后别处归档的行不在了；
    - A 工作区迟到的回答不进 B 的列表与游标。
- [ ] Step 2：实现到通过。
- [ ] Step 3：变异照上表 T3 各行。门禁同 T2。提交 `feat(M7/P4): ReconciledByKey.fetchPart、通知筛选的键与查询、列表的读——刷新换第一页，加载更多按游标`。

### T4 store 的写

**Files**：新 `C/store/notification/notification-changes.ts`（store 委托它，先例 `module-changes.ts`）、新 `notification.store.changes.test.ts`；`notification.service.ts` 加 `update`、`markAllRead` 与测试；`filters.ts` 加 `snoozedNow`、`matches`、`markAllBody` 与测试。

**Steps**：
- [ ] Step 1：先写测试（`fake-queue.ts` 的 `inTurn`、`sent`、`fetchedWhileChangeIsOut`），看到失败：
  - service：PATCH 的 `{snoozed_till: null}` 原样发出；全部已读的 `type` 是数组。
  - `markAllBody` 的 T8（空的类型不发、“提及”不发、没有 `read`）；`snoozedNow`：过去的、`null`、将来的；`matches`：页签、类型、已归档、暂缓中各一格，读状态不影响。
  - `update` 一个接一个发出，请求体在轮到时算。
  - 回答写进这个工作区持有的每一份列表：默认列表与“只看未读”各持有一份。归档在默认列表里移出；暂缓到将来在默认列表里移出；读在“只看未读”里原地替换。不往新相符的列表里插行，那一份下次取数时带来（spec R8 的代价）。
  - 别的工作区的列表不动。
  - `notification.not_found` 处处移出、重取计数、再交出拒绝；其余拒绝值不变、交出。
  - 每次确认之后重取计数。
  - `markAllRead` 发 `markAllBody`，不在本地改行；成功时重取第一页与计数，回答的 `marked` 交给调用者；**被拒时同样重取第一页与计数**，再交出拒绝（re-93 m2）。
  - 取数在外时确认的修改重放到它的回答上（至少一例）。
  - `SessionChangedError` 不改这一代。
- [ ] Step 2：实现到通过。
- [ ] Step 3：变异照上表 T4 各行。门禁同 T2。提交 `feat(M7/P4): 通知 store 的写——一个接一个，确认写进持有的每一份，写后重取计数`。

### T5 通知页的骨架、预览与旧代码的删除

本段唯一的大 Task（spec R2）：新树替换旧树，旧树、旧数据层、旧常量与三个包里随之失去读者的导出一同删除。**一个提交**：守卫先看红（只留收据，不提交），与让它变绿的删除同一提交（re-93 m1）。

**Files**：
- 新：
  - `WN/page-state.tsx`（含 `DEFAULT_FILTERS` 的第一个读者）与测试；
  - `WN/root.tsx`（改写的预览）与测试；
  - `WN/sidebar/{root,header,tabs,list}.tsx` 与测试；
  - `WN/card/item.tsx`：主按钮、第二行的工作项、时间与 “Till”、未读的点。句子在 T6，动作在 T8。
  - `WN/use-notification-list.ts`：列表的 `useSessionSWR`，fetcher 经 `filtersOf` 从键得到筛选。
  - `WN/index.ts`。
- 留用：`WN/sidebar/{loader,empty-state}.tsx`（`empty-state` 改收 `NotificationTab`）。
- 路由：`A/notifications/{layout,page}.tsx`（改名、页面状态、`useRouteWorkspace`）。
- 删除：spec §2.9 全部，含 `@nerve/utils` 的 `convertToEpoch`、`stripAndTruncateHTML` 与 `string.test.ts` 里它的用例；`getNumberCount` 留用（读者 `WN/sidebar/tabs.tsx`），`CheckDoneOutline`、`RefreshOutline` 是外部依赖 `@makeplane/propel` 的图标，不归本仓删（T7 的页头接着读）；`web/packages/constants/src/notification.ts` 改写（只留页签）；`RootStore` 22 → 21。
- 守卫：`K`（`old-notification-urls`；`workspace-address-route` 加两个文件与新文件）；`.oxlintrc.json`（swr 名单、`no-empty-pattern: "error"`）；`web/apps/web/package.json`（上限 40 → 31）；`activity-list.tsx:46-47`（re-93 m8）。
- 文案：en、zh-CN 的 `notification.card.till`、`notification.card.unread`、`notification.a11y.*`（不叫 “Notifications”）。

**Steps**：
- [ ] Step 1：守卫先看红，不提交。在 `K` 加 `old-notification-urls`（spec §2.10 的模式与样例），看到 `node tools/keywords.mjs` 因旧 service 命中而失败（收据 `t5-red-old-urls`）。`workspace-address-route` 加两个文件，看到它因 `currentWorkspace` 命中而失败（收据 `t5-red-address`）。
- [ ] Step 2：先写组件测试（jsdom，`@/lib/fake-session-swr`、`@/hooks/store/fake-store-hooks`、`@/lib/fake-refusal`），看到失败：
  - 页面状态：
    - 卸下再挂上回到默认（W6）；
    - “已归档”与“已暂缓”互斥；
    - 换页签时“全部”的类型保留，选中保留（spec §2.5）。
  - 页签：Headless `Tab`、`aria-selected`，名字含数，0 不显示，99 以上 `99+`。
  - 列表：
    - 键带 `filterKey`，换筛选即换键、取新的；
    - 首读的占位；
    - 读失败的可重试（`PlanningReadFailure`），不是占位；
    - 每个页签的空态；零行而有 `next_cursor` 时给“加载更多”、不给空态（re-93 m6）；
    - “加载更多”有 `next_cursor` 才出现、是 `<button>`、在途停用。
  - 卡片：
    - 主按钮按下即选中、打开预览；未读的排队 PATCH `{read: true}`、已读的不发；PATCH 被扣住或被拒时预览照样打开（R23）；
    - “Till”只在暂缓中；
    - 未读的点有 `sr-only` 文字。
  - 预览：
    - 选中设浮层，`hostScope` 是 `JSON.stringify(["notification", slug, notificationId])`；
    - 项目不在调用者的项目列表里时经 `useProjectRead` 读它：读在路上显示占位，读完浮层渲染；**读被拒时清选中并提示**（re-93 I4）；
    - 选中的那一行因 `notification.not_found` 被移出时清选中；
    - 收集箱条目照普通浮层（没有 `InboxContentRoot`）；
    - 关掉浮层清选中；
    - 页面卸载清浮层。
  - 页面标题经 `useRouteWorkspace()`（router 的工作区不同的一格）。
- [ ] Step 3：实现新树；删除旧树、旧数据层与 spec §2.9 列的包导出；常量改写；`RootStore` 21；`activity-list.tsx` 删两行。
  - knip：`make knip` 的收据（没有未用的文件、导出与依赖；它查不出包的 barrel 导出，所以另有下一条）。
  - 死成员：重跑 Step 0 的脚本（收据 `t5-dead-after`），比对 `dead-before` 与 `dead-after` 的**整份**差异：消失的行逐条对上删掉的文件；**新出现的每一行**都要有去处（同一提交删掉，或写明留用与它的读者），台账列表（re-93 I2）。
  - 告警：`oxlint` 计数 31 的收据（`t5-oxlint-count`），同一提交改上限与 `no-empty-pattern`。
- [ ] Step 4：变异照上表 T5 各行。`make lint-web`、`make knip`、`make test-web`；守卫的绿收据（`t5-green-keywords`）。提交 `feat(M7/P4): 通知页的骨架与预览，删除 Plane 的通知 store、service、类型、常量与随之失去读者的导出，加 old-notification-urls`。核对 T3、T4 的 store 成员 `getList`、`fetchList`、`fetchMore`、`update` 已有产品读者，台账记下。

### T6 卡片的文字与执行者

**Files**：新 `WN/card/describe-change.ts` 与测试；新 `WN/card/sentence.tsx`；`WN/card/item.tsx`（句子、头像、摘录）；新 `C/hooks/use-actor-name.ts` 与测试；`…/activity/actions/helpers/issue-user.tsx`（改读它，spec R22）；en、zh-CN 的 `notification.card.*`（spec §2.6 的表）。

**Steps**：
- [ ] Step 1：先写测试，看到失败：
  - `describeChange` 逐行（spec §2.6 的表每一行，含加入与移出、`new_issue`/`old_issue` 都在、只有一端、都为空、未知字段）。移除与清掉的格用服务端真实的形状：`new_value: ""`，设置的格 `old_value: ""`（spec §2.6“有新值是非空串”）。父工作项与关联的 `old_value`/`new_value` 填上假值时不出现在参数里。
  - 日期按日历日：`"2026-10-17"` 在 `TZ=America/Los_Angeles` 下另跑一次，显示的仍是 17 日（re-93 m15）。
  - `parent_removed` 的 en、zh-CN 文案说“不再以 {ref} 为父工作项”（re-93 m14）。
  - 一个测试遍历表能给出的每个键，核对 en 与 zh-CN 都有。
  - `useActorName`：`null` → `SITE_NAME`；不在成员里的 → `common.deactivated_user`；成员 → `display_name`。
  - 卡片：
    - 摘录以纯文本显示（没有 `LiteTextEditor`）；
    - 提及的句子；
    - 附件没有链接。
- [ ] Step 2：实现到通过；`IssueUser` 改读 hook，渲染不变（链接与名字）。
- [ ] Step 3：变异照上表 T6 各行。门禁同 T2。提交 `feat(M7/P4): 通知卡片按新通知的字段与动词写文字，执行者的显示一处判定`。

### T7 页头：全部已读、刷新、类型筛选与视图菜单

**Files**：新 `WN/sidebar/{header-options,type-filter-menu,view-menu,applied-filters}.tsx` 与测试；`WN/sidebar/header.tsx`；`web/packages/constants/src/notification.ts`（`NOTIFICATION_TYPE_OPTIONS`，由 `NotificationType` 推出）。

**Steps**：
- [ ] Step 1：先写测试，看到失败：
  - 全部已读：
    - 请求体取页面状态，“全部”带类型数组、“提及”与无类型不带；
    - 在途停用；
    - 成功提示、被拒提示，成败都重取（store 的方法被调）；
    - 会话换了之后迟到的回答不提示（`followInSession`）。
  - 刷新：调 `fetchList` 与计数，在途时转圈。
  - 类型菜单：
    - 只在“全部”出现；
    - 按钮里没有 `<button>`；
    - 项是 `<button aria-pressed>`，项之间用 Tab，Enter、Space 各切换一次，在一个 `<form>` 里也不提交（旧 `menu-option-item.test.tsx` 的性质，接上 T5 的空窗）；
    - Escape 关上并把焦点还给按钮（re-93 I3：`PopoverMenu` 没有方向键，不给项加键盘处理）。
  - 已选的小块与“清除全部”只在“全部”。
  - 视图菜单：三项各切换，“已归档”与“已暂缓”互斥；键盘同类型菜单。
- [ ] Step 2：实现到通过。核对 `markAllRead` 已有产品读者，台账记下。
- [ ] Step 3：变异照上表 T7 各行。门禁同 T2。提交 `feat(M7/P4): 通知页的页头——全部已读按当前页签与筛选，菜单不嵌套按钮`。

### T8 卡片的动作与预设的暂缓

**Files**：新 `WN/card/{actions,snooze-menu}.tsx` 与测试；新 `WN/card/snooze-time.ts`（本 Task 只有预设：`SNOOZE_PRESETS`、`presetMoment`）与测试；`filters.ts` 加 `viewOffers` 与测试（spec R25）；`WN/card/item.tsx`；en、zh-CN 的 `notification.card.actions.*`。

**Steps**：
- [ ] Step 1：先写测试，看到失败：
  - `viewOffers`：默认视图两个都给；“已暂缓”不给归档；“已归档”不给暂缓（主控裁定，re-93 I1）。
  - 卡片：“已暂缓”的视图里没有“归档”按钮、有暂缓菜单且首项是“取消暂缓”；“已归档”的视图里没有暂缓菜单、有“取消归档”。
  - 已读 / 未读、归档 / 取消归档的请求体与成功提示（`notification.toasts.*`）；被拒经 `useRefusalToast`，不提示成功。
  - `notification.not_found` 之后这一行消失；是选中的那一行时选中也清掉。
  - 在途停用。
  - 按钮以这一行命名，Tab 可达（不在 `hidden` 之后）。
  - 暂缓菜单（`CustomMenu`）：
    - 预设的时刻 = 此刻 + N×24 小时（假时钟）；
    - 暂缓中首项是“取消暂缓”，发 `{snoozed_till: null}`；
    - “自定义”这一项在 T9 接上弹窗之前不出现；
    - 方向键与 Enter 可用，Escape 把焦点还给按钮。
  - 跟进经 `followInSession`：会话换了之后迟到的回答不提示。
- [ ] Step 2：实现到通过。
- [ ] Step 3：变异照上表 T8 各行。门禁同 T2。提交 `feat(M7/P4): 卡片的已读、归档与预设的暂缓——按行命名，键盘可达，随视图收窄，被拒时提示`。

### T9 自定义暂缓的弹窗

**Files**：新 `WN/card/snooze-modal.tsx` 与测试；`snooze-time.ts` 加 `snoozeSlots`、`slotMoment`、`slotLabel` 与测试；`snooze-menu.tsx`（“自定义”）；en、zh-CN 的 `notification.snooze_modal.*`；删除不再有读者的旧 `notification.*` 文案键（Global Constraints 的“文案”，人工对照，台账列出）。

**Steps**：
- [ ] Step 1：先写测试，看到失败：
  - `snoozeSlots`：48 个；今天只给晚于此刻的（假时钟，含 23:45 时一个都没有）；别的日子全给。
  - `slotMoment`：`"00:30"` 是零点半、`"12:30"` 是中午半（W1），在负偏移时区 `TZ=America/Los_Angeles` 下另跑一次。
  - `slotLabel`：en 的 12 小时制、zh-CN 的 24 小时制。
  - 弹窗：
    - 名字是 `ModalTitle`；
    - 没选时两个 `FieldError` 显示、`aria-invalid`、第一个被拒的取得焦点；
    - 422 `must_be_future` 在时间一栏下；
    - 其余拒绝提示、弹窗留着；
    - 在途时 Escape 与点外面关不掉；
    - Escape 与“取消”关上之后焦点回到这一行的暂缓按钮；
    - 成功之后关上；
    - 会话换了之后迟到的回答不提示、不关（`followInSession`）。
- [ ] Step 2：实现到通过。
- [ ] Step 3：变异照上表 T9 各行。门禁同 T2。提交 `feat(M7/P4): 自定义暂缓的弹窗——日期与半小时的时段按语言标注，字段提示，在途关不掉`。

### T10 关注与取消关注

**Files**：
- `C/services/issue/issue-detail.service.ts`（两个方法）与测试；
- 新 `C/store/issue/issue-details/subscription.store.ts` 与测试；`issue-details/root.store.ts`（接线，照 `reaction`）；
- 新 `C/components/issues/issue-detail/subscription-button.tsx` 与测试；`issue-detail-quick-actions.tsx`；`peek-overview/header.tsx`；
- 新 `…/work-item/subscription-command.ts` 与 `subscription-command.test.tsx`；`commands.ts`；
- 允许清单里页头与命令面板测试的编译跟随；`.oxlintrc.json`（如有取数）。

**Steps**：
- [ ] Step 1：先写测试，看到失败：
  - store：
    - POST 在工作项的队列里（同一工作项在途的写之后才发）；
    - 之后 `GET /api/v0/issues/{id}`，`is_subscribed` 取自它；
    - 重读被扣住时 `isPending` 仍真、详情的值不变；
    - 拒绝（`issue.not_found`、`forbidden`、`server_busy`）之后同样重读，再交出拒绝。
  - 按钮：
    - 管理员、成员、访客（看全部与只看自己的各一格）看得到；
    - 没有角色、持有的不是详情、工作项已归档、项目已归档时没有；
    - 标签随 `is_subscribed`；
    - 在途 `common.loading` 且停用；
    - 成功、被拒各自的提示；会话换了之后迟到的回答不提示（`followInSession`）；
    - 名字在窄屏也是文字。
  - 详情页头与浮层页头各渲染它一次（嵌入的浮层也是）。
  - 命令：
    - 标题随状态；
    - `modifierShortcut` 为 `shift+s`；
    - 可见与可用的条件同按钮；
    - 提示经 `followInSession`；
    - `cycle-command.test.tsx:202` 照常过。
- [ ] Step 2：实现到通过。
- [ ] Step 3：变异照上表 T10 各行。门禁同 T2。提交 `feat(M7/P4): 工作项详情、浮层与命令面板的关注——不乐观，写后重读详情，访客可用`。

### T11 收起“保存为视图” → **检查点 1**

**Files**：`C/components/issues/issue-layouts/roots/{project-layout-root,planning-page-layout-root}.tsx`；新 `roots/save-view-collapsed.test.tsx`（用 `roots/fake-planning-page.tsx` 与项目页已有的替身）。

**Steps**：
- [ ] Step 1：先写测试，看到失败：项目页、迭代页、模块页的筛选行在有筛选、`issue_views_view` 打开、调用者是管理员时，没有 “Save view” 按钮。
- [ ] Step 2：删两个 prop。台账记 `enableSaveView`、`enableUpdateView` 与工作区 HOC 的分支从此没有传入者，到 P6a/P6b（spec R17）。
- [ ] Step 3：变异 `t11-mut-save-view-back`。门禁同 T2。提交 `refactor(M7/P4): 收起筛选行的“保存为视图”，P6a 随视图接口接回`。
- [ ] Step 4（**检查点 1**）：
  - `gates.sh cp1`；`playwright.m4-detail.config.ts` 与 `m6-pages` 三种引擎各一次（收据 `cp1-m4-detail`、`cp1-m6-pages`）。
  - 主控推送、核分支 CI。
  - 台账记 web 作业与 e2e 两片的时长（那时还没有新故事）。

### T12 e2e 的配置、夹具与 H1、H2–H4 (page)

**Files**：
- 新 `e2e/playwright.m7-pages.config.ts`（spec §2.11）；`e2e/package.json`（`test:m7-pages`）。
- 新 `e2e/fixtures/notification-pages.ts`：世界、地址、定位器；共用的帮手从 `cycle-pages.ts` 导入。
- 新 `e2e/fixtures/assert/notification.ts`：`subscriptionRows` 移来；通知行的读。
- `stories/collaboration/h1-subscription.spec.ts`：只改导入。
- 新 `stories/collaboration/{h1-subscription-page,h2-h4-notification-cards-page,h2-h4-auto-archive-page}.spec.ts`。

**Steps**：
- [ ] Step 1：写故事（spec §2.11 的三行）：
  - 只看自己的访客的工作项照 spec 的造法：作为成员时建，再降为访客；建的人自动关注，所以那一格从“取消关注”开始。
  - H2–H4 的卡片文件含收集箱条目的一格；“离开项目”一格在刷新之前先点一次那张卡片（提示、行消失、预览回到没有选中，re-93 I4）。
  - 时钟一格单独成文件 `h2-h4-auto-archive-page.spec.ts`：只用 `clockTest`（`e2e/fixtures/test.ts:103`），推进之后以新进程重新登录（`i16-auto-archive.spec.ts:217-218`），不断言“多久之前”（re-93 m10、m13）。
  - 先跑一个期望错误的版本，看到断言失败，再改正。
- [ ] Step 2：
  - `make build build-s3-fixture`，聚焦 chromium（`cd e2e && NERVE_VERSION=0.1.0-dev pnpm exec playwright test --project=chromium stories/collaboration/h1-subscription-page.spec.ts …`）；
  - 再 `--config=playwright.m7-pages.config.ts` 三种引擎，再 `--repeat-each=3`；
  - 台账记每个文件各引擎的时长；API 故事 `h1-subscription.spec.ts` 照常过。
  - `make lint-web`（e2e 的类型、lint、格式）。
  - 提交 `test(M7/P4): 三引擎的页面配置与 H1、H2–H4 的页面故事`。

### T13 H5、H6、H7 (page)

**Files**：新 `stories/collaboration/{h5-notification-list-page,h6-notification-states-page,h7-unread-counts-page}.spec.ts`；夹具按需。

**Steps**：
- [ ] Step 1：写故事（spec §2.11 的三行）：
  - 101 条：先建 100 个标签，一次修改加上去（`label_ids` 一次至多 100），再加一条别的变化；
  - 过期的暂缓经接口设两秒之后（照 `h5-h7-notification-inbox.spec.ts:210-233`）；
  - H5、H6 断言动作随视图收窄：“已暂缓”的卡片没有“归档”，“已归档”的卡片没有暂缓菜单；H6 的归档、暂缓在默认视图里做，取消在各自的视图里做（spec R25）；
  - H6 以 `test.use({ timezoneId: "America/Los_Angeles" })` 定时区，自定义暂缓选明天的 00:30，期望值按这个时区算（re-93 m11）；
  - 精确的请求表（`mounts.ts` 的 `APP`、`WORKSPACE` 与列表的读）只在直接打开通知页时断言；应用内跳转之后用 `expect.poll` 等内容，不数计数的请求（re-93 m12）。

  先看到错误的期望失败，再改正。
- [ ] Step 2：同 T12 的三步跑法与时长记录；提交 `test(M7/P4): H5、H6、H7 的页面故事`。

### T14 H20 (page) 的通知页 → **检查点 2**

**Files**：新 `stories/collaboration/h20-notification-pages.spec.ts`，过 400 行时按关注点拆成两个，标题同为 `H20 (page)`；夹具按需。

**Steps**：
- [ ] Step 1：写故事（spec §2.11 的 H20 一行的七项）：
  - 迟到的回答用 `sentHeld`/`holdAnswer`；
  - 换账号用 `anotherTabSignsIn`；
  - 换工作区与后退用 `moveWithinApp` 与浏览器的后退、前进；
  - 键盘：两个 `PopoverMenu` 走 Tab、Enter、Space 与 Escape，暂缓菜单走方向键（re-93 I3）；
  - 剪贴板用 `refuseClipboardWrites`；
  - 弹窗用 `dialogs.ts`；
  - 列表在按钮旁用 `expectListBesideButton`。

  先看到错误的期望失败，再改正。
- [ ] Step 2：同 T12 的跑法与时长记录；提交 `test(M7/P4): H20 的通知页`。
- [ ] Step 3（**检查点 2**）：
  - `sh .superpowers/sdd/P4-web-notifications/final.sh cp2`：七项加 m7-pages、m6-pages、m6-acceptance，各留 JSON 报告。
  - 七个新文件 `--repeat-each=3`（收据 `cp2-repeat`）。
  - 照 spec §2.11 估出 e2e 两片的时长，估出超过 30 分钟先报告、不推送。
  - 主控推送、核分支 CI，记两片的实际时长与 web 作业。

### T15 文档与交接（主控）

- **plane-diff**：spec R3、R5（应用侧栏没有数）、R6（不在聚焦时重取）、R8、R14（访客可关注）、R15（时段按语言标注、没有 AM/PM 切换）、R20 的 W1–W8 逐条、R25（动作随视图收窄；既归档又在暂缓中的通知在网页里看不到，直到暂缓到期），与 D1、D2、D5 的网页一侧。行号对照主目录的只读 `plane/` 逐一核过。
- **frontend-changes**：
  - 3.1 的 M7 一行改为进行中（P4 的范围）；
  - 3.2 加五行：通知的数据层、通知页、未读数、关注、收起；
  - “挂载时的取数”一行（`docs/v0/frontend-changes.md:233`）补“顶栏的未读数由 M7/P4 加回”。
- **交接**：spec §5 的表，每条写“处理结果（M7/P4）”；表里写关闭的照写关闭，其余保持 open；`HO/M3-closeout.md:42` 写“本段不涉及”。
- **死成员**：重跑脚本（收据 `t15-dead-final`），剩下的行（P5–P7 的，含收集箱组件上通知的两个 prop）写进阶段记录。
- **告警**：核对本段改到与新建的文件 oxlint 为零、上限 31。
- **设计**：§12 的 P4 一行加 spec 链接，§14 的 P4 一行。
- **阶段记录** `reviews/P4-web-notifications-review.md`：§1 方法与范围、§2 验收 → Task、§3 实施中的裁定、§4 向负责人说明（R6 与 R25 的缺口，进主控的 M7 收官报告）、§5 门禁与 CI、§6 时长、§7 交给 M8、§8 文档、§9 审查与集成。
- 检查文档链接与关键词，提交 `docs(M7/P4): …`。

### T16 门禁、整审与集成（主控）

1. **整分支审查**：一次 fresh 只读的整分支 deep-review（fable）。Critical/Important 在一次修复轮里各有 RED/GREEN；Minor 记台账，主控裁定改否。之后一次限定复审。
2. **终检**：`final.sh fin`（八张收据之外加三个页面配置）。
3. **推送前的估算**：照 spec §2.11 估出 e2e 时长。
4. **最终分支 CI**：推送准确的 HEAD，核分支 CI。
5. **合并**：`--no-ff` 合入 main，推送后以 `ls-remote` 核对。
6. **main CI**：核准确的 main CI。
7. **归档与清理**：归档 `.superpowers/sdd/M7-P4-integration-<日期>`，清理 worktree 与分支。
8. **记忆**：spec A8 与设计 §14 由下一段的 T1 回填。

## Review Focus

1. **会话分代（v0 §7.7）**：客户端来自这一代；SWR 键带 `loginId`、工作区 id 与 slug，列表的 fetcher 从键算筛选；换代与换工作区后旧的回答不写进新页面（W3、W5）；`await` 之后都经 `followInSession`；写一个接一个；确认写进持有的每一份；`fetchPart` 扩展 `Reconciled` 而不复制。
2. **契约的陷阱**：“提及”不发 `type`；空的类型不发；“只看未读”是 `read=false`；每页 100；游标只取回答的；取消暂缓发 `null`；暂缓按此刻；全部已读不带 `read`、不在本地标、成败都重取。
3. **页面状态与地址**：页签、筛选、选中在通知布局的 context 里，随页面与工作区结束；页面只读路由的工作区；预览以通知为 key，项目不在手时经 `useProjectRead`；预览渲染不出来时有占位或提示，不留空白。
4. **视图与动作**：两个开关互斥，“已暂缓”不给归档、“已归档”不给暂缓（R25），页面造不出哪个视图都看不到的通知。
5. **可访问与拒绝**：
   - 卡片一个主按钮加同级的动作、动作键盘可达且按行命名；
   - 页签、菜单、弹窗的名字、键盘、焦点；两个 `PopoverMenu` 走 Tab，不给项加键盘处理；
   - 没有嵌套按钮；
   - 没有区域叫 “Notifications”；
   - 每个失败都提示、不 `console.error`；
   - 被拒不提示成功。
6. **关注**：不乐观、经工作项的队列、写后 `refreshAuthoritative`（被拒也重读）；访客可用；已归档隐藏；三处同一个门。
7. **删除干净**：Plane 的通知 store、service、hook、类型、常量、工具、预览 hook 与收集箱分支都删除；三个包里随之失去读者的导出也删（或写明留用的读者）；死成员前后整份比对；`old-notification-urls`；上限 31、`no-empty-pattern` 为 error、`activity-list.tsx` 的空解构删掉；“保存为视图”只去两个 prop、其余登记给 P6a/P6b。
8. **共用的外壳**：`WORKSPACE` 一处改；S2 与 M6 的页面故事不回归（检查点 0、1 与终检的 m6-pages、m6-acceptance、m4-detail）。
