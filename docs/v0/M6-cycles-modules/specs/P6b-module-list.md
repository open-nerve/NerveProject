# M6 / P6b：模块页的工作项列表

> 状态：实现与审查已完成，待分支 CI 与合入：整分支审查 Ready after fixes（[re-71](../../../reviews/re-71-M6P6b整体审核.md)），修复轮 F（F1–F9）经限定复审 Ready to merge，最终候选 `dd98bcaf`；A1–A7 已勾选，A8 由 P7 的 T1 勾选（[阶段记录](../reviews/P6b-module-list-review.md)）。预检通过（修正后，[re-70](../../../reviews/re-70-M6P6b预检.md)）。基线 main `06b72d4b65bcba911344a4ed3c358f164a5fdeeb`（P6a 的 noff，双父 `aadaf61a`、`8188582e`；P6a 的 main CI 43469787621 attempt 1 的 e2e 因 M4 的 C7 未过，照 [M6-M8-01](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-01) 重跑一次，attempt 2 通过，由本段 T1 回填）。设计见 [M6 设计](../M6-design.md) §2 的 G15、G19，§3.3、§7、§8、§11、§12 的 P6b 一行、§15 第 12 条；P6b 的边界见 [P6a spec](P6a-web-modules.md) §1、R15、R16、§5 的 P6b 一段；迭代页的先例见 [P5b spec](P5b-cycle-list.md)（R1–R11、§5 的 P6 一段）与[阶段记录](../reviews/P5b-cycle-list-review.md)。网页的长期规则见[总体设计 §7.2、§7.7](../../v0-design.md#77-stores-按会话分代长期有效m2-起)。接口合同见 [P3a spec](P3a-issue-membership.md)（工作项的 `module_ids`）、[P3b spec](P3b-list-engine.md) §2.1、§2.3、R8、R9（列表的 `module_id` 与模块页偏好）。现状调研见本阶段台账 `research/web.md`（网页的现状、P5b 的接法、工作项根 store 的拆法、交接各项、e2e 与守卫）、`research/plane-contract.md`（Plane 的模块页列表、nerve 的契约、差异）（ignored，随阶段归档）。负责人 2026-11-11 授权连续推进至 M6 收官，重要高风险决策仍停下确认；本段不改 `server/`、`api/`，本机 `make test` 若因 M6-M8-05 的签名失败，以分支 CI 的服务端两路为准（负责人 2026-11-11）。

## 1. 目标与边界

P6a 交付了模块的页面，详情页的正文只有侧栏（[P6a spec](P6a-web-modules.md) §1）。P6b 把模块页的工作项列表接上 M4 的列表引擎，做法照 P5b 的迭代页：

- **先拆**：往工作项根 store（`core/store/issue/root.store.ts`，615 行）加模块的任何东西之前，按职责把它拆开（[M6-M8-08](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-08)、P6a R16，本段 R1）。
- **共用**：P5b 只为迭代页写的列表件，与模块页只差领域的，改为两域共用的一件；行为真不同的分开（R2，§2.2 的表）。
- **列表**：列表引擎的 `EIssuesStoreType.MODULE`；模块页自己的偏好（`/me/modules/{id}/preferences`，M3 交接 §1 的模块 user-properties）；四种布局（列表、看板、日历、表格）；页头（数目、布局、筛选、显示、新建工作项）；空态；已归档模块的列表只读；本页新建（页头、快速新建、分组头）一次发出，模块可加入时带本模块；行的“移出模块”（`module_ids` 整集合，请求体轮到时算）；写后重读进度；附件列（M5-M6-02）；在本页不出现“按模块分组”（P3b R8）；侧栏的行切换本页筛选。
- **三页的偏好读取**：页头的布局、筛选、显示在本页偏好读到之后才出现（P6a 研究笔记第 8 条，re-67 M8）；读到过之后的重读失败时正文沿用已读到的偏好、只提示重读失败并可重试（第 9 条，re-67 限定复审 M1）。两条都用于模块页、迭代页与项目页（R3、R4）。
- **验收**：G15 与 G19 中模块页列表的部分（P6a R15，页面故事，三种引擎）；M4 交接 M6-02 与 M6-01 的模块部分、M5 交接 M5-M6-02 的模块部分、M3 交接 §1 的模块 user-properties、M6-M8-08。

**不做**：
- “添加已有工作项”（空态、分组头、日历的 `addIssuesToView`）、`existing-issues-list-modal.tsx` 的模块调用方，以及工作项详情、列表行、新建弹窗里的模块下拉框与列表词表（筛选、分组、排序里的迭代与模块），都在 P7（设计 §12 的 P7 一行、§15 第 12 条，R10）。
- 模块页的移动端页头在 P7（P5b 对迭代页的先例）。
- 在模块页的偏好里按别的模块筛选（Plane 可以）：nerve 的模块页偏好以 400 拒绝 `module_id`，本段不改接口（R8）。

## 2. 交付合同

### 2.1 工作项根 store 的拆分（R1）

**现状**（`core/store/issue/root.store.ts`，615 行）：组成（9 个 service、各页的 store、`PermissionReads`）、router 与目录的 getter、一代的身份与退役、读的凭据与可见性（`captureRead`、`isReadCurrent`、`scope`、`assertProjectReadable`、`canSeeIssue` 等）、列表与资源的登记、成员的快照（私有的 `fields()`，`:361-382`）与读的观察（`observeRead`，`:346`），以及写的一侧：每个工作项一条的写队列（`enqueueIssueChanges`，`:233-276`）、确认的流水线（私有的 `write()`，`:383-509`，127 行）、六个命令与两份请求体的白名单（`:510-614`）。

**拆法**：写的一侧整体移到一个写入单元（`core/store/issue/` 下，名字由实现定，例如 `IssueWrites`，由根在构造时建立并持有）：

| 移到写入单元 | 留在根 |
|---|---|
| 每个工作项一条的写队列与 `enqueueIssueChanges`；拖拽的队列；在途的写（`pending`）与 `pendingIDs`；确认的流水线（原 `write()`）；`createIssue`、`confirmCreation`、`updateIssue`、`deleteIssue`、`archiveIssue`、`unarchiveIssue`；创建与更新的请求体（白名单与 `skip_activity` 的规则，写成纯函数） | 组成与各页的 store；router 与目录的 getter；`assertActive`、`retire`；`captureRead`、`captureWorkspaceWrite`、`isReadCurrent`、`scope`、`assertProjectReadable`、`canSeeIssue`、`acceptListIssues`；`registerList`、`registerResource`、`listsForIssue`、`markScopeListsStale`、`invalidateProjectReads`；`observeRead`（工作项有在途的写时交写入单元记下这次读，否则按成员的快照标记列表） |

- 成员的快照（原 `fields()`）改为一个纯函数，根的 `observeRead` 与写入单元的确认共用这一份。
- 写入单元经根读取它登记的列表（只读的一个访问，不另存一份）、工作项 store、权限读与详情；类型随职责走（`IssueDrop` 等随写入单元，`IssueWriteScope` 随根的 `scope()`），导入随之改。
- **没有值的循环导入**（re-70 F3）：写入单元对根只有 `import type`，或只依赖构造时传入的一个窄接口（根把自己或所需的几样传进来）；根以值导入写入单元来建立它。不出现 `root.store.ts` → 写入单元 → `root.store.ts` 的值导入（“没有循环依赖”的质量要求）。
- **不加入 `project-address-route`**（re-70 F5）：写入单元带着 M4 的 router 退路（`this.workspaceSlug`，自 `:552` 原样搬来），规则的内容模式也认不出这种读法，列进去反而误导；P7 去掉这条退路时再加入（§5 的 P7）。
- **调用方改为经写入单元**（R1 的选项 B）：写 spec 时 grep 到产品代码 12 个文件 18 处（`helpers/base-issues.store.ts` 6 处、`cycle/issue.store.ts`、`workspace-draft/issue.store.ts`、`issue-details/` 的评论、链接、表情、关系（2 处）、子项与描述版本 store、`issue-list-binding.ts` 的 `pendingIDs`、命令面板的 `mutation-commands.ts`、`issue-detail/use-issue-operations.ts`），测试约 22 个文件 84 处（预检 re-70 的计数，含 `vi.spyOn` 这些方法的与测试替身 `fake-issues.ts`）。合入的状态里根上没有这些方法的转发。
- **行为不变**：只搬不改，包括 `updateIssue` 没有给出范围时退回 router 的工作区（`:552`，M4 起如此；§5 的 P7）。M4、M5 与 P5b 的测试是回归保护，断言不改。
- **大小**：拆后根与写入单元各在约 400 行以内（写 spec 时估：根约 330 行，加上本段模块页的两个 store 约 340 行；写入单元约 300 行）。之后加写入（P7 的迭代、模块下拉框与“添加已有”）加在写入单元，不加回根。

### 2.2 迭代页列表件的共用（R2）

P5b 的迭代页列表件逐个处理如下（“共用”指改为按域的一件，迭代与模块都用；迭代的测试只改导入路径、构造与调用处的名字，断言不改）：

| P5b 的件 | P6b | 理由 |
|---|---|---|
| `core/store/issue/cycle/issue.store.ts` 的本页列表（`pageList`、`isPreferenceQueryCurrent`、`pageTotal`、取首页、续页、按现有分页重取） | 共用：抽到 `helpers/` 下按本页的项（`cycle_id` 或 `module_id`）取的一件，迭代与模块的列表 store 继承它；本页的形状（现为 `CyclePage`）改为两域共用 | 只差查询里本页的那一项 |
| 同文件的 `createIssue`（加入本页） | 分开：迭代加 `cycle_id`；模块把本模块并进 `module_ids`，另有“移出模块”（§2.5） | 迭代是单值、模块是整集合（设计 §3.3） |
| `core/store/issue/cycle/filter.store.ts`（`CycleIssuesFilter`，16 行） | 删除：两域都以 `PagePreferencesFilter`（service、store 类型）直接构造 | 子类除这两个参数外没有内容 |
| `core/services/planning/cycle-preferences.service.ts` | 分开：新 `module-preferences.service.ts` | 每个接口操作一个方法、类型来自它的操作（v0 §7.2）；两域 `filters` 的类型互为镜像（迭代页的没有 `cycle_id`、有 `module_id`，模块页的相反），合成一个会抹掉让本页的键进不去的那点差别 |
| `core/hooks/use-cycle-issue-actions.ts` | 共用：按域的一个 hook；加入的条件与移出各取本域的 | 只差 store 类型、路由参数、可加入的判断与跟进的 store |
| `core/hooks/use-cycle-list-state.ts` | 分开：新 `use-module-list-state.ts`，同形的 `{readOnly, joinable}`；共用件按域取 | 规则不同：迭代已结束或已归档只读；模块只看已归档，不看状态（设计 §3.3“模块不限状态”） |
| `core/hooks/use-cycle-preferences-follow.ts` | 共用：按域，仍建立在 `use-page-preferences-follow.ts` 上 | 只差 store 类型、路由参数与页面范围的名字 |
| `core/hooks/use-cycle-progress-follow.ts` | 共用：按域 | 只差进度所在的 store 与页面范围的名字 |
| `issue-layouts/roots/cycle-layout-root.tsx` | 共用：按域的布局根；它不导入 `core/components/cycles/peek-on-entry.tsx`，迭代页把进入时打开预览（`PeekOnEntry`）作为子节点或插槽交给它（或在它旁边渲染）（re-70 F4） | 只差 store 类型、取数门的域、SWR 键、显示选项常量、预览宿主的前缀；进入时打开预览只来自进行中卡片，模块没有 |
| `{list,kanban,calendar,spreadsheet}/roots/cycle-root.tsx`（各 38 行） | 共用：每种布局一件；只读的来源与行的快捷操作经 props 或一张按域的小表给出，不写 `if (domain === …)` 的分支（re-70 F4） | 只差只读的来源与行的快捷操作 |
| `issue-layouts/roots/fake-cycle-page.tsx`（测试替身） | 共用：按域 | 同上 |
| `issue-layouts/empty-states/cycle.tsx` | 共用：文案、列表状态与 store 类型经同一张按域的小表（或 props）给出 | 只差这三者（文案见 R11） |
| `quick-action-dropdowns/cycle-issue.tsx`、`command-host.tsx` 的 `remove-from-cycle`、`helper.tsx` 的菜单项 | 共用：“移出本页”，按域的菜单文案与成功提示；移出经本域的命令 | 只差文案与命令；模块的命令不能是行的 `handleUpdate`（它发调用时固定的请求体，§2.5） |
| `R/cycles/(detail)/header.tsx` 的数目、列表控件与新建的三个条件 | 共用：三个派生值抽成按域的一个 hook，两个页头各自排版 | 条件只差域；页头的其余部分（面包屑、侧栏开关、快捷操作）P5a、P6a 已各有一份 |
| `cycles/progress-sidebar/progress-stats.tsx` 的行切换 | 共用：按 store 类型与 id 给出 `ProgressTabs` 的 `choices`（`core/components/core/sidebar/` 下） | 只差 store 类型与 id |
| `@nerve/constants` 的 `ISSUE_DISPLAY_FILTERS_BY_PAGE.cycle_issues` | 分开：新 `module_issues` | P7 加词后各页只藏自己的键（Plane 的模块页只藏“模块”，R12） |
| `cycle/priority-issues.store.ts`、`cycles/active-cycle/*`、`use-enter-filtered.ts`、`peek-on-entry.tsx` | 迭代独有，不动 | 模块没有进行中卡片 |
| e2e 夹具 `fixtures/cycle-list-pages.ts` 的 `cycleListWorld` 与列表页的帮手 | 共用：按域的世界（迭代或模块打开，访客的工作项在迭代或模块里），帮手从同一处导入，不复制；T13 把夹具改名为 `fixtures/planning-list-pages.ts`（私有的 `listPageWorld` 之上的 `cycleListWorld`、`moduleListWorld`） | P6a §2.11 的规则；迭代页故事只改调用处的名字与导入 |

新的共用文件的名字由实现定；旧的迭代路径留在 `project-address-route` 的文件表里阻止复活（M4/P12 的做法）。

### 2.3 列表即项目列表加 `module_id`

- 模块页的列表是项目列表加上 `module_id=<本模块>`（设计 §7）：`IssueListScope` 仍是 `scope: "project"`，查询多一项 `module_id: [moduleId]`（P5b R1）；绑定的键是整份 scope，每个模块各有一个绑定。请求 `GET /api/v0/projects/{project_id}/issues`：本页偏好得出的筛选、分组、排序、子工作项，加 `module_id`、`limit`、游标。工作项的未删除的归属都算，已归档的模块照样可列（接口说明）。
- 成员判断与 `fields()` 的 `module_id`、`module_ids` 已在 P5b（R2）：别处的写或读让一个工作项进出本模块，本页列表按 nerve 的回答增减，不在网页上推算。
- 取数：取数门 `usePlanningReads(…, "modules")` 的“列表可取”（调用者在项目里有角色、项目未归档、`module_view` 打开）；访客与只看自己的访客照常读，nerve 只给他看得到的（列表与总数都按 OwnOnly）。挂载键带模块 id。
- **模块关闭时**：页头与侧栏照 P6a（只读，写入口已由 `canWriteModules` 收起），正文的列表区是停用空态（`PlanningDisabled`，`domain="modules"`），不取本页偏好与列表，页头没有数目、布局、筛选、显示与新建（P5b 的 re-66 H1）。nerve 在模块关闭时拒绝任何 `module_ids` 的变化（`planning.modules_disabled`）。
- **按本页分组**：服务端接受 `group_by: "module_id"`，结果是本模块一组加上本页工作项所在的其他模块各一组（多值分组按全部归属，`list_issue_groups.sql:97`）；网页的词表在 P7 之前没有它，经接口写入的照 P3b R9 显示为不分组、修改时保留（R12）。

### 2.4 模块页的偏好（M3 交接 §1）

- `core/services/planning/module-preferences.service.ts`（新）：`get(moduleId)`、`update(moduleId, ModulePreferencesUpdate)` 调 `getModulePreferences`、`updateModulePreferences`，构造时接收这一代的客户端，经 `unwrap`，不转换数据（先例 `cycle-preferences.service.ts`）。
- 筛选 store：`IssueRootStore` 以 `PagePreferencesFilter`（这个 service、`EIssuesStoreType.MODULE`）建立 `moduleIssuesFilter`，按模块 id 存取，不读 router、没有反应，随这一代退役（`assertActive`）（R2）。读的先后、整字段替换、被拒、看板折叠的本地值都是基类的（P5b R3）。
- 读取：在布局根经 `useSessionSWR(["MODULE_ISSUE_FILTERS", workspaceSlug, moduleId])`，只在“列表可取”时；模块关闭时不读。
- `ModulePreferences` 只有 `filters`、`display_filters`、`display_properties`；`filters` 没有 `module_id`（接口以 400 拒绝，R8），有 `cycle_id`；网页的筛选词表两者都没有（P3b R9），经接口存下的 `cycle_id` 网页不显示，整字段替换 `filters` 时随网页的条件替换掉（同迭代页对 `module_id`）。
- 偏好修改确认后，本页列表若查询变了就按现有分页重取（共用的偏好跟进，§2.2）。
- 网页里没有 `/user-properties/` 的地址（`old-module-urls` 已拦住）。

### 2.5 写入的请求体（R5、R6、R7）

- **白名单**：写入单元的创建与更新的请求体放行 `module_ids`（更新的不可为 `null`，省略即不改）。只是放行，不在网页上判断能否加入：nerve 的拒绝（`planning.modules_disabled`、`planning.archived`、422 `module_ids[i]`、`project.archived`）照常经各入口的拒绝提示。
- **本页新建一次发出**（R6）：模块页的列表 store 的 `createIssue` 在给出本页、目标项目就是模块的项目时，把本模块并进请求体的 `module_ids`（调用者给了集合就取并，去重）；目标是别的项目时不带（否则 nerve 以 422 `module_ids[i]` 拒绝，而表单在 P7 之前没有模块字段可落这个错）。模块页的动作只在模块按页面所见可加入（§2.7）时给出本页。全局的“新建工作项”（侧栏、命令面板，不指定 store 类型）在模块页的路由上经 MODULE 的动作，同样只在可加入时带上本模块，否则照项目页新建（P5b T5 之后的主控裁定）。页头、空态与分组头打开的新建弹窗锁定项目。
- **移出模块**（R5）：模块页的列表 store 另有一个移出的命令，经写入单元的 `updateIssue` 发出，请求体在队列轮到它时由所持的工作项算出：`module_ids` = 所持的 `module_ids` 去掉本模块（已归档的模块、别的模块都保留；nerve 只检查新加入的，留下与离开的不检查）。不经 `BaseIssuesStore.updateIssue`，也不经行的 `handleUpdate`：两者发的都是调用时固定的请求体。nerve 确认后行经成员判断离开本页列表、提示成功、重读本模块的进度；被拒时行留着并说明原因。接口没有前置条件，两次读之间别处的修改仍可能被这次整体替换盖掉（v0 §7.7 的“轮到时才算”缩小而不消除这个窗口）。
- **修改与复制不覆写模块集合**（R7）：行的“修改”以工作项本身打开弹窗（不像 Plane 那样把 `module_ids` 换成只有本模块）；表单的字段（`issue-modal/types.ts:19-30` 的 `ISSUE_FORM_FIELDS`）没有 `module_ids`，修改的请求体不带它。“复制一份”的初值（`command-host.tsx` 的 `copyOf`）没有 `module_ids`，经模块的动作新建时只加入本模块；描述带图片的经内容复制建立，不加入本模块（同迭代页，§5 的 P7）。拖拽的请求体不碰 `module_ids`（P7 之前没有按模块分组）。

### 2.6 列表引擎的 MODULE

- `EIssuesStoreType.MODULE`；`IssueRootStore` 建立 `moduleIssuesFilter`（§2.4）与 `moduleIssues`（共用的本页列表加模块的新建与移出，§2.2、§2.5）；`useIssues`、`useIssuesActions`（模块的动作经共用的按域 hook）、`ISSUE_STORE_TO_FILTERS_MAP`（`module_issues`）、`TCreateModalStoreTypes`、拖拽的 store 类型、四个 base root 的 store 类型、`HeaderFilters`、`usePageFilterExpression`、命令宿主的 `DialogStore`、`WorkItemQuickActions` 的 `storeType`、空态的分支都加 MODULE；模块页的路由映射到 MODULE（现在映射到 DEFAULT，测试钉住，按契约改）；`useIssueListMount` 的键带模块 id；`usePageIssueFilters` 的 MODULE 按路由参数的 `moduleId` 读本页筛选。
- 分组头的“+”与全局新建弹窗锁定项目的条件由“是迭代页”改为“是迭代页或模块页”（`list/headers/group-by-card.tsx`、`kanban/headers/group-by-card.tsx`、`modals/work-item-level.tsx`）。

### 2.7 页面

- **正文**：模块页的正文在侧栏左边放列表（共用的布局根，域为模块），侧栏照 P6a 浮在右边（同迭代页）。布局根提供 `IssuesStoreContext` MODULE、筛选的实例（`ProjectLevelWorkItemFiltersHOC`，`entityType` MODULE、`entityId` 模块 id，筛选项 `module_issues.filters`；保存视图照迭代页）、已选筛选条、四种布局的根、工作项预览（`hostScope` 为 `["module", 工作区, 项目, 模块]`，不与项目页、迭代页共用）。没有进入时打开预览（模块没有进行中卡片）。偏好第一次读失败时说明并可重试（`PlanningReadFailure`）；之后的重读失败见 §2.8。
- **可加入与只读**：`readOnly` = 页面还没有这个模块，或模块已归档（`archived_at` 不为 `null`）；`joinable` = 不只读且 `module_view` 打开。模块的状态（已完成、已取消等）两者都不看（设计 §3.3“模块不限状态”，Plane 同）。
- **页头**（P6a 的面包屑、侧栏开关与快捷操作之外加）：本页的工作项数（绑定给出的 nerve 的总数 `pageTotal`，只在当前列表是本页的时给出；提示文案为新键）；四种布局、筛选开关、显示（显示选项用 `module_issues`，R12）在本页偏好读到之后才出现（§2.8）；“新建工作项”给能建工作项的人（项目的 `allowPermissions([ADMIN, MEMBER])`，网页以工作区管理员为项目 ADMIN）且模块可加入时，打开锁定项目的弹窗（`toggleCreateIssueModal(true, MODULE)`）。模块关闭时这些都不显示。新加的列表控件与新建照迭代页头只在中等以上的宽度显示（移动端页头在 P7）；侧栏开关与快捷操作照 P6a。
- **只读**（已归档的模块，R9）：没有行内修改、快速新建、分组头的新建与拖拽；日历只读；行的快捷操作只有新标签页打开、复制链接；页头没有“新建工作项”；空态没有操作。机制是 P5b 给四个 base root 的只读 prop。本页偏好照常可改（布局、筛选、显示、侧栏的行切换），nerve 允许。
- **空态**（R11）：已归档时只有说明（新键）；有筛选时“清除筛选”（看得到页面的人都有，同迭代页）；否则 `project_empty_state.module_work_items.{title,description}`，能建工作项的人在模块可加入时有“新建工作项”（`cta_primary`）。
- **行的快捷操作**：项目页的快捷操作之外加“移出模块”（新键 `issue.remove.module.label`）：能改工作项的人（工作项项目的 ADMIN、MEMBER）、列表不只读、模块可加入时显示；经命令宿主一次一个，确认后提示 `issue.remove.module.success`。修改与“复制一份”的弹窗经命令宿主以 MODULE 打开，行在核验中离开页面时弹窗不随之关闭（P5b T13）。
- **写后重读进度**（R13）：模块页列表的动作（新建、修改、移出、删除、归档、拖拽）经 nerve 确认后，经 `followInSession` 与页面范围的捕获重取本模块的 `/progress`（`MODULE_PROGRESS`，`ReconciledByKey` 取最新的回答）；被拒的写不重读；预览里经工作项详情的写不是列表的写，进度到下次读取更新。没有乐观的分布（M4 交接 M6-02 不许恢复）。
- **侧栏的行切换本页筛选**（R14）：模块页上，进度侧栏的状态页签的组行、负责人行、标签行切换本页筛选里该值（已选则去掉，条件空了去掉条件），经本页筛选的实例（`getFilter(EIssuesStoreType.MODULE, moduleId)`）保存；行是以行命名的按钮、以 `aria-pressed` 说明是否已选；本页的筛选实例还没有时（偏好未读到、模块关闭）与列表页的预览（`?peekModule`）里行不可点；“无负责人”“无标签”行不可点（照 Plane）。
- **访客与工作区管理员兼访客**（R15）：访客看得到列表（只看自己的访客只看到自己的），数目照 nerve；没有新建与移出；布局、筛选、显示照常可用（偏好不属于 planning 的写）。工作区管理员兼访客只看到自己的，可新建、可移出自己的（P5b T14 的裁定）。

### 2.8 三页的偏好读取（第 8、9 条，R3、R4）

- **第 8 条**：页头的布局、筛选、显示只在本页偏好读到之后出现，与正文同一条件：模块页（新）、迭代页（P5b 的 re-67 M8 已如此）、项目页（现在 `core/components/issues/header.tsx:93` 无条件给出 `HeaderFilters`，改为在 `projectIssuesFilter.getIssueFilters(projectId)` 有值之后）。读到之前、第一次读失败时都不给。
- **第 9 条**：读到过偏好之后的重读失败（例如重连时的重新校验、再次挂载时的读）时，正文沿用已读到的偏好，列表照常显示，列表上方一行提示重读失败（原因经 `useRefusalMessage`）并可重试（重读本页偏好）；页头的控件仍在。三页同此。第一次读失败照旧：模块页、迭代页是可重试的 `PlanningReadFailure`；项目页是 M4 的 `role="alert"` 文字（`roots/project-layout-root.tsx:75`），本段不改。基类在读失败时本来就不改所持的值（`fetchFilters` 只在答到时写），所以改动在三个布局根，经一件共用的提示（名字由实现定）。

### 2.9 守卫与告警

- `project-address-route`：新建与改写的组件、hook、store 随各自的 Task 加入文件表并带样例；新文件只经路由参数或 props 取工作区、项目、模块。规则已列的 `empty-states/module.tsx`、`quick-action-dropdowns/module-issue.tsx` 与被共用件取代的迭代路径留在表里阻止复活。写入单元本段不加入（§2.1，re-70 F5）。
- `.oxlintrc.json` 的 swr 限制（只能从 `swr` 导入类型）：加入共用的布局根（取代 `cycle-layout-root.tsx`，后者今天不在其中）与 `cycles/active-cycle/priority-issues.tsx`（今天不在其中、本段不改，由 T1 加入），以及本段新的用 `useSessionSWR` 的文件。只收紧。
- 新文案 en 与 zh-CN 在同一提交加入（`check:sync` 也不许一个键是另一个键的前缀，`tools/i18n-keys.mjs`）：`issue.remove.module.label`、模块页数目的提示、已归档模块的空态、重读失败的提示。`project_empty_state.module_work_items.cta_secondary` 与 `issue.remove.module.{loading,failed}` 不读（迭代的同名键也不读），交 P7（R11、§5）。
- 告警上限 `lint-cap` 恰为 44，只降不升：清掉告警的提交同时下调。knip：新文件在第一个读者出现的 Task 里建立，只给测试读的导出不算读者（本阶段由审查执行）。
- 按本页分组（R12）：`module_issues` 不含模块，测试钉住。
- 关键词 phase 仍是 `M6/P6`；`root.store.test.ts` 的 store 数仍是 21（模块页的两个 store 在工作项根之内）。

### 2.10 e2e

页面故事放在 `e2e/stories/planning/`，标题 `G<n> (page): …`，纳入三种引擎的页面配置 `e2e/playwright.m6-pages.config.ts`（文件名须合 `planning/g\d+-[\w-]+-pages?\.spec\.ts`）；CI 照常只跑默认配置。世界经接口建立（新项目的 `module_view` 默认关闭，故事先打开）；夹具按域共用（§2.2 的最后一行，T13 起为 `fixtures/planning-list-pages.ts`），与 P6a 共用的帮手（`dismissToasts`、`toast`、`tabStops`、`recordApi`、`seenBy`、`itemMenu`、`entriesOf`）从原处导入，不复制；名字冲突（`module-pages.ts` 与列表页夹具各有一个 `layoutButton`）时改名，不留别名（T13 把列表页夹具的一个改为 `workItemLayoutButton`）。数据库断言用 `fixtures/planning-lists.ts` 的 `memberships` 与 `fixtures/assert/issue-planning.ts` 的模块断言。

| 文件 | 覆盖 |
|---|---|
| `g15-module-list-page.spec.ts` | 模块页的列表在四种布局里只列本模块的工作项（别的模块的、没有模块的不出现，同在两个模块的在两页都出现），每种布局写进本页偏好；分组与显示属性改后由同一人另开的浏览器载入仍在，项目页的偏好不变；已归档模块的页面同样列出且只读；模块关闭时载入不读偏好与列表；每种布局以 `expectListPageQuiet` 收尾（M4 M6-01 的“整页零非预期失败”）。51 个工作项、两个带真实附件（1 个、3 个）：接口按附件数两个方向续页不重复、不丢、最后一页没有游标（`expectPages`，带 `module_id`），页面按分组的游标两个方向续完；表格的附件列与表头排序（`aria-sort`），换会话（另开的浏览器重新登录）后照常，增删附件后数目随读取更新（M5-M6-02）；按状态分组时跨组改一行，两组的数随 nerve 变化；在本页归档一行，行离开、数目随之（归档范围）。页头新建与快速新建各只发一个带本模块的 `module_ids` 的请求；一个工作项同在本模块、另一个模块与一个已归档的模块，移出本模块的请求体是另两个，数据库里只有本模块的归属被移除；行的修改弹窗不随核验关闭，请求体不带 `module_ids`；已归档模块的页面没有这些入口 |
| `g19-module-list-page.spec.ts` | 页头的布局、筛选、显示与新建可用 Tab 到达、以名字命名；列表读被扣住时经面包屑换模块、后退前进、换到别的项目的模块页，各页只列自己的，迟到的回答不进新页面；列表重读被扣住时调用者被移出项目，页面说没有权限，重新加入后列出 nerve 此刻的工作项，迟到的回答不起作用；另一个标签页登录工作区管理员兼访客 wanda（只看自己的）：页面跟着换会话、列出她自己的、数目 1，她可新建、可移出自己的；纯访客 gus 只看到自己的、数目 1，没有新建与移出；侧栏的行把 mem 加入本页筛选又去掉（`aria-pressed`），模块列表页的预览里行不可点 |

**时长**（re-70 F1）：期限 35 分钟，报告线 30 分钟。基线是最后一次绿的 main CI 的 e2e 作业：P6a 的 main CI 43469787621 attempt 2 的 e2e 作业 132062963902，**25 分 00 秒**（17:51:22Z–18:16:22Z；attempt 1 的 e2e 作业 24 分 17 秒，失败于 C7；P6a 的最终分支 CI 为 21 分 38 秒）。T13、T14 从本机三种引擎的运行记下每个新文件在各引擎的时长；T16 推送之前据此估出 CI 的 e2e 作业时长，三个数一起看：
- **估算**：基线加上新文件 chromium 一栏的时长乘以页面故事在 CI 与本机的时长之比（P6a 记下的 1.67–2.25），两个 worker 摊开；
- **上界**：基线加上新文件 chromium 一栏的时长乘以 P6a 实际的增量比——main 到 main 由 19 分 14 秒到 24 分 17 秒、25 分 00 秒，约 303–346 秒，除以 P6a 新故事本机 chromium 的 83.5 秒，约 3.6–4.1 倍（含 runner 的差异，P6a 原先的估算是加 1.2–3.1 分钟，低估了）；
- **校准点**：T10 检查点的分支 CI 实测的 e2e 作业时长（那时跑的故事与 main 相同，还没有新故事），与基线之比就是那次 runner 的快慢。

写 spec 时的数：两个新文件本机 chromium 约 25–30 秒；正常的 runner 约 25.5 分钟；慢的 runner（×1.31，[M6-M8-01](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-01) 记下的比值）约 29–33 分钟，**可能超过 30 分钟的报告线**。T16 以这三个数给出估算的范围，上界与校准点一并写进台账；估算或任一次实测（T10 的检查点、T16 推送后的分支 CI）超过 30 分钟时，T16 先停下报告负责人、不推送；改 CI 的期限或作业结构须负责人同意（M6-M8-06 的教训）。主控已预先告知负责人这种可能（[re-70](../../../reviews/re-70-M6P6b预检.md) §5）。

**写法**（P5b、P6a 阶段记录 §5 的教训）：应用内跳转之后以目标页独有的内容为锚（等目标页自己的列表读答到）；“载入后仍在”由同一人另开的浏览器核对，不在重新载入之后核对控制台（M6-M8-09）；关通知只用共用的 `dismissToasts`（M6-M8-10）；有两道守卫时以嵌套的变异证明故事守着它们，并写明单去一道为何等价；没有能编译又让故事失败的产品变异时，以期望错误的 RED 证实断言起作用。

## 3. 局部裁定

- **R1 先按职责拆开工作项根 store，调用方改经写入单元（主控裁定）**：M6-M8-08 要求 P7 往里加写入之前拆到约 400 行以内，P6a R16 把它提到 P6b 加入模块之前。拆缝取最清楚的一条：写队列、确认的流水线、命令与请求体是一个职责（“发出一个修改，确认后写进每一份”），127 行的 `write()` 是它的核心。选项 B（调用方改经写入单元），不取选项 A（根上留一行的转发）：删掉不用的代码的规则不许兼容层；也不取把读的凭据、列表登记一并拆出的选项 C（要改 M5 的资产 store 与收集箱，超出 P6b 所需）。改调用方的量大，分两个 Task：T2 搬代码、改产品代码的调用方，为让 T2 的提交仍能编译与通过测试，根上暂留测试所用方法的转发；T3 改测试的调用处并删去这些转发（同 P6a R2 的并存只在本分支上，合入的状态里没有）。代价：约 35 个文件只改接收者的改动，M4、M5 的测试随之改调用处（断言不改，以基线收据核对）；`updateIssue` 的 router 退路照搬（§5 的 P7）。
- **R2 只差领域的迭代页列表件改为两域共用（主控裁定）**：照搬 P5b 的迭代件会多出约 12 份近似的副本（约 700 行），不合质量要求；P6a 把读失败、停用空态、分布与进度页签移到共用处是先例。§2.2 的表逐个写明共用或分开及理由；行为真不同的分开（列表状态的规则、加入与移出的请求体、偏好 service 的类型、显示选项常量、进行中卡片的件）。共用的重构单独成 Task、以迭代的测试为基线（先例 P5b T3、P6a T5），之后模块才接上。代价：P5b 的迭代文件改名、改为按域，迭代的测试改导入路径与调用处的名字（断言不改）；迭代一侧的差异变大，审查以“迭代的测试断言不改”为准。
- **R3 页头的列表控件在本页偏好读到之后才出现（第 8 条）**：Plane 的模块页、项目页的页头一开始就给布局、筛选与显示，筛选未到时改动静默不起作用；nerve 的项目页点了只得一个笼统的错误提示（re-67 M8，M4 起如此）。三页用正文的同一个条件。代价：项目页第一次载入时这三个控件比现在晚一点出现；登记差异。
- **R4 重读失败时正文沿用已读到的偏好（第 9 条）**：store 在读失败时本来就留着所持的值，把一个可用的列表因后台的重新校验失败换成读失败，比留着更糟，也让页头与正文不一致（re-67 限定复审 M1）。提示说明重读失败并可重试。代价：别处改了本页偏好时，重试之前列表按旧的偏好显示，提示说明这一点；第一次读失败的样子三页不统一（项目页是 M4 的，§5 的 M8）。
- **R5 移出模块的请求体轮到时由所持的整集合算出**：`module_ids` 整体替换，请求体漏掉已归档的模块就会把工作项移出它（设计 §3.3“PATCH 时保留工作项上已有的隐藏 id”）；调用时算会让连续的两个写中后一个用前一个回答之前的值（v0 §7.7）。Plane 的“移出”是按对删除、不等回答也不接拒绝，nerve 等确认后才离开、提示与重读进度，被拒时说明。代价：接口没有前置条件，读与写之间别处的修改仍可能被盖掉（登记）。
- **R6 本页新建一次发出，只在模块可加入时带本模块，与给定的集合取并**：nerve 的 `IssueCreate.module_ids` 在同一事务里建立归属与“加入”的动态，被拒时整个不建；Plane 先建再加入（两个请求，第二个可能失败），且其“多于一个才保留”的规则会把唯一的另一个模块换成本模块。取并让 P7 的模块字段给出的集合不被本页覆盖。全局新建只在可加入时带本模块（P5b 的同一裁定）。代价：无。
- **R7 修改与复制不覆写模块集合**：Plane 的“修改”把弹窗里的 `module_ids` 换成只有本模块，在 Plane 的增量写法下无害、在 nerve 的整体替换下会移出其他模块；nerve 的表单没有模块字段，修改不带 `module_ids`，测试钉住，P7 加模块下拉框时轮到发出时算。Plane 的“复制一份”在源有两个以上模块时带上它们，nerve 的复制只加入本模块。代价：复制不带源的其他模块，直到 P7 的表单有模块字段；登记差异。
- **R8 模块页偏好里按别的模块筛选：登记差异，不改接口（主控裁定）**：Plane 的模块页把整个富筛选与 `module=<本模块>` 一起发出、由服务端取交，所以可以“在本模块且在 X 模块”；nerve 的模块页偏好的 `filters` 不收 `module_id`（400，P3b R8、设计 §7“偏好里的筛选不含本域的键”，已批准），与迭代页偏好不收 `cycle_id` 同类。改它要改接口与服务端，不在本段（“本段不改 `server/`、`api/`”），不是负责人的决定。T15 更正差异清单的两行：“按迭代、模块分组”一行的“结果只有一组”只对迭代页成立；“迭代、模块页的显示设置”一行 Plane 一栏“本页的筛选键也收下，由网页发请求时去掉”对富筛选不准确（那段去掉的代码不起作用）。代价：模块页上要按别的模块看，只能经项目页的筛选（P7 的词表之后）。
- **R9 已归档模块的列表只读**：照 P5b R5（已结束、已归档迭代只读）与 P6a R12（已归档模块的详情只读）。nerve 允许改其中的工作项、移出（只拒绝加入）；Plane 没有已归档模块的页面。模块的状态不让列表只读（两边同）。代价：要改已归档模块里的工作项，先恢复模块或在项目页改；登记差异。
- **R10 “添加已有”在 P7**：同 P5b R11。设计 §12 把它与模块下拉框放在 P7；nerve 没有批量接口，要逐个 `PATCH`（设计 §15 第 12 条）。M4 交接 M6-02 写“对应列表第一次开放前完成”，字面上包含“添加已有”；本段先开放列表，“分组头添加已有、日历 `addIssuesToView`、`existing-issues-list-modal.tsx` 的模块调用方”依设计 §12 的 P7 一行与 §15 第 12 条在 P7，回执照此直说。
- **R11 空态用上 `module_work_items` 的三个键，`cta_secondary` 留给 P7，已归档另立一键**：设计没有别的说法，模块页的空态照 Plane 用 `project_empty_state.module_work_items.{title,description,cta_primary}`（P6a §5 记下它们自基线起没有读者）。`cta_secondary`（“添加现有工作项”）是“添加已有”的入口，随 P7 读到，同迭代页的 `cycle_work_items.cta_secondary`（P5b 起留给 P7）。已归档模块的空态不能用“创建工作项以开始跟踪”的说明，迭代的 `completed_no_issues` 说的是迭代的转移，另立一个模块的键。代价：一个键到 P7 才有读者。
- **R12 不出现“按模块分组”**：同 P5b R9。网页的词表在 P7 才有模块，现在没有可藏的选项；本段给模块页一份显示选项常量（同项目页、不含模块），测试钉住，P7 加词时模块页仍不出现。
- **R13 写后重读进度**：同 P5b R10。M4 删除了 Plane 的分布乐观更新；本页列表确认一个写之后重读本模块的 `/progress` 一次，只跟进 nerve 确认的写、只在发出时的会话与页面里；列表页的行计数在下次取数时更新（P6a R9）。
- **R14 侧栏的行切换本页筛选，照 Plane 的切换**：Plane 在 `!peekModule && moduleFilter !== undefined` 时切换；nerve 照迭代页侧栏（P5b R8），经本页的筛选实例保存。Plane 的进度一节在日期不合时整个打不开，行也点不到；nerve 的页签总显示（P6a R8），行在模块页上总可点。登记差异。
- **R15 访客看得到模块页的列表**：Plane 的访客打不开模块页（`retrieve` 只给管理员与成员）；nerve 的访客读得到模块（设计 §15 第 11 条）与它的偏好（§3.7），P6a 已让访客看到模块的详情。列表与数目按 nerve 给的（只看自己的访客只算自己的）；没有新建与移出。登记差异。
- **R16 偏好的筛选类型本段不收窄，交 P7**：按页的基类以项目页的偏好类型约束 service，编译器挡不住一个 `module_id` 进模块页的偏好（400）。今天没有路径产生它：网页的筛选词表没有这个词，`toProjectFilters` 丢掉词表外的键（`core/lib/issue-filters.ts`）。收窄要把条件到偏好的转换按页分型，与 P7 给词表加迭代、模块的词是同一件事。代价：P7 之前只靠词表挡住；§5 的 P7 写明。

本段的裁定都是局部决定：都在已批准的设计与 P5b/P6a 的先例之内，不改接口、不改服务端、不跨后端模块。R1、R2、R8 是主控在写 spec 之前的裁定，第 8、9 条（R3、R4）由交接指定三页一并处理。**没有须负责人决定的事。**

## 4. 验收标准

- [x] A1 拆分：写入单元持有写队列、确认的流水线、六个命令与请求体，根与它各在约 400 行以内、之间没有值的循环导入；合入的状态里根上没有这些方法的转发；调用方经写入单元；M4、M5、P5b 的测试只改接收者与名字、断言不改（基线收据同数）；白名单与在途的写各有能让测试失败的变异。
- [x] A2 共用：§2.2 的表逐个落实（按域的差别经 props 或按域的小表，共用件不导入迭代独有的件）；迭代页的测试只改导入路径、构造与调用处的名字，断言不改（基线收据同数）。
- [x] A3 引擎与 store：MODULE 的 store、动作、路由映射、挂载键、本页筛选、空态；模块页偏好的 service 与筛选（按模块 id、不读 router、随一代退役、模块关闭时不读）；白名单放行 `module_ids`；本页新建只在可加入时带本模块并与给定的取并；移出的请求体轮到时算、保留已归档与别的模块；修改的请求体不带 `module_ids`。每个性质有单测，变异能让测试失败。
- [x] A4 页面：四种布局、页头（数目、布局、筛选、显示、新建）、空态、已归档只读、模块关闭的停用、新建（页头、快速新建、分组头）一个请求、移出模块、拖拽、修改与复制、被拒时说明、写后重读进度、侧栏的行切换；第 8、9 条在模块页、迭代页、项目页。
- [x] A5 G15、G19 的模块页列表部分（三种引擎，§2.10）：四种布局、双向续页、本页偏好的读写、模块关闭、已归档只读、换模块与后退前进与换项目时迟到的回答不进新页面、失权的迟到回答、OwnOnly 与工作区管理员兼访客、换会话、跨分组的写后计数、归档范围、附件列（M5-M6-02）、移出保留已归档的模块、整页零非预期失败（M4 M6-01）。
- [x] A6 守卫：`project-address-route`、`.oxlintrc.json` 的 swr 限制（含 `priority-issues.tsx` 与共用的布局根）、告警上限 44、knip、关键词、i18n、“按本页分组”的守卫、`root.store.test.ts` 21 个 store。
- [x] A7 文档：P6a 的回填（阶段记录 §7、spec A8、设计 §14、前端改动清单的 P6a 各行、M6-M8-01 的 C7 再现）；交接回执（M4 交接 M6-01、M6-02 的模块部分，M5 交接 M5-M6-02 的模块部分，M3 交接 §1 的模块 user-properties，M8 交接 M6-M8-08）；差异清单（含两行更正）、前端改动清单、设计 §8/§12/§14。
- [ ] A8 七项本地门禁（`make test` 照负责人 2026-11-11 的规则）、deep-review 修复闭环、准确的分支 CI 与 noff 后的 main CI 全绿、归档清理。（合入后由下一段的 T1 勾选）

## 5. 后续责任

**P7**：“添加已有工作项”（空态的 `cta_secondary`、分组头、日历的 `addIssuesToView`、`existing-issues-list-modal.tsx` 的模块调用方，R10、R11）；模块下拉框与列表词表（工作项详情、行、新建弹窗的模块字段：请求体轮到发出时以所持的整集合算，保留已归档的 id，不照 Plane 把集合换成只有本模块，R5、R7）；模块页的移动端页头；词表加入迭代、模块之后按页收窄偏好的筛选类型，本页的键不进本页的偏好（R16）；`issue.remove.cycle.{loading,failed}` 与 `issue.remove.module.{loading,failed}` 随下拉框用上或一并删除；模块页“复制一份”描述带图片的工作项经内容复制建立，不加入本模块（迭代页同一条路）。往工作项写入加东西时加在写入单元（R1），并给写入单元的 `updateIssue` 明确的范围，去掉没有范围时退回 router 工作区的一支（`root.store.ts:552` 搬来的，T2 起在 `issue-writes.ts:261`，M4 起如此）；去掉之后把写入单元加入 `project-address-route`（本段因这条退路没有加入，re-70 F5）。P7 加入让三页的显示选项常量不再相同的词时，`cycle_issues`、`module_issues` 改为由 `issues` 去掉本页的键推出，不让这两份副本在理由消失之后还留着（re-70 F6；本段照 P5b R9 的先例让 `module_issues` 是 `issues` 的第三份副本）。P7 给新建与修改的表单加模块字段时另钉住“复制一份”的模块集合：今天源的其他模块到不了复制的请求，只因表单的字段（`ISSUE_FORM_FIELDS`）没有模块，命令宿主的 `copyOf` 带上源的 `module_ids` 与新建的请求体带上表单的 `module_ids` 两处一起改的嵌套变异在 T10 等价存活，由一次期望错误的运行证实断言起作用（R7）。规划页测试的共用替身 `issue-layouts/roots/fake-planning-page.tsx` 的模块级 `opened`（`:203-214`，F1 起帮手作用于 `planningPageTab` 最后打开的页）在标签页关闭时不清空，今后一个忘了先打开页的测试会作用于上一个测试已卸下的标签页；今天每个测试都先打开页（`f1-mut-first-tab-kept` 说明这个语义有读者），P7 在替身的 `close` 外包一层或以 `afterEach` 清空它（[re-71](../../../reviews/re-71-M6P6b整体审核.md) §4 的 M1）。

**修复轮（整审之后）**：下面各项在 T16 的整审之后的修复轮里做，不推到别的阶段；整审照此核对。整审 [re-71](../../../reviews/re-71-M6P6b整体审核.md) 把前四项列为 M1–M4，另列后三项；修复轮 F 都已做（各项的处理与提交见 re-71 §3），经限定复审 Ready to merge（re-71 §4）：
- **测试帮手的重复**：`roots/module-row-dialogs.test.tsx` 复制了 `row-dialog-lifetime.test.tsx` 的 `outlivesTheRow`（约 40 行，T8）；`roots/module-create.test.tsx` 与 `cycle-create.test.tsx` 重复约 90 行的替身与帮手（`send`、`projectField`、`quickAddForm`、`headerButton` 等，T9）；`roots/module-quick-actions.test.tsx` 与 `module-create.test.tsx` 重复约 25 行的 `ModalCore`、编辑器等替身（T10）；`roots/preferences-reread.test.tsx` 与页头的测试重复约 25 行的装置（T12）。`vi.mock` 不能共用，帮手移进按域的替身（如 `fake-planning-page.tsx`），迭代的测试只改导入行（回归边界“共用”允许），断言不改。**已做**（F1，`5b1046b7`；迭代测试里多改的导入行——`cycle-create` 的两处 `vi.mock` 工厂、`cycle-quick-actions` 与迭代页头测试——主控接受）：帮手移进 `fake-planning-page.tsx`，作用于 `planningPageTab` 最后打开的页，调用处不改；弹窗与编辑器的替身移进 `issue-modal/fake-dialog.tsx`，各文件的 `vi.mock` 导入它；`preferences-reread.test.tsx` 与页头测试共有的约 25 行是 `vi.mock` 的调用，各留在本文件，共用的只有 `inHeader`；18 个文件 201 例同名同数。
- **列表状态不核对实体属于路由的项目**：`moduleListState` 与 `cycleListState`（P5b 起）只看实体在不在、是否归档（迭代另看是否已结束），不看它的 `project_id`。路由里的模块或迭代属于别的项目时，页面说“这里没有这个模块（迭代）”，页头却仍可能给“新建工作项”，全局的“新建工作项”经按域的表也会带上它（nerve 以 422 `module_ids[i]`、`cycle_id` 拒绝）。两域一起改为：项目在、而实体的 `project_id` 不是路由的项目时，同“页面还没有这个实体”，只读、不可加入；迭代的行为在这一边缘处改变，迭代的测试只增不改（T9 留意、主控裁定）。**已做**（F2，`11b9ea74`）：两个纯函数；新测试 8 例（纯函数、页头、全局新建，两域各有），变异“不看项目”两域各三例失败。页头的 `preferencesShown` 与数目、迭代页头的面包屑与 `CycleQuickActions` 同样不核对实体的项目，修复轮中发现，主控裁定现在修。**已做**（F9，`09aba5f8`）：`core/hooks/planning-page-entity.ts` 的 `pageEntity` 是“本页持有”的唯一说法，两个列表状态的 hook 另给 `held`（已归档、已结束的照样算持有），页头的数目与偏好的控件要 `listed && held`；迭代页头照模块页头只在迭代属路由的项目时给面包屑标题与快捷操作；两个页头测试各加一例，能编译的变异五个。
- **测试文件名**：`core/store/issue/cycle/filter.store.test.ts` 旁边已没有同名的源文件（T4 删去 `cycle/filter.store.ts`），它测的是 `PagePreferencesFilter` 以 CYCLE 构造的用法，考虑按所测之物改名（模块的同类测试叫 `module/page-preferences.test.ts`）。**已做**（F3，`9b67b78c`）：`git mv` 为 `cycle/page-preferences.test.ts`，内容不改。
- **两个相近的形状**：`cycle/priority-issues.store.ts` 留了文件内的 `CyclePage`（`{projectID, cycleID}`），与两域共用的 `PlanningPage`（`{projectID, entityID}`）相近（T4 的裁定：进行中卡片的优先级页签迭代独有，共用的形状不收它）；整审决定合并或写明保留的理由。**已做**（F4，`238a29d2`）：整审建议保留，加一行注释写明理由（优先级列表迭代独有，`entityID` 只会让它更含糊）。
- **`CycleListState` 的导出**（re-71 M5）：文件外没有读者。**已做**（F5，与 F2 同一文件，并入 `11b9ea74`）：去掉导出。
- **移出被拒的状态码**（re-71 M6）：`core/store/issue/module/issue.store.test.ts` 以 400 答 `planning.modules_disabled`，nerve 答 409。**已做**（F6，`36e23061`）：改为 409，断言不变。
- **swr 的限制**（re-71 的观察，主控裁定现在收紧）：`.oxlintrc.json` 的 swr 表缺六个既有的 `useSessionSWR` 读者。**已做**（F7，`d302141f`）：加入五个，第六个（`workspace/settings/use-members-settings-fetch.ts`）已由表中的 `core/components/workspace/settings/**` 覆盖（主控接受只加五个）；告警上限仍为 44。

**M8**：项目页第一次读偏好失败时只有 `role="alert"` 的文字、没有重试，与迭代页、模块页的可重试的读失败不同（M4 起如此，R4 没有改它）；随最终矩阵统一或写明理由，交接为 [M6-M8-12](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-12)。[M6-M8-08](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-08) 写下本段的拆法、行数与验收的结论（条件已满足，留到 M6 收尾核对）；[M6-M8-07](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-07) 再现一次（T10 检查点本机 e2e 的 F11 (page) [local]，只重跑该文件即通过）；[M6-M8-01](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-01) 的余量复核多一个数据（T10 检查点的分支 CI 43517662603 的 e2e 作业 25 分 38 秒，跑的故事与 main 相同）。

## 6. 风险

| 风险 | 处理 |
|---|---|
| 拆分碰坏 M4 的写入路径 | 只搬不改；T2 第一步对受影响的测试记基线收据，T2、T3 之后同数；白名单（`cycle_id`、`module_ids`）与在途的写（`pending`）的变异 |
| 共用的重构碰坏迭代页 | 单独的 Task（T4、T7）先跑迭代的测试留基线，断言不改；T9–T11 的共用部分同样先留基线 |
| 整体替换误移出模块 | 请求体轮到时由所持的整集合算；单测以两个在途的写、已归档的模块钉住，变异“调用时算”“去掉已归档的”使它失败；e2e 以数据库的归属核对 |
| 项目里的每个写都送到模块页的绑定 | 与项目页、迭代页同；成员判断的 `module_id` 已在 P5b 有单测与变异 |
| e2e 时长在慢的 runner 上可能超过报告线 | 基线 25 分 00 秒（P6a 的 main CI attempt 2）；T13、T14 记下每个新文件在各引擎的时长，T16 推送前算估算、上界与 T10 的校准点（§2.10）；估出或实测超过 30 分钟先报告负责人、不推送，改期限或作业结构须负责人同意 |
| Task 的量 | 拆分分两个 Task；MODULE 的联合分散到用它的 Task；T7 超出一个实现者的量时把空态移到 T8 |
