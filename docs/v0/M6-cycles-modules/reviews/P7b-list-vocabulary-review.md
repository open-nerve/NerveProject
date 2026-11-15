# M6/P7b 阶段评审记录

> 已完成（2026-11-15，noff `d052213f`，main CI 43706438924 attempt 1 通过），整分支审查 Ready to merge（[re-75](../../../reviews/re-75-M6P7b整体审核.md)：没有 Critical、Important，四处 Minor 与六条观察），主控仍以修复轮 F（F1–F3）处理两处 Minor（M1、M2；M3、M4 不改），经限定复审 Ready to merge（没有 Minor，re-75 §4）；最终候选 `16c52f41`（最后一个改代码的提交是 F1 的 `e0fe04ea`，改两个分组面板、`utils.tsx` 与分组面板的测试；最后一个改 e2e 故事的是 T15 的 `55a225c8`；`16c52f41` 只改文档），其后是本记录所在的提交 `5d7adaaa`（只改文档）。`5d7adaaa` 推送后的分支 CI 43696983860 的 e2e 在 C4 (page) relation 失败，原因是描述的编辑器晚挂载、点 “Relations 1” 的按下与抬起落在不同的元素上（[M6-M8-13](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-13) 一类），本提交改这两个故事（只改 e2e 的故事、夹具与文档），最终 HEAD 因此是本提交 `4fe308ce`。最终 HEAD 的分支 CI、合入、main CI 与归档清理见 §7（由 P7c 的 T1 回填）。基线 main `f42f9d6457ccc9540a687019f3530e253fe0f7b5`（P7a 的 noff，双父 `7868a6b4`、`5d6dab04`，树与 `5d6dab04` 相同）。

## 1. 评审方式与范围

P7b 交付设计 §12 的 P7b 一行（P7 在实施之前拆为 P7a/P7b/P7c，[P7a spec](../specs/P7a-work-item-planning.md) §1、R1）：列表的目录（P3b R9 留下的空目录改由规划 store 作答，按迭代、模块的名字排序不再失败）；筛选与分组的词（含“无”，分组含已归档的组、“无”在最后，已结束与已归档的列拒收）；偏好的筛选按页以类型收窄；迭代页、模块页的显示选项由项目页推出，开关关闭时不给未选的；分组里的快速新建带组的值；拖到迭代、模块的组；表格两列表头的两向排序；空态、分组头与日历的“添加已有”（弹窗在命令宿主，每项一个变化、至多 2 个在途）；要加代码的 `issue-list-binding.ts` 先拆；G15 (page) 的工作项一侧与 G4、G11 (page) 的“添加已有”。命令面板、移动端页头与遗留在 P7c（[spec](../specs/P7b-list-vocabulary.md) §1、§5）。

- **调研**：P7 的两份现状笔记（写于 P7a 之前的 `c0572192`：`research/web.md` 606 行、`research/plane-contract.md` 375 行）与本段的增量笔记 `research/web-delta.md`（467 行，只读代理写于 `5d6dab04`，即 main `f42f9d64` 的树：P7a 造好的接缝、目录、词表、“添加已有”、e2e 的余量、Task 的估算），随台账归档；spec 引用的行号都在本段的树上重核过。
- **界定**：主控在起草 spec 之前裁定十五件事（E1–E15，台账 `controller-rulings.md`）：范围照增量笔记的 16 个 Task，先拆列表的绑定；“添加已有”取 A′（排除已在本页的、迭代页的各项显示它当前的迭代）；模块拖到“无”只移出来源；列以目录的次序、已归档的在后、“无”最后，已结束与已归档的列拒收并藏起新建；分组头的“添加已有”只加入本页；一批每项目至多 2 个在途、一次进度重读；筛选含“无”；开关关闭时藏起选项、不悄悄改存下的查询；删去死的排序分支；归档页不加词表；弹窗在命令宿主；分组里的快速新建带组的值；e2e 预算 P7b ≤ 14 秒本机 chromium（P7c ≤ 8 秒），估算或上界过 30 分钟即停下报告负责人；只改网页；约 400 行与质量。起草代理据此写 [spec](../specs/P7b-list-vocabulary.md)（R1–R22，R1–R15 照录 E1–E15；A1–A15）与 [plan](../plans/P7b-list-vocabulary.md)（16 个 Task）（`cf92e3d5`）；主控确认 R17（目录以列表自己的读为门、不看开关：开关关掉之后存下的名字排序 nerve 仍回答，以开关为门会让整个列表失败）。
- **预检**：实施前一次 fresh 只读预检（[re-74](../../../reviews/re-74-M6P7b预检.md)），结论为修正后可实施：四处 Important（F1 R18 的类型保证因结构子类型不成立，须本页的键为 `never` 的页类型、写成属性的 `update` 与非字面量的类型测试；F2 目录叫不出名字就重读、store 已有也读，纯函数只定次序；F3 分组里快速新建的门在 `quick-add/root.tsx`；F4 原 T6 拆为 T6、T7，计划为 17 个 Task）与五处 Minor（F5 G15 加模块列的一个排序方向；F6 `filter-config.test.tsx` 写全路径；F7 T2 的基线清单；F8 目录先取不在 store 的项目；F9 被超过的读照实写、不加机制），主控全部接受，与审者有三处出入（`filter-config.test.tsx` 实际存在、`priority-issues.store.test.ts` 留在基线、项目取数的先例在 `:245-246`），在实施前改入（`4ec4e9d3`）；e2e 的余量 P7b 14 + P7c 8 = 22.0 秒对最坏的 22.4 秒，在刀口上。
- **实施**：T1–T16 由实现代理完成，主控逐 Task 复核并记台账。拆分与只改类型、调用写法的 Task（T2，T4 的构造，T9 的动作类型）先以既有测试留基线收据，结束时逐文件同名同数。主控复核时另要了一个修正提交：T11 的整批与它的一次进度跟进挂到命令宿主上（`3d93c6c9`）；并接受一处偏离：T8 的筛选用项目外壳已读进 store 的两份列表、不自己读（spec §2.6 在 T16 照实改写）。
- **门禁与审查**：T9 一次检查点（七项门禁与分支 CI，CI 的实测作 T17 估算的校准点）；T17 在 `93229770` 上的七项门禁与一次 fresh 只读整分支审查（[re-75](../../../reviews/re-75-M6P7b整体审核.md)），修复轮 F 另经一次限定复审；最终候选的七项门禁、三种引擎的页面配置，与推送前的 CI 时长估算（spec §2.14）。

