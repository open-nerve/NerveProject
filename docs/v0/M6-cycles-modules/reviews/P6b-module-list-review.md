# M6/P6b 阶段评审记录

> 实现与审查已完成，待分支 CI 与合入：整分支审查 Ready after fixes（[re-71](../../../reviews/re-71-M6P6b整体审核.md)：没有 Critical、Important，六处 Minor 与四条观察），修复轮 F（F1–F9）处理六处 Minor、一条观察与修复轮中另见的两处同类边缘，经限定复审 Ready to merge（一处 Minor 交 P7，re-71 §4）；最终候选 `dd98bcaf`（最后一个改代码的提交是 F9 的 `09aba5f8`，`dd98bcaf` 只改文档），其后只有本记录所在的提交（只改文档）。最终 HEAD 的分支 CI、合入、main CI 与归档清理见 §7（由 P7 的 T1 回填）。基线 main `06b72d4b65bcba911344a4ed3c358f164a5fdeeb`（P6a 的 noff，双父 `aadaf61a`、`8188582e`；main CI 43469787621 attempt 2 通过）。

## 1. 评审方式与范围

P6b 交付设计 §12 的 P6b 一行（P6 因超出一段的上限拆为 P6a/P6b，[P6a spec](../specs/P6a-web-modules.md) §1、R15、R16）：往工作项根 store 加模块之前，先按职责拆出写入单元，调用方改经它（[M6-M8-08](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-08)）；P5b 只为迭代页写的列表件中只差领域的，改为迭代与模块共用；模块页的工作项列表接上 M4 的列表引擎（`EIssuesStoreType.MODULE`、本页偏好、四种布局、页头、空态、已归档只读、本页新建与“移出模块”的 `module_ids` 整集合、修改与复制不覆写模块集合、写后重读进度、附件列、侧栏的行切换本页筛选）；项目页、迭代页、模块页的页头控件在本页偏好读到之后才出现，重读失败时正文沿用已读的偏好；G15、G19 的模块页列表部分。“添加已有”、模块下拉框与列表词表、模块页的移动端页头在 P7（[spec](../specs/P6b-module-list.md) §1、R10）。

- **调研**：两个只读调研代理在 main `06b72d4b` 上写出现状笔记（网页的现状、P5b 的接法、工作项根 store 的拆法、交接各项、e2e 与守卫；Plane 的模块页列表、nerve 的契约与两者的差异），随台账归档。
- **界定**：主控先裁定四件事（工作项根 store 取“调用方改经写入单元”的拆法、P5b 的迭代页列表件中只差领域的改为两域共用、第 8、9 条用于三页、按别的模块筛选只登记差异），再由起草代理写 [spec](../specs/P6b-module-list.md)（R1–R16、A1–A8）与 [plan](../plans/P6b-module-list.md)（16 个 Task，T10 检查点）（`ca4e62ac`）。
- **预检**：实施前一次 fresh 只读预检（[re-70](../../../reviews/re-70-M6P6b预检.md)），结论为修正后可实施：一处 Important（F1：e2e 时长的基线应是 P6a 的 main CI attempt 2 的 25 分 00 秒，T16 另算上界与 T10 的校准点，慢的 runner 上可能越过 30 分钟的报告线）与六处 Minor（F2 T1 以 P6a 的归档为准；F3 写入单元对根没有值的循环导入；F4 共用的布局根不导入 `PeekOnEntry`，按域的差别经 props 或按域的小表；F5 写入单元不加入 `project-address-route`；F6 `module_issues` 的副本由 P7 改为推出；F7 T3 的收据只查方法的定义），在实施前改入（`50e19cd8`）。
- **实施**：T1–T15 由实现代理完成，主控逐 Task 复核并记台账；拆分（T2、T3）与共用（T4、T7 与 T9–T12 的共用部分）先以既有测试留基线收据，结束时逐用例名、逐文件同数。
- **门禁与审查**：T10 一次检查点（七项门禁与分支 CI，CI 的实测作 T16 估算的校准点）；T16 一次 fresh 只读整分支审查（[re-71](../../../reviews/re-71-M6P6b整体审核.md)），修复轮 F 另经一次限定复审；最终候选的七项门禁、三种引擎的页面配置，与推送前的 CI 时长估算（spec §2.10）。

