# M6 / P7b 整体审核

> 2026-11-15：Ready to merge；主控仍以修复轮 F 处理两处 Minor（M1、M2，同 P6b、P7a 的先例），另两处不改（M3、M4），修复轮 F 经限定复审 Ready to merge（§4）。
> - **目标**：分支 `codex/m6-p7b` 相对 main `f42f9d6457ccc9540a687019f3530e253fe0f7b5`（P7a 的 noff）的全部修改（`f42f9d64..93229770`，`cf92e3d5` 至 `93229770` 的 19 个提交，含 spec 与 plan、按预检的修正与 T1–T16，114 个文件）；修复轮 F（`e0fe04ea`、`16c52f41` 两个提交，F1–F3；F2 只有收据，没有提交）另经一次限定复审（§4）。
> - **依据**：[P7b spec](../v0/M6-cycles-modules/specs/P7b-list-vocabulary.md)、[P7b plan](../v0/M6-cycles-modules/plans/P7b-list-vocabulary.md)、[预检 re-74](re-74-M6P7b预检.md)、[M6 设计](../v0/M6-cycles-modules/M6-design.md)、[总体设计 §7.2、§7.7](../v0/v0-design.md#77-stores-按会话分代长期有效m2-起)、[硬规则](../v0/claude_to_gpt/03-rules-and-pitfalls.md)。
> - **阶段记录**：[P7b 阶段记录](../v0/M6-cycles-modules/reviews/P7b-list-vocabulary-review.md)。

## 1. 方法

一次 fresh 只读审查（`fable`，台账 Task 17），范围 `f42f9d64..93229770`（spec 与 plan、按预检的修正与 T1–T16 的全部提交），不改动文件与 git 状态。它与 `93229770` 的本机门禁（`t17-gate`）并行，没有跑构建与测试，引用的收据是台账的。它读了整份差异、spec、plan、re-74、主控的 E1–E15 与台账，对照服务端的写闸与偏好的校验（`server/configs/config.yaml` 的 `write_gate.per_project`、`list_settings.go:49`）读写入、目录、词表与每个新的面，承重用例的新测试与两个 e2e 故事；文档引用的 Plane 源码在主目录的只读 `plane/` 抽查 7 处。

原始报告没有另存文件；结论、核对过的性质与各条发现照它转述，逐条内容与处理见 §2、§3。

## 2. 核对过的性质

- **写入**：拖拽的请求体在写轮到时算（`base-issues.store.ts:92-94` 以 `changeBody(makeChange(held), held.module_ids)` 再加 `sort_order`；`group-drop.ts:149-184`：迭代 `{cycle_id: 目标|null}`，模块 `{remove: [来源], add: [目标]}`，拖到“无”是 `remove: [来源], add: []`，从“无”拖出是 `add: [目标]`；`modulesAfter`（`planning-directory.ts:67-72`）留着已归档与不认识的 id；`IssueChange.module_ids?: never` 有类型测试与收据 `t9-type-issue-update-final`）。拒收的列：`dropRefusal`（`group-drop.ts:105-112`）只看拖拽改变的那一维；`list-group.tsx:176-177`、`kanban-group.tsx:154-155` 经 `useJoinRefusalWarning` → `problemMessageKey` 提示，与 nerve 的拒绝同一行 `PROBLEM_MESSAGES`（`error-messages.ts:129-131`）；列内重排只发 `{sort_order}`（T9 的裁定，收据 `t9-probe-refuses-reorder`；Plane 拒绝它，已登记差异，Plane 的 `list-group.tsx:220-227` 核过）。“添加已有”：`use-planning-page-issue-actions.ts:132-143` 是 `followProgress(() => settleEach(ids, WRITES_PER_PROJECT, …))`，每项一个 `changeIssue`，至多 2 个在途（`settle-each.ts:18-44`，等于 `server/configs/config.yaml` 的 `write_gate.per_project: 2`），一批一次进度读；结果恰覆盖给出的 id（`existing-issues-list-modal.tsx:155-159`，失败的留在选择里）；整批在宿主里跑（`command-host.tsx:139`；hook 的次序稳定，`use-issues-actions.tsx:67-72` 五个子 hook 都调）。日历：`quick-add-issue-actions.tsx:60-65`；规划页以 `{...extra, ...join}` 合成一个变化，项目页只发 `extra`（`use-issues-actions.tsx:155-164`，I4 未动）。快速新建：`useGroupQuickAddPrefill`（`quick-add/root.tsx:55-61`）是默认状态 ⊕ 列的值 ⊕ 泳道的值；`QuickAddPrefill` 与提交的白名单带 `cycle_id`、`module_ids`（`:45-46`、`:217-219`）。
- **读与会话分代**：目录（`issue-list-catalog.ts`）的查找是 `getCycleById`、`getModuleById`（`:13-18`）；只在按 `±cycle_name`、`±module_name` 排序时读（`:98`），按项目，先 `fetchProject`（`:105`），先未归档、再归档，经 `nextPlanningList`（已持的也重读，`:74-81`），每次读都有 `assertActive`，仍叫不出名字就大声失败（`:147`）；门是列表自己的读，不是开关（`:97`，R17）；被超过的 `Reconciled.fetch` → `markUnavailable` 的路径不变；`readPage` 在目录之后再查 `isReadCurrent`（`issue-list-read.ts:123`）。筛选用包装层读进 store 的列表（`use-planning-filter-configs.tsx:98-104`，自己不经 SWR 读），已持的已归档值经 `usePlanningNames`（`:94-95`）；列与泳道的名字读（`group-columns.tsx:70-79`，在 `list/default.tsx:88`、`kanban/default.tsx:100`、`swimlanes.tsx:285-286` 调用）与当前迭代的说明（`current-cycle-caption.tsx:20`）同；17 个新测试文件没有一个替换 `usePlanningNames`。没有过时的写：`addIssuesToView` 没有工作区或页面时抛 `SessionChangedError`；每个 `changeIssue` 经写入单元的检查（`issue-writes.ts:75-86`）；弹窗的结果由 `current()` 把门；路由或权限变化时宿主的弹窗结束（`command-host.tsx:238-243`）。
- **词表**：筛选的词含 `none`（`issue-filters.ts:17-18`、`:110-113`），成员列表排除两个分组（`:299`）；R18 对不是字面量的值也成立（`issue-filters.ts:198-199` 以 `never` 的页类型；`page-preferences-filter.store.ts:33-36`、`:132-137` 的 `NoInfer` 与写成属性的 `update`，在 `strictFunctionTypes` 下是逆变的；`page-preferences-filter.types.test.ts:42-45`；收据 `t4-type-never` 8 个 TS2578、`t4-type-property` 4 个 TS2578）；运行时 `toCycleFilters` 去掉本页的键（`:202-205`，`t4-mut-keeps-own-key`）。`withoutOwn`（`constants/filter.ts:107-128`，两份副本删去，389 → 305 行）。列的次序（`group-columns.tsx:184-199`）：列出的 → 回答里有的已归档（带说明）→ 放不下的“—” → “无”；拒收的原因的次序：开关关着 → 已归档 → 已完成（`:130-134`、`:194`）。泳道并入两级回答的 id；`listWordOn`、`useRouteListWords` 给面板把门；存下的选择留着（`group-by.tsx:39`、`sub-group-by.tsx:49-51`，但见 M1）。表头排序（`common.ts:267-282`、`spreadsheet-sort.ts`；`header-column.tsx`、`spreadsheet-header-column.tsx` 回到 P7a 之前的样子，`aria-sort` 总在，E9）。T16 写的 P6b 差异那一句核实为真（`page-preferences-filter.store.ts:51-61`、`:158`；`list_settings.go:49`）。
- **界面与可访问性**：弹窗的名字经 `ModalTitle`，后面的页面 inert，方向键可达，移除按钮有名字，说明读作“…, Cycle: Next”（读屏的前缀 `current-cycle-caption.tsx:26`），全被排除的提示 `role="status"`（`existing-issues-list-modal.tsx:273-276`），都在 `g4-add-existing-page.spec.ts:94-109`、`:133` 断言；组头按钮的 `aria-label` 是“{添加已有}: {组}”（`group-add-existing.tsx:40`），与“+”同一道门，另要规划页与 `joinable`（`:26`、`:36`）；筛选的选项“无”在最前、已归档的带说明、开关关着时 `isEnabled` 为假；只读：空态 `readOnly`，`canCreate` 要 `joinable` 与管理员或成员。
- **设计质量**：改到的产品文件都 ≤ 400 行（swimlanes 347、issue-filters 349、use-issues-actions 347、use-work-item-filters-config 366、group-columns 325、filter.ts 305、command-host 296、binding 286）；没有值的循环导入；`displayPropertyOn` 删去、不留别名；死的排序分支、两份 `prePopulateQuickAddData`、`[...new Set(ids)]` 删去；`GroupJoinRefusal` 是手写的字面量联合，有理由（问题码没有生成的枚举；`error-messages.ts` 的先例）；每个新的或重写的组件、hook、store 都加进 `project-address-route`；`.oxlintrc.json` 未改（没有新的直接读 `useSessionSWR` 的文件）；oxlint 44、knip、i18n 照收据；新键 `common.filter_options.fields.cycle_id`、`module_id`，`cycle.found_already_in`、`module.found_already_in` 在 en 与 zh-CN 都有。
- **测试与 e2e**：17 个新测试文件都是具体的；两个 `.types.test.ts` 的运行时断言照设计是空的（值在 tsc 里核）；三处既有断言的改写（`base-issues.store.test.ts` 的规划目录用例；`planning-columns.test.tsx:139`、`:158-159`；`module/page-preferences.test.ts:98-104`）都是 spec 要的（§2.2、§2.8、R7），台账有记；20 个点名的变异都有失败的收据（M2 所说的一处除外）；故事不用 `page.reload()`，锚照已有的夹具；chromium 3.5 + 3.4 = 6.9 秒 ≤ 14 秒。
- **文档**：plane-diff（扩充 3 行、新增 8 行）、frontend-changes（新增 6 行，`displayPropertyOn` → `listWordOn`）、设计 §8、§12、§14，M3、M4 的回执，M6-M8-01、-08、-13 与 `wc -l` 和台账一致；Plane 的引用抽查都对（`utils.tsx:594-600`、`search/issue.py:78-83`、`list/headers/group-by-card.tsx:123-157` 与 `:69-89`、`display-filters-selection.tsx:53-59`、`list-group.tsx:220-227`、`existing-issues-list-modal.tsx:82-98`）。
- **负责人**：没有要决定的：不改 `api/`、`server/`（E14），e2e 6.9 秒在 14 秒之内（E13），没有改 CI 的结构；R17 在预检之前已确认。

## 3. 结论与处理

结论 **Ready to merge**：没有 Critical、Important，四处 Minor 与六条观察，没有须负责人决定的事。主控仍以修复轮 F 处理 M1、M2（同 P6b、P7a 的先例），M3、M4 不改，六条观察照下表：

| 编号 | 问题 | 处理 |
|---|---|---|
| M1 | 存着的本页分组在面板里不勾也不列：`group-by.tsx:39` 只在本页的选项里留住存下的分组（`useGroupByOptions(groupByOptions)`）；模块页的选项没有 `module`（`withoutOwn`，P6b R12、P7b §2.5），而 nerve 对每一页都接受 `group_by: module_id`（`server/internal/modules/issue/app/list_settings.go:49`），`page-preferences-filter.store.ts` 的 `show()` 也认得它：经接口存下之后列表按模块分组，面板一项不勾、也没有一项叫出它；`sub-group-by.tsx:49-51` 同形，迭代页的 `cycle_id` 同 | F1 照主控裁定（re-75 的选项 (a)，与 R8 和组件注释“存下的那一项总在”一致）：一项列出当且仅当它是存下的那一项，或在本页的选项里且开关给它，次序照 `ISSUE_GROUP_BY_OPTIONS`；改选别的之后它不再是存下的，于是不再列出；看板上分组与子分组互藏对方的键照旧；`useGroupByOptions` 失去最后一个读者，删去。新测试两例（模块页的列表存着按模块分组、迭代页的看板存着按迭代子分组：列出并勾着，改选之后不再列出），两个面板各一个“只在本页的选项里留住”的变异都失败；plane-diff 的“模块页的按模块分组（M6/P6b）”一行与 frontend-changes 的词表一行随之改（`e0fe04ea`） |
| M2 | spec §2.5 要两页页头的测试在本页的词漏出时都失败，台账只有迭代一侧的收据（`t6-mut-cycle-page-keeps-cycle`：页头 2 例、常量 3 例），`t6-mut-module-page-keeps-filter` 只跑了常量的测试 | F2 只补收据，没有提交：变异 `f2-mut-module-page-keeps-grouping`（模块页的常量推出为 `withoutOwn(PROJECT_ISSUES, "module_id", "cycle")`，留着按模块分组，能编译）让模块页页头的 “never by a module (R12)” 两例与常量测试三例都失败，`restored=yes`，之后以干净的源重建常量（`f2-rebuild-constants`） |
| M3 | 看板（`kanban/default.tsx:111-121`）经 `getGroupByColumns({groupBy: sub_group_by, groupIds: [sub_group_id]})` 找泳道，离开项目的成员的一道泳道会得 `undefined` | 不改（主控核过）：泳道本身由同一函数、同一范围的 `getGroupByColumns` 生成（`swimlanes.tsx:295-302`），成员的列 `memberColumns`（`group-columns.tsx:295-319`）只列现有的成员，离开的成员没有泳道，查找总能找到渲染它的那一道 |
| M4 | `base-issues.store.test.ts` 1640 行 | 是测试文件，不在 E15 产品文件的上限之内；记为观察，后面的段再动它时按关注点拆，不另立交接 |
| 观察 | spec §2.7、§2.9 在三处有裁定的地方比实现窄（T9 只拒拖拽改变的那一维；T11 整批与一次 `followProgress` 在宿主里；T12 以 `exclusion` 取代 `shouldHideIssue` 里的本页排除）；台账 T16 把它们写进了前端改动清单与差异清单 | 主控的[阶段记录](../v0/M6-cycles-modules/reviews/P7b-list-vocabulary-review.md) §3 逐一列出 |
| 观察 | `NoInfer` 要 TypeScript ≥ 5.4（catalog 的版本满足，门禁通过） | 不需动作 |
| 观察 | `t8-mut-filter-ignores-switch(-module)` 最初的收据在测试加强之前存活，`-2`、`-3`、`-final` 都失败（台账） | 不需动作 |
| 观察 | “添加已有”的弹窗打开时不把焦点放进搜索框 | 已交 P7c（spec §5） |
| 观察 | 宿主里的 `useIssuesActions` 让宿主每次渲染多五个 hook 调用（T11 的裁定已接受） | 不需动作 |
| 观察 | 收据的卫生好（`inputs_unchanged`、恢复后的源、嵌套变异的内层收据） | 不需动作 |

F3 写本记录，并在 spec §5 加修复轮的清单、改 spec 与 plan 的状态、设计 §14 的 P7b 一行（`docs` 提交，即本记录所在的提交）。

修复轮中实现者的局部裁定（台账修复轮 F 的 Ruling，交限定复审核对）：
- F1 的规则在两个面板各写一行，不另抽共用函数：两个面板的选项来源与互藏的规则不同，一行 `filter` 抽出去只换来一个两个读者的小函数；简报要每个面板一个变异，两处各自可变异。两处的文档注释写明 R8 与这一情形并互指。
- F1 的规则照字面也适用于“无”（`null`）：看板的分组选项没有 `null`，经接口存下 `group_by: null` 的看板（网页自己的修改会把它改成 `state_id`，`page-preferences-filter.store.ts:268`）从此列出“None”并勾着，与“照存下的显示”一致；既有断言都不涉及。
- F1 的文档只改 plane-diff 的 P6b 一行（“面板里没有这一项”改为存下时列出并勾着、改选之后不再列出，本页的分组只能经接口存下）与 frontend-changes 的词表一行；spec §2.5 的“本页的键从不在本页给出”与 §2.3 的“不给”说的是不给出选项，不改；M4 的旧审核记录里提到 `useGroupByOptions` 的几处是历史记录，不改。
- F2 的变异取“模块页推出时去掉迭代而不是模块”（一处、能编译）；收据的命令只在页头与常量两份测试都失败时才以 1 退出。页头那两例失败在该例开头对本页常量的断言（渲染面板之前），与迭代一侧的 `t6-mut-cycle-page-keeps-cycle` 相同。

## 4. 修复轮 F 的复审

修复轮 F（`93229770..16c52f41`，2 个提交，F1、F3；F2 只有收据）经一次 fresh 限定复审（`fable`，范围 `93229770..16c52f41`，只读，只跑了文档链接的检查，没有跑构建与测试，与最终候选 `16c52f41` 的七项门禁并行）。原始报告没有另存，结论与核对过的性质照它转述。逐项核对：

- **范围**：F1 只改两个分组面板与 `utils.tsx`（`useGroupByOptions` 连同只为它的导入删去，不留别名，没有别的读者），没有别的产品改动；F2 只有收据；F3 只改文档。
- **F1 的规则**：两个面板（`group-by.tsx:40-42`、`sub-group-by.tsx:50-52`）一项列出当且仅当它是存下的那一项，或在本页的选项里且被给出，次序照 `ISSUE_GROUP_BY_OPTIONS`；看板上分组与子分组互藏对方的键不变；两处的文档注释属实。
- **别的页**：个人页（`profile/filter.store.ts:87-97` 把分组规范到它的选项）、归档页与子工作项的组件不受影响：它们存下的分组总在自己的选项里；每个子分组的选项都有“无”，那里的 “None” 不变。
- **副作用**（合意）：经接口存下 `group_by: state_group` 的项目列表从此勾着 “State groups”；经接口存下 `group_by: null` 的看板勾着 “None”，板为一列 “All work items”（`group-columns.tsx:44`），合 R8 与主控接受的裁定（§3）。
- **M3 不改**：泳道（`swimlanes.tsx:295-302`）以 `sub_group_id={_list.id}`（`:206`）交给看板，看板的查找（`kanban/default.tsx:111-121`）是同一函数、同一范围；成员的列 `memberColumns` 不看 `groupIds`、只列现有的成员；规划的泳道由 `[sub_group_id]` 复现。主控的理由成立。
- **测试**：两例新测试（`group-by.test.tsx:125-177`）是具体的，既有断言一行不改；RED `f1-red-own-grouping-listed` 2 例失败、14 例通过；F1 的两个变异都 `TSC_EXIT=0`、各只让自己那个面板的一例失败；F2 的 `f2-mut-module-page-keeps-grouping` 页头 `HEADER_EXIT=1`（R12 两例）、常量 `FILTER_EXIT=1`（三例），收据的 HEAD 为 `e0fe04ea`、`inputs_unchanged` 为真。
- **文档**：本记录的 19 个提交、114 个文件核过，§2、§3 照审查者的报告；spec §5、状态行、设计 §14 的一行（与台账一致）、差异清单与前端改动清单相符；唯一的断链是当时还没写的阶段记录（随后由 T17 的记录提交补上）。

结论 **Ready to merge**：没有 Critical、Important、Minor，三条观察都不需本段动作，没有须负责人决定的事：

| 编号 | 观察 | 处理 |
|---|---|---|
| 观察 | F2 的页头失败落在该例开头对本页常量的断言（模块页页头 `header.test.tsx:270`），在面板渲染之前；面板对存下的本页分组的行为由 F1 的两例钉住 | 不需动作 |
| 观察 | 经接口存下 `group_by: null` 的看板，点勾着的 “None” 会按状态分组（`page-preferences-filter.store.ts:268` 把看板的 `null` 规范为 `state_id`）；早于 F1，是用户自己的写 | 不需动作 |
| 观察 | 经接口存下 `group_by: target_date` 的项目列表（nerve 接受，`server/internal/modules/issue/domain/listvocab.go:49`）：`show()` 认得它（`groupValues`），`ISSUE_GROUP_BY_OPTIONS` 没有这一项，`getGroupByColumns` 也没有它的列；只能经接口到达，早于 P7b | 主控按只读追踪交 M8（[M6-M8-14](../v0/M8-open-release/handoffs/M6-closeout.md#M6-M8-14)）：迭代页、模块页同样；有工作项时列表与看板在渲染时抛错，正文落到版式的出错说明，选任一分组即恢复 |
