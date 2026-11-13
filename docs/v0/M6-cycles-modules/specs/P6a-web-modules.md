# M6 / P6a：网页的模块（数据层与页面）

> 状态：预检通过（修正后，[re-68](../../../reviews/re-68-M6P6a预检.md)）。基线 main `aadaf61a5929b57d8aa53b93e5ae04b03cc046c6`（P5b 的 noff `9675bb31` 与其 main CI 失败的根因修正 noff `aadaf61a`，main CI 43373039561 首次通过）。设计见 [M6 设计](../M6-design.md) §2 的 G9–G13、G19，§3.2、§3.5–§3.8、§5、§8、§11、§12；负责人 2026-11-07 的决定 A/B/C 见设计 §15（第 5、9、10、11、13、15、17 条与本段直接相关）。网页的长期规则见[总体设计 §7.2、§7.7](../../v0-design.md#77-stores-按会话分代长期有效m2-起)。接口合同见 [P2 spec](P2-modules-backend.md)、[P4 spec](P4-progress-transfer.md) §2.3；迭代一侧的先例与交给本段的事见 [P5a spec](P5a-web-cycles.md) §5、[P5b spec](P5b-cycle-list.md) §5 的 P6 一行。现状调研见本阶段台账 `research/web.md`（网页的模块现状、P5a/P5b 的约定、生成的客户端、守卫与交接）、`research/plane-contract.md`（Plane 的模块页面、nerve 的模块契约、两者的差异）（ignored，随阶段归档）。负责人 2026-11-11 授权连续推进至 M6 收官，重要高风险决策仍停下确认；本段不改 `server/`、`api/`，本机 `make test` 若因 M6-M8-05 的签名失败，以分支 CI 的服务端两路为准（负责人 2026-11-11）。

## 1. 目标与边界

设计 §12 的 P6（前端模块）同 P5 一样超出一段约 16 个 Task 的上限（调研：模块领域约 73 个文件、6,770 行仍是 Plane 的形状、调 Plane 的旧地址，与 P5a 开工时的 72 个文件、6,138 行相当；模块页的工作项列表还要给列表引擎加一种 store），按设计 §12“超限在实施前拆分”由主控拆为两段，关键词 phase 两段都是 `M6/P6`（先例 P5a/P5b 都用 `M6/P5`）：

- **P6a（本 spec）**：
  - **数据层**：模块的 service 与 store 按会话分代、只调 `/api/v0`、类型来自 OpenAPI；项目包装层按权限与开关取模块；迭代与模块共用一个取数门；模块的写与删的门照 nerve 的规则；筛选 store 不跟随 router；模块的进度只经 `progressPercent`，燃尽图与分布用 P5a 的件。
  - **页面**：模块列表（列表与卡片两种版式、排序、筛选、搜索）与预览、归档目录、详情页的页头与侧栏（状态、负责人、成员、日期、链接、进度）、新建/修改弹窗、快捷操作（修改、新标签页打开、复制链接、归档、恢复、删除）、删除与归档确认框。
  - **收尾**：跨领域读者（命令面板、收藏、通知页、快捷操作的帮助函数）改用新 store，旧 service、store、手写类型、Plane 的地址与只剩模块读者的旧进度件删除；本段改到的文件里交接列出的键盘、地址、复制、死成员与告警逐项完成。
  - 覆盖 G9、G10、G12、G13 的页面与 G19 中模块页面的部分。
- **P6b（P6a 合入后再写 spec 与 plan）**：模块页的工作项列表：列表引擎的 `EIssuesStoreType.MODULE`、模块页偏好（`/me/modules/{id}/preferences`）的筛选 store（P5b 的按页基类）、工作项写入的 `module_ids` 整集合（本页新建、行的“移出模块”）、四种布局与页头、空态、在本页不出现“按模块分组”（P3b R8）、附件列（M5-M6-02）、侧栏的行切换本页筛选；G15 与 G19 中模块页列表的部分（R15）；在往工作项根 store 加这些之前先把它按职责拆开（M6-M8-08，R16）。

**P6a 不做：**
- 模块页的工作项列表（P6b）。详情页在 P6a 与 P6b 之间照 P5a 的先例只有页头与侧栏，侧栏的分布行不可点。
- 工作项一侧（P7）：工作项的模块属性与下拉框、列表筛选/分组/排序/显示的模块词表、“添加已有”、`existing-issues-list-modal.tsx` 的模块调用方、命令面板里模块菜单的本地过滤与切换。G11 的页面在 P7。
- 收藏（M7）：本段只删除模块的收藏入口（设计 §15 第 13 条）。

## 2. 交付合同

### 2.1 service

`core/services/planning/modules.service.ts`（新）：构造函数接收这一代的客户端（`ApiClient`），每个接口操作一个方法，经 `unwrap` 调用，返回生成的类型，不做数据转换（先例 `cycles.service.ts`）。

| 方法 | 操作 |
|---|---|
| `list(projectId)`、`listArchived(projectId)` | `listModules`（`archived=true`） |
| `get(moduleId)` | `getModule`（已归档的也给，开关关闭也给） |
| `create(projectId, body: ModuleCreate)` | `createModule` |
| `update(moduleId, body: ModuleUpdate)` | `updateModule` |
| `remove(moduleId)` | `deleteModule` |
| `archive(moduleId)`、`unarchive(moduleId)` | `archiveModule`、`unarchiveModule` |
| `progress(moduleId)` | `getModuleProgress` |
| `links(moduleId)`、`addLink(moduleId, body: ModuleLinkCreate)` | `listModuleLinks`、`createModuleLink` |
| `changeLink(linkId, body: ModuleLinkPatch)`、`removeLink(linkId)` | `updateModuleLink`、`deleteModuleLink` |

模块页偏好（`getModulePreferences`/`updateModulePreferences`）在 P6b。没有工作区级的模块列表（设计 §5：Plane 的 `GET /api/workspaces/{slug}/modules/` 只给通知页与工作区视图，属 M7）、收藏、Plane 的 `PUT`。

### 2.2 store

**位置与接线**：`PlanningStore`（`core/store/planning/index.ts`）加 `modules: ModuleStore` 与 `moduleFilters: ModuleFilterStore`；hook `useModules()`、`useModuleFilters()`（`core/hooks/store/`）。旧的 `RootStore.module`、`moduleFilter`、`core/store/module.store.ts`、`module_filter.store.ts`、`use-module.ts`、`use-module-filter.ts` 在最后一个读者改用新 store 的 Task（T12）删除（R2）；`root.store.test.ts` 的 store 数随之由 23 变 21。

**`ModuleStore` 的值**（`core/lib/reconciled.ts`，v0 §7.7“取数和修改的应答对齐”）：

| 值 | 键 | 来源 |
|---|---|---|
| `unarchived: ReconciledByKey<ModuleListItem[]>` | 项目 id | `list` |
| `archived: ReconciledByKey<ModuleListItem[]>` | 项目 id | `listArchived` |
| `details: ReconciledByKey<Module \| null>` | 模块 id | `get`（删除后写成 `null`，先例 `CycleStore`） |
| `progress: ReconciledByKey<Progress>` | 模块 id | `progress` |
| `links: ReconciledByKey<ModuleLink[]>` | 模块 id | `links` |

- **读**：模块自身的字段取各副本中 nerve 最后回答的那一份（`answeredAt`，先例 `CycleStore.getCycleById`）；六个计数只有列表行有，从它所在的列表行读（`getModuleRow`，行叠上最新的字段）。按项目的选择器收项目 id（来自路由参数），不读 router store：本项目未归档的模块 id（服务端次序 `sort_order, created_at, id`）、已归档的模块 id（`archived_at DESC, id`）。项目不在调用者此刻的项目列表里（删除、离开）时回答 `undefined`。进度与链接只给 `getModuleById` 能给的模块（删除之后读不到，同 P5a T3 的裁定）。
- **取数**：`fetchModules(projectId)`、`fetchArchivedModules(projectId)`、`fetchModule(moduleId)`、`fetchProgress(moduleId)`、`fetchLinks(moduleId)`，都只经 `useSessionSWR`（键 `["PROJECT_MODULES", loginId, projectId]`、`PROJECT_ARCHIVED_MODULES`、`MODULE`、`MODULE_PROGRESS`、`MODULE_LINKS`，由 `sessionKey` 生成），失败是 SWR 的 `error`，不是未处理的拒绝；`SessionChangedError` 给出 `undefined`。
- **写**（`core/store/planning/module-changes.ts`）：经 `oneAtATime()` 一个接一个发出，请求体在轮到它时算（v0 §7.7）。nerve 确认后写进持有它的每一份：
  - 建：`Module` 补成六个计数全为 0 的列表行，放在本项目未归档列表的最前（nerve 给新模块最小的 `sort_order`，`planning.yaml` 的 `createModule`；新模块没有工作项，这些值是准确的）。
  - 改：模块的字段写进每一份（两个列表的行、自己的读），行保留自己的计数（R9）。成员的增减以“加入谁、去掉谁”交给 store，轮到它发出时以 nerve 最近一次回答的 `member_ids` 算出整集合（`member_ids` 整体替换，R6）；其余字段照调用者给的发。
  - 归档、恢复：在两个列表之间移动该行（保留计数），放在最前；重复归档/恢复（200、无写入）照常写入回答；没有行时不造行（列表的下次取数带来）。
  - 删：从两个列表删去，读到的模块写成 `null`；删除在轮到时找模块。
  - 链接：加的回答（元数据为空）放在该模块链接列表的最前（nerve 的次序是最新的在前）；改与删在轮到时按链接的 `module_id` 找它所在的列表，写入回答或删去。元数据的读取见 R10。
- **与迭代的不同**：`ModuleListItem` 没有 `assignee_ids` 与冻结；模块不冻结，没有 `cycleCounts`；没有转移；多了链接与成员的整体替换。
- **拒绝**：每个写的拒绝都交给调用者（`ApiError`），store 不吞错误、不打日志了事（Plane 的归档、恢复吞掉错误，失败也提示成功，R11）。
- **文件大小**：值、读与选择器一个文件（`module.store.ts`），写另一个（`module-changes.ts`），各约 400 行以内（先例 P5a 的切法）。

**项目包装层的取数**（M3 交接 §1）：`core/layouts/auth-layout/use-project-fetch.ts` 在调用者是项目的有效成员、项目未归档且 `module_view` 打开时取本项目未归档的模块（`PROJECT_MODULES`，同 `PROJECT_CYCLES`）；vitest 钉住“非成员、已归档、开关关闭时不取”。e2e 夹具 `fixtures/mounts.ts` 记下这一取数（`PROJECT_MODULES`，先例 `PROJECT_CYCLES`）。

### 2.3 取数门与写的门

- **取数门**（re-65 M6，R3）：一个 hook（`core/hooks/use-planning-reads.ts`，名字由实现定）对路由的工作区与项目回答两件事：**可读**：调用者在项目里有角色（`getProjectRoleByWorkspaceSlugAndProjectId` 不为 `undefined`，访客也算）且项目未归档；**列表可取**：可读，且该域的开关（`cycle_view`、`module_view`）打开。开关这一条是一个导出的谓词（如 `planningOn(project, "cycles" | "modules")`），hook 与包装层都用它；包装层自己的“有效成员”判断（`access.kind === "member"`，还要等 nerve 对项目的读）不变，它之下的页面用 hook。取数门在 T5 抽出（T2 先把模块的条件写在包装层迭代的条件旁）。每一处取数的取舍（迭代的各处行为不变，既有测试断言不改）：

| 取数处 | 现在的条件 | 用什么 |
|---|---|---|
| 包装层 `use-project-fetch.ts` | 有效成员 + `cycle_view`（T2 加 `module_view` 一条） | 开关的谓词，成员判断照旧 |
| 迭代列表页 `R/cycles/(list)/page.tsx`、命令面板的迭代菜单 `project-cycles-menu.tsx` | 角色 + 未归档 + `cycle_view` | 列表可取 |
| 迭代详情页 `R/cycles/(detail)/[cycleId]/page.tsx`、迭代归档目录 `archived-cycles/root.tsx` | 角色 + 未归档 | 可读 |
| 进行中卡片的优先级页签 `cycles/active-cycle/priority-issues.tsx` | 角色 + 未归档 + `cycle_view` | 列表可取 |
| 迭代页的列表根 `issues/issue-layouts/roots/cycle-layout-root.tsx` | 角色 + `cycle_view`（P5b 的 re-66 H1），不看已归档 | 可读（否则不渲染）与列表可取（否则停用空态）：它只在包装层之下渲染，已归档的项目到不了它，多出的“未归档”一条不改行为 |
| 迭代详情页头 `R/cycles/(detail)/header.tsx` | 角色 + `cycle_view`（是否显示列表的数目与偏好的控件） | 列表可取（同列表根，理由同上） |
| 模块的列表页、命令面板的模块菜单 | — | 列表可取 |
| 模块的详情页、预览、归档目录 | — | 可读 |

  开关的其余读者不是取数门，各自留着条件：导航与项目侧栏的标签（`use-navigation-items.ts`、`project-navigation.tsx`）、归档目录的标签（`archive-tabs-list.tsx`）、命令面板命令的可见性（`config/navigation/commands.ts`、`config/creation/command.ts`：`create_module` 改用下面的写的门，其余在 P7）、`useCycleListState` 的“可加入”（写入的条件，模块的在 P6b）与下面的写的门。它们回答“显示哪个入口、能不能写”，不是“取不取”，并进取数门会把两种问题混在一个函数里。
- **写的门**（R5）：`UserPermissionStore` 加 `canWriteModules(workspaceSlug, projectId)` 与 `canDeleteModule(workspaceSlug, projectId, createdByCaller)`，照 `access/domain/decide.go` 与 `planning/app/lock.go`：能写是项目的管理员或成员（工作区管理员算管理员），且 nerve 对这个项目的模块不是一律拒绝（只看自己的访客、项目的模块关闭 `module_view`、项目不在）；能删是管理员，或建立者且按项目角色是管理员或成员，同样在“一律拒绝”时没有。“一律拒绝”的条件与迭代共用一处（现在的 `cycleWritesRefused` 按域取开关），`canWriteCycles`、`canDeleteCycle` 的行为不变。所有写入口（新建、修改、归档、恢复、删除、侧栏与行内的修改、链接的加改删、命令面板的模块命令）只经这两个门。

### 2.4 筛选 store 与工具函数

**`ModuleFilterStore`**（`core/store/planning/module_filter.store.ts`）：按项目 id 保存 `{default, archived}` 两组筛选（状态、负责人、成员、开始日、目标日）与两个搜索词（列表、归档目录），读写都收项目 id。**不跟随 router**：没有 `reaction`；不写 `localStorage`：筛选里有成员的 id，不能在同一浏览器里带到另一个账户（R4）。Plane 的 `displayFilters.favorites` 删除（收藏属 M7）。

**列表的显示设置**：版式（`list`、`board`）与排序（`order_by`）是纯展示的设置，在组件里经 `useLocalStorage` 按项目、按浏览器存（同 Plane，同 `module_sidebar_collapsed`），默认 `list` 与 `name`（同 Plane）。一个 hook（`core/components/modules/` 下，名字由实现定）读写它们，键带项目 id（不沿用 Plane 的整张表 `module_display_filters`），存的值不在选项里时回到默认；页头的版式与排序、移动端的版式菜单、列表、归档目录都经它，归档目录用同一个排序（同 Plane）（R4）。

**负责人与成员的筛选**列工作区的成员（同 Plane），读 `getWorkspaceMemberIds(workspaceSlug)`（`workspace-member.store.ts`，按地址的 slug），slug 来自路由参数；不读 `workspaceMemberIds`（它读 router store 的 `workspaceSlug`，同文件的 getter；`module-view-header.tsx`、`archived-modules/header.tsx` 现在以嵌套的解构读它，`project-address-route` 的内容模式认不出这种写法，§2.10）（R14）。

**只给界面用的类型**：`ModuleLayout`、`ModuleOrderBy` 在 `@nerve/constants` 的 `src/module.ts` 由选项数组（`MODULE_VIEW_LAYOUTS`、`MODULE_ORDER_BY_OPTIONS`）推出（先例 `constants/src/cycle.ts` 的内联联合）；状态用 `@nerve/api-client` 的 `ModuleStatus`；筛选的形状 `ModuleFilters` 放在 `@nerve/utils` 的 `planning.ts`、`matchesModuleFilters` 旁（先例 `CycleFilters`）。`@nerve/utils` 已依赖 `@nerve/constants`（`utils/package.json`，`src/tab-indices.ts` 已从它导入），`orderModuleRows` 从它取 `ModuleOrderBy`。T4 建立它们，T12 删除 `@nerve/types` 的 `module/`（§2.9）。

**`@nerve/utils` 的 `planning.ts`** 加下面两个函数（旧的 `module.ts` 有 `orderModules`、`shouldFilterModule`，由旧 store 读到 T12，所以新函数另起名字，`module.ts` 在 T12 删除，先例 re-64 H2）：
- `orderModuleRows(rows, orderBy)`：名字（Plane 的自然排序，`Intl.Collator` 带 `numeric`）、进度（`progressPercent`）、工作项数（`total_issues`）、目标日、建立时刻，各有升降两向；“手动”是 `sort_order`、没有方向。相等的保持传入的次序（服务端次序）；没有目标日的在两个方向都排在最后（R14）。
- `matchesModuleFilters(row, filters)`：状态任一；负责人（`lead_id` 在所选里）；成员（与所选相交）；开始日、目标日照 Plane 的选项按日历日比较，与 `matchesCycleFilters` 共用同一段日子的判断，不经 `new Date("YYYY-MM-DD")`。搜索词按名字的小写包含（同 Plane）。
- **进度的唯一口径**：`progressPercent`（P5a 已有，文档已写“迭代（或模块）”）是模块唯一的百分比，列表行的圆环、卡片的进度条与数字、按进度排序、侧栏的“工作项”与状态页签都调它（M1 收尾交接“进度的口径”第 2 条）。模块不冻结，计数直接来自列表行或 `/progress`。

### 2.5 迭代与模块共用的件

- **读失败与停用**：`cycles/read-failure.tsx`（可重试的读失败）与 `cycles/disabled.tsx`（开关关闭的停用空态，按钮只给项目管理员）移到 `core/components/planning/`，停用空态按域取文案、图与设置页的地址；迭代的读者改用它们，行为不变。
- **分布的行**：`cycles/distribution-rows.ts`、`distribution-stats.tsx` 移到 `core/components/core/sidebar/`（与 `BurndownChart`、`single-progress-stats.tsx` 同处），迭代与模块共用（P5a §5 的 P6）。
- **进度的页签**：迭代侧栏的状态、负责人、标签三个页签（`cycles/progress-sidebar/progress-stats.tsx`）抽成共用的组件（同处），收 `Progress`、项目 id、页签记在 `localStorage` 的键与可选的行切换；迭代侧栏以它加上 CYCLE 的行切换（行为不变，既有测试断言不改），模块侧栏用它、不带行切换（P6b 加 MODULE 的）。Plane 的模块侧栏页签按状态、负责人、标签而面板按负责人、标签、状态（M4 交接 M6-04），共用之后页签与面板一致。
- **燃尽图**：模块侧栏用 `BurndownChart`（开始日到目标日）。`ProgressChart`、`TModuleCompletionChartDistribution`、`core/sidebar/progress-stats/{assignee,label,state_group}.tsx` 在模块侧栏改用上面的件之后没有读者，随之删除（M3 交接 §2 的 8 行随之了结）；`@nerve/propel` 的 `AreaChart` 的 `comparisonLine` 没有调用方（`BurndownChart` 与 `ProgressChart` 都不传），连同 `@nerve/types` 里它的类型删除，它的单点除零随之不存在（M4 交接 M6-04、P5a §5）。

### 2.6 页面

所有页面读路由参数（`useParams`）得到工作区、项目、模块，不读 router store 的地址；按权限隐藏的区域，它的取数同样按权限启用（v0 §7.7）。“能写”指 `canWriteModules`，“能删”指 `canDeleteModule`（§2.3）。

| 页面/组件 | 合同 |
|---|---|
| **列表页** `(list)/page.tsx`、`header.tsx`、`mobile-header.tsx`、`layout.tsx` | `module_view` 关闭：停用的空态（§2.5，同 Plane）。取数中：加载占位。取数失败：可重试的错误态（不是永远加载）。没有未归档的模块：空态，“新建模块”对不能写的人停用。否则：已选筛选条与列表。页头：面包屑；搜索、排序、筛选、版式对看得到页面的人都有（同 Plane）；能写的人有“新建模块”（经项目级弹窗宿主）。移动端页头：版式菜单（读写本项目的显示设置，§2.4）。布局挂 `ModuleActionsHost`。 |
| **列表与卡片** `modules-list-view.tsx`、`module-list-item.tsx`、`module-list-item-action.tsx`、`module-card-item.tsx` | 次序与筛选见 §2.4；搜索与筛选都为空时的空态区分“清除筛选”“清除搜索”。行：名字链接到详情页；“信息”按钮切换 `?peekModule=`；前缀是 `progressPercent` 的圆环，已完成而不到 100% 时显示“!”（同 Plane）；日期范围、状态（行内可改）；负责人的头像或“没有负责人”；快捷操作。卡片：名字链接到详情页，“信息”按钮，状态（行内可改），“已完成/（总数 − 已取消）个工作项”与同一份 `progressPercent` 的进度条，负责人，日期（行内可改），快捷操作。行内的修改只给能写的人、未归档的模块，等 nerve 确认后显示，在途时停用，被拒时提示、值回到 nerve 的（R11、R13）。没有收藏星。行与卡片的点击区域是链接或按钮，不是带点击的 `div`（`module-card-item.tsx` 的 5 条告警随之清掉）。 |
| **预览** `module-peek-overview.tsx` | `?peekModule=<id>` 打开右侧面板，内容是侧栏，以所预览的模块为 key（re-65 I1 的先例）；取 `MODULE`、`MODULE_PROGRESS`、`MODULE_LINKS`（“可读”时），失败可重试；已归档的模块同样能看（R12）；关闭时去掉查询参数。 |
| **归档目录** `archives/modules/*`、`archived-modules/{root,header,view}.tsx` | 取已归档的模块（“可读”时；失败可重试，不永远加载）；搜索、排序、筛选（同 Plane，无“收藏”）；空态；行打开预览；恢复经快捷操作。标签页在 `module_view` 关闭时隐藏（现状）。布局挂 `ModuleActionsHost`。 |
| **新建/修改弹窗** `modal.tsx`、`form.tsx` | 见 §2.7。 |
| **快捷操作** `quick-actions.tsx`、`common/quick-actions-helper.tsx` 的模块菜单 | 修改：能写、未归档。新标签页打开：总是（`window.open` 带 `"noopener,noreferrer"`）。复制链接：总是，经 `useCopyLink`，失败有提示。归档：能写、未归档；状态不是已完成、已取消时停用并说明“只有已完成或已取消的模块能归档”（经 `t()`，现在是英文字面量）。恢复：能写、已归档；在途时再触发不发（re-65 M4 的先例）。删除：能删、未归档（同 Plane 的菜单条件，权限按 nerve 的规则收窄，R5）。菜单按钮以模块命名（`common.item_options`），不嵌套在另一个按钮里；右键菜单关着时不可交互。弹窗经 `ModuleActionsHost`（§2.7）。 |
| **删除确认框** `delete-module-modal.tsx` | 只在 nerve 确认后关闭并提示；被拒绝时留着、说明拒绝（`useRefusalToast`/`errorMessageKey`）。在详情页或预览里删除所显示的模块后回到列表（Plane 以 `useParams()` 读搜索参数 `peekModule`，永远读不到，R11）。 |
| **归档确认框** `archived-modules/modal.tsx` | 标题经 `ModalTitle`，自然 Tab 次序（删去 `tabIndex={1}`）；只在 nerve 确认后关闭、提示并回到列表；被拒绝时留着并说明。恢复同样等 nerve 确认后提示并进入归档目录（同 Plane，关掉预览）；被拒绝时提示拒绝，不提示成功。 |
| **详情页** `(detail)/layout.tsx`、`[moduleId]/page.tsx`、`header.tsx` | 子树以 `moduleId` 为 key（M3 交接 §1），`ModuleActionsHost` 在 key 之内（先例迭代）。取 `MODULE`、`MODULE_PROGRESS`、`MODULE_LINKS`（“可读”时，开关关闭也读，R3）。404（`planning.module_not_found`）：“模块不存在”与回到列表的按钮（文案经 `t()`，替换写死的英文）；其他失败可重试。页头：面包屑（项目、“模块”、切换模块的 `BreadcrumbNavigationSearchDropdown`，列出本项目未归档的模块，服务端次序，键盘可用；它内部的 `CustomSearchSelect` 每次打开调一次 `onOpen`，M2 交接 §5(c) 在浏览器核对）、侧栏开关（`module_sidebar_collapsed`，同 Plane）、快捷操作。已归档的模块照常显示（只读，写入口按上面的条件隐藏；不在切换列表里，R12）。正文在 P6a 是侧栏；页头的数目、版式、筛选、显示与“新建工作项”在 P6b。 |
| **侧栏** `progress-sidebar/*` | 页头：关闭按钮有名字（预览里）；状态（`CustomSelect`，M2 交接 §5(b)）。名字与说明（纯文本）。详情：日期范围、负责人、成员、“工作项”：已完成/（总数 − 已取消）（`/progress` 的计数，与状态页签同一份）。可改的条件：能写且模块未归档（Plane 同条件，权限按 nerve 收窄）。每个修改只发它改的字段的 `PATCH`，**等 nerve 确认**后才提示成功并显示新值，在途时该控件停用；被拒绝时提示拒绝，值回到 nerve 的（Plane 改日期不等回答就提示成功、状态与成员的失败没人接，R11）。负责人与成员只列项目里不是访客的成员，成员的增减照 R6；日期照 R13。进度：有燃尽时 `BurndownChart`，没有（缺开始或目标日、跨度超过 3660 天）时“暂无数据”，分布照常（R8）；三个页签（§2.5），页签记在 `localStorage` 的 `module-progress-tab-{id}`，行不可点（P6b）。链接见 §2.8。 |

### 2.7 新建/修改弹窗与页面级宿主

- **项目级宿主**：`ProjectLevelModals`（P5a 已以 `projectId` 为 key、卸载时关上新建迭代、模块、视图三个开关）照旧渲染新建模块的弹窗，改读新 store。
- **页面级宿主**：`ModuleActionsHost`（`core/components/modules/actions-host.tsx`，先例 `CycleActionsHost`）持有模块的修改、归档、删除三个弹窗与“页面是否还在”（`pageMounted`），挂在模块列表、归档目录与详情页三个布局上；快捷操作经 `useModulePage()` 打开弹窗。T6 建立宿主时只接修改弹窗与 `pageMounted`；归档与删除确认框在 T7 改写时接入（旧的两个收 `IModule` 与旧 store 的参数，`delete-module-modal.tsx`、`archived-modules/modal.tsx`，改写之前接不进收 `Module` 的宿主）。行在自己的修改确认后可能消失（归档、删除、改名后的筛选），以页面为准，确认后的提示与跳转照常、换页面后迟到的回答只进 store（P5a T7 的裁定）。链接的弹窗属于侧栏的链接一节（侧栏以模块为 key，链接的确认不卸下侧栏）。
- **弹窗**：`ModalCore` 带 `handleClose`（Escape、点外面都经它；提交中不能关）；标题经 `ModalTitle`（“新建模块”“修改模块”）；关闭后焦点回到打开它的控件。
- **表单**：项目选择（新建时，只列能写模块的项目，`canWriteModules`，R7；M2 交接 §5(d) 在浏览器核对）；名字（必填、≤255、自动聚焦）；说明（纯文本）；日期范围（`DateRangeDropdown`，已固定 `modal={false}`；两天都可空、可只有一天，R13；M2 交接 §5(d) 在浏览器核对浮层在弹窗之内可用）；状态（`ModuleStatusSelect`，默认“积压”，同 Plane 的网页；没过规则时有经 `t()` 的提示、`aria-invalid`、被拒时取得焦点，不再收 `tabIndex`，M3 交接 §3）；负责人与成员（只列所选项目里不是访客的成员；换项目时清空已选的，R6）。删除 `getTabIndex(ETabIndices.PROJECT_MODULE)` 的正 `tabIndex` 与 `PROJECT_MODULE` 的表，用自然次序（`getTabIndex` 本身留给 M7，M3 交接 §5）。
- **提交**：
  - 新建发 `ModuleCreate`：名字、状态，以及有值的说明、日期、负责人、成员；没有的日期不发键（`ModuleCreate` 的日期不可为 `null`，请求体封闭），不发 `project_id` 等多余的键。修改发 `ModuleUpdate`，只发改了的字段（日期为一对，清空的为 `null`；`member_ids` 为表单上的整集合）。
  - **只在 nerve 确认后关闭**：被拒绝时弹窗留着、输入保留（Plane 失败后把表单重置为默认值，`form.tsx:69-75`，R11）；字段错误（422 `name`、`start_date`、`target_date`、`lead_id`、`member_ids[i]`）经 `useFormRefusal` 显示在字段下（`FieldError`、`aria-invalid`、`useRefusedFocus`，`shouldFocusError: false`）；其余拒绝（409 `planning.module_name_taken`、`planning.archived`、`planning.modules_disabled`、`project.archived`，403，503）经 `useRefusalToast`/`errorMessageKey`（M3 交接 §1：`modal.tsx` 读 `error?.detail ?? error?.error` 的两处）。
  - 关闭、提示等后续经 `followInSession`，只在标签页仍在发出时的会话里、页面仍在时跟进。

### 2.8 链接

- **列出**：`MODULE_LINKS`，nerve 的次序（最新的在前）；对看得到模块的人都显示（Plane 只给能编辑的人看，访客看到“还没有链接”，R10）。
- **一行**：图标：元数据的 `favicon` 是内嵌的 `data:image/` 时显示它，否则按地址的图标（Plane 的 `getIconForLink`）；不让浏览器去取 `favicon_url`（M4 工作项链接的规则，`issue-detail/links/link-item.tsx`）。文字：链接的标题（非空时），否则元数据的标题，否则地址；`<a target="_blank" rel="noopener noreferrer">`。“<多久之前> 由 <名字> 添加”（`created_by_id` 按工作区成员显示，账户已不在时只写时刻）。复制地址对所有人，经 `useCopyText`，成功与失败各有经 `t()` 的提示（M2 交接 §6）。
- **加、改、删**：能写且模块未归档。加与改经链接弹窗：标题经 `ModalTitle`（“添加链接”“修改链接”）；地址必填，空时显示提示（`FieldError`，Plane 的“URL is required”从不显示，M3 交接 §3）；标题可空；地址照用户输入发出，由 nerve 规整（去空白、没有协议时补 `http://`，`shared.CheckLinkURL`；Plane 的网页自己补 `http://`，结果相同）；只在 nerve 确认后关闭；422 `url`、`title` 在字段下，409 `planning.module_link_exists`、`planning.archived` 等经提示。删除不确认（同 Plane），nerve 确认后行消失并提示，被拒时提示、行留着。
- **元数据**：新加或改了地址的链接，nerve 回答时元数据为空，抓取在后台；元数据的读取照 R10。

### 2.9 跨领域读者与删除

改用新 store（行为不变的只改读法）：
- 命令面板：`power-k/menus/modules.tsx`、`ui/pages/context-based/module/{commands,root,status-menu}.tsx`、`ui/pages/open-entity/project-modules-menu.tsx`、`hooks/use-context-indicator.ts`、`config/navigation/commands.ts`（类型改为 `Module`）、`config/creation/command.ts`（`create_module` 改用 `canWriteModules`，同 P5a 的 `create_cycle`）。上下文命令：删去收藏的两条；改状态与成员经新 store，成员的增减照 R6；被拒时经 `useRefusalToast`；复制链接经 `useCopyLink`。本地过滤与切换在 P7。
- `core/store/favorite.store.ts`、`core/hooks/use-favorite-item-details.tsx`：删除模块分支对模块 store 的读写（收藏属 M7；设计 §15 第 13 条），收藏的其他类型不动。
- `core/hooks/use-workspace-issue-properties.ts`：它只剩取工作区级模块的一半（P5a 删了迭代的一半），nerve 没有这个接口：整个 hook 与 `workspace-notifications/root.tsx` 对它的调用删除（通知页每次打开不再请求 Plane 的 `GET /api/workspaces/{slug}/modules/`）。
- `common/quick-actions-helper.tsx` 的模块菜单（T7）、`modals/project-level.tsx`（T6）、`issues/issue-layouts/removal.test.tsx` 与 `R/domain-pages.test.tsx` 中读模块 store 的部分。

删除：`core/services/module.service.ts`、`module_archive.service.ts`、旧的两个 store 与两个 hook、`@nerve/types` 的 `module/`（`IModule`、`TModuleStatus`、`TModuleDistribution`、`TModuleCompletionChartDistribution`、`TModuleLayoutOptions`、`TModuleOrderByOptions`、`TModuleFilters`、`TModuleDisplayFilters` 等）与只剩模块读者的 `ILinkDetails`（只给界面用的版式、排序与筛选的形状在 T4 已立在 `@nerve/constants` 与 `@nerve/utils`，状态用生成的 `ModuleStatus`，§2.4；`@nerve/constants` 的 `MODULE_STATUS` 随之改用 `ModuleStatus`）、`@nerve/utils` 的 `module.ts`、`WORKSPACE_MODULES` 取数键、`ETabIndices.PROJECT_MODULE`、`ProgressChart` 与 `progress-stats/{assignee,label,state_group}.tsx`、模块的收藏入口、没有读者的 `router.store.ts` 的 `moduleId`、不再有读者的文案键。`@nerve/propel` 自己的 `TModuleStatus`（`icons/module/module-status-icon.tsx`）是包内的界面类型，保留。

### 2.10 守卫与告警

- `tools/keywords.json`：
  - `phase` 改为 `M6/P6`（T1）。
  - 新规则 `old-module-urls`：`web/**` 的源文件（测试也算）不得出现 Plane 的模块接口地址，照 `old-cycle-urls` 以 `/api/workspaces/` 前缀锚定、以负向前瞻定边界：`/api/workspaces/(?:[^/"'`\s]+/)+(?:modules|archived-modules|module-links|user-favorite-modules)(?![\w-])…`；命中的样例含工作区级列表、项目列表、详情、`archive/`、`module-links/`、`archived-modules/`、无尾段的；不命中的含 `/api/v0/projects/{project_id}/modules`、`/api/v0/module-links/{module_link_id}`、`/api/v0/me/modules/{module_id}/preferences`、组件目录的导入（`./archived-modules/modal`）、网页内的链接（`/${workspaceSlug}/projects/${projectId}/modules/${moduleId}`、`…/archives/modules`、`?peekModule=`）、`…/modules-overview/` 与迭代的地址。`old-cycle-urls` 的说明里“模块的地址到 P6 另加规则”一句随之改为指向它。
  - `project-address-route` 的文件表随各 Task 加入本段改写的模块文件（路由页面、组件、hook），带样例。它的内容模式认得 `{ workspaceMemberIds } =` 这样的一层解构，认不出 `const { workspace: { workspaceMemberIds } } = useMember()` 这样的嵌套解构（模块的两个页头今天就这样读 router 驱动的 getter，I1）：第一个把模块页头加进文件表的 Task（T8 的详情页头）同时给内容模式加一个嵌套一层的分支（四个 getter 同 `projectStates|projectLabels|joinedProjectIds|workspaceMemberIds`），`samples.hit` 加嵌套解构的两例，`samples.miss` 加 `workspace: { getWorkspaceMemberIds }`。写 spec 时对 `aadaf61a` 试跑过这个分支：规则文件表现有的 121 个文件里命中 0 个，所以加上它不会让现有文件变红；表外命中的 9 个文件（模块的两个页头、成员下拉框、项目与工作区成员页等）不在规则的范围内，模块的两个页头随 T10、T11 改读 `getWorkspaceMemberIds` 后加入。
  - `copy-hook` 中模块的三条例外（`modules/quick-actions.tsx`、`modules/links/list-item.tsx`、`power-k/ui/pages/context-based/module/commands.tsx`）随复制改经 hook 的同一提交删除。
- oxlint：本段改到的文件告警为零；模块的 9 条（`module-card-item.tsx` 5 条、`applied-filters/root.tsx` 2 条 `prefer-set-has`、`dropdowns/filters/root.tsx` 的 `autoFocus`、`archived-modules/modal.tsx` 的正 `tabIndex`）随之清掉。`lint-cap` 要求告警数恰好等于上限，所以**清掉告警的那个提交同时下调** `web/apps/web/package.json` 的上限（53 → 44，分在 T7、T10、T11），T12 只核对。
- 死成员：M1 收尾交接的 `--rows M6` 中模块的 18 行（P5a 台账 Task 12 列出）与本段改到的文件里的新行，各删去或在阶段记录写明。三个脚本照 P5a 写进台账目录，先跑 `react-router typegen` 再跑。
- `.oxlintrc.json` 的 `no-restricted-imports`（只能从 `swr` 导入类型）列入新的 hook、详情页、列表页、预览、归档目录与命令面板的模块菜单（v0 §7.7）。
- 新文案 en 与 zh-CN 在同一提交加入（`check:sync`、`tools/i18n-keys.mjs`）；模块的问题码在 `error-messages.ts` 已有，本段不加码。

### 2.11 e2e

页面故事放在 `e2e/stories/planning/`，标题 `G<n> (page): …`，与同号的接口故事共用 `fixtures/assert/planning.ts`、`progress.ts` 的数据库断言；世界经接口建立（`createModule`、`moduleCalls`、`progressWorld`、`createModuleLink`；新项目的 `module_view` 默认关闭，故事先打开），新夹具放 `fixtures/module-pages.ts`，与迭代共用的帮手（`toast`、`dismissToasts`、`tabStops`、`recordApi`、`menuEntries`、`pickDays`）从 `cycle-pages.ts` 导入，不复制。浏览器证据用 `watchPage`/`expectQuietConsole`、`dialogs.ts` 的 `modalDialog`、`settings-pages.ts` 的 `sentTo`/`sentHeld`/`moveWithinApp`、`fixtures/auth.ts` 的 `anotherTabSignsIn`。全部纳入三种引擎的页面配置 `e2e/playwright.m6-pages.config.ts`（T13、T14、T16 在本机以它跑并留收据）；CI 照常只跑默认配置（每个故事在 chromium 跑一次）。**时长**：T13、T14 从本机三种引擎的运行记下每个新文件在各引擎的时长；T16 推送之前据此估出 CI 的 e2e 作业时长：上一次绿的 main CI（43373039561）的 e2e 作业 19 分 14 秒，加上新文件 chromium 一栏的时长乘以迭代页面故事在 CI 与本机的时长之比（同一批故事，CI 的数取那次作业的报告），期限 35 分钟，估出超过 30 分钟时先报告负责人。T10 检查点的分支 CI 核的是守卫与到那时的代码，那时还没有新故事，量不到它们的时长。P5b 阶段记录 §5 的教训照做：应用内跳转之后以目标页独有的内容为锚；不在重新载入之后核对控制台（Firefox 的字体预载告警，M6-M8-09），需要“载入后仍在”时由同一人另开的浏览器核对；`dismissToasts` 已容忍自动关闭。

| 文件 | 覆盖 |
|---|---|
| `g9-create-module-page.spec.ts` | 页头与空态的“新建模块”；只给名字（请求体 `{name, status}`，没有日期键）与带日期、负责人、成员的模块；负责人与成员的下拉没有访客；重名被拒（409）时弹窗留着、输入保留并说明；名字的字段错误与焦点；访客没有“新建”；开关关闭的停用空态（管理员可去设置）；弹窗修改只发改了的字段；侧栏改状态、负责人、成员、日期等 nerve 确认，被拒时回到原值；弹窗的名字、Tab 次序、Escape、焦点回到按钮、弹窗里的日期选择 |
| `g10-module-links-page.spec.ts` | 加链接（不带协议的地址、空标题）、抓取之后行显示元数据的标题与内嵌图标；同一地址被拒（409）时弹窗留着；改标题；删除；访客看得到链接、能复制，没有加改删；已归档模块的链接只读；数据库断言同接口故事 |
| `g12-module-progress-page.spec.ts` | 列表行的圆环、卡片的数字与进度条、侧栏的“工作项”一致（已取消不在分母里）；燃尽（开始到目标日，未来为空）；状态、负责人、标签页签的名字、“没有负责人”“没有标签”、离开的成员的兜底名字；页签与面板一致；只看自己的访客只看到自己的数 |
| `g13-archive-delete-module-page.spec.ts` | 未完成、未取消时归档停用并说明；归档已完成的模块、归档目录、预览、恢复进入归档目录；删除：管理员与建立者可见、其他成员与访客不可见、已归档的菜单里没有；被拒（如开关在别处关闭）时确认框留着并说明；在详情页删除后回到列表；打开不存在的模块地址显示“模块不存在”、只有 `/api/v0` 的读；没有已归档模块时归档目录显示空态、不是加载占位（S2 移出的两条由此接上） |
| `g19-module-pages.spec.ts` | 另一个标签页换账号后迟到的回答不写进新页面；弹窗开着时后退/换项目/换工作区，弹窗随之结束、迟到的回答不在新页面提示；经面包屑换模块后旧模块迟到的进度不写入；快捷操作与链接一行的复制失败有提示；行菜单、排序、筛选、版式按钮的名字与 `aria-expanded`；右键菜单 |

### 2.12 文件与职责

| 路径 | 职责 |
|---|---|
| `web/apps/web/core/services/planning/modules.service.ts` | §2.1 |
| `web/apps/web/core/store/planning/` | `ModuleStore`、`module-changes.ts`、`ModuleFilterStore` 与测试、替身（§2.2、§2.4） |
| `web/apps/web/core/store/root.store.ts`、`core/lib/store-context.tsx` | 旧 store 的接线删除；`dispose()` 的说明随 R4 更新；`root.store.test.ts` 的 store 数随之改 |
| `web/apps/web/core/layouts/auth-layout/use-project-fetch.ts`、`core/hooks/use-planning-reads.ts`（名字由实现定） | §2.2 的包装层取数、§2.3 的取数门 |
| `web/apps/web/core/store/user/permissions.store.ts` | §2.3 的写的门 |
| `web/packages/utils/src/planning.ts`、`module.ts`（T12 删除）；`web/packages/constants/src/module.ts` | §2.4 的函数、类型与测试 |
| `web/apps/web/core/components/planning/`、`core/components/core/sidebar/`、`web/packages/propel/src/charts/area-chart/` | §2.5 |
| `web/apps/web/core/components/modules/**` | §2.6–§2.8，列表的显示设置的 hook（§2.4） |
| `web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/modules/**`、`archives/modules/**` | 路由页面（§2.6） |
| §2.9 所列 | 跨领域读者与删除 |
| `web/packages/i18n/src/locales/{en,zh-CN}/` | 新文案与删除的键 |
| `tools/keywords.json`、`.oxlintrc.json`、`web/apps/web/package.json` | §2.10 |
| `e2e/` | §2.11；`fixtures/mounts.ts` 的 `PROJECT_MODULES`；S2 冒烟故事移出模块的两条 |
| `docs/` | P5b 回填（T1）；`plane-diff.md`、`frontend-changes.md`；输入交接的“处理结果（M6/P6a）”；设计 §8、§12、§14；`v0-design.md` §7.7 中“一代退役时释放它的反应”一句（R4） |

## 3. 局部裁定

- **R1 拆段**：§1。P6b 的边界写在本 spec，它的 spec 与 plan 在 P6a 合入后写（设计 §12“后续阶段只在前置事实落实后写可执行 plan”）。研究笔记 §5 的 26 条交接中，第 7、8、9、21、23、26 条归 P6b（§5），其余归本段。
- **R2 新 store 并行到 T12**：`PlanningStore.modules` 先与旧的 `RootStore.module`/`moduleFilter` 并存，逐个页面改用新 store，最后一个读者改完的 Task（T12）删除旧的。每个 Task 的提交都能通过类型检查与测试；并存只在本分支上，合入的是只有新 store 的状态。这不是兼容层：旧 store 调的都是 nerve 不提供的地址，本来就不工作（先例 P5a R3）。代价：中间提交里未改的页面仍是坏的（与 main 现状相同）。
- **R3 取数门一个 hook（D6）**：re-65 M6 要求在模块再加之前把“有效成员且项目未归档（且开关打开）”抽成一个。抽取放在共用件的 T5（以迭代的既有测试为基线、在第一个模块页面 T8 之前），T2 只在包装层迭代的条件旁加模块的一条；§2.3 的表逐处写明用 hook 的哪一项，开关的其余读者不是取数门、留着自己的条件。模块的接口在开关关闭时照常可读（`getModule` 等的说明，设计 §3.8），页面照 P5a 的先例分两种：列表页、包装层与命令面板的菜单看开关（关闭时列表页是停用空态，没有要列的；Plane 的列表页同样只显示停用空态），详情页、预览与归档目录不看开关（直接打开地址时照常显示、只读，写入口由 `canWriteModules` 收起，它含开关）。理由：与迭代一致，用户在两域看到同一种行为；nerve 允许的读不在网页上藏起来。代价：开关关闭时直接打开模块的地址仍能看到它（登记差异：Plane 的详情页不看开关，照样可写）。
- **R4 版式与排序存在浏览器，筛选与搜索存在会话（D1，预检 re-68 I2 的主控裁定）**：Plane 的模块筛选 store 以 `reaction` 跟随 `router.projectId` 初始化，并把显示设置（版式、排序、收藏）与筛选写进 `localStorage`（`plane/apps/web/core/store/module_filter.store.ts:18-19,90-131` 的 `module_display_filters`、`module_filters`，按浏览器、不按账户）；Plane 的迭代筛选 store 从不写 `localStorage`，所以 P5a R4 的先例只管“去掉反应”，不管“去掉持久化”。分开处理：
  - **版式与排序**是纯展示的设置，同 `module_sidebar_collapsed`、`module-progress-tab-{id}` 一类：在组件里经 `useLocalStorage` 按项目、按浏览器存，默认 `list`、`name`，跨重新载入保留（同 Plane，§2.4）。它们不含账户的数据，同一浏览器里换账户带过去无妨。
  - **筛选与搜索**按项目 id 存在 `PlanningStore` 的 `ModuleFilterStore` 里：不跟随 router，没有要随退役的一代释放的反应（已批准的设计 §8 与 v0 §7.7 写“模块的……照迭代的做法去掉”）；不写 `localStorage`：筛选里有成员的 id，不能在同一浏览器里带到另一个账户，退役的一代也不再写它（M3 交接 §1 记下的“退役的模块筛选的反应在每次换路由时仍写 localStorage”随之消失）。
  - 代价：重新载入页面后筛选与搜索回到空（Plane 在同一浏览器里保留筛选），版式与排序照 Plane 保留；登记差异。本地决定，不须负责人。v0 §7.7 与设计 §8 的那句由 T15 改写为已完成，`store-context.test.ts` 加一条断言。
- **R5 写与删的门按 nerve 的规则；已归档的菜单里没有删除（D7）**：写入口只给 `canWriteModules`（Plane 给项目的管理员与成员，不看只看自己的访客与开关，nerve 会 403、409）；删除只给 `canDeleteModule`（Plane 给任何管理员与成员，nerve 只许管理员与建立者）。菜单的状态条件照 Plane：已归档的模块菜单里没有修改与删除（nerve 允许删除已归档的模块，“whatever its status and archive”），先例 P5a 对已结束、已归档迭代的同一取舍。理由：照 Plane 一比一的界面，只在权限上按 nerve 收窄。代价：要删除已归档的模块须先恢复；登记差异。
- **R6 负责人与成员（D2）**：nerve 对访客、已离开的成员作负责人或新成员回答 422（设计 §3.2）。核对现状：`MemberDropdown` 带 `projectId` 时只列不是访客的项目成员（`dropdowns/member/dropdown.tsx:34` 的 `getProjectMemberIds(projectId, false)`），表单与侧栏都传了项目，所以已经不给访客（调研笔记推测“没有排除”，不确）；本段以组件测试与变异钉住。新建弹窗换项目时清空已选的负责人与成员（他们是上一个项目的成员，Plane 留着，提交时 nerve 422）。成员的增减：侧栏的多选与命令面板的“加/去成员”都是一次增减，以“加入谁、去掉谁”交给 store，轮到它发出时以 nerve 最近一次回答的集合算出整集合（v0 §7.7“修改的请求体取决于 store 所持的值时，轮到它发出时才算”），连续两次增减都不丢，已离开项目的成员留在集合里（nerve 不回写，设计 §3.2）；修改弹窗是一次快照式的编辑，发表单上的整集合。代价：无。
- **R7 新建表单的项目下拉（D3）**：只列 `canWriteModules` 的项目（Plane 用工作项的 `projectsWithCreatePermissions`，包括只看自己的访客与模块关闭的项目，nerve 会拒绝），同 P5a 对迭代表单的做法。默认状态照 Plane 的网页为“积压”（两边的服务端默认都是“已计划”），请求体总带 `status`，不是差异。
- **R8 进度一个口径、一套件**：Plane 有三种口径（列表行与按进度排序是（完成 + 取消）÷ 总数，卡片与侧栏是完成 ÷ 总数），nerve 只用 `progressPercent`（设计 §3.5）。分布用 P5a 的行（按名字、兜底名字、`null` 行最后）；燃尽用 `BurndownChart`。Plane 在开始日晚于浏览器的今天或缺日期时把整个进度一节换成一句说明、打不开；nerve 有燃尽就画（未来的点为空），没有就“暂无数据”，分布照常显示（nerve 的数按调用者看得到的工作项）。页签与面板一致（共用件）。登记差异。
- **R9 写入的回答不写进列表行的计数**：`Module` 不带计数（P4 R9 的模块一侧）。建立时补 0（准确）；修改、归档、恢复只改模块的字段、保留行的计数；计数在列表下次取数时更新（与迭代相同）。
- **R10 链接的显示与元数据的读取（D8）**：nerve 的链接回答时元数据为空，抓取在后台（`planning.crawl_link`，设计 §9）；P2 把“含链接的元数据显示”交给本段（P2 spec §5）。做法照 M4 工作项链接已有的有界轮询（`issue-detail-widgets/links/use-link-metadata-refresh.ts` 的界限）：显示中的链接里有元数据为空的，就每秒重读一次本模块的链接（经 `MODULE_LINKS` 的 SWR 重新校验，不另开取数路径），至多 30 次、至多 30 秒；都有了元数据（抓取失败时元数据带 `error`，也算有）、读被拒、侧栏卸载、换会话时停止；到期后照常显示地址与按地址的图标。M4 的那个 hook 绑在工作项详情的 store 与宿主上，不改它去兼顾模块（跨 M4 的重构不在本段），模块一侧写一个只为 SWR 重读的小 hook。另：链接对所有看得到模块的人显示（修正 Plane 的缺陷）；删除不确认（同 Plane）；只显示内嵌的图标。代价：抓取慢于 30 秒时，元数据到下次打开侧栏才出现；登记差异。
- **R11 不吞拒绝**：Plane 的归档、恢复吞掉错误、失败也提示成功；删除确认框失败也关闭；新建/修改失败把表单重置为默认值；侧栏改日期不等回答就提示成功；侧栏的状态、负责人、成员失败没人接（未处理的拒绝）；行内的修改是乐观的；删除后回到列表的判断读不到预览的参数。全部改为等 nerve 确认、被拒时留着并说明（M3 交接 §3、M4 交接 M6-04“表单失败即关闭”“日期提交未等待”）。登记差异。
- **R12 已归档模块的详情可读**：nerve 允许读已归档的模块、进度与链接（Plane 的 `retrieve` 排除已归档的，回答“模块不存在”）。归档目录的行照 Plane 打开预览；直接打开已归档模块的地址时详情照常显示、只读，不在面包屑的切换列表里（先例 P5a R11）。模块的计数来自 `/progress`，详情页不需要列表行（与迭代不同，迭代的成员只在行里）。登记差异。
- **R13 日期是日历日，侧栏与行内选完一段才发**：模块的开始日、目标日是项目时区的日历日（`date`），显示、选择与比较都按日历日，不经 `new Date("YYYY-MM-DD")`；`minDate` 照 Plane。nerve 允许只有一天（设计 §3.2），弹窗里可以只选一天或清空。侧栏、列表行与卡片的日期选择在两天都选了、或清除时发出一次 `PATCH`（一对），与 nerve 的值相同不发，在途时停用（先例 P5a T8 的裁定）；Plane 每次选择都发（`analytics-sidebar/root.tsx` 与 `module-list-item-action.tsx` 的 `onSelect`：选到一半时 `to` 为空，目标日就发成 `null`，选完再发一次）。代价：在侧栏与行内不能只设一天，要只设一天经修改弹窗；登记差异。
- **R14 列表的排序与筛选照 Plane，修正它的错**：排序的选项、默认与“手动”照 Plane；Plane 的“工作项数”“目标日”“建立时刻”的降序按布尔值排序（`orderModules` 的 `!m.total_issues` 等），实际不是降序，nerve 按值降序；没有目标日的两向都在最后。负责人、成员的筛选列工作区的成员（同 Plane），但读按 slug 的 `getWorkspaceMemberIds(workspaceSlug)`、slug 取自路由参数，不读 Plane 用的 `workspaceMemberIds`（它读 router store，预检 re-68 I1；守卫见 §2.10）。登记差异。
- **R15 G15 的模块部分在 P6b（D4）**：设计 §12 的 P6 一行列 G9–G13、G19，没有 G15；G15 的迭代页部分由 P5b 交付，模块页的列表与它同属一种（列表引擎、本页偏好、双向分页、附件列），随模块页的列表在 P6b 交付；P7 一行的 G15 是工作项一侧的筛选、分组与显示。T15 把设计 §12 的 P6 一行拆成 P6a/P6b 时写明。
- **R16 工作项根 store 的拆分在 P6b（D5）**：M6-M8-08 记下 `core/store/issue/root.store.ts`（M4 的文件）615 行，P6b 要往里加模块页的列表与筛选 store、`module_ids` 的写入。P6a 不碰这个文件（本段改的是应用的 `core/store/root.store.ts`，只删两个字段，行数减少）；P6b 在加入之前先按职责把它拆开（拆法写在 P6b 的 spec，经它的预检），不留给 P7、M8。

本段的裁定都是局部决定：都在已批准的设计与 P5a/P5b 的先例之内，不改接口、不改服务端、不跨后端模块；D1–D8 对应 R4、R6、R7、R15、R16、R3、R5、R10，另加 R6 的成员增减、R13 的日期、R14 的排序。R4 经预检 re-68 I2 由主控按局部决定改定（版式与排序同 Plane 存在浏览器，筛选与搜索存在会话）。**没有须负责人决定的事。**

## 4. 验收标准

- [ ] A1 数据层：service 只调 `/api/v0`；store 按会话分代（构造时拿这一代的客户端、取数经 `useSessionSWR` 且键带 `loginId`、修改经 `oneAtATime`、副本取最后回答的一份、拒绝交给调用者、成员的增减在轮到时算）；包装层按权限与开关取数；取数门一个 hook、§2.3 表中的各处都用它、迭代的行为不变；筛选 store 没有跟随 router 的反应、不写 `localStorage`，版式与排序按项目存在浏览器；筛选的成员列表不读 router；每个性质有单测，权限门与迟到应答各有能让测试失败的变异。
- [ ] A2 进度：`progressPercent` 是唯一口径，列表行、卡片、侧栏、按进度排序一致；燃尽图与分布用 P5a 的件，页签与面板一致；`ProgressChart`、`TModuleCompletionChartDistribution`、`progress-stats/*` 的三个文件与 `comparisonLine` 删除。
- [ ] A3 G9、G10、G13 的页面：建、改、删、归档、恢复、链接的加改删按 §2.6–§2.8，被拒时留着并说明；权限与状态条件按 R5；负责人与成员的下拉没有访客；元数据按 R10 出现。
- [ ] A4 G12 的页面：列表行、卡片、侧栏的计数、百分比、燃尽、分布与只看自己的访客的数。
- [ ] A5 G19（模块页面）：换账号、换工作区/项目/模块、后退前进时迟到的回答不写进新页面；详情子树以 `moduleId` 为 key、预览侧栏以所预览的模块为 key；弹窗、菜单、下拉框的键盘、名字与焦点逐项（M3 交接 §3 的模块各项）；两处复制失败有提示。
- [ ] A6 删除与守卫：§2.9 的旧代码全部删除；`old-module-urls` 规则；`project-address-route` 收入改写的文件、加上嵌套解构的一支；`copy-hook` 模块的三条例外删除；改到的文件 oxlint 告警为零、上限下调到 44；模块的死成员与 progress-stats 8 行有结论；`root.store.test.ts` 21 个 store。
- [ ] A7 文档：差异清单、前端改动清单、输入交接的“处理结果（M6/P6a）”、设计 §8/§12/§14、总体设计 §7.7 一句；P5b 的阶段记录 §7、spec A8 与设计 §14 已回填。
- [ ] A8 七项本地门禁（`make test` 照负责人 2026-11-11 的规则）、deep-review 修复闭环、准确的分支 CI 与 noff 后的 main CI 全绿、归档清理。（合入后由 P6b 的 T1 勾选）

## 5. 后续责任

**P6b**（模块页的工作项列表）：§1 所列；研究笔记 §5 的第 7 条（模块页列表照 P5b：`EIssuesStoreType.MODULE`、偏好的筛选 store 用同一基类，成员判断的 `module_id` 已在 P5b）、第 8 条（模块页页头的版式、筛选、显示在本页偏好读到之后才出现，re-67 M8；项目页同因）、第 9 条（读到过偏好之后的重读失败时正文是否沿用已读的偏好、只提示重读失败，项目页与迭代页随之，re-67 限定复审 M1）、第 21 条（M4 交接 M6-02 的模块部分：同 fixture、双向分页、跨分组写后计数、OwnOnly 与工作区管理员兼访客、失权迟到答、归档范围、换会话）、第 23 条（M5 交接 M5-M6-02 的模块部分：附件列）、第 26 条（M6-M8-08：先拆工作项根 store，R16）；`module_ids` 整集合的写入（创建与“移出模块”，请求体由行当时持有的完整集合算出，保留已归档、看不到的模块）；侧栏的行切换本页筛选（`getFilter(EIssuesStoreType.MODULE, id)`，Plane 的 `handleFiltersUpdate`）；模块页偏好的 `user-properties`（M3 交接 §1）；G15、G19 的模块页列表部分。

**P7**：工作项一侧（模块的属性与下拉框、新建弹窗的模块、列表词表、添加已有、`existing-issues-list-modal.tsx` 的模块调用方）用本段的 `useModules()`；列表引擎的目录（P3b R9 的空 `Map`）改由本段的 store 提供；命令面板模块菜单的本地过滤与切换；命令面板“打开模块”在取数门为假（已归档的项目、不是成员）时菜单一直加载（与迭代的同一条，re-65 M7，并入 P5a §5 的 P7 一条）。

**P8**：模块表单里项目选择（`dropdowns/project/base.tsx`）的浏览器核对随最终矩阵（M2 交接 §5(d)，本段有组件测试）；“按规则清一类”的 oxlint（M1 收尾交接）。

**M7**：模块的收藏随收藏一起恢复（本段删除了入口）；工作区级的模块列表（通知页、工作区视图）。

## 6. 风险

- **大范围的类型替换**：读旧的模块类型、store 或 hook（`IModule`、`TModule*`、`ILinkDetails`、`useModule`、`useModuleFilter`、旧 service）的 web 文件有五十余个（含几个测试与它们自己的定义）。R2 的并行让每个 Task 只改它负责的页面；T12 以类型检查、knip 与 `old-module-urls` 确认没有残留。
- **共用件的重构碰坏迭代**：§2.5 的移动与抽取改的是迭代的现有组件，迭代的既有测试断言不改，作回归保护；取数门同理。
- **成员的整体替换**：连续的增减若在调用时算请求体会丢掉前一次（R6）；store 的单测以两次在途的增减钉住，变异“调用时算”使它失败。
- **列表行与详情的副本**：写入的回答（`Module`）不带计数，合并时不能把行的计数写成 0 或 `undefined`（R9）；store 的单测钉住。
- **元数据的轮询**：有界（30 次、30 秒），卸载与换会话停止；单测以假时钟钉住次数、间隔与停止，变异“不设上限”使它失败。
- **e2e 时长**：新增 5 个页面故事；CI 的 e2e 作业期限 35 分钟（负责人 2026-11-07 同意的值），上一次绿的 main CI 的 e2e 作业 19 分 14 秒。T10 检查点时还没有新故事，量不到它们；T13、T14 记下每个新文件在本机各引擎的时长，T16 推送之前照 §2.11 估出 CI 的时长，估出或实测超过 30 分钟先报告负责人，不自行改期限或作业结构（M6-M8-06 的教训）。
