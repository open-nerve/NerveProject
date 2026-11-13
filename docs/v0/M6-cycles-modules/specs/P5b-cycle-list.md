# M6 / P5b：迭代页的工作项列表

> 状态：整分支审查 Ready to merge（[re-67](../../../reviews/re-67-M6P5b整体审核.md)，修复轮 F 另经限定复审），待合入；A1–A7 已勾选，A8 由 P6 的 T1 勾选（[阶段记录](../reviews/P5b-cycle-list-review.md)）。预检通过（修正后，[re-66](../../../reviews/re-66-M6P5b预检.md)）。基线 main `0b8efcd701edf7e73a37d5bb82ad176a7795b910`（P5a 的 noff，main CI 42995581409 首次通过）。设计见 [M6 设计](../M6-design.md) §2 的 G15、G19，§3.3、§7、§8、§12；P5 的拆分与 P5b 的边界见 [P5a spec](P5a-web-cycles.md) §1、R1、R10、§5。网页的长期规则见[总体设计 §7.2、§7.7](../../v0-design.md#77-stores-按会话分代长期有效m2-起)。接口合同见 [P3a spec](P3a-issue-membership.md)（工作项的 `cycle_id`）、[P3b spec](P3b-list-engine.md) §2.1、§2.3、R6、R8、R9（列表的词表与迭代页偏好）。现状调研见本阶段台账 `research/web.md`（网页的列表引擎）、`research/plane-contract.md`（Plane 的迭代页列表、nerve 的接口与交接）（ignored，随阶段归档）。负责人 2026-11-11 授权连续推进至 M6 收官，重要高风险决策仍停下确认；本段不改 `server/`、`api/`，本机 `make test` 若因 M6-M8-05 的签名失败，以分支 CI 的服务端两路为准（负责人 2026-11-11）。

## 1. 目标与边界

P5a 交付了迭代的页面，详情页的正文只有转移横幅与进度侧栏。P5b 把迭代页的工作项列表接上 M4 的列表引擎，并补上进行中卡片的优先级页签和卡片、侧栏的行点击：

- **列表**：列表引擎的 `EIssuesStoreType.CYCLE`；迭代页自己的偏好（`/me/cycles/{id}/preferences`）；列表的成员判断认得 `cycle_id`；四种布局（列表、看板、日历、表格）；页头（数目、布局、筛选、显示、新建工作项）；空态；已结束与已归档迭代的列表只读；本页新建（页头、快速新建、分组头）带上本迭代；行的“移出迭代”；附件列（M5-M6-02）；在本页不出现“按迭代分组”（P3b R8）。
- **进行中卡片**：优先级页签（P5a R10）。
- **行点击**：进行中卡片的状态组、负责人、标签、优先级行设迭代页的筛选并进入迭代页；迭代页侧栏的状态组、负责人、标签行切换筛选。
- **验收**：G15 与 G19 中迭代页列表的部分（页面故事，三种引擎）；M4 交接 M6-02 与 M6-01 的迭代部分、M5-M6-02 的迭代部分。

**不做**：
- “添加已有工作项”、分组头的“添加已有”、日历的 `addIssuesToView`、`existing-issues-list-modal.tsx` 的迭代调用方，以及工作项详情、列表行、新建弹窗里的迭代下拉框，都在 P7（设计 §12 的 P7 一行）。本段的空态只有“新建工作项”。
- 列表的词表（筛选、分组、排序里的迭代、模块）与目录（P3b R9）在 P7。
- 模块页在 P6；本段为成员判断加 `module_id`（R2），给 P6 用。
- 迭代页的移动端页头在 P7（Plane 的移动端页头只有布局与显示，nerve 没有分析）。

## 2. 交付合同

### 2.1 列表即项目列表加 `cycle_id`

- 迭代页的列表是项目列表加上 `cycle_id=<本迭代>`（设计 §7）：`IssueListScope` 仍是 `scope: "project"`，查询多一项 `cycle_id: [cycleId]`；不加新的 scope 变体（R1）。绑定的键是整份 scope，所以每个迭代各有一个绑定。
- 请求：`GET /api/v0/projects/{project_id}/issues`，查询是迭代页偏好得出的筛选、分组、排序、子工作项，加 `cycle_id`、`limit`、游标；已归档的迭代照样可列（接口说明）。
- 取数：只在调用者是项目的有效成员、项目未归档、项目的迭代打开（`cycle_view`）时取（前两条同项目列表，`useIssueListMount` 的 `role !== undefined`）；访客与只看自己的访客照常读，nerve 只给他看得到的（列表与总数都按 OwnOnly）。挂载键带迭代 id。
- **迭代关闭时**：P5a 的详情页在迭代关闭时照常显示页头与进度侧栏（只读，写入口已由 `canWriteCycles` 收起），正文的列表区改为与列表页相同的停用空态（`cycles/(list)/page.tsx` 的那一个），不取本页偏好与列表，页头不显示布局、筛选、显示与新建（re-66 H1）。nerve 在迭代关闭时拒绝工作项加入与离开迭代（`planning.cycles_disabled`），列表区也就没有可做的事。

### 2.2 成员判断

- `issueMembership` 认得查询里的 `cycle_id` 与 `module_id`（各为 uuid 或 `none` 的可选值，与 `assignee_id` 等同形）：工作项的 `cycle_id` 不在所选值里就不属于这个列表；`module_id` 看 `module_ids` 是否与所选值相交，`none` 是没有模块（R2）。
- `IssueRootStore.fields()` 加入 `cycle_id` 与 `module_ids`：别处的读发现工作项换了迭代或模块时，列表标为成员可能已变。
- 写的确认（`acceptConfirmed`）与别处的读（`markMembershipStale`）都经这两处，所以别的页面把工作项移出本迭代、在本页移出、在本页建立，本页列表都按 nerve 的回答增减；不在网页上推算。

### 2.3 写入的请求体

- `IssueRootStore.createIssue` 的白名单加 `cycle_id`，`updateIssue` 的白名单加 `cycle_id`（可为 `null`）。只是放行字段，不在网页上判断能否加入：nerve 的拒绝（`planning.cycles_disabled`、`planning.archived`、`planning.cycle_completed`、422 `cycle_id`）照常经各入口的拒绝提示（R4、R6）。
- 迭代页的新建只发一个请求：`createIssue` 带 `cycle_id`（Plane 是建立后再 `POST cycle-issues`）。

### 2.4 迭代页偏好的筛选 store

- 新的 `CycleIssuesFilter` 按迭代 id 存迭代页的偏好（`getCyclePreferences`/`updateCyclePreferences`），与项目页的 `ProjectIssuesFilter` 共用一个按页的基类：取数的先后（较旧的读不覆盖较新的、本地已确认的字段比读到的新时保留）、每个字段一个 `oneAtATime` 的修改、整字段替换、回答写回、显示映射（`show`）、看板分组的折叠（本地）。项目页的行为不变，既有测试不改断言（R3）。
- `CyclePreferences` 只有 `filters`、`display_filters`、`display_properties`（没有项目的 `navigation`、`sort_order`）；`filters` 没有 `cycle_id`（接口以 400 拒绝），网页的筛选词表本来没有它（P3b R9）。
- 不读 router store：迭代页按路由参数的 `cycleId` 读本页的筛选；store 不加 `rootIssueStore.cycleId`，没有反应（P5a R4 的先例）。
- 偏好修改确认后，本页列表若查询变了就按现有分页重取（项目页 `useProjectPreferencesFollow` 的规则，改为按 store 类型与本页的 id）。
- 整字段替换不依赖已读的值：卡片的行点击（§2.8）在迭代页的偏好从未读过时也只发一个 `PATCH`（`filters` 整字段），以回答（完整的 `CyclePreferences`）种下本地的值并升该字段的版本；之后迭代页的读按“本地已确认的字段比读到的新时保留”不覆盖它（re-66 M1）。现在的 `change()` 在没有已读的值时抛错，基类要放开这一点。

### 2.5 列表引擎的 CYCLE

- `EIssuesStoreType.CYCLE`；`IssueRootStore` 增加迭代页的列表 store（`CycleIssues`，继承 `BaseIssuesStore`）与 `CycleIssuesFilter`；`useIssues`、`useIssuesActions`、`ISSUE_STORE_TO_FILTERS_MAP`、`TCreateModalStoreTypes`、拖拽的 store 类型、空态的分支都加 CYCLE；迭代页的路由映射到 CYCLE（现在映射到 DEFAULT，测试钉住，按契约变化改）。
- `CycleIssues.fetchIssues` 的查询是本页偏好得出的查询加 `cycle_id`；`createIssue`（快速新建、弹窗、分组头）在目标项目就是迭代的项目时带上本迭代的 `cycle_id`，调用方不必知道迭代；目标是别的项目时不带（否则 nerve 以 422 `cycle_id` 拒绝，而表单在 P7 之前没有迭代字段可落这个错，re-66 M3）。迭代页页头打开的新建弹窗锁定项目（`isProjectSelectionDisabled`）。迭代页的路由映射到 CYCLE 之后，全局的“新建工作项”（侧栏、命令面板，不指定 store 类型）在迭代页上也经 CYCLE 的动作；它只在迭代按页面所见可以加入时（迭代打开、未结束、未归档，与本页新建入口的条件相同）带上本迭代，否则照项目页新建，免得在已结束、已归档迭代的页面上整个新建被 nerve 拒绝而表单没有迭代字段可去掉（T5 的发现，主控裁定）。
- 读本页筛选的地方（四个 base root、日历的子树）改经一个按 store 类型给出本页筛选的 hook：PROJECT、PROFILE、ARCHIVED 照旧，CYCLE 按路由参数的 `cycleId`。日历子树现在拿整个筛选 store 读 router 驱动的 getter（`week-days`、`header`、`day-tile`、`months-dropdown`、`options-dropdown`），改为拿已解析的日历显示设置与更新回调（re-66 H2）。
- 日历的快速新建读当前 store 类型的 `viewFlags`（现在固定读项目的）。

### 2.6 页面

- **布局根**：`CycleLayoutRoot` 提供 `IssuesStoreContext` CYCLE、筛选的实例（`ProjectLevelWorkItemFiltersHOC`，`entityType` CYCLE、`entityId` 迭代 id；保存视图照项目页）、已选筛选条、四种布局的迭代根、工作项预览；取本页偏好经 `useSessionSWR`（键带迭代 id），失败可重试。放在详情页正文里转移横幅之下，进度侧栏照旧在右边。
- **页头**（`CycleIssuesHeader` 在 P5a 的面包屑、切换、侧栏开关与快捷操作之外加）：本页的工作项数（绑定给出的 nerve 的总数 `CycleIssues.pageTotal`，只在当前列表是本页的时给出，G19）、四种布局、筛选开关、显示（显示筛选与显示属性，词表同项目页，见 R9）、“新建工作项”。布局、筛选、显示对看得到页面的人都可用（照 Plane），在本页偏好读到之后出现：第一次读未答、读失败时不显示（re-67 M8）；读到过之后的重读失败时（例如重连时的重新校验），正文显示读失败并可重试，页头的控件仍在（§5 的 P6）。
- **新建工作项**（页头、快速新建、分组头的“+”）：只给能建工作项的人（项目页的 `allowPermissions([ADMIN, MEMBER])`），且迭代能被加入：迭代打开（`cycle_view`）、迭代未结束（`status` 不是 `completed`）、未归档（R5）。新建弹窗的请求体带本迭代；表单不显示迭代字段（P7 恢复下拉框）。确认后照项目页提示，本页列表经成员判断出现这一行。
- **只读**（迭代已结束或已归档，R5）：没有行内修改、快速新建、分组头的新建与拖拽；行的快捷操作只有新标签页打开、复制链接（照 Plane，没有“复制一份”，T10）；页头没有“新建工作项”；空态没有操作。机制：四个 base root 加一个只读的 prop，迭代根以 `status === "completed" || archived_at !== null` 传入，并进 `disableIssueCreation`、日历的 `readOnly` 与 `canEditProperties`（现在这三者只看路由权限，`viewFlags` 是每个 store 实例一份，re-66 M2）。
- **行的快捷操作**：在项目页的快捷操作之外加“移出迭代”（R6）：能改工作项的人、列表不只读、迭代打开时显示，经 `updateIssue` 发 `cycle_id: null`，确认后行离开本页列表并提示，被拒时说明。快捷操作里的修改与“复制一份”经列表的命令宿主（`IssueListCommandHost`）以 CYCLE 打开弹窗，行在核验中离开页面时弹窗不随之关闭（T13）；复制的一份经迭代的动作新建，同样加入本迭代，描述带图片的除外（经内容复制建立，不带 `cycle_id`，见 §5 的 P7）；工作项预览的宿主是迭代页自己的（`hostScope` 带迭代 id），不与项目页共用（re-66 L4）。
- **空态**：没有工作项时，未结束且能新建的人看到“新建工作项”；有筛选时显示“清除筛选”；已结束或已归档时只有说明。
- **写后重读进度**（R10）：迭代页列表的动作（新建、修改、移出、删除、归档、拖拽）经 nerve 确认后，经 `followInSession` 与页面范围的捕获重取本迭代的 `/progress`（`ReconciledByKey` 取最新的回答）；预览里经工作项详情的写不是列表的写，进度到下次读取更新（re-66 L5）。页头的数目来自绑定，不在网页上推算。

### 2.7 进行中卡片的优先级页签

- 统计卡的页签为“优先级工作项”“负责人”“标签”（照 Plane 的次序，默认仍是负责人，记在 `localStorage` 的 `activeCycleTab`）。
- 取数：另一个列表 store 实例，在 `IssueRootStore` 构造时建一个（同 `projectIssues`），随根退役；每个迭代一个绑定（绑定在构造时登记进根，根的列表只增不删，所以不能按组件建实例，re-66 M4）。查询是显式的完整查询 `{cycle_id: [id], priority: ["urgent", "high"], order_by: "-created_at", sub_issue: "true", limit: 30}`，不并入任何偏好，不改本页列表的当前绑定（R7）；滚到底按游标续页。
- 行：编号、名字、优先级图标、状态（只读）、目标日期；没有时显示空态；加载时显示占位；第一页或续页读失败时说明原因并可重试（`CyclesReadFailure`）。
- 点击一行：把迭代页的筛选换成“优先级为紧急或高”，nerve 确认后进入迭代页，再在迭代页的预览宿主下打开这个工作项的预览（照 Plane 的结果；nerve 的预览是工作项详情 store 的状态加宿主归属，别的宿主挂载时会清掉不属于它的选择，所以先进入、后打开，re-66 M1）。

### 2.8 行点击

- **进行中卡片**（列表页）：进度卡的状态组行（只显示数大于 0 的）、统计卡的负责人行、标签行、优先级行，点击时把该迭代的页面筛选**换成**这一个条件（`state_group`、`assignee_id`、`label_id`、`priority` 的 `in`），nerve 确认后进入迭代页；被拒时提示原因、不跳转。“无负责人”“无标签”行不可点（照 Plane）。
- **迭代页的进度侧栏**：状态页签的组行、负责人行、标签行**切换**该值（已选则去掉，条件空了就去掉条件），经本页筛选的实例保存；行显示是否已选。列表页预览里的侧栏行不可点（照 Plane）。
- 行可点时是按钮，以行命名并说明是否已选（键盘可用）。

### 2.9 守卫与告警

- `project-address-route` 收入新建与改写的文件（规则已列 Plane 旧的迭代根、空态与快捷操作的路径；`roots/cycle-layout-root.tsx`、`calendar/roots/cycle-root.tsx`、改写的 `issues/filters.tsx`、`issue-modal/base.tsx`、`calendar/calendar.tsx`、`core/sidebar/single-progress-stats.tsx` 与新 hook、store 不在，随各 Task 加入并带样例，re-66 M6）；新文件只经路由参数或 props 取工作区、项目、迭代。
- 新文案（移出迭代、优先级页签与其空态等）en 与 zh-CN 在同一提交加入（`check:sync`、`i18n-keys.mjs`）。
- 告警上限只降不升：清掉告警的提交同时下调上限。
- 按本页分组（R9）：迭代页的显示选项常量不含迭代，测试钉住。
- 关键词 phase 仍是 `M6/P5`。

### 2.10 e2e

页面故事（`e2e/stories/planning/g15-*-page.spec.ts`、`g19-*` 的迭代页列表部分），纳入三种引擎的页面配置：

- **G15**：迭代页的列表在四种布局中只列本迭代的工作项（含已归档迭代的页面），续页按 M4 的双向分页（`expectPages`、`scrollToLastListRow` 的既有写法：接口两个方向各页不重复、不丢，最后一页没有游标；页面两个方向都先列 50 行、再从分组的游标续完，之后没有更多可载），本页偏好的读写（布局、分组与显示属性改后重新载入仍在、项目页的偏好不变，表头写入的排序列表照着续页）；本页偏好的被拒钉在单测（`core/store/issue/cycle/filter.store.test.ts`、`core/hooks/cycle-preferences-follow.integration.test.tsx`），故事不另走一遍（re-67 M3）；本页新建（页头、快速新建）带上迭代、移出迭代、已结束的只读、迭代关闭时的停用空态；在迭代页归档一行后行离开、数目随 nerve 变化（归档范围）；附件列（真实附件的数目、按附件数排序、续页、换会话，增删附件后数目随读取更新，M5-M6-02）。每种布局以整页安静的断言收尾（`expectListPageQuiet`，M4 M6-01 的“整页零非预期失败”）。
- **G19**：换迭代（面包屑）、后退前进、换项目、换会话时，旧迭代的列表与迟到的回答不进新页面；失权的迟到回答（列表回答被扣住时调用者被移出项目，放行后页面不显示那一页、不报非预期失败）；只看自己的访客与工作区管理员兼访客只看到自己的；行点击与优先级页签。
- 跨分组的写之后计数（M4 M6-02）：在迭代页按状态分组时改一行的状态，两组的数随 nerve 的回答变化。

## 3. 局部裁定

- **R1 不加新的 scope 变体**：设计 §7 写“迭代页的列表就是项目列表加 `cycle_id` 筛选”。复用 `scope: "project"` 让 `read()`、`covers()`、`prepareCatalog`、权限读与续页都不用改；正确性只靠成员判断认得 `cycle_id`（§2.2）。代价：项目里的每个写都送到迭代页的绑定，由成员判断决定是否属于它（与项目页同）。
- **R2 成员判断同时加 `module_id`**：两者同形，各一处；P6 的模块页直接用。`fields()` 同时加 `module_ids`。
- **R3 偏好的筛选 store 共用一个基类**：复制项目页 260 行的先后与版本逻辑不合质量要求（设计 §8、硬规则）。基类只管偏好的三个字段与显示映射；项目页与迭代页各给 service 与键。P6 的模块页照用。项目页的既有测试是这次重构的回归保护，断言不改。
- **R4 新建一次发出**：nerve 的 `IssueCreate.cycle_id` 在同一个事务里建立成员与“加入”的动态，被拒时整个不建。Plane 的两步（建立后再加入）不照搬，登记差异。
- **R5 已结束、已归档迭代的列表只读**：照 Plane（设计 §8“照 Plane”）。nerve 的服务端允许改其中的工作项、移出（只拒绝加入），网页仍按 Plane 只读，登记差异。已归档迭代的页面照 P5a R11 可读，列表同样只读。
- **R6 移出迭代**：Plane 的迭代页快捷操作有“移出迭代”。nerve 经 `PATCH /issues/{id}` 的 `cycle_id: null`（P3a），本段放行更新的 `cycle_id`。改到别的迭代属于迭代下拉框（P7）。
- **R7 优先级页签另用一个 store 实例**：`BaseIssuesStore.select` 会把当前列表换成它选的绑定，同一实例会让迭代页的列表被卡片的查询替换。另一个实例只持有这个绑定；绑定照常登记在工作项的根里，写的确认照常送到它（成员判断加上优先级与迭代决定去留）。续页不按迭代页的显示次序重排（Plane 重排），登记差异。
- **R8 行点击照 Plane**：卡片的行**替换**迭代页的筛选（整字段 `filters`）并进入；侧栏的行**切换**一个值。nerve 的偏好是整字段替换，卡片的写不必先读。保存经 nerve 确认后才跳转。优先级行另打开预览。
- **R9 不出现“按迭代分组”**：nerve 的词表在 P7 才有迭代与模块（P3b R9），现在没有可藏的选项。本段给迭代页一份显示选项常量（同项目页、不含迭代），以测试钉住，P7 加词时迭代页仍不出现。服务端接受 `group_by: "cycle_id"`（只有一组），经接口写入的照 P3b R9 显示为不分组、修改时保留。
- **R10 写后重读进度**：M4 删除了 Plane 的分布乐观更新（M4 交接 M6-02 不许恢复）。本页列表确认一个写之后重取本迭代的 `/progress`；列表页的行计数在下次取数时更新（同转移，P5a）。
- **R11 “添加已有”在 P7**：设计 §12 把它与迭代下拉框放在 P7；nerve 没有批量接口，要逐个 `PATCH`，搜索也不排除已在迭代里的（设计 §3.3 第 12 条）。M4 交接 M6-02 写“对应列表第一次开放前完成”，字面上包含“添加已有”；本段先开放列表，“分组头添加已有、日历 `addIssuesToView`、`existing-issues-list-modal.tsx` 的领域调用方”依设计 §12 的 P7 一行与 §15 第 12 条在 P7 完成，回执照此直说（re-66 L8）。

## 4. 验收标准

- [x] A1 引擎：`EIssuesStoreType.CYCLE` 的 store、动作、路由映射、空态；绑定的查询带 `cycle_id`、挂载键带迭代 id；成员判断认得 `cycle_id`、`module_id`，`fields()` 带 `cycle_id`、`module_ids`；创建与更新放行 `cycle_id`。每个性质有单测，成员判断与迟到的应答各有能让测试失败的变异。
- [x] A2 偏好：基类重构后项目页的既有测试不改断言全过；迭代页偏好的读、修改、被拒、较旧的读不覆盖较新的；不读 router store。
- [x] A3 页面：四种布局、页头、空态、只读（已结束、已归档）、新建（页头、快速新建、分组头）只发一个带 `cycle_id` 的请求、移出迭代、拖拽；被拒时说明；写后重读进度。
- [x] A4 优先级页签与行点击：显式的完整查询、续页、点击设筛选并进入；侧栏切换；预览里不可点。
- [x] A5 G15、G19 的迭代页列表部分（三种引擎）：四种布局、双向续页、本页偏好的读写（被拒在单测，§2.10）、迭代关闭时的停用空态、换迭代与后退前进与换会话时迟到的回答不进新页面、失权的迟到回答、OwnOnly 与工作区管理员兼访客、跨分组的写后计数、归档范围、附件列（M5-M6-02）、整页零非预期失败。
- [x] A6 守卫：`project-address-route`、告警上限、knip、关键词；“按本页分组”的守卫。
- [x] A7 文档：P5a 的回填（阶段记录 §7、spec A8、设计 §14、M6-M8-05 的补充）；交接回执（M4 交接 M6-01、M6-02 的迭代部分，M5 交接 M5-M6-02 的迭代部分，M3 交接 §1 的迭代 user-properties）；差异清单、前端改动清单、设计 §8/§12/§14。
- [ ] A8 七项本地门禁（`make test` 照负责人 2026-11-11 的规则）、deep-review 修复闭环、准确的分支 CI 与 noff 后的 main CI 全绿、归档清理。（合入后由 P6 的 T1 勾选）

## 5. 后续责任

**P6**：模块页的列表照本段（`EIssuesStoreType.MODULE`、偏好的筛选 store 用同一基类、成员判断的 `module_id` 已在本段）。模块页页头的布局、筛选、显示照迭代页，在本页偏好读到之后才出现（re-67 M8）；项目页页头在偏好读到之前与读失败之后仍给这三者，点了只得一个笼统的错误提示——同一成因，M4 起如此，T16 不在本段改。迭代页读到过偏好之后的重读失败（例如重连时的重新校验）时，正文以读失败替换列表、页头的控件仍在、改了能保存而正文要按“重试”才回来（修复轮的限定复审 M1）：模块页一并决定正文是否沿用已读到的偏好而只提示重读失败，项目页与迭代页随之。

**P7**：“添加已有工作项”（空态、分组头、日历）、迭代下拉框与列表词表（R9、R11）；迭代页的移动端页头。迭代页“复制一份”时描述带图片的工作项经内容复制（草稿再发布）建立，不带 `cycle_id`、不加入本迭代（项目页同一条路；P7 随新建弹窗与草稿的迭代字段处理，T10 发现）。项目页与归档页的页头数目（`getGroupIssueCount(undefined, undefined, false)`）在列表分组时读不到 `""` 组而不显示（M4 起如此，T9 发现；迭代页改用绑定的总数 `pageTotal`）。分组头“+”打开的新建弹窗在布局之内：表单打开期间若有选父项的读或写的确认回放引起核验，它会随布局的骨架卸下（T13 的审计，未见实例、未改）；P7 给新建弹窗加迭代、模块字段时核对，须在核验中存活就照本段的修复移到列表的命令宿主（`IssueListCommandHost`）。`web/apps/web/core/store/issue/root.store.ts` 有 615 行（M4 的文件，基线 593 行，P5b 加 22 行），超过手写文件约 400 行的上限：P7 再往里加迭代、模块的写入时先拆，否则由 M8 拆（[M6-M8-08](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-08)）。归档页行的菜单仍给菜单工厂传两个空的 setter（`archived-issue.tsx`，菜单里没有修改与复制，继承自 Plane 的工厂形状；修复轮的限定复审 M3），P7 动快捷操作时以较窄的参数类型去掉。

**M8**：[M6-M8-03](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-03) 再现一次（T10 检查点本机 e2e 的 W7，只重跑这一项通过）；[M6-M8-05](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-05) 的签名也出现在本机 e2e 的全局准备（T13、T14 的三次变异运行，重跑后全局准备照常）；新增 [M6-M8-06](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-06)（e2e 作业每次在线安装浏览器的系统依赖，T10 检查点的分支 CI 43047769115 因 runner 解析不了 Ubuntu 的镜像两次取消；加固改作业结构，须负责人同意）；新增 [M6-M8-07](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-07)（T16 本机门禁中 M5 的 F11 (page) [s3] 失败一次：编辑器在相邻的事务里重建已有图片的节点、中止了未答的内容读取，只重跑该文件即通过）、[M6-M8-08](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-08)（`root.store.ts` 的行数，re-67 M10）与 [M6-M8-09](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-09)（Firefox 在同一浏览器再次载入后约 10 s 报根预加载的字体未用；G15 已避开，其余三引擎故事同属潜在间歇）。

## 6. 风险

| 风险 | 处理 |
|---|---|
| 偏好基类的重构碰坏项目页 | 项目页的既有测试（`filter.store.test.ts` 等）断言不改，作回归保护；重构单独一个 Task，先跑项目页再接迭代页 |
| 项目里的每个写都送到迭代页的绑定 | 与项目页同；成员判断的单测覆盖迭代、模块的各种值（含 `none`），变异钉住 |
| 优先级页签的绑定与迭代页的列表互相替换 | 另一个 store 实例（R7），测试核对迭代页的当前绑定不变 |
| 卡片点击后迭代页读到旧的偏好 | 先保存、nerve 确认后再跳转；迭代页按会话的 SWR 取偏好 |
| e2e 期限 | 新故事计入三种引擎的页面配置在本机跑；CI 只跑默认配置，T 检查点核 e2e 作业时长（期限 35 分钟） |
