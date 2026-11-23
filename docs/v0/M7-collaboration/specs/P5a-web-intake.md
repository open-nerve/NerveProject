# M7 / P5a：网页的收集箱——数据层、列表、详情与分诊

> 状态：草稿；预检 [re-94](../../../reviews/re-94-M7P5a预检.md) 修正后通过（处理见它的 §6，主控的裁定在台账 `rulings.md`），T1 提交。基线 main `89be91e5`（P4 的 noff）。设计见 [M7 设计](../M7-design.md) §2（H9–H11、H20）、§3.5、§3.6、§3.10、§3.11、§7（P5 一条 `:245`）、§11、§12（P5a、P5b 两行 `:306-307`）、§15 A、B；网页的长期规则见[总体设计 §7.2、§7.7](../../v0-design.md#77-stores-按会话分代长期有效m2-起)。接口合同见 [P2a spec](P2a-intake-submit-triage.md)（提交、分诊、`IssueDetail.intake`）、[P2b spec](P2b-intake-list-edit-delete.md)（列表、内容修改、删除、待处理数）与 [P2c spec](P2c-intake-assets.md)（资产，P5b 用）。现状调研在台账 `research/scope.md`（现状、交接、Plane 行为、共用件）、`research/split.md`（两段的划分与 Task 纲要）、`research/questions.md`（Q-n 的选项与建议）；主控裁定在台账 `rulings.md`（下文 Q-n 指它）。下文 `C/` = `web/apps/web/core/`，`A/` = `web/apps/web/app/(all)/[workspaceSlug]/(projects)/`，`IN/` = `C/components/inbox/`，`K` = `tools/keywords.json`，`IY` = `api/modules/issue.yaml`，`SG` = `web/packages/api-client/src/schema.gen.ts`，`HO/` = `docs/v0/M7-collaboration/handoffs/`，`D` = `docs/v0/M7-collaboration/M7-design.md`（行号按 T1 提交之后），`pw/` = 只读的 `plane/apps/web/`，`pa/` = 只读的 `plane/apps/api/plane/`。本段不改 `server/`、`api/`、`deploy/` 与 CI。

## 1. 目标与边界（含 P5a / P5b 的划分）

设计 §12 的 P5（`D:245`）约 27 个 Task，超过每段约 16 Task 的上限（`D:291`），实施前按用户看得到的面再划为两段（Q1，先例 P2a–P2c、P3a–P3d）：

- **P5a（本 spec；`research/split.md` §2 的 15 条，plan 细化为 17 个 Task：T6 拆为详情侧 T6a、列表侧 T6b 与旧数据层 T6c）**：
  - 收集箱的数据层；M4 逐工作项写队列上的接缝；
  - 收集箱页：页签与总数、筛选、排序、滚动分页、地址里的选中；
  - 详情：名字、描述正文、表情、按状态名的动态、评论、描述版本、按角色与状态的属性、“重复于”；
  - 页头按关注点拆开与分诊：接受弹窗、拒绝、暂缓与取消暂缓、标重复、删除；
  - 侧栏的待处理数；入口规则 `intakeEntryOn`（侧栏、页签栏、命令面板，M6-M7-07）；
  - Plane 的旧收集箱层整套删除；
  - 新建弹窗只改到经新接口提交所必需的（§2.10）；
  - H9、H10、H11（文字内容与边界）、H20（收集箱页）的页面故事。
- **P5b**：
  - 新建弹窗的其余：M6-M7-01 的 `requestClose`、`ModalTitle`、`FieldError`、去掉文档级 Escape、四个文件离开 `getTabIndex`、英文字面量的提示；
  - 详情的描述图片与附件（M5-M7-01）；详情的链接（Q14）；
  - 通用详情与浮层在 `intake` 未被接受时隐去状态（Q12）；浏览页的重定向（Q12）；
  - 通知预览里嵌入收集箱的内容（Q13）；助手多选的忙时守卫（Q34）；
  - H8 (page)；H11 (page) 的资产，`[local]` 与 `[s3]` 三引擎（Q21）；嵌入分诊的故事；H2–H4 (page) 收集箱一步。

关键词的 phase 两段共用 `M7/P5`（Q2；`tools/keywords.mjs` 只认 `M<n>/P<k>`）。

**P5a 不做：**
- 上面 P5b 的各项（Q12、Q13、Q14、Q34、Q21 由主控标为 P5b）。
- `getTabIndex` 与 `projectMemberIds` 的最终删除（P6a，Q36）：P5a 只去掉收集箱筛选的 `projectMemberIds` 读者（`IN/inbox-filter/filters/filter-selection.tsx`）；新建弹窗的四个 `getTabIndex` 读者随 P5b。
- 服务端、合同与 CI：G1（重复目标的摘要）、G2（选择器给已归档的目标）、G3（带修改的接受合为一个写）都不改（O1，§7 登记）；计数的实时性与 P4 R6 一起进 M7 收官报告（O2、R8）；CI 余量只报告（O3、R25）。

**P5a 合入、P5b 之前 main 的样子**（同 P2a spec §1 `:14` 的写法，T14 写进阶段记录）：
- 收集箱页经 `/api/v0` 可用：列表、筛选、排序、滚动分页、详情、分诊、删除、侧栏的待处理数，网页没有 Plane 的收集箱地址。
- 新建弹窗经新的接口提交，成功之后照旧跳到新条目；关闭策略仍是 Plane 的（main 现状：文档级 Escape、编辑器未就绪时的关闭）。
- 收集箱详情没有图片上传，也没有附件与链接一栏（main 现状，`IN/content/issue-root.tsx:191` 是空的 `div`）。
- 收集箱条目的通知仍以普通浮层打开，属性里 “State” 一栏是占位（P4 R12、P4 阶段记录 §7 `reviews/P4-web-notifications-review.md:147`）。
- 浏览页仍把收集箱条目显示成普通详情，它的写答 409 `issue.in_triage`。

这些在 main 上本来就是这样，P5a 不让它们变坏；不算不安全：只有显式打开开关的项目有收集箱，写都在服务端按合同判定。阶段记录向负责人说明。

## 2. 交付合同

### 2.1 service

`C/services/issue/intake.service.ts`（新，Q32）：构造函数收这一代的客户端（`ApiClient`），每个操作一个方法，经 `unwrap`，回答生成的类型，不转换（先例 `C/services/notification.service.ts`）。由 issue 根 store 在构造时建（`C/store/issue/root.store.ts:126-134` 的一列 service 之后加 `intakeApi`）。

| 方法 | 操作（`IY`） | 第一个产品读者 |
|---|---|---|
| `counts(slug)` | `getIntakeCounts` `GET /api/v0/workspaces/{slug}/intake-counts`（`IY:1058-1088`） | T2 |
| `list(projectId, query)` | `listIntakeIssues` `GET /api/v0/projects/{project_id}/intake-issues`（`IY:744-857`；查询的类型 `SG:7524`，数组经 CSV，`web/packages/api-client/src/index.ts:35-40`、`:62`） | T3 |
| `triage(issueId, body: IntakeTriage)` | `triageIntakeIssue` `POST /api/v0/intake-issues/{issue_id}/triage`（`IY:904-965`），回答 `IssueDetail` | T5 |
| `update(issueId, body: IntakeIssueUpdate)` | `updateIntakeIssue` `PATCH /api/v0/intake-issues/{issue_id}`（`IY:966-1022`），回答 `IssueDetail` | T5 |
| `delete(issueId)` | `deleteIntakeIssue` `DELETE /api/v0/intake-issues/{issue_id}`（`IY:1023-1057`），204 | T5 |
| `submit(projectId, body: IntakeIssueCreate)` | `submitIntakeIssue` `POST /api/v0/projects/{project_id}/intake-issues`（`IY:858-903`），201 `IssueDetail` | T6c |

problem 码已在 `PROBLEM_MESSAGES`（`C/lib/error-messages.ts:87-91` 收集箱的五个，`:118` `not_allowed`，`:122` `must_be_future`），不加码。

### 2.2 纯函数

`C/store/issue/intake/filters.ts`（新：筛选、键、查询、`matches`）、`C/store/issue/intake/status.ts`（新：条目的状态与暂缓的时刻，re-94 O5）与 `C/store/issue/issue-write-fields.ts`（98 行，加一个函数）。每个陷阱一格单测与一个变异（plan 的性质表）。

**合同的陷阱**：

| # | 陷阱 | 规则 |
|---|---|---|
| C1 | `status` 省略 = 每个状态（`IY:782-786`；P2b R9，`specs/P2b-intake-list-edit-delete.md:107`） | 每个页签都显式发 `status`：“未处理”默认 `pending`，可加 `snoozed`；“已处理”默认 `accepted,declined,duplicate`；至少留一个（Q7，R9） |
| C2 | 每个参数只给一次，重复的参数 422 `not_allowed`（`IY:753-757`） | 数组只经 CSV 的一个参数；空数组不发；同一个键不出现两次 |
| C3 | 日期只一个区间：一天、`a..b`、`a..`、`..b`，不许 `none`，按 UTC 日（`IY:824-838`） | 预设与自定义都化成这四种之一；选中预设时按本地日写成日期串（Q19，P2b R8 `:106`，已登记） |
| C4 | `order_by` 只有六个值（`IY:839-846`） | 字段（创建、更新、编号）与升降两个控件合成一个值；Plane 的 `issue__created_at` 与 `sort_by` 不发 |
| C5 | `limit` 1–100，默认 50（`api/common.yaml:12-22`） | 显式发 50（Q9） |
| C6 | 游标只来自回答（`IntakeIssuePage.next_cursor`，`IY:2524-2537`） | `cursor` 只取持有页的 `next_cursor` |
| C7 | 状态按回答的时刻：到期的暂缓是待处理（`IY:782-786`，`IssueIntake` `IY:2332-2352`） | `matches` 用 `now` 时的有效状态 |
| C8 | `snoozed_till` 在请求时刻之后、到微秒（`IY:933-935`）；`snoozed_till`、`duplicate_to_id` 各只随自己的状态（`IntakeTriage` `IY:2389-2409`） | 暂缓的时刻见 R13；请求体只带本状态的字段 |
| C9 | 内容修改只有七个字段（`IntakeIssueUpdate` `IY:2538-2579`，`additionalProperties: false`）；访客提交者只给名字、描述与 `skip_activity`（`IY:975-977`） | `intakeUpdateBody` 只取这七个，别的字段有值即抛错 |
| C10 | 提交的 `target_date` 是不可为 `null` 的日期串，“不给即无”（`IntakeIssueCreate` `IY:2385-2388`，`additionalProperties: false`） | 没选日期时请求体不带这个键 |

**`filters.ts`**：
- `IntakeTab`（`"open" | "closed"`）、`IntakeFilters`（页签、状态、优先级、负责人、创建者、标签、创建日、更新日、排序），`defaultFilters(tab)`。
- `filterKey(filters)` 与 `filtersOf(key)`：规范化（数组排序、去重）之后可逆，每个字段一格往返（P4 re-93 m5：取数只读键）。
- `listQuery(filters, cursor?)`：C1–C6；`limit: 50`；没给游标不带 `cursor`。
- `matches(row, filters, now, me)`：每个筛选维度（`assignee_id` 的 `me` 由 `me` 解析，标签与负责人的 `none`），状态用有效状态。
- 日期区间的换算（T7）：预设与 `DateFilterModal` 的输出（`C/components/core/filters/date-filter-modal.tsx`）→ C3 的四种形状。

**`status.ts`**：
- `effectiveStatus(intake, now)`。
- `inIntake(detail: IssueDetail | undefined)`：`detail !== undefined && detail.intake !== null && detail.intake.status !== "accepted"`（`IssueDetail.intake` 的说法，`IY:2212-2222`）；**`inIntake(undefined)` 为假**：只持有行（`Issue`，没有 `intake`）的工作项不在收集箱，服务端的列表不列收集箱中的工作项（`IY:293`、`:396`、`:526`）。
- `rowOf(detail: IssueDetail): IntakeIssue | undefined`：取 `IntakeIssue` 的键（`IY:2410-2423`），`intake` 为 `null` 时没有行。
- `snoozeDay(now)` 给最早可选的日（明天）；`snoozeInstant(day)` 给那天的第一个本地时刻（R13）。

`issue-write-fields.ts` 加 `intakeUpdateBody(body: IssueUpdate): IntakeIssueUpdate`（C9），它的输入是 `updateBody` 的输出（`issue-write-fields.ts:26-52`，`skip_activity` 的规则照旧）。

只给界面用的形状（`IntakeFilters`、`IntakeTab`）在 `filters.ts`；页签、状态名、排序、日期预设的选项（值与文案键的字面量）在 `@nerve/constants` 的 `intake.ts`，随各自第一个读者加入（T6a、T6b、T7），旧的在 T6c 删除。其余类型只来自 `@nerve/api-client`。

### 2.3 store

`C/store/issue/intake/intake.store.ts`（新）：issue 根 store 的子 store `root.intake`（Q32，设计 §15 A `D:356`），经它找兄弟（`root.writes`、`root.issues`、`root.rootStore`），不另建 `RootStore` 的成员（`C/store/root.store.test.ts:106` 的数在 T2 不变）。

- **计数**：`ReconciledByKey<IntakeCounts>`，按工作区的 id 存；`fetchCounts(workspace: WorkspaceRef)`；`pendingCount(slug, projectId)` 经调用者此刻的工作区列表把 slug 换成 id，回答里没有那个项目时是 `undefined`（v0 §7.7“按工作区的 id 存”）。
- **列表**：`ReconciledByKey<IntakeIssuePage>`，键 `${projectId}|${filterKey}`；`fetched` 记每个键的项目与筛选。
  - `getList(projectId, filters)`；`fetchList(projectId, filters)` 以 `fetch` 换第一页（之后的页丢掉）；
  - `fetchMore(…)` 经 `ReconciledByKey.fetchPart`（`C/lib/reconciled.ts:167-175`）取下一页：以持有页的 `next_cursor` 发；回答到时持有页的 `next_cursor` 仍等于发出的游标才接上（去重、`next_cursor` 与 `total` 取回答的），否则什么都不写（P4 `3f4c3ba8` 的认游标合并，先例 `C/store/notification/notification.store.ts:113-122`）；没有 `next_cursor` 不发。
  - 列表的行是这个 store 自己的副本：只随写的确认与冲突之后的重读变（下一条），别人改了要等列表重读（同 P4 的通知列表，主控裁定，§7 登记）。
- **确认**（写的接缝调用，§2.4）：
  - `confirm(answer: IssueDetail)`：写进这个项目持有的每一份列表：`matches` 则原地替换，否则移出；**不往列表里插行**（新相符的行由下一次取数带来，代价同 P4 R8）；移出一行时那一份的 `total` 减一，替换不变（R31）。
  - `drop(projectId, issueId)`：从每一份移出，`total` 同上。
  - `countAgain(workspace)`：只重取这一代已经取过的那个工作区的计数（Q10 的门开着时才有），从不越过门自己起一个读；不占队列；失败留着原值。
  - 迟到的确认只写它的项目的列表，别的项目的列表不动（项目的 id 全局唯一，列表的读也只认项目，所以不另比工作区）。
- **提交**：`submit(scope, body)` 经 `IntakeService.submit`；成功之后重取这个项目持有的每一份列表的第一页，`countAgain`（不在本地插行，R30），把回答交给调用者（新建弹窗以它的 `id` 跳转，§2.10）；被拒时值不变、交出拒绝。
- **会话**：`SessionChangedError` 时什么都不写；较旧的取数被较新的超过时不写（`Reconciled` 的既有性质，`C/lib/reconciled.ts:58`、`:85-107`）。

### 2.4 写的接缝：M4 的逐工作项写队列（Q3、Q28，高风险）

收集箱条目的写都经它的工作项在 `IssueWrites`（`C/store/issue/issue-writes.ts`）里的那一条队列（`enqueueIssueChanges`，`:56-99`），与它的评论、表情、关注等写（`C/store/issue/issue-details/{comment,reaction,subscription,link,relation,sub_issues}.store.ts` 都经它）一个接一个；不另开队列（Q3）。理由：收集箱详情复用的名字、描述、标签控件经 `useIssueOperations.update`（`C/components/issues/issue-detail/use-issue-operations.ts:50`）→ `projectIssues.updateIssue` → `changeIssue`（`C/store/issue/helpers/base-issues.store.ts:73-74`）→ `IssueWrites.updateIssue`（`:239-267`）写，版本恢复也经它（`C/store/issue/issue-details/description-versions.store.ts:76`）；另开一条队列，同一工作项的两条写就不再一个接一个。

**先拆再加**（Q3 的前提）：`issue-writes.ts` 302 行、“不再加长”（P4 plan）。T5 的第一个提交是纯搬移：
- `write()`（`:100-227`，确认进持有的每一份）与它的 `pending`、`pendingIDs`、`observePendingRead`（`:44-54`）移到新文件 `C/store/issue/issue-write-confirm.ts`（一个类，`IssueWrites` 持有它并转发这两个公开成员，公开的名字与签名不变；读者 `root.store.ts:306`、`helpers/issue-list-binding.ts:58` 不改）；
- `IssueWrites` 留队列与七个命令；
- 既有测试一行不改且全绿（`issue-writes.test.ts`、`issue-changes.test.ts`、`issue-drag.test.ts` 与读 `root.writes` 的详情测试）；
- `K` 的 `project-address-route` 收入新文件（它的文件名单与 hit 样例现有 `issue-writes.ts`，`K:2539`）。

拆完两份各约 150–165 行；之后的新命令让 `issue-writes.ts` 不超过 220 行（以 `wc -l` 收据记）。

**收集箱的去处**（内容修改、四种非接受的分诊、删除）是 `issue-write-confirm.ts` 里与 `write()` 并列的一条路，**不经 `write()` 的两段目录准备**（re-94 S1）：
- 不做发出之前的目录准备（`:110-123` 的 `prepareCatalog([initial])` 循环），也不做确认时的目录准备（`:161-180`，含被拒之后的 `list.markUnavailable`）；不调任何 `list.acceptConfirmed`，不 `invalidateProjectReads`，不登记 `pending`（它只为项目列表的 `before` 服务）。
- 只取 `write()` 的权限水位：发出之前的 `assertActive` 与权限的捕获、核对（`:107-108`、`:124-133`），回答之后的核对（`:146-160`：`assertActive`、回答的工作区、`current()`）。
- 理由一：项目列表用到状态时（按状态组分组或子分组、按状态名排序、带状态组筛选），目录准备先重读项目的状态，目录里仍没有这个工作项的 `state_id` 就抛错（`C/store/issue/helpers/issue-list-catalog.ts:85-94`、`:108-112`、`:138-148`）；收集箱中的条目在 Triage 状态，状态接口从不列它（`C/services/project/states.service.ts:9`）；列表在这一代里只登记、不注销（`C/store/issue/root.store.ts:225-228`），工作区级的列表覆盖工作区里每个项目。经 `write()` 走，持有一份这样的列表时收集箱的写在发出之前就失败，删除在确认时把那份列表整份标成失败。
- 理由二：项目列表的成员判定 `issueMembership`（`C/lib/issue-membership.ts:44-99`）不看 `intake`：没有状态组筛选的项目列表会把收集箱条目当作相符的行（有状态组筛选时，Triage 不在目录里，答“未知”，`:84-87`），确认若进项目列表就会插错行或改动计数；服务端的项目列表不列它们（`D:111`，`IY:293`、`:396`、`:526`）。在收集箱中的工作项不能做父工作项或子工作项（`D:111`），所以删除没有要连带的后代。
- 接受不受这两条影响：`write(undefined, …)` 没有 `initial`，确认时只准备回答，回答已在项目的默认状态（re-94 §3.2）。

**端点与去处**，在它轮到时、发出之前决定（v0 §7.7“修改的请求体取决于 store 所持的值时，轮到它发出时才算”）：

| 命令 | 轮到时所持的详情 | 发出 | 确认 |
|---|---|---|---|
| `updateIssue`（既有：控件、版本恢复、接受弹窗之后的修改） | `inIntake(root.issues.getIssueDetailById(id))` | `intakeApi.update(id, intakeUpdateBody(updateBody(makeBody(held))))` | 收集箱的去处：canonical 取回答（内容修改改 `updated_at`）、详情的动态与评论（`refreshAfterChange`，`issue-details/root.store.ts:149-154`）、收集箱的列表 |
| 同上 | 不在收集箱（已接受、从未提交、只持有行） | `api.update`（不变，`:261`） | M4 照旧 |
| `triageIntake(id, body, scope)`（新） | — | `intakeApi.triage`，之后同一轮次里重读（R34） | 见下 |
| `deleteIntake(id, scope)`（新） | — | `intakeApi.delete` | 收集箱的去处：canonical 的删除（`after: null`）、收集箱的列表移出、`countAgain` |

**分诊在它的轮次里重读**（R34，re-94 I1 甲，高风险，与 R3、R4 同列）：
- 非接受的分诊不改工作项（`IY:945-946`；`server/internal/modules/issue/app/triage_intake.go:184`），`updated_at` 不动；canonical 只按 `updated_at` 排先后，相等的读照样写入（`C/store/issue/issue.store.ts:28-34`）。所以分诊的回答不直接写进 canonical：一个在分诊之前发出、之后到达的详情读会把 `intake` 改回去。
- 五种分诊都在它的轮次里，分诊做成之后：
  1. 收集箱的列表照分诊的回答确认（替换或移出）；
  2. 以 `root.issues.refreshAuthoritative(id, { afterWrite: true })`（`issue.store.ts:113-120`）重读：它把新鲜度的下限提到写之后，之前发出的读被拒（`C/store/issue/issue-verification.ts:139-146`）；canonical 的最终值来自这次重读（先例：关注，`C/store/issue/issue-details/subscription.store.ts:55-64`）；
  3. `countAgain`。
- **接受另外像新建一样**（Q28，R4）：接受经 `confirmCreation`（`issue-writes.ts:232-238`，即 `write(undefined, …)`，`before` 为 `absent`，各列表的来源在 `send` 之前冻结，`:141`）在它的队列轮次里做；`send` 是“分诊、再重读”两步，交出的是重读的回答，于是持有的项目列表确认的是重读的回答，像新建的工作项一样出现、只一次、不用刷新。
- **重读的三种结局**：
  - 给出详情：如上。
  - 给 `undefined`（读不到，或被别的读超过而未定）：从收集箱的每份列表移出；项目列表不确认（接受时 `send` 以“没有可确认的回答”结束，命令接住它，不当作拒绝交出）；页面按选中读不到处理（§2.6）。
  - 失败（非 404）：分诊已做成，错误是这个工作项自己的（详情显示它与重试，核验已把覆盖这个项目的列表的格标成陈旧），命令不因它失败（先例 `subscription.store.ts:55-64`）；项目列表不确认，等它们自己的重读。`SessionChangedError` 照常交出。
- `issue-changes.test.ts` 的 `"inbox"` 一格（`:605`、`:651-662`）改为 `triageIntake(…, { status: "accepted" })`，三条 `it.each`（`:663-684` 写后的第二个 `GET`、写之前发出的读到达时不被采纳；`:685-695` 被拒时不发写后的读；`:696-712` 写后的读在途时换代即停）的断言都照旧成立：接受做成之后有写后的重读；被拒（通用的拒绝，不是 R28 的两个冲突码）不重读；重读在途时换代，`send` 里的 `assertActive` 抛 `SessionChangedError`、`refreshAfterChange` 不执行。

**拒绝**：值不变，交出拒绝，调用方经 `followInSession` 与 `useRefusalToast` 提示。另外：
- `issue.intake_not_found`：从收集箱的每份列表移出、`countAgain`，再交出（页面清选中）。
- `issue.intake_transition_not_allowed`、`issue.intake_closed`（别人先分诊了）：**在被拒那次写的轮次里、交出拒绝之前**以 `refreshAuthoritative(id, { afterWrite: true })` 重读（re-94 m1，同既有的写后重读都在轮次里：关注、评论、表情、链接）；重读的回答确认进收集箱的列表（P4 对 `notification.not_found` 的同一做法），给 `undefined` 时从收集箱的每份列表移出；`countAgain`；再交出（R28）。排在后面的同一条目的写等这次重读落定，不会在重读在途时读到 `undefined`（`issue-writes.ts:255` 的 `Issue has not been read`）。

**行盖在详情上**（R33，re-94 m2）：今天列表的行落在所持的详情上时，canonical 保留详情的 `intake`（`issue.store.ts:27-35` 的 `choose`）。别人接受了一个本人此前在收集箱读过详情的条目，之后它的行从项目列表落下，`inIntake` 仍为真，下一次修改会发到收集箱的端点。规则：`choose` 合并一个不带 `intake` 的行（`Issue`）与一个 `intake` 未被接受的详情时，合并结果的 `intake` 取已接受的形 `{ status: "accepted", snoozed_till: null, duplicate_to_id: null }`：列表不列 Triage 的工作项（`IY:293`、`:396`、`:526`），行的出现就说明它已被接受，这正是服务端此刻的值（`IssueIntake` 的两个字段只随暂缓与重复，`IY:2332-2352`）；`intake` 为 `null` 或已接受时照旧。收集箱的列表行（`IntakeIssue`）不进 canonical，不受此影响。

**权限与换代**：照 `enqueueIssueChanges` 与 `write` 既有的权限水位与 `SessionChangedError`（`:62-86`、`:107-108`、`:124-133`、`:146-160`），不另写。`WritesRoot`（`:24-35`）加 `intake`、`intakeApi`。

### 2.5 取数、键与门

会话的取数都经 `useSessionSWR`（`C/lib/use-session-swr.ts`）。它只有一套配置（`:25`），选项只有 `dedupingInterval`、`revalidateOnMount: false`、`revalidateOnReconnect: false`（`:37`），后者只给能力的属主（`:20-21`）；fetcher 只收键里的字符串（`:28-32`）。

| 取数 | 键 | 门（为 `null` 时不取） | 调用处 |
|---|---|---|---|
| 待处理数 | `["INTAKE_COUNTS", workspaceId, slug, projectIds]`，`projectIds` 是过门的项目 id 排序后以逗号连成的一个串 | `useRouteWorkspace()`（`C/hooks/use-route-workspace.ts:16`）不为 `null`，且调用者在这个工作区里至少有一个项目：未归档、`intake_view` 开着、`member_role` 不为 `null`（取自 project store 的 `knownProjectIds(workspace.id)`，`C/store/project/project.store.ts:64`、`:244-251`）（Q10，R7） | `C/hooks/use-intake-counts.ts`，只在 `C/components/workspace/sidebar/projects-list.tsx`（240 行）调一次 |
| 列表的第一页 | `["INTAKE_ISSUES", identity, workspaceId, slug, projectId, filterKey]`，`identity` 是 `root.permissionReads.identity(workspaceId, [projectId])` | 路由的工作区与项目在手，调用者有项目角色，项目的 `intake_view` 开着（Q31） | `IN/use-intake-list.ts` |
| 列表的下一页 | 不经 SWR：滚到哨兵时 `fetchMore`（`C/hooks/use-intersection-observer.ts:10`，Plane `pw/core/components/inbox/sidebar/root.tsx:73`） | 有 `next_cursor`、没有在途的 `fetchMore` | 侧栏的列表 |
| 选中条目的详情 | `["intake-issue", identity, slug, projectId, issueId]`，在 `IssueDetailHost`（`C/components/issues/detail-host.tsx:13`）之内，fetcher 是 `useIssueOperations(…).fetch`；`revalidateOnMount: false`，每次挂载以 `mutate()` 读一次自己的（P4 m1，`C/components/workspace-notifications/root.tsx:72-86`、`C/hooks/use-project-read.ts:8`） | 有选中 | `IN/content/root.tsx` |
| “重复于”的目标 | `["intake-duplicate", identity, slug, projectId, duplicateToId]`，fetcher 是 `root.issueDetail.issue.fetchIssueCore(slug, projectId, duplicateToId)`（`C/store/issue/issue-details/issue.store.ts:56-70`；读另一个工作项来显示的先例是父工作项，`:81`），一次 `getIssue`（Q17） | 详情的 `intake.duplicate_to_id` 不为 `null` | `IN/content/duplicate-of.tsx` |

- 列表与计数保留 SWR 的重连重读（不传 `revalidateOnReconnect: false`）；列表的重连重读即换第一页。
- “重复于”不经 `refreshAuthoritative`：那是核验的入口，会把覆盖目标所在项目的列表的格标成陈旧，读失败时把错误写到这些项目列表上（`issue-verification.ts:139-146`、`:236`、`:253`；re-94 m3）。

### 2.6 页面状态与地址（Q8，R5）

- **页签与选中在地址**：`?currentTab=open|closed`（未知的值按 `open`）与 `?inboxIssueId=<id>`，同 Plane。**用户点行是一次 `push`**（可后退）；**由动作引起的选中变化一律 `replace`**：没有选中而列表有行时选第一行（Plane 同样用 `replace`，`IN/sidebar/root.tsx:75-84`），分诊之后移到邻行、删除或读不到之后清选中（R14）。理由：这些变化是动作的结果，不是用户的导航；`push` 时后退会回到刚分诊掉、已不在这个页签列表里的条目；P4 的通知页在动作移走选中时也不增加历史。Plane 分诊之后是 `push`（`IN/content/inbox-issue-header.tsx:127-135`），§7 登记。
- **筛选与排序在收集箱布局的 context 里**（`IN/page-state.tsx`，由 `A/projects/(detail)/[projectId]/intake/layout.tsx` 提供），不进 store；以页签为 key：换页签即回到那个页签的默认值（Plane 的 `handleCurrentTab` 同样复位状态筛选与排序，`pw/core/store/inbox/project-inbox.store.ts:262-278`）；离开收集箱页即结束；项目、工作区的布局各以 id 为 key（`A/projects/(detail)/[projectId]/layout.tsx:36`，`web/apps/web/app/(all)/[workspaceSlug]/layout.tsx:21`），换项目、换工作区即结束。Plane 按项目把筛选存在会话的 store 里（`pw/core/store/inbox/project-inbox.store.ts:43-44`），§7 登记。
- **详情以条目 id 为 key**：换选中不带走上一个条目的输入、弹窗与在途状态。
- **选中读不到**只在这次挂载自己的读落定之后判定（re-94 m4；SWR 对这个键留着上一次的回答或失败，新挂载的第一次渲染就会给出它）：读落定为 404（`issue.not_found`）或无权，或回答的 `intake` 为 `null`：以 `replace` 清掉选中并提示。**别的失败**（网络、5xx）：不清选中，详情区域显示可重试的失败态（同浮层的 `error` 与 `retry`，`C/components/issues/peek-overview/root.tsx:93-116`）。都不留空白。
- **迟到的回答不写进新页面**：分诊之后移选中（R14）只在页面仍挂着、地址的选中仍是那一条时做（`followInSession`，`C/lib/in-session.ts:40`）。

### 2.7 页面与组件

- **路由**：`A/projects/(detail)/[projectId]/intake/page.tsx`（88 行）从路由参数读项目，`getProjectById(projectId)` 与 `getProjectRoleByWorkspaceSlugAndProjectId(slug, projectId)`（`C/store/user/permissions.store.ts:68-75`），不读 `currentProjectDetails`（`:38`）、不调不带 slug 的 `allowPermissions`（`:41`）；`intake_view` 关着时保留 Plane 的停用页（`:44-61`），不读列表（Q31）；页面标题经 `t()`。`layout.tsx`（22 行）渲染移来的页头与页面状态。
- **页头**：`C/components/projects/settings/intake/header.tsx`（92 行）移到 `IN/page-header.tsx`（Q32）：面包屑的名字经 `t("sidebar.intake")`（`web/packages/i18n/src/locales/en/navigation.json:19`，不再是字面量 “Intake”，`header.tsx:55`，`HO/M3-closeout.md:64`）；“Add work item” 只在收集箱开着、项目未归档、调用者有项目角色时给（R21、R27）；去掉对旧 store `loader` 的读（`:36`、`:65-70`）。
- **侧栏**（`IN/sidebar/`）：
  - 页签是 Headless `Tab`（先例 `C/components/cycles/active-cycle/cycle-stats.tsx:67-93`），名字经 `t("inbox_issue.tabs.*")`；选中的“未处理”页签显示列表的 `total`，“已处理”不显示数（Plane `IN/sidebar/root.tsx:105-109`，R31）；
  - 筛选按钮与已选筛选的一行（T7）；
  - 行：一个以编号与名字命名的链接，显示编号、名字、状态（按状态名，暂缓显示剩余天数 `findHowManyDaysLeft`，`web/packages/utils/src/datetime.ts:136`）、优先级、标签、创建者与时间；数据来自 `IntakeIssue`；
  - 滚动到底取下一页；首读的占位（`C/components/ui/loader/layouts/project-inbox/` 的两个 loader 留用）；读失败给可重试的 `PlanningReadFailure`（`C/components/planning/read-failure.tsx:23`），不是占位；每个页签的空态；`IN/sidebar/root.tsx` 的 `// TODO: Add translation` 一处改为经 `t()`。
- **窄屏**：侧栏的开关是一个有名字的按钮（今天是可点的 SVG，`IN/root.tsx:75-78`）。
- **详情区域**（`IN/content/`，导出名 `InboxContentRoot` 留在 `IN/content/index.ts`，R32）：
  - 最外层是一个有名字、可聚焦（`tabIndex={-1}`）的区域（名字经 `t("inbox_issue.a11y.detail")`），画在 `IssueDetailHost` 之外：分诊在轮次里重读（R34），重读在途时 canonical 给 `undefined`，`IssueDetailHost` 撤下详情的正文与画在里面的弹窗（`C/components/issues/detail-host.tsx:40-45`）；区域本身、分诊的弹窗（以条目 id 为 key）与读失败的提示都在它之外，不随重读撤下。
  - 正文在 `IssueDetailHost` 之内：名字（`IssueTitleInput`）、描述（`DescriptionInput`）、表情、描述版本（`IssueDescriptionVersions`，`C/components/issues/issue-detail/description-versions.tsx`，它的 `canRead` 只给管理员与成员，`description-versions.store.ts:20-25`，Q15）、动态与评论（`IssueActivity` 传 `isIntakeIssue`，评论不给复制链接，`issue-activity/activity-comment-root.tsx:81`）、属性（§2.9）。没有附件、链接与图片上传（P5b）。
  - 删除 Plane 的 FORMS 分支 “Intake Form user”（`IN/content/issue-root.tsx:170-173`，`HO/M4-closeout.md:19-20`）与模块级的 `IntakeWorkItemVersionService`（`:36`，`HO/M2-closeout.md:29`）。
- **可改**：项目未归档、收集箱开着、`inIntake(detail)`，且调用者是管理员或提交者（`created_by_id`）；访客提交者只改名字与描述，属性只读（Q16）；已处理的（接受、拒绝、重复）只读（Plane `pw/core/components/inbox/content/root.tsx:75-85`）。不可改时描述版本的恢复停用（`isRestoreDisabled`）。
- **页头**（`IN/content/header/`，470 + 239 行按关注点拆，Q35）：
  - 动作模型 `use-intake-actions.ts`：按角色、状态与项目决定给哪些动作（§2.8 的表）与每个状态的主动作，并持有它们的处理；桌面与窄屏两个页头都读它，不各写一份；
  - `navigation.tsx`（T6b，它要列表的顺序）：上一条、下一条两个按钮，名字经 `t()`（今天是英文 `aria-label`，`IN/content/inbox-issue-header.tsx:315`、`:322`，`inbox-issue-mobile-header.tsx:123`、`:130`，`HO/M3-closeout.md:65`）；↑、↓ 只在焦点不在编辑器与输入框里时换条目（Plane `inbox-issue-header.tsx:204-213`、`:227-233` 的键盘处理改为这条规则）；
  - `status-chip.tsx`：状态按名字；
  - 复制链接经 `useCopyLink`（`C/hooks/use-copy-link.ts:42`），成功与失败各自提示（`HO/M3-closeout.md:63`、`HO/M2-closeout.md:50`），删 `K:3262-3268` 的 `copy-hook` 例外；
  - “Open work item” 照 Plane。
- **动态**（Q18）：`C/components/issues/issue-detail/issue-activity/activity/actions/inbox.tsx` 按 `verb` 的状态名（服务端写 `Verb: to`，`server/internal/modules/issue/domain/intake.go:204`；设计 `D:100`）经 `t("inbox_issue.activity.<status>")` 说：待处理（“结束了暂缓”，Plane 没有）、暂缓、接受、拒绝、标为重复（不带目标，P2a R7）；别的值用一条通用的键（合同里 `IssueActivity.verb` 是不设枚举的字符串，`IY:3202-3203`，re-94 O1）；`activity-list.tsx:85-87` 删去 `case "inbox":`（nerve 只写 `intake`）。en 与 zh-CN 同一提交。
- **标签控件**：`C/components/issues/issue-detail/label/root.tsx` 删 `isInboxIssue`（`:27`、`:40`、`:51-53`、`:93`）与它对旧 hook 的读（`:17`），条目从 canonical 读（收集箱详情的读已把它放进 canonical）；新建标签的提示照工作项详情。

### 2.8 分诊（Q4、Q5、Q6、Q17、Q30、Q33）

**谁看到什么**（Q4、Q16、Q33、R27）：

| 调用者 \ 状态 | 待处理 | 暂缓 | 接受、拒绝、重复 |
|---|---|---|---|
| 管理员（含是项目成员的工作区管理员：权限 store 给 ADMIN，`permissions.store.ts:72-73`） | 接受、拒绝、暂缓、标重复、删除、复制、打开 | 同左，“暂缓”换成“取消暂缓”与“改暂缓” | 复制、打开 |
| 成员提交者 | 删除、复制、打开 | 同左 | 复制、打开 |
| 成员（非提交者）、访客（含访客提交者） | 复制、打开 | 同左 | 复制、打开 |
| 任何人，项目已归档 | 复制、打开 | 同左 | 复制、打开 |

**主动作**（R35 的落点用）：管理员面对待处理与暂缓的条目时是“接受”；其余情形没有主动作。

成员不再看到分诊按钮（Plane 给成员显示、点了提示 “Permission denied”，`pw/core/components/inbox/content/inbox-issue-header.tsx:92-100`、`:218-227`）；删去 `inbox_issue.errors.{snooze,accept,decline}_permission` 三个键（`web/packages/i18n/src/locales/en/inbox.json:34-38`，zh-CN 同）。

- **接受**（Q5，R12、R29，re-94 I2）：保留 Plane 的接受弹窗：`CreateUpdateIssueModal`，标题 `inbox_issue.actions.move`，`fetchIssueDetails={false}`（今天 `IN/content/inbox-issue-header.tsx:254-269`）。
  - 弹窗画在详情区域、`IssueDetailHost` 之外（§2.7），它的 `data` 是打开时取的那一份（canonical 的冻结对象），提交途中不随重读变。
  - `beforeFormSubmit` **只分诊**：`triageIntake(id, { status: "accepted" })`；被拒即提示并抛出，弹窗不再发 `PATCH /issues`（`C/components/issues/issue-modal/base.tsx:214` 在它抛出时停下）；今天的旧 store 吞掉这个拒绝（`C/store/inbox/inbox-issue.store.ts:100-103`）。它**不关弹窗、不移选中**：弹窗在 `beforeFormSubmit` 之后核对 `formScope()`（`base.tsx:93-98`、`:212-216`），它的键含 `isOpen` 与条目 id；弹窗一关或详情因选中移走而卸下，之后的修改不发、也没有提示（Plane 在这里关弹窗、跳到邻行，`IN/content/inbox-issue-header.tsx:137-142`）。
  - 弹窗的状态字段从项目的默认状态起（`StateStore.getProjectStates(projectId)` 中 `default` 的那个，`C/store/state.store.ts:37`、`:114`）：条目的 Triage 状态不在任何目录里（`C/services/project/states.service.ts:9`）。已有工作项的提交总带描述与改过的字段，所以这个初值不会多发 `state_id`。
  - 接受确认之后，弹窗经 M4 的更新命令发它改过的字段（`C/components/issues/issue-modal/update-command.tsx:54-102`）；此时 canonical 已是重读的回答、条目不在收集箱，走 `PATCH /issues`。
  - **选中的移动与弹窗的关闭在弹窗的提交整个落定之后**：更新确认时弹窗调 `onSubmit`（`base.tsx:224-225`）、再关自己（`:234` 的 `finishClose`，它调 `onClose`，`:149-169`）；动作模型在 `onSubmit` 记下“已确认”，在随后的 `onClose` 里移选中（R14）。修改被拒时：错误在字段下（`form-submission.ts:121` 的 `onRefused: underFields`），选中不动，弹窗开着并说明条目已被接受（G3）。
  - **再次提交时**，所持详情已不在收集箱（不论是本人还是别人接受的）：`beforeFormSubmit` 不再分诊，只发修改（R29）。
  - **分诊被冲突拒绝**（`issue.intake_transition_not_allowed`：别人先分诊了；`issue.intake_not_found`：条目已删）：提示；R28 的重读已在轮次里把页头换成条目此刻的状态（或清了选中）；弹窗关上，选中不移，焦点照 R35。别的拒绝（无权、`server_busy` 等）：提示，弹窗开着可重试。
- **拒绝**：确认弹窗（`IN/modals/decline-issue-modal.tsx`）→ `{ status: "declined" }`。
- **暂缓**（Q6，R13）：弹窗用 `ModalTitle`，日历只给明天及以后、默认明天（Plane 默认今天、今天可选，`pw/core/components/inbox/modals/snooze-issue-modal.tsx:24`、`:47-51`，nerve 会答 `must_be_future`）；发 `{ status: "snoozed", snoozed_till: snoozeInstant(day) }`。暂缓中的条目可“取消暂缓”（`{ status: "pending" }`）或改暂缓（再选一天）。
- **标重复**（Q17）：选择器弹窗（`IN/modals/select-duplicate.tsx`）以 prop 收条目与项目的 id（今天读 `useParams()`，`:28`、`:39`），标题经 `ModalTitle`（`:77`）；列表是弹窗的内容（`HO/M3-closeout.md:52`）；候选来自 M4 的搜索，不含条目自己，不含已归档的（G2，已在 P2a spec §2.5 `:83` 登记）；发 `{ status: "duplicate", duplicate_to_id }`。
- **删除**：确认弹窗（`IN/modals/delete-issue-modal.tsx`）→ `deleteIntake`；成功以 `replace` 清选中（之后自动选第一行，同 Plane）；只给待处理与暂缓（Q33）。
- **之后的选中**（Q30，R14）：接受、拒绝、暂缓、改暂缓、取消暂缓确认之后，选中以 `replace` 移到动作开始时那一行的下一行，没有下一行则上一行，都没有则清选中（Plane `inbox-issue-header.tsx:116-135` 同样在动作开始时取邻行，因为确认之后那一行可能已移出）；接受另照上面在弹窗的提交落定之后才移；标重复不移；被拒不移；选中在等待时已被换掉则不移。
- **在途**：动作按钮在途时停用；弹窗在途时不能关；弹窗取消时焦点回到打开它的按钮（窄屏页头的菜单里打开的，回到菜单的按钮）。
- **焦点的落点**（R35，re-94 I4；任何情形都不落到 `body`）：
  - 选中移到邻行 → 那一行（列表里的链接）；
  - 选中被清空（没有邻行、删除、读不到）→ 列表（没有行时是列表的空态区域）；
  - 选中没有移、打开者已不在（标重复成功；分诊被冲突拒绝之后；分诊在轮次里重读，重读在途时页头随正文撤下，所以任何分诊之后原来的按钮都不在了）→ 新页头里这个状态的主动作；没有主动作（已处理）→ 详情区域；
  - 选中在等待时已被用户换掉 → 不动焦点（用户已在别处）；
  - 被拒（通用的拒绝）→ 弹窗里（字段错误时是那个字段，`useRefusedFocus`）；没有弹窗的动作（取消暂缓）被拒 → 回到那个按钮；
  - 落点在命令落定、页头重新画出之后取得（不在 `beforeFormSubmit` 里，也不在重读在途时）。
  - P4 的教训：停用的按钮收不回焦点；Headless UI 的菜单项在指针按下时取得焦点而菜单的按钮不取（P4 阶段记录 `reviews/P4-web-notifications-review.md:166`、`:173`，`03a6acf1`、`3df03b12`），所以键盘、指针与窄屏菜单三条路分别看（§2.12 的 H20）。
- **失败**：每个拒绝经 `followInSession` 之后由 `useRefusalToast`（`C/hooks/use-refusal-toast.ts:28`）提示；字段错误（`must_be_future`）在字段下以 `FieldError`，焦点经 `useRefusedFocus`（`C/lib/refused-focus.ts:29`）；不 `console.error`。

### 2.9 属性与“重复于”

- **属性**（`IN/content/issue-properties.tsx`，217 行）：负责人、优先级、截止日、标签；可改的规则同 §2.7；状态一栏只在已接受时显示（Plane 同，`:76`），在收集箱中的不显示（Triage 不在目录里）。负责人的 `MemberDropdown` 在页面上核对（`HO/M3-closeout.md:47`，收集箱详情的那一个）。
- **重复于**（G1）：`intake.duplicate_to_id` 不为 `null` 时，经 §2.5 的一次 `fetchIssueCore` 显示 “重复于 WEB-12 · 名字”（编号用同一项目的 identifier），链接到它；读被拒或目标已删时不显示这一行，持有的项目列表不因这次读带上错误（re-94 m3）。合同在目标删除或看不到时给 `null`（`IY:2345-2352`）。Plane 的 `duplicate_issue_detail` 不在合同里。

### 2.10 删除与新建弹窗（Q35）

旧的收集箱层整套删除，不拆分：它的替代是新的 store 与 issue 的写队列。分三个 Task（re-94 m11），每个是一个提交，各自全绿（含 knip）；守卫先看红、与让它变绿的删除同一提交。

- **T6a 详情侧**：新的 `IN/content/`（`InboxContentRoot`、详情区域、页头的动作模型、状态标签、复制与打开）与改接线的四个弹窗；旧的 `IN/root.tsx` 与旧侧栏暂时照旧，渲染新的 `InboxContentRoot`。删除只被旧详情读的：`IN/content/` 的旧文件（含 `description-versions.tsx` 46 行、`inbox-issue-header.tsx` 470 行、`inbox-issue-mobile-header.tsx` 239 行）、`C/services/inbox/` 的版本 service、它的 DTO 与测试和 `index.ts` 的那一行、`label/root.tsx` 的 `isInboxIssue`、`C/components/issues/fake-issue-properties.tsx:30-32` 对旧 hook 的 mock、三个 `*_permission` 键、`K:3262-3268` 的例外、`.oxlintrc.json:165` 的那一行。
- **T6b 列表侧**：路由、页面状态、侧栏、页头（从项目设置移来）、新的 `IN/root.tsx` 与上一条、下一条。删除 `IN/inbox-filter/` 整棵（十七个文件，含 `projectMemberIds` 的读者与 `applied-filters.test.tsx`）、`IN/sidebar/` 的旧文件、`IN/inbox-issue-status.tsx`、`C/components/projects/settings/intake/header.tsx`。
- **T6c 旧数据层**：`C/store/inbox/{project-inbox.store.ts 511, inbox-issue.store.ts 194, inbox-issue.request.test.ts 212}`；`C/services/inbox/` 余下的（`inbox-issue.service.ts`、`index.ts`）；`C/hooks/store/use-project-inbox.ts`；`RootStore` 的 `projectInbox`（`C/store/root.store.ts:27-28`、`:71`、`:105`），`root.store.test.ts:106` 21 → 20；`web/packages/types/src/inbox.ts`（99 行）与它在 barrel 的一行（`web/packages/types/src/index.ts:13`）；`TPaginationInfo`（`web/packages/types/src/common.ts:9`，唯一的读者是 `inbox.ts:10`）；`TLegacyIssueUpdateRequest`（`web/packages/types/src/issues/issue.ts:33-34`，读者全在旧收集箱，`HO/M4-closeout.md:63-64`）；`web/packages/utils/src/intake.ts`（40 行）与 barrel（`web/packages/utils/src/index.ts:16`）；`web/packages/constants/src/intake.ts` 的旧导出（`EPastDurationFilters` 与英文名字的 `PAST_DURATION_FILTER_OPTIONS`，`:74-101`，旧的状态与排序选项）。`findHowManyDaysLeft` 留用（新的状态标签读它）。三个包里任何随之失去最后读者的导出一并删（死成员的整份比对兜底）。
- **新建弹窗（T6c）只改到经新接口提交所必需的**（re-94 I3）：
  - `IN/modals/create-modal/create-root.tsx`：提交（`createInboxIssue`，`:61`、`:136`）改为 `root.intake.submit`；请求体照 `IntakeIssueCreate`（`IY:2353-2388`）：没有 `source` 与 `issue` 的外层，没选日期时不带 `target_date`（今天 `:132` 发 `null`，C10）；成功之后照旧跳到新条目，读回答的 `id`（今天 `:140` 读旧回答的 `res?.issue?.id`），仍是 `push`（用户的动作）；失败经 `useRefusalToast`，去掉 `console.error`（`:153`）与英文的失败提示（`:157`），因为拒绝的形状随新 service 变了（`ApiError`）。成功的英文提示（`:149`）与“编辑器仍在处理”的英文提示（`:89`、`:121`、`:215`）留给 P5b。
  - `IN/modals/create-modal/issue-description.tsx`：去掉对旧 hook 的读（`:21`、`:43`）与它的那段占位（`:48` 起的 `loader === "issue-loading"` 分支）。
  - 两份测试：`create-root.request.test.tsx` 改写为断言新的请求与回答之后的跳转、关闭、列表的重取；`create-root.files.test.tsx:144-152` 的断言是旧路径的特征（旧地址、旧请求体 `{issue: …}`、“没有任何生成客户端的请求”），改接之后不再成立，改为新地址与新请求体、“生成客户端只有那一个 `POST`”；它钉的性质（发出的描述不含图片、没有文件请求）不变（plan 的允许清单逐条写明）。
  - 其余（关闭策略、`ModalTitle`、`FieldError`、文档级 Escape、`getTabIndex`）不动，属 P5b。

### 2.11 守卫与告警

- **`old-intake-urls`**（新规则，phase `M7/P5`，T6c）：web 的源文件（测试也算）不得出现 Plane 的收集箱地址：`/api/workspaces/…/(inbox-issues|intake-issues|intake-work-items|inboxes)`（照 `old-notification-urls` 的写法与 `/api/workspaces/` 前缀锚定，`K:3096`）。hit 样例取旧 service 的地址（`C/services/inbox/inbox-issue.service.ts:18`、`:29`、`:54`，`intake-work_item_version.service.ts:22`、`:37`）；miss 样例含 `/api/v0/projects/{project_id}/intake-issues`、`/api/v0/intake-issues/{issue_id}`、`…/triage`、`/api/v0/workspaces/{slug}/intake-counts`、`/${workspaceSlug}/projects/${projectId}/intake` 与 `@/components/inbox`。它也命中 `create-root.files.test.tsx:145` 与 `create-root.request.test.tsx:142`，两处在同一提交改写（§2.10）。`old-issue-urls` 的两个 miss 样例（`K:2961-2962`）不动。
- **地址规则**：`project-address-route` 收入 `issue-write-confirm.ts`（T5）与读项目的新文件（T6a、T6b，`HO/M3-closeout.md:61`）；`workspace-address-route`（`K:2785`）收入读工作区的新文件（计数的 hook、页头）；每次带样例，改后跑 `node tools/keywords.mjs`。
- **`copy-hook`**：删 `K:3262-3268` 的例外（T6a）。
- **swr 名单**：`.oxlintrc.json` 的 `no-restricted-imports` 覆盖（文件名单 `:121-203`）收入每个新的取数文件，删去被删的。
- **oxlint**：上限 31（`web/apps/web/package.json:10`）是等式；新文件零告警；删掉的文件若带告警，同一提交照实降上限。
- **文案**：新键 en 与 zh-CN 同一提交，不许一个键是另一个键的前缀；三个 `*_permission` 键删除；不再有读者的旧 `inbox_issue.*` 键只在 T9 删一次，人工逐个 `grep`，台账列出。
- **死成员**：T2 的基线、T6a、T6b、T6c 的整份比对、T14 的终表（P4 阶段记录 §5 `:112`：P5 的 39 行）。

### 2.12 e2e

- **配置**：沿用 `e2e/playwright.m7-pages.config.ts`（`testMatch` `/collaboration\/h\d+(?:-h\d+)?-[\w-]+-pages?\.spec\.ts$/`、`grep` `/H\d+(?:–H\d+)? \(page\)/`，三引擎），不改；CI 照常以默认配置在 Chromium 跑一次。检查点与终检跑四个页面配置：m7-pages、m6-pages、m6-acceptance、m4-detail（re-94 m5，台账的 `final.sh` 已是这四个）。
- **挂载清单**（Q10，R7）：`e2e/fixtures/mounts.ts` 加条件化的 `INTAKE_COUNTS = ["GET /api/v0/workspaces/{slug}/intake-counts"]`（先例 `PROJECT_CYCLES` `:42`），头注释（`:5-16`）改写；`WORKSPACE`（`:24-30`）不变。读 `mounts.ts` 的文件（`h2-h4-notification-cards-page`、`h5-notification-list-page`、`g3`、`g8`、`g13`、`p2-visibility`、`s2-issue-list-removal`、`s2-web-app`）只经常量跟随；其中只有 `h2-h4-notification-cards-page.spec.ts` 打开收集箱，它的精确请求表在通知页（`:237-244`），通知页不挂应用侧栏（`A/_sidebar.tsx:27`、`:32`），所以不读计数。
- **夹具**：新 `e2e/fixtures/intake-pages.ts`（世界、地址、定位器；共用的帮手从 `cycle-pages.ts`、`notification-pages.ts`、`settings-pages.ts`、`browser.ts`、`dialogs.ts`、`auth.ts` 导入，`cycle-pages.ts` 388 行不加长）；新 `e2e/fixtures/assert/intake.ts`（条目行、动态行与通知行的只读查询）；造数据只经 `e2e/fixtures/intake.ts` 的接口帮手。
- **故事**（都在 `e2e/stories/collaboration/`，每个 `test()` 的标题都带 `H<n> (page)`）：

| 文件 | 标题 | 覆盖 |
|---|---|---|
| `h9-intake-list-page.spec.ts`（至少三个 `test()`，re-94 m9） | `H9 (page)` | ①空的收集箱：直接打开时的精确请求表（`APP`、`WORKSPACE`、`PROJECT`、`PROJECT_MEMBER`、`INTAKE_COUNTS` 与列表的读；空的收集箱没有自动选中，所以没有详情的读）；入口对访客与成员；收集箱关着时的停用页、没有列表的读。②51 条（多于一页）：两个页签与“未处理”的总数；滚动分页；自动选中第一行之后详情的读另以 `expect.poll` 断言（工作项、表情、链接、动态、评论、子工作项、关联、状态、描述版本）；只看自己的访客只看到自己的；侧栏的待处理数在一次分诊前后。③状态、优先级、负责人、创建者、标签、日期筛选与排序各一次，断言精确的查询 |
| `h10-intake-triage-page.spec.ts` | `H10 (page)` | 管理员经弹窗接受并改一个字段（断言 `PATCH /issues` 的请求体；之前在应用内打开过的项目列表回去时有它、不刷新；一条状态动态；关注者的通知行；弹窗关上之后才移选中）；接受被拒：弹窗开着时另一个管理员经接口先接受，再提交，没有 `PATCH /issues`、提示并重读；拒绝；暂缓到一个时刻（固定时区）、改暂缓与取消暂缓；经选择器标重复；之后的选中；成员与访客没有分诊控件；项目列表里没有未接受的条目 |
| `h11-intake-content-page.spec.ts` | `H11 (page)` | 管理员、成员提交者、访客提交者（只改名字与描述）、非提交者的成员只读、已处理只读；评论与表情；访客看不到描述版本，管理员恢复一个版本走 `PATCH /api/v0/intake-issues/{id}`；按角色的删除（访客提交者没有，已处理没有）；“重复于”的一行；项目列表里没有未接受的条目 |
| `h20-intake-pages.spec.ts`（超过 400 行时另拆 `h20-intake-late-pages.spec.ts`） | `H20 (page)` | 键盘、读屏与焦点：页签、筛选菜单（有名字的按钮、`aria-expanded`、Escape，`HO/M3-closeout.md:48`）、排序菜单、动作菜单、四个弹窗（名字、焦点、背后不可交互、Escape、在途锁）、重复选择器；**标重复、暂缓、改暂缓各走键盘一次、指针一次、窄屏页头的菜单一次，成功与取消各看一次焦点**，成功的那次扣住回答、放行并等页头重画之后再断言落点（R35）；↑、↓ 不在编辑器里时才换条目；窄屏的开关；换账号、换工作区、换项目与后退前进时扣住的回答不写进新页面；复制失败（`refuseClipboardWrites`） |

- **时长**：本机 chromium 每个文件约 8 秒为先例的上限（P4）；H9 与 H20 会超，写明原因；任何一个 `test()` 单引擎超过 30 秒先报主控（所以 H9 拆成三个 `test()`）。CP2 之前照 P4 spec §2.11 的方法估出 e2e 两片的时长。

### 2.13 文件与职责

| 文件 | 职责 | Task |
|---|---|---|
| `C/services/issue/intake.service.ts` | 六个操作 | T2、T3、T5、T6c |
| `C/store/issue/intake/filters.ts` | 筛选、键、查询、`matches`、日期区间 | T3、T4、T6b、T7 |
| `C/store/issue/intake/status.ts` | `effectiveStatus`、`inIntake`、`rowOf`、暂缓的日与时刻 | T4、T5、T8 |
| `C/store/issue/intake/intake.store.ts`（过 300 行把确认拆到 `intake-changes.ts`） | 计数、列表、确认、提交 | T2–T5、T6c |
| `C/store/issue/intake/fake-intake.ts` | 测试数据，随第一个测试读者逐个加 | T2–T5 |
| `C/store/issue/issue-write-confirm.ts` | 搬来的 `write()`；收集箱的去处 | T5 |
| `C/store/issue/issue-writes.ts`、`issue-write-fields.ts`、`issue.store.ts`、`root.store.ts` | 两个新命令与端点的选择；`intakeUpdateBody`；`choose` 的行规则（R33）；`intakeApi`、`intake` | T2、T5 |
| `C/hooks/store/use-intake.ts`、`C/hooks/use-intake-counts.ts` | 取到 `root.intake`；计数的取数与门 | T2 |
| `C/lib/intake-entry.ts` | `intakeEntryOn` | T2 |
| `IN/content/{index,root,detail,issue-properties}.tsx`、`IN/content/header/{root,mobile,status-chip}.tsx`、`use-intake-actions.ts` | 详情区域、页头与动作模型 | T6a |
| `IN/modals/{decline,delete,snooze}-issue-modal.tsx`、`select-duplicate.tsx` | 分诊的弹窗（T6a 只改接线，T8 落规则） | T6a、T8 |
| `IN/modals/accept.tsx` | 接受弹窗的接线（R12、R29、R35） | T8 |
| `A/projects/(detail)/[projectId]/intake/{layout,page}.tsx`、`IN/page-state.tsx`、`IN/page-header.tsx`、`IN/root.tsx`、`IN/use-intake-list.ts`、`IN/sidebar/{root,tabs,list,list-item}.tsx`、`IN/content/header/navigation.tsx` | 列表侧 | T6b |
| `IN/modals/create-modal/{create-root,issue-description}.tsx` | 经新接口提交（§2.10） | T6c |
| `IN/filters/{menu,applied,order-by}.tsx` | 筛选与排序（旧的筛选树在 T6b 删除） | T7 |
| `IN/content/duplicate-of.tsx` | 重复于 | T9 |
| `…/activity/actions/inbox.tsx`、`activity-list.tsx` | 按名字的动态 | T9 |

## 3. 局部裁定

主控的裁定在台账 `rulings.md`；下表把每条落在 P5a 的裁定对到本 spec 的 R。标 “(P5b)” 的五条不在本段：它们只进 §1 的“不做”与 §5 的后续。R27–R32 是起草时为让裁定能落地而定的细节，主控已同意（`rulings.md` “起草时的补充裁定”）；R33–R35 来自预检 re-94 的主控裁定。

| Q | R | Q | R |
|---|---|---|---|
| Q1、Q2 | R1 | Q16 | R15 |
| Q32 | R2 | Q15 | R16 |
| Q3 | R3（L!） | Q17 | R17 |
| Q28 | R4（L!） | Q18 | R18 |
| Q8 | R5 | Q31 | R19 |
| Q9 | R6 | Q33 | R20 |
| Q10 | R7 | Q11 | R21 |
| Q29 | R8 | Q35 | R22 |
| Q7 | R9 | Q36 | R23 |
| Q19 | R10 | Q27、Q37、Q38 | R24 |
| Q4 | R11 | 检查点与门禁、O3 | R25 |
| Q5 | R12 | O1、O2 | R26 |
| Q6 | R13 | Q12、Q13、Q14、Q34、Q21 | —（P5b：§1、§5） |
| Q30 | R14 | re-94 m2、I1、I4 | R33、R34（L!）、R35 |

**预检 re-94 的处理**：S1 → R3、§2.4；I1 → R34、§2.4；I2 → R12、R14、§2.8；I3 → §2.10；I4 → R35、§2.8、§2.12；m1 → R28、§2.4；m2 → R33；m3 → §2.5、§2.9；m4 → §2.6；m5 → §2.12、R25；m6 → plan 的“knip 与第一个读者”；m7 → plan 的性质表；m8 → §2.3、§7；m9 → §2.12；m10 → §2.6、§2.8、R14、R29；m11 → §2.10、plan 的 T6a–T6c；O1 → §2.7；O3 → R25；O5 → §2.2；O6 → R31；O2、O4 不改。

- **R1 两段与 P4 的回填**（Q1、Q2）：§1。分支 `codex/m7-p5a`，台账 `.superpowers/sdd/P5a-web-intake/`；关键词 phase `M7/P5`，两段共用。T1 补 P4 的阶段记录（状态行、A8、§5 的最终分支 CI 45048776015 与 main CI 45051719903、§9），数字取自归档台账（`.superpowers/sdd/M7-P4-integration-20261123/payload/worktree/.superpowers/sdd/P4-web-notifications/progress.md:602`）；设计 §12、§14 的行由主控改。
- **R2 代码的位置**（Q32）：§2.1、§2.3、§2.7。service 在 `C/services/issue/`；intake store 是 issue 根 store 的子 store（设计 §15 A：收集箱放进 issue 模块），因为它的写经 issue 的写队列（R3）；页头从项目设置的目录移到 `IN/`。
- **R3 写经 M4 的逐工作项写队列**（Q3，**L!，高风险**，阶段记录写明）：§2.4。不另开队列；先纯搬移拆分 `issue-writes.ts`，既有测试不改；端点在轮到时按所持详情的 `intake` 选；收集箱的去处不经 `write()` 的两段目录准备、不写项目列表（re-94 S1）。代价：M4 每个工作项的写都经过改动的文件，所以 T5 之后另跑一次 chromium 的 `m4-detail`，CP1 跑整套 e2e。
- **R4 接受像新建一样**（Q28，**L!**）：§2.4。接受经 `confirmCreation`（`before` 为 `absent`）把重读的回答确认进所持的项目列表，不用刷新就出现；拒绝、暂缓、标重复、回到待处理、删除只动收集箱的列表与计数。代价：接受之后项目列表多一行而它的行证据来自新建的路径；单测钉住“接受的条目在持有的项目列表里出现，且只出现一次”，含取数在外与列表的读在接受发出之后发出、在回答之前落定两种。
- **R5 页签与选中在地址、筛选与排序在页面状态**（Q8）：§2.6。离开即复位（同 P4 R3），换页签复位（同 Plane）；§7 登记。
- **R6 无限滚动与每页 50**（Q9）：§2.3、§2.5。保留 Plane 的哨兵；下一页经 `fetchPart` 与认游标的合并；每页 50 是合同的默认（Plane 是 10，`pw/core/store/inbox/project-inbox.store.ts:87`，是界面常数）。H9 (page) 因此提交 51 条。
- **R7 待处理数的门与键**（Q10）：§2.5。只在调用者的项目里有开着收集箱的（未归档、在职）时读，键带这些项目的 id，开关一变即重读；条件化的 `INTAKE_COUNTS` 挂载清单，`WORKSPACE` 不变。侧栏只在项目展开时渲染项目的导航，所以取数放在项目列表（`projects-list.tsx`）调一次，导航项只读 store；写后的 `countAgain` 也不越过这个门。
- **R8 计数的新鲜度**（Q29）：本人的提交、分诊、删除之后重取；没有聚焦重取（v0 §7.7）。与 P4 R6 同类，进 M7 收官报告（O2）。
- **R9 每个页签显式发 `status`**（Q7）：§2.2 的 C1。状态筛选至少留一个（Plane `pw/core/components/inbox/inbox-filter/filters/status.tsx:44-47`）。
- **R10 日期预设**（Q19）：§2.2 的 C3。本地日的预设发成 UTC 日（P2b R8，已登记）；预设的名字经 `t()`（今天英文字面量，`web/packages/constants/src/intake.ts:81-101`）。日期筛选每个字段只一个区间（Plane 的日期筛选也是单选）。
- **R11 只有管理员分诊**（Q4，设计 §15 B `D:357`）：§2.8 的表；三个 `*_permission` 键删除；§7 登记。
- **R12 接受弹窗**（Q5，re-94 I2）：§2.8。分诊被拒不发 `PATCH /issues`；`beforeFormSubmit` 只分诊，不关弹窗、不移选中，二者在弹窗的提交整个落定之后；状态字段从默认状态起；接受之后的修改被拒在字段下说明（G3，合同不改）；冲突的拒绝之后弹窗关上。
- **R13 暂缓到某日**（Q6）：那天本地的第一个时刻（通常 00:00；午夜被跳过的时区是那天第一个存在的时刻），最早明天，默认明天；§7 登记。
- **R14 分诊之后的选中**（Q30，re-94 m10）：§2.6、§2.8。邻行在动作开始时取，确认之后才移（接受在弹窗的提交落定之后），以 `replace` 移；标重复、被拒、选中已换都不移；取消暂缓也移（同 Plane，主控已同意）。
- **R15 提交者**（Q16）：访客提交者只改名字与描述，没有删除（合同 `IY:975-977`、`:1026-1030`；Plane 让创建者删，`pa/app/views/intake/base.py:552-553`）；成员提交者可删待处理与暂缓的；§7 登记。
- **R16 描述版本**（Q15）：复用 M4 的 `IssueDescriptionVersions`，访客看不到（plane-diff `docs/v0/plane-diff.md:362` 已登记）；恢复经 R3 走收集箱的端点。
- **R17 重复于**（Q17，re-94 m3）：§2.9。一次 `fetchIssueCore`（G1 的代价），不经核验的入口；选择器以 prop 收条目 id（`D:248` 的交接项）。
- **R18 动态按名字**（Q18）：§2.7；未知的值用通用的键（re-94 O1）。
- **R19 收集箱关着**（Q31）：保留 Plane 的停用页，不读列表；合同在关着时照常可读（`IY:752-753`），所以这是页面的选择，§7 不登记（同 Plane）。
- **R20 已处理的不给删除**（Q33）：同 Plane 的界面（`pw/core/components/inbox/content/inbox-issue-header.tsx:365` 起的已处理分支只有复制与打开）；合同允许删任何状态（`IY:1023-1057`）。
- **R21 入口规则**（Q11，设计 §3.11 `D:157-159`，M6-M7-07）：`C/lib/intake-entry.ts` 的 `intakeEntryOn(project, role)` = 开关开着且调用者有项目角色（管理员、成员、访客），不排除已归档（收集箱在已归档的项目照常可读，`IY:752-753`；`planningEntryOn` 另排除已归档，`C/lib/planning-switch.ts:62-71`，这里不仿）。侧栏（`C/components/workspace/sidebar/project-navigation.tsx:108-117`）、页签栏（`C/components/navigation/use-navigation-items.ts:76-84`）、命令面板（`C/components/power-k/config/navigation/project-commands.ts:191-202`）共用它；单测钉住管理员、成员、访客、没有角色、关着、已归档六格。命令面板的特征测试一行（`commands.test.tsx:139`，页面 `workspace, admin, guest, off, archived`，`:173`）不变，它没有成员一列，所以另写单测。已归档的项目不给“Add work item”（提交答 `project.archived`，`IY:888`）。
- **R22 旧 store 删除替换、页头按关注点拆**（Q35，设计 `D:245` 的“先拆 … 511 行的 store”）：§2.10、§2.7。
- **R23 `getTabIndex` 与 `projectMemberIds`**（Q36）：P5a 去掉 `filter-selection.tsx` 对 `projectMemberIds` 的读（T6b 随旧筛选树删；T7 的成员筛选从路由项目的显式 id 取，`getProjectMemberIds`）；`getTabIndex` 的四个读者随 P5b；两者的最终删除在 P6a（`HO/M4-closeout.md:73-88`，`HO/M6-closeout.md:62-68`）。
- **R24 三项交接关闭**（Q27、Q37、Q38）：设计胜：项目设置里收集箱的说明不改（`D:97`，只有应用内提交），关闭 `HO/M4-closeout.md:27` 的那一子项；`HO/M2-closeout.md:41` 的收集箱状态下拉已不存在（M3/P8b `ba8cb1f4` 删除），收集箱一半按不适用关闭；命令面板没有收集箱的搜索结果（Plane 也没有，`D:245`），关闭 `HO/M4-closeout.md:65` 的该项。
- **R25 检查点与报告线**（re-94 m5、O3）：三个检查点在 T2、T9、T13 之后：本机七项、`m7-pages` 三引擎、`m6-pages`、`m6-acceptance`、`m4-detail`、分支 CI。第 1 路：以 P5a 检查点 0 的本机第 1 路为基线，超过 +30 秒停下查；CI 的八成只报告。web 作业 960 秒、e2e 每片 30 分钟是报告线（O3）。**判别慢 runner 与真实的增长**：web 作业越线时逐步对照同一次 main CI（45051719903）的 Lint、Unit tests、Build 三步：三步同比例变慢是 runner（照报告、不停）；只有 Unit tests（或只有新文件所在的一步）变长才是本段的增长，先报告再继续。
- **R26 不改合同**（O1、O2）：G1（一次额外的读）、G2（已登记）、G3（两个请求，有窗口）在 §7 登记；计数的实时性进收官报告。
- **R27 已归档的项目只读**：合同对已归档项目的分诊、修改、删除都答 `project.archived`（`IY:950`、`:1007`、`:1052`），所以已归档的项目里收集箱页只读：不给分诊、删除与修改，与 M4 的详情一致（`C/components/issues/issue-detail/root.tsx:64-67`）。
- **R28 冲突的拒绝之后重读**（re-94 m1）：§2.4。重读在被拒那次写的轮次里、交出拒绝之前；给 `undefined` 时从收集箱的列表移出。页头与列表随之回到 nerve 此刻的状态，不让用户对一个已处理的条目再点一次。
- **R29 接受之后再提交只发修改**（G3 的窗口，re-94 m10）：所持详情已不在收集箱时，`beforeFormSubmit` 不再分诊，不论是本人还是别人接受的；否则那次提交会答 `issue.intake_transition_not_allowed` 而修改永远发不出去。
- **R30 提交不排队、写后重取列表**：提交同 M4 的新建（`issue-writes.ts:228-238` 不经逐工作项的队列）；新条目不经 `confirmCreation`（那会把收集箱条目确认进项目列表，见 §2.4 的理由二），只重取这个项目持有的收集箱列表的第一页与计数；成功之后跳到新条目（读回答的 `id`，§2.10）。
- **R31 列表的总数**（re-94 O6）：只有选中的“未处理”页签显示数（同 Plane）；它是列表回答的 `total`，确认把一行移出时减一，替换不变，不插入所以不加。只对持有的行准：经地址直接选中一个不在已取各页里的条目并分诊，确认时没有行可移，`total` 不减，到下一次取数才对（与“不插入”同类的代价）。
- **R32 通知嵌入的两个 prop**：`isNotificationEmbed`、`embedRemoveCurrentNotification`（`IN/content/root.tsx:28-29`，P4 起没有传入者，P5 的死成员之一）随 T6a 的改写删除，P5b 的嵌入与它的读者一起加回。`InboxContentRoot` 仍从 `@/components/inbox/content` 导出：产品读者是 `IN/root.tsx`，`C/components/workspace-notifications/root.test.tsx:58` 把它 mock 掉，`:215-221` 那一格（普通浮层、没有分诊）在 P5a 照旧通过，由 P5b 改。
- **R33 行盖在详情上**（re-94 m2）：§2.4。列表的行落在 `intake` 未被接受的详情上时，合并结果的 `intake` 取已接受；`inIntake(undefined)` 为假。代价：改 M4 的 canonical 合并一处（`issue.store.ts:27-35`），只在所持详情的 `intake` 未被接受时起作用（P5a 之前这样的详情不进 canonical）。
- **R34 分诊在轮次里重读**（re-94 I1 甲，**L!，高风险**，与 R3、R4 同列）：§2.4。五种分诊都在它的轮次里以 `refreshAuthoritative(id, { afterWrite: true })` 重读，canonical 的最终值来自重读；接受另以 `confirmCreation` 把重读的回答确认进持有的项目列表。代价：每次分诊多一个 `GET`；核验把覆盖这个项目、按 `issueMembership` 算出这一行会在的项目列表的格标成陈旧，回到项目列表时多一次那一格的重读，重读失败时那些列表带上这次的错误（既有机制，re-94 O2）；重读在途时详情的正文被撤下，所以分诊的弹窗画在正文之外。`issue-changes.test.ts` 的三条 `it.each` 的断言照旧。
- **R35 焦点的落点**（re-94 I4）：§2.8 的落点表；任何结局都不落到 `body`；H20 (page) 以键盘、指针与窄屏菜单三条路核。

**实施中的补记**（2026-11-23，T14 写；来自主控在 T5、T8、T12 的裁定，经过见[阶段记录](../reviews/P5a-web-intake-review.md) §3）：
- **非接受的分诊之后**（T5 的主控裁定 3）：拒绝、暂缓、改暂缓、取消暂缓与标重复在工作项的队列里依次：发出分诊，把回答确认进这个项目持有的收集箱列表，在轮次里重读工作项，重取这一代持有的计数（`C/store/issue/issue-write-intake.ts:36-48`）；重读给出详情时再读详情的动态与评论（`root.issueDetail.refreshAfterChange`，`C/store/issue/issue-writes.ts:196`；它重读动态与评论两个 store，`C/store/issue/issue-details/root.store.ts:149-154`），重读给 `undefined` 时不读。接受不走这一步（它的重读的回答由 `confirmCreation` 确认进项目列表，`issue-writes.ts:181-191`）。单测 `C/store/issue/intake/intake-writes.read-again.test.ts:205-247`，变异 `t5-mut-triage-no-detail-refresh`。
- **删除在轮次里重读**（R34 的删除一半，T5 的主控裁定 5）：`deleteIntake` 在工作项的队列里发出 `DELETE /intake-issues/{issue_id}`，做成之后先把它从这个项目持有的每份收集箱列表移出（收集箱的行已没了），再在轮次里重读（`C/store/issue/issue-write-intake.ts:57-64`）。canonical 只在重读读不到、且核验记它为 missing 时删除：`refreshAuthoritative` 对 404 与被超过、未定都给 `undefined`，核验只为 404 记 missing（`C/store/issue/issue-verification.ts:44-47`、`:243-256`），被超过时 canonical 交给超过它的那次读写。重读读到详情（另一位管理员在这之前接受了它，nerve 只删收集箱的行，`IY:1023-1057`）时 canonical 取那份详情。不碰项目列表；计数再取。代价：删除多一个 `GET`，核验把覆盖这个项目的列表格标成陈旧（同 R34）。单测 `intake-writes.read-again.test.ts:299-344`，变异 `t5-mut-delete-no-reread`、`t5-mut-delete-overtaken-deleted`、`t5-mut-delete-drops-neither`。
- **R12 的落地**（T8 的主控裁定 A）：`base.tsx` 对已有的工作项的更新从不调 `onSubmit`（`C/components/issues/issue-modal/base.tsx:216-221`；`update-command.tsx:60`、`:98` 交回 `undefined`），所以动作模型不在 `onSubmit` 记下：它在接受弹窗的 `beforeFormSubmit`（动作模型的 `accept`）分诊做成、或 R29 见所持详情已不在收集箱时，记下“已接受”与那时的邻行（`IN/content/header/use-intake-actions.ts:268-281`）；此后弹窗的 `onClose`——确认的修改之后 `finishClose` 的那一次（`base.tsx:234`），或用户的取消——都以 `replace` 移选中（`use-intake-actions.ts:261-266`；提交在途时 `isBusy` 挡住关闭，`base.tsx:308`）。修改被拒时弹窗留着，标题说条目已被接受、修改未保存（`IN/modals/accept.tsx:54-56`）；用户这时取消，选中也移走：条目已被接受，已离开“未处理”页签。单测 `IN/modals/accept.test.tsx:206-218`；变异 `t8-mut-move-on-update-refusal`（与 `t8-mut-move-in-before-submit` 同一处改法，由修改被拒的一格杀）。
- **R27 在页面上**（T12 的主控裁定）：收集箱页的路由在项目的布局之下（`web/apps/web/app/routes/core.ts:102` 的 `[projectId]/layout.tsx` 包着 `:165-170` 的 `intake/layout.tsx`），已归档的项目在那里由 M3 的 `ProjectAuthWrapper` 换成 “This project is archived”（`A/projects/(detail)/[projectId]/layout.tsx:57`，`C/layouts/auth-layout/project-wrapper.tsx:61-62`，`use-project-fetch.ts:91`），项目的每一页都如此：收集箱页不渲染、不读列表、没有详情。收集箱自身的只读规则——详情的 `editableBy` 不给已归档的项目改（`IN/content/detail.tsx:44-57`），动作模型对已归档的项目只给复制与打开（`IN/content/header/use-intake-actions.ts:108-116`）——在页面上走不到，由 T6a、T9 的单测与变异钉住；H11 (page) ④ 断言归档之后打开收集箱是 M3 的归档页、不读列表。

## 4. 验收标准

- [ ] **A1 数据层**：service 只调 `/api/v0`，每代以自己的客户端构造；store 是 issue 根 store 的子 store，计数按工作区 id 存、列表按项目与筛选键存；取数经 `useSessionSWR`、键带 `loginId`（与 `identity`）、fetcher 只读键；`fetchPart` 与认游标的合并；§2.2 的 C1–C10 各有单测与变异；页面状态不在 store。
- [ ] **A2 写的接缝**：`issue-writes.ts` 先纯搬移拆分、既有测试不改；内容修改、版本恢复、分诊、删除都在工作项的那一条队列上，与评论的写一个接一个；端点在轮到时选；收集箱的去处不经两段目录准备、不写项目列表（持有带状态组筛选与按状态组分组的列表时照样发出、列表的值与错误不变）；分诊在轮次里重读，写之前发出的旧读不改回 `intake`；接受像新建一样进持有的项目列表、只一次；拒绝值不变，`issue.intake_not_found` 移出，冲突在轮次里重读；行盖在详情上不留未接受的 `intake`；三条 `it.each` 照旧；每条有能让测试失败的变异。
- [ ] **A3 列表与筛选**：两个页签、总数、七种筛选、排序、已选筛选的一行、滚动分页、空态、读失败可重试；每个请求的查询照 C1–C6；换页签复位；离开复位；动作引起的选中变化是 `replace`。
- [ ] **A4 详情**：名字、描述、表情、动态（按状态名与通用键，en 与 zh-CN）、评论、描述版本（访客看不到）、属性按角色与状态、“重复于”（不碰项目列表）；FORMS 分支与模块级 service 删除；读不到在这次挂载自己的读落定之后才清选中，别的失败可重试；分诊的弹窗不随重读撤下。
- [ ] **A5 分诊**：只有管理员看到；接受弹窗：只在 `beforeFormSubmit` 分诊，修改落定之后才关弹窗、移选中，R29，冲突关弹窗；拒绝、暂缓（最早明天、那天的第一个时刻）、改暂缓、取消暂缓、标重复（选择器以 prop 收 id、不列自己）、删除（按角色，已处理没有）；之后的选中；在途锁；焦点照 R35 的表，不落到 `body`；每个拒绝都提示，没有 `console.error`。
- [ ] **A6 侧栏与入口**：待处理数按 §2.5 的门与键读一次，大于 0 时显示，名字含数；`intakeEntryOn` 一处定义、三处使用，六格单测；已归档的项目没有“Add work item”。
- [ ] **A7 删除干净与守卫**：§2.10 全部（三个 Task 各自全绿）；新建弹窗经新接口提交、没有日期不带键、成功跳到新条目；`RootStore` 20；`old-intake-urls`；地址规则收入新文件；`copy-hook` 例外删除；swr 名单；oxlint 等于上限；死成员整份比对（P5 的 39 行各有去处）；文案键 en 与 zh-CN、三个权限键删除。
- [ ] **A8 e2e**：§2.12 的四个页面故事三引擎各一次、`--repeat-each=3`；`INTAKE_COUNTS`；既有故事不回归（检查点的 `m6-pages`、`m6-acceptance`、`m4-detail` 与整套默认 e2e）。
- [ ] **A9 收尾**：终检、最终分支 CI、合并、main CI 与归档，在 P5b 的 T1 补记（同 P4 A8 在 P5 的 T1 补记）。

## 5. 交接与后续责任

**本段处理的输入交接**：都写“处理结果（M7/P5a）”，照设计 §11 保持 open 至收尾，除非下表写关闭。T14 写。

| 交接 | 条款 | 本段 | Task |
|---|---|---|---|
| `HO/M4-closeout.md:14-27`（M7-01） | FORMS 分支与旧 `inbox-issue` 服务；设置的说明；`select-duplicate` 的真实页面；窄屏 | 推进：FORMS 与旧服务删除，选择器在 H10 (page) 走真实页面，窄屏在 H20 (page)；设置说明按 R24 关闭该子项；新建弹窗的其余随 P5b，本条在 P5b 关闭 | T6a、T6c、T8、T13 |
| `HO/M4-closeout.md:60-71`（M7-04） | 旧 DTO、`TLegacyIssueUpdateRequest`、版本 DTO/service 与 `inbox/content/description-versions` 的过渡；命令面板的收集箱结果；`frame-src`；关闭政策 | 旧 DTO、请求类型与版本的过渡删除；命令面板一项按 R24 关闭；`frame-src` 写明不嵌 iframe；关闭政策随 P5b | T6a、T6c |
| `HO/M4-closeout.md:73-88`（M7-05） | `projectMemberIds` 的收集箱读者；`getTabIndex` 的四个收集箱文件；新页面的弹窗名、字段提示、Escape、行名控件、翻译 | 收集箱读者去掉（R23）；`getTabIndex` 随 P5b；收集箱页照规则写；最终删除在 P6a | T6b、T7、T8 |
| `HO/M6-closeout.md:78-83`（M6-M7-07） | 命令面板的收集箱命令给访客；侧栏与命令面板一条规则、单测钉住 | 收集箱的一半关闭（R21）；视图的一半随 P6a | T2 |
| `HO/M6-closeout.md:62-68`（M6-M7-05） | `getTabIndex` 的最后删除 | 本段不涉及（P5b、P6a） | — |
| `HO/M3-closeout.md:14` | 侧栏的 `intake_count`、按权限 | 关闭：经 `getIntakeCounts` 按门读，挂载清单随之 | T2 |
| `HO/M3-closeout.md:33`（B） | 收集箱的弹窗与选中在以 id 为 key 的布局之下 | 关闭（§2.6） | T6a、T6b |
| `HO/M3-closeout.md:40-41` | 收集箱弹窗的 Tab、Escape、点外面、焦点、名字 | 拒绝、删除、暂缓、重复四个关闭；新建弹窗随 P5b | T8、T13 |
| `HO/M3-closeout.md:47` | 两个负责人下拉框在页面上核对 | 详情的一个关闭；新建弹窗的随 P5b | T9、T12 |
| `HO/M3-closeout.md:48` | `FiltersDropdown` 的按钮名、`aria-expanded`、Escape | 收集箱一个调用方关闭 | T7、T13 |
| `HO/M3-closeout.md:52` | 重复列表是弹窗的内容 | 关闭 | T8、T13 |
| `HO/M3-closeout.md:61` | 四个收集箱文件读 router store，收入 `project-address-route` | 关闭：新文件读路由参数并收入规则 | T6a、T6b |
| `HO/M3-closeout.md:63` | 页头的复制经 `useCopyLink`，删 `copy-hook` 例外 | 关闭 | T6a |
| `HO/M3-closeout.md:64-65` | 面包屑的 “Intake”；上一条、下一条的名字 | 关闭（en、zh-CN） | T6b |
| `HO/M3-closeout.md:68`、§7 `:93-105` | `projectMemberIds` | 收集箱读者去掉（R23）：旧的筛选树在 T6b 删除，T7 的成员筛选读显式的项目成员 id | T6b、T7 |
| `HO/M3-closeout.md:55`、`:56` | 标题长度提示、新建弹窗的文档级 Escape | 本段不涉及（P5b） | — |
| `HO/M2-closeout.md:29` | 两个单例；新键带 `loginId` | 收集箱的单例删除（`issue-root.tsx:36` 在 T6a，`project-inbox.store.ts:127` 在 T6c），键经 `useSessionSWR` | T6a、T6c |
| `HO/M2-closeout.md:41` | `dropdowns/intake-state/base.tsx` | 收集箱一半按不适用关闭（R24） | T14 |
| `HO/M2-closeout.md:50` | 收集箱页头复制被拒没有处理 | 关闭（`useCopyLink`） | T6a |
| `HO/M2-closeout.md:68` | 是否嵌 iframe | 写明：收集箱页不嵌 iframe | T14 |
| `HO/M1-closeout.md:12-24`、`:26-32` | 死成员；改到的文件告警为零 | P5 的 39 行各有去处；改到的文件零告警 | T2、T6a–T6c、T14 |
| `HO/M5-closeout.md:13-22`（M5-M7-01） | 收集箱的图片与附件、三浏览器本机与 S3 的证据 | 本段不涉及（P5b，Q21） | — |
| `HO/M6-closeout.md:12-19`（M6-M7-01） | 新建弹窗在编辑器建好之前拒绝关闭 | 本段不涉及（P5b） | — |
| P4 阶段记录 §5 `:112`、§7 `:147`；P4 spec `:478` | 收集箱的死成员；浮层的 “State” 占位；通知的两个 prop；收集箱的单例 | 死成员与单例本段；占位与两个 prop 随 P5b（R32） | T6a、T6c |
| P2a spec `:150`、P2b spec `:132`、P2c spec `:97`；P2c 审查 `reviews/P2c-intake-assets-review.md:65` | 网页接上新接口；详情按状态开图片与附件；多选的忙时守卫 | 网页接上（本段）；图片、附件与守卫随 P5b | T2–T9 |

**后续**：
- **P5b**：新建弹窗的其余（M6-M7-01、`HO/M3-closeout.md:40-41` 的新建弹窗一半、`:55`、`:56`、`getTabIndex` 的四个文件、弹窗里的英文提示）；详情的描述图片与附件（M5-M7-01，`[local]` 与 `[s3]` 三引擎，Q21）；链接（Q14，登记与 Plane 的差异）；通用详情与浮层在未接受时隐去状态、浏览页把未接受的条目重定向到收集箱页（Q12）；通知预览嵌入收集箱的内容、两个 prop 加回、`root.test.tsx:215-221` 改写、H2–H4 (page) 收集箱一步（Q13，登记“已拒绝的也嵌入”）；助手多选的忙时守卫（Q34：故事里快速连点，重现即在共用控件按根因修，否则写“未重现”交 M8）；P5a 的 A9 回填；M7-01、M7-04 的关闭。
- **P6a**：`getTabIndex`、`projectMemberIds` 的最终删除（R23）；M6-M7-07 的视图一半。
- **M7 收官报告**：R8 与 P4 R6（计数与未读点不实时）；R3、R4、R33、R34 是动了 M4 共用代码的局部裁定；G1–G3（R26）。
- **M8**：Q34 若未重现；P5a 发现的、不属本段的共用件问题。

## 6. 风险

- **高风险的局部裁定 Q3（R3）：收集箱的写进了 M4 的共用写队列。**
  - `IssueWrites` 是每个工作项写、每份列表确认的必经之路；拆分与新的去处一旦出错，坏的是全部 M4 的写。
  - 缓解：先纯搬移、既有测试一行不改（单独一个提交、单独的收据）；收集箱的去处是与 `write()` 并列的一条路，只取它的权限水位，不经两段目录准备、不碰项目列表（re-94 S1：持有一份带状态组筛选的项目列表或按状态组分组的工作区列表时，经 `write()` 走的收集箱写会在发出之前失败，删除会把列表整份标成失败）；单测以这两种列表钉住；T5 之后另跑一次 chromium 的 `m4-detail`，CP1 跑整套默认 e2e，终检同。
  - 隐患：项目列表的 `issueMembership` 不看 `intake`（`C/lib/issue-membership.ts:44-99`）。所以收集箱的确认绝不交给项目列表；`h10` 与 `h11` 的页面故事各断言一次项目列表没有未接受的条目。
- **高风险的局部裁定 Q28（R4）：接受改写持有的项目列表。** `confirmCreation` 的路径以 `before: absent` 让列表把它当新行；列表的读在接受之前、之中、之后到达都只能有一行（取数在外一格；列表的读在接受发出之后发出、在回答之前落定一格，`issue-writes.ts:141` 的来源冻结管的那一种；`Reconciled` 的重放，`C/lib/reconciled.ts:117`、`:126`）。H10 (page) 有一格。
- **高风险的局部裁定 R34（re-94 I1）：每次分诊在轮次里重读。** 它让 canonical 不被分诊之前发出的旧读改回，也让 `issue-changes.test.ts` 的三条 `it.each` 照旧成立；代价是多一个 `GET`、核验在项目列表上的副作用（陈旧的格、重读失败时的错误，re-94 O2），以及重读在途时详情正文被撤下：分诊的弹窗、详情区域与焦点的落点都因此放在正文之外（§2.7、R35）。
- **M4 的 canonical 合并改一处（R33）**：只在所持详情的 `intake` 未被接受时起作用；`issue.store.test.ts` 与 M4 的详情测试是回归。
- **大段的删除与新写**：T6a–T6c 删约 5,800 行旧代码（`research/scope.md` §6）。knip、`old-intake-urls` 与死成员的整份比对兜底；三个 Task 各一个提交、各自全绿，守卫先红后绿。
- **挂载清单**：`INTAKE_COUNTS` 只在有开着收集箱的项目时出现。规划与项目的故事有几处经接口打开收集箱（`e2e/fixtures/furnish.ts:134-150`、`:168-180`；`i15`、`i17`、`i21`、`p3`、`p4`、`w3` 等），它们按条件过滤请求，不读 `mounts.ts`；CP0 以整套默认 e2e 确认，任何一条红先停下查是不是精确请求表。
- **接受弹窗是 M4 的弹窗**：`beforeFormSubmit` 抛出即停（`base.tsx:214`）、`formScope` 的键含 `isOpen` 与条目 id（`:93-98`）是既有行为；状态字段的初值、R29 与“落定之后才移选中”只在收集箱传入，`close-policy.test.tsx`、`update-command.test.tsx` 是回归。
- **jsdom 查不出真实的焦点问题**（P4 的教训）：弹窗、菜单、↑/↓ 与 R35 的落点以 H20 (page) 三引擎为准，键盘与指针分开看。
- **e2e 与 web 作业的时长**：web 作业 20 分钟、960 秒报告线（P4 最终分支 CI 956 秒，慢 runner）；判别法见 R25；e2e 每片 35 分钟、30 分钟报告线；P5a 加四个页面故事文件与一批 vitest，每个检查点记时长，超报告线先报告。服务端第 1 路以 CP0 的本机值为基线，+30 秒停下查（P5a 不加 Go，变化只能来自环境）。

## 7. 与 Plane 的差异（T14 登记进 `docs/v0/plane-diff.md`，现有收集箱的行在 `:358-368`）

- 筛选与排序离开页面即复位、换页签复位；Plane 按项目存在会话里（R5）。
- 收集箱列表的行是页面自己的副本，只随本人的写与冲突之后的重读变，别人改了要等列表重读；Plane 的条目对象随它自己的写在本地改（同 P4 的通知列表，主控裁定）。
- 分诊、删除之后的选中变化不增加浏览历史（`replace`）；Plane 分诊之后 `push`（R14）。
- 每页 50（Plane 10）（R6）。
- 侧栏的待处理数：一个工作区级的读，只在有开着收集箱的项目时读，本人的提交、分诊、删除之后重取，不实时；Plane 用项目的 `intake_count` 并在本地加减（`pw/core/store/inbox/inbox-issue.store.ts:114-127`，`pw/core/components/workspace/sidebar/project-navigation.tsx:191-203`）（R7、R8；补进 `:367` 一行）。
- 只有管理员看到分诊控件，成员没有 “Permission denied” 的提示（R11）。
- 接受弹窗：分诊被拒即停；状态从默认状态起；接受与修改是两个请求，修改可在接受之后被拒（G3）；修改落定之后才关弹窗、移选中（R12、R29）。
- 暂缓到某日：那天的第一个本地时刻，最早明天；Plane 默认今天（R13）。
- 访客提交者属性只读、没有删除；Plane 的创建者都能删（R15）。
- “重复于”经一次读；选择器没有已归档的目标（G1；G2 已在 P2a 登记）（R17）。
- 动态按状态名，含“结束暂缓”；Plane 按数字（R18）。
- 入口一条规则；已归档项目有入口、没有“Add work item”、页面只读（R21、R27）。
- 冲突的拒绝之后重读（R28）。
- 没有 FORMS 来源与 “Intake Form user”（M7-01）。

日期预设按 UTC 日（P2b R8）与描述版本对访客隐藏（`:362`）已登记，不重复。
