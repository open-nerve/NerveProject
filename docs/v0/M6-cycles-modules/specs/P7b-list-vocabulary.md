# M6 / P7b：工作项一侧（二）——列表词表、目录与“添加已有”

> 状态：spec 与 plan 已写（2026-11-14），预检通过（修正后，[re-74](../../../reviews/re-74-M6P7b预检.md)），待实施。基线 main `f42f9d6457ccc9540a687019f3530e253fe0f7b5`（P7a 的 noff，双父 `7868a6b4`、`5d6dab04`，树与 `5d6dab04` 相同；P7a 的最终分支 CI 43600392398 全绿，e2e 作业 23 分 48 秒；合入后的 main CI 43603772626 attempt 1 全绿，e2e 作业 132478430823 为 14:31:09Z–14:54:52Z，23 分 43 秒；P7a 的归档在主目录的 `.superpowers/sdd/M6-P7a-integration-20261114/`，含 `cleanup.log`；这些由本段 T1 以归档与公开的 jobs 接口核对后回填进 P7a 的记录）。设计见 [M6 设计](../M6-design.md) §2 的 G4、G11、G15，§3.3、§3.6、§7、§8 的“工作项一侧”、§12 的 P7b 一行、§13–§15；P7 的拆分见 [P7a spec](P7a-work-item-planning.md) §1、R1，交给本段的事见它的 §5 与 P7a 的[阶段记录](../reviews/P7a-work-item-planning-review.md) §3、§5；主控在写 spec 之前的裁定 E1–E15 见本阶段台账 `controller-rulings.md`（本段 R1–R15 照录）。网页的长期规则见[总体设计 §7.2、§7.7](../../v0-design.md#77-stores-按会话分代长期有效m2-起)。接口合同见 [P3b spec](P3b-list-engine.md)（列表的筛选、分组、排序与迭代页、模块页的偏好）与 [P3a spec](P3a-issue-membership.md)（工作项的 `cycle_id`、`module_ids`）。现状调研见本阶段台账 `research/web-delta.md`（P7a 合入之后的增量：P7a 造好的接缝、目录、词表、“添加已有”、e2e 的余量、Task 的估算）与 P7 的两份 `research/web.md`、`research/plane-contract.md`（写于 P7a 之前的 `c0572192`，Plane 的行为与 nerve 的合同；本 spec 引用的行号都在本段的树上重核过）（ignored，随阶段归档）。负责人 2026-11-11 授权连续推进至 M6 收官，重要高风险决策仍停下确认；本段不改 `server/`、`api/`（E14），本机 `make test` 若因 [M6-M8-05](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-05) 的签名失败，以分支 CI 的服务端两路为准（负责人 2026-11-11）。

文中的路径：`core/…` 在 `web/apps/web/` 下；`issue-layouts/…` 是 `core/components/issues/issue-layouts/…`；`@nerve/…` 的包在 `web/packages/` 下。

## 1. 目标与边界

P7a 交付了工作项自己的迭代与模块（写入单元的 `changeIssue` 与 `IssueChange`、按项目的规划目录 `core/lib/planning-directory.ts` 与 `core/hooks/use-planning-{names,options}.ts`、两个下拉框、各处的显示、弹窗与草稿）；列表这一侧还停在 P3b：网页的筛选、分组、排序词表里没有迭代与模块（`core/lib/issue-filters.ts:6-67`），列表的目录是空的 `Map`（`core/store/issue/helpers/issue-list-binding.ts:84-86`，按名字排序时大声失败），迭代页、模块页没有“添加已有”，日历在规划页上只改日子、不加入本页（`research/web-delta.md` §0）。本段（E1）：

- **回填与阶段**：P7a 的回填（阶段记录 §7、spec A12、设计 §14、[M6-M8-01](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-01) 的数据点），并以 P7a 实测的 CI 重算 e2e 的余量（E13，§2.14）；关键词 phase 仍是 `M6/P7`（三段共用）。
- **先拆**：`issue-list-binding.ts`（487 行）在加目录之前按职责拆为读、目录与绑定三件（R16，§2.1）。
- **目录**（A7）：列表按迭代名、模块名排序时，从 P7a 的规划 store 叫出名字（含已归档的），读法与键同 P7a，不另开读的路径（R17，§2.2）。
- **词表**（A6）：筛选 `cycle_id`、`module_id`（含“无”），分组与子分组 `cycle`、`module`，排序 `±cycle_name`、`±module_name`；偏好的筛选按页收窄（A9，P6b R16，§2.3）；迭代页、模块页的显示选项常量由项目页的去掉本页的词推出（A8，§2.5）。
- **界面**：分组的列与泳道（含已归档的组、“无”在最后、拒收的列，§2.4）；开关关闭时的选项（§2.5）；分组里的新建带上组的值（§2.5）；筛选的界面（§2.6）；拖到迭代、模块的组（§2.7）；表格两列表头的排序（P7a R21，§2.8）。
- **“添加已有”**（A4、A5）：空态的 `cta_secondary`、分组头、日历三个入口，弹窗都在列表的命令宿主；逐个的 `PATCH` 经 P7a 的 `changeIssue`，至多 2 个在途，一批一次进度重读（§2.9、§2.10）。
- **验收**：G15 的工作项一侧的页面、G4 与 G11 的“添加已有”的页面（§2.14）；[M4 交接 M6-02](../handoffs/M4-closeout.md#M6-02) 的回执（含“何时验收”的逐项）；[M3 交接](../handoffs/M3-closeout.md) §1、§3 中 `existing-issues-list-modal.tsx` 的各项。

**不做**（交给后来的，详见 §5）：
- **P7c**：分组时页头的数目（A19，“何时验收”的“跨分组写后计数”在本段只核组的数目与 nerve 的总数，见 §2.14）；迭代页、模块页与项目页的移动端页头（A15、X2；项目页的移动端页头的显示面板读同一份 `issues` 常量，本段在组件一级的开关门由它继承，§2.5）；命令面板（A16）、A17、A18、A20、A21、设计 §8 的四个视图文件（X1）、G19 的工作项一侧。
- **P8**：归档页（`archived_issues`）的迭代、模块词表（E10，R10）；[M6-M8-13](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-13)；新的筛选与分组控件的焦点轮廓随最终矩阵核对。
- **M7**：工作区级的迭代、模块列表与跨项目的目录（视图）；成员列表（个人页）的名字排序仍按项目读（§2.2）。
- 任何接口或服务端的变化（E14）：搜索没有按迭代、模块排除的参数、没有批量写，本段照现有合同做（§6，R2、R6）。

## 2. 交付合同

### 2.1 先拆 `issue-list-binding.ts`（R16）

**现状**：`IssueListBinding`（`core/store/issue/helpers/issue-list-binding.ts`，487 行）一个类里有范围的类型（`:26-42`）、生命周期与权限（`:47-124`）、页的读与范围的核对（`read()`，`:125-225`）、目录（`context.catalog` `:75-89` 与 `prepareCatalog` `:226-289`）、取数与续页（`:290-417`）、确认与核验（`:419-487`）。`prepareCatalog` 的调用者：页的读（`:181`）与写的两处确认（`core/store/issue/issue-writes.ts:117`、`:168`）。

**做法**：只搬不改，三件（名字由实现定，例如）：
- `issue-list-read.ts`：页的读与范围的核对（约 110 行）；
- `issue-list-catalog.ts`：目录（`context.catalog` 的目录部分与 `prepareCatalog`，约 80 行，§2.2 之后约 130 行）；
- `issue-list-binding.ts`：生命周期、取数、确认与核验（约 300 行）。
- 绑定对外的成员（`prepareCatalog`、`context`、`read` 的调用路径）不变，写入单元的两处调用不改；测试只改导入；以 `git diff -M`（`--color-moved`）核对是搬移；先留基线收据，拆后逐文件同名同数（P7a T4 的写法）。三件各在约 400 行以内（只拆出目录会剩约 410 行，`research/web-delta.md` §2.2）。三件都不读路由，随拆分加入 `project-address-route`（带样例）。

### 2.2 目录（A7，R17）

**现状**：`context.catalog` 的状态、标签、成员取自根的三张表（`issue-list-binding.ts:80-83`，根的 getter `core/store/issue/root.store.ts:71-79`），`cycles`、`modules` 是空的 `Map`（`:84-86`）；`prepareCatalog` 遇到 `±cycle_name`、`±module_name`（`:238-239`）什么都不读，行上有值就抛“Required issue directory is unavailable”（`:278-288`）；`IssueCatalog` 要求两项含已归档的（`core/lib/issue-order.ts:5-12`），比较器只用 `.get`（`:86`、`:88`），`prepareCatalog` 只用 `.has`（`issue-list-binding.ts:284-285`）。钉住它的是 `helpers/base-issues.store.test.ts:453-474` 的 `planningDirectoryIsUnavailableUntilP7` 与 `:476` 起的 `workItemInNoCycleOrModuleNeedsNoDirectory`。

**做法**：
- **查找取自规划 store**：`IssueCatalog.cycles`、`modules` 收窄为只读的查找（`Pick<ReadonlyMap<string, Cycle>, "get" | "has">`，模块同），目录的两项以 `planning.cycles.getCycleById`、`planning.modules.getModuleById` 作答（`core/store/planning/cycle.store.ts:74-83`：两份列表与单读里 nerve 最后回答的一份，只给项目 store 仍给的项目的；`module.store.ts` 同形）。规则只在 store 里一处，根不加 getter（`root.store.ts` 仍 320 行，[M6-M8-08](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-08)）；测试里的 `new Map()` 字面量照旧满足收窄后的类型（`issue-order.test.ts:38-39`、`:66`、`:70`，`paged-groups.test.ts:12`，`fake-issues.ts:110`），不改。
- **何时读、读什么**（re-74 F2、F8）：只在列表按 `±cycle_name`（`±module_name`）排序时，照状态、标签的先例（`:242-264`）。对行所在的每个项目：
  1. 项目不在项目 store 里时先读它（`fetchProject`，`:245-246` 的先例；`getCycleById` 只给项目 store 仍给的项目的迭代，不先读项目，成员列表上别的项目的行就永远叫不出）；
  2. 行上有叫不出名字的迭代时读该项目未归档的列表（`fetchCycles`），**store 已有它也读**（同状态、标签：叫不出就重读，`:249-260`）——别处新建的迭代在两份列表都已在 store 之后才被行加入，这一读叫出它；
  3. 之后仍有叫不出的，读已归档的列表（`fetchArchivedCycles`），同样已有也读；
  4. 两份都读了仍叫不出时大声失败（见下）。

  模块同（`fetchModules`、`fetchArchivedModules`）。每个项目每次 `prepareCatalog` 各至多一次，都叫得出时什么都不读。“先未归档、后已归档”的次序与 `usePlanningNames`（`core/hooks/use-planning-names.ts:59-61`）是同一条，抽为 `core/lib/planning-directory.ts` 的一个纯函数（名字由实现定），hook 与目录都调它；纯函数只定次序，“已持则不读”的守卫只在 hook 里（hook 随渲染反复求值，对其间删掉的值不能无尽地重读；目录是一次列表的读，每次至多各读一次）。读经 store 的读法（与包装层、归档页、`usePlanningNames` 的 SWR 键背后是同一组函数），值进同一份 `Reconciled`：没有第二条读的路径。读的每一步前后照旧 `assertActive`（`:244-262` 的写法）。
- **被超过的读**（re-74 F9）：行的 hook 与目录同时读同一份列表时，被较新的读超过的那一次 `Reconciled.fetch` 回答 `undefined`、什么都不写（`core/lib/reconciled.ts:52-56`），由较新的那一次决定 store 的值；目录这一次若因此仍叫不出就失败：列表的读照旧失败、可重试，写的确认照旧把那份列表 `markUnavailable`（`issue-writes.ts:178`，绑定 `:468`），之后的重读叫出。这是状态、标签今天已有的行为（同一份 `Reconciled`），本段不加机制。
- **门**（R17）：目录随列表自己的读，以绑定的范围为门（同状态、标签、成员），**不看项目的开关**：nerve 的列表词表与两份列表的读都不看开关（`server/internal/modules/issue/domain/listvocab.go` 里没有开关；`api/modules/planning.yaml:15-27`“whether its cycles are turned on or off and whether it is archived”），存下的 `order_by: cycle_name` 在开关关闭之后照旧由 nerve 回答（E8），目录若看开关，这样的列表就整个失败。界面上的新读者都以开关为门（§2.12）。
- **仍叫不出**：两份列表都重读了、仍有叫不出的值（其间被删除）时照旧大声失败（P3b R9 的理由：比较器缺名字会把确认的行放错位置），不改为“无名”。
- **成员列表**（个人页）：nerve 允许按两个名字排序（P3b R3），`prepareCatalog` 按行的项目逐个读（项目不在 store 时先读项目），已经覆盖；网页不给控件（§2.3）。
- **新鲜**：别处新建的迭代（模块）被行加入之后，列表的读与写的确认（`issue-writes.ts:117`、`:168`）为叫不出的 id 调 `prepareCatalog` 时重读未归档的列表而叫出；别处的改名在 store 下次重读之前是旧的名字（只在叫不出时重读），同状态、标签。
- **测试**：`planningDirectoryIsUnavailableUntilP7` 改写为本段的合同（已归档的名字读到、两份都在 store 里而行带别处新建的迭代时重读叫出、两份都重读了仍叫不出时大声失败、不按名字排序的列表什么都不读、开关关闭时照读、成员列表先读不在 store 的项目）；`workItemInNoCycleOrModuleNeedsNoDirectory` 留着；共用夹具的 `ordersAllSharedFixtureScenarios` 照旧覆盖比较器。

### 2.3 筛选的词与偏好的按页收窄（A6、A9，R7、R18、R22）

**现状**：`issue-filters.ts` 的 `fields`（`:6-17`）、`validValue`（`:93-113`，默认支 `isId`，“无”会被拒）、`memberListQuery` 的分组排除（`:269`）都没有迭代与模块；`WORK_ITEM_FILTER_PROPERTY_KEYS`（`@nerve/types` 的 `view-props.ts:67-81`）没有两个键。偏好的基类以项目页的类型为参数：`PagePreferences = Pick<ProjectPreferences, …>`（`page-preferences-filter.store.ts:21-22`），service 的 `update` 是方法写法（`:25-28`），`updateFilterExpression` 对每一页都以 `toProjectFilters` 转换（`:222-223`）；迭代页、模块页的基类直接构造（`root.store.ts:196`、`:200-204`），service 收生成的类型（`core/services/planning/cycle-preferences.service.ts:22` 的 `CyclePreferencesUpdate`）。生成的 `CycleIssueFilters` 没有 `cycle_id`、`ModuleIssueFilters` 没有 `module_id`（`web/packages/api-client/src/schema.gen.ts:4152-4165`、`:4179-4192`），但它们只是 `ProjectIssueFilters`（`:3493-3507`，属性都可省）的**结构子集**：一个类型为 `ProjectIssueFilters` 的值（不是对象字面量，没有多余属性的检查）不论参数的变型都赋得进 `CycleIssueFilters`，`toProjectFilters` 照样能当迭代页的转换通过类型检查——缺一个属性在类型上不是“不许有”（re-74 F1；问题在结构子类型，不只在方法参数的双变）。**今天只有词表挡着迭代页发出 `cycle_id`**（`research/web-delta.md` §3.4）：`fields` 一加上它就没有了。

**做法**：
- **词**：`fields` 加 `cycle_id`、`module_id`，`validValue` 对二者照 `label_id` 收 `none` 与 id；`WORK_ITEM_FILTER_PROPERTY_KEYS` 加两个键；`memberListQuery` 的分组排除加 `cycle_id`、`module_id`（成员列表 422，`e2e/stories/planning/g15-issue-lists.spec.ts:27`）；`cleanMemberFields`（`:134-138`）已不放它们的筛选，不改。
- **按页收窄**（R18，re-74 F1）：
  - **本页的类型**：网页一侧由生成的类型推出每页的筛选类型（`issue-filters.ts`，名字由实现定）：迭代页 `Omit<ProjectIssueFilters, "cycle_id"> & { cycle_id?: never }`，模块页 `Omit<ProjectIssueFilters, "module_id"> & { module_id?: never }`，项目页仍是 `ProjectIssueFilters`。`never` 是要害：`ProjectIssueFilters` 的 `cycle_id?: string[]` 赋不进 `cycle_id?: never`，于是任何带本页的键的值（字面量或不是）都赋不进本页的类型。
  - **基类与转换**：基类以筛选的类型 `F` 为参数，每页一个转换 `(expression) => F`（项目页 `toProjectFilters`；迭代页、模块页去掉本页的键并以本页的类型回答）；`toProjectFilters` 回答 `ProjectIssueFilters`，当不了迭代页的转换。
  - **service**：`update` 写成属性（`strictFunctionTypes` 下的严格参数变型），以 `F` 定类型；迭代页、模块页的 service 的 `update` 收本页的类型（由 `CyclePreferencesUpdate`、`ModulePreferencesUpdate` 换掉 `filters` 一项推出）。于是项目页的基类配不上迭代页的 service（`ProjectIssueFilters` 赋不进本页的类型）；方法写法的双变会放过它。
  - 根上构造时给出每页的 service 与转换，`root.store.ts` 的行数不增或只增一两行。
  - **类型测试**（`@ts-expect-error`，迭代页、模块页各一组）：迭代页的基类以 `toProjectFilters` 为转换（转换配错）；项目页的基类配迭代页的 service（service 配错）；一个类型为 `ProjectIssueFilters` 的变量（不是字面量）赋给迭代页的筛选类型。变异的收据：去掉 `never`（只留 `Omit`）让三处都成为未用的指令、`tsc` 报错；`update` 改回方法写法让“service 配错”一处成为未用的指令。
  - **运行时**：迭代页的筛选修改即使表达式里有 `cycle_id` 也只发 `module_id` 等（单测，变异“转换留着本页的键”须失败）。
- **本地存的两页**（个人页、归档页）的本地词表（`helpers/issue-filter-helper.store.ts:30-64`）不加：个人页的分组、筛选没有迭代与模块（nerve 422），排序没有控件；归档页照 R10（R22）。

### 2.4 分组的词、列与泳道（A6，R4、R19）

**现状**：`TIssueGroupByOptions`（`view-props.ts:15-24`）、`GroupByColumnTypes`（`@nerve/types` 的 `issues.ts:20-27`）、`issue-filters.ts` 的 `groups`（`:33-42`）、基类的 `groupWire`、`groupValues`（`page-preferences-filter.store.ts:43-66`）、`ISSUE_GROUP_BY_OPTIONS`（`@nerve/constants` 的 `issue/common.ts:53-65`）都没有迭代与模块。`getGroupByColumns`（`issue-layouts/group-columns.tsx:19-49`）是同步的投影，“读目录归宿主”（`:18`）；`labelColumns` 只在工作区一级并入回答的组（`:127`）；泳道调它时不给回答的组（`issue-layouts/kanban/swimlanes.tsx:280-292`）。`IGroupByColumn`（`issues.ts:29-34`）没有拒收的字段。nerve 只回答非空的组，含已归档的迭代、模块的组，`none` 在最后、其余按 id 的文字（`list_issue_groups.sql`，plane-contract §7.5），组头不带名字。

**做法**：
- **词**：`TIssueGroupByOptions`、`GroupByColumnTypes` 加 `cycle`、`module`；`groups` 与 `groupWire`、`groupValues` 加 `cycle → cycle_id`、`module → module_id`（`groupedIssueField` 已把 `module_id` 映到 `module_ids`，`issue-filters.ts:279-292`）；`ISSUE_GROUP_BY_OPTIONS` 加两项，标题与显示属性同名：`common.cycle`、`common.modules`（R19）。词与列在同一个 Task 加入：存下的 `group_by: cycle_id` 一被显示词认出就要有列（否则 `getGroupByColumns` 找不到列）。
- **列的次序**（R4）：迭代按 store 里项目未归档的迭代、nerve 的次序（同下拉框的次序，含已结束的），然后是回答里有的已归档迭代（按项目的已归档列表的次序；叫出名字之前按回答的次序，名字为“—”，叫出之后是名字与“已归档”的说明 `common.archived`），回答里有、两份列表里都没有的（其间被删除）以“—”在它们之后，“无”在最后；网页定次序，不照 nerve 的 id 文字。模块同（没有状态的图标）。图标：迭代按状态 `CycleGroupIcon`，模块 `ModuleStatusIcon`（`@nerve/propel/icons`）。payload：迭代 `{cycle_id}`，模块 `{module_ids: [id]}`，“无”`{}`。
- **拒收的列**（R4）：已结束的迭代（`status === "completed"`）、已归档的迭代与模块，以及项目关着该域时的每个有值的列，不能加入：列上带拒收的原因（一个字段，取值是 nerve 拒绝时的问题码 `planning.cycle_completed`、`planning.archived`、`planning.cycles_disabled`、`planning.modules_disabled`，名字由实现定）。这一个字段同时决定：放下时不发请求（§2.7）、不给“+”、快速新建与“添加已有”（§2.5、§2.9）；没有“拒收放下却给新建”的组合可写。说明经 `errorMessageKey` 读 nerve 拒绝的同一句（`core/lib/error-messages.ts:87-95`），不加新键。“无”列不拒收（离开不检查，`api/modules/issue.yaml:670-688`；开关关闭时离开被 nerve 以 409 拒绝，说 nerve 的原因）。
- **名字**（P7a 的约束）：列表、看板、泳道的宿主（`issue-layouts/list/default.tsx:88-95`、`kanban/default.tsx:100-107`、`kanban/swimlanes.tsx:280-292`）在按迭代、模块分组或子分组时以组的 id 调 `usePlanningNames`（不分组时以空的 id 调，hook 不条件地调用）：已归档的组的名字由它按需读（一个项目一次，开关关着时不读）。`getGroupByColumns` 仍是同步的投影，只读 store。
- **泳道**：两层都并入回答的组（分组的 id 与各组的子分组的 id 之并），已归档的迭代、模块的泳道不丢。
- 多模块的工作项照 nerve 在它的每个模块的组里各一次（成员判断 `core/lib/issue-membership.ts` 已认得，`groupKeys` 的 `module_id` 一支）。
- 分组时页头的数目（A19）在 P7c。

### 2.5 按页的显示选项常量、开关与分组里的新建（A8，R8、R12）

**现状**：`ISSUE_DISPLAY_FILTERS_BY_PAGE.issues`（`@nerve/constants` 的 `issue/filter.ts:126-182`）的筛选、分组、子分组没有迭代与模块；`cycle_issues`（`:184-243`）、`module_issues`（`:245-304`）是它的两份逐字的副本（注释 `:183`、`:241-244` 说 P7 由 `issues` 推出）。分组面板（`issue-layouts/filters/header/display-filters/group-by.tsx:31`，`sub-group-by.tsx:41`）不看开关；显示属性的开关门是 `core/lib/planning-switch.ts:27-43` 的私有表与 `displayPropertyOn`（读者：`display-properties.tsx:46`、`spreadsheet/spreadsheet-view.tsx:66`）。快速新建的预填（`list/list-group.tsx:139-161`、`kanban/kanban-group.tsx:146-185`，两份）对没有列出的分组退回 `{[groupByKey]: value}`，按迭代、模块分组时给出 `{cycle: id}`、`{module: id}`；快速新建的根（`issue-layouts/quick-add/root.tsx`）的预填类型 `QuickAddPrefill`（`:34-36`，七个字段：`state_id`、`priority`、`assignee_ids`、`label_ids`、`start_date`、`target_date`、`parent_id`）与提交时的白名单（`:179-180`）都没有 `cycle_id`、`module_ids`，快速新建因此不加入组的迭代、模块（`research/web-delta.md` §3.5）。`createBody` 本身放行两者（`core/store/issue/issue-write-fields.ts:17-18`，P7a），不是门（re-74 F3）。

**做法**：
- **常量**（A8）：`issues` 的筛选加 `cycle_id`、`module_id`，列表与看板的分组、看板的子分组加 `cycle`、`module`（在 `null` 之前）；`cycle_issues` 由 `issues` 去掉 `cycle_id` 与 `cycle` 推出，`module_issues` 去掉 `module_id` 与 `module`（一个小函数，两份副本删去，`filter.ts` 约 389 → 280 行）。本页的键从不在本页给出（P5b R9、P6b R12，nerve 拒绝 400，plane-contract §7.6）：`@nerve/constants` 的 `filter.test.ts:10-26` 扩到迭代页；两页页头的测试（`[projectId]/cycles/(detail)/header.test.tsx:273` “never by a cycle (R9)”、`modules/(detail)/header.test.tsx:265` “never by a module (R12)”）不改，从此 `issues` 里有这两项，它们是真的回归保护（推出时不去掉本页的键的变异须让它们失败）。
- **开关的词表**（R8）：`planning-switch.ts` 的私有表扩为“规划的词 → 域”一张：显示属性 `cycle`、`modules`，分组 `cycle`、`module`，筛选 `cycle_id`、`module_id`；一个谓词（名字由实现定）回答“这个词在这个项目上给不给”，显示面板、表格的列、分组面板与筛选（§2.6）都调它，`displayPropertyOn` 由它取代（两个读者改调它，不留别名）。
- **开关关闭时**（R8，E8）：项目的页上（路由有项目）只在路由的项目打开该域时给出迭代、模块的分组与子分组；存下的已选的那一项照旧显示并选中，可改掉（不悄悄改存下的查询，nerve 照旧回答；§2.4 的列此时不读名字、有值的列拒收）。个人页（没有路由的项目）照给出的选项（它的选项里本就没有两项）。项目页的移动端页头读同一个面板件，随之继承（P7c 只管它自己的页头）。
- **分组里的新建**（R12，E12）：快速新建的预填改为一个共用的函数：项目的默认状态并上列的 payload（看板的格子再并上泳道的 payload），两份 `prePopulateQuickAddData` 删去；`quick-add/root.tsx` 的 `QuickAddPrefill` 与提交的白名单各加 `cycle_id`、`module_ids`（预填与白名单是先后两道门，一个“迭代的列上快速新建带上 `cycle_id`”的测试同时钉住两者：去掉任一道请求体都没有 `cycle_id`，故“根不传两键”与“预填不带列的 payload”是落在同一个测试上的等价变异，记后者一个即可，台账写明）；按状态、优先级、标签、负责人、建立者分组时得到与今天相同的值（快速新建的既有测试一行不改，基线同数），按迭代、模块分组时是 `{cycle_id}`、`{module_ids: [组]}`，“无”什么都不带。规划页的快速新建照旧经本页的注入（P7a R2）：迭代页按模块分组时请求体有组的模块与本迭代，模块页按迭代分组时有组的迭代与本模块并上的集合。拒收的列（§2.4）不给“+”（分组头）与快速新建。

### 2.6 筛选的界面（A6，R7、R8）

**现状**：`@nerve/utils` 的筛选工厂（`web/packages/utils/src/work-item-filters/configs/filters/`）没有迭代、模块（Plane 有 `cycle.ts`、`module.ts`，各 64 行，`plane/packages/utils/src/work-item-filters/configs/filters/`）；`useWorkItemFiltersConfig`（`core/hooks/work-item-filters/use-work-item-filters-config.tsx`，353 行）建 13 个配置（`:337-351`）；项目一级的 HOC 给 `labelIds`、`memberIds`、`stateIds`（`core/components/work-item-filters/filters-hoc/project-level.tsx:202-204`，项目页、迭代页、模块页与归档页的根都用它）。已选的值不在选项里时显示“不可用的选项”（`core/components/rich-filters/filter-value-input/select/selected-options-display.tsx:26-33`）；已应用的条件按属性取配置、不看 `isEnabled`（`rich-filters/filter-item/root.tsx:44`），“加筛选”只列启用的配置（`web/packages/shared-state/src/store/rich-filters/config-manager.ts:110-113`、`:182-184`）。

**做法**：
- **工厂**：`@nerve/utils` 加迭代、模块两个筛选工厂（照 Plane 的两个文件与本仓的 `label.ts` 的写法，类型来自 `@nerve/api-client` 的 `Cycle`、`Module`），运算符 `in`，单值时 `exact`，选项图标按状态。
- **选项**（R7）：迭代是项目未归档的迭代（含已结束的：筛选可以指名任何未归档的，不取下拉框的“不列已结束”），nerve 的次序；模块是项目未归档的模块；两者都另有“无”一项（G15“含无”，Plane 没有，登记差异），文字读已有的 `cycle.no_cycle`、`module.no_module`。已存的条件里的已归档的值（以及当前开着的条件里的）列为已选的选项，以名字与“已归档”的说明显示（经 `usePlanningNames` 叫出），可以去掉；不另给已归档的作新选项。
- **读**：两份未归档的列表经 P7a 的 `usePlanningListReads`（`core/hooks/use-planning-names.ts:99-106`，与包装层同一份键，开关关着时不读）；已归档的已持值经 `usePlanningNames`。两个配置放进一个新的小 hook（`useWorkItemFiltersConfig` 已 353 行，加两个配置会到约 400，E15），它组合两者；HOC 给出项目。
- **开关**（R8）：配置的 `isEnabled` 除了本页允许之外还要路由的项目打开该域（§2.5 的谓词）且列表已读到；关着时“加筛选”不列它，已应用的条件仍以条件显示（值叫不出时是“不可用的选项”）、可去掉，列表照 nerve 的回答。
- **本页的键**：迭代页不给迭代的筛选、模块页不给模块的（常量，§2.5）。归档页不加（R10）。
- **文案**：筛选的字段 `common.filter_options.fields.cycle_id`、`module_id` 两个新键（en 与 zh-CN 同一提交）。

### 2.7 拖拽（A6，R3、R4）

**现状**：`DRAG_ALLOWED_GROUPS`（`common.ts:45`）门住列表（`list/list-group.tsx:211`）、看板（`kanban/kanban-group.tsx:218-222`）与看板 store（`core/store/issue/issue_kanban_view.store.ts:46-50`），没有迭代、模块。`handleGroupDragDrop`（`issue-layouts/group-drop.ts:100-158`）在轮到时由所持的工作项算请求体（`:129-155`），标签、负责人拖到“无”清空集合（`:142-147`），别的分组抛“Unsupported drop group”（`:148-149`）；它与 `dropIssue` 的函数给的是 `IssueUpdate`（`group-drop.ts:105-110`，`core/store/issue/helpers/base-issues.store.ts:74-80`，动作的类型 `core/hooks/use-issues-actions.tsx:40-45`、`use-planning-page-issue-actions.ts:91-97`），`module_ids` 照样写得进去。拖拽的拒绝经 `useRefusalToast` 说 nerve 的原因（`core/hooks/use-group-dragndrop.ts:85-87`）。

**做法**：
- `DRAG_ALLOWED_GROUPS` 加 `cycle`、`module`。
- **请求体在写的轮到时算**（R3，E3）：`handleGroupDragDrop` 与 `dropIssue` 收的函数改为给 `IssueChange`（P7a 的 `@nerve/types` 的 `issues/issue-change.ts`，`module_ids?: never`）；`dropIssue` 在写队列轮到时以 `changeBody(change, held.module_ids)`（`issue-write-fields.ts`）得出请求体，再照旧设 `sort_order`（M4 起的规则，`base-issues.store.ts:81-143`）。于是：
  - 迭代：拖到别的迭代的组 `{cycle_id: 目标}`，拖到“无”`{cycle_id: null}`；
  - 模块：在两个模块的组之间 `{modules: {remove: [来源], add: [目标]}}`，拖到“无”只 `{remove: [来源]}`（不清空，P7a 的增减规则；Plane 同），从“无”拖来只 `{add: [目标]}`；所持集合里已归档的、页面不认识的模块都留着；
  - 标签、负责人、状态、优先级的请求体与今天相同（既有的拖拽测试断言不改，基线同数）。
  - 类型测试：给出 `{ module_ids: [...] }` 的函数赋不进 `dropIssue`（`@ts-expect-error`）。
- **拒收**（R4）：目标的列（或看板的格子所在的泳道）拒收时，放下不发请求，以 `TOAST_TYPE.WARNING` 说列上的原因（§2.4 的那一句，Plane 的“不能移到已结束的迭代”的同类）；网页不知道的拒收（例如 store 里的迭代今天刚结束）由 nerve 的 409 经 `refuse(error)` 说原因。没有乐观的移动：组与数目都照 nerve 确认的回答（M4 的确认流水线），确认之后两组的数目与 nerve 的总数一起变化。
- 规划页的拖拽照旧跟进进度（`use-planning-page-issue-actions.ts:91-97`）。

### 2.8 表格两列表头的排序（P7a R21，R9）

**现状**：`SPREADSHEET_PROPERTY_DETAILS.modules`、`cycle`（`common.ts:269-270`）没有排序键，`TUnsortedSpreadsheetProperty`（`:167-179`）让这两项的四个键可省；`spreadsheetSort`（`issue-layouts/spreadsheet/spreadsheet-sort.ts:10-18`）对没有键的给 `undefined`，表头只给标题（`spreadsheet/columns/header-column.tsx:33-50`），`<th>` 不带 `aria-sort`。`spreadsheetSort` 经 `projectListQuery` 读 `orders`（`issue-filters.ts:43-67`，不认识的映到 `-created_at`，`:228-234`），存下的 `cycle_name` 今天在显示词里也映到 `-created_at`（基类的 `orderValues`，`page-preferences-filter.store.ts:67-91`，`show()` `:139`）。

**做法**：
- **排序的词**：`orders` 与 `orderValues` 加 `cycle_name`、`-cycle_name`、`module_name`、`-module_name`（存下的名字排序从此照原样显示、照原样发出）；显示面板的排序选项照 Plane 的一组，不加名字（plane-contract §2.3）。
- **两列的键**：`modules` 为 `module_name`/`-module_name`，`cycle` 为 `cycle_name`/`-cycle_name`，标题照名字列的先例“A”“Z”（`common.ts` 的标签、负责人）。
- **删去死分支**（R9，E9）：两项都有键之后，`TUnsortedSpreadsheetProperty` 与映射类型里的条件分支、`spreadsheetSort` 的 `undefined` 一支、表头的“只有标题”一支与它的注释（`spreadsheet-sort.ts:6-9`、`header-column.tsx:33-36`、`common.ts:167-170`）都没有读者，删去；`aria-sort` 恒有。P7a 钉住“两列没有排序”的测试（`spreadsheet/planning-columns.test.tsx:130` 的 “each header its title and its icon with no sort”，`:147-151`）按契约改为两列的排序（台账写明）。
- 按名字排序要的目录在 §2.2；开关关闭时两列不显示（P7a R11），表头随之不在。

### 2.9 “添加已有”：写、命令宿主与入口（A4、A5，R2、R5、R6、R11、R20、R21）

**现状**：弹窗 `core/components/core/modals/existing-issues-list-modal.tsx`（295 行）：搜索经 `useIssueSearch`（`:83`，按会话与身份分代），提交（`:134-156`）逐项收 `confirmedIds`/`failures`（`:24-28`），失败的留在选择里、提示第一个的原因，全部确认才关（`:147-149`）；卸下时 `owner.active` 为假（`:92-97`），卸下之后的结果不报。没有迭代、模块的调用方。空态只有“新建”（`issue-layouts/empty-states/planning-page.tsx:73-89`，注释 `:17-23`）；分组头只有“+”（`list/headers/group-by-card.tsx:75-90`、`kanban/headers/group-by-card.tsx:90-104`）。空态在列表为空时、加载在核验时都渲染在 `IssueListCommandHost` 之内（`issue-layouts/issue-layout-HOC.tsx:52-62`）：弹窗若挂在空态里，第一项一确认（数目 > 0）就卸下；挂在分组头里，列表核验时卸下（P7a R16 的理由）。搜索接口（`api/modules/issue.yaml:98-118`）没有按迭代、模块排除的参数，`limit` 1–100（默认 50，`api/common.yaml:12-22`），没有分页与总数。写闸门每项目至多 2 个持连接的写，排队至多 64、各等至多 2 秒，超出 503 `server_busy`（`server/configs/config.yaml:111-116`），网页的写不重试 503。

**做法**：
- **本页的加入**（R2）：`PLANNING_PAGE_DOMAINS`（`core/hooks/planning-page-domains.ts:37-61`）每域加一列“加入本页的变化”（名字由实现定）：迭代 `{cycle_id: 本迭代}`，模块 `{modules: {add: [本模块]}}`，类型是 `IssueChange`；排除用已有的 `holds`（`:46`、`:58`）：迭代页排除已在本迭代的，模块页排除已在本模块的。
- **一批**（R6）：`usePlanningPageIssueActions` 加一个“添加已有”的动作（名字由实现定，例如 `addIssuesToView(ids, extra?)`）：每项一个 `changeIssue(id, {...extra, ...加入本页}, scope)`，经 `core/lib/` 的一个有界并发的小函数至多 2 个在途（nerve 的 `per_project`；同项目的写，`oneAtATime` `core/lib/one-at-a-time.ts:14-21` 是一个一个的先例），回答逐项的 `SelectionSubmitResult`；整批经 `usePlanningPageProgressFollow` 包一次（有确认的才重读，一批一次，不是 N 次，`use-planning-page-progress-follow.ts:26-40`）。项目页的动作也有这一项，只发给出的 `extra`（日历的 `{target_date}`），不跟进进度。
- **命令宿主**（R11，E11）：`IssueListCommandHost`（`issue-layouts/command-host.tsx`）加一种“添加已有”的请求（与 P7a 的“新建”并列，名字由实现定）：请求时定下 store 类型、给出的 `extra`（日历的日子）、搜索的参数与隐藏的规则；对话由路由的项目与调用者在它的权限身份持有（同 `holds`，`:204-209`），换路由或权限身份变了即结束；宿主以请求的 store 类型的动作提交。三个入口都经它；入口自己不渲染弹窗。
- **搜索**（R2）：规划页的入口给 `limit: 100`；隐藏已在本页的（客户端，`shouldHideIssue`）；搜索的上限与客户端的排除登记差异。
- **当前的迭代**（R2，A′）：迭代页的弹窗里，在别的迭代里的工作项的选项另显示它当前的迭代的名字（已归档的带“已归档”的说明），名字经 `usePlanningNames` 叫出（一个项目一次，必要时读已归档的列表）：提交之前就看得见“移过来”。弹窗加一个可省的、给每个选项一段说明的 prop（名字由实现定）；子工作项、关联的调用者不给，照旧。模块页不显示（加入模块从不移出别的）。
- **全被排除时**（R21）：搜索答了工作项而全被排除时，弹窗说这些都已在本迭代（本模块）（按域的新键，在 `PLANNING_PAGE_LAYOUTS` 的表里），而不是“没有找到”。
- **逐项的结果**：确认的离开选择与列表（它们已在本页，被排除）；被拒的留在选择里，弹窗照旧提示第一个的原因、不关；503 `server_busy` 同样留着、可再提交；全部确认才关。
- **空态的入口**：`PlanningPageEmptyState` 在“新建”旁加“添加已有”（`project_empty_state.cycle_work_items.cta_secondary`、`module_work_items.cta_secondary`，en `web/packages/i18n/src/locales/en/empty-state.json:38`、`:49`），条件同“新建”（`canCreate`：可加入且项目的管理员、成员，`:41-50`）；只读的页与有筛选时没有。`PLANNING_PAGE_LAYOUTS.empty.none`（`issue-layouts/planning-page-layouts.ts:50-54`、`:69-73`）加这个键；空态的注释改为本段。
- **分组头的入口**（R5、R20）：只在规划页上，“+”给出时（同一个门：`!disableIssueCreation` 等）且可加入、调用者是项目的管理员或成员、列不拒收，在“+”旁加一个“添加已有”的按钮，以“{添加已有}: {组}”为名（M3 交接 §3“每一行的控件以它那一行为名”，同“+”的 `${t("create_work_item")}: ${title}`）。只加入本页，不带组的 payload（Plane 的 `addIssuesToView` 同）。“+”不改（P5b、P6b、P7a 的测试照旧）。
- **M3 交接 §1、§3**：弹窗经 M4 的搜索接口（§1 已由 M4 做到，新调用方保留它的取消、会话与异步边界）；标题经 `ModalTitle`（`:165`），`ModalCore` 是 Headless UI 的 `Dialog`，后面的页面 inert；选项里没有只在指针经过时显示的链接（`:233-251`），列表 `static modal={false}`（`:224-229`）；已选的移除按钮以“{移除} {名字}”为名（`:196-203`）。本段在浏览器里核对（§2.14，A5）。

### 2.10 日历的“添加已有”（A4，R6）

**现状**：日子的“添加已有”（`issue-layouts/calendar/quick-add-issue-actions.tsx:46-59`）经 `useIssueDetail().updateIssue` 对每项只发 `{target_date}`（经项目页的 `updateIssue`），全部同时发出（`Promise.allSettled`）；在迭代页、模块页上工作项得了日子却不加入本页，于是不出现在本页，也不跟进进度。搜索是 `{target_date: ["none"]}`（`:79`），隐藏的规则在 `:81-84`。弹窗渲染在日子的格子里。日历的 store 类型是 PROJECT、CYCLE、MODULE（`calendar/base-calendar-root.tsx:35`）。M4 的 I4 钉住项目页：搜索的参数恰为 `{project_id, target_date: "none"}`、请求体恰为 `{ target_date }`（`e2e/stories/issue/i4-list-order-calendar.spec.ts:503`、`:516-517`）。

**做法**：
- 日子的“添加已有”经命令宿主（§2.9）请求，以本页的动作提交：规划页上每项**一个**变化 `{target_date: 那天, ...加入本页}`（nerve 原子地改，被拒时日子也不变，plane-contract R7），搜索加 `limit: 100`，另隐藏已在本页的；项目页上照旧恰为 `{target_date}`，搜索的参数不变（I4 一行不改，是回归保护）。
- 两种页都至多 2 个在途；规划页一批一次进度重读。

### 2.11 写的路径一览（请求体何时定、被拒时）

| 写 | 经 | 请求体何时定 | 被拒时 |
|---|---|---|---|
| 拖到迭代的组 | `dropIssue` → 写入单元的拖拽队列 | 写队列轮到时：`{cycle_id: 目标 \| null}` 经 `changeBody`，再加 `sort_order`（那时的列表证据） | 拒收的列：不发，提示列上的原因；nerve 拒绝：`useRefusalToast` 说原因；没有乐观的移动，行与数目照 nerve |
| 拖到模块的组 | 同上 | 写队列轮到时：所持集合去掉来源、补上目标（拖到“无”只去掉来源，从“无”来只补上），已归档与不认识的留着 | 同上 |
| 分组里的快速新建 | 本页的 `createIssue`（规划页经本页的注入，P7a R2） | 新建没有所持：调用时以组的 payload 并默认状态，经 `quick-add/root.tsx` 的预填类型与白名单（加两键） | 拒收的列不给；被拒照快速新建既有的写法 |
| “添加已有”（空态、分组头） | 每项 `changeIssue(id, 加入本页, scope)`，至多 2 个在途，一批一次进度重读 | 写队列轮到时：迭代 `{cycle_id}`（单值，P7a R3）；模块 `{module_ids: 所持 ∪ {本模块}}` | 逐项：确认的离开选择，被拒的留在选择里，提示第一个的原因、弹窗不关；503 同样留着可再提交 |
| 日历的“添加已有” | 规划页：每项 `changeIssue(id, {target_date, ...加入本页})`；项目页：`{target_date}` | 同上，一个请求，原子 | 同上；被拒时日子也不变 |
| 本页偏好（筛选、分组、排序、表头） | `PagePreferencesFilter` 每个字段一条队列（P5b 起） | 轮到时整字段替换；筛选经每页的转换（§2.3），本页的键赋不进去 | 照 P6b：提示、控件留在 nerve 确认的值 |

### 2.12 读者一览（门与名字）

| 读者 | 读什么 | 门 | 值的名字 |
|---|---|---|---|
| 目录（`prepareCatalog`） | 行的各项目的两份列表（项目不在 store 时先读项目） | 列表自己的读（绑定的范围），不看开关（R17） | —（给比较器） |
| 列表、看板、泳道的宿主 | 组的名字 | `usePlanningNames`（有角色、项目未归档、开关打开） | `usePlanningNames` |
| 筛选的选项 | 两份未归档的列表 | `usePlanningListReads`（同上） | 已持的已归档值：`usePlanningNames` |
| 分组、子分组的选项 | — | 路由的项目的开关（已选的留着） | — |
| “添加已有”选项的当前迭代 | 选项的工作项的迭代 | `usePlanningNames` | `usePlanningNames` |
| “添加已有”的入口 | — | 可加入（`joinable`，含开关）且管理员、成员，列不拒收 | — |

每个显示迭代、模块的值的新面（组头、泳道头、筛选里已持的已归档值、“添加已有”的当前迭代）都调 `usePlanningNames`（P7a 的约束），各有一个测试以“已归档的名字由它读到”核对、一个“不调它”的变异须让测试失败。

### 2.13 守卫、文案与告警

- **`project-address-route`**：新建与改写的组件、hook、store 随各自的 Task 加入规则的文件表并带样例（hit 与 miss 各至少一例）；新文件只经路由参数或 props 取工作区、项目；被取代的路径留在表里阻止复活。
- **`.oxlintrc.json` 的 swr 限制**：用 `useSessionSWR` 的新文件随它的 Task 加入（本段的新读者多经 P7a 的 hook，预计没有新的直接读者）；只收紧。
- **告警上限** `lint-cap` 恰为 44，只降不升；新文件不得新增告警。**knip**：新文件在第一个产品读者出现的 Task 里建立；只给测试读的导出不算读者（由审查执行）。
- **关键词**：phase 仍为 `M6/P7`（T1 以 `node tools/keywords.mjs` 核对没有到期的例外）。
- **文案**（R19）：读 `common.cycle`、`common.modules`（分组的标题，与显示属性同名，不加 `common.module`）、`cycle.no_cycle`、`module.no_module`（筛选的“无”）、`common.archived`、`errors.planning_*`（拒收的说明）、`issue.add.existing`、`project_empty_state.*.cta_secondary`（P7a 留给本段）；新键：`common.filter_options.fields.cycle_id`、`module_id`，分组头的“添加已有”按钮名若要新键，“全被排除”的两句（按域）；en 与 zh-CN 同一提交，`check:sync` 不许一个键是另一个键的前缀。
- **行数**（E15）：手写文件约 400 行以内：`issue-list-binding.ts` 先拆（§2.1）；`use-work-item-filters-config.tsx`（353）的两个配置另出一件；`command-host.tsx`（255，约 +60）、`kanban/swimlanes.tsx`（338，约 +10）、`issue-filters.ts`（317，约 +25）、`use-issues-actions.tsx`（316，约 +15）在上限内；`@nerve/constants` 的 `filter.ts` 约 389 → 280；工作项根 store（320）与写入单元（302）不加（M6-M8-08，T17 以 `wc -l` 核对）。
- `core/store/root.store.test.ts:106` 的 store 数仍是 21。
- 规划页测试的共用替身 `issue-layouts/roots/fake-planning-page.tsx` 的模块级 `opened`（A21）属 P7c；用它的测试都先打开页。

### 2.14 e2e（E13，R13）

页面故事放在 `e2e/stories/planning/`，标题 `G<n> (page): …`，文件名合 `planning/g\d+-[\w-]+-pages?\.spec\.ts`，纳入三种引擎的页面配置 `e2e/playwright.m6-pages.config.ts`；CI 照常只跑默认配置。世界经接口建立（新项目的两个开关默认关闭，故事先打开）；帮手从原处导入，不复制（`planning-list-pages.ts` 的 `openListPage`、`rowOptions`、`shownItems`、`bodiesSent`、`cycleListWorld`、`moduleListWorld`，`settings-pages.ts` 的 `answerTo`、`sentHeld`、`expectListBesideButton`、`transitionsEnded`，`cycle-pages.ts` 的 `dismissToasts`、`toast`、`recordApi`，`dialogs.ts` 的 `modalDialog`，`assert/issue-planning.ts` 的 `expectCycleRows`、`expectModuleRows`、`expectPlanningActivity`）；名字冲突时改名，不留别名。

| 文件 | 覆盖 | 本机 chromium |
|---|---|---|
| `g15-work-item-lists-page.spec.ts` | G15 (page) 的工作项一侧，项目页的列表。迭代：进行中 Sprint、未开始 Next、已结束 Ended、已归档 Old；模块 Alpha、Beta 与已归档的 Gamma；W1 在 Sprint 与 Alpha、Gamma，W2 在 Old 与 Gamma，W3 都没有，W4 在 Ended 与 Beta。**筛选**：迭代一栏的选项恰为“无”、Sprint、Next、Ended（没有 Old），选“无”：查询带 `cycle_id=none`，行恰为 W3；去掉。**分组**：按迭代（列表）列的次序 Sprint、Next、Ended、Old（名字与“已归档”的说明，只一次已归档列表的读）、“无”最后；按模块：W1 在 Alpha 与 Gamma 两组，Gamma 一组（W1、W2）以名字与说明。**拖拽**（看板，按迭代）：W3 从“无”拖到 Sprint，只发一个 `PATCH`，请求体恰为 `{cycle_id, sort_order}`，两列的数目随 nerve 的回答变化，数据库与一条动态；W1 拖到 Ended 的列：不发请求，提示“已结束”的那一句。**表头**（表格）：迭代一列 A→Z 与 Z→A：查询的 `order_by` 为 `cycle_name`、`-cycle_name`，行的次序，`aria-sort` 为 `ascending`、`descending`；模块一列 Z→A 一次（re-74 F5）：`order_by=-module_name`，行依次 W2、W4、W1、W3（多模块的 W1 以它最小的模块名 Alpha 排，已归档的 Gamma 也有名字，没有模块的在最后；取最大的名字时 W1 会排在 W4 之前），`aria-sort` 为 `descending` | ≤ 9 秒 |
| `g4-add-existing-page.spec.ts` | **G4 (page)**：迭代 Sprint（进行中、空）、Next（未开始）里有 Y，X、Z 不在迭代里。空态的“添加已有”打开弹窗：`toHaveAccessibleName`、后面的页面 inert、搜索带 `limit=100`、方向键到达选项；选项含 X、Y（另显示 “Next”）、Z；三个都选，接口在页面背后归档 Z；提交：X、Y 各一个 `PATCH` 恰为 `{cycle_id: Sprint}`，Z 被拒（409 `issue.archived`）留在选择里（移除按钮以“Remove Z”为名可见）、提示原因，空态在 X 确认之后卸下而弹窗仍在（命令宿主），本迭代的进度只重读一次；数据库 X、Y 在 Sprint，Y 一条“移过来”的动态。同页换到日历：某天的“添加已有”不列已在本迭代的，选一个没有日子、不在迭代里的 U：只一个 `PATCH`，恰为 `{target_date, cycle_id}`，数据库两者都改。**G11 (page)**：模块 Alpha 页（按状态分组）在 Todo 组头点“添加已有: Todo”，选在 Beta 与已归档 Gamma 里、状态为 Done 的 M：只一个 `PATCH`，请求体恰为 `{module_ids: [Beta, Gamma, Alpha]}`（按所持的次序补上本模块；没有 `state_id`：只加入本页） | ≤ 5 秒 |

**写法**（P5b–P7a 阶段记录 §5 的教训，P7a 记录 §5）：打开列表之前先等上方的内容与过渡（M6-M8-13）；扣住回答、读请求体的路由在页面载入之后才装；可访问名的 `,:;` 前多一个空格，以 ` ?,` 之类的正则匹配；弹窗自己的读以 `answerTo` 等它；e2e 的变异要落在被证的一步，没有能编译又让故事失败的产品变异时以期望错误的 RED 证实断言；变异之后以干净的源重建；不用 `page.reload()`，不在重新载入之后核对控制台（M6-M8-09）；关通知只用 `dismissToasts`（M6-M8-10）；有两道守卫时以嵌套变异证明并写明单去一道为何等价。

**时长**（E13）：本段两个新文件本机 chromium 合计 **≤ 14 秒**（P7c ≤ 8 秒；re-74 F5 给 G15 加了模块一列的一向，约 1 秒，两个文件之间的分配由 8.5 + 5.5 改为 9 + 5 秒；分配只是指引，约束的是合计）；超过时先压缩（例如把已有单测覆盖的一步移出故事），仍超就停下报告主控（预算与 CI 的作业结构是负责人的）。数据点：

| 数据点 | 值 | 出处 |
|---|---|---|
| 记下的最慢的 runner：main CI 43549053092 的 e2e 作业（那时还没有 I4 的拆分与 P7a 的故事） | 27 分 41 秒 = 1661 秒 | P7a spec §2.11 |
| I4 的拆分（本机 chromium） | +5 秒 | 同上 |
| P7a 的两个页面故事（本机 chromium） | 6.5 秒 | P7a 阶段记录 §5 |
| P7a 的最终分支 CI 43600392398 的 e2e 作业（已含二者） | 23 分 48 秒 = 1428 秒 | 台账 setup |
| P7a 合入后的 main CI 43603772626 attempt 1 的 e2e 作业 132478430823（14:31:09Z–14:54:52Z，全绿） | 23 分 43 秒 = 1423 秒 | 主控以 jobs 接口核对（台账），T1 再以收据核对 |

T1 照 P6b spec §2.10 的公式（P7a spec §2.11）重算余量并写进台账；T9 的检查点与 T17 推送之前再各算一次：
- **估算**：基线（最后一次绿的 main CI 的 e2e 作业，即 43603772626 的 1423 秒）加上新文件 chromium 时长 X × 1.67–2.25、两个 worker 摊开（14 秒约加 12–16 秒），约 1435–1439 秒，即 23 分 55–59 秒。
- **上界**：S + X × f，f = 3.6–4.1；S 取“1661 + (5 + 6.5) × f”与已含 I4 的拆分和 P7a 故事的各次实测（43600392398 的 1428 秒、43603772626 的 1423 秒、T9 检查点的分支 CI）中的最大者。以 27 分 41 秒为锚，S 约 1702–1708 秒，X = 14 秒时上界约 29 分 13–26 秒；P7b 与 P7c 合计的余量 X ≤ (1800 − S) / f，约 22.4–27.1 秒（取代 P7a spec §5 的“约剩 28.5–50.5 秒”，`research/web-delta.md` §5.2）。**余量在刀口上**：P7b 14 + P7c 8 = 22.0 秒，对 f = 4.1 一端的 22.4 秒只差 0.4 秒；本段两个文件超出 14 秒的任何一点都会让 P7c 的 8 秒越过它，故 14 秒是硬上限，不是目标。
- **校准点**：T9 检查点的分支 CI 实测的 e2e 作业时长与基线之比（那时还没有新故事，但项目页的列表多了词表），慢的 runner（×1.31，M6-M8-01）另算一次。
- 43603772626（23 分 43 秒）比 27 分 41 秒快，锚不变；更快的不放宽（上界以最慢的为锚），之后的实测更慢时它成为锚，余量随之变小。T1 算出的 P7b+P7c 的余量小于 22 秒、或估算、上界、任一次实测（T9、推送后的分支 CI）超过 30 分钟时，**停下报告负责人、不推送**；e2e 的分片、期限与作业结构不经负责人不改（E13，M6-M8-06 的教训）。

## 3. 局部裁定

- **R1 范围与拆段（主控裁定 E1）**：P7b 是 P7 的第二段（P7a spec R1）：P7a 的回填与余量、先拆绑定、目录、词表、按页收窄、分组、按页的常量、筛选的界面、拖拽（检查点）、表头的排序、“添加已有”的写与三个入口、页面故事、文档与回执、门禁与合入，17 个 Task（预检 re-74 F4 把按页的常量与开关、分组里的新建拆为 T6、T7；主控接受 17 个在设计 §12 的“约 16”之内）。理由：调研估本段约 1,100–1,250 行产品代码、50–60 个文件，是 P7a 的一半，一段之内（`research/web-delta.md` §6.2）；依赖的次序是目录在词表之前（按名字排序要目录）、词与列同时（存下的分组一被认出就要有列）、写在入口之前（knip 的读者）。不取把“添加已有”拆到 P7c（P7c 的 G19 要核它的弹窗）。代价：本段 Task 多是中等大小，T5、T11 偏大，各写明超量时的退路（plan）。
- **R2 “添加已有”排除本页的、各项显示当前的迭代（主控裁定 E2，A′）**：迭代页排除已在本迭代的，在别的迭代里的照列、选项另显示它当前的迭代（名字经 `usePlanningNames`，已归档的带说明），提交即移过来（一条 `updated` 的动态）；模块页排除已在本模块的；规划页的搜索页大小 100。理由：设计 §7“排除已在其中的工作项”（`../M6-design.md` §7）与 G4“加入别的迭代就移过去”且点名迭代页的“添加已有”；在上限 100、没有分页的搜索下，排除得少，弹窗空得少；显示当前的迭代让“移过来”在提交之前看得见。不取 Plane 的“排除在任一迭代里的”（`plane/apps/api/plane/app/views/search/issue.py:78-83`、`:132-133`；更多行被藏、移动要走下拉框），也不取不显示当前迭代的 A（悄悄地移）。代价：一次名字的读（一个项目一次）、选项的一段说明；与 Plane 不同、搜索的上限与客户端的排除登记差异（§2.9）。
- **R3 拖拽的请求体在写的轮到时算，模块拖到“无”只移出来源（主控裁定 E3）**：`dropIssue` 收的函数给 `IssueChange`，请求体在轮到时经 `changeBody`；迭代 `{cycle_id: 目标 | null}`；模块之间 `{remove: [来源], add: [目标]}`，拖到“无”只 `{remove: [来源]}`。理由：`module_ids` 整体替换，P7a 的增减规则（R3）在轮到时作用于所持的集合，已归档与不认识的留着；Plane 拖到“无”也只移出来源（`plane/apps/web/core/components/issues/issue-layouts/utils.tsx:594-600`）；一个工作项在多个模块里，“拖出这一组”只是离开这一个。`IssueChange` 声明 `module_ids?: never`，给整集合的拖拽函数编译不过（re-72 F1 的教训：类型让错的写法赋不进去）。不取“拖到‘无’清空集合”（标签的先例 `group-drop.ts:142-147`：模块的集合里有页面看不见的已归档模块，清空会悄悄移出它们）。代价：拖到“无”的工作项若还在别的模块里，仍出现在那些组。
- **R4 分组的列：目录的次序、已归档的在后、“无”最后；拒收的列（主控裁定 E4）**：列按 store 里未归档的迭代（模块）的 nerve 次序，然后是回答里有的已归档的（以名字与说明），“无”在最后；网页定次序，不照 nerve 按 id 文字的次序；已结束的迭代、已归档的迭代与模块（以及开关关闭时有值的列）拒收：放下不发请求、提示 nerve 拒绝的同一句，不给“+”、快速新建、“添加已有”；网页不知道的拒收由 nerve 的 409 说原因；泳道并入回答的组。拒收是列上的一个字段（取值为 nerve 的问题码），三处都读它。理由：nerve 回答已归档的组且不带名字（plane-contract §7.5、R8），Plane 只给未归档的列（`utils.tsx:185-240`），已归档的迭代里的工作项分组时就不见了；完成的迭代与已归档的对象 nerve 以 409 拒绝加入，Plane 在完成的迭代上照样新建再加入失败（plane-contract §8 #15）。一个字段而非“拒绝放下”“禁止新建”两个，免得两者走岔。不取 nerve 的 id 文字次序（对用户没有意义），也不取隐藏已归档的组（工作项会消失）。代价：已归档的组的名字要一次已归档列表的读（`usePlanningNames`）；登记差异。
- **R5 分组头的“添加已有”只加入本页（主控裁定 E5）**：不带组的 payload。理由：Plane 的分组头调本页的 `addIssuesToView`（plane-contract §3.3）；“添加已有”是加入本页，不是改成那一组的值（把一项拖进组才是）。不取“同一个 `PATCH` 再带组的值”（一次改两件事，被拒时说不清是哪一件）。代价：加进来的工作项可能落在别的组。
- **R6 一批：逐个经 `changeIssue`、每项目至多 2 个在途、一批一次进度重读（主控裁定 E6）**：逐项的结果、失败的留在选择里可再提交；日历在规划页上每项一个 `{target_date, ...加入本页}`，项目页恰为 `{target_date}`。理由：设计 §15 第 12 条（逐个 `PATCH`、逐项报告）；写闸门 `per_project: 2`（`server/configs/config.yaml:111-116`），网页不重试 503，全部同时发出（日历今天的 `Promise.allSettled`）选满 100 项会撞上 503；一批的进度重读一次就够（`followProgress` 每写一次读一次是为单个的写）；日历的一个请求让 nerve 原子地改（plane-contract R7），Plane 先改日子再加入，加入失败时日子已改。不取并发不设限（503），也不取一个一个（慢一倍，没有必要：nerve 允许 2 个）。代价：大批的加入要若干个来回；I4 的项目页断言不改即回归保护。
- **R7 筛选含“无”，选项是未归档的，已持的已归档值有名字，本页的键不给（主控裁定 E7）**：理由：G15“含无”，nerve 支持 `none`（`api/modules/issue.yaml:467-488`）而 Plane 的富筛选没有（plane-contract §2.1，登记差异）；已归档的不作新选项（同下拉框），但存下的条件里的已归档值要显示（否则是“不可用的选项”）；迭代页的偏好不收 `cycle_id`（400，P5b R9、P6b R12）。不取 Plane 给本页的键（nerve 拒绝）。代价：一次已归档列表的读（只在已持时）。
- **R8 开关关闭：不给未选的选项，存下的照旧（主控裁定 E8）**：项目的页上，路由的项目关着某域时不给该域未选的筛选与分组（显示属性与表格的两列 P7a R11 已如此），已选的分组留在面板里并选中、已应用的筛选条件留着，都可改掉；列表照 nerve 的回答（列表的词与两份列表的读都不看开关，`listvocab.go`、`planning.yaml:15-27`），界面不读该域的名字（组名“—”、条件“不可用的选项”），有值的列拒收。理由：Plane 在开关关着时藏起选项（`plane/apps/web/core/components/issues/issue-layouts/filters/header/display-filters/display-filters-selection.tsx:53-59`）；悄悄去掉存下的词会改变列表、也改变别的标签页里同一人的偏好；把已选的也藏起来，用户就看不到列表为什么这样分组、也改不掉。不取“关着时从查询里去掉这些词”（改变结果），也不取“关着时照常给全部选项”（为关掉的功能给控件）。代价：开关关着而偏好里还有这些词时，组头与条件没有名字；登记差异。
- **R9 删去排序键的死分支（主控裁定 E9）**：两列有了排序键之后，`TUnsortedSpreadsheetProperty`、映射类型的条件分支、`spreadsheetSort` 的 `undefined`、表头的“只有标题”与它们的注释没有读者，删去。理由：删掉不用的代码（项目的规则），P7a spec §5 写的“不改类型”是在本段之前的接缝说明，不是要留着死分支。代价：P7a 钉住“两列没有排序”的测试按契约改写。
- **R10 归档页不加迭代、模块的词（主控裁定 E10）**：理由：G15 说的是“项目的工作项列表”；归档页的筛选与分组 Plane 有（plane-contract §2.1），是否给它由 P8 的最终矩阵定；本段登记差异。代价：归档页上不能按迭代、模块筛选分组。
- **R11 “添加已有”的弹窗在命令宿主（主控裁定 E11）**：三个入口都经 `IssueListCommandHost` 请求。理由：空态在第一项确认后卸下、分组头在核验时卸下（`issue-layout-HOC.tsx:52-62`），弹窗挂在那里，其余的结果就不报（`owner.active` 为假）；这是构造上的（P7a R16 的先例）。日历的格子今天在项目页上活得过（I4），一并经宿主，三处一个写法。代价：宿主多一种对话。
- **R12 分组里的快速新建带上组的值（主控裁定 E12）**：快速新建的预填取自列（与泳道）的 payload，两份逐字的预填函数合为一个。理由：今天按迭代、模块分组时预填给出 `{cycle: id}`、`{module: id}`，快速新建的根的预填类型与提交的白名单（`quick-add/root.tsx:34-36`、`:179-180`）没有 `cycle_id`、`module_ids`，快速新建不加入组的迭代、模块（主控的 E12 原写作“`createBody` 丢掉”，`createBody` 自 P7a 起放行两者，预检 re-74 F3 更正了机制，裁定不变）；payload 已是每种分组的正确的值（Plane 的快速新建同样用组的值，plane-contract §2.6）。代价：快速新建的既有测试以基线核对同数。
- **R13 e2e 的时长预算（主控裁定 E13）**：本段新的页面故事本机 chromium ≤ 14 秒（P7c ≤ 8 秒）；T1 以 P7a 实测的 CI 重算余量，T9、T17 再算；估算或上界超过 30 分钟时停下报告负责人、不推送；CI 的期限、超时与作业结构不经负责人不改（§2.14）。
- **R14 只改网页（主控裁定 E14）**：不改 `server/`、`api/`；遇到网页上处理不了的契约缺口就停下报告（架构一级）。写 spec 时没有找到这样的缺口（§6）。
- **R15 行数与质量（主控裁定 E15）**：手写文件约 400 行以内（先拆、只搬不改、留基线）；没有开关、桩或兼容；类型来自 OpenAPI 生成的类型；守卫（`project-address-route`、swr 限制、告警上限 44、knip、i18n）只收紧（§2.13）。
- **R16 绑定拆为读、目录与绑定三件**：只拆出目录会剩约 410 行，仍过上限；读与目录是两份不同的职责（页的读与范围的核对、名字的准备），各自约 110、130 行。理由：re-72 F9、P7a spec §5、D7 的先例。不取只拆目录。代价：一个只搬不改的 Task，以基线收据核对。
- **R17 目录取自规划 store、只在按名字排序时读、以列表自己的读为门**：目录的两项是规划 store 的 `getCycleById`、`getModuleById` 的查找（`IssueCatalog` 的两项收窄为只读的查找）；只在 `±cycle_name`、`±module_name` 时读行的各项目（不在 store 的先读项目）的两份列表：有叫不出的就重读未归档的、仍叫不出再重读已归档的，store 已有也读（同状态、标签，re-74 F2），与 `usePlanningNames` 共用定次序的纯函数（hook 另有“已持则不读”）；门是列表自己的读，不看开关。理由：P7a spec §2.3“P7b 的目录从同一份 store 的值建立，不另读一份”；`getCycleById` 已有“最新的一份、项目仍给”的规则，另建一张表会重写它、根还要加 getter（M6-M8-08）；开关关着时 nerve 照旧按名字排序（E8 的“照 nerve 的回答”），以开关为门会让这样的列表整个失败。**这一条与主控的说明“新的规划数据的读者都以开关为门”在开关关闭的边上不同**：界面上的读者（§2.12）都以开关为门，只有目录这个列表引擎的部件不看开关；另一种写法（目录也以开关为门）的代价是存下的名字排序在开关关闭后让列表失败。主控在预检之前已确认这一分界（台账的 Ruling：开关关掉后保存的名字排序 nerve 仍回答，按开关把门会让整个列表失败，合 E8；代价是开关关掉时目录仍会为名字排序读一次规划列表），预检 re-74 没有异议。不取“建一张计算出的表并在根上加 getter”（规则两处），也不取“分组时也读”（组头的名字是界面的事，P7a 的约束由宿主的 `usePlanningNames`）。代价：共用的类型改窄（字面量照旧满足）；别处的改名在叫不出之前不重读，名字可能是旧的（同状态、标签）。
- **R18 偏好的筛选按页收窄的写法**：网页一侧的本页筛选类型以 `never` 禁止本页的键（`Omit<ProjectIssueFilters, "cycle_id"> & { cycle_id?: never }`，模块页同）；基类以筛选的类型为参数，每页一个回答本页类型的转换，service 的 `update` 写成属性、收本页的类型。理由：P6b R16；生成的 `CycleIssueFilters` 只是 `ProjectIssueFilters` 的结构子集（缺一个可省的属性），一个 `ProjectIssueFilters` 的值不论变型都赋得进去，只“去掉一个键”的类型挡不住任何东西（预检 re-74 F1，与 re-72 F1 同一个教训：类型要让错的写法赋不进去，`never` 才做得到）；属性写法另让配错的 service 赋不进去（方法写法的双变会放过它）。类型测试含一个不是字面量的 `ProjectIssueFilters` 的值，去掉 `never` 的变异让它们失效。不取只在运行时滤掉本页的键（类型上仍可赋），也不取只用生成的类型（挡不住）。代价：基类的一个类型参数、两个本页的类型与三处构造的一个转换。
- **R19 名字与文案**：分组的标题与显示属性、表格的列同名（`common.cycle`、`common.modules`），不加 `common.module`；筛选的“无”读已有的 `cycle.no_cycle`、`module.no_module`；拒收的说明读 nerve 拒绝的同一句（`errors.planning_*`）；新键只有筛选的两个字段名、按域的“全被排除”两句与分组头按钮名（若要）。理由：同一个面板里同一个词一个名字；拒收的说明与 nerve 的拒绝不该说两种话。代价：分组面板里“Modules”是复数（与“States”“Labels”“Assignees”一样）。
- **R20 分组头的“添加已有”是“+”旁的第二个按钮**：以“{添加已有}: {组}”为名，只在规划页、“+”给出、可加入、管理员与成员、列不拒收时。理由：Plane 把两项放进“+”的菜单（`plane/apps/web/core/components/issues/issue-layouts/list/headers/group-by-card.tsx:123-157`），换成菜单要改 P5b–P7a 都按名字点的“+”与它的测试；两个各以组为名的按钮对键盘与读屏同样清楚（M3 交接 §3）。代价：规划页的分组头多一个图标按钮；登记差异。
- **R21 全被排除时说明**：搜索答了工作项而全被排除时，弹窗说它们已在本迭代（本模块）。理由：搜索的上限 100、没有分页、排除在客户端（合同的缺口，设计 §7 已接受），最近改过的 100 项都已在本页时，“没有找到”是错的说法。不取改接口加排除参数（`api/` 的变化，负责人一级）。代价：按域的两个新键。
- **R22 成员列表与本地存的两页**：个人页的分组排除迭代、模块（`memberListQuery`，nerve 422），筛选本已不放，不给名字排序的控件，目录照按项目读；本地存的两页（个人页、归档页）的本地词表不加。理由：G15 与设计 §7 的范围是项目列表（“我的工作项”对两词的筛选与分组回答 422，plane-contract §7.5）；个人页的排序选项里本没有名字（`@nerve/constants` 的 `filter.ts:42-68`）。代价：无。

本段的裁定都是局部决定：R1–R15 是主控在写 spec 之前的裁定（E1–E15），R16–R22 在已批准的设计与 P5b–P7a 的先例之内，不改接口、不改服务端、不跨后端模块。R17 与主控说明的一句在开关关闭的边上不同，主控已确认（R17）；预检 [re-74](../../../reviews/re-74-M6P7b预检.md) 的 F1–F9 都已改入（F3 更正了 E12 所说的机制、裁定不变）；**没有须负责人决定的事**（e2e 的作业结构只在余量不够时才提出，R13）。

## 4. 验收标准

- [ ] A1 拆分：`issue-list-binding.ts` 拆为读、目录与绑定三件，各在约 400 行以内，只搬不改（`git diff -M`），测试只改导入（基线逐文件同名同数），三件在 `project-address-route` 里。
- [ ] A2 目录：两项取自规划 store 的查找（`IssueCatalog` 收窄）；只在按名字排序时读：行的各项目不在 store 的先读项目，有叫不出的值时重读未归档的列表（store 已有也读）、仍叫不出再重读已归档的，每次各至多一次，都叫得出时不读，与 `usePlanningNames` 同一条定次序的纯函数（hook 另有“已持则不读”）；别处新建的迭代被行加入后叫得出；两份都重读了仍叫不出时大声失败；开关关闭时照读；写的确认为新的 id 读；根不加 getter。每个性质有单测，变异能让测试失败。
- [ ] A3 筛选的词与按页收窄：`cycle_id`、`module_id` 含 `none`；成员列表排除两词的分组；迭代页的筛选修改不发 `cycle_id`、模块页不发 `module_id`（单测与变异）；本页的键在类型上赋不进去：本页的类型以 `never` 禁止本页的键，类型测试含转换配错、service 配错与一个不是字面量的 `ProjectIssueFilters` 的值，去掉 `never` 的变异让它们失效（收据）。
- [ ] A4 分组与泳道：列的次序（未归档的、已归档的以名字与说明、回答里被删除的、“无”最后）；多模块的工作项在每个模块的组；泳道并入回答的组；组头与泳道头的名字经 `usePlanningNames`；拒收的列（已结束、已归档、开关关闭）一个字段。
- [ ] A5 按页的常量与开关：`issues` 有两词的筛选、分组、子分组，`cycle_issues`、`module_issues` 由它推出、不含本页的词（两页页头的测试不改即通过）；开关关闭时不给未选的、已选的留着；`displayPropertyOn` 由同一张表的谓词取代。
- [ ] A6 分组里的新建：快速新建带组的值（按迭代、模块分组时 `{cycle_id}`、`{module_ids}`），一个共用的预填函数，`quick-add/root.tsx` 的预填类型与白名单加两键，既有的快速新建测试同数；拒收的列没有“+”、快速新建与“添加已有”。
- [ ] A7 筛选的界面：两个工厂、选项（未归档的，含“无”，状态图标）、已持的已归档值以名字与说明显示、开关关闭时不列入“加筛选”而已应用的条件留着；配置另出一件，`use-work-item-filters-config.tsx` 在上限内。
- [ ] A8 拖拽：迭代、模块的组可拖；请求体在轮到时经 `changeBody`（迭代单值，模块增减，拖到“无”只移出来源，已归档与不认识的留着）；拖拽函数给 `IssueChange`（类型测试）；拒收的列不发请求、提示那一句；nerve 的拒绝说原因；既有的拖拽测试断言不改。
- [ ] A9 表头的排序：两列两向的名字排序（查询、行的次序、`aria-sort`）；存下的名字排序照原样显示与发出；死分支删去。
- [ ] A10 “添加已有”：三个入口都经命令宿主，弹窗活过空态的卸下与列表的核验；排除已在本页的、迭代页的选项显示当前的迭代（`usePlanningNames`）、全被排除时说明、规划页的搜索 `limit: 100`；每项一个 `changeIssue`、至多 2 个在途、一批一次进度重读、逐项结果、失败的留着；分组头只加入本页、只在规划页可加入时且列不拒收、以组为名；空态的入口条件同“新建”。
- [ ] A11 日历：规划页每项一个 `{target_date, ...加入本页}`、原子、隐藏已在本页的；项目页的搜索参数与请求体不变（I4 一行不改）。
- [ ] A12 页面故事：G15 (page) 的工作项一侧与 G4、G11 (page) 的“添加已有”在三种引擎（§2.14），含 M3 交接 §3 的弹窗各项（A5）；两个新文件本机 chromium 合计 ≤ 14 秒。
- [ ] A13 守卫：`project-address-route`（各 Task 的新件与改写的件）、swr 限制、告警上限 44、knip、关键词（phase `M6/P7`）、i18n（新键、读的键）、`root.store.test.ts` 21 个 store、根与写入单元的行数（M6-M8-08）。
- [ ] A14 文档：P7a 的回填（阶段记录 §7、spec A12、设计 §14、M6-M8-01 的数据点）与 e2e 余量的重算；交接回执（M4 M6-02 含“何时验收”的逐项，M3 §1、§3 的 `existing-issues-list-modal.tsx`）；差异清单、前端改动清单、设计 §8/§12/§14、M8 交接的相关条目。
- [ ] A15 七项本地门禁（`make test` 照负责人 2026-11-11 的规则）、deep-review 修复闭环、准确的分支 CI 与 noff 后的 main CI 全绿、归档清理。（合入后由 P7c 的 T1 勾选）

## 5. 后续责任

**P7c**（命令面板、移动端页头与遗留）：分组时页头的数目（A19，项目页与归档页的页头在分组时读不到 `""` 组的数，`core/components/issues/header.tsx:49`）；迭代页、模块页与项目页的移动端页头（A15、X2：项目页的移动端页头的显示面板已随本段的组件一级的开关门给出两词的分组，P7c 只管页头本身）；命令面板（A16）、A17、A18、A20（本段读了 `cta_secondary`）、A21（`fake-planning-page.tsx` 的 `opened`）、X1；G19 的工作项一侧（其中“添加已有”的弹窗的键盘与迟到的回答可复用本段的宿主与故事的帮手）。e2e 预算 ≤ 8 秒本机 chromium（R13），余量以本段 T17 的实测与 T1 的重算为准。

**P8**：归档页的迭代、模块词表（R10）；[M6-M8-13](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-13)；新的筛选、分组控件与“添加已有”按钮的焦点轮廓随最终矩阵核对；开关关闭时存下的词的显示（R8 的“—”与“不可用的选项”）是否要更好的说法。

**M7**：工作区级的迭代、模块列表出现之后，成员列表的目录与跨项目的视图可改读它；视图的筛选与分组的迭代、模块词随视图的页面。

**M8**：[M6-M8-01](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-01) 多 P7a 的分支 CI、main CI 与本段 T9 检查点的数据点；[M6-M8-08](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-08) 在 M6 收尾核对根与写入单元的行数（本段不加，T17 记下）；[M6-M8-03](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-03)、[M6-M8-05](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-05)、[M6-M8-07](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-07) 照旧（再现时照各条的验收语记下）。

**负责人**：e2e 的作业结构（分片）、期限与超时（R13）；搜索的服务端排除参数或批量写（R2、R6，`api/` 的变化）只在负责人要 Plane 的确切行为时才提出。

## 6. 风险

| 风险 | 处理 |
|---|---|
| 搜索的上限 100、没有分页、排除在客户端 | 规划页给 `limit: 100`，全被排除时说明（R21）；排除只排本页的（R2），藏得少；登记差异。服务端的排除参数是 `api/` 的变化，设计 §7 已接受这个缺口，不是须停下的契约缺口 |
| 一批的写撞上写闸门（503） | 至多 2 个在途（nerve 的 `per_project`）；别的标签页同时写时仍可能 503，逐项报告、留着可再提交（R6） |
| 整体替换盖掉别处的改动 | 模块的写都是增减、在轮到时作用于所持的集合（P7a R3）；接口没有前置条件，别的标签页在读与写之间的改动仍可能被盖掉，这是 nerve 的合同（P7a §6），不改接口 |
| “添加已有”把工作项从别的迭代移走 | 选项显示当前的迭代（R2），移走有一条动态；离开已结束、已归档的迭代 nerve 不拦（设计 §3.3） |
| 目录不看开关与“读者以开关为门”的说明不一致 | R17 写明分界与理由，主控已确认；另一种写法的代价写在 R17 |
| 目录与行的 hook 同时读同一份列表 | 被超过的那一次 `Reconciled.fetch` 回答 `undefined`，目录这一次可能失败；列表的读可重试，写的确认 `markUnavailable` 后重读叫出（§2.2，状态、标签今天同样），不加机制（re-74 F9） |
| 别处新建的迭代、模块 | 目录叫不出就重读，store 已有也读（§2.2，re-74 F2）；两份都重读了仍叫不出才失败 |
| 开关关闭而偏好里仍有两词 | 列表照 nerve 的回答，界面不读名字（R8）；选项不给、已选的留着可改；登记差异，P8 再看说法 |
| 拒收在网页上判断错（迭代今天刚结束） | nerve 以 409 拒绝，说 nerve 的原因（R4）；没有乐观的移动 |
| 拆分与快速新建、拖拽的改写碰坏既有行为 | T2 只搬不改留基线；快速新建与拖拽的既有测试断言不改、基线同数；类型测试挡住整集合 |
| 两页页头的“不按本页分组”测试此前不是真的保护 | 推出常量的变异（不去掉本页的词）让它们失败（T6），从此是 |
| e2e 时长 | 预算 ≤ 14 秒（R13）；T1 重算、T9 校准、T17 估算与上界；最慢的 runner 上上界约 29 分 13–26 秒，越过 30 分钟先报告负责人、不推送 |
| Task 的量 | 17 个 Task（re-74 F4 拆开了原 T6）；T5（分组的词与列）与 T11（“添加已有”的写、宿主与空态）偏大，plan 写明超量时移到下一个 Task 的部分，不再加 Task |
