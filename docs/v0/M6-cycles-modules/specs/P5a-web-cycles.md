# M6 / P5a：网页的迭代（数据层与页面）

> 状态：待预检。基线 main `a2b50624fd872c183628a55d04b7961f9a1dbc60`（P4 的 noff `c3ba5a40` 与其 main CI 失败的根因修复 noff `a2b50624`，main CI 42824266166 首次通过）。设计见 [M6 设计](../M6-design.md) §2 的 G1–G3、G5–G8、G19，§3.1–§3.8、§5、§8；负责人 2026-11-07 的决定 A/B/C 见设计 §15（第 2、3、4、5、6、11、13、14 条与本段直接相关）。网页的长期规则见[总体设计 §7.2、§7.7](../../v0-design.md#77-stores-按会话分代长期有效m2-起)。接口合同见 [P1 spec](P1-cycles-backend.md)、[P3b spec](P3b-list-engine.md) R8/R9、[P4 spec](P4-progress-transfer.md) §2.3、R3、R9。现状调研见本阶段台账 `research/web.md`（网页现状与约定）、`research/plane-contract.md`（Plane 的页面行为、接口契约与交接）（ignored，随阶段归档）。负责人 2026-11-11 授权连续推进至 M6 收官，重要高风险决策仍停下确认。

## 1. 目标与边界

设计 §12 的 P5（前端迭代）超出一段约 16 个 Task 的上限（调研：迭代领域 72 个文件、6,138 行全部仍调 Plane 的旧地址；迭代页的工作项列表还要给列表引擎加一种 store），按设计 §12“超限在实施前拆分”拆为两段，关键词 phase 两段都是 `M6/P5`（先例 P3a/P3b 都用 `M6/P3`）：

- **P5a（本 spec）**：
  - **数据层**：迭代的 service 与 store 按会话分代、只调 `/api/v0`、类型来自 OpenAPI；项目包装层按权限取迭代；筛选 store 不再跟随 router；每域一个进度函数；燃尽图按 `Burndown` 画。
  - **页面**：迭代列表（进行中卡片、即将开始、已结束）与预览侧栏、归档目录、详情页的页头与进度侧栏、新建/修改弹窗、快捷操作（修改、新标签页打开、复制链接、归档、恢复、删除）、删除与归档确认框、转移弹窗与入口。
  - **收尾**：跨领域读者改用新 store，旧 service、store、手写类型与旧地址删除；本段改到的文件里交接列出的键盘、地址、复制、死成员与告警逐项完成。
  - 覆盖 G1–G3、G5–G8 的页面与 G19 中迭代页面的部分。
- **P5b（P5a 合入后再写 spec 与 plan）**：迭代页的工作项列表：列表引擎的 `EIssuesStoreType.CYCLE`、迭代页偏好（`/me/cycles/{id}/preferences`）的筛选 store、列表的成员判断认得本页的 `cycle_id`、四种布局、迭代页的“新建工作项”、已结束迭代的列表只读、在本页隐藏“按迭代分组”（P3b R8）、附件列（M5-M6-02）；进行中卡片与进度侧栏的行点击（设筛选并进入迭代页）。覆盖 G15 与 G19 中迭代页列表的部分。

**P5a 不做：**
- 迭代页的工作项列表（P5b）。详情页在 P5a 与 P5b 之间照现状只有页头与侧栏（M4 删了旧的列表根）。
- 工作项一侧（P7）：工作项的迭代属性与下拉框、列表筛选/分组/排序/显示的迭代与模块词表、“添加已有”、日历、命令面板里迭代的本地过滤与切换。G4 的页面在 P7。
- 收藏（M7）：本段只删除迭代的收藏入口（设计 §15 第 13 条）。
- 模块（P6）。模块与迭代共用的 `ProgressChart`、`progress-stats/*` 中模块仍在用的部分保留给 P6。

## 2. 交付合同

### 2.1 service

`core/services/planning/cycles.service.ts`（新）：构造函数接收这一代的客户端（`ApiClient`），每个接口操作一个方法，经 `unwrap` 调用，返回生成的类型，不做数据转换（先例 `core/services/project/labels.service.ts`）。

| 方法 | 操作 |
|---|---|
| `list(projectId)`、`listArchived(projectId)` | `listCycles`（`archived=true`） |
| `get(cycleId)` | `getCycle` |
| `create(projectId, body: CycleCreate)` | `createCycle` |
| `update(cycleId, body: CycleUpdate)` | `updateCycle` |
| `remove(cycleId)` | `deleteCycle` |
| `archive(cycleId)`、`unarchive(cycleId)` | `archiveCycle`、`unarchiveCycle` |
| `progress(cycleId)` | `getCycleProgress` |
| `transfer(cycleId, targetCycleId)` | `transferCycleIssues` |

迭代页偏好（`getCyclePreferences`/`updateCyclePreferences`）在 P5b。没有 `date-check`、收藏、工作区级迭代、`analytics`。

### 2.2 store

**位置与接线**：`core/store/planning/`（新）的 `PlanningStore` 持有 `cycles: CycleStore` 与 `cycleFilters: CycleFilterStore`（P6 在同处加模块）。`RootStore` 的 `planning` 字段以 `(this, api)` 建它；hook `useCycles()`、`useCycleFilters()`（`core/hooks/store/`）。旧的 `RootStore.cycle`、`cycleFilter`、`core/store/cycle.store.ts`、`cycle_filter.store.ts`、`use-cycle.ts`、`use-cycle-filter.ts` 在最后一个读者改用新 store 的 Task（T12）删除（R3）。

**`CycleStore` 的值**（`core/lib/reconciled.ts`，v0 §7.7“取数和修改的应答对齐”）：

| 值 | 键 | 来源 |
|---|---|---|
| `unarchived: ReconciledByKey<CycleListItem[]>` | 项目 id | `list` |
| `archived: ReconciledByKey<CycleListItem[]>` | 项目 id | `listArchived` |
| `details: ReconciledByKey<Cycle>` | 迭代 id | `get` |
| `progress: ReconciledByKey<CycleProgress>` | 迭代 id | `progress` |

- **读**：迭代自身的字段取各副本中 nerve 最后回答的那一份（`answeredAt`，v0 §7.7“一个实体有几份副本”；先例 `project.store.ts` 的 `getProjectById`）；计数、`assignee_ids`、冻结的计数只有列表行有，从它所在的列表行读。按项目的选择器收项目 id（来自路由参数），不读 router store：本项目未归档的迭代 id（服务端次序）、进行中的迭代 id（第一个 `status === "current"` 的）、已归档的迭代 id。项目不在调用者此刻的项目列表里（删除、离开）时回答 `undefined`（先例 `label.store.ts:126-130`）。
- **取数**：`fetchCycles(projectId)`、`fetchArchivedCycles(projectId)`、`fetchCycle(cycleId)`、`fetchProgress(cycleId)`，都只经 `useSessionSWR`（键 `["PROJECT_CYCLES", loginId, projectId]` 等，由 `sessionKey` 生成），失败是 SWR 的 `error`，不是未处理的拒绝；`SessionChangedError` 给出 `undefined`。
- **写**：经 `oneAtATime()` 一个接一个发出，请求体在轮到它时算（v0 §7.7）。nerve 确认后写进持有它的每一份：
  - 建：`Cycle` 补成计数全为 0、`assignee_ids: []`、`progress_snapshot: null` 的列表行，放在本项目未归档列表的最前（新迭代的 `sort_order` 最小，服务端次序同此；新迭代不会有工作项，所以这些值是准确的，不是猜测）。
  - 改：迭代的字段（含服务端回答的 `status`）写进每一份，列表行保留自己的计数。
  - 归档、恢复：在两个列表之间移动该行（保留计数），归档目录按 `archived_at DESC` 放在最前；重复归档/恢复（200、无写入）照常写入回答。
  - 删：从每一份删去，进度一并删去。
  - 转移：回答转走的个数；之后重取源与目标的进度和本项目的未归档列表（计数变了），不在网页上推算新的计数。
- **拒绝**：每个写的拒绝都交给调用者（`ApiError`），store 不吞错误、不打日志了事（Plane 的归档、恢复吞掉错误，失败也提示成功，不照搬）。

**项目包装层的取数**（M3 交接 §1）：`core/layouts/auth-layout/use-project-fetch.ts` 在调用者是项目的有效成员（`member_role` 不为 `null`）、项目未归档且 `cycle_view` 打开时取本项目未归档的迭代（`PROJECT_CYCLES`）；有 vitest 钉住“非成员、已归档、开关关闭时不取”。归档目录与详情页各自取自己要的（`PROJECT_ARCHIVED_CYCLES`、`CYCLE`、`CYCLE_PROGRESS`），同样只在成员且项目未归档时取。

### 2.3 筛选 store 与工具函数

**`CycleFilterStore`**：按项目 id 保存 `{default, archived}` 两组筛选（状态、开始日、结束日）与两个搜索词（列表、归档目录），读写都收项目 id。**不再跟随 router**：没有 `reaction`，于是没有要在退役时释放的反应（M2 交接 §3、M3 交接 §1 的条件以更强的形式满足，R4）。Plane 的 `displayFilters{active_tab, layout}` 没有读者，删除。

**`@nerve/utils` 的 `cycle.ts`**（输入是生成的类型）：
- `orderCycles(rows)`：Plane 的次序（进行中、即将开始、草稿；即将开始按开始日，其余按名字的小写），只用服务端的 `status`。
- `shouldFilterCycle(row, filters)`：状态按服务端的 `status`；日期筛选照 Plane 的选项（之后若干周、自定义），比较的是日历日（`"YYYY-MM-DD"`）与浏览器的今天，不经时区换算。
- **进度的唯一口径**（M1 收尾交接“进度的口径”、设计 §3.5）：
  - `cycleCounts(status, live, frozen)`：`status === "completed"` 且 `frozen` 不为 `null` 时给冻结的数，否则给实时的数。列表行、`/progress` 的回答都经它选数，冻结的条件只在这里写一次。
  - `progressPercent(counts)`：已完成 ÷（总数 − 已取消），分母为 0 时为 0，取整同 Plane。列表行的圆环、进行中卡片、侧栏的“工作项”、侧栏状态页签都调它；模块在 P6 调同一个函数。
- **日历日**：迭代的日期是项目时区的日历日（设计 §3.1）。显示、日期选择器、比较都按日历日处理，不经 `new Date("YYYY-MM-DD")`（按 UTC 解析，西半球会显示前一天）。删除只给迭代用的 `useTimeZoneConverter` 与“按你的时区”提示（R6）。

### 2.4 燃尽图与分布

- **`BurndownChart`**（`core/components/core/sidebar/burndown-chart.tsx`，新）：输入 `Burndown`。横轴按 `points` 的次序；实际线取 `remaining`，`null`（今天之后）不画点、不画成 0；理想线从首点的 `scope` 线性降到末点的 0，只有一个点时理想值为 0；所有坐标是有限数；0 个点不会出现（`burndown` 为 `null` 时显示“暂无数据”）。`ProgressChart` 与它的模块类型留给 P6 的模块侧栏（P6 改用 `BurndownChart` 后删除）。
- **分布的行**：`AssigneeDistribution`、`LabelDistribution` 只带 id（P4 R3）。网页按项目成员显示名字与头像、按标签显示名字与颜色；成员已离开项目或标签已删除（冻结里会有）时显示兜底名字（“已离开的成员”“已删除的标签”）；`null` 行是“没有负责人”“没有标签”。显示次序：按显示的名字（`localeCompare`），兜底名字的行在后，`null` 行最后（Plane 按名字排）。

### 2.5 页面

所有页面读路由参数（`useParams`）得到工作区、项目、迭代，不读 router store 的地址；按权限隐藏的区域，它的取数同样按权限启用（v0 §7.7）。“能写”指调用者在项目中的角色是管理员或成员（访客与只看自己的访客都不能写，OwnOnly 只出现在访客身上）。

| 页面/组件 | 合同 |
|---|---|
| **列表页** `(list)/page.tsx`、`header.tsx`、`mobile-header.tsx` | `cycle_view` 关闭：停用的空态，按钮只对管理员可用（同 Plane）。取数中：加载占位。取数失败：可重试的错误态（不是永远加载）。没有未归档的迭代：空态，“新建迭代”对不能写的人停用。否则：已选筛选条、`CyclesView`。页头：面包屑；能写的人有搜索、筛选与“新建迭代”（经项目级弹窗宿主）。移动端的版式菜单没有读者，删除。 |
| **列表** `cycles-view.tsx`、`list/*` | 依次为进行中卡片、“即将开始”（默认展开，含草稿，`orderCycles` 的次序减去进行中的那个）、“已结束”（默认收起，服务端次序）。搜索与筛选都为空时的空态区分“清除筛选”“清除搜索”。行：标题链接到详情页；已归档的行打开预览；前缀是 `progressPercent(cycleCounts(...))` 的圆环，100% 时打勾；草稿与即将开始显示工作项数；日期按日历日显示；建立者头像（`created_by_id`）；负责人头像组（`assignee_ids`）；转移入口（§2.5 末行）；快捷操作。没有收藏星。 |
| **预览** `cycle-peek-overview.tsx` | `?peekCycle=<id>` 打开右侧面板，内容是进度侧栏（不可编辑统计）；已归档的迭代同样能看（nerve 允许读已归档的迭代）。 |
| **进行中卡片** `active-cycle/*` | 没有进行中的迭代：空态。否则：该行、进度卡（状态组的行、已取消的说明、`progressPercent`）、燃尽卡（`BurndownChart`，待处理数 = 积压+未开始+进行中）、统计卡三个页签：优先级（本迭代里紧急与高优先级的工作项，经 `listIssues` 的 `cycle_id`、`priority` 筛选，按游标分页、滚到底取下一页）、负责人、标签（§2.4）。页签记在 localStorage（`activeCycleTab`，同 Plane）。行点击的跟进在 P5b。 |
| **归档目录** `archives/cycles/*`、`archived-cycles/*` | 取已归档的迭代（失败可重试，不永远加载）；搜索、筛选（无状态筛选）；空态；行打开预览；恢复经快捷操作。标签页在 `cycle_view` 关闭时隐藏（现状）。 |
| **新建/修改弹窗** `modal.tsx`、`form.tsx` | 见 §2.6。 |
| **快捷操作** `quick-actions.tsx`、`common/quick-actions-helper.tsx` 的迭代菜单 | 修改：能写、未结束、未归档。新标签页打开：总是（`window.open` 带 `"noopener,noreferrer"`）。复制链接：总是，经 `useCopyLink`，失败有提示。归档：能写、未归档；未结束时停用并说明“只有已结束的迭代能归档”。恢复：能写、已归档。删除：项目管理员，或建立者（且是管理员或成员）；未结束、未归档（同 Plane 的菜单条件，权限按 nerve 的规则收窄）。菜单按钮以迭代命名（`common.item_options`），不嵌套在另一个按钮里；右键菜单关着时不可交互。 |
| **删除确认框** `delete-modal.tsx` | 只在 nerve 确认后关闭并提示；被拒绝时留着、显示拒绝的说明（`useRefusalToast`/`errorMessageKey`）。在详情页或预览里删除后回到列表。 |
| **归档确认框** `archived-cycles/modal.tsx` | 标题经 `ModalTitle`，自然 Tab 次序（删去 `tabIndex={1}`）；只在 nerve 确认后关闭、提示并回到列表；被拒绝时留着并说明。恢复同样等 nerve 确认后提示并进入归档目录之外的列表页；被拒绝时提示拒绝，不提示成功。 |
| **详情页** `(detail)/layout.tsx`、`[cycleId]/page.tsx`、`header.tsx` | 子树以 `cycleId` 为 key（M3 交接 §1）。404：“迭代不存在”与回到列表的按钮（文案经 `t()`）；其他失败可重试。页头：面包屑（项目、“迭代”、切换迭代的 `BreadcrumbNavigationSearchDropdown`，列出本项目未归档的迭代，键盘可用，`onOpen` 每次打开调一次）、快捷操作、侧栏开关（`cycle_sidebar_collapsed`）。已归档的迭代照常显示（只读，写入口按快捷操作的条件隐藏）。正文在 P5a 是转移横幅与进度侧栏（工作项列表在 P5b）。 |
| **进度侧栏** `progress-sidebar/*` | 页头：关闭按钮有名字；名字与状态徽标（服务端 `status`）；日期范围（能写、未结束、未归档时可改，否则停用）。改日期：发出 PATCH，**等 nerve 确认**后才提示成功并显示新日期；被拒绝（重叠 409、已结束 409 等）时提示拒绝，日期回到 nerve 的值。详情：说明（纯文本）、负责人（`owned_by_id`）、成员（列表行的 `assignee_ids`）、工作项“已完成/（总数 − 已取消）”（经 `cycleCounts` 与 `progressPercent` 的同一份计数）。进度：草稿显示“暂无数据”；有燃尽时 `BurndownChart`；状态、负责人、标签三个页签（页签记在 localStorage `cycle-progress-tab-{id}`），行不可点击（P5b）。数字来自 `/progress`：`status === "completed"` 且冻结不为 `null` 时读冻结（`cycleCounts` 与同一条件选分布与燃尽），否则读实时。 |
| **转移** `transfer-issues-modal.tsx`、`transfer-issues.tsx`（重建） | 入口：列表行的“转移 N 个工作项”与详情页的横幅（“已结束的迭代不能再加入工作项”与“转移工作项”按钮）。显示条件：能写、`status === "completed"`、未归档、实时的未完成数（积压+未开始+进行中）> 0（不看是否已冻结；可以多次转移）。弹窗：`ModalTitle` 命名，搜索框，目标是本项目未归档、状态为草稿/即将开始/进行中、不是源本身的迭代，每个目标是有名字的按钮（名字与状态）；没有目标时说明“先建一个迭代”。选中后**等 nerve 回答**：成功时关闭并提示转走的个数（0 也是成功），随后的重取见 §2.2；被拒绝时留着并说明拒绝（`target_cycle_id` 的 422 与 409 都经 `errorMessageKey`），不提示成功。 |

### 2.6 新建/修改弹窗与项目级弹窗宿主

- **宿主**：`core/components/modals/project-level.tsx` 的 `ProjectLevelModals` 以 `projectId` 为 key，卸载时（`useLayoutEffect` 的清理）关上它渲染的三个开关（新建迭代、模块、视图），先例 `WorkspaceLevelModals`（M3 交接 §1；设计 §12 把“弹窗宿主与 key”放在 P7，P5a 是第一个让它宿主的弹窗能用的阶段，所以在此完成，R2）。
- **弹窗**：`ModalCore` 带 `handleClose`（Escape、点外面都经它；提交中不能关）；标题经 `ModalTitle`（“新建迭代”“修改迭代”）；关闭后焦点回到打开它的控件。
- **表单**：项目选择（新建时，限于调用者能写的项目；M2 交接 §5(d) 在浏览器核对）、名字（必填、≤255、自动聚焦）、说明（纯文本）、日期范围（`DateRangeDropdown`，在弹窗里 `modal={false}`，选择器的浮层在弹窗之内可用；`minDate` 为浏览器的今天，同 Plane）。删除 `getTabIndex(ETabIndices.PROJECT_CYCLE)` 的正 `tabIndex`，用自然次序（`getTabIndex` 本身留给 M7，M3 交接 §5）。
- **提交**：
  - 新建：`CycleCreate`；草稿**不带**日期键（`CycleCreate` 的日期不可为 `null`，请求体封闭）；不带 `project_id` 等多余的键。修改：只发改了的字段（`CycleUpdate`，日期为一对）。
  - 没有 `date-check`：重叠由 nerve 在写入时拒绝（409 `planning.cycle_dates_overlap`）。
  - **只在 nerve 确认后关闭**：被拒绝时弹窗留着、输入保留；字段错误（422 `name`、`start_date`、`end_date`）经 `useFormRefusal` 显示在字段下（`FieldError`、`aria-invalid`、`useRefusedFocus`，`shouldFocusError: false`），其余拒绝（409、403、503）经 `useRefusalToast` 提示。
  - 关闭、提示等后续经 `followInSession`，只在标签页仍在发出时的会话里跟进；宿主卸载（换项目、后退）后迟到的回答只更新 store，不在新页面提示或跳转。

### 2.7 跨领域读者与删除

改用新 store（行为不变的只改读法）：
- `core/store/favorite.store.ts`、`core/hooks/use-favorite-item-details.tsx`：删除迭代分支对迭代 store 的读写（收藏属 M7；设计 §15 第 13 条），收藏的其他类型不动。
- 命令面板：`power-k/ui/pages/open-entity/project-cycles-menu.tsx`、`power-k/menus/cycles.tsx`、`power-k/hooks/use-context-indicator.ts`、`power-k/config/navigation/commands.ts`、`power-k/ui/pages/context-based/cycle/commands.ts`（删除收藏命令；复制链接经 `useCopyLink`；权限带项目）。行为上的其余改动（本地过滤与切换）在 P7。
- `core/hooks/use-workspace-issue-properties.ts`：删除取工作区级迭代（没有这个接口，设计 §8）。
- `common/quick-actions-helper.tsx` 的迭代菜单、`issues/issue-layouts/removal.test.tsx` 中读迭代 store 的部分。

删除：`core/services/cycle.service.ts`（及其测试）、`cycle_archive.service.ts`、旧的两个 store 与两个 hook、`@nerve/types` 的 `cycle/`（`ICycle`、`TCycleDistribution`、`TProgressSnapshot`、`CycleDateCheckData` 等；只给网页筛选用的形状移到筛选 store 旁）、`@nerve/utils` 中基于 `ICycle` 的旧函数与 `progress.test.ts` 的旧用例、`validateCycleSnapshot`、`useTimeZoneConverter`、`ETabIndices.PROJECT_CYCLE`、`WORKSPACE_CYCLES` 取数键、不再有读者的文案键。

### 2.8 守卫与告警

- `tools/keywords.json`：
  - `phase` 改为 `M6/P5`（T1）。
  - 新规则 `old-cycle-urls`：`web/**` 不得出现 Plane 的迭代地址（`/cycles/` 结尾的斜杠形式、`archived-cycles`、`date-check`、`cycle-issues`、`transfer-issues/`），先例 `old-issue-urls`。
  - `project-address-route` 的文件表加入本段改写的迭代文件。
  - `copy-hook` 中迭代的两条例外（`cycles/quick-actions.tsx`、`power-k/.../cycle/commands.ts`）随复制改经 hook 的同一提交删除（例外不再匹配即判为过期而失败）。
- oxlint：本段改到的文件告警为零；迭代领域与共用文件现有的 5 条（`single-progress-stats.tsx` 2 条、`archived-cycles/modal.tsx` 的正 `tabIndex`、`dropdowns/filters/root.tsx` 的 `autoFocus`、`applied-filters/root.tsx` 的 `prefer-set-has`）随之清掉，`web/apps/web/package.json` 的上限随之下调（`lint-cap` 要求恰好相等）。
- 死成员：M1 收尾交接的 `--rows M6` 中迭代的行、M3 交接 §2 的 `core/sidebar/progress-stats/` 8 行，各删去或在阶段记录写明（模块仍在用的留给 P6）。
- `.oxlintrc.json` 的 `no-restricted-imports`（只能从 `swr` 导入类型）列入新的 store 与取数文件（v0 §7.7）。

### 2.9 e2e

新页面故事放在 `e2e/stories/planning/`，标题 `G<n> (page): …`，与同号的接口故事共用 `fixtures/assert/planning.ts`、`progress.ts` 的数据库断言；世界经接口建立（`progressWorld`、`endCycle`、`cycleCalls`；新项目的 `cycle_view` 默认关闭，故事先打开）。浏览器证据用 `watchPage`/`expectQuietConsole`、`dialogs.ts`、`settings-pages.ts` 的 `sentTo`/`sentHeld`/`moveWithinApp`、`fixtures/auth.ts` 的 `anotherTabSignsIn`。

| 文件 | 覆盖 |
|---|---|
| `g1-create-cycle-page.spec.ts` | 建草稿（请求体没有日期键）与带日期的迭代；重叠被拒时弹窗留着、输入保留并说明；名字的字段错误与焦点；访客没有“新建”；开关关闭的停用空态；弹窗的名字、Tab 次序、Escape、焦点回到按钮 |
| `g2-cycle-status-page.spec.ts` | 列表的三段与进行中卡片按服务端 `status`（草稿在“即将开始”里）；与东八区项目的日界一致（不按浏览器推算） |
| `g3-change-cycle-page.spec.ts` | 弹窗修改名字与日期；侧栏改日期等 nerve 确认、被拒时回到原值；已结束的迭代没有“修改”、日期停用；删除：管理员与建立者可见、其他成员不可见，被拒时确认框留着 |
| `g8-archive-cycle-page.spec.ts` | 归档已结束的迭代、归档目录、预览、恢复；未结束时归档停用；被拒（如开关在别处关闭）时说明拒绝、不提示成功 |
| `g5-cycle-progress-page.spec.ts` | 进行中卡片与侧栏的计数、百分比、燃尽（一天的迭代、未来为空）、负责人与标签页签的名字、“没有负责人”、离开的成员的兜底名字；只看自己的访客只看到自己的数 |
| `g6-cycle-freeze-page.spec.ts` | 已结束的迭代冻结后改状态、移出：侧栏与列表行显示冻结的数，实时的不同 |
| `g7-cycle-transfer-page.spec.ts` | 从列表行与横幅转移；目标的范围；回答个数；被拒时留着；可以再转；访客没有入口 |
| `g19-cycle-pages.spec.ts` | 另一个标签页换账号后迟到的回答不写进新页面；弹窗开着时后退/换项目，弹窗随之结束；换迭代（面包屑切换）后详情为新迭代、迟到的旧进度不写入；复制链接失败有提示；菜单与筛选按钮的名字与 `aria-expanded` |

### 2.10 文件与职责

| 路径 | 职责 |
|---|---|
| `web/apps/web/core/services/planning/` | `cycles.service.ts`（§2.1） |
| `web/apps/web/core/store/planning/` | `PlanningStore`、`CycleStore`、`CycleFilterStore` 与测试（§2.2、§2.3） |
| `web/apps/web/core/store/root.store.ts`、`core/lib/store-context.tsx` | 接线；`dispose()` 的说明随 R4 更新；`root.store.test.ts` 的 store 数随之改 |
| `web/apps/web/core/layouts/auth-layout/use-project-fetch.ts` | 按权限取迭代（§2.2） |
| `web/packages/utils/src/cycle.ts` | §2.3 的函数与测试 |
| `web/apps/web/core/components/cycles/**`、`core/components/core/sidebar/burndown-chart.tsx`、`progress-stats/*` | §2.4–§2.6 |
| `web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/cycles/**`、`archives/cycles/**` | 路由页面（§2.5） |
| `web/apps/web/core/components/modals/project-level.tsx` | 宿主的 key 与清理（§2.6） |
| §2.7 所列 | 跨领域读者与删除 |
| `web/packages/i18n/src/locales/{en,zh-CN}/` | 新文案（转移、兜底名字、错误态等）与删除的键 |
| `tools/keywords.json`、`.oxlintrc.json`、`web/apps/web/package.json` | §2.8 |
| `e2e/` | §2.9 |
| `docs/` | P4 回填（T1）；`plane-diff.md`、`frontend-changes.md`；输入交接的“处理结果（M6/P5a）”；设计 §8、§12、§14 |

## 3. 局部裁定

- **R1 拆段**：§1。P5b 的边界写在本 spec，它的 spec 与 plan 在 P5a 合入后写（设计 §12“后续阶段只在前置事实落实后写可执行 plan”）。
- **R2 设计 §12 列给 P7 的几项在本段完成**：P5a 重写的文件里就有它们，留到 P7 会把同一批文件改两次：
  - 迭代领域文件的地址改读路由参数并加入 `project-address-route`；
  - 迭代的两处复制经 hook（含命令面板里迭代的复制命令）；
  - 迭代的收藏入口删除（列表行的星、快捷操作、命令面板的收藏命令、收藏 store 的迭代分支）；
  - 项目级弹窗宿主的 key 与清理（新建迭代是它第一个能用的弹窗）。
  - P7 仍做工作项一侧与命令面板的其余行为。设计 §12 的表随之改写（T15）。
- **R3 新 store 并行到 T12**：`RootStore.planning` 先与旧的 `cycle`/`cycleFilter` 并存，逐个页面改用新 store，最后一个读者改完的 Task 删除旧的。每个 Task 的提交都能通过类型检查与测试；并存只在本分支上，合入的是只有新 store 的状态。这不是兼容层：旧 store 调的都是 nerve 不提供的地址，本来就不工作。代价：中间提交里未改的页面仍是坏的（与 main 现状相同）。
- **R4 筛选 store 不跟随 router**：Plane 的 `CycleFilterStore` 以 `reaction` 跟随 `router.projectId` 来初始化筛选并清空搜索词。改为按项目 id 存、按项目 id 读之后不需要这个反应，退役时也就没有要释放的（交接的条件“`dispose()` 释放迭代筛选的反应”以“没有反应”满足，`store-context.test.ts` 加一条断言：退役的一代不再响应 router）。代价：搜索词按项目保留到会话结束（Plane 换项目时清空），记入前端改动清单。
- **R5 列表的三段照 Plane**：进行中卡片、“即将开始”（含草稿）、“已结束”。设计 §8 写的“进行中卡片、未开始、已结束、草稿”是状态的列举，不是另立“草稿”一段（Plane 没有），T15 改写设计 §8。次序：即将开始照 Plane 的 `orderCycles`（只用服务端的 `status`）；已结束按服务端次序；归档目录按服务端次序（`archived_at DESC`）。界面上没有写 `sort_order` 的入口（Plane 同）。
- **R6 日期是日历日**：设计 §3.1 把迭代的日期改成项目时区的日历日，Plane 的“按你的时区”提示与时区换算对日历日没有意义，删除。日期选择器的 `minDate` 照 Plane 为浏览器的今天（选择器的提示，不是状态的推算）；状态只读服务端的 `status`（G2）。
- **R7 冻结只在已结束时读**：`cycleCounts` 一处决定（§2.3）。列表行的冻结是六个数（`FrozenCounts`），`/progress` 的冻结是整份（`FrozenProgress`），两者都经同一条件。Plane 的 `validateCycleSnapshot` 不看状态，删除。
- **R8 权限按 nerve 的规则**：写入口（新建、修改、归档、恢复、转移、侧栏改日期）只给能写的人；删除只给管理员或建立者（Plane 给任何成员，nerve 会 403）；转移入口 Plane 没有权限门，nerve 加上（访客会 403）。菜单的状态条件照 Plane（已结束不能修改、不能删除）。
- **R9 写入的回答不写进列表行的计数**：`Cycle` 不带计数（P4 R9）。建立时补 0（准确）；修改、归档、恢复只改迭代的字段、保留行的计数；转移之后重取。
- **R10 优先级页签经列表接口**：nerve 没有 Plane 的 `cycle-issues` 接口；进行中卡片的优先级页签用 `listIssues`（`cycle_id`、`priority=urgent,high`、按游标分页），不新增接口。
- **R11 已归档迭代的详情可读**：nerve 允许读已归档的迭代与进度（Plane 回答 404）。归档目录的行照 Plane 打开预览；直接打开已归档迭代的地址时详情照常显示、只读。登记差异。
- **R12 不吞拒绝**：Plane 的归档、恢复吞掉错误，删除与新建/修改弹窗失败也关闭，转移弹窗不等回答就关闭，侧栏改日期不等回答就提示成功。全部改为等 nerve 确认、被拒时留着并说明（M3 交接 §3、M4 交接 M6-04 的“表单失败即关闭”“日期提交未等待”）。登记差异。

## 4. 验收标准

- [ ] A1 数据层：service 只调 `/api/v0`；store 按会话分代（构造时拿这一代的客户端、取数经 `useSessionSWR` 且键带 `loginId`、修改经 `oneAtATime`、副本取最后回答的一份、拒绝交给调用者）；包装层按权限与开关取数；筛选 store 没有跟随 router 的反应；每个性质有单测，权限门与迟到应答各有能让测试失败的变异。
- [ ] A2 进度口径：`cycleCounts`、`progressPercent` 是唯一口径，列表行、进行中卡片、侧栏的数字一致；冻结只在已结束时读；燃尽图 0/1/多个点、`null` 不画成 0、坐标有限；分布按名字、兜底名字、`null` 行最后。
- [ ] A3 G1–G3、G8 的页面：建、改、删、归档、恢复按 §2.5、§2.6，被拒时留着并说明；权限与状态条件按 R8；草稿的请求体没有日期键；没有 `date-check`。
- [ ] A4 G5–G7 的页面：进行中卡片、侧栏、冻结、转移按 §2.5；转移看实时的未完成数、可以多次转移。
- [ ] A5 G19（迭代页面）：换账号、换工作区/项目/迭代、后退前进时迟到的回答不写进新页面；详情子树以 `cycleId` 为 key；项目级宿主以 `projectId` 为 key 并在卸载时关上开关；弹窗、菜单、下拉框的键盘、名字与焦点逐项（M3 交接 §3 的迭代各项）；复制链接失败有提示。
- [ ] A6 删除与守卫：§2.7 的旧代码全部删除；`old-cycle-urls` 规则；`project-address-route` 收入改写的文件；`copy-hook` 迭代的例外删除；改到的文件 oxlint 告警为零、上限下调；迭代的死成员与 progress-stats 8 行有结论。
- [ ] A7 文档：差异清单、前端改动清单、输入交接的“处理结果（M6/P5a）”（M1-P2、M1-P3 在网页改用新接口后关闭）、设计 §8/§12/§14；P4 的阶段记录 §7、spec A8 与设计 §14 已回填。
- [ ] A8 七项本地门禁、deep-review 修复闭环、准确的分支 CI 与 noff 后的 main CI 全绿、归档清理。（合入后由 P5b 的 T1 勾选）

## 5. 后续责任

**P5b**（迭代页的工作项列表）：§1 所列；进行中卡片与侧栏的行点击（Plane 的 `handleFiltersUpdate`：设迭代页的筛选并进入迭代页）；G15、G19 的列表部分；M4 交接 M6-02 的迭代部分、M5-M6-02。

**P6**（模块页面）：模块改用 `BurndownChart` 与 `progressPercent` 后删除 `ProgressChart`、`TModuleCompletionChartDistribution` 与 `progress-stats/*` 中只有模块用的部分；模块筛选 store 的 `reaction`（照 R4 去掉）。

**P7**：工作项一侧（属性与下拉框、新建弹窗的迭代、列表词表、添加已有、日历）用本段的 `useCycles()`；列表引擎的目录（P3b R9 的空 `Map`）改由本段的 store 提供；命令面板的本地过滤与切换。

**M7**：迭代的收藏随收藏一起恢复（本段删除了入口）。

## 6. 风险

- **大范围的类型替换**：`ICycle` 有 37 个读者。R3 的并行让每个 Task 只改它负责的页面；T12 以类型检查与 knip 确认没有残留。
- **日历日的显示**：任何一处经 `new Date("YYYY-MM-DD")` 都会在西半球显示前一天；工具函数的单测用负偏移时区（`TZ`）跑一次，e2e 的世界在东八区（`noonZone()`）。
- **列表行与详情的副本**：写入的回答（`Cycle`）不带计数，合并时不能把行的计数写成 0 或 `undefined`（R9）；store 的单测钉住。
- **迟到的回答**：页面的跟进只在发出时的会话与页面里做；e2e 的 `sentHeld` 与 `anotherTabSignsIn` 覆盖。
- **e2e 时长**：新增 8 个页面故事；CI 的 e2e 作业期限 35 分钟（负责人 2026-11-07 同意的值），检查点的分支 CI 核对实际时长，超出要先报告负责人，不自行改期限。