## 2. 验收状态

A1–A14 满足（[spec §4](../specs/P7b-list-vocabulary.md#4-验收标准)已勾选），A15 待合入后由 P7c 的 T1 勾选。证据见 §4、§5：

| 验收 | 内容 | Task |
|---|---|---|
| A1 | 拆分：`issue-list-binding.ts` 487 → 286，拆出读 `issue-list-read.ts`（163 行）与目录 `issue-list-catalog.ts`（T2 时 82 行，T3 之后 148 行）；11 个回归文件一行不改、逐个同名同数（295 例）；值的导入无环；三件在 `project-address-route` 里 | T2 |
| A2 | 目录：`IssueCatalog.cycles/modules` 收窄为 `get`/`has` 的查找，取自规划 store 的 `getCycleById`、`getModuleById`，根不加 getter；只在按 `±cycle_name`、`±module_name` 排序时读，先读不在 store 的项目，叫不出就重读未归档的列表（已有也读）、仍叫不出再读归档，仍叫不出即大声失败；与 `usePlanningNames` 共用定次序的纯函数 `nextPlanningList`（hook 另有“已持则不读”）；门是列表自己的读，不是开关（R17）；plan 的九个变异与另两个都失败 | T3 |
| A3 | 筛选的词与按页收窄：`cycle_id`、`module_id` 含 `none`，成员列表不发两种分组；`CyclePageFilters`、`ModulePageFilters` 以 `never` 禁止本页的键，`update` 是属性，service 收 `PagePreferencesService<NoInfer<F>>`；类型测试含转换配错、service 配错、非字面量的值与根的推断写法，去掉 `never`、改回方法的收据（`t4-type-never` 8 个、`t4-type-property` 4 个指令成为未用） | T4 |
| A4 | 分组与泳道：列的次序（未归档的照 nerve 的次序含已结束的、回答里的已归档的以名字与说明、叫不出的“—”、“无”最后）；泳道并入两级回答的组；组头与泳道头的名字经 `usePlanningNames`（列表、看板与泳道两层各自调用）；拒收的列一个字段 `refusal?: GroupJoinRefusal`（开关关闭先、已归档次之、已结束最后） | T5、T7、T9 |
| A5 | 按页的常量与开关：`issues` 有两词的筛选、分组、子分组，`cycle_issues`、`module_issues` 由 `withoutOwn` 推出（两份副本删去）；本页的词漏出时两页页头的测试都失败（迭代页 `t6-mut-cycle-page-keeps-cycle`，模块页 `f2-mut-module-page-keeps-grouping`）；开关关闭时不给未选的、已选的留着，存下的本页分组也列出并勾着（F1）；`displayPropertyOn` 由 `PLANNING_WORDS` 与 `listWordOn` 取代、不留别名 | T6、F1、F2 |
| A6 | 分组里的新建：共用的 `useGroupQuickAddPrefill`（默认状态、列的、看板格子再加泳道的 payload），`quick-add/root.tsx` 的 `QuickAddPrefill` 与提交的白名单加两键，两份 `prePopulateQuickAddData` 删去，基线 19 个文件 182 例同名同数；拒收的列没有“+”、快速新建（格子另看泳道）与“添加已有” | T7、T12 |
| A7 | 筛选的界面：`@nerve/utils` 两个工厂；新 hook `use-planning-filter-configs.tsx`（135 行），`use-work-item-filters-config.tsx` 366 行；选项“无”、项目未归档的（含已结束）、持有的已归档值以“名字，已归档”；开关关闭时不列入“加筛选”、已应用的条件可移除 | T8 |
| A8 | 拖拽：两词可拖；`dropIssue` 收 `(held) => IssueChange`，在拖拽队列轮到时以 `changeBody` 得出请求体再设 `sort_order`；拖到“无”只移出来源，已归档与不认识的留着；类型测试 `drop-issue.types.test.ts`；移进拒收的列不发请求、以警告说那一句；基线 17 个文件 187 例同名同数 | T9 |
| A9 | 表头的排序：两列两向的名字排序（查询、行的次序、`aria-sort`）；存下的名字排序照原样显示与发出；死分支删去，三个表头文件与 `f1646370^` 逐字节相同 | T10 |
| A10 | “添加已有”：三个入口都经命令宿主，弹窗活过空态的卸下与列表的核验；排除已在本页的、迭代页的选项显示当前的迭代、全被排除时说明、规划页 `limit: 100`；每项一个 `changeIssue`、至多 2 个在途、一批一次进度重读（中途关掉弹窗也读一次）、逐项结果、失败的留着；分组头只加入本页、以“{添加已有}: {组}”为名 | T11、T11 的修正、T12、T15 |
| A11 | 日历：规划页每项一个 `{target_date, ...加入本页}`、隐藏已在本页的；项目页恰为 `{target_date}`，I4 一行不改 | T13、T15 |
| A12 | 页面故事：G15 (page) 与 G4、G11 (page) 在三种引擎，含 M3 交接 §3 的弹窗各项；两个新文件本机 chromium 合计 6.9 秒（≤ 14 秒） | T14、T15 |
| A13 | 守卫：`project-address-route`（各 Task 的新件与改写的件）、swr 限制（没有新的直接读者，`.oxlintrc.json` 不改）、告警上限 44、knip、关键词（phase `M6/P7`，78 条规则、7 个例外）、i18n（新键 `common.filter_options.fields.cycle_id`、`module_id`、`cycle.found_already_in`、`module.found_already_in`，读的键 1491 → 1498）、`root.store.test.ts` 21 个 store（根 store 未改）；根 320 → 319、写入单元 302（[M6-M8-08](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-08)，`t17-record-line-counts`） | T1、各 Task、T17 |
| A14 | 文档：P7a 的回填（阶段记录 §7、spec A12、设计 §14、M6-M8-01 的数据点）与 e2e 余量的重算；交接回执（M4 的 M6-02，M3 §1、§3 的 `existing-issues-list-modal.tsx`）；差异清单、前端改动清单、设计 §8/§12/§14、M8 交接的相关条目；re-75 的记录 | T1、T16、F1、F3、T17 |

## 3. 局部裁定

台账中 109 条 `Ruling:`（T1–T16 与 T11 的修正 99 条、修复轮 F 4 条、主控 6 条）与主控在复核时的裁定，均为局部取舍，没有架构待决项，也没有需要负责人决定的问题。要点：

- **列表绑定的拆分**（R16；T2）：拆为读 `helpers/issue-list-read.ts`（`IssueListScope`、`ReadRequest`、`scopeRole`、`allowsRole` 与页的读 `readPage`）、目录 `helpers/issue-list-catalog.ts`（`listCatalog`、`prepareListCatalog`）与绑定（生命周期、取数、确认与核验）。绑定对外的成员不变：`prepareCatalog` 仍是公开方法、委托目录，写入单元与测试的 `spyOn(list, "prepareCatalog")` 照旧；读到绑定只有类型导入，值的导入无环。“目录之后再核对读的凭据”原来没有测试钉住（`t2-mut-read-current` 全套存活），在读这一件的新测试里补一例，RED 由同一变异给出（P7a T2、T4 的先例）。
- **目录**（R17，re-74 F2、F8、F9；T3）：定次序的纯函数 `nextPlanningList` 在 `lib/planning-directory.ts`，只定次序，“读过”由调用者给：`usePlanningNames` 以 store 已持为读过（已持的守卫留在 hook），目录以本次准备自己的读为读过（已持也读）。按域的表 `PLANNING_NAMES` 在目录件里（以 `Map` 免得取到原型的键），不共用 hook 的 SWR 表（store 不依赖 hooks）；查找只一处 `planningLookups`；先读项目放在状态、标签那一块；被超过的读不加机制（主控对 F9 的裁定）。`use-planning-names.test.tsx` 追加一例钉住“两份都已持、仍叫不出时不再读”，既有的行不改。
- **词表与按页收窄**（R7、R8、R18、R22；T4、T6）：`groups` 的两项提前到 T4（成员列表排除两种分组才可测），显示一侧的词仍在 T5。页类型与转换在 `lib/issue-filters.ts`；基类 `PagePreferencesFilter<F>` 的 service 参数为 `PagePreferencesService<NoInfer<F>>`——不写 `NoInfer` 时根的推断写法会把 F 推成生成的类型、放过配错（探针证实），类型测试另加“按根的写法配错”。P6b `module/page-preferences.test.ts` 的一条按 R7 改写（别的域的键可在本页筛选，契约变化，主控接受）。`withoutOwn` 只在已有的分组键上去掉本页的词（不给日历、表格添空的分组节）；路由的项目的读法收进新 hook `useRouteListWords` 供三个面板，表格的列直接调 `listWordOn`（没有路由项目时不给两列，P7a R11）；P6b 的三条常量测试扩为两页、改标题，断言全留。
- **分组的列、泳道与拖拽**（R3、R4、R19；T5、T9）：显示词的 RED 另立（查询一侧 T4 已绿）。列的字段 `archived?` 与 `refusal?: GroupJoinRefusal`（nerve 的四个问题码的字面量联合，生成的客户端没有它们的枚举）；拒收的次序照 nerve 加入时的检查（开关 → 已归档 → 已结束），“无”与叫不出的“—”不拒，由 nerve 的 409/422 说原因。列在 `planningColumns` 一张两域的表里产生；宿主经 `usePlanningGroupNames` 调 `usePlanningNames`（别的分组以空的 id 调），测试按宿主分开、泳道两层各自核；三个组头只加 `archived` 一个 prop，照 P7a 的先例不加入 `project-address-route`。放下的拒收在两个分组件的 `onDrop` 里判（钩子不认识列，Plane 也在这两处），提示经新的 `problemMessageKey` 与 `useJoinRefusalWarning`（警告，标题读已有的 `common.warning`）；`dropIssue` 的形参改名 `makeChange`；共用替身 `fake-toast.ts` 加 `WARNING`。
- **分组里的新建**（R12，re-74 F3；T7）：共用的预填是 hook `useGroupQuickAddPrefill`（读路由的项目与状态），与 `QuickAddPrefill` 同在 `quick-add/root.tsx`。根的白名单对 `module_ids` 只在非空时放行（回归的 `root.pending` 以整个工作项为预填，断言不能改；新建时空集与不给相同）；`KanBan` 自己由 `sub_group_id` 投影出所在泳道的列（回归的 `group-consumers.test` 直接渲染它）；看板的格子在列或泳道拒收时都不给快速新建。“根丢掉两键”是 `t7-mut-quick-add-key` 的等价变异（失败的恰是同样的 7 例），只作证据；与今天的请求体逐项对照，别的分组相同。
- **筛选的界面**（R7、R8；T8，主控接受一处偏离）：两份未归档的列表不自己读，用项目外壳以同样的键已读进 store 的（同状态、标签的先例）——照 spec 每次挂载都读会改动 M4 list-data 与 P7a planning-create 等 26 条回归的请求次数，代价是外壳答到之前两个筛选暂不出现；spec §2.6 在 T16 照实改写。已持的值取筛选实例当下的条件（不取只是存下的 `initialWorkItemFilters`）；迭代的选项按状态的图标、模块一律 `ModuleOutline`、“无”不带图标；两个工厂照 Plane 改写、保留 Plane 的版权头；开关关闭的测试补强之后 `t8-mut-filter-ignores-switch` 才失败（最初的收据存活，`-2`、`-3`、`-final` 失败）。
- **表头的排序**（R9；T10）：新三例放在 `planning-columns.test.tsx`（帮手 `table()` 加一个可省的参数，断言不改）；本地存的显示词（个人页、归档页）不加两个名字排序（这两种页没有规划的词）；死分支照 P7a 之前的样子还原；`no-keys` 变异不能编译，改以 `order-unknown`。
- **“添加已有”与命令宿主**（R2、R5、R6、R11、R20、R21；T11、T12，主控不接受一处）：弹窗与请求的类型拆为 `add-existing-dialog.tsx`（宿主不逼近上限，296 行）；`settleEach` 与 `WRITES_PER_PROJECT = 2` 在 `lib/settle-each.ts`（以递归代替循环里的 `await`，告警不增）。T11 起初让整批的进度跟进随弹窗的寿命（批未完而关掉弹窗即不重读），主控不接受：照 P6b T10 的先例，跟进挂在本页期间一直在的东西上，`3d93c6c9` 改为整批与它的一次 `followProgress` 由命令宿主的动作发。“全被排除”经弹窗新的 `exclusion` prop，只在 nerve 找到的每一项都被本页排除时说（日历自己的隐藏不算），T11 的本页排除随之从 `shouldHideIssue` 移来；说明件与提示的键在 `PLANNING_PAGE_LAYOUTS` 每域的 `addExisting`（没有按域的分支）。组头的入口是新件 `GroupAddExisting`，与“+”同一道门、另看 `joinable`，按钮名复用 `issue.add.existing`；新键 `cycle.found_already_in`、`module.found_already_in`；空态的 `cta_secondary` 早有、不加键。
- **日历**（R6；T13）：日子取本页的 store 类型，照日历既有的写法；入口与日子的“新建”同门，不另看 `joinable`（开关关闭时整页是关闭态），其余由 nerve 的 409 说原因；删去旧的 `[...new Set(ids)]`（弹窗的选择已去重）；回归的 `quick-add-issue-actions.test.tsx` 只按契约包在宿主之下，断言不改、对旧写法中性。
- **e2e**（T14、T15）：`dayFromToday` 收进 `fixtures/cycle-pages.ts`（两个既有故事只改导入，三种引擎通过）；偏好先经接口存下，页面上只经界面改分组、筛选、版式与表头；其余见 §5 的教训。
- **修复轮 F**：F1 的规则在两个面板各写一行，不抽共用函数（两个面板的选项来源与互藏的规则不同，各自可变异）；规则照字面也适用于“无”：经接口存下 `group_by: null` 的看板从此列出 “None” 并勾着（主控接受：照 R8 如实显示存下的，网页自己的修改仍规范为 `state_id`）；文档只改 plane-diff 的 P6b 一行与 frontend-changes 的词表一行；F2 的变异取“模块页推出时去掉迭代而不是模块”（一处、能编译）。re-75 的 M3 主控核过不改，M4 记为观察。
- **实现比 spec §2.7、§2.9 的正文窄或细的三处**（T16 按 Ruling 不回改正文、写进两份清单，re-75 列为观察）：
  1. **T9 的拒收只看改变的那一维**：§2.7 写“目标的列（或看板的格子所在的泳道）拒收时，放下不发请求”。实现只在这次放下从别的列（泳道）移进拒收的列（泳道）时不发、以警告说明；在拒收的列（泳道）之内的移动只改次序，照发 `{sort_order}`、不拒——nerve 只在加入时检查，`planning.cycle_completed` 那一句也说已结束的迭代只能改次序；Plane 连列内的重排也拒绝。所以 §2.7 的“放下不发请求”不包括列内的重排。登记在前端改动清单“分组的列、拖拽与分组里的新建（M6/P7b）”一行与差异清单“拖到迭代、模块的组（M6/P7b）”一行。
  2. **T11 的整批在命令宿主里跑**：§2.9 写动作“整批经 `usePlanningPageProgressFollow` 包一次”、“宿主以请求的 store 类型的动作提交”。实现进一步定下：整批与它唯一的一次 `followProgress` 由命令宿主的动作发，弹窗只发起并在开着时显示逐项结果，中途关掉弹窗时三个写都落、进度仍重读一次（主控的裁定，`3d93c6c9`；代价是切换版式时进行中的批的跟进照旧丢失，与版式根自己的写同）。登记在前端改动清单“‘添加已有’与命令宿主（M6/P7b）”一行与差异清单“‘添加已有’（M6/P7b）”一行。
  3. **T12 的排除是弹窗的 `exclusion` prop**：§2.9 写“隐藏已在本页的（客户端，`shouldHideIssue`）”。实现以弹窗新的 `exclusion` prop 排除本页的（它也给“全被排除”的那一句），`shouldHideIssue` 只留调用者自己的隐藏规则（日历的“开始晚于那天”），所以日历自己的隐藏不触发那一句，混合的情形照旧说“没有找到”。登记在前端改动清单“‘添加已有’与命令宿主（M6/P7b）”一行。
- 其余见台账。

## 4. 实施记录

变异数为能编译的产品变异（e2e 的为能编译、能构建的），守卫（关键词、swr）的变异另记在台账；探针是 plan 之外为钉住裁定而做的变异；例数为网页包（`web`）的文件数/用例数之差。

| Task | 提交 | 内容 |
|---|---|---|
| spec/plan 与预检 | `cf92e3d5`、`4ec4e9d3` | spec 275 行、plan 303 行（16 个 Task；R1–R22、A1–A15）；预检 re-74 的修正（F1–F4 Important、F5–F9 Minor）与 re-74 的记录，计划为 17 个 Task |
| T1 P7a 的回填与余量 | `9a64480a` | P7a 阶段记录 §7、spec A12、plan、设计抬头与 §14、前端改动清单、M6-M8-01 新数据点；原件逐项核对（收据 `t1-*`）；P7b 与 P7c 合计的余量 22.4–27.1 秒，不小于 22.0 秒；只改文档 6 个文件 |
| T2 拆开列表的绑定 | `cd335acf` | 绑定 487 → 286、读 163、目录 82；11 个回归文件一行不改、逐个同名同数（295 例）；+1/1（存活的变异补测）；变异 3 个 |
| T3 目录 | `3ee469c7` | 规划 store 的查找、只为名字排序读、先取项目、叫不出即重读；基线 5 个文件 110 → 134 例（`planningDirectoryIsUnavailableUntilP7` 的四例换为两域各 12 例，契约变化）；+0/24；变异 11 个；目录 148 行 |
| T4 筛选的词与按页收窄 | `05da3a3e` | 两词含“无”、成员列表不发两种分组、两页的页类型与转换、属性写法的 `update`、`NoInfer`；+1/7，P6b `module/page-preferences.test.ts` 一条按 R7 改写（契约变化）；变异 4 个、探针 1 个，类型收据 2 个；根 320 → 319 |
| T5 分组的列与泳道 | `89e9b8f8` | 分组词、列的 `archived` 与 `refusal`、列的次序、组头与泳道头的名字；+2/11，四个回归文件只补 mock 的接口；变异 6 个、探针 5 个；`group-columns.tsx` 174 → 325 |
| T6 按页的常量与开关 | `c7ee9165` | `withoutOwn`（`filter.ts` 389 → 305）、`PLANNING_WORDS` 与 `listWordOn`、`useRouteListWords`；+2/28，常量包 2/13 → 2/22（P6b 三例扩为两页、改标题）；变异 5 个、探针 2 个 |
| T7 分组里的新建 | `0dbedc48` | `useGroupQuickAddPrefill`、根的白名单、拒收的列没有“+”与快速新建；基线 19 个文件 182 例同名同数；+2/18；变异 3 个、探针 6 个 |
| T8 筛选的界面 | `ff72a952` | 两个工厂、`use-planning-filter-configs.tsx`（135 行）、新键两个；基线 33 个文件 359 例同名同数；+1/10，`filter-config.test.tsx` 只补 5 行 mock；变异 5 个、探针 5 个 |
| T9 拖拽（检查点） | `b3279330` | 两词可拖、变化在轮到时算、拒收不发并警告；基线 17 个文件 187 例同名同数；+2/25；变异 6 个、探针 3 个，类型收据 1 个 |
| T10 表头的排序 | `7fd3d1ca` | 四个名字排序、死分支删去；`planning-columns.test.tsx:130` 照 R9 改写（契约变化）；+0/7；变异 3 个、探针 1 个（`no-keys` 不能编译、不计） |
| T11 “添加已有”的写、宿主与空态 | `d2b390e0` | `settle-each.ts`、`addIssuesToView`、宿主的请求、`add-existing-dialog.tsx`、空态的入口；回归的 42 例与 131 例同名同数；+3/21；变异 9 个 |
| T11 的修正 | `3d93c6c9` | 整批与一次进度跟进由命令宿主的动作发（主控的裁定）；+0/1；变异 1 个 |
| T12 当前的迭代、全被排除与组头 | `8653ff3a` | `current-cycle-caption.tsx`、`exclusion` 与提示、`group-add-existing.tsx`、新键两个；基线 18 个文件 182 例同名同数；+2/18；变异 10 个 |
| T13 日历 | `f6ce345c` | 规划页一个 `{target_date, ...加入本页}`、项目页照旧；基线 10 个文件 83 例同名同数（一个回归文件只包在宿主之下）；+1/4；变异 6 个 |
| T14 页面故事 G15 | `565c5691` | `g15-work-item-lists-page.spec.ts`（208 行），`dayFromToday` 移进夹具；期望错误的 RED 1 个，e2e 变异 5 个都失败 |
| T15 页面故事 G4、G11 | `55a225c8` | `g4-add-existing-page.spec.ts`（246 行，两条故事）；期望错误的 RED 2 个，e2e 变异 8 个都失败 |
| T16 文档 | `93229770` | 差异清单扩 3 行、新 8 行；前端改动清单改 3 行、新 6 行；设计 §8、§12、§14；spec §2.6、§2.12、§5；M3、M4 交接的回执；M6-M8-01、-08、-13 补记 |
| F1 | `e0fe04ea` | 存下的本页分组在两个面板里列出并勾着（re-75 M1），`useGroupByOptions` 删去；+0/2；变异 2 个 |
| F2 | 无提交 | 模块页页头的变异收据 `f2-mut-module-page-keeps-grouping`（re-75 M2） |
| F3 | `16c52f41` | re-75 的记录、spec §5 的修复轮、spec 与 plan 的状态、设计 §14 |
| T17 记录 | 本记录所在提交 | re-75 §4（修复轮的限定复审）、本记录、spec §4 的勾选与 §5、spec 与 plan 的状态、设计 §14、前端改动清单、M8 交接（M6-M8-01、-03、-07、-08 补记，新 M6-M8-14） |

网页包从基线的 449 个文件 5132 例到 466 个文件 5309 例（T13 的全量收据 466/5307 加 F1 的两例）；常量包 2/13 → 2/22，其余包同基线。

## 5. 门禁与 CI

- **检查点**（T9 之后，`b3279330`）：
  - `t9-gate` 七项一次全过、没有重跑：`make test` 本机 902 秒通过（没有 M6-M8-05 的签名），`make build && make e2e` 474 过（3.7 分钟，没有 M6-M8-03、-07 的签名）；本机 chromium 的 `stories/planning/` 各条合计 256.2 秒（P7a 的检查点 249.7 秒，9 个工作进程，只作对照：既有故事没有因新的选项与项目页的新词失败）。三种引擎的页面配置 `t9-m6-pages` 117 例通过（2.2 分钟）。
  - 分支 CI 43629579359 attempt 1 全绿：server (1) 132553001739、web 132553002009、server (0) 132553002012、e2e 132558145042（e2e 作业 19:17:14Z–19:37:39Z，20 分 25 秒）。这时还没有本段的页面故事，比基线 23 分 43 秒少 3 分 18 秒，作 T17 估算的校准点（T16 记入 [M6-M8-01](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-01)）。检查点推送之后的提交留在本地，直到 T17。
- **三种引擎的页面配置**（`playwright.m6-pages.config.ts`，新文件照它的 `testMatch` 自然进入，配置不改）：
  - T14：G15 三种引擎 3 例（chromium 3.5、firefox 5.0、webkit 4.2 秒，3 个 worker）、`--repeat-each=3` 9 例；改了导入的两个既有故事三种引擎 15 例。
  - T15：G4、G11 三种引擎 6 例（G4 chromium 2.4、firefox 3.3、webkit 3.1 秒，G11 1.0、1.3、1.4 秒，本文件 3.4、4.6、4.5 秒）、`--repeat-each=3` 18 例，整套 126 例（2.2 分钟）。
  - 两个新文件 chromium 合计 6.9 秒（预算 14 秒），没有间歇失败、没有重跑。
- **T17 的门禁**（`93229770`，与整分支审查并行）：`t17-gate` 的 `make gen-check`、`make lint-go`、`make test`（本机 905 秒通过）、`make lint-web`、`make knip`、`make test-web` 通过；`make build && make e2e` 477 条中 474 过、3 条失败（3.7 分钟）：
  - F11 (page) [s3] 以 [M6-M8-07](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-07) 的签名（`net::ERR_ABORTED GET /api/v0/assets/<id>/content`）；
  - W12 以 [M6-M8-03](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-03) 的签名（`nerve workspaces reactivate-member` 连数据库时收到 `FATAL: sorry, too many clients already`，53300）；
  - 与 W12 同一时刻的 C8 command-url（`collaboration-interactions.spec.ts:556`，详情页 5 秒内没有标题的输入框）：它的附件被随后的运行覆盖、nerve 日志已不在，签名无从核实，记为“很可能是 M6-M8-03、未核实”（P5a 时同一行查实是 M6-M8-03）。
  - 只重跑这三个文件一次 42 条通过（`t17-gate-e2e-failed-rerun`，1.9 分钟；这次收据的 `inputs_unchanged` 为假，唯一变的输入是修复轮 F1 那时写的 vitest 测试，那次 e2e 没有重新构建、不读它）。
  - 三种引擎的页面配置（`t17-m6-pages`）在会话结束时（2026-11-14 约 22:03Z）被切断、没有跑完：日志里 99 条已过，最后两条（webkit 的 G4 (page)、G19 (page)）在切断的那一刻记为失败、没有打印详情，不作结论。2026-11-15 恢复时 Docker 守护进程没有运行，由主控启动 Docker Desktop（其他项目的容器随 Docker 自己的重启策略起来，主控没有动它们）。修复轮在这次门禁之后开始。
- **最终候选**：`f-final`（`16c52f418cdd75d76b6a26eb365b7eaf704ff46f`）——`make gen-check`、`make lint-go`、`make test`（本机 931 秒通过，没有 M6-M8-05 的签名）、`make lint-web`、`make knip`、`make test-web` 第一次就通过；`make build && make e2e` 477 条中 476 过、1 条失败（3.7 分钟）：W2 (API)（`w2-landing.spec.ts:40`）测试一侧的 pg 连接池与之后的 `pg_dump` 都是 “sorry, too many clients already”，即 M6-M8-03 的签名（`e2e/test-results` 存于台账 `f-final-test-results`）；照该条只重跑这一个文件一次 7 条通过（`f-final-e2e-w2-rerun`，19.6 秒）。三种引擎的页面配置 `f-final-m6-pages` 126 例通过（137 秒，没有 flaky）。全部收据的 `inputs_unchanged` 为真。e2e 现有 477 条（P7a 的 474 条加 G15、G4、G11）。
  - 修复轮的限定复审与这次门禁并行，结论见 §6。
  - 行数（[M6-M8-08](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-08)、E15，`t17-record-line-counts`，HEAD `16c52f41`）：根 `root.store.ts` 319、写入单元 `issue-writes.ts` 302、请求体 `issue-write-fields.ts` 98；列表的绑定 286、读 163、目录 148；`command-host.tsx` 296、`add-existing-dialog.tsx` 73；`use-work-item-filters-config.tsx` 366、`use-planning-filter-configs.tsx` 135；`group-columns.tsx` 325、`kanban/swimlanes.tsx` 347、`issue-filters.ts` 349、`use-issues-actions.tsx` 347、`existing-issues-list-modal.tsx` 313、常量的 `filter.ts` 305；F1 改的 `group-by.tsx` 75、`sub-group-by.tsx` 69、`utils.tsx` 171。都在约 400 行之内，前十六件与 T16 的 `t16-line-counts` 相同。
  - 本记录所在提交只改文档，另跑文档链接、关键词与 `make lint-web`（收据 `t17-record-*`），其余以准确 HEAD 的分支 CI 为准。
- **CI 的 e2e 时长估算**（推送前，E13、spec §2.14）：本段新的页面故事本机 chromium 6.9 秒（G15 3.5，G4 与 G11 3.4；修复轮没有改 e2e）。
  - **估算**：基线 1423 秒（P7a 的 main CI 43603772626）加 6.9 秒 × 1.67–2.25、两个 worker 摊开，约 23 分 49–51 秒。
  - **上界**：S = 1702.40–1708.15 秒（T1 的算法）加 6.9 秒 × 3.6–4.1，约 28 分 47–56 秒。
  - **校准点**：T9 检查点 43629579359 的 1225 秒加同一增量，约 20 分 31–33 秒；慢的 runner（×1.31，M6-M8-01）乘在估算上约 31 分 12–14 秒，这是 T1 的观察，不是 spec 的停下条件（主控已告知负责人：e2e 作业的结构迟早要他决定）。
  - **P7c 的余量**：(1800 − S)/f − 6.9 = 15.5–20.2 秒，不少于 P7c 的 8 秒。
  - 估算与上界都低于 30 分钟的报告线，所以推送；实测超过 30 分钟仍先停下报告负责人。实际的 e2e 作业时长见 §7。
- **测试与 e2e 的教训**（交 P7c）：
  - **在途的数目以页面自己的 Resource Timing 核，不装路由**（T15）：提交之前 `performance.clearResourceTimings()`（页面的缓冲至多 250 条，载入已用去许多），批完之后取各个 `PATCH` 的 `startTime` 与 `responseEnd` 算同时在途的最大数；`settleEach` 在拿到回答之后才起下一个，所以第三个的 `startTime` 不早于先答完的 `responseEnd`。无界的变异得 3。
  - **WebKit、Firefox 不装路由**（T14、T15）：路由在 WebKit 上会带出 modulepreload 的控制台告警（P7a 的教训）；请求以 `request` 事件记（`recordApi`、`listQueries`），偏好的修改与弹窗的读以 `answerTo` 等回答，请求体只在 chromium 以直通的路由读（`bodiesSent`）；“不发请求”以记下的请求里没有那个 `PATCH` 核，数目在 nerve 的回答之后从看板的列头读。
  - **“添加已有”的弹窗打开时焦点不在搜索框**（T15）：故事先以 `field.focus()` 把焦点放进搜索框，再以方向键选；这是 M4 的弹窗，不在 M3 交接 §3 的各项之内，主控交 P7c（照下拉框“打开即聚焦搜索”的写法）。
  - **门禁的 e2e 失败时先留下 `test-results`**（T17）：C8 的附件被随后的三种引擎运行覆盖，签名再也核不了；先把 `e2e/test-results` 拷进台账再跑下一个配置（`final.sh` 已这样做）。
  - **e2e 变异要落在被证的一步、各只一道守卫**（T14、T15）：每个产品变异单去一道守卫即让故事失败；嵌套只在注入要改两件时（组头“不带组的 payload”）；断言本身以期望错误的 RED 证实（G15 的模块一列 Z→A 的次序是 nerve 的，网页上没有能编译又改变它的产品变异，只能这样证；G4、G11 另以“每项各读一次进度”“组头带 `state_id`”两个核断言）；变异之后以干净的源重建，`bin/nerve` 才不带变异。
  - **世界的建法**（T14、T15）：nerve 以 `sort_order` 列出迭代与模块、新建的在前，故事按要列出的次序倒着建；要归档的先完成、先加入再归档；偏好先经接口存下，页面上只经界面改。
  - **e2e 的 lint 上限是 0**（T15）：按键的序列以 `reduce` 串起，不在循环里 `await`（`no-await-in-loop`）。
  - **时长**：台账里的 `durations.py` 从 JSON 报告取每个结果自己的时长、可按故事分；P7c 的 8 秒照此量。
- **`5d7adaaa` 的分支 CI 与本提交**：43696983860 attempt 1 的 web、server 两路通过，e2e 作业 132762846784 477 条中 1 条失败——C4 (page) relation（`detail-interactions.spec.ts:13`，`:54` 等不到 “Related target options”）。不是登记过的签名，没有重跑；trace 查实描述的编辑器在点击 “Relations 1” 的按下与抬起之间挂上、Relations 下移约 28 像素，click 没有落在按钮上（[M6-M8-13](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-13) 一类，P7b 没有改详情页）。本提交照主控的裁定改故事、不改产品：C4 与 C8/C12 的 relation 一支先等描述的编辑器再点（夹具新的 `descriptionEditorMounted`）；探针在旧故事上复现、新故事通过（`ci-probe-old-hold` 15 例全在同一步失败、`ci-probe-new` 15 例全过），两个文件默认配置 48 条（`ci-after-two-files`）与 chromium `--repeat-each=3` 134 条（`ci-repeat3`）通过，一次多约 0.8 秒；经过见 M6-M8-13 的“处理结果（M6/P7b 的分支 CI）”。
- **最终 HEAD 的分支 CI 与合入后的 main CI**：见 §7。

## 6. 整分支审查

fresh 只读审查（[re-75](../../../reviews/re-75-M6P7b整体审核.md)，`fable`，范围 `f42f9d64..93229770`，19 个提交、114 个文件，与 `93229770` 的本机门禁并行）逐项核对了：

- 写入（拖拽的请求体在轮到时算、拖到“无”只移出来源、`IssueChange` 的类型测试；拒收只看改变的那一维；“添加已有”每项一个 `changeIssue`、至多 2 个在途、一批一次进度读、整批在宿主里；日历一个请求；快速新建的预填与白名单）；
- 读与会话分代（目录的查找、只为名字排序读、先取项目、叫不出即重读、门是列表自己的读；筛选用外壳读进 store 的列表；名字都经 `usePlanningNames`，没有一个新测试替换它；没有过时的写）；
- 词表（含“无”、成员列表排除两种分组、R18 对非字面量也成立、`withoutOwn`、列与拒收的次序、表头的排序；T16 写的 P6b 差异那一句核实为真）；
- 界面与可访问性（弹窗的名字与背后 inert、方向键、移除按钮的名字、说明的读法、`role="status"` 的提示、组头按钮的名字与门、只读）；
- 设计质量（改到的产品文件都在约 400 行之内、没有值的循环导入、死分支与两份副本删去、`GroupJoinRefusal` 的字面量联合有理由、守卫）；
- 测试（17 个新测试文件是具体的，三处既有断言的改写都是 spec 要的，plan 点名的变异都有失败的收据，故事不重新载入，6.9 秒）；
- 文档（差异清单、前端改动清单、设计、交接与 `wc -l` 和台账一致；Plane 的引用抽查 7 处）。

结论 Ready to merge：没有 Critical、Important，四处 Minor（M1 存着的本页分组在面板里不勾也不列；M2 spec §2.5 要的“两个页头测试都失败”只有迭代一侧的收据；M3 看板按 id 找泳道；M4 `base-issues.store.test.ts` 1640 行）与六条观察，没有须负责人决定的事。主控仍以修复轮 F 处理 M1、M2（同 P6b、P7a 的先例）：F1 照 re-75 的选项 (a)，一项列出当且仅当它是存下的那一项、或在本页的选项里且被给出；F2 补模块页的变异收据（没有提交）；F3 写 re-75 的记录。M3 主控核过不改（泳道与查找是同一函数、同一范围，离开项目的成员没有泳道），M4 是测试文件、记为观察。六条观察里，三处比 spec 窄或细的写法在 §3 逐一列出，弹窗的焦点交 P7c，其余不需动作。

修复轮的限定复审（`fable`，范围 `93229770..16c52f41`，只读、只跑了文档链接的检查，与最终候选的七项门禁并行）结论 Ready to merge，没有 Critical、Important、Minor：两个面板的列出规则恰为“存下的，或在本页的选项里且被给出”，次序照 `ISSUE_GROUP_BY_OPTIONS`，看板的互藏不变；`useGroupByOptions` 连同导入删净、不留别名，没有别的产品改动；个人页、归档页与子工作项不受影响；副作用合意（经接口存下 `state_group` 的项目列表从此勾着 “State groups”，存下 `null` 的看板勾着 “None”、板为一列，合 R8）；M3 不改的理由成立；RED 与三个变异的收据、re-75 的记录与各处文档都核过，唯一的断链是当时还没写的本记录。三条观察不需本段动作，见 re-75 §4：F2 的页头失败落在该例开头对本页常量的断言（面板一级由 F1 的两例钉住）；存下 `null` 的看板点勾着的 “None” 会按状态分组（早于 F1，是用户自己的写）；经接口存下 `group_by: target_date` 的列表，nerve 接受而网页的面板与列都不认得（早于 P7b）——主控按只读追踪写成 M8 的新条目 [M6-M8-14](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-14)：项目页、迭代页、模块页的列表与看板有工作项时，`getGroupByColumns` 在渲染时抛错，正文落到版式的 “Something went wrong”，页头的分组面板没有一项勾着，选任一项即恢复。

## 7. 集成、归档与接续

由 P7c 的 T1 按归档里的台账（Task 17 的末尾）、`manifest.json`、`cleanup.log` 与 `ci-evidence/` 回填；作业号、时刻、attempt 与结论以 GitHub 公开的 jobs 接口（只读）核对（收据 `t1-ci-jobs`、`t1-ci-jobs-parse`），e2e 的条数与逐条的时长以作业日志核对（分支 CI 的取归档里的 `ci-evidence/ci-43702923005/e2e.log`，main CI 的经接口取得，收据 `t1-e2e-job-logs-5`、`t1-e2e-per-test`），双父与树以 `git rev-parse` 核对（收据 `t1-merge-tree`）；spec 的 A15 已勾选。

- **最终 HEAD 的分支 CI**：`4fe308ceb535d38951a3d3831737085ca82a2738`（抬头与 §5 所说的“本提交”）的 43702923005 attempt 1 全绿（web 132777749603、server (0) 132777749874、server (1) 132777749893、e2e 132781366495；e2e 作业 10:26:46Z–10:54:36Z，27 分 50 秒 = 1670 秒，477 过、故事 25.7 分钟），是至今最慢的 e2e 作业（此前最慢的是 43549053092 的 27 分 41 秒）。高于 §5 的估算约 23 分 49–51 秒（约 1.17 倍），低于上界约 28 分 47–56 秒（约 0.96–0.97 倍）与 30 分钟的报告线；是校准点约 20 分 31–33 秒的约 1.36 倍。本段的三个新页面故事在这次 CI 上合计 21.5 秒（G15 11.8、G4 6.3、G11 3.4 秒），是本机 6.9 秒的约 3.1 倍，在上界所取的 f = 3.6–4.1 之下。前一次 `5d7adaaa` 的 43696983860（e2e 26 分 11 秒，含 C4 一次 30 秒的超时）见 §5。
- **合入与 main CI**：noff `d052213f82aa4fba234f5684722335e1a5cbb5be`，双父 `f42f9d64`、`4fe308ce`，树与分支相同（`9137798a`）。main CI 43706438924 attempt 1 全绿，没有重跑（web 132788437685、server (0) 132788437824、server (1) 132788437917、e2e 132792042944；e2e 作业 11:10:31Z–11:34:13Z，23 分 42 秒 = 1422 秒，477 过、故事 21.7 分钟；三个新页面故事合计 17.3 秒）。同一棵树、同样的 477 条，分支 CI 的作业慢 1.17 倍，逐条合计慢 1.20 倍：接口故事 (API) 持平（105 条，0.999），页面故事 (page) 慢 22%（221 条）、其余慢 19%（151 条）。与 P7a §7 所记同形（浏览器的故事比接口的慢得多），判为 runner 的速度差异。这次的 23 分 42 秒是 P7c 的 e2e 时长基线（[P7c spec](../specs/P7c-palette-mobile.md) §2.12）。
- **归档**（主目录，P7c 开工时以 `archive.py verify` 复核，收据 `t1-p7b-archive-verify`）：`.superpowers/sdd/M6-P7b-integration-20261115`：3964 个文件、920,634,424 字节、5216 个成员、未知 0、清单 sha256 `7a17a397fa181c56c3d46425d07783d2b689ec482e8f6c0f2cb041eca582799c`，清单的 `head` 为 `4fe308ce`（台账、调研笔记、收据、变异与探针、CI 证据 `ci-evidence/`（两次分支 CI 的 e2e 日志与 C4 的产物、合入信息、等待器）、e2e 的 `test-results` 与 `playwright-report`、各包的 `.turbo`）。
- **清理**：`cleanup.log`（2026-11-15T11:35:17Z）记录 `codex/m6-p7b` 本地、远端与 GitHub API 都在 `4fe308ce`、已含于 main，main 与远端 main（以 GitHub API 核对）都在 `d052213f`；worktree `m6-p7b` 与本地分支已删除，远端分支删除之后 GitHub API 对它答 404。

**接续**：P7c 命令面板、移动端页头与遗留进行中（[P7c spec](../specs/P7c-palette-mobile.md)、[P7c plan](../plans/P7c-palette-mobile.md)，预检 re-76 进行中），基线 main `d052213f`：先拆导航的命令，规划入口一条规则与菜单的空文案，搜索的迭代、模块结果，工作项的“改迭代”“改模块”，快捷操作的“恢复”与归档页的行菜单（A17、A18），共用的移动端页头（项目页改用，X2）与迭代页、模块页的接上，分组时页头的数目、没有读者的键、`fake-planning-page.tsx` 的 `opened` 与“添加已有”打开即聚焦（A19–A21），设计 §8 的四个视图文件（X1），G19 的工作项一侧的页面故事。它的 T1 以本节的实测重算 e2e 的余量（P7c spec §2.12）：S = 1661 + 19.3 × f = 1730.5–1740.1 秒（f = 3.6–4.1，大于 1670、1422 两次实测），余量 (1800 − S) / f = 14.6–19.3 秒（C4、C8 关系一支的等待约 0.9 秒算作加进来的故事时长；不计它即 §5 的 15.5–20.2 秒），本段的新文件 ≤ 8 秒；X = 8 秒时上界约 29 分 19–33 秒，以 1422 秒为基线的估算约 23 分 49–51 秒。之后是 P8（归档页的迭代、模块词表 R10、[M6-M8-13](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-13) 等，spec §5）与 M6 的收尾；M8 另接 [M6-M8-14](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-14)。