## 2. 验收状态

A1–A7 满足（[spec §4](../specs/P6b-module-list.md#4-验收标准)已勾选），A8 待合入后由 P7 的 T1 勾选。证据见 §4、§5：

| 验收 | 内容 | Task |
|---|---|---|
| A1 | 拆分：写入单元持有写队列、在途的写、确认的流水线、六个命令与请求体；根 320 行、写入单元 300 行、请求体的纯函数 75 行，之间没有值的循环导入；合入的状态里根上没有转发；M4、M5、P5b 的测试只改接收者（受影响 21 个文件 509 例、相关 14 个文件 164 例同数）；白名单与在途的写各有变异 | T2、T3、T5 |
| A2 | 共用：spec §2.2 的表逐个落实，按域的差别在两张以 store 类型为键的小表里，共用件没有按域的分支、不导入迭代独有的件；迭代的测试只改导入、构造与调用处的名字（基线同数） | T4、T7、T9–T12、F1、F4 |
| A3 | 引擎与 store：MODULE 的 store、动作、路由映射、挂载键、本页筛选、空态；模块页偏好（按模块 id、不读 router、随一代退役、模块关闭时不读）；`module_ids` 的白名单、本页新建只在可加入时取并、移出轮到时算并保留已归档与别的模块、修改不带 `module_ids` | T5、T6、T8、T10、F2 |
| A4 | 页面：四种布局、页头、空态、已归档只读、模块关闭的停用、新建一个请求、移出模块、拖拽、修改与复制、被拒时说明、写后重读进度、侧栏的行切换；第 8、9 条在三页 | T8–T12、F2、F9 |
| A5 | G15、G19 的模块页列表部分（三种引擎） | T13、T14 |
| A6 | 守卫：`project-address-route`（各 Task 的新件，F9 的 `planning-page-entity`）、swr 限制（`priority-issues.tsx`、共用布局根，F7 再加五个既有读者）、告警上限 44、knip、关键词、i18n、按本页分组、`root.store.test.ts` 21 个 store | T1、T7、各 Task、F7、F9 |
| A7 | 文档：P6a 的回填、交接回执、差异清单（含更正）、前端改动清单、设计 §8/§12/§14 | T1、T15、F8、T16 |

## 3. 局部裁定

台账中 124 条 `Ruling:`（T1–T15 111 条、修复轮 F 13 条）与主控的裁定（写 spec 前的四件；T2 之后给 `skip_activity` 的检查补测试；T9 留意的列表状态放进修复轮；修复轮中 swr 的观察现在收紧、两处偏离接受、同类的两处边缘现在修为 F9），均为局部取舍，没有架构待决项，也没有需要负责人决定的问题。要点：

- **工作项根 store 的拆分**（R1 的选项 B，主控裁定；T2、T3）：写的一侧整体移到写入单元 `IssueWrites`（`core/store/issue/issue-writes.ts`，300 行：每个工作项一条的写队列、拖拽的队列、在途的写与 `pendingIDs`、确认的流水线、六个命令），请求体与成员快照是旁边的纯函数（`issue-write-fields.ts` 的 `createBody`、`updateBody`、`membershipFields`，70 行，T5 放行 `module_ids` 后 75 行）。根以 `readonly writes = new IssueWrites(this)` 建立它；写入单元对根只有 `import type` 的窄接口 `WritesRoot`（`Pick<IssueRootStore, …>` 的 10 个键），沿值导入走不到 `root.store.ts`（re-70 F3）。根 615 → 298 行（T3），T4 以 `PagePreferencesFilter` 直接构造迭代页偏好、T5 加入模块页的两个 store 之后 320 行。T2 改产品代码 12 个文件 18 处，根上暂留的 7 个转发只给测试（删去它们，tsc 的 86 个错误全在测试文件）；T3 按 tsc 的行列在 21 个测试文件 83 处只插入 `writes.`（折回后与 HEAD 逐文件全等），删去转发。`updateIssue` 没有范围时退回 router 工作区的一支照搬（`issue-writes.ts:261`），写入单元因此不加入 `project-address-route`（re-70 F5），P7 去掉这一支后加入。
- **`skip_activity` 的检查补测试**（T2 之后的主控裁定，T3）：T2 补做的变异“去掉 `skip_activity` 只可与 `description_html` 单独同发的检查”在整个 web 套件（411 个文件 4793 例）存活：M4 起这道检查没有能失败的测试。请求体成了纯函数之后，T3 新增 `issue-write-fields.test.ts`（6 例）钉住白名单的 `null` 与 `undefined` 和这条规则（以 JSON 往返看线上的请求体），不改既有断言；那个变异与检查的三个子句各一个变异都让它失败。
- **两域共用与两张按域的小表**（R2，主控裁定；T4、T6–T12）：共用件不写 `if (domain === …)`（re-70 F4），按域的差别放进两张以 store 类型为键的表：hooks 层的 `core/hooks/planning-page-domains.ts`（`PLANNING_PAGE_DOMAINS`：路由参数、页面的名字（跟进的范围名，也是预览宿主的前缀）、进度所在的 store、可加入、移出的命令）与组件层的 `issue-layouts/planning-page-layouts.ts`（`PLANNING_PAGE_LAYOUTS`：取数门的域、偏好的 SWR 键、筛选项、列表状态的 hook、移出的文案与成功提示、空态的文案）。分成两张，是因为组件层的表起初持有行的快捷操作（组件，T10 改为 `remove` 一列）与只读来源的 hook，放进 hooks 层会让 hooks 依赖组件（T7）。共用件：本页列表 `PlanningPageIssues`（本页的形状 `PlanningPage {projectID, entityID}`）、按域的 `usePlanningPage{IssueActions,PreferencesFollow,ProgressFollow}`、布局根 `PlanningPageLayoutRoot`（迭代页把 `PeekOnEntry` 作为 children 交给它，保住“宿主在位之后才请求预览”的时序；自布局根的导入图 634 个文件到不了 `peek-on-entry`）、四种布局的 `PlanningPage*Layout`、空态 `PlanningPageEmptyState`、行的快捷操作 `PlanningPageIssueQuickActions`、页头的 `usePlanningPageHeader`、侧栏的 `usePageRowChoices`、重读失败的 `PreferencesRereadFailure`。迭代的布局根、四个根、空态与快捷操作以 `git mv` 改为共用件（T7、T10），被取代的旧路径留在 `project-address-route` 里阻止复活；各 Task 以脚本核对共用件没有按域的分支。分开的：列表状态（`useCycleListState`、`useModuleListState`）、偏好的 service、显示选项常量 `module_issues`、进行中卡片的 `CyclePage`（F4 加一行注释写明理由）。迭代的测试只改导入、构造与调用处的名字（如本页形状的 `cycleID` → `entityID` 7 处），以折回名字后与 HEAD 全等的脚本与逐用例名的基线核对。
- **三页的偏好读取**（R3、R4；T9、T12）：页头的布局、筛选、显示在本页偏好读到之后出现：迭代页与模块页经 `usePlanningPageHeader` 的 `preferencesShown`，项目页的 `HeaderFilters` 在 `projectIssuesFilter.getIssueFilters(路由的项目)` 有值之后。“第一次读”以 store 是否持有本页偏好判断，而不是 SWR 是否答过：同一会话回到页面时，挂载时的读失败按重读处理。持有时正文照常，最上方一行共用的 `PreferencesRereadFailure`（`role="alert"`，原因经 `useRefusalMessage`，重试与第一次读失败的重试同一个）；未持有时照旧（迭代页、模块页是可重试的 `PlanningReadFailure`，项目页是 M4 的文字、没有重试，交 [M6-M8-12](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-12)）。
- **模块集合的写入**（R5、R6、R7；T5、T6、T10）：两份白名单放行 `module_ids`。“移出模块”是 `ModuleIssues.removeIssueFromModule`，经写入单元的 `updateIssue(id, (held) => …)` 在队列轮到时以所持的整集合去掉本模块（已归档的与别的模块都留着），经按域的动作 `removeIssueFromView` 跟进进度，工作项不在本页的项目时不发。行的移出命令由四个 base root 经 `IQuickActionProps.handleRemoveFromView` 传下（照 Plane），进度的跟进挂在本页期间一直在的 base root 上，而不是确认后随即卸下的行（T10）。本页新建只在目标项目是模块的项目、且模块按页面所见可加入时，把本模块并进给定的集合（给定的在前、去重）；全局的“新建工作项”同样只在可加入时带上。修改的请求体不带 `module_ids`（表单的字段没有它）；“复制一份”只加入本模块：命令宿主的 `copyOf` 带上源的模块与新建的请求体带上表单的模块两处一起改的嵌套变异等价存活（表单的字段是第三道），以一次期望错误的运行证实断言，P7 加模块字段时另钉（spec §5）。
- **“本页持有”**（re-71 M2 与修复轮中另见的两处，F2、F9）：`core/hooks/planning-page-entity.ts` 的 `pageEntity` 是唯一的说法：实体存在，且页面有项目时它的 `project_id` 是路由的项目；属路由项目的已归档模块、已结束或已归档的迭代照样持有（只读，而不是“没有”）。两个列表状态的纯函数经它算只读与可加入（实体属别的项目时只读、不可加入）；两个 hook 另给 `held`，页头的数目与偏好的控件要 `listed && held`；迭代页头照模块页头（P6a 起如此）只在迭代属路由的项目时给面包屑标题与快捷操作。迭代的行为只在这一边缘处改变，迭代的测试只增。
- **侧栏的行**（R14；T11）：两域共用 `usePageRowChoices(storeType, id)`，行可点只看本页的筛选实例在不在：实例只由布局根在读到偏好之后建立、卸下时删除，所以“预览里不可点”“偏好读到前不可点”“模块关闭时不可点”由这一条同时保证；不照 Plane 的日期门（nerve 的页签总显示，P6a R8）。
- **e2e 的世界与帮手**（T13、T14）：列表页的夹具按域共用，迭代的故事只改调用处；换会话用同一浏览器的另一标签页重新登录（`anotherTabSignsIn`），“载入后仍在”由另开的浏览器核对，不用 `page.reload()`；详见 §5。
- **修复轮 F 的取舍**：F1 的帮手作用于 `planningPageTab` 最后打开的页（迭代测试的调用处一字不改的唯一办法），弹窗与编辑器的替身另立 `issue-modal/fake-dialog.tsx`（放进 `fake-planning-page.tsx` 会在 `@nerve/ui` 的模拟工厂里成环）；`preferences-reread.test.tsx` 与页头测试共有的约 25 行是 `vi.mock` 的调用，各留在本文件。F3 以 `git mv` 改名，P5b 的旧记录不改。F4 保留 `CyclePage` 并加注释。F7 只加五个既有读者，第六个已由 `core/components/workspace/settings/**` 覆盖（以一次变异证实）；它与 F1 在迭代测试里多改的导入行，主控都接受。F9 的 `held` 由 hook 给、不加进纯函数的返回（纯函数的单测以 `toEqual` 比整个对象）。
- 其余见台账。

## 4. 实施记录

变异数为能编译的产品变异（e2e 的为能编译、能构建的），守卫（关键词、swr）的变异另记在台账。

| Task | 提交 | 内容 |
|---|---|---|
| spec/plan 与预检 | `ca4e62ac`、`50e19cd8` | 16 个 Task；预检 re-70 的修正（F1 Important，F2–F7） |
| T1 回填与阶段 | `97f1962a` | P6a 阶段记录 §7 与抬头、P6a spec A8、设计 §14、前端改动清单、M6-M8-01 补 C7 的再现；swr 限制加 `priority-issues.tsx`（变异证实） |
| T2 拆分一 | `c331aa9d` | 写入单元 `IssueWrites` 与纯函数 `issue-write-fields.ts`，产品代码 12 个文件 18 处改经它，根 615 → 325 行（含 27 行只给测试的转发）；测试未改，受影响 21 个文件 509 例与相关 14 个文件 164 例同数；5 个变异失败，计划外 1 个存活（`skip_activity` 的检查，交 T3） |
| T3 拆分二 | `69130263` | 21 个测试文件 83 处只改接收者，删去转发（根 298 行）；新 `issue-write-fields.test.ts` 6 例；7 个变异失败（含 T2 存活的那个） |
| T4 共用一 | `85d3a0cc` | `PlanningPageIssues`、三个按域的 hook 与 hooks 层的表；`CycleIssuesFilter` 与三个迭代的 hook 删去；迭代 14 个边界文件 161 例与相关 12 个文件 98 例同名同数；6 个变异失败 |
| T5 模块页的偏好与列表 store | `c09f8113` | `ModulePreferencesService`、`ModuleIssues`（新建并入本模块、移出在轮到时算）、两份白名单放行 `module_ids`；新 23 例与既有文件加 5 例；11 个变异失败 |
| T6 列表引擎的 MODULE | `64ac0312` | `useIssues`、`useIssuesActions`、路由映射、挂载键、`usePageIssueFilters`、`module_issues`、表的模块行与 `removeIssueFromView`；新 33 例、既有文件加 2 例，契约删 2 例、改 1 例；12 个变异失败 |
| T7 共用二 | `6a174b7c` | 布局根、四种布局与空态以 `git mv` 改为共用件，组件层的表，迭代页以 children 交出 `PeekOnEntry`，swr 限制加共用布局根；迭代 160 例与相关 106 例同名同数；9 个变异失败 |
| T8 模块页的列表 | `d4eb1b08` | 模块页正文经共用布局根，空态、已归档只读、模块关闭的停用，MODULE 进命令宿主、四个 base root、拖拽与筛选表达式；新 32 例；15 个变异失败；P6a 的模块页面故事（6 个文件 11 个故事）chromium 照过 |
| T9 页头与本页新建 | `18a5ff8f` | `usePlanningPageHeader`（迭代页头改用）、模块页头的数目、控件与新建，项目页头的控件在偏好读到之后，三处锁项目；新 29 例；13 个变异失败 |
| T10 移出、修改与复制 | `3a5c3cf6` | 共用的 `PlanningPageIssueQuickActions` 取代两域的快捷操作，`handleRemoveFromView`，命令宿主的 `useRemovalFromViewRequest`；新 24 例；12 个变异失败，复制的嵌套变异等价存活、以期望错误的 RED 证实；检查点 |
| T11 侧栏的行 | `c3a300cf` | `usePageRowChoices` 两域共用，模块侧栏的行切换本页筛选；新 10 例；8 个变异失败 |
| T12 偏好的重读失败 | `b2ffe170` | `PreferencesRereadFailure`，共用布局根与项目页的根；新 9 例，边界 44 个文件 547 例同名同数、既有测试一行未改；10 个变异失败 |
| T13 页面故事一 | `8200f347`、`c65f7d1b` | 列表页夹具按域（`planning-list-pages.ts`），迭代故事只改调用处；G15 模块页列表（3 个故事，400 行）；e2e 变异 9 个失败、1 个等价存活已证实 |
| T14 页面故事二 | `61e6075e` | G19 模块页列表（4 个故事，319 行）；期望错误的 RED 1 个、e2e 变异 13 个失败（两组嵌套），单去一道的存活与失败已说明 |
| T15 文档 | `513d2ff3` | 差异清单更正三行（R8 两行与 P5b 一行的同一句）、新增 12 行；前端改动清单；M4、M5、M3 交接的 P6b 回执；M6-M8-08 的处理结果、新 M6-M8-12、M6-M8-07 与 M6-M8-01 补记；设计 §8/§12/§14；spec §5 的修复轮清单 |
| F1 | `5b1046b7` | 规划页测试的帮手移进 `fake-planning-page.tsx`，弹窗与编辑器的替身移进 `issue-modal/fake-dialog.tsx`（re-71 M1）；18 个文件 201 例同名同数，迭代测试只改导入行；2 个变异失败 |
| F2（含 F5） | `11b9ea74` | 两域的列表状态核对路由的项目（re-71 M2）；新 8 例（RED 6 例）、迭代测试只增；2 个变异各让 3 例失败；`CycleListState` 去掉导出（re-71 M5） |
| F3 | `9b67b78c` | `cycle/filter.store.test.ts` 以 `git mv` 改名为 `cycle/page-preferences.test.ts`，内容不改（re-71 M3） |
| F4 | `238a29d2` | `priority-issues.store.ts` 的 `CyclePage` 加一行注释（re-71 M4） |
| F6 | `36e23061` | 模块关闭时移出被拒的单测以 409 作答（re-71 M6） |
| F7 | `d302141f` | swr 限制加五个既有的 `useSessionSWR` 读者（re-71 的观察）；变异证实 |
| F8 | `45c5e665` | re-71 的记录、spec §5 的修复轮清单、spec 与 plan 的状态、设计 §14 |
| F9 | `09aba5f8` | `pageEntity` 与 `held`：页头的数目与偏好的控件只给本页持有的实体，迭代页头核对项目（修复轮中另见）；两个页头测试各加一例，测试只增；5 个变异失败 |
| F9 的文档 | `dd98bcaf` | re-71 §3 与 spec §5 记下 F9 与主控对两处偏离的裁定，设计 §14 |
| T16 记录 | 本记录所在提交 | re-71 §4（修复轮的限定复审）、本记录、spec §4 的勾选与 §5 的 P7、spec 与 plan 的状态、设计 §14、前端改动清单 M6 一行 |

## 5. 门禁与 CI

- **检查点**（T10 之后，`3a5c3cf6`）：
  - `t10-gate`：`make gen-check`、`make lint-go`、`make test`（本机 925 秒通过）、`make lint-web`、`make knip`、`make test-web` 通过；`make build && make e2e` 461 过、1 败：F11 (page) [local]（`e2e/fixtures/file-story.ts:37`）的一次内容读取 `net::ERR_ABORTED`（`e2e/fixtures/content-files-page.ts:225`），即 [M6-M8-07](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-07) 的签名，与本段无关；只重跑这一个文件一次 7 条通过（`t10-gate-rerun-f11`），T15 记入该条。
  - 分支 CI 43517662603 attempt 1 全绿：server (0) 132203647956、web 132203648276、server (1) 132203648324、e2e 132209901127（e2e 作业 22:28:06Z–22:53:44Z，25 分 38 秒）。这时还没有本段的页面故事，跑的与 main 相同，比基线 25 分 00 秒多 38 秒，作 T16 估算的校准点（T15 记入 [M6-M8-01](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-01)）。
- **三种引擎的页面配置**（`playwright.m6-pages.config.ts`）：
  - T13：新文件与迭代的 G15、G19 列表页故事 30 例，新文件 `--repeat-each=3` 27 例；`g15-module-list-page.spec.ts` 的时长 chromium 18.5 秒、firefox 32.9 秒、webkit 27.0 秒（9 个 worker）。
  - T14：新文件、模块的 G15 与迭代的 G15、G19 42 例，新文件 `--repeat-each=3` 36 例，整套 18 个文件 111 例；`g19-module-list-page.spec.ts` 的时长 chromium 13.0 秒、firefox 22.0 秒、webkit 15.7 秒。
  - 修复轮：F8 之后（`45c5e665`）`f-gate-m6-pages` 整套 111 过、两个模块列表故事 `--repeat-each=3` 63 过，同一棵树上 `f-gate-lint-web`、`f-gate-knip`、`f-gate-test-web`、`f-gate-build` 通过；F9 之后（`09aba5f8`）`f9-gate-m6-pages` 整套 111 过（2.2 分钟），迭代与模块的四个列表故事 `--repeat-each=3` 126 过（`f9-gate-repeat3-list-pages`）。
  - 都通过，没有遇到 M6-M8-03、M6-M8-07。
- **最终候选**：`f-final-gate`（`dd98bcafbd24fd747602b29dc5fed2e76aa1a624`）七项全过：`make gen-check`、`make lint-go`、`make test`（本机 942 秒通过）、`make lint-web`、`make knip`、`make test-web`、`make build && make e2e`（469 过，3.6 分钟，没有间歇失败）。三种引擎的页面配置以 F9 之后的 111 例为准：`dd98bcaf` 相对 `09aba5f8` 只改了文档。
  - 修复轮的限定复审与这次门禁并行，结论见 §6。
  - 本记录所在提交只改文档，另跑文档链接、关键词与 `make lint-web`（收据 `t16-record-*`），其余以准确 HEAD 的分支 CI 为准。
- **CI 的 e2e 时长估算**（推送前，spec §2.10，re-70 F1）：本段新的页面故事在 `f-final-gate` 本机 chromium 合计 28.0 秒（g15 17.0、g19 11.0）。
  - **估算**：×1.67–2.25 为 47–63 秒串行，2 个 worker 摊开约加 0.4–0.5 分钟；以检查点 43517662603 的 25 分 38 秒为准约 26.1 分钟（以 main 基线 25 分 00 秒为准约 25.5 分钟）。
  - **上界**：×3.6–4.1 为 101–115 秒，约加 1.7–1.9 分钟，约 27.3–27.6 分钟。
  - **校准点**：25 分 38 秒。
  - 三者都低于 30 分钟的报告线，所以推送；实测超过 30 分钟则停下报告负责人。实际的 e2e 作业时长见 §7。
- **测试与 e2e 的教训**（交 P7）：
  - **P5b T14 的更正**（T14）：P5b 的 T14 台账说失权迟到答的两道守卫“单去任一道都存活”。这只对被取代的部分读（`core/lib/reconciled.ts:93`）成立：旧读发出在失权之前，权限水位先拒绝它，单去部分读是等价变异。单去权限水位（`IssueRootStore.isReadCurrent`）本身就让模块页与迭代页的 G19 故事二失败（`t14-e2e-mut-late-guard-inner-alone`、`-2`、`-cycle`）：水位不只挡绑定的合并，还挡列表回答里写进工作项 store 的每一项（`core/store/issue/issue.store.ts:40`）；两道一起去掉的嵌套变异同样失败。P5b 的阶段记录不改，更正写在 [M4 交接](../handoffs/M4-closeout.md)的“处理结果（M6/P6b）”一节。
  - **共用的测试帮手**（F1）：规划页测试的帮手在 `issue-layouts/roots/fake-planning-page.tsx`（`outlivesTheRow`、`dialogTitle`、`choose`，新建表单的 `creates`、`form`、`projectField`、`send`、`quickAddForm`、`headerButton`、`made`、`globalCreate`，页头的 `inHeader`），作用于 `planningPageTab` 最后打开的页（模块级的 `opened` 与 `current()`）；弹窗与编辑器的替身在 `issue-modal/fake-dialog.tsx`，各文件的 `vi.mock` 仍留在本文件，工厂以 `await import` 取它（`vi.mock` 的工厂读不到文件顶部的导入）。P7 的规划页测试从这两处导入，不再复制；`opened` 在标签页关闭时不清空（re-71 §4 的 Minor），P7 先清掉它（spec §5 的 P7）。
  - **变异脚本先跑 typegen**（T13）：`make build` 会删掉 `.react-router/types`，之后的 `tsc --noEmit` 找不到路由的 `+types`（TS2307），那次变异不算（`t13-e2e-mut-create-no-module` 的 `tsc_exit=2`；T9、T11、F9 的类型检查同因）。`t13-e2e.sh`（T14 照写 `t14-e2e.sh`）依次跑 `react-router typegen`、web 与 e2e 的 `tsc --noEmit`、`make build build-s3-fixture` 与 chromium 的故事，只有能编译、能构建的变异才计。
  - **列表页的夹具按域**（T13）：`e2e/fixtures/cycle-list-pages.ts` 以 `git mv` 改为 `planning-list-pages.ts`：私有的 `listPageWorld`（人、项目、状态、造物）之上，每域一个导出的世界 `cycleListWorld`、`moduleListWorld`，而不是带 domain 参数的一个世界（两域的造物与路径不同，knip 也不许没有读者的导出先于故事加入）；`openCyclePage` 改为 `openListPage`，`layoutButton` 改为 `workItemLayoutButton`（与 `module-pages.ts` 的模块列表的同名帮手冲突，改列表页这一个，不留别名）。迭代的故事只改导入与调用处的名字（`t13-call-sites-only.py` 核对正文逐字相同，7 个故事 chromium 同基线）。P7 的列表页故事从这里取世界与帮手。
  - **页面上等价的变异**（T13）：T5 单测里“移出时去掉 store 里已归档的模块”的变异在页面上等价存活（`t13-e2e-mut-removal-drops-held-archived`）：模块页只读到项目未归档的模块列表，已归档的模块不在 store 里；改为“只保留页面读到的未归档模块”（只凭页面所知算集合的错法）才在数据库的归属处失败。写 e2e 变异之前先弄清页面手里有什么。
  - **不重新载入**（T14）：重读由“应用内到别处再后退”触发，不用 `page.reload()`（[M6-M8-09](../../M8-open-release/handoffs/M6-closeout.md#M6-M8-09)）；装了扣住请求的路由之后只在同一路由模块内跳转（WebKit 的 modulepreload），要载入新路由模块的一步不装路由。访客的“移出模块”有两道门（行只读与移出的角色门），单去一道等价存活，以嵌套变异证明故事守着它们，角色门另由 T10 的单测钉住。
- **最终 HEAD 的分支 CI 与合入后的 main CI**：见 §7。

## 6. 整分支审查

fresh 只读审查（[re-71](../../../reviews/re-71-M6P6b整体审核.md)，`fable`，范围 `06b72d4b..513d2ff3`）逐项核对了：

- 工作项根 store 的拆分（照搬不改，写入单元对根没有值的循环导入，合并后没有转发）；
- 两域共用的件（没有按域的分支，迭代测试的断言不改）；
- 列表引擎的 MODULE（store、动作、路由映射、挂载键、本页筛选与空态的接线）；
- 模块集合的写入（本页新建、移出与修改的请求体对照 nerve 的 `membership.go`：整集合、轮到时算、保留已归档与别的模块）；
- 三页的偏好读取（R3、R4）与按页收窄偏好（R16）；
- 写后的后续动作只在确认之后、会话分代下迟到的回答不上身、旧件删除完整、守卫、测试与 e2e、文档与其中 Plane 的引用。

结论 Ready after fixes：没有 Critical、Important，六处 Minor（M1–M4 即 spec §5 预先列出的修复轮清单，M5、M6 是新的）与四条观察，没有须负责人决定的事。修复轮 F 处理六处 Minor（F1–F4、F5 并入 F2、F6）与 swr 的一条观察（F7，主控裁定现在收紧），另三条观察不需动作，F8 写 re-71 的记录。修复轮中又见与 M2 同类的两处边缘（页头的偏好控件与数目不看实体的项目；迭代页头的面包屑标题与快捷操作不核对 `project_id`），都只在手改 URL 时出现、nerve 按实体自己的项目授权，主控裁定现在修，为 F9。

修复轮的限定复审（`fable`，范围 `513d2ff3..dd98bcaf`，与最终候选的七项门禁并行）结论 Ready to merge：每一处修复只做它所说的；既有的迭代测试只改导入，用例名与数不变（F2、F9 对迭代只加用例）；F2、F9 的“持有”只在实体存在且 `project_id` 等于路由的项目时成立，属本项目的已归档、已结束的照样持有，没有按域的分支、不读 router，持有的实体原来显示的东西一样不少；F1 的帮手是对的，M1 的重复已消除；F7 只收紧；文档把每一条发现对到它的修复。一处 Minor（`fake-planning-page.tsx` 的 `opened` 不在关闭时清空）交 P7，三条观察不需动作，见 re-71 §4。

## 7. 集成、归档与接续

（待回填：由 P7 的 T1 按归档里的台账、CI 原件与 `cleanup.log` 回填，作业号与时刻以 GitHub 公开的 jobs 接口核对。）

- **最终 HEAD 的分支 CI**：待回填（含 e2e 作业的实际时长，与 §5 的估算对照）。
- **合入与 main CI**：待回填。
- **归档**：待回填。
- **清理**：待回填。

**接续**：P7（工作项一侧：属性与下拉框、列表词表、“添加已有”、迭代页与模块页的移动端页头，设计 §12 的 P7 一行；交给它的事见 [spec](../specs/P6b-module-list.md) §5 的 P7）。
