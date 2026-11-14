# M6 / P7a：工作项一侧（一）——写入路径与工作项自己的迭代、模块

> 状态：预检通过（修正后，[re-72](../../../reviews/re-72-M6P7a预检.md)），待实施。基线 main `954a1838bff44e98cc44f15df524da27e42fb1d8`（P6b 的 noff，双父 `06b72d4b`、`c0572192`；P6b 的最终分支 CI 43545459485 attempt 1 全绿，e2e 作业 23 分 09 秒；合入后的 main CI 43549053092 attempt 1 的 e2e 只在 M4 的 I4 (page) 的表格故事失败（故事的时间预算，没有重跑），由修复分支 `codex/m6-p6b-ci` 处理，修复的 noff 在 T1 之前并入本分支；这些与归档、清理由本段 T1 回填）。设计见 [M6 设计](../M6-design.md) §2 的 G4、G11，§3.3、§3.6、§8 的“工作项一侧”、§12 的 P7 一行；P7 的拆分与主控在写 spec 之前的裁定 D1–D10 见本阶段台账 `controller-rulings.md`（本段 R1–R10 照录）；交给 P7 的事见 [P5a spec](P5a-web-cycles.md) §5、[P5b spec](P5b-cycle-list.md) §5、[P6a spec](P6a-web-modules.md) §5、[P6b spec](P6b-module-list.md) §5 与 P6b 的[阶段记录](../reviews/P6b-module-list-review.md) §5。网页的长期规则见[总体设计 §7.2、§7.7](../../v0-design.md#77-stores-按会话分代长期有效m2-起)。接口合同见 [P3a spec](P3a-issue-membership.md)（工作项与草稿的 `cycle_id`、`module_ids`）。现状调研见本阶段台账 `research/web.md`（网页的现状、交接各项 A1–A24 与 X1–X2、e2e 与守卫、拆分的估算）、`research/plane-contract.md`（Plane 的工作项一侧、nerve 的契约、差异与风险）（ignored，随阶段归档）。负责人 2026-11-11 授权连续推进至 M6 收官，重要高风险决策仍停下确认；本段不改 `server/`、`api/`（D10），本机 `make test` 若因 [M6-M8-05](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-05) 的签名失败，以分支 CI 的服务端两路为准（负责人 2026-11-11）。

## 1. 目标与边界

P5a–P6b 交付了迭代、模块自己的页面与两页的工作项列表；工作项一侧还什么都没有：网页上没有迭代、模块的下拉框，没有一处组件读工作项的 `cycle_id`、`module_ids`（`research/web.md` §0）。设计 §12 的 P7 一行（属性与下拉框、新建弹窗与草稿、列表的筛选/分组/显示、添加已有、日历、命令面板、两页的移动端页头）估为 P6b 的 1.3–1.6 倍（约 2,800–3,000 行产品代码、75–90 个文件，`research/web.md` §10.2），超过一段约 16 个 Task 的上限，实施之前拆为三段（R1，主控裁定 D1）：

- **P7a（本段）——写入路径与工作项自己的迭代、模块**（G4、G11 的页面）：
  - **回填与阶段**：P6b 的回填（阶段记录 §7 含合入后 main CI 的失败与修复分支 `codex/m6-p6b-ci`、spec A8、设计 §14，A22；M6-M8-01 的新段随修复合入，本段只链接）；关键词阶段改为 `M6/P7`（A24）。
  - **写入路径**：写入单元的范围必给，去掉 M4 起的 router 退路，`issue-writes.ts` 加入 `project-address-route`（A10、A11，§2.1）；模块集合的变化以增减在写队列轮到时作用于所持的集合，迭代是单值字段（D3，§2.2）。
  - **按项目的规划目录**：下拉框的选项与工作项上的值的名字（未归档的与已归档的），P7b 的目录复用同一份读（D4、D5，§2.3）。
  - **两个下拉框**与交接的键盘各项（A1–A3，M2 交接 §4、§5(a)，M3 交接 §3，§2.4）。
  - **各处**：工作项详情的侧栏、预览、列表与看板的行、表格的列、显示属性 `cycle`/`modules`（§2.5）。
  - **新建与修改**：表单的两个字段、本页的预填（表单优先，D2）、修改发出差异、拒绝落到字段（§2.6）；分组头的新建弹窗移到列表的命令宿主、复制（A13、A14，§2.7）；草稿的字段、行与发布（D6，§2.8）。
  - **先拆**：要加代码的三个到上限的文件先按职责拆开（D7，§2.9）。
  - **文案**：下拉框的键（A20 的一部分，§2.4）。
  - **验收**：G4、G11 的页面（工作项一侧，§2.11）；M2 交接 §4、§5(a)；M3 交接 §3 中两个下拉框的“弹窗里的列表”；M6-M8-08 的行数。
- **P7b——列表词表与“添加已有”**（G15 的工作项一侧）：目录（A7）、按页的显示选项常量（A8）、偏好的筛选按页收窄（A9，P6b R16）、筛选/分组/排序/拖拽（A6）、空态与分组头与日历的“添加已有”（A4、A5），M4 交接 M6-02 与 M3 交接 §1 的回执。
- **P7c——命令面板、移动端页头与遗留**（G19 的工作项一侧）：命令面板（A16）、两页与项目页的移动端页头（A15、X2）、A17–A19、A21、其余没有读者的键（A20）、设计 §8 的四个视图文件（X1，主控把这项无人认领的事交给 P7c）、M2/M3/M1 其余的回执。

P7b、P7c 在 P7a 合入之后各写自己的 spec、plan 与预检，各自的 §1 写自己的边界；设计 §12 的 P7 一行由本段 T15 拆为三行（P5、P6 的先例）。

**不做**：
- P7b 与 P7c 的上述各项。具体地：本段的筛选、分组、排序词表里没有迭代与模块（`core/lib/issue-filters.ts`、`@nerve/constants` 的分组与排序选项不动）；表格里迭代、模块两列的表头没有按名字排序（R21）；没有“添加已有”与日历的 `addIssuesToView`；没有拖到迭代、模块的组；命令面板里没有改工作项的迭代、模块的命令。
- 任何接口或服务端的变化（D10）：接口没有前置条件、没有工作区级的迭代/模块列表、没有批量写，本段照现有合同做（§6）。

## 2. 交付合同

### 2.1 写入单元：范围必给（A10、A11）

**现状**：`IssueWrites.updateIssue(id, makeBody, drop?, operationScope?)`（`core/store/issue/issue-writes.ts:239-273`）在没有给出范围时退回 router 的工作区（`:260-261`，M4 起如此，P6b 照搬，P6b spec §2.1、§5），为此 `WritesRoot` 带着 `"workspaceSlug"`（`:23-35`、`:31`）。产品代码的 5 处调用都给了范围（`helpers/base-issues.store.ts:71`、`:83`，`module/issue.store.ts:42`，命令面板的 `ui/pages/context-based/work-item/mutation-commands.ts:35`，`issue-details/description-versions.store.ts:76`）；靠这条退路的只有 8 个测试文件 32 处（`research/web.md` §1.6 的表：`issue.store.test.ts` 11、`issue-drag.test.ts` 8、`issue-changes.test.ts` 6、`issue-details/detail-mutations.test.ts` 3，`detail-collaboration`、`detail-changes`、`resource`、`relations/data` 各 1）。

**做法**：
- `operationScope` 改为必给（参数次序不变，不是拖拽的调用处给 `drop` 为 `undefined`）；`WritesRoot` 去掉 `"workspaceSlug"`，退路与它的注释一并删去。写入单元从此不读 router：窄接口里没有 router 的 getter，经 `rootStore.router` 的读由 `project-address-route` 的内容模式（`\brootStore\.router\b`）挡住（下一条）。
- 32 处测试调用只加范围（`root.scope(…)`，同文件已有的写法），断言不改，以逐文件的基线收据核对同数。
- 根上 router 的 getter（`core/store/issue/root.store.ts:62-67`）只在仍有读者时留着：以 grep 核对，没有读者的随之删去。
- `issue-writes.ts` 加入 `project-address-route`（带样例）；P6b 因这条退路没有加入它（re-70 F5）。
- 行数：根 320 行、写入单元 300 行（P6b 的 A1）；本段的写入加在写入单元（§2.2），不加回根，两者仍各在约 400 行以内（[M6-M8-08](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-08)），T16 以 `wc -l` 核对。

### 2.2 模块集合的变化与迭代（D3，R3、R19）

**现状**：列表与详情的写都在调用时定下请求体：`BaseIssuesStore.updateIssue` 发 `() => body`（`helpers/base-issues.store.ts:70-72`），`useIssuesActions`（`core/hooks/use-issues-actions.tsx:114-119`、`:199-204`）、规划页的动作（`core/hooks/use-planning-page-issue-actions.ts:70-76`）、详情（`issue-details/issue.store.ts:151` 经 `projectIssues.updateIssue`，`issue-detail/use-issue-operations.ts:50`）、行（`properties/all-properties.tsx:56-59`）、表格的列（`onChange(issue, data)`）与弹窗的修改（`issue-modal/base.tsx:295-303`）都经它。只有 P6b 的“移出模块”在轮到时由所持的集合算（`module/issue.store.ts:41-47`）。`module_ids` 是整体替换（`api/modules/issue.yaml:2152-2160`），已归档的模块在工作项上（`:1717-1728`）。

**做法**：
- **变化的类型**（名字由实现定，例如 `IssueChange`）：由 `IssueUpdate` 推出，声明 `module_ids?: never`，另加一个可省的模块增减 `{ add, remove }`（两组 id 互不相交：下拉框的一下只有一个 id，弹窗的差异按构造不相交）。列表、详情、行、表格与弹窗的修改都经它。今天行与表格的写的类型是 `Partial<IssueDetail>`（`properties/all-properties.tsx:37`、`spreadsheet/columns/label-column.tsx:20`），它带着 `module_ids?: string[]`；改为这个变化之后，这样带整集合的值赋不进去，这些路径因此发不出调用时定下的整集合（re-72 F1）。类型测试以一个类型为 `Partial<IssueDetail>` 的值核对，不只核对对象字面量。
- **写入单元的命令**（名字由实现定，例如 `changeIssue(id, change, scope)`）：经 `updateIssue` 的 `makeBody`，在队列轮到时以所持工作项的 `module_ids` 为底，去掉 `remove`、按次序补上 `add` 中还没有的，得出整集合（纯函数，放在 `issue-write-fields.ts`）；请求体由变化的字段逐个点名写出（不写 `...change`），其余字段照给出的原样。`updateBody` 的白名单照旧放行 `module_ids`（`issue-write-fields.ts:35-36`），只给这个命令自己算出的整集合用（命令面板、拖拽与描述版本的 `makeBody` 不带它）。已归档的、页面不认识的 id 都在所持的集合里，照样保留（设计 §3.3“PATCH 时保留工作项上已有的隐藏 id”）。同一工作项连续的两次变化各排一个，后一个以前一个的回答为底；前一个被拒时以没有变的所持为底。
- `BaseIssuesStore.updateIssue`（各页的 store 继承）改经这个命令；规划页的动作照旧经它并跟进进度（P5b R10）。`TIssueOperations`（`issue-detail/root.tsx:29-35`，已是以更新类型为参数的泛型）、详情 store 的 `updateIssue`、行的 `updateIssue` 与表格的 `onChange` 的类型随之改为这个变化。
- P6b 的“移出模块”改为这个命令的 `{ remove: [本模块] }`（R19）：请求体与时机不变，P6b 的测试断言不改（基线收据）。
- **迭代是单值**：变化里的 `cycle_id` 照调用时给出的发（`null` 移出），不按所持算：给出当前的迭代 nerve 什么都不改（`issue.yaml:670-688`），先后两次选择以后发的为准，正是用户最后的选择。
- 不恢复 Plane 的 `addCycleToIssue`、`changeModulesInIssue` 的写链与乐观的分布更新（[M4 交接 M6-02](../handoffs/M4-closeout.md#M6-02)、P5b R10、P6b R13，plane-contract R15）。
- 接口没有前置条件：别的标签页或别人在这次读与写之间加入的模块，网页所持的集合里没有，整体替换仍会把它盖掉；“轮到时才算”（v0 §7.7）缩小而不消除这个窗口（P6b R5 的代价，§6）。

### 2.3 按项目的规划目录（D4、D5，R4、R5、R14）

- **选项**（可以加入的）：迭代是项目未归档的迭代中状态不是 `completed` 的，在 nerve 的次序（`sort_order, created_at, id`，`api/modules/planning.yaml:7-45`），前面一项“无迭代”；模块是项目未归档的模块（状态不论，设计 §3.3），nerve 的次序（R13）。选项在下拉框打开时读（`fetchCycles`、`fetchModules`，`core/store/planning/cycle.store.ts:125-126`、`module.store.ts:125`），经 `useDirectoryRead`（`issue-detail-widgets/directory-read.tsx:11-43`），照状态下拉框（`dropdowns/state/dropdown.tsx:21-43`）：按项目与权限身份分范围，迟到的回答不写（规划 store 的 `Reconciled` 也不让较旧的取数写入，P5a 的回执），读失败时列表底部说明并可重试（`DirectoryReadError`）。
- **值**（工作项上的）：以名字显示，含已结束、已归档的迭代与已归档的模块（设计 §3.6、G8）。名字取自规划 store（`getCycleById` 认两份列表与单读，`cycle.store.ts:74-83`；`module.store.ts:77` 同）；store 叫不出名字、而调用者在项目里有角色（`usePlanningReads(…).readable`，`core/hooks/use-planning-reads.ts:28-40`）时按需读：项目未归档的列表（还没有时）与已归档的列表，经 `useSessionSWR` 的键 `PROJECT_CYCLES`、`PROJECT_ARCHIVED_CYCLES`（模块的 `PROJECT_MODULES`、`PROJECT_ARCHIVED_MODULES`，与项目包装层 `use-project-fetch.ts:69-72`、归档目录同一份），一个项目一次，不按行读（R14）。仍叫不出的（其间被删除）不显示名字。已归档的值带一个“已归档”的说明（新键）。
- **跨项目的行**（工作区的草稿、个人页的列表；工作区的视图页属 M7）：每行按自己工作项的项目照上面两条读，不建工作区级的目录（nerve 没有工作区级的迭代/模块列表，M7，设计 §11；R5）。
- **给 P7b**：读都经规划 store 的读法（`fetchCycles`、`fetchArchivedCycles`，模块同）与上面这些键，选项与名字的规则在一处（`core/lib/` 的一个纯模块与一组 hook，名字由实现定）；P7b 的目录（P3b R9 的空 `Map`，`issue-list-binding.ts:84-86`）从同一份 store 的值建立，不另读一份。
- 新的用 `useSessionSWR` 的文件加入 `.oxlintrc.json` 的 swr 限制。

### 2.4 迭代与模块的下拉框（A1–A3，R4、R12、R13、R18、R20）

- **件**：新 `core/components/dropdowns/cycle/`、`dropdowns/module/`，建在 `@nerve/ui` 的 `CustomSearchSelect` 上（单选照状态下拉框 `dropdowns/state/base.tsx:38-148`，多选照成员下拉框 `dropdowns/member/base.tsx`）。共用件已满足 M2 交接 §5(a) 与 M3 交接 §3 的写法，本段在浏览器里核对（§2.11）：
  - 一个触发器，弹层持有列表，popper 的 ref 在它放置的 `Popover.Panel` 上（`web/packages/ui/src/dropdowns/custom-search-select.tsx:162-164`），列表在按钮旁展开；
  - 有搜索框，打开时搜索框取得焦点（`focusSearchOnOpen`，`:193`），按名字不分大小写过滤；Escape 关上，焦点回到按钮，不传给外面的弹窗；
  - 列表 `modal={false}`（`:187`），在工作项的弹窗里可用；
  - 触发器以字段与值为名（sr-only，照状态下拉框）；列表的行与表格的格里的，名字另含那一行的工作项（M3 交接 §3“每一行的控件以它那一行为名”，R20）。
- **迭代**：单选，选中即关；“无迭代”给 `null`；选当前的值不发；现值不是选项（已结束、已归档）时只在按钮上显示（带状态的图标、已归档的说明），不列在列表里（Plane 同，R4）。
- **模块**：多选，选了不关（Plane 同）；`onChange` 给出这一下的增减（一个 id 的加入或去掉），不给整集合（R3）；工作项上已归档的模块在选项之后列为已选的项、标明已归档，只能去掉、不能再选（R12）；按钮里不放嵌套的移除按钮（Plane 的 chip 的 “x” 不照搬，M3 交接 §3“嵌套的按钮”）；多于一个时按钮显示个数（文案经 `t()`）。选项不把已选的移到前面（R13）。
- **列表开着时所属的项目变了**：下拉框以项目为 key，列表关上，下次打开读新项目；旧项目的回答不进新项目的选项（M2 交接 §4 的第一点；第二、三点的结论已在 P5a 的回执：并发的两次取数由 `Reconciled` 保证、`status` 是生成的 `CycleStatus`）。
- **在途**：写在途时控件照常；nerve 确认之后才显示新值（v0 §7.7），模块的连续点击各是一个增减（§2.2）。
- **文案**：读 `common.cycle`、`common.modules`、`cycle.no_cycle`、`module.no_module`（没有模块时按钮的字）与 `cycle.label`（弹窗里迭代字段的空值）；“已归档”的说明是新键（en 与 zh-CN 同一提交；`check:sync` 不许一个键是另一个键的前缀）；删去没有读者的 `issue.add.cycle.failed`、`issue.remove.cycle.{loading,failed}`、`issue.remove.module.{loading,failed}`（R18）。`issue.remove.{cycle,module}.{label,success}` 由 P5b、P6b 的“移出本页”读着，不动；`project_empty_state.*.cta_secondary` 留给 P7b 的“添加已有”。

### 2.5 工作项的详情、预览、行、表格与显示属性（R11、R14、R20、R21）

- **开关**：每一处只在工作项的项目的 `cycle_view`、`module_view` 打开时显示迭代、模块（`planningOn`，`core/lib/planning-switch.ts:21-23`；Plane 同，`plane/apps/web/core/components/issues/issue-detail/sidebar.tsx:206-229`，R11）。
- **详情的侧栏与预览**（`issue-detail/sidebar.tsx:78-195`、`peek-overview/properties.tsx:74-186`）：在截止日期之后、父工作项之前加“模块”“迭代”两行（Plane 的次序与标题 `common.modules`、`common.cycle`），两处共用一件（名字由实现定）；不可编辑（`isEditable` 为假、预览的 `disabled`、已归档的工作项）时只读；写经 `issueOperations.update`：迭代 `{ cycle_id }`，模块 `{ modules: 增减 }`；被拒经操作已有的拒绝提示，显示留在 nerve 的值。
- **列表与看板的行**：`properties/all-properties.tsx`（334 行）旁新一件行的属性（名字由实现定），`all-properties.tsx` 只加渲染它的一处；显示属性 `modules`、`cycle` 打开时显示（Plane 的 `all-properties.tsx:343-380`）；`isReadOnly` 时不可改；写经行的 `updateIssue`（各页的动作；迭代页、模块页的跟进进度照旧，行因此离开或进入本页的列表照成员判断）。
- **表格**：`spreadsheet/columns/` 新两列；`SPREADSHEET_PROPERTY_LIST`（`web/packages/constants/src/issue/common.ts:139-151`）在标签之后加 `modules`、`cycle`（Plane 的次序），`SPREADSHEET_PROPERTY_DETAILS`（`:153-251`）给标题与图标；两列的表头在本段没有排序（R21）；格里的写同行。
- **显示属性**：`IIssueDisplayProperties`（`web/packages/types/src/view-props.ts:124-137`）加 `cycle`、`modules`；`ISSUE_DISPLAY_PROPERTIES_KEYS`、`ISSUE_DISPLAY_PROPERTIES`（`common.ts:79-137`，标题 `common.cycle`、`common.modules`）；`getComputedDisplayProperties`（`web/packages/utils/src/work-item/base.ts:76-91`）默认 `true`（nerve 的默认，`server/internal/shared/list_settings.go:92-95` 的 `DefaultListDisplayProperties`）；本地存的页（个人页、归档页）的 `displayPropertyNames`（`helpers/issue-filter-helper.store.ts:65-78`）；按页偏好的基类已原样收下与保留（`page-preferences-filter.store.ts:146`、`:247`）。显示面板（`filters/header/display-filters/display-properties.tsx`）在项目的页上只在路由的项目的开关打开时给这两项（Plane 的 `cycleViewDisabled`、`moduleViewDisabled`），在工作区一级的页（个人页）都给；迭代页、模块页照常给两项（Plane 同，不藏本页的）。
- 跨项目的行（个人页、草稿）的名字与选项照 §2.3。

### 2.6 新建与修改的弹窗（D2，R2、R3、R15、R17）

- **字段**：`IssueFormValues`、`ISSUE_FORM_FIELDS`（`issue-modal/types.ts:6-30`）加 `cycle_id`、`module_ids`；`DEFAULT_WORK_ITEM_FORM_VALUES`（`web/packages/constants/src/issue/modal.ts:11-22`）加 `cycle_id: null`、`module_ids: []`；弹窗的默认属性一栏（`issue-modal/components/default-properties.tsx`）给两个字段（§2.4 的下拉框，表单内的模块字段把下拉框的增减作用于表单自己的值），只在表单的项目的开关打开时。非草稿的表单换项目时两者随 `getUpdateFormDataForReset`（`web/packages/utils/src/work-item/modal.ts:12-20`）回到默认、清空；草稿见 §2.8。
- **新建的初值**（R2、R15）：迭代 = 调用者给的 `data.cycle_id` 中是选项的，否则本页可加入时的本页迭代；模块 = 调用者给的 `data.module_ids` 中是选项的，并上本页可加入时的本页模块；表单的项目没开开关时不带这个键。“本页”与“可加入”取自按域的表（`core/hooks/planning-page-domains.ts:26-47` 的 `joinable`，加一列本页的初值，名字由实现定，不写 `if (domain === …)`），只在表单的项目是本页的项目时。
- **初值在弹窗一级并进表单的默认值**（re-72 F2）：弹窗（`base.tsx` 或 T4 从它拆出的新建命令一件）在表单挂上之前把初值并进表单的默认值，而不是由字段的组件去填。`roots/cycle-create.test.tsx:66-68`、`module-create.test.tsx:66-68` 把 `IssueDefaultProperties` 换成 `() => null`，它们的断言（`cycle-create.test.tsx:155-160`、`module-create.test.tsx:162`）今天只靠 store 的注入成立；初值在弹窗一级并入，这些测试不改就仍是 R2 的回归保护（字段的组件不在场，请求体照样带本页的值）。
- **按选项过滤要等两份未归档的列表**（re-72 F7）：弹窗带着项目打开时，经同一份 SWR 键（`PROJECT_CYCLES`、`PROJECT_MODULES`）读这个项目的两份未归档列表；调用者给的值只在两份列表答到之后才按选项过滤（以改默认值的方式，不算改动）；之前字段显示加载，值留着、不丢。调用者给的值的过滤与它的测试在 T12（第一个真正的调用者是复制，re-72 F8），T10 只有本页的初值。
- **修改不预填本页**：修改显示工作项真实的值（含已结束、已归档的）；Plane 在修改时把路由的迭代、模块填进表单（`plane/apps/web/core/components/issues/issue-modal/modal.tsx:39-46`、`base.tsx:392-399`），不照搬（P6b R7）。
- **新建发出表单所持的**（D2）：请求体（`issue-modal/base.tsx:246-256`）加 `cycle_id`（为空时省略：`IssueCreate.cycle_id` 不可为 `null`，`api/modules/issue.yaml:2065-2071`）与 `module_ids`（为空时省略）：两个值为空或字段因开关关着不显示时，新建的请求体没有这两个键，M4、M5 的新建请求体照旧。“存为草稿”照 §2.8 的一条规则。弹窗的新建不经本页的注入（迭代页的 store 覆写 `cycle_id`，`cycle/issue.store.ts:21-30`；模块页的并入本模块，`module/issue.store.ts:23-34`）：规划页的动作给表单另一个照给出的发出的新建（名字由实现定），确认的工作项在本页时照旧跟进进度；快速新建没有字段，照旧经本页的注入（`use-planning-page-issue-actions.ts:61-69`）。于是在迭代页选别的迭代或“无迭代”、在模块页去掉本模块时，nerve 收到的就是表单的值；照旧只发一个请求。
- **修改发出差异**（D3）：修改的请求体照旧只带改过的字段（`form.tsx:165-174`）；`cycle_id` 改了就带；模块改了时以表单的初值（表单挂上时的值，不是之后随读取变化的 `data`）与终值算出增减，经 §2.2 的命令在轮到时作用于所持的集合：打开之后所持的集合里多出的模块（例如弹窗自己读工作项时读到的）留着。模块没改时请求体不带模块（P6b R7 的测试照旧成立）。
- **拒绝落到字段**（R17）：`useFormRefusal`（`form.tsx:153`）加 `cycle_id`、`module_ids`（`module_ids[i]` 落到模块一栏，`core/lib/form-refusal.ts:20-26` 已如此）；`planning.cycle_completed` 落到迭代一栏（它只可能指迭代）；其余（`planning.archived`、`planning.{cycles,modules}_disabled`、`project.archived`）照旧提示。

### 2.7 分组头的弹窗与复制（A13、A14，R15、R16）

- **分组头的新建弹窗移到列表的命令宿主**（R16）：列表与看板的分组头把弹窗放在组里（`list/headers/group-by-card.tsx:92-100`、`kanban/headers/group-by-card.tsx:56-64`）；列表核验时整个布局换成加载（`issue-layout-HOC.tsx:52-58`，`getIssueLoader` 在核验中给 `"init-loader"`，`helpers/base-issues.store.ts:64-67`），弹窗随之卸下，而命令宿主（`IssueListCommandHost`，`issue-layout-HOC.tsx:55`）留着。命令宿主照行的修改与复制（P5b T13）改三处（re-72 F4）：
  - 加一种“新建”的请求，带分组的 `data` 与 store 类型（`command-host.tsx:27` 的 `RequestDialog`）。今天宿主只在 `holds(dialog, dialog.data.project_id)` 时渲染对话（`:160-165`、`:196`），分组的 payload 多半没有 `project_id`（按状态分组是 `{state_id}`），照旧就永远不显示：新建的对话由路由的项目持有，按项目分组时由分组的 `project_id` 持有，没有路由的项目的页（个人页）由工作区的权限身份持有（项目列表为空）。
  - 锁定项目（`isProjectSelectionDisabled`）由宿主按 `isPlanningPageStoreType` 给出，同今天的两个分组头（`list/headers/group-by-card.tsx:99`、`kanban/headers/group-by-card.tsx:63`；P6b §2.6）。
  - `DialogStore`（`:26`）包括分组头会给“+”的每一种 store 类型（以各 store 的 `viewFlags.enableIssueCreation` 为准：项目、迭代、模块、个人页）。
- **复制**（R15）：命令宿主的 `copyOf`（`command-host.tsx:54-66`）带上源的 `cycle_id`、`module_ids`；新建的初值（§2.6）只留其中是选项的（源的已结束、已归档的迭代与已归档的模块不带：nerve 会以 409 拒绝整个新建），模块页上再并上本模块；过滤等两份未归档的列表答到，之前字段显示加载、值留着（§2.6，re-72 F7）。描述带图片的复制经内容复制（`issue-modal/content-copy.tsx`，草稿再发布；`base.tsx:227-241` 的 `draftInput`）：表单的两个值随 `draftInput` 进草稿、随发布加入，P5b §5、P6b §5 记下的“内容复制不加入本页”随之了结。P6b §5 的嵌套变异（`copyOf` 带源的模块、新建的请求体带表单的模块两处一起改）现在有能失败的断言。

### 2.8 草稿（D6，R6）

- **表单**：`draftPayloadFields`（`issue-modal/draft-form.ts:9-21`）加 `cycle_id`、`module_ids`；存下的值可以是已结束、已归档的，照 §2.3 显示。
- **一条规则：迭代为空就省掉，模块的空集合照发**（re-72 F6，主控裁定）：`draftPayload`（`core/services/issue/draft-input.ts:20-40`）不改。去掉迭代是一个带 `cycle_id: null`（或 `""`）的修改，键随之从存下的 payload 里去掉（修改里有的键以 `Object.hasOwn` 盖过 `before`）；`module_ids: []` 照留、照发：契约允许空集合，`[]` 正是清空草稿的模块的写法，省掉这个键反而留下 `before` 的集合。新草稿没有迭代时没有 `cycle_id`（不发 `null`、`""`，plane-contract R10），可以带 `module_ids: []`。本段以测试钉住这两点。
- **换项目清空**：草稿的表单换项目时（`form.tsx:157-163` 只在非草稿时重置）清空两者，照上一条保存（`cycle_id: null` 去掉键，`module_ids: []` 清空集合），免得发布时以 422 `cycle_id`、`module_ids[i]` 被拒（plane-contract R11）。
- **行**：`workspace-draft/draft-issue-properties.tsx`（165 行）加两个下拉框（草稿有项目、项目的开关打开时）；迭代经已有的 `change({ cycle_id })`（`:45-53`），模块的增减经草稿 store 的队列在轮到时以所持的草稿算（`workspace-draft/issue.store.ts:137-154` 的 `input(held)`，行已是 `(held) => draftInput(…, held.payload)`，`draft-issue-block.tsx:184-185`）。草稿的整个 payload 被替换（`api/modules/issue.yaml:1615-1621`），其余键照所持的保留。
- **发布**：被拒（迭代已结束、已归档、已删除，开关关闭）时草稿留着；在草稿的弹窗里发布时，拒绝照 §2.6 落到字段（现在只提示，`base.tsx:321-342`、`form.tsx:205-223`）；草稿列表的行上发布照旧提示。

### 2.9 到上限的文件先拆（D7，R7）

- `issue-modal/base.tsx`（452 行）：新建、修改、发布三个命令（`:220-342`）与描述的读（`:110-152`）各出一件（名字由实现定）。
- `issue-modal/form.tsx`（432 行）：提交（`submissionData`、`handleFormSubmit`、`handleMoveToProjects` 与拒绝落到字段，`:153-223`）与页脚的按钮（`:359-425`）各出一件。
- `issue-layouts/utils.tsx`（399 行）：拖放的件（`GroupDropLocation`、`getSourceFromDropPayload`、`getDestinationFromDropPayload`、`handleGroupDragDrop`，`:59-237`，P7b 往拖放加迭代、模块的组）与表格的列表（`SpreadSheetPropertyIcon`、图标表、`SPREADSHEET_COLUMNS`，`:308-389`）各出一件。
- 只搬不改，先留基线收据，测试只改导入；拆后各约 400 行以内，之后的 Task 加在拆出的件里。工作项根 store（320）与写入单元（300）不拆，保持在上限内（M6-M8-08）。

### 2.10 守卫、文案与告警

- **`project-address-route`**：新建与改写的组件、hook、store 随各自的 Task 加入规则的文件表并带样例（hit 与 miss 各至少一例）；新文件只经路由参数或 props 取工作区、项目；`issue-writes.ts` 在 T2 加入。
- **`.oxlintrc.json` 的 swr 限制**：用 `useSessionSWR` 的新文件随它的 Task 加入；只收紧。
- **告警上限** `lint-cap` 恰为 44，只降不升；新文件不得新增告警。**knip**：新文件在第一个产品读者出现的 Task 里建立，只给测试读的导出不算读者（本阶段由审查执行）。
- **关键词**：phase 自 T1 起为 `M6/P7`（三段共用，同 P6a/P6b 共用 `M6/P6`）；没有在 `M6/P7` 到期的例外（`research/web.md` §9.2）。
- **文案**：§2.4 的读与删、“已归档”的说明；行的名字若要新键（R20）随它的 Task 加入；en 与 zh-CN 同一提交，`check:sync` 不许前缀键。
- `core/store/root.store.test.ts:106` 的 store 数仍是 21（本段不加 `RootStore` 一级的 store）。
- 规划页测试的共用替身 `issue-layouts/roots/fake-planning-page.tsx` 的模块级 `opened` 不在关闭时清空（re-71 §4，A21）属 P7c（D1）；本段用它的帮手的测试都先打开页。

### 2.11 e2e（D9，R9）

页面故事放在 `e2e/stories/planning/`，标题 `G<n> (page): …`，文件名合 `planning/g\d+-[\w-]+-pages?\.spec\.ts`，纳入三种引擎的页面配置 `e2e/playwright.m6-pages.config.ts`；CI 照常只跑默认配置。世界经接口建立（新项目的 `cycle_view`、`module_view` 默认关闭，`server/migrations/sql/00010_project_projects.sql:18`，故事先打开）；帮手从原处导入，不复制（`cycle-pages.ts` 的 `dismissToasts`、`toast`、`recordApi`，`settings-pages.ts` 的 `sentHeld`，`planning-list-pages.ts` 的 `openListPage`、`bodiesSent`，`dialogs.ts` 的 `modalDialog`，`assert/issue-planning.ts` 的 `expectCycleRows`、`expectModuleRows`、`expectPlanningActivity` 与 `cycleJoined` 等）；名字冲突时改名，不留别名。

| 文件 | 覆盖 |
|---|---|
| `g4-issue-cycle-page.spec.ts` | 一个故事。项目里进行中、未开始、已结束、已归档（已结束后归档）四个迭代。工作项详情页的迭代一栏：以键盘打开，列表在按钮旁（以两者的框核对）、搜索框取得焦点；选项恰为“无迭代”、进行中、未开始（不列已结束、已归档）；输入过滤；选中只发一个带 `cycle_id` 的 `PATCH`；再换到另一个（移过去）、选“无迭代”（`null`，移出）；数据库的归属与三条动态（`expectCycleRows`、`expectPlanningActivity`）；Escape 关上，焦点回到按钮。项目页的列表里一个在已归档迭代里的工作项显示该迭代的名字与“已归档”的说明（只一次已归档列表的读）。迭代页页头的新建：弹窗的迭代一栏预填本迭代，在弹窗里打开它可用，换成未开始的迭代，只发一个 `POST`、带它（表单优先），新工作项不在本迭代的列表里，数据库里在未开始的迭代 |
| `g11-issue-modules-page.spec.ts` | 一个故事。模块 Alpha、Beta、Delta 与已归档的 Gamma（已完成后归档）；工作项 W 在 Alpha 与 Gamma。项目页预览 W：模块一栏显示 Alpha 与 Gamma（“已归档”的说明）；打开列表，选项 Alpha（已选）、Beta、Delta，Gamma 是只能去掉的已选项；扣住第一个 `PATCH` 的回答，点 Beta、再点 Alpha，放开：两个请求体依次是 {Alpha, Gamma, Beta} 与 {Gamma, Beta}（第二个以第一个的回答为底，不是点下时下拉框的值）；数据库 Gamma 仍在、Beta 加入、Alpha 离开，动态两条。接口在页面背后给 W 加入 Delta 之后，行的“修改”打开弹窗（弹窗自己读工作项时读到 Delta），在弹窗的模块一栏去掉 Beta 保存：请求体是 {Gamma, Delta}（差异作用于所持的集合，Delta 留着）；数据库同 |

**写法**（P5b、P6a、P6b 阶段记录 §5 的教训）：应用内跳转之后以目标页独有的内容为锚；“载入后仍在”由另开的浏览器核对，不用 `page.reload()`、不在重新载入之后核对控制台（M6-M8-09）；关通知只用共用的 `dismissToasts`（M6-M8-10）；有两道守卫时以嵌套变异证明并写明单去一道为何等价；没有能编译又让故事失败的产品变异时，以期望错误的 RED 证实断言；e2e 变异的脚本先跑 `react-router typegen` 与两边的 `tsc --noEmit`，只计能编译、能构建的；写 e2e 变异之前先弄清页面手里有什么（P6b T13 的等价变异）。

**时长**（D9，re-72 F5）：本段两个新文件本机 chromium 合计约 20 秒（每个约 10 秒；超过 25 秒先压缩再继续，台账写明）。数据点：
- P6a 的 main CI 43469787621 attempt 2 的 e2e 作业 25 分 00 秒；
- P6b 的检查点分支 CI 43517662603 为 25 分 38 秒，最终分支 CI 43545459485 为 23 分 09 秒；
- P6b 合入后的 main CI 43549053092（main `954a1838`）的 e2e 作业 132311928038 为 03:14:42Z–03:42:23Z，**27 分 41 秒**，故事本身约 25.4 分钟，是记下的最慢的 runner；它唯一的失败是 M4 的 I4 (page) 的表格故事超了时间预算，修复分支 `codex/m6-p6b-ci` 把它拆成四个（本机 chromium 约多 5 秒，经过见修复合入后 [M6-M8-01](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-01) 的新段）；
- 修复合入后的 main CI 43557098427（main `7868a6b4`）的 e2e 作业 132336647667 为 04:48:59Z–05:09:19Z，20 分 20 秒（472 条，故事 18.5 分钟；T1 回填），是 T16 估算的基线；T9 检查点的分支 CI 43573631430（`f1646370`）的 e2e 作业 132386476220 为 08:13:18Z–08:32:50Z，19 分 32 秒（那时还没有本段的页面故事，校准点）。

T16 推送之前照 P6b spec §2.10 的写法算三个数：
- **估算**：基线（最后一次绿的 main CI 的 e2e 作业，即修复合入后的那次，T1 回填）加上新文件 chromium 时长 × 1.67–2.25、两个 worker 摊开，约加 17–23 秒；
- **上界**：已测到的最慢的数据点（现为 27 分 41 秒）加上新文件 chromium 时长 × 3.6–4.1（约 72–82 秒），那次运行还没有 I4 的拆分时再加它的 5 秒 × 3.6–4.1（约 18–21 秒）：约 29.2–29.4 分钟；
- **校准点**：T9 检查点的分支 CI 实测的 e2e 作业时长（那时还没有新故事，但迭代页、模块页的故事已打开开关、行上多了两个属性），与基线之比就是那次 runner 的快慢，也量出本段对既有故事的影响。

正常的 runner 上约 23.5–26 分钟；慢的 runner（×1.31，[M6-M8-01](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-01) 记下的比值）或像 43549053092 那样的 runner 上，本段的上界接近或超过 30 分钟的报告线。T9 的检查点是校准；估算、上界或任一次实测（T9、T16 推送后的分支 CI）超过 30 分钟时，T16 先停下报告负责人、不推送；改 CI 的期限或作业结构（例如分片）须负责人同意（D9，M6-M8-06 的教训）。

## 3. 局部裁定

- **R1 P7 拆为 P7a、P7b、P7c（主控裁定 D1）**：设计 §12 要每段约 16 个 Task 以内、超限在实施前拆分；调研估出 P7 约为 P6b 的 1.3–1.6 倍，拆两段时第二段约 25 个 Task。拆缝取依赖的次序：写入路径与工作项自己的值在前（“添加已有”与拖拽都要复用 §2.2 的模块增减与 §2.3 的目录），列表词表与“添加已有”其次，命令面板、移动端页头与遗留最后。X1（设计 §8 的四个视图文件，没有阶段认领）交 P7c：只改网页、与 `project-address-route` 同一主题。不取两段（超上限），也不取一段。代价：三轮 spec、预检、整审与 CI；设计 §12 的 P7 一行拆为三行（T15）。
- **R2 表单优先于本页（主控裁定 D2）**：弹窗有了迭代、模块字段之后，本页的迭代、模块只是新建的初值（可加入时，照现在），表单所持的就是发出的；页面 store 不再覆写 `cycle_id`、不再把本模块并进表单给出的集合；本页的注入只留给没有字段的调用者（快速新建）；修改不预填路由的迭代、模块，显示工作项真实的值。理由：Plane 让表单的选择胜过页面（`plane/apps/web/core/components/issues/issue-modal/base.tsx:174-183`、`:200-216`）；留着 store 一级的注入会悄悄加回用户刚去掉的（plane-contract R2），P6b R6 的“取并”只防住了覆盖，防不住去掉。不取“页面优先”（表单显示的不是发出的），也不取模块“取并”（去不掉本模块）。代价：规划页的动作多一个给表单的新建；在迭代页以别的迭代新建的工作项不出现在本页（照用户所选）；登记差异。
- **R3 模块集合以增减在写队列轮到时作用于所持的集合，迭代是单值（主控裁定 D3）**：下拉框（详情、预览、行、表格、草稿的行）的每一下是一个增减，在轮到时作用于最新的所持集合，连续的点击各排一个、不是下拉框的值的快照，已归档与页面不认识的 id 保留；弹窗的修改发出表单的差异（初值到终值），同样在轮到时作用。迭代是 `{cycle_id}` 的固定字段。新的写在写入单元（P6b R1）。理由：`module_ids` 整体替换，漏掉已归档的模块会把工作项移出它（设计 §3.3）；Plane 的行把加与减作为两个并发的请求发出（`plane/apps/web/core/components/issues/issue-layouts/properties/all-properties.tsx:133-134`），多选点了不关（`plane/apps/web/core/components/dropdowns/module/base.tsx:96-99`），照快照发会让第二下盖掉第一下；P6b R5 的先例。变化的类型声明 `module_ids?: never`，`Partial<IssueDetail>` 这样带整集合的值赋不进去，命令逐个点名写请求体，列表与详情的路径因此发不出整集合（编译器挡住，类型测试以 `Partial<IssueDetail>` 的值核对，re-72 F1）。不取“发下拉框的值”（第二下丢掉第一下），也不取“所持变了就拒绝修改”（接口没有前置条件，只会拒绝正当的修改）。代价：接口没有前置条件，别的标签页或别人在读与写之间的改动仍可能被盖掉（§6）；P6b 的“移出模块”改经同一命令（R19）。
- **R4 选项与值分开（主控裁定 D4）**：选项是未归档且未结束的迭代加“无迭代”、未归档的模块；值是工作项真实的归属，以名字显示，含已归档的模块与已结束、已归档的迭代；已归档的模块可以去掉；选一个选项不丢隐藏的 id；现值不是选项的迭代只在按钮上显示。理由：G8、设计 §3.3、§3.6；nerve 以 409 拒绝加入已结束（`planning.cycle_completed`）、已归档（`planning.archived`）的；Plane 的工作项回答去掉已归档的模块（`plane/apps/api/plane/utils/grouper.py:59-68`，差异清单已登记），所以它不必显示或保留，nerve 的回答含着。不取“已归档的也作选项”（409），也不取“隐藏已归档的值”（G8 要显示，整体替换要保留）。代价：要显示已归档的名字时多读一次已归档的列表（R14）。
- **R5 跨项目的行照状态下拉框按项目读（主控裁定 D5）**：工作区的草稿、个人页等列出多个项目的行时，每行读自己项目的目录（选项在打开时经 `useDirectoryRead`，按会话与权限分范围，迟到的回答不写；名字按 R14）；不建工作区级的目录（M7）。代价：这些页上按项目各读一次。
- **R6 草稿（主控裁定 D6）**：没有迭代时省略 `cycle_id`（新建与保存草稿都不发 `null` 或 `""`，plane-contract R10；`module_ids: []` 照发，它是清空草稿的模块的写法，`draftPayload` 不改，re-72 F6）；草稿换项目时清空迭代与模块（R11）；发布的拒绝落到字段。理由：`IssueCreate.cycle_id` 与草稿 payload 的 `cycle_id` 都不可为 `null`（`api/modules/issue.yaml:2065-2071`、`:1604`），Plane 去掉迭代时发 `""`（`plane/apps/web/core/components/issues/workspace-draft/draft-issue-properties.tsx:68-71`）；换项目后留着旧项目的 id，发布时 422。代价：草稿换项目丢掉迭代与模块（同非草稿的表单换项目）。
- **R7 到上限的文件先拆（主控裁定 D7）**：`issue-modal/base.tsx`（452）、`form.tsx`（432）、`issue-layouts/utils.tsx`（399）本段要加代码，先按职责拆开（约 400 行的上限、没有大文件、没有兼容层）；工作项根 store（320）与写入单元（300）保持在约 400 行以内（M6-M8-08）。代价：一个只搬不改的 Task，以基线收据核对。
- **R8 Plane 的缺陷不照搬（主控裁定 D8）**：下拉框从不取数（nerve 打开时读，§2.3）；popper 的 ref 不在列表上（共用件已在列表上）；改模块不接拒绝（nerve 说明原因）；新建之后的加入失败时提示“创建失败”（nerve 只有一个请求）；“添加已有”在模块页说“迭代”、命令面板的几处（P7b、P7c）。
- **R9 e2e 的时长预算（主控裁定 D9）**：本段新的页面故事本机 chromium 合计约 20 秒；T16 照 P6b spec §2.10 的公式与已测的数据点（含 27 分 41 秒的最慢一次）估算，T9 的检查点校准；估算或上界超过 30 分钟时停下报告负责人、不推送；CI 的期限、超时与作业结构不经负责人不改。
- **R10 只改网页（主控裁定 D10）**：不改 `server/`、`api/`；遇到网页上处理不了的契约缺口就停下报告（架构一级）。写 spec 时没有找到这样的缺口（§6）。
- **R11 属性随项目的开关显示**：详情、预览、行、表格、弹窗与草稿的行只在工作项（表单、草稿）的项目打开 `cycle_view`、`module_view` 时显示迭代、模块；显示面板在项目的页上同此。理由：Plane 同（`plane/apps/web/core/components/issues/` 下的 `issue-detail/sidebar.tsx:206-229`、`issue-layouts/properties/all-properties.tsx:343-380`、`issue-layouts/filters/header/display-filters/display-properties.tsx:22-23` 的 `cycleViewDisabled`、`moduleViewDisabled`）；关闭时 nerve 拒绝任何变化（409），数据留着、再打开就在（G18）；新项目默认关闭（`00010_project_projects.sql:18`），M4、M5 的故事所在的项目因此不受影响。不取“关闭时只读地显示”（为项目关掉的功能给一个不能用的控件）。代价：开关关着时工作项上看不到它的迭代、模块；登记差异。
- **R12 已归档的模块经列表去掉，按钮里没有嵌套的按钮**：已归档的模块在列表里是标明已归档、只能去掉的已选项。理由：设计 §3.3“移出不受归档的限制”，D4 允许去掉；Plane 的 chip 把 “x” 按钮放在下拉框的按钮里（`plane/apps/web/core/components/dropdowns/module/button-content.tsx:85-97`），是 M3 交接 §3 要去掉的嵌套按钮。不取“chip 上的 x”（嵌套的交互）、也不取“去不掉”。代价：去掉一个已归档的模块要先打开列表。
- **R13 选项的次序**：nerve 的次序（`sort_order, created_at, id`，即 Plane 的 `sort_order`）；“无迭代”在首（Plane 同）；模块的已选项不移到前面，已归档的已选项在选项之后。理由：Plane 的模块选项每次渲染都把已选的排到前面（`plane/apps/web/core/components/dropdowns/module/module-options.tsx:110-113`），多选点了不关，每点一下指针或键盘所在的项就换了位置。代价：已选的模块不集中在顶部；登记差异。
- **R14 名字按需读，一个项目一次**：store 叫不出名字时才读项目的列表（未归档的、已归档的），经与项目包装层、归档目录同一份的 SWR 键，同一项目的多行只读一次。不取逐个读实体（`fetchCycle(id)`：一行一个请求），也不取每页都读已归档的列表（大多数页没有已归档的值）。代价：已归档的值第一次显示时等一次读（之前不显示名字）。
- **R15 复制带源的可加入的迭代与模块，模块页并上本模块**：`copyOf` 带上源的 `cycle_id`、`module_ids`，新建的初值只留其中是选项的；初值的规则与新建一致（调用者给的在前，本页的补上）。理由：Plane 的复制带源的值、源没有时用路由的（`plane/apps/web/core/components/issues/issue-modal/modal.tsx:39-46`）；nerve 的新建检查给出的每一个（`api/modules/issue.yaml:566-578`），带上已结束、已归档的会让整个新建被 409 拒绝。P6b R7 写明“直到 P7 的表单有模块字段”。不取“只加入本页的”（表单已显示模块，用户看得见、改得了），也不取“全带”（被拒）。代价：源在已归档的模块里时，复制的那份不在其中；登记差异。
- **R16 分组头的新建弹窗移到列表的命令宿主（A14）**：核验时布局整个换成加载（`issue-layout-HOC.tsx:52-58`），组与分组头随之卸下，弹窗不能活过核验；这是构造上的，不是偶然。P5b T13 为行的修改与复制做了同样的事；本段的表单打开期间还多了目录的读。不取“留着、出了事再说”，也不取改 M4 的加载（核验中的组不是可信的证据）。代价：分组头经命令宿主请求对话，宿主多一种对话。
- **R17 拒绝落到字段**：422 的 `cycle_id`、`module_ids[i]` 落到迭代、模块一栏；409 的 `planning.cycle_completed` 落到迭代一栏；其余提示。理由：M2、M3 起表单的被拒落到字段并取得焦点（`useRefusedFocus`）；`planning.cycle_completed` 只可能指迭代，`planning.archived` 可能是迭代也可能是模块，开关与项目的拒绝不属任何字段。代价：无。
- **R18 删去 Plane 下拉框的五个提示键**：`issue.add.cycle.failed`、`issue.remove.cycle.{loading,failed}`、`issue.remove.module.{loading,failed}` 没有读者（P6a §5、P6b §5）。nerve 的属性修改不出加载中的提示（状态、标签等都如此），被拒时说 nerve 的原因（`useRefusalToast`），比 Plane 的笼统文案多出原因。不取“读上它们”（与现有写法不一致）。代价：无。
- **R19 P6b 的“移出模块”改经同一个增减命令**：模块集合只有一种写法；请求体与时机不变，P6b 的测试断言不改。代价：动 P6b 的一个文件。
- **R20 行与表格里的下拉框以那一行为名**：M3 交接 §3 的规则（一行一个、每行都一样的控件以那一行为名，如 `spreadsheet/issue-row.tsx:276` 的子工作项按钮）。代价：可能要一个带 `{name}` 的新键。
- **R21 表格两列的表头在本段没有排序**：按迭代名、模块名排序要目录，`prepareCatalog` 在目录之前对 `cycle_name`、`module_name` 大声失败（P3b R9，`issue-list-binding.ts:226-289`）；本段给了表头的排序会让列表失败。P7b 随目录加入。代价：P7b 之前网页上不能按这两列排序（接口可以）。

本段的裁定都是局部决定：R1–R10 是主控在写 spec 之前的裁定（D1–D10），R11–R21 在已批准的设计与 P5b–P6b 的先例之内，不改接口、不改服务端、不跨后端模块。**没有须负责人决定的事。**

## 4. 验收标准

- [ ] A1 写入单元：`updateIssue` 的范围必给，`WritesRoot` 没有 `workspaceSlug`，没有 router 退路；`issue-writes.ts` 在 `project-address-route` 里（样例与变异）；32 处测试调用只加范围、断言不改（基线收据同数）；根与写入单元各在约 400 行以内。
- [ ] A2 模块集合：增减在轮到时作用于所持的集合（连续两次的变化、前一个被拒、已归档与不认识的 id 保留）；迭代是单值；列表与详情的更新类型声明 `module_ids?: never`（`Partial<IssueDetail>` 的值赋不进去）、命令逐个点名写请求体；P6b 的“移出模块”改经同一命令（基线同数）。每个性质有单测，变异能让测试失败。
- [ ] A3 目录：选项在打开时读、按项目与权限分范围、迟到的不写、读失败可重试；名字含已结束与已归档的，按需、一个项目一次；跨项目的行按各自的项目；新 SWR 读者在 swr 限制里。
- [ ] A4 下拉框：两个下拉框的选项、值、“无迭代”、已归档的已选项只能去掉、模块的 `onChange` 是增减、以行为名；列表开着时换项目关上并改读新项目；M2 交接 §4、§5(a) 与 M3 交接 §3 的两个下拉框各项在浏览器里核对（G4、G11 的页面故事）。
- [ ] A5 各处：详情、预览、行、表格、显示属性 `cycle`/`modules`（类型、常量、默认、本地存的页、显示面板）；开关关闭时不显示；只读时不可改；被拒时说明；表头没有排序。
- [ ] A6 弹窗：新建的初值（调用者的选项 + 本页可加入时的）在弹窗一级并进表单的默认值、按选项过滤等两份列表答到、表单优先、只发一个请求、为空时省略；修改不预填本页、发出差异；拒绝落到字段；分组头的新建弹窗在命令宿主（按状态分组也能打开）、活过核验；复制带源的可加入的值并本页的，内容复制随之加入。
- [ ] A7 草稿：表单与行的两个字段、换项目清空、不发空的迭代（`module_ids: []` 照发）、行的模块增减在轮到时算、发布的拒绝落到字段。
- [ ] A8 拆分：三个文件按职责拆开、各约 400 行以内，测试只改导入（基线收据同数）。
- [ ] A9 G4、G11 的页面故事（三种引擎，§2.11）。
- [ ] A10 守卫：`project-address-route`、swr 限制、告警上限 44、knip、关键词（phase `M6/P7`）、i18n（读、新键、删去的五个键）、`root.store.test.ts` 21 个 store。
- [ ] A11 文档：P6b 的回填（阶段记录 §7 含 CI 的修复、spec A8、设计 §14）；交接回执（M2 交接 §4、§5(a)，M3 交接 §3 的两个下拉框，M6-M8-08 的行数）；差异清单、前端改动清单、设计 §8/§12（P7 一行拆为三行）/§14。
- [ ] A12 七项本地门禁（`make test` 照负责人 2026-11-11 的规则）、deep-review 修复闭环、准确的分支 CI 与 noff 后的 main CI 全绿、归档清理。（合入后由 P7b 的 T1 勾选）

## 5. 后续责任

**P7b**（列表词表与“添加已有”）：目录（A7）从 §2.3 的 store 读法与键建立，`prepareCatalog` 取两份列表；`core/store/issue/helpers/issue-list-binding.ts` 已有 487 行，加目录之前先按职责拆开（D7，re-72 F9）；改写 `helpers/base-issues.store.test.ts:451-458` 的 `planningDirectoryIsUnavailableUntilP7`；`cycle_issues`、`module_issues` 由 `issues` 去掉本页的键推出（A8，re-70 F6）；偏好的筛选类型按页收窄（A9，P6b R16）；筛选（含“无”）、分组、排序、拖拽（A6；模块拖到“无”是只去掉来源的模块还是清空，由 P7b 定；拖拽的请求体用 §2.2 的增减）；表格两列表头的 `cycle_name`、`module_name` 排序（R21）与成员列表（个人页）上是否给名字排序；“添加已有”（空态的 `cta_secondary`、分组头、日历的 `addIssuesToView`、`existing-issues-list-modal.tsx` 的迭代与模块调用方，A4；排除“在本迭代”还是“在任一迭代”、搜索的页大小）与 M3 交接 §3 的弹窗核对（A5），逐个的 `PATCH` 用 §2.2 的命令；M4 交接 M6-02、M3 交接 §1 的回执。P7a 留下的接缝（台账 T3–T13）：选项与模块集合的纯函数在 `core/lib/planning-directory.ts`（`cycleOptions`、`moduleOptions`、`modulesAfter`、`moduleSetChange`），选项的读在 `core/hooks/use-planning-options.ts`，值的名字与两份未归档列表的读在 `core/hooks/use-planning-names.ts`（`usePlanningNames`、`usePlanningListReads`，键见 §2.3）；拖拽与“添加已有”的模块写经写入单元的 `changeIssue` 与 `IssueChange` 的 `modules: { add, remove }`（`@nerve/types` 的 `issues/issue-change.ts`）；`SPREADSHEET_PROPERTY_DETAILS` 的类型让 `modules`、`cycle` 两项的四个排序键可省（省就四个一起省），P7b 给出两向的 `module_name`、`cycle_name` 时不改类型，表头在条目有排序键时照旧给排序菜单与 `aria-sort`；显示面板与表格的开关门共用 `core/lib/planning-switch.ts` 的 `displayPropertyOn`。

**P7c**（命令面板、移动端页头与遗留）：命令面板（A16：迭代、模块的本地过滤与切换，取数门为假时菜单一直加载，英文的 `emptyText`，命令的可见性；是否给工作项改迭代、模块的命令，若给就用 §2.2 的命令与 §2.4 的选项规则）；迭代页、模块页的移动端页头与项目页的（A15、X2）；快捷操作“恢复”在途时禁用（A17）、归档页行菜单的参数类型（A18）、分组时页头的数目（A19）、其余没有读者的键（A20）、`fake-planning-page.tsx` 的 `opened`（A21）；设计 §8 的四个视图文件（X1）；G19 的工作项一侧；M2、M3、M1 其余的回执。

**e2e 的预算（P7b、P7c）**（re-72 F5）：照调研的余量（`research/web.md` §9.3：按上界算，P7 三段的新页面故事合计约 35–57 秒本机 chromium），本段两个新文件实际本机 chromium 合计 6.5 秒（G4 3.9 秒、G11 2.6 秒；firefox 5.6、3.9 秒，webkit 4.6、3.8 秒；T14 的三种引擎运行），少于约 20 秒的预算，P7b 与 P7c 两段合计约剩 28.5–50.5 秒；T9 检查点的分支 CI 的 e2e 作业 19 分 32 秒（基线 20 分 20 秒），T16 推送之前的估算、上界与推送后的实测由 T16 记下，以它们与 27 分 41 秒那样的慢 runner 看，余量可能更少。e2e 作业的结构（例如分片）是负责人的决定，P7b、P7c 多半要提出来，不自行改。

**M7**：工作区级的迭代、模块列表出现之后，跨项目的行（§2.3、R5）可改读它。

**M8**：[M6-M8-08](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-08) 在 M6 收尾核对根与写入单元的行数（本段 T16 记下）；[M6-M8-01](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-01) 多 P6b、P7a 的 e2e 数据点；[M6-M8-05](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-05) 照旧。T15 记下：[M6-M8-08](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-08) 的本段之后的行数（根 320、写入单元 302、请求体 98，T16 以 `wc -l` 再核）；[M6-M8-01](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-01) 多 T9 检查点的数据（e2e 19 分 32 秒）；[M6-M8-03](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-03) 再现一次（T9 检查点本机 e2e 的 W1 (page) 与 W6 (API)，只重跑这两个文件即通过）；新 [M6-M8-13](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-13)：上方的内容晚到时，打开着的下拉列表不随按钮移动（T14 发现：预览的描述编辑器约晚 0.25 秒挂上、属性一栏下移约 40 像素；不是本段引入，以 react-popper 放置列表的下拉框共有，与 M5-closeout 所记详情页晚挂载描述编辑器的版面位移同源），G4、G11 在打开列表之前先等上方的内容到齐。

## 6. 风险

| 风险 | 处理 |
|---|---|
| 整体替换盖掉别处的改动 | 网页一侧以增减在轮到时作用于所持的集合（R3），同一标签页内的连续改动与弹窗打开期间的读不丢；别的标签页或别人在读与写之间的改动仍可能被盖掉：接口没有前置条件（`If-Match` 之类），这是 nerve 的合同，不是网页能补的，登记差异、不改接口（D10）。不是须停下的契约缺口：设计 §3.3 已接受整体替换与“添加已有”逐个 `PATCH` |
| 已归档的名字要多读 | 按需、一个项目一次（R14）；跨项目的页按项目读（R5），工作区级的列表在 M7 |
| 写入路径的改动碰坏 M4–P6b | T2、T3 先留基线收据，测试只加范围或改类型，断言不改；变化命令的变异（调用时算、丢掉已归档的）；P6b 的移出改经它后同数 |
| 表单优先改变迭代页、模块页新建的行为 | P5b、P6b 的新建测试（页头、分组头、全局新建只在可加入时带本页）照旧成立、作回归：它们把字段的组件换成 `() => null`，初值在弹窗一级并入，才不依赖字段（re-72 F2）；新测试钉住“表单选别的、去掉本页的”；快速新建照旧注入 |
| 行与表格多了两个属性，影响既有的页面故事 | 只在开关打开的项目显示（R11）；迭代页、模块页的故事开着开关：T9 的检查点以默认配置跑全部故事，按契约补的只在回归边界允许处，台账写明 |
| 三个大文件的拆分 | 单独的 Task（T4）只搬不改，以基线收据核对 |
| e2e 时长 | 预算约 20 秒（D9）；T9 校准、T16 估算与上界；最慢的 runner（27 分 41 秒）上上界约 29.2–29.4 分钟，可能越过 30 分钟，越过就先报告负责人、不推送（§2.11） |
| Task 的量 | 16 个 Task（re-72 F3 把目录与迭代下拉框拆成两个）；弹窗分新建与修改两个 Task；若 T8（显示属性与行）超出一个实现者的量，把显示面板的开关门移到 T9，不加 Task |
